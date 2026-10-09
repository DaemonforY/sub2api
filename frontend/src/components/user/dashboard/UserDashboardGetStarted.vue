<template>
  <section v-if="visible" class="card overflow-hidden" data-testid="dashboard-get-started">
    <div class="flex flex-wrap items-start justify-between gap-2 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ allDone ? t('getStarted.doneTitle') : t('getStarted.title') }}</h2>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ allDone ? t('getStarted.doneSubtitle') : t('getStarted.subtitle') }}</p>
      </div>
      <button type="button" class="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-dark-200" data-testid="get-started-dismiss" @click="dismiss">
        {{ allDone ? t('getStarted.finish') : t('getStarted.later') }}
      </button>
    </div>

    <!-- Most sign-ups come to draw, not to code: offer the canvas before the Codex steps. -->
    <div v-if="!allDone" class="mx-4 mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-gradient-to-r from-fuchsia-50 to-violet-50 px-4 py-3 dark:from-fuchsia-900/15 dark:to-violet-900/15" data-testid="get-started-canvas">
      <div class="min-w-0">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('getStarted.canvas.title') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('getStarted.canvas.desc') }}</p>
      </div>
      <a :href="canvasHref" target="_blank" rel="noopener noreferrer" class="btn btn-sm btn-primary shrink-0" data-testid="get-started-canvas-link">
        {{ t('getStarted.canvas.action') }} ↗
      </a>
    </div>

    <!-- Chat in a desktop client: one-click import of the starter key -->
    <div v-if="!allDone && starterKey" class="mx-4 mt-3 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-sky-50 px-4 py-3 dark:bg-sky-900/15" data-testid="get-started-chat">
      <div class="min-w-0">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('getStarted.chat.title') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('getStarted.chat.desc') }}</p>
      </div>
      <span class="flex shrink-0 gap-2">
        <a :href="chatLinks.cherry" class="btn btn-sm btn-secondary" data-testid="get-started-cherry" @click="track('key_config_copied', { client: 'cherry-studio', os: 'get-started' })">Cherry Studio</a>
        <a :href="chatLinks.chatbox" class="btn btn-sm btn-secondary" data-testid="get-started-chatbox" @click="track('key_config_copied', { client: 'chatbox', os: 'get-started' })">Chatbox</a>
      </span>
    </div>

    <ol class="grid gap-3 p-4 md:grid-cols-3">
      <li v-for="(s, i) in steps" :key="s.key" class="flex gap-3 rounded-xl p-4" :class="s.done ? 'bg-emerald-50 dark:bg-emerald-900/15' : s.current ? 'bg-primary-50 ring-1 ring-primary-200 dark:bg-primary-900/20 dark:ring-primary-800' : 'bg-gray-50 dark:bg-dark-800/50'" :data-testid="`get-started-step-${s.key}`" :data-done="s.done">
        <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full text-sm font-semibold" :class="s.done ? 'bg-emerald-500 text-white' : 'bg-white text-gray-600 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-dark-200 dark:ring-dark-600'">
          <Icon v-if="s.done" name="check" size="sm" />
          <template v-else>{{ i + 1 }}</template>
        </span>
        <div class="min-w-0 space-y-1.5">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`getStarted.steps.${s.key}.title`) }}</p>
          <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ stepText(s) }}</p>
          <router-link v-if="s.key === 'key' && !s.done && starterFailed" to="/keys?action=create" class="btn btn-sm btn-primary" data-testid="get-started-go-key">
            {{ t('getStarted.steps.key.action') }}
          </router-link>
          <button v-if="s.key === 'config' && hasKey && !configOpen" type="button" class="btn btn-sm" :class="s.current ? 'btn-primary' : 'btn-secondary'" data-testid="get-started-go-config" @click="openConfig">
            {{ t('getStarted.steps.config.action') }}
          </button>
          <p v-if="s.key === 'call' && !s.done && waiting" class="flex items-center gap-1.5 text-xs text-primary-600 dark:text-primary-400" data-testid="get-started-waiting">
            <span class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>{{ t('getStarted.waiting') }}
          </p>
        </div>
      </li>
    </ol>

    <!-- Set up Codex without a terminal: download config.toml, drop it into the .codex folder. -->
    <div v-if="configOpen && starterKey" class="mx-4 mb-4 space-y-3 rounded-xl border border-primary-200 bg-white p-4 dark:border-primary-900/40 dark:bg-dark-800" data-testid="get-started-config">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('getStarted.codex.title') }}</p>
        <div class="flex gap-1" role="tablist">
          <button
            v-for="o in OS_LIST"
            :key="o"
            type="button"
            role="tab"
            class="rounded-md px-2.5 py-1 text-xs"
            :class="os === o ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'"
            :data-testid="`get-started-os-${o}`"
            @click="os = o"
          >
            {{ t(`getStarted.codex.os.${o}`) }}
          </button>
        </div>
      </div>
      <ol class="space-y-2.5 text-sm text-gray-700 dark:text-dark-200">
        <li class="flex gap-2">
          <span class="font-semibold text-primary-600">1</span>
          <span>{{ t('getStarted.codex.install') }}<a href="/learn/codex/hivegpt" target="_blank" rel="noopener" class="ml-1 text-primary-600 hover:underline">{{ t('getStarted.codex.installLink') }} ↗</a></span>
        </li>
        <li class="flex flex-wrap items-center gap-2">
          <span class="font-semibold text-primary-600">2</span>
          <span>{{ t('getStarted.codex.download') }}</span>
          <button type="button" class="btn btn-sm btn-primary" data-testid="get-started-download" @click="downloadConfig">⬇ config.toml</button>
        </li>
        <li class="flex gap-2">
          <span class="font-semibold text-primary-600">3</span>
          <span class="space-y-1">
            <span class="block">{{ t(`getStarted.codex.open.${os}`) }}</span>
            <span class="inline-flex items-center gap-2">
              <code class="rounded bg-gray-100 px-2 py-0.5 font-mono text-xs dark:bg-dark-700" data-testid="get-started-folder">{{ folder }}</code>
              <button type="button" class="text-xs text-primary-600 hover:underline" @click="copyFolder">{{ folderCopied ? t('getStarted.codex.copied') : t('getStarted.codex.copy') }}</button>
            </span>
          </span>
        </li>
        <li class="flex gap-2">
          <span class="font-semibold text-primary-600">4</span>
          <span>{{ t('getStarted.codex.drop') }}</span>
        </li>
        <li class="flex gap-2">
          <span class="font-semibold text-primary-600">5</span>
          <span>{{ t('getStarted.codex.restart') }}</span>
        </li>
      </ol>
      <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">
        {{ t('getStarted.codex.notes') }}
        <router-link to="/keys?action=use" class="text-primary-600 hover:underline">{{ t('getStarted.codex.otherTools') }}</router-link>
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-x-5 gap-y-1 border-t border-gray-100 px-6 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
      <template v-if="allDone">
        <router-link to="/usage" class="text-primary-600 hover:underline">{{ t('getStarted.viewUsage') }}</router-link>
        <a href="/learn/" target="_blank" rel="noopener" class="text-primary-600 hover:underline">{{ t('getStarted.learnMore') }} ↗</a>
      </template>
      <template v-else>
        <span v-if="needsCredit" class="text-amber-600 dark:text-amber-400" data-testid="get-started-credit">
          {{ t('getStarted.needCredit') }}
          <router-link to="/purchase" class="underline">{{ t('getStarted.buy') }}</router-link>
        </span>
        <a href="/learn/codex/hivegpt" target="_blank" rel="noopener" class="text-primary-600 hover:underline">{{ t('getStarted.tutorial') }} ↗</a>
        <a href="/home" target="_blank" rel="noopener" class="text-primary-600 hover:underline">{{ t('getStarted.askAssistant') }} ↗</a>
      </template>
      <a href="/learn/scenes/?utm_source=dashboard&utm_medium=get-started" target="_blank" rel="noopener" class="text-primary-600 hover:underline" data-testid="get-started-scenes">{{ t('getStarted.scenes') }} ↗</a>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI } from '@/api/usage'
