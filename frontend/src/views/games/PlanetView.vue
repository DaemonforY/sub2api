<template>
  <div class="planet-page" :class="[`is-${hud.phase}`, { touch: isTouch }]">
    <canvas
      ref="canvasEl"
      class="planet-canvas"
      @pointerdown="stickDown"
      @pointermove="stickMove"
      @pointerup="stickUp"
      @pointercancel="stickUp"
    ></canvas>

    <RouterLink v-if="hud.phase !== 'fly'" to="/games" class="planet-back">← 小游戏</RouterLink>

    <!-- title -->
    <Transition name="planet-fade">
      <section v-if="hud.phase === 'title'" class="planet-title">
        <p class="planet-kicker">HiveGPT 小游戏</p>
        <h1>蜂巢星球</h1>
        <p class="planet-tagline">这颗星球睡着了。飞过的地方，花都会开。</p>
        <button class="planet-go" :disabled="!ready" data-testid="planet-start" @click="start">
          {{ !ready ? '正在造星球…' : resume.fields ? '继续飞' : '起飞' }}
        </button>
        <p v-if="resume.fields" class="planet-resume">
          {{ resume.done ? '星球已经全醒了' : `已唤醒 ${resume.islands} 座岛、${resume.fields} 片花田` }}
          <button type="button" @click="restart">重新开始</button>
        </p>
        <p class="planet-keys">
          <template v-if="isTouch">按住屏幕拖动转向和升降 · 右下角加速</template>
          <template v-else>
            <span><kbd>A</kbd><kbd>D</kbd> 转向</span><span><kbd>W</kbd><kbd>S</kbd> 升降</span><span><kbd>空格</kbd> 加速</span><span>建议开声音</span>
          </template>
        </p>
        <p v-if="failed" class="planet-failed">这台设备没能打开 3D 画面（WebGL 不可用），换个浏览器再试试。</p>
      </section>
    </Transition>

    <!-- flying -->
    <template v-if="hud.phase === 'fly'">
      <div class="planet-hud planet-hud-tl">
        <div class="planet-region">{{ hud.region || '海上' }}</div>
        <div v-if="hud.region" class="planet-fields" :aria-label="`花田 ${fieldsDone}/${hud.fields.length}`">
          <i v-for="(f, i) in hud.fields" :key="i" :class="{ on: f }"></i>
        </div>
        <div class="planet-pollen">花粉 {{ hud.pollen }}</div>
        <div class="planet-honey" :title="hud.levelName ? `下一级：${hud.levelName}` : '蜂蜜已满级'">
          <span>🍯 蜂蜜 {{ hud.level }} 级</span>
          <i v-if="hud.levelNeed"><b :style="{ width: `${honeyPct}%` }"></b></i>
          <em v-if="hud.levelNeed">{{ hud.honey }}/{{ hud.levelNeed }} · {{ hud.levelName }}</em>
          <em v-else>满级</em>
        </div>
      </div>

      <div v-if="hud.race" class="planet-race">
        <strong>{{ hud.race.seconds.toFixed(1) }}<small>秒</small></strong>
        <span>环 {{ Math.min(hud.race.passed, hud.race.total) }}/{{ hud.race.total }}</span>
        <span>金牌 {{ hud.race.gold.toFixed(1) }} · 银牌 {{ hud.race.silver.toFixed(1) }}</span>
        <span v-if="hud.race.best">最好 {{ hud.race.best.toFixed(1) }}</span>
      </div>

      <div class="planet-hud planet-hud-tr">
        <button class="planet-icon" :title="muted ? '打开声音' : '关闭声音'" @click="toggleMute">{{ muted ? '🔇' : '🔊' }}</button>
        <button v-if="canFullscreen" class="planet-icon" title="全屏" @click="toggleFullscreen">⛶</button>
        <RouterLink to="/games" class="planet-icon" title="离开">✕</RouterLink>
      </div>

      <div class="planet-hud planet-hud-bl">
        <div class="planet-islands">
          <i v-for="(a, i) in hud.awake" :key="i" :class="[{ on: a }, `medal-${hud.medals[i] || 0}`]" :style="{ '--c': islandColors[i] }" :title="islandNames[i]"></i>
        </div>
        <span>{{ awakeCount }}/{{ hud.awake.length }} 座岛醒了</span>
      </div>

      <div class="planet-compass-row">
        <div v-if="hud.arrowLabel" class="planet-compass">
          <span class="planet-arrow" :style="{ transform: `rotate(${hud.arrow}rad)` }">▲</span>
          <span>{{ hud.arrowLabel }} · {{ Math.round(hud.arrowDist) }} 米</span>
        </div>
        <div v-if="hud.pollen > 0 && !hud.race" class="planet-compass hive">
          <span class="planet-arrow" :style="{ transform: `rotate(${hud.hiveArrow}rad)` }">▲</span>
          <span>🍯 蜂巢 · {{ Math.round(hud.hiveDist) }} 米</span>
        </div>
      </div>

      <Transition name="planet-fade">
        <div v-if="showKeys && !isTouch" class="planet-hud planet-hud-br">
          <div><kbd>A</kbd><kbd>D</kbd> 转向</div>
          <div><kbd>W</kbd><kbd>S</kbd> 升降</div>
          <div><kbd>空格</kbd> 加速</div>
        </div>
      </Transition>

      <button
        v-if="isTouch"
        class="planet-boost"
        @pointerdown.prevent="setBoost(true)"
        @pointerup.prevent="setBoost(false)"
        @pointercancel="setBoost(false)"
        @pointerleave="setBoost(false)"
      >
        加速
      </button>
      <div v-if="stick.on" class="planet-stick" :style="{ left: `${stick.x}px`, top: `${stick.y}px` }">
        <i :style="{ transform: `translate(${stick.dx}px, ${stick.dy}px)` }"></i>
      </div>
    </template>

    <Transition name="planet-toast">
      <div v-if="toast" :key="toast.id" class="planet-toast" :class="{ big: toast.big }">
        <strong>{{ toast.title }}</strong>
        <span v-if="toast.sub">{{ toast.sub }}</span>
      </div>
    </Transition>

    <!-- ending -->
    <Transition name="planet-fade">
      <section v-if="hud.phase === 'finale'" class="planet-title planet-finale">
        <p class="planet-kicker">六座岛，三十六片花田</p>
        <h1>星球醒了</h1>
        <p class="planet-tagline">用时 {{ clock }} · 采到 {{ stats.pollen }} 粒花粉</p>
        <button class="planet-go" @click="start">再飞一会儿</button>
        <RouterLink to="/games" class="planet-more">看看别的小游戏</RouterLink>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef } from 'vue'
