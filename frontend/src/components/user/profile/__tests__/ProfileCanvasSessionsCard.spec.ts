import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))
const showSuccess = vi.fn()
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess }) }))
const listCanvasSessions = vi.fn()
const revokeCanvasSession = vi.fn()
vi.mock('@/api/canvasSessions', async () => {
  const actual = await vi.importActual<typeof import('@/api/canvasSessions')>('@/api/canvasSessions')
  return { ...actual, listCanvasSessions: () => listCanvasSessions(), revokeCanvasSession: (id: number) => revokeCanvasSession(id) }
})

import ProfileCanvasSessionsCard from '../ProfileCanvasSessionsCard.vue'
import { describeUserAgent } from '@/api/canvasSessions'

describe('ProfileCanvasSessionsCard', () => {
  beforeEach(() => {
    listCanvasSessions.mockReset()
    revokeCanvasSession.mockReset().mockResolvedValue(undefined)
  })

  it('lists signed-in devices and signs one out', async () => {
    const ua = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130 Safari/537.36'
    listCanvasSessions.mockResolvedValue([
      { id: 1, user_agent: ua, ip: '1.2.3.4', created_at: '2026-10-01T00:00:00Z', last_used_at: '2026-10-02T00:00:00Z', expires_at: '2026-11-01T00:00:00Z' },
      { id: 2, user_agent: '', ip: '', created_at: '2026-10-01T00:00:00Z', last_used_at: '2026-10-01T00:00:00Z', expires_at: '2026-11-01T00:00:00Z' }
    ])
    const w = mount(ProfileCanvasSessionsCard)
    await flushPromises()
    const rows = w.findAll('[data-testid="canvas-session"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('Chrome · macOS')
    expect(rows[1].text()).toContain('profile.canvasSessions.unknownDevice')
    await rows[0].get('button').trigger('click')
    await flushPromises()
    expect(revokeCanvasSession).toHaveBeenCalledWith(1)
    expect(w.findAll('[data-testid="canvas-session"]')).toHaveLength(1)
    expect(showSuccess).toHaveBeenCalled()
  })

  it('describes common browsers', () => {
    expect(describeUserAgent('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1')).toBe('Safari · iOS')
    expect(describeUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130 Safari/537.36 Edg/130')).toBe('Edge · Windows')
    expect(describeUserAgent('')).toBe('')
  })
})
