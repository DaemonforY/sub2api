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
  /** Set when the report is about a comment of the work. */
  comment_id?: number
  comment_body?: string
  comment_status?: AdminCommentStatus
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

export type AdminCommentStatus = 'approved' | 'pending' | 'hidden' | 'deleted'

export interface AdminComment {
  id: number
  work_id: number
  work_title: string
  owner_id: number
  parent_id?: number
  author?: { handle: string; display_name: string; avatar_url: string }
  body: string
  status: AdminCommentStatus
  review_flags?: string[]
  report_count?: number
  created_at: string
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
/** pending | reported | hidden | '' (all) */
export async function comments(status: string, page = 1): Promise<AdminComment[]> {
  const { data } = await apiClient.get('/admin/community/comments', { params: { status: status || undefined, page } })
  return data
}
export async function moderateComment(id: number, action: 'approve' | 'hide'): Promise<void> {
  await apiClient.post(`/admin/community/comments/${id}/moderate`, { action })
}
/**
 * review_all: hand-review every new work; comments_enabled / comments_review_all: comments on works;
 * cloud quotas: canvas cloud sync space per user (MB).
 */
export interface CommunitySettings {
  review_all: boolean
  comments_enabled: boolean
  comments_review_all: boolean
  cloud_quota_mb: number
  cloud_subscriber_quota_mb: number
}
export async function getSettings(): Promise<CommunitySettings> {
  const { data } = await apiClient.get('/admin/community/settings')
  return data
}
export async function saveSettings(settings: CommunitySettings): Promise<CommunitySettings> {
  const { data } = await apiClient.put('/admin/community/settings', settings)
  return data
}

export default { works, moderate, ban, restricted, reports, setReport, comments, moderateComment, getSettings, saveSettings }
