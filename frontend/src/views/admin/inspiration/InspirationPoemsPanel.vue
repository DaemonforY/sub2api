<template>
  <div class="space-y-4">
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap gap-2">
        <input
          v-model="q"
          class="input min-w-[16rem] flex-1"
          :placeholder="t('inspiration.poems.search')"
          data-testid="poems-q"
          @keydown.enter="search()"
        />
        <button class="btn btn-primary" data-testid="poems-search" @click="search()">{{ t('inspiration.poems.searchBtn') }}</button>
        <button class="btn btn-secondary" data-testid="poems-random" @click="search(true)">🎲 {{ t('inspiration.poems.random') }}</button>
      </div>
      <div class="flex flex-wrap items-center gap-3 text-sm">
        <select v-model="dynasty" class="input w-32 py-1.5" data-testid="poems-dynasty" @change="search()">
          <option value="">{{ t('inspiration.allDynasties') }}</option>
          <option v-for="d in dynasties" :key="d" :value="d">{{ d }}</option>
        </select>
        <select v-model="collection" class="input w-44 py-1.5" data-testid="poems-collection" @change="search()">
          <option value="">{{ t('inspiration.allCollections') }}</option>
          <option v-for="c in collections" :key="c.id" :value="c.id">{{ c.name }}（{{ c.count.toLocaleString() }}）</option>
        </select>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300"><input v-model="famous" type="checkbox" @change="search()" /> {{ t('inspiration.poems.famous') }}</label>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300"><input v-model="starred" type="checkbox" @change="search()" /> {{ t('inspiration.poems.starred') }}</label>
        <button v-if="author" class="rounded-full bg-primary-50 px-2.5 py-0.5 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300" @click="setAuthor('')">
          {{ t('inspiration.poems.author', { name: author }) }} ✕
        </button>
        <span class="ml-auto text-xs text-gray-400">{{ total ? t('inspiration.poems.total', { n: total.toLocaleString() }) : '' }}</span>
      </div>
      <p class="text-xs text-gray-400">{{ t('inspiration.poems.corpusNote') }}</p>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1fr_20rem]">
      <div class="space-y-3">
        <p v-if="loading && !poems.length" class="py-10 text-center text-sm text-gray-400">{{ t('inspiration.loading') }}</p>
        <p v-else-if="searched && !poems.length" class="card py-10 text-center text-sm text-gray-400">{{ t('inspiration.empty') }}</p>
        <div class="grid gap-3 lg:grid-cols-2">
          <InspirationPoemCard
            v-for="p in poems"
            :key="p.id"
            :poem="p"
            :words="words"
            :names="names"
            @author="setAuthor"
            @starred="onStarred"
          />
        </div>
        <div v-if="poems.length" class="text-center">
          <button v-if="hasMore" class="btn btn-secondary btn-sm" :disabled="loading" data-testid="poems-more" @click="more">{{ t('inspiration.loadMore') }}</button>
          <span v-else class="text-xs text-gray-400">{{ t('inspiration.noMore') }}</span>
        </div>
      </div>

      <!-- The same words in pasted lyrics: internal reference only. -->
      <aside v-if="words.length" class="card h-fit p-4" data-testid="poems-song-hits">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.poems.songHits') }}</h3>
        <p v-if="!songHits.length" class="mt-2 text-xs text-gray-400">{{ t('inspiration.poems.noLyricHits') }}</p>
        <div v-for="s in songHits" :key="s.id" class="mt-3 border-t border-gray-100 pt-3 text-sm first:border-0 first:pt-0 dark:border-dark-700">
          <div class="font-medium text-gray-800 dark:text-gray-100">{{ s.title }} <span class="text-xs font-normal text-gray-500">{{ s.artist }}</span></div>
          <p v-for="(m, i) in s.matches" :key="i" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
            <template v-for="(p, j) in highlightParts(m, words)" :key="j"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
          </p>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'
import InspirationPoemCard from './InspirationPoemCard.vue'
import { highlightParts, queryWords } from './highlight'

const { t } = useI18n()
const appStore = useAppStore()
const dynasties = ['先秦', '汉', '魏晋', '南北朝', '隋', '唐', '五代', '宋', '辽', '金', '元', '明', '清']
const pageSize = 30

const q = ref('')
const dynasty = ref('')
const collection = ref('')
const author = ref('')
const famous = ref(false)
const starred = ref(false)
const random = ref(false)
const poems = ref<api.LibPoem[]>([])
const hasMore = ref(false)
const loading = ref(false)
const searched = ref(false)
const words = ref<string[]>([])
const songHits = ref<api.LibSong[]>([])
const counts = ref<Record<string, number>>({})
const names = ref<Record<string, string>>({})

const collections = computed(() =>
  Object.entries(counts.value)
    .map(([id, count]) => ({ id, count, name: names.value[id] ?? id }))
    .sort((a, b) => b.count - a.count)
)
const total = computed(() => Object.values(counts.value).reduce((a, b) => a + b, 0))

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function query(offset: number): api.LibPoemQuery {
  return {
    q: q.value.trim(), dynasty: dynasty.value, collection: collection.value, author: author.value,
    famous: famous.value, starred: starred.value, random: random.value, limit: pageSize, offset
  }
}

async function search(rand = false) {
  random.value = rand
  loading.value = true
  words.value = rand ? [] : queryWords(q.value)
  try {
    const res = await api.searchPoems(query(0))
    poems.value = res.items
    hasMore.value = res.has_more && !rand
    searched.value = true
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
  songHits.value = []
  if (words.value.length) {
    try {
      songHits.value = (await api.listSongs({ q: words.value.join(' '), limit: 8 })).items.filter((s) => s.matches?.length)
    } catch {
      /* the side list is optional */
    }
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

function onStarred(id: number, value: boolean) {
  poems.value = poems.value.map((p) => (p.id === id ? { ...p, starred: value } : p))
}

/** Used by the page: search for a poem named in a song card. */
function searchFor(text: string) {
  q.value = text
  dynasty.value = collection.value = author.value = ''
  famous.value = starred.value = false
  void search()
}
defineExpose({ searchFor })

onMounted(async () => {
  try {
    const res = await api.poemCounts()
    counts.value = res.counts
    names.value = res.names
  } catch (e) {
    fail(e)
  }
  // Start with famous works so the page isn't empty.
  famous.value = true
  void search(true)
})
</script>
