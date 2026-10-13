<template>
  <div class="space-y-4" data-testid="content-item-panel">
    <div class="flex items-center justify-between gap-2">
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('close')">← {{ t('admin.content.review.back') }}</button>
      <div class="flex gap-2">
        <button v-if="item?.desk" type="button" class="btn btn-secondary btn-sm" @click="emit('desk')">{{ t('admin.desk.view.openDesk') }}</button>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ t('admin.content.common.refresh') }}</button>
      </div>
    </div>

    <p v-if="!item && loading" class="py-8 text-center text-sm text-gray-400">{{ t('admin.content.common.loading') }}</p>

    <template v-if="item">
      <div class="card space-y-3 p-4">
        <div class="flex flex-wrap items-start justify-between gap-2">
          <div class="min-w-0">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ item.title || item.brief.topic }}</h2>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
              <span :class="['rounded px-2 py-0.5 font-medium', itemStatusClass(item.status)]">{{ t(`admin.content.itemStatus.${item.status}`) }}</span>
              <span>{{ t('admin.content.review.created', { time: fmtTime(item.created_at) }) }}</span>
              <span>{{ t('admin.content.review.tokens', { n: (item.prompt_tokens + item.completion_tokens).toLocaleString() }) }}</span>
              <span v-if="item.model">{{ item.model }}</span>
              <span v-if="item.landing_path">{{ t('admin.content.item.landing') }}：<a :href="item.landing_path" target="_blank" class="text-primary-600 hover:underline">{{ item.landing_path }}</a></span>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <button v-if="item.running" class="btn btn-secondary btn-sm" @click="cancel">{{ t('admin.content.item.cancel') }}</button>
            <button v-if="item.status === 'failed' || item.status === 'canceled'" class="btn btn-primary btn-sm" :disabled="busy" @click="retry">{{ t('admin.content.item.retry') }}</button>
            <button v-if="!item.running && item.status !== 'rejected'" class="btn btn-secondary btn-sm" @click="rejectOpen = !rejectOpen">{{ t('admin.content.item.reject') }}</button>
            <button v-if="!item.running" class="btn btn-secondary btn-sm text-red-600" @click="remove">{{ t('admin.content.item.delete') }}</button>
          </div>
        </div>

        <p v-if="item.error" class="rounded-lg bg-red-50 p-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ t('admin.content.item.error', { msg: item.error }) }}</p>
        <p v-if="item.reject_reason" class="text-sm text-gray-500">{{ t('admin.content.item.rejectedWith', { reason: item.reject_reason }) }}</p>

        <div v-if="rejectOpen" class="space-y-2 rounded-lg border border-gray-100 p-3 dark:border-dark-700">
          <div class="text-sm font-medium">{{ t('admin.content.item.rejectTitle') }}</div>
          <input v-model="rejectReason" class="input" maxlength="300" :placeholder="t('admin.content.item.rejectReasonPlaceholder')" />
          <label class="flex items-center gap-2 text-sm">
            <input v-model="rejectAsRule" type="checkbox" class="h-4 w-4" />
            {{ t('admin.content.item.rejectAsRule') }}
          </label>
          <div class="flex gap-2">
            <button class="btn btn-primary btn-sm" :disabled="busy" @click="reject">{{ t('admin.content.common.confirm') }}</button>
            <button class="btn btn-secondary btn-sm" @click="rejectOpen = false">{{ t('admin.content.common.cancel') }}</button>
          </div>
        </div>

        <!-- Progress while running -->
        <div v-if="item.running || ['outlining', 'drafting', 'reviewing', 'humanizing', 'adapting'].includes(item.status)" class="space-y-1">
          <div class="text-xs font-medium text-gray-500">{{ t('admin.content.item.events') }}</div>
          <ul class="max-h-40 space-y-0.5 overflow-auto text-xs text-gray-600 dark:text-dark-300">
            <li v-for="(e, i) in item.events.slice(-12)" :key="i" :class="e.kind === 'error' ? 'text-red-600' : ''">{{ fmtTime(e.at) }} · {{ e.text }}</li>
          </ul>
        </div>

        <ul v-if="item.checks?.length" class="space-y-1">
          <li v-for="(c, i) in item.checks" :key="i" class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">{{ c.text }}</li>
        </ul>

        <div v-if="item.markdown || item.draft">
          <button type="button" class="text-sm text-primary-600 hover:underline" @click="showMaster = !showMaster">
            {{ showMaster ? t('admin.content.item.hideMaster') : t('admin.content.item.showMaster') }}
          </button>
          <div v-if="showMaster" class="mt-3 space-y-3">
            <div v-if="item.review_notes?.length" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
              <div class="mb-1 text-xs font-medium text-gray-500">
                {{ t('admin.content.item.reviewNotes') }}
                <span v-if="item.ai_score" class="ml-2 font-normal">{{ t('admin.content.item.aiScore', { n: item.ai_score }) }}</span>
              </div>
              <ul class="list-disc space-y-0.5 pl-5 text-gray-700 dark:text-dark-200">
                <li v-for="(n, i) in item.review_notes" :key="i">{{ n }}</li>
              </ul>
            </div>
            <div class="prose prose-sm max-w-none rounded-lg border border-gray-100 p-4 dark:prose-invert dark:border-dark-700" v-html="masterHtml" />
            <div v-if="item.sources?.length" class="text-xs text-gray-500">
              <div class="font-medium">{{ t('admin.content.item.sources') }}</div>
              <a v-for="s in item.sources" :key="s.url" :href="s.url" target="_blank" rel="noopener" class="block truncate hover:underline">{{ s.title || s.url }}</a>
            </div>
          </div>
        </div>
      </div>

      <!-- Platform versions -->
      <div v-if="item.versions.length" class="card p-4">
        <div class="mb-4 flex flex-wrap gap-2" role="tablist">
          <button
            v-for="v in item.versions"
            :key="v.id"
            type="button"
            role="tab"
            :data-testid="`content-tab-${v.platform}`"
            :class="['btn btn-sm', active === v.platform ? 'btn-primary' : 'btn-secondary']"
            @click="active = v.platform"
          >
            {{ t(`admin.content.platform.${v.platform}`) }}
            <span :class="['ml-1 inline-block h-2 w-2 rounded-full', dotClass(v.status)]" />
          </button>
        </div>
        <ContentVersionCard v-if="current" :key="current.id" :version="current" :spec="specs[current.platform]" :brand="brand" @updated="onUpdated" />
      </div>
      <p v-else-if="!item.running" class="text-center text-sm text-gray-400">{{ t('admin.content.item.noVersions') }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import * as contentAPI from '@/api/admin/content'
import type { ContentItemDetail, ContentPlatform, ContentVersion, ItemStatus, PlatformSpec, VersionStatus } from '@/api/admin/content'
import ContentVersionCard from './ContentVersionCard.vue'

const props = defineProps<{ itemId: number; platforms: PlatformSpec[]; brand: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'changed'): void; (e: 'desk'): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const item = ref<ContentItemDetail | null>(null)
const loading = ref(false)
const busy = ref(false)
const showMaster = ref(false)
const active = ref<ContentPlatform | ''>('')
const rejectOpen = ref(false)
const rejectReason = ref('')
const rejectAsRule = ref(true)
let timer: ReturnType<typeof setTimeout> | null = null

const specs = computed(() => Object.fromEntries(props.platforms.map((p) => [p.id, p])) as Record<ContentPlatform, PlatformSpec>)
const current = computed(() => item.value?.versions.find((v) => v.platform === active.value) || item.value?.versions[0])
const masterHtml = computed(() => renderMarkdown(item.value?.markdown || item.value?.draft || ''))

// Poll while the article (or one of its versions) is being written.
function busyWriting(d: ContentItemDetail) {
  return d.running || d.versions.some((v) => v.status === 'generating')
}

async function load() {
  loading.value = true
  try {
    const d = await contentAPI.getItem(props.itemId)
    item.value = d
    if (!active.value || !d.versions.some((v) => v.platform === active.value)) {
      active.value = (d.versions.find((v) => v.status === 'pending_review') || d.versions[0])?.platform || ''
    }
    schedule(d)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}

function schedule(d: ContentItemDetail) {
  if (timer) clearTimeout(timer)
  timer = busyWriting(d) ? setTimeout(load, 3000) : null
}

watch(() => props.itemId, load, { immediate: true })
onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})

