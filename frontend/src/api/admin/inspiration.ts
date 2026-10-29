/** Admin API for the 灵感库: the public-domain poem corpus, find-by-scene, and song cards (lyrics are internal only). */
import { apiClient } from '../client'

const BASE = '/admin/inspiration'

export interface LibPoem {
  id: number
  collection: string
  title: string
  author: string
  dynasty: string
  content: string
  notes?: string
  tags: string[]
  famous: boolean
  popularity: number
  starred: boolean
  score?: number
  in_shihua?: string
}

export interface LibPoemQuery {
  q?: string
  dynasty?: string
  collection?: string
  author?: string
  famous?: boolean
  starred?: boolean
  random?: boolean
  limit?: number
  offset?: number
}

export interface LibAllusion {
  title: string
  author: string
  poem_id?: number
  note: string
}

export interface LibSong {
  id: number
  title: string
  artist: string
  lyricist: string
  composer: string
  year?: number | null
  album: string
  lyrics?: string
  has_lyrics: boolean
  summary: string
  imagery: string[]
  moods: string[]
  tags: string[]
  allusions: LibAllusion[]
  notes: string
  link_url: string
  starred: boolean
  created_at: string
  updated_at: string
  matches?: string[]
}

export type LibSongInput = Omit<LibSong, 'id' | 'has_lyrics' | 'starred' | 'created_at' | 'updated_at' | 'matches'>

export interface LibSongQuery {
  q?: string
  artist?: string
  tag?: string
  year_from?: number
  year_to?: number
  starred?: boolean
  limit?: number
  offset?: number
}

export interface LibFacet {
  name: string
  count: number
}

export interface LibSongFacets {
  artists: LibFacet[]
  tags: LibFacet[]
  years: number[]
}

export interface LibPick {
  poem: LibPoem
  reason: string
}

export interface LibFindResult {
  keywords: string[]
  picks: LibPick[]
  candidates: LibPoem[]
  songs: LibSong[]
  model: string
}

export interface LibSongAnalysis {
  summary: string
  imagery: string[]
  moods: string[]
  tags: string[]
  allusions: LibAllusion[]
  model: string
  dropped: string[]
}

/** Model calls (find, analyze) can take a few minutes with a reasoning model. */
const MODEL_TIMEOUT = 300000

function flags(q: Record<string, unknown>) {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(q)) {
    if (v === undefined || v === '' || v === false || v === 0) continue
    out[k] = v === true ? 1 : v
  }
  return out
}

export async function searchPoems(q: LibPoemQuery): Promise<{ items: LibPoem[]; has_more: boolean }> {
  const { data } = await apiClient.get(`${BASE}/poems`, { params: flags({ ...q }) })
  return data
}

export async function poemCounts(): Promise<{ counts: Record<string, number>; names: Record<string, string> }> {
  const { data } = await apiClient.get(`${BASE}/poems/counts`)
  return data
}

export async function getPoem(id: number): Promise<LibPoem> {
  const { data } = await apiClient.get(`${BASE}/poems/${id}`)
  return data
}

export async function starPoem(id: number, starred: boolean): Promise<void> {
  await apiClient.post(`${BASE}/poems/${id}/star`, { starred })
}

export async function poemToShihua(id: number, stage = ''): Promise<{ work: { id: number; slug: string; title: string }; created: boolean }> {
  const { data } = await apiClient.post(`${BASE}/poems/${id}/to-shihua`, { stage })
  return data
}

export async function findByScene(input: { scene: string; dynasty?: string; collection?: string; count?: number; rerank?: boolean }): Promise<LibFindResult> {
  const { data } = await apiClient.post(`${BASE}/find`, input, { timeout: MODEL_TIMEOUT })
  return data
}

export async function listSongs(q: LibSongQuery): Promise<{ items: LibSong[]; total: number }> {
  const { data } = await apiClient.get(`${BASE}/songs`, { params: flags({ ...q }) })
  return data
}

export async function songFacets(): Promise<LibSongFacets> {
  const { data } = await apiClient.get(`${BASE}/songs/facets`)
  return data
}

export async function getSong(id: number): Promise<LibSong> {
  const { data } = await apiClient.get(`${BASE}/songs/${id}`)
  return data
}

export async function createSong(input: LibSongInput): Promise<LibSong> {
  const { data } = await apiClient.post(`${BASE}/songs`, input)
  return data
}

export async function updateSong(id: number, input: LibSongInput): Promise<LibSong> {
  const { data } = await apiClient.put(`${BASE}/songs/${id}`, input)
  return data
}

export async function deleteSong(id: number): Promise<void> {
  await apiClient.delete(`${BASE}/songs/${id}`)
}

export async function starSong(id: number, starred: boolean): Promise<void> {
  await apiClient.post(`${BASE}/songs/${id}/star`, { starred })
}

export async function analyzeSong(id: number): Promise<LibSongAnalysis> {
  const { data } = await apiClient.post(`${BASE}/songs/${id}/analyze`, {}, { timeout: MODEL_TIMEOUT })
  return data
}

export interface LibPublicSettings {
  enabled: boolean
  find_enabled: boolean
  free_finds_per_day: number
  find_price: number
  find_model: string
  daily_find_cap: number
  guest_per_minute: number
  guest_per_day: number
  user_per_minute: number
  user_per_day: number
  find_count: number
}

export interface LibPublicStats {
  runs_today: number
  paid_today: number
  revenue_today: number
  paid_7d: number
  revenue_7d: number
}

export async function getPublicSettings(): Promise<{ settings: LibPublicSettings; stats: LibPublicStats }> {
  const { data } = await apiClient.get(`${BASE}/public`)
  return data
}

export async function savePublicSettings(s: LibPublicSettings): Promise<{ settings: LibPublicSettings }> {
  const { data } = await apiClient.put(`${BASE}/public`, s)
  return data
}
