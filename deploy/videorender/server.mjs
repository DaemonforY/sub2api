// HiveGPT 视频 MP4 renderer. sub2api posts one job at a time; this loads the video site's
// /render.html (the sandboxed player plus subtitles, landscape or portrait), seeks the film frame by
// frame, pipes screenshots into ffmpeg with the narration mixed in, and writes the MP4 and a cover
// next to the project's audio on the shared data volume.
//
// Speed: seeking costs a few ms; drawing and capturing the frame is nearly all of it. The box has no
// GPU, and SwiftShader (a GPU emulated on the CPU) made Chromium's gpu-process the bottleneck. Plain
// CPU rasterization draws the same frames 2.4–3.3× faster, three.js included (WebGL falls back to
// Chromium's software implementation). RENDER_GPU=swiftshader brings the old flags back.
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
// x264 'slow' became the bottleneck once frames came faster; 'medium' encodes ~2× faster at CRF 16,
// 'veryfast' again much faster — at CRF 16 the difference is file size, not visible quality.
const PRESET = process.env.FFMPEG_PRESET || 'veryfast'
// Bitrate cap (Mbps at 1920×1080, scaled by pixel count): CRF 16 alone sent a busy particle film to
// 15.9 Mbps (282 MB for 2:22) that users in China, at 5–15 KB/s to this server, could not download.
// Most films average ~2 Mbps and never reach the cap. 0 = no cap.
const MAXRATE_1080 = Math.max(0, Number(process.env.FFMPEG_MAXRATE_1080 ?? 6))
// Pages capturing frames at the same time (see render()).
const WORKERS = Math.min(6, Math.max(1, Number(process.env.RENDER_WORKERS || 3)))
const PRESETS = ['ultrafast', 'superfast', 'veryfast', 'faster', 'fast', 'medium', 'slow']
// Screenshots are re-encoded by x264, so q85 costs nothing visible and moves 60% fewer bytes.
const JPEG_QUALITY = Math.min(100, Math.max(50, Number(process.env.JPEG_QUALITY || 85)))
const GPU = process.env.RENDER_GPU === 'swiftshader' ? 'swiftshader' : 'cpu'
const GPU_FLAGS = GPU === 'swiftshader'
  ? ['--use-angle=swiftshader', '--enable-unsafe-swiftshader']
  : ['--disable-gpu', '--disable-gpu-compositing', '--disable-gpu-rasterization']

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

