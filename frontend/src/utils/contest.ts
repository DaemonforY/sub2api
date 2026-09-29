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
