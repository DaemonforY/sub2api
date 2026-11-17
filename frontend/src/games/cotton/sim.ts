// The rules of 棉花糖大作战, with no DOM in sight: one call to step() is one 60 Hz frame.
// The same code runs the solo and same-keyboard games, and the host of an online game; the guest
// only moves its own kid with movePlayer() and draws what the host sends.
//
// Shots wrap enemies in candy floss a layer at a time; four layers make a ball. Kick a ball and
// it rolls, bounces off walls and flattens everything in its way, kicking other balls along the
// same combo. Each kill in a combo doubles the score and drops a snack; potions make a kid faster,
// shoot further or wrap thicker. A dead kid floats up as a candy cloud the partner can touch to
// bring back.

import { FLOORS, FLOOR_COUNT, type BossKind, type EnemyKind } from './levels'

export const W = 320
export const H = 240
export const T = 8
export const COLS = 40
export const ROWS = 30
export const GROUND_Y = 224
export const SNOW_FULL = 4
export const FPS = 60

const GRAV = 0.2
const MAX_FALL = 4.5
const RUN = 1.3
const JUMP_V = -4.55
const BALL_SPEED = 3.7
const BALL_TIME = 420
const HURRY_WARN = 40 * FPS
const HURRY_AT = 45 * FPS

export interface Buttons {
  l: boolean
  r: boolean
  u: boolean
  d: boolean
  jump: boolean
  fire: boolean
}

export const NO_BUTTONS: Buttons = { l: false, r: false, u: false, d: false, jump: false, fire: false }

export const ALIVE = 0
export const GHOST = 1
export const OUT = 2

export interface Player {
  idx: 0 | 1
  /** taking part in this game */
  on: boolean
  state: number
  x: number
  y: number
  w: number
  h: number
  vx: number
  vy: number
  face: 1 | -1
  ground: boolean
  coyote: number
  jbuf: number
  cd: number
  auto: number
  shootT: number
  kickT: number
  inv: number
  ghostT: number
  respawnT: number
  walk: number
  drop: number
  bump: number
  push: number
  jumpHeld: boolean
  fireHeld: boolean
  lives: number
  score: number
  floorScore: number
  kills: number
  bestChain: number
  speed: boolean
  range: boolean
  power: boolean
  /** bumped every time the host moves this kid by force (respawn, revive, new floor) */
  epoch: number
  /** an online guest moves this kid itself; the host only checks what happens to it */
  remote: boolean
}

export type Kind = EnemyKind | BossKind | 'slipper'

export interface Enemy {
  id: number
  kind: Kind
  boss: boolean
  x: number
  y: number
  w: number
  h: number
  bw: number
  bh: number
  vx: number
  vy: number
  face: 1 | -1
  ground: boolean
  snow: number
  hold: number
  ballT: number
  rolling: boolean
  combo: number
  bounces: number
  rollT: number
  dying: number
  angry: boolean
  t: number
  st: number
  cd: number
  hp: number
  maxHp: number
  flash: number
  spawnT: number
  edge: number
}

export interface Shot {
  id: number
  o: number
  x: number
  y: number
  vx: number
  vy: number
  t: number
  life: number
  big: boolean
}

export type ProjKind = 'chalk' | 'wave' | 'plane' | 'page'

export interface Proj {
  id: number
  kind: ProjKind
  x: number
  y: number
  w: number
  h: number
  vx: number
  vy: number
  t: number
  life: number
}

/** 0–5 snacks, 6 red 力量 potion, 7 blue 速度, 8 yellow 射程, 9 糖葫芦 (extra life), 10 红包 */
export const SNACK_POINTS = [100, 200, 500, 1000, 2000, 5000]
export const ITEM_RED = 6
export const ITEM_BLUE = 7
export const ITEM_YELLOW = 8
export const ITEM_LIFE = 9
export const ITEM_PACKET = 10

export interface Item {
  id: number
  kind: number
  x: number
  y: number
  vx: number
  vy: number
  t: number
}

export type EvKind =
  | 'jump'
  | 'shot'
  | 'hit'
  | 'ball'
  | 'nudge'
  | 'kick'
  | 'bounce'
  | 'kill'
  | 'burst'
  | 'thaw'
  | 'pickup'
  | 'die'
  | 'revive'
  | 'respawn'
  | 'bump'
  | 'land'
  | 'hurry'
  | 'slipper'
  | 'throw'
  | 'quake'
  | 'ring'
  | 'bosshit'
  | 'bossdie'
  | 'sweep'
  | 'clear'
  | 'floor'
  | 'over'
  | 'won'
  | 'poof'
  | 'spawn'
  | 'push'

export interface Ev {
  k: EvKind
  x: number
  y: number
  /** points, combo length, item kind… depends on k */
  n?: number
  /** player index or extra detail */
  p?: number
}

interface Combo {
  n: number
  owner: number
  /** enemies standing when the first ball was kicked: clearing them all is 一球清场 */
  alive: number
}

export type Phase = 'intro' | 'play' | 'clear' | 'result' | 'over' | 'won'

export interface FloorResult {
  seconds: number
  bonus: number
  floorScore: [number, number]
  bestChain: number
  sweep: boolean
}

export interface GameState {
  frame: number
  rng: number
  /** how many kids play: 1 or 2 */
  count: number
  floor: number
  phase: Phase
  phaseT: number
  floorT: number
  tiles: Uint8Array
  players: [Player, Player]
  enemies: Enemy[]
  shots: Shot[]
  projs: Proj[]
  items: Item[]
  combos: Record<number, Combo>
  nextId: number
  hurry: number
  events: Ev[]
  result: FloorResult | null
  floorBest: number
  sweep: boolean
  /** highest floor reached (1-based) this run */
  reached: number
  /** frames played this run */
  played: number
}

// ---------------------------------------------------------------- set-up

function rand(s: GameState): number {
  // mulberry32
  let t = (s.rng = (s.rng + 0x6d2b79f5) | 0)
  t = Math.imul(t ^ (t >>> 15), t | 1)
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296
}

function newPlayer(idx: 0 | 1): Player {
  return {
    idx,
    on: false,
    state: OUT,
    x: 0,
    y: 0,
    w: 10,
    h: 16,
    vx: 0,
    vy: 0,
    face: idx === 0 ? 1 : -1,
    ground: false,
    coyote: 0,
    jbuf: 0,
    cd: 0,
    auto: 0,
    shootT: 0,
    kickT: 0,
    inv: 0,
    ghostT: 0,
    respawnT: 0,
    walk: 0,
    drop: 0,
    bump: 0,
    push: 0,
    jumpHeld: false,
    fireHeld: false,
    lives: 3,
    score: 0,
    floorScore: 0,
    kills: 0,
    bestChain: 0,
    speed: false,
    range: false,
    power: false,
    epoch: 0,
    remote: false
  }
}

