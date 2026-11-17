// Turns the pixel art into canvases: sprites from spriteData.ts, theme tiles and the parallax
// backgrounds, which are painted here pixel by pixel so everything shares one crisp 1:1 pixel grid.

import { CHILI_PALETTE, HERO_PALETTES, PALETTE, SPRITES } from './spriteData'
import type { Theme } from './types'
import { TILE, VIEW_H } from './types'

export type Canvas = HTMLCanvasElement

function makeCanvas(w: number, h: number): { c: Canvas; g: CanvasRenderingContext2D } {
  const c = document.createElement('canvas')
  c.width = w
  c.height = h
  const g = c.getContext('2d') as CanvasRenderingContext2D
  g.imageSmoothingEnabled = false
  return { c, g }
}

function paint(rows: string[], pal: Record<string, string>, flip = false): Canvas {
  const w = rows[0].length
  const { c, g } = makeCanvas(w, rows.length)
  rows.forEach((r, y) => {
    for (let x = 0; x < w; x++) {
      const ch = r[x]
      if (ch === '.') continue
      g.fillStyle = pal[ch] || PALETTE[ch] || '#f0f'
      g.fillRect(flip ? w - 1 - x : x, y, 1, 1)
    }
  })
  return c
}

const cache = new Map<string, Canvas>()

/** a sprite by name, optionally mirrored */
export function sprite(name: string, flip = false): Canvas {
  const key = name + (flip ? '<' : '>')
  let c = cache.get(key)
  if (!c) {
    c = paint(SPRITES[name] || SPRITES.used, PALETTE, flip)
    cache.set(key, c)
  }
  return c
}

/** a hero frame: head (+ rabbit ears) stacked on a body; drawn with feet at the canvas bottom */
export function heroFrame(hero: 'tiger' | 'rabbit', body: string, chili: boolean, flip: boolean, flash = false): Canvas {
  const key = `${hero}|${body}|${chili ? 1 : 0}|${flip ? 1 : 0}|${flash ? 1 : 0}`
  let c = cache.get(key)
  if (!c) {
    const pal: Record<string, string> = { ...PALETTE, ...HERO_PALETTES[hero], ...(chili ? CHILI_PALETTE : {}) }
    if (flash) for (const k of Object.keys(pal)) if (k !== 'o') pal[k] = '#ffffff'
    const rows = [...(hero === 'rabbit' ? SPRITES.ears_rabbit : []), ...SPRITES[`head_${hero}`], ...SPRITES[body]]
    c = paint(rows, pal, flip)
    cache.set(key, c)
  }
  return c
}

// ───────────────────────── tiles ─────────────────────────

interface ThemeColors {
  top: string[]
  soil: string[]
  brick: string[]
  wall: string[]
  hard: string[]
  liquid: string[]
  plank: string[]
  sky: string[]
}

