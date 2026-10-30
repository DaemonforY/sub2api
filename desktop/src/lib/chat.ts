// Turns app-server notifications into the conversation the window shows, keeps the approval queue,
// the list of past conversations (thread/list), and undo (folder snapshot + thread/rollback).
// Kept free of Vue components so it can be unit-tested later.

import { reactive } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import { CodexClient, type ServerRequest } from './codex'
import { track } from './track'
import { fetchUsage, type UsageView } from './usage'

export type Entry =
  | { kind: 'user'; id: string; text: string }
  | { kind: 'agent'; id: string; text: string; done: boolean }
  | { kind: 'command'; id: string; command: string; status: string; output: string; exitCode: number | null }
  | { kind: 'files'; id: string; changes: { path: string; kind: string }[]; status: string }
  | {
      kind: 'notice'
      id: string
      text: string
      tone: 'info' | 'error'
      /** 'recharge' shows a 充值 button; 'open' a button opening `path`. */
      action?: 'recharge' | 'open'
      path?: string
    }

export interface Approval {
  req: ServerRequest
  title: string
  detail: string
  reason: string | null
  /** Command/file approvals can be granted for the rest of the conversation. */
  allowSession: boolean
}

export interface ThreadSummary {
  id: string
  title: string
  cwd: string
  /** Unix ms */
  updatedAt: number
}

/** What undoing the last turn will do, so the confirm dialog can say it plainly. */
export interface UndoPreview {
  /** The folder will be put back as it was before the turn. */
  files: boolean
  /** Why it can't be (folder too big, the turn predates this version…). */
  reason: string
}

interface SnapshotMeta {
  id: string
  cwd: string
  turn_index: number
  skipped: string | null
}

interface Restored {
  restored: number
  moved_aside: number
  trash: string
}

/** Per-turn bookkeeping for analytics and the cost line. */
interface TurnInfo {
  template: string
  startedAt: number
  totalBefore: number | null
}

const APPROVAL_METHODS = new Set(['item/commandExecution/requestApproval', 'item/fileChange/requestApproval'])

/** Conversation entries for one stored item (thread/read, thread/rollback) or a live one. */
export function itemToEntry(item: any): Entry | null {
  switch (item?.type) {
    case 'userMessage': {
      const text = (item.content ?? [])
        .filter((c: any) => c?.type === 'text')
        .map((c: any) => c.text ?? '')
        .join('\n')
      return { kind: 'user', id: item.id, text }
    }
    case 'agentMessage':
      return { kind: 'agent', id: item.id, text: item.text ?? '', done: true }
    case 'commandExecution':
      return {
        kind: 'command',
        id: item.id,
        command: item.command ?? '',
        status: item.status ?? 'completed',
        output: item.aggregatedOutput ?? '',
        exitCode: item.exitCode ?? null
      }
    case 'fileChange':
      return {
        kind: 'files',
        id: item.id,
        changes: (item.changes ?? []).map((c: any) => ({ path: c.path, kind: c.kind?.type ?? 'update' })),
        status: item.status ?? 'completed'
      }
  }
  return null
}

export function entriesFromTurns(turns: any[]): Entry[] {
  const out: Entry[] = []
  for (const turn of turns ?? []) {
    for (const item of turn?.items ?? []) {
      const e = itemToEntry(item)
      if (e) out.push(e)
    }
    if (turn?.status === 'failed' && turn.error?.message) {
      out.push({ kind: 'notice', id: `f${turn.id}`, text: friendlyError(turn.error.message), tone: 'error' })
    } else if (turn?.status === 'interrupted') {
      out.push({ kind: 'notice', id: `i${turn.id}`, text: '已停止。', tone: 'info' })
    }
  }
  return out
}

