// The planet of 蜂巢星球: six islands on a small sphere, asleep (grey) until the bee wakes their
// flower fields. Built once from a fixed seed; no model files, every shape is made here.
import * as THREE from 'three'
import { mergeGeometries } from 'three/examples/jsm/utils/BufferGeometryUtils.js'
import { clamp, easeOutBack, easeOutCubic, makeNoise, mulberry32, smoothstep } from './noise'

export const R = 100
export const PATCHES_PER_REGION = 6
const FLOWERS_PER_PATCH = 40
const PATCH_RADIUS = 5.2
/** how far round an open flower field the ground wakes, in units */
const FIELD_REACH = 10

export type TreeKind = 'round' | 'pine' | 'palm'

export interface RegionDef {
  name: string
  dir: [number, number, number]
  /** angular radius in radians */
  rad: number
  low: string
  high: string
  peak: string
  trees: string[]
  kind: TreeKind
  flowers: string[]
}

export const REGIONS: RegionDef[] = [
  { name: '蜂巢岛', dir: [0, 0.45, 0.9], rad: 0.5, low: '#9fd660', high: '#4f9e45', peak: '#b9c98a', trees: ['#3f9d4b', '#58b55a', '#2f8a50'], kind: 'round', flowers: ['#ffd84a', '#ffffff', '#ff9a3c'] },
  { name: '樱花岛', dir: [0.9, 0.3, 0.1], rad: 0.48, low: '#b4e07f', high: '#6bb36a', peak: '#ecd6de', trees: ['#ffb1cc', '#ff94bb', '#ffd0e0'], kind: 'round', flowers: ['#ff7fb0', '#fff0f5', '#ff5a94'] },
  { name: '枫叶岛', dir: [0.2, -0.75, 0.6], rad: 0.48, low: '#dccb6c', high: '#b98a3e', peak: '#c9a27a', trees: ['#e8642c', '#f2a03a', '#c9402a'], kind: 'round', flowers: ['#ffb347', '#ff6b3d', '#ffe08a'] },
  { name: '雪松岛', dir: [-0.85, 0.1, 0.45], rad: 0.46, low: '#e6f0f6', high: '#c0d8e8', peak: '#ffffff', trees: ['#3d7f78', '#5aa39a', '#2f6a72'], kind: 'pine', flowers: ['#8fd3ff', '#ffffff', '#b6a8ff'] },
  { name: '沙丘岛', dir: [-0.3, 0.6, -0.75], rad: 0.48, low: '#f3d88e', high: '#e0a85a', peak: '#c98a52', trees: ['#5fae6a', '#7cc47a'], kind: 'palm', flowers: ['#ff5f6d', '#ffc371', '#ff8fb1'] },
  { name: '薰衣草岛', dir: [0.3, -0.5, -0.8], rad: 0.5, low: '#a4d481', high: '#6aa870', peak: '#b7a9d6', trees: ['#4a9c6a', '#3f8f7a'], kind: 'pine', flowers: ['#a47bff', '#c9a8ff', '#7f6bff'] }
]

/** Sea routes between neighbouring islands, drawn as arcs of pollen. */
const BRIDGES: [number, number][] = [[0, 3], [0, 1], [0, 2], [1, 5], [3, 4], [4, 5], [2, 5], [1, 2]]

export interface Sample {
  e: number
  region: number
  d: number
}

export interface Patch {
  region: number
  index: number
  n: THREE.Vector3
  ground: number
  bloomed: boolean
  /** seconds since it started to bloom, -1 while closed */
  t: number
  origin: THREE.Vector3
  first: number
  marker: THREE.Group
  ring: THREE.Mesh
}

export interface Orb {
  n: THREE.Vector3
  alt: number
  taken: boolean
  phase: number
  /** where it grows; a magnet may pull it away before it is taken */
  homeN: THREE.Vector3
  homeAlt: number
  /** world time at which a taken grain grows back */
  back: number
}

interface Landmark {
  group: THREE.Group
  petals: THREE.Object3D[]
  glow: THREE.Mesh
  beam: THREE.Mesh
  parts: THREE.MeshLambertMaterial[]
  alive: THREE.Color[]
  t: number
}

const UP = new THREE.Vector3(0, 1, 0)
const RIGHT = new THREE.Vector3(1, 0, 0)
const SLEEP = new THREE.Color('#8f90a8')

/** Shared by every material that can be asleep or awake. */
export const wakeUniforms = {
  uFront: { value: [0, 0, 0, 0, 0, 0] },
  uLife: { value: [0, 0, 0, 0, 0, 0] },
  /** how far the colour has spread round each flower field, 0..1.2 */
  uField: { value: new Array<number>(REGIONS.length * PATCHES_PER_REGION).fill(0) }
}

/** The sun's direction, shared by everything that glows at night. Set by the World. */
export const sunUniform = { value: new THREE.Vector3(0.5, 0.35, 0.8).normalize() }

/** `fields`: the ground also wakes in a circle round each open flower field. */
function patchWake(mat: THREE.Material, fields = false, extra?: (sh: THREE.WebGLProgramParametersWithUniforms) => void): void {
  mat.onBeforeCompile = (sh) => {
    sh.uniforms.uFront = wakeUniforms.uFront
    sh.uniforms.uLife = wakeUniforms.uLife
    if (fields) sh.uniforms.uField = wakeUniforms.uField
    sh.vertexShader = sh.vertexShader
      .replace('#include <common>', `#include <common>\nattribute vec2 aWake;\nvarying vec2 vWake;${fields ? '\nattribute vec2 aField;\nvarying vec2 vField;' : ''}`)
      .replace('#include <begin_vertex>', `#include <begin_vertex>\nvWake = aWake;${fields ? '\nvField = aField;' : ''}`)
    sh.fragmentShader = sh.fragmentShader
      .replace(
        '#include <common>',
        `#include <common>\nvarying vec2 vWake;\nuniform float uFront[6];\nuniform float uLife[6];${fields ? '\nvarying vec2 vField;\nuniform float uField[36];' : ''}`
      )
      .replace(
        '#include <color_fragment>',
        `#include <color_fragment>
        {
          int ri = int(vWake.x + 0.5);
          float fr = uFront[ri];
          float wk = 1.0 - smoothstep(fr - 0.16, fr, vWake.y);
          ${fields ? 'float ff = uField[int(vField.x + 0.5)]; wk = max(wk, (1.0 - smoothstep(ff - 0.3, ff, vField.y)) * step(0.001, ff));' : ''}
          float lum = dot(diffuseColor.rgb, vec3(0.3, 0.59, 0.11));
          vec3 asleep = mix((vec3(lum) * 0.5 + 0.3) * vec3(0.8, 0.83, 1.0), diffuseColor.rgb, 0.1 + uLife[ri]);
          diffuseColor.rgb = mix(asleep, diffuseColor.rgb, wk);
          float ring = smoothstep(0.075, 0.0, abs(vWake.y - fr + 0.04)) * step(0.01, fr) * (1.0 - step(1.75, fr));
          diffuseColor.rgb += vec3(1.0, 0.8, 0.35) * ring * 0.9;
        }`
      )
    extra?.(sh)
  }
}