import type { HudState, PlanetGame } from '@/games/planet/game'
import { track } from '@/utils/analytics'

const canvasEl = ref<HTMLCanvasElement | null>(null)
const game = shallowRef<PlanetGame | null>(null)
const ready = ref(false)
const failed = ref(false)
const muted = ref(false)
const showKeys = ref(true)
const isTouch = typeof window !== 'undefined' && window.matchMedia('(pointer: coarse)').matches
const canFullscreen = typeof document !== 'undefined' && !!document.documentElement.requestFullscreen
const islandNames = ref<string[]>([])
const islandColors = ref<string[]>([])

const hud = ref<HudState>({
  phase: 'title',
  region: '',
  regionIndex: 0,
  fields: [],
  awake: [false, false, false, false, false, false],
  pollen: 0,
  honey: 0,
  level: 0,
  levelFrom: 0,
  levelNeed: 20,
  levelName: '',
  arrow: 0,
  arrowDist: 0,
  arrowLabel: '',
  hiveArrow: 0,
  hiveDist: 0,
  race: null,
  medals: [0, 0, 0, 0, 0, 0]
})
const stats = reactive({ seconds: 0, pollen: 0 })
const resume = reactive({ fields: 0, islands: 0, done: false })
const toast = ref<{ id: number; title: string; sub: string; big: boolean } | null>(null)
let toastTimer = 0
let toastId = 0
let keysTimer = 0

const fieldsDone = computed(() => hud.value.fields.filter(Boolean).length)
const awakeCount = computed(() => hud.value.awake.filter(Boolean).length)
const honeyPct = computed(() => {
  const h = hud.value
  const span = h.levelNeed - h.levelFrom
  return span > 0 ? Math.max(0, Math.min(100, ((h.honey - h.levelFrom) / span) * 100)) : 100
})
const clock = computed(() => {
  const m = Math.floor(stats.seconds / 60)
  const s = stats.seconds % 60
  return m ? `${m} 分 ${s} 秒` : `${s} 秒`
})

function showToast(title: string, sub: string, big: boolean): void {
  window.clearTimeout(toastTimer)
  toast.value = { id: ++toastId, title, sub, big }
  toastTimer = window.setTimeout(() => (toast.value = null), big ? 4200 : 2400)
}

