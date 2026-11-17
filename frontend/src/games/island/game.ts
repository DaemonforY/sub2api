// 椰风岛大冒险: the game loop, local modes and online host / guest.
import { EV, type GameEvent } from './defs'
import { GameAudio, type Track } from './audio'
import { Input } from './input'
import { stages } from './level'
import { Renderer, type RView } from './render'
import { Room, decodeEnt, decodePlayer, makeSnapshot, phaseOf, type Snapshot } from './net'
import { continueGame, createWorld, loadStage, movePlayer, newPlayer, stepWorld, type Ent, type PhysCtx, type Player, type World } from './sim'

export type Mode = 'solo' | 'local2' | 'host' | 'guest'
export type NetStatus = 'connecting' | 'waiting' | 'joined' | 'playing' | 'peer-left' | 'host-left' | 'error'

export interface Callbacks {
  banner(kicker: string, title: string): void
  phase(phase: 'play' | 'clear' | 'over' | 'ending', info: { stage: number; scores: number[]; fruits: number[]; twoP: boolean }): void
  net(status: NetStatus, info: { code?: string; error?: string }): void
  started(mode: Mode): void
}

export const BOSS_NAMES = ['钳钳大王', '千藤花妖', '熔岩龟王']
const STEP_MS = 1000 / 60

/** Dev only: ?relay=ws://127.0.0.1:8131 talks to a locally running room relay instead of /api. */
function relayBase(): string {
  if (!import.meta.env.DEV) return ''
  const r = new URLSearchParams(location.search).get('relay')
  return r && /^wss?:\/\/(localhost|127\.0\.0\.1)(:\d+)?$/.test(r) ? r : ''
}

/** Dev only: ?lag=150 delays every outgoing message by half of it (so the round trip grows by ~150 ms). */
function devLag(): number {
  if (!import.meta.env.DEV) return 0
  return Math.min(1000, Number(new URLSearchParams(location.search).get('lag')) || 0) / 2
}
const INTERP_MS = 95

export class IslandGame {
  readonly renderer: Renderer
  readonly audio = new GameAudio()
  readonly input = new Input()
  private cb: Callbacks
  private raf = 0
  private last = 0
  private acc = 0
  mode: Mode | null = null
  world: World | null = null
  paused = false
  private lastPhase = ''
  private lastStage = -1
  // online
  private room: Room | null = null
  private peerOn = false
  private guestQueue: { s: number; b: number }[] = []
  private guestBits = 0
  private guestAck = 0
  private netEvents: GameEvent[] = []
  private tick = 0
  // guest side
  private snaps: { s: Snapshot; at: number }[] = []
  private pendingEv: { at: number; ev: GameEvent[] }[] = []
  private pred: Player = newPlayer(1, true)
  private predOffset = { x: 0, y: 0 }
  private seq = 0
  private history: { s: number; b: number }[] = []
  private batch: number[] = []
  private batchStart = 1
  private localEv: GameEvent[] = []
  private guestStarted = false

  constructor(canvas: HTMLCanvasElement, cb: Callbacks) {
    this.renderer = new Renderer(canvas)
    this.cb = cb
  }

  async init() {
    await this.renderer.ready()
    this.last = performance.now()
    const loop = (now: number) => {
      this.raf = requestAnimationFrame(loop)
      this.frame(now)
    }
    this.raf = requestAnimationFrame(loop)
  }

  destroy() {
    cancelAnimationFrame(this.raf)
    this.input.destroy()
    this.audio.destroy()
    this.room?.close()
    this.room = null
  }

  resize(w: number, h: number) {
    this.renderer.resize(w, h, Math.min(2, window.devicePixelRatio || 1))
  }

  // ───────────── starting ─────────────

  startLocal(twoP: boolean) {
    this.leaveRoom()
    this.mode = twoP ? 'local2' : 'solo'
    this.world = createWorld(twoP)
    this.begin()
  }

  private begin() {
    this.paused = false
    this.lastPhase = ''
    this.lastStage = -1
    this.acc = 0
    this.renderer.flashWhite = 0
    if (this.mode) this.cb.started(this.mode)
  }

  hostRoom() {
    this.leaveRoom()
    this.mode = 'host'
    this.world = null
    this.peerOn = false
    this.cb.net('connecting', {})
    this.room = new Room(null, {
      room: (code) => this.cb.net('waiting', { code }),
      peer: (on) => this.hostPeer(on),
      msg: (m) => this.hostMsg(m),
      error: (code) => this.cb.net('error', { error: code }),
      closed: () => this.cb.net('error', { error: 'closed' })
    }, relayBase(), devLag())
  }

