// Draws a World onto the 400×224 canvas: parallax backdrop, tiles, enemies, heroes, effects and
// the HUD. Works the same for the host and for an online guest, whose World is filled from
// snapshots. Effects (debris, coin pops, score numbers, bursts) are spawned from World events.

import { backgrounds, hash, heroFrame, sprite, tileArt, THEME, type Canvas } from './art'
import { bodyH } from './physics'
import type { Ev, Player, World } from './world'
import { ST } from './world'
import { ROWS, T, TILE, VIEW_H, VIEW_W } from './types'
import { isSolid } from './physics'

interface Fx {
  k: 'debris' | 'coin' | 'pts' | 'puff' | 'boom' | 'spark' | 'firework' | 'dust'
  x: number
  y: number
  vx: number
  vy: number
  t: number
  life: number
  text?: string
  col?: string
}

// 3×5 pixel digits and a few letters for the HUD
const GLYPHS: Record<string, string> = {
  '0': '111101101101111',
  '1': '010110010010111',
  '2': '111001111100111',
  '3': '111001111001111',
  '4': '101101111001001',
  '5': '111100111001111',
  '6': '111100111101111',
  '7': '111001010010010',
  '8': '111101111101111',
  '9': '111101111001111',
  x: '000101010101000',
  '-': '000000111000000',
  U: '101101101101111',
  P: '111101111100100',
  ':': '000010000010000',
  ' ': '000000000000000'
}

export class Renderer {
  private g: CanvasRenderingContext2D
  private fx: Fx[] = []
  private bumps = new Map<number, number>()
  private shake = 0
  private frame = 0
  private fireworks: Fx[] = []

  constructor(readonly canvas: HTMLCanvasElement) {
    canvas.width = VIEW_W
    canvas.height = VIEW_H
    this.g = canvas.getContext('2d') as CanvasRenderingContext2D
    this.g.imageSmoothingEnabled = false
  }

  /** feed World events (local or from the network) */
  onEvents(evs: Ev[], world: World): void {
    for (const e of evs) {
      switch (e.e) {
        case 'bump':
          this.bumps.set(e.ty * world.area.width + e.tx, this.frame)
          break
        case 'brk':
          for (let i = 0; i < 4; i++)
            this.fx.push({ k: 'debris', x: e.tx * TILE + 4 + (i % 2) * 8, y: e.ty * TILE + 4 + Math.floor(i / 2) * 8, vx: (i % 2 ? 1 : -1) * (1 + Math.random()), vy: -4 - (i < 2 ? 1.5 : 0), t: 0, life: 70 })
          break
        case 'coin':
          this.fx.push({ k: 'coin', x: e.x, y: e.y, vx: 0, vy: -5, t: 0, life: 32 })
          break
        case 'pts':
          this.fx.push({ k: 'pts', x: e.x, y: e.y, vx: 0, vy: -0.6, t: 0, life: 50, text: e.v })
          break
        case 'puff':
          this.fx.push({ k: 'puff', x: e.x, y: e.y, vx: 0, vy: 0, t: 0, life: 16 })
          break
        case 'boom':
          this.fx.push({ k: 'boom', x: e.x, y: e.y, vx: 0, vy: 0, t: 0, life: 24 })
          for (let i = 0; i < 14; i++) {
            const a = (i / 14) * Math.PI * 2
            this.fx.push({ k: 'spark', x: e.x, y: e.y, vx: Math.cos(a) * 2.5, vy: Math.sin(a) * 2.5 - 1, t: 0, life: 28, col: i % 2 ? '#fff07a' : '#ff9d2e' })
          }
          break
        case 'shake':
          this.shake = Math.max(this.shake, e.n)
          break
        case 'edit':
          break
      }
    }
  }

  /** a puff of dust at the feet (skids, landings) */
  dust(x: number, y: number): void {
    this.fx.push({ k: 'dust', x, y: y - 2, vx: (Math.random() - 0.5) * 0.6, vy: -0.3, t: 0, life: 14 })
  }

