<template>
  <div class="min-h-screen bg-gradient-to-b from-amber-50/60 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/style" />

    <main class="mx-auto max-w-2xl px-4 pb-20 pt-8 sm:px-6">
      <!-- 标题 -->
      <section class="text-center">
        <span class="inline-flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-800 dark:bg-amber-500/15 dark:text-amber-300">✍️ {{ t('style.badge') }}</span>
        <h1 class="mt-3 text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">{{ t('style.title') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-[15px] leading-relaxed text-gray-600 dark:text-dark-300">{{ t('style.subtitle') }}</p>
        <ol class="mt-5 flex flex-wrap justify-center gap-2 text-xs text-gray-600 dark:text-dark-300">
          <li v-for="(k, i) in ['s1', 's2', 's3']" :key="k" class="flex items-center gap-1.5 rounded-full border border-gray-200 bg-white px-3 py-1 dark:border-dark-700 dark:bg-dark-900">
            <span class="flex h-4 w-4 items-center justify-center rounded-full bg-gray-900 text-[10px] font-bold text-white dark:bg-white dark:text-dark-900">{{ i + 1 }}</span>{{ t(`style.steps.${k}`) }}
          </li>
        </ol>
        <div class="mt-4 flex justify-center"><PageShareButton :input="pageShare" /></div>
      </section>

      <!-- 文章 -->
      <section class="card mt-8 p-5">
        <div class="flex items-baseline justify-between">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('style.input.title') }}</h2>
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.input.count', { n: articles.length, chars: totalChars }) }}</span>
        </div>

        <ul v-if="articles.length" class="mt-3 space-y-2" data-testid="style-articles">
          <li v-for="(a, i) in articles" :key="i" class="flex items-center gap-3 rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-dark-800">
            <span class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-lg bg-white text-xs font-semibold text-gray-600 ring-1 ring-gray-200 dark:bg-dark-900 dark:text-dark-300 dark:ring-dark-700">{{ i + 1 }}</span>
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ a.title || t('style.input.untitled') }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.input.chars', { n: a.text.length }) }}</div>
            </div>
            <button type="button" class="rounded-lg px-2 py-1 text-xs text-gray-500 hover:bg-gray-200 hover:text-red-600 dark:text-dark-400 dark:hover:bg-dark-700" @click="removeArticle(i)">{{ t('style.input.remove') }}</button>
          </li>
        </ul>

        <div class="mt-4 inline-flex rounded-xl bg-gray-100 p-1 text-sm dark:bg-dark-800">
          <button v-for="m in (['link', 'paste'] as const)" :key="m" type="button"
            :class="['rounded-lg px-3 py-1.5 transition', mode === m ? 'bg-white font-semibold text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-400']"
            @click="mode = m">{{ m === 'link' ? t('style.input.tabLink') : t('style.input.tabPaste') }}</button>
        </div>

        <div v-if="mode === 'link'" class="mt-3">
          <div class="flex gap-2">
            <input v-model.trim="link" type="url" inputmode="url" class="input min-w-0 flex-1" :placeholder="t('style.input.linkPlaceholder')" :disabled="fetching || full" data-testid="style-link" @keydown.enter="addLink" />
            <button type="button" class="btn btn-primary flex-shrink-0" :disabled="!link || fetching || full" @click="addLink">{{ fetching ? t('style.input.adding') : t('style.input.add') }}</button>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('style.input.linkHint') }}</p>
        </div>
        <div v-else class="mt-3 space-y-2">
          <input v-model.trim="pasteTitle" class="input w-full" :placeholder="t('style.input.pasteTitle')" maxlength="100" :disabled="full" />
          <textarea v-model="pasteText" rows="6" class="input w-full resize-y leading-relaxed" :placeholder="t('style.input.pastePlaceholder')" :disabled="full" data-testid="style-paste"></textarea>
          <button type="button" class="btn btn-primary w-full sm:w-auto" :disabled="!pasteText.trim() || full" @click="addPaste">{{ t('style.input.addPaste') }}</button>
        </div>

        <p v-if="inputError" class="mt-2 text-sm text-red-600 dark:text-red-400" data-testid="style-input-error">{{ inputError }}</p>
        <p v-else-if="full" class="mt-2 text-sm text-amber-700 dark:text-amber-400">{{ t('style.input.full') }}</p>
        <p class="mt-3 text-xs text-gray-400 dark:text-dark-500">🔒 {{ t('style.input.privacy') }}</p>
      </section>

      <!-- 基础分析（免费） -->
      <section v-if="stats" class="card mt-5 p-5" data-testid="style-stats">
        <div class="flex items-baseline justify-between">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('style.stats.title') }}</h2>
          <span class="text-xs text-emerald-600 dark:text-emerald-400">{{ t('style.stats.free') }}</span>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.avgSentence') }}</div>
            <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ t('style.stats.avgSentenceUnit', { n: stats.avgSentence }) }}</div>
            <div class="text-xs text-primary-600 dark:text-primary-400">{{ t(`style.stats.rhythm.${stats.rhythm}`) }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.shortPct') }}</div>
            <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ stats.shortPct }}%</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.avgParagraph') }}</div>
            <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ t('style.stats.avgSentenceUnit', { n: stats.avgParagraph }) }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.questions') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ t('style.stats.perK', { n: stats.questions }) }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.voice') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ voiceLabel }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('style.stats.aiFlavor') }}</div>
            <div :class="['mt-1 text-sm font-semibold', stats.aiLevel === 'low' ? 'text-emerald-600 dark:text-emerald-400' : stats.aiLevel === 'some' ? 'text-amber-600 dark:text-amber-400' : 'text-red-600 dark:text-red-400']">{{ t(`style.stats.aiLevel.${stats.aiLevel}`) }}</div>
          </div>
        </div>

        <div class="mt-4">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('style.stats.catchphrases') }}</div>
          <div v-if="stats.catchphrases.length" class="mt-2 flex flex-wrap gap-1.5">
            <span v-for="c in stats.catchphrases" :key="c.text" class="rounded-full bg-primary-50 px-2.5 py-1 text-sm text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">「{{ c.text }}」<span class="text-xs opacity-60">{{ t('style.stats.times', { n: c.count }) }}</span></span>
          </div>
          <p v-else class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('style.stats.catchphrasesNone') }}</p>
        </div>
        <div class="mt-3">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('style.stats.aiFlavor') }}</div>
          <div v-if="stats.aiWords.length" class="mt-2 flex flex-wrap gap-1.5">
            <span v-for="w in stats.aiWords" :key="w.text" class="rounded-full bg-red-50 px-2.5 py-1 text-sm text-red-700 dark:bg-red-500/10 dark:text-red-300">{{ w.text }} <span class="text-xs opacity-60">{{ t('style.stats.times', { n: w.count }) }}</span></span>
          </div>
          <p v-else class="mt-1 text-sm text-emerald-700 dark:text-emerald-400">{{ t('style.stats.aiWordsNone') }}</p>
        </div>
      </section>

      <!-- AI 风格画像 -->
      <section class="card mt-5 p-5" data-testid="style-ai">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">✨ {{ t('style.ai.title') }}</h2>
        <p class="mt-2 text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ t('style.ai.desc') }}</p>

        <template v-if="!authStore.isAuthenticated">
          <RouterLink :to="{ path: '/login', query: { redirect: '/style' } }" class="btn btn-primary mt-4 w-full justify-center py-3 text-base" data-testid="style-login">{{ t('style.ai.loginCta') }}</RouterLink>
          <p class="mt-2 text-center text-xs text-gray-500 dark:text-dark-400">{{ t('style.ai.loginHint') }}</p>
        </template>

        <template v-else>
          <div v-if="noKey" class="mt-4 rounded-xl bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
            {{ t('style.ai.noKey') }}
            <RouterLink to="/keys" class="ml-1 font-semibold underline">{{ t('style.ai.goKeys') }}</RouterLink>
          </div>
          <div v-else-if="keys.length > 1" class="mt-4 flex items-center gap-2 text-sm">
            <label for="style-key" class="flex-shrink-0 text-gray-500 dark:text-dark-400">{{ t('style.ai.key') }}</label>
            <select id="style-key" v-model.number="keyID" class="input min-w-0 flex-1 py-1.5">
              <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
            </select>
          </div>

          <div v-if="running" class="mt-4 rounded-xl bg-primary-50 p-4 dark:bg-primary-500/10">
            <div class="flex items-center gap-2 text-sm font-semibold text-primary-800 dark:text-primary-200">
              <span class="h-4 w-4 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>{{ t('style.ai.running') }}
            </div>
            <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-primary-100 dark:bg-primary-900/40">
              <div class="h-full rounded-full bg-primary-500 transition-all duration-500" :style="{ width: progressPct + '%' }"></div>
            </div>
            <div class="mt-2 flex items-center justify-between text-xs text-primary-700 dark:text-primary-300">
              <span>{{ t('style.ai.progress', { n: progress }) }}</span>
              <button type="button" class="underline" @click="stop">{{ t('style.ai.stop') }}</button>
            </div>
          </div>
          <template v-else>
            <button type="button" class="btn btn-primary mt-4 w-full justify-center py-3 text-base" :disabled="!canRun" data-testid="style-run" @click="run">
              {{ analysis ? t('style.ai.rerun') : t('style.ai.run') }}
            </button>
            <p class="mt-2 text-center text-xs text-gray-500 dark:text-dark-400">
              <template v-if="totalChars < 300">{{ t('style.ai.needArticles') }}</template>
              <template v-else-if="price !== null">{{ t('style.ai.price', { price: fmtUSD(price) }) }}</template>
              <template v-else>{{ t('style.ai.priceUnknown') }}</template>
            </p>
          </template>
          <div v-if="runError" class="mt-3 rounded-xl bg-red-50 p-3 text-sm text-red-700 dark:bg-red-500/10 dark:text-red-300" data-testid="style-error">
            {{ runError }}
            <RouterLink v-if="needsTopUp" to="/purchase" class="ml-1 font-semibold underline">{{ t('style.ai.recharge') }}</RouterLink>
          </div>
        </template>
      </section>

      <!-- 结果 -->
      <section v-if="analysis" class="mt-5 overflow-hidden rounded-2xl border border-amber-200 bg-white shadow-sm dark:border-amber-500/30 dark:bg-dark-900" data-testid="style-result">
        <div class="bg-gradient-to-br from-amber-400 via-orange-400 to-rose-400 px-5 py-6 text-white">
          <div class="text-xs font-medium opacity-90">{{ savedAt ? t('style.result.saved', { time: savedAt }) : t('style.result.title') }}</div>
          <div class="mt-1 text-2xl font-bold">{{ r.archetype || t('style.result.title') }}</div>
          <p class="mt-2 text-[15px] leading-relaxed">{{ r.portrait }}</p>
          <div v-if="r.tags.length" class="mt-3 flex flex-wrap gap-1.5">
            <span v-for="tag in r.tags" :key="tag" class="rounded-full bg-white/25 px-2.5 py-0.5 text-sm">#{{ tag }}</span>
          </div>
        </div>

        <div class="space-y-5 p-5 text-[15px] leading-relaxed text-gray-800 dark:text-dark-200">
          <div v-if="r.voice"><h3 class="style-h">{{ t('style.result.voice') }}</h3><p>{{ r.voice }}</p></div>
          <div v-if="r.rhythm"><h3 class="style-h">{{ t('style.result.rhythm') }}</h3><p>{{ r.rhythm }}</p></div>
          <div v-if="r.openings.length"><h3 class="style-h">{{ t('style.result.openings') }}</h3><ul class="style-list"><li v-for="x in r.openings" :key="x">{{ x }}</li></ul></div>
          <div v-if="r.endings.length"><h3 class="style-h">{{ t('style.result.endings') }}</h3><ul class="style-list"><li v-for="x in r.endings" :key="x">{{ x }}</li></ul></div>
          <div v-if="r.catchphrases.length">
            <h3 class="style-h">{{ t('style.result.catchphrases') }}</h3>
            <div class="flex flex-wrap gap-1.5"><span v-for="x in r.catchphrases" :key="x" class="rounded-full bg-primary-50 px-2.5 py-1 text-sm text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">「{{ x }}」</span></div>
          </div>
          <div v-if="r.strengths.length"><h3 class="style-h">{{ t('style.result.strengths') }}</h3><ul class="style-list"><li v-for="x in r.strengths" :key="x">{{ x }}</li></ul></div>
          <div v-if="r.suggestions.length"><h3 class="style-h">{{ t('style.result.suggestions') }}</h3><ul class="style-list"><li v-for="x in r.suggestions" :key="x">{{ x }}</li></ul></div>
          <div v-if="r.avoid.length">
            <h3 class="style-h">{{ t('style.result.avoid') }}</h3>
            <div class="flex flex-wrap gap-1.5"><span v-for="x in r.avoid" :key="x" class="rounded-full bg-red-50 px-2.5 py-1 text-sm text-red-700 line-through decoration-red-300 dark:bg-red-500/10 dark:text-red-300">{{ x }}</span></div>
          </div>
          <div v-if="r.samples.length">
            <h3 class="style-h">{{ t('style.result.samples') }}</h3>
            <blockquote v-for="x in r.samples" :key="x" class="mb-2 border-l-4 border-amber-300 pl-3 text-gray-600 dark:text-dark-300">{{ x }}</blockquote>
          </div>

          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
            <h3 class="style-h">{{ t('style.result.prompt') }}</h3>
            <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('style.result.promptHint') }}</p>
            <pre class="max-h-64 overflow-auto whitespace-pre-wrap break-words font-sans text-sm text-gray-700 dark:text-dark-200" data-testid="style-prompt">{{ r.prompt }}</pre>
          </div>

          <div class="grid grid-cols-2 gap-2">
            <button type="button" class="btn btn-primary justify-center" @click="copyPrompt">{{ copied ? t('style.result.copied') : t('style.result.copy') }}</button>
            <button type="button" class="btn btn-secondary justify-center" data-testid="style-share" @click="shareOpen = true">{{ t('style.result.share') }}</button>
          </div>
          <p v-if="analysis.cost !== undefined || analysis.prompt_tokens" class="text-center text-xs text-gray-400 dark:text-dark-500">
            {{ t('style.ai.cost', { cost: fmtUSD(analysis.cost ?? 0), tokens: analysis.prompt_tokens + analysis.completion_tokens, model: analysis.model }) }}
          </p>

          <div class="rounded-xl border border-dashed border-gray-300 p-4 dark:border-dark-700">
            <div class="font-semibold text-gray-900 dark:text-white">{{ t('style.result.next') }}</div>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('style.result.nextDesc') }}</p>
            <a href="/editor/" class="btn btn-secondary mt-3 inline-flex">{{ t('style.result.openEditor') }}</a>
          </div>
        </div>
      </section>
    </main>

    <ShareCardDialog :show="shareOpen" :input="shareInput" @close="shareOpen = false" />
  </div>
