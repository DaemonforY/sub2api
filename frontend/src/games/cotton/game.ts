// 棉花糖大作战: the game loop. Runs the simulation at a fixed 60 Hz for solo, same-keyboard
// and online-host games, or follows the host's snapshots as an online guest, and tells the page
// what to show (menus, room code, floor results, the final score).

import { CottonAudio } from './audio'
import { botButtons } from './bot'
import { Controls } from './input'
import { FLOORS, FLOOR_COUNT, worldOf } from './levels'
import { applyReport, blend, decodeEvents, encode, guestState, report, Room, type GuestReport, type RoomError, type Snapshot } from './net'
import { Renderer } from './render'
import {
  ALIVE,
  continueGame,
  createGame,
  fire,
  joinPlayer,
  leavePlayer,
  movePlayer,
  NO_BUTTONS,
  SNOW_FULL,
  step,
  type Buttons,
  type Ev,
  type FloorResult,
  type GameState,
  type Phase,
  type Player
} from './sim'

export type Mode = 'solo' | 'duo' | 'host' | 'guest'
export type Screen = 'title' | 'lobby' | 'play'

export interface RoomUi {
  code: string
  seat: number
  peer: boolean
  status: 'connecting' | 'open' | 'error' | 'closed'
  error: RoomError | ''
}

export interface KidUi {
  on: boolean
  score: number
  kills: number
  bestChain: number
  lives: number
}

export interface Ui {
  screen: Screen
  mode: Mode | null
  paused: boolean
  phase: Phase
  /** frames since the phase began */
  phaseT: number
  floor: number
  reached: number
  seconds: number
  result: FloorResult | null
  kids: [KidUi, KidUi]
  room: RoomUi | null
  ping: number
  /** a short line for the page to toast */
  notice: string
  /** goes up each time a new run starts (an online guest uses it to leave the final screen) */
  round: number
}

export interface Hooks {
  ui(u: Ui): void
  finish(u: Ui): void
}

const STEP = 1000 / 60
const INTERP = 110

interface Buffered {
  at: number
  snap: Snapshot
  evs: Ev[]
}

interface Predicted {
  x: number
  y: number
  vx: number
  vy: number
  t: number
  life: number
  big: boolean
}

export class CottonGame {
  readonly renderer: Renderer
  readonly audio = new CottonAudio()
  private controls = new Controls()
  private s: GameState | null = null
  private mode: Mode | null = null
  private screen: Screen = 'title'
  private paused = false
  private raf = 0
  private last = 0
  private acc = 0
  private room: Room | null = null
  private roomUi: RoomUi | null = null
  private ping = 0
  private notice = ''
  private uiKey = ''
  private uiTick = 0
  private finished = false
  private round = 0
  // host
  private guestReport: GuestReport | null = null
  private guestFires = 0
  private netEvents: Ev[] = []
  private sendTick = 0
  // guest
  private snaps: Buffered[] = []
  private view: GameState | null = null
  private me: Player | null = null
  private shadow: Player | null = null
  private myFires = 0
  private predicted: Predicted[] = []
  private pending: Buffered[] = []
  private lastPing = 0
  private worker: Worker | null = null
  private rafAt = 0
  private disposed = false
  /** dev only: let a simple bot play, for screenshots and soak tests */
  bot = false

  constructor(
    canvas: HTMLCanvasElement,
    private hooks: Hooks,
    mediaBase: string
  ) {
    this.renderer = new Renderer(canvas, mediaBase)
    this.renderer.resize()
    this.controls.attach()
    document.addEventListener('visibilitychange', this.onVisibility)
    this.raf = requestAnimationFrame(this.loop)
  }

  // ------------------------------------------------------------ page actions

  start(mode: 'solo' | 'duo'): void {
    this.leaveRoom()
    this.audio.start()
    this.mode = mode
    this.s = createGame(mode === 'duo' ? 2 : 1)
    this.screen = 'play'
    this.paused = false
    this.finished = false
    this.renderer.hitstop = 0
    this.pushUi(true)
  }

  host(): void {
    this.leaveRoom()
    this.audio.start()
    this.mode = 'host'
    this.screen = 'lobby'
    this.s = null
    this.roomUi = { code: '', seat: 0, peer: false, status: 'connecting', error: '' }
    this.room = new Room(this.roomHooks())
    this.startTicker()
    this.pushUi(true)
  }

