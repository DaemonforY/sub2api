// The game world of 椰风岛大冒险. Pure logic: no DOM, no audio. The host (or a local game) runs it at 60 Hz;
// online guests only run `movePlayer` for their own hero as prediction.
import {
  W,
  H,
  T,
  ROWS,
  TS,
  TO,
  TW,
  TL,
  TB,
  IN_L,
  IN_R,
  IN_J,
  IN_F,
  IN_D,
  EV,
  ENERGY_MAX,
  FAIRY_FRAMES,
  CLOUD_FRAMES,
  IT_BOOM,
  IT_BOARD,
  IT_FAIRY,
  IT_CLOUD,
  IT_BASKET,
  IT_GOLD,
  type GameEvent,
  type Kind,
  type Phase,
  type Spawn,
  type Stage
} from './defs'
import { stages, STAGE_COUNT } from './level'

export const PW = 10
export const PH = 22

export type PState = 'play' | 'dying' | 'ghost' | 'out' | 'win'

export interface Player {
  i: number
  on: boolean
  st: PState
  x: number
  y: number
  vx: number
  vy: number
  face: number
  ground: boolean
  coyote: number
  jbuf: number
  jheld: boolean
  plat: number
  energy: number
  weapon: number // 0 none, 1 boomerang, 2 golden
  board: boolean
  fairy: number
  cloud: number
  inv: number
  trip: number
  tripCd: number
  throwT: number
  throwCd: number
  runT: number
  airT: number
  dyingT: number
  score: number
  fruits: number
  combo: number
  lastFruit: number
  bits: number
  prev: number
}

export interface Ent {
  id: number
  k: Kind
  x: number
  y: number
  w: number
  h: number
  vx: number
  vy: number
  dir: number
  t: number
  st: number
  hp: number
  a: number
  b: number
  c: number
  ox: number
  oy: number
  flash: number
  ground: boolean
  owner: number
  dead: boolean
}

/** What player movement needs; the guest builds one from snapshots to predict its own hero. */
export interface PhysCtx {
  tiles: Uint8Array
  cols: number
  camX: number
  plats: Ent[]
  springs: Ent[]
  events?: GameEvent[]
}

const GRAV = 0.34
const GRAV_HOLD = 0.19
const MAX_FALL = 5.6
const DRAIN = ENERGY_MAX / (60 * 48)

export function newPlayer(i: number, on: boolean): Player {
  return {
    i,
    on,
    st: on ? 'play' : 'out',
    x: 0,
    y: 0,
    vx: 0,
    vy: 0,
    face: 1,
    ground: false,
    coyote: 0,
    jbuf: 0,
    jheld: false,
    plat: 0,
    energy: ENERGY_MAX,
    weapon: 0,
    board: false,
    fairy: 0,
    cloud: 0,
    inv: 0,
    trip: 0,
    tripCd: 0,
    throwT: 0,
    throwCd: 0,
    runT: 0,
    airT: 0,
    dyingT: 0,
    score: 0,
    fruits: 0,
    combo: 0,
    lastFruit: -999,
    bits: 0,
    prev: 0
  }
}

export function tileAt(tiles: Uint8Array, cols: number, px: number, py: number): number {
  const c = Math.floor(px / T)
  const r = Math.floor(py / T)
  if (c < 0 || c >= cols) return TS
  if (r < 0 || r >= ROWS) return 0
  return tiles[r * cols + c]
}

const isSolid = (t: number) => t === TS || t === TB

export function overlap(ax: number, ay: number, aw: number, ah: number, bx: number, by: number, bw: number, bh: number): boolean {
  return ax < bx + bw && ax + aw > bx && ay < by + bh && ay + ah > by
}

/**
 * Move a box through the tile map, axis by axis. Returns flags: 1 hit wall, 2 landed, 4 bumped head.
 * `oneWay` lets the box stand on one-way planks unless it is dropping through.
 */
export function moveBox(
  ctx: { tiles: Uint8Array; cols: number },
  b: { x: number; y: number; vx: number; vy: number },
  w: number,
  h: number,
  oneWay = true
): number {
  let flags = 0
  const { tiles, cols } = ctx
  // X
  b.x += b.vx
  if (b.vx > 0) {
    const ex = b.x + w
    for (let yy = b.y + 1; yy < b.y + h - 1; yy += 8) {
      if (isSolid(tileAt(tiles, cols, ex, yy)) || isSolid(tileAt(tiles, cols, ex, b.y + h - 2))) {
        b.x = Math.floor(ex / T) * T - w - 0.01
        b.vx = 0
        flags |= 1
        break
      }
    }
  } else if (b.vx < 0) {
    for (let yy = b.y + 1; yy < b.y + h - 1; yy += 8) {
      if (isSolid(tileAt(tiles, cols, b.x, yy)) || isSolid(tileAt(tiles, cols, b.x, b.y + h - 2))) {
        b.x = Math.floor(b.x / T) * T + T
        b.vx = 0
        flags |= 1
        break
      }
    }
  }
  // Y
  const prevBottom = b.y + h
  b.y += b.vy
  if (b.vy > 0) {
    const by = b.y + h
    for (const xx of [b.x + 1, b.x + w - 1]) {
      const t = tileAt(tiles, cols, xx, by)
      const top = Math.floor(by / T) * T
      if (isSolid(t) || (oneWay && t === TO && prevBottom <= top + 0.5)) {
        b.y = top - h
        b.vy = 0
        flags |= 2
        break
      }
    }
  } else if (b.vy < 0) {
    for (const xx of [b.x + 1, b.x + w - 1]) {
      if (isSolid(tileAt(tiles, cols, xx, b.y))) {
        b.y = Math.floor(b.y / T) * T + T
        b.vy = 0
        flags |= 4
        break
      }
    }
  }
  return flags
}