  /** Host pressed 开始: both heroes in. */
  beginOnline() {
    if (this.mode !== 'host' || !this.peerOn) return
    this.world = createWorld(true)
    this.guestQueue = []
    this.guestBits = 0
    this.room?.send({ t: 'start' })
    this.cb.net('playing', { code: this.room?.code })
    this.begin()
  }

  joinRoom(code: string) {
    this.leaveRoom()
    this.mode = 'guest'
    this.world = null
    this.snaps = []
    this.pendingEv = []
    this.history = []
    this.batch = []
    this.seq = 0
    this.guestStarted = false
    this.cb.net('connecting', { code })
    this.room = new Room(code, {
      room: (c) => this.cb.net('waiting', { code: c }),
      peer: (on) => {
        if (!on) this.cb.net('host-left', {})
      },
      msg: (m) => this.guestMsg(m),
      error: (c) => (c === 'host_left' ? this.cb.net('host-left', {}) : this.cb.net('error', { error: c })),
      closed: () => this.cb.net('host-left', {})
    }, relayBase(), devLag())
  }

  leaveRoom() {
    this.room?.close()
    this.room = null
    this.peerOn = false
  }

  quit() {
    this.leaveRoom()
    this.mode = null
    this.world = null
    this.snaps = []
    this.audio.play('title')
  }

  continue() {
    if (!this.world || (this.mode !== 'solo' && this.mode !== 'local2' && this.mode !== 'host')) return
    continueGame(this.world)
    this.lastPhase = ''
    this.lastStage = -1
  }

  /** Dev helper: jump to a spot of a stage (used for screenshots and tuning). */
  debugGoto(stage: number, x: number, god = true) {
    if (!this.world) this.startLocal(false)
    const w = this.world as World
    loadStage(w, stage, 0)
    w.god = god
    w.camX = Math.max(0, Math.min(x, w.stage.arenaX))
    for (const p of w.players) {
      p.x = w.camX + 80 + p.i * 24
      p.y = 20
    }
    w.ents = w.ents.filter((e) => e.x > w.camX - 40)
  }

  /** Dev helper: run the world synchronously (the hidden preview pane throttles rAF). */
  debugBits: ((player: number, w: World) => number) | null = null
  debugStep(frames: number) {
    const w = this.world
    if (!w) return
    for (let i = 0; i < frames; i++) this.stepLocal(w)
    this.renderer.draw(w as unknown as RView, null)
  }

  setPaused(p: boolean) {
    if (this.mode === 'host' || this.mode === 'guest') return
    this.paused = p
  }

  // ───────────── host ─────────────

  private hostPeer(on: boolean) {
    this.peerOn = on
    const w = this.world
    if (on) {
      if (w) {
        // the friend came back mid-game: bring P2 back next to P1 as a bubble
        const p2 = w.players[1]
        p2.on = true
        p2.st = 'ghost'
        p2.x = w.camX + 60
        p2.y = 40
        this.room?.send({ t: 'start' })
        this.cb.net('playing', { code: this.room?.code })
      } else this.cb.net('joined', { code: this.room?.code })
    } else {
      if (w) {
        w.players[1].on = false
        w.players[1].st = 'out'
      }
      this.cb.net('peer-left', { code: this.room?.code })
    }
  }

  private hostMsg(m: Record<string, unknown>) {
    if (m.t === 'i' && Array.isArray(m.b)) {
      const s = Number(m.s) || 0
      ;(m.b as number[]).forEach((b, i) => this.guestQueue.push({ s: s + i, b: Number(b) | 0 }))
      if (this.guestQueue.length > 8) {
        // catching up after a hiccup: keep presses, drop the backlog
        const drop = this.guestQueue.splice(0, this.guestQueue.length - 3)
        const merged = drop.reduce((acc, q) => acc | q.b, 0)
        this.guestQueue[0].b |= merged & 4
      }
    }
  }

  // ───────────── guest ─────────────

