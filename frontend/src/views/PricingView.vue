<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar login-redirect="/pricing" />
    <main class="mx-auto max-w-6xl space-y-10 px-4 py-8 sm:px-6 lg:px-8" data-testid="pricing-view">
      <header class="space-y-2 text-center">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ t('pricing.title') }}</h1>
        <p class="mx-auto max-w-2xl text-sm text-gray-600 dark:text-dark-300">{{ t('pricing.subtitle') }}</p>
      </header>

      <div v-if="loading" class="grid gap-4 md:grid-cols-2">
        <div v-for="i in 2" :key="i" class="h-56 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      </div>
      <div v-else-if="error" class="card p-6 text-center text-sm text-red-600">{{ error }}</div>

      <template v-else-if="data">
        <!-- The two ways to pay -->
        <section class="grid gap-4 md:grid-cols-2" data-testid="pricing-modes">
          <div class="card space-y-3 p-6">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('pricing.payg.title') }}</h2>
              <span class="rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">{{ t('pricing.payg.badge') }}</span>
            </div>
            <p class="text-3xl font-bold text-gray-900 dark:text-white">
              ¥1 = ${{ fmt(data.recharge_multiplier) }}
              <span class="text-sm font-normal text-gray-500">{{ t('pricing.payg.unit') }}</span>
            </p>
            <ul class="space-y-1.5 text-sm text-gray-600 dark:text-dark-300">
              <li v-for="k in ['p1', 'p2', 'p3']" :key="k" class="flex gap-2"><span class="text-emerald-500">✓</span>{{ t(`pricing.payg.${k}`, { min: fmt(data.min_recharge) }) }}</li>
            </ul>
            <p v-if="paygGroups.length" class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('pricing.payg.groups') }}{{ paygGroups.map((g) => g.name + (g.rate_multiplier !== 1 ? `（×${fmt(g.rate_multiplier)}）` : '')).join('、') }}
            </p>
            <p class="text-xs font-medium text-gray-700 dark:text-dark-200">{{ t('pricing.payg.fit') }}</p>
          </div>
          <div class="card space-y-3 p-6 ring-1 ring-primary-200 dark:ring-primary-800">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('pricing.sub.title') }}</h2>
              <span class="rounded-full bg-primary-50 px-2.5 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ t('pricing.sub.badge') }}</span>
            </div>
            <p v-if="cheapestPlan" class="text-3xl font-bold text-gray-900 dark:text-white">
              ¥{{ fmt(cheapestPlan.price) }}
              <span class="text-sm font-normal text-gray-500">{{ t('pricing.sub.from', { days: planDays(cheapestPlan) }) }}</span>
            </p>
            <ul class="space-y-1.5 text-sm text-gray-600 dark:text-dark-300">
              <li v-for="k in ['p1', 'p2', 'p3']" :key="k" class="flex gap-2"><span class="text-primary-500">✓</span>{{ t(`pricing.sub.${k}`) }}</li>
            </ul>
            <p class="text-xs font-medium text-gray-700 dark:text-dark-200">{{ t('pricing.sub.fit') }}</p>
          </div>
        </section>

        <!-- Plans on sale -->
        <section v-if="data.plans.length" class="space-y-3" data-testid="pricing-plans">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('pricing.plans.title') }}</h2>
          <div class="overflow-x-auto rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
            <table class="w-full min-w-[720px] text-sm">
              <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-2.5">{{ t('pricing.plans.plan') }}</th>
                  <th class="px-4 py-2.5">{{ t('pricing.plans.price') }}</th>
                  <th class="px-4 py-2.5">{{ t('pricing.plans.perDay') }}</th>
                  <th class="px-4 py-2.5">{{ t('pricing.plans.limits') }}</th>
                  <th class="px-4 py-2.5">{{ t('pricing.plans.value') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in data.plans" :key="p.id" class="border-t border-gray-100 align-top dark:border-dark-700" :data-testid="`pricing-plan-${p.id}`">
                  <td class="px-4 py-3">
                    <div class="font-medium text-gray-900 dark:text-white">{{ p.name }}</div>
                    <div class="text-xs text-gray-500">{{ p.description || p.group_name }}</div>
                  </td>
                  <td class="px-4 py-3 whitespace-nowrap">
                    <span class="font-semibold text-gray-900 dark:text-white">¥{{ fmt(p.price) }}</span>
                    <span class="text-xs text-gray-500"> / {{ t('pricing.plans.days', { n: planDays(p) }) }}</span>
                    <div v-if="p.original_price && p.original_price > p.price" class="text-xs text-gray-400 line-through">¥{{ fmt(p.original_price) }}</div>
                  </td>
                  <td class="px-4 py-3 whitespace-nowrap">¥{{ fmt(p.price / Math.max(1, planDays(p))) }}</td>
                  <td class="px-4 py-3 text-xs leading-5 text-gray-600 dark:text-dark-300">
                    <div v-if="p.daily_limit_usd">{{ t('pricing.plans.daily', { v: fmt0(p.daily_limit_usd) }) }}</div>
                    <div v-if="p.weekly_limit_usd">{{ t('pricing.plans.weekly', { v: fmt0(p.weekly_limit_usd) }) }}</div>
                    <div v-if="p.monthly_limit_usd">{{ t('pricing.plans.monthly', { v: fmt0(p.monthly_limit_usd) }) }}</div>
                    <div v-if="!p.daily_limit_usd && !p.weekly_limit_usd && !p.monthly_limit_usd">{{ t('pricing.plans.noLimit') }}</div>
                  </td>
                  <td class="px-4 py-3 text-xs leading-5 text-gray-600 dark:text-dark-300">
                    <template v-if="planPeriodCapUSD(p)">{{ t('pricing.plans.valueText', { days: planDays(p), usd: fmt0(planPeriodCapUSD(p)!), cny: fmt0(planPeriodCapUSD(p)! / data.recharge_multiplier) }) }}</template>
                    <template v-else>–</template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('pricing.plans.note') }}</p>
        </section>

        <!-- Estimator -->
        <section v-if="canEstimate" class="card space-y-4 p-6" data-testid="pricing-estimator">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('pricing.estimate.title') }}</h2>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('pricing.estimate.basis', { cost: fmt(data.avg_request_cost_usd, 3), n: data.avg_sample_requests }) }}</p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <label class="space-y-1 text-sm">
              <span class="flex justify-between text-gray-700 dark:text-dark-200">{{ t('pricing.estimate.perDay') }}<b data-testid="pricing-requests">{{ requestsPerDay }}</b></span>
              <input v-model.number="requestsPerDay" type="range" min="5" max="1000" step="5" class="w-full" data-testid="pricing-requests-input" />
            </label>
            <label class="space-y-1 text-sm">
              <span class="flex justify-between text-gray-700 dark:text-dark-200">{{ t('pricing.estimate.days') }}<b>{{ daysPerMonth }}</b></span>
              <input v-model.number="daysPerMonth" type="range" min="1" max="30" step="1" class="w-full" />
            </label>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800/60" :class="best?.key === 'payg' ? 'ring-2 ring-emerald-400' : ''" data-testid="pricing-estimate-payg">
              <div class="text-xs text-gray-500">{{ t('pricing.payg.title') }}</div>
              <div class="text-xl font-bold text-gray-900 dark:text-white">¥{{ fmt0(paygMonthly) }}<span class="text-xs font-normal text-gray-500"> / {{ t('pricing.estimate.month') }}</span></div>
            </div>
            <div v-for="o in planOptions" :key="o.plan.id" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800/60" :class="best?.key === `plan-${o.plan.id}` ? 'ring-2 ring-primary-400' : ''" :data-testid="`pricing-estimate-plan-${o.plan.id}`">
              <div class="text-xs text-gray-500">{{ o.plan.name }}</div>
              <div class="text-xl font-bold text-gray-900 dark:text-white">¥{{ fmt0(o.monthly) }}<span class="text-xs font-normal text-gray-500"> / {{ t('pricing.estimate.month') }}</span></div>
              <div v-if="!o.fits" class="text-xs text-amber-600">{{ t('pricing.estimate.overLimit') }}</div>
            </div>
          </div>
          <p v-if="best" class="text-sm font-medium text-gray-800 dark:text-dark-100" data-testid="pricing-recommendation">
            {{ best.key === 'payg' ? t('pricing.estimate.pickPayg') : t('pricing.estimate.pickPlan', { name: best.name }) }}
          </p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('pricing.estimate.disclaimer') }}</p>
        </section>

        <!-- FAQ -->
        <section class="space-y-3">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('pricing.faq.title') }}</h2>
          <div class="grid gap-3 md:grid-cols-2">
            <div v-for="k in faqKeys" :key="k" class="card p-4">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`pricing.faq.${k}`) }}</p>
              <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t(`pricing.faq.${k}a${smartBilling && SMART_FAQ.includes(k) ? 'Smart' : ''}`, { min: fmt(data.min_recharge), days: priceLockGraceDays ?? 0 }) }}</p>
            </div>
          </div>
        </section>

        <section class="flex flex-wrap justify-center gap-3">
          <RouterLink to="/purchase" class="btn btn-primary" data-testid="pricing-buy">{{ t('pricing.buy') }}</RouterLink>
          <a href="/learn/connect/" class="btn btn-secondary">{{ t('pricing.guide') }}</a>
        </section>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { getPublicPricing, planDays, planPeriodCapUSD, planMonthlyPrice, type PricingPlan, type PublicPricing } from '@/api/pricing'
