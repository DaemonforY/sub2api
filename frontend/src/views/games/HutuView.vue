<template>
  <div ref="pageEl" class="hutu-page" :class="{ touch: isTouch, playing: screen === 'game' }">
    <div class="hutu-stage" :style="stageStyle">
      <canvas ref="canvasEl" class="hutu-canvas" data-testid="hutu-canvas"></canvas>

      <!-- level card -->
      <Transition name="hutu-fade">
        <div v-if="screen === 'game' && hud && hud.phase === 'intro'" class="hutu-card">
          <p class="hutu-card-world">第{{ hud.levelId.startsWith('1') ? '一' : '二' }}世界 · {{ hud.worldName }}</p>
          <h2>{{ hud.levelId }} {{ hud.levelName }}</h2>
          <p class="hutu-card-lives">剩余 <b>{{ hud.lives }}</b> 条命{{ hud.mode === 'solo' ? '' : '（两人共用）' }}</p>
        </div>
      </Transition>

      <div v-if="screen === 'game' && hud && hud.phase === 'clear'" class="hutu-banner">过关！</div>

      <!-- pause -->
      <div v-if="screen === 'game' && hud && hud.paused" class="hutu-modal">
        <h2>暂停</h2>
        <button class="hutu-btn" @click="game?.togglePause()">继续</button>
        <button class="hutu-btn ghost" @click="restartLevel">重来这一关</button>
        <button class="hutu-btn ghost" @click="toTitle">回到标题</button>
      </div>

      <!-- game over -->
      <div v-if="screen === 'game' && hud && hud.phase === 'over'" class="hutu-modal">
        <h2>游戏结束</h2>
        <p>{{ scoreLine }}</p>
        <GameLeaderboard game="hutu" :score="teamScore" :detail="runDetail" :describe="describeRun" :limit="3" class="hutu-board" />
        <template v-if="hud.mode !== 'guest'">
          <button class="hutu-btn" data-testid="hutu-continue" @click="continueGame">从 {{ hud.levelId }} 接着玩（分数减半）</button>
        </template>
        <p v-else class="hutu-dim">等房主选择是否接着玩…</p>
        <button class="hutu-btn ghost" @click="toTitle">回到标题</button>
      </div>

      <!-- the end -->
      <div v-if="screen === 'game' && hud && hud.phase === 'won'" class="hutu-modal hutu-won">
        <p class="hutu-kicker">年灯找回来了</p>
        <h2>元宵灯会，照常开张！</h2>
        <p>{{ scoreLine }}</p>
        <GameLeaderboard game="hutu" :score="teamScore" :detail="runDetail" :describe="describeRun" :limit="3" class="hutu-board" />
        <button class="hutu-btn" @click="toTitle">回到标题</button>
        <RouterLink to="/games" class="hutu-link">看看别的小游戏</RouterLink>
      </div>

      <!-- partner left (online) -->
      <div v-if="screen === 'game' && net.state === 'error' && hud" class="hutu-modal">
        <h2>{{ errorText }}</h2>
        <button class="hutu-btn" @click="toTitle">回到标题</button>
      </div>
      <div v-else-if="screen === 'game' && hud && hud.mode === 'host' && !net.peer" class="hutu-toast">队友掉线了，等 TA 用同一个房间码回来（{{ net.code }}）</div>
    </div>

    <!-- title -->
    <section v-if="screen === 'title'" class="hutu-title" data-testid="hutu-title">
      <img class="hutu-cover" :src="COVER" alt="" />
      <div class="hutu-title-inner">
        <p class="hutu-kicker">HiveGPT 小游戏 · 横版闯关</p>
        <h1>虎兔大冒险</h1>
        <p class="hutu-tagline">年兽的小喽啰偷走了村里的年灯。戴上虎头帽和兔耳帽，去把灯找回来！</p>
        <div class="hutu-menu">
          <button class="hutu-btn" data-testid="hutu-solo" @click="startLocal(1)">单人开始</button>
          <button class="hutu-btn" data-testid="hutu-local" @click="startLocal(2)">同屏双人</button>
          <button class="hutu-btn alt" data-testid="hutu-host" @click="openRoom()">在线双排 · 开房间</button>
          <form class="hutu-join" @submit.prevent="joinRoom()">
            <input v-model="joinCode" maxlength="8" placeholder="房间码" aria-label="房间码" data-testid="hutu-code" @input="joinCode = joinCode.toUpperCase()" />
            <button class="hutu-btn small" type="submit" :disabled="joinCode.trim().length < 3">加入</button>
          </form>
        </div>
        <div v-if="saved.reached > 0" class="hutu-levels">
          <span>选关：</span>
          <button v-for="(l, i) in levels.slice(0, saved.reached + 1)" :key="l.id" class="hutu-chip" @click="startLocal(1, i)">{{ l.id }}</button>
          <span class="hutu-dim">（单人）</span>
        </div>
        <p v-if="saved.best" class="hutu-dim">最高分 {{ saved.best }}</p>
        <GameAccountBar game="hutu" has-save />
        <div class="hutu-keys">
          <p><b>P1 虎妞</b> A D 移动 · S 蹲 · J 跳 · K 跑 / 喷火</p>
          <p><b>P2 兔宝</b> ← → 移动 · ↓ 蹲 · 小键盘 1 跳 · 2 跑 / 喷火（笔记本用 . 和 /）</p>
          <p>单人时两套键都能用，空格也能跳；手柄插上就能用。按住跳得更高，跑起来跳得更远。Esc 暂停，M 静音。</p>
        </div>
        <RouterLink to="/games" class="hutu-link">← 小游戏</RouterLink>
      </div>
    </section>

    <!-- online lobby -->
    <section v-if="screen === 'lobby'" class="hutu-title hutu-lobby" data-testid="hutu-lobby">
      <div class="hutu-title-inner">
        <template v-if="net.state === 'connecting'">
          <h2>正在连接房间…</h2>
        </template>
        <template v-else-if="net.state === 'waiting'">
          <p class="hutu-kicker">房间已开好，你是 P1 虎妞</p>
          <div class="hutu-code" data-testid="hutu-room-code">{{ net.code }}</div>
          <p>把房间码或下面的链接发给朋友，TA 打开就能进来当 P2 兔宝。</p>
          <div class="hutu-share">
            <input :value="shareLink" readonly aria-label="邀请链接" @focus="($event.target as HTMLInputElement).select()" />
            <button class="hutu-btn small" @click="copyLink">{{ copied ? '已复制' : '复制链接' }}</button>
          </div>
          <p class="hutu-dim">等待队友加入…（关掉这个页面房间就没了）</p>
        </template>
        <template v-else-if="net.state === 'error'">
          <h2>{{ errorText }}</h2>
        </template>
        <button class="hutu-btn ghost" @click="toTitle">返回</button>
      </div>
    </section>

    <!-- touch controls -->
    <div v-if="isTouch && screen === 'game'" class="hutu-touch">
      <div class="hutu-pad">
        <button v-for="b in padButtons" :key="b.k" :class="['hutu-tb', b.k]" @pointerdown.prevent="press(b.bit, true, $event)" @pointerup.prevent="press(b.bit, false, $event)" @pointercancel="press(b.bit, false, $event)" @pointerleave="press(b.bit, false, $event)">{{ b.label }}</button>
      </div>
      <div class="hutu-ab">
        <button class="hutu-tb fire" @pointerdown.prevent="press(32, true, $event)" @pointerup.prevent="press(32, false, $event)" @pointercancel="press(32, false, $event)" @pointerleave="press(32, false, $event)">跑</button>
        <button class="hutu-tb jump" @pointerdown.prevent="press(16, true, $event)" @pointerup.prevent="press(16, false, $event)" @pointercancel="press(16, false, $event)" @pointerleave="press(16, false, $event)">跳</button>
      </div>
    </div>

    <div v-if="screen === 'game'" class="hutu-corner">
      <span v-if="hud && hud.mode === 'host'" class="hutu-roomtag">房间 {{ net.code }}</span>
      <span v-if="hud && hud.mode === 'guest'" class="hutu-roomtag">已加入 {{ net.code }}</span>
      <button class="hutu-icon" :title="muted ? '打开声音' : '静音'" @click="toggleMute">{{ muted ? '🔇' : '🔊' }}</button>
      <button v-if="canFullscreen" class="hutu-icon" title="全屏" @click="toggleFullscreen">⛶</button>
      <button v-if="hud && (hud.mode === 'solo' || hud.mode === 'local')" class="hutu-icon" title="暂停" @click="game?.togglePause()">⏸</button>
      <button v-else class="hutu-icon" title="离开" @click="toTitle">✕</button>
    </div>
  </div>
