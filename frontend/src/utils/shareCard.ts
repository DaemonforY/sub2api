// Share cards: an image of a page for WeChat Moments / groups — either a passage the reader
// selected (quote) or the page's key points (summary) — with a QR code back to the page that
// carries the sharer's invite code. Drawn on a <canvas> in the browser; no server involved.
//
// Two identical copies: frontend/src/utils/shareCard.ts and learn/.vitepress/theme/shareCard.ts
// (frontend/src/utils/__tests__/shareCard.spec.ts fails when they differ). Keep it framework-free.
//
// Design: a saturated gradient frame with a honeycomb pattern (HiveGPT) around a paper card, so the
// image stands out on both the white and the dark Moments background, also as a cropped thumbnail.
import QRCode from 'qrcode'

export type ShareTheme = 'violet' | 'honey' | 'ocean' | 'ink'
export const SHARE_THEMES: ShareTheme[] = ['violet', 'honey', 'ocean', 'ink']
export const SHARE_THEME_NAMES: Record<ShareTheme, string> = { violet: '蜂巢紫', honey: '蜂蜜橙', ocean: '青碧', ink: '墨夜' }

export interface ShareBlock {
  kind: 'text' | 'code'
  text: string
}

interface ShareCardBase {
  theme: ShareTheme
  /** Top-left brand, e.g. 「HiveGPT AI 学习」. */
  brand: string
  /** Top-right pill: track / section / category. */
  label?: string
  title: string
  /** QR target (see shareUrl). */
  url: string
  /** Under 「扫码阅读全文」, e.g. 「直达这段原文 · 3 分钟读完」. */
  meta?: string
  /** Attribution for reprinted content, e.g. 「来源 JavaGuide · Apache-2.0」. */
  source?: string
  /** e.g. 「xxx 推荐」, only when the sharer opted in to show their name. */
  sharer?: string
  /** Bottom line on the frame; defaults to the URL's host. */
  footer?: string
  /** Text under the QR code; defaults to 「扫码阅读全文」. */
  cta?: string
}

export interface QuoteCard extends ShareCardBase {
  kind: 'quote'
  blocks: ShareBlock[]
}

export interface SummaryCard extends ShareCardBase {
  kind: 'summary'
  summary?: string
  points: string[]
}

export type ShareCard = QuoteCard | SummaryCard

export const QUOTE_MAX_CHARS = 300
export const QUOTE_MIN_CHARS = 6

// --- Text ------------------------------------------------------------------------------------------

