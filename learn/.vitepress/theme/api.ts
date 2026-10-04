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

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
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

export interface QuizResult {
  correct: number
  total: number
  attempts: number
}

export interface Certificate {
  code: string
  track: string
  track_title: string
  display_name: string
  project_url: string
  quiz_score: number
  issued_at: string
  invite_code?: string
}

interface MeResponse {
  completed: Record<string, string>
  runs_left: number
  tutor_left: number
  interviews_left: number
  quizzes: Record<string, QuizResult>
  checkpoints: Record<string, string>
  certificates: Certificate[]
}

export const progress = reactive<{
  completed: Record<string, string>
  loaded: boolean
  runsLeft: number | null
  tutorLeft: number | null
  interviewsLeft: number | null
  quizzes: Record<string, QuizResult>
  checkpoints: Record<string, string>
  certificates: Certificate[]
}>({
  completed: {},
  loaded: false,
  runsLeft: null,
  tutorLeft: null,
  interviewsLeft: null,
  quizzes: {},
  checkpoints: {},
  certificates: [],
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

function applyMe(me: MeResponse) {
  progress.completed = me.completed || {}
  progress.runsLeft = me.runs_left
  progress.tutorLeft = me.tutor_left
  progress.interviewsLeft = me.interviews_left
  progress.quizzes = me.quizzes || {}
  progress.checkpoints = me.checkpoints || {}
  progress.certificates = me.certificates || []
}

/** Reloads the signed-in learner's state (after a quiz, a checkpoint, a certificate). */
export async function refreshMe(): Promise<void> {
  if (!token()) return
  try {
    applyMe(await api<MeResponse>('/learn/me'))
  } catch {
    // keep what we have
  }
}

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
        applyMe(await api<MeResponse>('/learn/me'))
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
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  tool_calls?: unknown
  tool_call_id?: string
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
  own_key?: boolean
}

export interface LearnConfig {
  run_enabled: boolean
  free_runs_per_day: number
  tutor_free_per_day: number
  interviews_per_day: number
  model: string
}

let configPromise: Promise<LearnConfig> | null = null
export function learnConfig(): Promise<LearnConfig> {
  configPromise ||= api<LearnConfig>('/learn/config').catch(() => ({ run_enabled: false, free_runs_per_day: 0, tutor_free_per_day: 0, interviews_per_day: 0, model: '' }))
  return configPromise
}

export async function runExample(input: { lesson: string; messages: RunMessage[]; response_format?: unknown; tools?: unknown }): Promise<RunResult> {
  const keyId = progress.runsLeft === 0 ? ownKey.id : 0
  const result = await api<RunResult>('/learn/run', { method: 'POST', body: JSON.stringify({ ...input, key_id: keyId || undefined }) })
  progress.runsLeft = result.runs_left
  return result
}

// --- Going on with the learner's own key ------------------------------------------------------

export interface KeyOption {
  id: number
  name: string
  group: string
}

const OWN_KEY = 'learn_own_key'

/**
 * The learner's own key, chosen to go on once the day's free calls are used up (billed as usual on
 * that key). Free calls are always used first.
 */
export const ownKey = reactive<{ id: number; name: string }>({ id: 0, name: '' })

if (inBrowser) {
  try {
    Object.assign(ownKey, JSON.parse(localStorage.getItem(OWN_KEY) || '{}'))
  } catch {
    // ignore
  }
}

export function chooseOwnKey(key: KeyOption | null) {
  ownKey.id = key?.id || 0
  ownKey.name = key?.name || ''
  localStorage.setItem(OWN_KEY, JSON.stringify(ownKey))
}

let keysPromise: Promise<KeyOption[]> | null = null
export function listOwnKeys(): Promise<KeyOption[]> {
  keysPromise ||= api<KeyOption[]>('/learn/keys').catch(() => {
    keysPromise = null
    return []
  })
  return keysPromise
}

/** Reasons meaning the free calls are used up (the learner may go on with their own key). */
export const QUOTA_REASONS = ['LEARN_RUN_QUOTA', 'LEARN_TUTOR_QUOTA', 'LEARN_INTERVIEW_QUOTA']

// --- Quizzes, checkpoints, certificates -------------------------------------------------------

export interface QuizView {
  lesson: string
  questions: { q: string; options: string[]; multi: boolean }[]
}

export interface QuizGrade {
  correct: number
  total: number
  passed: boolean
  items: { correct: boolean; answer: number[]; explain: string }[]
  best: QuizResult
}

export function getQuiz(lesson: string): Promise<QuizView> {
  return api<QuizView>(`/learn/quiz/${lesson}`)
}

export async function submitQuiz(lesson: string, answers: number[][]): Promise<QuizGrade> {
  const grade = await api<QuizGrade>(`/learn/quiz/${lesson}`, { method: 'POST', body: JSON.stringify({ answers }) })
  progress.quizzes = { ...progress.quizzes, [lesson]: grade.best }
  if (grade.passed) progress.completed = { ...progress.completed, [lesson]: progress.completed[lesson] || new Date().toISOString() }
  return grade
}

export function checkpointInfo(id: string): Promise<{ id: string; title: string; hint: string }> {
  return api(`/learn/checkpoints/${id}`)
}

export async function verifyCheckpoint(id: string): Promise<{ id: string; title: string; passed: boolean; hint?: string }> {
  const res = await api<{ id: string; title: string; passed: boolean; hint?: string }>(`/learn/checkpoints/${id}/verify`, { method: 'POST' })
  if (res.passed) progress.checkpoints = { ...progress.checkpoints, [id]: new Date().toISOString() }
  return res
}

export interface CertItem {
  kind: string
  id?: string
  title: string
  done: boolean
  detail?: string
}

export interface CertStatus {
  track: string
  title: string
  eligible: boolean
  quiz_score: number
  items: CertItem[]
  projects: { url: string; title: string }[]
  project_link: boolean
  certificate?: Certificate
}

export function certStatus(track: string): Promise<CertStatus> {
  return api<CertStatus>(`/learn/certificates/${track}`)
}

export function claimCert(track: string, displayName: string, projectUrl: string): Promise<Certificate> {
  return api<Certificate>(`/learn/certificates/${track}`, { method: 'POST', body: JSON.stringify({ display_name: displayName, project_url: projectUrl }) })
}

export function publicCert(code: string): Promise<Certificate> {
  return api<Certificate>(`/learn/cert/${encodeURIComponent(code)}`)
}

// --- AI tutor ---------------------------------------------------------------------------------

/** Asks the tutor; onDelta gets each piece of the reply as it streams. */
export async function askTutor(
  lesson: string,
  messages: RunMessage[],
  keyId: number,
  onDelta: (text: string) => void,
): Promise<{ tutor_left?: number; own_key?: boolean }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'text/event-stream' }
  const t = token()
  if (t) headers.Authorization = `Bearer ${t}`
  let res: Response
  try {
    res = await fetch('/api/v1/learn/tutor', { method: 'POST', headers, body: JSON.stringify({ lesson, messages, key_id: keyId || undefined }) })
  } catch {
    throw new ApiError('网络连接失败，请稍后再试', 0)
  }
  const type = res.headers.get('Content-Type') || ''
  if (!type.includes('text/event-stream')) {
    let body: { code?: number; message?: string; reason?: string } = {}
    try {
      body = await res.json()
    } catch {
      // not JSON
    }
    throw new ApiError(body.message || `请求失败（HTTP ${res.status}）`, res.status, body.reason || '')
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  let final: { tutor_left?: number; own_key?: boolean } = {}
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    let i: number
    while ((i = buf.indexOf('\n\n')) >= 0) {
      const line = buf.slice(0, i).trim()
      buf = buf.slice(i + 2)
      if (!line.startsWith('data:')) continue
      const ev = JSON.parse(line.slice(5).trim())
      if (ev.delta) onDelta(ev.delta)
      else if (ev.error) throw new ApiError(ev.error, 502)
      else if (ev.done) final = ev
    }
  }
  if (typeof final.tutor_left === 'number' && !final.own_key) progress.tutorLeft = final.tutor_left
  return final
}