  draw(w: World, local: number[]): void {
    this.frame++
    const g = this.g
    const area = w.area
    const th = area.theme
    let sx = 0
    let sy = 0
    if (this.shake > 0) {
      sx = Math.round((Math.random() - 0.5) * Math.min(6, this.shake))
      sy = Math.round((Math.random() - 0.5) * Math.min(6, this.shake))
      this.shake--
    }
    const cam = Math.round(w.cam)

    // backdrop
    for (const L of backgrounds(th)) {
      const off = Math.round(cam * L.k) % L.c.width
      for (let x = -off; x < VIEW_W; x += L.c.width) g.drawImage(L.c, x, L.y)
    }
    if (th === 'tower' || th === 'canal') this.drawFireworks(th === 'tower' ? 0.035 : 0.012)
    if (th === 'cave') this.drawFireflies(cam)

    g.save()
    g.translate(-cam + sx, sy)
    this.drawDecoBehind(w, cam)

    // things that live behind tiles: 竹鼠 in tubes, items climbing out of boxes
    for (const e of w.ents) if (e.k === 'rat' || e.st === ST.EMERGE) this.drawEnt(w, e)

    this.drawTiles(w, cam)
    this.drawGoal(w)
    for (const e of w.ents) if (e.k !== 'rat' && e.st !== ST.EMERGE) this.drawEnt(w, e)
    // movers
    for (const m of area.def.movers) {
      const pos = moverPosCached(m, w.frame)
      this.drawMover(th, pos.x, pos.y, m.w)
    }
    w.players.forEach((p, i) => this.drawPlayer(p, i, local.includes(i)))
    this.drawFx()
    g.restore()

    this.drawHud(w)
  }

  private drawFireworks(rate: number): void {
    const g = this.g
    if (Math.random() < rate) {
      const x = 30 + Math.random() * (VIEW_W - 60)
      const y = 30 + Math.random() * 70
      const cols = ['#ff5a5a', '#ffd23f', '#7fd4ff', '#bfa7ff', '#7ed957']
      const col = cols[Math.floor(Math.random() * cols.length)]
      for (let i = 0; i < 18; i++) {
        const a = (i / 18) * Math.PI * 2
        this.fireworks.push({ k: 'firework', x, y, vx: Math.cos(a) * 0.9, vy: Math.sin(a) * 0.9, t: 0, life: 50, col })
      }
    }
    for (const f of this.fireworks) {
      f.t++
      f.x += f.vx
      f.y += f.vy
      f.vy += 0.015
      f.vx *= 0.98
      if (f.t > f.life * 0.6 && f.t % 4 < 2) continue
      g.fillStyle = f.col || '#fff'
      g.fillRect(Math.round(f.x), Math.round(f.y), 1, 1)
    }
    this.fireworks = this.fireworks.filter((f) => f.t < f.life)
  }

  private drawFireflies(cam: number): void {
    const g = this.g
    for (let i = 0; i < 18; i++) {
      const x = ((hash(i, 1) * 600 - cam * 0.4 + Math.sin(this.frame * 0.01 + i) * 20) % 600 + 600) % 600 - 100
      const y = 50 + hash(i, 2) * 130 + Math.sin(this.frame * 0.03 + i * 2) * 6
      if (Math.sin(this.frame * 0.05 + i * 3) < -0.3) continue
      g.fillStyle = '#d8ff7a'
      g.fillRect(Math.round(x), Math.round(y), 1, 1)
      g.fillStyle = 'rgba(216,255,122,0.25)'
      g.fillRect(Math.round(x) - 1, Math.round(y) - 1, 3, 3)
    }
  }

