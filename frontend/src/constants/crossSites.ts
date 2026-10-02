/**
 * Cross-site links to companion products deployed alongside this gateway.
 * Kept in one place so URLs and UTM tagging stay consistent across views.
 */

/** 无限画布（infinite-canvas）：部署在子域名上的 AI 生图工作台，浏览器直连本网关的 /v1 接口 */
// VITE_CANVAS_SITE_URL overrides it for local development (e.g. http://localhost:3100/).
export const CANVAS_SITE_URL = (import.meta.env.VITE_CANVAS_SITE_URL as string | undefined) || 'https://canvas.hivegpt.cn/'

// Prompt-library cover hosts that are unreachable from some networks (e.g. mainland China). The
// canvas relays them at /img-proxy/<host>/<path> (allowlisted in its nginx.conf; keep in sync).
const CANVAS_PROXIED_IMAGE_HOSTS = new Set(['raw.githubusercontent.com', 'pbs.twimg.com', 'cms-assets.youmind.com', 'cdn.imgedify.com', 'bibigpt-apps.chatvid.ai', 'cdn.jsdelivr.net'])

/** Loads an allowlisted overseas image through the canvas relay; other URLs are returned unchanged. */
export function canvasProxiedImageUrl(url: string): string {
  if (!url) return url
  try {
    const parsed = new URL(url)
    if (parsed.protocol !== 'https:' || parsed.port || !CANVAS_PROXIED_IMAGE_HOSTS.has(parsed.host)) return url
    return `${CANVAS_SITE_URL.replace(/\/+$/, '')}/img-proxy/${parsed.host}${parsed.pathname}${parsed.search}`
  } catch {
    return url
  }
}

export interface CanvasUrlOptions {
  /** utm_medium，用于统计入口来源 */
  medium: string
  /**
   * 网关 API 根地址（不带 /v1）。传入后画布会自动创建一个指向本网关的渠道，
   * 用户只需再粘贴自己的 API Key。
   */
  baseUrl?: string
  /** 画布内页面：'/'（首页）、'/image'（生图工作台）、'/video'、'/tools'、'/explore'（发现）、'/w/12'（作品）… */
  path?: '/' | `/${string}`
  /** 预填到生图工作台的提示词（仅 path 为 /image 时生效） */
  prompt?: string
}

export function canvasUrl({ medium, baseUrl, path = '/', prompt }: CanvasUrlOptions): string {
  const params = new URLSearchParams({ utm_source: 'hivegpt', utm_medium: medium })
  const root = (baseUrl || '').trim().replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  if (root) params.set('baseUrl', root)
  const text = (prompt || '').trim()
  if (text && path === '/image') params.set('prompt', text.slice(0, 1000))
  const base = CANVAS_SITE_URL.replace(/\/+$/, '')
  return `${base}${path === '/' ? '/' : path}?${params.toString()}`
}

/**
 * 合作站点（同一运营方的兄弟站）：用户也可以在这些站点购买 Key。主站仍是本站。
 * 以后扩展新站点只需在这里追加一项。
 */
export interface PartnerSite {
  key: string
  name: string
  url: string
}

export const PARTNER_SITES: PartnerSite[] = [{ key: 'gorustai', name: 'GoRustAI', url: 'https://gorustai.com' }]

/** 当前站点之外的合作站点（同一套前端部署到合作站点时不会链接到自己）。 */
export function otherPartnerSites(currentHost = typeof window !== 'undefined' ? window.location.host : ''): PartnerSite[] {
  return PARTNER_SITES.filter((site) => new URL(site.url).host !== currentHost)
}

export function partnerSiteUrl(site: PartnerSite, path: string, medium: string): string {
  const url = new URL(path.startsWith('/') ? path : `/${path}`, `${site.url}/`)
  url.searchParams.set('utm_source', 'hivegpt')
  url.searchParams.set('utm_medium', medium)
  return url.toString()
}
