// HiveGPT 视频 MP4 renderer. sub2api posts one job at a time; this loads the video site's
// /render.html (the sandboxed player plus subtitles, landscape or portrait), seeks the film frame by
// frame, pipes screenshots into ffmpeg with the narration mixed in, and writes the MP4 and a cover
// next to the project's audio on the shared data volume.
import http from 'node:http'
import fs from 'node:fs'
import path from 'node:path'
import { spawn } from 'node:child_process'
import puppeteer from 'puppeteer-core'

const PORT = Number(process.env.PORT || 7100)
const ROOT = path.resolve(process.env.VIDEO_ROOT || '/app/data/video')
const ORIGIN = (process.env.RENDER_ORIGIN || 'https://video.hivegpt.cn').replace(/\/$/, '')
const CHROME = process.env.CHROME_PATH || '/usr/bin/chromium'
const THREADS = String(process.env.FFMPEG_THREADS || 2)

let current = null // { out, promise } of the job being rendered

function inside(p) {
  const abs = path.resolve(ROOT, p)
  if (abs !== ROOT && !abs.startsWith(ROOT + path.sep)) throw new Error('path outside the video dir: ' + p)
  return abs
}

function run(cmd, args) {
  return new Promise((resolve, reject) => {
    const proc = spawn(cmd, args, { stdio: ['ignore', 'ignore', 'pipe'] })
    let err = ''
    proc.stderr.on('data', (d) => (err = (err + d).slice(-2000)))
    proc.on('error', reject)
    proc.on('close', (code) => (code === 0 ? resolve() : reject(new Error(`${cmd} exited ${code}: ${err}`))))
  })
}

// The narration track: every scene's audio at its start time, as 48 kHz stereo WAV.
async function mixNarration(audio, total, out) {
  const args = ['-y', '-loglevel', 'error']
  const parts = audio.filter((a) => a.file)
  if (!parts.length) {
    await run('ffmpeg', [...args, '-f', 'lavfi', '-i', 'anullsrc=r=48000:cl=stereo', '-t', String(total), out])
    return
  }
  for (const a of parts) args.push('-i', inside(a.file))
  const chains = parts.map((a, i) => {
    const ms = Math.max(0, Math.round(a.start * 1000))
    return `[${i}:a]aresample=48000,aformat=channel_layouts=stereo,adelay=${ms}|${ms}[a${i}]`
  })
  const mix = parts.map((_, i) => `[a${i}]`).join('') + `amix=inputs=${parts.length}:normalize=0:dropout_transition=0,loudnorm=I=-16:TP=-1.5:LRA=11,aresample=48000,apad[out]`
  await run('ffmpeg', [...args, '-filter_complex', chains.join(';') + ';' + mix, '-map', '[out]', '-t', String(total), '-c:a', 'pcm_s16le', out])
}

