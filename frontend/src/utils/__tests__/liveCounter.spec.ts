import { describe, expect, it } from 'vitest'
import { formatCount, splitIncrease } from '../liveCounter'

describe('live token counter', () => {
  it('plays a real increase out in a few steps that add up to it exactly', () => {
    let seed = 1
    const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647)
    for (const delta of [1, 7, 999, 1000, 54321, 160000, 3_000_000]) {
      const steps = splitIncrease(delta, rnd)
      expect(steps.reduce((a, b) => a + b, 0)).toBe(delta)
      expect(steps.every((n) => n >= 1)).toBe(true)
      expect(steps.length).toBeLessThanOrEqual(4)
    }
    expect(splitIncrease(160000, rnd)).toHaveLength(4)
    expect(splitIncrease(1, rnd)).toEqual([1])
  })

  it('shows nothing for no increase', () => {
    expect(splitIncrease(0)).toEqual([])
    expect(splitIncrease(-5)).toEqual([])
  })

  it('groups thousands', () => {
    expect(formatCount(1912852691)).toBe('1,912,852,691')
  })
})
