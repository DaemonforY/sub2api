import { marked } from 'marked'
import DOMPurify from 'dompurify'

/** Markdown written by admins (course pages) to sanitised HTML; links open in a new tab. */
export function renderMarkdown(md: string | undefined): string {
  if (!md) return ''
  const html = marked.parse(md, { gfm: true, breaks: true, async: false }) as string
  const clean = DOMPurify.sanitize(html)
  return clean.replace(/<a /g, '<a target="_blank" rel="noopener noreferrer" ')
}