const CLOSING = /^[，。、；：！？）」』》〉】”’,.;:!?)\]%…]$/u
const OPENING = /[（「『《〈【“‘(\[]$/u
const CJK = '\\u2e80-\\u9fff\\uf900-\\ufaff\\uff00-\\uffef\\u3000-\\u303f'

/**
 * Greedy line breaking for CJK / Latin mixed text: breaks between CJK characters and at spaces
 * between Latin words, splits words longer than a line, never starts a line with closing
 * punctuation nor ends one with opening punctuation. '\n' starts a new line.
 */
export function wrapText(text: string, maxWidth: number, measure: (s: string) => number): string[] {
  const lines: string[] = []
  for (const paragraph of text.split('\n')) {
    const tokens = paragraph.match(/[A-Za-z0-9_\-./:@#&+=?'"~$¥%]+|\s+|./gu) || []
    let line = ''
    const push = () => {
      let carry = ''
      const m = line.match(OPENING)
      if (m && line.length > 1) {
        carry = m[0]
        line = line.slice(0, -1)
      }
      lines.push(line.trimEnd())
      line = carry
    }
    for (let token of tokens) {
      if (!line && /^\s+$/.test(token)) continue
      if (!line || measure(line + token) <= maxWidth || CLOSING.test(token)) {
        line += token
      } else {
        push()
        if (/^\s+$/.test(token)) continue
        line += token
      }
      // A single token wider than the line (a long URL or identifier): split it by character.
      while (measure(line) > maxWidth && line.length > 1 && !CLOSING.test(token)) {
        let cut = line.length - 1
        while (cut > 1 && measure(line.slice(0, cut)) > maxWidth) cut--
        const rest = line.slice(cut)
        line = line.slice(0, cut)
        push()
        line += rest
        token = rest
      }
    }
    lines.push(line.trimEnd())
  }
  return lines
}

/** At most `max` lines; the last one kept ends with '…' when anything was cut. */
export function clampLines(lines: string[], max: number, maxWidth: number, measure: (s: string) => number): string[] {
  if (lines.length <= max) return lines
  const out = lines.slice(0, max)
  let last = out[max - 1].replace(/[，。、；：,.;:\s]+$/u, '')
  while (last && measure(last + '…') > maxWidth) last = last.slice(0, -1)
  out[max - 1] = last + '…'
  return out
}

/** Keeps the first `max` characters of the blocks (whitespace not counted), marking a cut with '…'. */
export function clampBlocks(blocks: ShareBlock[], max = QUOTE_MAX_CHARS): { blocks: ShareBlock[]; chars: number; truncated: boolean } {
  const out: ShareBlock[] = []
  let used = 0
  let total = 0
  for (const b of blocks) total += b.text.replace(/\s/g, '').length
  for (const b of blocks) {
    const n = b.text.replace(/\s/g, '').length
    if (used + n <= max) {
      out.push(b)
      used += n
      continue
    }
    let keep = max - used
    let i = 0
    for (; i < b.text.length && keep > 0; i++) if (!/\s/.test(b.text[i])) keep--
    const text = b.text.slice(0, i).trimEnd()
    if (text) out.push({ kind: b.kind, text: b.kind === 'code' ? `${text}\n…` : `${text}…` })
    break
  }
  return { blocks: out, chars: total, truncated: total > max }
}

/** Reading time at 400 characters a minute, at least one minute. */
export function readingMinutes(chars: number): number {
  return Math.max(1, Math.round(chars / 400))
}

// --- Selection → blocks ---------------------------------------------------------------------------

const BLOCK_TAGS = new Set(['P', 'DIV', 'LI', 'UL', 'OL', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'BLOCKQUOTE', 'TR', 'TABLE', 'SECTION', 'DETAILS', 'SUMMARY', 'DL', 'DT', 'DD', 'FIGURE'])
const SKIP_SELECTOR = 'button, svg, img, video, audio, iframe, style, script, textarea, input, select, .lang, .header-anchor, .katex-mathml, [aria-hidden="true"]'

function tidy(text: string): string {
  return text
    .replace(/\s+/g, ' ')
    .replace(new RegExp(`([${CJK}]) (?=[${CJK}])`, 'gu'), '$1')
    .trim()
}

/**
 * The selected content as paragraphs and code blocks. `skip` is a selector for areas whose
 * selection doesn't count (interactive widgets); returns [] when the selection is outside `root`
 * or touches a skipped area.
 */
export function blocksFromRange(range: Range, root: Element, skip = ''): ShareBlock[] {
  const anchor = range.commonAncestorContainer
  const anchorEl = anchor.nodeType === 1 ? (anchor as Element) : anchor.parentElement
  if (!anchorEl || !root.contains(anchorEl)) return []
  const touches = (n: Node) => {
    const el = n.nodeType === 1 ? (n as Element) : n.parentElement
    return !!(skip && el?.closest(skip))
  }
  if (touches(range.startContainer) || touches(range.endContainer) || touches(anchor)) return []
  const inCode = anchorEl.closest('pre')
  if (inCode) {
    const text = range.toString().replace(/\n+$/, '')
    return text.trim() ? [{ kind: 'code', text }] : []
  }

  const blocks: ShareBlock[] = []
  let buf = ''
  const flush = () => {
    const text = tidy(buf)
    if (text) blocks.push({ kind: 'text', text })
    buf = ''
  }
  const walk = (node: Node) => {
    if (node.nodeType === 3) {
      buf += node.textContent || ''
      return
    }
    if (node.nodeType !== 1 && node.nodeType !== 11) return
    if (node.nodeType === 1) {
      const el = node as Element
      if (el.matches(SKIP_SELECTOR) || (skip && el.matches(skip))) return
      if (el.tagName === 'PRE') {
        flush()
        const text = (el.textContent || '').replace(/\n+$/, '')
        if (text.trim()) blocks.push({ kind: 'code', text })
        return
      }
      if (el.tagName === 'BR') {
        flush()
        return
      }
      const block = BLOCK_TAGS.has(el.tagName)
      if (block) flush()
      if (el.tagName === 'LI') buf += '• '
      node.childNodes.forEach(walk)
      if (block) flush()
      else if (el.tagName === 'TD' || el.tagName === 'TH') buf += '  '
      return
    }
    node.childNodes.forEach(walk)
  }
  walk(range.cloneContents())
  flush()
  return blocks
}

/** id of the closest h2 / h3 before the selection inside root, for the QR link's #anchor. */
export function headingBefore(range: Range, root: Element): string {
  let id = ''
  root.querySelectorAll('h2[id], h3[id]').forEach((h) => {
    if (h.compareDocumentPosition(range.startContainer) & Node.DOCUMENT_POSITION_FOLLOWING) id = h.id
  })
  return id
}

// --- Link ------------------------------------------------------------------------------------------

/** The QR target: the page without its old tracking params, plus the sharer's invite code. */
export function shareUrl(pageUrl: string, opts: { aff?: string; medium: 'quote' | 'summary'; anchor?: string }): string {
  const url = new URL(pageUrl)
  for (const key of ['aff', 'aff_code', 'utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term']) url.searchParams.delete(key)
  if (opts.aff) url.searchParams.set('aff', opts.aff)
  url.searchParams.set('utm_source', 'share_card')
  url.searchParams.set('utm_medium', opts.medium)
  url.hash = opts.anchor ? encodeURIComponent(opts.anchor) : ''
  return url.toString()
}

/** A PNG data URL as a Blob, without fetch(): the site CSP's connect-src doesn't allow data:. */
export function dataUrlToBlob(dataUrl: string): Blob {
  const [head, body] = dataUrl.split(',', 2)
  const type = /data:([^;]+)/.exec(head)?.[1] || 'image/png'
  const bin = atob(body || '')
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new Blob([bytes], { type })
}

// --- Drawing ---------------------------------------------------------------------------------------

interface Palette {
  stops: string[]
  comb: string
  glow: string
  paper: string
  ink: string
  muted: string
  accent: string
  accentSoft: string
  honey: string
  code: string
  line: string
  pill: string
}

const PALETTES: Record<ShareTheme, Palette> = {
  violet: {
    stops: ['#4c1d95', '#7c3aed', '#c026d3'],
    comb: 'rgba(255,255,255,0.15)',
    glow: 'rgba(255,255,255,0.10)',
    paper: '#ffffff',
    ink: '#1e1238',
    muted: '#6b6584',
    accent: '#7c3aed',
    accentSoft: 'rgba(124,58,237,0.16)',
    honey: '#fbbf24',
    code: '#f4f1fb',
    line: '#e7e2f3',
    pill: 'rgba(255,255,255,0.18)',
  },
  honey: {
    stops: ['#f59e0b', '#f97316', '#e11d48'],
    comb: 'rgba(255,255,255,0.2)',
    glow: 'rgba(255,255,255,0.14)',
    paper: '#fffbf3',
    ink: '#3a1a07',
    muted: '#86634a',
    accent: '#ea580c',
    accentSoft: 'rgba(234,88,12,0.16)',
    honey: '#7c2d12',
    code: '#fbf0e0',
    line: '#f1e2cc',
    pill: 'rgba(255,255,255,0.22)',
  },
  ocean: {
    stops: ['#0f766e', '#0891b2', '#2563eb'],
    comb: 'rgba(255,255,255,0.15)',
    glow: 'rgba(255,255,255,0.10)',
    paper: '#ffffff',
    ink: '#0b2530',
    muted: '#557079',
    accent: '#0e7490',
    accentSoft: 'rgba(14,116,144,0.15)',
    honey: '#fcd34d',
    code: '#eef7f9',
    line: '#dceef2',
    pill: 'rgba(255,255,255,0.18)',
  },
  ink: {
    stops: ['#0b1023', '#1e1b4b', '#3b0764'],
    comb: 'rgba(251,191,36,0.20)',
    glow: 'rgba(251,191,36,0.10)',
    paper: '#171b33',
    ink: '#f5f3ff',
    muted: '#a8a3c7',
    accent: '#fbbf24',
    accentSoft: 'rgba(251,191,36,0.18)',
    honey: '#fbbf24',
    code: '#22274a',
    line: '#2d3258',
    pill: 'rgba(251,191,36,0.16)',
  },
}

const SANS = '"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans CJK SC","Noto Sans SC",sans-serif'
const SERIF = '"Songti SC","STSong","Noto Serif SC","Source Han Serif SC","SimSun",serif'
const MONO = '"SF Mono",Menlo,Consolas,"Liberation Mono","Courier New",monospace'

const W = 1080
const FRAME = 64 // gradient visible around the card
const HEAD = 196 // brand row above the card
const FOOT = 132 // host line below the card
const PAD = 72 // inside the card
const TEXT_W = W - FRAME * 2 - PAD * 2
const MIN_H = 1350
const MAX_H = 1920
const QR = 208

type Ctx = CanvasRenderingContext2D
type Painter = (ctx: Ctx, y: number) => void

interface Laid {
  height: number
  paint: Painter
}

function hexPath(ctx: Ctx, cx: number, cy: number, r: number) {
  ctx.beginPath()
  for (let i = 0; i < 6; i++) {
    const a = (Math.PI / 3) * i - Math.PI / 2
    const x = cx + r * Math.cos(a)
    const y = cy + r * Math.sin(a)
    if (i === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  }
  ctx.closePath()
}

function drawBackground(ctx: Ctx, p: Palette, h: number) {
  const bg = ctx.createLinearGradient(0, 0, W, h)
  p.stops.forEach((c, i) => bg.addColorStop(i / (p.stops.length - 1), c))
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, h)

  // Honeycomb: pointy-top hexagons, denser toward the top-right corner.
  const r = 44
  const dx = Math.sqrt(3) * r
  const dy = 1.5 * r
  ctx.lineWidth = 2
  for (let row = -1; row * dy < h + r; row++) {
    for (let col = -1; col * dx < W + dx; col++) {
      const cx = col * dx + (row % 2 ? dx / 2 : 0)
      const cy = row * dy
      const fade = Math.max(0.25, 1 - Math.hypot(W - cx, cy) / (W * 1.25))
      ctx.globalAlpha = fade
      ctx.strokeStyle = p.comb
      hexPath(ctx, cx, cy, r - 3)
      ctx.stroke()
    }
  }
  ctx.globalAlpha = 1

  // The signature mark: a small cluster of filled cells, one honey-coloured, in the bottom-right
  // corner of the frame (the top-right holds the section pill).
  let lastRow = Math.round((h - FOOT / 2) / dy)
  while (lastRow * dy - r < h - FOOT + 8) lastRow++
  const cell = (col: number, row: number) => [col * dx + (row % 2 ? dx / 2 : 0), row * dy] as const
  ctx.fillStyle = p.glow
  for (const [col, row] of [[10, lastRow], [11, lastRow - 1], [11, lastRow + 1], [0, 2], [0, 3]]) {
    const [cx, cy] = cell(col, row)
    hexPath(ctx, cx, cy, r - 3)
    ctx.fill()
  }
  ctx.fillStyle = p.honey
  const [hx, hy] = cell(11, lastRow)
  hexPath(ctx, hx, hy, r - 3)
  ctx.fill()
}

function drawLogo(ctx: Ctx, p: Palette, x: number, cy: number) {
  ctx.fillStyle = '#ffffff'
  hexPath(ctx, x + 30, cy, 30)
  ctx.fill()
  ctx.fillStyle = p.stops[1]
  hexPath(ctx, x + 30, cy, 15)
  ctx.fill()
  ctx.fillStyle = p.honey
  hexPath(ctx, x + 30, cy, 7)
  ctx.fill()
}

function drawHeader(ctx: Ctx, p: Palette, card: ShareCard) {
  const cy = HEAD / 2 + 12
  drawLogo(ctx, p, FRAME, cy)
  ctx.textBaseline = 'middle'
  ctx.textAlign = 'left'
  ctx.fillStyle = '#ffffff'
  ctx.font = `700 40px ${SANS}`
  ctx.fillText(card.brand, FRAME + 80, cy)
  const brandEnd = FRAME + 80 + ctx.measureText(card.brand).width

  if (card.label) {
    ctx.font = `500 28px ${SANS}`
    let label = card.label
    const maxW = W - FRAME - brandEnd - 40 - 48
    while (label.length > 1 && ctx.measureText(label).width > maxW) label = label.slice(0, -1)
    if (label !== card.label) label = label.slice(0, -1) + '…'
    const w = ctx.measureText(label).width + 48
    const x = W - FRAME - w
    ctx.fillStyle = p.pill
    ctx.beginPath()
    ctx.roundRect(x, cy - 28, w, 56, 28)
    ctx.fill()
    ctx.fillStyle = '#ffffff'
    ctx.fillText(label, x + 24, cy + 1)
  }
  ctx.textBaseline = 'alphabetic'
}

function drawFooter(ctx: Ctx, card: ShareCard, h: number) {
  let host = card.footer || ''
  if (!host) {
    try {
      host = new URL(card.url).host
    } catch {
      host = ''
    }
  }
  if (!host) return
  ctx.textAlign = 'center'
  ctx.fillStyle = 'rgba(255,255,255,0.86)'
  ctx.font = `500 30px ${SANS}`
  ctx.fillText(host, W / 2, h - FOOT / 2 + 4)
  ctx.textAlign = 'left'
}

function textBlock(ctx: Ctx, text: string, font: string, color: string, lineHeight: number, maxLines: number, width = TEXT_W, x = FRAME + PAD): Laid {
  ctx.font = font
  const measure = (s: string) => ctx.measureText(s).width
  const lines = clampLines(wrapText(text, width, measure), maxLines, width, measure)
  return {
    height: lines.length * lineHeight,
    paint: (c, y) => {
      c.font = font
      c.fillStyle = color
      lines.forEach((line, i) => c.fillText(line, x, y + i * lineHeight + lineHeight * 0.72))
    },
  }
}

function codeBlock(ctx: Ctx, p: Palette, text: string, maxLines: number): Laid {
  const font = `400 30px ${MONO}`
  const lh = 46
  const inner = 28
  ctx.font = font
  const measure = (s: string) => ctx.measureText(s).width
  const raw = text.replace(/\t/g, '  ').split('\n')
  const lines: string[] = []
  for (const line of raw) {
    let rest = line
    if (!rest) lines.push('')
    while (rest) {
      let cut = rest.length
      while (cut > 1 && measure(rest.slice(0, cut)) > TEXT_W - inner * 2) cut--
      lines.push(rest.slice(0, cut))
      rest = rest.slice(cut)
    }
  }
  const shown = lines.length > maxLines ? [...lines.slice(0, maxLines - 1), '…'] : lines
  const height = shown.length * lh + inner * 2
  return {
    height,
    paint: (c, y) => {
      c.fillStyle = p.code
      c.beginPath()
      c.roundRect(FRAME + PAD, y, TEXT_W, height, 18)
      c.fill()
      c.font = font
      c.fillStyle = p.ink
      shown.forEach((line, i) => c.fillText(line, FRAME + PAD + inner, y + inner + i * lh + lh * 0.72))
    },
  }
}

function stack(items: { laid: Laid; gap: number }[]): Laid {
  let height = 0
  const at: number[] = []
  items.forEach((it, i) => {
    if (i) height += it.gap
    at.push(height)
    height += it.laid.height
  })
  return { height, paint: (c, y) => items.forEach((it, i) => it.laid.paint(c, y + at[i])) }
}

function qrRow(ctx: Ctx, p: Palette, card: ShareCard, qr: HTMLCanvasElement): Laid {
  const x = FRAME + PAD + QR + 40
  const width = TEXT_W - QR - 40
  const lines: Laid[] = [textBlock(ctx, card.cta || '扫码阅读全文', `700 38px ${SANS}`, p.ink, 52, 1, width, x)]
  if (card.meta) lines.push(textBlock(ctx, card.meta, `400 28px ${SANS}`, p.muted, 42, 2, width, x))
  if (card.sharer) lines.push(textBlock(ctx, card.sharer, `500 28px ${SANS}`, p.accent, 42, 1, width, x))
  if (card.source) lines.push(textBlock(ctx, card.source, `400 24px ${SANS}`, p.muted, 36, 2, width, x))
  const right = stack(lines.map((laid, i) => ({ laid, gap: i ? 6 : 0 })))
  const height = Math.max(QR, right.height) + 48
  return {
    height,
    paint: (c, y) => {
      c.strokeStyle = p.line
      c.lineWidth = 2
      c.setLineDash([10, 10])
      c.beginPath()
      c.moveTo(FRAME + PAD, y)
      c.lineTo(FRAME + PAD + TEXT_W, y)
      c.stroke()
      c.setLineDash([])
      const top = y + 48
      c.fillStyle = '#ffffff'
      c.beginPath()
      c.roundRect(FRAME + PAD - 8, top - 8, QR + 16, QR + 16, 16)
      c.fill()
      c.drawImage(qr, FRAME + PAD, top, QR, QR)
      right.paint(c, top + (QR - right.height) / 2)
    },
  }
}

function quoteBody(ctx: Ctx, p: Palette, card: QuoteCard, budget: number): Laid {
  const items: { laid: Laid; gap: number }[] = []
  const textFont = `500 46px ${SANS}`
  const lh = 78
  let left = budget
  for (const b of card.blocks) {
    if (left < lh) break
    const maxLines = Math.max(1, Math.floor(left / lh))
    const laid = b.kind === 'code' ? codeBlock(ctx, p, b.text, Math.max(2, Math.floor((left - 56) / 46))) : textBlock(ctx, b.text, textFont, p.ink, lh, maxLines)
    items.push({ laid, gap: 30 })
    left -= laid.height + 30
  }
  return stack(items)
}

function layoutQuote(ctx: Ctx, p: Palette, card: QuoteCard, qr: HTMLCanvasElement): { height: number; paint: Painter } {
  const mark: Laid = {
    height: 92,
    paint: (c, y) => {
      c.font = `700 180px ${SERIF}`
      c.fillStyle = p.accentSoft
      c.fillText('“', FRAME + PAD - 12, y + 150)
      c.fillStyle = p.accent
      c.beginPath()
      c.roundRect(FRAME + PAD, y + 70, 72, 8, 4)
      c.fill()
    },
  }
  const title = textBlock(ctx, `—— 《${card.title}》`, `500 32px ${SANS}`, p.muted, 48, 2)
  const footer = qrRow(ctx, p, card, qr)
  const fixed = HEAD + FOOT + PAD * 2 + mark.height + 36 + 44 + title.height + 52 + footer.height
  const body = quoteBody(ctx, p, card, MAX_H - fixed)
  const content = stack([
    { laid: mark, gap: 0 },
    { laid: body, gap: 36 },
    { laid: title, gap: 44 },
    { laid: footer, gap: 52 },
  ])
  return finishLayout(content)
}

function layoutSummary(ctx: Ctx, p: Palette, card: SummaryCard, qr: HTMLCanvasElement): { height: number; paint: Painter } {
  const items: { laid: Laid; gap: number }[] = []
  items.push({ laid: textBlock(ctx, card.title, `700 60px ${SANS}`, p.ink, 84, 3), gap: 0 })
  if (card.summary) items.push({ laid: textBlock(ctx, card.summary, `400 32px ${SANS}`, p.muted, 54, 3), gap: 28 })
  const bar: Laid = {
    height: 8,
    paint: (c, y) => {
      c.fillStyle = p.accent
      c.beginPath()
      c.roundRect(FRAME + PAD, y, 72, 8, 4)
      c.fill()
    },
  }
  items.push({ laid: bar, gap: 44 })
  const points = card.points.filter(Boolean).slice(0, 4)
  points.forEach((point, i) => {
    const indent = 76
    const text = textBlock(ctx, point, `500 38px ${SANS}`, p.ink, 58, 2, TEXT_W - indent, FRAME + PAD + indent)
    const laid: Laid = {
      height: text.height,
      paint: (c, y) => {
        const cx = FRAME + PAD + 26
        const cy = y + 29
        c.fillStyle = p.accent
        hexPath(c, cx, cy, 26)
        c.fill()
        c.fillStyle = p.paper
        c.font = `700 24px ${SANS}`
        c.textAlign = 'center'
        c.fillText(String(i + 1), cx, cy + 9)
        c.textAlign = 'left'
        text.paint(c, y)
      },
    }
    items.push({ laid, gap: i ? 30 : 44 })
  })
  items.push({ laid: qrRow(ctx, p, card, qr), gap: 60 })
  return finishLayout(stack(items))
}

/** Card height from its content, the canvas between MIN_H and MAX_H; extra room goes above the QR row. */
function finishLayout(content: Laid): { height: number; paint: Painter } {
  const cardH = content.height + PAD * 2
  const height = Math.min(MAX_H, Math.max(MIN_H, HEAD + cardH + FOOT))
  const extra = height - (HEAD + cardH + FOOT)
  return {
    height,
    paint: (ctx, y) => content.paint(ctx, y + PAD + Math.max(0, extra) / 2),
  }
}

/** Draws the card onto `canvas` (resized to fit) and returns it. */
export async function drawShareCard(card: ShareCard, canvas: HTMLCanvasElement = document.createElement('canvas')): Promise<HTMLCanvasElement> {
  if (typeof document !== 'undefined' && document.fonts?.ready) await document.fonts.ready
  const p = PALETTES[card.theme] || PALETTES.violet
  const qr = document.createElement('canvas')
  await QRCode.toCanvas(qr, card.url, { width: QR * 2, margin: 1, errorCorrectionLevel: 'M', color: { dark: '#111111', light: '#ffffff' } })

  canvas.width = W
  canvas.height = MIN_H
  const measureCtx = canvas.getContext('2d')
  if (!measureCtx) throw new Error('canvas unavailable')
  const layout = card.kind === 'quote' ? layoutQuote(measureCtx, p, card, qr) : layoutSummary(measureCtx, p, card, qr)

  canvas.height = layout.height
  const ctx = canvas.getContext('2d')!
  const h = layout.height
  drawBackground(ctx, p, h)
  drawHeader(ctx, p, card)

  ctx.save()
  ctx.shadowColor = 'rgba(0,0,0,0.28)'
  ctx.shadowBlur = 48
  ctx.shadowOffsetY = 18
  ctx.fillStyle = p.paper
  ctx.beginPath()
  ctx.roundRect(FRAME, HEAD, W - FRAME * 2, h - HEAD - FOOT, 40)
  ctx.fill()
  ctx.restore()

  ctx.textAlign = 'left'
  ctx.textBaseline = 'alphabetic'
  layout.paint(ctx, HEAD)
  drawFooter(ctx, card, h)
  return canvas
}
