import { describe, expect, it } from 'vitest'
import { isSiteUploadFile, nextSiteIsPaid, statBars, type MySitesQuota } from '../sites'

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

  it('scales stat bars to the busiest day', () => {
    expect(statBars([{ day: 'a', views: 0, visitors: 0, bytes: 0 }, { day: 'b', views: 5, visitors: 1, bytes: 0 }, { day: 'c', views: 10, visitors: 2, bytes: 0 }])).toEqual([0, 50, 100])
    expect(statBars([{ day: 'a', views: 0, visitors: 0, bytes: 0 }])).toEqual([0])
  })
})
