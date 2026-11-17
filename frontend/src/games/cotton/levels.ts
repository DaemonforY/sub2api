// The fifteen floors of 棉花糖大作战: three places from a 90s summer holiday (卧室, 教室, 屋顶),
// five floors each, the fifth one a boss. Every floor is a single 40×30 screen of 8 px tiles; the
// top two rows hold the score bar, the bottom two are the ground and the outer columns are walls.
//
// Platforms are one-way shelves (jump up through them, land on top). Rows 23 / 18 / 13 / 8 are
// 40 px apart, comfortably inside the 51 px jump. Enemies stand on the platform row given.

export type EnemyKind = 'book' | 'clock' | 'mosquito' | 'chalk' | 'coil'
export type BossKind = 'bigclock' | 'bookpile' | 'calendar'

export interface FloorDef {
  /** one-way platforms: [row, firstCol, lastCol] (columns 1..38 are open) */
  plats: [number, number, number][]
  /** solid blocks: [row, firstCol, lastCol] */
  solids?: [number, number, number][]
  /** [kind, column, platform row the enemy stands on] — row 28 is the ground */
  enemies: [EnemyKind, number, number][]
  boss?: BossKind
}

export interface World {
  name: string
  /** background image under /games-media/cotton/ */
  bg: string
  /** shelf colours: top highlight, face, shadow, dark edge */
  shelf: [string, string, string, string]
  ground: [string, string, string]
  song: number
}

export const WORLDS: World[] = [
  { name: '卧室', bg: 'bg-bedroom.png', shelf: ['#f2c38b', '#c98a55', '#8e5a35', '#4a2c22'], ground: ['#8c7fb8', '#5f548e', '#3a3263'], song: 1 },
  { name: '教室', bg: 'bg-classroom.png', shelf: ['#e9b878', '#b47a43', '#7a4c2a', '#3e2619'], ground: ['#6aa59a', '#3f7470', '#234a4b'], song: 2 },
  { name: '屋顶', bg: 'bg-rooftop.png', shelf: ['#e7a07a', '#b8644a', '#7e3c34', '#41201f'], ground: ['#a3889e', '#6e5772', '#40324a'], song: 3 }
]

/** columns 1..38 are open; mirror a span to the other half */
function sym(row: number, a: number, b: number): [number, number, number][] {
  return [
    [row, a, b],
    [row, 39 - b, 39 - a]
  ]
}

