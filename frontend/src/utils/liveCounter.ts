/**
 * The home page's TOKENS PROCESSED counter only ever shows real usage. When the total grows between
 * two polls, the increase is played out as a few visible steps — each shown as 「+N」 — that add up
 * to exactly what was used.
 */
export function splitIncrease(delta: number, rnd: () => number = Math.random): number[] {
  const total = Math.floor(delta)
  if (total <= 0) return []
  const steps = Math.min(total, total < 1000 ? 2 : total < 100000 ? 3 : 4)
  const weights = Array.from({ length: steps }, () => 0.5 + rnd())
  const sum = weights.reduce((a, b) => a + b, 0)
  const parts = weights.map((w) => Math.max(1, Math.floor((total * w) / sum)))
  parts[parts.length - 1] += total - parts.reduce((a, b) => a + b, 0) // exact, whatever the rounding
  return parts.filter((p) => p > 0)
}

export function formatCount(n: number): string {
  return Math.round(n).toLocaleString('en-US')
}
