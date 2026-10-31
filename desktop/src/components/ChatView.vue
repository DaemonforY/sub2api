<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { openPath, openUrl } from '@tauri-apps/plugin-opener'
import { createChat, type Approval, type ThreadSummary, type UndoPreview } from '../lib/chat'
import { TEMPLATES, type TaskTemplate } from '../lib/templates'
import { syncTemplates } from '../lib/skills'
import { money } from '../lib/usage'
import { track } from '../lib/track'

const props = defineProps<{ workspace: string }>()
const emit = defineEmits<{ 'change-folder': []; relogin: [] }>()

const { state, connect, send, interrupt, answer, newThread, openThread, renameThread, archiveThread, refreshUsage, undo, undoPreview } =
  createChat()
const draft = ref('')
const connecting = ref(false)
const connectError = ref('')
const scroller = ref<HTMLElement | null>(null)

const picked = ref<TaskTemplate | null>(null)
/** Skills published in the admin (synced on start); the built-in ones until then or when offline. */
const templates = ref<TaskTemplate[]>(TEMPLATES)
const composer = ref<HTMLTextAreaElement | null>(null)

/** Own files: put the prompt in the composer so it can be read and tweaked before sending. */
async function useOwnFiles(t: TaskTemplate) {
  picked.value = null
  draft.value = t.prompt
  draftTemplate.value = t
  await nextTick()
  composer.value?.focus()
}

/** Sample: makes fake files in 「示例」 and runs the task on them straight away. */
function trySample(t: TaskTemplate) {
  picked.value = null
  send(t.samplePrompt, { template: `sample:${t.id}`, title: `${t.title}（示例）` })
}

/** The template whose prompt is in the composer, if the user hasn't replaced it. */
const draftTemplate = ref<TaskTemplate | null>(null)

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
    // Skills first: Codex reads the SKILL.md files when the engine starts.
    templates.value = await syncTemplates()
    await connect(props.workspace)
    const devPrompt = await invoke<string | null>('dev_autorun')
    if (devPrompt && !state.entries.length) {
      await send(devPrompt)
      if (await invoke<boolean>('dev_undo')) {
        const stop = watch(
          () => state.busy,
          (busy) => {
            if (busy || state.turnCount === 0) return
            stop()
            setTimeout(confirmUndo, 3000)
          }
        )
      }
    }
  } catch (e) {
    connectError.value = String((e as Error).message ?? e)
  } finally {
    connecting.value = false
  }
}

const baseUrl = ref('https://hivegpt.cn')

onMounted(async () => {
  const account = await invoke<{ base_url: string }>('account_state').catch(() => null)
  if (account?.base_url) baseUrl.value = account.base_url
  start()
})

function recharge() {
  track('desktop_recharge_click', { low: Boolean(state.usage?.low) })
  openUrl(`${baseUrl.value}/purchase`)
}

const usageText = computed(() => {
  const u = state.usage
  if (!u) return ''
  const today = `今日 ${money(u.today)}`
  if (!u.subscription) return `余额 ${money(u.remaining)} · ${today}`
  return u.remaining !== null ? `${u.plan} · 剩 ${money(u.remaining)} · ${today}` : `${u.plan} · ${today}`
})

// ---- history sidebar ----
/** The rename box takes focus with its text selected. */
const vFocus = { mounted: (el: HTMLInputElement) => (el.focus(), el.select()) }

/** A draft (e.g. a message put back by undo) belongs to the conversation it was written in. */
async function open(t: ThreadSummary) {
  if (editing.value === t.id || t.id === state.threadId || state.busy || state.working) return
  draft.value = ''
  draftTemplate.value = null
  await openThread(t.id)
}

const editing = ref('')
const editName = ref('')

function startRename(t: ThreadSummary) {
  editing.value = t.id
  editName.value = t.title.replace(/…$/, '')
}

async function saveRename() {
  const id = editing.value
  editing.value = ''
  if (id) await renameThread(id, editName.value)
}

async function startNew() {
  if (state.busy || state.working) return
  draft.value = ''
  draftTemplate.value = null
  await newThread()
}

const folderName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() ?? p

