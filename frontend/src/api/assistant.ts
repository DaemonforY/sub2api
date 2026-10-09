/** Homepage support assistant (智能客服): open to visitors, signed-in users get their own daily allowance. */
import { apiClient } from './client'
import { buildApiUrl } from './url'

export interface AssistantConfig {
  enabled: boolean
  left: number
  per_day: number
  logged_in: boolean
  /** What signing in would give a visitor. */
  user_per_day: number
  /** 账户诊断: this user's questions can look up their own account, keys and errors. */
  tools: boolean
}

export interface AssistantMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface AssistantSource {
  title: string
  url: string
}

export interface AssistantDone {
  sources: AssistantSource[]
  left: number
}

export async function getAssistantConfig(): Promise<AssistantConfig> {
  const { data } = await apiClient.get('/assistant/config')
  return data
}

/**
 * Asks the last question of messages; the answer streams to onDelta. With 账户诊断 on, onTool gets a
 * label (正在查你最近的报错…) each time the assistant looks something up. Errors carry the server's message.
 */
export async function streamAssistant(
  messages: AssistantMessage[],
  onDelta: (text: string) => void,
  signal?: AbortSignal,
  onTool?: (label: string) => void
): Promise<AssistantDone> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'text/event-stream' }
  const token = localStorage.getItem('auth_token')
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(buildApiUrl('/assistant/chat'), { method: 'POST', headers, body: JSON.stringify({ messages }), signal })
  const type = res.headers.get('Content-Type') || ''
  if (!type.includes('text/event-stream')) {
    let body: { message?: string; data?: AssistantDone } = {}
    try {
      body = await res.json()
    } catch {
      // not JSON
    }
    if (!res.ok) throw new Error(body.message || `请求失败（HTTP ${res.status}）`)
    return { sources: body.data?.sources || [], left: body.data?.left ?? 0 }
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let done: AssistantDone | null = null
  for (;;) {
    const { value, done: finished } = await reader.read()
    if (finished) break
    buffer += decoder.decode(value, { stream: true })
    let i: number
    while ((i = buffer.indexOf('\n\n')) >= 0) {
      const event = buffer.slice(0, i)
      buffer = buffer.slice(i + 2)
      const line = event.split('\n').find((l) => l.startsWith('data: '))
      if (!line) continue
      const msg = JSON.parse(line.slice(6)) as {
        delta?: string
        tool?: string
        done?: boolean
        error?: string
        sources?: AssistantSource[]
        left?: number
      }
      if (msg.error) throw new Error(msg.error)
      if (msg.tool) onTool?.(msg.tool)
      if (msg.delta) onDelta(msg.delta)
      if (msg.done) done = { sources: msg.sources || [], left: msg.left ?? 0 }
    }
  }
  if (!done) throw new Error('回答中断了，请重试（Stream interrupted）')
  return done
}
