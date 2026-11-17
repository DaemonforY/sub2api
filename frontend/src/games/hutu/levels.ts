// The eight levels of 虎兔大冒险, two worlds of four. Every layout here is our own: each level is
// built in code from small pieces (ground runs, terraces, tubes, brick shapes) so it stays readable
// and easy to retune. Rows go 0 (top) to 13 (bottom); the default ground top is row 12.
//
// 第一世界「竹林村」: 竹林小径 → 萤火溶洞 → 竹梢云桥 → 泥王寨
// 第二世界「灯火古镇」: 古镇屋檐 → 运河夜市 → 烟花塔 → 年兽宫

import type { Area, EnemyKind, Level, Mover, MusicId, Spawn, Theme, Warp } from './types'
import { ROWS, T } from './types'

class LB {
  readonly g: string[][]
  readonly spawns: Spawn[] = []
  readonly movers: Mover[] = []
  readonly warps: Warp[] = []
  goal?: number
  boss?: boolean
  checkpoint?: number

  constructor(
    readonly width: number,
    readonly theme: Theme
  ) {
    this.g = Array.from({ length: ROWS }, () => Array.from({ length: width }, () => T.EMPTY as string))
  }

  set(x: number, y: number, c: string): this {
    if (x >= 0 && x < this.width && y >= 0 && y < ROWS) this.g[y][x] = c
    return this
  }

  fill(x0: number, y0: number, x1: number, y1: number, c: string): this {
    for (let y = y0; y <= y1; y++) for (let x = x0; x <= x1; x++) this.set(x, y, c)
    return this
  }

  /** solid ground from column x0 to x1 (inclusive), top surface at row `top` */
  ground(x0: number, x1: number, top = 12, c: string = T.GROUND): this {
    return this.fill(x0, top, x1, ROWS - 1, c)
  }

  /** a floating slab one row thick (tree canopies, ledges) */
  slab(x0: number, x1: number, y: number, c: string = T.GROUND): this {
    return this.fill(x0, y, x1, y, c)
  }

  clear(x0: number, x1: number, y0 = 0, y1 = ROWS - 1): this {
    return this.fill(x0, y0, x1, y1, T.EMPTY)
  }

  /** a string of tiles laid left to right; spaces leave what is there */
  row(x: number, y: number, s: string): this {
    for (let i = 0; i < s.length; i++) if (s[i] !== ' ') this.set(x + i, y, s[i])
    return this
  }

  /** several rows at once, top row first */
  shape(x: number, y: number, rows: string[]): this {
    rows.forEach((r, i) => this.row(x, y + i, r))
    return this
  }

  /** a two-wide bamboo tube with its top at row `top`, reaching down to row `bottom` */
  tube(x: number, top: number, bottom = 11): this {
    this.set(x, top, T.TUBE_TL).set(x + 1, top, T.TUBE_TR)
    for (let y = top + 1; y <= bottom; y++) this.set(x, y, T.TUBE_L).set(x + 1, y, T.TUBE_R)
    return this
  }

  /** a staircase of stone blocks, `n` steps, rising to the right (dir 1) or left (dir -1), standing on row base+1 */
  stairs(x: number, n: number, dir = 1, base = 11, c: string = T.HARD): this {
    for (let i = 0; i < n; i++) {
      const h = dir > 0 ? i + 1 : n - i
      this.fill(x + i, base - h + 1, x + i, base, c)
    }
    return this
  }

  coins(x: number, y: number, n: number): this {
    for (let i = 0; i < n; i++) this.set(x + i, y, T.COIN)
    return this
  }

  e(kind: EnemyKind, tx: number, ty = 11, extra: Partial<Spawn> = {}): this {
    this.spawns.push({ kind, tx, ty, ...extra })
    return this
  }

  mover(tx: number, ty: number, w: number, dx: number, dy: number, period: number, phase = 0): this {
    this.movers.push({ tx, ty, w, dx, dy, period, phase })
    return this
  }

