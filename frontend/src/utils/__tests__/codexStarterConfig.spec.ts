import { describe, expect, it } from 'vitest'
import { codexConfigFolder, codexStarterConfig, detectDesktopOS, pickCodexDefaultModel } from '../codexStarterConfig'

describe('codex starter config', () => {
  it('points Codex at the site with the key in the file', () => {
    const toml = codexStarterConfig('https://hivegpt.cn/v1/', 'sk-test"x')
    expect(toml).toContain('base_url = "https://hivegpt.cn"')
    expect(toml).toContain('wire_api = "responses"')
    expect(toml).toContain('requires_openai_auth = false')
    expect(toml).toContain('experimental_bearer_token = "sk-test\\"x"')
    expect(toml).toContain('model = "gpt-6.1-sol"')
    expect(toml).not.toContain('model_catalog_json')
  })

  it('knows the desktop and where the folder is', () => {
    expect(detectDesktopOS('Mozilla/5.0 (Windows NT 10.0; Win64; x64)')).toBe('windows')
    expect(detectDesktopOS('Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)')).toBe('mac')
    expect(detectDesktopOS('Mozilla/5.0 (X11; Linux x86_64)')).toBe('linux')
    expect(codexConfigFolder('windows')).toBe('%USERPROFILE%\\.codex')
    expect(codexConfigFolder('mac')).toBe('~/.codex')
  })
})

describe('pickCodexDefaultModel', () => {
  it("follows Codex's order through what the group's catalog has", () => {
    expect(pickCodexDefaultModel(['gpt-5.5', 'gpt-6-luna', 'gpt-6.1-sol'])).toBe('gpt-6.1-sol')
    expect(pickCodexDefaultModel(['gpt-5.5', 'gpt-6-luna'])).toBe('gpt-6-luna')
    expect(pickCodexDefaultModel(['claude-opus-4-8', 'gpt-5.5'])).toBe('gpt-5.5')
    expect(pickCodexDefaultModel(['deepseek-v4-pro'])).toBe('deepseek-v4-pro')
    expect(pickCodexDefaultModel([])).toBe('gpt-6.1-sol')
  })
})
