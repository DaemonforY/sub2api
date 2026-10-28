import { describe, expect, it } from 'vitest'
import { calmBox, fitText, fitTitled, layoutColumns, posterSize, posterTitle, shihuaFontStack, splitColumns, textForAspect, titledWidth } from '../shihuaLayout'

const xiaoshitan =
  '从小丘西行百二十步，隔篁竹，闻水声，如鸣珮环，心乐之。伐竹取道，下见小潭，水尤清冽。全石以为底，近岸，卷石底以出，为坻，为屿，为嵁，为岩。青树翠蔓，蒙络摇缀，参差披拂。\n潭中鱼可百许头，皆若空游无所依。日光下澈，影布石上，佁然不动；俶尔远逝，往来翕忽。似与游者相乐。'

describe('shihua layout', () => {
  it('keeps one poem line per column when every line fits', () => {
    expect(layoutColumns('空山新雨后，天气晚来秋。\n明月松间照，清泉石上流。', 12)).toEqual(['空山新雨后，天气晚来秋。', '明月松间照，清泉石上流。'])
  })

  it('wraps prose at sentence ends and commas without losing or reordering characters', () => {
    for (const rows of [8, 12, 20, 27]) {
      const cols = layoutColumns(xiaoshitan, rows)
      expect(cols.join('')).toBe(xiaoshitan.replace(/\n/g, ''))
      for (const c of cols) expect(Array.from(c).length).toBeLessThanOrEqual(rows)
    }
  })

  it('never starts a column with a punctuation mark when hard-wrapping', () => {
    const cols = splitColumns('一二三四五六七八九十，甲乙丙。', 10)
    expect(cols.every((c) => !'，。'.includes(c[0]))).toBe(true)
    expect(cols.join('')).toBe('一二三四五六七八九十，甲乙丙。')
  })

  it('fits the largest cell that keeps all columns inside the box', () => {
    const { cell, columns } = fitText('床前明月光，\n疑是地上霜。\n举头望明月，\n低头思故乡。', 600, 900, 200)
    expect(columns).toHaveLength(4)
    expect(columns.length * cell * 1.4).toBeLessThanOrEqual(600)
    expect(cell * 6).toBeLessThanOrEqual(900)
    expect(cell).toBeGreaterThan(100)
    const capped = fitText('床前明月光，', 600, 900, 50)
    expect(capped.cell).toBe(50)
  })

  it('fits the title, author and text together, the title larger', () => {
    const text = '草长莺飞二月天，\n拂堤杨柳醉春烟。\n儿童散学归来早，\n忙趁东风放纸鸢。'
    const f = fitTitled(text, '村居', '〔清〕高鼎', 500, 800, 200)
    expect(f.titleColumns).toEqual(['村居'])
    expect(f.bylineColumns).toEqual(['〔清〕高鼎'])
    expect(f.columns).toHaveLength(4)
    expect(f.titleCell).toBeGreaterThan(f.cell)
    expect(f.bylineCell).toBeLessThan(f.cell)
    expect(titledWidth(f)).toBeLessThanOrEqual(500)
    expect(f.titleCell * 2).toBeLessThanOrEqual(800)
    // One size up would not fit.
    const bigger = fitTitled(text, '村居', '〔清〕高鼎', titledWidth(f) - 1, 800, 200)
    expect(bigger.cell).toBeLessThan(f.cell)
  })

  it('drops the note that tells same-named poems apart from the title', () => {
    expect(posterTitle('绝句（两个黄鹂鸣翠柳）')).toBe('绝句')
    expect(posterTitle('村居')).toBe('村居')
    expect(posterTitle('（其一）题')).toBe('（其一）题')
  })

  it('finds the blank area the prompt left at the top right', () => {
    const w = 64, h = 96
    const gray = new Float32Array(w * h)
    // Busy noise everywhere except a flat block at x 40–60, y 4–40.
    for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) gray[y * w + x] = ((x * 7 + y * 13) % 5) * 50
    for (let y = 4; y < 40; y++) for (let x = 40; x < 60; x++) gray[y * w + x] = 230
    const box = calmBox(gray, w, h, { x0: 0.45, y0: 0, x1: 1, y1: 0.6 })!
    expect(box).not.toBeNull()
    expect(box.x0).toBeGreaterThanOrEqual(40 / w)
    expect(box.x1).toBeLessThanOrEqual(60 / w)
    expect(box.y0).toBeGreaterThanOrEqual(4 / h)
    expect(box.y1).toBeLessThanOrEqual(40 / h)
    expect((box.x1 - box.x0) * w).toBeGreaterThan(10)
  })

  it('crops portrait aspects from the 2:3 picture and sizes landscape ones', () => {
    expect(posterSize('2x3', 2048, 3072)).toEqual({ width: 2048, height: 3072 })
    expect(posterSize('3x4', 2048, 3072)).toEqual({ width: 2048, height: 2731 })
    // 9:16 is extended upward instead of cropping the picture's sides
    expect(posterSize('9x16', 2048, 3072)).toEqual({ width: 2048, height: 3641 })
    expect(posterSize('1x1', 2048, 3072)).toEqual({ width: 2048, height: 2048 })
    expect(posterSize('16x9', 2048, 3072)).toEqual({ width: 4096, height: 2304 })
  })

  it('puts only the famous lines of a long text on narrow and landscape posters', () => {
    expect(textForAspect('9x16', xiaoshitan, '潭中鱼可百许头，皆若空游无所依。')).toBe('潭中鱼可百许头，皆若空游无所依。')
    expect(textForAspect('2x3', xiaoshitan, '潭中鱼可百许头')).toBe(xiaoshitan)
    expect(textForAspect('9x16', '床前明月光，疑是地上霜。', '床前明月光')).toBe('床前明月光，疑是地上霜。')
    expect(textForAspect('9x16', xiaoshitan, '')).toBe(xiaoshitan)
  })
})

describe('shihuaFontStack', () => {
  it('puts the chosen font first and always falls back to LXGW WenKai', () => {
    expect(shihuaFontStack('mashanzheng').startsWith("'Ma Shan Zheng', 'LXGW WenKai'")).toBe(true)
    expect(shihuaFontStack('zhimangxing').startsWith("'Zhi Mang Xing', 'LXGW WenKai'")).toBe(true)
    expect(shihuaFontStack('liujianmaocao').startsWith("'Liu Jian Mao Cao', 'LXGW WenKai'")).toBe(true)
    expect(shihuaFontStack('wenkai').startsWith("'LXGW WenKai', 'Kaiti SC'")).toBe(true)
    expect(shihuaFontStack()).toBe(shihuaFontStack('mashanzheng'))
  })
})
