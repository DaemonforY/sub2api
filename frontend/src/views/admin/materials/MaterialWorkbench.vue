<template>
  <div v-if="!segment" class="card p-10 text-center text-sm text-gray-400">{{ loading ? t('materials.loading') : t('materials.bench.empty') }}</div>
  <div v-else class="space-y-4" data-testid="materials-workbench">
    <!-- the segment -->
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <div class="font-semibold text-gray-900 dark:text-white">
          {{ segment.title }} · {{ segment.label }}
          <span class="text-sm font-normal text-gray-500">{{ segment.dynasty }} {{ segment.author_name }}</span>
        </div>
        <span class="text-xs" :class="overlayAllowed ? 'text-emerald-600' : 'text-amber-600'">
          {{ t(`materials.licenses.${segment.license_status}`) }} · {{ overlayAllowed ? t('materials.licenseHint.allowed') : t('materials.licenseHint.refOnly') }}
        </span>
      </div>
      <p v-if="segment.content" class="text-lg leading-relaxed text-gray-800 dark:text-gray-100" data-testid="materials-bench-content">{{ segment.content }}</p>
      <label class="block text-xs font-medium text-gray-500">{{ t('materials.bench.scene') }}</label>
      <textarea v-model="scene" rows="3" class="input text-sm" :placeholder="t('materials.bench.scenePlaceholder')" data-testid="materials-bench-scene" />
    </div>

    <!-- imagery tags -->
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.bench.imagery') }}</h3>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="extracting" data-testid="materials-extract" @click="extract">
            {{ extracting ? t('materials.bench.extracting') : t('materials.bench.extract') }}
          </button>
          <button class="btn btn-primary btn-sm" :disabled="saving" data-testid="materials-save-tags" @click="saveTags">{{ t('materials.bench.saveTags') }}</button>
        </div>
      </div>
      <p v-if="!tags.length" class="text-sm text-gray-400">{{ t('materials.bench.noImagery') }}</p>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="tag in tags"
          :key="tag.imagery_id"
          class="flex items-center gap-1.5 rounded-lg border px-2 py-1 text-sm"
          :class="tag.confirmed ? 'border-gray-200 dark:border-dark-600' : 'border-dashed border-sky-400'"
          :title="tag.visual_zh"
          :data-testid="`materials-tag-${tag.name}`"
        >
          <span class="font-medium text-gray-800 dark:text-gray-100">{{ tag.name }}</span>
          <select v-model.number="tag.weight" class="rounded border-0 bg-transparent py-0 pl-1 pr-6 text-xs text-gray-600 dark:text-gray-300">
            <option v-for="w in [3, 2, 1]" :key="w" :value="w">{{ t(`materials.bench.weights.${w}`) }}</option>
          </select>
          <span class="text-[10px] text-gray-400">{{ t(`materials.bench.source.${tag.source}`) }}</span>
          <span v-if="!tag.confirmed" class="text-[10px] text-sky-600">{{ t('materials.bench.unconfirmed') }}</span>
          <span v-if="tag.status !== 'published'" class="text-[10px] text-amber-600">{{ t('materials.bench.draftImagery') }}</span>
          <button class="text-gray-400 hover:text-red-500" :aria-label="`remove ${tag.name}`" @click="removeTag(tag.imagery_id)">×</button>
        </div>
      </div>
      <div class="relative max-w-xs">
        <input v-model="search" class="input text-sm" :placeholder="t('materials.bench.addPlaceholder')" data-testid="materials-add-imagery" />
        <div v-if="matches.length" class="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800">
          <button
            v-for="im in matches"
            :key="im.id"
            class="block w-full px-3 py-1.5 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
            @click="addTag(im)"
          >
            {{ im.name }} <span class="text-xs text-gray-400">{{ im.category }}<template v-if="im.aliases.length"> · {{ im.aliases.join('、') }}</template></span>
          </button>
        </div>
      </div>
    </div>

    <!-- styles and options -->
    <div class="card space-y-3 p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.bench.styles') }}</h3>
      <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
        <button
          v-for="r in recs"
          :key="r.style.id"
          class="rounded-lg border p-2 text-left text-sm"
          :class="styleId === r.style.id ? 'border-primary-500 ring-1 ring-primary-500' : 'border-gray-200 hover:border-gray-300 dark:border-dark-600'"
          :data-testid="`materials-style-${r.style.id}`"
          @click="styleId = r.style.id"
        >
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-800 dark:text-gray-100">{{ r.style.name }}</span>
            <span class="text-xs text-gray-500">{{ t('materials.bench.score', { score: r.score }) }}</span>
          </div>
          <div class="mt-0.5 text-xs text-gray-500">{{ r.reasons.length ? r.reasons.join('，') : t('materials.bench.noReason') }}</div>
        </button>
      </div>

      <div v-if="current" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label class="text-xs text-gray-500">
          {{ t('materials.bench.size') }}
          <select v-model="size" class="input mt-1 text-sm">
            <option v-for="s in sizes" :key="s" :value="s">{{ t(`materials.bench.sizes.${s || 'default'}`) }}</option>
          </select>
        </label>
        <label v-for="field in optionFields" :key="field.key" class="text-xs text-gray-500">
          {{ t(`materials.bench.${field.label}`) }}
          <select v-model="picks[field.key]" class="input mt-1 text-sm" :data-testid="`materials-option-${field.key}`">
            <option value="">{{ t('materials.bench.none') }}</option>
            <option v-for="o in current.options[field.list] || []" :key="o.name" :value="o.name" :title="o.prompt_en">{{ o.name }}</option>
          </select>
        </label>
      </div>
      <label class="block text-xs text-gray-500">
        {{ t('materials.bench.extra') }}
        <input v-model="extra" class="input mt-1 text-sm" />
      </label>

      <div class="flex flex-wrap items-center gap-2">
        <button class="btn btn-secondary btn-sm" :disabled="!styleId || previewing" data-testid="materials-preview" @click="preview">
          {{ previewing ? t('materials.bench.previewing') : t('materials.bench.preview') }}
        </button>
        <label class="ml-auto flex items-center gap-1 text-xs text-gray-500">
          {{ t('materials.bench.count') }}
          <select v-model.number="count" class="input w-16 py-1 text-sm">
            <option v-for="n in 4" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
        <button class="btn btn-primary btn-sm" :disabled="!styleId || drawing || prompt?.blocked" data-testid="materials-draw" @click="draw">{{ t('materials.bench.draw') }}</button>
      </div>

      <div v-if="prompt" class="space-y-2" data-testid="materials-prompt">
        <p v-if="prompt.blocked" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ t('materials.bench.blocked') }}</p>
        <ul v-if="prompt.hits.length" class="text-xs text-red-600">
          <li v-for="(h, i) in prompt.hits" :key="i">{{ t(`materials.rules.actions.${h.action}`) }}：{{ h.match }}<template v-if="h.reason"> — {{ h.reason }}</template></li>
        </ul>
        <ul v-if="prompt.notes.length" class="text-xs text-amber-600">
          <li v-for="(n, i) in prompt.notes" :key="i">{{ n }}</li>
        </ul>
        <pre class="max-h-72 overflow-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-200">{{ prompt.prompt }}</pre>
      </div>
    </div>

    <!-- artworks of the segment -->
    <div class="card space-y-3 p-4">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.bench.artworks') }}</h3>
        <button class="btn btn-secondary btn-sm" @click="loadArtworks">{{ t('materials.refresh') }}</button>
      </div>
      <p v-if="!artworks.length" class="text-sm text-gray-400">{{ t('materials.bench.noArtworks') }}</p>
      <MaterialArtworkGrid v-else :items="artworks" @changed="loadArtworks" @queued="emit('queued')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'
