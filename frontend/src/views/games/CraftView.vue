<template>
  <div class="craft-page">
    <header class="craft-top">
      <RouterLink to="/games" class="craft-back">← {{ t('games.title') }}</RouterLink>
      <div class="craft-title">
        <h1>{{ t('games.craft.title') }}</h1>
        <p>{{ t('games.craft.tagline') }}</p>
      </div>
      <div class="craft-stats">
        <span>{{ t('games.craft.siteTotal', { n: siteTotal }) }}</span>
        <span>{{ t('games.craft.mine', { n: items.length }) }}</span>
      </div>
    </header>

    <div v-if="recent.length" class="craft-ticker" aria-live="polite">
      <span class="craft-ticker-k">{{ t('games.craft.latest') }}</span>
      <div class="craft-ticker-track" :class="{ loop: recent.length >= 6 }">
        <span v-for="(el, i) in recent.length >= 6 ? [...recent, ...recent] : recent" :key="i" class="craft-ticker-item">{{ el.emoji }} {{ el.name }}</span>
      </div>
    </div>

    <main class="craft-main">
      <section
        ref="boardEl"
        class="craft-board"
        :class="{ 'is-hint': !chips.length }"
        data-testid="craft-board"
      >
        <canvas ref="fxEl" class="craft-fx" aria-hidden="true"></canvas>
        <p v-if="!chips.length" class="craft-hint">{{ isTouch ? t('games.craft.hintTouch') : t('games.craft.hint') }}</p>
        <div
          v-for="c in chips"
          :key="c.uid"
          class="craft-chip"
          :class="{ busy: c.busy, drag: dragging?.uid === c.uid, target: dropTarget === c.uid, pop: c.pop, first: c.first }"
          :style="{ transform: `translate(${c.x}px, ${c.y}px) translate(-50%, -50%)` }"
          :title="c.desc"
          :data-name="c.name"
          @pointerdown="startDrag($event, c)"
          @dblclick="duplicate(c)"
        >
          <span class="e">{{ c.emoji }}</span><span class="n">{{ c.name }}</span>
          <i v-if="c.busy" class="craft-orbit" aria-hidden="true"></i>
        </div>
        <div v-if="busyText" class="craft-busy">{{ busyText }}</div>
        <div class="craft-board-actions">
          <button type="button" @click="clearBoard">{{ t('games.craft.clear') }}</button>
        </div>
      </section>

      <aside ref="shelfEl" class="craft-shelf" :class="{ drop: overShelf }">
        <input v-model="query" class="craft-search" :placeholder="t('games.craft.search', { n: items.length })" />
        <div class="craft-items">
          <button
            v-for="it in shownItems"
            :key="it.name"
            type="button"
            class="craft-item"
            :class="{ first: it.first, fresh: it.name === freshName }"
            :title="it.desc"
            @pointerdown="shelfDown($event, it)"
            @click="shelfClick(it)"
          >
            <span class="e">{{ it.emoji }}</span>{{ it.name }}<b v-if="it.first" class="star">★</b>
          </button>
        </div>
        <p class="craft-shelf-note">{{ t('games.craft.shelfNote') }}</p>
        <button type="button" class="craft-reset" @click="resetAll">{{ t('games.craft.reset') }}</button>
      </aside>
    </main>

    <Transition name="craft-toast">
      <div v-if="toast" class="craft-toast" :class="toast.kind" role="status">{{ toast.text }}</div>
    </Transition>

    <Transition name="craft-modal">
      <div v-if="discovery" class="craft-modal" @click.self="discovery = null">
        <div class="craft-card" data-testid="craft-first">
          <div class="craft-card-k">{{ t('games.craft.firstKicker') }}</div>
          <div class="craft-card-emoji">{{ discovery.el.emoji }}</div>
          <h2>{{ discovery.el.name }}</h2>
          <p class="craft-card-recipe">{{ discovery.a }} + {{ discovery.b }}</p>
          <p class="craft-card-desc">{{ discovery.el.desc }}</p>
          <p class="craft-card-no">{{ t('games.craft.firstNo', { n: discovery.el.id }) }}</p>
          <div class="craft-card-actions">
            <button type="button" class="primary" @click="openShare">{{ t('games.craft.share') }}</button>
            <button type="button" @click="discovery = null">{{ t('games.craft.keepGoing') }}</button>
          </div>
        </div>
      </div>
    </Transition>

    <ShareCardDialog :show="shareOpen" :input="shareInput" @close="shareOpen = false" />
  </div>
