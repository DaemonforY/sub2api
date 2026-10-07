// Talks to the main site's API (same origin) with the login token the main site keeps in
// localStorage — the same scheme as the learning site (learn/.vitepress/theme/api.ts).
// Backend contract: editor/API.md.

const TOKEN_KEY = 'auth_token';
const USER_KEY = 'auth_user';
const EXPIRES_KEY = 'token_expires_at';

export class ApiError extends Error {
  constructor(message, status, reason = '') {
    super(message);
    this.status = status;
    this.reason = reason;
  }
}

export function token() {
  const expires = Number(localStorage.getItem(EXPIRES_KEY) || 0);
  if (expires && expires < Date.now()) return '';
  return localStorage.getItem(TOKEN_KEY) || '';
}

export function currentUser() {
  if (!token()) return null;
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null');
  } catch {
    return null;
  }
}

/** The main site's sign-in page, coming back to the editor afterwards. */
export function loginUrl() {
  return `/login?redirect=${encodeURIComponent('/editor/')}`;
}

function authHeaders(extra = {}) {
  const headers = { ...extra };
  const t = token();
  if (t) headers.Authorization = `Bearer ${t}`;
  return headers;
}

async function errorFrom(res) {
  let body = {};
  try {
    body = await res.json();
  } catch {
    // not JSON
  }
  if (res.status === 401) {
    return new ApiError(body.message || '登录已过期，请重新登录', 401, body.reason || '');
  }
  return new ApiError(body.message || `请求失败（HTTP ${res.status}）`, res.status, body.reason || '');
}

/**
 * Calls /api/v1<path>. `json` is sent as a JSON body, `form` as multipart/form-data.
 * Returns `data` of the {code,message,data} envelope; throws ApiError with the server's message.
 */
export async function api(path, { method = 'GET', json, form, signal } = {}) {
  const headers = authHeaders({ Accept: 'application/json' });
  let body;
  if (json !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(json);
  } else if (form) {
    body = form;
  }
  let res;
  try {
    res = await fetch(`/api/v1${path}`, { method, headers, body, signal });
  } catch (error) {
    if (error && error.name === 'AbortError') throw error;
    throw new ApiError('网络连接失败，请稍后再试', 0);
  }
  if (!res.ok) throw await errorFrom(res);
  let data = {};
  try {
    data = await res.json();
  } catch {
    throw new ApiError(`服务器返回的内容无法解析（HTTP ${res.status}）`, res.status);
  }
  if (data.code !== undefined && data.code !== 0) {
    throw new ApiError(data.message || '请求失败', res.status, data.reason || '');
  }
  return data.data;
}

// --- 公众号 ---------------------------------------------------------------------------------------

export function wechatCheck(account) {
  return api('/editor/wechat/check', { method: 'POST', json: { appid: account.appid, secret: account.secret } });
}

/** kind: 'content' (→ {url}) or 'cover' (→ {media_id, url}). */
export function wechatUpload(account, kind, blob, filename) {
  const form = new FormData();
  form.append('appid', account.appid);
  form.append('secret', account.secret);
  form.append('kind', kind);
  form.append('file', blob, filename);
  return api('/editor/wechat/upload', { method: 'POST', form });
}

export function wechatDraft(account, article) {
  return api('/editor/wechat/draft', { method: 'POST', json: { appid: account.appid, secret: account.secret, article } });
}

// --- 导入 ----------------------------------------------------------------------------------------

export function importArticle(url) {
  return api('/editor/article/import', { method: 'POST', json: { url } });
}

const PROXY_HOSTS = ['mmbiz.qpic.cn', 'mmbiz.qlogo.cn'];

/** True for 公众号 image hosts (hotlink-protected; fetched through the server proxy). */
export function isWechatImageUrl(src) {
  try {
    const u = new URL(src, window.location.href);
    return (u.protocol === 'https:' || u.protocol === 'http:') && PROXY_HOSTS.includes(u.hostname);
  } catch {
    return false;
  }
}

/** Fetches a 公众号 image through GET /api/v1/editor/image and returns the Blob. */
export async function fetchProxiedImage(url) {
  let res;
  try {
    res = await fetch(`/api/v1/editor/image?url=${encodeURIComponent(url)}`, { headers: authHeaders() });
  } catch {
    throw new ApiError('网络连接失败，请稍后再试', 0);
  }
  if (!res.ok) throw await errorFrom(res);
  const type = res.headers.get('Content-Type') || '';
  if (type.includes('application/json')) throw await errorFrom(res);
  return res.blob();
}

// --- AI ------------------------------------------------------------------------------------------

/** The signed-in user's GPT-group keys: [{id, name, group}]. */
export function listKeys() {
  return api('/learn/keys');
}

/**
 * POST /editor/ai/text (SSE). onDelta gets each piece as it streams. Resolves when {"done":true}
 * arrives; throws ApiError on an {"error"} event or a non-stream error response.
 */
export async function streamAiText(body, onDelta, signal) {
  const headers = authHeaders({ 'Content-Type': 'application/json', Accept: 'text/event-stream' });
  let res;
  try {
    res = await fetch('/api/v1/editor/ai/text', { method: 'POST', headers, body: JSON.stringify(body), signal });
  } catch (error) {
    if (error && error.name === 'AbortError') throw error;
    throw new ApiError('网络连接失败，请稍后再试', 0);
  }
  const type = res.headers.get('Content-Type') || '';
  if (!res.ok || !type.includes('text/event-stream')) throw await errorFrom(res);

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  const handle = (chunk) => {
    for (const raw of chunk.split('\n')) {
      const line = raw.trim();
      if (!line.startsWith('data:')) continue;
      let ev;
      try {
        ev = JSON.parse(line.slice(5).trim());
      } catch {
        continue;
      }
      if (ev.error) throw new ApiError(ev.error, 502);
      if (typeof ev.delta === 'string' && ev.delta) onDelta(ev.delta);
    }
  };
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let i;
    while ((i = buf.indexOf('\n\n')) >= 0) {
      const chunk = buf.slice(0, i);
      buf = buf.slice(i + 2);
      handle(chunk);
    }
  }
  if (buf.trim()) handle(buf);
}

/** POST /editor/ai/image → {b64_json, mime}. */
export function aiImage(keyId, prompt, size) {
  return api('/editor/ai/image', { method: 'POST', json: { key_id: keyId, prompt, size } });
}
