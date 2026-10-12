// Whether this device gets the motion screens, and how much (no three.js in here).
import type { MotionOptions } from './stage'

let detected: MotionOptions | null | undefined

/** null: no WebGL, data saver on, or a weak device — keep the static background. Probed once per page. */
export function detectMotion(): MotionOptions | null {
  if (detected === undefined) detected = probe()
  return detected
}

function probe(): MotionOptions | null {
  if (typeof window === 'undefined' || typeof WebGLRenderingContext === 'undefined') return null
  const nav = navigator as Navigator & { connection?: { saveData?: boolean }; deviceMemory?: number }
  if (nav.connection?.saveData) return null
  if ((nav.deviceMemory ?? 8) <= 2 || (nav.hardwareConcurrency ?? 8) <= 2) return null
  try {
    const c = document.createElement('canvas')
    if (!(c.getContext('webgl2') || c.getContext('webgl'))) return null
  } catch {
    return null
  }
  const still = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  const narrow = window.matchMedia?.('(max-width: 760px)').matches ?? false
  return { tier: narrow ? 'mobile' : 'desktop', still, density: 1 }
}

export function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false)
}
