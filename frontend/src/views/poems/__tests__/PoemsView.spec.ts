import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'

import PoemsView from '../PoemsView.vue'

const api = vi.hoisted(() => ({ poemsConfig: vi.fn(), searchPoems: vi.fn(), getPoem: vi.fn(), findPoems: vi.fn() }))
const auth = vi.hoisted(() => ({ isAuthenticated: false }))
const { showError } = vi.hoisted(() => ({ showError: vi.fn() }))

vi.mock('@/api/poems', () => api)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn() }) }))
vi.mock('@/utils/analytics', () => ({ track: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key} ${JSON.stringify(p)}` : key) })
}))

const poem = { id: 1, collection: 'lidai', title: '村居', author: '高鼎', dynasty: '清', content: '草长莺飞二月天，拂堤杨柳醉春烟。', tags: [], famous: false }
const config = { enabled: true, find_enabled: true, collections: [{ id: 'tang', name: '全唐诗', count: 5 }], total: 5, free_left: 2, free_per_day: 3, find_price: 0.1 }

function mountView() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/poems', component: PoemsView }, { path: '/poems/:id', component: { template: '<div />' } }, { path: '/login', component: { template: '<div />' } }] })
  void router.push('/poems')
  return router.isReady().then(() => mount(PoemsView, { global: { plugins: [router], stubs: { PlazaNavBar: true } } }))
}

describe('PoemsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.poemsConfig.mockResolvedValue({ ...config })
    api.searchPoems.mockResolvedValue({ items: [poem], has_more: false })
  })

  it('opens with famous poems and searches with highlighted words', async () => {
    const wrapper = await mountView()
    await flushPromises()
    expect(api.searchPoems).toHaveBeenCalledWith(expect.objectContaining({ famous: true, random: true }))
    await wrapper.get('[data-testid="poems-q"]').setValue('纸鸢 东风')
    await wrapper.get('[data-testid="poems-search"]').trigger('click')
    await flushPromises()
    expect(api.searchPoems).toHaveBeenLastCalledWith(expect.objectContaining({ q: '纸鸢 东风', random: false }))
    expect(wrapper.find('[data-testid="poem-draw-1"]').attributes('href')).toContain('prompt=')
  })

  it('asks guests to sign in before finding by scene', async () => {
    auth.isAuthenticated = false
    const wrapper = await mountView()
    await flushPromises()
    await wrapper.get('[data-testid="poems-tab-find"]').trigger('click')
    expect(wrapper.find('[data-testid="poems-find"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="poems-login"]').attributes('href')).toContain('/login')
  })

  it('finds by scene for signed-in users and shows what it cost', async () => {
    auth.isAuthenticated = true
    api.findPoems.mockResolvedValue({ keywords: ['纸鸢'], picks: [{ poem, reason: '放纸鸢' }], candidates: [poem], free: false, charged: 0.1, free_left: 0, balance: 1.2 })
    const wrapper = await mountView()
    await flushPromises()
    expect(wrapper.get('[data-testid="poems-quota"]').text()).toContain('quotaFree')
    await wrapper.get('[data-testid="poems-tab-find"]').trigger('click')
    await wrapper.get('[data-testid="poems-scene"]').setValue('春天放风筝')
    await wrapper.get('[data-testid="poems-find"]').trigger('click')
    await flushPromises()
    expect(api.findPoems).toHaveBeenCalledWith({ scene: '春天放风筝' })
    expect(wrapper.get('[data-testid="poems-picks"]').text()).toContain('放纸鸢')
    expect(wrapper.text()).toContain('poems.find.charged')
    expect(wrapper.get('[data-testid="poems-quota"]').text()).toContain('quotaPaid')
  })
})
