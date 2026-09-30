import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { APP_MODULES, appModuleLink } from '../appModules'
import UserDashboardApps from '@/components/user/dashboard/UserDashboardApps.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('app modules', () => {
  it('lists the canvas first and links it with the gateway origin and entry UTM', () => {
    expect(APP_MODULES[0].key).toBe('canvas')
    const { href, to } = appModuleLink(APP_MODULES[0], 'sidebar')
    expect(to).toBeUndefined()
    const url = new URL(href!)
    expect(url.origin).toBe('https://canvas.hivegpt.cn')
    expect(url.searchParams.get('utm_source')).toBe('hivegpt')
    expect(url.searchParams.get('utm_medium')).toBe('sidebar')
    expect(url.searchParams.get('baseUrl')).toBe(window.location.origin)
    expect(url.searchParams.has('apiKey')).toBe(false)
  })

  it('renders external modules on the dashboard as new-tab links', () => {
    const wrapper = mount(UserDashboardApps, { global: { stubs: { RouterLink: true } } })
    const link = wrapper.get('[data-testid="dashboard-app-canvas"]')
    expect(link.element.tagName).toBe('A')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('href')).toContain('utm_medium=dashboard-apps')
  })
})
