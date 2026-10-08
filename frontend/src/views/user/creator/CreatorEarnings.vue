<template>
  <div class="space-y-4" data-testid="creator-earnings">
    <!-- Withdraw -->
    <div class="card space-y-3 p-5">
      <div>
        <h3 class="text-sm font-medium">{{ t('creator.withdraw.title') }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('creator.withdraw.hint', { min: home.withdraw_min_cny, days: home.settle_days }) }}</p>
      </div>
      <p v-if="pending" class="text-sm text-blue-600" data-testid="creator-withdraw-pending">{{ t('creator.withdraw.pending') }}</p>
      <p v-else-if="available < home.withdraw_min_cny" class="text-sm text-gray-500" data-testid="creator-withdraw-low">{{ t('creator.withdraw.belowMin', { min: home.withdraw_min_cny }) }}</p>
      <div v-else class="grid gap-3 md:grid-cols-2" data-testid="creator-withdraw-form">
        <label class="text-sm">
          <span class="input-label">{{ t('creator.withdraw.amount') }}</span>
          <input v-model.number="form.cny_amount" type="number" :min="home.withdraw_min_cny" :max="available" step="0.01" class="input" data-testid="creator-withdraw-amount" />
          <span class="input-hint">{{ t('creator.withdraw.max', { amount: available.toFixed(2) }) }}</span>
        </label>
        <div class="text-sm">
          <span class="input-label">{{ t('creator.withdraw.method') }}</span>
          <div class="flex gap-4 pt-2">
            <label class="flex items-center gap-1"><input v-model="form.method" type="radio" value="alipay" />{{ t('creator.withdraw.alipay') }}</label>
            <label class="flex items-center gap-1"><input v-model="form.method" type="radio" value="wechat" />{{ t('creator.withdraw.wechat') }}</label>
          </div>
        </div>
        <label class="text-sm">
          <span class="input-label">{{ t('creator.withdraw.account') }}</span>
          <input v-model="form.account" class="input" maxlength="100" data-testid="creator-withdraw-account" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('creator.withdraw.realName') }}</span>
          <input v-model="form.real_name" class="input" maxlength="50" data-testid="creator-withdraw-name" />
        </label>
        <label class="text-sm md:col-span-2">
          <span class="input-label">{{ t('creator.withdraw.note') }}</span>
          <input v-model="form.note" class="input" maxlength="200" />
        </label>
        <div class="md:col-span-2">
          <button class="btn btn-primary" :disabled="busy || !form.account || !form.real_name || !form.cny_amount" data-testid="creator-withdraw-submit" @click="withdraw">{{ t('creator.withdraw.submit') }}</button>
        </div>
      </div>

      <ul v-if="home.withdrawals.length" class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
        <li v-for="w in home.withdrawals" :key="w.id" class="flex flex-wrap items-center gap-2 py-2">
          <span class="font-medium">¥{{ w.cny_amount.toFixed(2) }}</span>
          <span :class="['badge', statusBadge(w.status)]">{{ t(`creator.withdraw.status.${w.status}`) }}</span>
          <span class="text-xs text-gray-500">{{ t(`creator.withdraw.${w.method}`) }} {{ w.account }}</span>
          <span class="text-xs text-gray-400">{{ formatDateTime(w.created_at) }}</span>
          <span v-if="w.admin_note" class="w-full text-xs text-gray-500">{{ t('creator.withdraw.adminNote') }}{{ w.admin_note }}</span>
          <button v-if="w.status === 'pending'" class="btn btn-secondary btn-sm ml-auto" :disabled="busy" @click="cancel(w.id)">{{ t('creator.withdraw.cancel') }}</button>
        </li>
      </ul>
    </div>

    <!-- Sales -->
    <div class="card p-5">
      <h3 class="mb-1 text-sm font-medium">{{ t('creator.sales.title') }}</h3>
      <p class="mb-3 text-xs text-gray-500 dark:text-dark-400">{{ t('creator.sales.hint') }}</p>
      <p v-if="!sales.length" class="py-6 text-center text-sm text-gray-500">{{ t('creator.sales.empty') }}</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full whitespace-nowrap text-sm" data-testid="creator-sales">
          <thead class="text-left text-xs text-gray-500">
            <tr>
              <th class="py-1 pr-3">{{ t('creator.sales.time') }}</th>
              <th class="py-1 pr-3">{{ t('creator.sales.course') }}</th>
              <th class="py-1 pr-3">{{ t('creator.buyer') }}</th>
              <th class="py-1 pr-3">{{ t('creator.sales.gross') }}</th>
              <th class="py-1 pr-3">{{ t('creator.sales.fee') }}</th>
              <th class="py-1 pr-3">{{ t('creator.sales.net') }}</th>
              <th class="py-1">{{ t('creator.sales.status') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="s in sales" :key="s.order_id">
              <td class="py-2 pr-3 text-xs text-gray-500">{{ formatDateTime(s.created_at) }}</td>
              <td class="max-w-[14rem] truncate py-2 pr-3">{{ s.course_title }}</td>
              <td class="py-2 pr-3 text-xs">{{ s.buyer }}</td>
              <td class="py-2 pr-3">¥{{ s.gross.toFixed(2) }}</td>
              <td class="py-2 pr-3 text-xs text-gray-500">{{ s.commission_percent }}%</td>
              <td class="py-2 pr-3 font-medium">¥{{ s.net.toFixed(2) }}</td>
              <td class="py-2 text-xs">
                {{ t(`creator.sales.statuses.${s.status}`) }}
                <span v-if="s.status === 'frozen'" class="text-gray-400">· {{ t('creator.sales.availableAt', { time: formatDateTime(s.available_at) }) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="total > sales.length || page > 1" class="mt-3 flex gap-2">
        <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="loadSales(page - 1)">{{ t('creator.sales.prev') }}</button>
        <button class="btn btn-secondary btn-sm" :disabled="page * 30 >= total" @click="loadSales(page + 1)">{{ t('creator.sales.next') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import creatorAPI, { type CreatorHome, type CreatorSale, type CreatorWithdrawInput } from '@/api/creator'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ home: CreatorHome }>()
const emit = defineEmits<{ changed: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const busy = ref(false)
const sales = ref<CreatorSale[]>([])
const total = ref(0)
const page = ref(1)

const available = computed(() => Math.max(0, props.home.balance?.available || 0))
const pending = computed(() => (props.home.balance?.pending || 0) > 0)
const form = reactive<CreatorWithdrawInput>({ cny_amount: 0, method: 'alipay', account: '', real_name: '', note: '' })

// Prefill the last payee and the whole available amount.
watch(
  () => props.home,
  (home) => {
    form.method = home.last_method || form.method
    form.account = form.account || home.last_account || ''
    form.real_name = form.real_name || home.last_real_name || ''
    form.cny_amount = Math.floor(available.value * 100) / 100
  },
  { immediate: true },
)

function fail(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

function formatDateTime(value: string) {
  return value ? new Date(value).toLocaleString() : ''
}

function statusBadge(status: string) {
  return status === 'paid' ? 'badge-success' : status === 'pending' ? 'badge-warning' : 'badge-gray'
}

async function loadSales(p: number) {
  try {
    const res = await creatorAPI.sales(p)
    sales.value = res.items
    total.value = res.total
    page.value = p
  } catch (err) {
    fail(err)
  }
}

async function withdraw() {
  busy.value = true
  try {
    await creatorAPI.withdraw({ ...form })
    form.note = ''
    appStore.showSuccess(t('creator.withdraw.requested'))
    emit('changed')
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function cancel(id: number) {
  busy.value = true
  try {
    await creatorAPI.cancelWithdraw(id)
    emit('changed')
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

onMounted(() => loadSales(1))
</script>