</template>

<script setup lang="ts">
// 测测你的写作风格 (/style): articles in, free local stats, then an AI portrait on the reader's own key.
// Articles and the last result are kept in localStorage, so logging in half-way loses nothing.
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import PageShareButton from '@/components/share/PageShareButton.vue'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { analyze, fetchArticle, quote, type StyleAnalysis, type StyleArticle } from '@/api/style'
import { listKeys } from '@/api/tutors'
import { keysAPI } from '@/api/keys'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { track } from '@/utils/analytics'
import { analyzeWriting } from '@/utils/writingStats'

const ARTICLES_KEY = 'style_articles'
const RESULT_KEY = 'style_result'
const MAX_ARTICLES = 5
const MIN_PASTE = 100
const MAX_TEXT = 6000
const EXPECTED_OUTPUT = 2000 // characters of JSON the AI usually writes, for the progress bar

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

function load<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}

const articles = ref<StyleArticle[]>(load<StyleArticle[]>(ARTICLES_KEY, []).filter((a) => a && typeof a.text === 'string').slice(0, MAX_ARTICLES))
const mode = ref<'link' | 'paste'>('link')
const link = ref('')
const pasteTitle = ref('')
const pasteText = ref('')
const fetching = ref(false)
const inputError = ref('')

const totalChars = computed(() => articles.value.reduce((n, a) => n + Math.min(a.text.length, MAX_TEXT), 0))
const full = computed(() => articles.value.length >= MAX_ARTICLES)
const stats = computed(() => (articles.value.length ? analyzeWriting(articles.value.map((a) => a.text)) : null))

