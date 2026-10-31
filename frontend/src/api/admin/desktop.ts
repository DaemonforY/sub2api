/** Admin API for HiveGPT 桌面助手: the skill library the app downloads, and the overview. */
import { apiClient } from '../client'

const BASE = '/admin/desktop'

export type DesktopSkillStatus = 'draft' | 'published' | 'hidden'

export interface DesktopSkill {
  id: number
  slug: string
  title: string
  icon: string
  summary: string
  prepare: string
  prompt: string
  sample_prompt: string
  instructions: string
  needs_office: boolean
  status: DesktopSkillStatus
  sort_order: number
  version: number
  created_at: string
  updated_at: string
}

export type DesktopSkillInput = Omit<DesktopSkill, 'id' | 'version' | 'created_at' | 'updated_at'>

export interface DesktopFunnelStep {
  event: string
  events: number
  visitors: number
}

export interface DesktopTemplateStat {
  template: string
  starts: number
  done: number
  failed: number
  avg_done_sec: number
  top_failure: string
}

export interface DesktopDevice {
  key_id: number
  name: string
  user_id: number
  user_email: string
  status: string
  last_used_at: string | null
  created_at: string
}

export interface DesktopOverview {
  days: number
  devices: number
  active_devices: number
  funnel: DesktopFunnelStep[]
  returned: number
  templates: DesktopTemplateStat[]
  failures: { step: string; reason: string; count: number }[]
  recent_devices: DesktopDevice[]
}

export async function getOverview(days = 30): Promise<DesktopOverview> {
  const { data } = await apiClient.get(`${BASE}/overview`, { params: { days } })
  return data
}

export async function listSkills(): Promise<DesktopSkill[]> {
  const { data } = await apiClient.get(`${BASE}/skills`)
  return data.items
}

export async function getSkill(id: number): Promise<{ skill: DesktopSkill; skill_md: string }> {
  const { data } = await apiClient.get(`${BASE}/skills/${id}`)
  return data
}

export async function createSkill(input: DesktopSkillInput): Promise<DesktopSkill> {
  const { data } = await apiClient.post(`${BASE}/skills`, input)
  return data
}

export async function updateSkill(id: number, input: DesktopSkillInput): Promise<DesktopSkill> {
  const { data } = await apiClient.put(`${BASE}/skills/${id}`, input)
  return data
}

export async function setSkillStatus(id: number, status: DesktopSkillStatus): Promise<void> {
  await apiClient.post(`${BASE}/skills/${id}/status`, { status })
}

export async function deleteSkill(id: number): Promise<void> {
  await apiClient.delete(`${BASE}/skills/${id}`)
}
