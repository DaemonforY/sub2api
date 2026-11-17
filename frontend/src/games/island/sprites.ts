// All in-game sprites for 椰风岛大冒险, rasterised at native resolution from simple shapes.
// Each body part is shaded and outlined on its own layer before compositing, which gives the
// inner contour lines of hand-made pixel art while keeping every frame consistent.
import { Pix, rgb, mix, shadowOf, lightOf, OUTLINE, type Color } from './pix'

export interface Sheet {
  r: HTMLCanvasElement[] // facing right
  l: HTMLCanvasElement[] // facing left
  wr: HTMLCanvasElement[] // white flash, facing right
  wl: HTMLCanvasElement[]
  w: number
  h: number
  ox: number // anchor x inside the cell (bottom-centre for walkers)
  oy: number
}

export function sheet(frames: Pix[], ox?: number, oy?: number): Sheet {
  const w = frames[0].w
  const h = frames[0].h
  const white = rgb('#ffffff')
  return {
    r: frames.map((f) => f.canvas()),
    l: frames.map((f) => f.flipX().canvas()),
    wr: frames.map((f) => f.tint(white).canvas()),
    wl: frames.map((f) => f.tint(white).flipX().canvas()),
    w,
    h,
    ox: ox ?? w / 2,
    oy: oy ?? h
  }
}

const C = (h: string) => rgb(h)

/** Shade + outline a part layer, then composite onto `dst`. */
function part(dst: Pix, w: number, h: number, paint: (p: Pix) => void, opts: { shade?: boolean; outline?: Color | null } = {}) {
  const p = new Pix(w, h)
  paint(p)
  if (opts.shade !== false) p.shade()
  if (opts.outline !== null) p.outline(opts.outline ?? OUTLINE)
  dst.draw(p)
}

/** Paint `fn` but keep only pixels inside `mask`'s opaque area. */
function clipped(dst: Pix, mask: Pix, fn: (p: Pix) => void) {
  const t = new Pix(dst.w, dst.h)
  fn(t)
  for (let i = 0; i < t.d.length; i++) if (t.d[i] && mask.d[i]) dst.d[i] = t.d[i]
}

function spiral(p: Pix, cx: number, cy: number, r1: number, turns: number, c: Color, t = 1) {
  let px = 0
  let py = 0
  for (let i = 0; i <= 48; i++) {
    const k = i / 48
    const a = k * turns * Math.PI * 2 - Math.PI / 2
    const r = r1 * (1 - k * 0.92)
    const x = cx + Math.cos(a) * r
    const y = cy + Math.sin(a) * r * 0.9
    if (i) p.line(px, py, x, y, c, t)
    px = x
    py = y
  }
}

// ───────────────────────────── heroes ─────────────────────────────

export interface HeroLook {
  hair: Color
  top: Color
  shorts: Color
  accent: Color
  skin: Color
  style: 0 | 1 // 0 = 阿浪 spiky hair + bandana, 1 = 小椰 buns + flower
}

export const HERO_LOOKS: HeroLook[] = [
  { hair: C('#2c2546'), top: C('#1fb5a5'), shorts: C('#ff8a26'), accent: C('#e8323c'), skin: C('#ffc58e'), style: 0 },
  { hair: C('#d9692b'), top: C('#ffd23a'), shorts: C('#45c25b'), accent: C('#ff5fa2'), skin: C('#ffcb98'), style: 1 }
]

interface Pose {
  bob: number
  legB: [number, number]
  legF: [number, number]
  armB: number
  armF: number
  bendF?: number
  bendB?: number
  eye: 0 | 1 | 2 | 3 // open, blink, hurt (x), happy (^)
  mouth: 0 | 1 | 2 // smile, open, grit
  tail: number
  headDy?: number
  lean?: number
}

const EYE_DARK = C('#20142c')
const WHITE = C('#ffffff')
const MOUTH = C('#a82f3c')
const BLUSH = C('#ff8f8f')