/** Hero movement: running, jumping, planks, moving platforms and springs. Shared by host and guest prediction. */
export function movePlayer(p: Player, bits: number, ctx: PhysCtx): void {
  const pressed = bits & ~p.prev
  let left = !!(bits & IN_L)
  let right = !!(bits & IN_R)
  let jump = !!(bits & IN_J)
  if (p.trip > 0) {
    left = right = jump = false
  }
  // carried by a moving platform
  if (p.plat) {
    const pl = ctx.plats.find((e) => e.id === p.plat)
    if (pl && overlap(p.x, p.y + PH, PW, 2, pl.x, pl.y - 1, pl.w, 4)) {
      p.x += pl.vx
      p.y = pl.y - PH
    } else p.plat = 0
  }
  const maxV = p.board ? 2.75 : 1.9
  const acc = p.board ? 0.15 : p.ground ? 0.13 : 0.1
  const dir = (right ? 1 : 0) - (left ? 1 : 0)
  if (dir) {
    p.face = dir
    if (p.vx * dir < maxV) p.vx += dir * acc * (p.vx * dir < 0 ? 2.2 : 1)
    if (p.vx * dir > maxV) p.vx = dir * maxV
  } else {
    const fr = p.board ? 0.03 : p.ground ? 0.16 : 0.05
    if (Math.abs(p.vx) <= fr) p.vx = 0
    else p.vx -= Math.sign(p.vx) * fr
  }
  if (p.trip > 0) p.vx *= 0.9

  if (pressed & IN_J) p.jbuf = 7
  else if (p.jbuf > 0) p.jbuf--
  if (p.ground) p.coyote = 6
  else if (p.coyote > 0) p.coyote--

  // drop through a plank: down + jump
  const dropping = !!(bits & IN_D) && p.jbuf > 0 && p.ground && !isSolid(tileAt(ctx.tiles, ctx.cols, p.x + PW / 2, p.y + PH + 1))
  if (dropping && tileAt(ctx.tiles, ctx.cols, p.x + PW / 2, p.y + PH + 1) === TO) {
    p.y += 3
    p.jbuf = 0
    p.ground = false
    p.coyote = 0
  } else if (p.jbuf > 0 && p.coyote > 0) {
    p.vy = -(4.3 + Math.min(1, Math.abs(p.vx) / maxV) * 0.75 + (p.board ? 0.4 : 0))
    p.jbuf = 0
    p.coyote = 0
    p.ground = false
    p.plat = 0
    p.jheld = true
    ctx.events?.push([EV.jump, p.x + PW / 2, p.y + PH, p.i])
  }
  if (!jump) p.jheld = false
  const g = p.vy < 0 && p.jheld ? GRAV_HOLD : GRAV
  p.vy = Math.min(MAX_FALL, p.vy + g)

  const wasGround = p.ground
  const fl = moveBox(ctx, p, PW, PH, true)
  p.ground = !!(fl & 2)
  if (fl & 4) p.jheld = false
  // moving platforms (one-way tops)
  if (p.vy >= 0 && !p.ground) {
    for (const pl of ctx.plats) {
      if (pl.st === 3) continue
      const bottom = p.y + PH
      if (p.x + PW > pl.x + 1 && p.x < pl.x + pl.w - 1 && bottom >= pl.y - 0.5 && bottom - p.vy - Math.max(0, pl.vy) <= pl.y + 2) {
        p.y = pl.y - PH
        p.vy = 0
        p.ground = true
        p.plat = pl.id
        if (pl.a === 2 && pl.st === 0) pl.st = 1
        break
      }
    }
  } else if (p.ground) p.plat = p.plat && ctx.plats.some((e) => e.id === p.plat) ? p.plat : 0
  // springs
  if (p.vy >= 0) {
    for (const s of ctx.springs) {
      if (p.x + PW > s.x && p.x < s.x + s.w && p.y + PH >= s.y && p.y + PH <= s.y + 10) {
        p.vy = -7.4
        p.ground = false
        p.jheld = true
        s.st = 10
        ctx.events?.push([EV.spring, s.x + s.w / 2, s.y, 0])
        break
      }
    }
  }
  if (p.ground && !wasGround && p.airT > 8) ctx.events?.push([EV.land, p.x + PW / 2, p.y + PH, p.i])
  p.airT = p.ground ? 0 : p.airT + 1
  // the screen edges hold the heroes in
  const minX = ctx.camX + 2
  const maxX = ctx.camX + W - PW - 2
  if (p.x < minX) {
    p.x = minX
    if (p.vx < 0) p.vx = 0
  }
  if (p.x > maxX) {
    p.x = maxX
    if (p.vx > 0) p.vx = 0
  }
  p.runT += p.ground ? Math.abs(p.vx) : 0
  p.prev = bits
}

const SIZES: Partial<Record<Kind, [number, number]>> = {
  fruit: [12, 12],
  clam: [20, 14],
  item: [14, 12],
  rock: [12, 8],
  fire: [10, 13],
  crab: [16, 13],
  gull: [16, 9],
  fish: [12, 12],
  frog: [14, 11],
  spider: [12, 10],
  spiky: [14, 14],
  bat: [14, 9],
  flame: [10, 14],
  boulder: [20, 20],
  geyser: [14, 20],
  spring: [16, 8],
  flag: [12, 44],
  plat: [48, 8],
  boom: [10, 10],
  shot: [8, 8],
  sun: [16, 16]
}

const SCORE: Partial<Record<Kind, number>> = { crab: 100, gull: 200, fish: 200, frog: 150, spider: 200, spiky: 300, bat: 300, flame: 400, boulder: 500 }
export const FRUIT_ENERGY = [9, 12, 14, 16, 12, 100]
export const FRUIT_SCORE = [50, 100, 200, 300, 150, 1000]
const ENEMIES = new Set<Kind>(['crab', 'gull', 'fish', 'frog', 'spider', 'spiky', 'bat', 'flame', 'boulder'])

export interface World {
  stageIndex: number
  stage: Stage
  twoP: boolean
  players: Player[]
  ents: Ent[]
  nextId: number
  camX: number
  lock: boolean
  frame: number
  phase: Phase
  phaseT: number
  lives: number
  cp: number
  events: GameEvent[]
  shake: number
  hitstop: number
  bossMax: number
  fruitTimer: number
  totalFruits: number
  god?: boolean
}

function makeEnt(w: World, s: Spawn): Ent {
  const [ew, eh] = SIZES[s.k] ?? [12, 12]
  const centred = s.k === 'fruit' || s.k === 'gull' || s.k === 'bat' || s.k === 'spider' || s.k === 'plat' || s.k === 'fish' || s.k === 'geyser'
  const e: Ent = {
    id: w.nextId++,
    k: s.k,
    x: s.x - ew / 2,
    y: centred ? s.y - eh / 2 : s.y - eh,
    w: ew,
    h: eh,
    vx: 0,
    vy: 0,
    dir: -1,
    t: 0,
    st: 0,
    hp: 1,
    a: s.a ?? 0,
    b: s.b ?? 0,
    c: 0,
    ox: 0,
    oy: 0,
    flash: 0,
    ground: false,
    owner: -1,
    dead: false
  }
  if (s.k === 'plat') e.y = s.y - 8
  if (s.k === 'flame' || s.k === 'boulder') e.hp = 2
  if (s.k === 'spider') e.y = s.y
  if (s.k === 'geyser') {
    e.y = H - 16
    e.c = (e.id * 37) % 120
  }
  if (s.k === 'fish') {
    e.y = H + 4
    e.c = (e.id * 29) % 90
  }
  e.ox = e.x
  e.oy = e.y
  return e
}

export function spawnEnt(w: World, k: Kind, x: number, y: number, init: Partial<Ent> = {}): Ent {
  const [ew, eh] = SIZES[k] ?? [12, 12]
  const e = makeEnt(w, { k, x, y })
  e.w = init.w ?? ew
  e.h = init.h ?? eh
  e.x = x - e.w / 2
  e.y = y - e.h / 2
  Object.assign(e, init)
  w.ents.push(e)
  return e
}

