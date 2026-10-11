import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ChannelLinksView from '@/views/admin/ChannelLinksView.vue'

const listChannelLinks = vi.fn()
const createChannelLink = vi.fn()
const updateChannelLink = vi.fn()
const deleteChannelLink = vi.fn()

vi.mock('@/api/admin/analytics', () => ({
  listChannelLinks: (...a: unknown[]) => listChannelLinks(...a),
  createChannelLink: (...a: unknown[]) => createChannelLink(...a),
  updateChannelLink: (...a: unknown[]) => updateChannelLink(...a),
  deleteChannelLink: (...a: unknown[]) => deleteChannelLink(...a)
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key),
      te: (key: string) => key === 'admin.channelLinks.source.xiaohongshu'
    })
  }
})

const link = {
  id: 1, code: 'xhs-oct', name: '小红书 10 月', source: 'xiaohongshu', medium: 'post', target_path: '/pricing',
  aff_code: '', promo_code: '', note: '', clicks: 40, created_at: '', updated_at: '',
  visitors: 31, signups: 6, activated: 3, paid_users: 1, revenue: 60,
  usage_usd: 400, rebates: 12, cost_cny: 0, profit: 0
}

const mountView = () =>
  mount(ChannelLinksView, {
    global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, RouterLink: { template: '<a><slot /></a>' } } }
  })

describe('ChannelLinksView', () => {
  it('lists links with their short URL and funnel', async () => {
    listChannelLinks.mockResolvedValue({ days: 30, cost_per_usd: 0, links: [link] })
    const w = mountView()
    await flushPromises()
    expect(listChannelLinks).toHaveBeenCalledWith(30)
    const table = w.get('[data-testid="channel-links-table"]').text()
    expect(table).toContain('小红书 10 月')
    expect(table).toContain('admin.channelLinks.source.xiaohongshu')
    expect(table).toContain('/go/xhs-oct')
    expect(table).toContain('60.00')
    // No cost rate: a prompt to set it, no profit column.
    expect(w.find('[data-testid="channel-links-cost-missing"]').exists()).toBe(true)
    expect(w.find('[data-testid="channel-link-profit-xhs-oct"]').exists()).toBe(false)
  })

  it('shows cost and profit when the cost rate is set, and reloads per period', async () => {
    listChannelLinks.mockResolvedValue({
      days: 30, cost_per_usd: 0.15,
      links: [{ ...link, cost_cny: 60, profit: -12, aff_code: 'KOLA', promo_code: 'KOLA10' }]
    })
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="channel-links-cost-missing"]').exists()).toBe(false)
    const profit = w.get('[data-testid="channel-link-profit-xhs-oct"]')
    expect(profit.text()).toBe('-12.00')
    expect(profit.classes()).toContain('text-red-600')
    expect(w.text()).toContain('admin.channelLinks.promoTag{"code":"KOLA10"}')
    await w.get('[data-testid="channel-links-period"]').setValue('0')
    await flushPromises()
    expect(listChannelLinks).toHaveBeenLastCalledWith(0)
  })

  it('creates a link from the form', async () => {
    listChannelLinks.mockResolvedValue({ days: 30, cost_per_usd: 0, links: [] })
    createChannelLink.mockResolvedValue({ ...link, id: 2, code: 'bilibili-k3m9', source: 'bilibili' })
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="channel-link-new"]').trigger('click')
    await w.get('#cl-name').setValue('B 站演示视频')
    await w.get('#cl-source').setValue('bilibili')
    await w.get('#cl-target').setValue('/learn/connect/')
    await w.get('#cl-aff').setValue(' abc123 ')
    await w.get('#cl-promo').setValue(' kol10 ')
    await w.get('[data-testid="channel-link-form"]').trigger('submit')
    await flushPromises()
    expect(createChannelLink).toHaveBeenCalledWith(expect.objectContaining({
      name: 'B 站演示视频', source: 'bilibili', target_path: '/learn/connect/', aff_code: 'ABC123', promo_code: 'KOL10', code: ''
    }))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringMatching(/\/go\/bilibili-k3m9$/))
  })
})
