// Shared stage for the home page's motion screens: one WebGL canvas per screen, drawn only while the
// screen is in view and the tab is visible, pixel ratio capped, a single still frame for reduced
// motion. Loaded on demand (three.js is in this chunk only).
import { Camera, Color, Scene, WebGLRenderer, type Mesh, type MeshBasicMaterial, type Object3D, type ShaderMaterial, type Texture } from 'three'

export type MotionTier = 'desktop' | 'mobile'

export interface MotionOptions {
  tier: MotionTier
  /** draw one still frame instead of animating */
  still: boolean
  density: number
}

export interface SceneHandle {
  scene: Scene
  camera: Camera
  resize(w: number, h: number, dpr: number): void
  frame(t: number, dt: number): void
  onEnter?(): void
  onLeave?(): void
  /** put the scene in its finished state for a still frame */
  settle?(): void
  dispose?(): void
  invalidate?: () => void
}

// v2 · 科技蓝: cyan and electric blue on a blue-black ground; amber only for value.
export const COLORS = {
  ink: new Color('#050a14'),
  blue: new Color('#3b82f6'),
  ice: new Color('#9fdcff'),
  pale: new Color('#eaf7ff'),
  cyan: new Color('#5ee7ff'),
  amber: new Color('#ffbe6b'),
}

export function rand(seed: number): () => number {
  let s = seed >>> 0
  return () => ((s = (s * 1664525 + 1013904223) >>> 0) / 4294967296)
}

/** A rect of `el` relative to `host`, normalised to 0..1 with y pointing up (GL style). */
export function relRect(el: Element, host: Element) {
  const h = host.getBoundingClientRect(), r = el.getBoundingClientRect()
  const x0 = (r.left - h.left) / h.width, x1 = (r.right - h.left) / h.width
  const y1 = 1 - (r.top - h.top) / h.height, y0 = 1 - (r.bottom - h.top) / h.height
  return { x0, x1, y0, y1, cx: (x0 + x1) / 2, cy: (y0 + y1) / 2, w: r.width, h: r.height }
}

function disposeTree(root: Object3D) {
  root.traverse((o) => {
    const mesh = o as Mesh
    mesh.geometry?.dispose()
    const mats = mesh.material ? (Array.isArray(mesh.material) ? mesh.material : [mesh.material]) : []
    for (const mat of mats) {
      for (const u of Object.values((mat as ShaderMaterial).uniforms ?? {})) {
        if ((u.value as Texture | null)?.isTexture) (u.value as Texture).dispose()
      }
      const map = (mat as MeshBasicMaterial).map
      map?.dispose()
      mat.dispose()
    }
  })
}

/** Mounts a scene into `host`; returns a function that tears it down. */
export function createStage(host: HTMLElement, opts: MotionOptions, setup: (r: WebGLRenderer, o: MotionOptions) => SceneHandle): () => void {
  const renderer = new WebGLRenderer({ antialias: false, alpha: true, powerPreference: 'high-performance' })
  renderer.setClearColor(0x000000, 0)
  const canvas = renderer.domElement
  canvas.setAttribute('aria-hidden', 'true')
  canvas.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;display:block'
  host.prepend(canvas)
  const s = setup(renderer, opts)
  const mobile = opts.tier === 'mobile'
  const dprCap = mobile ? 1 : 1.5
  const minFrameMs = mobile ? 1000 / 30 - 2 : 0
  let raf = 0, t = 0, last = 0, visible = false, gone = false

  const render = (dt: number) => {
    if (!opts.still) s.frame(t, dt)
    renderer.render(s.scene, s.camera)
  }
  const resize = () => {
    const w = host.clientWidth, h = host.clientHeight
    if (!w || !h || gone) return
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, dprCap))
    renderer.setSize(w, h, false)
    s.resize(w, h, renderer.getPixelRatio())
    if (!raf) render(0)
  }
  const loop = (now: number) => {
    raf = requestAnimationFrame(loop)
    if (now - last < minFrameMs) return
    const dt = Math.min(0.05, (now - last) / 1000)
    last = now
    t += dt
    render(dt)
  }
  const start = () => {
    if (raf || opts.still || gone) return
    last = performance.now()
    raf = requestAnimationFrame(loop)
  }
  const stop = () => {
    cancelAnimationFrame(raf)
    raf = 0
  }
  const ro = new ResizeObserver(resize)
  ro.observe(host)
  const io = new IntersectionObserver(([e]) => {
    visible = e.isIntersecting
    if (visible) {
      s.onEnter?.()
      if (!document.hidden) start()
    } else {
      stop()
      s.onLeave?.()
    }
  }, { threshold: 0.02 })
  io.observe(host)
  const onVisibility = () => (document.hidden ? stop() : visible && start())
  document.addEventListener('visibilitychange', onVisibility)
  s.invalidate = () => { if (!raf) render(0) }
  if (opts.still) s.settle?.()
  resize()

  return () => {
    gone = true
    stop()
    ro.disconnect()
    io.disconnect()
    document.removeEventListener('visibilitychange', onVisibility)
    s.dispose?.()
    disposeTree(s.scene)
    renderer.dispose()
    canvas.remove()
  }
}
