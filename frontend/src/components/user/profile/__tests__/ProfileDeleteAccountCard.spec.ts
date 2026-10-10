import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileDeleteAccountCard from '@/components/user/profile/ProfileDeleteAccountCard.vue'

const { deleteAccountMock, logoutMock, replaceMock, showErrorMock, authState } = vi.hoisted(() => ({
  deleteAccountMock: vi.fn(),
  logoutMock: vi.fn(),
  replaceMock: vi.fn(),
  showErrorMock: vi.fn(),
  authState: { user: { role: 'user', balance: 3.5 } as { role: string; balance: number } }
}))

vi.mock('@/api/user', () => ({ deleteAccount: deleteAccountMock }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ ...authState, logout: logoutMock }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: showErrorMock }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ replace: replaceMock }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, p?: Record<string, string>) => (p ? `${key}:${JSON.stringify(p)}` : key), locale: { value: 'zh' } })
}))

describe('ProfileDeleteAccountCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.user = { role: 'user', balance: 3.5 }
  })

  it('needs the phrase, warns about the balance, then deletes and signs out', async () => {
    const w = mount(ProfileDeleteAccountCard)
    await w.get('[data-testid="delete-account-open"]').trigger('click')
    expect(w.get('[data-testid="delete-account-balance"]').text()).toContain('$3.50')
    const submit = w.get('[data-testid="delete-account-submit"]')
    expect(submit.attributes('disabled')).toBeDefined()
    await w.get('[data-testid="delete-account-confirm"]').setValue('注销账号')
    expect(submit.attributes('disabled')).toBeUndefined()
    await submit.trigger('click')
    await flushPromises()
    expect(deleteAccountMock).toHaveBeenCalledWith('注销账号')
    expect(logoutMock).toHaveBeenCalled()
    expect(replaceMock).toHaveBeenCalledWith('/')
  })

  it('stays signed in when the server refuses', async () => {
    deleteAccountMock.mockRejectedValueOnce(new Error('pending withdrawal'))
    const w = mount(ProfileDeleteAccountCard)
    await w.get('[data-testid="delete-account-open"]').trigger('click')
    await w.get('[data-testid="delete-account-confirm"]').setValue('注销账号')
    await w.get('[data-testid="delete-account-submit"]').trigger('click')
    await flushPromises()
    expect(showErrorMock).toHaveBeenCalled()
    expect(logoutMock).not.toHaveBeenCalled()
  })

  it('has no delete button for admins', () => {
    authState.user = { role: 'admin', balance: 0 }
    const w = mount(ProfileDeleteAccountCard)
    expect(w.find('[data-testid="delete-account-open"]').exists()).toBe(false)
  })
})
