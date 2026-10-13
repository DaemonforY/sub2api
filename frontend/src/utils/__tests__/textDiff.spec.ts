import { describe, expect, it } from 'vitest'
import { changeRatio, textDiff } from '@/utils/textDiff'

describe('textDiff', () => {
  it('diffs Chinese by character and merges runs', () => {
    const parts = textDiff('它第一次就跑通了', '它居然一次就跑通了')
    expect(parts.map((p) => p.kind)).toEqual(['same', 'del', 'add', 'same'])
    expect(parts.filter((p) => p.kind === 'add').map((p) => p.text).join('')).toBe('居然')
    expect(changeRatio(parts)).toBeGreaterThan(0)
    expect(changeRatio(textDiff('一样', '一样'))).toBe(0)
  })

  it('diffs English by word', () => {
    const parts = textDiff('the quick fox', 'the slow fox')
    expect(parts.find((p) => p.kind === 'add')?.text).toBe('slow')
  })
})