watch(articles, (v) => localStorage.setItem(ARTICLES_KEY, JSON.stringify(v)), { deep: true })

function errMessage(err: unknown): string {
  return (err as { message?: string })?.message || String(err)
}

function addArticle(a: StyleArticle): boolean {
  const text = a.text.trim()
  if (articles.value.some((x) => x.text.trim() === text)) {
    inputError.value = t('style.input.duplicate')
    return false
  }
  articles.value.push({ title: a.title.trim(), text })
  inputError.value = ''
  return true
}

async function addLink() {
  if (!link.value || fetching.value || full.value) return
  fetching.value = true
  inputError.value = ''
  try {
    const a = await fetchArticle(link.value)
    if (addArticle(a)) {
      link.value = ''
      track('style_fetch')
    }
  } catch (err) {
    inputError.value = errMessage(err)
  } finally {
    fetching.value = false
  }
}

function addPaste() {
  const text = pasteText.value.trim()
  if (text.replace(/\s+/g, '').length < MIN_PASTE) {
    inputError.value = t('style.input.tooShort')
    return
  }
  if (addArticle({ title: pasteTitle.value, text })) {
    pasteText.value = ''
    pasteTitle.value = ''
  }
}

function removeArticle(i: number) {
  articles.value.splice(i, 1)
  inputError.value = ''
}

