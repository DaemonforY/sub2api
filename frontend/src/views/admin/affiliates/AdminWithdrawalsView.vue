<template>
  <AppLayout>
    <div class="space-y-6" data-testid="admin-withdrawals">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.withdrawals.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.withdrawals.description') }}</p>
        <p v-if="!settingsEnabled" class="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('growth.admin.settings.withdrawTitle') }}：
          <router-link to="/admin/growth" class="underline">{{ t('growth.admin.settings.withdrawEnabled') }}</router-link>
        </p>
      </div>

      <div class="card p-6">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex flex-wrap gap-2">
            <button
              v-for="tab in tabs"
              :key="tab"
              type="button"
              :class="['btn btn-sm', status === tab ? 'btn-primary' : 'btn-secondary']"
              @click="setStatus(tab)"
            >
              {{ t(`growth.admin.withdrawals.tabs.${tab}`) }}
              <span v-if="tab === 'pending' && pendingCount > 0" class="ml-1 rounded-full bg-red-500 px-1.5 text-xs text-white">{{ pendingCount }}</span>
            </button>
          </div>
          <input v-model="search" type="text" class="input w-full lg:w-72" :placeholder="t('growth.admin.withdrawals.search')" @input="debouncedLoad" />
        </div>

        <div v-if="loading" class="flex justify-center py-10">
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <div v-else-if="items.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          {{ t('growth.admin.withdrawals.empty') }}
        </div>
        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[880px] text-left text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.withdrawals.columns.user') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.withdrawals.columns.amount') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.withdrawals.columns.payee') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.withdrawals.columns.note') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.withdrawals.columns.time') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.withdrawals.columns.status') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.withdrawals.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="w in items" :key="w.id" class="border-b border-gray-100 align-top last:border-b-0 dark:border-dark-800">
                <td class="px-3 py-2.5">
                  <div class="text-gray-900 dark:text-white">{{ w.user_email }}</div>
                  <div class="text-xs text-gray-400">#{{ w.user_id }} {{ w.username }}</div>
                </td>
                <td class="px-3 py-2.5 text-right">
                  <div class="font-semibold text-gray-900 dark:text-white">¥{{ w.cny_amount.toFixed(2) }}</div>
                  <div class="text-xs text-gray-400">{{ t('growth.admin.withdrawals.quota', { quota: formatCurrency(w.quota_amount) }) }}</div>
                </td>
                <td class="px-3 py-2.5">
                  <div class="text-gray-900 dark:text-white">{{ t(`growth.withdraw.methods.${w.method}`) }} · {{ w.real_name }}</div>
                  <div class="flex items-center gap-1 font-mono text-xs text-gray-600 dark:text-gray-300">
                    {{ w.account }}
                    <button type="button" class="text-primary-600 hover:underline dark:text-primary-400" :title="t('growth.admin.withdrawals.copy')" @click="copyToClipboard(w.account, t('growth.admin.withdrawals.copied'))">
                      <Icon name="copy" size="xs" />
                    </button>
                  </div>
                </td>
                <td class="max-w-[200px] px-3 py-2.5 text-xs text-gray-500 dark:text-dark-400">{{ w.user_note || '-' }}</td>
                <td class="px-3 py-2.5 text-gray-500 dark:text-dark-400">{{ formatDateTime(w.created_at) }}</td>
                <td class="px-3 py-2.5">
                  <span :class="['rounded px-1.5 py-0.5 text-xs', statusClass(w.status)]">{{ t(`growth.withdraw.status.${w.status}`) }}</span>
                  <div v-if="w.admin_note" class="mt-1 max-w-[180px] text-xs text-gray-500 dark:text-dark-400">{{ w.admin_note }}</div>
                  <div v-if="w.reviewed_at" class="mt-0.5 text-xs text-gray-400">{{ t('growth.admin.withdrawals.reviewedAt', { time: formatDateTime(w.reviewed_at) }) }}</div>
                </td>
                <td class="px-3 py-2.5 text-right">
                  <div v-if="w.status === 'pending'" class="flex justify-end gap-2">
                    <button type="button" class="btn btn-primary btn-sm" :data-testid="`withdraw-paid-${w.id}`" @click="openReview(w, 'paid')">{{ t('growth.admin.withdrawals.markPaid') }}</button>
                    <button type="button" class="btn btn-secondary btn-sm" :data-testid="`withdraw-reject-${w.id}`" @click="openReview(w, 'reject')">{{ t('growth.admin.withdrawals.reject') }}</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="pages > 1" class="mt-4 flex justify-end gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="goPage(page - 1)">‹</button>
          <span class="px-2 text-sm text-gray-500">{{ page }} / {{ pages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="page >= pages" @click="goPage(page + 1)">›</button>
        </div>
      </div>

      <ConfirmDialog
        :show="review !== null"
        :title="review?.action === 'paid' ? t('growth.admin.withdrawals.paidTitle') : t('growth.admin.withdrawals.rejectTitle')"
        :message="reviewMessage"
        :danger="review?.action === 'reject'"
        @confirm="submitReview"
        @cancel="review = null"
      >
        <input
          v-model.trim="reviewNote"
          type="text"
          maxlength="500"
          class="input"
          data-testid="withdraw-review-note"
          :placeholder="review?.action === 'paid' ? t('growth.admin.withdrawals.paidNote') : t('growth.admin.withdrawals.rejectNote')"
        />
      </ConfirmDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { adminGrowthAPI, type AffiliateWithdrawal, type WithdrawStatusValue } from '@/api/growth'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

type Tab = WithdrawStatusValue | 'all'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const tabs: Tab[] = ['pending', 'paid', 'rejected', 'cancelled', 'all']
const status = ref<Tab>('pending')
const search = ref('')
const items = ref<AffiliateWithdrawal[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const pendingCount = ref(0)
const settingsEnabled = ref(true)
const review = ref<{ item: AffiliateWithdrawal; action: 'paid' | 'reject' } | null>(null)
const reviewNote = ref('')
const reviewing = ref(false)

const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const reviewMessage = computed(() => {
  const r = review.value
  if (!r) return ''
  const amount = r.item.cny_amount.toFixed(2)
  if (r.action === 'reject') return t('growth.admin.withdrawals.rejectBody', { amount })
  return t('growth.admin.withdrawals.paidBody', {
    amount,
    name: r.item.real_name,
    method: t(`growth.withdraw.methods.${r.item.method}`),
    account: r.item.account,
  })
})

function statusClass(s: WithdrawStatusValue): string {
  switch (s) {
    case 'paid':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'rejected':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const res = await adminGrowthAPI.listWithdrawals({
      status: status.value === 'all' ? '' : status.value,
      search: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize,
    })
    items.value = res.items
    total.value = res.total
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
  adminGrowthAPI.withdrawalPendingCount().then((n) => (pendingCount.value = n)).catch(() => {})
}

let timer: ReturnType<typeof setTimeout> | undefined
function debouncedLoad(): void {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    page.value = 1
    void load()
  }, 300)
}

function setStatus(tab: Tab): void {
  status.value = tab
  page.value = 1
  void load()
}

function goPage(p: number): void {
  page.value = p
  void load()
}

function openReview(item: AffiliateWithdrawal, action: 'paid' | 'reject'): void {
  reviewNote.value = ''
  review.value = { item, action }
}

async function submitReview(): Promise<void> {
  const r = review.value
  if (!r || reviewing.value) return
  if (r.action === 'reject' && !reviewNote.value) {
    appStore.showError(t('growth.admin.withdrawals.rejectNote'))
    return
  }
  reviewing.value = true
  try {
    if (r.action === 'paid') await adminGrowthAPI.markWithdrawalPaid(r.item.id, reviewNote.value)
    else await adminGrowthAPI.rejectWithdrawal(r.item.id, reviewNote.value)
    review.value = null
    appStore.showSuccess(t('growth.admin.withdrawals.done'))
    await load()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    reviewing.value = false
  }
}

onMounted(() => {
  void load()
  adminGrowthAPI.getSettings().then((s) => (settingsEnabled.value = !!s.withdraw_enabled)).catch(() => {})
})
</script>
