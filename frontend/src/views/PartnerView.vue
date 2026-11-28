<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar login-redirect="/partner" />
    <main class="mx-auto max-w-5xl space-y-14 px-4 py-10 sm:px-6 lg:px-8" data-testid="partner-view">
      <!-- What this is, with the live numbers from 推广设置 -->
      <header class="text-center">
        <p class="text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('partner.eyebrow') }}</p>
        <h1 class="mt-2 text-3xl font-extrabold tracking-tight text-gray-900 sm:text-4xl dark:text-white">{{ t('partner.heroTitle') }}</h1>
        <p class="mx-auto mt-4 max-w-2xl text-base leading-relaxed text-gray-600 dark:text-dark-300">{{ t('partner.heroSubtitle', { site: siteName }) }}</p>

        <div v-if="stats.length" class="mx-auto mt-8 grid max-w-3xl gap-3" :class="stats.length >= 3 ? 'sm:grid-cols-3' : 'sm:grid-cols-2'" data-testid="partner-stats">
          <div v-for="s in stats" :key="s.key" class="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-4 dark:border-amber-700/40 dark:bg-amber-900/20">
            <p class="text-3xl font-extrabold text-amber-700 dark:text-amber-300">{{ s.value }}</p>
            <p class="mt-1 text-sm text-amber-900/80 dark:text-amber-100/70">{{ s.label }}</p>
          </div>
        </div>
        <p v-if="rate > 0" class="mx-auto mt-4 max-w-2xl text-sm text-gray-500 dark:text-dark-400" data-testid="partner-example">
          {{ t('partner.example', { rebate: money(rate), rate: money(rate) }) }}
        </p>

        <div class="mt-7 flex justify-center">
          <router-link :to="inviteTo" class="btn btn-primary px-6 py-3 text-base" data-testid="partner-hero-cta" @click="trackCta('hero')">
            {{ isAuthenticated ? t('partner.ctaInvite') : t('partner.ctaRegister') }} →
          </router-link>
        </div>
      </header>

      <!-- Three ways to partner -->
      <section>
        <div class="text-center">
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('partner.ways.title') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('partner.ways.subtitle') }}</p>
        </div>
        <div class="mt-6 grid gap-4 md:grid-cols-3">
          <div v-for="w in ways" :key="w.key" class="card flex flex-col p-6" :data-testid="`partner-way-${w.key}`">
            <span class="flex h-11 w-11 items-center justify-center rounded-xl text-white" :class="w.tone">
              <Icon :name="w.icon" size="md" />
            </span>
            <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white">{{ w.title }}</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('partner.ways.fit') }}{{ w.fit }}</p>
            <ul class="mt-4 flex-1 space-y-2 text-sm text-gray-600 dark:text-dark-300">
              <li v-for="(p, i) in w.points" :key="i" class="flex gap-2"><span class="text-emerald-500">✓</span><span>{{ p }}</span></li>
            </ul>
            <p v-if="w.note" class="mt-4 rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ w.note }}</p>
            <router-link v-if="w.to" :to="w.to" class="mt-5 inline-flex items-center justify-center rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-800 transition hover:bg-gray-50 dark:border-dark-600 dark:text-dark-100 dark:hover:bg-dark-800" @click="trackCta(`way-${w.key}`)">
              {{ w.cta }} →
            </router-link>
          </div>
        </div>
      </section>

      <!-- How the money arrives -->
      <section class="grid gap-8 md:grid-cols-2">
        <div>
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('partner.steps.title') }}</h2>
          <ol class="mt-4 space-y-3">
            <li v-for="(s, i) in steps" :key="i" class="flex items-start gap-3">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-600 text-sm font-semibold text-white">{{ i + 1 }}</span>
              <span class="pt-0.5 text-sm text-gray-700 dark:text-dark-200">{{ s }}</span>
            </li>
          </ol>
        </div>
        <div>
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('partner.rules.title') }}</h2>
          <ul class="mt-4 space-y-2.5 text-sm text-gray-600 dark:text-dark-300" data-testid="partner-rules">
            <li v-for="k in ['r1', 'r2', 'r3', 'r4', 'r5']" :key="k" class="flex gap-2"><Icon name="shield" size="sm" class="mt-0.5 shrink-0 text-gray-400" /><span>{{ t(`partner.rules.${k}`) }}</span></li>
          </ul>
        </div>
      </section>

      <section class="rounded-3xl bg-gradient-to-br from-primary-600 to-sky-500 px-6 py-10 text-center text-white">
        <h2 class="text-2xl font-bold">{{ t('partner.final.title') }}</h2>
        <p class="mt-2 text-white/80">{{ t('partner.final.subtitle') }}</p>
        <router-link :to="inviteTo" class="mt-6 inline-flex items-center rounded-xl bg-white px-6 py-3 font-semibold text-primary-700 transition hover:bg-white/90" @click="trackCta('final')">
          {{ isAuthenticated ? t('partner.ctaInvite') : t('partner.ctaRegister') }} →
        </router-link>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import Icon from '@/components/icons/Icon.vue'
