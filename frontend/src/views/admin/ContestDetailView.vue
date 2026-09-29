<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card flex flex-wrap items-start justify-between gap-4 p-5">
        <div class="min-w-0">
          <RouterLink to="/admin/contests" class="text-sm text-gray-500 hover:text-primary-600">← {{ t('admin.contests.title') }}</RouterLink>
          <h2 class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ contest?.title || '…' }}</h2>
          <div v-if="contest" class="mt-2 flex flex-wrap items-center gap-3 text-sm text-gray-500 dark:text-dark-400">
            <span :class="phaseBadgeClass(contest.phase)">{{ t(`contests.phase.${contest.phase}`) }}</span>
            <span>{{ t('contests.schedule.deadline') }} {{ formatDateTime(contest.voting_end_at) }}</span>
            <span>{{ t('contests.stats.entries', { n: contest.entry_count }) }}</span>
            <span>{{ t('contests.stats.votes', { n: contest.vote_count }) }}</span>
          </div>
        </div>
        <div class="flex flex-wrap gap-3">
          <RouterLink v-if="contest && contest.status !== 'draft'" :to="`/contests/${contestId}`" target="_blank" class="btn btn-secondary">
            {{ t('admin.contests.viewPublic') }}
          </RouterLink>
          <button v-if="contest?.phase === 'tallying'" type="button" class="btn btn-primary" @click="confirmState = { kind: 'settle' }">
            {{ t('admin.contests.actions.settle') }}
          </button>
        </div>
      </div>

      <div class="flex gap-2 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          :class="[
            '-mb-px border-b-2 px-4 py-2 text-sm font-medium',
            activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800'
          ]"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Entries -->
      <section v-show="activeTab === 'entries'" class="space-y-4">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="w-48">
            <label class="input-label">{{ t('admin.contests.detail.filterStatus') }}</label>
            <Select v-model="statusFilter" :options="statusOptions" @change="loadEntries" />
          </div>
          <p class="text-xs text-gray-500">{{ t('admin.contests.detail.disqualifyHint') }}</p>
        </div>
        <DataTable :columns="entryColumns" :data="entries" :loading="entriesLoading" row-key="id">
          <template #cell-entry="{ row }">
            <div class="flex items-center gap-3">
              <a :href="row.image_url" target="_blank" rel="noopener noreferrer">
                <img :src="row.image_url" alt="" class="h-14 w-14 rounded-lg object-cover" />
              </a>
              <div class="min-w-0 max-w-xs">
                <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.title">{{ row.title }}</div>
                <div class="truncate text-xs text-gray-500">{{ row.user_email }} · #{{ row.user_id }}</div>
                <div v-if="row.review_note" class="truncate text-xs text-red-500">{{ row.review_note }}</div>
              </div>
            </div>
          </template>
          <template #cell-status="{ row }">
            <span class="whitespace-nowrap text-sm">{{ t(`contests.entries.status.${row.status}`) }}</span>
          </template>
          <template #cell-votes="{ row }">
            <span class="font-semibold">{{ row.vote_count }}</span>
            <span v-if="row.final_rank" class="ml-2 text-xs text-amber-600">#{{ row.final_rank }}</span>
          </template>
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-xs text-gray-500">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap gap-3 text-sm font-medium">
              <button type="button" class="text-primary-600 hover:text-primary-700" @click="openVotes(row)">{{ t('admin.contests.detail.viewVotes') }}</button>
              <template v-if="contest?.status !== 'settled'">
                <button v-if="row.status !== 'approved'" type="button" class="text-emerald-600" @click="review(row, 'approved')">
                  {{ t('admin.contests.detail.approve') }}
                </button>
                <button v-if="row.status === 'pending'" type="button" class="text-gray-500 hover:text-red-600" @click="openNote('rejected', row)">
                  {{ t('admin.contests.detail.reject') }}
                </button>
                <button v-if="row.status === 'approved'" type="button" class="text-red-500" @click="openNote('disqualified', row)">
                  {{ t('admin.contests.detail.disqualify') }}
                </button>
              </template>
            </div>
          </template>
        </DataTable>
        <Pagination v-if="entriesTotal > pageSize" :total="entriesTotal" :page="page" :page-size="pageSize" @update:page="onPage" />
      </section>

      <!-- Awards -->
      <section v-show="activeTab === 'awards'" class="space-y-4">
        <div v-if="awards.length === 0" class="card py-12 text-center text-sm text-gray-500">{{ t('admin.contests.detail.noAwards') }}</div>
        <template v-else>
          <div class="flex justify-end">
            <button type="button" class="btn btn-primary" :disabled="!hasGrantableBalance" @click="grantAll">
              {{ t('admin.contests.detail.grantAllBalance') }}
            </button>
          </div>
          <DataTable :columns="awardColumns" :data="awards" row-key="id">
            <template #cell-place="{ value }"><span class="font-bold text-amber-600">#{{ value }}</span></template>
            <template #cell-winner="{ row }">
              <div class="text-sm">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.user_email }}</div>
                <div class="text-xs text-gray-500">{{ row.entry_title }}</div>
              </div>
            </template>
            <template #cell-prize="{ row }">
              <span class="text-sm">{{ formatPrizeReward({ type: row.prize_type, amount: row.amount, label: row.label }, t) }}</span>
            </template>
            <template #cell-status="{ row }">
              <span :class="awardBadge(row.status)">{{ t(`admin.contests.award.${row.status}`) }}</span>
              <div v-if="row.note" class="mt-1 max-w-xs truncate text-xs text-gray-500" :title="row.note">{{ row.note }}</div>
            </template>
            <template #cell-actions="{ row }">
              <button
                v-if="row.status === 'pending' || row.status === 'failed'"
                type="button"
                class="text-sm font-medium text-primary-600 hover:text-primary-700"
                @click="openGrant(row)"
              >
                {{ row.prize_type === 'balance' ? t('admin.contests.detail.grant') : t('admin.contests.detail.markDelivered') }}
              </button>
            </template>
          </DataTable>
        </template>
      </section>
    </div>

    <!-- Votes dialog -->
    <BaseDialog :show="!!votesEntry" :title="`${t('admin.contests.detail.votesTitle')} · ${votesEntry?.title || ''}`" width="wide" @close="votesEntry = null">
      <p v-if="votes.length === 0" class="py-8 text-center text-sm text-gray-500">{{ t('admin.contests.detail.noVotes') }}</p>
      <table v-else class="w-full text-sm">
        <thead class="text-left text-xs text-gray-500">
          <tr>
            <th class="py-2">{{ t('admin.contests.detail.voter') }}</th>
            <th>{{ t('admin.contests.detail.voterRegistered') }}</th>
            <th>{{ t('admin.contests.detail.ip') }}</th>
            <th>{{ t('admin.contests.detail.time') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-for="v in votes" :key="v.id">
            <td class="py-2">{{ v.user_email || `#${v.user_id}` }}</td>
            <td class="text-xs text-gray-500">{{ formatDateTime(v.user_created_at) }}</td>
            <td class="font-mono text-xs">{{ v.client_ip || '—' }}</td>
            <td class="text-xs text-gray-500">{{ formatDateTime(v.created_at) }}</td>
            <td class="text-right">
              <button v-if="contest?.status !== 'settled'" type="button" class="text-xs text-red-500 hover:text-red-700" @click="confirmState = { kind: 'void', voteId: v.id }">
                {{ t('admin.contests.detail.voidVote') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </BaseDialog>

    <!-- Note dialog (reject / disqualify / grant) -->
    <BaseDialog :show="!!noteState" :title="noteTitle" width="normal" @close="noteState = null">
      <p v-if="noteState?.kind === 'grant'" class="mb-3 text-sm text-gray-600 dark:text-gray-300">{{ grantMessage }}</p>
      <label class="input-label">{{ noteState?.kind === 'grant' ? t('admin.contests.detail.notePrompt') : t('admin.contests.detail.reviewNotePrompt') }}</label>
      <textarea v-model="noteText" rows="3" maxlength="500" class="input"></textarea>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="noteState = null">{{ t('common.cancel') }}</button>
          <button type="button" :class="['btn', noteState?.kind === 'grant' ? 'btn-primary' : 'btn-danger']" :disabled="busy" @click="submitNote">
            {{ t('common.confirm') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!confirmState"
      :title="confirmState?.kind === 'settle' ? t('admin.contests.actions.settle') : t('admin.contests.detail.voidVote')"
      :message="confirmState?.kind === 'settle' ? t('admin.contests.settleConfirm') : t('admin.contests.detail.voidConfirm')"
      :danger="confirmState?.kind === 'void'"
      @confirm="runConfirm"
      @cancel="confirmState = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useAppStore } from '@/stores'
import { adminAPI, type ContestVote } from '@/api/admin'
import type { Contest, ContestAward, ContestEntry } from '@/api/contests'
import { formatDateTime, formatPrizeReward, phaseBadgeClass } from '@/utils/contest'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const contestId = computed(() => Number(route.params.id))

const contest = ref<Contest | null>(null)
const activeTab = ref<'entries' | 'awards'>('entries')
const tabs = computed(() => [
  { key: 'entries' as const, label: t('admin.contests.detail.entries') },
  { key: 'awards' as const, label: t('admin.contests.detail.awards') }
])

// ---- entries ----
const entries = ref<ContestEntry[]>([])
const entriesTotal = ref(0)
const entriesLoading = ref(false)
const page = ref(1)
const pageSize = 30
const statusFilter = ref('')
const statusOptions = computed(() => [
  { value: '', label: t('admin.contests.detail.all') },
  ...['pending', 'approved', 'rejected', 'disqualified', 'withdrawn'].map((s) => ({ value: s, label: t(`contests.entries.status.${s}`) }))
])
const entryColumns = computed<Column[]>(() => [
  { key: 'entry', label: t('admin.contests.detail.entry') },
  { key: 'status', label: t('admin.contests.fields.status') },
  { key: 'votes', label: t('admin.contests.columns.votes') },
  { key: 'created_at', label: t('admin.contests.detail.time') },
  { key: 'actions', label: t('admin.contests.columns.actions') }
])

async function loadContest() {
  contest.value = await adminAPI.contests.get(contestId.value)
}

async function loadEntries() {
  entriesLoading.value = true
  try {
    const res = await adminAPI.contests.listEntries(contestId.value, {
      status: statusFilter.value || undefined,
      sort: contest.value?.status === 'settled' ? 'rank' : 'votes',
      page: page.value,
      page_size: pageSize
    })
    entries.value = res.items
    entriesTotal.value = res.total
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    entriesLoading.value = false
  }
}

function onPage(p: number) {
  page.value = p
  void loadEntries()
}

async function review(e: ContestEntry, status: string, note = '') {
  try {
    await adminAPI.contests.reviewEntry(contestId.value, e.id, status, note)
    await Promise.all([loadEntries(), loadContest()])
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

// ---- votes ----
const votesEntry = ref<ContestEntry | null>(null)
const votes = ref<ContestVote[]>([])

async function openVotes(e: ContestEntry) {
  votesEntry.value = e
  votes.value = []
  try {
    votes.value = await adminAPI.contests.listEntryVotes(contestId.value, e.id)
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

// ---- awards ----
const awards = ref<ContestAward[]>([])
const awardColumns = computed<Column[]>(() => [
  { key: 'place', label: t('admin.contests.detail.place') },
  { key: 'winner', label: t('admin.contests.detail.winner') },
  { key: 'prize', label: t('admin.contests.detail.prize') },
  { key: 'status', label: t('admin.contests.detail.awardStatus') },
  { key: 'actions', label: t('admin.contests.columns.actions') }
])
const hasGrantableBalance = computed(() => awards.value.some((a) => a.prize_type === 'balance' && (a.status === 'pending' || a.status === 'failed')))

async function loadAwards() {
  awards.value = await adminAPI.contests.listAwards(contestId.value)
}

function awardBadge(status: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-semibold '
  if (status === 'granted') return base + 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'failed') return base + 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return base + 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

async function grantAll() {
  try {
    const res = await adminAPI.contests.grantAllBalance(contestId.value)
    appStore.showSuccess(t('admin.contests.detail.grantAllResult', res))
    await loadAwards()
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

// ---- note dialog ----
type NoteState = { kind: 'rejected' | 'disqualified'; entry: ContestEntry } | { kind: 'grant'; award: ContestAward }
const noteState = ref<NoteState | null>(null)
const noteText = ref('')
const busy = ref(false)
const noteTitle = computed(() => {
  const s = noteState.value
  if (!s) return ''
  if (s.kind === 'grant') return s.award.prize_type === 'balance' ? t('admin.contests.detail.grant') : t('admin.contests.detail.markDelivered')
  return s.kind === 'rejected' ? t('admin.contests.detail.reject') : t('admin.contests.detail.disqualify')
})
const grantMessage = computed(() => {
  const s = noteState.value
  if (!s || s.kind !== 'grant') return ''
  return t('admin.contests.detail.grantConfirm', {
    winner: s.award.user_email || `#${s.award.user_id}`,
    prize: formatPrizeReward({ type: s.award.prize_type, amount: s.award.amount, label: s.award.label }, t)
  })
})

function openNote(kind: 'rejected' | 'disqualified', entry: ContestEntry) {
  noteText.value = ''
  noteState.value = { kind, entry }
}

function openGrant(award: ContestAward) {
  noteText.value = ''
  noteState.value = { kind: 'grant', award }
}

async function submitNote() {
  const s = noteState.value
  if (!s) return
  busy.value = true
  try {
    if (s.kind === 'grant') {
      await adminAPI.contests.grantAward(contestId.value, s.award.id, noteText.value)
      await loadAwards()
    } else {
      await review(s.entry, s.kind, noteText.value)
    }
    noteState.value = null
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    busy.value = false
  }
}

// ---- confirm dialog ----
const confirmState = ref<{ kind: 'settle' } | { kind: 'void'; voteId: number } | null>(null)

async function runConfirm() {
  const s = confirmState.value
  confirmState.value = null
  if (!s) return
  try {
    if (s.kind === 'settle') {
      await adminAPI.contests.settle(contestId.value)
      appStore.showSuccess(t('admin.contests.settleSuccess'))
      await Promise.all([loadContest(), loadAwards()])
      await loadEntries()
      activeTab.value = 'awards'
    } else {
      await adminAPI.contests.voidVote(contestId.value, s.voteId)
      if (votesEntry.value) await openVotes(votesEntry.value)
      await Promise.all([loadEntries(), loadContest()])
    }
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

onMounted(async () => {
  try {
    await loadContest()
    await Promise.all([loadEntries(), loadAwards()])
    if (contest.value?.status === 'settled') activeTab.value = 'awards'
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.contests.loadFailed'))
  }
})
</script>
