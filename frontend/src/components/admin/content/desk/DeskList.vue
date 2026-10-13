<template>
  <div class="space-y-4" data-testid="desk-list">
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.list.newTitle') }}</h3>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.list.newHint') }}</p>
        </div>
        <button v-if="!formOpen" class="btn btn-primary btn-sm" data-testid="desk-new" @click="formOpen = true">{{ t('admin.desk.list.new') }}</button>
      </div>
      <p v-if="!personaReady" class="rounded-lg bg-amber-50 p-2 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.desk.list.noPersona') }}
        <button class="ml-1 font-medium underline" @click="emit('persona')">{{ t('admin.desk.list.goPersona') }}</button>
      </p>
      <div v-if="formOpen" class="space-y-3">
        <label class="block text-sm">
          <span class="input-label">{{ t('admin.desk.list.title') }}</span>
          <input v-model.trim="form.title" class="input" maxlength="200" :placeholder="t('admin.desk.list.titlePlaceholder')" data-testid="desk-title" />
        </label>
        <label class="block text-sm">
          <span class="input-label">{{ t('admin.desk.list.idea') }}</span>
          <input v-model.trim="form.idea" class="input" maxlength="500" :placeholder="t('admin.desk.list.ideaPlaceholder')" />
        </label>
        <label class="block text-sm">
          <span class="input-label">{{ t('admin.desk.list.notes') }}</span>
          <textarea v-model="form.notes" class="input min-h-[140px] text-sm" maxlength="6000" :placeholder="t('admin.desk.list.notesPlaceholder')" />
          <span class="input-hint">{{ t('admin.desk.list.notesHint') }}</span>
        </label>
        <div class="flex gap-2">
          <button class="btn btn-primary" :disabled="creating || !form.title" @click="create">{{ t('admin.desk.list.start') }}</button>
          <button class="btn btn-secondary" @click="formOpen = false">{{ t('admin.desk.common.cancel') }}</button>
        </div>
      </div>
    </div>

    <p v-if="!items.length && !loading" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.desk.list.empty') }}</p>
    <ul class="space-y-2">
      <li v-for="it in items" :key="it.id">
        <button type="button" class="card block w-full p-4 text-left hover:ring-1 hover:ring-primary-200" :data-testid="`desk-item-${it.id}`" @click="emit('open', it.id)">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ it.title || it.brief.topic }}</span>
            <span class="text-xs text-gray-400">{{ fmtTime(it.updated_at) }}</span>
          </div>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-1">
              <span
                v-for="(st, i) in STAGES"
                :key="st"
                :class="['h-1.5 w-8 rounded-full', i <= stageIndex(it) ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-700']"
              />
            </div>
            <span class="text-xs text-gray-600 dark:text-dark-300">{{ t(`admin.desk.stage.${it.desk?.stage || 'materials'}`) }}</span>
            <span v-if="it.desk?.job?.status === 'running'" class="animate-pulse text-xs text-primary-600">{{ t(`admin.desk.job.${it.desk.job.kind}`) }}…</span>
            <span v-else-if="it.desk?.job?.status === 'failed'" class="text-xs text-red-600">{{ t('admin.desk.list.failed') }}</span>
          </div>
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, DeskStage } from '@/api/admin/content'

const emit = defineEmits<{ (e: 'open', id: number): void; (e: 'persona'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const STAGES: DeskStage[] = ['materials', 'angles', 'drafts', 'edit', 'final']
const items = ref<ContentItem[]>([])
const loading = ref(false)
const creating = ref(false)
const formOpen = ref(false)
const personaReady = ref(true)
const form = reactive({ title: '', idea: '', notes: '' })

function stageIndex(it: ContentItem) {
  return STAGES.indexOf(it.desk?.stage || 'materials')
}

async function load() {
  loading.value = true
  try {
    const [list, p] = await Promise.all([contentAPI.listItems('desk'), contentAPI.getPersona()])
    items.value = list
    personaReady.value = !!(p.identity || p.voice || p.samples.length)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function create() {
  creating.value = true
  try {
    const it = await contentAPI.createDesk({ title: form.title, idea: form.idea, notes: form.notes })
    emit('open', it.id)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    creating.value = false
  }
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

defineExpose({ load })
</script>