export const THEME: Record<Theme, ThemeColors> = {
  bamboo: {
    top: ['#9be15a', '#5fbf3a', '#2f7d2a'],
    soil: ['#d69a5c', '#b07240', '#7f4d2c'],
    brick: ['#e0864a', '#b0532e', '#5a2a1e'],
    wall: ['#8f8a7a', '#6c6658', '#4a4538'],
    hard: ['#d8c6a0', '#a8946c', '#6e5d42'],
    liquid: ['#7fd4ff', '#3d9be0', '#2a64b0'],
    plank: ['#f0d278', '#c9a04a', '#7a5a26'],
    sky: ['#8fd6ff', '#bfe8ff', '#e9f8ff']
  },
  cave: {
    top: ['#b6a6e8', '#7d6bb8', '#4a3d7a'],
    soil: ['#5a4b8a', '#433769', '#2c2449'],
    brick: ['#6f86c8', '#4a5e9e', '#262f5a'],
    wall: ['#3d3366', '#2e264f', '#1c1736'],
    hard: ['#8d8fb0', '#62658a', '#3a3c5c'],
    liquid: ['#9a6a3a', '#73482a', '#4a2c1a'],
    plank: ['#c9a06a', '#9a7444', '#5a4224'],
    sky: ['#120f24', '#1c1838', '#2a2450']
  },
  treetop: {
    top: ['#a6ec6a', '#62c040', '#2e7a2c'],
    soil: ['#62c040', '#3f9a34', '#2a6a28'],
    brick: ['#e0864a', '#b0532e', '#5a2a1e'],
    wall: ['#8f8a7a', '#6c6658', '#4a4538'],
    hard: ['#d8c6a0', '#a8946c', '#6e5d42'],
    liquid: ['#7fd4ff', '#3d9be0', '#2a64b0'],
    plank: ['#f0d278', '#c9a04a', '#7a5a26'],
    sky: ['#6cc6ff', '#a6deff', '#e2f6ff']
  },
  fort: {
    top: ['#a8764e', '#7e5234', '#4e3020'],
    soil: ['#6e4a32', '#56382a', '#3a2418'],
    brick: ['#b06a3e', '#844a2a', '#4a2618'],
    wall: ['#5a4034', '#463026', '#2e1e18'],
    hard: ['#9a8a78', '#706252', '#463c32'],
    liquid: ['#a8743e', '#7e522a', '#55341a'],
    plank: ['#c9a06a', '#9a7444', '#5a4224'],
    sky: ['#1e1418', '#2c1c20', '#3c2626']
  },
  town: {
    top: ['#c9c4b8', '#9c968a', '#6a655c'],
    soil: ['#8a847a', '#6e695f', '#4e4a42'],
    brick: ['#c46a48', '#96462e', '#4e2418'],
    wall: ['#f2ead8', '#d4c8ae', '#5a3a2a'],
    hard: ['#b08a5a', '#86643c', '#523a20'],
    liquid: ['#5a8fd0', '#3a63a8', '#24407a'],
    plank: ['#e85a4a', '#b03a32', '#fff0d8'],
    sky: ['#ff9e6a', '#ffc98a', '#ffe6b0']
  },
  canal: {
    top: ['#a9b3c4', '#7d8799', '#4f5869'],
    soil: ['#6d7688', '#545c6c', '#3a4150'],
    brick: ['#b65c46', '#8a3e2e', '#45201a'],
    wall: ['#d8d0c0', '#b4aa96', '#4a3428'],
    hard: ['#a9b3c4', '#7d8799', '#4f5869'],
    liquid: ['#2e5aa8', '#1f3f80', '#142a58'],
    plank: ['#e85a4a', '#b03a32', '#fff0d8'],
    sky: ['#0e1430', '#1a2250', '#2c3672']
  },
  tower: {
    top: ['#c8763c', '#9a5228', '#5e2e16'],
    soil: ['#8a4424', '#6a321a', '#401c0e'],
    brick: ['#d06848', '#a0442e', '#521e14'],
    wall: ['#a8322c', '#7e2220', '#4a1210'],
    hard: ['#d8a84a', '#a87a2a', '#644416'],
    liquid: ['#2e5aa8', '#1f3f80', '#142a58'],
    plank: ['#e8b85a', '#b08236', '#5a3e14'],
    sky: ['#0a0c22', '#161a3c', '#262a58']
  },
  palace: {
    top: ['#e8b84a', '#b04030', '#6a1a18'],
    soil: ['#9a2e26', '#74201c', '#4a1210'],
    brick: ['#c85a3a', '#98382a', '#4e1810'],
    wall: ['#5a1a1e', '#441216', '#2a0a0e'],
    hard: ['#e0b048', '#a87a2a', '#644416'],
    liquid: ['#ffb02e', '#f06a1a', '#b0300e'],
    plank: ['#e8b85a', '#b08236', '#5a3e14'],
    sky: ['#1a0a0e', '#2a1016', '#3c161c']
  }
}

/** small deterministic hash for scattering details */
export function hash(x: number, y = 0, s = 0): number {
  let h = (x * 374761393 + y * 668265263 + s * 2147483647) | 0
  h = Math.imul(h ^ (h >>> 13), 1274126177)
  return ((h ^ (h >>> 16)) >>> 0) / 4294967296
}

type Pix = (x: number, y: number, col: string, w?: number, h?: number) => void

function tileCanvas(draw: (px: Pix) => void, w = TILE, h = TILE): Canvas {
  const { c, g } = makeCanvas(w, h)
  draw((x, y, col, ww = 1, hh = 1) => {
    g.fillStyle = col
    g.fillRect(x, y, ww, hh)
  })
  return c
}

