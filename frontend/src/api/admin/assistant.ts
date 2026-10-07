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
