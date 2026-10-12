// 首屏 · 令牌河流: what a visitor types becomes tokens. They stream in from the left edge, above and
// below the words (never through them), into the demo card; honeycomb rings turn round the card like
// a reactor, and finished work flows out of its bottom edge on down the page.
import { AdditiveBlending, BufferAttribute, BufferGeometry, Camera, Points, Scene, ShaderMaterial, Vector2, Vector4, type IUniform } from 'three'
import { COLORS, rand, relRect, type MotionOptions, type SceneHandle } from './stage'
import { glyphAtlas } from './glyphs'

const PATH = /* glsl */ `
uniform float uTime, uDpr, uStacked;
uniform vec2 uRes;
uniform vec4 uText; // x0, y0, x1, y1 of the words (0..1, y up)
uniform vec4 uCard; // x0, y0, x1, y1 of the demo card
vec2 bez(vec2 a, vec2 b, vec2 c, vec2 d, float t) { float u = 1. - t; return u*u*u*a + 3.*u*u*t*b + 3.*u*t*t*c + t*t*t*d; }
// 1 inside the words' box (soft edge): anything straying there is dimmed.
float calmAt(vec2 p) {
  float d = max(max(uText.x - p.x, p.x - uText.z), max(uText.y - p.y, p.y - uText.w));
  return 1. - smoothstep(-0.02, 0.05, d);
}
// s.z picks the lane (above or below the words), s.w spreads it; ends on the card's near edge.
vec2 inflow(vec4 s, float t) {
  vec2 a, b, c, d;
  if (uStacked > 0.5) { // phone: words on top, card below — come in from both sides at the card's top
    float side = s.w < 0.5 ? -1. : 1.;
    float y = uCard.w + 0.01 + s.z * 0.07;
    a = vec2(side < 0. ? -0.05 : 1.05, y);
    b = vec2(side < 0. ? 0.2 : 0.8, y + 0.02);
    d = vec2(mix(uCard.x + 0.1, uCard.z - 0.1, fract(s.w * 2.)), uCard.w - 0.005);
    c = vec2(d.x, y + 0.01);
  } else {
    bool above = s.z < 0.5;
    float k = fract(s.z * 2.);
    float y = above ? mix(min(uText.w + 0.04, 0.97), 1.02, k) : mix(-0.02, max(uText.y - 0.04, 0.03), k);
    a = vec2(-0.05, y);
    b = vec2(uText.z * 0.55, y + (s.w - 0.5) * 0.03);
    c = vec2(uCard.x - 0.03, mix(y, (uCard.y + uCard.w) * 0.5, 0.55));
    d = vec2(uCard.x + 0.004, mix(uCard.y + 0.06, uCard.w - 0.06, s.w));
  }
  vec2 p = bez(a, b, c, d, t);
  p.y += sin(t * 8. + s.w * 6.283) * 0.005 * (1. - t);
  return p;
}
`

// Comets: K points per particle along its path, the head bright, the tail thinning out.
const COMET_VS = /* glsl */ `
attribute vec4 aSeed;
attribute float aTail; // 0 head … 1 end of tail
uniform vec3 uC0, uC1, uC2;
uniform float uMode; // 0 into the card, 1 out of it
varying float vAlpha;
varying vec3 vColor;
${PATH}
void main() {
  float t; vec2 p;
  if (uMode < 0.5) {
    t = fract(aSeed.x + uTime * (0.045 + 0.055 * aSeed.y)) - aTail * 0.05;
    p = inflow(aSeed, clamp(t, 0., 1.));
    vColor = mix(uC0, uC1, smoothstep(0.3, 0.95, t));
  } else {
    t = fract(aSeed.x + uTime * (0.05 + 0.06 * aSeed.y)) - aTail * 0.06;
    vec2 a = vec2(mix(uCard.x + 0.04, uCard.z - 0.04, aSeed.z), uCard.y);
    vec2 d = vec2(a.x + (aSeed.w - 0.5) * 0.5, -0.06);
    p = bez(a, a + vec2(0., -0.05), vec2(mix(a.x, d.x, 0.4), mix(a.y, d.y, 0.55)), d, clamp(t, 0., 1.));
    vColor = aSeed.y > 0.5 ? uC2 : uC1;
  }
  float tail = 1. - aTail;
  float fade = smoothstep(0., 0.07, t) * (1. - smoothstep(0.9, 1., t)) * step(0., t);
  vAlpha = fade * tail * tail * (0.75 + 0.25 * aSeed.y) * mix(1., 0.15, calmAt(p));
  gl_PointSize = (aTail < 0.01 ? 3. + 3. * aSeed.w : 1.8 + 1.8 * tail) * uDpr;
  gl_Position = vec4(p * 2. - 1., 0., 1.);
}`

