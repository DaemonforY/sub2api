import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MarginView from '@/views/admin/MarginView.vue'

const getMargin = vi.fn()
const setMarginCost = vi.fn()

vi.mock('@/api/admin/analytics', () => ({
  getMargin: (...a: unknown[]) => getMargin(...a),
  setMarginCost: (...a: unknown[]) => setMarginCost(...a)
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key) }) }
})

const report = (configured: boolean) => ({
  cost_per_usd: configured ? 0.3 : 0,
  configured,
  days: 30,
  since: '2026-09-10T00:00:00+08:00',
  totals: { revenue: 343.75, recharge_revenue: 61.75, subscription_revenue: 282, other_revenue: 0, paygo_billed: 290.99, paygo_cost: 71.5, subscription_cost: 120, admin_cost: 210, member_cost: 191.5, margin: 152.25 },
  groups: [{ group_id: 5, name: 'GPT-按量', subscription: false, users: 6, usage_usd: 238.33, billed_usd: 290.99, cost: 71.5, margin: 219.49 }],
  plans: [{ id: 1, name: '月度订阅', group_name: 'Codex-Pro', price: 120, days: 30, cap_usd: 1200, max_cost: 360, max_loss: 240, break_even_usd: 400, sold: 2, revenue: 240 }],
  subscriptions: [
    { id: 14, user_id: 23, email: 'h****@example.com', group_name: 'Codex-Pro', starts_at: '2026-10-05T00:00:00Z', expires_at: '2026-11-04T00:00:00Z', paid: 120, usage_usd: 160, active: true, days_elapsed: 4, days_total: 30, cost: 48, margin: 72, projected_usd: 1200, projected_cost: 360, projected_margin: -240, status: 'risk' },
    { id: 3, user_id: 4, email: 't****@example.com', group_name: 'Codex-Pro', starts_at: '2026-09-11T00:00:00Z', expires_at: '2026-09-14T00:00:00Z', paid: 0, usage_usd: 324, active: false, days_elapsed: 3, days_total: 3, cost: 97.2, margin: -97.2, projected_usd: 324, projected_cost: 97.2, projected_margin: -97.2, status: 'gift' }
  ],
  users: [{ user_id: 23, email: 'h****@example.com', usage_usd: 357, billed_usd: 0, paid: 120, cost: 107.1, margin: 12.9 }]
})

const mountView = () => mount(MarginView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } } })

describe('MarginView', () => {
  it('shows totals, the losing plan and subscriptions worst first', async () => {
    getMargin.mockResolvedValue(report(true))
    const w = mountView()
    await flushPromises()
    expect(getMargin).toHaveBeenCalledWith(30)
    expect(w.find('[data-testid="margin-unconfigured"]').exists()).toBe(false)
    expect(w.get('[data-testid="margin-totals"]').text()).toContain('¥152.25')
    const plans = w.get('[data-testid="margin-plans"]').text()
    expect(plans).toContain('$1,200.00')
    expect(plans).toContain('−¥240.00')
    expect(plans).toContain('$400.00')
    const risk = w.get('[data-testid="margin-sub-14"]').text()
    expect(risk).toContain('−¥240.00')
    expect(risk).toContain('admin.margin.status.risk')
    expect(w.get('[data-testid="margin-sub-3"]').text()).toContain('admin.margin.status.gift')
    expect((w.get('#margin-cost').element as HTMLInputElement).value).toBe('0.3')
  })

  it('asks for the cost rate, then saves it', async () => {
    getMargin.mockResolvedValue(report(false))
    setMarginCost.mockResolvedValue(report(true))
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="margin-unconfigured"]').exists()).toBe(true)
    await w.get('#margin-cost').setValue('0.3')
    await w.get('[data-testid="margin-save-cost"]').trigger('submit')
    await flushPromises()
    expect(setMarginCost).toHaveBeenCalledWith(0.3, 30)
    expect(w.find('[data-testid="margin-unconfigured"]').exists()).toBe(false)
  })
})
