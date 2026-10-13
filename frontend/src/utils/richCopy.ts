/**
 * Copies HTML as rich text (with a plain-text fallback), so pasting into the 公众号, 知乎 or 头条
 * editors keeps headings, lists and code blocks. Returns false when the browser refuses.
 * `html` must already be sanitised (renderMarkdown runs DOMPurify).
 */
export async function copyRichText(html: string, text: string): Promise<boolean> {
  if (typeof ClipboardItem !== 'undefined' && navigator.clipboard?.write && window.isSecureContext) {
    try {
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/html': new Blob([html], { type: 'text/html' }),
          'text/plain': new Blob([text], { type: 'text/plain' })
        })
      ])
      return true
    } catch {
      // fall through
    }
  }
  // Older browsers: select a rendered copy and use execCommand.
  const holder = document.createElement('div')
  holder.innerHTML = html
  holder.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0'
  document.body.appendChild(holder)
  const range = document.createRange()
  range.selectNodeContents(holder)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
  try {
    return document.execCommand('copy')
  } finally {
    sel?.removeAllRanges()
    document.body.removeChild(holder)
  }
}
