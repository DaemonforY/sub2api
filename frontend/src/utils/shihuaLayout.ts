/**
 * 课本诗画 poster text layer. The image model draws the picture with the title and the author's name;
 * this sets the body text (vertical, right to left), a seal and the AI label, so every character is
 * exact. Ported from promo/textbook-poems/tools/compose.py.
 *
 * Pure layout (columns, fitting, the calm area finder) is separated from the canvas drawing so it can
 * be unit tested.
 */

export type ShihuaAspect = '2x3' | '3x4' | '9x16' | '1x1' | '16x9' | '235x100'

export const SHIHUA_ASPECTS: { id: ShihuaAspect; ratio: number; landscape: boolean; extend?: boolean }[] = [
  { id: '2x3', ratio: 2 / 3, landscape: false },
  { id: '3x4', ratio: 3 / 4, landscape: false },
  // Narrower than the 2:3 picture: the canvas is extended upward (sky fading into paper) rather than
  // cropping the picture's sides.
  { id: '9x16', ratio: 9 / 16, landscape: false, extend: true },
  { id: '1x1', ratio: 1, landscape: false },
  { id: '16x9', ratio: 16 / 9, landscape: true },
  { id: '235x100', ratio: 2.35, landscape: true }
]

/** A rectangle in fractions of the picture (0–1). */
export interface ShihuaBox {
  x0: number
  y0: number
  x1: number
  y1: number
}

export interface ShihuaLayoutOptions {
  /** Where the text goes; omitted = the largest calm area inside `hint`. */
  box?: ShihuaBox
  hint?: ShihuaBox
  /** Multiplies the fitted character size (0.6–1.2). */
  scale?: number
  color?: 'auto' | 'ink' | 'light'
  seal?: string
  /** Explicit AI label (GB 45438: text height ≥ 5% of the short side). Empty = none. */
  label?: string
  labelScale?: number
  /** Body text, title (landscape) and seal font. */
  font?: ShihuaFontId
}

/** What was drawn, saved with the render so it can be reproduced. */
export interface ShihuaLayoutResult {
  aspect: ShihuaAspect
  width: number
  height: number
  box: ShihuaBox
  cell: number
  columns: string[]
  color: 'ink' | 'light'
  text: string
  font: ShihuaFontId
}

/**
 * Body fonts (all SIL OFL). 马善政 and 志莽行书 are served from /shihua-fonts/, 霞鹜文楷 from the video
 * app; 霞鹜文楷 is always loaded too, as the fallback for rare characters the brush fonts lack.
 */
export type ShihuaFontId = 'mashanzheng' | 'zhimangxing' | 'wenkai'
export const SHIHUA_FONTS: { id: ShihuaFontId; family: string }[] = [
  { id: 'mashanzheng', family: 'Ma Shan Zheng' },
  { id: 'zhimangxing', family: 'Zhi Mang Xing' },
  { id: 'wenkai', family: 'LXGW WenKai' }
]
export const SHIHUA_DEFAULT_FONT: ShihuaFontId = 'mashanzheng'
const FALLBACK_STACK = "'LXGW WenKai', 'Kaiti SC', 'STKaiti', 'KaiTi', serif"
/** The AI label stays in a plain sans-serif whatever the body font, so it reads as a notice. */
export const SHIHUA_LABEL_FONT = "'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans SC', sans-serif"

function fontOf(id?: ShihuaFontId) {
  return SHIHUA_FONTS.find((f) => f.id === id) ?? SHIHUA_FONTS.find((f) => f.id === SHIHUA_DEFAULT_FONT)!
}

/** The CSS font stack for a body font, falling back to 霞鹜文楷 and the system 楷体. */
export function shihuaFontStack(id?: ShihuaFontId): string {
  const f = fontOf(id)
  return f.id === 'wenkai' ? FALLBACK_STACK : `'${f.family}', ${FALLBACK_STACK}`
}
export const SHIHUA_DEFAULT_HINT: ShihuaBox = { x0: 0.45, y0: 0, x1: 1, y1: 0.6 }
export const SHIHUA_PITCH = 1.4
const PUNCT = '，。、；：！？'
const SENTENCE_END = '。；！？'
// Horizontal punctuation that has a vertical presentation form.
const VERTICAL_FORMS: Record<string, string> = {
  '「': '﹁', '」': '﹂', '『': '﹃', '』': '﹄', '“': '﹁', '”': '﹂', '‘': '﹃', '’': '﹄',
  '《': '︽', '》': '︾', '（': '︵', '）': '︶', '(': '︵', ')': '︶', '…': '︙', '—': '︱', '〈': '︿', '〉': '﹀'
}

