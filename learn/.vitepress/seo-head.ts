import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { HeadConfig, PageData } from 'vitepress'
import { tracks } from './theme/tracks'

// Per-page <head> for search engines and AI crawlers: canonical URL (the .html form, as in the
// sitemap), Open Graph tags, and JSON-LD — TechArticle and BreadcrumbList on every page, FAQPage
// from the page's 「常见问题」 section. JSON-LD is a data block, not a script, so the main site's CSP
// (script-src 'self' + nonce) does not apply to it.

const ORIGIN = 'https://hivegpt.cn'
const BASE = `${ORIGIN}/learn/`
const SITE_NAME = 'HiveGPT AI 学习'

export const SECTIONS: Record<string, string> = {
  connect: '接入教程',
  errors: '报错排查',
  prompts: '生图提示词',
  codex: 'Codex 教程',
  scenes: '场景玩法',
  guide: '延伸阅读',
  bigdata: '大数据',
  ...Object.fromEntries(tracks.map((t) => [t.id, `${t.letter} · ${t.title}`])),
}

export function pageURL(relativePath: string): string {
  const path = relativePath.replace(/(^|\/)index\.md$/, '$1').replace(/\.md$/, '.html')
  return BASE + path
}

/** Plain text of a Markdown fragment: links, emphasis, code marks and HTML removed. */
function plain(md: string): string {
  return md
    .replace(/<[^>]+>/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[*`]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

/** Q&A pairs from the 「## 常见问题」 section: `**问题？** 回答` paragraphs, or a 现象 / 原因 / 处理 table. */
export function faqFromMarkdown(md: string): { q: string; a: string }[] {
  const start = md.search(/^## 常见问题\s*$/m)
  if (start < 0) return []
  const rest = md.slice(start).replace(/^## 常见问题\s*$/m, '')
  const end = rest.search(/^## /m)
  const section = end < 0 ? rest : rest.slice(0, end)
  const out: { q: string; a: string }[] = []

  for (const para of section.split(/\n\s*\n/)) {
    const m = para.trim().match(/^\*\*([^*]+?[？?])\*\*\s*([\s\S]+)$/)
    if (m) out.push({ q: plain(m[1]), a: plain(m[2]) })
  }

  const rows = section.split('\n').filter((l) => l.trim().startsWith('|'))
  if (rows.length > 2) {
    const head = rows[0].split('|').map((c) => c.trim()).filter(Boolean)
    for (const row of rows.slice(2)) {
      const cells = row.split('|').slice(1, -1).map((c) => plain(c))
      if (cells.length < 2 || !cells[0]) continue
      const answer = cells
        .slice(1)
        .map((c, i) => (head[i + 1] ? `${head[i + 1]}：${c}` : c))
        .join('；')
      out.push({ q: cells[0], a: answer })
    }
  }
  return out.filter((p) => p.q && p.a)
}

function ld(data: object): HeadConfig {
  return ['script', { type: 'application/ld+json' }, JSON.stringify(data)]
}

export function seoHead(pageData: PageData, srcDir: string): HeadConfig[] {
  if (pageData.isNotFound || pageData.relativePath === '404.md') return []
  const url = pageURL(pageData.relativePath)
  const title = pageData.title || SITE_NAME
  const description = pageData.description || ''
  const head: HeadConfig[] = [
    ['link', { rel: 'canonical', href: url }],
    ['meta', { property: 'og:type', content: 'article' }],
    ['meta', { property: 'og:site_name', content: SITE_NAME }],
    ['meta', { property: 'og:title', content: title }],
    ['meta', { property: 'og:url', content: url }],
    ['meta', { property: 'og:locale', content: 'zh_CN' }],
    ['meta', { name: 'twitter:card', content: 'summary' }],
  ]
  if (description) head.push(['meta', { property: 'og:description', content: description }])

  const publisher = { '@type': 'Organization', name: 'HiveGPT', url: `${ORIGIN}/` }
  head.push(
    ld({
      '@context': 'https://schema.org',
      '@type': 'TechArticle',
      headline: title,
      description,
      url,
      inLanguage: 'zh-CN',
      author: publisher,
      publisher,
      isPartOf: { '@type': 'WebSite', name: SITE_NAME, url: BASE },
    }),
  )

  const crumbs = [
    { name: 'HiveGPT', item: `${ORIGIN}/` },
    { name: SITE_NAME, item: BASE },
  ]
  const segments = pageData.relativePath.split('/').slice(0, -1)
  segments.forEach((seg, i) => {
    const name = SECTIONS[seg]
    if (name) crumbs.push({ name, item: `${BASE}${segments.slice(0, i + 1).join('/')}/` })
  })
  if (!/(^|\/)index\.md$/.test(pageData.relativePath)) crumbs.push({ name: title, item: url })
  head.push(
    ld({
      '@context': 'https://schema.org',
      '@type': 'BreadcrumbList',
      itemListElement: crumbs.map((c, i) => ({ '@type': 'ListItem', position: i + 1, name: c.name, item: c.item })),
    }),
  )

  let md = ''
  try {
    md = readFileSync(join(srcDir, pageData.relativePath), 'utf8')
  } catch {
    // generated or missing source: no FAQ
  }
  const faq = faqFromMarkdown(md)
  if (faq.length >= 2) {
    head.push(
      ld({
        '@context': 'https://schema.org',
        '@type': 'FAQPage',
        mainEntity: faq.map((p) => ({ '@type': 'Question', name: p.q, acceptedAnswer: { '@type': 'Answer', text: p.a } })),
      }),
    )
  }
  return head
}
