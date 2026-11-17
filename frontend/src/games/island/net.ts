// Online 双排 over the shared room relay (/api/v1/games/rooms/ws). The host runs the world; the guest
// sends its inputs and draws the host's snapshots, predicting only its own hero.
import { KINDS, KIND_INDEX, type GameEvent, type Kind } from './defs'
import type { Ent, Player, World } from './sim'

export const GAME_SLUG = 'island'

export interface RoomHandlers {
  room(code: string, seat: number): void
  peer(on: boolean): void
  msg(m: Record<string, unknown>): void
  error(code: string): void
  closed(): void
}

export class Room {
  private ws: WebSocket
  private closedByUs = false
  code = ''
  seat = -1

  constructor(code: string | null, h: RoomHandlers, base = '') {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const host = base || `${proto}://${location.host}`
    const url = `${host}/api/v1/games/rooms/ws?game=${GAME_SLUG}${code ? `&room=${encodeURIComponent(code)}` : ''}`
    this.ws = new WebSocket(url)
    this.ws.onmessage = (ev) => {
      let m: Record<string, unknown>
      try {
        m = JSON.parse(String(ev.data))
      } catch {
        return
      }
      if (m.sys === 'room') {
        this.code = String(m.code)
        this.seat = Number(m.seat)
        h.room(this.code, this.seat)
      } else if (m.sys === 'peer') h.peer(!!m.on)
      else if (m.sys === 'error') h.error(String(m.code))
      else if (!m.sys) h.msg(m)
    }
    this.ws.onclose = () => {
      if (!this.closedByUs) h.closed()
    }
    this.ws.onerror = () => undefined
  }

  get open(): boolean {
    return this.ws.readyState === WebSocket.OPEN
  }

  send(m: Record<string, unknown>) {
    if (this.ws.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(m))
  }

  close() {
    this.closedByUs = true
    this.ws.close()
  }
}

const r1 = (n: number) => Math.round(n * 10) / 10
const r2 = (n: number) => Math.round(n * 100) / 100
const ST = ['play', 'dying', 'ghost', 'out', 'win'] as const
const PH = ['play', 'clear', 'over', 'ending'] as const

export type PlayerTuple = number[]
export type EntTuple = number[]

export function encodePlayer(p: Player): PlayerTuple {
  return [
    p.on ? 1 : 0,
    ST.indexOf(p.st),
    r2(p.x),
    r2(p.y),
    r2(p.vx),
    r2(p.vy),
    p.face,
    (p.ground ? 1 : 0) | (p.board ? 2 : 0) | (p.jheld ? 4 : 0),
    p.weapon,
    p.fairy,
    p.cloud,
    p.inv,
    p.trip,
    p.throwT,
    r1(p.runT),
    r2(p.energy),
    p.score,
    p.dyingT,
    p.coyote,
    p.jbuf,
    p.plat,
    p.airT,
    p.prev,
    p.tripCd
  ]
}

export function decodePlayer(t: PlayerTuple, into: Player) {
  into.on = !!t[0]
  into.st = ST[t[1]] ?? 'play'
  into.x = t[2]
  into.y = t[3]
  into.vx = t[4]
  into.vy = t[5]
  into.face = t[6]
  into.ground = !!(t[7] & 1)
  into.board = !!(t[7] & 2)
  into.jheld = !!(t[7] & 4)
  into.weapon = t[8]
  into.fairy = t[9]
  into.cloud = t[10]
  into.inv = t[11]
  into.trip = t[12]
  into.throwT = t[13]
  into.runT = t[14]
  into.energy = t[15]
  into.score = t[16]
  into.dyingT = t[17]
  into.coyote = t[18]
  into.jbuf = t[19]
  into.plat = t[20]
  into.airT = t[21]
  into.prev = t[22]
  into.tripCd = t[23]
}

export function encodeEnt(e: Ent): EntTuple {
  return [e.id, KIND_INDEX[e.k], r1(e.x), r1(e.y), e.w, r1(e.h), e.st, e.dir, e.a, e.t, e.flash, e.hp, r1(e.c), r2(e.vx), r2(e.vy), e.ground ? 1 : 0]
}

export function decodeEnt(t: EntTuple): Ent {
  return {
    id: t[0],
    k: KINDS[t[1]] as Kind,
    x: t[2],
    y: t[3],
    w: t[4],
    h: t[5],
    st: t[6],
    dir: t[7],
    a: t[8],
    t: t[9],
    flash: t[10],
    hp: t[11],
    c: t[12],
    vx: t[13],
    vy: t[14],
    ground: !!t[15],
    b: 0,
    ox: 0,
    oy: 0,
    owner: -1,
    dead: false
  }
}

export interface Snapshot {
  t: 's'
  f: number
  k: number // last guest input seq the host used
  g: number // stage
  cx: number
  lk: number
  ph: number
  pt: number
  lv: number
  sh: number
  bm: number
  p: PlayerTuple[]
  e: EntTuple[]
  v: GameEvent[]
}

export function makeSnapshot(w: World, ack: number, events: GameEvent[]): Snapshot {
  const lo = w.camX - 48
  const hi = w.camX + 400 + 48
  const ents = w.ents.filter((e) => e.k === 'boss' || e.k === 'boom' || e.k === 'shot' || e.k === 'sun' || (e.x + e.w > lo && e.x < hi))
  return {
    t: 's',
    f: w.frame,
    k: ack,
    g: w.stageIndex,
    cx: r1(w.camX),
    lk: w.lock ? 1 : 0,
    ph: PH.indexOf(w.phase),
    pt: w.phaseT,
    lv: w.lives,
    sh: w.shake,
    bm: w.bossMax,
    p: w.players.map(encodePlayer),
    e: ents.slice(0, 90).map(encodeEnt),
    v: events.slice(-60)
  }
}

export function phaseOf(s: Snapshot): (typeof PH)[number] {
  return PH[s.ph] ?? 'play'
}
