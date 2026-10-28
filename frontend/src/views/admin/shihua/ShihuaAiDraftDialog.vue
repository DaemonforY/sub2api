<template>
  <BaseDialog :show="show" :title="t('shihua.ai.title')" width="extra-wide" @close="close">
    <!-- step 1: what to look for -->
    <div class="space-y-3" data-testid="shihua-ai-form">
      <div class="flex gap-1">
        <button
          v-for="m in modes"
          :key="m"
          class="rounded-lg px-3 py-1 text-sm"
          :class="mode === m ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
          @click="mode = m"
        >
          {{ t(`shihua.ai.modes.${m}`) }}
        </button>
      </div>
      <textarea
        v-if="mode === 'topic'"
        v-model="topic"
        rows="2"
        class="input text-sm"
        :placeholder="t('shihua.ai.topicPlaceholder')"
        data-testid="shihua-ai-topic"
      />
      <textarea v-else v-model="titles" rows="5" class="input text-sm" :placeholder="t('shihua.ai.titlesPlaceholder')" data-testid="shihua-ai-titles" />
      <div class="flex flex-wrap items-end gap-3">
        <label class="text-xs text-gray-500">
          {{ t('shihua.ai.stage') }}
          <select v-model="stage" class="input mt-1 w-28 text-sm">
            <option value="">{{ t('shihua.ai.anyStage') }}</option>
            <option v-for="s in stages" :key="s" :value="s">{{ t(`shihua.stages.${s}`) }}</option>
          </select>
        </label>
        <label v-if="mode === 'topic'" class="text-xs text-gray-500">
          {{ t('shihua.ai.count') }}
          <input v-model.number="count" type="number" min="1" max="30" class="input mt-1 w-20 text-sm" />
        </label>
        <button class="btn btn-primary" :disabled="generating || !canGenerate" data-testid="shihua-ai-generate" @click="generate">
          {{ generating ? t('shihua.ai.generating') : items.length ? t('shihua.ai.regenerate') : t('shihua.ai.generate') }}
        </button>
        <span class="text-xs text-gray-400">{{ t('shihua.ai.hint') }}</span>
      </div>
    </div>

    <!-- step 2: the draft -->
    <div v-if="items.length" class="mt-4 space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700">
      <div class="flex flex-wrap items-center gap-3 text-xs">
        <span v-for="(n, s) in counts" :key="s" class="rounded px-1.5 py-0.5" :class="statusClass(s)">{{ t(`shihua.ai.status.${s}`) }} {{ n }}</span>
        <span class="ml-auto text-gray-500">{{ t('shihua.ai.selected', { n: kept.length }) }}</span>
      </div>
      <div
        v-for="(it, i) in items"
        :key="i"
        class="rounded-lg border p-3"
        :class="[borderClass(it.status), checkable(it) ? '' : 'opacity-60']"
        :data-testid="`shihua-ai-item-${it.work.slug}`"
      >
        <div class="flex flex-wrap items-start gap-3">
          <input v-model="keep[i]" type="checkbox" class="mt-1.5" :disabled="!checkable(it)" />
          <div class="min-w-0 flex-1 space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <input v-model="it.work.title" class="input w-44 py-1 text-sm font-medium" />
              <input v-model="it.work.author" class="input w-24 py-1 text-sm" :placeholder="t('shihua.edit.author')" />
              <input v-model="it.work.dynasty" class="input w-16 py-1 text-sm" :placeholder="t('shihua.edit.dynasty')" />
              <select v-model="it.work.genre" class="input w-20 py-1 text-sm">
                <option v-for="g in genres" :key="g" :value="g">{{ t(`shihua.genres.${g}`) }}</option>
              </select>
              <select v-model="it.work.stage" class="input w-20 py-1 text-sm">
                <option v-for="s in stages" :key="s" :value="s">{{ t(`shihua.stages.${s}`) }}</option>
              </select>
              <input v-model="it.work.grade" class="input w-20 py-1 text-sm" :placeholder="t('shihua.ai.grade')" />
              <span class="rounded px-1.5 py-0.5 text-xs" :class="statusClass(it.status)">{{ t(`shihua.ai.status.${it.status}`) }}</span>
              <a v-if="it.page_url" :href="it.page_url" target="_blank" rel="noopener" class="text-xs text-primary-600 hover:underline">{{ t('shihua.ai.source', { page: it.page }) }}</a>
            </div>
            <p v-if="it.problem" class="text-xs text-red-600">{{ it.problem }}</p>
            <p v-if="it.diffs.length" class="text-xs text-amber-700 dark:text-amber-300" data-testid="shihua-ai-diffs">
              {{ t('shihua.ai.diffs') }}
              <span v-for="(d, j) in it.diffs" :key="j" class="mr-2">维基文库「<b>{{ d.source || '—' }}</b>」/ AI「<b>{{ d.ai || '—' }}</b>」</span>
            </p>
            <textarea v-if="checkable(it)" v-model="it.work.full_text" rows="3" class="input font-serif text-sm" />
            <p v-else class="whitespace-pre-line text-sm text-gray-500">{{ it.work.full_text || it.ai_text }}</p>
            <div class="flex flex-wrap gap-x-4 text-xs text-gray-400">
              <span v-if="it.reason">{{ it.reason }}</span>
              <details v-if="it.ai_text && it.status !== 'ai_only'">
                <summary class="cursor-pointer">{{ t('shihua.ai.aiText') }}</summary>
                <pre class="mt-1 whitespace-pre-wrap font-serif text-gray-600 dark:text-gray-300">{{ it.ai_text }}</pre>
              </details>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <label v-if="items.length" class="mr-auto flex items-center gap-1 text-sm text-gray-600 dark:text-gray-300">
        <input v-model="verify1" type="checkbox" data-testid="shihua-ai-verify1" /> {{ t('shihua.ai.verify1') }}
      </label>
      <button class="btn btn-secondary" @click="close">{{ t('shihua.edit.cancel') }}</button>
      <button class="btn btn-primary" :disabled="!kept.length || saving" data-testid="shihua-ai-confirm" @click="confirm">
        {{ saving ? t('shihua.ai.importing') : t('shihua.ai.confirm', { n: kept.length }) }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/shihua'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; imported: [slugs: string[]] }>()
const { t } = useI18n()
const appStore = useAppStore()

const modes = ['topic', 'titles'] as const
const stages = ['primary', 'junior', 'senior'] as const
const genres = ['shi', 'ci', 'qu', 'wen', 'fu'] as const

const mode = ref<(typeof modes)[number]>('topic')
const topic = ref('')
const titles = ref('')
const stage = ref('')
const count = ref(10)
const generating = ref(false)
const saving = ref(false)
const verify1 = ref(false)
const items = ref<api.ShihuaDraftItem[]>([])
const keep = ref<boolean[]>([])

const canGenerate = computed(() => (mode.value === 'topic' ? topic.value.trim() : titles.value.trim()) !== '')
const kept = computed(() => items.value.filter((it, i) => keep.value[i] && checkable(it)))
const counts = computed(() => {
  const c: Partial<Record<api.ShihuaDraftStatus, number>> = {}
  for (const it of items.value) c[it.status] = (c[it.status] ?? 0) + 1
  return c
})

function checkable(it: api.ShihuaDraftItem) {
  return it.status === 'match' || it.status === 'diff' || it.status === 'ai_only'
}

function statusClass(s: api.ShihuaDraftStatus) {
  return {
    match: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
    diff: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    ai_only: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300',
    exists: 'bg-gray-100 text-gray-500 dark:bg-dark-700',
    rejected: 'bg-gray-100 text-gray-500 dark:bg-dark-700'
  }[s]
}

function borderClass(s: api.ShihuaDraftStatus) {
  return { match: 'border-emerald-200', diff: 'border-amber-300', ai_only: 'border-red-300', exists: 'border-gray-200', rejected: 'border-gray-200' }[s] + ' dark:border-dark-600'
}

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function generate() {
  generating.value = true
  try {
    const res = await api.draftWorks(
      mode.value === 'topic'
        ? { topic: topic.value.trim(), stage: stage.value, count: Math.min(30, Math.max(1, count.value || 10)) }
        : { titles: titles.value.trim(), stage: stage.value }
    )
    items.value = res.items
    keep.value = res.items.map((it) => checkable(it))
  } catch (e) {
    fail(e)
  } finally {
    generating.value = false
  }
}

async function confirm() {
  const works = kept.value.map((it) => it.work)
  saving.value = true
  try {
    const res = await api.importWorks(works, false)
    if (res.errors.length) appStore.showError(res.errors.join('；'))
    const slugs = works.map((w) => w.slug)
    if (verify1.value && res.created + res.updated > 0) {
      const all = await api.listWorks()
      for (const w of all.filter((w) => slugs.includes(w.slug) && w.text_status === 'draft')) await api.verifyWork(w.id, 1)
    }
    appStore.showSuccess(t('shihua.import.done', { created: res.created, updated: res.updated }))
    emit('imported', slugs)
    items.value = []
    keep.value = []
    emit('close')
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

function close() {
  if (generating.value || saving.value) return
  emit('close')
}
</script>
