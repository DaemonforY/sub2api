// Pixel art for 棉花糖大作战, drawn in code: small character grids for the kids, pests and
// snacks, simple shapes for the bosses and candy balls. Everything is built once into little
// canvases with a 1 px dark outline, then blitted at 1:1 onto the 320×240 screen.

export type Sprite = HTMLCanvasElement

const OUTLINE = '#1b1028'

/** shared palette; per-sprite maps can override letters */
const BASE: Record<string, string> = {
  K: '#1b1028',
  k: '#3a2a4d',
  H: '#2b1d33',
  h: '#56406b',
  S: '#ffd5b0',
  s: '#eaa47f',
  E: '#1b1028',
  B: '#ff8fa3',
  m: '#c2416b',
  W: '#fff7f0',
  w: '#cfc3dc',
  Y: '#ffd23f',
  y: '#fff2a0',
  O: '#ff9a3c',
  o: '#c4622a',
  P: '#ff86c3',
  p: '#ffc9e6',
  M: '#d54b98',
  G: '#6ccb6a',
  g: '#3a8f4f',
  L: '#7cc8ff',
  l: '#c7ecff',
  N: '#3657b5',
  n: '#26387a',
  C: '#8f6ae0',
  c: '#c8b6f5',
  T: '#9b6a45',
  t: '#5e3d2c',
  R: '#e8414f',
  r: '#a82b45',
  X: '#ff5a3c',
  Z: '#7a6e8c'
}

function canvas(w: number, h: number): [HTMLCanvasElement, CanvasRenderingContext2D] {
  const c = document.createElement('canvas')
  c.width = w
  c.height = h
  const ctx = c.getContext('2d')!
  ctx.imageSmoothingEnabled = false
  return [c, ctx]
}

/** add a 1 px outline round every opaque pixel (on a canvas that already has a 1 px margin) */
function outline(c: HTMLCanvasElement, color = OUTLINE): HTMLCanvasElement {
  const ctx = c.getContext('2d')!
  const img = ctx.getImageData(0, 0, c.width, c.height)
  const d = img.data
  const w = c.width
  const h = c.height
  const solid = (x: number, y: number) => x >= 0 && y >= 0 && x < w && y < h && d[(y * w + x) * 4 + 3] > 0
  const [or, og, ob] = [parseInt(color.slice(1, 3), 16), parseInt(color.slice(3, 5), 16), parseInt(color.slice(5, 7), 16)]
  const add: number[] = []
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      if (solid(x, y)) continue
      if (solid(x - 1, y) || solid(x + 1, y) || solid(x, y - 1) || solid(x, y + 1)) add.push(x, y)
    }
  }
  for (let i = 0; i < add.length; i += 2) {
    const o = (add[i + 1] * w + add[i]) * 4
    d[o] = or
    d[o + 1] = og
    d[o + 2] = ob
    d[o + 3] = 255
  }
  ctx.putImageData(img, 0, 0)
  return c
}

/** snap anti-aliased shapes to hard pixels: alpha to 0 or 255 */
function harden(c: HTMLCanvasElement, cut = 110): HTMLCanvasElement {
  const ctx = c.getContext('2d')!
  const img = ctx.getImageData(0, 0, c.width, c.height)
  const d = img.data
  for (let i = 3; i < d.length; i += 4) d[i] = d[i] >= cut ? 255 : 0
  ctx.putImageData(img, 0, 0)
  return c
}

function paint(ctx: CanvasRenderingContext2D, rows: string[], pal: Record<string, string>, ox = 0, oy = 0): void {
  rows.forEach((row, y) => {
    for (let x = 0; x < row.length; x++) {
      const ch = row[x]
      if (ch === '.' || ch === ' ') continue
      const col = pal[ch] ?? BASE[ch]
      if (!col) continue
      ctx.fillStyle = col
      ctx.fillRect(ox + x, oy + y, 1, 1)
    }
  })
}

function grid(rows: string[], pal: Record<string, string> = {}, line = true): Sprite {
  const w = Math.max(...rows.map((r) => r.length))
  const [c, ctx] = canvas(w + 2, rows.length + 2)
  paint(ctx, rows, pal, 1, 1)
  return line ? outline(c) : c
}

export function flip(src: Sprite): Sprite {
  const [c, ctx] = canvas(src.width, src.height)
  ctx.translate(src.width, 0)
  ctx.scale(-1, 1)
  ctx.drawImage(src, 0, 0)
  return c
}

