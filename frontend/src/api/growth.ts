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
  /** Global share (%) of each invitee payment the inviter earns; per-user overrides not reflected. */
  inviter_rebate_rate_percent?: number
  /** 老用户锁价: buyers keep their price on renewal within price_lock_grace_days of expiry. */
  price_lock_enabled: boolean
  price_lock_grace_days: number
  /** One key serves both subscriptions and balance (smart billing). */
  smart_billing?: boolean
  leaderboard_enabled: boolean
  edu_verify_enabled: boolean
  edu_discount_percent: number
  edu_email_suffixes: string[]
  /** 返利提现是否开放（人工打款） */
  withdraw_enabled?: boolean
  withdraw_min_cny?: number
  /** 首充奖励: % of the first gateway-paid top-up given back as balance (0 = off). */
  first_topup_bonus_percent?: number
  first_topup_bonus_cap?: number
  first_topup_min_amount?: number
}

export type WithdrawMethod = 'alipay' | 'wechat'
export type WithdrawStatusValue = 'pending' | 'paid' | 'rejected' | 'cancelled'

export interface AffiliateWithdrawal {
  id: number
  user_id: number
  user_email?: string
  username?: string
  /** 扣除的返利额度（与「可用返利」同单位） */
  quota_amount: number
  /** 应打款金额（人民币） */
  cny_amount: number
  method: WithdrawMethod
  account: string
  real_name: string
  user_note: string
  status: WithdrawStatusValue
  /** 打款备注或驳回原因 */
  admin_note: string
  reviewed_at?: string | null
  created_at: string
}

export interface WithdrawStatus {
  enabled: boolean
  min_cny: number
  /** 0 = 不限 */
  monthly_limit: number
  month_used: number
  withdrawable_cny: number
  available_quota: number
  /** 付费订单带来的返利，折合人民币（累计） */
  cash_cny: number
  /** 已提现 + 处理中（人民币） */
  withdrawn_cny: number
  has_pending: boolean
  last_method?: WithdrawMethod
  last_account?: string
  last_real_name?: string
  withdrawals: AffiliateWithdrawal[]
  freeze_hours: number
}

export interface WithdrawRequestPayload {
  cny_amount: number
  method: WithdrawMethod
  account: string
  real_name: string
  note?: string
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
  price_lock_enabled: boolean
  price_lock_grace_days: number
  leaderboard_enabled: boolean
  edu_verify_enabled: boolean
  edu_email_suffixes: string[]
  edu_discount_percent: number
  withdraw_enabled: boolean
  withdraw_min_cny: number
  /** 0 = 不限 */
  withdraw_monthly_limit: number
  first_topup_bonus_percent: number
  first_topup_bonus_cap: number
  first_topup_min_amount: number
  abandoned_order_reminder: boolean
  trial_exhausted_email: boolean
  winback_email: boolean
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
  async getWithdrawStatus(): Promise<WithdrawStatus> {
    const { data } = await apiClient.get<WithdrawStatus>('/user/aff/withdraw')
    return data
  },
  async requestWithdraw(payload: WithdrawRequestPayload): Promise<AffiliateWithdrawal> {
    const { data } = await apiClient.post<AffiliateWithdrawal>('/user/aff/withdraw', payload)
    return data
  },
  async cancelWithdraw(id: number): Promise<AffiliateWithdrawal> {
    const { data } = await apiClient.post<AffiliateWithdrawal>(`/user/aff/withdraw/${id}/cancel`)
    return data
  },
}

/** Members holding an active price lock on a plan on sale. */
export interface PlanPriceLockStat {
  plan_id: number
  plan_name: string
  price: number
  locked: number
  min_locked: number
  max_locked: number
  /** Locks below today's price. */
  below: number
}

export const adminGrowthAPI = {
  async getSmartBilling(): Promise<{ enabled: boolean }> {
    const { data } = await apiClient.get<{ enabled: boolean }>('/admin/growth/smart-billing')
    return data
  },
  async setSmartBilling(enabled: boolean): Promise<{ enabled: boolean }> {
    const { data } = await apiClient.put<{ enabled: boolean }>('/admin/growth/smart-billing', { enabled })
    return data
  },
  async priceLocks(): Promise<PlanPriceLockStat[]> {
    const { data } = await apiClient.get<PlanPriceLockStat[]>('/admin/growth/price-locks')
    return data
  },
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
  async listWithdrawals(params: { status?: WithdrawStatusValue | ''; search?: string; page?: number; page_size?: number }): Promise<BasePaginationResponse<AffiliateWithdrawal>> {
    const { data } = await apiClient.get<BasePaginationResponse<AffiliateWithdrawal>>('/admin/growth/withdrawals', { params })
    return data
  },
  async withdrawalPendingCount(): Promise<number> {
    const { data } = await apiClient.get<{ count: number }>('/admin/growth/withdrawals/pending-count')
    return data.count
  },
  /** note: 打款流水号等，可留空 */
  async markWithdrawalPaid(id: number, note: string): Promise<AffiliateWithdrawal> {
    const { data } = await apiClient.post<AffiliateWithdrawal>(`/admin/growth/withdrawals/${id}/paid`, { note })
    return data
  },
  /** note: 驳回原因，用户会看到 */
  async rejectWithdrawal(id: number, note: string): Promise<AffiliateWithdrawal> {
    const { data } = await apiClient.post<AffiliateWithdrawal>(`/admin/growth/withdrawals/${id}/reject`, { note })
    return data
  },
}
