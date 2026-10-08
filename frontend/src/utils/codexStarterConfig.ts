/**
 * The Codex config.toml the dashboard's 「三步开始使用」 offers as a download, so setting up
 * Codex needs no terminal: download, drop it into the .codex folder, restart Codex.
 * Same provider block as the 「使用密钥」 dialog in API-key mode (the key travels in the file,
 * never in a URL); no model catalog line, since that file wouldn't exist yet.
 */

export type DesktopOS = 'windows' | 'mac' | 'linux'

export function detectDesktopOS(userAgent: string): DesktopOS {
  if (/windows/i.test(userAgent)) return 'windows'
  if (/mac os|macintosh/i.test(userAgent)) return 'mac'
  return 'linux'
}

/** Where Codex keeps config.toml, as typed into Win+R / Finder's 前往文件夹 / a file manager. */
export function codexConfigFolder(os: DesktopOS): string {
  return os === 'windows' ? '%USERPROFILE%\\.codex' : '~/.codex'
}

function tomlString(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

export function codexStarterConfig(baseUrl: string, apiKey: string, model = 'gpt-5.5'): string {
  const base = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  return `model_provider = "OpenAI"
model = "${tomlString(model)}"
review_model = "${tomlString(model)}"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${tomlString(base)}"
wire_api = "responses"
requires_openai_auth = false
experimental_bearer_token = "${tomlString(apiKey)}"
http_headers = { "x-openai-actor-authorization" = "local-image-extension" }

[features]
goals = true
`
}

/** Saves text as a file through the browser (nothing leaves the page). */
export function downloadTextFile(name: string, text: string): void {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