  private drawDecoBehind(w: World, cam: number): void {
    const g = this.g
    const area = w.area
    const th = area.theme
    const tx0 = Math.max(0, Math.floor(cam / TILE) - 1)
    const tx1 = Math.min(area.width - 1, tx0 + Math.ceil(VIEW_W / TILE) + 2)
    for (let tx = tx0; tx <= tx1; tx++) {
      for (let ty = 1; ty < ROWS; ty++) {
        const c = area.get(tx, ty)
        if (c !== T.GROUND) continue
        const above = area.get(tx, ty - 1)
        if (isSolid(above) || above === T.LAVA) continue
        const x = tx * TILE
        const y = ty * TILE
        const h = hash(tx, ty, 11)
        if (th === 'treetop' && (ty === ROWS - 1 || area.get(tx, ty + 1) === T.EMPTY) && h > 0.55) {
          // bamboo trunk down from the canopy
          g.fillStyle = '#3f8a3a'
          g.fillRect(x + 6, y + 8, 5, VIEW_H - y)
          g.fillStyle = '#7ed957'
          g.fillRect(x + 6, y + 8, 2, VIEW_H - y)
          g.fillStyle = '#21632a'
          for (let yy = y + 26; yy < VIEW_H; yy += 22) g.fillRect(x + 5, yy, 7, 1)
        }
        if (th === 'bamboo' || th === 'treetop') {
          if (h > 0.8) {
            // grass tuft and a flower
            g.fillStyle = '#2f7d2a'
            g.fillRect(x + 3, y - 3, 1, 3)
            g.fillRect(x + 5, y - 4, 1, 4)
            g.fillRect(x + 7, y - 2, 1, 2)
            if (h > 0.9) {
              g.fillStyle = h > 0.95 ? '#ff8ab0' : '#fff07a'
              g.fillRect(x + 4, y - 6, 3, 2)
            }
          } else if (h < 0.08 && ty >= 11) {
            // little bush
            g.fillStyle = '#2f7d2a'
            g.fillRect(x + 1, y - 7, 14, 7)
            g.fillStyle = '#5fbf3a'
            g.fillRect(x + 2, y - 8, 11, 6)
            g.fillStyle = '#9be15a'
            g.fillRect(x + 4, y - 8, 4, 2)
          }
        } else if (th === 'cave' && h > 0.85) {
          g.fillStyle = '#9ef0ff'
          g.fillRect(x + 6, y - 5, 2, 5)
          g.fillRect(x + 9, y - 3, 2, 3)
          g.fillStyle = '#e8ffff'
          g.fillRect(x + 6, y - 5, 1, 2)
        } else if ((th === 'town' || th === 'canal') && h > 0.9 && ty >= 11) {
          // potted plant
          g.fillStyle = '#8a4a2a'
          g.fillRect(x + 5, y - 5, 6, 5)
          g.fillStyle = '#5fbf3a'
          g.fillRect(x + 4, y - 10, 8, 5)
        }
      }
    }
    // checkpoint lantern
    const cp = area.def.checkpoint
    if (cp !== undefined && w.areaIdx === 0) {
      const x = cp * TILE + 4
      const y = this.surface(w, cp)
      g.fillStyle = '#5a3a2a'
      g.fillRect(x + 3, y - 30, 2, 30)
      g.fillRect(x + 3, y - 30, 8, 2)
      const lit = w.checkpointHit
      g.fillStyle = lit ? '#e8282a' : '#6a5a5a'
      g.fillRect(x + 6, y - 27, 7, 9)
      g.fillStyle = lit ? '#ffd23f' : '#8a7a7a'
      g.fillRect(x + 8, y - 25, 3, 5)
      g.fillRect(x + 8, y - 18, 3, 2)
      if (lit && this.frame % 30 < 20) {
        g.fillStyle = 'rgba(255,210,63,0.25)'
        g.fillRect(x + 3, y - 30, 13, 15)
      }
    }
  }

  private surface(w: World, tx: number): number {
    for (let ty = 1; ty < ROWS; ty++) if (isSolid(w.area.get(tx, ty)) && !isSolid(w.area.get(tx, ty - 1))) return ty * TILE
    return 12 * TILE
  }

  private drawTiles(w: World, cam: number): void {
    const g = this.g
    const area = w.area
    const th = area.theme
    const tx0 = Math.max(0, Math.floor(cam / TILE))
    const tx1 = Math.min(area.width - 1, tx0 + Math.ceil(VIEW_W / TILE) + 1)
    for (let ty = 0; ty < ROWS; ty++)
      for (let tx = tx0; tx <= tx1; tx++) {
        const c = area.get(tx, ty)
        if (c === T.EMPTY || c === T.BOX_HIDDEN || c === T.BOX_LIFE || c === T.POLE) continue
        const above = ty > 0 ? area.get(tx, ty - 1) : T.EMPTY
        const top = c === T.LAVA ? above !== T.LAVA : !isSolid(above) && above !== c
        const art = tileArt(th, c, { top, x: tx, y: ty, frame: this.frame })
        if (!art) continue
        let dy = 0
        const bi = ty * area.width + tx
        const b = this.bumps.get(bi)
        if (b !== undefined) {
          const t = this.frame - b
          if (t > 10) this.bumps.delete(bi)
          else dy = -Math.round(Math.sin((t / 10) * Math.PI) * 5)
        }
        g.drawImage(art, tx * TILE, ty * TILE + dy)
      }
  }

