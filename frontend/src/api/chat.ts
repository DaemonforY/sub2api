/**
 * AI 对话 (/chat): conversations kept on the server, model calls on the user's own keys.
 */
import { apiClient } from './client'
import { refreshAuthTokens } from './tokenRefresh'
import { buildApiUrl } from './url'

export interface ChatKey {
  id: number
  name: string
  group: string
  platform: string
  rate: number
  /** The group is OpenAI, so 画图 works on it. */
  can_draw: boolean
}

export interface ChatModels {
  models: string[]
  can_draw: boolean
}

export interface ChatConversation {
  id: number
  title: string
  system_prompt: string
  api_key_id: number | null
  model: string
  pinned: boolean
  created_at: string
  updated_at: string
}

export type ChatMessageStatus = 'pending' | 'done' | 'stopped' | 'error'

export interface ChatMessage {
  id: number
  conversation_id: number
  role: 'user' | 'assistant'
  kind: 'text' | 'image'
  content: string
  reasoning?: string
  files: string[]
  model?: string
  status: ChatMessageStatus
  error?: string
  prompt_tokens?: number
  completion_tokens?: number
  created_at: string
}

export interface ChatConversationDetail {
  conversation: ChatConversation
  messages: ChatMessage[]
  /** A reply (e.g. a picture) is still being made on the server. */
  running: boolean
}

export interface ChatUploaded {
  name: string
  width: number
  height: number
}

export interface ChatSendInput {
  key_id: number
  model: string
  mode: 'text' | 'image'
  content?: string
  files?: string[]
  image_size?: string
  regenerate?: boolean
  edit_id?: number
}

export interface ChatEvent {
  user?: ChatMessage
  assistant?: ChatMessage
  conversation?: ChatConversation
  delta?: string
  reasoning?: string
  done?: ChatMessage
  error?: string
}

export async function listKeys(): Promise<ChatKey[]> {
  const { data } = await apiClient.get<ChatKey[]>('/chat/keys')
  return data
}

export async function listModels(keyId: number): Promise<ChatModels> {
  const { data } = await apiClient.get<ChatModels>('/chat/models', { params: { key_id: keyId } })
  return data
}

export async function listConversations(q = ''): Promise<ChatConversation[]> {
  const { data } = await apiClient.get<ChatConversation[]>('/chat/conversations', { params: q ? { q } : {} })
  return data
}

export async function createConversation(input: { title?: string; system_prompt?: string } = {}): Promise<ChatConversation> {
  const { data } = await apiClient.post<ChatConversation>('/chat/conversations', input)
  return data
}

export async function getConversation(id: number): Promise<ChatConversationDetail> {
  const { data } = await apiClient.get<ChatConversationDetail>(`/chat/conversations/${id}`)
  return data
}

export async function updateConversation(
  id: number,
  input: { title?: string; system_prompt?: string; pinned?: boolean }
): Promise<ChatConversation> {
  const { data } = await apiClient.put<ChatConversation>(`/chat/conversations/${id}`, input)
  return data
}

export async function deleteConversation(id: number): Promise<void> {
  await apiClient.delete(`/chat/conversations/${id}`)
}

export async function uploadFile(file: Blob, filename = 'image.jpg'): Promise<ChatUploaded> {
  const form = new FormData()
  form.append('file', file, filename)
  const { data } = await apiClient.post<ChatUploaded>('/chat/files', form, { timeout: 120_000 })
  return data
}

const fileURLs = new Map<string, Promise<string>>()

/**
 * An object URL for one of the user's pictures, fetched once with the login token. The page shows
 * previews (a JPEG of at most 1280 px); the original is for 下载 and the full-size view.
 */
export function fileURL(name: string, preview = true): Promise<string> {
  const key = (preview ? 'p:' : 'o:') + name
  let p = fileURLs.get(key)
  if (!p) {
    p = apiClient
      .get<Blob>(`/chat/files/${encodeURIComponent(name)}`, {
        params: preview ? { preview: 1 } : {},
        responseType: 'blob',
        timeout: 120_000
      })
      .then((res) => URL.createObjectURL(res.data as unknown as Blob))
    p.catch(() => fileURLs.delete(key))
    fileURLs.set(key, p)
  }
  return p
}

/** Forgets a picture that failed to show, so the next fileURL call fetches it again. */
export function forgetFileURL(name: string) {
  fileURLs.delete('p:' + name)
  fileURLs.delete('o:' + name)
}

async function authedFetch(url: string, init: RequestInit): Promise<Response> {
  const call = (token: string | null) => {
    const headers = new Headers(init.headers)
    if (token) headers.set('Authorization', `Bearer ${token}`)
    return fetch(url, { ...init, headers })
  }
  const token = localStorage.getItem('auth_token')
  const res = await call(token)
  if (res.status !== 401 || !localStorage.getItem('refresh_token')) return res
  try {
    const fresh = await refreshAuthTokens({ failedAccessToken: token })
    return await call(fresh.access_token)
  } catch {
    return res
  }
}

/**
 * Sends one turn and streams its events to onEvent. Rejects with the server's message when the turn
 * could not start; once it started, failures arrive as an event with `error`.
 */
export async function sendTurn(
  conversationId: number,
  input: ChatSendInput,
  onEvent: (ev: ChatEvent) => void,
  signal?: AbortSignal
): Promise<void> {
  const res = await authedFetch(buildApiUrl(`/chat/conversations/${conversationId}/send`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
    body: JSON.stringify(input),
    signal
  })
  const type = res.headers.get('Content-Type') || ''
  if (!type.includes('text/event-stream')) {
    let body: { message?: string } = {}
    try {
      body = await res.json()
    } catch {
      // not JSON
    }
    throw new Error(body.message || `请求失败（HTTP ${res.status}）`)
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let i: number
    while ((i = buffer.indexOf('\n\n')) >= 0) {
      const event = buffer.slice(0, i)
      buffer = buffer.slice(i + 2)
      const line = event.split('\n').find((l) => l.startsWith('data: '))
      if (line) onEvent(JSON.parse(line.slice(6)) as ChatEvent)
    }
  }
}
