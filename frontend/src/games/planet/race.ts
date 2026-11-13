// Ring races for 蜂巢星球. Every island has one lap of hoops round it. The cyan start hoop shows
// once the island is awake; flying through it starts the clock, and the lap ends back at the same
// hoop. Medals depend on the time; the best time per island is kept.
import * as THREE from 'three'
import { mulberry32 } from './noise'
import { R, REGIONS, beamMaterial, type World } from './world'

export const GATES = 14
const HOOP = 4.8
/** 0 none, 1 bronze, 2 silver, 3 gold */
export type Medal = 0 | 1 | 2 | 3

interface Gate {
  pos: THREE.Vector3
  n: THREE.Vector3
  alt: number
  quat: THREE.Quaternion
}

export interface Course {
  gates: Gate[]
  length: number
  gold: number
  silver: number
  bronze: number
}

export interface RaceEvents {
  started(region: number): void
  gate(region: number, passed: number): void
  finished(region: number, seconds: number, medal: Medal, record: boolean): void
  expired(region: number): void
}

export function medalFor(course: Course, seconds: number): Medal {
  if (seconds <= 0) return 0
  return seconds <= course.gold ? 3 : seconds <= course.silver ? 2 : seconds <= course.bronze ? 1 : 0
}

export class Races {
  readonly group = new THREE.Group()
  readonly courses: Course[] = []
  /** best lap per island in seconds, 0 = never finished */
  readonly best: number[] = REGIONS.map(() => 0)
  /** island being raced, -1 when idle */
  active = -1
  /** gates passed so far in this lap (the start hoop counts as the first) */
  passed = 0
  clock = 0

  private readonly starts: THREE.Group[] = []
  private readonly hoops: THREE.Mesh[] = []
  private readonly nextMat = new THREE.MeshBasicMaterial({ color: '#ffd23f', toneMapped: false })
  private readonly laterMat = new THREE.MeshBasicMaterial({ color: '#ffffff', transparent: true, opacity: 0.4, depthWrite: false })
  private readonly startMat = new THREE.MeshBasicMaterial({ color: '#6ef3ff', toneMapped: false })
  private readonly beamMat = beamMaterial('#6ef3ff', 0.3, false)
  private cooldown = 0
  private time = 0

  constructor(
    world: World,
    private readonly events: RaceEvents
  ) {
    const rnd = mulberry32(8128)
    const hoopGeo = new THREE.TorusGeometry(HOOP, 0.24, 6, 30)
    const startGeo = new THREE.TorusGeometry(HOOP + 0.4, 0.36, 6, 30)
    const beamGeo = new THREE.CylinderGeometry(1.0, 0.45, 46, 12, 1, true).translate(0, 28, 0)
    const x = new THREE.Vector3()
    const f = new THREE.Vector3()
    const basis = new THREE.Matrix4()
    for (let r = 0; r < REGIONS.length; r++) {
      const turn = rnd() < 0.5 ? 1 : -1
      const phase = rnd() * 6.283
      const s1 = rnd() * 6.283
      const s2 = rnd() * 6.283
      const gates: Gate[] = []
      for (let k = 0; k < GATES; k++) {
        const a = phase + turn * (k / GATES) * Math.PI * 2
        const n = new THREE.Vector3()
        world.around(r, a, REGIONS[r].rad * (0.76 + 0.15 * Math.sin(3 * a + s1)), n)
        gates.push({ pos: new THREE.Vector3(), n, alt: 7.5 + 3.5 * Math.sin(2 * a + s2), quat: new THREE.Quaternion() })
      }
      // a hoop hangs above the highest ground on the way to it, because the bee rides over hills
      for (let k = 0; k < GATES; k++) {
        const g = gates[k]
        const prev = gates[(k + GATES - 1) % GATES].n
        let top = world.ground(g.n)
        for (const t of [0.4, 0.6, 0.8]) top = Math.max(top, world.ground(x.copy(prev).lerp(g.n, t).normalize()))
        g.alt += top
        g.pos.copy(g.n).multiplyScalar(R + g.alt)
      }
      let length = 0
      for (let k = 0; k < GATES; k++) {
        const prev = gates[(k + GATES - 1) % GATES]
        const next = gates[(k + 1) % GATES]
        length += gates[k].pos.distanceTo(next.pos)
        // a hoop faces along the lap
        f.copy(next.pos).sub(prev.pos)
        f.addScaledVector(gates[k].n, -f.dot(gates[k].n)).normalize()
        x.crossVectors(gates[k].n, f).normalize()
        basis.makeBasis(x, gates[k].n, f)
        gates[k].quat.setFromRotationMatrix(basis)
      }
      // medal times come from test laps: a steady lap at cruising speed averages 22 units a second
      // and earns silver; the best boosted laps average 30, so gold takes boosting on the straights
      this.courses.push({ gates, length, gold: length / 26.5, silver: length / 21, bronze: length / 15 })

      const start = new THREE.Group()
      start.position.copy(gates[0].pos)
      start.quaternion.copy(gates[0].quat)
      start.add(new THREE.Mesh(startGeo, this.startMat))
      const beam = new THREE.Mesh(beamGeo, this.beamMat)
      start.add(beam)
      start.visible = false
      this.starts.push(start)
      this.group.add(start)
    }
    for (let k = 0; k < GATES; k++) {
      const hoop = new THREE.Mesh(hoopGeo, this.laterMat)
      hoop.visible = false
      this.hoops.push(hoop)
      this.group.add(hoop)
    }
  }