  warp(w: Warp): this {
    this.warps.push(w)
    return this
  }

  build(): Area {
    return {
      grid: this.g.map((r) => r.join('')),
      width: this.width,
      theme: this.theme,
      spawns: this.spawns,
      movers: this.movers,
      warps: this.warps,
      goal: this.goal,
      boss: this.boss,
      checkpoint: this.checkpoint
    }
  }
}

/** a little underground room full of coins, reached through a tube */
function coinRoom(theme: Theme, backTo: { ax: number; ay: number }, layout: 'stack' | 'arch'): Area {
  const b = new LB(25, theme)
  b.ground(0, 24).fill(0, 0, 0, 11, T.WALL).fill(24, 0, 24, 11, T.WALL).fill(1, 0, 23, 1, T.WALL)
  if (layout === 'stack') {
    b.fill(5, 9, 17, 11, T.HARD)
    b.coins(5, 8, 13).coins(6, 6, 11).coins(7, 4, 9)
    b.row(9, 4, 'oMo')
  } else {
    b.shape(3, 4, ['oooo   oooo   oooo', ' oo     oo     oo ', '                  ', 'BBBB   BBBB   BBBB'])
    b.coins(4, 9, 15)
  }
  // the way back: a sideways tube in the right wall
  b.tube(20, 9, 11).set(20, 9, T.TUBE_TL).set(21, 9, T.TUBE_TR)
  b.warp({ tx: 20, ty: 9, to: 0, ax: backTo.ax, ay: backTo.ay, exitTube: true })
  return b.build()
}

// ───────────────────────────── 1-1 竹林小径 ─────────────────────────────
// Teaches one thing at a time: a box low enough to find by accident, a terrace to hop, the first
// 泥丸子 on open ground, a plank shelf, the first 包子, tubes with a 竹鼠, a "pagoda" of bricks
// with a 刺团 walking on its roof, rice terraces with 泥丸子 tumbling down them, a tube grove
// hiding a coin room, and a stone pyramid before the goal.
function lv11(): Level {
  const b = new LB(214, 'bamboo')
  b.ground(0, 63)
  b.set(11, 9, T.BOX)
  b.ground(16, 19, 11).ground(20, 23, 10).ground(24, 26, 11)
  b.coins(20, 7, 4)
  b.e('mud', 33)
  b.slab(36, 41, 8, T.PLANK).coins(36, 6, 6)
  b.row(45, 8, 'BPB')
  b.e('mud', 49).e('mud', 51)
  b.tube(55, 10)
  b.tube(60, 9).e('rat', 60, 9)
  // a stream: two steps across, coins in an arc over it
  b.ground(66, 128)
  b.set(63, 9, T.COIN).set(64, 8, T.COIN).set(65, 9, T.COIN)
  // pagoda of bricks
  b.shape(72, 4, ['  B  ', ' BMB ', '     ', '     ', 'BB?BB'])
  b.e('hog', 74, 7)
  b.e('mud', 82).e('mud', 84)
  b.row(88, 8, '? ?')
  b.checkpoint = 92
  // rice terraces climbing to the right; 泥丸子 walk down them
  b.ground(96, 99, 11).ground(100, 103, 10).ground(104, 107, 9).ground(108, 117, 8)
  b.e('mud', 106, 8).e('mud', 111, 7).e('mud', 115, 7)
  b.coins(109, 5, 3).set(114, 4, T.BOX_LIFE)
  b.clear(118, 120)
  b.ground(121, 160)
  b.row(124, 8, 'BQB')
  b.e('hog', 127)
  // tube grove
  b.tube(133, 10)
  b.tube(138, 9)
  b.warp({ tx: 138, ty: 9, to: 1, ax: 2, ay: 11 })
  b.tube(143, 10).e('rat', 143, 10)
  b.tube(148, 8)
  b.coins(135, 7, 2).coins(140, 6, 2).coins(145, 6, 2)
  b.row(152, 8, 'BBBNBBB').coins(152, 4, 7)
  b.e('kite', 156, 5)
  b.clear(161, 162)
  b.ground(163, 213)
  b.tube(166, 10)
  b.e('mud', 172).e('mud', 174).e('hog', 178)
  b.row(170, 8, '?B?')
  b.stairs(183, 4).stairs(187, 4, -1)
  b.stairs(193, 5)
  b.clear(198, 199)
  b.set(204, 11, T.HARD)
  b.goal = 204
  return {
    id: '1-1',
    world: 1,
    name: '竹林小径',
    music: 'field',
    time: 300,
    start: 3,
    areas: [b.build(), coinRoom('cave', { ax: 166, ay: 10 }, 'stack')]
  }
}

