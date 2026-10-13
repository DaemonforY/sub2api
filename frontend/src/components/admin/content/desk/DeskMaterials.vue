<template>
  <div class="space-y-4" data-testid="desk-materials">
    <div class="card space-y-2 p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.materials.title') }}</h3>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.materials.hint') }}</p>
        </div>
        <div class="text-right">
          <div :class="['text-2xl font-semibold', usable >= 5 ? 'text-emerald-600' : 'text-amber-600']">{{ usable }} / 5</div>
          <div class="text-xs text-gray-500">{{ t('admin.desk.materials.usable') }}</div>
        </div>
      </div>
      <p v-if="usable < 5" class="text-xs text-amber-700 dark:text-amber-300">{{ t('admin.desk.materials.notEnough') }}</p>
    </div>

    <!-- Questions -->
    <div v-if="desk.questions.length" class="card space-y-3 p-4">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.materials.questions') }}</h4>
      <div v-for="q in desk.questions" :key="q.id" class="space-y-1">
        <div class="text-sm text-gray-800 dark:text-dark-100">{{ q.question }}</div>
        <div v-if="q.why" class="text-xs text-gray-400">{{ q.why }}</div>
        <textarea v-model="answers[q.id]" class="input min-h-[64px] text-sm" :placeholder="t('admin.desk.materials.answerPlaceholder')" :data-testid="`desk-answer-${q.id}`" />
      </div>
    </div>

    <!-- Fact cards -->
    <div class="card space-y-2 p-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.materials.facts') }}</h4>
        <div class="flex flex-wrap gap-2 text-xs">
          <span v-for="k in ['user', 'persona', 'answer', 'web']" :key="k" class="rounded bg-gray-100 px-1.5 py-0.5 text-gray-600 dark:bg-dark-700 dark:text-dark-300">
            {{ t(`admin.desk.fact.kind.${k}`) }} {{ facts.filter((f) => f.kind === k).length }}
          </span>
        </div>
      </div>
      <p v-if="!facts.length" class="py-4 text-center text-sm text-gray-400">{{ busy ? t('admin.desk.materials.researching') : t('admin.desk.materials.empty') }}</p>
      <ul class="divide-y divide-gray-100 dark:divide-dark-700">
        <li v-for="f in facts" :key="f.id || f.text" :class="['flex flex-col gap-2 py-2.5 sm:flex-row sm:items-start', f.status === 'rejected' ? 'opacity-50' : '']">
          <div class="min-w-0 flex-1">
            <textarea v-model="f.text" rows="1" class="w-full resize-none border-0 bg-transparent p-0 text-sm text-gray-800 focus:ring-0 dark:text-dark-100" @input="autoGrow" />
            <div class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-gray-400">
              <span class="rounded bg-gray-100 px-1.5 dark:bg-dark-700">{{ t(`admin.desk.fact.kind.${f.kind}`) }}</span>
              <a v-if="f.source && f.source.startsWith('http')" :href="f.source" target="_blank" rel="noopener" class="truncate text-primary-600 hover:underline">{{ f.source_title || f.source }}</a>
            </div>
          </div>
          <div class="flex flex-shrink-0 gap-1">
            <button
              v-for="st in (['ok', 'doubt', 'rejected'] as const)"
              :key="st"
              type="button"
              :class="['rounded px-2 py-0.5 text-xs', f.status === st ? statusOn(st) : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400']"
              @click="f.status = st"
            >{{ t(`admin.desk.fact.status.${st}`) }}</button>
          </div>
        </li>
      </ul>
      <div class="flex gap-2 pt-2">
        <input v-model.trim="newFact" class="input min-w-0 flex-1" :placeholder="t('admin.desk.materials.addPlaceholder')" @keyup.enter="addOwn" />
        <button class="btn btn-secondary" :disabled="!newFact" @click="addOwn">{{ t('admin.desk.materials.add') }}</button>
      </div>
    </div>

    <div class="card flex flex-wrap items-center gap-2 p-4">
      <input v-model.trim="hint" class="input min-w-0 flex-1" :placeholder="t('admin.desk.materials.hintPlaceholder')" />
      <button class="btn btn-secondary" :disabled="busy" @click="again">{{ t('admin.desk.materials.again') }}</button>
      <button class="btn btn-secondary" :disabled="busy || !dirty" @click="save()">{{ t('admin.desk.materials.save') }}</button>
      <button class="btn btn-primary" :disabled="busy || usable === 0" data-testid="desk-to-angles" @click="next">{{ t('admin.desk.materials.next') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, Fact, FactStatus } from '@/api/admin/content'

const props = defineProps<{ item: ContentItem; busy: boolean }>()
const emit = defineEmits<{ (e: 'updated', it: ContentItem): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const desk = computed(() => props.item.desk!)
const facts = ref<Fact[]>([])
const answers = reactive<Record<string, string>>({})
const newFact = ref('')
const hint = ref('')
const original = ref('')

function load() {
  facts.value = desk.value.facts.map((f) => ({ ...f, status: f.status === 'new' ? 'ok' : f.status }))
  for (const q of desk.value.questions) answers[q.id] = q.answer || ''
  original.value = snapshot()
}
// New facts arrive when a research run finishes; keep local edits otherwise.
watch(() => desk.value.facts.length + ':' + desk.value.questions.length, load, { immediate: true })

function snapshot() {
  return JSON.stringify([facts.value.map((f) => [f.id, f.text, f.status]), answers])
}
const dirty = computed(() => snapshot() !== original.value)
const usable = computed(() => facts.value.filter((f) => f.status !== 'rejected').length + Object.entries(answers).filter(([id, a]) => a.trim() && !desk.value.questions.find((q) => q.id === id)?.answer).length)

function statusOn(st: FactStatus) {
  if (st === 'ok') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (st === 'doubt') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
}

function addOwn() {
  if (!newFact.value) return
  facts.value.push({ id: '', text: newFact.value, kind: 'user', status: 'ok' })
  newFact.value = ''
}

function autoGrow(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = 'auto'
  el.style.height = `${el.scrollHeight}px`
}

async function save(quiet = false) {
  try {
    const it = await contentAPI.saveMaterials(props.item.id, facts.value, { ...answers })
    emit('updated', it)
    if (!quiet) appStore.showSuccess(t('admin.desk.materials.saved'))
    return true
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
    return false
  }
}

async function again() {
  if (dirty.value && !(await save(true))) return
  try {
    emit('updated', await contentAPI.research(props.item.id, hint.value))
    hint.value = ''
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function next() {
  if (dirty.value && !(await save(true))) return
  try {
    emit('updated', await contentAPI.makeAngles(props.item.id))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
</script>
