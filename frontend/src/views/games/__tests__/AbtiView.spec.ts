import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import AbtiView from '../AbtiView.vue'
import { answersForPattern } from '@/games/abti/score'
import { TYPES } from '@/games/abti/data'

const { route, replace, push, track, savePendingPrompt, auth } = vi.hoisted(() => ({
  route: { params: {} as Record<string, string>, path: '/games/abti' },
  replace: vi.fn(),
  push: vi.fn(),
  track: vi.fn(),
  savePendingPrompt: vi.fn(),
  auth: { isAuthenticated: false }
}))

vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => ({ replace, push }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, p?: Record<string, unknown>) => (p ? `${key} ${JSON.stringify(p)}` : key) })
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ publicSettingsLoaded: true, cachedPublicSettings: { signup_bonus: 2 } }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/utils/analytics', () => ({ track }))
vi.mock('@/utils/pendingPrompt', () => ({ savePendingPrompt }))

const stubs = { PlazaNavBar: true, ShareCardDialog: true, RouterLink: { template: '<a><slot /></a>' }, Transition: { template: '<div><slot /></div>' } }

function setRoute(code?: string) {
  route.params = code ? { code } : {}
  route.path = code ? `/games/abti/${code}` : '/games/abti'
}

async function mountView() {
  const w = mount(AbtiView, { global: { stubs } })
  await flushPromises()
  return w
}

// Answers the visible question with the option worth `value`, then lets the card move on.
async function answer(w: VueWrapper, value: number) {
  const q = w.get('[data-testid="abti-question"]').text()
  const opts = w.findAll('[data-testid="abti-option"]')
  await opts[value - 1].trigger('click')
  vi.advanceTimersByTime(300)
  await flushPromises()
  return q
}

describe('AbtiView', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    localStorage.clear()
    window.scrollTo = vi.fn() as unknown as typeof window.scrollTo
    setRoute()
    auth.isAuthenticated = false
    for (const fn of [replace, push, track, savePendingPrompt]) fn.mockReset()
    replace.mockImplementation(({ path }: { path: string }) => setRoute(path.split('/')[3]))
  })
  afterEach(() => vi.useRealTimers())

  it('plays through to the hidden type when the AI owns up, and remembers it', async () => {
    const w = await mountView()
    await w.get('[data-testid="abti-start"]').trigger('click')
    expect(track).toHaveBeenCalledWith('game_start', { game: 'abti' })
    let asked = 0
    while (w.find('[data-testid="abti-question"]').exists() && asked < 40) {
      const q = w.get('[data-testid="abti-question"]').text()
      await answer(w, q.includes('你是怎么答的') ? 3 : 2)
      asked++
    }
    expect(asked).toBe(32) // 30 + the gate + its follow-up
    expect(w.get('[data-testid="abti-code"]').text()).toBe('BOTX')
    expect(replace).toHaveBeenLastCalledWith({ path: '/games/abti/BOTX' })
    expect(track).toHaveBeenCalledWith('game_finish', { game: 'abti', type: 'BOTX', kind: 'hidden' })
    const saved = JSON.parse(localStorage.getItem('abti_result')!).answers
    expect(Object.values(saved).filter((v) => v === 2)).toHaveLength(31)

    // Coming back to the result's link shows the result again.
    const again = await mountView()
    expect(again.get('[data-testid="abti-code"]').text()).toBe('BOTX')
  })

  it('ignores a click on a card that has already moved on', async () => {
    const w = await mountView()
    await w.get('[data-testid="abti-start"]').trigger('click')
    const firstId = w.get('[data-testid="abti-option"]').attributes('data-q')
    await answer(w, 1)
    const progress = w.get('[data-testid="abti-progress"]').text()
    // A button of the card that slid out still carries the first question's id.
    const stale = w.findAll('[data-testid="abti-option"]')[0]
    stale.element.setAttribute('data-q', firstId!)
    await stale.trigger('click')
    vi.advanceTimersByTime(300)
    await flushPromises()
    expect(w.get('[data-testid="abti-progress"]').text()).toBe(progress)
    expect(w.findAll('[data-testid="abti-option"]').some((b) => b.classes().includes('ring-2'))).toBe(false)
  })

  it("shows a friend's result from the link above the start button", async () => {
    setRoute('PROM')
    const w = await mountView()
    expect(w.get('[data-testid="abti-friend"]').text()).toContain('提示词魔法师')
    expect(w.find('[data-testid="abti-result"]').exists()).toBe(false)
  })

  it('sends a bad code back to the test', async () => {
    setRoute('ABCD')
    await mountView()
    expect(replace).toHaveBeenCalledWith({ path: '/games/abti' })
  })

  it('keeps the drawing prompt and sends visitors through sign-up for the image step', async () => {
    const prom = TYPES.find((t) => t.code === 'PROM')!
    localStorage.setItem('abti_result', JSON.stringify({ answers: answersForPattern(prom.pattern!), at: 1 }))
    setRoute('PROM')
    const w = await mountView()
    await w.get('[data-testid="abti-cta"]').trigger('click')
    expect(savePendingPrompt).toHaveBeenCalledWith(expect.stringContaining('提示词魔法师'))
    expect(push).toHaveBeenCalledWith({ path: '/register', query: { redirect: '/first-image' } })
    expect(track).toHaveBeenCalledWith('game_cta', { game: 'abti', type: 'PROM', cta: 'image' })

    auth.isAuthenticated = true
    await w.get('[data-testid="abti-cta"]').trigger('click')
    expect(push).toHaveBeenLastCalledWith('/first-image')
  })
})
