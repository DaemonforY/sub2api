import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import EduVerifyCard from '../EduVerifyCard.vue'

const getEduStatus = vi.hoisted(() => vi.fn())
const sendEduCode = vi.hoisted(() => vi.fn())
const verifyEdu = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())

vi.mock('@/api/growth', () => ({ growthAPI: { getEduStatus, sendEduCode, verifyEdu } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const baseStatus = { enabled: true, discount_percent: 20, suffixes: ['edu.cn'] }

describe('EduVerifyCard', () => {
  beforeEach(() => {
    getEduStatus.mockReset()
    sendEduCode.mockReset().mockResolvedValue(undefined)
    verifyEdu.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('renders nothing when verification is disabled', async () => {
    getEduStatus.mockResolvedValue({ ...baseStatus, enabled: false })
    const wrapper = mount(EduVerifyCard)
    await flushPromises()
    expect(wrapper.find('[data-testid="edu-verify-card"]').exists()).toBe(false)
  })

  it('shows the verified state', async () => {
    getEduStatus.mockResolvedValue({ ...baseStatus, verification: { user_id: 1, email: 'a@pku.edu.cn', verified_at: '2026-09-30T00:00:00Z' } })
    const wrapper = mount(EduVerifyCard)
    await flushPromises()
    expect(wrapper.find('[data-testid="edu-verified"]').exists()).toBe(true)
  })

  it('sends a code, verifies and emits verified', async () => {
    getEduStatus.mockResolvedValueOnce(baseStatus).mockResolvedValueOnce({ ...baseStatus, verification: { user_id: 1, email: 'a@pku.edu.cn', verified_at: '' } })
    verifyEdu.mockResolvedValue({ user_id: 1, email: 'a@pku.edu.cn', verified_at: '' })
    const wrapper = mount(EduVerifyCard)
    await flushPromises()

    await wrapper.get('#edu-email').setValue('a@pku.edu.cn')
    await wrapper.get('[data-testid="edu-send-code"]').trigger('click')
    await flushPromises()
    expect(sendEduCode).toHaveBeenCalledWith('a@pku.edu.cn')

    await wrapper.get('#edu-code').setValue('123456')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(verifyEdu).toHaveBeenCalledWith('a@pku.edu.cn', '123456')
    expect(wrapper.emitted('verified')).toHaveLength(1)
    expect(wrapper.find('[data-testid="edu-verified"]').exists()).toBe(true)
  })
})
