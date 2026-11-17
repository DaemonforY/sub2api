// Drawing for 椰风岛大冒险: everything is drawn at 400x224 and scaled up with nearest-neighbour.
import { W, H, T, ROWS, TS, TO, TB, TW, TL, EV, ENERGY_MAX, IT_CLOUD, IT_GOLD, type GameEvent, type Kind, type Stage, type Theme } from './defs'
import { Pix, rgb, mix, rng, shadowOf, lightOf, OUTLINE, type Color } from './pix'
import { buildArt, type Art, type Sheet } from './sprites'
import { stages } from './level'

export interface RPlayer {
  i: number
  on: boolean
  st: string
  x: number
  y: number
  vx: number
  vy: number
  face: number
  ground: boolean
  weapon: number
  board: boolean
  fairy: number
  cloud: number
  inv: number
  trip: number
  throwT: number
  runT: number
  energy: number
  score: number
  dyingT: number
}

export interface REnt {
  id: number
  k: Kind
  x: number
  y: number
  w: number
  h: number
  st: number
  dir: number
  a: number
  t: number
  flash: number
  hp: number
  c: number
  vx: number
  vy: number
  ground: boolean
}

export interface RView {
  stageIndex: number
  camX: number
  frame: number
  lock: boolean
  phase: string
  phaseT: number
  lives: number
  shake: number
  players: RPlayer[]
  ents: REnt[]
  bossMax: number
}

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  g: number
  life: number
  max: number
  c: string
  s: number
  kind: 0 | 1 | 2 | 3 // 0 square, 1 sparkle, 2 text digits, 3 ring
  text?: string
}

const THEME_SKY: Record<Theme, [string, string]> = {
  beach: ['#2a7fff', '#9fe0ff'],
  jungle: ['#3d7a6e', '#a8d7c0'],
  volcano: ['#2a1030', '#a0303a']
}

// 3x5 pixel digits for the HUD
const DIGITS: Record<string, string[]> = {
  '0': ['111', '101', '101', '101', '111'],
  '1': ['010', '110', '010', '010', '111'],
  '2': ['111', '001', '111', '100', '111'],
  '3': ['111', '001', '011', '001', '111'],
  '4': ['101', '101', '111', '001', '001'],
  '5': ['111', '100', '111', '001', '111'],
  '6': ['111', '100', '111', '101', '111'],
  '7': ['111', '001', '010', '010', '010'],
  '8': ['111', '101', '111', '101', '111'],
  '9': ['111', '101', '111', '001', '111'],
  x: ['000', '101', '010', '101', '000'],
  '+': ['000', '010', '111', '010', '000'],
  P: ['110', '101', '110', '100', '100']
}

export class Renderer {
  readonly low: HTMLCanvasElement
  private readonly lctx: CanvasRenderingContext2D
  private readonly out: HTMLCanvasElement
  private readonly octx: CanvasRenderingContext2D
  readonly art: Art
  private far: Partial<Record<Theme, HTMLImageElement>> = {}
  private mid: Partial<Record<Theme, HTMLImageElement>> = {}
  private stageCanvas = new Map<number, HTMLCanvasElement>()
  private parts: Particle[] = []
  private glyphs = new Map<string, HTMLCanvasElement>()
  flashWhite = 0

  constructor(out: HTMLCanvasElement, base = '/games-media/island/') {
    this.out = out
    const octx = out.getContext('2d')
    if (!octx) throw new Error('no 2d context')
    this.octx = octx
    this.low = document.createElement('canvas')
    this.low.width = W
    this.low.height = H
    const l = this.low.getContext('2d')
    if (!l) throw new Error('no 2d context')
    this.lctx = l
    this.art = buildArt()
    for (const th of ['beach', 'jungle', 'volcano'] as Theme[]) {
      const f = new Image()
      f.src = `${base}far-${th}.webp`
      this.far[th] = f
      const m = new Image()
      m.src = `${base}mid-${th}.webp`
      this.mid[th] = m
    }
  }

  /** Resolve once the background images have arrived (or failed). */
  async ready(): Promise<void> {
    const imgs = [...Object.values(this.far), ...Object.values(this.mid)] as HTMLImageElement[]
    await Promise.all(imgs.map((i) => (i.complete ? Promise.resolve() : new Promise<void>((r) => ((i.onload = () => r()), (i.onerror = () => r()))))))
  }

  resize(cssW: number, cssH: number, dpr: number) {
    this.out.width = Math.max(1, Math.round(cssW * dpr))
    this.out.height = Math.max(1, Math.round(cssH * dpr))
  }

  /** Where the game picture sits inside the output canvas (for touch hit tests). */
  viewport(): { x: number; y: number; s: number } {
    const s = Math.min(this.out.width / W, this.out.height / H)
    return { x: Math.floor((this.out.width - W * s) / 2), y: Math.floor((this.out.height - H * s) / 2), s }
  }

  // ───────────────────────── particles from events ─────────────────────────

  onEvents(evs: GameEvent[]) {
    for (const [type, x, y, a] of evs) this.onEvent(type, x, y, a)
  }

  private burst(x: number, y: number, n: number, colors: string[], speed = 2, g = 0.12, life = 30, size = 2) {
    for (let i = 0; i < n; i++) {
      const ang = Math.random() * Math.PI * 2
      const sp = speed * (0.4 + Math.random() * 0.8)
      this.parts.push({ x, y, vx: Math.cos(ang) * sp, vy: Math.sin(ang) * sp - speed * 0.4, g, life: life * (0.6 + Math.random() * 0.6), max: life, c: colors[i % colors.length], s: size, kind: 0 })
    }
  }

  private text(x: number, y: number, t: string, c = '#ffffff') {
    this.parts.push({ x, y, vx: 0, vy: -0.7, g: 0.012, life: 50, max: 50, c, s: 1, kind: 2, text: t })
  }