export function createWorld(twoP: boolean, stageIndex = 0): World {
  const w: World = {
    stageIndex,
    stage: stages()[stageIndex],
    twoP,
    players: [newPlayer(0, true), newPlayer(1, twoP)],
    ents: [],
    nextId: 1,
    camX: 0,
    lock: false,
    frame: 0,
    phase: 'play',
    phaseT: 0,
    lives: twoP ? 4 : 3,
    cp: 0,
    events: [],
    shake: 0,
    hitstop: 0,
    bossMax: 1,
    fruitTimer: 0,
    totalFruits: 0
  }
  loadStage(w, stageIndex, 0)
  return w
}

export function loadStage(w: World, index: number, cp: number) {
  w.stageIndex = index
  w.stage = stages()[index]
  w.cp = cp
  w.ents = []
  w.lock = false
  w.phase = 'play'
  w.phaseT = 0
  w.hitstop = 0
  const startX = w.stage.checkpoints[cp]
  w.camX = Math.max(0, startX - 80)
  for (const s of w.stage.spawns) if (s.x > startX + 24) w.ents.push(makeEnt(w, s))
  w.stage.checkpoints.forEach((x, i) => {
    if (i > 0) {
      const f = makeEnt(w, { k: 'flag', x, y: surfaceAt(w, x) })
      f.a = i
      f.st = i <= cp ? 1 : 0
      w.ents.push(f)
    }
  })
  for (const p of w.players) {
    if (!p.on) continue
    p.st = 'play'
    p.x = startX - 6 + p.i * 18
    p.y = surfaceAt(w, p.x + PW / 2) - PH - 8
    p.vx = 0
    p.vy = 0
    p.energy = ENERGY_MAX
    p.weapon = 0
    p.board = false
    p.fairy = 0
    p.cloud = 0
    p.inv = 60
    p.trip = 0
    p.dyingT = 0
    p.plat = 0
    p.face = 1
  }
}

function surfaceAt(w: World, x: number): number {
  for (let r = 0; r < ROWS; r++) {
    const t = tileAt(w.stage.tiles, w.stage.cols, x, r * T + 1)
    if (t === TS || t === TB || t === TO) return r * T
  }
  return ROWS * T
}

function ev(w: World, type: number, x: number, y: number, a = 0) {
  w.events.push([type, Math.round(x), Math.round(y), a])
}

function alivePlayers(w: World): Player[] {
  return w.players.filter((p) => p.on && p.st === 'play')
}

function nearest(w: World, x: number): Player | null {
  let best: Player | null = null
  let bd = Infinity
  for (const p of alivePlayers(w)) {
    const d = Math.abs(p.x + PW / 2 - x)
    if (d < bd) {
      bd = d
      best = p
    }
  }
  return best
}

// ───────────────────────────── player life ─────────────────────────────

function killPlayer(w: World, p: Player, fell = false) {
  if (p.st !== 'play') return
  if (w.god) {
    p.energy = ENERGY_MAX
    if (fell) {
      p.y = 0
      p.vy = 0
    }
    return
  }
  p.st = 'dying'
  p.dyingT = fell ? 40 : 0
  p.vy = fell ? 0 : -5
  p.vx = 0
  p.weapon = 0
  p.board = false
  p.fairy = 0
  p.cloud = 0
  p.trip = 0
  ev(w, EV.die, p.x + PW / 2, p.y + PH / 2, p.i)
  w.shake = Math.max(w.shake, 8)
}

/** Returns true when the hit was absorbed (board lost) or ignored. */
function hurtPlayer(w: World, p: Player): boolean {
  if (p.st !== 'play' || p.inv > 0 || p.fairy > 0) return true
  if (p.board) {
    p.board = false
    p.inv = 90
    p.vx = -1.5 * p.face
    p.vy = -3
    ev(w, EV.boardLost, p.x + PW / 2, p.y + PH, p.i)
    w.shake = Math.max(w.shake, 4)
    return true
  }
  killPlayer(w, p)
  return false
}

function revive(w: World, p: Player, x: number, y: number) {
  p.st = 'play'
  p.x = x
  p.y = y
  p.vx = 0
  p.vy = -3
  p.energy = Math.max(p.energy, ENERGY_MAX * 0.6)
  p.inv = 120
  ev(w, EV.revive, x + PW / 2, y + PH / 2, p.i)
}

function loseLife(w: World) {
  w.lives--
  if (w.lives < 0) {
    w.phase = 'over'
    w.phaseT = 0
    return
  }
  ev(w, EV.life, 0, 0, -1)
  loadStage(w, w.stageIndex, w.cp)
}

export function continueGame(w: World) {
  w.lives = w.twoP ? 4 : 3
  for (const p of w.players) p.score = Math.floor(p.score / 2)
  loadStage(w, w.stageIndex, w.cp)
}

function updateDying(w: World, p: Player) {
  p.dyingT++
  if (p.dyingT < 40) {
    p.vy += 0.25
    p.y += p.vy
  }
  if (p.dyingT < 80) return
  const others = w.players.filter((q) => q !== p && q.on)
  if (others.some((q) => q.st === 'play')) {
    p.st = 'ghost'
    p.x = Math.max(w.camX + 20, Math.min(w.camX + W - 30, p.x))
    p.y = 50
    p.vx = 0
    p.vy = 0
    return
  }
  // wait for a buddy who is still mid-fall; the last one down ends the life
  if (others.some((q) => q.st === 'dying' && q.dyingT < 80)) {
    p.dyingT = 80
    return
  }
  loseLife(w)
}

function updateGhost(w: World, p: Player) {
  const mate = alivePlayers(w)[0]
  if (!mate) return
  const tx = mate.x + (mate.x > w.camX + W / 2 ? -34 : 34)
  const ty = 46 + Math.sin(w.frame / 30 + p.i) * 8
  p.x += Math.max(-1.1, Math.min(1.1, (tx - p.x) * 0.04))
  p.y += (ty - p.y) * 0.05
  p.x = Math.max(w.camX + 6, Math.min(w.camX + W - PW - 6, p.x))
  // the buddy touches the bubble: pop!
  if (Math.abs(mate.x - p.x) < 18 && Math.abs(mate.y + PH / 2 - (p.y + PH / 2)) < 22) revive(w, p, p.x, p.y)
}

// ───────────────────────────── main step ─────────────────────────────

export function physCtx(w: World): PhysCtx {
  return {
    tiles: w.stage.tiles,
    cols: w.stage.cols,
    camX: w.camX,
    plats: w.ents.filter((e) => e.k === 'plat'),
    springs: w.ents.filter((e) => e.k === 'spring'),
    events: w.events
  }
}

