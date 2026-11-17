<template>
  <div class="cc-page" :class="{ touch: isTouch, playing: ui.screen === 'play' }" @pointerdown="wake" @keydown="wake">
    <canvas ref="canvasEl" class="cc-canvas"></canvas>

    <!-- title -->
    <section v-if="ui.screen === 'title' && !final" class="cc-title" data-testid="cotton-title">
      <img class="cc-keyart" :src="`${MEDIA}keyart.webp`" alt="" />
      <div class="cc-title-shade"></div>
      <RouterLink to="/games" class="cc-back">← 小游戏</RouterLink>
      <div class="cc-title-box">
        <p class="cc-kicker">HiveGPT 小游戏 · 90 后街机回忆</p>
        <h1 class="cc-logo"><span>棉花糖</span><span>大作战</span></h1>
        <p class="cc-tagline">暑假最后一夜，作业本、闹钟、蚊子全成了精。<br />拿起糖丝棒，把它们裹成棉花糖球，一脚踢下去！</p>
        <div v-if="panel === 'menu'" class="cc-menu">
          <button class="cc-btn cc-btn-main" data-testid="cotton-solo" @click="startLocal('solo')">单人开始</button>
          <button class="cc-btn" data-testid="cotton-duo" @click="startLocal('duo')">同屏双人</button>
          <button class="cc-btn cc-btn-online" data-testid="cotton-online" @click="panel = 'online'">联机双排</button>
          <div class="cc-links">
            <button type="button" @click="panel = 'help'">怎么玩</button>
            <button type="button" @click="panel = 'rank'">排行榜</button>
          </div>
        </div>
        <div v-else-if="panel === 'online'" class="cc-card">
          <h2>联机双排</h2>
          <p>开一个房间，把房间码或链接发给朋友，两个人在各自的电脑或手机上一起打。不用登录。</p>
          <button class="cc-btn cc-btn-main" data-testid="cotton-host" @click="hostRoom">开房间</button>
          <form class="cc-join" @submit.prevent="joinRoom(joinCode)">
            <input v-model="joinCode" maxlength="8" placeholder="输入房间码" autocomplete="off" spellcheck="false" data-testid="cotton-code" />
            <button class="cc-btn" type="submit" :disabled="joinCode.trim().length < 3">加入</button>
          </form>
          <button type="button" class="cc-link" @click="panel = 'menu'">返回</button>
        </div>
        <div v-else-if="panel === 'help'" class="cc-card cc-help">
          <h2>怎么玩</h2>
          <ul>
            <li>朝怪物发射糖丝，<b>打四下</b>就裹成一个棉花糖球。</li>
            <li>走到球边上再按发射，<b>一脚踢出去</b>：球会弹墙、一路碾过去，连着撞的越多分越高（400 → 800 → 1600…）。</li>
            <li>球撞到别的球会一起滚，一脚清场有<b>红包雨</b>。</li>
            <li>彩色汽水：<span class="red">红＝力量</span> · <span class="blue">蓝＝速度</span> · <span class="yellow">黄＝射程</span>；糖葫芦加一条命。</li>
            <li>双人时队友倒下会变成一朵棉花糖云，<b>碰一下就复活</b>。</li>
            <li>拖太久妈妈会来查房，拖鞋可是打不掉的。每 5 层有一个大怪。</li>
          </ul>
          <div class="cc-keys">
            <div><b class="p1">1P</b> <kbd>A</kbd><kbd>D</kbd> 移动 · <kbd>J</kbd> 发射 · <kbd>K</kbd>/<kbd>W</kbd> 跳 · <kbd>S</kbd>+跳 下平台</div>
            <div><b class="p2">2P</b> <kbd>←</kbd><kbd>→</kbd> 移动 · <kbd>.</kbd> 发射 · <kbd>/</kbd>/<kbd>↑</kbd> 跳（小键盘 1 / 2 也行）</div>
            <div>单人时两套按键都能用，也支持手柄；手机上有触屏按键。</div>
          </div>
          <button type="button" class="cc-link" @click="panel = 'menu'">返回</button>
        </div>
        <div v-else-if="panel === 'rank'" class="cc-card">
          <h2>本机排行榜</h2>
          <ol v-if="ranks.length" class="cc-rank">
            <li v-for="(r, i) in ranks" :key="i">
              <span class="n">{{ i + 1 }}</span><b>{{ r.score.toLocaleString() }}</b><span>第 {{ r.floor }} 层</span><span>{{ modeName(r.mode) }}</span><span class="d">{{ r.date }}</span>
            </li>
          </ol>
          <p v-else>还没有记录，打一局试试。</p>
          <button type="button" class="cc-link" @click="panel = 'menu'">返回</button>
        </div>
        <p class="cc-small">原创角色与关卡 · 背景和封面由 AI 生成 · 建议开声音</p>
      </div>
    </section>

    <!-- online lobby -->
    <section v-if="ui.screen === 'lobby'" class="cc-overlay cc-lobby-wrap" data-testid="cotton-lobby">
      <img class="cc-keyart" :src="`${MEDIA}keyart.webp`" alt="" />
      <div class="cc-title-shade"></div>
      <div class="cc-card cc-lobby cc-raise">
        <template v-if="ui.room?.status === 'error' || ui.room?.status === 'closed'">
          <h2>{{ ui.room?.error === 'closed' ? '房间已关闭' : '连不上房间' }}</h2>
          <p class="cc-err">{{ roomError(ui.room?.error || '') }}</p>
          <button class="cc-btn" @click="backToTitle">回到标题</button>
        </template>
        <template v-else-if="ui.mode === 'host'">
          <h2>房间已开好</h2>
          <p v-if="ui.room?.status === 'connecting'">正在开房间…</p>
          <template v-else>
            <div class="cc-code" data-testid="cotton-room-code">{{ ui.room?.code }}</div>
            <p>把房间码发给朋友，或者直接发链接：</p>
            <div class="cc-share">
              <input :value="shareLink" readonly @focus="($event.target as HTMLInputElement).select()" />
              <button class="cc-btn" @click="copyLink">{{ copied ? '已复制' : '复制链接' }}</button>
            </div>
            <p class="cc-status" :class="{ ok: ui.room?.peer }">{{ ui.room?.peer ? '✔ 队友已进房间' : '等队友进来…（也可以先开打，队友随时能加入）' }}</p>
            <button class="cc-btn cc-btn-main" data-testid="cotton-host-start" @click="game?.hostStart()">{{ ui.room?.peer ? '两个人，开打！' : '先自己开打' }}</button>
          </template>
          <button type="button" class="cc-link" @click="backToTitle">取消</button>
        </template>
        <template v-else>
          <h2>加入房间 {{ ui.room?.code }}</h2>
          <p>{{ ui.room?.status === 'connecting' ? '正在连接…' : '已进房间，等房主开始…' }}</p>
          <p class="cc-small">你是 2P 阿豆（蓝色）。键盘任意一套按键都能操作。</p>
          <button type="button" class="cc-link" @click="backToTitle">取消</button>
        </template>
      </div>
    </section>

    <!-- in-game buttons -->
    <div v-if="ui.screen === 'play'" class="cc-hud">
      <button class="cc-icon" :title="muted ? '打开声音' : '关闭声音'" @click="toggleMute">{{ muted ? '🔇' : '🔊' }}</button>
      <button v-if="ui.mode === 'solo' || ui.mode === 'duo'" class="cc-icon" title="暂停 (Esc)" @click="game?.setPaused(true)">⏸</button>
      <button v-if="canFullscreen" class="cc-icon" title="全屏" @click="toggleFullscreen">⛶</button>
      <button class="cc-icon" title="离开" @click="backToTitle">✕</button>
      <span v-if="ui.room" class="cc-room">房间 {{ ui.room.code }}<template v-if="ui.mode === 'host' && !ui.room.peer"> · 等队友</template></span>
    </div>

    <!-- touch pad -->
    <div v-if="ui.screen === 'play' && isTouch" class="cc-touch">
      <div class="cc-pad">
        <button v-for="k in ['l', 'd', 'r']" :key="k" :class="`k-${k}`" @pointerdown.prevent="touch(k, true)" @pointerup.prevent="touch(k, false)" @pointercancel="touch(k, false)" @pointerleave="touch(k, false)">
          {{ k === 'l' ? '◀' : k === 'r' ? '▶' : '▼' }}
        </button>
      </div>
      <div class="cc-act">
        <button class="k-fire" @pointerdown.prevent="touch('fire', true)" @pointerup.prevent="touch('fire', false)" @pointercancel="touch('fire', false)" @pointerleave="touch('fire', false)">发射</button>
        <button class="k-jump" @pointerdown.prevent="touch('jump', true)" @pointerup.prevent="touch('jump', false)" @pointercancel="touch('jump', false)" @pointerleave="touch('jump', false)">跳</button>
      </div>
    </div>

    <!-- floor result -->
    <Transition name="cc-pop">
      <div v-if="ui.screen === 'play' && ui.phase === 'result' && ui.result" class="cc-result" data-testid="cotton-floor-result">
        <h2>第 {{ ui.floor + 1 }} 层 通关</h2>
        <p v-if="ui.result.sweep" class="cc-badge">一球清场！</p>
        <dl>
          <div><dt>用时</dt><dd>{{ ui.result.seconds }} 秒</dd></div>
          <div><dt>{{ bossFloor ? '打败大怪' : '时间奖励' }}</dt><dd>+{{ ui.result.bonus }}</dd></div>
          <div><dt>最长连锁</dt><dd>{{ ui.result.bestChain || 0 }} 连</dd></div>
        </dl>
        <div class="cc-kids">
          <div v-for="(k, i) in ui.kids" v-show="k.on" :key="i" :class="`kid kid-${i}`">
            <b>{{ KIDS[i] }}</b><span>本层 {{ ui.result.floorScore[i].toLocaleString() }}</span><span>总分 {{ k.score.toLocaleString() }}</span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- paused -->
    <div v-if="ui.screen === 'play' && ui.paused" class="cc-overlay">
      <div class="cc-card cc-lobby">
        <h2>暂停</h2>
        <button class="cc-btn cc-btn-main" @click="game?.setPaused(false)">继续</button>
        <button class="cc-btn" @click="backToTitle">回到标题</button>
      </div>
    </div>

    <!-- game over -->
    <div v-if="ui.screen === 'play' && ui.phase === 'over' && ui.phaseT >= 80 && !final" class="cc-overlay" data-testid="cotton-over">
      <div class="cc-card cc-lobby">
        <h2 class="cc-over">妈妈喊你睡觉了</h2>
        <p>打到第 {{ ui.floor + 1 }} 层 · {{ totalScore.toLocaleString() }} 分</p>
        <button class="cc-btn cc-btn-main" data-testid="cotton-continue" @click="game?.continueRun()">接关（从这一层重来，分数清零）</button>
        <button class="cc-btn" data-testid="cotton-end" @click="game?.endRun()">看成绩</button>
      </div>
    </div>

    <!-- final -->
    <section v-if="final" class="cc-overlay cc-final-wrap" data-testid="cotton-final">
      <div class="cc-card cc-final">
        <p class="cc-kicker">{{ final.won ? '暑假保住了！' : '这一局' }}</p>
        <h2 class="cc-honor">「{{ honor }}」</h2>
        <div class="cc-big">{{ final.score.toLocaleString() }}<small>分</small></div>
        <p class="cc-sub">打到第 {{ final.reached }} 层 · {{ minutes }} · {{ modeName(final.mode) }}</p>
        <div class="cc-kids">
          <div v-for="(k, i) in final.kids" v-show="k.on" :key="i" :class="`kid kid-${i}`">
            <b>{{ KIDS[i] }}</b><span>{{ k.score.toLocaleString() }} 分</span><span>裹掉 {{ k.kills }} 只</span><span>最长 {{ k.bestChain }} 连</span>
          </div>
        </div>
        <ol class="cc-rank">
          <li v-for="(r, i) in ranks.slice(0, 5)" :key="i" :class="{ me: i === rankIndex }">
            <span class="n">{{ i + 1 }}</span><b>{{ r.score.toLocaleString() }}</b><span>第 {{ r.floor }} 层</span><span>{{ modeName(r.mode) }}</span><span class="d">{{ r.date }}</span>
          </li>
        </ol>
        <div class="cc-final-btns">
          <button class="cc-btn cc-btn-main" data-testid="cotton-share" :disabled="sharing" @click="shareCard">{{ sharing ? '生成中…' : '生成战绩图' }}</button>
          <button v-if="final.mode !== 'guest'" class="cc-btn" @click="playAgain">再来一局</button>
          <span v-else class="cc-wait">等房主开下一局…</span>
          <RouterLink to="/games" class="cc-btn">别的小游戏</RouterLink>
        </div>
        <img v-if="cardUrl" :src="cardUrl" class="cc-card-img" alt="战绩图" />
        <p v-if="cardUrl" class="cc-small">{{ isTouch ? '长按图片保存，发给朋友' : '图片已下载，也可以右键另存' }}</p>
      </div>
    </section>

    <Transition name="cc-pop">
      <div v-if="toast" :key="toast" class="cc-toast">{{ toast }}</div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { CottonGame, Mode, Ui } from '@/games/cotton/game'
