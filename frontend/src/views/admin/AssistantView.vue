<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card space-y-4 p-5" data-testid="assistant-settings">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.assistant.settingsTitle') }}</h3>
          <a href="/" target="_blank" class="text-sm text-primary-600 hover:underline">{{ t('admin.assistant.openHome') }} ↗</a>
        </div>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.assistant.hint') }}</p>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.enabled" type="checkbox" class="h-4 w-4" data-testid="assistant-enabled" />
          {{ t('admin.assistant.enabled') }}
        </label>
        <label class="flex items-start gap-2 text-sm">
          <input v-model="form.tools" type="checkbox" class="mt-0.5 h-4 w-4" data-testid="assistant-tools" />
          <span>
            {{ t('admin.assistant.tools') }}
            <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t('admin.assistant.toolsHint') }}</span>
          </span>
        </label>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm">
            <span class="input-label">{{ t('admin.assistant.key') }}</span>
            <select v-model.number="form.key_id" class="input" data-testid="assistant-key">
              <option :value="0">{{ t('admin.assistant.keyPlaceholder') }}</option>
              <option v-if="otherAdminKey" :value="form.key_id">{{ form.key_name || `#${form.key_id}` }}</option>
              <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
            </select>
            <span class="input-hint">
              {{ t('admin.assistant.keyHint') }}
              <a href="/keys" target="_blank" class="text-primary-600 hover:underline">{{ t('admin.assistant.keyCreate') }} ↗</a>
            </span>
            <span v-if="otherAdminKey" class="input-hint">{{ t('admin.assistant.keyOther', { id: form.key_id }) }}</span>
            <span v-if="!keys.length && !loading" class="input-hint text-amber-600">{{ t('admin.assistant.noKeys') }}</span>
            <span v-if="form.key_problem" class="input-hint text-red-600" data-testid="assistant-key-problem">{{ t('admin.assistant.keyProblem', { msg: form.key_problem }) }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.assistant.model') }}</span>
            <input v-model.trim="form.model" class="input" placeholder="gpt-5.6-terra" data-testid="assistant-model" />
            <span class="input-hint">{{ t('admin.assistant.modelHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.assistant.userPerDay') }}</span>
            <input v-model.number="form.user_per_day" type="number" min="0" max="500" class="input" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.assistant.guestPerDay') }}</span>
            <input v-model.number="form.guest_per_day" type="number" min="0" max="100" class="input" />
            <span class="input-hint">{{ t('admin.assistant.guestHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.assistant.dailyCap') }}</span>
            <input v-model.number="form.daily_cap" type="number" min="0" class="input" />
            <span class="input-hint">{{ t('admin.assistant.dailyCapHint') }}</span>
          </label>
        </div>
        <button class="btn btn-primary" :disabled="saving || loading" data-testid="assistant-save" @click="save">{{ t('common.save') }}</button>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.assistant.today') }}</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            {{ form.today }}<span v-if="form.daily_cap" class="text-sm font-normal text-gray-400"> / {{ form.daily_cap }}</span>
          </div>
        </div>
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.assistant.pages') }}</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ form.pages }}</div>
          <div class="mt-1 text-xs text-gray-400">{{ t('admin.assistant.pagesHint') }}</div>
        </div>
      </div>

      <div class="card p-5" data-testid="assistant-runs">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.assistant.runsTitle') }}</h3>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.assistant.runsHint') }}</p>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <button class="btn btn-secondary btn-sm" :disabled="runsPage <= 1 || runsLoading" @click="loadRuns(runsPage - 1)">‹</button>
            <span class="text-gray-500">{{ runsPage }} / {{ runsPages }}</span>
            <button class="btn btn-secondary btn-sm" :disabled="runsPage >= runsPages || runsLoading" @click="loadRuns(runsPage + 1)">›</button>
          </div>
        </div>
        <p v-if="!runs.length && !runsLoading" class="py-6 text-center text-sm text-gray-400">{{ t('admin.assistant.runsEmpty') }}</p>
        <ul class="divide-y divide-gray-100 dark:divide-dark-700">
          <li v-for="run in runs" :key="run.id" class="py-2.5">
            <button type="button" class="flex w-full items-start gap-3 text-left" @click="toggle(run.id)">
              <span class="w-28 flex-shrink-0 text-xs text-gray-400">{{ formatTime(run.created_at) }}</span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm text-gray-900 dark:text-white">{{ run.question }}</span>
                <span class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ run.user_email || t('admin.assistant.guest') }}</span>
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
              <div class="whitespace-pre-wrap rounded-lg border border-gray-100 p-2 text-sm text-gray-800 dark:border-dark-700 dark:text-dark-100">{{ run.answer || '—' }}</div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as assistantAPI from '@/api/admin/assistant'
import type { AgentRun, AssistantKeyOption, AssistantSettings } from '@/api/admin/assistant'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const form = reactive<AssistantSettings>({
  enabled: false,
  model: 'gpt-5.6-terra',
  key_id: 0,
  user_per_day: 20,
  guest_per_day: 5,
  daily_cap: 300,
  tools: false,
  today: 0,
  pages: 0
})
const keys = ref<AssistantKeyOption[]>([])
const loading = ref(true)
const saving = ref(false)

// The chosen key belongs to another admin: it isn't in this admin's list but stays selected.
const otherAdminKey = computed(
  () => form.key_id > 0 && !!form.key_owner && form.key_owner !== authStore.user?.id && !keys.value.some((k) => k.id === form.key_id)
)

async function load() {
  loading.value = true
  try {
    const [settings, options] = await Promise.all([assistantAPI.getSettings(), assistantAPI.listKeys()])
    Object.assign(form, settings)
    keys.value = options
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    Object.assign(form, await assistantAPI.saveSettings({ ...form }))
    appStore.showSuccess(t('admin.assistant.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

const runs = ref<AgentRun[]>([])
const runsTotal = ref(0)
const runsPage = ref(1)
const runsLoading = ref(false)
const expanded = ref<number | null>(null)
const runsPages = computed(() => Math.max(1, Math.ceil(runsTotal.value / 20)))

async function loadRuns(page = 1) {
  runsLoading.value = true
  try {
    const out = await assistantAPI.listRuns(page)
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
})
</script>