function flag(geo: THREE.BufferGeometry, name: string, value: number): THREE.BufferGeometry {
  geo.setAttribute(name, new THREE.BufferAttribute(new Float32Array(geo.attributes.position.count).fill(value), 1))
  return geo
}

function tint(geo: THREE.BufferGeometry, color: THREE.ColorRepresentation): THREE.BufferGeometry {
  const g = geo.index ? geo.toNonIndexed() : geo
  g.deleteAttribute('uv')
  const c = new THREE.Color(color)
  const n = g.attributes.position.count
  const arr = new Float32Array(n * 3)
  for (let i = 0; i < n; i++) {
    arr[i * 3] = c.r
    arr[i * 3 + 1] = c.g
    arr[i * 3 + 2] = c.b
  }
  g.setAttribute('color', new THREE.BufferAttribute(arr, 3))
  return g
}

function merged(parts: THREE.BufferGeometry[]): THREE.BufferGeometry {
  const g = mergeGeometries(parts, false)
  g.computeVertexNormals()
  return g
}

function treeGeometry(kind: TreeKind): THREE.BufferGeometry {
  const trunk = '#7a5a40'
  if (kind === 'pine') {
    return merged([
      tint(new THREE.CylinderGeometry(0.18, 0.28, 1.2, 5).translate(0, 0.6, 0), trunk),
      tint(new THREE.ConeGeometry(1.3, 1.9, 6).translate(0, 1.9, 0), '#ffffff'),
      tint(new THREE.ConeGeometry(1.0, 1.6, 6).translate(0, 2.95, 0), '#f2f2f2'),
      tint(new THREE.ConeGeometry(0.62, 1.3, 6).translate(0, 3.9, 0), '#ffffff')
    ])
  }
  if (kind === 'palm') {
    const parts = [tint(new THREE.CylinderGeometry(0.13, 0.22, 3.2, 5).translate(0, 1.6, 0), '#8a6a48')]
    for (let i = 0; i < 6; i++) {
      const leaf = new THREE.ConeGeometry(0.34, 2.1, 4)
      leaf.translate(0, 1.05, 0)
      leaf.rotateZ(-1.95)
      leaf.rotateY((i / 6) * Math.PI * 2)
      leaf.translate(0, 3.2, 0)
      parts.push(tint(leaf, i % 2 ? '#ffffff' : '#e6e6e6'))
    }
    return merged(parts)
  }
  return merged([
    tint(new THREE.CylinderGeometry(0.2, 0.32, 1.7, 5).translate(0, 0.85, 0), trunk),
    tint(new THREE.IcosahedronGeometry(1.5, 0).scale(1, 0.92, 1).translate(0, 2.75, 0), '#ffffff'),
    tint(new THREE.IcosahedronGeometry(0.95, 0).translate(0.95, 2.1, 0.35), '#ececec'),
    tint(new THREE.IcosahedronGeometry(0.8, 0).translate(-0.8, 2.25, -0.5), '#f6f6f6')
  ])
}

function flowerGeometry(): THREE.BufferGeometry {
  const parts = [
    tint(new THREE.CylinderGeometry(0.05, 0.08, 0.8, 4).translate(0, 0.4, 0), '#4f9a4a'),
    tint(new THREE.IcosahedronGeometry(0.24, 0).translate(0, 0.92, 0), '#fff6b0')
  ]
  for (let i = 0; i < 5; i++) {
    const a = (i / 5) * Math.PI * 2
    parts.push(tint(new THREE.OctahedronGeometry(0.4, 0).scale(1, 0.4, 1).translate(Math.cos(a) * 0.46, 0.84, Math.sin(a) * 0.46), '#ffffff'))
  }
  return merged(parts)
}

export function beamMaterial(color: THREE.ColorRepresentation, strength: number, additive = true): THREE.ShaderMaterial {
  return new THREE.ShaderMaterial({
    uniforms: { uColor: { value: new THREE.Color(color) }, uOpacity: { value: strength }, uTime: { value: 0 } },
    vertexShader: 'varying vec2 vUv; void main(){ vUv = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }',
    fragmentShader:
      'uniform vec3 uColor; uniform float uOpacity; uniform float uTime; varying vec2 vUv; void main(){ float a = pow(1.0 - vUv.y, 1.4) * uOpacity * (0.82 + 0.18 * sin(uTime * 2.2 + vUv.y * 9.0)); gl_FragColor = vec4(uColor * (' + (additive ? 'a' : '1.0') + '), a); }',
    transparent: true,
    depthWrite: false,
    side: THREE.DoubleSide,
    blending: additive ? THREE.AdditiveBlending : THREE.NormalBlending
  })
}

export class World {
  readonly group = new THREE.Group()
  readonly centers: THREE.Vector3[] = REGIONS.map((r) => new THREE.Vector3(...r.dir).normalize())
  readonly patches: Patch[] = []
  readonly orbs: Orb[] = []
  readonly awake: boolean[] = REGIONS.map(() => false)
  /** seconds since each island started waking, -1 while asleep */
  readonly wakeT: number[] = REGIONS.map(() => -1)
  readonly sun = sunUniform.value
  /** where the houses stand, so trees keep off them */
  private readonly houseSpots: THREE.Vector3[] = []
  readonly sunLight = new THREE.DirectionalLight('#fff1dc', 2.0)

  private readonly noise = makeNoise(20261113)
  private readonly rnd = mulberry32(977)
  private readonly tmp: Sample = { e: 0, region: 0, d: 0 }
  private readonly v = new THREE.Vector3()
  private readonly m = new THREE.Matrix4()
  private readonly q = new THREE.Quaternion()
  private readonly s = new THREE.Vector3()

  private flowers!: THREE.InstancedMesh
  private readonly fPos: THREE.Vector3[] = []
  private readonly fQuat: THREE.Quaternion[] = []
  private readonly fScale: number[] = []
  private readonly fColor: THREE.Color[] = []
  private readonly fDelay: number[] = []
  private orbMesh!: THREE.InstancedMesh
  private water!: THREE.Mesh
  private waterUniform = { value: 0 }
  private sky!: THREE.Mesh
  private stars!: THREE.Points
  private rim!: THREE.Mesh
  private markerMat = beamMaterial('#ffe9a0', 0.42, false)
  private ringMat = new THREE.MeshBasicMaterial({ color: '#ffd23f', toneMapped: false })
  private readonly landmarks: Landmark[] = []
  private readonly cloudRigs: THREE.Group[] = []
  private readonly waterSleep = new THREE.Color('#4a97b0')
  private readonly waterAlive = new THREE.Color('#1fc4e2')
  private time = 0

  constructor() {
    this.pickPatches()
    this.buildTerrain()
    this.buildWater()
    this.buildFlowers()
    this.buildLandmarks()
    this.buildHouses()
    this.buildTrees()
    this.buildRocks()
    this.buildOrbs()
    this.buildClouds()
    this.buildSky()
    this.group.add(this.sunLight)
    this.group.add(new THREE.HemisphereLight('#d6e4ff', '#8a7fa6', 1.5))
  }

