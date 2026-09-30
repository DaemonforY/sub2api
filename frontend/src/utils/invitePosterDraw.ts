import { canvasCopy, inviteCopy, minimalCopy, POSTER_HEIGHT as H, POSTER_WIDTH as W, posterInviteUrl, renderQr, studentCopy, wrapText, type PosterData, type PosterFooter, type PosterTemplateId } from './invitePoster'

type Translate = (key: string, params?: Record<string, unknown>) => string
type Ctx = CanvasRenderingContext2D

const SANS = '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif'
const SERIF = '"Songti SC", "STSong", "Noto Serif SC", "Source Han Serif SC", "SimSun", serif'
const PAD = 88

const NAVY = '#0a1a39'
const PAPER = '#f4f1e8'
const TEAL = '#0d9488'
const TEAL_LIGHT = '#2dd4bf'

/** Draws a poster into `canvas` (1080×1440). Resolves when every image (logo, QR) is on it. */
export async function drawInvitePoster(canvas: HTMLCanvasElement, template: PosterTemplateId, data: PosterData, t: Translate): Promise<void> {
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas unavailable')
  const [logo, qr] = await Promise.all([loadImage(data.logoUrl), renderQr(posterInviteUrl(data.inviteLink, template), 440)])
  ctx.textBaseline = 'alphabetic'
  switch (template) {
    case 'invite':
      return drawInvite(ctx, data, t, logo, qr)
    case 'student':
      return drawStudent(ctx, data, t, logo, qr)
    case 'canvas':
      return drawCanvasTheme(ctx, data, t, logo, qr)
    default:
      return drawMinimal(ctx, data, t, logo, qr)
  }
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

function drawInvite(ctx: Ctx, data: PosterData, t: Translate, logo: HTMLImageElement | null, qr: HTMLCanvasElement) {
  const copy = inviteCopy(data, t)
  const bg = ctx.createLinearGradient(0, 0, W, H)
  bg.addColorStop(0, '#0f2552')
  bg.addColorStop(1, NAVY)
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, H)
  honeycomb(ctx, W - 520, -40, 620, 560, 'rgba(45,212,191,0.10)')

  brand(ctx, logo, data.siteName, PAD, 96, '#ffffff')
  pill(ctx, copy.tag, W - PAD, 96, TEAL_LIGHT)

  ctx.fillStyle = '#ffffff'
  ctx.font = `900 100px ${SERIF}`
  copy.headline.forEach((line, i) => ctx.fillText(line, PAD, 300 + i * 124))

  ctx.fillStyle = 'rgba(255,255,255,0.72)'
  ctx.font = `30px ${SANS}`
  paragraph(ctx, copy.sub, PAD, 490, W - PAD * 2, 44)

  // Reward columns.
  const top = 580
  ctx.strokeStyle = 'rgba(255,255,255,0.14)'
  ctx.lineWidth = 2
  line(ctx, PAD, top, W - PAD, top)
  line(ctx, PAD, top + 380, W - PAD, top + 380)
  const colW = (W - PAD * 2) / copy.stats.length
  copy.stats.forEach((stat, i) => {
    const x = PAD + i * colW + (i ? 48 : 0)
    if (i) line(ctx, PAD + colW * i, top, PAD + colW * i, top + 380)
    ctx.fillStyle = '#ffffff'
    ctx.font = `600 32px ${SANS}`
    ctx.fillText(stat.label, x, top + 64)
    const grad = ctx.createLinearGradient(x, top + 100, x + 300, top + 250)
    grad.addColorStop(0, '#5eead4')
    grad.addColorStop(1, '#22d3ee')
    ctx.fillStyle = grad
    ctx.font = `900 170px ${SERIF}`
    ctx.fillText(stat.value, x - 6, top + 250)
    const numberWidth = ctx.measureText(stat.value).width
    ctx.font = `700 64px ${SANS}`
    ctx.fillText('%', x + numberWidth + 4, top + 250)
    ctx.fillStyle = 'rgba(255,255,255,0.72)'
    ctx.font = `28px ${SANS}`
    paragraph(ctx, stat.desc, x, top + 312, colW - 64, 40)
  })

  ctx.fillStyle = 'rgba(255,255,255,0.85)'
  ctx.font = `30px ${SANS}`
  paragraph(ctx, copy.example, PAD, top + 450, W - PAD * 2, 44)

  footerBand(ctx, copy.footer, qr, { bg: PAPER, ink: NAVY, muted: 'rgba(10,26,57,0.6)' })
}