// ───────────────────────────── 1-2 萤火溶洞 ─────────────────────────────
// A cave under the bamboo. Boxes laid out as steps, a crawlway with a low ceiling, a mud pool with
// 跳跳鱼, tubes with 竹鼠, a brick maze, lifts over a chasm. Secret: break into the brick ceiling
// near the start and run along the roof past rows of coins, skipping half the cave.
function lv12(): Level {
  const b = new LB(184, 'cave')
  b.ground(0, 183).fill(0, 0, 183, 1, T.WALL)
  b.fill(0, 2, 0, 11, T.WALL)
  // stepping boxes
  b.set(14, 9, T.BOX).set(17, 7, T.BOX_POWER).set(20, 9, T.BOX)
  b.e('mud', 24).e('mud', 26)
  // crawlway: ceiling comes down to row 7
  b.fill(28, 2, 36, 7, T.WALL)
  b.coins(29, 10, 8)
  b.e('mud', 32).e('mud', 35)
  // a stone stair up to a gap in the ceiling: the secret path runs on top of the brick roof
  b.stairs(39, 7)
  b.clear(44, 126, 0, 1)
  b.fill(46, 2, 125, 2, T.BRICK)
  b.set(48, 2, T.BRICK_POWER)
  b.coins(56, 1, 10).coins(76, 1, 10).coins(96, 1, 10).coins(112, 1, 8)
  b.fill(120, 2, 125, 2, T.HARD)
  b.fill(127, 0, 127, 2, T.WALL)
  // mud pool with stepping stones
  b.fill(58, 12, 67, 13, T.LAVA)
  b.fill(61, 10, 61, 11, T.HARD).fill(64, 10, 64, 11, T.HARD)
  b.e('fish', 59, 13).e('fish', 66, 13)
  b.row(56, 7, '?').row(69, 7, '?')
  // tubes with 竹鼠
  b.tube(74, 9).e('rat', 74, 9)
  b.tube(80, 8).e('rat', 80, 8)
  b.tube(86, 10)
  b.e('hog', 83)
  b.checkpoint = 94
  // brick maze
  b.shape(98, 5, ['BBBBMBBB', '        ', '        ', '        ', 'BBQBBBBB'])
  b.e('hog', 101, 8).e('mud', 104, 4).e('mud', 109)
  b.fill(108, 6, 108, 8, T.HARD)
  b.row(110, 7, 'B?B')
  // chasm with two lifts
  b.clear(130, 140, 2, 13)
  b.mover(131, 11, 3, 0, -6, 300)
  b.mover(136, 5, 3, 0, 6, 300)
  b.e('mud', 146).e('mud', 148).e('mud', 150)
  b.row(146, 8, 'BBH?B')
  b.e('hog', 158)
  b.stairs(160, 3)
  b.coins(164, 7, 3)
  b.tube(170, 9)
  b.warp({ tx: 170, ty: 9, to: 1, ax: 2, ay: 10, exitTube: true })
  b.fill(176, 2, 183, 11, T.WALL)

  // outside again: a short walk to the goal
  const o = new LB(40, 'bamboo')
  o.ground(0, 39)
  o.tube(2, 10)
  o.stairs(10, 4)
  o.coins(15, 6, 3)
  o.set(26, 11, T.HARD)
  o.goal = 26
  return { id: '1-2', world: 1, name: '萤火溶洞', music: 'cave', time: 300, start: 3, areas: [b.build(), o.build()] }
}

