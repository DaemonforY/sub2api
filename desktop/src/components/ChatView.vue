<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { openPath } from '@tauri-apps/plugin-opener'
import { createChat, type Approval } from '../lib/chat'
import { TEMPLATES, type TaskTemplate } from '../lib/templates'

const props = defineProps<{ workspace: string }>()
const emit = defineEmits<{ 'change-folder': []; relogin: [] }>()

const { state, connect, send, interrupt, answer } = createChat()
const draft = ref('')
const connecting = ref(false)
const connectError = ref('')
const scroller = ref<HTMLElement | null>(null)

const picked = ref<TaskTemplate | null>(null)
const composer = ref<HTMLTextAreaElement | null>(null)

/** Own files: put the prompt in the composer so it can be read and tweaked before sending. */
async function useOwnFiles(t: TaskTemplate) {
  picked.value = null
  draft.value = t.prompt
  await nextTick()
  composer.value?.focus()
}

/** Sample: makes fake files in 「示例」 and runs the task on them straight away. */
function trySample(t: TaskTemplate) {
  picked.value = null
  send(t.samplePrompt)
}

async function start() {
  connecting.value = true
  connectError.value = ''
  try {
    // A revoked key would otherwise surface only after Codex's retries; check it up front.
    const key = await invoke<{ status: string; message: string }>('check_key')
    if (key.status === 'invalid') {
      state.authFailed = true
      return
    }
    await connect(props.workspace)
    const devPrompt = await invoke<string | null>('dev_autorun')
    if (devPrompt && !state.entries.length) send(devPrompt)
  } catch (e) {
    connectError.value = String((e as Error).message ?? e)
  } finally {
    connecting.value = false
  }
}

onMounted(start)

async function relogin() {
  await invoke('sign_out').catch(() => {})
  emit('relogin')
}

function submit() {
  const text = draft.value
  draft.value = ''
  send(text)
}

const statusText: Record<string, string> = {
  inProgress: '进行中',
  completed: '完成',
  failed: '失败',
  declined: '已拒绝'
}
const fileKind: Record<string, string> = { add: '新建', delete: '删除', update: '修改' }

function decide(a: Approval, d: 'accept' | 'acceptForSession' | 'decline') {
  answer(a, d)
}

watch(
  () => state.entries.map((e) => (e.kind === 'agent' || e.kind === 'command' ? e.kind + e.id + (e.kind === 'agent' ? e.text.length : e.output.length) : e.id)).join(),
  async () => {
    await nextTick()
    scroller.value?.scrollTo({ top: scroller.value.scrollHeight })
  }
)
</script>

<template>
  <div class="chat">
    <header>
      <div class="folder" :title="workspace">📁 {{ workspace }}</div>
      <button @click="openPath(workspace)">打开文件夹</button>
      <button @click="emit('change-folder')">换文件夹</button>
    </header>

    <main ref="scroller">
      <p v-if="connecting" class="muted">正在启动…</p>
      <div v-if="state.authFailed" class="relogin">
        <p>登录已失效：这台电脑的 Key 被停用或删除了（也可能是在网页上退出了这台电脑）。</p>
        <button class="primary" @click="relogin">重新扫码登录</button>
      </div>
      <div v-else-if="connectError || state.fatal" class="notice error">
        {{ connectError || state.fatal }}
        <button @click="start">重新连接</button>
      </div>

      <div v-if="state.ready && !state.entries.length" class="empty">
        <p class="empty-title">想让它帮你做什么？选一个常用任务，或者直接在下面打字。</p>
        <div class="cards">
          <button v-for="t in TEMPLATES" :key="t.id" class="card" @click="picked = t">
            <span class="icon">{{ t.icon }}</span>
            <span class="card-title">{{ t.title }}</span>
            <span class="card-desc">{{ t.desc }}</span>
          </button>
        </div>
      </div>

      <template v-for="e in state.entries" :key="e.kind + e.id">
        <div v-if="e.kind === 'user'" class="bubble user">{{ e.text }}</div>
        <div v-else-if="e.kind === 'agent'" class="bubble agent">{{ e.text }}<span v-if="!e.done" class="cursor">▍</span></div>
        <details v-else-if="e.kind === 'command'" class="step">
          <summary>
            <span class="tag">运行</span><code>{{ e.command }}</code>
            <span class="status" :class="e.status">{{ statusText[e.status] ?? e.status }}</span>
          </summary>
          <pre v-if="e.output">{{ e.output }}</pre>
        </details>
        <div v-else-if="e.kind === 'files'" class="step">
          <div><span class="tag">文件</span><span class="status" :class="e.status">{{ statusText[e.status] ?? e.status }}</span></div>
          <ul>
            <li v-for="c in e.changes" :key="c.path">
              {{ fileKind[c.kind] ?? c.kind }}：<a href="#" @click.prevent="openPath(c.path)">{{ c.path }}</a>
            </li>
          </ul>
        </div>
        <div v-else-if="e.kind === 'notice'" class="notice" :class="e.tone">{{ e.text }}</div>
      </template>
    </main>

    <footer>
      <textarea
        v-model="draft"
        rows="3"
        placeholder="用中文描述你要做的事，回车发送，Shift+回车换行"
        :disabled="!state.ready"
        ref="composer"
        @keydown.enter.exact.prevent="submit"
      />
      <button v-if="state.busy" @click="interrupt">停止</button>
      <button v-else class="primary" :disabled="!state.ready || !draft.trim()" @click="submit">发送</button>
    </footer>

    <div v-if="picked" class="overlay" @click.self="picked = null">
      <div class="dialog">
        <h3>{{ picked.icon }} {{ picked.title }}</h3>
        <p>{{ picked.desc }}。</p>
        <p class="reason">用自己的文件：{{ picked.prepare }}结果会放在文件夹里的「输出」中，原文件不会被改动。</p>
        <div class="actions">
          <button @click="picked = null">取消</button>
          <button @click="trySample(picked)">先用示例试试</button>
          <button class="primary" @click="useOwnFiles(picked)">用这个文件夹里的文件</button>
        </div>
      </div>
    </div>

    <div v-if="state.approvals.length" class="overlay">
      <div class="dialog">
        <h3>{{ state.approvals[0].title }}</h3>
        <p v-if="state.approvals[0].reason" class="reason">原因：{{ state.approvals[0].reason }}</p>
        <pre v-if="state.approvals[0].detail" class="detail">{{ state.approvals[0].detail }}</pre>
        <div class="actions">
          <button @click="decide(state.approvals[0], 'decline')">不允许</button>
          <button v-if="state.approvals[0].allowSession" @click="decide(state.approvals[0], 'acceptForSession')">这次对话都允许</button>
          <button class="primary" @click="decide(state.approvals[0], 'accept')">允许</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat {
  height: 100%;
  display: flex;
  flex-direction: column;
}

