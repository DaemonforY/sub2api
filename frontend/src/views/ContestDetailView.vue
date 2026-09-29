<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar :login-redirect="route.fullPath" />
    <main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      <RouterLink to="/contests" class="text-sm text-gray-500 hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-400">
        ← {{ t('contests.backToList') }}
      </RouterLink>

      <div v-if="loading" class="mt-6 h-64 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      <div v-else-if="!detail" class="card mt-6 py-16 text-center text-sm text-gray-500">{{ t('contests.loadFailed') }}</div>

      <template v-else>
        <!-- Header -->
        <section class="mt-4 overflow-hidden rounded-2xl border border-gray-200/70 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div v-if="safeCover(contest.cover_image)" class="h-48 w-full overflow-hidden sm:h-64">
            <img :src="safeCover(contest.cover_image)" alt="" class="h-full w-full object-cover" />
          </div>
          <div class="p-6">
            <div class="flex flex-wrap items-center gap-3">
              <span :class="phaseBadgeClass(contest.phase)">{{ t(`contests.phase.${contest.phase}`) }}</span>
              <span v-if="countdown" class="text-sm font-medium text-primary-600 dark:text-primary-400" data-testid="contest-countdown">
                {{ t(countdown.labelKey) }} {{ remaining }}
              </span>
            </div>
            <h1 class="mt-3 text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ contest.title }}</h1>
            <p v-if="contest.description" class="mt-3 whitespace-pre-line text-sm leading-relaxed text-gray-600 dark:text-dark-300">
              {{ contest.description }}
            </p>
            <dl class="mt-5 grid gap-3 text-sm sm:grid-cols-2">
              <div>
                <dt class="text-gray-500 dark:text-dark-400">{{ t('contests.schedule.submission') }}</dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(contest.submission_start_at) }} – {{ formatDateTime(contest.submission_end_at) }}
                </dd>
              </div>
              <div>
                <dt class="text-gray-500 dark:text-dark-400">{{ t('contests.schedule.voting') }}</dt>
                <dd class="font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(contest.voting_start_at) }} – {{ formatDateTime(contest.voting_end_at) }}
                </dd>
              </div>
            </dl>
          </div>
        </section>

        <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_320px]">
          <!-- Entries -->
          <section>
            <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('contests.entries.title') }}
                <span class="ml-1 text-sm font-normal text-gray-500">({{ entriesTotal }})</span>
              </h2>
              <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
                <button
                  v-for="opt in sortOptions"
                  :key="opt.value"
                  type="button"
                  :class="[
                    'rounded-md px-3 py-1 text-sm font-medium transition',
                    sortBy === opt.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-800 dark:text-dark-400'
                  ]"
                  @click="changeSort(opt.value)"
                >
                  {{ opt.label }}
                </button>
              </div>
            </div>

            <div v-if="entries.length === 0 && !entriesLoading" class="card py-16 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('contests.entries.empty') }}
            </div>
            <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              <article
                v-for="e in entries"
                :key="e.id"
                class="overflow-hidden rounded-xl border border-gray-200/70 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
                data-testid="contest-entry"
              >
                <button type="button" class="block aspect-square w-full overflow-hidden bg-gray-100 dark:bg-dark-900" @click="lightbox = e">
                  <img :src="e.image_url" :alt="e.title" loading="lazy" class="h-full w-full object-cover transition hover:scale-105" />
                </button>
                <div class="p-3">
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white" :title="e.title">{{ e.title }}</h3>
                      <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('contests.entries.by', { name: e.author_name }) }}</p>
                    </div>
                    <span v-if="e.is_mine" class="flex-shrink-0 rounded bg-primary-50 px-1.5 py-0.5 text-[10px] font-semibold text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                      {{ t('contests.entries.mine') }}
                    </span>
                  </div>
                  <div class="mt-3 flex items-center justify-between">
                    <span class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('contests.entries.votes', { n: e.vote_count }) }}</span>
                    <button
                      v-if="votingOpen"
                      type="button"
                      :disabled="voteBusy === e.id || (!e.voted_by_me && !canCastNewVote(e))"
                      :class="[
                        'btn btn-sm',
                        e.voted_by_me ? 'btn-secondary' : 'btn-primary'
                      ]"
                      :title="voteButtonTitle(e)"
                      @click="toggleVote(e)"
                    >
                      {{ e.voted_by_me ? t('contests.entries.voted') : t('contests.entries.vote') }}
                    </button>
                  </div>
                </div>
              </article>
            </div>
            <div v-if="entries.length < entriesTotal" class="mt-6 text-center">
              <button type="button" class="btn btn-secondary" :disabled="entriesLoading" @click="loadEntries(false)">
                {{ t('common.loadMore') }}
              </button>
            </div>
          </section>

          <!-- Sidebar -->
          <aside class="space-y-4">
            <div v-if="submittingOpen || votingOpen" class="card p-5" data-testid="contest-action-card">
              <template v-if="!viewer.logged_in">
                <RouterLink :to="{ path: '/login', query: { redirect: route.fullPath } }" class="btn btn-primary w-full">
                  {{ submittingOpen ? t('contests.viewer.loginToSubmit') : t('contests.viewer.loginToVote') }}
                </RouterLink>
              </template>
              <template v-else>
                <button
                  v-if="submittingOpen"
                  type="button"
                  class="btn btn-primary w-full"
                  :disabled="!viewer.can_submit"
                  data-testid="contest-submit-open"
                  @click="openSubmit"
                >
                  {{ t('contests.submit.button') }}
                </button>
                <p v-if="submittingOpen" class="mt-2 text-center text-xs text-gray-500 dark:text-dark-400">
                  {{ t('contests.submit.limit', { n: contest.max_entries_per_user, left: viewer.entries_left }) }}
                </p>
                <div v-if="votingOpen" class="mt-3 rounded-lg bg-gray-50 px-3 py-2 text-center text-sm dark:bg-dark-900">
                  <template v-if="viewer.vote_blocked_reason === 'account_too_new'">
                    <span class="text-amber-600 dark:text-amber-400">{{ t('contests.viewer.accountTooNew', { hours: contest.min_account_age_hours }) }}</span>
                  </template>
                  <template v-else>
                    <span class="font-medium text-gray-800 dark:text-gray-200">
                      {{ t('contests.viewer.votesLeft', { left: viewer.votes_left, total: contest.votes_per_user }) }}
                    </span>
                    <p v-if="viewer.votes_left === 0" class="mt-1 text-xs text-gray-500">{{ t('contests.viewer.noVotesLeft') }}</p>
                  </template>
                </div>
              </template>
            </div>

            <!-- My entries -->
            <div v-if="viewer.my_entries.length" class="card p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.viewer.myEntries') }}</h3>
              <ul class="mt-3 space-y-3">
                <li v-for="e in viewer.my_entries" :key="e.id" class="flex items-center gap-3">
                  <img :src="e.image_url" alt="" class="h-12 w-12 flex-shrink-0 rounded-lg object-cover" />
                  <div class="min-w-0 flex-1">
                    <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ e.title }}</div>
                    <div class="text-xs text-gray-500">
                      {{ t(`contests.entries.status.${e.status}`) }} · {{ t('contests.entries.votes', { n: e.vote_count }) }}
                    </div>
                    <div v-if="e.review_note" class="text-xs text-red-500">{{ e.review_note }}</div>
                  </div>
                  <button
                    v-if="canWithdraw(e)"
                    type="button"
                    class="flex-shrink-0 text-xs text-gray-400 hover:text-red-500"
                    @click="withdrawTarget = e"
                  >
                    {{ t('contests.entries.withdraw') }}
                  </button>
                </li>
              </ul>
            </div>

            <!-- Winners -->
            <div v-if="detail.awards?.length" class="card p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">🏆 {{ t('contests.winners.title') }}</h3>
              <ul class="mt-3 space-y-2 text-sm">
                <li v-for="a in detail.awards" :key="a.id" class="flex items-start gap-2">
                  <span class="w-8 flex-shrink-0 font-semibold text-amber-600">#{{ a.place }}</span>
                  <div class="min-w-0">
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ a.entry_title }} · {{ a.author_name }}</div>
                    <div class="text-xs text-gray-500">{{ formatPrizeReward({ type: a.prize_type, amount: a.amount, label: a.label }, t) }}</div>
                  </div>
                </li>
              </ul>
            </div>

            <!-- Prizes -->
            <div v-if="contest.prizes.length" class="card p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.prizes.title') }}</h3>
              <ul class="mt-3 space-y-2 text-sm">
                <li v-for="(p, i) in contest.prizes" :key="i" class="flex gap-2">
                  <span class="flex-shrink-0 font-semibold text-gray-800 dark:text-gray-200">{{ formatPrizePlaces(p, t) }}</span>
                  <span class="text-gray-600 dark:text-dark-300">{{ formatPrizeReward(p, t) }}</span>
                </li>
              </ul>
              <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">{{ t('contests.prizes.rulesNote') }}</p>
            </div>

            <!-- Leaderboard -->
            <div class="card p-5">
              <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.leaderboard.title') }}</h3>
                <span class="text-xs text-gray-500">{{ board[0]?.final ? t('contests.leaderboard.final') : t('contests.leaderboard.live') }}</span>
              </div>
              <p v-if="board.length === 0" class="mt-3 text-sm text-gray-500">{{ t('contests.leaderboard.empty') }}</p>
              <ol class="mt-3 space-y-2">
                <li v-for="row in board" :key="row.entry_id" class="flex items-center gap-3 text-sm">
                  <span :class="['w-6 flex-shrink-0 text-center font-bold', rankColor(row.rank)]">{{ row.rank }}</span>
                  <img :src="row.image_url" alt="" class="h-8 w-8 flex-shrink-0 rounded object-cover" />
                  <span class="min-w-0 flex-1 truncate text-gray-800 dark:text-gray-200">{{ row.title }}</span>
                  <span class="flex-shrink-0 font-semibold text-gray-900 dark:text-white">{{ row.votes }}</span>
                </li>
              </ol>
            </div>

            <div v-if="contest.rules" class="card p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.rules') }}</h3>
              <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ contest.rules }}</p>
            </div>
          </aside>
        </div>
      </template>
    </main>

    <!-- Submit dialog -->
    <BaseDialog :show="submitOpen" :title="t('contests.submit.title')" width="normal" @close="submitOpen = false">
      <form class="space-y-4" @submit.prevent="submitEntry">
        <div>
          <label class="input-label">{{ t('contests.submit.image') }}</label>
          <label
            class="flex cursor-pointer flex-col items-center justify-center overflow-hidden rounded-xl border-2 border-dashed border-gray-300 bg-gray-50 transition hover:border-primary-400 dark:border-dark-600 dark:bg-dark-900"
            :class="previewUrl ? 'h-64' : 'h-40'"
          >
            <img v-if="previewUrl" :src="previewUrl" alt="" class="h-full w-full object-contain" />
            <span v-else class="text-sm text-gray-500">{{ t('contests.submit.chooseImage') }}</span>
            <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" class="hidden" data-testid="contest-image-input" @change="onFileChange" />
          </label>
          <p class="mt-1 text-xs text-gray-500">{{ t('contests.submit.imageHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('contests.submit.titleLabel') }}</label>
          <input v-model.trim="form.title" type="text" maxlength="120" class="input" :placeholder="t('contests.submit.titlePlaceholder')" />
        </div>
        <div>
          <label class="input-label">{{ t('contests.submit.description') }}</label>
          <textarea v-model="form.description" rows="3" maxlength="2000" class="input" :placeholder="t('contests.submit.descriptionPlaceholder')"></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('contests.submit.prompt') }}</label>
          <textarea v-model="form.prompt" rows="3" maxlength="4000" class="input font-mono text-xs" :placeholder="t('contests.submit.promptPlaceholder')"></textarea>
        </div>
        <p v-if="formError" class="text-sm text-red-500">{{ formError }}</p>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="submitOpen = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="submitting" @click="submitEntry">
            {{ submitting ? t('contests.submit.submitting') : t('contests.submit.submit') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Lightbox -->
    <BaseDialog :show="!!lightbox" :title="lightbox?.title || ''" width="wide" :close-on-click-outside="true" @close="lightbox = null">
      <div v-if="lightbox" class="space-y-4">
        <img :src="lightbox.image_url" :alt="lightbox.title" class="mx-auto max-h-[70vh] rounded-lg object-contain" />
        <div class="text-sm text-gray-500">{{ t('contests.entries.by', { name: lightbox.author_name }) }} · {{ t('contests.entries.votes', { n: lightbox.vote_count }) }}</div>
        <p v-if="lightbox.description" class="whitespace-pre-line text-sm text-gray-700 dark:text-gray-300">{{ lightbox.description }}</p>
        <div v-if="lightbox.prompt">
          <div class="text-xs font-semibold text-gray-500">{{ t('contests.entries.prompt') }}</div>
          <pre class="mt-1 whitespace-pre-wrap rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-300">{{ lightbox.prompt }}</pre>
        </div>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="!!withdrawTarget"
      :title="t('contests.entries.withdraw')"
      :message="t('contests.entries.withdrawConfirm')"
      danger
      @confirm="confirmWithdraw"
      @cancel="withdrawTarget = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useAppStore } from '@/stores'
import {
  contestsAPI,
  type ContestDetail,
  type ContestEntry,
  type ContestLeaderboardRow,
  type ContestViewer
} from '@/api/contests'
import {
  contestCountdown,
  formatDateTime,
  formatPrizePlaces,
  formatPrizeReward,
  formatRemaining,
  phaseBadgeClass,
  safeCover
} from '@/utils/contest'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const contestId = computed(() => Number(route.params.id))

const loading = ref(true)
const detail = ref<ContestDetail | null>(null)
const contest = computed(() => detail.value!.contest)
const emptyViewer: ContestViewer = {
  logged_in: false, votes_used: 0, votes_left: 0, voted_entry_ids: [], my_entries: [],
  entries_left: 0, can_submit: false, can_vote: false
}
const viewer = computed(() => detail.value?.viewer || emptyViewer)

const entries = ref<ContestEntry[]>([])
const entriesTotal = ref(0)
const entriesPage = ref(1)
const entriesLoading = ref(false)
const sortBy = ref<'votes' | 'new'>('votes')
const sortOptions = computed(() => [
  { value: 'votes' as const, label: t('contests.entries.sortVotes') },
  { value: 'new' as const, label: t('contests.entries.sortNew') }
])
const board = ref<ContestLeaderboardRow[]>([])

const voteBusy = ref(0)
const lightbox = ref<ContestEntry | null>(null)
const withdrawTarget = ref<ContestEntry | null>(null)

const submittingOpen = computed(() => ['submitting', 'submitting_voting'].includes(contest.value.phase))
const votingOpen = computed(() => ['voting', 'submitting_voting'].includes(contest.value.phase))

// Countdown ticks every 30s.
const nowTick = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
const countdown = computed(() => (detail.value ? contestCountdown(contest.value) : null))
const remaining = computed(() => (countdown.value ? formatRemaining(new Date(countdown.value.target).getTime() - nowTick.value, t) : ''))

function rankColor(rank: number): string {
  return rank === 1 ? 'text-amber-500' : rank === 2 ? 'text-gray-400' : rank === 3 ? 'text-orange-400' : 'text-gray-500'
}

function canCastNewVote(e: ContestEntry): boolean {
  if (!viewer.value.logged_in) return true // clicking redirects to login
  if (e.is_mine && !contest.value.allow_self_vote) return false
  return viewer.value.can_vote
}

function voteButtonTitle(e: ContestEntry): string {
  if (!viewer.value.logged_in) return t('contests.viewer.loginToVote')
  if (e.voted_by_me) return t('contests.entries.unvote')
  if (viewer.value.votes_left === 0) return t('contests.viewer.noVotesLeft')
  return ''
}

function canWithdraw(e: ContestEntry): boolean {
  return e.status !== 'withdrawn' && contest.value.status !== 'settled' && new Date(contest.value.voting_end_at).getTime() > Date.now()
}

async function loadDetail() {
  detail.value = await contestsAPI.get(contestId.value)
}

async function loadBoard() {
  board.value = await contestsAPI.leaderboard(contestId.value, 10)
}

async function loadEntries(reset: boolean) {
  entriesLoading.value = true
  try {
    const page = reset ? 1 : entriesPage.value + 1
    const res = await contestsAPI.listEntries(contestId.value, { sort: sortBy.value, page, page_size: 24 })
    entries.value = reset ? res.items : [...entries.value, ...res.items]
    entriesTotal.value = res.total
    entriesPage.value = page
  } finally {
    entriesLoading.value = false
  }
}

function changeSort(value: 'votes' | 'new') {
  if (sortBy.value === value) return
  sortBy.value = value
  void loadEntries(true)
}

async function refreshAll() {
  await Promise.all([loadDetail(), loadBoard(), loadEntries(true)])
}

async function toggleVote(e: ContestEntry) {
  if (!viewer.value.logged_in) {
    window.location.assign(`/login?redirect=${encodeURIComponent(route.fullPath)}`)
    return
  }
  voteBusy.value = e.id
  try {
    if (e.voted_by_me) {
      await contestsAPI.unvote(contestId.value, e.id)
      e.voted_by_me = false
      e.vote_count = Math.max(0, e.vote_count - 1)
      appStore.showSuccess(t('contests.toast.unvoted'))
    } else {
      await contestsAPI.vote(contestId.value, e.id)
      e.voted_by_me = true
      e.vote_count += 1
      appStore.showSuccess(t('contests.toast.voted'))
    }
    await Promise.all([loadDetail(), loadBoard()])
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    voteBusy.value = 0
  }
}

// ---- Submit ----
const submitOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const file = ref<File | null>(null)
const previewUrl = ref('')
const form = reactive({ title: '', description: '', prompt: '' })

function openSubmit() {
  formError.value = ''
  submitOpen.value = true
}

function onFileChange(ev: Event) {
  const f = (ev.target as HTMLInputElement).files?.[0] || null
  formError.value = ''
  if (f && f.size > 10 * 1024 * 1024) {
    formError.value = t('contests.submit.imageTooLarge')
    return
  }
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  file.value = f
  previewUrl.value = f ? URL.createObjectURL(f) : ''
}

async function submitEntry() {
  formError.value = ''
  if (!file.value) {
    formError.value = t('contests.submit.imageRequired')
    return
  }
  if (!form.title) {
    formError.value = t('contests.submit.titleRequired')
    return
  }
  submitting.value = true
  try {
    const entry = await contestsAPI.submitEntry(contestId.value, { ...form, image: file.value })
    appStore.showSuccess(entry.status === 'pending' ? t('contests.submit.pendingReview') : t('contests.submit.success'))
    submitOpen.value = false
    Object.assign(form, { title: '', description: '', prompt: '' })
    onFileChange({ target: { files: [] } } as unknown as Event)
    await refreshAll()
  } catch (err: any) {
    formError.value = err?.message || t('common.error')
  } finally {
    submitting.value = false
  }
}

async function confirmWithdraw() {
  const e = withdrawTarget.value
  withdrawTarget.value = null
  if (!e) return
  try {
    await contestsAPI.withdrawEntry(contestId.value, e.id)
    appStore.showSuccess(t('contests.toast.withdrawn'))
    await refreshAll()
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

onMounted(async () => {
  timer = setInterval(() => (nowTick.value = Date.now()), 30000)
  try {
    await refreshAll()
  } catch {
    detail.value = null
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})
</script>
