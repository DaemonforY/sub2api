<template>
  <div class="mx-auto max-w-[1440px] gap-6 px-4 pb-12 pt-8 sm:px-6 xl:flex">
    <HistorySidebar class="hidden w-60 shrink-0 xl:block" />

    <div class="mx-auto min-w-0 max-w-4xl flex-1">
      <h1 class="text-2xl font-bold">上传作品</h1>
      <p class="mt-1 text-sm leading-6 text-ink-500">把你在本地做好的视频或 SVG 动画放进「历史创作」，可以分享，也可以发布到案例库（审核通过后公开）。</p>

      <div v-if="!session.key" class="card mt-6 p-10 text-center">
        <p class="text-ink-500">登录后才能上传作品</p>
        <button class="btn-primary mt-4" @click="doSignIn"><LogIn class="h-4 w-4" />登录 HiveGPT 账号</button>
      </div>

      <template v-else>
        <!-- drop zone -->
        <label
          v-if="!file"
          class="mt-6 flex cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed px-6 py-16 text-center transition"
          :class="dragging ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20' : 'border-ink-300 bg-white hover:border-brand-400 dark:border-ink-700 dark:bg-ink-900'"
          data-testid="upload-drop"
          @dragover.prevent="dragging = true"
          @dragleave.prevent="dragging = false"
          @drop.prevent="onDrop"
        >
          <UploadCloud class="h-10 w-10 text-brand-500" />
          <span class="mt-3 text-base font-medium">把文件拖到这里，或点击选择</span>
          <span class="mt-2 text-sm text-ink-500">MP4 / WebM 视频：不超过 100 MB、3 分钟 · SVG 动画：不超过 2 MB</span>
          <input ref="input" type="file" class="sr-only" accept=".mp4,.m4v,.webm,.svg,video/mp4,video/webm,image/svg+xml" data-testid="upload-input" @change="onPick" />
        </label>

        <div v-else class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
          <!-- preview -->
          <div>
            <PosterPicker v-if="kind === 'video'" :src="localUrl" @meta="onMeta" @poster="onPoster" @error="(m) => (problem = m)" />
            <div v-else class="relative overflow-hidden rounded-xl bg-white" :style="{ aspectRatio: '16 / 9' }">
              <img :src="localUrl" alt="" class="absolute inset-0 h-full w-full object-contain" @error="problem = 'SVG 无法显示，请确认是标准的 SVG 文件（Invalid SVG）'" />
            </div>
            <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-500">
              <span class="truncate font-medium text-ink-700 dark:text-ink-200">{{ file.name }}</span>
              <span>{{ fmtBytes(file.size) }}</span>
              <span v-if="meta.duration">{{ fmtTime(meta.duration) }}</span>
              <span v-if="meta.width">{{ meta.width }}×{{ meta.height }}</span>
              <button class="text-brand-600 hover:underline" :disabled="uploading" @click="reset">换一个文件</button>
            </div>
            <p v-if="kind === 'svg'" class="mt-2 text-xs leading-5 text-ink-500">SVG 里的脚本、外部链接和嵌入网页会被移除，SMIL 和 CSS 动画会保留。</p>
          </div>

          <!-- details -->
          <div class="space-y-4">
            <div v-if="kind === 'video' && posterUrl">
              <span class="block text-sm font-medium">封面</span>
              <img :src="posterUrl" alt="封面" class="mt-1 w-full rounded-lg border border-ink-200 dark:border-ink-700" />
            </div>
            <label class="block">
              <span class="text-sm font-medium">标题</span>
              <input v-model="title" maxlength="60" class="input mt-1" placeholder="给作品起个名字" data-testid="upload-title" />
            </label>
            <label class="block">
              <span class="text-sm font-medium">作品介绍<span class="font-normal text-ink-400">（选填）</span></span>
              <textarea v-model="description" rows="4" maxlength="1000" class="input mt-1 resize-none" placeholder="讲讲它是什么、用什么做的"></textarea>
            </label>
            <label class="block">
              <span class="text-sm font-medium">分类</span>
              <select v-model="category" class="input mt-1">
                <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
            </label>

            <p v-if="problem" class="rounded-xl bg-red-50 p-3 text-sm leading-6 text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="upload-error">{{ problem }}</p>

            <div v-if="uploading" class="space-y-1.5">
              <div class="h-2 overflow-hidden rounded-full bg-ink-100 dark:bg-ink-800">
                <div class="h-full rounded-full bg-gradient-to-r from-brand-500 to-brand-600 transition-all" :style="{ width: `${Math.round(progress * 100)}%` }"></div>
              </div>
              <p class="flex justify-between text-xs text-ink-500">
                <span>{{ progress < 1 ? `正在上传 ${fmtBytes(file.size * progress)} / ${fmtBytes(file.size)}` : '服务器正在检查文件…' }}</span>
                <span class="tabular-nums">{{ Math.round(progress * 100) }}%</span>
              </p>
            </div>

            <div class="flex gap-2">
              <button class="btn-primary flex-1 py-2.5" :disabled="uploading || !!problem || !title.trim() || (kind === 'video' && !meta.width)" data-testid="upload-submit" @click="submit">
                <Loader2 v-if="uploading" class="h-4 w-4 animate-spin" />
                <UploadCloud v-else class="h-4 w-4" />
                {{ uploading ? '上传中…' : '上传' }}
              </button>
              <button v-if="uploading" class="btn-ghost" @click="cancel">取消</button>
            </div>
            <p class="text-xs leading-5 text-ink-400">每个账号 24 小时内最多上传 {{ UPLOAD_LIMITS.perDay }} 个作品，上传作品共用 2 GB 空间。请勿上传侵权、违法或含个人隐私的内容。</p>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Loader2, LogIn, UploadCloud } from 'lucide-vue-next'
