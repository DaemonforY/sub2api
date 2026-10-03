import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CommunityView from '../CommunityView.vue'

const { works, moderate, ban, restricted, reports, setReport, comments, moderateComment, getSettings, saveSettings, showSuccess, showError } = vi.hoisted(() => ({
  works: vi.fn(),
  moderate: vi.fn(),
  ban: vi.fn(),
  restricted: vi.fn(),
  reports: vi.fn(),
  setReport: vi.fn(),
  comments: vi.fn(),
  moderateComment: vi.fn(),
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { community: { works, moderate, ban, restricted, reports, setReport, comments, moderateComment, getSettings, saveSettings } } }))
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
    for (const fn of [works, moderate, ban, restricted, reports, setReport, comments, moderateComment, getSettings, saveSettings]) fn.mockReset()
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

    getSettings.mockResolvedValue({ review_all: false, cloud_quota_mb: 200, cloud_subscriber_quota_mb: 2048 })
    saveSettings.mockResolvedValue({ review_all: true, cloud_quota_mb: 500, cloud_subscriber_quota_mb: 2048 })
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.settings')!.trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="community-settings"] input').setValue(true)
    await wrapper.get('[data-testid="cloud-quota"]').setValue(500)
    await wrapper.get('[data-testid="community-settings"] button').trigger('click')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith({ review_all: true, cloud_quota_mb: 500, cloud_subscriber_quota_mb: 2048 })
  })

  it('plays video works in the queue and saves the video review switch', async () => {
    works.mockResolvedValue([{ ...pendingWork, id: 8, title: '海边日落', review_flags: [], kind: 'video', video: { url: '/api/v1/community/media/v.mp4', mime_type: 'video/mp4', size_bytes: 1, duration_ms: 5000 } }])
    const wrapper = mount(CommunityView, { global: { stubs } })
    await flushPromises()
    const video = wrapper.get('[data-testid="community-work-video"]')
    expect(video.attributes('src')).toBe('/api/v1/community/media/v.mp4')
    expect(video.attributes('poster')).toBe(pendingWork.cover_thumb_url)
    expect(wrapper.get('[data-testid="community-work"]').text()).toContain('admin.community.video')

    getSettings.mockResolvedValue({ review_all: false, video_review: true, cloud_quota_mb: 200, cloud_subscriber_quota_mb: 2048 })
    saveSettings.mockResolvedValue({ review_all: false, video_review: false, cloud_quota_mb: 200, cloud_subscriber_quota_mb: 2048 })
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.settings')!.trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="video-review"]').setValue(false)
    await wrapper.get('[data-testid="community-settings"] button').trigger('click')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith(expect.objectContaining({ video_review: false }))
  })

  it('lists restricted authors and lifts a restriction', async () => {
    works.mockResolvedValue([])
    restricted.mockResolvedValue([{ user_id: 3, email: 'a@x.test', handle: 'xiaolin', display_name: '小林', works_count: 4, updated_at: '2026-10-02T00:00:00Z' }])
    ban.mockResolvedValue(undefined)
    const wrapper = mount(CommunityView, { global: { stubs } })
    await flushPromises()
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.restricted')!.trigger('click')
    await flushPromises()
    const row = wrapper.get('[data-testid="community-restricted-row"]')
    expect(row.text()).toContain('@xiaolin')
    await row.get('button').trigger('click')
    await flushPromises()
    expect(ban).toHaveBeenCalledWith(3, false)
    expect(wrapper.find('[data-testid="community-restricted-row"]').exists()).toBe(false)
    expect(showSuccess).toHaveBeenCalled()
  })

  it('moderates comments and saves the comment switches', async () => {
    works.mockResolvedValue([])
    comments.mockResolvedValue([
      { id: 31, work_id: 7, work_title: '海报', owner_id: 4, author: { handle: 'bee', display_name: 'Bee', avatar_url: '' }, body: '加我领资料', status: 'approved', report_count: 2, created_at: '2026-10-02T00:00:00Z' }
    ])
    moderateComment.mockResolvedValue(undefined)
    const wrapper = mount(CommunityView, { global: { stubs } })
    await flushPromises()
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.comments')!.trigger('click')
    await flushPromises()
    expect(comments).toHaveBeenCalledWith('pending', 1)
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.commentFilters.reported')!.trigger('click')
    await flushPromises()
    expect(comments).toHaveBeenLastCalledWith('reported', 1)
    const row = wrapper.get('[data-testid="community-comment"]')
    expect(row.text()).toContain('加我领资料')
    expect(row.findAll('button').map((b) => b.text())).not.toContain('admin.community.actions.approve')
    await row.findAll('button').find((b) => b.text() === 'admin.community.actions.hide')!.trigger('click')
    await flushPromises()
    expect(moderateComment).toHaveBeenCalledWith(31, 'hide')

    getSettings.mockResolvedValue({ review_all: false, comments_enabled: true, comments_review_all: false, cloud_quota_mb: 200, cloud_subscriber_quota_mb: 2048 })
    saveSettings.mockImplementation(async (v) => v)
    await wrapper.findAll('button').find((b) => b.text() === 'admin.community.tabs.settings')!.trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="comments-review-all"]').setValue(true)
    await wrapper.get('[data-testid="community-settings"]').findAll('button').find((b) => b.text() === 'common.save')!.trigger('click')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith(expect.objectContaining({ comments_enabled: true, comments_review_all: true }))
  })
})
