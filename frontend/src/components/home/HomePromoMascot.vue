<template>
  <div
    v-if="offers.length"
    ref="rootEl"
    class="fixed right-0 top-[38%] z-30 flex flex-row-reverse items-start"
    data-testid="home-promo-mascot"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
  >
    <button
      type="button"
      class="mascot relative h-[62px] w-14 shrink-0 sm:h-[92px] sm:w-[84px]"
      :class="{ 'mascot-open': open }"
      :aria-label="open ? t('promoMascot.close') : t('promoMascot.open')"
      :aria-expanded="open"
      data-testid="home-promo-mascot-toggle"
      @click="toggle"
    >
      <span
        v-if="showBubble && !open"
        class="absolute -top-1.5 right-12 whitespace-nowrap rounded-full bg-pink-500 px-2.5 py-0.5 text-xs font-medium text-white shadow sm:right-[70px]"
      >{{ t('promoMascot.bubble', { n: unclaimedCount }) }}</span>
      <svg viewBox="0 0 84 92" class="mascot-bob h-full w-full" aria-hidden="true">
        <ellipse cx="30" cy="30" rx="15" ry="20" fill="#B5D4F4" stroke="#185FA5" stroke-width="2" transform="rotate(-25 30 30)" />
        <ellipse cx="54" cy="28" rx="13" ry="18" fill="#E6F1FB" stroke="#185FA5" stroke-width="2" transform="rotate(20 54 28)" />
        <ellipse cx="42" cy="58" rx="28" ry="26" fill="#FAC775" stroke="#633806" stroke-width="2.5" />
        <path d="M20 52 Q42 46 64 52" stroke="#412402" stroke-width="7" fill="none" stroke-linecap="round" />
        <path d="M18 66 Q42 72 66 66" stroke="#412402" stroke-width="7" fill="none" stroke-linecap="round" />
        <circle cx="33" cy="58" r="5.5" fill="#fff" />
        <circle cx="51" cy="58" r="5.5" fill="#fff" />
        <circle cx="34" cy="59" r="3" fill="#2C2C2A" />
        <circle cx="52" cy="59" r="3" fill="#2C2C2A" />
        <circle cx="26" cy="66" r="3.5" fill="#ED93B1" />
        <circle cx="58" cy="66" r="3.5" fill="#ED93B1" />
        <path d="M37 66 Q42 71 47 66" stroke="#412402" stroke-width="2" fill="none" stroke-linecap="round" />
        <path d="M34 34 Q30 22 24 18" stroke="#412402" stroke-width="2" fill="none" />
        <circle cx="23" cy="17" r="3.5" fill="#D4537E" />
        <path d="M50 34 Q54 22 60 18" stroke="#412402" stroke-width="2" fill="none" />
        <circle cx="61" cy="17" r="3.5" fill="#D4537E" />
      </svg>
    </button>

    <Transition name="promo-panel">
      <div
        v-if="open"
        class="mt-1 w-[min(318px,calc(100vw-4.5rem))] rounded-2xl border-2 border-amber-400 bg-white p-3 shadow-xl dark:border-amber-500/70 dark:bg-dark-900"
        data-testid="home-promo-mascot-panel"
      >
        <div class="flex items-center gap-1.5">
          <span class="text-[15px] font-semibold text-gray-900 dark:text-white">{{ t('promoMascot.title') }}</span>
          <span class="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">{{ t('promoMascot.stackable') }}</span>
        </div>
        <router-link
          v-for="offer in offers"
          :key="offer.key"
          :to="offer.to"
          class="promo-card mt-2 flex items-center gap-2.5 rounded-xl px-2.5 py-2"
          :class="[offer.tone.card, { 'opacity-60': offer.claimed }]"
          :data-testid="`promo-offer-${offer.key}`"
          @click="track('promo_mascot_click', { offer: offer.key })"
        >
          <span class="flex h-[34px] w-[34px] shrink-0 items-center justify-center rounded-[10px] text-white" :class="offer.tone.icon">
            <Icon :name="offer.icon" size="md" />
          </span>
          <span class="min-w-0">
            <span class="block text-[13px] font-medium" :class="offer.tone.title">{{ offer.title }}</span>
            <span class="block text-[11px] leading-snug" :class="offer.tone.desc">{{ offer.desc }}</span>
          </span>
          <span class="ml-auto whitespace-nowrap text-base font-semibold" :class="offer.tone.title">
            {{ offer.claimed ? t('promoMascot.claimed') : offer.value }}
          </span>
        </router-link>
        <div class="mt-3 flex gap-2">
          <router-link
            :to="primaryCta.to"
            class="flex-1 rounded-lg bg-amber-400 px-3 py-2 text-center text-sm font-semibold text-amber-950 transition hover:bg-amber-300"
            @click="track('promo_mascot_click', { offer: 'cta' })"
          >{{ primaryCta.label }}</router-link>
          <router-link
            to="/pricing"
            class="flex-1 rounded-lg border border-gray-200 px-3 py-2 text-center text-sm text-gray-700 transition hover:bg-gray-50 dark:border-dark-600 dark:text-dark-200 dark:hover:bg-dark-800"
            @click="track('promo_mascot_click', { offer: 'pricing' })"
          >{{ t('promoMascot.ctaPricing') }}</router-link>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { growthAPI, type GrowthPublicConfig } from '@/api/growth'
