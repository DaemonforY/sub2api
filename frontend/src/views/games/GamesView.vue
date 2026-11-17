<template>
  <div class="min-h-screen bg-gradient-to-b from-primary-50/70 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/games" />

    <main class="mx-auto max-w-4xl px-4 pb-20 pt-8 sm:px-6">
      <section class="text-center">
        <h1 class="text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">🎮 {{ t('games.title') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-[15px] leading-relaxed text-gray-600 dark:text-dark-300">{{ t('games.subtitle') }}</p>
        <div class="mt-4 flex justify-center"><PageShareButton :input="shareInput" /></div>
      </section>

      <div class="mt-8 grid gap-5 sm:grid-cols-2">
        <component
          :is="g.href ? 'a' : RouterLink"
          v-for="g in GAMES"
          :key="g.key"
          v-bind="g.href ? { href: g.href } : { to: g.to }"
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
            <!-- open-source games credit their author; a nested <a> is not allowed inside the card link -->
            <span
              v-if="g.credit"
              role="link"
              tabindex="0"
              class="mt-2 block w-fit text-xs text-gray-500 underline decoration-dotted underline-offset-2 hover:text-primary-600 dark:text-dark-400"
              :data-testid="`game-${g.key}-credit`"
              @click.stop.prevent="openCredit(g.credit.href)"
              @keydown.enter.stop.prevent="openCredit(g.credit.href)"
              >{{ t('games.credit', { author: g.credit.author }) }}</span
            >
            <span class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.play') }} →</span>
          </div>
        </component>

        <div class="flex flex-col items-center justify-center rounded-3xl border border-dashed border-gray-300 p-8 text-center dark:border-dark-700 sm:col-span-2">
          <div class="text-3xl">🧩</div>
          <div class="mt-2 font-semibold text-gray-800 dark:text-dark-200">{{ t('games.soon') }}</div>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('games.soonDesc') }}</p>
          <button v-if="assistant?.enabled" type="button" class="btn btn-secondary mt-4" data-testid="games-ask" @click="askForGame">💬 {{ t('games.ask') }}</button>
        </div>
      </div>

      <!-- 站外精选: other people's work we link to (never re-hosted); they open in a new tab on the author's site. -->
      <section class="mt-14" data-testid="games-picks">
        <h2 class="text-xl font-bold text-gray-950 dark:text-white">✨ {{ t('games.picks.title') }}</h2>
        <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('games.picks.desc') }}</p>
        <div class="mt-5 grid gap-5 sm:grid-cols-2">
          <a
            v-for="p in PICKS"
            :key="p.key"
            :href="p.href"
            target="_blank"
            rel="noopener noreferrer"
            class="group overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/50"
            :data-testid="`pick-${p.key}`"
          >
            <div class="h-44 overflow-hidden"><img :src="p.image" alt="" class="h-full w-full object-cover" loading="lazy" /></div>
            <div class="p-5">
              <div class="flex flex-wrap items-center gap-2 text-xs">
                <span class="rounded-full bg-gray-100 px-2.5 py-0.5 font-semibold text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ t(`games.picks.list.${p.key}.tag`) }}</span>
                <span class="text-gray-500 dark:text-dark-400">{{ t('games.picks.by', { author: t(`games.picks.list.${p.key}.author`) }) }}</span>
              </div>
              <h3 class="mt-2 text-lg font-bold text-gray-950 group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ t(`games.picks.list.${p.key}.title`) }}</h3>
              <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t(`games.picks.list.${p.key}.desc`) }}</p>
              <span class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.picks.open') }} ↗</span>
            </div>
          </a>
        </div>
      </section>
    </main>

    <HomeAssistant ref="assistant" :site-name="siteName" />
  </div>
</template>

<script setup lang="ts">
// 小游戏 (/games): the list of small games and tests. Each game is its own route.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import HomeAssistant from '@/components/home/HomeAssistant.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import PageShareButton from '@/components/share/PageShareButton.vue'
import type { ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'HiveGPT')
const assistant = ref<InstanceType<typeof HomeAssistant> | null>(null)

// The chat is logged (后台「智能客服」记录), so what people ask for here is where the next game comes from.
function askForGame() {
  assistant.value?.openWith(t('games.askDraft'))
}

function openCredit(href: string) {
  window.open(href, '_blank', 'noopener,noreferrer')
}

// A game with `href` is a static page under public/games-play/ (its own HTML, not an SPA route).
const GAMES: { key: string; to?: string; href?: string; minutes: number; images: string[]; emoji: string; cover: string; credit?: { author: string; href: string } }[] = [
  {
    key: 'hutu',
    to: '/games/hutu',
    minutes: 15,
    images: ['/games-media/hutu/cover.webp'],
    emoji: '',
    cover: ''
  },
  {
    key: 'island',
    to: '/games/island',
    minutes: 15,
    images: ['/games-media/island/cover.webp'],
    emoji: '',
    cover: ''
  },
  {
    key: 'cotton',
    to: '/games/cotton',
    minutes: 15,
    images: ['/games-media/cotton/cover.webp'],
    emoji: '',
    cover: ''
  },
  {
    key: 'rustedcannon',
    href: '/games-play/rusted-cannon/',
    minutes: 15,
    images: ['/games-media/rusted-cannon/cover.webp'],
    emoji: '',
    cover: '',
    credit: { author: 'Cyril Dieumegard', href: 'https://github.com/CyrilDieumegard/rusted-cannon' }
  },
  {
    key: 'planet',
    to: '/games/planet',
    minutes: 10,
    images: ['/games-media/planet/cover.webp'],
    emoji: '',
    cover: ''
  },
  {
    key: 'craft',
    to: '/games/craft',
    minutes: 10,
    images: [],
    emoji: '🪙💧🔥',
    cover: 'bg-gradient-to-br from-violet-950 via-indigo-900 to-fuchsia-900'
  },
  {
    key: 'ink',
    href: '/games-play/ink/',
    minutes: 5,
    images: ['/games-media/ink/cover.webp'],
    emoji: '',
    cover: ''
  },
  {
    key: 'fps',
    href: '/games-play/hivegpt-3a/',
    minutes: 10,
    images: ['/games-media/hivegpt-3a/cover.webp'],
    emoji: '',
    cover: ''
  },
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
    images: [],
    emoji: '✍️',
    cover: 'bg-gradient-to-br from-amber-100 via-orange-100 to-rose-100 dark:from-amber-500/10 dark:via-orange-500/10 dark:to-rose-500/10'
  }
]

// Other people's browser games, linked on the author's own site. Their licenses don't allow re-hosting
// (雾锁山门's source says so; the UI8 Forge demo has no license), so only the cover lives here.
const PICKS: { key: string; href: string; image: string }[] = [
  { key: 'wusuo', href: 'https://op7418.github.io/guizang-game-skill/', image: '/games-media/wusuo/cover.webp' },
  { key: 'fly', href: 'https://remix.ui8.ai/fly', image: '/games-media/fly/cover.webp' }
]

const shareInput = computed<ShareCardInput>(() => ({
  kind: 'summary',
  brand: t('games.share.brand'),
  title: t('games.share.title'),
  summary: t('games.subtitle'),
  points: GAMES.map((g) => `${t(`games.list.${g.key}.title`)}：${t(`games.list.${g.key}.desc`)}`),
  cta: t('games.share.cta')
}))

onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings().catch(() => undefined)
})
</script>
