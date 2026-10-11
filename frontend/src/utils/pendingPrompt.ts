/**
 * The picture a visitor described on the home page before they had an account. It waits in
 * localStorage (not the URL, so it stays out of logs) through sign-up — email code, WeChat or a
 * login — and /first-image takes it to the canvas. Kept for a day.
 */
const KEY = 'hg_pending_prompt'
const TTL_MS = 24 * 60 * 60 * 1000
const MAX_LENGTH = 1000

export function savePendingPrompt(prompt: string): void {
  const text = prompt.trim().slice(0, MAX_LENGTH)
  try {
    if (text) localStorage.setItem(KEY, JSON.stringify({ text, at: Date.now() }))
    else localStorage.removeItem(KEY)
  } catch {
    // storage blocked: the visitor just starts from an empty prompt
  }
}

/** Returns the waiting prompt ('' if none or expired) and forgets it. */
export function takePendingPrompt(): string {
  try {
    const raw = localStorage.getItem(KEY)
    localStorage.removeItem(KEY)
    if (!raw) return ''
    const saved = JSON.parse(raw) as { text?: unknown; at?: unknown }
    if (typeof saved.text !== 'string' || typeof saved.at !== 'number' || Date.now() - saved.at > TTL_MS) return ''
    return saved.text.slice(0, MAX_LENGTH)
  } catch {
    return ''
  }
}
