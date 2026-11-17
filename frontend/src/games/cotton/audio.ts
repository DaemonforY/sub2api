// Chiptune for 棉花糖大作战, played live with WebAudio: two pulse voices, a triangle bass and a
// noise drum kit, like a 90s arcade board. The five tunes are original; each bar names a chord,
// the bass and the arpeggio are built from it, the lead is written note by note (16 steps a bar).
// Sound effects react to the simulation's events.

import type { Ev } from './sim'

interface Song {
  bpm: number
  chords: string[]
  lead: string[]
  loop: boolean
  /** drum pattern, 16 steps: k kick, s snare, h hat */
  drums: string
  swing?: number
}

const SONGS: Record<string, Song> = {
  title: {
    bpm: 136,
    chords: ['G', 'Em', 'C', 'D', 'G', 'Em', 'C D', 'G'],
    lead: [
      'D5 . G5 . B5 . D6 - . . B5 . G5 . A5 .',
      'B5 - - . G5 . E5 . G5 . B5 . E6 - - .',
      'C6 . B5 . A5 . G5 . E5 . G5 . C6 . E6 .',
      'D6 - - - F#5 - - . A5 . D6 . . . . .',
      'D5 . G5 . B5 . D6 - . . G6 . F#6 . E6 .',
      'D6 - B5 . G5 . E5 . G5 . B5 . E6 . D6 .',
      'C6 . E6 . G6 - E6 . D6 . F#6 . A6 - - .',
      'G6 - - - D6 - - - G5 - - . . . . .'
    ],
    loop: true,
    drums: 'k . h . s . h h k . h . s . h .'
  },
  bedroom: {
    bpm: 140,
    chords: ['C', 'Am', 'F', 'G', 'C', 'Am', 'Dm G', 'C'],
    lead: [
      'E5 . G5 . C6 - B5 . A5 . G5 . E5 - - .',
      'C5 . E5 . A5 - G5 . E5 . C5 . D5 - - .',
      'F5 . A5 . C6 - A5 . F5 . A5 . C6 . D6 .',
      'D6 - B5 . G5 . B5 . D6 - - . G5 . . .',
      'E5 . G5 . C6 - B5 . A5 . G5 . E5 . G5 .',
      'A5 - - . G5 . E5 . C5 . E5 . A5 - G5 .',
      'F5 . A5 . D6 - . . B5 . G5 . D5 . F5 .',
      'E5 - - - C5 - - - C6 - - . . . . .'
    ],
    loop: true,
    drums: 'k . h . s . h . k . k h s . h .'
  },
  classroom: {
    bpm: 150,
    chords: ['Dm', 'Bb', 'C', 'F', 'Dm', 'Bb', 'Gm A', 'Dm'],
    lead: [
      'D5 . F5 . A5 . D6 . C6 . A5 . F5 . G5 .',
      'F5 - D5 . Bb4 . D5 . F5 . Bb5 - A5 . F5 .',
      'E5 . G5 . C6 - . . Bb5 . A5 . G5 . E5 .',
      'F5 - - . A5 . C6 . F6 - - . . . . .',
      'D6 . C6 . A5 . F5 . D5 . F5 . A5 . D6 .',
      'Bb5 - - . A5 . F5 . D5 . F5 . Bb5 . D6 .',
      'G5 . Bb5 . D6 . Bb5 . A5 . C#6 . E6 . A5 .',
      'D6 - - - A5 - - - D5 - - . . . . .'
    ],
    loop: true,
    drums: 'k . h k s . h . k . h k s . h h',
    swing: 0.12
  },
  rooftop: {
    bpm: 128,
    chords: ['Am', 'F', 'C', 'G', 'Am', 'F', 'E', 'E'],
    lead: [
      'A5 - - . C6 . E6 - - . D6 . C6 . B5 .',
      'A5 - - . F5 . A5 . C6 - - . A5 . . .',
      'G5 - - . E5 . G5 . C6 . E6 . D6 . C6 .',
      'B5 - - - D6 - - . B5 . G5 . . . . .',
      'A5 . C6 . E6 . A6 - G6 . E6 . C6 . E6 .',
      'F6 - - . E6 . C6 . A5 . C6 . F6 - E6 .',
      'E6 - D6 . C6 . B5 . G#5 . B5 . E6 - - .',
      'E6 - - - - - - . G#5 . B5 . D6 . . .'
    ],
    loop: true,
    drums: 'k . h . s . h . k . h . s . h k'
  },
  boss: {
    bpm: 162,
    chords: ['Em', 'Em', 'C', 'D', 'Em', 'Em', 'C', 'B'],
    lead: [
      'E5 . E5 . G5 . E5 . A5 . G5 . E5 . D5 .',
      'E5 . E5 . B5 . A5 . G5 . F#5 . E5 . . .',
      'C6 . B5 . G5 . E5 . C6 . B5 . G5 . A5 .',
      'D6 - - . C6 . B5 . A5 - - . F#5 . . .',
      'E6 . D6 . B5 . G5 . E6 . D6 . B5 . G5 .',
      'A5 . B5 . G5 . E5 . F#5 . G5 . A5 . B5 .',
      'C6 - - . B5 . G5 . E5 - G5 . C6 - B5 .',
      'B5 - - - D#6 - - - F#6 - - . . . . .'
    ],
    loop: true,
    drums: 'k . h k s . h k k . h k s s h s'
  },
  clear: {
    bpm: 170,
    chords: ['C', 'F G', 'C'],
    lead: ['C5 E5 G5 C6 - . G5 C6 E6 - - . D6 E6 F6 -', 'E6 - D6 . C6 . D6 . E6 - G6 - - - . .', 'C7 - - - - - - - . . . . . . . .'],
    loop: false,
    drums: 'k . . . s . . . k . k . s . . .'
  },
  over: {
    bpm: 110,
    chords: ['Am', 'F', 'E', 'Am'],
    lead: ['E5 - - . D5 - - . C5 - - . B4 - - .', 'C5 - - . A4 - - . F4 - - . A4 - - .', 'G#4 - - - B4 - - - E5 - - - D5 - - -', 'C5 - - - - - - - A4 - - - - - . .'],
    loop: false,
    drums: 'k . . . . . . . k . . . . . . .'
  },
  won: {
    bpm: 132,
    chords: ['C', 'F', 'G', 'C', 'F', 'G', 'Am G', 'C'],
    lead: [
      'G5 . C6 . E6 . G6 - - . E6 . G6 . . .',
      'A6 - - . F6 . A6 . C7 - - . A6 . F6 .',
      'G6 - - . D6 . B5 . G5 . B5 . D6 . F6 .',
      'E6 - - - C6 - - - G5 - - - C6 - - .',
      'F6 . E6 . F6 . A6 - - . G6 . F6 . E6 .',
      'D6 - - . B5 . D6 . G6 - - . F6 . D6 .',
      'C6 . E6 . A6 - - . B5 . D6 . G6 - - .',
      'C7 - - - - - - - - - - - . . . .'
    ],
    loop: false,
    drums: 'k . h . s . h . k . h . s . h h'
  }
}