export function buildTiles(floor: number): Uint8Array {
  const tiles = new Uint8Array(COLS * ROWS)
  for (let r = 0; r < ROWS; r++) {
    for (let c = 0; c < COLS; c++) {
      if (c === 0 || c === COLS - 1 || r >= 28 || r <= 1) tiles[r * COLS + c] = 1
    }
  }
  const def = FLOORS[Math.max(0, Math.min(FLOOR_COUNT - 1, floor))]
  for (const [r, a, b] of def.plats) for (let c = a; c <= b; c++) tiles[r * COLS + c] = 2
  for (const [r, a, b] of def.solids ?? []) for (let c = a; c <= b; c++) tiles[r * COLS + c] = 1
  return tiles
}

export function createGame(count: number, seed = (Math.random() * 2 ** 31) | 0, floor = 0): GameState {
  const s: GameState = {
    frame: 0,
    rng: seed,
    count,
    floor,
    phase: 'intro',
    phaseT: 0,
    floorT: 0,
    tiles: buildTiles(floor),
    players: [newPlayer(0), newPlayer(1)],
    enemies: [],
    shots: [],
    projs: [],
    items: [],
    combos: {},
    nextId: 1,
    hurry: 0,
    events: [],
    result: null,
    floorBest: 0,
    sweep: false,
    reached: floor + 1,
    played: 0
  }
  s.players[0].on = true
  s.players[1].on = count > 1
  startFloor(s, floor)
  return s
}

/** a second kid joins mid-game (same keyboard press-to-join, or an online friend) */
export function joinPlayer(s: GameState, idx: 0 | 1): void {
  const p = s.players[idx]
  if (p.on) return
  p.on = true
  s.count = 2
  p.lives = 3
  spawnAt(s, p)
  s.events.push({ k: 'respawn', x: p.x + 5, y: p.y + 8, p: idx })
}

export function leavePlayer(s: GameState, idx: 0 | 1): void {
  const p = s.players[idx]
  p.on = false
  p.state = OUT
  s.count = s.players.filter((q) => q.on).length
}

function spawnAt(_s: GameState, p: Player): void {
  p.state = ALIVE
  p.x = p.idx === 0 ? 9 * T : 30 * T - 2
  p.y = GROUND_Y - p.h
  p.vx = 0
  p.vy = 0
  p.face = p.idx === 0 ? 1 : -1
  p.inv = 120
  p.ghostT = 0
  p.respawnT = 0
  p.bump = 0
  p.epoch++
}

function makeEnemy(s: GameState, kind: Kind, x: number, y: number): Enemy {
  const boss = kind === 'bigclock' || kind === 'bookpile' || kind === 'calendar'
  const size: Record<string, [number, number]> = {
    book: [12, 14],
    clock: [12, 13],
    mosquito: [12, 10],
    chalk: [8, 14],
    coil: [13, 13],
    slipper: [14, 10],
    bigclock: [36, 38],
    bookpile: [44, 54],
    calendar: [40, 46]
  }
  const [w, h] = size[kind]
  const hp = kind === 'bigclock' ? 50 : kind === 'bookpile' ? 70 : kind === 'calendar' ? 80 : 0
  return {
    id: s.nextId++,
    kind,
    boss,
    x: x - w / 2,
    y: y - h,
    w,
    h,
    bw: w,
    bh: h,
    vx: 0,
    vy: 0,
    face: x < W / 2 ? 1 : -1,
    ground: false,
    snow: 0,
    hold: 0,
    ballT: 0,
    rolling: false,
    combo: 0,
    bounces: 0,
    rollT: 0,
    dying: 0,
    angry: false,
    t: Math.floor(rand(s) * 120),
    st: 0,
    cd: 60 + Math.floor(rand(s) * 120),
    hp,
    maxHp: hp,
    flash: 0,
    spawnT: 0,
    edge: 0
  }
}

export function startFloor(s: GameState, floor: number): void {
  s.floor = floor
  s.reached = Math.max(s.reached, floor + 1)
  s.tiles = buildTiles(floor)
  s.phase = 'intro'
  s.phaseT = 0
  s.floorT = 0
  s.enemies = []
  s.shots = []
  s.projs = []
  s.items = []
  s.combos = {}
  s.hurry = 0
  s.result = null
  s.floorBest = 0
  s.sweep = false
  const def = FLOORS[floor]
  def.enemies.forEach(([kind, col, row], i) => {
    const e = makeEnemy(s, kind, col * T + T / 2, row * T)
    e.spawnT = 20 + i * 9
    s.enemies.push(e)
  })
  if (def.boss) {
    const x = def.boss === 'bookpile' ? 30 * T : W / 2
    const y = def.boss === 'calendar' ? 90 : GROUND_Y
    const b = makeEnemy(s, def.boss, x, y)
    if (def.boss !== 'calendar') b.y = 17
    b.spawnT = 40
    b.face = -1
    s.enemies.push(b)
  }
  for (const p of s.players) {
    p.floorScore = 0
    if (!p.on) continue
    // a kid who ran out of lives in a duo comes back for the next floor with none to spare
    spawnAt(s, p)
    p.inv = 90
  }
  s.events.push({ k: 'floor', x: W / 2, y: H / 2, n: floor })
}

// ---------------------------------------------------------------- collisions

export function tileAt(s: GameState, c: number, r: number): number {
  if (c < 0 || c >= COLS) return 1
  if (r < 0 || r >= ROWS) return r < 0 ? 0 : 1
  return s.tiles[r * COLS + c]
}

interface Body {
  x: number
  y: number
  w: number
  h: number
  vx: number
  vy: number
  ground: boolean
}

/** returns 1 if a wall stopped the move */
function moveX(s: GameState, b: Body): number {
  const nx = b.x + b.vx
  const r0 = Math.floor(b.y / T)
  const r1 = Math.floor((b.y + b.h - 1) / T)
  if (b.vx > 0) {
    const c = Math.floor((nx + b.w - 1) / T)
    for (let r = r0; r <= r1; r++) {
      if (tileAt(s, c, r) === 1) {
        b.x = c * T - b.w
        return 1
      }
    }
  } else if (b.vx < 0) {
    const c = Math.floor(nx / T)
    for (let r = r0; r <= r1; r++) {
      if (tileAt(s, c, r) === 1) {
        b.x = (c + 1) * T
        return 1
      }
    }
  }
  b.x = nx
  return 0
}