import { growthAPI, type GrowthPublicConfig } from '@/api/growth'
import { useAppStore, useAuthStore } from '@/stores'
import { track } from '@/utils/analytics'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const contactInfo = computed(() => String(appStore.cachedPublicSettings?.contact_info || '').trim())
const isAuthenticated = computed(() => authStore.isAuthenticated)
const inviteTo = computed(() => (isAuthenticated.value ? '/affiliate' : '/register?redirect=/affiliate'))
const trackCta = (where: string) => track('cta_click', { where: `partner-${where}` })

const cfg = ref<GrowthPublicConfig | null>(null)
const money = (n: number) => String(Math.round(n * 100) / 100)
const rate = computed(() => (cfg.value?.affiliate_enabled ? Number(cfg.value.inviter_rebate_rate_percent || 0) : 0))
const withdrawOn = computed(() => !!cfg.value?.withdraw_enabled)

// Only what is switched on in 推广设置 is shown, so the page never promises a closed program.
const stats = computed(() => {
  const c = cfg.value
  if (!c?.affiliate_enabled) return []
  const out: { key: string; value: string; label: string }[] = []
  if (rate.value > 0) out.push({ key: 'rebate', value: `${money(rate.value)}%`, label: t('partner.stats.rebate') })
  if (c.invitee_bonus_rate_percent > 0) out.push({ key: 'bonus', value: `${money(c.invitee_bonus_rate_percent)}%`, label: t('partner.stats.bonus') })
  else if (c.invitee_signup_bonus > 0) out.push({ key: 'signup', value: `$${money(c.invitee_signup_bonus)}`, label: t('partner.stats.signup') })
  if (c.withdraw_enabled) out.push({ key: 'withdraw', value: t('partner.stats.withdrawValue', { min: money(Number(c.withdraw_min_cny || 0)) }), label: t('partner.stats.withdraw') })
  return out
})

type IconName = 'users' | 'chatBubble' | 'book'
const ways = computed(() => {
  const list: { key: string; icon: IconName; tone: string; title: string; fit: string; points: string[]; note?: string; cta: string; to?: string }[] = []
  if (cfg.value?.affiliate_enabled) {
    list.push({
      key: 'invite',
      icon: 'users',
      tone: 'bg-amber-500',
      title: t('partner.ways.invite.title'),
      fit: t('partner.ways.invite.fit'),
      points: [t('partner.ways.invite.p1'), t('partner.ways.invite.p2'), rate.value > 0 ? t('partner.ways.invite.p3', { rate: money(rate.value) }) : t('partner.ways.invite.p3NoRate')],
      cta: t('partner.ways.invite.cta'),
      to: inviteTo.value,
    })
  }
  list.push({
    key: 'kol',
    icon: 'chatBubble',
    tone: 'bg-pink-500',
    title: t('partner.ways.kol.title'),
    fit: t('partner.ways.kol.fit'),
    points: [t('partner.ways.kol.p1'), t('partner.ways.kol.p2'), t('partner.ways.kol.p3')],
    note: contactInfo.value ? t('partner.ways.kol.contact', { info: contactInfo.value }) : t('partner.ways.kol.noContact'),
    cta: t('partner.ways.kol.cta'),
    to: contactInfo.value ? undefined : inviteTo.value,
  })
  list.push({
    key: 'creator',
    icon: 'book',
    tone: 'bg-emerald-500',
    title: t('partner.ways.creator.title'),
    fit: t('partner.ways.creator.fit'),
    points: [t('partner.ways.creator.p1'), t('partner.ways.creator.p2'), t('partner.ways.creator.p3')],
    cta: t('partner.ways.creator.cta'),
    to: '/creator',
  })
  return list
})

const steps = computed(() => [
  t('partner.steps.s1'),
  t('partner.steps.s2'),
  t('partner.steps.s3'),
  withdrawOn.value ? t('partner.steps.s4') : t('partner.steps.s4NoWithdraw'),
])

onMounted(() => {
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
  growthAPI
    .getPublicConfig()
    .then((c) => (cfg.value = c))
    .catch(() => {})
})
</script>
