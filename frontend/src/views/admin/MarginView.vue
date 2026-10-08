<template>
  <AppLayout>
    <div class="space-y-4" data-testid="margin-view">
      <div class="card flex flex-wrap items-end justify-between gap-3 p-4">
        <div class="max-w-3xl space-y-1">
          <p v-if="data" class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.margin.since', { since: sinceText, days: data.days }) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.margin.basis') }}</p>
        </div>
        <div class="flex flex-wrap items-end gap-2">
          <form class="flex items-end gap-2" @submit.prevent="saveCost">
            <div>
              <label class="input-label" for="margin-cost">{{ t('admin.margin.costLabel') }}</label>
              <input id="margin-cost" v-model.number="cost" type="number" min="0" max="10" step="0.01" placeholder="0.3" class="input w-28" />
            </div>
            <button type="submit" class="btn btn-secondary btn-sm" :disabled="saving" data-testid="margin-save-cost">{{ t('admin.margin.saveCost') }}</button>
          </form>
          <button
            v-for="n in ranges"
            :key="n"
            type="button"
            class="btn btn-sm"
            :class="days === n ? 'btn-primary' : 'btn-secondary'"
            @click="setDays(n)"
          >
            {{ t('admin.analytics.range', { n }) }}
          </button>
        </div>
      </div>

      <p v-if="data && !data.configured" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200" data-testid="margin-unconfigured">
        {{ t('admin.margin.unconfigured') }}
      </p>

      <template v-if="data">
        <!-- Totals -->
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4" data-testid="margin-totals">
          <div class="card p-4">
            <p class="text-xs text-gray-500">{{ t('admin.margin.kpi.revenue') }}</p>
            <p class="text-xl font-semibold text-gray-900 dark:text-white">¥{{ money(data.totals.revenue) }}</p>
            <p class="text-xs text-gray-400">{{ t('admin.margin.kpi.revenueSplit', { recharge: money(data.totals.recharge_revenue), sub: money(data.totals.subscription_revenue) }) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs text-gray-500">{{ t('admin.margin.kpi.cost') }}</p>
            <p class="text-xl font-semibold text-gray-900 dark:text-white">¥{{ money(data.totals.member_cost) }}</p>
            <p class="text-xs text-gray-400">{{ t('admin.margin.kpi.costSplit', { paygo: money(data.totals.paygo_cost), sub: money(data.totals.subscription_cost) }) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs text-gray-500">{{ t('admin.margin.kpi.margin') }}</p>
            <p class="text-xl font-semibold" :class="signClass(data.totals.margin)">{{ yuan(data.totals.margin) }}</p>
            <p class="text-xs text-gray-400">{{ t('admin.margin.kpi.marginHint') }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs text-gray-500">{{ t('admin.margin.kpi.admin') }}</p>
            <p class="text-xl font-semibold text-gray-900 dark:text-white">¥{{ money(data.totals.admin_cost) }}</p>
            <p class="text-xs text-gray-400">{{ t('admin.margin.kpi.adminHint') }}</p>
          </div>
        </div>

        <!-- Plans -->
        <div class="card p-4" data-testid="margin-plans">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.margin.plans') }}</h3>
          <p class="mb-2 text-xs text-gray-500">{{ t('admin.margin.plansHint') }}</p>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[820px] text-sm">
              <thead>
                <tr class="whitespace-nowrap text-left text-xs text-gray-500">
                  <th class="py-1">{{ t('admin.margin.col.plan') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.price') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.cap') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.maxCost') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.breakEven') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.maxLoss') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.sold') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.revenue') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in data.plans" :key="p.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-1.5">
                    <div class="font-medium text-gray-900 dark:text-white">{{ p.name }}</div>
                    <div class="text-xs text-gray-500">{{ p.group_name }} · {{ t('admin.margin.daysN', { n: p.days }) }}</div>
                  </td>
                  <td class="px-2 text-right tabular-nums">¥{{ money(p.price) }}</td>
                  <td class="px-2 text-right tabular-nums">{{ p.cap_usd == null ? t('admin.margin.noCap') : `$${money(p.cap_usd)}` }}</td>
                  <td class="px-2 text-right tabular-nums">{{ p.max_cost == null ? '—' : `¥${money(p.max_cost)}` }}</td>
                  <td class="px-2 text-right tabular-nums">{{ data.configured ? `$${money(p.break_even_usd)}` : '—' }}</td>
                  <td class="px-2 text-right tabular-nums" :class="p.max_loss > 0 ? 'font-semibold text-red-600 dark:text-red-400' : 'text-emerald-600'">
                    {{ data.configured ? (p.max_loss > 0 ? `−¥${money(p.max_loss)}` : t('admin.margin.cannotLose')) : '—' }}
                  </td>
                  <td class="px-2 text-right tabular-nums">{{ p.sold }}</td>
                  <td class="px-2 text-right tabular-nums">¥{{ money(p.revenue) }}</td>
                </tr>
                <tr v-if="!data.plans.length"><td colspan="8" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Subscriptions -->
        <div class="card p-4" data-testid="margin-subscriptions">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.margin.subscriptions') }}</h3>
            <div class="flex flex-wrap gap-2 text-xs">
              <span v-for="s in statusOrder" :key="s" class="rounded-full px-2 py-0.5" :class="statusClass(s)">{{ t(`admin.margin.status.${s}`) }} {{ statusCount[s] || 0 }}</span>
            </div>
          </div>
          <p class="mb-2 text-xs text-gray-500">{{ t('admin.margin.subscriptionsHint') }}</p>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[900px] text-sm">
              <thead>
                <tr class="whitespace-nowrap text-left text-xs text-gray-500">
                  <th class="py-1">{{ t('admin.margin.col.user') }}</th>
                  <th>{{ t('admin.margin.col.term') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.paid') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.projectedMargin') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.used') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.cost') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.margin') }}</th>
                  <th class="px-2 text-right">{{ t('admin.margin.col.projected') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in data.subscriptions" :key="s.id" class="border-t border-gray-100 dark:border-dark-700" :data-testid="`margin-sub-${s.id}`">
                  <td class="py-1.5">
                    <div class="font-medium text-gray-900 dark:text-white">
                      <span class="mr-1 whitespace-nowrap rounded-full px-2 py-0.5 text-xs" :class="statusClass(s.status)">{{ t(`admin.margin.status.${s.status}`) }}</span>
                      {{ s.email }} <span class="text-xs text-gray-400">#{{ s.user_id }}</span>
                    </div>
                    <div class="text-xs text-gray-500">{{ s.group_name }}</div>
                  </td>
                  <td class="whitespace-nowrap text-xs text-gray-500">
                    {{ day(s.starts_at) }} → {{ day(s.expires_at) }}
                    <div>{{ t(s.active ? 'admin.margin.elapsed' : 'admin.margin.ended', { n: fmtDays(s.days_elapsed), total: fmtDays(s.days_total) }) }}</div>
                  </td>
                  <td class="px-2 text-right tabular-nums">¥{{ money(s.paid) }}</td>
                  <td class="px-2 text-right tabular-nums" :class="signClass(s.projected_margin)">{{ yuan(s.projected_margin) }}</td>
                  <td class="px-2 text-right tabular-nums">${{ money(s.usage_usd) }}</td>
                  <td class="px-2 text-right tabular-nums">¥{{ money(s.cost) }}</td>
                  <td class="px-2 text-right tabular-nums" :class="signClass(s.margin)">{{ yuan(s.margin) }}</td>
                  <td class="px-2 text-right tabular-nums">{{ s.active ? `$${money(s.projected_usd)}` : '—' }}</td>
                </tr>
                <tr v-if="!data.subscriptions.length"><td colspan="8" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="grid gap-4 xl:grid-cols-2">
          <!-- Groups -->
          <div class="card p-4" data-testid="margin-groups">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.margin.groups') }}</h3>
            <p class="mb-2 text-xs text-gray-500">{{ t('admin.margin.groupsHint') }}</p>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="whitespace-nowrap text-left text-xs text-gray-500">
                    <th class="py-1">{{ t('admin.margin.col.group') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.users') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.used') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.billed') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.cost') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.margin') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="g in data.groups" :key="g.group_id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1.5">
                      {{ g.name || t('admin.margin.noGroup') }}
                      <span class="ml-1 text-xs text-gray-400">{{ g.subscription ? t('admin.margin.typeSub') : t('admin.margin.typePaygo') }}</span>
                    </td>
                    <td class="px-2 text-right tabular-nums">{{ g.users }}</td>
                    <td class="px-2 text-right tabular-nums">${{ money(g.usage_usd) }}</td>
                    <td class="px-2 text-right tabular-nums">{{ g.subscription ? '—' : `¥${money(g.billed_usd)}` }}</td>
                    <td class="px-2 text-right tabular-nums">¥{{ money(g.cost) }}</td>
                    <td class="px-2 text-right tabular-nums" :class="g.margin == null ? '' : signClass(g.margin)">{{ g.margin == null ? t('admin.margin.seeSubs') : yuan(g.margin) }}</td>
                  </tr>
                  <tr v-if="!data.groups.length"><td colspan="6" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Users -->
          <div class="card p-4" data-testid="margin-users">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.margin.users') }}</h3>
            <p class="mb-2 text-xs text-gray-500">{{ t('admin.margin.usersHint') }}</p>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="whitespace-nowrap text-left text-xs text-gray-500">
                    <th class="py-1">{{ t('admin.margin.col.user') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.used') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.paidPeriod') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.cost') }}</th>
                    <th class="px-2 text-right">{{ t('admin.margin.col.margin') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="u in data.users" :key="u.user_id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1.5">{{ u.email }} <span class="text-xs text-gray-400">#{{ u.user_id }}</span></td>
                    <td class="px-2 text-right tabular-nums">${{ money(u.usage_usd) }}</td>
                    <td class="px-2 text-right tabular-nums">¥{{ money(u.paid) }}</td>
                    <td class="px-2 text-right tabular-nums">¥{{ money(u.cost) }}</td>
                    <td class="px-2 text-right tabular-nums" :class="signClass(u.margin)">{{ yuan(u.margin) }}</td>
                  </tr>
                  <tr v-if="!data.users.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </template>
      <div v-else-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getMargin, setMarginCost, type MarginReport, type MarginSubscription } from '@/api/admin/analytics'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const ranges = [7, 30, 90]
const days = ref(30)
const data = ref<MarginReport | null>(null)
const loading = ref(false)
const saving = ref(false)
const cost = ref<number | ''>('')

const statusOrder: MarginSubscription['status'][] = ['loss', 'risk', 'gift', 'ok']
const statusCount = computed(() => {
  const out: Record<string, number> = {}
  for (const s of data.value?.subscriptions || []) out[s.status] = (out[s.status] || 0) + 1
  return out
})
const sinceText = computed(() => (data.value ? day(data.value.since) : ''))

function money(v: number): string {
  return (Number(v) || 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
/** ¥ amount with a leading minus sign for losses: −¥240.00. */
function yuan(v: number): string {
  return v < 0 ? `−¥${money(-v)}` : `¥${money(v)}`
}
function fmtDays(v: number): string {
  return String(Math.round((Number(v) || 0) * 10) / 10)
}
function day(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function signClass(v: number): string {
  return v < 0 ? 'font-semibold text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
}
function statusClass(s: string): string {
  switch (s) {
    case 'loss':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'risk':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'gift':
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
    default:
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    data.value = await getMargin(days.value)
    cost.value = data.value.configured ? data.value.cost_per_usd : ''
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  } finally {
    loading.value = false
  }
}

function setDays(n: number): void {
  days.value = n
  void load()
}

async function saveCost(): Promise<void> {
  if (saving.value) return
  saving.value = true
  try {
    data.value = await setMarginCost(Number(cost.value) || 0, days.value)
    cost.value = data.value.configured ? data.value.cost_per_usd : ''
    appStore.showSuccess(t('admin.margin.costSaved'))
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
