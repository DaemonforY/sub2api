import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import LearnView from '../LearnView.vue'

const api = vi.hoisted(() => ({ getSettings: vi.fn(), saveSettings: vi.fn(), stats: vi.fn() }))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin', () => ({ adminAPI: { learn: api } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { AppLayout: { template: '<div><slot /></div>' } }

describe('admin LearnView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    api.getSettings.mockResolvedValue({ run_enabled: false, model: '', free_runs_per_day: 20, daily_cap: 1000, api_key_set: true })
    api.stats.mockResolvedValue({ learners: 3, learners_today: 1, runs_today: 4, runs_7d: 9, failed_runs_7d: 1, tokens_7d: 1234, lessons: [{ lesson_id: 'a3', completed: 2, runs: 5 }] })
  })

  it('keeps the stored key unless a new one is typed, and shows stats', async () => {
    api.saveSettings.mockImplementation(async (s) => ({ ...s, api_key: undefined, api_key_set: true }))
    const wrapper = mount(LearnView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="learn-stats"]').text()).toContain('1,234')
    expect(wrapper.text()).toContain('A3')

    await wrapper.get('[data-testid="learn-enabled"]').setValue(true)
    await wrapper.get('[data-testid="learn-model"]').setValue('gpt-5.5')
    await wrapper.get('[data-testid="learn-save"]').trigger('click')
    await flushPromises()
    expect(api.saveSettings.mock.calls[0][0]).not.toHaveProperty('api_key')
    expect(api.saveSettings.mock.calls[0][0]).toMatchObject({ run_enabled: true, model: 'gpt-5.5' })

    await wrapper.get('[data-testid="learn-key"]').setValue('sk-new')
    await wrapper.get('[data-testid="learn-save"]').trigger('click')
    await flushPromises()
    expect(api.saveSettings.mock.calls[1][0].api_key).toBe('sk-new')
    expect((wrapper.get('[data-testid="learn-key"]').element as HTMLInputElement).value).toBe('')
  })
})
