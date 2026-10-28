import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ShihuaPickPanel from '../ShihuaPickPanel.vue'

const api = vi.hoisted(() => ({ listImages: vi.fn(), reviewImage: vi.fn(), redraw: vi.fn() }))
const layout = vi.hoisted(() => ({ loadShihuaArt: vi.fn(), renderShihuaPoster: vi.fn(), canvasToBlob: vi.fn() }))

vi.mock('@/api/admin/shihua', () => api)
vi.mock('@/utils/shihuaLayout', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/utils/shihuaLayout')>()), ...layout }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { BaseDialog: true, Icon: true }

const work = {
  id: 1, slug: 'cun-ju', title: '村居', author: '高鼎', dynasty: '清', grade: '二下', full_text: '草长莺飞二月天，\n拂堤杨柳醉春烟。', display_text: '',
  famous_lines: '', theme: { prompt_theme: '早春' }, text_status: 'verified_2'
}
function image(id: number, titleDrawn: boolean) {
  return { id, work_id: 1, status: 'succeeded', title_drawn: titleDrawn, thumb_url: `/t/${id}.jpg`, url: `/o/${id}.png`, duration_ms: 1000, cost_usd: 0.2, upscale_status: '', title_check: '', rejected: false }
}

describe('ShihuaPickPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.listImages.mockResolvedValue([image(7, false), image(8, true)])
    layout.loadShihuaArt.mockResolvedValue({ width: 10, height: 15 })
    layout.renderShihuaPoster.mockResolvedValue({ canvas: {}, layout: {} })
    layout.canvasToBlob.mockResolvedValue(new Blob())
    let n = 0
    vi.stubGlobal('URL', { ...URL, createObjectURL: () => `blob:${++n}`, revokeObjectURL: vi.fn() })
  })

  it('shows each drawing with the poem laid out, setting the title only where the model drew none', async () => {
    const wrapper = mount(ShihuaPickPanel, { props: { works: [work as never] }, global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="shihua-pick-thumb-7"]').attributes('src')).toBe('/t/7.jpg')
    expect(layout.renderShihuaPoster).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="shihua-pick-view-poster"]').trigger('click')
    await flushPromises()
    const calls = layout.renderShihuaPoster.mock.calls
    expect(calls).toHaveLength(2)
    expect(calls[0][0]).toMatchObject({ aspect: '2x3', title: '村居', withTitle: true, byline: '〔清〕高鼎', text: work.full_text })
    expect(calls[1][0]).toMatchObject({ withTitle: false, byline: '' })
    expect(calls[0][1]).toEqual({ font: 'mashanzheng' })
    expect(wrapper.get('[data-testid="shihua-pick-thumb-7"]').attributes('src')).toMatch(/^blob:/)

    // Another font renders again; going back shows the plain pictures.
    await wrapper.get('[data-testid="shihua-pick-font"]').setValue('liujianmaocao')
    await flushPromises()
    expect(calls).toHaveLength(4)
    expect(calls[3][1]).toEqual({ font: 'liujianmaocao' })
    await wrapper.get('[data-testid="shihua-pick-view-art"]').trigger('click')
    expect(wrapper.get('[data-testid="shihua-pick-thumb-7"]').attributes('src')).toBe('/t/7.jpg')
  })
})
