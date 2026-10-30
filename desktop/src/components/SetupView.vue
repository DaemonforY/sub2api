<script setup lang="ts">
// First-run checklist: 1) the Codex engine is downloaded, 2) a key is saved, 3) a folder is chosen.
// Step 2 is a temporary paste-a-key form; WeChat QR sign-in replaces it.
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'
import { open } from '@tauri-apps/plugin-dialog'

const emit = defineEmits<{ ready: [dir: string] }>()

interface EngineStatus {
  ready: boolean
  version: string
  path: string
}
interface AccountState {
  signed_in: boolean
  base_url: string
  key_hint: string | null
}
interface Progress {
  stage: 'download' | 'unpack' | 'done'
  downloaded: number
  total: number
  mirror: string
}

const engine = ref<EngineStatus | null>(null)
const account = ref<AccountState | null>(null)
const progress = ref<Progress | null>(null)
const installing = ref(false)
const installError = ref('')
const keyInput = ref('')
const keyError = ref('')
const folder = ref('')
let unlisten: UnlistenFn | null = null

const percent = computed(() => {
  const p = progress.value
  if (!p || !p.total) return 0
  return Math.min(100, Math.round((p.downloaded / p.total) * 100))
})

const mb = (n: number) => (n / 1048576).toFixed(1)

onMounted(async () => {
  unlisten = await listen<Progress>('engine-install-progress', (e) => (progress.value = e.payload))
  engine.value = await invoke<EngineStatus>('engine_status')
  account.value = await invoke<AccountState>('account_state')
  folder.value = await invoke<string>('default_workspace').catch(() => '')
  if (!engine.value.ready) await installEngine()
  // Debug smoke test (HIVEGPT_DEV_PROMPT): skip the start button once everything is ready.
  if ((await invoke<string | null>('dev_autorun')) && canStart.value) emit('ready', folder.value)
})

onUnmounted(() => unlisten?.())

async function installEngine() {
  installing.value = true
  installError.value = ''
  try {
    engine.value = await invoke<EngineStatus>('engine_install')
  } catch (e) {
    installError.value = String(e)
  } finally {
    installing.value = false
  }
}

async function saveKey() {
  keyError.value = ''
  try {
    account.value = await invoke<AccountState>('save_key', { key: keyInput.value })
    keyInput.value = ''
  } catch (e) {
    keyError.value = String(e)
  }
}

async function signOut() {
  account.value = await invoke<AccountState>('sign_out')
}

async function pickFolder() {
  const dir = await open({ directory: true, multiple: false, defaultPath: folder.value || undefined })
  if (typeof dir === 'string') folder.value = dir
}

const canStart = computed(() => engine.value?.ready && account.value?.signed_in && folder.value)
</script>

<template>
  <div class="setup">
    <h1>HiveGPT 桌面助手</h1>
    <p class="lead">用中文告诉它要做什么，它在你的电脑上帮你把事情做完。</p>

    <section class="step">
      <h2><span class="num">1</span>准备 AI 引擎</h2>
      <p v-if="engine?.ready" class="ok">已就绪（版本 {{ engine.version }}）</p>
      <template v-else>
        <p v-if="installing && progress?.stage === 'download'">
          正在下载… {{ mb(progress.downloaded) }} / {{ mb(progress.total) }} MB（{{ percent }}%）
        </p>
        <p v-else-if="installing && progress?.stage === 'unpack'">正在安装…</p>
        <p v-else-if="installing">正在连接下载服务器…</p>
        <div v-if="installing" class="bar"><div :style="{ width: percent + '%' }" /></div>
        <p v-if="installError" class="error">{{ installError }}</p>
        <button v-if="!installing" class="primary" @click="installEngine">
          {{ installError ? '重试' : '开始下载' }}
        </button>
        <p class="hint">第一次使用需要下载约 120 MB，走国内镜像，一般一两分钟。</p>
      </template>
    </section>

    <section class="step">
      <h2><span class="num">2</span>登录</h2>
      <p v-if="account?.signed_in" class="ok">
        已登录（{{ account.key_hint }}）
        <button class="link" @click="signOut">退出</button>
      </p>
      <template v-else>
        <p class="hint">（开发版）微信扫码登录还没接上，先粘贴一个 HiveGPT 的 Key。</p>
        <div class="row">
          <input v-model="keyInput" type="password" placeholder="sk-…" @keyup.enter="saveKey" />
          <button class="primary" :disabled="!keyInput" @click="saveKey">保存</button>
        </div>
        <p v-if="keyError" class="error">{{ keyError }}</p>
      </template>
    </section>

    <section class="step">
      <h2><span class="num">3</span>选一个工作文件夹</h2>
      <p class="hint">它只能读写这个文件夹里的文件。</p>
      <div class="row">
        <code class="path">{{ folder || '还没有选择' }}</code>
        <button @click="pickFolder">换一个</button>
      </div>
    </section>

    <button class="primary start" :disabled="!canStart" @click="emit('ready', folder)">开始使用</button>
  </div>
</template>

<style scoped>
.setup {
  max-width: 640px;
  margin: 0 auto;
  padding: 48px 24px;
}

h1 {
  margin: 0 0 8px;
  font-size: 26px;
}

.lead {
  color: var(--muted);
  margin: 0 0 32px;
}

.step {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 12px;
  padding: 16px 20px;
  margin-bottom: 16px;
}

h2 {
  font-size: 16px;
  margin: 0 0 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.num {
  display: inline-grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--brand);
  color: #fff;
  font-size: 13px;
}

.ok {
  color: #15803d;
  margin: 0;
}

.error {
  color: var(--danger);
  white-space: pre-wrap;
}

.hint {
  color: var(--muted);
  font-size: 13px;
}

.row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.row input {
  flex: 1;
}

.path {
  flex: 1;
  font-size: 13px;
  word-break: break-all;
}

.bar {
  height: 6px;
  background: var(--line);
  border-radius: 3px;
  overflow: hidden;
  margin: 8px 0;
}

.bar div {
  height: 100%;
  background: var(--brand);
  transition: width 0.2s;
}

button.link {
  border: none;
  background: none;
  color: var(--muted);
  text-decoration: underline;
  padding: 0 4px;
}

.start {
  width: 100%;
  padding: 12px;
  font-size: 16px;
}
</style>