// ───────────────────────────── 1-3 竹梢云桥 ─────────────────────────────
// High in the bamboo tops: islands of canopy over a long drop, bamboo rafts that drift, kites in
// the wind. Islands shift up and down so each jump reads differently.
function lv13(): Level {
  const b = new LB(196, 'treetop')
  b.ground(0, 13)
  b.slab(16, 20, 10)
  b.slab(23, 28, 8).set(25, 4, T.BOX)
  b.slab(31, 34, 11).e('hog', 33, 10)
  b.slab(37, 44, 7).set(40, 3, T.BOX_POWER).e('mud', 43, 6)
  b.mover(47, 9, 3, 5, 0, 260)
  b.slab(57, 62, 9).e('kite', 61, 5)
  b.slab(65, 67, 6).coins(65, 4, 3)
  b.slab(70, 76, 10).e('mud', 72, 9).e('mud', 74, 9)
  b.mover(79, 10, 3, 0, -4, 240)
  b.slab(84, 90, 5).coins(84, 3, 7)
  b.slab(93, 97, 9)
  b.checkpoint = 95
  b.slab(100, 106, 11).row(101, 7, 'BNB?')
  b.e('kite', 106, 4)
  b.mover(109, 9, 3, 4, 0, 220)
  b.mover(118, 7, 3, 4, 0, 220, 110)
  b.slab(128, 134, 8).e('hog', 130, 7).e('kite', 134, 3)
  b.slab(137, 139, 6)
  b.slab(142, 144, 9).set(143, 5, T.BOX_LIFE)
  b.slab(147, 153, 7).e('kite', 152, 3).e('mud', 150, 6)
  b.mover(156, 9, 2, 0, 3, 160)
  b.slab(160, 165, 10).coins(160, 8, 6)
  b.ground(168, 195)
  b.stairs(172, 4)
  b.set(185, 11, T.HARD)
  b.goal = 185
  return { id: '1-3', world: 1, name: '竹梢云桥', music: 'sky', time: 300, start: 3, areas: [b.build()] }
}

// ───────────────────────────── 1-4 泥王寨 ─────────────────────────────
// The mud king's stockade: low ceilings, boiling mud with 火星子 leaping out, turning 火链, rafts
// over a long pool, and the 泥大王 waiting on a bridge at the end.
function lv14(): Level {
  const b = new LB(132, 'fort')
  b.ground(0, 131).fill(0, 0, 131, 2, T.WALL).fill(0, 3, 0, 11, T.WALL)
  b.ground(1, 6, 9).fill(1, 3, 12, 4, T.WALL)
  b.set(16, 8, T.USED).e('firebar', 16, 8, { len: 5, dir: 1 })
  // pool with a pillar
  b.fill(21, 12, 28, 13, T.LAVA).fill(24, 9, 25, 11, T.HARD)
  b.e('spark', 22, 13).e('spark', 27, 13)
  // raised hall with a firebar on its floor
  b.ground(31, 40, 9).fill(31, 3, 40, 4, T.WALL)
  b.set(36, 9, T.USED).e('firebar', 36, 9, { len: 4, dir: -1 })
  b.set(43, 7, T.BOX_POWER)
  b.e('mud', 44).e('mud', 46)
  // long pool with rafts
  b.fill(48, 12, 61, 13, T.LAVA)
  b.mover(49, 10, 3, 4, 0, 200).mover(55, 8, 3, 3, 0, 180, 90)
  b.e('spark', 52, 13).e('spark', 59, 13)
  b.checkpoint = 65
  b.set(70, 6, T.USED).e('firebar', 70, 6, { len: 5, dir: 1 })
  b.set(75, 10, T.USED).e('firebar', 75, 10, { len: 4, dir: -1 })
  b.fill(68, 3, 78, 3, T.WALL)
  b.stairs(80, 3)
  b.fill(83, 12, 87, 13, T.LAVA).e('spark', 85, 13)
  b.stairs(88, 3, -1)
  b.row(93, 8, '?P?')
  b.e('hog', 96)
  // boss arena: bridge over mud
  b.fill(100, 3, 101, 8, T.WALL)
  b.clear(104, 125, 12, 13)
  b.fill(104, 13, 125, 13, T.LAVA)
  b.fill(104, 12, 125, 12, T.BRIDGE)
  b.ground(126, 131, 10)
  b.e('mudking', 121, 11)
  b.boss = true
  return { id: '1-4', world: 1, name: '泥王寨', music: 'fort', time: 300, start: 3, areas: [b.build()] }
}

