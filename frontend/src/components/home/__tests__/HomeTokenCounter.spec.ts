import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import HomeTokenCounter from '../HomeTokenCounter.vue'

const { getUsageTotals } = vi.hoisted(() => ({ getUsageTotals: vi.fn() }))
vi.mock('@/api/pricing', () => ({ getUsageTotals }))

const value = (w: ReturnType<typeof mount>) => w.get('[data-testid="home-token-counter-value"]').text()

describe('TOKENS PROCESSED', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval', 'requestAnimationFrame', 'cancelAnimationFrame', 'performance'] })
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    getUsageTotals.mockReset()
  })
  afterEach(() => vi.useRealTimers())

  it('counts up to the real total, then pops each real increase as +N', async () => {
    getUsageTotals.mockResolvedValueOnce({ total_tokens: 1_000_000, total_requests: 10 }).mockResolvedValue({ total_tokens: 1_050_000, total_requests: 11 })
    const w = mount(HomeTokenCounter)
    await flushPromises()
    vi.advanceTimersByTime(3000)
    await flushPromises()
    expect(value(w)).toBe('1,000,000')

    vi.advanceTimersByTime(6000) // next poll: 50,000 more were used
    await flushPromises()
    const seen = new Set<string>()
    for (let i = 0; i < 12; i++) { // the increase plays out over a few seconds; each pop floats off after 1.5 s
      vi.advanceTimersByTime(400)
      await flushPromises()
      w.findAll('[data-testid="home-token-counter-pop"]').forEach((p) => seen.add(p.text()))
    }
    expect(value(w)).toBe('1,050,000')
    expect(seen.size).toBeGreaterThan(1)
    expect([...seen].every((t) => /^\+[\d,]+$/.test(t))).toBe(true)
    expect([...seen].reduce((sum, t) => sum + Number(t.replace(/[+,]/g, '')), 0)).toBe(50_000)
  })

  it('never invents growth: an unchanged total pops nothing', async () => {
    getUsageTotals.mockResolvedValue({ total_tokens: 500, total_requests: 1 })
    const w = mount(HomeTokenCounter)
    await flushPromises()
    vi.advanceTimersByTime(20000)
    await flushPromises()
    expect(value(w)).toBe('500')
    expect(w.findAll('[data-testid="home-token-counter-pop"]')).toHaveLength(0)
  })

  it('shows a dash until the number arrives', () => {
    getUsageTotals.mockReturnValue(new Promise(() => {}))
    expect(value(mount(HomeTokenCounter))).toBe('—')
  })
})