function start(): void {
  const g = game.value
  if (!g) return
  const first = hud.value.phase === 'title'
  g.start()
  hud.value = { ...hud.value, phase: 'fly' }
  if (first) {
    track('game_start', { game: 'planet' })
    keysTimer = window.setTimeout(() => (showKeys.value = false), 14000)
  }
}

function restart(): void {
  game.value?.clearSave()
  window.location.reload()
}

function toggleMute(): void {
  muted.value = !muted.value
  game.value?.setMuted(muted.value)
}

function toggleFullscreen(): void {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void document.documentElement.requestFullscreen().catch(() => undefined)
}

function setBoost(on: boolean): void {
  game.value?.setBoost(on)
}

// drag anywhere on the picture: left and right turn, up and down climb and dive
const stick = reactive({ on: false, id: -1, x: 0, y: 0, dx: 0, dy: 0 })
const REACH = 56

function stickDown(e: PointerEvent): void {
  if (hud.value.phase !== 'fly' || stick.on) return
  stick.on = true
  stick.id = e.pointerId
  stick.x = e.clientX
  stick.y = e.clientY
  stick.dx = 0
  stick.dy = 0
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
}

function stickMove(e: PointerEvent): void {
  if (!stick.on || e.pointerId !== stick.id) return
  const dx = e.clientX - stick.x
  const dy = e.clientY - stick.y
  const len = Math.hypot(dx, dy) || 1
  const k = Math.min(1, REACH / len)
  stick.dx = dx * k
  stick.dy = dy * k
  game.value?.setStick(stick.dx / REACH, -stick.dy / REACH)
}

function stickUp(e: PointerEvent): void {
  if (e.pointerId !== stick.id) return
  stick.on = false
  stick.id = -1
  game.value?.setStick(0, 0)
}

function onResize(): void {
  game.value?.resize()
}

onMounted(async () => {
  const canvas = canvasEl.value
  if (!canvas) return
  try {
    const [{ PlanetGame: Game }, { REGIONS }] = await Promise.all([import('@/games/planet/game'), import('@/games/planet/world')])
    islandNames.value = REGIONS.map((r) => r.name)
    islandColors.value = REGIONS.map((r) => r.flowers[0])
    const g = new Game(canvas, {
      hud: (s) => {
        // the page owns the title → fly switch, so a late HUD tick cannot undo it
        hud.value = s
      },
      toast: showToast,
      finale: (s) => {
        stats.seconds = s.seconds
        stats.pollen = s.pollen
        track('game_finish', { game: 'planet', seconds: s.seconds, pollen: s.pollen })
      }
    })
    game.value = g
    Object.assign(resume, g.saved())
    ready.value = true
    if (import.meta.env.DEV) (window as unknown as Record<string, unknown>).__planet = g
  } catch (err) {
    console.error(err)
    failed.value = true
  }
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.clearTimeout(toastTimer)
  window.clearTimeout(keysTimer)
  game.value?.dispose()
  game.value = null
})
</script>

<style scoped>
.planet-page {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: #04050f;
  color: #fff;
  font-family: 'PingFang SC', 'HarmonyOS Sans SC', 'Microsoft YaHei', system-ui, sans-serif;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}
.planet-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  touch-action: none;
}
.planet-back {
  position: absolute;
  top: 18px;
  left: 20px;
  z-index: 3;
  color: rgba(255, 255, 255, 0.72);
  font-size: 14px;
  text-decoration: none;
}
.planet-back:hover {
  color: #fff;
}

