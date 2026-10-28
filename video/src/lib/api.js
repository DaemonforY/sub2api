// API client for HiveGPT 视频. The app runs on video.<domain>, served by the same backend, so the API
// and the gateway are same-origin. Signed-in calls carry the user's API key.
import { reactive } from 'vue'

const KEY_STORAGE = 'hivegpt-video:key'
export const MAIN_SITE_URL = import.meta.env.VITE_MAIN_SITE_URL || 'https://hivegpt.cn'

export const session = reactive({
  key: localStorage.getItem(KEY_STORAGE) || '',
  me: null,
  loading: false
})

export class ApiError extends Error {
  constructor(message, status, reason) {
    super(message)
    this.status = status
    this.reason = reason
  }
}

// Server messages end with the English original in parentheses; show the Chinese part.
const stripEnglish = (msg) => String(msg || '').replace(/（[A-Za-z][^（）]*）\s*$/, '').trim()

export async function api(path, { method = 'GET', body, auth = true, signal } = {}) {
  const headers = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth && session.key) headers.Authorization = `Bearer ${session.key}`
  let res
  try {
    res = await fetch(`/api/v1/video${path}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body), signal, cache: 'no-store' })
  } catch (err) {
    if (err?.name === 'AbortError') throw err
    throw new ApiError('网络连接失败，请检查网络后重试', 0)
  }
  let data = null
  try {
    data = await res.json()
  } catch {
    // not JSON
  }
  if (res.ok && data && data.code === 0) return data.data
  if (res.status === 401 && auth) {
    signOut()
  }
  const message = stripEnglish(data?.message) || (res.status === 429 ? '请求太频繁，请稍后再试' : `请求失败（HTTP ${res.status}）`)
  throw new ApiError(message, res.status, data?.reason)
}

/**
 * Sends a multipart form (uploads) with progress: onProgress gets 0–1 while the body is sent.
 * Errors keep the server's English in parentheses so a failed upload can be looked up.
 */
export function apiUpload(path, form, { onProgress, signal } = {}) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `/api/v1/video${path}`)
    if (session.key) xhr.setRequestHeader('Authorization', `Bearer ${session.key}`)
    xhr.upload.onprogress = (e) => e.lengthComputable && onProgress?.(e.loaded / e.total)
    xhr.onload = () => {
      let data = null
      try {
        data = JSON.parse(xhr.responseText)
      } catch {
        // not JSON (e.g. a proxy error page)
      }
      if (xhr.status >= 200 && xhr.status < 300 && data?.code === 0) return resolve(data.data)
      if (xhr.status === 401) signOut()
      const fallback =
        xhr.status === 413 ? '文件太大，视频不能超过 100 MB（File too large）' : xhr.status === 429 ? '请求太频繁，请稍后再试（Too many requests）' : `上传失败，请重试（HTTP ${xhr.status}）`
      reject(new ApiError(data?.message || fallback, xhr.status, data?.reason))
    }
    xhr.onerror = () => reject(new ApiError('网络中断，上传没有完成，请检查网络后重试（Network error）', 0))
    xhr.onabort = () => reject(new DOMException('aborted', 'AbortError'))
    signal?.addEventListener('abort', () => xhr.abort())
    xhr.send(form)
  })
}

/** Fetches a narration file with the key (own projects) or anonymously (public works) as a blob URL. */
export async function audioBlob(url, withKey) {
  const res = await fetch(url, { headers: withKey && session.key ? { Authorization: `Bearer ${session.key}` } : {} })
  if (!res.ok) throw new ApiError('配音加载失败', res.status)
  return res.arrayBuffer()
}

export async function loadMe() {
  if (!session.key) {
    session.me = null
    return null
  }
  session.loading = true
  try {
    session.me = await api('/me')
    return session.me
  } catch (err) {
    if (err.status === 401) signOut()
    return null
  } finally {
    session.loading = false
  }
}

export function setKey(key) {
  session.key = key
  localStorage.setItem(KEY_STORAGE, key)
  return loadMe()
}

export function signOut() {
  session.key = ''
  session.me = null
  localStorage.removeItem(KEY_STORAGE)
}

const CONNECT_STATE_STORAGE = 'hivegpt-video:connect-state'
const CONNECT_STATE_TTL_MS = 30 * 60 * 1000

function randomState() {
  return Array.from(crypto.getRandomValues(new Uint8Array(18)), (b) => b.toString(16).padStart(2, '0')).join('')
}

function connectURL(state, returnPath) {
  const params = new URLSearchParams({ state, utm_source: 'video', utm_medium: 'sign-in' })
  if (returnPath) params.set('return', returnPath)
  return `${MAIN_SITE_URL}/video-connect?${params.toString()}`
}

/**
 * Full-page sign-in for the WeChat in-app browser (no popups) or when the popup was blocked: the
 * main site's /video-connect?return= sends the user back here with the key in the URL fragment,
 * which consumeConnectRedirect() checks against the state saved now.
 */
function signInByRedirect(state) {
  try {
    localStorage.setItem(CONNECT_STATE_STORAGE, JSON.stringify({ state, at: Date.now() }))
  } catch {
    // storage unavailable: the state check on return will fail and the user can try again
  }
  const returnPath = `${window.location.pathname}${window.location.search}` || '/'
  window.location.href = connectURL(state, returnPath)
  // The page is leaving; never settle so callers keep their "signing in" state.
  return new Promise(() => {})
}

/**
 * Picks up the key handed back by the main site's full-page sign-in (see signInByRedirect) and
 * removes it from the address bar right away. Returns true when it signed in.
 */
export async function consumeConnectRedirect() {
  const hash = window.location.hash.replace(/^#/, '')
  if (!hash.includes('hivegpt_video_key=')) return false
  const params = new URLSearchParams(hash)
  window.history.replaceState(window.history.state, '', `${window.location.pathname}${window.location.search}`)
  let saved = null
  try {
    saved = JSON.parse(localStorage.getItem(CONNECT_STATE_STORAGE) || 'null')
    localStorage.removeItem(CONNECT_STATE_STORAGE)
  } catch {
    saved = null
  }
  const state = params.get('hivegpt_video_state') || ''
  const key = params.get('hivegpt_video_key') || ''
  const fresh = saved && typeof saved.at === 'number' && Date.now() - saved.at < CONNECT_STATE_TTL_MS
  if (!key || !state || !fresh || saved.state !== state) return false
  await setKey(key)
  return true
}

/**
 * Opens the main site's 授权 popup; resolves when the user picked a key there. The key arrives by
 * postMessage from the main site's origin with the random state we sent. Inside WeChat, or when the
 * popup is blocked, it signs in with a full-page round trip instead.
 */
export function signIn() {
  const state = randomState()
  if (/MicroMessenger/i.test(navigator.userAgent || '')) return signInByRedirect(state)
  const popup = window.open(connectURL(state), 'hivegpt-video-connect', 'width=480,height=720')
  if (!popup) return signInByRedirect(state)
  return new Promise((resolve) => {
    const origin = new URL(MAIN_SITE_URL).origin
    const timer = setInterval(() => {
      if (popup.closed) {
        cleanup()
        resolve(false)
      }
    }, 600)
    function onMessage(e) {
      if (e.origin !== origin || e.data?.type !== 'hivegpt:video-key' || e.data?.state !== state || typeof e.data.apiKey !== 'string') return
      cleanup()
      setKey(e.data.apiKey).then(() => resolve(true))
    }
    function cleanup() {
      clearInterval(timer)
      window.removeEventListener('message', onMessage)
    }
    window.addEventListener('message', onMessage)
  })
}

let catalogPromise = null
/** Styles, genres, categories, ratios and voices (cached). */
export function catalog() {
  catalogPromise ||= api('/catalog', { auth: false }).catch((err) => {
    catalogPromise = null
    throw err
  })
  return catalogPromise
}

/** Text models the signed-in key can use (from the gateway's /v1/models). */
export async function textModels() {
  if (!session.key) return []
  try {
    const res = await fetch('/v1/models', { headers: { Authorization: `Bearer ${session.key}` } })
    const data = await res.json()
    return (data.data || [])
      .map((m) => m.id)
      .filter((id) => /^(gpt-|claude|gemini|deepseek|qwen|glm|kimi|doubao)/i.test(id) && !/(image|tts|audio|realtime|embedding|whisper|transcribe|search|review)/i.test(id))
  } catch {
    return []
  }
}

/** The signed-in user's cloned voices (我的音色); [] when signed out or unavailable. */
export async function myClones() {
  if (!session.key) return []
  try {
    return (await api('/clones')).items || []
  } catch {
    return []
  }
}