/** returns 1 on landing, -1 on a bonk */
function moveY(s: GameState, b: Body, oneway: boolean): number {
  const ny = b.y + b.vy
  const c0 = Math.floor(b.x / T)
  const c1 = Math.floor((b.x + b.w - 1) / T)
  if (b.vy > 0) {
    const bottom = b.y + b.h
    const r = Math.floor((ny + b.h) / T)
    for (let c = c0; c <= c1; c++) {
      const t = tileAt(s, c, r)
      if (t === 1 || (t === 2 && oneway && bottom <= r * T + 0.01)) {
        b.y = r * T - b.h
        b.vy = 0
        b.ground = true
        return 1
      }
    }
  } else if (b.vy < 0) {
    const r = Math.floor(ny / T)
    for (let c = c0; c <= c1; c++) {
      if (tileAt(s, c, r) === 1) {
        b.y = (r + 1) * T
        b.vy = 0
        b.ground = false
        return -1
      }
    }
  }
  b.y = ny
  b.ground = false
  return 0
}

function overlap(a: { x: number; y: number; w: number; h: number }, b: { x: number; y: number; w: number; h: number }, shrink = 0): boolean {
  return a.x + shrink < b.x + b.w && b.x + shrink < a.x + a.w && a.y + shrink < b.y + b.h && b.y + shrink < a.y + a.h
}

function isBall(e: Enemy): boolean {
  return e.snow >= SNOW_FULL && !e.dying && !e.boss
}

function onOneway(s: GameState, b: Body): boolean {
  const r = Math.floor((b.y + b.h) / T)
  const c0 = Math.floor(b.x / T)
  const c1 = Math.floor((b.x + b.w - 1) / T)
  let any = false
  for (let c = c0; c <= c1; c++) {
    const t = tileAt(s, c, r)
    if (t === 1) return false
    if (t === 2) any = true
  }
  return any
}

// ---------------------------------------------------------------- players

/** movement only: shared by the simulation and an online guest predicting its own kid */
export function movePlayer(s: GameState, p: Player, b: Buttons): void {
  const dir = (b.r ? 1 : 0) - (b.l ? 1 : 0)
  const max = RUN * (p.speed ? 1.38 : 1) * (p.push > 0 ? 0.45 : 1)
  if (p.bump > 0) p.bump--
  else {
    const target = dir * max
    const acc = p.ground ? (dir === 0 ? 0.32 : 0.26) : 0.17
    if (p.vx < target) p.vx = Math.min(target, p.vx + acc)
    else if (p.vx > target) p.vx = Math.max(target, p.vx - acc)
  }
  if (dir !== 0 && p.kickT <= 0) p.face = dir as 1 | -1

  const pressed = b.jump && !p.jumpHeld
  p.jumpHeld = b.jump
  if (pressed) p.jbuf = 7
  else if (p.jbuf > 0) p.jbuf--
  if (p.ground) p.coyote = 6
  else if (p.coyote > 0) p.coyote--
  if (p.jbuf > 0 && p.coyote > 0) {
    if (b.d && p.ground && onOneway(s, p)) {
      p.drop = 10
      p.y += 1
    } else {
      p.vy = JUMP_V
      s.events.push({ k: 'jump', x: p.x + p.w / 2, y: p.y + p.h, p: p.idx })
    }
    p.jbuf = 0
    p.coyote = 0
    p.ground = false
  }
  if (!b.jump && p.vy < -1.7) p.vy = -1.7
  p.vy = Math.min(MAX_FALL, p.vy + GRAV)
  if (p.drop > 0) p.drop--

  const wasGround = p.ground
  const fall = p.vy
  moveX(s, p)
  if (p.x < T) p.x = T
  if (p.x > W - T - p.w) p.x = W - T - p.w
  const prevBottom = p.y + p.h
  moveY(s, p, p.drop === 0)
  // stand on candy balls
  if (!p.ground && p.vy >= 0) {
    for (const e of s.enemies) {
      if (!isBall(e) || e.rolling) continue
      if (p.x + p.w > e.x + 2 && p.x < e.x + e.w - 2 && prevBottom <= e.y + 3 && p.y + p.h >= e.y) {
        p.y = e.y - p.h
        p.vy = 0
        p.ground = true
        break
      }
    }
  }
  if (p.ground && !wasGround && fall > 2.5) s.events.push({ k: 'land', x: p.x + p.w / 2, y: p.y + p.h, p: p.idx })
  if (p.ground && Math.abs(p.vx) > 0.2) p.walk += Math.abs(p.vx)
  else if (p.ground) p.walk = 0
  if (p.shootT > 0) p.shootT--
  if (p.kickT > 0) p.kickT--
  if (p.inv > 0) p.inv--
}

function ballInReach(s: GameState, p: Player): Enemy | null {
  const reach = { x: p.face > 0 ? p.x + 2 : p.x - 8, y: p.y + 2, w: p.w + 6, h: p.h - 2 }
  let best: Enemy | null = null
  let bestD = 1e9
  for (const e of s.enemies) {
    if (!isBall(e) || e.rolling || e.spawnT > 0) continue
    if (!overlap(reach, e)) continue
    const d = Math.abs(e.x + e.w / 2 - (p.x + p.w / 2))
    if (d < bestD) {
      bestD = d
      best = e
    }
  }
  return best
}

/** pressing fire: kick a ball in reach, otherwise throw candy floss */
export function fire(s: GameState, p: Player): void {
  if (p.cd > 0 || p.state !== ALIVE) return
  p.cd = 8
  const ball = ballInReach(s, p)
  if (ball) {
    kick(s, p, ball)
    return
  }
  const mine = s.shots.filter((x) => x.o === p.idx).length
  if (mine >= 3) return
  p.shootT = 10
  const life = p.range ? 30 : 17
  s.shots.push({
    id: s.nextId++,
    o: p.idx,
    x: p.face > 0 ? p.x + p.w - 1 : p.x - 7,
    y: p.y + 4,
    vx: p.face * (p.range ? 4.2 : 3.6),
    vy: 0,
    t: 0,
    life,
    big: p.power
  })
  s.events.push({ k: 'shot', x: p.x + p.w / 2 + p.face * 8, y: p.y + 7, p: p.idx })
}

function kick(s: GameState, p: Player, e: Enemy): void {
  p.kickT = 14
  const id = s.nextId++
  const alive = s.enemies.filter((o) => !o.dying && o.kind !== 'slipper' && !o.boss).length
  s.combos[id] = { n: 0, owner: p.idx, alive }
  e.rolling = true
  e.combo = id
  e.vx = p.face * BALL_SPEED * (p.power ? 1.12 : 1)
  e.vy = -1.2
  e.bounces = 0
  e.rollT = 0
  // start clear of the kicker so the ball does not bump them
  if (p.face > 0) e.x = Math.max(e.x, p.x + p.w)
  else e.x = Math.min(e.x, p.x - e.w)
  s.events.push({ k: 'kick', x: e.x + e.w / 2, y: e.y + e.h / 2, p: p.idx })
}

