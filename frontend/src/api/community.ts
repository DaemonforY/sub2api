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
  description?: string
  /** site: a web page made on the canvas (the cover is a screenshot). */
  kind?: 'image' | 'site'
  visibility?: 'public' | 'unlisted' | 'private'
  status?: 'approved' | 'pending' | 'rejected' | 'hidden'
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

/** The logged-in user's own works in every state (the contest page's "enter a canvas work" picker). */
export async function myWorks(): Promise<CommunityWork[]> {
  const { data } = await apiClient.get<{ works: CommunityWork[] }>('/user/community/works')
  return data.works || []
}

/** Why a work cannot be entered in a contest ('' when it can). */
export function contestBlockReason(w: CommunityWork): '' | 'private' | 'review' {
  if (w.visibility === 'private') return 'private'
  if (w.status && w.status !== 'approved') return 'review'
  return ''
}
