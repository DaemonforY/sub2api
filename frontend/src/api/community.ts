/**
 * Canvas community works shown on the main site's home page (public works only).
 * The works themselves live on the canvas site (/w/:id); images are public files on this site.
 */
import { apiClient } from './client'

export interface CommunityAuthor {
  handle: string
  display_name: string
  avatar_url: string
}

export interface CommunityWork {
  id: number
  title: string
  prompt: string
  author: CommunityAuthor
  cover_url: string
  cover_thumb_url: string
  cover_width: number
  cover_height: number
  image_count: number
  like_count: number
  remix_count: number
}

let recommended: Promise<CommunityWork[]> | null = null

/** Recommended public works (one request per page load; failures resolve to an empty list). */
export function recommendedWorks(): Promise<CommunityWork[]> {
  recommended ??= apiClient
    .get<{ works: CommunityWork[] }>('/community/works', { params: { feed: 'recommended', limit: 24 } })
    .then(({ data }) => data.works || [])
    .catch(() => {
      recommended = null
      return []
    })
  return recommended
}

export function authorName(author: CommunityAuthor): string {
  return author.display_name || `@${author.handle}`
}
