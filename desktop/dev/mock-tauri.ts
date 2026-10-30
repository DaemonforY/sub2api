// UI preview without Tauri: `pnpm dev`, then open http://localhost:1420/dev/mock.html in a browser.
// Every Rust command and the Codex app-server are faked here (Tauri's own mockIPC), so the setup
// screen, chat, history sidebar, undo and the balance chip can be looked at and clicked through.
// Nothing here ships: the build only bundles index.html.

import { mockIPC } from '@tauri-apps/api/mocks'
import { emit } from '@tauri-apps/api/event'

const WS = '/Users/demo/Documents/HiveGPT 工作区'
const now = Math.floor(Date.now() / 1000)

type Turn = { id: string; status: string; items: any[] }
const threads: Record<string, { id: string; name: string | null; preview: string; cwd: string; updatedAt: number; turns: Turn[]; archived?: boolean }> = {
  t1: {
    id: 't1',
    name: '发票单据整理成 Excel',
    preview: '这个文件夹里有一些发票…',
    cwd: WS,
    updatedAt: now - 3600,
    turns: [
      {
        id: 'turn-a',
        status: 'completed',
        items: [
          { type: 'userMessage', id: 'u1', content: [{ type: 'text', text: '把发票整理成「输出/发票汇总.xlsx」' }] },
          { type: 'agentMessage', id: 'a1', text: '整理好了：共 8 张，总金额 ¥12,480.00，有 1 张需要核对。结果在「输出/发票汇总.xlsx」。' }
        ]
      }
    ]
  },
  t2: {
    id: 't2',
    name: null,
    preview: '帮我把会议纪要里的待办列成表格，按负责人分组',
    cwd: '/Users/demo/Desktop/周会',
    updatedAt: now - 86400 - 600,
    turns: [
      {
        id: 'turn-b',
        status: 'completed',
        items: [
          { type: 'userMessage', id: 'u2', content: [{ type: 'text', text: '帮我把会议纪要里的待办列成表格，按负责人分组' }] },
          { type: 'agentMessage', id: 'a2', text: '已生成「输出/待办清单.xlsx」，共 14 项，分给 5 个人。' }
        ]
      }
    ]
  }
}
let seq = 0
let current = ''
const snapshots: { id: string; turn_index: number; skipped: string | null; thread: string }[] = []

const send = (msg: object) => setTimeout(() => emit('engine-message', JSON.stringify(msg)), 30)
const summary = (t: (typeof threads)[string]) => ({ ...t, turns: [] })

function rpc(line: string) {
  const m = JSON.parse(line)
  if (m.id === undefined || !m.method) return
  const reply = (result: unknown) => send({ id: m.id, result })
  const p = m.params ?? {}
  switch (m.method) {
    case 'initialize':
      return reply({ userAgent: 'mock', codexHome: '/tmp', platformFamily: 'unix', platformOs: 'macos' })
    case 'thread/start': {
      const id = `new${++seq}`
      threads[id] = { id, name: null, preview: '', cwd: p.cwd, updatedAt: now, turns: [] }
      current = id
      return reply({ thread: summary(threads[id]) })
    }
    case 'thread/list':
      return reply({
        data: Object.values(threads)
          .filter((t) => !t.archived)
          .sort((a, b) => b.updatedAt - a.updatedAt)
          .map(summary),
        nextCursor: null,
        backwardsCursor: null
      })
    case 'thread/resume':
      current = p.threadId
      return reply({ thread: threads[p.threadId], cwd: threads[p.threadId].cwd })
    case 'thread/read':
      return reply({ thread: threads[p.threadId] })
    case 'thread/name/set':
      threads[p.threadId].name = p.name
      return reply({})
    case 'thread/archive':
      threads[p.threadId].archived = true
      return reply({})
    case 'thread/rollback': {
      const t = threads[p.threadId]
      t.turns = t.turns.slice(0, Math.max(0, t.turns.length - p.numTurns))
      return reply({ thread: t })
    }
    case 'turn/start': {
      const t = threads[p.threadId]
      const text = p.input?.[0]?.text ?? ''
      const turnId = `turn${++seq}`
      if (!t.preview) t.preview = text
      t.updatedAt = Math.floor(Date.now() / 1000)
      reply({ turn: { id: turnId } })
      send({ method: 'turn/started', params: { turn: { id: turnId } } })
      const user = { type: 'userMessage', id: `u${seq}`, content: [{ type: 'text', text }] }
      const file = { type: 'fileChange', id: `f${seq}`, status: 'completed', changes: [{ path: `${t.cwd}/输出/结果.xlsx`, kind: { type: 'add' } }] }
      const agent = { type: 'agentMessage', id: `a${seq}`, text: '好的，已经做完了，结果在「输出/结果.xlsx」。' }
      setTimeout(() => send({ method: 'item/completed', params: { item: file } }), 300)
      setTimeout(() => send({ method: 'item/completed', params: { item: agent } }), 600)
      setTimeout(() => {
        t.turns.push({ id: turnId, status: 'completed', items: [user, agent] })
        usageTotal += 0.0421
        send({ method: 'turn/completed', params: { turn: { id: turnId, status: 'completed' } } })
      }, 900)
      return
    }
    default:
      return reply({})
  }
}

let usageTotal = 3.2
mockIPC(
  (cmd, args: any) => {
    switch (cmd) {
      case 'engine_status':
        return { ready: true, version: '0.142.3', path: '' }
      case 'office_status':
        return { ready: true }
      case 'account_state':
        return { signed_in: true, base_url: 'https://hivegpt.cn', key_hint: 'sk-ab…wxyz', user_name: '演示用户', group_name: 'GPT-按量' }
      case 'default_workspace':
        return WS
      case 'dev_autorun':
        return null
      case 'dev_undo':
        return false
      case 'check_key':
        return { status: 'ok', message: '' }
      case 'usage_summary':
        return {
          isValid: true,
          planName: '钱包余额',
          remaining: 0.86,
          balance: 0.86,
          usage: { today: { actual_cost: usageTotal - 3.0 }, total: { actual_cost: usageTotal } }
        }
      case 'engine_send':
        rpc(args.line)
        return null
      case 'snapshot_take': {
        const s = { id: `s${++seq}`, cwd: args.cwd, turn_index: args.turnIndex, skipped: null, thread: args.threadId }
        snapshots.push(s)
        return s
      }
      case 'snapshot_list':
        return snapshots.filter((s) => s.thread === args.threadId)
      case 'snapshot_restore': {
        const i = snapshots.findIndex((s) => s.id === args.id)
        if (i >= 0) snapshots.splice(i, 1)
        return { restored: 1, moved_aside: 1, trash: '/Users/demo/Library/Application Support/cn.hivegpt.desktop/撤销回收站/2026-10-31_10-00-00' }
      }
      default:
        return null
    }
  },
  { shouldMockEvents: true }
)
void current