/** Splits one paragraph into columns of at most maxCells, breaking after sentence ends, then commas. */
export function splitColumns(text: string, maxCells: number): string[] {
  const chars = Array.from(text.replace(/\s+/g, ''))
  if (maxCells < 2) return []
  const parts: string[][] = []
  let cur: string[] = []
  for (const ch of chars) {
    cur.push(ch)
    if (PUNCT.includes(ch)) {
      parts.push(cur)
      cur = []
    }
  }
  if (cur.length) parts.push(cur)
  const cols: string[] = []
  let col: string[] = []
  for (let part of parts) {
    if (col.length + part.length <= maxCells) {
      col = col.concat(part)
      const last = part[part.length - 1]
      if (SENTENCE_END.includes(last) && col.length >= maxCells * 0.5) {
        cols.push(col.join(''))
        col = []
      }
      continue
    }
    if (col.length) cols.push(col.join(''))
    col = []
    while (part.length > maxCells) {
      // Keep a punctuation mark with the character before it.
      let cut = maxCells
      if (PUNCT.includes(part[cut])) cut -= 1
      cols.push(part.slice(0, cut).join(''))
      part = part.slice(cut)
    }
    col = part
  }
  if (col.length) cols.push(col.join(''))
  return cols
}

/** Columns for the whole text: a poem whose lines all fit keeps one line per column. */
export function layoutColumns(text: string, rows: number): string[] {
  const lines = text.split('\n').map((l) => l.replace(/\s+/g, '')).filter(Boolean)
  if (lines.length && lines.every((l) => Array.from(l).length <= rows)) return lines
  return lines.flatMap((l) => splitColumns(l, rows))
}

/** The largest character cell (px) whose columns fit a w×h box, capped at maxCell. */
export function fitText(text: string, w: number, h: number, maxCell: number, pitch = SHIHUA_PITCH): { cell: number; columns: string[] } {
  for (let cell = Math.floor(Math.min(maxCell, h / 2)); cell >= 12; cell--) {
    const rows = Math.floor(h / cell)
    const columns = layoutColumns(text, rows)
    if (columns.length && columns.length * cell * pitch <= w) return { cell, columns }
  }
  return { cell: 12, columns: layoutColumns(text, Math.max(2, Math.floor(h / 12))) }
}

/**
 * The largest rectangle inside `hint` whose pixels are calm (low edge energy): the blank sky or paper
 * the prompt asked the model to leave. `gray` is a luminance grid of w×h (a downscaled picture).
 */
export function calmBox(gray: Float32Array, w: number, h: number, hint: ShihuaBox, threshold = 40, minFill = 0.97): ShihuaBox | null {
  // Edge energy, blurred a little (3×3 box twice).
  const e = new Float32Array(w * h)
  for (let y = 1; y < h - 1; y++) {
    for (let x = 1; x < w - 1; x++) {
      const gx = (gray[y * w + x + 1] - gray[y * w + x - 1]) / 2
      const gy = (gray[(y + 1) * w + x] - gray[(y - 1) * w + x]) / 2
      e[y * w + x] = Math.min(255, Math.hypot(gx, gy) * 8)
    }
  }
  const blurred = boxBlur(boxBlur(e, w, h), w, h)
  // Summed-area table of calm cells.
  const ii = new Int32Array((w + 1) * (h + 1))
  for (let y = 0; y < h; y++) {
    let row = 0
    for (let x = 0; x < w; x++) {
      row += blurred[y * w + x] < threshold ? 1 : 0
      ii[(y + 1) * (w + 1) + x + 1] = ii[y * (w + 1) + x + 1] + row
    }
  }
  const sum = (x0: number, y0: number, x1: number, y1: number) =>
    ii[y1 * (w + 1) + x1] - ii[y0 * (w + 1) + x1] - ii[y1 * (w + 1) + x0] + ii[y0 * (w + 1) + x0]
  const hx0 = Math.floor(hint.x0 * w), hy0 = Math.floor(hint.y0 * h), hx1 = Math.floor(hint.x1 * w), hy1 = Math.floor(hint.y1 * h)
  const step = Math.max(1, Math.round(Math.min(w, h) / 64))
  let best: [number, number, number, number, number] | null = null
  for (let y0 = hy0; y0 < hy1; y0 += step) {
    for (let y1 = y0 + Math.floor(h / 8); y1 <= hy1; y1 += step) {
      for (let x0 = hx0; x0 < hx1; x0 += step) {
        for (let x1 = x0 + Math.floor(w / 8); x1 <= hx1; x1 += step) {
          const area = (y1 - y0) * (x1 - x0)
          if (best && area <= best[0]) continue
          if (sum(x0, y0, x1, y1) >= area * minFill) best = [area, x0, y0, x1, y1]
        }
      }
    }
  }
  if (!best) return null
  const [, x0, y0, x1, y1] = best
  const m = 0.04 // inner margin
  return {
    x0: (x0 + m * (x1 - x0)) / w,
    y0: (y0 + m * (y1 - y0)) / h,
    x1: (x1 - m * (x1 - x0)) / w,
    y1: (y1 - m * (y1 - y0)) / h
  }
}

