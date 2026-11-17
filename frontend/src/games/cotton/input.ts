// Keyboard, gamepads and on-screen buttons → one Buttons record per kid.
// Same keyboard: 1P uses WASD + J/K (F/G and Space also work), 2P uses the arrows + . and /
// (numpad 1 / 2 also work). Playing alone, every key drives the one kid.

import { NO_BUTTONS, type Buttons } from './sim'

type Scheme = Record<keyof Buttons, string[]>

const P1: Scheme = {
  l: ['KeyA'],
  r: ['KeyD'],
  u: [],
  d: ['KeyS'],
  jump: ['KeyK', 'KeyG', 'Space', 'KeyW'],
  fire: ['KeyJ', 'KeyF']
}

const P2: Scheme = {
  l: ['ArrowLeft'],
  r: ['ArrowRight'],
  u: [],
  d: ['ArrowDown'],
  jump: ['Slash', 'Numpad2', 'ArrowUp'],
  fire: ['Period', 'Numpad1', 'Comma']
}

const SOLO_EXTRA: Scheme = { l: [], r: [], u: [], d: [], jump: ['KeyX'], fire: ['KeyZ'] }

const GAME_KEYS = new Set([...Object.values(P1).flat(), ...Object.values(P2).flat(), ...Object.values(SOLO_EXTRA).flat()])

export class Controls {
  private keys = new Set<string>()
  private touch: Buttons = { ...NO_BUTTONS }
  /** a key was pressed this frame: used for "press to join" and menus */
  pressed: string[] = []
  private onDown = (e: KeyboardEvent) => {
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
    if (GAME_KEYS.has(e.code)) e.preventDefault()
    if (!e.repeat) this.pressed.push(e.code)
    this.keys.add(e.code)
  }
  private onUp = (e: KeyboardEvent) => {
    this.keys.delete(e.code)
  }
  private onBlur = () => {
    this.keys.clear()
  }

  attach(): void {
    window.addEventListener('keydown', this.onDown)
    window.addEventListener('keyup', this.onUp)
    window.addEventListener('blur', this.onBlur)
  }

  detach(): void {
    window.removeEventListener('keydown', this.onDown)
    window.removeEventListener('keyup', this.onUp)
    window.removeEventListener('blur', this.onBlur)
  }

  setTouch(b: Partial<Buttons>): void {
    this.touch = { ...this.touch, ...b }
  }

  private read(s: Scheme): Buttons {
    const k = this.keys
    const any = (codes: string[]) => codes.some((c) => k.has(c))
    return { l: any(s.l), r: any(s.r), u: any(s.u), d: any(s.d), jump: any(s.jump), fire: any(s.fire) }
  }

  private pad(i: number): Buttons {
    const pads = typeof navigator !== 'undefined' && navigator.getGamepads ? navigator.getGamepads() : []
    const gp = pads?.[i]
    if (!gp) return { ...NO_BUTTONS }
    const b = (n: number) => !!gp.buttons[n]?.pressed
    const ax = gp.axes[0] ?? 0
    const ay = gp.axes[1] ?? 0
    return {
      l: b(14) || ax < -0.4,
      r: b(15) || ax > 0.4,
      u: b(12) || ay < -0.6,
      d: b(13) || ay > 0.6,
      jump: b(0),
      fire: b(1) || b(2) || b(3)
    }
  }

  /** controls for the kids on this machine. `solo` merges every key into the first kid. */
  poll(solo: boolean): [Buttons, Buttons] {
    const a = this.read(P1)
    const b = this.read(P2)
    const pa = this.pad(0)
    const pb = this.pad(1)
    if (solo) {
      const x = this.read(SOLO_EXTRA)
      return [or(a, b, x, pa, pb, this.touch), { ...NO_BUTTONS }]
    }
    return [or(a, pa, this.touch), or(b, pb)]
  }

  /** a 2P key or the second pad was pressed (press to join) */
  p2Pressed(): boolean {
    const p2 = [...P2.fire, ...P2.jump]
    if (this.pressed.some((c) => p2.includes(c))) return true
    const pb = this.pad(1)
    return pb.fire || pb.jump
  }

  endFrame(): void {
    this.pressed = []
  }
}

function or(...bs: Buttons[]): Buttons {
  const r = { ...NO_BUTTONS }
  for (const b of bs) {
    r.l ||= b.l
    r.r ||= b.r
    r.u ||= b.u
    r.d ||= b.d
    r.jump ||= b.jump
    r.fire ||= b.fire
  }
  return r
}
