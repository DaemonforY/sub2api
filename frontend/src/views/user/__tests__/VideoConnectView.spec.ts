import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import VideoConnectView from '../VideoConnectView.vue'

const routeState = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
const list = vi.hoisted(() => vi.fn())
const create = vi.hoisted(() => vi.fn())
const getAvailable = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({ useRoute: () => routeState }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/keys', () => ({ keysAPI: { list, create } }))
vi.mock('@/api/groups', () => ({ getAvailable }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { site_name: 'HiveGPT' }, siteName: 'HiveGPT', showError: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 12 } }) }))

const group = { id: 2, name: 'Codex', status: 'active', subscription_type: 'subscription' }
const STATE = 'b'.repeat(36)
const originalLocation = window.location

describe('VideoConnectView', () => {
  let locationState: { href: string }

  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(window, 'close').mockImplementation(() => {})
    list.mockReset().mockResolvedValue({
      items: [{ id: 7, name: 'video-key', key: 'sk-video-0000000077777', status: 'active', group }],
    })
    getAvailable.mockReset().mockResolvedValue([group])
    locationState = { href: 'http://localhost/video-connect' }
    Object.defineProperty(window, 'location', { configurable: true, value: locationState })
    Object.defineProperty(window, 'opener', { configurable: true, value: null })
  })

  afterEach(() => {
    vi.useRealTimers()
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    Object.defineProperty(window, 'opener', { configurable: true, value: null })
  })

  it('posts the key to the video popup opener, as before', async () => {
    routeState.query = { state: STATE }
    const postMessage = vi.fn()
    Object.defineProperty(window, 'opener', { configurable: true, value: { closed: false, postMessage } })
    const wrapper = mount(VideoConnectView)
    await flushPromises()

    await wrapper.get('[data-testid="video-connect-authorize"]').trigger('click')
    expect(postMessage).toHaveBeenCalledWith(
      { type: 'hivegpt:video-key', state: STATE, apiKey: 'sk-video-0000000077777', keyName: 'video-key' },
      'https://video.hivegpt.cn',
    )
    expect(locationState.href).toBe('http://localhost/video-connect')
  })

  it('refuses without an opener and without a return path', async () => {
    routeState.query = { state: STATE }
    const wrapper = mount(VideoConnectView)
    await flushPromises()
    expect(wrapper.find('[data-testid="video-connect-invalid"]').exists()).toBe(true)
    expect(list).not.toHaveBeenCalled()
  })

  it('in full-page mode sends the key back to the video origin in the URL fragment', async () => {
    routeState.query = { state: STATE, return: '/p/42?tab=script' }
    const wrapper = mount(VideoConnectView)
    await flushPromises()

    expect(wrapper.find('[data-testid="video-connect-invalid"]').exists()).toBe(false)
    await wrapper.get('[data-testid="video-connect-authorize"]').trigger('click')

    const target = new URL(locationState.href)
    expect(target.origin).toBe('https://video.hivegpt.cn')
    expect(target.pathname).toBe('/p/42')
    expect(target.search).toBe('?tab=script')
    const fragment = new URLSearchParams(target.hash.slice(1))
    expect(fragment.get('hivegpt_video_state')).toBe(STATE)
    expect(fragment.get('hivegpt_video_key')).toBe('sk-video-0000000077777')
    expect(wrapper.find('[data-testid="video-connect-done"]').exists()).toBe(true)
  })

  it.each([
    ['https://evil.example/steal'],
    ['//evil.example/steal'],
    ['/ok\\..\\evil'],
    ['/a#b'],
  ])('rejects the unsafe return path %s', async (unsafe) => {
    routeState.query = { state: STATE, return: unsafe }
    const wrapper = mount(VideoConnectView)
    await flushPromises()
    expect(wrapper.find('[data-testid="video-connect-invalid"]').exists()).toBe(true)
    expect(list).not.toHaveBeenCalled()
  })
})
