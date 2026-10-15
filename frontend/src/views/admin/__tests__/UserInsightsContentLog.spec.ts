import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import UserInsightsView from '@/views/admin/UserInsightsView.vue'

const getOverview = vi.fn()
const getProfile = vi.fn()
const listEvents = vi.fn()
const listContentLogTargets = vi.fn()
const enableContentLog = vi.fn()
const disableContentLog = vi.fn()
const listContentLog = vi.fn()
const listContentLogAccess = vi.fn()
const getContentLogConfig = vi.fn()
const updateContentLogConfig = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()

vi.mock('@/api/admin/userInsights', () => ({
  getOverview: (...a: unknown[]) => getOverview(...a),
  getProfile: (...a: unknown[]) => getProfile(...a),
  listEvents: (...a: unknown[]) => listEvents(...a),
  runMonitor: vi.fn(),
  shadowUser: vi.fn(),
  unshadowUser: vi.fn(),
  listContentLogTargets: (...a: unknown[]) => listContentLogTargets(...a),
  enableContentLog: (...a: unknown[]) => enableContentLog(...a),
  disableContentLog: (...a: unknown[]) => disableContentLog(...a),
  listContentLog: (...a: unknown[]) => listContentLog(...a),
  listContentLogAccess: (...a: unknown[]) => listContentLogAccess(...a),
  getContentLogConfig: (...a: unknown[]) => getContentLogConfig(...a),
  updateContentLogConfig: (...a: unknown[]) => updateContentLogConfig(...a)
}))
vi.mock('@/api/admin', () => ({ adminAPI: { users: { toggleStatus: vi.fn() } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-chartjs', () => ({ Line: { template: '<div />' }, Bar: { template: '<div />' } }))

const route = reactive<{ query: Record<string, string> }>({ query: {} })
const replace = vi.fn((to: { query: Record<string, string> }) => {
  route.query = to.query
  return Promise.resolve()
})
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key),
      te: () => false
    })
  }
})

const target = (id: number, extra: Record<string, unknown> = {}) => ({
  user_id: id,
  email: `u${id}@x.com`,
  username: '',
  note: 'suspected reseller',
  created_by: 1,
  created_by_email: 'admin@x.com',
  created_at: '2026-10-15T00:00:00Z',
  log_count: 12,
  last_logged_at: '2026-10-16T00:00:00Z',
  ...extra
})

const entry = (id: number, extra: Record<string, unknown> = {}) => ({
  id,
  created_at: '2026-10-16T01:00:00Z',
  user_id: 45,
  api_key_id: 3,
  group_id: 2,
  request_id: `req-${id}`,
  session_id: 'sess-a',
  model: 'gpt-5.4',
  inbound_endpoint: '/v1/responses',
  user_agent: 'axonhub/1.0 (linux; go1.24)',
  ip_address: '1.2.3.4',
  user_text: 'write a crawler',
  tool_calls: [],
  reply_head: 'Sure, here is',
  input_tokens: 100,
  output_tokens: 20,
  cache_read_tokens: 50,
  ...extra
})

const mounted: { unmount: () => void }[] = []

function mountView() {
  const w = mount(UserInsightsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { props: ['to'], template: '<a><slot /></a>' },
        Pagination: true,
        teleport: true
      }
    }
  })
  mounted.push(w)
  return w
}

function confirmButton(w: ReturnType<typeof mountView>, text: string) {
  return w.findAll('button').find((b) => b.text() === text && b.attributes('data-testid') === undefined)
}