import { FLOORS } from '@/games/cotton/levels'
import { track } from '@/utils/analytics'

const MEDIA = '/games-media/cotton/'
const KIDS = ['1P 小满', '2P 阿豆']
const RANK_KEY = 'hg_cotton_rank_v1'

interface RankRow {
  score: number
  floor: number
  mode: Mode | null
  date: string
  at: number
}

interface Final {
  score: number
  reached: number
  seconds: number
  mode: Mode | null
  won: boolean
  kids: Ui['kids']
}

const route = useRoute()
const router = useRouter()
const canvasEl = ref<HTMLCanvasElement | null>(null)
const game = shallowRef<CottonGame | null>(null)
const isTouch = typeof window !== 'undefined' && window.matchMedia('(pointer: coarse)').matches
const canFullscreen = typeof document !== 'undefined' && !!document.documentElement.requestFullscreen
const panel = ref<'menu' | 'online' | 'help' | 'rank'>('menu')
const joinCode = ref('')
const muted = ref(localStorage.getItem('hg_cotton_muted') === '1')
const copied = ref(false)
const toast = ref('')
const final = ref<Final | null>(null)
const ranks = ref<RankRow[]>(loadRanks())
const rankIndex = ref(-1)
const cardUrl = ref('')
const sharing = ref(false)
let toastTimer = 0

