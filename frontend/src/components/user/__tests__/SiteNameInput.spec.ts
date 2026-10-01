import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const checkSiteName = vi.fn()
vi.mock('@/api/sites', async () => {
  const actual = await vi.importActual<typeof import('@/api/sites')>('@/api/sites')
  return { ...actual, checkSiteName: (...args: unknown[]) => checkSiteName(...args) }
})

import SiteNameInput from '../SiteNameInput.vue'

describe('SiteNameInput', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    checkSiteName.mockReset()
  })
  afterEach(() => vi.useRealTimers())

  const lastValid = (w: ReturnType<typeof mount>) => (w.emitted('update:valid') || []).at(-1)?.[0]

  it('treats an empty name as valid when creating and lower-cases input', async () => {
    const w = mount(SiteNameInput, { props: { modelValue: '', domain: 's.example.test' } })
    expect(lastValid(w)).toBe(true)
    await w.find('input').setValue('My-Shop')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['my-shop'])
  })

  it('rejects malformed names without asking the server', async () => {
    const w = mount(SiteNameInput, { props: { modelValue: '1ab', domain: 's.example.test' } })
    expect(w.text()).toContain('sites.name.rule')
    expect(lastValid(w)).toBe(false)
    expect(checkSiteName).not.toHaveBeenCalled()
  })

  it('shows the server reason when a name is taken, and accepts a free one', async () => {
    checkSiteName.mockResolvedValueOnce({ name: 'taken-one', available: false, reason: '已被使用' })
    const w = mount(SiteNameInput, { props: { modelValue: 'taken-one', domain: 's.example.test', siteId: 7 } })
    expect(w.text()).toContain('sites.name.checking')
    await vi.advanceTimersByTimeAsync(400)
    await flushPromises()
    expect(checkSiteName).toHaveBeenCalledWith('taken-one', 7)
    expect(w.text()).toContain('已被使用')
    expect(lastValid(w)).toBe(false)

    checkSiteName.mockResolvedValueOnce({ name: 'free-one', available: true })
    await w.setProps({ modelValue: 'free-one' })
    await vi.advanceTimersByTimeAsync(400)
    await flushPromises()
    expect(w.text()).toContain('sites.name.available')
    expect(lastValid(w)).toBe(true)
  })

  it('keeping the current name is not a change', async () => {
    const w = mount(SiteNameInput, { props: { modelValue: 'mine', domain: 's.example.test', current: 'mine' } })
    expect(lastValid(w)).toBe(false)
    expect(checkSiteName).not.toHaveBeenCalled()
  })
})
