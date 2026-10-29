<template>
  <article class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900" :data-testid="`poem-${poem.id}`">
    <div class="flex items-start justify-between gap-2">
      <div class="min-w-0">
        <RouterLink :to="`/poems/${poem.id}`" class="font-serif text-lg font-semibold text-gray-950 hover:text-primary-700 dark:text-white">
          <template v-for="(p, i) in highlightParts(poem.title, words)" :key="i"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
        </RouterLink>
        <div class="mt-0.5 flex flex-wrap items-center gap-x-2 text-xs text-gray-500 dark:text-dark-400">
          <button class="hover:text-primary-600 hover:underline" @click="emit('author', poem.author)">{{ poem.dynasty ? `〔${poem.dynasty}〕` : '' }}{{ poem.author }}</button>
          <span v-if="poem.famous" class="rounded bg-rose-50 px-1.5 text-rose-600 dark:bg-rose-900/30 dark:text-rose-300">{{ t('poems.famousTag') }}</span>
        </div>
      </div>
    </div>
    <p v-if="reason" class="mt-3 rounded-xl bg-primary-50 px-3 py-2 text-sm text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">{{ reason }}</p>
    <div class="mt-3 space-y-0.5 font-serif text-[15px] leading-7 text-gray-800 dark:text-dark-200">
      <p v-for="(line, li) in shownLines" :key="li">
        <template v-for="(p, i) in highlightParts(line, words)" :key="i"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
      </p>
    </div>
    <div class="mt-4 flex flex-wrap items-center gap-4 text-sm">
      <a :href="poemDrawLink(poem)" target="_blank" rel="noopener" class="font-semibold text-primary-600 hover:underline dark:text-primary-400" :data-testid="`poem-draw-${poem.id}`" @click="track('poem_draw', { id: poem.id })">🎨 {{ t('poems.draw') }}</a>
      <RouterLink v-if="cut" :to="`/poems/${poem.id}`" class="text-gray-500 hover:underline">{{ t('poems.full') }}</RouterLink>
      <button class="text-gray-500 hover:underline" @click="copy">{{ t('poems.copy') }}</button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { track } from '@/utils/analytics'
import type { Poem } from '@/api/poems'
import { highlightParts } from '@/views/admin/inspiration/highlight'
import { poemDrawLink } from './drawPrompt'

const props = withDefaults(defineProps<{ poem: Poem; words?: string[]; reason?: string }>(), { words: () => [], reason: '' })
const emit = defineEmits<{ author: [name: string] }>()
const { t } = useI18n()
const appStore = useAppStore()

const maxLines = 8
const maxChars = 160
const lines = computed(() => props.poem.content.split('\n'))
const cut = computed(() => lines.value.length > maxLines || props.poem.content.length > maxChars)
/** Up to eight lines and about 160 characters; the poem page has the rest. */
const shownLines = computed(() => {
  if (!cut.value) return lines.value
  const out: string[] = []
  let budget = maxChars
  for (const line of lines.value.slice(0, maxLines)) {
    if (budget <= 0) break
    out.push(line.length > budget ? line.slice(0, budget) + '……' : line)
    budget -= line.length
  }
  return out
})

async function copy() {
  const p = props.poem
  try {
    await navigator.clipboard.writeText(`${p.title}\n${p.dynasty ? `〔${p.dynasty}〕` : ''}${p.author}\n${p.content}`)
    appStore.showSuccess(t('poems.copied'))
  } catch {
    /* clipboard denied */
  }
}
</script>
