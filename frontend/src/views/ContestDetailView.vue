<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar :login-redirect="route.fullPath" />
    <main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      <RouterLink to="/contests" class="text-sm text-gray-500 hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-400">
        ← {{ t('contests.backToList') }}
      </RouterLink>

      <div v-if="loading" class="mt-4 h-80 animate-pulse rounded-3xl bg-gray-200 dark:bg-dark-800"></div>
      <div v-else-if="!detail" class="card mt-6 py-16 text-center text-sm text-gray-500">{{ t('contests.loadFailed') }}</div>

      <template v-else>
        <!-- Hero: cover, title, countdown, the prize and what to do now -->
        <section class="relative mt-4 overflow-hidden rounded-3xl bg-gradient-to-br from-[#2e1065] via-[#5b21b6] to-[#a21caf] text-white shadow-xl" data-testid="contest-hero">
          <img v-if="cover" :src="cover" alt="" class="absolute inset-y-0 right-0 h-full w-full object-cover md:w-3/4" />
          <div class="absolute inset-0 bg-gradient-to-r from-[#1e0b4b] via-[#2e1065]/90 to-[#2e1065]/10 md:via-[#2e1065]/80 md:to-transparent"></div>
          <svg class="pointer-events-none absolute inset-0 h-full w-full opacity-[0.12]" aria-hidden="true">
            <defs>
              <pattern id="contest-comb" width="56" height="97" patternUnits="userSpaceOnUse" patternTransform="scale(0.7)">
                <path d="M28 0 56 16v32L28 64 0 48V16zM28 64v33" fill="none" stroke="#fff" stroke-width="1.5" />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#contest-comb)" />
          </svg>

          <div class="relative grid gap-8 p-6 sm:p-10 lg:grid-cols-[minmax(0,1fr)_300px] lg:items-end">
            <div class="max-w-2xl">
              <div class="flex flex-wrap items-center gap-2">
                <span :class="phaseBadgeClass(contest.phase)">{{ t(`contests.phase.${contest.phase}`) }}</span>
                <span class="rounded-full bg-white/15 px-2.5 py-0.5 text-xs font-medium backdrop-blur">
                  {{ t('contests.hero.stats', { entries: contest.entry_count, votes: contest.vote_count }) }}
                </span>
              </div>
              <h1 class="mt-4 text-3xl font-black tracking-tight drop-shadow sm:text-5xl">{{ contest.title }}</h1>
              <p v-if="contest.description" class="mt-4 max-w-xl whitespace-pre-line text-sm leading-relaxed text-white/85 sm:text-base">{{ contest.description }}</p>

              <div v-if="countdown" class="mt-6" data-testid="contest-countdown">
                <div class="text-xs font-medium uppercase tracking-wider text-white/70">{{ t(countdown.labelKey) }}</div>
                <div class="mt-2 flex gap-2">
                  <div v-for="part in countdownBoxes" :key="part.unit" class="min-w-[64px] rounded-xl bg-white/12 px-3 py-2 text-center ring-1 ring-white/20 backdrop-blur">
                    <div class="text-2xl font-bold tabular-nums sm:text-3xl">{{ part.value }}</div>
                    <div class="text-[11px] text-white/70">{{ part.unit }}</div>
                  </div>
                </div>
              </div>

              <div class="mt-7 flex flex-wrap gap-3">
                <template v-if="submittingOpen">
                  <button
                    v-if="viewer.logged_in"
                    type="button"
                    class="inline-flex items-center gap-1.5 rounded-xl bg-white px-5 py-2.5 text-sm font-semibold text-violet-700 shadow-lg transition hover:bg-violet-50 disabled:opacity-60"
                    :disabled="!viewer.can_submit"
                    data-testid="contest-submit-open"
                    @click="openSubmit"
                  >🎨 {{ t('contests.submit.button') }}</button>
                  <RouterLink
                    v-else
                    :to="{ path: '/login', query: { redirect: route.fullPath } }"
                    class="inline-flex items-center gap-1.5 rounded-xl bg-white px-5 py-2.5 text-sm font-semibold text-violet-700 shadow-lg transition hover:bg-violet-50"
                  >🎨 {{ t('contests.viewer.loginToSubmit') }}</RouterLink>
                </template>
                <a
                  v-if="contest.phase !== 'settled' && contest.phase !== 'cancelled' && contest.phase !== 'tallying'"
                  :href="canvasUrl({ medium: 'contest-hero', path: '/image' })"
                  target="_blank"
                  rel="noopener noreferrer"
                  :class="[
                    'inline-flex items-center gap-1.5 rounded-xl px-5 py-2.5 text-sm font-semibold transition',
                    submittingOpen ? 'bg-white/10 text-white ring-1 ring-white/30 hover:bg-white/20' : 'bg-white text-violet-700 shadow-lg hover:bg-violet-50'
                  ]"
                  data-testid="contest-go-create"
                  @click="track('cta_click', { where: 'contest_hero_create' })"
                >✨ {{ t('contests.hero.create') }}</a>
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-xl bg-white/10 px-5 py-2.5 text-sm font-semibold text-white ring-1 ring-white/30 transition hover:bg-white/20"
                  data-testid="contest-share"
                  @click="shareOpen = true"
                >📣 {{ t('contests.hero.share') }}</button>
              </div>
              <p v-if="contest.phase === 'upcoming'" class="mt-3 text-xs text-white/75">{{ t('contests.hero.upcomingHint', { time: formatDateTime(contest.submission_start_at) }) }}</p>
            </div>

            <!-- First prize -->
            <div v-if="firstPrize" class="rounded-2xl bg-white/12 p-5 ring-1 ring-white/25 backdrop-blur-md" data-testid="contest-hero-prize">
              <div class="flex items-center gap-3">
                <span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-amber-300 to-amber-500 text-2xl shadow-lg">🏆</span>
                <div>
                  <div class="text-xs font-medium text-white/75">{{ formatPrizePlaces(firstPrize, t) }}</div>
                  <div class="text-lg font-bold leading-tight text-amber-200">{{ formatPrizeReward(firstPrize, t) }}</div>
                </div>
              </div>
              <p v-if="contest.min_votes_for_prize > 0" class="mt-3 text-xs leading-relaxed text-white/75">{{ t('contests.hero.prizeNote', { n: contest.min_votes_for_prize }) }}</p>
            </div>
          </div>
        </section>

        <!-- Timeline + how to take part -->
        <section class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <div class="card p-5 sm:p-6">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.timeline.title') }}</h2>
            <ol class="mt-4 space-y-0">
              <li v-for="(m, i) in milestones" :key="m.key" class="relative flex gap-3 pb-5 last:pb-0">
                <span v-if="i < milestones.length" class="absolute left-[9px] top-5 h-full w-px bg-gray-200 dark:bg-dark-700" :class="{ hidden: i === milestones.length }"></span>
                <span :class="['relative z-10 mt-0.5 flex h-[19px] w-[19px] flex-shrink-0 items-center justify-center rounded-full text-[10px] font-bold', m.done ? 'bg-primary-500 text-white' : nextMilestone === m.key ? 'bg-white ring-2 ring-primary-500 dark:bg-dark-800' : 'bg-gray-200 dark:bg-dark-700']">{{ m.done ? '✓' : '' }}</span>
                <div class="min-w-0">
                  <div :class="['text-sm font-medium', m.done ? 'text-gray-500 dark:text-dark-400' : 'text-gray-900 dark:text-white']">{{ t(`contests.timeline.${m.key}`) }}</div>
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(m.at) }}</div>
                </div>
              </li>
              <li class="relative flex gap-3">
                <span class="relative z-10 mt-0.5 h-[19px] w-[19px] flex-shrink-0 rounded-full bg-amber-100 text-center text-[11px] leading-[19px] dark:bg-amber-900/40">🎁</span>
                <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('contests.timeline.award') }}</div>
              </li>
            </ol>
          </div>

          <div class="card p-5 sm:p-6">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.howto.title') }}</h2>
            <ol class="mt-4 space-y-4">
              <li v-for="(step, i) in howto" :key="i" class="flex gap-3">
                <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl bg-primary-50 text-lg dark:bg-primary-900/30">{{ step.icon }}</span>
                <div class="min-w-0">
                  <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ i + 1 }}. {{ step.title }}</div>
                  <p class="mt-0.5 text-xs leading-relaxed text-gray-600 dark:text-dark-300">{{ step.text }}</p>
                </div>
              </li>
            </ol>
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
              <div v-if="entriesTotal > 0" class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
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

            <div v-if="entries.length === 0 && !entriesLoading" class="rounded-2xl border-2 border-dashed border-gray-200 bg-white px-6 py-12 text-center dark:border-dark-700 dark:bg-dark-800/60" data-testid="contest-empty">
              <div class="text-5xl">🖼️</div>
              <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('contests.noEntries.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                {{ contest.phase === 'upcoming' ? t('contests.noEntries.upcoming', { time: formatDateTime(contest.submission_start_at) }) : t('contests.noEntries.text') }}
              </p>
              <div class="mx-auto mt-6 max-w-2xl">
                <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('contests.noEntries.ideas') }}</div>
                <div class="mt-3 flex flex-wrap justify-center gap-2">
                  <a
                    v-for="idea in ideas"
                    :key="idea"
                    :href="canvasUrl({ medium: 'contest-idea', path: '/image', prompt: idea })"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="rounded-full border border-gray-200 bg-gray-50 px-3 py-1.5 text-xs text-gray-700 transition hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700 dark:border-dark-600 dark:bg-dark-900 dark:text-dark-200 dark:hover:border-primary-700 dark:hover:bg-primary-900/30"
                    data-testid="contest-idea"
                    @click="track('cta_click', { where: 'contest_idea' })"
                  >✨ {{ idea }}</a>
                </div>
              </div>
            </div>
            <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              <article
                v-for="e in entries"
                :key="e.id"
                class="group overflow-hidden rounded-2xl border border-gray-200/70 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg dark:border-dark-700 dark:bg-dark-800"
                data-testid="contest-entry"
              >
                <RouterLink :to="contestEntryPath(contestId, e.id)" class="relative block aspect-square w-full overflow-hidden bg-gray-100 dark:bg-dark-900" data-testid="contest-entry-link">
                  <img :src="e.image_url" :alt="e.title" loading="lazy" class="h-full w-full object-cover transition duration-500 group-hover:scale-105" />
                  <span class="absolute right-2 top-2 rounded-full bg-black/55 px-2 py-0.5 text-xs font-semibold text-white backdrop-blur">👍 {{ e.vote_count }}</span>
                  <span v-if="e.is_mine" class="absolute left-2 top-2 rounded-full bg-primary-500 px-2 py-0.5 text-[10px] font-semibold text-white">{{ t('contests.entries.mine') }}</span>
                </RouterLink>
                <div class="p-3">
                  <div class="flex items-center gap-2.5">
                    <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-500 text-xs font-bold text-white">{{ initial(e.author_name) }}</span>
                    <div class="min-w-0 flex-1">
                      <RouterLink :to="contestEntryPath(contestId, e.id)" class="block truncate text-sm font-semibold text-gray-900 hover:text-primary-600 dark:text-white" :title="e.title">{{ e.title }}</RouterLink>
                      <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('contests.entries.by', { name: e.author_name }) }}</p>
                    </div>
                    <button
                      v-if="votingOpen"
                      type="button"
                      :disabled="voteBusy === e.id || (!e.voted_by_me && !canCastNewVote(e))"
                      :class="['btn btn-sm flex-shrink-0', e.voted_by_me ? 'btn-secondary' : 'btn-primary']"
                      :title="voteButtonTitle(e)"
                      @click="toggleVote(e)"
                    >
                      {{ e.voted_by_me ? t('contests.entries.voted') : t('contests.entries.vote') }}
                    </button>
                  </div>
                  <a
                    v-if="e.work_id"
                    :href="canvasUrl({ medium: 'contest-entry', path: `/w/${e.work_id}` })"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="mt-2 inline-block text-xs text-primary-600 hover:underline dark:text-primary-400"
                    data-testid="contest-entry-work"
                  >{{ t('contests.entries.viewWork') }} ↗</a>
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
            <div v-if="votingOpen || submittingOpen" class="card p-5" data-testid="contest-action-card">
              <template v-if="!viewer.logged_in">
                <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('contests.viewer.loginPitch', { n: contest.votes_per_user }) }}</p>
                <RouterLink :to="{ path: '/login', query: { redirect: route.fullPath } }" class="btn btn-primary mt-3 w-full">
                  {{ votingOpen ? t('contests.viewer.loginToVote') : t('contests.viewer.loginToSubmit') }}
                </RouterLink>
              </template>
              <template v-else>
                <div v-if="votingOpen" class="text-center">
                  <template v-if="viewer.vote_blocked_reason === 'account_too_new'">
                    <span class="text-sm text-amber-600 dark:text-amber-400">{{ t('contests.viewer.accountTooNew', { hours: contest.min_account_age_hours }) }}</span>
                  </template>
                  <template v-else>
                    <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('contests.viewer.myVotes') }}</div>
                    <div class="mt-1 flex justify-center gap-1.5" :title="t('contests.viewer.votesLeft', { left: viewer.votes_left, total: contest.votes_per_user })">
                      <span v-for="i in contest.votes_per_user" :key="i" :class="['text-xl', i <= viewer.votes_left ? '' : 'opacity-25 grayscale']">👍</span>
                    </div>
                    <p class="mt-1 text-xs text-gray-500">{{ viewer.votes_left === 0 ? t('contests.viewer.noVotesLeft') : t('contests.viewer.votesLeft', { left: viewer.votes_left, total: contest.votes_per_user }) }}</p>
                  </template>
                </div>
                <p v-if="submittingOpen" :class="['text-center text-xs text-gray-500 dark:text-dark-400', votingOpen ? 'mt-3 border-t border-gray-100 pt-3 dark:border-dark-700' : '']">
                  {{ t('contests.submit.limit', { n: contest.max_entries_per_user, left: viewer.entries_left }) }}
                </p>
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
                  <RouterLink
                    v-if="e.status === 'approved' || e.status === 'pending'"
                    :to="{ path: contestEntryPath(contestId, e.id), query: { share: '1' } }"
                    class="flex-shrink-0 rounded-md bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-300"
                    data-testid="contest-my-entry-share"
                  >📣 {{ t('contests.entries.share') }}</RouterLink>
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

            <!-- Leaderboard -->
            <div class="card p-5">
              <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('contests.leaderboard.title') }}</h3>
                <span class="text-xs text-gray-500">{{ board[0]?.final ? t('contests.leaderboard.final') : t('contests.leaderboard.live') }}</span>
              </div>
              <div v-if="board.length === 0" class="mt-4 flex flex-col items-center py-4 text-center">
                <div class="flex items-end gap-2 opacity-60">
                  <span class="flex h-10 w-10 items-end justify-center rounded-t-lg bg-gray-200 pb-1 text-sm dark:bg-dark-700">🥈</span>
                  <span class="flex h-14 w-10 items-end justify-center rounded-t-lg bg-amber-100 pb-1 text-sm dark:bg-amber-900/30">🥇</span>
                  <span class="flex h-8 w-10 items-end justify-center rounded-t-lg bg-orange-100 pb-1 text-sm dark:bg-orange-900/20">🥉</span>
                </div>
                <p class="mt-3 text-xs text-gray-500">{{ t('contests.leaderboard.empty') }}</p>
              </div>
              <ol class="mt-3 space-y-2">
                <li v-for="row in board" :key="row.entry_id">
                  <RouterLink :to="contestEntryPath(contestId, row.entry_id)" class="flex items-center gap-3 rounded-lg p-1 text-sm transition hover:bg-gray-50 dark:hover:bg-dark-700/50">
                    <span :class="['w-6 flex-shrink-0 text-center font-bold', rankColor(row.rank)]">{{ row.rank <= 3 ? ['🥇', '🥈', '🥉'][row.rank - 1] : row.rank }}</span>
                    <img :src="row.image_url" alt="" :class="['flex-shrink-0 rounded object-cover', row.rank <= 3 ? 'h-10 w-10' : 'h-8 w-8']" />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-gray-800 dark:text-gray-200">{{ row.title }}</span>
                      <span class="block truncate text-xs text-gray-500">{{ row.author_name }}</span>
                    </span>
                    <span class="flex-shrink-0 font-semibold text-gray-900 dark:text-white">{{ row.votes }}</span>
                  </RouterLink>
                </li>
              </ol>
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
        <div class="inline-flex rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-900" role="tablist">
          <button
            v-for="s in (['upload', 'work'] as const)"
            :key="s"
            type="button"
            role="tab"
            :aria-selected="source === s"
            :class="['rounded-md px-3 py-1 transition', source === s ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200']"
            :data-testid="`contest-source-${s}`"
            @click="s === 'work' ? useWorks() : (source = 'upload')"
          >
            {{ t(s === 'work' ? 'contests.submit.fromWork' : 'contests.submit.fromUpload') }}
          </button>
        </div>
        <div v-if="source === 'work'" data-testid="contest-work-picker">
          <label class="input-label">{{ t('contests.submit.pickWork') }}</label>
          <div v-if="myWorks === null" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
          <div v-else-if="!myWorks.length" class="rounded-xl border border-dashed border-gray-300 py-8 text-center text-sm text-gray-500 dark:border-dark-600">
            <p>{{ worksError || t('contests.submit.noWorks') }}</p>
            <a :href="canvasUrl({ medium: 'contest-submit', path: '/image' })" target="_blank" rel="noopener noreferrer" class="mt-2 inline-block text-primary-600 hover:underline dark:text-primary-400">{{ t('contests.submit.goCreate') }} ↗</a>
          </div>
          <div v-else class="grid max-h-72 grid-cols-3 gap-2 overflow-y-auto pr-1 sm:grid-cols-4">
            <button
              v-for="w in myWorks"
              :key="w.id"
              type="button"
              :disabled="!!contestBlockReason(w)"
              :title="contestBlockReason(w) ? t(`contests.submit.blocked.${contestBlockReason(w)}`) : w.title"
              :class="['relative aspect-square overflow-hidden rounded-lg border-2 bg-gray-100 transition dark:bg-dark-900', pickedWork?.id === w.id ? 'border-primary-500 ring-2 ring-primary-500/30' : 'border-transparent hover:border-gray-300 dark:hover:border-dark-500', contestBlockReason(w) ? 'cursor-not-allowed opacity-40' : '']"
              data-testid="contest-work-option"
              @click="pickWork(w)"
            >
              <img :src="w.cover_thumb_url" :alt="w.title" loading="lazy" class="h-full w-full object-cover" />
              <span v-if="contestBlockReason(w)" class="absolute inset-x-0 bottom-0 bg-black/60 px-1 py-0.5 text-[10px] text-white">{{ t(`contests.submit.blocked.${contestBlockReason(w)}`) }}</span>
              <span v-else-if="w.image_count > 1" class="absolute right-1 top-1 rounded bg-black/55 px-1 text-[10px] text-white">{{ w.image_count }}</span>
            </button>
          </div>
          <p class="mt-1 text-xs text-gray-500">{{ t('contests.submit.workHint') }}</p>
        </div>
        <div v-else>
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
        <div v-if="source === 'upload'">
          <label class="input-label">{{ t('contests.submit.prompt') }}</label>
          <textarea v-model="form.prompt" rows="3" maxlength="4000" class="input font-mono text-xs" :placeholder="t('contests.submit.promptPlaceholder')"></textarea>
        </div>
        <p v-else class="text-xs text-gray-500">{{ t('contests.submit.workPromptHint') }}</p>
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

    <ShareCardDialog :show="shareOpen" :input="shareInput" @close="shareOpen = false" />

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
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { useAppStore } from '@/stores'
import {
  contestsAPI,
  type ContestDetail,
  type ContestEntry,
  type ContestEntrySort,
  type ContestLeaderboardRow,
  type ContestViewer
} from '@/api/contests'
import { contestBlockReason, myWorks as fetchMyWorks, type CommunityWork } from '@/api/community'
import { canvasUrl } from '@/constants/crossSites'
import {
  contestCountdown,
  contestCover,
  contestMilestones,
  countdownParts,
  formatDateTime,
  formatPrizePlaces,
  formatPrizeReward,
  phaseBadgeClass
} from '@/utils/contest'
import { track } from '@/utils/analytics'
import { contestEntryPath, contestListSeed, saveVoteIntent, takeVoteIntent } from '@/utils/contestShare'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
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
// While votes are open the list is shuffled per visitor, so early leaders don't take all the attention.
const sortBy = ref<ContestEntrySort>('random')
const sortOptions = computed(() => [
  ...(votingOpen.value ? [{ value: 'random' as const, label: t('contests.entries.sortRandom') }] : []),
  { value: 'votes' as const, label: t('contests.entries.sortVotes') },
  { value: 'new' as const, label: t('contests.entries.sortNew') }
])
const board = ref<ContestLeaderboardRow[]>([])

