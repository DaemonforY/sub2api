// What a page's share card shows, worked out at build time from its Markdown (transformPageData in
// config.mts) so the browser only reads page.shareCard. Order for the key points: a hand-written
// `share.points` in the frontmatter, then the lesson's `goals`, then the page's ## headings.
// Pure (no fs) so the frontend's vitest can cover it.

export interface PageShareCard {
  title: string
  label: string
  summary: string
  points: string[]
  minutes: number
  source: string
}

type Frontmatter = Record<string, unknown>

const SKIP_HEADINGS = /^(小结|总结|准备|前置|目录|常见问题|高频问题|常见报错|参考资料?|相关专题|相关阅读|延伸阅读|下一步|建议阅读顺序|适合谁看|练习|课后练习|动手练习|FAQ)$/i

/** Plain text of a Markdown fragment: links, emphasis, code marks, HTML and emoji shortcodes removed. */
export function plainText(md: string): string {
  return md
    .replace(/<[^>]+>/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/\{#[^}]*\}/g, '')
    .replace(/[*`_~]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

/** The Markdown body without frontmatter, fenced code, ::: container lines and HTML blocks. */
function body(md: string): { prose: string; code: string } {
  const noFm = md.replace(/^---\n[\s\S]*?\n---\n/, '')
  const code: string[] = []
  const prose = noFm
    .replace(/^(```|~~~)[^\n]*\n[\s\S]*?^\1\s*$/gm, (m) => {
      code.push(m)
      return ''
    })
    .split('\n')
    .filter((l) => !/^\s*:::/.test(l) && !/^\s*<\/?[A-Za-z][^>]*>\s*$/.test(l))
    .join('\n')
  return { prose, code: code.join('\n') }
}

function strings(value: unknown): string[] {
  return Array.isArray(value) ? value.map((v) => plainText(String(v ?? ''))).filter(Boolean) : []
}

/** `## ` headings worth showing as key points, numbering like 「1.」「一、」「第 2 步：」 removed. */
export function headingPoints(prose: string): string[] {
  const out: string[] = []
  for (const m of prose.matchAll(/^##\s+(.+?)\s*#*\s*$/gm)) {
    const text = plainText(m[1])
      .replace(/^(第\s*[0-9一二三四五六七八九十]+\s*[步章节部分]\s*[：:、.]?|[0-9]+\s*[.、)）]|[一二三四五六七八九十]+\s*[、.])\s*/, '')
      .trim()
    if (text && !SKIP_HEADINGS.test(text)) out.push(text)
  }
  return out
}

function firstParagraph(prose: string): string {
  for (const para of prose.split(/\n\s*\n/)) {
    const t = para.trim()
    if (!t || /^(#|\||>|-|\*|\d+\.|!\[)/.test(t)) continue
    const text = plainText(t)
    if (text.length >= 20) return text.length > 120 ? `${text.slice(0, 118)}…` : text
  }
  return ''
}

/**
 * The share card data for a page, or null for pages without one (home / index layouts, or
 * `share: false`). `label` is the section name the caller resolved; `minutes` overrides the
 * estimate (lessons have their own).
 */
export function pageShareCard(md: string, fm: Frontmatter, title: string, label: string, minutes = 0): PageShareCard | null {
  if (fm.share === false || fm.layout === 'home' || fm.layout === 'page') return null
  const share = (fm.share && typeof fm.share === 'object' ? fm.share : {}) as Frontmatter
  const { prose, code } = body(md)

  let points = strings(share.points)
  if (points.length < 2) points = strings(fm.goals)
  if (points.length < 2) points = headingPoints(prose)

  const summary = plainText(String(share.summary || fm.description || '')) || firstParagraph(prose)
  const chars = plainText(prose).replace(/\s/g, '').length + Math.round(code.length / 4)
  const reprint = /本文来自\s*\[JavaGuide\]/.test(md) || /javaguide\.cn/.test(JSON.stringify(fm.head || ''))

  return {
    title: plainText(String(share.title || title || fm.title || '')),
    label,
    summary: summary.length > 120 ? `${summary.slice(0, 118)}…` : summary,
    points: points.slice(0, 4),
    minutes: minutes || Math.max(1, Math.round(chars / 400)),
    source: reprint ? '来源 JavaGuide · Apache-2.0 许可' : '',
  }
}
