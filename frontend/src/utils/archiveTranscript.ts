/**
 * Turns the raw items of an archived conversation turn (对话归档) into display blocks.
 *
 * Archived turns keep input items exactly as the client sent them, so one
 * transcript can mix several dialects:
 * - OpenAI Responses items (message / function_call / function_call_output /
 *   custom_tool_call(_output) / local_shell_call(_output) / reasoning / image_generation_call …)
 * - Chat Completions messages (incl. tool_calls and role "tool")
 * - Anthropic Messages content blocks (text / tool_use / tool_result / image / thinking)
 * - Gemini contents (parts: text / inlineData / functionCall / functionResponse)
 * - image prompts ({ prompt } or a bare string)
 * Anything we don't recognise is shown as pretty JSON, never dropped.
 */

export type BlockKind = 'text' | 'tool_call' | 'tool_output' | 'image' | 'other'

export interface TranscriptBlock {
  /** user | assistant | system | developer | tool */
  role: string
  kind: BlockKind
  /** tool name, call id, part type … (may be empty) */
  title: string
  /** text, arguments, output or pretty JSON; for images the URL / a short description */
  body: string
  /** set only for inline data: image URLs, safe to render directly */
  imageSrc?: string
}

export interface ResponseLike {
  status?: string
  text?: string
  tool_calls?: Array<{ name?: string; call_id?: string; arguments?: unknown }> | null
  items?: unknown[] | null
}

type Obj = Record<string, unknown>

const isObj = (v: unknown): v is Obj => typeof v === 'object' && v !== null && !Array.isArray(v)
const str = (v: unknown): string => (typeof v === 'string' ? v : '')

export function prettyJSON(v: unknown): string {
  if (v === undefined) return ''
  try {
    return JSON.stringify(v, null, 2) ?? String(v)
  } catch {
    return String(v)
  }
}

/** Tool arguments usually arrive as a JSON string: pretty-print it when it parses. */
export function prettyArgs(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') {
    const s = v.trim()
    if ((s.startsWith('{') && s.endsWith('}')) || (s.startsWith('[') && s.endsWith(']'))) {
      try {
        return JSON.stringify(JSON.parse(s), null, 2)
      } catch {
        return v
      }
    }
    return v
  }
  return prettyJSON(v)
}

function normRole(role: unknown, fallback: string): string {
  const r = str(role).toLowerCase()
  if (!r) return fallback
  if (r === 'model') return 'assistant'
  if (r === 'function') return 'tool'
  return r
}

function imageBlock(role: string, url: string, title = 'image'): TranscriptBlock {
  const isData = url.startsWith('data:image/')
  return {
    role,
    kind: 'image',
    title,
    // a data URL is shown as the picture itself; the body only says what it is
    body: isData ? url.slice(0, url.indexOf(',') > 0 ? url.indexOf(',') : 40) : url,
    imageSrc: isData ? url : undefined
  }
}

/** Text of a tool output, which may be a string, a list of parts or anything else. */
function outputText(v: unknown, role: string, extra: TranscriptBlock[]): string {
  if (v == null) return ''
  if (typeof v === 'string') return v
  if (Array.isArray(v)) {
    const texts: string[] = []
    for (const p of v) {
      if (typeof p === 'string') {
        texts.push(p)
      } else if (isObj(p) && typeof p.text === 'string') {
        texts.push(p.text)
      } else if (isObj(p) && (p.type === 'image' || p.type === 'input_image' || p.type === 'image_url')) {
        extra.push(...partBlocks(p, role))
      } else {
        texts.push(prettyJSON(p))
      }
    }
    return texts.join('\n')
  }
  if (isObj(v) && typeof v.output === 'string') return v.output
  return prettyJSON(v)
}