const voiceLabel = computed(() => {
  const s = stats.value
  if (!s) return ''
  if (s.iRate >= 8 && s.youRate >= 5) return t('style.stats.voiceBoth')
  if (s.youRate >= 5 && s.youRate > s.iRate) return t('style.stats.voiceYou')
  if (s.iRate >= 5) return t('style.stats.voiceI')
  if (s.youRate >= 3) return t('style.stats.voiceYou')
  return t('style.stats.voiceNone')
})

// --- AI ---------------------------------------------------------------------------------------

interface KeyOption {
  id: number
  name: string
  group: string
}
const keys = ref<KeyOption[]>([])
const keyID = ref(0)
const noKey = ref(false)
const price = ref<number | null>(null)
const running = ref(false)
const progress = ref(0)
const runError = ref('')
const saved = load<{ analysis: StyleAnalysis; at: number } | null>(RESULT_KEY, null)
const analysis = ref<StyleAnalysis | null>(saved?.analysis?.result ? saved.analysis : null)
const savedAt = ref(saved?.at && analysis.value ? new Date(saved.at).toLocaleString() : '')
const copied = ref(false)
const shareOpen = ref(false)
let controller: AbortController | null = null

const r = computed(() => analysis.value!.result)
const needsTopUp = computed(() => /余额|balance|充值/i.test(runError.value))
const canRun = computed(() => keyID.value > 0 && totalChars.value >= 300 && !running.value)
const progressPct = computed(() => Math.min(95, 5 + (progress.value / EXPECTED_OUTPUT) * 90))

