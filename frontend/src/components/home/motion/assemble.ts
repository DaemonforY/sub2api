// 收尾 · 令牌成字: drifting tokens fly together into the slogan when the last screen scrolls in, then
// keep a faint shimmer with a band of light sweeping across; the pointer pushes them apart a little.
// The words are also in the page for screen readers; the button is plain HTML under the particles.
import { AdditiveBlending, BufferAttribute, BufferGeometry, Camera, Color, Points, Scene, ShaderMaterial, Vector2 } from 'three'
import { COLORS, rand, type MotionOptions, type SceneHandle } from './stage'
import { FONT } from './glyphs'

const VS = /* glsl */ `
attribute vec2 aStart, aTarget;
attribute vec4 aSeed;
attribute float aText, aLine;
uniform float uTime, uProgress, uDpr, uMouseOn, uDot;
uniform vec2 uRes, uMouse, uGrad;
uniform vec3 uC0, uC1, uC2, uC3;
varying vec3 vColor;
varying float vAlpha;
void main() {
  float st = clamp(uProgress * 1.4 - aSeed.x * 0.4, 0., 1.);
  float e = 1. - pow(1. - st, 3.);
  vec2 drift = aStart + vec2(uTime * (12. + 30. * aSeed.y), sin(uTime * 0.6 + aSeed.z * 6.283) * 18.);
  drift.x = mod(drift.x + 100., uRes.x + 200.) - 100.;
  vec2 jitter = vec2(sin(uTime * 1.9 + aSeed.w * 40.), cos(uTime * 1.6 + aSeed.z * 30.)) * 0.6;
  vec2 p = mix(drift, aTarget + jitter, e * aText);
  vec2 dm = p - uMouse; float dd = length(dm);
  p += (dd > 0.5 ? dm / dd : vec2(0.)) * uMouseOn * 34. * exp(-dd * dd / 9800.) * aText * e;
  float gx = clamp((aTarget.x - uGrad.x) / max(uGrad.y - uGrad.x, 1.), 0., 1.);
  vec3 word = aLine < 0.5 ? uC0 : mix(uC1, uC2, gx);
  vColor = mix(uC3, word, e * aText);
  vAlpha = aText > 0.5 ? mix(0.3, 1., e) : 0.28;
  // 流光: once the words are in, a band of light keeps sweeping across them
  float sweep = fract(uTime * 0.22) * 1.6 - 0.3;
  float boost = exp(-pow((p.x / uRes.x - sweep) / 0.045, 2.)) * e * aText;
  vColor = mix(vColor, vec3(1.), boost * 0.85);
  vAlpha = min(1., vAlpha + boost * 0.3);
  gl_PointSize = ((aText > 0.5 ? mix(1.6, uDot, e) : 1.4 + 1.6 * aSeed.w) + boost * 1.6) * uDpr;
  gl_Position = vec4(p.x / uRes.x * 2. - 1., 1. - p.y / uRes.y * 2., 0., 1.);
}`

const FS = /* glsl */ `
varying vec3 vColor;
varying float vAlpha;
void main() {
  float d = length(gl_PointCoord - 0.5);
  float a = (smoothstep(0.5, 0.0, d) * 0.6 + smoothstep(0.25, 0.0, d) * 0.8) * vAlpha;
  gl_FragColor = vec4(vColor * a, a);
}`

type Target = [x: number, y: number, line: number]

/** Where to put particles: pixels of the slogan drawn on a canvas, line 1 in red, line 2 in green. */
function sampleText(lines: [string, string], w: number, h: number, step: number) {
  const c = document.createElement('canvas')
  c.width = w
  c.height = h
  const g = c.getContext('2d', { willReadFrequently: true })!
  const at100 = (s: string) => { g.font = `800 100px ${FONT}`; return g.measureText(s).width }
  const one = at100(lines.join(''))
  // on phones (or when one line won't fit) two big lines read better than one small one
  const twoLines = w < 700 || (Math.min(118, w * 0.072) * one) / 100 > w * 0.88
  const size = twoLines ? Math.min(w * 0.15, (100 * w * 0.86) / Math.max(at100(lines[0]), at100(lines[1]))) : Math.min(118, w * 0.072)
  g.font = `800 ${size}px ${FONT}`
  g.textBaseline = 'middle'
  const cy = h * 0.38
  const w0 = g.measureText(lines[0]).width, w1 = g.measureText(lines[1]).width
  const parts = twoLines
    ? [{ x: (w - w0) / 2, y: cy - size * 0.58 }, { x: (w - w1) / 2, y: cy + size * 0.58 }]
    : [{ x: (w - w0 - w1) / 2, y: cy }, { x: (w - w0 - w1) / 2 + w0, y: cy }]
  parts.forEach((p, i) => { g.fillStyle = i === 0 ? '#f00' : '#0f0'; g.fillText(lines[i], p.x, p.y) })
  const data = g.getImageData(0, 0, w, h).data
  const pts: Target[] = []
  let gx0 = Infinity, gx1 = -Infinity
  for (let y = 0; y < h; y += step) {
    for (let x = 0; x < w; x += step) {
      const o = (Math.floor(y) * w + Math.floor(x)) * 4
      if (data[o + 3] < 140) continue
      const line = data[o + 1] > data[o] ? 1 : 0
      pts.push([x, y, line])
      if (line === 1) { gx0 = Math.min(gx0, x); gx1 = Math.max(gx1, x) }
    }
  }
  return { pts, grad: [gx0, gx1] as const }
}

