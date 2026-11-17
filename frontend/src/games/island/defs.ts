// Shared constants and types for 椰风岛大冒险.

export const W = 400 // internal resolution
export const H = 224
export const T = 16 // tile size
export const ROWS = 14
export const FPS = 60

export type Theme = 'beach' | 'jungle' | 'volcano'

// tile ids
export const TE = 0 // empty
export const TS = 1 // solid ground
export const TO = 2 // one-way platform (pier plank / log / slab)
export const TW = 3 // water (gap bottom, kills)
export const TL = 4 // lava (gap bottom, kills)
export const TB = 5 // solid stone block

// input bits
export const IN_L = 1
export const IN_R = 2
export const IN_J = 4
export const IN_F = 8
export const IN_D = 16

export const KINDS = [
  'fruit',
  'clam',
  'item',
  'rock',
  'fire',
  'crab',
  'gull',
  'fish',
  'frog',
  'spider',
  'spiky',
  'bat',
  'flame',
  'boulder',
  'geyser',
  'spring',
  'flag',
  'plat',
  'boom',
  'shot',
  'boss',
  'sun'
] as const
export type Kind = (typeof KINDS)[number]
export const KIND_INDEX = Object.fromEntries(KINDS.map((k, i) => [k, i])) as Record<Kind, number>

// item ids carried by clams / item entities
export const IT_BOOM = 0
export const IT_BOARD = 1
export const IT_FAIRY = 2
export const IT_CLOUD = 3
export const IT_BASKET = 4
export const IT_GOLD = 5

export const ENERGY_MAX = 100
export const FAIRY_FRAMES = 600
export const CLOUD_FRAMES = 480

export interface Spawn {
  k: Kind
  x: number // px
  y: number // px (bottom for ground things)
  a?: number
  b?: number
}

export interface Stage {
  id: number
  name: string
  theme: Theme
  cols: number
  tiles: Uint8Array
  spawns: Spawn[]
  checkpoints: number[] // px x of each flag (index 0 = start)
  arenaX: number // px: left edge of the boss arena; the camera locks here
  boss: 0 | 1 | 2
}

// events emitted by the simulation for sound + particles (host sends them to the guest)
export const EV = {
  jump: 1,
  throw: 2,
  kill: 3,
  fruit: 4,
  clam: 5,
  item: 6,
  hurt: 7,
  trip: 8,
  die: 9,
  revive: 10,
  flag: 11,
  spring: 12,
  bossHit: 13,
  bossDie: 14,
  breakRock: 15,
  splash: 16,
  land: 17,
  boardLost: 18,
  fairy: 19,
  cloud: 20,
  clear: 21,
  catch: 22,
  clink: 23,
  quake: 24,
  bossRoar: 25,
  spit: 26,
  lowEnergy: 27,
  extinguish: 28,
  life: 29,
  sun: 30
} as const

export type GameEvent = [number, number, number, number] // type, x, y, extra

export type Phase = 'play' | 'clear' | 'over' | 'ending'
