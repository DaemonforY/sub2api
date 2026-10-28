import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import MaterialsView from '../MaterialsView.vue'

const api = vi.hoisted(() => ({
  getQueue: vi.fn(),
  listTexts: vi.fn(),
  getText: vi.fn(),
  importShihua: vi.fn(),
  getSegmentImagery: vi.fn(),
  listImagery: vi.fn(),
  recommendStyles: vi.fn(),
  listArtworks: vi.fn(),
  extractImagery: vi.fn(),
  saveSegmentImagery: vi.fn(),
  buildPrompt: vi.fn(),
  drawSegment: vi.fn(),
  listCandidates: vi.fn(),
  draftSeeds: vi.fn(),
  setImageryStatus: vi.fn(),
  updateImagery: vi.fn(),
  listStyles: vi.fn(),
  listBlocklist: vi.fn(),
  checkPrompt: vi.fn(),
  reviewArtwork: vi.fn(),
  redrawArtwork: vi.fn(),
  retryArtwork: vi.fn(),
  cancelArtworks: vi.fn(),
  listRenders: vi.fn(),
  saveRender: vi.fn(),
  exportZip: vi.fn()
}))
// jsdom has no canvas: the poster renderer is faked, the tests check which words are sent.
const layoutUtils = vi.hoisted(() => ({
  renderShihuaPoster: vi.fn(),
  canvasToBlob: vi.fn(),
  loadShihuaArt: vi.fn(),
  loadShihuaFont: vi.fn()
}))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin/materials', () => api)
vi.mock('@/utils/shihuaLayout', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/utils/shihuaLayout')>()), ...layoutUtils }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' }
}

const text = {
  id: 1, slug: 'shi-zhi-saishang', kind: 'shi', title: '使至塞上', performer: '', body: '', translation: '', appreciation: '',
  license_status: 'public_domain', license_note: '', sources: '', status: 'published', external_url: '', themes: [], seasons: [],
  created_at: '', updated_at: '', author_name: '王维', dynasty: '唐', segment_count: 4, text_overlay_allowed: true
}
const segment = { id: 13, text_id: 1, seq: 3, label: '第3联', content: '大漠孤烟直，长河落日圆。', scene_desc: '', is_famous: true }
const segCtx = { ...segment, kind: 'shi', title: '使至塞上', author_name: '王维', dynasty: '唐', license_status: 'public_domain', body: '' }
const desert = { id: 1, name: '大漠', aliases: ['沙漠'], category: '地理', meaning: '', visual_zh: '无边沙丘', visual_en: 'dunes', emotions: ['壮阔'], seasons: [], times: [], weathers: [], eras: [], style_ids: [], status: 'published', created_at: '', updated_at: '', segment_count: 1 }
const sun = { ...desert, id: 2, name: '落日', aliases: ['夕阳'], category: '天象', visual_zh: '圆日低垂', status: 'draft' }
const ink = { id: 20, name: '水墨', category: '国画', desc_zh: '', prompt_en: '', negative_en: '', suits_emotions: [], suits_categories: [], options: {}, default_size: '1024x1536', default_quality: 'high', enabled: true, sort: 20 }
const recs = [
  { style: ink, score: 12.5, reasons: ['情绪「壮阔」'], options: { compositions: [{ name: '远景小人', prompt_en: 'x' }], palettes: [], moods: [{ name: '冷白月光', prompt_en: 'y' }] } },
  { style: { ...ink, id: 30, name: '版画' }, score: 0, reasons: [], options: {} }
]
const artwork = {
  id: 7, segment_id: 13, style_id: 20, prompt_final: 'Tall vertical…', model: 'gpt-image-2', size: '1024x1536', quality: 'high',
  status: 'succeeded', attempts: 1, error: '', width: 1024, height: 1536, cost_usd: 0.25, duration_ms: 54000, rating: 0,
  review_status: 'pending', reject_reason: '', text_overlay_allowed: true, created_at: '', imagery_ids: [1, 2], style_name: '水墨',
  text_id: 1, text_title: '使至塞上', segment_label: '第3联', url: '/api/v1/materials/files/artworks/0/7.png?e=1&s=x',
  thumb_url: '/api/v1/materials/files/artworks/0/7-thumb.jpg?e=1&s=x'
}

