// 虎兔大冒险: the loop that ties world, renderer, sound, input and network together.
// Modes: 'solo' (one player), 'local' (two players, one keyboard), 'host' / 'guest' (online).

import { Audio8 } from './audio'
import { Input } from './input'
import { LEVELS, WORLD_NAMES } from './levels'
import { clonePState, stepPlayer, type PState } from './physics'
import { Renderer } from './render'
import { AreaRT, World, moverPlats, type Ev, type Phase } from './world'
import { RoomClient, fitSnapshot, packEnt, packPlayer, unpackEnt, unpackPlayer, type NetMsg, type RelayError, type SnapMsg } from './net'
import { STEP_HZ } from './types'
import type { MusicId } from './types'

export type Mode = 'solo' | 'local' | 'host' | 'guest'

export interface Hud {
  phase: Phase
  mode: Mode
  levelIdx: number
  levelId: string
  levelName: string
  worldName: string
  lives: number
  paused: boolean
  peer: boolean
  scores: number[]
  coins: number[]
}

export interface NetStatus {
  state: 'idle' | 'connecting' | 'waiting' | 'playing' | 'error'
  code: string
  seat: number
  peer: boolean
  error: RelayError | ''
}

const STEP = 1 / STEP_HZ
const INTERP_MS = 100
const SAVE_KEY = 'hutu:v1'

interface Saved {
  best: number
  reached: number
}

export function loadSave(): Saved {
  try {
    const s = JSON.parse(localStorage.getItem(SAVE_KEY) || '{}')
    return { best: Number(s.best) || 0, reached: Math.min(LEVELS.length - 1, Number(s.reached) || 0) }
  } catch {
    return { best: 0, reached: 0 }
  }
}

function save(s: Saved): void {
  try {
    localStorage.setItem(SAVE_KEY, JSON.stringify(s))
  } catch {
    /* private mode */
  }
}

export class HutuGame {
  world: World | null = null
  mode: Mode = 'solo'
  readonly renderer: Renderer
  readonly audio = new Audio8()
  readonly input = new Input()
  private raf = 0
  private last = 0
  private acc = 0
  private hudT = 0
  private lastHud = ''
  private muted = false

  // online
  room: RoomClient | null = null
  net: NetStatus = { state: 'idle', code: '', seat: -1, peer: false, error: '' }
  // host side
  private guestQueue: { s: number; b: number }[] = []
  private guestBtn = 0
  private ackSeq = -1
  private outEv: Ev[] = []
  private sendTick = 0
  private needFull = false
  // guest side
  private seq = 0
  private pending: { s: number; b: number }[] = []
  private batch: number[] = []
  private batchStart = 0
  private snaps: { at: number; m: SnapMsg }[] = []
  private pred: PState | null = null
  private predFrame = 0
  private smooth = { x: 0, y: 0 }
  private editCount = 0
  private guestLv = -1
  private guestAr = -1

  constructor(
    readonly canvas: HTMLCanvasElement,
    private onHud: (h: Hud) => void,
    private onNet: (n: NetStatus) => void = () => undefined
  ) {
    this.renderer = new Renderer(canvas)
    this.input.attach()
    this.input.onAction = (a) => {
      if (a === 'mute') this.setMuted(!this.muted)
      else this.togglePause()
    }
    this.last = performance.now()
    this.raf = requestAnimationFrame(this.frame)
    // dev builds: expose the game for play-testing from the console
    if (import.meta.env.DEV) (window as unknown as { __hutu?: HutuGame }).__hutu = this
  }

  destroy(): void {
    cancelAnimationFrame(this.raf)
    window.clearInterval(this.hiddenTimer)
    this.input.detach()
    this.audio.destroy()
    this.room?.close()
  }

  setMuted(m: boolean): void {
    this.muted = m
    this.audio.setMuted(m)
  }

  togglePause(): void {
    const w = this.world
    if (!w || this.mode === 'guest' || this.mode === 'host') return
    if (w.phase !== 'play') return
    w.paused = !w.paused
    this.audio.sfx('pause')
    this.emitHud(true)
  }

  /** start a local game (one or two players on this device) */
  startLocal(players: 1 | 2, level = 0): void {
    this.audio.start()
    this.mode = players === 2 ? 'local' : 'solo'
    this.world = new World({ players })
    if (level) this.world.restartFrom(level)
    this.renderer.onEvents(this.drain(), this.world)
  }

  // ───────────────────────── online ─────────────────────────

