// The game simulation: one World holds the current level, both players, enemies and items, and
// advances one fixed step at a time from the two players' buttons. Only the host (or the local
// game) runs tick(); an online guest keeps a World too, but fills it from the host's snapshots
// and never simulates anything except its own player's movement (see net.ts).

import { LEVELS } from './levels'
import { PHYS, bodyH, bodyW, boxHitsSolid, cell, isSolid, newPState, stepPlayer, stompBounce, type PState, type Plat, type TileMap } from './physics'
import type { Area, ItemKind, Level, Mover, MusicId, PlayerAnim, Spawn, Theme, Warp } from './types'
import { BTN, ROWS, T, TILE, VIEW_H, VIEW_W } from './types'

export type SfxId =
  | 'jump'
  | 'jumpBig'
  | 'stomp'
  | 'coin'
  | 'bump'
  | 'break'
  | 'sprout'
  | 'grow'
  | 'shrink'
  | 'fire'
  | 'kick'
  | 'die'
  | 'oneup'
  | 'bell'
  | 'tube'
  | 'pole'
  | 'boom'
  | 'fuse'
  | 'splash'
  | 'bosshit'
  | 'roar'
  | 'tick'
  | 'lantern'
  | 'pop'
  | 'thud'
  | 'pause'

export type Ev =
  | { e: 'sfx'; s: SfxId }
  | { e: 'bump'; tx: number; ty: number }
  | { e: 'brk'; tx: number; ty: number }
  | { e: 'coin'; x: number; y: number }
  | { e: 'pts'; x: number; y: number; v: string }
  | { e: 'boom'; x: number; y: number }
  | { e: 'puff'; x: number; y: number }
  | { e: 'edit'; i: number; c: string }
  | { e: 'music'; m: MusicId | '' }
  | { e: 'shake'; n: number }
  | { e: 'area'; lv: number; ar: number }

export type Phase = 'intro' | 'play' | 'clear' | 'dead' | 'over' | 'won'
export type PState2 = 'play' | 'dead' | 'bubble' | 'pole' | 'walk' | 'tube' | 'done' | 'out'

export interface Player {
  id: number
  ps: PState
  st: PState2
  t: number
  /** invulnerable steps after being hit */
  inv: number
  /** 金钟罩 steps left */
  bell: number
  fireCd: number
  anim: PlayerAnim
  animT: number
  combo: number
  score: number
  coins: number
  /** buttons this step */
  btn: number
  /** for tubes: +1 going down, -1 coming up; tubeY is the tube mouth */
  tubeDir: number
  tubeY: number
  /** size change flicker steps */
  morph: number
  /** steps left of being squashed by the partner jumping on your head */
  squash: number
  /** fire button last step */
  prevF: number
}

export interface Ent {
  id: number
  k: string
  x: number
  y: number
  w: number
  h: number
  vx: number
  vy: number
  st: number
  t: number
  dir: number
  ground: boolean
  dead: boolean
  hp: number
  ax: number
  ay: number
  len: number
  owner: number
  /** steps since this entity appeared, for animation */
  age: number
  hitWall: boolean
}

// enemy / entity states
export const ST = { NORMAL: 0, SQUASH: 1, FLIP: 2, BALL: 3, ROLL: 4, LIT: 5, EMERGE: 6, HURT: 7, CHARGE: 8, STUN: 9, ROAR: 10, DEFEAT: 11 } as const

const STOMP_POINTS = [100, 200, 400, 500, 800, 1000, 2000, 4000, 5000, 8000]

export class AreaRT implements TileMap {
  readonly width: number
  readonly theme: Theme
  readonly cells: string[]
  readonly def: Area
  constructor(def: Area) {
    this.def = def
    this.width = def.width
    this.theme = def.theme
    this.cells = []
    for (let y = 0; y < ROWS; y++) for (let x = 0; x < def.width; x++) this.cells.push(def.grid[y][x])
  }
  get(tx: number, ty: number): string {
    return this.cells[ty * this.width + tx]
  }
  set(tx: number, ty: number, c: string): number {
    const i = ty * this.width + tx
    this.cells[i] = c
    return i
  }
}

/** position of a moving platform at a given level step (pure: the guest computes the same) */
export function moverPos(m: Mover, frame: number): { x: number; y: number } {
  const ph = (((frame + (m.phase || 0)) % m.period) + m.period) % m.period
  const k = ph / m.period
  // ease in and out at both ends of the path
  const tri = k < 0.5 ? k * 2 : 2 - k * 2
  const e = tri * tri * (3 - 2 * tri)
  return { x: (m.tx + m.dx * e) * TILE, y: (m.ty + m.dy * e) * TILE }
}

export function moverPlats(movers: Mover[], frame: number): Plat[] {
  return movers.map((m) => {
    const a = moverPos(m, frame)
    const b = moverPos(m, frame + 1)
    return { x0: a.x, y0: a.y, x1: b.x, y1: b.y, w: m.w * TILE }
  })
}

/** top surface (pixel y) of the first standing spot in column tx, scanning down from the top */
export function standY(map: TileMap, tx: number): number {
  for (let ty = 1; ty < ROWS; ty++) {
    if (isSolid(cell(map, tx, ty)) && !isSolid(cell(map, tx, ty - 1)) && (ty < 2 || !isSolid(cell(map, tx, ty - 2)))) return ty * TILE
  }
  return 11 * TILE
}

function overlap(a: { l: number; t: number; r: number; b: number }, b: { l: number; t: number; r: number; b: number }): boolean {
  return a.l < b.r && a.r > b.l && a.t < b.b && a.b > b.t
}

function pbox(p: Player) {
  const w = bodyW(p.ps)
  const h = bodyH(p.ps)
  return { l: p.ps.x - w / 2, r: p.ps.x + w / 2, t: p.ps.y - h, b: p.ps.y }
}
function ebox(e: Ent) {
  return { l: e.x - e.w / 2, r: e.x + e.w / 2, t: e.y - e.h, b: e.y }
}

const ENEMY_SIZE: Record<string, [number, number]> = {
  mud: [14, 12],
  hog: [14, 13],
  kite: [14, 12],
  rat: [12, 20],
  fish: [12, 11],
  cracker: [11, 15],
  firebar: [8, 8],
  spark: [10, 10],
  mudking: [28, 28],
  nian: [36, 30],
  bun: [14, 14],
  chili: [12, 14],
  bell: [14, 14],
  haw: [12, 15],
  fire: [7, 7],
  mudball: [9, 9],
  flame: [12, 8],
  lamp: [14, 18]
}

export interface WorldOpts {
  players: number
  lives?: number
}

export class World {
  levelIdx = 0
  level: Level
  areaIdx = 0
  area: AreaRT
  phase: Phase = 'intro'
  phaseT = 0
  frame = 0
  cam = 0
  time = 0
  timeSub = 0
  lives: number
  players: Player[]
  ents: Ent[] = []
  events: Ev[] = []
  checkpointHit = false
  spawned: boolean[] = []
  coinHits = new Map<number, number>()
  nextId = 1
  musicNow: MusicId | '' = ''
  clearT = 0
  bossDown = false
  paused = false
  readonly nPlayers: number

  constructor(opts: WorldOpts) {
    this.nPlayers = opts.players
    this.lives = opts.lives ?? (opts.players > 1 ? 6 : 4)
    this.players = []
    for (let i = 0; i < opts.players; i++) this.players.push(this.newPlayer(i))
    this.level = LEVELS[0]
    this.area = new AreaRT(this.level.areas[0])
    this.loadLevel(0)
  }

