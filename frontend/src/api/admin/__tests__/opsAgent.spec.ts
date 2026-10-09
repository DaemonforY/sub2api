import { afterEach, describe, expect, it, vi } from 'vitest'
import { streamOpsAgent } from '../opsAgent'

function sseResponse(events: object[], split = 7): Response {
  const text = events.map((e) => `data: ${JSON.stringify(e)}\n\n`).join('')
  const bytes = new TextEncoder().encode(text)
  const body = new ReadableStream({
    start(controller) {
      // Chunks that cut events (and multi-byte characters) in half, as the network does.
      for (let i = 0; i < bytes.length; i += split) controller.enqueue(bytes.slice(i, i + split))
      controller.close()
    }
  })
  return new Response(body, { status: 200, headers: { 'Content-Type': 'text/event-stream' } })
}

afterEach(() => vi.unstubAllGlobals())

describe('streamOpsAgent', () => {
  it('streams lookups and answer text, then resolves with the model', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      sseResponse([{ tool: '正在汇总报错' }, { delta: '上游 #9 ' }, { tool: '正在汇总上游站的报错' }, { delta: '过载。' }, { done: true, model: 'gpt-5.6-terra' }])
    )
    vi.stubGlobal('fetch', fetchMock)
    const tools: string[] = []
    let text = ''
    const out = await streamOpsAgent([{ role: 'user', content: '为什么报错' }], (d) => (text += d), (l) => tools.push(l))
    expect(out).toEqual({ model: 'gpt-5.6-terra' })
    expect(text).toBe('上游 #9 过载。')
    expect(tools).toEqual(['正在汇总报错', '正在汇总上游站的报错'])
    const [url, init] = fetchMock.mock.calls[0]
    expect(String(url)).toContain('/admin/ops-agent/chat')
    expect(JSON.parse(init.body)).toEqual({ messages: [{ role: 'user', content: '为什么报错' }] })
  })

  it('throws the server message for JSON errors and stream errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 503, message: '运维助手还没开启' }), {
      status: 503, headers: { 'Content-Type': 'application/json' }
    })))
    await expect(streamOpsAgent([{ role: 'user', content: 'x' }], () => {}, () => {})).rejects.toThrow('运维助手还没开启')

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([{ delta: '一半' }, { error: '连接模型服务超时' }])))
    await expect(streamOpsAgent([{ role: 'user', content: 'x' }], () => {}, () => {})).rejects.toThrow('连接模型服务超时')

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([{ delta: '没有结尾' }])))
    await expect(streamOpsAgent([{ role: 'user', content: 'x' }], () => {}, () => {})).rejects.toThrow('回答中断了')
  })
})
