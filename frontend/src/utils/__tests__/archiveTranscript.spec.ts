import { describe, expect, it } from 'vitest'
import {
  collapsedByDefault,
  normalizeItem,
  normalizeItems,
  normalizeResponse,
  prettyArgs,
  shortUA,
  type TranscriptBlock
} from '../archiveTranscript'

const pick = (b: TranscriptBlock[]) => b.map(({ role, kind, title, body }) => ({ role, kind, title, body }))

describe('archiveTranscript — OpenAI Responses', () => {
  it('maps message roles and text / image parts', () => {
    const blocks = normalizeItems([
      { type: 'message', role: 'developer', content: [{ type: 'input_text', text: 'be brief' }] },
      { type: 'message', role: 'system', content: 'sys' },
      { role: 'user', content: [{ type: 'input_text', text: 'hi' }, { type: 'input_image', image_url: 'data:image/png;base64,AAAA' }] },
      { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'hello' }] }
    ])
    expect(pick(blocks).map((b) => [b.role, b.kind, b.body.slice(0, 22)])).toEqual([
      ['developer', 'text', 'be brief'],
      ['system', 'text', 'sys'],
      ['user', 'text', 'hi'],
      ['user', 'image', 'data:image/png;base64'],
      ['assistant', 'text', 'hello']
    ])
    expect(blocks[3].imageSrc).toBe('data:image/png;base64,AAAA')
  })

  it('does not inline remote image URLs', () => {
    const [b] = normalizeItem({ role: 'user', content: [{ type: 'input_image', image_url: 'https://example.com/a.png' }] })
    expect(b.kind).toBe('image')
    expect(b.body).toBe('https://example.com/a.png')
    expect(b.imageSrc).toBeUndefined()
  })

  it('maps function / custom / local shell calls and their outputs', () => {
    const blocks = normalizeItems([
      { type: 'function_call', name: 'shell', call_id: 'c1', arguments: '{"cmd":["ls"]}' },
      { type: 'function_call_output', call_id: 'c1', output: 'a.txt' },
      { type: 'custom_tool_call', name: 'apply_patch', call_id: 'c2', input: '*** Begin Patch' },
      { type: 'custom_tool_call_output', call_id: 'c2', output: 'ok' },
      { type: 'local_shell_call', call_id: 'c3', action: { type: 'exec', command: ['git', 'status'] } },
      { type: 'local_shell_call_output', call_id: 'c3', output: 'clean' }
    ])
    expect(pick(blocks)).toEqual([
      { role: 'assistant', kind: 'tool_call', title: 'shell · c1', body: '{\n  "cmd": [\n    "ls"\n  ]\n}' },
      { role: 'tool', kind: 'tool_output', title: 'c1', body: 'a.txt' },
      { role: 'assistant', kind: 'tool_call', title: 'apply_patch · c2', body: '*** Begin Patch' },
      { role: 'tool', kind: 'tool_output', title: 'c2', body: 'ok' },
      { role: 'assistant', kind: 'tool_call', title: 'local_shell · c3', body: 'git status' },
      { role: 'tool', kind: 'tool_output', title: 'c3', body: 'clean' }
    ])
  })

  it('shows the reasoning summary and skips empty reasoning', () => {
    expect(pick(normalizeItem({ type: 'reasoning', summary: [{ type: 'summary_text', text: 'thinking about it' }] }))).toEqual([
      { role: 'assistant', kind: 'other', title: 'reasoning', body: 'thinking about it' }
    ])
    expect(normalizeItem({ type: 'reasoning', summary: [] })).toEqual([])
  })

  it('renders image_generation_call results inline', () => {
    const [b] = normalizeItem({ type: 'image_generation_call', result: 'QUJD', output_format: 'webp' }, 'assistant')
    expect(b.kind).toBe('image')
    expect(b.imageSrc).toBe('data:image/webp;base64,QUJD')
  })
})

describe('archiveTranscript — Chat Completions', () => {
  it('maps tool_calls and role tool', () => {
    const blocks = normalizeItems([
      { role: 'system', content: 'you are helpful' },
      { role: 'user', content: [{ type: 'text', text: 'weather?' }, { type: 'image_url', image_url: { url: 'data:image/jpeg;base64,/9j' } }] },
      { role: 'assistant', content: null, tool_calls: [{ id: 'call_1', type: 'function', function: { name: 'get_weather', arguments: '{"city":"Seoul"}' } }] },
      { role: 'tool', tool_call_id: 'call_1', content: 'sunny' }
    ])
    expect(pick(blocks).map((b) => [b.role, b.kind, b.title])).toEqual([
      ['system', 'text', ''],
      ['user', 'text', ''],
      ['user', 'image', 'image'],
      ['assistant', 'tool_call', 'get_weather · call_1'],
      ['tool', 'tool_output', 'call_1']
    ])
    expect(blocks[3].body).toBe('{\n  "city": "Seoul"\n}')
    expect(blocks[4].body).toBe('sunny')
  })
})

