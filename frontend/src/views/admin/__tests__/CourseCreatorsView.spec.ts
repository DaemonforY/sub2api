import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CourseCreatorsView from '../CourseCreatorsView.vue'

const api = vi.hoisted(() => ({
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  creators: vi.fn(),
  updateCreator: vi.fn(),
  reviews: vi.fn(),
  review: vi.fn(),
  deliveries: vi.fn(),
  creatorWithdrawals: vi.fn(),
  payCreatorWithdrawal: vi.fn(),
  rejectCreatorWithdrawal: vi.fn()
}))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin', () => ({ adminAPI: { courses: api } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' } }
const settings = { affiliate_rate_percent: 10, creator_enabled: true, creator_commission_percent: 20, creator_settle_days: 7, creator_withdraw_min_cny: 50 }
const applicant = { user_id: 3, status: 'pending', display_name: '王老师', bio: '', contact: 'wx', plan: 'AI 办公', admin_note: '', created_at: '', course_count: 0, on_sale_count: 0, pending_count: 0, commission_percent: null }
const submitted = {
  id: 7, slug: 'ai-office', title: 'AI 办公', price: 99, cover_url: '', outline: [], status: 'draft', creator_name: '王老师', review_status: 'pending', delivery_version: 1,
  draft: { title: 'AI 办公', subtitle: '', category: '', price: 99, original_price: 0, intro_md: '## 介绍', outline: [], trial_md: '', faq_md: '', sale_price: 0, sale_ends_at: null, edu_discount: true, trial_video_url: '' }
}

describe('admin CourseCreatorsView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    api.getSettings.mockResolvedValue(settings)
    api.creators.mockResolvedValue([applicant])
    api.reviews.mockResolvedValue([submitted])
    api.creatorWithdrawals.mockResolvedValue({ items: [], total: 0 })
    api.deliveries.mockResolvedValue([{ version: 1, link: 'https://pan.baidu.com/s/1x', code: 'ab12', password: '', note: '', updated_at: '' }])
  })

  it('approves an applicant with their own commission', async () => {
    api.updateCreator.mockResolvedValue({})
    const wrapper = mount(CourseCreatorsView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creator-row"] button').trigger('click')
    await wrapper.get('[data-testid="creator-own-rate"]').setValue('10')
    await wrapper.get('[data-testid="creator-save"]').trigger('click')
    await flushPromises()
    expect(api.updateCreator).toHaveBeenCalledWith(3, { status: 'approved', commission_percent: 10, admin_note: '' })
  })

  it('shows the netdisk link of a submission and needs a reason to reject it', async () => {
    api.review.mockResolvedValue({})
    const wrapper = mount(CourseCreatorsView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creators-tab-reviews"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="review-row"] button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="review-delivery"]').text()).toContain('https://pan.baidu.com/s/1x')
    expect((wrapper.get('[data-testid="review-reject"]').element as HTMLButtonElement).disabled).toBe(true)
    await wrapper.get('[data-testid="review-approve"]').trigger('click')
    await flushPromises()
    expect(api.review).toHaveBeenCalledWith(7, true, '')
  })

  it('marks a withdrawal paid', async () => {
    const pending = { id: 5, user_id: 3, display_name: '王老师', cny_amount: 120, method: 'alipay', account: 'a@alipay', real_name: '王某', user_note: '', status: 'pending', admin_note: '', created_at: '' }
    api.creatorWithdrawals.mockResolvedValue({ items: [pending], total: 1 })
    api.payCreatorWithdrawal.mockResolvedValue({})
    const wrapper = mount(CourseCreatorsView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creators-tab-withdrawals"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="creator-withdrawal"]').text()).toContain('a@alipay')
    await wrapper.get('[data-testid="withdrawal-paid"]').trigger('click')
    await flushPromises()
    expect(api.payCreatorWithdrawal).toHaveBeenCalledWith(5, '')
  })
})
