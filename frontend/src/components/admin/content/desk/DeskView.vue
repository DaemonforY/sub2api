<template>
  <div class="space-y-4" data-testid="desk-view">
    <div class="flex items-center justify-between gap-2">
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('close')">← {{ t('admin.desk.view.back') }}</button>
      <span v-if="item" class="text-xs text-gray-400">{{ t('admin.desk.view.tokens', { n: (item.prompt_tokens + item.completion_tokens).toLocaleString() }) }}</span>
    </div>

    <template v-if="item?.desk">
      <div class="card space-y-3 p-4">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ item.title || item.brief.topic }}</h2>
        <!-- Stepper: done steps can be revisited -->
        <ol class="flex flex-wrap items-center gap-1 text-sm">
          <li v-for="(st, i) in STAGES" :key="st" class="flex items-center gap-1">
            <button
              type="button"
              :disabled="!reachable(st) || running"
              :class="['rounded-full px-3 py-1', st === stage ? 'bg-primary-600 text-white' : reachable(st) ? 'bg-primary-50 text-primary-700 hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-300' : 'bg-gray-100 text-gray-400 dark:bg-dark-700']"
              :data-testid="`desk-step-${st}`"
              @click="goto(st)"
            >
              {{ i + 1 }}. {{ t(`admin.desk.stage.${st}`) }}
            </button>
            <span v-if="i < STAGES.length - 1" class="text-gray-300">›</span>
          </li>
        </ol>
        <!-- The running / failed AI step -->
        <div v-if="job?.status === 'running'" class="flex items-center justify-between gap-2 rounded-lg bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:bg-primary-900/20 dark:text-primary-300" data-testid="desk-job">
          <span class="animate-pulse">{{ t(`admin.desk.job.${job.kind}`) }}… {{ t('admin.desk.view.elapsed', { s: elapsed }) }}</span>
          <button class="text-xs hover:underline" @click="cancel">{{ t('admin.desk.view.stop') }}</button>
        </div>
        <div v-else-if="job?.status === 'failed'" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
          {{ t('admin.desk.view.failed', { step: t(`admin.desk.job.${job.kind}`), msg: job.error || '' }) }}
        </div>
      </div>

      <DeskMaterials v-if="stage === 'materials'" :item="item" :busy="running" @updated="onUpdated" />
      <DeskAngles v-else-if="stage === 'angles'" :item="item" :busy="running" @updated="onUpdated" @stage="goto" />
      <DeskDrafts v-else-if="stage === 'drafts'" :item="item" :busy="running" @updated="onUpdated" @stage="goto" />
      <DeskEditor v-else-if="stage === 'edit'" :key="editorKey" ref="editorRef" :item="item" :busy="running" @updated="onUpdated" @stage="goto" @reload="reloadEditor" />
      <DeskFinal v-else-if="stage === 'final'" :item="item" :busy="running" @updated="onUpdated" @stage="goto" />
    </template>
    <p v-else class="py-8 text-center text-sm text-gray-400">{{ t('admin.desk.common.loading') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, DeskStage } from '@/api/admin/content'
import DeskMaterials from './DeskMaterials.vue'
import DeskAngles from './DeskAngles.vue'
import DeskDrafts from './DeskDrafts.vue'
import DeskEditor from './DeskEditor.vue'
import DeskFinal from './DeskFinal.vue'

const props = defineProps<{ itemId: number }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'review', id: number): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const STAGES: DeskStage[] = ['materials', 'angles', 'drafts', 'edit', 'final']
const item = ref<ContentItem | null>(null)
const editorKey = ref(0)
const editorRef = ref<InstanceType<typeof DeskEditor> | null>(null)
const now = ref(Date.now())
let poll: ReturnType<typeof setTimeout> | null = null
let tick: ReturnType<typeof setInterval> | null = null

const stage = computed(() => item.value?.desk?.stage || 'materials')
const job = computed(() => item.value?.desk?.job)
const running = computed(() => job.value?.status === 'running')
const elapsed = computed(() => (job.value ? Math.max(0, Math.round((now.value - new Date(job.value.started_at).getTime()) / 1000)) : 0))

function reachable(st: DeskStage) {
  const d = item.value?.desk
  if (!d) return false
  switch (st) {
    case 'materials':
      return true
    case 'angles':
      return d.angles.length > 0 || d.stage !== 'materials'
    case 'drafts':
      return !!d.angle
    case 'edit':
      return !!d.body
    case 'final':
      return !!d.final_at
  }
  return false
}

async function load() {
  try {
    const it = await contentAPI.getItem(props.itemId)
    const before = item.value?.desk?.job
    item.value = it
    // An AI step that replaced the text (merge) remounts the editor with it.
    if (before?.status === 'running' && before.kind === 'merge' && it.desk?.job?.status !== 'running') editorKey.value++
    // Platform versions just got written: hand over to the 待发布 review.
    if (before?.status === 'running' && before.kind === 'platforms' && it.status === 'pending_review') emit('review', it.id)
    schedule()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function schedule() {
  if (poll) clearTimeout(poll)
  poll = running.value ? setTimeout(load, 2500) : null
}

watch(() => props.itemId, () => {
  item.value = null
  load()
}, { immediate: true })

tick = setInterval(() => {
  now.value = Date.now()
}, 1000)

onBeforeUnmount(() => {
  if (poll) clearTimeout(poll)
  if (tick) clearInterval(tick)
})

function onUpdated(it: ContentItem) {
  const stageChanged = it.desk?.stage !== item.value?.desk?.stage
  item.value = it
  if (stageChanged && it.desk?.stage === 'edit') editorKey.value++
  schedule()
}

function reloadEditor() {
  editorKey.value++
  load()
}

async function goto(st: DeskStage) {
  if (st === stage.value || !reachable(st)) return
  if (stage.value === 'edit') await editorRef.value?.save()
  try {
    onUpdated(await contentAPI.setStage(props.itemId, st))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function cancel() {
  try {
    await contentAPI.cancelDeskJob(props.itemId)
    setTimeout(load, 500)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
</script>
