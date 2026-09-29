<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar login-redirect="/contests" />
    <main class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div class="mb-8">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ t('contests.title') }}</h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ t('contests.subtitle') }}</p>
      </div>

      <div v-if="loading" class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="i in 3" :key="i" class="h-72 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      </div>

      <div v-else-if="contests.length === 0" class="card flex flex-col items-center py-16 text-center">
        <span class="text-5xl">🏆</span>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">{{ loadFailed ? t('contests.loadFailed') : t('contests.empty') }}</p>
      </div>

      <div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="c in contests"
          :key="c.id"
          :to="`/contests/${c.id}`"
          class="group overflow-hidden rounded-2xl border border-gray-200/70 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="relative h-40 overflow-hidden bg-gradient-to-br from-fuchsia-500 via-violet-500 to-amber-400">
            <img v-if="safeCover(c.cover_image)" :src="safeCover(c.cover_image)" alt="" class="h-full w-full object-cover" />
            <span v-else class="absolute inset-0 flex items-center justify-center text-6xl opacity-90">🎨</span>
            <span :class="['absolute left-3 top-3', phaseBadgeClass(c.phase)]">{{ t(`contests.phase.${c.phase}`) }}</span>
          </div>
          <div class="p-5">
            <h2 class="line-clamp-1 text-lg font-semibold text-gray-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
              {{ c.title }}
            </h2>
            <p class="mt-1 line-clamp-2 min-h-[2.5rem] text-sm text-gray-600 dark:text-dark-300">{{ c.description }}</p>
            <div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
              <span>{{ t('contests.stats.entries', { n: c.entry_count }) }}</span>
              <span>{{ t('contests.stats.votes', { n: c.vote_count }) }}</span>
              <span>{{ t('contests.schedule.deadline') }} {{ formatDateTime(c.voting_end_at) }}</span>
            </div>
            <div v-if="topPrize(c)" class="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
              🥇 {{ topPrize(c) }}
            </div>
          </div>
        </RouterLink>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { contestsAPI, type Contest } from '@/api/contests'
import { formatContestPrize, phaseBadgeClass, formatDateTime, safeCover } from '@/utils/contest'

const { t } = useI18n()
const contests = ref<Contest[]>([])
const loading = ref(true)
const loadFailed = ref(false)

function topPrize(c: Contest): string {
  const p = c.prizes?.[0]
  return p ? formatContestPrize(p, t) : ''
}

onMounted(async () => {
  try {
    contests.value = await contestsAPI.list()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})
</script>