  private guestMsg(m: Record<string, unknown>) {
    if (m.t === 'start') {
      this.guestStarted = true
      this.cb.net('playing', { code: this.room?.code })
      this.begin()
      return
    }
    if (m.t !== 's') return
    const s = m as unknown as Snapshot
    const at = performance.now()
    this.snaps.push({ s, at })
    if (this.snaps.length > 12) this.snaps.shift()
    if (!this.guestStarted) {
      this.guestStarted = true
      this.cb.net('playing', { code: this.room?.code })
      this.begin()
    }
    // the host's view of us: rewind to it and replay our unacknowledged inputs
    const auth = newPlayer(1, true)
    decodePlayer(s.p[1], auth)
    const before = { x: this.pred.x, y: this.pred.y }
    const ctx = this.guestCtx(s)
    const wasPlaying = this.pred.st === 'play'
    Object.assign(this.pred, auth)
    this.history = this.history.filter((h) => h.s > s.k)
    if (auth.st === 'play') for (const h of this.history) movePlayer(this.pred, h.b, ctx)
    if (wasPlaying && auth.st === 'play') {
      this.predOffset.x += before.x - this.pred.x
      this.predOffset.y += before.y - this.pred.y
      if (Math.abs(this.predOffset.x) > 40 || Math.abs(this.predOffset.y) > 40) this.predOffset = { x: 0, y: 0 }
    } else this.predOffset = { x: 0, y: 0 }
    // our own jump/land sounds already played locally
    const ev = s.v.filter((e) => !((e[0] === EV.jump || e[0] === EV.land) && e[3] === 1))
    this.pendingEv.push({ at, ev })
  }

  private guestCtx(s: Snapshot): PhysCtx {
    const st = stages()[s.g]
    const ents = s.e.map(decodeEnt)
    return { tiles: st.tiles, cols: st.cols, camX: s.cx, plats: ents.filter((e) => e.k === 'plat'), springs: ents.filter((e) => e.k === 'spring') }
  }

  private guestTick() {
    const latest = this.snaps[this.snaps.length - 1]
    if (!latest) return
    const bits = this.paused ? 0 : this.input.bits(0, false)
    this.seq++
    this.history.push({ s: this.seq, b: bits })
    if (this.history.length > 240) this.history.shift()
    if (!this.batch.length) this.batchStart = this.seq
    this.batch.push(bits)
    if (this.batch.length >= 2) {
      this.room?.send({ t: 'i', s: this.batchStart, b: this.batch })
      this.batch = []
    }
    if (this.pred.st === 'play' && phaseOf(latest.s) === 'play') {
      const ctx = this.guestCtx(latest.s)
      this.localEv = []
      ctx.events = this.localEv
      movePlayer(this.pred, bits, ctx)
      const mine = this.localEv.filter((e) => e[0] === EV.jump || e[0] === EV.land)
      if (mine.length) {
        this.renderer.onEvents(mine)
        this.audio.sfx(mine, [1])
      }
    }
    this.predOffset.x *= 0.82
    this.predOffset.y *= 0.82
  }

  private guestView(now: number): RView | null {
    if (!this.snaps.length) return null
    const rt = now - INTERP_MS
    let a = this.snaps[0]
    let b = this.snaps[this.snaps.length - 1]
    for (let i = 0; i < this.snaps.length - 1; i++) {
      if (this.snaps[i].at <= rt && this.snaps[i + 1].at >= rt) {
        a = this.snaps[i]
        b = this.snaps[i + 1]
        break
      }
    }
    if (rt >= b.at) a = b
    const k = a === b ? 1 : Math.max(0, Math.min(1, (rt - a.at) / (b.at - a.at)))
    const lerp = (x: number, y: number) => x + (y - x) * k
    const s = b.s
    // events are fired at the moment they are shown
    while (this.pendingEv.length && this.pendingEv[0].at <= rt + 1) {
      const e = this.pendingEv.shift()
      if (e && e.ev.length) {
        this.renderer.onEvents(e.ev)
        this.audio.sfx(e.ev, [1])
        for (const x of e.ev) if (x[0] === EV.bossRoar) this.cb.banner('首领出现', BOSS_NAMES[x[3]] ?? '')
      }
    }
    const prev = new Map(a.s.e.map((t) => [t[0], t]))
    const ents: Ent[] = s.e.map((t) => {
      const e = decodeEnt(t)
      const o = prev.get(e.id)
      if (o && a !== b && o[1] === t[1]) {
        e.x = lerp(o[2], t[2])
        e.y = lerp(o[3], t[3])
      }
      return e
    })
    const players = s.p.map((t, i) => {
      const p = newPlayer(i, true)
      decodePlayer(t, p)
      const o = a.s.p[i]
      if (o && a !== b && o[1] === t[1]) {
        p.x = lerp(o[2], t[2])
        p.y = lerp(o[3], t[3])
      }
      return p
    })
    if (this.pred.st === 'play' && players[1]?.st === 'play') {
      const me = players[1]
      Object.assign(me, this.pred)
      me.x = this.pred.x + this.predOffset.x
      me.y = this.pred.y + this.predOffset.y
      // effects the host owns (weapon, fairy, hits) come from the snapshot
      decodeOwned(s.p[1], me)
    }
    return {
      stageIndex: s.g,
      camX: lerp(a.s.cx, s.cx),
      frame: Math.round(lerp(a.s.f, s.f)),
      lock: !!s.lk,
      phase: phaseOf(s),
      phaseT: s.pt,
      lives: s.lv,
      shake: s.sh,
      players,
      ents,
      bossMax: s.bm
    }
  }