import { keysAPI } from '@/api/keys'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { guideStepDone, markGuideStep } from '@/utils/getStarted'
import { codexConfigFolder, codexStarterConfig, detectDesktopOS, downloadTextFile, type DesktopOS } from '@/utils/codexStarterConfig'
import { track } from '@/utils/analytics'
import { canvasUrl } from '@/constants/crossSites'
import { chatboxImportLink, cherryStudioImportLink } from '@/utils/chatClientImport'
import type { ApiKey } from '@/types'

const props = defineProps<{
  userId: number
  balance: number
  apiKeys: number
  requests: number
}>()

const { t } = useI18n()
const canvasHref = canvasUrl({ medium: 'dashboard-guide', baseUrl: window.location.origin, path: '/image' })
const appStore = useAppStore()
const subscriptions = useSubscriptionStore()

const POLL_MS = 10_000
const POLL_FOR_MS = 15 * 60_000
const OS_LIST: DesktopOS[] = ['windows', 'mac', 'linux']

const keys = ref(props.apiKeys)
const requests = ref(props.requests)
const copied = ref(guideStepDone('copied', props.userId))
const dismissed = ref(guideStepDone('dismissed', props.userId))
// Shown to people who haven't made a call yet, and kept until closed once they do.
const startedEmpty = props.requests === 0
watch(() => [props.apiKeys, props.requests], ([k, r]) => {
  keys.value = Math.max(keys.value, k)
  requests.value = Math.max(requests.value, r)
})