const OUT = '#2b1b2e'

function drawGround(th: Theme, px: Pix, top: boolean, variant: number): void {
  const t = THEME[th]
  const [s0, s1, s2] = t.soil
  px(0, 0, s1, 16, 16)
  // texture
  for (let i = 0; i < 14; i++) {
    const x = Math.floor(hash(i, variant, 7) * 15)
    const y = Math.floor(hash(i, variant, 9) * 15)
    px(x, y, hash(i, variant, 3) > 0.5 ? s0 : s2, 2, 1)
  }
  if (th === 'town' || th === 'canal') {
    // stone slabs
    px(0, 7, s2, 16, 1)
    px(variant % 2 ? 5 : 11, 0, s2, 1, 7)
    px(variant % 2 ? 9 : 3, 8, s2, 1, 8)
    px(0, 0, s0, 16, 1)
  }
  if (th === 'fort' || th === 'palace' || th === 'tower') {
    px(0, 7, s2, 16, 1)
    px(0, 15, s2, 16, 1)
    px(variant % 2 ? 4 : 10, 0, s2, 1, 7)
    px(variant % 2 ? 12 : 6, 8, s2, 1, 7)
    px(0, 0, s0, 16, 1)
    px(0, 8, s0, 16, 1)
  }
  if (!top) return
  const [g0, g1, g2] = t.top
  if (th === 'bamboo' || th === 'treetop') {
    px(0, 0, g1, 16, 5)
    px(0, 0, g0, 16, 1)
    for (let x = 0; x < 16; x++) {
      const d = 4 + Math.floor(hash(x, variant, 1) * 3)
      px(x, 5, g1, 1, d - 4)
      px(x, d + 1, g2, 1, 1)
      if (hash(x, variant, 2) > 0.7) px(x, 1, g0, 1, 2)
    }
  } else if (th === 'cave') {
    px(0, 0, g1, 16, 3)
    px(0, 0, g0, 16, 1)
    for (let x = 0; x < 16; x++) if (hash(x, variant, 5) > 0.6) px(x, 3, g2, 1, 1 + Math.floor(hash(x, variant) * 2))
  } else if (th === 'town' || th === 'canal') {
    px(0, 0, g0, 16, 2)
    px(0, 2, g2, 16, 1)
  } else {
    px(0, 0, g0, 16, 2)
    px(0, 2, g1, 16, 1)
    px(0, 3, g2, 16, 1)
  }
}

function drawBrick(th: Theme, px: Pix): void {
  const [b0, b1, b2] = THEME[th].brick
  px(0, 0, b1, 16, 16)
  px(0, 0, b0, 16, 1)
  for (const y of [3, 7, 11, 15]) px(0, y, b2, 16, 1)
  for (let r = 0; r < 4; r++) {
    const y = r * 4
    const off = r % 2 ? 4 : 0
    for (let x = off; x < 16; x += 8) {
      px(x, y, b2, 1, 3)
      px(x + 1, y, b0, 5, 1)
    }
  }
}

function drawHard(th: Theme, px: Pix): void {
  const [h0, h1, h2] = THEME[th].hard
  px(0, 0, h1, 16, 16)
  px(0, 0, h0, 15, 1)
  px(0, 0, h0, 1, 15)
  px(1, 15, h2, 15, 1)
  px(15, 1, h2, 1, 15)
  px(2, 2, h0, 3, 1)
  px(2, 2, h0, 1, 3)
  px(11, 13, h2, 3, 1)
  px(13, 11, h2, 1, 3)
  px(0, 0, OUT, 16, 0)
}

