/**
 * Copy of frontend/src/utils/analytics.ts for the learning site — same storage keys and payload,
 * so a visitor is one visitor across hivegpt.cn, /learn and /editor. Keep the two in sync.
 *
 * First-party usage analytics (埋点). Events are queued and posted in small batches to
 * /api/v1/events on this site — no third-party scripts, no cookies.
 *
 * - visitor id: random, in localStorage (shared with /learn and /editor, same origin)
 * - first touch: utm_*, invite code, referrer and landing page of the first visit, kept 90 days
 *   and sent with every batch and with the sign-up request, so registrations and payments can be
 *   traced to the poster / invite / site that brought the visitor
 * - never send input text, prompts or keys as props
 */

const VISITOR_KEY = 'hg_vid'
const ATTR_KEY = 'hg_attr'
const SESSION_KEY = 'hg_sid'
const ATTR_TTL_MS = 90 * 24 * 3600 * 1000
const FLUSH_MS = 3000
const MAX_BATCH = 20

export type AnalyticsApp = 'main' | 'learn' | 'editor'
export type AnalyticsProps = Record<string, string | number | boolean>

export interface AnalyticsAttribution {
  source: string
  medium: string
  campaign: string
  aff: string
  referrer: string
  landing: string
  first_seen: number
}

interface QueuedEvent {
  name: string
  path: string
  props?: AnalyticsProps
}

let app: AnalyticsApp = 'learn'
let queue: QueuedEvent[] = []
let timer: ReturnType<typeof setTimeout> | null = null
let started = false

function randomId(): string {
  const bytes = new Uint8Array(12)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

function storage(kind: 'local' | 'session'): Storage | null {
  try {
    return kind === 'local' ? window.localStorage : window.sessionStorage
  } catch {
    return null
  }
}

export function visitorId(): string {
  const ls = storage('local')
  let id = ls?.getItem(VISITOR_KEY) || ''
  if (!/^[A-Za-z0-9_-]{8,64}$/.test(id)) {
    id = randomId()
    ls?.setItem(VISITOR_KEY, id)
  }
  return id
}

function sessionId(): string {
  const ss = storage('session')
  let id = ss?.getItem(SESSION_KEY) || ''
  if (!id) {
    id = randomId()
    ss?.setItem(SESSION_KEY, id)
  }
  return id
}

/** Records the first touch once (or again after 90 days); later visits keep the original. */
export function captureAttribution(href = window.location.href, referrer = document.referrer): AnalyticsAttribution {
  const ls = storage('local')
  try {
    const saved = JSON.parse(ls?.getItem(ATTR_KEY) || 'null') as AnalyticsAttribution | null
    if (saved && Date.now() - saved.first_seen < ATTR_TTL_MS) return saved
  } catch {
    // corrupted: start over
  }
  const url = new URL(href)
  const q = url.searchParams
  let ref = ''
  try {
    if (referrer && new URL(referrer).host !== url.host) ref = referrer.slice(0, 300)
  } catch {
    ref = ''
  }
  const attr: AnalyticsAttribution = {
    source: (q.get('utm_source') || '').slice(0, 64),
    medium: (q.get('utm_medium') || '').slice(0, 64),
    campaign: (q.get('utm_campaign') || '').slice(0, 64),
    aff: (q.get('aff') || q.get('aff_code') || '').slice(0, 64),
    referrer: ref,
    landing: url.pathname,
    first_seen: Date.now()
  }
  ls?.setItem(ATTR_KEY, JSON.stringify(attr))
  return attr
}

/** Extra fields for the sign-up request. */
export function signupAttribution(): { visitor_id: string; attribution: AnalyticsAttribution } {
  return { visitor_id: visitorId(), attribution: captureAttribution() }
}

function authHeader(): Record<string, string> {
  const token = storage('local')?.getItem('auth_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function post(events: QueuedEvent[], withAuth = true): Promise<void> {
  const body = JSON.stringify({
    visitor_id: visitorId(),
    session_id: sessionId(),
    app,
    attr: captureAttribution(),
    events
  })
  const res = await fetch('/api/v1/events', {
    method: 'POST',
    keepalive: true,
    headers: { 'Content-Type': 'application/json', ...(withAuth ? authHeader() : {}) },
    body
  })
  // An expired token must not lose the batch: send it again anonymously.
  if (res.status === 401 && withAuth) await post(events, false)
}

export function flush(): void {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
  while (queue.length) {
    const batch = queue.splice(0, MAX_BATCH)
    post(batch).catch(() => {
      // analytics must never break the page
    })
  }
}

/** Queues an event; batches go out every few seconds and when the page is hidden. */
export function track(name: string, props?: AnalyticsProps, path = window.location.pathname): void {
  if (typeof window === 'undefined' || navigator.webdriver) return
  queue.push(props ? { name, path, props } : { name, path })
  if (queue.length >= MAX_BATCH) flush()
  else if (!timer) timer = setTimeout(flush, FLUSH_MS)
}

let lastView = { path: '', at: 0 }

/** Counts a page view; the same path again within 2 s (a router firing twice on load) is ignored. */
export function trackPageView(path = window.location.pathname): void {
  const now = Date.now()
  if (path === lastView.path && now - lastView.at < 2000) return
  lastView = { path, at: now }
  track('page_view', undefined, path)
}

/** Call once at start-up. */
export function initAnalytics(name: AnalyticsApp = 'main'): void {
  if (started || typeof window === 'undefined') return
  started = true
  app = name
  captureAttribution()
  // Links out to the canvas (nav, prompt box, key dialog, learn pages …): one listener for all.
  document.addEventListener(
    'click',
    (e) => {
      const a = (e.target as Element | null)?.closest?.('a[href]') as HTMLAnchorElement | null
      if (a && /^canvas\./.test(a.hostname) && a.hostname !== window.location.hostname) {
        track('canvas_click', { to: a.pathname.slice(0, 60) })
      }
    },
    true
  )
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') flush()
  })
  window.addEventListener('pagehide', flush)
}