</template>

<script setup lang="ts">
// 小游戏「AI 万物合成」(/games/craft). Drag one element onto another and the server's text model invents
// the result; every pair is asked once for the whole site, so most combines come back from the recipe
// book at once. The player's collection lives in localStorage; the board is per visit.
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { craftCombine, craftStart, type CraftElement } from '@/api/gameCraft'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'
import { track } from '@/utils/analytics'
import { extractApiErrorMessage } from '@/utils/apiError'

interface Item {
  id: number
  name: string
  emoji: string
  desc: string
  first?: boolean
}
interface Chip extends Item {
  uid: number
  x: number
  y: number
  busy?: boolean
  pop?: boolean
}

const SAVE_KEY = 'hg_craft_v1'
const MERGE_DIST = 58
const FALLBACK: Item[] = [
  { id: 1, name: '金', emoji: '🪙', desc: '五行之金' },
  { id: 2, name: '木', emoji: '🌳', desc: '五行之木' },
  { id: 3, name: '水', emoji: '💧', desc: '五行之水' },
  { id: 4, name: '火', emoji: '🔥', desc: '五行之火' },
  { id: 5, name: '土', emoji: '⛰️', desc: '五行之土' }
]

const { t } = useI18n()
const items = ref<Item[]>([])
const chips = ref<Chip[]>([])
const recent = ref<CraftElement[]>([])
const siteTotal = ref(5)
const query = ref('')
const freshName = ref('')
const toast = ref<{ text: string; kind: string } | null>(null)
const discovery = ref<{ el: Item; a: string; b: string } | null>(null)
const shareOpen = ref(false)
const busyText = ref('')
const boardEl = ref<HTMLElement | null>(null)
const shelfEl = ref<HTMLElement | null>(null)
const fxEl = ref<HTMLCanvasElement | null>(null)
const dragging = ref<Chip | null>(null)
const dropTarget = ref<number | null>(null)
const overShelf = ref(false)
const isTouch = ref(false)
let uidSeq = 0
let started = false
let toastTimer: ReturnType<typeof setTimeout> | null = null
let busyTimer: ReturnType<typeof setInterval> | null = null

const shownItems = computed(() => {
  const q = query.value.trim()
  return q ? items.value.filter((it) => it.name.includes(q)) : items.value
})

// ---- collection ----

function load(): Item[] {
  try {
    const saved = JSON.parse(localStorage.getItem(SAVE_KEY) || 'null') as { items?: Item[] } | null
    if (saved?.items?.length) return saved.items
  } catch {
    // corrupted: start over
  }
  return []
}

function save() {
  try {
    localStorage.setItem(SAVE_KEY, JSON.stringify({ items: items.value }))
  } catch {
    // private mode: the collection lasts this visit
  }
}

function addItem(el: Item): boolean {
  const old = items.value.find((it) => it.name === el.name)
  if (old) {
    if (el.first && !old.first) old.first = true
    return false
  }
  items.value.push({ id: el.id, name: el.name, emoji: el.emoji, desc: el.desc, first: el.first || undefined })
  save()
  return true
}

function showToast(text: string, kind = '') {
  toast.value = { text, kind }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = null), 3200)
}

// ---- board ----

function boardRect() {
  return boardEl.value?.getBoundingClientRect() ?? new DOMRect(0, 0, 600, 400)
}

function spawn(it: Item, x?: number, y?: number): Chip {
  const r = boardRect()
  const chip: Chip = {
    ...it,
    uid: ++uidSeq,
    x: x ?? r.width * (0.3 + Math.random() * 0.4),
    y: y ?? r.height * (0.3 + Math.random() * 0.4)
  }
  chips.value.push(chip)
  return chips.value[chips.value.length - 1]
}

function clamp(c: Chip) {
  const r = boardRect()
  c.x = Math.min(Math.max(c.x, 40), r.width - 40)
  c.y = Math.min(Math.max(c.y, 24), r.height - 24)
}

function duplicate(c: Chip) {
  if (c.busy) return
  spawn(c, c.x + 24, c.y + 24)
}

function clearBoard() {
  chips.value = chips.value.filter((c) => c.busy)
}

