import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import WechatOAuthSection from '@/components/auth/WechatOAuthSection.vue'
import { useAppStore } from '@/stores'
import type { PublicSettings } from '@/types'

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
}))

const authApiMocks = vi.hoisted(() => ({
  createWeChatQRLogin: vi.fn(),
  getWeChatQRLoginStatus: vi.fn(),
  createWeChatCodeLogin: vi.fn(),
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    createWeChatQRLogin: (...args: unknown[]) => authApiMocks.createWeChatQRLogin(...args),
    getWeChatQRLoginStatus: (...args: unknown[]) => authApiMocks.getWeChatQRLoginStatus(...args),
    createWeChatCodeLogin: (...args: unknown[]) => authApiMocks.createWeChatCodeLogin(...args),
  }
})

const PC_UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36'
const IPHONE_UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1'
const WECHAT_UA = `${IPHONE_UA} MicroMessenger/8.0.50`

function setUserAgent(value: string): void {
  Object.defineProperty(window.navigator, 'userAgent', { configurable: true, value })
}

let pinia: ReturnType<typeof createPinia>

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en' },
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'auth.wechatProviderName') {
          return 'Mock WeChat'
        }
        if (key === 'auth.oidc.signIn') {
          return `Continue with ${params?.providerName ?? ''}`.trim()
        }
        if (key === 'auth.oauthFlow.wechatSystemBrowserOnly') {
          return 'MOCK-SYSTEM-BROWSER-ONLY'
        }
        if (key === 'auth.oauthFlow.wechatBrowserOnly') {
          return 'MOCK-WECHAT-BROWSER-ONLY'
        }
        if (key === 'auth.oauthFlow.wechatNotConfigured') {
          return 'MOCK-NOT-CONFIGURED'
        }
        if (key === 'auth.oauthOrContinue') {
          return 'or continue'
        }
        return key
      },
    }),
  }
})

type WeChatPublicSettings = PublicSettings

function buildPublicSettings(overrides: Partial<WeChatPublicSettings> = {}): WeChatPublicSettings {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '/api/v1',
    contact_info: '',
    doc_url: '',
    home_content: '',
    compact_home_enabled: false,
    hide_ccs_import_button: false,
    payment_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [10, 20, 50, 100],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    wechat_oauth_enabled: true,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    backend_mode_enabled: false,
    version: 'test',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    ...overrides,
  }
}

function seedPublicSettings(overrides: Partial<WeChatPublicSettings> = {}): void {
  const appStore = useAppStore()
  const settings = buildPublicSettings(overrides)
  appStore.cachedPublicSettings = settings
  appStore.publicSettingsLoaded = true
}

