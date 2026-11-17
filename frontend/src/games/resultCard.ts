// 游戏战绩卡: one 1080×1440 PNG shared by the co-op games, built for WeChat Moments.
//
// Layout (what people screenshot is one honour title + one number over the game's own art):
//   0–900     key art, cover-fitted, fading into a deep colour sampled from the art
//   ~760–1180 kicker, the 称号 (hero), score, one meta line, duo pills, a progress row
//   1210–1400 challenge line + site on the left, QR (with the sharer's invite code) on the right
// The honour title and score sit inside the centre square (y≈180–1260) that the Moments grid crops to.
// No outlined text: a soft shadow over a darkened band keeps it readable.

import QRCode from 'qrcode'
import { getAffiliateDetail } from '@/api/user'
import { useAuthStore } from '@/stores/auth'
import { shareUrl } from '@/utils/shareCard'

export interface ResultCardPlayer {
  name: string
  score: number
  color: string
}

export interface ResultCardInput {
  /** route slug, e.g. 'cotton' → /games/cotton */
  game: string
  title: string
  art: string
  /** horizontal focus of the art, 0 = left, 1 = right */
  focusX?: number
  /** colour the art fades into (sampled from the art), and the score / progress colour */
  deep: string
  accent: string
  kicker: string
  honor: string
  score: number
  meta: string
  players?: ResultCardPlayer[]
  progress?: { done: number; total: number }
  challenge: string
}

const W = 1080
const H = 1440
const FONT = '"PingFang SC","HarmonyOS Sans SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC",system-ui,sans-serif'

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}

