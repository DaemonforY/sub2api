// The bee and the glowing dust for 蜂巢星球.
import * as THREE from 'three'

/** A pool of soft additive points: pollen trail, bloom bursts, fireworks. */
export class Sparks {
  readonly points: THREE.Points
  private readonly pos: Float32Array
  private readonly col: Float32Array
  private readonly size: Float32Array
  private readonly vel: Float32Array
  private readonly life: Float32Array
  private readonly span: Float32Array
  private readonly base: Float32Array
  private next = 0
  private readonly material: THREE.ShaderMaterial

  constructor(private readonly count = 900) {
    this.pos = new Float32Array(count * 3)
    this.col = new Float32Array(count * 3)
    this.size = new Float32Array(count)
    this.vel = new Float32Array(count * 3)
    this.life = new Float32Array(count)
    this.span = new Float32Array(count)
    this.base = new Float32Array(count)
    const geo = new THREE.BufferGeometry()
    geo.setAttribute('position', new THREE.BufferAttribute(this.pos, 3).setUsage(THREE.DynamicDrawUsage))
    geo.setAttribute('aColor', new THREE.BufferAttribute(this.col, 3).setUsage(THREE.DynamicDrawUsage))
    geo.setAttribute('aSize', new THREE.BufferAttribute(this.size, 1).setUsage(THREE.DynamicDrawUsage))
    this.material = new THREE.ShaderMaterial({
      uniforms: { uScale: { value: 600 }, uMax: { value: 40 } },
      vertexShader:
        'attribute vec3 aColor; attribute float aSize; uniform float uScale; uniform float uMax; varying vec3 vColor; void main(){ vColor = aColor; vec4 mv = modelViewMatrix * vec4(position, 1.0); gl_PointSize = min(uMax, aSize * uScale / max(0.1, -mv.z)); gl_Position = projectionMatrix * mv; }',
      fragmentShader:
        'varying vec3 vColor; void main(){ float d = length(gl_PointCoord - 0.5); float a = smoothstep(0.5, 0.0, d); a *= a; gl_FragColor = vec4(vColor * a, a); }',
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })
    this.points = new THREE.Points(geo, this.material)
    this.points.frustumCulled = false
  }

  /** `scale` = canvas height in device pixels, so points keep their size on every screen. */
  setScale(scale: number): void {
    this.material.uniforms.uScale.value = scale
    this.material.uniforms.uMax.value = scale * 0.045
  }

  emit(p: THREE.Vector3, v: THREE.Vector3, color: THREE.Color, size: number, life: number): void {
    const i = this.next
    this.next = (this.next + 1) % this.count
    this.pos.set([p.x, p.y, p.z], i * 3)
    this.vel.set([v.x, v.y, v.z], i * 3)
    this.col.set([color.r, color.g, color.b], i * 3)
    this.life[i] = life
    this.span[i] = life
    this.base[i] = size
  }

  update(dt: number): void {
    for (let i = 0; i < this.count; i++) {
      if (this.life[i] <= 0) {
        this.size[i] = 0
        continue
      }
      this.life[i] -= dt
      const k = Math.max(0, this.life[i] / this.span[i])
      this.pos[i * 3] += this.vel[i * 3] * dt
      this.pos[i * 3 + 1] += this.vel[i * 3 + 1] * dt
      this.pos[i * 3 + 2] += this.vel[i * 3 + 2] * dt
      const drag = 1 - 1.4 * dt
      this.vel[i * 3] *= drag
      this.vel[i * 3 + 1] *= drag
      this.vel[i * 3 + 2] *= drag
      this.size[i] = this.base[i] * Math.sin(Math.min(1, k * 1.15) * Math.PI * 0.5)
    }
    const g = this.points.geometry
    g.attributes.position.needsUpdate = true
    g.attributes.aSize.needsUpdate = true
    g.attributes.aColor.needsUpdate = true
  }

  dispose(): void {
    this.points.geometry.dispose()
    this.material.dispose()
  }
}

export interface Bee {
  /** placed and turned by the game */
  root: THREE.Group
  /** rolls, pitches and bobs inside the root */
  model: THREE.Group
  wings: THREE.Object3D[]
}