describe('WechatOAuthSection', () => {
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    routeState.query = { redirect: '/billing?plan=pro' }
    Object.defineProperty(window.navigator, 'userAgent', {
      configurable: true,
      value: 'Mozilla/5.0',
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('starts the open WeChat OAuth flow with the current redirect target when open mode is configured', async () => {
    seedPublicSettings({
      wechat_oauth_open_enabled: true,
      wechat_oauth_mp_enabled: false,
    })
    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    expect(wrapper.text()).toContain('Mock WeChat')

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider: 'wechat',
      params: { mode: 'open', redirect: '/billing?plan=pro' }
    })
  })

  it('uses mp mode inside the WeChat browser when mp mode is configured', async () => {
    Object.defineProperty(window.navigator, 'userAgent', {
      configurable: true,
      value: 'Mozilla/5.0 MicroMessenger',
    })
    seedPublicSettings({
      wechat_oauth_open_enabled: false,
      wechat_oauth_mp_enabled: true,
    })
    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider: 'wechat',
      params: { mode: 'mp', redirect: '/billing?plan=pro' }
    })
  })

  it('disables the button outside the WeChat browser when only mp mode is configured', async () => {
    seedPublicSettings({
      wechat_oauth_open_enabled: false,
      wechat_oauth_mp_enabled: true,
    })
    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('MOCK-WECHAT-BROWSER-ONLY')

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')).toBeUndefined()
  })

  it('disables the button inside the WeChat browser when only open mode is configured', async () => {
    Object.defineProperty(window.navigator, 'userAgent', {
      configurable: true,
      value: 'Mozilla/5.0 MicroMessenger',
    })
    seedPublicSettings({
      wechat_oauth_open_enabled: true,
      wechat_oauth_mp_enabled: false,
    })
    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('MOCK-SYSTEM-BROWSER-ONLY')

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')).toBeUndefined()
  })

  it('uses the legacy overall enabled flag when per-mode settings are not present', async () => {
    seedPublicSettings({
      wechat_oauth_enabled: true,
    })
    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider: 'wechat',
      params: { mode: 'open', redirect: '/billing?plan=pro' }
    })
  })

  it('shows the localized not-configured hint when WeChat OAuth is unavailable', async () => {
    seedPublicSettings({
      wechat_oauth_enabled: false,
      wechat_oauth_open_enabled: false,
      wechat_oauth_mp_enabled: false,
    })

    const wrapper = mount(WechatOAuthSection, {
      global: {
        plugins: [pinia],
      },
    })

    expect(wrapper.text()).toContain('MOCK-NOT-CONFIGURED')
  })

  describe('公众号 QR / code sign-in outside WeChat', () => {
    const locationState = { href: 'http://localhost/login', pathname: '/login' }
    const originalLocation = window.location
    const MP_ONLY_WITH_PANELS: Partial<WeChatPublicSettings> = {
      wechat_oauth_open_enabled: false,
      wechat_oauth_mp_enabled: true,
      wechat_qr_login_enabled: true,
      wechat_code_login_enabled: true,
      wechat_mp_account_name: 'HiveGPT 助手',
      wechat_mp_follow_qr_url: 'https://open.weixin.qq.com/qr/code?username=gh_abc',
    }

    beforeEach(() => {
      vi.useFakeTimers()
      localStorage.clear()
      sessionStorage.clear()
      localStorage.setItem('hg_vid', 'visitor-1')
      locationState.href = 'http://localhost/login'
      Object.defineProperty(window, 'location', { configurable: true, value: locationState })
      authApiMocks.createWeChatQRLogin.mockReset().mockResolvedValue({
        ticket_id: 'ticket-1',
        qr_url: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=abc',
        expires_in: 300,
        poll_interval: 2,
      })
      authApiMocks.getWeChatQRLoginStatus.mockReset()
      authApiMocks.createWeChatCodeLogin.mockReset().mockResolvedValue({
        ticket_id: 'code-ticket-1',
        code: '482913',
        expires_in: 300,
        poll_interval: 2,
      })
    })

    afterEach(() => {
      // Unmounting also removes each panel's visibilitychange listener.
      while (mounted.length) {
        try {
          mounted.pop()?.unmount()
        } catch {
          // already unmounted by the test
        }
      }
      vi.useRealTimers()
      Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    })

    const mounted: Array<{ unmount: () => void }> = []

    function mountSection(props: Record<string, unknown> = {}) {
      const wrapper = mount(WechatOAuthSection, { props, global: { plugins: [pinia] } })
      mounted.push(wrapper)
      return wrapper
    }

    it('keeps the mp redirect inside WeChat even when the QR / code flows are on', async () => {
      setUserAgent(WECHAT_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')

      expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
        provider: 'wechat',
        params: { mode: 'mp', redirect: '/billing?plan=pro' },
      })
      expect(wrapper.find('[data-testid="wechat-login-panel"]').exists()).toBe(false)
    })

    it('falls back to the WeChat-browser-only hint on PC when the QR / code flags are off', () => {
      setUserAgent(PC_UA)
      seedPublicSettings({ wechat_oauth_open_enabled: false, wechat_oauth_mp_enabled: true })
      const wrapper = mountSection()

      expect(wrapper.get('[data-testid="wechat-oauth-button"]').attributes('disabled')).toBeDefined()
      expect(wrapper.text()).toContain('MOCK-WECHAT-BROWSER-ONLY')
    })

    it('shows the QR panel on PC, polls it and goes to the callback page once done', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus
        .mockResolvedValueOnce({ status: 'pending' })
        .mockResolvedValueOnce({ status: 'scanned' })
        .mockResolvedValueOnce({ status: 'done', redirect: '/auth/wechat/callback' })
      const wrapper = mountSection({ affCode: 'AFF1', promoCode: 'PROMO1' })

      expect(wrapper.find('[data-testid="wechat-oauth-hint"]').exists()).toBe(false)
      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="wechat-qr-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="wechat-qr-image"]').attributes('src')).toBe(
        'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=abc',
      )
      expect(wrapper.text()).toContain('auth.wechatLogin.qrCaption')
      expect(authApiMocks.createWeChatQRLogin).toHaveBeenCalledWith(
        { redirect: '/billing?plan=pro', intent: 'login', aff_code: 'AFF1', promo_code: 'PROMO1' },
        {},
      )

      await vi.advanceTimersByTimeAsync(2000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledWith('ticket-1')
      expect(locationState.href).toBe('http://localhost/login')

      await vi.advanceTimersByTimeAsync(2000)
      expect(wrapper.find('[data-testid="wechat-qr-scanned"]').exists()).toBe(true)

      await vi.advanceTimersByTimeAsync(2000)
      expect(locationState.href).toBe('/auth/wechat/callback')
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(3)

      await vi.advanceTimersByTimeAsync(10000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(3)
    })

    it('offers a refresh when the QR code expires and creates a new one on click', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'expired' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(2000)

      const refresh = wrapper.get('[data-testid="wechat-qr-refresh"]')
      expect(refresh.text()).toContain('auth.wechatLogin.qrExpired')
      await vi.advanceTimersByTimeAsync(10000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)

      await refresh.trigger('click')
      await flushPromises()
      expect(authApiMocks.createWeChatQRLogin).toHaveBeenCalledTimes(2)
      expect(wrapper.find('[data-testid="wechat-qr-refresh"]').exists()).toBe(false)
    })

    it('expires the QR code locally after expires_in even without a status answer', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.createWeChatQRLogin.mockResolvedValue({
        ticket_id: 'ticket-2', qr_url: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=x', expires_in: 5, poll_interval: 10,
      })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(5000)

      expect(wrapper.get('[data-testid="wechat-qr-refresh"]').text()).toContain('auth.wechatLogin.qrExpired')
      await vi.advanceTimersByTimeAsync(20000)
      expect(authApiMocks.getWeChatQRLoginStatus).not.toHaveBeenCalled()
    })

    it('stops polling when the section is unmounted', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'pending' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(2000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)

      wrapper.unmount()
      await vi.advanceTimersByTimeAsync(10000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)
    })

    it('runs the page captcha before creating the QR code and stops when it is cancelled', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      const requestProof = vi.fn().mockResolvedValueOnce(null).mockResolvedValueOnce({ turnstile_token: 'tok' })
      const wrapper = mountSection({ requestProof })

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      expect(authApiMocks.createWeChatQRLogin).not.toHaveBeenCalled()
      expect(wrapper.get('[data-testid="wechat-qr-refresh"]').text()).toContain('auth.wechatLogin.qrLoad')

      await wrapper.get('[data-testid="wechat-qr-refresh"]').trigger('click')
      await flushPromises()
      expect(authApiMocks.createWeChatQRLogin).toHaveBeenCalledWith(
        expect.objectContaining({ redirect: '/billing?plan=pro', intent: 'login' }),
        { turnstile_token: 'tok' },
      )
    })

    it('switches from the QR panel to the code panel on PC and polls the code ticket instead', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'pending' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="wechat-switch-code"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="wechat-code-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="wechat-login-code"]').text()).toBe('482913')
      // The copy-link hint and the 打开微信 button are for phones only.
      expect(wrapper.find('[data-testid="wechat-open-in-wechat"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="wechat-open-app"]').exists()).toBe(false)
      await vi.advanceTimersByTimeAsync(2000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledWith('code-ticket-1')
    })

    it('shows a code to send in a phone browser and signs in once the 公众号 receives it', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus
        .mockResolvedValueOnce({ status: 'pending' })
        .mockResolvedValueOnce({ status: 'scanned' })
        .mockResolvedValueOnce({ status: 'done', redirect: '/auth/wechat/callback' })
      const wrapper = mountSection({ affCode: 'AFF2' })

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()

      expect(authApiMocks.createWeChatQRLogin).not.toHaveBeenCalled()
      expect(authApiMocks.createWeChatCodeLogin).toHaveBeenCalledWith(
        { redirect: '/billing?plan=pro', intent: 'login', aff_code: 'AFF2', promo_code: undefined },
        {},
      )
      expect(wrapper.get('[data-testid="wechat-login-code"]').text()).toBe('482913')
      expect(wrapper.text()).toContain('auth.wechatLogin.codeTitle')
      expect(wrapper.find('[data-testid="wechat-copy-code"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="wechat-copy-name"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="wechat-open-app"]').attributes('href')).toBe('weixin://')
      expect(wrapper.find('[data-testid="wechat-follow-qr"] img').attributes('src')).toBe(
        'https://open.weixin.qq.com/qr/code?username=gh_abc',
      )
      expect(wrapper.find('[data-testid="wechat-open-in-wechat"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="wechat-code-input"]').exists()).toBe(false)

      await vi.advanceTimersByTimeAsync(2000)
      await vi.advanceTimersByTimeAsync(2000)
      // scanned is only a short step: keep polling.
      expect(locationState.href).toBe('http://localhost/login')
      await vi.advanceTimersByTimeAsync(2000)
      expect(locationState.href).toBe('/auth/wechat/callback')
    })

    it('says the code expired and gets a new one on click', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'expired' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(2000)

      const refresh = wrapper.get('[data-testid="wechat-code-refresh"]')
      expect(refresh.text()).toContain('auth.wechatLogin.codeExpired')
      expect(wrapper.find('[data-testid="wechat-login-code"]').exists()).toBe(false)

      await refresh.trigger('click')
      await flushPromises()
      expect(authApiMocks.createWeChatCodeLogin).toHaveBeenCalledTimes(2)
      expect(wrapper.get('[data-testid="wechat-login-code"]').text()).toBe('482913')
    })

    it('shows the error message from the status and retries on click', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({
        status: 'error',
        message: '该微信已绑定其他账号（WeChat already bound）',
      })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(2000)

      const retry = wrapper.get('[data-testid="wechat-code-refresh"]')
      expect(retry.text()).toContain('该微信已绑定其他账号')
      expect(retry.text()).toContain('auth.wechatLogin.retry')
      await vi.advanceTimersByTimeAsync(10000)
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)

      await retry.trigger('click')
      await flushPromises()
      expect(authApiMocks.createWeChatCodeLogin).toHaveBeenCalledTimes(2)
    })

    it('polls right away when the page becomes visible again', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'done', redirect: '/auth/wechat/callback' })
      let visibility: DocumentVisibilityState = 'visible'
      Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => visibility })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      expect(authApiMocks.getWeChatQRLoginStatus).not.toHaveBeenCalled()

      visibility = 'hidden'
      document.dispatchEvent(new Event('visibilitychange'))
      await flushPromises()
      expect(authApiMocks.getWeChatQRLoginStatus).not.toHaveBeenCalled()

      visibility = 'visible'
      document.dispatchEvent(new Event('visibilitychange'))
      await flushPromises()
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledTimes(1)
      expect(locationState.href).toBe('/auth/wechat/callback')
    })

    it('resumes a stored, unexpired code ticket after the tab reloads instead of creating a new one', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      sessionStorage.setItem('hg_wechat_login_ticket:login:/login', JSON.stringify({
        ticket_id: 'kept-ticket',
        kind: 'code',
        code: '135790',
        poll_interval: 2,
        expires_at: Date.now() + 120_000,
      }))
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'done', redirect: '/auth/wechat/callback' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()

      expect(authApiMocks.createWeChatCodeLogin).not.toHaveBeenCalled()
      // Polled right away: the user is most likely back from WeChat.
      expect(authApiMocks.getWeChatQRLoginStatus).toHaveBeenCalledWith('kept-ticket')
      expect(locationState.href).toBe('/auth/wechat/callback')
      expect(sessionStorage.getItem('hg_wechat_login_ticket:login:/login')).toBeNull()
    })

    it('ignores an expired or other-kind stored ticket and stores the new one', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      sessionStorage.setItem('hg_wechat_login_ticket:login:/login', JSON.stringify({
        ticket_id: 'old-ticket', kind: 'code', code: '111111', expires_at: Date.now() - 1000,
      }))
      authApiMocks.getWeChatQRLoginStatus.mockResolvedValue({ status: 'pending' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()

      expect(authApiMocks.createWeChatCodeLogin).toHaveBeenCalledTimes(1)
      expect(wrapper.get('[data-testid="wechat-login-code"]').text()).toBe('482913')
      const stored = JSON.parse(sessionStorage.getItem('hg_wechat_login_ticket:login:/login') || '{}')
      expect(stored).toMatchObject({ ticket_id: 'code-ticket-1', kind: 'code', code: '482913' })
      expect(stored.expires_at).toBeGreaterThan(Date.now())
    })

    it('clears the stored ticket when it expires or ends in an error', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      authApiMocks.getWeChatQRLoginStatus
        .mockResolvedValueOnce({ status: 'expired' })
        .mockResolvedValueOnce({ status: 'error', message: '登录失败' })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')
      await flushPromises()
      expect(sessionStorage.getItem('hg_wechat_login_ticket:login:/login')).not.toBeNull()
      await vi.advanceTimersByTimeAsync(2000)
      expect(sessionStorage.getItem('hg_wechat_login_ticket:login:/login')).toBeNull()

      await wrapper.get('[data-testid="wechat-code-refresh"]').trigger('click')
      await flushPromises()
      expect(sessionStorage.getItem('hg_wechat_login_ticket:login:/login')).not.toBeNull()
      await vi.advanceTimersByTimeAsync(2000)
      expect(wrapper.get('[data-testid="wechat-code-refresh"]').text()).toContain('登录失败')
      expect(sessionStorage.getItem('hg_wechat_login_ticket:login:/login')).toBeNull()
    })

    it('keeps the 开放平台 redirect on PC when an open app is configured', async () => {
      setUserAgent(PC_UA)
      seedPublicSettings({ ...MP_ONLY_WITH_PANELS, wechat_oauth_open_enabled: true })
      const wrapper = mountSection()

      await wrapper.get('[data-testid="wechat-oauth-button"]').trigger('click')

      expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
        provider: 'wechat',
        params: { mode: 'open', redirect: '/billing?plan=pro' },
      })
      expect(wrapper.find('[data-testid="wechat-login-panel"]').exists()).toBe(false)
    })

    it('opens the panel right away with autoOpen', async () => {
      setUserAgent(IPHONE_UA)
      seedPublicSettings(MP_ONLY_WITH_PANELS)
      const wrapper = mountSection({ autoOpen: true })
      await flushPromises()

      expect(wrapper.find('[data-testid="wechat-code-panel"]').exists()).toBe(true)
    })
  })
})
