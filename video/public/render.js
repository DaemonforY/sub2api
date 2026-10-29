// Render page for MP4 export (see render.html). __setup({layout, width, height, title, payload, scenes, subtitles})
// builds the frame: the sandboxed player plus subtitles, either the film filling the frame (landscape,
// like the site's player) or a portrait 1080×1920 frame with the title above the film and large
// subtitles below. window.__ready / __TOTAL tell the renderer when the film is loaded.
;(function () {
  window.__ready = false
  window.__errors = []

  // Subtitles: a copy of src/lib/subtitles.js (keep the two in sync).
  var BREAK = /[，。！？；：、,.!?;:\n]/
  var MAX = 22
  var CJK = /[\u3400-\u9fff\uf900-\ufaff]/
  function clauses(text) {
    var out = [], cur = ''
    var chars = Array.from(text)
    chars.forEach(function (ch, i) {
      cur += ch
      var next = chars[i + 1] || ''
      var canCut = !next || CJK.test(next) || (cur.length >= MAX * 2 && next === ' ')
      if (BREAK.test(ch) || (cur.length >= MAX && canCut)) {
        if (cur.trim()) out.push(cur)
        cur = ''
      }
    })
    if (cur.trim()) out.push(cur)
    return out
  }
  function tidy(s) { return s.replace(/^[\s，。！？；：、,.!?;:]+|[\s，。、,;；:：]+$/g, '').trim() }
  function buildSubtitles(scenes, starts) {
    var cues = []
    scenes.forEach(function (sc, i) {
      var narration = (sc.narration || '').trim()
      if (!narration) return
      var begin = starts[i] || 0
      var words = (sc.audio && sc.audio.words) || []
      var parts = clauses(narration)
      var charTime = [], pos = 0
      words.forEach(function (w) {
        var at = narration.indexOf(w.text, pos)
        if (at < 0) return
        charTime.push([at, w.start])
        pos = at + w.text.length
      })
      function timeAt(offset) {
        var best = null
        for (var k = 0; k < charTime.length; k++) {
          if (charTime[k][0] <= offset) best = charTime[k][1]
          else break
        }
        if (best != null) return best
        return (offset / narration.length) * ((sc.audio && sc.audio.duration) || sc.duration || 0)
      }
      var offset = 0
      var local = parts.map(function (p) {
        var start = timeAt(offset)
        offset += p.length
        return { start: start, text: tidy(p) }
      })
      local.forEach(function (c, j) {
        if (!c.text) return
        var end = j + 1 < local.length ? local[j + 1].start : (sc.audio && sc.audio.duration) || sc.duration
        cues.push({ start: begin + c.start, end: begin + Math.max(c.start + 0.3, end), text: c.text })
      })
    })
    return cues
  }

  var cues = [], subEl = null, frame = null, current = null

  function el(tag, cls, text) {
    var n = document.createElement(tag)
    if (cls) n.className = cls
    if (text) n.textContent = text
    return n
  }

  window.__setup = function (cfg) {
    var root = document.getElementById('root')
    var W = cfg.width, H = cfg.height, theme = (cfg.payload && cfg.payload.theme) || {}
    var portrait = cfg.layout === 'portrait'
    var fw = cfg.payload.width || 1920, fh = cfg.payload.height || 1080
    // Only a landscape film is wrapped (title above, film in the middle); a portrait film fills the frame
    // and gets the player's subtitles at the bottom, in the strip its scenes keep free.
    var wrapped = portrait && fw / fh > 0.75
    root.className = wrapped ? 'portrait' : (H > W ? 'landscape tall' : 'landscape')
    root.style.width = W + 'px'
    root.style.height = H + 'px'
    root.style.background = theme.bg || '#000'
    root.style.color = theme.text || '#fff'
    root.style.fontFamily = theme.font || "'Noto Sans CJK SC', sans-serif"

    var starts = [], at = 0
    cfg.scenes.forEach(function (sc) { starts.push(at); at += sc.duration })
    // The owner's subtitle style (spec.subtitles, see src/lib/subtitles.js): none, size, position.
    var st = cfg.subtitles || {}
    var scale = { s: 0.8, l: 1.3 }[st.size] || 1
    cues = st.off ? [] : buildSubtitles(cfg.scenes, starts)

    frame = document.createElement('iframe')
    frame.setAttribute('sandbox', 'allow-scripts')
    frame.setAttribute('title', '视频画面')
    if (wrapped) {
      // A landscape film in a portrait frame: title on top, film across the middle, subtitles below.
      var h = Math.round((W * fh) / fw)
      frame.style.cssText = 'left:0;width:' + W + 'px;top:' + Math.round(H * 0.33) + 'px;height:' + h + 'px'
      var head = el('div', 'head')
      var title = String(cfg.title || '')
      var cut = title.search(/[：:｜|]/)
      var series = cut > 0 ? title.slice(0, cut).trim() : ''
      var main = cut > 0 ? title.slice(cut + 1).trim() : title
      if (series) {
        var s = el('div', 'series', series)
        s.style.color = theme.accent || theme.primary || theme.text || '#fff'
        head.appendChild(s)
      }
      var mainEl = el('div', 'title', main)
      if (theme.display) mainEl.style.fontFamily = theme.display
      head.appendChild(mainEl)
      root.appendChild(head)
      var sub = el('div', 'sub')
      sub.style.top = Math.round(H * 0.33 + h + 120) + 'px'
      subEl = el('span')
      subEl.style.fontSize = Math.round(60 * scale) + 'px'
      sub.appendChild(subEl)
      root.appendChild(sub)
      if (cfg.brand) root.appendChild(el('div', 'brand', cfg.brand))
    } else {
      frame.style.cssText = 'left:0;top:0;width:' + W + 'px;height:' + H + 'px'
      var sub2 = el('div', 'sub')
      subEl = el('span')
      subEl.style.fontSize = Math.max(12, Math.round((H > W ? W / 24 : Math.min(W, H * 16 / 9) / 34) * scale)) + 'px'
      if (st.position === 'raised') sub2.style.bottom = '16%'
      else if (st.position === 'top') { sub2.style.bottom = 'auto'; sub2.style.top = H > W ? '6%' : '4%' }
      sub2.appendChild(subEl)
      root.appendChild(sub2)
    }
    root.insertBefore(frame, root.firstChild)

    window.addEventListener('message', function (e) {
      if (e.source !== frame.contentWindow) return
      var d = e.data || {}
      if (d.type === 'boot') frame.contentWindow.postMessage({ type: 'load', project: cfg.payload }, '*')
      else if (d.type === 'ready') { window.__TOTAL = d.duration; window.__ready = true }
      else if (d.type === 'error') window.__errors.push(d)
    })
    frame.src = '/player.html'
  }

  window.__subtitle = function (t) {
    var cue = null
    for (var i = 0; i < cues.length; i++) if (t >= cues[i].start && t < cues[i].end) { cue = cues[i]; break }
    var text = cue ? cue.text : ''
    if (text === current) return
    current = text
    subEl.textContent = text
    subEl.style.visibility = text ? 'visible' : 'hidden'
  }
})()