  private drawGoal(w: World): void {
    const goal = w.area.def.goal
    if (goal === undefined) return
    const g = this.g
    const x = goal * TILE
    // bamboo pole with a red banner
    g.fillStyle = '#21632a'
    g.fillRect(x + 6, 2 * TILE, 4, 9 * TILE)
    g.fillStyle = '#7ed957'
    g.fillRect(x + 7, 2 * TILE, 2, 9 * TILE)
    g.fillStyle = '#21632a'
    for (let y = 3 * TILE; y < 11 * TILE; y += 20) g.fillRect(x + 6, y, 4, 1)
    g.fillStyle = '#ffd23f'
    g.fillRect(x + 5, 2 * TILE - 5, 6, 6)
    const sway = Math.round(Math.sin(this.frame * 0.1) * 1)
    g.fillStyle = '#2b1b2e'
    g.fillRect(x - 15, 2 * TILE + 3, 21, 14 + sway)
    g.fillStyle = '#e8322e'
    g.fillRect(x - 14, 2 * TILE + 4, 20, 12 + sway)
    g.fillStyle = '#ffd23f'
    // a little gold 福-like knot pattern (abstract, not a character)
    g.fillRect(x - 8, 2 * TILE + 7, 7, 1)
    g.fillRect(x - 8, 2 * TILE + 12, 7, 1)
    g.fillRect(x - 8, 2 * TILE + 7, 1, 6)
    g.fillRect(x - 2, 2 * TILE + 7, 1, 6)
    g.fillRect(x - 5, 2 * TILE + 9, 1, 2)

    // 牌坊 gate the heroes walk through
    const gx = (goal + 5) * TILE
    const gy = this.surface(w, goal + 6)
    g.fillStyle = '#2b1b2e'
    g.fillRect(gx - 1, gy - 70, 4 * TILE + 2, 70)
    g.fillStyle = '#b02a30'
    g.fillRect(gx, gy - 60, 8, 60)
    g.fillRect(gx + 4 * TILE - 8, gy - 60, 8, 60)
    g.fillStyle = '#e8443a'
    g.fillRect(gx + 1, gy - 60, 3, 60)
    g.fillRect(gx + 4 * TILE - 7, gy - 60, 3, 60)
    g.fillStyle = '#3e4660'
    g.fillRect(gx - 8, gy - 74, 4 * TILE + 16, 8)
    g.fillRect(gx - 12, gy - 77, 6, 4)
    g.fillRect(gx + 4 * TILE + 6, gy - 77, 6, 4)
    g.fillStyle = '#6a7290'
    for (let i = gx - 8; i < gx + 4 * TILE + 8; i += 4) g.fillRect(i, gy - 74, 3, 6)
    g.fillStyle = '#e8b84a'
    g.fillRect(gx + 8, gy - 64, 4 * TILE - 16, 10)
    g.fillStyle = '#b02a30'
    g.fillRect(gx + 18, gy - 62, 4 * TILE - 36, 6)
    g.fillStyle = '#2b1b2e'
    g.fillRect(gx + 8, gy - 48, 4 * TILE - 16, 3)
    // the doorway: dark so the heroes vanish into it
    g.fillStyle = 'rgba(20,10,20,0.0)'
    g.fillRect(gx + 8, gy - 45, 4 * TILE - 16, 45)
  }