  private newPlayer(id: number): Player {
    return {
      id,
      ps: newPState(0, 0),
      st: 'play',
      t: 0,
      inv: 0,
      bell: 0,
      fireCd: 0,
      anim: 'stand',
      animT: 0,
      combo: 0,
      score: 0,
      coins: 0,
      btn: 0,
      tubeDir: 0,
      tubeY: 0,
      morph: 0,
      squash: 0,
      prevF: 0
    }
  }

  // ───────────────────────── level flow ─────────────────────────

  loadLevel(idx: number, keepCheckpoint = false): void {
    this.levelIdx = idx
    this.level = LEVELS[idx]
    if (!keepCheckpoint) this.checkpointHit = false
    this.time = this.level.time
    this.timeSub = 0
    this.bossDown = false
    this.phase = 'intro'
    this.phaseT = 0
    this.clearT = 0
    this.enterArea(0, null)
    const def = this.level.areas[0]
    const startTx = this.checkpointHit && def.checkpoint ? def.checkpoint : this.level.start
    this.players.forEach((p, i) => {
      const tx = startTx + i
      const keepPower = p.st !== 'out' && p.st !== 'dead' ? p.ps.power : 0
      p.ps = newPState((tx + 0.5) * TILE, standY(this.area, tx), keepPower)
      p.st = 'play'
      p.t = 0
      p.inv = 0
      p.bell = 0
      p.combo = 0
      p.morph = 0
      p.anim = 'stand'
    })
    this.cam = Math.max(0, Math.min(this.maxCam(), (startTx - 4) * TILE))
    this.music('')
  }

  private enterArea(idx: number, warp: Warp | null): void {
    this.areaIdx = idx
    this.area = new AreaRT(this.level.areas[idx])
    this.ents = []
    this.spawned = this.area.def.spawns.map(() => false)
    this.coinHits.clear()
    this.frame = 0
    this.editLog = []
    this.events.push({ e: 'area', lv: this.levelIdx, ar: idx })
    if (warp) {
      this.cam = Math.max(0, Math.min(this.maxCam(), warp.ax * TILE - 120))
    }
  }

  maxCam(): number {
    return Math.max(0, this.area.width * TILE - VIEW_W)
  }

  music(m: MusicId | ''): void {
    if (this.musicNow === m) return
    this.musicNow = m
    this.events.push({ e: 'music', m })
  }

  sfx(s: SfxId): void {
    this.events.push({ e: 'sfx', s })
  }

  /** tile edits since the area was entered, so a late-joining guest can catch up */
  editLog: [number, string][] = []

  private edit(tx: number, ty: number, c: string): void {
    const i = this.area.set(tx, ty, c)
    this.editLog.push([i, c])
    this.events.push({ e: 'edit', i, c })
  }

  private addEnt(k: string, x: number, y: number, extra: Partial<Ent> = {}): Ent {
    const [w, h] = ENEMY_SIZE[k] || [14, 14]
    const e: Ent = { id: this.nextId++, k, x, y, w, h, vx: 0, vy: 0, st: ST.NORMAL, t: 0, dir: -1, ground: false, dead: false, hp: 1, ax: 0, ay: 0, len: 0, owner: -1, age: 0, hitWall: false, ...extra }
    this.ents.push(e)
    return e
  }

  private points(p: Player | null, v: number, x: number, y: number): void {
    if (p) p.score += v
    this.events.push({ e: 'pts', x, y, v: String(v) })
  }

  private addCoin(p: Player | null): void {
    this.sfx('coin')
    if (!p) return
    p.coins++
    p.score += 200
    if (p.coins % 100 === 0) this.oneUp(p, p.ps.x, p.ps.y - 24)
  }

  private oneUp(p: Player | null, x: number, y: number): void {
    this.lives = Math.min(99, this.lives + 1)
    this.sfx('oneup')
    this.events.push({ e: 'pts', x, y, v: '1UP' })
    void p
  }

  alivePlayers(): Player[] {
    return this.players.filter((p) => p.st === 'play' || p.st === 'pole' || p.st === 'walk' || p.st === 'tube')
  }

  // ───────────────────────── main step ─────────────────────────

  tick(btns: number[]): void {
    this.players.forEach((p, i) => (p.btn = btns[i] || 0))
    if (this.paused) return
    this.phaseT++
    switch (this.phase) {
      case 'intro':
        if (this.phaseT >= 150) {
          this.phase = 'play'
          this.phaseT = 0
          this.music(this.level.music)
        }
        return
      case 'over':
      case 'won':
        return
      case 'dead':
        this.stepEnts()
        this.stepPlayers()
        if (this.phaseT > 150) {
          if (this.lives > 0) {
            this.lives--
            this.loadLevel(this.levelIdx, true)
          } else {
            this.phase = 'over'
            this.phaseT = 0
            this.music('over')
          }
        }
        return
      default:
        break
    }
    this.frame++
    this.spawnEnts()
    this.stepPlayers()
    this.stepEnts()
    this.interact()
    this.stepCamera()
    this.stepTimer()
    this.checkFlow()
  }

  /** start again after game over: keep the level, reset lives (score halves) */
  continueGame(): void {
    this.lives = this.nPlayers > 1 ? 6 : 4
    this.players.forEach((p) => {
      p.score = Math.floor(p.score / 2)
      p.coins = 0
      p.st = 'play'
      p.ps.power = 0
    })
    this.loadLevel(this.levelIdx, true)
  }

  restartFrom(idx: number): void {
    this.players.forEach((p) => {
      p.st = 'play'
      p.ps.power = 0
    })
    this.loadLevel(idx)
  }

  private stepTimer(): void {
    if (this.phase !== 'play' || this.bossDown) return
    if (++this.timeSub >= 60) {
      this.timeSub = 0
      this.time--
      if (this.time === 60) this.sfx('pause')
      if (this.time <= 0) {
        this.time = 0
        this.players.forEach((p) => {
          if (p.st === 'play' || p.st === 'bubble') this.kill(p)
        })
      }
    }
  }

  private checkFlow(): void {
    if (this.phase === 'play') {
      const living = this.players.filter((p) => p.st !== 'dead' && p.st !== 'out')
      const anyDying = this.players.some((p) => p.st === 'dead')
      if (!living.some((p) => p.st !== 'bubble')) {
        if (!anyDying || this.players.every((p) => p.st === 'out' || (p.st === 'dead' && p.t > 100) || p.st === 'bubble')) {
          this.phase = 'dead'
          this.phaseT = 0
          this.music('')
        }
      }
      // goal reached: once someone grabs the pole the others get a few seconds to join
      const finished = this.players.filter((p) => p.st === 'pole' || p.st === 'walk' || p.st === 'done')
      if (finished.length) {
        this.clearT++
        const others = this.players.filter((p) => p.st === 'play' || p.st === 'bubble' || p.st === 'tube')
        if (!others.length || this.clearT > 240) {
          for (const p of others) {
            p.st = 'done'
            p.bell = 0
          }
          if (this.players.every((p) => p.st === 'done' || p.st === 'out' || p.st === 'dead')) {
            this.phase = 'clear'
            this.phaseT = 0
          }
        }
      }
    } else if (this.phase === 'clear') {
      // count the remaining time into score
      if (this.phaseT > 60 && this.time > 0) {
        const n = Math.min(this.time, 3)
        this.time -= n
        const share = this.players.filter((p) => p.st !== 'out')
        for (const p of share.length ? share : this.players) p.score += (50 * n) / Math.max(1, share.length)
        if (this.phaseT % 4 === 0) this.sfx('tick')
        this.phaseT = 61
        this.clearT = 0
      } else if (this.phaseT > 60) {
        if (++this.clearT > 90) {
          if (this.levelIdx + 1 >= LEVELS.length) {
            this.phase = 'won'
            this.phaseT = 0
            this.music('clear')
          } else {
            this.players.forEach((p) => {
              if (p.st === 'out') {
                p.st = 'play'
                p.ps.power = 0
              }
            })
            this.loadLevel(this.levelIdx + 1)
          }
        }
      }
    }
  }

