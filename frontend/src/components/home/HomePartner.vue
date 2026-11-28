<template>
  <!-- 一起挣钱: the referral program as a reason to come back, with the live numbers from 推广设置 -->
  <section v-if="stats.length" class="mx-auto max-w-6xl px-4 pt-20 sm:px-6" data-testid="home-partner">
    <div class="grid items-center gap-8 rounded-3xl border border-amber-200 bg-gradient-to-br from-amber-50 to-white p-6 sm:p-10 lg:grid-cols-[1.1fr_1fr] dark:border-amber-700/40 dark:from-amber-900/20 dark:to-dark-900">
      <div>
        <p class="text-sm font-semibold text-amber-700 dark:text-amber-300">{{ t('partner.home.eyebrow') }}</p>
        <h2 class="mt-2 text-2xl font-bold tracking-tight text-gray-900 sm:text-3xl dark:text-white">{{ t('partner.home.title') }}</h2>
        <p class="mt-3 text-base leading-relaxed text-gray-600 dark:text-dark-300">{{ t('partner.home.subtitle', { site: siteName }) }}</p>
        <router-link to="/partner" class="mt-6 inline-flex items-center rounded-xl bg-amber-400 px-5 py-2.5 font-semibold text-amber-950 transition hover:bg-amber-300" data-testid="home-partner-cta" @click="track('cta_click', { where: 'home-partner' })">
          {{ t('partner.home.cta') }} →
        </router-link>
      </div>
      <div class="grid gap-3 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
        <div v-for="s in stats" :key="s.key" class="rounded-2xl bg-white/80 px-4 py-4 shadow-sm dark:bg-dark-900/70">
          <p class="text-2xl font-extrabold text-amber-700 dark:text-amber-300">{{ s.value }}</p>
          <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ s.label }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { growthAPI, type GrowthPublicConfig } from '@/api/growth'
import { track } from '@/utils/analytics'

const props = defineProps<{ siteName: string }>()
const { t } = useI18n()
const siteName = computed(() => props.siteName)

const cfg = ref<GrowthPublicConfig | null>(null)
const money = (n: number) => String(Math.round(n * 100) / 100)

// Hidden unless the referral program is on and pays the inviter something.
const stats = computed(() => {
  const c = cfg.value
  const rate = Number(c?.inviter_rebate_rate_percent || 0)
  if (!c?.affiliate_enabled || rate <= 0) return []
  const out = [{ key: 'rebate', value: `${money(rate)}%`, label: t('partner.stats.rebate') }]
  if (c.invitee_bonus_rate_percent > 0) out.push({ key: 'bonus', value: `${money(c.invitee_bonus_rate_percent)}%`, label: t('partner.stats.bonus') })
  if (c.withdraw_enabled) out.push({ key: 'withdraw', value: t('partner.stats.withdrawValue', { min: money(Number(c.withdraw_min_cny || 0)) }), label: t('partner.stats.withdraw') })
  return out
})

onMounted(() => {
  growthAPI
    .getPublicConfig()
    .then((c) => (cfg.value = c))
    .catch(() => {})
})
</script>