function resetAll() {
  if (!window.confirm(t('games.craft.resetConfirm'))) return
  items.value = FALLBACK.map((f) => ({ ...f }))
  void craftStart().then((s) => {
    if (s.starters.length) items.value = s.starters.map((e) => ({ ...e }))
    save()
  }).catch(() => save())
  chips.value = []
}

let drag: { chip: Chip; dx: number; dy: number; moved: boolean; sx: number; sy: number; pid: number } | null = null

function nearest(c: Chip): Chip | null {
  let best: Chip | null = null
  let bestD = MERGE_DIST
  for (const o of chips.value) {
    if (o.uid === c.uid || o.busy) continue
    const d = Math.hypot(o.x - c.x, o.y - c.y)
    if (d < bestD) {
      best = o
      bestD = d
    }
  }
  return best
}

function overShelfAt(x: number, y: number): boolean {
  const r = shelfEl.value?.getBoundingClientRect()
  return !!r && x >= r.left && x <= r.right && y >= r.top && y <= r.bottom
}

function beginDrag(e: PointerEvent, chip: Chip) {
  const r = boardRect()
  drag = { chip, dx: e.clientX - r.left - chip.x, dy: e.clientY - r.top - chip.y, moved: false, sx: e.clientX, sy: e.clientY, pid: e.pointerId }
  dragging.value = chip
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

function startDrag(e: PointerEvent, c: Chip) {
  if (c.busy || e.button > 0) return
  e.preventDefault()
  // Bring to front.
  chips.value = [...chips.value.filter((o) => o.uid !== c.uid), c]
  beginDrag(e, chips.value[chips.value.length - 1])
}

function onMove(e: PointerEvent) {
  if (!drag || e.pointerId !== drag.pid) return
  const r = boardRect()
  if (Math.hypot(e.clientX - drag.sx, e.clientY - drag.sy) > 4) drag.moved = true
  drag.chip.x = e.clientX - r.left - drag.dx
  drag.chip.y = e.clientY - r.top - drag.dy
  dropTarget.value = nearest(drag.chip)?.uid ?? null
  overShelf.value = overShelfAt(e.clientX, e.clientY)
}

function onUp(e: PointerEvent) {
  if (!drag || e.pointerId !== drag.pid) return
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onUp)
  const chip = drag.chip
  drag = null
  dragging.value = null
  dropTarget.value = null
  if (overShelf.value) {
    overShelf.value = false
    chips.value = chips.value.filter((c) => c.uid !== chip.uid)
    return
  }
  clamp(chip)
  const other = nearest(chip)
  if (other) void combine(chip, other)
}

// Shelf: a mouse drags a copy out; a tap (or a touch, so the shelf can scroll) places one on the board.
let shelfPress: { it: Item; x: number; y: number; pid: number } | null = null
let shelfDragged = false

function shelfDown(e: PointerEvent, it: Item) {
  shelfDragged = false
  if (e.pointerType !== 'mouse' || e.button > 0) return
  shelfPress = { it, x: e.clientX, y: e.clientY, pid: e.pointerId }
  window.addEventListener('pointermove', shelfMove)
  window.addEventListener('pointerup', shelfUp, { once: true })
}

function shelfMove(e: PointerEvent) {
  if (!shelfPress || Math.hypot(e.clientX - shelfPress.x, e.clientY - shelfPress.y) < 6) return
  window.removeEventListener('pointermove', shelfMove)
  const r = boardRect()
  const chip = spawn(shelfPress.it, e.clientX - r.left, e.clientY - r.top)
  shelfPress = null
  shelfDragged = true
  beginDrag(e, chip)
  drag!.dx = 0
  drag!.dy = 0
  drag!.moved = true
}

function shelfUp() {
  window.removeEventListener('pointermove', shelfMove)
  shelfPress = null
}

function shelfClick(it: Item) {
  if (shelfDragged) {
    shelfDragged = false
    return
  }
  const placed = spawn(it)
  // On a phone the last placed chip is where the next one wants to go: drop it next to the newest one.
  const prev = chips.value[chips.value.length - 2]
  if (isTouch.value && prev && !prev.busy) {
    placed.x = prev.x + 70
    placed.y = prev.y
    clamp(placed)
  }
}

// ---- combine ----

const THINKING = computed(() => [t('games.craft.thinking1'), t('games.craft.thinking2'), t('games.craft.thinking3'), t('games.craft.thinking4')])

