// 小红书 image cards for the content engine: a 3:4 cover with the big title, one card per point
// and a closing card, drawn on a <canvas> in the browser (no image-generation cost). No links or QR
// codes — 小红书 limits posts that carry them; readers are pointed to search the brand instead.
import { wrapText } from './shareCard'

export interface NoteCardData {
  title: string
  points: string[]
}

export type NoteTheme = 'cream' | 'mint' | 'peach' | 'ink'
export const NOTE_THEMES: NoteTheme[] = ['cream', 'mint', 'peach', 'ink']
export const NOTE_THEME_NAMES: Record<NoteTheme, string> = { cream: '奶油', mint: '薄荷', peach: '蜜桃', ink: '墨夜' }

interface Palette {
  bg: string
  card: string
  accent: string
  title: string
  text: string
  muted: string
}

const PALETTES: Record<NoteTheme, Palette> = {
  cream: { bg: '#fbf3e4', card: '#ffffff', accent: '#f2a93b', title: '#2b2118', text: '#3d3329', muted: '#9a8b78' },
  mint: { bg: '#e3f4ee', card: '#ffffff', accent: '#2fa37c', title: '#123b2e', text: '#24453a', muted: '#7a9a8e' },
  peach: { bg: '#fde8e4', card: '#ffffff', accent: '#ef6a5b', title: '#3b1a16', text: '#4a2a25', muted: '#a8827c' },
  ink: { bg: '#1d2130', card: '#272c3f', accent: '#f5c451', title: '#ffffff', text: '#e4e6ef', muted: '#9097b0' }
}

export const NOTE_W = 1080
export const NOTE_H = 1440
const PAD = 96
const FONT = '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif'

export interface NoteLayout {
  cover: boolean
  titleSize: number
  titleLines: string[]
  pointSize: number
  points: string[][] // wrapped lines per point
  height: number // of the whole text block, for vertical centring
}

const LINE = 1.6 // point line height, in font sizes
const GAP = 0.9 // between points
const CARD_PAD = 64 // inside the white card
const NUMBER_H = 110 // the 01 / 02 above a heading
const AVAILABLE = NOTE_H - PAD * 2 - 90 // above the footer

function pointsHeight(points: string[][], size: number) {
  return points.reduce((h, lines) => h + lines.length * size * LINE, 0) + Math.max(0, points.length - 1) * size * GAP
}

/**
 * Lays a card out: the cover (index 0) gets a big title and the points as a subtitle; the other
 * cards a heading and bullet points on a white card. Font sizes shrink until everything fits.
 */
export function layoutNoteCard(card: NoteCardData, index: number, measure: (s: string, size: number) => number): NoteLayout {
  const cover = index === 0
  const textWidth = NOTE_W - PAD * 2 - (cover ? 0 : CARD_PAD * 2 + 40)
  const titleWidth = NOTE_W - PAD * 2
  let last: NoteLayout | null = null
  for (let scale = 1; scale >= 0.5; scale -= 0.05) {
    const titleSize = Math.round((cover ? 128 : 88) * scale)
    const pointSize = Math.round((cover ? 56 : 58) * scale)
    const titleLines = wrapText(card.title || '', titleWidth, (s) => measure(s, titleSize)).filter((l) => l !== '')
    const points = card.points.map((p) => wrapText(p, textWidth, (s) => measure(s, pointSize)).filter((l) => l !== ''))
    const titleH = titleLines.length * titleSize * 1.3
    const height = cover
      ? titleH + 70 + pointsHeight(points, pointSize)
      : NUMBER_H + titleH + 56 + pointsHeight(points, pointSize) + CARD_PAD * 2
    last = { cover, titleSize, titleLines, pointSize, points, height }
    if (height <= AVAILABLE) return last
  }
  return last!
}