function tint(src: Sprite, color: string, alpha = 1): Sprite {
  const [c, ctx] = canvas(src.width, src.height)
  ctx.drawImage(src, 0, 0)
  ctx.globalCompositeOperation = 'source-atop'
  ctx.globalAlpha = alpha
  ctx.fillStyle = color
  ctx.fillRect(0, 0, c.width, c.height)
  return c
}

// ------------------------------------------------------------------ kids (22×20, facing right)

const HEAD_GIRL = [
  '....HHHHHHH.....',
  '..HHHHHHHHHHH...',
  '.HHHHHhhhHHHHH..',
  '.HHHHHHHHHHHHHY.',
  '.HHHHHHHHHHHHYYY',
  '.HHHHHHHHHHHHHY.',
  '.HHHHSSSSESSES..',
  '.HHHHSSSSESSES..',
  '.HHHHSBBSSSSBS..',
  '..HHHSSSSSSmSS..',
  '..HHH.SSSSSSS...'
]

const HEAD_BOY = [
  '...H...H..H.....',
  '...HH.HHH.HH....',
  '..HHHHHHhHHHH...',
  '.HHHHHHHHhhHHH..',
  '.HHHHHHHHHHHHHH.',
  '.HHHHHHHHHHHHH..',
  '.HHHSSSSSESSES..',
  '.HHHSSSSSESSES..',
  '..HSSWBSSSSSBS..',
  '..HSSSSSSSSmSS..',
  '...SSSSSSSSSS...'
]

const TORSO = [
  '....RRRWRRR.....',
  '...RRRRWRRRRSS..',
  '...WRRRWRRR.....',
  '...SRRRWRRR.....',
  '....rrrrrrr.....'
]

const TORSO_SHOOT = [
  '....RRRWRRR.....',
  '...RRRRWRRRRRRSS',
  '...WRRRWRRR.....',
  '...SRRRWRRR.....',
  '....rrrrrrr.....'
]

const LEGS: Record<string, string[]> = {
  stand: ['....rRRRRRr.....', '....RRR.RRR.....', '....RRR.RRR.....', '...WWWW.WWWW....'],
  runA: ['....rRRRRRr.....', '...RRR...RRR....', '..RRR.....RRR...', '.WWW.......WWW..'],
  runB: ['....rRRRRRr.....', '.....RRRRR......', '.....RRRRR......', '....WWWWWWW.....'],
  jump: ['....rRRRRRr.....', '...RRR..RRR.....', '..WWW....RRR....', '..........WWW...'],
  fall: ['....rRRRRRr.....', '....RRR.RRR.....', '...RRR...RRR....', '..WWW.....WWW...'],
  kick: ['....rRRRRRRRRR..', '....RRR....RWWW.', '....RRR.........', '...WWWW.........']
}

const STAR = ['..Y..', '.YYY.', 'YYYYY', '.YyY.', '.Y.Y.']

export interface KidSprites {
  /** frames by name, each [right, left] */
  f: Record<string, [Sprite, Sprite]>
  ghost: Sprite
  icon: Sprite
}

function kid(head: string[], suit: [string, string], hair: [string, string]): KidSprites {
  const pal = { R: suit[0], r: suit[1], H: hair[0], h: hair[1] }
  const make = (legs: string[], torso: string[], wand: 'up' | 'fwd' | 'none', bob: number): Sprite => {
    const [c, ctx] = canvas(24, 22)
    const o = 1
    paint(ctx, legs, pal, o, o + 16)
    paint(ctx, torso, pal, o, o + 11 + bob)
    paint(ctx, head, pal, o, o + bob)
    if (wand === 'up') {
      ctx.fillStyle = BASE.T
      ctx.fillRect(o + 14, o + 11 + bob, 1, 1)
      ctx.fillRect(o + 15, o + 10 + bob, 1, 1)
      paint(ctx, STAR, pal, o + 15, o + 5 + bob)
    } else if (wand === 'fwd') {
      ctx.fillStyle = BASE.T
      ctx.fillRect(o + 16, o + 12 + bob, 2, 1)
      paint(ctx, STAR, pal, o + 17, o + 10 + bob)
    }
    return outline(c)
  }
  const both = (s: Sprite): [Sprite, Sprite] => [s, flip(s)]
  const f: Record<string, [Sprite, Sprite]> = {
    stand: both(make(LEGS.stand, TORSO, 'up', 0)),
    breathe: both(make(LEGS.stand, TORSO, 'up', 1)),
    runA: both(make(LEGS.runA, TORSO, 'up', 1)),
    runB: both(make(LEGS.runB, TORSO, 'up', 0)),
    jump: both(make(LEGS.jump, TORSO, 'up', 0)),
    fall: both(make(LEGS.fall, TORSO, 'up', 0)),
    shoot: both(make(LEGS.stand, TORSO_SHOOT, 'fwd', 0)),
    shootAir: both(make(LEGS.jump, TORSO_SHOOT, 'fwd', 0)),
    kick: both(make(LEGS.kick, TORSO_SHOOT, 'none', 0))
  }
  // a candy cloud with the kid's face peeking out
  const [g, gx] = canvas(26, 24)
  gx.fillStyle = BASE.p
  for (const [x, y, r] of [
    [7, 15, 6],
    [13, 12, 8],
    [19, 15, 6],
    [10, 18, 5],
    [16, 18, 5]
  ])
    circle(gx, x, y, r)
  paint(gx, head.slice(0, 11), pal, 5, 1)
  gx.fillStyle = BASE.W
  circle(gx, 9, 17, 2)
  circle(gx, 18, 18, 2)
  const ghost = outline(harden(g))
  const [ic, icx] = canvas(18, 13)
  paint(icx, head, pal, 1, 1)
  return { f, ghost, icon: outline(ic) }
}

