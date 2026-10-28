/** Admin API for the 诗画素材库: texts and segments, imagery, styles, prompts, drawings and review. */
import { apiClient } from '../client'

const BASE = '/admin/materials'

export type MaterialLicense = 'public_domain' | 'original' | 'licensed' | 'reference_only'
export type MaterialStatus = 'draft' | 'review' | 'published' | 'hidden'
export type MaterialArtworkStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'
export type MaterialReviewStatus = 'pending' | 'approved' | 'rejected'

export interface MaterialSegment {
  id: number
  text_id: number
  seq: number
  label: string
  content: string
  scene_desc: string
  is_famous: boolean
}

export interface MaterialText {
  id: number
  slug: string
  kind: string
  title: string
  author_id?: number
  performer: string
  body: string
  translation: string
  appreciation: string
  license_status: MaterialLicense
  license_note: string
  sources: string
  status: MaterialStatus
  external_url: string
  themes: string[]
  seasons: string[]
  shihua_work_id?: number
  created_at: string
  updated_at: string
  author_name: string
  dynasty: string
  segment_count: number
  segments?: MaterialSegment[]
  text_overlay_allowed: boolean
}

export interface MaterialImportResult {
  imported: number
  existing: number
  skipped: { slug: string; title: string; reason: string }[]
}

export interface MaterialImagery {
  id: number
  name: string
  aliases: string[]
  category: string
  meaning: string
  visual_zh: string
  visual_en: string
  emotions: string[]
  seasons: string[]
  times: string[]
  weathers: string[]
  eras: string[]
  style_ids: number[]
  status: MaterialStatus
  created_at: string
  updated_at: string
  segment_count: number
}

export interface MaterialImageryCandidate {
  rank: number
  name: string
  category: string
  aliases: string[]
  poems: number
  in_library: boolean
}

export interface MaterialBlockHit {
  term: string
  kind: string
  action: 'block' | 'warn' | 'strip'
  match: string
  reason: string
}

export interface MaterialSeedResult {
  drafted: string[]
  existing: number
  failed: string[]
  warnings: MaterialBlockHit[]
  model: string
  left: number
}

export interface MaterialSegmentContext extends MaterialSegment {
  kind: string
  title: string
  author_name: string
  dynasty: string
  license_status: MaterialLicense
  body: string
}

export interface MaterialSegmentImagery {
  segment_id: number
  imagery_id: number
  weight: number
  source: 'ai' | 'manual'
  confirmed: boolean
  name: string
  category: string
  status: MaterialStatus
  visual_zh: string
}

export interface MaterialExtractResult {
  segment: MaterialSegmentContext
  links: MaterialSegmentImagery[]
  created: string[]
  warnings: MaterialBlockHit[]
  model: string
}

export interface MaterialStyleOption {
  name: string
  prompt_en: string
}

export interface MaterialStyleOptions {
  compositions?: MaterialStyleOption[]
  palettes?: MaterialStyleOption[]
  moods?: MaterialStyleOption[]
}

export interface MaterialStyle {
  id: number
  name: string
  category: string
  desc_zh: string
  prompt_en: string
  negative_en: string
  suits_emotions: string[]
  suits_categories: string[]
  options: MaterialStyleOptions
  default_size: string
  default_quality: string
  enabled: boolean
  sort: number
}

export interface MaterialStyleScore {
  style: MaterialStyle
  score: number
  reasons: string[]
  options: MaterialStyleOptions
}

export interface MaterialPromptInput {
  style_id: number
  size?: string
  composition?: string
  palette?: string
  mood?: string
  extra?: string
  imagery_ids?: number[]
}

export interface MaterialPrompt {
  prompt: string
  parts: Record<string, unknown>
  size: string
  quality: string
  style_id: number
  imagery_ids: number[]
  blocked: boolean
  hits: MaterialBlockHit[]
  notes: string[]
  text_overlay_allowed: boolean
}

export interface MaterialArtwork {
  id: number
  segment_id?: number
  style_id?: number
  prompt_final: string
  model: string
  size: string
  quality: string
  status: MaterialArtworkStatus
  attempts: number
  not_before?: string
  error: string
  width: number
  height: number
  cost_usd: number
  duration_ms: number
  parent_id?: number
  rating: number
  review_status: MaterialReviewStatus
  reject_reason: string
  text_overlay_allowed: boolean
  created_at: string
  finished_at?: string
  imagery_ids: number[]
  style_name: string
  text_id?: number
  text_title: string
  segment_label: string
  url?: string
  thumb_url?: string
}

export interface MaterialQueue {
  queued: number
  running: number
  started_today: number
  concurrency: number
  daily_limit: number
  paused: string
}

export interface MaterialBlockTerm {
  id: number
  term: string
  kind: string
  match: 'contains' | 'regex'
  action: 'block' | 'warn' | 'strip'
  applies_to: 'prompt' | 'text' | 'both'
  reason: string
  enabled: boolean
}

export interface Page<T> {
  items: T[]
  total: number
}

// Text-model calls: reasoning models take a minute or two.
const MODEL_TIMEOUT = 200000

export async function listTexts(params: { kind?: string; status?: string; license?: string; q?: string; limit?: number; offset?: number } = {}): Promise<Page<MaterialText>> {
  const { data } = await apiClient.get(`${BASE}/texts`, { params })
  return data
}
export async function getText(id: number): Promise<MaterialText> {
  const { data } = await apiClient.get(`${BASE}/texts/${id}`)
  return data
}
export async function importShihua(): Promise<MaterialImportResult> {
  const { data } = await apiClient.post(`${BASE}/import/shihua`, {}, { timeout: 120000 })
  return data
}

