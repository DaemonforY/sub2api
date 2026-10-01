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

export default { getSettings, saveSettings, list, setStatus, remove, reports, setReportStatus }
