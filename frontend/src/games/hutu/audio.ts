// Music and sound for 虎兔大冒险, synthesised in the browser in the style of an 8-bit console:
// two pulse channels, a triangle bass and a noise drum. All tunes are original, mostly on Chinese
// pentatonic scales. A song is a chord per bar plus a melody; bass, arpeggio and drums are generated
// from the chords so every song stays in the same voice.

import type { MusicId } from './types'
import type { SfxId } from './world'

type Chord = [root: number, minor: boolean]
const NOTE: Record<string, number> = { C: 0, 'C#': 1, D: 2, 'D#': 3, E: 4, F: 5, 'F#': 6, G: 7, 'G#': 8, A: 9, 'A#': 10, B: 11 }
const CH: Record<string, Chord> = {
  C: [48, false],
  D: [50, false],
  Dm: [50, true],
  E: [52, false],
  Em: [52, true],
  F: [53, false],
  G: [55, false],
  Gm: [55, true],
  A: [57, false],
  Am: [57, true],
  'A#': [58, false],
  B: [59, false]
}

export interface Song {
  tempo: number
  chords: string[]
  melody: string
  drums: 'full' | 'light' | 'march' | 'none'
  loop: boolean
}

export const SONGS: Record<Exclude<MusicId, 'world'>, Song> = {
  title: {
    tempo: 132,
    chords: ['G', 'C', 'D', 'G', 'Em', 'C', 'D', 'G'],
    melody:
      'G5:4 D5:2 E5:2 G5:4 A5:4 | E5:2 G5:2 E5:2 D5:2 E5:8 | A5:4 G5:2 E5:2 D5:4 A4:4 | B4:4 D5:4 G5:8 | E5:2 G5:2 B5:4 A5:2 G5:2 E5:4 | G5:2 E5:2 C5:4 E5:4 G5:4 | A5:2 B5:2 A5:2 G5:2 D5:4 F#5:4 | G5:12 -:4',
    drums: 'full',
    loop: true
  },
  field: {
    tempo: 150,
    chords: ['C', 'C', 'Am', 'G', 'F', 'C', 'Dm', 'G'],
    melody:
      'E5:2 G5:2 A5:4 G5:2 E5:2 D5:4 | C5:2 D5:2 E5:2 G5:2 E5:4 -:4 | A4:2 C5:2 D5:4 E5:2 D5:2 C5:4 | D5:4 E5:2 G5:2 A5:6 -:2 | C6:2 A5:2 G5:4 A5:2 G5:2 E5:4 | G5:2 E5:2 D5:2 C5:2 D5:4 E5:4 | A4:2 A4:2 C5:2 D5:2 E5:2 G5:2 E5:2 D5:2 | D5:4 G4:2 A4:2 B4:4 D5:4',
    drums: 'full',
    loop: true
  },
  cave: {
    tempo: 112,
    chords: ['Am', 'Am', 'F', 'G', 'Am', 'Am', 'Dm', 'E'],
    melody:
      'A4:4 -:2 C5:2 E5:4 D5:4 | C5:2 A4:2 -:4 G4:4 A4:4 | F4:4 A4:2 C5:2 F5:4 E5:4 | D5:6 -:2 B4:4 G4:4 | A4:4 -:2 E5:2 A5:4 G5:4 | E5:2 D5:2 C5:4 D5:4 E5:4 | F5:4 E5:2 D5:2 A4:4 D5:4 | E5:6 -:2 G#4:4 B4:4',
    drums: 'light',
    loop: true
  },
  sky: {
    tempo: 126,
    chords: ['F', 'C', 'Dm', 'A#', 'F', 'C', 'A#', 'C'],
    melody:
      'A5:4 C6:4 A5:2 G5:2 F5:4 | G5:4 E5:4 C5:8 | D5:2 F5:2 A5:4 G5:2 F5:2 D5:4 | F5:4 D5:4 A#4:8 | A5:2 C6:2 F6:4 E6:2 C6:2 A5:4 | G5:4 C6:4 E5:8 | D5:2 F5:2 A#5:4 A5:2 G5:2 F5:4 | G5:4 E5:4 C5:4 -:4',
    drums: 'light',
    loop: true
  },
  fort: {
    tempo: 128,
    chords: ['Dm', 'Dm', 'A#', 'A', 'Dm', 'Dm', 'Gm', 'A'],
    melody:
      'D5:2 -:2 D5:2 F5:2 E5:4 D5:4 | A4:4 A#4:4 A4:4 -:4 | F5:2 -:2 F5:2 G5:2 F5:4 D5:4 | E5:4 C#5:4 A4:8 | D5:2 F5:2 A5:4 G#5:2 A5:2 F5:4 | D5:4 E5:2 F5:2 D5:8 | G5:2 -:2 A#5:4 A5:2 G5:2 D5:4 | E5:4 C#5:4 E5:2 C#5:2 A4:4',
    drums: 'march',
    loop: true
  },
  town: {
    tempo: 144,
    chords: ['G', 'G', 'Em', 'D', 'C', 'G', 'Am', 'D'],
    melody:
      'D5:2 E5:2 G5:2 A5:2 B5:4 A5:2 G5:2 | E5:2 G5:2 D5:4 B4:2 D5:2 G4:4 | E5:2 E5:2 G5:2 E5:2 D5:4 B4:4 | A4:2 B4:2 D5:2 E5:2 D5:4 A4:4 | G5:2 A5:2 G5:2 E5:2 G5:4 E5:4 | D5:2 E5:2 D5:2 B4:2 D5:4 G5:4 | A5:2 G5:2 E5:2 D5:2 E5:2 G5:2 A5:4 | A5:4 F#5:4 D5:4 -:4',
    drums: 'full',
    loop: true
  },
  night: {
    tempo: 96,
    chords: ['Am', 'F', 'C', 'G', 'Am', 'F', 'G', 'Am'],
    melody:
      'E5:6 D5:2 C5:4 A4:4 | F5:6 E5:2 C5:4 A4:4 | G5:4 E5:4 C5:4 E5:4 | D5:8 B4:4 G4:4 | A5:6 G5:2 E5:4 C5:4 | F5:4 A5:4 C6:4 A5:4 | G5:4 D5:4 B4:4 D5:4 | A4:8 -:8',
    drums: 'light',
    loop: true
  },
  boss: {
    tempo: 168,
    chords: ['Em', 'Em', 'C', 'D', 'Em', 'Em', 'C', 'B'],
    melody:
      'E5:2 E5:2 G5:2 E5:2 A5:2 G5:2 E5:2 D5:2 | E5:2 -:2 B4:2 -:2 E5:2 F#5:2 G5:4 | C6:2 B5:2 G5:2 E5:2 C5:2 E5:2 G5:4 | D6:2 C6:2 A5:2 F#5:2 D5:4 A5:4 | E5:2 E5:2 G5:2 E5:2 B5:2 A5:2 G5:2 F#5:2 | E5:4 G5:4 B5:4 E6:4 | C6:2 B5:2 A5:2 G5:2 E5:2 G5:2 C6:4 | B5:4 A5:2 F#5:2 D#5:4 B4:4',
    drums: 'full',
    loop: true
  },
  bell: {
    tempo: 176,
    chords: ['C', 'F', 'G', 'C'],
    melody:
      'C5:2 E5:2 G5:2 C6:2 G5:2 E5:2 G5:2 C6:2 | A5:2 F5:2 A5:2 C6:2 F6:2 C6:2 A5:2 F5:2 | G5:2 B5:2 D6:2 G6:2 D6:2 B5:2 G5:2 B5:2 | C6:4 G5:2 E5:2 C5:4 -:4',
    drums: 'full',
    loop: true
  },
  clear: {
    tempo: 150,
    chords: ['C', 'F', 'G', 'C'],
    melody: 'G4:2 C5:2 E5:2 G5:2 C6:4 G5:4 | A5:2 C6:2 A5:2 F5:2 A5:8 | B5:2 D6:2 B5:2 G5:2 D6:4 B5:4 | C6:12 -:4',
    drums: 'march',
    loop: false
  },
  over: {
    tempo: 100,
    chords: ['Am', 'F', 'G', 'Am'],
    melody: 'E5:4 C5:4 A4:8 | F4:4 G4:4 A4:4 C5:4 | B4:4 G4:4 D5:8 | A4:12 -:4',
    drums: 'none',
    loop: false
  }
}

