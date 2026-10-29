import { describe, expect, it } from 'vitest'
import { poemDrawLink } from '../drawPrompt'

describe('poemDrawLink', () => {
  it('opens the canvas 诗画 with the poem id and the gateway', () => {
    const url = new URL(poemDrawLink({ id: 627041 }))
    expect(url.pathname).toBe('/poem')
    expect(url.searchParams.get('poem')).toBe('627041')
    expect(url.searchParams.get('utm_medium')).toBe('poems')
    expect(url.searchParams.get('baseUrl')).toBe(window.location.origin)
  })
})
