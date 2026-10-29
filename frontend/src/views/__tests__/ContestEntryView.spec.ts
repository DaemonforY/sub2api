import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ContestEntryView from '../ContestEntryView.vue'

const { get, getEntry, leaderboard, vote, showSuccess, showError, replace, route } = vi.hoisted(() => ({
  get: vi.fn(),
  getEntry: vi.fn(),
  leaderboard: vi.fn(),
  vote: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  replace: vi.fn(),
  route: { params: { id: '5', entryId: '9' }, fullPath: '/contests/5/e/9', query: {} as Record<string, string> }
}))

vi.mock('@/api/contests', () => ({ contestsAPI: { get, getEntry, leaderboard, vote, unvote: vi.fn() } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))
vi.mock('@/utils/analytics', () => ({ track: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  PlazaNavBar: true,
  RouterLink: { template: '<a><slot /></a>' },
  BaseDialog: true,
  ShareCardDialog: { props: ['show', 'input'], template: '<div v-if="show" data-testid="share-dialog-stub">{{ input.pitch("L") }}</div>' }
}

const now = Date.now()
const contest = {
  id: 5, title: '11 月画图赛', description: '', rules: '', cover_image: '', status: 'published', phase: 'submitting_voting',
  submission_start_at: new Date(now - 3600e3).toISOString(), submission_end_at: new Date(now + 3600e3).toISOString(),
  voting_start_at: new Date(now - 3600e3).toISOString(), voting_end_at: new Date(now + 7200e3).toISOString(),
  max_entries_per_user: 3, votes_per_user: 5, allow_self_vote: false, require_review: true, min_account_age_hours: 0,
  prizes: [{ rank_from: 1, rank_to: 1, type: 'custom', amount: 0, label: '¥120 月度订阅' }], entry_count: 1
}
const signedIn = { logged_in: true, votes_used: 0, votes_left: 5, voted_entry_ids: [], my_entries: [], entries_left: 3, can_submit: true, can_vote: true }
const signedOut = { logged_in: false, votes_used: 0, votes_left: 0, voted_entry_ids: [], my_entries: [], entries_left: 0, can_submit: false, can_vote: false }
const entry = (extra = {}) => ({
  id: 9, contest_id: 5, user_id: 2, title: '月球上的猫', description: '', prompt: '', status: 'approved', review_note: '', vote_count: 3,
  created_at: '', updated_at: '', image_url: '/api/v1/contest-images/a.png', author_name: '小林', voted_by_me: false, is_mine: false, ...extra
})

describe('ContestEntryView', () => {
  beforeEach(() => {
    for (const fn of [get, getEntry, leaderboard, vote, showSuccess, showError, replace]) fn.mockReset()
    route.query = {}
    localStorage.clear()
    leaderboard.mockResolvedValue([{ rank: 2, entry_id: 9, title: '', author_name: '', image_url: '', votes: 3, final: false }])
  })

  it('shows the entry with its live rank; a signed-out vote is remembered and leads to sign in', async () => {
    get.mockResolvedValue({ contest, viewer: signedOut })
    getEntry.mockResolvedValue(entry())
    const assign = vi.fn()
    vi.stubGlobal('location', { ...window.location, assign, origin: 'https://hivegpt.cn' })
    const wrapper = mount(ContestEntryView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.text()).toContain('月球上的猫')
    expect(wrapper.text()).toContain('contests.entry.rank')
    expect(wrapper.get('[data-testid="entry-votes"]').text()).toBe('3')
    await wrapper.get('[data-testid="entry-vote"]').trigger('click')
    expect(JSON.parse(localStorage.getItem('contest_vote_intent')!)).toMatchObject({ contestId: 5, entryId: 9 })
    expect(assign).toHaveBeenCalledWith('/login?redirect=%2Fcontests%2F5%2Fe%2F9')
    vi.unstubAllGlobals()
  })

  it('casts the remembered vote when the visitor comes back signed in', async () => {
    localStorage.setItem('contest_vote_intent', JSON.stringify({ contestId: 5, entryId: 9, at: Date.now() }))
    get.mockResolvedValueOnce({ contest, viewer: signedIn }).mockResolvedValue({ contest, viewer: { ...signedIn, votes_left: 4, voted_entry_ids: [9] } })
    getEntry.mockResolvedValueOnce(entry()).mockResolvedValue(entry({ voted_by_me: true, vote_count: 4 }))
    vote.mockResolvedValue(undefined)
    const wrapper = mount(ContestEntryView, { global: { stubs } })
    await flushPromises()
    expect(vote).toHaveBeenCalledWith(5, 9)
    expect(showSuccess).toHaveBeenCalledWith('contests.entry.votedAfterLogin · contests.entry.votesLeftAfter')
    expect(wrapper.get('[data-testid="entry-votes"]').text()).toBe('4')
    expect(localStorage.getItem('contest_vote_intent')).toBeNull()
  })

  it('does not vote again for an entry already voted for', async () => {
    localStorage.setItem('contest_vote_intent', JSON.stringify({ contestId: 5, entryId: 9, at: Date.now() }))
    get.mockResolvedValue({ contest, viewer: signedIn })
    getEntry.mockResolvedValue(entry({ voted_by_me: true }))
    mount(ContestEntryView, { global: { stubs } })
    await flushPromises()
    expect(vote).not.toHaveBeenCalled()
  })

  it('opens the 拉票 dialog for the author after submitting, with their own pitch', async () => {
    route.query = { share: '1' }
    get.mockResolvedValue({ contest, viewer: signedIn })
    getEntry.mockResolvedValue(entry({ is_mine: true, status: 'pending' }))
    const wrapper = mount(ContestEntryView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.find('[data-testid="entry-vote"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('contests.entry.pendingNote')
    const pitch = wrapper.get('[data-testid="share-dialog-stub"]').text()
    expect(pitch).toContain('「月球上的猫」')
    expect(pitch).not.toContain('作者是')
    expect(replace).toHaveBeenCalled()
  })

  it('shows a not-found card when the entry is gone', async () => {
    get.mockResolvedValue({ contest, viewer: signedOut })
    getEntry.mockRejectedValue(new Error('not found'))
    const wrapper = mount(ContestEntryView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.find('[data-testid="entry-not-found"]').exists()).toBe(true)
  })
})