  /** open a room (host) or join one (guest, with a code) */
  openRoom(code?: string): void {
    this.audio.start()
    this.room?.close()
    this.net = { state: 'connecting', code: code || '', seat: -1, peer: false, error: '' }
    this.onNet({ ...this.net })
    const rc = new RoomClient({
      onRoom: (c, seat) => {
        this.net.code = c
        this.net.seat = seat
        this.net.state = seat === 0 ? 'waiting' : 'playing'
        if (seat === 1) this.beginGuest()
        this.onNet({ ...this.net })
      },
      onPeer: (on) => {
        this.net.peer = on
        if (this.net.seat === 0) {
          if (on) {
            if (!this.world || this.mode !== 'host') this.beginHost()
            this.needFull = true
            this.net.state = 'playing'
          }
        }
        this.onNet({ ...this.net })
        this.emitHud(true)
      },
      onMsg: (m) => this.onMsg(m),
      onError: (e) => {
        this.net.state = 'error'
        this.net.error = e
        this.onNet({ ...this.net })
      }
    })
    this.room = rc
    rc.connect(code)
  }

  leaveRoom(): void {
    this.room?.close()
    this.room = null
    this.net = { state: 'idle', code: '', seat: -1, peer: false, error: '' }
    this.onNet({ ...this.net })
    this.world = null
    this.audio.play('')
  }

  private beginHost(): void {
    this.mode = 'host'
    this.world = new World({ players: 2 })
    this.guestQueue = []
    this.ackSeq = -1
    this.outEv = []
    this.renderer.onEvents(this.drain(), this.world)
  }

  private beginGuest(): void {
    this.mode = 'guest'
    this.world = new World({ players: 2 })
    this.world.events.length = 0
    this.snaps = []
    this.pending = []
    this.pred = null
    this.seq = 0
    this.guestLv = -1
    this.guestAr = -1
    this.room?.send({ t: 'sync' })
  }

  private onMsg(m: NetMsg): void {
    if (this.mode === 'host') {
      if (m.t === 'i') {
        for (let k = 0; k < m.b.length; k++) this.guestQueue.push({ s: m.s + k, b: m.b[k] })
      } else if (m.t === 'sync') this.needFull = true
      return
    }
    if (this.mode !== 'guest' || !this.world) return
    if (m.t === 'full') {
      this.guestArea(m.lv, m.ar)
      for (const [i, c] of m.edits) this.world.area.cells[i] = c
      this.editCount = m.edits.length
    } else if (m.t === 's') this.onSnap(m)
  }

  private guestArea(lv: number, ar: number): void {
    const w = this.world as World
    w.levelIdx = lv
    w.level = LEVELS[lv]
    w.areaIdx = ar
    w.area = new AreaRT(w.level.areas[ar])
    this.guestLv = lv
    this.guestAr = ar
    this.editCount = 0
  }

  private onSnap(m: SnapMsg): void {
    const w = this.world as World
    for (const ev of m.ev) {
      if (ev.e === 'area') this.guestArea(ev.lv, ev.ar)
      else if (ev.e === 'edit') {
        w.area.cells[ev.i] = ev.c
        this.editCount++
      }
    }
    if (m.lv !== this.guestLv || m.ar !== this.guestAr) {
      this.guestArea(m.lv, m.ar)
      this.room?.send({ t: 'sync' })
    } else if (this.editCount !== m.en) {
      this.editCount = m.en
      this.room?.send({ t: 'sync' })
    }
    this.handleEvents(m.ev.filter((e) => e.e !== 'area' && e.e !== 'edit' && e.e !== 'music'))
    this.audio.play((m.mu || '') as MusicId | '')
    this.snaps.push({ at: performance.now(), m })
    if (this.snaps.length > 30) this.snaps.shift()

    // reconcile the predicted hero
    const mine = m.p[1]
    if (!mine) return
    this.pending = this.pending.filter((p) => p.s > m.a)
    const tmp = w.players[1]
    unpackPlayer(mine, tmp)
    if (tmp.st !== 'play') {
      this.pred = null
      return
    }
    const before = this.pred ? { x: this.pred.x, y: this.pred.y } : null
    const p = clonePState(tmp.ps)
    let f = m.f
    for (const inp of this.pending) {
      stepPlayer(p, inp.b, w.area, moverPlats(w.area.def.movers, f))
      f++
    }
    this.pred = p
    this.predFrame = f
    if (before) {
      this.smooth.x += before.x - p.x
      this.smooth.y += before.y - p.y
      if (Math.abs(this.smooth.x) > 48 || Math.abs(this.smooth.y) > 48) this.smooth = { x: 0, y: 0 }
    }
  }

