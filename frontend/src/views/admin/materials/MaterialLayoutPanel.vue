<template>
  <div class="grid gap-4 lg:grid-cols-[18rem_1fr]">
    <div class="card h-fit p-3">
      <div class="mb-2 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.layout.works') }}</h3>
        <input type="checkbox" :checked="allChecked" @change="toggleAll" />
      </div>
      <p v-if="!approved.length" class="py-6 text-center text-sm text-gray-400">{{ t('materials.layout.empty') }}</p>
      <div class="max-h-[70vh] space-y-1 overflow-y-auto">
        <div
          v-for="a in approved"
          :key="a.id"
          class="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm"
          :class="current?.id === a.id ? 'bg-primary-50 dark:bg-primary-900/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
          :data-testid="`materials-layout-${a.id}`"
          @click="open(a)"
        >
          <input type="checkbox" :checked="checked.has(a.id)" @click.stop @change="toggle(a.id)" />
          <img v-if="a.thumb_url" :src="a.thumb_url" class="h-9 w-6 shrink-0 rounded object-cover" />
          <span class="min-w-0 flex-1 truncate">{{ a.text_title || '—' }} · {{ a.segment_label }}</span>
          <span v-if="renderCounts[a.id]" class="text-xs text-emerald-600">{{ t('materials.layout.savedCount', { n: renderCounts[a.id] }) }}</span>
        </div>
      </div>
      <div class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
        <div class="text-xs text-gray-500">{{ t('materials.layout.exportAspects') }}</div>
        <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs">
          <label v-for="a in SHIHUA_ASPECTS" :key="a.id" class="flex items-center gap-1">
            <input v-model="exportAspects" type="checkbox" :value="a.id" /> {{ a.id.replace('x', ':') }}
          </label>
        </div>
        <button class="btn btn-primary btn-sm w-full" :disabled="!checked.size || exporting" data-testid="materials-export" @click="exportZip">
          {{ exporting ? t('materials.layout.exporting') : t('materials.layout.export') }}
        </button>
      </div>
    </div>

    <div class="space-y-4">
      <p v-if="fontMissing" class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">{{ t('materials.layout.fontMissing') }}</p>
      <div v-if="!current" class="card p-10 text-center text-sm text-gray-400">{{ t('materials.layout.pick') }}</div>
      <div v-else class="card space-y-3 p-4" data-testid="materials-layout-editor">
        <div class="font-semibold text-gray-900 dark:text-white">
          {{ current.text_title || '—' }} · {{ current.segment_label }}
          <span class="text-sm font-normal text-gray-500">{{ current.style_name }}</span>
        </div>

        <div class="grid gap-3 md:grid-cols-[1fr_auto]">
          <label class="block text-xs text-gray-500">
            {{ t('materials.layout.text') }}
            <textarea v-model="text" rows="2" class="input mt-1 text-sm" :disabled="!wordsAllowed" data-testid="materials-layout-text" />
            <span class="text-[11px]" :class="wordsAllowed ? 'text-gray-400' : 'text-amber-600'">
              {{ !segment ? t('materials.layout.noSegment') : wordsAllowed ? t('materials.layout.textHint') : t('materials.layout.refOnly') }}
            </span>
          </label>
          <div class="flex flex-wrap items-end gap-3">
            <label class="flex items-center gap-1 text-xs text-gray-500">
              <input v-model="withByline" type="checkbox" :disabled="!segment" data-testid="materials-layout-byline" /> {{ t('materials.layout.byline') }}：{{ byline || '—' }}
            </label>
          </div>
        </div>

        <div class="flex flex-wrap items-end gap-3">
          <label class="text-xs text-gray-500">
            {{ t('materials.layout.font') }}
            <select v-model="font" class="input mt-1 w-32" @change="changeFont">
              <option v-for="f in SHIHUA_FONTS" :key="f.id" :value="f.id">{{ t(`materials.layout.fonts.${f.id}`) }}</option>
            </select>
          </label>
          <label class="text-xs text-gray-500">{{ t('materials.layout.seal') }}<input v-model="seal" class="input mt-1 w-28" maxlength="4" /></label>
          <label class="text-xs text-gray-500">{{ t('materials.layout.label') }}<input v-model="label" class="input mt-1 w-32" /></label>
          <button class="btn btn-secondary" :disabled="!art" @click="redetect">{{ t('materials.layout.redetect') }}</button>
          <button class="btn btn-primary ml-auto" :disabled="!art || saving" data-testid="materials-layout-save" @click="saveAll">{{ t('materials.layout.saveAll') }}</button>
        </div>
        <p class="text-xs text-gray-400">{{ t('materials.layout.dragHint') }}</p>

        <p v-if="!art" class="py-10 text-center text-sm text-gray-400">{{ t('materials.layout.loadingArt') }}</p>
        <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div v-for="a in SHIHUA_ASPECTS" :key="a.id" class="space-y-2">
            <div class="flex items-center justify-between text-xs">
              <span class="font-medium text-gray-700 dark:text-gray-200">{{ t(`materials.layout.aspects.${a.id}`) }}</span>
              <span v-if="savedAspects.has(a.id)" class="text-emerald-600">✓</span>
            </div>
            <div
              class="relative select-none overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-800"
              @mousedown="a.id === '2x3' && startDrag($event)"
              @mousemove="a.id === '2x3' && moveDrag($event)"
              @mouseup="a.id === '2x3' && endDrag()"
              @mouseleave="a.id === '2x3' && endDrag()"
            >
              <img v-if="previews[a.id]" :src="previews[a.id]!.url" class="block w-full" draggable="false" :data-testid="`materials-preview-${a.id}`" />
              <div
                v-if="a.id === '2x3' && dragBox"
                class="pointer-events-none absolute border-2 border-dashed border-primary-500 bg-primary-500/10"
                :style="{ left: `${dragBox.x0 * 100}%`, top: `${dragBox.y0 * 100}%`, width: `${(dragBox.x1 - dragBox.x0) * 100}%`, height: `${(dragBox.y1 - dragBox.y0) * 100}%` }"
              />
            </div>
            <div class="flex items-center gap-2 text-xs text-gray-500">
              {{ t('materials.layout.scale') }}
              <input v-model.number="options[a.id].scale" type="range" min="0.6" max="1.2" step="0.05" class="flex-1" @change="renderOne(a.id)" />
              <select v-model="options[a.id].color" class="input w-20 py-0.5 text-xs" @change="renderOne(a.id)">
                <option v-for="c in colors" :key="c" :value="c">{{ t(`materials.layout.colors.${c}`) }}</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'