function rounded(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number): void {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

/** Largest size ≤ max at which text fits maxWidth. */
function fit(ctx: CanvasRenderingContext2D, text: string, weight: number, max: number, min: number, maxWidth: number): number {
  let size = max
  for (; size > min; size -= 2) {
    ctx.font = `${weight} ${size}px ${FONT}`
    if (ctx.measureText(text).width <= maxWidth) break
  }
  return size
}

function rgba(hex: string, a: number): string {
  const n = parseInt(hex.replace('#', ''), 16)
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${a})`
}

let cachedAff: { uid: number; code: string } | null = null

/** The QR target: the game page with the sharer's invite code when they are signed in. */
export async function resultCardLink(game: string): Promise<string> {
  let aff = ''
  try {
    const auth = useAuthStore()
    const uid = auth.user?.id
    if (auth.isAuthenticated && uid) {
      if (cachedAff?.uid !== uid) cachedAff = { uid, code: (await getAffiliateDetail()).aff_code || '' }
      aff = cachedAff.code
    }
  } catch {
    aff = ''
  }
  return shareUrl(`${location.origin}/games/${game}`, { aff, medium: 'game_card', campaign: game })
}

/** Draws the card and returns a PNG data URL. */
export async function drawGameResultCard(input: ResultCardInput, link?: string): Promise<string> {
  const c = document.createElement('canvas')
  c.width = W
  c.height = H
  const ctx = c.getContext('2d')!
  ctx.fillStyle = input.deep
  ctx.fillRect(0, 0, W, H)

  // Layout, bottom up: the footer is fixed; the result stack ends 60px above it.
  const L = 64
  const maxW = W - L * 2
  const QR = 200
  const PAD = 16
  const qx = W - 64 - QR - PAD * 2
  const qy = H - 64 - QR - PAD * 2
  const honor = `「${input.honor}」`
  const hs = fit(ctx, honor, 900, 112, 64, maxW + 30)
  const duo = !!(input.players && input.players.length > 1)
  const prog = !!(input.progress && input.progress.total > 0)
  const stackH = 36 + 24 + hs + 112 + 58 + (duo ? 110 : 34) + (prog ? 48 : 0)
  const kickerY = qy - 60 - stackH + 36

  // Art: cover-fit into the top, fading into the deep colour behind the result stack.
  const ART_H = Math.max(820, kickerY + 220)
  try {
    const art = await loadImage(input.art)
    const scale = Math.max(W / art.width, ART_H / art.height)
    const dw = art.width * scale
    const dh = art.height * scale
    const dx = (W - dw) * (input.focusX ?? 0.5)
    ctx.drawImage(art, dx, 0, dw, dh)
  } catch {
    /* plain colour */
  }
  const fadeTop = kickerY - 260
  const fade = ctx.createLinearGradient(0, fadeTop, 0, ART_H)
  fade.addColorStop(0, rgba(input.deep, 0))
  fade.addColorStop(0.5, rgba(input.deep, 0.78))
  fade.addColorStop(1, rgba(input.deep, 1))
  ctx.fillStyle = fade
  ctx.fillRect(0, fadeTop, W, ART_H - fadeTop)
  ctx.fillStyle = input.deep
  ctx.fillRect(0, ART_H, W, H - ART_H)
  const top = ctx.createLinearGradient(0, 0, 0, 200)
  top.addColorStop(0, 'rgba(0,0,0,0.45)')
  top.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.fillStyle = top
  ctx.fillRect(0, 0, W, 200)

  // Brand pill, top left.
  ctx.font = `700 30px ${FONT}`
  const brand = `HiveGPT 小游戏 · ${input.title}`
  const bw = ctx.measureText(brand).width + 48
  rounded(ctx, 48, 48, bw, 60, 30)
  ctx.fillStyle = 'rgba(0,0,0,0.45)'
  ctx.fill()
  ctx.fillStyle = '#fff'
  ctx.textBaseline = 'middle'
  ctx.fillText(brand, 72, 79)

  ctx.textBaseline = 'alphabetic'
  ctx.shadowColor = 'rgba(0,0,0,0.45)'
  ctx.shadowBlur = 18
  ctx.shadowOffsetY = 4

  // Kicker.
  ctx.font = `700 36px ${FONT}`
  ctx.fillStyle = rgba('#ffffff', 0.86)
  ctx.fillText(input.kicker, L, kickerY)

  // Honour title (hero).
  ctx.font = `900 ${hs}px ${FONT}`
  ctx.fillStyle = '#ffffff'
  ctx.fillText(honor, L - hs * 0.42, kickerY + 24 + hs)

  // Score.
  const scoreY = kickerY + 24 + hs + 112
  const num = input.score.toLocaleString('zh-CN')
  ctx.font = `900 104px ${FONT}`
  ctx.fillStyle = input.accent
  ctx.fillText(num, L, scoreY)
  const nw = ctx.measureText(num).width
  ctx.font = `800 44px ${FONT}`
  ctx.fillText('分', L + nw + 12, scoreY)
  ctx.shadowColor = 'transparent'

  // Meta line.
  ctx.font = `500 32px ${FONT}`
  ctx.fillStyle = rgba('#ffffff', 0.72)
  ctx.fillText(input.meta, L, scoreY + 58)
  let y = scoreY + 92

  // Duo: one pill per player, side by side.
  if (input.players && input.players.length > 1) {
    let x = L
    ctx.font = `700 30px ${FONT}`
    for (const p of input.players) {
      const label = `${p.name}  ${p.score.toLocaleString('zh-CN')}`
      const w = ctx.measureText(label).width + 76
      rounded(ctx, x, y, w, 56, 28)
      ctx.fillStyle = 'rgba(255,255,255,0.12)'
      ctx.fill()
      ctx.beginPath()
      ctx.arc(x + 30, y + 28, 10, 0, Math.PI * 2)
      ctx.fillStyle = p.color
      ctx.fill()
      ctx.fillStyle = '#fff'
      ctx.textBaseline = 'middle'
      ctx.fillText(label, x + 52, y + 29)
      ctx.textBaseline = 'alphabetic'
      x += w + 16
    }
    y += 76
  }

  // Progress row: one square per level/floor, filled up to where the run got.
  if (input.progress && input.progress.total > 0) {
    const { done, total } = input.progress
    const gap = 8
    const size = Math.min(44, Math.floor((maxW - gap * (total - 1)) / total))
    for (let i = 0; i < total; i++) {
      rounded(ctx, L + i * (size + gap), y + 4, size, size, 8)
      if (i < done) {
        ctx.fillStyle = input.accent
        ctx.fill()
      } else {
        ctx.strokeStyle = 'rgba(255,255,255,0.32)'
        ctx.lineWidth = 3
        ctx.stroke()
      }
    }
  }

  // Footer: challenge + site on the left, QR on the right.
  ctx.fillStyle = 'rgba(255,255,255,0.10)'
  ctx.fillRect(L, qy - 36, W - L * 2, 2)
  const qrCanvas = document.createElement('canvas')
  await QRCode.toCanvas(qrCanvas, link || (await resultCardLink(input.game)), {
    width: QR * 2,
    margin: 0,
    errorCorrectionLevel: 'M',
    color: { dark: '#111111', light: '#ffffff' }
  })
  rounded(ctx, qx, qy, QR + PAD * 2, QR + PAD * 2, 20)
  ctx.fillStyle = '#ffffff'
  ctx.fill()
  ctx.drawImage(qrCanvas, qx + PAD, qy + PAD, QR, QR)

  const textW = qx - L - 32
  const cs = fit(ctx, input.challenge, 800, 48, 32, textW)
  ctx.font = `800 ${cs}px ${FONT}`
  ctx.fillStyle = '#ffffff'
  ctx.fillText(input.challenge, L, qy + 70)
  ctx.font = `500 30px ${FONT}`
  ctx.fillStyle = rgba('#ffffff', 0.7)
  ctx.fillText('扫码和我一起玩 · 免费 · 不用登录', L, qy + 130)
  ctx.font = `600 28px ${FONT}`
  ctx.fillStyle = input.accent
  ctx.fillText(`hivegpt.cn/games/${input.game}`, L, qy + 182)

  // A thin light keyline so the card has an edge on a white feed.
  ctx.strokeStyle = 'rgba(255,255,255,0.22)'
  ctx.lineWidth = 4
  ctx.strokeRect(2, 2, W - 4, H - 4)

  return c.toDataURL('image/png')
}
