/**
 * The 「一键安装」 commands in the 使用密钥 dialog. They run public/install.sh or install.ps1 from
 * this site, which install Claude Code / Codex from domestic mirrors and merge the key into their
 * configs (backing up the old ones). The key travels in an environment variable, never in a URL.
 */

import type { GroupPlatform } from '@/types'

export type InstallOS = 'unix' | 'windows'

/** The script's built-in default; other origins (self-hosted) pass HIVEGPT_BASE_URL. */
export const DEFAULT_INSTALL_BASE = 'https://hivegpt.cn'

// Groups served through /v1/messages and /v1/responses at the site root. Gemini and Antigravity
// need their own paths and Grok its own model names, so those keep the manual configs.
const SUPPORTED_PLATFORMS: GroupPlatform[] = ['anthropic', 'openai', 'deepseek', 'composite', 'kimi', 'zhipu']

export function supportsOneClickInstall(platform: GroupPlatform | null | undefined): boolean {
  return Boolean(platform && SUPPORTED_PLATFORMS.includes(platform))
}

export function installBase(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, '').replace(/\/v1$/i, '')
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

function psQuote(value: string): string {
  return `'${value.replace(/'/g, "''")}'`
}

export function oneClickInstallCommand(os: InstallOS, baseUrl: string, apiKey: string): string {
  const base = installBase(baseUrl) || DEFAULT_INSTALL_BASE
  const custom = base !== DEFAULT_INSTALL_BASE
  if (os === 'windows') {
    const env = [`$env:HIVEGPT_KEY=${psQuote(apiKey)}`]
    if (custom) env.push(`$env:HIVEGPT_BASE_URL=${psQuote(base)}`)
    return `${env.join('; ')}; irm ${base}/install.ps1 | iex`
  }
  const env = [`HIVEGPT_KEY=${shellQuote(apiKey)}`]
  if (custom) env.push(`HIVEGPT_BASE_URL=${shellQuote(base)}`)
  return `curl -fsSL ${base}/install.sh | ${env.join(' ')} bash`
}

export function oneClickRestoreCommand(os: InstallOS, baseUrl: string): string {
  const base = installBase(baseUrl) || DEFAULT_INSTALL_BASE
  const custom = base !== DEFAULT_INSTALL_BASE
  if (os === 'windows') {
    return `$env:HIVEGPT_RESTORE='1'; ${custom ? `$env:HIVEGPT_BASE_URL=${psQuote(base)}; ` : ''}irm ${base}/install.ps1 | iex`
  }
  return `curl -fsSL ${base}/install.sh | ${custom ? `HIVEGPT_BASE_URL=${shellQuote(base)} ` : ''}bash -s -- --restore`
}