function drawHead(dst: Pix, look: HeroLook, cx: number, cy: number, pose: Pose) {
  const W = dst.w
  const H = dst.h
  // back accessories first (bandana tails / buns) on their own layers
  if (look.style === 0) {
    const wv = Math.sin(pose.tail * Math.PI * 2)
    const wv2 = Math.sin(pose.tail * Math.PI * 2 + 1.7)
    part(dst, W, H, (p) => {
      p.poly([cx - 6, cy - 3.5, cx - 11.5, cy - 4.5 + wv * 1.5, cx - 12, cy - 2.6 + wv * 1.5, cx - 6.5, cy - 1.6], look.accent)
      p.poly([cx - 6.5, cy - 2.4, cx - 10.5, cy + 0.5 + wv2 * 1.4, cx - 10.6, cy + 2.2 + wv2 * 1.4, cx - 6, cy - 0.6], shadowOf(look.accent, 0.2))
    })
  } else {
    part(dst, W, H, (p) => {
      p.circle(cx - 3.6, cy - 7.4, 3.1, look.hair)
      p.circle(cx + 2.6, cy - 8.2, 2.8, look.hair)
      // little tail of hair at the nape
      p.poly([cx - 5.5, cy - 1, cx - 8.5, cy + 3.5, cx - 5.5, cy + 3], look.hair)
    })
  }
  // head
  const head = new Pix(W, H)
  head.circle(cx, cy, 6.7, look.hair)
  if (look.style === 0) {
    head.poly([cx - 5.5, cy - 3.5, cx - 10.5, cy - 6, cx - 6.5, cy - 0.5], look.hair)
    head.poly([cx - 4.5, cy - 5, cx - 6, cy - 10.5, cx - 1.5, cy - 6.4], look.hair)
    head.poly([cx - 1.5, cy - 6.2, cx + 1.5, cy - 11, cx + 3, cy - 5.6], look.hair)
    head.poly([cx + 2.5, cy - 5.6, cx + 7.5, cy - 8, cx + 5.6, cy - 3], look.hair)
  }
  const mask = new Pix(W, H)
  mask.circle(cx, cy, 6.7, 1)
  // face: front-lower part of the head is skin
  clipped(head, mask, (p) => {
    p.ellipse(cx + 1.9, cy + 1.6, 5.4, 5.3, look.skin)
  })
  // ear
  head.circle(cx - 1.2, cy + 1.2, 1.5, look.skin)
  head.set(cx - 1.4, cy + 1.2, shadowOf(look.skin, 0.25))
  // bangs over the forehead
  head.poly([cx - 1, cy - 5.5, cx + 6.8, cy - 3.5, cx + 3.2, cy - 1.2, cx + 1.5, cy - 2.4, cx - 0.5, cy - 0.5], look.hair)
  if (look.style === 1) head.poly([cx + 4.5, cy - 3, cx + 6.9, cy + 1.5, cx + 5.4, cy + 3.6, cx + 4.6, cy], look.hair)
  head.shade(1, 0.22, 0.3)
  // bandana band across the forehead (style 0), drawn after shading so it stays a clean stripe
  if (look.style === 0) {
    clipped(head, mask.tint(1), (p) => {
      p.rect(cx - 8, cy - 3.6, 16, 2, look.accent)
      p.rect(cx - 8, cy - 1.6, 16, 1, shadowOf(look.accent, 0.3))
    })
    head.circle(cx - 6.2, cy - 2.4, 1.6, shadowOf(look.accent, 0.1))
  }
  head.outline()
  dst.draw(head)
  // flower on the bun (style 1) + hair ties and swirl highlights so the buns read as buns
  if (look.style === 1) {
    const hl = lightOf(look.hair, 0.45)
    dst.set(cx - 4.6, cy - 9.2, hl)
    dst.set(cx - 3.6, cy - 9.6, hl)
    dst.set(cx + 2.2, cy - 10, hl)
    dst.set(cx + 3.2, cy - 10, hl)
    dst.rect(cx + 1.2, cy - 6.4, 3, 1, look.accent)
    const fx = cx - 4.2
    const fy = cy - 7.4
    const pet = look.accent
    const P: Record<string, Color> = { p: pet, q: lightOf(pet, 0.35), y: C('#ffe14d'), k: OUTLINE, g: C('#2f9e4a') }
    dst.stamp(['.kk.kk.', 'kqpkppk', 'kpqyppk', '.kpypk.', 'kppppk.', '.kkgk..', '...g...'], Math.round(fx - 3), Math.round(fy - 3), P)
  }
  // face details
  const ex = Math.round(cx + 3)
  const ey = Math.round(cy - 0.5)
  if (pose.eye === 0) {
    dst.rect(ex, ey, 2, 3, EYE_DARK)
    dst.set(ex + 1, ey, WHITE)
    dst.rect(ex - 1, ey - 1, 3, 1, EYE_DARK) // brow
  } else if (pose.eye === 1) {
    dst.rect(ex - 1, ey + 2, 3, 1, EYE_DARK)
  } else if (pose.eye === 2) {
    dst.set(ex - 1, ey, EYE_DARK)
    dst.set(ex + 1, ey, EYE_DARK)
    dst.set(ex, ey + 1, EYE_DARK)
    dst.set(ex - 1, ey + 2, EYE_DARK)
    dst.set(ex + 1, ey + 2, EYE_DARK)
  } else {
    dst.set(ex - 1, ey + 2, EYE_DARK)
    dst.set(ex, ey + 1, EYE_DARK)
    dst.set(ex + 1, ey + 2, EYE_DARK)
  }
  dst.set(ex - 1, ey + 3, BLUSH)
  dst.set(ex - 2, ey + 3, BLUSH)
  const mx = Math.round(cx + 4)
  const my = Math.round(cy + 3.4)
  if (pose.mouth === 0) {
    dst.set(mx, my, MOUTH)
    dst.set(mx + 1, my - 1, MOUTH)
  } else if (pose.mouth === 1) {
    dst.rect(mx, my - 1, 2, 2, MOUTH)
    dst.set(mx, my, C('#ff7a7a'))
  } else {
    dst.rect(mx - 1, my, 3, 1, MOUTH)
  }
}

function heroFrame(look: HeroLook, pose: Pose): Pix {
  const W = 26
  const H = 30
  const p = new Pix(W, H)
  const bob = pose.bob
  const lean = pose.lean ?? 0
  const hipY = 22.4 + bob
  const shY = 16.6 + bob
  const skin = look.skin

  const leg = (hx: number, a: [number, number], front: boolean) => {
    part(p, W, H, (q) => {
      const kx = hx + Math.sin(a[0]) * 3.4
      const ky = hipY + Math.cos(a[0]) * 3.4
      const a2 = a[0] + a[1]
      const fx = kx + Math.sin(a2) * 3.3
      const fy = ky + Math.cos(a2) * 3.3
      q.line(kx, ky, fx, fy, front ? skin : shadowOf(skin, 0.12), 2.6)
      q.ellipse(fx + 1.1, fy + 0.4, 2.1, 1.2, front ? skin : shadowOf(skin, 0.12))
      q.line(hx, hipY - 0.5, kx, ky, look.shorts, 3.4)
    })
  }
  const arm = (sx: number, ang: number, bend: number, front: boolean) => {
    part(p, W, H, (q) => {
      const ex = sx + Math.sin(ang) * 2.8
      const ey = shY + Math.cos(ang) * 2.8
      const a2 = ang + bend
      const hx = ex + Math.sin(a2) * 2.6
      const hy = ey + Math.cos(a2) * 2.6
      const c = front ? skin : shadowOf(skin, 0.15)
      q.line(sx, shY, ex, ey, c, 2.2)
      q.line(ex, ey, hx, hy, c, 2.2)
      q.circle(hx, hy, 1.4, c)
    })
  }

  arm(11.6 + lean * 0.4, pose.armB, pose.bendB ?? 0.5, false)
  leg(12.1, pose.legB, false)
  // torso: tank top + shorts
  part(p, W, H, (q) => {
    const tx = 13 + lean * 0.4
    q.ellipse(tx, shY + 2.6, 3.7, 3.6, look.top)
    q.rect(tx - 3.6, shY + 2, 7.2, 3.6, look.top)
    q.rect(tx - 3.8, hipY - 2.6, 7.6, 3.2, look.shorts)
    // waist knot / stripe detail
    q.rect(tx - 3.6, hipY - 2.6, 7.2, 1, shadowOf(look.shorts, 0.25))
  })
  // a flower print dot on the shorts (阿浪) / a stripe on the top (小椰)
  if (look.style === 0) p.set(Math.round(14 + lean * 0.4), Math.round(hipY - 0.6), C('#fff1d6'))
  else p.rect(Math.round(10 + lean * 0.4), Math.round(shY + 3), 6, 1, lightOf(look.top, 0.45))
  leg(14, pose.legF, true)
  arm(14.6 + lean * 0.4, pose.armF, pose.bendF ?? 0.5, true)
  drawHead(p, look, 13.4 + lean, 9.4 + bob + (pose.headDy ?? 0), pose)
  return p
}

export const HERO_FRAMES = ['idle0', 'idle1', 'run0', 'run1', 'run2', 'run3', 'run4', 'run5', 'jump', 'fall', 'throw0', 'throw1', 'trip', 'dead', 'ride', 'win', 'blink'] as const
export type HeroFrame = (typeof HERO_FRAMES)[number]