function updatePlayer(s: GameState, p: Player, b: Buttons): void {
  if (!p.on) return
  if (p.state === GHOST) {
    updateGhost(s, p)
    return
  }
  if (p.state !== ALIVE) return
  if (!p.remote) movePlayer(s, p, b)
  else if (p.inv > 0) p.inv--
  if (p.cd > 0) p.cd--
  if (!p.remote) {
    const pressed = b.fire && !p.fireHeld
    p.fireHeld = b.fire
    if (pressed) {
      fire(s, p)
      p.auto = 16
    } else if (b.fire && --p.auto <= 0) {
      fire(s, p)
      p.auto = 14
    }
  }
  // push candy balls along
  p.push = Math.max(0, p.push - 1)
  for (const e of s.enemies) {
    if (!isBall(e) || e.rolling || e.spawnT > 0) continue
    if (!overlap(p, e, 1)) continue
    if (p.y + p.h <= e.y + 4) continue
    const pc = p.x + p.w / 2
    const ec = e.x + e.w / 2
    const side = pc < ec ? 1 : -1
    if (Math.sign(p.vx) === side && Math.abs(p.vx) > 0.1) {
      const before = e.x
      e.vx = side * 0.6
      moveX(s, e)
      e.vx = 0
      if (e.x !== before) {
        p.push = 4
        if (s.frame % 12 === 0) s.events.push({ k: 'push', x: ec, y: e.y + e.h, p: p.idx })
      }
    }
    // stay outside the ball
    if (side > 0) p.x = Math.min(p.x, e.x - p.w + 1)
    else p.x = Math.max(p.x, e.x + e.w - 1)
  }
}

function die(s: GameState, p: Player): void {
  p.state = GHOST
  p.vx = 0
  p.vy = 0
  p.speed = p.range = p.power = false
  const duo = s.players.filter((q) => q.on).length > 1
  p.ghostT = duo ? 360 : 0
  p.respawnT = duo ? 0 : 110
  s.events.push({ k: 'die', x: p.x + p.w / 2, y: p.y + p.h / 2, p: p.idx })
}

function updateGhost(s: GameState, p: Player): void {
  // drift up as a little candy cloud
  if (p.y > 36) p.y -= 0.22
  p.x += Math.sin((s.frame + p.idx * 40) / 30) * 0.25
  p.x = Math.max(T, Math.min(W - T - p.w, p.x))
  const mate = s.players[1 - p.idx]
  if (mate.on && mate.state === ALIVE && Math.hypot(mate.x - p.x, mate.y - p.y) < 18) {
    p.state = ALIVE
    p.inv = 150
    p.vy = -2
    p.epoch++
    p.ghostT = 0
    mate.score += 1000
    mate.floorScore += 1000
    s.events.push({ k: 'revive', x: p.x + p.w / 2, y: p.y + p.h / 2, p: p.idx, n: 1000 })
    return
  }
  if (p.respawnT > 0) {
    // solo: a short pause, then the next life
    if (--p.respawnT === 0) {
      if (p.lives > 0) {
        p.lives--
        spawnAt(s, p)
        s.events.push({ k: 'respawn', x: p.x + p.w / 2, y: p.y + p.h / 2, p: p.idx })
      } else p.respawnT = -1
    }
    return
  }
  if (p.ghostT > 0 && --p.ghostT === 0) {
    if (p.lives > 0) {
      p.lives--
      spawnAt(s, p)
      s.events.push({ k: 'respawn', x: p.x + p.w / 2, y: p.y + p.h / 2, p: p.idx })
    } else p.ghostT = -1
  }
}

// ---------------------------------------------------------------- enemies

function nearestPlayer(s: GameState, e: { x: number; y: number; w: number; h: number }): Player | null {
  let best: Player | null = null
  let bd = 1e9
  for (const p of s.players) {
    if (!p.on || p.state !== ALIVE) continue
    const d = Math.abs(p.x - e.x) + Math.abs(p.y - e.y) * 1.5
    if (d < bd) {
      bd = d
      best = p
    }
  }
  return best
}

function becomeBall(s: GameState, e: Enemy): void {
  const cx = e.x + e.w / 2
  const by = e.y + e.h
  e.snow = SNOW_FULL
  e.ballT = 0
  e.w = 18
  e.h = 18
  e.x = Math.max(T, Math.min(W - T - 18, cx - 9))
  e.y = by - 18
  e.vx = 0
  e.vy = Math.max(0, e.vy)
  // never stick a ball into the wall
  if (e.kind === 'mosquito') e.vy = 0.5
  s.events.push({ k: 'ball', x: cx, y: by - 9 })
}

function thaw(s: GameState, e: Enemy): void {
  const cx = e.x + e.w / 2
  const by = e.y + e.h
  e.snow = 0
  e.hold = 0
  e.w = e.bw
  e.h = e.bh
  e.x = cx - e.w / 2
  e.y = by - e.h
  e.angry = true
  e.vy = -2.5
  e.st = 0
  s.events.push({ k: 'thaw', x: cx, y: by - 8 })
}

function kill(s: GameState, e: Enemy, comboId: number, burst = false): void {
  const combo = s.combos[comboId]
  e.dying = 1
  e.rolling = false
  e.vy = -3.6
  e.vx = (rand(s) - 0.5) * 3
  const cx = e.x + e.w / 2
  const cy = e.y + e.h / 2
  let pts = 400
  let n = 1
  if (combo) {
    combo.n++
    n = combo.n
    pts = Math.min(12800, 400 * 2 ** (n - 1))
    const owner = s.players[combo.owner]
    owner.score += pts
    owner.floorScore += pts
    owner.kills++
    owner.bestChain = Math.max(owner.bestChain, n)
  }
  s.floorBest = Math.max(s.floorBest, n)
  s.events.push({ k: burst ? 'burst' : 'kill', x: cx, y: cy, n: pts, p: n })
  dropItem(s, cx, cy, n)
}

function dropItem(s: GameState, x: number, y: number, chain: number): void {
  const r = rand(s)
  let kind = -1
  if (r < 0.1) kind = ITEM_RED + Math.floor(rand(s) * 3)
  else if (r < 0.115) kind = ITEM_LIFE
  else if (chain >= 2 || r < 0.6) kind = Math.min(5, Math.max(0, chain - 1 + (rand(s) < 0.3 ? 1 : 0)))
  if (kind < 0) return
  s.items.push({ id: s.nextId++, kind, x: x - 5, y: y - 5, vx: (rand(s) - 0.5) * 1.2, vy: -2.4, t: 0 })
}