  private onEvent(type: number, x: number, y: number, a: number) {
    switch (type) {
      case EV.jump:
        this.burst(x, y, 4, ['#ffffff', '#f3e1b8'], 0.8, 0.02, 14)
        break
      case EV.land:
        this.burst(x, y, 5, ['#ffffff', '#f3e1b8'], 0.9, 0.02, 14)
        break
      case EV.kill:
        this.burst(x, y, 10, ['#ffe14d', '#ffffff', '#ff8a2a'], 2.4, 0.1, 26)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 12, max: 12, c: '#ffffff', s: 10, kind: 3 })
        this.text(x, y - 8, String(a))
        break
      case EV.fruit: {
        const combo = a & 15
        this.burst(x, y, 6, ['#fff6b0', '#ffffff'], 1.4, 0, 20, 1)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 10, max: 10, c: '#fff6b0', s: 7, kind: 3 })
        if (combo >= 2) this.text(x, y - 10, `x${combo + 1}`, '#ffe14d')
        break
      }
      case EV.clam:
        this.burst(x, y, 14, ['#ffffff', '#ffe9a8', '#ff9ac8'], 2.2, 0.08, 30)
        break
      case EV.item:
        this.burst(x, y, 16, ['#ffe14d', '#ffffff', '#38b6ff'], 2.4, 0.05, 30)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 16, max: 16, c: '#ffffff', s: 18, kind: 3 })
        break
      case EV.trip:
        this.burst(x, y, 6, ['#d9c7a0', '#ffffff'], 1.2, 0.05, 20)
        this.text(x, y - 30, '-10', '#ff6b6b')
        break
      case EV.die:
        this.burst(x, y, 14, ['#ffffff', '#ffe14d'], 2.6, 0.04, 30)
        this.flashWhite = 4
        break
      case EV.revive:
        this.burst(x, y, 18, ['#9fe6ff', '#ffffff', '#ff9ac8'], 2.6, 0.02, 34)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 18, max: 18, c: '#9fe6ff', s: 22, kind: 3 })
        break
      case EV.flag:
        this.burst(x + 10, y - 36, 16, ['#38b6ff', '#ffffff', '#ffe14d'], 2.2, 0.06, 36)
        break
      case EV.spring:
        this.burst(x, y, 6, ['#ff4f6a', '#ffffff'], 1.6, 0.06, 16)
        break
      case EV.bossHit:
        this.burst(x, y, 10, ['#ffffff', '#ffe14d'], 2.6, 0.06, 18)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 10, max: 10, c: '#ffffff', s: 14, kind: 3 })
        break
      case EV.bossDie:
        this.burst(x, y, a < 0 ? 12 : 30, ['#ffffff', '#ffe14d', '#ff8a2a', '#ff4f4f'], a < 0 ? 2.6 : 3.6, 0.04, 40, 3)
        this.parts.push({ x, y, vx: 0, vy: 0, g: 0, life: 22, max: 22, c: '#ffe14d', s: a < 0 ? 18 : 40, kind: 3 })
        if (a >= 0) this.flashWhite = 10
        break
      case EV.breakRock:
        this.burst(x, y, 10, ['#8e8a9e', '#b9b5c8', '#5d5970'], 2.2, 0.16, 30, 2)
        this.text(x, y - 8, '50')
        break
      case EV.extinguish:
        this.burst(x, y, 12, ['#d0d0d8', '#ffffff', '#8a8a96'], 1.4, -0.04, 40, 3)
        break
      case EV.splash:
        this.burst(x, y, 12, a ? ['#ff8a2a', '#ffe14d'] : ['#9fe6ff', '#ffffff'], 2.4, 0.16, 28)
        break
      case EV.boardLost:
        this.burst(x, y, 12, ['#38b6ff', '#ffffff'], 2.6, 0.12, 26)
        break
      case EV.clink:
        this.burst(x, y, 5, ['#ffffff', '#ffe14d'], 1.8, 0.05, 12, 1)
        break
      case EV.quake:
        this.burst(x, y, 14, ['#d9c7a0', '#a8946e', '#ffffff'], 2.2, 0.14, 30, 3)
        break
      case EV.sun:
        this.burst(x, y, 40, ['#ffe14d', '#ffffff', '#ff9a1f'], 3.4, 0.02, 60, 2)
        this.flashWhite = 12
        break
      default:
    }
  }

  private stepParticles() {
    for (const p of this.parts) {
      p.vy += p.g
      p.x += p.vx
      p.y += p.vy
      p.vx *= 0.98
      p.life--
    }
    this.parts = this.parts.filter((p) => p.life > 0)
    if (this.parts.length > 500) this.parts.splice(0, this.parts.length - 500)
  }

  // ───────────────────────── stage tile layer ─────────────────────────

  private tiles(stage: Stage): HTMLCanvasElement {
    const hit = this.stageCanvas.get(stage.id)
    if (hit) return hit
    const cv = buildStageLayer(stage).canvas()
    this.stageCanvas.set(stage.id, cv)
    return cv
  }

  // ───────────────────────── main draw ─────────────────────────

  draw(v: RView, localIndex: number | null = null) {
    const c = this.lctx
    const stage = stages()[v.stageIndex]
    const theme = stage.theme
    this.stepParticles()
    const shake = v.shake > 0 ? Math.round((Math.random() - 0.5) * Math.min(6, v.shake * 0.5)) : 0
    const shakeY = v.shake > 0 ? Math.round((Math.random() - 0.5) * Math.min(5, v.shake * 0.4)) : 0
    const cam = Math.round(v.camX) - shake
    c.imageSmoothingEnabled = false
    // sky
    const [s0, s1] = THEME_SKY[theme]
    const grad = c.createLinearGradient(0, 0, 0, H)
    grad.addColorStop(0, s0)
    grad.addColorStop(1, s1)
    c.fillStyle = grad
    c.fillRect(0, 0, W, H)
    this.layer(this.far[theme], cam * 0.12, 0)
    this.ambient(theme, cam, v.frame)
    const mid = this.mid[theme]
    if (mid?.complete && mid.naturalWidth) this.layer(mid, cam * 0.42, H - 26 - mid.naturalHeight)
    this.liquid(theme, cam, v.frame)
    c.save()
    c.translate(0, shakeY)
    c.drawImage(this.tiles(stage), cam, 0, W, H, 0, 0, W, H)
    // entities, back to front
    const order: Kind[] = ['flag', 'plat', 'spring', 'geyser', 'rock', 'fire', 'clam', 'fruit', 'item', 'sun', 'crab', 'frog', 'spiky', 'boulder', 'flame', 'spider', 'fish', 'gull', 'bat', 'boss', 'shot', 'boom']
    const rank = new Map(order.map((k, i) => [k, i]))
    const ents = [...v.ents].sort((a, b) => (rank.get(a.k) ?? 0) - (rank.get(b.k) ?? 0))
    for (const e of ents) this.ent(e, cam, v)
    for (const p of v.players) if (p.on && p.st !== 'out') this.player(p, cam, v.frame, localIndex)
    this.drawParticles(cam)
    c.restore()
    this.hud(v, stage)
    if (this.flashWhite > 0) {
      c.fillStyle = `rgba(255,255,255,${this.flashWhite / 14})`
      c.fillRect(0, 0, W, H)
      this.flashWhite--
    }
    this.present()
  }

  private present() {
    const o = this.octx
    o.imageSmoothingEnabled = false
    o.fillStyle = '#05030a'
    o.fillRect(0, 0, this.out.width, this.out.height)
    const vp = this.viewport()
    o.drawImage(this.low, vp.x, vp.y, Math.round(W * vp.s), Math.round(H * vp.s))
  }

  /** Horizontally repeating parallax layer, mirrored every other copy so the seams match. */
  private layer(img: HTMLImageElement | undefined, offset: number, y: number) {
    if (!img || !img.complete || !img.naturalWidth) return
    const c = this.lctx
    const iw = img.naturalWidth
    const period = iw * 2
    let x = -(((offset % period) + period) % period)
    for (let k = 0; x < W; k++) {
      const flip = k % 2 === 1
      if (flip) {
        c.save()
        c.translate(Math.round(x) + iw, Math.round(y))
        c.scale(-1, 1)
        c.drawImage(img, 0, 0)
        c.restore()
      } else c.drawImage(img, Math.round(x), Math.round(y))
      x += iw
    }
  }

  private ambient(theme: Theme, cam: number, frame: number) {
    const c = this.lctx
    if (theme === 'volcano') {
      for (let i = 0; i < 24; i++) {
        const seed = i * 97.13
        const x = ((seed * 7 - cam * 0.3 + frame * 0.2 * ((i % 3) + 1)) % (W + 20) + W + 20) % (W + 20) - 10
        const y = H - ((frame * (0.4 + (i % 5) * 0.15) + seed * 3) % (H + 20))
        c.fillStyle = i % 3 ? '#ff8a2a' : '#ffe14d'
        c.fillRect(Math.round(x), Math.round(y), 1, 1)
      }
    } else if (theme === 'jungle') {
      // drifting pollen motes in the light
      for (let i = 0; i < 16; i++) {
        const seed = i * 53.7
        const x = ((seed * 11 - cam * 0.25 + Math.sin(frame / 60 + i) * 10) % (W + 20) + W + 20) % (W + 20) - 10
        const y = 30 + ((seed * 5 + frame * 0.15) % 140)
        c.fillStyle = 'rgba(255,250,200,0.75)'
        c.fillRect(Math.round(x), Math.round(y), 1, 1)
      }
    } else {
      // glints on the far sea
      for (let i = 0; i < 10; i++) {
        const x = ((i * 61 - cam * 0.12 + frame * 0.05) % W + W) % W
        if (((frame >> 4) + i) % 3 === 0) {
          c.fillStyle = '#ffffff'
          c.fillRect(Math.round(x), 150 + ((i * 7) % 20), 2, 1)
        }
      }
    }
  }

  private liquid(theme: Theme, cam: number, frame: number) {
    const c = this.lctx
    const top = H - 22
    if (theme === 'jungle') {
      const g = c.createLinearGradient(0, H - 60, 0, H)
      g.addColorStop(0, 'rgba(10,30,26,0)')
      g.addColorStop(1, 'rgba(6,18,16,0.95)')
      c.fillStyle = g
      c.fillRect(0, H - 60, W, 60)
      return
    }
    const lava = theme === 'volcano'
    c.fillStyle = lava ? '#e2401c' : '#1aa6c8'
    c.fillRect(0, top, W, H - top)
    c.fillStyle = lava ? '#ff8a2a' : '#5fd6ee'
    for (let x = 0; x < W; x += 2) {
      const yy = top + Math.round(Math.sin((x + cam) / 9 + frame / 14) * 1.5)
      c.fillRect(x, yy, 2, 2)
    }
    c.fillStyle = lava ? '#ffe14d' : '#ffffff'
    for (let i = 0; i < 12; i++) {
      const x = ((i * 47 - cam + (frame >> 1)) % W + W) % W
      if (((frame >> 3) + i) % 4) c.fillRect(Math.round(x), top + 4 + ((i * 5) % 12), lava ? 2 : 3, 1)
    }
    c.fillStyle = lava ? '#b8260f' : '#1386a6'
    c.fillRect(0, top + 10, W, H - top - 10)
  }

  private spr(sh: Sheet, frame: number, x: number, y: number, left: boolean, white = false, alpha = 1) {
    const c = this.lctx
    const arr = white ? (left ? sh.wl : sh.wr) : left ? sh.l : sh.r
    const img = arr[((frame % arr.length) + arr.length) % arr.length]
    const ox = left ? sh.w - sh.ox : sh.ox
    if (alpha < 1) c.globalAlpha = alpha
    c.drawImage(img, Math.round(x - ox), Math.round(y - sh.oy))
    if (alpha < 1) c.globalAlpha = 1
  }

  private ent(e: REnt, cam: number, v: RView) {
    const a = this.art
    const c = this.lctx
    const x = e.x - cam
    const cx = x + e.w / 2
    const bottom = e.y + e.h
    if (cx < -80 || cx > W + 80) return
    const left = e.dir < 0
    const t = e.t
    const fl = e.flash > 0 && e.flash % 4 < 2
    if (e.st === -1 && e.k !== 'boss') {
      // knocked out: drawn upside down while it falls away
      const sh = this.sheetFor(e)
      if (!sh) return
      c.save()
      c.translate(Math.round(cx), Math.round(e.y + e.h / 2))
      c.scale(1, -1)
      this.spr(sh, 0, 0, sh.oy / 2, left)
      c.restore()
      return
    }
    switch (e.k) {
      case 'fruit': {
        const bob = e.c > 0 ? 0 : Math.round(Math.sin((v.frame + e.id * 13) / 14) * 1.5)
        const blink = e.c > 0 && e.c < 90 && e.c % 8 < 4
        if (!blink) this.spr(a.fruit[e.a] ?? a.fruit[0], 0, cx, e.y + e.h / 2 + bob, false)
        if ((v.frame + e.id * 7) % 50 === 0) this.parts.push({ x: cx + cam + 3, y: e.y + 2, vx: 0, vy: -0.2, g: 0, life: 14, max: 14, c: '#ffffff', s: 1, kind: 1 })
        return
      }
      case 'clam':
        this.spr(a.clam, e.st === 0 ? (v.frame % 120 < 6 ? 1 : 0) : e.st === 1 ? 1 : 2, cx, bottom + 1, false)
        if (e.st === 0 && v.frame % 40 === e.id % 40) this.parts.push({ x: cx + cam - 6 + (e.id % 12), y: e.y + 2, vx: 0, vy: -0.3, g: 0, life: 18, max: 18, c: '#fff6b0', s: 1, kind: 1 })
        return
      case 'item': {
        if (e.t > 600 && e.t % 8 < 4) return
        const bob = Math.round(Math.sin(v.frame / 8) * 1)
        const sh = [a.boom, a.board, a.fairy, a.cloud, a.basket, a.gold][e.a]
        if (e.a === IT_CLOUD) {
          this.spr(sh, (v.frame >> 4) % 2, cx, e.y + e.h / 2, e.vx < 0)
          return
        }
        // glow ring so pickups read at a glance
        c.fillStyle = 'rgba(255,255,220,0.35)'
        c.beginPath()
        c.arc(Math.round(cx), Math.round(e.y + e.h / 2 + bob), 10 + ((v.frame >> 3) % 2), 0, Math.PI * 2)
        c.fill()
        this.spr(sh, e.a === 0 || e.a === IT_GOLD ? (v.frame >> 3) % 4 : (v.frame >> 4) % 2, cx, e.a === 1 ? bottom + bob : e.y + e.h / 2 + bob + (sh.oy === sh.h ? sh.h / 2 : 0), false)
        return
      }
      case 'sun':
        this.spr(a.sun, (v.frame >> 3) % 2, cx, e.y + e.h / 2, false)
        return
      case 'rock':
        this.spr(a.rock, 0, cx, bottom + 1, false)
        return
      case 'fire':
        this.spr(a.fire, (v.frame >> 3) % 3, cx, bottom + 2, false)
        if (v.frame % 6 === 0) this.parts.push({ x: cx + cam + (Math.random() - 0.5) * 6, y: e.y, vx: (Math.random() - 0.5) * 0.3, vy: -0.6, g: -0.01, life: 26, max: 26, c: Math.random() < 0.5 ? '#ffb52b' : '#ff5a1f', s: 1, kind: 0 })
        return
      case 'crab':
        this.spr(a.crab, (t >> 3) % 2, cx, bottom + 1, !left, fl)
        return
      case 'gull':
        this.spr(a.gull, e.st === 1 ? 2 : (t >> 3) % 2, cx, e.y + e.h / 2, true, fl)
        return
      case 'fish':
        if (e.y > H) return
        this.spr(a.fish, e.vy > 0 ? 1 : 0, cx, e.y + e.h / 2, true, fl)
        return
      case 'frog':
        this.spr(a.frog, e.ground ? 0 : 1, cx, bottom + 1, left, fl)
        return
      case 'spider': {
        c.fillStyle = '#e8e8f0'
        c.fillRect(Math.round(cx), 0, 1, Math.round(e.y + 2))
        this.spr(a.spider, (t >> 3) % 2, cx, e.y + e.h / 2, false, fl)
        return
      }
      case 'spiky': {
        const fr = Math.floor(-e.c / 5) % 4
        this.spr(a.spiky, fr, cx, e.y + e.h / 2, false, fl)
        return
      }
      case 'boulder': {
        const fr = Math.floor(-e.c / 7) % 4
        this.spr(a.boulder, fr, cx, e.y + e.h / 2, false, fl)
        return
      }
      case 'bat':
        this.spr(a.bat, e.st === 0 ? 1 : (t >> 2) % 2, cx, e.y + e.h / 2, false, fl)
        return
      case 'flame':
        this.spr(a.flame, (v.frame >> 2) % 3, cx, bottom + 2, left, fl)
        if (v.frame % 5 === 0) this.parts.push({ x: cx + cam, y: e.y + 2, vx: 0, vy: -0.5, g: 0, life: 18, max: 18, c: '#ffb52b', s: 1, kind: 0 })
        return
      case 'geyser':
        this.geyser(e, cx, v.frame)
        return
      case 'spring':
        this.spr(a.spring, e.st > 0 ? 1 : 0, cx, bottom + 8, false)
        return
      case 'flag':
        this.spr(a.flag, e.st ? 1 + ((v.frame >> 4) % 2) : 0, cx, bottom, false)
        return
      case 'plat': {
        const th = stages()[v.stageIndex].theme
        const seg = e.a === 2 ? a.plat[3] : th === 'beach' ? a.plat[0] : th === 'jungle' ? a.plat[1] : a.plat[2]
        if (e.st === 3) return
        const jit = e.a === 2 && e.st === 1 ? (v.frame % 4 < 2 ? 1 : -1) : 0
        for (let i = 0; i < e.w; i += 16) c.drawImage(seg.r[0], Math.round(x + i + jit), Math.round(e.y))
        return
      }
      case 'boom': {
        const sh = e.a === 1 ? a.gold : a.boom
        this.spr(sh, Math.floor(t / 2), cx, e.y + e.h / 2, false)
        if (t % 2 === 0) this.parts.push({ x: cx + cam, y: e.y + e.h / 2, vx: 0, vy: 0, g: 0, life: 8, max: 8, c: e.a === 1 ? '#ffe14d' : '#ffd8a0', s: 1, kind: 0 })
        return
      }
      case 'shot':
        this.spr(a.shot[e.a] ?? a.shot[0], 0, cx, e.y + e.h / 2, false)
        if (e.a === 2 && t % 2 === 0) this.parts.push({ x: cx + cam, y: e.y + 4, vx: 0, vy: -0.3, g: 0, life: 14, max: 14, c: '#ff8a2a', s: 2, kind: 0 })
        return
      case 'boss':
        this.boss(e, x, v)
        return
      default:
    }
  }

  private sheetFor(e: REnt): Sheet | null {
    const a = this.art as unknown as Record<string, Sheet>
    return a[e.k] ?? null
  }

  private geyser(e: REnt, cx: number, frame: number) {
    const c = this.lctx
    const vine = e.a === 1
    if (e.st === 1) {
      // warning: bubbles / sprouts
      c.fillStyle = vine ? '#7bd43c' : frame % 6 < 3 ? '#ffe14d' : '#ff8a2a'
      for (let i = 0; i < 3; i++) c.fillRect(Math.round(cx - 5 + i * 4), Math.round(e.y + e.h - 3 - ((frame + i * 3) % 6)), 2, 2)
      return
    }
    if (e.st !== 2) return
    const top = Math.round(e.y)
    const hgt = Math.round(e.h)
    if (vine) {
      c.fillStyle = '#2f9e4a'
      c.fillRect(Math.round(cx - 4), top, 8, hgt)
      c.fillStyle = '#7bd43c'
      for (let y = top; y < top + hgt; y += 6) c.fillRect(Math.round(cx - 6 + ((y >> 2) % 2) * 9), y, 3, 2)
      c.fillStyle = '#ff4f8a'
      c.fillRect(Math.round(cx - 3), top - 2, 6, 3)
      return
    }
    const wob = Math.sin(frame / 3) * 1.5
    c.fillStyle = '#ff5a1f'
    c.fillRect(Math.round(cx - 7 + wob), top, 14, hgt)
    c.fillStyle = '#ffb52b'
    c.fillRect(Math.round(cx - 4 - wob), top + 2, 8, hgt - 2)
    c.fillStyle = '#fff3b0'
    c.fillRect(Math.round(cx - 1), top + 4, 2, hgt - 4)
    if (frame % 3 === 0) this.parts.push({ x: cx + this.lastCam + (Math.random() - 0.5) * 10, y: top, vx: (Math.random() - 0.5) * 1.2, vy: -1.5, g: 0.08, life: 20, max: 20, c: '#ffb52b', s: 2, kind: 0 })
  }

  private lastCam = 0

  private boss(e: REnt, x: number, v: RView) {
    const a = this.art
    const c = this.lctx
    const fl = e.flash > 0 && e.flash % 4 < 2
    const dying = e.st === 9
    const jx = dying ? Math.round((Math.random() - 0.5) * 4) : 0
    if (e.a === 0) {
      let fr = (e.t >> 4) % 2
      if (e.st === 3 && e.t <= 34) fr = 2
      if (e.st === 3 && e.t > 34) fr = 2
      if (e.st === 2) fr = 3
      if (e.st === 5 || dying) fr = 4
      this.spr(a.crabKing, fr, x + e.w / 2 + jx, e.y + e.h + 4, e.dir > 0 ? false : true, fl)
    } else if (e.a === 1) {
      // the stem grows from the arena floor on the right to the head
      const baseX = this.lastCam + W - 40 - this.lastCam
      const hx = x + e.w / 2
      const hy = e.y + e.h / 2
      c.strokeStyle = '#1b1028'
      c.lineWidth = 9
      c.beginPath()
      c.moveTo(baseX, H - 48)
      c.quadraticCurveTo(baseX + 10, (hy + H - 48) / 2, hx + 6, hy + 10)
      c.stroke()
      c.strokeStyle = '#2f9e4a'
      c.lineWidth = 6
      c.stroke()
      c.strokeStyle = '#5fd06a'
      c.lineWidth = 1
      c.stroke()
      let fr = (e.t >> 5) % 2
      if (e.st === 2 || e.st === 4) fr = 2
      if (dying) fr = 3
      // head frame: the sheet includes a stem stub; draw head centred on the hit box
      this.spr(a.bloom, fr, hx + jx, hy + 66, false, fl)
    } else {
      let fr = (e.t >> 4) % 2
      if (e.st === 2) fr = 2
      if (e.st === 5) fr = e.t < 30 ? 3 : 3 + ((e.t >> 2) % 2)
      if (e.st === 6) fr = (e.t >> 3) % 2 ? 5 : 0
      if (dying) fr = 5
      this.spr(a.tortoise, fr, x + e.w / 2 + jx, e.y + e.h + 6, e.dir < 0, fl)
      if (e.st === 6) {
        for (let i = 0; i < 3; i++) {
          const ang = v.frame / 8 + (i * Math.PI * 2) / 3
          c.fillStyle = '#ffe14d'
          c.fillRect(Math.round(x + e.w / 2 + (e.dir < 0 ? -26 : 26) + Math.cos(ang) * 8), Math.round(e.y - 8 + Math.sin(ang) * 3), 2, 2)
        }
      }
    }
  }

  private player(p: RPlayer, cam: number, frame: number, localIndex: number | null) {
    const c = this.lctx
    const sh = this.art.hero[p.i]
    const x = p.x - cam + 5
    const feet = p.y + 22
    if (p.st === 'ghost') {
      // bubble with the hero's head inside; the buddy pops it
      const bx = Math.round(x)
      const by = Math.round(p.y + 11)
      c.globalAlpha = 0.85
      c.drawImage(this.art.heads[p.i], bx - 11, by - 12)
      c.globalAlpha = 1
      c.strokeStyle = 'rgba(190,240,255,0.9)'
      c.lineWidth = 1
      c.beginPath()
      c.arc(bx + 0.5, by + 0.5, 14 + Math.sin(frame / 10), 0, Math.PI * 2)
      c.stroke()
      c.fillStyle = 'rgba(160,230,255,0.18)'
      c.fill()
      c.fillStyle = '#ffffff'
      c.fillRect(bx - 8, by - 9, 3, 2)
      c.fillRect(bx - 9, by - 7, 2, 2)
      return
    }
    if (p.st === 'dying') {
      if (p.dyingT >= 40) return
      this.spr(sh, sh.index.dead, x, feet, p.face < 0)
      return
    }
    if (p.inv > 0 && p.st === 'play' && frame % 6 < 3) return
    let f: number
    const ix = sh.index
    if (p.st === 'win') f = ix.win
    else if (p.trip > 0) f = ix.trip
    else if (p.throwT > 0) f = p.throwT > 9 ? ix.throw0 : ix.throw1
    else if (p.board) f = p.ground ? ix.ride : p.vy < 0 ? ix.jump : ix.fall
    else if (!p.ground) f = p.vy < 0 ? ix.jump : ix.fall
    else if (Math.abs(p.vx) > 0.2) f = ix.run0 + (Math.floor(p.runT / 7) % 6)
    else f = frame % 200 < 6 ? ix.blink : (frame >> 5) % 2 ? ix.idle1 : ix.idle0
    // fairy: a rainbow halo and a little star friend circling
    if (p.fairy > 0) {
      const hue = (frame * 8) % 360
      c.fillStyle = `hsla(${hue},100%,70%,0.35)`
      c.beginPath()
      c.ellipse(Math.round(x), Math.round(feet - 12), 13, 16, 0, 0, Math.PI * 2)
      c.fill()
    }
    const lift = p.board ? 5 : 0
    if (p.board) this.spr(this.art.board, 0, x, feet + 2, p.face < 0)
    this.spr(sh, f, x, feet - lift, p.face < 0, p.fairy > 0 && p.fairy < 120 && frame % 8 < 2)
    if (p.fairy > 0) {
      const ang = frame / 7
      this.spr(this.art.fairy, (frame >> 3) % 2, x + Math.cos(ang) * 16, feet - 14 + Math.sin(ang) * 8, false)
      if (frame % 3 === 0) this.parts.push({ x: x + cam + (Math.random() - 0.5) * 14, y: feet - Math.random() * 24, vx: 0, vy: -0.2, g: 0, life: 16, max: 16, c: `hsl(${(frame * 12) % 360},100%,75%)`, s: 1, kind: 1 })
    }
    if (p.cloud > 0) {
      const cx = x + Math.sin(frame / 20) * 3
      this.spr(this.art.cloud, (frame >> 4) % 2, cx, feet - 40, false)
      if (frame % 3 === 0) this.parts.push({ x: cx + cam + (Math.random() - 0.5) * 14, y: feet - 34, vx: 0, vy: 2.4, g: 0, life: 10, max: 10, c: '#8fc8ff', s: 1, kind: 0 })
    }
    if (p.board && p.ground && Math.abs(p.vx) > 1.5 && frame % 3 === 0) this.parts.push({ x: x + cam - p.face * 10, y: feet + 2, vx: -p.face * 0.6, vy: -0.4, g: 0.02, life: 14, max: 14, c: '#ffffff', s: 1, kind: 0 })
    if (p.ground && !p.board && Math.abs(p.vx) > 1.7 && frame % 9 === 0) this.parts.push({ x: x + cam - p.face * 4, y: feet, vx: -p.face * 0.3, vy: -0.3, g: 0.01, life: 12, max: 12, c: '#f3e1b8', s: 2, kind: 0 })
    // P1 / P2 tag over the head in two-player games (yours is gold online)
    if (localIndex !== null || this.twoTag) {
      const tag = `P${p.i + 1}`
      this.digits(tag, Math.round(x) - 3, Math.round(feet - 40), localIndex === p.i ? '#ffe14d' : p.i === 0 ? '#9fe6ff' : '#ffb0d8')
    }
  }

  twoTag = false

  private drawParticles(cam: number) {
    const c = this.lctx
    for (const p of this.parts) {
      const x = Math.round(p.x - cam)
      const y = Math.round(p.y)
      const k = p.life / p.max
      if (p.kind === 0) {
        c.fillStyle = p.c
        c.fillRect(x, y, p.s, p.s)
      } else if (p.kind === 1) {
        c.fillStyle = p.c
        c.fillRect(x, y - 1, 1, 3)
        c.fillRect(x - 1, y, 3, 1)
      } else if (p.kind === 2 && p.text) {
        if (k > 0.2 || p.life % 4 < 2) this.digits(p.text, x - p.text.length * 2, y, p.c)
      } else if (p.kind === 3) {
        c.strokeStyle = p.c
        c.globalAlpha = k
        c.lineWidth = 1
        c.beginPath()
        c.arc(x + 0.5, y + 0.5, p.s * (1.4 - k), 0, Math.PI * 2)
        c.stroke()
        c.globalAlpha = 1
      }
    }
    this.lastCam = cam
  }

  /** Outlined 3x5 pixel text (digits, x, +, P). */
  digits(s: string, x: number, y: number, color = '#ffffff') {
    const key = `${s}|${color}`
    let g = this.glyphs.get(key)
    if (!g) {
      const p = new Pix(s.length * 4 + 2, 7)
      const col = rgb(color)
      for (let i = 0; i < s.length; i++) {
        const rows = DIGITS[s[i]]
        if (!rows) continue
        for (let r = 0; r < 5; r++) for (let q = 0; q < 3; q++) if (rows[r][q] === '1') p.set(1 + i * 4 + q, 1 + r, col)
      }
      p.outline(OUTLINE, true)
      g = p.canvas()
      if (this.glyphs.size > 400) this.glyphs.clear()
      this.glyphs.set(key, g)
    }
    this.lctx.drawImage(g, x, y)
  }

  private hud(v: RView, stage: Stage) {
    const c = this.lctx
    const multi = v.players.filter((p) => p.on).length > 1
    this.twoTag = multi
    v.players.forEach((p) => {
      if (!p.on) return
      const right = p.i === 1
      const x0 = right ? W - 120 : 4
      // portrait
      c.drawImage(this.art.heads[p.i], right ? W - 26 : x0 - 1, 1)
      // energy bar: 20 segments
      const bx = right ? W - 112 : x0 + 24
      const by = 6
      c.fillStyle = '#1b1028'
      c.fillRect(bx - 1, by - 1, 84, 9)
      const frac = Math.max(0, p.energy) / ENERGY_MAX
      const segs = Math.ceil(frac * 20)
      const low = frac < 0.25
      const col = frac > 0.5 ? '#5ce65c' : frac > 0.25 ? '#ffd23a' : '#ff4f4f'
      for (let i = 0; i < 20; i++) {
        const k = right ? 19 - i : i
        const on = i < segs
        c.fillStyle = on ? (low && v.frame % 20 < 10 ? '#ffffff' : col) : '#3a2a4a'
        c.fillRect(bx + k * 4, by, 3, 7)
        if (on) {
          c.fillStyle = 'rgba(255,255,255,0.45)'
          c.fillRect(bx + k * 4, by, 3, 1)
        }
      }
      // fruit icon at the bar's end so the bar reads as "food"
      const fruit = this.art.fruit[1]
      c.drawImage(fruit.r[0], right ? bx - 15 : bx + 82, by - 4)
      // score
      const sc = String(p.score).padStart(6, '0')
      this.digits(sc, right ? W - 26 - sc.length * 4 - 4 : x0 + 24, 16, '#ffffff')
      // powers
      let ix = right ? W - 26 - 50 : x0 + 54
      const step = right ? -14 : 14
      if (p.weapon > 0) {
        c.drawImage((p.weapon === 2 ? this.art.gold : this.art.boom).r[0], ix, 14)
        ix += step
      }
      if (p.board) {
        c.drawImage(this.art.board.r[0], 0, 0, 28, 9, ix - 2, 17, 16, 5)
        ix += step
      }
      if (p.fairy > 0) {
        c.drawImage(this.art.fairy.r[0], ix, 13)
        ix += step
      }
      if (p.cloud > 0) c.drawImage(this.art.cloud.r[0], 0, 0, 22, 14, ix, 15, 14, 9)
    })
    // lives
    const lx = W / 2 - 16
    c.drawImage(this.art.fruit[5].r[0], lx, 2)
    this.digits(`x${Math.max(0, v.lives)}`, lx + 14, 5, '#ffe14d')
    // progress to the boss
    if (!v.lock) {
      const px = W / 2 - 50
      const pw = 100
      c.fillStyle = '#1b1028'
      c.fillRect(px - 1, 17, pw + 2, 4)
      c.fillStyle = '#ffffff'
      c.fillRect(px, 18, pw, 2)
      for (const cpx of stage.checkpoints.slice(1)) {
        c.fillStyle = '#38b6ff'
        c.fillRect(px + Math.round((cpx / stage.arenaX) * pw), 16, 2, 6)
      }
      c.fillStyle = '#ff4f4f'
      c.fillRect(px + pw - 2, 15, 4, 8)
      for (const p of v.players) {
        if (!p.on) continue
        const k = Math.max(0, Math.min(1, p.x / stage.arenaX))
        c.fillStyle = p.i === 0 ? '#1fb5a5' : '#ffd23a'
        c.fillRect(px + Math.round(k * pw) - 1, 16 + p.i * 0, 3, 6)
      }
    }
    // boss life
    const boss = v.ents.find((e) => e.k === 'boss')
    if (boss && boss.st !== 0) {
      const bw = 160
      const bx = W / 2 - bw / 2
      c.fillStyle = '#1b1028'
      c.fillRect(bx - 2, H - 12, bw + 4, 8)
      c.fillStyle = '#5a1830'
      c.fillRect(bx, H - 10, bw, 4)
      c.fillStyle = boss.flash > 0 ? '#ffffff' : '#ff4f4f'
      c.fillRect(bx, H - 10, Math.round((Math.max(0, boss.hp) / v.bossMax) * bw), 4)
      c.fillStyle = 'rgba(255,255,255,0.5)'
      c.fillRect(bx, H - 10, Math.round((Math.max(0, boss.hp) / v.bossMax) * bw), 1)
    }
  }
}

