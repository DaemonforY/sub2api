// Player movement. This file decides how the game feels, so every number here was tuned by hand.
//
// The model follows the classic 8-bit platformers: speed builds up and runs down instead of
// snapping, turning round at speed skids, holding jump makes the jump higher (lighter gravity while
// rising), and running faster gives a higher jump. On top of that come three modern kindnesses:
// coyote time (you can still jump a few steps after running off a ledge), jump buffering (a jump
// pressed just before landing still fires) and corner correction (clipping a block's corner with
// your head slides you past it instead of stopping the jump).
//
// stepPlayer only knows about tiles and moving platforms, never enemies, so the online guest can
// run it to predict its own player and replay inputs when the host's answer arrives.

import { BTN, ROWS, T, TILE } from './types'

export const PHYS = {
  walkMax: 1.5,
  runMax: 2.55,
  /** speed you start at when you begin to walk from standstill */
  startSpeed: 0.13,
  accWalk: 0.048,
  accRun: 0.062,
  /** slowing when no direction is held (on the ground) */
  decRelease: 0.055,
  /** slowing when holding the opposite direction */
  decSkid: 0.15,
  airAcc: 0.05,
  airTurn: 0.09,
  /** jump speed by how fast you were running at take-off */
  jumpV: [4.0, 4.15, 4.6],
  /** gravity while rising with jump held, by take-off speed */
  gravHold: [0.13, 0.125, 0.15],
  /** gravity after jump is released or while falling */
  gravFall: [0.42, 0.4, 0.48],
  /** a few steps around the top of the arc fall gently: easier to aim a landing */
  apexBand: 0.6,
  apexScale: 0.6,
  fallMax: 4.4,
  coyote: 6,
  buffer: 8,
  /** how many pixels of head overlap get nudged round a corner */
  corner: 5,
  stompBounce: 4.0,
  stompBounceHeld: 5.4
}

export interface PState {
  /** feet position: x is the centre, y is the bottom edge */
  x: number
  y: number
  vx: number
  vy: number
  /** 0 small, 1 big (大力包子), 2 chili (辣椒) */
  power: number
  crouch: boolean
  ground: boolean
  coyote: number
  buffer: number
  /** inside a jump whose height still depends on holding the button */
  rising: boolean
  speedClass: number
  facing: number
  /** buttons held last step, for edge detection */
  prev: number
  skid: boolean
  /** index of the moving platform being ridden, -1 when none */
  ride: number
}

export interface Plat {
  /** top-left corner and width, in pixels, before and after this step */
  x0: number
  y0: number
  x1: number
  y1: number
  w: number
}

export interface TileMap {
  width: number
  get(tx: number, ty: number): string
}

export interface StepResult {
  /** tile hit from below this step, if any */
  bump: { tx: number; ty: number } | null
  jumped: boolean
  landed: boolean
}

export function newPState(x: number, y: number, power = 0): PState {
  return {
    x,
    y,
    vx: 0,
    vy: 0,
    power,
    crouch: false,
    ground: false,
    coyote: 0,
    buffer: 0,
    rising: false,
    speedClass: 0,
    facing: 1,
    prev: 0,
    skid: false,
    ride: -1
  }
}

export function clonePState(p: PState): PState {
  return { ...p }
}

const SOLID = new Set<string>([
  T.GROUND,
  T.BRICK,
  T.BOX,
  T.BOX_POWER,
  T.BOX_BELL,
  T.BRICK_COINS,
  T.BRICK_BELL,
  T.BRICK_POWER,
  T.USED,
  T.HARD,
  T.TUBE_TL,
  T.TUBE_TR,
  T.TUBE_L,
  T.TUBE_R,
  T.WALL,
  T.BRIDGE
])
const HIDDEN = new Set<string>([T.BOX_LIFE, T.BOX_HIDDEN])

export function isSolid(c: string): boolean {
  return SOLID.has(c)
}
export function isHidden(c: string): boolean {
  return HIDDEN.has(c)
}

/** cell lookup with the level edges: the sides are walls, above and below are open air */
export function cell(map: TileMap, tx: number, ty: number): string {
  if (ty >= ROWS) return T.EMPTY
  if (tx < 0 || tx >= map.width) return T.HARD
  if (ty < 0) return T.EMPTY
  return map.get(tx, ty)
}

export function bodyW(p: PState): number {
  return p.power > 0 ? 12 : 10
}
export function bodyH(p: PState): number {
  return p.power > 0 && !p.crouch ? 26 : 14
}

function solidAt(map: TileMap, tx: number, ty: number): boolean {
  return isSolid(cell(map, tx, ty))
}

