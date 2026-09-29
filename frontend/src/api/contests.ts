/**
 * Public contest API (drawing contests etc.).
 * Browsing is anonymous; submitting and voting require login.
 */
import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export type ContestStatus = 'draft' | 'published' | 'settled' | 'cancelled'
export type ContestPhase =
  | 'draft'
  | 'upcoming'
  | 'submitting'
  | 'submitting_voting'
  | 'waiting_vote'
  | 'voting'
  | 'tallying'
  | 'settled'
  | 'cancelled'

export interface ContestPrize {
  rank_from: number
  rank_to: number
  type: 'balance' | 'custom'
  amount: number
  label: string
}

export interface Contest {
  id: number
  title: string
  description: string
  rules: string
  cover_image: string
  status: ContestStatus
  phase: ContestPhase
  submission_start_at: string
  submission_end_at: string
  voting_start_at: string
  voting_end_at: string
  max_entries_per_user: number
  votes_per_user: number
  allow_self_vote: boolean
  require_review: boolean
  min_account_age_hours: number
  one_prize_per_user: boolean
  min_votes_for_prize: number
  prizes: ContestPrize[]
  settled_at?: string
  entry_count: number
  vote_count: number
  created_at: string
  updated_at: string
}

export interface ContestEntry {
  id: number
  contest_id: number
  user_id: number
  title: string
  description: string
  prompt: string
  status: 'pending' | 'approved' | 'rejected' | 'withdrawn' | 'disqualified'
  review_note: string
  vote_count: number
  final_rank?: number
  final_votes?: number
  image_url: string
  author_name: string
  user_email?: string
  voted_by_me: boolean
  is_mine: boolean
  created_at: string
}

export interface ContestAward {
  id: number
  contest_id: number
  entry_id: number
  user_id: number
  place: number
  prize_type: 'balance' | 'custom'
  amount: number
  label: string
  status: 'pending' | 'granting' | 'granted' | 'failed'
  note: string
  granted_at?: string
  entry_title?: string
  author_name?: string
  user_email?: string
}

export interface ContestViewer {
  logged_in: boolean
  votes_used: number
  votes_left: number
  voted_entry_ids: number[]
  my_entries: ContestEntry[]
  entries_left: number
  can_submit: boolean
  can_vote: boolean
  vote_blocked_reason?: string
}

export interface ContestDetail {
  contest: Contest
  viewer: ContestViewer
  awards?: ContestAward[]
}

export interface ContestLeaderboardRow {
  rank: number
  entry_id: number
  title: string
  author_name: string
  image_url: string
  votes: number
  final: boolean
}

export async function list(): Promise<Contest[]> {
  const { data } = await apiClient.get('/contests')
  return data
}

export async function get(id: number): Promise<ContestDetail> {
  const { data } = await apiClient.get(`/contests/${id}`)
  return data
}

export async function listEntries(
  id: number,
  params: { sort?: 'votes' | 'new'; page?: number; page_size?: number }
): Promise<PaginatedResponse<ContestEntry>> {
  const { data } = await apiClient.get(`/contests/${id}/entries`, { params })
  return data
}

export async function leaderboard(id: number, limit = 10): Promise<ContestLeaderboardRow[]> {
  const { data } = await apiClient.get(`/contests/${id}/leaderboard`, { params: { limit } })
  return data
}

export async function submitEntry(
  id: number,
  input: { title: string; description?: string; prompt?: string; image: File }
): Promise<ContestEntry> {
  const form = new FormData()
  form.append('title', input.title)
  form.append('description', input.description || '')
  form.append('prompt', input.prompt || '')
  form.append('image', input.image)
  const { data } = await apiClient.post(`/contests/${id}/entries`, form)
  return data
}

export async function withdrawEntry(id: number, entryId: number): Promise<void> {
  await apiClient.delete(`/contests/${id}/entries/${entryId}`)
}

export async function vote(id: number, entryId: number): Promise<void> {
  await apiClient.post(`/contests/${id}/entries/${entryId}/vote`)
}

export async function unvote(id: number, entryId: number): Promise<void> {
  await apiClient.delete(`/contests/${id}/entries/${entryId}/vote`)
}

export const contestsAPI = { list, get, listEntries, leaderboard, submitEntry, withdrawEntry, vote, unvote }
export default contestsAPI