import MaterialArtworkGrid from './MaterialArtworkGrid.vue'

interface Tag {
  imagery_id: number
  name: string
  weight: number
  source: 'ai' | 'manual'
  confirmed: boolean
  status: api.MaterialStatus
  visual_zh: string
}

const props = defineProps<{ segmentId: number | null; tick: number }>()
const emit = defineEmits<{ queued: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const segment = ref<api.MaterialSegmentContext | null>(null)
const scene = ref('')
const tags = ref<Tag[]>([])
const library = ref<api.MaterialImagery[]>([])
const search = ref('')
const recs = ref<api.MaterialStyleScore[]>([])
const styleId = ref<number | null>(null)
const sizes = ['', '1024x1536', '1536x1024', '1024x1024'] as const
const size = ref<string>('')
const picks = reactive<Record<'composition' | 'palette' | 'mood', string>>({ composition: '', palette: '', mood: '' })
const optionFields = [
  { key: 'composition', label: 'composition', list: 'compositions' },
  { key: 'palette', label: 'palette', list: 'palettes' },
  { key: 'mood', label: 'mood', list: 'moods' }
] as const
const extra = ref('')
const count = ref(2)
const prompt = ref<api.MaterialPrompt | null>(null)
const artworks = ref<api.MaterialArtwork[]>([])
const extracting = ref(false)
const saving = ref(false)
const previewing = ref(false)
const drawing = ref(false)

const overlayAllowed = computed(() => segment.value?.license_status !== 'reference_only')
const current = computed(() => recs.value.find((r) => r.style.id === styleId.value) ?? null)
const matches = computed(() => {
  const q = search.value.trim()
  if (!q) return []
  const have = new Set(tags.value.map((x) => x.imagery_id))
  return library.value.filter((im) => !have.has(im.id) && (im.name.includes(q) || im.aliases.some((a) => a.includes(q)))).slice(0, 12)
})

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function setLinks(seg: api.MaterialSegmentContext, links: api.MaterialSegmentImagery[]) {
  segment.value = seg
  scene.value = seg.scene_desc
  tags.value = links.map((l) => ({ imagery_id: l.imagery_id, name: l.name, weight: l.weight, source: l.source, confirmed: l.confirmed, status: l.status, visual_zh: l.visual_zh }))
}

async function loadRecs() {
  if (!props.segmentId) return
  recs.value = await api.recommendStyles(props.segmentId)
  if (!recs.value.some((r) => r.style.id === styleId.value)) styleId.value = recs.value[0]?.style.id ?? null
}

async function loadArtworks() {
  if (!props.segmentId) return
  try {
    artworks.value = (await api.listArtworks({ segment_id: props.segmentId, limit: 60 })).items
  } catch (e) {
    fail(e)
  }
}

async function load() {
  segment.value = null
  prompt.value = null
  if (!props.segmentId) return
  loading.value = true
  try {
    const [data, lib] = await Promise.all([api.getSegmentImagery(props.segmentId), library.value.length ? null : api.listImagery({ limit: 500 })])
    if (lib) library.value = lib.items.filter((im) => im.status !== 'hidden')
    setLinks(data.segment, data.links)
    await Promise.all([loadRecs(), loadArtworks()])
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
}

watch(() => props.segmentId, load, { immediate: true })
watch(() => props.tick, loadArtworks)
watch(styleId, () => {
  picks.composition = picks.palette = picks.mood = ''
  prompt.value = null
})

async function extract() {
  if (!props.segmentId) return
  extracting.value = true
  try {
    const res = await api.extractImagery(props.segmentId)
    setLinks(res.segment, res.links)
    appStore.showSuccess(t('materials.bench.extracted', { n: res.links.length }))
    if (res.created.length) {
      appStore.showSuccess(t('materials.bench.created', { names: res.created.join('、') }))
      library.value = (await api.listImagery({ limit: 500 })).items.filter((im) => im.status !== 'hidden')
    }
    await loadRecs()
  } catch (e) {
    fail(e)
  } finally {
    extracting.value = false
  }
}

function addTag(im: api.MaterialImagery) {
  tags.value.push({ imagery_id: im.id, name: im.name, weight: 2, source: 'manual', confirmed: false, status: im.status, visual_zh: im.visual_zh })
  search.value = ''
}

function removeTag(id: number) {
  tags.value = tags.value.filter((x) => x.imagery_id !== id)
}

async function saveTags() {
  if (!props.segmentId) return
  saving.value = true
  try {
    const res = await api.saveSegmentImagery(
      props.segmentId,
      tags.value.map((x) => ({ imagery_id: x.imagery_id, weight: x.weight })),
      scene.value
    )
    setLinks(res.segment, res.links)
    appStore.showSuccess(t('materials.bench.saved'))
    emit('saved')
    await loadRecs()
    prompt.value = null
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

function input(): api.MaterialPromptInput {
  return { style_id: styleId.value as number, size: size.value, composition: picks.composition, palette: picks.palette, mood: picks.mood, extra: extra.value }
}

async function preview() {
  if (!props.segmentId || !styleId.value) return
  previewing.value = true
  try {
    prompt.value = await api.buildPrompt(props.segmentId, input())
  } catch (e) {
    fail(e)
  } finally {
    previewing.value = false
  }
}

async function draw() {
  if (!props.segmentId || !styleId.value) return
  drawing.value = true
  try {
    const arts = await api.drawSegment(props.segmentId, input(), count.value)
    appStore.showSuccess(t('materials.bench.queued', { n: arts.length }))
    emit('queued')
    await loadArtworks()
  } catch (e) {
    fail(e)
  } finally {
    drawing.value = false
  }
}
</script>
