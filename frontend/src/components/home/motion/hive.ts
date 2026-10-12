// 数据在流动: a honeycomb floor receding into the dark. Each request is a pulse — cyan for chat,
// amber for a picture, blue for code — that lifts the cells it passes and lights their rims.
import { Color, CylinderGeometry, InstancedBufferAttribute, InstancedMesh, Matrix4, PerspectiveCamera, Scene, ShaderMaterial, Vector2, Vector4 } from 'three'
import { COLORS, rand, type MotionOptions, type SceneHandle } from './stage'

const PULSES = 8

const VS = /* glsl */ `
attribute vec3 aCell; // x, z, random
uniform float uTime, uLift;
uniform vec4 uPulse[${PULSES}]; // x, z, start time, kind
uniform vec3 uK0, uK1, uK2;
varying float vGlow, vTop, vDepth, vBreath;
varying vec3 vGlowCol;
varying vec2 vLocal;
vec3 kindColor(float k) { return k < 0.5 ? uK0 : (k < 1.5 ? uK1 : uK2); }
void main() {
  float glow = 0.; vec3 gc = vec3(0.);
  for (int i = 0; i < ${PULSES}; i++) {
    vec4 P = uPulse[i];
    float age = uTime - P.z;
    if (age < 0. || age > 5.) continue;
    float d = distance(aCell.xy, P.xy);
    float ring = exp(-pow((d - age * 6.5) / 1.5, 2.)) * (1. - age / 5.);
    float hit = exp(-d * d * 0.6) * exp(-age * 1.5) * 1.6;
    float k = ring * 0.75 + hit;
    glow += k; gc += k * kindColor(P.w);
  }
  float br = 0.5 + 0.5 * sin(length(aCell.xy) * 0.22 - uTime * 0.7);
  vec3 pos = position;
  pos.y *= 0.1 + 0.05 * br + aCell.z * 0.07 + glow * 0.9 * uLift;
  vec4 mv = viewMatrix * modelMatrix * instanceMatrix * vec4(pos, 1.);
  gl_Position = projectionMatrix * mv;
  vGlow = glow; vGlowCol = gc / max(glow, 1e-3); vTop = normal.y; vLocal = position.xz; vDepth = -mv.z; vBreath = br;
}`

const FS = /* glsl */ `
uniform vec3 uRim;
uniform vec2 uFog;
varying float vGlow, vTop, vDepth, vBreath;
varying vec3 vGlowCol;
varying vec2 vLocal;
void main() {
  // distance to the hexagon's edge (pointy along z), 1 at the rim
  float hd = max(abs(vLocal.x), abs(vLocal.x) * 0.5 + abs(vLocal.y) * 0.8660254) / (0.8660254 * 0.92);
  float top = step(0.5, vTop);
  vec3 col = mix(vec3(0.02, 0.045, 0.09), vec3(0.04, 0.1, 0.19), top);
  float rim = smoothstep(0.8, 0.96, hd) * top;
  float g = clamp(vGlow, 0., 1.2);
  col += mix(uRim, vGlowCol, min(g, 1.)) * rim * (0.32 + 0.3 * vBreath + g * 1.6);
  col += vGlowCol * g * (top * 0.5 + (1. - top) * 0.3);
  float a = 1. - smoothstep(uFog.x, uFog.y, vDepth);
  gl_FragColor = vec4(col * a, a);
}`

export function hiveScene(o: MotionOptions): SceneHandle {
  const scene = new Scene()
  const camera = new PerspectiveCamera(34, 1, 0.1, 140)
  const r = rand(11)
  const k = o.density * (o.tier === 'mobile' ? 0.45 : 1)
  const cols = Math.round(46 * Math.sqrt(k)), rows = Math.round(34 * Math.sqrt(k))
  const dx = Math.sqrt(3), dz = 1.5
  const count = cols * rows

  const geo = new CylinderGeometry(0.92, 0.92, 1, 6, 1)
  geo.translate(0, 0.5, 0) // stand on y = 0, grow upward
  const cell = new Float32Array(count * 3)
  const uniforms = {
    uTime: { value: 0 },
    uLift: { value: 1 },
    uPulse: { value: Array.from({ length: PULSES }, () => new Vector4(0, 0, -100, 0)) },
    uK0: { value: COLORS.cyan }, uK1: { value: COLORS.amber }, uK2: { value: new Color('#4d8dff') },
    uRim: { value: new Color('#2f7fd8') },
    uFog: { value: new Vector2(14, 50) },
  }
  const mesh = new InstancedMesh(geo, new ShaderMaterial({ vertexShader: VS, fragmentShader: FS, uniforms }), count)
  const m = new Matrix4()
  let i = 0
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const x = (col - cols / 2) * dx + ((row % 2) * dx) / 2
      const z = 10 - row * dz
      m.makeTranslation(x, 0, z)
      mesh.setMatrixAt(i, m)
      cell[i * 3] = x
      cell[i * 3 + 1] = z
      cell[i * 3 + 2] = r()
      i++
    }
  }
  geo.setAttribute('aCell', new InstancedBufferAttribute(cell, 3))
  mesh.frustumCulled = false
  scene.add(mesh)

  // Requests arrive as pulses: about half chat, a third pictures, the rest code.
  let next = 0, slot = 0
  function pulse(t: number) {
    const roll = r()
    uniforms.uPulse.value[slot].set((r() - 0.5) * 34, 8 - r() * 30, t, roll < 0.5 ? 0 : roll < 0.82 ? 1 : 2)
    slot = (slot + 1) % PULSES
  }
  let lookZ = -5

  return {
    scene, camera,
    resize(w, h) {
      camera.aspect = w / h
      const portrait = w / h < 0.8 // phones: back off and widen so the cells aren't giant
      camera.fov = portrait ? 50 : 34
      if (portrait) camera.position.set(0, 15, 24)
      else camera.position.set(0, 10, 16)
      lookZ = portrait ? -6 : -5
      camera.lookAt(0, 0, lookZ)
      camera.updateProjectionMatrix()
    },
    frame(t) {
      uniforms.uTime.value = t
      if (t >= next) { pulse(t); next = t + 0.45 + r() * 0.7 }
      const sway = Math.sin(t * 0.06) * 2.4
      camera.position.x = sway
      camera.lookAt(sway * 0.4, 0, lookZ)
    },
    onEnter() { for (let j = 0; j < 3; j++) pulse(uniforms.uTime.value - j * 0.6) }, // alive the moment it shows
    settle() {
      uniforms.uTime.value = 12
      for (let j = 0; j < PULSES; j++) pulse(12 - j * 0.55)
    },
  }
}