const ui = ref<Ui>({
  screen: 'title',
  mode: null,
  paused: false,
  phase: 'intro',
  phaseT: 0,
  floor: 0,
  reached: 1,
  seconds: 0,
  result: null,
  kids: [
    { on: true, score: 0, kills: 0, bestChain: 0, lives: 3 },
    { on: false, score: 0, kills: 0, bestChain: 0, lives: 3 }
  ],
  room: null,
  ping: 0,
  notice: '',
  round: 0
})

const bossFloor = computed(() => !!FLOORS[ui.value.floor]?.boss)
const totalScore = computed(() => ui.value.kids.reduce((n, k) => n + (k.on ? k.score : 0), 0))
const shareLink = computed(() => (ui.value.room?.code ? `${location.origin}/games/cotton?room=${ui.value.room.code}` : ''))
const minutes = computed(() => {
  const s = final.value?.seconds ?? 0
  return s >= 60 ? `${Math.floor(s / 60)} 分 ${s % 60} 秒` : `${s} 秒`
})
const honor = computed(() => honorFor(final.value))

function honorFor(f: Final | null): string {
  if (!f) return ''
  if (f.won) return '暑假守护神'
  const cleared = f.reached - 1
  if (cleared >= 14) return '开学前夜的勇士'
  if (cleared >= 10) return '作业山推土机'
  if (cleared >= 7) return '粉笔灰里的英雄'
  if (cleared >= 5) return '闹钟克星'
  if (cleared >= 3) return '卧室保卫者'
  if (cleared >= 1) return '熬夜小能手'
  return '暑假刚开始'
}