function circle(ctx: CanvasRenderingContext2D, x: number, y: number, r: number): void {
  ctx.beginPath()
  ctx.arc(x, y, r, 0, Math.PI * 2)
  ctx.fill()
}

// ------------------------------------------------------------------ pests (facing right)

const BOOK = [
  [
    '..WGGGGGGGGGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGyyyyyyGG..',
    '.WWgGykkkkyGG..',
    '.WWgGyyyyyyGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGKKGGKKGG..',
    '.WWgGEWGGEWGG..',
    '.WWgGEEGGEEGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGGGrrrGGG..',
    '..WgGGGGGGGGG..',
    '...gggggggggg..',
    '....t.....t....',
    '...tt....tt....'
  ],
  [
    '..WGGGGGGGGGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGyyyyyyGG..',
    '.WWgGykkkkyGG..',
    '.WWgGyyyyyyGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGKKGGKKGG..',
    '.WWgGEWGGEWGG..',
    '.WWgGEEGGEEGG..',
    '.WWgGGGGGGGGG..',
    '.WWgGGGrrrGGG..',
    '..WgGGGGGGGGG..',
    '...gggggggggg..',
    '...t.......t...',
    '..tt......tt...'
  ]
]

const CLOCK = [
  [
    '.YY.......YY.',
    'YYYY.KKK.YYYY',
    'YYY.RRRRR.YYY',
    '...RRRRRRR...',
    '..RRWWWWWRR..',
    '.RRWWWKWWWRR.',
    '.RWWEWKWEWWR.',
    '.RWWEWKKEWWR.',
    '.RWWWWWWWWWR.',
    '.RRWWKKKWWRR.',
    '..RRWWWWWRR..',
    '...rRRRRRr...',
    '...t.....t...',
    '..tt.....tt..'
  ],
  [
    '..YY.....YY..',
    '.YYYYKKKYYYY.',
    '.YYY.RRR.YYY.',
    '...RRRRRRR...',
    '..RRWWWWWRR..',
    '.RRWWWKWWWRR.',
    '.RWWEWKWEWWR.',
    '.RWWEWKKEWWR.',
    '.RWWWWWWWWWR.',
    '.RRWWKKKWWRR.',
    '..RRWWWWWRR..',
    '...rRRRRRr...',
    '....t...t....',
    '...tt...tt...'
  ]
]

const MOSQUITO = [
  [
    '...ll...ll.....',
    '..llll.llll....',
    '..lllllllll....',
    '...llCCCll.....',
    '.....CCCCC.....',
    '....CCWECCKKKK.',
    '..ccCCCCCC.....',
    '.cCcCcCCC......',
    '.ccCcCc........',
    '...k.k.k.......'
  ],
  [
    '...............',
    '...............',
    '..lllllllll....',
    '.llllCCClll....',
    'lll..CCCCC.lll.',
    '....CCWECCKKKK.',
    '..ccCCCCCC.....',
    '.cCcCcCCC......',
    '.ccCcCc........',
    '..k..k..k......'
  ]
]

