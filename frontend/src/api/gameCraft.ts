/** 小游戏「AI 万物合成」(/games/craft): open to visitors; new pairs are limited per day on the server. */
import { apiClient } from './client'

export interface CraftElement {
  id: number
  name: string
  emoji: string
  desc: string
  created_at?: string
}

export interface CraftStart {
  starters: CraftElement[]
  total: number
  recent: CraftElement[]
}

export interface CraftResult {
  element: CraftElement
  /** True only for whoever made the element first on the whole site. */
  first: boolean
}

export async function craftStart(): Promise<CraftStart> {
  const { data } = await apiClient.get('/games/craft/start')
  return data
}

export async function craftCombine(a: string, b: string): Promise<CraftResult> {
  const { data } = await apiClient.post('/games/craft/combine', { a, b }, { timeout: 90000 })
  return data
}
