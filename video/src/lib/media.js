// Uploaded works (mode "upload"): the user's own MP4 / WebM video or animated SVG.

export const UPLOAD_LIMITS = {
  videoBytes: 100 * 1024 * 1024,
  svgBytes: 2 * 1024 * 1024,
  posterBytes: 2 * 1024 * 1024,
  seconds: 180,
  perDay: 20
}

/** What a work is called on cards and pages. */
export function modeLabel(work) {
  if (work?.mode === 'film') return 'HTML 视频'
  if (work?.mode === 'upload') return work.media?.kind === 'svg' ? 'SVG 动画' : '上传视频'
  return '动画'
}

/** Which kind of upload a picked file is, from its name and type (the server checks the content). */
export function uploadKind(file) {
  const name = (file?.name || '').toLowerCase()
  if (name.endsWith('.svg') || file?.type === 'image/svg+xml') return 'svg'
  if (/\.(mp4|m4v|webm)$/.test(name) || file?.type === 'video/mp4' || file?.type === 'video/webm') return 'video'
  if (/\.(mov|qt)$/.test(name) || file?.type === 'video/quicktime') return 'mov'
  return ''
}

export const fmtBytes = (n) => (n >= 1024 * 1024 ? `${(n / 1024 / 1024).toFixed(1)} MB` : `${Math.max(1, Math.round(n / 1024))} KB`)
export const fmtTime = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`

/** Draws the current frame of a video element as a JPEG (long side at most 1280 px). */
export function captureFrame(video, quality = 0.85) {
  const w = video.videoWidth
  const h = video.videoHeight
  if (!w || !h) return Promise.resolve(null)
  const k = Math.min(1, 1280 / Math.max(w, h))
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(w * k)
  canvas.height = Math.round(h * k)
  canvas.getContext('2d').drawImage(video, 0, 0, canvas.width, canvas.height)
  return new Promise((resolve) => canvas.toBlob((b) => resolve(b), 'image/jpeg', quality))
}