  private drawMover(th: string, x: number, y: number, wTiles: number): void {
    const g = this.g
    const w = wTiles * TILE
    const rx = Math.round(x)
    const ry = Math.round(y)
    if (th === 'canal') {
      // a little wooden boat
      g.fillStyle = '#2b1b2e'
      g.fillRect(rx - 2, ry, w + 4, 9)
      g.fillStyle = '#9a5a32'
      g.fillRect(rx, ry + 1, w, 6)
      g.fillStyle = '#c98a52'
      g.fillRect(rx, ry + 1, w, 2)
      g.fillStyle = '#e8322e'
      g.fillRect(rx + w / 2 - 1, ry - 10, 2, 10)
      g.fillStyle = '#ffd23f'
      g.fillRect(rx + w / 2 - 4, ry - 16, 8, 6)
      return
    }
    if (th === 'fort' || th === 'palace' || th === 'tower') {
      g.fillStyle = '#2b1b2e'
      g.fillRect(rx - 1, ry - 1, w + 2, 8)
      g.fillStyle = '#c9a06a'
      g.fillRect(rx, ry, w, 6)
      g.fillStyle = '#f0d278'
      g.fillRect(rx, ry, w, 1)
      g.fillStyle = '#5a4224'
      for (let i = 4; i < w; i += 8) g.fillRect(rx + i, ry + 1, 1, 5)
      return
    }
    // bamboo raft
    g.fillStyle = '#2b1b2e'
    g.fillRect(rx - 1, ry - 1, w + 2, 9)
    for (let i = 0; i < w; i += 4) {
      g.fillStyle = '#5fbf3a'
      g.fillRect(rx + i, ry, 4, 7)
      g.fillStyle = '#a6ec6a'
      g.fillRect(rx + i, ry, 1, 7)
      g.fillStyle = '#2f7d2a'
      g.fillRect(rx + i + 3, ry, 1, 7)
    }
    g.fillStyle = '#c9a04a'
    g.fillRect(rx, ry + 3, w, 1)
  }

  private drawImg(c: Canvas, x: number, y: number, flipY = false): void {
    const g = this.g
    if (!flipY) {
      g.drawImage(c, Math.round(x - c.width / 2), Math.round(y - c.height))
      return
    }
    g.save()
    g.translate(Math.round(x), Math.round(y - c.height / 2))
    g.scale(1, -1)
    g.drawImage(c, Math.round(-c.width / 2), Math.round(-c.height / 2))
    g.restore()
  }

