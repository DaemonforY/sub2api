/**
 * Admin prompt library API: curate the canvas prompt library (scenes, tags, visibility), review
 * prompts users share, manage the synced community sources.
 */
import { apiClient } from '../client'

export const PROMPT_SCENES = ['poster', 'ecommerce', 'ui', 'infographic', 'portrait', 'photo', 'illustration', '3d', 'brand', 'social', 'life', 'history', 'creative', 'video', 'other'] as const
export type PromptScene = (typeof PROMPT_SCENES)[number]
export type PromptStatus = 'active' | 'hidden' | 'pending' | 'rejected' | 'duplicate'
export const PROMPT_STATUSES: PromptStatus[] = ['active', 'pending', 'hidden', 'rejected', 'duplicate']
export const PROMPT_MODELS = ['gpt-image-2', 'nano-banana', 'gpt-4o', 'unknown'] as const

export interface PromptItem {
  id: number
  source_id: string
  source_name: string
  external_id: string
  owner_user_id?: number
  owner_email?: string
  kind: 'image' | 'video'
  title: string
  /** Source title when a Chinese translation is shown instead. */
  original_title?: string
  prompt: string
  description: string
  cover_url: string
  reference_image_urls: string[]
  source_tags: string[]
  scenes: string[]
  tags: string[]
  model: string
  lang: string
  needs_reference: boolean
  auto_flags?: string[]
  author: string
  source_url: string
  visibility: 'public' | 'private'
  status: PromptStatus
  review_note?: string
  curated: boolean
  featured: boolean
  quality_score: number
  use_count: number
  favorite_count: number
  published_at?: string
  created_at: string
  updated_at: string
}

export interface PromptListResult {
  items: PromptItem[]
  total: number
  page: number
  page_size: number
  scene_counts: Record<string, number>
}

export interface PromptListParams {
  q?: string
  status?: string
  source?: string
  scene?: string
  kind?: string
  tag?: string
  curated?: string
  featured?: string
  sort?: string
  page?: number
  page_size?: number
}

export interface PromptAdminInput {
  title: string
  prompt: string
  description: string
  cover_url: string
  kind: 'image' | 'video'
  scenes: string[]
  tags: string[]
  model: string
  needs_reference: boolean
  status: PromptStatus
  visibility: 'public' | 'private'
  featured: boolean
  review_note: string
}

export type PromptBatchAction = 'add_scenes' | 'remove_scenes' | 'set_scenes' | 'add_tags' | 'remove_tags' | 'set_status' | 'set_featured' | 'mark_reviewed'

export interface PromptBatchInput {
  ids: number[]
  action: PromptBatchAction
  scenes?: string[]
  tags?: string[]
  status?: PromptStatus
  note?: string
  featured?: boolean
}

export interface PromptSource {
  id: string
  name: string
  format: string
  url: string
  homepage: string
  enabled: boolean
  item_count: number
  active_count: number
  last_synced_at?: string
  last_error: string
  syncing: boolean
}

export interface PromptLibraryStats {
  status_counts: Record<string, number>
  pending_user: number
  uncurated: number
  total_uses: number
}

export async function list(params: PromptListParams): Promise<PromptListResult> {
  const { data } = await apiClient.get('/admin/prompt-library/items', { params })
  return data
}
export async function get(id: number): Promise<PromptItem> {
  const { data } = await apiClient.get(`/admin/prompt-library/items/${id}`)
  return data
}
export async function create(input: PromptAdminInput): Promise<PromptItem> {
  const { data } = await apiClient.post('/admin/prompt-library/items', input)
  return data
}
export async function update(id: number, input: PromptAdminInput): Promise<PromptItem> {
  const { data } = await apiClient.put(`/admin/prompt-library/items/${id}`, input)
  return data
}
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/prompt-library/items/${id}`)
}
export async function batch(input: PromptBatchInput): Promise<{ updated: number }> {
  const { data } = await apiClient.post('/admin/prompt-library/items/batch', input)
  return data
}
export async function stats(): Promise<PromptLibraryStats> {
  const { data } = await apiClient.get('/admin/prompt-library/stats')
  return data
}
export async function tags(): Promise<{ tag: string; count: number }[]> {
  const { data } = await apiClient.get('/admin/prompt-library/tags')
  return data
}
export async function sources(): Promise<PromptSource[]> {
  const { data } = await apiClient.get('/admin/prompt-library/sources')
  return data
}
export async function setSourceEnabled(id: string, enabled: boolean): Promise<void> {
  await apiClient.put(`/admin/prompt-library/sources/${encodeURIComponent(id)}`, { enabled })
}
export async function syncSource(id: string): Promise<void> {
  await apiClient.post(`/admin/prompt-library/sources/${encodeURIComponent(id)}/sync`)
}

export interface PromptTranslationStatus {
  base_url: string
  model: string
  api_key_configured: boolean
  untranslated: number
  unchecked_scenes: number
  running: boolean
  last_run_at?: string
  last_translated: number
  last_scenes: number
  last_error: string
}

export interface PromptTranslationInput {
  base_url: string
  model: string
  api_key?: string
  clear_api_key?: boolean
}

export async function translationStatus(): Promise<PromptTranslationStatus> {
  const { data } = await apiClient.get('/admin/prompt-library/translation')
  return data
}
export async function saveTranslation(input: PromptTranslationInput): Promise<PromptTranslationStatus> {
  const { data } = await apiClient.put('/admin/prompt-library/translation', input)
  return data
}
export async function runTranslation(): Promise<void> {
  await apiClient.post('/admin/prompt-library/translation/run')
}

/** Form values for an item (the editor edits a copy). */
export function toAdminInput(item: PromptItem): PromptAdminInput {
  return {
    title: item.title,
    prompt: item.prompt,
    description: item.description,
    cover_url: item.cover_url,
    kind: item.kind,
    scenes: [...item.scenes],
    tags: [...item.tags],
    model: item.model,
    needs_reference: item.needs_reference,
    status: item.status,
    visibility: item.visibility,
    featured: item.featured,
    review_note: item.review_note || ''
  }
}

/** Splits "国风, 手绘 #山水" style input into tags. */
export function parseTags(text: string): string[] {
  const out: string[] = []
  for (const raw of text.split(/[,，、\s#＃]+/)) {
    const tag = raw.trim()
    if (tag && !out.includes(tag)) out.push(tag)
  }
  return out
}

export default { list, get, create, update, remove, batch, stats, tags, sources, setSourceEnabled, syncSource, translationStatus, saveTranslation, runTranslation }
