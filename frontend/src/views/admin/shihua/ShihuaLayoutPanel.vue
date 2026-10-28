<template>
  <div class="grid gap-4 lg:grid-cols-[18rem_1fr]">
    <div class="card h-fit p-3">
      <div class="mb-2 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('shihua.layout.works') }}</h3>
        <input type="checkbox" :checked="allChecked" @change="toggleAll" />
      </div>
      <p v-if="!picked.length" class="py-6 text-center text-sm text-gray-400">{{ t('shihua.layout.noPicked') }}</p>
      <div class="max-h-[70vh] space-y-1 overflow-y-auto">
        <div
          v-for="w in picked"
          :key="w.id"
          class="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm"
          :class="current?.id === w.id ? 'bg-primary-50 dark:bg-primary-900/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
          :data-testid="`shihua-layout-${w.slug}`"
          @click="open(w)"
        >
          <input type="checkbox" :checked="checked.has(w.id)" @click.stop @change="toggle(w.id)" />
          <span class="flex-1 truncate">{{ w.title }}</span>
          <span v-if="w.text_status !== 'verified_2'" class="text-xs text-amber-500" :title="t('shihua.layout.notVerified')">!</span>
          <span v-else-if="w.renders" class="text-xs text-emerald-600">{{ w.renders }}</span>
        </div>
      </div>
      <div class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
        <button class="btn btn-secondary btn-sm w-full" :disabled="!checked.size || batchBusy" data-testid="shihua-batch-layout" @click="batch">
          {{ batchBusy ? t('shihua.layout.rendering', batchProgress) : t('shihua.layout.saveBatch') }}
        </button>
        <div class="text-xs text-gray-500">{{ t('shihua.layout.exportAspects') }}</div>
        <div class="flex flex-wrap gap-x-3 gap-y-1 text-xs">
          <label v-for="a in SHIHUA_ASPECTS" :key="a.id" class="flex items-center gap-1">
            <input v-model="exportAspects" type="checkbox" :value="a.id" /> {{ a.id.replace('x', ':') }}
          </label>
        </div>
        <button class="btn btn-primary btn-sm w-full" :disabled="!checked.size || exporting" data-testid="shihua-export" @click="exportZip">
          {{ exporting ? (batchProgress.done < batchProgress.total ? t('shihua.layout.rendering', batchProgress) : t('shihua.layout.exporting')) : t('shihua.layout.export') }}
        </button>
        <p class="text-xs text-gray-400">{{ t('shihua.layout.exportHint') }}</p>
      </div>
    </div>

    <div class="space-y-4">
      <p v-if="fontMissing" class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">{{ t('shihua.layout.fontMissing') }}</p>
      <div v-if="current" class="card p-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="mr-auto">
            <div class="font-semibold text-gray-900 dark:text-white">{{ current.title }} <span class="text-sm font-normal text-gray-500">{{ current.author }}</span></div>
            <div class="text-xs text-gray-500">{{ image?.upscaled_url ? t('shihua.layout.useUpscaled') : t('shihua.layout.noUpscaled') }}</div>
            <div v-if="current.text_status !== 'verified_2'" class="text-xs text-amber-600">{{ t('shihua.layout.notVerified') }}</div>
          </div>
          <label class="text-xs text-gray-500"
            >{{ t('shihua.layout.font') }}
            <select v-model="font" class="input mt-1 w-32" data-testid="shihua-font" @change="changeFont">
              <option v-for="f in SHIHUA_FONTS" :key="f.id" :value="f.id">{{ t(`shihua.layout.fonts.${f.id}`) }}</option>
            </select>
          </label>
          <label v-if="image && !image.title_drawn" class="text-xs text-gray-500"
            >{{ t('shihua.layout.titleFont') }}
            <select v-model="titleFont" class="input mt-1 w-32" data-testid="shihua-title-font" @change="changeFont">
              <option value="">{{ t('shihua.layout.sameFont') }}</option>
              <option v-for="f in SHIHUA_FONTS" :key="f.id" :value="f.id">{{ t(`shihua.layout.fonts.${f.id}`) }}</option>
            </select>
          </label>
          <label class="text-xs text-gray-500">{{ t('shihua.layout.seal') }}<input v-model="seal" class="input mt-1 w-28" maxlength="4" /></label>
          <label class="text-xs text-gray-500">{{ t('shihua.layout.label') }}<input v-model="label" class="input mt-1 w-32" /></label>
          <button class="btn btn-secondary" :disabled="!art" @click="redetect">{{ t('shihua.layout.redetect') }}</button>
          <button class="btn btn-primary" :disabled="!art || saving || current.text_status !== 'verified_2'" data-testid="shihua-save-all" @click="saveAll">{{ t('shihua.layout.saveAll') }}</button>
        </div>
        <p class="mt-2 text-xs text-gray-400">{{ t('shihua.layout.dragHint') }}<template v-if="image?.title_drawn"> · {{ t('shihua.layout.titleDrawn') }}</template></p>
        <p v-if="!art" class="py-10 text-center text-sm text-gray-400">{{ t('shihua.layout.loadingArt') }}</p>
        <div v-else class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div v-for="a in SHIHUA_ASPECTS" :key="a.id" class="space-y-2">
            <div class="flex items-center justify-between text-xs">
              <span class="font-medium text-gray-700 dark:text-gray-200">{{ t(`shihua.layout.aspects.${a.id}`) }}</span>
              <span v-if="savedAspects.has(a.id)" class="text-emerald-600">✓</span>
            </div>
            <div
              class="relative select-none overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-800"
              @mousedown="a.id === '2x3' && startDrag($event)"
              @mousemove="a.id === '2x3' && moveDrag($event)"
              @mouseup="a.id === '2x3' && endDrag()"
              @mouseleave="a.id === '2x3' && endDrag()"
            >
              <img v-if="previews[a.id]" :src="previews[a.id]!.url" class="block w-full" draggable="false" :data-testid="`shihua-preview-${a.id}`" />
              <div
                v-if="a.id === '2x3' && dragBox"
                class="pointer-events-none absolute border-2 border-dashed border-primary-500 bg-primary-500/10"
                :style="{ left: `${dragBox.x0 * 100}%`, top: `${dragBox.y0 * 100}%`, width: `${(dragBox.x1 - dragBox.x0) * 100}%`, height: `${(dragBox.y1 - dragBox.y0) * 100}%` }"
              />
            </div>
            <div class="flex items-center gap-2 text-xs text-gray-500">
              {{ t('shihua.layout.scale') }}
              <input v-model.number="options[a.id].scale" type="range" min="0.6" max="1.2" step="0.05" class="flex-1" @change="renderOne(a.id)" />
              <select v-model="options[a.id].color" class="input w-20 py-0.5 text-xs" @change="renderOne(a.id)">
                <option v-for="c in colors" :key="c" :value="c">{{ t(`shihua.layout.colors.${c}`) }}</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/shihua'
