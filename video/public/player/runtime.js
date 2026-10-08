/* HiveGPT 视频 player runtime.
 *
 * Runs AI-written scene code inside a sandboxed iframe (player.html: scripts allowed, no network,
 * opaque origin). The parent page owns the clock, audio and subtitles and sends {type:'seek', t};
 * every frame is a pure function of t, so seeking and frame-by-frame export give the same picture.
 * The scene API documented to the model lives in backend/internal/service/video_prompts.go.
 */
(function () {
  'use strict';

  var ICONS = window.__FILM_ICONS__ || {};
  var realRAF = window.requestAnimationFrame.bind(window);
  var SVGNS = 'http://www.w3.org/2000/svg';
  var FADE = 0.5;

  // ---- determinism: one clock, seeded randomness, no timers -----------------------------------
  var clock = 0;
  var BASE_TIME = Date.UTC(2025, 0, 1, 8, 0, 0);
  function mulberry32(a) {
    return function () {
      a |= 0; a = (a + 0x6D2B79F5) | 0;
      var t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }
  Math.random = mulberry32(20250101);
  Date.now = function () { return BASE_TIME + Math.round(clock * 1000); };
  try { performance.now = function () { return clock * 1000; }; } catch (e) { /* read-only in some engines */ }
  var noop = function () { return 0; };
  window.setTimeout = noop; window.setInterval = noop; window.requestAnimationFrame = noop;
  window.fetch = undefined; window.XMLHttpRequest = undefined; window.WebSocket = undefined;

  // ---- easing & math ----------------------------------------------------------------------------
  var ease = {
    linear: function (x) { return x; },
    inQuad: function (x) { return x * x; },
    outQuad: function (x) { return 1 - (1 - x) * (1 - x); },
    inOutQuad: function (x) { return x < 0.5 ? 2 * x * x : 1 - Math.pow(-2 * x + 2, 2) / 2; },
    inCubic: function (x) { return x * x * x; },
    outCubic: function (x) { return 1 - Math.pow(1 - x, 3); },
    inOutCubic: function (x) { return x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2; },
    outQuart: function (x) { return 1 - Math.pow(1 - x, 4); },
    outExpo: function (x) { return x >= 1 ? 1 : 1 - Math.pow(2, -10 * x); },
    inOutSine: function (x) { return -(Math.cos(Math.PI * x) - 1) / 2; },
    outBack: function (x) { var c1 = 1.70158, c3 = c1 + 1; return 1 + c3 * Math.pow(x - 1, 3) + c1 * Math.pow(x - 1, 2); },
    outElastic: function (x) { return x <= 0 ? 0 : x >= 1 ? 1 : Math.pow(2, -10 * x) * Math.sin((x * 10 - 0.75) * (2 * Math.PI) / 3) + 1; },
    outBounce: function (x) {
      var n = 7.5625, d = 2.75;
      if (x < 1 / d) return n * x * x;
      if (x < 2 / d) return n * (x -= 1.5 / d) * x + 0.75;
      if (x < 2.5 / d) return n * (x -= 2.25 / d) * x + 0.9375;
      return n * (x -= 2.625 / d) * x + 0.984375;
    }
  };
  function clamp(x, lo, hi) { return x < lo ? lo : x > hi ? hi : x; }
  function lerp(a, b, k) { return a + (b - a) * k; }
  function progress(t, start, dur) { return dur <= 0 ? (t >= start ? 1 : 0) : clamp((t - start) / dur, 0, 1); }

  // ---- element helpers ---------------------------------------------------------------------------
  function applyProps(el, props) {
    if (!props) return;
    Object.keys(props).forEach(function (k) {
      var v = props[k];
      if (v == null) return;
      if (k === 'class' || k === 'className') el.setAttribute('class', v);
      else if (k === 'style') {
        if (typeof v === 'string') el.style.cssText += ';' + v;
        else Object.keys(v).forEach(function (s) { if (s.indexOf('-') >= 0) el.style.setProperty(s, v[s]); else el.style[s] = v[s]; });
      } else if (k === 'text') el.textContent = v;
      else if (k === 'html') el.innerHTML = v;
      else el.setAttribute(k, v);
    });
  }
  function append(el, children) {
    children.forEach(function (c) {
      if (c == null || c === false) return;
      if (Array.isArray(c)) append(el, c);
      else el.appendChild(typeof c === 'string' || typeof c === 'number' ? document.createTextNode(String(c)) : c);
    });
  }
  var SVG_TAGS = /^(svg|g|path|circle|ellipse|line|polyline|polygon|rect|text|tspan|defs|linearGradient|radialGradient|stop|filter|fe[A-Z]\w*|mask|clipPath|use|pattern|marker|symbol|foreignObject)$/;
  function h(tag, props) {
    var el = SVG_TAGS.test(tag) ? document.createElementNS(SVGNS, tag) : document.createElement(tag);
    var rest = Array.prototype.slice.call(arguments, 2);
    if (props && (props.nodeType || typeof props !== 'object' || Array.isArray(props))) { rest.unshift(props); props = null; }
    applyProps(el, props);
    append(el, rest);
    return el;
  }
  function parseSVG(markup) {
    var text = String(markup).trim();
    var doc = new DOMParser().parseFromString(text, 'image/svg+xml');
    var root = doc.documentElement;
    if (root && root.nodeName.toLowerCase() === 'svg' && !doc.getElementsByTagName('parsererror').length) {
      return document.importNode(root, true);
    }
    // Not well-formed XML (an unescaped & or a quote inside an attribute): let the forgiving HTML
    // parser build it, as a browser would for inline <svg>.
    var tpl = document.createElement('template');
    tpl.innerHTML = text;
    var svg = tpl.content.querySelector('svg');
    if (!svg) throw new Error('S.svg: the markup must be one <svg> element');
    return document.importNode(svg, true);
  }
  function icon(name, opts) {
    opts = opts || {};
    var size = opts.size || 48, color = opts.color || 'currentColor', stroke = opts.stroke || 2;
    var svg = document.createElementNS(SVGNS, 'svg');
    svg.setAttribute('viewBox', '0 0 24 24');
    svg.setAttribute('width', size); svg.setAttribute('height', size);
    svg.setAttribute('fill', opts.fill || 'none'); svg.setAttribute('stroke', color);
    svg.setAttribute('stroke-width', stroke); svg.setAttribute('stroke-linecap', 'round'); svg.setAttribute('stroke-linejoin', 'round');
    var nodes = ICONS[name] || ICONS['circle'] || [['circle', { cx: 12, cy: 12, r: 9 }]];
    nodes.forEach(function (n) {
      var child = document.createElementNS(SVGNS, n[0]);
      Object.keys(n[1]).forEach(function (k) { child.setAttribute(k, n[1][k]); });
      svg.appendChild(child);
    });
    return svg;
  }
  var lengths = new WeakMap();
  function draw(shape, k) {
    if (!shape) return;
    var len = lengths.get(shape);
    if (len == null) {
      try { len = shape.getTotalLength(); } catch (e) { len = 1000; }
      lengths.set(shape, len);
      shape.style.strokeDasharray = len + ' ' + len;
    }
    shape.style.strokeDashoffset = String(len * (1 - clamp(k, 0, 1)));
  }
  var transforms = new WeakMap();
  function setT(el, v) {
    if (!el) return;
    var cur = transforms.get(el) || { x: 0, y: 0, scale: 1, rotate: 0 };
    ['x', 'y', 'scale', 'rotate'].forEach(function (k) { if (v[k] != null) cur[k] = v[k]; });
    transforms.set(el, cur);
    el.style.transform = 'translate(' + cur.x + 'px,' + cur.y + 'px) rotate(' + cur.rotate + 'deg) scale(' + cur.scale + ')';
    if (v.opacity != null) el.style.opacity = String(clamp(v.opacity, 0, 1));
    if (v.blur != null) el.style.filter = v.blur > 0.01 ? 'blur(' + v.blur + 'px)' : '';
  }
  function tween(el, t, start, dur, from, to, fn) {
    var k = (fn || ease.outCubic)(progress(t, start, dur));
    var v = {};
    Object.keys(to).forEach(function (key) {
      var a = from[key] != null ? from[key] : (key === 'scale' || key === 'opacity' ? 1 : 0);
      v[key] = lerp(a, to[key], k);
    });
    setT(el, v);
    return k;
  }
  function typeText(el, text, k) {
    var chars = Array.from(String(text));
    el.textContent = chars.slice(0, Math.ceil(clamp(k, 0, 1) * chars.length)).join('');
  }
  function count(el, from, to, k, opts) {
    opts = opts || {};
    var v = lerp(from, to, clamp(k, 0, 1));
    var s = v.toFixed(opts.decimals || 0);
    if (opts.comma !== false) {
      var parts = s.split('.');
      parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
      s = parts.join('.');
    }
    el.textContent = (opts.prefix || '') + s + (opts.suffix || '');
  }

  // ---- the film ----------------------------------------------------------------------------------
  var film = null;
  var stage = document.getElementById('stage');
  var styleHost = document.head;

  function post(msg) { try { parent.postMessage(Object.assign({ source: 'hivegpt-film' }, msg), '*'); } catch (e) { /* no parent */ } }
  function report(scene, phase, err) {
    var message = err && err.message ? err.message : String(err);
    var line = '';
    var m = err && err.stack && /<anonymous>:(\d+):(\d+)/.exec(err.stack);
    if (m) line = ' (line ' + Math.max(1, +m[1] - 2) + ')';
    post({ type: 'error', scene: scene.id, phase: phase, message: message + line });
  }

  function fit() {
    if (!film) return;
    var sx = window.innerWidth / film.width, sy = window.innerHeight / film.height;
    var s = Math.min(sx, sy);
    stage.style.transform = 'translate(' + (window.innerWidth - film.width * s) / 2 + 'px,' + (window.innerHeight - film.height * s) / 2 + 'px) scale(' + s + ')';
  }

  function wordsHelper(words) {
    words = words || [];
    var joined = '', starts = [];
    words.forEach(function (w, i) { starts.push(joined.length); joined += w.text; });
    return {
      word: function (i) { var w = words[i]; return w ? { start: w.start, end: w.end } : null; },
      at: function (text) {
        var idx = joined.indexOf(String(text).replace(/\s+/g, ''));
        if (idx < 0) idx = joined.indexOf(String(text));
        if (idx < 0) return null;
        for (var i = starts.length - 1; i >= 0; i--) if (starts[i] <= idx) return words[i].start;
        return null;
      }
    };
  }

  function buildScene(sc, index, total) {
    var root = document.createElement('div');
    var id = 'sc' + index + '-' + String(sc.id).replace(/[^\w-]/g, '');
    root.className = 'scene ' + id;
    root.style.cssText = 'position:absolute;inset:0;width:' + film.width + 'px;height:' + film.height + 'px;overflow:hidden;opacity:0;visibility:hidden;';
    stage.appendChild(root);
    var words = wordsHelper(sc.words);
    var canvases = [];
    var S = {
      root: root, id: id, index: index, total: total, width: film.width, height: film.height, duration: sc.duration, theme: film.theme,
      h: h, svg: parseSVG, icon: icon, ease: ease, lerp: lerp, clamp: clamp, p: progress,
      set: setT, tween: tween, draw: draw, type: typeText, count: count,
      random: mulberry32(1000 + index * 7919),
      word: words.word, at: words.at,
      css: function (text) { var st = document.createElement('style'); st.setAttribute('data-scene', id); st.textContent = String(text); styleHost.appendChild(st); },
      canvas: function () {
        var el = document.createElement('canvas');
        var dpr = 2;
        el.width = film.width * dpr; el.height = film.height * dpr;
        el.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;';
        var ctx = el.getContext('2d');
        ctx.scale(dpr, dpr);
        root.appendChild(el);
        canvases.push(ctx);
        return { el: el, ctx: ctx };
      }
    };
    var scene = { id: sc.id, start: 0, duration: sc.duration, root: root, update: null, failed: false };
    Math.random = mulberry32(42 + index * 104729);
    try {
      var fn = new Function('S', '"use strict";\n' + (sc.code || 'return function(){};'));
      var update = fn(S);
      scene.update = typeof update === 'function' ? update : update && typeof update.update === 'function' ? update.update.bind(update) : null;
      if (!scene.update) throw new Error('the scene code must return an update(t) function');
    } catch (err) {
      scene.failed = true;
      report(scene, 'build', err);
      root.appendChild(h('div', { style: 'position:absolute;inset:0;display:flex;align-items:center;justify-content:center;color:#999;font:28px sans-serif', text: '这个分镜出错了，正在修复…' }));
    }
    return scene;
  }

  function load(project) {
    stage.innerHTML = '';
    Array.prototype.slice.call(document.head.querySelectorAll('style[data-scene]')).forEach(function (n) { n.remove(); });
    film = { width: project.width || 1920, height: project.height || 1080, theme: project.theme || {}, loop: !!project.loop, scenes: [] };
    stage.style.width = film.width + 'px';
    stage.style.height = film.height + 'px';
    stage.style.background = film.theme.bg || '#000';
    stage.style.color = film.theme.text || '#fff';
    stage.style.fontFamily = film.theme.font || 'sans-serif';
    document.body.style.background = film.theme.bg || '#000';
    var total = project.scenes.length, at = 0;
    project.scenes.forEach(function (sc, i) {
      var scene = buildScene(sc, i, total);
      scene.start = at;
      at += sc.duration;
      film.scenes.push(scene);
    });
    film.duration = at;
    fit();
    render(0);
    post({ type: 'ready', duration: at });
  }

  function render(t) {
    if (!film) return;
    clock = t;
    var scenes = film.scenes, n = scenes.length;
    if (!n) return;
    if (film.loop && film.duration > 0) t = ((t % film.duration) + film.duration) % film.duration;
    t = clamp(t, 0, Math.max(0, film.duration - 1e-4));
    var cur = 0;
    for (var i = 0; i < n; i++) if (t >= scenes[i].start) cur = i;
    for (var j = 0; j < n; j++) {
      var sc = scenes[j], opacity = 0, local = 0;
      if (j === cur) {
        opacity = 1; local = t - sc.start;
      } else if (j === cur + 1 && sc.start - t < FADE) {
        // Cross-fade: the next scene fades in (on top) over the last FADE seconds of this one.
        opacity = 1 - (sc.start - t) / FADE; local = 0;
      }
      if (opacity <= 0) {
        if (sc.root.style.visibility !== 'hidden') { sc.root.style.visibility = 'hidden'; sc.root.style.opacity = '0'; }
        continue;
      }
      sc.root.style.visibility = 'visible';
      sc.root.style.opacity = String(opacity);
      sc.root.style.zIndex = j === cur ? '1' : '2';
      if (sc.update && !sc.failed) {
        try { sc.update(clamp(local, 0, sc.duration)); } catch (err) { sc.failed = true; report(sc, 'update', err); }
      }
    }
  }

  window.__film = {
    load: load,
    seek: function (t) { render(+t || 0); },
    get duration() { return film ? film.duration : 0; },
    raf: realRAF
  };

  window.addEventListener('message', function (e) {
    var d = e.data;
    if (!d || typeof d !== 'object') return;
    if (d.type === 'load' && d.project) load(d.project);
    else if (d.type === 'seek') render(+d.t || 0);
  });
  window.addEventListener('resize', fit);
  window.addEventListener('error', function (e) { post({ type: 'error', scene: '', phase: 'page', message: e.message || 'error' }); });

  if (window.__FILM_DATA__) {
    load(window.__FILM_DATA__);
  } else {
    post({ type: 'boot' });
  }
})();
