<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex gap-2">
        <button v-for="name in tabs" :key="name" :class="['btn btn-sm', tab === name ? 'btn-primary' : 'btn-secondary']" @click="switchTab(name)">
          {{ t(`admin.sites.tabs.${name}`) }}<span v-if="name === 'reports' && openReports" class="ml-1">({{ openReports }})</span><span v-if="name === 'reviews' && pendingReviews" class="ml-1">({{ pendingReviews }})</span>
        </button>
      </div>

      <template v-if="tab === 'sites'">
        <div class="card p-4">
          <div class="flex flex-wrap items-center gap-3">
            <input v-model="filters.q" class="input w-72" :placeholder="t('admin.sites.search')" @keyup.enter="reloadSites" />
            <Select v-model="filters.status" :options="statusOptions" class="w-40" @change="reloadSites" />
            <button class="btn btn-secondary ml-auto" :title="t('common.refresh')" @click="loadSites"><Icon name="refresh" size="md" /></button>
          </div>
        </div>
        <div class="card overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.site') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.owner') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.status') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.size') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.billing') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.columns.updated') }}</th>
                <th class="px-4 py-3 font-medium"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="site in sites" :key="site.id">
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ site.title || site.name }}</div>
                  <a :href="site.url" target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 hover:underline">{{ site.url }}</a>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">{{ site.user_email || site.user_id }}</td>
                <td class="px-4 py-3">
                  <span :class="['badge', site.status === 'active' ? 'badge-success' : site.status === 'disabled' ? 'badge-danger' : 'badge-warning']">{{ t(`admin.sites.status.${site.status}`) }}</span>
                  <div v-if="site.status_reason" class="mt-1 max-w-xs text-xs text-gray-500">{{ site.status_reason }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatBytes(site.size_bytes, 1) }} · {{ site.file_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ site.paid ? t('admin.sites.paidUntil', { date: formatDateOnly(site.paid_until) }) : t('admin.sites.free') }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatDateTime(site.updated_at) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <button v-if="site.status !== 'disabled'" class="btn btn-warning btn-sm" @click="openDisable(site)">{{ t('admin.sites.actions.disable') }}</button>
                  <button v-else class="btn btn-secondary btn-sm" @click="enable(site)">{{ t('admin.sites.actions.enable') }}</button>
                  <button class="btn btn-danger btn-sm ml-2" @click="confirmDelete = site">{{ t('admin.sites.actions.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!sites.length">
                <td colspan="7" class="px-4 py-10 text-center text-gray-500">{{ t('admin.sites.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="sitesTotal > pageSize" :page="sitesPage" :total="sitesTotal" :page-size="pageSize" @update:page="(p: number) => { sitesPage = p; loadSites() }" />
      </template>

      <template v-else-if="tab === 'reviews'">
        <div v-for="item in reviewItems" :key="`${item.site_id}-${item.version}`" class="card space-y-3 p-4" data-testid="review-item">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">{{ item.title || item.site_name }} · {{ t('admin.sites.reviews.version', { version: item.version }) }}</div>
              <div class="text-xs text-gray-500">{{ item.owner_email }} · {{ formatDateTime(item.created_at) }} · {{ formatBytes(item.size_bytes, 1) }} · {{ t(`admin.sites.status.${item.site_status}`) }}</div>
              <div v-if="item.reason" class="mt-1 text-sm text-amber-700 dark:text-amber-300">{{ item.reason }}</div>
              <div v-if="item.flags.length" class="mt-1 flex flex-wrap gap-1">
                <span v-for="flag in item.flags" :key="flag" class="badge badge-gray">{{ flag }}</span>
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <a :href="item.preview_url" target="_blank" rel="noopener noreferrer" class="btn btn-secondary btn-sm">{{ t('admin.sites.reviews.preview') }}</a>
              <button class="btn btn-success btn-sm" @click="approve(item)">{{ t('admin.sites.reviews.approve') }}</button>
              <button class="btn btn-danger btn-sm" @click="openReject(item)">{{ t('admin.sites.reviews.reject') }}</button>
            </div>
          </div>
          <pre class="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-300">{{ item.excerpt || t('admin.sites.reviews.noText') }}</pre>
        </div>
        <div v-if="!reviewItems.length" class="card p-10 text-center text-sm text-gray-500">{{ t('admin.sites.reviews.empty') }}</div>
        <Pagination v-if="reviewsTotal > pageSize" :page="reviewsPage" :total="reviewsTotal" :page-size="pageSize" @update:page="(p: number) => { reviewsPage = p; loadReviews() }" />
      </template>

      <template v-else-if="tab === 'reports'">
        <div class="card p-4">
          <Select v-model="reportStatus" :options="reportStatusOptions" class="w-40" @change="reloadReports" />
        </div>
        <div class="card overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.reports.time') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.reports.site') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.reports.reason') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('admin.sites.reports.contact') }}</th>
                <th class="px-4 py-3 font-medium"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="report in reports" :key="report.id">
                <td class="whitespace-nowrap px-4 py-3 text-gray-500">{{ formatDateTime(report.created_at) }}</td>
                <td class="px-4 py-3">
                  <a v-if="report.site_status" :href="siteURL(report.site_name)" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline">{{ report.site_name }}</a>
                  <span v-else class="text-gray-500">{{ report.site_name }}（{{ t('admin.sites.reports.gone') }}）</span>
                  <div class="text-xs text-gray-500">{{ report.owner_email }} <span v-if="report.site_status">· {{ t(`admin.sites.status.${report.site_status}`) }}</span></div>
                </td>
                <td class="px-4 py-3">
                  <div class="text-gray-900 dark:text-white">{{ t(`siteReport.reasons.${report.reason}`) }}</div>
                  <div class="max-w-md whitespace-pre-wrap text-xs text-gray-500">{{ report.detail }}</div>
                </td>
                <td class="px-4 py-3 text-xs text-gray-500">{{ report.contact }}<div>{{ report.reporter_ip }}</div></td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <button v-if="report.site_id && report.site_status && report.site_status !== 'disabled'" class="btn btn-warning btn-sm" @click="disableFromReport(report)">{{ t('admin.sites.actions.disable') }}</button>
                  <button v-if="report.status === 'open'" class="btn btn-secondary btn-sm ml-2" @click="setReport(report, 'resolved')">{{ t('admin.sites.reports.resolve') }}</button>
                  <button v-if="report.status === 'open'" class="btn btn-secondary btn-sm ml-2" @click="setReport(report, 'dismissed')">{{ t('admin.sites.reports.dismiss') }}</button>
                  <span v-else class="text-xs text-gray-500">{{ t(`admin.sites.reports.statuses.${report.status}`) }}</span>
                </td>
              </tr>
              <tr v-if="!reports.length">
                <td colspan="5" class="px-4 py-10 text-center text-gray-500">{{ t('admin.sites.reports.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="reportsTotal > pageSize" :page="reportsPage" :total="reportsTotal" :page-size="pageSize" @update:page="(p: number) => { reportsPage = p; loadReports() }" />
      </template>

      <div v-else-if="settings" class="card space-y-4 p-5">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.sites.settings.hint', { domain: settings.domain || '-' }) }}</p>
        <p v-if="!settings.domain" class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">{{ t('admin.sites.settings.noDomain') }}</p>
        <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
          <Toggle v-model="settings.enabled" />
          {{ t('admin.sites.settings.enabled') }}
        </label>
        <div class="grid gap-4 md:grid-cols-4">
          <div v-for="field in numberFields" :key="field.key">
            <label class="input-label">{{ t(`admin.sites.settings.${field.key}`) }}</label>
            <input v-model.number="settings[field.key]" type="number" :min="field.min" :max="field.max" :step="field.step" class="input" />
          </div>
        </div>
        <div class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700">
          <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.sites.settings.reviewTitle') }}</div>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.sites.settings.reviewHint') }}</p>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="settings.review_all" />
            {{ t('admin.sites.settings.reviewAll') }}
          </label>
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.sites.settings.reviewBaseUrl') }}</label>
              <input v-model="settings.review_base_url" class="input" placeholder="http://127.0.0.1:8080/v1" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.sites.settings.reviewModel') }}</label>
              <input v-model="settings.review_model" class="input" placeholder="gpt-5-mini" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.sites.settings.reviewApiKey') }}</label>
              <input v-model="reviewKey" type="password" autocomplete="new-password" class="input" :placeholder="settings.review_api_key_configured ? t('admin.sites.settings.reviewKeyKeep') : t('admin.sites.settings.reviewKeyNone')" />
            </div>
          </div>
        </div>
        <button class="btn btn-primary" :disabled="saving" @click="saveSettings">{{ t('common.save') }}</button>
      </div>
    </div>

    <BaseDialog :show="!!rejecting" :title="t('admin.sites.reviews.reject')" width="narrow" @close="rejecting = null">
      <input v-model="rejectReason" class="input" maxlength="200" :placeholder="t('admin.sites.reviews.rejectPlaceholder')" />
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="rejecting = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" @click="reject">{{ t('admin.sites.reviews.reject') }}</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!disabling" :title="t('admin.sites.actions.disable')" width="narrow" @close="disabling = null">
      <div class="space-y-2">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.sites.disableHint') }}</p>
        <input v-model="disableReason" class="input" maxlength="300" :placeholder="t('admin.sites.disablePlaceholder')" />
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="disabling = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" @click="disable">{{ t('admin.sites.actions.disable') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!confirmDelete"
      :title="t('admin.sites.actions.delete')"
      :message="t('admin.sites.deleteConfirm', { url: confirmDelete?.url || '' })"
      danger
      @confirm="remove"
      @cancel="confirmDelete = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateOnly, formatDateTime } from '@/utils/format'
import type { Site } from '@/api/sites'
import type { SiteHostingSettings, SiteReport, SiteReviewItem } from '@/api/admin/sites'

type Tab = 'sites' | 'reviews' | 'reports' | 'settings'
type NumberKey = 'max_per_user' | 'max_mb' | 'max_files' | 'free_per_user' | 'extra_price' | 'grace_days' | 'retention_days'

const { t } = useI18n()
const appStore = useAppStore()
const tabs: Tab[] = ['sites', 'reviews', 'reports', 'settings']
const tab = ref<Tab>('sites')
const pageSize = 20

const sites = ref<Site[]>([])
const sitesTotal = ref(0)
const sitesPage = ref(1)
const filters = reactive({ q: '', status: '' })
const reports = ref<SiteReport[]>([])
const reportsTotal = ref(0)
const reportsPage = ref(1)
const reportStatus = ref('open')
const openReports = ref(0)
const settings = ref<SiteHostingSettings | null>(null)
const saving = ref(false)
const disabling = ref<Site | null>(null)
const disableReason = ref('')
const confirmDelete = ref<Site | null>(null)
const reviewItems = ref<SiteReviewItem[]>([])
const reviewsTotal = ref(0)
const reviewsPage = ref(1)
const pendingReviews = ref(0)
const rejecting = ref<SiteReviewItem | null>(null)
const rejectReason = ref('')
const reviewKey = ref('')

const numberFields: { key: NumberKey; min: number; max: number; step: number }[] = [
  { key: 'max_per_user', min: 1, max: 50, step: 1 },
  { key: 'free_per_user', min: 0, max: 50, step: 1 },
  { key: 'extra_price', min: 0, max: 1000, step: 0.5 },
  { key: 'max_mb', min: 1, max: 200, step: 1 },
  { key: 'max_files', min: 1, max: 5000, step: 1 },
  { key: 'grace_days', min: 0, max: 365, step: 1 },
  { key: 'retention_days', min: 0, max: 365, step: 1 }
]
const statusOptions = computed(() => [{ value: '', label: t('admin.sites.allStatuses') }, ...['active', 'pending', 'disabled', 'unpaid', 'lapsed'].map((value) => ({ value, label: t(`admin.sites.status.${value}`) }))])
const reportStatusOptions = computed(() => ['open', 'resolved', 'dismissed', ''].map((value) => ({ value, label: value ? t(`admin.sites.reports.statuses.${value}`) : t('admin.sites.allStatuses') })))

function showError(error: unknown) {
  appStore.showError(extractApiErrorMessage(error, t('common.error')))
}

function siteURL(name: string) {
  return settings.value?.domain ? `https://${name}.${settings.value.domain}` : '#'
}

async function loadSites() {
  try {
    const res = await adminAPI.sites.list({ ...filters, page: sitesPage.value, page_size: pageSize })
    sites.value = res.items
    sitesTotal.value = res.total
  } catch (error) {
    showError(error)
  }
}
function reloadSites() {
  sitesPage.value = 1
  void loadSites()
}

async function loadReports() {
  try {
    const res = await adminAPI.sites.reports({ status: reportStatus.value, page: reportsPage.value, page_size: pageSize })
    reports.value = res.items
    reportsTotal.value = res.total
    if (reportStatus.value === 'open') openReports.value = res.total
  } catch (error) {
    showError(error)
  }
}
function reloadReports() {
  reportsPage.value = 1
  void loadReports()
}

function switchTab(name: Tab) {
  tab.value = name
  if (name === 'sites') void loadSites()
  if (name === 'reports') void loadReports()
  if (name === 'reviews') void loadReviews()
}

async function loadReviews() {
  try {
    const res = await adminAPI.sites.reviews({ page: reviewsPage.value, page_size: pageSize })
    reviewItems.value = res.items
    reviewsTotal.value = pendingReviews.value = res.total
  } catch (error) {
    showError(error)
  }
}

async function approve(item: SiteReviewItem) {
  try {
    await adminAPI.sites.review(item.site_id, item.version, 'approve')
    appStore.showSuccess(t('admin.sites.reviews.approved'))
    await loadReviews()
  } catch (error) {
    showError(error)
  }
}

function openReject(item: SiteReviewItem) {
  rejecting.value = item
  rejectReason.value = ''
}

async function reject() {
  const item = rejecting.value
  rejecting.value = null
  if (!item) return
  try {
    await adminAPI.sites.review(item.site_id, item.version, 'reject', rejectReason.value)
    appStore.showSuccess(t('admin.sites.reviews.rejected'))
    await loadReviews()
  } catch (error) {
    showError(error)
  }
}

function openDisable(site: Site) {
  disabling.value = site
  disableReason.value = ''
}

async function disable() {
  const site = disabling.value
  disabling.value = null
  if (!site) return
  try {
    await adminAPI.sites.setStatus(site.id, 'disabled', disableReason.value)
    appStore.showSuccess(t('admin.sites.disabled'))
    await loadSites()
  } catch (error) {
    showError(error)
  }
}

async function enable(site: Site) {
  try {
    await adminAPI.sites.setStatus(site.id, 'active')
    appStore.showSuccess(t('admin.sites.enabled'))
    await loadSites()
  } catch (error) {
    showError(error)
  }
}

async function remove() {
  const site = confirmDelete.value
  confirmDelete.value = null
  if (!site) return
  try {
    await adminAPI.sites.remove(site.id)
    await loadSites()
  } catch (error) {
    showError(error)
  }
}

async function disableFromReport(report: SiteReport) {
  if (!report.site_id) return
  try {
    await adminAPI.sites.setStatus(report.site_id, 'disabled', t(`siteReport.reasons.${report.reason}`))
    await adminAPI.sites.setReportStatus(report.id, 'resolved')
    appStore.showSuccess(t('admin.sites.disabled'))
    await loadReports()
  } catch (error) {
    showError(error)
  }
}

async function setReport(report: SiteReport, status: SiteReport['status']) {
  try {
    await adminAPI.sites.setReportStatus(report.id, status)
    await loadReports()
  } catch (error) {
    showError(error)
  }
}

async function saveSettings() {
  if (!settings.value) return
  saving.value = true
  try {
    settings.value = await adminAPI.sites.saveSettings({ ...settings.value, review_api_key: reviewKey.value || undefined })
    reviewKey.value = ''
    appStore.showSuccess(t('admin.sites.settings.saved'))
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    settings.value = await adminAPI.sites.getSettings()
  } catch (error) {
    showError(error)
  }
  await Promise.all([loadSites(), (async () => {
    try {
      openReports.value = (await adminAPI.sites.reports({ status: 'open', page: 1, page_size: 1 })).total
      pendingReviews.value = (await adminAPI.sites.reviews({ page: 1, page_size: 1 })).total
    } catch {
      openReports.value = 0
    }
  })()])
})
</script>