  /** Where the racer should head next, or null when no lap is running. */
  target(): Gate | null {
    if (this.active < 0) return null
    return this.courses[this.active].gates[this.passed % GATES]
  }

  private show(): void {
    const gates = this.active >= 0 ? this.courses[this.active].gates : []
    this.hoops.forEach((hoop, k) => {
      // hoop 0 is the start hoop itself, which also ends the lap
      const ahead = this.active >= 0 && k > 0 && k >= this.passed
      hoop.visible = ahead
      if (!ahead) return
      hoop.position.copy(gates[k].pos)
      hoop.quaternion.copy(gates[k].quat)
      hoop.material = k === this.passed ? this.nextMat : this.laterMat
      hoop.scale.setScalar(1)
    })
  }

  private stop(): void {
    this.active = -1
    this.passed = 0
    this.cooldown = 3
    this.show()
  }

  /** Give up the lap in progress. */
  abandon(): void {
    if (this.active >= 0) this.stop()
  }

  update(dt: number, bee: THREE.Vector3, awake: boolean[]): void {
    this.time += dt
    this.cooldown = Math.max(0, this.cooldown - dt)
    this.beamMat.uniforms.uTime.value = this.time
    const pulse = 1 + Math.sin(this.time * 5) * 0.06

    this.starts.forEach((start, r) => {
      // while a lap runs, only its own start hoop stays, and turns gold when it is the finish
      start.visible = awake[r] && (this.active < 0 || this.active === r)
      const finishing = this.active === r && this.passed >= GATES
      ;(start.children[0] as THREE.Mesh).material = finishing ? this.nextMat : this.startMat
      start.children[1].visible = this.active < 0
      start.scale.setScalar(finishing || this.active < 0 ? pulse : 1)
    })

    if (this.active < 0) {
      if (this.cooldown > 0) return
      for (let r = 0; r < this.courses.length; r++) {
        if (!awake[r] || this.courses[r].gates[0].pos.distanceTo(bee) > HOOP + 0.3) continue
        this.active = r
        this.passed = 1
        this.clock = 0
        this.show()
        this.events.started(r)
        return
      }
      return
    }

    const course = this.courses[this.active]
    this.clock += dt
    const hoop = this.hoops[this.passed]
    if (hoop && this.passed < GATES) hoop.scale.setScalar(pulse)
    if (course.gates[this.passed % GATES].pos.distanceTo(bee) <= HOOP + 0.4) {
      this.passed++
      if (this.passed > GATES) {
        const region = this.active
        const seconds = Math.round(this.clock * 100) / 100
        const record = this.best[region] === 0 || seconds < this.best[region]
        if (record) this.best[region] = seconds
        this.stop()
        this.events.finished(region, seconds, medalFor(course, seconds), record)
        return
      }
      this.show()
      this.events.gate(this.active, this.passed - 1)
      return
    }
    if (this.clock > course.bronze * 2 + 10) {
      const region = this.active
      this.stop()
      this.events.expired(region)
    }
  }
}
