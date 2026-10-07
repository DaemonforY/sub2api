import { describe, expect, it } from 'vitest'

// UsageView.spec.ts mocks xlsx; this runs the real SheetJS build through the
// same calls exportToExcel makes, so a version bump that changes the API fails here.
describe('UsageView xlsx export (real SheetJS)', () => {
  it('writes header and appended rows into a readable workbook', async () => {
    const XLSX = await import('xlsx')
    const ws = XLSX.utils.aoa_to_sheet([['time', 'user', 'cost']])
    XLSX.utils.sheet_add_aoa(ws, [['2026-10-07T00:00:00Z', 'a@example.com', '0.000123']], { origin: -1 })
    XLSX.utils.sheet_add_aoa(ws, [['2026-10-07T00:01:00Z', 'b@example.com', '1.500000']], { origin: -1 })
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, ws, 'Usage')

    const out = XLSX.write(wb, { bookType: 'xlsx', type: 'array' })
    expect(out).toBeInstanceOf(ArrayBuffer)

    const back = XLSX.read(new Uint8Array(out), { type: 'array' })
    expect(back.SheetNames).toEqual(['Usage'])
    expect(XLSX.utils.sheet_to_json(back.Sheets.Usage, { header: 1 })).toEqual([
      ['time', 'user', 'cost'],
      ['2026-10-07T00:00:00Z', 'a@example.com', '0.000123'],
      ['2026-10-07T00:01:00Z', 'b@example.com', '1.500000'],
    ])
  })
})
