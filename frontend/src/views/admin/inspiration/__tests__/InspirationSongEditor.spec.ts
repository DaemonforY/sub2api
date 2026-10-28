import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import InspirationSongEditor from '../InspirationSongEditor.vue'

const api = vi.hoisted(() => ({ getSong: vi.fn(), createSong: vi.fn(), updateSong: vi.fn(), analyzeSong: vi.fn(), deleteSong: vi.fn() }))
const { showSuccess, showError } = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/admin/inspiration', () => api)
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = {
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  ConfirmDialog: true
}

function song(id: number, extra = {}) {
  return {
    id, title: '测试歌', artist: '歌手', lyricist: '', composer: '', year: null, album: '', lyrics: '一行歌词', has_lyrics: true,
    summary: '', imagery: [], moods: [], tags: [], allusions: [], notes: '', link_url: '', starred: false, created_at: '', updated_at: '', ...extra
  }
}

describe('InspirationSongEditor', () => {
  beforeEach(() => vi.clearAllMocks())

  it('saves a new song, then fills the card from the AI analysis', async () => {
    api.createSong.mockResolvedValue(song(5))
    api.updateSong.mockResolvedValue(song(5))
    api.analyzeSong.mockResolvedValue({
      summary: '', imagery: ['青石板', '雨巷'], moods: ['怀旧'], tags: ['中国风'], model: 'm', dropped: ['summary'],
      allusions: [{ title: '村居', author: '高鼎', poem_id: 1, note: '放风筝' }]
    })
    const wrapper = mount(InspirationSongEditor, { props: { show: true, songId: null }, global: { stubs } })
    await flushPromises()
    expect(wrapper.get('[data-testid="song-analyze"]').attributes('disabled')).toBeDefined()

    await wrapper.get('[data-testid="song-title"]').setValue(' 测试歌 ')
    await wrapper.get('[data-testid="song-lyrics"]').setValue('一行歌词')
    await wrapper.get('[data-testid="song-imagery"]').setValue('青石板，雨巷、 ')
    await wrapper.get('[data-testid="song-save"]').trigger('click')
    await flushPromises()
    expect(api.createSong.mock.calls[0][0]).toMatchObject({ lyrics: '一行歌词', imagery: ['青石板', '雨巷'], year: null })
    expect(wrapper.emitted('saved')?.[0]).toEqual([song(5)])

    // The page passes the new id back; the form keeps what was typed.
    await wrapper.setProps({ songId: 5 })
    await flushPromises()
    expect(api.getSong).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="song-analyze"]').trigger('click')
    await flushPromises()
    expect(api.updateSong).toHaveBeenCalledWith(5, expect.objectContaining({ lyrics: '一行歌词' }))
    expect((wrapper.get('[data-testid="song-imagery"]').element as HTMLInputElement).value).toBe('青石板、雨巷')
    expect(wrapper.text()).toContain('inspiration.editor.dropped')
  })

  it('loads an existing song with its lyrics', async () => {
    api.getSong.mockResolvedValue(song(9, { imagery: ['月'], year: 2003 }))
    const wrapper = mount(InspirationSongEditor, { props: { show: true, songId: 9 }, global: { stubs } })
    await flushPromises()
    expect((wrapper.get('[data-testid="song-lyrics"]').element as HTMLTextAreaElement).value).toBe('一行歌词')
    expect((wrapper.get('[data-testid="song-imagery"]').element as HTMLInputElement).value).toBe('月')
  })
})
