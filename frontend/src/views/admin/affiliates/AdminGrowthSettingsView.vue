<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6" data-testid="admin-growth-settings">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.settings.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.settings.description') }}</p>
      </div>

      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <form v-else-if="form" class="space-y-6" @submit.prevent="save">
        <p v-if="!affiliateEnabled" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('growth.admin.settings.affiliateOff') }}
        </p>

        <section class="card space-y-4 p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🎁 {{ t('growth.admin.settings.inviteeBonusTitle') }}</h2>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label" for="growth-bonus-rate">{{ t('growth.admin.settings.inviteeBonusRate') }}</label>
              <input id="growth-bonus-rate" v-model.number="form.invitee_bonus_rate_percent" type="number" min="0" max="100" step="0.1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.inviteeBonusRateHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-bonus-cap">{{ t('growth.admin.settings.inviteeBonusCap') }}</label>
              <input id="growth-bonus-cap" v-model.number="form.invitee_bonus_cap" type="number" min="0" step="0.01" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.inviteeBonusCapHint') }}</p>
            </div>
          </div>
        </section>

        <section class="card space-y-4 p-6" data-testid="growth-first-topup">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">💰 {{ t('growth.admin.settings.firstTopupTitle') }}</h2>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.settings.firstTopupDesc') }}</p>
          <div class="grid gap-4 sm:grid-cols-3">
            <div>
              <label class="input-label" for="growth-topup-rate">{{ t('growth.admin.settings.firstTopupRate') }}</label>
              <input id="growth-topup-rate" v-model.number="form.first_topup_bonus_percent" type="number" min="0" max="100" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.firstTopupRateHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-topup-cap">{{ t('growth.admin.settings.firstTopupCap') }}</label>
              <input id="growth-topup-cap" v-model.number="form.first_topup_bonus_cap" type="number" min="0" step="0.01" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.firstTopupCapHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-topup-min">{{ t('growth.admin.settings.firstTopupMin') }}</label>
              <input id="growth-topup-min" v-model.number="form.first_topup_min_amount" type="number" min="0" step="0.01" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.firstTopupMinHint') }}</p>
            </div>
          </div>
        </section>

        <section class="card space-y-4 p-6" data-testid="growth-signup-bonus">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🎟️ {{ t('growth.admin.settings.signupBonusTitle') }}</h2>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label" for="growth-signup-bonus">{{ t('growth.admin.settings.signupBonus') }}</label>
              <input id="growth-signup-bonus" v-model.number="form.invitee_signup_bonus" type="number" min="0" max="5" step="0.1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.signupBonusHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-signup-limit">{{ t('growth.admin.settings.signupDailyLimit') }}</label>
              <input id="growth-signup-limit" v-model.number="form.invitee_signup_daily_limit" type="number" min="1" max="1000" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.signupDailyLimitHint') }}</p>
            </div>
          </div>
        </section>

        <section class="card space-y-4 p-6" data-testid="growth-withdraw">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">💸 {{ t('growth.admin.settings.withdrawTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.withdraw_enabled" data-testid="withdraw-enabled-toggle" />
            {{ t('growth.admin.settings.withdrawEnabled') }}
          </label>
          <p class="text-xs leading-relaxed text-gray-400">{{ t('growth.admin.settings.withdrawHint') }}</p>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label" for="growth-withdraw-min">{{ t('growth.admin.settings.withdrawMin') }}</label>
              <input id="growth-withdraw-min" v-model.number="form.withdraw_min_cny" type="number" min="1" max="100000" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.withdrawMinHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-withdraw-monthly">{{ t('growth.admin.settings.withdrawMonthly') }}</label>
              <input id="growth-withdraw-monthly" v-model.number="form.withdraw_monthly_limit" type="number" min="0" max="31" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.withdrawMonthlyHint') }}</p>
            </div>
          </div>
          <router-link to="/admin/affiliates/withdrawals" class="inline-block text-sm text-primary-600 hover:underline dark:text-primary-400">{{ t('growth.admin.nav.withdrawals') }} →</router-link>
        </section>

        <section class="card space-y-3 p-6" data-testid="growth-smart-billing">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🔀 {{ t('growth.admin.settings.smartBillingTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle :model-value="smartBilling" :disabled="smartBillingSaving" @update:model-value="toggleSmartBilling" />
            {{ t('growth.admin.settings.smartBillingEnabled') }}
          </label>
          <p class="text-xs leading-relaxed text-gray-400">{{ t('growth.admin.settings.smartBillingHint') }}</p>
        </section>

        <section class="card space-y-4 p-6" data-testid="growth-price-lock">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🔒 {{ t('growth.admin.settings.priceLockTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.price_lock_enabled" />
            {{ t('growth.admin.settings.priceLockEnabled') }}
          </label>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label" for="growth-lock-grace">{{ t('growth.admin.settings.priceLockGrace') }}</label>
              <input id="growth-lock-grace" v-model.number="form.price_lock_grace_days" type="number" min="0" max="365" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.priceLockGraceHint') }}</p>
            </div>
          </div>
          <p class="text-xs text-gray-500">{{ t('growth.admin.settings.priceLockHint') }}</p>
          <table v-if="priceLocks.length" class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs text-gray-500">
                <th class="py-1">{{ t('growth.admin.settings.priceLockCol.plan') }}</th>
                <th class="text-right">{{ t('growth.admin.settings.priceLockCol.price') }}</th>
                <th class="text-right">{{ t('growth.admin.settings.priceLockCol.locked') }}</th>
                <th class="text-right">{{ t('growth.admin.settings.priceLockCol.range') }}</th>
                <th class="text-right">{{ t('growth.admin.settings.priceLockCol.below') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="l in priceLocks" :key="l.plan_id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="py-1.5">{{ l.plan_name }}</td>
                <td class="text-right tabular-nums">¥{{ l.price }}</td>
                <td class="text-right tabular-nums">{{ l.locked }}</td>
                <td class="text-right tabular-nums">{{ l.locked ? (l.min_locked === l.max_locked ? `¥${l.min_locked}` : `¥${l.min_locked}–${l.max_locked}`) : '—' }}</td>
                <td class="text-right tabular-nums">{{ l.below }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section class="card space-y-3 p-6" data-testid="growth-abandoned-order">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">📬 {{ t('growth.admin.settings.abandonedTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.abandoned_order_reminder" />
            {{ t('growth.admin.settings.abandonedEnabled') }}
          </label>
          <p class="text-xs text-gray-400">{{ t('growth.admin.settings.abandonedHint') }}</p>
        </section>

        <section class="card space-y-3 p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🏆 {{ t('growth.admin.settings.leaderboardTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.leaderboard_enabled" />
            {{ t('growth.admin.settings.leaderboardEnabled') }}
          </label>
          <p class="text-xs text-gray-400">{{ t('growth.admin.settings.leaderboardHint') }}</p>
        </section>

        <section class="card space-y-4 p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">🎓 {{ t('growth.admin.settings.eduTitle') }}</h2>
          <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.edu_verify_enabled" data-testid="edu-enabled-toggle" />
            {{ t('growth.admin.settings.eduEnabled') }}
          </label>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label" for="growth-edu-suffixes">{{ t('growth.admin.settings.eduSuffixes') }}</label>
              <textarea id="growth-edu-suffixes" v-model="suffixText" rows="4" class="input font-mono text-sm"></textarea>
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.eduSuffixesHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="growth-edu-discount">{{ t('growth.admin.settings.eduDiscount') }}</label>
              <input id="growth-edu-discount" v-model.number="form.edu_discount_percent" type="number" min="0" max="90" step="1" class="input" />
              <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.eduDiscountHint') }}</p>
            </div>
          </div>
        </section>

        <div class="flex justify-end">
          <button type="submit" class="btn btn-primary" :disabled="saving" data-testid="growth-save">{{ t('growth.admin.settings.save') }}</button>
        </div>
      </form>

      <section class="card p-6">
        <form class="mb-6 rounded-xl border border-gray-200 p-4 dark:border-dark-700" data-testid="edu-grant-form" @submit.prevent="grant">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.settings.grantTitle') }}</h3>
          <p class="mt-1 text-xs text-gray-400">{{ t('growth.admin.settings.grantHint') }}</p>
          <div class="mt-3 grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_auto]">
            <input v-model.trim="grantUser" type="text" class="input" data-testid="edu-grant-user" :placeholder="t('growth.admin.settings.grantUser')" />
            <input v-model.trim="grantNote" type="text" maxlength="100" class="input" data-testid="edu-grant-note" :placeholder="t('growth.admin.settings.grantNote')" />
            <button type="submit" class="btn btn-primary" :disabled="!grantUser || !grantNote || granting" data-testid="edu-grant-submit">
              {{ t('growth.admin.settings.grant') }}
            </button>
          </div>
        </form>
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.settings.verificationsTitle') }} ({{ total }})</h2>
          <input v-model="search" type="text" class="input w-full sm:w-72" :placeholder="t('growth.admin.settings.searchPlaceholder')" @input="debouncedLoad" />
        </div>
        <div v-if="verifications.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          {{ t('growth.admin.settings.empty') }}
        </div>
        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[560px] text-left text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.settings.columns.user') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.settings.columns.eduEmail') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.settings.columns.verifiedAt') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.settings.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="v in verifications" :key="v.user_id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                <td class="px-3 py-2.5">
                  <div class="text-gray-900 dark:text-white">{{ v.user_email }}</div>
                  <div class="text-xs text-gray-400">#{{ v.user_id }} {{ v.username }}</div>
                </td>
                <td class="px-3 py-2.5">
                  <template v-if="v.method === 'manual'">
                    <span class="rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">{{ t('growth.admin.settings.manual') }}</span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ v.note }}</div>
                  </template>
                  <span v-else class="font-mono text-gray-700 dark:text-gray-300">{{ v.email }}</span>
                </td>
                <td class="px-3 py-2.5 text-gray-500 dark:text-dark-400">{{ formatDateTime(v.verified_at) }}</td>
                <td class="px-3 py-2.5 text-right">
                  <button type="button" class="btn btn-secondary btn-sm" @click="revokeTarget = v">{{ t('growth.admin.settings.revoke') }}</button>
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
      </section>
    </div>

    <ConfirmDialog
      :show="revokeTarget !== null"
      :title="t('growth.admin.settings.revoke')"
      :message="t('growth.admin.settings.revokeConfirm', { email: revokeTarget?.email || '' })"
      danger
      @confirm="revoke"
      @cancel="revokeTarget = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { adminGrowthAPI, growthAPI, type EduVerification, type GrowthSettings, type PlanPriceLockStat } from '@/api/growth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)
const form = ref<GrowthSettings | null>(null)
const suffixText = ref('')
const affiliateEnabled = ref(true)
const priceLocks = ref<PlanPriceLockStat[]>([])
const smartBilling = ref(true)
const smartBillingSaving = ref(false)

// Saved at once (its own switch, not part of the form below).
async function toggleSmartBilling(on: boolean) {
  smartBillingSaving.value = true
  try {
    smartBilling.value = (await adminGrowthAPI.setSmartBilling(on)).enabled
    appStore.showSuccess(t('growth.admin.settings.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    smartBillingSaving.value = false
  }
}

const verifications = ref<EduVerification[]>([])
const total = ref(0)
const page = ref(1)
const pages = ref(1)
const search = ref('')
const revokeTarget = ref<EduVerification | null>(null)
const grantUser = ref('')
const grantNote = ref('')
const granting = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | undefined

async function loadSettings() {
  loading.value = true
  try {
    const [settings, publicConfig] = await Promise.all([adminGrowthAPI.getSettings(), growthAPI.getPublicConfig()])
    form.value = settings
    suffixText.value = settings.edu_email_suffixes.join('\n')
    affiliateEnabled.value = publicConfig.affiliate_enabled
    adminGrowthAPI.priceLocks().then((v) => { priceLocks.value = v }).catch(() => {})
    adminGrowthAPI.getSmartBilling().then((v) => { smartBilling.value = v.enabled }).catch(() => {})
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.value || saving.value) return
  saving.value = true
  try {
    const payload: GrowthSettings = {
      ...form.value,
      invitee_bonus_rate_percent: Number(form.value.invitee_bonus_rate_percent) || 0,
      invitee_bonus_cap: Number(form.value.invitee_bonus_cap) || 0,
      invitee_signup_bonus: Number(form.value.invitee_signup_bonus) || 0,
      first_topup_bonus_percent: Number(form.value.first_topup_bonus_percent) || 0,
      first_topup_bonus_cap: Number(form.value.first_topup_bonus_cap) || 0,
      first_topup_min_amount: Number(form.value.first_topup_min_amount) || 0,
      invitee_signup_daily_limit: Number(form.value.invitee_signup_daily_limit) || 20,
      price_lock_grace_days: Math.max(0, Math.round(Number(form.value.price_lock_grace_days) || 0)),
      edu_discount_percent: Number(form.value.edu_discount_percent) || 0,
      withdraw_min_cny: Number(form.value.withdraw_min_cny) || 50,
      withdraw_monthly_limit: Math.max(0, Math.floor(Number(form.value.withdraw_monthly_limit) || 0)),
      edu_email_suffixes: suffixText.value.split(/[\s,，、]+/).map((s) => s.trim()).filter(Boolean),
    }
    form.value = await adminGrowthAPI.updateSettings(payload)
    suffixText.value = form.value.edu_email_suffixes.join('\n')
    appStore.showSuccess(t('growth.admin.settings.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function loadVerifications() {
  try {
    const res = await adminGrowthAPI.listEduVerifications({ page: page.value, page_size: 20, search: search.value || undefined })
    verifications.value = res.items
    total.value = res.total
    pages.value = Math.max(1, res.pages)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

function debouncedLoad() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    void loadVerifications()
  }, 300)
}

function goPage(p: number) {
  page.value = p
  void loadVerifications()
}

async function grant() {
  if (!grantUser.value || !grantNote.value || granting.value) return
  granting.value = true
  try {
    const v = await adminGrowthAPI.grantEduVerification(grantUser.value, grantNote.value)
    appStore.showSuccess(t('growth.admin.settings.granted', { email: v.email }))
    grantUser.value = ''
    grantNote.value = ''
    page.value = 1
    await loadVerifications()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    granting.value = false
  }
}

async function revoke() {
  const target = revokeTarget.value
  revokeTarget.value = null
  if (!target) return
  try {
    await adminGrowthAPI.revokeEduVerification(target.user_id)
    appStore.showSuccess(t('growth.admin.settings.revoked'))
    await loadVerifications()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(() => {
  void loadSettings()
  void loadVerifications()
})
</script>
