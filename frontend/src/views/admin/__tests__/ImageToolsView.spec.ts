import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ImageToolsView from '../ImageToolsView.vue'

const { getSettings, saveSettings, stats, uses, showSuccess, showError } = vi.hoisted(() => ({
  getSettings: vi.fn(),
  saveSettings: vi.fn(),
  stats: vi.fn(),
  uses: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { imageTools: { getSettings, saveSettings, stats, uses } } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

describe('admin ImageToolsView', () => {
  it('loads settings, stats and records, and saves new prices', async () => {
    getSettings.mockResolvedValue({ enabled: true, price_remove_bg: 0.02, price_upscale: 0.05, free_daily: 3, service_configured: true })
    stats.mockResolvedValue({ today: [{ tool: 'remove_bg', runs: 4, free_runs: 3, cost: 0.02, users: 2 }], week: [], month: [] })
    uses.mockResolvedValue({ items: [{ id: 1, user_id: 7, user_email: 'u@x.test', api_key_name: '画布', tool: 'upscale', free: false, cost: 0.05, input_bytes: 1024, output_bytes: 4096, duration_ms: 8200, created_at: '2026-10-01T10:00:00Z' }], total: 1, page: 1, page_size: 20 })
    saveSettings.mockImplementation(async (input) => ({ ...input, service_configured: true }))

    const wrapper = mount(ImageToolsView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' } } } })
    await flushPromises()

    expect(wrapper.text()).toContain('u@x.test')
    expect(wrapper.text()).toContain('¥0.05')
    expect(wrapper.text()).toContain('8.2s')
    expect((wrapper.get('[data-testid="price-upscale"]').element as HTMLInputElement).value).toBe('0.05')

    await wrapper.get('[data-testid="price-upscale"]').setValue('0.1')
    await wrapper.get('[data-testid="free-daily"]').setValue('5')
    await wrapper.findAll('button').find((b) => b.text() === 'admin.imageTools.settings.save')!.trigger('click')
    await flushPromises()
    expect(saveSettings).toHaveBeenCalledWith({ enabled: true, price_remove_bg: 0.02, price_upscale: 0.1, free_daily: 5 })
    expect(showSuccess).toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
  })
})
