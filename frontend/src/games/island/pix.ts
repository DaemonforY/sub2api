// Tiny pixel raster: every sprite in 椰风岛 is drawn into a Pix buffer with hard edges (no anti-aliasing),
// then outlined and shaded, so all art shares one crisp style at native resolution.

export type Color = number // 0xAABBGGRR (ImageData byte order on little-endian machines)

export function rgb(hex: string, a = 255): Color {
  const n = parseInt(hex.replace('#', ''), 16)
  const r = (n >> 16) & 255
  const g = (n >> 8) & 255
  const b = n & 255
  return ((a << 24) | (b << 16) | (g << 8) | r) >>> 0
}

export function mix(c: Color, d: Color, t: number): Color {
  const r = (c & 255) * (1 - t) + (d & 255) * t
  const g = ((c >> 8) & 255) * (1 - t) + ((d >> 8) & 255) * t
  const b = ((c >> 16) & 255) * (1 - t) + ((d >> 16) & 255) * t
  return ((255 << 24) | (Math.round(b) << 16) | (Math.round(g) << 8) | Math.round(r)) >>> 0
}

const SHADOW = rgb('#2b1d52')
const LIGHT = rgb('#fff6d0')
export const shadowOf = (c: Color, k = 0.32): Color => mix(c, SHADOW, k)
export const lightOf = (c: Color, k = 0.32): Color => mix(c, LIGHT, k)
export const OUTLINE = rgb('#1b1028')

export class Pix {
  readonly w: number
  readonly h: number
  readonly d: Uint32Array

  constructor(w: number, h: number) {
    this.w = w
    this.h = h
    this.d = new Uint32Array(w * h)
  }

  get(x: number, y: number): Color {
    if (x < 0 || y < 0 || x >= this.w || y >= this.h) return 0
    return this.d[y * this.w + x]
  }

  set(x: number, y: number, c: Color): void {
    x |= 0
    y |= 0
    if (x < 0 || y < 0 || x >= this.w || y >= this.h) return
    this.d[y * this.w + x] = c
  }

  rect(x: number, y: number, w: number, h: number, c: Color): this {
    for (let j = Math.round(y); j < Math.round(y + h); j++) for (let i = Math.round(x); i < Math.round(x + w); i++) this.set(i, j, c)
    return this
  }

  /** Filled ellipse: a pixel is in when its centre is. */
  ellipse(cx: number, cy: number, rx: number, ry: number, c: Color): this {
    const x0 = Math.floor(cx - rx - 1)
    const x1 = Math.ceil(cx + rx + 1)
    const y0 = Math.floor(cy - ry - 1)
    const y1 = Math.ceil(cy + ry + 1)
    for (let y = y0; y <= y1; y++) {
      for (let x = x0; x <= x1; x++) {
        const dx = (x + 0.5 - cx) / rx
        const dy = (y + 0.5 - cy) / ry
        if (dx * dx + dy * dy <= 1) this.set(x, y, c)
      }
    }
    return this
  }

  circle(cx: number, cy: number, r: number, c: Color): this {
    return this.ellipse(cx, cy, r, r, c)
  }

  /** Thick line: discs of radius `t/2` stamped along the segment. */
  line(x0: number, y0: number, x1: number, y1: number, c: Color, t = 1): this {
    const n = Math.max(1, Math.ceil(Math.hypot(x1 - x0, y1 - y0) * 2))
    for (let i = 0; i <= n; i++) {
      const x = x0 + ((x1 - x0) * i) / n
      const y = y0 + ((y1 - y0) * i) / n
      if (t <= 1) this.set(Math.floor(x), Math.floor(y), c)
      else this.ellipse(x, y, t / 2, t / 2, c)
    }
    return this
  }

  /** Filled polygon (even-odd), sampled at pixel centres. pts = [x0,y0,x1,y1,...] */
  poly(pts: number[], c: Color): this {
    let minY = Infinity
    let maxY = -Infinity
    for (let i = 1; i < pts.length; i += 2) {
      minY = Math.min(minY, pts[i])
      maxY = Math.max(maxY, pts[i])
    }
    const n = pts.length / 2
    for (let y = Math.floor(minY); y <= Math.ceil(maxY); y++) {
      const sy = y + 0.5
      const xs: number[] = []
      for (let i = 0; i < n; i++) {
        const ax = pts[i * 2]
        const ay = pts[i * 2 + 1]
        const bx = pts[((i + 1) % n) * 2]
        const by = pts[((i + 1) % n) * 2 + 1]
        if ((ay <= sy && by > sy) || (by <= sy && ay > sy)) xs.push(ax + ((sy - ay) / (by - ay)) * (bx - ax))
      }
      xs.sort((a, b) => a - b)
      for (let k = 0; k + 1 < xs.length; k += 2) {
        for (let x = Math.ceil(xs[k] - 0.5); x <= Math.floor(xs[k + 1] - 0.5); x++) this.set(x, y, c)
      }
    }
    return this
  }

