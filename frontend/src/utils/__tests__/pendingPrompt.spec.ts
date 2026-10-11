import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { savePendingPrompt, takePendingPrompt } from '../pendingPrompt'

describe('pending home-page prompt', () => {
  beforeEach(() => localStorage.clear())
  afterEach(() => vi.useRealTimers())

  it('is handed over once', () => {
    savePendingPrompt('  一只猫  ')
    expect(takePendingPrompt()).toBe('一只猫')
    expect(takePendingPrompt()).toBe('')
  })

  it('expires after a day', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-12T00:00:00Z'))
    savePendingPrompt('一只猫')
    vi.setSystemTime(new Date('2026-10-13T00:00:01Z'))
    expect(takePendingPrompt()).toBe('')
  })

  it('forgets an empty prompt and ignores junk', () => {
    savePendingPrompt('一只猫')
    savePendingPrompt('   ')
    expect(takePendingPrompt()).toBe('')
    localStorage.setItem('hg_pending_prompt', '{not json')
    expect(takePendingPrompt()).toBe('')
  })
})