function hurtBoss(s: GameState, b: Enemy, dmg: number, x: number, y: number): void {
  if (b.dying || b.spawnT > 0) return
  b.hp -= dmg
  b.flash = 8
  s.events.push({ k: 'bosshit', x, y, n: dmg })
  if (b.hp <= 0) {
    b.hp = 0
    b.dying = 1
    b.vx = 0
    b.vy = -2
    for (const p of s.players) {
      if (!p.on) continue
      p.score += 10000
      p.floorScore += 10000
    }
    s.events.push({ k: 'bossdie', x: b.x + b.w / 2, y: b.y + b.h / 2, n: 10000 })
    for (const o of s.enemies) {
      if (o === b || o.dying) continue
      kill(s, o, 0)
    }
    s.projs = []
  } else if (!b.angry && b.hp < b.maxHp / 2) b.angry = true
}

function walkerAI(s: GameState, e: Enemy, speed: number, canJump: boolean): void {
  if (e.ground) {
    e.vx = e.face * speed
    // look at the floor one step ahead
    const fx = e.face > 0 ? e.x + e.w + 1 : e.x - 1
    const below = tileAt(s, Math.floor(fx / T), Math.floor((e.y + e.h + 2) / T))
    if (below === 0 && e.edge <= 0) {
      e.edge = 30
      if (rand(s) < 0.55) {
        e.face = -e.face as 1 | -1
        e.vx = e.face * speed
      }
    }
    if (e.edge > 0) e.edge--
    if (canJump && --e.cd <= 0) {
      e.cd = 120 + Math.floor(rand(s) * 160)
      // hop up if a shelf sits a floor above
      const c = Math.floor((e.x + e.w / 2) / T)
      const r = Math.floor((e.y + e.h) / T)
      for (let up = 3; up <= 6; up++) {
        if (tileAt(s, c, r - up) === 2) {
          e.vy = -4.5
          e.ground = false
          break
        }
      }
    }
  }
  if (moveX(s, e)) e.face = -e.face as 1 | -1
  if (e.x <= T) e.face = 1
  if (e.x + e.w >= W - T) e.face = -1
}

function updateEnemy(s: GameState, e: Enemy): void {
  if (e.dying) {
    e.dying++
    if (e.boss) {
      if (e.dying % 6 === 0) s.events.push({ k: 'burst', x: e.x + rand(s) * e.w, y: e.y + rand(s) * e.h, n: 0, p: 0 })
      return
    }
    e.x += e.vx
    e.vy += 0.25
    e.y += e.vy
    return
  }
  if (e.spawnT > 0) {
    if (--e.spawnT === 0) s.events.push({ k: 'spawn', x: e.x + e.w / 2, y: e.y + e.h / 2 })
    return
  }
  if (e.flash > 0) e.flash--
  // pests wait for the floor title to finish
  if (s.phase === 'intro') return
  e.t++
  if (e.boss) {
    updateBoss(s, e)
    return
  }
  if (e.kind === 'slipper') {
    const p = nearestPlayer(s, e)
    if (p) {
      const dx = p.x + p.w / 2 - (e.x + e.w / 2)
      const dy = p.y + p.h / 2 - (e.y + e.h / 2)
      const d = Math.hypot(dx, dy) || 1
      const sp = 0.75 + s.hurry * 0.12
      e.vx += ((dx / d) * sp - e.vx) * 0.05
      e.vy += ((dy / d) * sp - e.vy) * 0.05
      e.face = dx > 0 ? 1 : -1
    }
    e.x += e.vx
    e.y += e.vy
    return
  }
  if (e.rolling) {
    rollBall(s, e)
    return
  }
  if (e.snow >= SNOW_FULL) {
    e.ballT++
    if (e.ballT >= BALL_TIME) {
      thaw(s, e)
      return
    }
    e.vy = Math.min(MAX_FALL, e.vy + GRAV)
    moveY(s, e, true)
    return
  }
  // floss melts away when nobody keeps shooting
  if (e.snow > 0) {
    e.hold++
    if (e.hold > 80) e.snow = Math.max(0, e.snow - 1 / 55)
  }
  const slow = e.snow >= 2 ? 0 : e.snow > 0 ? 0.35 : 1
  const mad = e.angry ? 1.55 : 1
  const flying = e.kind === 'mosquito' && e.snow < 1
  if (!flying) e.vy = Math.min(MAX_FALL, e.vy + GRAV)
  switch (e.kind) {
    case 'book':
      walkerAI(s, e, 0.55 * mad * slow, slow === 1)
      break
    case 'clock': {
      if (e.ground) {
        e.vx = 0
        if (slow === 1 && --e.cd <= 0) {
          const p = nearestPlayer(s, e)
          e.cd = 45 + Math.floor(rand(s) * 40) - (e.angry ? 20 : 0)
          if (p) e.face = p.x > e.x ? 1 : -1
          const up = p && p.y + p.h < e.y - 20
          e.vx = e.face * (up ? 0.9 : 1.15) * mad
          e.vy = up ? -4.6 : -3.4
          e.ground = false
          s.events.push({ k: 'ring', x: e.x + e.w / 2, y: e.y, p: 0 })
        }
      } else e.vx *= slow === 1 ? 1 : 0.9
      if (moveX(s, e)) {
        e.face = -e.face as 1 | -1
        e.vx = -e.vx
      }
      break
    }
    case 'mosquito': {
      if (flying) {
        const p = nearestPlayer(s, e)
        const sp = 0.7 * mad * slow
        if (p) {
          const dx = p.x - e.x
          const dy = p.y - 6 - e.y
          const d = Math.hypot(dx, dy) || 1
          e.vx += ((dx / d) * sp - e.vx) * 0.03
          e.vy += ((dy / d) * sp - e.vy) * 0.03
        }
        e.vy += Math.sin(e.t / 14) * 0.03
        e.face = e.vx >= 0 ? 1 : -1
        e.x = Math.max(T + 1, Math.min(W - T - e.w - 1, e.x + e.vx))
        e.y = Math.max(20, Math.min(GROUND_Y - e.h, e.y + e.vy))
        return
      }
      e.vx = 0
      break
    }
    case 'chalk': {
      if (e.st > 0) {
        // winding up a throw
        e.vx = 0
        if (--e.st === 0) {
          s.projs.push({ id: s.nextId++, kind: 'chalk', x: e.x + e.w / 2 - 3, y: e.y + 3, w: 6, h: 4, vx: e.face * 2.1, vy: -1.2, t: 0, life: 150 })
          s.events.push({ k: 'throw', x: e.x + e.w / 2, y: e.y + 4 })
        }
        moveX(s, e)
        break
      }
      walkerAI(s, e, 0.4 * mad * slow, slow === 1)
      if (slow === 1 && e.ground && e.t % (e.angry ? 100 : 150) === 0) {
        const p = nearestPlayer(s, e)
        if (p && Math.abs(p.y + p.h - (e.y + e.h)) < 24) {
          e.face = p.x > e.x ? 1 : -1
          e.st = 26
        }
      }
      break
    }
    case 'coil': {
      if (e.st > 0) {
        e.st--
        e.vx = e.face * 2.3 * mad * slow
        if (moveX(s, e)) {
          e.face = -e.face as 1 | -1
          e.st = 0
          e.cd = 40
        }
        if (e.st === 0) e.cd = 50
        break
      }
      walkerAI(s, e, 0.45 * mad * slow, false)
      if (e.cd > 0) e.cd--
      else if (slow === 1 && e.ground) {
        const p = nearestPlayer(s, e)
        if (p && Math.abs(p.y + p.h - (e.y + e.h)) < 10 && Math.sign(p.x - e.x) === e.face && Math.abs(p.x - e.x) < 150) {
          e.st = 60
          s.events.push({ k: 'throw', x: e.x + e.w / 2, y: e.y + 4, p: 1 })
        }
      }
      break
    }
  }
  moveY(s, e, true)
}

