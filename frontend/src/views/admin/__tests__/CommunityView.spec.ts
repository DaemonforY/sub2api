import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CommunityView from '../CommunityView.vue'

const { works, moderate, ban, reports, setReport, getSettings, saveSettings, showSuccess, showError } = vi.hoisted(() => ({
  works: vi.fn(),
  moderate: vi.fn(),
  ban: vi.fn(),
  reports: vi.fn(),
  setReport: vi.fn(),
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { community: { works, moderate, ban, reports, setReport, getSettings, saveSettings } } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' }, BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' } }

const pendingWork = {
  id: 7, owner_id: 3, author: { handle: 'xiaolin', display_name: '小林', avatar_url: '' }, title: '赌场海报', description: '', prompt: '百家乐', model: '', tags: [],
  visibility: 'public', status: 'pending', review_reason: '内容命中敏感词', review_flags: ['gambling:百家乐'], featured: false, cover_thumb_url: '/api/v1/community/media/x_t.jpg',
  image_count: 1, like_count: 0, created_at: '2026-10-02T00:00:00Z'
}

describe('admin CommunityView', () => {
  beforeEach(() => {
    for (const fn of [works, moderate, ban, reports, setReport, getSettings, saveSettings]) fn.mockReset()
    moderate.mockResolvedValue(undefined)
  })

  it('shows the review queue with flags and rejects with a reason', async () => {
    works.mockResolvedValue([pendingWork])
    const wrapper = mount(CommunityView, { global: { stubs } })
    await flushPromises()
    expect(works).toHaveBeenCalledWith('pending', 1)
    const card = wrapper.get('[data-testid="community-work"]')
    expect(card.text()).toContain('赌场海报')
    expect(card.text()).toContain('gambling:百家乐')

    const reject = card.findAll('button').find((b) => b.text() === 'admin.community.actions.reject')!
    await reject.trigger('click')
    await wrapper.get('[data-testid="community-reason"]').setValue('涉赌')
    works.mockResolvedValue([])
    await wrapper.get('[data-testid="community-reason-confirm"]').trigger('click')
    await flushPromises()
    expect(moderate).toHaveBeenCalledWith(7, 'reject', '涉赌')
    expect(wrapper.find('[data-testid="community-work"]').exists()).toBe(false)
  })

  it('approves directly and saves the review-all setting', async () => {
    works.mockResolvedValue([pendingWork])
    const wrapper = mount(CommunityView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="community-work"]').findAll('button').find((b) => b.text() === 'admin.community.actions.approve')!.trigger('click')
    await flushPromises()
    expect(moderate).toHaveBeenCalledWith(7, 'approve', '')

    getSettings.mockResolvedValue({ review_all: false })
    saveSettings.mockResolvedValue({ review_all: true })
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.settings')!.trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="community-settings"] input').setValue(true)
    await wrapper.get('[data-testid="community-settings"] button').trigger('click')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith(true)
  })
})
