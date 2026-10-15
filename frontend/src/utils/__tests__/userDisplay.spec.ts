import { describe, expect, it } from 'vitest'
import {
  displayEmail,
  displayIdentity,
  isPlaceholderEmail,
  placeholderEmailHint,
  placeholderEmailProvider,
} from '@/utils/userDisplay'

const zh: Record<string, string> = {
  'userDisplay.placeholderSecondary': '{provider} · 未绑定邮箱',
  'userDisplay.emailUnbound': '未绑定邮箱（{provider}）',
  'userDisplay.compact': '{name}（未绑定邮箱）',
  'userDisplay.login.wechat': '微信登录',
  'userDisplay.login.linuxdo': 'LinuxDo 登录',
  'userDisplay.login.other': '第三方登录',
  'userDisplay.unnamed.wechat': '微信用户',
  'userDisplay.unnamed.other': '第三方用户',
}
const t = (key: string, params?: Record<string, unknown>) =>
  (zh[key] ?? key).replace(/\{(\w+)\}/g, (_, name: string) => String(params?.[name] ?? ''))

const WECHAT = 'wechat-o0pO06labcdefghijklmn@wechat-connect.invalid'

describe('userDisplay', () => {
  it('recognises placeholder emails under the reserved .invalid TLD only', () => {
    expect(isPlaceholderEmail(WECHAT)).toBe(true)
    expect(isPlaceholderEmail('x@linuxdo-connect.invalid')).toBe(true)
    expect(isPlaceholderEmail('  X@SOMETHING.INVALID ')).toBe(true)
    expect(isPlaceholderEmail('alice@example.com')).toBe(false)
    expect(isPlaceholderEmail('invalid@example.com')).toBe(false)
    expect(isPlaceholderEmail('alice@invalid.example.com')).toBe(false)
    expect(isPlaceholderEmail('')).toBe(false)
    expect(isPlaceholderEmail(null)).toBe(false)
  })

  it('maps reserved domains to their sign-up provider', () => {
    expect(placeholderEmailProvider(WECHAT)).toBe('wechat')
    expect(placeholderEmailProvider('a@linuxdo-connect.invalid')).toBe('linuxdo')
    expect(placeholderEmailProvider('a@oidc-connect.invalid')).toBe('oidc')
    expect(placeholderEmailProvider('a@dingtalk-connect.invalid')).toBe('dingtalk')
    expect(placeholderEmailProvider('a@mystery.invalid')).toBe('other')
    expect(placeholderEmailProvider('alice@example.com')).toBeNull()
  })

  it('shows a WeChat-only user by username with 「微信登录 · 未绑定邮箱」', () => {
    expect(displayIdentity({ email: WECHAT, username: '小蜂' }, t)).toEqual({
      primary: '小蜂',
      secondary: '微信登录 · 未绑定邮箱',
      initial: '小',
      placeholder: true,
      provider: 'wechat',
    })
  })

  it('falls back to 「微信用户」 when the username is empty', () => {
    const identity = displayIdentity({ email: WECHAT, username: '  ' }, t)
    expect(identity.primary).toBe('微信用户')
    expect(identity.secondary).toBe('微信登录 · 未绑定邮箱')
  })

  it('names other providers from the domain', () => {
    expect(displayIdentity({ email: 'x@linuxdo-connect.invalid', username: 'ld' }, t).secondary).toBe(
      'LinuxDo 登录 · 未绑定邮箱',
    )
  })

  it('keeps real emails as they are', () => {
    expect(displayIdentity({ email: 'alice@example.com', username: 'alice' }, t)).toEqual({
      primary: 'alice@example.com',
      secondary: 'alice',
      initial: 'A',
      placeholder: false,
      provider: null,
    })
    expect(displayIdentity({ email: 'bob@example.com', username: '' }, t).initial).toBe('B')
  })

  it('gives short labels for email-only cells and read-only email fields', () => {
    expect(displayEmail(WECHAT, t, '小蜂')).toBe('小蜂（未绑定邮箱）')
    expect(displayEmail(WECHAT, t)).toBe('微信用户（未绑定邮箱）')
    expect(displayEmail('alice@example.com', t, 'alice')).toBe('alice@example.com')
    expect(placeholderEmailHint(WECHAT, t)).toBe('未绑定邮箱（微信登录）')
    expect(placeholderEmailHint('alice@example.com', t)).toBe('')
  })
})
