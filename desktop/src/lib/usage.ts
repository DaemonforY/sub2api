// Turns the gateway's GET /v1/usage answer (fetched by src-tauri `usage_summary`) into what the
// header shows: wallet balance or subscription allowance, plus today's spend.

import { invoke } from '@tauri-apps/api/core'

export interface UsageView {
  /** 钱包余额 / the subscription group's name */
  plan: string
  subscription: boolean
  /** Balance (wallet) or what is left of the tightest subscription limit; null when unknown. */
  remaining: number | null
  /** Spent today through this computer's key. */
  today: number | null
  /** Spent so far through this key (used to work out what one turn cost). */
  total: number | null
  /** Low enough that the next task may stop half way. */
  low: boolean
}

/** Below this many dollars the header turns red and offers 充值. */
export const LOW_BALANCE_USD = 1

const num = (v: unknown): number | null => (typeof v === 'number' && Number.isFinite(v) ? v : null)

export function toUsageView(raw: any): UsageView | null {
  if (!raw || raw.isValid === false) return null
  const subscription = Boolean(raw.subscription) || (raw.planName && raw.planName !== '钱包余额' && raw.balance === undefined)
  // A subscription with no daily/weekly/monthly limit reports -1.
  const r = num(raw.remaining)
  const remaining = r !== null && r < 0 ? null : r
  return {
    plan: String(raw.planName ?? ''),
    subscription,
    remaining,
    today: num(raw.usage?.today?.actual_cost),
    total: num(raw.usage?.total?.actual_cost),
    low: remaining !== null && remaining < LOW_BALANCE_USD
  }
}

export async function fetchUsage(): Promise<UsageView | null> {
  const raw = await invoke<any>('usage_summary').catch(() => null)
  return toUsageView(raw)
}

export function money(v: number | null): string {
  if (v === null) return '—'
  if (Math.abs(v) < 0.01 && v !== 0) return `$${v.toFixed(3)}`
  return `$${v.toFixed(2)}`
}