const CHALK = [
  [
    '..wyyyw..',
    '.yyyyyyy.',
    '.yyyyyyy.',
    '.yEyyyEy.',
    '.yEyyyEy.',
    '.yyyyyyy.',
    '.yyykkyy.',
    '.yyyyyyy.',
    '.Yyyyyyy.',
    '.YYyyyyY.',
    '.YYYYYYY.',
    '..t...t..',
    '.tt..tt..'
  ],
  [
    '..wyyyw..',
    '.yyyyyyy.',
    '.yyyyyyy.',
    '.yEyyyEy.',
    '.yEyyyEy.',
    '.yyyyyyy.',
    '.yyykkyy.',
    '.yyyyyyy.',
    '.Yyyyyyy.',
    '.YYyyyyY.',
    '.YYYYYYY.',
    '...t.t...',
    '..tt.tt..'
  ]
]

/** 蚊香圈: a green mosquito coil that rolls like a wheel when it charges; the face stays upright */
function coil(rot: number): Sprite {
  const [c, ctx] = canvas(17, 17)
  const cx = 8.5
  const cy = 8.5
  ctx.fillStyle = '#2f6b3a'
  circle(ctx, cx, cy, 7.4)
  harden(c)
  // the spiral, turning with the roll
  ctx.fillStyle = '#7fd36a'
  for (let t = 0; t < 1; t += 0.004) {
    const a = rot + t * Math.PI * 2 * 2.6
    const r = 1.2 + t * 5.6
    ctx.fillRect(Math.round(cx + Math.cos(a) * r - 0.5), Math.round(cy + Math.sin(a) * r - 0.5), 1, 1)
  }
  // glowing ember at the outer end
  const ea = rot + Math.PI * 2 * 2.6
  ctx.fillStyle = '#ff5a3c'
  ctx.fillRect(Math.round(cx + Math.cos(ea) * 6.8 - 1), Math.round(cy + Math.sin(ea) * 6.8 - 1), 2, 2)
  ctx.fillStyle = '#ffd23f'
  ctx.fillRect(Math.round(cx + Math.cos(ea) * 6.8 - 0.5), Math.round(cy + Math.sin(ea) * 6.8 - 0.5), 1, 1)
  // grumpy face
  ctx.fillStyle = BASE.K
  ctx.fillRect(5, 6, 3, 1)
  ctx.fillRect(10, 6, 3, 1)
  ctx.fillStyle = BASE.W
  ctx.fillRect(5, 7, 2, 2)
  ctx.fillRect(10, 7, 2, 2)
  ctx.fillStyle = BASE.K
  ctx.fillRect(6, 8, 1, 1)
  ctx.fillRect(11, 8, 1, 1)
  ctx.fillRect(7, 11, 4, 1)
  return outline(c)
}

const SLIPPER = [
  '......PPPPPP....',
  '....PPpppppPPP..',
  '..PPpWpppWppPPP.',
  '.PPpppppppppppPP',
  'MMMMMMMMMMMMMMMM',
  'RRRRRRRRRRRRRRRR',
  '.rrrrrrrrrrrrrr.'
]

export interface PestSprites {
  /** kind → frames → [right, left] */
  f: Record<string, [Sprite, Sprite][]>
}

// ------------------------------------------------------------------ bosses (shapes)

function bigClock(frame: number, angry: boolean): Sprite {
  const [c, ctx] = canvas(50, 50)
  const sh = frame ? 1 : -1
  // bells and hammer
  ctx.fillStyle = BASE.t
  ctx.fillRect(24, 4, 2, 6)
  ctx.fillStyle = BASE.Y
  circle(ctx, 10 + sh, 10, 7)
  circle(ctx, 39 - sh, 10, 7)
  ctx.fillStyle = BASE.y
  circle(ctx, 8 + sh, 8, 2)
  circle(ctx, 37 - sh, 8, 2)
  // legs
  ctx.fillStyle = BASE.t
  ctx.fillRect(13, 41, 4, 7)
  ctx.fillRect(33, 41, 4, 7)
  ctx.fillRect(10, 46, 8, 3)
  ctx.fillRect(32, 46, 8, 3)
  // body
  ctx.fillStyle = angry ? '#ff3b30' : BASE.R
  circle(ctx, 25, 27, 19)
  ctx.fillStyle = BASE.r
  ctx.beginPath()
  ctx.arc(25, 27, 19, 0.15 * Math.PI, 0.85 * Math.PI)
  ctx.arc(25, 25, 17, 0.85 * Math.PI, 0.15 * Math.PI, true)
  ctx.fill()
  ctx.fillStyle = BASE.W
  circle(ctx, 25, 27, 14)
  harden(c)
  // ticks
  ctx.fillStyle = BASE.w
  for (let i = 0; i < 12; i++) {
    const a = (i / 12) * Math.PI * 2
    ctx.fillRect(Math.round(25 + Math.cos(a) * 12), Math.round(27 + Math.sin(a) * 12), 1, 1)
  }
  // angry face: brows, eyes, hands as a nose, a grimace
  ctx.fillStyle = BASE.K
  ctx.fillRect(15, 19, 6, 2)
  ctx.fillRect(29, 19, 6, 2)
  ctx.fillRect(17, 21, 2, 1)
  ctx.fillRect(31, 21, 2, 1)
  ctx.fillRect(17, 22, 3, 4)
  ctx.fillRect(30, 22, 3, 4)
  ctx.fillStyle = BASE.W
  ctx.fillRect(18, 22, 1, 1)
  ctx.fillRect(31, 22, 1, 1)
  ctx.fillStyle = BASE.K
  ctx.fillRect(24, 16, 2, 11)
  ctx.fillRect(25, 26, 6, 2)
  ctx.fillStyle = BASE.m
  ctx.fillRect(19, 31, 12, 4)
  ctx.fillStyle = BASE.W
  for (let x = 20; x < 31; x += 2) ctx.fillRect(x, 31, 1, 2)
  return outline(c)
}

