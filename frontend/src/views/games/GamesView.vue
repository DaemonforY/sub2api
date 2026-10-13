<template>
  <div class="min-h-screen bg-gradient-to-b from-primary-50/70 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/games" />

    <main class="mx-auto max-w-4xl px-4 pb-20 pt-8 sm:px-6">
      <section class="text-center">
        <h1 class="text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">🎮 {{ t('games.title') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-[15px] leading-relaxed text-gray-600 dark:text-dark-300">{{ t('games.subtitle') }}</p>
      </section>

      <div class="mt-8 grid gap-5 sm:grid-cols-2">
        <RouterLink
          v-for="g in GAMES"
          :key="g.key"
          :to="g.to"
          class="group overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/50"
          :data-testid="`game-${g.key}`"
        >
          <div :class="['grid h-44 gap-1 overflow-hidden', g.images.length > 1 ? 'grid-cols-4' : '']">
            <img v-for="src in g.images" :key="src" :src="src" alt="" :class="['h-full w-full object-cover', g.images.length > 1 ? '' : 'object-center']" loading="lazy" />
            <div v-if="!g.images.length" :class="['flex h-full items-center justify-center text-6xl', g.cover]">{{ g.emoji }}</div>
          </div>
          <div class="p-5">
            <div class="flex items-center gap-2 text-xs">
              <span class="rounded-full bg-primary-100 px-2.5 py-0.5 font-semibold text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">{{ t(`games.list.${g.key}.tag`) }}</span>
              <span class="text-gray-500 dark:text-dark-400">{{ t('games.minutes', { n: g.minutes }) }}</span>
            </div>
            <h2 class="mt-2 text-lg font-bold text-gray-950 group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ t(`games.list.${g.key}.title`) }}</h2>
            <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t(`games.list.${g.key}.desc`) }}</p>
            <span class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.play') }} →</span>
          </div>
        </RouterLink>

        <div class="flex flex-col items-center justify-center rounded-3xl border border-dashed border-gray-300 p-8 text-center dark:border-dark-700 sm:col-span-2">
          <div class="text-3xl">🧩</div>
          <div class="mt-2 font-semibold text-gray-800 dark:text-dark-200">{{ t('games.soon') }}</div>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('games.soonDesc') }}</p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
// 小游戏 (/games): the list of small games and tests. Each game is its own route.
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const GAMES = [
  {
    key: 'abti',
    to: '/games/abti',
    minutes: 3,
    images: ['prom', 'boss', 'fish', 'humn'].map((c) => `/games-media/abti/${c}.webp`),
    emoji: '',
    cover: ''
  },
  {
    key: 'style',
    to: '/style',
    minutes: 5,
    images: [] as string[],
    emoji: '✍️',
    cover: 'bg-gradient-to-br from-amber-100 via-orange-100 to-rose-100 dark:from-amber-500/10 dark:via-orange-500/10 dark:to-rose-500/10'
  }
]

onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings().catch(() => undefined)
})
</script>