function drawWall(th: Theme, px: Pix, exposedTop: boolean, x: number, y: number): void {
  const [w0, w1, w2] = THEME[th].wall
  if (th === 'town' || th === 'canal') {
    // 白墙黛瓦: whitewashed wall, dark timber posts, a few lattice windows, tiled roof on top
    px(0, 0, w0, 16, 16)
    px(0, 15, w1, 16, 1)
    if (x % 4 === 0) px(0, 0, w2, 2, 16)
    if (hash(x, y, 4) > 0.62 && x % 4 !== 0) {
      px(4, 4, w2, 9, 8)
      px(5, 5, '#ffcf6a', 7, 6)
      px(8, 5, w2, 1, 6)
      px(5, 7, w2, 7, 1)
      px(5, 9, w2, 7, 1)
    }
    if (exposedTop) {
      const R0 = '#3a3f58'
      const R1 = '#5a6282'
      const R2 = '#22253a'
      px(0, 0, R2, 16, 2)
      px(0, 2, R0, 16, 7)
      for (let i = 0; i < 16; i += 3) {
        px(i, 2, R1, 2, 6)
        px(i + 2, 2, R2, 1, 7)
      }
      px(0, 9, R2, 16, 1)
      px(0, 10, '#c9bfae', 16, 1)
      // ridge ornament at house corners
      if (x % 7 === 0) {
        px(1, 0, R2, 3, 1)
        px(0, 0, '#8a90a8', 1, 1)
      }
    }
    return
  }
  if (th === 'tower' || th === 'palace') {
    px(0, 0, w1, 16, 16)
    px(0, 0, w0, 16, 1)
    px(7, 0, w2, 2, 16)
    if (exposedTop) {
      px(0, 0, '#e8b84a', 16, 3)
      px(0, 3, '#8a5a1a', 16, 1)
    }
    return
  }
  px(0, 0, w1, 16, 16)
  for (let i = 0; i < 10; i++) px(Math.floor(hash(i, x, y) * 15), Math.floor(hash(y, i, x) * 15), hash(i, x) > 0.5 ? w0 : w2, 2, 2)
  if (th === 'fort') {
    px(0, 7, w2, 16, 1)
    px(0, 15, w2, 16, 1)
    px((x % 2) * 8 + 3, 0, w2, 1, 7)
    px((x % 2) * 8 + 7, 8, w2, 1, 7)
  }
}

function drawTube(px: Pix, part: string): void {
  const J = '#7ed957'
  const j = '#3f9a34'
  const d = '#21632a'
  const a = '#c2e85a'
  if (part === 'TL' || part === 'TR') {
    const left = part === 'TL'
    px(0, 0, OUT, 16, 16)
    const x0 = left ? 1 : 0
    const w = 15
    px(x0, 1, J, w, 13)
    px(x0, 1, a, w, 2)
    px(x0, 6, d, w, 1)
    px(x0, 7, a, w, 1)
    px(x0, 12, j, w, 2)
    if (left) {
      px(3, 3, a, 2, 9)
      px(6, 3, '#a6ec6a', 1, 9)
    } else {
      px(10, 3, j, 3, 9)
      px(13, 3, d, 2, 9)
    }
    px(0, 15, OUT, 16, 1)
    return
  }
  const left = part === 'L'
  px(0, 0, J, 16, 16)
  if (left) {
    px(0, 0, OUT, 1, 16)
    px(2, 0, J, 2, 16)
    px(4, 0, a, 2, 16)
    px(7, 0, '#a6ec6a', 1, 16)
  } else {
    px(15, 0, OUT, 1, 16)
    px(9, 0, j, 3, 16)
    px(12, 0, d, 3, 16)
  }
  // bamboo node
  px(left ? 1 : 0, 10, d, 15, 1)
  px(left ? 1 : 0, 11, a, 15, 1)
}

const tileCache = new Map<string, Canvas>()

