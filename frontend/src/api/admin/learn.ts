/** Admin API for AI 学习 (/learn): example-run settings and learning stats. */
import { apiClient } from '../client'

export interface LearnSettings {
  run_enabled: boolean
  model: string
  free_runs_per_day: number
  daily_cap: number
  tutor_free_per_day: number
  interviews_per_day: number
  api_key_set: boolean
  /** Write-only: send to replace the learning key; empty keeps it. */
  api_key?: string
}

export interface LearnStats {
  learners: number
  learners_today: number
  runs_today: number
  runs_7d: number
  failed_runs_7d: number
  tokens_7d: number
  certificates: number
  quiz_passed: number
  tutor_7d: number
  interviews_7d: number
  own_key_runs_7d: number
  lessons: { lesson_id: string; completed: number; runs: number }[]
}

export async function getSettings(): Promise<LearnSettings> {
  const { data } = await apiClient.get('/admin/learn/settings')
  return data
}
export async function saveSettings(settings: LearnSettings): Promise<LearnSettings> {
  const { data } = await apiClient.put('/admin/learn/settings', settings)
  return data
}
export async function stats(): Promise<LearnStats> {
  const { data } = await apiClient.get('/admin/learn/stats')
  return data
}

export interface LearnCertificate {
  code: string
  user_email?: string
  track: string
  track_title: string
  display_name: string
  project_url: string
  quiz_score: number
  issued_at: string
  revoked_at?: string
  showcase: boolean
  showcase_hidden: boolean
}

export interface LearnInsights {
  funnels: { track: string; title: string; lessons: number; started: number; half: number; finished: number; certificates: number }[]
  days: { date: string; learners: number; completions: number; runs: number; tutor: number; interviews: number; certificates: number }[]
  quizzes: Record<string, { takers: number; passed: number; avg_score: number; attempts: number }>
}

export async function insights(): Promise<LearnInsights> {
  const { data } = await apiClient.get('/admin/learn/insights')
  return data
}
export async function setShowcaseHidden(code: string, hidden: boolean): Promise<void> {
  await apiClient.post(`/admin/learn/certificates/${encodeURIComponent(code)}/showcase`, { hidden })
}

export async function certificates(page = 1, pageSize = 20): Promise<{ items: LearnCertificate[]; total: number }> {
  const { data } = await apiClient.get('/admin/learn/certificates', { params: { page, page_size: pageSize } })
  return data
}
export async function setCertificateRevoked(code: string, revoked: boolean): Promise<void> {
  await apiClient.post(`/admin/learn/certificates/${encodeURIComponent(code)}/revoke`, { revoked })
}

export default { getSettings, saveSettings, stats, certificates, setCertificateRevoked, insights, setShowcaseHidden }
