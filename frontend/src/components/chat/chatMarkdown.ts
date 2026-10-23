import { Marked, type Tokens } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/common'

/** Model answers to sanitised HTML: GFM, highlighted code blocks with a copy button, links in a new tab. */

function escapeHTML(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

let copyLabel = 'Copy'

const marked = new Marked({ gfm: true, breaks: true })
marked.use({
  renderer: {
    code(token: Tokens.Code): string {
      const lang = (token.lang || '').trim().split(/\s+/)[0].toLowerCase()
      let body: string
      if (lang && hljs.getLanguage(lang)) {
        body = hljs.highlight(token.text, { language: lang, ignoreIllegals: true }).value
      } else {
        body = escapeHTML(token.text)
      }
      const label = escapeHTML(lang || 'text')
      return (
        `<div class="chat-code"><div class="chat-code-bar"><span>${label}</span>` +
        `<button type="button" class="chat-code-copy">${escapeHTML(copyLabel)}</button></div>` +
        `<pre><code class="hljs">${body}</code></pre></div>`
      )
    }
  }
})

export function renderChatMarkdown(md: string, copy = 'Copy'): string {
  if (!md) return ''
  copyLabel = copy
  const html = marked.parse(md, { async: false }) as string
  const clean = DOMPurify.sanitize(html, { ADD_ATTR: ['target'] })
  return clean.replace(/<a /g, '<a target="_blank" rel="noopener noreferrer" ')
}