  private drawEnt(w: World, e: World['ents'][number]): void {
    const g = this.g
    const f = Math.floor(e.age / 10) % 2
    const flipX = e.dir > 0
    const flipY = e.st === ST.FLIP
    switch (e.k) {
      case 'mud':
        this.drawImg(sprite(e.st === ST.SQUASH ? 'mud_flat' : f ? 'mud2' : 'mud1', flipX), e.x, e.y + 1, flipY)
        break
      case 'hog':
        if (e.st === ST.BALL || e.st === ST.ROLL) {
          const shaking = e.st === ST.BALL && e.t > 330 ? (e.t % 4 < 2 ? 1 : -1) : 0
          this.drawImg(sprite('hog_ball', e.st === ST.ROLL && Math.floor(e.age / 3) % 2 === 0), e.x + shaking, e.y, flipY)
        } else this.drawImg(sprite(f ? 'hog2' : 'hog1', flipX), e.x, e.y, flipY)
        break
      case 'kite':
        this.drawImg(sprite(f ? 'kite2' : 'kite1', false), e.x, e.y + 2, flipY)
        break
      case 'rat': {
        // only the part above the tube mouth shows
        g.save()
        g.beginPath()
        g.rect(e.ax - 16, e.ay - 40, 32, 40)
        g.clip()
        this.drawImg(sprite('rat', Math.floor(e.age / 20) % 2 === 0), e.x, e.y)
        g.restore()
        break
      }
      case 'fish':
        this.drawImg(sprite(f ? 'fish2' : 'fish1', e.vx > 0), e.x, e.y, flipY || e.vy > 1)
        break
      case 'cracker':
        if (e.st === ST.LIT) {
          const flash = e.t > 60 && e.t % 6 < 3
          this.drawImg(sprite('cracker_lit', flipX), e.x + (flash ? 1 : 0), e.y)
          if (e.t % 4 < 2) {
            g.fillStyle = '#fff07a'
            g.fillRect(Math.round(e.x - 2 + Math.random() * 3), Math.round(e.y - 17), 1, 1)
          }
        } else this.drawImg(sprite(f ? 'cracker2' : 'cracker1', flipX), e.x, e.y, flipY)
        break
      case 'spark':
        if (e.y < VIEW_H + 4) this.drawImg(sprite(Math.floor(e.age / 5) % 2 ? 'spark2' : 'spark1'), e.x, e.y, e.vy > 0)
        break
      case 'firebar': {
        const ang = w.firebarAngle(e)
        const em = sprite('ember')
        for (let i = 0; i < e.len; i++) g.drawImage(em, Math.round(e.ax + Math.cos(ang) * i * 8 - 3), Math.round(e.ay + Math.sin(ang) * i * 8 - 3))
        break
      }
      case 'fire':
        this.drawImg(sprite(Math.floor(e.age / 4) % 2 ? 'fire2' : 'fire1', e.vx < 0), e.x, e.y)
        break
      case 'mudball':
        this.drawImg(sprite('mudball'), e.x, e.y + 4)
        break
      case 'flame':
        this.drawImg(sprite('flame', e.vx < 0), e.x, e.y)
        break
      case 'bun':
      case 'chili':
      case 'haw':
        this.drawImg(sprite(e.k), e.x, e.y + 1)
        break
      case 'bell': {
        const bob = Math.round(Math.sin(e.age * 0.3) * 1)
        this.drawImg(sprite('bell'), e.x + bob, e.y + 1)
        break
      }
      case 'lamp': {
        this.drawImg(sprite('lamp'), e.x, e.y)
        if (this.frame % 20 < 12) {
          g.fillStyle = 'rgba(255,210,63,0.2)'
          g.fillRect(Math.round(e.x - 14), Math.round(e.y - 26), 28, 28)
        }
        break
      }
      case 'mudking': {
        if (e.st === ST.HURT && e.t % 6 < 3) break
        const fr = e.st === ST.DEFEAT || e.st === ST.HURT || !e.ground ? 'mudking2' : 'mudking1'
        this.drawImg(sprite(fr, e.dir > 0), e.x, e.y + 1)
        break
      }
      case 'nian': {
        if (e.st === ST.HURT && e.t % 6 < 3) break
        const shake = e.st === ST.ROAR ? (e.t % 4 < 2 ? 1 : -1) : 0
        const fr = e.st === ST.CHARGE ? (Math.floor(e.age / 4) % 2 ? 'nian2' : 'nian1') : Math.floor(e.age / 12) % 2 ? 'nian2' : 'nian1'
        this.drawImg(sprite(fr, e.dir > 0), e.x + shake, e.y + 1)
        if (e.st === ST.STUN) {
          for (let i = 0; i < 3; i++) {
            const a = this.frame * 0.12 + (i * Math.PI * 2) / 3
            g.fillStyle = '#ffd23f'
            g.fillRect(Math.round(e.x + Math.cos(a) * 14), Math.round(e.y - 36 + Math.sin(a) * 4), 2, 2)
          }
        }
        break
      }
    }
  }