export function toThreadSummary(t: any): ThreadSummary {
  const title = String(t?.name || t?.preview || '新对话').replace(/\s+/g, ' ').trim()
  const updated = Number(t?.updatedAt ?? t?.createdAt ?? 0)
  return {
    id: String(t?.id ?? ''),
    title: title.length > 40 ? title.slice(0, 40) + '…' : title,
    cwd: String(t?.cwd ?? ''),
    updatedAt: updated < 1e12 ? updated * 1000 : updated
  }
}

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
    authFailed: false,
    /** Turns in the current conversation (undo rolls back the last one). */
    turnCount: 0,
    threads: [] as ThreadSummary[],
    usage: null as UsageView | null,
    /** Opening a conversation or undoing: the composer waits. */
    working: false
  })

  let noticeSeq = 0
  const notice = (text: string, tone: 'info' | 'error' = 'info', extra: { action?: 'recharge' | 'open'; path?: string } = {}) =>
    state.entries.push({ kind: 'notice', id: `n${++noticeSeq}`, text, tone, ...extra })

  const errorNotice = (message: string) => {
    const text = friendlyError(message)
    notice(text, 'error', isQuotaError(message) ? { action: 'recharge' } : {})
  }

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

  let currentTurn: TurnInfo | null = null

  function onTurnCompleted(turn: any) {
    const info = currentTurn
    currentTurn = null
    if (!info) return
    const secs = Math.round((Date.now() - info.startedAt) / 1000)
    if (turn?.status === 'completed') {
      track('desktop_task_done', { template: info.template, secs })
    } else {
      const reason = state.authFailed
        ? 'auth'
        : turn?.status === 'interrupted'
          ? 'interrupted'
          : isQuotaError(turn?.error?.message ?? '')
            ? 'quota'
            : 'error'
      track('desktop_task_failed', { template: info.template, secs, reason })
    }
    // Usage is written a moment after the response ends; read it back, then show what the turn cost.
    setTimeout(async () => {
      await refreshUsage()
      const after = state.usage?.total ?? null
      if (info.totalBefore !== null && after !== null && after - info.totalBefore >= 0.001) {
        notice(`这一轮约花费 $${(after - info.totalBefore).toFixed(3)}`)
      }
    }, 2500)
    refreshThreads()
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
        if (!state.authFailed) {
          if (turn?.status === 'failed' && turn.error?.message) errorNotice(turn.error.message)
          if (turn?.status === 'interrupted') notice('已停止。')
        }
        for (const e of state.entries) if (e.kind === 'agent') e.done = true
        onTurnCompleted(turn)
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
      case 'thread/name/updated':
        refreshThreads()
        break
      case 'windowsSandbox/setupCompleted':
        sandboxSetup?.(Boolean(params?.success), params?.error ?? null)
        break
      case 'error':
        if (isAuthError(params?.error)) {
          onAuthFailed()
          break
        }
        if (params?.willRetry) notice('网络有点慢，正在自动重试…')
        else if (params?.error?.message) errorNotice(params.error.message)
        break
    }
  }

  client.onServerRequest = (req) => {
    if (!APPROVAL_METHODS.has(req.method)) {
      // Not supported yet (user-input questions, MCP elicitation, permission profiles…).
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

  const threadParams = { approvalPolicy: 'on-request', sandbox: 'workspace-write' }

  async function connect(cwd: string) {
    state.fatal = ''
    state.authFailed = false
    state.cwd = cwd
    const init = await client.start()
    if (init?.platformFamily === 'windows') await prepareWindowsSandbox(cwd)
    await newThread(cwd)
    state.ready = true
    refreshThreads()
    refreshUsage()
  }

  /** A fresh, empty conversation in `cwd` (the current folder by default). */
  async function newThread(cwd: string = state.cwd) {
    const res = await client.request('thread/start', { cwd, ...threadParams })
    state.threadId = res?.thread?.id ?? ''
    state.cwd = res?.thread?.cwd || cwd
    state.entries = []
    state.approvals = []
    state.turnCount = 0
  }

  async function refreshThreads() {
    const res = await client
      .request('thread/list', { limit: 50, sortKey: 'updated_at', archived: false })
      .catch(() => null)
    if (!res) return
    state.threads = (res.data ?? [])
      .filter((t: any) => t?.id && (t.preview || t.name))
      .map(toThreadSummary)
  }

  async function refreshUsage() {
    const u = await fetchUsage()
    if (u) state.usage = u
  }

  /** Loads a past conversation (any folder) and continues it there. */
  async function openThread(id: string) {
    if (state.busy || state.working || id === state.threadId) return
    state.working = true
    try {
      const resumed = await client.request('thread/resume', { threadId: id, ...threadParams })
      const read = await client.request('thread/read', { threadId: id, includeTurns: true })
      const thread = read?.thread ?? resumed?.thread
      state.threadId = id
      state.cwd = thread?.cwd || resumed?.cwd || state.cwd
      state.entries = entriesFromTurns(thread?.turns ?? [])
      state.approvals = []
      state.turnCount = (thread?.turns ?? []).length
      track('desktop_history_open', { turns: state.turnCount })
    } catch (e) {
      notice(`打开这个对话失败：${String((e as Error).message ?? e)}`, 'error')
    } finally {
      state.working = false
    }
  }

  async function renameThread(id: string, name: string) {
    const n = name.trim()
    if (!n) return
    await client.request('thread/name/set', { threadId: id, name: n }).catch(() => {})
    await refreshThreads()
  }

  async function archiveThread(id: string) {
    await client.request('thread/archive', { threadId: id }).catch(() => {})
    if (id === state.threadId && !state.busy) await newThread()
    await refreshThreads()
  }

  /**
   * Sends a message. `template` (a template id, `sample:<id>`, or nothing for free text) is only
   * used for analytics and to name a new conversation after the template.
   */
  async function send(text: string, meta: { template?: string; title?: string } = {}) {
    const t = text.trim()
    if (!t || !state.threadId || state.busy || state.working) return
    state.entries.push({ kind: 'user', id: `u${Date.now()}`, text: t })
    state.busy = true
    const threadId = state.threadId
    const turnIndex = state.turnCount
    // Copy the folder first so this turn can be undone; a failed copy never blocks the turn.
    const snap = await invoke<SnapshotMeta>('snapshot_take', { threadId, cwd: state.cwd, turnIndex }).catch(() => null)
    const template = meta.template ?? 'custom'
    currentTurn = { template, startedAt: Date.now(), totalBefore: state.usage?.total ?? null }
    track('desktop_task_start', { template, turn: turnIndex, snapshot: snap ? (snap.skipped ? 'skipped' : 'ok') : 'error' })
    try {
      await client.request('turn/start', {
        threadId,
        input: [{ type: 'text', text: t, text_elements: [] }]
      })
      state.turnCount = turnIndex + 1
      if (turnIndex === 0 && meta.title) {
        client.request('thread/name/set', { threadId, name: meta.title }).catch(() => {})
      }
    } catch (e) {
      state.busy = false
      currentTurn = null
      if (snap) invoke('snapshot_discard', { threadId, id: snap.id }).catch(() => {})
      errorNotice(String((e as Error).message ?? e))
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

  async function lastSnapshot(): Promise<SnapshotMeta | null> {
    const list = await invoke<SnapshotMeta[]>('snapshot_list', { threadId: state.threadId }).catch(() => [])
    const last = list[list.length - 1]
    // Only a snapshot taken right before the last turn belongs to it.
    return last && last.turn_index === state.turnCount - 1 ? last : null
  }

  async function undoPreview(): Promise<UndoPreview> {
    const snap = await lastSnapshot()
    if (!snap) return { files: false, reason: '这一轮开始前没有保存文件（可能是旧版本留下的对话），只能撤销对话，文件改动不会还原。' }
    if (snap.skipped) return { files: false, reason: `这一轮开始前没有保存文件（${snap.skipped}），只能撤销对话，文件改动不会还原。` }
    return { files: true, reason: '' }
  }

  /**
   * Undoes the last turn: puts the folder back as it was before it (files that turn created go to
   * 撤销回收站), then drops the turn from the conversation. Returns the undone message so the
   * composer can offer it again; null when nothing was undone.
   */
  async function undo(): Promise<string | null> {
    if (state.busy || state.working || !state.threadId || state.turnCount === 0) return null
    state.working = true
    try {
      const lastUser = [...state.entries].reverse().find((e) => e.kind === 'user') as Extract<Entry, { kind: 'user' }> | undefined
      const snap = await lastSnapshot()
      let restored: Restored | null = null
      if (snap && !snap.skipped) {
        try {
          restored = await invoke<Restored>('snapshot_restore', { threadId: state.threadId, id: snap.id })
        } catch (e) {
          // Don't touch the conversation when the files could not be put back.
          notice(`撤销失败：${String(e)}。文件和对话都没有改动；如果有文件正在被 Excel / Word 打开，关掉再试。`, 'error')
          return null
        }
      }
      const res = await client.request('thread/rollback', { threadId: state.threadId, numTurns: 1 })
      let turns = res?.thread?.turns
      if (!Array.isArray(turns) || (turns.length === 0 && state.turnCount > 1)) {
        const read = await client.request('thread/read', { threadId: state.threadId, includeTurns: true })
        turns = read?.thread?.turns ?? []
      }
      state.entries = entriesFromTurns(turns)
      state.turnCount = turns.length
      if (restored) {
        const parts = [`已撤销上一轮，文件夹恢复到了这一轮开始前的样子（还原 ${restored.restored} 个文件）。`]
        if (restored.moved_aside > 0) parts.push(`这一轮新建的 ${restored.moved_aside} 个文件没有删除，挪到了「撤销回收站」。`)
        notice(parts.join(''), 'info', restored.trash ? { action: 'open', path: restored.trash } : {})
      } else {
        notice('已撤销上一轮对话。这一轮改过的文件没有还原。')
      }
      track('desktop_undo', { files: restored ? 'restored' : snap?.skipped ? 'skipped' : 'none' })
      refreshThreads()
      return lastUser?.text ?? null
    } catch (e) {
      notice(`撤销失败：${String((e as Error).message ?? e)}`, 'error')
      return null
    } finally {
      state.working = false
    }
  }

  return {
    state,
    connect,
    send,
    interrupt,
    answer,
    newThread,
    openThread,
    renameThread,
    archiveThread,
    refreshUsage,
    undo,
    undoPreview
  }
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

/** Out of balance / allowance: the gateway's 402/403/429 quota messages. */
export function isQuotaError(message: string): boolean {
  return /余额不足|余额|额度不足|额度已用完|insufficient|balance|quota|402/i.test(message)
}

/** Gateway errors are already Chinese; translate the common transport ones. */
export function friendlyError(message: string): string {
  if (/401|unauthorized|invalid api key|无效/i.test(message)) return `Key 无效或已过期，请重新登录。（${message}）`
  if (isQuotaError(message)) return `余额或额度不够了，充值后再继续。（${message}）`
  if (/429|rate limit/i.test(message)) return message
  if (/timeout|timed out|connection|network|dns/i.test(message)) return `连接 HiveGPT 失败，请检查网络后重试。（${message}）`
  return message
}