  // ───────────────────────── players ─────────────────────────

  private plats: Plat[] = []

  private stepPlayers(): void {
    this.plats = moverPlats(this.area.def.movers, this.frame)
    for (const p of this.players) this.stepPlayer(p)
  }

  private stepPlayer(p: Player): void {
    p.t++
    if (p.inv > 0) p.inv--
    if (p.morph > 0) p.morph--
    if (p.squash > 0) p.squash--
    if (p.fireCd > 0) p.fireCd--
    if (p.bell > 0) {
      p.bell--
      if (p.bell === 0 && !this.players.some((q) => q.bell > 0) && this.phase === 'play') this.music(this.level.music)
    }
    const ps = p.ps
    switch (p.st) {
      case 'out':
      case 'done':
        return
      case 'dead': {
        if (p.t < 30) return
        ps.vy = Math.min(4, ps.vy + 0.25)
        ps.y += ps.vy
        if (p.t > 120 && this.phase === 'play') {
          const partner = this.players.find((q) => q !== p && (q.st === 'play' || q.st === 'pole' || q.st === 'walk'))
          if (partner && this.lives > 0) {
            this.lives--
            p.st = 'bubble'
            p.t = 0
            ps.power = 0
            ps.x = partner.ps.x
            ps.y = 40
            ps.vx = 0
            ps.vy = 0
          } else if (partner) {
            p.st = 'out'
          }
        }
        return
      }
      case 'bubble': {
        const partner = this.players.find((q) => q !== p && q.st === 'play')
        if (!partner) {
          // nobody left to float to
          if (!this.players.some((q) => q.st === 'play' || q.st === 'pole' || q.st === 'walk')) return
          return
        }
        const tx = partner.ps.x
        const ty = partner.ps.y - 40
        ps.x += (tx - ps.x) * 0.05
        ps.y += (ty - ps.y) * 0.05 + Math.sin(p.t * 0.08) * 0.3
        ps.x = Math.max(this.cam + 12, Math.min(this.cam + VIEW_W - 12, ps.x))
        const near = Math.abs(ps.x - tx) < 18 && Math.abs(ps.y - partner.ps.y) < 30
        const wantPop = (p.btn & BTN.J) !== 0 && (ps.prev & BTN.J) === 0 && p.t > 30
        ps.prev = p.btn
        if ((near || wantPop || p.t > 600) && !boxHitsSolid(this.area, ps.x - 5, ps.y - 14, ps.x + 5, ps.y)) {
          p.st = 'play'
          p.t = 0
          p.inv = 120
          ps.vx = partner.ps.vx
          ps.vy = -2
          this.sfx('pop')
          this.events.push({ e: 'puff', x: ps.x, y: ps.y - 8 })
        }
        return
      }
      case 'pole': {
        const goal = this.area.def.goal ?? 0
        ps.x = goal * TILE + 8 - 5
        const bottom = 11 * TILE
        if (ps.y < bottom) ps.y = Math.min(bottom, ps.y + 2.5)
        else if (p.t > 50) {
          p.st = 'walk'
          p.t = 0
          ps.facing = 1
        }
        p.anim = 'pole'
        return
      }
      case 'walk': {
        ps.vx = 1.3
        const target = ((this.area.def.goal ?? 0) + 6) * TILE + 8
        const r = stepPlayer(ps, BTN.R, this.area)
        void r
        p.anim = ps.ground ? 'walk' : 'fall'
        p.animT++
        if (ps.x >= target) {
          p.st = 'done'
          ps.x = target
        }
        return
      }
      case 'tube': {
        if (p.tubeDir > 0) {
          if (p.t <= 32) ps.y += 0.75
        } else if (p.t >= 0) {
          ps.y -= 0.75
          if (ps.y <= p.tubeY) {
            ps.y = p.tubeY
            ps.ground = true
            p.st = 'play'
            p.t = 0
          }
        }
        return
      }
      default:
        break
    }

    // ── normal play
    const wasGround = ps.ground
    const r = stepPlayer(ps, p.btn, this.area, this.plats)
    if (r.jumped) this.sfx(ps.power > 0 ? 'jumpBig' : 'jump')
    if (ps.ground) p.combo = 0
    if (r.bump) this.bumpTile(r.bump.tx, r.bump.ty, p)
    void wasGround

    // fire chili sparks
    if (ps.power === 2 && (p.btn & BTN.F) && !(p.anim === 'crouch') && p.prevF === 0 && p.fireCd === 0) {
      const mine = this.ents.filter((e) => e.k === 'fire' && e.owner === p.id).length
      if (mine < 2) {
        this.addEnt('fire', ps.x + ps.facing * 8, ps.y - bodyH(ps) + 10, { vx: ps.facing * 3.6, vy: 1, owner: p.id, dir: ps.facing })
        this.sfx('fire')
        p.fireCd = 10
      }
    }
    p.prevF = p.btn & BTN.F

    // keep inside the camera: the left edge pushes, a player stuck against a wall gets bubbled to the partner
    const w = bodyW(ps)
    const minX = this.cam + w / 2
    const maxX = this.cam + VIEW_W - w / 2
    if (ps.x < minX) {
      ps.x = minX
      if (ps.vx < 0) ps.vx = 0
      if (boxHitsSolid(this.area, ps.x - w / 2, ps.y - bodyH(ps), ps.x + w / 2, ps.y - 0.5)) {
        const partner = this.players.find((q) => q !== p && q.st === 'play')
        if (partner) {
          p.st = 'bubble'
          p.t = 0
        } else ps.x = minX
      }
    }
    if (ps.x > maxX) {
      ps.x = maxX
      if (ps.vx > 0) ps.vx = 0
    }

    // coins in the air
    this.collectCoins(p)

    // fell out or into lava
    if (ps.y > VIEW_H + 24) {
      this.kill(p, true)
      return
    }
    const feet = cell(this.area, Math.floor(ps.x / TILE), Math.floor((ps.y - 3) / TILE))
    if (feet === T.LAVA) {
      this.kill(p, true)
      this.sfx('splash')
      return
    }

    // goal pole
    const goal = this.area.def.goal
    if (goal !== undefined && ps.x + w / 2 >= goal * TILE + 6 && ps.x - w / 2 <= goal * TILE + 10 && this.phase === 'play') {
      p.st = 'pole'
      p.t = 0
      ps.vx = 0
      ps.vy = 0
      p.bell = 0
      const h = Math.max(0, 11 * TILE - ps.y)
      const v = h > 120 ? 5000 : h > 90 ? 2000 : h > 60 ? 800 : h > 30 ? 400 : 100
      this.points(p, v, ps.x, ps.y - 20)
      this.sfx('pole')
      if (!this.players.some((q) => q !== p && (q.st === 'pole' || q.st === 'walk' || q.st === 'done'))) this.music('clear')
      return
    }

    // checkpoint lantern
    const cp = this.area.def.checkpoint
    if (cp !== undefined && !this.checkpointHit && this.areaIdx === 0 && ps.x > cp * TILE) {
      this.checkpointHit = true
      this.sfx('lantern')
    }

    // tubes
    this.tryWarp(p)

    // animation
    if (!ps.ground) p.anim = ps.vy < 0 ? 'jump' : 'fall'
    else if (ps.crouch) p.anim = 'crouch'
    else if (ps.skid) p.anim = 'skid'
    else if (Math.abs(ps.vx) > 0.05) p.anim = 'walk'
    else p.anim = 'stand'
    p.animT += ps.ground ? Math.abs(ps.vx) * 0.25 + 0.05 : 0
  }