import HistorySidebar from '../components/HistorySidebar.vue'
import PosterPicker from '../components/PosterPicker.vue'
import { apiUpload, catalog, session, signIn } from '../lib/api'
import { UPLOAD_LIMITS, fmtBytes, fmtTime, uploadKind } from '../lib/media'
import { toast, toastError } from '../lib/toast'

const router = useRouter()
const input = ref(null)
const dragging = ref(false)
const file = ref(null)
const kind = ref('')
const localUrl = ref('')
const meta = ref({ duration: 0, width: 0, height: 0 })
const poster = ref(null)
const posterUrl = ref('')
const title = ref('')
const description = ref('')
const category = ref('other')
const categories = ref([])
const problem = ref('')
const uploading = ref(false)
const progress = ref(0)
let abort = null

async function doSignIn() {
  try {
    await signIn()
  } catch (err) {
    toastError(err)
  }
}

function onDrop(e) {
  dragging.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) pick(f)
}
function onPick(e) {
  const f = e.target.files?.[0]
  if (f) pick(f)
}

function pick(f) {
  reset()
  const k = uploadKind(f)
  if (k === 'mov') {
    toast('这是 MOV（QuickTime）文件，请导出或转换为 MP4 后再传（Convert the MOV file to MP4）', 'error', 6000)
    return
  }
  if (!k) {
    toast('只支持 MP4、WebM 视频或 SVG 动画，请换一个文件（Only MP4, WebM or SVG files）', 'error', 6000)
    return
  }
  if (k === 'video' && f.size > UPLOAD_LIMITS.videoBytes) {
    toast(`视频有 ${fmtBytes(f.size)}，不能超过 100 MB，请压缩或剪短后再传（Videos must be 100 MB or smaller）`, 'error', 6000)
    return
  }
  if (k === 'svg' && f.size > UPLOAD_LIMITS.svgBytes) {
    toast(`SVG 有 ${fmtBytes(f.size)}，不能超过 2 MB，请删掉不需要的图层或压缩路径后再传（SVG files must be 2 MB or smaller）`, 'error', 6000)
    return
  }
  file.value = f
  kind.value = k
  localUrl.value = URL.createObjectURL(f)
  title.value = f.name.replace(/\.[^.]+$/, '').slice(0, 60)
  category.value = k === 'video' ? 'film' : 'other'
}

function onMeta(m) {
  meta.value = m
  if (m.duration > UPLOAD_LIMITS.seconds + 0.5) problem.value = `视频有 ${fmtTime(m.duration)}，不能超过 3 分钟，请剪短后再传（Videos must be 3 minutes or shorter）`
}
function onPoster({ blob }) {
  poster.value = blob
  if (posterUrl.value) URL.revokeObjectURL(posterUrl.value)
  posterUrl.value = URL.createObjectURL(blob)
}

function reset() {
  if (localUrl.value) URL.revokeObjectURL(localUrl.value)
  if (posterUrl.value) URL.revokeObjectURL(posterUrl.value)
  file.value = null
  kind.value = ''
  localUrl.value = ''
  posterUrl.value = ''
  poster.value = null
  meta.value = { duration: 0, width: 0, height: 0 }
  problem.value = ''
  if (input.value) input.value.value = ''
}

async function submit() {
  if (!file.value || uploading.value) return
  const form = new FormData()
  form.append('title', title.value.trim())
  form.append('description', description.value.trim())
  form.append('category', category.value)
  if (kind.value === 'video' && poster.value && poster.value.size <= UPLOAD_LIMITS.posterBytes) form.append('poster', poster.value, 'poster.jpg')
  form.append('file', file.value, file.value.name)
  uploading.value = true
  progress.value = 0
  problem.value = ''
  const ctrl = new AbortController()
  abort = ctrl
  try {
    const p = await apiUpload('/uploads', form, { signal: ctrl.signal, onProgress: (x) => (progress.value = x) })
    toast('上传成功', 'success')
    router.push(`/p/${p.id}`)
  } catch (err) {
    if (err?.name !== 'AbortError') problem.value = err.message
  } finally {
    uploading.value = false
    abort = null
  }
}
function cancel() {
  abort?.abort()
}

onMounted(() => catalog().then((c) => (categories.value = c.categories)).catch(toastError))
onBeforeUnmount(() => {
  abort?.abort()
  reset()
})
</script>
