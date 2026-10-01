<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageToolUses.today') }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
            <template v-if="quota?.subscribed">{{ t('imageToolUses.todayValue', { left: quota.free_left, daily: quota.free_daily }) }}</template>
            <span v-else class="text-sm font-normal text-gray-500">{{ t('imageToolUses.noSubscription', { daily: quota?.free_daily ?? 0 }) }}</span>
          </div>
        </div>
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageToolUses.prices') }}</div>
          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('imageToolUses.priceValue', { removeBg: quota?.prices.remove_bg ?? '-', upscale: quota?.prices.upscale ?? '-' }) }}
          </div>
        </div>
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageToolUses.monthRuns') }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ t('imageToolUses.runsValue', { runs: month.runs, free: month.free_runs }) }}</div>
        </div>
        <div class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageToolUses.monthCost') }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">¥{{ formatCost(month.cost) }}</div>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="tool" :options="toolOptions" class="w-40" @change="reload" />
          <div class="flex flex-1 items-center justify-end gap-2">
            <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <a class="btn btn-primary" :href="canvasToolsUrl" target="_blank" rel="noopener">{{ t('imageToolUses.openCanvas') }}</a>
          </div>
        </div>
      </div>

      <div class="card overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.time') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.tool') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.charge') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.size') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.duration') }}</th>
              <th class="px-4 py-3 font-medium">{{ t('imageToolUses.columns.key') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="row in items" :key="row.id" data-testid="image-tool-use-row">
              <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(row.created_at) }}</td>
              <td class="px-4 py-3 text-gray-900 dark:text-white">{{ t(`imageToolUses.tools.${row.tool}`) }}</td>
              <td class="px-4 py-3">
                <span v-if="row.free" class="badge badge-success">{{ t('imageToolUses.free') }}</span>
                <span v-else class="font-medium text-gray-900 dark:text-white">¥{{ formatCost(row.cost) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatBytes(row.input_bytes, 1) }} → {{ formatBytes(row.output_bytes, 1) }}</td>
              <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ (row.duration_ms / 1000).toFixed(1) }}s</td>
              <td class="px-4 py-3 text-gray-500">{{ row.api_key_name || '-' }}</td>
            </tr>
            <tr v-if="!loading && !items.length">
              <td colspan="6" class="px-4 py-10 text-center text-sm text-gray-500">{{ t('imageToolUses.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <Pagination v-if="total > pageSize" :page="page" :total="total" :page-size="pageSize" @update:page="onPage" @update:page-size="onPageSize" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'
import { canvasUrl } from '@/constants/crossSites'
import { IMAGE_TOOLS, myUses, sumImageToolStats, type ImageToolsQuota, type ImageToolStat, type ImageToolUse } from '@/api/imageTools'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<ImageToolUse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const tool = ref('')
const loading = ref(false)
const quota = ref<ImageToolsQuota | null>(null)
const monthStats = ref<ImageToolStat[]>([])

const month = computed(() => sumImageToolStats(monthStats.value))
const toolOptions = computed(() => [{ value: '', label: t('imageToolUses.allTools') }, ...IMAGE_TOOLS.map((value) => ({ value, label: t(`imageToolUses.tools.${value}`) }))])
const canvasToolsUrl = canvasUrl({ medium: 'image-tool-uses', path: '/tools' })

function formatCost(value: number) {
  return Number(value.toFixed(4)).toString()
}

async function load() {
  loading.value = true
  try {
    const res = await myUses({ page: page.value, page_size: pageSize.value, tool: tool.value || undefined })
    items.value = res.items
    total.value = res.total
    quota.value = res.summary.quota
    monthStats.value = res.summary.month
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  void load()
}
function onPage(value: number) {
  page.value = value
  void load()
}
function onPageSize(value: number) {
  pageSize.value = value
  reload()
}

onMounted(load)
</script>