function onUpdated(v: ContentVersion) {
  if (!item.value) return
  const i = item.value.versions.findIndex((x) => x.id === v.id)
  if (i >= 0) item.value.versions[i] = v
  emit('changed')
  if (v.status === 'generating') schedule(item.value)
  // Approving / skipping may finish the article; reload its status quietly.
  load()
}

async function run(fn: () => Promise<unknown>) {
  busy.value = true
  try {
    await fn()
    emit('changed')
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

const retry = () => run(() => contentAPI.retryItem(props.itemId))
const cancel = () => run(() => contentAPI.cancelItem(props.itemId))

async function reject() {
  await run(() => contentAPI.rejectItem(props.itemId, rejectReason.value.trim(), rejectAsRule.value))
  rejectOpen.value = false
  appStore.showSuccess(t('admin.content.item.rejected'))
}

async function remove() {
  if (!window.confirm(t('admin.content.item.deleteConfirm'))) return
  busy.value = true
  try {
    await contentAPI.deleteItem(props.itemId)
    emit('changed')
    emit('close')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

function itemStatusClass(s: ItemStatus) {
  if (s === 'pending_review') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
  if (s === 'done') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (s === 'failed') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  if (s === 'rejected' || s === 'canceled') return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  return 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
}

function dotClass(s: VersionStatus) {
  if (s === 'pending_review') return 'bg-amber-400'
  if (s === 'approved') return 'bg-primary-400'
  if (s === 'published') return 'bg-emerald-500'
  if (s === 'failed') return 'bg-red-500'
  if (s === 'generating') return 'animate-pulse bg-primary-400'
  return 'bg-gray-300'
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>
