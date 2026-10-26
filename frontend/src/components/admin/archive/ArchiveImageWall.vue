<template>
  <!-- 图片墙. Every list call is audited server-side, so nothing loads until the admin asks. -->
  <div class="card space-y-3 p-4" data-testid="ca-wall">
    <div>
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.conversationArchive.wall.title') }}</h3>
      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.auditNotice') }}</p>
    </div>

    <form class="flex flex-wrap items-end gap-2" data-testid="ca-wall-form" @submit.prevent="search">
      <input v-model.trim="q" type="text" class="input w-64" :placeholder="t('admin.conversationArchive.browser.q')" data-testid="ca-wall-q" />
      <label class="flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
        {{ t('admin.conversationArchive.browser.from') }}
        <input v-model="fromDate" type="date" class="input w-40" data-testid="ca-wall-from" />
      </label>
      <label class="flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
        {{ t('admin.conversationArchive.browser.to') }}
        <input v-model="toDate" type="date" class="input w-40" data-testid="ca-wall-to" />
      </label>
      <select v-model="model" class="input w-48" data-testid="ca-wall-model">
        <option value="">{{ t('admin.conversationArchive.wall.allModels') }}</option>
        <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
      </select>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="loading" data-testid="ca-wall-search">
        {{ searched ? t('admin.conversationArchive.browser.search') : t('admin.conversationArchive.wall.load') }}
      </button>
    </form>

    <div v-if="error" class="text-sm text-red-600" data-testid="ca-wall-error">{{ t('admin.conversationArchive.loadFailed') }}：{{ error }}</div>

    <p v-if="!searched" class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.hint') }}</p>
    <template v-else>
      <div class="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-6 xl:grid-cols-8" data-testid="ca-wall-grid">
        <ArchiveWallTile
          v-for="(it, i) in items"
          :key="it.id"
          :item="it"
          :active="openIndex === i"
          @open="openAt(i)"
          @loaded="thumbs.set(it.id, $event)"
        />
      </div>
      <p v-if="!items.length && !loading" class="text-sm text-gray-500 dark:text-dark-400" data-testid="ca-wall-empty">{{ t('admin.conversationArchive.wall.empty') }}</p>
      <div class="flex items-center justify-center gap-3 text-xs text-gray-500 dark:text-dark-400">
        <span v-if="items.length" data-testid="ca-wall-count">{{ t('admin.conversationArchive.wall.count', { n: items.length }) }}</span>
        <button v-if="nextBeforeId" type="button" class="btn btn-secondary btn-sm" :disabled="loading" data-testid="ca-wall-more" @click="loadMore">
          {{ loading ? t('common.loading') : t('admin.conversationArchive.browser.loadMore') }}
        </button>
      </div>
    </template>

    <!-- Detail: the original, the prompt and where it came from -->
    <div
      v-if="openItem"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      data-testid="ca-wall-detail"
      @click="closeDetail"
    >
      <div class="flex max-h-full w-full max-w-6xl flex-col overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-dark-900 md:flex-row" @click.stop>
        <div class="relative flex min-h-[16rem] flex-1 items-center justify-center bg-gray-950">
          <img v-if="detailSrc" :src="detailSrc" alt="" class="max-h-[80vh] max-w-full object-contain" data-testid="ca-wall-detail-img" />
          <span v-else-if="detailFailed" class="text-sm text-red-400">{{ t('admin.conversationArchive.transcript.imageFailed') }}</span>
          <span v-else class="text-sm text-gray-400">{{ t('common.loading') }}</span>
          <button
            v-if="openIndex > 0"
            type="button"
            class="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-black/50 px-3 py-2 text-white hover:bg-black/70"
            :aria-label="t('admin.conversationArchive.wall.prev')"
            data-testid="ca-wall-prev"
            @click="openAt(openIndex - 1)"
          >
            ‹
          </button>
          <button
            v-if="openIndex < items.length - 1"
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-black/50 px-3 py-2 text-white hover:bg-black/70"
            :aria-label="t('admin.conversationArchive.wall.next')"
            data-testid="ca-wall-next"
            @click="openAt(openIndex + 1)"
          >
            ›
          </button>
        </div>
        <div class="flex w-full flex-col gap-3 overflow-y-auto p-4 text-sm md:w-80">
          <div>
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ openItem.email ? displayEmail(openItem.email, t) : `#${openItem.user_id}` }}
              <span class="text-xs font-normal text-gray-500">#{{ openItem.user_id }}</span>
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(openItem.created_at) }}</div>
          </div>
          <dl class="space-y-1 text-xs">
            <div class="flex gap-2">
              <dt class="w-14 shrink-0 text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.model') }}</dt>
              <dd class="break-all font-mono text-gray-900 dark:text-white">{{ openItem.model || '–' }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-14 shrink-0 text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.endpoint') }}</dt>
              <dd class="break-all font-mono text-gray-900 dark:text-white">{{ openItem.inbound_endpoint || '–' }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-14 shrink-0 text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.size') }}</dt>
              <dd class="text-gray-900 dark:text-white">{{ openItem.mime }} · {{ formatBytes(openItem.bytes, 1) }}</dd>
            </div>
            <div class="flex gap-2">
              <dt class="w-14 shrink-0 text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.wall.requestId') }}</dt>
              <dd class="min-w-0 break-all font-mono text-gray-900 dark:text-white">
                {{ openItem.request_id || '–' }}
                <button
                  v-if="openItem.request_id"
                  type="button"
                  class="ml-1 text-primary-600 hover:underline dark:text-primary-400"
                  @click="copyToClipboard(openItem.request_id)"
                >
                  {{ t('admin.conversationArchive.wall.copy') }}
                </button>
              </dd>
            </div>
          </dl>
          <div>
            <div class="mb-1 flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
              <span>{{ t('admin.conversationArchive.wall.prompt') }}</span>
              <button v-if="openItem.prompt" type="button" class="text-primary-600 hover:underline dark:text-primary-400" @click="copyToClipboard(openItem.prompt)">
                {{ t('admin.conversationArchive.wall.copy') }}
              </button>
            </div>
            <p class="whitespace-pre-wrap break-words rounded bg-gray-50 p-2 text-gray-800 dark:bg-dark-800 dark:text-dark-100" data-testid="ca-wall-prompt">
              {{ openItem.prompt || t('admin.conversationArchive.wall.noPrompt') }}
            </p>
            <p class="mt-1 text-[11px] text-gray-400">{{ t('admin.conversationArchive.wall.promptNote') }}</p>
          </div>
          <div class="mt-auto flex flex-wrap gap-2">
            <button
              type="button"
              class="btn btn-primary btn-sm"
              :disabled="!openItem.session_key"
              data-testid="ca-wall-open-session"
              @click="openSession(openItem)"
            >
              {{ t('admin.conversationArchive.wall.openSession') }}
            </button>
            <a v-if="detailSrc" :href="detailSrc" :download="downloadName(openItem)" class="btn btn-secondary btn-sm">{{ t('admin.conversationArchive.wall.download') }}</a>
            <button type="button" class="btn btn-secondary btn-sm" data-testid="ca-wall-close" @click="closeDetail">{{ t('admin.conversationArchive.transcript.close') }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ArchiveWallTile from '@/components/admin/archive/ArchiveWallTile.vue'
import { getImage, listImages, type ArchiveWallImage, type ArchiveWallQuery } from '@/api/admin/conversationArchive'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { dayToRFC3339 } from '@/utils/archiveTranscript'
import { formatBytes, formatDateLocalInput, formatDateTime } from '@/utils/format'
import { displayEmail } from '@/utils/userDisplay'

const PAGE = 60

const emit = defineEmits<{ (e: 'open-session', item: ArchiveWallImage): void }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()

function daysAgo(n: number): Date {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d
}

const q = ref('')
const fromDate = ref(formatDateLocalInput(daysAgo(6)))
const toDate = ref(formatDateLocalInput(new Date()))
const model = ref('')
const models = ref<string[]>([])
const items = ref<ArchiveWallImage[]>([])
const nextBeforeId = ref<number | null>(null)
const loading = ref(false)
const error = ref('')
const searched = ref(false)
// thumbnails the tiles have loaded (owned and revoked by the tiles), shown while the original loads
const thumbs = new Map<number, string>()
// the filter the current wall was loaded with, so 加载更多 continues the same query
let activeQuery: ArchiveWallQuery = {}
let generation = 0

// keep a chosen model selectable even when the server's list (last 90 days) doesn't have it
const modelOptions = computed(() => (model.value && !models.value.includes(model.value) ? [model.value, ...models.value] : models.value))

async function fetchPage(beforeId?: number) {
  const gen = ++generation
  loading.value = true
  error.value = ''
  try {
    const page = await listImages({ ...activeQuery, before_id: beforeId, limit: PAGE })
    if (gen !== generation) return
    const got = page?.items || []
    items.value = beforeId ? [...items.value, ...got] : got
    nextBeforeId.value = page?.next_before_id || null
    if (page?.models) models.value = page.models
    searched.value = true
  } catch (e) {
    if (gen === generation) error.value = extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed'))
  } finally {
    if (gen === generation) loading.value = false
  }
}

function search() {
  closeDetail()
  thumbs.clear()
  activeQuery = {
    q: q.value || undefined,
    model: model.value || undefined,
    from: dayToRFC3339(fromDate.value),
    to: dayToRFC3339(toDate.value, true)
  }
  return fetchPage()
}

function loadMore() {
  if (nextBeforeId.value && !loading.value) return fetchPage(nextBeforeId.value)
}

// ---------- Detail ----------

const openIndex = ref(-1)
const openItem = computed(() => (openIndex.value >= 0 ? items.value[openIndex.value] || null : null))
const detailSrc = ref('')
const detailFailed = ref(false)
let ownedSrc = '' // the original's object URL, revoked when the detail moves on or closes
let detailGen = 0

function releaseOriginal() {
  if (ownedSrc) URL.revokeObjectURL(ownedSrc)
  ownedSrc = ''
}

async function openAt(i: number) {
  const it = items.value[i]
  if (!it) return
  const gen = ++detailGen
  releaseOriginal()
  openIndex.value = i
  detailFailed.value = false
  detailSrc.value = thumbs.get(it.id) || ''
  try {
    const blob = await getImage(it.id)
    const url = URL.createObjectURL(blob)
    if (gen !== detailGen) {
      URL.revokeObjectURL(url)
      return
    }
    ownedSrc = url
    detailSrc.value = url
  } catch {
    if (gen === detailGen && !detailSrc.value) detailFailed.value = true
  }
}

function closeDetail() {
  detailGen++
  releaseOriginal()
  openIndex.value = -1
  detailSrc.value = ''
  detailFailed.value = false
}

function openSession(it: ArchiveWallImage) {
  closeDetail()
  emit('open-session', it)
}

function downloadName(it: ArchiveWallImage): string {
  const ext = (it.mime.split('/')[1] || 'png').replace('jpeg', 'jpg')
  return `archive-${it.user_id}-${it.id}.${ext}`
}

function onKey(e: KeyboardEvent) {
  if (openIndex.value < 0) return
  if (e.key === 'Escape') closeDetail()
  else if (e.key === 'ArrowLeft' && openIndex.value > 0) void openAt(openIndex.value - 1)
  else if (e.key === 'ArrowRight' && openIndex.value < items.value.length - 1) void openAt(openIndex.value + 1)
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  closeDetail()
})
</script>