import {
  SHIHUA_ASPECTS,
  SHIHUA_DEFAULT_FONT,
  SHIHUA_FONTS,
  canvasToBlob,
  loadShihuaArt,
  loadShihuaFont,
  renderShihuaPoster,
  textForAspect,
  type ShihuaAspect,
  type ShihuaBox,
  type ShihuaFontId,
  type ShihuaLayoutResult
} from '@/utils/shihuaLayout'

const props = defineProps<{ works: api.ShihuaWork[] }>()
const emit = defineEmits<{ (e: 'reload'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const colors = ['auto', 'ink', 'light'] as const

const picked = computed(() => props.works.filter((w) => w.picked_image_id))
const checked = ref(new Set<number>())
const allChecked = computed(() => picked.value.length > 0 && picked.value.every((w) => checked.value.has(w.id)))
function toggle(id: number) {
  const s = new Set(checked.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  checked.value = s
}
function toggleAll() {
  checked.value = allChecked.value ? new Set() : new Set(picked.value.map((w) => w.id))
}

const seal = ref('课本诗画')
const label = ref('AI 生成')
const font = ref<ShihuaFontId>(SHIHUA_DEFAULT_FONT)
const titleFont = ref<ShihuaFontId | ''>('')
const fontMissing = ref(false)
const current = ref<api.ShihuaWork | null>(null)
const image = ref<api.ShihuaImage | null>(null)
const art = ref<HTMLCanvasElement | null>(null)
const saving = ref(false)
const savedAspects = ref(new Set<string>())
type AspectOptions = { box?: ShihuaBox; scale: number; color: 'auto' | 'ink' | 'light' }
const freshOptions = () => Object.fromEntries(SHIHUA_ASPECTS.map((a) => [a.id, { scale: 1, color: 'auto' }])) as Record<ShihuaAspect, AspectOptions>
const options = reactive(freshOptions())
const previews = reactive<Partial<Record<ShihuaAspect, { url: string; canvas: HTMLCanvasElement; layout: ShihuaLayoutResult }>>>({})

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function posterText(w: api.ShihuaWork) {
  return w.display_text.trim() || w.full_text
}

/** The author as set beside the title: 〔清〕高鼎. */
function authorLine(w: api.ShihuaWork) {
  return w.author ? `${w.dynasty ? `〔${w.dynasty}〕` : ''}${w.author}` : ''
}

async function pickedImage(w: api.ShihuaWork): Promise<api.ShihuaImage | null> {
  const list = await api.listImages({ work_ids: String(w.id) })
  return list.find((i) => i.id === w.picked_image_id) ?? null
}

async function render(w: api.ShihuaWork, img: api.ShihuaImage, source: HTMLCanvasElement, aspect: ShihuaAspect, opt: AspectOptions) {
  // Pictures drawn without text get the title and author from the layout.
  const withTitle = !img.title_drawn
  return renderShihuaPoster(
    { art: source, aspect, text: textForAspect(aspect, posterText(w), w.famous_lines), title: w.title, author: w.author, withTitle, byline: withTitle ? authorLine(w) : '' },
    { box: opt.box, scale: opt.scale, color: opt.color, seal: seal.value, label: label.value, font: font.value, titleFont: titleFont.value || undefined }
  )
}

async function renderOne(aspect: ShihuaAspect) {
  const w = current.value
  if (!w || !art.value || !image.value) return
  const result = await render(w, image.value, art.value, aspect, options[aspect])
  const blob = await canvasToBlob(result.canvas, 0.8)
  const old = previews[aspect]
  if (old) URL.revokeObjectURL(old.url)
  previews[aspect] = { url: URL.createObjectURL(blob), canvas: result.canvas, layout: result.layout }
}

async function renderAll() {
  for (const a of SHIHUA_ASPECTS) await renderOne(a.id)
}

async function changeFont() {
  const loads = [loadShihuaFont(font.value)]
  if (titleFont.value) loads.push(loadShihuaFont(titleFont.value))
  fontMissing.value = (await Promise.all(loads)).includes(false)
  if (art.value) await renderAll()
}

/** Forgets dragged text areas and looks for the blank area again. */
function redetect() {
  for (const a of SHIHUA_ASPECTS) options[a.id].box = undefined
  void renderAll()
}

async function open(w: api.ShihuaWork) {
  current.value = w
  art.value = null
  Object.assign(options, freshOptions())
  dragBox.value = null
  try {
    image.value = await pickedImage(w)
    if (!image.value?.url) return
    const [source, renders] = await Promise.all([loadShihuaArt(image.value.url, image.value.upscaled_url), api.listRenders([w.id])])
    const mine = renders.filter((r) => r.image_id === w.picked_image_id)
    savedAspects.value = new Set(mine.map((r) => r.aspect))
    // Reopening a work keeps the font it was saved with.
    const savedFont = mine.map((r) => r.layout?.font).find((f): f is ShihuaFontId => SHIHUA_FONTS.some((x) => x.id === f))
    if (savedFont) font.value = savedFont
    const savedTitleFont = mine.map((r) => r.layout?.titleFont).find((f): f is ShihuaFontId => SHIHUA_FONTS.some((x) => x.id === f))
    titleFont.value = savedTitleFont && savedTitleFont !== font.value ? savedTitleFont : ''
    fontMissing.value = !(await loadShihuaFont(font.value))
    if (titleFont.value) await loadShihuaFont(titleFont.value)
    if (current.value?.id !== w.id) return
    art.value = source
    await renderAll()
  } catch (e) {
    fail(e)
  }
}

async function saveAll() {
  const w = current.value
  if (!w || !image.value) return
  saving.value = true
  try {
    let n = 0
    for (const a of SHIHUA_ASPECTS) {
      const p = previews[a.id]
      if (!p) continue
      await api.saveRender(image.value.id, a.id, await canvasToBlob(p.canvas), { ...p.layout, seal: seal.value, label: label.value })
      n++
    }
    savedAspects.value = new Set(SHIHUA_ASPECTS.map((a) => a.id))
    appStore.showSuccess(t('shihua.layout.saved', { n }))
    emit('reload')
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

// ---- batch: lay out checked works automatically and save the posters ----
const batchBusy = ref(false)
const batchProgress = reactive({ done: 0, total: 0 })

/**
 * Lays out and saves `aspects` of the checked works (only the ones without a saved poster when
 * onlyMissing). Works whose text is not checked twice are skipped (the server refuses their
 * posters); their titles are returned.
 */
async function layoutChecked(aspects: ShihuaAspect[], onlyMissing: boolean): Promise<{ saved: number; unverified: string[] }> {
  const works = picked.value.filter((w) => checked.value.has(w.id))
  const unverified = works.filter((w) => w.text_status !== 'verified_2').map((w) => w.title)
  const todo = works.filter((w) => w.text_status === 'verified_2')
  const have = new Set<string>()
  if (onlyMissing && todo.length) {
    for (const r of await api.listRenders(todo.map((w) => w.id))) have.add(`${r.image_id}:${r.aspect}`)
  }
  const jobs = todo.map((w) => ({ w, aspects: aspects.filter((a) => !have.has(`${w.picked_image_id}:${a}`)) })).filter((j) => j.aspects.length)
  Object.assign(batchProgress, { done: 0, total: jobs.reduce((n, j) => n + j.aspects.length, 0) })
  let saved = 0
  if (!jobs.length) return { saved, unverified }
  fontMissing.value = !(await loadShihuaFont(font.value))
  if (titleFont.value) await loadShihuaFont(titleFont.value)
  for (const { w, aspects: list } of jobs) {
    const img = await pickedImage(w)
    if (!img?.url) continue
    const source = await loadShihuaArt(img.url, img.upscaled_url)
    for (const a of list) {
      const result = await render(w, img, source, a, { scale: 1, color: 'auto' })
      await api.saveRender(img.id, a, await canvasToBlob(result.canvas), { ...result.layout, seal: seal.value, label: label.value })
      batchProgress.done++
      saved++
    }
  }
  return { saved, unverified }
}

function warnUnverified(titles: string[]) {
  if (titles.length) appStore.showError(t('shihua.layout.skippedUnverified', { titles: titles.map((x) => `《${x}》`).join('、') }))
}

async function batch() {
  batchBusy.value = true
  try {
    const { saved, unverified } = await layoutChecked(SHIHUA_ASPECTS.map((a) => a.id), false)
    if (saved) appStore.showSuccess(t('shihua.layout.saved', { n: saved }))
    warnUnverified(unverified)
    emit('reload')
  } catch (e) {
    fail(e)
  } finally {
    batchBusy.value = false
  }
}

// ---- export: posters not saved yet are laid out automatically first ----
const exportAspects = ref<string[]>(['2x3', '3x4', '9x16'])
const exporting = ref(false)
async function exportZip() {
  exporting.value = true
  batchBusy.value = true
  try {
    const { saved, unverified } = await layoutChecked(exportAspects.value as ShihuaAspect[], true)
    if (saved) emit('reload')
    const blob = await api.exportZip([...checked.value], exportAspects.value)
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `shihua-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    setTimeout(() => URL.revokeObjectURL(a.href), 10000)
    warnUnverified(unverified)
  } catch (e) {
    fail(e)
  } finally {
    exporting.value = false
    batchBusy.value = false
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

watch(
  () => [seal.value, label.value],
  () => {
    if (art.value) void renderAll()
  }
)

onBeforeUnmount(() => {
  for (const p of Object.values(previews)) if (p) URL.revokeObjectURL(p.url)
})
</script>
