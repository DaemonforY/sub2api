// Turns app-server notifications into the conversation the window shows, and keeps the
// approval queue. Kept free of Vue components so it can be unit-tested later.

import { reactive } from 'vue'
import { CodexClient, type ServerRequest } from './codex'

export type Entry =
  | { kind: 'user'; id: string; text: string }
  | { kind: 'agent'; id: string; text: string; done: boolean }
  | { kind: 'command'; id: string; command: string; status: string; output: string; exitCode: number | null }
  | { kind: 'files'; id: string; changes: { path: string; kind: string }[]; status: string }
  | { kind: 'notice'; id: string; text: string; tone: 'info' | 'error' }

export interface Approval {
  req: ServerRequest
  title: string
  detail: string
  reason: string | null
  /** Command/file approvals can be granted for the rest of the conversation. */
  allowSession: boolean
}

const APPROVAL_METHODS = new Set(['item/commandExecution/requestApproval', 'item/fileChange/requestApproval'])

export function createChat() {
  const client = new CodexClient()
  const state = reactive({
    ready: false,
    threadId: '' as string,
    cwd: '' as string,
    turnId: '' as string,
    busy: false,
    entries: [] as Entry[],
    approvals: [] as Approval[],
    fatal: '' as string,
    /** The gateway rejected the key (401): the user has to sign in again. */
    authFailed: false
  })

  let noticeSeq = 0
  const notice = (text: string, tone: 'info' | 'error' = 'info') =>
    state.entries.push({ kind: 'notice', id: `n${++noticeSeq}`, text, tone })

  const find = <K extends Entry['kind']>(kind: K, id: string) =>
    state.entries.find((e) => e.kind === kind && e.id === id) as Extract<Entry, { kind: K }> | undefined

  function upsertItem(item: any) {
    switch (item?.type) {
      case 'agentMessage': {
        const e = find('agent', item.id)
        if (e) e.text = item.text ?? e.text
        else state.entries.push({ kind: 'agent', id: item.id, text: item.text ?? '', done: false })
        break
      }
      case 'commandExecution': {
        const e = find('command', item.id)
        const next = {
          command: item.command ?? '',
          status: item.status ?? 'inProgress',
          output: item.aggregatedOutput ?? '',
          exitCode: item.exitCode ?? null
        }
        if (e) Object.assign(e, next)
        else state.entries.push({ kind: 'command', id: item.id, ...next })
        break
      }
      case 'fileChange': {
        const changes = (item.changes ?? []).map((c: any) => ({ path: c.path, kind: c.kind?.type ?? 'update' }))
        const e = find('files', item.id)
        if (e) Object.assign(e, { changes, status: item.status })
        else state.entries.push({ kind: 'files', id: item.id, changes, status: item.status })
        break
      }
    }
  }

  client.onNotification = (method, params) => {
    switch (method) {
      case 'turn/started':
        state.busy = true
        state.turnId = params?.turn?.id ?? ''
        break
      case 'turn/completed': {
        state.busy = false
        state.turnId = ''
        const turn = params?.turn
        if (state.authFailed) break
        if (turn?.status === 'failed' && turn.error?.message) notice(friendlyError(turn.error.message), 'error')
        if (turn?.status === 'interrupted') notice('已停止。')
        for (const e of state.entries) if (e.kind === 'agent') e.done = true
        break
      }
      case 'item/started':
      case 'item/completed':
        upsertItem(params?.item)
        if (method === 'item/completed' && params?.item?.type === 'agentMessage') {
          const e = find('agent', params.item.id)
          if (e) e.done = true
        }
        break
      case 'item/agentMessage/delta': {
        const e = find('agent', params?.itemId)
        if (e) e.text += params.delta ?? ''
        else state.entries.push({ kind: 'agent', id: params?.itemId, text: params?.delta ?? '', done: false })
        break
      }
      case 'item/commandExecution/outputDelta': {
        const e = find('command', params?.itemId)
        if (e) e.output += params.delta ?? ''
        break
      }
      case 'windowsSandbox/setupCompleted':
        sandboxSetup?.(Boolean(params?.success), params?.error ?? null)
        break
      case 'error':
        if (isAuthError(params?.error)) {
          onAuthFailed()
          break
        }
        if (params?.willRetry) notice('网络有点慢，正在自动重试…')
        else if (params?.error?.message) notice(friendlyError(params.error.message), 'error')
        break
    }
  }

  client.onServerRequest = (req) => {
    if (!APPROVAL_METHODS.has(req.method)) {
      // Not supported in the skeleton (user-input questions, MCP elicitation, permission profiles…).
      client.respondError(req.id, { code: -32601, message: `HiveGPT 桌面助手暂不支持 ${req.method}` })
      return
    }
    const p = req.params ?? {}
    if (req.method === 'item/commandExecution/requestApproval') {
      state.approvals.push({
        req,
        title: '它想运行一条命令',
        detail: p.command ?? '',
        reason: p.reason ?? null,
        allowSession: true
      })
    } else {
      const files = find('files', p.itemId)?.changes.map((c) => c.path) ?? []
      state.approvals.push({
        req,
        title: p.grantRoot ? `它想修改「${p.grantRoot}」里的文件` : '它想修改文件',
        detail: files.join('\n'),
        reason: p.reason ?? null,
        allowSession: true
      })
    }
  }

  // Windows: Codex's sandbox needs a one-time setup. "unelevated" avoids a UAC prompt; when it
  // fails, commands simply come back to the user for approval more often.
  let sandboxSetup: ((ok: boolean, error: string | null) => void) | null = null

  async function prepareWindowsSandbox(cwd: string) {
    const ready = await client.request<{ status: string }>('windowsSandbox/readiness').catch(() => null)
    if (!ready || ready.status === 'ready') return
    notice('第一次使用，正在准备安全运行环境…')
    const done = new Promise<{ ok: boolean; error: string | null }>((resolve) => {
      sandboxSetup = (ok, error) => resolve({ ok, error })
      setTimeout(() => resolve({ ok: false, error: '超时' }), 120_000)
    })
    const started = await client
      .request<{ started: boolean }>('windowsSandbox/setupStart', { mode: 'unelevated', cwd })
      .catch(() => null)
    const result = started?.started ? await done : { ok: false, error: '没有启动' }
    sandboxSetup = null
    if (!result.ok) notice(`安全运行环境没有准备好（${result.error ?? '未知原因'}），运行命令前会多问你几次。`)
  }

  client.onExit = () => {
    state.ready = false
    state.busy = false
    state.fatal = '引擎意外退出了，请点「重新连接」。'
  }

  // Codex retries a 401 about six times (~26 s); stop at the first one instead.
  function onAuthFailed() {
    if (state.authFailed) return
    state.authFailed = true
    state.busy = false
    if (state.threadId && state.turnId) {
      client.request('turn/interrupt', { threadId: state.threadId, turnId: state.turnId }).catch(() => {})
    }
  }

  async function connect(cwd: string) {
    state.fatal = ''
    state.authFailed = false
    state.cwd = cwd
    const init = await client.start()
    if (init?.platformFamily === 'windows') await prepareWindowsSandbox(cwd)
    const res = await client.request('thread/start', {
      cwd,
      approvalPolicy: 'on-request',
      sandbox: 'workspace-write'
    })
    state.threadId = res?.thread?.id ?? ''
    state.ready = true
  }

  async function send(text: string) {
    const t = text.trim()
    if (!t || !state.threadId || state.busy) return
    state.entries.push({ kind: 'user', id: `u${Date.now()}`, text: t })
    state.busy = true
    try {
      await client.request('turn/start', {
        threadId: state.threadId,
        input: [{ type: 'text', text: t, text_elements: [] }]
      })
    } catch (e) {
      state.busy = false
      notice(friendlyError(String((e as Error).message ?? e)), 'error')
    }
  }

  async function interrupt() {
    if (!state.threadId || !state.turnId) return
    await client.request('turn/interrupt', { threadId: state.threadId, turnId: state.turnId }).catch(() => {})
  }

  async function answer(approval: Approval, decision: 'accept' | 'acceptForSession' | 'decline') {
    state.approvals = state.approvals.filter((a) => a !== approval)
    await client.respond(approval.req.id, { decision })
  }

  return { state, connect, send, interrupt, answer }
}

/** 401 from the gateway, as Codex reports it in `codexErrorInfo` (or only in the message). */
export function isAuthError(error: any): boolean {
  const info = error?.codexErrorInfo
  if (info === 'unauthorized') return true
  if (info && typeof info === 'object') {
    for (const v of Object.values(info) as any[]) if (v?.httpStatusCode === 401) return true
  }
  return /\b401\b|unauthorized|invalid api key/i.test(String(error?.message ?? ''))
}

/** Gateway errors are already Chinese; translate the common transport ones. */
export function friendlyError(message: string): string {
  if (/401|unauthorized|invalid api key|无效/i.test(message)) return `Key 无效或已过期，请重新登录。（${message}）`
  if (/429|rate limit|额度/i.test(message)) return message
  if (/timeout|timed out|connection|network|dns/i.test(message)) return `连接 HiveGPT 失败，请检查网络后重试。（${message}）`
  return message
}