// ───────────────────────── stage tile layer ─────────────────────────

interface Pal {
  fill: Color
  fill2: Color
  deep: Color
  speck: Color[]
  top: Color
  topHi: Color
  blade: Color[]
  block: Color
}

const PALS: Record<Theme, Pal> = {
  beach: {
    fill: rgb('#e6bb72'),
    fill2: rgb('#d9a85d'),
    deep: rgb('#b9844a'),
    speck: [rgb('#f6dca0'), rgb('#c48d4e'), rgb('#fff1cc')],
    top: rgb('#f6d690'),
    topHi: rgb('#fff1c4'),
    blade: [rgb('#57c25a'), rgb('#3fa04a'), rgb('#8be06a')],
    block: rgb('#9d93a8')
  },
  jungle: {
    fill: rgb('#6b4330'),
    fill2: rgb('#5a3626'),
    deep: rgb('#43281c'),
    speck: [rgb('#7f5a44'), rgb('#3e2418'), rgb('#8a6a50')],
    top: rgb('#48b545'),
    topHi: rgb('#8be06a'),
    blade: [rgb('#48b545'), rgb('#2f8f3a'), rgb('#8be06a')],
    block: rgb('#7d8a6a')
  },
  volcano: {
    fill: rgb('#4e3349'),
    fill2: rgb('#42293e'),
    deep: rgb('#2e1a2c'),
    speck: [rgb('#644560'), rgb('#2a1828'), rgb('#ff7a2a')],
    top: rgb('#6e5068'),
    topHi: rgb('#9a7a92'),
    blade: [rgb('#3a2638'), rgb('#ff8a2a'), rgb('#6e5068')],
    block: rgb('#5d4a66')
  }
}