async function combine(a: Chip, b: Chip) {
  if (!started) {
    started = true
    track('game_start', { game: 'craft' })
  }
  const mx = (a.x + b.x) / 2
  const my = (a.y + b.y) / 2
  a.busy = b.busy = true
  a.x = b.x = mx
  a.y = b.y = my
  burst(mx, my, 10, ['#a78bfa', '#f0abfc'])
  // Most pairs are in the recipe book and come back at once; a slow one is a new invention.
  let step = 0
  const slow = setTimeout(() => {
    busyText.value = THINKING.value[0]
    busyTimer = setInterval(() => (busyText.value = THINKING.value[++step % THINKING.value.length]), 1800)
  }, 700)
  try {
    const res = await craftCombine(a.name, b.name)
    const el: Item = { ...res.element, first: res.first }
    chips.value = chips.value.filter((c) => c.uid !== a.uid && c.uid !== b.uid)
    const chip = spawn(el, mx, my)
    chip.pop = true
    setTimeout(() => (chip.pop = false), 700)
    const isNew = addItem(el)
    if (res.first) {
      siteTotal.value = Math.max(siteTotal.value + 1, res.element.id)
      recent.value = [res.element, ...recent.value].slice(0, 12)
      burst(mx, my, 90, ['#fde047', '#f59e0b', '#fb7185', '#a78bfa', '#ffffff'], 1.6)
      discovery.value = { el, a: a.name, b: b.name }
      track('game_finish', { game: 'craft', kind: 'first', type: el.name })
    } else if (isNew) {
      burst(mx, my, 40, ['#a78bfa', '#67e8f9', '#ffffff'])
      showToast(t('games.craft.newForYou', { e: el.emoji, n: el.name }), 'new')
    } else {
      burst(mx, my, 14, ['#c4b5fd'])
    }
    if (isNew) {
      freshName.value = el.name
      setTimeout(() => freshName.value === el.name && (freshName.value = ''), 2500)
    }
  } catch (err) {
    a.busy = b.busy = false
    a.x -= 34
    b.x += 34
    clamp(a)
    clamp(b)
    showToast(extractApiErrorMessage(err, t('games.craft.failed')), 'err')
  } finally {
    clearTimeout(slow)
    if (busyTimer) clearInterval(busyTimer)
    busyTimer = null
    busyText.value = ''
  }
}

// ---- particles ----

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  life: number
  max: number
  size: number
  color: string
}
let particles: Particle[] = []
let fxRaf = 0

function burst(x: number, y: number, n: number, colors: string[], power = 1) {
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
  for (let i = 0; i < n; i++) {
    const a = Math.random() * Math.PI * 2
    const sp = (1.5 + Math.random() * 4.5) * power
    particles.push({ x, y, vx: Math.cos(a) * sp, vy: Math.sin(a) * sp - 1.2 * power, life: 0, max: 40 + Math.random() * 40 * power, size: 1.5 + Math.random() * 3 * power, color: colors[i % colors.length] })
  }
  if (!fxRaf) fxRaf = requestAnimationFrame(tick)
}

function tick() {
  const cv = fxEl.value
  const ctx = cv?.getContext('2d')
  if (!cv || !ctx) {
    fxRaf = 0
    return
  }
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  const r = cv.getBoundingClientRect()
  if (cv.width !== Math.round(r.width * dpr)) {
    cv.width = Math.round(r.width * dpr)
    cv.height = Math.round(r.height * dpr)
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, r.width, r.height)
  ctx.globalCompositeOperation = 'lighter'
  particles = particles.filter((p) => ++p.life < p.max)
  for (const p of particles) {
    p.x += p.vx
    p.y += p.vy
    p.vx *= 0.96
    p.vy = p.vy * 0.96 + 0.06
    const k = 1 - p.life / p.max
    ctx.globalAlpha = k
    ctx.fillStyle = p.color
    ctx.beginPath()
    ctx.arc(p.x, p.y, p.size * (0.4 + k), 0, Math.PI * 2)
    ctx.fill()
  }
  ctx.globalAlpha = 1
  fxRaf = particles.length ? requestAnimationFrame(tick) : 0
  if (!fxRaf) ctx.clearRect(0, 0, r.width, r.height)
}

// ---- share ----

