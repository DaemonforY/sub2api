<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <!-- Signed up without a friend's code: add it within 7 days -->
        <div v-if="detail.can_bind_inviter" class="card border border-emerald-200 p-5 dark:border-emerald-900/50" data-testid="affiliate-bind-inviter">
          <h3 class="flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
            <Icon name="gift" size="md" class="text-emerald-500" />
            {{ t('affiliate.bind.title') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('affiliate.bind.description', { deadline: detail.bind_inviter_deadline ? formatDateTime(detail.bind_inviter_deadline) : '' }) }}
          </p>
          <div class="mt-3 flex flex-col gap-2 sm:flex-row">
            <input
              v-model="bindCode"
              type="text"
              class="input font-mono sm:max-w-xs"
              :placeholder="t('affiliate.bind.placeholder')"
              data-testid="affiliate-bind-input"
              @keyup.enter="bindInviter"
            />
            <button class="btn btn-primary" :disabled="!bindCode.trim() || binding" data-testid="affiliate-bind-submit" @click="bindInviter">
              {{ binding ? t('affiliate.bind.binding') : t('affiliate.bind.submit') }}
            </button>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
              <li v-if="inviteeSignupBonus > 0" data-testid="invitee-trial-tip">🎁 {{ t('growth.inviteeBonus.affiliateTrialTip', { amount: inviteeSignupBonus }) }}</li>
              <li v-if="inviteeBonusRate > 0" data-testid="invitee-bonus-tip">🎁 {{ t('growth.inviteeBonus.affiliateTip', { rate: inviteeBonusRate }) }}</li>
              <li>💡 {{ t('growth.inviteeBonus.balanceForSubscription') }}</li>
            </ul>
          </div>
        </div>

        <InvitePosterCard
          :invite-link="inviteLink"
          :aff-code="detail.aff_code"
          :rebate-rate="detail.effective_rebate_rate_percent ?? 0"
          :invitee-bonus-rate="inviteeBonusRate"
          :invitee-bonus-cap="inviteeBonusCap"
          :invitee-signup-bonus="inviteeSignupBonus"
          :edu-discount="eduDiscount"
        />

        <WithdrawCard ref="withdrawCard" @changed="loadAffiliateDetail(true)" @loaded="(s) => (withdrawableCny = s.enabled ? s.withdrawable_cny : 0)" />

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="requestTransfer"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <ConfirmDialog
          :show="confirmTransfer"
          :title="t('affiliate.transfer.button')"
          :message="t('growth.withdraw.transferWarn', { amount: withdrawableCny.toFixed(2) })"
          danger
          @confirm="confirmTransfer = false; transferQuota()"
          @cancel="confirmTransfer = false"
        />

        <InviteLeaderboardCard />

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { track } from '@/utils/analytics'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import InviteLeaderboardCard from '@/components/user/growth/InviteLeaderboardCard.vue'
import InvitePosterCard from '@/components/user/growth/InvitePosterCard.vue'
import WithdrawCard from '@/components/user/growth/WithdrawCard.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { growthAPI } from '@/api/growth'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const bindCode = ref('')
const binding = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const withdrawCard = ref<InstanceType<typeof WithdrawCard> | null>(null)
// Cash the user could still withdraw; moving rebates to balance gives that up, so ask first.
const withdrawableCny = ref(0)
const confirmTransfer = ref(false)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function bindInviter(): Promise<void> {
  const code = bindCode.value.trim()
  if (!code || binding.value) return
  binding.value = true
  try {
    detail.value = await userAPI.bindAffiliateInviter(code)
    bindCode.value = ''
    appStore.showSuccess(t('affiliate.bind.success'))
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'affiliate.bind.errors', t('affiliate.bind.failed')))
  } finally {
    binding.value = false
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  track('invite_link_copy')
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

function requestTransfer(): void {
  if (withdrawableCny.value > 0) {
    confirmTransfer.value = true
    return
  }
  void transferQuota()
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
      withdrawCard.value?.reload(),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

const inviteeBonusRate = ref(0)
const inviteeBonusCap = ref(0)
const inviteeSignupBonus = ref(0)
const eduDiscount = ref(0)

onMounted(() => {
  void loadAffiliateDetail()
  growthAPI.getPublicConfig().then((cfg) => {
    inviteeBonusRate.value = cfg.affiliate_enabled ? cfg.invitee_bonus_rate_percent : 0
    inviteeBonusCap.value = cfg.affiliate_enabled ? cfg.invitee_bonus_cap : 0
    inviteeSignupBonus.value = cfg.affiliate_enabled ? cfg.invitee_signup_bonus || 0 : 0
    eduDiscount.value = cfg.edu_verify_enabled ? cfg.edu_discount_percent : 0
  }).catch(() => {})
})
</script>
