<template>
  <AppLayout>
    <div class="space-y-4" data-testid="analytics-view">
      <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
        <div class="space-y-1">
          <p v-if="data" class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.analytics.since', { since: data.since, days: data.days }) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.analytics.startNote') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-for="n in ranges"
            :key="n"
            type="button"
            class="btn btn-sm"
            :class="days === n ? 'btn-primary' : 'btn-secondary'"
            :data-testid="`analytics-range-${n}`"
            @click="setDays(n)"
          >
            {{ t('admin.analytics.range', { n }) }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ t('admin.analytics.refresh') }}</button>
        </div>
      </div>

      <!-- Activation reminder email -->
      <div v-if="reminder" class="card space-y-2 p-4" data-testid="analytics-reminder">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.reminder.title') }}</h3>
          <div class="flex items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm" data-testid="analytics-reminder-preview" @click="showPreview = !showPreview">
              {{ showPreview ? t('admin.analytics.reminder.hidePreview') : t('admin.analytics.reminder.preview') }}
            </button>
            <button type="button" class="btn btn-sm" :class="reminder.enabled ? 'btn-secondary' : 'btn-primary'" :disabled="savingReminder" data-testid="analytics-reminder-toggle" @click="toggleReminder">
              {{ reminder.enabled ? t('admin.analytics.reminder.turnOff') : t('admin.analytics.reminder.turnOn') }}
            </button>
          </div>
        </div>
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.analytics.reminder.hint') }}</p>
        <p class="text-sm">
          <span :class="reminder.enabled ? 'text-emerald-600' : 'text-gray-500'">{{ reminder.enabled ? t('admin.analytics.reminder.on') : t('admin.analytics.reminder.off') }}</span>
          · {{ t('admin.analytics.reminder.counts', { sent: reminder.sent, due: reminder.due }) }}
        </p>
        <div v-if="showPreview" class="space-y-1">
          <p class="text-xs text-gray-500">{{ t('admin.analytics.reminder.subject') }}：{{ reminder.subject }}</p>
          <iframe :srcdoc="reminder.preview" sandbox="" class="h-[520px] w-full rounded-lg border border-gray-200 bg-white dark:border-dark-600" :title="reminder.subject"></iframe>
        </div>
      </div>

      <div v-if="error" class="card p-4 text-sm text-red-600">{{ t('admin.analytics.loadFailed') }}：{{ error }}</div>

      <template v-if="data">
        <!-- KPIs -->
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-7" data-testid="analytics-kpis">
          <div v-for="k in kpis" :key="k.key" class="card p-4">
            <div class="text-2xl font-bold text-primary-600 dark:text-primary-400">{{ k.value }}</div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.analytics.kpi.${k.key}`) }}</div>
            <div v-if="k.hint" class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.analytics.kpi.${k.hint}`) }}</div>
          </div>
        </div>

        <!-- Trend -->
        <div class="card p-4">
          <h3 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.trend') }}</h3>
          <div class="h-72"><Line :data="trendData" :options="lineOptions" /></div>
        </div>

        <div class="grid gap-4 lg:grid-cols-2">
          <!-- Funnel -->
          <div class="card p-4" data-testid="analytics-funnel">
            <h3 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.funnel') }}</h3>
            <div class="space-y-2">
              <div v-for="(s, i) in data.funnel" :key="s.step" class="flex items-center gap-3 text-sm">
                <span class="w-24 shrink-0 text-gray-600 dark:text-dark-300">{{ t(`admin.analytics.funnelSteps.${s.step}`) }}</span>
                <div class="h-6 flex-1 rounded bg-gray-100 dark:bg-dark-700">
                  <div class="flex h-6 items-center rounded bg-primary-500 px-2 text-xs font-semibold text-white" :style="{ width: barWidth(s.count, funnelTop) }">{{ s.count }}</div>
                </div>
                <span class="w-24 shrink-0 text-right text-xs text-gray-500">{{ i > 0 ? t('admin.analytics.fromPrev', { p: pct(s.count, data.funnel[i - 1].count) }) : '' }}</span>
              </div>
            </div>
          </div>

          <!-- Devices + retention -->
          <div class="card space-y-4 p-4">
            <div v-for="b in breakdowns" :key="b.key" :data-testid="`analytics-breakdown-${b.key}`">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t(`admin.analytics.breakdown.${b.key}.title`) }}</h3>
              <p class="mb-2 text-xs text-gray-500">{{ t(`admin.analytics.breakdown.${b.key}.hint`) }}</p>
              <table v-if="b.rows.length" class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500">
                    <th class="py-1">{{ t(`admin.analytics.breakdown.${b.key}.col`) }}</th>
                    <th>{{ t('admin.analytics.breakdown.count') }}</th>
                    <th>{{ t('admin.analytics.breakdown.visitors') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in b.rows" :key="r.key" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1 font-mono text-xs">{{ r.key }}</td>
                    <td>{{ r.count }}</td>
                    <td>{{ r.visitors }}</td>
                  </tr>
                </tbody>
              </table>
              <p v-else class="text-sm text-gray-500">{{ t('admin.analytics.empty') }}</p>
            </div>
            <div>
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.devices') }}</h3>
              <div v-if="deviceRows.length" class="flex flex-wrap gap-4 text-sm">
                <span v-for="d in deviceRows" :key="d.name">{{ t(`admin.analytics.device.${d.name}`) }} <b>{{ d.n }}</b>（{{ pct(d.n, deviceTotal) }}）</span>
              </div>
              <p v-else class="text-sm text-gray-500">{{ t('admin.analytics.empty') }}</p>
            </div>
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.retention') }}</h3>
              <p class="mb-2 text-xs text-gray-500">{{ t('admin.analytics.retentionHint') }}</p>
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500">
                    <th class="py-1">{{ t('admin.analytics.col.week') }}</th>
                    <th>{{ t('admin.analytics.col.cohort') }}</th>
                    <th>{{ t('admin.analytics.col.activated') }}</th>
                    <th>{{ t('admin.analytics.col.day1') }}</th>
                    <th>{{ t('admin.analytics.col.day2_7') }}</th>
                    <th>{{ t('admin.analytics.col.day8_30') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in data.retention || []" :key="r.week" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1">{{ r.week }}</td>
                    <td>{{ r.signups }}</td>
                    <td>{{ withPct(r.activated, r.signups) }}</td>
                    <td>{{ withPct(r.day1, r.signups) }}</td>
                    <td>{{ withPct(r.day2_7, r.signups) }}</td>
                    <td>{{ withPct(r.day8_30, r.signups) }}</td>
                  </tr>
                  <tr v-if="!data.retention?.length"><td colspan="6" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- Channels -->
        <div class="card p-4" data-testid="analytics-channels">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.channels') }}</h3>
          <p class="mb-2 text-xs text-gray-500">
            {{ t('admin.analytics.channelsHint') }}
            <router-link to="/admin/channel-links" class="text-primary-600 hover:underline dark:text-primary-400">{{ t('admin.analytics.manageChannelLinks') }}</router-link>
          </p>
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500">
                  <th class="py-1">{{ t('admin.analytics.col.source') }}</th>
                  <th>{{ t('admin.analytics.col.visitors') }}</th>
                  <th>{{ t('admin.analytics.col.signups') }}</th>
                  <th>{{ t('admin.analytics.col.activated') }}</th>
                  <th>{{ t('admin.analytics.col.paidUsers') }}</th>
                  <th>{{ t('admin.analytics.col.revenue') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in data.channels || []" :key="c.source" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-1 font-medium">{{ sourceLabel(c.source) }}</td>
                  <td>{{ c.visitors }}</td>
                  <td>{{ c.signups }}</td>
                  <td>{{ c.activated }}</td>
                  <td>{{ c.paid_users }}</td>
                  <td>{{ c.revenue ? c.revenue.toFixed(2) : 0 }}</td>
                </tr>
                <tr v-if="!data.channels?.length"><td colspan="6" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-2">
          <!-- Pages -->
          <div class="card p-4">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.pages') }}</h3>
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500">
                  <th class="py-1">{{ t('admin.analytics.col.page') }}</th>
                  <th>{{ t('admin.analytics.col.views') }}</th>
                  <th>{{ t('admin.analytics.col.visitors') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in data.pages || []" :key="p.app + p.path" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="max-w-xs truncate py-1 font-mono text-xs" :title="p.path">
                    <span class="mr-1 rounded bg-gray-100 px-1 font-sans text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t(`admin.analytics.apps.${p.app}`) }}</span>{{ p.path }}
                  </td>
                  <td>{{ p.views }}</td>
                  <td>{{ p.visitors }}</td>
                </tr>
                <tr v-if="!data.pages?.length"><td colspan="3" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
              </tbody>
            </table>
          </div>

          <!-- Features -->
          <div class="card p-4" data-testid="analytics-features">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.analytics.features') }}</h3>
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500">
                  <th class="py-1">{{ t('admin.analytics.col.event') }}</th>
                  <th>{{ t('admin.analytics.col.count') }}</th>
                  <th>{{ t('admin.analytics.col.visitors') }}</th>
                  <th>{{ t('admin.analytics.col.users') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="f in data.features || []" :key="f.event" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-1">{{ eventLabel(f.event) }}</td>
                  <td>{{ f.count }}</td>
                  <td>{{ f.visitors }}</td>
                  <td>{{ f.users }}</td>
                </tr>
                <tr v-if="!data.features?.length"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.analytics.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip } from 'chart.js'
import { Line } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getOverview, getReminder, setReminder, type ActivationReminderStatus, type AnalyticsOverview } from '@/api/admin/analytics'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)

const { t, te } = useI18n()
const ranges = [7, 30, 90]
const days = ref(30)
const data = ref<AnalyticsOverview | null>(null)
const loading = ref(false)
const error = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    data.value = await getOverview(days.value)
  } catch (e: unknown) {
    error.value = extractApiErrorMessage(e, t('admin.analytics.loadFailed'))
  } finally {
    loading.value = false
  }
}

const appStore = useAppStore()
const reminder = ref<ActivationReminderStatus | null>(null)
const showPreview = ref(false)
const savingReminder = ref(false)

async function loadReminder(): Promise<void> {
  try {
    reminder.value = await getReminder()
  } catch {
    reminder.value = null
  }
}

async function toggleReminder(): Promise<void> {
  if (!reminder.value) return
  const on = !reminder.value.enabled
  if (on && !window.confirm(t('admin.analytics.reminder.confirm', { due: reminder.value.due }))) return
  savingReminder.value = true
  try {
    reminder.value = await setReminder(on)
    appStore.showSuccess(on ? t('admin.analytics.reminder.on') : t('admin.analytics.reminder.off'))
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  } finally {
    savingReminder.value = false
  }
}

function setDays(n: number): void {
  days.value = n
  void load()
}

const kpis = computed(() => {
  const s = data.value?.totals
  if (!s) return []
  return [
    { key: 'visitors', value: s.visitors, hint: 'visitorsHint' },
    { key: 'activeUsers', value: s.active_users, hint: 'activeHint' },
    { key: 'wau', value: s.wau, hint: '' },
    { key: 'signups', value: s.signups, hint: '' },
    { key: 'activated', value: `${s.activated}（${pct(s.activated, s.signups)}）`, hint: 'activatedHint' },
    { key: 'paidUsers', value: s.paid_users, hint: '' },
    { key: 'revenue', value: s.revenue.toFixed(2), hint: 'revenueHint' }
  ]
})

const trendData = computed(() => {
  const d = data.value?.daily || []
  const line = (label: string, color: string, values: number[]) => ({
    label,
    data: values,
    borderColor: color,
    backgroundColor: color,
    tension: 0.3,
    pointRadius: 2
  })
  return {
    labels: d.map((x) => x.day.slice(5)),
    datasets: [
      line(t('admin.analytics.series.visitors'), '#6366f1', d.map((x) => x.visitors)),
      line(t('admin.analytics.series.activeUsers'), '#10b981', d.map((x) => x.active_users)),
      line(t('admin.analytics.series.apiUsers'), '#f59e0b', d.map((x) => x.api_users)),
      line(t('admin.analytics.series.signups'), '#ef4444', d.map((x) => x.signups))
    ]
  }
})

const lineOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: { legend: { position: 'top' as const, labels: { usePointStyle: true, boxWidth: 8 } } },
  scales: { y: { beginAtZero: true, ticks: { precision: 0 } } }
}

const funnelTop = computed(() => Math.max(1, ...(data.value?.funnel || []).map((s) => s.count)))
const deviceRows = computed(() =>
  Object.entries(data.value?.devices || {})
    .map(([name, n]) => ({ name, n }))
    .sort((a, b) => b.n - a.n)
)
const breakdowns = computed(() => [
  { key: 'signupErrors', rows: data.value?.signup_errors || [] },
  { key: 'ctaClicks', rows: data.value?.cta_clicks || [] }
])
const deviceTotal = computed(() => deviceRows.value.reduce((s, d) => s + d.n, 0))

function pct(n: number, of: number): string {
  return of > 0 ? `${Math.round((n / of) * 100)}%` : '–'
}

function withPct(n: number, of: number): string {
  return `${n}（${pct(n, of)}）`
}

function barWidth(n: number, top: number): string {
  return `${Math.max(8, (n / top) * 100)}%`
}

function sourceLabel(src: string): string {
  if (src.startsWith('ref:')) return t('admin.analytics.source.ref', { host: src.slice(4) })
  const key = `admin.analytics.source.${src}`
  return te(key) ? t(key) : src
}

function eventLabel(ev: string): string {
  const key = `admin.analytics.events.${ev}`
  return te(key) ? t(key) : ev
}

onMounted(() => {
  void load()
  void loadReminder()
})
</script>
