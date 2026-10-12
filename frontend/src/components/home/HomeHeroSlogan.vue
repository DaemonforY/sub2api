<template>
  <div ref="root" class="hero-slogan" :style="hudPad ? { paddingBottom: `${hudPad}px` } : undefined">
    <component :is="as ?? 'h1'" :class="['slogan', { sub: as === 'h2', waiting }]" :aria-label="lead + highlight" data-testid="home-hero-slogan">
      <span v-for="(line, li) in lines" :key="li" :class="['ln', li === 0 ? 'l1' : 'l2', { flow: done && li === 1, shine: done }]" :data-text="line.text" aria-hidden="true">
        <template v-if="done">{{ line.text }}</template>
        <span
          v-for="(c, ci) in line.chars"
          v-else
          :key="ci"
          :class="['ch', { scr: c.scrambling, lock: c.locked }]"
          :style="c.width ? { width: c.width } : undefined"
        >{{ c.shown }}<i v-if="c.locked && c.id" class="tokid">#{{ c.id }}</i></span>
      </span>
    </component>
    <div class="rail" aria-hidden="true"><i style="left: 20%"></i><i style="left: 45%"></i><i style="left: 70%"></i><i style="left: 92%"></i></div>
    <div v-for="(h, i) in huds" :key="h.idx" :class="['hud', { on: hudsOn }]" :style="hudStyle[i]" aria-hidden="true">
      <svg width="60" height="40" class="lead"><path :d="i === 0 ? 'M0 26 L22 26 L60 2' : 'M0 26 L22 26 L60 12'" /><circle cx="0" cy="26" r="2.4" /></svg>
      <div class="idx">{{ h.idx }}</div>
      <div class="nm">{{ h.name }}</div>
      <div class="sp">{{ h.spec }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { prefersReducedMotion } from './motion/tier'

/**
 * A two-line heading arriving as tokens once it scrolls into view: every character scrambles
 * through noise, then locks in left to right with a flash and the token id it "was" floating off.
 * Then the second line keeps a flowing gradient, both lines get a passing shine, a rail with light
 * runs underneath and two HUD tags point at the words (from the exploded-view film).
 * As an h2 it is set a size smaller.
 */
const props = defineProps<{ lead: string; highlight: string; huds: { idx: string; name: string; spec: string }[]; as?: 'h1' | 'h2' }>()

type Char = { c: string; shown: string; width: string; scrambling: boolean; locked: boolean; id: number }
const NOISE = ['0', '1', '#', '$', '{', '}', '<', '>', '=', '+', '%', '画', '写', '码', '词', '数', '流', '光', 'A', 'I']
const LEAD_MS = 380, STEP_MS = 95, SWAP_MS = 55, FLOAT_MS = 1500
const HUD_GAP = 62, HUD_H = 64 // a tag's step down and its height

const toChars = (s: string): Char[] => [...s].map((c, i) => ({ c, shown: c === ' ' ? ' ' : c, width: '', scrambling: false, locked: false, id: 1000 + ((i * 7919 + 4127) % 90000) }))
const lines = reactive([{ text: props.lead, chars: toChars(props.lead) }, { text: props.highlight, chars: toChars(props.highlight) }])
const done = ref(false)
const waiting = ref(false) // hidden until it scrolls into view and starts decoding
const hudsOn = ref(false)
const root = ref<HTMLElement | null>(null)
const hudPos = ref<{ x: number; y: number; show: boolean }[]>([])
const hudPad = ref(0) // room below the words so the lower tag clears whatever comes next
const hudStyle = computed(() => props.huds.map((_, i) => {
  const p = hudPos.value[i]
  return p ? { transform: `translate(${p.x}px, ${p.y}px)`, display: p.show ? '' : 'none' } : { display: 'none' }
}))

let raf = 0
const timers: ReturnType<typeof setTimeout>[] = []
let seen: IntersectionObserver | null = null

function placeHuds() {
  const el = root.value
  const l2 = el?.querySelector('.l2')
  if (!el || !l2) return
  const box = el.getBoundingClientRect(), r = l2.getBoundingClientRect()
  const x = r.right - box.left + 78
  const room = box.width + 40 - x >= 130 // the tags may run into the gap between the columns
  const y = (i: number) => r.top - box.top - 12 + i * HUD_GAP
  hudPos.value = props.huds.map((_, i) => ({ x, y: y(i), show: room }))
  const bottom = props.huds.length ? y(props.huds.length - 1) + HUD_H : 0
  hudPad.value = room ? Math.max(0, Math.ceil(bottom - (box.height - hudPad.value))) : 0
}

function settle() {
  done.value = true
  void nextTick(() => {
    placeHuds()
    props.huds.forEach((_, i) => timers.push(setTimeout(() => (hudsOn.value = true), 200 + i * 350)))
  })
}

async function decode() {
  waiting.value = false
  if (prefersReducedMotion()) return settle()
  await document.fonts?.ready
  await nextTick()
  // one fixed-width box per character, measured on the real glyph, so the noise never shifts the line
  const boxes = root.value?.querySelectorAll<HTMLElement>('.ch') ?? []
  const all = lines.flatMap((l) => l.chars)
  boxes.forEach((b, i) => { if (all[i]) all[i].width = `${b.getBoundingClientRect().width}px` })
  all.forEach((c) => { c.scrambling = c.c !== ' ' })
  const t0 = performance.now()
  let lastSwap = 0
  const tick = (now: number) => {
    const swap = now - lastSwap > SWAP_MS
    if (swap) lastSwap = now
    let locked = 0
    all.forEach((c, i) => {
      if (c.locked) { locked++; return }
      if (now - t0 >= LEAD_MS + i * STEP_MS) {
        c.locked = true
        c.scrambling = false
        c.shown = c.c === ' ' ? ' ' : c.c
        if ('，。, .'.includes(c.c)) c.id = 0
        locked++
      } else if (swap && c.scrambling) c.shown = NOISE[Math.floor(Math.random() * NOISE.length)]
    })
    if (locked < all.length) raf = requestAnimationFrame(tick)
    else timers.push(setTimeout(settle, FLOAT_MS)) // let the token ids float off first
  }
  raf = requestAnimationFrame(tick)
}

onMounted(() => {
  window.addEventListener('resize', placeHuds)
  if (typeof IntersectionObserver === 'undefined' || !root.value) return void decode()
  waiting.value = true
  seen = new IntersectionObserver(([e]) => {
    if (!e.isIntersecting) return
    seen?.disconnect()
    void decode()
  }, { threshold: 0.6 })
  seen.observe(root.value)
})

onBeforeUnmount(() => {
  seen?.disconnect()
  cancelAnimationFrame(raf)
  timers.forEach(clearTimeout)
  window.removeEventListener('resize', placeHuds)
})
</script>

<style scoped>
.hero-slogan {
  position: relative;
}
.slogan {
  position: relative;
  font-size: clamp(44px, 6.2vw, 88px);
  line-height: 1.06;
  font-weight: 800;
  letter-spacing: -0.035em;
  word-break: keep-all;
  color: #fff;
}
.slogan.sub {
  font-size: clamp(40px, 5vw, 72px);
}
.slogan.waiting .ln {
  opacity: 0;
}
.ln {
  position: relative;
  display: block;
  width: max-content;
  max-width: 100%;
}
.ch {
  position: relative;
  display: inline-block;
  text-align: center;
}
.ch.scr {
  color: #5ee7ff;
  opacity: 0.75;
  font-weight: 600;
}
.ch.lock {
  animation: lock 0.7s ease-out;
}
@keyframes lock {
  0% { text-shadow: 0 0 28px #5ee7ff, 0 0 6px #fff; color: #fff; }
  100% { text-shadow: none; }
}
.tokid {
  position: absolute;
  left: 50%;
  top: 100%;
  transform: translate(-50%, 0);
  font: 600 10px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  font-style: normal;
  letter-spacing: 0.06em;
  color: #5ee7ff;
  opacity: 0;
  white-space: nowrap;
  animation: tokid 1.6s ease-out forwards;
  pointer-events: none;
}
@keyframes tokid {
  0% { opacity: 0; transform: translate(-50%, 2px); }
  15% { opacity: 0.9; }
  70% { opacity: 0.6; }
  100% { opacity: 0; transform: translate(-50%, 10px); }
}
/* after decoding: the second line flows, and a band of light sweeps both lines now and then */
.l2.flow {
  background: linear-gradient(100deg, #9fdcff 0%, #5ee7ff 30%, #3b82f6 70%, #5ee7ff 100%);
  background-size: 220% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  animation: flow 5s linear infinite;
  filter: drop-shadow(0 0 18px rgba(59, 130, 246, 0.45));
}
@keyframes flow {
  to { background-position: -220% 0; }
}
.ln.shine::after {
  content: attr(data-text);
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(105deg, transparent 42%, rgba(255, 255, 255, 0.95) 50%, transparent 58%);
  background-size: 300% 100%;
  background-position: 150% 0;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  animation: shine 4.8s ease-in-out infinite;
}
.l2.shine::after {
  animation-delay: 0.25s;
}
@keyframes shine {
  0%, 55% { background-position: 150% 0; }
  85%, 100% { background-position: -50% 0; }
}
/* a rail under the slogan with light running along it */
.rail {
  position: relative;
  height: 2px;
  margin-top: 18px;
  width: min(420px, 80%);
  background: linear-gradient(90deg, rgba(94, 231, 255, 0.05), rgba(94, 231, 255, 0.35));
}
.rail::before,
.rail::after {
  content: '';
  position: absolute;
  top: -2px;
  width: 46px;
  height: 6px;
  border-radius: 6px;
  background: radial-gradient(closest-side, #fff, #5ee7ff 45%, transparent);
  animation: run 2.6s linear infinite;
}
.rail::after {
  animation-delay: 1.3s;
  width: 26px;
}
@keyframes run {
  0% { left: -10%; opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { left: 100%; opacity: 0; }
}
.rail i {
  position: absolute;
  top: -4px;
  width: 1px;
  height: 10px;
  background: rgba(94, 231, 255, 0.45);
}
/* HUD tags: an index, a name, a spec, a leader line */
.hud {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 3;
  padding: 7px 10px;
  border-left: 2px solid #5ee7ff;
  background: rgba(5, 12, 24, 0.66);
  backdrop-filter: blur(8px);
  opacity: 0;
  transition: opacity 0.5s;
  pointer-events: none;
}
.hud.on {
  opacity: 1;
}
.hud .lead {
  position: absolute;
  left: -60px;
  top: 14px;
  overflow: visible;
}
.hud .lead path {
  stroke: #5ee7ff;
  fill: none;
  stroke-width: 1.2;
  opacity: 0.7;
}
.hud .lead circle {
  fill: #5ee7ff;
}
.hud .idx {
  font: 600 9px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: 0.12em;
  color: #5ee7ff;
}
.hud .nm {
  margin-top: 4px;
  font-size: 12px;
  letter-spacing: 0.04em;
  white-space: nowrap;
  color: #eef6ff;
}
.hud .sp {
  margin-top: 2px;
  font-size: 10.5px;
  color: #9fb3cc;
  white-space: nowrap;
}
@media (max-width: 760px) {
  .slogan { font-size: min(46px, 11.5vw); }
  .slogan.sub { font-size: min(40px, 10.5vw); }
  .hud { display: none !important; }
}
@media (prefers-reduced-motion: reduce) {
  .l2.flow, .ln.shine::after, .rail::before, .rail::after, .ch.lock, .tokid { animation: none; }
}
</style>