/** One content part (Responses, Chat, Anthropic or Gemini) → blocks. */
function partBlocks(p: unknown, role: string): TranscriptBlock[] {
  if (typeof p === 'string') return p ? [{ role, kind: 'text', title: '', body: p }] : []
  if (!isObj(p)) return [{ role, kind: 'other', title: '', body: prettyJSON(p) }]
  const type = str(p.type)

  switch (type) {
    case 'input_text':
    case 'output_text':
    case 'text':
    case 'summary_text':
      return [{ role, kind: 'text', title: '', body: str(p.text) }]
    case 'refusal':
      return [{ role, kind: 'text', title: 'refusal', body: str(p.refusal) || str(p.text) }]
    case 'input_image': {
      const url = str(p.image_url) || (isObj(p.image_url) ? str(p.image_url.url) : '')
      if (url) return [imageBlock(role, url)]
      return [{ role, kind: 'image', title: 'image', body: str(p.file_id) || prettyJSON(p) }]
    }
    case 'image_url': {
      const url = isObj(p.image_url) ? str(p.image_url.url) : str(p.image_url)
      return [imageBlock(role, url)]
    }
    case 'image': {
      // Anthropic: { source: { type: base64, media_type, data } | { type: url, url } }
      const src = isObj(p.source) ? p.source : {}
      if (str(src.type) === 'base64' && str(src.data)) {
        return [imageBlock(role, `data:${str(src.media_type) || 'image/png'};base64,${str(src.data)}`)]
      }
      if (str(src.url)) return [imageBlock(role, str(src.url))]
      return [{ role, kind: 'image', title: 'image', body: prettyJSON(src) }]
    }
    case 'tool_use':
    case 'server_tool_use':
      return [{ role: 'assistant', kind: 'tool_call', title: [str(p.name), str(p.id)].filter(Boolean).join(' · '), body: prettyArgs(p.input) }]
    case 'tool_result':
    case 'web_search_tool_result': {
      const extra: TranscriptBlock[] = []
      const body = outputText(p.content, 'tool', extra)
      const title = [str(p.tool_use_id), p.is_error === true ? 'error' : ''].filter(Boolean).join(' · ')
      return [{ role: 'tool', kind: 'tool_output', title, body }, ...extra]
    }
    case 'thinking':
      return [{ role, kind: 'other', title: 'thinking', body: str(p.thinking) }]
    case 'redacted_thinking':
      return [{ role, kind: 'other', title: 'redacted_thinking', body: '' }]
    case 'input_file':
    case 'document':
      return [{ role, kind: 'other', title: type, body: str(p.filename) || str(p.file_id) || prettyJSON(isObj(p.source) ? { ...p.source, data: undefined } : p) }]
  }

  // Gemini parts have no type
  if (!type) {
    if (typeof p.text === 'string') {
      return [{ role, kind: p.thought === true ? 'other' : 'text', title: p.thought === true ? 'thinking' : '', body: p.text }]
    }
    const inline = (isObj(p.inlineData) ? p.inlineData : isObj(p.inline_data) ? p.inline_data : null) as Obj | null
    if (inline) {
      const mime = str(inline.mimeType) || str(inline.mime_type) || 'image/png'
      if (mime.startsWith('image/') && str(inline.data)) return [imageBlock(role, `data:${mime};base64,${str(inline.data)}`)]
      return [{ role, kind: 'other', title: mime, body: '' }]
    }
    const fc = (isObj(p.functionCall) ? p.functionCall : isObj(p.function_call) ? p.function_call : null) as Obj | null
    if (fc) return [{ role: 'assistant', kind: 'tool_call', title: str(fc.name), body: prettyArgs(fc.args) }]
    const fr = (isObj(p.functionResponse) ? p.functionResponse : isObj(p.function_response) ? p.function_response : null) as Obj | null
    if (fr) return [{ role: 'tool', kind: 'tool_output', title: str(fr.name), body: prettyArgs(fr.response) }]
    const fd = (isObj(p.fileData) ? p.fileData : isObj(p.file_data) ? p.file_data : null) as Obj | null
    if (fd) return [{ role, kind: 'other', title: str(fd.mimeType) || str(fd.mime_type) || 'file', body: str(fd.fileUri) || str(fd.file_uri) }]
  }

  return [{ role, kind: 'other', title: type, body: prettyJSON(p) }]
}

