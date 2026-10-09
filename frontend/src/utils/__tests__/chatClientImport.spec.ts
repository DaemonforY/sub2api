import { describe, expect, it } from 'vitest'
import { chatboxImportLink, chatClientRoot, cherryStudioImportLink } from '../chatClientImport'

const decode = (link: string, param: string) => {
  const raw = new URL(link.replace(/^(\w+):\/\//, 'https://')).searchParams.get(param) || ''
  return JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(raw), (c) => c.charCodeAt(0))))
}

describe('chat client import links', () => {
  it('normalises the site root', () => {
    expect(chatClientRoot('https://hivegpt.cn/')).toBe('https://hivegpt.cn')
    expect(chatClientRoot('https://hivegpt.cn/v1/')).toBe('https://hivegpt.cn')
  })

  it('builds the Cherry Studio provider link without /v1', () => {
    const link = cherryStudioImportLink('https://hivegpt.cn/', 'sk-abc+/=')
    expect(link.startsWith('cherrystudio://providers/api-keys?v=1&data=')).toBe(true)
    expect(decode(link, 'data')).toEqual({ id: 'hivegpt', name: 'HiveGPT', type: 'openai', baseUrl: 'https://hivegpt.cn', apiKey: 'sk-abc+/=' })
  })

  it('builds the Chatbox provider link with host + path = /v1/chat/completions', () => {
    const link = chatboxImportLink('https://hivegpt.cn/v1', 'sk-abc')
    expect(link.startsWith('chatbox://provider/import?config=')).toBe(true)
    const cfg = decode(link, 'config')
    expect(cfg.type).toBe('openai')
    expect(cfg.settings.apiHost + cfg.settings.apiPath).toBe('https://hivegpt.cn/v1/chat/completions')
    expect(cfg.settings.apiKey).toBe('sk-abc')
    expect(cfg.settings.models[0].modelId).toBe('gpt-5.5')
  })
})
