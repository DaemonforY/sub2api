import QRCode from 'qrcode'

/**
 * Invite posters drawn on a <canvas> in the browser (affiliate page). Every poster carries the
 * user's own invite QR code and code; the reward numbers come from the live settings so a poster
 * never promises more than the program gives.
 *
 * Copy rules (广告法 / 价格法): no superlatives, no struck-through "original" prices, no national
 * symbols, no "relay/中转" wording.
 */

export const POSTER_WIDTH = 1080
export const POSTER_HEIGHT = 1440

export type PosterTemplateId = 'invite' | 'student' | 'canvas' | 'minimal'
export const POSTER_TEMPLATES: PosterTemplateId[] = ['invite', 'student', 'canvas', 'minimal']

export interface PosterData {
  siteName: string
  siteHost: string
  logoUrl: string
  inviteLink: string
  affCode: string
  /** Inviter rebate on every payment of the invitee, percent. */
  rebateRate: number
  /** Invitee first-order bonus, percent (0 = off). */
  inviteeBonusRate: number
  /** Invitee bonus cap in yuan (0 = no cap). */
  inviteeBonusCap: number
  /** Education verification subscription discount, percent (0 = off). */
  eduDiscount: number
}

type Translate = (key: string, params?: Record<string, unknown>) => string

/** Posters always carry the brand: an unset site name or the upstream default "Sub2API" becomes HiveGPT. */
export function posterSiteName(name?: string | null): string {
  const value = (name || '').trim()
  return value && !/^sub2api$/i.test(value) ? value : 'HiveGPT'
}

/** Trims 20.00 → "20", 12.50 → "12.5". */
export function formatPercent(value: number): string {
  return String(Math.round((Number(value) || 0) * 100) / 100)
}

/** Worked example on a ¥100 payment, honouring the bonus cap. */
export function rewardExample(data: Pick<PosterData, 'rebateRate' | 'inviteeBonusRate' | 'inviteeBonusCap'>) {
  const amount = 100
  let bonus = (amount * data.inviteeBonusRate) / 100
  if (data.inviteeBonusCap > 0) bonus = Math.min(bonus, data.inviteeBonusCap)
  return { amount, bonus: trimMoney(bonus), rebate: trimMoney((amount * data.rebateRate) / 100) }
}

