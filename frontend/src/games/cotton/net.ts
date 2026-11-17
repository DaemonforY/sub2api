// Online 双排 for 棉花糖大作战 over the site's shared game-room relay.
//
// The host runs the whole game and sends a snapshot 20 times a second; the guest moves its own
// kid locally (so jumping feels instant at 50–150 ms from Seoul) and reports where it is 30
// times a second, plus how many times it pressed fire. Everything else the guest sees — pests,
// balls, shots, items, the other kid — is drawn ~100 ms in the past, blended between snapshots.

import {
  ALIVE,
  buildTiles,
  createGame,
  type Enemy,
  type Ev,
  type EvKind,
  type GameState,
  type Item,
  type Kind,
  type Phase,
  type Player,
  type Proj,
  type ProjKind,
  type Shot
} from './sim'

export const GAME_SLUG = 'cotton'

export type RoomError = 'not_found' | 'room_full' | 'bad_game' | 'too_big' | 'rate' | 'host_left' | 'closed' | 'network'

export interface RoomHooks {
  room(code: string, seat: number): void
  peer(on: boolean): void
  error(code: RoomError): void
  message(msg: Record<string, unknown>): void
  closed(): void
}

export class Room {
  private ws: WebSocket | null = null
  code = ''
  seat = -1
  peer = false
  private gotError = false

  constructor(
    hooks: RoomHooks,
    join?: string
  ) {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const q = new URLSearchParams({ game: GAME_SLUG })
    if (join) q.set('room', join.toUpperCase())
    let ws: WebSocket
    try {
      ws = new WebSocket(`${proto}://${location.host}/api/v1/games/rooms/ws?${q}`)
    } catch {
      queueMicrotask(() => hooks.error('network'))
      return
    }
    this.ws = ws
    ws.onmessage = (ev) => {
      let msg: Record<string, unknown>
      try {
        msg = JSON.parse(String(ev.data))
      } catch {
        return
      }
      if (msg.sys === 'room') {
        this.code = String(msg.code)
        this.seat = Number(msg.seat)
        hooks.room(this.code, this.seat)
      } else if (msg.sys === 'peer') {
        this.peer = !!msg.on
        hooks.peer(this.peer)
      } else if (msg.sys === 'error') {
        this.gotError = true
        hooks.error(String(msg.code) as RoomError)
      } else if (!msg.sys) hooks.message(msg)
    }
    ws.onerror = () => {
      if (!this.gotError && this.seat < 0) {
        this.gotError = true
        hooks.error('network')
      }
    }
    ws.onclose = () => {
      this.ws = null
      if (!this.gotError) hooks.closed()
    }
  }

  get open(): boolean {
    return !!this.ws && this.ws.readyState === WebSocket.OPEN
  }

  send(msg: Record<string, unknown>): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(msg))
  }

  close(): void {
    this.gotError = true
    this.ws?.close()
    this.ws = null
  }
}

// ---------------------------------------------------------------- snapshot codec

const PHASES: Phase[] = ['intro', 'play', 'clear', 'result', 'over', 'won']
const KINDS: Kind[] = ['book', 'clock', 'mosquito', 'chalk', 'coil', 'slipper', 'bigclock', 'bookpile', 'calendar']
const PROJS: ProjKind[] = ['chalk', 'wave', 'plane', 'page']
const EVS: EvKind[] = [
  'jump',
  'shot',
  'hit',
  'ball',
  'nudge',
  'kick',
  'bounce',
  'kill',
  'burst',
  'thaw',
  'pickup',
  'die',
  'revive',
  'respawn',
  'bump',
  'land',
  'hurry',
  'slipper',
  'throw',
  'quake',
  'ring',
  'bosshit',
  'bossdie',
  'sweep',
  'clear',
  'floor',
  'over',
  'won',
  'poof',
  'spawn',
  'push'
]

const r1 = (v: number) => Math.round(v * 10) / 10
const r2 = (v: number) => Math.round(v * 100) / 100

export type Snapshot = {
  t: 's'
  f: number
  fl: number
  ph: number
  pt: number
  ft: number
  c: number
  sw: number
  re: number
  pl: number
  r: number[] | 0
  p: number[][]
  e: number[][]
  s: number[][]
  j: number[][]
  i: number[][]
  ev: number[][]
}

