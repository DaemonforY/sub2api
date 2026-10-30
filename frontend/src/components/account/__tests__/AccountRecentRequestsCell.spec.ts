import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountRecentRequestsCell from '../AccountRecentRequestsCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
    })
  }
})

const buckets = (fill: Record<number, { s: number; f: number }>) =>
  Array.from({ length: 20 }, (_, i) => fill[i] ?? { s: 0, f: 0 })

describe('AccountRecentRequestsCell', () => {
  it('colors each 3-minute cell by outcome and shows totals with success rate', () => {
    const wrapper = mount(AccountRecentRequestsCell, {
      props: {
        stats: { success: 17, failure: 3, buckets: buckets({ 0: { s: 5, f: 0 }, 1: { s: 0, f: 2 }, 2: { s: 12, f: 1 } }) },
        windowStart: '2026-10-30T10:00:00Z',
        bucketSeconds: 180
      }
    })
    const cells = wrapper.find('[data-testid="recent-requests-grid"]').findAll('span')
    expect(cells).toHaveLength(20)
    expect(cells[0].classes()).toContain('bg-emerald-500')
    expect(cells[1].classes()).toContain('bg-red-500')
    expect(cells[2].classes()).toContain('bg-amber-400')
    expect(cells[3].classes()).toContain('bg-gray-200')
    expect(cells[1].attributes('title')).toContain('"s":0,"f":2')
    expect(wrapper.text()).toContain('17')
    expect(wrapper.text()).toContain('85%')
  })

  it('says there were no requests when the hour is empty', () => {
    const wrapper = mount(AccountRecentRequestsCell, {
      props: { stats: { success: 0, failure: 0, buckets: buckets({}) }, windowStart: null, bucketSeconds: 180 }
    })
    expect(wrapper.text()).toContain('admin.accounts.recentRequests.empty')
  })
})