  join(code: string): void {
    this.leaveRoom()
    this.audio.start()
    this.mode = 'guest'
    this.screen = 'lobby'
    this.s = null
    this.view = null
    this.me = null
    this.shadow = null
    this.snaps = []
    this.pending = []
    this.predicted = []
    this.myFires = 0
    this.roomUi = { code: code.toUpperCase(), seat: 1, peer: false, status: 'connecting', error: '' }
    this.room = new Room(this.roomHooks(), code)
    this.startTicker()
    this.pushUi(true)
  }

  /** host: both kids are in the room, go */
  hostStart(): void {
    if (this.mode !== 'host' || !this.room) return
    const s = createGame(this.room.peer ? 2 : 1)
    s.players[1].remote = true
    this.s = s
    this.screen = 'play'
    this.finished = false
    this.guestFires = 0
    this.guestReport = null
    this.renderer.hitstop = 0
    this.round++
    this.room.send({ t: 'go', round: this.round })
    this.pushUi(true)
  }

  continueRun(): void {
    if (this.mode === 'guest') {
      this.room?.send({ t: 'cont' })
      return
    }
    if (!this.s || this.s.phase !== 'over') return
    continueGame(this.s)
    this.finished = false
    this.pushUi(true)
  }

  /** stop after a game over or the ending, and show the final score */
  endRun(): void {
    if (this.mode === 'guest') {
      this.room?.send({ t: 'end' })
      return
    }
    if (this.mode === 'host') this.room?.send({ t: 'end' })
    this.finish()
  }

  toTitle(): void {
    this.leaveRoom()
    this.mode = null
    this.s = null
    this.view = null
    this.screen = 'title'
    this.paused = false
    this.pushUi(true)
  }

  setPaused(p: boolean): void {
    if (this.mode === 'host' || this.mode === 'guest') return
    this.paused = p
    this.pushUi(true)
  }

  setMuted(m: boolean): void {
    this.audio.setMuted(m)
  }

  setTouch(b: Partial<Buttons>): void {
    this.controls.setTouch(b)
  }

  setScanlines(on: boolean): void {
    this.renderer.scanlines = on
  }

  resize(): void {
    this.renderer.resize()
  }

  dispose(): void {
    this.disposed = true
    cancelAnimationFrame(this.raf)
    this.controls.detach()
    document.removeEventListener('visibilitychange', this.onVisibility)
    this.worker?.terminate()
    this.leaveRoom()
    this.audio.dispose()
  }

  /** for the share card and tests */
  get state(): GameState | null {
    return this.mode === 'guest' ? this.view : this.s
  }

  // ------------------------------------------------------------ room

  private roomHooks() {
    return {
      room: (code: string, seat: number) => {
        if (!this.roomUi) return
        this.roomUi.code = code
        this.roomUi.seat = seat
        this.roomUi.status = 'open'
        this.pushUi(true)
      },
      peer: (on: boolean) => {
        if (!this.roomUi) return
        this.roomUi.peer = on
        if (this.mode === 'host') {
          if (on) {
            this.notice = '队友进来了'
            if (this.s && this.screen === 'play') {
              joinPlayer(this.s, 1)
              this.s.players[1].remote = true
              this.guestFires = 0
              this.guestReport = null
              this.room?.send({ t: 'go', round: this.round })
            }
          } else {
            this.notice = this.s ? `队友掉线了，用房间码 ${this.roomUi.code} 可以重新加入` : '队友离开了房间'
            if (this.s) leavePlayer(this.s, 1)
          }
        }
        this.pushUi(true)
      },
      error: (code: RoomError) => {
        if (!this.roomUi) return
        this.roomUi.status = 'error'
        this.roomUi.error = code
        this.lostRoom()
      },
      message: (msg: Record<string, unknown>) => this.onMessage(msg),
      closed: () => {
        if (!this.roomUi) return
        this.roomUi.status = 'closed'
        this.roomUi.error = 'closed'
        this.lostRoom()
      }
    }
  }

  /** the room is gone: a guest goes back to the lobby card with the reason, a host plays on alone */
  private lostRoom(): void {
    if (this.mode === 'guest') {
      this.screen = 'lobby'
      this.view = null
    } else if (this.mode === 'host' && this.s && this.screen === 'play') {
      this.notice = '联机断开了，你可以接着一个人玩'
      if (this.s.players[1].on) leavePlayer(this.s, 1)
    }
    this.pushUi(true)
  }