  private drawPlayer(p: Player, i: number, isLocal: boolean): void {
    if (p.st === 'done' || p.st === 'out') return
    const g = this.g
    const ps = p.ps
    const hero = i === 0 ? 'tiger' : 'rabbit'
    if (p.inv > 0 && p.st === 'play' && Math.floor(p.inv / 3) % 2 === 0) return
    let big = ps.power > 0
    if (p.morph > 0 && Math.floor(p.morph / 4) % 2 === 0) big = !big
    let body = big ? 'bb_stand' : 'sb_stand'
    const pre = big ? 'bb_' : 'sb_'
    switch (p.anim) {
      case 'walk': {
        const cyc = Math.floor(p.animT) % 4
        body = pre + ['walk1', 'walk2', 'walk3', 'walk2'][cyc]
        break
      }
      case 'jump':
      case 'fall':
        body = pre + 'jump'
        break
      case 'skid':
        body = pre + 'skid'
        break
      case 'crouch':
        body = big ? 'bb_crouch' : 'sb_stand'
        break
      case 'die':
        body = 'sb_die'
        break
      case 'pole':
        body = pre + 'pole'
        break
    }
    if (p.st === 'dead') body = 'sb_die'
    const flash = p.bell > 0 && Math.floor(this.frame / 3) % 3 === 0
    const chili = ps.power === 2 || (p.bell > 0 && Math.floor(this.frame / 3) % 3 === 1)
    const c = heroFrame(hero, body, chili, ps.facing < 0, flash)
    let y = ps.y + 1
    const x = ps.x
    if (p.st === 'tube') {
      // sinking into / rising out of a tube: only the part above the mouth shows
      g.save()
      g.beginPath()
      g.rect(x - 20, p.tubeY - 60, 40, 60)
      g.clip()
      g.drawImage(c, Math.round(x - c.width / 2), Math.round(y - c.height))
      g.restore()
      return
    }
    if (p.st === 'bubble') {
      const r = 14
      g.fillStyle = 'rgba(190,230,255,0.35)'
      g.beginPath()
      g.arc(Math.round(x), Math.round(y - 9), r, 0, Math.PI * 2)
      g.fill()
      g.strokeStyle = 'rgba(255,255,255,0.9)'
      g.lineWidth = 1
      g.stroke()
      g.fillStyle = '#fff'
      g.fillRect(Math.round(x - 7), Math.round(y - 17), 3, 2)
      y -= 2
    }
    if (p.squash > 0) {
      g.drawImage(c, Math.round(x - c.width / 2), Math.round(y - c.height * 0.8), c.width, Math.round(c.height * 0.8))
    } else g.drawImage(c, Math.round(x - c.width / 2), Math.round(y - c.height))
    // name tag over the remote partner / both in local co-op
    if (p.st === 'play' && !isLocal) {
      const tagY = Math.round(ps.y - bodyH(ps) - (hero === 'rabbit' ? 14 : 8))
      g.fillStyle = i === 0 ? '#ff9a2e' : '#3aa0e8'
      g.fillRect(Math.round(x) - 1, tagY - 3, 3, 2)
      g.fillRect(Math.round(x), tagY - 1, 1, 1)
    }
    if (p.anim === 'skid' && this.frame % 4 === 0) this.dust(x - ps.facing * 4, ps.y)
    if (p.bell > 0 && this.frame % 4 === 0) this.fx.push({ k: 'spark', x: x + (Math.random() - 0.5) * 14, y: y - Math.random() * bodyH(ps), vx: 0, vy: -0.4, t: 0, life: 14, col: '#fff07a' })
  }

  private drawFx(): void {
    const g = this.g
    for (const f of this.fx) {
      f.t++
      switch (f.k) {
        case 'debris': {
          f.vy += 0.3
          f.x += f.vx
          f.y += f.vy
          const c = THEME.bamboo.brick
          g.fillStyle = '#2b1b2e'
          g.fillRect(Math.round(f.x) - 1, Math.round(f.y) - 1, 6, 6)
          g.fillStyle = c[f.t % 8 < 4 ? 0 : 1]
          g.fillRect(Math.round(f.x), Math.round(f.y), 4, 4)
          break
        }
        case 'coin': {
          f.vy += 0.35
          f.y += f.vy
          const fr = ['coin1', 'coin2', 'coin3', 'coin2'][Math.floor(f.t / 3) % 4]
          g.drawImage(sprite(fr), Math.round(f.x - 6), Math.round(f.y - 10))
          if (f.t === f.life - 1) this.fx.push({ k: 'pts', x: f.x, y: f.y - 6, vx: 0, vy: -0.6, t: 0, life: 30, text: '200' })
          break
        }
        case 'pts':
          f.y += f.vy
          this.text(f.text || '', Math.round(f.x) - (f.text || '').length * 2, Math.round(f.y), '#ffffff')
          break
        case 'puff': {
          const r = 2 + f.t * 0.5
          g.fillStyle = f.t < 8 ? '#ffffff' : '#d8d8d8'
          for (let i = 0; i < 6; i++) {
            const a = (i / 6) * Math.PI * 2
            g.fillRect(Math.round(f.x + Math.cos(a) * r), Math.round(f.y + Math.sin(a) * r), 2, 2)
          }
          break
        }
        case 'boom': {
          const r = 4 + f.t * 1.6
          g.fillStyle = f.t < 6 ? '#ffffff' : f.t < 14 ? '#fff07a' : '#ff9d2e'
          for (let i = 0; i < 20; i++) {
            const a = (i / 20) * Math.PI * 2
            g.fillRect(Math.round(f.x + Math.cos(a) * r), Math.round(f.y + Math.sin(a) * r), 3, 3)
          }
          if (f.t < 4) {
            g.fillStyle = 'rgba(255,240,180,0.6)'
            g.fillRect(Math.round(f.x - 20), Math.round(f.y - 20), 40, 40)
          }
          break
        }
        case 'spark':
        case 'dust':
          f.x += f.vx
          f.y += f.vy
          if (f.k === 'spark') f.vy += 0.08
          g.fillStyle = f.k === 'dust' ? (f.t < 7 ? '#ffffff' : '#d8d0c0') : f.col || '#fff'
          g.fillRect(Math.round(f.x), Math.round(f.y), f.k === 'dust' && f.t < 6 ? 2 : 1, f.k === 'dust' && f.t < 6 ? 2 : 1)
          break
      }
    }
    this.fx = this.fx.filter((f) => f.t < f.life)
  }

