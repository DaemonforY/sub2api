// Scoring for ABTI: each dimension's two answers add up to 2–6 points, read as L (≤3) / M (4) / H (≥5);
// the 15 levels are compared with every type's pattern by Manhattan distance and the nearest wins.
import {
  DIMENSIONS,
  FALLBACK_TYPE,
  GATE_FOLLOW_UP,
  GATE_QUESTION,
  HIDDEN_TYPE,
  MIN_SIMILARITY,
  QUESTIONS,
  TYPES,
  type Level,
  type PersonaType,
  type Question
} from './data'

export type Answers = Record<string, number>

export interface RankedType {
  type: PersonaType
  distance: number
  exact: number
  similarity: number
}

export interface AbtiResult {
  kind: 'normal' | 'hidden' | 'fallback'
  type: PersonaType
  // The closest everyday type, also shown under the hidden and fallback types.
  best: RankedType
  ranked: RankedType[]
  scores: Record<string, number>
  levels: Record<string, Level>
}

const LEVEL_NUM: Record<Level, number> = { L: 1, M: 2, H: 3 }
const MAX_DISTANCE = DIMENSIONS.length * 2

export function levelOf(score: number): Level {
  if (score <= 3) return 'L'
  if (score === 4) return 'M'
  return 'H'
}

export function parsePattern(pattern: string): Level[] {
  return pattern.replace(/-/g, '').split('') as Level[]
}

export function allTypes(): PersonaType[] {
  return [...TYPES, HIDDEN_TYPE, FALLBACK_TYPE]
}

export function findType(code: string): PersonaType | undefined {
  const c = code.toUpperCase()
  return allTypes().find((t) => t.code === c)
}

/** The questions in a fresh random order, with the gate question slipped in somewhere after the first. */
export function questionOrder(random: () => number = Math.random): Question[] {
  const list = [...QUESTIONS]
  for (let i = list.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[list[i], list[j]] = [list[j], list[i]]
  }
  const at = 1 + Math.floor(random() * list.length)
  list.splice(at, 0, GATE_QUESTION)
  return list
}

/** The order as shown: the follow-up appears right after the gate question once it is answered with 3. */
export function visibleQuestions(order: Question[], answers: Answers): Question[] {
  const out: Question[] = []
  for (const q of order) {
    out.push(q)
    if (q.id === GATE_QUESTION.id && answers[GATE_QUESTION.id] === 3) out.push(GATE_FOLLOW_UP)
  }
  return out
}

export function computeResult(answers: Answers): AbtiResult {
  const scores: Record<string, number> = {}
  for (const d of DIMENSIONS) scores[d.id] = 0
  for (const q of QUESTIONS) scores[q.dim] += Number(answers[q.id] || 0)
  const levels: Record<string, Level> = {}
  for (const d of DIMENSIONS) levels[d.id] = levelOf(scores[d.id])
  const user = DIMENSIONS.map((d) => LEVEL_NUM[levels[d.id]])

  const ranked = TYPES.map((type) => {
    const v = parsePattern(type.pattern || '').map((l) => LEVEL_NUM[l])
    let distance = 0
    let exact = 0
    v.forEach((n, i) => {
      const diff = Math.abs(user[i] - n)
      distance += diff
      if (diff === 0) exact++
    })
    return { type, distance, exact, similarity: Math.max(0, Math.round((1 - distance / MAX_DISTANCE) * 100)) }
  }).sort((a, b) => a.distance - b.distance || b.exact - a.exact)

  const best = ranked[0]
  let kind: AbtiResult['kind'] = 'normal'
  let type = best.type
  if (answers[GATE_QUESTION.id] === 3 && answers[GATE_FOLLOW_UP.id] === 2) {
    kind = 'hidden'
    type = HIDDEN_TYPE
  } else if (best.similarity < MIN_SIMILARITY) {
    kind = 'fallback'
    type = FALLBACK_TYPE
  }
  return { kind, type, best, ranked, scores, levels }
}

/** Answers that land exactly on a pattern: 1+1 for L, 2+2 for M, 3+3 for H. */
export function answersForPattern(pattern: string): Answers {
  const levels = parsePattern(pattern)
  const answers: Answers = { [GATE_QUESTION.id]: 1 }
  for (const q of QUESTIONS) {
    const i = DIMENSIONS.findIndex((d) => d.id === q.dim)
    answers[q.id] = LEVEL_NUM[levels[i]]
  }
  return answers
}
