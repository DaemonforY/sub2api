<template>
  <div class="space-y-4" data-testid="desk-drafts">
    <div class="card space-y-2 p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.drafts.title') }}</h3>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.drafts.hint') }}</p>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="busy" @click="rewrite">{{ desk.drafts.length ? t('admin.desk.drafts.again') : t('admin.desk.drafts.make') }}</button>
      </div>
      <div v-if="desk.judge?.reasons?.length" class="rounded-lg bg-gray-50 p-3 text-sm text-gray-700 dark:bg-dark-800 dark:text-dark-200">
        <span class="font-medium">{{ t('admin.desk.drafts.judge') }}</span>{{ desk.judge.reasons[0] }}
      </div>
      <div class="flex flex-wrap gap-3 text-xs text-gray-500">
        <span class="flex items-center gap-1"><span class="inline-block h-3 w-1 rounded bg-amber-400" />{{ t('admin.desk.drafts.skim') }}</span>
        <span class="flex items-center gap-1"><span class="inline-block h-3 w-1 rounded bg-red-500" />{{ t('admin.desk.drafts.leave') }}</span>
      </div>
    </div>

    <p v-if="!desk.drafts.length && !busy" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.desk.drafts.empty') }}</p>

    <!-- Tabs on small screens, columns on large -->
    <div v-if="desk.drafts.length" class="flex gap-2 lg:hidden">
      <button v-for="d in desk.drafts" :key="d.id" :class="['btn btn-sm', shown === d.id ? 'btn-primary' : 'btn-secondary']" @click="shown = d.id">
        {{ d.label }}<span v-if="rank(d.id) === 1"> 🏆</span>
      </button>
    </div>
    <div class="grid gap-3 lg:grid-cols-3">
      <div
        v-for="d in desk.drafts"
        :key="d.id"
        :class="['card flex-col p-4', shown === d.id ? 'flex' : 'hidden lg:flex', base === d.id ? 'ring-2 ring-primary-500' : '']"
        :data-testid="`desk-draft-${d.structure}`"
      >
        <div class="mb-2 flex items-center justify-between gap-2">
          <div class="font-semibold text-gray-900 dark:text-white">
            {{ t('admin.desk.drafts.label', { label: d.label }) }}
            <span v-if="rank(d.id)" :class="['ml-1 rounded px-1.5 py-0.5 text-xs', rank(d.id) === 1 ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700']">
              {{ t('admin.desk.drafts.rank', { n: rank(d.id) }) }}
            </span>
          </div>
          <span class="text-xs text-gray-400">{{ plainLen(d.markdown) }} {{ t('admin.desk.common.chars') }}</span>
        </div>
        <p v-if="d.error" class="text-sm text-red-600">{{ d.error }}</p>
        <template v-else>
          <div v-if="d.review" class="mb-2 grid grid-cols-4 gap-1 text-center text-xs">
            <div v-for="k in (['hook', 'flow', 'voice', 'substance'] as const)" :key="k" class="rounded bg-gray-50 py-1 dark:bg-dark-800">
              <div class="text-gray-500">{{ t(`admin.desk.drafts.score.${k}`) }}</div>
              <div class="font-semibold text-gray-900 dark:text-white">{{ d.review[k] }}</div>
            </div>
          </div>
          <div class="max-h-[60vh] flex-1 space-y-2 overflow-y-auto pr-1 text-sm leading-relaxed">
            <div v-for="(p, i) in paras(d.markdown)" :key="i" :class="['border-l-2 pl-2', markClass(d, i + 1)]">
              <div class="prose prose-sm max-w-none dark:prose-invert" v-html="render(p)" />
              <div v-if="mark(d, i + 1)?.why" :class="['mt-0.5 text-xs', mark(d, i + 1)?.v === 'leave' ? 'text-red-600' : 'text-amber-600']">👀 {{ mark(d, i + 1)?.why }}</div>
            </div>
          </div>
          <div v-if="d.review" class="mt-2 space-y-1 border-t border-gray-100 pt-2 text-xs dark:border-dark-700">
            <p v-if="d.review.best_line" class="text-emerald-700 dark:text-emerald-300">✨ {{ d.review.best_line }}</p>
            <p v-for="(pr, i) in d.review.problems" :key="i" class="text-gray-500">· {{ pr }}</p>
          </div>
          <button class="btn btn-secondary btn-sm mt-3" @click="base = d.id">{{ base === d.id ? t('admin.desk.drafts.isBase') : t('admin.desk.drafts.useBase') }}</button>
        </template>
      </div>
    </div>

    <div v-if="base" class="card space-y-2 p-4" data-testid="desk-highlights">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.drafts.highlights') }}</h4>
      <p v-if="!highlights.length" class="text-xs text-gray-400">{{ t('admin.desk.drafts.noHighlights') }}</p>
      <label v-for="h in highlights" :key="h.id" class="flex items-start gap-2 text-sm">
        <input v-model="picked" type="checkbox" :value="h.id" class="mt-1 h-4 w-4" />
        <span>
          <span class="text-gray-800 dark:text-dark-100">「{{ h.quote }}」</span>
          <span class="block text-xs text-gray-500">{{ draftLabel(h.draft_id) }} · {{ h.why }}</span>
        </span>
      </label>
    </div>

    <div class="card flex flex-wrap items-center justify-end gap-2 p-4">
      <button class="btn btn-secondary" @click="emit('stage', 'angles')">{{ t('admin.desk.common.back') }}</button>
      <button class="btn btn-primary" :disabled="busy || !base" data-testid="desk-use-base" @click="useBase">
        {{ picked.length ? t('admin.desk.drafts.merge', { n: picked.length }) : t('admin.desk.drafts.edit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, DeskStage, Draft } from '@/api/admin/content'

const props = defineProps<{ item: ContentItem; busy: boolean }>()
const emit = defineEmits<{ (e: 'updated', it: ContentItem): void; (e: 'stage', s: DeskStage): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const desk = computed(() => props.item.desk!)
const base = ref('')
const picked = ref<string[]>([])
const shown = ref('')

watch(
  () => desk.value.drafts.map((d) => d.id).join(','),
  () => {
    base.value = desk.value.base_id || desk.value.judge?.ranking?.[0] || ''
    picked.value = []
    shown.value = base.value || desk.value.drafts[0]?.id || ''
  },
  { immediate: true }
)

const highlights = computed(() => (desk.value.judge?.highlights || []).filter((h) => h.draft_id !== base.value))

function rank(id: string) {
  const i = desk.value.judge?.ranking?.indexOf(id) ?? -1
  return i >= 0 ? i + 1 : 0
}

function draftLabel(id: string) {
  const d = desk.value.drafts.find((x) => x.id === id)
  return d ? t('admin.desk.drafts.label', { label: d.label }) : ''
}

// Paragraphs as the reviewer saw them (blank-line separated, code blocks whole).
function paras(md: string) {
  const out: string[] = []
  let cur: string[] = []
  let inCode = false
  for (const line of md.split('\n')) {
    if (line.trim().startsWith('```')) inCode = !inCode
    if (!inCode && !line.trim()) {
      if (cur.join('').trim()) out.push(cur.join('\n'))
      cur = []
      continue
    }
    cur.push(line)
  }
  if (cur.join('').trim()) out.push(cur.join('\n'))
  return out
}

function mark(d: Draft, i: number) {
  return d.review?.paras.find((m) => m.i === i)
}

function markClass(d: Draft, i: number) {
  const v = mark(d, i)?.v
  if (v === 'leave') return 'border-red-500 bg-red-50/50 dark:bg-red-900/10'
  if (v === 'skim') return 'border-amber-400'
  return 'border-transparent'
}

const render = (md: string) => renderMarkdown(md)
const plainLen = (md: string) => md.replace(/\s|[#*`>]/g, '').length

async function call(fn: () => Promise<ContentItem>) {
  try {
    emit('updated', await fn())
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

const rewrite = () => call(() => contentAPI.makeDrafts(props.item.id))
const useBase = () => call(() => contentAPI.chooseBase(props.item.id, base.value, picked.value))
</script>
