import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CoursesView from '../CoursesView.vue'

const api = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  uploadCover: vi.fn(),
  uploadImage: vi.fn(),
  deliveries: vi.fn(),
  saveDelivery: vi.fn(),
  enrollments: vi.fn(),
  grant: vi.fn(),
  revoke: vi.fn(),
  getSettings: vi.fn(),
  saveSettings: vi.fn()
}))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin', () => ({ adminAPI: { courses: api } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' }, ConfirmDialog: true, RouterLink: { template: '<a><slot /></a>' } }
const saved = {
  id: 9, slug: 'ai-agent', title: 'AI Agent 实战', subtitle: '', category: '', cover_url: '', price: 199, original_price: 0, intro_md: '', trial_md: '', faq_md: '',
  outline: [], status: 'draft', sort_order: 0, lesson_count: 0, student_count: 0, owned: false, created_at: '', updated_at: '', revenue: 0, delivery_version: 0,
  sale_price: 0, edu_discount: true, trial_video_url: '', current_price: 199, sale_active: false, edu_applied: false
}

describe('admin CoursesView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    api.getSettings.mockResolvedValue({ affiliate_rate_percent: 0, creator_enabled: true, creator_commission_percent: 15, creator_settle_days: 7, creator_withdraw_min_cny: 50 })
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('creates a course, then saves a delivery version and grants a student', async () => {
    api.list.mockResolvedValue([])
    api.create.mockResolvedValue(saved)
    api.saveDelivery.mockResolvedValue({ version: 1 })
    api.deliveries.mockResolvedValue([{ version: 1, link: 'https://pan.baidu.com/s/1abc', code: 'ab12', password: '', note: '', updated_at: '' }])
    api.enrollments.mockResolvedValue({ items: [], total: 0 })
    api.grant.mockResolvedValue(undefined)
    const wrapper = mount(CoursesView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="course-new"]').trigger('click')
    await wrapper.get('[data-testid="course-title"]').setValue('AI Agent 实战')
    await wrapper.get('[data-testid="course-slug"]').setValue('ai-agent')
    await wrapper.get('[data-testid="course-price"]').setValue(199)
    await wrapper.get('[data-testid="course-save"]').trigger('click')
    await flushPromises()
    expect(api.create).toHaveBeenCalledWith(expect.objectContaining({ title: 'AI Agent 实战', slug: 'ai-agent', price: 199, status: 'draft', edu_discount: true, sale_price: 0, sale_ends_at: null }))

    await wrapper.get('[data-testid="course-tab-delivery"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="delivery-link"]').setValue('https://pan.baidu.com/s/1abc')
    await wrapper.get('[data-testid="delivery-code"]').setValue('ab12')
    await wrapper.get('[data-testid="delivery-save"]').trigger('click')
    await flushPromises()
    expect(api.saveDelivery).toHaveBeenCalledWith(9, { link: 'https://pan.baidu.com/s/1abc', code: 'ab12', password: '', note: '', notify: true })
    // With a first version saved, the next one can notify buyers.
    expect(wrapper.find('[data-testid="delivery-notify"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('https://pan.baidu.com/s/1abc')

    await wrapper.get('[data-testid="course-tab-students"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="grant-user"]').setValue('a@example.com')
    await wrapper.get('[data-testid="grant-submit"]').trigger('click')
    await flushPromises()
    expect(api.grant).toHaveBeenCalledWith(9, 'a@example.com', '')
  })

  it('takes the netdisk link while creating a course and saves it as the first version', async () => {
    api.list.mockResolvedValue([])
    api.create.mockResolvedValue(saved)
    api.saveDelivery.mockResolvedValue({ version: 1 })
    api.get.mockResolvedValue({ ...saved, delivery_version: 1 })
    const wrapper = mount(CoursesView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="course-new"]').trigger('click')
    await wrapper.get('[data-testid="course-title"]').setValue('AI Agent 实战')
    await wrapper.get('[data-testid="course-slug"]').setValue('ai-agent')
    await wrapper.get('[data-testid="new-delivery-link"]').setValue('https://pan.baidu.com/s/1abc')
    await wrapper.get('[data-testid="new-delivery-code"]').setValue('ab12')
    await wrapper.get('[data-testid="course-save"]').trigger('click')
    await flushPromises()
    expect(api.create).toHaveBeenCalled()
    expect(api.saveDelivery).toHaveBeenCalledWith(9, { link: 'https://pan.baidu.com/s/1abc', code: 'ab12', password: '', note: '', notify: false })
    expect(wrapper.find('[data-testid="course-new-delivery"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="course-missing-delivery"]').exists()).toBe(false)
  })

  it('warns when a saved course has no netdisk link yet', async () => {
    api.list.mockResolvedValue([])
    api.create.mockResolvedValue(saved)
    const wrapper = mount(CoursesView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="course-new"]').trigger('click')
    await wrapper.get('[data-testid="course-title"]').setValue('AI Agent 实战')
    await wrapper.get('[data-testid="course-save"]').trigger('click')
    await flushPromises()
    expect(api.saveDelivery).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="course-missing-delivery"]').exists()).toBe(true)
  })

  it('saves the course rebate rate and sends a limited-time price with its end time', async () => {
    api.list.mockResolvedValue([])
    api.saveSettings.mockResolvedValue({ affiliate_rate_percent: 20 })
    api.create.mockResolvedValue(saved)
    const wrapper = mount(CoursesView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="course-rebate-rate"]').setValue(20)
    await wrapper.get('[data-testid="course-settings-save"]').trigger('click')
    await flushPromises()
    // The creator rules on the same settings travel along unchanged.
    expect(api.saveSettings).toHaveBeenCalledWith({ affiliate_rate_percent: 20, creator_enabled: true, creator_commission_percent: 15, creator_settle_days: 7, creator_withdraw_min_cny: 50 })

    await wrapper.get('[data-testid="course-new"]').trigger('click')
    await wrapper.get('[data-testid="course-title"]').setValue('课')
    await wrapper.get('[data-testid="course-slug"]').setValue('sale-course')
    await wrapper.get('[data-testid="course-price"]').setValue(199)
    await wrapper.get('[data-testid="course-sale-price"]').setValue(99)
    await wrapper.get('[data-testid="course-sale-ends"]').setValue('2099-01-02T03:04')
    await wrapper.get('[data-testid="course-trial-url"]').setValue('https://www.bilibili.com/video/BV1xx411c7mD')
    await wrapper.get('[data-testid="course-save"]').trigger('click')
    await flushPromises()
    const sent = api.create.mock.calls[0][0]
    expect(sent.sale_price).toBe(99)
    expect(new Date(sent.sale_ends_at).getTime()).toBe(new Date('2099-01-02T03:04').getTime())
    expect(sent.trial_video_url).toBe('https://www.bilibili.com/video/BV1xx411c7mD')
  })
})
