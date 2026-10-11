/** Public pricing (no login): plans on sale, pay-as-you-go groups, recharge ratio. */
import { apiClient } from './client'

export interface PricingPlan {
  id: number
  name: string
  description: string
  price: number
  original_price?: number
  currency: string
  validity_days: number
  validity_unit: string
  features: string
  group_name: string
  platform: string
  daily_limit_usd: number | null
  weekly_limit_usd: number | null
  monthly_limit_usd: number | null
  recommended?: boolean
}

export interface PricingGroup {
  name: string
  platform: string
  description: string
  rate_multiplier: number
}

export interface PublicPricing {
  payment_enabled: boolean
  /** Balance (USD of usage) per 1 CNY paid. */
  recharge_multiplier: number
  min_recharge: number
  plans: PricingPlan[]
  pay_as_you_go: PricingGroup[]
  avg_request_cost_usd: number
  avg_sample_requests: number
  /** Requests members have made in total (admins excluded). */
  total_requests?: number
  /** Cost of one image / one other request on the starter key's group now (USD); 0 = not enough data. */
  starter_image_cost_usd?: number
  starter_request_cost_usd?: number
}

export async function getPublicPricing(): Promise<PublicPricing> {
  const { data } = await apiClient.get<PublicPricing>('/pricing')
  return data
}

/** Plan length in days (validity_unit may be days / weeks / months). */
export function planDays(p: Pick<PricingPlan, 'validity_days' | 'validity_unit'>): number {
  // Same as billing (psComputeValidityDays): week(s) ×7, month(s) ×30, anything else days.
  const n = p.validity_days || 0
  const unit = String(p.validity_unit || 'day').trim().toLowerCase().replace(/s$/, '')
  if (unit === 'week') return n * 7
  if (unit === 'month') return n * 30
  return n
}

/**
 * Most usage (USD, standard prices) a plan allows over its own length: the tightest of its daily,
 * weekly and monthly limits (a 7-day plan gets at most one week's allowance), or null without limits.
 */
export function planPeriodCapUSD(p: PricingPlan): number | null {
  const days = Math.max(1, planDays(p))
  const caps: number[] = []
  if (p.daily_limit_usd) caps.push(p.daily_limit_usd * days)
  if (p.weekly_limit_usd) caps.push(p.weekly_limit_usd * Math.max(1, days / 7))
  if (p.monthly_limit_usd) caps.push(p.monthly_limit_usd * Math.max(1, days / 30))
  return caps.length ? Math.min(...caps) : null
}

/** Price normalised to 30 days. */
export function planMonthlyPrice(p: PricingPlan): number {
  const days = planDays(p)
  return days > 0 ? (p.price / days) * 30 : p.price
}

// Only worth saying once the number is big enough to reassure (members' requests, admins excluded).
const TOTAL_REQUESTS_MIN = 10000

/** 「1.4 万」 in Chinese, 14,220 otherwise; '' below 10,000. */
export function formatTotalRequests(n: number | undefined, locale: string): string {
  if (!n || n < TOTAL_REQUESTS_MIN) return ''
  if (!locale.startsWith('zh')) return n.toLocaleString('en-US')
  return n >= 100000 ? `${Math.floor(n / 10000)} 万` : `${(Math.floor(n / 1000) / 10).toFixed(1)} 万`
}

/**
 * What the sign-up credit buys on the starter key: whole images and requests (0 when unknown or
 * under one). Requests above 20 are rounded down to tens so the page never promises too much.
 */
export function signupCreditEstimate(bonusUSD: number, p: Pick<PublicPricing, 'starter_image_cost_usd' | 'starter_request_cost_usd'> | null): { images: number; chats: number } {
  const per = (cost?: number) => (bonusUSD > 0 && cost && cost > 0 ? Math.floor(bonusUSD / cost) : 0)
  const chats = per(p?.starter_request_cost_usd)
  return { images: per(p?.starter_image_cost_usd), chats: chats > 20 ? Math.floor(chats / 10) * 10 : chats }
}
