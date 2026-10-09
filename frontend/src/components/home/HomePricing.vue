<template>
  <section v-if="plans.length" class="mb-20" data-testid="home-pricing">
    <div class="mb-8 text-center">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.pricing.title') }}</h2>
      <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('home.pricing.subtitle', { rate: rechargeText }) }}</p>
    </div>

    <div class="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
      <!-- Subscription plans -->
      <div
        v-for="p in plans"
        :key="p.id"
        class="relative flex flex-col rounded-2xl border bg-white/80 p-5 backdrop-blur-sm dark:bg-dark-800/80"
        :class="p.recommended ? 'border-amber-400 ring-2 ring-amber-300/60 dark:border-amber-500' : 'border-gray-200 dark:border-dark-700'"
        :data-testid="`home-plan-${p.id}`"
      >
        <span v-if="p.recommended" class="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-amber-400 px-3 py-0.5 text-xs font-bold text-amber-950 shadow">⭐ {{ t('home.pricing.recommended') }}</span>
        <p class="font-semibold text-gray-900 dark:text-white">{{ p.name }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ p.description || p.group_name }}</p>
        <div class="mt-3 flex items-baseline gap-1.5">
          <span class="text-3xl font-extrabold text-gray-900 dark:text-white">¥{{ fmt(p.price) }}</span>
          <span class="text-sm text-gray-500">/ {{ t('home.pricing.days', { n: p.days }) }}</span>
          <span v-if="p.original_price && p.original_price > p.price" class="text-sm text-gray-400 line-through">¥{{ fmt(p.original_price) }}</span>
        </div>
        <p v-if="p.saving >= 0.3" class="mt-2 inline-flex w-fit rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300" :data-testid="`home-plan-saving-${p.id}`">
          {{ t('home.pricing.saving', { value: fmt(p.valueCNY), percent: Math.round(p.saving * 100) }) }}
        </p>
        <ul class="mt-4 flex-1 space-y-1.5 text-sm text-gray-700 dark:text-dark-200">
          <li v-if="p.daily_limit_usd">✓ {{ t('home.pricing.daily', { usd: fmt(p.daily_limit_usd) }) }}<span v-if="p.perDay" class="text-gray-500">{{ t('home.pricing.perDay', { n: p.perDay }) }}</span></li>
          <li v-if="p.weekly_limit_usd && p.days >= 7">✓ {{ t('home.pricing.weekly', { usd: fmt(p.weekly_limit_usd) }) }}</li>
          <li v-if="p.monthly_limit_usd && p.days >= 30">✓ {{ t('home.pricing.monthly', { usd: fmt(p.monthly_limit_usd) }) }}</li>
          <li class="text-xs text-gray-500 dark:text-dark-400">{{ t('home.pricing.group', { group: p.group_name }) }}</li>
        </ul>
        <router-link
          :to="subscribeTarget(p.id)"
          class="btn mt-5 w-full"
          :class="p.recommended ? 'btn-primary' : 'btn-secondary'"
          @click="track('cta_click', { where: 'home-pricing-plan' })"
        >
          {{ t('home.pricing.subscribe') }}
        </router-link>
      </div>

      <!-- Pay as you go -->
      <div class="flex flex-col rounded-2xl border border-dashed border-gray-300 bg-white/60 p-5 dark:border-dark-600 dark:bg-dark-800/60" data-testid="home-plan-payg">
        <p class="font-semibold text-gray-900 dark:text-white">{{ t('home.pricing.paygTitle') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('home.pricing.paygDesc') }}</p>
        <div class="mt-3 flex items-baseline gap-1.5">
          <span class="text-3xl font-extrabold text-gray-900 dark:text-white">¥{{ fmt(minRecharge) }}</span>
          <span class="text-sm text-gray-500">{{ t('home.pricing.paygFrom') }}</span>
        </div>
        <ul class="mt-4 flex-1 space-y-1.5 text-sm text-gray-700 dark:text-dark-200">
          <li>✓ {{ t('home.pricing.paygRate', { rate: rechargeText }) }}</li>
          <li v-if="avgCost">✓ {{ t('home.pricing.paygAvg', { usd: avgCost }) }}</li>
          <li v-if="signupBonus > 0">✓ {{ t('home.pricing.paygBonus', { amount: fmt(signupBonus) }) }}</li>
        </ul>
        <router-link :to="isAuthenticated ? '/purchase' : '/register'" class="btn btn-secondary mt-5 w-full" @click="track('cta_click', { where: 'home-pricing-payg' })">
          {{ isAuthenticated ? t('home.pricing.topUp') : t('home.pricing.tryFree') }}
        </router-link>
      </div>
    </div>

    <!-- Why it is safe to pay: what a first-time buyer worries about -->
    <div class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4" data-testid="home-trust">
      <div v-for="item in trust" :key="item.key" class="rounded-xl bg-white/60 px-4 py-3 text-sm dark:bg-dark-800/60">
        <p class="font-medium text-gray-900 dark:text-white">{{ item.icon }} {{ t(`home.trust.${item.key}.title`) }}</p>
        <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-dark-300">{{ item.desc }}</p>
      </div>
    </div>
    <p v-if="totalRequestsText" class="mt-3 text-center text-sm text-gray-600 dark:text-dark-300" data-testid="home-total-requests">
      {{ t('home.trust.total', { n: totalRequestsText }) }}
    </p>

    <p class="mt-4 text-center text-xs text-gray-500 dark:text-dark-400">
      {{ t('home.pricing.note') }}
      <router-link to="/pricing" class="text-primary-600 hover:underline dark:text-primary-400">{{ t('home.pricing.more') }} →</router-link>
    </p>
  </section>
