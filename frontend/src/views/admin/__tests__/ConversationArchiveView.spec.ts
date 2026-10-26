import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ConversationArchiveView from '@/views/admin/ConversationArchiveView.vue'

// The real API module runs against a mocked axios client, so URLs, params and
// the "empty secret is not sent" rule are checked end to end.
type Cfg = { params?: Record<string, unknown>; responseType?: string }
const get = vi.fn<(url: string, cfg?: Cfg) => Promise<{ data: unknown }>>()
const put = vi.fn<(url: string, body?: unknown) => Promise<{ data: unknown }>>()
const post = vi.fn<(url: string, body?: unknown) => Promise<{ data: unknown }>>()
vi.mock('@/api/client', () => ({
  apiClient: {
    get: (url: string, cfg?: Cfg) => get(url, cfg),
    put: (url: string, body?: unknown) => put(url, body),
    post: (url: string, body?: unknown) => post(url, body)
  }
}))

const showSuccess = vi.fn()
const showError = vi.fn()
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copied: false, copyToClipboard: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key) })
  }
})

const BASE = '/admin/conversation-archive'

const status = {
  queue_len: 2,
  queue_bytes: 2048,
  dropped_total: 0,
  uploaded_today: 10,
  last_upload_at: '2026-10-16T01:00:00Z',
  last_error: '',
  last_error_at: null
}

const config = (extra: Record<string, unknown> = {}) => ({
  enabled: false,
  endpoint: 'https://cos.ap-seoul.myqcloud.com',
  region: 'ap-seoul',
  bucket: 'archive-1250000000',
  prefix: 'archive/',
  access_key_id: 'AKID',
  secret_configured: true,
  force_path_style: false,
  retention_days: 180,
  sse: true,
  status,
  ...extra
})

const turn = (id: number, extra: Record<string, unknown> = {}) => ({
  id,
  created_at: `2026-10-16T0${id}:00:00Z`,
  request_id: `req-${id}`,
  model: 'gpt-6.1-sol',
  inbound_endpoint: '/v1/responses',
  user_agent: 'codex_cli_rs/0.40',
  ip_address: '1.2.3.4',
  input_tokens: 10,
  output_tokens: 5,
  cache_read_tokens: 0,
  reset: false,
  prefix_items: 0,
  new_items: [{ type: 'message', role: 'user', content: [{ type: 'input_text', text: `question ${id}` }] }],
  request_params: {},
  response: { status: 'completed', text: `answer ${id}` },
  images: [],
  ...extra
})

const session = (key: string, extra: Record<string, unknown> = {}) => ({
  session_key: key,
  user_id: 45,
  email: 'u45@x.com',
  first_at: '2026-10-15T00:00:00Z',
  last_at: '2026-10-16T00:00:00Z',
  turns: 3,
  models: ['gpt-6.1-sol'],
  endpoints: ['/v1/responses'],
  user_agent: 'codex_cli_rs/0.40 (Mac OS)',
  preview: 'hello',
  image_count: 0,
  input_tokens: 1,
  output_tokens: 1,
  ...extra
})

const wallImage = (id: number) => ({
  id,
  turn_id: id * 10,
  user_id: 45,
  email: 'u45@x.com',
  mime: 'image/png',
  bytes: 2048,
  created_at: '2026-10-16T01:00:00Z',
  model: 'gpt-image-2',
  inbound_endpoint: '/v1/images/generations',
  session_key: 'sess-a',
  request_id: `req-img-${id}`,
  prompt: `a cat ${id}`
})

let cfg = config()
let turnPages: Record<string, { items: unknown[]; next_before_id: number | null }> = {}
const createObjectURL = vi.fn((_b: Blob) => `blob:img-${createObjectURL.mock.calls.length}`)
const revokeObjectURL = vi.fn((_u: string) => {})