  private text(s: string, x: number, y: number, col: string, shadow = true): number {
    const g = this.g
    let cx = x
    for (const ch of s) {
      const gl = GLYPHS[ch] || GLYPHS[' ']
      for (let i = 0; i < 15; i++) {
        if (gl[i] !== '1') continue
        const px = cx + (i % 3)
        const py = y + Math.floor(i / 3)
        if (shadow) {
          g.fillStyle = '#2b1b2e'
          g.fillRect(px + 1, py + 1, 1, 1)
        }
        g.fillStyle = col
        g.fillRect(px, py, 1, 1)
      }
      cx += 4
    }
    return cx
  }

  private drawHud(w: World): void {
    const g = this.g
    // soft dark strip
    g.fillStyle = 'rgba(20,12,24,0.35)'
    g.fillRect(0, 0, VIEW_W, 13)
    const icon = (hero: 'tiger' | 'rabbit', x: number) => {
      const c = heroFrame(hero, 'sb_stand', false, false)
      g.drawImage(c, 0, hero === 'rabbit' ? 6 : 0, 16, 10, x, 1, 16, 10)
    }
    const coin = sprite('coin1')
    w.players.forEach((p, i) => {
      const x = i === 0 ? 4 : 232
      icon(i === 0 ? 'tiger' : 'rabbit', x)
      this.text(String(Math.floor(p.score)).padStart(6, '0'), x + 19, 4, '#ffffff')
      g.drawImage(coin, 0, 0, 12, 10, x + 46, 2, 9, 8)
      this.text('x' + String(p.coins % 100).padStart(2, '0'), x + 56, 4, '#ffd23f')
    })
    // lives
    const lx = w.players.length > 1 ? 150 : 130
    g.fillStyle = '#e8322e'
    g.fillRect(lx, 4, 2, 2)
    g.fillRect(lx + 3, 4, 2, 2)
    g.fillRect(lx, 5, 5, 2)
    g.fillRect(lx + 1, 7, 3, 1)
    g.fillRect(lx + 2, 8, 1, 1)
    this.text('x' + String(w.lives), lx + 7, 4, '#ffffff')
    this.text(w.level.id, lx + 26, 4, '#bfe8ff')
    // timer
    const tx = VIEW_W - 26
    g.fillStyle = '#ffffff'
    g.fillRect(tx - 9, 3, 7, 7)
    g.fillStyle = '#2b1b2e'
    g.fillRect(tx - 8, 4, 5, 5)
    g.fillStyle = '#ffffff'
    g.fillRect(tx - 6, 5, 1, 2)
    g.fillRect(tx - 6, 6, 2, 1)
    const low = w.time <= 60 && w.phase === 'play'
    this.text(String(Math.max(0, w.time)).padStart(3, '0'), tx, 4, low && this.frame % 30 < 15 ? '#ff5a5a' : '#ffffff')
  }
}

// moverPos is cheap, but keep the import local to avoid a cycle with world.ts at module init
import { moverPos } from './world'
function moverPosCached(m: Parameters<typeof moverPos>[0], frame: number) {
  return moverPos(m, frame)
}
