import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'

const { getPublicSettingsMock, registerMock, showErrorMock, trackMock, validateAffiliateCodeMock, validateInvitationCodeMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  trackMock: vi.fn(),
  registerMock: vi.fn(),
  showErrorMock: vi.fn(),
  validateAffiliateCodeMock: vi.fn(),
  validateInvitationCodeMock: vi.fn()
}))

const publicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  promo_code_enabled: false,
  invitation_code_enabled: false,
  affiliate_enabled: true,
  turnstile_enabled: true,
  turnstile_site_key: 'site-key',
  site_name: 'Sub2API',
  registration_email_suffix_whitelist: [],
  linuxdo_oauth_enabled: false,
  wechat_oauth_enabled: false,
  oidc_oauth_enabled: false,
  github_oauth_enabled: false,
  google_oauth_enabled: false
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ query: {} })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) =>
      key === 'auth.emailDomainRegistrationLimit'
        ? '该邮箱域名无法注册新账户。请使用主流邮箱注册；如需使用企业邮箱，请联系客服添加域名白名单。'
        : key,
    locale: { value: 'en' }
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ register: (...args: unknown[]) => registerMock(...args) }),
  useAppStore: () => ({
    showError: (...args: unknown[]) => showErrorMock(...args),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/utils/analytics', async () => {
  const actual = await vi.importActual<typeof import('@/utils/analytics')>('@/utils/analytics')
  return { ...actual, track: (...args: unknown[]) => trackMock(...args) }
})

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
    validateAffiliateCode: (...args: unknown[]) => validateAffiliateCodeMock(...args),
    validateInvitationCode: (...args: unknown[]) => validateInvitationCodeMock(...args)
  }
})

function mountRegister() {
  return mount(RegisterView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true,
        TurnstileWidget: { template: '<div data-testid="turnstile-widget" />' },
        LoginAgreementPrompt: true,
        EmailOAuthButtons: true,
        LinuxDoOAuthSection: true,
        WechatOAuthSection: true,
        OidcOAuthSection: true,
        RouterLink: true,
        transition: false
      }
    }
  })
}