  private collectCoins(p: Player): void {
    const b = pbox(p)
    const tx0 = Math.floor(b.l / TILE)
    const tx1 = Math.floor((b.r - 0.01) / TILE)
    const ty0 = Math.max(0, Math.floor(b.t / TILE))
    const ty1 = Math.min(ROWS - 1, Math.floor((b.b - 0.01) / TILE))
    for (let ty = ty0; ty <= ty1; ty++)
      for (let tx = tx0; tx <= tx1; tx++)
        if (cell(this.area, tx, ty) === T.COIN) {
          this.edit(tx, ty, T.EMPTY)
          this.addCoin(p)
        }
  }

  private tryWarp(p: Player): void {
    const ps = p.ps
    if (!ps.ground || !(p.btn & BTN.D)) return
    for (const w of this.area.def.warps) {
      const cx = (w.tx + 1) * TILE
      if (Math.abs(ps.x - cx) < 8 && Math.abs(ps.y - w.ty * TILE) < 1) {
        // the whole team goes through together
        p.st = 'tube'
        p.t = 0
        p.tubeDir = 1
        p.tubeY = ps.y
        ps.vx = 0
        ps.x = cx
        this.sfx('tube')
        this.pendingWarp = { w, t: 40 }
        return
      }
    }
  }

  pendingWarp: { w: Warp; t: number } | null = null

  private doWarp(w: Warp): void {
    this.enterArea(w.to, w)
    const exitY = w.exitTube ? w.ay * TILE + 30 : (w.ay + 1) * TILE
    this.players.forEach((p, i) => {
      if (p.st === 'out' || p.st === 'dead') return
      const ps = p.ps
      ps.x = (w.exitTube ? (w.ax + 1) * TILE : (w.ax + 0.5) * TILE) + (w.exitTube ? 0 : i * 12)
      ps.y = exitY
      ps.vx = 0
      ps.vy = 0
      ps.ride = -1
      p.bell = 0
      if (w.exitTube) {
        p.st = 'tube'
        p.tubeDir = -1
        p.tubeY = w.ay * TILE
        p.t = i * -30
        ps.y = w.ay * TILE + 30
      } else {
        p.st = 'play'
        p.t = 0
      }
    })
    if (w.exitTube) this.sfx('tube')
    this.music(w.to === 0 ? this.level.music : this.area.theme === 'cave' ? 'cave' : this.level.music)
  }

  kill(p: Player, fell = false): void {
    if (p.st !== 'play' && p.st !== 'bubble') return
    p.st = 'dead'
    p.t = fell ? 30 : 0
    p.bell = 0
    p.ps.vx = 0
    p.ps.vy = fell ? 0 : -4.5
    p.ps.power = 0
    p.ps.crouch = false
    p.anim = 'die'
    this.sfx('die')
    if (!this.players.some((q) => q.st === 'play' || q.st === 'pole' || q.st === 'walk')) this.music('')
    if (fell) p.ps.y = VIEW_H + 60
  }

  hurt(p: Player): void {
    if (p.inv > 0 || p.bell > 0 || p.st !== 'play') return
    if (p.ps.power > 0) {
      p.ps.power = p.ps.power === 2 ? 1 : 0
      p.ps.crouch = false
      p.inv = 120
      p.morph = 30
      this.sfx('shrink')
    } else this.kill(p)
  }

  private grow(p: Player, to: number): void {
    const ps = p.ps
    if (ps.power >= to) return
    const was = ps.power
    ps.power = to
    p.morph = 30
    if (was === 0 && boxHitsSolid(this.area, ps.x - 6, ps.y - 26, ps.x + 6, ps.y - 14)) ps.crouch = true
    this.sfx('grow')
  }

  // ───────────────────────── blocks ─────────────────────────

  private bumpTile(tx: number, ty: number, p: Player | null, fromBlast = false): void {
    const c = cell(this.area, tx, ty)
    const big = p ? p.ps.power > 0 : true
    const x = (tx + 0.5) * TILE
    const y = ty * TILE
    const sprout = (kind: ItemKind) => {
      this.addEnt(kind, x, y + TILE, { st: ST.EMERGE, t: 0, ay: y, dir: 1 })
      this.sfx('sprout')
    }
    switch (c) {
      case T.BRICK:
        if (big) {
          this.edit(tx, ty, T.EMPTY)
          this.events.push({ e: 'brk', tx, ty })
          this.sfx('break')
          if (p) p.score += 50
        } else {
          this.events.push({ e: 'bump', tx, ty })
          this.sfx('bump')
        }
        break
      case T.BOX:
      case T.BOX_HIDDEN:
        this.edit(tx, ty, T.USED)
        this.events.push({ e: 'bump', tx, ty }, { e: 'coin', x, y: y - 8 })
        this.addCoin(p)
        break
      case T.BRICK_COINS: {
        const i = ty * this.area.width + tx
        const n = (this.coinHits.get(i) || 0) + 1
        this.coinHits.set(i, n)
        this.events.push({ e: 'bump', tx, ty }, { e: 'coin', x, y: y - 8 })
        this.addCoin(p)
        if (n >= 10) this.edit(tx, ty, T.USED)
        break
      }
      case T.BOX_POWER:
      case T.BRICK_POWER:
        this.edit(tx, ty, T.USED)
        this.events.push({ e: 'bump', tx, ty })
        sprout(p && p.ps.power === 0 ? 'bun' : 'chili')
        break
      case T.BOX_BELL:
      case T.BRICK_BELL:
        this.edit(tx, ty, T.USED)
        this.events.push({ e: 'bump', tx, ty })
        sprout('bell')
        break
      case T.BOX_LIFE:
        this.edit(tx, ty, T.USED)
        this.events.push({ e: 'bump', tx, ty })
        sprout('haw')
        break
      case T.USED:
      case T.HARD:
      case T.GROUND:
      case T.WALL:
      default:
        if (!fromBlast) this.sfx('thud')
        return
    }
    // whatever stands on a bumped block gets knocked
    const top = ty * TILE
    for (const e of this.ents) {
      if (e.dead || Math.abs(e.x - x) > 14 || Math.abs(e.y - top) > 3) continue
      if (e.k === 'bun' || e.k === 'haw' || e.k === 'bell') {
        e.vy = -3
        e.vx = Math.abs(e.vx || 1) * (e.x < x ? -1 : 1)
      } else if (isEnemy(e.k) && e.st !== ST.FLIP) this.flip(e, p, e.x < x ? -1 : 1)
    }
    // a coin sitting on top pops too
    if (cell(this.area, tx, ty - 1) === T.COIN) {
      this.edit(tx, ty - 1, T.EMPTY)
      this.events.push({ e: 'coin', x, y: y - 24 })
      this.addCoin(p)
    }
  }