export interface ParsedNote {
  step: number
  len: number
  midi: number
}

export function parseMelody(src: string): { notes: ParsedNote[]; bars: number[] } {
  const notes: ParsedNote[] = []
  const bars: number[] = []
  let step = 0
  for (const bar of src.split('|')) {
    let barLen = 0
    for (const tok of bar.trim().split(/\s+/)) {
      if (!tok) continue
      const [n, l] = tok.split(':')
      const len = Number(l)
      if (n !== '-') {
        const m = /^([A-G]#?)(\d)$/.exec(n)
        if (!m) throw new Error(`bad note ${tok}`)
        notes.push({ step, len, midi: 12 * (Number(m[2]) + 1) + NOTE[m[1]] })
      }
      step += len
      barLen += len
    }
    bars.push(barLen)
  }
  return { notes, bars }
}

function hz(m: number): number {
  return 440 * Math.pow(2, (m - 69) / 12)
}

export class Audio8 {
  private ctx?: AudioContext
  private master?: GainNode
  private musicBus?: GainNode
  private sfxBus?: GainNode
  private noise?: AudioBuffer
  private pulse: Record<string, PeriodicWave> = {}
  private muted = false
  private song: Song | null = null
  private songId = ''
  private parsed: ParsedNote[] = []
  private step = 0
  private nextTime = 0
  private timer = 0
  private songLen = 0

  start(): void {
    if (this.ctx) {
      void this.ctx.resume()
      return
    }
    const Ctor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctor) return
    const ctx = new Ctor()
    this.ctx = ctx
    const master = ctx.createGain()
    master.gain.value = this.muted ? 0 : 0.8
    const comp = ctx.createDynamicsCompressor()
    comp.threshold.value = -12
    master.connect(comp)
    comp.connect(ctx.destination)
    this.master = master
    this.musicBus = ctx.createGain()
    this.musicBus.gain.value = 0.55
    this.musicBus.connect(master)
    this.sfxBus = ctx.createGain()
    this.sfxBus.gain.value = 0.7
    this.sfxBus.connect(master)
    const buf = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate)
    const d = buf.getChannelData(0)
    for (let i = 0; i < d.length; i++) d[i] = Math.random() * 2 - 1
    this.noise = buf
    for (const duty of [0.125, 0.25, 0.5]) {
      const n = 32
      const re = new Float32Array(n)
      const im = new Float32Array(n)
      for (let k = 1; k < n; k++) im[k] = ((2 / (k * Math.PI)) * Math.sin(k * Math.PI * duty))
      this.pulse[String(duty)] = ctx.createPeriodicWave(re, im)
    }
    this.timer = window.setInterval(() => this.schedule(), 40)
  }

  destroy(): void {
    window.clearInterval(this.timer)
    void this.ctx?.close()
    this.ctx = undefined
  }

  setMuted(m: boolean): void {
    this.muted = m
    if (this.ctx && this.master) this.master.gain.setTargetAtTime(m ? 0 : 0.8, this.ctx.currentTime, 0.03)
  }

  play(id: MusicId | ''): void {
    if (id === this.songId) return
    this.songId = id
    if (!id || id === 'world') {
      this.song = null
      return
    }
    const s = SONGS[id]
    this.song = s
    this.parsed = parseMelody(s.melody).notes
    this.songLen = s.chords.length * 16
    this.step = 0
    if (this.ctx) this.nextTime = this.ctx.currentTime + 0.06
  }

  private schedule(): void {
    const ctx = this.ctx
    if (!ctx || !this.song || ctx.state !== 'running') return
    const s = this.song
    const dt = 60 / s.tempo / 4
    if (this.nextTime < ctx.currentTime - 0.3) this.nextTime = ctx.currentTime + 0.05
    while (this.nextTime < ctx.currentTime + 0.2) {
      if (this.step >= this.songLen) {
        if (!s.loop) {
          this.song = null
          return
        }
        this.step = 0
      }
      this.playStep(s, this.step, this.nextTime, dt)
      this.nextTime += dt
      this.step++
    }
  }

  private playStep(s: Song, step: number, t: number, dt: number): void {
    const bar = Math.floor(step / 16)
    const st = step % 16
    const [root, minor] = CH[s.chords[bar]] || CH.C
    const third = root + (minor ? 3 : 4)
    const fifth = root + 7
    // melody
    for (const n of this.parsed) if (n.step === step) this.tone('0.25', hz(n.midi), t, n.len * dt * 0.92, 0.13, true)
    // arpeggio (quiet, high)
    if (s.drums !== 'none') {
      const arp = [root + 12, third + 12, fifth + 12, third + 12]
      if (st % 2 === 0) this.tone('0.125', hz(arp[(st / 2) % 4] + 12), t, dt * 1.6, 0.035, false)
    }
    // triangle bass
    const bassPat = s.drums === 'light' ? [0, 8] : [0, 4, 8, 12]
    if (bassPat.includes(st)) {
      const n = st === 4 || st === 12 ? fifth - 12 : root - 12
      this.tri(hz(n), t, dt * (s.drums === 'light' ? 6 : 3), 0.32)
    }
    // drums
    switch (s.drums) {
      case 'full':
        if (st === 0 || st === 8 || st === 10) this.kick(t)
        if (st === 4 || st === 12) this.snare(t, 0.16)
        if (st % 2 === 1) this.hat(t, 0.035)
        break
      case 'light':
        if (st === 0) this.kick(t, 0.5)
        if (st === 8) this.hat(t, 0.05)
        if (st % 4 === 2) this.hat(t, 0.02)
        break
      case 'march':
        if (st % 4 === 0) this.kick(t, 0.7)
        if (st === 12 || st === 14 || st === 15) this.snare(t, 0.1)
        break
    }
  }

  private env(g: GainNode, t: number, len: number, vol: number, decay: boolean): void {
    g.gain.setValueAtTime(0.0001, t)
    g.gain.linearRampToValueAtTime(vol, t + 0.005)
    if (decay) g.gain.setTargetAtTime(vol * 0.6, t + 0.02, 0.08)
    g.gain.setTargetAtTime(0.0001, t + len, 0.02)
  }

  private tone(duty: string, f: number, t: number, len: number, vol: number, decay: boolean, bus?: GainNode): void {
    const ctx = this.ctx
    if (!ctx) return
    const o = ctx.createOscillator()
    o.setPeriodicWave(this.pulse[duty])
    o.frequency.value = f
    const g = ctx.createGain()
    this.env(g, t, len, vol, decay)
    o.connect(g)
    g.connect(bus || (this.musicBus as GainNode))
    o.start(t)
    o.stop(t + len + 0.15)
  }

  private tri(f: number, t: number, len: number, vol: number): void {
    const ctx = this.ctx
    if (!ctx) return
    const o = ctx.createOscillator()
    o.type = 'triangle'
    o.frequency.value = f
    const g = ctx.createGain()
    this.env(g, t, len, vol, false)
    o.connect(g)
    g.connect(this.musicBus as GainNode)
    o.start(t)
    o.stop(t + len + 0.1)
  }

  private noiseHit(t: number, len: number, vol: number, freq: number, type: BiquadFilterType, bus?: GainNode): void {
    const ctx = this.ctx
    if (!ctx || !this.noise) return
    const src = ctx.createBufferSource()
    src.buffer = this.noise
    const f = ctx.createBiquadFilter()
    f.type = type
    f.frequency.value = freq
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol, t)
    g.gain.exponentialRampToValueAtTime(0.0001, t + len)
    src.connect(f)
    f.connect(g)
    g.connect(bus || (this.musicBus as GainNode))
    src.start(t, Math.random() * 0.5)
    src.stop(t + len + 0.02)
  }

  private kick(t: number, vol = 0.8): void {
    const ctx = this.ctx
    if (!ctx) return
    const o = ctx.createOscillator()
    o.type = 'triangle'
    o.frequency.setValueAtTime(150, t)
    o.frequency.exponentialRampToValueAtTime(40, t + 0.12)
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol * 0.6, t)
    g.gain.exponentialRampToValueAtTime(0.0001, t + 0.14)
    o.connect(g)
    g.connect(this.musicBus as GainNode)
    o.start(t)
    o.stop(t + 0.16)
  }

  private snare(t: number, vol: number): void {
    this.noiseHit(t, 0.12, vol, 1800, 'bandpass')
  }

  private hat(t: number, vol: number): void {
    this.noiseHit(t, 0.04, vol, 7000, 'highpass')
  }

  /** a pitch sweep on a pulse channel, for most effects */
  private sweep(duty: string, f0: number, f1: number, len: number, vol: number, t0 = 0): void {
    const ctx = this.ctx
    if (!ctx) return
    const t = ctx.currentTime + t0
    const o = ctx.createOscillator()
    o.setPeriodicWave(this.pulse[duty])
    o.frequency.setValueAtTime(f0, t)
    o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + len)
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol, t)
    g.gain.setTargetAtTime(0.0001, t + len * 0.7, len * 0.15)
    o.connect(g)
    g.connect(this.sfxBus as GainNode)
    o.start(t)
    o.stop(t + len + 0.1)
  }

  private notes(duty: string, midis: number[], each: number, vol: number): void {
    midis.forEach((m, i) => this.sweep(duty, hz(m), hz(m), each, vol, i * each))
  }

  sfx(id: SfxId): void {
    const ctx = this.ctx
    if (!ctx || this.muted) return
    const now = ctx.currentTime
    switch (id) {
      case 'jump':
        this.sweep('0.5', 300, 760, 0.16, 0.12)
        break
      case 'jumpBig':
        this.sweep('0.5', 220, 620, 0.2, 0.13)
        break
      case 'stomp':
        this.sweep('0.25', 520, 1300, 0.08, 0.14)
        this.noiseHit(now, 0.06, 0.15, 900, 'lowpass', this.sfxBus)
        break
      case 'coin':
        this.notes('0.25', [83, 88], 0.06, 0.1)
        break
      case 'bump':
      case 'thud':
        this.sweep('0.5', 160, 70, 0.08, 0.16)
        break
      case 'break':
        this.noiseHit(now, 0.25, 0.3, 1200, 'lowpass', this.sfxBus)
        this.sweep('0.5', 200, 60, 0.15, 0.1)
        break
      case 'sprout':
        this.sweep('0.25', 200, 900, 0.4, 0.09)
        break
      case 'grow':
        this.notes('0.5', [60, 64, 67, 72, 64, 67, 72, 76], 0.045, 0.09)
        break
      case 'shrink':
        this.notes('0.5', [76, 72, 67, 64, 60, 55], 0.05, 0.1)
        break
      case 'fire':
        this.sweep('0.125', 900, 200, 0.1, 0.1)
        break
      case 'kick':
        this.sweep('0.25', 700, 200, 0.07, 0.14)
        break
      case 'die':
        this.notes('0.5', [72, 71, 70, 64, 63, 62, 52], 0.08, 0.12)
        break
      case 'oneup':
        this.notes('0.25', [76, 79, 88, 84, 86, 91], 0.07, 0.11)
        break
      case 'bell':
        this.notes('0.125', [84, 91, 96], 0.05, 0.1)
        break
      case 'tube':
        this.notes('0.5', [55, 50, 55, 50, 55, 50], 0.06, 0.12)
        break
      case 'pole':
        this.sweep('0.25', 1200, 300, 0.6, 0.1)
        break
      case 'boom':
        this.noiseHit(now, 0.5, 0.5, 600, 'lowpass', this.sfxBus)
        this.sweep('0.5', 120, 30, 0.35, 0.2)
        break
      case 'fuse':
        this.noiseHit(now, 0.05, 0.06, 5000, 'highpass', this.sfxBus)
        break
      case 'splash':
        this.noiseHit(now, 0.2, 0.15, 2500, 'bandpass', this.sfxBus)
        break
      case 'bosshit':
        this.notes('0.5', [48, 60, 48, 60], 0.05, 0.16)
        this.noiseHit(now, 0.2, 0.25, 900, 'lowpass', this.sfxBus)
        break
      case 'roar':
        this.sweep('0.125', 90, 50, 0.8, 0.2)
        this.noiseHit(now, 0.7, 0.2, 400, 'lowpass', this.sfxBus)
        break
      case 'tick':
        this.sweep('0.25', 1800, 1800, 0.03, 0.06)
        break
      case 'lantern':
        this.notes('0.25', [72, 76, 79, 84], 0.07, 0.1)
        break
      case 'pop':
        this.sweep('0.5', 400, 1600, 0.1, 0.1)
        break
      case 'pause':
        this.notes('0.5', [88, 83, 88, 83], 0.06, 0.08)
        break
    }
  }
}
