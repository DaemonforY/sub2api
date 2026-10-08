import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PricingView from '@/views/PricingView.vue'
import { planDays, planMonthlyCapUSD, planMonthlyPrice, type PricingPlan } from '@/api/pricing'

const getPublicPricing = vi.fn()
vi.mock('@/api/pricing', async () => {
  const actual = await vi.importActual<typeof import('@/api/pricing')>('@/api/pricing')
  return { ...actual, getPublicPricing: (...a: unknown[]) => getPublicPricing(...a) }
})
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (k: string, p?: Record<string, unknown>) => (p ? `${k}${JSON.stringify(p)}` : k) }) }
})

const plan = (over: Partial<PricingPlan>): PricingPlan => ({
  id: 1, name: '月度订阅', description: '', price: 120, currency: '', validity_days: 30, validity_unit: 'days', features: '',
  group_name: 'Codex-Pro', platform: 'openai', daily_limit_usd: 60, weekly_limit_usd: 350, monthly_limit_usd: 1200, ...over
})

const pricing = {
  payment_enabled: true,
  recharge_multiplier: 1,
  min_recharge: 1,
  plans: [plan({}), plan({ id: 3, name: '周订阅', price: 40, validity_days: 7 }), plan({ id: 4, name: '轻量版', price: 60, daily_limit_usd: 35, weekly_limit_usd: 200, monthly_limit_usd: 600 })],
  pay_as_you_go: [{ name: 'GPT-按量', platform: 'openai', description: '', rate_multiplier: 1 }],
  avg_request_cost_usd: 0.12,
  avg_sample_requests: 8000
}

function mountView() {
  return mount(PricingView, { global: { stubs: { PlazaNavBar: true, RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } } })
}

describe('pricing helpers', () => {
  it('normalises plan length and caps like billing does', () => {
    expect(planDays({ validity_days: 1, validity_unit: 'months' })).toBe(30)
    expect(planDays({ validity_days: 2, validity_unit: 'Week' })).toBe(14)
    expect(planDays({ validity_days: 7, validity_unit: 'days' })).toBe(7)
    expect(planMonthlyCapUSD(plan({}))).toBe(1200)
    expect(planMonthlyCapUSD(plan({ daily_limit_usd: 10 }))).toBe(300)
    expect(planMonthlyCapUSD(plan({ daily_limit_usd: null, weekly_limit_usd: null, monthly_limit_usd: null }))).toBeNull()
    expect(planMonthlyPrice(plan({ price: 40, validity_days: 7 }))).toBeCloseTo(171.43, 1)
  })
})

describe('PricingView', () => {
  it('lists live plans and recommends by usage', async () => {
    getPublicPricing.mockResolvedValue(pricing)
    const w = mountView()
    await flushPromises()
    expect(w.get('[data-testid="pricing-plans"]').text()).toContain('月度订阅')
    expect(w.get('[data-testid="pricing-plan-1"]').text()).toContain('"usd":"1,200"')

    // 80 requests × $0.12 × 22 days ≈ ¥211 pay as you go; the ¥60 plan fits ($9.6/day < $35).
    expect(w.get('[data-testid="pricing-estimate-payg"]').text()).toContain('¥211')
    expect(w.get('[data-testid="pricing-recommendation"]').text()).toContain('轻量版')

    // Light use: pay as you go wins.
    await w.get('[data-testid="pricing-requests-input"]').setValue('5')
    expect(w.get('[data-testid="pricing-recommendation"]').text()).toBe('pricing.estimate.pickPayg')

    // Heavy use beyond the light plan's daily $35: the ¥120 plan.
    await w.get('[data-testid="pricing-requests-input"]').setValue('400')
    expect(w.get('[data-testid="pricing-estimate-plan-4"]').text()).toContain('pricing.estimate.overLimit')
    expect(w.get('[data-testid="pricing-recommendation"]').text()).toContain('月度订阅')
  })

  it('hides the estimate without enough real requests', async () => {
    getPublicPricing.mockResolvedValue({ ...pricing, avg_sample_requests: 10 })
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="pricing-estimator"]').exists()).toBe(false)
    expect(w.find('[data-testid="pricing-plans"]').exists()).toBe(true)
  })
})