function setTab(tab: string) {
  history.replaceState(null, '', `/admin/materials?tab=${tab}`)
}

describe('admin MaterialsView', () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    setTab('texts')
    api.getQueue.mockResolvedValue({ queued: 2, running: 1, started_today: 3, concurrency: 2, daily_limit: 50, paused: '' })
    api.listTexts.mockResolvedValue({ items: [text], total: 1 })
    api.getText.mockResolvedValue({ ...text, segments: [segment] })
    api.getSegmentImagery.mockResolvedValue({ segment: segCtx, links: [] })
    api.listImagery.mockResolvedValue({ items: [desert, sun], total: 2, categories: ['地理', '天象'] })
    api.recommendStyles.mockResolvedValue(recs)
    api.listArtworks.mockResolvedValue({ items: [], total: 0 })
    api.listCandidates.mockResolvedValue([
      { rank: 1, name: '风', category: '气候', aliases: [], poems: 60000, in_library: true },
      { rank: 2, name: '山', category: '地理', aliases: [], poems: 59000, in_library: false }
    ])
  })

  it('shows the queue and why drawing is paused', async () => {
    api.getQueue.mockResolvedValue({ queued: 2, running: 0, started_today: 50, concurrency: 2, daily_limit: 50, paused: '今天已画 50 张' })
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.find('[data-testid="materials-queue"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="materials-paused"]').text()).toContain('materials.paused')
  })

  it('imports twice-checked works from 课本诗画', async () => {
    api.importShihua.mockResolvedValue({ imported: 1, existing: 0, skipped: [{ slug: 'x', title: '现代诗', reason: '未填作者卒年' }] })
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="materials-import"]').trigger('click')
    await flushPromises()
    expect(api.importShihua).toHaveBeenCalled()
    expect(api.listTexts).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('未填作者卒年')
  })

  it('works a segment from imagery to a drawing', async () => {
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="materials-text-shi-zhi-saishang"] button').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="materials-segment-13"]').trigger('click')
    await flushPromises()
    expect(api.getSegmentImagery).toHaveBeenCalledWith(13)
    expect(wrapper.get('[data-testid="materials-bench-content"]').text()).toBe('大漠孤烟直，长河落日圆。')
    expect(location.search).toContain('seg=13')

    // AI extraction fills the tags; the top style is picked.
    api.extractImagery.mockResolvedValue({
      segment: { ...segCtx, scene_desc: '黄昏的大漠' },
      links: [
        { segment_id: 13, imagery_id: 1, weight: 3, source: 'ai', confirmed: false, name: '大漠', category: '地理', status: 'published', visual_zh: '' },
        { segment_id: 13, imagery_id: 2, weight: 2, source: 'ai', confirmed: false, name: '落日', category: '天象', status: 'draft', visual_zh: '' }
      ],
      created: [], warnings: [], model: 'gpt'
    })
    await wrapper.get('[data-testid="materials-extract"]').trigger('click')
    await flushPromises()
    expect((wrapper.get('[data-testid="materials-bench-scene"]').element as HTMLTextAreaElement).value).toBe('黄昏的大漠')
    expect(wrapper.get('[data-testid="materials-tag-落日"]').text()).toContain('materials.bench.draftImagery')
    expect(wrapper.get('[data-testid="materials-style-20"]').classes()).toContain('ring-1')

    // Make 落日 the main subject and save: tags and scene go to the server.
    await wrapper.get('[data-testid="materials-tag-落日"] select').setValue(3)
    api.saveSegmentImagery.mockImplementation(async (_id, links) => ({
      segment: { ...segCtx, scene_desc: '黄昏的大漠' },
      links: links.map((l: { imagery_id: number; weight: number }) => ({ ...l, segment_id: 13, source: 'ai', confirmed: true, name: l.imagery_id === 1 ? '大漠' : '落日', category: '', status: 'published', visual_zh: '' })),
      created: [], warnings: [], model: ''
    }))
    await wrapper.get('[data-testid="materials-save-tags"]').trigger('click')
    await flushPromises()
    expect(api.saveSegmentImagery).toHaveBeenCalledWith(13, [{ imagery_id: 1, weight: 3 }, { imagery_id: 2, weight: 3 }], '黄昏的大漠')

    // Preview with options, then draw.
    await wrapper.get('[data-testid="materials-option-mood"]').setValue('冷白月光')
    api.buildPrompt.mockResolvedValue({ prompt: 'Tall vertical…\nCool silver moonlight', parts: {}, size: '1024x1536', quality: 'high', style_id: 20, imagery_ids: [1, 2], blocked: false, hits: [], notes: ['用到了还没审核发布的意象：落日'], text_overlay_allowed: true })
    await wrapper.get('[data-testid="materials-preview"]').trigger('click')
    await flushPromises()
    expect(api.buildPrompt).toHaveBeenCalledWith(13, { style_id: 20, size: '', composition: '', palette: '', mood: '冷白月光', extra: '' })
    expect(wrapper.get('[data-testid="materials-prompt"]').text()).toContain('Cool silver moonlight')
    expect(wrapper.get('[data-testid="materials-prompt"]').text()).toContain('落日')

    api.drawSegment.mockResolvedValue([artwork, { ...artwork, id: 8 }])
    await wrapper.get('[data-testid="materials-draw"]').trigger('click')
    await flushPromises()
    expect(api.drawSegment).toHaveBeenCalledWith(13, { style_id: 20, size: '', composition: '', palette: '', mood: '冷白月光', extra: '' }, 2)
    expect(api.getQueue).toHaveBeenCalledTimes(2)
  })

  it('blocks drawing when the preview hits the blocklist', async () => {
    history.replaceState(null, '', '/admin/materials?tab=texts&text=1&seg=13')
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    api.buildPrompt.mockResolvedValue({ prompt: 'p', parts: {}, size: '1024x1536', quality: 'high', style_id: 20, imagery_ids: [], blocked: true, hits: [{ term: 'ghibli', kind: 'studio', action: 'block', match: 'Ghibli', reason: '不模仿具体工作室' }], notes: [], text_overlay_allowed: true })
    await wrapper.get('[data-testid="materials-preview"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="materials-prompt"]').text()).toContain('Ghibli')
    expect(wrapper.get('[data-testid="materials-draw"]').attributes('disabled')).toBeDefined()
  })

  it('drafts seeds and publishes selected imagery', async () => {
    setTab('imagery')
    api.draftSeeds.mockResolvedValue({ drafted: ['山'], existing: 0, failed: [], warnings: [], model: 'gpt', left: 0 })
    api.setImageryStatus.mockResolvedValue(1)
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="materials-seed-progress"]').text()).toContain('materials.imagery.progress')
    await wrapper.get('[data-testid="materials-seed"]').trigger('click')
    await flushPromises()
    expect(api.draftSeeds).toHaveBeenCalledWith(30)

    await wrapper.get('[data-testid="materials-imagery-落日"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="materials-imagery-publish"]').trigger('click')
    await flushPromises()
    expect(api.setImageryStatus).toHaveBeenCalledWith([2], 'published')
  })

  it('keeps ticks made while the imagery list reloads', async () => {
    setTab('imagery')
    api.listStyles.mockResolvedValue([ink])
    api.updateImagery.mockResolvedValue(desert)
    api.setImageryStatus.mockResolvedValue(2)
    let release: () => void = () => {}
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="materials-imagery-edit-大漠"]').trigger('click')
    await flushPromises()
    // The reload after saving is slow; the admin ticks a box meanwhile.
    api.listImagery.mockImplementationOnce(() => new Promise((resolve) => (release = () => resolve({ items: [desert, sun], total: 2, categories: [] }))))
    await wrapper.get('[data-testid="materials-imagery-save"]').trigger('click')
    await wrapper.get('[data-testid="materials-imagery-落日"] input[type="checkbox"]').setValue(true)
    release()
    await flushPromises()
    await wrapper.get('[data-testid="materials-imagery-大漠"] input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="materials-imagery-publish"]').trigger('click')
    await flushPromises()
    expect(api.setImageryStatus).toHaveBeenCalledWith([2, 1], 'published')
  })

  it('edits an imagery entry with comma-separated lists', async () => {
    setTab('imagery')
    api.listStyles.mockResolvedValue([ink])
    api.updateImagery.mockResolvedValue(desert)
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="materials-imagery-edit-大漠"]').trigger('click')
    await flushPromises()
    const form = wrapper.get('[data-testid="materials-imagery-form"]')
    const inputs = form.findAll('input:not([type="checkbox"])')
    await inputs[1].setValue('沙漠， 瀚海 、黄沙')
    await wrapper.get('[data-testid="materials-imagery-save"]').trigger('click')
    await flushPromises()
    expect(api.updateImagery).toHaveBeenCalledWith(1, expect.objectContaining({ name: '大漠', aliases: ['沙漠', '瀚海', '黄沙'], emotions: ['壮阔'] }))
  })

  it('reviews artworks: approve, rate and reject with a reason', async () => {
    setTab('artworks')
    api.listArtworks.mockResolvedValue({ items: [artwork], total: 1 })
    api.reviewArtwork.mockResolvedValue(artwork)
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="materials-artwork-7"]').text()).toContain('使至塞上')

    await wrapper.get('[data-testid="materials-approve-7"]').trigger('click')
    await flushPromises()
    expect(api.reviewArtwork).toHaveBeenLastCalledWith(7, { review_status: 'approved', rating: 0, reject_reason: '' })

    await wrapper.get('[data-testid="materials-rate-7-4"]').trigger('click')
    await flushPromises()
    expect(api.reviewArtwork).toHaveBeenLastCalledWith(7, { review_status: 'pending', rating: 4, reject_reason: '' })

    vi.spyOn(window, 'prompt').mockReturnValue('画面里有字')
    await wrapper.get('[data-testid="materials-reject-7"]').trigger('click')
    await flushPromises()
    expect(api.reviewArtwork).toHaveBeenLastCalledWith(7, { review_status: 'rejected', rating: 0, reject_reason: '画面里有字' })
  })

  it('checks a prompt against the blocklist', async () => {
    setTab('rules')
    api.listStyles.mockResolvedValue([ink])
    api.listBlocklist.mockResolvedValue([{ id: 1, term: '(?i)ghibli', kind: 'studio', match: 'regex', action: 'block', applies_to: 'prompt', reason: '不模仿具体工作室', enabled: true }])
    api.checkPrompt.mockResolvedValue({ blocked: true, hits: [{ term: '(?i)ghibli', kind: 'studio', action: 'block', match: 'Ghibli', reason: '' }], prompt: 'a Ghibli village' })
    const wrapper = mount(MaterialsView, { global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="materials-rule-style-20"]').text()).toContain('水墨')
    await wrapper.get('[data-testid="materials-check-input"]').setValue('a Ghibli village')
    await wrapper.get('[data-testid="materials-check-run"]').trigger('click')
    await flushPromises()
    expect(api.checkPrompt).toHaveBeenCalledWith('a Ghibli village', 'prompt')
    expect(wrapper.get('[data-testid="materials-check-result"]').text()).toContain('materials.rules.checkBlocked')
  })

  describe('layout and export', () => {
    beforeEach(() => {
      setTab('layout')
      for (const fn of Object.values(layoutUtils)) fn.mockReset()
      layoutUtils.loadShihuaArt.mockResolvedValue({ width: 1024, height: 1536 })
      layoutUtils.loadShihuaFont.mockResolvedValue(true)
      layoutUtils.canvasToBlob.mockResolvedValue(new Blob(['x']))
      layoutUtils.renderShihuaPoster.mockImplementation(async (input: { aspect: string; text: string; byline?: string }) => ({
        canvas: {},
        layout: { aspect: input.aspect, width: 100, height: 150, box: { x0: 0, y0: 0, x1: 1, y1: 1 }, cell: 10, columns: [], color: 'ink', text: input.text, font: 'mashanzheng', byline: input.byline ?? '' }
      }))
      globalThis.URL.createObjectURL = vi.fn(() => 'blob:preview')
      globalThis.URL.revokeObjectURL = vi.fn()
      api.listArtworks.mockResolvedValue({ items: [{ ...artwork, review_status: 'approved' }], total: 1 })
      api.listRenders.mockResolvedValue([])
      api.saveRender.mockResolvedValue({})
    })

    it('sets the segment and the byline and saves every aspect with its words', async () => {
      const wrapper = mount(MaterialsView, { global: { stubs } })
      await flushPromises()
      expect(api.listArtworks).toHaveBeenCalledWith({ review_status: 'approved', status: 'succeeded', limit: 200 })
      await wrapper.get('[data-testid="materials-layout-7"]').trigger('click')
      await flushPromises()
      const textarea = wrapper.get('[data-testid="materials-layout-text"]').element as HTMLTextAreaElement
      expect(textarea.value).toBe('大漠孤烟直，长河落日圆。')
      expect(textarea.disabled).toBe(false)
      expect(layoutUtils.renderShihuaPoster).toHaveBeenCalledWith(
        expect.objectContaining({ text: '大漠孤烟直，长河落日圆。', byline: '王维《使至塞上》' }),
        expect.anything()
      )

      await wrapper.get('[data-testid="materials-layout-save"]').trigger('click')
      await flushPromises()
      expect(api.saveRender).toHaveBeenCalledTimes(6)
      expect(api.saveRender).toHaveBeenCalledWith(7, '2x3', expect.any(Blob), expect.objectContaining({ text: '大漠孤烟直，长河落日圆。', byline: '王维《使至塞上》', label: 'AI 生成' }))
    })

    it('keeps original words off a reference-only picture', async () => {
      api.getSegmentImagery.mockResolvedValue({
        segment: { ...segCtx, id: 31, label: '副歌', content: '', scene_desc: '雨夜的街口', kind: 'song', title: '测试歌曲', author_name: '测试词作者', dynasty: '当代', license_status: 'reference_only' },
        links: []
      })
      const wrapper = mount(MaterialsView, { global: { stubs } })
      await flushPromises()
      await wrapper.get('[data-testid="materials-layout-7"]').trigger('click')
      await flushPromises()
      const textarea = wrapper.get('[data-testid="materials-layout-text"]').element as HTMLTextAreaElement
      expect(textarea.disabled).toBe(true)
      expect(wrapper.get('[data-testid="materials-layout-editor"]').text()).toContain('materials.layout.refOnly')
      await wrapper.get('[data-testid="materials-layout-save"]').trigger('click')
      await flushPromises()
      expect(api.saveRender).toHaveBeenCalledWith(7, '3x4', expect.any(Blob), expect.objectContaining({ text: '', byline: '测试词作者《测试歌曲》' }))
    })

    it('exports the ticked artworks in the chosen aspects', async () => {
      api.exportZip.mockResolvedValue(new Blob(['zip']))
      const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
      const wrapper = mount(MaterialsView, { global: { stubs } })
      await flushPromises()
      await wrapper.get('[data-testid="materials-layout-7"] input[type="checkbox"]').trigger('change')
      await wrapper.get('[data-testid="materials-export"]').trigger('click')
      await flushPromises()
      expect(api.exportZip).toHaveBeenCalledWith([7], ['2x3', '3x4', '9x16'])
      expect(click).toHaveBeenCalled()
    })
  })
})
