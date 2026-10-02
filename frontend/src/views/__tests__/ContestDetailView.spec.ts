import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ContestDetailView from '../ContestDetailView.vue'

const { get, listEntries, leaderboard, submitEntry, submitWorkEntry, myWorks, showSuccess, showError } = vi.hoisted(() => ({
  get: vi.fn(),
  listEntries: vi.fn(),
  leaderboard: vi.fn(),
  submitEntry: vi.fn(),
  submitWorkEntry: vi.fn(),
  myWorks: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/contests', () => ({ contestsAPI: { get, listEntries, leaderboard, submitEntry, submitWorkEntry, withdrawEntry: vi.fn(), vote: vi.fn(), unvote: vi.fn() } }))
vi.mock('@/api/community', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/api/community')>()), myWorks }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { id: '5' }, fullPath: '/contests/5' }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  PlazaNavBar: true,
  RouterLink: { template: '<a><slot /></a>' },
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  ConfirmDialog: true
}

const now = Date.now()
const contest = {
  id: 5, title: '国庆赛', description: '', rules: '', cover_image: '', status: 'published', phase: 'submitting',
  submission_start_at: new Date(now - 3600e3).toISOString(), submission_end_at: new Date(now + 3600e3).toISOString(),
  voting_start_at: new Date(now + 3600e3).toISOString(), voting_end_at: new Date(now + 7200e3).toISOString(),
  max_entries_per_user: 2, votes_per_user: 3, allow_self_vote: false, require_review: false, prizes: [], entry_count: 1
}
const viewer = { logged_in: true, votes_used: 0, votes_left: 3, voted_entry_ids: [], my_entries: [], entries_left: 2, can_submit: true, can_vote: false }
const work = (id: number, extra = {}) => ({
  id, title: `作品${id}`, description: `说明${id}`, prompt: '', author: { handle: 'me', display_name: '', avatar_url: '' }, cover_url: '', cover_thumb_url: `/m/${id}_t.jpg`,
  cover_width: 640, cover_height: 640, image_count: 1, like_count: 0, remix_count: 0, visibility: 'public', status: 'approved', ...extra
})

describe('ContestDetailView entering a canvas work', () => {
  beforeEach(() => {
    for (const fn of [get, listEntries, leaderboard, submitEntry, submitWorkEntry, myWorks, showSuccess, showError]) fn.mockReset()
    get.mockResolvedValue({ contest, viewer })
    listEntries.mockResolvedValue({ items: [{ id: 9, contest_id: 5, user_id: 1, title: '已投', description: '', prompt: '', status: 'approved', review_note: '', vote_count: 0, created_at: '', updated_at: '', image_url: '/x.png', author_name: 'a', voted_by_me: false, is_mine: false, work_id: 33 }], total: 1, page: 1, page_size: 24, pages: 1 })
    leaderboard.mockResolvedValue([])
  })

  it('links entries made from works to the canvas', async () => {
    const wrapper = mount(ContestDetailView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="contest-entry-work"]').attributes('href')).toMatch(/\/w\/33\?utm_source=hivegpt&utm_medium=contest-entry/)
  })

  it('picks an own work, fills the title and submits it; private and pending works are disabled', async () => {
    myWorks.mockResolvedValue([work(1), work(2, { visibility: 'private' }), work(3, { status: 'pending' })])
    submitWorkEntry.mockResolvedValue({ id: 10, status: 'approved' })
    const wrapper = mount(ContestDetailView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="contest-submit-open"]').trigger('click')
    await wrapper.get('[data-testid="contest-source-work"]').trigger('click')
    await flushPromises()
    const options = wrapper.findAll('[data-testid="contest-work-option"]')
    expect(options).toHaveLength(3)
    expect(options[1].attributes('disabled')).toBeDefined()
    expect(options[1].text()).toContain('contests.submit.blocked.private')
    expect(options[2].text()).toContain('contests.submit.blocked.review')
    await options[0].trigger('click')
    expect((wrapper.get('input[type="text"]').element as HTMLInputElement).value).toBe('作品1')
    await wrapper.findAll('button').find((b) => b.text() === 'contests.submit.submit')!.trigger('click')
    await flushPromises()
    expect(submitWorkEntry).toHaveBeenCalledWith(5, { work_id: 1, title: '作品1', description: '说明1' })
    expect(submitEntry).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('contests.submit.success')
  })

  it('asks to pick a work before submitting', async () => {
    myWorks.mockResolvedValue([])
    const wrapper = mount(ContestDetailView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="contest-submit-open"]').trigger('click')
    await wrapper.get('[data-testid="contest-source-work"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('contests.submit.noWorks')
    await wrapper.findAll('button').find((b) => b.text() === 'contests.submit.submit')!.trigger('click')
    expect(wrapper.text()).toContain('contests.submit.workRequired')
    expect(submitWorkEntry).not.toHaveBeenCalled()
  })
})
