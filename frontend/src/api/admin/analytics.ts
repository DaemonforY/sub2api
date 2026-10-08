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
}

export async function getOverview(days: number): Promise<AnalyticsOverview> {
  const { data } = await apiClient.get('/admin/analytics/overview', { params: { days } })
  return data
}

export default { getOverview }
