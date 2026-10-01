/** Admin API for the canvas community: review queue, reports, editor's picks and author restrictions. */
import { apiClient } from '../client'

export type CommunityWorkStatus = 'approved' | 'pending' | 'rejected' | 'hidden'

export interface AdminCommunityWork {
  id: number
  owner_id: number
  author: { handle: string; display_name: string; avatar_url: string }
  title: string
  description: string
  prompt: string
  model: string
  tags: string[]
  visibility: 'public' | 'unlisted' | 'private'
  status: CommunityWorkStatus
  review_reason?: string
  review_flags?: string[]
  featured: boolean
  cover_thumb_url: string
  image_count: number
  like_count: number
  report_count?: number
  media?: { url: string; thumb_url: string }[]
  created_at: string
}

export interface AdminWorkReport {
  id: number
  work_id: number
  work_title: string
  reason: string
  detail: string
  status: 'open' | 'resolved' | 'dismissed'
  created_at: string
}

export interface RestrictedAuthor {
  user_id: number
  email: string
  handle: string
  display_name: string
  works_count: number
  updated_at: string
}

export type ModerateAction = 'approve' | 'reject' | 'hide' | 'feature' | 'unfeature'

export async function works(status: string, page = 1): Promise<AdminCommunityWork[]> {
  const { data } = await apiClient.get('/admin/community/works', { params: { status: status || undefined, page } })
  return data
}
export async function moderate(id: number, action: ModerateAction, reason = ''): Promise<void> {
  await apiClient.post(`/admin/community/works/${id}/moderate`, { action, reason })
}
export async function ban(userId: number, banned: boolean): Promise<void> {
  await apiClient.post(`/admin/community/users/${userId}/ban`, { banned })
}
export async function restricted(): Promise<RestrictedAuthor[]> {
  const { data } = await apiClient.get('/admin/community/restricted')
  return data
}
export async function reports(status: string, page = 1): Promise<AdminWorkReport[]> {
  const { data } = await apiClient.get('/admin/community/reports', { params: { status: status || undefined, page } })
  return data
}
export async function setReport(id: number, status: AdminWorkReport['status']): Promise<void> {
  await apiClient.put(`/admin/community/reports/${id}`, { status })
}
export async function getSettings(): Promise<{ review_all: boolean }> {
  const { data } = await apiClient.get('/admin/community/settings')
  return data
}
export async function saveSettings(reviewAll: boolean): Promise<{ review_all: boolean }> {
  const { data } = await apiClient.put('/admin/community/settings', { review_all: reviewAll })
  return data
}

export default { works, moderate, ban, restricted, reports, setReport, getSettings, saveSettings }
