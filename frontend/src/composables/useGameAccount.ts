// 小游戏可选登录: nobody has to sign in to play. A signed-in player's result goes onto the site-wide
// leaderboard and (for games that keep progress) the save follows them across devices. A visitor who
// signs in from the result screen comes back with that run's score still waiting: it is kept in
// sessionStorage and submitted on return.
import { computed, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { getGameSave, putGameSave, submitGameScore, type GameScoreResult } from '@/api/gameProgress'
import { track } from '@/utils/analytics'

const PENDING_TTL_MS = 30 * 60 * 1000

interface Pending {
  score: number
  detail: Record<string, unknown>
  at: number
}

export function useGameAccount(game: string) {
  const auth = useAuthStore()
  const app = useAppStore()
  const loggedIn = computed(() => auth.isAuthenticated)
  const lastResult = ref<GameScoreResult | null>(null)
  const submitting = ref(false)
  const pendingKey = `hg_game_pending_${game}`
  const signupBonus = computed(() => Number(app.cachedPublicSettings?.signup_bonus || 0))

  function loginHref(): string {
    return `/login?redirect=${encodeURIComponent(location.pathname)}`
  }

  /** Sends a finished run's score; for a visitor it is parked until they sign in. */
  async function submit(score: number, detail: Record<string, unknown> = {}): Promise<GameScoreResult | null> {
    if (score <= 0) return null
    if (!loggedIn.value) {
      try {
        sessionStorage.setItem(pendingKey, JSON.stringify({ score, detail, at: Date.now() } satisfies Pending))
      } catch {
        /* private mode */
      }
      return null
    }
    submitting.value = true
    try {
      lastResult.value = await submitGameScore(game, score, detail)
      return lastResult.value
    } catch {
      return null
    } finally {
      submitting.value = false
    }
  }

  /** After signing in from a result screen: submit the run that was waiting. */
  async function flushPending(): Promise<GameScoreResult | null> {
    if (!loggedIn.value) return null
    let p: Pending | null = null
    try {
      p = JSON.parse(sessionStorage.getItem(pendingKey) || 'null')
      sessionStorage.removeItem(pendingKey)
    } catch {
      p = null
    }
    if (!p || Date.now() - p.at > PENDING_TTL_MS) return null
    const r = await submit(p.score, p.detail)
    if (r) track('game_cta', { game, type: 'login', cta: 'pending_submitted' })
    return r
  }

  /**
   * Cloud save for games that keep progress in localStorage: merges the local copy with the account's,
   * writes the merge back to both, and returns it.
   */
  async function syncSave<T extends object>(localKey: string, merge: (local: T | null, remote: T | null) => T): Promise<T | null> {
    if (!loggedIn.value) return null
    let local: T | null = null
    try {
      local = JSON.parse(localStorage.getItem(localKey) || 'null')
    } catch {
      local = null
    }
    try {
      const remote = await getGameSave(game)
      const r = (remote.data?.[localKey] as T | undefined) ?? null
      const merged = merge(local, r)
      localStorage.setItem(localKey, JSON.stringify(merged))
      if (JSON.stringify(merged) !== JSON.stringify(r)) await putGameSave(game, { [localKey]: merged })
      return merged
    } catch {
      return null
    }
  }

  function trackCta(cta: string): void {
    track('game_cta', { game, type: 'account', cta })
  }

  return { loggedIn, loginHref, submit, flushPending, syncSave, lastResult, submitting, signupBonus, trackCta }
}
