import type { Poem } from '@/api/poems'
import { canvasUrl } from '@/constants/crossSites'

/** The canvas prompt for 用这首诗生图: the poem's scene in the 新中式 look, no text in the picture. */
export function poemDrawPrompt(p: Pick<Poem, 'title' | 'author' | 'dynasty' | 'content'>): string {
  const who = `${p.dynasty ? `${p.dynasty}·` : ''}${p.author}`
  const head = `新中式细线设色插画，竖版构图，画出古诗《${p.title}》（${who}）描绘的画面，意象和季节要符合原诗，画面干净雅致，不要出现任何文字。原诗：`
  const text = p.content.replace(/\n+/g, '')
  return (head + text).slice(0, 1000)
}

/** The canvas link that opens the image workbench with that prompt. */
export function poemDrawLink(p: Pick<Poem, 'title' | 'author' | 'dynasty' | 'content'>): string {
  return canvasUrl({ medium: 'poems', baseUrl: window.location.origin, path: '/image', prompt: poemDrawPrompt(p) })
}
