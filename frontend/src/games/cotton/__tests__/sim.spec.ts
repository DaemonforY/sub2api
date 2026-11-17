import { describe, expect, it } from 'vitest'
import { FLOORS } from '../levels'
import { blend, decodeEvents, encode, guestState } from '../net'
import { ALIVE, createGame, fire, NO_BUTTONS, SNOW_FULL, step, tileAt, type Buttons, type GameState } from '../sim'

function run(s: GameState, frames: number, a: Buttons = NO_BUTTONS, b: Buttons = NO_BUTTONS): void {
  for (let i = 0; i < frames; i++) step(s, [a, b])
}

describe('levels', () => {
  it('keeps platforms inside the walls and every pest on something to stand on', () => {
    FLOORS.forEach((f, i) => {
      for (const [r, a, b] of f.plats) {
        expect(a, `floor ${i + 1}`).toBeGreaterThanOrEqual(1)
        expect(b, `floor ${i + 1}`).toBeLessThanOrEqual(38)
        expect(r).toBeGreaterThanOrEqual(4)
        expect(r).toBeLessThanOrEqual(27)
      }
      const s = createGame(1, 1, i)
      for (const [, col, row] of f.enemies) {
        const t = tileAt(s, col, row)
        expect(t, `floor ${i + 1} pest at ${col},${row}`).toBeGreaterThan(0)
        expect(tileAt(s, col, row - 1), `floor ${i + 1} pest at ${col},${row} is buried`).toBe(0)
      }
      expect(f.enemies.length > 0 || !!f.boss).toBe(true)
    })
  })
})

describe('candy floss', () => {
  it('wraps a pest into a ball, and a kicked ball flattens the pests in its way', () => {
    const s = createGame(1, 7, 0)
    s.enemies = []
    run(s, 100) // intro
    const p = s.players[0]
    // three books in a row on the ground, to the right of the kid
    s.enemies = [0, 1, 2].map((i) => {
      const e = { ...createGame(1, 1, 0).enemies[0] }
      e.id = 100 + i
      e.x = p.x + 30 + i * 30
      e.y = 224 - e.h
      e.spawnT = 0
      e.cd = 9999
      e.angry = false
      return e
    })
    s.enemies.forEach((e) => (e.vx = 0))
    p.face = 1
    const first = s.enemies[0]
    for (let i = 0; i < 6 && first.snow < SNOW_FULL; i++) {
      p.cd = 0
      fire(s, p)
      run(s, 12)
    }
    expect(first.snow).toBe(SNOW_FULL)
    // walk up to it and kick
    for (let i = 0; i < 60 && first.x - (p.x + p.w) > 2; i++) step(s, [{ ...NO_BUTTONS, r: true }, NO_BUTTONS])
    p.cd = 0
    p.face = 1
    fire(s, p)
    expect(first.rolling).toBe(true)
    const before = p.score
    run(s, 200)
    const kills = s.events.filter((e) => e.k === 'kill' || e.k === 'burst')
    expect(kills.length).toBeGreaterThanOrEqual(2)
    expect(p.score - before).toBeGreaterThanOrEqual(400 + 800)
    expect(p.bestChain).toBeGreaterThanOrEqual(2)
  })

  it('lets a partner touch a fallen kid back to life', () => {
    const s = createGame(2, 3, 0)
    run(s, 100)
    const [a, b] = s.players
    s.enemies = []
    a.inv = 0
    // drop a pest on kid 1
    const pest = createGame(1, 1, 0).enemies[0]
    pest.spawnT = 0
    pest.x = a.x
    pest.y = a.y
    s.enemies.push(pest)
    s.phase = 'play'
    step(s, [NO_BUTTONS, NO_BUTTONS])
    expect(a.state).not.toBe(ALIVE)
    s.enemies = [{ ...pest, x: 300, y: 40, snow: 0 }]
    b.x = a.x
    b.y = a.y
    b.inv = 50
    step(s, [NO_BUTTONS, NO_BUTTONS])
    expect(a.state).toBe(ALIVE)
  })
})

describe('a long scrappy game', () => {
  it('runs every floor with a button-mashing bot without breaking', () => {
    for (let floor = 0; floor < FLOORS.length; floor++) {
      const s = createGame(2, 1000 + floor, floor)
      let seed = floor + 1
      const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647)
      let a: Buttons = { ...NO_BUTTONS }
      let b: Buttons = { ...NO_BUTTONS }
      for (let i = 0; i < 3600; i++) {
        if (i % 20 === 0) {
          a = { l: rnd() < 0.4, r: rnd() < 0.4, u: false, d: rnd() < 0.1, jump: rnd() < 0.3, fire: rnd() < 0.7 }
          b = { l: rnd() < 0.4, r: rnd() < 0.4, u: false, d: rnd() < 0.1, jump: rnd() < 0.3, fire: rnd() < 0.7 }
        }
        step(s, [a, b])
        s.events = []
        for (const p of s.players) {
          expect(Number.isFinite(p.x) && Number.isFinite(p.y)).toBe(true)
          expect(p.x).toBeGreaterThanOrEqual(7)
          expect(p.x).toBeLessThanOrEqual(312)
        }
        for (const e of s.enemies) expect(Number.isFinite(e.x) && Number.isFinite(e.y)).toBe(true)
      }
      // the snapshot stays well under the relay's 8 KB limit
      expect(JSON.stringify(encode(s, [])).length).toBeLessThan(6000)
    }
  })
})

describe('snapshots', () => {
  it('round-trips the state the guest draws', () => {
    const s = createGame(2, 5, 3)
    run(s, 150)
    const a = encode(s, [])
    run(s, 3)
    const b = encode(s, [{ k: 'kill', x: 10, y: 20, n: 800, p: 2 }])
    const v = guestState()
    blend(v, a, b, 1)
    expect(v.floor).toBe(3)
    expect(v.enemies.length).toBe(s.enemies.length)
    expect(v.enemies[0].x).toBeCloseTo(s.enemies[0].x, 0)
    expect(v.players[1].score).toBe(s.players[1].score)
    expect(decodeEvents(b)).toEqual([{ k: 'kill', x: 10, y: 20, n: 800, p: 2 }])
  })
})

describe('bosses', () => {
  it('can each be beaten by two kids rolling balls into them', async () => {
    const { botButtons } = await import('../bot')
    for (const floor of [4, 9, 14]) {
      const s = createGame(2, 77 + floor, floor)
      let died = -1
      for (let i = 0; i < 60 * 420 && died < 0; i++) {
        for (const p of s.players) p.inv = Math.max(p.inv, 2)
        step(s, [botButtons(s, 0), botButtons(s, 1)])
        if (s.events.some((e) => e.k === 'bossdie')) died = i
        s.events = []
      }
      expect(died, `boss on floor ${floor + 1}`).toBeGreaterThan(0)
    }
  })

  it('keep pests on their shelves until they walk off', () => {
    const s = createGame(1, 9, 0)
    run(s, 130)
    for (const e of s.enemies) expect(e.y + e.h).toBeLessThan(200)
  })
})
