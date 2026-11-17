// Drawing 棉花糖大作战: the stage, kids, pests and candy balls go onto a 320×240 canvas at 1:1,
// which is then blown up with hard pixels onto the page canvas. Banners and the short Chinese
// labels are drawn afterwards at full resolution so they stay sharp. Particles, score pops,
// screen shake and flashes react to the simulation's events here.

import { FLOORS, worldOf, BOSS_NAMES, type World } from './levels'
import { ALIVE, GHOST, H, ITEM_BLUE, ITEM_LIFE, ITEM_PACKET, ITEM_RED, ITEM_YELLOW, SNOW_FULL, T, W, COLS, ROWS, type Enemy, type Ev, type GameState, type Player } from './sim'
import { buildSheet, digits, KID_COLORS, textWidth, type Sheet, type Sprite } from './sprites'

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  life: number
  max: number
  color: string
  size: number
  grav: number
}

interface Pop {
  x: number
  y: number
  text: string
  color: string
  life: number
  big: boolean
}

interface Banner {
  text: string
  sub: string
  color: string
  life: number
  max: number
}

const FLOSS = ['#ff86c3', '#ffc9e6', '#ffffff', '#d54b98']
const CONFETTI = ['#ffd23f', '#ff86c3', '#7cc8ff', '#6ccb6a', '#ff9a3c', '#c8b6f5', '#ffffff']
const ITEM_NAMES: Record<number, string> = {
  [ITEM_RED]: '力量汽水',
  [ITEM_BLUE]: '速度汽水',
  [ITEM_YELLOW]: '射程汽水',
  [ITEM_LIFE]: '糖葫芦 +1 命'
}

const SHOT = ['..pp...', '.pPPp..', 'pPPWPp.', 'pPPPPPp', '.pPPPp.', '..pp...']
const SHOT_BIG = ['...ppp...', '.ppPPPpp.', 'pPPPWPPPp', 'pPPWWPPPp', 'pPPPPPPPp', '.pPPPPPp.', '..ppppp..']

export class Renderer {
  readonly lo: HTMLCanvasElement
  private ctx: CanvasRenderingContext2D
  private out: CanvasRenderingContext2D
  private sheet: Sheet
  private stage: HTMLCanvasElement | null = null
  private stageFloor = -1
  private bgs = new Map<string, HTMLImageElement>()
  private parts: Particle[] = []
  private pops: Pop[] = []
  private banner: Banner | null = null
  private shake = 0
  private flash = 0
  private flashColor = '#ffffff'
  private shots: [HTMLCanvasElement, HTMLCanvasElement]
  /** where the 320×240 picture sits on the page canvas */
  view = { x: 0, y: 0, scale: 1 }
  scanlines = true
  /** frames to freeze the game for after a big hit (read and cleared by the game loop) */
  hitstop = 0
  private tick = 0

  constructor(
    private display: HTMLCanvasElement,
    mediaBase: string
  ) {
    this.lo = document.createElement('canvas')
    this.lo.width = W
    this.lo.height = H
    this.ctx = this.lo.getContext('2d')!
    this.ctx.imageSmoothingEnabled = false
    this.out = display.getContext('2d')!
    this.sheet = buildSheet()
    this.shots = [this.small(SHOT), this.small(SHOT_BIG)]
    for (const w of ['bg-bedroom.png', 'bg-classroom.png', 'bg-rooftop.png']) {
      const img = new Image()
      img.src = `${mediaBase}${w}`
      this.bgs.set(w, img)
    }
  }

  get sprites(): Sheet {
    return this.sheet
  }

  private small(rows: string[]): HTMLCanvasElement {
    const c = document.createElement('canvas')
    c.width = rows[0].length + 2
    c.height = rows.length + 2
    const x = c.getContext('2d')!
    const pal: Record<string, string> = { p: '#ffc9e6', P: '#ff86c3', W: '#ffffff' }
    rows.forEach((r, y) => {
      for (let i = 0; i < r.length; i++) {
        if (r[i] === '.') continue
        x.fillStyle = pal[r[i]]
        x.fillRect(i + 1, y + 1, 1, 1)
      }
    })
    // soft outline in deep pink so the floss reads on dark walls
    const img = x.getImageData(0, 0, c.width, c.height)
    const d = img.data
    const solid = (i: number, j: number) => i >= 0 && j >= 0 && i < c.width && j < c.height && d[(j * c.width + i) * 4 + 3] > 0
    const add: number[] = []
    for (let j = 0; j < c.height; j++) for (let i = 0; i < c.width; i++) if (!solid(i, j) && (solid(i - 1, j) || solid(i + 1, j) || solid(i, j - 1) || solid(i, j + 1))) add.push(i, j)
    x.fillStyle = '#9c2d6c'
    for (let k = 0; k < add.length; k += 2) x.fillRect(add[k], add[k + 1], 1, 1)
    return c
  }

