import { describe, expect, it } from 'vitest'
import { renderChatMarkdown } from '../chatMarkdown'

describe('renderChatMarkdown', () => {
  it('highlights fenced code and adds a copy button', () => {
    const html = renderChatMarkdown('```python\nprint("hi")\n```', '复制')
    expect(html).toContain('class="chat-code"')
    expect(html).toContain('<span>python</span>')
    expect(html).toContain('>复制</button>')
    expect(html).toContain('hljs-built_in')
  })

  it('escapes code in unknown languages', () => {
    const html = renderChatMarkdown('```foo\n<b>x</b>\n```')
    expect(html).toContain('&lt;b&gt;x&lt;/b&gt;')
  })

  it('strips scripts and event handlers, opens links in a new tab', () => {
    const html = renderChatMarkdown('<img src=x onerror="alert(1)"><script>alert(2)</script>[site](https://hivegpt.cn)')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('<script')
    expect(html).toContain('target="_blank"')
    expect(html).toContain('rel="noopener noreferrer"')
  })

  it('renders tables and line breaks', () => {
    const html = renderChatMarkdown('| a | b |\n| - | - |\n| 1 | 2 |\n\nline1\nline2')
    expect(html).toContain('<table>')
    expect(html).toContain('line1<br>')
  })
})