/** canvas for a tile code in a theme; `ctx` carries neighbour info for auto-tiling */
export function tileArt(th: Theme, c: string, ctx: { top: boolean; x: number; y: number; frame: number }): Canvas | null {
  let key = ''
  let draw: ((px: Pix) => void) | null = null
  switch (c) {
    case '#': {
      const v = (ctx.x * 7 + ctx.y * 3) % 4
      key = `g${th}${ctx.top ? 1 : 0}${v}`
      draw = (px) => drawGround(th, px, ctx.top, v)
      break
    }
    case 'B':
    case 'M':
    case 'N':
    case 'Q':
      key = `b${th}`
      draw = (px) => drawBrick(th, px)
      break
    case 'X':
      key = `x${th}`
      draw = (px) => drawHard(th, px)
      break
    case 'W': {
      const v =
        th === 'town' || th === 'canal'
          ? `${hash(ctx.x, ctx.y, 4) > 0.62 ? 1 : 0}${ctx.x % 4 === 0 ? 1 : 0}${ctx.x % 7 === 0 ? 1 : 0}`
          : `${(ctx.x * 3 + ctx.y) % 5}|${ctx.x}|${ctx.y}`
      key = `w${th}${ctx.top ? 1 : 0}${v}${ctx.x % 2}`
      draw = (px) => drawWall(th, px, ctx.top, ctx.x, ctx.y)
      break
    }
    case '[':
      key = 'tTL'
      draw = (px) => drawTube(px, 'TL')
      break
    case ']':
      key = 'tTR'
      draw = (px) => drawTube(px, 'TR')
      break
    case '{':
      key = 'tL'
      draw = (px) => drawTube(px, 'L')
      break
    case '}':
      key = 'tR'
      draw = (px) => drawTube(px, 'R')
      break
    case '=': {
      const [p0, p1, p2] = THEME[th].plank
      const awning = th === 'town' || th === 'canal'
      key = `p${th}`
      draw = (px) => {
        if (awning) {
          for (let i = 0; i < 16; i += 4) px(i, 0, i % 8 ? p2 : p0, 4, 5)
          px(0, 5, p1, 16, 1)
          for (let i = 0; i < 16; i += 4) px(i + 1, 6, i % 8 ? p2 : p0, 2, 1)
        } else {
          px(0, 0, p0, 16, 2)
          px(0, 2, p1, 16, 3)
          px(0, 5, p2, 16, 1)
          px(4, 0, p2, 1, 5)
          px(12, 0, p2, 1, 5)
        }
      }
      break
    }
    case '_':
      key = `br${th}`
      draw = (px) => {
        px(0, 0, '#c9a06a', 16, 4)
        px(0, 0, '#f0d278', 16, 1)
        px(0, 4, '#5a4224', 16, 1)
        px(2, 5, '#5a4224', 2, 4)
        px(12, 5, '#5a4224', 2, 4)
        px(0, 8, '#7a5a26', 16, 1)
      }
      break
    case '~': {
      const f = Math.floor(ctx.frame / 10) % 4
      key = `l${th}${ctx.top ? 1 : 0}${f}`
      const [l0, l1, l2] = THEME[th].liquid
      draw = (px) => {
        for (let x = 0; x < 16; x++) {
          const wy = ctx.top ? Math.round(3 + Math.sin((x + f * 4) * 0.4) * 1.5) : 0
          px(x, wy, l1, 1, 16 - wy)
          if (ctx.top) px(x, wy, l0, 1, 1)
        }
        for (let i = 0; i < 6; i++) px(Math.floor(hash(i, 3, f) * 14), 7 + Math.floor(hash(i, 5) * 8), l2, 3, 1)
      }
      break
    }
    case '?':
    case 'P':
    case 'S': {
      const f = Math.floor(ctx.frame / 8) % 6
      return sprite(f < 3 ? 'box1' : f === 3 ? 'box2' : f === 4 ? 'box3' : 'box1')
    }
    case 'U':
      return sprite('used')
    case 'o': {
      const f = Math.floor(ctx.frame / 8) % 4
      return sprite(['coin1', 'coin2', 'coin3', 'coin2'][f])
    }
    default:
      return null
  }
  let cv = tileCache.get(key)
  if (!cv && draw) {
    cv = tileCanvas(draw)
    tileCache.set(key, cv)
  }
  return cv || null
}

// ───────────────────────── backgrounds ─────────────────────────

export interface BgLayer {
  c: Canvas
  /** parallax factor: 0 = fixed, 1 = moves with the world */
  k: number
  y: number
}

const bgCache = new Map<Theme, BgLayer[]>()

function gradientSky(cols: string[], h: number, w = 64): Canvas {
  // banded, dithered gradient: crisp pixel look instead of a smooth blend
  const { c, g } = makeCanvas(w, h)
  const bands = cols.length - 1
  for (let y = 0; y < h; y++) {
    const t = (y / h) * bands
    const i = Math.min(bands - 1, Math.floor(t))
    const f = t - i
    for (let x = 0; x < w; x++) {
      const bayer = [0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5][(y % 4) * 4 + (x % 4)] / 16
      g.fillStyle = f > bayer ? cols[i + 1] : cols[i]
      g.fillRect(x, y, 1, 1)
    }
  }
  return c
}

