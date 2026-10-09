<template>
  <div ref="box" class="group relative h-full w-full select-none overflow-hidden bg-black" :class="card ? '' : 'rounded-xl'" @mouseenter="hover = true" @mouseleave="hover = false">
    <!-- The scene code runs in a sandboxed frame: scripts only, no same-origin, no network (see player.html CSP). -->
    <iframe
      ref="frame"
      :src="src || '/player.html'"
      sandbox="allow-scripts"
      referrerpolicy="no-referrer"
      title="视频画面"
      class="pointer-events-none absolute inset-0 h-full w-full border-0"
      tabindex="-1"
    ></iframe>
    <div class="absolute inset-0" :class="card ? '' : 'cursor-pointer'" @click="!card && toggle()"></div>

    <div v-if="!ready" class="absolute inset-0 flex items-center justify-center bg-ink-950/70 text-sm text-white/70">
      <span class="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"></span>
    </div>

    <div v-if="subtitlesOn && subtitle && !card" class="pointer-events-none absolute inset-x-0 bottom-[11%] flex justify-center px-[8%]">
      <span class="rounded-lg bg-black/65 px-3 py-1 text-center font-medium leading-snug text-white" :style="{ fontSize: subtitleSize }">{{ subtitle }}</span>
    </div>

    <button v-if="!card && ready && !playing && t < 0.05" class="absolute left-1/2 top-1/2 flex h-16 w-16 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-brand-600 shadow-lg transition hover:scale-105" aria-label="播放" @click.stop="play()">
      <Play class="ml-1 h-7 w-7" fill="currentColor" />
    </button>

    <div v-if="!card" class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent px-3 pb-2 pt-8 text-white opacity-0 transition group-hover:opacity-100" :class="{ 'opacity-100': !playing }">
      <input
        type="range"
        min="0"
        :max="duration || 0"
        step="0.01"
        :value="t"
        class="h-1 w-full cursor-pointer accent-brand-500"
        aria-label="进度"
        @input="seek(+$event.target.value)"
      />
      <div class="mt-1 flex items-center gap-3 text-xs">
        <button class="rounded p-1 hover:bg-white/15" :aria-label="playing ? '暂停' : '播放'" @click.stop="toggle()">
          <Pause v-if="playing" class="h-4 w-4" fill="currentColor" />
          <Play v-else class="h-4 w-4" fill="currentColor" />
        </button>
        <span class="tabular-nums">{{ fmt(t) }} / {{ fmt(duration) }}</span>
        <span v-if="sceneLabel" class="hidden truncate text-white/70 sm:inline">{{ sceneLabel }}</span>
        <span class="flex-1"></span>
        <button v-if="hasNarration" class="rounded px-1.5 py-0.5 hover:bg-white/15" :class="subtitlesOn ? 'text-white' : 'text-white/50'" @click.stop="subtitlesOn = !subtitlesOn">字幕</button>
        <button class="rounded p-1 hover:bg-white/15" aria-label="全屏" @click.stop="fullscreen()"><Maximize class="h-4 w-4" /></button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Maximize, Pause, Play } from 'lucide-vue-next'
import { audioBlob } from '../lib/api'
import { buildSubtitles } from '../lib/subtitles'

const props = defineProps({
  spec: { type: Object, default: null },
  /** (scene) => URL of the scene's narration, or '' when there is none. */
  audioUrl: { type: Function, default: null },
  /** Send the API key with audio requests (own projects). */
  withKey: { type: Boolean, default: false },
  /** Gallery card: no controls, muted, plays while `active`. */
  card: { type: Boolean, default: false },
  active: { type: Boolean, default: false },
  posterAt: { type: Number, default: 1.6 },
  /** A self-contained player page (/play/<id>: the work's data inlined); without it the spec is posted to /player.html. */
  src: { type: String, default: '' }
})
const emit = defineEmits(['error', 'time', 'ready'])

const box = ref(null)
const frame = ref(null)
const ready = ref(false)
const playing = ref(false)
const t = ref(0)
const hover = ref(false)
const subtitlesOn = ref(true)
const boxWidth = ref(800)

const scenes = computed(() => props.spec?.scenes || [])
// From the ready message when the frame loads its own data (src), else from the spec.
const frameInfo = ref({ duration: 0, poster: 0 })
const duration = computed(() => scenes.value.reduce((s, sc) => s + (sc.duration || 0), 0) || frameInfo.value.duration)
const starts = computed(() => {
  let at = 0
  return scenes.value.map((sc) => {
    const s = at
    at += sc.duration || 0
    return s
  })
})
const hasNarration = computed(() => scenes.value.some((sc) => sc.narration))
const subtitleTrack = computed(() => buildSubtitles(scenes.value, starts.value))
const subtitle = computed(() => {
  const now = t.value
  const cue = subtitleTrack.value.find((c) => now >= c.start && now < c.end)
  return cue ? cue.text : ''
})
const subtitleSize = computed(() => `${Math.max(12, Math.round(boxWidth.value / 34))}px`)
const sceneIndex = computed(() => {
  let i = 0
  starts.value.forEach((s, j) => {
    if (t.value >= s) i = j
  })
  return i
})
const sceneLabel = computed(() => (scenes.value.length > 1 ? `${sceneIndex.value + 1}/${scenes.value.length} ${scenes.value[sceneIndex.value]?.title || ''}` : ''))

defineExpose({ seek, play, pause, sceneIndex, t })