const DOT_FS = /* glsl */ `
varying float vAlpha;
varying vec3 vColor;
void main() {
  float d = length(gl_PointCoord - 0.5);
  float a = (smoothstep(0.5, 0.0, d) * 0.55 + smoothstep(0.2, 0.0, d) * 0.7) * vAlpha;
  gl_FragColor = vec4(vColor * a, a);
}`

const GLYPH_VS = /* glsl */ `
attribute vec4 aSeed;
attribute float aGlyph;
uniform float uGrid;
uniform vec3 uC0, uC1;
varying float vAlpha;
varying vec2 vCell;
varying vec3 vColor;
${PATH}
void main() {
  float t = fract(aSeed.x + uTime * (0.02 + 0.024 * aSeed.y));
  vec2 p = inflow(aSeed, t);
  float fade = smoothstep(0., 0.1, t) * (1. - smoothstep(0.72, 0.96, t));
  vAlpha = fade * 0.9 * mix(1., 0.1, calmAt(p));
  vCell = vec2(mod(aGlyph, uGrid), floor(aGlyph / uGrid));
  vColor = mix(uC0, uC1, smoothstep(0.4, 0.95, t));
  gl_PointSize = (22. + 16. * aSeed.w) * (1. - 0.6 * smoothstep(0.55, 0.96, t)) * uDpr;
  gl_Position = vec4(p * 2. - 1., 0., 1.);
}`

const GLYPH_FS = /* glsl */ `
uniform sampler2D uAtlas;
uniform float uGrid;
varying float vAlpha;
varying vec2 vCell;
varying vec3 vColor;
void main() {
  vec2 uv = vec2((vCell.x + gl_PointCoord.x) / uGrid, 1. - (vCell.y + gl_PointCoord.y) / uGrid);
  float a = texture2D(uAtlas, uv).a * vAlpha;
  gl_FragColor = vec4(vColor * a, a);
}`

// Honeycomb rings round the card, packets running along them; the inner one shows through the glass.
const RING_VS = /* glsl */ `
attribute vec3 aHex; // perimeter position 0..1, ring 0..2, seed
uniform float uTime, uDpr, uRadius;
uniform vec2 uCenter, uRes;
uniform vec3 uC0, uC1;
uniform vec4 uText;
varying float vAlpha;
varying vec3 vColor;
vec2 corner(float k) { float a = 1.0471976 * k + 0.5235988; return vec2(cos(a), sin(a)); }
void main() {
  float dir = aHex.y == 1. ? -1. : 1.;
  float u = fract(aHex.x + dir * uTime * (0.01 + 0.006 * aHex.y));
  float e = floor(u * 6.), f = fract(u * 6.);
  vec2 hex = mix(corner(e), corner(e + 1.), f) * uRadius * (0.62 + 0.15 * aHex.y) * (1. + 0.015 * sin(uTime * 1.2 + aHex.y));
  vec2 p = uCenter + hex / uRes;
  float packet = pow(0.5 + 0.5 * sin(u * 43.98 - uTime * (2. + 0.6 * aHex.y)), 8.);
  float d = max(max(uText.x - p.x, p.x - uText.z), max(uText.y - p.y, p.y - uText.w));
  float calm = 1. - smoothstep(-0.02, 0.05, d);
  vAlpha = (0.28 + 0.72 * packet) * (1. - 0.2 * aHex.y) * mix(1., 0.12, calm);
  vColor = mix(uC0, uC1, packet);
  gl_PointSize = (1.6 + 2.6 * packet) * uDpr;
  gl_Position = vec4(p * 2. - 1., 0., 1.);
}`

// A faint field of specks for depth (allowed behind the words, at a whisper).
const DUST_VS = /* glsl */ `
attribute vec4 aSeed;
uniform float uTime, uDpr;
uniform vec3 uC0;
varying float vAlpha;
varying vec3 vColor;
void main() {
  vec2 p = vec2(fract(aSeed.x + uTime * 0.004 * (0.3 + aSeed.y)), aSeed.z);
  vAlpha = 0.12 + 0.18 * pow(0.5 + 0.5 * sin(uTime * (0.6 + aSeed.w) + aSeed.x * 40.), 3.);
  vColor = uC0;
  gl_PointSize = (1. + 1.4 * aSeed.w) * uDpr;
  gl_Position = vec4(p * 2. - 1., 0., 1.);
}`

type Fill = [name: string, size: number, fill: (a: Float32Array, i: number) => void]
type Uniforms = Record<string, IUniform>

function points(count: number, attrs: Fill[], vs: string, fs: string, uniforms: Uniforms) {
  const g = new BufferGeometry()
  g.setAttribute('position', new BufferAttribute(new Float32Array(count * 3), 3))
  for (const [name, size, fill] of attrs) {
    const a = new Float32Array(count * size)
    for (let i = 0; i < count; i++) fill(a, i)
    g.setAttribute(name, new BufferAttribute(a, size))
  }
  const m = new ShaderMaterial({ vertexShader: vs, fragmentShader: fs, uniforms, transparent: true, depthTest: false, depthWrite: false, blending: AdditiveBlending })
  const p = new Points(g, m)
  p.frustumCulled = false
  return p
}

