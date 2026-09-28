<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="card p-4 sm:p-6">
          <div class="flex flex-wrap items-end justify-between gap-4">
            <div class="flex flex-1 flex-wrap items-end gap-4">
              <div class="w-full sm:w-auto sm:min-w-[280px]">
                <label class="input-label">{{ t('admin.requestLogs.filters.q') }}</label>
                <div class="relative">
                  <Icon
                    name="search"
                    size="md"
                    class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                  />
                  <input
                    v-model.trim="filters.q"
                    type="text"
                    class="input pl-10"
                    data-testid="request-logs-q"
                    :placeholder="t('admin.requestLogs.filters.qPlaceholder')"
                    @keyup.enter="search"
                  />
                </div>
              </div>

              <div class="w-full sm:w-auto sm:min-w-[140px]">
                <label class="input-label">{{ t('admin.requestLogs.filters.result') }}</label>
                <Select v-model="filters.success" :options="resultOptions" @change="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[120px]">
                <label class="input-label">{{ t('admin.requestLogs.filters.statusCode') }}</label>
                <input
                  v-model.trim="filters.status_code"
                  type="text"
                  inputmode="numeric"
                  class="input"
                  placeholder="401"
                  @keyup.enter="search"
                />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[120px]">
                <label class="input-label">{{ t('admin.requestLogs.filters.method') }}</label>
                <Select v-model="filters.method" :options="methodOptions" @change="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[160px]">
                <label class="input-label">{{ t('admin.requestLogs.filters.timeRange') }}</label>
                <Select v-model="filters.range" :options="rangeOptions" @change="search" />
              </div>
            </div>

            <div class="flex w-full flex-wrap items-center justify-end gap-3 sm:w-auto">
              <button type="button" class="btn btn-primary" :disabled="loading" @click="search">
                {{ t('common.search') }}
              </button>
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="resetFilters">
                {{ t('common.reset') }}
              </button>
            </div>
          </div>
          <div class="mt-4 flex flex-wrap items-center justify-between gap-2 text-xs">
            <span class="text-amber-600 dark:text-amber-400">
              <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" />
              {{ t('admin.requestLogs.fullKeyNotice') }}
            </span>
            <span class="text-gray-500 dark:text-gray-400">{{ t('admin.requestLogs.total', { count: total }) }}</span>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="logs" :loading="loading" row-key="id">
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatTime(value) }}</span>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0 max-w-[200px]">
              <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.user_email">
                {{ row.user_email || '—' }}
              </div>
              <div v-if="row.user_id" class="mt-0.5 text-xs text-gray-400">#{{ row.user_id }}</div>
            </div>
          </template>

          <template #cell-api_key="{ row }">
            <div class="min-w-0 max-w-[300px]">
              <div class="flex items-center gap-1.5">
                <span
                  v-if="row.api_key"
                  class="break-all font-mono text-xs text-gray-800 dark:text-gray-200"
                  data-testid="request-log-full-key"
                  >{{ row.api_key }}</span
                >
                <span v-else class="text-xs text-gray-400">{{ t('admin.requestLogs.noKey') }}</span>
                <button
                  v-if="row.api_key"
                  type="button"
                  class="flex-shrink-0 text-gray-400 transition-colors hover:text-primary-600 dark:hover:text-primary-400"
                  :title="t('admin.requestLogs.copy')"
                  @click="copy(row.api_key)"
                >
                  <Icon name="copy" size="sm" />
                </button>
              </div>
              <div class="mt-0.5 truncate text-xs text-gray-400">
                {{ row.api_key_name || (row.api_key ? t('admin.requestLogs.unknownKey') : '') }}
              </div>
            </div>
          </template>

          <template #cell-request="{ row }">
            <div class="min-w-0 max-w-md">
              <div class="flex items-start gap-1.5">
                <span class="flex-shrink-0 rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ row.method }}
                </span>
                <span class="break-all font-mono text-xs text-gray-700 dark:text-gray-300" :title="row.url">{{ row.url }}</span>
              </div>
            </div>
          </template>

          <template #cell-status_code="{ row }">
            <div class="whitespace-nowrap">
              <span :class="statusBadgeClass(row.success)">
                <span class="h-1.5 w-1.5 rounded-full" :class="row.success ? 'bg-emerald-500' : 'bg-red-500'"></span>
                {{ row.status_code }}
                {{ row.success ? t('admin.requestLogs.filters.success') : t('admin.requestLogs.filters.failure') }}
              </span>
              <div v-if="row.error_code" class="mt-1 font-mono text-[11px] text-red-500 dark:text-red-400">
                {{ row.error_code }}
              </div>
            </div>
          </template>

          <template #cell-model="{ value }">
            <span class="whitespace-nowrap font-mono text-xs text-gray-700 dark:text-gray-300">{{ value || '—' }}</span>
          </template>

          <template #cell-duration_ms="{ value }">
            <span class="whitespace-nowrap text-gray-500 dark:text-gray-400">{{ formatDuration(value) }}</span>
          </template>

          <template #cell-client_ip="{ value }">
            <span class="whitespace-nowrap font-mono text-xs text-gray-600 dark:text-gray-300">{{ value || '—' }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="inline-flex items-center gap-1 font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              @click="detail = row"
            >
              <Icon name="eye" size="sm" />
              {{ t('admin.requestLogs.columns.detail') }}
            </button>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-8">
              <Icon name="document" size="xl" class="mb-4 h-12 w-12 text-gray-300 dark:text-dark-600" />
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.requestLogs.empty') }}</p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="!!detail"
      :title="t('admin.requestLogs.detail.title')"
      width="wide"
      :close-on-click-outside="true"
      @close="detail = null"
    >
      <dl v-if="detail" class="grid grid-cols-1 gap-x-6 gap-y-4 text-sm sm:grid-cols-[140px_1fr]">
        <template v-for="field in detailFields" :key="field.label">
          <dt class="font-medium text-gray-500 dark:text-gray-400">{{ field.label }}</dt>
          <dd class="flex items-start gap-2">
            <span :class="['break-all text-gray-900 dark:text-gray-100', field.mono ? 'font-mono text-xs' : '']">
              {{ field.value || '—' }}
            </span>
            <button
              v-if="field.copyable && field.value"
              type="button"
              class="flex-shrink-0 text-gray-400 transition-colors hover:text-primary-600 dark:hover:text-primary-400"
              :title="t('admin.requestLogs.copy')"
              @click="copy(String(field.value))"
            >
              <Icon name="copy" size="sm" />
            </button>
          </dd>
        </template>
      </dl>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type GatewayRequestLog } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const logs = ref<GatewayRequestLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const detail = ref<GatewayRequestLog | null>(null)

