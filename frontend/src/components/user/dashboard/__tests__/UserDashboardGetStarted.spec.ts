import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UserDashboardGetStarted from '@/components/user/dashboard/UserDashboardGetStarted.vue'
import { markGuideStep } from '@/utils/getStarted'

const getDashboardStats = vi.fn()
const track = vi.fn()

vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardStats: (...a: unknown[]) => getDashboardStats(...a) } }))
vi.mock('@/utils/analytics', () => ({ track: (...a: unknown[]) => track(...a) }))
vi.mock('@/api/subscriptions', () => ({ default: { getActiveSubscriptions: vi.fn().mockResolvedValue([]) }, getActiveSubscriptions: vi.fn().mockResolvedValue([]) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (k: string) => k }) }
})

function mountCard(props: Partial<{ userId: number; balance: number; apiKeys: number; requests: number }> = {}) {
  return mount(UserDashboardGetStarted, {
    props: { userId: 7, balance: 5, apiKeys: 0, requests: 0, ...props },
    global: { stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }, Icon: true } }
  })
}

describe('UserDashboardGetStarted', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    getDashboardStats.mockReset()
    track.mockReset()
    vi.useFakeTimers()
  })
  afterEach(() => vi.useRealTimers())

  it('is not shown to users who already made calls', () => {
    expect(mountCard({ apiKeys: 1, requests: 12 }).find('[data-testid="dashboard-get-started"]').exists()).toBe(false)
  })

  it('starts with creating a key and links to the keys page', async () => {
    const w = mountCard()
    await flushPromises()
    expect(w.get('[data-testid="get-started-step-key"]').attributes('data-done')).toBe('false')
    expect(w.get('[data-testid="get-started-go-key"]').attributes('href')).toBe('/keys?action=create')
    expect(w.find('[data-testid="get-started-waiting"]').exists()).toBe(false)
    expect(track).toHaveBeenCalledWith('guide_view', { step: 'key' })
  })

  it('waits for the first call and celebrates it', async () => {
    markGuideStep('copied', 7)
    getDashboardStats.mockResolvedValueOnce({ total_api_keys: 1, total_requests: 0 }).mockResolvedValue({ total_api_keys: 1, total_requests: 1 })
    const w = mountCard({ apiKeys: 1 })
    await flushPromises()
    expect(w.get('[data-testid="get-started-step-key"]').attributes('data-done')).toBe('true')
    expect(w.get('[data-testid="get-started-step-config"]').attributes('data-done')).toBe('true')
    expect(w.find('[data-testid="get-started-waiting"]').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(w.get('[data-testid="get-started-step-call"]').attributes('data-done')).toBe('false')
    await vi.advanceTimersByTimeAsync(10_000)
    expect(w.get('[data-testid="get-started-step-call"]').attributes('data-done')).toBe('true')
    expect(w.text()).toContain('getStarted.doneTitle')
    expect(track).toHaveBeenCalledWith('guide_first_call')
    expect(getDashboardStats).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(30_000)
    expect(getDashboardStats).toHaveBeenCalledTimes(2) // stops polling once done
  })

  it('warns about a zero balance and remembers being closed', async () => {
    const w = mountCard({ balance: 0 })
    await flushPromises()
    expect(w.find('[data-testid="get-started-credit"]').exists()).toBe(true)
    await w.get('[data-testid="get-started-dismiss"]').trigger('click')
    expect(w.find('[data-testid="dashboard-get-started"]').exists()).toBe(false)
    expect(mountCard({ balance: 0 }).find('[data-testid="dashboard-get-started"]').exists()).toBe(false)
  })
})