export function buildStageLayer(stage: Stage): Pix {
  const theme = stage.theme
  const pal = PALS[theme]
  const cols = stage.cols
  const p = new Pix(cols * T, ROWS * T)
  const R = rng(stage.id * 1000 + 7)
  const at = (c: number, r: number) => (c < 0 || c >= cols || r < 0 || r >= ROWS ? (r >= ROWS ? TS : 0) : stage.tiles[r * cols + c])
  const solid = (t: number) => t === TS || t === TB
  for (let cIdx = 0; cIdx < cols; cIdx++) {
    for (let r = 0; r < ROWS; r++) {
      const t = at(cIdx, r)
      const x0 = cIdx * T
      const y0 = r * T
      if (t === TS) {
        // depth below the surface decides the shade
        let depth = 0
        while (depth < 4 && solid(at(cIdx, r - depth - 1))) depth++
        const base = depth >= 2 ? mix(pal.fill2, pal.deep, Math.min(1, (depth - 2) / 2)) : pal.fill
        p.rect(x0, y0, T, T, base)
        for (let k = 0; k < 7; k++) p.set(x0 + Math.floor(R() * T), y0 + Math.floor(R() * T), pal.speck[Math.floor(R() * pal.speck.length)])
        if (theme === 'beach' && R() < 0.3) p.rect(x0 + Math.floor(R() * 12), y0 + 6 + Math.floor(R() * 8), 4, 1, shadowOf(base, 0.12))
        if (theme === 'jungle' && R() < 0.25) {
          const sx = x0 + Math.floor(R() * 10)
          p.rect(sx, y0 + 4 + Math.floor(R() * 8), 4, 3, rgb('#8a7a70'))
        }
        if (theme === 'volcano' && R() < 0.18) {
          const sx = x0 + Math.floor(R() * 12)
          const sy = y0 + 4 + Math.floor(R() * 10)
          p.line(sx, sy, sx + 3, sy + 2, rgb('#ff7a2a'))
        }
        if (!solid(at(cIdx, r - 1))) {
          // surface band
          const band = theme === 'jungle' ? 5 : 4
          p.rect(x0, y0, T, band, pal.top)
          p.rect(x0, y0, T, 1, pal.topHi)
          for (let i = 0; i < T; i++) if (R() < 0.5) p.set(x0 + i, y0 + band, pal.top)
          if (theme === 'beach') for (let i = 0; i < 3; i++) p.set(x0 + Math.floor(R() * T), y0 + 1 + Math.floor(R() * 3), pal.topHi)
          if (theme === 'volcano') p.rect(x0, y0 + band, T, 1, shadowOf(pal.top, 0.3))
          // blades / tufts poking above the surface
          const n = theme === 'jungle' ? 5 : theme === 'beach' ? (R() < 0.45 ? 3 : 0) : R() < 0.3 ? 2 : 0
          for (let i = 0; i < n; i++) {
            const bx = x0 + Math.floor(R() * T)
            const hgt = 1 + Math.floor(R() * (theme === 'jungle' ? 4 : 3))
            const col = pal.blade[Math.floor(R() * pal.blade.length)]
            for (let k = 1; k <= hgt; k++) p.set(bx + (k > 2 ? (R() < 0.5 ? 1 : -1) : 0), y0 - k, col)
          }
        }
        if (!solid(at(cIdx - 1, r))) p.rect(x0, y0, 1, T, lightOf(base, 0.2))
        if (!solid(at(cIdx + 1, r))) p.rect(x0 + T - 1, y0, 1, T, shadowOf(base, 0.3))
      } else if (t === TB) {
        p.rect(x0, y0, T, T, pal.block)
        p.rect(x0, y0, T, 1, lightOf(pal.block, 0.3))
        p.rect(x0, y0 + 7, T, 1, shadowOf(pal.block, 0.3))
        p.rect(x0 + ((r % 2) * 8 + 4), y0, 1, 7, shadowOf(pal.block, 0.3))
        p.rect(x0 + ((r % 2) * 8 + 12) % 16, y0 + 8, 1, 8, shadowOf(pal.block, 0.3))
      } else if (t === TO) {
        drawPlank(p, theme, x0, y0, at(cIdx - 1, r) !== TO, at(cIdx + 1, r) !== TO)
      }
    }
  }
  // surface decorations
  for (let cIdx = 1; cIdx < cols - 1; cIdx++) {
    for (let r = 1; r < ROWS; r++) {
      if (at(cIdx, r) !== TS || solid(at(cIdx, r - 1)) || at(cIdx, r - 1) === TO) continue
      if (R() > 0.22) continue
      decorate(p, theme, cIdx * T + 2 + Math.floor(R() * 10), r * T, R)
    }
  }
  p.outline(OUTLINE)
  void TW
  void TL
  return p
}