function setupRoutes() {
  get.mockImplementation(async (url: string, c?: Cfg) => {
    if (url === `${BASE}/config`) return { data: cfg }
    if (url === `${BASE}/stats`) return { data: { days: [{ day: '2026-10-15', turns: 4, users: 2, bytes: 4096, images: 1 }] } }
    if (url === `${BASE}/sessions`) return { data: { items: [session('sess-a'), session('sess-b')], next_before: null } }
    if (url === `${BASE}/sessions/turns`) {
      const p = c?.params || {}
      return { data: turnPages[`${p.session_key}:${p.before_id ?? ''}`] || { items: [], next_before_id: null } }
    }
    if (url === `${BASE}/turns/by-request`) return { data: { ...turn(7), user_id: 45, session_key: 'sess-a', email: 'u45@x.com' } }
    if (url === `${BASE}/images`) {
      const p = c?.params || {}
      return { data: p.before_id ? { items: [wallImage(3)], next_before_id: null } : { items: [wallImage(9), wallImage(8)], next_before_id: 8, models: ['gpt-image-2', 'gpt-5.6-sol'] } }
    }
    if (url.startsWith(`${BASE}/images/`)) return { data: new Blob(['x'], { type: 'image/png' }) }
    throw new Error(`unexpected GET ${url}`)
  })
  put.mockImplementation(async (_url: string, body?: unknown) => ({ data: { ...cfg, ...(body as object), secret_configured: true } }))
  post.mockImplementation(async () => ({ data: { ok: true } }))
}

const ConfirmStub = {
  props: ['show', 'message', 'title'],
  emits: ['confirm', 'cancel'],
  template: '<div v-if="show" class="confirm-stub"><p class="confirm-message">{{ message }}</p><button class="confirm-yes" @click="$emit(\'confirm\')">ok</button></div>'
}

function mountView(): VueWrapper {
  return mount(ConversationArchiveView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        ConfirmDialog: ConfirmStub,
        // stands in for the TOTP dialog: "verifies" with one click
        TotpStepUpDialog: {
          props: ['controller'],
          template: '<button v-if="controller.visible.value" class="stepup-ok" @click="controller.onVerified()">ok</button>'
        }
      }
    }
  })
}

const browseCalls = () => get.mock.calls.filter(([url]) => /\/sessions|\/turns/.test(url))
const turnOrder = (w: VueWrapper) =>
  w
    .findAll('[data-testid^="ca-turn-"]')
    .map((n) => n.attributes('data-testid') || '')
    .filter((id) => /^ca-turn-\d+$/.test(id))

