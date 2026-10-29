<template>
  <div class="space-y-4">
    <div class="card p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="text-sm text-gray-600 dark:text-gray-300">
          {{ t('inspiration.public.page') }}
          <a href="/poems" target="_blank" rel="noopener" class="text-primary-600 hover:underline">/poems ↗</a>
        </div>
        <div v-if="stats" class="flex flex-wrap gap-4 text-xs text-gray-500" data-testid="public-stats">
          <span>{{ t('inspiration.public.runsToday', { n: stats.runs_today }) }}</span>
          <span>{{ t('inspiration.public.paidToday', { n: stats.paid_today, usd: stats.revenue_today.toFixed(2) }) }}</span>
          <span>{{ t('inspiration.public.paid7d', { n: stats.paid_7d, usd: stats.revenue_7d.toFixed(2) }) }}</span>
        </div>
      </div>
    </div>

    <div v-if="form" class="card space-y-5 p-5">
      <div class="flex flex-wrap gap-6 text-sm text-gray-700 dark:text-gray-300">
        <label class="flex items-center gap-2"><Toggle v-model="form.enabled" /> {{ t('inspiration.public.enabled') }}</label>
        <label class="flex items-center gap-2"><Toggle v-model="form.find_enabled" /> {{ t('inspiration.public.findEnabled') }}</label>
      </div>

      <div>
        <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.public.findTitle') }}</h3>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <label class="text-xs text-gray-500">{{ t('inspiration.public.free') }}<input v-model.number="form.free_finds_per_day" type="number" min="0" class="input mt-1" data-testid="public-free" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.price') }}<input v-model.number="form.find_price" type="number" min="0" step="0.01" class="input mt-1" data-testid="public-price" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.cap') }}<input v-model.number="form.daily_find_cap" type="number" min="0" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.count') }}<input v-model.number="form.find_count" type="number" min="1" max="12" class="input mt-1" /></label>
          <label class="text-xs text-gray-500 sm:col-span-2">{{ t('inspiration.public.model') }}<input v-model="form.find_model" class="input mt-1" :placeholder="t('inspiration.public.modelHint')" /></label>
        </div>
        <p class="mt-2 text-xs text-gray-400">{{ t('inspiration.public.costHint') }}</p>
      </div>

      <div>
        <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.public.limitsTitle') }}</h3>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <label class="text-xs text-gray-500">{{ t('inspiration.public.guestMinute') }}<input v-model.number="form.guest_per_minute" type="number" min="1" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.guestDay') }}<input v-model.number="form.guest_per_day" type="number" min="1" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.userMinute') }}<input v-model.number="form.user_per_minute" type="number" min="1" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.public.userDay') }}<input v-model.number="form.user_per_day" type="number" min="1" class="input mt-1" /></label>
        </div>
        <p class="mt-2 text-xs text-gray-400">{{ t('inspiration.public.limitsHint') }}</p>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" :disabled="saving" data-testid="public-save" @click="save">{{ t('inspiration.editor.save') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'

const { t } = useI18n()
const appStore = useAppStore()
const form = ref<api.LibPublicSettings | null>(null)
const stats = ref<api.LibPublicStats | null>(null)
const saving = ref(false)

async function load() {
  try {
    const res = await api.getPublicSettings()
    form.value = res.settings
    stats.value = res.stats
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function save() {
  if (!form.value) return
  saving.value = true
  try {
    form.value = (await api.savePublicSettings(form.value)).settings
    appStore.showSuccess(t('inspiration.editor.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
