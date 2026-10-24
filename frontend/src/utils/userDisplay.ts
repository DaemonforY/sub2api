/**
 * How to show a user whose email may be a synthetic placeholder.
 *
 * Third-party sign-ups without an email get an address under the reserved .invalid TLD, e.g.
 * wechat-<openid>@wechat-connect.invalid. Those are long, unreadable and never receive mail, so the
 * UI shows the username and "微信登录 · 未绑定邮箱" instead. Search and filters still use the real
 * stored value; only the rendering changes.
 */

import { reactive } from 'vue'

export type PlaceholderEmailProvider = 'wechat' | 'linuxdo' | 'oidc' | 'dingtalk' | 'github' | 'google' | 'other'

type Translate = (key: string, params?: Record<string, unknown>) => string

const PROVIDER_DOMAINS: Record<string, PlaceholderEmailProvider> = {
  'wechat-connect.invalid': 'wechat',
  'linuxdo-connect.invalid': 'linuxdo',
  'oidc-connect.invalid': 'oidc',
  'dingtalk-connect.invalid': 'dingtalk',
  'github-connect.invalid': 'github',
  'google-connect.invalid': 'google',
}

/** True for addresses under the reserved .invalid TLD (never deliverable). */
export function isPlaceholderEmail(email: string | null | undefined): boolean {
  const value = (email || '').trim().toLowerCase()
  if (!value.includes('@')) return false
  const domain = value.slice(value.lastIndexOf('@') + 1)
  return domain === 'invalid' || domain.endsWith('.invalid')
}

/** The sign-up provider a placeholder email came from; null for real emails. */
export function placeholderEmailProvider(email: string | null | undefined): PlaceholderEmailProvider | null {
  if (!isPlaceholderEmail(email)) return null
  const value = (email || '').trim().toLowerCase()
  const domain = value.slice(value.lastIndexOf('@') + 1)
  if (PROVIDER_DOMAINS[domain]) return PROVIDER_DOMAINS[domain]
  // Unknown reserved domain: guess from its first label (e.g. "wechat-xyz.invalid").
  const known = (Object.values(PROVIDER_DOMAINS) as PlaceholderEmailProvider[]).find((provider) => domain.startsWith(`${provider}`))
  return known || 'other'
}

// ---------------------------------------------------------------------------
// Username lookup for rows that only carry a placeholder email
// ---------------------------------------------------------------------------

type NameResolver = (emails: string[]) => Promise<Record<string, string>>

/** Lower-cased placeholder email -> username, filled by the resolver. Reactive so templates re-render. */
const resolvedNames = reactive(new Map<string, string>())
const requestedNames = new Set<string>()
let pendingNames: string[] = []
let flushTimer: ReturnType<typeof setTimeout> | null = null
let nameResolver: NameResolver | null = null

const NAME_LOOKUP_BATCH = 100
const NAME_LOOKUP_DELAY_MS = 30

/**
 * Lets displayEmail / displayIdentity look up usernames for placeholder emails shown without one
 * (admin tables that only carry user_email). Lookups are batched; each email is asked for once.
 * Pass null to turn it off.
 */
export function setPlaceholderNameResolver(resolver: NameResolver | null): void {
  nameResolver = resolver
}

/** Test helper: forget every looked-up name. */
export function resetPlaceholderNames(): void {
  resolvedNames.clear()
  requestedNames.clear()
  pendingNames = []
  if (flushTimer) clearTimeout(flushTimer)
  flushTimer = null
}

function lookedUpName(email: string): string {
  const key = email.trim().toLowerCase()
  const name = resolvedNames.get(key)
  if (name !== undefined) return name
  if (nameResolver && !requestedNames.has(key)) {
    // Only plain bookkeeping here: this runs during render, the reactive write happens after the request.
    requestedNames.add(key)
    pendingNames.push(email.trim())
    if (!flushTimer) flushTimer = setTimeout(flushNameLookups, NAME_LOOKUP_DELAY_MS)
  }
  return ''
}

async function flushNameLookups(): Promise<void> {
  flushTimer = null
  const resolver = nameResolver
  const batch = pendingNames.splice(0, NAME_LOOKUP_BATCH)
  if (pendingNames.length > 0) flushTimer = setTimeout(flushNameLookups, NAME_LOOKUP_DELAY_MS)
  if (!resolver || batch.length === 0) return
  try {
    const names = await resolver(batch)
    for (const [email, name] of Object.entries(names || {})) {
      if (name) resolvedNames.set(email.trim().toLowerCase(), name)
    }
  } catch {
    // Keep the generic label; a failed lookup is not retried on every render.
  }
}

export interface UserLike {
  email?: string | null
  username?: string | null
}

export interface DisplayIdentity {
  /** Main line: the email, or the username for placeholder users. */
  primary: string
  /** Second line: the username for real emails, "微信登录 · 未绑定邮箱" for placeholder users. */
  secondary: string
  /** Letter for avatars. */
  initial: string
  placeholder: boolean
  provider: PlaceholderEmailProvider | null
}

export function displayIdentity(user: UserLike | null | undefined, t: Translate): DisplayIdentity {
  const email = (user?.email || '').trim()
  const username = (user?.username || '').trim()
  const provider = placeholderEmailProvider(email)
  if (provider) {
    const primary = username || lookedUpName(email) || t(`userDisplay.unnamed.${provider}`)
    return {
      primary,
      secondary: t('userDisplay.placeholderSecondary', { provider: t(`userDisplay.login.${provider}`) }),
      initial: primary.charAt(0).toUpperCase() || 'U',
      placeholder: true,
      provider,
    }
  }
  const primary = email || username
  return {
    primary,
    secondary: email && username ? username : '',
    initial: (username || email).charAt(0).toUpperCase() || 'U',
    placeholder: false,
    provider: null,
  }
}

/**
 * One-line label for places that only have an email (tables showing user_email): placeholder
 * addresses become "<昵称>（未绑定邮箱）" — the username when passed, else the one the resolver looks
 * up (admin pages), else "微信用户（未绑定邮箱）".
 */
export function displayEmail(email: string | null | undefined, t: Translate, username?: string | null): string {
  const provider = placeholderEmailProvider(email)
  if (!provider) return (email || '').trim()
  const name = (username || '').trim() || lookedUpName(email || '') || t(`userDisplay.unnamed.${provider}`)
  return t('userDisplay.compact', { name })
}

/** Read-only text for an email field whose value is a placeholder: "未绑定邮箱（微信登录）". */
export function placeholderEmailHint(email: string | null | undefined, t: Translate): string {
  const provider = placeholderEmailProvider(email)
  if (!provider) return ''
  return t('userDisplay.emailUnbound', { provider: t(`userDisplay.login.${provider}`) })
}

/**
 * The name the server gives a WeChat sign-up when WeChat returned no nickname (QR / code logins
 * only get an openid): 「微信用户」 or 「微信用户_」 + up to 4 letters/digits. Matches wechatFallbackUsername.
 */
const WECHAT_FALLBACK_USERNAME = /^微信用户(_[a-z0-9]{1,4})?$/

export function isWeChatFallbackUsername(username: string | null | undefined): boolean {
  return WECHAT_FALLBACK_USERNAME.test((username || '').trim())
}