/** Forward is +Z, up is +Y. */
export function makeBee(): Bee {
  const root = new THREE.Group()
  const model = new THREE.Group()
  root.add(model)
  const lambert = (color: string): THREE.MeshLambertMaterial => new THREE.MeshLambertMaterial({ color, flatShading: true })

  // striped body
  // rings run round the body: a sphere with its poles on the flight axis, one colour per band
  const bodyGeo = new THREE.SphereGeometry(0.62, 18, 11).toNonIndexed()
  bodyGeo.rotateX(Math.PI / 2)
  const bp = bodyGeo.attributes.position
  const colors = new Float32Array(bp.count * 3)
  const yellow = new THREE.Color('#ffc52e')
  const dark = new THREE.Color('#3b2a20')
  for (let i = 0; i < bp.count; i += 3) {
    const z = (bp.getZ(i) + bp.getZ(i + 1) + bp.getZ(i + 2)) / 3
    const band = Math.floor((z + 0.62) / (1.24 / 11) + 0.001)
    const c = band === 3 || band === 5 || band === 7 ? dark : yellow
    for (let k = 0; k < 3; k++) colors.set([c.r, c.g, c.b], (i + k) * 3)
  }
  bodyGeo.setAttribute('color', new THREE.BufferAttribute(colors, 3))
  bodyGeo.scale(0.98, 0.92, 1.3)
  bodyGeo.computeVertexNormals()
  model.add(new THREE.Mesh(bodyGeo, new THREE.MeshLambertMaterial({ vertexColors: true })))

  const head = new THREE.Mesh(new THREE.IcosahedronGeometry(0.44, 1), lambert('#4a3526'))
  head.position.set(0, 0.06, 0.92)
  model.add(head)
  const white = new THREE.MeshBasicMaterial({ color: '#ffffff' })
  const black = new THREE.MeshBasicMaterial({ color: '#20140e' })
  for (const sx of [-1, 1]) {
    const eye = new THREE.Mesh(new THREE.SphereGeometry(0.17, 8, 6), white)
    eye.position.set(sx * 0.2, 0.16, 1.24)
    model.add(eye)
    const pupil = new THREE.Mesh(new THREE.SphereGeometry(0.085, 6, 5), black)
    pupil.position.set(sx * 0.2, 0.17, 1.37)
    model.add(pupil)
    const cheek = new THREE.Mesh(new THREE.SphereGeometry(0.09, 6, 4), new THREE.MeshBasicMaterial({ color: '#ff9aa8' }))
    cheek.position.set(sx * 0.33, -0.02, 1.18)
    cheek.scale.set(1, 0.6, 0.5)
    model.add(cheek)
    const feeler = new THREE.Mesh(new THREE.CylinderGeometry(0.025, 0.025, 0.5, 4), lambert('#3b2a20'))
    feeler.position.set(sx * 0.16, 0.58, 1.02)
    feeler.rotation.set(0.5, 0, -sx * 0.35)
    model.add(feeler)
    const tip = new THREE.Mesh(new THREE.SphereGeometry(0.07, 6, 5), lambert('#ffc52e'))
    tip.position.set(sx * 0.25, 0.8, 1.14)
    model.add(tip)
  }
  const sting = new THREE.Mesh(new THREE.ConeGeometry(0.09, 0.3, 5), lambert('#3b2a20'))
  sting.rotation.x = -Math.PI / 2
  sting.position.set(0, 0, -0.92)
  model.add(sting)

  const wingMat = new THREE.MeshBasicMaterial({ color: '#f4fbff', transparent: true, opacity: 0.72, side: THREE.DoubleSide, depthWrite: false })
  const wings: THREE.Object3D[] = []
  for (const sx of [-1, 1]) {
    for (const back of [0, 1]) {
      const pivot = new THREE.Group()
      pivot.position.set(sx * 0.22, 0.46, 0.18 - back * 0.34)
      const wing = new THREE.Mesh(new THREE.CircleGeometry(0.5, 9), wingMat)
      wing.rotation.x = -Math.PI / 2
      wing.scale.set(1.75 - back * 0.5, 0.86 - back * 0.2, 1)
      wing.position.x = sx * (0.82 - back * 0.22)
      pivot.add(wing)
      pivot.userData.side = sx
      pivot.userData.back = back
      model.add(pivot)
      wings.push(pivot)
    }
  }
  model.scale.setScalar(1.35)
  return { root, model, wings }
}