// The key the config uses: made for the user when they have none.
const starterKey = ref<ApiKey | null>(null)
const chatLinks = computed(() => {
  const base = appStore.cachedPublicSettings?.api_base_url || window.location.origin
  const key = starterKey.value?.key || ''
  return { cherry: cherryStudioImportLink(base, key), chatbox: chatboxImportLink(base, key) }
})
const starterCreated = ref(false)
const starterFailed = ref(false)
const configOpen = ref(false)
const os = ref<DesktopOS>(detectDesktopOS(typeof navigator === 'undefined' ? '' : navigator.userAgent))
const folder = computed(() => codexConfigFolder(os.value))
const folderCopied = ref(false)

const hasKey = computed(() => keys.value > 0)
const hasCall = computed(() => requests.value > 0)
const allDone = computed(() => hasKey.value && hasCall.value)
const visible = computed(() => !dismissed.value && startedEmpty)
const needsCredit = computed(() => props.balance <= 0 && !subscriptions.hasActiveSubscriptions)

type Step = { key: 'key' | 'config' | 'call'; done: boolean; current: boolean }
const steps = computed<Step[]>(() => {
  const copiedDone = copied.value || hasCall.value
  return [
    { key: 'key', done: hasKey.value, current: !hasKey.value },
    { key: 'config', done: copiedDone, current: hasKey.value && !copiedDone },
    { key: 'call', done: hasCall.value, current: hasKey.value && copiedDone && !hasCall.value }
  ]
})

function stepText(s: Step): string {
  if (s.key === 'key' && s.done && starterCreated.value) return t('getStarted.steps.key.created', { name: starterKey.value?.name || '' })
  if (s.key === 'key' && !s.done && !starterFailed.value) return t('getStarted.steps.key.preparing')
  return s.done ? t(`getStarted.steps.${s.key}.done`) : t(`getStarted.steps.${s.key}.hint`)
}

async function ensureStarterKey(): Promise<void> {
  try {
    const res = await keysAPI.starter()
    starterKey.value = res.key
    starterCreated.value = res.created
    keys.value = Math.max(keys.value, 1)
    if (res.created) track('key_created', { source: 'starter' })
  } catch {
    starterFailed.value = true
  }
}

function openConfig(): void {
  configOpen.value = true
  if (!starterKey.value) void ensureStarterKey()
  track('guide_config_open', { os: os.value })
}

function downloadConfig(): void {
  if (!starterKey.value) return
  const base = appStore.cachedPublicSettings?.api_base_url || window.location.origin
  downloadTextFile('config.toml', codexStarterConfig(base, starterKey.value.key))
  markGuideStep('copied', props.userId)
  copied.value = true
  track('key_config_copied', { client: 'codex', via: 'download', os: os.value })
}

async function copyFolder(): Promise<void> {
  try {
    await navigator.clipboard.writeText(folder.value)
    folderCopied.value = true
    setTimeout(() => (folderCopied.value = false), 2000)
  } catch {
    window.prompt(t('getStarted.codex.copy'), folder.value)
  }
}

// Waiting for the first call: check every 10 s for up to 15 minutes while the tab is visible.
const waiting = computed(() => visible.value && hasKey.value && !hasCall.value)
let timer: ReturnType<typeof setInterval> | null = null
let pollStarted = 0

async function check(): Promise<void> {
  if (document.visibilityState === 'hidden') return
  if (Date.now() - pollStarted > POLL_FOR_MS) return stopPolling()
  try {
    const s = await usageAPI.getDashboardStats()
    keys.value = Math.max(keys.value, s.total_api_keys)
    requests.value = Math.max(requests.value, s.total_requests)
  } catch {
    // try again next tick
  }
}

function startPolling(): void {
  if (timer) return
  pollStarted = Date.now()
  timer = setInterval(check, POLL_MS)
}

function stopPolling(): void {
  if (timer) clearInterval(timer)
  timer = null
}

watch(waiting, (w) => (w ? startPolling() : stopPolling()), { immediate: true })
watch(hasCall, (done, before) => {
  if (done && !before && visible.value) track('guide_first_call')
})

function onStorage(e: StorageEvent): void {
  // "Copy config" done in another tab (the keys page opened from here).
  if (e.key?.startsWith('hg_guide_copied_')) copied.value = guideStepDone('copied', props.userId)
}

function dismiss(): void {
  markGuideStep('dismissed', props.userId)
  dismissed.value = true
  track('guide_dismiss', { done: allDone.value })
}

onMounted(() => {
  window.addEventListener('storage', onStorage)
  if (visible.value) {
    track('guide_view', { step: steps.value.find((s) => !s.done)?.key || 'done' })
    void subscriptions.fetchActiveSubscriptions().catch(() => undefined)
    // No key yet: make one now, so step 1 is already done and the Codex setup opens right away.
    if (!hasKey.value) {
      void ensureStarterKey().then(() => {
        if (starterKey.value) configOpen.value = true
      })
    }
  }
})

onUnmounted(() => {
  stopPolling()
  window.removeEventListener('storage', onStorage)
})
</script>
