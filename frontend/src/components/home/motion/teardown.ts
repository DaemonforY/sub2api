// 拆开看看 (after the earlier phone exploded-view film): the path from a sentence to a picture as five
// glass layers — 你说的一句话 → token → GPT-6 Astra → gpt-image-2 → 成品 — that stack, fan apart, light up
// one by one and close again, with tokens rising through them and labelled leader lines.
import {
  ACESFilmicToneMapping, AdditiveBlending, BufferAttribute, BufferGeometry, CanvasTexture, CircleGeometry, DirectionalLight, DoubleSide,
  ExtrudeGeometry, FogExp2, Group, HemisphereLight, LineBasicMaterial, LineLoop, Mesh, MeshBasicMaterial, MeshPhysicalMaterial,
  PerspectiveCamera, PlaneGeometry, PointLight, Points, RingGeometry, SRGBColorSpace, Scene, ShaderMaterial, Shape, Vector3, type WebGLRenderer,
} from 'three'
import { COLORS, rand, type MotionOptions, type SceneHandle } from './stage'
import { FONT, glyphAtlas } from './glyphs'
import { TEARDOWN_PROMPT } from './content'

const W = 6.4, H = 4.0, DEPTH = 0.1
const DURATION = 11
const PROMPT = TEARDOWN_PROMPT
const TOKENS = ['一只', '橘色', '短毛', '猫', '趴在', '老式', '木窗', '的', '窗台', '上', '，', '侧脸', '望向', '窗外', '下雨', '的', '街道', '。', '傍晚', '暖黄', '台灯', '冷蓝', '雨滴', '水彩', '插画', '风格']
const LAYER_KEYS = ['prompt', 'tokens', 'model', 'image', 'result'] as const
type LayerKey = (typeof LAYER_KEYS)[number]
const SIDES = [-1, 1, -1, 1, -1]
// [until (s), phase index, layer to light up or -1]; names come from the page (translated)
const PHASES: [number, number, number][] = [[1.2, 0, -1], [3.6, 1, -1], [5.4, 2, 2], [7.6, 3, 3], [8.6, 4, 4], [11, 5, -1]]

const ease = (t: number) => (t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2)
function stateAt(t: number) {
  const amount = t < 1.2 ? 0 : t < 3.6 ? ease((t - 1.2) / 2.4) : t < 8.6 ? 1 : 1 - ease((t - 8.6) / 2.4)
  const p = PHASES.find((x) => t < x[0]) || PHASES[PHASES.length - 1]
  return { amount, phase: p[1], focus: p[2] }
}

