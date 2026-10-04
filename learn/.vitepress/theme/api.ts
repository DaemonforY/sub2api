// Talks to the main site's API from /learn (same origin): the login token the main site keeps in
// localStorage, lesson progress (kept in the browser when signed out, merged into the account on
// sign-in) and example runs.
import { reactive } from 'vue'

const TOKEN_KEY = 'auth_token'
const USER_KEY = 'auth_user'
const EXPIRES_KEY = 'token_expires_at'
const LOCAL_PROGRESS_KEY = 'learn_progress'

export interface LearnUser {
  id: number
  username?: string
  email?: string
  balance?: number
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public reason = '',
  ) {
    super(message)
  }
}

const inBrowser = typeof window !== 'undefined'

export function token(): string {
  if (!inBrowser) return ''
  const expires = Number(localStorage.getItem(EXPIRES_KEY) || 0)
  if (expires && expires < Date.now()) return ''
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function currentUser(): LearnUser | null {
  if (!inBrowser || !token()) return null
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null')
  } catch {
    return null
  }
}

/** The main site's sign-in page, coming back to this page afterwards. */
export function loginUrl(): string {
  const back = inBrowser ? window.location.pathname + window.location.search + window.location.hash : '/learn/'
  return `/login?redirect=${encodeURIComponent(back)}`
}

async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (init.body) headers['Content-Type'] = 'application/json'
  const t = token()
  if (t) headers.Authorization = `Bearer ${t}`
  let res: Response
  try {
    res = await fetch(`/api/v1${path}`, { ...init, headers: { ...headers, ...(init.headers as Record<string, string>) } })
  } catch {
    throw new ApiError('网络连接失败，请稍后再试', 0)
  }
  let body: { code?: number; message?: string; reason?: string; data?: T } = {}
  try {
    body = await res.json()
  } catch {
    // not JSON
  }
  if (!res.ok || (body.code !== undefined && body.code !== 0)) {
    throw new ApiError(body.message || `请求失败（HTTP ${res.status}）`, res.status, body.reason || '')
  }
  return body.data as T
}

// --- Progress -------------------------------------------------------------------------------------

export const progress = reactive<{ completed: Record<string, string>; loaded: boolean; runsLeft: number | null }>({
  completed: {},
  loaded: false,
  runsLeft: null,
})

function readLocal(): Record<string, string> {
  try {
    return JSON.parse(localStorage.getItem(LOCAL_PROGRESS_KEY) || '{}') || {}
  } catch {
    return {}
  }
}

function writeLocal(done: Record<string, string>) {
  localStorage.setItem(LOCAL_PROGRESS_KEY, JSON.stringify(done))
}

let loading: Promise<void> | null = null

/** Loads progress once per page load: local, plus the account's when signed in (local merged in). */
export function loadProgress(): Promise<void> {
  if (!inBrowser) return Promise.resolve()
  if (loading) return loading
  loading = (async () => {
    const local = readLocal()
    progress.completed = { ...local }
    if (token()) {
      try {
        const localIds = Object.keys(local)
        if (localIds.length) {
          await api('/learn/progress', { method: 'POST', body: JSON.stringify({ lesson_ids: localIds }) })
          localStorage.removeItem(LOCAL_PROGRESS_KEY)
        }
        const me = await api<{ completed: Record<string, string>; runs_left: number }>('/learn/me')
        progress.completed = me.completed || {}
        progress.runsLeft = me.runs_left
      } catch {
        // signed-out view of local progress
      }
    }
    progress.loaded = true
  })()
  return loading
}

export async function markDone(lessonId: string): Promise<void> {
  const now = new Date().toISOString()
  progress.completed = { ...progress.completed, [lessonId]: progress.completed[lessonId] || now }
  if (token()) {
    try {
      const res = await api<{ completed: Record<string, string> }>('/learn/progress', { method: 'POST', body: JSON.stringify({ lesson_ids: [lessonId] }) })
      progress.completed = res.completed
      return
    } catch {
      // fall back to the browser
    }
  }
  writeLocal({ ...readLocal(), [lessonId]: now })
}

// --- Runs -----------------------------------------------------------------------------------------

export interface RunMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface RunResult {
  content: string
  tool_calls?: unknown
  finish_reason: string
  model: string
  prompt_tokens: number
  completion_tokens: number
  latency_ms: number
  runs_left: number
}

export interface LearnConfig {
  run_enabled: boolean
  free_runs_per_day: number
  model: string
}

let configPromise: Promise<LearnConfig> | null = null
export function learnConfig(): Promise<LearnConfig> {
  configPromise ||= api<LearnConfig>('/learn/config').catch(() => ({ run_enabled: false, free_runs_per_day: 0, model: '' }))
  return configPromise
}

export async function runExample(input: { lesson: string; messages: RunMessage[]; response_format?: unknown; tools?: unknown }): Promise<RunResult> {
  const result = await api<RunResult>('/learn/run', { method: 'POST', body: JSON.stringify(input) })
  progress.runsLeft = result.runs_left
  return result
}
