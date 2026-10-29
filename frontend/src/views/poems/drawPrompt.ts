import type { Poem } from '@/api/poems'
import { canvasUrl } from '@/constants/crossSites'

/**
 * 用这首诗生图: opens the canvas 诗画 with this poem. The canvas fetches the poem, writes the scene,
 * draws it in the chosen style and sets the poem on the picture in calligraphy.
 */
export function poemDrawLink(p: Pick<Poem, 'id'>): string {
  return canvasUrl({ medium: 'poems', baseUrl: window.location.origin, path: '/poem', query: { poem: String(p.id) } })
}