// ───────────────────────────── 2-1 古镇屋檐 ─────────────────────────────
// Dusk over the old town. Run the lanes, climb onto the tiled roofs, jump the canals between
// houses. 炮仗精 first appear here: stomp one and get clear before it bursts.
function lv21(): Level {
  const b = new LB(206, 'town')
  b.ground(0, 205)
  b.row(11, 8, '?').row(14, 8, 'BPB')
  b.e('mud', 20)
  // first houses
  b.fill(24, 9, 30, 11, T.WALL).coins(25, 7, 5)
  b.e('cracker', 34)
  b.set(36, 11, T.HARD).fill(38, 8, 44, 11, T.WALL).e('mud', 41, 7)
  b.fill(46, 12, 48, 13, T.LAVA)
  b.fill(50, 9, 56, 11, T.WALL)
  b.row(52, 5, 'BMB')
  b.e('cracker', 58).e('mud', 61)
  b.tube(64, 9).e('rat', 64, 9)
  // roof run: three houses with canals between
  b.fill(70, 8, 77, 11, T.WALL).e('hog', 74, 7)
  b.fill(78, 12, 80, 13, T.LAVA)
  b.fill(81, 6, 86, 11, T.WALL).row(82, 2, '?N?')
  b.fill(87, 12, 89, 13, T.LAVA)
  b.fill(90, 8, 97, 11, T.WALL).e('cracker', 94, 7)
  b.checkpoint = 102
  b.e('mud', 106).e('mud', 108).e('mud', 110)
  b.row(105, 8, 'B?BB?B')
  b.slab(112, 116, 5, T.PLANK).coins(112, 4, 5)
  b.e('cracker', 118)
  b.fill(121, 12, 124, 13, T.LAVA)
  b.fill(126, 9, 131, 11, T.WALL).set(128, 5, T.BOX_LIFE)
  b.fill(132, 12, 134, 13, T.LAVA)
  b.fill(135, 7, 140, 11, T.WALL).e('hog', 138, 6)
  b.tube(144, 9)
  b.warp({ tx: 144, ty: 9, to: 1, ax: 2, ay: 11 })
  b.e('cracker', 150).e('cracker', 153)
  b.row(150, 8, 'BBBB')
  b.tube(158, 10)
  b.e('kite', 164, 4)
  b.fill(166, 12, 168, 13, T.LAVA)
  b.stairs(171, 5)
  b.fill(176, 12, 177, 13, T.LAVA)
  b.stairs(178, 3, -1)
  b.set(194, 11, T.HARD)
  b.goal = 194
  return {
    id: '2-1',
    world: 2,
    name: '古镇屋檐',
    music: 'town',
    time: 300,
    start: 3,
    areas: [b.build(), coinRoom('cave', { ax: 158, ay: 10 }, 'arch')]
  }
}

