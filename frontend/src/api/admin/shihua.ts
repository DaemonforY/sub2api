/** Admin API for 课本诗画: works, scene drafts, drawing jobs, picking, finished posters and export. */
import { apiClient } from '../client'

const BASE = '/admin/shihua'

export type ShihuaTextStatus = 'draft' | 'verified_1' | 'verified_2'
export type ShihuaImageStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'

export interface ShihuaTextbook {
  edition: string
  era?: string
  grade: string
  unit?: number
  lesson?: string
  kind?: string
}

export interface ShihuaTheme {
  subject?: string
  scene?: string
  season?: string
  time?: string
  figures?: string
  key_imagery?: string[]
  palette?: string
  avoid?: string[]
  prompt_theme: string
  model?: string
  generated_at?: string
}

export interface ShihuaImageCounts {
  total: number
  queued: number
  running: number
  succeeded: number
  failed: number
}

export interface ShihuaWork {
  id: number
  slug: string
  title: string
  author: string
  dynasty: string
  genre: string
  stage: string
  grade: string
  textbooks: ShihuaTextbook[]
  full_text: string
  display_text: string
  excerpt_note: string
  famous_lines: string
  sources: string
  variant_note: string
  text_status: ShihuaTextStatus
  verify1_by?: number
  verify1_at?: string
  verify2_by?: number
  verify2_at?: string
  theme: ShihuaTheme
  picked_image_id?: number
  sort: number
  created_at: string
  updated_at: string
  char_count: number
  display_char_count: number
  images: ShihuaImageCounts
  renders: number
}

export type ShihuaWorkInput = Pick<
  ShihuaWork,
  'slug' | 'title' | 'author' | 'dynasty' | 'genre' | 'stage' | 'grade' | 'textbooks' | 'full_text' | 'display_text' | 'excerpt_note' | 'famous_lines' | 'sources' | 'variant_note' | 'sort'
> & { theme?: ShihuaTheme }

export interface ShihuaImage {
  id: number
  work_id: number
  job_id?: number
  status: ShihuaImageStatus
  attempts: number
  not_before?: string
  prompt: string
  size: string
  quality: string
  error?: string
  duration_ms: number
  cost_usd: number
  width: number
  height: number
  upscale_status: '' | 'queued' | 'running' | 'done' | 'failed'
  upscale_error?: string
  title_check: '' | 'ok' | 'wrong'
  rejected: boolean
  created_at: string
  started_at?: string
  finished_at?: string
  url?: string
  thumb_url?: string
  upscaled_url?: string
}

export interface ShihuaJob {
  id: number
  name: string
  candidates: number
  created_at: string
  works: number
  counts: ShihuaImageCounts
  canceled: number
  cost_usd: number
}

export interface ShihuaRender {
  id: number
  image_id: number
  work_id: number
  aspect: string
  width: number
  height: number
  layout: Record<string, unknown>
  created_at: string
  url?: string
}

export interface ShihuaSettings {
  key_id: number
  text_model: string
  size: string
  quality: string
  concurrency: number
  daily_limit: number
  style_prompt: string
  auto_upscale: boolean
  two_people_verify: boolean
}

export interface ShihuaSettingsView extends ShihuaSettings {
  default_style_prompt: string
  upscale_available: boolean
  queue: { queued: number; running: number; started_today: number; upscale_queue: number }
  paused?: string
}

export interface ShihuaImportResult {
  created: number
  updated: number
  unchanged: number
  reset: string[]
  errors: string[]
}

export async function getSettings(): Promise<ShihuaSettingsView> {
  const { data } = await apiClient.get(`${BASE}/settings`)
  return data
}
export async function saveSettings(input: ShihuaSettings): Promise<ShihuaSettingsView> {
  const { data } = await apiClient.put(`${BASE}/settings`, input)
  return data
}

