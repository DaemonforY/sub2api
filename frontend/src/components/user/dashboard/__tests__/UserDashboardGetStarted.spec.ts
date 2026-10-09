import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UserDashboardGetStarted from '@/components/user/dashboard/UserDashboardGetStarted.vue'
import { markGuideStep } from '@/utils/getStarted'

const getDashboardStats = vi.fn()
const track = vi.fn()

const starter = vi.fn()
const download = vi.fn()
vi.mock('@/api/keys', () => ({ keysAPI: { starter: (...a: unknown[]) => starter(...a) } }))
vi.mock('@/utils/codexStarterConfig', async () => {
  const actual = await vi.importActual<typeof import('@/utils/codexStarterConfig')>('@/utils/codexStarterConfig')
  return { ...actual, downloadTextFile: (...a: unknown[]) => download(...a) }
})
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
    starter.mockReset()
    download.mockReset()
    starter.mockResolvedValue({ key: { id: 1, name: '我的第一个 Key', key: 'sk-starter' }, created: true })
    vi.useFakeTimers()
  })
  afterEach(() => vi.useRealTimers())

  it('is not shown to users who already made calls', () => {
    expect(mountCard({ apiKeys: 1, requests: 12 }).find('[data-testid="dashboard-get-started"]').exists()).toBe(false)
  })

  it('makes a key for newcomers and walks them through Codex without a terminal', async () => {
    const w = mountCard()
    await flushPromises()
    expect(starter).toHaveBeenCalledTimes(1)
    expect(w.get('[data-testid="get-started-step-key"]').attributes('data-done')).toBe('true')
    expect(track).toHaveBeenCalledWith('guide_view', { step: 'key' })
    expect(track).toHaveBeenCalledWith('key_created', { source: 'starter' })

    // The Codex setup opens by itself, for the visitor's system.
    expect(w.find('[data-testid="get-started-config"]').exists()).toBe(true)
    await w.get('[data-testid="get-started-os-windows"]').trigger('click')
    expect(w.get('[data-testid="get-started-folder"]').text()).toBe('%USERPROFILE%\\.codex')
    await w.get('[data-testid="get-started-os-mac"]').trigger('click')
    expect(w.get('[data-testid="get-started-folder"]').text()).toBe('~/.codex')

    // Downloading the file ticks step 2; the key is in the file, not in any URL.
    await w.get('[data-testid="get-started-download"]').trigger('click')
    expect(download).toHaveBeenCalledTimes(1)
    const [name, toml] = download.mock.calls[0] as [string, string]
    expect(name).toBe('config.toml')
    expect(toml).toContain('experimental_bearer_token = "sk-starter"')
    expect(w.get('[data-testid="get-started-step-config"]').attributes('data-done')).toBe('true')
    expect(w.find('[data-testid="get-started-waiting"]').exists()).toBe(true)
  })

  it('offers the canvas to newcomers who only want to draw', async () => {
    const w = mountCard()
    await flushPromises()
    const href = w.get('[data-testid="get-started-canvas-link"]').attributes('href')
    expect(href).toContain('/image?')
    expect(href).toContain('utm_medium=dashboard-guide')
  })

  it('falls back to the key form when the key cannot be made', async () => {
    starter.mockRejectedValue(new Error('nope'))
    const w = mountCard()
    await flushPromises()
    expect(w.get('[data-testid="get-started-step-key"]').attributes('data-done')).toBe('false')
    expect(w.get('[data-testid="get-started-go-key"]').attributes('href')).toBe('/keys?action=create')
    expect(w.find('[data-testid="get-started-config"]').exists()).toBe(false)
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