export async function listImagery(params: { status?: string; category?: string; q?: string; limit?: number; offset?: number } = {}): Promise<Page<MaterialImagery> & { categories: string[] }> {
  const { data } = await apiClient.get(`${BASE}/imagery`, { params })
  return data
}
export async function updateImagery(id: number, input: Partial<MaterialImagery>): Promise<MaterialImagery> {
  const { data } = await apiClient.put(`${BASE}/imagery/${id}`, input)
  return data
}
export async function setImageryStatus(ids: number[], status: MaterialStatus): Promise<number> {
  const { data } = await apiClient.post(`${BASE}/imagery/status`, { ids, status })
  return data.updated
}
export async function listCandidates(): Promise<MaterialImageryCandidate[]> {
  const { data } = await apiClient.get(`${BASE}/imagery/candidates`)
  return data.items
}
export async function draftSeeds(limit: number): Promise<MaterialSeedResult> {
  const { data } = await apiClient.post(`${BASE}/imagery/seed`, { limit }, { timeout: MODEL_TIMEOUT * 2 })
  return data
}

export async function getSegmentImagery(segmentId: number): Promise<{ segment: MaterialSegmentContext; links: MaterialSegmentImagery[] }> {
  const { data } = await apiClient.get(`${BASE}/segments/${segmentId}/imagery`)
  return data
}
export async function extractImagery(segmentId: number): Promise<MaterialExtractResult> {
  const { data } = await apiClient.post(`${BASE}/segments/${segmentId}/imagery/extract`, {}, { timeout: MODEL_TIMEOUT })
  return data
}
export async function saveSegmentImagery(segmentId: number, links: { imagery_id: number; weight: number }[], sceneDesc?: string): Promise<MaterialExtractResult> {
  const { data } = await apiClient.put(`${BASE}/segments/${segmentId}/imagery`, { links, scene_desc: sceneDesc })
  return data
}
export async function recommendStyles(segmentId: number): Promise<MaterialStyleScore[]> {
  const { data } = await apiClient.get(`${BASE}/segments/${segmentId}/styles`)
  return data.items
}
export async function buildPrompt(segmentId: number, input: MaterialPromptInput): Promise<MaterialPrompt> {
  const { data } = await apiClient.post(`${BASE}/segments/${segmentId}/prompt`, input)
  return data
}
export async function drawSegment(segmentId: number, input: MaterialPromptInput, count: number): Promise<MaterialArtwork[]> {
  const { data } = await apiClient.post(`${BASE}/segments/${segmentId}/draw`, { ...input, count })
  return data.items
}

export async function listArtworks(params: { segment_id?: number; text_id?: number; status?: string; review_status?: string; limit?: number; offset?: number } = {}): Promise<Page<MaterialArtwork>> {
  const { data } = await apiClient.get(`${BASE}/artworks`, { params })
  return data
}
export async function redrawArtwork(id: number): Promise<MaterialArtwork> {
  const { data } = await apiClient.post(`${BASE}/artworks/${id}/redraw`)
  return data
}
export async function reviewArtwork(id: number, input: { review_status: MaterialReviewStatus; rating: number; reject_reason?: string }): Promise<MaterialArtwork> {
  const { data } = await apiClient.post(`${BASE}/artworks/${id}/review`, input)
  return data
}
export async function retryArtwork(id: number): Promise<MaterialArtwork> {
  const { data } = await apiClient.post(`${BASE}/artworks/${id}/retry`)
  return data
}
export async function cancelArtworks(ids: number[]): Promise<number> {
  const { data } = await apiClient.post(`${BASE}/artworks/cancel`, { ids })
  return data.canceled
}
export async function getQueue(): Promise<MaterialQueue> {
  const { data } = await apiClient.get(`${BASE}/queue`)
  return data
}

export async function listStyles(): Promise<MaterialStyle[]> {
  const { data } = await apiClient.get(`${BASE}/styles`)
  return data.items
}
export async function listBlocklist(): Promise<MaterialBlockTerm[]> {
  const { data } = await apiClient.get(`${BASE}/blocklist`)
  return data.items
}
export async function checkPrompt(input: string, target: 'prompt' | 'text'): Promise<{ blocked: boolean; hits: MaterialBlockHit[]; prompt: string }> {
  const { data } = await apiClient.post(`${BASE}/check`, { input, target })
  return data
}

export interface MaterialRender {
  id: number
  artwork_id: number
  aspect: string
  file: string
  width: number
  height: number
  layout: Record<string, unknown>
  created_at: string
  url?: string
}

/** Uploads one laid-out poster; layout must carry the words set on it (text, byline). */
export async function saveRender(artworkId: number, aspect: string, file: Blob, layout: unknown): Promise<MaterialRender> {
  const form = new FormData()
  form.append('file', file, `${aspect}.jpg`)
  form.append('layout', JSON.stringify(layout))
  const { data } = await apiClient.put(`${BASE}/artworks/${artworkId}/renders/${aspect}`, form, { timeout: 120000 })
  return data
}
export async function listRenders(artworkIds: number[]): Promise<MaterialRender[]> {
  const { data } = await apiClient.get(`${BASE}/renders`, { params: { artwork_ids: artworkIds.join(',') } })
  return data.items
}
export async function exportZip(artworkIds: number[], aspects: string[]): Promise<Blob> {
  const { data } = await apiClient.get(`${BASE}/export`, {
    params: { artwork_ids: artworkIds.join(','), aspects: aspects.join(',') },
    responseType: 'blob',
    timeout: 300000
  })
  return data
}
