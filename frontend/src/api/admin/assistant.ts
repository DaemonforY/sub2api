/** Admin API for the homepage support assistant (智能客服). */
import { apiClient } from '../client'

export interface AssistantSettings {
  enabled: boolean
  model: string
  /** One of the signed-in admin's own GPT keys; only its ID is stored. */
  key_id: number
  key_owner?: number
  key_name?: string
  /** Why the chosen key can't be used (deleted, disabled, not a GPT group). */
  key_problem?: string
  user_per_day: number
  guest_per_day: number
  daily_cap: number
  /** 账户诊断: signed-in users' questions may look up their own account, keys, errors and usage. */
  tools: boolean
  /** Questions answered today, site-wide. */
  today: number
  /** Learning-site pages in the knowledge base. */
  pages: number
}

export interface AssistantKeyOption {
  id: number
  name: string
  group: string
}

/** One tool call in a run. */
export interface AgentStep {
  tool: string
  args?: string
  result?: string
  error?: string
  ms: number
}

/** One question the assistant answered, with what it looked up. */
export interface AgentRun {
  id: number
  user_id: number | null
  user_email?: string
  question: string
  answer: string
  steps: AgentStep[]
  model: string
  model_calls: number
  prompt_tokens: number
  completion_tokens: number
  status: 'ok' | 'error'
  error?: string
  duration_ms: number
  created_at: string
}

export async function listRuns(page = 1): Promise<{ items: AgentRun[]; total: number }> {
  const { data } = await apiClient.get('/admin/assistant/runs', { params: { page } })
  return data
}

export async function getSettings(): Promise<AssistantSettings> {
  const { data } = await apiClient.get('/admin/assistant/settings')
  return data
}
export async function saveSettings(settings: AssistantSettings): Promise<AssistantSettings> {
  const { data } = await apiClient.put('/admin/assistant/settings', settings)
  return data
}
export async function listKeys(): Promise<AssistantKeyOption[]> {
  const { data } = await apiClient.get('/admin/assistant/keys')
  return data
}
