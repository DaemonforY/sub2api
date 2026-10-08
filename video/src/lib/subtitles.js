// Subtitles from narration + TTS word timings: the narration is cut into short clauses at
// punctuation, and each clause is shown from the moment its first word is spoken.

const BREAK = /[，。！？；：、,.!?;:\n]/
const MAX = 22

function clauses(text) {
  const out = []
  let cur = ''
  for (const ch of Array.from(text)) {
    cur += ch
    if (BREAK.test(ch) || cur.length >= MAX) {
      if (cur.trim()) out.push(cur)
      cur = ''
    }
  }
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