const TAIL = 6
function comets(n: number, r: () => number, uniforms: Uniforms) {
  const seeds = Array.from({ length: n }, () => [r(), r(), r(), r()])
  return points(n * TAIL, [
    ['aSeed', 4, (a, i) => a.set(seeds[Math.floor(i / TAIL)], i * 4)],
    ['aTail', 1, (a, i) => { a[i] = (i % TAIL) / TAIL }],
  ], COMET_VS, DOT_FS, uniforms)
}

type Box = { x0: number; x1: number; y0: number; y1: number }
const union = (rs: Box[]): Box => ({ x0: Math.min(...rs.map((q) => q.x0)), x1: Math.max(...rs.map((q) => q.x1)), y0: Math.min(...rs.map((q) => q.y0)), y1: Math.max(...rs.map((q) => q.y1)) })

export interface RiverContext {
  host: HTMLElement
  /** the elements to keep clear (the copy column) */
  text: () => Element[]
  /** the demo card */
  card: () => Element | null
}

export function riverScene(o: MotionOptions, ctx: RiverContext): SceneHandle {
  const scene = new Scene()
  const camera = new Camera() // the shaders write clip space directly
  const r = rand(7)
  const k = o.density * (o.tier === 'mobile' ? 0.35 : 1)
  const shared = {
    uTime: { value: 0 }, uDpr: { value: 1 }, uRes: { value: new Vector2(1, 1) }, uStacked: { value: 0 },
    uText: { value: new Vector4(0, 0, 0, 0) }, uCard: { value: new Vector4(0.5, 0.2, 0.9, 0.8) },
  }
  const seed4: Fill = ['aSeed', 4, (a, i) => { a[i * 4] = r(); a[i * 4 + 1] = r(); a[i * 4 + 2] = r(); a[i * 4 + 3] = r() }]
  const colors = { uC0: { value: COLORS.ice }, uC1: { value: COLORS.cyan }, uC2: { value: COLORS.amber } }

  const dust = points(Math.round(500 * k), [seed4], DUST_VS, DOT_FS, { uTime: shared.uTime, uDpr: shared.uDpr, uC0: { value: COLORS.ice } })
  const inflow = comets(Math.round(700 * k), r, { ...shared, ...colors, uMode: { value: 0 } })
  const outflow = comets(Math.round(220 * k), r, { ...shared, ...colors, uMode: { value: 1 } })
  const atlas = glyphAtlas()
  const glyphs = points(Math.round(150 * k) + 16, [seed4, ['aGlyph', 1, (a, i) => { a[i] = Math.floor(r() * atlas.count) }]], GLYPH_VS, GLYPH_FS,
    { ...shared, uAtlas: { value: atlas.tex }, uGrid: { value: atlas.grid }, uC0: { value: COLORS.pale }, uC1: { value: COLORS.cyan } })
  const ringU = { uTime: shared.uTime, uDpr: shared.uDpr, uRes: shared.uRes, uText: shared.uText, uCenter: { value: new Vector2(0.75, 0.5) }, uRadius: { value: 300 }, uC0: { value: COLORS.blue }, uC1: { value: COLORS.cyan } }
  const rings = points(Math.round(2700 * Math.max(k, 0.5)), [['aHex', 3, (a, i) => { a[i * 3] = r(); a[i * 3 + 1] = i % 3; a[i * 3 + 2] = r() }]], RING_VS, DOT_FS, ringU)
  scene.add(dust, rings, inflow, outflow, glyphs)

  // where the words and the card are; re-measured now and then, since the page settles after load
  let measuredAt = -1
  function measure() {
    const cardEl = ctx.card(), textEls = ctx.text()
    if (!cardEl || !textEls.length) return
    const c = relRect(cardEl, ctx.host), tx = union(textEls.map((el) => relRect(el, ctx.host)))
    shared.uStacked.value = c.y1 < tx.y0 + 0.02 ? 1 : 0
    shared.uText.value.set(tx.x0 - 0.01, tx.y0 - 0.01, tx.x1 + 0.01, tx.y1 + 0.01)
    shared.uCard.value.set(c.x0, c.y0, c.x1, c.y1)
    ringU.uCenter.value.set(c.cx, c.cy)
    ringU.uRadius.value = Math.max(c.w, c.h) * 0.82
  }

  return {
    scene, camera,
    resize(w, h, dpr) {
      shared.uDpr.value = dpr
      shared.uRes.value.set(w, h)
      measure()
    },
    frame(t) {
      shared.uTime.value = t
      if (t - measuredAt > 1) { measuredAt = t; measure() }
    },
    settle() { shared.uTime.value = 23.5 },
  }
}
