/**
 * AI 建站: the model writes a single-page site and draws its pictures with the user's own key; the
 * user changes it by chat and publishes it through site hosting.
 */
import { apiClient } from './client'
import type { Site } from './sites'

export type SiteDraftStatus = 'generating' | 'drawing' | 'ready' | 'failed'

export interface SiteDraftBrief {
  description: string
  style: string
  images: number
}

export interface SiteDraftTurn {
  at: string
  instruction: string
  status: 'running' | 'ok' | 'failed'
  error?: string
}

export interface SiteDraftImage {
  n: number
  prompt: string
  size: string
  status: 'pending' | 'ok' | 'failed'
  error?: string
  bytes?: number
}

export interface SiteDraftEvent {
  at: string
  kind: 'info' | 'step' | 'done' | 'error'
  text: string
}

export interface SiteDraft {
  id: number
  key_id: number
  status: SiteDraftStatus
  title: string
  error: string
  can_undo: boolean
  created_at: string
  updated_at: string
  brief: SiteDraftBrief
  html: string
  /** The page being written (while generating). */
  draft_html?: string
  turns: SiteDraftTurn[]
  images: SiteDraftImage[]
  events: SiteDraftEvent[]
  site_id?: number
  site_name?: string
  site_url?: string
  published_at?: string
  model?: string
  prompt_tokens: number
  completion_tokens: number
  images_drawn: number
}

export interface SiteBuilderConfig {
  max_images: number
  default_images: number
  /** USD on the chosen key, when it can be worked out. */
  image_price?: number
  page_price?: number
  revise_price?: number
}

export function siteDraftActive(status: SiteDraftStatus): boolean {
  return status === 'generating' || status === 'drawing'
}

export async function siteBuilderConfig(keyId: number): Promise<SiteBuilderConfig> {
  const { data } = await apiClient.get('/site-drafts/config', { params: { key_id: keyId || undefined } })
  return data
}

export async function listSiteDrafts(): Promise<SiteDraft[]> {
  const { data } = await apiClient.get('/site-drafts')
  return data
}

export async function createSiteDraft(keyId: number, brief: SiteDraftBrief): Promise<SiteDraft> {
  const { data } = await apiClient.post('/site-drafts', { key_id: keyId, ...brief })
  return data
}

export async function getSiteDraft(id: number): Promise<SiteDraft> {
  const { data } = await apiClient.get(`/site-drafts/${id}`)
  return data
}

export async function reviseSiteDraft(id: number, instruction: string): Promise<SiteDraft> {
  const { data } = await apiClient.post(`/site-drafts/${id}/revise`, { instruction })
  return data
}

export async function retrySiteDraft(id: number): Promise<SiteDraft> {
  const { data } = await apiClient.post(`/site-drafts/${id}/retry`)
  return data
}

export async function undoSiteDraft(id: number): Promise<SiteDraft> {
  const { data } = await apiClient.post(`/site-drafts/${id}/undo`)
  return data
}

export async function cancelSiteDraft(id: number): Promise<void> {
  await apiClient.post(`/site-drafts/${id}/cancel`)
}

export async function deleteSiteDraft(id: number): Promise<void> {
  await apiClient.delete(`/site-drafts/${id}`)
}

export async function publishSiteDraft(id: number, input: { site_id?: number; name?: string; title?: string }): Promise<Site> {
  const { data } = await apiClient.post(`/site-drafts/${id}/publish`, input)
  return data
}

export async function siteDraftImage(id: number, n: number): Promise<Blob> {
  const { data } = await apiClient.get(`/site-drafts/${id}/images/${n}`, { responseType: 'blob' })
  return data
}

const placeholder = (label: string) =>
  'data:image/svg+xml;charset=utf-8,' +
  encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" width="1536" height="1024" viewBox="0 0 1536 1024"><rect width="100%" height="100%" fill="#e5e7eb"/><text x="50%" y="50%" font-size="56" fill="#6b7280" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif">${label}</text></svg>`
  )

/**
 * The page with its pictures pointing at the given (data) URLs (or a placeholder while a picture is
 * not drawn yet), for an iframe srcdoc.
 */
export function previewHtml(page: string, urls: Record<number, string>, labels: { pending: string; failed: string }, images: SiteDraftImage[]): string {
  const status: Record<number, SiteDraftImage['status']> = {}
  for (const im of images) status[im.n] = im.status
  return page.replace(/(\bsrc\s*=\s*["'])img\/(\d+)\.jpg(["'])/gi, (_m, a: string, n: string, b: string) => {
    const num = Number(n)
    const url = urls[num] || placeholder(status[num] === 'failed' ? labels.failed : labels.pending)
    return a + url + b
  })
}
