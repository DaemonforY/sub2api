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
    getDesktopLogin.mockReset().mockResolvedValue({ device_name: '张三的笔记本', status: 'pending', expires_in: 500 })
    decideDesktopLogin.mockReset()
  })

  it('shows the device and confirms the sign-in', async () => {
    decideDesktopLogin.mockResolvedValue({ status: 'approved' })
    const w = mount(DesktopLoginView)
    await flushPromises()
    expect(getDesktopLogin).toHaveBeenCalledWith(TICKET)
    expect(w.get('[data-testid="desktop-login-device"]').text()).toContain('张三的笔记本')
    await w.get('[data-testid="desktop-login-approve"]').trigger('click')
    await flushPromises()
    expect(decideDesktopLogin).toHaveBeenCalledWith(TICKET, true)
    expect(w.find('[data-testid="desktop-login-approved"]').exists()).toBe(true)
  })

  it('rejects when the user says it was not them', async () => {
    decideDesktopLogin.mockResolvedValue({ status: 'rejected' })
    const w = mount(DesktopLoginView)
    await flushPromises()
    await w.get('[data-testid="desktop-login-reject"]').trigger('click')
    await flushPromises()
    expect(decideDesktopLogin).toHaveBeenCalledWith(TICKET, false)
    expect(w.find('[data-testid="desktop-login-rejected"]').exists()).toBe(true)
  })

  it('does not call the API for a malformed ticket', async () => {
    routeState.query = { t: 'nope' }
    const w = mount(DesktopLoginView)
    await flushPromises()
    expect(getDesktopLogin).not.toHaveBeenCalled()
    expect(w.get('[data-testid="desktop-login-message"]').text()).toBe('desktopLogin.invalid')
  })

  it('says the request was handled when it is no longer pending', async () => {
    getDesktopLogin.mockResolvedValue({ device_name: 'PC', status: 'approved', expires_in: 100 })
    const w = mount(DesktopLoginView)
    await flushPromises()
    expect(w.get('[data-testid="desktop-login-message"]').text()).toBe('desktopLogin.handled')
  })
})