  /** Height above sea level, nearest island and how far into it (0 centre, ~0.75 shore). */
  sample(v: THREE.Vector3, out: Sample = this.tmp): Sample {
    let best = 9
    let bi = 0
    for (let i = 0; i < this.centers.length; i++) {
      const d = Math.acos(clamp(v.dot(this.centers[i]), -1, 1)) / REGIONS[i].rad
      if (d < best) {
        best = d
        bi = i
      }
    }
    const n = this.noise
    const coast = (n(v.x * 2.3 + 7, v.y * 2.3, v.z * 2.3) * 0.6 + n(v.x * 5.1, v.y * 5.1 + 3, v.z * 5.1) * 0.4) * 0.2
    const mask = smoothstep(1.05, 0.45, best + coast)
    const hills = n(v.x * 3.1, v.y * 3.1, v.z * 3.1 + 11) * 0.5 + 0.5
    const ridge = 1 - Math.abs(n(v.x * 1.7 + 19, v.y * 1.7, v.z * 1.7))
    const inner = smoothstep(0.1, 0.36, best)
    let e = -2.7 + mask * (5.3 + 7.5 * hills * ridge * mask * inner)
    e += n(v.x * 7, v.y * 7, v.z * 7) * 0.4 * mask
    out.e = e
    out.region = bi
    out.d = best
    return out
  }

  /** Seconds the planet has been running. */
  get now(): number {
    return this.time
  }

  /** Ground or water height under a point. */
  ground(v: THREE.Vector3): number {
    return Math.max(0, this.sample(v).e)
  }

  /** What the bee has to clear: the ground, plus the big thing in the middle of each island. */
  clearance(v: THREE.Vector3): number {
    const sm = this.sample(v)
    const d = sm.d * REGIONS[sm.region].rad * R
    return Math.max(0, sm.e) + (sm.region === 0 ? 11 : 13) * (1 - smoothstep(4, 10, d))
  }

  place(n: THREE.Vector3, height: number, yaw: number, scale: number, out: THREE.Matrix4): THREE.Matrix4 {
    const ref = Math.abs(n.y) < 0.9 ? UP : RIGHT
    const x = new THREE.Vector3().crossVectors(ref, n).normalize()
    x.applyAxisAngle(n, yaw)
    const z = new THREE.Vector3().crossVectors(x, n)
    out.makeBasis(x, n, z)
    out.scale(this.s.set(scale, scale, scale))
    out.setPosition(n.x * (R + height), n.y * (R + height), n.z * (R + height))
    return out
  }

  /** A point `dist` radians from an island centre, `angle` around it. */
  around(region: number, angle: number, dist: number, out: THREE.Vector3): THREE.Vector3 {
    const c = this.centers[region]
    const ref = Math.abs(c.y) < 0.9 ? UP : RIGHT
    const u = new THREE.Vector3().crossVectors(ref, c).normalize()
    const w = new THREE.Vector3().crossVectors(c, u)
    const dir = u.multiplyScalar(Math.cos(angle)).addScaledVector(w, Math.sin(angle))
    return out.copy(c).multiplyScalar(Math.cos(dist)).addScaledVector(dir, Math.sin(dist)).normalize()
  }

  private buildTerrain(): void {
    const geo = new THREE.IcosahedronGeometry(1, 44)
    geo.deleteAttribute('uv')
    const pos = geo.attributes.position as THREE.BufferAttribute
    const count = pos.count
    const colors = new Float32Array(count * 3)
    const wake = new Float32Array(count * 2)
    const field = new Float32Array(count * 2)
    const reach = FIELD_REACH / R
    const p = [new THREE.Vector3(), new THREE.Vector3(), new THREE.Vector3()]
    const cen = new THREE.Vector3()
    const col = new THREE.Color()
    const sand = new THREE.Color('#f0e2ae')
    const seabed = new THREE.Color('#4f9aa0')
    const pal = REGIONS.map((r) => ({ low: new THREE.Color(r.low), high: new THREE.Color(r.high), peak: new THREE.Color(r.peak) }))
    for (let i = 0; i < count; i += 3) {
      cen.set(0, 0, 0)
      for (let k = 0; k < 3; k++) {
        p[k].fromBufferAttribute(pos, i + k).normalize()
        cen.add(p[k])
      }
      cen.normalize()
      const sc = this.sample(cen)
      const e = sc.e
      const region = sc.region
      if (e < 0) col.copy(sand).lerp(seabed, smoothstep(0, -2.4, e))
      else if (e < 0.7) col.copy(sand)
      else {
        col.copy(pal[region].low).lerp(pal[region].high, smoothstep(0.7, 6.2, e))
        if (e > 7) col.lerp(pal[region].peak, smoothstep(7, 9, e))
      }
      col.multiplyScalar(0.93 + this.rnd() * 0.14)
      for (let k = 0; k < 3; k++) {
        const h = this.sample(p[k]).e
        pos.setXYZ(i + k, p[k].x * (R + h), p[k].y * (R + h), p[k].z * (R + h))
        colors[(i + k) * 3] = col.r
        colors[(i + k) * 3 + 1] = col.g
        colors[(i + k) * 3 + 2] = col.b
        wake[(i + k) * 2] = region
        wake[(i + k) * 2 + 1] = Math.acos(clamp(p[k].dot(this.centers[region]), -1, 1)) / REGIONS[region].rad
        // the nearest flower field of this island, if one is close
        let fi = 0
        let fd = 9
        for (let q = region * PATCHES_PER_REGION; q < (region + 1) * PATCHES_PER_REGION; q++) {
          const d = Math.acos(clamp(p[k].dot(this.patches[q].n), -1, 1)) / reach
          if (d < fd) {
            fd = d
            fi = q
          }
        }
        field[(i + k) * 2] = fi
        field[(i + k) * 2 + 1] = fd
      }
    }
    geo.setAttribute('color', new THREE.BufferAttribute(colors, 3))
    geo.setAttribute('aWake', new THREE.BufferAttribute(wake, 2))
    geo.setAttribute('aField', new THREE.BufferAttribute(field, 2))
    geo.computeVertexNormals()
    const mat = new THREE.MeshLambertMaterial({ vertexColors: true })
    patchWake(mat, true)
    this.group.add(new THREE.Mesh(geo, mat))
  }

