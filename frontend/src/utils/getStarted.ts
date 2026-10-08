/**
 * Progress of the dashboard "get started" card, kept in the browser per user. Creating a key and
 * the first API call come from the server (dashboard stats); copying a client config and closing
 * the card are only known here.
 */
type GuideStep = 'copied' | 'dismissed'

function key(step: GuideStep, userId?: number | string): string {
  return `hg_guide_${step}_${userId ?? currentUserId()}`
}

function currentUserId(): string {
  try {
    const u = JSON.parse(localStorage.getItem('auth_user') || 'null') as { id?: number } | null
    return u?.id ? String(u.id) : 'guest'
  } catch {
    return 'guest'
  }
}

export function markGuideStep(step: GuideStep, userId?: number | string): void {
  try {
    localStorage.setItem(key(step, userId), '1')
  } catch {
    // storage unavailable
  }
}

export function guideStepDone(step: GuideStep, userId?: number | string): boolean {
  try {
    return localStorage.getItem(key(step, userId)) === '1'
  } catch {
    return false
  }
}