export async function listWorks(params: { stage?: string; grade?: string; status?: string; q?: string; ids?: string } = {}): Promise<ShihuaWork[]> {
  const { data } = await apiClient.get(`${BASE}/works`, { params })
  return data.items
}
export async function createWork(input: ShihuaWorkInput): Promise<ShihuaWork> {
  const { data } = await apiClient.post(`${BASE}/works`, input)
  return data
}
export async function updateWork(id: number, input: ShihuaWorkInput): Promise<ShihuaWork> {
  const { data } = await apiClient.put(`${BASE}/works/${id}`, input)
  return data
}
export async function deleteWork(id: number): Promise<void> {
  await apiClient.delete(`${BASE}/works/${id}`)
}
export async function importWorks(works: ShihuaWorkInput[], dryRun: boolean): Promise<ShihuaImportResult> {
  const { data } = await apiClient.post(`${BASE}/works/import`, { works, dry_run: dryRun }, { timeout: 120000 })
  return data
}
export async function verifyWork(id: number, level: 0 | 1 | 2): Promise<ShihuaWork> {
  const { data } = await apiClient.post(`${BASE}/works/${id}/verify`, { level })
  return data
}
/** Asks the text model for a scene draft (reasoning models take a minute or two). */
export async function generateTheme(id: number): Promise<ShihuaWork> {
  const { data } = await apiClient.post(`${BASE}/works/${id}/theme`, {}, { timeout: 200000 })
  return data
}
export async function redraw(id: number, count: number, promptTheme?: string): Promise<ShihuaImage[]> {
  const { data } = await apiClient.post(`${BASE}/works/${id}/redraw`, { count, prompt_theme: promptTheme ?? '' })
  return data.items
}

export async function listJobs(): Promise<ShihuaJob[]> {
  const { data } = await apiClient.get(`${BASE}/jobs`)
  return data.items
}
export async function createJob(workIds: number[], candidates: number, name = ''): Promise<ShihuaJob> {
  const { data } = await apiClient.post(`${BASE}/jobs`, { work_ids: workIds, candidates, name })
  return data
}
/** jobId 0 cancels every queued drawing. */
export async function cancelJob(jobId: number): Promise<number> {
  const { data } = await apiClient.post(`${BASE}/jobs/${jobId}/cancel`)
  return data.canceled
}
export async function retryJob(jobId: number): Promise<number> {
  const { data } = await apiClient.post(`${BASE}/jobs/${jobId}/retry`)
  return data.queued
}

export async function listImages(params: { work_ids?: string; job_id?: number; status?: string } = {}): Promise<ShihuaImage[]> {
  const { data } = await apiClient.get(`${BASE}/images`, { params })
  return data.items
}
export async function reviewImage(id: number, action: 'pick' | 'unpick' | 'reject' | 'restore' | 'title' | 'upscale', titleCheck = ''): Promise<ShihuaImage> {
  const { data } = await apiClient.post(`${BASE}/images/${id}/review`, { action, title_check: titleCheck })
  return data
}

export async function saveRender(imageId: number, aspect: string, file: Blob, layout: unknown): Promise<ShihuaRender> {
  const form = new FormData()
  form.append('file', file, `${aspect}.jpg`)
  form.append('layout', JSON.stringify(layout))
  const { data } = await apiClient.put(`${BASE}/images/${imageId}/renders/${aspect}`, form, { timeout: 120000 })
  return data
}
export async function listRenders(workIds?: number[]): Promise<ShihuaRender[]> {
  const { data } = await apiClient.get(`${BASE}/renders`, { params: workIds ? { work_ids: workIds.join(',') } : {} })
  return data.items
}
export async function exportZip(workIds: number[], aspects: string[]): Promise<Blob> {
  const { data } = await apiClient.get(`${BASE}/export`, {
    params: { work_ids: workIds.join(','), aspects: aspects.join(',') },
    responseType: 'blob',
    timeout: 300000
  })
  return data
}
