import { describe, expect, it } from 'vitest'
import {
  fromTemplateNoticeForms,
  isSafeSiteUrlPath,
  toTemplateNoticeForms,
  validateTemplateNoticeForms,
} from '@/utils/wechatTemplateNotices'

describe('wechatTemplateNotices', () => {
  it('fills one form per event with the default pages when nothing is stored', () => {
    const forms = toTemplateNoticeForms(null)
    expect(forms.map((form) => [form.event, form.url_path, form.enabled])).toEqual([
      ['balance_low', '/purchase', false],
      ['trial_exhausted', '/purchase', false],
      ['subscription_expiring', '/purchase?tab=subscription', false],
      ['order_abandoned', '/orders', false],
    ])
  })

  it('round-trips the stored JSON (object or string) and drops blank field rows', () => {
    const stored = {
      balance_low: {
        enabled: true,
        template_id: 'TPL-1',
        url_path: '/purchase?from=wx',
        fields: { thing1: '{username}', amount2: '${balance}' },
      },
    }
    const forms = toTemplateNoticeForms(JSON.stringify(stored))
    expect(forms[0]).toMatchObject({ enabled: true, template_id: 'TPL-1', url_path: '/purchase?from=wx' })
    expect(forms[0].fields).toEqual([
      { key: 'thing1', value: '{username}' },
      { key: 'amount2', value: '${balance}' },
    ])
    forms[0].fields.push({ key: '', value: '  ' })
    const saved = fromTemplateNoticeForms(forms)
    expect(saved.balance_low).toEqual(stored.balance_low)
    expect(saved.order_abandoned).toEqual({ enabled: false, template_id: '', url_path: '/orders', fields: {} })
  })

  it('accepts only same-site paths', () => {
    expect(isSafeSiteUrlPath('/purchase?tab=subscription')).toBe(true)
    expect(isSafeSiteUrlPath('https://evil.example')).toBe(false)
    expect(isSafeSiteUrlPath('//evil.example')).toBe(false)
    expect(isSafeSiteUrlPath('/a\\b')).toBe(false)
    expect(isSafeSiteUrlPath('purchase')).toBe(false)
  })

  it('reports bad field keys, duplicates, unsafe paths and incomplete enabled events', () => {
    const forms = toTemplateNoticeForms(null)
    expect(validateTemplateNoticeForms(forms)).toBeNull()

    forms[0].fields = [{ key: 'Thing1', value: 'x' }]
    expect(validateTemplateNoticeForms(forms)).toEqual({ event: 'balance_low', kind: 'field_key', key: 'Thing1' })

    forms[0].fields = [{ key: 'thing1', value: 'x' }, { key: 'thing1', value: 'y' }]
    expect(validateTemplateNoticeForms(forms)).toEqual({ event: 'balance_low', kind: 'duplicate_key', key: 'thing1' })

    forms[0].fields = [{ key: 'character_string2', value: 'x' }]
    forms[0].url_path = 'https://evil.example'
    expect(validateTemplateNoticeForms(forms)).toEqual({ event: 'balance_low', kind: 'url_path' })

    forms[0].url_path = '/purchase'
    forms[0].enabled = true
    expect(validateTemplateNoticeForms(forms)).toEqual({ event: 'balance_low', kind: 'template_id' })

    forms[0].template_id = 'TPL'
    forms[0].fields = []
    expect(validateTemplateNoticeForms(forms)).toEqual({ event: 'balance_low', kind: 'no_fields' })

    forms[0].fields = [{ key: 'thing1', value: '{username}' }]
    expect(validateTemplateNoticeForms(forms)).toBeNull()
  })
})