// ---- talking to the frame --------------------------------------------------------------------
function post(msg) {
  frame.value?.contentWindow?.postMessage(msg, '*')
}
function payload() {
  const s = props.spec
  return {
    width: s.width,
    height: s.height,
    theme: s.theme,
    loop: !!s.loop,
    scenes: s.scenes.map((sc) => ({ id: sc.id, duration: sc.duration, code: sc.code || '', words: sc.audio?.words || [] }))
  }
}
let booted = false
function sendProject() {
  if (props.src || !booted || !props.spec?.scenes?.length) return
  ready.value = false
  post({ type: 'load', project: JSON.parse(JSON.stringify(payload())) })
}
function onMessage(e) {
  if (!frame.value || e.source !== frame.value.contentWindow || e.data?.source !== 'hivegpt-film') return
  const d = e.data
  if (d.type === 'boot') {
    booted = true
    sendProject()
  } else if (d.type === 'ready') {
    ready.value = true
    frameInfo.value = { duration: d.duration || 0, poster: d.poster || 0 }
    // Until playback starts, show a frame from a third of the way in (the very first frame is often empty).
    post({ type: 'seek', t: (props.card && !props.active) || (!playing.value && t.value === 0) ? posterTime() : t.value })
    emit('ready', d.duration)
    if (props.card && props.active) play()
  } else if (d.type === 'error') {
    emit('error', { scene: d.scene, message: d.message, phase: d.phase })
  }
}
// A work may name its best frame (spec.poster, seconds); otherwise an early frame.
const posterTime = () => {
  if (props.spec?.poster > 0) return Math.min(props.spec.poster, duration.value)
  if (props.src) return frameInfo.value.poster
  return Math.min(props.posterAt, Math.max(0, duration.value * 0.3))
}

// Reload when scene code / timing changes (not on every poll that returns the same spec).
const signature = computed(() => (props.spec ? props.spec.scenes.map((sc) => `${sc.id}:${sc.duration}:${(sc.code || '').length}:${hash(sc.code || '')}`).join('|') + `|${props.spec.width}x${props.spec.height}|${props.spec.theme?.bg}` : ''))
watch(signature, () => {
  sendProject()
  buffers.clear()
})

// ---- clock & audio ---------------------------------------------------------------------------
let ctx = null
let base = 0
let sources = []
let raf = 0
const buffers = new Map()
let wallStart = 0

async function ensureAudio() {
  if (props.card || !props.audioUrl) return
  ctx ||= new (window.AudioContext || window.webkitAudioContext)()
  if (ctx.state === 'suspended') await ctx.resume()
  await Promise.all(
    scenes.value.map(async (sc) => {
      const url = props.audioUrl(sc)
      if (!url || buffers.has(url)) return
      buffers.set(url, null)
      try {
        const data = await audioBlob(url, props.withKey)
        buffers.set(url, await ctx.decodeAudioData(data))
      } catch {
        buffers.delete(url)
      }
    })
  )
}

function startSources(from) {
  stopSources()
  if (!ctx) return
  scenes.value.forEach((sc, i) => {
    const url = props.audioUrl?.(sc)
    const buf = url && buffers.get(url)
    if (!buf) return
    const begin = starts.value[i]
    if (from >= begin + buf.duration) return
    const src = ctx.createBufferSource()
    src.buffer = buf
    src.connect(ctx.destination)
    const when = Math.max(0, begin - from)
    src.start(ctx.currentTime + when, Math.max(0, from - begin))
    sources.push(src)
  })
  base = ctx.currentTime - from
}
function stopSources() {
  sources.forEach((s) => {
    try {
      s.stop()
    } catch {
      /* already stopped */
    }
  })
  sources = []
}

function now() {
  return ctx && sources.length ? ctx.currentTime - base : (performance.now() - wallStart) / 1000
}

async function play() {
  if (!ready.value || !duration.value) return
  if (t.value >= duration.value - 0.05) t.value = 0
  playing.value = true
  await ensureAudio()
  if (!playing.value) return
  startSources(t.value)
  wallStart = performance.now() - t.value * 1000
  cancelAnimationFrame(raf)
  const tick = () => {
    if (!playing.value) return
    let next = now()
    if (next >= duration.value) {
      if (props.spec?.loop || props.card || props.src) {
        next = 0
        wallStart = performance.now()
        startSources(0)
      } else {
        t.value = duration.value
        post({ type: 'seek', t: duration.value - 0.001 })
        pause()
        return
      }
    }
    t.value = next
    post({ type: 'seek', t: next })
    emit('time', next)
    raf = requestAnimationFrame(tick)
  }
  raf = requestAnimationFrame(tick)
}
function pause() {
  playing.value = false
  cancelAnimationFrame(raf)
  stopSources()
}
function toggle() {
  playing.value ? pause() : play()
}
function seek(to) {
  t.value = Math.max(0, Math.min(duration.value, to))
  post({ type: 'seek', t: t.value })
  emit('time', t.value)
  if (playing.value) {
    wallStart = performance.now() - t.value * 1000
    startSources(t.value)
  }
}
function fullscreen() {
  const el = box.value
  if (document.fullscreenElement) document.exitFullscreen()
  else el?.requestFullscreen?.()
}
watch(
  () => props.active,
  (on) => {
    if (!props.card || !ready.value) return
    if (on) play()
    else {
      pause()
      t.value = 0
      post({ type: 'seek', t: posterTime() })
    }
  }
)

function fmt(s) {
  s = Math.max(0, s || 0)
  return `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`
}
function hash(str) {
  let h = 0
  for (let i = 0; i < str.length; i += 7) h = (h * 31 + str.charCodeAt(i)) | 0
  return h
}

let resize = null
onMounted(() => {
  window.addEventListener('message', onMessage)
  resize = new ResizeObserver(() => (boxWidth.value = box.value?.clientWidth || 800))
  if (box.value) resize.observe(box.value)
})
onBeforeUnmount(() => {
  window.removeEventListener('message', onMessage)
  pause()
  resize?.disconnect()
  ctx?.close?.()
})
</script>