  resize(): void {
    const dpr = Math.min(window.devicePixelRatio || 1, 2)
    const cw = this.display.clientWidth
    const ch = this.display.clientHeight
    this.display.width = Math.round(cw * dpr)
    this.display.height = Math.round(ch * dpr)
    // keep a strip at the top for the page's buttons (sound, pause, room code)
    const top = Math.round(46 * dpr)
    let scale = Math.min(this.display.width / W, (this.display.height - top) / H)
    if (scale >= 3) scale = Math.floor(scale)
    this.view.scale = scale
    this.view.x = Math.round((this.display.width - W * scale) / 2)
    this.view.y = top + Math.round((this.display.height - top - H * scale) / 2)
    // tall phone screens: picture at the top, the touch pad gets the space below
    if (this.display.height > this.display.width * 1.3) this.view.y = Math.min(this.view.y, Math.round(56 * dpr))
  }

  // ------------------------------------------------------------ effects from events

  private burst(x: number, y: number, n: number, colors: string[], speed: number, size = 2, grav = 0.12, life = 30): void {
    for (let i = 0; i < n; i++) {
      const a = Math.random() * Math.PI * 2
      const v = speed * (0.4 + Math.random() * 0.8)
      this.parts.push({ x, y, vx: Math.cos(a) * v, vy: Math.sin(a) * v - speed * 0.3, life: life * (0.6 + Math.random() * 0.6), max: life, color: colors[Math.floor(Math.random() * colors.length)], size, grav })
    }
  }

  private pop(x: number, y: number, text: string, color: string, big = false): void {
    this.pops.push({ x, y, text, color, life: big ? 70 : 50, big })
  }

  showBanner(text: string, sub = '', color = '#ffffff', frames = 110): void {
    this.banner = { text, sub, color, life: frames, max: frames }
  }

  private addShake(n: number): void {
    this.shake = Math.max(this.shake, n)
  }

