<template>
  <div class="min-h-screen bg-gradient-to-b from-primary-50/70 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/games/abti" />

    <main class="mx-auto max-w-2xl px-4 pb-20 pt-6 sm:px-6">
      <!-- 开始 -->
      <section v-if="screen === 'intro'" data-testid="abti-intro">
        <div class="flex items-center justify-between">
          <RouterLink to="/games" class="text-sm text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white">← {{ t('games.title') }}</RouterLink>
          <PageShareButton :input="introShare" />
        </div>

        <div v-if="friendType" class="card mt-4 flex items-center gap-4 p-4" data-testid="abti-friend">
          <img :src="imageOf(friendType.code)" :alt="friendType.name" class="h-20 w-20 flex-shrink-0 rounded-2xl object-cover" />
          <div class="min-w-0">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('games.abti.friend') }}</div>
            <div class="mt-0.5 text-lg font-bold text-gray-950 dark:text-white">{{ friendType.code }} · {{ friendType.name }}</div>
            <div class="mt-0.5 text-sm text-gray-600 dark:text-dark-300">{{ friendType.intro }}</div>
            <div class="mt-1 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ t('games.abti.friendAsk') }}</div>
          </div>
        </div>

        <div class="mt-6 text-center">
          <span class="inline-flex items-center gap-1.5 rounded-full bg-primary-100 px-3 py-1 text-xs font-semibold text-primary-800 dark:bg-primary-500/15 dark:text-primary-300">🎮 {{ t('games.abti.badge') }}</span>
          <h1 class="mt-3 text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">{{ t('games.abti.title') }}</h1>
          <p class="mx-auto mt-3 max-w-xl text-[15px] leading-relaxed text-gray-600 dark:text-dark-300">{{ t('games.abti.subtitle') }}</p>
          <ul class="mt-5 flex flex-wrap justify-center gap-2 text-xs text-gray-600 dark:text-dark-300">
            <li v-for="k in (['questions', 'dims', 'types'] as const)" :key="k" class="rounded-full border border-gray-200 bg-white px-3 py-1 dark:border-dark-700 dark:bg-dark-900">{{ t(`games.abti.facts.${k}`) }}</li>
          </ul>
        </div>

        <div class="mx-auto mt-6 grid max-w-md grid-cols-4 gap-2">
          <img v-for="code in PREVIEW" :key="code" :src="imageOf(code)" alt="" class="aspect-square w-full rounded-2xl object-cover shadow-sm ring-1 ring-primary-100 dark:ring-dark-700" loading="lazy" />
        </div>

        <div class="mx-auto mt-8 max-w-md">
          <button type="button" class="btn btn-primary w-full justify-center py-3.5 text-base" data-testid="abti-start" @click="start">{{ t('games.abti.start') }}</button>
          <button v-if="saved" type="button" class="mt-3 w-full text-center text-sm text-primary-600 underline-offset-2 hover:underline dark:text-primary-400" data-testid="abti-last" @click="showSaved">
            {{ t('games.abti.lastResult', { code: saved.type.code, name: saved.type.name }) }}
          </button>
          <p class="mt-6 text-center text-xs leading-relaxed text-gray-400 dark:text-dark-500">{{ t('games.abti.disclaimer') }}</p>
        </div>
      </section>

      <!-- 答题 -->
      <section v-else-if="screen === 'quiz' && current" data-testid="abti-quiz">
        <div class="flex items-center justify-between text-sm">
          <button type="button" class="text-gray-500 hover:text-gray-800 disabled:opacity-30 dark:text-dark-400 dark:hover:text-white" :disabled="index === 0" @click="back">← {{ t('games.abti.quiz.back') }}</button>
          <span class="font-medium text-gray-600 dark:text-dark-300" data-testid="abti-progress">{{ t('games.abti.quiz.progress', { n: index + 1, total: visible.length }) }}</span>
          <button type="button" class="text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white" @click="screen = 'intro'">{{ t('games.abti.quiz.quit') }}</button>
        </div>
        <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-primary-100 dark:bg-dark-800">
          <div class="h-full rounded-full bg-primary-500 transition-all duration-300" :style="{ width: `${(index / visible.length) * 100}%` }"></div>
        </div>

        <Transition name="abti-q" mode="out-in">
          <div :key="current.id" class="card mt-6 p-5 sm:p-6">
            <span v-if="!current.dim" class="inline-block rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-semibold text-amber-800 dark:bg-amber-500/15 dark:text-amber-300">{{ t('games.abti.quiz.extra') }}</span>
            <h2 class="mt-1 text-lg font-semibold leading-relaxed text-gray-950 dark:text-white sm:text-xl" data-testid="abti-question">{{ current.text }}</h2>
            <div class="mt-5 space-y-2.5">
              <button
                v-for="(o, i) in current.options"
                :key="o.value"
                type="button"
                :class="[
                  'flex w-full items-center gap-3 rounded-xl border px-4 py-3.5 text-left text-[15px] leading-snug transition',
                  answers[current.id] === o.value
                    ? 'border-primary-500 bg-primary-50 font-semibold text-primary-800 ring-2 ring-primary-500/25 dark:bg-primary-500/15 dark:text-primary-200'
                    : 'border-gray-200 bg-white text-gray-800 hover:border-primary-300 hover:bg-primary-50/50 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:border-primary-500/50'
                ]"
                :data-q="current.id"
                data-testid="abti-option"
                @click="onOption($event, o.value)"
              >
                <span class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-xs font-bold text-gray-500 dark:bg-dark-800 dark:text-dark-400">{{ 'ABC'[i] }}</span>
                <span>{{ o.label }}</span>
              </button>
            </div>
          </div>
        </Transition>
      </section>

      <!-- 结果 -->
      <section v-else-if="screen === 'result' && result" data-testid="abti-result">
        <div class="overflow-hidden rounded-3xl border border-primary-200 bg-white shadow-sm dark:border-primary-500/30 dark:bg-dark-900">
          <div class="bg-gradient-to-br from-primary-500 via-violet-500 to-fuchsia-500 px-5 pb-5 pt-6 text-center text-white">
            <div class="text-xs font-semibold uppercase tracking-wider opacity-90">{{ t(`games.abti.result.kicker.${result.kind}`) }}</div>
            <div class="mt-1 text-5xl font-black tracking-wider" data-testid="abti-code">{{ result.type.code }}</div>
            <div class="mt-1 text-xl font-bold">{{ result.type.name }}</div>
            <div class="mt-2 inline-block rounded-full bg-white/20 px-3 py-0.5 text-xs">{{ matchText }}</div>
          </div>
          <img :src="imageOf(result.type.code)" :alt="result.type.name" class="mx-auto -mt-1 aspect-square w-full max-w-sm object-cover" />
          <div class="px-5 pb-6">
            <p class="text-center text-base font-semibold text-gray-900 dark:text-white">「{{ result.type.intro }}」</p>
            <p class="mt-3 text-[15px] leading-relaxed text-gray-700 dark:text-dark-200">{{ result.type.desc }}</p>
            <div class="mt-5 grid grid-cols-2 gap-2">
              <button type="button" class="btn btn-primary justify-center" data-testid="abti-share" @click="shareOpen = true">{{ t('games.abti.result.share') }}</button>
              <button type="button" class="btn btn-secondary justify-center" @click="start">{{ t('games.abti.result.restart') }}</button>
            </div>
          </div>
        </div>

        <!-- AI 打工建议 -->
        <div class="card mt-5 p-5" data-testid="abti-tip">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">💡 {{ t('games.abti.result.tip') }}</h2>
          <p class="mt-2 text-[15px] leading-relaxed text-gray-700 dark:text-dark-200">{{ result.type.tip }}</p>
          <button type="button" class="btn btn-primary mt-4 w-full justify-center py-3" data-testid="abti-cta" @click="openCta">{{ t(`games.abti.result.cta.${result.type.cta}`) }}</button>
          <RouterLink v-if="!authStore.isAuthenticated" :to="{ path: '/register', query: { redirect: route.path } }" class="mt-2 block text-center text-sm text-primary-600 hover:underline dark:text-primary-400" @click="trackCta('signup')">
            {{ signupBonus > 0 ? t('games.abti.result.signup', { n: signupBonus }) : t('games.abti.result.signupPlain') }}
          </RouterLink>
        </div>

        <!-- 维度 -->
        <div class="card mt-5 p-5">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('games.abti.result.dims') }}</h2>
          <div v-for="g in groups" :key="g.model" class="mt-4">
            <div class="text-xs font-semibold text-gray-400 dark:text-dark-500">{{ g.model }}</div>
            <div v-for="d in g.dims" :key="d.id" class="mt-2.5">
              <div class="flex items-center gap-3">
                <span class="w-20 flex-shrink-0 text-sm font-medium text-gray-800 dark:text-dark-200">{{ d.name }}</span>
                <div class="h-2 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
                  <div class="h-full rounded-full bg-gradient-to-r from-primary-400 to-fuchsia-500" :style="{ width: `${(result.scores[d.id] / 6) * 100}%` }"></div>
                </div>
                <span class="w-6 flex-shrink-0 text-right text-xs font-semibold text-primary-600 dark:text-primary-400">{{ t(`games.abti.result.level.${result.levels[d.id]}`) }}</span>
              </div>
              <p class="mt-1 pl-[5.75rem] text-xs leading-relaxed text-gray-500 dark:text-dark-400">{{ d.explain[result.levels[d.id]] }}</p>
            </div>
          </div>
        </div>

        <!-- 相近人格 -->
        <div class="card mt-5 p-5">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('games.abti.result.close') }}</h2>
          <ul class="mt-3 space-y-2">
            <li v-for="r in closest" :key="r.type.code" class="flex items-center gap-3 rounded-xl bg-gray-50 p-2.5 dark:bg-dark-800">
              <img :src="imageOf(r.type.code)" alt="" class="h-12 w-12 flex-shrink-0 rounded-xl object-cover" loading="lazy" />
              <div class="min-w-0 flex-1">
                <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ r.type.code }} · {{ r.type.name }}</div>
                <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ r.type.intro }}</div>
              </div>
              <span class="flex-shrink-0 text-sm font-semibold text-primary-600 dark:text-primary-400">{{ r.similarity }}%</span>
            </li>
          </ul>
        </div>

        <!-- 全部人格 -->
        <div class="card mt-5 p-5">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('games.abti.result.all') }}</h2>
          <div class="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-6">
            <div v-for="type in gallery" :key="type.code" :class="['rounded-xl p-1.5 text-center', type.code === result.type.code ? 'bg-primary-100 ring-2 ring-primary-500 dark:bg-primary-500/20' : '']">
              <img v-if="!isLocked(type)" :src="imageOf(type.code)" alt="" class="aspect-square w-full rounded-lg object-cover" loading="lazy" />
              <div v-else class="flex aspect-square w-full items-center justify-center rounded-lg bg-gray-100 text-2xl font-black text-gray-300 dark:bg-dark-800 dark:text-dark-600">?</div>
              <div class="mt-1 text-[11px] font-bold text-gray-800 dark:text-dark-200">{{ isLocked(type) ? '????' : type.code }}</div>
              <div class="truncate text-[11px] text-gray-500 dark:text-dark-400">{{ isLocked(type) ? t('games.abti.result.hiddenSlot') : type.name }}</div>
            </div>
          </div>
        </div>

        <div class="mt-6 flex justify-center gap-4 text-sm">
          <RouterLink to="/games" class="text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white">{{ t('games.abti.result.hall') }}</RouterLink>
        </div>
        <p class="mt-4 text-center text-xs leading-relaxed text-gray-400 dark:text-dark-500">{{ t('games.abti.disclaimer') }} {{ t('games.abti.aiArt') }}</p>
      </section>
    </main>

    <ShareCardDialog :show="shareOpen" :input="shareInput" @close="shareOpen = false" />
    <!-- Not while answering: the button would sit over the options on a phone. -->
    <HomeAssistant v-if="screen !== 'quiz'" :site-name="siteName" />
  </div>