// ---------- the pictures printed on each layer ----------
function rr(g: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  g.beginPath(); g.moveTo(x + r, y); g.arcTo(x + w, y, x + w, y + h, r); g.arcTo(x + w, y + h, x, y + h, r)
  g.arcTo(x, y + h, x, y, r); g.arcTo(x, y, x + w, y, r); g.closePath()
}
function frame(g: CanvasRenderingContext2D, w: number, h: number, index: string, title: string) {
  g.clearRect(0, 0, w, h)
  g.fillStyle = 'rgba(8,24,44,.55)'; rr(g, 0, 0, w, h, 34); g.fill()
  g.strokeStyle = 'rgba(94,231,255,.08)'; g.lineWidth = 1
  for (let x = 32; x < w; x += 32) { g.beginPath(); g.moveTo(x, 0); g.lineTo(x, h); g.stroke() }
  for (let y = 32; y < h; y += 32) { g.beginPath(); g.moveTo(0, y); g.lineTo(w, y); g.stroke() }
  g.fillStyle = '#5ee7ff'; g.font = `600 22px ui-monospace, Menlo, monospace`
  g.textAlign = 'left'; g.textBaseline = 'top'
  g.fillText(`${index} / 05 · ${title}`, 40, 34)
}
function wrap(g: CanvasRenderingContext2D, text: string, x: number, y: number, maxW: number, lh: number) {
  let line = '', yy = y
  for (const ch of text) {
    if (g.measureText(line + ch).width > maxW) { g.fillText(line, x, yy); line = ch; yy += lh } else line += ch
  }
  g.fillText(line, x, yy)
  return yy
}
function drawLayer(key: LayerKey, g: CanvasRenderingContext2D, w: number, h: number, img: HTMLImageElement) {
  const i = LAYER_KEYS.indexOf(key)
  frame(g, w, h, `0${i + 1}`, ['PROMPT', 'TOKENIZE', 'GPT-6 ASTRA', 'GPT-IMAGE-2', 'OUTPUT'][i])
  g.textAlign = 'left'; g.textBaseline = 'top'
  if (key === 'prompt') {
    g.fillStyle = 'rgba(94,231,255,.18)'; rr(g, 40, 92, 92, 44, 10); g.fill()
    g.fillStyle = '#9fdcff'; g.font = `600 26px ${FONT}`; g.fillText('你说', 60, 100)
    g.fillStyle = '#eaf7ff'; g.font = `500 42px ${FONT}`
    const end = wrap(g, PROMPT.slice(0, 54) + '…', 40, 166, w - 90, 64)
    g.fillStyle = '#5ee7ff'; g.fillRect(40, end + 72, 22, 4)
  } else if (key === 'tokens') {
    let x = 40, y = 96
    g.font = `600 32px ${FONT}`
    TOKENS.forEach((t, j) => {
      const tw = g.measureText(t).width + 30
      if (x + tw > w - 36) { x = 40; y += 92 }
      if (y > h - 90) return
      g.strokeStyle = j % 3 === 0 ? '#ffbe6b' : j % 2 ? '#5ee7ff' : '#4d8dff'; g.lineWidth = 2.5
      rr(g, x, y, tw, 52, 10); g.stroke()
      g.fillStyle = 'rgba(94,231,255,.10)'; g.fill()
      g.fillStyle = '#eaf7ff'; g.fillText(t, x + 15, y + 9)
      g.fillStyle = 'rgba(159,220,255,.6)'; g.font = `500 16px ui-monospace, Menlo, monospace`
      g.fillText(String(1000 + ((j * 7919) % 90000)), x + 4, y + 58)
      g.font = `600 32px ${FONT}`
      x += tw + 16
    })
  } else if (key === 'model') {
    const cols = [150, 330, 510], rows = [[150, 260, 370, 480], [200, 320, 440], [150, 260, 370, 480]]
    g.strokeStyle = 'rgba(94,231,255,.35)'; g.lineWidth = 1.5
    for (let c = 0; c < 2; c++) for (const a of rows[c]) for (const b of rows[c + 1]) { g.beginPath(); g.moveTo(cols[c], a); g.lineTo(cols[c + 1], b); g.stroke() }
    rows.forEach((r, c) => r.forEach((y) => { g.fillStyle = c === 1 ? '#ffbe6b' : '#5ee7ff'; g.beginPath(); g.arc(cols[c], y, 11, 0, 7); g.fill() }))
    g.fillStyle = '#eaf7ff'; g.font = `700 52px ${FONT}`; g.fillText('读懂你要什么', 600, 140)
    g.font = `500 32px ${FONT}`; g.fillStyle = '#9fdcff'
    ;['主体：一只橘猫', '场景：雨夜 · 窗台', '光线：暖灯 + 冷蓝', '风格：水彩插画'].forEach((s, j) => g.fillText(s, 604, 232 + j * 64))
  } else if (key === 'image' || key === 'result') {
    const x = 40, y = 92, iw = w - 80, ih = h - 172
    g.save(); rr(g, x, y, iw, ih, 18); g.clip()
    if (img.complete && img.naturalWidth) {
      const s = Math.max(iw / img.naturalWidth, ih / img.naturalHeight)
      const dw = img.naturalWidth * s, dh = img.naturalHeight * s
      if (key === 'image') { // half-made: coarse pixels
        const px = 22, small = document.createElement('canvas')
        small.width = Math.ceil(iw / px); small.height = Math.ceil(ih / px)
        small.getContext('2d')!.drawImage(img, (iw - dw) / 2 / px, ((ih - dh) / 2 - 40) / px, dw / px, dh / px)
        g.imageSmoothingEnabled = false
        g.drawImage(small, x, y, iw, ih)
        g.imageSmoothingEnabled = true
      } else g.drawImage(img, x + (iw - dw) / 2, y + (ih - dh) / 2 - 40, dw, dh)
    } else { g.fillStyle = '#0b2440'; g.fillRect(x, y, iw, ih) }
    g.restore()
    g.fillStyle = key === 'image' ? '#5ee7ff' : '#ffbe6b'; g.font = `600 30px ${FONT}`
    g.fillText(key === 'image' ? '生成中 · 2K' : '成品 · 2K · 约 $0.2 一张', 40, h - 64)
  }
}