const voteBusy = ref(0)
const withdrawTarget = ref<ContestEntry | null>(null)

const submittingOpen = computed(() => ['submitting', 'submitting_voting'].includes(contest.value.phase))
const votingOpen = computed(() => ['voting', 'submitting_voting'].includes(contest.value.phase))

// Countdown ticks every 30s.
const nowTick = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
const countdown = computed(() => (detail.value ? contestCountdown(contest.value) : null))
const countdownBoxes = computed(() => {
  if (!countdown.value) return []
  const p = countdownParts(new Date(countdown.value.target).getTime() - nowTick.value)
  return [
    { unit: t('contests.hero.days'), value: p.days },
    { unit: t('contests.hero.hours'), value: String(p.hours).padStart(2, '0') },
    { unit: t('contests.hero.minutes'), value: String(p.minutes).padStart(2, '0') }
  ]
})
const cover = computed(() => (detail.value ? contestCover(contest.value) : ''))
const firstPrize = computed(() => contest.value.prizes.find((p) => p.rank_from === 1) || null)
const milestones = computed(() => contestMilestones(contest.value, nowTick.value))
const nextMilestone = computed(() => milestones.value.find((m) => !m.done)?.key || '')
const howto = computed(() => [
  { icon: '🎨', title: t('contests.howto.drawTitle'), text: t('contests.howto.drawText') },
  { icon: '📤', title: t('contests.howto.submitTitle'), text: t('contests.howto.submitText', { n: contest.value.max_entries_per_user }) },
  { icon: '📣', title: t('contests.howto.shareTitle'), text: t('contests.howto.shareText', { n: contest.value.votes_per_user }) }
])
const ideas = computed(() => [1, 2, 3, 4, 5, 6].map((i) => t(`contests.noEntries.idea${i}`)))