  // ───────────────────────── entities ─────────────────────────

  private spawnEnts(): void {
    const sp = this.area.def.spawns
    const edge = this.cam + VIEW_W + 40
    for (let i = 0; i < sp.length; i++) {
      if (this.spawned[i]) continue
      const s = sp[i]
      if (s.tx * TILE > edge) continue
      this.spawned[i] = true
      if ((s.tx + 1) * TILE < this.cam - 32 && s.kind !== 'firebar') continue
      this.spawnEnemy(s)
    }
  }

  private spawnEnemy(s: Spawn): void {
    const x = (s.tx + 0.5) * TILE
    const y = (s.ty + 1) * TILE
    switch (s.kind) {
      case 'rat':
        this.addEnt('rat', (s.tx + 1) * TILE, s.ty * TILE + 22, { ax: (s.tx + 1) * TILE, ay: s.ty * TILE, t: 60 })
        break
      case 'firebar':
        this.addEnt('firebar', x, y - 8, { ax: x, ay: s.ty * TILE + 8, len: s.len ?? 5, dir: s.dir ?? 1, t: 0 })
        break
      case 'spark':
        this.addEnt('spark', x, VIEW_H + 20, { ay: VIEW_H + 20, t: (s.tx * 37) % 90 })
        break
      case 'fish':
        this.addEnt('fish', x, VIEW_H + 16, { ay: VIEW_H + 16, ax: x, t: (s.tx * 53) % 120 })
        break
      case 'kite':
        this.addEnt('kite', x, y, { ay: y, vx: -0.45 })
        break
      case 'mudking':
        this.addEnt('mudking', x, y, { hp: 3, t: 0, ax: x })
        this.music('boss')
        break
      case 'nian':
        this.addEnt('nian', x, y, { hp: this.nPlayers > 1 ? 5 : 4, t: 0, ax: x })
        this.music('boss')
        break
      default:
        this.addEnt(s.kind, x, y, { vx: -0.5 * (s.dir ?? 1), dir: -1 })
    }
  }

  private moveBody(e: Ent, grav = 0.3, turn = true): void {
    e.vy = Math.min(4.2, e.vy + grav)
    e.x += e.vx
    const l = e.x - e.w / 2
    const r = e.x + e.w / 2
    const t = e.y - e.h
    const ty0 = Math.floor(t / TILE)
    const ty1 = Math.floor((e.y - 0.01) / TILE)
    let hitWall = false
    for (let ty = ty0; ty <= ty1; ty++) {
      if (e.vx > 0 && isSolid(cell(this.area, Math.floor(r / TILE), ty))) {
        e.x = Math.floor(r / TILE) * TILE - e.w / 2
        hitWall = true
        break
      }
      if (e.vx < 0 && isSolid(cell(this.area, Math.floor(l / TILE), ty))) {
        e.x = (Math.floor(l / TILE) + 1) * TILE + e.w / 2
        hitWall = true
        break
      }
    }
    if (hitWall) {
      if (turn) e.vx = -e.vx
      e.dir = Math.sign(e.vx) || e.dir
      e.hitWall = true
    } else e.hitWall = false
    const prevY = e.y
    e.y += e.vy
    e.ground = false
    const l2 = e.x - e.w / 2 + 1
    const r2 = e.x + e.w / 2 - 1
    if (e.vy >= 0) {
      const ty = Math.floor(e.y / TILE)
      for (let tx = Math.floor(l2 / TILE); tx <= Math.floor(r2 / TILE); tx++) {
        const c = cell(this.area, tx, ty)
        if (isSolid(c) || (c === T.PLANK && prevY <= ty * TILE + 0.01)) {
          e.y = ty * TILE
          e.vy = 0
          e.ground = true
          break
        }
      }
      if (!e.ground)
        for (const pl of this.plats) {
          if (r2 <= pl.x1 || l2 >= pl.x1 + pl.w) continue
          if (prevY <= pl.y0 + 0.5 && e.y >= pl.y1) {
            e.y = pl.y1
            e.vy = 0
            e.ground = true
            e.x += pl.x1 - pl.x0
            break
          }
        }
    } else {
      const ty = Math.floor((e.y - e.h) / TILE)
      for (let tx = Math.floor(l2 / TILE); tx <= Math.floor(r2 / TILE); tx++)
        if (isSolid(cell(this.area, tx, ty))) {
          e.y = (ty + 1) * TILE + e.h
          e.vy = 0
          break
        }
    }
  }

  private stepEnts(): void {
    for (const e of this.ents) {
      if (e.dead) continue
      e.age++
      e.t++
      this.stepEnt(e)
      if (e.k !== 'firebar' && e.k !== 'mudking' && e.k !== 'nian' && e.k !== 'spark' && e.k !== 'fish') {
        if (e.y > VIEW_H + 48 || e.x < this.cam - 96 || e.x > this.cam + VIEW_W + 160) e.dead = true
      } else if (e.k === 'firebar' && e.ax < this.cam - 120) e.dead = true
      else if ((e.k === 'spark' || e.k === 'fish') && e.ax < this.cam - 64 && e.x < this.cam - 64) e.dead = true
      if (!e.dead && e.k !== 'spark' && e.k !== 'fish' && e.st !== ST.FLIP && isEnemy(e.k) && e.k !== 'mudking' && e.k !== 'nian') {
        const c = cell(this.area, Math.floor(e.x / TILE), Math.floor((e.y - 2) / TILE))
        if (c === T.LAVA) {
          e.dead = true
          this.events.push({ e: 'puff', x: e.x, y: e.y - 4 })
        }
      }
    }
    this.ents = this.ents.filter((e) => !e.dead)
    if (this.pendingWarp) {
      if (--this.pendingWarp.t <= 0) {
        const w = this.pendingWarp.w
        this.pendingWarp = null
        this.doWarp(w)
      }
    }
  }

  private stepEnt(e: Ent): void {
    switch (e.k) {
      case 'mud':
      case 'hog':
      case 'cracker':
        this.stepWalker(e)
        break
      case 'kite':
        if (e.st === ST.FLIP) {
          e.vy += 0.25
          e.y += e.vy
          e.x += e.vx
          break
        }
        e.x += e.vx
        e.y = e.ay + Math.sin(e.t * 0.045) * 22
        e.dir = Math.sign(e.vx) || -1
        break
      case 'rat':
        this.stepRat(e)
        break
      case 'fish':
        this.stepFish(e)
        break
      case 'spark':
        this.stepSpark(e)
        break
      case 'firebar':
        e.t += 0
        break
      case 'bun':
      case 'haw':
      case 'chili':
      case 'bell':
        this.stepItem(e)
        break
      case 'fire':
        this.stepFire(e)
        break
      case 'mudball':
        e.vy += 0.18
        e.x += e.vx
        e.y += e.vy
        if (isSolid(cell(this.area, Math.floor(e.x / TILE), Math.floor(e.y / TILE))) || cell(this.area, Math.floor(e.x / TILE), Math.floor(e.y / TILE)) === T.LAVA) {
          e.dead = true
          this.events.push({ e: 'puff', x: e.x, y: e.y })
        }
        break
      case 'flame':
        e.x += e.vx
        e.y = e.ay + Math.sin(e.t * 0.15) * 6
        if (isSolid(cell(this.area, Math.floor(e.x / TILE), Math.floor((e.y - 4) / TILE)))) e.dead = true
        break
      case 'mudking':
        this.stepMudKing(e)
        break
      case 'nian':
        this.stepNian(e)
        break
      case 'lamp':
        e.vy = Math.min(3, e.vy + 0.15)
        this.moveBody(e, 0, false)
        break
    }
  }