function heroPoses(): Record<HeroFrame, Pose> {
  const base: Pose = { bob: 0, legB: [-0.12, 0], legF: [0.14, 0], armB: -0.2, armF: 0.25, eye: 0, mouth: 0, tail: 0 }
  const run = (i: number): Pose => {
    const ph = (i / 6) * Math.PI * 2
    const s = Math.sin(ph)
    const kneeF = Math.cos(ph) < 0 ? -1.3 * -Math.cos(ph) : -0.15
    const kneeB = Math.cos(ph + Math.PI) < 0 ? -1.3 * -Math.cos(ph + Math.PI) : -0.15
    return {
      ...base,
      bob: i % 3 === 0 ? 1 : 0,
      legF: [0.95 * s, kneeF],
      legB: [-0.95 * s, kneeB],
      armF: -1.0 * s + 0.2,
      armB: 1.0 * s - 0.1,
      bendF: 1.1,
      bendB: 1.1,
      mouth: 0,
      tail: i / 6,
      lean: 1,
      headDy: 0
    }
  }
  return {
    idle0: { ...base },
    idle1: { ...base, bob: 1, armF: 0.32, armB: -0.28, tail: 0.3 },
    blink: { ...base, eye: 1 },
    run0: run(0),
    run1: run(1),
    run2: run(2),
    run3: run(3),
    run4: run(4),
    run5: run(5),
    jump: { ...base, bob: -1, legF: [1.2, -1.9], legB: [-0.25, 0.7], armF: 2.6, armB: -1.1, bendF: 0.3, mouth: 1, tail: 0.6, lean: 1 },
    fall: { ...base, bob: 0, legF: [0.45, -0.4], legB: [-0.4, 0.3], armF: 2.0, armB: -2.1, bendF: 0.4, bendB: -0.3, mouth: 1, tail: 0.85 },
    throw0: { ...base, armF: -2.6, bendF: -0.7, armB: 0.6, lean: -1, mouth: 2, legF: [0.35, 0], legB: [-0.35, 0], tail: 0.2 },
    throw1: { ...base, armF: 1.65, bendF: 0.05, armB: -0.9, lean: 2, mouth: 1, legF: [0.45, 0], legB: [-0.45, 0.1], tail: 0.5 },
    trip: { ...base, lean: 3, bob: 1, armF: 1.4, armB: 1.0, eye: 2, mouth: 1, legF: [0.2, 0], legB: [-1.2, -0.4], headDy: 1 },
    dead: { ...base, armF: 2.9, armB: -2.9, bendF: 0.2, bendB: -0.2, eye: 2, mouth: 1, legF: [0.3, 0], legB: [-0.3, 0] },
    ride: { ...base, bob: 2, legF: [0.7, -1.1], legB: [-0.6, 0.9], armF: 1.3, armB: -1.5, bendF: 0.3, mouth: 1, tail: 0.7, lean: 1 },
    win: { ...base, armF: 2.9, armB: -2.7, bendF: 0.1, eye: 3, mouth: 1, bob: -1, legF: [0.25, 0], legB: [-0.25, 0] }
  }
}

export type HeroSheet = Sheet & { index: Record<HeroFrame, number> }

function heroSheet(look: HeroLook): HeroSheet {
  const poses = heroPoses()
  const frames = HERO_FRAMES.map((f) => heroFrame(look, poses[f]))
  const index = Object.fromEntries(HERO_FRAMES.map((f, i) => [f, i])) as Record<HeroFrame, number>
  return { ...sheet(frames, 13, 30), index }
}

/** Just the head, for HUD portraits and the revive bubble. */
function heroHead(look: HeroLook): Pix {
  const p = new Pix(22, 22)
  drawHead(p, look, 10.5, 12, { bob: 0, legB: [0, 0], legF: [0, 0], armB: 0, armF: 0, eye: 0, mouth: 0, tail: 0.2 })
  return p
}

// ───────────────────────────── enemies ─────────────────────────────

const RED = C('#e2463c')
const CREAM = C('#f5dcb0')
const BROWN = C('#8a5a33')

function crabFrames(): Pix[] {
  return [0, 1].map((f) => {
    const p = new Pix(22, 18)
    // legs
    part(p, 22, 18, (q) => {
      for (let i = 0; i < 3; i++) {
        const lx = 9 + i * 3
        const off = (i + f) % 2 ? 1 : -1
        q.line(lx, 13, lx + off, 16.5, shadowOf(RED, 0.2), 1.4)
      }
    })
    // spiral shell
    part(p, 22, 18, (q) => {
      q.ellipse(8.5, 8.5, 7, 6.6, CREAM)
      q.poly([2, 9, 1, 14.5, 7, 12], CREAM)
    })
    spiral(p, 8.5, 8.5, 5.2, 1.6, BROWN)
    p.set(5, 5, BROWN)
    p.set(4, 11, BROWN)
    p.set(12, 11, BROWN)
    // body + eyes + claw
    part(p, 22, 18, (q) => {
      q.ellipse(14.5, 12, 4.4, 3.2, RED)
      q.line(15, 10, 15, 6 + f * 0.5, RED, 1.2)
      q.line(17.4, 10, 18.4, 6.2 - f * 0.5, RED, 1.2)
    })
    part(p, 22, 18, (q) => {
      q.circle(15, 5.5 + f * 0.5, 1.6, WHITE)
      q.circle(18.6, 5.7 - f * 0.5, 1.6, WHITE)
    })
    p.set(15, Math.round(5.5 + f * 0.5), EYE_DARK)
    p.set(19, Math.round(5.7 - f * 0.5), EYE_DARK)
    part(p, 22, 18, (q) => {
      q.circle(19, 12.5 - f, 2.6, RED)
      q.poly([19.5, 10 - f, 22, 9 - f, 21, 12 - f], RED)
    })
    return p
  })
}

function gullFrames(): Pix[] {
  const out: Pix[] = []
  for (let f = 0; f < 3; f++) {
    const p = new Pix(22, 16)
    const wing = C('#b7c4d8')
    if (f === 0) part(p, 22, 16, (q) => q.poly([7, 8, 3, 1, 9, 3, 13, 8], wing))
    part(p, 22, 16, (q) => {
      q.ellipse(10, 9, 6.5, 3.2, WHITE)
      q.circle(16, 7, 2.8, WHITE)
      q.poly([3, 8, 0, 6, 1, 10], C('#9aa8bf'))
    })
    p.poly([18.5, 6.5, 22, 7.5, 18.5, 8.6], C('#ffa62b'))
    p.set(17, 6, EYE_DARK)
    if (f === 1) part(p, 22, 16, (q) => q.poly([7, 9, 4, 15, 10, 13, 13, 9], wing))
    if (f === 2) part(p, 22, 16, (q) => q.poly([6, 7, 1, 10, 7, 11, 12, 8], wing))
    out.push(p)
  }
  return out
}

