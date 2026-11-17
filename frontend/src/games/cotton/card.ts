// The 战绩图 for 棉花糖大作战: key art on top, the 称号 and score, the kids blown up as big pixel
// sprites, and a QR code back to the game. Returns a PNG data URL.

import QRCode from 'qrcode'
import type { Ui } from './game'
import type { Sheet } from './sprites'

export interface CardInput {
  score: number
  reached: number
  seconds: number
  won: boolean
  honor: string
  mode: string | null
  kids: Ui['kids']
  sprites: Sheet
  keyart: string
  link: string
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}

const FONT = '"PingFang SC","HarmonyOS Sans SC","Microsoft YaHei",system-ui,sans-serif'

function outlined(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, size: number, fill: string, stroke = '#1b1028', weight = 900): void {
  ctx.font = `${weight} ${size}px ${FONT}`
  ctx.lineJoin = 'round'
  ctx.lineWidth = Math.max(4, size / 6)
  ctx.strokeStyle = stroke
  ctx.strokeText(text, x, y)
  ctx.fillStyle = fill
  ctx.fillText(text, x, y)
}

export async function drawResultCard(f: CardInput): Promise<string> {
  const W = 750
  const H = 1260
  const c = document.createElement('canvas')
  c.width = W
  c.height = H
  const ctx = c.getContext('2d')!
  ctx.imageSmoothingEnabled = false
  ctx.fillStyle = '#1b1028'
  ctx.fillRect(0, 0, W, H)
  try {
    const art = await loadImage(f.keyart)
    const h = Math.round((W / art.width) * art.height)
    ctx.drawImage(art, 0, 0, W, h)
    const g = ctx.createLinearGradient(0, h * 0.45, 0, h)
    g.addColorStop(0, 'rgba(27,16,40,0)')
    g.addColorStop(1, 'rgba(27,16,40,1)')
    ctx.fillStyle = g
    ctx.fillRect(0, 0, W, h)
  } catch {
    // no key art: plain background
  }
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  outlined(ctx, '棉花糖大作战', W / 2, 70, 64, '#ffc9e6')
  ctx.textBaseline = 'alphabetic'

  // panel
  const top = 470
  ctx.fillStyle = '#2a1d3a'
  ctx.fillRect(30, top, W - 60, H - top - 30)
  ctx.strokeStyle = '#ff86c3'
  ctx.lineWidth = 6
  ctx.strokeRect(30, top, W - 60, H - top - 30)

  outlined(ctx, f.won ? '暑假保住了！' : `打到了第 ${f.reached} 层`, W / 2, top + 66, 34, '#fff2a0', '#1b1028', 800)
  outlined(ctx, `「${f.honor}」`, W / 2, top + 140, 58, '#ffffff')
  outlined(ctx, f.score.toLocaleString(), W / 2, top + 240, 96, '#ff86c3')
  ctx.font = `700 26px ${FONT}`
  ctx.fillStyle = '#c8b6f5'
  const mins = f.seconds >= 60 ? `${Math.floor(f.seconds / 60)} 分 ${f.seconds % 60} 秒` : `${f.seconds} 秒`
  ctx.fillText(`${f.mode === 'duo' ? '同屏双人' : f.mode === 'host' || f.mode === 'guest' ? '联机双排' : '单人'} · ${mins}`, W / 2, top + 290)

  // the kids, big pixels
  const on = f.kids.map((k, i) => ({ k, i })).filter((x) => x.k.on)
  on.forEach(({ k, i }, n) => {
    const cx = on.length === 1 ? W / 2 : W / 2 + (n === 0 ? -150 : 150)
    const spr = f.sprites.kids[i].f.stand[i === 0 ? 0 : 1]
    const scale = 6
    ctx.drawImage(spr, cx - (spr.width * scale) / 2, top + 320, spr.width * scale, spr.height * scale)
    ctx.font = `800 24px ${FONT}`
    ctx.fillStyle = i === 0 ? '#ff6b7a' : '#6aa8ff'
    ctx.fillText(i === 0 ? '小满' : '阿豆', cx, top + 470)
    ctx.font = `600 20px ${FONT}`
    ctx.fillStyle = '#e8dcff'
    ctx.fillText(`${k.score.toLocaleString()} 分 · 裹掉 ${k.kills} 只 · 最长 ${k.bestChain} 连`, cx, top + 500)
  })

  // QR + call to action
  const qr = document.createElement('canvas')
  await QRCode.toCanvas(qr, f.link, { margin: 1, width: 132, color: { dark: '#1b1028', light: '#ffffff' } })
  const qy = H - 30 - 132 - 24
  ctx.drawImage(qr, 60, qy)
  ctx.textAlign = 'left'
  outlined(ctx, '你能打到第几层？', 216, qy + 46, 34, '#fff2a0', '#1b1028', 900)
  ctx.font = `600 22px ${FONT}`
  ctx.fillStyle = '#c8b6f5'
  ctx.fillText('扫码直接玩 · 能联机双排 · 不用登录', 216, qy + 88)
  ctx.fillText('hivegpt.cn/games/cotton', 216, qy + 120)
  return c.toDataURL('image/png')
}
