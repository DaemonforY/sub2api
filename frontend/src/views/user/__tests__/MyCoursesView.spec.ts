import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import MyCoursesView from '../MyCoursesView.vue'

const { myCourses, getDelivery, showError, copyToClipboard } = vi.hoisted(() => ({
  myCourses: vi.fn(),
  getDelivery: vi.fn(),
  showError: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/courses', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/api/courses')>()), myCourses, getDelivery }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' }, RouterLink: { template: '<a><slot /></a>' } }
const item = (extra: Record<string, unknown> = {}, enrollment: Record<string, unknown> = {}) => ({
  course: { id: 5, slug: 'ai-agent', title: 'AI Agent 实战', cover_url: '', lesson_count: 2, outline: [] },
  enrollment: { user_id: 1, course_id: 5, source: 'purchase', status: 'active', view_count: 0, seen_version: 1, refunded: false, created_at: '2026-10-04T00:00:00Z', recent_ips: 0, ...enrollment },
  updated: false,
  ...extra
})

describe('MyCoursesView', () => {
  beforeEach(() => {
    for (const fn of [myCourses, getDelivery, showError, copyToClipboard]) fn.mockReset()
  })

  it('shows the netdisk link and code on demand and copies them', async () => {
    myCourses.mockResolvedValue([item({ updated: true })])
    getDelivery.mockResolvedValue({ version: 2, link: 'https://pan.baidu.com/s/1abc', code: 'ab12', password: '', note: '先看 README', updated_at: '2026-10-04T00:00:00Z' })
    const wrapper = mount(MyCoursesView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.find('[data-testid="my-course-updated"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="my-course-delivery"]').exists()).toBe(false)

    await wrapper.get('[data-testid="my-course-get"]').trigger('click')
    await flushPromises()
    expect(getDelivery).toHaveBeenCalledWith(5)
    const delivery = wrapper.get('[data-testid="my-course-delivery"]')
    expect(delivery.text()).toContain('https://pan.baidu.com/s/1abc')
    expect(delivery.text()).toContain('ab12')
    expect(delivery.text()).toContain('先看 README')
    expect(wrapper.find('[data-testid="my-course-updated"]').exists()).toBe(false)

    await delivery.findAll('button')[1].trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith('ab12', 'courses.mine.copied')
  })

  it('offers nothing to fetch for refunded courses', async () => {
    myCourses.mockResolvedValue([item({}, { refunded: true })])
    const wrapper = mount(MyCoursesView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.text()).toContain('courses.mine.refundedHint')
    expect(wrapper.find('[data-testid="my-course-get"]').exists()).toBe(false)
  })
})