function bookPile(frame: number, angry: boolean): Sprite {
  const [c, ctx] = canvas(58, 60)
  const books: [number, number, number, string, string][] = [
    [6, 48, 46, '#3f7fe8', '#2a4fa8'],
    [2, 40, 50, '#e8414f', '#a82b45'],
    [8, 32, 42, '#6ccb6a', '#3a8f4f'],
    [4, 24, 48, '#ffd23f', '#c49a1f'],
    [9, 16, 40, '#8f6ae0', '#5d42a8'],
    [12, 8, 34, '#ff9a3c', '#c4622a']
  ]
  books.forEach(([x, y, w, col, dark], i) => {
    const dx = frame && i % 2 ? 1 : 0
    ctx.fillStyle = dark
    ctx.fillRect(x + dx, y, w, 8)
    ctx.fillStyle = col
    ctx.fillRect(x + dx, y, w, 6)
    ctx.fillStyle = BASE.W
    ctx.fillRect(x + dx + w - 5, y + 1, 4, 5)
    ctx.fillStyle = BASE.w
    ctx.fillRect(x + dx + w - 5, y + 3, 4, 1)
  })
  // a sheet of homework sticking out of the top like a tongue / crown
  ctx.fillStyle = BASE.W
  ctx.fillRect(18, 1, 18, 8)
  ctx.fillStyle = BASE.L
  for (let y = 3; y < 9; y += 2) ctx.fillRect(20, y, 14, 1)
  ctx.fillStyle = BASE.R
  ctx.fillRect(30, 2, 4, 1)
  ctx.fillRect(31, 1, 1, 3)
  // face on the yellow book
  ctx.fillStyle = BASE.K
  const ey = angry ? 26 : 26
  ctx.fillRect(16, ey - 2, 7, 2)
  ctx.fillRect(31, ey - 2, 7, 2)
  ctx.fillRect(18, ey, 3, 3)
  ctx.fillRect(33, ey, 3, 3)
  ctx.fillStyle = BASE.W
  ctx.fillRect(19, ey, 1, 1)
  ctx.fillRect(34, ey, 1, 1)
  ctx.fillStyle = BASE.m
  ctx.fillRect(22, 34, 10, frame ? 4 : 3)
  ctx.fillStyle = BASE.W
  ctx.fillRect(23, 34, 2, 2)
  ctx.fillRect(29, 34, 2, 2)
  // little arms
  ctx.fillStyle = BASE.t
  ctx.fillRect(0, 30 - (frame ? 2 : 0), 3, 2)
  ctx.fillRect(53, 30 - (frame ? 0 : 2), 3, 2)
  return outline(c)
}