export function stepWorld(w: World, inputs: number[]): void {
  w.frame++
  if (w.shake > 0) w.shake--
  if (w.phase === 'over' || w.phase === 'ending') {
    w.phaseT++
    return
  }
  if (w.phase === 'clear') {
    stepClear(w)
    return
  }
  if (w.hitstop > 0) {
    w.hitstop--
    return
  }
  const ctx = physCtx(w)
  for (const p of w.players) {
    if (!p.on) continue
    p.bits = inputs[p.i] ?? 0
    if (p.st === 'play') stepPlayer(w, p, ctx)
    else if (p.st === 'dying') updateDying(w, p)
    else if (p.st === 'ghost') updateGhost(w, p)
    if (w.phase !== 'play') return // a lost life reloads the stage
  }
  updateCamera(w)
  for (const e of w.ents) if (!e.dead) stepEnt(w, e)
  w.ents = w.ents.filter((e) => !e.dead)
  if (w.lock) arenaFruit(w)
}

function stepPlayer(w: World, p: Player, ctx: PhysCtx) {
  const pressed = p.bits & ~p.prev
  ctx.camX = w.camX
  movePlayer(p, p.bits, ctx)
  if (p.inv > 0) p.inv--
  if (p.trip > 0) p.trip--
  if (p.tripCd > 0) p.tripCd--
  if (p.throwT > 0) p.throwT--
  if (p.throwCd > 0) p.throwCd--
  if (p.fairy > 0) p.fairy--
  if (p.cloud > 0) p.cloud--
  // energy drains all the time; the cloud makes it three times faster
  p.energy -= DRAIN * (p.cloud > 0 ? 3 : 1)
  if (p.energy < ENERGY_MAX * 0.25 && w.frame % 40 === 0) ev(w, EV.lowEnergy, p.x, p.y, p.i)
  if (p.energy <= 0) {
    p.energy = 0
    killPlayer(w, p)
    return
  }
  // throw
  if (pressed & IN_F && p.weapon > 0 && p.throwCd === 0 && p.trip === 0) {
    const mine = w.ents.filter((e) => e.k === 'boom' && e.owner === p.i).length
    if (mine < 2) {
      const gold = p.weapon === 2
      spawnEnt(w, 'boom', p.x + PW / 2 + p.face * 7, p.y + 10, {
        owner: p.i,
        a: gold ? 1 : 0,
        dir: p.face,
        vx: p.face * (gold ? 6.4 : 5.6) + p.vx * 0.4,
        w: gold ? 14 : 10,
        h: gold ? 14 : 10
      })
      p.throwT = 14
      p.throwCd = 11
      ev(w, EV.throw, p.x + PW / 2, p.y + 10, p.i)
    }
  }
  // fell into a pit, the sea or lava
  const feet = tileAt(w.stage.tiles, w.stage.cols, p.x + PW / 2, p.y + PH - 2)
  if (feet === TW || feet === TL || p.y > H + 4) {
    if (feet === TW || feet === TL) ev(w, EV.splash, p.x + PW / 2, H - 14, feet === TL ? 1 : 0)
    killPlayer(w, p, true)
    if (p.st !== 'play') p.y = H + 40
  }
}

function updateCamera(w: World) {
  const alive = alivePlayers(w)
  if (!alive.length) return
  let lead = -Infinity
  let trail = Infinity
  for (const p of alive) {
    lead = Math.max(lead, p.x)
    trail = Math.min(trail, p.x)
  }
  let target = lead + PW / 2 - W * 0.42
  if (alive.length > 1) target = Math.min(target, trail - 24)
  const maxCam = w.stage.arenaX
  w.camX = Math.max(w.camX, Math.min(target, maxCam))
  if (!w.lock && w.camX >= maxCam - 0.5) {
    w.lock = true
    w.camX = maxCam
    startBoss(w)
  }
  // flags
  for (const e of w.ents) {
    if (e.k !== 'flag' || e.st) continue
    if (alive.some((p) => p.x + PW / 2 >= e.x)) {
      e.st = 1
      w.cp = Math.max(w.cp, e.a)
      ev(w, EV.flag, e.x, e.y, 0)
      for (const p of w.players) if (p.on && p.st === 'ghost') revive(w, p, e.x + 4, e.y + 10)
    }
  }
}

function arenaFruit(w: World) {
  w.fruitTimer++
  if (w.fruitTimer % 330 === 0 && w.ents.some((e) => e.k === 'boss')) {
    const x = w.camX + 40 + ((w.frame * 97) % (W - 80))
    spawnEnt(w, 'fruit', x, -10, { a: (w.frame >> 3) % 5, st: 1, c: 600 })
  }
}

// ───────────────────────────── entities ─────────────────────────────

function active(w: World, e: Ent, margin = 40): boolean {
  return e.x + e.w > w.camX - margin && e.x < w.camX + W + margin
}

function ko(w: World, e: Ent, dir: number, by: Player | null) {
  e.st = -1
  e.vy = -3.4
  e.vx = dir * 1.2
  e.t = 0
  const pts = SCORE[e.k] ?? 100
  if (by) by.score += pts
  ev(w, EV.kill, e.x + e.w / 2, e.y + e.h / 2, pts)
  w.shake = Math.max(w.shake, 3)
}

function stepEnt(w: World, e: Ent) {
  if (e.flash > 0) e.flash--
  if (e.st === -1 && e.k !== 'boss') {
    // knocked-out enemies spin off the bottom of the screen
    e.t++
    e.vy += 0.25
    e.x += e.vx
    e.y += e.vy
    if (e.y > H + 40) e.dead = true
    return
  }
  const always = e.k === 'boom' || e.k === 'shot' || e.k === 'boss' || e.k === 'item' || e.k === 'sun' || e.k === 'plat'
  if (!always && e.x + e.w < w.camX - 64) {
    e.dead = true
    return
  }
  if (!always && !active(w, e, e.k === 'spiky' || e.k === 'boulder' ? 0 : 40)) return
  e.t++
  switch (e.k) {
    case 'fruit':
      return fruitStep(w, e)
    case 'clam':
      return clamStep(w, e)
    case 'item':
      return itemStep(w, e)
    case 'rock':
      return rockStep(w, e)
    case 'fire':
      return hazardTouch(w, e, false)
    case 'crab':
      return crabStep(w, e)
    case 'gull':
      return gullStep(w, e)
    case 'fish':
      return fishStep(w, e)
    case 'frog':
      return frogStep(w, e)
    case 'spider':
      return spiderStep(w, e)
    case 'spiky':
    case 'boulder':
      return rollStep(w, e)
    case 'bat':
      return batStep(w, e)
    case 'flame':
      return flameStep(w, e)
    case 'geyser':
      return geyserStep(w, e)
    case 'spring':
      if (e.st > 0) e.st--
      return
    case 'plat':
      return platStep(w, e)
    case 'boom':
      return boomStep(w, e)
    case 'shot':
      return shotStep(w, e)
    case 'boss':
      return bossStep(w, e)
    case 'sun':
      return sunStep(w, e)
    default:
  }
}

function touching(w: World, e: Ent, shrink = 2): Player[] {
  return alivePlayers(w).filter((p) => overlap(p.x + 1, p.y + 2, PW - 2, PH - 2, e.x + shrink, e.y + shrink, e.w - shrink * 2, e.h - shrink * 2))
}