const shareInput = computed<ShareCardInput>(() => {
  const d = discovery.value
  return {
    kind: 'summary',
    brand: t('games.share.brand'),
    title: d ? t('games.craft.shareTitle', { e: d.el.emoji, n: d.el.name }) : t('games.craft.title'),
    summary: d?.el.desc || t('games.craft.tagline'),
    points: d ? [`${d.a} + ${d.b} = ${d.el.emoji} ${d.el.name}`, t('games.craft.firstNo', { n: d.el.id })] : [],
    cta: t('games.craft.shareCta'),
    pageUrl: window.location.origin + '/games/craft',
    trackKind: 'craft_first'
  }
})

function openShare() {
  shareOpen.value = true
}

// ---- lifecycle ----

onMounted(async () => {
  isTouch.value = window.matchMedia?.('(pointer: coarse)').matches ?? false
  items.value = load()
  if (!items.value.length) items.value = FALLBACK.map((f) => ({ ...f }))
  await nextTick()
  // A first visit gets the five elements on the board.
  if (!chips.value.length) {
    const r = boardRect()
    const starters = items.value.slice(0, 5)
    starters.forEach((it, i) => {
      const a = (i / starters.length) * Math.PI * 2 - Math.PI / 2
      const rad = Math.min(r.width, r.height) * 0.28
      spawn(it, r.width / 2 + Math.cos(a) * rad, r.height / 2 + Math.sin(a) * rad)
    })
  }
  try {
    const s = await craftStart()
    siteTotal.value = s.total
    recent.value = s.recent
    if (s.starters.length) {
      for (const st of s.starters) addItem(st)
    }
  } catch {
    // offline API: the starters above still let the page render
  }
})

onUnmounted(() => {
  if (fxRaf) cancelAnimationFrame(fxRaf)
  if (toastTimer) clearTimeout(toastTimer)
  if (busyTimer) clearInterval(busyTimer)
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onUp)
  window.removeEventListener('pointermove', shelfMove)
})
</script>

