<template>
  <div class="space-y-4">
    <div class="card flex flex-wrap items-center gap-2 p-3">
      <button
        v-for="f in modes"
        :key="f"
        class="rounded-lg px-3 py-1 text-sm"
        :class="mode === f ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
        @click="mode = f"
      >
        {{ t(`shihua.pick.filter.${f}`) }}
      </button>
      <div class="ml-auto flex items-center gap-2">
        <div class="flex rounded-lg border border-gray-200 p-0.5 dark:border-dark-600">
          <button
            v-for="v in views"
            :key="v"
            class="rounded-md px-2.5 py-0.5 text-sm"
            :class="view === v ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-600 dark:text-gray-300'"
            :data-testid="`shihua-pick-view-${v}`"
            @click="view = v"
          >
            {{ t(`shihua.pick.view.${v}`) }}
          </button>
        </div>
        <select v-if="view === 'poster'" v-model="previewFont" class="input w-36 py-1 text-sm" :title="t('shihua.pick.previewFont')" data-testid="shihua-pick-font">
          <option v-for="f in SHIHUA_FONTS" :key="f.id" :value="f.id">{{ t(`shihua.layout.fonts.${f.id}`) }}</option>
        </select>
        <button class="btn btn-secondary btn-sm" @click="reload"><Icon name="refresh" size="sm" /></button>
      </div>
    </div>
    <p v-if="view === 'poster'" class="-mt-2 text-xs text-gray-400">{{ t('shihua.pick.previewHint') }}</p>

    <div v-if="!rows.length" class="card p-10 text-center text-gray-400">{{ t('shihua.pick.empty') }}</div>

    <div v-for="row in rows" :key="row.work.id" class="card p-4" :data-testid="`shihua-pick-${row.work.slug}`">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="font-semibold text-gray-900 dark:text-white">
            {{ row.work.title }} <span class="text-sm font-normal text-gray-500">{{ row.work.author }} · {{ row.work.grade }}</span>
          </div>
          <p class="mt-1 line-clamp-2 max-w-3xl text-xs text-gray-500">{{ row.work.theme.prompt_theme }}</p>
          <p v-if="row.waiting" class="mt-1 text-xs text-sky-600">{{ t('shihua.pick.waiting', { n: row.waiting }) }}</p>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary btn-sm" @click="redraw(row.work, 2)">{{ t('shihua.pick.redraw') }} ×2</button>
          <button class="btn btn-secondary btn-sm" @click="openRedraw(row.work)">{{ t('shihua.pick.redrawWith') }}</button>
        </div>
      </div>
      <div class="mt-3 grid gap-3" :class="view === 'poster' ? 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4' : 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 xl:grid-cols-6'">
        <div
          v-for="img in row.images"
          :key="img.id"
          class="relative overflow-hidden rounded-lg border"
          :class="[
            row.work.picked_image_id === img.id ? 'border-emerald-500 ring-2 ring-emerald-500' : 'border-gray-200 dark:border-dark-600',
            img.rejected ? 'opacity-40' : ''
          ]"
        >
          <a v-if="img.status === 'succeeded'" :href="img.url" target="_blank" rel="noopener" :title="t('shihua.pick.open')">
            <img
              :src="view === 'poster' ? posters[posterKey(img)] || img.thumb_url : img.thumb_url"
              class="aspect-[2/3] w-full bg-gray-100 object-cover"
              :class="view === 'poster' && !posters[posterKey(img)] ? 'opacity-60' : ''"
              loading="lazy"
              :data-testid="`shihua-pick-thumb-${img.id}`"
            />
          </a>
          <div v-else class="flex aspect-[2/3] w-full items-center justify-center bg-gray-50 p-2 text-center text-xs text-gray-500 dark:bg-dark-800">
            <span v-if="img.status === 'failed'" class="text-red-500">{{ img.error }}</span>
            <span v-else>{{ img.status }}<template v-if="img.error"> · {{ img.error }}</template></span>
          </div>
          <div v-if="img.status === 'succeeded'" class="space-y-1 p-2 text-xs">
            <div class="flex flex-wrap gap-x-2 gap-y-1">
              <button v-if="row.work.picked_image_id !== img.id" class="font-medium text-emerald-600 hover:underline" :data-testid="`shihua-pick-img-${img.id}`" @click="review(img, 'pick')">{{ t('shihua.pick.pick') }}</button>
              <button v-else class="text-gray-500 hover:underline" @click="review(img, 'unpick')">{{ t('shihua.pick.unpick') }}</button>
              <button class="hover:underline" :class="img.title_check === 'wrong' ? 'text-red-600' : 'text-gray-500'" @click="review(img, 'title', img.title_check === 'wrong' ? '' : 'wrong')">
                {{ t('shihua.pick.titleWrong') }}
              </button>
              <button class="text-gray-500 hover:underline" @click="review(img, img.rejected ? 'restore' : 'reject')">{{ img.rejected ? t('shihua.pick.restore') : t('shihua.pick.reject') }}</button>
            </div>
            <div class="flex items-center justify-between text-gray-400">
              <span>{{ (img.duration_ms / 1000).toFixed(0) }}s · ${{ img.cost_usd.toFixed(3) }}</span>
              <span v-if="img.upscale_status" :class="img.upscale_status === 'failed' ? 'text-red-500' : ''" :title="img.upscale_error">{{ t(`shihua.pick.upscaleStatus.${img.upscale_status}`) }}</span>
              <button v-else-if="row.work.picked_image_id === img.id" class="text-primary-600 hover:underline" @click="review(img, 'upscale')">{{ t('shihua.pick.upscale') }}</button>
            </div>
            <button v-if="img.upscale_status === 'failed'" class="text-primary-600 hover:underline" @click="review(img, 'upscale')">{{ t('shihua.pick.upscale') }}</button>
          </div>
        </div>
      </div>
    </div>

    <BaseDialog :show="!!redrawWork" :title="t('shihua.pick.redrawWith')" width="wide" @close="redrawWork = null">
      <textarea v-model="redrawTheme" rows="6" class="input" />
      <template #footer>
        <button class="btn btn-secondary" @click="redrawWork = null">{{ t('shihua.edit.cancel') }}</button>
        <button class="btn btn-primary" @click="confirmRedraw">{{ t('shihua.pick.redraw') }} ×2</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/shihua'
