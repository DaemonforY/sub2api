/**
 * AI 助教: teachers' assistants (/tutors, signed in) and the students' page (/t/:code, no account:
 * a class password and a name give a token kept in this browser).
 */
import { apiClient } from './client'
import { buildApiUrl } from './url'

export interface TutorTemplate {
  id: string
  name: string
  description: string
  greeting: string
  suggestions: string[]
  materials: string
}

export interface TutorMaterial {
  id: number
  name: string
  chars: number
  created_at: string
}

export interface Tutor {
  id: number
  key_id: number
  name: string
  template: string
  subject: string
  grade: string
  style: string
  answer_mode: 'guide' | 'answer'
  rules: string
  greeting: string
  share_code: string
  pass_code: string
  per_student_day: number
  daily_cap: number
  enabled: boolean
  created_at: string
  updated_at: string
  material_chars: number
  materials?: TutorMaterial[]
  key_name?: string
  key_problem?: string
  /** One question on the key with the current materials: low when the prompt is cached, high when not (USD). */
  cost?: TutorCost
}

export interface TutorCost {
  low: number
  high: number
  input_tokens: number
  output_tokens: number
  model: string
}

export type TutorInput = Pick<
  Tutor,
  'key_id' | 'name' | 'template' | 'subject' | 'grade' | 'style' | 'answer_mode' | 'rules' | 'greeting' | 'pass_code' | 'per_student_day' | 'daily_cap' | 'enabled'
>

export interface TutorStudent {
  id: number
  name: string
  questions: number
  created_at: string
  last_seen_at: string
}

export interface TutorMessage {
  id: number
  student_id: number
  student_name: string
  question: string
  answer: string
  created_at: string
}

export interface TutorStats {
  students: TutorStudent[]
  today: number
  week: number
  recent: TutorMessage[]
}

export interface KeyOption {
  id: number
  name: string
  group: string
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export async function listTemplates(): Promise<TutorTemplate[]> {
  const { data } = await apiClient.get('/tutors/templates')
  return data
}
export async function listTutors(): Promise<Tutor[]> {
  const { data } = await apiClient.get('/tutors')
  return data
}
export async function getTutor(id: number): Promise<Tutor> {
  const { data } = await apiClient.get(`/tutors/${id}`)
  return data
}
export async function createTutor(input: TutorInput): Promise<Tutor> {
  const { data } = await apiClient.post('/tutors', input)
  return data
}
export async function updateTutor(id: number, input: TutorInput): Promise<Tutor> {
  const { data } = await apiClient.put(`/tutors/${id}`, input)
  return data
}
export async function deleteTutor(id: number): Promise<void> {
  await apiClient.delete(`/tutors/${id}`)
}
export async function uploadMaterial(id: number, file: File): Promise<TutorMaterial> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await apiClient.post(`/tutors/${id}/materials`, form, { timeout: 120000 })
  return data
}
export async function pasteMaterial(id: number, name: string, text: string): Promise<TutorMaterial> {
  const { data } = await apiClient.post(`/tutors/${id}/materials`, { name, text })
  return data
}
export async function deleteMaterial(id: number, materialId: number): Promise<void> {
  await apiClient.delete(`/tutors/${id}/materials/${materialId}`)
}
export async function getStats(id: number): Promise<TutorStats> {
  const { data } = await apiClient.get(`/tutors/${id}/stats`)
  return data
}
/** The signed-in user's GPT-group keys. */
export async function listKeys(): Promise<KeyOption[]> {
  const { data } = await apiClient.get('/learn/keys')
  return data
}

/** POSTs and reads the SSE answer: {"delta"} events, then {"done":true, ...extra} or {"error"}. */
async function stream(
  path: string,
  body: unknown,
  headers: Record<string, string>,
  onDelta: (text: string) => void,
  signal?: AbortSignal
): Promise<Record<string, unknown>> {
  const res = await fetch(buildApiUrl(path), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream', ...headers },
    body: JSON.stringify(body ?? {}),
    signal
  })
  const type = res.headers.get('Content-Type') || ''
  if (!type.includes('text/event-stream')) {
    let parsed: { message?: string; reason?: string; data?: Record<string, unknown> } = {}
    try {
      parsed = await res.json()
    } catch {
      // not JSON
    }
    if (!res.ok) {
      const err = new Error(parsed.message || `请求失败（HTTP ${res.status}）`) as Error & { reason?: string; status?: number }
      err.reason = parsed.reason
      err.status = res.status
      throw err
    }
    return parsed.data || {}
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let done: Record<string, unknown> | null = null
  for (;;) {
    const { value, done: finished } = await reader.read()
    if (finished) break
    buffer += decoder.decode(value, { stream: true })
    let i: number
    while ((i = buffer.indexOf('\n\n')) >= 0) {
      const event = buffer.slice(0, i)
      buffer = buffer.slice(i + 2)
      const line = event.split('\n').find((l) => l.startsWith('data: '))
      if (!line) continue
      const msg = JSON.parse(line.slice(6)) as { delta?: string; done?: boolean; error?: string }
      if (msg.error) throw new Error(msg.error)
      if (msg.delta) onDelta(msg.delta)
      if (msg.done) done = msg as Record<string, unknown>
    }
  }
  if (!done) throw new Error('回答中断了，请重试（Stream interrupted）')
  return done
}

function authHeader(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

/** The teacher tries the assistant as a student (not counted). */
export function streamPreview(id: number, messages: ChatMessage[], onDelta: (t: string) => void, signal?: AbortSignal) {
  return stream(`/tutors/${id}/preview`, { messages }, authHeader(), onDelta, signal)
}

/** AI summary of the last 7 days' questions. */
export function streamInsights(id: number, onDelta: (t: string) => void, signal?: AbortSignal) {
  return stream(`/tutors/${id}/insights`, {}, authHeader(), onDelta, signal)
}

// --- students ---

export interface TutorPublic {
  name: string
  template: string
  subject: string
  greeting: string
  suggestions: string[]
  needs_pass: boolean
  enabled: boolean
}

export interface TutorJoin {
  token: string
  name: string
  left: number
}

export async function getPublic(code: string): Promise<TutorPublic> {
  const { data } = await apiClient.get(`/t/${encodeURIComponent(code)}`)
  return data
}
export async function joinTutor(code: string, name: string, pass: string): Promise<TutorJoin> {
  const { data } = await apiClient.post(`/t/${encodeURIComponent(code)}/join`, { name, pass })
  return data
}
/** Resolves to {left}; errors carry the server's message and reason (TUTOR_SESSION: join again). */
export async function streamStudentChat(code: string, token: string, messages: ChatMessage[], onDelta: (t: string) => void, signal?: AbortSignal) {
  const done = await stream(`/t/${encodeURIComponent(code)}/chat`, { messages }, { 'X-Tutor-Token': token }, onDelta, signal)
  return { left: Number(done.left ?? 0) }
}