function fmtUSD(v: number): string {
  return v >= 0.1 ? v.toFixed(2) : v >= 0.01 ? v.toFixed(3) : v.toFixed(4)
}

async function loadKeys() {
  if (!authStore.isAuthenticated) return
  try {
    let list = await listKeys()
    if (!list.length) {
      // Most people arriving from an article have no key yet: make the starter key (pay-as-you-go GPT group).
      await keysAPI.starter()
      list = await listKeys()
    }
    keys.value = list
    noKey.value = !list.length
    if (!list.some((k) => k.id === keyID.value)) keyID.value = list[0]?.id || 0
  } catch (err) {
    runError.value = errMessage(err)
  }
}

let quoteTimer: ReturnType<typeof setTimeout> | undefined
watch([keyID, totalChars], () => {
  clearTimeout(quoteTimer)
  if (!keyID.value || totalChars.value < 300) {
    price.value = null
    return
  }
  quoteTimer = setTimeout(async () => {
    try {
      price.value = await quote(keyID.value, totalChars.value)
    } catch {
      price.value = null
    }
  }, 300)
})

async function run() {
  if (!canRun.value) return
  running.value = true
  progress.value = 0
  runError.value = ''
  controller = new AbortController()
  track('style_analyze', { articles: articles.value.length })
  try {
    const out = await analyze(keyID.value, articles.value, (n) => (progress.value = n), controller.signal)
    analysis.value = out
    savedAt.value = ''
    localStorage.setItem(RESULT_KEY, JSON.stringify({ analysis: out, at: Date.now() }))
  } catch (err) {
    if ((err as { name?: string })?.name !== 'AbortError') runError.value = errMessage(err)
  } finally {
    running.value = false
    controller = null
  }
}

