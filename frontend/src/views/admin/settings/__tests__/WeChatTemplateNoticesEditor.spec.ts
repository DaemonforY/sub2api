import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import WeChatTemplateNoticesEditor from '../WeChatTemplateNoticesEditor.vue'
import { toTemplateNoticeForms } from '@/utils/wechatTemplateNotices'

const testNotice = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin/settings', () => ({ testWeChatTemplateNotice: (...args: unknown[]) => testNotice(...args) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function mountEditor() {
  const forms = ref(toTemplateNoticeForms(null))
  const wrapper = mount(WeChatTemplateNoticesEditor, {
    props: {
      modelValue: forms.value,
      'onUpdate:modelValue': (value: unknown) => {
        forms.value = value as typeof forms.value
      },
    },
    global: { stubs: { Toggle: true } },
  })
  return { wrapper, forms }
}

describe('WeChatTemplateNoticesEditor', () => {
  beforeEach(() => {
    testNotice.mockReset()
  })

  it('shows the four events with default pages and their placeholders', () => {
    const { wrapper } = mountEditor()
    for (const event of ['balance_low', 'trial_exhausted', 'subscription_expiring', 'order_abandoned']) {
      expect(wrapper.find(`[data-testid="wechat-template-notice-${event}"]`).exists()).toBe(true)
    }
    expect(
      (wrapper.get('[data-testid="wechat-template-notice-subscription_expiring-url"]').element as HTMLInputElement).value,
    ).toBe('/purchase?tab=subscription')
    const order = wrapper.get('[data-testid="wechat-template-notice-order_abandoned"]').text()
    expect(order).toContain('{site_name}')
    expect(order).toContain('{amount_cny}')
    expect(order).not.toContain('{threshold}')
  })

  it('adds field rows and flags a bad field key', async () => {
    const { wrapper, forms } = mountEditor()
    await wrapper.get('[data-testid="wechat-template-notice-balance_low-add-field"]').trigger('click')
    const key = wrapper.get('[data-testid="wechat-template-notice-balance_low-field-key-0"]')
    await key.setValue('Thing-1')
    expect(wrapper.find('[data-testid="wechat-template-notice-balance_low-key-error"]').exists()).toBe(true)
    await key.setValue('thing1')
    expect(wrapper.find('[data-testid="wechat-template-notice-balance_low-key-error"]').exists()).toBe(false)
    expect(forms.value[0].fields).toEqual([{ key: 'thing1', value: '' }])
  })

  it('sends a test message and shows the result or the error', async () => {
    const { wrapper } = mountEditor()
    testNotice.mockResolvedValueOnce({ message: '已发送到你的微信' })
    await wrapper.get('[data-testid="wechat-template-notice-balance_low-test"]').trigger('click')
    await flushPromises()
    expect(testNotice).toHaveBeenCalledWith('balance_low')
    expect(wrapper.get('[data-testid="wechat-template-notice-balance_low-test-result"]').text()).toBe('已发送到你的微信')

    testNotice.mockRejectedValueOnce({ message: '你的账号还没有关注公众号' })
    await wrapper.get('[data-testid="wechat-template-notice-order_abandoned-test"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="wechat-template-notice-order_abandoned-test-result"]').text()).toBe(
      '你的账号还没有关注公众号',
    )
  })
})
