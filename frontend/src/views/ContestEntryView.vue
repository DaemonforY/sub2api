<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar :login-redirect="route.fullPath" />
    <main class="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
      <RouterLink :to="`/contests/${contestId}`" class="text-sm text-gray-500 hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-400">
        ← {{ detail ? detail.contest.title : t('contests.entry.back') }}
      </RouterLink>

      <div v-if="loading" class="mt-6 h-96 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      <div v-else-if="!detail || !entry" class="card mt-6 py-16 text-center text-sm text-gray-500" data-testid="entry-not-found">
        <p>{{ t('contests.entry.notFound') }}</p>
        <RouterLink :to="`/contests/${contestId}`" class="btn btn-secondary mt-4">{{ t('contests.entry.seeOthers') }}</RouterLink>
      </div>

      <div v-else class="mt-4 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <section class="overflow-hidden rounded-2xl border border-gray-200/70 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <button type="button" class="block w-full bg-gray-100 dark:bg-dark-900" @click="zoom = true">
            <img :src="entry.image_url" :alt="entry.title" class="mx-auto max-h-[78vh] w-auto object-contain" data-testid="entry-image" />
          </button>
          <div v-if="entry.description || entry.prompt" class="space-y-4 p-5">
            <p v-if="entry.description" class="whitespace-pre-line text-sm leading-relaxed text-gray-700 dark:text-gray-300">{{ entry.description }}</p>
            <div v-if="entry.prompt">
              <div class="text-xs font-semibold text-gray-500">{{ t('contests.entries.prompt') }}</div>
              <pre class="mt-1 whitespace-pre-wrap rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-300">{{ entry.prompt }}</pre>
            </div>
          </div>
        </section>

        <aside class="space-y-4">
          <div class="card p-5" data-testid="entry-vote-card">
            <div class="flex flex-wrap items-center gap-2">
              <span :class="phaseBadgeClass(contest.phase)">{{ t(`contests.phase.${contest.phase}`) }}</span>
              <span v-if="rank" class="rounded-full bg-amber-50 px-2.5 py-0.5 text-xs font-semibold text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                🏆 {{ t('contests.entry.rank', { n: rank }) }}
              </span>
            </div>
            <h1 class="mt-3 text-xl font-bold text-gray-900 dark:text-white">{{ entry.title }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('contests.entries.by', { name: entry.author_name }) }}</p>

            <p v-if="entry.status !== 'approved'" class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
              {{ entry.status === 'pending' ? t('contests.entry.pendingNote') : t(`contests.entries.status.${entry.status}`) }}
            </p>

            <div class="mt-5 text-center">
              <div class="text-4xl font-bold text-gray-900 dark:text-white" data-testid="entry-votes">{{ entry.vote_count }}</div>
              <div class="text-xs text-gray-500">{{ t('contests.entry.votesLabel') }}</div>
            </div>

            <button
              v-if="!entry.is_mine && entry.status === 'approved'"
              type="button"
              :class="['btn mt-4 w-full py-3 text-base', entry.voted_by_me ? 'btn-secondary' : 'btn-primary']"
              :disabled="voteBusy || (!entry.voted_by_me && !!voteBlocked)"
              data-testid="entry-vote"
              @click="toggleVote"
            >
              {{ entry.voted_by_me ? `✓ ${t('contests.entry.votedCta')}` : `👍 ${t('contests.entry.voteCta')}` }}
            </button>
            <p v-if="voteHint" class="mt-2 text-center text-xs text-gray-500 dark:text-dark-400" data-testid="entry-vote-hint">{{ voteHint }}</p>

            <button
              type="button"
              :class="['btn mt-3 w-full', entry.is_mine ? 'btn-primary py-3 text-base' : 'btn-secondary']"
              data-testid="entry-share"
              @click="shareOpen = true"
            >
              📣 {{ entry.is_mine ? t('contests.entry.shareMine') : t('contests.entry.share') }}
            </button>
            <a
              v-if="entry.work_id"
              :href="canvasUrl({ medium: 'contest-entry', path: `/w/${entry.work_id}` })"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-3 block text-center text-xs text-primary-600 hover:underline dark:text-primary-400"
            >{{ t('contests.entries.viewWork') }} ↗</a>
          </div>

          <div v-if="!entry.is_mine" class="card p-5" data-testid="entry-join">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.entry.joinTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">
              {{ firstPrize ? t('contests.entry.joinPrize', { prize: firstPrize }) : t('contests.entry.joinNoPrize') }}
            </p>
            <div class="mt-3 flex gap-2">
              <a
                v-if="submittingOpen"
                :href="canvasUrl({ medium: 'contest-entry-join', path: '/image' })"
                target="_blank"
                rel="noopener noreferrer"
                class="btn btn-primary flex-1"
                @click="track('cta_click', { where: 'contest_entry_create' })"
              >{{ t('contests.entry.goCreate') }}</a>
              <RouterLink :to="`/contests/${contestId}`" class="btn btn-secondary flex-1">{{ t('contests.entry.seeOthers') }}</RouterLink>
            </div>
          </div>
        </aside>
      </div>
    </main>

    <BaseDialog :show="zoom && !!entry" :title="entry?.title || ''" width="wide" :close-on-click-outside="true" @close="zoom = false">
      <img v-if="entry" :src="entry.image_url" :alt="entry.title" class="mx-auto max-h-[80vh] rounded-lg object-contain" />
    </BaseDialog>

    <ShareCardDialog :show="shareOpen" :input="shareInput" @close="shareOpen = false" />
  </div>
