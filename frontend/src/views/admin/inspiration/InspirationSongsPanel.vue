<template>
  <div class="space-y-4">
    <p class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300" data-testid="songs-notice">
      🔒 {{ t('inspiration.songs.notice') }}
    </p>
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap gap-2">
        <input v-model="q" class="input min-w-[14rem] flex-1" :placeholder="t('inspiration.songs.search')" data-testid="songs-q" @keydown.enter="load()" />
        <button class="btn btn-primary" @click="load()">{{ t('inspiration.poems.searchBtn') }}</button>
        <button class="btn btn-secondary" data-testid="songs-add" @click="openEditor(null)">＋ {{ t('inspiration.songs.add') }}</button>
      </div>
      <div class="flex flex-wrap items-center gap-3 text-sm">
        <select v-model="artist" class="input w-40 py-1.5" @change="load()">
          <option value="">{{ t('inspiration.songs.allArtists') }}</option>
          <option v-for="a in facets.artists" :key="a.name" :value="a.name">{{ a.name }}（{{ a.count }}）</option>
        </select>
        <select v-model="tag" class="input w-40 py-1.5" @change="load()">
          <option value="">{{ t('inspiration.songs.allTags') }}</option>
          <option v-for="f in facets.tags" :key="f.name" :value="f.name">{{ f.name }}（{{ f.count }}）</option>
        </select>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300">
          {{ t('inspiration.songs.yearFrom') }}
          <input v-model.number="yearFrom" type="number" class="input w-20 py-1" :placeholder="String(facets.years[0] ?? '')" @change="load()" />
          {{ t('inspiration.songs.yearTo') }}
          <input v-model.number="yearTo" type="number" class="input w-20 py-1" :placeholder="String(facets.years[1] ?? '')" @change="load()" />
        </label>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300"><input v-model="starred" type="checkbox" @change="load()" /> {{ t('inspiration.songs.starred') }}</label>
        <span class="ml-auto text-xs text-gray-400">{{ t('inspiration.songs.total', { n: total }) }}</span>
      </div>
    </div>

    <p v-if="loaded && !songs.length" class="card py-10 text-center text-sm text-gray-400">{{ t('inspiration.songs.empty') }}</p>
    <div class="grid gap-3 lg:grid-cols-2 2xl:grid-cols-3">
      <div v-for="s in songs" :key="s.id" class="card p-4" :data-testid="`song-${s.id}`">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="font-semibold text-gray-900 dark:text-white">{{ s.title }}</div>
            <div class="text-xs text-gray-500">
              {{ [s.artist, s.year, s.lyricist && `词：${s.lyricist}`].filter(Boolean).join(' · ') }}
              <span v-if="!s.has_lyrics" class="ml-1 text-amber-600">{{ t('inspiration.songs.noLyrics') }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <button class="text-lg leading-none" :class="s.starred ? 'text-amber-500' : 'text-gray-300 hover:text-amber-400'" @click="toggleStar(s)">★</button>
            <button class="text-xs text-primary-600 hover:underline" :data-testid="`song-edit-${s.id}`" @click="openEditor(s.id)">{{ t('inspiration.songs.edit') }}</button>
          </div>
        </div>
        <div class="mt-2 flex flex-wrap gap-1 text-xs">
          <button
            v-for="x in [...s.imagery, ...s.moods, ...s.tags]"
            :key="x"
            class="rounded bg-gray-100 px-1.5 py-0.5 text-gray-600 hover:bg-primary-50 hover:text-primary-700 dark:bg-dark-700 dark:text-gray-300"
            @click="filterTag(x)"
          >
            {{ x }}
          </button>
        </div>
        <p v-if="s.summary" class="mt-2 text-sm text-gray-700 dark:text-gray-300">{{ s.summary }}</p>
        <div v-if="s.matches?.length" class="mt-2 rounded-lg bg-amber-50/60 px-2.5 py-1.5 text-xs text-gray-700 dark:bg-amber-900/10 dark:text-gray-300">
          <div class="mb-0.5 text-gray-400">{{ t('inspiration.songs.matches') }}</div>
          <p v-for="(m, i) in s.matches" :key="i">
            <template v-for="(p, j) in highlightParts(m, words)" :key="j"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
          </p>
        </div>
        <div v-if="s.allusions.length" class="mt-2 text-xs">
          <span class="text-gray-400">{{ t('inspiration.songs.allusions') }}：</span>
          <button v-for="a in s.allusions" :key="a.title + a.author" class="mr-2 text-primary-600 hover:underline" :title="a.note" @click="emit('search-poem', `${a.title} ${a.author}`.trim())">
            《{{ a.title }}》{{ a.author }}
          </button>
        </div>
      </div>
    </div>

    <InspirationSongEditor :song-id="editing" :show="editorOpen" @close="editorOpen = false" @saved="onSaved" @deleted="onDeleted" @search-poem="(q: string) => emit('search-poem', q)" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'
import InspirationSongEditor from './InspirationSongEditor.vue'
import { highlightParts, queryWords } from './highlight'

const emit = defineEmits<{ 'search-poem': [q: string] }>()
const { t } = useI18n()
const appStore = useAppStore()

const q = ref('')
const artist = ref('')
const tag = ref('')
const yearFrom = ref<number | ''>('')
const yearTo = ref<number | ''>('')
const starred = ref(false)
const songs = ref<api.LibSong[]>([])
const total = ref(0)
const loaded = ref(false)
const words = ref<string[]>([])
const facets = reactive<api.LibSongFacets>({ artists: [], tags: [], years: [] })
const editing = ref<number | null>(null)
const editorOpen = ref(false)

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function load() {
  words.value = queryWords(q.value)
  try {
    const res = await api.listSongs({
      q: q.value.trim(), artist: artist.value, tag: tag.value, starred: starred.value,
      year_from: Number(yearFrom.value) || 0, year_to: Number(yearTo.value) || 0, limit: 200
    })
    songs.value = res.items
    total.value = res.total
    loaded.value = true
  } catch (e) {
    fail(e)
  }
}

async function loadFacets() {
  try {
    Object.assign(facets, await api.songFacets())
  } catch (e) {
    fail(e)
  }
}

function filterTag(x: string) {
  tag.value = x
  void load()
}

async function toggleStar(s: api.LibSong) {
  try {
    await api.starSong(s.id, !s.starred)
    s.starred = !s.starred
  } catch (e) {
    fail(e)
  }
}

function openEditor(id: number | null) {
  editing.value = id
  editorOpen.value = true
}

function onSaved(song: api.LibSong) {
  editing.value = song.id
  void load()
  void loadFacets()
}

function onDeleted() {
  editorOpen.value = false
  void load()
  void loadFacets()
}

onMounted(() => {
  void load()
  void loadFacets()
})
</script>