  private stepWalker(e: Ent): void {
    if (e.st === ST.SQUASH) {
      if (e.t > 30) e.dead = true
      return
    }
    if (e.st === ST.FLIP) {
      e.vy += 0.3
      e.x += e.vx
      e.y += e.vy
      return
    }
    if (e.k === 'hog' && e.st === ST.BALL) {
      e.vx = 0
      this.moveBody(e)
      if (e.t > 420) {
        e.st = ST.NORMAL
        e.t = 0
        e.vx = -0.5
      }
      return
    }
    if (e.k === 'cracker' && e.st === ST.LIT) {
      if (Math.abs(e.vx) > 0) this.moveBody(e, 0.3, true)
      else this.moveBody(e)
      if (e.t % 20 === 0) this.sfx('fuse')
      if (e.t > 100 || (e.hitWall && Math.abs(e.vx) > 1 && e.t > 20)) this.explode(e)
      return
    }
    const speed = e.st === ST.ROLL ? 3.8 : e.k === 'cracker' ? 0.42 : 0.5
    if (e.vx === 0) e.vx = -speed
    e.vx = Math.sign(e.vx) * speed
    this.moveBody(e)
    e.dir = Math.sign(e.vx) || e.dir
    if (e.st === ST.ROLL && e.hitWall) this.sfx('thud')
  }

  private stepRat(e: Ent): void {
    // cycle: hidden 90, rising 30, up 70, sinking 30
    const cyc = e.t % 220
    const near = this.players.some((p) => p.st === 'play' && Math.abs(p.ps.x - e.ax) < 26 && p.ps.y >= e.ay - 2 && p.ps.y <= e.ay + 32)
    if (cyc === 0 && near) {
      e.t -= 1
      return
    }
    let rise = 0
    if (cyc < 90) rise = 0
    else if (cyc < 120) rise = (cyc - 90) / 30
    else if (cyc < 190) rise = 1
    else rise = 1 - (cyc - 190) / 30
    e.y = e.ay + 22 - rise * 22
    e.st = rise > 0.3 ? ST.NORMAL : ST.EMERGE
    if (e.st === ST.FLIP) e.dead = true
  }

  private stepFish(e: Ent): void {
    if (e.st === ST.FLIP || e.st === ST.SQUASH) {
      e.vy += 0.3
      e.y += e.vy
      e.x += e.vx
      if (e.y > VIEW_H + 40) e.dead = true
      return
    }
    if (e.y >= e.ay && e.vy >= 0) {
      e.y = e.ay
      e.vy = 0
      if (e.t > 100) {
        // leap towards the nearest player
        const p = this.nearestPlayer(e.x)
        e.vy = -6.6
        e.vx = p ? Math.max(-1, Math.min(1, (p.ps.x - e.x) / 90)) : 0
        e.t = 0
        this.sfx('splash')
      }
      return
    }
    e.vy += 0.17
    e.x += e.vx
    e.y += e.vy
    e.dir = Math.sign(e.vx) || e.dir
  }

  private stepSpark(e: Ent): void {
    if (e.y >= e.ay && e.vy >= 0) {
      e.y = e.ay
      e.vy = 0
      if (e.t > 110) {
        e.vy = -7.2
        e.t = 0
      }
      return
    }
    e.vy += 0.16
    e.y += e.vy
  }

  private stepItem(e: Ent): void {
    if (e.st === ST.EMERGE) {
      e.y = Math.max(e.ay, e.y - 0.5)
      if (e.y <= e.ay) {
        e.st = ST.NORMAL
        e.t = 0
        if (e.k === 'bun' || e.k === 'haw') e.vx = 1.0
        if (e.k === 'bell') {
          e.vx = 1.2
          e.vy = -3
        }
      }
      return
    }
    if (e.k === 'chili') return
    this.moveBody(e, e.k === 'bell' ? 0.2 : 0.3)
    if (e.k === 'bell' && e.ground) e.vy = -4.2
  }

  private stepFire(e: Ent): void {
    e.vy = Math.min(4, e.vy + 0.35)
    const vx = e.vx
    this.moveBody(e, 0, false)
    if (e.ground) e.vy = -2.6
    if (e.hitWall || Math.abs(vx) < 0.1) {
      e.dead = true
      this.events.push({ e: 'puff', x: e.x, y: e.y - 3 })
      this.sfx('thud')
    }
    if (e.t > 150) e.dead = true
  }

  private explode(e: Ent): void {
    e.dead = true
    const cx = e.x
    const cy = e.y - 8
    this.events.push({ e: 'boom', x: cx, y: cy }, { e: 'shake', n: 10 })
    this.sfx('boom')
    const R = 30
    const tx0 = Math.floor((cx - R) / TILE)
    const tx1 = Math.floor((cx + R) / TILE)
    const ty0 = Math.floor((cy - R) / TILE)
    const ty1 = Math.floor((cy + R) / TILE)
    for (let ty = ty0; ty <= ty1; ty++)
      for (let tx = tx0; tx <= tx1; tx++) {
        const c = cell(this.area, tx, ty)
        const dx = (tx + 0.5) * TILE - cx
        const dy = (ty + 0.5) * TILE - cy
        if (dx * dx + dy * dy > (R + 8) * (R + 8)) continue
        if (c === T.BRICK) {
          this.edit(tx, ty, T.EMPTY)
          this.events.push({ e: 'brk', tx, ty })
        } else if (c === T.BRICK_COINS || c === T.BRICK_POWER || c === T.BRICK_BELL) this.bumpTile(tx, ty, null, true)
      }
    for (const o of this.ents) {
      if (o === e || o.dead || !isEnemy(o.k) || o.k === 'firebar') continue
      if (Math.hypot(o.x - cx, o.y - o.h / 2 - cy) < R + 6) {
        if (o.k === 'mudking' || o.k === 'nian') this.hitBoss(o, null)
        else if (o.k === 'cracker' && o.st !== ST.FLIP) {
          o.st = ST.LIT
          o.t = 90
        } else this.flip(o, null, o.x < cx ? -1 : 1)
      }
    }
    for (const p of this.players) {
      if (p.st !== 'play') continue
      const b = pbox(p)
      if (Math.hypot(Math.max(b.l, Math.min(cx, b.r)) - cx, Math.max(b.t, Math.min(cy, b.b)) - cy) < R - 6) this.hurt(p)
    }
  }

  private nearestPlayer(x: number): Player | null {
    let best: Player | null = null
    for (const p of this.players) if (p.st === 'play' && (!best || Math.abs(p.ps.x - x) < Math.abs(best.ps.x - x))) best = p
    return best
  }

  // ── bosses

