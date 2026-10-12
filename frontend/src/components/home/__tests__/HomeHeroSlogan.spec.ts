import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import HomeHeroSlogan from '../HomeHeroSlogan.vue'

const props = { lead: '你当老板，', highlight: 'AI 打工。', huds: [{ idx: 'INPUT', name: '你的一句话', spec: '→ 9 tokens' }] }

describe('home slogan', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'requestAnimationFrame', 'cancelAnimationFrame', 'performance'] })
    Object.defineProperty(document, 'fonts', { configurable: true, value: { ready: Promise.resolve() } })
  })
  afterEach(() => vi.useRealTimers())

  it('decodes into the slogan, then the second line flows', async () => {
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    const w = mount(HomeHeroSlogan, { props })
    expect(w.get('[data-testid="home-hero-slogan"]').attributes('aria-label')).toBe('你当老板，AI 打工。')
    await flushPromises()
    vi.advanceTimersByTime(600)
    await flushPromises()
    expect(w.findAll('.ch.scr').length).toBeGreaterThan(0) // still tokens in the middle of it
    vi.advanceTimersByTime(4000)
    await flushPromises()
    expect(w.find('.ch').exists()).toBe(false)
    expect(w.get('.l1').text()).toBe('你当老板，')
    expect(w.get('.l2').classes()).toContain('flow')
  })

  it('skips the show for reduced motion', async () => {
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: true } as MediaQueryList)
    const w = mount(HomeHeroSlogan, { props })
    await flushPromises()
    expect(w.get('.l2').text()).toBe('AI 打工。')
    expect(w.get('.l2').classes()).toContain('flow')
  })
})
