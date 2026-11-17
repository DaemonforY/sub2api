<template>
  <div class="isl-page" :class="{ touch: isTouch, playing: screen === 'play' }">
    <canvas ref="canvasEl" class="isl-canvas" :class="{ hidden: screen !== 'play' }"></canvas>

    <!-- title -->
    <section v-if="screen === 'title'" class="isl-title" data-testid="island-title">
      <div class="isl-art" :style="{ backgroundImage: `url(${KEYART})` }"></div>
      <div class="isl-title-inner">
        <div class="isl-title-top">
          <p class="isl-kicker">HiveGPT 小游戏 · 双人合作</p>
          <h1 class="isl-logo"><span>椰风岛</span><b>大冒险</b></h1>
        </div>
        <div class="isl-title-bottom">
          <p class="isl-tag">火山里的熔岩龟王抢走了岛上的太阳果。阿浪和小椰一路吃水果、敲大贝壳、扔回旋镖，去把它抢回来。</p>
          <div class="isl-menu">
            <button class="isl-btn primary" :disabled="!ready" data-testid="island-solo" @click="startSolo">
              {{ ready ? '单人开始' : '正在上岛…' }}
            </button>
            <button class="isl-btn" :disabled="!ready" data-testid="island-local2" @click="startLocal2">同屏双人</button>
            <button class="isl-btn online" :disabled="!ready" data-testid="island-online" @click="openLobby">联机双排</button>
          </div>
          <GameAccountBar game="island" />
          <div v-if="!isTouch" class="isl-keys">
            <div>
              <b>P1 阿浪</b><span><kbd>A</kbd><kbd>D</kbd> 跑</span><span><kbd>W</kbd>/<kbd>K</kbd> 跳</span><span><kbd>J</kbd> 扔</span>
            </div>
            <div>
              <b>P2 小椰</b><span><kbd>←</kbd><kbd>→</kbd> 跑</span><span><kbd>↑</kbd> 跳</span><span><kbd>.</kbd>/<kbd>右Shift</kbd> 扔</span>
            </div>
            <div class="isl-keys-note">单人或联机时方向键、WASD 都能用 · 手柄即插即玩</div>
          </div>
          <div v-else class="isl-keys">
            <div class="isl-keys-note">横过手机玩 · 左边按钮跑，右边按钮跳和扔</div>
          </div>
          <ul class="isl-how">
            <li>🍌 体力条一直在掉，吃水果补回来</li>
            <li>🐚 敲开大贝壳：回旋镖、冲浪板、小星灵……也可能是倒霉云</li>
            <li>🫧 双人时队友倒下会变成泡泡，碰一下就能救回来</li>
          </ul>
          <p class="isl-ai">背景与封面插画由 AI 生成</p>
        </div>
        <RouterLink to="/games" class="isl-back">← 小游戏</RouterLink>
      </div>
    </section>

    <!-- online lobby -->
    <section v-if="screen === 'lobby'" class="isl-panel-wrap" data-testid="island-lobby">
      <div class="isl-panel">
        <h2>联机双排</h2>
        <template v-if="!net.role">
          <p class="isl-panel-sub">一人建房间，把房间码或链接发给好友，好友加入后房主点开始。</p>
          <button class="isl-btn primary wide" data-testid="island-host" @click="host">创建房间（我当阿浪）</button>
          <div class="isl-join">
            <input v-model="joinCode" maxlength="8" placeholder="输入房间码" data-testid="island-code-input" @keydown.enter="join(joinCode)" />
            <button class="isl-btn" :disabled="joinCode.trim().length < 3" @click="join(joinCode)">加入（我当小椰）</button>
          </div>
        </template>
        <template v-else-if="net.role === 'host'">
          <p v-if="net.status === 'connecting'" class="isl-panel-sub">正在创建房间…</p>
          <template v-else-if="net.code">
            <p class="isl-panel-sub">房间码</p>
            <div class="isl-code" data-testid="island-room-code">
              {{ net.code }}
            </div>
            <div class="isl-share">
              <input :value="shareLink" readonly @focus="($event.target as HTMLInputElement).select()" />
              <button class="isl-btn" @click="copyLink">
                {{ copied ? '已复制' : '复制邀请链接' }}
              </button>
            </div>
            <p class="isl-status" :class="{ ok: net.status === 'joined' }">
              {{ net.status === 'joined' ? '✅ 小椰已上岛！' : net.status === 'peer-left' ? '好友离开了，等待重新加入…' : '⏳ 等好友加入…' }}
            </p>
            <button class="isl-btn primary wide" :disabled="net.status !== 'joined'" data-testid="island-begin" @click="beginOnline">开始冒险</button>
          </template>
        </template>
        <template v-else>
          <p class="isl-panel-sub">房间 {{ net.code }}</p>
          <p class="isl-status" :class="{ ok: net.status === 'waiting' }">
            {{ net.status === 'connecting' ? '正在加入…' : net.status === 'host-left' ? '房主离开了，这个房间结束了。' : '✅ 已加入，等房主开始…' }}
          </p>
        </template>
        <p v-if="net.error" class="isl-error">{{ net.error }}</p>
        <button class="isl-link" @click="backToTitle">← 返回</button>
      </div>
    </section>

    <!-- guest arriving from an invite link: one tap to join (also unlocks sound) -->
    <section v-if="screen === 'invite'" class="isl-panel-wrap">
      <div class="isl-panel">
        <h2>好友邀请你一起玩</h2>
        <p class="isl-panel-sub">《椰风岛大冒险》双人合作，房间 {{ inviteCode }}</p>
        <button class="isl-btn primary wide" :disabled="!ready" data-testid="island-accept" @click="acceptInvite">
          {{ ready ? '加入房间' : '正在上岛…' }}
        </button>
        <button class="isl-link" @click="backToTitle">不了，自己玩</button>
      </div>
    </section>

    <!-- in-game -->
    <template v-if="screen === 'play'">
      <div class="isl-tools">
        <button class="isl-icon" :title="muted ? '打开声音' : '关闭声音'" @click="toggleMute">
          {{ muted ? '🔇' : '🔊' }}
        </button>
        <button v-if="canFullscreen" class="isl-icon" title="全屏" @click="toggleFullscreen">⛶</button>
        <button v-if="!online" class="isl-icon" title="暂停 (Esc)" @click="togglePause">⏸</button>
        <button class="isl-icon" title="离开" @click="leave">✕</button>
      </div>
      <div v-if="online && net.code" class="isl-room-tag">房间 {{ net.code }}<template v-if="net.status === 'peer-left'"> · 好友掉线了，等他回来</template></div>

      <Transition name="isl-banner">
        <div v-if="banner" :key="banner.id" class="isl-banner" :class="{ boss: banner.kicker === '首领出现' }">
          <small>{{ banner.kicker }}</small>
          <strong>{{ banner.title }}</strong>
        </div>
      </Transition>

      <div v-if="paused" class="isl-panel-wrap dim">
        <div class="isl-panel">
          <h2>暂停</h2>
          <button class="isl-btn primary wide" @click="togglePause">继续</button>
          <button class="isl-btn wide" @click="leave">回到标题</button>
        </div>
      </div>

      <div v-if="phase === 'clear'" class="isl-panel-wrap clear">
        <div class="isl-panel tally">
          <p class="isl-kicker">太阳果碎片 {{ info.stage + 1 }}/3</p>
          <h2>{{ stageNames[info.stage] }} 通关！</h2>
          <div class="isl-scores">
            <div v-for="(s, i) in info.scores.slice(0, info.twoP ? 2 : 1)" :key="i" :class="`p${i + 1}`">
              <span>{{ heroNames[i] }}</span>
              <b>{{ s.toLocaleString() }}</b>
            </div>
          </div>
          <p class="isl-panel-sub">剩下的体力换成分数…</p>
        </div>
      </div>

      <div v-if="phase === 'over'" class="isl-panel-wrap dim">
        <div class="isl-panel">
          <h2>冒险失败</h2>
          <p class="isl-panel-sub">倒在了「{{ stageNames[info.stage] }}」。从上一个旗子接着来？（分数减半）</p>
          <GameLeaderboard game="island" :score="teamScore" :detail="runDetail" :describe="describeRun" :limit="3" />
          <GameResultCardButton :build="islandCard" btn-class="isl-btn wide" />
          <button v-if="mode !== 'guest'" class="isl-btn primary wide" data-testid="island-continue" @click="doContinue">继续冒险</button>
          <p v-else class="isl-status">等房主决定要不要继续…</p>
          <button class="isl-btn wide" @click="leave">回到标题</button>
        </div>
      </div>

      <div v-if="phase === 'ending'" class="isl-panel-wrap ending">
        <div class="isl-panel tally">
          <p class="isl-kicker">三块太阳果碎片合在了一起</p>
          <h2>椰风岛又亮起来了！</h2>
          <p class="isl-panel-sub">果树重新结满了果子。谢谢你，{{ info.twoP ? '阿浪和小椰' : '阿浪' }}。</p>
          <div class="isl-scores">
            <div v-for="(s, i) in info.scores.slice(0, info.twoP ? 2 : 1)" :key="i" :class="`p${i + 1}`">
              <span>{{ heroNames[i] }}</span>
              <b>{{ s.toLocaleString() }}</b>
            </div>
          </div>
          <GameLeaderboard game="island" :score="teamScore" :detail="runDetail" :describe="describeRun" :limit="5" />
          <GameResultCardButton :build="islandCard" btn-class="isl-btn wide" />
          <button class="isl-btn primary wide" @click="leave">再玩一次</button>
          <RouterLink to="/games" class="isl-link">看看别的小游戏</RouterLink>
        </div>
      </div>

      <div v-if="net.status === 'host-left' && mode === 'guest'" class="isl-panel-wrap dim">
        <div class="isl-panel">
          <h2>房主离开了</h2>
          <p class="isl-panel-sub">这局结束了，可以自己建一个房间再邀请好友。</p>
          <button class="isl-btn primary wide" @click="leave">回到标题</button>
        </div>
      </div>

      <!-- touch controls -->
      <template v-if="isTouch">
        <div class="isl-pad" @pointerdown.prevent="padDown" @pointermove.prevent="padMove" @pointerup.prevent="padUp" @pointercancel="padUp" @pointerleave="padUp">
          <span :class="{ on: touchBits & 1 }">◀</span><span :class="{ on: touchBits & 2 }">▶</span>
        </div>
        <div class="isl-btns">
          <button
            :class="{ on: touchBits & 8 }"
            @pointerdown.prevent="setTouch(8, true)"
            @pointerup.prevent="setTouch(8, false)"
            @pointercancel="setTouch(8, false)"
            @pointerleave="setTouch(8, false)"
          >
            扔
          </button>
          <button
            class="jump"
            :class="{ on: touchBits & 4 }"
            @pointerdown.prevent="setTouch(4, true)"
            @pointerup.prevent="setTouch(4, false)"
            @pointercancel="setTouch(4, false)"
            @pointerleave="setTouch(4, false)"
          >
            跳
          </button>
        </div>
      </template>
    </template>

    <div v-if="isTouch && portrait && screen === 'play'" class="isl-rotate">把手机横过来玩 ↻</div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { IslandGame, Mode, NetStatus } from '@/games/island/game'