function drawStudent(ctx: Ctx, data: PosterData, t: Translate, logo: HTMLImageElement | null, qr: HTMLCanvasElement) {
  const copy = studentCopy(data, t)
  ctx.fillStyle = PAPER
  ctx.fillRect(0, 0, W, H)
  honeycomb(ctx, W - 560, -60, 640, 560, 'rgba(10,26,57,0.07)')

  brand(ctx, logo, data.siteName, PAD, 96, NAVY)
  ctx.fillStyle = 'rgba(10,26,57,0.6)'
  ctx.font = `24px ${SANS}`
  ctx.textAlign = 'right'
  ctx.fillText(data.siteHost, W - PAD, 106)
  ctx.textAlign = 'left'

  ctx.fillStyle = TEAL
  ctx.fillRect(PAD, 214, 44, 4)
  ctx.font = `600 28px ${SANS}`
  ctx.fillText(copy.tag, PAD + 62, 226)

  ctx.font = `900 108px ${SERIF}`
  ctx.fillStyle = NAVY
  ctx.fillText(copy.headline[0], PAD, 380)
  ctx.fillText(copy.headline[1], PAD, 510)
  const lead = ctx.measureText(copy.headline[1]).width
  ctx.fillStyle = TEAL
  ctx.fillText(copy.accent, PAD + lead, 510)

  ctx.fillStyle = '#3b4a66'
  ctx.font = `30px ${SANS}`
  paragraph(ctx, copy.sub, PAD, 590, W - PAD * 2, 46)

  ctx.strokeStyle = NAVY
  ctx.lineWidth = 3
  line(ctx, PAD, 690, W - PAD, 690)
  copy.items.forEach((item, i) => {
    const y = 770 + i * 118
    ctx.fillStyle = TEAL
    ctx.font = `700 34px ${SERIF}`
    ctx.fillText(`0${i + 1}`, PAD, y)
    ctx.fillStyle = NAVY
    ctx.font = `700 34px ${SANS}`
    ctx.fillText(item.title, PAD + 86, y)
    ctx.fillStyle = '#3b4a66'
    ctx.font = `26px ${SANS}`
    ctx.fillText(ellipsize(ctx, item.desc, W - PAD * 2 - 86), PAD + 86, y + 42)
  })

  footerBand(ctx, copy.footer, qr, { bg: NAVY, ink: '#ffffff', muted: 'rgba(255,255,255,0.6)' })
}

