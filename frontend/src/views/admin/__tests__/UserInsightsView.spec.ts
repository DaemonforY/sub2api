import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import UserInsightsView from '@/views/admin/UserInsightsView.vue'

const getOverview = vi.fn()
const getProfile = vi.fn()
const listEvents = vi.fn()
const runMonitor = vi.fn()
const shadowUser = vi.fn()
const unshadowUser = vi.fn()
const toggleStatus = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()

vi.mock('@/api/admin/userInsights', () => ({
  getOverview: (...a: unknown[]) => getOverview(...a),
  getProfile: (...a: unknown[]) => getProfile(...a),
  listEvents: (...a: unknown[]) => listEvents(...a),
  runMonitor: (...a: unknown[]) => runMonitor(...a),
  shadowUser: (...a: unknown[]) => shadowUser(...a),
  unshadowUser: (...a: unknown[]) => unshadowUser(...a)
}))
vi.mock('@/api/admin', () => ({ adminAPI: { users: { toggleStatus: (...a: unknown[]) => toggleStatus(...a) } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-chartjs', () => ({ Line: { template: '<div data-testid="chart" />' }, Bar: { template: '<div data-testid="chart" />' } }))

const route = reactive<{ query: Record<string, string> }>({ query: {} })
const replace = vi.fn((to: { query: Record<string, string> }) => {
  route.query = to.query
  return Promise.resolve()
})
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace }) }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const known = new Set([
    'admin.userInsights.clientClass.relay',
    'admin.userInsights.rules.trial_day1',
    'admin.userInsights.via.visitor'
  ])
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, p?: Record<string, unknown>) => (p ? `${key}${JSON.stringify(p)}` : key),
      te: (key: string) => known.has(key)
    })
  }
})

const ref = (id: number, extra: Record<string, unknown> = {}) => ({
  id,
  email: `u${id}@x.com`,
  username: '',
  status: 'active',
  created_at: '2026-10-01T00:00:00Z',
  shadowed: false,
  paid_cny: 0,
  spend_usd: 0,
  ...extra
})

const overview = {
  days: 30,
  since: '2026-09-15T00:00:00Z',
  segments: { inactive: 11, trial: 7, paying: 3, churned: 1 },
  signups: [{ date: '2026-10-14', inactive: 2, trial: 1, paying: 0, churned: 0 }],
  spend: [{ date: '2026-10-14', free_usd: 4.5, used_usd: 10, paid_cny: 30 }],
  clients: [{ class: 'relay', requests: 100, users: 2, cost_usd: 3.21 }],
  clusters: [
    { kind: 'visitor', key: 'vis-abc', users: [ref(48, { status: 'disabled' }), ref(50, { shadowed: true })], free_usd: 6, paid_cny: 0 }
  ],
  relay_users: [{ user: ref(60), user_agent: 'axonhub/1.0', requests: 80, cost_usd: 2 }]
}

const profile = {
  user: ref(50),
  role: 'user',
  signup_source: 'email',
  balance: 0,
  visitor_id: 'vis-abc',
  signup_ip_prefix: '1.2.3.0/24',
  trial_granted: true,
  web_prefixes: [],
  api_ips: [],
  clients: [{ class: 'relay', user_agent: 'axonhub/1.0', requests: 3, cost_usd: 1, last_seen: '2026-10-14T00:00:00Z' }],
  groups: [],
  models: [],
  spend: { total_usd: 6, free_sub_usd: 6, paid_sub_usd: 0, balance_usd: 0, paid_cny: 0, has_real_paying: false },
  payments: [],
  subscriptions: [],
  invite_up: [ref(7)],
  invite_down: [],
  linked: [{ user: ref(48, { status: 'disabled' }), via: ['visitor'] }],
  events: [{ id: 1, user_id: 50, rule: 'trial_day1', detail: 'used $5', created_at: '2026-10-14T00:00:00Z' }],
  risk_score: 75,
  risk_reasons: ['+25 与 1 个已封禁账号同浏览器/网段/IP/UA']
}

const mounted: { unmount: () => void }[] = []

function mountView() {
  const w = mount(UserInsightsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
        Pagination: true,
        teleport: true
      }
    }
  })
  mounted.push(w)
  return w
}