import { track } from '@/utils/analytics'
import GameAccountBar from '@/components/games/GameAccountBar.vue'
import GameLeaderboard from '@/components/games/GameLeaderboard.vue'
import GameResultCardButton from '@/components/games/GameResultCardButton.vue'
import type { ResultCardInput } from '@/games/resultCard'

const KEYART = '/games-media/island/keyart.webp'
const stageNames = ['浪花海滩', '雨林深处', '熔岩火山']
const heroNames = ['阿浪', '小椰']

const route = useRoute()
const router = useRouter()
const canvasEl = ref<HTMLCanvasElement | null>(null)
const game = shallowRef<IslandGame | null>(null)
const ready = ref(false)
const screen = ref<'title' | 'lobby' | 'invite' | 'play'>('title')
const mode = ref<Mode | null>(null)
const phase = ref<'play' | 'clear' | 'over' | 'ending'>('play')
const info = reactive({
  stage: 0,
  scores: [0, 0],
  fruits: [0, 0],
  twoP: false
})
// 小游戏可选登录: the team's score goes on the site-wide board with how far the run got.
const teamScore = computed(() => info.scores.slice(0, info.twoP ? 2 : 1).reduce((n, x) => n + Math.floor(x), 0))
const runDetail = computed(() => ({ mode: mode.value ?? 'solo', stage: info.stage + 1, done: phase.value === 'ending' }))
function describeRun(d: Record<string, unknown>): string {
  const m = d.mode === 'solo' ? '单人' : d.mode === 'local2' ? '同屏双人' : '联机双排'
  return d.done ? `${m} · 通关` : `${m} · 第 ${Number(d.stage) || 1} 关`
}
// 战绩图: honour title by the furthest stage (beach, jungle, volcano).
function islandCard(): ResultCardInput {
  const done = phase.value === 'ending' ? 3 : info.stage
  const honor = done >= 3 ? '椰风岛守护者' : info.stage >= 2 ? '火山勇者' : info.stage >= 1 ? '雨林探险家' : '浪花海滩新手'
  return {
    game: 'island',
    title: '椰风岛大冒险',
    art: '/games-media/island/keyart.webp',
    focusX: 0.55,
    deep: '#06303a',
    accent: '#ffd166',
    kicker: done >= 3 ? '三块太阳果碎片合在一起了！' : '这一局',
    honor,
    score: teamScore.value,
    meta: `${done >= 3 ? '通关三关' : `倒在了「${stageNames[info.stage]}」`} · ${describeRun(runDetail.value).split(' · ')[0]}`,
    players: info.twoP
      ? [
          { name: heroNames[0], score: Math.floor(info.scores[0]), color: '#ff6b4a' },
          { name: heroNames[1], score: Math.floor(info.scores[1]), color: '#ffb3d9' }
        ]
      : undefined,
    progress: { done, total: 3 },
    challenge: '你能抢回太阳果吗？'
  }
}
const banner = ref<{ id: number; kicker: string; title: string } | null>(null)
const paused = ref(false)
const muted = ref(false)
const copied = ref(false)
const joinCode = ref('')
const inviteCode = ref('')
const touchBits = ref(0)
const portrait = ref(false)
const net = reactive<{
  role: '' | 'host' | 'guest'
  status: NetStatus | ''
  code: string
  error: string
}>({ role: '', status: '', code: '', error: '' })
const isTouch = typeof window !== 'undefined' && (window.matchMedia('(pointer: coarse)').matches || (import.meta.env.DEV && new URLSearchParams(location.search).has('touch')))
const canFullscreen = typeof document !== 'undefined' && !!document.documentElement.requestFullscreen
const online = computed(() => mode.value === 'host' || mode.value === 'guest')
const shareLink = computed(() => (net.code ? `${location.origin}/games/island?room=${net.code}` : ''))
let bannerTimer = 0
let startedAt = 0
let resizeObs: ResizeObserver | null = null