describe('archiveTranscript — Anthropic Messages', () => {
  it('maps text, tool_use, tool_result and base64 images', () => {
    const blocks = normalizeItems([
      { role: 'user', content: [{ type: 'text', text: 'look' }, { type: 'image', source: { type: 'base64', media_type: 'image/png', data: 'iVBO' } }] },
      { role: 'assistant', content: [{ type: 'thinking', thinking: 'hmm' }, { type: 'tool_use', id: 'tu_1', name: 'Read', input: { file_path: '/a' } }] },
      { role: 'user', content: [{ type: 'tool_result', tool_use_id: 'tu_1', content: [{ type: 'text', text: 'file body' }] }] }
    ])
    expect(pick(blocks).map((b) => [b.role, b.kind, b.title])).toEqual([
      ['user', 'text', ''],
      ['user', 'image', 'image'],
      ['assistant', 'other', 'thinking'],
      ['assistant', 'tool_call', 'Read · tu_1'],
      ['tool', 'tool_output', 'tu_1']
    ])
    expect(blocks[1].imageSrc).toBe('data:image/png;base64,iVBO')
    expect(blocks[3].body).toBe('{\n  "file_path": "/a"\n}')
    expect(blocks[4].body).toBe('file body')
  })

  it('maps bare content blocks in response items', () => {
    const blocks = normalizeResponse({
      text: 'done',
      items: [{ type: 'text', text: 'done' }, { type: 'tool_use', id: 't', name: 'Bash', input: { command: 'ls' } }]
    })
    expect(pick(blocks).map((b) => [b.role, b.kind, b.title])).toEqual([
      ['assistant', 'text', ''],
      ['assistant', 'tool_call', 'Bash · t']
    ])
  })
})

describe('archiveTranscript — Gemini', () => {
  it('maps parts, model role and function calls', () => {
    const blocks = normalizeItems([
      { role: 'user', parts: [{ text: 'draw' }, { inlineData: { mimeType: 'image/png', data: 'AAA' } }] },
      { role: 'model', parts: [{ functionCall: { name: 'search', args: { q: 'x' } } }] },
      { role: 'user', parts: [{ functionResponse: { name: 'search', response: { r: 1 } } }] }
    ])
    expect(pick(blocks).map((b) => [b.role, b.kind, b.title])).toEqual([
      ['user', 'text', ''],
      ['user', 'image', 'image'],
      ['assistant', 'tool_call', 'search'],
      ['tool', 'tool_output', 'search']
    ])
  })
})

describe('archiveTranscript — images prompt and unknown shapes', () => {
  it('maps a prompt object and a bare string', () => {
    expect(pick(normalizeItems([{ prompt: 'a cat' }, 'a dog']))).toEqual([
      { role: 'user', kind: 'text', title: 'prompt', body: 'a cat' },
      { role: 'user', kind: 'text', title: '', body: 'a dog' }
    ])
  })

  it('falls back to pretty JSON', () => {
    const [b] = normalizeItem({ weird: true })
    expect(b.kind).toBe('other')
    expect(b.body).toBe('{\n  "weird": true\n}')
    const [c] = normalizeItem({ type: 'brand_new_item', x: 1 })
    expect(c.kind).toBe('other')
    expect(c.title).toBe('brand_new_item')
    expect(c.body).toContain('"x": 1')
  })

  it('ignores null and empty input', () => {
    expect(normalizeItems(null)).toEqual([])
    expect(normalizeItems([null, ''])).toEqual([])
  })
})

describe('archiveTranscript — response and helpers', () => {
  it('uses text and tool_calls when there are no raw items', () => {
    expect(pick(normalizeResponse({ text: 'hi', tool_calls: [{ name: 'f', call_id: 'c', arguments: '{"a":1}' }] }))).toEqual([
      { role: 'assistant', kind: 'text', title: '', body: 'hi' },
      { role: 'assistant', kind: 'tool_call', title: 'f · c', body: '{\n  "a": 1\n}' }
    ])
    expect(normalizeResponse(null)).toEqual([])
  })

  it('collapses tool blocks and long text', () => {
    expect(collapsedByDefault({ role: 'assistant', kind: 'tool_call', title: '', body: 'x' })).toBe(true)
    expect(collapsedByDefault({ role: 'tool', kind: 'tool_output', title: '', body: 'x' })).toBe(true)
    expect(collapsedByDefault({ role: 'user', kind: 'text', title: '', body: 'a\n'.repeat(12) })).toBe(true)
    expect(collapsedByDefault({ role: 'user', kind: 'text', title: '', body: 'a\nb' })).toBe(false)
  })

  it('pretty-prints JSON args and leaves plain strings alone', () => {
    expect(prettyArgs('{"a":1}')).toBe('{\n  "a": 1\n}')
    expect(prettyArgs('{broken')).toBe('{broken')
    expect(prettyArgs({ a: 1 })).toBe('{\n  "a": 1\n}')
  })

  it('shortens user agents', () => {
    expect(shortUA('codex_cli_rs/0.40.0 (Mac OS 15.0; arm64)')).toBe('codex_cli_rs/0.40.0')
    expect(shortUA('')).toBe('–')
  })
})