function contentBlocks(content: unknown, role: string): TranscriptBlock[] {
  if (content == null) return []
  if (typeof content === 'string') return content ? [{ role, kind: 'text', title: '', body: content }] : []
  if (Array.isArray(content)) return content.flatMap((p) => partBlocks(p, role))
  return partBlocks(content, role)
}

/** Chat Completions tool_calls on an assistant message. */
function chatToolCalls(calls: unknown): TranscriptBlock[] {
  if (!Array.isArray(calls)) return []
  return calls.map((c) => {
    const o = isObj(c) ? c : {}
    const fn = isObj(o.function) ? o.function : isObj(o.custom) ? o.custom : {}
    return {
      role: 'assistant',
      kind: 'tool_call' as const,
      title: [str(fn.name) || str(o.name), str(o.id)].filter(Boolean).join(' · '),
      body: prettyArgs(fn.arguments ?? fn.input ?? o.arguments)
    }
  })
}

function shellCommand(action: unknown): string {
  if (!isObj(action)) return prettyJSON(action)
  const cmd = action.command
  if (Array.isArray(cmd)) return cmd.map(String).join(' ')
  if (typeof cmd === 'string') return cmd
  return prettyJSON(action)
}

/** One input/output item → blocks. `defaultRole` applies to items without a role. */
export function normalizeItem(item: unknown, defaultRole = 'user'): TranscriptBlock[] {
  if (typeof item === 'string') return item ? [{ role: defaultRole, kind: 'text', title: '', body: item }] : []
  if (!isObj(item)) return item == null ? [] : [{ role: defaultRole, kind: 'other', title: '', body: prettyJSON(item) }]

  const type = str(item.type)
  switch (type) {
    case 'function_call':
      return [{ role: 'assistant', kind: 'tool_call', title: [str(item.name), str(item.call_id)].filter(Boolean).join(' · '), body: prettyArgs(item.arguments) }]
    case 'custom_tool_call':
      return [{ role: 'assistant', kind: 'tool_call', title: [str(item.name), str(item.call_id)].filter(Boolean).join(' · '), body: prettyArgs(item.input) }]
    case 'local_shell_call':
      return [{ role: 'assistant', kind: 'tool_call', title: ['local_shell', str(item.call_id)].filter(Boolean).join(' · '), body: shellCommand(item.action) }]
    case 'function_call_output':
    case 'custom_tool_call_output':
    case 'local_shell_call_output': {
      const extra: TranscriptBlock[] = []
      const body = outputText(item.output, 'tool', extra)
      return [{ role: 'tool', kind: 'tool_output', title: str(item.call_id) || str(item.id), body }, ...extra]
    }
    case 'reasoning': {
      const summary = Array.isArray(item.summary)
        ? item.summary.map((s) => (isObj(s) ? str(s.text) : str(s))).filter(Boolean).join('\n\n')
        : ''
      const content = Array.isArray(item.content)
        ? item.content.map((s) => (isObj(s) ? str(s.text) : str(s))).filter(Boolean).join('\n\n')
        : ''
      const body = summary || content
      return body ? [{ role: 'assistant', kind: 'other', title: 'reasoning', body }] : []
    }
    case 'image_generation_call': {
      const result = str(item.result)
      if (result) {
        const src = result.startsWith('data:') ? result : `data:image/${str(item.output_format) || 'png'};base64,${result}`
        return [imageBlock('assistant', src, 'image_generation_call')]
      }
      const rest = { ...item }
      delete rest.result
      return [{ role: 'assistant', kind: 'tool_call', title: 'image_generation_call', body: prettyJSON(rest) }]
    }
    case 'web_search_call':
    case 'file_search_call':
    case 'computer_call':
    case 'code_interpreter_call':
    case 'mcp_call':
    case 'tool_search_call':
      return [{ role: 'assistant', kind: 'tool_call', title: [type, str(item.name)].filter(Boolean).join(' · '), body: prettyJSON(item) }]
    case 'computer_call_output':
    case 'mcp_call_output':
      return [{ role: 'tool', kind: 'tool_output', title: str(item.call_id), body: prettyJSON(item.output ?? item) }]
  }

  // A message: Responses (type "message" or bare {role, content}), Chat, Anthropic or Gemini.
  if (type === 'message' || (!type && ('role' in item || 'parts' in item || 'content' in item))) {
    const roleRaw = str(item.role)
    // Chat role "tool" (and legacy "function"): the content is the tool's output.
    if (roleRaw === 'tool' || roleRaw === 'function') {
      const extra: TranscriptBlock[] = []
      const body = outputText(item.content, 'tool', extra)
      return [{ role: 'tool', kind: 'tool_output', title: str(item.tool_call_id) || str(item.name), body }, ...extra]
    }
    const role = normRole(roleRaw, defaultRole)
    const blocks: TranscriptBlock[] = []
    if (Array.isArray(item.parts)) blocks.push(...item.parts.flatMap((p) => partBlocks(p, role)))
    else blocks.push(...contentBlocks(item.content, role))
    if (typeof item.reasoning_content === 'string' && item.reasoning_content) {
      blocks.unshift({ role, kind: 'other', title: 'reasoning', body: item.reasoning_content })
    }
    blocks.push(...chatToolCalls(item.tool_calls))
    if (isObj(item.function_call)) blocks.push(...chatToolCalls([{ function: item.function_call }]))
    return blocks
  }

  // Chat choice wrapper { message: {...} }
  if (isObj(item.message)) return normalizeItem(item.message, defaultRole)

  // Image prompt
  if (typeof item.prompt === 'string' && !type) return [{ role: 'user', kind: 'text', title: 'prompt', body: item.prompt }]

  // A bare content part at the top level (Anthropic output blocks, Gemini parts)
  if (type || 'text' in item || 'inlineData' in item || 'functionCall' in item || 'functionResponse' in item) {
    return partBlocks(item, defaultRole)
  }

  return [{ role: defaultRole, kind: 'other', title: '', body: prettyJSON(item) }]
}