function fishFrames(): Pix[] {
  const blue = C('#3f8fe0')
  const base = new Pix(18, 14)
  part(base, 18, 14, (q) => {
    q.poly([2, 3, 5, 7, 2, 11], C('#2f6fc0'))
    q.ellipse(10, 7, 6.5, 4, blue)
    q.poly([8, 3.5, 11, 0.5, 13, 3.5], C('#ffb03a'))
  })
  base.rect(7, 8, 7, 1, lightOf(blue, 0.5))
  base.circle(13.5, 6, 1.6, WHITE)
  base.set(14, 6, EYE_DARK)
  return [base.rotate(-0.6), base.rotate(0.7)]
}

function frogFrames(): Pix[] {
  const g = C('#43b84a')
  return [0, 1].map((f) => {
    const p = new Pix(20, 16)
    if (f === 1) part(p, 20, 16, (q) => q.line(6, 11, 1, 15, shadowOf(g, 0.15), 2.4))
    part(p, 20, 16, (q) => {
      q.ellipse(10, f ? 9 : 11, 6.8, f ? 4.4 : 4, g)
      if (!f) q.ellipse(6, 13.6, 3.4, 1.8, shadowOf(g, 0.12))
      q.ellipse(14.6, 14.4, 2.2, 1.4, g)
    })
    p.ellipse(11, f ? 11 : 12.6, 4, 1.6, C('#f2f0b8'))
    part(p, 20, 16, (q) => {
      q.circle(9, f ? 4.6 : 6.6, 2.6, g)
      q.circle(14, f ? 4.6 : 6.6, 2.6, g)
    })
    const ey = f ? 4 : 6
    p.rect(9, ey, 2, 2, EYE_DARK)
    p.rect(14, ey, 2, 2, EYE_DARK)
    p.set(10, ey, WHITE)
    p.set(15, ey, WHITE)
    p.line(12, f ? 9 : 10, 16, f ? 9 : 10, shadowOf(g, 0.45))
    return p
  })
}

function spiderFrames(): Pix[] {
  const pu = C('#7a4bc8')
  return [0, 1].map((f) => {
    const p = new Pix(18, 14)
    part(p, 18, 14, (q) => {
      for (let i = 0; i < 4; i++) {
        const k = i - 1.5
        const sw = f ? 1 : -1
        q.line(9, 7, 9 - 4 - Math.abs(k), 4 + i * 2 + sw * (i % 2), shadowOf(pu, 0.4), 1)
        q.line(9, 7, 9 + 4 + Math.abs(k), 4 + i * 2 - sw * (i % 2), shadowOf(pu, 0.4), 1)
      }
    }, { shade: false })
    part(p, 18, 14, (q) => q.circle(9, 7, 4.4, pu))
    p.rect(7, 7, 1, 2, C('#ff4d5a'))
    p.rect(10, 7, 1, 2, C('#ff4d5a'))
    p.set(8, 4, lightOf(pu, 0.6))
    return p
  })
}

function spikyFrames(): Pix[] {
  const br = C('#9b6a3a')
  const base = new Pix(18, 18)
  part(base, 18, 18, (q) => {
    for (let i = 0; i < 8; i++) {
      const a = (i / 8) * Math.PI * 2
      const cx = 9 + Math.cos(a) * 5
      const cy = 9 + Math.sin(a) * 5
      q.poly([cx + Math.cos(a + 1.6) * 2, cy + Math.sin(a + 1.6) * 2, 9 + Math.cos(a) * 8.6, 9 + Math.sin(a) * 8.6, cx + Math.cos(a - 1.6) * 2, cy + Math.sin(a - 1.6) * 2], C('#e8d7a8'))
    }
    q.circle(9, 9, 5.6, br)
  })
  base.rect(6, 8, 2, 2, EYE_DARK)
  base.rect(11, 8, 2, 2, EYE_DARK)
  base.line(5, 6, 8, 7, EYE_DARK)
  base.line(13, 6, 10, 7, EYE_DARK)
  return [0, 1, 2, 3].map((i) => base.rotate((i * Math.PI) / 2))
}

function batFrames(): Pix[] {
  const b = C('#7d2a4a')
  return [0, 1].map((f) => {
    const p = new Pix(22, 14)
    part(p, 22, 14, (q) => {
      if (f === 0) {
        q.poly([10, 6, 1, 1, 3, 6, 1, 9, 9, 9], shadowOf(b, 0.1))
        q.poly([12, 6, 21, 1, 19, 6, 21, 9, 13, 9], shadowOf(b, 0.1))
      } else {
        q.poly([10, 7, 2, 12, 4, 8, 1, 6, 9, 5], shadowOf(b, 0.1))
        q.poly([12, 7, 20, 12, 18, 8, 21, 6, 13, 5], shadowOf(b, 0.1))
      }
    })
    part(p, 22, 14, (q) => {
      q.circle(11, 7, 3.4, b)
      q.poly([8.5, 4.5, 9, 1.5, 10.5, 4], b)
      q.poly([13.5, 4.5, 13, 1.5, 11.5, 4], b)
    })
    p.set(10, 7, C('#ffd23a'))
    p.set(12, 7, C('#ffd23a'))
    return p
  })
}

function flameFrames(): Pix[] {
  return [0, 1, 2].map((f) => {
    const p = new Pix(16, 20)
    const sway = [0, 1.2, -1][f]
    part(p, 16, 20, (q) => {
      q.poly([8 + sway, 0.5 + (f % 2), 13.5, 10, 13, 15, 8, 19, 3, 15, 2.5, 10], C('#ff7a1f'))
    }, { shade: false })
    p.poly([8 + sway * 0.6, 5, 11, 11, 10.5, 15.5, 8, 17.5, 5.5, 15.5, 5, 11], C('#ffd23a'))
    p.poly([8, 11, 9.4, 14, 8, 16, 6.6, 14], C('#fff6c0'))
    p.rect(6, 11, 1, 2, EYE_DARK)
    p.rect(9, 11, 1, 2, EYE_DARK)
    return p
  })
}

function boulderFrames(): Pix[] {
  const g = C('#7c6a7e')
  const base = new Pix(24, 24)
  part(base, 24, 24, (q) => {
    q.circle(12, 12, 10.5, g)
  })
  base.line(6, 9, 10, 12, shadowOf(g, 0.5))
  base.line(10, 12, 9, 16, shadowOf(g, 0.5))
  base.line(14, 5, 16, 9, shadowOf(g, 0.5))
  base.line(15, 15, 19, 14, shadowOf(g, 0.5))
  base.set(8, 6, lightOf(g, 0.5))
  base.set(7, 7, lightOf(g, 0.5))
  return [0, 1, 2, 3].map((i) => base.rotate((i * Math.PI) / 2))
}

