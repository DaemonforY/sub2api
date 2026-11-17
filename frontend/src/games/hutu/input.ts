// Keyboard, gamepad and touch input, packed into BTN bitmasks.
//
// Same keyboard, two players: P1 = WASD + J (jump) / K (run · fire);
// P2 = arrow keys + numpad 1 (jump) / numpad 2 (run · fire), or . and / on laptops.
// Playing alone (or online), either layout works, plus Space / Z / X / Shift.

import { BTN } from './types'

const P1: Record<string, number> = {
  KeyA: BTN.L,
  KeyD: BTN.R,
  KeyW: BTN.U,
  KeyS: BTN.D,
  KeyJ: BTN.J,
  KeyK: BTN.F
}
const P2: Record<string, number> = {
  ArrowLeft: BTN.L,
  ArrowRight: BTN.R,
  ArrowUp: BTN.U,
  ArrowDown: BTN.D,
  Numpad1: BTN.J,
  Numpad2: BTN.F,
  Period: BTN.J,
  Slash: BTN.F,
  NumpadDecimal: BTN.J,
  Numpad0: BTN.F
}
const SOLO_EXTRA: Record<string, number> = {
  Space: BTN.J,
  KeyZ: BTN.J,
  KeyX: BTN.F,
  ShiftLeft: BTN.F,
  ShiftRight: BTN.F,
  KeyL: BTN.J
}

export class Input {
  private down = new Set<string>()
  /** bits from the on-screen buttons */
  touch = 0
  /** true while the page wants keys (not typing in a text field) */
  enabled = true
  private onKey = (e: KeyboardEvent) => this.key(e, true)
  private onKeyUp = (e: KeyboardEvent) => this.key(e, false)
  private onBlur = () => this.down.clear()
  onAction: ((a: 'pause' | 'mute') => void) | null = null

  attach(): void {
    window.addEventListener('keydown', this.onKey)
    window.addEventListener('keyup', this.onKeyUp)
    window.addEventListener('blur', this.onBlur)
  }

  detach(): void {
    window.removeEventListener('keydown', this.onKey)
    window.removeEventListener('keyup', this.onKeyUp)
    window.removeEventListener('blur', this.onBlur)
  }

  private key(e: KeyboardEvent, on: boolean): void {
    const t = e.target as HTMLElement | null
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA')) return
    if (!this.enabled) return
    if (on && !e.repeat && (e.code === 'Escape' || e.code === 'KeyP' || e.code === 'Enter')) this.onAction?.('pause')
    if (on && !e.repeat && e.code === 'KeyM') this.onAction?.('mute')
    if (e.code in P1 || e.code in P2 || e.code in SOLO_EXTRA) e.preventDefault()
    if (on) this.down.add(e.code)
    else this.down.delete(e.code)
  }

  private bits(map: Record<string, number>): number {
    let b = 0
    for (const code of this.down) if (map[code]) b |= map[code]
    return b
  }

  private pad(i: number): number {
    const pads = typeof navigator !== 'undefined' && navigator.getGamepads ? navigator.getGamepads() : []
    const p = pads[i]
    if (!p) return 0
    let b = 0
    const btn = (n: number) => !!p.buttons[n]?.pressed
    const ax = p.axes[0] || 0
    const ay = p.axes[1] || 0
    if (btn(14) || ax < -0.4) b |= BTN.L
    if (btn(15) || ax > 0.4) b |= BTN.R
    if (btn(12) || ay < -0.6) b |= BTN.U
    if (btn(13) || ay > 0.6) b |= BTN.D
    if (btn(0) || btn(3)) b |= BTN.J
    if (btn(1) || btn(2) || btn(7)) b |= BTN.F
    if (btn(9) && !this.padStart[i]) this.onAction?.('pause')
    this.padStart[i] = btn(9)
    return b
  }
  private padStart: boolean[] = [false, false]

  /** buttons for one player. `split` = two players share this keyboard */
  read(slot: number, split: boolean): number {
    let b: number
    if (!split) b = this.bits(P1) | this.bits(P2) | this.bits(SOLO_EXTRA) | this.pad(0) | this.pad(1) | this.touch
    else if (slot === 0) b = this.bits(P1) | this.bits(SOLO_EXTRA_P1) | this.pad(0) | this.touch
    else b = this.bits(P2) | this.pad(1)
    // up also jumps (handy on a d-pad); opposite directions cancel
    if (b & BTN.U) b |= BTN.J
    if ((b & BTN.L) && (b & BTN.R)) b &= ~(BTN.L | BTN.R)
    return b
  }

  clear(): void {
    this.down.clear()
    this.touch = 0
  }
}

const SOLO_EXTRA_P1: Record<string, number> = { Space: BTN.J, ShiftLeft: BTN.F }
