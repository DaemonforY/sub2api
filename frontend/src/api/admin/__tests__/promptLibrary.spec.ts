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

describe('admin covers', () => {
  it('relays blocked overseas hosts through the canvas, leaves others alone', async () => {
    const { canvasProxiedImageUrl } = await import('@/constants/crossSites')
    expect(canvasProxiedImageUrl('https://cms-assets.youmind.com/media/a b.jpg?x=1')).toBe('https://canvas.hivegpt.cn/img-proxy/cms-assets.youmind.com/media/a%20b.jpg?x=1')
    expect(canvasProxiedImageUrl('https://raw.githubusercontent.com/o/r/main/x.png')).toBe('https://canvas.hivegpt.cn/img-proxy/raw.githubusercontent.com/o/r/main/x.png')
    expect(canvasProxiedImageUrl('/api/v1/prompt-library/covers/x.png')).toBe('/api/v1/prompt-library/covers/x.png')
    expect(canvasProxiedImageUrl('https://example.com/x.png')).toBe('https://example.com/x.png')
    expect(canvasProxiedImageUrl('http://cms-assets.youmind.com/x.png')).toBe('http://cms-assets.youmind.com/x.png')
  })
})
