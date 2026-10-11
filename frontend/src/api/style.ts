/** 测测你的写作风格 (/style): read a 公众号 article, quote and run the AI portrait on the reader's key. */
import { apiClient } from './client'
import { buildApiUrl } from './url'

export interface StyleArticle {
  title: string
  author?: string
  text: string
}

export interface StyleResult {
  archetype: string
  portrait: string
  tags: string[]
  voice: string
  rhythm: string
  openings: string[]
  endings: string[]
  catchphrases: string[]
  strengths: string[]
  suggestions: string[]
  avoid: string[]
  samples: string[]
  prompt: string
}

export interface StyleAnalysis {
  result: StyleResult
  model: string
  prompt_tokens: number
  completion_tokens: number
  /** USD on the key, priced the way usage is billed. */
  cost?: number
}

/** Text of a mp.weixin.qq.com article (no login needed). */
export async function fetchArticle(url: string): Promise<StyleArticle> {
  const { data } = await apiClient.post('/style/fetch', { url })
  return data
}

/** Expected cost in USD of analysing this many characters on the key; null when unknown. */
export async function quote(keyID: number, chars: number): Promise<number | null> {
  const { data } = await apiClient.get('/style/quote', { params: { key_id: keyID, chars } })
  return typeof data?.price === 'number' ? data.price : null
}

/** Runs the analysis; onProgress gets the characters the AI has written so far. Errors carry the server's message. */
export async function analyze(
  keyID: number,
  articles: StyleArticle[],
  onProgress: (chars: number) => void,
  signal?: AbortSignal
): Promise<StyleAnalysis> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', Accept: 'text/event-stream' }
  const token = localStorage.getItem('auth_token')
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(buildApiUrl('/style/analyze'), {
    method: 'POST',
    headers,
    body: JSON.stringify({ key_id: keyID, articles: articles.map((a) => ({ title: a.title, text: a.text })) }),
    signal
  })
  if (!(res.headers.get('Content-Type') || '').includes('text/event-stream')) {
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
      if (!line) continue
      const msg = JSON.parse(line.slice(6)) as { progress?: number; done?: boolean; error?: string; analysis?: StyleAnalysis }
      if (msg.error) throw new Error(msg.error)
      if (typeof msg.progress === 'number') onProgress(msg.progress)
      if (msg.done && msg.analysis) return msg.analysis
    }
  }
  throw new Error('分析中断了，请重试（Stream interrupted）')
}
