import { describe, expect, it } from 'vitest'
import { analyzeWriting, findCatchphrases } from '../writingStats'

const sample = `大家好，我是J哥。说实话，这周我又熬夜了。

说实话，写作这件事没有捷径。你也许会问：那怎么办？接下来，J哥带你一步步拆解。

值得注意的是，AI 能写，但写不出你。说实话，真诚才是护城河！你觉得呢？`

describe('analyzeWriting', () => {
  it('counts sentences, voice and AI words', () => {
    const s = analyzeWriting([sample])
    expect(s.articles).toBe(1)
    expect(s.sentences).toBeGreaterThan(5)
    expect(s.avgSentence).toBeGreaterThan(3)
    expect(s.questions).toBeGreaterThan(0)
    expect(s.iRate).toBeGreaterThan(0)
    expect(s.youRate).toBeGreaterThan(0)
    expect(s.aiWords).toEqual([{ text: '值得注意的是', count: 1 }])
    expect(s.rhythm).toBe('short')
    expect(s.catchphrases.map((c) => c.text)).toContain('说实话')
  })

  it('handles empty input', () => {
    const s = analyzeWriting(['', '  '])
    expect(s).toMatchObject({ articles: 0, chars: 0, avgSentence: 0, aiLevel: 'low', catchphrases: [] })
  })
})

describe('findCatchphrases', () => {
  it('keeps the full phrase, not its prefixes, and skips common words', () => {
    const texts = ['说实话，我们走吧。说实话，我们再看。', '说实话，这样不行。我们一起。']
    const got = findCatchphrases(texts).map((c) => c.text)
    expect(got[0]).toBe('说实话')
    expect(got).not.toContain('说实')
    expect(got).not.toContain('我们')
  })
})
