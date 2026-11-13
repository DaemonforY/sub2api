// 蜂巢星球: fly a bee round a small sleeping planet and wake its six islands. This file is the
// game loop: flight, camera, what blooms and when. The page (PlanetView.vue) draws the HUD.
import * as THREE from 'three'
import { PlanetAudio } from './audio'
import { makeBee, Sparks, type Bee } from './fx'
import { Life } from './life'
import { clamp, damp, lerp, smoothstep } from './noise'
import { GATES, medalFor, Races } from './race'
import { PATCHES_PER_REGION, R, REGIONS, World, type Patch } from './world'

export type Phase = 'title' | 'fly' | 'finale'

export interface HudState {
  phase: Phase
  /** island under the bee, or '' over open sea */
  region: string
  regionIndex: number
  /** the six flower fields of that island */
  fields: boolean[]
  awake: boolean[]
  /** pollen the bee is carrying; it turns into honey at the hive */
  pollen: number
  honey: number
  /** honey level 0..4, and what the next one takes and gives */
  level: number
  levelFrom: number
  levelNeed: number
  levelName: string
  /** direction of the next flower field or race hoop: radians, 0 ahead, positive to the right */
  arrow: number
  arrowDist: number
  arrowLabel: string
  /** the way home, shown while carrying pollen */
  hiveArrow: number
  hiveDist: number
  /** the lap being flown, if any */
  race: { seconds: number; passed: number; total: number; gold: number; silver: number; best: number } | null
  /** best medal per island: 0 none, 1 bronze, 2 silver, 3 gold */
  medals: number[]
}

export interface PlanetEvents {
  hud(s: HudState): void
  toast(title: string, sub: string, big: boolean): void
  finale(stats: { seconds: number; pollen: number }): void
}

const SAVE_KEY = 'hg_planet_v1'

interface Save {
  fields: boolean[]
  pollen: number
  honey: number
  gathered: number
  seconds: number
  done: boolean
  best: number[]
}

/** Honey levels: how much honey each takes in all, and what it unlocks. */
const LEVELS = [
  { need: 20, name: '花粉磁铁', desc: '附近的花粉会自己飞过来' },
  { need: 50, name: '彩虹尾迹', desc: '身后拖出一道彩虹' },
  { need: 90, name: '疾风', desc: '加速飞得更快' },
  { need: 150, name: '金蜂', desc: '全身发出金光' }
]
const MEDALS = ['完成', '铜牌', '银牌', '金牌']
const PRIZES = [5, 10, 20, 30]

const CRUISE = 22
const BOOST = 39
const GALE = 47
const SLOW = 13
const GOLD = new THREE.Color('#ffd76a')
const WHITE = new THREE.Color('#fff7d6')

export class PlanetGame {
  private readonly renderer: THREE.WebGLRenderer
  private readonly scene = new THREE.Scene()
  private readonly camera = new THREE.PerspectiveCamera(62, 1, 0.3, 4000)
  private readonly world = new World()
  private readonly life = new Life(this.world)
  private readonly bee: Bee = makeBee()
  private readonly sparks = new Sparks()
  private readonly audio = new PlanetAudio()
  private readonly shadow: THREE.Mesh
  private readonly halo: THREE.Mesh
  private readonly races: Races

  // the bee: where it is on the sphere, which way it faces, how high
  private readonly n = new THREE.Vector3()
  private readonly fwd = new THREE.Vector3()
  private alt = 9
  /** height above the ground, set by the player; the ground itself is followed automatically */
  private hover = 7
  private base = 3
  private speed = CRUISE
  private turn = 0
  private vAlt = 0

  private readonly keys = new Set<string>()
  private stickX = 0
  private stickY = 0
  private boostHeld = false

  private phase: Phase = 'title'
  private blendRaw = 0
  private readonly camPos = new THREE.Vector3()
  private readonly camLook = new THREE.Vector3()
  private readonly camUp = new THREE.Vector3(0, 1, 0)
  private camReady = false

  private time = 0
  private flown = 0
  private pollen = 0
  private honey = 0
  private gathered = 0
  private lv = 0
  private hinted = false
  private atHive = false
  private readonly hue = new THREE.Color()
  /** a test pilot for setting medal times; never on for players */
  private bot: { boost: boolean } | null = null
  private lastLap = 0
  private hudT = 0
  private trailT = 0
  private fireT = 0
  private readonly pending: { at: number; run: () => void }[] = []
  private finished = false