</template>

<script setup lang="ts">
// 虎兔大冒险 (/games/hutu): an original two-player co-op platformer. The game itself lives in
// src/games/hutu; this view is the title screen, online lobby, overlays and touch buttons.
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { HutuGame, Hud, NetStatus } from '@/games/hutu/game'
import { LEVELS } from '@/games/hutu/levels'
import { track } from '@/utils/analytics'
import GameAccountBar from '@/components/games/GameAccountBar.vue'
import GameLeaderboard from '@/components/games/GameLeaderboard.vue'
import { useGameAccount } from '@/composables/useGameAccount'

const route = useRoute()
const router = useRouter()
const canvasEl = ref<HTMLCanvasElement | null>(null)
const pageEl = ref<HTMLElement | null>(null)
const game = shallowRef<HutuGame | null>(null)
const hud = ref<Hud | null>(null)
const net = reactive<NetStatus>({ state: 'idle', code: '', seat: -1, peer: false, error: '' })
const screen = ref<'title' | 'lobby' | 'game'>('title')
const joinCode = ref('')
const muted = ref(false)
const copied = ref(false)
const levels = LEVELS
const COVER = '/games-media/hutu/title.webp'
const saved = reactive({ best: 0, reached: 0 })
const isTouch = typeof window !== 'undefined' && !!window.matchMedia && window.matchMedia('(pointer: coarse)').matches
const canFullscreen = typeof document !== 'undefined' && !!document.documentElement.requestFullscreen
const view = reactive({ w: 1280, h: 720 })

