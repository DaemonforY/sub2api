// Chiptune music + sound effects for 椰风岛大冒险, synthesised with WebAudio. All tunes are original.
import { EV, type GameEvent } from './defs'

type Track = 'title' | 'beach' | 'jungle' | 'volcano' | 'boss' | 'fairy' | 'clear' | 'over' | 'ending'

interface Song {
  bpm: number
  loop: boolean
  lead: string
  harm?: string
  bass: string
  drums: string
  leadType?: OscillatorType | 'pulse25' | 'pulse12'
  harmType?: OscillatorType | 'pulse25' | 'pulse12'
}

const NOTE: Record<string, number> = { C: 0, 'C#': 1, Db: 1, D: 2, 'D#': 3, Eb: 3, E: 4, F: 5, 'F#': 6, Gb: 6, G: 7, 'G#': 8, Ab: 8, A: 9, 'A#': 10, Bb: 10, B: 11 }

function freq(n: string): number {
  const m = /^([A-G][#b]?)(\d)$/.exec(n)
  if (!m) return 0
  const midi = (Number(m[2]) + 1) * 12 + NOTE[m[1]]
  return 440 * Math.pow(2, (midi - 69) / 12)
}

/** "C5:2 r:2 E5:4" → per-step list of [freq, steps] note-ons (null where nothing starts). */
function parse(seq: string): ([number, number] | null)[] {
  const out: ([number, number] | null)[] = []
  for (const tok of seq.trim().split(/\s+/)) {
    const [n, d] = tok.split(':')
    const len = Number(d || 1)
    out.push(n === 'r' ? null : [freq(n), len])
    for (let i = 1; i < len; i++) out.push(null)
  }
  return out
}

function bassBars(chords: string[], pattern: (r: string, f: string, o: string) => string): string {
  return chords
    .map((ch) => {
      const [r, f, o] = ch.split('/')
      return pattern(r, f, o)
    })
    .join(' ')
}

const calypso = (r: string, f: string) => `${r}:3 ${r}:1 ${f}:2 ${r}:2 r:2 ${r}:2 ${f}:2 ${r}:2`
const drive = (r: string, f: string, o: string) => `${r}:2 ${r}:2 ${o}:2 ${r}:2 ${r}:2 ${o}:2 ${r}:2 ${f}:2`
const groove = (r: string, f: string) => `${r}:2 r:1 ${r}:1 ${f}:2 ${r}:2 r:2 ${r}:2 ${f}:2 ${r}:2`
const pump = (r: string, _f: string, o: string) => `${r}:1 ${r}:1 ${r}:2 ${r}:1 ${r}:1 ${r}:2 ${r}:1 ${r}:1 ${r}:2 ${o}:2 ${r}:2`
const offbeat = (notes: string[]) => notes.map((n) => `r:2 ${n}:2 r:2 ${n}:2 r:2 ${n}:2 r:2 ${n}:2`).join(' ')

const SONGS: Record<Track, Song> = {
  title: {
    bpm: 118,
    loop: true,
    lead: 'G5:2 B5:2 D6:4 B5:2 D6:2 G6:4 E6:2 D6:2 B5:2 G5:2 A5:8 C6:2 B5:2 A5:2 G5:2 E5:4 G5:4 A5:2 B5:2 C6:2 A5:2 G5:8',
    harm: offbeat(['B4', 'G4', 'E4', 'F#4']),
    bass: bassBars(['G2/D3/G3', 'E2/B2/E3', 'C3/G3/C4', 'D3/A3/D4'], (r, f) => `${r}:4 ${f}:4 ${r}:4 ${f}:4`),
    drums: 'k...s...k.k.s...k...s...k.k.s.h.k...s...k.k.s...k...s...kks.s.ss',
    leadType: 'pulse25'
  },
  beach: {
    bpm: 140,
    loop: true,
    lead:
      'E5:2 G5:2 C6:3 B5:1 A5:2 G5:2 E5:4 A5:2 C6:2 A5:2 G5:2 E5:4 D5:2 E5:2 F5:3 F5:1 A5:2 C6:2 A5:2 F5:2 G5:2 A5:2 G5:3 D5:1 G5:2 B5:2 A5:4 G5:4 ' +
      'E5:2 G5:2 C6:3 D6:1 E6:2 D6:2 C6:4 A5:2 C6:2 E6:2 D6:2 C6:2 A5:2 G5:4 F5:2 A5:2 C6:2 A5:2 G5:2 B5:2 D6:4 C6:6 G5:2 E5:4 r:4',
    harm: offbeat(['G4', 'E4', 'A4', 'B4', 'G4', 'E4', 'A4', 'E4']),
    bass: bassBars(['C3/G2', 'A2/E2', 'F2/C3', 'G2/D3', 'C3/G2', 'A2/E2', 'F2/G2', 'C3/G2'], calypso),
    drums: 'k.h.s.hkk.h.s.hh'.repeat(8),
    leadType: 'pulse25'
  },
  jungle: {
    bpm: 124,
    loop: true,
    lead:
      'D5:2 F5:2 A5:2 G5:2 F5:2 D5:2 C5:2 D5:2 E5:2 G5:2 C6:2 A5:2 G5:4 E5:4 D5:2 F5:2 Bb5:3 A5:1 G5:2 F5:2 D5:4 E5:3 F5:1 G5:2 A5:2 E5:4 C5:4 ' +
      'D5:1 F5:1 A5:2 D6:4 C6:2 A5:2 F5:4 G5:2 E5:2 C5:2 E5:2 G5:4 C6:4 Bb5:2 A5:2 G5:2 F5:2 D5:4 F5:4 E5:2 C#5:2 E5:2 A5:2 D5:8',
    harm: offbeat(['A4', 'G4', 'F4', 'G4', 'A4', 'G4', 'F4', 'E4']),
    bass: bassBars(['D3/A2', 'C3/G2', 'Bb2/F2', 'C3/G2', 'D3/A2', 'C3/G2', 'Bb2/F2', 'A2/E2'], groove),
    drums: 'k..k..s.k.k.s..h'.repeat(8),
    leadType: 'triangle',
    harmType: 'pulse12'
  },
  volcano: {
    bpm: 156,
    loop: true,
    lead:
      'A4:2 C5:2 E5:2 A5:4 G5:2 E5:2 C5:2 F5:2 E5:2 C5:2 A4:2 C5:4 F5:4 G5:3 F5:1 E5:2 D5:2 B4:2 D5:2 G5:4 G#5:4 E5:2 B4:2 E5:4 G#5:4 ' +
      'A5:2 B5:2 C6:4 B5:2 A5:2 E5:4 F5:2 A5:2 C6:4 A5:2 F5:2 C5:4 D5:2 G5:2 B5:4 A5:2 G5:2 D5:4 E5:3 E5:1 G#5:2 B5:2 E6:8',
    harm: offbeat(['E4', 'C4', 'D4', 'B3', 'E4', 'C4', 'D4', 'G#4']),
    bass: bassBars(['A2/E2/A3', 'F2/C3/F3', 'G2/D3/G3', 'E2/B2/E3', 'A2/E2/A3', 'F2/C3/F3', 'G2/D3/G3', 'E2/B2/E3'], drive),
    drums: 'k.h.s.h.k.k.s.hs'.repeat(8),
    leadType: 'pulse25'
  },
  boss: {
    bpm: 168,
    loop: true,
    lead: 'E5:2 r:1 E5:1 F5:2 E5:2 B4:2 E5:2 G5:4 F5:2 r:1 F5:1 G5:2 F5:2 C5:2 F5:2 A5:4 E5:2 G5:2 B5:2 A#5:2 A5:2 G5:2 F5:2 E5:2 D5:2 F5:2 A5:2 G5:2 F5:4 E5:4',
    harm: offbeat(['B4', 'C5', 'B4', 'A4']),
    bass: bassBars(['E2/B2/E3', 'F2/C3/F3', 'E2/B2/E3', 'D2/A2/D3'], pump),
    drums: 'k.s.k.s.kks.k.ss'.repeat(4),
    leadType: 'square'
  },
  fairy: {
    bpm: 180,
    loop: true,
    lead: 'C6:1 E6:1 G6:1 C7:1 G6:1 E6:1 C6:2 D6:1 F6:1 A6:1 D7:1 A6:1 F6:1 D6:2 E6:1 G6:1 B6:1 E7:1 B6:1 G6:1 E6:2 F6:1 A6:1 C7:1 F7:1 G6:2 r:2',
    bass: 'C3:2 C4:2 C3:2 C4:2 D3:2 D4:2 D3:2 D4:2 E3:2 E4:2 E3:2 E4:2 F3:2 F4:2 G3:2 G4:2',
    drums: 'k.h.s.h.k.h.s.h.'.repeat(2),
    leadType: 'pulse12'
  },
  clear: {
    bpm: 150,
    loop: false,
    lead: 'C5:1 E5:1 G5:1 C6:3 r:1 G5:1 C6:2 D6:2 E6:8',
    bass: 'C3:2 r:2 G2:2 r:1 C3:3 G2:2 C3:8',
    drums: 'k.h.k.h.s.h.k...s.......',
    leadType: 'pulse25'
  },
  over: {
    bpm: 100,
    loop: false,
    lead: 'G4:3 F#4:3 F4:3 E4:6 r:1',
    bass: 'C3:3 B2:3 Bb2:3 A2:6',
    drums: '................'
  },
  ending: {
    bpm: 112,
    loop: true,
    lead: 'G5:2 B5:2 D6:4 B5:2 D6:2 G6:4 E6:2 D6:2 B5:2 G5:2 A5:8 C6:2 B5:2 A5:2 G5:2 E5:4 G5:4 A5:2 B5:2 C6:2 D6:2 G6:8',
    harm: offbeat(['B4', 'G4', 'E4', 'F#4']),
    bass: bassBars(['G2/D3/G3', 'E2/B2/E3', 'C3/G3/C4', 'D3/A3/D4'], (r, f) => `${r}:4 ${f}:4 ${r}:4 ${f}:4`),
    drums: 'k...s...k.k.s...'.repeat(4),
    leadType: 'pulse25'
  }
}

export class GameAudio {
  ctx: AudioContext | null = null
  private master!: GainNode
  private musicBus!: GainNode
  private sfxBus!: GainNode
  private noise!: AudioBuffer
  private waves: Record<string, PeriodicWave> = {}
  private track: Track | null = null
  private song: Song | null = null
  private parsed: { lead: ([number, number] | null)[]; harm: ([number, number] | null)[]; bass: ([number, number] | null)[]; drums: string } | null = null
  private step = 0
  private nextTime = 0
  private timer: number | null = null
  muted = false
  musicOn = true

  /** Must be called from a user gesture. */
  unlock() {
    if (this.ctx) {
      void this.ctx.resume()
      return
    }
    const Ctor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctor) return
    const ctx = new Ctor()
    this.ctx = ctx
    this.master = ctx.createGain()
    this.master.gain.value = this.muted ? 0 : 0.7
    this.master.connect(ctx.destination)
    this.musicBus = ctx.createGain()
    this.musicBus.gain.value = 0.32
    this.musicBus.connect(this.master)
    this.sfxBus = ctx.createGain()
    this.sfxBus.gain.value = 0.55
    this.sfxBus.connect(this.master)
    this.noise = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate)
    const d = this.noise.getChannelData(0)
    for (let i = 0; i < d.length; i++) d[i] = Math.random() * 2 - 1
    this.waves.pulse25 = this.pulse(0.25)
    this.waves.pulse12 = this.pulse(0.125)
    this.timer = window.setInterval(() => this.schedule(), 25)
    if (this.track) {
      const t = this.track
      this.track = null
      this.play(t)
    }
  }

  private pulse(duty: number): PeriodicWave {
    const ctx = this.ctx as AudioContext
    const n = 32
    const real = new Float32Array(n)
    const imag = new Float32Array(n)
    for (let k = 1; k < n; k++) imag[k] = (2 / (k * Math.PI)) * Math.sin(k * Math.PI * duty)
    return ctx.createPeriodicWave(real, imag)
  }

  setMuted(m: boolean) {
    this.muted = m
    if (this.ctx) this.master.gain.setTargetAtTime(m ? 0 : 0.7, this.ctx.currentTime, 0.02)
  }

  play(t: Track) {
    if (this.track === t) return
    this.track = t
    this.song = SONGS[t]
    this.parsed = {
      lead: parse(this.song.lead),
      harm: this.song.harm ? parse(this.song.harm) : [],
      bass: parse(this.song.bass),
      drums: this.song.drums
    }
    this.step = 0
    if (this.ctx) this.nextTime = this.ctx.currentTime + 0.06
  }

  stop() {
    this.track = null
    this.song = null
    this.parsed = null
  }

  get current(): Track | null {
    return this.track
  }

  destroy() {
    if (this.timer !== null) window.clearInterval(this.timer)
    this.timer = null
    void this.ctx?.close()
    this.ctx = null
  }

  private schedule() {
    const ctx = this.ctx
    if (!ctx || !this.song || !this.parsed) return
    const stepDur = 60 / this.song.bpm / 4
    if (this.nextTime < ctx.currentTime - 0.2) this.nextTime = ctx.currentTime + 0.02
    while (this.nextTime < ctx.currentTime + 0.12) {
      const p = this.parsed
      const len = Math.max(p.lead.length, p.bass.length, p.drums.length)
      if (this.step >= len) {
        if (!this.song.loop) {
          this.song = null
          return
        }
        this.step = 0
      }
      const s = this.step
      const t = this.nextTime
      const ln = p.lead[s % p.lead.length]
      if (ln && s < p.lead.length) this.tone(this.song.leadType ?? 'pulse25', ln[0], t, ln[1] * stepDur * 0.92, 0.16, this.musicBus, true)
      const hn = p.harm.length ? p.harm[s % p.harm.length] : null
      if (hn) this.tone(this.song.harmType ?? 'pulse12', hn[0], t, hn[1] * stepDur * 0.7, 0.07, this.musicBus)
      const bn = p.bass[s % p.bass.length]
      if (bn && s < p.bass.length) this.tone('triangle', bn[0], t, bn[1] * stepDur * 0.9, 0.3, this.musicBus)
      const dch = p.drums[s % p.drums.length]
      if (dch && dch !== '.' && s < p.drums.length) this.drum(dch, t)
      this.nextTime += stepDur
      this.step++
    }
  }

  private tone(type: string, f: number, t: number, dur: number, vol: number, bus: GainNode, vibrato = false) {
    const ctx = this.ctx
    if (!ctx || !this.musicOn && bus === this.musicBus) return
    const o = ctx.createOscillator()
    if (this.waves[type]) o.setPeriodicWave(this.waves[type])
    else o.type = type as OscillatorType
    o.frequency.setValueAtTime(f, t)
    if (vibrato && dur > 0.25) {
      const lfo = ctx.createOscillator()
      const lg = ctx.createGain()
      lfo.frequency.value = 6
      lg.gain.setValueAtTime(0, t)
      lg.gain.linearRampToValueAtTime(f * 0.012, t + dur)
      lfo.connect(lg)
      lg.connect(o.frequency)
      lfo.start(t)
      lfo.stop(t + dur + 0.05)
    }
    const g = ctx.createGain()
    g.gain.setValueAtTime(0, t)
    g.gain.linearRampToValueAtTime(vol, t + 0.005)
    g.gain.setValueAtTime(vol, t + Math.max(0.01, dur - 0.03))
    g.gain.linearRampToValueAtTime(0, t + dur)
    o.connect(g)
    g.connect(bus)
    o.start(t)
    o.stop(t + dur + 0.02)
  }

  private drum(ch: string, t: number) {
    const ctx = this.ctx
    if (!ctx || !this.musicOn) return
    if (ch === 'k') {
      const o = ctx.createOscillator()
      const g = ctx.createGain()
      o.frequency.setValueAtTime(150, t)
      o.frequency.exponentialRampToValueAtTime(45, t + 0.1)
      g.gain.setValueAtTime(0.55, t)
      g.gain.exponentialRampToValueAtTime(0.001, t + 0.13)
      o.connect(g)
      g.connect(this.musicBus)
      o.start(t)
      o.stop(t + 0.14)
      return
    }
    const src = ctx.createBufferSource()
    src.buffer = this.noise
    const f = ctx.createBiquadFilter()
    const g = ctx.createGain()
    const dur = ch === 's' ? 0.11 : ch === 'o' ? 0.16 : 0.03
    f.type = ch === 's' ? 'bandpass' : 'highpass'
    f.frequency.value = ch === 's' ? 1800 : 7000
    g.gain.setValueAtTime(ch === 's' ? 0.32 : 0.12, t)
    g.gain.exponentialRampToValueAtTime(0.001, t + dur)
    src.connect(f)
    f.connect(g)
    g.connect(this.musicBus)
    src.start(t, Math.random() * 0.5)
    src.stop(t + dur + 0.01)
  }

  // ───────────── sound effects ─────────────

  private sweep(type: string, f0: number, f1: number, dur: number, vol = 0.2, delay = 0) {
    const ctx = this.ctx
    if (!ctx) return
    const t = ctx.currentTime + delay
    const o = ctx.createOscillator()
    if (this.waves[type]) o.setPeriodicWave(this.waves[type])
    else o.type = type as OscillatorType
    o.frequency.setValueAtTime(f0, t)
    o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur)
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol, t)
    g.gain.exponentialRampToValueAtTime(0.001, t + dur)
    o.connect(g)
    g.connect(this.sfxBus)
    o.start(t)
    o.stop(t + dur + 0.02)
  }

  private hiss(dur: number, vol: number, f0: number, f1: number, type: BiquadFilterType = 'bandpass', delay = 0) {
    const ctx = this.ctx
    if (!ctx) return
    const t = ctx.currentTime + delay
    const src = ctx.createBufferSource()
    src.buffer = this.noise
    const f = ctx.createBiquadFilter()
    f.type = type
    f.frequency.setValueAtTime(f0, t)
    f.frequency.exponentialRampToValueAtTime(Math.max(30, f1), t + dur)
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol, t)
    g.gain.exponentialRampToValueAtTime(0.001, t + dur)
    src.connect(f)
    f.connect(g)
    g.connect(this.sfxBus)
    src.start(t, Math.random() * 0.5)
    src.stop(t + dur + 0.02)
  }

  private arp(notes: number[], step: number, type = 'pulse25', vol = 0.16) {
    notes.forEach((f, i) => this.sweep(type, f, f * 1.001, step * 1.6, vol, i * step))
  }

  sfx(evs: GameEvent[], localPlayers: number[]) {
    if (!this.ctx) return
    const mine = (i: number) => localPlayers.includes(i)
    for (const [type, , , a] of evs) {
      switch (type) {
        case EV.jump:
          this.sweep('pulse25', 260, 620, 0.13, mine(a) ? 0.12 : 0.06)
          break
        case EV.throw:
          this.hiss(0.14, 0.18, 1200, 4000)
          this.sweep('pulse12', 900, 450, 0.08, 0.08)
          break
        case EV.kill:
          this.sweep('square', 520, 1400, 0.07, 0.13)
          this.sweep('pulse25', 1400, 1900, 0.08, 0.1, 0.06)
          this.hiss(0.12, 0.2, 3000, 600)
          break
        case EV.fruit: {
          const combo = a & 15
          const base = 1046.5 * Math.pow(2, Math.min(12, combo) / 12)
          this.sweep('pulse25', base, base, 0.05, 0.12)
          this.sweep('pulse25', base * 1.5, base * 1.5, 0.09, 0.12, 0.05)
          break
        }
        case EV.clam:
          this.hiss(0.12, 0.3, 2500, 800)
          this.sweep('triangle', 300, 120, 0.15, 0.3)
          break
        case EV.item:
          this.arp([523, 659, 784, 1047, 1319], 0.05)
          break
        case EV.hurt:
        case EV.boardLost:
          this.sweep('square', 700, 140, 0.3, 0.14)
          break
        case EV.trip:
          this.sweep('triangle', 220, 70, 0.18, 0.4)
          this.hiss(0.1, 0.15, 900, 300)
          break
        case EV.die:
          this.arp([784, 698, 587, 523, 392, 330, 262], 0.07, 'square', 0.12)
          break
        case EV.revive:
          this.arp([392, 523, 659, 784, 1047, 1319], 0.045, 'pulse12', 0.13)
          break
        case EV.flag:
          this.arp([523, 659, 784, 1047, 784, 1047], 0.07)
          break
        case EV.spring:
          this.sweep('sine', 180, 900, 0.28, 0.3)
          break
        case EV.bossHit:
          this.hiss(0.1, 0.3, 1800, 500)
          this.sweep('square', 220, 90, 0.12, 0.15)
          break
        case EV.bossDie:
          this.hiss(a < 0 ? 0.3 : 0.9, a < 0 ? 0.25 : 0.45, 1500, 60, 'lowpass')
          if (a >= 0) this.sweep('sawtooth', 300, 40, 0.9, 0.12)
          break
        case EV.breakRock:
          this.hiss(0.16, 0.35, 1600, 200, 'lowpass')
          break
        case EV.extinguish:
          this.hiss(0.4, 0.22, 6000, 1500, 'highpass')
          break
        case EV.splash:
          this.hiss(0.3, 0.3, a ? 600 : 2200, a ? 200 : 500)
          break
        case EV.clink:
          this.sweep('square', 2200, 1800, 0.05, 0.07)
          break
        case EV.quake:
          this.hiss(0.5, 0.4, 300, 40, 'lowpass')
          this.sweep('triangle', 90, 40, 0.4, 0.4)
          break
        case EV.bossRoar:
          this.sweep('sawtooth', 120, 55, 0.9, 0.14)
          this.hiss(0.8, 0.12, 400, 100, 'lowpass')
          break
        case EV.spit:
          this.hiss(0.08, 0.2, 2000, 700)
          this.sweep('pulse25', 420, 200, 0.1, 0.07)
          break
        case EV.lowEnergy:
          if (mine(a)) this.sweep('pulse12', 1320, 1320, 0.05, 0.06)
          break
        case EV.life:
          if (a > 0) this.arp([659, 784, 1319, 1047, 1175, 1568], 0.06)
          break
        case EV.sun:
          this.arp([523, 784, 1047, 1319, 1568, 2093], 0.06, 'triangle', 0.25)
          break
        case EV.fairy:
          this.arp([1047, 1319, 1568, 2093], 0.04, 'pulse12', 0.1)
          break
        case EV.cloud:
          this.sweep('triangle', 400, 160, 0.4, 0.25)
          break
        case EV.catch:
          if (mine(a)) this.sweep('pulse12', 1500, 1500, 0.025, 0.05)
          break
        case EV.land:
          if (mine(a)) this.hiss(0.05, 0.06, 500, 200, 'lowpass')
          break
        default:
      }
    }
  }
}

export type { Track }