<style scoped>
.craft-page {
  position: relative;
  display: flex;
  height: 100dvh;
  flex-direction: column;
  overflow: hidden;
  color: #ede9fe;
  background:
    radial-gradient(1200px 600px at 20% -10%, rgba(124, 58, 237, 0.35), transparent 60%),
    radial-gradient(900px 500px at 110% 110%, rgba(14, 165, 233, 0.22), transparent 60%),
    #0b0a16;
  font-family: 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif;
  -webkit-user-select: none;
  user-select: none;
}
.craft-top {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px max(16px, env(safe-area-inset-left)) 8px;
}
.craft-back {
  flex: none;
  color: #c4b5fd;
  font-size: 14px;
}
.craft-title {
  flex: 1;
  min-width: 0;
}
.craft-title h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.06em;
  background: linear-gradient(90deg, #fde68a, #f0abfc 45%, #a5b4fc);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.craft-title p {
  margin: 2px 0 0;
  font-size: 13px;
  color: rgba(237, 233, 254, 0.62);
}
.craft-stats {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-size: 12px;
  color: rgba(237, 233, 254, 0.7);
  white-space: nowrap;
}
.craft-ticker {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 16px 8px;
  overflow: hidden;
  font-size: 12px;
  color: rgba(237, 233, 254, 0.75);
  mask-image: linear-gradient(90deg, #000 85%, transparent);
}
.craft-ticker-k {
  flex: none;
  border-radius: 999px;
  background: rgba(250, 204, 21, 0.15);
  padding: 2px 8px;
  color: #fde68a;
}
.craft-ticker-track {
  display: flex;
  gap: 14px;
  white-space: nowrap;
}
.craft-ticker-track.loop {
  animation: craft-scroll 40s linear infinite;
}
.craft-ticker-item {
  flex: none;
}
@keyframes craft-scroll {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(-50%);
  }
}
.craft-main {
  display: flex;
  flex: 1;
  min-height: 0;
  gap: 12px;
  padding: 0 16px max(16px, env(safe-area-inset-bottom));
}
.craft-board {
  position: relative;
  flex: 1;
  min-height: 420px;
  overflow: hidden;
  border: 1px solid rgba(167, 139, 250, 0.22);
  border-radius: 22px;
  background-color: rgba(15, 12, 35, 0.55);
  background-image: radial-gradient(rgba(196, 181, 253, 0.13) 1px, transparent 1px);
  background-size: 22px 22px;
  touch-action: none;
}
.craft-board::before {
  content: '';
  position: absolute;
  inset: -40%;
  z-index: 0;
  background: conic-gradient(from 0deg, transparent 0deg, rgba(167, 139, 250, 0.16) 60deg, transparent 120deg, rgba(56, 189, 248, 0.12) 200deg, transparent 260deg, rgba(240, 171, 252, 0.12) 320deg, transparent 360deg);
  filter: blur(40px);
  animation: craft-aurora 28s linear infinite;
  pointer-events: none;
}
@keyframes craft-aurora {
  to {
    transform: rotate(360deg);
  }
}
@media (prefers-reduced-motion: reduce) {
  .craft-board::before,
  .craft-ticker-track.loop {
    animation: none;
  }
}
.craft-fx {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 5;
}
.craft-hint {
  position: absolute;
  left: 50%;
  bottom: 18px;
  transform: translateX(-50%);
  margin: 0;
  font-size: 13px;
  color: rgba(237, 233, 254, 0.5);
  white-space: nowrap;
}
.craft-chip {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px 8px 10px;
  border: 1px solid rgba(196, 181, 253, 0.35);
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(49, 39, 92, 0.92), rgba(30, 24, 60, 0.92));
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.08);
  font-size: 16px;
  font-weight: 600;
  color: #f5f3ff;
  white-space: nowrap;
  cursor: grab;
  touch-action: none;
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.35s cubic-bezier(0.2, 0.9, 0.3, 1.2);
}
.craft-chip.drag {
  z-index: 4;
  cursor: grabbing;
  transition: box-shadow 0.15s, border-color 0.15s;
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(196, 181, 253, 0.6);
}
.craft-chip.target {
  border-color: #fde68a;
  box-shadow: 0 0 0 4px rgba(253, 230, 138, 0.25), 0 0 26px rgba(253, 230, 138, 0.45);
}
.craft-chip.first {
  border-color: rgba(253, 230, 138, 0.7);
}
.craft-chip.busy {
  cursor: progress;
  opacity: 0.85;
}
.craft-chip.pop .e,
.craft-chip.pop .n {
  animation: craft-pop 0.6s cubic-bezier(0.2, 1.4, 0.4, 1);
}
.craft-chip .e {
  font-size: 22px;
  line-height: 1;
}
@keyframes craft-pop {
  0% {
    opacity: 0;
    transform: scale(0.2);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}
.craft-orbit {
  position: absolute;
  inset: -8px;
  border-radius: 999px;
  border: 2px solid transparent;
  border-top-color: #fde68a;
  border-right-color: #f0abfc;
  animation: craft-spin 0.9s linear infinite;
}
@keyframes craft-spin {
  to {
    transform: rotate(360deg);
  }
}
.craft-busy {
  position: absolute;
  left: 50%;
  top: 14px;
  z-index: 6;
  transform: translateX(-50%);
  border-radius: 999px;
  background: rgba(253, 230, 138, 0.12);
  padding: 5px 14px;
  font-size: 13px;
  color: #fde68a;
  white-space: nowrap;
}
.craft-board-actions {
  position: absolute;
  right: 12px;
  bottom: 12px;
  z-index: 3;
}
.craft-board-actions button,
.craft-reset {
  border: 1px solid rgba(196, 181, 253, 0.25);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  padding: 4px 12px;
  font-size: 12px;
  color: rgba(237, 233, 254, 0.7);
}
.craft-shelf {
  display: flex;
  width: 300px;
  flex: none;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  border: 1px solid rgba(167, 139, 250, 0.18);
  border-radius: 22px;
  background: rgba(15, 12, 35, 0.5);
  padding: 12px;
  transition: border-color 0.15s, background 0.15s;
}
.craft-shelf.drop {
  border-color: rgba(251, 113, 133, 0.6);
  background: rgba(251, 113, 133, 0.08);
}
.craft-search {
  width: 100%;
  border: 1px solid rgba(196, 181, 253, 0.2);
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.25);
  padding: 8px 12px;
  font-size: 16px;
  color: #f5f3ff;
  outline: none;
  -webkit-user-select: text;
  user-select: text;
}
.craft-items {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 6px;
  min-height: 0;
  overflow-y: auto;
}
.craft-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 1px solid rgba(196, 181, 253, 0.2);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.04);
  padding: 5px 9px;
  font-size: 14px;
  color: #ede9fe;
  cursor: grab;
  transition: background 0.15s, border-color 0.15s;
}
.craft-item:hover {
  background: rgba(167, 139, 250, 0.16);
}
.craft-item.first {
  border-color: rgba(253, 230, 138, 0.55);
}
.craft-item.fresh {
  animation: craft-fresh 2.4s ease-out;
}
@keyframes craft-fresh {
  0% {
    background: rgba(253, 230, 138, 0.5);
  }
  100% {
    background: rgba(255, 255, 255, 0.04);
  }
}
.craft-item .star {
  color: #fde68a;
  font-size: 11px;
}
.craft-shelf-note {
  margin: 0;
  font-size: 11px;
  line-height: 1.5;
  color: rgba(237, 233, 254, 0.45);
}
.craft-reset {
  align-self: flex-start;
}
.craft-toast {
  position: fixed;
  left: 50%;
  bottom: calc(28px + env(safe-area-inset-bottom));
  z-index: 40;
  transform: translateX(-50%);
  max-width: min(92vw, 520px);
  border-radius: 14px;
  background: rgba(30, 24, 60, 0.95);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  padding: 10px 16px;
  font-size: 14px;
  color: #f5f3ff;
}
.craft-toast.new {
  border: 1px solid rgba(103, 232, 249, 0.5);
}
.craft-toast.err {
  border: 1px solid rgba(251, 113, 133, 0.6);
}
.craft-toast-enter-active,
.craft-toast-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.craft-toast-enter-from,
.craft-toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 10px);
}
.craft-modal {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(5, 4, 12, 0.6);
  backdrop-filter: blur(6px);
  padding: 20px;
}
.craft-card {
  position: relative;
  width: min(380px, 100%);
  border: 1px solid rgba(253, 230, 138, 0.5);
  border-radius: 26px;
  background:
    radial-gradient(300px 200px at 50% 0%, rgba(253, 230, 138, 0.22), transparent 70%),
    linear-gradient(180deg, #221a46, #140f2c);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.6), 0 0 60px rgba(253, 230, 138, 0.18);
  padding: 28px 24px 22px;
  text-align: center;
}
.craft-card-k {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.3em;
  color: #fde68a;
}
.craft-card-emoji {
  margin: 14px 0 4px;
  font-size: 72px;
  line-height: 1;
  animation: craft-pop 0.8s cubic-bezier(0.2, 1.4, 0.4, 1);
}
.craft-card h2 {
  margin: 6px 0 2px;
  font-size: 30px;
  font-weight: 800;
  color: #fff;
}
.craft-card-recipe {
  margin: 0;
  font-size: 13px;
  color: rgba(237, 233, 254, 0.6);
}
.craft-card-desc {
  margin: 14px 0 6px;
  font-size: 15px;
  line-height: 1.6;
  color: #ede9fe;
}
.craft-card-no {
  margin: 0 0 18px;
  font-size: 12px;
  color: rgba(253, 230, 138, 0.8);
}
.craft-card-actions {
  display: flex;
  justify-content: center;
  gap: 10px;
}
.craft-card-actions button {
  border: 1px solid rgba(196, 181, 253, 0.35);
  border-radius: 999px;
  padding: 9px 18px;
  font-size: 14px;
  color: #ede9fe;
}
.craft-card-actions .primary {
  border-color: transparent;
  background: linear-gradient(90deg, #f59e0b, #ec4899);
  font-weight: 700;
  color: #fff;
}
.craft-modal-enter-active,
.craft-modal-leave-active {
  transition: opacity 0.25s;
}
.craft-modal-enter-from,
.craft-modal-leave-to {
  opacity: 0;
}
@media (max-width: 760px) {
  .craft-top {
    gap: 10px;
    padding-top: 10px;
  }
  .craft-title h1 {
    font-size: 19px;
  }
  .craft-title p {
    display: none;
  }
  .craft-main {
    flex-direction: column;
    gap: 10px;
    padding: 0 10px max(10px, env(safe-area-inset-bottom));
  }
  .craft-board {
    min-height: 0;
    flex: 1 1 58%;
  }
  .craft-shelf {
    width: auto;
    flex: 1 1 42%;
    padding: 10px;
  }
  .craft-shelf-note {
    display: none;
  }
  .craft-reset {
    display: none;
  }
}
</style>
