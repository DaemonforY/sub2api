import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import WithdrawCard from '../WithdrawCard.vue'

const getWithdrawStatus = vi.hoisted(() => vi.fn())
const requestWithdraw = vi.hoisted(() => vi.fn())
const cancelWithdraw = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())

vi.mock('@/api/growth', () => ({ growthAPI: { getWithdrawStatus, requestWithdraw, cancelWithdraw } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('@/utils/analytics', () => ({ track: vi.fn() }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const base = {
  enabled: true,
  min_cny: 50,
  monthly_limit: 2,
  month_used: 0,
  withdrawable_cny: 120.5,
  available_quota: 20,
  cash_cny: 144,
  withdrawn_cny: 23.5,
  has_pending: false,
  withdrawals: [],
  freeze_hours: 168,
}

// ConfirmDialog renders through BaseDialog's teleport; stub it to a plain block with a confirm button.
const ConfirmDialogStub = {
  props: ['show', 'title', 'message'],
  emits: ['confirm', 'cancel'],
  template: '<div v-if="show" class="confirm-stub"><button class="confirm-yes" @click="$emit(\'confirm\')">ok</button></div>',
}

function mountCard() {
  return mount(WithdrawCard, { global: { stubs: { ConfirmDialog: ConfirmDialogStub, Icon: true } } })
}

describe('WithdrawCard', () => {
  beforeEach(() => {
    getWithdrawStatus.mockReset()
    requestWithdraw.mockReset()
    cancelWithdraw.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('stays hidden when withdrawals are off', async () => {
    getWithdrawStatus.mockResolvedValue({ ...base, enabled: false })
    const wrapper = mountCard()
    await flushPromises()
    expect(wrapper.find('[data-testid="affiliate-withdraw"]').exists()).toBe(false)
  })

  it('explains why there is no form below the minimum', async () => {
    getWithdrawStatus.mockResolvedValue({ ...base, withdrawable_cny: 30 })
    const wrapper = mountCard()
    await flushPromises()
    expect(wrapper.find('[data-testid="withdraw-form"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('growth.withdraw.notEnough')
  })

  it('prefills the last payee, asks to confirm, submits and reports the change', async () => {
    getWithdrawStatus.mockResolvedValue({ ...base, last_method: 'wechat', last_account: 'wx_abc', last_real_name: '李四' })
    requestWithdraw.mockResolvedValue({ id: 9 })
    const wrapper = mountCard()
    await flushPromises()

    expect((wrapper.get('#withdraw-account').element as HTMLInputElement).value).toBe('wx_abc')
    expect((wrapper.get('#withdraw-name').element as HTMLInputElement).value).toBe('李四')
    expect(wrapper.get('[data-testid="withdrawable-cny"]').text()).toBe('¥120.50')

    await wrapper.get('[data-testid="withdraw-amount"]').setValue('100')
    await wrapper.get('[data-testid="withdraw-form"]').trigger('submit')
    expect(requestWithdraw).not.toHaveBeenCalled()
    await wrapper.get('.confirm-yes').trigger('click')
    await flushPromises()

    expect(requestWithdraw).toHaveBeenCalledWith({ cny_amount: 100, method: 'wechat', account: 'wx_abc', real_name: '李四', note: undefined })
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })

  it('will not submit more than is withdrawable', async () => {
    getWithdrawStatus.mockResolvedValue(base)
    const wrapper = mountCard()
    await flushPromises()
    await wrapper.get('#withdraw-account').setValue('13800000000')
    await wrapper.get('#withdraw-name').setValue('张三')
    await wrapper.get('[data-testid="withdraw-amount"]').setValue('200')
    expect(wrapper.get('[data-testid="withdraw-submit"]').attributes('disabled')).toBeDefined()
  })

  it('cancels a pending withdrawal', async () => {
    const pending = { id: 5, user_id: 1, quota_amount: 3, cny_amount: 60, method: 'alipay', account: '13800000000', real_name: '张三', user_note: '', status: 'pending', admin_note: '', created_at: '2026-10-09T00:00:00Z' }
    getWithdrawStatus.mockResolvedValue({ ...base, has_pending: true, withdrawals: [pending] })
    cancelWithdraw.mockResolvedValue({ ...pending, status: 'cancelled' })
    const wrapper = mountCard()
    await flushPromises()
    expect(wrapper.text()).toContain('growth.withdraw.pendingHint')
    expect(wrapper.text()).toContain('138****00')

    await wrapper.get('[data-testid="withdraw-cancel-5"]').trigger('click')
    await wrapper.get('.confirm-yes').trigger('click')
    await flushPromises()
    expect(cancelWithdraw).toHaveBeenCalledWith(5)
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })
})
