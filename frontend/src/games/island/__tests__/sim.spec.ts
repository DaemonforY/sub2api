import { describe, expect, it } from 'vitest'
import { ROWS, T, TS, TO, TB, IN_L, IN_R, IN_J, IN_F, ENERGY_MAX } from '../defs'
import { stages } from '../level'
import { createWorld, movePlayer, newPlayer, physCtx, stepWorld, tileAt, PW, PH, type Player, type World } from '../sim'
import { decodePlayer, encodePlayer, makeSnapshot } from '../net'

const solid = (t: number) => t === TS || t === TB

/** A simple bot: runs right, jumps over gaps and walls, fights the boss with a golden boomerang. */
function bot(w: World, p: Player, mem: { held: boolean }): number {
  if (p.st !== 'play') return 0
  const boss = w.ents.find((e) => e.k === 'boss')
  let bits = 0
  if (boss && boss.st !== 9) {
    p.weapon = 2
    const bx = boss.x + boss.w / 2
    bits |= bx < p.x ? IN_L : IN_R
    if (w.frame % 14 < 7) bits |= IN_F
    if (p.ground && w.frame % 50 === 0) bits |= IN_J
  } else {
    const sun = w.ents.find((e) => e.k === 'sun')
    const target = sun ? sun.x : Infinity
    bits |= target < p.x ? IN_L : IN_R
    const ahead = p.x + PW + 10
    const foot = p.y + PH + 4
    const ground = solid(tileAt(w.stage.tiles, w.stage.cols, ahead, foot)) || tileAt(w.stage.tiles, w.stage.cols, ahead, foot) === TO
    const wall = solid(tileAt(w.stage.tiles, w.stage.cols, ahead, p.y + 8)) || solid(tileAt(w.stage.tiles, w.stage.cols, ahead, p.y + 18))
    const want = p.ground && (!ground || wall)
    if (want && !mem.held) bits |= IN_J
    else if (!p.ground && p.vy < 0 && mem.held) bits |= IN_J
  }
  mem.held = !!(bits & IN_J)
  return bits
}

describe('island stages', () => {
  it('builds three stages that end in a 25-column boss arena', () => {
    const list = stages()
    expect(list).toHaveLength(3)
    for (const s of list) {
      expect(s.cols).toBeGreaterThan(200)
      expect(s.arenaX).toBe((s.cols - 25) * T)
      expect(s.checkpoints.length).toBe(3)
      for (let i = 1; i < s.checkpoints.length; i++) expect(s.checkpoints[i]).toBeGreaterThan(s.checkpoints[i - 1])
      for (const sp of s.spawns) {
        expect(sp.x).toBeGreaterThan(0)
        expect(sp.x).toBeLessThan(s.cols * T)
      }
    }
  })

  it('has no gap wider than a running jump unless something bridges it', () => {
    for (const s of stages()) {
      let run = 0
      let bridged = false
      for (let c = 0; c < s.cols; c++) {
        const floor = solid(s.tiles[(ROWS - 1) * s.cols + c])
        if (floor) {
          if (!bridged) expect(run, `${s.name} gap ending at column ${c}`).toBeLessThanOrEqual(5)
          run = 0
          bridged = false
          continue
        }
        run++
        for (let r = 0; r < ROWS; r++) if (s.tiles[r * s.cols + c] === TO) bridged = true
        if (s.spawns.some((sp) => sp.k === 'plat' && Math.abs(sp.x - (c * T + 8)) < 24)) bridged = true
      }
    }
  })
})