  // ───────────────────────── loop ─────────────────────────

  private frame = (now: number): void => {
    this.raf = requestAnimationFrame(this.frame)
    this.lastRaf = performance.now()
    this.advance(now, 6)
    this.draw(now)
    if (++this.hudT % 6 === 0) this.emitHud(false)
  }

  private advance(now: number, maxSteps: number): void {
    const dt = Math.min(0.25, (now - this.last) / 1000)
    this.last = now
    this.acc += dt
    let n = 0
    while (this.acc >= STEP && n < maxSteps) {
      this.step()
      this.acc -= STEP
      n++
    }
    if (n >= maxSteps) this.acc = 0
  }

  /**
   * Browsers stop animation frames in background tabs (and hidden windows). Online, the host must keep the world
   * running (and the guest keep sending input) when the tab is hidden, so a timer takes over.
   */
  private lastRaf = 0
  private hiddenTimer = window.setInterval(() => {
    if (this.mode !== 'host' && this.mode !== 'guest') return
    const now = performance.now()
    if (now - this.lastRaf < 150) return
    this.advance(now, 16)
  }, 50)

  private drain(): Ev[] {
    const w = this.world
    if (!w) return []
    const evs = w.events.slice()
    w.events.length = 0
    return evs
  }

  private handleEvents(evs: Ev[]): void {
    const w = this.world
    if (!w) return
    for (const e of evs) {
      if (e.e === 'sfx') this.audio.sfx(e.s)
      else if (e.e === 'music') this.audio.play(e.m)
    }
    this.renderer.onEvents(evs, w)
  }

  private step(): void {
    const w = this.world
    if (!w) return
    if (this.mode === 'guest') {
      this.guestStep()
      return
    }
    const b0 = this.input.read(0, this.mode === 'local')
    let b1 = 0
    if (this.mode === 'local') b1 = this.input.read(1, true)
    else if (this.mode === 'host') {
      const next = this.guestQueue.shift()
      if (next) {
        this.guestBtn = next.b
        this.ackSeq = next.s
      }
      // fell behind (lag spike): catch up rather than drift further back
      while (this.guestQueue.length > 6) {
        const n2 = this.guestQueue.shift()
        if (n2) {
          this.guestBtn = n2.b
          this.ackSeq = n2.s
        }
      }
      b1 = this.net.peer ? this.guestBtn : 0
    }
    const wasPhase = w.phase
    w.tick([b0, b1])
    const evs = this.drain()
    this.handleEvents(evs)
    if (this.mode === 'host') {
      this.outEv.push(...evs)
      if (++this.sendTick % 2 === 0) this.sendSnap()
    }
    if (w.phase === 'clear' && wasPhase !== 'clear') this.recordProgress(w.levelIdx + 1)
    if (w.phase === 'won' && wasPhase !== 'won') this.recordProgress(LEVELS.length - 1)
  }

  private recordProgress(reached: number): void {
    const w = this.world
    if (!w) return
    const s = loadSave()
    s.reached = Math.max(s.reached, Math.min(LEVELS.length - 1, reached))
    s.best = Math.max(s.best, ...w.players.map((p) => Math.floor(p.score)))
    save(s)
  }

  private sendSnap(): void {
    const w = this.world
    if (!w || !this.room || !this.net.peer) {
      this.outEv = []
      return
    }
    if (this.needFull) {
      this.needFull = false
      this.room.send({ t: 'full', lv: w.levelIdx, ar: w.areaIdx, edits: w.editLog })
      this.outEv = this.outEv.filter((e) => e.e !== 'edit' && e.e !== 'area')
    }
    const m: SnapMsg = {
      t: 's',
      f: w.frame,
      ph: w.phase,
      pt: w.phaseT,
      lv: w.levelIdx,
      ar: w.areaIdx,
      cam: Math.round(w.cam * 10) / 10,
      time: w.time,
      lives: w.lives,
      cp: w.checkpointHit ? 1 : 0,
      en: w.editLog.length,
      mu: w.musicNow,
      paused: 0,
      a: this.ackSeq,
      p: w.players.map(packPlayer),
      e: w.ents.filter((e) => e.x > w.cam - 120 && e.x < w.cam + 560).map(packEnt),
      ev: this.outEv.filter((e) => e.e !== 'music')
    }
    this.outEv = []
    this.room.sendRaw(fitSnapshot(m))
  }

