/** Admin API for 用户画像 (user insights), 风控事件 (risk events) and 静默限制 (silent restriction). */
import { apiClient } from '../client'

export interface InsightsUserRef {
  id: number
  email: string
  username: string
  status: string
  created_at: string
  shadowed: boolean
  /** lifetime payments through a payment provider (CNY) */
  paid_cny: number
  /** usage cost in the window (USD) */
  spend_usd: number
}

export interface InsightsSegments {
  inactive: number
  trial: number
  paying: number
  churned: number
}

export interface InsightsSignupDay extends InsightsSegments {
  date: string
}

export interface InsightsSpendDay {
  date: string
  /** usage by accounts that never paid */
  free_usd: number
  used_usd: number
  paid_cny: number
}

/** codex_desktop | codex_cli | claude_code | cherry_studio | relay | browser | sdk | other */
export type ClientClass = string

export interface InsightsClient {
  class: ClientClass
  requests: number
  users: number
  cost_usd: number
}

export interface InsightsCluster {
  kind: 'visitor' | 'ip_prefix'
  key: string
  users: InsightsUserRef[]
  free_usd: number
  paid_cny: number
}

export interface InsightsRelayUser {
  user: InsightsUserRef
  user_agent: string
  requests: number
  cost_usd: number
}

export interface UserInsightsOverview {
  days: number
  since: string
  segments: InsightsSegments
  signups: InsightsSignupDay[]
  spend: InsightsSpendDay[]
  clients: InsightsClient[]
  clusters: InsightsCluster[]
  relay_users: InsightsRelayUser[]
}

export interface InsightsCount {
  key: string
  count: number
  cost_usd?: number
  last_seen: string
}

export interface InsightsClientUA {
  class: ClientClass
  user_agent: string
  requests: number
  cost_usd: number
  last_seen: string
}

export interface InsightsPayment {
  id: number
  order_type: string
  payment_type: string
  amount: number
  pay_amount: number
  status: string
  created_at: string
  paid_at?: string
}

export interface InsightsSubscription {
  id: number
  group_name: string
  status: string
  /** the sign-up default (trial) subscription */
  free: boolean
  starts_at: string
  expires_at: string
  daily_usage_usd: number
  daily_limit_usd?: number
}

export interface InsightsSpend {
  total_usd: number
  free_sub_usd: number
  paid_sub_usd: number
  balance_usd: number
  paid_cny: number
  has_real_paying: boolean
}

export interface InsightsLinked {
  user: InsightsUserRef
  /** visitor | ip_prefix | api_ip | user_agent */
  via: string[]
}

export interface InsightsAttribution {
  source: string
  medium: string
  campaign: string
  aff_code: string
  referrer_host: string
  landing_path: string
}

export interface UserShadow {
  user_id: number
  reason: string
  /** auto (sign-up rules) | admin */
  source: string
  related_user_id?: number
  created_by?: number
  created_at: string
}

/** linked_signup | trial_day1 | relay_free | burst_signups | invitee_fingerprint */
export interface RiskEvent {
  id: number
  user_id: number
  user_email?: string
  username?: string
  rule: string
  detail: string
  created_at: string
}

export interface UserInsightsProfile {
  user: InsightsUserRef
  role: string
  signup_source: string
  balance: number
  attribution?: InsightsAttribution
  visitor_id: string
  signup_ip_prefix: string
  trial_granted: boolean
  web_prefixes: InsightsCount[]
  api_ips: InsightsCount[]
  clients: InsightsClientUA[]
  groups: InsightsCount[]
  models: InsightsCount[]
  spend: InsightsSpend
  payments: InsightsPayment[]
  subscriptions: InsightsSubscription[]
  invite_up: InsightsUserRef[]
  invite_down: InsightsUserRef[]
  linked: InsightsLinked[]
  shadow?: UserShadow
  events: RiskEvent[]
  risk_score: number
  risk_reasons: string[]
}

export async function getOverview(days: number): Promise<UserInsightsOverview> {
  const { data } = await apiClient.get('/admin/user-insights/overview', { params: { days } })
  return data
}

/** q: a user ID or an email */
export async function getProfile(q: string | number): Promise<UserInsightsProfile> {
  const { data } = await apiClient.get('/admin/user-insights/profile', { params: { q } })
  return data
}

export async function listEvents(params: { user_id?: number; page?: number; page_size?: number }): Promise<{ items: RiskEvent[]; total: number }> {
  const { data } = await apiClient.get('/admin/user-insights/events', { params })
  return data
}

export async function runMonitor(): Promise<{ new_events: RiskEvent[] }> {
  const { data } = await apiClient.post('/admin/user-insights/monitor/run')
  return data
}

/** Restrictions of these users, keyed by user ID. */
export async function getShadows(userIds: number[]): Promise<Record<string, UserShadow>> {
  if (userIds.length === 0) return {}
  const { data } = await apiClient.get('/admin/user-insights/shadows', { params: { user_ids: userIds.join(',') } })
  return data
}

export async function shadowUser(userId: number, reason: string): Promise<UserShadow> {
  const { data } = await apiClient.put(`/admin/user-insights/shadows/${userId}`, { reason })
  return data
}

export async function unshadowUser(userId: number): Promise<void> {
  await apiClient.delete(`/admin/user-insights/shadows/${userId}`)
}

export const userInsightsAPI = { getOverview, getProfile, listEvents, runMonitor, getShadows, shadowUser, unshadowUser }
export default userInsightsAPI
