import { i18n } from '@/i18n'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { CustomMenuItem } from '@/types'

/**
 * 统一生成页面标题，避免多处写入 document.title 产生覆盖冲突。
 * 优先使用 titleKey 通过 i18n 翻译，fallback 到静态 routeTitle。
 */
export function resolveDocumentTitle(routeTitle: unknown, siteName?: string, titleKey?: string): string {
  const normalizedSiteName = typeof siteName === 'string' && siteName.trim() ? siteName.trim() : 'Sub2API'

  if (typeof titleKey === 'string' && titleKey.trim()) {
    const translated = i18n.global.t(titleKey)
    if (translated && translated !== titleKey) {
      return `${translated} - ${normalizedSiteName}`
    }
  }

  if (typeof routeTitle === 'string' && routeTitle.trim()) {
    return `${routeTitle.trim()} - ${normalizedSiteName}`
  }

  return normalizedSiteName
}

/**
 * 服务端为可收录的公开页写了 SEO 标题（backend/internal/web/seo.go）。首屏保留它，
 * 搜索引擎渲染后看到的仍是这个标题；站内跳转到别的页面后再用路由标题。
 */
function readServerTitle(): { path: string; title: string } | null {
  if (typeof document === 'undefined' || typeof location === 'undefined') return null
  const robots = document.querySelector('meta[name="robots"]')?.getAttribute('content') || ''
  if (!robots.startsWith('index') || !document.title.trim()) return null
  return { path: normalizePath(location.pathname), title: document.title }
}

function normalizePath(path: string): string {
  const trimmed = path.replace(/\/+$/, '')
  return trimmed === '' || trimmed === '/home' ? '/' : trimmed
}

const serverTitle = readServerTitle()

/** The server's SEO title when route is the page the server rendered. */
export function serverDocumentTitle(path?: string): string | null {
  if (!serverTitle || path === undefined) return null
  return normalizePath(path) === serverTitle.path ? serverTitle.title : null
}

export function resolveRouteDocumentTitle(
  route: Pick<RouteLocationNormalizedLoaded, 'name' | 'params' | 'meta'> & { path?: string },
  siteName: string | undefined,
  customMenuItems: CustomMenuItem[] = [],
): string {
  const fromServer = serverDocumentTitle(route.path)
  if (fromServer) return fromServer

  const id = typeof route.params.id === 'string' ? route.params.id : ''
  const menuItem = route.name === 'CustomPage' && id
    ? customMenuItems.find((item) => item.id === id)
    : undefined
  const menuTitle = menuItem?.label.trim()

  return resolveDocumentTitle(menuTitle || route.meta.title, siteName, menuTitle ? undefined : route.meta.titleKey as string)
}
