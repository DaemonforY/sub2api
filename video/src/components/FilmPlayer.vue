<template>
  <div ref="box" class="group relative h-full w-full select-none overflow-hidden bg-black" :class="card ? '' : 'rounded-xl'" @mousemove="!card && reveal()" @mouseleave="shown = false">
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
    <!-- A tap on the picture only shows the controls (touch screens have no hover); pausing is the button's job. -->
    <div class="absolute inset-0" @click="!card && onPicture()"></div>

    <div v-if="!ready" class="absolute inset-0 flex items-center justify-center bg-ink-950/70 text-sm text-white/70">
      <span class="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"></span>
    </div>

    <!-- Scenes keep the bottom 14% free for subtitles (video_prompts.go), so by default the line sits inside
         that strip; it only rises above the control bar while the controls are showing. -->
    <div v-if="!sub.off && subtitle && !card" class="pointer-events-none absolute inset-x-0 flex justify-center px-[8%] transition-[bottom] duration-200" :style="subtitlePlace">
      <span class="rounded-lg bg-black/65 px-3 py-1 text-center font-medium leading-snug text-white" :style="{ fontSize: subtitleSize }">{{ subtitle }}</span>
    </div>

    <button v-if="!card && ready && !playing && t < 0.05" class="absolute left-1/2 top-1/2 flex h-16 w-16 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 text-brand-600 shadow-lg transition hover:scale-105" aria-label="播放" @click.stop="play()">
      <Play class="ml-1 h-7 w-7" fill="currentColor" />
    </button>

    <div v-if="!card" class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent px-3 pb-2 pt-8 text-white opacity-0 transition" :class="{ 'opacity-100': controlsShown }" @mouseenter="overBar = true" @mouseleave="overBar = false">
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
        <div v-if="hasNarration" class="relative">
          <button class="rounded px-1.5 py-0.5 hover:bg-white/15" :class="sub.off ? 'text-white/50' : 'text-white'" aria-haspopup="true" :aria-expanded="subMenu" @click.stop="subMenu = !subMenu">字幕<span class="ml-0.5 text-[10px] opacity-70">▾</span></button>
          <div v-if="subMenu" class="absolute bottom-full right-0 z-10 mb-2 w-44 rounded-lg bg-black/85 p-2 text-xs text-white shadow-lg" @click.stop>
            <div class="mb-1 flex items-center justify-between">
              <span class="text-white/60">字幕</span>
              <span class="flex gap-1">
                <button v-for="o in [false, true]" :key="String(o)" class="rounded px-2 py-0.5" :class="sub.off === o ? 'bg-white/25' : 'hover:bg-white/15'" @click="setSub({ off: o })">{{ o ? '关' : '开' }}</button>
              </span>
            </div>
            <template v-if="!sub.off">
              <div class="mb-1 text-white/60">位置</div>
              <div class="mb-2 flex gap-1">
                <button v-for="o in SUBTITLE_POSITIONS" :key="o.id" class="flex-1 rounded py-0.5" :class="sub.position === o.id ? 'bg-white/25' : 'hover:bg-white/15'" @click="setSub({ position: o.id })">{{ o.name }}</button>
              </div>
              <div class="mb-1 text-white/60">大小</div>
              <div class="flex gap-1">
                <button v-for="o in SUBTITLE_SIZES" :key="o.id" class="flex-1 rounded py-0.5" :class="sub.size === o.id ? 'bg-white/25' : 'hover:bg-white/15'" @click="setSub({ size: o.id })">{{ o.name }}</button>
              </div>
            </template>
            <p class="mt-2 leading-4 text-white/45">{{ editable ? '导出的 MP4 / HTML 也按这个来' : '只在你的浏览器里生效' }}</p>
          </div>
        </div>
        <button class="rounded p-1 hover:bg-white/15" aria-label="全屏" @click.stop="fullscreen()"><Maximize class="h-4 w-4" /></button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Maximize, Pause, Play } from 'lucide-vue-next'
