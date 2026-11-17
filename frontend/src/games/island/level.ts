// The three areas of 椰风岛, built from small hand-designed chunks. Columns are 16px tiles; the ground
// height `g` counts tiles from the bottom of the 14-row screen (3 = surface at y 176).
import { ROWS, T, TS, TO, TW, TL, TB, IT_BOOM, IT_BOARD, IT_FAIRY, IT_CLOUD, IT_BASKET, IT_GOLD, type Kind, type Spawn, type Stage, type Theme } from './defs'

class Builder {
  readonly cols: number
  readonly tiles: Uint8Array
  readonly spawns: Spawn[] = []
  private readonly grounded: { s: Spawn; c: number }[] = []
  readonly cps: number[] = [3 * T]
  x = 0
  g = 3

  constructor(cols = 400) {
    this.cols = cols
    this.tiles = new Uint8Array(cols * ROWS)
  }

  setTile(c: number, r: number, v: number) {
    if (c >= 0 && c < this.cols && r >= 0 && r < ROWS) this.tiles[r * this.cols + c] = v
  }

  /** Solid ground of `n` columns at height h. */
  run(n: number, h = this.g): number {
    const s = this.x
    for (let c = s; c < s + n; c++) for (let r = ROWS - h; r < ROWS; r++) this.setTile(c, r, TS)
    this.g = h
    this.x += n
    return s
  }

  /** A pit. fill = water / lava strip at the bottom, or 0 for a bottomless drop. */
  gap(n: number, fill = 0): number {
    const s = this.x
    if (fill) for (let c = s; c < s + n; c++) this.setTile(c, ROWS - 1, fill)
    this.x += n
    return s
  }

  /** One-way platform `n` wide whose top is on row r. */
  plank(c: number, r: number, n: number) {
    for (let i = 0; i < n; i++) this.setTile(c + i, r, TO)
  }

  block(c: number, r: number, w: number, h: number) {
    for (let i = 0; i < w; i++) for (let j = 0; j < h; j++) this.setTile(c + i, r + j, TB)
  }

  surface(c: number): number {
    for (let r = 0; r < ROWS; r++) {
      const t = this.tiles[r * this.cols + c]
      if (t === TS || t === TB || t === TO) return r * T
    }
    return ROWS * T
  }

  /** Something standing on the ground at column c (x centred in the tile). */
  on(k: Kind, c: number, a?: number, b?: number) {
    // the ground may not be built yet (rollers wait further right), so y is resolved in finish()
    const sp: Spawn = { k, x: c * T + 8, y: 0, a, b }
    this.spawns.push(sp)
    this.grounded.push({ s: sp, c })
  }

  /** Something placed by its centre at column c, row r (rows from the top). */
  at(k: Kind, c: number, r: number, a?: number, b?: number) {
    this.spawns.push({ k, x: c * T + 8, y: r * T + 8, a, b })
  }

  /** A line or arc of fruit. */
  fruits(c: number, r: number, n: number, type: number, arc = 0) {
    for (let i = 0; i < n; i++) {
      const k = n > 1 ? i / (n - 1) : 0.5
      const lift = arc * Math.sin(k * Math.PI)
      this.spawns.push({ k: 'fruit', x: c * T + 8 + i * 18, y: r * T + 8 - lift, a: type })
    }
  }

  checkpoint() {
    this.cps.push(this.x * T + 8)
  }

  finish(id: number, name: string, theme: Theme, boss: 0 | 1 | 2): Stage {
    const arenaCol = this.x
    this.run(25, 3)
    for (const g of this.grounded) g.s.y = this.surface(g.c)
    // trim to the built width; arena walls are the stage edges and the camera locks at arenaX
    const cols = this.x
    const tiles = new Uint8Array(cols * ROWS)
    for (let r = 0; r < ROWS; r++) tiles.set(this.tiles.subarray(r * this.cols, r * this.cols + cols), r * cols)
    return { id, name, theme, cols, tiles, spawns: this.spawns, checkpoints: this.cps, arenaX: arenaCol * T, boss }
  }
}