/** The new input items of a turn → blocks. */
export function normalizeItems(items: unknown[] | null | undefined): TranscriptBlock[] {
  if (!Array.isArray(items)) return []
  return items.flatMap((it) => normalizeItem(it, 'user'))
}

/**
 * The model's response → blocks. Raw output items are preferred (they keep the
 * order of text, tool calls and images); the flat text / tool_calls fill in
 * whatever the items don't carry.
 */
export function normalizeResponse(resp: ResponseLike | null | undefined): TranscriptBlock[] {
  if (!resp) return []
  const blocks = Array.isArray(resp.items) ? resp.items.flatMap((it) => normalizeItem(it, 'assistant')) : []
  const text = str(resp.text)
  if (text && !blocks.some((b) => b.kind === 'text')) {
    blocks.unshift({ role: 'assistant', kind: 'text', title: '', body: text })
  }
  if (Array.isArray(resp.tool_calls) && !blocks.some((b) => b.kind === 'tool_call')) {
    for (const c of resp.tool_calls) {
      blocks.push({ role: 'assistant', kind: 'tool_call', title: [str(c?.name), str(c?.call_id)].filter(Boolean).join(' · '), body: prettyArgs(c?.arguments) })
    }
  }
  return blocks
}

export const COLLAPSE_LINES = 12

export function lineCount(s: string): number {
  if (!s) return 0
  return s.split('\n').length
}

/** Tool calls/outputs, reasoning and anything longer than 12 lines start collapsed. */
export function collapsedByDefault(b: TranscriptBlock): boolean {
  if (b.kind === 'tool_call' || b.kind === 'tool_output') return true
  if (b.kind === 'image') return false
  return lineCount(b.body) > COLLAPSE_LINES
}

/** "codex_cli_rs/0.40 (Mac OS…)" → "codex_cli_rs/0.40" */
export function shortUA(ua: string | null | undefined, max = 28): string {
  const s = (ua || '').trim()
  if (!s) return '–'
  const m = s.match(/^[^/()]*\/[^\s;()]*/)
  const head = (m ? m[0] : s).trim()
  return head.length > max ? `${head.slice(0, max - 1)}…` : head
}