// ───────────────────────────── 2-2 运河夜市 ─────────────────────────────
// Night market along the canal. Most of the floor is water: ride the boats, hop stone bridges and
// market awnings, mind the 跳跳鱼.
function lv22(): Level {
  const b = new LB(200, 'canal')
  b.ground(0, 11)
  b.fill(12, 12, 199, 13, T.LAVA)
  b.row(8, 8, 'P')
  // stone bridge (an arch of blocks)
  b.shape(14, 8, ['  XXXX  ', ' X    X ', 'X      X', 'X      X'])
  b.e('fish', 13, 13).e('fish', 21, 13)
  b.coins(16, 6, 4)
  b.mover(24, 10, 3, 6, 0, 260)
  b.ground(34, 41, 11).e('mud', 37, 10).e('mud', 39, 10)
  b.slab(35, 40, 7, T.PLANK).row(36, 3, '?B?')
  b.mover(44, 10, 3, 5, 0, 240).e('fish', 47, 13)
  b.ground(54, 58, 10)
  b.e('fish', 61, 13)
  b.ground(64, 70, 11).e('cracker', 67, 10)
  b.shape(72, 7, ['   XXXX   ', '  X    X  ', ' X      X ', 'X        X', 'X        X'])
  b.e('hog', 76, 6).coins(75, 9, 4)
  b.e('fish', 74, 13).e('fish', 79, 13)
  b.checkpoint = 87
  b.ground(83, 92, 11)
  b.slab(85, 90, 7, T.PLANK).set(88, 3, T.BOX_BELL)
  b.mover(95, 10, 3, 0, -4, 200)
  b.mover(101, 6, 3, 5, 0, 220)
  b.ground(111, 116, 9).e('mud', 114, 8)
  b.e('kite', 118, 4)
  b.mover(119, 10, 2, 4, 0, 160)
  b.ground(127, 134, 11).e('cracker', 130, 10).e('cracker', 133, 10)
  b.row(128, 7, 'BMB')
  b.e('fish', 136, 13)
  b.mover(137, 9, 3, 5, -2, 240)
  b.ground(147, 151, 8).set(149, 4, T.BOX_LIFE)
  b.e('fish', 153, 13)
  b.mover(154, 10, 3, 4, 0, 200, 100)
  b.ground(162, 199)
  b.e('hog', 168).e('mud', 172)
  b.stairs(176, 4)
  b.set(188, 11, T.HARD)
  b.goal = 188
  return { id: '2-2', world: 2, name: '运河夜市', music: 'night', time: 300, start: 3, areas: [b.build()] }
}

// ───────────────────────────── 2-3 烟花塔 ─────────────────────────────
// A tall fireworks pagoda: floors stacked on floors, lifts between them, packs of 炮仗精 whose
// bursts clear the brick walls for you, and kites around the eaves.
function lv23(): Level {
  const b = new LB(190, 'tower')
  b.ground(0, 189)
  b.row(10, 8, '?P?')
  b.e('cracker', 18).e('cracker', 20)
  b.fill(24, 9, 25, 11, T.BRICK)
  b.e('cracker', 27)
  b.ground(30, 37, 8).fill(30, 3, 37, 3, T.WALL)
  b.e('mud', 34, 7)
  b.slab(40, 44, 9, T.PLANK).slab(40, 44, 5, T.PLANK).coins(40, 4, 5)
  b.mover(47, 11, 3, 0, -6, 240)
  b.ground(52, 60, 6).e('hog', 56, 5)
  b.clear(61, 63)
  b.slab(64, 66, 9, T.PLANK)
  b.ground(68, 76, 10).e('cracker', 71, 9).e('cracker', 74, 9)
  b.fill(77, 7, 78, 9, T.BRICK)
  b.checkpoint = 82
  b.set(86, 8, T.USED).e('firebar', 86, 8, { len: 5, dir: 1 })
  b.e('kite', 92, 3).e('kite', 98, 5)
  b.row(94, 8, 'BBNBB')
  b.clear(101, 106)
  b.mover(102, 9, 3, 0, -5, 200)
  b.ground(108, 113, 6).set(110, 2, T.BOX_LIFE)
  b.slab(115, 120, 9, T.PLANK)
  b.e('cracker', 117, 8)
  b.ground(122, 128, 11)
  b.fill(126, 8, 127, 10, T.BRICK)
  b.e('cracker', 124, 10)
  b.clear(129, 133)
  b.mover(129, 9, 2, 3, 0, 150)
  b.ground(135, 189)
  b.e('hog', 140).e('mud', 144).e('mud', 146)
  b.set(150, 8, T.USED).e('firebar', 150, 8, { len: 4, dir: -1 })
  b.row(154, 8, '?M?')
  b.stairs(160, 6)
  b.clear(166, 168)
  b.set(178, 11, T.HARD)
  b.goal = 178
  return { id: '2-3', world: 2, name: '烟花塔', music: 'night', time: 300, start: 3, areas: [b.build()] }
}