header {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--line);
  background: var(--panel);
}

.folder {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--muted);
  font-size: 13px;
}

main {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.relogin {
  margin: auto;
  max-width: 420px;
  text-align: center;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 12px;
  padding: 20px 24px;
}

.muted {
  color: var(--muted);
}

.empty {
  margin: auto;
  max-width: 760px;
  width: 100%;
}

.empty-title {
  color: var(--muted);
  text-align: center;
}

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  text-align: left;
  padding: 14px;
  border-radius: 12px;
}

.card:hover {
  border-color: var(--brand);
}

.icon {
  font-size: 22px;
}

.card-title {
  font-weight: 600;
}

.card-desc {
  font-size: 12px;
  color: var(--muted);
  line-height: 1.5;
}

.bubble {
  max-width: 80%;
  padding: 10px 14px;
  border-radius: 12px;
  white-space: pre-wrap;
  line-height: 1.6;
}

.bubble.user {
  align-self: flex-end;
  background: #fef3c7;
}

.bubble.agent {
  align-self: flex-start;
  background: var(--panel);
  border: 1px solid var(--line);
}

.cursor {
  animation: blink 1s steps(1) infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

.step {
  align-self: flex-start;
  max-width: 90%;
  background: var(--panel);
  border: 1px dashed var(--line);
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 13px;
}

.step summary {
  cursor: pointer;
  display: flex;
  gap: 8px;
  align-items: center;
}

.step pre {
  margin-top: 8px;
  max-height: 240px;
  overflow: auto;
}

.step ul {
  margin: 6px 0 0;
  padding-left: 20px;
}

.tag {
  background: var(--bg);
  border-radius: 4px;
  padding: 1px 6px;
  color: var(--muted);
}

.status {
  margin-left: auto;
  color: var(--muted);
}

.status.failed {
  color: var(--danger);
}

.status.completed {
  color: #15803d;
}

.notice {
  align-self: center;
  font-size: 13px;
  color: var(--muted);
}

.notice.error {
  color: var(--danger);
}

footer {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  padding: 12px 16px;
  border-top: 1px solid var(--line);
  background: var(--panel);
}

footer textarea {
  flex: 1;
  resize: none;
}

.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: grid;
  place-items: center;
}

.dialog {
  background: var(--panel);
  border-radius: 12px;
  padding: 20px 24px;
  width: min(560px, 90vw);
}

.dialog h3 {
  margin: 0 0 8px;
}

.reason {
  color: var(--muted);
}

.detail {
  background: var(--bg);
  padding: 10px;
  border-radius: 8px;
  max-height: 200px;
  overflow: auto;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