/** host: the state as a compact JSON-able object, plus the events since the last one */
export function encode(s: GameState, evs: Ev[]): Snapshot {
  return {
    t: 's',
    f: s.frame,
    fl: s.floor,
    ph: PHASES.indexOf(s.phase),
    pt: s.phaseT,
    ft: s.floorT,
    c: s.count,
    sw: s.sweep ? 1 : 0,
    re: s.reached,
    pl: s.played,
    r: s.result ? [s.result.seconds, s.result.bonus, s.result.floorScore[0], s.result.floorScore[1], s.result.bestChain, s.result.sweep ? 1 : 0] : 0,
    p: s.players.map((p) => [
      p.on ? 1 : 0,
      p.state,
      r1(p.x),
      r1(p.y),
      r2(p.vx),
      r2(p.vy),
      p.face,
      p.ground ? 1 : 0,
      Math.round(p.walk),
      p.shootT,
      p.kickT,
      p.inv,
      p.ghostT,
      p.lives,
      p.score,
      p.kills,
      p.bestChain,
      (p.speed ? 1 : 0) | (p.range ? 2 : 0) | (p.power ? 4 : 0),
      p.epoch,
      p.floorScore,
      p.respawnT
    ]),
    e: s.enemies.map((e) => [
      e.id,
      KINDS.indexOf(e.kind),
      r1(e.x),
      r1(e.y),
      e.w,
      e.h,
      e.face,
      r1(e.snow),
      (e.rolling ? 1 : 0) | (e.angry ? 2 : 0) | (e.ground ? 4 : 0),
      e.dying,
      e.t,
      e.ballT,
      r1(e.hp),
      e.maxHp,
      e.flash,
      e.spawnT,
      r2(e.vx)
    ]),
    s: s.shots.map((x) => [x.id, x.o, r1(x.x), r1(x.y), r2(x.vx), x.t, x.big ? 1 : 0]),
    j: s.projs.map((x) => [x.id, PROJS.indexOf(x.kind), r1(x.x), r1(x.y), x.w, x.h, r2(x.vx)]),
    i: s.items.map((x) => [x.id, x.kind, r1(x.x), r1(x.y), x.t]),
    ev: evs.map((e) => [EVS.indexOf(e.k), Math.round(e.x), Math.round(e.y), e.n ?? -1, e.p ?? -1])
  }
}

export function decodeEvents(snap: Snapshot): Ev[] {
  return snap.ev.map(([k, x, y, n, p]) => {
    const e: Ev = { k: EVS[k] ?? 'poof', x, y }
    if (n !== -1) e.n = n
    if (p !== -1) e.p = p
    return e
  })
}

function applyPlayer(p: Player, a: number[]): void {
  p.on = !!a[0]
  p.state = a[1]
  p.x = a[2]
  p.y = a[3]
  p.vx = a[4]
  p.vy = a[5]
  p.face = a[6] as 1 | -1
  p.ground = !!a[7]
  p.walk = a[8]
  p.shootT = a[9]
  p.kickT = a[10]
  p.inv = a[11]
  p.ghostT = a[12]
  p.lives = a[13]
  p.score = a[14]
  p.kills = a[15]
  p.bestChain = a[16]
  p.speed = !!(a[17] & 1)
  p.range = !!(a[17] & 2)
  p.power = !!(a[17] & 4)
  p.epoch = a[18]
  p.floorScore = a[19]
  p.respawnT = a[20]
}

function enemyFrom(a: number[]): Enemy {
  return {
    id: a[0],
    kind: KINDS[a[1]] ?? 'book',
    boss: a[1] >= 6,
    x: a[2],
    y: a[3],
    w: a[4],
    h: a[5],
    bw: a[4],
    bh: a[5],
    vx: a[16],
    vy: 0,
    face: a[6] as 1 | -1,
    ground: !!(a[8] & 4),
    snow: a[7],
    hold: 0,
    ballT: a[11],
    rolling: !!(a[8] & 1),
    combo: 0,
    bounces: 0,
    rollT: 0,
    dying: a[9],
    angry: !!(a[8] & 2),
    t: a[10],
    st: 0,
    cd: 0,
    hp: a[12],
    maxHp: a[13],
    flash: a[14],
    spawnT: a[15],
    edge: 0
  }
}

