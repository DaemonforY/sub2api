/** Admin API for hosted sites: settings, take-downs and abuse reports. */
import { apiClient } from '../client'
import type { Site } from '../sites'

export interface SiteHostingSettings {
  domain: string
  enabled: boolean
  max_per_user: number
  max_mb: number
  max_files: number
  free_per_user: number
  extra_price: number
  grace_days: number
  retention_days: number
  review_all: boolean
  review_base_url: string
  review_model: string
  review_api_key_configured: boolean
  /** Write-only: a new key for the review model. */
  review_api_key?: string
  clear_review_api_key?: boolean
}

export interface SiteReviewItem {
  site_id: number
  site_name: string
  title: string
  owner_email: string
  site_status: string
  version: number
  size_bytes: number
  file_count: number
  reason: string
  flags: string[]
  excerpt: string
  created_at: string
  preview_url: string
}

export interface SiteReport {
  id: number
  site_id?: number
  site_name: string
  site_status: string
  owner_email: string
  reason: string
  detail: string
  contact: string
  reporter_ip: string
  status: 'open' | 'resolved' | 'dismissed'
  handled_at?: string
  created_at: string
}

export interface Paged<T> {
  items: T[]
  total: number
}

export async function getSettings(): Promise<SiteHostingSettings> {
  const { data } = await apiClient.get('/admin/sites/settings')
  return data
}
export async function saveSettings(input: SiteHostingSettings): Promise<SiteHostingSettings> {
  const { data } = await apiClient.put('/admin/sites/settings', input)
  return data
}
export async function list(params: { q?: string; status?: string; page?: number; page_size?: number }): Promise<Paged<Site>> {
  const { data } = await apiClient.get('/admin/sites', { params })
  return data
}
export async function setStatus(id: number, status: 'active' | 'disabled', reason = ''): Promise<void> {
  await apiClient.put(`/admin/sites/${id}/status`, { status, reason })
}
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/sites/${id}`)
}
export async function reports(params: { status?: string; page?: number; page_size?: number }): Promise<Paged<SiteReport>> {
  const { data } = await apiClient.get('/admin/sites/reports', { params })
  return data
}
export async function setReportStatus(id: number, status: SiteReport['status']): Promise<void> {
  await apiClient.put(`/admin/sites/reports/${id}`, { status })
}

export async function reviews(params: { page?: number; page_size?: number }): Promise<Paged<SiteReviewItem>> {
  const { data } = await apiClient.get('/admin/sites/reviews', { params })
  return data
}
export async function review(siteId: number, version: number, action: 'approve' | 'reject', reason = ''): Promise<void> {
  await apiClient.post(`/admin/sites/${siteId}/review`, { version, action, reason })
}

export default { getSettings, saveSettings, list, setStatus, remove, reports, setReportStatus, reviews, review }