</template>

<script setup lang="ts">
// ABTI · AI 打工人格测试 (/games/abti): a quiz scored in the browser. The result's code goes into the
// path (/games/abti/PROM) so a shared link or share-card QR shows friends what you got; answers are
// kept in localStorage and re-scored on load, nothing is sent to the server except analytics events.
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import HomeAssistant from '@/components/home/HomeAssistant.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import PageShareButton from '@/components/share/PageShareButton.vue'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { DIMENSIONS, FALLBACK_TYPE, GATE_FOLLOW_UP, GATE_QUESTION, HIDDEN_TYPE, TYPES, type CtaKind, type PersonaType, type Question } from '@/games/abti/data'
import { computeResult, findType, questionOrder, visibleQuestions, type AbtiResult, type Answers } from '@/games/abti/score'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { track } from '@/utils/analytics'
import { savePendingPrompt } from '@/utils/pendingPrompt'

const SAVE_KEY = 'abti_result'
const PREVIEW = ['PROM', 'FISH', 'BOSS', 'HUMN']
const ADVANCE_MS = 220

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

function loadSaved(): AbtiResult | null {
  try {
    const raw = localStorage.getItem(SAVE_KEY)
    const data = raw ? (JSON.parse(raw) as { answers?: Answers }) : null
    return data?.answers ? computeResult(data.answers) : null
  } catch {
    return null
  }
}