function drawPlank(p: Pix, theme: Theme, x0: number, y0: number, leftEnd: boolean, rightEnd: boolean) {
  if (theme === 'beach') {
    const wood = rgb('#c48a4a')
    p.rect(x0, y0, T, 5, wood)
    p.rect(x0, y0, T, 1, lightOf(wood, 0.35))
    p.rect(x0, y0 + 4, T, 1, shadowOf(wood, 0.3))
    p.rect(x0 + 7, y0, 1, 5, shadowOf(wood, 0.25))
    p.rect(x0 + 3, y0 + 5, 2, 6, rgb('#8a5a33'))
    p.rect(x0 + 11, y0 + 5, 2, 4, rgb('#8a5a33'))
  } else if (theme === 'jungle') {
    const wood = rgb('#9a6438')
    p.rect(x0, y0 + 1, T, 6, wood)
    p.rect(x0, y0 + 1, T, 1, lightOf(wood, 0.35))
    p.rect(x0, y0 + 6, T, 1, shadowOf(wood, 0.35))
    p.set(x0 + 5, y0 + 3, shadowOf(wood, 0.4))
    p.set(x0 + 11, y0 + 4, shadowOf(wood, 0.4))
    if (leftEnd) p.ellipse(x0 + 1, y0 + 4, 2, 3, lightOf(wood, 0.25))
    if (rightEnd) p.ellipse(x0 + 15, y0 + 4, 2, 3, lightOf(wood, 0.25))
    p.set(x0 + 8, y0, rgb('#48b545'))
    p.set(x0 + 9, y0, rgb('#48b545'))
  } else {
    const st = rgb('#5d4a66')
    p.rect(x0, y0, T, 7, st)
    p.rect(x0, y0, T, 1, lightOf(st, 0.35))
    p.rect(x0, y0 + 2, T, 1, rgb('#ff8a2a'))
    p.rect(x0, y0 + 6, T, 1, shadowOf(st, 0.35))
  }
}