import {
  SHIHUA_ASPECTS,
  SHIHUA_DEFAULT_FONT,
  SHIHUA_FONTS,
  canvasToBlob,
  loadShihuaArt,
  loadShihuaFont,
  renderShihuaPoster,
  type ShihuaAspect,
  type ShihuaBox,
  type ShihuaFontId,
  type ShihuaLayoutResult
} from '@/utils/shihuaLayout'

const { t } = useI18n()
const appStore = useAppStore()
const colors = ['auto', 'ink', 'light'] as const

const approved = ref<api.MaterialArtwork[]>([])
const renderCounts = ref<Record<number, number>>({})
const checked = ref(new Set<number>())
const allChecked = computed(() => approved.value.length > 0 && approved.value.every((a) => checked.value.has(a.id)))
function toggle(id: number) {
  const s = new Set(checked.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  checked.value = s
}
function toggleAll() {
  checked.value = allChecked.value ? new Set() : new Set(approved.value.map((a) => a.id))
}

const current = ref<api.MaterialArtwork | null>(null)
const segment = ref<api.MaterialSegmentContext | null>(null)
const text = ref('')
const withByline = ref(true)
const seal = ref('诗画')
const label = ref('AI 生成')
const font = ref<ShihuaFontId>(SHIHUA_DEFAULT_FONT)
const fontMissing = ref(false)
const art = ref<HTMLCanvasElement | null>(null)
const saving = ref(false)
const savedAspects = ref(new Set<string>())
type AspectOptions = { box?: ShihuaBox; scale: number; color: 'auto' | 'ink' | 'light' }
const freshOptions = () => Object.fromEntries(SHIHUA_ASPECTS.map((a) => [a.id, { scale: 1, color: 'auto' }])) as Record<ShihuaAspect, AspectOptions>
const options = reactive(freshOptions())
const previews = reactive<Partial<Record<ShihuaAspect, { url: string; canvas: HTMLCanvasElement; layout: ShihuaLayoutResult }>>>({})

// Only texts that may carry their words get any; the server checks the same rule on save.
const wordsAllowed = computed(() => !!segment.value && segment.value.license_status !== 'reference_only')
const byline = computed(() => {
  const s = segment.value
  if (!s) return ''
  return s.title ? `${s.author_name}《${s.title}》` : s.author_name
})

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function loadList() {
  try {
    approved.value = (await api.listArtworks({ review_status: 'approved', status: 'succeeded', limit: 200 })).items
    if (approved.value.length) {
      const renders = await api.listRenders(approved.value.map((a) => a.id))
      const counts: Record<number, number> = {}
      for (const r of renders) counts[r.artwork_id] = (counts[r.artwork_id] ?? 0) + 1
      renderCounts.value = counts
    }
  } catch (e) {
    fail(e)
  }
}

function words() {
  return { text: wordsAllowed.value ? text.value.trim() : '', byline: withByline.value ? byline.value : '' }
}

async function renderOne(aspect: ShihuaAspect) {
  const a = current.value
  if (!a || !art.value) return
  const w = words()
  const opt = options[aspect]
  const result = await renderShihuaPoster(
    { art: art.value, aspect, text: w.text, title: segment.value?.title ?? '', author: segment.value?.author_name ?? '', byline: w.byline },
    { box: opt.box, scale: opt.scale, color: opt.color, seal: seal.value, label: label.value, font: font.value }
  )
  const blob = await canvasToBlob(result.canvas, 0.8)
  const old = previews[aspect]
  if (old) URL.revokeObjectURL(old.url)
  previews[aspect] = { url: URL.createObjectURL(blob), canvas: result.canvas, layout: result.layout }
}

async function renderAll() {
  for (const a of SHIHUA_ASPECTS) await renderOne(a.id)
}

async function changeFont() {
  fontMissing.value = !(await loadShihuaFont(font.value))
  if (art.value) await renderAll()
}

function redetect() {
  for (const a of SHIHUA_ASPECTS) options[a.id].box = undefined
  void renderAll()
}

async function open(a: api.MaterialArtwork) {
  current.value = a
  art.value = null
  segment.value = null
  Object.assign(options, freshOptions())
  dragBox.value = null
  try {
    if (a.segment_id) segment.value = (await api.getSegmentImagery(a.segment_id)).segment
    text.value = wordsAllowed.value ? segment.value?.content ?? '' : ''
    withByline.value = !!segment.value
    if (!a.url) return
    const [source, renders] = await Promise.all([loadShihuaArt(a.url), api.listRenders([a.id])])
    savedAspects.value = new Set(renders.map((r) => r.aspect))
    const savedFont = renders.map((r) => r.layout?.font).find((f): f is ShihuaFontId => SHIHUA_FONTS.some((x) => x.id === f))
    if (savedFont) font.value = savedFont
    fontMissing.value = !(await loadShihuaFont(font.value))
    if (current.value?.id !== a.id) return
    art.value = source
    await renderAll()
  } catch (e) {
    fail(e)
  }
}

async function saveAll() {
  const a = current.value
  if (!a) return
  saving.value = true
  try {
    let n = 0
    for (const asp of SHIHUA_ASPECTS) {
      const p = previews[asp.id]
      if (!p) continue
      await api.saveRender(a.id, asp.id, await canvasToBlob(p.canvas), { ...p.layout, ...words(), seal: seal.value, label: label.value })
      n++
    }
    savedAspects.value = new Set(SHIHUA_ASPECTS.map((x) => x.id))
    renderCounts.value = { ...renderCounts.value, [a.id]: n }
    appStore.showSuccess(t('materials.layout.saved', { n }))
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

const exportAspects = ref<string[]>(['2x3', '3x4', '9x16'])
const exporting = ref(false)
async function exportZip() {
  exporting.value = true
  try {
    const blob = await api.exportZip([...checked.value], exportAspects.value)
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `materials-${new Date().toISOString().slice(0, 10)}.zip`
    link.click()
    setTimeout(() => URL.revokeObjectURL(link.href), 10000)
  } catch (e) {
    fail(e)
  } finally {
    exporting.value = false
  }
}

// ---- drag a text box on the 2:3 preview ----
const dragBox = ref<ShihuaBox | null>(null)
let dragStart: { x: number; y: number; rect: DOMRect } | null = null
function point(ev: MouseEvent, rect: DOMRect) {
  return { x: Math.min(1, Math.max(0, (ev.clientX - rect.left) / rect.width)), y: Math.min(1, Math.max(0, (ev.clientY - rect.top) / rect.height)) }
}
function startDrag(ev: MouseEvent) {
  const rect = (ev.currentTarget as HTMLElement).getBoundingClientRect()
  const p = point(ev, rect)
  dragStart = { ...p, rect }
  dragBox.value = { x0: p.x, y0: p.y, x1: p.x, y1: p.y }
}
function moveDrag(ev: MouseEvent) {
  if (!dragStart) return
  const p = point(ev, dragStart.rect)
  dragBox.value = { x0: Math.min(dragStart.x, p.x), y0: Math.min(dragStart.y, p.y), x1: Math.max(dragStart.x, p.x), y1: Math.max(dragStart.y, p.y) }
}
function endDrag() {
  if (!dragStart) return
  dragStart = null
  const b = dragBox.value
  if (b && b.x1 - b.x0 > 0.05 && b.y1 - b.y0 > 0.05) {
    options['2x3'].box = b
    void renderOne('2x3')
  }
  dragBox.value = null
}

let rerender: ReturnType<typeof setTimeout> | undefined
watch([text, withByline, seal, label], () => {
  clearTimeout(rerender)
  rerender = setTimeout(() => {
    if (art.value) void renderAll()
  }, 400)
})

onMounted(loadList)
onBeforeUnmount(() => {
  clearTimeout(rerender)
  for (const p of Object.values(previews)) if (p) URL.revokeObjectURL(p.url)
})
</script>
