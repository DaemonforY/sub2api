import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ShihuaAiDraftDialog from '../ShihuaAiDraftDialog.vue'

const api = vi.hoisted(() => ({ draftWorks: vi.fn(), importWorks: vi.fn(), listWorks: vi.fn(), verifyWork: vi.fn() }))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin/shihua', () => api)
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' } }

function work(slug: string, title: string, author: string, text: string) {
  return { slug, title, author, dynasty: '南宋', genre: 'shi', stage: 'primary', grade: '', textbooks: [], full_text: text, display_text: '', excerpt_note: '', famous_lines: '', sources: '', variant_note: '', sort: 1000 }
}
const draft = {
  model: 'gpt-test',
  items: [
    { work: work('cun-ju', '村居', '高鼎', '草长莺飞二月天，拂堤杨柳醉春烟。'), status: 'match', diffs: [], ai_text: '', page: '村居 (高鼎)', page_url: 'https://zh.wikisource.org/wiki/x', reason: '早春放风筝', problem: '' },
    { work: work('su-xin-shi', '宿新市徐公店', '杨万里', '篱落疏疏一迳深'), status: 'diff', diffs: [{ source: '迳', ai: '径' }], ai_text: '篱落疏疏一径深', page: '宿新市徐公店', page_url: '', reason: '', problem: '' },
    { work: work('zhou-guo', '舟过安仁', '杨万里', '一叶渔船两小童'), status: 'ai_only', diffs: [], ai_text: '一叶渔船两小童', page: '', page_url: '', reason: '', problem: '维基文库没找到这篇' },
    { work: work('shan-ju', '山居秋暝', '王维', ''), status: 'exists', diffs: [], ai_text: '', page: '', page_url: '', reason: '', problem: '库里已有这篇' },
    { work: work('modern', '某首现代诗', '某诗人', ''), status: 'rejected', diffs: [], ai_text: '', page: '', page_url: '', reason: '', problem: '不是古代作品' }
  ]
}

describe('ShihuaAiDraftDialog', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    api.draftWorks.mockResolvedValue(structuredClone(draft))
  })

  it('drafts by theme, then imports only the kept works as edited, checking them once', async () => {
    api.importWorks.mockResolvedValue({ created: 2, updated: 0, unchanged: 0, reset: [], errors: [] })
    api.listWorks.mockResolvedValue([
      { id: 11, slug: 'cun-ju', text_status: 'draft' },
      { id: 12, slug: 'zhou-guo', text_status: 'draft' },
      { id: 13, slug: 'shan-ju', text_status: 'verified_1' }
    ])
    api.verifyWork.mockResolvedValue({})
    const wrapper = mount(ShihuaAiDraftDialog, { props: { show: true }, global: { stubs } })

    expect(wrapper.get('[data-testid="shihua-ai-generate"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="shihua-ai-topic"]').setValue('小学春天田园、有孩子在活动的古诗')
    await wrapper.get('[data-testid="shihua-ai-generate"]').trigger('click')
    await flushPromises()
    expect(api.draftWorks).toHaveBeenCalledWith({ topic: '小学春天田园、有孩子在活动的古诗', stage: '', count: 10 })

    // Existing and rejected works can't be ticked; the difference is spelled out.
    const box = (slug: string) => wrapper.get(`[data-testid="shihua-ai-item-${slug}"] input[type="checkbox"]`).element as HTMLInputElement
    expect(box('cun-ju').checked).toBe(true)
    expect(box('shan-ju').disabled).toBe(true)
    expect(box('modern').disabled).toBe(true)
    expect(wrapper.get('[data-testid="shihua-ai-item-su-xin-shi"] [data-testid="shihua-ai-diffs"]').text()).toContain('迳')
    expect(wrapper.text()).toContain('维基文库没找到这篇')

    // Fix the text from the textbook, drop one work, count the import as check 1.
    await wrapper.get('[data-testid="shihua-ai-item-su-xin-shi"] input[type="checkbox"]').setValue(false)
    await wrapper.get('[data-testid="shihua-ai-item-cun-ju"] input.font-medium').setValue('村居（草长莺飞二月天）')
    await wrapper.get('[data-testid="shihua-ai-verify1"]').setValue(true)
    await wrapper.get('[data-testid="shihua-ai-confirm"]').trigger('click')
    await flushPromises()

    const sent = api.importWorks.mock.calls[0]
    expect(sent[1]).toBe(false)
    expect(sent[0].map((w: { slug: string }) => w.slug)).toEqual(['cun-ju', 'zhou-guo'])
    expect(sent[0][0].title).toBe('村居（草长莺飞二月天）')
    expect(api.verifyWork.mock.calls).toEqual([[11, 1], [12, 1]])
    expect(wrapper.emitted('imported')?.[0]).toEqual([['cun-ju', 'zhou-guo']])
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('drafts by titles and does not check when not asked to', async () => {
    api.importWorks.mockResolvedValue({ created: 3, updated: 0, unchanged: 0, reset: [], errors: [] })
    const wrapper = mount(ShihuaAiDraftDialog, { props: { show: true }, global: { stubs } })
    const buttons = wrapper.findAll('button')
    await buttons.find((b) => b.text() === 'shihua.ai.modes.titles')!.trigger('click')
    await wrapper.get('[data-testid="shihua-ai-titles"]').setValue('村居 高鼎\n宿新市徐公店 杨万里')
    await wrapper.get('[data-testid="shihua-ai-generate"]').trigger('click')
    await flushPromises()
    expect(api.draftWorks).toHaveBeenCalledWith({ titles: '村居 高鼎\n宿新市徐公店 杨万里', stage: '' })

    await wrapper.get('[data-testid="shihua-ai-confirm"]').trigger('click')
    await flushPromises()
    expect(api.importWorks.mock.calls[0][0]).toHaveLength(3)
    expect(api.listWorks).not.toHaveBeenCalled()
    expect(api.verifyWork).not.toHaveBeenCalled()
  })

  it('shows the error when the model fails', async () => {
    api.draftWorks.mockRejectedValue({ message: '文本模型调用失败' })
    const wrapper = mount(ShihuaAiDraftDialog, { props: { show: true }, global: { stubs } })
    await wrapper.get('[data-testid="shihua-ai-topic"]').setValue('春天')
    await wrapper.get('[data-testid="shihua-ai-generate"]').trigger('click')
    await flushPromises()
    expect(showError).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="shihua-ai-confirm"]').attributes('disabled')).toBeDefined()
  })
})