function when(ms: number): string {
  if (!ms) return ''
  const d = new Date(ms)
  const now = new Date()
  const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString()
  if (sameDay(d, now)) return d.toTimeString().slice(0, 5)
  const y = new Date(now)
  y.setDate(now.getDate() - 1)
  if (sameDay(d, y)) return '昨天'
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

// ---- undo ----
const undoAsk = ref<UndoPreview | null>(null)
const canUndo = computed(() => state.ready && !state.busy && !state.working && state.turnCount > 0 && !state.approvals.length)

async function askUndo() {
  undoAsk.value = await undoPreview()
}

async function confirmUndo() {
  undoAsk.value = null
  const text = await undo()
  if (text && !draft.value.trim()) {
    draft.value = text
    draftTemplate.value = null
    await nextTick()
    composer.value?.focus()
  }
}

async function relogin() {
  await invoke('sign_out').catch(() => {})
  emit('relogin')
}

function submit() {
  const text = draft.value
  const t = draftTemplate.value
  draft.value = ''
  draftTemplate.value = null
  // Still the template's prompt (possibly edited a little): count it as that template.
  send(text, t ? { template: t.id, title: t.title } : {})
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
    <aside class="side">
      <button class="new" :disabled="state.busy || state.working || !state.ready" @click="startNew">＋ 新对话</button>
      <p class="side-title">历史对话</p>
      <p v-if="state.ready && !state.threads.length" class="side-empty">还没有。做完的事情会出现在这里，随时可以接着聊。</p>
      <ul>
        <li
          v-for="t in state.threads"
          :key="t.id"
          :class="{ active: t.id === state.threadId }"
          :title="t.cwd"
          @click="open(t)"
        >
          <input
            v-if="editing === t.id"
            v-model="editName"
            v-focus
            class="rename"
            maxlength="40"
            @keydown.enter.prevent="saveRename"
            @keydown.esc.prevent="editing = ''"
            @blur="saveRename"
            @click.stop
          />
          <template v-else>
            <span class="t-title">{{ t.title }}</span>
            <span class="t-meta">{{ when(t.updatedAt) }}<template v-if="t.cwd !== state.cwd"> · 📁 {{ folderName(t.cwd) }}</template></span>
            <span class="t-actions">
              <button title="改名" @click.stop="startRename(t)">✎</button>
              <button title="从列表移除（归档）" @click.stop="archiveThread(t.id)">✕</button>
            </span>
          </template>
        </li>
      </ul>
    </aside>

    <div class="pane">
    <header>
      <div class="folder" :title="state.cwd || workspace">📁 {{ state.cwd || workspace }}</div>
      <button
        v-if="usageText"
        class="usage"
        :class="{ low: state.usage?.low }"
        title="点一下刷新"
        @click="refreshUsage"
      >{{ usageText }}</button>
      <button v-if="state.usage?.low" class="primary" @click="recharge">充值</button>
      <button @click="openPath(state.cwd || workspace)">打开文件夹</button>
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
          <button v-for="t in templates" :key="t.id" class="card" @click="picked = t">
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
        <div v-else-if="e.kind === 'notice'" class="notice" :class="e.tone">
          {{ e.text }}
          <button v-if="e.action === 'recharge'" class="link" @click="recharge">去充值</button>
          <button v-else-if="e.action === 'open' && e.path" class="link" @click="openPath(e.path)">打开撤销回收站</button>
        </div>
      </template>

      <div v-if="canUndo" class="undo-row">
        <button class="link" title="把文件夹和对话都恢复到这一轮开始前" @click="askUndo">↶ 撤销这一轮</button>
      </div>
      <p v-if="state.working" class="muted">请稍候…</p>
    </main>

    <footer>
      <textarea
        v-model="draft"
        rows="3"
        placeholder="用中文描述你要做的事，回车发送，Shift+回车换行"
        :disabled="!state.ready || state.working"
        ref="composer"
        @keydown.enter.exact.prevent="submit"
      />
      <button v-if="state.busy" @click="interrupt">停止</button>
      <button v-else class="primary" :disabled="!state.ready || state.working || !draft.trim()" @click="submit">发送</button>
    </footer>
    </div>

    <div v-if="undoAsk" class="overlay" @click.self="undoAsk = null">
      <div class="dialog">
        <h3>撤销这一轮？</h3>
        <p v-if="undoAsk.files">
          文件夹会恢复到这一轮开始前的样子：改过的文件改回去，删掉的放回来；这一轮新建的文件不会删除，会挪到「撤销回收站」。对话也会退回到这一轮之前，你的这条消息会放回输入框。
        </p>
        <p v-else class="reason">{{ undoAsk.reason }}</p>
        <div class="actions">
          <button @click="undoAsk = null">取消</button>
          <button class="primary" @click="confirmUndo">撤销</button>
        </div>
      </div>
    </div>

    <div v-if="picked" class="overlay" @click.self="picked = null">
      <div class="dialog">
        <h3>{{ picked.icon }} {{ picked.title }}</h3>
        <p>{{ picked.desc }}。</p>
        <p class="reason">用自己的文件：{{ picked.prepare }}结果会放在文件夹里的「输出」中，原文件不会被改动。</p>
        <div class="actions">
          <button @click="picked = null">取消</button>
          <button v-if="picked.samplePrompt" @click="trySample(picked)">先用示例试试</button>
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
}

.pane {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.side {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid var(--line);
  background: var(--bg);
  display: flex;
  flex-direction: column;
  padding: 10px 8px;
  overflow-y: auto;
}

.side .new {
  width: 100%;
}

.side-title {
  margin: 14px 6px 6px;
  font-size: 12px;
  color: var(--muted);
}

.side-empty {
  margin: 0 6px;
  font-size: 12px;
  color: var(--muted);
  line-height: 1.6;
}

.side ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.side li {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 7px 8px;
  border-radius: 8px;
  cursor: pointer;
}

.side li:hover {
  background: var(--panel);
}

.side li.active {
  background: #fef3c7;
}

.t-title {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 36px;
}

.t-meta {
  font-size: 11px;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.t-actions {
  position: absolute;
  top: 5px;
  right: 4px;
  display: none;
  gap: 2px;
}

.side li:hover .t-actions {
  display: flex;
}

.t-actions button {
  padding: 0 5px;
  font-size: 12px;
  line-height: 20px;
  border: none;
  background: transparent;
  color: var(--muted);
}

.t-actions button:hover {
  color: var(--text, #111);
}

.rename {
  width: 100%;
  font-size: 13px;
}

.usage {
  font-size: 12px;
  color: var(--muted);
  white-space: nowrap;
}

.usage.low {
  color: var(--danger);
  border-color: var(--danger);
}

.link {
  border: none;
  background: transparent;
  color: var(--brand);
  padding: 0 4px;
  cursor: pointer;
}

.undo-row {
  align-self: flex-end;
  font-size: 13px;
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