  // ───────────── frame ─────────────

  private frame(now: number) {
    const dt = Math.min(250, now - this.last)
    this.last = now
    if (this.mode === 'guest') {
      this.acc += dt
      let n = 0
      while (this.acc >= STEP_MS && n < 5) {
        this.guestTick()
        this.acc -= STEP_MS
        n++
      }
      const v = this.guestView(now)
      if (v) {
        this.renderer.draw(v, 1)
        this.afterFrame(v, [1])
      }
      return
    }
    const w = this.world
    if (!w || !this.mode) return
    if (!this.paused) {
      this.acc += dt
      let n = 0
      while (this.acc >= STEP_MS && n < 5) {
        this.stepLocal(w)
        this.acc -= STEP_MS
        n++
      }
      if (n === 5) this.acc = 0
    }
    const locals = this.mode === 'host' ? [0] : this.mode === 'local2' ? [0, 1] : [0]
    this.renderer.draw(w as unknown as RView, this.mode === 'host' ? 0 : null)
    this.afterFrame(w as unknown as RView, locals)
  }

  private stepLocal(w: World) {
    const shared = this.mode === 'local2'
    let p2 = 0
    if (this.mode === 'local2') p2 = this.input.bits(1, true)
    else if (this.mode === 'host') {
      const q = this.guestQueue.shift()
      if (q) {
        this.guestBits = q.b
        this.guestAck = q.s
      }
      p2 = this.guestBits
    }
    if (this.debugBits) stepWorld(w, [this.debugBits(0, w), this.debugBits(1, w)])
    else stepWorld(w, [this.input.bits(0, shared), p2])
    const evs = w.events
    w.events = []
    if (evs.length) {
      this.renderer.onEvents(evs)
      this.audio.sfx(evs, this.mode === 'local2' ? [0, 1] : [0])
    }
    if (this.mode === 'host') {
      this.netEvents.push(...evs)
      this.tick++
      if (this.peerOn && this.tick % 2 === 0) {
        this.room?.send(makeSnapshot(w, this.guestAck, this.netEvents) as unknown as Record<string, unknown>)
        this.netEvents = []
      }
    }
    for (const e of evs) {
      if (e[0] === EV.bossRoar) this.cb.banner('首领出现', BOSS_NAMES[e[3]] ?? '')
    }
  }

  /** Phase changes, banners and music for whatever is on screen. */
  private afterFrame(v: RView, locals: number[]) {
    const stage = stages()[v.stageIndex]
    if (v.stageIndex !== this.lastStage && v.phase === 'play') {
      this.lastStage = v.stageIndex
      this.cb.banner(`第 ${v.stageIndex + 1} 关`, stage.name)
    }
    if (v.phase !== this.lastPhase) {
      this.lastPhase = v.phase
      this.cb.phase(v.phase as 'play', {
        stage: v.stageIndex,
        scores: v.players.map((p) => p.score),
        fruits: (v.players as unknown as Player[]).map((p) => p.fruits ?? 0),
        twoP: v.players.filter((p) => p.on).length > 1
      })
    }
    let track: Track = stage.theme
    if (v.phase === 'over') track = 'over'
    else if (v.phase === 'ending') track = 'ending'
    else if (v.phase === 'clear') track = 'clear'
    else if (v.lock && v.ents.some((e) => e.k === 'boss')) track = 'boss'
    else if (v.players.some((p) => locals.includes(p.i) && p.fairy > 0)) track = 'fairy'
    this.audio.play(track)
  }
}

/** Fields of the guest's hero that only the host decides. */
function decodeOwned(t: number[], me: Player) {
  me.weapon = t[8]
  me.fairy = t[9]
  me.cloud = t[10]
  me.inv = t[11]
  me.throwT = t[13]
  me.energy = t[15]
  me.score = t[16]
  me.board = !!(t[7] & 2)
}
