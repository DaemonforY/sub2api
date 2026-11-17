<template>
  <div class="card" data-testid="quota-reset-settings">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('quotaReset.admin.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('quotaReset.admin.description') }}</p>
    </div>
    <div v-if="loading" class="flex justify-center p-6">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>
    <div v-else class="space-y-5 p-6">
      <div class="flex items-center justify-between">
        <label class="font-medium text-gray-900 dark:text-white">{{ t('quotaReset.admin.enabled') }}</label>
        <Toggle v-model="form.enabled" data-testid="quota-reset-enabled" />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('quotaReset.admin.pricePerUsd') }}</label>
          <input v-model.number="form.price_per_usd" type="number" min="0" step="0.01" class="input" data-testid="quota-reset-coef" />
          <p class="input-hint">{{ t('quotaReset.admin.pricePerUsdHint') }}</p>
          <p class="mt-1 text-xs text-primary-600 dark:text-primary-400">{{ t('quotaReset.admin.example', { price: examplePrice }) }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('quotaReset.admin.maxPerDay') }}</label>
          <input v-model.number="form.max_per_day" type="number" min="1" step="1" class="input" data-testid="quota-reset-max" />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('quotaReset.admin.platforms') }}</label>
        <div class="mt-1 flex flex-wrap gap-4">
          <label v-for="p in PLATFORMS" :key="p" class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input type="checkbox" class="h-4 w-4" :checked="form.platforms.includes(p)" @change="togglePlatform(p)" />
            {{ platformLabel(p) }}
          </label>
        </div>
        <p class="input-hint">{{ t('quotaReset.admin.platformsHint') }}</p>
      </div>
      <div class="flex items-center justify-between gap-4">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">{{ t('quotaReset.admin.wechatNotice') }}</label>
          <p class="input-hint">{{ t('quotaReset.admin.wechatNoticeHint') }}</p>
        </div>
        <Toggle v-model="form.wechat_notice" data-testid="quota-reset-wechat" />
      </div>
      <div class="flex justify-end">
        <button type="button" class="btn btn-primary" :disabled="saving" data-testid="quota-reset-save" @click="save">
          {{ t('quotaReset.admin.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { adminPaymentAPI, type QuotaResetSettings } from '@/api/admin/payment'
import { useAppStore } from '@/stores'
import { platformLabel } from '@/utils/platformColors'
import { extractI18nErrorMessage } from '@/utils/apiError'

const PLATFORMS = ['openai', 'anthropic', 'gemini', 'antigravity', 'grok']

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const form = reactive<QuotaResetSettings>({ enabled: false, price_per_usd: 0.5, max_per_day: 1, platforms: ['openai'], wechat_notice: true })

const examplePrice = computed(() => {
  const v = Math.round(40 * (Number(form.price_per_usd) || 0) * 100) / 100
  return Number.isInteger(v) ? String(v) : v.toFixed(2)
})

function togglePlatform(p: string) {
  form.platforms = form.platforms.includes(p) ? form.platforms.filter(x => x !== p) : [...form.platforms, p]
}

async function load() {
  loading.value = true
  try {
    const { data } = await adminPaymentAPI.getQuotaResetSettings()
    Object.assign(form, { ...data, platforms: data.platforms ?? [] })
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('quotaReset.admin.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const { data } = await adminPaymentAPI.updateQuotaResetSettings({
      enabled: form.enabled,
      price_per_usd: Number(form.price_per_usd) || 0,
      max_per_day: Math.max(1, Math.floor(Number(form.max_per_day) || 1)),
      platforms: [...form.platforms],
      wechat_notice: form.wechat_notice,
    })
    if (data) Object.assign(form, { ...data, platforms: data.platforms ?? [] })
    appStore.showSuccess(t('quotaReset.admin.saved'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('quotaReset.admin.saveFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
