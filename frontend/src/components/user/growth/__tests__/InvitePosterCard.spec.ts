import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import InvitePosterCard from '../InvitePosterCard.vue'

const showSuccess = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showWarning, cachedPublicSettings: null, siteName: 'HiveGPT' })
}))
vi.mock('@/utils/invitePosterDraw', () => ({ drawInvitePoster: vi.fn().mockResolvedValue(undefined) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }) }
})

// "PNG" in base64.
const DATA_URL = 'data:image/png;base64,UE5H'

describe('InvitePosterCard', () => {
  const write = vi.fn().mockResolvedValue(undefined)
  const fetchSpy = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue(DATA_URL)
    vi.stubGlobal('fetch', fetchSpy)
    vi.stubGlobal('ClipboardItem', class { constructor(public items: Record<string, Blob>) {} })
    Object.defineProperty(navigator, 'clipboard', { value: { write }, configurable: true })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('copies the poster without fetching the data URL (blocked by the CSP connect-src)', async () => {
    const wrapper = mount(InvitePosterCard, {
      props: { inviteLink: 'https://hivegpt.cn/r/abc', affCode: 'abc', rebateRate: 10, inviteeBonusRate: 20, inviteeBonusCap: 10, eduDiscount: 0 },
      global: { stubs: { Icon: true } }
    })
    await flushPromises()

    await wrapper.findAll('button.btn-secondary')[0].trigger('click')
    await flushPromises()

    expect(fetchSpy).not.toHaveBeenCalled()
    expect(write).toHaveBeenCalledTimes(1)
    const blob = (write.mock.calls[0][0][0] as { items: Record<string, Blob> }).items['image/png']
    expect(blob.type).toBe('image/png')
    expect(blob.size).toBe(3)
    expect(showSuccess).toHaveBeenCalledWith('affiliatePoster.copied')
    expect(showWarning).not.toHaveBeenCalled()
  })
})