function ridge(w: number, h: number, base: number, amp: number, col: string, seed: number, edge?: string, rough = 1): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let x = 0; x < w; x++) {
    const t = (x / w) * Math.PI * 2
    const y =
      base -
      amp * (0.55 * Math.sin(t * 2 + seed) + 0.3 * Math.sin(t * 5 + seed * 2.1) + 0.15 * rough * Math.sin(t * 11 + seed * 3.7)) -
      amp * 0.3
    const yy = Math.round(y)
    g.fillStyle = col
    g.fillRect(x, yy, 1, h - yy)
    if (edge) {
      g.fillStyle = edge
      g.fillRect(x, yy, 1, 1)
    }
  }
  return c
}

function bambooGrove(w: number, h: number, cols: [string, string, string], seed: number, density: number, minW: number): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let i = 0; i < density; i++) {
    const x = Math.floor(hash(i, seed) * w)
    const sw = minW + Math.floor(hash(i, seed, 1) * 2)
    const top = Math.floor(hash(i, seed, 2) * 30)
    g.fillStyle = cols[1]
    g.fillRect(x, top, sw, h - top)
    g.fillStyle = cols[0]
    g.fillRect(x, top, 1, h - top)
    g.fillStyle = cols[2]
    for (let y = top + 8 + Math.floor(hash(i, seed, 3) * 10); y < h; y += 18 + Math.floor(hash(i, seed, 4) * 6)) g.fillRect(x, y, sw + 1, 1)
    // leaves
    for (let k = 0; k < 4; k++) {
      const ly = top + Math.floor(hash(i, k, seed) * 50)
      const dir = hash(k, i) > 0.5 ? 1 : -1
      g.fillStyle = cols[hash(k, i, 3) > 0.5 ? 0 : 1]
      for (let j = 0; j < 7; j++) g.fillRect(x + dir * (j + 1), ly + Math.floor(j / 2), 2, 1)
    }
  }
  return c
}

function stars(w: number, h: number, n: number, seed: number): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let i = 0; i < n; i++) {
    const x = Math.floor(hash(i, seed) * w)
    const y = Math.floor(hash(i, seed, 1) * h)
    g.fillStyle = hash(i, seed, 2) > 0.8 ? '#fff6c8' : '#9aa6e0'
    g.fillRect(x, y, 1, 1)
    if (hash(i, seed, 3) > 0.93) {
      g.fillRect(x - 1, y, 3, 1)
      g.fillRect(x, y - 1, 1, 3)
    }
  }
  return c
}

function clouds(w: number, h: number, n: number, seed: number, col: string, shade: string): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let i = 0; i < n; i++) {
    const cx = Math.floor(hash(i, seed) * w)
    const cy = Math.floor(hash(i, seed, 1) * h * 0.8) + 8
    const r = 6 + Math.floor(hash(i, seed, 2) * 8)
    for (let k = 0; k < 5; k++) {
      const bx = cx + (k - 2) * r * 0.7
      const br = r * (1 - Math.abs(k - 2) * 0.22)
      for (let y = -br; y <= 0; y++) {
        const half = Math.sqrt(br * br - y * y)
        g.fillStyle = y > -2 ? shade : col
        for (const wrap of [0, w, -w]) g.fillRect(Math.round(bx - half + wrap), Math.round(cy + y), Math.round(half * 2), 1)
      }
    }
  }
  return c
}

