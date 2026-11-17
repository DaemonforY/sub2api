import { describe, expect, it } from 'vitest'

import { SONGS, parseMelody } from '../audio'
import { LEVELS } from '../levels'
import { packEnt, packPlayer, unpackEnt, unpackPlayer } from '../net'
import { PHYS, isSolid, newPState, stepPlayer, type TileMap } from '../physics'
import { BTN, ROWS, T, TILE } from '../types'
import { World, standY } from '../world'
import { playLevel } from './bot'

/** a flat test map: ground on rows 12-13, optional extra tiles */
function flat(width = 60, extra: [number, number, string][] = []): TileMap {
  const cells: string[][] = Array.from({ length: ROWS }, (_, y) => Array.from({ length: width }, () => (y >= 12 ? T.GROUND : T.EMPTY)))
  for (const [x, y, c] of extra) cells[y][x] = c
  return { width, get: (x, y) => cells[y][x] }
}

function settle(map: TileMap, x = 100) {
  const p = newPState(x, 12 * TILE)
  stepPlayer(p, 0, map)
  return p
}

describe('movement', () => {
  it('holding jump goes about four tiles up, a tap about one', () => {
    const map = flat()
    const p = settle(map)
    const y0 = p.y
    let top = 0
    for (let i = 0; i < 80; i++) {
      stepPlayer(p, BTN.J, map)
      top = Math.max(top, y0 - p.y)
    }
    expect(top).toBeGreaterThan(3.4 * TILE)
    expect(top).toBeLessThan(4.4 * TILE)

    const q = settle(map)
    let tap = 0
    stepPlayer(q, BTN.J, map)
    for (let i = 0; i < 60; i++) {
      stepPlayer(q, 0, map)
      tap = Math.max(tap, y0 - q.y)
    }
    expect(tap).toBeGreaterThan(0.8 * TILE)
    expect(tap).toBeLessThan(1.8 * TILE)
  })

  it('builds speed gradually and running jumps higher', () => {
    const map = flat(200)
    const p = settle(map)
    stepPlayer(p, BTN.R, map)
    expect(p.vx).toBeLessThan(0.5)
    for (let i = 0; i < 120; i++) stepPlayer(p, BTN.R | BTN.F, map)
    expect(p.vx).toBeCloseTo(PHYS.runMax, 1)
    const y0 = p.y
    let top = 0
    for (let i = 0; i < 80; i++) {
      stepPlayer(p, BTN.R | BTN.F | BTN.J, map)
      top = Math.max(top, y0 - p.y)
    }
    expect(top).toBeGreaterThan(4 * TILE)
  })

  it('allows a jump a few steps after running off a ledge (coyote time)', () => {
    // ground ends at column 10
    const cells: [number, number, string][] = []
    for (let x = 10; x < 60; x++) cells.push([x, 12, T.EMPTY], [x, 13, T.EMPTY])
    const map = flat(60, cells)
    const p = settle(map, 150)
    let steps = 0
    while (p.ground && steps < 200) {
      stepPlayer(p, BTN.R, map)
      steps++
    }
    expect(p.ground).toBe(false)
    stepPlayer(p, BTN.R, map)
    stepPlayer(p, BTN.R, map)
    stepPlayer(p, BTN.R | BTN.J, map)
    expect(p.vy).toBeLessThan(-3)
  })

  it('remembers a jump pressed just before landing (buffer)', () => {
    const map = flat()
    const p = newPState(100, 12 * TILE - 8)
    // falling; press jump while still in the air, keep holding
    let jumped = false
    for (let i = 0; i < 40; i++) {
      const r = stepPlayer(p, i >= 2 ? BTN.J : 0, map)
      if (r.jumped) jumped = true
    }
    expect(jumped).toBe(true)
  })

  it('slides round a block corner instead of bonking', () => {
    // a block right above, overlapping the head by 3 px
    const map = flat(60, [[7, 9, T.HARD]])
    const p = settle(map, 7 * TILE - 2)
    const r = stepPlayer(p, BTN.J, map)
    let bumped = !!r.bump
    for (let i = 0; i < 30; i++) bumped = bumped || !!stepPlayer(p, BTN.J, map).bump
    expect(bumped).toBe(false)
    expect(p.y).toBeLessThan(10 * TILE)
  })
})