import { extractApiErrorMessage } from '@/utils/apiError'
import { growthAPI } from '@/api/growth'

const { t } = useI18n()
const data = ref<PublicPricing | null>(null)
const loading = ref(true)
const error = ref('')
const requestsPerDay = ref(80)
const daysPerMonth = ref(22)

// Only estimate from a meaningful sample of real requests.
const MIN_SAMPLES = 200

// 老用户锁价 FAQ, shown when the program is on.
const priceLockGraceDays = ref<number | null>(null)
// With smart billing one key serves both subscriptions and balance; these answers change.
const smartBilling = ref(false)
const SMART_FAQ = ['q1', 'q2', 'q3']
const faqKeys = computed(() => (priceLockGraceDays.value === null ? ['q1', 'q2', 'q3', 'q4', 'q5', 'q6'] : ['q7', 'q1', 'q2', 'q3', 'q4', 'q5', 'q6']))

onMounted(async () => {
  growthAPI.getPublicConfig().then((cfg) => {
    priceLockGraceDays.value = cfg.price_lock_enabled ? cfg.price_lock_grace_days : null
    smartBilling.value = !!cfg.smart_billing
  }).catch(() => {})
  try {
    data.value = await getPublicPricing()
  } catch (e: unknown) {
    error.value = extractApiErrorMessage(e, t('pricing.loadFailed'))
  } finally {
    loading.value = false
  }
})