function boxBlur(src: Float32Array, w: number, h: number): Float32Array {
  const out = new Float32Array(w * h)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      let s = 0, n = 0
      for (let dy = -1; dy <= 1; dy++) {
        for (let dx = -1; dx <= 1; dx++) {
          const xx = x + dx, yy = y + dy
          if (xx >= 0 && yy >= 0 && xx < w && yy < h) {
            s += src[yy * w + xx]
            n++
          }
        }
      }
      out[y * w + x] = s / n
    }
  }
  return out
}

/** Poster size for an aspect, from a 2:3 picture `artW` wide. */
export function posterSize(aspect: ShihuaAspect, artW: number, artH: number): { width: number; height: number } {
  const a = SHIHUA_ASPECTS.find((x) => x.id === aspect)!
  if (a.landscape) return { width: Math.round(artH * a.ratio * 0.75), height: Math.round(artH * 0.75) }
  if (artW / artH > a.ratio) return a.extend ? { width: artW, height: Math.round(artW / a.ratio) } : { width: Math.round(artH * a.ratio), height: artH }
  return { width: artW, height: Math.round(artW / a.ratio) }
}

/** The text for an aspect: narrow and landscape posters take only the famous lines of a long text. */
export function textForAspect(aspect: ShihuaAspect, posterText: string, famousLines: string): string {
  const count = Array.from(posterText).filter((c) => /\p{Script=Han}/u.test(c)).length
  if ((aspect === '9x16' || aspect === '16x9' || aspect === '235x100') && count > 60 && famousLines.trim()) return famousLines.trim()
  return posterText
}

// ---------------------------------------------------------------- canvas

const cssLoads = new Map<string, Promise<boolean>>()

/** Adds a same-origin font stylesheet once (the site's CSP allows only same-origin fonts). */
function loadFontCss(url: string, rewrite: (css: string) => string = (css) => css): Promise<boolean> {
  let p = cssLoads.get(url)
  if (!p) {
    p = (async () => {
      try {
        const res = await fetch(url)
        // A missing file comes back as the SPA's index.html with status 200.
        if (!res.ok || !(res.headers.get('content-type') ?? '').includes('css')) return false
        const style = document.createElement('style')
        style.dataset.shihua = 'fonts'
        style.textContent = rewrite(await res.text())
        document.head.appendChild(style)
        return true
      } catch {
        return false
      }
    })()
    cssLoads.set(url, p)
  }
  return p
}

/**
 * Loads a body font plus 霞鹜文楷 (the fallback). The fonts are split by unicode range, so only the
 * pieces a text needs are downloaded. False when the body font can't be loaded (the system 楷体 is used).
 */
export async function loadShihuaFont(id: ShihuaFontId = SHIHUA_DEFAULT_FONT): Promise<boolean> {
  const wenkai = loadFontCss('/video/fonts/fonts.css', (css) => css.split('/fonts/wenkai/').join('/video/fonts/wenkai/'))
  if (fontOf(id).id === 'wenkai') return wenkai
  const [, own] = await Promise.all([wenkai, loadFontCss(`/shihua-fonts/${fontOf(id).id}/fonts.css`)])
  return own
}

