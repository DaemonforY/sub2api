import { describe, expect, it } from 'vitest'
import { resolveDocumentTitle, resolveRouteDocumentTitle } from '@/router/title'

describe('resolveDocumentTitle', () => {
  it('路由存在标题时，使用“路由标题 - 站点名”格式', () => {
    expect(resolveDocumentTitle('Usage Records', 'My Site')).toBe('Usage Records - My Site')
  })

  it('路由无标题时，回退到站点名', () => {
    expect(resolveDocumentTitle(undefined, 'My Site')).toBe('My Site')
  })

  it('站点名为空时，回退默认站点名', () => {
    expect(resolveDocumentTitle('Dashboard', '')).toBe('Dashboard - Sub2API')
    expect(resolveDocumentTitle(undefined, '   ')).toBe('Sub2API')
  })

  it('站点名变更时仅影响后续路由标题计算', () => {
    const before = resolveDocumentTitle('Admin Dashboard', 'Alpha')
    const after = resolveDocumentTitle('Admin Dashboard', 'Beta')

    expect(before).toBe('Admin Dashboard - Alpha')
    expect(after).toBe('Admin Dashboard - Beta')
  })
})

describe('resolveRouteDocumentTitle', () => {
  it('自定义页面菜单加载后，使用菜单名称作为标题', () => {
    const route = {
      name: 'CustomPage',
      params: { id: 'scheduler' },
      meta: {
        title: 'Custom Page'
      }
    }

    expect(resolveRouteDocumentTitle(route, 'EzouAPI')).toBe('Custom Page - EzouAPI')
    expect(resolveRouteDocumentTitle(route, 'EzouAPI', [
      {
        id: 'scheduler',
        label: '账号调度器',
        icon_svg: '',
        url: 'https://example.com',
        visibility: 'admin',
        sort_order: 0
      }
    ])).toBe('账号调度器 - EzouAPI')
  })
})

describe('serverDocumentTitle', () => {
  it('首屏保留服务端为可收录页面写的 SEO 标题，其他页面照常', async () => {
    const { vi } = await import('vitest')
    vi.resetModules()
    document.head.innerHTML = '<meta name="robots" content="index,follow" />'
    document.title = '价格与套餐 | HiveGPT'
    window.history.replaceState(null, '', '/pricing/')
    const title = await import('@/router/title')

    expect(title.serverDocumentTitle('/pricing')).toBe('价格与套餐 | HiveGPT')
    expect(title.resolveRouteDocumentTitle({ name: 'Pricing', params: {}, meta: { title: 'Pricing' }, path: '/pricing' }, 'HiveGPT')).toBe('价格与套餐 | HiveGPT')
    expect(title.resolveRouteDocumentTitle({ name: 'Keys', params: {}, meta: { title: 'API Keys' }, path: '/keys' }, 'HiveGPT')).toBe('API Keys - HiveGPT')
  })

  it('noindex 页面不保留服务端标题', async () => {
    const { vi } = await import('vitest')
    vi.resetModules()
    document.head.innerHTML = '<meta name="robots" content="noindex,follow" />'
    document.title = 'HiveGPT'
    window.history.replaceState(null, '', '/dashboard')
    const title = await import('@/router/title')

    expect(title.serverDocumentTitle('/dashboard')).toBeNull()
  })
})
