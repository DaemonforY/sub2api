import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { _resetAnalytics, captureAttribution, flush, initAnalytics, signupAttribution, track, trackPageView } from '@/utils/analytics'

describe('analytics', () => {
  const fetchMock = vi.fn()

  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    _resetAnalytics()
    fetchMock.mockReset()
    fetchMock.mockResolvedValue({ status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('keeps the first touch and ignores later campaigns', () => {
    const first = captureAttribution('https://hivegpt.cn/register?aff=INV1&utm_source=poster&utm_medium=edu', 'https://www.v2ex.com/t/1')
    expect(first).toMatchObject({ source: 'poster', medium: 'edu', aff: 'INV1', landing: '/register', referrer: 'https://www.v2ex.com/t/1' })
    const later = captureAttribution('https://hivegpt.cn/home?utm_source=other', '')
    expect(later.source).toBe('poster')
    expect(signupAttribution().attribution.aff).toBe('INV1')
    expect(signupAttribution().visitor_id).toMatch(/^[a-f0-9]{24}$/)
  })

  it('does not treat a same-site referrer as a source', () => {
    expect(captureAttribution('https://hivegpt.cn/keys', 'https://hivegpt.cn/home').referrer).toBe('')
  })

  it('batches events and sends them with the visitor, session and first touch', async () => {
    initAnalytics('main')
    localStorage.setItem('auth_token', 'tok')
    track('page_view', undefined, '/home')
    track('key_created', { n: 1 }, '/keys')
    expect(fetchMock).not.toHaveBeenCalled()
    vi.advanceTimersByTime(3000)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/events')
    expect(init.keepalive).toBe(true)
    expect(init.headers.Authorization).toBe('Bearer tok')
    const body = JSON.parse(init.body)
    expect(body.app).toBe('main')
    expect(body.events).toEqual([{ name: 'page_view', path: '/home' }, { name: 'key_created', path: '/keys', props: { n: 1 } }])
    expect(body.visitor_id).toBe(localStorage.getItem('hg_vid'))
    expect(body.session_id).toBeTruthy()
  })

  it('resends anonymously when the token has expired', async () => {
    localStorage.setItem('auth_token', 'expired')
    fetchMock.mockResolvedValueOnce({ status: 401 }).mockResolvedValueOnce({ status: 204 })
    track('page_view', undefined, '/')
    flush()
    await vi.runAllTimersAsync()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock.mock.calls[1][1].headers.Authorization).toBeUndefined()
  })

  it('flushes when the page is hidden and counts canvas links', () => {
    initAnalytics('main')
    const a = document.createElement('a')
    a.href = 'https://canvas.hivegpt.cn/image?prompt=x'
    a.addEventListener('click', (e) => e.preventDefault())
    document.body.appendChild(a)
    a.click()
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })
    document.dispatchEvent(new Event('visibilitychange'))
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const body = JSON.parse(fetchMock.mock.calls[0][1].body)
    expect(body.events[0]).toMatchObject({ name: 'canvas_click', props: { to: '/image' } })
  })

  it('ignores the same page view fired twice on load', () => {
    trackPageView('/learn/')
    trackPageView('/learn/')
    trackPageView('/learn/a/')
    flush()
    const body = JSON.parse(fetchMock.mock.calls[0][1].body)
    expect(body.events.map((e: { path: string }) => e.path)).toEqual(['/learn/', '/learn/a/'])
  })
})
