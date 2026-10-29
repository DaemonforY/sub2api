import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ContestAutoCard from '../ContestAutoCard.vue'

const { getAuto, saveAuto, showSuccess, showError } = vi.hoisted(() => ({
  getAuto: vi.fn(),
  saveAuto: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { contests: { getAuto, saveAuto } } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  ConfirmDialog: { props: ['show'], emits: ['confirm', 'cancel'], template: '<button v-if="show" data-testid="confirm" @click="$emit(\'confirm\')">ok</button>' },
  Select: true,
  Icon: true
}

const template = {
  title_template: '{month} 月 AI 画图赛', description: '', rules: '', cover_image: '', submission_end_day: 22, voting_end_time: '20:00', lead_days: 3, remind_days: 1,
  max_entries_per_user: 3, votes_per_user: 5, allow_self_vote: false, require_review: true, min_account_age_hours: 0, one_prize_per_user: true,
  min_votes_for_prize: 5, prizes: [{ rank_from: 1, rank_to: 1, type: 'custom', amount: 0, label: '¥120 月度订阅（1 个月）' }]
}
const month = (period: string, status: string, extra = {}) => ({
  period, title: `${period} 赛`, submission_start_at: '2026-11-01T00:00:00+08:00', submission_end_at: '2026-11-23T00:00:00+08:00',
  voting_end_at: '2026-11-30T20:00:00+08:00', create_at: '2026-10-29T00:00:00+08:00', remind_at: '2026-10-28T00:00:00+08:00', reminded: false, status, ...extra
})

describe('ContestAutoCard', () => {
  beforeEach(() => {
    for (const fn of [getAuto, saveAuto, showSuccess, showError]) fn.mockReset()
  })

  it('starts the schedule and tells the list to reload when a month was published', async () => {
    getAuto.mockResolvedValue({ enabled: false, template, months: [month('2026-10', 'missed'), month('2026-11', 'paused')] })
    saveAuto.mockResolvedValue({ enabled: true, template, months: [month('2026-10', 'missed'), month('2026-11', 'created', { contest_id: 8, phase: 'upcoming' })] })
    const wrapper = mount(ContestAutoCard, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="contest-auto-state"]').text()).toBe('admin.contests.auto.stopped')
    await wrapper.get('[data-testid="contest-auto-toggle"]').trigger('click')
    await wrapper.get('[data-testid="confirm"]').trigger('click')
    await flushPromises()
    expect(saveAuto).toHaveBeenCalledWith(true, expect.objectContaining({ title_template: '{month} 月 AI 画图赛', lead_days: 3 }))
    expect(wrapper.emitted('changed')).toHaveLength(1)
    expect(wrapper.get('[data-testid="contest-auto-state"]').text()).toBe('admin.contests.auto.running')
    expect(wrapper.find('a[href="/admin/contests/8"]').exists()).toBe(true)
  })

  it('stops the schedule without touching the template', async () => {
    getAuto.mockResolvedValue({ enabled: true, template, months: [month('2026-11', 'created', { contest_id: 8 }), month('2026-12', 'scheduled')] })
    saveAuto.mockResolvedValue({ enabled: false, template, months: [month('2026-11', 'created', { contest_id: 8 }), month('2026-12', 'paused')] })
    const wrapper = mount(ContestAutoCard, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="contest-auto-remind"]').text()).toContain('admin.contests.auto.remindsAt')
    await wrapper.get('[data-testid="contest-auto-toggle"]').trigger('click')
    await wrapper.get('[data-testid="confirm"]').trigger('click')
    await flushPromises()
    expect(saveAuto).toHaveBeenCalledWith(false, template)
    expect(wrapper.emitted('changed')).toBeUndefined()
    expect(showSuccess).toHaveBeenCalledWith('admin.contests.auto.stoppedToast')
  })

  it('saves an edited template and keeps the dialog open on a server error', async () => {
    getAuto.mockResolvedValue({ enabled: true, template, months: [] })
    saveAuto.mockRejectedValueOnce(new Error('点赞截止时间格式应为 HH:MM')).mockResolvedValueOnce({ enabled: true, template: { ...template, title_template: '{year} 年 {month} 月赛' }, months: [] })
    const wrapper = mount(ContestAutoCard, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="contest-auto-edit"]').trigger('click')
    await wrapper.get('[data-testid="contest-auto-title"]').setValue('{year} 年 {month} 月赛')
    await wrapper.get('[data-testid="contest-auto-save"]').trigger('click')
    await flushPromises()
    expect(showError).toHaveBeenCalledWith('点赞截止时间格式应为 HH:MM')
    expect(wrapper.find('[data-testid="contest-auto-save"]').exists()).toBe(true)
    await wrapper.get('[data-testid="contest-auto-save"]').trigger('click')
    await flushPromises()
    expect(saveAuto).toHaveBeenLastCalledWith(true, expect.objectContaining({ title_template: '{year} 年 {month} 月赛' }))
    expect(wrapper.find('[data-testid="contest-auto-save"]').exists()).toBe(false)
  })
})