.planet-title {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 24px;
  text-align: center;
  pointer-events: none;
  background: radial-gradient(ellipse at center, rgba(4, 5, 15, 0) 30%, rgba(4, 5, 15, 0.55) 100%);
}
.planet-title > * {
  pointer-events: auto;
}
.planet-kicker {
  margin: 0;
  font-size: 13px;
  letter-spacing: 0.42em;
  text-indent: 0.42em;
  color: rgba(255, 233, 160, 0.86);
}
.planet-title h1 {
  margin: 0;
  font-size: clamp(56px, 13vw, 132px);
  font-weight: 900;
  line-height: 1.02;
  letter-spacing: 0.06em;
  text-indent: 0.06em;
  background: linear-gradient(180deg, #fffbe6 8%, #ffd957 52%, #ff9d2e 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  filter: drop-shadow(0 6px 28px rgba(255, 170, 40, 0.45)) drop-shadow(0 2px 0 rgba(120, 60, 0, 0.5));
}
.planet-tagline {
  margin: 0 0 10px;
  font-size: clamp(15px, 2.3vw, 20px);
  color: rgba(255, 255, 255, 0.9);
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.7);
}
.planet-go {
  min-width: 184px;
  padding: 16px 44px;
  border: 0;
  border-radius: 999px;
  font-size: 20px;
  font-weight: 800;
  letter-spacing: 0.3em;
  text-indent: 0.3em;
  color: #3a2300;
  background: linear-gradient(180deg, #ffe680, #ffb52e);
  box-shadow: 0 10px 34px rgba(255, 176, 46, 0.5), inset 0 -4px 0 rgba(180, 96, 0, 0.35);
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.planet-go:hover:not(:disabled) {
  transform: translateY(-2px) scale(1.03);
  box-shadow: 0 14px 44px rgba(255, 176, 46, 0.65), inset 0 -4px 0 rgba(180, 96, 0, 0.35);
}
.planet-go:disabled {
  opacity: 0.6;
  cursor: wait;
  letter-spacing: 0.08em;
  text-indent: 0;
}
.planet-keys {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px 18px;
  margin: 8px 0 0;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.66);
}
.planet-resume {
  margin: 0;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.8);
}
.planet-resume button {
  margin-left: 10px;
  padding: 0;
  border: 0;
  background: none;
  color: #ffe08a;
  font-size: 13px;
  text-decoration: underline;
  cursor: pointer;
}
.planet-failed {
  max-width: 420px;
  font-size: 14px;
  color: #ffb4a8;
}
.planet-more {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.75);
}
kbd {
  display: inline-block;
  min-width: 22px;
  margin: 0 2px;
  padding: 2px 6px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.35);
  background: rgba(255, 255, 255, 0.1);
  font: 600 12px/1.4 ui-monospace, SFMono-Regular, Menlo, monospace;
  color: #fff;
  text-align: center;
}

