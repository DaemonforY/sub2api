import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import RequireNicknameDialog from '@/components/user/RequireNicknameDialog.vue'
import { isWeChatFallbackUsername } from '@/utils/userDisplay'

const { updateProfileMock, showSuccessMock, showErrorMock, authState } = vi.hoisted(() => ({
  updateProfileMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  authState: { user: null as { username: string; role: string } | null, refreshUser: vi.fn() },
}))

vi.mock('@/api', () => ({
  userAPI: { updateProfile: updateProfileMock },
}))
vi.mock('@/api/auth', () => ({
  buildWeChatProfileSyncURL: (redirect: string) => `/api/v1/auth/oauth/wechat/sync-profile/start?redirect=${redirect}`,
  detectWeChatLoginTerminal: () => 'pc',
}))
vi.mock('@/stores', async () => {
  const { reactive: makeReactive } = await import('vue')
  const store = makeReactive(authState)
  return {
    useAuthStore: () => store,
    useAppStore: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
  }
})
vi.mock('vue-router', () => ({
  useRoute: () => reactive({ fullPath: '/keys' }),
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

enableAutoUnmount(afterEach)

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show" class="dialog"><slot /></div>',
}

async function mountWith(user: { username: string; role: string } | null) {
  const { useAuthStore } = await import('@/stores')
  const store = useAuthStore() as unknown as typeof authState
  store.user = user
  const wrapper = mount(RequireNicknameDialog, {
    global: { stubs: { BaseDialog: BaseDialogStub } },
  })
  return { wrapper, store }
}

describe('isWeChatFallbackUsername', () => {
  it.each([
    ['微信用户', true],
    ['微信用户_rb1a', true],
    [' 微信用户_9x ', true],
    ['微信用户小明', false],
    ['微信用户_ABCD', false],
    ['阳', false],
    ['', false],
  ])('%j → %s', (name, expected) => {
    expect(isWeChatFallbackUsername(name)).toBe(expected)
  })
})

describe('RequireNicknameDialog', () => {
  beforeEach(() => {
    updateProfileMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
  })

  it('stays hidden for real nicknames and for admins', async () => {
    expect((await mountWith({ username: '阳', role: 'user' })).wrapper.find('.dialog').exists()).toBe(false)
    expect((await mountWith({ username: '微信用户_rb1a', role: 'admin' })).wrapper.find('.dialog').exists()).toBe(false)
    expect((await mountWith(null)).wrapper.find('.dialog').exists()).toBe(false)
  })

  it('rejects an empty or default-looking nickname without calling the API', async () => {
    const { wrapper } = await mountWith({ username: '微信用户_rb1a', role: 'user' })
    expect(wrapper.find('.dialog').exists()).toBe(true)

    await wrapper.get('form').trigger('submit.prevent')
    expect(wrapper.text()).toContain('profile.usernameRequired')

    await wrapper.get('#require-nickname').setValue('微信用户_abcd')
    await wrapper.get('form').trigger('submit.prevent')
    expect(wrapper.text()).toContain('profile.requireNickname.stillDefault')
    expect(updateProfileMock).not.toHaveBeenCalled()
  })

  it('saves the nickname and closes once the name is no longer the default', async () => {
    const { wrapper } = await mountWith({ username: '微信用户_rb1a', role: 'user' })
    updateProfileMock.mockResolvedValue({ username: '小王', role: 'user' })

    await wrapper.get('#require-nickname').setValue('  小王 ')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(updateProfileMock).toHaveBeenCalledWith({ username: '小王' })
    expect(wrapper.find('.dialog').exists()).toBe(false)
    expect(showSuccessMock).toHaveBeenCalledTimes(1)
  })

  it('keeps the dialog open and shows the server error when saving fails', async () => {
    const { wrapper } = await mountWith({ username: '微信用户_rb1a', role: 'user' })
    updateProfileMock.mockRejectedValue({ message: 'boom' })

    await wrapper.get('#require-nickname').setValue('小王')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.find('.dialog').exists()).toBe(true)
    expect(wrapper.text()).toContain('boom')
  })
})
