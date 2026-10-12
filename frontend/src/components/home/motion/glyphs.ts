// Token glyphs drawn once into an 8×8 atlas. Every other one sits in a chip (a rounded box), so the
// stream reads as tokens rather than loose letters. Only characters every Chinese system font has.
import { CanvasTexture } from 'three'

export const GLYPHS = ['你', '说', '画', '写', '码', '图', '句', '话', '老', '板', '打', '工', '数', '据', '流', '光',
  'AI', 'GPT', 'tok', 'en', 'ing', '{}', '</>', '01', '10', '$', '¥', '#', '=>', '++', '的', '一']

export const FONT = '-apple-system, BlinkMacSystemFont, "PingFang SC", "Noto Sans SC", "Microsoft YaHei", sans-serif'

export function roundRect(g: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  g.beginPath()
  g.moveTo(x + r, y)
  g.arcTo(x + w, y, x + w, y + h, r)
  g.arcTo(x + w, y + h, x, y + h, r)
  g.arcTo(x, y + h, x, y, r)
  g.arcTo(x, y, x + w, y, r)
  g.closePath()
}

export function glyphAtlas() {
  const N = 8, S = 64
  const c = document.createElement('canvas')
  c.width = c.height = N * S
  const g = c.getContext('2d')!
  g.textAlign = 'center'
  g.textBaseline = 'middle'
  GLYPHS.forEach((ch, i) => {
    const x = (i % N) * S, y = Math.floor(i / N) * S
    const chip = i % 2 === 1
    const size = ch.length > 2 ? (chip ? 18 : 22) : ch.length === 2 ? (chip ? 22 : 28) : chip ? 30 : 38
    if (chip) {
      g.strokeStyle = 'rgba(255,255,255,.9)'
      g.lineWidth = 2.5
      roundRect(g, x + 6, y + 12, S - 12, S - 24, 8)
      g.stroke()
      g.fillStyle = 'rgba(255,255,255,.14)'
      g.fill()
    }
    g.fillStyle = '#fff'
    g.font = `600 ${size}px ${FONT}`
    g.fillText(ch, x + S / 2, y + S / 2 + 2)
  })
  return { tex: new CanvasTexture(c), count: GLYPHS.length, grid: N }
}