function drawCanvasTheme(ctx: Ctx, data: PosterData, t: Translate, logo: HTMLImageElement | null, qr: HTMLCanvasElement) {
  const copy = canvasCopy(data, t)
  const bg = ctx.createLinearGradient(0, 0, W, H)
  bg.addColorStop(0, '#1e1b4b')
  bg.addColorStop(0.55, '#6d28d9')
  bg.addColorStop(1, '#db2777')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, H)
  glow(ctx, 900, 220, 360, 'rgba(34,211,238,0.35)')
  glow(ctx, 120, 820, 420, 'rgba(251,191,36,0.22)')

  brand(ctx, logo, data.siteName, PAD, 96, '#ffffff')
  pill(ctx, copy.tag, W - PAD, 96, '#fde68a')

  ctx.fillStyle = '#ffffff'
  ctx.font = `900 108px ${SERIF}`
  copy.headline.forEach((text, i) => ctx.fillText(text, PAD, 310 + i * 132))
  ctx.fillStyle = 'rgba(255,255,255,0.8)'
  ctx.font = `30px ${SANS}`
  paragraph(ctx, copy.sub, PAD, 520, W - PAD * 2, 46)

  // A mock prompt box → result, to show what the product does.
  const boxY = 620
  ctx.font = `30px ${SANS}`
  const promptLines = Math.min(3, wrapText(copy.prompt, W - PAD * 2 - 72, (s) => ctx.measureText(s).width).length)
  const boxH = 56 + promptLines * 44
  roundRect(ctx, PAD, boxY, W - PAD * 2, boxH, 28)
  ctx.fillStyle = 'rgba(255,255,255,0.14)'
  ctx.fill()
  ctx.strokeStyle = 'rgba(255,255,255,0.35)'
  ctx.lineWidth = 2
  ctx.stroke()
  ctx.fillStyle = '#ffffff'
  paragraph(ctx, copy.prompt, PAD + 36, boxY + 62, W - PAD * 2 - 72, 44)
  ctx.fillStyle = '#fde68a'
  ctx.font = `700 32px ${SANS}`
  ctx.fillText(copy.result, PAD, boxY + boxH + 70)

  let chipX = PAD
  ctx.font = `28px ${SANS}`
  for (const chip of copy.chips) {
    const w = ctx.measureText(chip).width + 48
    roundRect(ctx, chipX, boxY + boxH + 110, w, 60, 30)
    ctx.fillStyle = 'rgba(255,255,255,0.16)'
    ctx.fill()
    ctx.fillStyle = '#ffffff'
    ctx.fillText(chip, chipX + 24, boxY + boxH + 150)
    chipX += w + 16
  }

  footerBand(ctx, copy.footer, qr, { bg: '#ffffff', ink: '#1e1b4b', muted: 'rgba(30,27,75,0.6)' })
}

function drawMinimal(ctx: Ctx, data: PosterData, t: Translate, logo: HTMLImageElement | null, qr: HTMLCanvasElement) {
  const copy = minimalCopy(data, t)
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, W, H)
  ctx.strokeStyle = '#e5e7eb'
  ctx.lineWidth = 4
  roundRect(ctx, 48, 48, W - 96, H - 96, 40)
  ctx.stroke()
  ctx.textAlign = 'center'

  const cx = W / 2
  if (logo) ctx.drawImage(logo, cx - 60, 150, 120, 120)
  ctx.fillStyle = NAVY
  ctx.font = `800 64px ${SANS}`
  ctx.fillText(data.siteName, cx, 350)
  ctx.fillStyle = '#6b7280'
  ctx.font = `30px ${SANS}`
  ctx.fillText(copy.tagline, cx, 406)

  ctx.fillStyle = '#111827'
  ctx.font = `600 38px ${SANS}`
  ctx.fillText(copy.invite, cx, 520)

  roundRect(ctx, cx - 250, 572, 500, 500, 32)
  ctx.fillStyle = '#f9fafb'
  ctx.fill()
  ctx.drawImage(qr, cx - 220, 602, 440, 440)

  ctx.fillStyle = TEAL
  ctx.font = `700 34px ${SANS}`
  ctx.fillText(copy.offer, cx, 1150)
  ctx.fillStyle = '#374151'
  ctx.font = `30px ${SANS}`
  ctx.fillText(copy.footer.code, cx, 1210)
  ctx.fillStyle = '#9ca3af'
  ctx.font = `24px ${SANS}`
  ctx.fillText(copy.footer.fine, cx, 1330)
  ctx.textAlign = 'left'
}

// ---------------------------------------------------------------------------
// Building blocks
// ---------------------------------------------------------------------------

function footerBand(ctx: Ctx, footer: PosterFooter, qr: HTMLCanvasElement, colors: { bg: string; ink: string; muted: string }) {
  const top = H - 330
  ctx.fillStyle = colors.bg
  ctx.fillRect(0, top, W, 330)
  const qrSize = 240
  const qrX = W - PAD - qrSize
  const qrY = top + 45
  roundRect(ctx, qrX - 14, qrY - 14, qrSize + 28, qrSize + 28, 20)
  ctx.fillStyle = '#ffffff'
  ctx.fill()
  ctx.drawImage(qr, qrX, qrY, qrSize, qrSize)

  const textW = qrX - PAD - 40
  ctx.fillStyle = colors.ink
  ctx.font = `700 44px ${SANS}`
  ctx.fillText(ellipsize(ctx, footer.cta, textW), PAD, top + 110)
  ctx.font = `600 32px ${SANS}`
  ctx.fillText(ellipsize(ctx, footer.code, textW), PAD, top + 170)
  ctx.fillStyle = colors.muted
  ctx.font = `22px ${SANS}`
  paragraph(ctx, footer.fine, PAD, top + 232, textW, 34)
}

