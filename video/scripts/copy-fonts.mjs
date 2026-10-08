// Copies the open-licensed fonts the player offers into public/fonts and writes fonts.css:
// LXGW WenKai regular (OFL, split by unicode-range so pages only load the glyphs they use) and
// JetBrains Mono (OFL).
import { cpSync, mkdirSync, readFileSync, writeFileSync, readdirSync } from 'node:fs'

const dest = new URL('../public/fonts/', import.meta.url)
mkdirSync(new URL('wenkai/', dest), { recursive: true })
mkdirSync(new URL('mono/', dest), { recursive: true })
const wk = new URL('../node_modules/lxgw-wenkai-webfont/', import.meta.url)
for (const f of readdirSync(new URL('files/', wk))) {
  if (f.startsWith('lxgwwenkai-regular-subset-')) cpSync(new URL('files/' + f, wk), new URL('wenkai/' + f, dest))
}
cpSync(new URL('OFL.txt', wk), new URL('wenkai/OFL.txt', dest))
let css = readFileSync(new URL('lxgwwenkai-regular.css', wk), 'utf8').replaceAll('./files/', '/fonts/wenkai/')

const mono = new URL('../node_modules/@fontsource/jetbrains-mono/files/', import.meta.url)
for (const weight of [400, 700]) {
  const file = `jetbrains-mono-latin-${weight}-normal.woff2`
  cpSync(new URL(file, mono), new URL('mono/' + file, dest))
  css += `\n@font-face{font-family:'JetBrains Mono';font-style:normal;font-weight:${weight};font-display:swap;src:url('/fonts/mono/${file}') format('woff2')}\n`
}
writeFileSync(new URL('fonts.css', dest), css)
console.log('fonts.css written')
