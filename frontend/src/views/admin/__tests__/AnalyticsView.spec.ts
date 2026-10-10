import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AnalyticsView from '@/views/admin/AnalyticsView.vue'

const getOverview = vi.fn()
const getReminder = vi.fn()
const setReminder = vi.fn()

vi.mock('@/api/admin/analytics', () => ({
  getOverview: (...a: unknown[]) => getOverview(...a),
  getReminder: (...a: unknown[]) => getReminder(...a),
  setReminder: (...a: unknown[]) => setReminder(...a)
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
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
    getReminder.mockResolvedValue(null)
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

  it('labels AI assistants and sums them in an AI total row', async () => {
    getOverview.mockResolvedValue({
      ...overview,
      channels: [
        { source: 'ai:chatgpt', visitors: 5, signups: 2, activated: 1, paid_users: 1, revenue: 20 },
        { source: 'ai:doubao', visitors: 3, signups: 1, activated: 0, paid_users: 0, revenue: 0 },
        { source: 'invite', visitors: 26, signups: 5, activated: 2, paid_users: 1, revenue: 120 }
      ]
    })
    getReminder.mockResolvedValue(null)
    const w = mount(AnalyticsView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } } })
    await flushPromises()
    const channels = w.get('[data-testid="analytics-channels"]').text()
    expect(channels).toContain('{"name":"ChatGPT"}')
    expect(channels).toContain('{"name":"豆包"}')
    const total = w.get('[data-testid="analytics-ai-total"]').text()
    expect(total).toContain('8')
    expect(total).toContain('20.00')

    getOverview.mockResolvedValue(overview)
    await w.get('[data-testid="analytics-range-7"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="analytics-ai-total"]').exists()).toBe(false)
  })

  it('turns the reminder email on only after confirming', async () => {
    getOverview.mockResolvedValue(overview)
    const off = { enabled: false, sent: 0, due: 3, subject: 'S', preview: '<p>hi</p>' }
    getReminder.mockResolvedValue(off)
    setReminder.mockResolvedValue({ ...off, enabled: true })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false).mockReturnValueOnce(true)
    const w = mount(AnalyticsView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } } })
    await flushPromises()
    expect(w.get('[data-testid="analytics-reminder"]').text()).toContain('{"sent":0,"due":3}')

    await w.get('[data-testid="analytics-reminder-toggle"]').trigger('click')
    expect(setReminder).not.toHaveBeenCalled()
    await w.get('[data-testid="analytics-reminder-toggle"]').trigger('click')
    await flushPromises()
    expect(setReminder).toHaveBeenCalledWith(true)
    expect(confirm.mock.calls[1][0]).toContain('{"due":3}')
    expect(w.get('[data-testid="analytics-reminder-toggle"]').text()).toBe('admin.analytics.reminder.turnOff')

    await w.get('[data-testid="analytics-reminder-preview"]').trigger('click')
    expect(w.get('iframe').attributes('sandbox')).toBe('')
    confirm.mockRestore()
  })
})