const padButtons = [
  { k: 'left', bit: 1, label: '◀' },
  { k: 'down', bit: 8, label: '▼' },
  { k: 'right', bit: 2, label: '▶' }
]

const stageStyle = computed(() => {
  // fit 400×224, using whole-pixel scales when there is room so the pixel art stays even
  let s = Math.min(view.w / 400, view.h / 224)
  if (s >= 2) s = Math.floor(s * 2) / 2
  return { width: `${Math.floor(400 * s)}px`, height: `${Math.floor(224 * s)}px` }
})

const shareLink = computed(() => `${location.origin}/games/hutu?room=${net.code}`)

const scoreLine = computed(() => {
  const h = hud.value
  if (!h) return ''
  return h.scores.length > 1 ? `虎妞 ${h.scores[0]} 分 · 兔宝 ${h.scores[1]} 分` : `得分 ${h.scores[0]}`
})

// 小游戏可选登录: the team's score goes on the site-wide board; progress (best, furthest level) syncs.
const account = useGameAccount('hutu')
const teamScore = computed(() => (hud.value ? hud.value.scores.reduce((n, x) => n + Math.floor(x), 0) : 0))
const runDetail = computed(() => ({ mode: hud.value?.mode ?? 'solo', level: hud.value?.levelId ?? '' }))
function describeRun(d: Record<string, unknown>): string {
  const mode = d.mode === 'solo' ? '单人' : d.mode === 'local' ? '同屏双人' : '在线双排'
  return d.level ? `${mode} · ${String(d.level)}` : mode
}
async function syncProgress(): Promise<void> {
  const merged = await account.syncSave<{ best: number; reached: number }>('hutu:v1', (a, b) => ({
    best: Math.max(Number(a?.best) || 0, Number(b?.best) || 0),
    reached: Math.max(Number(a?.reached) || 0, Number(b?.reached) || 0)
  }))
  if (merged) {
    const { loadSave } = await import('@/games/hutu/game')
    Object.assign(saved, loadSave())
  }
}
watch(
  () => hud.value?.phase,
  (p) => {
    if (p === 'clear' || p === 'over' || p === 'won') void syncProgress()
  }
)

