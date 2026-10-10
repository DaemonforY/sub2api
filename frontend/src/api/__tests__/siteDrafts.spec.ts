import { describe, expect, it } from 'vitest'
import { previewHtml, type SiteDraftImage } from '@/api/siteDrafts'

describe('previewHtml', () => {
  const images: SiteDraftImage[] = [
    { n: 1, prompt: 'a', size: '1536x1024', status: 'ok' },
    { n: 2, prompt: 'b', size: '1536x1024', status: 'pending' },
    { n: 3, prompt: 'c', size: '1536x1024', status: 'failed' }
  ]
  const labels = { pending: '画图中', failed: '失败' }

  it('points drawn pictures at their object URLs and the rest at placeholders', () => {
    const out = previewHtml(
      `<img src="img/1.jpg"><img SRC='img/2.jpg'><img src="img/3.jpg"><img src="logo.png">`,
      { 1: 'blob:x/1' },
      labels,
      images
    )
    expect(out).toContain('src="blob:x/1"')
    expect(out).toContain(encodeURIComponent('画图中'))
    expect(out).toContain(encodeURIComponent('失败'))
    expect(out).toContain('src="logo.png"')
    expect(out).not.toContain('img/2.jpg')
  })
})