beforeEach(() => {
  vi.clearAllMocks()
  cfg = config()
  turnPages = {}
  setupRoutes()
  Object.assign(URL, { createObjectURL, revokeObjectURL })
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ConversationArchiveView — settings', () => {
  it('loads config and stats only, and collapses the settings once enabled and configured', async () => {
    cfg = config({ enabled: true })
    const w = mountView()
    await flushPromises()
    expect(get).toHaveBeenCalledWith(`${BASE}/config`, undefined)
    expect(get).toHaveBeenCalledWith(`${BASE}/stats`, undefined)
    expect(browseCalls()).toHaveLength(0)
    expect(w.find('[data-testid="ca-status-chip"]').text()).toBe('admin.conversationArchive.status.enabled')
    expect(w.find('[data-testid="ca-settings-form"]').exists()).toBe(false)
    expect(w.find('[data-testid="ca-status-block"]').text()).toContain('"size":"2 KB"')
    await w.find('[data-testid="ca-settings-toggle"]').trigger('click')
    expect(w.find('[data-testid="ca-settings-form"]').exists()).toBe(true)
  })

  it('shows 未配置 without a secret and 上传出错 when the last error is newer than the last upload', async () => {
    cfg = config({ secret_configured: false })
    let w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="ca-status-chip"]').text()).toBe('admin.conversationArchive.status.unconfigured')
    w.unmount()

    cfg = config({ enabled: true, status: { ...status, last_error: '403 AccessDenied', last_error_at: '2026-10-16T02:00:00Z' } })
    w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="ca-status-chip"]').text()).toBe('admin.conversationArchive.status.error')
    expect(w.find('[data-testid="ca-last-error"]').text()).toContain('403 AccessDenied')
  })

  it('keeps the stored secret when the field is left empty', async () => {
    const w = mountView()
    await flushPromises()
    expect((w.find('[data-testid="ca-secret"]').element as HTMLInputElement).placeholder).toBe('admin.conversationArchive.settings.secretConfigured')
    await w.find('[data-testid="ca-prefix"]').setValue('conv/')
    await w.find('[data-testid="ca-settings-form"]').trigger('submit')
    await flushPromises()
    expect(put).toHaveBeenCalledTimes(1)
    const [url, body] = put.mock.calls[0]
    expect(url).toBe(`${BASE}/config`)
    expect(body).toMatchObject({ prefix: 'conv/', enabled: false, retention_days: 180 })
    expect(body).not.toHaveProperty('secret_access_key')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('sends a new secret when one is typed', async () => {
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-secret"]').setValue('new-secret')
    await w.find('[data-testid="ca-settings-form"]').trigger('submit')
    await flushPromises()
    expect(put.mock.calls[0][1]).toMatchObject({ secret_access_key: 'new-secret' })
  })

  it('asks for confirmation, repeating the red notice, before enabling', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="ca-red-notice"]').text()).toContain('"days":180')
    await w.find('[data-testid="ca-enabled"]').setValue(true)
    await w.find('[data-testid="ca-settings-form"]').trigger('submit')
    await flushPromises()
    expect(put).not.toHaveBeenCalled()
    const dialog = w.find('.confirm-stub')
    expect(dialog.exists()).toBe(true)
    expect(dialog.find('.confirm-message').text()).toBe('admin.conversationArchive.settings.notice{"days":180}')
    await dialog.find('.confirm-yes').trigger('click')
    await flushPromises()
    expect(put).toHaveBeenCalledTimes(1)
    expect(put.mock.calls[0][1]).toMatchObject({ enabled: true })
  })

  it('runs 测试连接 through step-up and retries after verification', async () => {
    post.mockRejectedValueOnce({ status: 403, code: 'STEP_UP_REQUIRED', message: 'step up' })
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-secret"]').setValue('')
    await w.find('[data-testid="ca-test"]').trigger('click')
    await flushPromises()
    expect(post).toHaveBeenCalledTimes(1)
    expect(post.mock.calls[0][0]).toBe(`${BASE}/config/test`)
    expect(post.mock.calls[0][1]).not.toHaveProperty('secret_access_key')
    expect(showError).not.toHaveBeenCalled()
    await w.find('.stepup-ok').trigger('click')
    await flushPromises()
    expect(post).toHaveBeenCalledTimes(2)
    expect(showSuccess).toHaveBeenCalledWith('admin.conversationArchive.settings.testOk')
  })

  it('shows failed_total, active and the lifecycle result when present', async () => {
    cfg = config({ status: { ...status, failed_total: 3, active: true, lifecycle_applied: false, lifecycle_error: 'AccessDenied: PutBucketLifecycle' } })
    let w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="ca-failed-total"]').text()).toBe('3')
    expect(w.find('[data-testid="ca-active"]').text()).toBe('admin.conversationArchive.statusBlock.activeYes')
    expect(w.find('[data-testid="ca-lifecycle-status"]').text()).toContain('AccessDenied')
    w.unmount()

    cfg = config()
    w = mountView()
    await flushPromises()
    expect(w.find('[data-testid="ca-failed-total"]').exists()).toBe(false)
    expect(w.find('[data-testid="ca-lifecycle-status"]').exists()).toBe(false)
  })

  it('rejects a retention outside 7..730', async () => {
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-retention"]').setValue(3)
    await w.find('[data-testid="ca-settings-form"]').trigger('submit')
    await flushPromises()
    expect(put).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.conversationArchive.settings.retentionInvalid')
  })
})

