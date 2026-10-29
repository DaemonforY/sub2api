// Contest entries: their own page, the 拉票 link and text, the vote a visitor asked for before
// signing in, and the per-visitor random order of the entry list.

const VOTE_INTENT_KEY = 'contest_vote_intent'
const VOTE_INTENT_TTL_MS = 30 * 60 * 1000
const LIST_SEED_KEY = 'contest_list_seed'

export function contestEntryPath(contestId: number, entryId: number): string {
  return `/contests/${contestId}/e/${entryId}`
}

/** The link people share: the entry page, the sharer's invite code, and contest tracking. */
export function contestEntryShareUrl(origin: string, contestId: number, entryId: number, opts: { aff?: string; medium?: string } = {}): string {
  const url = new URL(contestEntryPath(contestId, entryId), origin)
  if (opts.aff) url.searchParams.set('aff', opts.aff)
  url.searchParams.set('utm_source', 'contest_share')
  url.searchParams.set('utm_medium', opts.medium || 'link')
  url.searchParams.set('utm_campaign', `contest-${contestId}`)
  return url.toString()
}

const PITCHES = [
  (title: string, contest: string, url: string) => `我用 AI 画的「${title}」在参加${contest}，帮我点个赞吧 👍 打开链接一键点赞：${url}`,
  (title: string, contest: string, url: string) => `求赞！我的 AI 作品「${title}」正在${contest}里拉票，点开就能看到大图，顺手点个赞 🙏 ${url}`,
  (title: string, contest: string, url: string) => `${contest}进行中，我的作品「${title}」需要你的一个赞 ❤️ 注册后就能点，拜托啦：${url}`
]

/**
 * A ready-to-paste message asking friends to vote. The author asks for themselves; anyone else
 * recommends the entry. `pick` chooses the author's template (random by default).
 */
export function contestPitch(
  entry: { title: string; author: string; mine: boolean },
  contestTitle: string,
  url: string,
  pick = Math.floor(Math.random() * PITCHES.length)
): string {
  if (!entry.mine) return `「${entry.title}」正在参加${contestTitle}，作者是 ${entry.author}。觉得好看就帮 TA 点个赞吧 👍 ${url}`
  const fn = PITCHES[Math.abs(pick) % PITCHES.length]
  return fn(entry.title, contestTitle, url)
}

export interface ContestVoteIntent {
  contestId: number
  entryId: number
  at: number
}

function storage(kind: 'local' | 'session'): Storage | null {
  try {
    return kind === 'local' ? window.localStorage : window.sessionStorage
  } catch {
    return null
  }
}

/**
 * Remembers the vote a signed-out visitor asked for, so it can be cast once they are back from
 * sign-in (WeChat authorisation or a QR scan can take a few minutes; localStorage survives both).
 */
export function saveVoteIntent(contestId: number, entryId: number, now = Date.now()): void {
  storage('local')?.setItem(VOTE_INTENT_KEY, JSON.stringify({ contestId, entryId, at: now }))
}

/** Returns and clears the pending vote for this contest; a stale or other-contest intent is dropped. */
export function takeVoteIntent(contestId: number, now = Date.now()): ContestVoteIntent | null {
  const store = storage('local')
  const raw = store?.getItem(VOTE_INTENT_KEY)
  if (!store || !raw) return null
  let intent: ContestVoteIntent | null = null
  try {
    intent = JSON.parse(raw) as ContestVoteIntent
  } catch {
    intent = null
  }
  if (!intent || typeof intent.entryId !== 'number' || typeof intent.at !== 'number' || now - intent.at > VOTE_INTENT_TTL_MS) {
    store.removeItem(VOTE_INTENT_KEY)
    return null
  }
  if (intent.contestId !== contestId) return null
  store.removeItem(VOTE_INTENT_KEY)
  return intent
}

/** A random seed kept for the browser tab, so 「随机」 pages don't repeat or skip entries. */
export function contestListSeed(): string {
  const store = storage('session')
  let seed = store?.getItem(LIST_SEED_KEY) || ''
  if (!seed) {
    seed = Math.random().toString(36).slice(2, 12)
    store?.setItem(LIST_SEED_KEY, seed)
  }
  return seed
}
