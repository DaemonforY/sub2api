import { describe, expect, it } from 'vitest'
import { formatTotalRequests, signupCreditEstimate } from '../pricing'

describe('signupCreditEstimate', () => {
  it('turns the sign-up credit into whole images and requests', () => {
    expect(signupCreditEstimate(2, { starter_image_cost_usd: 0.6, starter_request_cost_usd: 0.11 })).toEqual({ images: 3, chats: 18 })
  })

  it('rounds big request counts down to tens', () => {
    expect(signupCreditEstimate(2, { starter_image_cost_usd: 0.6, starter_request_cost_usd: 0.03 })).toEqual({ images: 3, chats: 60 })
  })

  it('says nothing without credit or data', () => {
    expect(signupCreditEstimate(0, { starter_image_cost_usd: 0.6, starter_request_cost_usd: 0.1 })).toEqual({ images: 0, chats: 0 })
    expect(signupCreditEstimate(2, { starter_image_cost_usd: 0, starter_request_cost_usd: undefined })).toEqual({ images: 0, chats: 0 })
    expect(signupCreditEstimate(2, null)).toEqual({ images: 0, chats: 0 })
  })
})

describe('formatTotalRequests', () => {
  it('only reassures from 10,000 up', () => {
    expect(formatTotalRequests(9999, 'zh')).toBe('')
    expect(formatTotalRequests(14220, 'zh')).toBe('1.4 万')
    expect(formatTotalRequests(123456, 'zh')).toBe('12 万')
    expect(formatTotalRequests(14220, 'en')).toBe('14,220')
  })
})