function rollBall(s: GameState, e: Enemy): void {
  e.rollT++
  e.vy = Math.min(MAX_FALL, e.vy + GRAV)
  if (moveX(s, e) || e.x <= T || e.x + e.w >= W - T) {
    e.x = Math.max(T, Math.min(W - T - e.w, e.x))
    e.vx = -e.vx
    e.bounces++
    s.events.push({ k: 'bounce', x: e.x + (e.vx < 0 ? e.w : 0), y: e.y + e.h / 2, n: e.bounces })
  }
  const was = e.ground
  moveY(s, e, true)
  if (e.ground && !was && e.rollT > 4) s.events.push({ k: 'land', x: e.x + e.w / 2, y: e.y + e.h, p: 9 })
  for (const o of s.enemies) {
    if (o === e || o.dying || o.spawnT > 0) continue
    if (!overlap(e, o, 1)) continue
    if (o.boss) {
      hurtBoss(s, o, 10, e.x + e.w / 2, e.y + e.h / 2)
      kill(s, e, e.combo, true)
      return
    }
    if (o.kind === 'slipper') continue
    if (isBall(o)) {
      if (o.rolling) continue
      o.rolling = true
      o.combo = e.combo
      o.vx = Math.sign(e.vx) * BALL_SPEED
      o.vy = -1.5
      o.bounces = 0
      o.rollT = 0
      s.events.push({ k: 'kick', x: o.x + o.w / 2, y: o.y + o.h / 2, p: 9 })
    } else kill(s, o, e.combo)
  }
  for (const p of s.players) {
    if (!p.on || p.state !== ALIVE || p.bump > 0) continue
    if (!overlap(e, p, 2)) continue
    if (e.rollT < 10 && s.combos[e.combo]?.owner === p.idx) continue
    p.vy = -3.6
    p.vx = Math.sign(e.vx) * 1.6
    p.bump = 22
    p.ground = false
    s.events.push({ k: 'bump', x: p.x + p.w / 2, y: p.y + p.h / 2, p: p.idx })
  }
  if (e.bounces >= 4 || e.rollT > 540) kill(s, e, e.combo, true)
}

function spawnMinion(s: GameState, kind: EnemyKind, x: number, y: number): void {
  const minions = s.enemies.filter((o) => !o.boss && !o.dying && o.kind !== 'slipper').length
  if (minions >= 4) return
  const m = makeEnemy(s, kind, x, y)
  m.spawnT = 12
  m.vy = -2
  m.face = rand(s) < 0.5 ? 1 : -1
  s.enemies.push(m)
}

function updateBoss(s: GameState, b: Enemy): void {
  const p = nearestPlayer(s, b)
  const tx = p ? p.x + p.w / 2 : W / 2
  const cx = b.x + b.w / 2
  const fast = b.angry ? 1.35 : 1
  if (b.kind === 'bigclock') {
    b.vy = Math.min(MAX_FALL + 2, b.vy + GRAV * 1.2)
    // st: 0 walk, 1 jump, 2 ring
    if (b.st === 0) {
      b.face = tx > cx ? 1 : -1
      b.vx = b.ground ? b.face * 0.5 * fast : b.vx
      if (--b.cd <= 0 && b.ground) {
        b.cd = 70
        if (rand(s) < 0.55) {
          b.st = 1
          b.vy = b.angry ? -6.4 : -5.6
          b.vx = Math.max(-1.6, Math.min(1.6, (tx - cx) / 70))
          b.ground = false
        } else {
          b.st = 2
          b.vx = 0
        }
      }
    } else if (b.st === 2) {
      b.vx = 0
      if (--b.cd <= 0) {
        s.events.push({ k: 'ring', x: cx, y: b.y, p: 1 })
        spawnMinion(s, 'clock', b.x + 4, b.y)
        spawnMinion(s, 'clock', b.x + b.w - 4, b.y)
        b.st = 0
        b.cd = 110 + Math.floor(rand(s) * 60)
      }
    }
    if (moveX(s, b)) b.vx = -b.vx
    const was = b.ground
    moveY(s, b, false)
    if (b.ground && !was && b.y > 100) {
      s.events.push({ k: 'quake', x: cx, y: b.y + b.h })
      if (b.st === 1) {
        const y = b.y + b.h - 8
        for (const d of b.angry ? [-1, 1, -0.6, 0.6] : [-1, 1]) {
          s.projs.push({ id: s.nextId++, kind: 'wave', x: cx - 5 + d * 14, y, w: 10, h: 8, vx: d * 1.9, vy: 0, t: 0, life: 170 })
        }
        b.st = 0
        b.cd = 90 + Math.floor(rand(s) * 60)
      }
      // landing kicks kids standing on the ground into the air
      for (const q of s.players) {
        if (q.on && q.state === ALIVE && q.ground && q.y + q.h >= GROUND_Y - 1) {
          q.vy = -2.2
          q.ground = false
        }
      }
    }
  } else if (b.kind === 'bookpile') {
    b.vy = Math.min(MAX_FALL, b.vy + GRAV)
    if (b.ground) {
      // shuffle along the right-hand side
      if (b.t % 200 === 0) b.face = rand(s) < 0.5 ? 1 : -1
      b.vx = b.face * 0.3
      if (b.x < 190) b.face = 1
      if (b.x > W - T - b.w - 2) b.face = -1
    }
    if (--b.cd <= 0 && b.ground && p) {
      b.cd = b.angry ? 62 : 95
      const n = b.angry ? 3 : 1
      for (let i = 0; i < n; i++) {
        const dx = tx - cx
        const dy = p.y - b.y
        const d = Math.hypot(dx, dy) || 1
        const spread = (i - (n - 1) / 2) * 0.35
        s.projs.push({
          id: s.nextId++,
          kind: 'plane',
          x: cx - 6,
          y: b.y + 6,
          w: 12,
          h: 6,
          vx: (dx / d) * 1.5 + spread * 0.4,
          vy: (dy / d) * 1.5 + spread,
          t: 0,
          life: 320
        })
      }
      s.events.push({ k: 'throw', x: cx, y: b.y + 4, p: 2 })
    }
    if (b.t % (b.angry ? 260 : 360) === 120) {
      spawnMinion(s, 'book', cx, b.y)
      s.events.push({ k: 'ring', x: cx, y: b.y, p: 2 })
    }
    moveX(s, b)
    const was = b.ground
    moveY(s, b, false)
    if (b.ground && !was) s.events.push({ k: 'quake', x: cx, y: b.y + b.h })
  } else {
    // calendar: floats in a figure eight, drops pages, calls the whole holiday's worth of pests
    if (b.st === 1) {
      // diving at a kid
      b.cd--
      b.x += b.vx
      b.y += b.vy
      if (b.cd <= 0 || b.y + b.h > GROUND_Y - 2) {
        b.st = 2
        b.cd = 60
      }
    } else {
      const t = b.t * 0.011 * fast
      const gx = W / 2 + Math.sin(t) * 112 - b.w / 2
      const gy = 66 + Math.sin(t * 2) * 40
      const k = b.st === 2 ? 0.04 : 0.08
      b.x += (gx - b.x) * k
      b.y += (gy - b.y) * k
      if (b.st === 2 && --b.cd <= 0) b.st = 0
      if (b.st === 0 && b.t % (b.angry ? 55 : 80) === 0) {
        s.projs.push({ id: s.nextId++, kind: 'page', x: cx - 5, y: b.y + b.h - 6, w: 10, h: 8, vx: 0, vy: 0.7, t: 0, life: 420 })
        s.events.push({ k: 'throw', x: cx, y: b.y + b.h, p: 3 })
      }
      if (b.st === 0 && b.t % 420 === 200) {
        const kinds: EnemyKind[] = ['mosquito', 'coil', 'chalk', 'book', 'clock']
        spawnMinion(s, kinds[Math.floor(rand(s) * kinds.length)], cx - 16, b.y + b.h)
        spawnMinion(s, kinds[Math.floor(rand(s) * kinds.length)], cx + 16, b.y + b.h)
        s.events.push({ k: 'ring', x: cx, y: b.y, p: 3 })
      }
      if (b.st === 0 && b.t % (b.angry ? 240 : 360) === 150 && p) {
        const dx = tx - cx
        const dy = p.y - b.y
        const d = Math.hypot(dx, dy) || 1
        b.vx = (dx / d) * 2.6
        b.vy = (dy / d) * 2.6
        b.st = 1
        b.cd = 70
        s.events.push({ k: 'ring', x: cx, y: b.y, p: 4 })
      }
    }
    b.face = tx > cx ? 1 : -1
    b.x = Math.max(T, Math.min(W - T - b.w, b.x))
  }
}

