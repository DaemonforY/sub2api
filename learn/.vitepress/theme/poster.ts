// The certificate's share poster, drawn in the browser: certificate details plus a QR code to the
// public certificate page (which invites the viewer with the holder's invite code).
import QRCode from 'qrcode'
import type { Certificate } from './api'

const FONT = '"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans CJK SC",sans-serif'

export function certPageUrl(code: string): string {
  return `${window.location.origin}/learn/cert.html?c=${encodeURIComponent(code)}`
}

export async function drawPoster(cert: Certificate): Promise<string> {
  const W = 1080
  const H = 1440
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!
  const bg = ctx.createLinearGradient(0, 0, W, H)
  bg.addColorStop(0, '#7c3aed')
  bg.addColorStop(1, '#4c1d95')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, W, H)

  ctx.fillStyle = '#ffffff'
  ctx.beginPath()
  ctx.roundRect(70, 70, W - 140, H - 140, 36)
  ctx.fill()

  ctx.textAlign = 'center'
  ctx.fillStyle = '#7c3aed'
  ctx.font = `600 34px ${FONT}`
  ctx.fillText('HiveGPT AI 学习', W / 2, 190)
  ctx.fillStyle = '#111827'
  ctx.font = `700 76px ${FONT}`
  ctx.fillText('结业证书', W / 2, 300)

  ctx.fillStyle = '#6b7280'
  ctx.font = `400 34px ${FONT}`
  ctx.fillText('兹证明', W / 2, 420)
  ctx.fillStyle = '#111827'
  ctx.font = `700 68px ${FONT}`
  ctx.fillText(cert.display_name, W / 2, 510)
  ctx.fillStyle = '#374151'
  ctx.font = `400 36px ${FONT}`
  ctx.fillText('完成了学习路线', W / 2, 600)
  ctx.fillStyle = '#6d28d9'
  ctx.font = `700 48px ${FONT}`
  ctx.fillText(`「${cert.track_title}」`, W / 2, 680)

  ctx.fillStyle = '#6b7280'
  ctx.font = `400 30px ${FONT}`
  const date = new Date(cert.issued_at).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
  const lines = [cert.quiz_score ? `测验正确率 ${cert.quiz_score}% · ${date}` : date, `证书编号 ${cert.code}`]
  lines.forEach((line, i) => ctx.fillText(line, W / 2, 770 + i * 50))

  const qr = document.createElement('canvas')
  await QRCode.toCanvas(qr, certPageUrl(cert.code), { width: 300, margin: 1, color: { dark: '#111827', light: '#ffffff' } })
  ctx.drawImage(qr, (W - 300) / 2, 920)
  ctx.fillStyle = '#374151'
  ctx.font = `400 30px ${FONT}`
  ctx.fillText('扫码验证证书，和我一起边学边做', W / 2, 1290)
  return canvas.toDataURL('image/png')
}