// --- Mock interviews --------------------------------------------------------------------------

export interface InterviewAnswer {
  answer: string
  score: number
  comment: string
  better: string
}

export interface Interview {
  id: number
  topic: string
  topic_title: string
  questions: { id: string; q: string }[]
  answers: InterviewAnswer[] | null
  status: 'active' | 'finished'
  score: number
  own_key?: boolean
  created_at: string
  finished_at?: string
  interviews_left?: number
}

export interface InterviewTopic {
  id: string
  title: string
  count: number
  best: number
}

export function listInterviews(): Promise<{ topics: InterviewTopic[]; recent: Interview[] }> {
  return api('/learn/interviews')
}

export async function startInterview(topic: string, keyId: number): Promise<Interview> {
  const iv = await api<Interview>('/learn/interviews', { method: 'POST', body: JSON.stringify({ topic, key_id: keyId || undefined }) })
  if (typeof iv.interviews_left === 'number') progress.interviewsLeft = iv.interviews_left
  return iv
}

export function getInterview(id: number): Promise<Interview> {
  return api<Interview>(`/learn/interviews/${id}`)
}

export function answerInterview(id: number, answer: string): Promise<Interview> {
  return api<Interview>(`/learn/interviews/${id}/answer`, { method: 'POST', body: JSON.stringify({ answer }) })
}
