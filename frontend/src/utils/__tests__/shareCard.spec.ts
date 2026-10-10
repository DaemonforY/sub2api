import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { blocksFromRange, clampBlocks, dataUrlToBlob, clampLines, headingBefore, readingMinutes, shareUrl, wrapText } from '../shareCard'
import { headingPoints, pageShareCard } from '../../../../learn/.vitepress/share-data'

// One "unit" per character: CJK and Latin alike, so widths are easy to reason about.
const measure = (s: string) => s.length

describe('shareCard copies', () => {
  it('learn keeps an identical copy of the module', () => {
    const main = readFileSync(resolve(__dirname, '../shareCard.ts'), 'utf8')
    const learn = readFileSync(resolve(__dirname, '../../../../learn/.vitepress/theme/shareCard.ts'), 'utf8')
    expect(learn).toBe(main)
  })
})

describe('wrapText', () => {
  it('breaks between CJK characters', () => {
    expect(wrapText('模型决定下一步调用哪个工具', 5, measure)).toEqual(['模型决定下', '一步调用哪', '个工具'])
  })

  it('keeps Latin words whole and drops the space at the break', () => {
    expect(wrapText('call the tool again', 9, measure)).toEqual(['call the', 'tool', 'again'])
  })

  it('never starts a line with closing punctuation', () => {
    const lines = wrapText('一二三四五，六七', 5, measure)
    expect(lines[0]).toBe('一二三四五，')
    expect(lines[1]).toBe('六七')
  })

  it('never ends a line with opening punctuation', () => {
    const lines = wrapText('一二三四「五六」', 5, measure)
    expect(lines[0]).toBe('一二三四')
    expect(lines[1].startsWith('「')).toBe(true)
  })

  it('splits a word longer than the line', () => {
    const lines = wrapText('https://hivegpt.cn/v1/chat/completions', 10, measure)
    expect(lines.every((l) => l.length <= 10)).toBe(true)
    expect(lines.join('')).toBe('https://hivegpt.cn/v1/chat/completions')
  })

  it('keeps explicit line breaks', () => {
    expect(wrapText('第一行\n第二行', 10, measure)).toEqual(['第一行', '第二行'])
  })
})

describe('clampLines / clampBlocks', () => {
  it('ends the last kept line with an ellipsis', () => {
    expect(clampLines(['一二三四五', '六七八九十', '十一'], 2, 5, measure)).toEqual(['一二三四五', '六七八九…'])
    expect(clampLines(['一二'], 2, 5, measure)).toEqual(['一二'])
  })

  it('keeps the first N characters across blocks', () => {
    const out = clampBlocks(
      [
        { kind: 'text', text: '一二三四五' },
        { kind: 'text', text: '六七八九十' }
      ],
      7
    )
    expect(out.truncated).toBe(true)
    expect(out.chars).toBe(10)
    expect(out.blocks).toEqual([
      { kind: 'text', text: '一二三四五' },
      { kind: 'text', text: '六七…' }
    ])
  })

  it('leaves short selections alone', () => {
    const blocks = [{ kind: 'code' as const, text: 'print(1)' }]
    expect(clampBlocks(blocks, 300)).toEqual({ blocks, chars: 8, truncated: false })
  })

  it('estimates reading time', () => {
    expect(readingMinutes(0)).toBe(1)
    expect(readingMinutes(4000)).toBe(10)
  })
})

describe('dataUrlToBlob', () => {
  it('decodes a data URL without fetch (the CSP blocks data: in connect-src)', async () => {
    const blob = dataUrlToBlob(`data:image/png;base64,${btoa('PNG!')}`)
    expect(blob.type).toBe('image/png')
    expect(blob.size).toBe(4)
  })
})

describe('shareUrl', () => {
  it('replaces old tracking params and adds the invite code', () => {
    const url = shareUrl('https://hivegpt.cn/learn/a/a6.html?aff=OLD&utm_source=x&tab=2#old', { aff: 'ABC', medium: 'quote', anchor: 'agent-循环' })
    const u = new URL(url)
    expect(u.searchParams.get('aff')).toBe('ABC')
    expect(u.searchParams.get('utm_source')).toBe('share_card')
    expect(u.searchParams.get('utm_medium')).toBe('quote')
    expect(u.searchParams.get('tab')).toBe('2')
    expect(decodeURIComponent(u.hash)).toBe('#agent-循环')
  })

  it('leaves out aff when signed out', () => {
    const u = new URL(shareUrl('https://hivegpt.cn/courses/x', { medium: 'summary' }))
    expect(u.searchParams.has('aff')).toBe(false)
    expect(u.hash).toBe('')
  })
})