function brand(ctx: Ctx, logo: HTMLImageElement | null, name: string, x: number, y: number, color: string) {
  let textX = x
  if (logo) {
    ctx.drawImage(logo, x, y - 44, 60, 60)
    textX = x + 76
  }
  ctx.fillStyle = color
  ctx.font = `700 38px ${SANS}`
  ctx.fillText(name, textX, y)
}

function pill(ctx: Ctx, text: string, right: number, y: number, color: string) {
  ctx.font = `600 26px ${SANS}`
  const w = ctx.measureText(text).width + 56
  roundRect(ctx, right - w, y - 44, w, 64, 32)
  ctx.strokeStyle = color
  ctx.lineWidth = 2
  ctx.stroke()
  ctx.fillStyle = color
  ctx.fillText(text, right - w + 28, y - 3)
}

function paragraph(ctx: Ctx, text: string, x: number, y: number, maxWidth: number, lineHeight: number, maxLines = 3) {
  const lines = wrapText(text, maxWidth, (s) => ctx.measureText(s).width).slice(0, maxLines)
  lines.forEach((l, i) => ctx.fillText(l, x, y + i * lineHeight))
}

function ellipsize(ctx: Ctx, text: string, maxWidth: number): string {
  if (ctx.measureText(text).width <= maxWidth) return text
  let out = text
  while (out.length > 1 && ctx.measureText(`${out}…`).width > maxWidth) out = out.slice(0, -1)
  return `${out}…`
}

function line(ctx: Ctx, x1: number, y1: number, x2: number, y2: number) {
  ctx.beginPath()
  ctx.moveTo(x1, y1)
  ctx.lineTo(x2, y2)
  ctx.stroke()
}

function roundRect(ctx: Ctx, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

function honeycomb(ctx: Ctx, x0: number, y0: number, w: number, h: number, color: string) {
  ctx.save()
  const fade = ctx.createRadialGradient(x0 + w * 0.7, y0 + h * 0.3, 0, x0 + w * 0.7, y0 + h * 0.3, w * 0.7)
  fade.addColorStop(0, color)
  fade.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.strokeStyle = fade
  ctx.lineWidth = 2
  const size = 36
  const dx = size * Math.sqrt(3)
  for (let row = 0; row * size * 1.5 < h; row += 1) {
    for (let col = 0; col * dx < w + dx; col += 1) {
      const cx = x0 + col * dx + (row % 2 ? dx / 2 : 0)
      const cy = y0 + row * size * 1.5
      ctx.beginPath()
      for (let k = 0; k < 6; k += 1) {
        const a = (Math.PI / 3) * k + Math.PI / 6
        const px = cx + size * Math.cos(a)
        const py = cy + size * Math.sin(a)
        if (k) ctx.lineTo(px, py)
        else ctx.moveTo(px, py)
      }
      ctx.closePath()
      ctx.stroke()
    }
  }
  ctx.restore()
}

function glow(ctx: Ctx, x: number, y: number, r: number, color: string) {
  const g = ctx.createRadialGradient(x, y, 0, x, y, r)
  g.addColorStop(0, color)
  g.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.fillStyle = g
  ctx.fillRect(x - r, y - r, r * 2, r * 2)
}

/** Loads a same-origin (or CORS-enabled) image; resolves null instead of tainting the canvas. */
function loadImage(src: string): Promise<HTMLImageElement | null> {
  if (!src) return Promise.resolve(null)
  return new Promise((resolve) => {
    const img = new Image()
    if (/^https?:/i.test(src) && !src.startsWith(window.location.origin)) img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = () => resolve(null)
    img.src = src
  })
}