  private leaveRoom(): void {
    this.room?.close()
    this.room = null
    this.roomUi = null
    this.ping = 0
    this.worker?.terminate()
    this.worker = null
  }

  private onMessage(msg: Record<string, unknown>): void {
    const t = msg.t
    if (t === 'ping') {
      this.room?.send({ t: 'pong', ts: msg.ts })
      return
    }
    if (t === 'pong') {
      this.ping = Math.round(performance.now() - Number(msg.ts))
      return
    }
    if (this.mode === 'host') {
      if (t === 'g') this.guestReport = msg as unknown as GuestReport
      else if (t === 'cont') this.continueRun()
      else if (t === 'end') this.endRun()
      else if (t === 'hello' && this.s) this.room?.send({ t: 'go', round: this.round })
      return
    }
    if (this.mode === 'guest') {
      if (t === 's') {
        const snap = msg as unknown as Snapshot
        const b: Buffered = { at: performance.now(), snap, evs: decodeEvents(snap) }
        this.snaps.push(b)
        if (this.snaps.length > 12) this.snaps.shift()
        this.pending.push(b)
        if (this.screen !== 'play') {
          this.screen = 'play'
          this.finished = false
          this.pushUi(true)
        }
      } else if (t === 'go') {
        this.finished = false
        if (typeof msg.round === 'number' && msg.round !== this.round) {
          // the host started a new run: drop the old picture
          this.round = msg.round
          this.snaps = []
          this.pending = []
          this.predicted = []
          this.me = null
          this.shadow = null
          this.view = null
          this.pushUi(true)
        }
        if (this.screen !== 'play') {
          this.screen = 'play'
          this.pushUi(true)
        }
      } else if (t === 'end') this.finish()
    }
  }

  private onVisibility = () => {
    if (!document.hidden) this.last = performance.now()
  }

  /**
   * A tab in the background (or a hidden window) gets few or no animation frames, which would
   * freeze an online game for both players. While online, a worker ticks every frame and the
   * game advances from it whenever animation frames have stalled.
   */
  private startTicker(): void {
    if (this.worker || typeof Worker === 'undefined') return
    try {
      this.worker = new Worker(new URL('./ticker.ts', import.meta.url), { type: 'module' })
      this.worker.onmessage = () => {
        const now = performance.now()
        if (now - this.rafAt > 120) this.advance(now, false)
      }
    } catch {
      this.worker = null
    }
  }

  // ------------------------------------------------------------ loop

  private loop = () => {
    if (this.disposed) return
    this.raf = requestAnimationFrame(this.loop)
    this.rafAt = performance.now()
    this.advance(performance.now(), true)
  }

  private advance(now: number, draw: boolean): void {
    if (!this.last) this.last = now
    let dt = now - this.last
    this.last = now
    if (dt > 250) dt = 250
    this.acc += dt
    let steps = 0
    while (this.acc >= STEP && steps < 6) {
      this.acc -= STEP
      steps++
      this.frame()
    }
    if (steps === 6) this.acc = 0
    if (draw) this.paint()
    this.music()
    if (++this.uiTick % 6 === 0) this.pushUi(false)
  }

  private frame(): void {
    if (!this.paused) this.renderer.fxStep()
    if (this.mode === 'guest') {
      this.guestFrame()
      this.controls.endFrame()
      return
    }
    const s = this.s
    if (!s || this.screen !== 'play' || this.paused) {
      this.controls.endFrame()
      return
    }
    if (this.controls.pressed.includes('Escape') || this.controls.pressed.includes('KeyP')) {
      if (this.mode !== 'host') {
        this.setPaused(true)
        this.controls.endFrame()
        return
      }
    }
    const solo = this.mode !== 'duo'
    let [a, b] = this.controls.poll(solo)
    if (this.bot) {
      a = botButtons(s, 0)
      if (this.mode === 'duo') b = botButtons(s, 1)
    }
    if (this.mode === 'host') {
      const p2 = s.players[1]
      if (this.guestReport) {
        const n = applyReport(p2, this.guestReport, this.guestFires)
        const fires = Math.min(4, n - this.guestFires)
        this.guestFires = n
        for (let i = 0; i < fires; i++) {
          p2.cd = 0
          fire(s, p2)
        }
        this.guestReport = null
      }
    }
    if (this.renderer.hitstop > 0) this.renderer.hitstop--
    else step(s, [a, this.mode === 'duo' ? b : NO_BUTTONS])
    const evs = s.events
    s.events = []
    this.renderer.react(evs, s)
    this.audio.react(evs, this.mode === 'duo' ? [0, 1] : [0])
    if (this.mode === 'host' && this.room) {
      this.netEvents.push(...evs)
      if (++this.sendTick % 3 === 0) {
        this.room.send(encode(s, this.netEvents) as unknown as Record<string, unknown>)
        this.netEvents = []
      }
      this.pingTick()
    }
    // the ending plays for a while, then the final score
    if (s.phase === 'won' && s.phaseT === 400) this.endRun()
    this.controls.endFrame()
  }

