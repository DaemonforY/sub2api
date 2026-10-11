import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import FirstImageView from '../FirstImageView.vue'

const { starter, authStore } = vi.hoisted(() => ({
  starter: vi.fn(),
  authStore: { user: { balance: 2 } as { balance: number } | null, refreshUser: vi.fn() },
}))

vi.mock('@/api/keys', () => ({ keysAPI: { starter } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { site_name: 'HiveGPT' }, siteName: 'HiveGPT' }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}:${JSON.stringify(p)}` : key) }),
}))

const mountView = () => mount(FirstImageView, { global: { stubs: { RouterLink: RouterLinkStub } } })

describe('first image after signing up', () => {
  const assign = vi.fn()
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    starter.mockReset()
    authStore.refreshUser.mockReset().mockResolvedValue({})
    assign.mockReset()
    vi.stubGlobal('location', { ...window.location, origin: 'https://hivegpt.cn', assign })
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('makes sure there is a key, then opens the canvas with the saved prompt', async () => {
    localStorage.setItem('hg_pending_prompt', JSON.stringify({ text: '一只猫', at: Date.now() }))
    starter.mockResolvedValue({ key: { id: 1 }, created: true })
    const w = mountView()
    await flushPromises()
    expect(starter).toHaveBeenCalledOnce()
    expect(w.get('[data-testid="first-image-prompt"]').text()).toContain('一只猫')
    expect(w.text()).toContain('home.v3.firstImage.balance:{"amount":"2"}')
    const href = new URL(w.get('[data-testid="first-image-go"]').attributes('href')!)
    expect(href.pathname).toBe('/image')
    expect(href.searchParams.get('prompt')).toBe('一只猫')
    expect(localStorage.getItem('hg_pending_prompt')).toBeNull()

    vi.advanceTimersByTime(4000)
    expect(assign).toHaveBeenCalledWith(href.toString())
  })

  it('stays put when the key cannot be made, leaving the canvas one click away', async () => {
    starter.mockRejectedValue(new Error('boom'))
    const w = mountView()
    await flushPromises()
    vi.advanceTimersByTime(10_000)
    expect(assign).not.toHaveBeenCalled()
    expect(w.text()).toContain('home.v3.firstImage.keyFailed')
    expect(w.find('[data-testid="first-image-prompt"]').exists()).toBe(false)
  })
})