const defaultFilters = () => ({ q: '', success: '', status_code: '', method: '', range: '24h' })
const filters = reactive(defaultFilters())

const RANGE_MINUTES: Record<string, number> = { '1h': 60, '24h': 24 * 60, '7d': 7 * 24 * 60, '30d': 30 * 24 * 60 }

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.requestLogs.columns.time') },
  { key: 'user', label: t('admin.requestLogs.columns.user') },
  { key: 'api_key', label: t('admin.requestLogs.columns.apiKey') },
  { key: 'request', label: t('admin.requestLogs.columns.request') },
  { key: 'status_code', label: t('admin.requestLogs.columns.result') },
  { key: 'model', label: t('admin.requestLogs.columns.model') },
  { key: 'duration_ms', label: t('admin.requestLogs.columns.duration') },
  { key: 'client_ip', label: t('admin.requestLogs.columns.clientIp') },
  { key: 'actions', label: t('common.actions') }
])

const resultOptions = computed(() => [
  { value: '', label: t('admin.requestLogs.filters.all') },
  { value: 'true', label: t('admin.requestLogs.filters.success') },
  { value: 'false', label: t('admin.requestLogs.filters.failure') }
])

const methodOptions = computed(() => [
  { value: '', label: t('admin.requestLogs.filters.all') },
  ...['POST', 'GET', 'PUT', 'DELETE'].map((m) => ({ value: m, label: m }))
])

const rangeOptions = computed(() => [
  { value: '1h', label: t('admin.requestLogs.filters.last1h') },
  { value: '24h', label: t('admin.requestLogs.filters.last24h') },
  { value: '7d', label: t('admin.requestLogs.filters.last7d') },
  { value: '30d', label: t('admin.requestLogs.filters.last30d') },
  { value: '', label: t('admin.requestLogs.filters.all') }
])

const detailFields = computed(() => {
  const d = detail.value
  if (!d) return []
  return [
    { label: t('admin.requestLogs.columns.time'), value: formatTime(d.created_at) },
    { label: t('admin.requestLogs.detail.url'), value: `${d.method} ${d.url}`, mono: true, copyable: true },
    { label: t('admin.requestLogs.detail.apiKey'), value: d.api_key, mono: true, copyable: true },
    { label: t('admin.requestLogs.detail.keyName'), value: d.api_key_name },
    { label: t('admin.requestLogs.detail.user'), value: d.user_email ? `${d.user_email} (#${d.user_id})` : '' },
    { label: t('admin.requestLogs.detail.status'), value: String(d.status_code) },
    { label: t('admin.requestLogs.detail.errorCode'), value: d.error_code, mono: true },
    { label: t('admin.requestLogs.detail.model'), value: d.model, mono: true },
    { label: t('admin.requestLogs.detail.duration'), value: formatDuration(d.duration_ms) },
    { label: t('admin.requestLogs.detail.clientIp'), value: d.client_ip, mono: true },
    { label: t('admin.requestLogs.detail.userAgent'), value: d.user_agent, mono: true },
    { label: t('admin.requestLogs.detail.requestId'), value: d.request_id, mono: true, copyable: true },
    { label: t('admin.requestLogs.detail.groupId'), value: d.group_id ? String(d.group_id) : '' },
    { label: t('admin.requestLogs.detail.accountId'), value: d.account_id ? String(d.account_id) : '' }
  ]
})

function formatTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function formatDuration(ms: number): string {
  if (!ms && ms !== 0) return '—'
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)} s` : `${ms} ms`
}

function statusBadgeClass(success: boolean): string {
  const base = 'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold '
  return (
    base +
    (success
      ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
      : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300')
  )
}

function copy(text: string) {
  void copyToClipboard(text, t('admin.requestLogs.copied'))
}

function buildQuery() {
  const minutes = RANGE_MINUTES[filters.range]
  const code = Number.parseInt(filters.status_code, 10)
  return {
    page: page.value,
    page_size: pageSize.value,
    q: filters.q || undefined,
    success: filters.success || undefined,
    method: filters.method || undefined,
    status_code: Number.isFinite(code) && code >= 100 && code <= 599 ? code : undefined,
    start_time: minutes ? new Date(Date.now() - minutes * 60 * 1000).toISOString() : undefined
  }
}

async function fetchLogs() {
  loading.value = true
  try {
    const res = await adminAPI.requestLogs.list(buildQuery())
    logs.value = res.items
    total.value = res.total
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.requestLogs.loadFailed'))
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  void fetchLogs()
}

function resetFilters() {
  Object.assign(filters, defaultFilters())
  search()
}

function onPageChange(p: number) {
  page.value = p
  void fetchLogs()
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  void fetchLogs()
}

onMounted(fetchLogs)
</script>
