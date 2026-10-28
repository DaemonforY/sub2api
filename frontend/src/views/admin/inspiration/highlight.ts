/** Splits a text into plain and matched parts, so search words can be marked without v-html. */
export interface HighlightPart {
  text: string
  hit: boolean
}

export function highlightParts(text: string, words: string[]): HighlightPart[] {
  const list = words.map((w) => w.trim()).filter(Boolean)
  if (!list.length || !text) return [{ text, hit: false }]
  const escaped = list.sort((a, b) => b.length - a.length).map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
  const re = new RegExp(`(${escaped.join('|')})`, 'gi')
  return text
    .split(re)
    .filter((p) => p !== '')
    .map((p) => ({ text: p, hit: list.some((w) => w.toLowerCase() === p.toLowerCase()) }))
}

/** The query split into words, the way the server splits it. */
export function queryWords(q: string): string[] {
  return q.split(/[\s,，、\u3000]+/).map((w) => w.trim()).filter(Boolean).slice(0, 8)
}