  private guestStep(): void {
    const w = this.world as World
    const b = this.input.read(0, false)
    this.seq++
    this.pending.push({ s: this.seq, b })
    if (this.batch.length === 0) this.batchStart = this.seq
    this.batch.push(b)
    if (this.batch.length >= 2) {
      this.room?.send({ t: 'i', s: this.batchStart, b: this.batch })
      this.batch = []
    }
    if (this.pending.length > 120) this.pending.shift()
    if (this.pred && this.snaps.length) {
      stepPlayer(this.pred, b, w.area, moverPlats(w.area.def.movers, this.predFrame))
      this.predFrame++
      // the camera's left edge pushes on the host too; mirror it so prediction doesn't fight it
      const minX = w.cam + 6
      if (this.pred.x < minX) this.pred.x = minX
    }
    this.smooth.x *= 0.85
    this.smooth.y *= 0.85
  }

  /** fill the guest's World from the two snapshots around (now - delay) */
  private guestInterpolate(now: number): void {
    const w = this.world as World
    if (!this.snaps.length) return
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
    if (rt > b.at) a = b
    const span = b.at - a.at
    const k = span > 0 ? Math.max(0, Math.min(1, (rt - a.at) / span)) : 1
    const A = a.m
    const B = b.m
    const lerp = (x: number, y: number) => x + (y - x) * k
    w.phase = B.ph
    w.phaseT = B.pt
    w.frame = lerp(A.f, B.f)
    w.cam = lerp(A.cam, B.cam)
    w.time = B.time
    w.lives = B.lives
    w.checkpointHit = !!B.cp
    w.paused = false
    // players
    B.p.forEach((t, i) => {
      const pl = w.players[i]
      unpackPlayer(t, pl)
      const at = A.p[i]
      if (at) {
        pl.ps.x = lerp(at[0] as number, t[0] as number)
        pl.ps.y = lerp(at[1] as number, t[1] as number)
      }
    })
    // entities, matched by id
    const prev = new Map<number, (number | string)[]>()
    for (const e of A.e) prev.set(e[0] as number, e)
    w.ents = B.e.map((t) => {
      const e = unpackEnt(t)
      const pe = prev.get(e.id)
      if (pe) {
        e.x = lerp(pe[2] as number, e.x)
        e.y = lerp(pe[3] as number, e.y)
      }
      return e
    })
    // my own hero: predicted
    const me = w.players[1]
    if (this.pred && me.st === 'play') {
      me.ps = { ...this.pred, x: this.pred.x + this.smooth.x, y: this.pred.y + this.smooth.y }
      // animation from the predicted state, so it matches what the player is doing now
      const ps = this.pred
      if (!ps.ground) me.anim = ps.vy < 0 ? 'jump' : 'fall'
      else if (ps.crouch) me.anim = 'crouch'
      else if (ps.skid) me.anim = 'skid'
      else if (Math.abs(ps.vx) > 0.05) me.anim = 'walk'
      else me.anim = 'stand'
    }
    // pick up prediction again once back in play (after a respawn or a tube)
    if (!this.pred && me.st === 'play') this.pred = clonePState(me.ps)
  }

  private draw(now: number): void {
    const w = this.world
    if (!w) return
    if (this.mode === 'guest') this.guestInterpolate(now)
    const local = this.mode === 'guest' ? [1] : this.mode === 'host' ? [0] : [0, 1]
    this.renderer.draw(w, local)
  }

  private emitHud(force: boolean): void {
    const w = this.world
    if (!w) return
    const h: Hud = {
      phase: w.phase,
      mode: this.mode,
      levelIdx: w.levelIdx,
      levelId: w.level.id,
      levelName: w.level.name,
      worldName: WORLD_NAMES[w.level.world] || '',
      lives: w.lives,
      paused: w.paused,
      peer: this.net.peer,
      scores: w.players.map((p) => Math.floor(p.score)),
      coins: w.players.map((p) => p.coins)
    }
    const key = JSON.stringify(h)
    if (!force && key === this.lastHud) return
    this.lastHud = key
    this.onHud(h)
  }

  /** after game over (local or host): try again from this level */
  continueGame(): void {
    if (!this.world || this.mode === 'guest') return
    this.world.continueGame()
  }
}