// fruit types: 0 banana, 1 mango, 2 pineapple, 3 melon, 4 grapes, 5 golden coconut
function beach(): Stage {
  const b = new Builder()
  let s = b.run(16)
  b.fruits(s + 5, 9, 3, 0)
  b.on('clam', s + 11, IT_BOOM)
  s = b.run(10)
  b.on('rock', s + 3)
  b.on('crab', s + 9)
  b.fruits(s + 4, 8, 3, 1, 10)
  b.gap(2, TW)
  s = b.run(9)
  b.on('crab', s + 6)
  b.on('rock', s + 3)
  b.fruits(s + 1, 7, 4, 0, 12)
  s = b.run(3, 4)
  s = b.run(5, 5)
  b.fruits(s + 1, 6, 3, 2)
  b.at('gull', s + 12, 4)
  s = b.run(3, 4)
  s = b.gap(3, TW)
  b.at('fish', s + 1, 13)
  b.fruits(s, 7, 3, 1, 10)
  s = b.run(12, 3)
  b.on('fire', s + 5)
  b.on('crab', s + 9)
  b.at('gull', s + 16, 5)
  b.fruits(s + 3, 8, 4, 0, 14)

  b.checkpoint() // ① col ~60
  s = b.run(6)
  b.fruits(s + 1, 9, 2, 4)
  // pier over the lagoon
  s = b.gap(9, TW)
  b.plank(s + 1, 9, 3)
  b.plank(s + 5, 8, 3)
  b.at('fish', s + 4, 13)
  b.at('fish', s + 8, 13)
  b.fruits(s + 1, 7, 3, 1)
  b.fruits(s + 5, 6, 3, 3)
  s = b.run(8)
  b.on('clam', s + 3, IT_BOARD)
  b.on('rock', s + 7)
  // long beach sprint: perfect for the board
  s = b.run(34)
  for (const r of [3, 9, 15, 22, 28]) b.on('rock', s + r)
  b.on('crab', s + 12)
  b.on('crab', s + 25)
  b.at('gull', s + 18, 4)
  b.at('gull', s + 31, 5)
  b.fruits(s + 5, 8, 5, 0, 8)
  b.fruits(s + 17, 9, 3, 1)
  b.on('spring', s + 32)
  b.fruits(s + 33, 3, 4, 3)
  s = b.gap(3, TW)
  b.at('fish', s + 1, 13)
  s = b.run(10, 4)
  b.on('fire', s + 4)
  b.fruits(s + 2, 8, 3, 2, 10)

  b.checkpoint() // ② col ~131
  s = b.run(22, 3)
  b.on('spiky', s + 20)
  b.on('spiky', s + 30)
  b.on('rock', s + 8)
  b.on('clam', s + 4, IT_FAIRY)
  b.fruits(s + 9, 8, 4, 4, 10)
  b.on('crab', s + 15)
  b.gap(2, TW)
  s = b.run(16)
  b.on('crab', s + 4)
  b.on('crab', s + 11)
  b.at('gull', s + 9, 4)
  b.at('gull', s + 14, 6)
  b.on('clam', s + 7, IT_CLOUD)
  b.fruits(s + 6, 6, 3, 5)
  s = b.run(10)
  b.on('fire', s + 3)
  b.on('fire', s + 7)
  b.fruits(s + 4, 7, 2, 1)
  s = b.gap(5, TW)
  b.plank(s + 1, 9, 3)
  b.at('fish', s + 2, 13)
  s = b.run(12)
  b.on('clam', s + 4, IT_BASKET)
  b.fruits(s + 6, 9, 3, 0)
  return b.finish(0, '浪花海滩', 'beach', 0)
}

function jungle(): Stage {
  const b = new Builder()
  let s = b.run(14)
  b.on('clam', s + 7, IT_BOOM)
  b.fruits(s + 2, 9, 3, 1)
  s = b.run(8)
  b.on('frog', s + 6)
  b.on('rock', s + 3)
  s = b.gap(3)
  b.fruits(s, 8, 3, 0, 10)
  s = b.run(10)
  b.at('spider', s + 5, 2, 90)
  b.on('rock', s + 8)
  b.fruits(s + 2, 9, 3, 4)
  // stacked logs over a ravine
  s = b.gap(10)
  b.plank(s + 1, 10, 3)
  b.plank(s + 5, 8, 3)
  b.plank(s + 8, 10, 2)
  b.fruits(s + 5, 6, 3, 2)
  b.at('spider', s + 6, 1, 70)
  s = b.run(12, 4)
  b.on('frog', s + 5)
  b.on('frog', s + 10)
  b.on('fire', s + 8)
  b.fruits(s + 1, 7, 4, 3, 10)

  b.checkpoint() // ①
  s = b.run(6, 3)
  s = b.gap(12)
  b.at('plat', s + 2, 9, 0, 56) // horizontal log raft
  b.at('plat', s + 8, 8, 1, 32)
  b.fruits(s + 3, 6, 4, 1, 12)
  s = b.run(8, 3)
  b.on('clam', s + 3, IT_BOARD)
  s = b.run(30, 3)
  for (const r of [4, 11, 17, 24]) b.on('rock', s + r)
  b.on('spiky', s + 26)
  b.on('spiky', s + 36)
  b.on('frog', s + 14)
  b.at('spider', s + 20, 2, 96)
  b.fruits(s + 6, 8, 4, 0, 10)
  b.fruits(s + 19, 9, 3, 4)
  b.on('spring', s + 28)
  b.fruits(s + 29, 2, 4, 3)
  s = b.gap(3)
  s = b.run(4, 5)
  s = b.run(4, 7)
  b.fruits(s + 1, 5, 2, 2)
  s = b.gap(4)
  b.plank(s + 1, 8, 2)
  s = b.run(8, 4)
  b.on('fire', s + 3)
  b.on('frog', s + 6)

  b.checkpoint() // ②
  s = b.run(10, 3)
  b.on('clam', s + 3, IT_GOLD)
  b.on('fire', s + 8)
  s = b.gap(14)
  b.plank(s + 1, 10, 2)
  b.at('plat', s + 5, 9, 1, 40)
  b.plank(s + 9, 8, 2)
  b.plank(s + 12, 10, 2)
  b.at('spider', s + 9, 1, 60)
  b.fruits(s + 4, 5, 3, 5)
  s = b.run(18, 3)
  b.on('frog', s + 4)
  b.on('frog', s + 9)
  b.on('rock', s + 6)
  b.on('fire', s + 13)
  b.on('clam', s + 11, IT_FAIRY)
  b.at('spider', s + 15, 2, 90)
  b.fruits(s + 2, 8, 4, 1, 10)
  s = b.run(10, 3)
  b.on('clam', s + 4, IT_CLOUD)
  b.fruits(s + 3, 6, 3, 2)
  b.on('spiky', s + 16)
  s = b.gap(2)
  s = b.run(12, 3)
  b.on('clam', s + 6, IT_BASKET)
  return b.finish(1, '雨林深处', 'jungle', 1)
}