/** does the box [l,r)×[t,b) overlap any solid tile? */
export function boxHitsSolid(map: TileMap, l: number, t: number, r: number, b: number): boolean {
  const tx0 = Math.floor(l / TILE)
  const tx1 = Math.floor((r - 0.001) / TILE)
  const ty0 = Math.floor(t / TILE)
  const ty1 = Math.floor((b - 0.001) / TILE)
  for (let ty = ty0; ty <= ty1; ty++) for (let tx = tx0; tx <= tx1; tx++) if (solidAt(map, tx, ty)) return true
  return false
}

function approach(v: number, target: number, amount: number): number {
  if (v < target) return Math.min(target, v + amount)
  if (v > target) return Math.max(target, v - amount)
  return v
}

/**
 * One fixed step of player movement. `btn` is the BTN bitmask held this step. Platforms are given
 * at their positions before and after the step, so riding one carries the player along.
 */
export function stepPlayer(p: PState, btn: number, map: TileMap, plats: Plat[] = []): StepResult {
  const res: StepResult = { bump: null, jumped: false, landed: false }
  const left = (btn & BTN.L) !== 0
  const right = (btn & BTN.R) !== 0
  const down = (btn & BTN.D) !== 0
  const jumpHeld = (btn & BTN.J) !== 0
  const jumpPressed = jumpHeld && (p.prev & BTN.J) === 0
  const run = (btn & BTN.F) !== 0
  const dir = left === right ? 0 : right ? 1 : -1

  // ride a moving platform: move with it before anything else
  if (p.ride >= 0 && p.ride < plats.length) {
    const pl = plats[p.ride]
    p.x += pl.x1 - pl.x0
    p.y = pl.y1
  }

  // crouch (big players only, on the ground): smaller hitbox, slides to a stop
  const wasCrouch = p.crouch
  if (p.ground) p.crouch = p.power > 0 && down && dir === 0 ? true : p.power > 0 && down && p.crouch
  if (!down || p.power === 0) p.crouch = false
  // standing up under a low ceiling keeps you crouched
  if (wasCrouch && !p.crouch && boxHitsSolid(map, p.x - bodyW(p) / 2, p.y - 26, p.x + bodyW(p) / 2, p.y - 14)) p.crouch = true

  // horizontal speed
  const max = run ? PHYS.runMax : PHYS.walkMax
  p.skid = false
  if (p.ground) {
    if (p.crouch || dir === 0) {
      p.vx = approach(p.vx, 0, PHYS.decRelease)
    } else if (p.vx !== 0 && Math.sign(p.vx) !== dir) {
      p.vx = approach(p.vx, 0, PHYS.decSkid)
      p.skid = Math.abs(p.vx) > 0.6
      if (Math.abs(p.vx) < 0.3) p.vx = dir * PHYS.startSpeed
    } else {
      if (Math.abs(p.vx) < PHYS.startSpeed) p.vx = dir * PHYS.startSpeed
      const acc = run ? PHYS.accRun : PHYS.accWalk
      // above the cap (e.g. let go of run): ease back down rather than snap
      if (Math.abs(p.vx) > max) p.vx = approach(p.vx, dir * max, PHYS.decRelease)
      else p.vx = approach(p.vx, dir * max, acc)
    }
    if (dir !== 0 && !p.crouch) p.facing = dir
  } else if (dir !== 0) {
    // air control: speed keeps its size, turning is quicker than on the ground but not instant
    const cap = Math.max(max, Math.abs(p.vx) * (Math.sign(p.vx) === dir ? 1 : 0))
    if (p.vx !== 0 && Math.sign(p.vx) !== dir) p.vx = approach(p.vx, 0, PHYS.airTurn)
    else p.vx = approach(p.vx, dir * Math.min(cap, max), PHYS.airAcc)
  }

  // jump: buffered presses and coyote steps
  if (jumpPressed) p.buffer = PHYS.buffer
  else if (p.buffer > 0) p.buffer--
  if (p.ground) p.coyote = PHYS.coyote
  else if (p.coyote > 0) p.coyote--
  if (p.buffer > 0 && p.coyote > 0) {
    const s = Math.abs(p.vx)
    p.speedClass = s < 1 ? 0 : s < 2.3 ? 1 : 2
    p.vy = -PHYS.jumpV[p.speedClass]
    p.rising = true
    p.ground = false
    p.coyote = 0
    p.buffer = 0
    p.ride = -1
    res.jumped = true
  }

  // gravity: light while rising with the button held, heavy otherwise
  if (p.vy >= 0 || !jumpHeld) p.rising = false
  let g = p.rising ? PHYS.gravHold[p.speedClass] : PHYS.gravFall[p.speedClass]
  if (jumpHeld && Math.abs(p.vy) < PHYS.apexBand && !p.ground) g *= PHYS.apexScale
  p.vy = Math.min(PHYS.fallMax, p.vy + g)

  const w = bodyW(p)
  const h = bodyH(p)

  // --- move along x
  p.x += p.vx
  {
    const l = p.x - w / 2
    const r = p.x + w / 2
    const t = p.y - h
    const b = p.y
    const ty0 = Math.floor(t / TILE)
    const ty1 = Math.floor((b - 0.001) / TILE)
    if (p.vx > 0 || (p.vx === 0 && dir > 0)) {
      const tx = Math.floor((r - 0.001) / TILE)
      for (let ty = ty0; ty <= ty1; ty++)
        if (solidAt(map, tx, ty)) {
          p.x = tx * TILE - w / 2
          p.vx = 0
          break
        }
    }
    if (p.vx < 0 || (p.vx === 0 && dir < 0)) {
      const tx = Math.floor(l / TILE)
      for (let ty = ty0; ty <= ty1; ty++)
        if (solidAt(map, tx, ty)) {
          p.x = (tx + 1) * TILE + w / 2
          p.vx = 0
          break
        }
    }
  }

  // --- move along y
  const prevBottom = p.y
  p.y += p.vy
  const wasGround = p.ground
  p.ground = false
  const ridePrev = p.ride
  p.ride = -1
  {
    const l = p.x - w / 2
    const r = p.x + w / 2
    const tx0 = Math.floor(l / TILE)
    const tx1 = Math.floor((r - 0.001) / TILE)
    if (p.vy >= 0) {
      const ty = Math.floor((p.y - 0.001) / TILE)
      const top = ty * TILE
      for (let tx = tx0; tx <= tx1; tx++) {
        const c = cell(map, tx, ty)
        if (isSolid(c) || (c === T.PLANK && prevBottom <= top + 0.01)) {
          p.y = top
          p.vy = 0
          p.ground = true
          break
        }
      }
      // moving platforms (one-way, from above)
      if (!p.ground)
        for (let i = 0; i < plats.length; i++) {
          const pl = plats[i]
          if (r <= pl.x1 || l >= pl.x1 + pl.w) continue
          const wasAbove = prevBottom <= pl.y0 + 0.5 || (ridePrev === i && prevBottom <= pl.y1 + 0.5)
          if (wasAbove && p.y >= pl.y1) {
            p.y = pl.y1
            p.vy = 0
            p.ground = true
            p.ride = i
            break
          }
        }
    } else {
      const t = p.y - h
      const ty = Math.floor(t / TILE)
      const hits: number[] = []
      for (let tx = tx0; tx <= tx1; tx++) {
        const c = cell(map, tx, ty)
        if (isSolid(c) || isHidden(c)) hits.push(tx)
      }
      if (hits.length) {
        // corner correction: only one column hit and only a sliver of the head inside it
        let nudged = false
        if (hits.length === 1 && !isHidden(cell(map, hits[0], ty))) {
          const tx = hits[0]
          const overlapL = (tx + 1) * TILE - l // head's left part inside a tile to the left
          const overlapR = r - tx * TILE // head's right part inside a tile to the right
          if (tx === tx0 && overlapL <= PHYS.corner && !solidAt(map, tx + 1, ty)) {
            p.x += overlapL
            nudged = true
          } else if (tx === tx1 && overlapR <= PHYS.corner && !solidAt(map, tx - 1, ty)) {
            p.x -= overlapR
            nudged = true
          }
        }
        if (!nudged) {
          // bump the block nearest the head's centre
          let best = hits[0]
          for (const tx of hits) if (Math.abs((tx + 0.5) * TILE - p.x) < Math.abs((best + 0.5) * TILE - p.x)) best = tx
          res.bump = { tx: best, ty }
          p.y = (ty + 1) * TILE + h
          p.vy = Math.max(p.vy, 0.5)
          p.rising = false
        }
      }
    }
  }
  if (p.ground && !wasGround) res.landed = true
  p.prev = btn
  return res
}

/** bounce after stomping an enemy; holding jump bounces higher */
export function stompBounce(p: PState, jumpHeld: boolean): void {
  p.vy = -(jumpHeld ? PHYS.stompBounceHeld : PHYS.stompBounce)
  p.rising = jumpHeld
  p.speedClass = 1
  p.ground = false
  p.ride = -1
}