describe('RegisterView invitation layout', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    registerMock.mockReset()
    showErrorMock.mockReset()
    trackMock.mockReset()
    getPublicSettingsMock.mockResolvedValue(publicSettings)
    registerMock.mockResolvedValue({})
    validateAffiliateCodeMock.mockReset()
    validateInvitationCodeMock.mockReset()
    validateAffiliateCodeMock.mockResolvedValue({ valid: true })
  })

  it('keeps the optional affiliate invitation field before Turnstile', async () => {
    const wrapper = mountRegister()
    await flushPromises()

    const invitationField = wrapper.get('[data-testid="affiliate-invitation-field"]')
    const turnstile = wrapper.get('[data-testid="registration-turnstile"]')

    expect(invitationField.get('input').attributes('id')).toBe('affiliate_code')
    expect(invitationField.text()).toContain('common.optional')
    expect(
      invitationField.element.compareDocumentPosition(turnstile.element) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
  })

  it('keeps the friend\'s invite code field next to a mandatory invitation code', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      invitation_code_enabled: true
    })

    const wrapper = mountRegister()
    await flushPromises()

    expect(wrapper.get('#invitation_code').exists()).toBe(true)
    expect(wrapper.get('[data-testid="affiliate-invitation-field"]').text()).toContain('auth.affiliateCodeLabel')
    expect(wrapper.text()).toContain('auth.registrationInvitationCodeLabel')
  })

  it('moves a friend\'s invite code typed into the invitation code box', async () => {
    vi.useFakeTimers()
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, invitation_code_enabled: true })
    validateInvitationCodeMock.mockResolvedValue({ valid: false, error_code: 'INVITATION_CODE_NOT_FOUND' })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#invitation_code').setValue('U9GCR4C53L2B')
    await vi.advanceTimersByTimeAsync(600)
    await flushPromises()
    vi.useRealTimers()

    expect(validateAffiliateCodeMock).toHaveBeenCalledWith('U9GCR4C53L2B')
    expect((wrapper.get('#invitation_code').element as HTMLInputElement).value).toBe('')
    expect((wrapper.get('#affiliate_code').element as HTMLInputElement).value).toBe('U9GCR4C53L2B')
    expect(wrapper.find('[data-testid="affiliate-moved-notice"]').exists()).toBe(true)
  })

  it('blocks sign-up with an unknown friend\'s invite code and sends a valid one', async () => {
    vi.useFakeTimers()
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false })
    validateAffiliateCodeMock.mockResolvedValueOnce({ valid: false, error_code: 'AFFILIATE_CODE_INVALID' })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('friend@allowed.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('#affiliate_code').setValue('TYPO1234')
    await vi.advanceTimersByTimeAsync(600)
    await flushPromises()
    expect(wrapper.find('[data-testid="affiliate-invalid"]').exists()).toBe(true)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(registerMock).not.toHaveBeenCalled()
    expect(showErrorMock).toHaveBeenCalledWith('auth.affiliateCodeInvalidCannotRegister')

    await wrapper.get('#affiliate_code').setValue('U9GCR4C53L2B')
    await vi.advanceTimersByTimeAsync(600)
    await flushPromises()
    vi.useRealTimers()
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(registerMock).toHaveBeenCalledWith(expect.objectContaining({ aff_code: 'U9GCR4C53L2B' }))
  })

  it('submits a non-whitelist email domain so the backend can enforce its registration quota', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com'],
      registration_email_domain_quota_enabled: true
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('first@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({ email: 'first@custom.example' })
    )
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('shows the localized registration domain quota message returned by the backend', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com'],
      registration_email_domain_quota_enabled: true
    })
    registerMock.mockRejectedValueOnce({
      reason: 'EMAIL_DOMAIN_REGISTRATION_LIMIT',
      message: 'raw backend message'
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('second@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(showErrorMock).toHaveBeenCalledWith(
      '该邮箱域名无法注册新账户。请使用主流邮箱注册；如需使用企业邮箱，请联系客服添加域名白名单。'
    )
    expect(trackMock).toHaveBeenCalledWith('signup_error', { reason: 'EMAIL_DOMAIN_REGISTRATION_LIMIT', step: 'register' })
  })

  // 域名限量注册开关默认关闭：恢复 PR5423 之前的客户端白名单预检，非白名单域名不发起注册请求。
  it('rejects a non-whitelist email domain locally when the domain quota switch is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com']
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('first@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).not.toHaveBeenCalled()
    // 校验失败通过 validationToastMessage watcher 弹 toast
    expect(showErrorMock).toHaveBeenCalledWith('auth.emailSuffixNotAllowedWithAllowed')
    expect(wrapper.get('#email').classes()).toContain('input-error')
    expect(trackMock).toHaveBeenCalledWith('signup_error', { reason: 'email_suffix' })
  })

  it('still submits whitelisted email domains when the domain quota switch is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com']
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('user@allowed.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({ email: 'user@allowed.com' })
    )
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('says what a new account gets before any field, and folds the promo code', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false, promo_code_enabled: true, signup_bonus: 2 })
    const wrapper = mountRegister()
    await flushPromises()
    const perks = wrapper.get('[data-testid="register-perks"]')
    expect(perks.text()).toContain('auth.perks.signupBonus')
    expect(perks.text()).toContain('auth.perks.what')
    expect(wrapper.find('#promo_code').exists()).toBe(false)
    await wrapper.get('[data-testid="register-promo-toggle"]').trigger('click')
    expect(wrapper.find('#promo_code').exists()).toBe(true)
  })

  it('lists the accepted mailboxes under the email field', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false, registration_email_suffix_whitelist: ['@qq.com', '*.edu.cn', '@example.org'] })
    const wrapper = mountRegister()
    await flushPromises()
    expect(wrapper.find('[data-testid="register-email-hint"]').exists()).toBe(true)

    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false, registration_email_suffix_whitelist: ['@qq.com'], registration_email_domain_quota_enabled: true })
    const quota = mountRegister()
    await flushPromises()
    expect(quota.find('[data-testid="register-email-hint"]').exists()).toBe(false)
  })

  it('makes WeChat sign-up the main button inside WeChat', async () => {
    const ua = vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue('Mozilla/5.0 (iPhone) MicroMessenger/8.0.50')
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false, wechat_oauth_enabled: true, wechat_oauth_mp_enabled: true })
    const wrapper = mountRegister()
    await flushPromises()
    expect(wrapper.find('[data-testid="register-wechat-first"]').exists()).toBe(true)
    ua.mockRestore()

    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: false, wechat_oauth_enabled: true, wechat_oauth_mp_enabled: true })
    const desktop = mountRegister()
    await flushPromises()
    expect(desktop.find('[data-testid="register-wechat-first"]').exists()).toBe(false)
  })
})