/** Enemy contact: the fairy knocks them out, otherwise the hero gets hurt. */
function enemyTouch(w: World, e: Ent) {
  for (const p of touching(w, e)) {
    if (p.fairy > 0) {
      ko(w, e, p.face, p)
      return
    }
    hurtPlayer(w, p)
  }
}

function hazardTouch(w: World, e: Ent, killable: boolean) {
  for (const p of touching(w, e, 2)) {
    if (p.fairy > 0) {
      if (killable) ko(w, e, p.face, p)
      continue
    }
    hurtPlayer(w, p)
  }
}

function fall(w: World, e: Ent, g = GRAV) {
  e.vy = Math.min(MAX_FALL, e.vy + g)
  const fl = moveBox({ tiles: w.stage.tiles, cols: w.stage.cols }, e, e.w, e.h, true)
  e.ground = !!(fl & 2)
  if (e.y > H + 20) e.dead = true
  return fl
}

function fruitStep(w: World, e: Ent) {
  if (e.st === 1) {
    fall(w, e, 0.15)
    if (--e.c <= 0) e.dead = true
  }
  for (const p of touching(w, e, 0)) {
    e.dead = true
    p.energy = Math.min(ENERGY_MAX, p.energy + FRUIT_ENERGY[e.a])
    p.score += FRUIT_SCORE[e.a]
    p.fruits++
    w.totalFruits++
    p.combo = w.frame - p.lastFruit < 50 ? p.combo + 1 : 0
    p.lastFruit = w.frame
    ev(w, EV.fruit, e.x + 6, e.y + 6, e.a * 16 + Math.min(15, p.combo))
    if (w.totalFruits % 60 === 0) {
      w.lives++
      ev(w, EV.life, e.x, e.y, 1)
    }
    return
  }
}

function openClam(w: World, e: Ent) {
  if (e.st !== 0) return
  e.st = 1
  e.t = 0
  ev(w, EV.clam, e.x + e.w / 2, e.y, e.a)
}

function clamStep(w: World, e: Ent) {
  if (e.st === 0) {
    if (touching(w, e, 0).length) openClam(w, e)
  } else if (e.st === 1 && e.t >= 14) {
    e.st = 2
    spawnEnt(w, 'item', e.x + e.w / 2, e.y - 4, { a: e.a, vy: e.a === IT_CLOUD ? -1 : -3.8, vx: 0.4, c: 0 })
  }
}

function itemStep(w: World, e: Ent) {
  if (e.a === IT_CLOUD) {
    // the grumpy cloud drifts after the nearest hero
    e.y += e.vy
    e.vy *= 0.94
    const p = nearest(w, e.x)
    if (e.t > 30 && p) {
      const dx = p.x + PW / 2 - (e.x + e.w / 2)
      const dy = p.y - 14 - e.y
      e.x += Math.max(-0.75, Math.min(0.75, dx * 0.05))
      e.y += Math.max(-0.6, Math.min(0.6, dy * 0.05))
    }
    if (e.t > 30)
      for (const q of touching(w, e, 0)) {
        if (q.fairy > 0) continue
        q.cloud = CLOUD_FRAMES
        e.dead = true
        ev(w, EV.cloud, q.x, q.y, q.i)
        return
      }
    if (e.t > 420 || e.x + e.w < w.camX) e.dead = true
    return
  }
  fall(w, e, 0.22)
  if (e.ground) e.vx = 0
  if (e.t > 720) e.dead = true
  if (e.t < 12) return
  for (const p of touching(w, e, 0)) {
    e.dead = true
    applyItem(w, p, e.a)
    return
  }
}

function applyItem(w: World, p: Player, it: number) {
  if (it === IT_BOOM) p.weapon = Math.max(1, p.weapon === 1 ? 2 : p.weapon)
  else if (it === IT_GOLD) p.weapon = 2
  else if (it === IT_BOARD) p.board = true
  else if (it === IT_FAIRY) {
    p.fairy = FAIRY_FRAMES
    ev(w, EV.fairy, p.x, p.y, p.i)
  } else if (it === IT_BASKET) {
    p.energy = ENERGY_MAX
    p.score += 2000
  }
  p.score += 100
  ev(w, EV.item, p.x + PW / 2, p.y, it * 2 + p.i)
}

function rockStep(w: World, e: Ent) {
  for (const p of touching(w, e, 1)) {
    if (p.board || p.fairy > 0) {
      e.dead = true
      p.score += 50
      ev(w, EV.breakRock, e.x + e.w / 2, e.y + e.h / 2, 0)
      return
    }
    if (p.ground && p.tripCd === 0) {
      p.trip = 26
      p.tripCd = 60
      p.energy -= 10
      p.vx *= 0.4
      ev(w, EV.trip, p.x + PW / 2, p.y + PH, p.i)
    }
  }
}

function crabStep(w: World, e: Ent) {
  e.vx = 0.45 * e.dir
  const fl = fall(w, e)
  // turn at walls and ledges
  const ahead = tileAt(w.stage.tiles, w.stage.cols, e.x + (e.dir > 0 ? e.w + 1 : -1), e.y + e.h + 4)
  if (fl & 1 || (e.ground && !(ahead === TS || ahead === TB || ahead === TO))) e.dir *= -1
  enemyTouch(w, e)
}

function gullStep(w: World, e: Ent) {
  if (e.st === 0) {
    e.x -= 1.15
    e.y = e.oy + Math.sin(e.t / 16) * 10
    const p = nearest(w, e.x)
    if (p && Math.abs(p.x + PW / 2 - (e.x + e.w / 2)) < 26 && p.y > e.y) {
      e.st = 1
      e.vy = 2.6
      e.vx = -1.6
    }
  } else if (e.st === 1) {
    e.x += e.vx
    e.y += e.vy
    e.vy -= 0.05
    if (e.vy < -1.5) e.st = 2
  } else {
    e.x -= 1.6
    e.y -= 1.2
    if (e.y < -20) e.dead = true
  }
  enemyTouch(w, e)
}

function fishStep(w: World, e: Ent) {
  if (e.st === 0) {
    e.c++
    if (e.c > 100) {
      e.st = 1
      e.vy = -6.4
      e.y = H
      e.c = 0
      ev(w, EV.splash, e.x + e.w / 2, H - 14, 0)
    }
    return
  }
  e.vy += 0.2
  e.y += e.vy
  if (e.y > H + 4 && e.vy > 0) {
    e.st = 0
    e.y = H + 4
    ev(w, EV.splash, e.x + e.w / 2, H - 14, 0)
  }
  enemyTouch(w, e)
}

function frogStep(w: World, e: Ent) {
  fall(w, e)
  if (e.ground) {
    e.vx = 0
    e.c++
    const p = nearest(w, e.x)
    if (p) e.dir = p.x < e.x ? -1 : 1
    if (e.c > 70) {
      e.c = 0
      e.vy = -4.4
      e.vx = 1.15 * e.dir
    }
  }
  enemyTouch(w, e)
}

