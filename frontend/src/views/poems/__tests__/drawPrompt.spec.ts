import { describe, expect, it } from 'vitest'
import { poemDrawLink, poemDrawPrompt } from '../drawPrompt'

const cunju = { title: '村居', author: '高鼎', dynasty: '清', content: '草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。' }

describe('poemDrawPrompt', () => {
  it('names the poem, asks for no text and includes the whole poem', () => {
    const p = poemDrawPrompt(cunju)
    expect(p).toContain('《村居》（清·高鼎）')
    expect(p).toContain('不要出现任何文字')
    expect(p).toContain('儿童散学归来早，忙趁东风放纸鸢。')
    expect(p).not.toContain('\n')
  })

  it('stays within the canvas prompt limit for long prose', () => {
    expect(poemDrawPrompt({ ...cunju, content: '长'.repeat(5000) }).length).toBe(1000)
  })

  it('opens the canvas image workbench with the prompt', () => {
    const url = new URL(poemDrawLink(cunju))
    expect(url.pathname).toBe('/image')
    expect(url.searchParams.get('prompt')).toContain('村居')
    expect(url.searchParams.get('utm_medium')).toBe('poems')
  })
})
