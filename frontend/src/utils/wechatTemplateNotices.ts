import type {
  WeChatTemplateNotice,
  WeChatTemplateNoticeEvent,
  WeChatTemplateNotices,
} from '@/api/admin/settings'

/**
 * 微信提醒 (公众号 template messages) settings: the stored JSON keyed by event ⇄ the admin form,
 * where field mappings are editable rows.
 */

export interface WeChatTemplateNoticeEventInfo {
  event: WeChatTemplateNoticeEvent
  defaultUrlPath: string
  /** Event-specific placeholders, on top of COMMON_TEMPLATE_PLACEHOLDERS. */
  placeholders: string[]
}

export const COMMON_TEMPLATE_PLACEHOLDERS = ['site_name', 'username', 'now']

export const WECHAT_TEMPLATE_NOTICE_EVENTS: WeChatTemplateNoticeEventInfo[] = [
  { event: 'balance_low', defaultUrlPath: '/purchase', placeholders: ['balance', 'threshold'] },
  { event: 'trial_exhausted', defaultUrlPath: '/purchase', placeholders: ['balance'] },
  {
    event: 'subscription_expiring',
    defaultUrlPath: '/purchase?tab=subscription',
    placeholders: ['plan_name', 'expires_at', 'days_left'],
  },
  {
    event: 'order_abandoned',
    defaultUrlPath: '/orders',
    placeholders: ['order_no', 'product', 'amount_cny', 'created_at'],
  },
]

export interface WeChatTemplateFieldRow {
  key: string
  value: string
}

export interface WeChatTemplateNoticeForm {
  event: WeChatTemplateNoticeEvent
  enabled: boolean
  template_id: string
  url_path: string
  fields: WeChatTemplateFieldRow[]
}

export const TEMPLATE_FIELD_KEY_PATTERN = /^[a-z_]+[0-9]+$/

export function isSafeSiteUrlPath(path: string): boolean {
  const value = path.trim()
  return (
    value.startsWith('/') &&
    !value.startsWith('//') &&
    !value.includes('://') &&
    !/[\s\\]/.test(value) &&
    value.length <= 512
  )
}

function parseNotices(raw: unknown): WeChatTemplateNotices {
  let value = raw
  if (typeof value === 'string') {
    try {
      value = value.trim() ? JSON.parse(value) : {}
    } catch {
      value = {}
    }
  }
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as WeChatTemplateNotices) : {}
}

/** One form per event, in display order, filled from the stored settings (or defaults). */
export function toTemplateNoticeForms(raw: unknown): WeChatTemplateNoticeForm[] {
  const notices = parseNotices(raw)
  return WECHAT_TEMPLATE_NOTICE_EVENTS.map(({ event, defaultUrlPath }) => {
    const stored = (notices[event] || {}) as Partial<WeChatTemplateNotice>
    const fields = stored.fields && typeof stored.fields === 'object' ? stored.fields : {}
    return {
      event,
      enabled: stored.enabled === true,
      template_id: typeof stored.template_id === 'string' ? stored.template_id : '',
      url_path: typeof stored.url_path === 'string' && stored.url_path.trim() ? stored.url_path : defaultUrlPath,
      fields: Object.entries(fields).map(([key, value]) => ({ key, value: String(value ?? '') })),
    }
  })
}

/** The JSON to save; blank field rows are dropped. */
export function fromTemplateNoticeForms(forms: WeChatTemplateNoticeForm[]): WeChatTemplateNotices {
  const out: WeChatTemplateNotices = {}
  for (const form of forms) {
    const fields: Record<string, string> = {}
    for (const row of form.fields) {
      const key = row.key.trim()
      if (!key && !row.value.trim()) continue
      fields[key] = row.value
    }
    out[form.event] = {
      enabled: form.enabled,
      template_id: form.template_id.trim(),
      url_path: form.url_path.trim(),
      fields,
    }
  }
  return out
}

export type TemplateNoticeProblem =
  | { event: WeChatTemplateNoticeEvent; kind: 'field_key'; key: string }
  | { event: WeChatTemplateNoticeEvent; kind: 'duplicate_key'; key: string }
  | { event: WeChatTemplateNoticeEvent; kind: 'url_path' }
  | { event: WeChatTemplateNoticeEvent; kind: 'template_id' }
  | { event: WeChatTemplateNoticeEvent; kind: 'no_fields' }

export function isTemplateFieldKeyInvalid(key: string): boolean {
  const value = key.trim()
  return value !== '' && !TEMPLATE_FIELD_KEY_PATTERN.test(value)
}

/** The first problem that blocks saving, or null. */
export function validateTemplateNoticeForms(forms: WeChatTemplateNoticeForm[]): TemplateNoticeProblem | null {
  for (const form of forms) {
    const seen = new Set<string>()
    let filled = 0
    for (const row of form.fields) {
      const key = row.key.trim()
      if (!key && !row.value.trim()) continue
      if (!TEMPLATE_FIELD_KEY_PATTERN.test(key)) return { event: form.event, kind: 'field_key', key }
      if (seen.has(key)) return { event: form.event, kind: 'duplicate_key', key }
      seen.add(key)
      filled += 1
    }
    if (form.url_path.trim() && !isSafeSiteUrlPath(form.url_path)) return { event: form.event, kind: 'url_path' }
    if (form.enabled) {
      if (!form.template_id.trim()) return { event: form.event, kind: 'template_id' }
      if (!form.url_path.trim()) return { event: form.event, kind: 'url_path' }
      if (filled === 0) return { event: form.event, kind: 'no_fields' }
    }
  }
  return null
}
