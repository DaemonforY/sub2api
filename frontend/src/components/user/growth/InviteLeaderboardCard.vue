<template>
  <div v-if="available" class="card p-6" data-testid="invite-leaderboard">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">🏆 {{ t('growth.leaderboard.title') }}</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.leaderboard.description') }}</p>
      </div>
      <div class="flex rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-800">
        <button
          v-for="p in periods"
          :key="p"
          type="button"
          :class="['rounded-md px-3 py-1', period === p ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-400']"
          @click="changePeriod(p)"
        >
          {{ t(`growth.leaderboard.periods.${p}`) }}
        </button>
      </div>
    </div>

    <p v-if="board?.me" class="mt-4 rounded-xl bg-primary-50 px-4 py-3 text-sm text-primary-800 dark:bg-primary-900/20 dark:text-primary-200" data-testid="leaderboard-me">
      {{ t('growth.leaderboard.myRank', { rank: board.me.rank, paying: board.me.paying_invitees, invited: board.me.invited_count }) }}
    </p>

    <div v-if="loading" class="flex justify-center py-8">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>
    <div v-else-if="!board || board.entries.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
      {{ t('growth.leaderboard.empty') }}
    </div>
    <div v-else class="mt-4 overflow-x-auto">
      <table class="w-full min-w-[420px] text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
            <th class="px-3 py-2 font-medium">{{ t('growth.leaderboard.columns.rank') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('growth.leaderboard.columns.user') }}</th>
            <th class="px-3 py-2 text-right font-medium">{{ t('growth.leaderboard.columns.paying') }}</th>
            <th class="px-3 py-2 text-right font-medium">{{ t('growth.leaderboard.columns.invited') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(e, i) in board.entries"
            :key="i"
            :class="['border-b border-gray-100 last:border-b-0 dark:border-dark-800', e.is_current_user ? 'bg-primary-50/60 dark:bg-primary-900/10' : '']"
          >
            <td class="px-3 py-2.5 font-semibold text-gray-900 dark:text-white">{{ medal(e.rank) }}</td>
            <td class="px-3 py-2.5 text-gray-700 dark:text-gray-300">
              {{ e.display_name }}<span v-if="e.is_current_user" class="ml-1 text-xs text-primary-600">({{ t('growth.leaderboard.you') }})</span>
            </td>
            <td class="px-3 py-2.5 text-right font-medium text-gray-900 dark:text-white">{{ e.paying_invitees }}</td>
            <td class="px-3 py-2.5 text-right text-gray-500 dark:text-dark-400">{{ e.invited_count }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { growthAPI, type InviteLeaderboard, type LeaderboardPeriod } from '@/api/growth'

const { t } = useI18n()
const periods: LeaderboardPeriod[] = ['month', 'last_month', 'all']
const period = ref<LeaderboardPeriod>('month')
const board = ref<InviteLeaderboard | null>(null)
const loading = ref(false)
// Hidden entirely when the leaderboard (or the referral program) is turned off.
const available = ref(false)

function medal(rank: number): string {
  return ['🥇', '🥈', '🥉'][rank - 1] ?? String(rank)
}

async function load() {
  loading.value = true
  try {
    board.value = await growthAPI.getLeaderboard(period.value)
    available.value = true
  } catch {
    available.value = false
  } finally {
    loading.value = false
  }
}

function changePeriod(p: LeaderboardPeriod) {
  if (p === period.value) return
  period.value = p
  void load()
}

onMounted(load)
</script>
