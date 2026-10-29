/** 诗词灵感 for users: poem search and pages (open), 按画面找 (signed in; free runs, then billed). */
import { apiClient } from './client'

export interface Poem {
  id: number
  collection: string
  title: string
  author: string
  dynasty: string
  content: string
  notes?: string
  tags: string[]
  famous: boolean
  score?: number
}

export interface PoemsConfig {
  enabled: boolean
  find_enabled: boolean
  collections: { id: string; name: string; count: number }[]
  total: number
  free_left: number
  free_per_day: number
  find_price: number
  balance?: number
}

export interface PoemQuery {
  q?: string
  dynasty?: string
  collection?: string
  author?: string
  famous?: boolean
  random?: boolean
  offset?: number
}

export interface PoemFindResult {
  keywords: string[]
  picks: { poem: Poem; reason: string }[]
  candidates: Poem[]
  free: boolean
  charged: number
  free_left: number
  balance?: number
}

function params(q: PoemQuery) {
  const out: Record<string, string | number> = {}
  for (const [k, v] of Object.entries(q)) {
    if (v === undefined || v === '' || v === false || v === 0) continue
    out[k] = v === true ? 1 : (v as string | number)
  }
  return out
}

export async function poemsConfig(): Promise<PoemsConfig> {
  const { data } = await apiClient.get('/poems/config')
  return data
}

export async function searchPoems(q: PoemQuery): Promise<{ items: Poem[]; has_more: boolean }> {
  const { data } = await apiClient.get('/poems', { params: params(q) })
  return data
}

export async function getPoem(id: number): Promise<Poem> {
  const { data } = await apiClient.get(`/poems/${id}`)
  return data
}

export async function findPoems(input: { scene: string; dynasty?: string }): Promise<PoemFindResult> {
  const { data } = await apiClient.post('/poems/find', input, { timeout: 300000 })
  return data
}