  private pingTick(): void {
    const now = performance.now()
    if (this.room?.peer && now - this.lastPing > 2000) {
      this.lastPing = now
      this.room.send({ t: 'ping', ts: now })
    }
  }

  // ------------------------------------------------------------ guest

  private guestFrame(): void {
    if (!this.room) return
    this.pingTick()
    if (this.screen !== 'play' || this.snaps.length === 0) return
    const seat = Math.max(1, this.room.seat) as 0 | 1
    const view = (this.view ??= guestState())
    const renderAt = performance.now() - INTERP
    // pick the two snapshots around the render time
    let a = this.snaps[0]
    let b = this.snaps[this.snaps.length - 1]
    for (let i = this.snaps.length - 1; i > 0; i--) {
      if (this.snaps[i - 1].at <= renderAt) {
        a = this.snaps[i - 1]
        b = this.snaps[i]
        break
      }
    }
    const span = b.at - a.at
    const alpha = span > 0 ? Math.max(0, Math.min(1, (renderAt - a.at) / span)) : 1
    // blend writes the host's account of our kid into a spare object, never into `me`
    if (this.me && view.players[seat] === this.me) view.players[seat] = this.shadow ?? (this.shadow = { ...this.me })
    blend(view, a.snap, b.snap, alpha)
    // events play when the picture reaches them
    const due = this.pending.filter((p) => p.at <= renderAt + 16)
    this.pending = this.pending.filter((p) => p.at > renderAt + 16)
    const latest = this.snaps[this.snaps.length - 1].snap
    const host = latest.p[seat]
    if (!this.me) this.me = { ...view.players[seat] }
    const me = this.me
    const hostAlive = host && host[1] === ALIVE
    if (!hostAlive || me.epoch !== host[18] || me.state !== ALIVE) {
      // dead, respawned, revived or a new floor: take the host's word for where the kid is
      Object.assign(me, view.players[seat])
      if (host) {
        me.x = host[2]
        me.y = host[3]
        me.epoch = host[18]
        me.state = host[1]
      }
      me.jumpHeld = true
      me.fireHeld = true
    }
    // keep potions and score in step with the host
    const hv = view.players[seat]
    me.speed = hv.speed
    me.range = hv.range
    me.power = hv.power
    me.lives = hv.lives
    me.score = hv.score
    me.inv = hv.inv
    me.on = hv.on
    for (const d of due) {
      for (const e of d.evs) {
        if (e.k === 'bump' && e.p === seat && me.state === ALIVE) {
          me.vy = -3.6
          me.bump = 22
          me.ground = false
        }
        if (e.k === 'quake' && me.ground && me.y + me.h >= 223 && me.state === ALIVE) {
          me.vy = -2.2
          me.ground = false
        }
      }
      const evs = d.evs.filter((e) => !(e.k === 'shot' && e.p === seat))
      this.renderer.react(evs, view)
      this.audio.react(evs, [seat])
    }
    const [btn] = this.controls.poll(true)
    const live = me.state === ALIVE && view.phase !== 'result' && view.phase !== 'over' && view.phase !== 'won'
    if (live) {
      view.events = []
      movePlayer(view, me, btn)
      const pressed = btn.fire && !me.fireHeld
      me.fireHeld = btn.fire
      if (me.cd > 0) me.cd--
      if (pressed || (btn.fire && --me.auto <= 0)) {
        me.auto = pressed ? 16 : 14
        if (me.cd <= 0) {
          me.cd = 8
          this.myFires++
          this.predictFire(view, me)
        }
      }
      this.renderer.react(view.events, view)
      this.audio.react(view.events, [seat])
      view.events = []
    }
    view.players[seat] = me
    this.stepPredicted(view)
    view.shots = view.shots.filter((x) => x.o !== seat)
    for (const p of this.predicted) view.shots.push({ id: -1, o: seat, x: p.x, y: p.y, vx: p.vx, vy: p.vy, t: p.t, life: p.life, big: p.big })
    if (++this.sendTick % 2 === 0) this.room.send(report(me, this.myFires) as unknown as Record<string, unknown>)
  }