function initial(name: string): string {
  return (name || '?').trim().slice(0, 1).toUpperCase()
}

// ---- share the contest ----
const shareOpen = ref(false)
const shareInput = computed<ShareCardInput | null>(() => {
  if (!detail.value) return null
  const c = contest.value
  const points = [
    firstPrize.value ? `${formatPrizePlaces(firstPrize.value, t)}：${formatPrizeReward(firstPrize.value, t)}` : '',
    t('contests.shareCard.schedule', { start: formatDateTime(c.submission_start_at), end: formatDateTime(c.voting_end_at) }),
    t('contests.shareCard.how', { n: c.votes_per_user })
  ].filter(Boolean)
  return {
    kind: 'summary',
    brand: 'HiveGPT 创作比赛',
    label: t(`contests.phase.${c.phase}`),
    title: c.title,
    summary: c.description,
    points,
    cta: t('contests.shareCard.cta'),
    pageUrl: window.location.origin + `/contests/${c.id}`,
    source: 'contest_share',
    campaign: `contest-${c.id}`,
    trackKind: 'contest_page',
    dialogTitle: t('contests.hero.share'),
    pitch: (link) => t('contests.shareCard.pitch', { title: c.title, link })
  }
})

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
    const res = await contestsAPI.listEntries(contestId.value, {
      sort: sortBy.value,
      seed: sortBy.value === 'random' ? contestListSeed() : undefined,
      page,
      page_size: 24
    })
    entries.value = reset ? res.items : [...entries.value, ...res.items]
    entriesTotal.value = res.total
    entriesPage.value = page
  } finally {
    entriesLoading.value = false
  }
}

