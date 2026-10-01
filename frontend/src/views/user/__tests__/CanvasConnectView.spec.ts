import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CanvasConnectView from '../CanvasConnectView.vue'

const routeState = vi.hoisted(() => ({ query: {} as Record<string, unknown> }))
const list = vi.hoisted(() => vi.fn())
const create = vi.hoisted(() => vi.fn())
const getAvailable = vi.hoisted(() => vi.fn())
const createCanvasSession = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({ useRoute: () => routeState }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/keys', () => ({ keysAPI: { list, create } }))
vi.mock('@/api/groups', () => ({ getAvailable }))
vi.mock('@/api/canvasSessions', () => ({ createCanvasSession }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { site_name: 'HiveGPT' }, siteName: 'HiveGPT', showError: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 12 } }) }))

const drawGroup = { id: 2, name: 'Codex', allow_image_generation: true, status: 'active', subscription_type: 'subscription' }
const chatGroup = { id: 4, name: 'Claude', allow_image_generation: false, status: 'active', subscription_type: 'subscription' }
const STATE = 'a'.repeat(36)

function stubOpener() {
  const postMessage = vi.fn()
  Object.defineProperty(window, 'opener', { configurable: true, value: { closed: false, postMessage } })
  return postMessage
}

describe('CanvasConnectView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(window, 'close').mockImplementation(() => {})
    list.mockReset().mockResolvedValue({
      items: [
        { id: 1, name: 'claude-key', key: 'sk-claude-000000001111', status: 'active', group: chatGroup },
        { id: 2, name: 'draw-key', key: 'sk-draw-0000000022222', status: 'active', group: drawGroup },
      ],
    })
    getAvailable.mockReset().mockResolvedValue([drawGroup, chatGroup])
    create.mockReset()
    createCanvasSession.mockReset().mockResolvedValue({ id: 1 })
  })
  afterEach(() => {
    vi.useRealTimers()
    Object.defineProperty(window, 'opener', { configurable: true, value: null })
  })

  it('refuses to hand off without the canvas opener or with a malformed state', async () => {
    routeState.query = { state: 'short' }
    stubOpener()
    const wrapper = mount(CanvasConnectView)
    await flushPromises()
    expect(wrapper.find('[data-testid="canvas-connect-invalid"]').exists()).toBe(true)
    expect(list).not.toHaveBeenCalled()
  })

  it('preselects an image-capable key and posts it only to the canvas origin after authorize', async () => {
    routeState.query = { state: STATE }
    const postMessage = stubOpener()
    const wrapper = mount(CanvasConnectView)
    await flushPromises()

    const radios = wrapper.findAll('input[type="radio"]')
    expect(radios).toHaveLength(2)
    // Image-capable key is sorted first and selected; the chat-only key is disabled.
    expect((radios[0].element as HTMLInputElement).checked).toBe(true)
    expect((radios[1].element as HTMLInputElement).disabled).toBe(true)
    expect(postMessage).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="canvas-connect-authorize"]').trigger('click')
    await flushPromises()
    expect(createCanvasSession).toHaveBeenCalledTimes(1)
    expect(postMessage).toHaveBeenCalledWith(
      { type: 'hivegpt:canvas-key', state: STATE, signedIn: true, apiKey: 'sk-draw-0000000022222', keyName: 'draw-key' },
      'https://canvas.hivegpt.cn',
    )
    expect(wrapper.find('[data-testid="canvas-connect-done"]').exists()).toBe(true)
    vi.advanceTimersByTime(1600)
    expect(window.close).toHaveBeenCalled()
  })

  it('offers to create a key in an image-capable group when none can draw', async () => {
    routeState.query = { state: STATE }
    stubOpener()
    list.mockResolvedValue({ items: [{ id: 1, name: 'claude-key', key: 'sk-claude-000000001111', status: 'active', group: chatGroup }] })
    const wrapper = mount(CanvasConnectView)
    await flushPromises()
    expect(wrapper.find('[data-testid="canvas-connect-create"]').exists()).toBe(true)
    // Without a key the canvas can still be signed in.
    expect(wrapper.get('[data-testid="canvas-connect-authorize"]').text()).toBe('canvasConnect.signInOnly')

    create.mockResolvedValue({ id: 9 })
    list.mockResolvedValueOnce({ items: [{ id: 9, name: '无限画布', key: 'sk-new-00000000009999', status: 'active', group: drawGroup }] })
    await wrapper.get('[data-testid="canvas-connect-create"] button').trigger('click')
    await flushPromises()
    expect(create).toHaveBeenCalledWith('canvasConnect.keyName', 2)
    expect(wrapper.get('[data-testid="canvas-connect-authorize"]').text()).toBe('canvasConnect.authorize')
  })

  it('signs in without a key and does not hand anything over when the session fails', async () => {
    routeState.query = { state: STATE }
    const postMessage = stubOpener()
    list.mockResolvedValue({ items: [] })
    getAvailable.mockResolvedValue([])
    const wrapper = mount(CanvasConnectView)
    await flushPromises()

    createCanvasSession.mockRejectedValueOnce(new Error('boom'))
    await wrapper.get('[data-testid="canvas-connect-authorize"]').trigger('click')
    await flushPromises()
    expect(postMessage).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="canvas-connect-authorize"]').trigger('click')
    await flushPromises()
    expect(postMessage).toHaveBeenCalledWith({ type: 'hivegpt:canvas-key', state: STATE, signedIn: true }, 'https://canvas.hivegpt.cn')
  })
})
