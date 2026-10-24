<template>
  <AppLayout>
    <div class="space-y-4" data-testid="prompt-market-view">
      <div class="card space-y-1 p-4">
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('promptMarket.admin.description') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('promptMarket.admin.hint') }}</p>
      </div>

      <!-- Settings -->
      <form class="card flex flex-wrap items-end gap-4 p-4" data-testid="prompt-market-config" @submit.prevent="saveConfig">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="config.enabled" type="checkbox" class="h-4 w-4" data-testid="prompt-market-enabled" />
          {{ t('promptMarket.admin.enabled') }}
        </label>
        <div>
          <label class="input-label">{{ t('promptMarket.admin.fee') }}</label>
          <input v-model.number="feePercent" type="number" min="0" max="90" step="1" class="input w-28" required />
        </div>
        <div>
          <label class="input-label">{{ t('promptMarket.admin.minPrice') }}</label>
          <input v-model.number="config.min_price" type="number" min="0.01" step="0.01" class="input w-28" required />
        </div>
        <div>
          <label class="input-label">{{ t('promptMarket.admin.maxPrice') }}</label>
          <input v-model.number="config.max_price" type="number" min="0.01" max="1000" step="0.01" class="input w-28" required />
        </div>
        <button type="submit" class="btn btn-primary btn-sm" :disabled="savingConfig">{{ t('promptMarket.admin.save') }}</button>
      </form>

      <!-- Tabs -->
      <div class="flex flex-wrap gap-2">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          class="rounded-full border px-3 py-1 text-sm"
          :class="active === tab.value ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300'"
          :data-testid="`prompt-market-tab-${tab.value}`"
          @click="switchTab(tab.value)"
        >
          {{ tab.label }}<span v-if="tab.value === 'review' && reviewCount" class="ml-1 text-xs">{{ reviewCount }}</span>
        </button>
      </div>

      <!-- Listings -->
      <div v-if="active !== 'purchases'" class="space-y-3">
        <div v-if="!listings.length" class="card p-8 text-center text-sm text-gray-500">{{ loading ? t('common.loading') : t('promptMarket.admin.empty') }}</div>
        <div v-for="l in listings" :key="l.id" class="card space-y-3 p-4" data-testid="prompt-market-listing">
          <div class="flex flex-wrap items-start gap-3">
            <img v-if="l.cover_url" :src="l.cover_url" alt="" class="h-20 w-20 rounded-lg object-cover" />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="font-semibold text-gray-900 dark:text-white">{{ l.title }}</h3>
                <span class="rounded-full px-2 py-0.5 text-xs" :class="statusClass(l.status)">{{ t(`promptMarket.status.${l.status}`) }}</span>
                <span v-if="l.draft" class="rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">{{ t('promptMarket.admin.hasDraft') }}</span>
                <span v-for="flag in l.flags || []" :key="flag" class="rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-700 dark:bg-red-900/40 dark:text-red-300">{{ flag }}</span>
              </div>
              <p class="mt-1 text-xs text-gray-500">
                #{{ l.id }} · {{ l.seller_email ? displayEmail(l.seller_email, t, l.seller_name) : l.seller_name }} · ${{ l.price.toFixed(2) }} · {{ t('promptMarket.admin.items', { count: l.item_count }) }} ·
                {{ t('promptMarket.admin.sales', { count: l.sales_count }) }} · {{ dateText(l.updated_at) }}
              </p>
              <p v-if="l.description" class="mt-1 whitespace-pre-wrap text-sm text-gray-600 dark:text-dark-300">{{ l.description }}</p>
              <p v-if="l.review_note" class="mt-1 text-xs text-red-600">{{ t('promptMarket.admin.note') }}：{{ l.review_note }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button v-if="l.status === 'pending' || l.draft" type="button" class="btn btn-primary btn-sm" data-testid="prompt-market-approve" @click="review(l, 'approve')">{{ t('promptMarket.admin.approve') }}</button>
              <button v-if="l.status === 'pending' || l.draft" type="button" class="btn btn-secondary btn-sm" @click="review(l, 'reject')">{{ t('promptMarket.admin.reject') }}</button>
              <button v-if="l.status !== 'hidden'" type="button" class="btn btn-secondary btn-sm" @click="review(l, 'hide')">{{ t('promptMarket.admin.hide') }}</button>
              <button v-if="l.status === 'hidden' && l.published_at" type="button" class="btn btn-secondary btn-sm" @click="review(l, 'restore')">{{ t('promptMarket.admin.restore') }}</button>
            </div>
          </div>

          <div v-if="l.sample_urls.length" class="flex flex-wrap gap-2">
            <img v-for="url in l.sample_urls" :key="url" :src="url" alt="" class="h-16 w-16 rounded object-cover" />
          </div>

          <details class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800" :open="l.status === 'pending'">
            <summary class="cursor-pointer text-gray-700 dark:text-gray-200">{{ t('promptMarket.admin.content', { count: l.items.length }) }}</summary>
            <ol class="mt-2 space-y-2">
              <li v-for="item in l.items" :key="item.position">
                <div class="font-medium text-gray-800 dark:text-gray-100">{{ item.position + 1 }}. {{ item.title }}</div>
                <div class="whitespace-pre-wrap text-xs leading-5 text-gray-600 dark:text-dark-300">{{ item.prompt }}</div>
              </li>
            </ol>
          </details>

          <details v-if="l.draft" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm dark:border-amber-900 dark:bg-amber-950/30" open>
            <summary class="cursor-pointer text-amber-800 dark:text-amber-300">
              {{ t('promptMarket.admin.draft', { title: l.draft.title, price: l.draft.price.toFixed(2), count: l.draft.items.length }) }}
            </summary>
            <p v-if="l.draft.description" class="mt-2 whitespace-pre-wrap text-xs text-gray-600 dark:text-dark-300">{{ l.draft.description }}</p>
            <ol class="mt-2 space-y-2">
              <li v-for="item in l.draft.items" :key="item.position">
                <div class="font-medium text-gray-800 dark:text-gray-100">{{ item.position + 1 }}. {{ item.title }}</div>
                <div class="whitespace-pre-wrap text-xs leading-5 text-gray-600 dark:text-dark-300">{{ item.prompt }}</div>
              </li>
            </ol>
          </details>
        </div>
        <div v-if="total > listings.length" class="text-center">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadListings(page + 1)">{{ t('promptMarket.admin.more') }}</button>
        </div>
      </div>

      <!-- Purchases -->
      <div v-else class="card overflow-x-auto p-4" data-testid="prompt-market-purchases">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs text-gray-500">
              <th class="py-2 pr-3">{{ t('promptMarket.admin.time') }}</th>
              <th class="py-2 pr-3">{{ t('promptMarket.admin.listing') }}</th>
              <th class="py-2 pr-3">{{ t('promptMarket.admin.buyer') }}</th>
              <th class="py-2 pr-3">{{ t('promptMarket.admin.seller') }}</th>
              <th class="py-2 pr-3">{{ t('promptMarket.admin.price') }}</th>
              <th class="py-2 pr-3">{{ t('promptMarket.admin.sellerAmount') }}</th>
              <th class="py-2 pr-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in purchases" :key="p.id" class="border-t border-gray-100 dark:border-dark-700">
              <td class="py-2 pr-3 text-gray-500">{{ dateText(p.created_at) }}</td>
              <td class="py-2 pr-3 text-gray-900 dark:text-white">#{{ p.listing_id }} {{ p.listing_title }}</td>
              <td class="py-2 pr-3">{{ p.buyer_email ? displayEmail(p.buyer_email, t) : p.buyer_user_id }}</td>
              <td class="py-2 pr-3">{{ p.seller_email ? displayEmail(p.seller_email, t) : p.seller_user_id }}</td>
              <td class="py-2 pr-3">${{ p.price.toFixed(2) }}</td>
              <td class="py-2 pr-3">${{ p.seller_amount.toFixed(4) }}</td>
              <td class="py-2 pr-3 text-right">
                <span v-if="p.status === 'refunded'" class="text-xs text-gray-400">{{ t('promptMarket.admin.refunded') }}</span>
                <button v-else type="button" class="btn btn-secondary btn-sm" @click="refund(p)">{{ t('promptMarket.admin.refund') }}</button>
              </td>
            </tr>
            <tr v-if="!purchases.length">
              <td colspan="7" class="py-8 text-center text-gray-500">{{ t('promptMarket.admin.noPurchases') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { displayEmail } from '@/utils/userDisplay'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  getPromptMarketConfig,
  listPromptListings,
  listPromptPurchases,
  refundPromptPurchase,
  reviewPromptListing,
  savePromptMarketConfig,
  type PromptListing,
  type PromptMarketConfig,
  type PromptPurchase
} from '@/api/promptMarket'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

type Tab = 'review' | 'active' | 'all' | 'purchases'
const tabs = computed<{ value: Tab; label: string }[]>(() => [
  { value: 'review', label: t('promptMarket.admin.tabReview') },
  { value: 'active', label: t('promptMarket.admin.tabActive') },
  { value: 'all', label: t('promptMarket.admin.tabAll') },
  { value: 'purchases', label: t('promptMarket.admin.tabPurchases') }
])

const config = ref<PromptMarketConfig>({ enabled: false, fee_rate: 0.2, min_price: 0.1, max_price: 50 })
const feePercent = ref(20)
const savingConfig = ref(false)
const active = ref<Tab>('review')
const listings = ref<PromptListing[]>([])
const purchases = ref<PromptPurchase[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const reviewCount = ref(0)

function dateText(value: string): string {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}

function statusClass(status: string): string {
  switch (status) {
    case 'active':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
    case 'rejected':
    case 'hidden':
      return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

function showError(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

async function loadConfig() {
  try {
    config.value = await getPromptMarketConfig()
    feePercent.value = Math.round(config.value.fee_rate * 100)
  } catch (err) {
    showError(err)
  }
}

async function saveConfig() {
  savingConfig.value = true
  try {
    config.value = await savePromptMarketConfig({ ...config.value, fee_rate: (Number(feePercent.value) || 0) / 100 })
    feePercent.value = Math.round(config.value.fee_rate * 100)
    appStore.showSuccess(t('promptMarket.admin.saved'))
  } catch (err) {
    showError(err)
  } finally {
    savingConfig.value = false
  }
}

async function loadListings(next = 1) {
  loading.value = true
  try {
    const status = active.value === 'all' ? '' : active.value
    const res = await listPromptListings({ status, page: next, page_size: 20 })
    listings.value = next === 1 ? res.items : [...listings.value, ...res.items]
    total.value = res.total
    page.value = next
    if (active.value === 'review') reviewCount.value = res.total
  } catch (err) {
    showError(err)
  } finally {
    loading.value = false
  }
}

async function loadPurchases() {
  loading.value = true
  try {
    purchases.value = (await listPromptPurchases({ page: 1, page_size: 100 })).items
  } catch (err) {
    showError(err)
  } finally {
    loading.value = false
  }
}

function switchTab(tab: Tab) {
  active.value = tab
  if (tab === 'purchases') void loadPurchases()
  else void loadListings(1)
}

async function review(l: PromptListing, action: 'approve' | 'reject' | 'hide' | 'restore') {
  let note = ''
  if (action === 'reject' || action === 'hide') {
    const input = window.prompt(t(action === 'reject' ? 'promptMarket.admin.rejectPrompt' : 'promptMarket.admin.hidePrompt'))
    if (input === null) return
    note = input.trim()
  }
  try {
    await reviewPromptListing(l.id, action, note)
    appStore.showSuccess(t(`promptMarket.admin.done.${action}`))
    await loadListings(1)
    if (active.value !== 'review') void listPromptListings({ status: 'review', page_size: 1 }).then((res) => (reviewCount.value = res.total))
  } catch (err) {
    showError(err)
  }
}

async function refund(p: PromptPurchase) {
  if (!window.confirm(t('promptMarket.admin.refundConfirm', { price: p.price.toFixed(2), amount: p.seller_amount.toFixed(4) }))) return
  try {
    await refundPromptPurchase(p.id)
    appStore.showSuccess(t('promptMarket.admin.refundDone'))
    await loadPurchases()
  } catch (err) {
    showError(err)
  }
}

onMounted(() => {
  void loadConfig()
  void loadListings(1)
})
</script>
