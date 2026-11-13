// What moves on 蜂巢星球 besides the bee: butterflies over open flower fields, fireflies on woken
// islands at night, hot-air balloons, and sailing boats.
import * as THREE from 'three'
import { mergeGeometries } from 'three/examples/jsm/utils/BufferGeometryUtils.js'
import { clamp, mulberry32 } from './noise'
import { R, REGIONS, sunUniform, wakeUniforms, type World } from './world'

const PER_FIELD = 5

interface Flutter {
  field: number
  radius: number
  speed: number
  height: number
  phase: number
}

interface Boat {
  mesh: THREE.Mesh
  centre: THREE.Vector3
  u: THREE.Vector3
  w: THREE.Vector3
  radius: number
  speed: number
  phase: number
}

function coloured(geo: THREE.BufferGeometry, pick: (x: number, y: number, z: number) => THREE.Color): THREE.BufferGeometry {
  const g = geo.index ? geo.toNonIndexed() : geo
  g.deleteAttribute('uv')
  const p = g.attributes.position
  const arr = new Float32Array(p.count * 3)
  for (let i = 0; i < p.count; i += 3) {
    const c = pick((p.getX(i) + p.getX(i + 1) + p.getX(i + 2)) / 3, (p.getY(i) + p.getY(i + 1) + p.getY(i + 2)) / 3, (p.getZ(i) + p.getZ(i + 1) + p.getZ(i + 2)) / 3)
    for (let k = 0; k < 3; k++) arr.set([c.r, c.g, c.b], (i + k) * 3)
  }
  g.setAttribute('color', new THREE.BufferAttribute(arr, 3))
  return g
}

export class Life {
  readonly group = new THREE.Group()
  private readonly rnd = mulberry32(4242)
  private readonly timeUniform = { value: 0 }
  private butterflies!: THREE.InstancedMesh
  private readonly flutters: Flutter[] = []
  private readonly fieldU: THREE.Vector3[] = []
  private readonly fieldW: THREE.Vector3[] = []
  private fireflies!: THREE.Points
  private readonly rigs: THREE.Group[] = []
  private readonly boats: Boat[] = []
  private readonly m = new THREE.Matrix4()
  private readonly n = new THREE.Vector3()
  private readonly f = new THREE.Vector3()
  private readonly x = new THREE.Vector3()
  private time = 0

  constructor(private readonly world: World) {
    this.buildButterflies()
    this.buildFireflies()
    this.buildBalloons()
    this.buildBoats()
  }

