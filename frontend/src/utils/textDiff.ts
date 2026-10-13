// Word-level differences between two texts, for showing an AI rewrite against the original.
// Chinese has no spaces, so "words" are runs of Han characters split into single characters by
// diffChars when the texts are mostly Chinese; Latin text diffs by word.
import { diffChars, diffWordsWithSpace } from 'diff'

export interface DiffPart {
  text: string
  kind: 'same' | 'add' | 'del'
}

const HAN = /[㐀-鿿]/g

export function textDiff(before: string, after: string): DiffPart[] {
  const han = (before.match(HAN)?.length || 0) + (after.match(HAN)?.length || 0)
  const mostlyHan = han > (before.length + after.length) * 0.3
  const parts = mostlyHan ? diffChars(before, after) : diffWordsWithSpace(before, after)
  const out: DiffPart[] = []
  for (const p of parts) {
    const kind: DiffPart['kind'] = p.added ? 'add' : p.removed ? 'del' : 'same'
    const last = out[out.length - 1]
    if (last && last.kind === kind) last.text += p.value
    else out.push({ text: p.value, kind })
  }
  return out
}

/** How much of the text changed (0–1), counting added and removed characters. */
export function changeRatio(parts: DiffPart[]): number {
  let changed = 0
  let total = 0
  for (const p of parts) {
    total += p.text.length
    if (p.kind !== 'same') changed += p.text.length
  }
  return total ? changed / total : 0
}
