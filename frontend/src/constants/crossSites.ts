/**
 * Cross-site links to companion products deployed alongside this gateway.
 * Kept in one place so URLs and UTM tagging stay consistent across views.
 */

/** 无限画布（infinite-canvas）：部署在子域名上的 AI 生图工作台，浏览器直连本网关的 /v1 接口 */
export const CANVAS_SITE_URL = 'https://canvas.hivegpt.cn/'

export interface CanvasUrlOptions {
  /** utm_medium，用于统计入口来源 */
  medium: string
  /**
   * 网关 API 根地址（不带 /v1）。传入后画布会自动创建一个指向本网关的渠道，
   * 用户只需再粘贴自己的 API Key。
   */
  baseUrl?: string
  /** 画布内页面：'/'（首页）、'/image'（生图工作台）、'/video'（视频工作台） */
  path?: '/' | '/image' | '/video'
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
