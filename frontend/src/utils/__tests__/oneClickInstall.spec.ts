import { describe, expect, it } from 'vitest'
import {
  installBase,
  oneClickInstallCommand,
  oneClickRestoreCommand,
  supportsOneClickInstall,
  windowsInstallerScript
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

describe('windowsInstallerScript', () => {
  it('sets the key, runs install.ps1 and deletes itself, with CRLF line ends', () => {
    const script = windowsInstallerScript('https://hivegpt.cn/v1', 'sk-abc_123')!
    expect(script).toContain('set "HIVEGPT_KEY=sk-abc_123"\r\n')
    expect(script).toContain('irm https://hivegpt.cn/install.ps1 | iex')
    expect(script).not.toContain('HIVEGPT_BASE_URL')
    expect(script.trimEnd().endsWith('del "%~f0"')).toBe(true)
    expect(script.replace(/\r\n/g, '')).not.toContain('\n')
    expect(/^[\x20-\x7e\r\n]*$/.test(script)).toBe(true)
  })

  it('passes another origin and refuses values cmd would interpret', () => {
    expect(windowsInstallerScript('https://api.example.com', 'sk-x')).toContain('set "HIVEGPT_BASE_URL=https://api.example.com"')
    expect(windowsInstallerScript('https://hivegpt.cn', 'sk-a&b')).toBeNull()
    expect(windowsInstallerScript('https://hivegpt.cn', 'sk-%PATH%')).toBeNull()
  })
})