/** Draws card `index` of `total` and returns the canvas (1080×1440). */
export async function drawNoteCard(
  card: NoteCardData,
  index: number,
  total: number,
  opts: { theme: NoteTheme; brand: string },
  canvas: HTMLCanvasElement = document.createElement('canvas')
): Promise<HTMLCanvasElement> {
  if (typeof document !== 'undefined' && document.fonts?.ready) await document.fonts.ready
  const p = PALETTES[opts.theme] || PALETTES.cream
  canvas.width = NOTE_W
  canvas.height = NOTE_H
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas unavailable')
  const measure = (s: string, size: number) => {
    ctx.font = `600 ${size}px ${FONT}`
    return ctx.measureText(s).width
  }
  const layout = layoutNoteCard(card, index, measure)

  ctx.fillStyle = p.bg
  ctx.fillRect(0, 0, NOTE_W, NOTE_H)
  // A soft accent blob, top right.
  ctx.fillStyle = p.accent
  ctx.globalAlpha = 0.16
  ctx.beginPath()
  ctx.arc(NOTE_W - 120, 140, 260, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = 1
  ctx.textBaseline = 'alphabetic'
  ctx.textAlign = 'left'

  // The text block is centred vertically (the cover a little above the middle).
  const free = Math.max(0, AVAILABLE - layout.height)
  let y = PAD + (layout.cover ? free * 0.4 : free / 2)
  const ps = layout.pointSize

  if (layout.cover) {
    ctx.fillStyle = p.accent
    ctx.beginPath()
    ctx.roundRect(PAD, y - 40, 150, 16, 8)
    ctx.fill()
    ctx.fillStyle = p.title
    ctx.font = `800 ${layout.titleSize}px ${FONT}`
    for (const line of layout.titleLines) {
      y += layout.titleSize * 1.3
      ctx.fillText(line, PAD, y)
    }
    y += 70
    ctx.fillStyle = p.text
    ctx.font = `500 ${ps}px ${FONT}`
    layout.points.forEach((lines, i) => {
      if (i > 0) y += ps * GAP
      for (const line of lines) {
        y += ps * LINE
        ctx.fillText(line, PAD, y - ps * 0.4)
      }
    })
  } else {
    ctx.fillStyle = p.accent
    ctx.font = `800 72px ${FONT}`
    ctx.fillText(String(index).padStart(2, '0'), PAD, y + 72)
    y += NUMBER_H
    ctx.fillStyle = p.title
    ctx.font = `800 ${layout.titleSize}px ${FONT}`
    for (const line of layout.titleLines) {
      y += layout.titleSize * 1.3
      ctx.fillText(line, PAD, y - layout.titleSize * 0.25)
    }
    y += 56
    const cardH = pointsHeight(layout.points, ps) + CARD_PAD * 2
    ctx.fillStyle = p.card
    ctx.beginPath()
    ctx.roundRect(PAD - 20, y, NOTE_W - (PAD - 20) * 2, cardH, 36)
    ctx.fill()
    y += CARD_PAD
    const textX = PAD - 20 + CARD_PAD + 40
    ctx.font = `500 ${ps}px ${FONT}`
    layout.points.forEach((lines, i) => {
      if (i > 0) y += ps * GAP
      lines.forEach((line, j) => {
        y += ps * LINE
        const baseline = y - ps * 0.42
        if (j === 0) {
          ctx.fillStyle = p.accent
          ctx.beginPath()
          ctx.arc(textX - 34, baseline - ps * 0.34, ps * 0.17, 0, Math.PI * 2)
          ctx.fill()
        }
        ctx.fillStyle = p.text
        ctx.fillText(line, textX, baseline)
      })
    })
  }

  // Footer: brand and page number.
  ctx.font = `500 32px ${FONT}`
  ctx.fillStyle = p.muted
  ctx.fillText(opts.brand, PAD, NOTE_H - PAD + 20)
  ctx.textAlign = 'right'
  ctx.fillText(`${index + 1} / ${total}`, NOTE_W - PAD, NOTE_H - PAD + 20)
  return canvas
}
