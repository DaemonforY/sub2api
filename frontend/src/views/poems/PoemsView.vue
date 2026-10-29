<template>
  <div class="min-h-screen bg-gradient-to-b from-amber-50/60 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar login-redirect="/poems" />

    <main class="mx-auto max-w-5xl px-4 pb-20 pt-8 sm:px-6">
      <section class="text-center">
        <h1 class="font-serif text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">📜 {{ t('poems.title') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-[15px] leading-relaxed text-gray-600 dark:text-dark-300">{{ t('poems.subtitle') }}</p>
      </section>

      <p v-if="config && !config.enabled" class="mt-10 text-center text-gray-500">{{ t('poems.empty') }}</p>
      <template v-else>
        <div class="mt-6 flex justify-center gap-1">
          <button
            v-for="tab in tabs"
            :key="tab"
            class="rounded-full px-4 py-1.5 text-sm font-medium"
            :class="active === tab ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800'"
            :data-testid="`poems-tab-${tab}`"
            @click="active = tab"
          >
            {{ t(`poems.tabs.${tab}`) }}
          </button>
        </div>

        <!-- 搜诗词 -->
        <section v-show="active === 'search'" class="mt-6">
          <div class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="flex flex-wrap gap-2">
              <input v-model="q" class="input min-w-[14rem] flex-1" :placeholder="t('poems.search')" data-testid="poems-q" @keydown.enter="search()" />
              <button class="btn btn-primary" data-testid="poems-search" @click="search()">{{ t('poems.searchBtn') }}</button>
              <button class="btn btn-secondary" @click="search(true)">🎲 {{ t('poems.random') }}</button>
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-3 text-sm">
              <select v-model="dynasty" class="input w-32 py-1.5" @change="search()">
                <option value="">{{ t('poems.allDynasties') }}</option>
                <option v-for="d in dynasties" :key="d" :value="d">{{ d }}</option>
              </select>
              <select v-model="collection" class="input w-44 py-1.5" @change="search()">
                <option value="">{{ t('poems.allCollections') }}</option>
                <option v-for="c in config?.collections ?? []" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
              <label class="flex items-center gap-1 text-gray-600 dark:text-dark-300"><input v-model="famous" type="checkbox" @change="search()" /> {{ t('poems.famous') }}</label>
              <button v-if="author" class="rounded-full bg-primary-50 px-2.5 py-0.5 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300" @click="setAuthor('')">
                {{ t('poems.author', { name: author }) }} ✕
              </button>
            </div>
          </div>

          <p v-if="loading && !poems.length" class="py-10 text-center text-sm text-gray-400">{{ t('poems.loading') }}</p>
          <p v-else-if="searched && !poems.length" class="py-10 text-center text-sm text-gray-400">{{ t('poems.empty') }}</p>
          <div class="mt-4 grid gap-4 md:grid-cols-2">
            <PoemCard v-for="p in poems" :key="p.id" :poem="p" :words="words" @author="setAuthor" />
          </div>
          <div v-if="poems.length && !random" class="mt-6 text-center">
            <button v-if="hasMore" class="btn btn-secondary" :disabled="loading" @click="more">{{ t('poems.loadMore') }}</button>
            <span v-else class="text-xs text-gray-400">{{ t('poems.noMore') }}</span>
          </div>
        </section>

        <!-- 按画面找 -->
        <section v-show="active === 'find'" class="mt-6">
          <div class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <textarea v-model="scene" rows="3" maxlength="200" class="input" :placeholder="t('poems.find.scene')" data-testid="poems-scene" />
            <div class="mt-3 flex flex-wrap items-center gap-3 text-sm">
              <template v-if="signedIn">
                <button class="btn btn-primary" :disabled="finding || !scene.trim() || !config?.find_enabled" data-testid="poems-find" @click="find">
                  {{ finding ? t('poems.find.going') : t('poems.find.go') }}
                </button>
                <span class="text-gray-500 dark:text-dark-400" data-testid="poems-quota">{{ quotaText }}</span>
                <span v-if="balance !== undefined" class="text-gray-400">{{ t('poems.find.balance', { b: balance.toFixed(2) }) }}</span>
              </template>
              <template v-else>
                <RouterLink :to="{ path: '/login', query: { redirect: '/poems?tab=find' } }" class="btn btn-primary" data-testid="poems-login">{{ t('poems.find.login') }}</RouterLink>
                <span class="text-gray-500 dark:text-dark-400">{{ t('poems.find.quotaPaidFree', { n: config?.free_per_day ?? 0, price: (config?.find_price ?? 0).toFixed(2) }) }}</span>
              </template>
            </div>
            <p class="mt-2 text-xs text-gray-400">{{ t('poems.find.hint') }}</p>
          </div>

          <template v-if="found">
            <div class="mt-4 flex flex-wrap items-center gap-2 text-sm">
              <span class="text-gray-500">{{ t('poems.find.keywords') }}</span>
              <span v-for="w in found.keywords" :key="w" class="rounded-full bg-amber-50 px-2.5 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">{{ w }}</span>
              <span class="ml-auto text-xs text-gray-400">{{ found.free ? t('poems.find.freeUsed') : t('poems.find.charged', { price: found.charged.toFixed(2) }) }}</span>
            </div>
            <h2 v-if="found.picks.length" class="mt-5 text-base font-semibold text-gray-900 dark:text-white">{{ t('poems.find.picks') }}</h2>
            <div class="mt-3 grid gap-4 md:grid-cols-2" data-testid="poems-picks">
              <PoemCard v-for="p in found.picks" :key="p.poem.id" :poem="p.poem" :reason="p.reason" :words="found.keywords" @author="searchAuthor" />
            </div>
            <h2 v-if="otherCandidates.length" class="mt-8 text-base font-semibold text-gray-900 dark:text-white">{{ t('poems.find.candidates') }}</h2>
            <div class="mt-3 grid gap-4 md:grid-cols-2">
              <PoemCard v-for="p in otherCandidates" :key="p.id" :poem="p" :words="found.keywords" @author="searchAuthor" />
            </div>
          </template>
        </section>

        <p class="mt-10 text-center text-xs text-gray-400">{{ t('poems.versionNote') }}</p>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
// 诗词灵感 (/poems): search the public-domain poem corpus, or describe a scene and let AI find poems
// for it (signed in; a few free a day, then billed). Every poem links to the canvas to draw it.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import { track } from '@/utils/analytics'
import * as api from '@/api/poems'
import { queryWords } from '@/views/admin/inspiration/highlight'
import PoemCard from './PoemCard.vue'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const tabs = ['search', 'find'] as const
const dynasties = ['先秦', '汉', '魏晋', '南北朝', '隋', '唐', '五代', '宋', '辽', '金', '元', '明', '清']

const active = ref<(typeof tabs)[number]>(route.query.tab === 'find' ? 'find' : 'search')
const config = ref<api.PoemsConfig | null>(null)
const signedIn = computed(() => authStore.isAuthenticated)

const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const dynasty = ref('')
const collection = ref('')
const author = ref('')
const famous = ref(false)
const random = ref(false)
const poems = ref<api.Poem[]>([])
const hasMore = ref(false)
const loading = ref(false)
const searched = ref(false)
const words = ref<string[]>([])

const scene = ref('')
const finding = ref(false)
const found = ref<api.PoemFindResult | null>(null)
const freeLeft = ref(0)
const balance = ref<number | undefined>(undefined)

const otherCandidates = computed(() => {
  const picked = new Set(found.value?.picks.map((p) => p.poem.id))
  return (found.value?.candidates ?? []).filter((p) => !picked.has(p.id)).slice(0, 12)
})
const quotaText = computed(() => {
  const price = (config.value?.find_price ?? 0).toFixed(2)
  return freeLeft.value > 0 ? t('poems.find.quotaFree', { n: freeLeft.value }) : t('poems.find.quotaPaid', { price })
})

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function query(offset: number): api.PoemQuery {
  return { q: q.value.trim(), dynasty: dynasty.value, collection: collection.value, author: author.value, famous: famous.value, random: random.value, offset }
}

async function search(rand = false) {
  random.value = rand
  words.value = rand ? [] : queryWords(q.value)
  loading.value = true
  try {
    const res = await api.searchPoems(query(0))
    poems.value = res.items
    hasMore.value = res.has_more
    searched.value = true
    track('poems_search', { random: rand, has_q: !!q.value.trim() })
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
}

async function more() {
  loading.value = true
  try {
    const res = await api.searchPoems(query(poems.value.length))
    poems.value = poems.value.concat(res.items)
    hasMore.value = res.has_more
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
}

function setAuthor(name: string) {
  author.value = name
  void search()
}

function searchAuthor(name: string) {
  active.value = 'search'
  q.value = ''
  setAuthor(name)
}

async function find() {
  finding.value = true
  try {
    const res = await api.findPoems({ scene: scene.value.trim() })
    found.value = res
    freeLeft.value = res.free_left
    if (res.balance !== undefined) balance.value = res.balance
    track('poems_find', { free: res.free, picks: res.picks.length })
  } catch (e) {
    fail(e)
  } finally {
    finding.value = false
  }
}

onMounted(async () => {
  try {
    config.value = await api.poemsConfig()
    freeLeft.value = config.value.free_left
    balance.value = config.value.balance
  } catch (e) {
    fail(e)
  }
  if (!config.value?.enabled) return
  if (q.value) void search()
  else {
    famous.value = true
    void search(true)
  }
})
</script>