function stop() {
  controller?.abort()
}

async function copyPrompt() {
  if (!analysis.value) return
  try {
    await navigator.clipboard.writeText(r.value.prompt)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = r.value.prompt
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
  }
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
  track('style_prompt_copy')
}

const shareInput = computed<ShareCardInput | null>(() => {
  if (!analysis.value) return null
  const points: string[] = []
  const s = stats.value
  if (s) points.push(t('style.share.rhythm', { n: s.avgSentence, label: t(`style.stats.rhythm.${s.rhythm}`) }))
  const phrases = r.value.catchphrases.slice(0, 3).map((x) => `「${x}」`).join('')
  if (phrases) points.push(t('style.share.catchphrases', { list: phrases }))
  if (s) points.push(t('style.share.aiFlavor', { level: t(`style.stats.aiLevel.${s.aiLevel}`) }))
  if (r.value.tags.length) points.push(r.value.tags.map((x) => `#${x}`).join(' '))
  return {
    kind: 'summary',
    brand: t('style.share.brand'),
    title: t('style.share.title', { archetype: r.value.archetype || '…' }),
    summary: r.value.portrait,
    points,
    cta: t('style.share.cta')
  }
})

// The page itself, before any result.
const pageShare = computed<ShareCardInput>(() => ({
  kind: 'summary',
  brand: t('style.share.brand'),
  title: t('style.title'),
  summary: t('style.subtitle'),
  points: (['s1', 's2', 's3'] as const).map((k, i) => `${i + 1}. ${t(`style.steps.${k}`)}`),
  cta: t('style.share.pageCta')
}))

watch(() => authStore.isAuthenticated, (v) => v && void loadKeys())
onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings().catch(() => undefined)
  void loadKeys()
})
onUnmounted(() => {
  controller?.abort()
  clearTimeout(quoteTimer)
})
</script>

<style scoped>
.style-h {
  @apply mb-1.5 text-sm font-semibold text-gray-500 dark:text-dark-400;
}
.style-list {
  @apply list-disc space-y-1 pl-5;
}
</style>