describe('island world', () => {
  it('a bot can run every stage, beat every boss and reach the ending', () => {
    const w = createWorld(false)
    w.god = true
    const mem = { held: false }
    const seen = new Set<number>()
    let maxSnap = 0
    for (let f = 0; f < 60 * 60 * 12 && w.phase !== 'ending'; f++) {
      seen.add(w.stageIndex)
      stepWorld(w, [bot(w, w.players[0], mem), 0])
      if (f % 2 === 0) maxSnap = Math.max(maxSnap, JSON.stringify(makeSnapshot(w, 0, w.events)).length)
      w.events = []
    }
    expect([...seen]).toEqual([0, 1, 2])
    expect(w.phase).toBe('ending')
    expect(w.players[0].score).toBeGreaterThan(10000)
    // the relay drops messages over 8 KB
    expect(maxSnap).toBeLessThan(8000)
  })

  it('energy drains and fruit refills it', () => {
    const w = createWorld(false)
    const p = w.players[0]
    for (let i = 0; i < 600; i++) stepWorld(w, [0, 0])
    expect(p.energy).toBeLessThan(ENERGY_MAX * 0.85)
    const before = p.energy
    const fruit = w.ents.find((e) => e.k === 'fruit')
    expect(fruit).toBeDefined()
    if (fruit) {
      p.x = fruit.x
      p.y = fruit.y
    }
    stepWorld(w, [0, 0])
    expect(p.energy).toBeGreaterThan(before)
  })

  it('a fallen partner becomes a bubble and is revived by a touch', () => {
    const w = createWorld(true)
    const [p1, p2] = w.players
    p2.y = 400 // falls out of the world
    for (let i = 0; i < 90; i++) stepWorld(w, [0, 0])
    expect(p2.st).toBe('ghost')
    expect(w.lives).toBe(4)
    for (let i = 0; i < 400 && p2.st === 'ghost'; i++) {
      p1.x = p2.x
      p1.y = p2.y
      p1.vy = 0
      stepWorld(w, [0, 0])
    }
    expect(p2.st).toBe('play')
  })

  it('losing both heroes costs one life and restarts at the flag', () => {
    const w = createWorld(true)
    w.players[0].y = 400
    w.players[1].y = 400
    for (let i = 0; i < 120; i++) stepWorld(w, [0, 0])
    expect(w.lives).toBe(3)
    expect(w.players.every((p) => p.st === 'play')).toBe(true)
  })

  it('a boomerang from a clam knocks out a crab', () => {
    const w = createWorld(false)
    const p = w.players[0]
    p.weapon = 1
    const crab = w.ents.find((e) => e.k === 'crab')
    expect(crab).toBeDefined()
    if (!crab) return
    p.x = crab.x - 60
    p.y = crab.y + crab.h - PH
    w.camX = Math.max(0, p.x - 100)
    let ko = false
    for (let i = 0; i < 60 && !ko; i++) {
      stepWorld(w, [i === 2 ? IN_F : 0, 0])
      ko = crab.st === -1 || crab.dead
    }
    expect(ko).toBe(true)
    expect(p.score).toBeGreaterThan(0)
  })
})

describe('island netcode', () => {
  it('player state survives the snapshot encoding', () => {
    const w = createWorld(true)
    for (let i = 0; i < 30; i++) stepWorld(w, [IN_R, IN_R | (i === 5 ? IN_J : 0)])
    const p = w.players[1]
    const q = newPlayer(1, false)
    decodePlayer(encodePlayer(p), q)
    for (const k of ['x', 'y', 'vx', 'vy', 'face', 'weapon', 'prev'] as const) expect(q[k]).toBeCloseTo(p[k], 1)
    for (const k of ['ground', 'st', 'jheld', 'board'] as const) expect(q[k]).toEqual(p[k])
  })

  it('guest prediction replays to the same spot as the host', () => {
    const w = createWorld(true)
    const inputs: number[] = []
    for (let i = 0; i < 90; i++) inputs.push(IN_R | (i === 20 || i === 60 ? IN_J : 0) | (i > 20 && i < 40 ? IN_J : 0))
    // guest: start from the host state at frame 0 and replay every input locally
    const pred = newPlayer(1, true)
    decodePlayer(encodePlayer(w.players[1]), pred)
    const ctx = physCtx(w)
    for (const b of inputs) {
      ctx.camX = w.camX
      movePlayer(pred, b, ctx)
      stepWorld(w, [0, b])
    }
    expect(Math.abs(pred.x - w.players[1].x)).toBeLessThan(1)
    expect(Math.abs(pred.y - w.players[1].y)).toBeLessThan(1)
  })
})