  react(evs: Ev[], s: GameState): void {
    for (const e of evs) {
      switch (e.k) {
        case 'shot':
          this.burst(e.x, e.y, 3, ['#ffffff', '#ffc9e6'], 0.8, 1, 0, 12)
          break
        case 'hit':
          this.burst(e.x, e.y, 7, FLOSS, 1.4, 2, 0.05, 22)
          break
        case 'ball':
          this.burst(e.x, e.y, 16, FLOSS, 2, 2, 0.04, 30)
          this.burst(e.x, e.y, 6, ['#ffffff', '#fff2a0'], 2.6, 1, 0, 18)
          this.addShake(1.5)
          break
        case 'nudge':
          this.burst(e.x, e.y, 4, FLOSS, 1, 1, 0.05, 14)
          break
        case 'kick':
          this.burst(e.x, e.y, 10, ['#ffffff', '#ffd23f', '#fff2a0'], 3.4, 1, 0, 14)
          this.addShake(e.p === 9 ? 2 : 3)
          this.hitstop = Math.max(this.hitstop, e.p === 9 ? 2 : 4)
          break
        case 'bounce':
          this.burst(e.x, e.y, 6, ['#ffffff', '#ffc9e6'], 1.6, 1, 0.08, 16)
          this.addShake(1.5)
          break
        case 'kill': {
          const chain = e.p ?? 1
          this.burst(e.x, e.y, 10 + chain * 3, CONFETTI, 2.2 + chain * 0.25, 2, 0.1, 34)
          this.pop(e.x, e.y - 6, String(e.n ?? 0), chain >= 3 ? '#ffd23f' : '#ffffff', chain >= 2)
          if (chain >= 2) this.pop(e.x, e.y - 16, `${chain} 连锁！`, chain >= 4 ? '#ff5a8a' : '#ffd23f', true)
          this.addShake(1.5 + Math.min(4, chain * 0.7))
          this.hitstop = Math.max(this.hitstop, Math.min(6, 1 + chain))
          break
        }
        case 'burst':
          this.burst(e.x, e.y, 24, FLOSS, 2.8, 2, 0.06, 34)
          if (e.n) this.pop(e.x, e.y - 6, String(e.n), '#ffffff', (e.p ?? 1) >= 2)
          if ((e.p ?? 1) >= 2) this.pop(e.x, e.y - 16, `${e.p} 连锁！`, '#ffd23f', true)
          this.addShake(2.5)
          break
        case 'thaw':
          this.burst(e.x, e.y, 12, FLOSS, 2, 2, 0.15, 24)
          this.pop(e.x, e.y - 10, '生气了！', '#ff5a3c')
          break
        case 'pickup': {
          const k = e.p ?? 0
          this.burst(e.x, e.y, 8, ['#ffffff', '#ffd23f'], 1.4, 1, 0, 18)
          if (ITEM_NAMES[k]) this.pop(e.x, e.y - 8, ITEM_NAMES[k], '#7cf0ff', true)
          else if (e.n) this.pop(e.x, e.y - 6, String(e.n), k === ITEM_PACKET ? '#ff5a5a' : '#fff2a0')
          break
        }
        case 'die':
          this.burst(e.x, e.y, 26, ['#ffffff', '#ff86c3', KID_COLORS[e.p ?? 0].css], 2.6, 2, 0.08, 40)
          this.addShake(4)
          this.flash = 6
          this.flashColor = '#ff3b5c'
          this.hitstop = 8
          break
        case 'revive':
          this.burst(e.x, e.y, 18, ['#ffffff', '#ff86c3', '#ffd23f'], 2, 2, -0.02, 36)
          this.pop(e.x, e.y - 14, '复活！', '#7cf0ff', true)
          break
        case 'respawn':
          this.burst(e.x, e.y, 14, ['#ffffff', '#fff2a0'], 1.8, 1, -0.02, 28)
          break
        case 'bump':
          this.burst(e.x, e.y - 8, 5, ['#ffd23f', '#ffffff'], 1.2, 1, 0, 16)
          break
        case 'land':
          this.burst(e.x, e.y, e.p === 9 ? 6 : 4, ['#d9cde6', '#ffffff'], 0.9, 1, 0.02, 14)
          break
        case 'jump':
          this.burst(e.x, e.y, 3, ['#d9cde6'], 0.6, 1, 0, 10)
          break
        case 'push':
          this.burst(e.x, e.y, 2, ['#d9cde6'], 0.5, 1, 0, 10)
          break
        case 'spawn':
          this.burst(e.x, e.y, 10, ['#ffffff', '#c8b6f5'], 1.6, 1, 0, 18)
          break
        case 'poof':
          this.burst(e.x, e.y, 4, FLOSS, 0.8, 1, 0.02, 12)
          if (e.n) this.pop(e.x, e.y - 4, String(e.n), '#ffffff')
          break
        case 'quake':
          this.addShake(5)
          this.burst(e.x, e.y, 14, ['#d9cde6', '#ffffff'], 2, 2, 0.1, 22)
          break
        case 'bosshit':
          this.burst(e.x, e.y, (e.n ?? 1) > 4 ? 18 : 5, ['#ffffff', '#ffd23f'], 2.4, 1, 0.05, 16)
          if ((e.n ?? 0) > 4) {
            this.addShake(4)
            this.hitstop = 5
            this.pop(e.x, e.y - 8, `-${e.n}`, '#ffd23f', true)
          }
          break
        case 'bossdie':
          this.addShake(8)
          this.flash = 12
          this.flashColor = '#ffffff'
          this.burst(e.x, e.y, 60, CONFETTI, 4, 2, 0.08, 60)
          this.showBanner(`打败了${BOSS_NAMES[FLOORS[s.floor].boss ?? 'bigclock']}！`, '每人 +10000', '#ffd23f', 140)
          break
        case 'hurry':
          this.showBanner('快！妈妈要来查房了！', '再不清场，拖鞋就飞过来了', '#ff8a5c', 130)
          break
        case 'slipper':
          this.addShake(2)
          break
        case 'sweep':
          this.showBanner('一球清场！', '红包雨，快捡！', '#ff5a5a', 160)
          this.flash = 8
          this.flashColor = '#ffd23f'
          break
        case 'clear':
          if (!s.sweep && !FLOORS[s.floor].boss) this.showBanner('清场！', '捡完零食就上楼', '#7cf0ff', 100)
          break
        case 'won':
          this.showBanner('暑假保住了！', '开学怪被撕成了一沓草稿纸', '#fff2a0', 380)
          this.flash = 10
          this.flashColor = '#ffffff'
          this.burst(e.x, e.y, 80, CONFETTI, 4, 2, 0.06, 90)
          break
        case 'over':
          this.showBanner('妈妈喊你睡觉了', '', '#ff8a9a', 140)
          break
        case 'floor': {
          const def = FLOORS[e.n ?? 0]
          const world = worldOf(e.n ?? 0)
          if (def.boss) this.showBanner(`第 ${(e.n ?? 0) + 1} 层 · ${BOSS_NAMES[def.boss]}`, bossLine(def.boss), '#ff6b7a', 130)
          else this.showBanner(`第 ${(e.n ?? 0) + 1} 层`, `${world.name} · 把它们裹成棉花糖！`, '#ffffff', 110)
          break
        }
        default:
          break
      }
    }
  }