function modeName(m: Mode | null): string {
  return m === 'duo' ? '同屏双人' : m === 'host' || m === 'guest' ? '联机双排' : '单人'
}

function loadRanks(): RankRow[] {
  try {
    const v = JSON.parse(localStorage.getItem(RANK_KEY) || '[]')
    return Array.isArray(v) ? v.slice(0, 10) : []
  } catch {
    return []
  }
}

function saveRank(f: Final): void {
  if (f.score <= 0) return
  const d = new Date()
  const row: RankRow = { score: f.score, floor: f.reached, mode: f.mode, date: `${d.getMonth() + 1}/${d.getDate()}`, at: d.getTime() }
  const list = [...loadRanks(), row].sort((a, b) => b.score - a.score).slice(0, 10)
  localStorage.setItem(RANK_KEY, JSON.stringify(list))
  ranks.value = list
  rankIndex.value = list.findIndex((r) => r.at === row.at)
}

function showToast(text: string): void {
  toast.value = text
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = ''), 3200)
}

function roomError(code: string): string {
  switch (code) {
    case 'not_found':
      return '房间不存在或已经结束了，让朋友重新开一个（Room not found）'
    case 'room_full':
      return '这个房间已经有两个人了（Room is full）'
    case 'rate':
      return '操作太频繁，稍等几秒再试（Too many requests）'
    case 'closed':
      return '房主离开了，房间已关闭（Room closed）'
    case 'bad_game':
    case 'too_big':
      return '连接出错了，刷新页面再试（Bad request）'
    default:
      return '网络连不上联机服务器，检查网络后再试（Network error）'
  }
}

