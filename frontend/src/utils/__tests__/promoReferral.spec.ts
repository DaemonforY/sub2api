import { beforeEach, describe, expect, it } from 'vitest'
import { loadPromoCode, normalizePromoCode, resolvePromoCode, storePromoCode } from '@/utils/promoReferral'

describe('promoReferral', () => {
  beforeEach(() => window.localStorage.clear())

  it('normalizes and rejects junk', () => {
    expect(normalizePromoCode(' kol10 ')).toBe('KOL10')
    expect(normalizePromoCode(['abc'])).toBe('ABC')
    expect(normalizePromoCode('<script>')).toBe('')
    expect(normalizePromoCode(undefined)).toBe('')
  })

  it('remembers a URL code for 30 days', () => {
    const now = 1_000_000
    storePromoCode('kol10', now)
    expect(loadPromoCode(now + 1000)).toBe('KOL10')
    expect(loadPromoCode(now + 31 * 24 * 3600 * 1000)).toBe('')
    expect(window.localStorage.getItem('promo_referral_code')).toBeNull()
  })

  it('prefers the URL, falls back to the stored code', () => {
    expect(resolvePromoCode(undefined)).toBe('')
    expect(resolvePromoCode('first')).toBe('FIRST')
    expect(resolvePromoCode(undefined)).toBe('FIRST')
    expect(resolvePromoCode('second')).toBe('SECOND')
  })
})