</template>

<script setup lang="ts">
// One contest entry, the page people share to ask for votes: the picture, a big 点赞 button that
// survives sign-in (the vote is remembered and cast on return), and the 拉票 poster / link.
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { useAppStore } from '@/stores'
import { contestsAPI, type ContestDetail, type ContestEntry } from '@/api/contests'
import { canvasUrl } from '@/constants/crossSites'
import { track } from '@/utils/analytics'
import { formatPrizeReward, phaseBadgeClass } from '@/utils/contest'
import { contestEntryPath, contestPitch, saveVoteIntent, takeVoteIntent } from '@/utils/contestShare'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const contestId = computed(() => Number(route.params.id))
const entryId = computed(() => Number(route.params.entryId))

const loading = ref(true)
const detail = ref<ContestDetail | null>(null)
const entry = ref<ContestEntry | null>(null)
const rank = ref(0)
const voteBusy = ref(false)
const zoom = ref(false)
const shareOpen = ref(false)

const contest = computed(() => detail.value!.contest)
const viewer = computed(() => detail.value?.viewer)
const votingOpen = computed(() => ['voting', 'submitting_voting'].includes(contest.value.phase))
const submittingOpen = computed(() => ['submitting', 'submitting_voting'].includes(contest.value.phase))

const firstPrize = computed(() => {
  const p = detail.value?.contest.prizes.find((x) => x.rank_from === 1)
  return p ? t('contests.entry.firstPrize', { reward: formatPrizeReward(p, t) }) : ''
})

/** Why a new vote can't be cast now (signed-out visitors can always click: it leads to sign-in). */
const voteBlocked = computed(() => {
  if (!detail.value || !entry.value) return ''
  if (!votingOpen.value) return t('contests.entry.votingClosed')
  if (!viewer.value?.logged_in) return ''
  if (entry.value.is_mine && !contest.value.allow_self_vote) return t('contests.entry.selfVote')
  if (viewer.value.vote_blocked_reason === 'account_too_new') return t('contests.viewer.accountTooNew', { hours: contest.value.min_account_age_hours })
  if (viewer.value.votes_left <= 0) return t('contests.viewer.noVotesLeft')
  return ''
})