function wake(): void {
  game.value?.audio.start()
}

function startLocal(mode: 'solo' | 'duo'): void {
  final.value = null
  cardUrl.value = ''
  game.value?.start(mode)
  game.value?.setMuted(muted.value)
  track('game_start', { game: 'cotton', mode })
}

function hostRoom(): void {
  final.value = null
  cardUrl.value = ''
  game.value?.host()
  game.value?.setMuted(muted.value)
  track('game_start', { game: 'cotton', mode: 'host' })
}

function joinRoom(code: string): void {
  const c = code.trim().toUpperCase().replace(/[^A-Z0-9]/g, '')
  if (c.length < 3) return
  final.value = null
  cardUrl.value = ''
  game.value?.join(c)
  game.value?.setMuted(muted.value)
  track('game_start', { game: 'cotton', mode: 'guest' })
}

function backToTitle(): void {
  game.value?.toTitle()
  final.value = null
  cardUrl.value = ''
  panel.value = 'menu'
  if (route.query.room) void router.replace({ query: {} })
}

function playAgain(): void {
  const mode = final.value?.mode
  final.value = null
  cardUrl.value = ''
  if (mode === 'host') {
    // same room, same friend: just go again
    if (ui.value.room?.status === 'open') game.value?.hostStart()
    else hostRoom()
  } else if (mode === 'guest') backToTitle()
  else startLocal(mode === 'duo' ? 'duo' : 'solo')
}

async function copyLink(): Promise<void> {
  try {
    await navigator.clipboard.writeText(`来和我一起玩《棉花糖大作战》，房间码 ${ui.value.room?.code}：${shareLink.value}`)
    copied.value = true
    window.setTimeout(() => (copied.value = false), 2000)
  } catch {
    showToast('复制失败，手动选中链接复制吧')
  }
}

function toggleMute(): void {
  muted.value = !muted.value
  localStorage.setItem('hg_cotton_muted', muted.value ? '1' : '0')
  game.value?.setMuted(muted.value)
}

function toggleFullscreen(): void {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void document.documentElement.requestFullscreen().catch(() => undefined)
}

function touch(k: string, on: boolean): void {
  game.value?.setTouch({ [k]: on })
}

async function shareCard(): Promise<void> {
  const f = final.value
  const g = game.value
  if (!f || !g) return
  sharing.value = true
  try {
    const { drawResultCard } = await import('@/games/cotton/card')
    const url = await drawResultCard({ ...f, honor: honor.value, sprites: g.renderer.sprites, keyart: `${MEDIA}keyart.webp`, link: `${location.origin}/games/cotton` })
    cardUrl.value = url
    if (!isTouch) {
      const a = document.createElement('a')
      a.href = url
      a.download = `棉花糖大作战-${f.score}.png`
      a.click()
    }
    track('game_cta', { game: 'cotton', action: 'share_card' })
  } catch (err) {
    console.error(err)
    showToast('战绩图没生成出来，再试一次')
  } finally {
    sharing.value = false
  }
}

function onResize(): void {
  game.value?.resize()
}

