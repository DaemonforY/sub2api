import { describe, expect, it } from 'vitest'
import { isSiteUploadFile, nextSiteIsPaid, type MySitesQuota } from '../sites'

describe('site hosting helpers', () => {
  const quota = { free_sites: 1, free_used: 0, extra_price: 5 } as MySitesQuota

  it('charges for a site once the included ones are used', () => {
    expect(nextSiteIsPaid(quota)).toBe(false)
    expect(nextSiteIsPaid({ ...quota, free_used: 1 })).toBe(true)
    expect(nextSiteIsPaid({ ...quota, free_used: 1, extra_price: 0 })).toBe(false)
  })

  it('accepts html and zip uploads only', () => {
    expect(isSiteUploadFile(new File(['x'], 'Index.HTML'))).toBe(true)
    expect(isSiteUploadFile(new File(['x'], 'site.zip'))).toBe(true)
    expect(isSiteUploadFile(new File(['x'], 'shell.php'))).toBe(false)
  })
})