const RISE_VS = /* glsl */ `
attribute vec4 aSeed;
attribute float aGlyph;
uniform float uTime, uDpr, uTop, uVisible, uGrid, uScale;
uniform vec3 uC0, uC1;
varying float vAlpha;
varying vec2 vCell;
varying vec3 vColor;
void main() {
  float t = fract(aSeed.x + uTime * (0.07 + 0.06 * aSeed.y));
  vec3 p = vec3((aSeed.z - 0.5) * 3.6, t * uTop, (aSeed.w - 0.5) * 2.2);
  vec4 mv = modelViewMatrix * vec4(p, 1.);
  gl_Position = projectionMatrix * mv;
  gl_PointSize = (16. + 12. * aSeed.y) * uDpr * uScale / -mv.z;
  vAlpha = smoothstep(0., 0.06, t) * (1. - smoothstep(0.86, 1., t)) * uVisible;
  vCell = vec2(mod(aGlyph, uGrid), floor(aGlyph / uGrid));
  vColor = mix(uC0, uC1, t);
}`
const RISE_FS = /* glsl */ `
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

function plateShape() {
  const s = new Shape(), r = 0.36, x = -W / 2, y = -H / 2
  s.moveTo(x + r, y)
  s.lineTo(x + W - r, y)
  s.quadraticCurveTo(x + W, y, x + W, y + r)
  s.lineTo(x + W, y + H - r)
  s.quadraticCurveTo(x + W, y + H, x + W - r, y + H)
  s.lineTo(x + r, y + H)
  s.quadraticCurveTo(x, y + H, x, y + H - r)
  s.lineTo(x, y + r)
  s.quadraticCurveTo(x, y, x + r, y)
  return s
}

export interface TeardownContext {
  /** HUD the scene writes into: label boxes, leader-line svg, timecode, progress fill, phase name */
  labels: HTMLElement
  leaders: SVGSVGElement
  time: HTMLElement
  fill: HTMLElement
  phase: HTMLElement
  /** translated names: one { name, spec } per layer (bottom → top), six phase names */
  layerText: { name: string; spec: string }[]
  phaseNames: string[]
  imageUrl: string
}

export function teardownScene(renderer: WebGLRenderer, o: MotionOptions, ctx: TeardownContext): SceneHandle {
  renderer.toneMapping = ACESFilmicToneMapping
  renderer.toneMappingExposure = 1.15
  const scene = new Scene()
  scene.fog = new FogExp2(0x050a14, 0.022)
  const camera = new PerspectiveCamera(30, 1, 0.1, 200)
  scene.add(new HemisphereLight(0x9cc8ff, 0x0a1220, 1.1))
  const key = new DirectionalLight(0xe6f4ff, 3.6)
  key.position.set(-6, 10, 8)
  const rim = new DirectionalLight(0x5ee7ff, 3.2)
  rim.position.set(8, 4, -8)
  const warm = new PointLight(0xffa35f, 6, 16, 2)
  warm.position.set(-5, -1, 4)
  scene.add(key, rim, warm)

  const stack = new Group()
  stack.rotation.x = 0.38 // tops lean toward the viewer so the printed layers read
  scene.add(stack)
  const shape = plateShape()
  const plateGeo = new ExtrudeGeometry(shape, { depth: DEPTH, bevelEnabled: true, bevelThickness: 0.035, bevelSize: 0.035, bevelSegments: 3, curveSegments: 10 })
  plateGeo.rotateX(-Math.PI / 2)
  const edgePts = shape.getPoints(40).map((p) => new Vector3(p.x, DEPTH + 0.05, -p.y))
  const img = new Image()
  img.src = ctx.imageUrl
  const redraws: (() => void)[] = []
  img.addEventListener('load', () => { redraws.forEach((f) => f()); api.invalidate?.() })

  const layers = LAYER_KEYS.map((layerKey, i) => {
    const g = new Group()
    const body = new Mesh(plateGeo, new MeshPhysicalMaterial({
      color: 0x0b1d33, metalness: 0.35, roughness: 0.22, clearcoat: 1, clearcoatRoughness: 0.12,
      transparent: true, opacity: 0.9, emissive: 0x0a3a66, emissiveIntensity: 0.35,
    }))
    g.add(body)
    const c = document.createElement('canvas')
    c.width = 1024
    c.height = 640
    const tex = new CanvasTexture(c)
    tex.colorSpace = SRGBColorSpace
    tex.anisotropy = renderer.capabilities.getMaxAnisotropy()
    const draw = () => { drawLayer(layerKey, c.getContext('2d')!, c.width, c.height, img); tex.needsUpdate = true }
    draw()
    if (layerKey === 'image' || layerKey === 'result') redraws.push(draw)
    const face = new Mesh(new PlaneGeometry(W - 0.32, H - 0.32), new MeshBasicMaterial({ map: tex, transparent: true, toneMapped: false, depthWrite: false }))
    face.rotation.x = -Math.PI / 2
    face.position.y = DEPTH + 0.045
    g.add(face)
    const edge = new LineLoop(new BufferGeometry().setFromPoints(edgePts), new LineBasicMaterial({ color: COLORS.cyan.clone(), transparent: true, opacity: 0.85, toneMapped: false }))
    g.add(edge)
    stack.add(g)
    return { g, body, edge, i }
  })

  // the floor: a soft disc of light and two rings
  const halo = new Mesh(new CircleGeometry(6.5, 64), new MeshBasicMaterial({ color: 0x0f3a66, transparent: true, opacity: 0.35, toneMapped: false }))
  halo.rotation.x = -Math.PI / 2
  halo.position.y = -0.6
  stack.add(halo)
  for (const rad of [5.2, 6.4]) {
    const ring = new Mesh(new RingGeometry(rad, rad + 0.02, 96), new MeshBasicMaterial({ color: COLORS.cyan, transparent: true, opacity: 0.25, toneMapped: false, side: DoubleSide }))
    ring.rotation.x = -Math.PI / 2
    ring.position.y = -0.59
    stack.add(ring)
  }

  const r = rand(31)
  const atlas = glyphAtlas()
  const n = Math.round((o.tier === 'mobile' ? 90 : 220) * o.density)
  const rg = new BufferGeometry()
  const seeds = new Float32Array(n * 4), glyph = new Float32Array(n)
  for (let i = 0; i < n; i++) { seeds.set([r(), r(), r(), r()], i * 4); glyph[i] = Math.floor(r() * atlas.count) }
  rg.setAttribute('position', new BufferAttribute(new Float32Array(n * 3), 3))
  rg.setAttribute('aSeed', new BufferAttribute(seeds, 4))
  rg.setAttribute('aGlyph', new BufferAttribute(glyph, 1))
  const riseU = { uTime: { value: 0 }, uDpr: { value: 1 }, uTop: { value: 6 }, uVisible: { value: 0 }, uGrid: { value: atlas.grid }, uScale: { value: 14 }, uAtlas: { value: atlas.tex }, uC0: { value: COLORS.cyan }, uC1: { value: COLORS.pale } }
  const rise = new Points(rg, new ShaderMaterial({ vertexShader: RISE_VS, fragmentShader: RISE_FS, uniforms: riseU, transparent: true, depthWrite: false, blending: AdditiveBlending }))
  rise.frustumCulled = false
  stack.add(rise)

  // HUD: one label per layer, with a leader line from the layer's edge
  ctx.labels.replaceChildren(...ctx.layerText.map((l, i) => {
    const el = document.createElement('div')
    el.className = 'td-label'
    const idx = document.createElement('div'), nm = document.createElement('div'), sp = document.createElement('div')
    idx.className = 'idx'; idx.textContent = `0${i + 1} / 05`
    nm.className = 'nm'; nm.textContent = l.name
    sp.className = 'sp'; sp.textContent = l.spec
    el.append(idx, nm, sp)
    return el
  }))
  const labelEls = [...ctx.labels.children] as HTMLElement[]
  let w = 1, h = 1, portrait = false, enteredAt = -1, still = false
  const v = new Vector3()

  function pose(t: number, now: number) {
    const s = still ? { amount: 1, phase: 1, focus: 2 } : stateAt(t)
    const gap = 0.18 + s.amount * (portrait ? 1.0 : 1.12)
    layers.forEach(({ g, body, edge, i }) => {
      g.position.y = i * gap
      // fan out like a dealt deck: higher layers slide back, so every layer's face shows
      g.position.z = -(i - 2) * (portrait ? 0.8 : 1.05) * s.amount
      g.position.x = (i - 2) * (portrait ? 0 : 0.35) * s.amount + Math.sin(now * 0.00045 + i) * 0.04 * s.amount
      const lit = s.focus === i
      const mat = body.material
      mat.emissive.setHex(lit ? 0x8a4a12 : 0x0a3a66)
      mat.emissiveIntensity = lit ? 0.75 + Math.sin(now * 0.008) * 0.25 : 0.35
      edge.material.color.copy(lit ? COLORS.amber : COLORS.cyan)
      edge.material.opacity = lit ? 1 : 0.55 + 0.3 * s.amount
    })
    riseU.uTop.value = 4 * gap + 0.4
    rise.position.z = -0.6 * s.amount // the token column leans back with the fan
    riseU.uVisible.value = s.amount
    riseU.uTime.value = (now / 1000) % 1000
    // camera: a slow half-orbit, never round the back, so the layers stay readable
    const a = -0.55 + Math.sin(now * 0.00012) * 0.3
    const R = portrait ? 27 : 24
    const cy = 2.0 * gap + 1
    camera.position.set(Math.sin(a) * R, cy + (portrait ? 9 : 7.6), Math.cos(a) * R)
    camera.lookAt(portrait ? 0 : -4.6, cy - (portrait ? 2.2 : 0.9), 0)
    stack.updateMatrixWorld(true)
    // labels + leader lines
    const vis = Math.max(0, Math.min(1, s.amount * 1.8 - 0.2))
    let paths = ''
    layers.forEach(({ g, i }) => {
      const side = SIDES[i]
      v.set(side * W * 0.47, DEPTH, 0).applyMatrix4(g.matrixWorld).project(camera)
      const x = (v.x * 0.5 + 0.5) * w, y = (-v.y * 0.5 + 0.5) * h
      const lx = Math.max(90, Math.min(w - 90, x + side * (portrait ? 70 : 150)))
      const ly = y - 6
      const el = labelEls[i]
      el.style.transform = `translate(${lx}px, ${ly}px) translate(-50%, -50%)`
      el.style.opacity = String(vis)
      el.classList.toggle('focus', s.focus === i)
      if (vis > 0.02) {
        const col = s.focus === i ? '#ffbe6b' : '#5ee7ff'
        const bx = x + side * 30
        paths += `<path d="M${x.toFixed(1)} ${y.toFixed(1)} L${bx.toFixed(1)} ${y.toFixed(1)} L${(lx - side * 62).toFixed(1)} ${ly.toFixed(1)}" stroke="${col}" stroke-width="${s.focus === i ? 1.8 : 1.2}" fill="none" opacity="${(0.75 * vis).toFixed(2)}"/><circle cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="${s.focus === i ? 3.4 : 2.4}" fill="${col}" opacity="${vis.toFixed(2)}"/>`
      }
    })
    ctx.leaders.innerHTML = paths // numbers and two fixed colours only
    if (!still) {
      ctx.time.textContent = `00:${String(Math.floor(t)).padStart(2, '0')} / 00:${DURATION}`
      ctx.fill.style.width = `${(t / DURATION) * 100}%`
      ctx.phase.textContent = ctx.phaseNames[s.phase] || ''
    }
  }

  const api: SceneHandle = {
    scene, camera,
    resize(ww, hh, dpr) {
      w = ww
      h = hh
      portrait = w / h < 0.9
      camera.aspect = w / h
      camera.fov = portrait ? 40 : 30
      camera.updateProjectionMatrix()
      riseU.uDpr.value = dpr
      riseU.uScale.value = h / 60
      ctx.leaders.setAttribute('viewBox', `0 0 ${w} ${h}`)
      if (still) pose(0, 4000)
    },
    frame() {
      const now = performance.now()
      if (enteredAt < 0) enteredAt = now
      pose(((now - enteredAt) / 1000) % DURATION, now)
    },
    onEnter() { if (enteredAt < 0) enteredAt = performance.now() },
    onLeave() { enteredAt = -1 }, // start the film again next time
    settle() {
      still = true
      ctx.phase.textContent = ctx.phaseNames[1] || ''
      ctx.fill.style.width = '40%'
    },
    dispose() { plateGeo.dispose() },
  }
  return api
}