async function ensureGlyphs(text: string, px: number, id?: ShihuaFontId) {
  await loadShihuaFont(id)
  try {
    await Promise.all([document.fonts.load(`${px}px '${fontOf(id).family}'`, text), document.fonts.load(`${px}px 'LXGW WenKai'`, text)])
  } catch {
    /* fall back to the system 楷体 */
  }
}

export function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('图片加载失败'))
    img.src = url
  })
}

function grayOf(canvas: HTMLCanvasElement, box: { x: number; y: number; w: number; h: number }, gw: number, gh: number): Float32Array {
  const small = document.createElement('canvas')
  small.width = gw
  small.height = gh
  const ctx = small.getContext('2d', { willReadFrequently: true })!
  ctx.drawImage(canvas, box.x, box.y, box.w, box.h, 0, 0, gw, gh)
  const data = ctx.getImageData(0, 0, gw, gh).data
  const gray = new Float32Array(gw * gh)
  for (let i = 0; i < gw * gh; i++) gray[i] = 0.299 * data[i * 4] + 0.587 * data[i * 4 + 1] + 0.114 * data[i * 4 + 2]
  return gray
}

function drawVerticalColumns(ctx: CanvasRenderingContext2D, columns: string[], right: number, top: number, cell: number, color: string, rule: string, font: string) {
  const pitch = cell * SHIHUA_PITCH
  ctx.textBaseline = 'top'
  ctx.textAlign = 'left'
  ctx.font = `${Math.round(cell * 0.86)}px ${font}`
  const longest = Math.max(...columns.map((c) => Array.from(c).length))
  columns.forEach((col, ci) => {
    const cx = right - ci * pitch - pitch / 2
    if (ci > 0) {
      ctx.strokeStyle = rule
      ctx.lineWidth = Math.max(1, cell / 30)
      ctx.beginPath()
      ctx.moveTo(cx + pitch / 2, top)
      ctx.lineTo(cx + pitch / 2, top + longest * cell)
      ctx.stroke()
    }
    ctx.fillStyle = color
    Array.from(col).forEach((raw, ri) => {
      const ch = VERTICAL_FORMS[raw] ?? raw
      const y = top + ri * cell
      if (PUNCT.includes(ch)) ctx.fillText(ch, cx - cell * 0.05, y - cell * 0.42)
      else ctx.fillText(ch, cx - cell * 0.43, y + cell * 0.05)
    })
  })
}

function drawSeal(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, size: number, font: string) {
  ctx.save()
  ctx.fillStyle = 'rgba(178, 34, 34, 0.92)'
  const r = size / 14
  ctx.beginPath()
  ctx.roundRect(x, y, size, size, r)
  ctx.fill()
  ctx.fillStyle = 'rgba(250, 240, 225, 1)'
  ctx.font = `${Math.round(size * 0.4)}px ${font}`
  ctx.textBaseline = 'top'
  Array.from(text).slice(0, 4).forEach((ch, i) => {
    const cx = x + size * (i < 2 ? 0.52 : 0.08)
    const cy = y + size * (0.06 + 0.46 * (i % 2))
    ctx.fillText(ch, cx, cy)
  })
  ctx.restore()
}

function drawLabel(ctx: CanvasRenderingContext2D, label: string, W: number, H: number, scale: number) {
  if (!label) return
  const px = Math.round(Math.min(W, H) * 0.05 * Math.max(1, scale))
  ctx.save()
  ctx.font = `${px}px ${SHIHUA_LABEL_FONT}`
  ctx.textBaseline = 'middle'
  const tw = ctx.measureText(label).width
  const pad = px * 0.35
  ctx.fillStyle = 'rgba(0, 0, 0, 0.35)'
  ctx.beginPath()
  ctx.roundRect(W - tw - pad * 3, H - px - pad * 3, tw + pad * 2, px + pad * 2, pad)
  ctx.fill()
  ctx.fillStyle = 'rgba(255, 255, 255, 0.92)'
  ctx.fillText(label, W - tw - pad * 2, H - px / 2 - pad * 2)
  ctx.restore()
}

