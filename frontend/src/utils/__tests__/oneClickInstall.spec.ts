import { describe, expect, it } from 'vitest'
import {
  installBase,
  oneClickInstallCommand,
  oneClickRestoreCommand,
  supportsOneClickInstall
} from '../oneClickInstall'

describe('oneClickInstall', () => {
  it('builds the hivegpt.cn commands without a base override', () => {
    expect(oneClickInstallCommand('unix', 'https://hivegpt.cn/v1', 'sk-abc')).toBe(
      "curl -fsSL https://hivegpt.cn/install.sh | HIVEGPT_KEY='sk-abc' bash"
    )
    expect(oneClickInstallCommand('windows', 'https://hivegpt.cn/', 'sk-abc')).toBe(
      "$env:HIVEGPT_KEY='sk-abc'; irm https://hivegpt.cn/install.ps1 | iex"
    )
    expect(oneClickRestoreCommand('unix', 'https://hivegpt.cn')).toBe('curl -fsSL https://hivegpt.cn/install.sh | bash -s -- --restore')
    expect(oneClickRestoreCommand('windows', 'https://hivegpt.cn')).toBe("$env:HIVEGPT_RESTORE='1'; irm https://hivegpt.cn/install.ps1 | iex")
  })

  it('passes another origin to the script', () => {
    expect(oneClickInstallCommand('unix', 'https://api.example.com/v1', 'sk-x')).toBe(
      "curl -fsSL https://api.example.com/install.sh | HIVEGPT_KEY='sk-x' HIVEGPT_BASE_URL='https://api.example.com' bash"
    )
    expect(oneClickInstallCommand('windows', 'https://api.example.com', 'sk-x')).toBe(
      "$env:HIVEGPT_KEY='sk-x'; $env:HIVEGPT_BASE_URL='https://api.example.com'; irm https://api.example.com/install.ps1 | iex"
    )
  })

  it('quotes keys safely for each shell', () => {
    expect(oneClickInstallCommand('unix', 'https://hivegpt.cn', "a'b")).toContain(`HIVEGPT_KEY='a'\\''b'`)
    expect(oneClickInstallCommand('windows', 'https://hivegpt.cn', "a'b")).toContain(`$env:HIVEGPT_KEY='a''b'`)
  })

  it('only offers platforms the script can configure', () => {
    expect(supportsOneClickInstall('anthropic')).toBe(true)
    expect(supportsOneClickInstall('openai')).toBe(true)
    expect(supportsOneClickInstall('deepseek')).toBe(true)
    expect(supportsOneClickInstall('gemini')).toBe(false)
    expect(supportsOneClickInstall('antigravity')).toBe(false)
    expect(supportsOneClickInstall('grok')).toBe(false)
    expect(supportsOneClickInstall(null)).toBe(false)
    expect(installBase('https://hivegpt.cn/v1/')).toBe('https://hivegpt.cn')
  })
})
