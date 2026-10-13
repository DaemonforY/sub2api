import { describe, expect, it } from 'vitest'
import { layoutNoteCard, NOTE_H } from '@/utils/noteCards'

// One em per character: wrapping is predictable.
const measure = (s: string, size: number) => [...s].length * size

describe('layoutNoteCard', () => {
  it('gives the cover a big title and the other cards a heading with bullet points', () => {
    const cover = layoutNoteCard({ title: 'Base URL 填错？', points: ['一招搞定'] }, 0, measure)
    const inner = layoutNoteCard({ title: '判断方法', points: ['看最终请求地址', '拼起来等于 /v1/chat/completions'] }, 1, measure)
    expect(cover.cover).toBe(true)
    expect(inner.cover).toBe(false)
    expect(cover.titleSize).toBeGreaterThan(inner.titleSize)
    expect(inner.points).toHaveLength(2)
  })

  it('shrinks the text until a long card fits', () => {
    const points = Array.from({ length: 6 }, (_, i) => `第 ${i + 1} 条：` + '这是一条很长很长的要点内容，需要折成好几行才能放下。'.repeat(2))
    const short = layoutNoteCard({ title: '要点', points: ['短'] }, 1, measure)
    const long = layoutNoteCard({ title: '要点', points }, 1, measure)
    expect(long.pointSize).toBeLessThan(short.pointSize)
    expect(long.height).toBeLessThan(NOTE_H)
    expect(short.height).toBeLessThan(long.height)
  })
})