import { SHIHUA_DEFAULT_FONT, SHIHUA_FONTS, canvasToBlob, loadShihuaArt, renderShihuaPoster, type ShihuaFontId } from '@/utils/shihuaLayout'

const props = defineProps<{ works: api.ShihuaWork[] }>()
const emit = defineEmits<{ (e: 'reload'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const modes = ['todo', 'picked', 'all'] as const
const mode = ref<(typeof modes)[number]>('todo')
const images = ref<api.ShihuaImage[]>([])
let timer: ReturnType<typeof setInterval> | undefined

const rows = computed(() => {
  const byWork = new Map<number, api.ShihuaImage[]>()
  for (const img of images.value) {
    if (img.status === 'canceled') continue
    const list = byWork.get(img.work_id) ?? []
    list.push(img)
    byWork.set(img.work_id, list)
  }
  return props.works
    .filter((w) => byWork.has(w.id))
    .filter((w) => mode.value === 'all' || (mode.value === 'picked' ? !!w.picked_image_id : !w.picked_image_id))
    .map((work) => {
      const list = byWork.get(work.id)!
      return { work, images: list, waiting: list.filter((i) => i.status === 'queued' || i.status === 'running').length }
    })
})

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function reload() {
  try {
    images.value = await api.listImages()
  } catch (e) {
    fail(e)
  }
}

async function review(img: api.ShihuaImage, action: 'pick' | 'unpick' | 'reject' | 'restore' | 'title' | 'upscale', titleCheck = '') {
  try {
    const updated = await api.reviewImage(img.id, action, titleCheck)
    images.value = images.value.map((i) => (i.id === updated.id ? updated : i))
    if (action === 'pick' || action === 'unpick' || action === 'reject') emit('reload')
  } catch (e) {
    fail(e)
  }
}

async function redraw(w: api.ShihuaWork, count: number, theme?: string) {
  try {
    const queued = await api.redraw(w.id, count, theme)
    appStore.showSuccess(t('shihua.pick.redrawQueued', { n: queued.length }))
    await reload()
    emit('reload')
  } catch (e) {
    fail(e)
  }
}

const redrawWork = ref<api.ShihuaWork | null>(null)
const redrawTheme = ref('')
function openRedraw(w: api.ShihuaWork) {
  redrawWork.value = w
  redrawTheme.value = w.theme.prompt_theme
}
async function confirmRedraw() {
  const w = redrawWork.value
  if (!w) return
  redrawWork.value = null
  await redraw(w, 2, redrawTheme.value)
}

// ---- 配诗预览: each drawing with the poem laid out automatically (2:3, from the thumbnail) ----
const views = ['art', 'poster'] as const
const view = ref<(typeof views)[number]>('art')
const previewFont = ref<ShihuaFontId>(SHIHUA_DEFAULT_FONT)
const posters = reactive<Record<string, string>>({})
const failedPosters = new Set<string>()
let rendering = false

function posterKey(img: api.ShihuaImage) {
  return `${img.id}:${previewFont.value}`
}

function authorLine(w: api.ShihuaWork) {
  return w.author ? `${w.dynasty ? `〔${w.dynasty}〕` : ''}${w.author}` : ''
}

/** Renders the missing previews one at a time (each is a few hundred ms of canvas work). */
async function renderPosters() {
  if (rendering || view.value !== 'poster') return
  rendering = true
  try {
    for (;;) {
      const todo = rows.value
        .flatMap((r) => r.images.map((img) => ({ work: r.work, img })))
        .find(({ img }) => img.status === 'succeeded' && img.thumb_url && !posters[posterKey(img)] && !failedPosters.has(posterKey(img)))
      if (!todo || view.value !== 'poster') break
      const { work, img } = todo
      const key = posterKey(img)
      try {
        const art = await loadShihuaArt(img.thumb_url!)
        const { canvas } = await renderShihuaPoster(
          {
            art,
            aspect: '2x3',
            text: work.display_text.trim() || work.full_text,
            title: work.title,
            author: work.author,
            withTitle: !img.title_drawn,
            byline: img.title_drawn ? '' : authorLine(work)
          },
          { font: previewFont.value }
        )
        posters[key] = URL.createObjectURL(await canvasToBlob(canvas, 0.85))
      } catch {
        failedPosters.add(key)
      }
    }
  } finally {
    rendering = false
  }
}

watch([view, previewFont, rows], () => void renderPosters())

onMounted(() => {
  void reload()
  timer = setInterval(() => {
    if (!document.hidden && images.value.some((i) => i.status === 'queued' || i.status === 'running' || i.upscale_status === 'queued' || i.upscale_status === 'running')) void reload()
  }, 8000)
})
onBeforeUnmount(() => {
  clearInterval(timer)
  for (const url of Object.values(posters)) URL.revokeObjectURL(url)
})
</script>