  private buildWater(): void {
    const mat = new THREE.MeshPhongMaterial({
      color: this.waterSleep,
      transparent: true,
      opacity: 0.8,
      shininess: 110,
      specular: new THREE.Color('#ffffff'),
      flatShading: true
    })
    const time = this.waterUniform
    mat.onBeforeCompile = (sh) => {
      sh.uniforms.uTime = time
      sh.vertexShader = sh.vertexShader
        .replace('#include <common>', '#include <common>\nuniform float uTime;\nattribute float aShore;\nvarying float vShore;')
        .replace(
          '#include <begin_vertex>',
          '#include <begin_vertex>\nvShore = aShore;\ntransformed += normalize(position) * (sin(position.x * 0.31 + uTime * 1.3) + sin(position.y * 0.37 + uTime * 1.1) + sin(position.z * 0.29 + uTime * 0.9)) * 0.14;'
        )
      // shallows turn pale, and a band of foam breathes in and out along every shore
      sh.fragmentShader = sh.fragmentShader
        .replace('#include <common>', '#include <common>\nuniform float uTime;\nvarying float vShore;')
        .replace(
          '#include <color_fragment>',
          `#include <color_fragment>
          {
            float shallow = smoothstep(-2.3, -0.5, vShore);
            diffuseColor.rgb = mix(diffuseColor.rgb, diffuseColor.rgb * 0.6 + vec3(0.22, 0.46, 0.42), shallow * 0.7);
            float edge = vShore + sin(uTime * 1.1 + vShore * 5.0) * 0.16;
            float foam = smoothstep(-0.95, -0.55, edge) * (1.0 - smoothstep(-0.28, -0.1, edge)) * 0.55 + smoothstep(-0.34, -0.16, edge);
            diffuseColor.rgb = mix(diffuseColor.rgb, vec3(1.0), clamp(foam, 0.0, 1.0) * 0.85);
            diffuseColor.a = mix(diffuseColor.a, 0.97, clamp(foam, 0.0, 1.0));
          }`
        )
    }
    const geo = new THREE.IcosahedronGeometry(R, 50)
    const wp = geo.attributes.position
    const shore = new Float32Array(wp.count)
    const v = new THREE.Vector3()
    for (let i = 0; i < wp.count; i++) shore[i] = clamp(this.sample(v.fromBufferAttribute(wp, i).normalize()).e, -3, 1)
    geo.setAttribute('aShore', new THREE.BufferAttribute(shore, 1))
    this.water = new THREE.Mesh(geo, mat)
    this.group.add(this.water)
  }

  /** Choose where the flower fields go. Done first: the ground is coloured round them. */
  private pickPatches(): void {
    const columnGeo = new THREE.CylinderGeometry(2.6, 1.6, 15, 14, 1, true).translate(0, 7.5, 0)
    const ringGeo = new THREE.TorusGeometry(3.3, 0.2, 6, 28).rotateX(Math.PI / 2)
    const n = new THREE.Vector3()
    for (let r = 0; r < REGIONS.length; r++) {
      for (let k = 0; k < PATCHES_PER_REGION; k++) {
        // look for a gentle piece of land on this side of the island
        let found = false
        for (let tries = 0; tries < 40 && !found; tries++) {
          const ang = (k / PATCHES_PER_REGION) * Math.PI * 2 + (this.rnd() - 0.5) * 0.5
          const dist = (0.26 + this.rnd() * 0.32) * REGIONS[r].rad
          this.around(r, ang, dist, n)
          const e = this.sample(n).e
          found = e > 1.1 && e < 5.5
        }
        const pn = n.clone()
        const marker = new THREE.Group()
        const ground = this.ground(pn)
        this.place(pn, ground, 0, 1, marker.matrix)
        marker.matrixAutoUpdate = false
        marker.add(new THREE.Mesh(columnGeo, this.markerMat))
        const ring = new THREE.Mesh(ringGeo, this.ringMat)
        ring.position.y = 6
        marker.add(ring)
        this.group.add(marker)
        const first = this.patches.length * FLOWERS_PER_PATCH
        this.patches.push({ region: r, index: k, n: pn, ground, bloomed: false, t: -1, origin: pn.clone(), first, marker, ring })
      }
    }
  }

