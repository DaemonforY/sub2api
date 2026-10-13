import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { DIMENSIONS, GATE_FOLLOW_UP, GATE_QUESTION, QUESTIONS, TYPES } from '../data'
import { allTypes, answersForPattern, computeResult, findType, levelOf, questionOrder, visibleQuestions } from '../score'

describe('ABTI data', () => {
  it('has two questions with values 1–3 for every dimension', () => {
    for (const d of DIMENSIONS) {
      const qs = QUESTIONS.filter((q) => q.dim === d.id)
      expect(qs, d.id).toHaveLength(2)
      for (const q of qs) expect(q.options.map((o) => o.value).sort()).toEqual([1, 2, 3])
    }
    expect(new Set(QUESTIONS.map((q) => q.id)).size).toBe(QUESTIONS.length)
  })

  it('gives every type a distinct, well-formed pattern', () => {
    const patterns = TYPES.map((t) => t.pattern)
    expect(new Set(patterns).size).toBe(TYPES.length)
    for (const p of patterns) expect(p).toMatch(/^[LMH]{3}(-[LMH]{3}){4}$/)
    expect(new Set(TYPES.map((t) => t.code)).size).toBe(TYPES.length)
  })

  it('has a picture for every type, and the server knows every code (link previews)', () => {
    const seo = readFileSync(resolve(__dirname, '../../../../../backend/internal/web/seo.go'), 'utf8')
    const block = /var abtiCodes = map\[string\]bool\{([\s\S]*?)\n\}/.exec(seo)?.[1] || ''
    const serverCodes = [...block.matchAll(/"([A-Z0-9]{4})"/g)].map((m) => m[1]).sort()
    const codes = allTypes().map((t) => t.code)
    expect(serverCodes).toEqual([...codes].sort())
    for (const code of codes) {
      expect(existsSync(resolve(__dirname, `../../../../public/games-media/abti/${code.toLowerCase()}.webp`)), code).toBe(true)
    }
  })
})

describe('computeResult', () => {
  it('reads two answers as a level', () => {
    expect([2, 3, 4, 5, 6].map(levelOf)).toEqual(['L', 'L', 'M', 'H', 'H'])
  })

  it('lands on each type when the answers match its pattern exactly', () => {
    for (const type of TYPES) {
      const r = computeResult(answersForPattern(type.pattern!))
      expect(r.kind, type.code).toBe('normal')
      expect(r.type.code).toBe(type.code)
      expect(r.best.similarity).toBe(100)
      expect(r.best.exact).toBe(DIMENSIONS.length)
    }
  })

  it('unlocks the hidden type only when the AI owns up', () => {
    const base = answersForPattern(TYPES[0].pattern!)
    expect(computeResult({ ...base, [GATE_QUESTION.id]: 3, [GATE_FOLLOW_UP.id]: 2 }).type.code).toBe('BOTX')
    expect(computeResult({ ...base, [GATE_QUESTION.id]: 3, [GATE_FOLLOW_UP.id]: 1 }).type.code).toBe(TYPES[0].code)
    expect(computeResult({ ...base, [GATE_QUESTION.id]: 1, [GATE_FOLLOW_UP.id]: 2 }).type.code).toBe(TYPES[0].code)
  })

  it('falls back when no type is close enough', () => {
    // At least 15 levels away from every type (found by search), so the best match is 50%.
    const r = computeResult(answersForPattern('HLH-HHL-LLH-HHL-LLL'))
    expect(r.best.similarity).toBeLessThan(60)
    expect(r.kind).toBe('fallback')
    expect(r.type.code).toBe('NULL')
  })
})

describe('question order', () => {
  it('keeps all questions, adds the gate once and shows the follow-up only after a 3', () => {
    const order = questionOrder(() => 0.42)
    expect(order).toHaveLength(QUESTIONS.length + 1)
    expect(order[0].id).not.toBe(GATE_QUESTION.id)
    expect(visibleQuestions(order, {})).toHaveLength(QUESTIONS.length + 1)
    const shown = visibleQuestions(order, { [GATE_QUESTION.id]: 3 })
    expect(shown).toHaveLength(QUESTIONS.length + 2)
    expect(shown[shown.findIndex((q) => q.id === GATE_QUESTION.id) + 1].id).toBe(GATE_FOLLOW_UP.id)
  })

  it('finds types by code in any case', () => {
    expect(findType('prom')?.name).toBe('提示词魔法师')
    expect(findType('botx')?.code).toBe('BOTX')
    expect(findType('nope')).toBeUndefined()
  })
})