// ───────────────────────────── props & items ─────────────────────────────

function campfireFrames(): Pix[] {
  return [0, 1, 2].map((f) => {
    const p = new Pix(18, 20)
    const sway = [0, 1.4, -1.2][f]
    part(p, 18, 20, (q) => {
      q.poly([9 + sway, 1 + f, 15, 11, 14, 16, 9, 17, 4, 16, 3, 11], C('#ff5a1f'))
    }, { shade: false })
    p.poly([9 + sway * 0.6, 6, 12.5, 12, 11.5, 16, 9, 16.5, 6.5, 16, 5.5, 12], C('#ffb52b'))
    p.poly([9, 11, 10.5, 14, 9, 16, 7.5, 14], C('#fff3b0'))
    part(p, 18, 20, (q) => {
      q.line(2, 18.5, 15, 15.5, BROWN, 3)
      q.line(3, 15.5, 16, 18.5, shadowOf(BROWN, 0.15), 3)
    })
    return p
  })
}

function rockFrame(): Pix {
  const g = C('#8e8a9e')
  const p = new Pix(16, 11)
  part(p, 16, 11, (q) => {
    q.ellipse(8, 7, 6.5, 4, g)
    q.ellipse(6, 5.5, 3.6, 3, g)
  })
  p.set(5, 4, lightOf(g, 0.6))
  return p
}

function clamFrames(): Pix[] {
  const shell = C('#f7e3c4')
  const spot = C('#c4703c')
  const make = (open: number, crack: boolean) => {
    const p = new Pix(24, 22)
    // bottom half
    part(p, 24, 22, (q) => {
      q.ellipse(12, 15, 10, 5.5, shell)
      q.rect(2, 14.5, 20, 1, shadowOf(shell, 0.2))
    })
    if (open > 0) {
      p.ellipse(12, 15, 7, 2.2, C('#ffe9a8'))
      p.circle(12, 14, 2.2, C('#ffffff'))
    }
    // top half (lid), lifted when open
    part(p, 24, 22, (q) => {
      q.ellipse(12, 12 - open, 10, 7.5, shell)
      q.rect(1, 12 - open, 22, 3, 0)
    })
    for (const [sx, sy] of [
      [7, 7],
      [12, 5],
      [16.5, 8],
      [9.5, 10],
      [15, 11]
    ]) {
      p.circle(sx, sy - open, 1.4, spot)
    }
    for (let i = 0; i < 5; i++) p.line(12, 12 - open, 4 + i * 4, 6 - open + Math.abs(i - 2), shadowOf(shell, 0.15))
    if (crack) {
      p.line(9, 5, 11, 8, OUTLINE)
      p.line(11, 8, 10, 11, OUTLINE)
      p.line(14, 6, 13, 9, OUTLINE)
    }
    return p
  }
  return [make(0, false), make(0, true), make(5, false)]
}

function boomFrames(gold: boolean): Pix[] {
  const wood = gold ? C('#ffcf2e') : C('#b8743a')
  const band = gold ? C('#fff6b0') : C('#ffd27a')
  const base = new Pix(14, 14)
  part(base, 14, 14, (q) => {
    q.line(2.5, 3, 7, 7.5, wood, 3.2)
    q.line(7, 7.5, 2.5, 12, wood, 3.2)
  })
  base.set(4, 4, band)
  base.set(5, 5, band)
  base.set(4, 10, band)
  base.set(5, 9, band)
  return [0, 1, 2, 3].map((i) => base.rotate((i * Math.PI) / 2))
}

function boardFrame(): Pix {
  const p = new Pix(28, 9)
  const b = C('#38b6ff')
  part(p, 28, 9, (q) => {
    q.ellipse(14, 4, 13, 2.6, b)
    q.poly([24, 2.5, 27.5, 1, 26, 4.5], b)
  })
  p.rect(4, 3, 18, 1, C('#ffffff'))
  p.rect(12, 2, 3, 3, C('#ff5fa2'))
  p.set(13, 3, C('#ffe14d'))
  // little wheels
  p.circle(7, 7.5, 1.4, C('#3a2a4a'))
  p.circle(20, 7.5, 1.4, C('#3a2a4a'))
  return p
}

function fairyFrames(): Pix[] {
  return [0, 1].map((f) => {
    const p = new Pix(16, 16)
    part(p, 16, 16, (q) => {
      q.ellipse(4, f ? 5 : 7, 3, 2.2, C('#c9f3ff'))
      q.ellipse(12, f ? 5 : 7, 3, 2.2, C('#c9f3ff'))
    }, { outline: C('#5aa9d6') })
    const pts: number[] = []
    for (let i = 0; i < 10; i++) {
      const a = -Math.PI / 2 + (i / 10) * Math.PI * 2
      const r = i % 2 ? 2.6 : 5.6
      pts.push(8 + Math.cos(a) * r, 9 + Math.sin(a) * r)
    }
    part(p, 16, 16, (q) => q.poly(pts, C('#ffe14d')))
    p.set(7, 9, EYE_DARK)
    p.set(9, 9, EYE_DARK)
    p.set(8, 11, MOUTH)
    return p
  })
}

function cloudFrames(): Pix[] {
  const g = C('#6d6a86')
  return [0, 1].map((f) => {
    const p = new Pix(22, 14)
    part(p, 22, 14, (q) => {
      q.circle(7, 8, 4.6, g)
      q.circle(12, 6, 5.4, g)
      q.circle(16.5, 8.4, 4, g)
      q.rect(5, 8, 13, 4.6, g)
    })
    p.line(8, 6 + f, 10, 7 + f, EYE_DARK)
    p.line(15, 6 + f, 13, 7 + f, EYE_DARK)
    p.set(9, 8 + f, EYE_DARK)
    p.set(14, 8 + f, EYE_DARK)
    p.line(10, 11, 13, 11, EYE_DARK)
    return p
  })
}

function basketFrame(): Pix {
  const p = new Pix(18, 16)
  part(p, 18, 16, (q) => {
    q.circle(6, 6, 3, C('#ffd23a'))
    q.circle(11, 5, 3, C('#ff7a3a'))
    q.circle(9, 3.5, 2.6, C('#6fd14a'))
  })
  part(p, 18, 16, (q) => {
    q.poly([1.5, 8, 16.5, 8, 14, 15, 4, 15], C('#c9893f'))
  })
  for (let x = 4; x < 15; x += 3) p.line(x, 9, x - 0.5, 14, shadowOf(C('#c9893f'), 0.35))
  p.line(3, 11, 15, 11, shadowOf(C('#c9893f'), 0.35))
  return p
}

