<template>
  <div>
    <div class="relative overflow-hidden rounded-xl bg-black" :style="{ aspectRatio: ratio }">
      <video ref="video" :src="src" class="absolute inset-0 h-full w-full object-contain" muted playsinline preload="auto" @loadedmetadata="onMeta" @seeked="grab" @error="onError"></video>
      <div v-if="!ready && !failed" class="absolute inset-0 flex items-center justify-center text-ink-400"><Loader2 class="h-6 w-6 animate-spin" /></div>
    </div>
    <div v-if="ready" class="mt-3 flex items-center gap-3">
      <span class="w-10 text-right text-xs tabular-nums text-ink-500">{{ fmtTime(time) }}</span>
      <input v-model.number="time" type="range" min="0" :max="duration" step="0.1" class="flex-1 accent-brand-500" aria-label="选择封面画面" @input="seek" />
      <span class="w-10 text-xs tabular-nums text-ink-500">{{ fmtTime(duration) }}</span>
    </div>
    <p v-if="ready" class="mt-1 text-xs text-ink-500">拖动滑块选一帧作为封面（案例库卡片和播放前显示）。</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import { captureFrame, fmtTime } from '../lib/media'

const props = defineProps({ src: { type: String, required: true } })
const emit = defineEmits(['meta', 'poster', 'error'])
const video = ref(null)
const ready = ref(false)
const failed = ref(false)
const duration = ref(0)
const time = ref(0)
const size = ref({ w: 16, h: 9 })
const ratio = computed(() => `${size.value.w} / ${size.value.h}`)

function onMeta() {
  const v = video.value
  duration.value = Number.isFinite(v.duration) ? v.duration : 0
  size.value = { w: v.videoWidth || 16, h: v.videoHeight || 9 }
  emit('meta', { duration: duration.value, width: v.videoWidth, height: v.videoHeight })
  if (!v.videoWidth) {
    failed.value = true
    emit('error', '浏览器无法播放这个视频的编码，请导出为 H.264 编码的 MP4 后再传（Unsupported video codec）')
    return
  }
  ready.value = true
  // Default cover: a frame a little into the video (the very first one is often black).
  time.value = Math.min(1, duration.value / 10)
  v.currentTime = time.value
}
function seek() {
  if (video.value) video.value.currentTime = time.value
}
let seq = 0
async function grab() {
  const n = ++seq
  const blob = await captureFrame(video.value)
  if (blob && n === seq) emit('poster', { blob, time: time.value })
}
function onError() {
  failed.value = true
  emit('error', '视频无法读取，可能已损坏或编码不受支持，请导出为 H.264 编码的 MP4 后再传（Could not read the video）')
}
</script>
