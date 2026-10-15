import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import WechatSyncedView from '@/views/auth/WechatSyncedView.vue'

const routeState = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
const authState = vi.hoisted(() => ({ isAuthenticated: false, refreshUser: vi.fn() }))

vi.mock('vue-router', () => ({ useRoute: () => routeState }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => (params?.name ? `${key}:${params.name}` : key),
  }),
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))

function mountView(query: Record<string, unknown>) {
  routeState.query = query
  return mount(WechatSyncedView)
}

describe('WechatSyncedView', () => {
  beforeEach(() => {
    authState.isAuthenticated = false
    authState.refreshUser.mockReset().mockResolvedValue({})
  })

  it('shows the new nickname and continues to a safe redirect', async () => {
    const wrapper = mountView({ status: 'ok', name: '小蜂', redirect: '/profile' })
    await flushPromises()

    expect(wrapper.get('[data-testid="wechat-synced-title"]').text()).toBe('auth.wechatSynced.okTitle:小蜂')
    expect(wrapper.text()).toContain('auth.wechatSynced.okHint')
    expect(wrapper.get('[data-testid="wechat-synced-continue"]').attributes('href')).toBe('/profile')
    // Not signed in here (usually inside WeChat after a PC sign-in): nothing to refresh.
    expect(authState.refreshUser).not.toHaveBeenCalled()
  })

  it('refreshes the signed-in user after a successful sync', async () => {
    authState.isAuthenticated = true
    mountView({ status: 'ok', name: 'Bee' })
    await flushPromises()
    expect(authState.refreshUser).toHaveBeenCalledTimes(1)
  })

  it('tells a WeChat user without an account to sign in with WeChat first', async () => {
    authState.isAuthenticated = true
    const wrapper = mountView({ status: 'not_bound', redirect: '/dashboard?tab=1' })
    await flushPromises()

    expect(wrapper.get('[data-testid="wechat-synced-title"]').text()).toBe('auth.wechatSynced.notBoundTitle')
    expect(wrapper.get('[data-testid="wechat-synced-continue"]').attributes('href')).toBe('/dashboard?tab=1')
    expect(authState.refreshUser).not.toHaveBeenCalled()
  })

  it.each([['error'], ['weird'], [undefined]])('shows the retry message for status %s', (status) => {
    const wrapper = mountView({ status })
    expect(wrapper.get('[data-testid="wechat-synced-title"]').text()).toBe('auth.wechatSynced.errorTitle')
    expect(wrapper.text()).toContain('auth.wechatSynced.errorHint')
    expect(wrapper.find('[data-testid="wechat-synced-continue"]').exists()).toBe(false)
  })

  it.each([
    ['https://evil.example/x'],
    ['//evil.example/x'],
    ['/\\evil.example'],
    ['javascript:alert(1)'],
    ['/auth/wechat/synced?status=ok'],
    ['/auth/wechat/synced/'],
  ])('shows no continue button for the unsafe or self redirect %s', (redirect) => {
    const wrapper = mountView({ status: 'ok', name: 'x', redirect })
    expect(wrapper.find('[data-testid="wechat-synced-continue"]').exists()).toBe(false)
  })
})
