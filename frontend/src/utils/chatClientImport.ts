/**
 * One-click import of a HiveGPT key into desktop chat clients, using each client's own deep link:
 * - Cherry Studio: cherrystudio://providers/api-keys?v=1&data=<base64 JSON {id, name, type, baseUrl, apiKey}>
 *   (src/main/services/protocol/handlers/providersImport.ts). Cherry appends /v1 itself, so baseUrl has none.
 * - Chatbox: chatbox://provider/import?config=<base64 JSON provider config> (src/main/deeplinks.ts,
 *   decoded with js-base64 in routes/settings/provider/route.tsx).
 * Both clients show the provider and ask before saving it.
 */

export const CHAT_CLIENT_DEFAULT_MODEL = 'gpt-5.5'

/** Site root without a trailing slash or /v1, e.g. https://hivegpt.cn */
export function chatClientRoot(baseUrl: string): string {
  return (baseUrl || '').trim().replace(/\/+$/, '').replace(/\/v1$/, '').replace(/\/+$/, '')
}

function base64Utf8(text: string): string {
  const bytes = new TextEncoder().encode(text)
  let binary = ''
  bytes.forEach((b) => (binary += String.fromCharCode(b)))
  return btoa(binary)
}

export function cherryStudioImportLink(baseUrl: string, apiKey: string, name = 'HiveGPT'): string {
  const data = { id: 'hivegpt', name, type: 'openai', baseUrl: chatClientRoot(baseUrl), apiKey }
  return `cherrystudio://providers/api-keys?v=1&data=${encodeURIComponent(base64Utf8(JSON.stringify(data)))}`
}

export function chatboxImportLink(baseUrl: string, apiKey: string, name = 'HiveGPT'): string {
  const root = chatClientRoot(baseUrl)
  const config = {
    id: 'hivegpt',
    name,
    type: 'openai',
    urls: { website: root },
    settings: {
      apiHost: root,
      apiPath: '/v1/chat/completions',
      apiKey,
      models: [{ modelId: CHAT_CLIENT_DEFAULT_MODEL, nickname: CHAT_CLIENT_DEFAULT_MODEL, type: 'chat' }]
    }
  }
  return `chatbox://provider/import?config=${encodeURIComponent(base64Utf8(JSON.stringify(config)))}`
}