async function render(job) {
  const { setup, fps = 60, width, height } = job
  const out = inside(job.out)
  const cover = job.cover ? inside(job.cover) : ''
  const progress = job.progress ? inside(job.progress) : ''
  if (!job.url || !job.url.startsWith(ORIGIN + '/')) throw new Error('render page must be on ' + ORIGIN)
  if (!(width > 0 && height > 0 && width <= 3840 && height <= 3840)) throw new Error('bad size')
  fs.mkdirSync(path.dirname(out), { recursive: true })
  const note = (data) => progress && fs.writeFileSync(progress, JSON.stringify({ ...data, at: Date.now() }))
  note({ stage: 'loading', frame: 0, frames: 0 })

  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: 'new',
    args: ['--no-sandbox', '--disable-dev-shm-usage', '--hide-scrollbars', '--mute-audio', '--force-color-profile=srgb',
      '--use-angle=swiftshader', '--enable-unsafe-swiftshader', `--window-size=${width},${height}`]
  })
  const wav = out.replace(/\.mp4$/, '') + '.narration.wav'
  const tmp = out + '.part'
  const errors = []
  try {
    const page = await browser.newPage()
    await page.setViewport({ width, height, deviceScaleFactor: 1 })
    page.on('pageerror', (e) => errors.push('page: ' + e.message))
    await page.goto(job.url, { waitUntil: 'networkidle0', timeout: 60000 })
    await page.evaluate((s) => window.__setup(s), setup)
    await page.waitForFunction('window.__ready === true', { timeout: 120000 })
    const player = page.frames().find((f) => f.url().includes('/player.html'))
    if (!player) throw new Error('player frame not found')
    await page.evaluate(() => document.fonts.ready)
    await player.evaluate(() => document.fonts.ready)
    const full = await page.evaluate('window.__TOTAL')
    if (!(full > 0)) throw new Error('empty film')
    const total = job.limit > 0 ? Math.min(full, job.limit) : full // limit: render only the first seconds (tests)
    const frames = Math.round(total * fps)

    note({ stage: 'audio', frame: 0, frames })
    await mixNarration(job.audio || [], total, wav)

    const ff = spawn('ffmpeg', ['-y', '-loglevel', 'error', '-threads', THREADS,
      '-f', 'image2pipe', '-framerate', String(fps), '-c:v', 'mjpeg', '-i', '-', '-i', wav,
      '-map', '0:v', '-map', '1:a',
      '-c:v', 'libx264', '-preset', 'slow', '-tune', 'animation', '-crf', '16', '-profile:v', 'high', '-level', '4.2',
      '-pix_fmt', 'yuv420p', '-r', String(fps), '-g', String(fps * 2),
      '-colorspace', 'bt709', '-color_primaries', 'bt709', '-color_trc', 'bt709',
      '-c:a', 'aac', '-b:a', '256k', '-ar', '48000', '-ac', '2',
      '-t', String(total), '-movflags', '+faststart', '-f', 'mp4', tmp], { stdio: ['pipe', 'ignore', 'pipe'] })
    let ffErr = ''
    ff.stderr.on('data', (d) => (ffErr = (ffErr + d).slice(-2000)))
    const done = new Promise((resolve, reject) => {
      ff.on('error', reject)
      ff.on('close', (code) => (code === 0 ? resolve() : reject(new Error('ffmpeg exited ' + code + ': ' + ffErr))))
    })
    ff.stdin.on('error', () => {}) // a dying ffmpeg is reported by `done`
    const write = async (buf) => {
      if (!ff.stdin.write(buf)) await Promise.race([new Promise((r) => ff.stdin.once('drain', r)), done])
    }
    const cdp = await page.createCDPSession()
    const shot = async () => Buffer.from((await cdp.send('Page.captureScreenshot', { format: 'jpeg', quality: 95, optimizeForSpeed: true })).data, 'base64')
    const seek = async (t) => {
      await player.evaluate((t) => window.__film.seek(t), t)
      await page.evaluate((t) => window.__subtitle(t), t)
    }
    const started = Date.now()
    for (let i = 0; i < frames; i++) {
      await seek(i / fps)
      await write(await shot())
      if (i % 30 === 0) note({ stage: 'frames', frame: i, frames, fps: i / Math.max(1, (Date.now() - started) / 1000) })
    }
    ff.stdin.end()
    await done
    fs.renameSync(tmp, out)
    if (cover) {
      await seek(Math.min(total, Math.max(0, job.coverAt ?? total * 0.3)))
      fs.writeFileSync(cover, await shot())
    }
    errors.push(...(await page.evaluate(() => (window.__errors || []).map((e) => `${e.scene || ''} ${e.phase || ''} ${e.message}`))))
    const size = fs.statSync(out).size
    note({ stage: 'done', frame: frames, frames })
    return { size, duration: total, frames, seconds: Math.round((Date.now() - started) / 1000), errors: errors.slice(0, 20) }
  } finally {
    await browser.close().catch(() => {})
    fs.rmSync(wav, { force: true })
    fs.rmSync(tmp, { force: true })
  }
}

http.createServer(async (req, res) => {
  const send = (code, body) => {
    res.writeHead(code, { 'content-type': 'application/json' })
    res.end(JSON.stringify(body))
  }
  if (req.method === 'GET' && req.url === '/health') return send(200, { ok: true, busy: !!current })
  if (req.method !== 'POST' || req.url !== '/render') return send(404, { error: 'not found' })
  let raw = ''
  for await (const chunk of req) {
    raw += chunk
    if (raw.length > 20 << 20) return send(413, { error: 'job too large' })
  }
  let job
  try {
    job = JSON.parse(raw)
  } catch {
    return send(400, { error: 'bad json' })
  }
  // sub2api re-sends a job after it restarts; wait for the render already running instead of refusing.
  const mine = !current
  if (current && current.out !== job.out) return send(409, { error: 'busy' })
  if (mine) {
    const promise = render(job)
    current = { out: job.out, promise }
    promise.catch(() => {}).finally(() => { if (current && current.promise === promise) current = null })
  }
  const pending = current.promise
  try {
    send(200, await pending)
  } catch (err) {
    if (mine) console.error('render failed:', err)
    send(500, { error: String(err.message || err).slice(0, 1000) })
  }
}).listen(PORT, () => console.log('videorender listening on', PORT, 'root', ROOT, 'origin', ORIGIN))
