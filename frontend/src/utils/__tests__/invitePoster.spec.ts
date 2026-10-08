import { describe, expect, it } from 'vitest'
import zhGrowth from '@/i18n/locales/zh/growth'
import enGrowth from '@/i18n/locales/en/growth'
import { canvasCopy, formatPercent, posterSiteName, inviteCopy, minimalCopy, posterInviteUrl, rewardExample, studentCopy, wrapText, type PosterData } from '../invitePoster'

const data: PosterData = {
  siteName: 'HiveGPT',
  siteHost: 'hivegpt.cn',
  logoUrl: '/logo.svg',
  inviteLink: 'https://hivegpt.cn/register?aff=ABC123',
  affCode: 'ABC123',
  rebateRate: 20,
  inviteeBonusRate: 10,
  inviteeBonusCap: 30,
  eduDiscount: 10
}

// Minimal translator over the zh messages, enough to check the rendered copy.
function t(key: string, params: Record<string, unknown> = {}): string {
  const value = key.split('.').reduce<unknown>((node, part) => (node as Record<string, unknown>)?.[part], zhGrowth)
  if (typeof value !== 'string') throw new Error(`missing ${key}`)
  return value.replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? ''))
}

describe('invite poster copy', () => {
  it('uses the live reward numbers', () => {
    const invite = inviteCopy(data, t)
    expect(invite.stats.map((s) => s.value)).toEqual(['10', '20'])
    expect(invite.stats[0].desc).toContain('最高 ¥30')
    expect(invite.example).toBe('举个例子：好友付费 ¥100 → 好友得 ¥10 余额，你得 ¥20 返利')
    expect(invite.footer.code).toBe('邀请码 ABC123')
    expect(studentCopy(data, t).items[0].title).toBe('教育邮箱认证，订阅立减 10%')
    expect(canvasCopy(data, t).footer.cta).toBe('扫码注册，首单返 10%')
    expect(minimalCopy(data, t).offer).toBe('扫码注册，首单返 10% 余额，最高 ¥30')
  })

  it('mentions the sign-up trial balance when there is one', () => {
    expect(inviteCopy(data, t).sub).toBe('扫码注册并付费后，奖励自动到账，余额可以直接买订阅。')
    expect(inviteCopy({ ...data, inviteeSignupBonus: 1 }, t).sub).toBe('扫码注册就送 $1 试用余额，付费后奖励自动到账。')
  })

  it('drops the friend reward when the invitee bonus is off', () => {
    const off = { ...data, inviteeBonusRate: 0, eduDiscount: 0 }
    expect(inviteCopy(off, t).stats.map((s) => s.label)).toEqual(['你'])
    expect(inviteCopy(off, t).example).toBe('举个例子：好友付费 ¥100 → 你得 ¥20 返利')
    expect(studentCopy(off, t).items.map((i) => i.title)).toEqual(['Codex 订阅：写代码、改论文', '无限画布：文生图、改图', '邀请同学，一起省'])
    expect(minimalCopy(off, t).offer).toBe('扫码注册，开始用 AI')
  })

  it('the worked example honours the bonus cap', () => {
    expect(rewardExample({ rebateRate: 12.5, inviteeBonusRate: 50, inviteeBonusCap: 30 })).toEqual({ amount: 100, bonus: '30', rebate: '12.5' })
    expect(rewardExample({ rebateRate: 20, inviteeBonusRate: 10, inviteeBonusCap: 0 })).toEqual({ amount: 100, bonus: '10', rebate: '20' })
    expect(formatPercent(20)).toBe('20')
    expect(formatPercent(12.504)).toBe('12.5')
  })

  it('poster copy avoids superlatives, fake original prices and relay wording (广告法)', () => {
    const banned = /最|第一|全网|首选|国家级|顶级|原价|中转|翻墙|best|number one|cheapest/i
    const walk = (node: unknown, path: string) => {
      if (typeof node === 'string') expect(banned.test(node), `${path}: ${node}`).toBe(false)
      else if (node && typeof node === 'object') for (const [k, v] of Object.entries(node)) walk(v, `${path}.${k}`)
    }
    // "最高" (a cap) is a factual limit, not a superlative claim.
    const strip = (o: unknown): unknown => JSON.parse(JSON.stringify(o).replace(/最高/g, '上限'))
    walk(strip(zhGrowth.affiliatePoster), 'zh')
    walk(strip(enGrowth.affiliatePoster), 'en')
  })
})

describe('invite poster helpers', () => {
  it('always brands posters as HiveGPT unless the site has its own name', () => {
    expect(posterSiteName('Sub2API')).toBe('HiveGPT')
    expect(posterSiteName(' sub2api ')).toBe('HiveGPT')
    expect(posterSiteName('')).toBe('HiveGPT')
    expect(posterSiteName(undefined)).toBe('HiveGPT')
    expect(posterSiteName('HiveGPT')).toBe('HiveGPT')
  })

  it('tags the QR link with the poster design', () => {
    expect(posterInviteUrl('https://hivegpt.cn/register?aff=ABC123', 'student')).toBe('https://hivegpt.cn/register?aff=ABC123&utm_source=poster&utm_medium=student')
    expect(posterInviteUrl('', 'invite')).toBe('')
  })

  it('wraps CJK anywhere, keeps Latin words and punctuation together', () => {
    const measure = (s: string) => s.length
    expect(wrapText('扫码注册并付费后，奖励自动到账。', 8, measure)).toEqual(['扫码注册并付费后，', '奖励自动到账。'])
    expect(wrapText('use Codex to write code', 10, measure)).toEqual(['use Codex', 'to write', 'code'])
    expect(wrapText('一行\n两行', 10, measure)).toEqual(['一行', '两行'])
  })
})
