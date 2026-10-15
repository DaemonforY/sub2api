import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileEmailReminderCard from '@/components/user/profile/ProfileEmailReminderCard.vue'
import type { User } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function user(overrides: Partial<User> = {}): User {
  return {
    id: 9,
    username: '微信用户_ab12',
    email: 'wechat-o0pO06abc@wechat-connect.invalid',
    role: 'user',
    balance: 0,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-10-16T00:00:00Z',
    updated_at: '2026-10-16T00:00:00Z',
    ...overrides,
  } as User
}

describe('ProfileEmailReminderCard', () => {
  beforeEach(() => localStorage.clear())

  it('asks a placeholder-email user to add an email and leads to the bind form', async () => {
    const wrapper = mount(ProfileEmailReminderCard, { props: { user: user() }, global: { stubs: { Icon: true } } })
    expect(wrapper.text()).toContain('profile.emailReminder.title')
    expect(wrapper.get('[data-testid="profile-email-reminder-subtitle"]').text()).toBe('profile.emailReminder.subtitle')
    await wrapper.get('[data-testid="profile-email-reminder-bind"]').trigger('click')
    expect(wrapper.emitted('bind')).toHaveLength(1)
  })

  it('says reminders already arrive in WeChat for a bound follower', () => {
    const wrapper = mount(ProfileEmailReminderCard, {
      props: { user: user({ wechat_bound: true, wechat_mp_subscribed: true }) },
      global: { stubs: { Icon: true } },
    })
    expect(wrapper.get('[data-testid="profile-email-reminder-subtitle"]').text()).toBe(
      'profile.emailReminder.subtitleWeChat',
    )
  })

  it('is hidden for real or bound emails', () => {
    expect(
      mount(ProfileEmailReminderCard, { props: { user: user({ email: 'a@example.com' }) }, global: { stubs: { Icon: true } } })
        .find('[data-testid="profile-email-reminder"]').exists(),
    ).toBe(false)
    expect(
      mount(ProfileEmailReminderCard, { props: { user: user({ email_bound: true }) }, global: { stubs: { Icon: true } } })
        .find('[data-testid="profile-email-reminder"]').exists(),
    ).toBe(false)
  })

  it('stays dismissed for this user', async () => {
    const wrapper = mount(ProfileEmailReminderCard, { props: { user: user() }, global: { stubs: { Icon: true } } })
    await wrapper.get('[data-testid="profile-email-reminder-dismiss"]').trigger('click')
    expect(wrapper.find('[data-testid="profile-email-reminder"]').exists()).toBe(false)

    const again = mount(ProfileEmailReminderCard, { props: { user: user() }, global: { stubs: { Icon: true } } })
    expect(again.find('[data-testid="profile-email-reminder"]').exists()).toBe(false)
    const other = mount(ProfileEmailReminderCard, { props: { user: user({ id: 10 }) }, global: { stubs: { Icon: true } } })
    expect(other.find('[data-testid="profile-email-reminder"]').exists()).toBe(true)
  })
})
