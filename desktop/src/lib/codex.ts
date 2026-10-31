// JSON-RPC client for `codex app-server`, spoken over the Rust bridge (src-tauri/src/bridge.rs):
// every line Codex prints arrives as an `engine-message` event; we send lines with `engine_send`.
//
// Message shapes follow the protocol Codex generates with `codex app-server generate-ts`
// (pinned engine 0.142.3). Only the subset the app uses is typed here.

import { invoke } from '@tauri-apps/api/core'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'

export type Json = null | boolean | number | string | Json[] | { [k: string]: Json }

export interface RpcError {
  code: number
  message: string
  data?: Json
}

export interface ServerRequest {
  id: number | string
  method: string
  params: any
}

export interface InitializeResult {
  userAgent: string
  codexHome: string
  platformFamily: string
  platformOs: string
}

type Pending = { resolve: (v: any) => void; reject: (e: Error) => void }

export class CodexClient {
  private nextId = 1
  private pending = new Map<number | string, Pending>()
  private unlisten: UnlistenFn[] = []

  onNotification: (method: string, params: any) => void = () => {}
  onServerRequest: (req: ServerRequest) => void = () => {}
  onExit: () => void = () => {}

  /** Starts a fresh engine process and completes the initialize handshake. */
  async start(): Promise<InitializeResult> {
    await this.dispose()
    // Listen before spawning so no early line is missed.
    this.unlisten.push(await listen<string>('engine-message', (e) => this.handleLine(e.payload)))
    this.unlisten.push(
      await listen('engine-exit', () => {
        for (const p of this.pending.values()) p.reject(new Error('引擎已退出'))
        this.pending.clear()
        this.onExit()
      })
    )
    // A dev-server reload leaves the old process running; it is already initialized.
    await invoke('engine_stop')
    await invoke('engine_start')
    const init = await this.request<InitializeResult>('initialize', {
      clientInfo: { name: 'hivegpt_desktop', title: 'HiveGPT 桌面助手', version: '0.2.1' },
      capabilities: null
    })
    await this.notify('initialized')
    return init
  }

  async dispose(): Promise<void> {
    for (const u of this.unlisten) u()
    this.unlisten = []
  }

  request<T = any>(method: string, params?: unknown): Promise<T> {
    const id = this.nextId++
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      this.send({ id, method, params }).catch((e) => {
        this.pending.delete(id)
        reject(e instanceof Error ? e : new Error(String(e)))
      })
    })
  }

  notify(method: string, params?: unknown): Promise<void> {
    return this.send(params === undefined ? { method } : { method, params })
  }

  respond(id: number | string, result: unknown): Promise<void> {
    return this.send({ id, result })
  }

  respondError(id: number | string, error: RpcError): Promise<void> {
    return this.send({ id, error })
  }

  private send(msg: object): Promise<void> {
    return invoke('engine_send', { line: JSON.stringify(msg) })
  }

  private handleLine(line: string) {
    let msg: any
    try {
      msg = JSON.parse(line)
    } catch {
      return
    }
    const hasId = msg && msg.id !== undefined && msg.id !== null
    if (hasId && ('result' in msg || 'error' in msg)) {
      const p = this.pending.get(msg.id)
      if (!p) return
      this.pending.delete(msg.id)
      if (msg.error) p.reject(new Error(msg.error.message || '请求失败'))
      else p.resolve(msg.result)
    } else if (hasId && typeof msg.method === 'string') {
      this.onServerRequest({ id: msg.id, method: msg.method, params: msg.params })
    } else if (typeof msg.method === 'string') {
      this.onNotification(msg.method, msg.params)
    }
  }
}