const ERRORS: Record<string, string> = {
  not_found: '没找到这个房间，检查一下房间码，或者房主已经离开了。',
  room_full: '这个房间已经有两个人了。',
  bad_game: '房间码不是这个游戏的。',
  too_big: '网络消息太大，连接断开了。',
  rate: '网络消息太频繁，连接断开了。',
  host_left: '房主已经离开，这个房间结束了。',
  closed: '和服务器的连接断开了，稍后再试。'
}

function cleanCode(c: string) {
  return c
    .trim()
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, '')
}

function unlock() {
  game.value?.audio.unlock()
}

function enterPlay(m: Mode) {
  mode.value = m
  screen.value = 'play'
  phase.value = 'play'
  paused.value = false
  startedAt = Date.now()
  track('game_start', { game: 'island', mode: m })
  if (isTouch) {
    void document.documentElement.requestFullscreen?.().catch(() => undefined)
    const o = window.screen as unknown as {
      orientation?: { lock?: (x: string) => Promise<void> }
    }
    void o.orientation?.lock?.('landscape').catch(() => undefined)
  }
}

function startSolo() {
  unlock()
  game.value?.startLocal(false)
}

function startLocal2() {
  unlock()
  game.value?.startLocal(true)
}

function openLobby() {
  unlock()
  net.role = ''
  net.status = ''
  net.code = ''
  net.error = ''
  screen.value = 'lobby'
}