function spiderStep(w: World, e: Ent) {
  const drop = e.a || 80
  const p = nearest(w, e.x)
  if (e.st === 0) {
    e.y = e.oy + Math.sin(e.t / 30) * 3
    if (p && Math.abs(p.x - e.x) < 60) e.st = 1
  } else if (e.st === 1) {
    e.y += 3
    if (e.y >= e.oy + drop) {
      e.st = 2
      e.c = 0
    }
  } else if (e.st === 2) {
    if (++e.c > 40) e.st = 3
  } else {
    e.y -= 0.8
    if (e.y <= e.oy) {
      e.st = 0
      e.t = 0
    }
  }
  enemyTouch(w, e)
}

function rollStep(w: World, e: Ent) {
  const big = e.k === 'boulder'
  e.vx = big ? -1.25 : -1.6
  const fl = fall(w, e)
  if (e.ground && e.t % (big ? 50 : 34) === 0) e.vy = big ? -1.8 : -2.4
  if (fl & 1) e.vy = -3.6
  e.c += e.vx
  enemyTouch(w, e)
}

function batStep(w: World, e: Ent) {
  const p = nearest(w, e.x)
  if (e.st === 0) {
    if (p && e.x - p.x < 110) {
      e.st = 1
      e.c = 0
      e.b = p.y + 6
    }
    return
  }
  e.c++
  e.x -= 1.5
  const k = Math.min(1, e.c / 50)
  e.y = e.oy + (e.b - e.oy) * Math.sin(k * Math.PI * 0.5) - (e.c > 50 ? (e.c - 50) * 1.1 : 0)
  if (e.y < -20) e.dead = true
  enemyTouch(w, e)
}

function flameStep(w: World, e: Ent) {
  fall(w, e, 0.22)
  if (e.ground) {
    e.vy = -3.4
    const p = nearest(w, e.x)
    e.dir = p && p.x < e.x ? -1 : 1
    if (Math.abs(e.x - e.ox) > 36) e.dir = e.x > e.ox ? -1 : 1
  }
  e.vx = 0.5 * e.dir
  enemyTouch(w, e)
}

function geyserStep(w: World, e: Ent) {
  const vine = e.a === 1
  const period = vine ? 9999 : 170
  const t = (e.t + e.c) % period
  if (vine) {
    // boss vine: warn 40 frames, burst 50 frames, gone
    e.st = e.t < 40 ? 1 : e.t < 90 ? 2 : 3
    if (e.st === 3) e.dead = true
  } else e.st = t < 100 ? 0 : t < 130 ? 1 : 2
  if (e.st === 2) {
    const k = vine ? Math.min(1, (e.t - 40) / 6) : Math.min(1, (t - 130) / 6)
    const top = vine ? e.b : H - 16 - 120
    e.h = Math.max(4, (H - 16 - top) * k)
    if (vine) e.h = Math.max(4, 70 * k)
    e.y = (vine ? e.oy + 16 : H - 16) - e.h
    hazardTouch(w, e, false)
  } else {
    e.h = 4
    e.y = (vine ? e.oy + 16 : H - 16) - 4
  }
  if (e.st === 2 && (vine ? e.t === 40 : t === 130)) ev(w, EV.spit, e.x + e.w / 2, e.y + e.h, 2)
}

function platStep(_w: World, e: Ent) {
  const px = e.x
  const py = e.y
  if (e.a === 0) e.x = e.ox + Math.sin(e.t / 70) * e.b
  else if (e.a === 1) e.y = e.oy + Math.sin(e.t / 60) * e.b
  else {
    // crumbling slab: shakes after being stepped on, falls, then grows back
    if (e.st === 1) {
      if (++e.c > 28) {
        e.st = 2
        e.c = 0
      }
    } else if (e.st === 2) {
      e.vy = Math.min(5, e.vy + 0.3)
      e.y += e.vy
      if (e.y > H + 20) {
        e.st = 3
        e.c = 0
      }
    } else if (e.st === 3) {
      if (++e.c > 200) {
        e.st = 0
        e.y = e.oy
        e.vy = 0
        e.c = 0
      }
    }
  }
  e.vx = e.x - px
  if (e.a !== 2) e.vy = e.y - py
}

function boomStep(w: World, e: Ent) {
  const owner = w.players[e.owner]
  const gold = e.a === 1
  if (e.st === 0) {
    e.vx -= e.dir * (gold ? 0.14 : 0.17)
    if (e.vx * e.dir <= 0.4 || e.t > 44) e.st = 1
    e.x += e.vx
    e.y += e.vy
  } else {
    if (owner && owner.st === 'play') {
      const dx = owner.x + PW / 2 - (e.x + e.w / 2)
      const dy = owner.y + 8 - (e.y + e.h / 2)
      const d = Math.hypot(dx, dy) || 1
      const sp = Math.min(6.4, 2 + e.c * 0.2)
      e.c++
      e.vx += ((dx / d) * sp - e.vx) * 0.2
      e.vy += ((dy / d) * sp - e.vy) * 0.2
      if (d < 12) {
        e.dead = true
        ev(w, EV.catch, e.x, e.y, e.owner)
        return
      }
    }
    e.x += e.vx
    e.y += e.vy
  }
  if (e.t > 170 || e.x < w.camX - 40 || e.x > w.camX + W + 40 || e.y > H + 20) {
    e.dead = true
    return
  }
  // hits
  for (const o of w.ents) {
    if (o === e || o.dead || o.st === -1) continue
    if (!overlap(e.x, e.y, e.w, e.h, o.x, o.y, o.w, o.h)) continue
    if (ENEMIES.has(o.k)) {
      if (o.k === 'fish' && o.st === 0) continue
      if (o.flash > 0) continue
      o.hp -= gold ? 2 : 1
      if (o.hp <= 0) ko(w, o, e.vx >= 0 ? 1 : -1, owner ?? null)
      else {
        o.flash = 10
        ev(w, EV.clink, o.x + o.w / 2, o.y + o.h / 2, 1)
      }
      if (!gold) e.st = 1
      w.hitstop = 2
      if (!gold) return
    } else if (o.k === 'rock') {
      o.dead = true
      if (owner) owner.score += 50
      ev(w, EV.breakRock, o.x + o.w / 2, o.y + o.h / 2, 0)
      if (!gold) e.st = 1
    } else if (o.k === 'fire' && e.st === 0) {
      if (gold) {
        o.dead = true
        if (owner) owner.score += 200
        ev(w, EV.extinguish, o.x + o.w / 2, o.y + o.h / 2, 0)
      } else {
        e.st = 1
        ev(w, EV.clink, e.x, e.y, 0)
      }
    } else if (o.k === 'clam' && o.st === 0) {
      openClam(w, o)
      e.st = 1
    } else if (o.k === 'shot' && (o.a === 0 || o.a === 1)) {
      o.dead = true
      ev(w, EV.clink, o.x, o.y, 2)
    } else if (o.k === 'boss') {
      if (bossHit(w, o, e, owner ?? null) && !gold) e.st = 1
    }
  }
}