  /** Paint an ASCII map; '.' and ' ' are transparent. */
  stamp(rows: string[], x: number, y: number, pal: Record<string, Color>, flip = false): this {
    for (let j = 0; j < rows.length; j++) {
      const row = rows[j]
      for (let i = 0; i < row.length; i++) {
        const ch = row[i]
        if (ch === '.' || ch === ' ') continue
        const c = pal[ch]
        if (c === undefined) continue
        this.set(flip ? x + row.length - 1 - i : x + i, y + j, c)
      }
    }
    return this
  }

  /** Light from the top-left: rim-light top/left edges, shade bottom/right edges. */
  shade(depth = 1, lightK = 0.28, shadowK = 0.34): this {
    const src = this.d.slice()
    const at = (x: number, y: number) => (x < 0 || y < 0 || x >= this.w || y >= this.h ? 0 : src[y * this.w + x])
    for (let y = 0; y < this.h; y++) {
      for (let x = 0; x < this.w; x++) {
        const c = src[y * this.w + x]
        if (!c) continue
        let dark = false
        for (let k = 1; k <= depth; k++) if (!at(x + k, y + k) || !at(x, y + k)) dark = true
        if (dark) this.d[y * this.w + x] = shadowOf(c, shadowK)
        else if (!at(x - 1, y - 1) || !at(x, y - 1)) this.d[y * this.w + x] = lightOf(c, lightK)
      }
    }
    return this
  }

  /** 1px outline around opaque pixels. */
  outline(c: Color = OUTLINE, diagonal = false): this {
    const src = this.d.slice()
    const at = (x: number, y: number) => (x < 0 || y < 0 || x >= this.w || y >= this.h ? 0 : src[y * this.w + x])
    for (let y = 0; y < this.h; y++) {
      for (let x = 0; x < this.w; x++) {
        if (src[y * this.w + x]) continue
        if (at(x - 1, y) || at(x + 1, y) || at(x, y - 1) || at(x, y + 1) || (diagonal && (at(x - 1, y - 1) || at(x + 1, y - 1) || at(x - 1, y + 1) || at(x + 1, y + 1)))) {
          this.d[y * this.w + x] = c
        }
      }
    }
    return this
  }

  /** Copy opaque pixels of `o` onto this buffer. */
  draw(o: Pix, dx = 0, dy = 0): this {
    for (let y = 0; y < o.h; y++) {
      for (let x = 0; x < o.w; x++) {
        const c = o.d[y * o.w + x]
        if (c) this.set(x + dx, y + dy, c)
      }
    }
    return this
  }

  /** Recolour every opaque pixel (silhouettes, hit flashes). */
  tint(c: Color): Pix {
    const p = new Pix(this.w, this.h)
    for (let i = 0; i < this.d.length; i++) if (this.d[i]) p.d[i] = c
    return p
  }

  flipX(): Pix {
    const p = new Pix(this.w, this.h)
    for (let y = 0; y < this.h; y++) for (let x = 0; x < this.w; x++) p.d[y * this.w + (this.w - 1 - x)] = this.d[y * this.w + x]
    return p
  }

  /** Rotate by a quarter turn steps around the centre (for rolling things), nearest-neighbour. */
  rotate(angle: number): Pix {
    const p = new Pix(this.w, this.h)
    const cx = this.w / 2
    const cy = this.h / 2
    const cs = Math.cos(-angle)
    const sn = Math.sin(-angle)
    for (let y = 0; y < this.h; y++) {
      for (let x = 0; x < this.w; x++) {
        const dx = x + 0.5 - cx
        const dy = y + 0.5 - cy
        const sx = Math.floor(cx + dx * cs - dy * sn)
        const sy = Math.floor(cy + dx * sn + dy * cs)
        p.d[y * this.w + x] = this.get(sx, sy)
      }
    }
    return p
  }

  canvas(): HTMLCanvasElement {
    const cv = document.createElement('canvas')
    cv.width = this.w
    cv.height = this.h
    const ctx = cv.getContext('2d')
    if (ctx) {
      const img = ctx.createImageData(this.w, this.h)
      new Uint32Array(img.data.buffer).set(this.d)
      ctx.putImageData(img, 0, 0)
    }
    return cv
  }
}

/** Deterministic small PRNG (mulberry32) for textures and level decoration. */
export function rng(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}
