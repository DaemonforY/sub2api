import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import HomeView from '../HomeView.vue'

const { appStore, authStore, track, getPublicPricing, getUsageTotals } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
  track: vi.fn(),
  getPublicPricing: vi.fn(),
  getUsageTotals: vi.fn(),
}))

vi.mock('@/stores', () => ({ useAppStore: () => appStore, useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/utils/analytics', () => ({ track: (...a: unknown[]) => track(...a) }))
vi.mock('@/api/courses', () => ({ listCourses: () => Promise.resolve([]) }))
vi.mock('@/api/pricing', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/api/pricing')>()), getPublicPricing, getUsageTotals }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}:${JSON.stringify(p)}` : key), locale: { value: 'zh' } }) }
})

// The demo is its own component; here it only needs to say 「生成」 with a sentence.
const DemoStub = defineComponent({
  emits: ['generate'],
  setup: (_, { emit }) => () => h('button', { 'data-testid': 'demo-generate', onClick: () => emit('generate', '一只猫') }),
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = { site_name: 'Test site', registration_enabled: true, ...settings }
  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: true,
        Icon: true,
        HomeHeroDemo: DemoStub,
        HomeHeroSlogan: true,
        HomeTokenCounter: true,
        HomeMotion: true,
        HomeTeardown: true,
        HomeLiveStats: true,
        HomeJobs: true,
        HomePricing: true,
        HomeCommunityWall: true,
        HomeAssistant: true,
        HomePromoMascot: true,
        HomeFaq: true,
      },
    },
  })
}

const linksTo = (w: ReturnType<typeof mountHome>, to: unknown) =>
  w.findAllComponents(RouterLinkStub).filter((l) => JSON.stringify(l.props('to')) === JSON.stringify(to))

describe('HomeView sign-up calls to action', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    track.mockReset()
    getPublicPricing.mockReset().mockResolvedValue({ starter_image_cost_usd: 0.6, starter_request_cost_usd: 0.11, total_requests: 14220 })
    getUsageTotals.mockReset().mockResolvedValue({ total_tokens: 1912852691, total_requests: 18728 })
    localStorage.clear()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('shows a register button next to login in the nav', async () => {
    const w = mountHome()
    const nav = linksTo(w, '/register').find((l) => l.text().startsWith('home.register'))
    expect(nav).toBeTruthy()
    await nav!.trigger('click')
    expect(track).toHaveBeenCalledWith('cta_click', { where: 'nav-register' })
  })

  it('hides every register button when registration is closed', () => {
    const w = mountHome({ registration_enabled: false })
    expect(linksTo(w, '/register')).toHaveLength(0)
    expect(w.find('[data-testid="home-offer"]').exists()).toBe(false)
    expect(w.find('[data-testid="home-mobile-bar"]').exists()).toBe(false)
  })

  it('says what the sign-up credit buys, from live billing, and passes the totals on', async () => {
    const w = mountHome({ signup_bonus: 2 })
    await flushPromises()
    const offer = w.get('[data-testid="home-offer"]').text()
    expect(offer).toContain('home.v3.hero.offerTitle:{"amount":"2"}')
    expect(offer).toContain('home.v3.hero.offerBoth:{"images":3,"chats":18}')
    expect(w.get('[data-testid="home-hero-signup"]').text()).toContain('home.v3.hero.cta:{"amount":"2"}')
    expect(w.findComponent({ name: 'HomeLiveStats' }).props()).toMatchObject({ totalTokens: 1912852691, totalRequests: 18728 })
    expect(w.findComponent({ name: 'HomeTeardown' }).props('imageCostUsd')).toBe(0.6)
  })

  it('promises no credit when none is configured', async () => {
    const w = mountHome({ signup_bonus: 0 })
    await flushPromises()
    expect(w.find('[data-testid="home-offer"]').exists()).toBe(false)
    expect(w.get('[data-testid="home-hero-signup"]').text()).toContain('home.v3.hero.ctaNoBonus')
  })

  it('keeps a visitor\'s sentence and asks them to sign up before drawing', async () => {
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    const w = mountHome({ signup_bonus: 2 })
    await flushPromises()
    await w.get('[data-testid="demo-generate"]').trigger('click')
    expect(track).toHaveBeenCalledWith('cta_click', { where: 'hero-generate' })
    expect(open).not.toHaveBeenCalled()
    expect(w.get('[data-testid="home-signup-sheet-prompt"]').text()).toBe('一只猫')
    expect(JSON.parse(localStorage.getItem('hg_pending_prompt') || '{}').text).toBe('一只猫')
    const register = w.findAllComponents(RouterLinkStub).find((l) => l.attributes('data-testid') === 'home-signup-sheet-register')
    expect(register!.props('to')).toEqual({ path: '/register', query: { redirect: '/first-image' } })
    open.mockRestore()
  })

  it('sends a member straight to the canvas with the sentence', async () => {
    authStore.isAuthenticated = true
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    const w = mountHome()
    await w.get('[data-testid="demo-generate"]').trigger('click')
    expect(open).toHaveBeenCalledOnce()
    const url = new URL(String(open.mock.calls[0][0]))
    expect(url.pathname).toBe('/image')
    expect(url.searchParams.get('prompt')).toBe('一只猫')
    expect(w.find('[data-testid="home-signup-sheet"]').exists()).toBe(false)
    expect(localStorage.getItem('hg_pending_prompt')).toBeNull()
    open.mockRestore()
  })
})
