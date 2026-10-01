import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SitesView from '../SitesView.vue'

const { getSettings, saveSettings, list, setStatus, remove, reports, setReportStatus, reviews, review, showSuccess, showError } = vi.hoisted(() => ({
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  list: vi.fn(),
  setStatus: vi.fn(),
  remove: vi.fn(),
  reports: vi.fn(),
  setReportStatus: vi.fn(),
  reviews: vi.fn(),
  review: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { sites: { getSettings, saveSettings, list, setStatus, remove, reports, setReportStatus, reviews, review } } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' }, BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' } }

describe('admin SitesView', () => {
  it('lists sites, takes one down with a reason and handles reports', async () => {
    getSettings.mockResolvedValue({ domain: 's.example.test', enabled: true, max_per_user: 3, max_mb: 20, max_files: 500, free_per_user: 1, extra_price: 5, grace_days: 7, retention_days: 30, review_all: false, review_base_url: '', review_model: '', review_api_key_configured: false })
    list.mockResolvedValue({ items: [{ id: 9, user_id: 2, user_email: 'u@x.test', name: 'abc1234', title: '活动页', status: 'active', status_reason: '', version: 1, pending_version: 0, has_password: false, views_7d: 0, views_total: 0, size_bytes: 2048, file_count: 3, paid: false, created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z', url: 'https://abc1234.s.example.test' }], total: 1 })
    reports.mockResolvedValue({ items: [{ id: 4, site_id: 9, site_name: 'abc1234', site_status: 'active', owner_email: 'u@x.test', reason: 'phishing', detail: '仿冒登录', contact: '', reporter_ip: '1.2.3.4', status: 'open', created_at: '2026-10-01T00:00:00Z' }], total: 1 })
    reviews.mockResolvedValue({ items: [{ site_id: 9, site_name: 'abc1234', title: '活动页', owner_email: 'u@x.test', site_status: 'pending', version: 2, size_bytes: 10, file_count: 1, reason: '页面含有需要人工确认的内容：密码输入框', flags: ['password_field'], excerpt: '[index.html] 账户登录', created_at: '2026-10-01T00:00:00Z', preview_url: 'https://abc1234.s.example.test/?__hg_preview=x' }], total: 1 })
    review.mockResolvedValue(undefined)
    setStatus.mockResolvedValue(undefined)
    setReportStatus.mockResolvedValue(undefined)

    const wrapper = mount(SitesView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.text()).toContain('https://abc1234.s.example.test')
    expect(wrapper.text()).toContain('u@x.test')

    await wrapper.findAll('button').find((b) => b.text() === 'admin.sites.actions.disable')!.trigger('click')
    await wrapper.find('input[placeholder="admin.sites.disablePlaceholder"]').setValue('钓鱼页面')
    await wrapper.findAll('button.btn-danger').find((b) => b.text() === 'admin.sites.actions.disable')!.trigger('click')
    await flushPromises()
    expect(setStatus).toHaveBeenCalledWith(9, 'disabled', '钓鱼页面')

    await wrapper.findAll('button').find((b) => b.text().startsWith('admin.sites.tabs.reports'))!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('仿冒登录')
    await wrapper.findAll('button').find((b) => b.text() === 'admin.sites.reports.dismiss')!.trigger('click')
    await flushPromises()
    expect(setReportStatus).toHaveBeenCalledWith(4, 'dismissed')

    await wrapper.findAll('button').find((b) => b.text().startsWith('admin.sites.tabs.reviews'))!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('[index.html] 账户登录')
    expect(wrapper.find('a[href="https://abc1234.s.example.test/?__hg_preview=x"]').exists()).toBe(true)
    await wrapper.findAll('button').find((b) => b.text() === 'admin.sites.reviews.reject')!.trigger('click')
    await wrapper.find('input[placeholder="admin.sites.reviews.rejectPlaceholder"]').setValue('仿冒登录')
    await wrapper.findAll('button.btn-danger').filter((b) => b.text() === 'admin.sites.reviews.reject').at(-1)!.trigger('click')
    await flushPromises()
    expect(review).toHaveBeenCalledWith(9, 2, 'reject', '仿冒登录')
    expect(showError).not.toHaveBeenCalled()
  })
})
