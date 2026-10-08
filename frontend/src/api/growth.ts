/**
 * Growth programs: invitee first-order bonus, education verification, invite leaderboard.
 */
import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export interface GrowthPublicConfig {
  affiliate_enabled: boolean
  invitee_bonus_rate_percent: number
  invitee_bonus_cap: number
  /** Trial balance for signing up with an invite ($, 0 = off). */
  invitee_signup_bonus: number
  leaderboard_enabled: boolean
  edu_verify_enabled: boolean
  edu_discount_percent: number
  edu_email_suffixes: string[]
}

export interface EduVerification {
  user_id: number
  email: string
  /** email: verified with a code sent to a school email; manual: verified by an admin */
  method?: 'email' | 'manual'
  note?: string
  verified_at: string
  user_email?: string
  username?: string
}

export interface EduStatus {
  enabled: boolean
  discount_percent: number
  suffixes: string[]
  verification?: EduVerification | null
}

export type LeaderboardPeriod = 'month' | 'last_month' | 'all'

export interface InviteLeaderboardEntry {
  rank: number
  user_id?: number
  display_name: string
  email?: string
  username?: string
  invited_count: number
  paying_invitees: number
  invitee_paid?: number
  rebate_accrued?: number
  custom_rate_percent?: number | null
  is_current_user?: boolean
}

export interface InviteLeaderboard {
  period: string
  start?: string
  end?: string
  entries: InviteLeaderboardEntry[]
  me?: InviteLeaderboardEntry | null
}

export interface GrowthSettings {
  invitee_bonus_rate_percent: number
  invitee_bonus_cap: number
  invitee_signup_bonus: number
  invitee_signup_daily_limit: number
  leaderboard_enabled: boolean
  edu_verify_enabled: boolean
  edu_email_suffixes: string[]
  edu_discount_percent: number
}

export const growthAPI = {
  async getPublicConfig(): Promise<GrowthPublicConfig> {
    const { data } = await apiClient.get<GrowthPublicConfig>('/growth/config')
    return data
  },
  async getEduStatus(): Promise<EduStatus> {
    const { data } = await apiClient.get<EduStatus>('/user/edu')
    return data
  },
  async sendEduCode(email: string): Promise<void> {
    await apiClient.post('/user/edu/send-code', { email })
  },
  async verifyEdu(email: string, code: string): Promise<EduVerification> {
    const { data } = await apiClient.post<EduVerification>('/user/edu/verify', { email, code })
    return data
  },
  async getLeaderboard(period: LeaderboardPeriod): Promise<InviteLeaderboard> {
    const { data } = await apiClient.get<InviteLeaderboard>('/user/aff/leaderboard', { params: { period } })
    return data
  },
}

export const adminGrowthAPI = {
  async getSettings(): Promise<GrowthSettings> {
    const { data } = await apiClient.get<GrowthSettings>('/admin/growth/settings')
    return data
  },
  async updateSettings(settings: GrowthSettings): Promise<GrowthSettings> {
    const { data } = await apiClient.put<GrowthSettings>('/admin/growth/settings', settings)
    return data
  },
  async listEduVerifications(params: { page?: number; page_size?: number; search?: string }): Promise<BasePaginationResponse<EduVerification>> {
    const { data } = await apiClient.get<BasePaginationResponse<EduVerification>>('/admin/growth/edu-verifications', { params })
    return data
  },
  /** user: account email or ID; note: school and how it was checked */
  async grantEduVerification(user: string, note: string): Promise<EduVerification> {
    const { data } = await apiClient.post<EduVerification>('/admin/growth/edu-verifications', { user, note })
    return data
  },
  async revokeEduVerification(userId: number): Promise<void> {
    await apiClient.delete(`/admin/growth/edu-verifications/${userId}`)
  },
  async getLeaderboard(params: { start?: string; end?: string; limit?: number }): Promise<InviteLeaderboard> {
    const { data } = await apiClient.get<InviteLeaderboard>('/admin/growth/leaderboard', { params })
    return data
  },
}
