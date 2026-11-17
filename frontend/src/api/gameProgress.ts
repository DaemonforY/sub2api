/** 小游戏可选登录: cloud saves and leaderboards. Saves and scores need a signed-in user; boards are public. */
import { apiClient } from './client'

export interface GameSave {
  data: Record<string, unknown>
  updated_at: string | null
}

export interface GameScoreResult {
  best: number
  improved: boolean
  rank: number
  players: number
}

export interface GameBoardEntry {
  rank: number
  display_name: string
  score: number
  detail?: Record<string, unknown>
  is_current_user: boolean
}

export interface GameLeaderboard {
  game: string
  board: string
  players: number
  top: GameBoardEntry[]
  me?: GameBoardEntry
}

export async function getGameSave(game: string): Promise<GameSave> {
  const { data } = await apiClient.get(`/games/progress/${game}/save`)
  return data
}

export async function putGameSave(game: string, save: Record<string, unknown>): Promise<GameSave> {
  const { data } = await apiClient.put(`/games/progress/${game}/save`, { data: save })
  return data
}

export async function submitGameScore(game: string, score: number, detail: Record<string, unknown> = {}, board = 'main'): Promise<GameScoreResult> {
  const { data } = await apiClient.post(`/games/progress/${game}/score`, { board, score: Math.max(0, Math.floor(score)), detail })
  return data
}

export async function getGameLeaderboard(game: string, board = 'main'): Promise<GameLeaderboard> {
  const { data } = await apiClient.get(`/games/progress/${game}/leaderboard`, { params: { board } })
  return data
}
