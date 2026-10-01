/**
 * Canvas image tools (AI background removal, AI upscaling): the signed-in user's runs and charges.
 * Subscribers get a few free runs a day; other runs are charged to the balance after they succeed.
 */
import { apiClient } from './client'

export type ImageTool = 'remove_bg' | 'upscale'
export const IMAGE_TOOLS: ImageTool[] = ['remove_bg', 'upscale']

export interface ImageToolUse {
  id: number
  user_id: number
  user_email?: string
  api_key_id?: number
  api_key_name: string
  tool: ImageTool
  free: boolean
  cost: number
  input_bytes: number
  output_bytes: number
  duration_ms: number
  created_at: string
}

export interface ImageToolStat {
  tool: ImageTool
  runs: number
  free_runs: number
  cost: number
  users: number
}

export interface ImageToolsQuota {
  enabled: boolean
  subscribed: boolean
  free_daily: number
  free_used: number
  free_left: number
  balance: number
  prices: Record<ImageTool, number>
}

export interface ImageToolUseList {
  items: ImageToolUse[]
  total: number
  page: number
  page_size: number
}

export interface MyImageToolUses extends ImageToolUseList {
  summary: { quota: ImageToolsQuota; month: ImageToolStat[] }
}

/** Totals over tools (missing tools count as zero). */
export function sumImageToolStats(stats: ImageToolStat[] | undefined): Omit<ImageToolStat, 'tool' | 'users'> {
  return (stats || []).reduce((sum, s) => ({ runs: sum.runs + s.runs, free_runs: sum.free_runs + s.free_runs, cost: sum.cost + s.cost }), { runs: 0, free_runs: 0, cost: 0 })
}

export function imageToolStat(stats: ImageToolStat[] | undefined, tool: ImageTool): ImageToolStat {
  return stats?.find((s) => s.tool === tool) || { tool, runs: 0, free_runs: 0, cost: 0, users: 0 }
}

export async function myUses(params: { page?: number; page_size?: number; tool?: string } = {}): Promise<MyImageToolUses> {
  const { data } = await apiClient.get('/user/image-tools/uses', { params })
  return data
}

export default { myUses }