describe('levels', () => {
  it('has eight levels in two worlds, each finishable in principle', () => {
    expect(LEVELS).toHaveLength(8)
    expect(LEVELS.filter((l) => l.world === 1)).toHaveLength(4)
    for (const lv of LEVELS) {
      for (const a of lv.areas) {
        expect(a.grid).toHaveLength(ROWS)
        for (const row of a.grid) expect(row.length).toBe(a.width)
      }
      const main = lv.areas[0]
      expect(lv.areas.some((a) => a.goal !== undefined) || !!main.boss).toBe(true)
      for (const area of lv.areas) {
        for (const wp of area.warps) {
          expect(lv.areas[wp.to]).toBeTruthy()
          expect(area.grid[wp.ty][wp.tx]).toBe(T.TUBE_TL)
          if (wp.exitTube) expect(lv.areas[wp.to].grid[wp.ay][wp.ax]).toBe(T.TUBE_TL)
        }
      }
      const map = { width: main.width, get: (x: number, y: number) => main.grid[y][x] }
      expect(isSolid(map.get(lv.start, standY(map, lv.start) / TILE))).toBe(true)
    }
  })

  it('every level can be finished (search bot, terrain only)', { timeout: 240000 }, () => {
    const results = LEVELS.map((_, i) => playLevel(i))
    const failed = results.map((r, i) => (r.done ? '' : `${LEVELS[i].id} stuck near x=${Math.round(r.best % 100000) / 16}`)).filter(Boolean)
    expect(failed).toEqual([])
  })
})

describe('world rules', () => {
  it('hitting a power box from below gives a 包子 to a small hero', () => {
    const w = new World({ players: 1 })
    while (w.phase === 'intro') w.tick([0])
    // put the hero under the 1-1 power box at column 46, row 8
    const p = w.players[0]
    w.cam = 30 * TILE
    p.ps.x = 46.5 * TILE
    p.ps.y = 12 * TILE
    for (let i = 0; i < 40; i++) w.tick([i < 20 ? BTN.J : 0])
    expect(w.area.get(46, 8)).toBe(T.USED)
    expect(w.ents.some((e) => e.k === 'bun')).toBe(true)
  })

  it('stomping a 泥丸子 squashes it and bounces the hero', () => {
    const w = new World({ players: 1 })
    while (w.phase === 'intro') w.tick([0])
    const p = w.players[0]
    w.cam = 20 * TILE
    for (let i = 0; i < 2; i++) w.tick([0])
    const mud = w.ents.find((e) => e.k === 'mud')
    expect(mud).toBeTruthy()
    if (!mud) return
    p.ps.x = mud.x
    p.ps.y = mud.y - 30
    p.ps.vy = 2
    let bounced = false
    for (let i = 0; i < 12; i++) {
      w.tick([0])
      if (p.ps.vy < -2) bounced = true
    }
    expect(bounced).toBe(true)
    expect(p.st).toBe('play')
  })

  it('a hero who falls behind the camera gets bubbled to the partner', () => {
    const w = new World({ players: 2 })
    while (w.phase === 'intro') w.tick([0, 0])
    const [a, b] = w.players
    a.ps.x = 30 * TILE
    b.ps.x = 2 * TILE
    // put a wall right of the trailing hero so the camera edge squeezes it
    w.area.set(3, 11, T.HARD)
    w.area.set(3, 10, T.HARD)
    for (let i = 0; i < 200; i++) w.tick([BTN.R, 0])
    expect(['bubble', 'play']).toContain(b.st)
    expect(b.ps.x).toBeGreaterThan(w.cam)
  })
})

