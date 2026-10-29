<template>
  <div class="min-h-screen bg-gradient-to-b from-amber-50/60 via-white to-white dark:from-dark-950 dark:via-dark-950 dark:to-dark-950">
    <PlazaNavBar :login-redirect="route.fullPath" />
    <main class="mx-auto max-w-2xl px-4 pb-20 pt-8 sm:px-6">
      <RouterLink to="/poems" class="text-sm text-gray-500 hover:text-primary-600">← {{ t('poems.detail.back') }}</RouterLink>
      <p v-if="loading" class="py-16 text-center text-sm text-gray-400">{{ t('poems.loading') }}</p>
      <p v-else-if="!poem" class="py-16 text-center text-sm text-gray-400">{{ error || t('poems.detail.notFound') }}</p>
      <article v-else class="mt-6 rounded-3xl border border-gray-200 bg-white p-8 text-center shadow-sm dark:border-dark-700 dark:bg-dark-900" data-testid="poem-detail">
        <h1 class="font-serif text-2xl font-bold text-gray-950 dark:text-white">{{ poem.title }}</h1>
        <div class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          <RouterLink :to="{ path: '/poems', query: { q: poem.author } }" class="hover:text-primary-600 hover:underline">{{ poem.dynasty ? `〔${poem.dynasty}〕` : '' }}{{ poem.author }}</RouterLink>
          <span v-if="poem.famous" class="ml-2 rounded bg-rose-50 px-1.5 text-xs text-rose-600 dark:bg-rose-900/30 dark:text-rose-300">{{ t('poems.famousTag') }}</span>
        </div>
        <div class="mt-6 space-y-1 font-serif text-lg leading-9 text-gray-800 dark:text-dark-100" :class="prose ? 'text-left' : ''">
          <p v-for="(line, i) in poem.content.split('\n')" :key="i" :class="prose ? 'indent-8' : ''">{{ line }}</p>
        </div>
        <div class="mt-8 flex flex-wrap justify-center gap-3">
          <a :href="poemDrawLink(poem)" target="_blank" rel="noopener" class="btn btn-primary" data-testid="poem-detail-draw" @click="track('poem_draw', { id: poem.id })">🎨 {{ t('poems.draw') }}</a>
          <button class="btn btn-secondary" @click="copy">{{ t('poems.copy') }}</button>
        </div>
        <details v-if="poem.notes" class="mt-6 text-left text-sm text-gray-600 dark:text-dark-300">
          <summary class="cursor-pointer text-gray-500">{{ t('poems.detail.notes') }}</summary>
          <p class="mt-2 whitespace-pre-line">{{ poem.notes }}</p>
        </details>
      </article>
      <p class="mt-8 text-center text-xs text-gray-400">{{ t('poems.versionNote') }}</p>
    </main>
  </div>
</template>

<script setup lang="ts">
// One poem (/poems/:id): the whole text, notes, and 用这首诗生图.
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { track } from '@/utils/analytics'
import * as api from '@/api/poems'
import { poemDrawLink } from './drawPrompt'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const poem = ref<api.Poem | null>(null)
const loading = ref(true)
const error = ref('')
/** Prose (古文观止, 论语) reads better left-aligned with indented paragraphs. */
const prose = computed(() => !!poem.value && (poem.value.collection === 'guwen' || poem.value.collection === 'lunyu'))

async function load(id: number) {
  loading.value = true
  error.value = ''
  try {
    poem.value = await api.getPoem(id)
    document.title = `${poem.value.title} · ${poem.value.author} - ${t('poems.title')}`
  } catch (e) {
    poem.value = null
    error.value = extractApiErrorMessage(e, t('poems.detail.notFound'))
  } finally {
    loading.value = false
  }
}

async function copy() {
  const p = poem.value
  if (!p) return
  try {
    await navigator.clipboard.writeText(`${p.title}\n${p.dynasty ? `〔${p.dynasty}〕` : ''}${p.author}\n${p.content}`)
    appStore.showSuccess(t('poems.copied'))
  } catch {
    /* clipboard denied */
  }
}

watch(
  () => route.params.id,
  (id) => void load(Number(id)),
  { immediate: true }
)
</script>
