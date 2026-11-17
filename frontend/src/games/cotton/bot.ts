// A very plain bot that plays 棉花糖大作战 badly but busily: walk to the nearest pest, face it
// and shoot until it is a ball, then kick. Used in development to fill screenshots and to soak
// the game; never wired to real players.

import { ALIVE, NO_BUTTONS, SNOW_FULL, tileAt, type Buttons, type GameState } from './sim'

export function botButtons(s: GameState, idx: number): Buttons {
  const p = s.players[idx]
  const b: Buttons = { ...NO_BUTTONS }
  if (!p.on || p.state !== ALIVE) return b
  const px = p.x + p.w / 2
  const py = p.y + p.h
  let best = null
  let bd = 1e9
  for (const e of s.enemies) {
    if (e.dying || e.spawnT > 0 || e.kind === 'slipper' || e.rolling) continue
    const d = Math.abs(e.x + e.w / 2 - px) + Math.abs(e.y + e.h - py) * 3
    if (d < bd) {
      bd = d
      best = e
    }
  }
  const t = s.frame + idx * 17
  if (!best) {
    b.r = t % 120 < 60
    b.l = !b.r
    return b
  }
  const ex = best.x + best.w / 2
  const dy = best.y + best.h - py
  const dx = ex - px
  const same = Math.abs(dy) < 10 || best.boss
  if (best.snow >= SNOW_FULL) {
    // walk into the ball and kick it
    if (Math.abs(dx) > 14) {
      b.r = dx > 0
      b.l = dx < 0
    } else {
      b.r = dx > 0 && Math.abs(dx) > 10
      b.l = dx < 0 && Math.abs(dx) > 10
      if (Math.sign(dx) === p.face || Math.abs(dx) < 3) b.fire = t % 6 < 3
      else {
        b.r = dx > 0
        b.l = dx < 0
      }
    }
  } else if (same) {
    if (Math.abs(dx) > 70) {
      b.r = dx > 0
      b.l = dx < 0
    } else if (Math.abs(dx) < 18) {
      // too close: back off
      b.r = dx < 0
      b.l = dx > 0
    } else if (Math.sign(dx) !== p.face) {
      b.r = dx > 0
      b.l = dx < 0
    }
    b.fire = t % 8 < 4
  } else if (dy < 0) {
    // climb: find a shelf just above and jump up through it
    const row = Math.floor(py / 8)
    let goal = -1
    let gd = 1e9
    for (let r = row - 2; r >= row - 6; r--) {
      for (let c = 1; c < 39; c++) {
        if (tileAt(s, c, r) !== 2) continue
        const d = Math.abs(c * 8 + 4 - px) + Math.abs(c * 8 + 4 - ex) * 0.5
        if (d < gd) {
          gd = d
          goal = c * 8 + 4
        }
      }
      if (goal >= 0) break
    }
    if (goal < 0) goal = ex
    const gx = goal - px
    b.r = gx > 4
    b.l = gx < -4
    b.jump = p.ground && Math.abs(gx) < 20 ? true : !p.ground && p.vy < 0
    b.fire = t % 10 < 5
  } else {
    b.r = dx > 0
    b.l = dx < 0
    b.d = Math.abs(dx) < 40
    b.jump = b.d && p.ground && t % 30 < 15
  }
  return b
}
