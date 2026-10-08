// 导出 HTML: one self-contained file that plays the work in any browser — the runtime, the scene
// code, the narration (base64) and the subtitles inlined; only the fonts load from the site.
import { audioBlob } from './api'
import { buildSubtitles } from './subtitles'

const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c])
// Keep "</script>" inside inlined code from closing the tag.
const inline = (s) => String(s).replace(/<\/script/gi, '<\\/script')

function toBase64(buffer) {
  let binary = ''
  const bytes = new Uint8Array(buffer)
  for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode.apply(null, bytes.subarray(i, i + 0x8000))
  return btoa(binary)
}

const CONTROLLER = `
(function(){
  var film = window.__film, data = window.__FILM_DATA__, cues = window.__FILM_CUES__ || [];
  var now = window.__realNow, raf = film.raf;
  var dur = film.duration, t = 0, playing = false, wall = 0;
  var starts = [], at = 0; data.scenes.forEach(function(s){ starts.push(at); at += s.duration; });
  var audios = data.scenes.map(function(s){ if (!s.audio) return null; var a = new Audio(s.audio); a.preload = 'auto'; return a; });
  var $ = function(id){ return document.getElementById(id); };
  var btn = $('play'), bar = $('bar'), time = $('time'), sub = $('sub'), big = $('big');
  function fmt(s){ s = Math.max(0, s); return Math.floor(s/60) + ':' + ('0' + Math.floor(s%60)).slice(-2); }
  function syncAudio(){
    audios.forEach(function(a, i){
      if (!a) return;
      var local = t - starts[i];
      var inside = playing && local >= 0 && local < (a.duration || data.scenes[i].duration);
      if (inside) { if (a.paused || Math.abs(a.currentTime - local) > 0.3) { a.currentTime = local; a.play().catch(function(){}); } }
      else if (!a.paused) a.pause();
    });
  }
  function draw(){
    film.seek(t); bar.value = t; time.textContent = fmt(t) + ' / ' + fmt(dur);
    var c = cues.find(function(c){ return t >= c.start && t < c.end; }); sub.textContent = c ? c.text : ''; sub.style.display = c ? '' : 'none';
  }
  function tick(){
    if (!playing) return;
    t = (now() - wall) / 1000;
    if (t >= dur) { if (data.loop) { t = 0; wall = now(); } else { t = dur; pause(); draw(); return; } }
    draw(); syncAudio(); raf(tick);
  }
  function play(){ if (t >= dur - 0.05) t = 0; playing = true; wall = now() - t * 1000; btn.textContent = '❚❚'; big.style.display = 'none'; syncAudio(); raf(tick); }
  function pause(){ playing = false; btn.textContent = '▶'; syncAudio(); }
  btn.onclick = function(){ playing ? pause() : play(); };
  big.onclick = play;
  $('screen').onclick = function(){ playing ? pause() : play(); };
  bar.max = dur; bar.oninput = function(){ t = +bar.value; wall = now() - t * 1000; draw(); syncAudio(); };
  $('fs').onclick = function(){ document.fullscreenElement ? document.exitFullscreen() : document.documentElement.requestFullscreen(); };
  draw();
})();`

export async function exportHtml(project, audioUrl, withKey) {
  const spec = project.spec
  const [runtime, icons] = await Promise.all([fetch('/player/runtime.js').then((r) => r.text()), fetch('/player/icons.js').then((r) => r.text())])
  const scenes = await Promise.all(
    spec.scenes.map(async (sc) => {
      let audio = ''
      const url = audioUrl(sc)
      if (url) {
        try {
          audio = `data:audio/mpeg;base64,${toBase64(await audioBlob(url, withKey))}`
        } catch {
          audio = ''
        }
      }
      return { id: sc.id, duration: sc.duration, code: sc.code || '', words: sc.audio?.words || [], audio }
    })
  )
  let at = 0
  const starts = spec.scenes.map((sc) => {
    const s = at
    at += sc.duration
    return s
  })
  const cues = buildSubtitles(spec.scenes, starts)
  const data = { width: spec.width, height: spec.height, theme: spec.theme, loop: !!spec.loop, scenes }
  const origin = location.origin
  const html = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(project.title)} · HiveGPT 视频</title>
<link rel="stylesheet" href="${origin}/fonts/fonts.css">
<style>
html,body{margin:0;height:100%;background:#000;overflow:hidden;font-family:"PingFang SC","Microsoft YaHei",sans-serif}
#stage{position:absolute;left:0;top:0;transform-origin:0 0;overflow:hidden}
#stage *{box-sizing:border-box}
#screen{position:fixed;inset:0;z-index:5;cursor:pointer}
#sub{position:fixed;left:50%;bottom:11%;transform:translateX(-50%);z-index:6;max-width:84%;padding:4px 14px;border-radius:8px;background:rgba(0,0,0,.65);color:#fff;font-size:clamp(14px,2.6vw,30px);text-align:center;pointer-events:none}
#ui{position:fixed;left:0;right:0;bottom:0;z-index:7;display:flex;align-items:center;gap:12px;padding:10px 16px;background:linear-gradient(transparent,rgba(0,0,0,.7));color:#fff;font-size:13px;opacity:0;transition:opacity .2s}
body:hover #ui{opacity:1}
#ui button{background:none;border:0;color:#fff;font-size:15px;cursor:pointer}
#bar{flex:1;accent-color:#f97316}
#big{position:fixed;left:50%;top:50%;z-index:8;width:72px;height:72px;margin:-36px 0 0 -36px;border:0;border-radius:50%;background:rgba(255,255,255,.92);color:#ea580c;font-size:28px;cursor:pointer}
#brand{position:fixed;right:14px;top:10px;z-index:7;color:rgba(255,255,255,.55);font-size:12px;text-decoration:none}
</style>
</head>
<body>
<div id="stage"></div>
<div id="screen"></div>
<div id="sub" style="display:none"></div>
<button id="big" aria-label="播放">▶</button>
<div id="ui"><button id="play">▶</button><input id="bar" type="range" min="0" step="0.01" value="0"><span id="time"></span><button id="fs">⛶</button></div>
<a id="brand" href="${origin}" target="_blank" rel="noopener">HiveGPT 视频</a>
<script>window.__realNow = performance.now.bind(performance);window.__FILM_DATA__ = ${inline(JSON.stringify(data))};window.__FILM_CUES__ = ${inline(JSON.stringify(cues))};</script>
<script>${inline(icons)}</script>
<script>${inline(runtime)}</script>
<script>${inline(CONTROLLER)}</script>
</body>
</html>
`
  const blob = new Blob([html], { type: 'text/html;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${(project.title || 'hivegpt-video').replace(/[\\/:*?"<>|\s]+/g, '-')}.html`
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(a.href), 4000)
}
