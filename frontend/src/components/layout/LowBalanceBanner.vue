<template>
  <div
    v-if="visible"
    class="mx-4 mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 md:mx-6 lg:mx-8 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-100"
    data-testid="low-balance-banner"
  >
    <span>
      <b>{{ firstTopup ? t('lowBalance.trialTitle') : t('lowBalance.title') }}</b>
      <span class="ml-1">{{ firstTopup ? t('lowBalance.trialDesc', { percent: firstTopup }) : t('lowBalance.desc', { balance: balanceText }) }}</span>
    </span>
    <span class="flex items-center gap-3">
      <router-link to="/purchase" class="btn btn-primary btn-sm" data-testid="low-balance-cta" @click="track('cta_click', { where: firstTopup ? 'low-balance-first-topup' : 'low-balance' })">
        {{ firstTopup ? t('lowBalance.trialCta', { percent: firstTopup }) : t('lowBalance.cta') }}
      </router-link>
      <button type="button" class="text-xs text-amber-700/70 hover:text-amber-900 dark:text-amber-200/70" data-testid="low-balance-dismiss" @click="dismiss">
        {{ t('lowBalance.later') }}
      </button>
    </span>
  </div>
</template>

<script setup lang="ts">
// Shown on every user page when the balance is about to run out (and the user has no subscription):
// the moment the trial credit is used up is when people are most ready to pay. Users who signed up
// with WeChat have no email, so this is the only nudge that reaches them. Hidden for a day when dismissed.
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { paymentAPI } from '@/api/payment'
import { growthAPI } from '@/api/growth'
import { track } from '@/utils/analytics'

const LOW_BALANCE = 0.5
const DISMISS_KEY = 'hg_low_balance_dismissed'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptions = useSubscriptionStore()

const dismissedToday = ref(localStorage.getItem(DISMISS_KEY) === new Date().toDateString())
const firstTopup = ref(0) // first top-up bonus % when the user still qualifies, else 0
let checkedOffer = false

const balance = computed(() => authStore.user?.balance ?? 0)
const balanceText = computed(() => balance.value.toFixed(2))
const low = computed(
  () =>
    authStore.isAuthenticated &&
    authStore.user?.role !== 'admin' &&
    appStore.cachedPublicSettings?.payment_enabled === true &&
    balance.value < LOW_BALANCE
)
const subsChecked = ref(false)
const visible = computed(() => low.value && !dismissedToday.value && subsChecked.value && !subscriptions.hasActiveSubscriptions)

async function checkOffer(): Promise<void> {
  if (checkedOffer || !low.value) return
  checkedOffer = true
  subscriptions.fetchActiveSubscriptions().then(() => { subsChecked.value = true }).catch(() => {})
  try {
    const cfg = await growthAPI.getPublicConfig()
    if (!cfg.first_topup_bonus_percent) return
    const info = await paymentAPI.getCheckoutInfo()
    if (info.data.first_topup_eligible) firstTopup.value = cfg.first_topup_bonus_percent
  } catch {
    // the plain low-balance banner still shows
  }
}

function dismiss(): void {
  dismissedToday.value = true
  localStorage.setItem(DISMISS_KEY, new Date().toDateString())
}

onMounted(checkOffer)
watch(low, checkOffer)
</script>