const errorText = computed(() => {
  switch (net.error) {
    case 'not_found':
      return '没有这个房间（房间码不对，或者房主已经离开）'
    case 'room_full':
      return '房间已经满了（两个人正在玩）'
    case 'rate':
      return '操作太频繁了，稍等几秒再试'
    case 'host_left':
      return '房主离开了，这一局结束了'
    case 'closed':
      return net.seat === 1 ? '房主离开了，这一局结束了' : '连接断开了'
    case 'bad_game':
    case 'too_big':
      return '房间出了点问题，换个房间再试试'
    default:
      return '连不上房间服务器，检查一下网络再试'
  }
})

function onResize(): void {
  view.w = window.innerWidth
  view.h = window.innerHeight - (isTouch ? 0 : 0)
}

async function ensureGame(): Promise<HutuGame | null> {
  if (game.value) return game.value
  if (!canvasEl.value) return null
  const { HutuGame: G, loadSave } = await import('@/games/hutu/game')
  Object.assign(saved, loadSave())
  const g = new G(
    canvasEl.value,
    (h) => (hud.value = h),
    (n) => Object.assign(net, n)
  )
  g.setMuted(muted.value)
  game.value = g
  return g
}

async function startLocal(players: 1 | 2, level = 0): Promise<void> {
  const g = await ensureGame()
  if (!g) return
  g.startLocal(players, level)
  screen.value = 'game'
  track('game_start', { game: 'hutu', mode: players === 2 ? 'local' : 'solo', level })
}

async function openRoom(): Promise<void> {
  const g = await ensureGame()
  if (!g) return
  screen.value = 'lobby'
  g.openRoom()
  track('game_start', { game: 'hutu', mode: 'host' })
}

async function joinRoom(code?: string): Promise<void> {
  const c = (code || joinCode.value).trim().toUpperCase()
  if (!c) return
  const g = await ensureGame()
  if (!g) return
  screen.value = 'lobby'
  g.openRoom(c)
  track('game_start', { game: 'hutu', mode: 'guest' })
}

function continueGame(): void {
  game.value?.continueGame()
}

function restartLevel(): void {
  const g = game.value
  if (!g?.world) return
  g.world.paused = false
  g.world.restartFrom(g.world.levelIdx)
}

function toTitle(): void {
  game.value?.leaveRoom()
  screen.value = 'title'
  hud.value = null
  if (route.query.room) void router.replace({ path: '/games/hutu' })
  void import('@/games/hutu/game').then(({ loadSave }) => Object.assign(saved, loadSave()))
}

function toggleMute(): void {
  muted.value = !muted.value
  game.value?.setMuted(muted.value)
}

function toggleFullscreen(): void {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void (pageEl.value || document.documentElement).requestFullscreen().catch(() => undefined)
}

async function copyLink(): Promise<void> {
  try {
    await navigator.clipboard.writeText(shareLink.value)
    copied.value = true
    window.setTimeout(() => (copied.value = false), 2000)
  } catch {
    /* the field is selectable as a fallback */
  }
}