import { audioBlob } from '../lib/api'
import { buildSubtitles, subtitleStyle, SUBTITLE_POSITIONS, SUBTITLE_SIZES } from '../lib/subtitles'

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
  src: { type: String, default: '' },
  /** The owner's project page: the 字幕 menu edits the film's own style (emits `subtitles`), which the exports use. */
  editable: { type: Boolean, default: false }
})
const emit = defineEmits(['error', 'time', 'ready', 'subtitles'])

const box = ref(null)
const frame = ref(null)
const ready = ref(false)
const playing = ref(false)
const t = ref(0)
const overBar = ref(false)
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
// Subtitle style: the film's own (spec.subtitles), or for viewers their own choice kept in this browser.
const SUB_KEY = 'hivegpt-video:subtitles'
const viewerSub = ref(null)
try {
  viewerSub.value = JSON.parse(localStorage.getItem(SUB_KEY) || 'null')
} catch {
  viewerSub.value = null
}
const subMenu = ref(false)
const sub = computed(() => subtitleStyle(!props.editable && viewerSub.value ? viewerSub.value : props.spec?.subtitles))
function setSub(change) {
  const next = { position: sub.value.position, size: sub.value.size, off: sub.value.off, ...change }
  if (props.editable) emit('subtitles', next)
  else {
    viewerSub.value = next
    localStorage.setItem(SUB_KEY, JSON.stringify(next))
  }
}
const subtitleSize = computed(() => `${Math.max(12, Math.round((boxWidth.value / 34) * sub.value.scale))}px`)
// Where the line sits; at the bottom it rises above the control bar while that is showing.
const subtitlePlace = computed(() => {
  const pos = sub.value.position
  if (pos === 'top') return { top: '4%' }
  if (pos === 'raised') return { bottom: controlsShown.value ? 'max(16%, 64px)' : '16%' }
  return { bottom: controlsShown.value ? '64px' : '3.5%' }
})
const sceneIndex = computed(() => {
  let i = 0
  starts.value.forEach((s, j) => {
    if (t.value >= s) i = j
  })
  return i
})
// The control bar shows while paused, while the pointer is on it, and for 3 s after the mouse moves or
// the picture is tapped — a resting cursor doesn't keep it (and the lifted subtitles) over the picture.
const controlsShown = computed(() => !playing.value || overBar.value || shown.value || subMenu.value)
const sceneLabel = computed(() => (scenes.value.length > 1 ? `${sceneIndex.value + 1}/${scenes.value.length} ${scenes.value[sceneIndex.value]?.title || ''}` : ''))

// The project page's 字幕 button opens the same menu (the controls stay up while it is open).
function openSubtitles() {
  subMenu.value = true
}
defineExpose({ seek, play, pause, sceneIndex, t, openSubtitles })

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
    common: s.common || '',
    scenes: s.scenes.map((sc) => ({ id: sc.id, duration: sc.duration, code: sc.code || '', words: sc.audio?.words || [] })),
    images: (s.images || []).map((img) => ({ id: img.id, url: img.url, width: img.width, height: img.height }))
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
const signature = computed(() => (props.spec ? props.spec.scenes.map((sc) => `${sc.id}:${sc.duration}:${(sc.code || '').length}:${hash(sc.code || '')}`).join('|') + `|${props.spec.width}x${props.spec.height}|${props.spec.theme?.bg}|${hash(props.spec.common || '')}` : ''))
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
const shown = ref(false)
let hideTimer = 0
function reveal() {
  shown.value = true
  clearTimeout(hideTimer)
  hideTimer = setTimeout(() => (shown.value = false), 3000)
}
function onPicture() {
  subMenu.value = false
  reveal()
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
  clearTimeout(hideTimer)
  window.removeEventListener('message', onMessage)
  pause()
  resize?.disconnect()
  ctx?.close?.()
})
</script>