function calendar(frame: number, angry: boolean, digits: (ctx: CanvasRenderingContext2D, text: string, x: number, y: number, color: string, scale: number) => void): Sprite {
  const [c, ctx] = canvas(46, 54)
  // pages behind
  ctx.fillStyle = BASE.w
  ctx.fillRect(4, 8, 38, 42)
  ctx.fillStyle = BASE.W
  ctx.fillRect(2, 6, 38, 42)
  // red header
  ctx.fillStyle = angry ? '#ff3b30' : BASE.R
  ctx.fillRect(2, 6, 38, 11)
  // binding rings
  ctx.fillStyle = BASE.Z
  for (let x = 6; x < 38; x += 6) ctx.fillRect(x, 2 + (frame && x % 12 ? 1 : 0), 2, 6)
  digits(ctx, '9.1', 9, 8, BASE.W, 1)
  // big 1 → it's the first day of school
  digits(ctx, '1', 16, 26, BASE.R, 3)
  // angry eyes over the date
  ctx.fillStyle = BASE.K
  ctx.fillRect(7, 20, 6, 2)
  ctx.fillRect(29, 20, 6, 2)
  ctx.fillRect(9, 22, 3, 4)
  ctx.fillRect(30, 22, 3, 4)
  ctx.fillStyle = BASE.m
  ctx.fillRect(8, 42, 26, 3)
  ctx.fillStyle = BASE.W
  for (let x = 9; x < 34; x += 3) ctx.fillRect(x, 42, 1, 2)
  // curled corner
  ctx.fillStyle = BASE.w
  ctx.fillRect(35, 43, 5, 5)
  ctx.clearRect(39, 46, 2, 2)
  // flapping arms
  ctx.fillStyle = BASE.t
  ctx.fillRect(0, 28 + (frame ? -3 : 0), 2, 6)
  ctx.fillRect(42, 28 + (frame ? 0 : -3), 2, 6)
  return outline(c)
}

// ------------------------------------------------------------------ candy balls and floss

function ballFrame(rot: number, crack: number): Sprite {
  const [c, ctx] = canvas(22, 22)
  ctx.fillStyle = BASE.M
  circle(ctx, 11, 11, 9.6)
  ctx.fillStyle = BASE.P
  circle(ctx, 10.5, 10.5, 8.4)
  harden(c)
  // swirl of floss that turns as the ball rolls
  ctx.strokeStyle = BASE.p
  ctx.lineWidth = 1.4
  for (let i = 0; i < 3; i++) {
    const a = rot + (i * Math.PI * 2) / 3
    ctx.beginPath()
    ctx.arc(11 + Math.cos(a) * 3, 11 + Math.sin(a) * 3, 4.2, a, a + 2.2)
    ctx.stroke()
  }
  harden(c, 140)
  ctx.fillStyle = BASE.W
  ctx.fillRect(6, 5, 3, 2)
  ctx.fillRect(5, 7, 2, 2)
  if (crack) {
    ctx.fillStyle = BASE.m
    ctx.fillRect(13, 6, 1, 3)
    ctx.fillRect(14, 9, 1, 2)
    ctx.fillRect(12, 9, 1, 1)
    if (crack > 1) {
      ctx.fillRect(6, 13, 3, 1)
      ctx.fillRect(9, 14, 1, 2)
    }
  }
  return outline(c)
}

/** floss stuck to a pest: level 1–3 of 4, a fluffy dome sized to the pest */
function flossCover(w: number, h: number, level: number): Sprite {
  const [c, ctx] = canvas(w + 8, h + 8)
  const cover = [0, 0.42, 0.72, 0.95][level]
  const cx = (w + 8) / 2
  const bottom = h + 5
  const rx = w / 2 + 2
  const ry = Math.max(4, h * cover + 1)
  ctx.fillStyle = BASE.P
  ctx.beginPath()
  ctx.ellipse(cx, bottom, rx, ry, 0, Math.PI, Math.PI * 2)
  ctx.lineTo(cx + rx, bottom)
  ctx.lineTo(cx - rx, bottom)
  ctx.fill()
  // tufts round the edge of the dome
  const n = 5 + level
  for (let i = 0; i <= n; i++) {
    const a = Math.PI + (i / n) * Math.PI
    circle(ctx, cx + Math.cos(a) * rx, bottom + Math.sin(a) * ry, 2.4 + (i % 2) * 0.8)
  }
  ctx.fillStyle = BASE.p
  for (let i = 1; i < n; i += 2) {
    const a = Math.PI + (i / n) * Math.PI
    circle(ctx, cx + Math.cos(a) * (rx - 2.5), bottom + Math.sin(a) * (ry - 2.5), 1.3)
  }
  harden(c)
  ctx.clearRect(0, bottom + 1, c.width, c.height)
  return outline(c)
}

// ------------------------------------------------------------------ snacks, potions, projectiles

