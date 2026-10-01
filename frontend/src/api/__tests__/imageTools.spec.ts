import { describe, expect, it } from 'vitest'
import { imageToolStat, sumImageToolStats } from '../imageTools'

describe('image tool stats', () => {
  const stats = [
    { tool: 'remove_bg' as const, runs: 5, free_runs: 3, cost: 0.04, users: 2 },
    { tool: 'upscale' as const, runs: 2, free_runs: 0, cost: 0.1, users: 1 }
  ]

  it('sums runs, free runs and charges over tools', () => {
    const sum = sumImageToolStats(stats)
    expect(sum.runs).toBe(7)
    expect(sum.free_runs).toBe(3)
    expect(sum.cost).toBeCloseTo(0.14)
    expect(sumImageToolStats(undefined)).toEqual({ runs: 0, free_runs: 0, cost: 0 })
  })

  it('reads one tool, zero when it has no runs', () => {
    expect(imageToolStat(stats, 'upscale').runs).toBe(2)
    expect(imageToolStat([stats[0]], 'upscale')).toEqual({ tool: 'upscale', runs: 0, free_runs: 0, cost: 0, users: 0 })
  })
})
