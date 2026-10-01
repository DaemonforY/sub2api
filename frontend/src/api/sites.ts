/**
 * Static-site hosting: the signed-in user's sites (publish an .html file or a .zip, update, renew,
 * delete) and the public abuse report form.
 */
import { apiClient } from './client'

export type SiteStatus = 'active' | 'disabled' | 'unpaid' | 'lapsed' | 'pending'

export interface Site {
  id: number
  user_id: number
  user_email?: string
  name: string
  title: string
  status: SiteStatus
  status_reason: string
  version: number
  /** Uploaded version waiting for review (0: none). */
  pending_version: number
  has_password: boolean
  views_7d: number
  views_total: number
  size_bytes: number
  file_count: number
  paid: boolean
  paid_until?: string
  lapsed_at?: string
  created_at: string
  updated_at: string
  url: string
  /** Opens the pending version (owner only). */
  preview_url?: string
  /** The name before the last rename; it redirects here for 30 days. */
  previous_name?: string
  renamed_at?: string
  /** When the next rename is allowed (absent: now). */
  rename_after?: string
}

export interface SiteNameCheck {
  name: string
  available: boolean
  /** Why not (user-facing). */
  reason?: string
}

/** Same rule as the server: 3–30 lowercase letters, digits and inner hyphens, starting with a letter. */
export const SITE_NAME_PATTERN = /^[a-z][a-z0-9-]{1,28}[a-z0-9]$/

export function normalizeSiteName(name: string): string {
  return name.trim().toLowerCase()
}

export interface SiteVersion {
  version: number
  size_bytes: number
  file_count: number
  review_status: 'approved' | 'pending' | 'rejected' | 'superseded'
  review_reason: string
  reviewed_by: string
  created_at: string
  current: boolean
  available: boolean
}

export interface SiteDailyStat {
  day: string
  views: number
  visitors: number
  bytes: number
}

export interface SiteStats {
  days: SiteDailyStat[]
  views: number
  visitors: number
  bytes: number
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

/** Uploads take a while on slow connections (up to the size limit). */
const UPLOAD_CONFIG = { timeout: 5 * 60 * 1000 }

function uploadForm(title: string, file?: File | null, name?: string): FormData {
  const form = new FormData()
  form.append('title', title)
  if (name) form.append('name', name)
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
export async function createSite(title: string, file: File, name = ''): Promise<Site> {
  const { data } = await apiClient.post('/sites', uploadForm(title, file, name), UPLOAD_CONFIG)
  return data
}
export async function checkSiteName(name: string, siteId = 0): Promise<SiteNameCheck> {
  const { data } = await apiClient.get('/sites/name-check', { params: { name, site_id: siteId || undefined } })
  return data
}
export async function renameSite(id: number, name: string): Promise<Site> {
  const { data } = await apiClient.put(`/sites/${id}/name`, { name })
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
export async function siteVersions(id: number): Promise<SiteVersion[]> {
  const { data } = await apiClient.get(`/sites/${id}/versions`)
  return data
}
export async function rollbackSite(id: number, version: number): Promise<Site> {
  const { data } = await apiClient.post(`/sites/${id}/rollback`, { version })
  return data
}
export async function setSitePassword(id: number, password: string): Promise<Site> {
  const { data } = await apiClient.put(`/sites/${id}/password`, { password })
  return data
}
export async function siteStats(id: number, days = 30): Promise<SiteStats> {
  const { data } = await apiClient.get(`/sites/${id}/stats`, { params: { days } })
  return data
}

/** Bar heights (0–100) for a stats chart, scaled to the busiest day. */
export function statBars(days: SiteDailyStat[]): number[] {
  const max = Math.max(1, ...days.map((d) => d.views))
  return days.map((d) => Math.round((d.views / max) * 100))
}

export async function reportSite(input: { site: string; reason: SiteReportReason; detail: string; contact: string }): Promise<void> {
  await apiClient.post('/site-reports', input)
}

export default { mySites, createSite, checkSiteName, renameSite, updateSite, deleteSite, renewSite, reportSite, siteVersions, rollbackSite, setSitePassword, siteStats }