function shotStep(w: World, e: Ent) {
  const grav = [0.025, 0.14, 0.16, 0.3][e.a] ?? 0.1
  e.vy += grav
  if (e.a === 0) {
    e.vx *= 0.995
    e.vy = Math.min(e.vy, 0.6)
  }
  e.x += e.vx
  e.y += e.vy
  if (e.t > (e.a === 0 ? 300 : 200) || e.y > H + 10 || e.x < w.camX - 30 || e.x > w.camX + W + 30) e.dead = true
  if (e.a !== 0 && tileAt(w.stage.tiles, w.stage.cols, e.x + e.w / 2, e.y + e.h) === TS) {
    e.dead = true
    ev(w, EV.clink, e.x + e.w / 2, e.y + e.h, 3)
    return
  }
  for (const p of touching(w, e, 0)) {
    if (p.fairy > 0) {
      e.dead = true
      return
    }
    hurtPlayer(w, p)
    if (e.a === 0) e.dead = true
  }
}

function sunStep(w: World, e: Ent) {
  fall(w, e, 0.12)
  for (const p of touching(w, e, 0)) {
    e.dead = true
    p.score += 5000
    ev(w, EV.sun, e.x, e.y, p.i)
    w.phase = 'clear'
    w.phaseT = 0
    for (const q of w.players) if (q.on && (q.st === 'ghost' || q.st === 'dying')) revive(w, q, p.x + 14, p.y)
    for (const q of w.players) if (q.on) q.st = 'win'
    return
  }
}

function stepClear(w: World) {
  w.phaseT++
  // fruit energy turns into points during the tally
  for (const p of w.players) {
    if (!p.on) continue
    p.vy = Math.min(MAX_FALL, p.vy + GRAV)
    moveBox({ tiles: w.stage.tiles, cols: w.stage.cols }, p, PW, PH)
    if (w.phaseT > 60 && p.energy > 0) {
      const d = Math.min(p.energy, 1)
      p.energy -= d
      p.score += 10
    }
  }
  if (w.phaseT === 300) {
    if (w.stageIndex + 1 >= STAGE_COUNT) {
      w.phase = 'ending'
      w.phaseT = 0
    } else {
      loadStage(w, w.stageIndex + 1, 0)
      ev(w, EV.clear, 0, 0, w.stageIndex)
    }
  }
}

// ───────────────────────────── bosses ─────────────────────────────

const BOSS_HP = [16, 20, 26]

function startBoss(w: World) {
  const type = w.stage.boss
  const floor = H - 3 * T
  const hp = Math.round(BOSS_HP[type] * (w.twoP ? 1.4 : 1))
  w.bossMax = hp
  const size: [number, number][] = [
    [54, 38],
    [36, 36],
    [62, 38]
  ]
  const [bw, bh] = size[type]
  const x = w.camX + W - 90
  const e = spawnEnt(w, 'boss', x, type === 1 ? 60 : -60, { a: type, hp, w: bw, h: bh, dir: -1 })
  e.x = x
  e.y = type === 1 ? floor - 92 : -bh - 20
  e.ox = x
  e.oy = type === 1 ? floor - 92 : floor - bh
  ev(w, EV.bossRoar, x, 80, type)
}

function bossHit(w: World, b: Ent, boom: Ent, by: Player | null): boolean {
  if (b.st === 0 || b.st === 9 || b.flash > 0) return false
  if (b.a === 2 && b.st === 5) {
    // shell spin: boomerangs bounce off
    ev(w, EV.clink, boom.x, boom.y, 0)
    return true
  }
  const dmg = boom.a === 1 ? 2 : 1
  b.hp -= dmg
  b.flash = 8
  w.hitstop = 3
  if (by) by.score += 100
  ev(w, EV.bossHit, boom.x + boom.w / 2, boom.y + boom.h / 2, b.a)
  if (b.hp <= 0) {
    b.st = 9
    b.t = 0
    w.shake = 30
    ev(w, EV.bossDie, b.x + b.w / 2, b.y + b.h / 2, b.a)
  }
  return true
}

function bossShoot(w: World, x: number, y: number, vx: number, vy: number, type: number) {
  spawnEnt(w, 'shot', x, y, { a: type, vx, vy, w: type === 0 ? 10 : 8, h: type === 0 ? 10 : 8 })
}

function aim(w: World, x: number): Player | null {
  const ps = alivePlayers(w)
  if (!ps.length) return null
  return ps[(w.frame >> 7) % ps.length] ?? nearest(w, x)
}

function bossStep(w: World, b: Ent) {
  const floor = H - 3 * T
  const left = w.camX + 8
  const right = w.camX + W - 8
  const angry = b.hp < w.bossMax / 2
  if (b.st === 9) {
    // death: shake, sparks, then the piece of the sun fruit drops
    b.t++
    if (b.t % 12 === 0) ev(w, EV.bossDie, b.x + ((b.t * 37) % b.w), b.y + ((b.t * 23) % b.h), -1)
    if (b.t > 120) {
      b.dead = true
      spawnEnt(w, 'sun', w.camX + W / 2, 30, { vy: -1 })
    }
    return
  }
  b.t++
  if (b.a === 0) crabKing(w, b, floor, left, right, angry)
  else if (b.a === 1) bloom(w, b, floor, angry)
  else tortoise(w, b, floor, left, right, angry)
  // contact damage
  if (b.st !== 0) {
    for (const p of touching(w, b, 4)) {
      if (p.fairy > 0) continue
      hurtPlayer(w, p)
    }
  }
}