function host() {
  unlock()
  net.role = 'host'
  net.error = ''
  game.value?.hostRoom()
}

function join(code: string) {
  const c = cleanCode(code)
  if (c.length < 3) return
  unlock()
  net.role = 'guest'
  net.code = c
  net.error = ''
  screen.value = 'lobby'
  game.value?.joinRoom(c)
}

function acceptInvite() {
  join(inviteCode.value)
}

function beginOnline() {
  game.value?.beginOnline()
}

function backToTitle() {
  game.value?.quit()
  net.role = ''
  net.status = ''
  net.code = ''
  net.error = ''
  screen.value = 'title'
  if (route.query.room) void router.replace({ query: {} })
}

function leave() {
  if (mode.value)
    track('game_finish', {
      game: 'island',
      mode: mode.value,
      stage: info.stage + 1,
      result: phase.value === 'ending' ? 'ending' : 'quit',
      seconds: Math.round((Date.now() - startedAt) / 1000)
    })
  mode.value = null
  backToTitle()
}

function doContinue() {
  game.value?.continue()
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(shareLink.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1800)
  } catch {
    copied.value = false
  }
}

function toggleMute() {
  muted.value = !muted.value
  game.value?.audio.setMuted(muted.value)
  try {
    localStorage.setItem('hg_island_muted', muted.value ? '1' : '0')
  } catch {
    /* private mode */
  }
}

