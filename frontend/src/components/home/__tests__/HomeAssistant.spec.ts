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
    expect(JSON.parse(sessionStorage.getItem('support_assistant_chat') || '[]')).toHaveLength(2)
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
