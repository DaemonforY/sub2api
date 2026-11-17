// Online 双排 over the site's shared room relay (GET /api/v1/games/rooms/ws).
//
// Host-authoritative: the host runs the World and sends a snapshot 30 times a second; the guest
// sends its buttons (two steps per message) and draws the host's world about 100 ms in the past,
// smoothly interpolated. The guest's own hero is predicted locally with the same movement code
// (physics.ts) and corrected from each snapshot by replaying the inputs the host hasn't used yet,
// so jumping feels immediate even at 150 ms round trip.

import type { PState } from './physics'
import type { Ent, Ev, Phase, Player, PState2 } from './world'
import type { PlayerAnim } from './types'

export type RelayError = 'not_found' | 'room_full' | 'bad_game' | 'too_big' | 'rate' | 'closed' | 'network'

export interface RoomHandlers {
  onRoom(code: string, seat: number): void
  onPeer(on: boolean): void
  onMsg(m: NetMsg): void
  onError(code: RelayError): void
}

export class RoomClient {
  private ws: WebSocket | null = null
  private closedByUs = false
  code = ''
  seat = -1

  constructor(private h: RoomHandlers) {}

  static url(room?: string): string {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const q = new URLSearchParams({ game: 'hutu' })
    if (room) q.set('room', room)
    return `${proto}://${location.host}/api/v1/games/rooms/ws?${q.toString()}`
  }

  connect(room?: string): void {
    this.closedByUs = false
    let ws: WebSocket
    try {
      ws = new WebSocket(RoomClient.url(room))
    } catch {
      this.h.onError('network')
      return
    }
    this.ws = ws
    let errored = false
    ws.onmessage = (ev) => {
      let m: Record<string, unknown>
      try {
        m = JSON.parse(String(ev.data))
      } catch {
        return
      }
      if (m.sys === 'room') {
        this.code = String(m.code)
        this.seat = Number(m.seat)
        this.h.onRoom(this.code, this.seat)
      } else if (m.sys === 'peer') this.h.onPeer(!!m.on)
      else if (m.sys === 'error') {
        errored = true
        this.h.onError((m.code as RelayError) || 'network')
      } else this.h.onMsg(m as unknown as NetMsg)
    }
    ws.onclose = () => {
      if (!this.closedByUs && !errored) this.h.onError(this.seat < 0 ? 'network' : 'closed')
    }
  }

  send(m: NetMsg): void {
    this.sendRaw(JSON.stringify(m))
  }

  sendRaw(s: string): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) this.ws.send(s)
  }

  close(): void {
    this.closedByUs = true
    this.ws?.close()
    this.ws = null
  }
}

// ───────────────────────── messages ─────────────────────────

/** guest → host: buttons for consecutive steps starting at sequence number s */
export interface InputMsg {
  t: 'i'
  s: number
  b: number[]
}
/** guest → host: please send everything again (joined late, or lost track) */
export interface SyncMsg {
  t: 'sync'
}
/** host → guest: the world */
export interface SnapMsg {
  t: 's'
  /** level step, phase, phase step, level, area, camera, time, lives, checkpoint */
  f: number
  ph: Phase
  pt: number
  lv: number
  ar: number
  cam: number
  time: number
  lives: number
  cp: number
  /** edit count in this area, so the guest can tell it missed nothing */
  en: number
  /** music playing on the host */
  mu: string
  paused: number
  /** last guest input sequence the host has used */
  a: number
  p: PlayerTuple[]
  e: EntTuple[]
  ev: Ev[]
}
/** host → guest: full tile edits for the current area (after a sync request or area change) */
export interface FullMsg {
  t: 'full'
  lv: number
  ar: number
  edits: [number, string][]
}
export interface HelloMsg {
  t: 'hello'
  v: number
}
export type NetMsg = InputMsg | SyncMsg | SnapMsg | FullMsg | HelloMsg

const r2 = (n: number) => Math.round(n * 100) / 100

type PlayerTuple = (number | string)[]
type EntTuple = (number | string)[]

const ANIMS: PlayerAnim[] = ['stand', 'walk', 'skid', 'jump', 'fall', 'crouch', 'die', 'pole', 'grow', 'bubble', 'tube', 'gone']
const PSTS: PState2[] = ['play', 'dead', 'bubble', 'pole', 'walk', 'tube', 'done', 'out']

export function packPlayer(p: Player): PlayerTuple {
  const s = p.ps
  return [
    r2(s.x),
    r2(s.y),
    r2(s.vx),
    r2(s.vy),
    s.power,
    s.crouch ? 1 : 0,
    s.ground ? 1 : 0,
    s.coyote,
    s.buffer,
    s.rising ? 1 : 0,
    s.speedClass,
    s.facing,
    s.prev,
    s.skid ? 1 : 0,
    s.ride,
    PSTS.indexOf(p.st),
    p.t,
    p.inv,
    p.bell,
    ANIMS.indexOf(p.anim),
    r2(p.animT),
    Math.floor(p.score),
    p.coins,
    p.morph,
    p.squash,
    r2(p.tubeY),
    p.tubeDir
  ]
}

export function unpackPlayer(t: PlayerTuple, into: Player): void {
  const n = t as number[]
  const s: PState = into.ps
  s.x = n[0]
  s.y = n[1]
  s.vx = n[2]
  s.vy = n[3]
  s.power = n[4]
  s.crouch = !!n[5]
  s.ground = !!n[6]
  s.coyote = n[7]
  s.buffer = n[8]
  s.rising = !!n[9]
  s.speedClass = n[10]
  s.facing = n[11]
  s.prev = n[12]
  s.skid = !!n[13]
  s.ride = n[14]
  into.st = PSTS[n[15]] || 'play'
  into.t = n[16]
  into.inv = n[17]
  into.bell = n[18]
  into.anim = ANIMS[n[19]] || 'stand'
  into.animT = n[20]
  into.score = n[21]
  into.coins = n[22]
  into.morph = n[23]
  into.squash = n[24]
  into.tubeY = n[25]
  into.tubeDir = n[26]
}

export function packEnt(e: Ent): EntTuple {
  return [e.id, e.k, r2(e.x), r2(e.y), e.st, e.t, e.dir, e.age, r2(e.vx), r2(e.vy), r2(e.ax), r2(e.ay), e.len, e.ground ? 1 : 0, e.hp]
}

export function unpackEnt(t: EntTuple): Ent {
  const n = t as number[]
  return {
    id: n[0],
    k: String(t[1]),
    x: n[2],
    y: n[3],
    st: n[4],
    t: n[5],
    dir: n[6],
    age: n[7],
    vx: n[8],
    vy: n[9],
    ax: n[10],
    ay: n[11],
    len: n[12],
    ground: !!n[13],
    hp: n[14],
    w: 14,
    h: 14,
    dead: false,
    owner: -1,
    hitWall: false
  }
}

/** keep messages under the relay's 8 KB cap: drop far-away cosmetic enemies first if ever needed */
export function fitSnapshot(m: SnapMsg, limit = 7600): string {
  let s = JSON.stringify(m)
  while (s.length > limit && m.e.length > 0) {
    m.e.pop()
    s = JSON.stringify(m)
  }
  return s
}