/**
 * guest: the picture to draw at `alpha` (0..1) between snapshots a and b. Positions blend for
 * things present in both; everything else is taken from b.
 */
export function blend(view: GameState, a: Snapshot, b: Snapshot, alpha: number): void {
  if (view.floor !== b.fl || view.tiles.length === 0) view.tiles = buildTiles(b.fl)
  view.frame = b.f
  view.floor = b.fl
  view.phase = PHASES[b.ph] ?? 'play'
  view.phaseT = b.pt
  view.floorT = b.ft
  view.count = b.c
  view.sweep = !!b.sw
  view.reached = b.re
  view.played = b.pl
  view.result = b.r ? { seconds: b.r[0], bonus: b.r[1], floorScore: [b.r[2], b.r[3]], bestChain: b.r[4], sweep: !!b.r[5] } : null
  const lerp = (x: number, y: number) => x + (y - x) * alpha
  b.p.forEach((pb, i) => {
    const p = view.players[i]
    applyPlayer(p, pb)
    const pa = a.p[i]
    if (pa && pa[18] === pb[18] && pa[1] === pb[1]) {
      p.x = lerp(pa[2], pb[2])
      p.y = lerp(pa[3], pb[3])
    }
  })
  const ea = new Map(a.e.map((x) => [x[0], x]))
  view.enemies = b.e.map((x) => {
    const e = enemyFrom(x)
    const old = ea.get(x[0])
    if (old && Math.abs(old[2] - x[2]) < 40 && Math.abs(old[3] - x[3]) < 40) {
      e.x = lerp(old[2], x[2])
      e.y = lerp(old[3], x[3])
      e.snow = alpha < 0.5 ? old[7] : x[7]
    }
    return e
  })
  const sa = new Map(a.s.map((x) => [x[0], x]))
  view.shots = b.s.map((x): Shot => {
    const old = sa.get(x[0])
    return {
      id: x[0],
      o: x[1],
      x: old ? lerp(old[2], x[2]) : x[2],
      y: old ? lerp(old[3], x[3]) : x[3],
      vx: x[4],
      vy: 0,
      t: x[5],
      life: 99,
      big: !!x[6]
    }
  })
  const ja = new Map(a.j.map((x) => [x[0], x]))
  view.projs = b.j.map((x): Proj => {
    const old = ja.get(x[0])
    return { id: x[0], kind: PROJS[x[1]] ?? 'chalk', x: old ? lerp(old[2], x[2]) : x[2], y: old ? lerp(old[3], x[3]) : x[3], w: x[4], h: x[5], vx: x[6], vy: 0, t: 0, life: 99 }
  })
  const ia = new Map(a.i.map((x) => [x[0], x]))
  view.items = b.i.map((x): Item => {
    const old = ia.get(x[0])
    return { id: x[0], kind: x[1], x: old ? lerp(old[2], x[2]) : x[2], y: old ? lerp(old[3], x[3]) : x[3], vx: 0, vy: 0, t: x[4] }
  })
}

/** an empty state for the guest to fill from snapshots */
export function guestState(): GameState {
  const s = createGame(2, 1)
  s.enemies = []
  return s
}

export interface GuestReport {
  t: 'g'
  ep: number
  x: number
  y: number
  vx: number
  vy: number
  f: number
  g: number
  w: number
  st: number
  kt: number
  /** total fire presses so far */
  n: number
}

export function report(p: Player, fires: number): GuestReport {
  return { t: 'g', ep: p.epoch, x: r1(p.x), y: r1(p.y), vx: r2(p.vx), vy: r2(p.vy), f: p.face, g: p.ground ? 1 : 0, w: Math.round(p.walk), st: p.shootT, kt: p.kickT, n: fires }
}

/** host: put the guest's own account of its kid into the simulation; returns new fire presses */
export function applyReport(p: Player, r: GuestReport, lastFires: number): number {
  if (p.state !== ALIVE || r.ep !== p.epoch) return Math.max(lastFires, r.n)
  p.x = r.x
  p.y = r.y
  p.vx = r.vx
  p.vy = r.vy
  p.face = r.f === -1 ? -1 : 1
  p.ground = !!r.g
  p.walk = r.w
  p.shootT = r.st
  p.kickT = r.kt
  return r.n
}