  private buildFlowers(): void {
    const geo = flowerGeometry()
    const total = this.patches.length * FLOWERS_PER_PATCH
    const mat = new THREE.MeshLambertMaterial({ vertexColors: true })
    // open flowers glow on the night side
    mat.onBeforeCompile = (sh) => {
      sh.uniforms.uSun = sunUniform
      sh.vertexShader = sh.vertexShader
        .replace('#include <common>', '#include <common>\nuniform vec3 uSun;\nvarying float vNight;')
        .replace('#include <begin_vertex>', '#include <begin_vertex>\nvNight = smoothstep(0.1, -0.22, dot(normalize((instanceMatrix * vec4(0.0, 0.0, 0.0, 1.0)).xyz), uSun));')
      sh.fragmentShader = sh.fragmentShader
        .replace('#include <common>', '#include <common>\nvarying float vNight;')
        .replace(
          '#include <emissivemap_fragment>',
          '#include <emissivemap_fragment>\n{ vec3 c = diffuseColor.rgb; float sat = max(max(c.r, c.g), c.b) - min(min(c.r, c.g), c.b); totalEmissiveRadiance += c * vNight * sat * 1.5; }'
        )
    }
    const mesh = new THREE.InstancedMesh(geo, mat, total)
    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage)
    let idx = 0
    for (let r = 0; r < REGIONS.length; r++) {
      const cols = REGIONS[r].flowers.map((c) => new THREE.Color(c))
      for (let k = 0; k < PATCHES_PER_REGION; k++) {
        const pn = this.patches[r * PATCHES_PER_REGION + k].n
        const ref = Math.abs(pn.y) < 0.9 ? UP : RIGHT
        const u = new THREE.Vector3().crossVectors(ref, pn).normalize()
        const w = new THREE.Vector3().crossVectors(pn, u)
        for (let f = 0; f < FLOWERS_PER_PATCH; f++) {
          const a = this.rnd() * Math.PI * 2
          const rr = (Math.sqrt(this.rnd()) * PATCH_RADIUS) / R
          const fn = pn.clone().addScaledVector(u, Math.cos(a) * rr).addScaledVector(w, Math.sin(a) * rr).normalize()
          const fe = Math.max(0.05, this.sample(fn).e)
          this.place(fn, fe - 0.05, this.rnd() * 6.28, 1, this.m)
          const fp = new THREE.Vector3()
          const fq = new THREE.Quaternion()
          this.m.decompose(fp, fq, this.s)
          this.fPos.push(fp)
          this.fQuat.push(fq)
          this.fScale.push(0.95 + this.rnd() * 0.6)
          this.fColor.push(cols[Math.floor(this.rnd() * cols.length)].clone())
          this.fDelay.push(0)
          this.writeFlower(mesh, idx, 0)
          idx++
        }
      }
    }
    mesh.instanceMatrix.needsUpdate = true
    if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true
    mesh.frustumCulled = false
    this.flowers = mesh
    this.group.add(mesh)
  }

  /** `k` 0 = closed bud, 1 = open. */
  private writeFlower(mesh: THREE.InstancedMesh, i: number, k: number): void {
    const open = k <= 0 ? 0 : easeOutBack(k)
    const sc = this.fScale[i] * (0.34 + 0.66 * open)
    this.m.compose(this.fPos[i], this.fQuat[i], this.s.set(sc, sc * (0.7 + 0.3 * open), sc))
    mesh.setMatrixAt(i, this.m)
    const c = this.v
    const t = clamp(k * 1.6, 0, 1)
    c.set(
      SLEEP.r + (this.fColor[i].r - SLEEP.r) * t,
      SLEEP.g + (this.fColor[i].g - SLEEP.g) * t,
      SLEEP.b + (this.fColor[i].b - SLEEP.b) * t
    )
    mesh.setColorAt(i, new THREE.Color(c.x, c.y, c.z))
  }

  private buildLandmarks(): void {
    for (let r = 0; r < REGIONS.length; r++) {
      const c = this.centers[r]
      const ground = this.ground(c)
      const group = new THREE.Group()
      this.place(c, ground - 0.3, r * 1.3, 1.25, group.matrix)
      group.matrixAutoUpdate = false
      const parts: THREE.MeshLambertMaterial[] = []
      const alive: THREE.Color[] = []
      const part = (geo: THREE.BufferGeometry, color: string, parent: THREE.Object3D = group): THREE.Mesh => {
        const mat = new THREE.MeshLambertMaterial({ color: SLEEP.clone(), flatShading: true })
        parts.push(mat)
        alive.push(new THREE.Color(color))
        const mesh = new THREE.Mesh(geo, mat)
        parent.add(mesh)
        return mesh
      }
      const petals: THREE.Object3D[] = []
      const main = REGIONS[r].flowers[0]
      let top = 6
      if (r === 0) {
        // the hive: a stump with a ringed dome on it
        part(new THREE.CylinderGeometry(2.3, 3.0, 2.6, 9).translate(0, 1.3, 0), '#8a6444')
        const rings = [2.9, 3.35, 3.3, 2.9, 2.25, 1.4]
        rings.forEach((rad, i) => {
          part(new THREE.SphereGeometry(rad, 12, 6).scale(1, 0.42, 1).translate(0, 3.3 + i * 1.12, 0), i % 2 ? '#f4b63a' : '#ffcf5a')
        })
        part(new THREE.CircleGeometry(0.85, 10).translate(0, 4.6, 3.32), '#3a2a1c')
        top = 10.5
      } else {
        // a giant bud that opens when the island wakes
        part(new THREE.CylinderGeometry(0.6, 1.0, 5.2, 7).translate(0, 2.6, 0), '#5aa84f')
        part(new THREE.OctahedronGeometry(1.5, 0).scale(1.6, 0.25, 0.7).rotateZ(0.5).translate(1.6, 1.6, 0), '#6fbf58')
        part(new THREE.OctahedronGeometry(1.5, 0).scale(1.6, 0.25, 0.7).rotateZ(-0.5).translate(-1.6, 2.3, 0), '#6fbf58')
        for (let i = 0; i < 7; i++) {
          const yaw = new THREE.Group()
          yaw.position.y = 5.2
          yaw.rotation.y = (i / 7) * Math.PI * 2
          const pivot = new THREE.Group()
          pivot.rotation.x = 0.1
          yaw.add(pivot)
          group.add(yaw)
          part(new THREE.SphereGeometry(1, 6, 5).scale(1.25, 3.6, 0.3).translate(0, 3.6, 0), i % 2 ? main : REGIONS[r].flowers[2] ?? main, pivot)
          petals.push(pivot)
        }
        top = 5.4
      }
      const glow = new THREE.Mesh(
        new THREE.IcosahedronGeometry(r === 0 ? 0.8 : 1.1, 1),
        new THREE.MeshBasicMaterial({ color: '#fff0a0', transparent: true, opacity: 0, toneMapped: false })
      )
      glow.position.y = top + (r === 0 ? 0.6 : 0.4)
      group.add(glow)
      const beam = new THREE.Mesh(new THREE.CylinderGeometry(1.5, 0.45, 240, 16, 1, true).translate(0, 120 + top, 0), beamMaterial(main, 0))
      beam.visible = false
      group.add(beam)
      this.group.add(group)
      this.landmarks.push({ group, petals, glow, beam, parts, alive, t: -1 })
    }
  }

  /** A few cottages on every island. Their windows light up at night once the island is awake. */
  private buildHouses(): void {
    const walls = merged([
      flag(tint(new THREE.BoxGeometry(1.5, 1.1, 1.3).translate(0, 0.55, 0), '#f7ecd6'), 'aGlow', 0),
      flag(tint(new THREE.BoxGeometry(0.34, 0.62, 0.06).translate(0.36, 0.31, 0.66), '#8a5a3a'), 'aGlow', 0),
      flag(tint(new THREE.BoxGeometry(0.36, 0.36, 0.06).translate(-0.34, 0.62, 0.66), '#ffe2a0'), 'aGlow', 1),
      flag(tint(new THREE.BoxGeometry(0.06, 0.36, 0.36).translate(0.76, 0.62, 0), '#ffe2a0'), 'aGlow', 1),
      flag(tint(new THREE.BoxGeometry(0.06, 0.36, 0.36).translate(-0.76, 0.62, 0.1), '#ffe2a0'), 'aGlow', 1)
    ])
    const roofs = merged([
      tint(new THREE.ConeGeometry(1.32, 0.95, 4).rotateY(Math.PI / 4).scale(1, 1, 0.9).translate(0, 1.57, 0), '#ffffff'),
      tint(new THREE.BoxGeometry(0.26, 0.6, 0.26).translate(-0.45, 1.75, -0.2), '#d8d8d8')
    ])
    const roofCols = ['#e2574c', '#d9698c', '#b5522d', '#5c7fa8', '#c2703d', '#7a62b8'].map((c) => new THREE.Color(c))
    const mats: THREE.Matrix4[] = []
    const cols: THREE.Color[] = []
    const wake: number[] = []
    const n = new THREE.Vector3()
    for (let r = 0; r < REGIONS.length; r++) {
      const hub = new THREE.Vector3()
      // a village needs a gentle spot away from the flower fields
      for (let tries = 0; tries < 60; tries++) {
        this.around(r, this.rnd() * 6.283, (0.3 + this.rnd() * 0.22) * REGIONS[r].rad, hub)
        const e = this.sample(hub).e
        if (e > 1.2 && e < 4.5 && !this.nearPatch(hub, PATCH_RADIUS + 7)) break
      }
      const ref = Math.abs(hub.y) < 0.9 ? UP : RIGHT
      const u = new THREE.Vector3().crossVectors(ref, hub).normalize()
      const w = new THREE.Vector3().crossVectors(hub, u)
      let made = 0
      for (let tries = 0; tries < 200 && made < 7; tries++) {
        const a = this.rnd() * 6.283
        const rr = (1.5 + this.rnd() * 6) / R
        n.copy(hub).addScaledVector(u, Math.cos(a) * rr).addScaledVector(w, Math.sin(a) * rr).normalize()
        const sm = this.sample(n)
        if (sm.e < 1 || sm.e > 6 || sm.region !== r || this.nearPatch(n, PATCH_RADIUS + 1.5) || this.nearHouse(n, 2.7)) continue
        if (sm.d * REGIONS[r].rad * R < 12) continue
        this.houseSpots.push(n.clone())
        mats.push(this.place(n, sm.e - 0.12, this.rnd() * 6.283, 1.05 + this.rnd() * 0.45, new THREE.Matrix4()))
        cols.push(roofCols[r].clone().multiplyScalar(0.85 + this.rnd() * 0.3))
        wake.push(r, sm.d)
        made++
      }
    }
    const wallMat = new THREE.MeshLambertMaterial({ vertexColors: true })
    patchWake(wallMat, false, (sh) => {
      sh.uniforms.uSun = sunUniform
      sh.vertexShader = sh.vertexShader
        .replace('#include <common>', '#include <common>\nuniform vec3 uSun;\nattribute float aGlow;\nvarying float vGlow;')
        .replace('#include <begin_vertex>', '#include <begin_vertex>\nvGlow = aGlow * smoothstep(0.1, -0.2, dot(normalize((instanceMatrix * vec4(0.0, 0.0, 0.0, 1.0)).xyz), uSun));')
      sh.fragmentShader = sh.fragmentShader
        .replace('#include <common>', '#include <common>\nvarying float vGlow;')
        .replace(
          '#include <emissivemap_fragment>',
          '#include <emissivemap_fragment>\n{ float f2 = uFront[int(vWake.x + 0.5)]; totalEmissiveRadiance += vec3(1.0, 0.7, 0.28) * vGlow * (1.0 - smoothstep(f2 - 0.16, f2, vWake.y)) * 2.2; }'
        )
    })
    const roofMat = new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true })
    patchWake(roofMat)
    for (const [geo, mat, tinted] of [[walls, wallMat, false], [roofs, roofMat, true]] as const) {
      geo.setAttribute('aWake', new THREE.InstancedBufferAttribute(new Float32Array(wake), 2))
      const mesh = new THREE.InstancedMesh(geo, mat, mats.length)
      mats.forEach((mm, k) => {
        mesh.setMatrixAt(k, mm)
        if (tinted) mesh.setColorAt(k, cols[k])
      })
      mesh.frustumCulled = false
      this.group.add(mesh)
    }
  }

  private nearHouse(n: THREE.Vector3, units: number): boolean {
    const lim = Math.cos(units / R)
    for (const h of this.houseSpots) if (h.dot(n) > lim) return true
    return false
  }

  private nearPatch(n: THREE.Vector3, units: number): boolean {
    const lim = Math.cos(units / R)
    for (const p of this.patches) if (p.n.dot(n) > lim) return true
    return false
  }

  private buildTrees(): void {
    const kinds: TreeKind[] = ['round', 'pine', 'palm']
    const data = kinds.map(() => ({ mats: [] as THREE.Matrix4[], cols: [] as THREE.Color[], wake: [] as number[] }))
    const n = new THREE.Vector3()
    for (let r = 0; r < REGIONS.length; r++) {
      const def = REGIONS[r]
      const bucket = data[kinds.indexOf(def.kind)]
      const cols = def.trees.map((c) => new THREE.Color(c))
      const want = Math.round(330 * (def.rad / 0.5) ** 2 * (def.kind === 'palm' ? 0.45 : 1))
      let made = 0
      for (let tries = 0; tries < want * 12 && made < want; tries++) {
        const dist = Math.sqrt(this.rnd()) * 0.82 * def.rad
        this.around(r, this.rnd() * 6.283, dist, n)
        const sm = this.sample(n)
        if (sm.e < 0.9 || sm.e > 8.2 || sm.region !== r) continue
        // clumps: keep trees where a slow noise is high
        const clump = this.noise(n.x * 4.2 + 31, n.y * 4.2, n.z * 4.2)
        if (clump < -0.15 && def.kind !== 'palm') continue
        if (dist * R < 11 || this.nearPatch(n, PATCH_RADIUS + 2.2) || this.nearHouse(n, 2.4)) continue
        const sc = 0.55 + this.rnd() * 0.5
        bucket.mats.push(this.place(n, sm.e - 0.15, this.rnd() * 6.283, sc, new THREE.Matrix4()))
        bucket.cols.push(cols[Math.floor(this.rnd() * cols.length)].clone().multiplyScalar(0.9 + this.rnd() * 0.2))
        bucket.wake.push(r, sm.d)
        made++
      }
    }
    kinds.forEach((kind, i) => {
      const b = data[i]
      if (!b.mats.length) return
      const geo = treeGeometry(kind)
      geo.setAttribute('aWake', new THREE.InstancedBufferAttribute(new Float32Array(b.wake), 2))
      const mat = new THREE.MeshLambertMaterial({ vertexColors: true })
      patchWake(mat)
      const mesh = new THREE.InstancedMesh(geo, mat, b.mats.length)
      b.mats.forEach((mm, k) => {
        mesh.setMatrixAt(k, mm)
        mesh.setColorAt(k, b.cols[k])
      })
      mesh.frustumCulled = false
      this.group.add(mesh)
    })
  }

  private buildRocks(): void {
    const geo = tint(new THREE.DodecahedronGeometry(1, 0), '#ffffff')
    geo.computeVertexNormals()
    const count = 150
    const wake = new Float32Array(count * 2)
    const mat = new THREE.MeshLambertMaterial({ vertexColors: true, flatShading: true })
    patchWake(mat)
    const mesh = new THREE.InstancedMesh(geo, mat, count)
    const n = new THREE.Vector3()
    const grey = new THREE.Color()
    let made = 0
    for (let tries = 0; tries < 6000 && made < count; tries++) {
      n.set(this.rnd() * 2 - 1, this.rnd() * 2 - 1, this.rnd() * 2 - 1)
      if (n.lengthSq() < 0.05 || n.lengthSq() > 1) continue
      n.normalize()
      const sm = this.sample(n)
      // shore rocks and a few on the heights
      if (!((sm.e > -1.2 && sm.e < 0.9) || sm.e > 6.5)) continue
      if (this.nearPatch(n, PATCH_RADIUS + 1)) continue
      const sc = 0.35 + this.rnd() * 0.8
      this.place(n, sm.e - 0.2, this.rnd() * 6.283, sc, this.m)
      this.m.scale(this.s.set(1, 0.6 + this.rnd() * 0.5, 1))
      mesh.setMatrixAt(made, this.m)
      mesh.setColorAt(made, grey.set('#a9a7b3').multiplyScalar(0.8 + this.rnd() * 0.35))
      wake[made * 2] = sm.region
      wake[made * 2 + 1] = sm.d
      made++
    }
    mesh.count = made
    geo.setAttribute('aWake', new THREE.InstancedBufferAttribute(wake, 2))
    mesh.frustumCulled = false
    this.group.add(mesh)
  }

  private arc(a: THREE.Vector3, b: THREE.Vector3, t: number, out: THREE.Vector3): THREE.Vector3 {
    const om = Math.acos(clamp(a.dot(b), -1, 1))
    const so = Math.sin(om) || 1
    return out.copy(a).multiplyScalar(Math.sin((1 - t) * om) / so).addScaledVector(b, Math.sin(t * om) / so).normalize()
  }

  private buildOrbs(): void {
    const n = new THREE.Vector3()
    const add = (alt: number): void => {
      this.orbs.push({ n: n.clone(), alt, taken: false, phase: this.rnd() * 6.283, homeN: n.clone(), homeAlt: alt, back: 0 })
    }
    for (let r = 0; r < REGIONS.length; r++) {
      const ps = this.patches.filter((p) => p.region === r)
      for (let k = 0; k < ps.length; k++) {
        const a = ps[k].n
        const b = ps[(k + 1) % ps.length].n
        const len = Math.acos(clamp(a.dot(b), -1, 1)) * R
        const steps = Math.max(2, Math.round(len / 6))
        for (let i = 1; i < steps; i++) {
          this.arc(a, b, i / steps, n)
          add(this.ground(n) + 3.4)
        }
      }
    }
    for (const [i, j] of BRIDGES) {
      const a = this.centers[i]
      const b = this.centers[j]
      const len = Math.acos(clamp(a.dot(b), -1, 1)) * R
      const steps = Math.round(len / 6.5)
      for (let k = 1; k < steps; k++) {
        const t = k / steps
        this.arc(a, b, t, n)
        const sm = this.sample(n)
        if (sm.d < 0.62) continue
        add(Math.max(0, sm.e) + 4 + Math.sin(t * Math.PI) * 6)
      }
    }
    const mat = new THREE.MeshBasicMaterial({ color: '#ffe27a', toneMapped: false })
    const mesh = new THREE.InstancedMesh(new THREE.IcosahedronGeometry(0.4, 1), mat, this.orbs.length)
    mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage)
    mesh.frustumCulled = false
    this.orbMesh = mesh
    this.group.add(mesh)
  }

  private buildClouds(): void {
    const mat = new THREE.MeshLambertMaterial({ color: '#ffffff', emissive: '#93a4c8', emissiveIntensity: 0.55, flatShading: true })
    const variants: THREE.BufferGeometry[] = []
    for (let v = 0; v < 3; v++) {
      const parts: THREE.BufferGeometry[] = []
      const blobs = 4 + v
      for (let i = 0; i < blobs; i++) {
        const rad = 1.6 + this.rnd() * 1.9
        parts.push(
          new THREE.IcosahedronGeometry(rad, 1)
            .scale(1.25, 0.72, 1)
            .translate((i - blobs / 2) * 2.3 + this.rnd(), this.rnd() * 0.9, (this.rnd() - 0.5) * 2.6)
        )
      }
      variants.push(merged(parts.map((p) => tint(p, '#ffffff'))))
    }
    for (let g = 0; g < 2; g++) {
      const rig = new THREE.Group()
      rig.userData.axis = new THREE.Vector3(g ? 0.3 : -0.2, 1, g ? -0.4 : 0.5).normalize()
      rig.userData.speed = g ? 0.012 : -0.008
      this.cloudRigs.push(rig)
      this.group.add(rig)
    }
    const n = new THREE.Vector3()
    for (let i = 0; i < 30; i++) {
      n.set(this.rnd() * 2 - 1, this.rnd() * 2 - 1, this.rnd() * 2 - 1).normalize()
      const mesh = new THREE.Mesh(variants[i % 3], mat)
      this.place(n, 27 + this.rnd() * 14, this.rnd() * 6.283, 0.8 + this.rnd() * 0.9, mesh.matrix)
      mesh.matrixAutoUpdate = false
      this.cloudRigs[i % 2].add(mesh)
    }
  }

  private buildSky(): void {
    this.sky = new THREE.Mesh(
      new THREE.SphereGeometry(1800, 24, 16),
      new THREE.ShaderMaterial({
        uniforms: { uUp: { value: new THREE.Vector3(0, 1, 0) }, uSun: { value: this.sun }, uSpace: { value: 1 } },
        vertexShader: 'varying vec3 vDir; void main(){ vDir = position; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }',
        fragmentShader: `
          uniform vec3 uUp; uniform vec3 uSun; uniform float uSpace; varying vec3 vDir;
          void main(){
            vec3 d = normalize(vDir);
            float h = dot(d, uUp);
            float day = smoothstep(-0.3, 0.35, dot(uUp, uSun));
            float t = smoothstep(-0.5, 0.85, h);
            vec3 dayC = mix(vec3(0.62, 0.84, 1.0), vec3(0.13, 0.36, 0.86), t);
            vec3 nightC = mix(vec3(0.13, 0.10, 0.32), vec3(0.012, 0.016, 0.06), t);
            vec3 col = mix(nightC, dayC, day);
            float dusk = 1.0 - abs(day * 2.0 - 1.0);
            float sd = max(dot(d, uSun), 0.0);
            col = mix(col, vec3(1.0, 0.5, 0.3), dusk * (1.0 - t) * (0.3 + 0.7 * pow(sd, 3.0)));
            col += vec3(1.0, 0.86, 0.62) * (pow(sd, 420.0) * 2.0 + pow(sd, 14.0) * 0.2) * (1.0 - uSpace * 0.6);
            vec3 space = vec3(0.016, 0.018, 0.06) + vec3(0.10, 0.05, 0.18) * pow(max(0.0, d.y * 0.5 + 0.5), 2.0) + vec3(0.02, 0.07, 0.10) * pow(max(0.0, 0.5 - d.y * 0.5), 3.0);
            gl_FragColor = vec4(mix(col, space, uSpace), 1.0);
            #include <colorspace_fragment>
          }`,
        side: THREE.BackSide,
        depthWrite: false
      })
    )
    this.sky.renderOrder = -10
    this.sky.frustumCulled = false
    this.group.add(this.sky)

    const count = 1600
    const pos = new Float32Array(count * 3)
    const col = new Float32Array(count * 3)
    const v = new THREE.Vector3()
    for (let i = 0; i < count; i++) {
      v.set(this.rnd() * 2 - 1, this.rnd() * 2 - 1, this.rnd() * 2 - 1).normalize().multiplyScalar(1500)
      pos.set([v.x, v.y, v.z], i * 3)
      const b = 0.45 + this.rnd() * 0.55
      const warm = this.rnd()
      col.set([b, b * (0.86 + warm * 0.14), b * (1.1 - warm * 0.3)], i * 3)
    }
    const sg = new THREE.BufferGeometry()
    sg.setAttribute('position', new THREE.BufferAttribute(pos, 3))
    sg.setAttribute('color', new THREE.BufferAttribute(col, 3))
    this.stars = new THREE.Points(
      sg,
      new THREE.PointsMaterial({ size: 2.2, sizeAttenuation: false, vertexColors: true, transparent: true, depthWrite: false })
    )
    this.stars.frustumCulled = false
    this.stars.renderOrder = -9
    this.group.add(this.stars)

    this.rim = new THREE.Mesh(
      new THREE.SphereGeometry(R * 1.17, 48, 32),
      new THREE.ShaderMaterial({
        uniforms: { uStrength: { value: 1 } },
        vertexShader:
          'varying vec3 vN; varying vec3 vV; void main(){ vec4 mv = modelViewMatrix * vec4(position, 1.0); vN = normalize(normalMatrix * normal); vV = normalize(-mv.xyz); gl_Position = projectionMatrix * mv; }',
        fragmentShader:
          'uniform float uStrength; varying vec3 vN; varying vec3 vV; void main(){ float i = pow(clamp(-dot(normalize(vN), normalize(vV)) * 1.95, 0.0, 1.0), 2.6) * uStrength; gl_FragColor = vec4(vec3(0.42, 0.72, 1.0) * i, i); }',
        side: THREE.BackSide,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })
    )
    this.group.add(this.rim)
  }

  /** Open a flower field; the bloom spreads out from where the bee came in. */
  bloom(p: Patch, from: THREE.Vector3): void {
    if (p.bloomed) return
    p.bloomed = true
    p.t = 0
    p.origin.copy(from)
    p.marker.children[0].visible = false
    for (let i = 0; i < FLOWERS_PER_PATCH; i++) {
      const fi = p.first + i
      const d = this.v.copy(this.fPos[fi]).normalize().angleTo(from) * R
      this.fDelay[fi] = Math.min(d, 14) * 0.055
    }
    const done = this.patches.filter((q) => q.region === p.region && q.bloomed).length
    wakeUniforms.uLife.value[p.region] = (done / PATCHES_PER_REGION) * 0.42
  }

  wake(region: number): void {
    if (this.awake[region]) return
    this.awake[region] = true
    this.wakeT[region] = 0
    this.landmarks[region].t = 0
    this.landmarks[region].beam.visible = true
  }

  /** Put the planet back the way a returning player left it, with no animation. */
  restore(fields: boolean[]): void {
    this.patches.forEach((p, i) => {
      if (!fields[i] || p.bloomed) return
      p.bloomed = true
      p.t = 3.5
      p.marker.visible = false
      wakeUniforms.uField.value[i] = 1.2
      for (let f = 0; f < FLOWERS_PER_PATCH; f++) this.writeFlower(this.flowers, p.first + f, 2)
    })
    this.flowers.instanceMatrix.needsUpdate = true
    if (this.flowers.instanceColor) this.flowers.instanceColor.needsUpdate = true
    for (let r = 0; r < REGIONS.length; r++) {
      const done = this.bloomedIn(r)
      wakeUniforms.uLife.value[r] = (done / PATCHES_PER_REGION) * 0.42
      if (done < PATCHES_PER_REGION) continue
      this.awake[r] = true
      this.wakeT[r] = 30
      this.landmarks[r].t = 30
      this.landmarks[r].beam.visible = true
    }
  }

  bloomedIn(region: number): number {
    let c = 0
    for (const p of this.patches) if (p.region === region && p.bloomed) c++
    return c
  }

  progress(): number {
    let c = 0
    for (const p of this.patches) if (p.bloomed) c++
    return c / this.patches.length
  }

  /** `space` 1 = seen from orbit (title, ending), 0 = flying low. */
  update(dt: number, cam: THREE.Camera, up: THREE.Vector3, space: number): void {
    this.time += dt
    this.waterUniform.value = this.time
    this.markerMat.uniforms.uTime.value = this.time

    // the sun goes round in about seven minutes
    const a = 1.0 + this.time * ((Math.PI * 2) / 420)
    this.sun.set(Math.cos(a), 0.35, Math.sin(a)).normalize()
    this.sunLight.position.copy(this.sun).multiplyScalar(400)

    const skyMat = this.sky.material as THREE.ShaderMaterial
    skyMat.uniforms.uUp.value.copy(up)
    skyMat.uniforms.uSpace.value = space
    this.sky.position.copy(cam.position)
    const night = 1 - smoothstep(-0.3, 0.3, up.dot(this.sun))
    ;(this.stars.material as THREE.PointsMaterial).opacity = Math.max(night, space)
    this.stars.position.copy(cam.position)
    this.rim.visible = space > 0.02
    ;(this.rim.material as THREE.ShaderMaterial).uniforms.uStrength.value = space

    ;(this.water.material as THREE.MeshPhongMaterial).color.copy(this.waterSleep).lerp(this.waterAlive, easeOutCubic(this.progress()))

    for (const rig of this.cloudRigs) rig.rotateOnAxis(rig.userData.axis as THREE.Vector3, (rig.userData.speed as number) * dt)

    // flower fields opening
    let dirty = false
    for (const p of this.patches) {
      if (p.t < 0) {
        p.ring.position.y = 6 + Math.sin(this.time * 1.7 + p.first) * 0.5
        p.ring.rotation.y += dt * 0.8
        continue
      }
      if (p.t > 3) continue
      p.t += dt
      wakeUniforms.uField.value[p.region * PATCHES_PER_REGION + p.index] = Math.min(1.2, easeOutCubic(p.t / 1.6) * 1.2)
      const burst = clamp(p.t / 0.45, 0, 1)
      p.ring.scale.setScalar(1 + easeOutCubic(burst) * 0.9)
      p.ring.scale.y = 1 - burst
      p.marker.visible = burst < 1
      for (let i = 0; i < FLOWERS_PER_PATCH; i++) {
        const fi = p.first + i
        this.writeFlower(this.flowers, fi, (p.t - this.fDelay[fi]) / 0.55)
      }
      dirty = true
    }
    if (dirty) {
      this.flowers.instanceMatrix.needsUpdate = true
      if (this.flowers.instanceColor) this.flowers.instanceColor.needsUpdate = true
    }

    // islands waking: the colour front runs out from the centre
    for (let r = 0; r < REGIONS.length; r++) {
      if (this.wakeT[r] < 0) continue
      this.wakeT[r] += dt
      wakeUniforms.uFront.value[r] = Math.min(1.8, easeOutCubic(this.wakeT[r] / 4.2) * 1.8)
      const lm = this.landmarks[r]
      lm.t += dt
      const k = easeOutBack(lm.t / 2.6)
      for (const pivot of lm.petals) pivot.rotation.x = 0.1 + 1.12 * k
      const c = clamp(lm.t / 1.2, 0, 1)
      lm.parts.forEach((m, i) => m.color.copy(SLEEP).lerp(lm.alive[i], c))
      ;(lm.glow.material as THREE.MeshBasicMaterial).opacity = c
      lm.glow.scale.setScalar(1 + Math.sin(this.time * 3) * 0.08)
      const bm = lm.beam.material as THREE.ShaderMaterial
      bm.uniforms.uOpacity.value = 0.34 * clamp(lm.t / 1.5, 0, 1)
      bm.uniforms.uTime.value = this.time
    }

    // pollen bobbing; a grain that was taken grows back after a while
    for (let i = 0; i < this.orbs.length; i++) {
      const o = this.orbs[i]
      if (o.taken && this.time >= o.back) {
        o.taken = false
        o.n.copy(o.homeN)
        o.alt = o.homeAlt
      }
      const sc = o.taken ? 0 : 1 + Math.sin(this.time * 3 + o.phase) * 0.14
      const h = R + o.alt + Math.sin(this.time * 1.8 + o.phase) * 0.3
      this.m.makeScale(sc, sc, sc)
      this.m.setPosition(o.n.x * h, o.n.y * h, o.n.z * h)
      this.orbMesh.setMatrixAt(i, this.m)
    }
    this.orbMesh.instanceMatrix.needsUpdate = true
    this.q.identity()
  }

  dispose(): void {
    this.group.traverse((o) => {
      const mesh = o as THREE.Mesh
      if (mesh.geometry) mesh.geometry.dispose()
      const mat = mesh.material as THREE.Material | THREE.Material[] | undefined
      if (Array.isArray(mat)) mat.forEach((x) => x.dispose())
      else if (mat) mat.dispose()
    })
    wakeUniforms.uFront.value.fill(0)
    wakeUniforms.uLife.value.fill(0)
    wakeUniforms.uField.value.fill(0)
  }
}
