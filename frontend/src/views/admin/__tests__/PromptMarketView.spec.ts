import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import PromptMarketView from '../PromptMarketView.vue'

const api = vi.hoisted(() => ({
  getPromptMarketConfig: vi.fn(),
  savePromptMarketConfig: vi.fn(),
  listPromptListings: vi.fn(),
  reviewPromptListing: vi.fn(),
  listPromptPurchases: vi.fn(),
  refundPromptPurchase: vi.fn()
}))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/promptMarket', () => api)
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' } }
const pending = {
  id: 5, seller_name: '小鹿', seller_email: 's@test.cn', title: '封面合集', description: '', cover_url: '', sample_urls: [], price: 0.8, item_count: 2,
  status: 'pending', flags: ['nsfw'], sales_count: 0, created_at: '', updated_at: '',
  items: [{ position: 0, title: 'A', prompt: '完整提示词 A', tags: [], kind: 'image' }, { position: 1, title: 'B', prompt: '完整提示词 B', tags: [], kind: 'image' }]
}

describe('admin PromptMarketView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    api.getPromptMarketConfig.mockResolvedValue({ enabled: false, fee_rate: 0.2, min_price: 0.1, max_price: 50 })
    api.listPromptListings.mockResolvedValue({ items: [pending], total: 1 })
    api.listPromptPurchases.mockResolvedValue({ items: [{ id: 9, listing_id: 5, buyer_user_id: 2, seller_user_id: 3, price: 0.8, fee_rate: 0.2, seller_amount: 0.64, status: 'paid', created_at: '', listing_title: '封面合集' }], total: 1 })
  })

  it('opens the market with the fee as a percentage', async () => {
    api.savePromptMarketConfig.mockImplementation(async (cfg) => cfg)
    const wrapper = mount(PromptMarketView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="prompt-market-enabled"]').setValue(true)
    await wrapper.get('[data-testid="prompt-market-config"]').trigger('submit')
    await flushPromises()
    expect(api.savePromptMarketConfig).toHaveBeenCalledWith({ enabled: true, fee_rate: 0.2, min_price: 0.1, max_price: 50 })
  })

  it('shows the full content of a pending listing and approves it', async () => {
    api.reviewPromptListing.mockResolvedValue({})
    const wrapper = mount(PromptMarketView, { global: { stubs } })
    await flushPromises()
    expect(api.listPromptListings).toHaveBeenCalledWith({ status: 'review', page: 1, page_size: 20 })
    const card = wrapper.get('[data-testid="prompt-market-listing"]')
    expect(card.text()).toContain('完整提示词 B')
    expect(card.text()).toContain('nsfw')
    await wrapper.get('[data-testid="prompt-market-approve"]').trigger('click')
    await flushPromises()
    expect(api.reviewPromptListing).toHaveBeenCalledWith(5, 'approve', '')
  })

  it('refunds a purchase after confirming', async () => {
    api.refundPromptPurchase.mockResolvedValue({})
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mount(PromptMarketView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="prompt-market-tab-purchases"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="prompt-market-purchases"] button').trigger('click')
    await flushPromises()
    expect(api.refundPromptPurchase).toHaveBeenCalledWith(9)
  })
})
