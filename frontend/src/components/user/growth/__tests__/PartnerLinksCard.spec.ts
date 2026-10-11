import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PartnerLinksCard from '@/components/user/growth/PartnerLinksCard.vue'

const getPartnerLinks = vi.fn()
vi.mock('@/api/growth', () => ({ growthAPI: { getPartnerLinks: (...a: unknown[]) => getPartnerLinks(...a) } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (k: string, p?: Record<string, unknown>) => (p ? `${k}${JSON.stringify(p)}` : k) }) }
})

const link = {
  code: 'xiaoming', name: '小明 B 站', promo_code: 'XIAOMING3', clicks: 120, visitors: 90,
  signups: 12, activated: 8, paid_users: 3, revenue: 300, rebates: 90
}

describe('PartnerLinksCard', () => {
  it('stays hidden for users without a partner link', async () => {
    getPartnerLinks.mockResolvedValue({ days: 30, links: [] })
    const w = mount(PartnerLinksCard)
    await flushPromises()
    expect(getPartnerLinks).toHaveBeenCalledWith(30)
    expect(w.find('[data-testid="partner-links"]').exists()).toBe(false)
  })

  it('shows the link with its numbers and reloads per period', async () => {
    getPartnerLinks.mockResolvedValue({ days: 30, links: [link] })
    const w = mount(PartnerLinksCard)
    await flushPromises()
    expect(w.text()).toContain('/go/xiaoming')
    expect(w.text()).toContain('growth.partnerLinks.promo{"code":"XIAOMING3"}')
    expect(w.get('[data-testid="partner-link-xiaoming-paid"]').text()).toBe('3')
    expect(w.get('[data-testid="partner-link-xiaoming-rebates"]').text()).toBe('90.00')
    getPartnerLinks.mockResolvedValue({ days: 0, links: [] })
    await w.get('[data-testid="partner-links-period-0"]').trigger('click')
    await flushPromises()
    expect(getPartnerLinks).toHaveBeenLastCalledWith(0)
    // Still visible with an empty period so the blogger can switch back.
    expect(w.find('[data-testid="partner-links"]').exists()).toBe(true)
  })
})