import { paymentAPI } from '@/api/payment'
import { useAppStore, useAuthStore } from '@/stores'
import { track } from '@/utils/analytics'

type IconName = 'gift' | 'dollar' | 'users' | 'book' | 'lock'
interface Offer {
  key: string
  icon: IconName
  title: string
  desc: string
  value: string
  to: string
  claimed: boolean
  tone: { card: string; icon: string; title: string; desc: string }
}

const TONES = {
  green: {
    card: 'bg-emerald-50 dark:bg-emerald-900/25',
    icon: 'bg-emerald-500',
    title: 'text-emerald-900 dark:text-emerald-200',
    desc: 'text-emerald-700 dark:text-emerald-300/80'
  },
  orange: {
    card: 'bg-orange-50 dark:bg-orange-900/25',
    icon: 'bg-orange-500',
    title: 'text-orange-900 dark:text-orange-200',
    desc: 'text-orange-700 dark:text-orange-300/80'
  },
  violet: {
    card: 'bg-violet-50 dark:bg-violet-900/25',
    icon: 'bg-violet-500',
    title: 'text-violet-900 dark:text-violet-200',
    desc: 'text-violet-700 dark:text-violet-300/80'
  },
  blue: {
    card: 'bg-sky-50 dark:bg-sky-900/25',
    icon: 'bg-sky-500',
    title: 'text-sky-900 dark:text-sky-200',
    desc: 'text-sky-700 dark:text-sky-300/80'
  },
  pink: {
    card: 'bg-pink-50 dark:bg-pink-900/25',
    icon: 'bg-pink-500',
    title: 'text-pink-900 dark:text-pink-200',
    desc: 'text-pink-700 dark:text-pink-300/80'
  }
}

// Opening the panel once hides the "N 个福利" bubble for the rest of the day.
const BUBBLE_SEEN_KEY = 'promo_mascot_seen'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const config = ref<GrowthPublicConfig | null>(null)
const firstTopupEligible = ref<boolean | null>(null)
const eduActive = ref(false)
const open = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const bubbleSeen = ref(localStorage.getItem(BUBBLE_SEEN_KEY) === new Date().toDateString())
let closeTimer: ReturnType<typeof setTimeout> | undefined
let checkoutLoaded = false

const isAuthenticated = computed(() => authStore.isAuthenticated)
const showBubble = computed(() => !bubbleSeen.value && unclaimedCount.value > 0)

const money = (n: number) => String(Math.round(n * 100) / 100)
const pct = (n: number) => String(Math.round(n * 10) / 10)

