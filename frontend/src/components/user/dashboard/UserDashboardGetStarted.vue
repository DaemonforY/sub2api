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

    <ol class="grid gap-3 p-4 md:grid-cols-3">
      <li v-for="(s, i) in steps" :key="s.key" class="flex gap-3 rounded-xl p-4" :class="s.done ? 'bg-emerald-50 dark:bg-emerald-900/15' : s.current ? 'bg-primary-50 ring-1 ring-primary-200 dark:bg-primary-900/20 dark:ring-primary-800' : 'bg-gray-50 dark:bg-dark-800/50'" :data-testid="`get-started-step-${s.key}`" :data-done="s.done">
        <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full text-sm font-semibold" :class="s.done ? 'bg-emerald-500 text-white' : 'bg-white text-gray-600 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-dark-200 dark:ring-dark-600'">
          <Icon v-if="s.done" name="check" size="sm" />
          <template v-else>{{ i + 1 }}</template>
        </span>
        <div class="min-w-0 space-y-1.5">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`getStarted.steps.${s.key}.title`) }}</p>
          <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ s.done ? t(`getStarted.steps.${s.key}.done`) : t(`getStarted.steps.${s.key}.hint`) }}</p>
          <router-link v-if="!s.done && s.to" :to="s.to" class="btn btn-sm" :class="s.current ? 'btn-primary' : 'btn-secondary'" :data-testid="`get-started-go-${s.key}`">
            {{ t(`getStarted.steps.${s.key}.action`) }}
          </router-link>
          <p v-if="s.key === 'call' && !s.done && waiting" class="flex items-center gap-1.5 text-xs text-primary-600 dark:text-primary-400" data-testid="get-started-waiting">
            <span class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>{{ t('getStarted.waiting') }}
          </p>
        </div>
      </li>
    </ol>

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
import { useSubscriptionStore } from '@/stores/subscriptions'
import { guideStepDone, markGuideStep } from '@/utils/getStarted'
import { track } from '@/utils/analytics'

const props = defineProps<{
  userId: number
  balance: number
  apiKeys: number
  requests: number
}>()

const { t } = useI18n()
const subscriptions = useSubscriptionStore()

const POLL_MS = 10_000
const POLL_FOR_MS = 15 * 60_000

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

const hasKey = computed(() => keys.value > 0)
const hasCall = computed(() => requests.value > 0)
const allDone = computed(() => hasKey.value && hasCall.value)
const visible = computed(() => !dismissed.value && startedEmpty)
const needsCredit = computed(() => props.balance <= 0 && !subscriptions.hasActiveSubscriptions)

const steps = computed(() => {
  const copiedDone = copied.value || hasCall.value
  return [
    { key: 'key', done: hasKey.value, current: !hasKey.value, to: '/keys?action=create' },
    { key: 'config', done: copiedDone, current: hasKey.value && !copiedDone, to: '/keys?action=use' },
    { key: 'call', done: hasCall.value, current: hasKey.value && copiedDone && !hasCall.value, to: '' }
  ]
})

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
  }
})

onUnmounted(() => {
  stopPolling()
  window.removeEventListener('storage', onStorage)
})
</script>
