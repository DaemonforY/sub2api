// Keyboard, gamepad and touch input for 椰风岛大冒险.
import { IN_L, IN_R, IN_J, IN_F, IN_D } from './defs'

type Map = Record<string, number>

// one keyboard, two players: P1 on the left hand side, P2 round the arrow keys
const P1_SHARED: Map = { KeyA: IN_L, KeyD: IN_R, KeyW: IN_J, KeyK: IN_J, Space: IN_J, KeyJ: IN_F, KeyF: IN_F, KeyS: IN_D }
const P2_SHARED: Map = {
  ArrowLeft: IN_L,
  ArrowRight: IN_R,
  ArrowUp: IN_J,
  Slash: IN_J,
  Numpad2: IN_J,
  Period: IN_F,
  Numpad1: IN_F,
  ShiftRight: IN_F,
  ArrowDown: IN_D
}
// playing alone (or online): every common layout works
const SOLO: Map = { ...P1_SHARED, ...P2_SHARED, KeyZ: IN_J, KeyX: IN_F, KeyL: IN_J, Enter: 0 }

export const GAME_KEYS = new Set([...Object.keys(SOLO), 'Escape', 'KeyP'])

export class Input {
  private down = new Set<string>()
  touch = 0
  private onKey: (e: KeyboardEvent) => void
  private onBlur: () => void
  enabled = true

  constructor() {
    this.onKey = (e) => {
      if (!this.enabled) return
      const t = e.target as HTMLElement | null
      if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA')) return
      if (e.type === 'keydown') this.down.add(e.code)
      else this.down.delete(e.code)
      if (GAME_KEYS.has(e.code)) e.preventDefault()
    }
    this.onBlur = () => this.down.clear()
    window.addEventListener('keydown', this.onKey)
    window.addEventListener('keyup', this.onKey)
    window.addEventListener('blur', this.onBlur)
  }

  destroy() {
    window.removeEventListener('keydown', this.onKey)
    window.removeEventListener('keyup', this.onKey)
    window.removeEventListener('blur', this.onBlur)
  }

  private fromKeys(map: Map): number {
    let b = 0
    for (const code of this.down) b |= map[code] ?? 0
    return b
  }

  private pad(i: number): number {
    const pads = typeof navigator !== 'undefined' && navigator.getGamepads ? navigator.getGamepads() : []
    const list = Array.from(pads).filter((p): p is Gamepad => !!p && p.connected)
    const p = list[i]
    if (!p) return 0
    const btn = (k: number) => !!p.buttons[k]?.pressed
    let b = 0
    const ax = p.axes[0] ?? 0
    const ay = p.axes[1] ?? 0
    if (ax < -0.35 || btn(14)) b |= IN_L
    if (ax > 0.35 || btn(15)) b |= IN_R
    if (ay > 0.6 || btn(13)) b |= IN_D
    if (btn(0) || btn(12)) b |= IN_J
    if (btn(1) || btn(2) || btn(3) || btn(5) || btn(7)) b |= IN_F
    return b
  }

  /** Bits for one local player. shared = two players on this keyboard. */
  bits(player: 0 | 1, shared: boolean): number {
    if (!shared) return this.fromKeys(SOLO) | this.pad(0) | this.pad(1) | this.touch
    return player === 0 ? this.fromKeys(P1_SHARED) | this.pad(0) | this.touch : this.fromKeys(P2_SHARED) | this.pad(1)
  }

  pressed(code: string): boolean {
    return this.down.has(code)
  }

  anyPadButton(): boolean {
    const pads = typeof navigator !== 'undefined' && navigator.getGamepads ? navigator.getGamepads() : []
    return Array.from(pads).some((p) => !!p && p.buttons.some((b) => b.pressed))
  }
}