onMounted(async () => {
  const canvas = canvasEl.value
  if (!canvas) return
  const { CottonGame: Game } = await import('@/games/cotton/game')
  const g = new Game(
    canvas,
    {
      ui: (u) => {
        // an online guest leaves the final screen when the host starts another run
        if (final.value && u.mode === 'guest' && u.round !== ui.value.round) {
          final.value = null
          cardUrl.value = ''
        }
        ui.value = u
        if (u.notice) showToast(u.notice)
      },
      finish: (u) => {
        const f: Final = {
          score: u.kids.reduce((n, k) => n + (k.on ? k.score : 0), 0),
          reached: u.reached,
          seconds: u.seconds,
          mode: u.mode,
          won: u.phase === 'won',
          kids: u.kids
        }
        final.value = f
        saveRank(f)
        track('game_finish', { game: 'cotton', mode: u.mode ?? '', floor: u.reached, score: f.score })
      }
    },
    MEDIA
  )
  g.setMuted(muted.value)
  game.value = g
  if (import.meta.env.DEV) (window as unknown as Record<string, unknown>).__cotton = g
  const room = typeof route.query.room === 'string' ? route.query.room : ''
  if (room) joinRoom(room)
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.clearTimeout(toastTimer)
  game.value?.dispose()
  game.value = null
})
</script>

<style scoped>
.cc-page {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: #0d0820;
  color: #fff;
  font-family: 'PingFang SC', 'HarmonyOS Sans SC', 'Microsoft YaHei', system-ui, sans-serif;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}
