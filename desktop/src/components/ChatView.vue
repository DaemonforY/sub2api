<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { openPath } from '@tauri-apps/plugin-opener'
import { createChat, type Approval } from '../lib/chat'

const props = defineProps<{ workspace: string }>()
defineEmits<{ 'change-folder': [] }>()

const { state, connect, send, interrupt, answer } = createChat()
const draft = ref('')
const connecting = ref(false)
const connectError = ref('')
const scroller = ref<HTMLElement | null>(null)

// Starter tasks; the real template library (with sample folders) comes later.
const examples = [
  '看看这个文件夹里有哪些文件，按类型分别统计数量',
  '把这个文件夹里的照片按拍摄月份分到不同的子文件夹',
  '新建一个 Excel 表格，列出这个文件夹里每个文件的名字、大小和修改时间'
]

async function start() {
  connecting.value = true
  connectError.value = ''
  try {
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
      <button @click="$emit('change-folder')">换文件夹</button>
    </header>

    <main ref="scroller">
      <p v-if="connecting" class="muted">正在启动…</p>
      <div v-if="connectError || state.fatal" class="notice error">
        {{ connectError || state.fatal }}
        <button @click="start">重新连接</button>
      </div>

      <div v-if="state.ready && !state.entries.length" class="empty">
        <p>想让它帮你做什么？可以试试：</p>
        <button v-for="ex in examples" :key="ex" class="example" @click="send(ex)">{{ ex }}</button>
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
        @keydown.enter.exact.prevent="submit"
      />
      <button v-if="state.busy" @click="interrupt">停止</button>
      <button v-else class="primary" :disabled="!state.ready || !draft.trim()" @click="submit">发送</button>
    </footer>

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

.muted {
  color: var(--muted);
}

.empty {
  margin: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 520px;
}

.example {
  text-align: left;
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