// ───────────────────────────── 2-4 年兽宫 ─────────────────────────────
// The 年兽's palace: lava channels, twin firebars, 火星子 everywhere, and the 年兽 itself in a
// wide hall at the end.
function lv24(): Level {
  const b = new LB(150, 'palace')
  b.ground(0, 149).fill(0, 0, 149, 2, T.WALL).fill(0, 3, 0, 11, T.WALL)
  b.ground(1, 5, 9).fill(1, 3, 10, 4, T.WALL)
  b.set(14, 8, T.USED).e('firebar', 14, 8, { len: 6, dir: 1 })
  b.fill(18, 12, 23, 13, T.LAVA).e('spark', 20, 13)
  b.ground(24, 30, 9).fill(24, 3, 30, 5, T.WALL)
  b.set(27, 9, T.USED).e('firebar', 27, 9, { len: 3, dir: 1 })
  b.fill(31, 12, 40, 13, T.LAVA).fill(35, 9, 36, 11, T.HARD)
  b.e('spark', 33, 13).e('spark', 38, 13)
  b.set(44, 7, T.BOX_POWER)
  b.e('hog', 46).e('mud', 49)
  b.fill(52, 12, 67, 13, T.LAVA)
  b.mover(53, 9, 3, 5, 0, 220).mover(60, 9, 3, -4, 0, 200)
  b.e('spark', 57, 13).e('spark', 64, 13)
  b.checkpoint = 71
  b.set(76, 6, T.USED).e('firebar', 76, 6, { len: 5, dir: -1 })
  b.set(76, 10, T.USED)
  b.set(83, 8, T.USED).e('firebar', 83, 8, { len: 6, dir: 1 })
  b.fill(72, 3, 88, 3, T.WALL)
  b.e('cracker', 88).e('cracker', 90)
  b.row(92, 8, 'B?B')
  b.fill(96, 12, 101, 13, T.LAVA).e('spark', 98, 13)
  b.ground(102, 108, 10).set(105, 6, T.BOX_POWER)
  // the hall
  b.fill(110, 3, 111, 8, T.WALL)
  b.clear(114, 143, 12, 13)
  b.fill(114, 13, 143, 13, T.LAVA)
  b.fill(114, 12, 143, 12, T.BRIDGE)
  b.ground(144, 149, 10)
  b.e('nian', 138, 11)
  b.boss = true
  return { id: '2-4', world: 2, name: '年兽宫', music: 'fort', time: 300, start: 3, areas: [b.build()] }
}

export const LEVELS: Level[] = [lv11(), lv12(), lv13(), lv14(), lv21(), lv22(), lv23(), lv24()]

export const WORLD_NAMES = ['', '竹林村', '灯火古镇']

export const THEME_MUSIC: Record<Theme, MusicId> = {
  bamboo: 'field',
  cave: 'cave',
  treetop: 'sky',
  fort: 'fort',
  town: 'town',
  canal: 'night',
  tower: 'night',
  palace: 'fort'
}
