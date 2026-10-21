import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import { formatTokenAmount } from '../format'

describe('formatTokenAmount', () => {
  afterEach(() => {
    i18n.global.locale.value = 'en'
  })

  it('uses K/M/B outside Chinese', () => {
    i18n.global.locale.value = 'en'
    expect(formatTokenAmount(999)).toBe('999')
    expect(formatTokenAmount(93_640)).toBe('93.64K')
    expect(formatTokenAmount(104_970_000)).toBe('104.97M')
    expect(formatTokenAmount(2_500_000_000)).toBe('2.50B')
    expect(formatTokenAmount(717_460, 1)).toBe('717.5K')
  })

  it('uses 万/亿 in Chinese', () => {
    i18n.global.locale.value = 'zh'
    expect(formatTokenAmount(9_999)).toBe('9,999')
    expect(formatTokenAmount(93_640)).toBe('9.36万')
    expect(formatTokenAmount(717_460)).toBe('71.75万')
    expect(formatTokenAmount(104_970_000)).toBe('1.05亿')
    expect(formatTokenAmount(104_970_000, 1)).toBe('1.0亿')
  })
})
