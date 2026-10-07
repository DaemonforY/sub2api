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
import type { AssistantKeyOption, AssistantSettings } from '@/api/admin/assistant'

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

onMounted(load)
</script>