const screen = ref<'intro' | 'quiz' | 'result'>('intro')
const saved = ref<AbtiResult | null>(loadSaved())
const order = ref<Question[]>([])
const answers = ref<Answers>({})
const index = ref(0)
const result = ref<AbtiResult | null>(null)
const shareOpen = ref(false)
let advanceTimer: ReturnType<typeof setTimeout> | undefined

const visible = computed(() => visibleQuestions(order.value, answers.value))
const current = computed(() => visible.value[index.value])
const routeCode = computed(() => String(route.params.code || '').toUpperCase())
// Someone else's result in the link: show it above the start button.
const friendType = computed(() => (routeCode.value && routeCode.value !== saved.value?.type.code ? findType(routeCode.value) : undefined))
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'HiveGPT')
const signupBonus = computed(() => Number(appStore.cachedPublicSettings?.signup_bonus || 0))

function imageOf(code: string) {
  return `/games-media/abti/${code.toLowerCase()}.webp`
}

function start() {
  clearTimeout(advanceTimer)
  order.value = questionOrder()
  answers.value = {}
  index.value = 0
  screen.value = 'quiz'
  track('game_start', { game: 'abti' })
  if (route.params.code) void router.replace({ path: '/games/abti' })
  window.scrollTo({ top: 0 })
}