describe('ConversationArchiveView — browser', () => {
  it('fetches sessions only when the admin searches', async () => {
    const w = mountView()
    await flushPromises()
    expect(browseCalls()).toHaveLength(0)
    await w.find('[data-testid="ca-q"]').setValue('u45@x.com')
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    const calls = browseCalls()
    expect(calls).toHaveLength(1)
    const params = calls[0][1]?.params || {}
    expect(params).toMatchObject({ q: 'u45@x.com', limit: 30 })
    expect(typeof params.from).toBe('string')
    expect(typeof params.to).toBe('string')
    expect(Date.parse(params.to as string) - Date.parse(params.from as string)).toBeGreaterThanOrEqual(7 * 86400_000 - 3600_000)
    expect(w.findAll('[data-testid="ca-session-row"]')).toHaveLength(2)
    // no turns are fetched until a session is opened
    expect(get.mock.calls.some(([url]) => url.endsWith('/sessions/turns'))).toBe(false)
  })

  it('shows the transcript oldest first and prepends earlier turns', async () => {
    turnPages['sess-a:'] = { items: [turn(3), turn(2, { reset: true, instructions: 'You are Codex.' })], next_before_id: 2 }
    turnPages['sess-a:2'] = { items: [turn(1)], next_before_id: null }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    await w.findAll('[data-testid="ca-session-row"]')[0].trigger('click')
    await flushPromises()

    expect(turnOrder(w)).toEqual(['ca-turn-2', 'ca-turn-3'])
    const text = w.find('[data-testid="ca-transcript"]').text()
    expect(text.indexOf('question 2')).toBeLessThan(text.indexOf('answer 2'))
    expect(text.indexOf('answer 2')).toBeLessThan(text.indexOf('question 3'))

    // reset divider only on the reset turn; instructions as a collapsed 系统提示
    expect(w.find('[data-testid="ca-turn-2"] [data-testid="ca-reset"]').exists()).toBe(true)
    expect(w.find('[data-testid="ca-turn-3"] [data-testid="ca-reset"]').exists()).toBe(false)
    const sys = w.find('[data-testid="ca-turn-2"] [data-testid="ca-instructions"]')
    expect(sys.text()).toContain('admin.conversationArchive.transcript.systemPrompt')
    expect(sys.find('[data-testid="ca-block-body"]').exists()).toBe(false)

    await w.find('[data-testid="ca-load-earlier"]').trigger('click')
    await flushPromises()
    const turnCalls = get.mock.calls.filter(([url]) => url.endsWith('/sessions/turns'))
    expect(turnCalls).toHaveLength(2)
    expect(turnCalls[1][1]?.params).toMatchObject({ user_id: 45, session_key: 'sess-a', before_id: 2, limit: 20 })
    expect(turnOrder(w)).toEqual(['ca-turn-1', 'ca-turn-2', 'ca-turn-3'])
    expect(w.find('[data-testid="ca-load-earlier"]').exists()).toBe(false)
  })

  it('collapses tool calls and shows fetch errors inline', async () => {
    turnPages['sess-a:'] = {
      items: [
        turn(2, { fetch_error: 'NoSuchKey' }),
        turn(1, {
          new_items: [{ type: 'function_call_output', call_id: 'c1', output: 'secret output' }],
          response: { text: '', items: [{ type: 'function_call', name: 'shell', call_id: 'c2', arguments: '{"cmd":"ls"}' }] }
        })
      ],
      next_before_id: null
    }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    await w.findAll('[data-testid="ca-session-row"]')[0].trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="ca-turn-2"] [data-testid="ca-fetch-error"]').text()).toContain('NoSuchKey')
    const call = w.find('[data-testid="ca-turn-1"] [data-testid="ca-block-tool_call"]')
    expect(call.text()).toContain('shell')
    expect(call.find('[data-testid="ca-block-body"]').exists()).toBe(false)
    await call.find('[data-testid="ca-block-toggle"]').trigger('click')
    expect(call.find('[data-testid="ca-block-body"]').text()).toContain('"cmd": "ls"')
    expect(w.find('[data-testid="ca-turn-1"] [data-testid="ca-block-tool_output"] [data-testid="ca-block-body"]').exists()).toBe(false)
  })

  it('loads images as blobs and revokes the object URLs when the session changes', async () => {
    turnPages['sess-a:'] = { items: [turn(1, { images: [{ id: 9, mime: 'image/png', bytes: 10 }] })], next_before_id: null }
    turnPages['sess-b:'] = { items: [turn(5)], next_before_id: null }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    await w.findAll('[data-testid="ca-session-row"]')[0].trigger('click')
    await flushPromises()

    expect(get).toHaveBeenCalledWith(`${BASE}/images/9`, { responseType: 'blob' })
    expect(createObjectURL).toHaveBeenCalledTimes(1)
    const url = createObjectURL.mock.results[0].value as string
    const img = w.find('[data-testid="ca-image-9"] img')
    expect(img.attributes('src')).toBe(url)
    expect(url.startsWith('blob:')).toBe(true)

    await w.find('[data-testid="ca-image-9"]').trigger('click')
    expect(w.find('[data-testid="ca-image-modal"] img').attributes('src')).toBe(url)

    expect(revokeObjectURL).not.toHaveBeenCalled()
    await w.findAll('[data-testid="ca-session-row"]')[1].trigger('click')
    await flushPromises()
    expect(revokeObjectURL).toHaveBeenCalledWith(url)
    expect(w.find('[data-testid="ca-image-modal"]').exists()).toBe(false)
  })

  it('renders image_urls as plain http(s) text links that load nothing until clicked', async () => {
    turnPages['sess-a:'] = {
      items: [
        turn(1, {
          image_urls: [
            { url: 'https://img.example.com/a.png', source: 'response' },
            { url: 'javascript:alert(1)', source: 'response' },
            { url: 'data:image/png;base64,AAAA' }
          ]
        })
      ],
      next_before_id: null
    }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    await w.findAll('[data-testid="ca-session-row"]')[0].trigger('click')
    await flushPromises()
    const links = w.findAll('[data-testid="ca-image-url"]')
    expect(links).toHaveLength(1)
    expect(links[0].attributes('href')).toBe('https://img.example.com/a.png')
    expect(links[0].attributes('target')).toBe('_blank')
    expect(links[0].attributes('rel')).toBe('noopener noreferrer')
    expect(links[0].text()).toBe('https://img.example.com/a.png')
    // text links only: nothing in the turn loads the external URL by itself
    expect(w.find('[data-testid="ca-image-urls"] img').exists()).toBe(false)
    expect(w.findAll('img').some((n) => (n.attributes('src') || '').includes('img.example.com'))).toBe(false)
    expect(get.mock.calls.some(([url]) => url.includes('img.example.com'))).toBe(false)
    expect(createObjectURL).not.toHaveBeenCalled()
  })

  it('revokes image URLs on unmount', async () => {
    turnPages['sess-a:'] = { items: [turn(1, { images: [{ id: 9, mime: 'image/png', bytes: 10 }] })], next_before_id: null }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-search-form"]').trigger('submit')
    await flushPromises()
    await w.findAll('[data-testid="ca-session-row"]')[0].trigger('click')
    await flushPromises()
    const url = createObjectURL.mock.results[0].value as string
    w.unmount()
    expect(revokeObjectURL).toHaveBeenCalledWith(url)
  })

  it('finds a turn by request id and opens its session ending at that turn', async () => {
    turnPages['sess-a:8'] = { items: [turn(7), turn(6)], next_before_id: 6 }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-request-id"]').setValue('req-7')
    await w.find('[data-testid="ca-by-request-form"]').trigger('submit')
    await flushPromises()
    expect(get).toHaveBeenCalledWith(`${BASE}/turns/by-request`, { params: { request_id: 'req-7' } })
    expect(turnOrder(w)).toEqual(['ca-turn-6', 'ca-turn-7'])
    expect(w.find('[data-testid="ca-show-latest"]').exists()).toBe(true)
  })
})

