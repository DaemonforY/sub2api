import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import HomeAssistant from '../HomeAssistant.vue'

const { getAssistantConfig, streamAssistant } = vi.hoisted(() => ({ getAssistantConfig: vi.fn(), streamAssistant: vi.fn() }))

vi.mock('@/api/assistant', () => ({ getAssistantConfig, streamAssistant }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, args?: Record<string, unknown>) => (args ? `${key} ${JSON.stringify(args)}` : key) })
}))

const config = (over = {}) => ({ enabled: true, left: 5, per_day: 5, logged_in: false, user_per_day: 20, ...over })

describe('home support assistant', () => {
  beforeEach(() => {
    getAssistantConfig.mockReset()
    streamAssistant.mockReset()
    sessionStorage.clear()
  })

  it('stays hidden while the assistant is off', async () => {
    getAssistantConfig.mockResolvedValue(config({ enabled: false }))
    const wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    await flushPromises()
    expect(wrapper.find('[data-testid="home-assistant"]').exists()).toBe(false)
  })

  it('streams an answer as Markdown with its sources and the count left', async () => {
    getAssistantConfig.mockResolvedValue(config())
    streamAssistant.mockImplementation(async (messages, onDelta) => {
      expect(messages).toEqual([{ role: 'user', content: '怎么创建 Key？' }])
      onDelta('在 **API 密钥** 页')
      onDelta('创建。')
      return { sources: [{ title: '怎么创建 API Key', url: '/keys' }], left: 4 }
    })
    const wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    await flushPromises()
    await wrapper.get('[data-testid="home-assistant-open"]').trigger('click')
    expect(wrapper.text()).toContain('supportAssistant.guestHint')
    await wrapper.get('[data-testid="home-assistant-input"]').setValue('怎么创建 Key？')
    await wrapper.get('[data-testid="home-assistant-send"]').trigger('click')
    await flushPromises()

    expect(wrapper.html()).toContain('<strong>API 密钥</strong>')
    expect(wrapper.get('a[href="/keys"]').text()).toBe('怎么创建 API Key')
    expect(wrapper.text()).toContain('supportAssistant.left {"n":4}')
    expect(JSON.parse(sessionStorage.getItem('support_assistant_chat') || '{}').messages).toHaveLength(2)
  })

  it("does not show another account's kept chat", async () => {
    getAssistantConfig.mockResolvedValue(config({ logged_in: true, tools: true }))
    const chat = [{ role: 'user', content: '我的余额' }, { role: 'assistant', content: '你的余额是 $12.34' }]
    localStorage.setItem('auth_user', JSON.stringify({ id: 6 }))
    sessionStorage.setItem('support_assistant_chat', JSON.stringify({ owner: 5, messages: chat }))
    let wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    await flushPromises()
    await wrapper.get('[data-testid="home-assistant-open"]').trigger('click')
    expect(wrapper.text()).not.toContain('$12.34')

    localStorage.setItem('auth_user', JSON.stringify({ id: 5 }))
    wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    await flushPromises()
    await wrapper.get('[data-testid="home-assistant-open"]').trigger('click')
    expect(wrapper.text()).toContain('$12.34')
    localStorage.removeItem('auth_user')
  })

  it('shows each account lookup while it runs and lists them under the answer', async () => {
    getAssistantConfig.mockResolvedValue(config({ logged_in: true, tools: true }))
    let seenWhileWorking = ''
    const view: { wrapper?: ReturnType<typeof mount> } = {}
    streamAssistant.mockImplementation(async (_messages, onDelta, _signal, onTool) => {
      onTool('正在查你最近的报错')
      await flushPromises()
      seenWhileWorking = view.wrapper!.text()
      onTool('正在查看你的 API Key')
      onTool('正在查你最近的报错')
      onDelta('你的 codex Key 不支持 gpt-4o。')
      return { sources: [], left: 19 }
    })
    const wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    view.wrapper = wrapper
    await flushPromises()
    await wrapper.get('[data-testid="home-assistant-open"]').trigger('click')
    expect(wrapper.text()).toContain('supportAssistant.greetingTools')
    expect(wrapper.text()).toContain('supportAssistant.suggestions.diagnose')
    expect(wrapper.text()).toContain('supportAssistant.disclaimerTools')
    await wrapper.get('[data-testid="home-assistant-input"]').setValue('为什么报错？')
    await wrapper.get('[data-testid="home-assistant-send"]').trigger('click')
    await flushPromises()

    expect(seenWhileWorking).toContain('正在查你最近的报错')
    expect(wrapper.text()).toContain('你的 codex Key 不支持 gpt-4o。')
    expect(wrapper.get('[data-testid="home-assistant-steps"]').text()).toBe('supportAssistant.checked {"list":"你最近的报错、你的 API Key"}')
    expect(wrapper.text()).not.toContain('正在查')
  })

  it('shows the server error and drops the unanswered question', async () => {
    getAssistantConfig.mockResolvedValueOnce(config()).mockResolvedValueOnce(config({ left: 0 }))
    streamAssistant.mockRejectedValue(new Error('今天的 5 次免费提问用完了'))
    const wrapper = mount(HomeAssistant, { props: { siteName: 'HiveGPT' } })
    await flushPromises()
    await wrapper.get('[data-testid="home-assistant-open"]').trigger('click')
    await wrapper.get('[data-testid="home-assistant-input"]').setValue('再问一个')
    await wrapper.get('[data-testid="home-assistant-input"]').trigger('keydown', { key: 'Enter' })
    await flushPromises()

    expect(wrapper.get('[data-testid="home-assistant-error"]').text()).toContain('今天的 5 次免费提问用完了')
    expect(wrapper.text()).not.toContain('再问一个')
    expect(wrapper.text()).toContain('supportAssistant.none')
  })
})
