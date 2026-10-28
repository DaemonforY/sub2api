import { describe, expect, it } from 'vitest'
import { highlightParts, queryWords } from '../highlight'

describe('highlightParts', () => {
  it('marks every word, longest first, without losing text', () => {
    const parts = highlightParts('忙趁东风放纸鸢。', ['纸鸢', '东风', '东'])
    expect(parts.map((p) => p.text).join('')).toBe('忙趁东风放纸鸢。')
    expect(parts.filter((p) => p.hit).map((p) => p.text)).toEqual(['东风', '纸鸢'])
  })

  it('treats regex characters literally', () => {
    expect(highlightParts('a(b)c', ['(b)'])).toEqual([{ text: 'a', hit: false }, { text: '(b)', hit: true }, { text: 'c', hit: false }])
    expect(highlightParts('abc', [])).toEqual([{ text: 'abc', hit: false }])
  })

  it('splits queries on spaces and Chinese commas', () => {
    expect(queryWords(' 东风，纸鸢 、 儿童  ')).toEqual(['东风', '纸鸢', '儿童'])
  })
})