.planet-hud {
  position: absolute;
  z-index: 2;
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.55);
  pointer-events: none;
}
.planet-hud-tl {
  top: 18px;
  left: 20px;
}
.planet-region {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.08em;
}
.planet-fields {
  display: flex;
  gap: 7px;
  margin-top: 8px;
}
.planet-fields i {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.75);
  transition: background 0.4s ease, transform 0.4s cubic-bezier(0.2, 1.6, 0.4, 1), box-shadow 0.4s ease;
}
.planet-fields i.on {
  background: #ffd957;
  border-color: #fff3bf;
  transform: scale(1.18);
  box-shadow: 0 0 12px rgba(255, 217, 87, 0.9);
}
.planet-pollen {
  margin-top: 10px;
  font-size: 13px;
  color: rgba(255, 240, 190, 0.92);
}
.planet-hud-tr {
  top: 14px;
  right: 16px;
  display: flex;
  gap: 8px;
  pointer-events: auto;
}
.planet-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.22);
  border-radius: 12px;
  background: rgba(10, 14, 34, 0.42);
  backdrop-filter: blur(8px);
  color: #fff;
  font-size: 16px;
  text-decoration: none;
  cursor: pointer;
}
.planet-hud-bl {
  left: 20px;
  bottom: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.86);
}
.planet-islands {
  display: flex;
  gap: 6px;
}
.planet-islands i {
  width: 16px;
  height: 16px;
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.35);
  transition: background 1s ease, box-shadow 1s ease;
}
.planet-islands i.on {
  background: var(--c);
  border-color: #fff;
  box-shadow: 0 0 12px var(--c);
}
.planet-hud-br {
  right: 18px;
  bottom: 18px;
  display: grid;
  gap: 6px;
  font-size: 13px;
  text-align: right;
  color: rgba(255, 255, 255, 0.85);
}
.planet-honey {
  margin-top: 8px;
  display: grid;
  gap: 4px;
  font-size: 13px;
  color: #ffe6a8;
}
.planet-honey i {
  display: block;
  width: 128px;
  height: 6px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.22);
  overflow: hidden;
}
.planet-honey b {
  display: block;
  height: 100%;
  border-radius: 3px;
  background: linear-gradient(90deg, #ffd957, #ff9d2e);
  transition: width 0.5s ease;
}
.planet-honey em {
  font-style: normal;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.72);
}
.planet-race {
  position: absolute;
  top: 16px;
  left: 50%;
  z-index: 2;
  transform: translateX(-50%);
  display: flex;
  align-items: baseline;
  gap: 14px;
  padding: 8px 20px;
  border-radius: 999px;
  border: 1px solid rgba(110, 243, 255, 0.5);
  background: rgba(10, 14, 34, 0.5);
  backdrop-filter: blur(8px);
  font-size: 13px;
  white-space: nowrap;
  pointer-events: none;
}
.planet-race strong {
  font: 800 26px/1 ui-monospace, SFMono-Regular, Menlo, monospace;
  color: #6ef3ff;
  min-width: 78px;
  text-align: right;
}
.planet-race small {
  margin-left: 3px;
  font-size: 12px;
  font-weight: 500;
}
.planet-islands i.medal-1 {
  outline: 2px solid #c98a52;
  outline-offset: 1px;
}
.planet-islands i.medal-2 {
  outline: 2px solid #d9e2ec;
  outline-offset: 1px;
}
.planet-islands i.medal-3 {
  outline: 2px solid #ffd23f;
  outline-offset: 1px;
}
.planet-compass-row {
  position: absolute;
  left: 50%;
  bottom: 22px;
  z-index: 2;
  transform: translateX(-50%);
  display: flex;
  gap: 10px;
  pointer-events: none;
}
.planet-compass.hive {
  border-color: rgba(255, 201, 87, 0.55);
}
.planet-compass {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px 8px 12px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(10, 14, 34, 0.45);
  backdrop-filter: blur(8px);
  font-size: 14px;
  white-space: nowrap;
  pointer-events: none;
}
.planet-arrow {
  display: inline-block;
  color: #ffd957;
  font-size: 16px;
  line-height: 1;
  transition: transform 0.12s linear;
  filter: drop-shadow(0 0 6px rgba(255, 217, 87, 0.8));
}
.planet-boost {
  position: absolute;
  right: 22px;
  bottom: 84px;
  z-index: 3;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  border: 2px solid rgba(255, 233, 160, 0.8);
  background: rgba(255, 190, 60, 0.28);
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  backdrop-filter: blur(6px);
  touch-action: none;
}
.planet-boost:active {
  background: rgba(255, 190, 60, 0.6);
  transform: scale(0.95);
}
.planet-stick {
  position: absolute;
  z-index: 2;
  width: 112px;
  height: 112px;
  margin: -56px 0 0 -56px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.06);
  pointer-events: none;
}
.planet-stick i {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 44px;
  height: 44px;
  margin: -22px 0 0 -22px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.55);
}

.planet-toast {
  position: absolute;
  top: 74px;
  left: 50%;
  z-index: 3;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 10px 22px;
  border-radius: 999px;
  background: rgba(10, 14, 34, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(8px);
  font-size: 15px;
  white-space: nowrap;
  pointer-events: none;
}
.planet-toast span {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.75);
}
.planet-toast.big {
  top: 26%;
  padding: 0;
  border: 0;
  background: none;
  backdrop-filter: none;
  gap: 10px;
}
.planet-toast.big strong {
  font-size: clamp(40px, 8vw, 84px);
  font-weight: 900;
  letter-spacing: 0.1em;
  text-indent: 0.1em;
  background: linear-gradient(180deg, #fffbe6, #ffd957 60%, #ff9d2e);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  filter: drop-shadow(0 4px 22px rgba(255, 170, 40, 0.6));
}
.planet-toast.big span {
  font-size: 16px;
  color: #fff;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.7);
}

.planet-fade-enter-active,
.planet-fade-leave-active {
  transition: opacity 0.7s ease;
}
.planet-fade-enter-from,
.planet-fade-leave-to {
  opacity: 0;
}
.planet-toast-enter-active {
  transition: opacity 0.35s ease, transform 0.5s cubic-bezier(0.2, 1.5, 0.4, 1);
}
.planet-toast-leave-active {
  transition: opacity 0.5s ease;
}
.planet-toast-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(14px) scale(0.92);
}
.planet-toast-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .planet-region {
    font-size: 18px;
  }
  .planet-compass-row {
    bottom: 64px;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
  .planet-race {
    top: 62px;
    gap: 8px;
    padding: 6px 14px;
  }
  .planet-race span:nth-of-type(2) {
    display: none;
  }
  .planet-hud-bl span {
    display: none;
  }
}
</style>