describe('blocksFromRange', () => {
  function setup(html: string) {
    document.body.innerHTML = `<div id="root">${html}</div><p id="outside">外面的文字</p>`
    return document.getElementById('root')!
  }

  function select(start: Node, startOffset: number, end: Node, endOffset: number) {
    const range = document.createRange()
    range.setStart(start, startOffset)
    range.setEnd(end, endOffset)
    return range
  }

  it('returns paragraphs and code blocks, skipping copy buttons and language labels', () => {
    const root = setup('<h2 id="call">调用</h2><p>接口地址\n统一是：</p><div class="language-text"><button class="copy">复制</button><span class="lang">text</span><pre><code>https://hivegpt.cn/v1\n</code></pre></div><p>结束</p>')
    const ps = root.querySelectorAll('p')
    const range = select(ps[0], 0, ps[1], 1)
    expect(blocksFromRange(range, root)).toEqual([
      { kind: 'text', text: '接口地址统一是：' },
      { kind: 'code', text: 'https://hivegpt.cn/v1' },
      { kind: 'text', text: '结束' }
    ])
    expect(headingBefore(range, root)).toBe('call')
  })

  it('keeps a selection inside one code block as code', () => {
    const root = setup('<pre><code>line one\nline two</code></pre>')
    const text = root.querySelector('code')!.firstChild!
    expect(blocksFromRange(select(text, 0, text, 17), root)).toEqual([{ kind: 'code', text: 'line one\nline two' }])
  })

  it('ignores selections outside the root or in skipped widgets', () => {
    const root = setup('<p>正文内容</p><div class="runbox"><p>运行框里的文字</p></div>')
    const outside = document.getElementById('outside')!.firstChild!
    expect(blocksFromRange(select(outside, 0, outside, 3), root)).toEqual([])
    const inBox = root.querySelector('.runbox p')!.firstChild!
    expect(blocksFromRange(select(inBox, 0, inBox, 4), root, '.runbox')).toEqual([])
  })

  it('drops skipped widgets in the middle of a selection', () => {
    const root = setup('<p>前面一段</p><div class="runbox"><p>运行框</p></div><p>后面一段</p>')
    const ps = root.querySelectorAll(':scope > p')
    const blocks = blocksFromRange(select(ps[0], 0, ps[1], 1), root, '.runbox')
    expect(blocks.map((b) => b.text)).toEqual(['前面一段', '后面一段'])
  })
})

describe('learn pageShareCard', () => {
  const md = `---
title: 测试
description: 一句话简介
---

# 测试

正文第一段，介绍这篇文章讲什么内容，足够长足够长。

## 1. 第一部分

\`\`\`python
print("hi")
\`\`\`

## 第 2 步：第二部分

## 小结

## 常见问题
`

  it('uses headings as points, without numbering or wrap-up sections', () => {
    expect(headingPoints(md)).toEqual(['第一部分', '第二部分'])
    const card = pageShareCard(md, { title: '测试', description: '一句话简介' }, '测试', '延伸阅读')
    expect(card).toMatchObject({ title: '测试', label: '延伸阅读', summary: '一句话简介', points: ['第一部分', '第二部分'], source: '' })
    expect(card!.minutes).toBeGreaterThanOrEqual(1)
  })

  it('prefers share.points, then lesson goals', () => {
    expect(pageShareCard(md, { share: { points: ['甲', '乙'] }, goals: ['丙', '丁'] }, 'T', '')!.points).toEqual(['甲', '乙'])
    expect(pageShareCard(md, { goals: ['丙', '丁'] }, 'T', '', 15)).toMatchObject({ points: ['丙', '丁'], minutes: 15 })
  })

  it('credits reprinted JavaGuide articles and skips home pages', () => {
    const reprint = `${md}\n::: info\n本文来自 [JavaGuide](https://javaguide.cn)\n:::\n`
    expect(pageShareCard(reprint, {}, 'T', '')!.source).toContain('JavaGuide')
    expect(pageShareCard(md, { layout: 'home' }, 'T', '')).toBeNull()
    expect(pageShareCard(md, { share: false }, 'T', '')).toBeNull()
  })
})
