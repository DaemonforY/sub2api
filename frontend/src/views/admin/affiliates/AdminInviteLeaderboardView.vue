<template>
  <AppLayout>
    <div class="space-y-6" data-testid="admin-invite-leaderboard">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.leaderboard.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.leaderboard.description') }}</p>
      </div>

      <div class="card flex flex-wrap items-end gap-3 p-4">
        <div>
          <label class="input-label" for="lb-start">{{ t('growth.admin.leaderboard.start') }}</label>
          <input id="lb-start" v-model="start" type="date" class="input w-44" />
        </div>
        <div>
          <label class="input-label" for="lb-end">{{ t('growth.admin.leaderboard.end') }}</label>
          <input id="lb-end" v-model="end" type="date" class="input w-44" />
        </div>
        <button type="button" class="btn btn-primary" :disabled="loading" @click="load">{{ t('growth.admin.leaderboard.query') }}</button>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="preset('month')">{{ t('growth.admin.leaderboard.thisMonth') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="preset('last_month')">{{ t('growth.admin.leaderboard.lastMonth') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="preset('all')">{{ t('growth.admin.leaderboard.allTime') }}</button>
        </div>
      </div>

      <div class="card p-4">
        <div v-if="loading" class="flex justify-center py-10">
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <div v-else-if="entries.length === 0" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('growth.admin.leaderboard.empty') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[760px] text-left text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.leaderboard.columns.rank') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('growth.admin.leaderboard.columns.user') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.leaderboard.columns.paying') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.leaderboard.columns.invited') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.leaderboard.columns.paid') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.leaderboard.columns.rebate') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('growth.admin.leaderboard.columns.rate') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in entries" :key="e.user_id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                <td class="px-3 py-2.5 font-semibold text-gray-900 dark:text-white">{{ e.rank }}</td>
                <td class="px-3 py-2.5">
                  <div class="text-gray-900 dark:text-white">{{ e.email }}</div>
                  <div class="text-xs text-gray-400">#{{ e.user_id }} {{ e.username }}</div>
                </td>
                <td class="px-3 py-2.5 text-right font-medium text-gray-900 dark:text-white">{{ e.paying_invitees }}</td>
                <td class="px-3 py-2.5 text-right text-gray-600 dark:text-gray-300">{{ e.invited_count }}</td>
                <td class="px-3 py-2.5 text-right text-gray-600 dark:text-gray-300">{{ (e.invitee_paid ?? 0).toFixed(2) }}</td>
                <td class="px-3 py-2.5 text-right text-emerald-600 dark:text-emerald-400">{{ (e.rebate_accrued ?? 0).toFixed(2) }}</td>
                <td class="px-3 py-2.5 text-right text-gray-500 dark:text-dark-400">
                  {{ e.custom_rate_percent != null ? `${e.custom_rate_percent}%` : t('growth.admin.leaderboard.defaultRate') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminGrowthAPI, type InviteLeaderboardEntry } from '@/api/growth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const start = ref('')
const end = ref('')
const loading = ref(false)
const entries = ref<InviteLeaderboardEntry[]>([])

function ymd(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

function preset(kind: 'month' | 'last_month' | 'all') {
  const now = new Date()
  if (kind === 'all') {
    start.value = ''
    end.value = ''
  } else {
    const first = new Date(now.getFullYear(), now.getMonth() + (kind === 'month' ? 0 : -1), 1)
    const next = new Date(first.getFullYear(), first.getMonth() + 1, 1)
    start.value = ymd(first)
    end.value = ymd(next)
  }
  void load()
}

async function load() {
  loading.value = true
  try {
    const board = await adminGrowthAPI.getLeaderboard({ start: start.value || undefined, end: end.value || undefined, limit: 200 })
    entries.value = board.entries
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(() => preset('month'))
</script>
