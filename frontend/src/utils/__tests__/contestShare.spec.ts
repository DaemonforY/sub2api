import { beforeEach, describe, expect, it } from 'vitest'
import { contestEntryShareUrl, contestListSeed, contestPitch, saveVoteIntent, takeVoteIntent } from '../contestShare'

describe('contestShare', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  it('builds the entry link with the invite code and contest tracking', () => {
    const url = new URL(contestEntryShareUrl('https://hivegpt.cn', 5, 9, { aff: 'AB12', medium: 'poster' }))
    expect(url.pathname).toBe('/contests/5/e/9')
    expect(url.searchParams.get('aff')).toBe('AB12')
    expect(url.searchParams.get('utm_source')).toBe('contest_share')
    expect(url.searchParams.get('utm_medium')).toBe('poster')
    expect(url.searchParams.get('utm_campaign')).toBe('contest-5')
  })

  it('writes pitches for the author and for everyone else, always ending with the link', () => {
    for (let i = 0; i < 3; i++) {
      const own = contestPitch({ title: '猫', author: '小林', mine: true }, '画图赛', 'https://x/e/9', i)
      expect(own).toContain('「猫」')
      expect(own.trim().endsWith('https://x/e/9')).toBe(true)
    }
    expect(contestPitch({ title: '猫', author: '小林', mine: false }, '画图赛', 'https://x/e/9')).toContain('作者是 小林')
  })

  it('keeps a pending vote for its contest only, once, for 30 minutes', () => {
    saveVoteIntent(5, 9, 1000)
    expect(takeVoteIntent(6, 2000)).toBeNull()
    expect(takeVoteIntent(5, 2000)).toMatchObject({ contestId: 5, entryId: 9 })
    expect(takeVoteIntent(5, 3000)).toBeNull()
    saveVoteIntent(5, 9, 0)
    expect(takeVoteIntent(5, 31 * 60 * 1000)).toBeNull()
    expect(localStorage.getItem('contest_vote_intent')).toBeNull()
  })

  it('keeps one shuffle seed per tab', () => {
    const seed = contestListSeed()
    expect(seed).toMatch(/^[a-z0-9]+$/)
    expect(contestListSeed()).toBe(seed)
  })
})