// A card that is sliding out (or already gone) must not answer the question that replaced it.
function onOption(e: MouseEvent, value: number) {
  if (screen.value === 'quiz' && (e.currentTarget as HTMLElement | null)?.dataset.q === current.value?.id) choose(value)
}

function choose(value: number) {
  const q = current.value
  if (!q) return
  const next = { ...answers.value, [q.id]: value }
  if (q.id === GATE_QUESTION.id && value !== 3) delete next[GATE_FOLLOW_UP.id]
  answers.value = next
  clearTimeout(advanceTimer)
  advanceTimer = setTimeout(() => {
    if (screen.value !== 'quiz') return
    if (index.value < visible.value.length - 1) index.value++
    else if (visible.value.every((x) => answers.value[x.id] !== undefined)) finish()
  }, ADVANCE_MS)
}

function back() {
  clearTimeout(advanceTimer)
  if (index.value > 0) index.value--
}

function finish() {
  const r = computeResult(answers.value)
  result.value = r
  saved.value = r
  try {
    localStorage.setItem(SAVE_KEY, JSON.stringify({ answers: answers.value, at: Date.now() }))
  } catch {
    // storage blocked: the result just isn't remembered
  }
  screen.value = 'result'
  track('game_finish', { game: 'abti', type: r.type.code, kind: r.kind })
  void router.replace({ path: `/games/abti/${r.type.code}` })
  window.scrollTo({ top: 0 })
}

function showSaved() {
  if (!saved.value) return
  result.value = saved.value
  screen.value = 'result'
  if (routeCode.value !== saved.value.type.code) void router.replace({ path: `/games/abti/${saved.value.type.code}` })
  window.scrollTo({ top: 0 })
}

const matchText = computed(() => {
  const r = result.value
  if (!r) return ''
  const best = { code: r.best.type.code, name: r.best.type.name, n: r.best.similarity }
  if (r.kind === 'hidden') return t('games.abti.result.hiddenMatch', best)
  if (r.kind === 'fallback') return t('games.abti.result.fallbackMatch', best)
  return t('games.abti.result.match', { n: r.best.similarity, exact: r.best.exact })
})

const groups = computed(() => {
  const out: { model: string; dims: typeof DIMENSIONS }[] = []
  for (const d of DIMENSIONS) {
    const g = out.find((x) => x.model === d.model)
    if (g) g.dims.push(d)
    else out.push({ model: d.model, dims: [d] })
  }
  return out
})