/** Softens whatever detail strays into the text area: a blurred, slightly tinted copy, feathered in. */
function soften(ctx: CanvasRenderingContext2D, src: HTMLCanvasElement, box: { x: number; y: number; w: number; h: number }, cell: number, light: boolean) {
  const W = src.width, H = src.height
  const soft = document.createElement('canvas')
  soft.width = W
  soft.height = H
  const s = soft.getContext('2d')!
  s.filter = `blur(${Math.round(cell * 0.35)}px)`
  s.drawImage(src, 0, 0)
  s.filter = 'none'
  s.fillStyle = light ? 'rgba(244, 236, 218, 0.18)' : 'rgba(20, 30, 45, 0.18)'
  s.fillRect(0, 0, W, H)
  const mask = document.createElement('canvas')
  mask.width = W
  mask.height = H
  const m = mask.getContext('2d')!
  m.filter = `blur(${Math.round(cell * 0.7)}px)`
  m.fillStyle = '#000'
  m.fillRect(box.x, box.y, box.w, box.h)
  s.globalCompositeOperation = 'destination-in'
  s.drawImage(mask, 0, 0)
  ctx.drawImage(soft, 0, 0)
}

/**
 * Fills the band above the picture for the taller aspects: the picture's top rows stretched upward and
 * blurred (so the sky continues), fading into paper towards the top edge, and feathered into the picture.
 */
function extendTop(ctx: CanvasRenderingContext2D, art: HTMLImageElement | HTMLCanvasElement, W: number, band: number) {
  const feather = Math.round(Math.min(band * 0.12, art.height * 0.025))
  const c = document.createElement('canvas')
  c.width = W
  c.height = band + feather
  const e = c.getContext('2d')!
  e.fillStyle = '#efe6d2'
  e.fillRect(0, 0, W, c.height)
  const rows = Math.max(2, Math.round(art.height * 0.01))
  e.filter = `blur(${Math.round(W / 40)}px)`
  e.drawImage(art, 0, 0, art.width, rows, -W * 0.05, -band * 0.05, W * 1.1, c.height * 1.1)
  e.filter = 'none'
  const paper = e.createLinearGradient(0, 0, 0, band)
  // Mostly paper through the band, giving way to the picture only near its edge, so a dark top edge
  // (mountains, trees) doesn't smear upward.
  paper.addColorStop(0, 'rgba(239, 230, 210, 0.96)')
  paper.addColorStop(0.55, 'rgba(239, 230, 210, 0.8)')
  paper.addColorStop(0.85, 'rgba(239, 230, 210, 0.3)')
  paper.addColorStop(1, 'rgba(239, 230, 210, 0)')
  e.fillStyle = paper
  e.fillRect(0, 0, W, band)
  // destination-in clears whatever the mask doesn't cover, so it spans the whole band: opaque down to
  // the picture's top edge, then fading out over the feather.
  const mask = e.createLinearGradient(0, 0, 0, band + feather)
  mask.addColorStop(0, 'rgba(0, 0, 0, 1)')
  mask.addColorStop(band / (band + feather), 'rgba(0, 0, 0, 1)')
  mask.addColorStop(1, 'rgba(0, 0, 0, 0)')
  e.globalCompositeOperation = 'destination-in'
  e.fillStyle = mask
  e.fillRect(0, 0, W, band + feather)
  e.globalCompositeOperation = 'source-over'
  ctx.drawImage(c, 0, 0)
}

export interface ShihuaPosterInput {
  art: HTMLImageElement | HTMLCanvasElement
  aspect: ShihuaAspect
  text: string
  title: string
  author: string
}

/**
 * Draws one poster. Portrait and square aspects crop the picture (keeping the top-left, where the
 * model put the title) and set the text in the calm area; 9:16 extends the canvas upward instead of
 * cropping. Landscape ones put the picture (with the model's title) on the right of a paper ground and
 * the text on the left.
 */