  private stepMudKing(e: Ent): void {
    if (e.st === ST.DEFEAT) {
      this.stepDefeat(e)
      return
    }
    if (e.st === ST.HURT && e.t > 50) {
      e.st = ST.NORMAL
      e.t = 0
    }
    const p = this.nearestPlayer(e.x)
    if (e.ground && e.st !== ST.HURT) {
      const angry = e.hp <= 1
      if (e.t % (angry ? 70 : 100) === 0) {
        e.vy = angry ? -6 : -4.8
        e.vx = p ? Math.sign(p.ps.x - e.x) * (angry ? 1.6 : 1.1) : 0
      } else {
        // pace near the home point
        if (Math.abs(e.x - e.ax) > 64) e.vx = Math.sign(e.ax - e.x) * 0.6
        else if (Math.abs(e.vx) < 0.1) e.vx = -0.6
        else e.vx = Math.sign(e.vx) * 0.6
      }
      if (e.t % (angry ? 90 : 130) === 45 && p) {
        const dx = p.ps.x - e.x
        this.addEnt('mudball', e.x, e.y - 24, { vx: Math.max(-2.6, Math.min(2.6, dx / 50)), vy: -4.2 })
        this.sfx('splash')
      }
    }
    const wasGround = e.ground
    this.moveBody(e, 0.25, true)
    if (e.ground && !wasGround) {
      this.events.push({ e: 'shake', n: 6 })
      this.sfx('thud')
    }
    e.dir = p ? Math.sign(p.ps.x - e.x) || e.dir : e.dir
  }

  private stepNian(e: Ent): void {
    if (e.st === ST.DEFEAT) {
      this.stepDefeat(e)
      return
    }
    const p = this.nearestPlayer(e.x)
    switch (e.st) {
      case ST.HURT:
        e.vx *= 0.9
        this.moveBody(e, 0.3, false)
        if (e.t > 50) {
          e.st = ST.NORMAL
          e.t = 0
        }
        return
      case ST.ROAR:
        e.vx = 0
        this.moveBody(e, 0.3, false)
        if (e.t === 1) {
          this.sfx('roar')
          this.events.push({ e: 'shake', n: 14 })
        }
        if (e.t > 45) {
          e.st = ST.CHARGE
          e.t = 0
          e.vx = (p ? Math.sign(p.ps.x - e.x) : e.dir) * (e.hp <= 2 ? 3.6 : 3)
          e.dir = Math.sign(e.vx)
        }
        return
      case ST.CHARGE:
        this.moveBody(e, 0.3, false)
        if (e.hitWall || e.t > 160) {
          e.st = ST.STUN
          e.t = 0
          e.vx = -e.dir * 1.5
          e.vy = -3
          this.events.push({ e: 'shake', n: 12 })
          this.sfx('thud')
        }
        return
      case ST.STUN:
        e.vx *= 0.92
        this.moveBody(e, 0.3, false)
        if (e.t > 110) {
          e.st = ST.NORMAL
          e.t = 0
        }
        return
      default: {
        if (p) e.dir = Math.sign(p.ps.x - e.x) || e.dir
        e.vx = e.dir * 0.5
        this.moveBody(e, 0.3, false)
        if (e.t % 90 === 40 && e.hp <= 3) {
          this.addEnt('flame', e.x + e.dir * 20, e.y - 18, { vx: e.dir * 2.2, ay: e.y - 18 })
          this.sfx('fire')
        }
        if (e.t > 150) {
          e.st = ST.ROAR
          e.t = 0
        }
      }
    }
  }

  private stepDefeat(e: Ent): void {
    // the bridge falls away tile by tile and the boss drops into the pit
    if (e.t % 4 === 0) {
      for (let tx = 0; tx < this.area.width; tx++) {
        if (cell(this.area, tx, 12) === T.BRIDGE) {
          this.edit(tx, 12, T.EMPTY)
          this.sfx('break')
          break
        }
      }
    }
    if (e.t > 30) {
      e.vy = Math.min(5, e.vy + 0.2)
      e.y += e.vy
    }
    if (e.t === 140) {
      // the stolen 年灯 drops down; touching it ends the level
      this.addEnt('lamp', (this.area.width - 4) * TILE, 3 * TILE, {})
    }
    if (e.y > VIEW_H + 80 && e.t > 140) e.dead = true
  }

  private hitBoss(e: Ent, p: Player | null): void {
    if (e.st === ST.HURT || e.st === ST.DEFEAT) return
    e.hp--
    this.sfx('bosshit')
    this.events.push({ e: 'shake', n: 8 })
    if (p) this.points(p, 1000, e.x, e.y - e.h)
    if (e.hp <= 0) {
      e.st = ST.DEFEAT
      e.t = 0
      e.vx = 0
      e.vy = -3
      this.bossDown = true
      this.music('')
      if (p) this.points(p, 5000, e.x, e.y - e.h - 12)
      this.ents.forEach((o) => {
        if (o !== e && isEnemy(o.k)) this.flip(o, null, 1)
      })
    } else {
      e.st = ST.HURT
      e.t = 0
      e.vx = 0
    }
  }

  private flip(e: Ent, p: Player | null, dir: number): void {
    if (e.k === 'firebar' || e.k === 'spark' || e.k === 'flame' || e.k === 'mudball') return
    if (e.k === 'mudking' || e.k === 'nian') {
      this.hitBoss(e, p)
      return
    }
    if (e.k === 'rat') {
      e.dead = true
      this.events.push({ e: 'puff', x: e.x, y: e.y - 10 })
    }
    e.st = ST.FLIP
    e.t = 0
    e.vy = -3.5
    e.vx = dir * 1.2
    this.sfx('kick')
    this.points(p, 200, e.x, e.y - e.h)
  }

  // ───────────────────────── contacts ─────────────────────────

  private interact(): void {
    const players = this.players.filter((p) => p.st === 'play')
    for (const p of players) {
      const pb = pbox(p)
      for (const e of this.ents) {
        if (e.dead || p.st !== 'play') continue
        if (e.k === 'firebar') {
          if (this.firebarHits(e, pb)) this.hurt(p)
          continue
        }
        const eb = ebox(e)
        if (!overlap(pb, eb)) continue
        this.contact(p, e, pb, eb)
      }
    }
    // players bouncing on each other's heads
    if (players.length === 2) {
      const [a, b] = players
      for (const [top, bot] of [
        [a, b],
        [b, a]
      ]) {
        const tb = pbox(top)
        const bb = pbox(bot)
        if (top.ps.vy > 0 && overlap(tb, bb) && top.ps.y - top.ps.vy <= bb.t + 4) {
          top.ps.y = bb.t
          stompBounce(top.ps, (top.btn & BTN.J) !== 0)
          bot.squash = 12
          this.sfx('stomp')
          break
        }
      }
    }
    // fireballs, rolling balls and the boss lamp
    for (const f of this.ents) {
      if (f.dead) continue
      if (f.k === 'fire') {
        for (const e of this.ents) {
          if (e.dead || e === f || !isEnemy(e.k) || e.st === ST.FLIP) continue
          if (e.k === 'rat' && e.st === ST.EMERGE) continue
          if (overlap(ebox(f), ebox(e))) {
            f.dead = true
            this.events.push({ e: 'puff', x: f.x, y: f.y })
            if (e.k === 'firebar' || e.k === 'spark') break
            const owner = this.players[f.owner] || null
            if (e.k === 'mudking' || e.k === 'nian') {
              e.len++
              this.sfx('thud')
              if (e.len >= 4) {
                e.len = 0
                this.hitBoss(e, owner)
              }
            } else this.flip(e, owner, Math.sign(f.vx))
            break
          }
        }
      } else if ((f.k === 'hog' && f.st === ST.ROLL) || (f.k === 'cracker' && f.st === ST.LIT && Math.abs(f.vx) > 1)) {
        for (const e of this.ents) {
          if (e.dead || e === f || !isEnemy(e.k) || e.st === ST.FLIP || e.k === 'firebar' || e.k === 'spark') continue
          if (overlap(ebox(f), ebox(e))) {
            if (e.k === 'hog' && e.st === ST.ROLL) {
              this.flip(e, null, Math.sign(f.vx))
              this.flip(f, null, -Math.sign(f.vx))
            } else this.flip(e, this.players[f.owner] || null, Math.sign(f.vx))
            if (f.k === 'cracker') this.explode(f)
          }
        }
      }
    }
    // walkers turn when they bump into each other
    const walkers = this.ents.filter((e) => !e.dead && (e.k === 'mud' || e.k === 'cracker' || e.k === 'hog') && e.st === ST.NORMAL)
    for (let i = 0; i < walkers.length; i++)
      for (let j = i + 1; j < walkers.length; j++) {
        const a = walkers[i]
        const b = walkers[j]
        if (overlap(ebox(a), ebox(b)) && Math.sign(b.x - a.x) === Math.sign(a.vx)) {
          a.vx = -a.vx
          b.vx = -b.vx
        }
      }
  }