  private raf = 0
  private last = 0
  private disposed = false

  private readonly a = new THREE.Vector3()
  private readonly b = new THREE.Vector3()
  private readonly c = new THREE.Vector3()
  private readonly basis = new THREE.Matrix4()

  constructor(
    private readonly canvas: HTMLCanvasElement,
    private readonly events: PlanetEvents
  ) {
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, powerPreference: 'high-performance' })
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping
    this.renderer.toneMappingExposure = 1.05
    this.scene.add(this.world.group)
    this.scene.add(this.life.group)
    this.races = new Races(this.world, {
      started: (r) => {
        this.audio.raceStart()
        this.events.toast(`${REGIONS[r].name} · 竞速开始`, '沿着金色的环飞一圈，回到起点', false)
      },
      gate: (_r, passed) => {
        this.audio.gate(passed)
        this.puff(GOLD, 14)
      },
      finished: (r, seconds, medal, record) => {
        this.lastLap = seconds
        const prize = PRIZES[medal]
        this.pollen += prize
        this.gathered += prize
        this.audio.fanfare(medal)
        this.puff(WHITE, 60)
        const course = this.races.courses[r]
        const next = medal === 3 ? '' : ` · ${MEDALS[medal + 1]}要 ${(medal === 2 ? course.gold : medal === 1 ? course.silver : course.bronze).toFixed(1)} 秒内`
        this.events.toast(`${MEDALS[medal]} · ${seconds.toFixed(1)} 秒`, `${record ? '新纪录 · ' : ''}花粉 +${prize}${next}`, true)
        this.save()
      },
      expired: () => {
        this.lastLap = -1
        this.audio.fanfare(0)
        this.events.toast('超时了', '再穿一次青色光环重新开始', false)
      }
    })
    this.scene.add(this.races.group)
    this.load()
    this.scene.add(this.bee.root)
    this.scene.add(this.sparks.points)
    this.shadow = new THREE.Mesh(
      new THREE.CircleGeometry(1, 14).rotateX(-Math.PI / 2),
      new THREE.MeshBasicMaterial({ color: '#0b1020', transparent: true, opacity: 0.3, depthWrite: false })
    )
    this.shadow.matrixAutoUpdate = false
    this.shadow.renderOrder = 2
    this.scene.add(this.shadow)
    this.halo = new THREE.Mesh(
      new THREE.IcosahedronGeometry(1.05, 2),
      new THREE.MeshBasicMaterial({ color: '#ffc21a', transparent: true, opacity: 0.2, depthWrite: false, toneMapped: false })
    )
    this.halo.visible = false
    this.bee.model.add(this.halo)

    // start above the hive island, heading for its first flower field
    // start off the hive island's shore, facing its first flower field, with the hive beyond
    const home = this.world.centers[0]
    const first = this.world.patches[0].n
    this.a.copy(first).addScaledVector(home, -home.dot(first)).normalize()
    this.n.copy(first).addScaledVector(this.a, 30 / R).normalize()
    this.fwd.copy(first).addScaledVector(this.n, -first.dot(this.n)).normalize()
    this.speed = 0

    this.resize()
    window.addEventListener('keydown', this.onKeyDown)
    window.addEventListener('keyup', this.onKeyUp)
    window.addEventListener('blur', this.onBlur)
    this.last = performance.now()
    this.raf = requestAnimationFrame(this.frame)
  }

  // ---- input ----

  private readonly onKeyDown = (e: KeyboardEvent): void => {
    const k = e.key.length === 1 ? e.key.toLowerCase() : e.key
    if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', ' '].includes(k) && this.phase === 'fly') e.preventDefault()
    this.keys.add(k)
  }

  private readonly onKeyUp = (e: KeyboardEvent): void => {
    this.keys.delete(e.key.length === 1 ? e.key.toLowerCase() : e.key)
  }

  private readonly onBlur = (): void => {
    this.keys.clear()
  }

  /** Touch or mouse stick: x right, y up, both -1..1. */
  setStick(x: number, y: number): void {
    this.stickX = clamp(x, -1, 1)
    this.stickY = clamp(y, -1, 1)
  }

  setBoost(on: boolean): void {
    this.boostHeld = on
  }

  setMuted(m: boolean): void {
    this.audio.setMuted(m)
  }

  /** Leave the title screen. Must be called from a click so the browser allows sound. */
  start(): void {
    this.audio.start()
    const awake = this.world.awake.filter(Boolean).length
    this.audio.setLevel(awake)
    if (this.phase === 'title') {
      this.phase = 'fly'
      if (this.finished) this.events.toast('星球已经醒了', '随便飞飞吧', false)
      else if (this.world.progress() > 0) this.events.toast('欢迎回来', `还有 ${REGIONS.length - awake} 座岛在睡`, false)
      else this.events.toast('穿过金色的光环', '花田就醒了', false)
    } else if (this.phase === 'finale') {
      this.phase = 'fly'
    }
  }

  resize(): void {
    const w = this.canvas.clientWidth || window.innerWidth
    const h = this.canvas.clientHeight || window.innerHeight
    const dpr = Math.min(window.devicePixelRatio || 1, w < 768 ? 1.75 : 2)
    this.renderer.setPixelRatio(dpr)
    this.renderer.setSize(w, h, false)
    this.camera.aspect = w / h
    this.camera.fov = w < h ? 78 : 62
    this.camera.updateProjectionMatrix()
    this.sparks.setScale(h * dpr * 0.9)
    this.life.setScale(h * dpr * 0.9)
  }

  // ---- loop ----

  private readonly frame = (now: number): void => {
    if (this.disposed) return
    const dt = Math.min(0.05, Math.max(0.001, (now - this.last) / 1000))
    this.last = now
    this.step(dt)
    this.renderer.render(this.scene, this.camera)
    this.raf = requestAnimationFrame(this.frame)
  }

  private step(dt: number): void {
    this.time += dt
    const flying = this.phase === 'fly'

    // input, or a lazy circle while the title or ending is up
    let steer = 0
    let pitch = 0
    let boost = false
    let slow = false
    if (flying) {
      const k = this.keys
      steer = (k.has('d') || k.has('ArrowRight') ? 1 : 0) - (k.has('a') || k.has('ArrowLeft') ? 1 : 0) + this.stickX
      pitch = (k.has('w') || k.has('ArrowUp') ? 1 : 0) - (k.has('s') || k.has('ArrowDown') ? 1 : 0) + this.stickY
      boost = k.has(' ') || k.has('Shift') || this.boostHeld
      slow = k.has('q')
      steer = clamp(steer, -1, 1)
      pitch = clamp(pitch, -1, 1)
      this.flown += dt
      const aim = this.bot ? this.races.target() : null
      if (this.bot && aim) {
        const turn = this.bearing(aim.n)
        steer = clamp(turn * 2.6, -1, 1)
        pitch = clamp((aim.alt - this.alt) * 0.45, -1, 1)
        boost = this.bot.boost && Math.abs(turn) < 0.3
        // a hoop that cannot be reached at this speed: slow down and come round tighter
        slow = Math.abs(turn) > 0.9
      }
    } else if (this.phase === 'finale') {
      steer = 0.22
    }
    // the bee waits on the title screen and eases off as the camera arrives
    const go = this.phase === 'title' ? 0 : this.phase === 'fly' ? smoothstep(0.5, 1, this.blendRaw) : 1

    this.turn = lerp(this.turn, steer * 1.55, damp(5.5, dt))
    this.speed = lerp(this.speed, (boost ? (this.lv >= 3 ? GALE : BOOST) : slow ? SLOW : CRUISE) * go, damp(boost ? 3 : 1.6, dt))
    this.vAlt = lerp(this.vAlt, pitch * 12, damp(4.5, dt))
    this.hover = clamp(this.hover + this.vAlt * dt, 4.2, 30)
    if ((this.hover <= 4.2 && this.vAlt < 0) || (this.hover >= 30 && this.vAlt > 0)) this.vAlt *= 0.8

    // turn, then move along the great circle
    this.fwd.applyAxisAngle(this.n, -this.turn * dt)
    this.a.crossVectors(this.n, this.fwd).normalize()
    const ang = (this.speed * dt) / (R + this.alt)
    this.n.applyAxisAngle(this.a, ang).normalize()
    this.fwd.applyAxisAngle(this.a, ang)
    this.fwd.addScaledVector(this.n, -this.fwd.dot(this.n)).normalize()

    // never crash: rise over what is under and just ahead
    const here = this.world.ground(this.n)
    const clearHere = this.world.clearance(this.n)
    this.b.copy(this.n).addScaledVector(this.fwd, 8 / R).normalize()
    const ahead = this.world.clearance(this.b)
    const under = Math.max(clearHere, ahead)
    this.base = lerp(this.base, under, damp(under > this.base ? 5 : 2.2, dt))
    this.alt = this.base + this.hover

    this.placeBee(dt, here)
    this.updateCamera(dt)
    const k = smoothstep(0, 1, this.blendRaw)
    this.a.copy(this.camera.position).normalize()
    this.world.update(dt, this.camera, k > 0.5 ? this.n : this.a, 1 - k)
    this.life.update(dt)

    if (flying && this.blendRaw > 0.9) {
      this.checkFields()
      this.checkPollen(dt)
      this.races.update(dt, this.bee.root.position, this.world.awake)
      // pollen is handed over on arriving at the hive, not grain by grain while circling it
      const atHive = this.n.dot(this.world.centers[0]) > Math.cos(13 / R)
      if (atHive && !this.atHive && this.pollen > 0) this.deliver()
      this.atHive = atHive
    }
    this.runPending()
    this.trail(dt, boost)
    if (this.phase === 'finale') this.fireworks(dt)
    this.sparks.update(dt)
    this.audio.flight(clamp((this.speed - SLOW) / (BOOST - SLOW), 0, 1), flying)

    this.hudT -= dt
    if (this.hudT <= 0) {
      this.hudT = 0.1
      this.pushHud()
    }
  }

  private placeBee(dt: number, ground: number): void {
    const h = R + this.alt
    // x = left, y = up, z = forward
    this.a.crossVectors(this.n, this.fwd).normalize()
    this.basis.makeBasis(this.a, this.n, this.fwd)
    this.bee.root.quaternion.setFromRotationMatrix(this.basis)
    this.bee.root.position.copy(this.n).multiplyScalar(h)
    const m = this.bee.model
    m.rotation.z = lerp(m.rotation.z, this.turn * 0.5, damp(8, dt))
    m.rotation.x = lerp(m.rotation.x, -this.vAlt * 0.035, damp(8, dt))
    m.position.y = Math.sin(this.time * 5.2) * 0.12
    for (const w of this.bee.wings) {
      const side = w.userData.side as number
      const back = w.userData.back as number
      w.rotation.z = side * (0.28 + Math.sin(this.time * 58 + back * 0.9) * 0.62)
    }
    this.bee.root.visible = this.blendRaw > 0.25
    this.halo.visible = this.lv >= 4
    this.halo.scale.setScalar(1 + Math.sin(this.time * 4) * 0.08)

    // a soft shadow on the ground tells you how high you are
    const gap = this.alt - ground
    const sc = clamp(1.5 - gap * 0.045, 0.35, 1.5)
    this.basis.scale(this.c.set(sc, 1, sc))
    this.basis.setPosition(this.n.x * (R + ground + 0.15), this.n.y * (R + ground + 0.15), this.n.z * (R + ground + 0.15))
    this.shadow.matrix.copy(this.basis)
    ;(this.shadow.material as THREE.MeshBasicMaterial).opacity = clamp(0.36 - gap * 0.012, 0, 0.36) * smoothstep(0.6, 1, this.blendRaw)
  }

  private updateCamera(dt: number): void {
    const target = this.phase === 'fly' ? 1 : 0
    const rate = this.phase === 'finale' ? 0.22 : 0.5
    this.blendRaw = clamp(this.blendRaw + Math.sign(target - this.blendRaw) * rate * dt, 0, 1)
    const k = smoothstep(0, 1, this.blendRaw)

    // chase pose: behind and above, looking a little ahead
    const sp = (this.speed - CRUISE) / (BOOST - CRUISE)
    const h = R + this.alt
    this.a.copy(this.n).multiplyScalar(h + 6.2).addScaledVector(this.fwd, -(10.5 + sp * 2.5))
    this.b.copy(this.n).multiplyScalar(h - 1.2).addScaledVector(this.fwd, 9)
    if (!this.camReady) {
      this.camPos.copy(this.a)
      this.camLook.copy(this.b)
      this.camUp.copy(this.n)
      this.camReady = true
    }
    this.camPos.lerp(this.a, damp(6.5, dt))
    this.camLook.lerp(this.b, damp(10, dt))
    this.camUp.lerp(this.n, damp(5, dt)).normalize()

    // orbit pose: the whole planet, turning slowly
    const oa = this.time * 0.06 + 0.4
    const far = 340 / Math.min(1, this.camera.aspect * 0.95)
    this.c.set(Math.sin(oa) * far, far * 0.3, Math.cos(oa) * far)

    this.camera.position.lerpVectors(this.c, this.camPos, k)
    this.a.set(0, 34, 0).lerp(this.camLook, k)
    this.camera.up.set(0, 1, 0).lerp(this.camUp, k).normalize()
    this.camera.lookAt(this.a)
    const fovBase = this.camera.aspect < 1 ? 78 : 62
    const fov = lerp(42, fovBase + sp * 9, k)
    if (Math.abs(fov - this.camera.fov) > 0.05) {
      this.camera.fov = fov
      this.camera.updateProjectionMatrix()
    }
  }

  // ---- what the bee does to the planet ----

  private checkFields(): void {
    const lim = Math.cos(6.6 / R)
    for (const p of this.world.patches) {
      if (p.bloomed || p.n.dot(this.n) < lim || this.hover > 12) continue
      this.bloomField(p)
    }
  }

  private bloomField(p: Patch): void {
    this.world.bloom(p, this.n)
    this.save()
    const done = this.world.bloomedIn(p.region)
    this.audio.bloom(done - 1)
    const col = new THREE.Color(REGIONS[p.region].flowers[0])
    const base = this.a.copy(p.n).multiplyScalar(R + p.ground + 1)
    for (let i = 0; i < 70; i++) {
      this.b.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).normalize()
      this.b.addScaledVector(p.n, 0.9).multiplyScalar(4 + Math.random() * 9)
      this.c.copy(base).addScaledVector(this.b, 0.15)
      this.sparks.emit(this.c, this.b, i % 3 ? col : WHITE, 0.3 + Math.random() * 0.5, 0.9 + Math.random() * 1.1)
    }
    if (done < PATCHES_PER_REGION) {
      this.events.toast(`${REGIONS[p.region].name} · 花田 ${done}/${PATCHES_PER_REGION}`, '', false)
      return
    }
    // the last field: a beat of silence, then the whole island wakes
    this.pending.push({
      at: this.time + 0.9,
      run: () => {
        this.world.wake(p.region)
        this.audio.wake()
        const count = this.world.awake.filter(Boolean).length
        this.audio.setLevel(count)
        this.events.toast(`${REGIONS[p.region].name}醒了`, count < REGIONS.length ? `还有 ${REGIONS.length - count} 座岛在睡 · 青色光环是竞速起点` : '整颗星球都醒了', true)
        this.burstAt(this.world.centers[p.region], col, 160)
        if (count === REGIONS.length && !this.finished) {
          this.finished = true
          this.pending.push({
            at: this.time + 5.5,
            run: () => {
              this.phase = 'finale'
              this.save()
              this.events.finale({ seconds: Math.round(this.flown), pollen: this.gathered })
            }
          })
        }
      }
    })
  }

  private burstAt(n: THREE.Vector3, col: THREE.Color, count: number): void {
    const base = this.a.copy(n).multiplyScalar(R + this.world.ground(n) + 9)
    for (let i = 0; i < count; i++) {
      this.b.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).normalize().multiplyScalar(8 + Math.random() * 22)
      this.sparks.emit(base, this.b, i % 2 ? col : GOLD, 0.5 + Math.random() * 0.9, 1.4 + Math.random() * 1.6)
    }
  }

  /** Which way a point lies from the bee's nose: radians, 0 ahead, positive to the right. */
  private bearing(n: THREE.Vector3): number {
    this.a.copy(n).addScaledVector(this.n, -n.dot(this.n))
    this.b.crossVectors(this.fwd, this.n)
    return Math.atan2(this.a.dot(this.b), this.a.dot(this.fwd))
  }

  /** A puff of sparks round the bee. */
  private puff(col: THREE.Color, count: number): void {
    for (let i = 0; i < count; i++) {
      this.b.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).normalize().multiplyScalar(5 + Math.random() * 9)
      this.b.addScaledVector(this.fwd, this.speed * 0.6)
      this.sparks.emit(this.bee.root.position, this.b, col, 0.3 + Math.random() * 0.4, 0.5 + Math.random() * 0.5)
    }
  }

  /** Over the hive: the pollen the bee carries becomes honey, and enough honey is a new level. */
  private deliver(): void {
    const before = this.lv
    const load = this.pollen
    this.honey += load
    this.pollen = 0
    this.lv = LEVELS.filter((l) => this.honey >= l.need).length
    this.audio.deliver(this.lv > before)
    // the pollen streams down into the hive
    const hive = this.world.centers[0]
    this.c.copy(hive).multiplyScalar(R + this.world.ground(hive) + 9)
    for (let i = 0; i < Math.min(90, 20 + load * 2); i++) {
      this.a.copy(this.bee.root.position)
      this.b.copy(this.c).sub(this.a).multiplyScalar(1.6 + Math.random() * 1.2)
      this.b.x += (Math.random() - 0.5) * 9
      this.b.y += (Math.random() - 0.5) * 9
      this.b.z += (Math.random() - 0.5) * 9
      this.sparks.emit(this.a, this.b, i % 4 ? GOLD : WHITE, 0.3 + Math.random() * 0.4, 0.5 + Math.random() * 0.5)
    }
    if (this.lv > before) {
      const got = LEVELS[this.lv - 1]
      this.events.toast(`蜂蜜 ${this.lv} 级 · ${got.name}`, got.desc, true)
      this.burstAt(hive, GOLD, 120)
    } else {
      const next = LEVELS[this.lv]
      this.events.toast(`酿了 ${load} 份蜂蜜`, next ? `再要 ${next.need - this.honey} 份解锁「${next.name}」` : '蜂巢已经满了', false)
    }
    this.save()
  }

  private checkPollen(dt: number): void {
    const lim = Math.cos(3.1 / R)
    const pull = Math.cos(11 / R)
    for (const o of this.world.orbs) {
      if (o.taken) continue
      const d = o.n.dot(this.n)
      // with the magnet, grains nearby fly to the bee
      if (this.lv >= 1 && d > pull && Math.abs(o.alt - this.alt) < 12) {
        o.n.lerp(this.n, damp(5, dt)).normalize()
        o.alt = lerp(o.alt, this.alt, damp(5, dt))
      }
      if (d < lim || Math.abs(o.alt - this.alt) > 4) continue
      o.taken = true
      o.back = this.world.now + 45
      this.pollen++
      this.gathered++
      if (this.gathered % 10 === 0) this.save()
      if (!this.hinted && this.honey === 0 && this.pollen >= 8) {
        this.hinted = true
        this.events.toast('把花粉带回蜂巢', '酿成蜂蜜，能学会新本领', false)
      }
      this.audio.note()
      const base = this.a.copy(o.n).multiplyScalar(R + o.alt)
      for (let i = 0; i < 9; i++) {
        this.b.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).multiplyScalar(9)
        this.sparks.emit(base, this.b, GOLD, 0.3 + Math.random() * 0.35, 0.5 + Math.random() * 0.4)
      }
    }
  }

  // ---- progress is kept in the browser, so a visitor who comes back finds the islands awake ----

  private load(): void {
    try {
      const raw = window.localStorage.getItem(SAVE_KEY)
      if (!raw) return
      const s = JSON.parse(raw) as Partial<Save>
      if (!Array.isArray(s.fields) || s.fields.length !== this.world.patches.length) return
      this.world.restore(s.fields.map(Boolean))
      this.pollen = Number(s.pollen) || 0
      this.honey = Number(s.honey) || 0
      this.gathered = Number(s.gathered) || this.pollen + this.honey
      this.lv = LEVELS.filter((l) => this.honey >= l.need).length
      this.hinted = this.honey > 0
      if (Array.isArray(s.best)) s.best.slice(0, this.races.best.length).forEach((b, i) => (this.races.best[i] = Number(b) || 0))
      this.flown = Number(s.seconds) || 0
      this.finished = !!s.done && this.world.awake.every(Boolean)
    } catch {
      // a broken or blocked store just means starting fresh
    }
  }

  private save(): void {
    try {
      const s: Save = {
        fields: this.world.patches.map((p) => p.bloomed),
        pollen: this.pollen,
        honey: this.honey,
        gathered: this.gathered,
        seconds: Math.round(this.flown),
        done: this.finished,
        best: [...this.races.best]
      }
      window.localStorage.setItem(SAVE_KEY, JSON.stringify(s))
    } catch {
      // private mode or a full store: play on without saving
    }
  }

  /** What a returning player already did, for the title screen. */
  saved(): { fields: number; islands: number; done: boolean } {
    return { fields: this.world.patches.filter((p) => p.bloomed).length, islands: this.world.awake.filter(Boolean).length, done: this.finished }
  }

  /** Forget everything. The page reloads afterwards to rebuild the sleeping planet. */
  clearSave(): void {
    try {
      window.localStorage.removeItem(SAVE_KEY)
    } catch {
      // nothing was saved
    }
  }

  private runPending(): void {
    for (let i = this.pending.length - 1; i >= 0; i--) {
      if (this.pending[i].at > this.time) continue
      const job = this.pending[i]
      this.pending.splice(i, 1)
      job.run()
    }
  }

  private trail(dt: number, boost: boolean): void {
    if (this.blendRaw < 0.6) return
    this.trailT -= dt
    const every = boost ? 0.012 : 0.035
    while (this.trailT <= 0) {
      this.trailT += every
      this.a.copy(this.bee.root.position).addScaledVector(this.fwd, -1.1)
      this.a.x += (Math.random() - 0.5) * 0.5
      this.a.y += (Math.random() - 0.5) * 0.5
      this.a.z += (Math.random() - 0.5) * 0.5
      this.b.copy(this.fwd).multiplyScalar(this.speed * 0.55).addScaledVector(this.n, -1.4)
      // honey level 2: the trail runs through the rainbow
      const col = this.lv >= 2 ? this.hue.setHSL((this.time * 0.55) % 1, 0.95, 0.62) : boost ? WHITE : GOLD
      this.sparks.emit(this.a, this.b, col, (boost ? 0.3 : 0.2) * (this.lv >= 2 ? 1.5 : 1), boost ? 0.55 : this.lv >= 2 ? 0.6 : 0.4)
    }
  }

  private fireworks(dt: number): void {
    this.fireT -= dt
    if (this.fireT > 0) return
    this.fireT = 0.35 + Math.random() * 0.4
    const r = Math.floor(Math.random() * REGIONS.length)
    this.burstAt(this.world.centers[r], new THREE.Color(REGIONS[r].flowers[Math.floor(Math.random() * 3) % REGIONS[r].flowers.length]), 60)
  }

  private pushHud(): void {
    const sm = this.world.sample(this.n)
    const over = sm.d < 1.0
    const region = sm.region
    const fields = this.world.patches.filter((p) => p.region === region).map((p) => p.bloomed)

    // point at the nearest sleeping field, on this island first
    let best: Patch | null = null
    let bestDot = -2
    let local = false
    for (const p of this.world.patches) {
      if (p.bloomed) continue
      const here = over && p.region === region
      const d = p.n.dot(this.n)
      if ((here && !local) || (here === local && d > bestDot)) {
        best = p
        bestDot = d
        local = here
      }
    }
    let arrow = 0
    let dist = 0
    let label = ''
    const gate = this.races.target()
    if (gate) {
      // during a lap the arrow leads to the next hoop
      arrow = this.bearing(gate.n)
      dist = Math.acos(clamp(gate.n.dot(this.n), -1, 1)) * R
      label = this.races.passed >= GATES ? '终点' : '下一个环'
    } else if (best) {
      arrow = this.bearing(best.n)
      dist = Math.acos(clamp(bestDot, -1, 1)) * R
      label = local ? '花田' : REGIONS[best.region].name
    }
    const hive = this.world.centers[0]
    const course = this.races.active >= 0 ? this.races.courses[this.races.active] : null
    const nextLevel = LEVELS[this.lv]
    this.events.hud({
      phase: this.phase,
      region: over ? REGIONS[region].name : '',
      regionIndex: region,
      fields,
      awake: [...this.world.awake],
      pollen: this.pollen,
      honey: this.honey,
      level: this.lv,
      levelFrom: this.lv ? LEVELS[this.lv - 1].need : 0,
      levelNeed: nextLevel ? nextLevel.need : 0,
      levelName: nextLevel ? nextLevel.name : '',
      arrow,
      arrowDist: dist,
      arrowLabel: label,
      hiveArrow: this.bearing(hive),
      hiveDist: Math.acos(clamp(hive.dot(this.n), -1, 1)) * R,
      race: course
        ? { seconds: this.races.clock, passed: this.races.passed, total: GATES, gold: course.gold, silver: course.silver, best: this.races.best[this.races.active] }
        : null,
      medals: this.races.best.map((b, i) => medalFor(this.races.courses[i], b))
    })
  }

  // ---- for checking the game by hand in a browser tab that is not on screen ----

  /** Run `count` frames of `dt` seconds and draw once. */
  debugStep(count: number, dt = 1 / 60): void {
    for (let i = 0; i < count; i++) this.step(dt)
    this.renderer.render(this.scene, this.camera)
  }

  /** Put the bee just short of a flower field, facing it. */
  debugGoto(region: number, field: number, back = 22): void {
    const p = this.world.patches[region * PATCHES_PER_REGION + field]
    const c = this.world.centers[region]
    this.a.copy(c).addScaledVector(p.n, -c.dot(p.n)).normalize()
    this.n.copy(p.n).addScaledVector(this.a, back / R).normalize()
    this.fwd.copy(p.n).addScaledVector(this.n, -p.n.dot(this.n)).normalize()
    this.hover = 6
    this.base = this.world.ground(this.n)
    this.alt = this.base + this.hover
    this.camReady = false
  }

  /** Fly one lap of an island's race on autopilot and return the time (-1 if it ran out). */
  debugRace(region: number, boost: boolean): number {
    this.world.awake[region] = true
    this.races.abandon()
    const gates = this.races.courses[region].gates
    const g0 = gates[0]
    this.a.copy(gates[GATES - 1].n).addScaledVector(g0.n, -g0.n.dot(gates[GATES - 1].n)).normalize()
    this.n.copy(g0.n).addScaledVector(this.a, 20 / R).normalize()
    this.fwd.copy(g0.n).addScaledVector(this.n, -g0.n.dot(this.n)).normalize()
    this.base = this.world.clearance(this.n)
    this.hover = clamp(g0.alt - this.base, 4.2, 30)
    this.speed = boost ? BOOST : CRUISE
    this.camReady = false
    const keep = this.races.best[region]
    this.lastLap = 0
    // let the restart cooldown run out, then head through the start hoop
    for (let i = 0; i < 60 * 4 && this.races.active < 0; i++) {
      if (i === 200) this.debugGotoGate(region)
      this.step(1 / 60)
    }
    this.bot = { boost }
    for (let i = 0; i < 60 * 200 && this.lastLap === 0; i++) this.step(1 / 60)
    this.bot = null
    this.races.best[region] = keep
    this.renderer.render(this.scene, this.camera)
    return this.lastLap
  }

  private debugGotoGate(region: number): void {
    const gates = this.races.courses[region].gates
    const g0 = gates[0]
    this.a.copy(gates[GATES - 1].n).addScaledVector(g0.n, -g0.n.dot(gates[GATES - 1].n)).normalize()
    this.n.copy(g0.n).addScaledVector(this.a, 12 / R).normalize()
    this.fwd.copy(g0.n).addScaledVector(this.n, -g0.n.dot(this.n)).normalize()
    this.base = this.world.clearance(this.n)
    this.hover = clamp(g0.alt - this.base, 4.2, 30)
  }

  debugWake(region: number): void {
    for (const p of this.world.patches) if (p.region === region && !p.bloomed) this.bloomField(p)
  }

  dispose(): void {
    this.disposed = true
    cancelAnimationFrame(this.raf)
    window.removeEventListener('keydown', this.onKeyDown)
    window.removeEventListener('keyup', this.onKeyUp)
    window.removeEventListener('blur', this.onBlur)
    this.audio.dispose()
    this.sparks.dispose()
    this.world.dispose()
    this.renderer.dispose()
  }
}