.cc-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  image-rendering: pixelated;
}
.cc-back {
  position: absolute;
  top: 16px;
  left: 18px;
  z-index: 3;
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  text-decoration: none;
}
.cc-title {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow-y: auto;
}
.cc-keyart {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  image-rendering: pixelated;
}
.cc-title-shade {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at 50% 45%, rgba(13, 8, 32, 0.55), rgba(13, 8, 32, 0.88) 75%);
}
.cc-title-box {
  position: relative;
  max-width: 560px;
  width: calc(100% - 32px);
  text-align: center;
  padding: 24px 0;
}
.cc-kicker {
  font-size: 13px;
  letter-spacing: 0.2em;
  color: #ffd23f;
  margin: 0;
}
.cc-logo {
  margin: 10px 0 6px;
  font-size: clamp(42px, 10vw, 76px);
  font-weight: 900;
  line-height: 1.02;
  letter-spacing: 0.04em;
  color: #ffc9e6;
  text-shadow:
    0 4px 0 #d54b98,
    0 8px 0 #1b1028,
    4px 0 0 #1b1028,
    -4px 0 0 #1b1028,
    0 -4px 0 #1b1028,
    4px 8px 0 #1b1028,
    -4px 8px 0 #1b1028;
}
.cc-logo span {
  display: inline-block;
}
.cc-logo span + span {
  color: #fff2a0;
  text-shadow:
    0 4px 0 #ff9a3c,
    0 8px 0 #1b1028,
    4px 0 0 #1b1028,
    -4px 0 0 #1b1028,
    0 -4px 0 #1b1028,
    4px 8px 0 #1b1028,
    -4px 8px 0 #1b1028;
}
.cc-tagline {
  margin: 14px auto 22px;
  font-size: 15px;
  line-height: 1.7;
  color: #e8dcff;
  text-shadow: 0 2px 0 #1b1028;
}
.cc-menu {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.cc-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 220px;
  padding: 12px 22px;
  border: 3px solid #1b1028;
  border-radius: 6px;
  background: #fff7f0;
  color: #1b1028;
  font-size: 17px;
  font-weight: 800;
  box-shadow:
    0 4px 0 #1b1028,
    inset 0 -4px 0 #cfc3dc;
  cursor: pointer;
  text-decoration: none;
  transition: transform 0.08s;
}
.cc-btn:hover {
  transform: translateY(-2px);
}
.cc-btn:active {
  transform: translateY(2px);
  box-shadow:
    0 2px 0 #1b1028,
    inset 0 -2px 0 #cfc3dc;
}
.cc-btn:disabled {
  opacity: 0.5;
  cursor: default;
  transform: none;
}
.cc-btn-main {
  background: #ff86c3;
  color: #fff;
  text-shadow: 0 2px 0 #9c2d6c;
  box-shadow:
    0 4px 0 #1b1028,
    inset 0 -4px 0 #d54b98;
}
.cc-btn-online {
  background: #7cc8ff;
  box-shadow:
    0 4px 0 #1b1028,
    inset 0 -4px 0 #3f7fe8;
}
.cc-links {
  display: flex;
  gap: 18px;
  margin-top: 4px;
}
.cc-links button,
.cc-link {
  background: none;
  border: 0;
  color: #c8b6f5;
  font-size: 14px;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 3px;
  margin-top: 6px;
}
.cc-small {
  margin-top: 18px;
  font-size: 12px;
  color: rgba(232, 220, 255, 0.6);
}
.cc-card {
  background: rgba(27, 16, 40, 0.94);
  border: 3px solid #1b1028;
  outline: 3px solid #56406b;
  border-radius: 8px;
  padding: 20px 22px;
  text-align: left;
  color: #e8dcff;
  font-size: 14px;
  line-height: 1.7;
}
.cc-card h2 {
  margin: 0 0 8px;
  font-size: 22px;
  color: #fff;
  font-weight: 900;
}
.cc-card .cc-btn {
  margin-top: 10px;
}
.cc-join {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
.cc-join input,
.cc-share input {
  flex: 1;
  min-width: 0;
  padding: 10px 12px;
  border: 3px solid #1b1028;
  border-radius: 6px;
  background: #fff;
  color: #1b1028;
  font-size: 17px;
  font-weight: 800;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  user-select: text;
}
.cc-share input {
  font-size: 13px;
  letter-spacing: 0;
  text-transform: none;
}
.cc-join .cc-btn,
.cc-share .cc-btn {
  min-width: 0;
  margin-top: 0;
}
.cc-share {
  display: flex;
  gap: 8px;
}
.cc-help ul {
  margin: 0 0 10px;
  padding-left: 18px;
}
.cc-help b {
  color: #fff2a0;
}
.cc-help .red {
  color: #ff7a85;
}
.cc-help .blue {
  color: #7cc8ff;
}
.cc-help .yellow {
  color: #ffd23f;
}
.cc-keys {
  font-size: 13px;
  display: grid;
  gap: 6px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 6px;
}
.cc-keys .p1 {
  color: #ff6b7a;
}
.cc-keys .p2 {
  color: #6aa8ff;
}
kbd {
  display: inline-block;
  min-width: 20px;
  padding: 0 5px;
  margin: 0 1px;
  border: 2px solid #1b1028;
  border-radius: 4px;
  background: #fff7f0;
  color: #1b1028;
  font-size: 12px;
  font-weight: 800;
  text-align: center;
  line-height: 18px;
}
.cc-overlay {
  position: absolute;
  inset: 0;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(13, 8, 32, 0.6);
  padding: 16px;
}
.cc-raise {
  position: relative;
}
.cc-lobby {
  width: min(440px, 100%);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  text-align: center;
}
.cc-code {
  font-size: 54px;
  font-weight: 900;
  letter-spacing: 0.18em;
  color: #fff2a0;
  text-shadow:
    0 4px 0 #ff9a3c,
    0 7px 0 #1b1028;
  margin: 2px 0 8px;
}
.cc-status {
  margin: 12px 0 0;
  color: #c8b6f5;
}
.cc-status.ok {
  color: #8ff0a4;
  font-weight: 800;
}
.cc-err {
  color: #ffb0b8;
}
.cc-over {
  color: #ff8a9a !important;
}
.cc-hud {
  position: absolute;
  top: 8px;
  right: 10px;
  z-index: 3;
  display: flex;
  gap: 6px;
  align-items: center;
}
.cc-icon {
  width: 34px;
  height: 34px;
  border: 2px solid rgba(255, 255, 255, 0.18);
  border-radius: 6px;
  background: rgba(27, 16, 40, 0.7);
  color: #fff;
  font-size: 15px;
  cursor: pointer;
}
.cc-room {
  font-size: 12px;
  color: #fff2a0;
  background: rgba(27, 16, 40, 0.7);
  padding: 6px 8px;
  border-radius: 6px;
}
.cc-result {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 3;
  transform: translate(-50%, -50%);
  width: min(420px, calc(100% - 32px));
  background: rgba(27, 16, 40, 0.95);
  border: 3px solid #1b1028;
  outline: 3px solid #ff86c3;
  border-radius: 8px;
  padding: 18px 22px;
  text-align: center;
}
.cc-result h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 900;
  color: #fff2a0;
}
.cc-badge {
  display: inline-block;
  margin: 8px 0 0;
  padding: 2px 12px;
  background: #ff5a5a;
  color: #fff;
  font-weight: 900;
  border-radius: 4px;
}
.cc-result dl {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin: 14px 0;
}
.cc-result dt {
  font-size: 12px;
  color: #c8b6f5;
}
.cc-result dd {
  margin: 2px 0 0;
  font-size: 18px;
  font-weight: 900;
}
.cc-kids {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
}
.kid {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.06);
  font-size: 13px;
  text-align: left;
}
.kid-0 b {
  color: #ff6b7a;
}
.kid-1 b {
  color: #6aa8ff;
}
.cc-final-wrap {
  overflow-y: auto;
  align-items: flex-start;
}
.cc-final {
  width: min(520px, 100%);
  margin: auto;
  text-align: center;
}
.cc-honor {
  font-size: 28px !important;
  color: #fff2a0 !important;
  margin-top: 4px !important;
}
.cc-big {
  font-size: 52px;
  font-weight: 900;
  color: #ffc9e6;
  text-shadow:
    0 3px 0 #d54b98,
    0 6px 0 #1b1028;
  line-height: 1.1;
}
.cc-big small {
  font-size: 18px;
  margin-left: 4px;
}
.cc-sub {
  margin: 4px 0 12px;
}
.cc-rank {
  list-style: none;
  padding: 0;
  margin: 14px 0 0;
  display: grid;
  gap: 4px;
  text-align: left;
}
.cc-rank li {
  display: grid;
  grid-template-columns: 22px 1fr auto auto auto;
  gap: 10px;
  padding: 4px 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.05);
  font-size: 13px;
}
.cc-rank li.me {
  background: rgba(255, 134, 195, 0.25);
  outline: 2px solid #ff86c3;
}
.cc-rank .n {
  color: #ffd23f;
  font-weight: 900;
}
.cc-rank .d {
  color: rgba(232, 220, 255, 0.55);
}
.cc-final-btns {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 14px;
}
.cc-final-btns .cc-btn {
  min-width: 140px;
  margin-top: 0;
}
.cc-wait {
  align-self: center;
  color: #c8b6f5;
}
.cc-card-img {
  display: block;
  width: 100%;
  max-width: 320px;
  margin: 14px auto 0;
  border-radius: 6px;
  user-select: auto;
  -webkit-user-select: auto;
  -webkit-touch-callout: default;
}
.cc-toast {
  position: absolute;
  left: 50%;
  bottom: 18px;
  z-index: 5;
  transform: translateX(-50%);
  background: rgba(27, 16, 40, 0.92);
  border: 2px solid #56406b;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
}
.cc-touch {
  position: absolute;
  inset: auto 0 0 0;
  z-index: 3;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  padding: 0 12px calc(14px + env(safe-area-inset-bottom));
  pointer-events: none;
}
.cc-touch button {
  pointer-events: auto;
  border: 3px solid rgba(255, 255, 255, 0.35);
  background: rgba(27, 16, 40, 0.45);
  color: #fff;
  font-weight: 900;
  border-radius: 50%;
  touch-action: none;
}
.cc-pad {
  display: grid;
  grid-template-columns: 58px 58px 58px;
  gap: 6px;
  align-items: end;
}
.cc-pad button {
  width: 58px;
  height: 58px;
  font-size: 22px;
}
.cc-act {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}
.cc-act button {
  width: 70px;
  height: 70px;
  font-size: 17px;
}
.cc-act .k-fire {
  background: rgba(255, 134, 195, 0.55);
  margin-bottom: 28px;
}
.cc-act .k-jump {
  background: rgba(124, 200, 255, 0.45);
}
.cc-pop-enter-active,
.cc-pop-leave-active {
  transition:
    opacity 0.25s,
    transform 0.25s;
}
.cc-pop-enter-from,
.cc-pop-leave-to {
  opacity: 0;
}
@media (max-height: 520px) {
  .cc-logo {
    font-size: 40px;
  }
  .cc-tagline {
    margin: 8px auto 12px;
    font-size: 13px;
  }
  .cc-btn {
    padding: 8px 18px;
    font-size: 15px;
  }
}
</style>
