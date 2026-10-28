<template>
  <div class="card p-4" :data-testid="`poem-${poem.id}`">
    <div class="flex items-start justify-between gap-2">
      <div class="min-w-0">
        <div class="font-serif text-base font-semibold text-gray-900 dark:text-white">
          <template v-for="(p, i) in highlightParts(poem.title, words)" :key="i"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
        </div>
        <div class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500">
          <button class="hover:text-primary-600 hover:underline" :title="t('inspiration.poems.byAuthor')" @click="emit('author', poem.author)">
            {{ poem.dynasty ? `〔${poem.dynasty}〕` : '' }}{{ poem.author }}
          </button>
          <span v-if="collectionName">· {{ collectionName }}</span>
          <span v-if="poem.famous" class="rounded bg-rose-50 px-1.5 text-rose-600 dark:bg-rose-900/30 dark:text-rose-300">{{ t('inspiration.poems.famousTag') }}</span>
          <span v-for="tag in shownTags" :key="tag" class="rounded bg-gray-100 px-1.5 dark:bg-dark-700">{{ tag }}</span>
          <span v-if="poem.score" class="text-primary-600">{{ t('inspiration.find.hits', { n: poem.score }) }}</span>
        </div>
      </div>
      <button
        class="shrink-0 text-lg leading-none"
        :class="poem.starred ? 'text-amber-500' : 'text-gray-300 hover:text-amber-400'"
        :title="poem.starred ? t('inspiration.unstar') : t('inspiration.star')"
        :data-testid="`poem-star-${poem.id}`"
        @click="toggleStar"
      >
        ★
      </button>
    </div>
    <p v-if="reason" class="mt-2 rounded-lg bg-primary-50 px-2.5 py-1.5 text-xs text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">{{ reason }}</p>
    <div class="mt-3 space-y-0.5 font-serif text-[15px] leading-7 text-gray-800 dark:text-gray-200">
      <p v-for="(line, li) in shownLines" :key="li">
        <template v-for="(p, i) in highlightParts(line, words)" :key="i"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
      </p>
    </div>
    <details v-if="expanded && poem.notes" class="mt-2 text-xs text-gray-500">
      <summary class="cursor-pointer">{{ t('inspiration.poems.notes') }}</summary>
      <p class="mt-1 whitespace-pre-line">{{ poem.notes }}</p>
    </details>
    <div class="mt-3 flex flex-wrap items-center gap-3 text-xs">
      <button v-if="long" class="text-gray-500 hover:underline" @click="expanded = !expanded">
        {{ expanded ? t('inspiration.collapse') : t('inspiration.expand') + `（${lines.length}）` }}
      </button>
      <button class="text-gray-500 hover:underline" @click="copy">{{ t('inspiration.copy') }}</button>
      <span v-if="inShihua" class="text-emerald-600">✓ {{ t('inspiration.poems.inShihua') }}</span>
      <button v-else class="font-medium text-primary-600 hover:underline" :disabled="adding" :data-testid="`poem-to-shihua-${poem.id}`" @click="toShihua">
        {{ t('inspiration.poems.toShihua') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'
import { highlightParts } from './highlight'

const props = withDefaults(defineProps<{ poem: api.LibPoem; words?: string[]; reason?: string; names?: Record<string, string> }>(), {
  words: () => [],
  reason: '',
  names: () => ({})
})
const emit = defineEmits<{ author: [name: string]; starred: [id: number, starred: boolean] }>()
const { t } = useI18n()
const appStore = useAppStore()

const previewLines = 6
const previewChars = 140
const expanded = ref(false)
const adding = ref(false)
const inShihua = ref(!!props.poem.in_shihua)

const lines = computed(() => props.poem.content.split('\n'))
const long = computed(() => lines.value.length > previewLines || props.poem.content.length > previewChars)
/**
 * Collapsed: up to six lines and about 140 characters (prose has long paragraphs), starting near the
 * first matched word.
 */
const shownLines = computed(() => {
  if (expanded.value || !long.value) return lines.value
  const hit = props.words.length ? lines.value.findIndex((l) => props.words.some((w) => l.includes(w))) : -1
  const start = hit > 0 && (hit > previewLines - 2 || lines.value.slice(0, hit).join('').length > previewChars / 2) ? hit : 0
  const out: string[] = start > 0 ? ['……'] : []
  let budget = previewChars
  for (const line of lines.value.slice(start, start + previewLines)) {
    if (budget <= 0) break
    out.push(line.length > budget ? line.slice(0, budget) + '……' : line)
    budget -= line.length
  }
  return out
})
const collectionName = computed(() => props.names[props.poem.collection] ?? '')
const shownTags = computed(() => props.poem.tags.filter((tag) => tag && tag !== collectionName.value).slice(0, 3))

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function toggleStar() {
  const next = !props.poem.starred
  try {
    await api.starPoem(props.poem.id, next)
    emit('starred', props.poem.id, next)
  } catch (e) {
    fail(e)
  }
}

async function copy() {
  const p = props.poem
  try {
    await navigator.clipboard.writeText(`${p.title}\n${p.dynasty ? `〔${p.dynasty}〕` : ''}${p.author}\n${p.content}`)
    appStore.showSuccess(t('inspiration.copied'))
  } catch (e) {
    fail(e)
  }
}

async function toShihua() {
  adding.value = true
  try {
    const res = await api.poemToShihua(props.poem.id)
    inShihua.value = true
    appStore.showSuccess(t(res.created ? 'inspiration.poems.addedShihua' : 'inspiration.poems.existsShihua'))
  } catch (e) {
    fail(e)
  } finally {
    adding.value = false
  }
}
</script>