  firebarHits(e: Ent, pb: { l: number; t: number; r: number; b: number }): boolean {
    const ang = this.firebarAngle(e)
    for (let i = 0; i < e.len; i++) {
      const ex = e.ax + Math.cos(ang) * i * 8
      const ey = e.ay + Math.sin(ang) * i * 8
      const nx = Math.max(pb.l, Math.min(ex, pb.r))
      const ny = Math.max(pb.t, Math.min(ey, pb.b))
      if ((nx - ex) * (nx - ex) + (ny - ey) * (ny - ey) < 3.5 * 3.5) return true
    }
    return false
  }

  firebarAngle(e: Ent): number {
    return e.dir * this.frame * 0.032
  }

  private contact(p: Player, e: Ent, pb: { l: number; t: number; r: number; b: number }, eb: { l: number; t: number; r: number; b: number }): void {
    const ps = p.ps
    // items
    if (e.k === 'bun' || e.k === 'chili' || e.k === 'bell' || e.k === 'haw') {
      if (e.st === ST.EMERGE && e.y > e.ay + 8) return
      e.dead = true
      if (e.k === 'bun') {
        this.grow(p, 1)
        this.points(p, 1000, e.x, e.y - 16)
      } else if (e.k === 'chili') {
        if (ps.power === 0) this.grow(p, 1)
        else this.grow(p, 2)
        this.points(p, 1000, e.x, e.y - 16)
      } else if (e.k === 'bell') {
        p.bell = 600
        this.sfx('bell')
        this.music('bell')
        this.points(p, 1000, e.x, e.y - 16)
      } else this.oneUp(p, e.x, e.y - 16)
      return
    }
    if (e.k === 'lamp') {
      e.dead = true
      this.sfx('pole')
      this.music('clear')
      for (const q of this.players) if (q.st !== 'out') q.st = 'done'
      this.points(p, 5000, e.x, e.y - 20)
      this.phase = 'clear'
      this.phaseT = 0
      return
    }
    if (e.k === 'fire' || e.st === ST.FLIP || e.st === ST.SQUASH || e.st === ST.DEFEAT) return
    if (e.k === 'rat' && e.st === ST.EMERGE) return

    // 金钟罩: anything harmful you touch is knocked away
    if (p.bell > 0) {
      if (e.k === 'firebar' || e.k === 'spark' || e.k === 'flame' || e.k === 'mudball') return
      p.combo++
      this.flip(e, p, Math.sign(e.x - ps.x) || 1)
      return
    }

    const prevFeet = ps.y - ps.vy
    const stomping = ps.vy > 0 && prevFeet <= eb.t + Math.max(6, ps.vy + 2)
    const stompable = e.k === 'mud' || e.k === 'hog' || e.k === 'kite' || e.k === 'cracker' || e.k === 'fish' || e.k === 'mudking' || e.k === 'nian'

    if (stomping && stompable) {
      ps.y = eb.t
      stompBounce(ps, (p.btn & BTN.J) !== 0)
      this.sfx('stomp')
      if (e.k === 'mudking' || e.k === 'nian') {
        if (e.st !== ST.HURT) this.hitBoss(e, p)
        return
      }
      const v = STOMP_POINTS[Math.min(p.combo, STOMP_POINTS.length - 1)]
      if (p.combo >= STOMP_POINTS.length) this.oneUp(p, e.x, e.y - e.h)
      else this.points(p, v, e.x, e.y - e.h)
      p.combo++
      switch (e.k) {
        case 'mud':
          e.st = ST.SQUASH
          e.t = 0
          e.vx = 0
          break
        case 'hog':
          if (e.st === ST.NORMAL || e.st === ST.ROLL) {
            e.st = ST.BALL
            e.t = 0
            e.vx = 0
          } else {
            e.st = ST.ROLL
            e.t = 0
            e.vx = (ps.x < e.x ? 1 : -1) * 3.8
            e.owner = p.id
            this.sfx('kick')
          }
          break
        case 'cracker':
          if (e.st === ST.NORMAL) {
            e.st = ST.LIT
            e.t = 0
            e.vx = 0
            this.sfx('fuse')
          } else {
            e.vx = (ps.x < e.x ? 1 : -1) * 3
            e.owner = p.id
            this.sfx('kick')
          }
          break
        default:
          this.flip(e, p, Math.sign(e.x - ps.x) || 1)
      }
      return
    }

    // side contact with a resting ball or a lit cracker: kick it
    if ((e.k === 'hog' && e.st === ST.BALL) || (e.k === 'cracker' && e.st === ST.LIT && Math.abs(e.vx) < 0.5)) {
      e.st = e.k === 'hog' ? ST.ROLL : ST.LIT
      if (e.k === 'hog') e.t = 0
      e.vx = (ps.x < e.x ? 1 : -1) * (e.k === 'hog' ? 3.8 : 3)
      e.x += Math.sign(e.vx) * 4
      e.owner = p.id
      this.sfx('kick')
      this.points(p, 400, e.x, e.y - e.h)
      return
    }
    if (e.k === 'hog' && e.st === ST.ROLL && e.owner === p.id && e.t < 12) return
    this.hurt(p)
    void pb
  }

  // ───────────────────────── camera ─────────────────────────

  private stepCamera(): void {
    const max = this.maxCam()
    if (this.area.def.boss && this.cam >= max - 40) {
      this.cam = Math.min(max, this.cam + 2)
      return
    }
    const ps = this.players.filter((p) => p.st === 'play' || p.st === 'walk' || p.st === 'pole')
    if (!ps.length) return
    const lead = Math.max(...ps.map((p) => p.ps.x))
    const target = Math.min(max, lead - VIEW_W * 0.42)
    if (target > this.cam) this.cam += Math.min(target - this.cam, 6)
  }
}

export function isEnemy(k: string): boolean {
  return k === 'mud' || k === 'hog' || k === 'kite' || k === 'rat' || k === 'fish' || k === 'cracker' || k === 'firebar' || k === 'spark' || k === 'mudking' || k === 'nian' || k === 'mudball' || k === 'flame'
}

export { PHYS }
