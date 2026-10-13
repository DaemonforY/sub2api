// Finds a piece of text in a ProseMirror document (within one paragraph or heading), so check
// results and comments that quote the text can be located and replaced in the editor.
import type { Node as PMNode } from '@tiptap/pm/model'

export interface TextRange {
  from: number
  to: number
}

/** Plain text of a quote as it appears in the editor (Markdown marks dropped). */
export function plainQuote(quote: string): string {
  return quote.replace(/\*\*|__|`/g, '').replace(/^#+\s+/, '').trim()
}

export function findText(doc: PMNode, quote: string): TextRange | null {
  const needle = plainQuote(quote)
  if (!needle) return null
  let found: TextRange | null = null
  doc.descendants((node, pos) => {
    if (found) return false
    if (!node.isTextblock) return true
    // Map character offsets of the block's text to document positions.
    const chars: number[] = []
    let text = ''
    node.forEach((child, offset) => {
      if (child.isText && child.text) {
        for (let i = 0; i < child.text.length; i++) chars.push(pos + 1 + offset + i)
        text += child.text
      } else {
        chars.push(pos + 1 + offset)
        text += '￼'
      }
    })
    const at = text.indexOf(needle)
    if (at >= 0) found = { from: chars[at], to: chars[at + needle.length - 1] + 1 }
    return false
  })
  return found
}
