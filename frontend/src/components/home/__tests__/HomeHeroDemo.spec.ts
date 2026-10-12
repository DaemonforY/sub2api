import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import HomeHeroDemo from '../HomeHeroDemo.vue'

vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

describe('home hero demo', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })
  afterEach(() => vi.useRealTimers())

  it('types the drawing example and moves on to writing', async () => {
    const w = mount(HomeHeroDemo)
    const input = w.get<HTMLTextAreaElement>('[data-testid="home-hero-prompt"]')
    vi.advanceTimersByTime(300 + 34 * 3)
    await nextTick()
    expect(input.element.value).toBe('home.v3.demo.draw.prompt'.slice(0, 4))
    vi.advanceTimersByTime(6500) // typed, shown, held: on to 写作
    await nextTick()
    expect(w.find('[data-testid="home-hero-generate"]').exists()).toBe(false) // 写作 / 编程 link out instead
    expect(w.findAll('a').some((x) => /^\/editor\//.test(x.attributes('href') || ''))).toBe(true)
  })

  it('makes whatever the visitor writes a drawing prompt and emits it', async () => {
    const w = mount(HomeHeroDemo)
    vi.advanceTimersByTime(20_000) // somewhere past drawing
    const input = w.get('[data-testid="home-hero-prompt"]')
    await input.trigger('focus')
    expect(w.find('[data-testid="home-hero-generate"]').exists()).toBe(true)
    await input.setValue('赛博朋克风的西湖夜景')
    await w.get('[data-testid="home-hero-generate"]').trigger('click')
    expect(w.emitted('generate')).toEqual([['赛博朋克风的西湖夜景']])
    vi.advanceTimersByTime(30_000)
    expect((input.element as HTMLTextAreaElement).value).toBe('赛博朋克风的西湖夜景') // the show stays stopped
  })

  it('falls back to the example when the box is empty', async () => {
    const w = mount(HomeHeroDemo)
    const input = w.get('[data-testid="home-hero-prompt"]')
    await input.trigger('focus')
    await input.setValue('  ')
    await w.get('[data-testid="home-hero-generate"]').trigger('click')
    expect(w.emitted('generate')).toEqual([['home.v3.demo.draw.prompt']])
  })
})
