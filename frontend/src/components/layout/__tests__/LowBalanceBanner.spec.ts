import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LowBalanceBanner from '../LowBalanceBanner.vue'

const { auth, app, subs, getPublicConfig, getCheckoutInfo, track } = vi.hoisted(() => ({
  auth: { isAuthenticated: true, user: { role: 'user', balance: 0.2 } as { role: string; balance: number } | null },
  app: { cachedPublicSettings: { payment_enabled: true } as Record<string, unknown> },
  subs: { hasActiveSubscriptions: false, fetchActiveSubscriptions: vi.fn().mockResolvedValue([]) },
  getPublicConfig: vi.fn(),
  getCheckoutInfo: vi.fn(),
  track: vi.fn(),
}))

vi.mock('@/stores', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => subs }))
vi.mock('@/api/growth', () => ({ growthAPI: { getPublicConfig } }))
vi.mock('@/api/payment', () => ({ paymentAPI: { getCheckoutInfo } }))
vi.mock('@/utils/analytics', () => ({ track }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }))

const mountBanner = () => mount(LowBalanceBanner, { global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } } })

describe('LowBalanceBanner', () => {
  beforeEach(() => {
    localStorage.clear()
    auth.user = { role: 'user', balance: 0.2 }
    subs.hasActiveSubscriptions = false
    getPublicConfig.mockReset().mockResolvedValue({ first_topup_bonus_percent: 20 })
    getCheckoutInfo.mockReset().mockResolvedValue({ data: { first_topup_eligible: true } })
  })

  it('offers the first top-up bonus when the trial credit is almost gone', async () => {
    const w = mountBanner()
    await flushPromises()
    expect(w.text()).toContain('lowBalance.trialTitle')
    expect(w.get('[data-testid="low-balance-cta"]').attributes('href')).toBe('/purchase')
  })

  it('falls back to a plain reminder once the user has topped up', async () => {
    getCheckoutInfo.mockResolvedValue({ data: { first_topup_eligible: false } })
    const w = mountBanner()
    await flushPromises()
    expect(w.text()).toContain('lowBalance.title')
  })

  it('stays away with enough balance, a subscription, or after being dismissed today', async () => {
    auth.user = { role: 'user', balance: 5 }
    expect(mountBanner().find('[data-testid="low-balance-banner"]').exists()).toBe(false)

    auth.user = { role: 'user', balance: 0.2 }
    subs.hasActiveSubscriptions = true
    const sub = mountBanner()
    await flushPromises()
    expect(sub.find('[data-testid="low-balance-banner"]').exists()).toBe(false)

    subs.hasActiveSubscriptions = false
    const w = mountBanner()
    await flushPromises()
    await w.get('[data-testid="low-balance-dismiss"]').trigger('click')
    expect(w.find('[data-testid="low-balance-banner"]').exists()).toBe(false)
    const again = mountBanner()
    await flushPromises()
    expect(again.find('[data-testid="low-balance-banner"]').exists()).toBe(false)
  })
})
