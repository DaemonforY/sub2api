/**
 * Admin contest API: configuration, moderation, vote review, settlement, prizes.
 */
import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { Contest, ContestAward, ContestEntry, ContestPrize } from '@/api/contests'

export interface ContestInput {
  title: string
  description: string
  rules: string
  cover_image: string
  status: 'draft' | 'published'
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
}

/** The monthly contest template; {year} / {month} in the texts become the month's. */
export interface ContestAutoTemplate {
  title_template: string
  description: string
  rules: string
  cover_image: string
  /** Submissions close at the end of this day (1–28). */
  submission_end_day: number
  /** "20:00" on the month's last day. */
  voting_end_time: string
  /** Published this many days before the 1st. */
  lead_days: number
  max_entries_per_user: number
  votes_per_user: number
  allow_self_vote: boolean
  require_review: boolean
  min_account_age_hours: number
  one_prize_per_user: boolean
  min_votes_for_prize: number
  prizes: ContestPrize[]
}

export type ContestAutoMonthStatus = 'created' | 'scheduled' | 'paused' | 'missed' | 'skipped'

export interface ContestAutoMonth {
  period: string
  title: string
  submission_start_at: string
  submission_end_at: string
  voting_end_at: string
  create_at: string
  status: ContestAutoMonthStatus
  contest_id?: number
  phase?: string
}

export interface ContestAutoSettings {
  enabled: boolean
  template: ContestAutoTemplate
  updated_at?: string
  months: ContestAutoMonth[]
}

export interface ContestVote {
  id: number
  entry_id: number
  user_id: number
  client_ip: string
  created_at: string
  user_email?: string
  user_created_at?: string
}

export async function list(): Promise<Contest[]> {
  const { data } = await apiClient.get('/admin/contests')
  return data
}
export async function get(id: number): Promise<Contest> {
  const { data } = await apiClient.get(`/admin/contests/${id}`)
  return data
}
export async function create(input: ContestInput): Promise<Contest> {
  const { data } = await apiClient.post('/admin/contests', input)
  return data
}
export async function update(id: number, input: ContestInput): Promise<Contest> {
  const { data } = await apiClient.put(`/admin/contests/${id}`, input)
  return data
}
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/contests/${id}`)
}
export async function cancel(id: number): Promise<Contest> {
  const { data } = await apiClient.post(`/admin/contests/${id}/cancel`)
  return data
}
export async function settle(id: number): Promise<Contest> {
  const { data } = await apiClient.post(`/admin/contests/${id}/settle`)
  return data
}
export async function listEntries(
  id: number,
  params: { status?: string; sort?: 'votes' | 'new' | 'rank'; page?: number; page_size?: number }
): Promise<PaginatedResponse<ContestEntry>> {
  const { data } = await apiClient.get(`/admin/contests/${id}/entries`, { params })
  return data
}
export async function reviewEntry(id: number, entryId: number, status: string, note = ''): Promise<void> {
  await apiClient.put(`/admin/contests/${id}/entries/${entryId}/status`, { status, note })
}
export async function listEntryVotes(id: number, entryId: number): Promise<ContestVote[]> {
  const { data } = await apiClient.get(`/admin/contests/${id}/entries/${entryId}/votes`)
  return data
}
export async function voidVote(id: number, voteId: number): Promise<void> {
  await apiClient.delete(`/admin/contests/${id}/votes/${voteId}`)
}
export async function listAwards(id: number): Promise<ContestAward[]> {
  const { data } = await apiClient.get(`/admin/contests/${id}/awards`)
  return data
}
export async function grantAward(id: number, awardId: number, note = ''): Promise<ContestAward> {
  const { data } = await apiClient.post(`/admin/contests/${id}/awards/${awardId}/grant`, { note })
  return data
}
export async function grantAllBalance(id: number): Promise<{ granted: number; failed: number }> {
  const { data } = await apiClient.post(`/admin/contests/${id}/awards/grant-balance`)
  return data
}

export async function getAuto(): Promise<ContestAutoSettings> {
  const { data } = await apiClient.get('/admin/contests/auto')
  return data
}

/** Starts or stops the monthly schedule and saves its template; due months are created at once. */
export async function saveAuto(enabled: boolean, template: ContestAutoTemplate): Promise<ContestAutoSettings> {
  const { data } = await apiClient.put('/admin/contests/auto', { enabled, template })
  return data
}

export const adminContestsAPI = {
  list, get, create, update, remove, cancel, settle, getAuto, saveAuto,
  listEntries, reviewEntry, listEntryVotes, voidVote,
  listAwards, grantAward, grantAllBalance
}
export default adminContestsAPI