  // ------------------------------------------------------------ stage

  private buildStage(s: GameState, world: World): HTMLCanvasElement {
    const c = document.createElement('canvas')
    c.width = W
    c.height = H
    const x = c.getContext('2d')!
    const [hi, face, shade, dark] = world.shelf
    const [g0, g1, g2] = world.ground
    const at = (cc: number, r: number) => (cc < 0 || cc >= COLS || r < 0 || r >= ROWS ? 1 : s.tiles[r * COLS + cc])
    for (let r = 2; r < ROWS; r++) {
      for (let cc = 0; cc < COLS; cc++) {
        const t = at(cc, r)
        const px = cc * T
        const py = r * T
        if (t === 2) {
          const left = at(cc - 1, r) !== 2
          const right = at(cc + 1, r) !== 2
          // soft shadow on the wall below the shelf
          x.fillStyle = 'rgba(10,6,24,0.35)'
          x.fillRect(px, py + 7, 8, 4)
          x.fillStyle = dark
          x.fillRect(px, py, 8, 7)
          x.fillStyle = face
          x.fillRect(px + (left ? 1 : 0), py + 1, 8 - (left ? 1 : 0) - (right ? 1 : 0), 4)
          x.fillStyle = hi
          x.fillRect(px + (left ? 1 : 0), py + 1, 8 - (left ? 1 : 0) - (right ? 1 : 0), 1)
          x.fillStyle = shade
          x.fillRect(px + (left ? 1 : 0), py + 5, 8 - (left ? 1 : 0) - (right ? 1 : 0), 1)
          if (cc % 3 === 0 && !left && !right) {
            x.fillStyle = shade
            x.fillRect(px, py + 2, 1, 3)
          }
          if (left || right) {
            // brackets under the ends of a shelf
            x.fillStyle = dark
            x.fillRect(px + (left ? 2 : 4), py + 7, 2, 3)
          }
        } else if (t === 1 && r >= 28) {
          x.fillStyle = r === 28 ? g0 : g1
          x.fillRect(px, py, 8, 8)
          x.fillStyle = g2
          if (r === 28) {
            x.fillRect(px, py + 7, 8, 1)
            if (cc % 2 === 0) x.fillRect(px, py + 1, 1, 6)
          } else if (cc % 2 === 1) x.fillRect(px, py, 1, 8)
          if (r === 28) {
            x.fillStyle = 'rgba(255,255,255,0.25)'
            x.fillRect(px, py, 8, 1)
          }
        } else if (t === 1 && (cc === 0 || cc === COLS - 1)) {
          x.fillStyle = dark
          x.fillRect(px, py, 8, 8)
          x.fillStyle = shade
          x.fillRect(px + (cc === 0 ? 5 : 1), py, 2, 8)
          if (r % 4 === 0) {
            x.fillStyle = face
            x.fillRect(px + 1, py, 6, 1)
          }
        } else if (t === 1 && r > 1) {
          // a toy chest / water tank block
          x.fillStyle = dark
          x.fillRect(px, py, 8, 8)
          x.fillStyle = face
          x.fillRect(px + 1, py + 1, 6, 6)
          x.fillStyle = hi
          x.fillRect(px + 1, py + 1, 6, 1)
          x.fillStyle = shade
          x.fillRect(px + 1, py + 4, 6, 1)
        }
      }
    }
    return c
  }

  // ------------------------------------------------------------ frame