// ---------------------------------------------------------------- shots, projectiles, items

function updateShots(s: GameState): void {
  const keep: Shot[] = []
  for (const sh of s.shots) {
    sh.t++
    sh.x += sh.vx
    if (sh.t > sh.life * 0.55) sh.vy += 0.09
    sh.y += sh.vy
    const box = { x: sh.x, y: sh.y, w: sh.big ? 9 : 7, h: sh.big ? 8 : 6 }
    let gone = sh.t > sh.life
    const c = Math.floor((sh.x + box.w / 2) / T)
    const r = Math.floor((sh.y + box.h / 2) / T)
    if (tileAt(s, c, r) === 1) gone = true
    if (!gone) {
      for (const e of s.enemies) {
        if (e.dying || e.spawnT > 0 || e.rolling || e.kind === 'slipper') continue
        if (!overlap(box, e)) continue
        gone = true
        const owner = s.players[sh.o]
        const pow = sh.big ? 2 : 1
        if (e.boss) {
          // floss barely scratches a boss: roll balls into it
          hurtBoss(s, e, pow * 0.2, sh.x + 3, sh.y + 3)
          break
        }
        if (e.snow >= SNOW_FULL) {
          e.ballT = Math.min(e.ballT, 120)
          s.events.push({ k: 'nudge', x: sh.x + 3, y: sh.y + 3 })
          break
        }
        e.snow = Math.min(SNOW_FULL, e.snow + pow)
        e.hold = 0
        owner.score += 10
        owner.floorScore += 10
        if (e.snow >= SNOW_FULL) {
          owner.score += 40
          owner.floorScore += 40
          becomeBall(s, e)
        } else s.events.push({ k: 'hit', x: sh.x + 3, y: sh.y + 3, n: Math.ceil(e.snow) })
        break
      }
    }
    if (!gone) {
      for (const pr of s.projs) {
        if (pr.kind === 'wave' || pr.t < 0) continue
        if (!overlap(box, pr)) continue
        pr.t = -1
        gone = true
        const owner = s.players[sh.o]
        owner.score += 50
        owner.floorScore += 50
        s.events.push({ k: 'poof', x: pr.x + pr.w / 2, y: pr.y + pr.h / 2, n: 50 })
        break
      }
    }
    if (gone) {
      if (sh.t > sh.life) s.events.push({ k: 'poof', x: sh.x + 3, y: sh.y + 3 })
    } else keep.push(sh)
  }
  s.shots = keep
}

function updateProjs(s: GameState): void {
  const keep: Proj[] = []
  for (const pr of s.projs) {
    if (pr.t < 0) continue
    pr.t++
    if (pr.kind === 'chalk') {
      pr.vy += 0.07
      pr.x += pr.vx
      pr.y += pr.vy
      if (tileAt(s, Math.floor((pr.x + 3) / T), Math.floor((pr.y + 2) / T)) === 1) pr.t = pr.life + 1
    } else if (pr.kind === 'wave') {
      pr.x += pr.vx
      if (pr.x < T || pr.x + pr.w > W - T) pr.t = pr.life + 1
    } else if (pr.kind === 'plane') {
      pr.x += pr.vx
      pr.y += pr.vy + Math.sin(pr.t / 9) * 0.5
    } else {
      pr.x += Math.sin(pr.t / 16) * 1.1
      pr.y += pr.vy
      if (pr.y + pr.h >= GROUND_Y) pr.t = Math.max(pr.t, pr.life - 40)
    }
    if (pr.t <= pr.life && pr.x > -20 && pr.x < W + 20 && pr.y < H + 10 && pr.y > -20) keep.push(pr)
  }
  s.projs = keep
}