function togglePause() {
  if (online.value || phase.value !== 'play') return
  paused.value = !paused.value
  game.value?.setPaused(paused.value)
}

function toggleFullscreen() {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void document.documentElement.requestFullscreen?.().catch(() => undefined)
}

// ───── touch ─────
function setTouch(bit: number, on: boolean) {
  touchBits.value = on ? touchBits.value | bit : touchBits.value & ~bit
  if (game.value) game.value.input.touch = touchBits.value
}

let padPointer = -1
function padBits(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  const k = (e.clientX - r.left) / r.width
  return k < 0.5 ? 1 : 2
}
function padDown(e: PointerEvent) {
  padPointer = e.pointerId
  ;(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId)
  setTouch(3, false)
  setTouch(padBits(e), true)
}
function padMove(e: PointerEvent) {
  if (e.pointerId !== padPointer) return
  setTouch(3, false)
  setTouch(padBits(e), true)
}
function padUp(e: PointerEvent) {
  if (e.pointerId !== padPointer) return
  padPointer = -1
  setTouch(3, false)
}

function onKey(e: KeyboardEvent) {
  if (screen.value !== 'play') return
  if (e.code === 'Escape' || e.code === 'KeyP') togglePause()
}

function onResize() {
  const el = canvasEl.value
  if (!el || !game.value) return
  game.value.resize(el.clientWidth, el.clientHeight)
  portrait.value = window.innerHeight > window.innerWidth
}

