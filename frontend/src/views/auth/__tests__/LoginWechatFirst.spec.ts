import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const getPublicSettingsMock = vi.fn()

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
  useAuthStore: () => ({ login: vi.fn(), loginWithPasskey: vi.fn() }),
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showWarning: vi.fn(), cachedPublicSettings: null })
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
    isTotp2FARequired: () => false
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
        WechatOAuthSection: { template: '<div data-testid="wechat-section" />' }
      }
    }
  })
}

// v-show toggles an inline display: none on the form.
function formHidden(wrapper: ReturnType<typeof mountLogin>): boolean {
  return (wrapper.get('form').attributes('style') || '').includes('display: none')
}

const wechatSettings = {
  turnstile_enabled: false,
  backend_mode_enabled: false,
  wechat_oauth_enabled: true,
  wechat_oauth_mp_enabled: true
}

describe('login page with email sign-up closed', () => {
  let ua: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    localStorage.clear()
    getPublicSettingsMock.mockReset()
    ua = vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Mozilla/5.0 (iPhone) MicroMessenger/8.0.50')
  })

  afterEach(() => {
    ua.mockRestore()
  })

  it('leads with WeChat and folds the email form until asked', async () => {
    getPublicSettingsMock.mockResolvedValue({ ...wechatSettings, email_registration_enabled: false })
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.find('[data-testid="login-wechat-first"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="wechat-section"]')).toHaveLength(1)
    expect(formHidden(wrapper)).toBe(true)

    await wrapper.get('[data-testid="login-email-toggle"]').trigger('click')
    expect(formHidden(wrapper)).toBe(false)
    expect(wrapper.find('[data-testid="login-email-toggle"]').exists()).toBe(false)
  })

  it('opens the email form for a browser that last signed in with email', async () => {
    localStorage.setItem('auth:last_login_method', 'email')
    getPublicSettingsMock.mockResolvedValue({ ...wechatSettings, email_registration_enabled: false })
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.find('[data-testid="login-wechat-first"]').exists()).toBe(true)
    expect(formHidden(wrapper)).toBe(false)
  })

  it('keeps the email form first while email sign-up is open', async () => {
    getPublicSettingsMock.mockResolvedValue({ ...wechatSettings, email_registration_enabled: true })
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.find('[data-testid="login-wechat-first"]').exists()).toBe(false)
    expect(formHidden(wrapper)).toBe(false)
    expect(wrapper.findAll('[data-testid="wechat-section"]')).toHaveLength(1)
  })
})