export const FLOORS: FloorDef[] = [
  // 1 — 卧室: four homework books, learn shoot → ball → kick
  {
    plats: [...sym(23, 1, 9), [18, 10, 29], ...sym(13, 1, 11), [8, 12, 27]],
    enemies: [
      ['book', 15, 8],
      ['book', 24, 8],
      ['book', 5, 13],
      ['book', 34, 13]
    ]
  },
  // 2 — alarm clocks hop in
  {
    plats: [...sym(23, 5, 14), ...sym(18, 1, 10), [18, 16, 23], [13, 9, 30], ...sym(8, 1, 12)],
    enemies: [
      ['book', 6, 8],
      ['book', 33, 8],
      ['clock', 19, 13],
      ['book', 20, 18],
      ['clock', 30, 23]
    ]
  },
  // 3 — staircase
  {
    plats: [[23, 1, 8], [23, 15, 24], [23, 31, 38], ...sym(18, 5, 15), ...sym(13, 1, 9), [13, 16, 23], [8, 8, 31]],
    enemies: [
      ['book', 12, 8],
      ['clock', 27, 8],
      ['book', 4, 13],
      ['clock', 35, 13],
      ['book', 19, 13],
      ['clock', 19, 23]
    ]
  },
  // 4 — a toy chest in the middle to bank balls off
  {
    plats: [...sym(23, 1, 12), [18, 8, 31], ...sym(13, 1, 13), [8, 14, 25], ...sym(8, 1, 5)],
    solids: [
      [26, 18, 21],
      [27, 18, 21]
    ],
    enemies: [
      ['book', 3, 8],
      ['book', 36, 8],
      ['clock', 17, 8],
      ['clock', 22, 8],
      ['book', 10, 18],
      ['book', 29, 18],
      ['clock', 6, 23]
    ]
  },
  // 5 — boss: 大闹钟
  {
    plats: [...sym(23, 1, 7), ...sym(18, 1, 11), [13, 14, 25], ...sym(10, 1, 6)],
    enemies: [],
    boss: 'bigclock'
  },
  // 6 — 教室: mosquitoes ignore the shelves
  {
    plats: [...sym(23, 1, 10), [23, 16, 23], ...sym(18, 6, 16), ...sym(13, 1, 10), [13, 16, 23], [8, 9, 30]],
    enemies: [
      ['book', 12, 8],
      ['book', 27, 8],
      ['mosquito', 6, 13],
      ['mosquito', 33, 13],
      ['book', 19, 23]
    ]
  },
  // 7 — chalk throws from the desks
  {
    plats: [...sym(23, 4, 13), ...sym(18, 1, 8), [18, 14, 25], ...sym(13, 6, 15), ...sym(8, 1, 9), [8, 16, 23]],
    enemies: [
      ['chalk', 19, 8],
      ['book', 5, 8],
      ['book', 34, 8],
      ['chalk', 8, 13],
      ['mosquito', 31, 13],
      ['clock', 19, 18]
    ]
  },
  // 8 — two long desks and a gap in the middle
  {
    plats: [[23, 1, 16], [23, 23, 38], [18, 6, 33], [13, 1, 15], [13, 24, 38], [8, 10, 29]],
    enemies: [
      ['chalk', 4, 13],
      ['chalk', 35, 13],
      ['mosquito', 14, 8],
      ['mosquito', 25, 8],
      ['book', 12, 18],
      ['clock', 27, 18],
      ['book', 19, 8]
    ]
  },
  // 9 — tower
  {
    plats: [...sym(23, 1, 6), [23, 13, 26], ...sym(18, 7, 14), ...sym(13, 1, 8), [13, 14, 25], ...sym(8, 6, 15)],
    solids: [
      [24, 1, 2],
      [24, 37, 38]
    ],
    enemies: [
      ['chalk', 19, 13],
      ['clock', 9, 8],
      ['clock', 30, 8],
      ['mosquito', 4, 13],
      ['mosquito', 35, 13],
      ['book', 10, 18],
      ['book', 29, 18],
      ['chalk', 19, 23]
    ]
  },
  // 10 — boss: 作业山
  {
    plats: [...sym(23, 1, 9), [18, 1, 22], [13, 4, 20], [8, 1, 14]],
    enemies: [],
    boss: 'bookpile'
  },
  // 11 — 屋顶: flip-flops charge along the ledges
  {
    plats: [...sym(23, 1, 11), [18, 8, 31], ...sym(13, 1, 12), [8, 13, 26]],
    enemies: [
      ['coil', 14, 8],
      ['coil', 20, 18],
      ['book', 5, 13],
      ['book', 34, 13],
      ['mosquito', 24, 18]
    ]
  },
  // 12
  {
    plats: [[23, 1, 12], [23, 27, 38], [20, 15, 24], ...sym(16, 1, 10), [13, 14, 25], ...sym(9, 4, 13)],
    enemies: [
      ['coil', 6, 9],
      ['coil', 33, 9],
      ['chalk', 19, 13],
      ['clock', 5, 16],
      ['clock', 34, 16],
      ['mosquito', 19, 20]
    ]
  },
  // 13 — water tanks to bounce balls off
  {
    plats: [...sym(23, 1, 9), [18, 6, 33], ...sym(13, 1, 10), [13, 16, 23], [8, 8, 31]],
    solids: [
      [24, 13, 14],
      [25, 13, 14],
      [26, 13, 14],
      [27, 13, 14],
      [24, 25, 26],
      [25, 25, 26],
      [26, 25, 26],
      [27, 25, 26]
    ],
    enemies: [
      ['coil', 19, 18],
      ['chalk', 12, 8],
      ['chalk', 27, 8],
      ['mosquito', 4, 13],
      ['mosquito', 35, 13],
      ['book', 19, 13],
      ['clock', 19, 28]
    ]
  },
  // 14 — everyone
  {
    plats: [...sym(23, 3, 12), [23, 17, 22], ...sym(18, 1, 7), [18, 12, 27], ...sym(13, 4, 14), ...sym(8, 1, 8), [8, 14, 25]],
    enemies: [
      ['coil', 5, 8],
      ['coil', 34, 8],
      ['chalk', 19, 8],
      ['clock', 8, 13],
      ['clock', 31, 13],
      ['mosquito', 15, 18],
      ['mosquito', 24, 18],
      ['book', 19, 23]
    ]
  },
  // 15 — boss: 开学怪
  {
    plats: [...sym(23, 1, 8), [23, 15, 24], ...sym(18, 5, 13), [14, 14, 25], ...sym(13, 1, 6)],
    enemies: [],
    boss: 'calendar'
  }
]

export const FLOOR_COUNT = FLOORS.length

export function worldOf(floor: number): World {
  return WORLDS[Math.min(WORLDS.length - 1, Math.floor(floor / 5))]
}

export const ENEMY_NAMES: Record<EnemyKind, string> = {
  book: '作业本精',
  clock: '闹闹',
  mosquito: '嗡嗡',
  chalk: '粉笔头',
  coil: '蚊香圈'
}

export const BOSS_NAMES: Record<BossKind, string> = {
  bigclock: '大闹钟',
  bookpile: '作业山',
  calendar: '开学怪'
}