export async function renderShihuaPoster(input: ShihuaPosterInput, opts: ShihuaLayoutOptions = {}): Promise<{ canvas: HTMLCanvasElement; layout: ShihuaLayoutResult }> {
  const { art, aspect } = input
  const artW = art.width, artH = art.height
  const { width: W, height: H } = posterSize(aspect, artW, artH)
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!
  ctx.imageSmoothingQuality = 'high'
  const landscape = SHIHUA_ASPECTS.find((a) => a.id === aspect)!.landscape
  const sealText = opts.seal ?? '课本诗画'
  const fontId = fontOf(opts.font).id
  const font = shihuaFontStack(fontId)
  await ensureGlyphs(input.text + sealText, 64, fontId)

  let box: { x: number; y: number; w: number; h: number }
  let light = true
  if (landscape) {
    // Paper ground, the picture's upper part on the right 56%.
    ctx.fillStyle = '#efe6d2'
    ctx.fillRect(0, 0, W, H)
    const pw = Math.round(W * 0.56)
    const scale = pw / artW
    const sh = Math.min(artH, H / scale)
    ctx.drawImage(art, 0, 0, artW, sh, W - pw, 0, pw, sh * scale)
    // The picture already carries the title and author the model drew, so only the text goes left.
    const pad = H * 0.1
    box = { x: pad, y: pad, w: W - pw - pad * 2, h: H - pad * 2.2 }
  } else {
    const band = H > artH ? H - artH : 0
    if (band > 0) {
      ctx.drawImage(art, 0, 0, W, artH, 0, band, W, artH)
      extendTop(ctx, art, W, band)
    } else {
      ctx.drawImage(art, 0, 0, W, H, 0, 0, W, H)
    }
    const auto = (() => {
      const gw = 160, gh = Math.round((160 * H) / W)
      const found = calmBox(grayOf(canvas, { x: 0, y: 0, w: W, h: H }, gw, gh), gw, gh, opts.hint ?? SHIHUA_DEFAULT_HINT)
      return found ?? { x0: 0.62, y0: 0.05, x1: 0.95, y1: 0.42 }
    })()
    const f = opts.box ?? auto
    box = { x: f.x0 * W, y: f.y0 * H, w: (f.x1 - f.x0) * W, h: (f.y1 - f.y0) * H }
    const g = grayOf(canvas, box, 32, 32)
    light = g.reduce((a, b) => a + b, 0) / g.length > 140
  }
  if (opts.color === 'ink') light = true
  if (opts.color === 'light') light = false

  const maxCell = Math.min(W, H) * (landscape ? 0.07 : 0.06)
  const fitted = fitText(input.text, box.w, box.h * 0.86, maxCell)
  const cell = Math.max(12, Math.round(fitted.cell * Math.min(1.2, Math.max(0.6, opts.scale ?? 1))))
  const columns = cell === fitted.cell ? fitted.columns : layoutColumns(input.text, Math.floor((box.h * 0.86) / cell))
  if (!landscape) soften(ctx, canvas, box, cell, light)

  const pitch = cell * SHIHUA_PITCH
  const blockW = columns.length * pitch
  const right = landscape ? box.x + blockW + (box.w - blockW) / 2 : box.x + box.w - (box.w - blockW) / 2
  const ink = light ? '#2b2622' : '#f6ecd6'
  const rule = light ? 'rgba(43, 38, 34, 0.28)' : 'rgba(246, 236, 214, 0.3)'
  drawVerticalColumns(ctx, columns, right, box.y, cell, ink, rule, font)
  if (sealText) {
    const size = Math.round(cell * 1.5)
    const lastLen = Array.from(columns[columns.length - 1] ?? '').length
    drawSeal(ctx, sealText, right - blockW - size * 1.1, box.y + lastLen * cell + cell * 0.3, size, font)
  }
  drawLabel(ctx, opts.label ?? 'AI 生成', W, H, opts.labelScale ?? 1)

  return {
    canvas,
    layout: {
      aspect,
      width: W,
      height: H,
      box: { x0: box.x / W, y0: box.y / H, x1: (box.x + box.w) / W, y1: (box.y + box.h) / H },
      cell,
      columns,
      color: light ? 'ink' : 'light',
      text: input.text,
      font: fontId
    }
  }
}

/** The picture at 2× for laying out: the upscaled file when there is one, else the original enlarged. */
export async function loadShihuaArt(url: string, upscaledUrl?: string): Promise<HTMLCanvasElement> {
  const img = await loadImage(upscaledUrl || url)
  const factor = upscaledUrl ? 1 : 2
  const c = document.createElement('canvas')
  c.width = img.naturalWidth * factor
  c.height = img.naturalHeight * factor
  const ctx = c.getContext('2d')!
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(img, 0, 0, c.width, c.height)
  return c
}

export function canvasToBlob(canvas: HTMLCanvasElement, quality = 0.92): Promise<Blob> {
  return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('导出失败'))), 'image/jpeg', quality))
}
