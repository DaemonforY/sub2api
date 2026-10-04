<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card space-y-4 p-5" data-testid="learn-settings">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.learn.runTitle') }}</h3>
          <a href="/learn/" target="_blank" class="text-sm text-primary-600 hover:underline">{{ t('admin.learn.openSite') }} ↗</a>
        </div>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.learn.runHint') }}</p>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.run_enabled" type="checkbox" class="h-4 w-4" data-testid="learn-enabled" />
          {{ t('admin.learn.enabled') }}
        </label>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.apiKey') }}</span>
            <input v-model="apiKey" type="password" autocomplete="new-password" class="input" :placeholder="form.api_key_set ? t('admin.learn.apiKeySet') : t('admin.learn.apiKeyPlaceholder')" data-testid="learn-key" />
            <span class="input-hint">{{ t('admin.learn.apiKeyHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.model') }}</span>
            <input v-model="form.model" class="input" placeholder="gpt-5.5" data-testid="learn-model" />
            <span class="input-hint">{{ t('admin.learn.modelHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.freeRuns') }}</span>
            <input v-model.number="form.free_runs_per_day" type="number" min="0" max="500" class="input" data-testid="learn-free" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.dailyCap') }}</span>
            <input v-model.number="form.daily_cap" type="number" min="0" class="input" />
            <span class="input-hint">{{ t('admin.learn.dailyCapHint') }}</span>
          </label>
        </div>
        <button class="btn btn-primary" :disabled="saving" data-testid="learn-save" @click="save">{{ t('common.save') }}</button>
      </div>

      <div class="grid gap-3 sm:grid-cols-3 lg:grid-cols-6" data-testid="learn-stats">
        <div v-for="item in statItems" :key="item.key" class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.learn.stats.${item.key}`) }}</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
        </div>
      </div>

      <div class="card overflow-x-auto">
        <table class="w-full whitespace-nowrap text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2">{{ t('admin.learn.lesson') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.completed') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.runs') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="l in stats?.lessons || []" :key="l.lesson_id">
              <td class="px-4 py-2"><a :href="`/learn/${l.lesson_id[0]}/${l.lesson_id}.html`" target="_blank" class="font-mono text-primary-600 hover:underline">{{ l.lesson_id.toUpperCase() }}</a></td>
              <td class="px-4 py-2">{{ l.completed }}</td>
              <td class="px-4 py-2">{{ l.runs }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="!stats?.lessons?.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.learn.noData') }}</p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { LearnSettings, LearnStats } from '@/api/admin/learn'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const form = reactive<LearnSettings>({ run_enabled: false, model: '', free_runs_per_day: 20, daily_cap: 1000, api_key_set: false })
const apiKey = ref('')
const saving = ref(false)
const stats = ref<LearnStats | null>(null)

const statItems = computed(() => {
  const s = stats.value
  return [
    { key: 'learners', value: s?.learners ?? 0 },
    { key: 'learnersToday', value: s?.learners_today ?? 0 },
    { key: 'runsToday', value: s?.runs_today ?? 0 },
    { key: 'runs7d', value: s?.runs_7d ?? 0 },
    { key: 'failed7d', value: s?.failed_runs_7d ?? 0 },
    { key: 'tokens7d', value: (s?.tokens_7d ?? 0).toLocaleString() },
  ]
})

async function save() {
  saving.value = true
  try {
    const payload: LearnSettings = { ...form }
    if (apiKey.value.trim()) payload.api_key = apiKey.value.trim()
    Object.assign(form, await adminAPI.learn.saveSettings(payload))
    apiKey.value = ''
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    Object.assign(form, await adminAPI.learn.getSettings())
    stats.value = await adminAPI.learn.stats()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
})
</script>
