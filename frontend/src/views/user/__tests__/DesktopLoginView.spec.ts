import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DesktopLoginView from '../DesktopLoginView.vue'

const routeState = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
const getDesktopLogin = vi.hoisted(() => vi.fn())
const decideDesktopLogin = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({ useRoute: () => routeState }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, p?: Record<string, string>) => (p ? `${key}:${JSON.stringify(p)}` : key) })
}))
vi.mock('@/api/desktop', () => ({ getDesktopLogin, decideDesktopLogin }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: {} }) }))

const TICKET = 'a'.repeat(32)

describe('DesktopLoginView', () => {
  beforeEach(() => {
    routeState.query = { t: TICKET }
    getDesktopLogin.mockReset().mockResolvedValue({
      device_name: '张三的笔记本',
      status: 'pending',
      expires_in: 500,
      same_network: true,
      computer_network: '198.51.100.*',
      started_ago: 20
    })
    decideDesktopLogin.mockReset()
  })

  it('shows the device and confirms the sign-in', async () => {
    decideDesktopLogin.mockResolvedValue({ status: 'approved' })
    const w = mount(DesktopLoginView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(getDesktopLogin).toHaveBeenCalledWith(TICKET)
    expect(w.get('[data-testid="desktop-login-device"]').text()).toContain('张三的笔记本')
    await w.get('[data-testid="desktop-login-approve"]').trigger('click')
    await flushPromises()
    expect(decideDesktopLogin).toHaveBeenCalledWith(TICKET, true, false)
    expect(w.find('[data-testid="desktop-login-approved"]').exists()).toBe(true)
    expect(w.find('[data-testid="desktop-login-not-me"]').exists()).toBe(true)
  })

  it('rejects when the user says it was not them', async () => {
    decideDesktopLogin.mockResolvedValue({ status: 'rejected' })
    const w = mount(DesktopLoginView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    await w.get('[data-testid="desktop-login-reject"]').trigger('click')
    await flushPromises()
    expect(decideDesktopLogin).toHaveBeenCalledWith(TICKET, false, false)
    expect(w.find('[data-testid="desktop-login-rejected"]').exists()).toBe(true)
  })

  it('does not call the API for a malformed ticket', async () => {
    routeState.query = { t: 'nope' }
    const w = mount(DesktopLoginView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(getDesktopLogin).not.toHaveBeenCalled()
    expect(w.get('[data-testid="desktop-login-message"]').text()).toBe('desktopLogin.invalid')
  })

  it('says the request was handled when it is no longer pending', async () => {
    getDesktopLogin.mockResolvedValue({ device_name: 'PC', status: 'approved', expires_in: 100 })
    const w = mount(DesktopLoginView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(w.get('[data-testid="desktop-login-message"]').text()).toBe('desktopLogin.handled')
  })

  it('on another network, approving needs 「这台电脑就在我面前」', async () => {
    getDesktopLogin.mockResolvedValue({
      device_name: '未命名电脑',
      status: 'pending',
      expires_in: 400,
      same_network: false,
      computer_network: '203.0.113.*',
      started_ago: 200
    })
    decideDesktopLogin.mockResolvedValue({ status: 'approved' })
    const w = mount(DesktopLoginView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(w.get('[data-testid="desktop-login-network"]').text()).toContain('desktopLogin.otherNetwork')
    expect(w.get('[data-testid="desktop-login-network"]').text()).toContain('203.0.113.*')
    expect(w.find('[data-testid="desktop-login-phishing"]').exists()).toBe(true)
    const approve = w.get('[data-testid="desktop-login-approve"]')
    expect(approve.attributes('disabled')).toBeDefined()
    await w.get('[data-testid="desktop-login-nearby"]').setValue(true)
    expect(approve.attributes('disabled')).toBeUndefined()
    await approve.trigger('click')
    await flushPromises()
    expect(decideDesktopLogin).toHaveBeenCalledWith(TICKET, true, true)
  })
})
