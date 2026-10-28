import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ShihuaLayoutPanel from '../ShihuaLayoutPanel.vue'

const api = vi.hoisted(() => ({ listImages: vi.fn(), listRenders: vi.fn(), saveRender: vi.fn(), exportZip: vi.fn() }))
const layout = vi.hoisted(() => ({ loadShihuaFont: vi.fn(), loadShihuaArt: vi.fn(), renderShihuaPoster: vi.fn(), canvasToBlob: vi.fn() }))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin/shihua', () => api)
vi.mock('@/utils/shihuaLayout', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/utils/shihuaLayout')>()), ...layout }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p?.titles ? `${key} ${p.titles}` : key) }) }))

function work(id: number, slug: string, title: string, status: string) {
  return { id, slug, title, author: '某', dynasty: '唐', full_text: '床前明月光，', display_text: '', famous_lines: '', text_status: status, picked_image_id: id * 10, renders: 0 }
}

describe('ShihuaLayoutPanel export', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.listImages.mockImplementation(async ({ work_ids }: { work_ids: string }) => [{ id: Number(work_ids) * 10, work_id: Number(work_ids), url: '/o.png', title_drawn: true }])
    // Work 1 already has its 2:3 poster.
    api.listRenders.mockResolvedValue([{ image_id: 10, work_id: 1, aspect: '2x3', layout: {} }])
    api.saveRender.mockResolvedValue({})
    api.exportZip.mockResolvedValue(new Blob(['zip']))
    layout.loadShihuaFont.mockResolvedValue(true)
    layout.loadShihuaArt.mockResolvedValue({})
    layout.renderShihuaPoster.mockImplementation(async (_in: unknown, _o: unknown) => ({ canvas: {}, layout: { aspect: 'x' } }))
    layout.canvasToBlob.mockResolvedValue(new Blob())
    vi.stubGlobal('URL', { ...URL, createObjectURL: () => 'blob:1', revokeObjectURL: vi.fn() })
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })

  it('lays out the missing posters before exporting and names the unchecked works', async () => {
    const works = [work(1, 'jing-ye-si', '静夜思', 'verified_2'), work(2, 'sanxia', '三峡', 'verified_1')]
    const wrapper = mount(ShihuaLayoutPanel, { props: { works: works as never } })
    await wrapper.get('h3 + input[type="checkbox"]').trigger('change')
    await wrapper.get('[data-testid="shihua-export"]').trigger('click')
    await flushPromises()

    // Default export sizes 2:3, 3:4, 9:16: only the two missing ones of the checked-twice work.
    expect(api.saveRender.mock.calls.map((c) => [c[0], c[1]])).toEqual([[10, '3x4'], [10, '9x16']])
    expect(api.exportZip).toHaveBeenCalledWith([1, 2], ['2x3', '3x4', '9x16'])
    expect(showError).toHaveBeenCalledWith('shihua.layout.skippedUnverified 《三峡》')
  })
})
