<template>
  <div v-if="status && status.enabled" class="card p-6" data-testid="affiliate-withdraw">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h3 class="flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
          <Icon name="creditCard" size="md" class="text-emerald-500" />
          {{ t('growth.withdraw.title') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.withdraw.description') }}</p>
        <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">{{ rulesText }}</p>
      </div>
      <div class="shrink-0 text-left sm:text-right">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('growth.withdraw.withdrawable') }}</p>
        <p class="text-2xl font-semibold text-emerald-600 dark:text-emerald-400" data-testid="withdrawable-cny">¥{{ money(status.withdrawable_cny) }}</p>
        <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
          {{ t('growth.withdraw.cashTotal') }} ¥{{ money(status.cash_cny) }} · {{ t('growth.withdraw.withdrawnTotal') }} ¥{{ money(status.withdrawn_cny) }}
        </p>
      </div>
    </div>

    <details class="mt-3 text-xs text-gray-500 dark:text-dark-400">
      <summary class="cursor-pointer select-none text-primary-600 dark:text-primary-400">{{ t('growth.withdraw.whyLess') }}</summary>
      <p class="mt-1 leading-relaxed">
        {{ status.freeze_hours > 0 ? t('growth.withdraw.whyLessAnswer', { hours: status.freeze_hours }) : t('growth.withdraw.whyLessNoFreeze') }}
      </p>
    </details>

    <p v-if="status.has_pending" class="mt-4 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
      {{ t('growth.withdraw.pendingHint') }}
    </p>
    <p v-else-if="limitReached" class="mt-4 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
      {{ t('growth.withdraw.limitHint') }}
    </p>
    <p v-else-if="status.withdrawable_cny < status.min_cny" class="mt-4 rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-dark-300">
      {{ t('growth.withdraw.notEnough', { min: money(status.min_cny) }) }}
    </p>

    <form v-else class="mt-5 space-y-4" data-testid="withdraw-form" @submit.prevent="askConfirm">
      <div>
        <p class="input-label">{{ t('growth.withdraw.method') }}</p>
        <div class="flex gap-2">
          <button
            v-for="m in methods"
            :key="m"
            type="button"
            :class="['btn btn-sm', form.method === m ? 'btn-primary' : 'btn-secondary']"
            :data-testid="`withdraw-method-${m}`"
            @click="form.method = m"
          >
            {{ t(`growth.withdraw.methods.${m}`) }}
          </button>
        </div>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label" for="withdraw-account">{{ t('growth.withdraw.account') }}</label>
          <input id="withdraw-account" v-model.trim="form.account" type="text" maxlength="100" class="input" autocomplete="off" :placeholder="t(`growth.withdraw.accountPlaceholder.${form.method}`)" />
        </div>
        <div>
          <label class="input-label" for="withdraw-name">{{ t('growth.withdraw.realName') }}</label>
          <input id="withdraw-name" v-model.trim="form.realName" type="text" maxlength="50" class="input" autocomplete="off" :placeholder="t('growth.withdraw.realNamePlaceholder')" />
        </div>
        <div>
          <label class="input-label" for="withdraw-amount">{{ t('growth.withdraw.amount') }}</label>
          <div class="flex gap-2">
            <input id="withdraw-amount" v-model.number="form.amount" type="number" :min="status.min_cny" :max="status.withdrawable_cny" step="0.01" class="input" data-testid="withdraw-amount" />
            <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="form.amount = status.withdrawable_cny">{{ t('growth.withdraw.all') }}</button>
          </div>
        </div>
        <div>
          <label class="input-label" for="withdraw-note">{{ t('growth.withdraw.note') }}</label>
          <input id="withdraw-note" v-model.trim="form.note" type="text" maxlength="200" class="input" :placeholder="t('growth.withdraw.notePlaceholder')" />
        </div>
      </div>
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs text-gray-400 dark:text-dark-500">🔒 {{ t('growth.withdraw.privacy') }}</p>
        <button type="submit" class="btn btn-primary" :disabled="!canSubmit || submitting" data-testid="withdraw-submit">
          {{ submitting ? t('growth.withdraw.submitting') : t('growth.withdraw.submit') }}
        </button>
      </div>
    </form>

    <div class="mt-6">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('growth.withdraw.history') }}</h4>
      <p v-if="status.withdrawals.length === 0" class="mt-2 text-sm text-gray-400 dark:text-dark-500">{{ t('growth.withdraw.empty') }}</p>
      <div v-else class="mt-2 overflow-x-auto">
        <table class="w-full min-w-[560px] text-left text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
              <th class="px-3 py-2 font-medium">{{ t('growth.withdraw.columns.time') }}</th>
              <th class="px-3 py-2 text-right font-medium">{{ t('growth.withdraw.columns.amount') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('growth.withdraw.columns.payee') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('growth.withdraw.columns.status') }}</th>
              <th class="px-3 py-2 font-medium">{{ t('growth.withdraw.columns.note') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="w in status.withdrawals" :key="w.id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
              <td class="px-3 py-2.5 text-gray-700 dark:text-gray-300">{{ formatDateTime(w.created_at) }}</td>
              <td class="px-3 py-2.5 text-right font-medium text-gray-900 dark:text-white">¥{{ money(w.cny_amount) }}</td>
              <td class="px-3 py-2.5 text-gray-700 dark:text-gray-300">{{ t(`growth.withdraw.methods.${w.method}`) }} {{ maskAccount(w.account) }}</td>
              <td class="px-3 py-2.5">
                <span :class="['rounded px-1.5 py-0.5 text-xs', statusClass(w.status)]">{{ t(`growth.withdraw.status.${w.status}`) }}</span>
              </td>
              <td class="px-3 py-2.5 text-xs text-gray-500 dark:text-dark-400">
                <button v-if="w.status === 'pending'" type="button" class="btn btn-secondary btn-sm" :data-testid="`withdraw-cancel-${w.id}`" @click="cancelTarget = w">
                  {{ t('growth.withdraw.cancel') }}
                </button>
                <span v-else-if="w.status === 'rejected'">{{ t('growth.withdraw.rejectedNote', { note: w.admin_note }) }}</span>
                <span v-else-if="w.status === 'paid'">{{ w.admin_note }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <ConfirmDialog
      :show="confirming"
      :title="t('growth.withdraw.submit')"
      :message="confirmMessage"
      @confirm="submit"
      @cancel="confirming = false"
    />
    <ConfirmDialog
      :show="cancelTarget !== null"
      :title="t('growth.withdraw.cancel')"
      :message="t('growth.withdraw.cancelConfirm')"
      danger
      @confirm="cancelWithdraw"
      @cancel="cancelTarget = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { growthAPI, type AffiliateWithdrawal, type WithdrawMethod, type WithdrawStatus, type WithdrawStatusValue } from '@/api/growth'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { track } from '@/utils/analytics'

const emit = defineEmits<{
  /** The rebate balance changed (request or cancel): the page should reload its numbers. */
  (e: 'changed'): void
  (e: 'loaded', status: WithdrawStatus): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const methods: WithdrawMethod[] = ['alipay', 'wechat']
const status = ref<WithdrawStatus | null>(null)
const submitting = ref(false)
const confirming = ref(false)
const cancelTarget = ref<AffiliateWithdrawal | null>(null)
const form = reactive({ method: 'alipay' as WithdrawMethod, account: '', realName: '', amount: 0, note: '' })

function money(v: number): string {
  return (Math.floor((v || 0) * 100 + 1e-6) / 100).toFixed(2)
}

function maskAccount(v: string): string {
  if (!v) return ''
  if (v.length <= 4) return v
  return `${v.slice(0, 3)}****${v.slice(-2)}`
}

function statusClass(s: WithdrawStatusValue): string {
  switch (s) {
    case 'paid':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'rejected':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

const limitReached = computed(() => !!status.value && status.value.monthly_limit > 0 && status.value.month_used >= status.value.monthly_limit)

const rulesText = computed(() => {
  if (!status.value) return ''
  const s = status.value
  const limit = s.monthly_limit > 0 ? t('growth.withdraw.rulesLimit', { n: s.monthly_limit }) : t('growth.withdraw.rulesNoLimit')
  return t('growth.withdraw.rules', { min: money(s.min_cny), limit })
})

const canSubmit = computed(() => {
  const s = status.value
  if (!s) return false
  const amount = Number(form.amount)
  return !!form.account && !!form.realName && amount >= s.min_cny && amount <= s.withdrawable_cny + 1e-9
})

const confirmMessage = computed(() =>
  t('growth.withdraw.confirm', {
    amount: money(Number(form.amount)),
    method: t(`growth.withdraw.methods.${form.method}`),
    account: form.account,
    name: form.realName,
  }),
)

async function load(): Promise<void> {
  try {
    const s = await growthAPI.getWithdrawStatus()
    status.value = s
    if (s.last_method) form.method = s.last_method
    if (!form.account && s.last_account) form.account = s.last_account
    if (!form.realName && s.last_real_name) form.realName = s.last_real_name
    form.amount = s.withdrawable_cny
    emit('loaded', s)
  } catch (error) {
    // The box simply stays hidden; the rest of the page still works.
    status.value = null
    if (import.meta.env.DEV) console.warn(extractApiErrorMessage(error, t('growth.withdraw.loadFailed')))
  }
}

function askConfirm(): void {
  if (canSubmit.value) confirming.value = true
}

async function submit(): Promise<void> {
  confirming.value = false
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    await growthAPI.requestWithdraw({
      cny_amount: Number(form.amount),
      method: form.method,
      account: form.account,
      real_name: form.realName,
      note: form.note || undefined,
    })
    track('withdraw_request', { amount: Number(form.amount), method: form.method })
    appStore.showSuccess(t('growth.withdraw.submitted'))
    form.note = ''
    await load()
    emit('changed')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('growth.withdraw.loadFailed')))
    await load()
  } finally {
    submitting.value = false
  }
}

async function cancelWithdraw(): Promise<void> {
  const target = cancelTarget.value
  cancelTarget.value = null
  if (!target) return
  try {
    await growthAPI.cancelWithdraw(target.id)
    appStore.showSuccess(t('growth.withdraw.cancelled'))
    await load()
    emit('changed')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('growth.withdraw.loadFailed')))
    await load()
  }
}

defineExpose({ reload: load })

onMounted(load)
</script>