/** houses with curved roofs and lit windows: the old town skyline */
function skyline(w: number, h: number, seed: number, wall: string, roof: string, light: string, scale: number): Canvas {
  const { c, g } = makeCanvas(w, h)
  let x = 0
  let i = 0
  while (x < w) {
    const bw = Math.floor((22 + hash(i, seed) * 26) * scale)
    const bh = Math.floor((24 + hash(i, seed, 1) * 40) * scale)
    const top = h - bh
    g.fillStyle = wall
    g.fillRect(x, top, bw, bh)
    // roof with upturned eaves
    g.fillStyle = roof
    const rh = Math.max(4, Math.floor(7 * scale))
    g.fillRect(x - 3, top - rh, bw + 6, rh)
    g.fillRect(x - 5, top - rh - 1, 3, 2)
    g.fillRect(x + bw + 2, top - rh - 1, 3, 2)
    if (hash(i, seed, 5) > 0.6) {
      // a second storey / pagoda tier
      const tw = Math.floor(bw * 0.6)
      const tx = x + Math.floor((bw - tw) / 2)
      g.fillStyle = wall
      g.fillRect(tx, top - rh - Math.floor(12 * scale), tw, Math.floor(12 * scale))
      g.fillStyle = roof
      g.fillRect(tx - 3, top - rh - Math.floor(12 * scale) - rh + 1, tw + 6, rh - 1)
    }
    g.fillStyle = light
    for (let wy = top + 6; wy < h - 6; wy += Math.floor(12 * scale) + 2)
      for (let wx = x + 4; wx < x + bw - 5; wx += 8) if (hash(wx, wy, seed) > 0.45) g.fillRect(wx, wy, 3, 4)
    x += bw + Math.floor(hash(i, seed, 2) * 8 * scale) + 4
    i++
  }
  return c
}

function stalactites(w: number, h: number, col: string, seed: number, up: boolean): Canvas {
  const { c, g } = makeCanvas(w, h)
  g.fillStyle = col
  for (let x = 0; x < w; x++) {
    const n = hash(Math.floor(x / 3), seed)
    const len = n > 0.7 ? Math.floor(n * 40) : Math.floor(n * 10) + 2
    if (up) g.fillRect(x, h - len, 1, len)
    else g.fillRect(x, 0, 1, len)
  }
  return c
}

function columns(w: number, h: number, col: string, dark: string, gold: string, seed: number): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let x = 0; x < w; x += 64) {
    const cx = x + 20 + Math.floor(hash(x, seed) * 10)
    g.fillStyle = dark
    g.fillRect(cx, 0, 14, h)
    g.fillStyle = col
    g.fillRect(cx + 2, 0, 10, h)
    g.fillStyle = gold
    g.fillRect(cx - 2, 30, 18, 3)
    g.fillRect(cx - 2, h - 40, 18, 3)
    // a torch between columns
    g.fillStyle = '#3a1a10'
    g.fillRect(cx + 34, 92, 3, 10)
    g.fillStyle = '#ffb02e'
    g.fillRect(cx + 33, 86, 5, 6)
    g.fillStyle = '#fff07a'
    g.fillRect(cx + 34, 87, 3, 3)
  }
  return c
}

function lanternString(w: number, h: number, seed: number): Canvas {
  const { c, g } = makeCanvas(w, h)
  for (let x = 0; x < w; x++) {
    const y = Math.round(10 + Math.sin((x / w) * Math.PI * 4) * 6)
    g.fillStyle = '#3a2020'
    g.fillRect(x, y, 1, 1)
    if (x % 24 === 6) {
      g.fillStyle = '#e8282a'
      g.fillRect(x - 3, y + 2, 7, 7)
      g.fillRect(x - 2, y + 1, 5, 9)
      g.fillStyle = '#ffd23f'
      g.fillRect(x - 1, y + 3, 3, 4)
      g.fillRect(x - 1, y + 10, 3, 2)
    }
  }
  void seed
  return c
}