describe('UserInsightsView content log', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
    listContentLogTargets.mockResolvedValue({ items: [target(45), target(46, { email: '', username: 'bob', last_logged_at: null })] })
    listContentLog.mockResolvedValue({ items: [], next_before_id: null, enabled: true, retention_days: 7 })
    listContentLogAccess.mockResolvedValue({ items: [] })
    getContentLogConfig.mockResolvedValue({ redact_secrets: true })
  })

  afterEach(() => {
    mounted.splice(0).forEach((w) => w.unmount())
  })

  it('lists the targets without reading any log until a user is picked', async () => {
    route.query = { tab: 'contentLog' }
    const w = mountView()
    await flushPromises()
    expect(getOverview).not.toHaveBeenCalled()
    expect(listContentLogTargets).toHaveBeenCalledTimes(1)
    expect(listContentLog).not.toHaveBeenCalled()
    expect(listContentLogAccess).not.toHaveBeenCalled()
    expect(w.get('[data-testid="ui-cl-notice"] p').text()).toBe('admin.userInsights.contentLog.notice')
    const targets = w.get('[data-testid="ui-cl-targets"]')
    expect(targets.text()).toContain('u45@x.com')
    expect(targets.text()).toContain('suspected reseller')
    expect(targets.text()).toContain('admin@x.com')
    expect(targets.text()).toContain('bob')
    expect(w.find('[data-testid="ui-cl-pick"]').exists()).toBe(true)

    await w.get('[data-testid="ui-cl-view-45"]').trigger('click')
    await flushPromises()
    expect(replace).toHaveBeenLastCalledWith({ query: { tab: 'contentLog', user_id: '45' } })
    expect(listContentLog).toHaveBeenCalledTimes(1)
    expect(listContentLog).toHaveBeenCalledWith({ user_id: 45, session_id: undefined, before_id: undefined, limit: 50 })
    expect(w.get('[data-testid="ui-cl-empty"]').text()).toBe('admin.userInsights.contentLog.empty')
  })

  it('opens from the URL, fetches the log once, collapses repeated questions and pages with before_id', async () => {
    vi.useFakeTimers()
    try {
      route.query = { tab: 'contentLog', user_id: '45' }
      listContentLog.mockResolvedValueOnce({
        items: [
          entry(10, { tool_calls: [{ name: 'shell', arguments: '{"cmd":"ls -la"}' }] }),
          entry(9, { reply_head: 'second' }),
          entry(8, { user_text: '', reply_head: 'after tool' }),
          entry(7, { session_id: 'sess-b' })
        ],
        next_before_id: 7,
        enabled: true,
        retention_days: 7
      })
      const w = mountView()
      await flushPromises()
      expect(listContentLog).toHaveBeenCalledTimes(1)
      expect(listContentLog).toHaveBeenCalledWith({ user_id: 45, session_id: undefined, before_id: undefined, limit: 50 })

      // No polling / auto refresh: nothing more is fetched as time passes.
      await vi.advanceTimersByTimeAsync(5 * 60 * 1000)
      expect(listContentLog).toHaveBeenCalledTimes(1)

      const r10 = w.get('[data-testid="ui-cl-row-10"]')
      expect(r10.get('[data-testid="ui-cl-question"]').text()).toBe('write a crawler')
      expect(r10.text()).toContain('axonhub/1.0')
      expect(r10.text()).not.toContain('linux; go1.24')
      expect(r10.get('[data-testid="ui-cl-tool"]').text()).toContain('shell')
      expect(r10.get('[data-testid="ui-cl-tool"]').text()).toContain('"cmd": "ls -la"')
      // Same session and same question as the row above: 同上.
      const r9 = w.get('[data-testid="ui-cl-row-9"]')
      expect(r9.find('[data-testid="ui-cl-same"]').exists()).toBe(true)
      expect(r9.find('[data-testid="ui-cl-question"]').exists()).toBe(false)
      // Empty text: tool result being sent back.
      expect(w.get('[data-testid="ui-cl-row-8"]').text()).toContain('admin.userInsights.contentLog.noText')
      // A different session repeats the question in full.
      const r7 = w.get('[data-testid="ui-cl-row-7"]')
      expect(r7.find('[data-testid="ui-cl-same"]').exists()).toBe(false)
      expect(r7.get('[data-testid="ui-cl-question"]').text()).toBe('write a crawler')

      listContentLog.mockResolvedValueOnce({ items: [entry(6, { session_id: 'sess-b' })], next_before_id: null, enabled: true, retention_days: 7 })
      await w.get('[data-testid="ui-cl-more"]').trigger('click')
      await flushPromises()
      expect(listContentLog).toHaveBeenLastCalledWith({ user_id: 45, session_id: undefined, before_id: 7, limit: 50 })
      expect(w.get('[data-testid="ui-cl-row-6"]').find('[data-testid="ui-cl-same"]').exists()).toBe(true)
      expect(w.find('[data-testid="ui-cl-more"]').exists()).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })

  it('filters by session when a session chip is clicked', async () => {
    route.query = { tab: 'contentLog', user_id: '45' }
    listContentLog.mockResolvedValue({ items: [entry(10)], next_before_id: null, enabled: true, retention_days: 7 })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-cl-session-chip"]').trigger('click')
    await flushPromises()
    expect(listContentLog).toHaveBeenCalledTimes(2)
    expect(listContentLog).toHaveBeenLastCalledWith({ user_id: 45, session_id: 'sess-a', before_id: undefined, limit: 50 })
    await w.get('[data-testid="ui-cl-session-clear"]').trigger('click')
    await flushPromises()
    expect(listContentLog).toHaveBeenLastCalledWith({ user_id: 45, session_id: undefined, before_id: undefined, limit: 50 })
  })

  it('enables a user from the form only after confirming', async () => {
    route.query = { tab: 'contentLog' }
    enableContentLog.mockResolvedValue({ enabled: true })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-cl-enable-user"]').setValue('77')
    await w.get('[data-testid="ui-cl-enable-note"]').setValue('relay UA')
    await w.get('[data-testid="ui-cl-enable-form"]').trigger('submit')
    await flushPromises()
    expect(enableContentLog).not.toHaveBeenCalled()
    await confirmButton(w, 'admin.userInsights.contentLog.enable')!.trigger('click')
    await flushPromises()
    expect(enableContentLog).toHaveBeenCalledWith(77, 'relay UA')
    expect(showSuccess).toHaveBeenCalledWith('admin.userInsights.contentLog.enabledToast')
    expect(listContentLogTargets).toHaveBeenCalledTimes(2)
    expect(listContentLog).not.toHaveBeenCalled()
  })

  it('rejects an invalid user id and shows backend errors', async () => {
    route.query = { tab: 'contentLog' }
    enableContentLog.mockRejectedValue({ message: 'user not found' })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-cl-enable-form"]').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.userInsights.contentLog.invalidUserId')

    await w.get('[data-testid="ui-cl-enable-user"]').setValue('999')
    await w.get('[data-testid="ui-cl-enable-form"]').trigger('submit')
    await flushPromises()
    await confirmButton(w, 'admin.userInsights.contentLog.enable')!.trigger('click')
    await flushPromises()
    expect(enableContentLog).toHaveBeenCalledWith(999, '')
    expect(showError).toHaveBeenLastCalledWith('user not found')
  })

  it('stops recording after confirming', async () => {
    route.query = { tab: 'contentLog' }
    disableContentLog.mockResolvedValue({ enabled: false })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-cl-stop-46"]').trigger('click')
    await flushPromises()
    expect(disableContentLog).not.toHaveBeenCalled()
    expect(w.text()).toContain('admin.userInsights.contentLog.disableMessage')
    await confirmButton(w, 'admin.userInsights.contentLog.stop')!.trigger('click')
    await flushPromises()
    expect(disableContentLog).toHaveBeenCalledWith(46)
    expect(listContentLogTargets).toHaveBeenCalledTimes(2)
  })

  it('loads the access trail only when expanded', async () => {
    route.query = { tab: 'contentLog', user_id: '45' }
    listContentLogAccess.mockResolvedValue({
      items: [
        { id: 1, admin_id: 1, admin_email: 'admin@x.com', action: 'view', target_user_id: 45, target_email: 'u45@x.com', detail: 'session=', ip_address: '9.9.9.9', created_at: '2026-10-16T00:00:00Z' }
      ]
    })
    const w = mountView()
    await flushPromises()
    expect(listContentLogAccess).not.toHaveBeenCalled()
    await w.get('[data-testid="ui-cl-access-toggle"]').trigger('click')
    await flushPromises()
    expect(listContentLogAccess).toHaveBeenCalledWith({ user_id: 45, limit: 100 })
    const access = w.get('[data-testid="ui-cl-access"]')
    expect(access.text()).toContain('admin@x.com')
    expect(access.text()).toContain('admin.userInsights.contentLog.access.actions.view')
    expect(access.text()).toContain('9.9.9.9')
  })

  it('turns secret masking off only after the confirm, and says so in the notice', async () => {
    route.query = { tab: 'contentLog' }
    updateContentLogConfig.mockResolvedValue({ redact_secrets: false })
    const w = mountView()
    await flushPromises()
    const box = w.get('[data-testid="ui-cl-redact"]')
    expect((box.element as HTMLInputElement).checked).toBe(true)
    await box.trigger('click')
    await flushPromises()
    expect(updateContentLogConfig).not.toHaveBeenCalled()
    expect(w.text()).toContain('admin.userInsights.contentLog.redactOffMessage')
    await confirmButton(w, 'common.confirm')!.trigger('click')
    await flushPromises()
    expect(updateContentLogConfig).toHaveBeenCalledWith({ redact_secrets: false })
    expect(w.get('[data-testid="ui-cl-notice"] p').text()).toBe('admin.userInsights.contentLog.noticeRaw')
    expect((w.get('[data-testid="ui-cl-redact"]').element as HTMLInputElement).checked).toBe(false)
    expect(listContentLog).not.toHaveBeenCalled()
  })
})