function sunFruitFrames(): Pix[] {
  return [0, 1].map((f) => {
    const p = new Pix(24, 24)
    for (let i = 0; i < 8; i++) {
      const a = (i / 8) * Math.PI * 2 + f * 0.4
      p.line(12 + Math.cos(a) * 7, 12 + Math.sin(a) * 7, 12 + Math.cos(a) * 11, 12 + Math.sin(a) * 11, C('#ffe680'), 2)
    }
    part(p, 24, 24, (q) => {
      q.circle(12, 13, 6.4, C('#ff9a1f'))
    })
    p.circle(10, 11, 2, C('#ffe680'))
    part(p, 24, 24, (q) => q.poly([12, 7, 16, 3, 17, 6, 13, 8], C('#4cc35a')))
    return p
  })
}

export const FRUIT_NAMES = ['banana', 'mango', 'pineapple', 'melon', 'grapes', 'coconut'] as const

function fruitFrames(): Pix[] {
  const out: Pix[] = []
  // banana
  {
    const p = new Pix(14, 14)
    part(p, 14, 14, (q) => {
      q.poly([2, 3, 4, 2, 5, 7, 9, 10, 12.5, 9.5, 12, 11.5, 8, 13, 3.5, 10], C('#ffd93b'))
    })
    p.set(2, 2, BROWN)
    p.set(3, 2, BROWN)
    out.push(p)
  }
  // mango
  {
    const p = new Pix(14, 14)
    part(p, 14, 14, (q) => q.ellipse(7, 8, 5, 4.6, C('#ff8a2a')))
    p.ellipse(9, 9.5, 2, 2, C('#ff4f3a'))
    part(p, 14, 14, (q) => q.poly([7, 3.5, 11, 1, 12, 3, 8, 4.5], C('#4cc35a')))
    out.push(p)
  }
  // pineapple
  {
    const p = new Pix(14, 16)
    part(p, 14, 16, (q) => {
      q.poly([7, 0.5, 8.5, 4, 11, 1.5, 9.5, 5.5, 4.5, 5.5, 3, 1.5, 5.5, 4], C('#3fae4a'))
    })
    part(p, 14, 16, (q) => q.ellipse(7, 10.5, 4.6, 5, C('#ffc531')))
    for (let y = 7; y < 15; y += 2) for (let x = 4 + (y % 4 ? 1 : 0); x < 11; x += 2) p.set(x, y, C('#c98a1f'))
    out.push(p)
  }
  // watermelon slice
  {
    const p = new Pix(16, 12)
    part(p, 16, 12, (q) => {
      q.poly([1, 2, 15, 2, 13, 7, 8, 10, 3, 7], C('#3fae4a'))
    })
    p.poly([2.5, 2, 13.5, 2, 12, 6, 8, 8.6, 4, 6], C('#ff4f6a'))
    p.rect(2, 2, 12, 1, lightOf(C('#ff4f6a'), 0.4))
    p.set(6, 4, EYE_DARK)
    p.set(9, 5, EYE_DARK)
    p.set(11, 3, EYE_DARK)
    out.push(p)
  }
  // grapes
  {
    const p = new Pix(14, 14)
    part(p, 14, 14, (q) => {
      for (const [x, y] of [
        [5, 5],
        [9, 5],
        [7, 7.5],
        [4, 8],
        [10, 8],
        [7, 10.5]
      ])
        q.circle(x, y, 2.2, C('#8b4fd8'))
    })
    p.line(7, 0.5, 7, 3, C('#4cc35a'))
    out.push(p)
  }
  // golden coconut
  {
    const p = new Pix(14, 14)
    part(p, 14, 14, (q) => q.circle(7, 7.5, 5.4, C('#ffcf2e')))
    p.set(5, 6, BROWN)
    p.set(8, 6, BROWN)
    p.set(6.5, 8.5, BROWN)
    p.set(4, 4, WHITE)
    out.push(p)
  }
  return out
}

function flagFrames(): Pix[] {
  return [0, 1, 2].map((f) => {
    const p = new Pix(22, 44)
    part(p, 22, 44, (q) => q.rect(3, 4, 2.4, 40, C('#a8743e')))
    part(p, 22, 44, (q) => q.circle(4.2, 3, 2.2, C('#ffd23a')))
    if (f === 0) {
      part(p, 22, 44, (q) => q.poly([5, 30, 13, 33, 5, 37], C('#38b6ff')))
    } else {
      const w = f === 1 ? 0 : 1.5
      part(p, 22, 44, (q) => q.poly([5, 6, 12, 5 + w, 20, 8, 13, 11 - w, 20, 15, 12, 14 + w, 5, 16], C('#38b6ff')))
      p.circle(11, 10, 2, C('#ffffff'))
    }
    return p
  })
}

function springFrames(): Pix[] {
  return [0, 1].map((f) => {
    const p = new Pix(20, 16)
    const red = C('#ff4f6a')
    part(p, 20, 16, (q) => q.rect(8, f ? 10 : 8, 4, f ? 6 : 8, C('#4cc35a')))
    part(p, 20, 16, (q) => {
      q.ellipse(10, f ? 10 : 6, 9, f ? 3 : 4.6, red)
      q.rect(1, f ? 10 : 6, 18, f ? 2 : 5, 0)
    })
    p.circle(6, f ? 9 : 4, 1.2, WHITE)
    p.circle(13, f ? 9 : 3.5, 1.2, WHITE)
    return p
  })
}

function shotFrames(): Pix[] {
  // 0 bubble, 1 seed, 2 fireball, 3 rock chunk
  const b = new Pix(12, 12)
  b.circle(6, 6, 5, rgb('#9fe6ff', 255))
  b.circle(6, 6, 3.6, 0)
  b.set(4, 3, WHITE)
  b.set(3, 4, WHITE)
  b.outline(C('#3a8fc0'))
  const s = new Pix(10, 10)
  part(s, 10, 10, (q) => q.ellipse(5, 5, 3.4, 2.6, C('#7bd43c')))
  const fb = new Pix(12, 12)
  part(fb, 12, 12, (q) => q.circle(6, 6, 4.4, C('#ff7a1f')), { shade: false })
  fb.circle(5.5, 5.5, 2.4, C('#ffe14d'))
  const r = new Pix(10, 10)
  part(r, 10, 10, (q) => q.poly([2, 3, 6, 1, 9, 5, 6, 9, 1, 7], C('#7c6a7e')))
  return [b, s, fb, r]
}