  private predictFire(view: GameState, me: Player): void {
    const reach = { x: me.face > 0 ? me.x + 2 : me.x - 8, y: me.y + 2, w: me.w + 6, h: me.h - 2 }
    const ball = view.enemies.find(
      (e) => e.snow >= SNOW_FULL && !e.rolling && !e.dying && !e.boss && reach.x < e.x + e.w && e.x < reach.x + reach.w && reach.y < e.y + e.h && e.y < reach.y + reach.h
    )
    if (ball) {
      me.kickT = 14
      return
    }
    if (this.predicted.length >= 3) return
    me.shootT = 10
    this.predicted.push({ x: me.face > 0 ? me.x + me.w - 1 : me.x - 7, y: me.y + 4, vx: me.face * (me.range ? 4.2 : 3.6), vy: 0, t: 0, life: me.range ? 30 : 17, big: me.power })
    this.renderer.react([{ k: 'shot', x: me.x + me.w / 2 + me.face * 8, y: me.y + 7, p: me.idx }], view)
    this.audio.react([{ k: 'shot', x: 0, y: 0, p: me.idx }], [me.idx])
  }

  private stepPredicted(view: GameState): void {
    this.predicted = this.predicted.filter((p) => {
      p.t++
      p.x += p.vx
      if (p.t > p.life * 0.55) p.vy += 0.09
      p.y += p.vy
      if (p.t > p.life) return false
      for (const e of view.enemies) {
        if (e.dying || e.spawnT > 0 || e.rolling || e.kind === 'slipper') continue
        if (p.x < e.x + e.w && e.x < p.x + 7 && p.y < e.y + e.h && e.y < p.y + 6) return false
      }
      return true
    })
  }

  // ------------------------------------------------------------ drawing, music, page state

  private paint(): void {
    const st = this.state
    if (!st || this.screen !== 'play') return
    const local = this.mode === 'duo' ? [0, 1] : this.mode === 'guest' ? [Math.max(1, this.room?.seat ?? 1)] : [0]
    this.renderer.draw(st, local, this.mode === 'host' || this.mode === 'guest' ? { ping: this.ping } : null)
  }

  private music(): void {
    const st = this.state
    if (this.screen !== 'play' || !st) {
      this.audio.play('title')
      return
    }
    if (st.phase === 'over') this.audio.play('over')
    else if (st.phase === 'won') this.audio.play('won')
    else if (st.phase === 'clear' || st.phase === 'result') this.audio.play('clear')
    else if (FLOORS[st.floor]?.boss) this.audio.play('boss')
    else this.audio.play(['bedroom', 'classroom', 'rooftop'][worldOf(st.floor).song - 1])
  }

  private finish(): void {
    if (this.finished) return
    this.finished = true
    this.hooks.finish(this.snapshotUi())
  }

  private snapshotUi(): Ui {
    const st = this.state
    const kid = (p: Player | undefined): KidUi => ({ on: !!p?.on, score: p?.score ?? 0, kills: p?.kills ?? 0, bestChain: p?.bestChain ?? 0, lives: p?.lives ?? 0 })
    return {
      screen: this.screen,
      mode: this.mode,
      paused: this.paused,
      phase: st?.phase ?? 'intro',
      phaseT: st?.phaseT ?? 0,
      floor: st?.floor ?? 0,
      reached: st?.reached ?? 1,
      seconds: Math.round((st?.played ?? 0) / 60),
      result: st?.result ?? null,
      kids: [kid(st?.players[0]), kid(st?.players[1])],
      room: this.roomUi ? { ...this.roomUi } : null,
      ping: this.ping,
      notice: this.notice,
      round: this.round
    }
  }

  private pushUi(force: boolean): void {
    const u = this.snapshotUi()
    const key = JSON.stringify([u.screen, u.mode, u.paused, u.phase, u.phase === 'over' || u.phase === 'won' ? Math.min(u.phaseT, 90) : 0, u.floor, u.result, u.room, u.notice, u.kids, Math.round(u.ping / 20), u.round])
    if (!force && key === this.uiKey) return
    this.uiKey = key
    this.hooks.ui(u)
    this.notice = ''
  }
}

export const FLOORS_TOTAL = FLOOR_COUNT
