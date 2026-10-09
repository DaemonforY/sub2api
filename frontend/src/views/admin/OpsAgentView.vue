<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card flex flex-col gap-4 p-5" data-testid="ops-agent-chat">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.opsAgent.chatTitle') }}</h3>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.opsAgent.chatHint') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span
              class="rounded-full px-2 py-0.5 text-xs"
              :class="ready ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700'"
            >
              {{ ready ? t('admin.opsAgent.statusOn') : t('admin.opsAgent.statusOff') }}
            </span>
            <span v-if="form.upstream_key_set && form.upstream_url" class="rounded-full bg-primary-50 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
              {{ t('admin.opsAgent.upstreamOn', { name: form.upstream_name || hostOf(form.upstream_url) }) }}
            </span>
            <button v-if="messages.length" class="btn btn-secondary btn-sm" :disabled="busy" @click="clearChat">{{ t('admin.opsAgent.clear') }}</button>
          </div>
        </div>

        <div ref="scroller" class="max-h-[60vh] min-h-[12rem] space-y-4 overflow-y-auto rounded-xl bg-gray-50 p-4 dark:bg-dark-800/60">
          <div v-if="!messages.length" class="space-y-3 py-4 text-center">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ ready ? t('admin.opsAgent.emptyReady') : t('admin.opsAgent.emptyOff') }}</p>
            <div v-if="ready" class="flex flex-wrap justify-center gap-2">
              <button
                v-for="q in quickQuestions"
                :key="q"
                class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs text-gray-700 hover:border-primary-400 hover:text-primary-700 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200"
                @click="ask(q)"
              >
                {{ q }}
              </button>
            </div>
          </div>
          <div v-for="(m, i) in messages" :key="i" :class="m.role === 'user' ? 'flex justify-end' : ''">
            <div v-if="m.role === 'user'" class="max-w-[85%] whitespace-pre-wrap rounded-2xl rounded-tr-sm bg-primary-600 px-3 py-2 text-sm text-white">{{ m.content }}</div>
            <div v-else class="space-y-2">
              <ul v-if="m.tools?.length" class="space-y-0.5 text-xs text-gray-500 dark:text-dark-400">
                <li v-for="(label, j) in m.tools" :key="j" class="flex items-center gap-1.5">
                  <span v-if="busy && i === messages.length - 1 && j === m.tools.length - 1 && !m.content" class="inline-block h-2 w-2 animate-pulse rounded-full bg-primary-500" />
                  <span v-else class="text-green-600">✓</span>
                  {{ label }}
                </li>
              </ul>
              <div
                v-if="m.content"
                class="ops-agent-md break-words rounded-2xl rounded-tl-sm bg-white px-4 py-3 text-sm text-gray-800 shadow-sm dark:bg-dark-700 dark:text-dark-100"
                v-html="renderMarkdown(m.content)"
              />
              <div v-else-if="busy && i === messages.length - 1 && !m.tools?.length" class="text-xs text-gray-400">{{ t('admin.opsAgent.thinking') }}</div>
              <p v-if="m.error" class="text-xs text-red-600">{{ m.error }}</p>
            </div>
          </div>
        </div>

        <form class="flex items-end gap-2" @submit.prevent="ask(draft)">
          <textarea
            v-model="draft"
            rows="2"
            class="input flex-1 resize-none"
            :placeholder="t('admin.opsAgent.placeholder')"
            :disabled="!ready"
            maxlength="2000"
            data-testid="ops-agent-input"
            @keydown.enter.exact.prevent="ask(draft)"
          />
          <button v-if="busy" type="button" class="btn btn-secondary" @click="stop">{{ t('admin.opsAgent.stop') }}</button>
          <button v-else type="submit" class="btn btn-primary" :disabled="!ready || !draft.trim()" data-testid="ops-agent-send">{{ t('admin.opsAgent.send') }}</button>
        </form>
      </div>

      <div class="card p-5" data-testid="ops-agent-settings">
        <button type="button" class="flex w-full items-center justify-between text-left" @click="showSettings = !showSettings">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.opsAgent.settingsTitle') }}</h3>
          <span class="text-sm text-gray-400">{{ showSettings ? '▲' : '▼' }}</span>
        </button>
        <div v-if="showSettings" class="mt-4 space-y-4">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.opsAgent.hint') }}</p>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4" data-testid="ops-agent-enabled" />
            {{ t('admin.opsAgent.enabled') }}
          </label>
          <label class="flex items-start gap-2 text-sm">
            <input v-model="form.auto_alert" type="checkbox" class="mt-0.5 h-4 w-4" data-testid="ops-agent-auto" />
            <span>
              {{ t('admin.opsAgent.autoAlert') }}
              <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t('admin.opsAgent.autoAlertHint') }}</span>
            </span>
          </label>
          <div class="grid gap-4 md:grid-cols-2">
            <label class="text-sm">
              <span class="input-label">{{ t('admin.assistant.key') }}</span>
              <select v-model.number="form.key_id" class="input" data-testid="ops-agent-key">
                <option :value="0">{{ t('admin.assistant.keyPlaceholder') }}</option>
                <option v-if="otherAdminKey" :value="form.key_id">{{ form.key_name || `#${form.key_id}` }}</option>
                <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
              </select>
              <span class="input-hint">
                {{ t('admin.opsAgent.keyHint') }}
                <a href="/keys" target="_blank" class="text-primary-600 hover:underline">{{ t('admin.assistant.keyCreate') }} ↗</a>
              </span>
              <span v-if="form.key_problem" class="input-hint text-red-600">{{ t('admin.assistant.keyProblem', { msg: form.key_problem }) }}</span>
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.assistant.model') }}</span>
              <input v-model.trim="form.model" class="input" placeholder="gpt-5.6-terra" data-testid="ops-agent-model" />
              <span class="input-hint">{{ t('admin.opsAgent.modelHint') }}</span>
            </label>
          </div>

          <div class="space-y-3 rounded-xl border border-gray-100 p-4 dark:border-dark-700">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.opsAgent.upstreamTitle') }}</div>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.opsAgent.upstreamHint') }}</p>
            </div>
            <div class="grid gap-4 md:grid-cols-3">
              <label class="text-sm">
                <span class="input-label">{{ t('admin.opsAgent.upstreamName') }}</span>
                <input v-model.trim="form.upstream_name" class="input" maxlength="40" placeholder="gorustai" />
              </label>
              <label class="text-sm">
                <span class="input-label">{{ t('admin.opsAgent.upstreamUrl') }}</span>
                <input v-model.trim="form.upstream_url" class="input" placeholder="https://gorustai.com" data-testid="ops-agent-upstream-url" />
              </label>
              <label class="text-sm">
                <span class="input-label">{{ t('admin.opsAgent.upstreamKey') }}</span>
                <input
                  v-model="upstreamKey"
                  type="password"
                  autocomplete="new-password"
                  class="input"
                  :placeholder="form.upstream_key_set ? t('admin.opsAgent.upstreamKeyKeep') : t('admin.opsAgent.upstreamKeyPlaceholder')"
                  data-testid="ops-agent-upstream-key"
                />
                <span class="input-hint">{{ t('admin.opsAgent.upstreamKeyHint') }}</span>
              </label>
            </div>
            <div class="flex flex-wrap items-center gap-3 text-sm">
              <label v-if="form.upstream_key_set" class="flex items-center gap-2">
                <input v-model="clearKey" type="checkbox" class="h-4 w-4" />
                {{ t('admin.opsAgent.upstreamKeyClear') }}
              </label>
              <button class="btn btn-secondary btn-sm" :disabled="testing || !form.upstream_key_set" data-testid="ops-agent-test" @click="test">
                {{ testing ? t('admin.opsAgent.testing') : t('admin.opsAgent.test') }}
              </button>
              <span v-if="testResult" class="text-xs" :class="testResult.ok ? 'text-green-600' : 'text-red-600'">{{ testResult.text }}</span>
            </div>
          </div>

          <button class="btn btn-primary" :disabled="saving || loading" data-testid="ops-agent-save" @click="save">{{ t('common.save') }}</button>
        </div>
      </div>

      <div class="card p-5" data-testid="ops-agent-runs">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.opsAgent.runsTitle') }}</h3>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.opsAgent.runsHint') }}</p>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <button class="btn btn-secondary btn-sm" :disabled="runsPage <= 1 || runsLoading" @click="loadRuns(runsPage - 1)">‹</button>
            <span class="text-gray-500">{{ runsPage }} / {{ runsPages }}</span>
            <button class="btn btn-secondary btn-sm" :disabled="runsPage >= runsPages || runsLoading" @click="loadRuns(runsPage + 1)">›</button>
          </div>
        </div>
        <p v-if="!runs.length && !runsLoading" class="py-6 text-center text-sm text-gray-400">{{ t('admin.opsAgent.runsEmpty') }}</p>
        <ul class="divide-y divide-gray-100 dark:divide-dark-700">
          <li v-for="run in runs" :key="run.id" class="py-2.5">
            <button type="button" class="flex w-full items-start gap-3 text-left" @click="toggle(run.id)">
              <span class="w-28 flex-shrink-0 text-xs text-gray-400">{{ formatTime(run.created_at) }}</span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm text-gray-900 dark:text-white">{{ run.question }}</span>
                <span class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                  <span v-if="run.user_id === null" class="rounded bg-amber-50 px-1.5 py-0.5 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">{{ t('admin.opsAgent.auto') }}</span>
                  <span v-else>{{ run.user_email }}</span>
                  <span
                    v-for="(s, i) in run.steps"
                    :key="i"
                    class="rounded px-1.5 py-0.5"
                    :class="s.error ? 'bg-red-50 text-red-600 dark:bg-red-900/20' : 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'"
                  >{{ s.tool }}</span>
                  <span>{{ t('admin.assistant.runUsage', { calls: run.model_calls, tokens: run.prompt_tokens + run.completion_tokens, s: (run.duration_ms / 1000).toFixed(1) }) }}</span>
                  <span v-if="run.status === 'error'" class="text-red-600">{{ run.error }}</span>
                </span>
              </span>
            </button>
            <div v-if="expanded === run.id" class="mt-2 space-y-2 pl-0 text-xs sm:pl-[7.75rem]">
              <div v-for="(s, i) in run.steps" :key="i" class="rounded-lg bg-gray-50 p-2 dark:bg-dark-800">
                <div class="font-medium text-gray-700 dark:text-dark-200">
                  {{ s.tool }} <span class="font-normal text-gray-400">{{ s.args }} · {{ s.ms }} ms</span>
                </div>
                <pre class="mt-1 max-h-48 overflow-auto whitespace-pre-wrap break-all text-gray-600 dark:text-dark-300">{{ s.error || s.result }}</pre>
              </div>
              <div
                class="ops-agent-md rounded-lg border border-gray-100 p-3 text-sm text-gray-800 dark:border-dark-700 dark:text-dark-100"
                v-html="renderMarkdown(run.answer || '—')"
              />
            </div>
          </li>
        </ul>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import * as opsAgentAPI from '@/api/admin/opsAgent'
