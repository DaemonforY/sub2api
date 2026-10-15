import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileWechatBindCard from '@/components/user/profile/ProfileWechatBindCard.vue'
import type { User } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function user(overrides: Partial<User> = {}): User {
  return {
    id: 7,
    username: 'alice',
    email: 'alice@qq.com',
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

function mountCard(u: User | null, wechatEnabled = true) {
  return mount(ProfileWechatBindCard, { props: { user: u, wechatEnabled }, global: { stubs: { Icon: true } } })
}

describe('ProfileWechatBindCard', () => {
  beforeEach(() => localStorage.clear())

  it('asks an email account without WeChat to bind it', async () => {
    const wrapper = mountCard(user())
    expect(wrapper.text()).toContain('profile.wechatBindReminder.title')
    await wrapper.get('[data-testid="profile-wechat-bind-reminder-bind"]').trigger('click')
    expect(wrapper.emitted('bind')).toHaveLength(1)
  })

  it('stays away once WeChat is bound, for WeChat sign-ups, and when WeChat binding is unavailable', () => {
    expect(mountCard(user({ wechat_bound: true })).find('[data-testid="profile-wechat-bind-reminder"]').exists()).toBe(false)
    expect(
      mountCard(user({ email: 'wechat-o0pO06abc@wechat-connect.invalid' })).find('[data-testid="profile-wechat-bind-reminder"]').exists()
    ).toBe(false)
    expect(mountCard(user(), false).find('[data-testid="profile-wechat-bind-reminder"]').exists()).toBe(false)
  })

  it('remembers a dismissal per user', async () => {
    const wrapper = mountCard(user())
    await wrapper.get('[data-testid="profile-wechat-bind-reminder-dismiss"]').trigger('click')
    expect(wrapper.find('[data-testid="profile-wechat-bind-reminder"]').exists()).toBe(false)
    expect(mountCard(user()).find('[data-testid="profile-wechat-bind-reminder"]').exists()).toBe(false)
    expect(mountCard(user({ id: 8 })).find('[data-testid="profile-wechat-bind-reminder"]').exists()).toBe(true)
  })
})
