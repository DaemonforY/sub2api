/** Admin API for the usage dashboard (埋点数据看板). */
import { apiClient } from '../client'

export interface AnalyticsTotals {
  visitors: number
  active_users: number
  wau: number
  signups: number
  activated: number
  paid_users: number
  revenue: number
}

export interface AnalyticsDay {
  day: string
  visitors: number
  page_views: number
  active_users: number
  api_users: number
  signups: number
  activated: number
  orders: number
  revenue: number
}

export interface AnalyticsChannel {
  source: string
  visitors: number
  signups: number
  activated: number
  paid_users: number
  revenue: number
}

export interface AnalyticsOverview {
  since: string
  days: number
  totals: AnalyticsTotals
  daily: AnalyticsDay[]
  funnel: { step: string; count: number }[]
  channels: AnalyticsChannel[] | null
  pages: { app: string; path: string; views: number; visitors: number }[] | null
  features: { event: string; count: number; visitors: number; users: number }[] | null
  retention: { week: string; signups: number; day1: number; day2_7: number; day8_30: number; activated: number }[] | null
  devices: Record<string, number>
  signup_errors?: AnalyticsBreakdown[] | null
  cta_clicks?: AnalyticsBreakdown[] | null
}

export interface AnalyticsBreakdown {
  key: string
  count: number
  visitors: number
}

export async function getOverview(days: number): Promise<AnalyticsOverview> {
  const { data } = await apiClient.get('/admin/analytics/overview', { params: { days } })
  return data
}

export interface ActivationReminderStatus {
  enabled: boolean
  sent: number
  /** Would go out on the next hourly run. */
  due: number
  subject: string
  /** The email's HTML. */
  preview: string
  /** The 公众号 客服消息 WeChat-only sign-ups get ("" when the 公众号 isn't wired). */
  wechat_preview?: string
}

export async function getReminder(): Promise<ActivationReminderStatus> {
  const { data } = await apiClient.get('/admin/analytics/reminder')
  return data
}

export async function setReminder(enabled: boolean): Promise<ActivationReminderStatus> {
  const { data } = await apiClient.put('/admin/analytics/reminder', { enabled })
  return data
}

/** 渠道链接: hivegpt.cn/go/<code> → target_path tagged with utm_source / utm_medium / utm_campaign=<code>. */
export interface ChannelLink {
  id: number
  code: string
  name: string
  source: string
  medium: string
  target_path: string
  aff_code: string
  promo_code: string
  note: string
  /** All time. */
  clicks: number
  created_at: string
  updated_at: string
  visitors: number
  signups: number
  /** Made an API call. */
  activated: number
  paid_users: number
  /** CNY received in the period, payment fee excluded. */
  revenue: number
  /** Usage at standard prices in the period. */
  usage_usd: number
  /** CNY value of the rebates accrued on their orders in the period. */
  rebates: number
  /** usage_usd × the 套餐毛利 cost rate; 0 when the rate isn't set. */
  cost_cny: number
  /** revenue − cost_cny − rebates. */
  profit: number
}

export interface ChannelLinkReport {
  days: number
  /** 0 = cost rate not set on 套餐毛利. */
  cost_per_usd: number
  links: ChannelLink[]
}

export type ChannelLinkInput = Pick<ChannelLink, 'name' | 'source' | 'medium' | 'target_path' | 'aff_code' | 'promo_code' | 'note'> & { code?: string }

/** days: 7 / 30 / 90, or 0 for all time. */
export async function listChannelLinks(days = 0): Promise<ChannelLinkReport> {
  const { data } = await apiClient.get('/admin/analytics/channel-links', { params: { days } })
  return data
}

export async function createChannelLink(input: ChannelLinkInput): Promise<ChannelLink> {
  const { data } = await apiClient.post('/admin/analytics/channel-links', input)
  return data
}

export async function updateChannelLink(id: number, input: ChannelLinkInput): Promise<ChannelLink> {
  const { data } = await apiClient.put(`/admin/analytics/channel-links/${id}`, input)
  return data
}

export async function deleteChannelLink(id: number): Promise<void> {
  await apiClient.delete(`/admin/analytics/channel-links/${id}`)
}

/** 套餐毛利: usage valued at standard prices × cost_per_usd (CNY per $1) against payments (CNY). */
export interface MarginReport {
  cost_per_usd: number
  configured: boolean
  days: number
  since: string
  totals: {
    revenue: number
    recharge_revenue: number
    subscription_revenue: number
    other_revenue: number
    paygo_billed: number
    paygo_cost: number
    subscription_cost: number
    admin_cost: number
    member_cost: number
    margin: number
  }
  groups: {
    group_id: number
    name: string
    subscription: boolean
    users: number
    usage_usd: number
    billed_usd: number
    cost: number
    margin?: number
  }[]
  plans: {
    id: number
    name: string
    group_name: string
    price: number
    days: number
    cap_usd: number | null
    max_cost: number | null
    max_loss: number
    break_even_usd: number
    sold: number
    revenue: number
  }[]
  subscriptions: MarginSubscription[]
  users: { user_id: number; email: string; usage_usd: number; billed_usd: number; paid: number; cost: number; margin: number }[]
}

export interface MarginSubscription {
  id: number
  user_id: number
  email: string
  group_name: string
  starts_at: string
  expires_at: string
  paid: number
  usage_usd: number
  active: boolean
  days_elapsed: number
  days_total: number
  cost: number
  margin: number
  projected_usd: number
  projected_cost: number
  projected_margin: number
  status: 'ok' | 'risk' | 'loss' | 'gift'
}

export async function getMargin(days: number): Promise<MarginReport> {
  const { data } = await apiClient.get('/admin/analytics/margin', { params: { days } })
  return data
}

export async function setMarginCost(costPerUSD: number, days: number): Promise<MarginReport> {
  const { data } = await apiClient.put('/admin/analytics/margin/cost', { cost_per_usd: costPerUSD }, { params: { days } })
  return data
}

export default { getOverview, getReminder, setReminder, getMargin, setMarginCost, listChannelLinks, createChannelLink, updateChannelLink, deleteChannelLink }