import type { AgentRun, AssistantKeyOption, OpsAgentSettings } from '@/api/admin/opsAgent'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const form = reactive<OpsAgentSettings>({
  enabled: false,
  model: 'gpt-5.6-terra',
  key_id: 0,
  auto_alert: false,
  upstream_name: '',
  upstream_url: '',
  upstream_key_set: false
})
const keys = ref<AssistantKeyOption[]>([])
const loading = ref(true)
const saving = ref(false)
const showSettings = ref(false)
const upstreamKey = ref('')
const clearKey = ref(false)
const testing = ref(false)
const testResult = ref<{ ok: boolean; text: string } | null>(null)

const ready = computed(() => form.enabled && form.key_id > 0 && !form.key_problem)
const otherAdminKey = computed(
  () => form.key_id > 0 && !!form.key_owner && form.key_owner !== authStore.user?.id && !keys.value.some((k) => k.id === form.key_id)
)

function hostOf(url: string) {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

async function load() {
  loading.value = true
  try {
    const [settings, options] = await Promise.all([opsAgentAPI.getSettings(), opsAgentAPI.listKeys()])
    Object.assign(form, settings)
    keys.value = options
    showSettings.value = !settings.enabled
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const out = await opsAgentAPI.saveSettings({ ...form, upstream_key: upstreamKey.value.trim(), clear_upstream_key: clearKey.value })
    Object.assign(form, out)
    upstreamKey.value = ''
    clearKey.value = false
    testResult.value = null
    appStore.showSuccess(t('admin.opsAgent.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  testResult.value = null
  try {
    const out = await opsAgentAPI.testUpstream()
    testResult.value = {
      ok: true,
      text: t('admin.opsAgent.testOk', { name: out.name, success: out.success, errors: out.errors, rate: out.error_rate_pct })
    }
  } catch (e) {
    testResult.value = { ok: false, text: extractApiErrorMessage(e, t('common.error')) }
  } finally {
    testing.value = false
  }
}

// ---- conversation (kept for this browser tab) ----

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  tools?: string[]
  error?: string
}

const storageKey = 'ops-agent-chat'
const messages = ref<ChatMessage[]>(loadChat())
const draft = ref('')
const busy = ref(false)
const scroller = ref<HTMLElement | null>(null)
let controller: AbortController | null = null

const quickQuestions = computed(() => [
  t('admin.opsAgent.q1'),
  t('admin.opsAgent.q2'),
  t('admin.opsAgent.q3'),
  t('admin.opsAgent.q4')
])

function loadChat(): ChatMessage[] {
  try {
    const raw = sessionStorage.getItem(storageKey)
    return raw ? (JSON.parse(raw) as ChatMessage[]) : []
  } catch {
    return []
  }
}

function persist() {
  try {
    sessionStorage.setItem(storageKey, JSON.stringify(messages.value.slice(-20)))
  } catch {
    // storage full or blocked
  }
}

function scrollDown() {
  nextTick(() => {
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
  })
}

function clearChat() {
  messages.value = []
  persist()
}

async function ask(text: string) {
  const question = text.trim()
  if (!question || busy.value || !ready.value) return
  draft.value = ''
  const history = messages.value
    .filter((m) => m.content && !m.error)
    .map((m) => ({ role: m.role, content: m.content }))
    .slice(-10)
  messages.value.push({ role: 'user', content: question })
  const reply = reactive<ChatMessage>({ role: 'assistant', content: '', tools: [] })
  messages.value.push(reply)
  busy.value = true
  controller = new AbortController()
  scrollDown()
  try {
    await opsAgentAPI.streamOpsAgent(
      [...history, { role: 'user', content: question }],
      (delta) => {
        reply.content += delta
        scrollDown()
      },
      (label) => {
        reply.tools!.push(label)
        scrollDown()
      },
      controller.signal
    )
  } catch (e) {
    reply.error = controller?.signal.aborted ? t('admin.opsAgent.stopped') : (e as Error).message
  } finally {
    busy.value = false
    controller = null
    persist()
    scrollDown()
    loadRuns(1)
  }
}

function stop() {
  controller?.abort()
}

// ---- run log ----

const runs = ref<AgentRun[]>([])
const runsTotal = ref(0)
const runsPage = ref(1)
const runsLoading = ref(false)
const expanded = ref<number | null>(null)
const runsPages = computed(() => Math.max(1, Math.ceil(runsTotal.value / 20)))

async function loadRuns(page = 1) {
  runsLoading.value = true
  try {
    const out = await opsAgentAPI.listRuns(page)
    runs.value = out.items
    runsTotal.value = out.total
    runsPage.value = page
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    runsLoading.value = false
  }
}

function toggle(id: number) {
  expanded.value = expanded.value === id ? null : id
}

function formatTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

onMounted(() => {
  load()
  loadRuns()
  scrollDown()
})

onBeforeUnmount(() => controller?.abort())
</script>

<style scoped>
.ops-agent-md :deep(p) {
  margin: 0.35rem 0;
}
.ops-agent-md :deep(h1),
.ops-agent-md :deep(h2),
.ops-agent-md :deep(h3),
.ops-agent-md :deep(h4) {
  margin: 0.6rem 0 0.3rem;
  font-weight: 600;
}
.ops-agent-md :deep(ul),
.ops-agent-md :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.25rem;
}
.ops-agent-md :deep(ul) {
  list-style: disc;
}
.ops-agent-md :deep(ol) {
  list-style: decimal;
}
.ops-agent-md :deep(table) {
  margin: 0.4rem 0;
  border-collapse: collapse;
  font-size: 0.85em;
}
.ops-agent-md :deep(th),
.ops-agent-md :deep(td) {
  border: 1px solid rgba(0, 0, 0, 0.1);
  padding: 0.2rem 0.5rem;
}
:global(.dark) .ops-agent-md :deep(th),
:global(.dark) .ops-agent-md :deep(td) {
  border-color: rgba(255, 255, 255, 0.15);
}
.ops-agent-md :deep(code) {
  border-radius: 0.25rem;
  background: rgba(0, 0, 0, 0.06);
  padding: 0 0.25rem;
  font-size: 0.85em;
}
:global(.dark) .ops-agent-md :deep(code) {
  background: rgba(255, 255, 255, 0.1);
}
.ops-agent-md :deep(pre) {
  margin: 0.375rem 0;
  overflow-x: auto;
  border-radius: 0.5rem;
  background: #1f2937;
  padding: 0.5rem 0.75rem;
  color: #f9fafb;
}
.ops-agent-md :deep(pre code) {
  background: transparent;
  padding: 0;
}
.ops-agent-md :deep(strong) {
  font-weight: 600;
}
</style>