  private buildButterflies(): void {
    const wing = (sx: number): THREE.BufferGeometry => new THREE.CircleGeometry(0.22, 5).scale(1, 1.25, 1).rotateX(-Math.PI / 2).translate(sx * 0.22, 0, 0).toNonIndexed()
    const geo = mergeGeometries([wing(-1), wing(1)], false)
    geo.deleteAttribute('uv')
    const count = this.world.patches.length * PER_FIELD
    const phase = new Float32Array(count)
    const time = this.timeUniform
    const mat = new THREE.MeshBasicMaterial({ side: THREE.DoubleSide })
    mat.onBeforeCompile = (sh) => {
      sh.uniforms.uTime = time
      sh.vertexShader = sh.vertexShader
        .replace('#include <common>', '#include <common>\nuniform float uTime;\nattribute float aPhase;')
        .replace('#include <begin_vertex>', '#include <begin_vertex>\ntransformed.y += abs(transformed.x) * sin(uTime * 15.0 + aPhase) * 1.15;')
    }
    const mesh = new THREE.InstancedMesh(geo, mat, count)
    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage)
    const zero = new THREE.Matrix4().makeScale(0, 0, 0)
    const white = new THREE.Color('#ffffff')
    this.world.patches.forEach((p, fi) => {
      const ref = Math.abs(p.n.y) < 0.9 ? new THREE.Vector3(0, 1, 0) : new THREE.Vector3(1, 0, 0)
      const u = new THREE.Vector3().crossVectors(ref, p.n).normalize()
      this.fieldU.push(u)
      this.fieldW.push(new THREE.Vector3().crossVectors(p.n, u))
      const cols = REGIONS[p.region].flowers
      for (let k = 0; k < PER_FIELD; k++) {
        const i = fi * PER_FIELD + k
        this.flutters.push({
          field: fi,
          radius: 1.6 + this.rnd() * 4.2,
          speed: (0.35 + this.rnd() * 0.5) * (this.rnd() < 0.5 ? -1 : 1),
          height: 1.3 + this.rnd() * 2.2,
          phase: this.rnd() * 6.283
        })
        phase[i] = this.rnd() * 6.283
        mesh.setMatrixAt(i, zero)
        mesh.setColorAt(i, k % 3 === 0 ? white : new THREE.Color(cols[k % cols.length]))
      }
    })
    geo.setAttribute('aPhase', new THREE.InstancedBufferAttribute(phase, 1))
    mesh.frustumCulled = false
    this.butterflies = mesh
    this.group.add(mesh)
  }

  private buildFireflies(): void {
    const per = 130
    const count = REGIONS.length * per
    const pos = new Float32Array(count * 3)
    const info = new Float32Array(count * 3)
    const n = new THREE.Vector3()
    let i = 0
    for (let r = 0; r < REGIONS.length; r++) {
      let made = 0
      for (let tries = 0; tries < per * 20 && made < per; tries++) {
        this.world.around(r, this.rnd() * 6.283, Math.sqrt(this.rnd()) * 0.8 * REGIONS[r].rad, n)
        const sm = this.world.sample(n)
        if (sm.e < 0.8 || sm.region !== r) continue
        const h = R + sm.e + 1 + this.rnd() * 3.5
        pos.set([n.x * h, n.y * h, n.z * h], i * 3)
        info.set([r, sm.d, this.rnd() * 6.283], i * 3)
        i++
        made++
      }
    }
    const geo = new THREE.BufferGeometry()
    geo.setAttribute('position', new THREE.BufferAttribute(pos.slice(0, i * 3), 3))
    geo.setAttribute('aInfo', new THREE.BufferAttribute(info.slice(0, i * 3), 3))
    this.fireflies = new THREE.Points(
      geo,
      new THREE.ShaderMaterial({
        uniforms: { uTime: this.timeUniform, uSun: sunUniform, uFront: wakeUniforms.uFront, uScale: { value: 600 }, uMax: { value: 30 } },
        vertexShader: `
          attribute vec3 aInfo; uniform float uTime; uniform vec3 uSun; uniform float uFront[6]; uniform float uScale; uniform float uMax; varying float vA;
          void main(){
            vec3 up = normalize(position);
            float night = smoothstep(0.08, -0.25, dot(up, uSun));
            float fr = uFront[int(aInfo.x + 0.5)];
            float awake = 1.0 - smoothstep(fr - 0.16, fr, aInfo.y);
            float tw = 0.5 + 0.5 * sin(uTime * 2.3 + aInfo.z);
            vA = night * awake * tw;
            vec3 p = position + up * sin(uTime * 0.7 + aInfo.z * 1.7) * 0.7;
            vec4 mv = modelViewMatrix * vec4(p, 1.0);
            gl_PointSize = min(uMax, 0.55 * uScale / max(0.1, -mv.z)) * step(0.01, vA);
            gl_Position = projectionMatrix * mv;
          }`,
        fragmentShader:
          'varying float vA; void main(){ float d = length(gl_PointCoord - 0.5); float a = smoothstep(0.5, 0.0, d); a *= a * vA; gl_FragColor = vec4(vec3(0.85, 1.0, 0.42) * a, a); }',
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })
    )
    this.fireflies.frustumCulled = false
    this.group.add(this.fireflies)
  }

  private buildBalloons(): void {
    const pairs: [string, string][] = [['#ff6b6b', '#fff3d6'], ['#4dabf7', '#fff3d6'], ['#ffd43b', '#ff922b'], ['#b197fc', '#fff3d6'], ['#63e6be', '#fff3d6'], ['#ff8fab', '#ffe066']]
    const mat = new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true, emissive: '#554433', emissiveIntensity: 0.5 })
    for (let g = 0; g < 2; g++) {
      const rig = new THREE.Group()
      rig.userData.axis = new THREE.Vector3(g ? -0.5 : 0.4, 1, g ? 0.2 : -0.3).normalize()
      rig.userData.speed = g ? 0.02 : -0.016
      this.rigs.push(rig)
      this.group.add(rig)
    }
    const n = new THREE.Vector3()
    pairs.forEach(([a, b], i) => {
      const ca = new THREE.Color(a)
      const cb = new THREE.Color(b)
      const basket = new THREE.Color('#8a5a3a')
      const envelope = coloured(new THREE.SphereGeometry(1.7, 10, 7).scale(1, 1.18, 1), (x, _y, z) => (Math.floor(((Math.atan2(x, z) + Math.PI) / (Math.PI * 2)) * 10 + 0.001) % 2 ? ca : cb))
      const neck = coloured(new THREE.CylinderGeometry(0.75, 0.42, 0.9, 8).translate(0, -2.1, 0), () => ca)
      const box = coloured(new THREE.BoxGeometry(0.62, 0.5, 0.62).translate(0, -3.2, 0), () => basket)
      const geo = mergeGeometries([envelope, neck, box], false)
      geo.computeVertexNormals()
      const mesh = new THREE.Mesh(geo, mat)
      n.set(this.rnd() * 2 - 1, this.rnd() * 2 - 1, this.rnd() * 2 - 1).normalize()
      this.world.place(n, 17 + this.rnd() * 7, this.rnd() * 6.283, 1, mesh.matrix)
      mesh.matrixAutoUpdate = false
      this.rigs[i % 2].add(mesh)
    })
  }

  private buildBoats(): void {
    const hulls = ['#e2574c', '#3b82c4', '#f2b134'].map((c) => new THREE.Color(c))
    const sail = new THREE.Color('#fffaf0')
    const mast = new THREE.Color('#6b4a32')
    const mat = new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true })
    const n = new THREE.Vector3()
    const probe = new THREE.Vector3()
    for (let tries = 0; tries < 400 && this.boats.length < 9; tries++) {
      n.set(this.rnd() * 2 - 1, this.rnd() * 2 - 1, this.rnd() * 2 - 1).normalize()
      const radius = 7 + this.rnd() * 7
      const ref = Math.abs(n.y) < 0.9 ? new THREE.Vector3(0, 1, 0) : new THREE.Vector3(1, 0, 0)
      const u = new THREE.Vector3().crossVectors(ref, n).normalize()
      const w = new THREE.Vector3().crossVectors(n, u)
      // the whole circle has to be open water
      let open = this.world.sample(n).e < -1.6
      for (let k = 0; k < 8 && open; k++) {
        const a = (k / 8) * 6.283
        probe.copy(n).addScaledVector(u, (Math.cos(a) * (radius + 3)) / R).addScaledVector(w, (Math.sin(a) * (radius + 3)) / R).normalize()
        open = this.world.sample(probe).e < -1.6
      }
      if (!open) continue
      const hull = hulls[this.boats.length % hulls.length]
      const geo = mergeGeometries(
        [
          coloured(new THREE.CylinderGeometry(0.5, 0.3, 2.2, 5).rotateX(Math.PI / 2).scale(1, 0.55, 1).translate(0, 0.12, 0), () => hull),
          coloured(new THREE.CylinderGeometry(0.05, 0.05, 2.3, 4).translate(0, 1.3, 0.1), () => mast),
          coloured(new THREE.ConeGeometry(0.85, 2.0, 3).scale(0.1, 1, 1).translate(0, 1.45, -0.42), () => sail),
          coloured(new THREE.ConeGeometry(0.5, 1.4, 3).scale(0.1, 1, 1).translate(0, 1.2, 0.62), () => sail)
        ],
        false
      )
      geo.computeVertexNormals()
      const mesh = new THREE.Mesh(geo, mat)
      mesh.matrixAutoUpdate = false
      this.group.add(mesh)
      this.boats.push({ mesh, centre: n.clone(), u, w, radius, speed: (0.05 + this.rnd() * 0.05) * (this.rnd() < 0.5 ? -1 : 1), phase: this.rnd() * 6.283 })
    }
  }

  setScale(scale: number): void {
    const mat = this.fireflies.material as THREE.ShaderMaterial
    mat.uniforms.uScale.value = scale
    mat.uniforms.uMax.value = scale * 0.03
  }

  update(dt: number): void {
    this.time += dt
    this.timeUniform.value = this.time

    for (const rig of this.rigs) rig.rotateOnAxis(rig.userData.axis as THREE.Vector3, (rig.userData.speed as number) * dt)

    // butterflies circle the fields that are open
    let dirty = false
    for (let i = 0; i < this.flutters.length; i++) {
      const fl = this.flutters[i]
      const p = this.world.patches[fl.field]
      if (p.t < 0.8) continue
      const grow = clamp((p.t - 0.8) / 1.2, 0, 1)
      const a = this.time * fl.speed + fl.phase
      const rr = (fl.radius * (0.8 + 0.2 * Math.sin(this.time * 0.6 + fl.phase))) / R
      this.n.copy(p.n).addScaledVector(this.fieldU[fl.field], Math.cos(a) * rr).addScaledVector(this.fieldW[fl.field], Math.sin(a) * rr).normalize()
      this.world.place(this.n, p.ground + fl.height + Math.sin(this.time * 1.6 + fl.phase) * 0.5, a * (fl.speed > 0 ? 1 : -1), grow, this.m)
      this.butterflies.setMatrixAt(i, this.m)
      dirty = true
    }
    if (dirty) this.butterflies.instanceMatrix.needsUpdate = true

    // boats sail round and round, bobbing
    for (const b of this.boats) {
      const a = this.time * b.speed + b.phase
      const rr = b.radius / R
      this.n.copy(b.centre).addScaledVector(b.u, Math.cos(a) * rr).addScaledVector(b.w, Math.sin(a) * rr).normalize()
      this.f.copy(b.u).multiplyScalar(-Math.sin(a)).addScaledVector(b.w, Math.cos(a)).multiplyScalar(Math.sign(b.speed))
      this.f.addScaledVector(this.n, -this.f.dot(this.n)).normalize()
      this.x.crossVectors(this.n, this.f).normalize()
      // lean a little with the swell
      this.x.addScaledVector(this.n, Math.sin(this.time * 1.4 + b.phase) * 0.08).normalize()
      this.m.makeBasis(this.x, new THREE.Vector3().crossVectors(this.f, this.x), this.f)
      const h = R + 0.12 + Math.sin(this.time * 1.9 + b.phase) * 0.1
      this.m.scale(new THREE.Vector3(1.25, 1.25, 1.25))
      this.m.setPosition(this.n.x * h, this.n.y * h, this.n.z * h)
      b.mesh.matrix.copy(this.m)
    }
  }
}
