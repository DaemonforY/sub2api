<template>
  <section v-if="works.length" class="mb-20" data-testid="home-showcase">
    <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.showcase.title') }}</h2>
        <p class="mt-2 text-gray-600 dark:text-dark-300">{{ t('home.v2.showcase.subtitle') }}</p>
      </div>
      <router-link to="/contests" class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">
        {{ t('home.v2.showcase.more') }} →
      </router-link>
    </div>
    <div class="columns-2 gap-4 md:columns-3 lg:columns-4">
      <router-link
        v-for="w in works"
        :key="w.id"
        :to="`/contests/${w.contest_id}`"
        class="group relative mb-4 block break-inside-avoid overflow-hidden rounded-2xl bg-gray-100 dark:bg-dark-800"
      >
        <img :src="w.image_url" :alt="w.title" loading="lazy" class="w-full object-cover transition duration-500 group-hover:scale-105" />
        <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/75 via-black/30 to-transparent p-3 pt-10 text-white">
          <div class="truncate text-sm font-semibold">{{ w.title }}</div>
          <div class="mt-0.5 flex items-center justify-between gap-2 text-xs text-white/80">
            <span class="truncate">{{ w.author_name }}</span>
            <span class="flex-shrink-0">❤ {{ t('home.v2.showcase.votes', { n: w.vote_count }) }}</span>
          </div>
        </div>
        <span class="absolute left-3 top-3 max-w-[80%] truncate rounded-full bg-black/50 px-2.5 py-0.5 text-[11px] text-white backdrop-blur-sm">
          {{ contestTitles[w.contest_id] }}
        </span>
      </router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { contestsAPI, type ContestEntry } from '@/api/contests'

const { t } = useI18n()
const works = ref<ContestEntry[]>([])
const contestTitles = ref<Record<number, string>>({})

const MAX_CONTESTS = 3
const MAX_WORKS = 12

onMounted(async () => {
  try {
    const contests = (await contestsAPI.list()).filter((c) => c.entry_count > 0).slice(0, MAX_CONTESTS)
    contestTitles.value = Object.fromEntries(contests.map((c) => [c.id, c.title]))
    const pages = await Promise.all(
      contests.map((c) => contestsAPI.listEntries(c.id, { sort: 'votes', page: 1, page_size: MAX_WORKS }).catch(() => null))
    )
    works.value = pages
      .flatMap((p) => p?.items || [])
      .sort((a, b) => b.vote_count - a.vote_count)
      .slice(0, MAX_WORKS)
  } catch {
    works.value = [] // the section simply stays hidden
  }
})
</script>
