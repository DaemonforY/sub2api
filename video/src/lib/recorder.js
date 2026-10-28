// Records the microphone as 16-bit mono WAV, which 豆包声音复刻 accepts everywhere (MediaRecorder gives
// WebM in Chrome and MP4 in Safari, and the server has no transcoder). The level callback drives a meter.

const TARGET_RATE = 24000

export async function startRecording({ onLevel } = {}) {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true, autoGainControl: true } })
  const Ctx = window.AudioContext || window.webkitAudioContext
  const ctx = new Ctx()
  const source = ctx.createMediaStreamSource(stream)
  // ScriptProcessorNode is deprecated but works in every browser, including WeChat's.
  const node = ctx.createScriptProcessor(4096, 1, 1)
  const chunks = []
  node.onaudioprocess = (e) => {
    const data = e.inputBuffer.getChannelData(0)
    chunks.push(new Float32Array(data))
    if (onLevel) {
      let peak = 0
      for (let i = 0; i < data.length; i += 16) peak = Math.max(peak, Math.abs(data[i]))
      onLevel(peak)
    }
  }
  source.connect(node)
  node.connect(ctx.destination)
  const startedAt = performance.now()

  function release() {
    node.disconnect()
    source.disconnect()
    stream.getTracks().forEach((t) => t.stop())
    ctx.close().catch(() => {})
  }

  return {
    seconds: () => (performance.now() - startedAt) / 1000,
    cancel: release,
    /** Stops and returns { blob, seconds }. */
    stop() {
      release()
      const total = chunks.reduce((n, c) => n + c.length, 0)
      const samples = new Float32Array(total)
      let off = 0
      for (const c of chunks) {
        samples.set(c, off)
        off += c.length
      }
      const pcm = resample(samples, ctx.sampleRate, TARGET_RATE)
      return { blob: encodeWAV(pcm, TARGET_RATE), seconds: pcm.length / TARGET_RATE }
    }
  }
}

function resample(input, from, to) {
  if (from === to) return input
  const ratio = from / to
  const out = new Float32Array(Math.floor(input.length / ratio))
  for (let i = 0; i < out.length; i++) {
    // average the source samples this output sample covers (a cheap low-pass)
    const start = Math.floor(i * ratio)
    const end = Math.min(input.length, Math.floor((i + 1) * ratio))
    let sum = 0
    for (let j = start; j < end; j++) sum += input[j]
    out[i] = end > start ? sum / (end - start) : input[start] || 0
  }
  return out
}

function encodeWAV(samples, rate) {
  const buf = new ArrayBuffer(44 + samples.length * 2)
  const v = new DataView(buf)
  const str = (o, s) => [...s].forEach((ch, i) => v.setUint8(o + i, ch.charCodeAt(0)))
  str(0, 'RIFF')
  v.setUint32(4, 36 + samples.length * 2, true)
  str(8, 'WAVEfmt ')
  v.setUint32(16, 16, true)
  v.setUint16(20, 1, true)
  v.setUint16(22, 1, true)
  v.setUint32(24, rate, true)
  v.setUint32(28, rate * 2, true)
  v.setUint16(32, 2, true)
  v.setUint16(34, 16, true)
  str(36, 'data')
  v.setUint32(40, samples.length * 2, true)
  for (let i = 0; i < samples.length; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]))
    v.setInt16(44 + i * 2, s < 0 ? s * 0x8000 : s * 0x7fff, true)
  }
  return new Blob([buf], { type: 'audio/wav' })
}
