import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AnalyticsView from '@/views/admin/AnalyticsView.vue'

const getOverview = vi.fn()

vi.mock('@/api/admin/analytics', () => ({ getOverview: (...a: unknown[]) => getOverview(...a) }))
vi.mock('vue-chartjs', () => ({ Line: { template: '<div data-testid="chart" />' } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const known = new Set(['admin.analytics.source.invite', 'admin.analytics.events.key_created'])
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key),
      te: (key: string) => known.has(key)
    })
  }
})

const overview = {
  since: '2026-09-09',
  days: 30,
  totals: { visitors: 120, active_users: 9, wau: 7, signups: 10, activated: 4, paid_users: 2, revenue: 160 },
  daily: [{ day: '2026-10-08', visitors: 61, page_views: 170, active_users: 5, api_users: 4, signups: 6, activated: 2, orders: 1, revenue: 120 }],
  funnel: [
    { step: 'visitors', count: 120 },
    { step: 'signup_view', count: 30 },
    { step: 'signups', count: 10 }
  ],
  channels: [
    { source: 'invite', visitors: 26, signups: 5, activated: 2, paid_users: 1, revenue: 120 },
    { source: 'ref:v2ex.com', visitors: 3, signups: 1, activated: 0, paid_users: 0, revenue: 0 },
    { source: 'poster', visitors: 30, signups: 2, activated: 1, paid_users: 0, revenue: 0 }
  ],
  pages: [{ app: 'learn', path: '/learn/a/a1', views: 12, visitors: 8 }],
  features: [{ event: 'key_created', count: 3, visitors: 3, users: 3 }],
  retention: [{ week: '2026-10-05', signups: 10, day1: 3, day2_7: 2, day8_30: 0, activated: 4 }],
  devices: { desktop: 80, mobile: 40 }
}

describe('AnalyticsView', () => {
  it('loads 30 days and renders KPIs, channels, features and retention', async () => {
    getOverview.mockResolvedValue(overview)
    const w = mount(AnalyticsView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } } })
    await flushPromises()
    expect(getOverview).toHaveBeenCalledWith(30)
    expect(w.get('[data-testid="analytics-kpis"]').text()).toContain('4（40%）')
    const channels = w.get('[data-testid="analytics-channels"]').text()
    expect(channels).toContain('admin.analytics.source.invite')
    expect(channels).toContain('{"host":"v2ex.com"}')
    expect(channels).toContain('poster')
    expect(w.get('[data-testid="analytics-features"]').text()).toContain('admin.analytics.events.key_created')
    expect(w.get('[data-testid="analytics-funnel"]').text()).toContain('{"p":"25%"}')
    expect(w.text()).toContain('3（30%）')

    getOverview.mockResolvedValue({ ...overview, days: 7 })
    await w.get('[data-testid="analytics-range-7"]').trigger('click')
    await flushPromises()
    expect(getOverview).toHaveBeenLastCalledWith(7)
  })
})