const touchBits = new Map<number, number>()
function press(bit: number, on: boolean, e: PointerEvent): void {
  const g = game.value
  if (!g) return
  if (on) {
    touchBits.set(e.pointerId, (touchBits.get(e.pointerId) || 0) | bit)
    ;(e.target as HTMLElement).setPointerCapture?.(e.pointerId)
  } else touchBits.set(e.pointerId, (touchBits.get(e.pointerId) || 0) & ~bit)
  let b = 0
  for (const v of touchBits.values()) b |= v
  g.input.touch = b
}

// the online lobby turns into the game as soon as both players are in
const stopWatch = window.setInterval(() => {
  if (screen.value === 'lobby' && net.state === 'playing' && hud.value) screen.value = 'game'
}, 200)

onMounted(async () => {
  onResize()
  window.addEventListener('resize', onResize)
  const { loadSave } = await import('@/games/hutu/game')
  Object.assign(saved, loadSave())
  void syncProgress()
  const room = typeof route.query.room === 'string' ? route.query.room : ''
  if (room) {
    joinCode.value = room.toUpperCase()
    void joinRoom(room)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.clearInterval(stopWatch)
  game.value?.destroy()
})
</script>

<style scoped>
.hutu-page {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: radial-gradient(ellipse at center, #2a1c34 0%, #120a18 70%);
  color: #fff8ec;
  font-family: 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}
.hutu-stage {
  position: relative;
  box-shadow: 0 0 0 2px #2b1b2e, 0 18px 60px rgba(0, 0, 0, 0.5);
}
.hutu-canvas {
  display: block;
  width: 100%;
  height: 100%;
  image-rendering: pixelated;
  image-rendering: crisp-edges;
  background: #120a18;
}
.hutu-title {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow-y: auto;
  background: #120a18;
}
.hutu-cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.9;
}
.hutu-title-inner {
  position: relative;
  max-width: 640px;
  margin: auto;
  padding: 24px 22px;
  border: 2px solid rgba(43, 27, 46, 0.9);
  border-radius: 14px;
  background: rgba(18, 10, 24, 0.7);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
  text-align: center;
  text-shadow: 0 2px 0 #2b1b2e;
}
.hutu-title h1 {
  margin: 4px 0 8px;
  font-size: clamp(40px, 9vw, 76px);
  font-weight: 900;
  letter-spacing: 0.08em;
  color: #ffd23f;
  -webkit-text-stroke: 2px #2b1b2e;
  text-shadow: 0 5px 0 #b02a30, 0 8px 0 #2b1b2e;
}
.hutu-title h2,
.hutu-modal h2 {
  margin: 0 0 12px;
  font-size: 24px;
  font-weight: 800;
}
.hutu-kicker {
  font-size: 13px;
  letter-spacing: 0.2em;
  color: #ffc98a;
}
.hutu-tagline {
  margin: 0 auto 18px;
  max-width: 460px;
  font-size: 15px;
  line-height: 1.6;
}
.hutu-menu {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}
.hutu-btn {
  min-width: 150px;
  padding: 11px 18px;
  border: 2px solid #2b1b2e;
  border-radius: 6px;
  background: #e8443a;
  box-shadow: 0 4px 0 #2b1b2e;
  color: #fff;
  font-size: 16px;
  font-weight: 800;
  cursor: pointer;
  transition: transform 0.08s;
}
.hutu-btn:hover {
  transform: translateY(-1px);
  background: #ff5a4a;
}
.hutu-btn:active {
  transform: translateY(3px);
  box-shadow: 0 1px 0 #2b1b2e;
}
.hutu-btn.alt {
  background: #3aa0e8;
}
.hutu-btn.ghost {
  background: rgba(255, 255, 255, 0.12);
  margin-top: 6px;
}
.hutu-btn.small {
  min-width: 0;
  padding: 8px 14px;
  font-size: 14px;
}
.hutu-btn:disabled {
  opacity: 0.5;
  cursor: default;
}
.hutu-join {
  display: flex;
  gap: 6px;
}
.hutu-join input,
.hutu-share input {
  width: 110px;
  padding: 8px 10px;
  border: 2px solid #2b1b2e;
  border-radius: 6px;
  background: #fff8ec;
  color: #2b1b2e;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.15em;
  text-align: center;
}
.hutu-share {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin: 12px 0;
}
.hutu-share input {
  width: min(340px, 60vw);
  letter-spacing: 0;
  font-size: 13px;
}
.hutu-levels {
  margin-top: 14px;
  font-size: 14px;
}
.hutu-chip {
  margin: 0 3px;
  padding: 3px 9px;
  border: 2px solid #2b1b2e;
  border-radius: 4px;
  background: #ffd23f;
  color: #2b1b2e;
  font-weight: 800;
  cursor: pointer;
}
.hutu-keys {
  margin: 16px auto 8px;
  max-width: 560px;
  font-size: 12.5px;
  line-height: 1.7;
  color: #e6d8c8;
}
.hutu-dim {
  font-size: 13px;
  color: #c9b8a8;
}
.hutu-link {
  display: inline-block;
  margin-top: 10px;
  font-size: 14px;
  color: #ffc98a;
  text-decoration: underline;
}
.hutu-code {
  margin: 10px 0;
  font-size: 56px;
  font-weight: 900;
  letter-spacing: 0.2em;
  color: #ffd23f;
}
.hutu-card,
.hutu-modal {
  position: absolute;
  inset: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: safe center;
  gap: 6px;
  text-align: center;
  text-shadow: 0 2px 0 #2b1b2e;
}
.hutu-card {
  background: #120a18;
}
.hutu-board {
  width: min(360px, 92%);
  text-shadow: none;
}
.hutu-card h2 {
  font-size: clamp(24px, 5vw, 44px);
  font-weight: 900;
  color: #ffd23f;
}
.hutu-card-world {
  color: #ffc98a;
  letter-spacing: 0.2em;
}
.hutu-modal {
  background: rgba(18, 10, 24, 0.78);
}
.hutu-won h2 {
  color: #ffd23f;
  font-size: 30px;
}
.hutu-banner {
  position: absolute;
  left: 50%;
  top: 28%;
  transform: translateX(-50%);
  font-size: clamp(28px, 6vw, 56px);
  font-weight: 900;
  color: #ffd23f;
  -webkit-text-stroke: 2px #2b1b2e;
  text-shadow: 0 4px 0 #b02a30;
  pointer-events: none;
}
.hutu-toast {
  position: absolute;
  left: 50%;
  bottom: 12px;
  transform: translateX(-50%);
  padding: 6px 12px;
  border-radius: 6px;
  background: rgba(18, 10, 24, 0.8);
  font-size: 13px;
}
.hutu-corner {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.touch .hutu-corner {
  top: 34px;
  opacity: 0.75;
}
.hutu-icon {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 16px;
}
.hutu-roomtag {
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.35);
  font-size: 12px;
}
.hutu-touch {
  position: absolute;
  inset: auto 0 0 0;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  padding: 0 14px 14px;
  pointer-events: none;
}
.hutu-pad,
.hutu-ab {
  display: flex;
  gap: 10px;
  pointer-events: auto;
}
.hutu-tb {
  width: 64px;
  height: 64px;
  border: 2px solid rgba(255, 255, 255, 0.5);
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  font-size: 20px;
  font-weight: 800;
  touch-action: none;
}
.hutu-tb.down {
  align-self: flex-end;
  width: 52px;
  height: 52px;
}
.hutu-tb.jump {
  width: 76px;
  height: 76px;
  background: rgba(232, 68, 58, 0.45);
}
.hutu-tb.fire {
  margin-bottom: 28px;
  background: rgba(58, 160, 232, 0.4);
}
.hutu-fade-enter-active,
.hutu-fade-leave-active {
  transition: opacity 0.35s;
}
.hutu-fade-enter-from,
.hutu-fade-leave-to {
  opacity: 0;
}
</style>
