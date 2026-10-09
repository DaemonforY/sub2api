import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore, track } = vi.hoisted(() => ({
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
}))

vi.mock('@/stores', () => ({ useAppStore: () => appStore, useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/utils/analytics', () => ({ track: (...a: unknown[]) => track(...a) }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}:${JSON.stringify(p)}` : key) }) }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = { site_name: 'Test site', registration_enabled: true, ...settings }
  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: true,
        Icon: true,
        HomePromptBox: true,
        HomeScenarios: true,
        HomeShowcase: true,
        HomeCommunityWall: true,
        HomeHeroCollage: true,
        HomeSteps: true,
        HomeSupport: true,
        HomeAssistant: true,
        HomeFaq: true,
      },
    },
  })
}

const linksTo = (w: ReturnType<typeof mountHome>, to: string) =>
  w.findAllComponents(RouterLinkStub).filter((l) => l.props('to') === to)

describe('HomeView sign-up calls to action', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    track.mockReset()
  })

  it('shows a register button next to login in the nav', async () => {
    const w = mountHome()
    const nav = linksTo(w, '/register').find((l) => l.text() === 'home.register')
    expect(nav).toBeTruthy()
    await nav!.trigger('click')
    expect(track).toHaveBeenCalledWith('cta_click', { where: 'nav-register' })
  })

  it('hides the register button when registration is closed', () => {
    const w = mountHome({ registration_enabled: false })
    expect(linksTo(w, '/register').some((l) => l.text() === 'home.register')).toBe(false)
  })

  it('promises sign-up credit only when it is configured', () => {
    expect(mountHome({ signup_bonus: 0 }).text()).not.toContain('home.signupBonus')
    expect(mountHome({ signup_bonus: 2 }).text()).toContain('home.signupBonus:{"amount":"2"}')
  })
})
