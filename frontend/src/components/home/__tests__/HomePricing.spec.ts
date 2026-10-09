import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomePricing from '../HomePricing.vue'

const { getPublicPricing } = vi.hoisted(() => ({ getPublicPricing: vi.fn() }))
vi.mock('@/api/pricing', async () => {
  const actual = await vi.importActual<typeof import('@/api/pricing')>('@/api/pricing')
  return { ...actual, getPublicPricing }
})
vi.mock('@/utils/analytics', () => ({ track: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (k: string, p?: Record<string, unknown>) => (p ? `${k}:${JSON.stringify(p)}` : k), locale: { value: 'zh' } }),
}))

const plan = (over: Record<string, unknown>) => ({
  id: 1, name: '月度订阅', description: '', price: 120, original_price: 299, currency: '', validity_days: 30, validity_unit: 'days',
  features: '', group_name: 'Codex-Pro', platform: 'openai', daily_limit_usd: 60, weekly_limit_usd: 350, monthly_limit_usd: 1200, ...over,
})
const pricing = (over: Record<string, unknown> = {}) => ({
  payment_enabled: true, recharge_multiplier: 1, min_recharge: 1, avg_request_cost_usd: 0.125, avg_sample_requests: 7000,
  plans: [plan({ id: 3, name: '周订阅', price: 40, validity_days: 7 }), plan({ recommended: true })], pay_as_you_go: [], ...over,
})
const mountIt = (auth = false) => mount(HomePricing, { props: { isAuthenticated: auth, signupBonus: 2 }, global: { stubs: { RouterLink: RouterLinkStub } } })

describe('HomePricing', () => {
  beforeEach(() => getPublicPricing.mockReset())

  it('puts the recommended plan first and shows what it is worth against pay-as-you-go', async () => {
    getPublicPricing.mockResolvedValue(pricing())
    const w = mountIt()
    await flushPromises()
    const cards = w.findAll('[data-testid^="home-plan-"]').filter((c) => /home-plan-\d+$/.test(c.attributes('data-testid') || ''))
    expect(cards[0].attributes('data-testid')).toBe('home-plan-1')
    expect(w.get('[data-testid="home-plan-saving-1"]').text()).toContain('"value":"1200","percent":90')
    expect(w.get('[data-testid="home-plan-1"]').text()).toContain('"n":480')
    // A 7-day plan doesn't advertise a monthly limit, and it is worth one week of usage.
    expect(w.get('[data-testid="home-plan-3"]').text()).not.toContain('home.pricing.monthly')
    expect(w.get('[data-testid="home-plan-saving-3"]').text()).toContain('"value":"350"')
    expect(w.find('[data-testid="home-plan-payg"]').exists()).toBe(true)
  })

  it('sends visitors through sign-up to the chosen plan', async () => {
    getPublicPricing.mockResolvedValue(pricing())
    const guest = mountIt(false)
    await flushPromises()
    const to = guest.get('[data-testid="home-plan-1"]').findComponent(RouterLinkStub).props('to')
    expect(to).toBe('/register?redirect=' + encodeURIComponent('/purchase?tab=subscription&plan=1'))

    const member = mountIt(true)
    await flushPromises()
    expect(member.get('[data-testid="home-plan-1"]').findComponent(RouterLinkStub).props('to')).toBe('/purchase?tab=subscription&plan=1')
  })

  it('stays hidden without payments or plans', async () => {
    getPublicPricing.mockResolvedValue(pricing({ payment_enabled: false }))
    const w = mountIt()
    await flushPromises()
    expect(w.find('[data-testid="home-pricing"]').exists()).toBe(false)
  })

  it('reassures buyers, and shows the request total only once it is large', async () => {
    getPublicPricing.mockResolvedValue(pricing({ total_requests: 8000 }))
    const small = mountIt()
    await flushPromises()
    expect(small.find('[data-testid="home-trust"]').text()).toContain('home.trust.failures.title')
    expect(small.find('[data-testid="home-total-requests"]').exists()).toBe(false)

    getPublicPricing.mockResolvedValue(pricing({ total_requests: 16384 }))
    const big = mountIt()
    await flushPromises()
    expect(big.get('[data-testid="home-total-requests"]').text()).toContain('"n":"1.6 万"')
  })
})
