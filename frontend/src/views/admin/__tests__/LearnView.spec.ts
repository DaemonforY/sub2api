import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import LearnView from '../LearnView.vue'

const api = vi.hoisted(() => ({ getSettings: vi.fn(), saveSettings: vi.fn(), stats: vi.fn(), certificates: vi.fn(), setCertificateRevoked: vi.fn(), insights: vi.fn(), setShowcaseHidden: vi.fn() }))
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
    api.certificates.mockResolvedValue({ items: [{ code: 'ABCDEFGH23', user_email: 'a@example.test', track: 'a', track_title: 'AI 应用开发入门', display_name: '小林', project_url: 'https://github.com/x/y', quiz_score: 90, issued_at: '2026-10-04T00:00:00Z', showcase: true, showcase_hidden: false }], total: 1 })
    api.insights.mockResolvedValue({
      funnels: [{ track: 'a', title: 'AI 应用开发入门', lessons: 8, started: 4, half: 2, finished: 1, certificates: 1 }],
      days: [{ date: '2026-10-04', learners: 3, completions: 5, runs: 9, tutor: 2, interviews: 1, certificates: 1 }],
      quizzes: { a3: { takers: 4, passed: 3, avg_score: 82, attempts: 6 } }
    })
    api.setShowcaseHidden.mockResolvedValue(undefined)
    api.setCertificateRevoked.mockResolvedValue(undefined)
  })

  it('keeps the stored key unless a new one is typed, and shows stats', async () => {
    api.saveSettings.mockImplementation(async (s) => ({ ...s, api_key: undefined, api_key_set: true }))
    const wrapper = mount(LearnView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="learn-stats"]').text()).toContain('1,234')
    expect(wrapper.text()).toContain('A3')
    expect(wrapper.text()).toContain('75%') // a3 quiz pass rate
    expect(wrapper.get('[data-testid="learn-funnels"]').text()).toContain('AI 应用开发入门')
    expect(wrapper.find('[data-testid="learn-trend"]').exists()).toBe(true)

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

  it('lists certificates and revokes one after confirming', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mount(LearnView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="learn-certs"]').text()).toContain('ABCDEFGH23')
    await wrapper.get('[data-testid="learn-cert-revoke"]').trigger('click')
    await flushPromises()
    expect(confirm).toHaveBeenCalled()
    expect(api.setCertificateRevoked).toHaveBeenCalledWith('ABCDEFGH23', true)
    expect(api.certificates).toHaveBeenCalledTimes(2)
    confirm.mockRestore()
  })

  it('takes a certificate off the learner wall', async () => {
    const wrapper = mount(LearnView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="learn-cert-wall"]').trigger('click')
    await flushPromises()
    expect(api.setShowcaseHidden).toHaveBeenCalledWith('ABCDEFGH23', true)
  })
})