const NOTE: Record<string, number> = { C: 0, 'C#': 1, Db: 1, D: 2, 'D#': 3, Eb: 3, E: 4, F: 5, 'F#': 6, Gb: 6, G: 7, 'G#': 8, Ab: 8, A: 9, 'A#': 10, Bb: 10, B: 11 }

function midi(tok: string): number {
  const m = /^([A-G][#b]?)(\d)$/.exec(tok)
  if (!m) return -1
  return NOTE[m[1]] + (Number(m[2]) + 1) * 12
}

function chordTones(name: string): number[] {
  const m = /^([A-G][#b]?)(m?)$/.exec(name)
  if (!m) return [48, 52, 55]
  const root = NOTE[m[1]] + 48
  return [root, root + (m[2] ? 3 : 4), root + 7]
}

function hz(n: number): number {
  return 440 * Math.pow(2, (n - 69) / 12)
}

interface Step {
  lead: number
  leadLen: number
  bass: number
  arp: number
  drum: string
}

function compile(song: Song): Step[] {
  const steps: Step[] = []
  song.chords.forEach((cname, bar) => {
    const parts = cname.split(' ')
    const toks = song.lead[bar].split(/\s+/)
    for (let i = 0; i < 16; i++) {
      const ch = chordTones(parts[parts.length > 1 && i >= 8 ? 1 : 0])
      const t = toks[i] ?? '.'
      let len = 0
      let note = -1
      if (t !== '.' && t !== '-') {
        note = midi(t)
        len = 1
        while (toks[i + len] === '-') len++
      }
      const bassPat = [0, -1, -1, -1, 12, -1, 0, -1, 7, -1, -1, -1, 0, -1, 7, -1]
      const b = bassPat[i]
      const arpPat = [0, 1, 2, 1]
      steps.push({
        lead: note,
        leadLen: len,
        bass: b >= 0 ? ch[0] - 12 + b : -1,
        arp: i % 2 === 0 ? ch[arpPat[(i / 2) % 4]] + 12 : -1,
        drum: song.drums.split(' ')[i] ?? '.'
      })
    }
  })
  return steps
}

export class CottonAudio {
  private ctx?: AudioContext
  private master?: GainNode
  private music?: GainNode
  private sfx?: GainNode
  private noise?: AudioBuffer
  private pulse25?: PeriodicWave
  private pulse12?: PeriodicWave
  private song: Step[] = []
  private songDef?: Song
  private songName = ''
  private pos = 0
  private nextTime = 0
  private timer = 0
  muted = false

  /** must run from a user gesture */
  start(): void {
    if (this.ctx) {
      void this.ctx.resume()
      return
    }
    const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
    if (!Ctx) return
    const ctx = new Ctx()
    this.ctx = ctx
    this.master = ctx.createGain()
    this.master.gain.value = this.muted ? 0 : 0.8
    this.master.connect(ctx.destination)
    this.music = ctx.createGain()
    this.music.gain.value = 0.32
    this.music.connect(this.master)
    this.sfx = ctx.createGain()
    this.sfx.gain.value = 0.55
    this.sfx.connect(this.master)
    const len = ctx.sampleRate
    this.noise = ctx.createBuffer(1, len, ctx.sampleRate)
    const d = this.noise.getChannelData(0)
    for (let i = 0; i < len; i++) d[i] = Math.random() * 2 - 1
    this.pulse25 = this.pulseWave(0.25)
    this.pulse12 = this.pulseWave(0.125)
    this.timer = window.setInterval(() => this.schedule(), 25)
  }

  private pulseWave(duty: number): PeriodicWave {
    const n = 32
    const real = new Float32Array(n)
    const imag = new Float32Array(n)
    for (let k = 1; k < n; k++) imag[k] = (2 / (k * Math.PI)) * Math.sin(k * Math.PI * duty)
    return this.ctx!.createPeriodicWave(real, imag)
  }

  setMuted(m: boolean): void {
    this.muted = m
    if (this.master && this.ctx) this.master.gain.setTargetAtTime(m ? 0 : 0.8, this.ctx.currentTime, 0.02)
  }

  play(name: string): void {
    if (this.songName === name) return
    this.songName = name
    const def = SONGS[name]
    if (!def) {
      this.song = []
      this.songDef = undefined
      return
    }
    this.songDef = def
    this.song = compile(def)
    this.pos = 0
    this.nextTime = this.ctx ? this.ctx.currentTime + 0.06 : 0
  }

  stopMusic(): void {
    this.songName = ''
    this.song = []
    this.songDef = undefined
  }

  dispose(): void {
    window.clearInterval(this.timer)
    void this.ctx?.close()
    this.ctx = undefined
  }

  private schedule(): void {
    const ctx = this.ctx
    const def = this.songDef
    if (!ctx || !def || !this.song.length) return
    const stepDur = 60 / def.bpm / 4
    if (this.nextTime < ctx.currentTime - 0.2) this.nextTime = ctx.currentTime + 0.02
    while (this.nextTime < ctx.currentTime + 0.12) {
      if (this.pos >= this.song.length) {
        if (!def.loop) {
          this.song = []
          this.songDef = undefined
          this.songName = ''
          return
        }
        this.pos = 0
      }
      const st = this.song[this.pos]
      const swing = def.swing && this.pos % 2 === 1 ? stepDur * def.swing : 0
      const t = this.nextTime + swing
      if (st.lead >= 0) this.tone(this.music!, this.pulse25!, hz(st.lead), t, st.leadLen * stepDur * 0.92, 0.22, 0.004)
      if (st.arp >= 0) this.tone(this.music!, this.pulse12!, hz(st.arp), t, stepDur * 0.8, 0.07, 0.002)
      if (st.bass >= 0) this.tone(this.music!, 'triangle', hz(st.bass), t, stepDur * 1.7, 0.38, 0.003)
      if (st.drum === 'k') this.kick(t)
      else if (st.drum === 's') this.snare(t)
      else if (st.drum === 'h') this.hat(t)
      this.pos++
      this.nextTime += stepDur
    }
  }

  private tone(dest: AudioNode, wave: OscillatorType | PeriodicWave, f: number, t: number, dur: number, vol: number, attack: number, slideTo = 0): void {
    const ctx = this.ctx!
    const o = ctx.createOscillator()
    if (typeof wave === 'string') o.type = wave as OscillatorType
    else o.setPeriodicWave(wave)
    o.frequency.setValueAtTime(f, t)
    if (slideTo) o.frequency.exponentialRampToValueAtTime(slideTo, t + dur)
    const g = ctx.createGain()
    g.gain.setValueAtTime(0.0001, t)
    g.gain.linearRampToValueAtTime(vol, t + attack)
    g.gain.setValueAtTime(vol, t + Math.max(attack, dur * 0.6))
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur)
    o.connect(g)
    g.connect(dest)
    o.start(t)
    o.stop(t + dur + 0.02)
  }

  private noiseHit(dest: AudioNode, t: number, dur: number, vol: number, freq: number, type: BiquadFilterType = 'highpass'): void {
    const ctx = this.ctx!
    const src = ctx.createBufferSource()
    src.buffer = this.noise!
    const f = ctx.createBiquadFilter()
    f.type = type
    f.frequency.value = freq
    const g = ctx.createGain()
    g.gain.setValueAtTime(vol, t)
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur)
    src.connect(f)
    f.connect(g)
    g.connect(dest)
    src.start(t, Math.random() * 0.5)
    src.stop(t + dur + 0.02)
  }

  private kick(t: number): void {
    this.tone(this.music!, 'sine', 150, t, 0.14, 0.7, 0.002, 45)
  }

  private snare(t: number): void {
    this.noiseHit(this.music!, t, 0.12, 0.32, 1800)
    this.tone(this.music!, 'triangle', 220, t, 0.06, 0.2, 0.001, 120)
  }

  private hat(t: number): void {
    this.noiseHit(this.music!, t, 0.035, 0.14, 7000)
  }

  // ------------------------------------------------------------ effects

  private last: Record<string, number> = {}

  private ok(key: string, gap: number): boolean {
    const now = this.ctx?.currentTime ?? 0
    if ((this.last[key] ?? -1) > now - gap) return false
    this.last[key] = now
    return true
  }

  react(evs: Ev[], localIdx: number[]): void {
    const ctx = this.ctx
    if (!ctx || !this.sfx) return
    const t = ctx.currentTime + 0.005
    const out = this.sfx
    const p25 = this.pulse25!
    for (const e of evs) {
      switch (e.k) {
        case 'shot':
          if (this.ok('shot' + e.p, 0.04)) this.tone(out, p25, 900, t, 0.07, 0.12, 0.002, 1500)
          break
        case 'hit':
          if (this.ok('hit', 0.03)) {
            this.noiseHit(out, t, 0.08, 0.25, 900, 'lowpass')
            this.tone(out, 'triangle', 300 + (e.n ?? 1) * 80, t, 0.07, 0.25, 0.002, 500 + (e.n ?? 1) * 100)
          }
          break
        case 'ball':
          [0, 0.05, 0.1].forEach((d, i) => this.tone(out, p25, hz(76 + i * 4), t + d, 0.08, 0.16, 0.002))
          break
        case 'nudge':
          this.tone(out, 'triangle', 500, t, 0.05, 0.15, 0.002)
          break
        case 'kick':
          this.noiseHit(out, t, 0.1, 0.5, 400, 'lowpass')
          this.tone(out, 'square', 220, t, 0.12, 0.25, 0.002, 70)
          break
        case 'bounce':
          if (this.ok('bounce', 0.05)) this.tone(out, 'square', 180, t, 0.08, 0.18, 0.002, 90)
          break
        case 'kill': {
          const n = Math.min(8, e.p ?? 1)
          const base = 72 + (n - 1) * 2
          this.tone(out, p25, hz(base), t, 0.06, 0.2, 0.002)
          this.tone(out, p25, hz(base + 7), t + 0.05, 0.06, 0.2, 0.002)
          this.tone(out, p25, hz(base + 12), t + 0.1, 0.12, 0.2, 0.002)
          this.noiseHit(out, t, 0.12, 0.25, 2000)
          break
        }
        case 'burst':
          this.noiseHit(out, t, 0.22, 0.45, 600, 'lowpass')
          this.tone(out, 'triangle', 400, t, 0.18, 0.25, 0.002, 900)
          break
        case 'thaw':
          this.tone(out, 'sawtooth', 300, t, 0.2, 0.12, 0.002, 150)
          break
        case 'pickup': {
          const k = e.p ?? 0
          if (k >= 6 && k <= 9) [0, 4, 7, 12, 16].forEach((d, i) => this.tone(out, p25, hz(72 + d), t + i * 0.045, 0.09, 0.18, 0.002))
          else {
            this.tone(out, p25, hz(84), t, 0.05, 0.16, 0.002)
            this.tone(out, p25, hz(91), t + 0.05, 0.1, 0.16, 0.002)
          }
          break
        }
        case 'die':
          [0, 1, 2, 3, 4, 5].forEach((i) => this.tone(out, p25, hz(76 - i * 3), t + i * 0.07, 0.08, 0.2, 0.002))
          break
        case 'revive':
        case 'respawn':
          [0, 4, 7, 12, 7, 12, 16].forEach((d, i) => this.tone(out, p25, hz(67 + d), t + i * 0.05, 0.08, 0.16, 0.002))
          break
        case 'bump':
          this.tone(out, 'square', 260, t, 0.1, 0.15, 0.002, 520)
          break
        case 'jump':
          if (e.p !== undefined && localIdx.includes(e.p)) this.tone(out, p25, 320, t, 0.12, 0.08, 0.002, 640)
          break
        case 'land':
          if (e.p === 9 && this.ok('land', 0.05)) this.noiseHit(out, t, 0.06, 0.2, 300, 'lowpass')
          break
        case 'hurry':
          [0, 0.18, 0.36, 0.54].forEach((d, i) => this.tone(out, 'square', i % 2 ? 660 : 880, t + d, 0.15, 0.18, 0.002))
          break
        case 'slipper':
          this.noiseHit(out, t, 0.4, 0.3, 1200, 'bandpass')
          break
        case 'throw':
          if (this.ok('throw', 0.08)) this.tone(out, 'triangle', 700, t, 0.1, 0.12, 0.002, 350)
          break
        case 'quake':
          this.noiseHit(out, t, 0.35, 0.6, 180, 'lowpass')
          this.tone(out, 'sine', 90, t, 0.3, 0.5, 0.002, 40)
          break
        case 'ring':
          if (this.ok('ring', 0.15)) for (let i = 0; i < 6; i++) this.tone(out, 'square', i % 2 ? 1320 : 1180, t + i * 0.04, 0.04, (e.p ?? 0) > 0 ? 0.12 : 0.05, 0.001)
          break
        case 'bosshit':
          if (this.ok('bosshit', 0.04)) this.tone(out, 'square', (e.n ?? 1) > 4 ? 120 : 240, t, 0.08, 0.22, 0.002, 90)
          break
        case 'bossdie':
          for (let i = 0; i < 6; i++) this.noiseHit(out, t + i * 0.12, 0.3, 0.5, 500 - i * 50, 'lowpass')
          break
        case 'sweep':
          for (let i = 0; i < 10; i++) this.tone(out, p25, hz(72 + [0, 4, 7, 12, 16, 19, 24, 19, 24, 28][i]), t + i * 0.06, 0.1, 0.18, 0.002)
          break
        case 'spawn':
          if (this.ok('spawn', 0.06)) this.tone(out, 'triangle', 600, t, 0.06, 0.08, 0.002, 1200)
          break
        case 'poof':
          if (e.n && this.ok('poof', 0.04)) this.tone(out, 'triangle', 800, t, 0.05, 0.12, 0.002, 400)
          break
        default:
          break
      }
    }
  }
}
