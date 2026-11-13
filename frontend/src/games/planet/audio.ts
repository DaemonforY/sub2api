// Sound for 蜂巢星球, all made in the browser. A slow eight-bar tune loops from the first second;
// it starts as a music box over a soft pad and gains one instrument for each island that wakes
// (arpeggio, bass, shaker, bells, sparkle, a second voice). Pollen grains play their own notes in
// order, flower fields open with a rising run, and wind and wings follow the flight.

const SCALE = [60, 62, 64, 67, 69, 72, 74, 76, 79, 81, 84, 86, 88]
const TUNE = [0, 2, 4, 5, 4, 2, 3, 5, 7, 5, 4, 2, 1, 3, 4, 6, 5, 3, 2, 0, 2, 4, 7, 9, 7, 5, 4, 5, 7, 4, 2, 0]

/** 80 beats a minute, sixteen steps to the bar. */
const STEP = 60 / 80 / 4

interface Chord {
  bass: number
  tones: number[]
}
const C: Chord = { bass: 48, tones: [60, 64, 67, 72] }
const AM: Chord = { bass: 45, tones: [57, 60, 64, 69] }
const F: Chord = { bass: 41, tones: [57, 60, 65, 69] }
const G: Chord = { bass: 43, tones: [55, 57, 62, 67] }
const BARS: Chord[] = [C, C, AM, AM, F, F, G, G]
const ARP = [0, 1, 2, 3, 2, 1, 2, 1]
/** the bell melody: for each bar, [step, note] pairs */
const MELODY: [number, number][][] = [
  [[0, 76], [4, 79], [8, 81], [12, 79]],
  [[0, 76], [6, 74], [8, 72]],
  [[0, 72], [4, 76], [8, 81], [12, 84]],
  [[0, 81], [8, 79]],
  [[0, 81], [4, 79], [8, 76], [12, 72]],
  [[0, 69], [4, 72], [8, 76]],
  [[0, 74], [4, 79], [8, 81], [12, 79]],
  [[0, 74], [8, 67]]
]

function hz(midi: number): number {
  return 440 * Math.pow(2, (midi - 69) / 12)
}

export class PlanetAudio {
  private ctx?: AudioContext
  private master?: GainNode
  private wet?: GainNode
  private windGain?: GainNode
  private windFilter?: BiquadFilterNode
  private buzzGain?: GainNode
  private buzz?: OscillatorNode
  private noise?: AudioBuffer
  private muted = false
  private step = 0
  private lastNote = 0
  /** how many islands are awake: decides which instruments play */
  private level = 0
  private timer = 0
  private seqStep = 0
  private seqTime = 0

  start(): void {
    if (this.ctx) {
      void this.ctx.resume()
      return
    }
    const Ctor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctor) return
    const ctx = new Ctor()
    this.ctx = ctx
    const comp = ctx.createDynamicsCompressor()
    comp.threshold.value = -16
    comp.ratio.value = 4
    comp.connect(ctx.destination)
    const master = ctx.createGain()
    master.gain.value = this.muted ? 0 : 0.9
    master.connect(comp)
    this.master = master