function platFrames(): Pix[] {
  // one 16px segment per theme: 0 pier plank, 1 log, 2 stone slab, 3 cracked stone (crumble)
  const out: Pix[] = []
  {
    const p = new Pix(16, 10)
    part(p, 16, 10, (q) => q.rect(0, 0, 16, 5, C('#c48a4a')))
    p.line(0, 2, 15, 2, shadowOf(C('#c48a4a'), 0.25))
    p.rect(2, 5, 2, 5, C('#8a5a33'))
    p.rect(12, 5, 2, 5, C('#8a5a33'))
    out.push(p)
  }
  {
    const p = new Pix(16, 9)
    part(p, 16, 9, (q) => q.rect(0, 1, 16, 6, C('#9a6438')))
    p.line(0, 3, 15, 3, shadowOf(C('#9a6438'), 0.3))
    p.set(5, 5, C('#4cc35a'))
    p.set(6, 5, C('#4cc35a'))
    p.set(12, 2, C('#4cc35a'))
    out.push(p)
  }
  {
    const p = new Pix(16, 10)
    part(p, 16, 10, (q) => q.rect(0, 0, 16, 8, C('#5d4a66')))
    p.line(0, 2, 15, 2, C('#ff8a2a'))
    out.push(p)
  }
  {
    const p = new Pix(16, 10)
    part(p, 16, 10, (q) => q.rect(0, 0, 16, 8, C('#6e5a72')))
    p.line(4, 0, 6, 4, OUTLINE)
    p.line(6, 4, 5, 8, OUTLINE)
    p.line(11, 0, 10, 3, OUTLINE)
    out.push(p)
  }
  return out
}

// ───────────────────────────── bosses ─────────────────────────────

/** 钳钳大王: a hermit-crab king wearing a crown of barnacles, one giant claw. Frames: idle0, idle1, raise, blow, hurt */
function crabKingFrames(): Pix[] {
  const red = C('#e8443a')
  const W = 72
  const H = 56
  return [0, 1, 2, 3, 4].map((f) => {
    const p = new Pix(W, H)
    const up = f === 2
    const blow = f === 3
    const hurt = f === 4
    const leg = f === 1 ? 1.5 : 0
    // legs
    part(p, W, H, (q) => {
      for (let i = 0; i < 3; i++) {
        q.line(26 + i * 8, 40, 20 + i * 8 - leg, 54, shadowOf(red, 0.2), 3)
        q.line(34 + i * 8, 40, 42 + i * 8 + leg, 54, shadowOf(red, 0.2), 3)
      }
    })
    // big spiral shell on its back
    part(p, W, H, (q) => {
      q.ellipse(26, 28, 20, 18, CREAM)
    })
    spiral(p, 26, 28, 15, 1.8, BROWN, 2)
    for (const [x, y] of [
      [12, 22],
      [16, 38],
      [36, 36],
      [30, 16]
    ])
      p.circle(x, y, 2, C('#d99a6a'))
    // body
    part(p, W, H, (q) => {
      q.ellipse(44, 38, 15, 9, red)
    })
    p.ellipse(44, 42, 11, 3, lightOf(red, 0.35))
    // barnacle crown
    part(p, W, H, (q) => {
      q.poly([33, 16, 35, 8, 38, 13, 41, 6, 44, 13, 47, 8, 49, 16], C('#ffd23a'))
    })
    p.circle(41, 10, 1.4, C('#38b6ff'))
    // eyestalks + eyes
    part(p, W, H, (q) => {
      q.line(42, 32, 40, 22, red, 2.4)
      q.line(50, 32, 53, 22, red, 2.4)
    })
    part(p, W, H, (q) => {
      q.circle(40, 20, 4, WHITE)
      q.circle(53.5, 20, 4, WHITE)
    })
    if (hurt) {
      for (const ex of [40, 53.5]) {
        p.line(ex - 2, 18, ex + 2, 22, EYE_DARK)
        p.line(ex + 2, 18, ex - 2, 22, EYE_DARK)
      }
    } else {
      p.rect(41, 19, 2, 3, EYE_DARK)
      p.rect(54, 19, 2, 3, EYE_DARK)
      p.line(36, 14, 43, 17, EYE_DARK, 1.6)
      p.line(57.5, 14, 51, 17, EYE_DARK, 1.6)
    }
    // mouth
    if (blow) {
      p.circle(52, 37, 2.6, C('#5a1830'))
    } else p.line(48, 37, 55, 37, C('#5a1830'))
    // small claw
    part(p, W, H, (q) => {
      q.circle(60, 44, 4, red)
    })
    // giant claw (raised when attacking)
    part(p, W, H, (q) => {
      const cx = up ? 62 : 63
      const cy = up ? 18 : 30
      q.line(54, 36, cx - 4, cy + 6, red, 5)
      q.ellipse(cx, cy, 9, 7.5, red)
      if (up) q.poly([cx + 3, cy - 3, cx + 10, cy - 8, cx + 9, cy + 1], red)
      q.rect(cx + 2, cy - 1, 8, 2.5, 0)
    })
    p.ellipse(up ? 60 : 61, up ? 16 : 28, 3, 2, lightOf(red, 0.4))
    return p
  })
}

/** 千藤花妖: a giant carnivorous bloom on a vine stem. Frames: idle0, idle1, open (spit), hurt */
function bloomFrames(): Pix[] {
  const W = 76
  const H = 96
  const stem = C('#2f9e4a')
  const petal = C('#ff4f8a')
  return [0, 1, 2, 3].map((f) => {
    const p = new Pix(W, H)
    const sway = f === 1 ? 2 : 0
    const open = f === 2
    const hurt = f === 3
    // stem + leaves
    part(p, W, H, (q) => {
      q.line(40, 95, 44, 70, stem, 7)
      q.line(44, 70, 38 + sway, 46, stem, 6)
    })
    part(p, W, H, (q) => {
      q.poly([42, 80, 20, 70, 14, 78, 30, 84], C('#43b84a'))
      q.poly([44, 72, 66, 62, 70, 70, 54, 76], C('#43b84a'))
    })
    const hx = 37 + sway
    const hy = 30
    // petals
    part(p, W, H, (q) => {
      for (let i = 0; i < 7; i++) {
        const a = (i / 7) * Math.PI * 2 + 0.3
        q.ellipse(hx + Math.cos(a) * 17, hy + Math.sin(a) * 15, 9, 8, i % 2 ? petal : lightOf(petal, 0.15))
      }
    })
    for (let i = 0; i < 7; i++) {
      const a = (i / 7) * Math.PI * 2 + 0.3
      p.circle(hx + Math.cos(a) * 19, hy + Math.sin(a) * 17, 1.4, C('#fff2a8'))
    }
    // face
    part(p, W, H, (q) => q.circle(hx, hy, 13, C('#ffd23a')))
    if (hurt) {
      p.line(hx - 8, hy - 6, hx - 3, hy - 2, EYE_DARK, 1.5)
      p.line(hx - 3, hy - 6, hx - 8, hy - 2, EYE_DARK, 1.5)
      p.line(hx + 3, hy - 6, hx + 8, hy - 2, EYE_DARK, 1.5)
      p.line(hx + 8, hy - 6, hx + 3, hy - 2, EYE_DARK, 1.5)
    } else {
      p.rect(hx - 7, hy - 6, 3, 4, EYE_DARK)
      p.rect(hx + 4, hy - 6, 3, 4, EYE_DARK)
      p.set(hx - 6, hy - 6, WHITE)
      p.set(hx + 5, hy - 6, WHITE)
      p.line(hx - 9, hy - 9, hx - 3, hy - 7, EYE_DARK, 1.4)
      p.line(hx + 9, hy - 9, hx + 3, hy - 7, EYE_DARK, 1.4)
    }
    // mouth with fangs
    const mh = open ? 7 : 3
    p.ellipse(hx, hy + 5, 7, mh / 1.4 + 1, C('#6a1030'))
    for (let i = -2; i <= 2; i++) p.poly([hx + i * 2.6 - 1, hy + 5 - mh / 2, hx + i * 2.6 + 1, hy + 5 - mh / 2, hx + i * 2.6, hy + 7 - mh / 2], WHITE)
    return p
  })
}