  draw(s: GameState, localIdx: number[], net: { ping: number } | null): void {
    const ctx = this.ctx
    const world = worldOf(s.floor)
    if (this.stageFloor !== s.floor || !this.stage) {
      this.stage = this.buildStage(s, world)
      this.stageFloor = s.floor
    }
    const bg = this.bgs.get(world.bg)
    if (bg && bg.complete && bg.naturalWidth) ctx.drawImage(bg, 0, 0, W, H)
    else {
      ctx.fillStyle = '#2a2350'
      ctx.fillRect(0, 0, W, H)
    }
    ctx.drawImage(this.stage, 0, 0)

    // items
    for (const it of s.items) {
      if (it.t > 480 && Math.floor(it.t / 4) % 2) continue
      const spr = this.sheet.items[it.kind]
      const bob = it.t > 20 ? Math.round(Math.sin((it.t + it.id * 7) / 10)) : 0
      ctx.drawImage(spr, Math.round(it.x + 5 - spr.width / 2), Math.round(it.y + 10 - spr.height + 1 + bob))
    }
    // pests and balls
    for (const e of s.enemies) this.drawEnemy(e)
    // kids
    for (const p of s.players) if (p.on) this.drawPlayer(s, p, localIdx.length > 1 || localIdx[0] !== p.idx)
    // candy floss shots
    for (const sh of s.shots) {
      const spr = this.shots[sh.big ? 1 : 0]
      ctx.drawImage(spr, Math.round(sh.x - 1), Math.round(sh.y - 1 + Math.sin(sh.t / 2) * 0.6))
    }
    // pests' projectiles
    for (const pr of s.projs) {
      const pair = this.sheet.proj[pr.kind]
      if (!pair) continue
      const spr = pr.vx < 0 ? pair[1] : pair[0]
      if (pr.kind === 'wave' && this.tick % 4 < 2) ctx.globalAlpha = 0.7
      ctx.drawImage(spr, Math.round(pr.x + pr.w / 2 - spr.width / 2), Math.round(pr.y + pr.h / 2 - spr.height / 2))
      ctx.globalAlpha = 1
    }
    this.drawParticles()
    this.drawHud(s, net)
    this.present(s)
  }

  private drawEnemy(e: Enemy): void {
    const ctx = this.ctx
    const sh = this.sheet
    if (e.spawnT > 0) {
      if (Math.floor(e.spawnT / 3) % 2) return
    }
    if (e.boss) {
      const frames = (e.angry ? sh.bossesAngry : sh.bosses)[e.kind]
      const fi = Math.floor(e.t / (e.angry ? 8 : 14)) % 2
      let spr = frames[fi][e.face > 0 ? 0 : 1]
      if (e.flash > 0 && e.flash % 4 < 2) spr = sh.bossFlash[e.kind]
      let dx = 0
      let dy = 0
      if (e.dying) {
        dx = (Math.random() - 0.5) * 4
        dy = (Math.random() - 0.5) * 4
        if (Math.floor(e.dying / 4) % 2) return
      }
      ctx.drawImage(spr, Math.round(e.x + e.w / 2 - spr.width / 2 + dx), Math.round(e.y + e.h - spr.height + 2 + dy))
      return
    }
    if (e.snow >= SNOW_FULL && !e.dying) {
      let spr: Sprite
      let dx = 0
      if (e.rolling) {
        const i = ((Math.floor(e.x / 3) % 8) + 8) % 8
        spr = sh.balls[e.vx > 0 ? i : 7 - i]
      } else if (e.ballT > 330) {
        spr = sh.ballsCrack[e.ballT > 390 ? 1 : 0][0]
        dx = this.tick % 4 < 2 ? -1 : 1
      } else spr = sh.balls[0]
      const by = Math.round(e.y + e.h - spr.height + 2)
      ctx.drawImage(spr, Math.round(e.x + e.w / 2 - spr.width / 2 + dx), by)
      if (!e.rolling) {
        // little feet stick out of the ball
        ctx.fillStyle = '#5e3d2c'
        ctx.fillRect(Math.round(e.x + 4), Math.round(e.y + e.h - 1), 2, 1)
        ctx.fillRect(Math.round(e.x + e.w - 6), Math.round(e.y + e.h - 1), 2, 1)
      }
      return
    }
    const frames = sh.pests[e.kind]
    if (!frames) return
    const moving = Math.abs(e.vx) > 0.05 || !e.ground || e.kind === 'mosquito'
    let fi = frames.length > 1 && moving ? Math.floor(e.t / (e.kind === 'mosquito' ? 3 : 9)) % frames.length : 0
    // the coil rolls: frame follows the distance travelled
    if (e.kind === 'coil') fi = (((Math.floor(e.x / 3) * (e.face > 0 ? 1 : -1)) % 4) + 4) % 4
    const pair = (e.angry && !e.dying && this.tick % 10 < 5 ? sh.white[e.kind] : frames)[fi]
    const spr = pair[e.face > 0 ? 0 : 1]
    const x = Math.round(e.x + e.w / 2 - spr.width / 2)
    const y = Math.round(e.y + e.h - spr.height + 1)
    if (e.dying) {
      ctx.save()
      ctx.translate(x + spr.width / 2, y + spr.height / 2)
      ctx.rotate(e.dying * 0.35 * (e.vx >= 0 ? 1 : -1))
      ctx.drawImage(spr, -spr.width / 2, -spr.height / 2)
      ctx.restore()
      return
    }
    if (e.kind === 'slipper') {
      ctx.save()
      ctx.translate(x + spr.width / 2, y + spr.height / 2)
      ctx.rotate(Math.round(Math.sin(e.t / 6) * 2) * 0.2)
      ctx.drawImage(spr, -spr.width / 2, -spr.height / 2)
      ctx.restore()
      if (this.tick % 6 === 0) this.parts.push({ x: e.x + e.w / 2, y: e.y, vx: 0, vy: -0.4, life: 18, max: 18, color: '#ffffff', size: 1, grav: 0 })
      return
    }
    ctx.drawImage(spr, x, y)
    if (e.snow > 0) {
      const lvl = Math.min(3, Math.ceil(e.snow))
      const fl = sh.floss(e.w, e.h, lvl)
      ctx.drawImage(fl, Math.round(e.x - 5), Math.round(e.y + e.h - fl.height + 3))
    }
  }

