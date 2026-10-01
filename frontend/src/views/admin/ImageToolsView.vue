<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-5">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.imageTools.settings.title') }}</h3>
            <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-dark-400">{{ t('admin.imageTools.settings.hint') }}</p>
          </div>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            {{ t('admin.imageTools.settings.enabled') }}
            <Toggle v-model="form.enabled" />
          </label>
        </div>
        <p v-if="settings && !settings.service_configured" class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
          {{ t('admin.imageTools.settings.notConfigured') }}
        </p>
        <div class="mt-4 grid gap-4 md:grid-cols-4">
          <div>
            <label class="input-label">{{ t('admin.imageTools.settings.priceRemoveBg') }}</label>
            <input v-model.number="form.price_remove_bg" type="number" min="0" max="100" step="0.01" class="input" data-testid="price-remove-bg" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.imageTools.settings.priceUpscale') }}</label>
            <input v-model.number="form.price_upscale" type="number" min="0" max="100" step="0.01" class="input" data-testid="price-upscale" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.imageTools.settings.freeDaily') }}</label>
            <input v-model.number="form.free_daily" type="number" min="0" max="1000" step="1" class="input" data-testid="free-daily" />
          </div>
          <div class="flex items-end">
            <button class="btn btn-primary w-full" :disabled="saving || !settings" @click="save">{{ t('admin.imageTools.settings.save') }}</button>
          </div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div v-for="period in periods" :key="period" class="card p-4">
          <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.imageTools.stats.${period}`) }}</div>
          <div class="mt-3 space-y-2 text-sm">
            <div v-for="toolName in IMAGE_TOOLS" :key="toolName" class="flex items-center justify-between gap-2">
              <span class="text-gray-500 dark:text-dark-400">{{ t(`admin.imageTools.tools.${toolName}`) }}</span>
              <span class="text-gray-900 dark:text-white">
                {{ statOf(period, toolName).runs }} {{ t('admin.imageTools.stats.runs') }} · {{ t('admin.imageTools.stats.free') }} {{ statOf(period, toolName).free_runs }} ·
                ¥{{ formatCost(statOf(period, toolName).cost) }} · {{ statOf(period, toolName).users }} {{ t('admin.imageTools.stats.users') }}
              </span>
            </div>
            <div class="flex items-center justify-between border-t border-gray-100 pt-2 font-medium dark:border-dark-700">
              <span class="text-gray-700 dark:text-gray-300">{{ t('admin.imageTools.stats.revenue') }}</span>
              <span class="text-gray-900 dark:text-white">¥{{ formatCost(sumImageToolStats(stats?.[period]).cost) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex flex-wrap items-end gap-3">
          <input v-model="filters.q" class="input w-64" :placeholder="t('admin.imageTools.filters.search')" @keyup.enter="reload" />
          <Select v-model="filters.tool" :options="toolOptions" class="w-40" @change="reload" />
          <label class="text-xs text-gray-500">
            {{ t('admin.imageTools.filters.from') }}
            <input v-model="filters.start_date" type="date" class="input mt-1" @change="reload" />
          </label>
          <label class="text-xs text-gray-500">
            {{ t('admin.imageTools.filters.to') }}
            <input v-model="filters.end_date" type="date" class="input mt-1" @change="reload" />
          </label>
          <button class="btn btn-secondary ml-auto" :disabled="loading" :title="t('common.refresh')" @click="refreshAll">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <div class="card overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.time') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.user') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.tool') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.charge') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.size') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.duration') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('admin.imageTools.columns.key') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="row in items" :key="row.id">
              <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(row.created_at) }}</td>
              <td class="px-4 py-3">
                <button class="text-primary-600 hover:underline" :title="String(row.user_id)" @click="filterUser(row.user_email || '')">{{ row.user_email || row.user_id }}</button>
              </td>
              <td class="px-4 py-3 text-gray-900 dark:text-white">{{ t(`admin.imageTools.tools.${row.tool}`) }}</td>
              <td class="px-4 py-3">
                <span v-if="row.free" class="badge badge-success">{{ t('admin.imageTools.free') }}</span>
                <span v-else class="font-medium text-gray-900 dark:text-white">¥{{ formatCost(row.cost) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatBytes(row.input_bytes, 1) }} → {{ formatBytes(row.output_bytes, 1) }}</td>
              <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ (row.duration_ms / 1000).toFixed(1) }}s</td>
              <td class="px-4 py-3 text-gray-500">{{ row.api_key_name || '-' }}</td>
            </tr>
            <tr v-if="!loading && !items.length">
              <td colspan="7" class="px-4 py-10 text-center text-sm text-gray-500">{{ t('admin.imageTools.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <Pagination v-if="total > pageSize" :page="page" :total="total" :page-size="pageSize" @update:page="onPage" @update:page-size="onPageSize" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'
import { IMAGE_TOOLS, imageToolStat, sumImageToolStats, type ImageTool, type ImageToolUse } from '@/api/imageTools'
import type { ImageToolsSettings, ImageToolsStats } from '@/api/admin/imageTools'

const { t } = useI18n()
const appStore = useAppStore()
const periods = ['today', 'week', 'month'] as const

const settings = ref<ImageToolsSettings | null>(null)
const form = reactive({ enabled: true, price_remove_bg: 0, price_upscale: 0, free_daily: 0 })
const saving = ref(false)
const stats = ref<ImageToolsStats | null>(null)
const items = ref<ImageToolUse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const filters = reactive({ q: '', tool: '', start_date: '', end_date: '' })

const toolOptions = computed(() => [{ value: '', label: t('admin.imageTools.filters.allTools') }, ...IMAGE_TOOLS.map((value) => ({ value, label: t(`admin.imageTools.tools.${value}`) }))])

function statOf(period: (typeof periods)[number], tool: ImageTool) {
  return imageToolStat(stats.value?.[period], tool)
}
function formatCost(value: number) {
  return Number(value.toFixed(4)).toString()
}
function showError(error: unknown) {
  appStore.showError(extractApiErrorMessage(error, t('common.error')))
}

function applySettings(value: ImageToolsSettings) {
  settings.value = value
  Object.assign(form, { enabled: value.enabled, price_remove_bg: value.price_remove_bg, price_upscale: value.price_upscale, free_daily: value.free_daily })
}

async function save() {
  saving.value = true
  try {
    applySettings(await adminAPI.imageTools.saveSettings({ ...form, price_remove_bg: Number(form.price_remove_bg) || 0, price_upscale: Number(form.price_upscale) || 0, free_daily: Math.round(Number(form.free_daily) || 0) }))
    appStore.showSuccess(t('admin.imageTools.settings.saved'))
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

async function loadUses() {
  loading.value = true
  try {
    const res = await adminAPI.imageTools.uses({ ...filters, page: page.value, page_size: pageSize.value })
    items.value = res.items
    total.value = res.total
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function refreshAll() {
  try {
    stats.value = await adminAPI.imageTools.stats()
  } catch (error) {
    showError(error)
  }
  await loadUses()
}

function reload() {
  page.value = 1
  void loadUses()
}
function filterUser(email: string) {
  filters.q = email
  reload()
}
function onPage(value: number) {
  page.value = value
  void loadUses()
}
function onPageSize(value: number) {
  pageSize.value = value
  reload()
}

onMounted(async () => {
  try {
    applySettings(await adminAPI.imageTools.getSettings())
  } catch (error) {
    showError(error)
  }
  await refreshAll()
})
</script>