async function render(job, maxWorkers = WORKERS) {
  const { setup, fps = 60, width, height } = job
  // scale < 1 captures the same page (laid out at width × height) with fewer pixels, e.g. 2/3 for a
  // 720p file of a 1080p layout: subtitles and titles keep their places, drawing costs less.
  const scale = job.scale > 0 && job.scale <= 1 ? job.scale : 1
  const outW = Math.round((width * scale) / 2) * 2, outH = Math.round((height * scale) / 2) * 2
  const preset = PRESETS.includes(job.preset) ? job.preset : PRESET
  const out = inside(job.out)
  const cover = job.cover ? inside(job.cover) : ''
  const progress = job.progress ? inside(job.progress) : ''
  if (!job.url || !job.url.startsWith(ORIGIN + '/')) throw new Error('render page must be on ' + ORIGIN)
  if (!(width > 0 && height > 0 && width <= 3840 && height <= 3840)) throw new Error('bad size')
  fs.mkdirSync(path.dirname(out), { recursive: true })
  const note = (data) => progress && fs.writeFileSync(progress, JSON.stringify({ ...data, at: Date.now() }))
  note({ stage: 'loading', frame: 0, frames: 0 })

  // One browser per page: inside a single browser only the foreground tab keeps producing frames on
  // Linux, so background tabs' screenshots hung until the protocol timeout.
  const browsers = []
  const launch = async () => {
    const b = await puppeteer.launch({
      executablePath: CHROME,
      headless: 'new',
      protocolTimeout: 120000,
      args: ['--no-sandbox', '--disable-dev-shm-usage', '--hide-scrollbars', '--mute-audio', '--force-color-profile=srgb',
        '--disable-background-timer-throttling', '--disable-renderer-backgrounding', '--disable-backgrounding-occluded-windows',
        ...GPU_FLAGS, `--window-size=${width},${height}`]
    })
    browsers.push(b)
    return b
  }
  const wav = out.replace(/\.mp4$/, '') + '.narration.wav'
  const tmp = out + '.part'
  const segFiles = []
  const errors = []
  const encoders = []
  let failed = null // the first worker error stops the others
  // A loaded render page: the frame is drawn by seeking the player and setting the subtitle.
  const openPage = async () => {
    const browser = await launch()
    const page = (await browser.pages())[0] || (await browser.newPage())
    await page.setViewport({ width, height, deviceScaleFactor: scale })
    page.on('pageerror', (e) => errors.push('page: ' + e.message))
    await page.goto(job.url, { waitUntil: 'networkidle0', timeout: 60000 })
    await page.evaluate((s) => window.__setup(s), setup)
    await page.waitForFunction('window.__ready === true', { timeout: 120000 })
    const player = page.frames().find((f) => f.url().includes('/player.html'))
    if (!player) throw new Error('player frame not found')
    await page.evaluate(() => document.fonts.ready)
    await player.evaluate(() => document.fonts.ready)
    const cdp = await page.createCDPSession()
    return {
      page,
      shot: async () => Buffer.from((await cdp.send('Page.captureScreenshot', { format: 'jpeg', quality: JPEG_QUALITY, optimizeForSpeed: true })).data, 'base64'),
      seek: async (t) => {
        await player.evaluate((t) => window.__film.seek(t), t)
        await page.evaluate((t) => window.__subtitle(t), t)
      }
    }
  }
  const maxrate = MAXRATE_1080 > 0 ? Math.round((MAXRATE_1080 * 1000 * outW * outH) / (1920 * 1080)) : 0
  const rateCap = maxrate ? ['-maxrate', `${maxrate}k`, '-bufsize', `${maxrate * 2}k`] : []
  // Encodes one run of frames (video only) into file.
  const encoder = (file) => {
    const ff = spawn('ffmpeg', ['-y', '-loglevel', 'error', '-threads', THREADS,
      '-f', 'image2pipe', '-framerate', String(fps), '-c:v', 'mjpeg', '-i', '-',
      ...(scale < 1 ? ['-vf', `scale=${outW}:${outH}`] : []), // fractional scales may round a pixel off
      '-c:v', 'libx264', '-preset', preset, '-tune', 'animation', '-crf', '16', ...rateCap, '-profile:v', 'high', '-level', '4.2',
      '-pix_fmt', 'yuv420p', '-r', String(fps), '-g', String(fps * 2),
      '-colorspace', 'bt709', '-color_primaries', 'bt709', '-color_trc', 'bt709', '-an', '-f', 'mp4', file], { stdio: ['pipe', 'ignore', 'pipe'] })
    let ffErr = ''
    ff.stderr.on('data', (d) => (ffErr = (ffErr + d).slice(-2000)))
    const done = new Promise((resolve, reject) => {
      ff.on('error', reject)
      ff.on('close', (code) => (code === 0 ? resolve() : reject(new Error('ffmpeg exited ' + code + ': ' + ffErr))))
    })
    ff.stdin.on('error', () => {}) // a dying ffmpeg is reported by `done`
    done.catch(() => {}) // …unless its worker already failed, killed it and never awaits it
    let closed = false
    ff.on('close', () => (closed = true))
    encoders.push({ kill: () => { if (!closed) ff.kill('SIGKILL') } })
    return {
      write: async (buf) => {
        if (!ff.stdin.write(buf)) await Promise.race([new Promise((r) => ff.stdin.once('drain', r)), done])
      },
      end: () => { ff.stdin.end(); return done }
    }
  }
  try {
    const first = await openPage()
    const full = await first.page.evaluate('window.__TOTAL')
    if (!(full > 0)) throw new Error('empty film')
    const total = job.limit > 0 ? Math.min(full, job.limit) : full // limit: render only the first seconds (tests)
    const frames = Math.round(total * fps)

    note({ stage: 'audio', frame: 0, frames })
    await mixNarration(job.audio || [], total, wav)

    // Capturing a frame mostly waits for Chromium's next compositor frame (~40 ms on any page, even
    // an unchanged one), not for the CPU, so browsers capturing side by side multiply the throughput
    // (1 → 3 pages: 28 → 72 frames/s on a laptop). The film is cut into WORKERS runs of frames, each
    // encoded on its own, then joined without re-encoding and muxed with the narration.
    const workers = Math.max(1, Math.min(maxWorkers, Math.floor(frames / (fps * 2)) || 1))
    const pages = [first]
    for (let k = 1; k < workers; k++) pages.push(await openPage())
    const per = Math.ceil(frames / workers)
    let doneFrames = 0
    const started = Date.now()
    // allSettled, not all: a failed worker must not leave the others capturing into browsers that the
    // finally block closes, nor their ffmpeg processes waiting on stdin (each held ~370 MB).
    const results = await Promise.allSettled(pages.map(async (pg, k) => {
      const from = k * per, to = Math.min(frames, from + per)
      if (from >= to) return
      const file = `${out}.seg${k}.mp4`
      segFiles[k] = file
      const enc = encoder(file)
      try {
        for (let i = from; i < to; i++) {
          if (failed) throw failed
          await pg.seek(i / fps)
          await enc.write(await pg.shot())
          if (++doneFrames % 30 === 0) note({ stage: 'frames', frame: doneFrames, frames, fps: doneFrames / Math.max(1, (Date.now() - started) / 1000) })
        }
        await enc.end()
      } catch (err) {
        if (!failed) {
          failed = err
          // Close the browsers now so the other workers' pending captures fail at once instead of
          // waiting out the protocol timeout.
          for (const b of browsers) b.close().catch(() => {})
        }
        throw err
      }
    }))
    const bad = results.find((r) => r.status === 'rejected')
    if (bad) throw failed || bad.reason
    const list = out + '.segments.txt'
    fs.writeFileSync(list, segFiles.filter(Boolean).map((f) => `file '${f.replace(/'/g, "'\\''")}'`).join('\n') + '\n')
    segFiles.push(list)
    await run('ffmpeg', ['-y', '-loglevel', 'error', '-f', 'concat', '-safe', '0', '-i', list, '-i', wav,
      '-map', '0:v', '-map', '1:a', '-c:v', 'copy', '-c:a', 'aac', '-b:a', '256k', '-ar', '48000', '-ac', '2',
      '-t', String(total), '-movflags', '+faststart', '-f', 'mp4', tmp])
    fs.renameSync(tmp, out)
    if (cover) {
      await first.seek(Math.min(total, Math.max(0, job.coverAt ?? total * 0.3)))
      fs.writeFileSync(cover, await first.shot())
    }
    for (const pg of pages) errors.push(...(await pg.page.evaluate(() => (window.__errors || []).map((e) => `${e.scene || ''} ${e.phase || ''} ${e.message}`))))
    const size = fs.statSync(out).size
    note({ stage: 'done', frame: frames, frames })
    const seconds = (Date.now() - started) / 1000
    return { size, duration: total, frames, seconds: Math.round(seconds), fps: +(frames / Math.max(seconds, 0.001)).toFixed(2),
      gpu: GPU, preset, maxrate_kbps: maxrate, scale, workers, width: outW, height: outH, errors: [...new Set(errors)].slice(0, 20) }
  } finally {
    for (const e of encoders) e.kill()
    await Promise.all(browsers.map((b) => b.close().catch(() => {})))
    fs.rmSync(wav, { force: true })
    fs.rmSync(tmp, { force: true })
    for (const f of segFiles) if (f) fs.rmSync(f, { force: true })
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
    // Several pages need several times the memory; a heavy film (full-frame grain, big canvases) can
    // run the container out of it and Chromium is killed mid-capture. Try again with one page.
    const promise = render(job).catch((err) => {
      if (WORKERS <= 1) throw err
      console.error('render failed with', WORKERS, 'pages, retrying with one:', err.message || err)
      return render(job, 1)
    })
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
}).listen(PORT, () => console.log('videorender listening on', PORT, 'root', ROOT, 'origin', ORIGIN, 'gpu', GPU, 'preset', PRESET, 'maxrate1080', MAXRATE_1080, 'jpeg', JPEG_QUALITY, 'workers', WORKERS))