  private drawPlayer(s: GameState, p: Player, label: boolean): void {
    const ctx = this.ctx
    const kid = this.sheet.kids[p.idx]
    if (p.state === GHOST) {
      const bob = Math.round(Math.sin(this.tick / 12) * 1.5)
      const blink = p.ghostT > 0 && p.ghostT < 120 && this.tick % 8 < 4
      if (!blink) ctx.drawImage(kid.ghost, Math.round(p.x + p.w / 2 - kid.ghost.width / 2), Math.round(p.y - 4 + bob))
      // countdown ring of dots
      if (p.ghostT > 0) {
        const n = Math.ceil(p.ghostT / 60)
        for (let i = 0; i < n; i++) {
          ctx.fillStyle = '#ffffff'
          ctx.fillRect(Math.round(p.x + p.w / 2 - n * 1.5 + i * 3), Math.round(p.y - 8 + bob), 2, 2)
        }
      }
      return
    }
    if (p.state !== ALIVE) return
    if (p.inv > 0 && s.phase === 'play' && Math.floor(p.inv / 3) % 2) return
    let f = 'stand'
    if (p.kickT > 0) f = 'kick'
    else if (p.shootT > 0) f = p.ground ? 'shoot' : 'shootAir'
    else if (!p.ground) f = p.vy < 0 ? 'jump' : 'fall'
    else if (Math.abs(p.vx) > 0.25) f = Math.floor(p.walk / 7) % 2 ? 'runB' : 'runA'
    else f = Math.floor(this.tick / 32) % 2 ? 'breathe' : 'stand'
    const spr = kid.f[f][p.face > 0 ? 0 : 1]
    const dx = p.face > 0 ? -4 : -10
    ctx.drawImage(spr, Math.round(p.x + dx), Math.round(p.y - 5))
    // speed potion trail
    if (p.speed && Math.abs(p.vx) > 1 && this.tick % 3 === 0) this.parts.push({ x: p.x + p.w / 2, y: p.y + 10, vx: -p.vx * 0.2, vy: 0, life: 12, max: 12, color: '#7cc8ff', size: 1, grav: 0 })
    if (label) {
      const col = KID_COLORS[p.idx].css
      ctx.fillStyle = col
      const cx = Math.round(p.x + p.w / 2)
      const ty = Math.round(p.y - 9)
      ctx.fillRect(cx - 1, ty + 3, 3, 1)
      ctx.fillRect(cx, ty + 4, 1, 1)
      digits(ctx, `P${p.idx + 1}`, cx - 3, ty - 3, col, 1, '#1b1028')
    }
  }

