// State of the share card dialog (one per page, mounted in Layout.vue) and the sharer's invite code.
import { reactive } from 'vue'
import { api, currentUser, token } from './api'
import type { ShareBlock } from './shareCard'

export const shareState = reactive<{
  open: boolean
  kind: 'quote' | 'summary'
  blocks: ShareBlock[]
  truncated: boolean
  anchor: string
}>({ open: false, kind: 'summary', blocks: [], truncated: false, anchor: '' })

export function openSummaryCard() {
  Object.assign(shareState, { open: true, kind: 'summary', blocks: [], truncated: false, anchor: '' })
}

export function openQuoteCard(blocks: ShareBlock[], truncated: boolean, anchor: string) {
  Object.assign(shareState, { open: true, kind: 'quote', blocks, truncated, anchor })
}

const AFF_KEY = 'learn_share_aff'

/** The signed-in user's invite code ('' when signed out or unavailable), cached for the session. */
export async function myInviteCode(): Promise<string> {
  const user = currentUser()
  if (!user || !token()) return ''
  try {
    const cached = JSON.parse(sessionStorage.getItem(AFF_KEY) || 'null')
    if (cached && cached.uid === user.id) return String(cached.code || '')
  } catch {
    // ignore
  }
  try {
    const detail = await api<{ aff_code?: string }>('/user/aff')
    const code = detail?.aff_code || ''
    sessionStorage.setItem(AFF_KEY, JSON.stringify({ uid: user.id, code }))
    return code
  } catch {
    return ''
  }
}
