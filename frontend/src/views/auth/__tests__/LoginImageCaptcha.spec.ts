import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const loginMock = vi.fn()
const getPublicSettingsMock = vi.fn()
const getLoginCaptchaMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    currentRoute: { value: { query: {} } },
    push: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: (...args: unknown[]) => loginMock(...args),
    loginWithPasskey: vi.fn()
  }),
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showWarning: vi.fn() })
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
    getLoginCaptcha: (...args: unknown[]) => getLoginCaptchaMock(...args),
    isTotp2FARequired: () => false,
    isWeChatWebOAuthEnabled: () => false
  }
})

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        RouterLink: true,
        TurnstileWidget: true,
        Icon: true,
        LoginAgreementPrompt: true,
        TotpLoginModal: true,
        EmailOAuthButtons: true,
        LinuxDoOAuthSection: true,
        DingTalkOAuthSection: true,
        OidcOAuthSection: true,
        WechatOAuthSection: true
      }
    }
  })
}

const wrongPassword = {
  status: 401,
  reason: 'INVALID_CREDENTIALS',
  message: '邮箱或密码错误（Invalid email or password）',
  metadata: { captcha_required: 'true' }
}

describe('login image captcha', () => {
  beforeEach(() => {
    loginMock.mockReset()
    getLoginCaptchaMock.mockReset()
    getPublicSettingsMock.mockResolvedValue({ turnstile_enabled: false, backend_mode_enabled: false })
    let n = 0
    getLoginCaptchaMock.mockImplementation(async () => {
      n += 1
      return { captcha_id: `cap-${n}`, image: 'data:image/png;base64,AAAA' }
    })
  })

  async function submit(wrapper: ReturnType<typeof mountLogin>) {
    await wrapper.get('form').trigger('submit')
    await flushPromises()
  }

  it('stays hidden until the server asks, then sends the answer and renews it after a failure', async () => {
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('wrong-pass')
    expect(wrapper.find('[data-testid="login-image-captcha"]').exists()).toBe(false)

    loginMock.mockRejectedValueOnce(wrongPassword)
    await submit(wrapper)
    expect(loginMock.mock.calls[0][0].captcha_id).toBeUndefined()
    expect(wrapper.find('[data-testid="login-image-captcha"]').exists()).toBe(true)
    expect(getLoginCaptchaMock).toHaveBeenCalledTimes(1)

    // Without an answer the form does not submit.
    await submit(wrapper)
    expect(loginMock).toHaveBeenCalledTimes(1)

    loginMock.mockRejectedValueOnce(wrongPassword)
    await wrapper.get('#login-captcha').setValue('ab7k')
    await submit(wrapper)
    expect(loginMock.mock.calls[1][0]).toMatchObject({ captcha_id: 'cap-1', captcha_code: 'ab7k' })
    expect(getLoginCaptchaMock).toHaveBeenCalledTimes(2)
    expect((wrapper.get('#login-captcha').element as HTMLInputElement).value).toBe('')

    loginMock.mockResolvedValueOnce({})
    await wrapper.get('#login-captcha').setValue('M3XQ')
    await submit(wrapper)
    expect(loginMock.mock.calls[2][0]).toMatchObject({ captcha_id: 'cap-2', captcha_code: 'M3XQ' })
  })

  it('does not appear for an ordinary wrong password', async () => {
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('user@example.com')
    await wrapper.get('#password').setValue('wrong-pass')
    loginMock.mockRejectedValueOnce({ ...wrongPassword, metadata: undefined })
    await submit(wrapper)
    expect(wrapper.find('[data-testid="login-image-captcha"]').exists()).toBe(false)
    expect(getLoginCaptchaMock).not.toHaveBeenCalled()
  })
})