  /** advance particles, pops, banner, shake and flash by one 60 Hz frame */
  fxStep(): void {
    const keep: Particle[] = []
    for (const p of this.parts) {
      p.x += p.vx
      p.y += p.vy
      p.vy += p.grav
      p.vx *= 0.97
      p.life--
      if (p.life > 0) keep.push(p)
    }
    this.parts = keep.length > 500 ? keep.slice(-500) : keep
    this.pops = this.pops.filter((p) => --p.life > 0)
    if (this.banner && --this.banner.life <= 0) this.banner = null
    this.shake = this.shake > 0.2 ? this.shake * 0.86 : 0
    if (this.flash > 0) this.flash--
    this.tick++
  }

  private drawParticles(): void {
    const ctx = this.ctx
    for (const p of this.parts) {
      ctx.fillStyle = p.color
      const sz = p.life < p.max * 0.3 && p.size > 1 ? p.size - 1 : p.size
      ctx.fillRect(Math.round(p.x), Math.round(p.y), sz, sz)
    }
  }

  private drawHud(s: GameState, net: { ping: number } | null): void {
    const ctx = this.ctx
    ctx.fillStyle = 'rgba(14,8,30,0.82)'
    ctx.fillRect(0, 0, W, 16)
    ctx.fillStyle = 'rgba(255,255,255,0.12)'
    ctx.fillRect(0, 15, W, 1)
    for (const p of s.players) {
      const left = p.idx === 0
      const kid = this.sheet.kids[p.idx]
      const col = KID_COLORS[p.idx].css
      const ix = left ? 3 : W - 3 - kid.icon.width
      if (!p.on) {
        if (!left && s.count < 2) {
          ctx.globalAlpha = 0.35
          ctx.drawImage(kid.icon, ix, 1)
          ctx.globalAlpha = 1
        }
        continue
      }
      ctx.drawImage(kid.icon, ix, 1)
      const score = String(p.score).padStart(6, '0')
      const sx = left ? ix + kid.icon.width + 3 : ix - 3 - textWidth(score)
      digits(ctx, score, sx, 3, col, 1, '#1b1028')
      // lives as little hearts
      for (let i = 0; i < Math.min(5, p.lives); i++) {
        const hx = left ? sx + i * 6 : sx + textWidth(score) - 5 - i * 6
        heart(ctx, hx, 10, '#ff5a7a')
      }
      // potions
      const pots = [p.power ? ITEM_RED : -1, p.speed ? ITEM_BLUE : -1, p.range ? ITEM_YELLOW : -1].filter((k) => k >= 0)
      pots.forEach((k, i) => {
        const spr = this.sheet.items[k]
        const px = left ? sx + textWidth(score) + 4 + i * 8 : sx - 12 - i * 8
        ctx.drawImage(spr, px, 2, 8, 9)
      })
      if (p.state === GHOST && p.lives === 0 && p.ghostT < 0) {
        ctx.fillStyle = 'rgba(14,8,30,0.6)'
        ctx.fillRect(left ? 0 : W / 2 + 40, 0, W / 2 - 40, 15)
      }
    }
    // floor number in the middle
    const fl = String(s.floor + 1).padStart(2, '0')
    digits(ctx, `F${fl}`, W / 2 - textWidth(`F${fl}`, 2) / 2, 3, '#ffd23f', 2, '#1b1028')
    // boss hp bar
    const boss = s.enemies.find((e) => e.boss)
    if (boss && boss.spawnT === 0) {
      const bw = 120
      const x = W / 2 - bw / 2
      ctx.fillStyle = '#1b1028'
      ctx.fillRect(x - 1, 18, bw + 2, 5)
      ctx.fillStyle = '#5a2340'
      ctx.fillRect(x, 19, bw, 3)
      ctx.fillStyle = boss.angry ? '#ff3b30' : '#ff86c3'
      ctx.fillRect(x, 19, Math.round((bw * boss.hp) / boss.maxHp), 3)
    }
    // hurry clock
    if (s.phase === 'play' && !FLOORS[s.floor].boss && s.floorT > 30 * 60) {
      const left = Math.max(0, 45 - Math.floor(s.floorT / 60))
      if (left > 0 && this.tick % 30 < 20) digits(ctx, String(left), W / 2 - textWidth(String(left)) / 2, 18, '#ff8a5c', 1, '#1b1028')
    }
    if (net && net.ping > 0) {
      const col = net.ping < 120 ? '#6ccb6a' : net.ping < 220 ? '#ffd23f' : '#ff5a5a'
      ctx.fillStyle = col
      ctx.fillRect(W / 2 + 18, 6, 2, 2)
      ctx.fillRect(W / 2 + 21, 4, 2, 4)
      ctx.fillRect(W / 2 + 24, 2, 2, 6)
    }
  }

