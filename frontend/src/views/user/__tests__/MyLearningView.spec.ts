import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import MyLearningView from '../MyLearningView.vue'

const api = vi.hoisted(() => ({ me: vi.fn(), tracks: vi.fn() }))
const showError = vi.hoisted(() => vi.fn())

vi.mock('@/api/learn', () => ({ default: api }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}:${JSON.stringify(p)}` : key) })
}))

const stubs = { AppLayout: { template: '<div><slot /></div>' } }

describe('MyLearningView', () => {
  beforeEach(() => {
    api.me.mockResolvedValue({
      completed: { a1: '2026-10-01T00:00:00Z', a2: '2026-10-02T00:00:00Z' },
      runs_left: 18, tutor_left: 9, interviews_left: 3, runs_today: 2, tutor_today: 1, interviews_today: 0,
      quizzes: { a1: { correct: 4, total: 4, attempts: 1 } },
      checkpoints: {},
      certificates: [{ code: 'ABCDEFGH23', track: 'b', track_title: 'AI 绘画与视频创作', display_name: '小林', project_url: '', quiz_score: 90, issued_at: '2026-10-03T00:00:00Z' }]
    })
    api.tracks.mockResolvedValue([
      { id: 'a', letter: 'A', title: 'AI 应用开发入门', project: '', lessons: [{ id: 'a1', title: '第一课', minutes: 15 }, { id: 'a2', title: '第二课', minutes: 15 }, { id: 'a3', title: '第三课', minutes: 15 }] },
      { id: 'b', letter: 'B', title: 'AI 绘画与视频创作', project: '', lessons: [{ id: 'b1', title: '提示词', minutes: 15 }] }
    ])
  })

  it('shows progress per track, the next lesson, certificates and usage', async () => {
    const wrapper = mount(MyLearningView, { global: { stubs } })
    await flushPromises()
    const tracks = wrapper.findAll('[data-testid="my-learning-track"]')
    expect(tracks).toHaveLength(2)
    expect(tracks[0].text()).toContain('"done":2,"total":3')
    expect(tracks[0].text()).toContain('4/4')
    expect(tracks[0].find('a[href="/learn/a/a3.html"]').exists()).toBe(true)
    expect(tracks[1].text()).toContain('learn.mine.certified')
    expect(wrapper.get('[data-testid="my-learning-certs"] a').attributes('href')).toBe('/learn/cert.html?c=ABCDEFGH23')
    expect(wrapper.get('[data-testid="my-learning-usage"]').text()).toContain('"n":18')
  })

  it('still shows usage when the track list cannot load', async () => {
    api.tracks.mockRejectedValue(new Error('offline'))
    const wrapper = mount(MyLearningView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.text()).toContain('learn.mine.noTracks')
    expect(showError).not.toHaveBeenCalled()
  })
})