function trimMoney(value: number): string {
  const rounded = Math.round(value * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(2).replace(/0$/, '')
}

/** The QR target: the invite link tagged with the poster design, for channel stats. */
export function posterInviteUrl(inviteLink: string, template: PosterTemplateId): string {
  if (!inviteLink) return ''
  try {
    const url = new URL(inviteLink)
    url.searchParams.set('utm_source', 'poster')
    url.searchParams.set('utm_medium', template)
    return url.toString()
  } catch {
    return inviteLink
  }
}

/**
 * Greedy line breaking for CJK / Latin mixed text: breaks anywhere between CJK characters and at
 * spaces for Latin words; never starts a line with closing punctuation.
 */
export function wrapText(text: string, maxWidth: number, measure: (s: string) => number): string[] {
  const lines: string[] = []
  for (const paragraph of text.split('\n')) {
    const tokens = paragraph.match(/[A-Za-z0-9¥$%.,:/@#&+\-_'"!?()[\]]+|\s+|./gu) || []
    let line = ''
    for (const token of tokens) {
      const next = line + token
      if (!line || measure(next) <= maxWidth || /^[，。、；：！？）」』》,.;:!?)]$/u.test(token)) {
        line = next
        continue
      }
      lines.push(line.trimEnd())
      line = token.trimStart()
    }
    lines.push(line.trimEnd())
  }
  return lines
}

function copyContext(data: PosterData, t: Translate) {
  const hasBonus = data.inviteeBonusRate > 0
  return {
    rebate: formatPercent(data.rebateRate),
    bonus: formatPercent(data.inviteeBonusRate),
    hasBonus,
    capText: hasBonus && data.inviteeBonusCap > 0 ? t('affiliatePoster.capSuffix', { cap: trimMoney(data.inviteeBonusCap) }) : '',
    footer: {
      cta: t('affiliatePoster.footer.cta'),
      code: t('affiliatePoster.footer.code', { code: data.affCode }),
      fine: t(hasBonus ? 'affiliatePoster.footer.fineBonus' : 'affiliatePoster.footer.fine', { site: data.siteHost }),
    },
  }
}

export type PosterFooter = { cta: string; code: string; fine: string }

/** Copy of the "both of you get rewards" poster. */
export function inviteCopy(data: PosterData, t: Translate) {
  const c = copyContext(data, t)
  return {
    tag: t('affiliatePoster.invite.tag'),
    headline: [t('affiliatePoster.invite.line1'), t('affiliatePoster.invite.line2')],
    sub: t('affiliatePoster.invite.sub'),
    stats: [
      ...(c.hasBonus ? [{ label: t('affiliatePoster.invite.friend'), value: c.bonus, desc: t('affiliatePoster.invite.friendDesc', { cap: c.capText }) }] : []),
      { label: t('affiliatePoster.invite.you'), value: c.rebate, desc: t('affiliatePoster.invite.youDesc') },
    ],
    example: t(c.hasBonus ? 'affiliatePoster.invite.example' : 'affiliatePoster.invite.exampleNoBonus', rewardExample(data)),
    footer: c.footer as PosterFooter,
  }
}

/** Copy of the students / teachers poster. */
export function studentCopy(data: PosterData, t: Translate) {
  const c = copyContext(data, t)
  return {
    tag: t('affiliatePoster.student.tag'),
    headline: [t('affiliatePoster.student.line1'), t('affiliatePoster.student.line2')],
    accent: t('affiliatePoster.student.accent'),
    sub: t('affiliatePoster.student.sub'),
    items: [
      data.eduDiscount > 0
        ? { title: t('affiliatePoster.student.eduTitle', { rate: formatPercent(data.eduDiscount) }), desc: t('affiliatePoster.student.eduDesc') }
        : { title: t('affiliatePoster.student.codexTitle'), desc: t('affiliatePoster.student.codexDesc') },
      { title: t('affiliatePoster.student.canvasTitle'), desc: t('affiliatePoster.student.canvasDesc') },
      c.hasBonus
        ? { title: t('affiliatePoster.student.bonusTitle', { rate: c.bonus }), desc: t('affiliatePoster.student.bonusDesc', { cap: c.capText }) }
        : { title: t('affiliatePoster.student.rebateTitle'), desc: t('affiliatePoster.student.rebateDesc', { rate: c.rebate }) },
    ],
    footer: c.footer as PosterFooter,
  }
}

/** Copy of the AI drawing (canvas) poster. */
export function canvasCopy(data: PosterData, t: Translate) {
  const c = copyContext(data, t)
  return {
    tag: t('affiliatePoster.canvas.tag'),
    headline: [t('affiliatePoster.canvas.line1'), t('affiliatePoster.canvas.line2')],
    sub: t('affiliatePoster.canvas.sub'),
    prompt: t('affiliatePoster.canvas.prompt'),
    result: t('affiliatePoster.canvas.result'),
    chips: [t('affiliatePoster.canvas.chip1'), t('affiliatePoster.canvas.chip2'), t('affiliatePoster.canvas.chip3'), t('affiliatePoster.canvas.chip4')],
    footer: { ...c.footer, cta: c.hasBonus ? t('affiliatePoster.canvas.ctaBonus', { rate: c.bonus }) : c.footer.cta } as PosterFooter,
  }
}

/** Copy of the minimal card poster. */
export function minimalCopy(data: PosterData, t: Translate) {
  const c = copyContext(data, t)
  return {
    tagline: t('affiliatePoster.minimal.tagline'),
    invite: t('affiliatePoster.minimal.invite', { site: data.siteName }),
    offer: c.hasBonus ? t('affiliatePoster.minimal.offer', { rate: c.bonus, cap: c.capText }) : t('affiliatePoster.minimal.offerNoBonus'),
    footer: c.footer as PosterFooter,
  }
}

export async function renderQr(url: string, size: number, dark = '#0a1a39'): Promise<HTMLCanvasElement> {
  const canvas = document.createElement('canvas')
  await QRCode.toCanvas(canvas, url, { width: size, margin: 2, errorCorrectionLevel: 'M', color: { dark, light: '#ffffff' } })
  return canvas
}
