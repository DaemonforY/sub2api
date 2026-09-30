import { describe, expect, it } from 'vitest'
import { parseTags, toAdminInput, type PromptItem } from '../promptLibrary'

describe('prompt library admin helpers', () => {
  it('splits tags on commas, spaces and hashes, dropping repeats', () => {
    expect(parseTags('国风, 手绘 #山水，国风、 ')).toEqual(['国风', '手绘', '山水'])
    expect(parseTags('')).toEqual([])
  })

  it('copies an item into an editable form without sharing arrays', () => {
    const item = {
      title: 't', prompt: 'p', description: '', cover_url: '', kind: 'image', scenes: ['poster'], tags: ['a'], model: 'gpt-image-2',
      needs_reference: false, status: 'active', visibility: 'public', featured: false
    } as unknown as PromptItem
    const form = toAdminInput(item)
    form.scenes.push('brand')
    expect(item.scenes).toEqual(['poster'])
    expect(form.review_note).toBe('')
  })
})
