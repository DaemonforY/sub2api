/** Admin API for AI 学习 (/learn): example-run settings and learning stats. */
import { apiClient } from '../client'

export interface LearnSettings {
  run_enabled: boolean
  model: string
  free_runs_per_day: number
  daily_cap: number
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

export default { getSettings, saveSettings, stats }