</template>

<script setup lang="ts">
// Subscription prices on the home page: what each plan costs and how much usage it allows, next to
// pay-as-you-go. A plan used to its limits is worth (cap in USD ÷ recharge ratio) in yuan of top-ups.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPublicPricing, planDays, planPeriodCapUSD, type PublicPricing } from '@/api/pricing'
import { track } from '@/utils/analytics'

const props = defineProps<{ isAuthenticated: boolean; signupBonus?: number; contactInfo?: string }>()
const { t, locale } = useI18n()
const data = ref<PublicPricing | null>(null)

const fmt = (n: number) => String(Math.round(n * 100) / 100)
const ratio = computed(() => (data.value?.recharge_multiplier || 1))
const rechargeText = computed(() => fmt(ratio.value))
const minRecharge = computed(() => Math.max(1, data.value?.min_recharge || 1))
const avgCost = computed(() => {
  const c = data.value?.avg_request_cost_usd || 0
  return c > 0 && (data.value?.avg_sample_requests || 0) >= 100 ? c.toFixed(2) : ''
})
const signupBonus = computed(() => props.signupBonus || 0)

const plans = computed(() => {
  const d = data.value
  if (!d?.payment_enabled) return []
  return [...(d.plans || [])]
    .sort((a, b) => Number(!!b.recommended) - Number(!!a.recommended))
    .slice(0, 3)
    .map((p) => {
      const cap = planPeriodCapUSD(p)
      const valueCNY = cap ? cap / ratio.value : 0
      const avg = d.avg_request_cost_usd || 0
      return {
        ...p,
        days: planDays(p),
        valueCNY,
        saving: valueCNY > 0 ? 1 - p.price / valueCNY : 0,
        perDay: p.daily_limit_usd && avg > 0 && d.avg_sample_requests >= 100 ? Math.floor(p.daily_limit_usd / avg) : 0
      }
    })
})

const trust = computed(() => [
  { key: 'metered', icon: '🧾', desc: t('home.trust.metered.desc') },
  { key: 'failures', icon: '🛡️', desc: t('home.trust.failures.desc') },
  { key: 'payment', icon: '⚡', desc: t('home.trust.payment.desc') },
  {
    key: 'support',
    icon: '💬',
    desc: props.contactInfo ? t('home.trust.support.descContact', { info: props.contactInfo }) : t('home.trust.support.desc')
  }
])

// Only worth saying once the number is big enough to reassure (members' requests, admins excluded).
const TOTAL_REQUESTS_MIN = 10000
const totalRequestsText = computed(() => {
  const n = data.value?.total_requests || 0
  if (n < TOTAL_REQUESTS_MIN) return ''
  if (!String(locale.value).startsWith('zh')) return n.toLocaleString('en-US')
  return n >= 100000 ? `${Math.floor(n / 10000)} 万` : `${(Math.floor(n / 1000) / 10).toFixed(1)} 万`
})

function subscribeTarget(id: number): string {
  const target = `/purchase?tab=subscription&plan=${id}`
  return props.isAuthenticated ? target : `/register?redirect=${encodeURIComponent(target)}`
}

onMounted(async () => {
  try {
    data.value = await getPublicPricing()
  } catch {
    // no section without prices
  }
})
</script>
