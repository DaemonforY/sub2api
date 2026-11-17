// A search bot for play-testing levels: it plays with the real World and physics, trying short
// bursts of input (run, jump of different lengths, wait, back up) and backtracking from deaths,
// to prove every level can be finished. The 金钟罩 is kept on so it tests the terrain (gaps,
// heights, lifts) rather than enemy timing.

import { World } from '../world'
import { BTN } from '../types'

interface Snap {
  json: string
  cells: string[]
  area: World['area']
}

const PLAIN = ['levelIdx', 'areaIdx', 'phase', 'phaseT', 'frame', 'cam', 'time', 'timeSub', 'lives', 'checkpointHit', 'nextId', 'musicNow', 'clearT', 'bossDown', 'paused'] as const

export function snap(w: World): Snap {
  const o: Record<string, unknown> = {}
  for (const k of PLAIN) o[k] = (w as unknown as Record<string, unknown>)[k]
  o.players = w.players
  o.ents = w.ents
  o.spawned = w.spawned
  o.coinHits = [...w.coinHits.entries()]
  o.pendingWarp = w.pendingWarp
  o.editLog = w.editLog
  return { json: JSON.stringify(o), cells: w.area.cells.slice(), area: w.area }
}

export function restore(w: World, s: Snap): void {
  const o = JSON.parse(s.json)
  for (const k of PLAIN) (w as unknown as Record<string, unknown>)[k] = o[k]
  w.players = o.players
  w.ents = o.ents
  w.spawned = o.spawned
  w.coinHits = new Map(o.coinHits)
  w.pendingWarp = o.pendingWarp
  w.editLog = o.editLog
  w.area = s.area
  w.area.cells.splice(0, w.area.cells.length, ...s.cells)
}

const R = BTN.R | BTN.F
const ACTIONS: { name: string; frames: number[] }[] = [
  { name: 'run', frames: Array(10).fill(R) },
  { name: 'jumpLong', frames: [...Array(26).fill(R | BTN.J), ...Array(10).fill(R)] },
  { name: 'jumpMid', frames: [...Array(12).fill(R | BTN.J), ...Array(14).fill(R)] },
  { name: 'walkJump', frames: [...Array(26).fill(BTN.R | BTN.J), ...Array(10).fill(BTN.R)] },
  { name: 'wait', frames: Array(20).fill(0) },
  { name: 'hop', frames: [...Array(5).fill(R | BTN.J), ...Array(10).fill(R)] },
  { name: 'standJump', frames: [...Array(26).fill(BTN.J), ...Array(8).fill(0)] },
  { name: 'down', frames: Array(50).fill(BTN.D) },
  { name: 'back', frames: Array(14).fill(BTN.L | BTN.F) },
  { name: 'backJump', frames: [...Array(26).fill(BTN.L | BTN.J), ...Array(6).fill(BTN.L)] }
]

function progress(w: World): number {
  const p = w.players[0]
  if (p.st === 'pole' || p.st === 'walk' || p.st === 'done' || w.phase === 'clear' || w.phase === 'won') return 1e9
  return w.areaIdx * 100000 + p.ps.x - (w.areaIdx > 0 ? 0 : 0)
}

function alive(w: World): boolean {
  const p = w.players[0]
  return p.st !== 'dead' && p.st !== 'out' && w.phase !== 'dead' && w.phase !== 'over' && w.phase !== 'intro'
}

/**
 * Try to finish level `idx`. Returns the furthest x reached and whether it got to the end.
 * `budget` caps simulated steps.
 */
export function playLevel(idx: number, budget = 400000, bell = true): { done: boolean; best: number; steps: number; trace: string[] } {
  const w = new World({ players: 1, lives: 99 })
  w.restartFrom(idx)
  while (w.phase === 'intro') w.tick([0])
  w.events.length = 0
  let steps = 0
  const visited = new Map<string, number>()
  let best = progress(w)
  const trace: string[] = []
  // depth-first with best-first ordering of children
  type Node = { s: Snap; depth: number; path: string[] }
  const stack: Node[] = [{ s: snap(w), depth: 0, path: [] }]
  while (stack.length && steps < budget) {
    const node = stack.pop() as Node
    const kids: { s: Snap; score: number; path: string[] }[] = []
    for (const a of ACTIONS) {
      restore(w, node.s)
      let ok = true
      for (const b of a.frames) {
        if (bell) w.players[0].bell = 9999
        w.tick([b])
        w.events.length = 0
        steps++
        if (!alive(w)) {
          ok = false
          break
        }
        if (progress(w) >= 1e9) break
      }
      if (!ok) continue
      const pr = progress(w)
      if (pr >= 1e9) return { done: true, best: pr, steps, trace: [...node.path, a.name] }
      const p = w.players[0].ps
      const key = `${w.areaIdx}:${Math.round(p.x / 6)}:${Math.round(p.y / 6)}:${p.ground ? 1 : 0}:${Math.floor(w.frame / 60) % 8}`
      const seen = visited.get(key) || 0
      if (seen > 1) continue
      visited.set(key, seen + 1)
      if (pr > best) best = pr
      kids.push({ s: snap(w), score: pr, path: [...node.path, a.name] })
    }
    kids.sort((a, b) => a.score - b.score)
    for (const k of kids) stack.push({ s: k.s, depth: node.depth + 1, path: k.path })
  }
  void trace
  return { done: false, best, steps, trace: [] }
}