onMounted(async () => {
  try {
    muted.value = localStorage.getItem('hg_island_muted') === '1'
  } catch {
    /* private mode */
  }
  const room = typeof route.query.room === 'string' ? cleanCode(route.query.room) : ''
  if (room) {
    inviteCode.value = room
    screen.value = 'invite'
  }
  const { IslandGame: Game } = await import('@/games/island/game')
  const el = canvasEl.value
  if (!el) return
  const g = new Game(el, {
    banner(kicker, title) {
      banner.value = { id: Date.now(), kicker, title }
      window.clearTimeout(bannerTimer)
      bannerTimer = window.setTimeout(() => (banner.value = null), 2600)
    },
    phase(p, i) {
      phase.value = p
      info.stage = i.stage
      info.scores = i.scores
      info.fruits = i.fruits
      info.twoP = i.twoP
      if (p === 'over' || p === 'ending') {
        track('game_finish', {
          game: 'island',
          mode: mode.value ?? '',
          stage: i.stage + 1,
          result: p,
          score: Math.max(...i.scores),
          seconds: Math.round((Date.now() - startedAt) / 1000)
        })
      }
    },
    net(status, i) {
      net.status = status
      if (i.code) net.code = i.code
      if (status === 'error') {
        net.error = ERRORS[i.error ?? ''] ?? '连接失败，稍后再试。'
        if (screen.value === 'play') screen.value = 'lobby'
      }
    },
    started(m) {
      enterPlay(m)
    }
  })
  game.value = g
  resizeObs = new ResizeObserver(() => onResize())
  resizeObs.observe(el)
  g.audio.setMuted(muted.value)
  g.audio.play('title')
  onResize()
  await g.init()
  ready.value = true
  if (import.meta.env.DEV) (window as unknown as Record<string, unknown>).__island = g
  window.addEventListener('resize', onResize)
  window.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  resizeObs?.disconnect()
  window.removeEventListener('resize', onResize)
  window.removeEventListener('keydown', onKey)
  window.clearTimeout(bannerTimer)
  game.value?.destroy()
})
</script>

<style scoped>
.isl-page {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: #05030a;
  color: #fff;
  font-family: 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', system-ui, sans-serif;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}
.isl-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  image-rendering: pixelated;
}
.isl-canvas.hidden {
  visibility: hidden;
}

/* title */
.isl-title {
  position: absolute;
  inset: 0;
  overflow-y: auto;
  touch-action: pan-y;
}
.isl-art {
  position: fixed;
  inset: 0;
  background-size: cover;
  background-position: center 30%;
  image-rendering: pixelated;
  animation: isl-drift 30s ease-in-out infinite alternate;
}
.isl-art::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(5, 3, 20, 0) 30%, rgba(5, 3, 20, 0.55) 62%, rgba(5, 3, 20, 0.92) 100%);
}
@keyframes isl-drift {
  from {
    transform: scale(1.04) translateX(-1%);
  }
  to {
    transform: scale(1.08) translateX(1%);
  }
}
.isl-title-inner {
  position: relative;
  display: flex;
  min-height: 100%;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 5vh 16px 22px;
  text-align: center;
}
.isl-title-top,
.isl-title-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.isl-kicker {
  font-size: 13px;
  letter-spacing: 0.2em;
  color: #ffe9a8;
  text-shadow: 0 2px 0 #1b1028;
}
.isl-logo {
  margin: 6px 0 8px;
  line-height: 1;
  font-weight: 900;
  letter-spacing: 0.04em;
}
.isl-logo span,
.isl-logo b {
  display: inline-block;
  font-size: clamp(44px, 8vw, 88px);
  color: #ffd23a;
  -webkit-text-stroke: 3px #1b1028;
  paint-order: stroke fill;
  text-shadow:
    0 5px 0 #c4501c,
    0 9px 0 #1b1028;
  transform: rotate(-3deg);
}
.isl-logo b {
  color: #5fe0ff;
  text-shadow:
    0 5px 0 #1f7fb5,
    0 9px 0 #1b1028;
  transform: rotate(2deg);
  margin-left: 0.15em;
}
.isl-tag {
  max-width: 560px;
  font-size: 15px;
  line-height: 1.6;
  color: #fff6e0;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.7);
}
.isl-menu {
  margin-top: 18px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
}
.isl-btn {
  min-width: 132px;
  padding: 12px 22px;
  border: 3px solid #1b1028;
  border-radius: 14px;
  background: #fff6e0;
  color: #1b1028;
  font-size: 17px;
  font-weight: 800;
  box-shadow: 0 5px 0 #1b1028;
  transition: transform 0.08s;
  cursor: pointer;
}
.isl-btn:hover:not(:disabled) {
  transform: translateY(-2px);
}
.isl-btn:active:not(:disabled) {
  transform: translateY(3px);
  box-shadow: 0 2px 0 #1b1028;
}
.isl-btn:disabled {
  opacity: 0.55;
  cursor: default;
}
.isl-btn.primary {
  background: #ffd23a;
}
.isl-btn.online {
  background: #5fe0ff;
}
.isl-btn.wide {
  display: block;
  width: 100%;
  margin-top: 12px;
}
.isl-keys {
  margin-top: 18px;
  display: grid;
  gap: 6px;
  font-size: 13px;
  color: #e8ddff;
}
.isl-keys > div {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}
.isl-keys b {
  color: #ffd23a;
}
.isl-keys-note {
  color: #bfb4d8;
}
kbd {
  display: inline-block;
  min-width: 20px;
  margin: 0 1px;
  padding: 1px 5px;
  border: 1px solid #6a5a8a;
  border-bottom-width: 3px;
  border-radius: 5px;
  background: #241a3a;
  font-family: inherit;
  font-size: 12px;
}
.isl-how {
  margin-top: 14px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px 18px;
  font-size: 13px;
  color: #fff6e0;
}
.isl-back {
  position: fixed;
  left: 14px;
  top: 12px;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(27, 16, 40, 0.65);
  font-size: 14px;
  color: #fff;
}
.isl-ai {
  margin-top: 12px;
  font-size: 11px;
  color: #9a8fb8;
}

