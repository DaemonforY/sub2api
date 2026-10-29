<template>
  <div class="flex flex-wrap items-center gap-2" data-testid="image-attach">
    <div v-for="(it, i) in props.items" :key="it.key" class="group relative h-16 w-16 overflow-hidden rounded-lg border border-ink-200 bg-ink-100 dark:border-ink-700 dark:bg-ink-800" :title="it.error || it.name">
      <img :src="it.preview" alt="" class="h-full w-full object-cover" :class="{ 'opacity-40': it.status !== 'done' }" />
      <span class="absolute left-1 top-1 rounded bg-black/60 px-1 text-[10px] leading-4 text-white">图{{ props.first + i }}</span>
      <div v-if="it.status === 'uploading'" class="absolute inset-x-1 bottom-1 h-1 overflow-hidden rounded-full bg-white/60">
        <div class="h-full bg-brand-500 transition-all" :style="{ width: `${Math.round(it.progress * 100)}%` }"></div>
      </div>
      <div v-if="it.status === 'error'" class="absolute inset-0 flex items-center justify-center bg-red-500/70 text-[11px] font-medium text-white">失败</div>
      <button class="absolute right-0.5 top-0.5 rounded-full bg-black/60 p-0.5 text-white opacity-80 hover:opacity-100" aria-label="删除图片" @click="remove(i)"><X class="h-3 w-3" /></button>
    </div>
    <button
      v-if="props.items.length < max"
      type="button"
      class="flex h-16 items-center gap-1.5 rounded-lg border border-dashed border-ink-300 px-3 text-xs text-ink-500 hover:border-brand-400 hover:text-brand-600 dark:border-ink-600"
      data-testid="image-attach-add"
      @click="pick"
    >
      <ImagePlus class="h-4 w-4" />
      <span class="text-left leading-4">添加图片<br /><span class="text-ink-400">{{ props.items.length ? `${props.items.length}/${max}` : '产品图、截图、照片' }}</span></span>
    </button>
    <input ref="input" type="file" accept="image/jpeg,image/png,image/webp,image/gif,image/heic,image/heif" multiple class="sr-only" @change="onPick" />
  </div>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { ImagePlus, X } from 'lucide-vue-next'
import { session, signIn } from '../lib/api'
import { isImageFile, stageImage } from '../lib/stageImage'
import { toast, toastError } from '../lib/toast'

// items: the parent's reactive array, changed in place (two pictures added in one go must not
// overwrite each other the way consecutive v-model assignments would).
const props = defineProps({ items: { type: Array, required: true }, max: { type: Number, default: 9 }, first: { type: Number, default: 1 } })
const input = ref(null)
let seq = 0

async function ensureSignedIn() {
  if (session.key) return true
  try {
    return await signIn()
  } catch (err) {
    toastError(err)
    return false
  }
}

async function pick() {
  if (!(await ensureSignedIn())) return
  input.value?.click()
}
function onPick(e) {
  add(Array.from(e.target.files || []))
  e.target.value = ''
}

/** Adds files (from the picker, a drop or a paste); returns how many were images. */
async function add(files) {
  const pics = files.filter(isImageFile)
  if (!pics.length) return 0
  if (!(await ensureSignedIn())) return 0
  const room = props.max - props.items.length
  if (pics.length > room) toast(`最多 ${props.max} 张图片，多出来的没有加上`, 'error')
  for (const f of pics.slice(0, Math.max(0, room))) {
    props.items.push({ key: ++seq, name: f.name, preview: URL.createObjectURL(f), status: 'uploading', progress: 0, token: '', error: '' })
    upload(props.items[props.items.length - 1], f) // the reactive copy
  }
  return pics.length
}

async function upload(it, file) {
  try {
    const res = await stageImage(file, (p) => (it.progress = p))
    Object.assign(it, { status: 'done', token: res.token, progress: 1 })
  } catch (err) {
    Object.assign(it, { status: 'error', error: err.message })
    toast(err.message || '图片上传失败', 'error', 6000)
  }
}

function remove(i) {
  const it = props.items[i]
  if (it) URL.revokeObjectURL(it.preview)
  props.items.splice(i, 1)
}

onBeforeUnmount(() => props.items.forEach((it) => URL.revokeObjectURL(it.preview)))
defineExpose({ add })
</script>
