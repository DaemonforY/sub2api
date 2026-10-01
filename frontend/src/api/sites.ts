/**
 * Static-site hosting: the signed-in user's sites (publish an .html file or a .zip, update, renew,
 * delete) and the public abuse report form.
 */
import { apiClient } from './client'

export type SiteStatus = 'active' | 'disabled' | 'unpaid' | 'lapsed'

export interface Site {
  id: number
  user_id: number
  user_email?: string
  name: string
  title: string
  status: SiteStatus
  status_reason: string
  version: number
  size_bytes: number
  file_count: number
  paid: boolean
  paid_until?: string
  lapsed_at?: string
  created_at: string
  updated_at: string
  url: string
}

export interface SiteCharge {
  id: number
  site_id?: number
  site_name: string
  amount: number
  period_end: string
  created_at: string
}

export interface MySitesQuota {
  available: boolean
  subscribed: boolean
  domain: string
  max_sites: number
  used: number
  free_sites: number
  free_used: number
  extra_price: number
  max_mb: number
  max_files: number
  grace_days: number
  balance: number
}

export interface MySites {
  sites: Site[]
  quota: MySitesQuota
  charges: SiteCharge[]
}

export const SITE_REPORT_REASONS = ['phishing', 'fraud', 'gambling', 'porn', 'malware', 'copyright', 'other'] as const
export type SiteReportReason = (typeof SITE_REPORT_REASONS)[number]

/**
 * Uploads take a while on slow connections (up to the size limit). The explicit multipart type
 * stops axios from turning the FormData into JSON (the client defaults to application/json);
 * the browser then fills in the boundary.
 */
const UPLOAD_CONFIG = { timeout: 5 * 60 * 1000, headers: { 'Content-Type': 'multipart/form-data' } }

function uploadForm(title: string, file?: File | null): FormData {
  const form = new FormData()
  form.append('title', title)
  if (file) form.append('file', file)
  return form
}

/** What a new site would cost: free while the allowance lasts, otherwise the 30-day price. */
export function nextSiteIsPaid(quota: MySitesQuota): boolean {
  return quota.free_used >= quota.free_sites && quota.extra_price > 0
}

/** Accepts what the server accepts (checked again there). */
export function isSiteUploadFile(file: File): boolean {
  return /\.(html?|zip)$/i.test(file.name)
}

export async function mySites(): Promise<MySites> {
  const { data } = await apiClient.get('/sites')
  return data
}
export async function createSite(title: string, file: File): Promise<Site> {
  const { data } = await apiClient.post('/sites', uploadForm(title, file), UPLOAD_CONFIG)
  return data
}
export async function updateSite(id: number, title: string, file?: File | null): Promise<Site> {
  const { data } = await apiClient.put(`/sites/${id}`, uploadForm(title, file), UPLOAD_CONFIG)
  return data
}
export async function deleteSite(id: number): Promise<void> {
  await apiClient.delete(`/sites/${id}`)
}
export async function renewSite(id: number): Promise<Site> {
  const { data } = await apiClient.post(`/sites/${id}/renew`)
  return data
}
export async function reportSite(input: { site: string; reason: SiteReportReason; detail: string; contact: string }): Promise<void> {
  await apiClient.post('/site-reports', input)
}

export default { mySites, createSite, updateSite, deleteSite, renewSite, reportSite }