function volcano(): Stage {
  const b = new Builder()
  let s = b.run(14)
  b.on('clam', s + 7, IT_BOOM)
  b.fruits(s + 3, 9, 3, 4)
  s = b.run(8)
  b.at('bat', s + 6, 3)
  b.on('rock', s + 3)
  s = b.gap(4, TL)
  b.at('geyser', s + 2, 13)
  b.fruits(s, 7, 3, 1, 12)
  s = b.run(10)
  b.on('flame', s + 5)
  b.on('fire', s + 8)
  b.fruits(s + 1, 8, 3, 0)
  s = b.gap(10, TL)
  b.at('plat', s + 3, 9, 0, 48)
  b.at('geyser', s + 7, 13)
  b.fruits(s + 2, 6, 4, 3, 8)
  s = b.run(12, 4)
  b.on('boulder', s + 24)
  b.at('bat', s + 9, 4)
  b.on('rock', s + 5)
  b.fruits(s + 3, 7, 4, 2, 10)

  b.checkpoint() // ①
  s = b.run(6, 3)
  s = b.gap(12, TL)
  b.at('plat', s + 1, 10, 2, 0) // crumbling slabs
  b.at('plat', s + 5, 9, 2, 0)
  b.at('plat', s + 9, 10, 2, 0)
  b.at('geyser', s + 4, 13)
  b.fruits(s + 5, 6, 3, 1)
  s = b.run(8, 3)
  b.on('clam', s + 4, IT_BOARD)
  s = b.run(30, 3)
  b.on('boulder', s + 30)
  b.on('boulder', s + 44)
  for (const r of [5, 13, 21]) b.on('rock', s + r)
  b.on('flame', s + 10)
  b.at('bat', s + 18, 3)
  b.at('bat', s + 26, 4)
  b.fruits(s + 6, 8, 4, 0, 10)
  b.fruits(s + 22, 9, 3, 4)
  b.on('spring', s + 28)
  b.fruits(s + 28, 3, 4, 3)
  s = b.gap(3, TL)
  s = b.run(10, 4)
  b.on('fire', s + 3)
  b.on('flame', s + 7)

  b.checkpoint() // ②
  s = b.run(8, 3)
  b.on('clam', s + 3, IT_GOLD)
  b.fruits(s + 5, 9, 2, 1)
  s = b.gap(14, TL)
  b.at('plat', s + 2, 9, 1, 36)
  b.at('plat', s + 6, 10, 2, 0)
  b.at('plat', s + 10, 8, 0, 32)
  b.at('geyser', s + 5, 13)
  b.at('geyser', s + 9, 13)
  b.at('bat', s + 9, 3)
  b.fruits(s + 3, 5, 4, 5, 6)
  s = b.run(16, 3)
  b.on('clam', s + 3, IT_FAIRY)
  b.on('flame', s + 8)
  b.on('flame', s + 12)
  b.on('boulder', s + 24)
  b.fruits(s + 5, 8, 4, 2, 10)
  s = b.run(10, 3)
  b.on('clam', s + 3, IT_CLOUD)
  b.on('fire', s + 7)
  b.fruits(s + 2, 6, 3, 3)
  s = b.gap(3, TL)
  s = b.run(12, 3)
  b.on('clam', s + 6, IT_BASKET)
  b.fruits(s + 2, 9, 3, 0)
  return b.finish(2, '熔岩火山', 'volcano', 2)
}

let cache: Stage[] | null = null
export function stages(): Stage[] {
  if (!cache) cache = [beach(), jungle(), volcano()]
  return cache
}

export const STAGE_COUNT = 3