  /** blow the low-res frame up onto the page canvas and add the sharp text on top */
  private present(s: GameState): void {
    const out = this.out
    const { x, y, scale } = this.view
    out.imageSmoothingEnabled = false
    out.fillStyle = '#0d0820'
    out.fillRect(0, 0, this.display.width, this.display.height)
    let sx = 0
    let sy = 0
    if (this.shake > 0.2) {
      sx = Math.round((Math.random() - 0.5) * this.shake * 2) * Math.max(1, Math.floor(scale))
      sy = Math.round((Math.random() - 0.5) * this.shake * 2) * Math.max(1, Math.floor(scale))
    }
    out.drawImage(this.lo, x + sx, y + sy, W * scale, H * scale)
    if (this.scanlines && scale >= 3) {
      out.fillStyle = 'rgba(0,0,0,0.13)'
      const step = Math.max(2, Math.round(scale))
      for (let yy = y + sy; yy < y + sy + H * scale; yy += step) out.fillRect(x + sx, yy + step - 1, W * scale, Math.max(1, Math.floor(step / 3)))
    }
    if (this.flash > 0) {
      out.globalAlpha = Math.min(0.6, this.flash / 12)
      out.fillStyle = this.flashColor
      out.fillRect(x, y, W * scale, H * scale)
      out.globalAlpha = 1
    }
    // score pops and chain labels
    for (const p of this.pops) {
      const age = (p.big ? 70 : 50) - p.life
      const rise = Math.min(14, age * 0.5)
      const px = x + sx + p.x * scale
      const py = y + sy + (p.y - rise) * scale
      const size = (p.big ? 9 : 7) * scale * (age < 6 ? 0.6 + age * 0.08 : 1)
      out.globalAlpha = p.life < 12 ? p.life / 12 : 1
      this.text(p.text, px, py, size, p.color)
      out.globalAlpha = 1
    }
    // banner
    if (this.banner) {
      const b = this.banner
      {
        const age = b.max - b.life
        const k = age < 10 ? age / 10 : b.life < 15 ? b.life / 15 : 1
        const cy = y + H * scale * 0.42
        out.globalAlpha = k
        out.fillStyle = 'rgba(14,8,30,0.62)'
        const bh = 44 * scale
        out.fillRect(x, cy - bh / 2, W * scale, bh * k)
        const pop = age < 10 ? 1.3 - age * 0.03 : 1
        this.text(b.text, x + (W * scale) / 2, cy - 6 * scale, 20 * scale * pop, b.color)
        if (b.sub) this.text(b.sub, x + (W * scale) / 2, cy + 11 * scale, 8 * scale, '#e8dcff')
        out.globalAlpha = 1
      }
    }
    // small labels for ghosts
    for (const p of s.players) {
      if (!p.on || p.state !== GHOST) continue
      const mate = s.players[1 - p.idx]
      if (!mate.on || mate.state !== ALIVE) continue
      if (this.tick % 60 < 40) this.text('碰我复活', x + (p.x + p.w / 2) * scale, y + (p.y + 26) * scale, 6 * scale, '#ffffff')
    }
  }

  private text(t: string, x: number, y: number, size: number, color: string): void {
    const out = this.out
    out.font = `900 ${Math.round(size)}px "PingFang SC","HarmonyOS Sans SC","Microsoft YaHei",system-ui,sans-serif`
    out.textAlign = 'center'
    out.textBaseline = 'middle'
    out.lineJoin = 'round'
    out.lineWidth = Math.max(2, size / 5)
    out.strokeStyle = '#1b1028'
    out.strokeText(t, x, y)
    out.fillStyle = color
    out.fillText(t, x, y)
  }
}

function heart(ctx: CanvasRenderingContext2D, x: number, y: number, col: string): void {
  ctx.fillStyle = col
  ctx.fillRect(x, y, 2, 1)
  ctx.fillRect(x + 3, y, 2, 1)
  ctx.fillRect(x, y + 1, 5, 2)
  ctx.fillRect(x + 1, y + 3, 3, 1)
  ctx.fillRect(x + 2, y + 4, 1, 1)
}

function bossLine(kind: string): string {
  if (kind === 'bigclock') return '它一跳，地板都在抖。把小闹钟裹起来砸它！'
  if (kind === 'bookpile') return '纸飞机能用糖丝打掉，作业本踢回去！'
  return '暑假最后一关。把它撕下来！'
}
