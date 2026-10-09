/** Admin API for the ops assistant (运维助手): ask why calls fail; it looks things up read-only. */
import { apiClient } from '../client'
import { buildApiUrl } from '../url'
import type { AgentRun, AssistantKeyOption } from './assistant'

export type { AgentRun, AssistantKeyOption }

export interface OpsAgentSettings {
  enabled: boolean
  model: string
  /** One of the signed-in admin's own GPT keys; only its ID is stored. */
  key_id: number
  key_owner?: number
  key_name?: string
  key_problem?: string
  /** 告警自动分析: fired alerts start a run and the answer is mailed to the alert recipients. */
  auto_alert: boolean
  /** An upstream Sub2API site (one of the API-key accounts points at it), read with its Admin API Key. */
  upstream_name: string
  upstream_url: string
  /** Write-only: a new value replaces the stored key; empty keeps it. */
  upstream_key?: string
  upstream_key_set: boolean
  clear_upstream_key?: boolean
}

export interface OpsAgentMessage {
  role: 'user' | 'assistant'
  content: string
}

export async function getSettings(): Promise<OpsAgentSettings> {
  const { data } = await apiClient.get('/admin/ops-agent/settings')
  return data
}

export async function saveSettings(settings: OpsAgentSettings): Promise<OpsAgentSettings> {
  const { data } = await apiClient.put('/admin/ops-agent/settings', settings)
  return data
}

export async function listKeys(): Promise<AssistantKeyOption[]> {
  const { data } = await apiClient.get('/admin/ops-agent/keys')
  return data
}

export async function listRuns(page = 1): Promise<{ items: AgentRun[]; total: number }> {
  const { data } = await apiClient.get('/admin/ops-agent/runs', { params: { page } })
  return data
}

/** Reads the upstream site's last 15 minutes with the stored address and key. */
export async function testUpstream(): Promise<Record<string, number | string>> {
  const { data } = await apiClient.post('/admin/ops-agent/upstream/test')
  return data
}

/**
 * Asks the last question of messages; the answer streams to onDelta and onTool gets a label
 * (正在汇总报错…) as each lookup starts. Errors carry the server's message.
 */
export async function streamOpsAgent(
  messages: OpsAgentMessage[],
  onDelta: (text: string) => void,
  onTool: (label: string) => void,
  signal?: AbortSignal
): Promise<{ model: string }> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'text/event-stream' }
  const token = localStorage.getItem('auth_token')
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(buildApiUrl('/admin/ops-agent/chat'), { method: 'POST', headers, body: JSON.stringify({ messages }), signal })
  const type = res.headers.get('Content-Type') || ''
  if (!type.includes('text/event-stream')) {
    let body: { message?: string; data?: { model?: string } } = {}
    try {
      body = await res.json()
    } catch {
      // not JSON
    }
    if (!res.ok) throw new Error(body.message || `请求失败（HTTP ${res.status}）`)
    return { model: body.data?.model || '' }
  }
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let done: { model: string } | null = null
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
      const msg = JSON.parse(line.slice(6)) as { delta?: string; tool?: string; done?: boolean; error?: string; model?: string }
      if (msg.error) throw new Error(msg.error)
      if (msg.tool) onTool(msg.tool)
      if (msg.delta) onDelta(msg.delta)
      if (msg.done) done = { model: msg.model || '' }
    }
  }
  if (!done) throw new Error('回答中断了，请重试（Stream interrupted）')
  return done
}
