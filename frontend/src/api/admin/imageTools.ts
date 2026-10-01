/** Admin API for the canvas image tools: prices, daily free runs, on/off switch, usage records. */
import { apiClient } from '../client'
import type { ImageToolStat, ImageToolUseList } from '../imageTools'

export interface ImageToolsSettings {
  enabled: boolean
  price_remove_bg: number
  price_upscale: number
  free_daily: number
  /** False when IMAGE_TOOLS_BASE_URL is not set on the server (the tools cannot run). */
  service_configured: boolean
}

export type ImageToolsSettingsInput = Omit<ImageToolsSettings, 'service_configured'>

export interface ImageToolsStats {
  today: ImageToolStat[]
  week: ImageToolStat[]
  month: ImageToolStat[]
}

export interface ImageToolUsesQuery {
  q?: string
  tool?: string
  user_id?: number
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export async function getSettings(): Promise<ImageToolsSettings> {
  const { data } = await apiClient.get('/admin/image-tools/settings')
  return data
}
export async function saveSettings(input: ImageToolsSettingsInput): Promise<ImageToolsSettings> {
  const { data } = await apiClient.put('/admin/image-tools/settings', input)
  return data
}
export async function stats(): Promise<ImageToolsStats> {
  const { data } = await apiClient.get('/admin/image-tools/stats')
  return data
}
export async function uses(query: ImageToolUsesQuery): Promise<ImageToolUseList> {
  const params = Object.fromEntries(Object.entries(query).filter(([, v]) => v !== '' && v !== undefined && v !== null))
  const { data } = await apiClient.get('/admin/image-tools/uses', { params })
  return data
}

export default { getSettings, saveSettings, stats, uses }