/* panels */
.isl-panel-wrap {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: radial-gradient(circle at 50% 40%, rgba(40, 20, 70, 0.6), rgba(5, 3, 10, 0.92));
  z-index: 5;
}
.isl-panel-wrap.dim {
  background: rgba(5, 3, 10, 0.6);
}
.isl-panel-wrap.clear,
.isl-panel-wrap.ending {
  background: rgba(5, 3, 10, 0.35);
  pointer-events: none;
}
.isl-panel-wrap.ending {
  pointer-events: auto;
  background:
    radial-gradient(circle at 50% 45%, rgba(255, 210, 90, 0.25), rgba(5, 3, 10, 0.75) 70%),
    url('/games-media/island/keyart.webp') center 30% / cover;
}
.isl-panel {
  width: min(420px, 100%);
  padding: 22px 22px 18px;
  border: 3px solid #1b1028;
  border-radius: 18px;
  background: #2a1d48;
  box-shadow:
    0 6px 0 #1b1028,
    inset 0 2px 0 rgba(255, 255, 255, 0.12);
  text-align: center;
}
.isl-panel h2 {
  font-size: 24px;
  font-weight: 900;
  color: #ffd23a;
  text-shadow: 0 3px 0 #1b1028;
}
.isl-panel-sub {
  margin-top: 8px;
  font-size: 14px;
  line-height: 1.6;
  color: #e8ddff;
}
.isl-code {
  margin: 6px auto 4px;
  font-size: 46px;
  font-weight: 900;
  letter-spacing: 0.18em;
  color: #5fe0ff;
  text-shadow: 0 4px 0 #1b1028;
  user-select: text;
}
.isl-share,
.isl-join {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.isl-share input,
.isl-join input {
  flex: 1;
  min-width: 0;
  padding: 10px 12px;
  border: 2px solid #1b1028;
  border-radius: 10px;
  background: #fff6e0;
  color: #1b1028;
  font-size: 15px;
  user-select: text;
}
.isl-join input {
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-weight: 700;
}
.isl-share .isl-btn,
.isl-join .isl-btn {
  min-width: 0;
  padding: 8px 12px;
  font-size: 14px;
}
.isl-status {
  margin-top: 14px;
  font-size: 15px;
  color: #ffe9a8;
}
.isl-status.ok {
  color: #7dff9a;
}
.isl-error {
  margin-top: 12px;
  font-size: 14px;
  color: #ff9a9a;
}
.isl-link {
  display: inline-block;
  margin-top: 14px;
  font-size: 14px;
  color: #bfb4d8;
  background: none;
  border: 0;
  cursor: pointer;
}
.isl-scores {
  display: flex;
  justify-content: center;
  gap: 18px;
  margin-top: 14px;
}
.isl-scores div {
  display: flex;
  flex-direction: column;
  padding: 8px 16px;
  border-radius: 12px;
  background: #1b1028;
}
.isl-scores span {
  font-size: 13px;
  color: #bfb4d8;
}
.isl-scores b {
  font-size: 24px;
  color: #fff;
}
.isl-scores .p1 b {
  color: #5fe0ff;
}
.isl-scores .p2 b {
  color: #ffd23a;
}

/* in-game chrome */
.isl-tools {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: flex;
  gap: 6px;
  z-index: 4;
}
.touch .isl-tools {
  top: 24%;
  bottom: auto;
  right: max(8px, env(safe-area-inset-right));
  flex-direction: column;
}
.isl-icon {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: rgba(27, 16, 40, 0.6);
  color: #fff;
  font-size: 15px;
  cursor: pointer;
}
.isl-room-tag {
  position: absolute;
  left: 10px;
  bottom: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(27, 16, 40, 0.6);
  font-size: 12px;
  color: #bfb4d8;
  z-index: 4;
}
.isl-banner {
  position: absolute;
  left: 50%;
  top: 26%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
  z-index: 3;
}
.isl-banner small {
  padding: 3px 14px;
  border: 2px solid #1b1028;
  border-radius: 999px;
  background: #ffd23a;
  color: #1b1028;
  font-size: 14px;
  font-weight: 800;
}
.isl-banner strong {
  margin-top: 6px;
  font-size: clamp(32px, 6vw, 60px);
  font-weight: 900;
  color: #fff6e0;
  -webkit-text-stroke: 2px #1b1028;
  paint-order: stroke fill;
  text-shadow: 0 5px 0 #1b1028;
}
.isl-banner.boss small {
  background: #ff4f4f;
  color: #fff;
}
.isl-banner.boss strong {
  color: #ffb0a0;
}
.isl-banner-enter-active,
.isl-banner-leave-active {
  transition:
    opacity 0.35s,
    transform 0.35s;
}
.isl-banner-enter-from,
.isl-banner-leave-to {
  opacity: 0;
  transform: translate(-50%, -14px) scale(0.92);
}
.isl-pad {
  position: absolute;
  left: max(16px, env(safe-area-inset-left));
  bottom: 18px;
  display: flex;
  gap: 10px;
  z-index: 4;
}
.isl-pad span,
.isl-btns button {
  display: flex;
  width: 68px;
  height: 68px;
  align-items: center;
  justify-content: center;
  border: 3px solid rgba(255, 255, 255, 0.55);
  border-radius: 50%;
  background: rgba(27, 16, 40, 0.35);
  color: #fff;
  font-size: 24px;
  font-weight: 800;
}
.isl-pad span.on,
.isl-btns button.on {
  background: rgba(255, 210, 58, 0.55);
}
.isl-btns {
  position: absolute;
  right: max(16px, env(safe-area-inset-right));
  bottom: 18px;
  display: flex;
  gap: 12px;
  align-items: flex-end;
  z-index: 4;
}
.isl-btns .jump {
  width: 80px;
  height: 80px;
  margin-bottom: 26px;
}
.isl-rotate {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(5, 3, 10, 0.88);
  font-size: 20px;
  z-index: 9;
}
@media (max-height: 480px) {
  .isl-title-inner {
    padding-top: 12px;
  }
  .isl-how,
  .isl-tag {
    display: none;
  }
}
</style>
