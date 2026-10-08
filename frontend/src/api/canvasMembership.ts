/**
 * 创作会员: while it lasts, images saved from the canvas image workbench come without the
 * 「AI 生成 · <site>」 watermark. Plans are bought on /purchase?tab=membership.
 */
import { apiClient } from './client'

export interface CanvasMembershipPlan {
  id: number
  name: string
  days: number
  price: number
  original_price?: number
}

export interface CanvasMembershipStatus {
  on_sale: boolean
  plans: CanvasMembershipPlan[]
  /** End of the membership, when the user is a member. */
  until?: string
}

export interface CanvasMembershipConfig {
  enabled: boolean
  plans: CanvasMembershipPlan[]
}

export interface CanvasMember {
  user_id: number
  email: string
  username: string
  member_until: string
  terms_accepted_at?: string
  unmarked_saves: number
}

export async function getCanvasMembership(): Promise<CanvasMembershipStatus> {
  const { data } = await apiClient.get('/canvas-membership')
  return data
}

export async function adminGetCanvasMembership(all = false): Promise<{ config: CanvasMembershipConfig; members: CanvasMember[] }> {
  const { data } = await apiClient.get('/admin/canvas-membership', { params: all ? { all: 1 } : undefined })
  return data
}

export async function adminSaveCanvasMembership(config: CanvasMembershipConfig): Promise<{ config: CanvasMembershipConfig; members: CanvasMember[] }> {
  const { data } = await apiClient.put('/admin/canvas-membership', config)
  return data
}

export async function adminGrantCanvasMembership(input: { user_id?: number; email?: string; days: number }): Promise<{ user_id: number; until?: string }> {
  const { data } = await apiClient.post('/admin/canvas-membership/grant', input)
  return data
}