export interface AssembleContext {
  lines: [string, string]
  /** the element whose pointer moves push the particles */
  pointerTarget: HTMLElement
}

export function assembleScene(o: MotionOptions, ctx: AssembleContext): SceneHandle {
  const mobile = o.tier === 'mobile'
  const scene = new Scene()
  const camera = new Camera()
  const r = rand(23)
  const total = Math.round((mobile ? 3000 : 9500) * o.density)
  const geo = new BufferGeometry()
  const buf = {
    position: new Float32Array(total * 3), aStart: new Float32Array(total * 2), aTarget: new Float32Array(total * 2),
    aSeed: new Float32Array(total * 4), aText: new Float32Array(total), aLine: new Float32Array(total),
  }
  for (let i = 0; i < total * 4; i++) buf.aSeed[i] = r()
  const sizes: Record<keyof typeof buf, number> = { position: 3, aStart: 2, aTarget: 2, aSeed: 4, aText: 1, aLine: 1 }
  for (const k of Object.keys(buf) as (keyof typeof buf)[]) geo.setAttribute(k, new BufferAttribute(buf[k], sizes[k]))
  const uniforms = {
    uTime: { value: 0 }, uProgress: { value: 0 }, uDpr: { value: 1 }, uMouseOn: { value: 0 }, uDot: { value: mobile ? 3 : 3.4 },
    uRes: { value: new Vector2(1, 1) }, uMouse: { value: new Vector2(-999, -999) }, uGrad: { value: new Vector2(0, 1) },
    uC0: { value: COLORS.pale }, uC1: { value: new Color('#5ee7ff') }, uC2: { value: new Color('#7aa8ff') }, uC3: { value: COLORS.blue },
  }
  const pts = new Points(geo, new ShaderMaterial({ vertexShader: VS, fragmentShader: FS, uniforms, transparent: true, depthTest: false, depthWrite: false, blending: AdditiveBlending }))
  pts.frustumCulled = false
  scene.add(pts)

  let enteredAt = -1 // wall clock, so a slow device still lands the words on time
  let mouseTarget = 0
  const onMove = (e: PointerEvent) => {
    const b = ctx.pointerTarget.getBoundingClientRect()
    uniforms.uMouse.value.set(e.clientX - b.left, e.clientY - b.top)
    mouseTarget = 1
  }
  const onLeave = () => { mouseTarget = 0 }
  ctx.pointerTarget.addEventListener('pointermove', onMove)
  ctx.pointerTarget.addEventListener('pointerleave', onLeave)

  let layoutKey = ''
  function layout(w: number, h: number) {
    const key = `${w}x${h}`
    if (key === layoutKey) return
    layoutKey = key
    const { pts: targets, grad } = sampleText(ctx.lines, w, h, mobile ? 2.3 : 2.5)
    for (let i = targets.length - 1; i > 0; i--) { // shuffle so a capped count still covers every stroke
      const j = Math.floor(r() * (i + 1))
      ;[targets[i], targets[j]] = [targets[j], targets[i]]
    }
    const nText = Math.min(targets.length, Math.round(total * 0.85))
    for (let i = 0; i < total; i++) {
      buf.aStart[i * 2] = r() * (w + 200) - 100
      buf.aStart[i * 2 + 1] = r() * h
      if (i < nText) {
        buf.aTarget[i * 2] = targets[i][0]
        buf.aTarget[i * 2 + 1] = targets[i][1]
        buf.aText[i] = 1
        buf.aLine[i] = targets[i][2]
      } else buf.aText[i] = 0
    }
    uniforms.uGrad.value.set(grad[0], grad[1])
    for (const k of ['aStart', 'aTarget', 'aText', 'aLine']) geo.attributes[k].needsUpdate = true
    api.invalidate?.()
  }

  const api: SceneHandle = {
    scene, camera,
    resize(w, h, dpr) {
      uniforms.uDpr.value = dpr
      uniforms.uRes.value.set(w, h)
      // measure only once the page font is in, or the words land in the wrong place
      document.fonts.ready.then(() => layout(w, h))
    },
    frame(t, dt) {
      uniforms.uTime.value = t
      if (enteredAt >= 0) uniforms.uProgress.value = Math.min(1, (performance.now() - enteredAt) / 2400)
      uniforms.uMouseOn.value += (mouseTarget - uniforms.uMouseOn.value) * Math.min(1, dt * 6)
    },
    onEnter() { if (enteredAt < 0) enteredAt = performance.now() },
    onLeave() { enteredAt = -1; uniforms.uProgress.value = 0 }, // fly in again next time
    settle() { uniforms.uProgress.value = 1; uniforms.uTime.value = 3 },
    dispose() {
      ctx.pointerTarget.removeEventListener('pointermove', onMove)
      ctx.pointerTarget.removeEventListener('pointerleave', onLeave)
    },
  }
  return api
}
