// 虎兔大冒险 — shared constants and types.
//
// The world is a grid of 16 px tiles, 14 rows tall (the screen is 400×224, so the whole level
// height is always on screen and the camera only scrolls sideways). Physics runs at a fixed 60 steps a
// second in pixel units, so every speed below is "pixels per step".

export const TILE = 16
export const ROWS = 14
export const VIEW_W = 400
export const VIEW_H = ROWS * TILE
export const STEP_HZ = 60

/** buttons, packed into one small integer so inputs travel cheaply over the network */
export const BTN = { L: 1, R: 2, U: 4, D: 8, J: 16, F: 32 } as const

/**
 * Tile codes used in the level grid. Lower-case letters are only used while building a level
 * (enemies, starts); the finished grid holds tiles only.
 */
export const T = {
  EMPTY: '.',
  GROUND: '#',
  BRICK: 'B',
  /** treasure box with one copper coin */
  BOX: '?',
  /** treasure box with a 大力包子 (or a 辣椒 if the player is already big) */
  BOX_POWER: 'P',
  /** treasure box with a 金铃 */
  BOX_BELL: 'S',
  /** invisible box holding a 糖葫芦 (extra life) */
  BOX_LIFE: 'L',
  /** invisible box holding a coin */
  BOX_HIDDEN: 'H',
  /** brick that gives up to ten coins */
  BRICK_COINS: 'M',
  /** brick holding a 金铃 */
  BRICK_BELL: 'N',
  /** brick holding a 大力包子 / 辣椒 */
  BRICK_POWER: 'Q',
  /** emptied box */
  USED: 'U',
  /** stone block that never breaks */
  HARD: 'X',
  /** copper coin floating in the air */
  COIN: 'o',
  /** bamboo tube: top-left, top-right, body-left, body-right */
  TUBE_TL: '[',
  TUBE_TR: ']',
  TUBE_L: '{',
  TUBE_R: '}',
  /** one-way plank: stand on it, jump through from below */
  PLANK: '=',
  /** deadly: lava / boiling mud / canal water */
  LAVA: '~',
  /** solid decoration that looks like the level's walls (castle stone, roof tiles) */
  WALL: 'W',
  /** goal pole (drawn, not solid) */
  POLE: '|',
  /** bridge over the boss pit: collapses when the boss is beaten */
  BRIDGE: '_'
} as const

export type TileCode = (typeof T)[keyof typeof T]

export type Theme = 'bamboo' | 'cave' | 'treetop' | 'fort' | 'town' | 'canal' | 'tower' | 'palace'
export type MusicId = 'title' | 'field' | 'cave' | 'sky' | 'fort' | 'town' | 'night' | 'boss' | 'clear' | 'over' | 'bell' | 'world'

export type EnemyKind =
  | 'mud' // 泥丸子: walks, falls off ledges, squashes when stomped
  | 'hog' // 刺团: stomp to curl into a ball, kick the ball to bowl others over
  | 'kite' // 纸鸢精: flies in waves
  | 'rat' // 竹鼠: pops out of bamboo tubes
  | 'fish' // 跳跳鱼: leaps out of water / mud
  | 'cracker' // 炮仗精: stomp to light its fuse, it bursts and breaks bricks around it
  | 'firebar' // 火链: a chain of embers turning round a block
  | 'spark' // 火星子: jumps out of lava
  | 'mudking' // boss of world 1
  | 'nian' // boss of world 2 (年兽)

export type ItemKind = 'bun' | 'chili' | 'bell' | 'haw'

export interface Spawn {
  kind: EnemyKind
  /** tile column / row of the spawn */
  tx: number
  ty: number
  /** firebar: length and turn direction; others: start facing */
  dir?: number
  len?: number
}

/** a platform that moves along a fixed path; its position depends only on the level clock */
export interface Mover {
  /** tile coords of the path start, width in tiles */
  tx: number
  ty: number
  w: number
  /** path offset in tiles (dx, dy) and steps for one full back-and-forth */
  dx: number
  dy: number
  period: number
  phase?: number
}

export interface Warp {
  /** tube top-left tile: enter by pressing down on it (or right, for side tubes) */
  tx: number
  ty: number
  side?: boolean
  /** destination area and tile; `exitTube` pops out of a tube there */
  to: number
  ax: number
  ay: number
  exitTube?: boolean
}

export interface Area {
  grid: string[]
  width: number
  theme: Theme
  spawns: Spawn[]
  movers: Mover[]
  warps: Warp[]
  /** goal pole column (main area only) */
  goal?: number
  /** boss arena: when the boss falls, the level ends */
  boss?: boolean
  checkpoint?: number
}

export interface Level {
  id: string
  world: number
  name: string
  music: MusicId
  time: number
  start: number
  areas: Area[]
}

export type PlayerAnim = 'stand' | 'walk' | 'skid' | 'jump' | 'fall' | 'crouch' | 'die' | 'pole' | 'grow' | 'bubble' | 'tube' | 'gone'