describe('level flow', () => {
  function ready(players = 1) {
    const w = new World({ players })
    while (w.phase === 'intro') w.tick(Array(players).fill(0))
    return w
  }

  it('grabbing the pole clears the level and moves on to the next', () => {
    const w = ready()
    const goal = w.area.def.goal as number
    w.cam = (goal - 12) * TILE
    const p = w.players[0]
    p.ps.x = (goal - 2) * TILE
    p.ps.y = 6 * TILE
    let steps = 0
    while (w.levelIdx === 0 && steps < 3000) {
      w.tick([steps % 40 < 20 ? BTN.R | BTN.J : BTN.R])
      steps++
    }
    expect(w.levelIdx).toBe(1)
    expect(p.score).toBeGreaterThan(0)
  })

  it('pressing down on the marked tube takes the team to the coin room and back', () => {
    const w = ready(2)
    const warp = w.area.def.warps[0]
    w.cam = (warp.tx - 10) * TILE
    w.players.forEach((p) => {
      p.ps.x = (warp.tx + 1) * TILE
      p.ps.y = warp.ty * TILE
      p.ps.ground = true
    })
    for (let i = 0; i < 90; i++) w.tick([BTN.D, 0])
    expect(w.areaIdx).toBe(1)
    expect(w.players.every((p) => p.st === 'play')).toBe(true)
  })

  it('falling in a pit costs a life and restarts; no lives left is game over', () => {
    const w = new World({ players: 1, lives: 1 })
    while (w.phase === 'intro') w.tick([0])
    const p = w.players[0]
    p.ps.y = 300
    for (let i = 0; i < 400 && w.phase !== 'intro'; i++) w.tick([0])
    expect(w.lives).toBe(0)
    while (w.phase === 'intro') w.tick([0])
    w.players[0].ps.y = 300
    for (let i = 0; i < 400 && w.phase !== 'over'; i++) w.tick([0])
    expect(w.phase).toBe('over')
    w.continueGame()
    expect(w.lives).toBeGreaterThan(0)
    expect(w.phase).toBe('intro')
  })

  it('a partner who dies comes back in a bubble while the other keeps playing', () => {
    const w = ready(2)
    const lives = w.lives
    w.kill(w.players[1])
    for (let i = 0; i < 160; i++) w.tick([0, 0])
    expect(w.players[1].st).toBe('bubble')
    expect(w.lives).toBe(lives - 1)
    for (let i = 0; i < 400 && w.players[1].st === 'bubble'; i++) w.tick([0, 0])
    expect(w.players[1].st).toBe('play')
  })

  it('beating the 泥大王 drops the 年灯 and touching it clears the level', () => {
    const w = new World({ players: 1 })
    w.restartFrom(3)
    while (w.phase === 'intro') w.tick([0])
    w.cam = w.maxCam()
    w.players[0].ps.x = (w.area.width - 26) * TILE
    for (let i = 0; i < 5; i++) w.tick([0])
    const boss = w.ents.find((e) => e.k === 'mudking')
    expect(boss).toBeTruthy()
    for (let hit = 0; hit < 6 && boss && boss.hp > 0; hit++) {
      // drop onto its head
      const p = w.players[0]
      p.ps.x = boss.x
      p.ps.y = boss.y - boss.h - 8
      p.ps.vy = 3
      p.inv = 999
      for (let i = 0; i < 70; i++) w.tick([0])
    }
    expect(w.bossDown).toBe(true)
    for (let i = 0; i < 400 && !w.ents.some((e) => e.k === 'lamp'); i++) w.tick([0])
    const lamp = w.ents.find((e) => e.k === 'lamp')
    expect(lamp).toBeTruthy()
    if (!lamp) return
    for (let i = 0; i < 200; i++) w.tick([0])
    w.players[0].st = 'play'
    w.players[0].ps.x = lamp.x
    w.players[0].ps.y = lamp.y
    w.tick([0])
    expect(w.phase).toBe('clear')
  })
})

describe('network packing', () => {
  it('round-trips players and entities', () => {
    const w = new World({ players: 2 })
    const p = w.players[1]
    p.ps.x = 123.456
    p.ps.power = 2
    p.bell = 50
    p.st = 'bubble'
    const t = packPlayer(p)
    const q = new World({ players: 2 }).players[1]
    unpackPlayer(JSON.parse(JSON.stringify(t)), q)
    expect(q.ps.x).toBeCloseTo(123.46, 2)
    expect(q.ps.power).toBe(2)
    expect(q.bell).toBe(50)
    expect(q.st).toBe('bubble')
    const e = { ...w.ents[0], id: 7, k: 'hog', x: 10, y: 20, st: 3, t: 5, dir: 1, age: 9, vx: 0, vy: 0, ax: 0, ay: 0, len: 0, ground: true, hp: 1, w: 1, h: 1, dead: false, owner: -1, hitWall: false }
    const back = unpackEnt(JSON.parse(JSON.stringify(packEnt(e))))
    expect(back.k).toBe('hog')
    expect(back.st).toBe(3)
    expect(back.x).toBe(10)
  })
})

describe('music', () => {
  it('every melody fills its bars exactly', () => {
    for (const [id, s] of Object.entries(SONGS)) {
      const { bars } = parseMelody(s.melody)
      expect(bars, id).toHaveLength(s.chords.length)
      for (const b of bars) expect(b, id).toBe(16)
    }
  })
})