const fmt = (v: number, digits = 2) => {
  const s = v.toFixed(digits)
  return s.includes('.') ? s.replace(/\.?0+$/, '') : s
}
const fmt0 = (v: number) => Math.round(v).toLocaleString('zh-CN')

const paygGroups = computed(() => data.value?.pay_as_you_go || [])
const cheapestPlan = computed(() => {
  const plans = data.value?.plans || []
  return plans.length ? plans.reduce((a, b) => (b.price < a.price ? b : a)) : null
})
// Coding-tool usage is billed on GPT groups: compare OpenAI plans only.
const gptPlans = computed(() => (data.value?.plans || []).filter((p) => p.platform === 'openai'))
const canEstimate = computed(() => !!data.value && data.value.avg_sample_requests >= MIN_SAMPLES && data.value.avg_request_cost_usd > 0)

const dailyUSD = computed(() => requestsPerDay.value * (data.value?.avg_request_cost_usd || 0))
const paygRate = computed(() => paygGroups.value.find((g) => g.platform === 'openai')?.rate_multiplier || 1)
const paygMonthly = computed(() => (dailyUSD.value * daysPerMonth.value * paygRate.value) / (data.value?.recharge_multiplier || 1))

function fits(p: PricingPlan): boolean {
  const d = dailyUSD.value
  if (p.daily_limit_usd && d > p.daily_limit_usd) return false
  if (p.weekly_limit_usd && d * Math.min(7, daysPerMonth.value) > p.weekly_limit_usd) return false
  if (p.monthly_limit_usd && d * daysPerMonth.value > p.monthly_limit_usd) return false
  return true
}

const planOptions = computed(() => gptPlans.value.map((plan) => ({ plan, monthly: planMonthlyPrice(plan), fits: fits(plan) })))

const best = computed(() => {
  if (!canEstimate.value) return null
  let pick: { key: string; name: string; cost: number } = { key: 'payg', name: '', cost: paygMonthly.value }
  for (const o of planOptions.value) {
    if (o.fits && o.monthly < pick.cost) pick = { key: `plan-${o.plan.id}`, name: o.plan.name, cost: o.monthly }
  }
  return pick
})
</script>
