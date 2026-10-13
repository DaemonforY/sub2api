import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ContentVersionCard from '@/components/admin/content/ContentVersionCard.vue'
import type { ContentVersion, PlatformSpec } from '@/api/admin/content'

const updateVersion = vi.fn()
const approveVersion = vi.fn()
const publishVersion = vi.fn()
const skipVersion = vi.fn()
const regenerateVersion = vi.fn()
const copyToClipboard = vi.fn()

vi.mock('@/api/admin/content', () => ({
  updateVersion: (...a: unknown[]) => updateVersion(...a),
  approveVersion: (...a: unknown[]) => approveVersion(...a),
  publishVersion: (...a: unknown[]) => publishVersion(...a),
  skipVersion: (...a: unknown[]) => skipVersion(...a),
  regenerateVersion: (...a: unknown[]) => regenerateVersion(...a)
}))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard }) }))
vi.mock('@/utils/noteCards', () => ({
  NOTE_THEMES: ['cream'],
  NOTE_THEME_NAMES: { cream: '奶油' },
  drawNoteCard: vi.fn().mockRejectedValue(new Error('no canvas in tests'))
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key) }) }
})

const spec: PlatformSpec = {
  id: 'zhihu', name: '知乎', source: 'zhihu', title_max: 50, body_min: 500, body_max: 4000, links: true, read_more: false, promo: false,
  tags_min: 1, tags_max: 5, cards: false, micro_max: 0, ai_label: '勾选「包含 AI 辅助创作」', publish_url: 'https://zhuanlan.zhihu.com/write'
}

const pending: ContentVersion = {
  id: 3, item_id: 1, platform: 'zhihu', kind: 'answer', status: 'pending_review', title: 'Base URL 加不加 v1', body: '先给结论。{{link}}',
  link_code: '', promo_code: '', post_url: '', question: 'Base URL 要不要加 /v1？', tags: ['API'],
  checks: [{ level: 'warn', code: 'body_short', text: '正文约 10 字' }]
}

describe('ContentVersionCard', () => {
  beforeEach(() => vi.clearAllMocks())

  it('saves the edit before approving and emits the approved version', async () => {
    updateVersion.mockResolvedValue({ ...pending, title: '新标题' })
    approveVersion.mockResolvedValue({ ...pending, title: '新标题', status: 'approved', link_code: 'ce3-zhihu', link_url: 'https://hivegpt.cn/go/ce3-zhihu' })
    const w = mount(ContentVersionCard, { props: { version: pending, spec, brand: 'HiveGPT' } })
    expect(w.text()).toContain('正文约 10 字')
    await w.get('[data-testid="content-version-title"]').setValue('新标题')
    await w.get('[data-testid="content-version-approve"]').trigger('click')
    await flushPromises()
    expect(updateVersion).toHaveBeenCalledWith(3, expect.objectContaining({ title: '新标题', question: 'Base URL 要不要加 /v1？', tags: ['API'] }))
    expect(approveVersion).toHaveBeenCalledWith(3)
    expect(w.emitted('updated')?.[0]?.[0]).toMatchObject({ status: 'approved' })
  })

  it('shows the publish package and records the post URL', async () => {
    const approved: ContentVersion = {
      ...pending, status: 'approved', link_code: 'ce3-zhihu', link_url: 'https://hivegpt.cn/go/ce3-zhihu',
      rendered: '先给结论。https://hivegpt.cn/go/ce3-zhihu'
    }
    publishVersion.mockResolvedValue({ ...approved, status: 'published', post_url: 'https://www.zhihu.com/question/1/answer/2' })
    const w = mount(ContentVersionCard, { props: { version: approved, spec, brand: 'HiveGPT' } })
    const pkg = w.get('[data-testid="content-publish"]').text()
    expect(pkg).toContain('https://hivegpt.cn/go/ce3-zhihu')
    expect(pkg).toContain('勾选「包含 AI 辅助创作」')
    expect(w.find('[data-testid="content-version-approve"]').exists()).toBe(false)
    await w.get('[data-testid="content-post-url"]').setValue('https://www.zhihu.com/question/1/answer/2')
    const record = w.findAll('button').find((b) => b.text() === 'admin.content.publish.record')!
    await record.trigger('click')
    await flushPromises()
    expect(publishVersion).toHaveBeenCalledWith(3, 'https://www.zhihu.com/question/1/answer/2')
  })
})
