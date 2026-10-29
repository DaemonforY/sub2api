// Subtitles from narration + TTS word timings: the narration is cut into short clauses at
// punctuation, and each clause is shown from the moment its first word is spoken.

// Subtitle style choices (spec.subtitles: {position, size, off}; empty = bottom, medium, on).
// public/render.js and lib/exportHtml.js place them the same way.
export const SUBTITLE_POSITIONS = [
  { id: 'bottom', name: '底部' },
  { id: 'raised', name: '中下' },
  { id: 'top', name: '顶部' }
]
export const SUBTITLE_SIZES = [
  { id: 's', name: '小', scale: 0.8 },
  { id: 'm', name: '中', scale: 1 },
  { id: 'l', name: '大', scale: 1.3 }
]
/** {position, size, off} with defaults filled in. */
export function subtitleStyle(sub) {
  const position = SUBTITLE_POSITIONS.some((x) => x.id === sub?.position) ? sub.position : 'bottom'
  const size = SUBTITLE_SIZES.find((x) => x.id === sub?.size) || SUBTITLE_SIZES[1]
  return { position, size: size.id, scale: size.scale, off: !!sub?.off }
}

const BREAK = /[，。！？；：、,.!?;:\n]/
const MAX = 22

const CJK = /[\u3400-\u9fff\uf900-\ufaff]/

function clauses(text) {
  const out = []
  let cur = ''
  const chars = Array.from(text)
  chars.forEach((ch, i) => {
    cur += ch
    // A long clause is cut once it reaches MAX characters, but only before a Chinese character, so
    // English words, numbers and bracketed terms stay whole (at a space if one runs very long).
    const next = chars[i + 1] || ''
    const canCut = !next || CJK.test(next) || (cur.length >= MAX * 2 && next === ' ')
    if (BREAK.test(ch) || (cur.length >= MAX && canCut)) {
      if (cur.trim()) out.push(cur)
      cur = ''
    }
  })
  if (cur.trim()) out.push(cur)
  return out
}

const tidy = (s) => s.replace(/^[\s，。！？；：、,.!?;:]+|[\s，。、,;；:：]+$/g, '').trim()

/** [{start, end, text}] in film time. */
export function buildSubtitles(scenes, starts) {
  const cues = []
  scenes.forEach((sc, i) => {
    const narration = (sc.narration || '').trim()
    if (!narration) return
    const begin = starts[i] || 0
    const words = sc.audio?.words || []
    const parts = clauses(narration)
    // Character offset → spoken time, by walking the words through the narration.
    const charTime = []
    let pos = 0
    for (const w of words) {
      const at = narration.indexOf(w.text, pos)
      if (at < 0) continue
      charTime.push([at, w.start])
      pos = at + w.text.length
    }
    const timeAt = (offset) => {
      let best = null
      for (const [at, time] of charTime) {
        if (at <= offset) best = time
        else break
      }
      if (best != null) return best
      // No timings (old audio): spread the clauses evenly over the scene.
      return (offset / narration.length) * (sc.audio?.duration || sc.duration || 0)
    }
    let offset = 0
    const local = parts.map((p) => {
      const start = timeAt(offset)
      offset += p.length
      return { start, text: tidy(p) }
    })
    local.forEach((c, j) => {
      if (!c.text) return
      const end = j + 1 < local.length ? local[j + 1].start : sc.audio?.duration || sc.duration
      cues.push({ start: begin + c.start, end: begin + Math.max(c.start + 0.3, end), text: c.text })
    })
  })
  return cues
}
