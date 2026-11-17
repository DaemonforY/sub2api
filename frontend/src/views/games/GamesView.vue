<template>
  <div class="min-h-screen bg-gradient-to-b from-primary-50/70 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/games" />

    <main class="mx-auto max-w-6xl px-4 pb-20 pt-6 sm:px-6">
      <!-- Featured: a rotating banner over each game's key art -->
      <section
        class="relative overflow-hidden rounded-3xl bg-gray-950 shadow-xl ring-1 ring-black/5"
        data-testid="games-featured"
        @mouseenter="paused = true"
        @mouseleave="paused = false"
      >
        <div class="relative h-[300px] sm:h-[360px]">
          <img
            v-for="(f, i) in FEATURED"
            :key="f.key"
            :src="f.art"
            alt=""
            :loading="i === 0 ? 'eager' : 'lazy'"
            :class="['absolute inset-0 h-full w-full object-cover transition-opacity duration-700', i === slide ? 'opacity-100' : 'opacity-0']"
            :style="{ objectPosition: f.pos }"
          />
          <div class="absolute inset-0 bg-gradient-to-r from-black/85 via-black/45 to-transparent"></div>
          <div class="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent"></div>

          <div class="relative flex h-full max-w-xl flex-col justify-end p-6 sm:p-10">
            <p class="text-xs font-semibold tracking-[0.25em] text-amber-300">🎮 {{ t('games.title') }} · {{ featuredKicker }}</p>
            <h1 class="mt-2 text-3xl font-black tracking-tight text-white sm:text-5xl">{{ t(`games.list.${current.key}.title`) }}</h1>
            <p class="mt-3 line-clamp-3 text-sm leading-relaxed text-white/85 sm:text-[15px]">{{ t(`games.list.${current.key}.desc`) }}</p>
            <div class="mt-5 flex flex-wrap items-center gap-3">
              <component
                :is="current.href ? 'a' : RouterLink"
                v-bind="current.href ? { href: current.href } : { to: current.to }"
                class="inline-flex items-center gap-1.5 rounded-full bg-amber-400 px-5 py-2.5 text-sm font-bold text-amber-950 shadow-lg shadow-amber-500/30 transition hover:bg-amber-300"
                data-testid="games-featured-play"
              >
                ▶ {{ t('games.play') }}
              </component>
              <span class="text-xs text-white/70">{{ t('games.stats', { n: GAMES.length }) }}</span>
            </div>
          </div>

          <div class="absolute bottom-4 right-5 flex gap-1.5">
            <button
              v-for="(f, i) in FEATURED"
              :key="f.key"
              type="button"
              :aria-label="t(`games.list.${f.key}.title`)"
              :class="['h-1.5 rounded-full transition-all', i === slide ? 'w-6 bg-white' : 'w-1.5 bg-white/45 hover:bg-white/70']"
              @click="slide = i"
            ></button>
          </div>
        </div>
      </section>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
        <p class="max-w-3xl text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t('games.subtitle') }}</p>
        <PageShareButton :input="shareInput" />
      </div>

      <!-- Category chips -->
      <nav class="sticky top-[64px] z-20 -mx-4 mt-5 bg-white/80 px-4 py-3 backdrop-blur-md dark:bg-dark-950/80 sm:-mx-6 sm:px-6" data-testid="games-cats">
        <div class="flex gap-2 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <button
            v-for="c in chips"
            :key="c.key"
            type="button"
            :class="[
              'shrink-0 whitespace-nowrap rounded-full border px-4 py-1.5 text-sm font-medium transition',
              filter === c.key
                ? 'border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-gray-900'
                : 'border-gray-200 bg-white text-gray-700 hover:border-gray-400 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200'
            ]"
            :data-testid="`games-cat-${c.key}`"
            @click="filter = c.key"
          >
            <span class="mr-1">{{ c.emoji }}</span>{{ c.label }}<span class="ml-1 text-xs opacity-60">{{ c.count }}</span>
          </button>
        </div>
      </nav>

      <!-- Sections -->
      <section v-for="cat in visibleCats" :key="cat.key" class="mt-8" :data-testid="`games-section-${cat.key}`">
        <div class="mb-4 flex items-end justify-between gap-3">
          <div>
            <h2 class="text-xl font-bold text-gray-950 dark:text-white"><span class="mr-1.5">{{ cat.emoji }}</span>{{ t(`games.cats.${cat.key}.title`) }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t(`games.cats.${cat.key}.desc`) }}</p>
          </div>
        </div>
        <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <component
            :is="g.href ? 'a' : RouterLink"
            v-for="g in gamesIn(cat.key)"
            :key="g.key"
            v-bind="g.href ? { href: g.href } : { to: g.to }"
            class="group flex flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm transition duration-300 hover:-translate-y-1 hover:border-primary-300 hover:shadow-xl dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/50"
            :data-testid="`game-${g.key}`"
          >
            <div class="relative aspect-[16/9] overflow-hidden bg-gray-100 dark:bg-dark-800">
              <div v-if="g.images.length > 1" class="grid h-full grid-cols-4 gap-0.5">
                <img v-for="src in g.images" :key="src" :src="src" alt="" class="h-full w-full object-cover" loading="lazy" />
              </div>
              <img
                v-else-if="g.images.length"
                :src="g.images[0]"
                alt=""
                class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
                loading="lazy"
              />
              <div v-else :class="['flex h-full items-center justify-center text-6xl', g.cover]">{{ g.emoji }}</div>
              <div class="absolute left-3 top-3 flex flex-wrap gap-1.5 text-[11px] font-bold">
                <span v-if="g.isNew" class="rounded-full bg-rose-500 px-2 py-0.5 text-white shadow">{{ t('games.badges.new') }}</span>
                <span v-if="g.duo" class="rounded-full bg-amber-400 px-2 py-0.5 text-amber-950 shadow">👥 {{ t('games.badges.duo') }}</span>
                <span v-if="g.pc" class="rounded-full bg-black/60 px-2 py-0.5 text-white backdrop-blur">💻 {{ t('games.badges.pc') }}</span>
              </div>
              <span class="absolute bottom-2 right-2 rounded-full bg-black/55 px-2 py-0.5 text-[11px] text-white backdrop-blur">{{ t('games.minutes', { n: g.minutes }) }}</span>
            </div>
            <div class="flex flex-1 flex-col p-4">
              <span class="w-fit rounded-full bg-primary-100 px-2.5 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">{{ t(`games.list.${g.key}.tag`) }}</span>
              <h3 class="mt-2 text-lg font-bold text-gray-950 group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ t(`games.list.${g.key}.title`) }}</h3>
              <p class="mt-1 line-clamp-3 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t(`games.list.${g.key}.desc`) }}</p>
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
              <span class="mt-auto pt-3 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.play') }} →</span>
            </div>
          </component>
        </div>
      </section>

      <!-- 站外精选: other people's work we link to (never re-hosted); they open in a new tab on the author's site. -->
      <section v-if="filter === 'all' || filter === 'picks'" class="mt-10" data-testid="games-picks">
        <h2 class="text-xl font-bold text-gray-950 dark:text-white"><span class="mr-1.5">🌐</span>{{ t('games.picks.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('games.picks.desc') }}</p>
        <div class="mt-4 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <a
            v-for="p in PICKS"
            :key="p.key"
            :href="p.href"
            target="_blank"
            rel="noopener noreferrer"
            class="group flex flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm transition duration-300 hover:-translate-y-1 hover:border-primary-300 hover:shadow-xl dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/50"
            :data-testid="`pick-${p.key}`"
          >
            <div class="aspect-[16/9] overflow-hidden"><img :src="p.image" alt="" class="h-full w-full object-cover transition duration-500 group-hover:scale-105" loading="lazy" /></div>
            <div class="flex flex-1 flex-col p-4">
              <div class="flex flex-wrap items-center gap-2 text-xs">
                <span class="rounded-full bg-gray-100 px-2.5 py-0.5 font-semibold text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ t(`games.picks.list.${p.key}.tag`) }}</span>
                <span class="text-gray-500 dark:text-dark-400">{{ t('games.picks.by', { author: t(`games.picks.list.${p.key}.author`) }) }}</span>
              </div>
              <h3 class="mt-2 text-lg font-bold text-gray-950 group-hover:text-primary-700 dark:text-white dark:group-hover:text-primary-300">{{ t(`games.picks.list.${p.key}.title`) }}</h3>
              <p class="mt-1 line-clamp-3 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t(`games.picks.list.${p.key}.desc`) }}</p>
              <span class="mt-auto pt-3 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.picks.open') }} ↗</span>
            </div>
          </a>
        </div>
      </section>

      <div class="mt-10 flex flex-col items-center justify-center rounded-3xl border border-dashed border-gray-300 p-8 text-center dark:border-dark-700">
        <div class="text-3xl">🧩</div>
        <div class="mt-2 font-semibold text-gray-800 dark:text-dark-200">{{ t('games.soon') }}</div>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('games.soonDesc') }}</p>
        <button v-if="assistant?.enabled" type="button" class="btn btn-secondary mt-4" data-testid="games-ask" @click="askForGame">💬 {{ t('games.ask') }}</button>
      </div>
    </main>

    <HomeAssistant ref="assistant" :site-name="siteName" />
  </div>
</template>

<script setup lang="ts">
// 小游戏 (/games): every game, grouped by kind, under a rotating banner of the newest ones. Each game is
// its own route or a static page under public/games-play/.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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

type Cat = 'coop' | 'guofeng' | 'action' | 'ai'

interface Game {
  key: string
  cat: Cat
  to?: string
  href?: string
  minutes: number
  images: string[]
  emoji?: string
  cover?: string
  duo?: boolean
  pc?: boolean
  isNew?: boolean
  credit?: { author: string; href: string }
}

const CATS: { key: Cat; emoji: string }[] = [
  { key: 'coop', emoji: '👥' },
  { key: 'guofeng', emoji: '🏮' },
  { key: 'action', emoji: '⚔️' },
  { key: 'ai', emoji: '✨' }
]

const GAMES: Game[] = [
  { key: 'hutu', cat: 'coop', to: '/games/hutu', minutes: 15, images: ['/games-media/hutu/cover.webp'], duo: true, isNew: true },
  { key: 'island', cat: 'coop', to: '/games/island', minutes: 15, images: ['/games-media/island/cover.webp'], duo: true, isNew: true },
  { key: 'cotton', cat: 'coop', to: '/games/cotton', minutes: 15, images: ['/games-media/cotton/cover.webp'], duo: true, isNew: true },
  { key: 'sunyu', cat: 'guofeng', href: '/games-play/sunyu/', minutes: 30, images: ['/games-media/sunyu/cover.webp'], pc: true, isNew: true },
  { key: 'suxiu', cat: 'guofeng', href: '/games-play/suxiu/', minutes: 10, images: ['/games-media/suxiu/cover.webp'], pc: true, isNew: true },
  { key: 'ink', cat: 'guofeng', href: '/games-play/ink/', minutes: 5, images: ['/games-media/ink/cover.webp'] },
  {
    key: 'rustedcannon',
    cat: 'action',
    href: '/games-play/rusted-cannon/',
    minutes: 15,
    images: ['/games-media/rusted-cannon/cover.webp'],
    credit: { author: 'Cyril Dieumegard', href: 'https://github.com/CyrilDieumegard/rusted-cannon' }
  },
  { key: 'planet', cat: 'action', to: '/games/planet', minutes: 10, images: ['/games-media/planet/cover.webp'] },
  { key: 'fps', cat: 'action', href: '/games-play/hivegpt-3a/', minutes: 10, images: ['/games-media/hivegpt-3a/cover.webp'] },
  { key: 'craft', cat: 'ai', to: '/games/craft', minutes: 10, images: [], emoji: '🪙💧🔥', cover: 'bg-gradient-to-br from-violet-950 via-indigo-900 to-fuchsia-900' },
  { key: 'abti', cat: 'ai', to: '/games/abti', minutes: 3, images: ['prom', 'boss', 'fish', 'humn'].map((c) => `/games-media/abti/${c}.webp`) },
  {
    key: 'style',
    cat: 'ai',
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

// Banner: the newest games over their biggest art.
const FEATURED: { key: string; art: string; pos: string }[] = [
  { key: 'sunyu', art: '/games-play/sunyu/art/bg/title.jpg', pos: '70% 50%' },
  { key: 'cotton', art: '/games-media/cotton/keyart.webp', pos: '60% 40%' },
  { key: 'hutu', art: '/games-media/hutu/title.webp', pos: '50% 40%' },
  { key: 'island', art: '/games-media/island/keyart.webp', pos: '60% 45%' },
  { key: 'suxiu', art: '/games-media/suxiu/cover.webp', pos: '50% 50%' }
]

const slide = ref(0)
const paused = ref(false)
let timer = 0
const current = computed(() => GAMES.find((g) => g.key === FEATURED[slide.value].key) as Game)
const featuredKicker = computed(() => {
  const g = current.value
  return `${t(`games.cats.${g.cat}.title`)}${g.isNew ? ` · ${t('games.badges.newLong')}` : ''}`
})

const filter = ref<'all' | Cat | 'picks'>('all')
const chips = computed(() => [
  { key: 'all' as const, emoji: '🎮', label: t('games.all'), count: GAMES.length + PICKS.length },
  ...CATS.map((c) => ({ key: c.key, emoji: c.emoji, label: t(`games.cats.${c.key}.title`), count: GAMES.filter((g) => g.cat === c.key).length })),
  { key: 'picks' as const, emoji: '🌐', label: t('games.picks.title'), count: PICKS.length }
])
const visibleCats = computed(() => (filter.value === 'all' ? CATS : CATS.filter((c) => c.key === filter.value)))
const gamesIn = (cat: Cat) => GAMES.filter((g) => g.cat === cat)

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
  const still = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  if (!still) {
    timer = window.setInterval(() => {
      if (!paused.value && !document.hidden) slide.value = (slide.value + 1) % FEATURED.length
    }, 6000)
  }
})

onBeforeUnmount(() => window.clearInterval(timer))
</script>