const ITEMS: string[][] = [
  // 0 辣条
  ['..RRRR....', '.RXXXXR...', 'RXOXOXXR..', 'RXXXXOXXR.', '.RXOXXXXR.', '..RXXOXXR.', '...RXXXR..', '....RRR...'],
  // 1 冰棍
  ['..OOOOO...', '.OyOOOOO..', '.OyOOOOO..', '.OOOOOOO..', '.OOOOOOO..', '.ooOOOOo..', '..ooooo...', '....T.....', '....T.....', '....t.....'],
  // 2 西瓜
  ['..........', 'RRRRRRRRRR', '.RRKRRKRR.', '..RRRRRR..', '...WWWW...', '....GG....', '..........'],
  // 3 奶糖
  ['..........', 'L.WWWWWW.L', 'LLWLLWWWLL', 'LLWWWWLWLL', 'L.WWWWWW.L', '..........'],
  // 4 跳跳糖
  ['.CCCCCC.', '.CYYYYC.', '.CYPYPC.', '.CYYYYC.', '.CPYGYC.', '.CYYYYC.', '.CCCCCC.', '.cccccc.'],
  // 5 奖状
  ['YYYYYYYYYY', 'YyyyyyyyyY', 'YykkkkkkyY', 'YyyyyyyyyY', 'YykkkkyyyY', 'YyyyyyyRRY', 'YyyyyyyRRY', 'YYYYYYYYYY'],
  // 6 红汽水 力量
  ['...ww...', '...ww...', '..WWWW..', '..WRRW..', '.WRRRRW.', '.WRyRRW.', '.WRRRRW.', '.WRRRRW.', '..WWWW..'],
  // 7 蓝汽水 速度
  ['...ww...', '...ww...', '..WWWW..', '..WLLW..', '.WLLLLW.', '.WLlLLW.', '.WNLLLW.', '.WNNLLW.', '..WWWW..'],
  // 8 黄汽水 射程
  ['...ww...', '...ww...', '..WWWW..', '..WYYW..', '.WYYYYW.', '.WYyYYW.', '.WOYYYW.', '.WOOYYW.', '..WWWW..'],
  // 9 糖葫芦
  ['....RR..', '...RyRR.', '...RRRR.', '..RRR...', '.RyRR...', '.RRRR...', 'RRR.....', 'T.......', 'T.......'],
  // 10 红包
  ['RRRRRRRR', 'RRrrrrRR', 'RRRYYRRR', 'RRYyyYRR', 'RRRYYRRR', 'RRRRRRRR', 'RRRRRRRR']
]

const PROJ: Record<string, string[]> = {
  chalk: ['wWWWW', 'WWWWw'],
  plane: ['WW..........', 'WWWWW.......', '.wWWWWWWWWW.', '..wwwwWWW...', '.......W....'],
  page: ['RRRRRRRR', 'WWWWWWWW', 'WkWkWkWW', 'WWWWWWWW', 'WkWkWWWW', 'WWWWWWww'],
  wave: ['..YY......', '.Y..Y.....', 'Y....Y..YY', '......YY..']
}

// ------------------------------------------------------------------ pixel font for numbers

const GLYPHS: Record<string, string[]> = {
  '0': ['111', '101', '101', '101', '111'],
  '1': ['010', '110', '010', '010', '111'],
  '2': ['111', '001', '111', '100', '111'],
  '3': ['111', '001', '011', '001', '111'],
  '4': ['101', '101', '111', '001', '001'],
  '5': ['111', '100', '111', '001', '111'],
  '6': ['111', '100', '111', '101', '111'],
  '7': ['111', '001', '010', '010', '010'],
  '8': ['111', '101', '111', '101', '111'],
  '9': ['111', '101', '111', '001', '111'],
  x: ['000', '101', '010', '101', '000'],
  '+': ['000', '010', '111', '010', '000'],
  ':': ['000', '010', '000', '010', '000'],
  '.': ['000', '000', '000', '000', '010'],
  '-': ['000', '000', '111', '000', '000'],
  P: ['110', '101', '110', '100', '100'],
  F: ['111', '100', '110', '100', '100'],
  U: ['101', '101', '101', '101', '111'],
  ' ': ['000', '000', '000', '000', '000']
}

export function textWidth(text: string, scale = 1): number {
  return text.length * 4 * scale - scale
}