function updateItems(s: GameState): void {
  const keep: Item[] = []
  for (const it of s.items) {
    it.t++
    const body = { x: it.x, y: it.y, w: 10, h: 10, vx: it.vx, vy: Math.min(2.5, it.vy + 0.15), ground: false }
    moveX(s, body)
    body.x = Math.max(T, Math.min(W - T - 10, body.x))
    moveY(s, body, true)
    it.x = body.x
    it.y = body.y
    it.vy = body.vy
    it.vx = body.ground ? 0 : it.vx
    let taken = false
    for (const p of s.players) {
      if (!p.on || p.state !== ALIVE) continue
      if (!overlap(p, { x: it.x, y: it.y, w: 10, h: 10 })) continue
      taken = true
      let pts = 0
      if (it.kind <= 5) pts = SNACK_POINTS[it.kind]
      else if (it.kind === ITEM_RED) p.power = true
      else if (it.kind === ITEM_BLUE) p.speed = true
      else if (it.kind === ITEM_YELLOW) p.range = true
      else if (it.kind === ITEM_LIFE) p.lives = Math.min(9, p.lives + 1)
      else if (it.kind === ITEM_PACKET) pts = 300
      if (it.kind >= ITEM_RED && it.kind <= ITEM_LIFE) pts = 500
      p.score += pts
      p.floorScore += pts
      s.events.push({ k: 'pickup', x: it.x + 5, y: it.y + 5, n: pts, p: it.kind })
      break
    }
    if (!taken && it.t < 600) keep.push(it)
  }
  s.items = keep
}

function checkHits(s: GameState): void {
  for (const p of s.players) {
    if (!p.on || p.state !== ALIVE || p.inv > 0) continue
    let dead = false
    for (const e of s.enemies) {
      if (e.dying || e.spawnT > 0) continue
      if (e.kind !== 'slipper' && !e.boss && (e.snow >= 2 || e.rolling)) continue
      if (overlap(p, e, e.boss ? 4 : 2)) {
        dead = true
        break
      }
    }
    if (!dead) {
      for (const pr of s.projs) {
        if (overlap(p, pr, 1)) {
          dead = true
          pr.t = -1
          break
        }
      }
    }
    if (dead) die(s, p)
  }
}

// ---------------------------------------------------------------- the frame

function anyoneLeft(s: GameState): boolean {
  for (const p of s.players) {
    if (!p.on) continue
    if (p.state === ALIVE) return true
    if (p.state === GHOST && (p.lives > 0 && (p.ghostT > 0 || p.respawnT > 0))) return true
  }
  return false
}

export function step(s: GameState, inputs: [Buttons, Buttons]): void {
  s.frame++
  s.phaseT++
  if (s.phase === 'over' || s.phase === 'won') return
  if (s.phase === 'result') {
    if (s.phaseT >= 210) {
      if (s.floor + 1 >= FLOOR_COUNT) {
        s.phase = 'won'
        s.phaseT = 0
        s.events.push({ k: 'won', x: W / 2, y: H / 2 })
      } else startFloor(s, s.floor + 1)
    }
    return
  }
  s.played++
  if (s.phase === 'intro' && s.phaseT >= 100) {
    s.phase = 'play'
    s.phaseT = 0
  }
  for (const p of s.players) updatePlayer(s, p, inputs[p.idx])
  updateShots(s)
  for (const e of s.enemies) updateEnemy(s, e)
  s.enemies = s.enemies.filter((e) => !(e.dying && (e.boss ? e.dying > 130 : e.y > H + 24 || e.dying > 160)))
  updateProjs(s)
  updateItems(s)
  if (s.phase === 'play') {
    checkHits(s)
    s.floorT++
    const def = FLOORS[s.floor]
    if (!def.boss) {
      if (s.floorT === HURRY_WARN) s.events.push({ k: 'hurry', x: W / 2, y: H / 2 })
      if (s.floorT >= HURRY_AT && s.hurry < 3 && (s.floorT - HURRY_AT) % (12 * FPS) === 0) {
        const p = nearestPlayer(s, { x: W / 2, y: H / 2, w: 0, h: 0 })
        const x = p && p.x < W / 2 ? W - 20 : 20
        const slip = makeEnemy(s, 'slipper', x, 30)
        slip.spawnT = 1
        s.enemies.push(slip)
        s.hurry++
        s.events.push({ k: 'slipper', x, y: 30 })
      }
    }
    const left = s.enemies.filter((e) => !e.dying && e.kind !== 'slipper').length
    if (left === 0 && s.enemies.every((e) => !e.boss || e.dying > 100)) {
      // 一球清场: one kick took out everything that stood on the floor
      const combos = Object.values(s.combos)
      s.sweep = combos.some((c) => c.alive >= 3 && c.n >= c.alive)
      if (s.sweep) {
        for (let i = 0; i < 14; i++) {
          s.items.push({ id: s.nextId++, kind: ITEM_PACKET, x: 20 + rand(s) * (W - 50), y: 18 + rand(s) * 30, vx: 0, vy: rand(s), t: 0 })
        }
        s.events.push({ k: 'sweep', x: W / 2, y: H / 2 })
      }
      s.enemies = s.enemies.filter((e) => e.kind !== 'slipper' || e.dying)
      s.projs = []
      s.phase = 'clear'
      s.phaseT = 0
      s.events.push({ k: 'clear', x: W / 2, y: H / 2 })
    }
    if (!anyoneLeft(s)) {
      s.phase = 'over'
      s.phaseT = 0
      s.events.push({ k: 'over', x: W / 2, y: H / 2 })
    }
  } else if (s.phase === 'clear') {
    if (!anyoneLeft(s)) {
      // a kid who fell in the clean-up keeps going with the partner
      for (const p of s.players) if (p.on && p.state !== ALIVE) spawnAt(s, p)
    }
    if (s.phaseT >= (s.sweep ? 260 : 150)) {
      const seconds = Math.round(s.floorT / FPS)
      const bonus = FLOORS[s.floor].boss ? 5000 : Math.max(0, 40 - seconds) * 100
      for (const p of s.players) {
        if (!p.on) continue
        p.score += bonus
        p.floorScore += bonus
      }
      s.result = {
        seconds,
        bonus,
        floorScore: [s.players[0].floorScore, s.players[1].floorScore],
        bestChain: s.floorBest,
        sweep: s.sweep
      }
      s.phase = 'result'
      s.phaseT = 0
      s.items = []
      s.shots = []
    }
  }
}

/** after a game over: same floor, fresh lives, the score starts again */
export function continueGame(s: GameState): void {
  for (const p of s.players) {
    if (!p.on) continue
    p.lives = 3
    p.score = 0
    p.kills = 0
    p.bestChain = 0
    p.speed = p.range = p.power = false
  }
  startFloor(s, s.floor)
}

export function totalScore(s: GameState): number {
  return s.players.reduce((n, p) => n + (p.on ? p.score : 0), 0)
}