const offers = computed<Offer[]>(() => {
  const cfg = config.value
  const authed = isAuthenticated.value
  const list: Offer[] = []
  const signupBonus = Number(appStore.cachedPublicSettings?.signup_bonus || 0)
  const registrationOpen = appStore.cachedPublicSettings?.registration_enabled !== false

  if (signupBonus > 0 && (authed || registrationOpen)) {
    list.push({
      key: 'signup',
      icon: 'gift',
      title: t('promoMascot.signup.title'),
      desc: t('promoMascot.signup.desc'),
      value: `$${money(signupBonus)}`,
      to: authed ? '/dashboard' : '/register',
      claimed: authed,
      tone: TONES.green
    })
  }
  if (!cfg) return list

  const topupPct = Number(cfg.first_topup_bonus_percent || 0)
  if (topupPct > 0) {
    const min = Number(cfg.first_topup_min_amount || 0)
    const cap = Number(cfg.first_topup_bonus_cap || 0)
    let desc = t('promoMascot.firstTopup.desc')
    if (min > 0 && cap > 0) desc = t('promoMascot.firstTopup.descMinCap', { min: money(min), cap: money(cap) })
    else if (min > 0) desc = t('promoMascot.firstTopup.descMin', { min: money(min) })
    else if (cap > 0) desc = t('promoMascot.firstTopup.descCap', { cap: money(cap) })
    list.push({
      key: 'first_topup',
      icon: 'dollar',
      title: t('promoMascot.firstTopup.title'),
      desc,
      value: `${pct(topupPct)}%`,
      to: authed ? '/purchase' : '/register',
      claimed: authed && firstTopupEligible.value === false,
      tone: TONES.orange
    })
  }

  const bonus = Number(cfg.invitee_bonus_rate_percent || 0)
  const rebate = Number(cfg.inviter_rebate_rate_percent || 0)
  if (cfg.affiliate_enabled && (bonus > 0 || rebate > 0)) {
    let desc =
      bonus > 0 && rebate > 0
        ? t('promoMascot.invite.descBoth', { bonus: pct(bonus), rebate: pct(rebate) })
        : rebate > 0
          ? t('promoMascot.invite.descRebate', { rebate: pct(rebate) })
          : t('promoMascot.invite.descBonus', { bonus: pct(bonus) })
    if (rebate > 0 && cfg.withdraw_enabled) desc += t('promoMascot.invite.descWithdraw')
    list.push({
      key: 'invite',
      icon: 'users',
      title: t('promoMascot.invite.title'),
      desc,
      value: t('promoMascot.invite.value'),
      to: authed ? '/affiliate' : '/register',
      claimed: false,
      tone: TONES.violet
    })
  }

  const eduRate = Number(cfg.edu_discount_percent || 0)
  if (cfg.edu_verify_enabled && eduRate > 0) {
    list.push({
      key: 'edu',
      icon: 'book',
      title: t('promoMascot.edu.title'),
      desc: t('promoMascot.edu.desc'),
      value: t('promoMascot.edu.value', { rate: pct(eduRate) }),
      to: authed ? '/profile' : '/register',
      claimed: authed && eduActive.value,
      tone: TONES.blue
    })
  }

  if (cfg.price_lock_enabled) {
    const days = Number(cfg.price_lock_grace_days || 0)
    list.push({
      key: 'price_lock',
      icon: 'lock',
      title: t('promoMascot.priceLock.title'),
      desc: days > 0 ? t('promoMascot.priceLock.desc', { days }) : t('promoMascot.priceLock.descNoGrace'),
      value: t('promoMascot.priceLock.value'),
      to: '/pricing',
      claimed: false,
      tone: TONES.pink
    })
  }

  // Stable sort: offers the user can still take come first.
  return list.sort((a, b) => Number(a.claimed) - Number(b.claimed))
})

const unclaimedCount = computed(() => offers.value.filter((o) => !o.claimed).length)

const primaryCta = computed(() =>
  isAuthenticated.value
    ? { to: '/purchase', label: t('promoMascot.ctaTopup') }
    : { to: '/register', label: t('promoMascot.ctaRegister') }
)

async function loadCheckoutState() {
  if (checkoutLoaded || !isAuthenticated.value) return
  checkoutLoaded = true
  try {
    const { data } = await paymentAPI.getCheckoutInfo()
    if (typeof data.first_topup_eligible === 'boolean') firstTopupEligible.value = data.first_topup_eligible
    eduActive.value = data.edu_discount_active === true
  } catch {
    // Payment disabled or unavailable: show the offers without claimed state.
  }
}

function show() {
  clearTimeout(closeTimer)
  if (open.value) return
  open.value = true
  track('promo_mascot_open')
  void loadCheckoutState()
  if (!bubbleSeen.value) {
    bubbleSeen.value = true
    localStorage.setItem(BUBBLE_SEEN_KEY, new Date().toDateString())
  }
}

function hide() {
  clearTimeout(closeTimer)
  open.value = false
}

// Hover only on devices that really hover; touch taps go through toggle().
const canHover = window.matchMedia?.('(hover: hover)').matches ?? true
function onEnter() {
  if (canHover) show()
}
function onLeave() {
  if (!canHover) return
  clearTimeout(closeTimer)
  closeTimer = setTimeout(hide, 250)
}
function toggle() {
  if (open.value && !canHover) hide()
  else show()
}

function onDocumentPointer(e: PointerEvent) {
  if (open.value && rootEl.value && !rootEl.value.contains(e.target as Node)) hide()
}

onMounted(async () => {
  document.addEventListener('pointerdown', onDocumentPointer)
  try {
    config.value = await growthAPI.getPublicConfig()
  } catch {
    // Without the growth config only the sign-up gift (from public settings) shows.
  }
})

onBeforeUnmount(() => {
  clearTimeout(closeTimer)
  document.removeEventListener('pointerdown', onDocumentPointer)
})
</script>

<style scoped>
.mascot {
  transform: translateX(18px);
  transition: transform 0.3s ease;
}
.mascot-open {
  transform: translateX(0) rotate(-8deg);
}
.promo-card {
  transition: transform 0.15s ease;
}
.promo-card:hover {
  transform: translateX(-4px);
}
@media (prefers-reduced-motion: no-preference) {
  .mascot-bob {
    animation: mascot-bob 2.4s ease-in-out infinite;
  }
}
@keyframes mascot-bob {
  50% {
    transform: translateY(-6px);
  }
}
.promo-panel-enter-active {
  transition: opacity 0.2s ease, transform 0.35s cubic-bezier(0.3, 1.3, 0.5, 1);
}
.promo-panel-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.promo-panel-enter-from,
.promo-panel-leave-to {
  opacity: 0;
  transform: translateX(16px) scale(0.96);
}
</style>
