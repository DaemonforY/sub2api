<template>
  <div class="grid gap-4 lg:grid-cols-[minmax(300px,380px)_1fr]">
    <div class="space-y-3">
      <div class="card space-y-2 p-3">
        <input v-model="q" class="input text-sm" :placeholder="t('materials.texts.search')" data-testid="materials-text-search" @keyup.enter="reload" />
        <div class="flex gap-2">
          <select v-model="kind" class="input text-sm" @change="reload">
            <option value="">{{ t('materials.texts.allKinds') }}</option>
            <option v-for="k in kinds" :key="k" :value="k">{{ t(`materials.kinds.${k}`) }}</option>
          </select>
          <select v-model="license" class="input text-sm" @change="reload">
            <option value="">{{ t('materials.texts.allLicenses') }}</option>
            <option v-for="l in licenses" :key="l" :value="l">{{ t(`materials.licenses.${l}`) }}</option>
          </select>
        </div>
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs text-gray-500">{{ t('materials.texts.total', { n: total }) }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="importing" data-testid="materials-import" @click="importShihua">
            {{ importing ? t('materials.texts.importing') : t('materials.texts.import') }}
          </button>
        </div>
        <details v-if="skipped.length" class="text-xs text-amber-700 dark:text-amber-300">
          <summary class="cursor-pointer">{{ t('materials.texts.skippedTitle') }}（{{ skipped.length }}）</summary>
          <ul class="mt-1 space-y-0.5">
            <li v-for="s in skipped" :key="s.slug">《{{ s.title }}》{{ s.reason }}</li>
          </ul>
        </details>
      </div>

      <div v-if="!texts.length && !loading" class="card p-6 text-center text-sm text-gray-400">{{ t('materials.texts.empty') }}</div>
      <div v-for="text in texts" :key="text.id" class="card overflow-hidden" :data-testid="`materials-text-${text.slug}`">
        <button class="flex w-full items-start justify-between gap-2 p-3 text-left hover:bg-gray-50 dark:hover:bg-dark-700" @click="toggle(text)">
          <div class="min-w-0">
            <div class="truncate font-medium text-gray-900 dark:text-white">{{ text.title }}</div>
            <div class="text-xs text-gray-500">{{ text.dynasty }} {{ text.author_name }} · {{ t(`materials.kinds.${text.kind}`) }} · {{ t('materials.texts.segments', { n: text.segment_count }) }}</div>
          </div>
          <span class="shrink-0 rounded px-1.5 py-0.5 text-[10px]" :class="text.license_status === 'reference_only' ? 'bg-amber-100 text-amber-700' : 'bg-emerald-50 text-emerald-700'">
            {{ t(`materials.licenses.${text.license_status}`) }}
          </span>
        </button>
        <div v-if="openId === text.id" class="border-t border-gray-100 dark:border-dark-700">
          <p v-if="!openSegments.length" class="p-3 text-xs text-gray-400">{{ t('materials.loading') }}</p>
          <button
            v-for="seg in openSegments"
            :key="seg.id"
            class="block w-full border-b border-gray-50 px-3 py-2 text-left text-sm last:border-0 dark:border-dark-700"
            :class="segmentId === seg.id ? 'bg-primary-50 dark:bg-primary-900/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
            :data-testid="`materials-segment-${seg.id}`"
            @click="segmentId = seg.id"
          >
            <span class="text-xs text-gray-400">{{ seg.label }}</span>
            <span v-if="seg.is_famous" class="ml-1 rounded bg-rose-50 px-1 text-[10px] text-rose-600">{{ t('materials.texts.famous') }}</span>
            <div class="text-gray-800 dark:text-gray-100">{{ seg.content || seg.scene_desc || t('materials.texts.noScene') }}</div>
          </button>
        </div>
      </div>
      <button v-if="texts.length < total" class="btn btn-secondary btn-sm w-full" :disabled="loading" @click="more">…</button>
    </div>

    <MaterialWorkbench :segment-id="segmentId" :tick="tick" @queued="emit('queued')" @saved="refreshOpen" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'
import MaterialWorkbench from './MaterialWorkbench.vue'

defineProps<{ tick: number }>()
const emit = defineEmits<{ queued: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const kinds = ['shi', 'ci', 'qu', 'fu', 'wen', 'modern_poem', 'song', 'original'] as const
const licenses = ['public_domain', 'original', 'licensed', 'reference_only'] as const
const PAGE = 50

const params = new URLSearchParams(location.search)
const q = ref('')
const kind = ref('')
const license = ref('')
const texts = ref<api.MaterialText[]>([])
const total = ref(0)
const loading = ref(false)
const importing = ref(false)
const skipped = ref<api.MaterialImportResult['skipped']>([])
const openId = ref<number | null>(Number(params.get('text')) || null)
const openSegments = ref<api.MaterialSegment[]>([])
const segmentId = ref<number | null>(Number(params.get('seg')) || null)

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function remember(key: string, value: number | null) {
  const url = new URL(location.href)
  if (value) url.searchParams.set(key, String(value))
  else url.searchParams.delete(key)
  history.replaceState(history.state, '', url)
}
watch(openId, (v) => remember('text', v))
watch(segmentId, (v) => remember('seg', v))

async function fetchPage(offset: number) {
  loading.value = true
  try {
    const page = await api.listTexts({ q: q.value.trim() || undefined, kind: kind.value || undefined, license: license.value || undefined, limit: PAGE, offset })
    texts.value = offset ? [...texts.value, ...page.items] : page.items
    total.value = page.total
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
}

const reload = () => fetchPage(0)
const more = () => fetchPage(texts.value.length)

async function loadSegments(id: number) {
  openSegments.value = []
  try {
    openSegments.value = (await api.getText(id)).segments ?? []
  } catch (e) {
    fail(e)
  }
}

function toggle(text: api.MaterialText) {
  if (openId.value === text.id) {
    openId.value = null
    return
  }
  openId.value = text.id
  void loadSegments(text.id)
}

// The workbench saved a scene description: show it in the list too.
function refreshOpen() {
  if (openId.value) void loadSegments(openId.value)
}

async function importShihua() {
  importing.value = true
  try {
    const res = await api.importShihua()
    skipped.value = res.skipped
    appStore.showSuccess(t('materials.texts.imported', { imported: res.imported, existing: res.existing, skipped: res.skipped.length }))
    await reload()
  } catch (e) {
    fail(e)
  } finally {
    importing.value = false
  }
}

onMounted(() => {
  void reload()
  if (openId.value) void loadSegments(openId.value)
})
</script>