function changeSort(value: ContestEntrySort) {
  if (sortBy.value === value) return
  sortBy.value = value
  void loadEntries(true)
}

async function refreshAll() {
  await loadDetail()
  if (sortBy.value === 'random' && !votingOpen.value) sortBy.value = 'votes'
  await Promise.all([loadBoard(), loadEntries(true)])
}

async function toggleVote(e: ContestEntry) {
  if (!viewer.value.logged_in) {
    saveVoteIntent(contestId.value, e.id)
    track('contest_vote_login', { contest: contestId.value, where: 'list' })
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
      track('contest_vote', { contest: contestId.value, after_login: false, where: 'list' })
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

// Entering a canvas community work instead of uploading a file.
const source = ref<'upload' | 'work'>('upload')
const myWorks = ref<CommunityWork[] | null>(null)
const worksError = ref('')
const pickedWork = ref<CommunityWork | null>(null)

async function useWorks() {
  source.value = 'work'
  formError.value = ''
  if (myWorks.value !== null) return
  try {
    myWorks.value = await fetchMyWorks()
  } catch (err: any) {
    worksError.value = err?.message || t('common.error')
    myWorks.value = []
  }
}

function pickWork(w: CommunityWork) {
  if (contestBlockReason(w)) return
  pickedWork.value = w
  form.title = (w.title || form.title).slice(0, 120)
  form.description = w.description || ''
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
  if (source.value === 'work') {
    if (!pickedWork.value) {
      formError.value = t('contests.submit.workRequired')
      return
    }
  } else if (!file.value) {
    formError.value = t('contests.submit.imageRequired')
    return
  }
  if (!form.title) {
    formError.value = t('contests.submit.titleRequired')
    return
  }
  submitting.value = true
  try {
    const entry =
      source.value === 'work' && pickedWork.value
        ? await contestsAPI.submitWorkEntry(contestId.value, { work_id: pickedWork.value.id, title: form.title, description: form.description })
        : await contestsAPI.submitEntry(contestId.value, { ...form, image: file.value! })
    appStore.showSuccess(entry.status === 'pending' ? t('contests.entry.submittedPending') : t('contests.entry.submittedShare'))
    submitOpen.value = false
    Object.assign(form, { title: '', description: '', prompt: '' })
    pickedWork.value = null
    onFileChange({ target: { files: [] } } as unknown as Event)
    // Straight to the new entry's page with the 拉票 dialog open: sharing is how entries get votes.
    await router.push({ path: contestEntryPath(contestId.value, entry.id), query: { share: '1' } })
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
  // Back from sign-in after clicking 点赞 on a card: cast that vote now.
  const intent = detail.value && viewer.value.logged_in ? takeVoteIntent(contestId.value) : null
  if (intent && votingOpen.value && !viewer.value.voted_entry_ids.includes(intent.entryId)) {
    try {
      await contestsAPI.vote(contestId.value, intent.entryId)
      track('contest_vote', { contest: contestId.value, after_login: true, where: 'list' })
      appStore.showSuccess(t('contests.entry.votedAfterLogin'))
      await refreshAll()
    } catch (err: any) {
      appStore.showError(err?.message || t('common.error'))
    }
  }
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})
</script>
