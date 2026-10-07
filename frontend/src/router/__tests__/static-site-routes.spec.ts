import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('router static-site routes', () => {
  // /learn and /editor are static apps served by the backend: a redirect there after signing in
  // (router.push('/editor/')) must leave the SPA instead of rendering its 404 page.
  it.each([
    ['/editor/', 'EditorSite'],
    ['/editor', 'EditorSite'],
    ['/learn/codex/', 'LearnSite'],
  ])('resolves %s to the %s pass-through route', async (path, name) => {
    const { default: router } = await import('@/router')
    const route = router.resolve(path)
    expect(route.name).toBe(name)
    expect(route.meta.requiresAuth).toBe(false)
  })
})