function crabKing(w: World, b: Ent, floor: number, left: number, right: number, angry: boolean) {
  const target = aim(w, b.x)
  switch (b.st) {
    case 0: // drop in
      b.vy = Math.min(7, b.vy + 0.3)
      b.y += b.vy
      if (b.y >= b.oy) {
        b.y = b.oy
        b.st = 1
        b.t = 0
        w.shake = 20
        ev(w, EV.quake, b.x + b.w / 2, floor, 0)
      }
      return
    case 1: {
      // stroll
      if (target) b.dir = target.x < b.x + b.w / 2 ? -1 : 1
      b.x += b.dir * (angry ? 0.9 : 0.55) * (b.t % 60 < 40 ? 1 : -0.4)
      b.x = Math.max(left, Math.min(right - b.w, b.x))
      if (b.t > (angry ? 70 : 110)) {
        b.c = (b.c + 1) % 3
        b.st = [2, 3, 4][b.c]
        b.t = 0
      }
      return
    }
    case 2: // bubbles
      if (b.t % (angry ? 16 : 22) === 0 && b.t <= 66) {
        const d = target && target.x < b.x ? -1 : 1
        bossShoot(w, b.x + b.w / 2 + d * 20, b.y + 14, d * (0.9 + Math.random() * 1.4), -1.2 - Math.random(), 0)
        ev(w, EV.spit, b.x + b.w / 2, b.y + 14, 0)
      }
      if (b.t > 90) {
        b.st = 1
        b.t = 0
      }
      return
    case 3: // claw up, then charge
      if (b.t === 1) b.dir = target && target.x < b.x ? -1 : 1
      if (b.t > 34) {
        b.x += b.dir * (angry ? 4.2 : 3.4)
        if (b.x <= left || b.x + b.w >= right) {
          b.x = Math.max(left, Math.min(right - b.w, b.x))
          w.shake = 14
          ev(w, EV.quake, b.x + (b.dir > 0 ? b.w : 0), floor - 10, 0)
          // rocks rattle loose from the cliff
          for (let i = 0; i < (angry ? 4 : 2); i++) bossShoot(w, left + 30 + ((w.frame * 71 + i * 113) % (W - 60)), -10, 0, 1, 3)
          b.st = 5
          b.t = 0
        }
      }
      return
    case 4: // jump at the hero
      if (b.t === 1) {
        b.vy = -6.6
        b.vx = target ? Math.max(-2.6, Math.min(2.6, (target.x - b.x - b.w / 2) / 50)) : 0
      }
      b.vy += 0.24
      b.x = Math.max(left, Math.min(right - b.w, b.x + b.vx))
      b.y += b.vy
      if (b.y >= b.oy && b.vy > 0) {
        b.y = b.oy
        w.shake = 18
        ev(w, EV.quake, b.x + b.w / 2, floor, 0)
        b.st = 1
        b.t = 0
      }
      return
    case 5: // dazed after hitting the wall
      if (b.t > 50) {
        b.st = 1
        b.t = 0
      }
      return
  }
}

function bloom(w: World, b: Ent, floor: number, angry: boolean) {
  const target = aim(w, b.x)
  // the head bobs on its stem; b.ox/oy is the rest position
  if (b.st === 0) {
    b.y = b.oy - 80 + Math.min(80, b.t * 2)
    if (b.t > 50) {
      b.st = 1
      b.t = 0
      w.shake = 12
      ev(w, EV.quake, b.x, floor, 1)
    }
    return
  }
  if (b.st !== 4) {
    b.x = b.ox + Math.sin(b.t / 40) * 6
    b.y = b.oy + Math.sin(b.t / 25) * 10
  }
  switch (b.st) {
    case 1:
      if (b.t > (angry ? 60 : 90)) {
        b.c = (b.c + 1) % 3
        b.st = [2, 3, 4][b.c]
        b.t = 0
      }
      return
    case 2: // seed spread
      if (b.t === 24 || (angry && b.t === 44)) {
        const n = angry ? 5 : 4
        for (let i = 0; i < n; i++) {
          const tx = target ? target.x - b.x : -150
          bossShoot(w, b.x + 4, b.y + 22, tx / 60 - 0.6 + i * 0.5 - n * 0.2, -3.4 - i * 0.2, 1)
        }
        ev(w, EV.spit, b.x, b.y + 20, 1)
      }
      if (b.t > 70) {
        b.st = 1
        b.t = 0
      }
      return
    case 3: // vines burst under the heroes
      if (b.t === 10 || (angry && b.t === 40)) {
        for (const p of alivePlayers(w)) spawnEnt(w, 'geyser', p.x + PW / 2, floor - 8, { a: 1, oy: floor - 16, b: floor - 70, w: 14, h: 4 })
        ev(w, EV.quake, b.x, floor, 1)
      }
      if (b.t > 100) {
        b.st = 1
        b.t = 0
      }
      return
    case 4: {
      // lunge: the head swings low across the arena and back
      const k = b.t / 70
      const reach = angry ? 250 : 200
      b.x = b.ox - Math.sin(Math.min(1, k) * Math.PI) * reach
      b.y = b.oy + Math.sin(Math.min(1, k) * Math.PI) * 74
      if (b.t > 70) {
        b.st = 1
        b.t = 0
      }
      return
    }
  }
}

function tortoise(w: World, b: Ent, floor: number, left: number, right: number, angry: boolean) {
  const target = aim(w, b.x)
  switch (b.st) {
    case 0:
      b.vy = Math.min(7, b.vy + 0.3)
      b.y += b.vy
      if (b.y >= b.oy) {
        b.y = b.oy
        b.st = 1
        b.t = 0
        w.shake = 24
        ev(w, EV.quake, b.x + b.w / 2, floor, 2)
      }
      return
    case 1:
      if (target) b.dir = target.x < b.x + b.w / 2 ? -1 : 1
      b.x = Math.max(left, Math.min(right - b.w, b.x + b.dir * (angry ? 0.7 : 0.45)))
      if (b.t > (angry ? 60 : 90)) {
        b.c = (b.c + 1) % 3
        b.st = [2, 3, 5][b.c]
        b.t = 0
      }
      return
    case 2: // fireballs
      if (b.t === 20 || b.t === 40 || (angry && b.t === 60)) {
        const d = b.dir
        bossShoot(w, b.x + b.w / 2 + d * 30, b.y + 6, d * (1.4 + (b.t / 20) * 0.6), -4, 2)
        ev(w, EV.spit, b.x + b.w / 2 + d * 30, b.y + 6, 2)
      }
      if (b.t > 80) {
        b.st = 1
        b.t = 0
      }
      return
    case 3: // big jump, the cave shakes and rocks fall
      if (b.t === 1) {
        b.vy = -7.2
        b.vx = target ? Math.max(-2.2, Math.min(2.2, (target.x - b.x - b.w / 2) / 55)) : 0
      }
      b.vy += 0.26
      b.x = Math.max(left, Math.min(right - b.w, b.x + b.vx))
      b.y += b.vy
      if (b.y >= b.oy && b.vy > 0) {
        b.y = b.oy
        w.shake = 26
        ev(w, EV.quake, b.x + b.w / 2, floor, 2)
        for (let i = 0; i < (angry ? 5 : 3); i++) bossShoot(w, left + 20 + ((w.frame * 53 + i * 89) % (W - 40)), -12 - i * 20, 0, 1, 3)
        b.st = 1
        b.t = 0
      }
      return
    case 5: // shell spin: bounces off the walls, boomerangs clink off
      if (b.t < 30) return
      if (b.t === 30) b.vx = (target && target.x < b.x ? -1 : 1) * (angry ? 4.2 : 3.4)
      b.x += b.vx
      if (b.x <= left || b.x + b.w >= right) {
        b.x = Math.max(left, Math.min(right - b.w, b.x))
        b.vx = -b.vx
        w.shake = 8
        ev(w, EV.quake, b.x, floor, 2)
        b.c += 10
      }
      if (b.t > 30 + (angry ? 220 : 170)) {
        b.st = 6
        b.t = 0
        b.c = b.c % 10
      }
      return
    case 6: // dizzy
      if (b.t > 70) {
        b.st = 1
        b.t = 0
      }
      return
  }
}

export function inBoss(w: World): Ent | undefined {
  return w.ents.find((e) => e.k === 'boss')
}

export { IN_L, IN_R, IN_J, IN_F, IN_D }
