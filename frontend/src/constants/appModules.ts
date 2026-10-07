/**
 * AI 应用模块注册表：用户后台侧边栏「AI 应用」分组和仪表盘「AI 应用」卡片都从这里读取。
 * 以后新增模块（站内页面或兄弟站点）只需在 APP_MODULES 里追加一项。
 */
import { canvasUrl } from './crossSites'

export interface AppModule {
  /** 稳定标识，用作 key / data-testid */
  key: string
  /** i18n key：名称与一句话介绍 */
  labelKey: string
  descriptionKey: string
  /** 图标：public 目录下的图片路径 */
  iconUrl: string
  /** 站内路由（二选一） */
  to?: string
  /** 站外地址（二选一），在新标签页打开；medium 用于 UTM 统计入口 */
  href?: (medium: string) => string
  /** 可选角标 i18n key，如「新」 */
  badgeKey?: string
}

export const APP_MODULES: AppModule[] = [
  {
    key: 'canvas',
    labelKey: 'appModules.canvas.name',
    descriptionKey: 'appModules.canvas.description',
    iconUrl: '/apps/canvas.svg',
    // 带上本站网关地址：画布会预建指向本站的渠道，再引导用户一键授权 Key。
    href: (medium) => canvasUrl({ medium, baseUrl: typeof window !== 'undefined' ? window.location.origin : '' }),
    badgeKey: 'appModules.badgeNew'
  },
  {
    key: 'editor',
    labelKey: 'appModules.editor.name',
    descriptionKey: 'appModules.editor.description',
    iconUrl: '/apps/editor.svg',
    // 公众号排版是同域的独立页面（/editor/，不是 SPA 路由），所以走 href。
    href: (medium) => `/editor/?utm_source=hivegpt&utm_medium=${encodeURIComponent(medium)}`,
    badgeKey: 'appModules.badgeNew'
  }
]

export function appModuleLink(module: AppModule, medium: string): { to?: string; href?: string } {
  if (module.to) return { to: module.to }
  return { href: module.href ? module.href(medium) : '' }
}
