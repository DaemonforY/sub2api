import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GeoMonitorView from '@/views/admin/GeoMonitorView.vue'

const api = vi.hoisted(() => ({
  listQuestions: vi.fn(),
  createQuestion: vi.fn(),
  updateQuestion: vi.fn(),
  deleteQuestion: vi.fn(),
  listEngines: vi.fn(),
  createEngine: vi.fn(),
  updateEngine: vi.fn(),
  deleteEngine: vi.fn(),
  testEngine: vi.fn(),
  listChecks: vi.fn(),
  addManualCheck: vi.fn(),
  deleteCheck: vi.fn(),
  startRun: vi.fn(),
  getStatus: vi.fn(),
  getSummary: vi.fn(),
  getSettings: vi.fn(),
  saveSettings: vi.fn()
}))

vi.mock('@/api/admin/geo', () => api)
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key) })
  }
})

const questions = [
  { id: 1, question: '国内怎么调用 GPT API？', category: '接入', enabled: true, sort: 10, created_at: '', updated_at: '' },
  { id: 2, question: '哪里能学 AI 应用开发？', category: '学习', enabled: true, sort: 20, created_at: '', updated_at: '' }
]
const check = (over: Record<string, unknown>) => ({
  id: 1, question_id: 1, question: '国内怎么调用 GPT API？', engine_id: 5, engine_name: 'Perplexity', source: 'auto', answer: '推荐 HiveGPT',
  mentioned: true, cited_urls: ['https://hivegpt.cn/learn/', 'https://other.com/'], our_urls: ['https://hivegpt.cn/learn/'], competitors: ['openrouter'],
  error: '', run_id: 'r1', created_at: '2026-10-10T08:00:00Z', ...over
})
const summary = {
  engines: [
    { name: 'Perplexity', auto: true, enabled: true, answered: 2, mentioned: 1, mention_rate: 0.5, last_checked_at: null },
    { name: '豆包', auto: false, enabled: false, answered: 1, mentioned: 0, mention_rate: 0, last_checked_at: null }
  ],
  questions,
  latest: [
    check({}),
    check({ id: 2, question_id: 2, question: '哪里能学 AI 应用开发？', mentioned: false, our_urls: [], cited_urls: [], competitors: [] }),
    check({ id: 3, engine_id: null, engine_name: '豆包', source: 'manual', mentioned: false, our_urls: [], cited_urls: [], competitors: [] })
  ],
  weekly: [{ engine_name: 'Perplexity', week: '2026-W41', week_start: '2026-10-05', total: 2, mentioned: 1, rate: 0.5 }],
  totals: { questions: 2, engines: 1, answered: 3, mentioned: 1, mention_rate: 1 / 3 }
}
const engine = {
  id: 5, name: 'Perplexity', base_url: 'https://api.perplexity.ai', model: 'sonar', extra_body: null, enabled: true,
  has_key: true, key_masked: '••••abcd', created_at: '', updated_at: ''
}

const mountView = () => mount(GeoMonitorView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, teleport: true } } })

describe('GeoMonitorView', () => {
  beforeEach(() => {
    Object.values(api).forEach((f) => f.mockReset())
    api.getSummary.mockResolvedValue(summary)
    api.listQuestions.mockResolvedValue(questions)
    api.listEngines.mockResolvedValue([engine])
    api.getStatus.mockResolvedValue({ running: false, run_id: '', started_at: null, last_run_at: null, done: 0, total: 0 })
  })

  it('renders the question × engine matrix from the summary and opens an answer', async () => {
    const w = mountView()
    await flushPromises()
    const matrix = w.get('[data-testid="geo-matrix"]')
    expect(matrix.text()).toContain('国内怎么调用 GPT API？')
    expect(matrix.text()).toContain('豆包')
    expect(w.get('[data-testid="geo-cell-1-Perplexity"]').text()).toBe('admin.geo.overview.yes')
    expect(w.get('[data-testid="geo-cell-2-Perplexity"]').text()).toBe('admin.geo.overview.no')
    expect(w.find('[data-testid="geo-cell-2-豆包"]').exists()).toBe(false) // no data → —
    expect(w.get('[data-testid="geo-engine-cards"]').text()).toContain('50%')
    expect(w.get('[data-testid="geo-trend"]').text()).toContain('2026-W41')

    await w.get('[data-testid="geo-cell-1-Perplexity"]').trigger('click')
    const detail = w.get('[data-testid="geo-detail"]')
    expect(detail.text()).toContain('推荐 HiveGPT')
    expect(detail.text()).toContain('openrouter')
    const ours = w.get('[data-testid="geo-detail-cited"]').findAll('a').find((a) => a.text() === 'https://hivegpt.cn/learn/')
    expect(ours?.classes()).toContain('text-green-600')
  })

  it('posts a manual entry', async () => {
    api.addManualCheck.mockResolvedValue(check({ id: 9, source: 'manual' }))
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="geo-manual-open"]').trigger('click')
    await w.get('#geo-m-q').setValue('2')
    await w.get('#geo-m-engine').setValue(' 腾讯元宝 ')
    await w.get('#geo-m-answer').setValue('可以看 hivegpt.cn 的教程')
    await w.get('#geo-m-cited').setValue('https://hivegpt.cn/learn/')
    await w.get('[data-testid="geo-manual-form"]').trigger('submit')
    await flushPromises()
    expect(api.addManualCheck).toHaveBeenCalledWith({
      question_id: 2, engine_name: '腾讯元宝', answer: '可以看 hivegpt.cn 的教程', cited_urls: 'https://hivegpt.cn/learn/'
    })
    expect(api.getSummary).toHaveBeenCalledTimes(2)
  })

  it('never shows an engine key and keeps it when the field is left empty', async () => {
    api.updateEngine.mockResolvedValue(engine)
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="geo-tab-engines"]').trigger('click')
    await flushPromises()
    const table = w.get('[data-testid="geo-engines-table"]').text()
    expect(table).toContain('••••abcd')
    expect(table).not.toContain('sk-')

    await w.get('[data-testid="geo-engines-table"]').findAll('button').find((b) => b.text() === 'admin.geo.common.edit')!.trigger('click')
    const keyInput = w.get('#geo-e-key').element as HTMLInputElement
    expect(keyInput.value).toBe('')
    expect(keyInput.type).toBe('password')
    await w.get('#geo-e-model').setValue('sonar-pro')
    await w.get('[data-testid="geo-engine-form"]').trigger('submit')
    await flushPromises()
    expect(api.updateEngine).toHaveBeenCalledWith(5, expect.objectContaining({ api_key: '', model: 'sonar-pro' }))
  })

  it('fills a preset and starts a run', async () => {
    api.createEngine.mockResolvedValue({ ...engine, id: 6 })
    api.startRun.mockResolvedValue({ run_id: 'r2' })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="geo-tab-engines"]').trigger('click')
    await w.get('[data-testid="geo-engine-new"]').trigger('click')
    await w.get('[data-testid="geo-preset-qwen"]').trigger('click')
    expect((w.get('#geo-e-base').element as HTMLInputElement).value).toBe('https://dashscope.aliyuncs.com/compatible-mode/v1')
    expect((w.get('#geo-e-extra').element as HTMLTextAreaElement).value).toBe('{"enable_search": true}')

    await w.get('[data-testid="geo-tab-overview"]').trigger('click')
    await w.get('[data-testid="geo-run"]').trigger('click')
    await flushPromises()
    expect(api.startRun).toHaveBeenCalled()
  })
})
