/** Admin API for GEO 监测: do AI assistants mention or cite the site when asked typical questions? */
import { apiClient } from '../client'

export interface GeoQuestion {
  id: number
  question: string
  category: string
  enabled: boolean
  sort: number
  created_at: string
  updated_at: string
}

export interface GeoQuestionInput {
  question: string
  category: string
  enabled: boolean
  sort: number
}

/** The key itself is never returned: only has_key and the last 4 characters (key_masked). */
export interface GeoEngine {
  id: number
  name: string
  base_url: string
  model: string
  extra_body: Record<string, unknown> | null
  enabled: boolean
  has_key: boolean
  key_masked: string
  created_at: string
  updated_at: string
}

/** api_key: empty on update keeps the stored key. extra_body: a JSON object or its text. */
export interface GeoEngineInput {
  name: string
  base_url: string
  api_key: string
  model: string
  extra_body: string
  enabled: boolean
}

export interface GeoCheck {
  id: number
  question_id: number | null
  question: string
  engine_id: number | null
  engine_name: string
  source: 'auto' | 'manual'
  answer: string
  mentioned: boolean
  cited_urls: string[]
  our_urls: string[]
  competitors: string[]
  error: string
  run_id: string
  created_at: string
}

export interface GeoCheckPage {
  items: GeoCheck[]
  total: number
}

export interface GeoCheckQuery {
  question_id?: number
  engine?: string
  source?: '' | 'auto' | 'manual'
  mentioned?: '' | 'true' | 'false'
  limit?: number
  offset?: number
}

export interface GeoManualInput {
  question_id: number
  engine_name: string
  answer: string
  cited_urls: string
}

export interface GeoEngineSummary {
  name: string
  auto: boolean
  enabled: boolean
  answered: number
  mentioned: number
  mention_rate: number
  last_checked_at: string | null
}

export interface GeoWeekRate {
  engine_name: string
  week: string
  week_start: string
  total: number
  mentioned: number
  rate: number
}

export interface GeoSummary {
  engines: GeoEngineSummary[]
  questions: GeoQuestion[]
  latest: GeoCheck[]
  weekly: GeoWeekRate[]
  totals: { questions: number; engines: number; answered: number; mentioned: number; mention_rate: number }
}

export interface GeoRunStatus {
  running: boolean
  run_id: string
  started_at: string | null
  last_run_at: string | null
  done: number
  total: number
}

export interface GeoSettings {
  schedule: 'off' | 'daily' | 'weekly'
  brand_keywords: string[]
  competitor_keywords: string[]
}

export interface GeoEngineTestResult {
  ok: boolean
  latency_ms: number
  answer?: string
  error?: string
}

export async function listQuestions(): Promise<GeoQuestion[]> {
  const { data } = await apiClient.get('/admin/geo/questions')
  return data
}
export async function createQuestion(input: GeoQuestionInput): Promise<GeoQuestion> {
  const { data } = await apiClient.post('/admin/geo/questions', input)
  return data
}
export async function updateQuestion(id: number, input: GeoQuestionInput): Promise<GeoQuestion> {
  const { data } = await apiClient.put(`/admin/geo/questions/${id}`, input)
  return data
}
export async function deleteQuestion(id: number): Promise<void> {
  await apiClient.delete(`/admin/geo/questions/${id}`)
}

export async function listEngines(): Promise<GeoEngine[]> {
  const { data } = await apiClient.get('/admin/geo/engines')
  return data
}
export async function createEngine(input: GeoEngineInput): Promise<GeoEngine> {
  const { data } = await apiClient.post('/admin/geo/engines', input)
  return data
}
export async function updateEngine(id: number, input: GeoEngineInput): Promise<GeoEngine> {
  const { data } = await apiClient.put(`/admin/geo/engines/${id}`, input)
  return data
}
export async function deleteEngine(id: number): Promise<void> {
  await apiClient.delete(`/admin/geo/engines/${id}`)
}
export async function testEngine(id: number): Promise<GeoEngineTestResult> {
  const { data } = await apiClient.post(`/admin/geo/engines/${id}/test`, {}, { timeout: 150000 })
  return data
}

export async function listChecks(query: GeoCheckQuery): Promise<GeoCheckPage> {
  const params: Record<string, string | number> = {}
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== '' && v !== 0) params[k] = v as string | number
  }
  const { data } = await apiClient.get('/admin/geo/checks', { params })
  return data
}
export async function addManualCheck(input: GeoManualInput): Promise<GeoCheck> {
  const { data } = await apiClient.post('/admin/geo/checks', input)
  return data
}
export async function deleteCheck(id: number): Promise<void> {
  await apiClient.delete(`/admin/geo/checks/${id}`)
}

export async function startRun(): Promise<{ run_id: string }> {
  const { data } = await apiClient.post('/admin/geo/run', {})
  return data
}
export async function getStatus(): Promise<GeoRunStatus> {
  const { data } = await apiClient.get('/admin/geo/status')
  return data
}
export async function getSummary(): Promise<GeoSummary> {
  const { data } = await apiClient.get('/admin/geo/summary')
  return data
}
export async function getSettings(): Promise<GeoSettings> {
  const { data } = await apiClient.get('/admin/geo/settings')
  return data
}
export async function saveSettings(input: GeoSettings): Promise<GeoSettings> {
  const { data } = await apiClient.put('/admin/geo/settings', input)
  return data
}

export default {
  listQuestions, createQuestion, updateQuestion, deleteQuestion,
  listEngines, createEngine, updateEngine, deleteEngine, testEngine,
  listChecks, addManualCheck, deleteCheck, startRun, getStatus, getSummary, getSettings, saveSettings
}