const voteHint = computed(() => {
  if (!detail.value || !entry.value || entry.value.is_mine || entry.value.status !== 'approved') return ''
  if (!entry.value.voted_by_me && voteBlocked.value) return voteBlocked.value
  if (!viewer.value?.logged_in) return votingOpen.value ? t('contests.entry.posterMeta') : ''
  return t('contests.viewer.votesLeft', { left: viewer.value.votes_left, total: contest.value.votes_per_user })
})

const shareInput = computed<ShareCardInput | null>(() => {
  const e = entry.value
  const d = detail.value
  if (!e || !d) return null
  return {
    kind: 'image',
    brand: 'HiveGPT 创作比赛',
    label: d.contest.title,
    title: e.title,
    image: e.image_url,
    caption: t('contests.entry.posterCaption', { author: e.author_name, contest: d.contest.title }),
    cta: t('contests.entry.posterCta'),
    meta: firstPrize.value || t('contests.entry.posterMeta'),
    pageUrl: window.location.origin + contestEntryPath(d.contest.id, e.id),
    source: 'contest_share',
    campaign: `contest-${d.contest.id}`,
    trackKind: 'contest',
    dialogTitle: t('contests.entry.shareTitle'),
    pitch: (link) => contestPitch({ title: e.title, author: e.author_name, mine: e.is_mine }, d.contest.title, link)
  }
})

async function load() {
  const [d, e] = await Promise.all([contestsAPI.get(contestId.value), contestsAPI.getEntry(contestId.value, entryId.value)])
  detail.value = d
  entry.value = e
  try {
    const board = await contestsAPI.leaderboard(contestId.value, 10)
    rank.value = board.find((r) => r.entry_id === e.id)?.rank || 0
  } catch {
    rank.value = 0
  }
}

async function castVote(afterLogin: boolean) {
  const e = entry.value
  if (!e) return
  voteBusy.value = true
  try {
    await contestsAPI.vote(contestId.value, e.id)
    track('contest_vote', { contest: contestId.value, after_login: afterLogin, where: 'entry' })
    await load()
    const left = detail.value?.viewer.votes_left || 0
    appStore.showSuccess(
      [afterLogin ? t('contests.entry.votedAfterLogin') : t('contests.toast.voted'), left > 0 ? t('contests.entry.votesLeftAfter', { n: left }) : ''].filter(Boolean).join(' · ')
    )
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    voteBusy.value = false
  }
}

async function toggleVote() {
  const e = entry.value
  if (!e || !detail.value) return
  if (!viewer.value?.logged_in) {
    saveVoteIntent(contestId.value, e.id)
    track('contest_vote_login', { contest: contestId.value, where: 'entry' })
    window.location.assign(`/login?redirect=${encodeURIComponent(route.fullPath)}`)
    return
  }
  if (e.voted_by_me) {
    voteBusy.value = true
    try {
      await contestsAPI.unvote(contestId.value, e.id)
      await load()
      appStore.showSuccess(t('contests.toast.unvoted'))
    } catch (err: any) {
      appStore.showError(err?.message || t('common.error'))
    } finally {
      voteBusy.value = false
    }
    return
  }
  await castVote(false)
}

onMounted(async () => {
  try {
    await load()
  } catch {
    detail.value = null
    entry.value = null
  } finally {
    loading.value = false
  }
  // Right after submitting, the contest page sends the author here with ?share=1.
  if (route.query.share === '1' && entry.value) {
    shareOpen.value = true
    void router.replace({ query: { ...route.query, share: undefined } })
  }
  // Back from sign-in with a vote the visitor asked for: cast it now.
  const intent = viewer.value?.logged_in ? takeVoteIntent(contestId.value) : null
  if (intent && entry.value && intent.entryId === entry.value.id && !entry.value.voted_by_me && !voteBlocked.value) {
    await castVote(true)
  }
})
</script>