/** 熔岩龟王: a basalt-shelled tortoise with glowing cracks. Frames: walk0, walk1, open, shell0, shell1, hurt */
function tortoiseFrames(): Pix[] {
  const W = 80
  const H = 52
  const shell = C('#4a3a52')
  const glow = C('#ff8a1f')
  const skin = C('#9b7a5a')
  const make = (f: number) => {
    const p = new Pix(W, H)
    const inShell = f === 3 || f === 4
    const step = f === 1 ? 2 : 0
    const open = f === 2
    const hurt = f === 5
    if (!inShell) {
      part(p, W, H, (q) => {
        q.rect(18 + step, 36, 9, 14, skin)
        q.rect(50 - step, 36, 9, 14, skin)
      })
      // head
      part(p, W, H, (q) => {
        q.line(58, 30, 66, 26, skin, 9)
        q.ellipse(70, 26, 9, 7, skin)
        q.poly([68, 18, 71, 9, 74, 19], C('#e8d7a8'))
      })
      if (hurt) {
        p.line(70, 21, 74, 25, EYE_DARK, 1.4)
        p.line(74, 21, 70, 25, EYE_DARK, 1.4)
      } else {
        p.rect(72, 22, 3, 3, C('#ffe14d'))
        p.set(74, 23, EYE_DARK)
        p.line(68, 20, 75, 21, EYE_DARK, 1.4)
      }
      if (open) {
        p.ellipse(75, 30, 4, 3, C('#6a1030'))
        p.circle(76, 30, 1.5, glow)
      } else p.line(71, 30, 78, 29, C('#3a1a20'))
    }
    // shell
    const rot = f === 4 ? 0.4 : 0
    part(p, W, H, (q) => {
      q.ellipse(38, inShell ? 32 : 28, 28, inShell ? 18 : 19, shell)
      q.rect(8, inShell ? 34 : 32, 60, 10, 0)
      q.rect(10, inShell ? 32 : 30, 56, 6, shadowOf(shell, 0.2))
    })
    // lava cracks
    const cr = (x0: number, y0: number, x1: number, y1: number) => {
      const dx = Math.cos(rot)
      p.line(38 + (x0 - 38) * dx, y0 + (inShell ? 4 : 0), 38 + (x1 - 38) * dx, y1 + (inShell ? 4 : 0), glow, 1.4)
    }
    cr(24, 16, 30, 24)
    cr(30, 24, 26, 30)
    cr(42, 12, 44, 22)
    cr(44, 22, 52, 26)
    cr(54, 18, 52, 26)
    cr(36, 26, 38, 30)
    p.set(30, inShell ? 18 : 14, lightOf(shell, 0.6))
    p.set(31, inShell ? 17 : 13, lightOf(shell, 0.6))
    return p
  }
  return [0, 1, 2, 3, 4, 5].map(make)
}

export interface Art {
  hero: HeroSheet[]
  heads: HTMLCanvasElement[]
  crab: Sheet
  gull: Sheet
  fish: Sheet
  frog: Sheet
  spider: Sheet
  spiky: Sheet
  bat: Sheet
  flame: Sheet
  boulder: Sheet
  fire: Sheet
  rock: Sheet
  clam: Sheet
  boom: Sheet
  gold: Sheet
  board: Sheet
  fairy: Sheet
  cloud: Sheet
  basket: Sheet
  sun: Sheet
  fruit: Sheet[]
  flag: Sheet
  spring: Sheet
  shot: Sheet[]
  plat: Sheet[]
  crabKing: Sheet
  bloom: Sheet
  tortoise: Sheet
}

export function buildArt(): Art {
  const fruit = fruitFrames()
  const shots = shotFrames()
  const plats = platFrames()
  return {
    hero: HERO_LOOKS.map(heroSheet),
    heads: HERO_LOOKS.map((l) => heroHead(l).canvas()),
    crab: sheet(crabFrames()),
    gull: sheet(gullFrames(), 11, 8),
    fish: sheet(fishFrames(), 9, 7),
    frog: sheet(frogFrames()),
    spider: sheet(spiderFrames(), 9, 7),
    spiky: sheet(spikyFrames(), 9, 9),
    bat: sheet(batFrames(), 11, 7),
    flame: sheet(flameFrames()),
    boulder: sheet(boulderFrames(), 12, 12),
    fire: sheet(campfireFrames()),
    rock: sheet([rockFrame()]),
    clam: sheet(clamFrames()),
    boom: sheet(boomFrames(false), 7, 7),
    gold: sheet(boomFrames(true), 7, 7),
    board: sheet([boardFrame()]),
    fairy: sheet(fairyFrames(), 8, 8),
    cloud: sheet(cloudFrames(), 11, 7),
    basket: sheet([basketFrame()]),
    sun: sheet(sunFruitFrames(), 12, 12),
    fruit: fruit.map((f) => sheet([f], f.w / 2, f.h / 2)),
    flag: sheet(flagFrames(), 4, 44),
    spring: sheet(springFrames(), 10, 16),
    shot: shots.map((s) => sheet([s], s.w / 2, s.h / 2)),
    plat: plats.map((s) => sheet([s], 0, 0)),
    crabKing: sheet(crabKingFrames(), 40, 56),
    bloom: sheet(bloomFrames(), 40, 96),
    tortoise: sheet(tortoiseFrames(), 40, 52)
  }
}

export { mix }