describe('UserInsightsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {}
  })

  afterEach(() => {
    mounted.splice(0).forEach((w) => w.unmount())
  })

  it('loads the 30-day overview with segments, clients, clusters and relay users', async () => {
    getOverview.mockResolvedValue(overview)
    const w = mountView()
    await flushPromises()
    expect(getOverview).toHaveBeenCalledWith(30)
    expect(w.get('[data-testid="ui-segments"]').text()).toContain('11')
    expect(w.get('[data-testid="ui-clients"]').text()).toContain('admin.userInsights.clientClass.relay')
    const clusters = w.get('[data-testid="ui-clusters"]')
    expect(clusters.text()).toContain('u48@x.com')
    expect(clusters.text()).toContain('admin.userInsights.badge.disabled')
    expect(clusters.text()).toContain('admin.userInsights.badge.shadowed')
    expect(clusters.find('a').attributes('data-to')).toBe(JSON.stringify({ query: { tab: 'profile', user: '48' } }))
    expect(w.get('[data-testid="ui-relay-users"]').text()).toContain('axonhub/1.0')
    expect(w.get('[data-testid="ui-spend"]').text()).toContain('¥30.00')

    getOverview.mockResolvedValue({ ...overview, days: 7 })
    await w.get('[data-testid="ui-range-7"]').trigger('click')
    await flushPromises()
    expect(getOverview).toHaveBeenLastCalledWith(7)
  })

  it('opens a profile from the URL and shadows the user with a reason', async () => {
    route.query = { tab: 'profile', user: '50' }
    getProfile.mockResolvedValue(profile)
    shadowUser.mockResolvedValue({})
    const w = mountView()
    await flushPromises()
    expect(getOverview).not.toHaveBeenCalled()
    expect(getProfile).toHaveBeenCalledWith('50')
    expect(w.get('[data-testid="ui-risk"]').text()).toContain('75')
    expect(w.get('[data-testid="ui-linked"]').text()).toContain('admin.userInsights.via.visitor')
    expect(w.get('[data-testid="ui-profile-events"]').text()).toContain('admin.userInsights.rules.trial_day1')
    expect(w.get('[data-testid="ui-invites"]').text()).toContain('u7@x.com')

    await w.get('[data-testid="ui-action-shadow"]').trigger('click')
    await flushPromises()
    // An empty reason is refused.
    const confirm = w.findAll('button').find((b) => b.text() === 'admin.userInsights.actions.shadow' && b.attributes('data-testid') === undefined)
    expect(confirm).toBeTruthy()
    await confirm!.trigger('click')
    expect(shadowUser).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalled()

    await w.get('[data-testid="ui-shadow-reason"]').setValue('same browser as #48')
    await confirm!.trigger('click')
    await flushPromises()
    expect(shadowUser).toHaveBeenCalledWith(50, 'same browser as #48')
    expect(getProfile).toHaveBeenCalledTimes(2)
  })

  it('bans through the admin users API', async () => {
    route.query = { tab: 'profile', user: 'u50@x.com' }
    getProfile.mockResolvedValue(profile)
    toggleStatus.mockResolvedValue({})
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-action-ban"]').trigger('click')
    await flushPromises()
    const confirm = w.findAll('button').find((b) => b.text() === 'admin.userInsights.actions.ban' && b.attributes('data-testid') === undefined)
    await confirm!.trigger('click')
    await flushPromises()
    expect(toggleStatus).toHaveBeenCalledWith(50, 'disabled')
    expect(getProfile).toHaveBeenLastCalledWith('50')
  })

  it('lists events and runs the monitor', async () => {
    route.query = { tab: 'events' }
    listEvents.mockResolvedValue({ items: [{ id: 3, user_id: 9, user_email: 'e@x.com', rule: 'trial_day1', detail: 'd', created_at: '2026-10-14T00:00:00Z' }], total: 1 })
    runMonitor.mockResolvedValue({ new_events: [{ id: 4 }, { id: 5 }] })
    const w = mountView()
    await flushPromises()
    expect(listEvents).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(w.get('[data-testid="ui-events"]').text()).toContain('e@x.com')

    await w.get('[data-testid="ui-run-monitor"]').trigger('click')
    await flushPromises()
    expect(runMonitor).toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('admin.userInsights.events.newEvents{"n":2}')
    expect(listEvents).toHaveBeenCalledTimes(2)
  })

  it('switching tabs writes the tab to the URL', async () => {
    getOverview.mockResolvedValue(overview)
    listEvents.mockResolvedValue({ items: [], total: 0 })
    const w = mountView()
    await flushPromises()
    await w.get('[data-testid="ui-tab-events"]').trigger('click')
    await flushPromises()
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'events' } })
    expect(listEvents).toHaveBeenCalled()
  })
})
