import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import HomePromoMascot from '../HomePromoMascot.vue'

const { getPublicConfig, getCheckoutInfo, appState, authState } = vi.hoisted(() => ({
  getPublicConfig: vi.fn(),
  getCheckoutInfo: vi.fn(),
  appState: { cachedPublicSettings: { signup_bonus: 2, registration_enabled: true } as Record<string, unknown> },
  authState: { isAuthenticated: false }
}))

vi.mock('@/api/growth', () => ({ growthAPI: { getPublicConfig } }))
vi.mock('@/api/payment', () => ({ paymentAPI: { getCheckoutInfo } }))
vi.mock('@/stores', () => ({ useAppStore: () => appState, useAuthStore: () => authState }))
vi.mock('@/utils/analytics', () => ({ track: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, args?: Record<string, unknown>) => (args ? `${key} ${JSON.stringify(args)}` : key) })
}))

const config = (over = {}) => ({
  affiliate_enabled: true,
  invitee_bonus_rate_percent: 10,
  invitee_bonus_cap: 0,
  invitee_signup_bonus: 0,
  inviter_rebate_rate_percent: 15,
  price_lock_enabled: true,
  price_lock_grace_days: 7,
  leaderboard_enabled: false,
  edu_verify_enabled: true,
  edu_discount_percent: 20,
  edu_email_suffixes: [],
  first_topup_bonus_percent: 20,
  first_topup_bonus_cap: 10,
  first_topup_min_amount: 10,
  ...over
})

const mountMascot = () => mount(HomePromoMascot, { global: { stubs: { RouterLink: RouterLinkStub } } })
const offerKeys = (wrapper: ReturnType<typeof mountMascot>) =>
  wrapper.findAll('[data-testid^="promo-offer-"]').map((el) => el.attributes('data-testid')!.replace('promo-offer-', ''))

describe('home promo mascot', () => {
  beforeEach(() => {
    getPublicConfig.mockReset()
    getCheckoutInfo.mockReset()
    localStorage.clear()
    authState.isAuthenticated = false
    appState.cachedPublicSettings = { signup_bonus: 2, registration_enabled: true }
  })

  it('lists every enabled offer for guests and points them at sign-up', async () => {
    getPublicConfig.mockResolvedValue(config())
    const wrapper = mountMascot()
    await flushPromises()
    expect(wrapper.text()).toContain('promoMascot.bubble {"n":5}')

    await wrapper.get('[data-testid="home-promo-mascot-toggle"]').trigger('click')
    expect(offerKeys(wrapper)).toEqual(['signup', 'first_topup', 'invite', 'edu', 'price_lock'])
    expect(wrapper.text()).toContain('promoMascot.firstTopup.descMinCap {"min":"10","cap":"10"}')
    expect(wrapper.text()).toContain('promoMascot.invite.descBoth {"bonus":"10","rebate":"15"}')
    expect(wrapper.findComponent(RouterLinkStub).props('to')).toBe('/register')
    expect(getCheckoutInfo).not.toHaveBeenCalled()
    expect(localStorage.getItem('promo_mascot_seen')).toBe(new Date().toDateString())
  })

  it('hides offers that are switched off and disappears when none are left', async () => {
    appState.cachedPublicSettings = { signup_bonus: 0 }
    getPublicConfig.mockResolvedValue(
      config({ affiliate_enabled: false, edu_verify_enabled: false, price_lock_enabled: false, first_topup_bonus_percent: 0 })
    )
    const wrapper = mountMascot()
    await flushPromises()
    expect(wrapper.find('[data-testid="home-promo-mascot"]').exists()).toBe(false)
  })

  it('marks used offers as claimed for signed-in users and sorts them last', async () => {
    authState.isAuthenticated = true
    getPublicConfig.mockResolvedValue(config())
    getCheckoutInfo.mockResolvedValue({ data: { first_topup_eligible: false, edu_discount_active: true } })
    const wrapper = mountMascot()
    await flushPromises()
    expect(wrapper.text()).toContain('promoMascot.bubble {"n":4}')

    await wrapper.get('[data-testid="home-promo-mascot-toggle"]').trigger('click')
    await flushPromises()
    expect(offerKeys(wrapper)).toEqual(['invite', 'price_lock', 'signup', 'first_topup', 'edu'])
    expect(wrapper.get('[data-testid="promo-offer-edu"]').text()).toContain('promoMascot.claimed')
  })
})