/** draw numbers in the chunky 3×5 font, with a dark shadow for readability */
export function digits(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, color: string, scale = 1, shadow: string | null = null): void {
  const draw = (ox: number, oy: number, col: string) => {
    ctx.fillStyle = col
    let cx = ox
    for (const ch of text) {
      const g = GLYPHS[ch] ?? GLYPHS[' ']
      for (let r = 0; r < 5; r++) for (let c = 0; c < 3; c++) if (g[r][c] === '1') ctx.fillRect(cx + c * scale, oy + r * scale, scale, scale)
      cx += 4 * scale
    }
  }
  if (shadow) {
    draw(x + scale, y + scale, shadow)
    draw(x - 1, y, shadow)
    draw(x + 1, y, shadow)
    draw(x, y - 1, shadow)
    draw(x, y + 1, shadow)
  }
  draw(x, y, color)
}

// ------------------------------------------------------------------ the whole sheet

export interface Sheet {
  kids: [KidSprites, KidSprites]
  pests: Record<string, [Sprite, Sprite][]>
  white: Record<string, [Sprite, Sprite][]>
  bosses: Record<string, [Sprite, Sprite][]>
  bossesAngry: Record<string, [Sprite, Sprite][]>
  bossFlash: Record<string, Sprite>
  balls: Sprite[]
  ballsCrack: Sprite[][]
  floss: (w: number, h: number, level: number) => Sprite
  items: Sprite[]
  proj: Record<string, [Sprite, Sprite]>
}

export const KID_COLORS = [
  { suit: ['#e8414f', '#a82b45'] as [string, string], hair: ['#2b1d33', '#56406b'] as [string, string], name: '小满', css: '#ff6b7a' },
  { suit: ['#3f7fe8', '#2a4fa8'] as [string, string], hair: ['#3a2418', '#7a4a2e'] as [string, string], name: '阿豆', css: '#6aa8ff' }
]

export function buildSheet(): Sheet {
  const both = (s: Sprite): [Sprite, Sprite] => [s, flip(s)]
  const pests: Record<string, [Sprite, Sprite][]> = {
    book: BOOK.map((g) => both(grid(g))),
    clock: CLOCK.map((g) => both(grid(g))),
    mosquito: MOSQUITO.map((g) => both(grid(g))),
    chalk: CHALK.map((g) => both(grid(g))),
    coil: [0, 1, 2, 3].map((i) => both(coil((i * Math.PI) / 2))),
    slipper: [both(grid(SLIPPER))]
  }
  const white: Record<string, [Sprite, Sprite][]> = {}
  for (const k of Object.keys(pests)) white[k] = pests[k].map(([a, b]) => [tint(a, '#ffffff'), tint(b, '#ffffff')])
  const bosses: Record<string, [Sprite, Sprite][]> = {
    bigclock: [0, 1].map((f) => both(bigClock(f, false))),
    bookpile: [0, 1].map((f) => both(bookPile(f, false))),
    calendar: [0, 1].map((f) => both(calendar(f, false, digits)))
  }
  const bossesAngry: Record<string, [Sprite, Sprite][]> = {
    bigclock: [0, 1].map((f) => both(bigClock(f, true))),
    bookpile: [0, 1].map((f) => both(bookPile(f, true))),
    calendar: [0, 1].map((f) => both(calendar(f, true, digits)))
  }
  const bossFlash: Record<string, Sprite> = {}
  for (const k of Object.keys(bosses)) bossFlash[k] = tint(bosses[k][0][0], '#ffffff', 0.85)
  const balls: Sprite[] = []
  const ballsCrack: Sprite[][] = [[], []]
  for (let i = 0; i < 8; i++) {
    balls.push(ballFrame((i / 8) * Math.PI * 2, 0))
  }
  ballsCrack[0].push(ballFrame(0, 1))
  ballsCrack[1].push(ballFrame(0, 2))
  const flossCache = new Map<string, Sprite>()
  const floss = (w: number, h: number, level: number): Sprite => {
    const key = `${w}x${h}x${level}`
    let s = flossCache.get(key)
    if (!s) {
      s = flossCover(w, h, level)
      flossCache.set(key, s)
    }
    return s
  }
  const proj: Record<string, [Sprite, Sprite]> = {}
  for (const k of Object.keys(PROJ)) proj[k] = both(grid(PROJ[k]))
  return {
    kids: [kid(HEAD_GIRL, KID_COLORS[0].suit, KID_COLORS[0].hair), kid(HEAD_BOY, KID_COLORS[1].suit, KID_COLORS[1].hair)],
    pests,
    white,
    bosses,
    bossesAngry,
    bossFlash,
    balls,
    ballsCrack,
    floss,
    items: ITEMS.map((g) => grid(g)),
    proj
  }
}