describe('ConversationArchiveView — image wall', () => {
  // tiles load their thumbnail when they scroll into view: here every tile is "visible" at once
  const SetupObserver = globalThis.IntersectionObserver
  beforeEach(() => {
    globalThis.IntersectionObserver = class {
      constructor(private cb: IntersectionObserverCallback) {}
      observe() {
        this.cb([{ isIntersecting: true } as IntersectionObserverEntry], this as unknown as IntersectionObserver)
      }
      disconnect() {}
      unobserve() {}
    } as unknown as typeof IntersectionObserver
  })
  afterEach(() => {
    globalThis.IntersectionObserver = SetupObserver
  })

  const wallCalls = () => get.mock.calls.filter(([url]) => url === `${BASE}/images`)

  it('lists nothing until loaded, then shows thumbnails and the model filter', async () => {
    const w = mountView()
    await flushPromises()
    expect(wallCalls()).toHaveLength(0)
    expect(w.find('[data-testid="ca-wall-grid"]').exists()).toBe(false)

    await w.find('[data-testid="ca-wall-q"]').setValue('u45@x.com')
    await w.find('[data-testid="ca-wall-form"]').trigger('submit')
    await flushPromises()
    expect(wallCalls()).toHaveLength(1)
    const params = wallCalls()[0][1]?.params || {}
    expect(params).toMatchObject({ limit: 60, q: 'u45@x.com' })
    expect(params.from).toMatch(/Z$/)
    expect(params.to).toMatch(/Z$/)
    expect(params).not.toHaveProperty('model')
    expect(params).not.toHaveProperty('before_id')

    // thumbnails, not the originals
    expect(get).toHaveBeenCalledWith(`${BASE}/images/9/thumb`, { responseType: 'blob' })
    expect(get).toHaveBeenCalledWith(`${BASE}/images/8/thumb`, { responseType: 'blob' })
    expect(get).not.toHaveBeenCalledWith(`${BASE}/images/9`, expect.anything())
    expect(w.find('[data-testid="ca-wall-tile-9"] img').attributes('src')).toMatch(/^blob:/)
    const options = w.findAll('[data-testid="ca-wall-model"] option').map((o) => o.attributes('value'))
    expect(options).toEqual(['', 'gpt-image-2', 'gpt-5.6-sol'])

    // 加载更多 continues the same filter below the last id and keeps the model list
    await w.find('[data-testid="ca-wall-model"]').setValue('gpt-image-2')
    await w.find('[data-testid="ca-wall-more"]').trigger('click')
    await flushPromises()
    expect(wallCalls()[1][1]?.params).toMatchObject({ before_id: 8, q: 'u45@x.com' })
    expect(wallCalls()[1][1]?.params).not.toHaveProperty('model')
    expect(w.findAll('[data-testid^="ca-wall-tile-"]')).toHaveLength(3)
    expect(w.find('[data-testid="ca-wall-more"]').exists()).toBe(false)
    expect(w.findAll('[data-testid="ca-wall-model"] option')).toHaveLength(3)

    // a new search applies the model
    await w.find('[data-testid="ca-wall-form"]').trigger('submit')
    await flushPromises()
    expect(wallCalls()[2][1]?.params).toMatchObject({ model: 'gpt-image-2' })
  })

  it('opens the original with its prompt, steps through, and revokes on close', async () => {
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-wall-form"]').trigger('submit')
    await flushPromises()

    await w.find('[data-testid="ca-wall-tile-9"]').trigger('click')
    await flushPromises()
    expect(get).toHaveBeenCalledWith(`${BASE}/images/9`, { responseType: 'blob' })
    const detail = w.find('[data-testid="ca-wall-detail"]')
    expect(detail.find('[data-testid="ca-wall-prompt"]').text()).toBe('a cat 9')
    expect(detail.text()).toContain('gpt-image-2')
    expect(detail.text()).toContain('req-img-9')
    const original = w.find('[data-testid="ca-wall-detail-img"]').attributes('src') as string
    expect(w.find('[data-testid="ca-wall-prev"]').exists()).toBe(false)

    await w.find('[data-testid="ca-wall-next"]').trigger('click')
    await flushPromises()
    expect(revokeObjectURL).toHaveBeenCalledWith(original)
    expect(w.find('[data-testid="ca-wall-prompt"]').text()).toBe('a cat 8')
    const second = w.find('[data-testid="ca-wall-detail-img"]').attributes('src') as string

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(w.find('[data-testid="ca-wall-detail"]').exists()).toBe(false)
    expect(revokeObjectURL).toHaveBeenCalledWith(second)
  })

  it('opens the image session ending at its turn', async () => {
    turnPages['sess-a:91'] = { items: [turn(7), turn(6)], next_before_id: 6 }
    const w = mountView()
    await flushPromises()
    await w.find('[data-testid="ca-wall-form"]').trigger('submit')
    await flushPromises()
    await w.find('[data-testid="ca-wall-tile-9"]').trigger('click')
    await flushPromises()
    await w.find('[data-testid="ca-wall-open-session"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-testid="ca-wall-detail"]').exists()).toBe(false)
    expect(get).toHaveBeenCalledWith(`${BASE}/sessions/turns`, { params: { user_id: 45, session_key: 'sess-a', limit: 20, before_id: 91 } })
    expect(turnOrder(w)).toEqual(['ca-turn-6', 'ca-turn-7'])
    expect(w.find('[data-testid="ca-show-latest"]').exists()).toBe(true)
  })
})
