// Remembers ?promo= from any landing page (a blogger's channel link) until registration, the way
// invite codes are kept: localStorage, 30 days. The learning site writes the same key.
const PROMO_REFERRAL_CODE_KEY = 'promo_referral_code'
const PROMO_REFERRAL_TTL_MS = 30 * 24 * 60 * 60 * 1000
const PROMO_CODE_PATTERN = /^[A-Za-z0-9_-]{1,32}$/

export function normalizePromoCode(value?: unknown): string {
  const raw = Array.isArray(value) ? value[0] : value
  const code = typeof raw === 'string' ? raw.trim() : ''
  return PROMO_CODE_PATTERN.test(code) ? code.toUpperCase() : ''
}

export function storePromoCode(value?: unknown, now = Date.now()): void {
  const code = normalizePromoCode(value)
  if (!code || typeof window === 'undefined') return
  try {
    window.localStorage.setItem(PROMO_REFERRAL_CODE_KEY, JSON.stringify({ code, expiresAt: now + PROMO_REFERRAL_TTL_MS }))
  } catch {
    // storage unavailable
  }
}

export function loadPromoCode(now = Date.now()): string {
  if (typeof window === 'undefined') return ''
  try {
    const raw = window.localStorage.getItem(PROMO_REFERRAL_CODE_KEY)
    if (!raw) return ''
    const parsed = JSON.parse(raw) as { code?: unknown; expiresAt?: unknown }
    const code = normalizePromoCode(parsed.code)
    if (!code || (Number(parsed.expiresAt) || 0) <= now) {
      window.localStorage.removeItem(PROMO_REFERRAL_CODE_KEY)
      return ''
    }
    return code
  } catch {
    return ''
  }
}

/** The promo code from the URL (remembered for later), else the remembered one. */
export function resolvePromoCode(value?: unknown): string {
  const code = normalizePromoCode(value)
  if (code) {
    storePromoCode(code)
    return code
  }
  return loadPromoCode()
}
