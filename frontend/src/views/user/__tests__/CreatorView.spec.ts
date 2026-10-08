import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CreatorView from '../CreatorView.vue'

const api = vi.hoisted(() => ({
  home: vi.fn(),
  apply: vi.fn(),
  courses: vi.fn(),
  get: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  uploadCover: vi.fn(),
  uploadImage: vi.fn(),
  submit: vi.fn(),
  setOnSale: vi.fn(),
  deliveries: vi.fn(),
  saveDelivery: vi.fn(),
  students: vi.fn(),
  sales: vi.fn(),
  withdraw: vi.fn(),
  cancelWithdraw: vi.fn()
}))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/creator', () => ({ default: api }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' } }
const rules = { enabled: true, commission_percent: 20, settle_days: 7, withdraw_min_cny: 50, max_courses: 20, withdrawals: [] }
const approved = { user_id: 3, status: 'approved', display_name: '王老师', bio: '', contact: 'wx', plan: 'p', admin_note: '', created_at: '', course_count: 1, on_sale_count: 0, pending_count: 0, commission_percent: null }
const balance = { orders: 2, gross: 200, net: 160, frozen: 40, paid: 0, pending: 0, available: 120 }
const course = {
  id: 7, slug: 'ai-office', title: 'AI 办公', subtitle: '', category: '', cover_url: '', price: 99, original_price: 0, outline: [], status: 'draft',
  sort_order: 0, sale_price: 0, edu_discount: true, trial_video_url: '', current_price: 99, sale_active: false, edu_applied: false, lesson_count: 0,
  student_count: 0, owned: false, created_at: '', updated_at: '', delivery_version: 1, review_status: 'draft', owner_id: 3,
  draft: { title: 'AI 办公（草稿）', subtitle: '', category: '', price: 129, original_price: 0, intro_md: '', outline: [], trial_md: '', faq_md: '', sale_price: 0, sale_ends_at: null, edu_discount: true, trial_video_url: '' }
}

describe('CreatorView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    api.deliveries.mockResolvedValue([{ version: 1, link: 'https://pan.baidu.com/s/1x', code: '', password: '', note: '', updated_at: '' }])
    api.sales.mockResolvedValue({ items: [], total: 0 })
  })

  it('shows the rules and sends an application', async () => {
    api.home.mockResolvedValueOnce({ ...rules, creator: null }).mockResolvedValueOnce({ ...rules, creator: { ...approved, status: 'pending' } })
    api.apply.mockResolvedValue({})
    const wrapper = mount(CreatorView, { global: { stubs } })
    await flushPromises()

    expect(wrapper.find('[data-testid="creator-intro"]').exists()).toBe(true)
    await wrapper.get('[data-testid="creator-apply-name"]').setValue('王老师')
    await wrapper.get('[data-testid="creator-apply-contact"]').setValue('wx: wang')
    await wrapper.get('[data-testid="creator-apply-plan"]').setValue('AI 办公课')
    await wrapper.get('[data-testid="creator-apply-submit"]').trigger('click')
    await flushPromises()
    expect(api.apply).toHaveBeenCalledWith({ display_name: '王老师', bio: '', contact: 'wx: wang', plan: 'AI 办公课' })
    expect(wrapper.find('[data-testid="creator-pending"]').exists()).toBe(true)
  })

  it('creates a course with its netdisk link in one go', async () => {
    api.home.mockResolvedValue({ ...rules, creator: approved, balance })
    api.courses.mockResolvedValue([])
    api.create.mockResolvedValue({ ...course, id: 8 })
    api.get.mockResolvedValue({ ...course, id: 8 })
    const wrapper = mount(CreatorView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creator-new"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="creator-title"]').setValue('AI 办公')
    await wrapper.get('[data-testid="creator-slug"]').setValue('ai-office')
    await wrapper.get('[data-testid="creator-price"]').setValue(100)
    expect(wrapper.get('[data-testid="creator-share-hint"]').exists()).toBe(true)
    await wrapper.get('[data-testid="creator-new-link"]').setValue('https://pan.baidu.com/s/1x')
    await wrapper.get('[data-testid="creator-save"]').trigger('click')
    await flushPromises()

    expect(api.create).toHaveBeenCalledWith(expect.objectContaining({ title: 'AI 办公', slug: 'ai-office', price: 100 }))
    expect(api.create.mock.calls[0][0]).not.toHaveProperty('status')
    expect(api.saveDelivery).toHaveBeenCalledWith(8, { link: 'https://pan.baidu.com/s/1x', code: '', password: '', note: '', notify: false })
    expect(wrapper.find('[data-testid="creator-review-state"]').exists()).toBe(true)
  })

  it('edits the draft (not the live version) and submits it for review', async () => {
    api.home.mockResolvedValue({ ...rules, creator: approved, balance })
    api.courses.mockResolvedValue([course])
    api.get.mockResolvedValue(course)
    api.update.mockResolvedValue(course)
    api.submit.mockResolvedValue({ ...course, review_status: 'pending' })
    const wrapper = mount(CreatorView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creator-courses"] button').trigger('click')
    await flushPromises()
    expect((wrapper.get('[data-testid="creator-title"]').element as HTMLInputElement).value).toBe('AI 办公（草稿）')
    expect((wrapper.get('[data-testid="creator-slug"]').element as HTMLInputElement).disabled).toBe(true)
    await wrapper.get('[data-testid="creator-submit"]').trigger('click')
    await flushPromises()
    expect(api.update).toHaveBeenCalledWith(7, expect.objectContaining({ title: 'AI 办公（草稿）', price: 129 }))
    expect(api.submit).toHaveBeenCalledWith(7)
    expect((wrapper.get('[data-testid="creator-submit"]').element as HTMLButtonElement).disabled).toBe(true)
  })

  it('withdraws the available balance to the last payee', async () => {
    api.home.mockResolvedValue({ ...rules, creator: approved, balance, last_method: 'wechat', last_account: 'wx-123', last_real_name: '王某' })
    api.courses.mockResolvedValue([])
    api.withdraw.mockResolvedValue({})
    const wrapper = mount(CreatorView, { global: { stubs } })
    await flushPromises()

    await wrapper.get('[data-testid="creator-tab-earnings"]').trigger('click')
    await flushPromises()
    expect((wrapper.get('[data-testid="creator-withdraw-account"]').element as HTMLInputElement).value).toBe('wx-123')
    await wrapper.get('[data-testid="creator-withdraw-submit"]').trigger('click')
    await flushPromises()
    expect(api.withdraw).toHaveBeenCalledWith({ cny_amount: 120, method: 'wechat', account: 'wx-123', real_name: '王某', note: '' })
  })
})