function decorate(p: Pix, theme: Theme, x: number, y: number, R: () => number) {
  const k = R()
  if (theme === 'beach') {
    if (k < 0.35) {
      // shell
      p.rect(x, y - 2, 3, 2, rgb('#ffd9e6'))
      p.set(x + 1, y - 3, rgb('#ffffff'))
    } else if (k < 0.55) {
      // starfish
      const c = rgb('#ff8a5a')
      p.set(x + 1, y - 3, c)
      p.rect(x, y - 2, 3, 1, c)
      p.set(x, y - 1, c)
      p.set(x + 2, y - 1, c)
    } else {
      // little beach plant
      const g = rgb('#3fa04a')
      p.line(x, y - 1, x - 2, y - 5, g)
      p.line(x + 1, y - 1, x + 3, y - 6, g)
      p.line(x + 1, y - 1, x + 1, y - 7, rgb('#57c25a'))
    }
  } else if (theme === 'jungle') {
    if (k < 0.4) {
      const petal = k < 0.2 ? rgb('#ff4f6a') : rgb('#ffd23a')
      p.line(x + 1, y - 1, x + 1, y - 4, rgb('#2f8f3a'))
      p.rect(x, y - 6, 3, 2, petal)
      p.set(x + 1, y - 7, petal)
      p.set(x + 1, y - 5, rgb('#fff3b0'))
    } else if (k < 0.65) {
      p.rect(x, y - 3, 4, 2, rgb('#e8463c'))
      p.set(x + 1, y - 3, rgb('#ffffff'))
      p.rect(x + 1, y - 1, 2, 1, rgb('#f2e6c8'))
    } else {
      const g = rgb('#2f8f3a')
      for (let i = 0; i < 4; i++) p.line(x + 2, y - 1, x - 2 + i * 2, y - 6 - (i % 2), g)
    }
  } else {
    if (k < 0.5) {
      p.rect(x, y - 2, 4, 2, rgb('#3a2638'))
      p.set(x + 1, y - 3, rgb('#5d4a66'))
    } else {
      p.rect(x, y - 3, 2, 3, rgb('#ff5fa2'))
      p.set(x, y - 4, rgb('#ffb0d8'))
      p.rect(x + 2, y - 2, 2, 2, rgb('#c03a8a'))
    }
  }
}