const closest = computed(() => {
  const r = result.value
  if (!r) return []
  return r.ranked.filter((x) => x.type.code !== r.type.code).slice(0, 3)
})

const gallery: PersonaType[] = [...TYPES, HIDDEN_TYPE, FALLBACK_TYPE]
function isLocked(type: PersonaType) {
  return (type === HIDDEN_TYPE || type === FALLBACK_TYPE) && result.value?.type.code !== type.code
}

// --- Next step -------------------------------------------------------------------------------------

const CTA_TARGETS: Record<CtaKind, { to: string; spa: boolean; auth: boolean }> = {
  image: { to: '/first-image', spa: true, auth: true },
  code: { to: '/learn/connect/?utm_source=games&utm_medium=abti', spa: false, auth: false },
  write: { to: '/editor/?utm_source=games&utm_medium=abti', spa: false, auth: false },
  learn: { to: '/learn/?utm_source=games&utm_medium=abti', spa: false, auth: false },
  plaza: { to: '/model-plaza', spa: true, auth: false },
  invite: { to: '/affiliate', spa: true, auth: true }
}

function trackCta(cta: string) {
  track('game_cta', { game: 'abti', type: result.value?.type.code || '', cta })
}

function openCta() {
  const type = result.value?.type
  if (!type) return
  const target = CTA_TARGETS[type.cta]
  trackCta(type.cta)
  if (type.cta === 'image') savePendingPrompt(t('games.abti.result.imagePrompt', { name: type.name, intro: type.intro }))
  if (target.auth && !authStore.isAuthenticated) {
    void router.push({ path: '/register', query: { redirect: target.to } })
  } else if (target.spa) {
    void router.push(target.to)
  } else {
    window.location.href = target.to
  }
}

// --- Share card ------------------------------------------------------------------------------------

const shareInput = computed<ShareCardInput | null>(() => {
  const r = result.value
  if (!r) return null
  // The three dimensions furthest from the middle say the most about you.
  const strongest = DIMENSIONS.filter((d) => r.levels[d.id] !== 'M')
    .sort((a, b) => Math.abs(r.scores[b.id] - 4) - Math.abs(r.scores[a.id] - 4))
    .slice(0, 3)
    .map((d) => `${d.name}${t(`games.abti.result.level.${r.levels[d.id]}`)}：${d.explain[r.levels[d.id]]}`)
  return {
    kind: 'summary',
    brand: t('games.abti.share.brand'),
    label: r.kind === 'normal' ? t('games.abti.share.match', { n: r.best.similarity }) : t(`games.abti.result.kicker.${r.kind}`),
    title: t('games.abti.share.title', { code: r.type.code, name: r.type.name }),
    summary: r.type.intro,
    points: strongest,
    cta: t('games.abti.share.cta')
  }
})

// The test itself, before taking it: a few of the types as a teaser.
const introShare = computed<ShareCardInput>(() => ({
  kind: 'summary',
  brand: t('games.abti.share.brand'),
  title: t('games.abti.share.introTitle'),
  summary: t('games.abti.subtitle'),
  points: ['PROM', 'BOSS', 'FISH', 'HUMN'].map((code) => {
    const type = findType(code)!
    return `${type.code} ${type.name}：${type.intro}`
  }),
  cta: t('games.abti.share.cta')
}))

// --- Route -----------------------------------------------------------------------------------------

function syncWithRoute() {
  const code = routeCode.value
  if (!code) return
  if (saved.value && saved.value.type.code === code) {
    result.value = saved.value
    screen.value = 'result'
  } else if (!findType(code)) {
    void router.replace({ path: '/games/abti' })
  }
}

watch(routeCode, (code, prev) => {
  // Back/forward between /games/abti and a result path.
  if (!code && prev && screen.value === 'result') screen.value = 'intro'
})

onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings().catch(() => undefined)
  syncWithRoute()
})
onUnmounted(() => clearTimeout(advanceTimer))
</script>

<style scoped>
.abti-q-enter-active,
.abti-q-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.abti-q-enter-from {
  opacity: 0;
  transform: translateX(16px);
}
.abti-q-leave-to {
  opacity: 0;
  transform: translateX(-16px);
}
/* choose() answers the current question, which is already the next one while this card leaves. */
.abti-q-leave-active {
  pointer-events: none;
}
</style>
