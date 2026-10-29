import { describe, expect, it } from 'vitest'
import type { Contest } from '@/api/contests'
import { contestCover, contestMilestones, countdownParts, MONTHLY_CONTEST_COVER } from '../contest'

const base = {
  submission_start_at: '2026-11-01T00:00:00+08:00',
  voting_start_at: '2026-11-01T00:00:00+08:00',
  submission_end_at: '2026-11-22T23:59:59+08:00',
  voting_end_at: '2026-11-30T20:00:00+08:00'
} as Contest

describe('contest page helpers', () => {
  it('merges the opening when entries and votes open together and marks what has passed', () => {
    const now = new Date('2026-11-23T10:00:00+08:00').getTime()
    const m = contestMilestones(base, now)
    expect(m.map((x) => x.key)).toEqual(['open', 'submissionClose', 'votingClose'])
    expect(m.map((x) => x.done)).toEqual([true, true, false])
  })

  it('lists separate openings in date order', () => {
    const c = { ...base, voting_start_at: '2026-11-23T00:00:00+08:00' } as Contest
    expect(contestMilestones(c, 0).map((x) => x.key)).toEqual(['submissionOpen', 'submissionClose', 'votingOpen', 'votingClose'])
  })

  it('splits a countdown and stops at zero', () => {
    expect(countdownParts(((2 * 24 + 8) * 60 + 59) * 60000 + 30000)).toEqual({ days: 2, hours: 8, minutes: 59 })
    expect(countdownParts(-5000)).toEqual({ days: 0, hours: 0, minutes: 0 })
  })

  it('gives scheduled contests the monthly cover unless they set their own', () => {
    expect(contestCover({ cover_image: '', auto_period: '2026-11' })).toBe(MONTHLY_CONTEST_COVER)
    expect(contestCover({ cover_image: 'https://x.test/a.png', auto_period: '2026-11' })).toBe('https://x.test/a.png')
    expect(contestCover({ cover_image: '' })).toBe('')
  })
})
