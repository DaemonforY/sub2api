import type { Contest, ContestPhase, ContestPrize } from '@/api/contests'
import { sanitizeUrl } from '@/utils/url'

type Translate = (key: string, params?: Record<string, unknown>) => string

const BADGE_BASE = 'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold shadow-sm '

/** Tailwind classes for a contest phase badge. */
export function phaseBadgeClass(phase: ContestPhase): string {
  switch (phase) {
    case 'submitting':
    case 'submitting_voting':
    case 'voting':
      return BADGE_BASE + 'bg-emerald-500 text-white'
    case 'upcoming':
    case 'waiting_vote':
      return BADGE_BASE + 'bg-sky-500 text-white'
    case 'tallying':
      return BADGE_BASE + 'bg-amber-500 text-white'
    case 'cancelled':
      return BADGE_BASE + 'bg-gray-400 text-white'
    default:
      return BADGE_BASE + 'bg-gray-700 text-white'
  }
}

/** Human label for the places covered by a prize tier. */
export function formatPrizePlaces(p: ContestPrize, t: Translate): string {
  return p.rank_from === p.rank_to
    ? t('contests.prizes.place', { n: p.rank_from })
    : t('contests.prizes.places', { from: p.rank_from, to: p.rank_to })
}

/** Human label for what a prize tier awards. */
export function formatPrizeReward(p: { type: string; amount: number; label: string }, t: Translate): string {
  if (p.type === 'balance') {
    const reward = t('contests.prizes.balance', { amount: Number(p.amount).toFixed(2).replace(/\.00$/, '') })
    return p.label ? `${p.label} · ${reward}` : reward
  }
  return p.label
}

export function formatContestPrize(p: ContestPrize, t: Translate): string {
  return `${formatPrizePlaces(p, t)} ${formatPrizeReward(p, t)}`
}

export function formatDateTime(value?: string): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** Cover of contests made by the monthly schedule when the template sets none. */
export const MONTHLY_CONTEST_COVER = '/contest-media/monthly-draw.webp'

/** The contest's cover: its own, else the monthly cover for scheduled contests, else ''. */
export function contestCover(c: Pick<Contest, 'cover_image' | 'auto_period'>): string {
  return safeCover(c.cover_image) || (c.auto_period ? MONTHLY_CONTEST_COVER : '')
}

/** Only allow https or site-relative cover images. */
export function safeCover(url?: string): string {
  if (!url) return ''
  return sanitizeUrl(url, { allowRelative: true }) || ''
}

export interface ContestCountdown {
  labelKey: string
  target: string
}

/** Next milestone the viewer is waiting for, or null when none applies. */
export function contestCountdown(c: Contest): ContestCountdown | null {
  switch (c.phase) {
    case 'upcoming':
      return { labelKey: 'contests.countdown.starts', target: c.submission_start_at }
    case 'submitting':
      return { labelKey: 'contests.countdown.submissionEnds', target: c.submission_end_at }
    case 'waiting_vote':
      return { labelKey: 'contests.countdown.votingStarts', target: c.voting_start_at }
    case 'submitting_voting':
    case 'voting':
      return { labelKey: 'contests.countdown.votingEnds', target: c.voting_end_at }
    default:
      return null
  }
}

/** Formats a remaining duration like "2 天 3 小时" / "15 分钟". */
export function formatRemaining(ms: number, t: Translate): string {
  if (ms <= 0) return t('contests.schedule.minutes', { n: 0 })
  const minutes = Math.floor(ms / 60000)
  const days = Math.floor(minutes / 1440)
  const hours = Math.floor((minutes % 1440) / 60)
  const mins = minutes % 60
  const parts: string[] = []
  if (days) parts.push(t('contests.schedule.days', { n: days }))
  if (hours) parts.push(t('contests.schedule.hours', { n: hours }))
  if (!days) parts.push(t('contests.schedule.minutes', { n: mins }))
  return parts.join(' ')
}

export interface ContestMilestone {
  key: 'open' | 'submissionOpen' | 'votingOpen' | 'submissionClose' | 'votingClose'
  at: string
  done: boolean
}

/** The contest's dates in order, merging submission and voting opening when they coincide. */
export function contestMilestones(c: Contest, now = Date.now()): ContestMilestone[] {
  const same = new Date(c.submission_start_at).getTime() === new Date(c.voting_start_at).getTime()
  const list: Omit<ContestMilestone, 'done'>[] = same
    ? [{ key: 'open', at: c.submission_start_at }]
    : [
        { key: 'submissionOpen', at: c.submission_start_at },
        { key: 'votingOpen', at: c.voting_start_at }
      ]
  list.push({ key: 'submissionClose', at: c.submission_end_at }, { key: 'votingClose', at: c.voting_end_at })
  return list
    .sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())
    .map((m) => ({ ...m, done: new Date(m.at).getTime() <= now }))
}

/** Remaining time split for a countdown display; all zero once passed. */
export function countdownParts(ms: number): { days: number; hours: number; minutes: number } {
  const total = Math.max(0, Math.floor(ms / 60000))
  return { days: Math.floor(total / 1440), hours: Math.floor((total % 1440) / 60), minutes: total % 60 }
}