    // a hall made from decaying noise
    const len = Math.floor(ctx.sampleRate * 3.2)
    const impulse = ctx.createBuffer(2, len, ctx.sampleRate)
    for (let ch = 0; ch < 2; ch++) {
      const d = impulse.getChannelData(ch)
      for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 2.6)
    }
    const hall = ctx.createConvolver()
    hall.buffer = impulse
    const wet = ctx.createGain()
    wet.gain.value = 0.5
    wet.connect(hall)
    hall.connect(master)
    this.wet = wet

    // one second of white noise, reused for the shaker and the breath of a bloom
    const white = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate)
    const wd = white.getChannelData(0)
    for (let i = 0; i < wd.length; i++) wd[i] = Math.random() * 2 - 1
    this.noise = white

    // wind: looped soft noise through a band that opens with speed
    const nb = ctx.createBuffer(1, ctx.sampleRate * 2, ctx.sampleRate)
    const nd = nb.getChannelData(0)
    let last = 0
    for (let i = 0; i < nd.length; i++) {
      last = last * 0.96 + (Math.random() * 2 - 1) * 0.04
      nd[i] = last * 6
    }
    const wind = ctx.createBufferSource()
    wind.buffer = nb
    wind.loop = true
    const wf = ctx.createBiquadFilter()
    wf.type = 'bandpass'
    wf.frequency.value = 500
    wf.Q.value = 0.5
    const wg = ctx.createGain()
    wg.gain.value = 0
    wind.connect(wf)
    wf.connect(wg)
    wg.connect(master)
    wind.start()
    this.windFilter = wf
    this.windGain = wg

    // wings: a quiet, fluttering hum
    const buzz = ctx.createOscillator()
    buzz.type = 'sawtooth'
    buzz.frequency.value = 176
    const bf = ctx.createBiquadFilter()
    bf.type = 'lowpass'
    bf.frequency.value = 620
    const bg = ctx.createGain()
    bg.gain.value = 0
    const flutter = ctx.createOscillator()
    flutter.frequency.value = 23
    const fg = ctx.createGain()
    fg.gain.value = 0.005
    flutter.connect(fg)
    fg.connect(bg.gain)
    buzz.connect(bf)
    bf.connect(bg)
    bg.connect(master)
    buzz.start()
    flutter.start()
    this.buzz = buzz
    this.buzzGain = bg

    // the tune: scheduled a little ahead of the clock, so it keeps time even if frames drop
    this.seqTime = ctx.currentTime + 0.4
    this.timer = window.setInterval(this.schedule, 60)
  }

  setMuted(m: boolean): void {
    this.muted = m
    if (this.ctx && this.master) this.master.gain.setTargetAtTime(m ? 0 : 0.9, this.ctx.currentTime, 0.05)
  }

  /** How many islands are awake. */
  setLevel(n: number): void {
    this.level = n
  }

  private readonly schedule = (): void => {
    const ctx = this.ctx
    if (!ctx || ctx.state !== 'running') return
    // after a long pause (tab in the background) start again from now instead of catching up
    if (this.seqTime < ctx.currentTime - 0.5) this.seqTime = ctx.currentTime + 0.1
    while (this.seqTime < ctx.currentTime + 0.3) {
      this.playStep(this.seqStep, this.seqTime)
      this.seqTime += STEP
      this.seqStep++
    }
  }

  private playStep(index: number, t: number): void {
    const s = index % 16
    const barIndex = Math.floor(index / 16) % BARS.length
    const chord = BARS[barIndex]
    const lv = this.level

    // always: a pad under everything and a sparse music box
    if (s === 0) for (const m of chord.tones.slice(0, 3)) this.pad(m - 12, t, STEP * 17, lv >= 5 ? 0.034 : 0.026)
    if (s === 0 || s === 6 || s === 10) this.pluck(chord.tones[(barIndex * 3 + s) % 4] + 12, 0.05, t, 1.6, 'sine')

    if (lv >= 1 && s % 2 === 0) this.pluck(chord.tones[ARP[s / 2]], 0.055, t, 0.5, 'triangle')
    if (lv >= 2) {
      if (s === 0 || s === 8) this.bass(chord.bass, t, STEP * 7, 0.2)
      if (s === 14) this.bass(chord.bass + 7, t, STEP * 2, 0.12)
    }
    if (lv >= 3) {
      if (s % 4 === 2) this.shake(t, 0.035)
      if (s === 0 || s === 8) this.thump(t)
    }
    if (lv >= 4) {
      for (const [at, note] of MELODY[barIndex]) {
        if (at !== s) continue
        this.chime(note, 0.085, 0, 2.4, t)
        if (lv >= 6) this.chime(note - 12, 0.05, 0, 2.4, t)
      }
    }
    if (lv >= 5 && Math.random() < 0.4) this.pluck(SCALE[7 + Math.floor(Math.random() * 6)] + 12, 0.018, t, 0.35, 'sine')
  }

  private out(node: AudioNode, reverb = true): void {
    if (!this.master || !this.wet) return
    node.connect(this.master)
    if (reverb) node.connect(this.wet)
  }

  private pluck(midi: number, vel: number, t: number, long: number, type: OscillatorType): void {
    const ctx = this.ctx
    if (!ctx) return
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.exponentialRampToValueAtTime(vel, t + 0.008)
    g.gain.exponentialRampToValueAtTime(0.0001, t + long)
    const o = ctx.createOscillator()
    o.type = type
    o.frequency.value = hz(midi)
    o.connect(g)
    o.start(t)
    o.stop(t + long + 0.05)
    this.out(g)
  }

  private pad(midi: number, t: number, long: number, vel: number): void {
    const ctx = this.ctx
    if (!ctx) return
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.linearRampToValueAtTime(vel, t + long * 0.35)
    g.gain.linearRampToValueAtTime(0.0001, t + long)
    for (const det of [-6, 6]) {
      const o = ctx.createOscillator()
      o.type = 'triangle'
      o.frequency.value = hz(midi)
      o.detune.value = det
      o.connect(g)
      o.start(t)
      o.stop(t + long + 0.05)
    }
    const lp = ctx.createBiquadFilter()
    lp.type = 'lowpass'
    lp.frequency.value = 900
    g.connect(lp)
    this.out(lp)
  }

  private bass(midi: number, t: number, long: number, vel: number): void {
    const ctx = this.ctx
    if (!ctx) return
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.exponentialRampToValueAtTime(vel, t + 0.02)
    g.gain.exponentialRampToValueAtTime(0.0001, t + long)
    const lp = ctx.createBiquadFilter()
    lp.type = 'lowpass'
    lp.frequency.value = 520
    const o = ctx.createOscillator()
    o.type = 'triangle'
    o.frequency.value = hz(midi)
    o.connect(lp)
    lp.connect(g)
    o.start(t)
    o.stop(t + long + 0.05)
    this.out(g, false)
  }

  private shake(t: number, vel: number): void {
    const ctx = this.ctx
    if (!ctx || !this.noise) return
    const src = ctx.createBufferSource()
    src.buffer = this.noise
    const hp = ctx.createBiquadFilter()
    hp.type = 'highpass'
    hp.frequency.value = 6500
    const g = ctx.createGain()
    g.gain.setValueAtTime(vel, t)
    g.gain.exponentialRampToValueAtTime(0.0001, t + 0.07)
    src.connect(hp)
    hp.connect(g)
    src.start(t, Math.random() * 0.8, 0.1)
    this.out(g, false)
  }

  private thump(t: number): void {
    const ctx = this.ctx
    if (!ctx) return
    const o = ctx.createOscillator()
    o.frequency.setValueAtTime(110, t)
    o.frequency.exponentialRampToValueAtTime(52, t + 0.16)
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.exponentialRampToValueAtTime(0.16, t + 0.01)
    g.gain.exponentialRampToValueAtTime(0.0001, t + 0.3)
    o.connect(g)
    o.start(t)
    o.stop(t + 0.35)
    this.out(g, false)
  }

  private chime(midi: number, vel: number, delay = 0, long = 1.7, at?: number): void {
    const ctx = this.ctx
    if (!ctx) return
    const t = at ?? ctx.currentTime + delay
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.exponentialRampToValueAtTime(vel, t + 0.006)
    g.gain.exponentialRampToValueAtTime(0.0001, t + long)
    const f = hz(midi)
    const parts: [number, number][] = [[1, 1], [2.0, 0.32], [3.01, 0.1], [4.2, 0.04]]
    for (const [mult, level] of parts) {
      const o = ctx.createOscillator()
      o.type = 'sine'
      o.frequency.value = f * mult
      const pg = ctx.createGain()
      pg.gain.value = level
      o.connect(pg)
      pg.connect(g)
      o.start(t)
      o.stop(t + long + 0.05)
    }
    this.out(g)
  }

  /** `speed` 0..1, called every frame. */
  flight(speed: number, flying: boolean): void {
    const ctx = this.ctx
    if (!ctx || !this.windGain || !this.windFilter || !this.buzzGain || !this.buzz) return
    const t = ctx.currentTime
    this.windGain.gain.setTargetAtTime(flying ? 0.03 + speed * 0.12 : 0.015, t, 0.2)
    this.windFilter.frequency.setTargetAtTime(380 + speed * 1100, t, 0.2)
    this.buzzGain.gain.setTargetAtTime(flying ? 0.009 + speed * 0.007 : 0, t, 0.2)
    this.buzz.frequency.setTargetAtTime(168 + speed * 60, t, 0.15)
  }

  /** One grain of pollen: the next note of its own little tune. */
  note(): void {
    const ctx = this.ctx
    if (!ctx) return
    // a long pause starts a new phrase
    if (ctx.currentTime - this.lastNote > 6) this.step -= this.step % 4
    this.lastNote = ctx.currentTime
    this.chime(SCALE[TUNE[this.step % TUNE.length]] + 12, 0.15, 0, 1.3)
    this.step++
  }

  /** A race hoop: each one a step higher. */
  gate(index: number): void {
    this.chime(SCALE[2 + (index % 10)] + 12, 0.16, 0, 0.9)
  }

  raceStart(): void {
    this.chime(79, 0.2, 0, 0.6)
    this.chime(84, 0.22, 0.12, 1.2)
  }

  /** The end of a lap: the better the medal, the longer the flourish. 0 = no medal. */
  fanfare(medal: number): void {
    if (medal === 0) {
      this.chime(72, 0.16, 0, 1)
      this.chime(67, 0.16, 0.18, 1.4)
      return
    }
    const run = [72, 76, 79, 84, 88, 91].slice(0, 3 + medal)
    run.forEach((m, k) => this.chime(m, 0.2, k * 0.09, 2.2))
    this.chime(run[run.length - 1] + 12, 0.14, run.length * 0.09 + 0.1, 3)
  }

  /** Pollen poured into the hive; a longer peal when it fills a honey level. */
  deliver(levelUp: boolean): void {
    const ctx = this.ctx
    if (!ctx) return
    for (let k = 0; k < 6; k++) this.pluck(SCALE[5 + k] + 12, 0.07, ctx.currentTime + k * 0.05, 0.5, 'triangle')
    if (!levelUp) return
    const peal = [60, 64, 67, 72, 76, 79, 84]
    peal.forEach((m, k) => this.chime(m, 0.2, 0.35 + k * 0.1, 3))
  }

  /** A flower field opening: a rising run, higher for each field already open on the island. */
  bloom(done: number): void {
    const ctx = this.ctx
    if (!ctx || !this.noise) return
    const run = [0, 2, 3, 5, 7, 8]
    run.forEach((i, k) => this.chime(SCALE[Math.min(SCALE.length - 1, i + Math.min(done, 4))], 0.2, k * 0.075, 2.2))
    // a breath of air
    const t = ctx.currentTime
    const src = ctx.createBufferSource()
    src.buffer = this.noise
    const hp = ctx.createBiquadFilter()
    hp.type = 'highpass'
    hp.frequency.value = 2600
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.linearRampToValueAtTime(0.08, t + 0.25)
    g.gain.linearRampToValueAtTime(0.0001, t + 0.7)
    src.connect(hp)
    hp.connect(g)
    src.start(t, 0, 0.75)
    this.out(g)
  }

  /** An island waking: a low swell and a cascade of bells. The new instrument joins by itself. */
  wake(): void {
    const ctx = this.ctx
    if (!ctx) return
    const t = ctx.currentTime
    const sub = ctx.createOscillator()
    sub.frequency.setValueAtTime(95, t)
    sub.frequency.exponentialRampToValueAtTime(42, t + 1.4)
    const sg = ctx.createGain()
    sg.gain.setValueAtTime(0.0001, t)
    sg.gain.exponentialRampToValueAtTime(0.45, t + 0.05)
    sg.gain.exponentialRampToValueAtTime(0.0001, t + 1.8)
    sub.connect(sg)
    sub.start(t)
    sub.stop(t + 2)
    this.out(sg, false)
    const bells = [0, 2, 4, 5, 7, 9, 7, 9, 10, 12]
    bells.forEach((i, k) => this.chime(SCALE[i], 0.18, 0.15 + k * 0.11, 3))
    for (const m of [48, 55, 64, 67]) {
      const o = ctx.createOscillator()
      o.type = 'sawtooth'
      o.frequency.value = hz(m)
      const lp = ctx.createBiquadFilter()
      lp.type = 'lowpass'
      lp.frequency.setValueAtTime(300, t)
      lp.frequency.linearRampToValueAtTime(1800, t + 2)
      lp.frequency.linearRampToValueAtTime(400, t + 6)
      const g = ctx.createGain()
      g.gain.setValueAtTime(0.0001, t)
      g.gain.linearRampToValueAtTime(0.04, t + 1.4)
      g.gain.linearRampToValueAtTime(0.0001, t + 6.5)
      o.connect(lp)
      lp.connect(g)
      o.start(t)
      o.stop(t + 7)
      this.out(g)
    }
  }

  dispose(): void {
    window.clearInterval(this.timer)
    void this.ctx?.close()
    this.ctx = undefined
  }
}