export function backgrounds(th: Theme): BgLayer[] {
  let layers = bgCache.get(th)
  if (layers) return layers
  const H = VIEW_H
  const W = 512
  const t = THEME[th]
  switch (th) {
    case 'bamboo':
      layers = [
        { c: gradientSky(['#7fcfff', '#a8e0ff', '#d8f4ff', '#f4fff0'], H), k: 0, y: 0 },
        { c: clouds(W, 90, 7, 1, '#ffffff', '#d8eeff'), k: 0.08, y: 4 },
        { c: ridge(W, H, 120, 50, '#9fc8d8', 2, '#c4e2ee'), k: 0.15, y: 0 },
        { c: ridge(W, H, 150, 34, '#6fb27a', 5, '#8fd08a'), k: 0.3, y: 0 },
        { c: bambooGrove(W, 140, ['#5fae4a', '#3f8a3a', '#2c6a2e'], 3, 40, 3), k: 0.5, y: 84 }
      ]
      break
    case 'treetop':
      layers = [
        { c: gradientSky(['#4cb4ff', '#86d0ff', '#c4ecff', '#f2fbff'], H), k: 0, y: 0 },
        { c: ridge(W, H, 110, 50, '#a8cbe8', 7, '#d0e6f8', 1.4), k: 0.1, y: 0 },
        { c: clouds(W, 120, 10, 9, '#ffffff', '#cfe6ff'), k: 0.2, y: 120 },
        { c: clouds(W, 80, 6, 4, '#ffffff', '#e0f0ff'), k: 0.06, y: 10 },
        { c: clouds(W, 70, 9, 11, '#f4faff', '#c4dcf4'), k: 0.45, y: 170 }
      ]
      break
    case 'cave':
      layers = [
        { c: gradientSky(['#0c0a1a', '#18142e', '#241e44'], H), k: 0, y: 0 },
        { c: stalactites(W, 70, '#1e1838', 2, false), k: 0.2, y: 30 },
        { c: stalactites(W, 80, '#1a1530', 5, true), k: 0.25, y: H - 80 },
        { c: stalactites(W, 50, '#2c2450', 8, false), k: 0.45, y: 28 },
        { c: ridge(W, H, 190, 18, '#2a2248', 3), k: 0.5, y: 0 }
      ]
      break
    case 'fort':
      layers = [
        { c: gradientSky(['#140c10', '#22141a', '#341e22'], H), k: 0, y: 0 },
        { c: columns(W, H, '#3e2a24', '#2a1c18', '#6e4a2a', 3), k: 0.35, y: 0 },
        { c: ridge(W, H, 200, 10, '#2a1a16', 6), k: 0.5, y: 0 }
      ]
      break
    case 'palace':
      layers = [
        { c: gradientSky(['#1a080c', '#2c0e14', '#46141a'], H), k: 0, y: 0 },
        { c: columns(W, H, '#8a1e1e', '#5a1214', '#e0b048', 7), k: 0.35, y: 0 },
        { c: ridge(W, H, 206, 8, '#3a0e10', 2), k: 0.5, y: 0 }
      ]
      break
    case 'town':
      layers = [
        { c: gradientSky(['#6a5a9e', '#c46a8a', '#ff9e6a', '#ffd08a'], H), k: 0, y: 0 },
        { c: ridge(W, H, 120, 40, '#8a6a9a', 4, '#a882a8'), k: 0.08, y: 0 },
        { c: skyline(W, 120, 3, '#6a4a6a', '#3e2a48', '#ffc86a', 0.8), k: 0.2, y: 70 },
        { c: skyline(W, 110, 9, '#4a3448', '#2a1c30', '#ffd36a', 1.1), k: 0.4, y: 114 },
        { c: lanternString(W, 30, 1), k: 0.6, y: 20 }
      ]
      break
    case 'canal':
      layers = [
        { c: gradientSky(['#070a1e', '#0e1430', '#1c2452', '#2c3672'], H), k: 0, y: 0 },
        { c: stars(W, 120, 120, 3), k: 0.02, y: 0 },
        { c: skyline(W, 120, 5, '#1c2040', '#0e1028', '#ffcf5a', 0.9), k: 0.2, y: 76 },
        { c: skyline(W, 100, 13, '#141830', '#0a0c1e', '#ffb84a', 1.2), k: 0.4, y: 120 },
        { c: lanternString(W, 30, 2), k: 0.6, y: 34 }
      ]
      break
    case 'tower':
      layers = [
        { c: gradientSky(['#05061a', '#0e1230', '#1c2050'], H), k: 0, y: 0 },
        { c: stars(W, 140, 140, 7), k: 0.02, y: 0 },
        { c: skyline(W, 150, 21, '#161a38', '#0c0e24', '#ff9a4a', 1.4), k: 0.25, y: 74 },
        { c: lanternString(W, 30, 3), k: 0.55, y: 8 }
      ]
      break
  }
  void t
  bgCache.set(th, layers)
  return layers
}

export { makeCanvas }
