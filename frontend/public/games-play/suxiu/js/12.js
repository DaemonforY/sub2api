'use strict';
// ════════════════════════════════════════════════════════════════
//  主程序：场景、三种模式（字 / 画 / 纹）、绣的调度、针和线
// ════════════════════════════════════════════════════════════════
const FABRICS = [
  {name:'月白', c:[233,220,202]},
  {name:'秋香', c:[214,196,140]},
  {name:'湖蓝', c:[52,92,110]},
  {name:'胭脂', c:[118,36,48]},
  {name:'玄青', c:[30,29,34]},
];
const PALETTES = [
  {name:'胭脂', a:[118,16,34],  b:[226,96,116]},
  {name:'黛青', a:[22,50,84],   b:[112,156,200]},
  {name:'石绿', a:[22,80,66],   b:[124,190,152]},
  {name:'藤黄', a:[150,88,12],  b:[244,198,90]},
  {name:'月白', a:[170,178,188],b:[252,252,250]},
  {name:'墨',   a:[14,14,16],   b:[84,84,90]},
];
const CATS = ['鸟','山','水','兽','日月','植物','花','其他'];
// 场景图里绢面圆的位置（按原图归一化，tools 里拟合出来的）
const ROOM = {src:'assets/room.jpg', iw:3840, ih:2160, cx:0.49830, cy:0.45693, r:0.34806};
// 桌上三只线轴（原图归一化坐标）：点一下换丝线，线就从那只线轴牵过来
const SPOOLS = [
  {x:0.139, y:0.574, w:0.022, h:0.078, pal:2},   // 绿
  {x:0.163, y:0.574, w:0.022, h:0.078, pal:0},   // 粉
  {x:0.186, y:0.574, w:0.022, h:0.078, pal:3},   // 黄
];

const S = {
  mode:'text', fabric:FABRICS[0], palette:PALETTES[0],
  brushW:12, motif:null, motifSize:300, cat:'鸟', sel:null, drag:null, ghost:{rot:0, fx:false, fy:false}, fontC:108, layout:'flow', frame:null,
  items:[], W:0, H:0, hoop:null, img:{x:0, y:0, s:1}, spool:1, hoverSpool:-1,
};
let MOTIFS = [];
const stage = document.getElementById('stage');
const fx = document.getElementById('fx'), fctx = fx.getContext('2d');
const bgImg = document.getElementById('bg');
const ime = document.getElementById('ime');
let seed = 1;
const newRnd = () => mulberry(seed++ * 9973 + 17);

// ───────── 字的数据 ─────────
const DATA_URL = ch => `https://cdn.jsdelivr.net/npm/hanzi-writer-data@2.0.1/${encodeURIComponent(ch)}.json`;
const hanCache = new Map();
// 笔画数据先找本地小包（data/hanzi/<码位>>5>.js，gzip+base64，用 <script> 加载，双击打开也能读），找不到再上网取
const hzWait = new Map(), hzPacks = new Map();
window.__hzPack = (id, b64) => { const cb = hzWait.get(id); if(cb) cb(b64); };
function loadPack(id){
  if(!hzPacks.has(id)) hzPacks.set(id, new Promise(res => {
    const decode = async b64 => {
      try{
        const bin = Uint8Array.from(atob(b64), c => c.charCodeAt(0));
        res(JSON.parse(await new Response(new Blob([bin]).stream().pipeThrough(new DecompressionStream('gzip'))).text()));
      }catch(e){ res(null); }
    };
    // 单文件版：笔画包已经嵌在页面里
    if(window.__HZ){ window.__HZ[id] ? decode(window.__HZ[id]) : res(null); return; }
    const s = document.createElement('script'); s.src = `data/hanzi/${id}.js`;
    hzWait.set(id, b64 => { decode(b64); s.remove(); });
    s.onerror = () => { s.remove(); res(null); };
    document.head.appendChild(s);
  }));
  return hzPacks.get(id);
}
const loadHan = ch => {
  if(!hanCache.has(ch)) hanCache.set(ch, loadPack((ch.codePointAt(0) >> 5).toString(16))
    .then(p => p ? (p[ch] || null) : fetch(DATA_URL(ch)).then(r => r.ok ? r.json() : null))
    .catch(() => null));
  return hanCache.get(ch);
};
const isHan = ch => /\p{Script=Han}/u.test(ch);

// ───────── 圆内竖排：从右往左，每列按圆的弦长决定能放几个字 ─────────
const TXT = {pad:40, gap:1.06};
const SIZES = [{n:'小', C:80}, {n:'中', C:108}, {n:'大', C:140}, {n:'特大', C:184}];
// 版式 frame = {C:字格大小, cols:[{x:左边, top:顶, rows:能放几个字}]}
function flowFrame(C){
  const cols = [], R = HOOP_R - TXT.pad;
  for(let i = 0; ; i++){
    const xc = (R - C*0.5) - i*C*TXT.gap;
    if(xc < -(R - C*0.5)) break;
    const xe = Math.abs(xc) + C*0.5;
    if(xe >= R) continue;
    const half = Math.sqrt(R*R - xe*xe), rows = Math.floor(2*half/C);
    if(rows < 1) continue;
    cols.push({x:xc - C*0.5, top:-rows*C*0.5, rows});
  }
  return {C, cols};
}
// 按换行把字分成几段（每段至少占一列）
function textRuns(){
  const runs = [[]];
  for(const it of S.items){
    if(it.kind === 'nl') runs.push([]);
    else if(it.kind === 'ch' || it.kind === 'sp') runs[runs.length-1].push(it);
  }
  while(runs.length > 1 && !runs[runs.length-1].length) runs.pop();
  return runs;
}
// 自动排版：整块文字居中放进圆里，列尽量长；放不下返回 null
function centerFrame(C){
  const R = HOOP_R - TXT.pad, runs = textRuns();
  const maxLen = Math.max(1, ...runs.map(r => r.length));
  const cap = Math.floor(2*R/C);
  for(let rows = Math.min(maxLen, cap); rows >= 1; rows--){
    const ncols = runs.reduce((a, r) => a + Math.max(1, Math.ceil(r.length/rows)), 0);
    const bw = ncols*C*TXT.gap - C*(TXT.gap - 1), top = -rows*C*0.5, x0 = bw/2 - C;
    const cols = []; let ok = true;
    for(let i = 0; i < ncols && ok; i++){
      const x = x0 - i*C*TXT.gap;
      for(const cx of [x, x + C]) for(const cy of [top, top + rows*C]) if(Math.hypot(cx, cy) > R) ok = false;
      cols.push({x, top, rows});
    }
    if(!ok) continue;
    // 往左再留几列，接着打字时有地方放
    for(let i = ncols; ; i++){
      const x = x0 - i*C*TXT.gap, xe = Math.max(Math.abs(x), Math.abs(x + C));
      if(x < -R || xe >= R) break;
      const half = Math.sqrt(R*R - xe*xe), t2 = Math.max(top, -half), r2 = Math.floor((Math.min(top + rows*C, half) - t2)/C);
      if(r2 < 1) break;
      cols.push({x, top:t2, rows:r2});
    }
    return {C, cols};
  }
  return null;
}
function layoutText(){
  const {C, cols} = S.frame;
  let ci = 0, ri = 0;
  for(const it of S.items){
    if(it.kind === 'nl'){ ci++; ri = 0; continue; }
    if(it.kind !== 'ch' && it.kind !== 'sp') continue;
    while(ci < cols.length && ri >= cols[ci].rows){ ci++; ri = 0; }
    it.cell = ci < cols.length ? {x:cols[ci].x, y:cols[ci].top + ri*C, s:C} : null;
    ri++;
  }
  while(ci < cols.length && ri >= cols[ci].rows){ ci++; ri = 0; }
  return ci < cols.length ? {x:cols[ci].x, y:cols[ci].top + ri*C, s:C} : null;
}
let cursor = null;

// ───────── 往绣面上加东西 ─────────
function addItem(it){ S.items.push(it); afterChange(); }
function afterChange(){
  cursor = layoutText();
  if(cursor){ const p = Render.toScreen(cursor.x, cursor.y); ime.style.left = p.x + 'px'; ime.style.top = p.y + 'px'; }
  document.getElementById('hint').style.opacity = S.items.length || S.mode === 'dance' ? 0 : 1;
}
function addChar(ch){
  if(ch === '\n' || ch === '\r'){ addItem({kind:'nl', done:true}); return; }
  if(ch === ' ' || ch === '　'){ addItem({kind:'sp', done:true}); return; }
  const it = {kind:'ch', ch, pal:S.palette, parts:null, ready:false, pi:0, k:0, done:false};
  addItem(it);
  stitchChar(it);
}
function stitchChar(it){
  const cell = it.cell, ch = it.ch, gen = it.gen = (it.gen || 0) + 1;
  if(!cell){ it.done = true; it.ready = true; it.parts = []; return; }
  const rnd = newRnd();
  if(isHan(ch)){
    loadHan(ch).then(d => {
      if(it.gen !== gen) return;                       // 这期间又重排过，以新的为准
      if(d && d.strokes){
        const k = cell.s*0.9/1024, ox = cell.x + cell.s*0.05, oy = cell.y + cell.s*0.05;
        const map = (x, y) => ({x:ox + x*k, y:oy + (900 - y)*k});
        it.parts = d.strokes.map((p, i) => genSatin(new Path2D(p), d.medians[i], {map, unit:k, pal:it.pal, rnd}));
      } else it.parts = [genGlyphKnots(ch, cell, {pal:it.pal, rnd})];
      it.ready = true;
    });
  } else { it.parts = [genGlyphKnots(ch, cell, {pal:it.pal, rnd})]; it.ready = true; }
}
// 版式变了：字拆掉重绣（画和纹样留着）
function restitchText(){
  afterChange();
  for(const it of S.items) if(it.kind === 'ch'){ it.parts = null; it.ready = false; it.done = false; it.pi = 0; it.k = 0; }
  redrawAll();
  for(const it of S.items) if(it.kind === 'ch') stitchChar(it);
}
function setFontSize(C){
  S.fontC = C;
  S.frame = S.layout === 'center' ? (centerFrame(C) || flowFrame(C)) : flowFrame(C);
  restitchText(); syncTextUI();
}
function autoLayout(){
  let C = S.fontC, f = centerFrame(C);
  if(!f) for(const z of [...SIZES].reverse()) if(z.C < C && (f = centerFrame(z.C))){ C = z.C; break; }
  if(!f){ C = 56; f = centerFrame(C) || flowFrame(C); }
  S.layout = 'center'; S.fontC = C; S.frame = f;
  restitchText(); syncTextUI();
}
function flowLayout(){ S.layout = 'flow'; S.frame = flowFrame(S.fontC); restitchText(); syncTextUI(); }
function syncTextUI(){
  document.querySelectorAll('.fs').forEach(b => b.classList.toggle('on', +b.dataset.c === S.fontC));
  document.getElementById('autoBtn').classList.toggle('on', S.layout === 'center');
  document.getElementById('flowBtn').classList.toggle('on', S.layout !== 'center');
}
function addBrush(points){
  const st = brushToStroke(points, S.brushW);
  if(!st) return;
  const it = {kind:'brush', parts:[genSatin(st.path, st.median, {map:(x, y) => ({x, y}), unit:1, pal:S.palette, rnd:newRnd()})],
              ready:true, pi:0, k:0, done:false};
  addItem(it);
}
const regionPal = col => {
  const lum = (col[0]*0.3 + col[1]*0.59 + col[2]*0.11)/255;
  return {a:col.map(v => v*(lum > 0.8 ? 0.84 : 0.7)), b:col.map(v => Math.min(255, v*1.1 + 14))};
};
// 纹样变换 T = {cx, cy, size, rot:弧度, fx:水平翻转, fy:垂直翻转}；纹样本地坐标以中心为原点、最长边为 1
function motifXf(T, noFlip){
  const c = Math.cos(T.rot), sn = Math.sin(T.rot);
  const sx = (!noFlip && T.fx ? -1 : 1)*T.size, sy = (!noFlip && T.fy ? -1 : 1)*T.size;
  return (x, y) => { const a = x*sx, b = y*sy; return {x:T.cx + a*c - b*sn, y:T.cy + a*sn + b*c}; };
}
function motifPolys(m, T){
  const f = motifXf(T);
  return m.regions.map(r => r.polys.map(p => { const out = []; for(let i = 0; i < p.length; i += 2) out.push(f(p[i], p[i+1])); return out; }));
}
// 同一个纹样每次重算都用同一颗随机种子：改位置、角度后针法的手工感不变
function genMotifParts(it){
  const {m, T} = it, rnd = mulberry(it.seed), polys = motifPolys(m, T);
  return m.regions.map((r, i) => {
    let a = r.major + Math.PI/2 + (r.aspect > 2.2 ? -0.3 : 0.55) + (rnd()-0.5)*0.2;
    if(T.fx) a = Math.PI - a;
    if(T.fy) a = -a;
    return genFill(polys[i], {angle:a + T.rot, pal:regionPal(r.col), rnd});
  });
}
function addMotif(m, T){
  const it = {kind:'motif', m, T:{...T}, seed:(seed++)*7919 + 3, ready:true, pi:0, k:0, done:false};
  it.parts = genMotifParts(it);
  addItem(it);
  select(it);
  return it;
}
// 改了变换：针脚重算，直接整面重画（不再一针针绣）
function updateMotif(it, T){
  it.T = {...T}; it.hidden = false;
  it.parts = genMotifParts(it); it.done = true; it.pi = it.parts.length; it.k = 0;
  redrawAll(); syncMotifUI();
}
function removeItem(it){
  const i = S.items.indexOf(it); if(i < 0) return;
  S.items.splice(i, 1);
  if(S.sel === it) S.sel = null;
  redrawAll(); afterChange(); syncMotifUI();
}
// 屏幕点 → 纹样本地坐标，判断是否落在纹样框里
function motifLocal(T, hx, hy){
  const dx = hx - T.cx, dy = hy - T.cy, c = Math.cos(T.rot), sn = Math.sin(T.rot);
  return {x:(dx*c + dy*sn)/T.size, y:(-dx*sn + dy*c)/T.size};
}
function motifAt(sx, sy){
  const h = Render.toHoop(sx, sy);
  for(let i = S.items.length - 1; i >= 0; i--){
    const it = S.items[i];
    if(it.kind !== 'motif') continue;
    const l = motifLocal(it.T, h.x, h.y);
    if(Math.abs(l.x) <= it.m.w/2 + 0.03 && Math.abs(l.y) <= it.m.h/2 + 0.03) return it;
  }
  return null;
}
// 选中框的把手（屏幕坐标）
const BOX_PAD = 0.05;
function motifHandles(it, T){
  const f = motifXf(T, true), hw = it.m.w/2 + BOX_PAD, hh = it.m.h/2 + BOX_PAD;
  const sc = (x, y) => { const q = f(x, y); return Render.toScreen(q.x, q.y); };
  const corners = [sc(-hw, -hh), sc(hw, -hh), sc(hw, hh), sc(-hw, hh)];
  const ctr = Render.toScreen(T.cx, T.cy), top = sc(0, -hh), bot = sc(0, hh);
  const out = (p, d) => { const vx = p.x - ctr.x, vy = p.y - ctr.y, l = Math.hypot(vx, vy) || 1; return {x:p.x + vx/l*d, y:p.y + vy/l*d}; };
  const rot = out(top, 30), base = out(bot, 26);
  const ux = Math.cos(T.rot), uy = Math.sin(T.rot);
  const btn = k => ({x:base.x + ux*k*30, y:base.y + uy*k*30});
  return {corners, top, rot, ctr, buttons:[{id:'fx', p:btn(-1), label:'⇋'}, {id:'fy', p:btn(0), label:'⇅'}, {id:'del', p:btn(1), label:'✕'}]};
}
function handleAt(sx, sy){
  if(!S.sel) return null;
  const H = motifHandles(S.sel, S.sel.T), near = (p, r) => Math.hypot(p.x - sx, p.y - sy) < r;
  for(const b of H.buttons) if(near(b.p, 12)) return {type:'button', id:b.id};
  if(near(H.rot, 12)) return {type:'rotate'};
  for(const c of H.corners) if(near(c, 11)) return {type:'scale'};
  return null;
}
function select(it){ S.sel = it; syncMotifUI(); }
function editT(){ return S.drag ? S.drag.T : S.sel ? S.sel.T : {size:S.motifSize, ...S.ghost}; }
function syncMotifUI(){
  const T = S.sel ? (S.drag ? S.drag.T : S.sel.T) : {size:S.motifSize, rot:S.ghost.rot, fx:S.ghost.fx, fy:S.ghost.fy};
  const deg = Math.round(((T.rot*180/Math.PI) % 360 + 540) % 360 - 180);
  document.getElementById('sizeR').value = T.size;
  document.getElementById('rotR').value = deg;
  document.getElementById('rotV').textContent = deg + '°';
  document.getElementById('fxBtn').classList.toggle('on', !!T.fx);
  document.getElementById('fyBtn').classList.toggle('on', !!T.fy);
  document.getElementById('delBtn').disabled = !S.sel;
  document.getElementById('selHint').textContent = S.sel ? `已选中：${S.sel.m.name}` : '点绷面落针；点已绣的纹样可选中调整';
}
// 改选中的纹样（或还没落针的影子）：先预览，停手后再重算针脚
let commitTimer = 0;
function previewEdit(patch, commitNow){
  if(!S.sel){
    if('size' in patch) S.motifSize = patch.size;
    for(const k of ['rot', 'fx', 'fy']) if(k in patch) S.ghost[k] = patch[k];
    syncMotifUI(); return;
  }
  if(!S.drag){ S.drag = {type:'edit', T:{...S.sel.T}}; S.sel.hidden = true; redrawAll(); }
  Object.assign(S.drag.T, patch);
  syncMotifUI();
  clearTimeout(commitTimer);
  const go = () => { if(S.drag && S.drag.type === 'edit'){ const T = S.drag.T; S.drag = null; updateMotif(S.sel, T); } };
  commitNow ? go() : (commitTimer = setTimeout(go, 280));
}
function undo(){
  const it = S.items.pop();
  if(it && it === S.sel){ S.sel = null; S.drag = null; }
  redrawAll(); afterChange(); syncMotifUI();
}
function clearAll(){ S.items = []; S.sel = null; S.drag = null; Render.clear(); S.frame = S.layout === 'center' ? (centerFrame(S.fontC) || flowFrame(S.fontC)) : flowFrame(S.fontC); afterChange(); }
function redrawAll(){
  Render.clear();
  for(const it of S.items){
    if(!it.parts || !it.ready || it.hidden) continue;
    const last = it.done ? it.parts.length : it.pi;
    for(let p = 0; p < last; p++) Render.drawStitches(it.parts[p], 0, it.parts[p].length);
    if(!it.done && it.k > 0) Render.drawStitches(it.parts[it.pi], 0, it.k);
  }
}

// ───────── 调度：按顺序一针针绣，字多了就加快 ─────────
const needle = {x:0, y:0, target:null, busy:false, flip:0};
function stitchStep(){
  let pending = 0;
  for(const it of S.items) if(!it.done) pending++;
  let budget = 14 + Math.max(0, pending - 1)*16;
  needle.busy = false;
  for(const it of S.items){
    if(budget <= 0) break;
    if(it.done) continue;
    if(!it.ready) break;
    needle.busy = true;
    while(budget > 0 && it.pi < it.parts.length){
      const part = it.parts[it.pi];
      if(it.k === 0 && part.length) pull();
      const n = Math.min(budget, part.length - it.k);
      if(n > 0){
        Render.drawStitches(part, it.k, it.k + n);
        const s = part[it.k + n - 1];
        needle.flip ^= 1;
        needle.target = Render.toScreen(needle.flip ? s.x1 : s.x2, needle.flip ? s.y1 : s.y2);
      }
      it.k += n; budget -= n;
      if(it.k >= part.length){ it.pi++; it.k = 0; }
    }
    if(it.pi >= it.parts.length) it.done = true; else break;
  }
}

// ───────── 声音：起一笔一声抽丝 ─────────
let AC = null, noiseBuf = null;
function initAudio(){
  if(AC) return;
  try{ AC = new (window.AudioContext || window.webkitAudioContext)();
    noiseBuf = AC.createBuffer(1, AC.sampleRate*0.3, AC.sampleRate);
    const d = noiseBuf.getChannelData(0); for(let i = 0; i < d.length; i++) d[i] = Math.random()*2 - 1;
  }catch(e){ AC = null; }
}
let lastPull = 0;
function pull(){
  if(!AC || performance.now() - lastPull < 90) return;
  lastPull = performance.now();
  const t = AC.currentTime, src = AC.createBufferSource(), bp = AC.createBiquadFilter(), g = AC.createGain();
  src.buffer = noiseBuf; bp.type = 'bandpass'; bp.Q.value = 1.4;
  bp.frequency.setValueAtTime(1600 + Math.random()*500, t); bp.frequency.exponentialRampToValueAtTime(5200, t + 0.16);
  g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(0.045, t + 0.03); g.gain.exponentialRampToValueAtTime(0.0008, t + 0.2);
  src.connect(bp).connect(g).connect(AC.destination); src.start(t); src.stop(t + 0.22);
}

function whoosh(){
  if(!AC) return;
  const t = AC.currentTime, src = AC.createBufferSource(), lp = AC.createBiquadFilter(), g = AC.createGain();
  src.buffer = noiseBuf; src.loop = true; lp.type = 'lowpass';
  lp.frequency.setValueAtTime(300, t); lp.frequency.linearRampToValueAtTime(1400, t + 0.5); lp.frequency.linearRampToValueAtTime(250, t + 2.2);
  g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(0.09, t + 0.4); g.gain.linearRampToValueAtTime(0, t + 2.3);
  src.connect(lp).connect(g).connect(AC.destination); src.start(t); src.stop(t + 2.4);
}

// ───────── 输入 ─────────
let composing = false;
function flush(){ if(composing) return; const v = ime.value; ime.value = ''; if(S.mode !== 'text') return; for(const ch of v) addChar(ch); }
ime.addEventListener('compositionstart', () => composing = true);
ime.addEventListener('compositionend', () => { composing = false; setTimeout(flush, 0); });
ime.addEventListener('input', e => { if(!e.isComposing) flush(); });
window.addEventListener('keydown', e => {
  initAudio();
  if(document.body.classList.contains('menu')){
    if(e.key === '1') chooseApp('craft'); else if(e.key === '2') chooseApp('game');
    return;
  }
  if(S.mode === 'dance'){ if(Dance.key(e)) e.preventDefault(); return; }
  if((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'z'){ e.preventDefault(); undo(); return; }
  if(e.isComposing || composing) return;
  if(S.mode === 'text'){
    if(document.activeElement !== ime) ime.focus();
    if(e.key === 'Enter'){ e.preventDefault(); addChar('\n'); }
    else if(e.key === 'Backspace' && ime.value === ''){ e.preventDefault(); undo(); }
  } else if(S.mode === 'motif' && !(e.target && e.target.tagName === 'INPUT')){
    const T = editT(), step = e.shiftKey ? 20 : 4, R15 = Math.PI/12;
    if((e.key === 'Backspace' || e.key === 'Delete')){ e.preventDefault(); S.sel ? removeItem(S.sel) : undo(); }
    else if(e.key === 'Escape'){ select(null); }
    else if(e.key === 'q' || e.key === 'Q'){ previewEdit({rot:T.rot - R15}); }
    else if(e.key === 'e' || e.key === 'E'){ previewEdit({rot:T.rot + R15}); }
    else if(e.key === 'f' || e.key === 'F'){ previewEdit({fx:!T.fx}); }
    else if(e.key === 'v' || e.key === 'V'){ previewEdit({fy:!T.fy}); }
    else if(S.sel && e.key.startsWith('Arrow')){
      e.preventDefault();
      const d = {ArrowLeft:[-step, 0], ArrowRight:[step, 0], ArrowUp:[0, -step], ArrowDown:[0, step]}[e.key];
      previewEdit({cx:T.cx + d[0], cy:T.cy + d[1]});
    }
  } else if(e.key === 'Backspace'){ e.preventDefault(); undo(); }
});

const mouse = {x:-1, y:-1, in:false, down:false, path:null};
const spoolRect = sp => ({x:S.img.x + (sp.x - sp.w/2)*ROOM.iw*S.img.s, y:S.img.y + (sp.y - sp.h/2)*ROOM.ih*S.img.s,
                          w:sp.w*ROOM.iw*S.img.s, h:sp.h*ROOM.ih*S.img.s});
const spoolAt = (x, y) => SPOOLS.findIndex(sp => { const r = spoolRect(sp); return x > r.x && x < r.x + r.w && y > r.y && y < r.y + r.h; });
function pickSpool(i){
  S.spool = i; S.palette = PALETTES[SPOOLS[i].pal];
  document.querySelectorAll('#palRow .sw').forEach((e, j) => e.classList.toggle('on', PALETTES[j] === S.palette));
}
const inHoop = (sx, sy) => { const h = Render.toHoop(sx, sy); return Math.hypot(h.x, h.y) < HOOP_R - 6; };
stage.addEventListener('pointermove', e => {
  mouse.x = e.clientX; mouse.y = e.clientY; mouse.in = inHoop(e.clientX, e.clientY);
  Scene.setMouse(e.clientX, e.clientY);
  S.hoverSpool = spoolAt(e.clientX, e.clientY);
  lightTarget = Math.atan2(-(e.clientY - S.hoop.cy), e.clientX - S.hoop.cx); lastMouse = performance.now();
  if(S.drag && S.drag.type !== 'edit'){ dragMove(e); return; }
  if(mouse.down && mouse.path){
    const h = Render.toHoop(e.clientX, e.clientY), l = mouse.path[mouse.path.length-1];
    if(Math.hypot(h.x - l.x, h.y - l.y) > 1.5) mouse.path.push(h);
  }
  let cur = S.hoverSpool >= 0 ? 'pointer' : S.mode === 'text' ? 'text' : (mouse.in ? 'crosshair' : 'default');
  if(S.mode === 'motif'){
    const h = handleAt(e.clientX, e.clientY);
    if(h) cur = h.type === 'button' ? 'pointer' : h.type === 'rotate' ? 'grab' : 'nwse-resize';
    else if(motifAt(e.clientX, e.clientY)) cur = 'move';
  }
  stage.style.cursor = cur;
});
stage.addEventListener('pointerdown', e => {
  initAudio();
  const sp = spoolAt(e.clientX, e.clientY);
  if(sp >= 0){ pickSpool(sp); refocus(); return; }
  if(!inHoop(e.clientX, e.clientY) && Scene.overWindow(e.clientX, e.clientY)){ Scene.gustBurst(); whoosh(); }
  if(S.mode === 'text'){ setTimeout(() => ime.focus(), 0); return; }
  if(S.mode === 'dance') return;
  if(!inHoop(e.clientX, e.clientY)) return;
  if(S.mode === 'draw'){ mouse.down = true; mouse.path = [Render.toHoop(e.clientX, e.clientY)]; stage.setPointerCapture(e.pointerId); }
  if(S.mode === 'motif') motifDown(e);
});
stage.addEventListener('pointerup', () => {
  if(S.drag && S.drag.type !== 'edit'){
    const d = S.drag; S.drag = null;
    if(d.moved) updateMotif(d.it, d.T); else { d.it.hidden = false; }
    syncMotifUI();
  }
  if(mouse.down && mouse.path && mouse.path.length > 2) addBrush(mouse.path);
  mouse.down = false; mouse.path = null;
});
stage.addEventListener('pointerleave', () => { mouse.in = false; });

function motifDown(e){
  const x = e.clientX, y = e.clientY, h = Render.toHoop(x, y);
  const hd = handleAt(x, y);
  if(hd && hd.type === 'button'){
    const T = S.sel.T;
    if(hd.id === 'del') removeItem(S.sel); else previewEdit({[hd.id]:!T[hd.id]}, true);
    return;
  }
  const startDrag = (type, it) => {
    S.drag = {type, it, T0:{...it.T}, T:{...it.T}, h0:h, moved:false,
              a0:Math.atan2(h.y - it.T.cy, h.x - it.T.cx), d0:Math.max(1, Math.hypot(h.x - it.T.cx, h.y - it.T.cy))};
    stage.setPointerCapture(e.pointerId);
  };
  if(hd){ startDrag(hd.type, S.sel); return; }
  const hit = motifAt(x, y);
  if(hit){ select(hit); startDrag('move', hit); return; }
  if(S.sel){ select(null); return; }                       // 点空白处：先取消选中
  if(S.motif && inHoop(x, y)) addMotif(S.motif, {cx:h.x, cy:h.y, size:S.motifSize, ...S.ghost});
}
function dragMove(e){
  const d = S.drag, h = Render.toHoop(e.clientX, e.clientY);
  if(!d.moved){
    if(Math.hypot(h.x - d.h0.x, h.y - d.h0.y)*Render.view.k < 3) return;
    d.moved = true; d.it.hidden = true; redrawAll();          // 拖动时先拆下来，松手再绣上
  }
  if(d.type === 'move'){ d.T.cx = d.T0.cx + h.x - d.h0.x; d.T.cy = d.T0.cy + h.y - d.h0.y; }
  if(d.type === 'rotate'){
    let r = d.T0.rot + Math.atan2(h.y - d.T0.cy, h.x - d.T0.cx) - d.a0;
    if(e.shiftKey) r = Math.round(r/(Math.PI/12))*(Math.PI/12);
    d.T.rot = r;
  }
  if(d.type === 'scale') d.T.size = Math.max(60, Math.min(900, d.T0.size*Math.hypot(h.x - d.T0.cx, h.y - d.T0.cy)/d.d0));
  syncMotifUI();
}
stage.addEventListener('wheel', e => {
  if(S.mode !== 'motif') return;
  e.preventDefault();
  const T = editT();
  if(e.shiftKey || e.altKey) previewEdit({rot:T.rot + (e.deltaY > 0 ? 1 : -1)*Math.PI/36});   // ⇧+滚轮：转 5°
  else previewEdit({size:Math.max(60, Math.min(900, T.size*(e.deltaY > 0 ? 0.93 : 1.07)))});
}, {passive:false});

// ───────── 光：跟鼠标转，静止时慢慢漂（默认从窗外右上方来）─────────
let lightA = 0.7, lightTarget = 0.7, lastMouse = 0;

// ───────── 针、线、光标、预览 ─────────
function drawFx(t){
  const {W, H} = S;
  fctx.setTransform(DPR, 0, 0, DPR, 0, 0);
  fctx.clearRect(0, 0, W, H);
  const pc = S.palette.a, pb = S.palette.b;
  const k = Render.view.k;
  // 字模式：下一个字的位置，一段平针虚线
  if(S.mode === 'text' && cursor && !needle.busy){
    const p = Render.toScreen(cursor.x, cursor.y), C = cursor.s*k, a = 0.35 + 0.3*Math.sin(t/300);
    fctx.save(); fctx.strokeStyle = `rgba(${pc.join(',')},${a})`; fctx.lineWidth = 1.3; fctx.setLineDash([5, 4]);
    fctx.beginPath(); fctx.moveTo(p.x + C*0.2, p.y + C*0.03); fctx.lineTo(p.x + C*0.8, p.y + C*0.03); fctx.stroke(); fctx.restore();
  }
  // 画模式：正在画的笔迹
  if(S.mode === 'draw' && mouse.path && mouse.path.length > 1){
    fctx.save(); fctx.strokeStyle = `rgba(${pc.join(',')},0.45)`; fctx.lineWidth = S.brushW*k; fctx.lineCap = fctx.lineJoin = 'round';
    fctx.beginPath(); mouse.path.forEach((h, i) => { const p = Render.toScreen(h.x, h.y); i ? fctx.lineTo(p.x, p.y) : fctx.moveTo(p.x, p.y); }); fctx.stroke(); fctx.restore();
  }
  if(S.mode === 'draw' && mouse.in && !mouse.down){
    fctx.save(); fctx.strokeStyle = `rgba(${pc.join(',')},0.6)`; fctx.lineWidth = 1;
    fctx.beginPath(); fctx.arc(mouse.x, mouse.y, S.brushW*k/2, 0, Math.PI*2); fctx.stroke(); fctx.restore();
  }
  // 纹模式：选中框 / 正在调整的影子 / 还没落针的影子
  if(S.mode === 'motif'){
    const ghost = (m, T, alpha) => {
      const polys = motifPolys(m, T);
      fctx.save(); fctx.globalAlpha = alpha;
      polys.forEach((P, i) => {
        fctx.fillStyle = `rgb(${m.regions[i].col.join(',')})`;
        fctx.beginPath();
        P.forEach(poly => poly.forEach((q, j) => { const p = Render.toScreen(q.x, q.y); j ? fctx.lineTo(p.x, p.y) : fctx.moveTo(p.x, p.y); }));
        fctx.fill('evenodd');
      });
      fctx.restore();
    };
    const editing = S.drag && (S.drag.type === 'edit' || S.drag.moved);
    if(S.sel){
      const T = editing ? S.drag.T : S.sel.T;
      if(editing) ghost(S.sel.m, T, 0.6);
      const H = motifHandles(S.sel, T);
      fctx.save();
      fctx.lineWidth = 1; fctx.setLineDash([5, 4]);
      for(const [col, w] of [['rgba(255,250,240,.9)', 3], ['rgba(60,40,28,.9)', 1]]){
        fctx.strokeStyle = col; fctx.lineWidth = w;
        fctx.beginPath(); H.corners.forEach((c, i) => i ? fctx.lineTo(c.x, c.y) : fctx.moveTo(c.x, c.y)); fctx.closePath(); fctx.stroke();
        fctx.beginPath(); fctx.moveTo(H.top.x, H.top.y); fctx.lineTo(H.rot.x, H.rot.y); fctx.stroke();
      }
      fctx.setLineDash([]);
      const dot = (p, r, fill) => { fctx.beginPath(); fctx.arc(p.x, p.y, r, 0, Math.PI*2); fctx.fillStyle = fill; fctx.fill();
        fctx.lineWidth = 1.2; fctx.strokeStyle = 'rgba(60,40,28,.9)'; fctx.stroke(); };
      H.corners.forEach(c => dot(c, 5, '#fbf6ea'));
      dot(H.rot, 7, '#8e2a2e');
      fctx.strokeStyle = '#fbf6ea'; fctx.lineWidth = 1.4;
      fctx.beginPath(); fctx.arc(H.rot.x, H.rot.y, 3.4, -0.3, Math.PI*1.4); fctx.stroke();
      fctx.font = '13px "Songti SC","SimSun",serif'; fctx.textAlign = 'center'; fctx.textBaseline = 'middle';
      H.buttons.forEach(b => {
        dot(b.p, 11, b.id === 'del' ? '#fbeaea' : (T[b.id] ? '#3b2a1e' : '#fbf6ea'));
        fctx.fillStyle = b.id === 'del' ? '#8e2a2e' : (T[b.id] ? '#fbf6ea' : '#3b2a1e');
        fctx.fillText(b.label, b.p.x, b.p.y + 0.5);
      });
      if(editing && S.drag.type === 'rotate'){
        const deg = Math.round(((T.rot*180/Math.PI) % 360 + 540) % 360 - 180);
        fctx.fillStyle = 'rgba(40,26,16,.75)'; fctx.fillRect(H.rot.x + 12, H.rot.y - 10, 44, 20);
        fctx.fillStyle = '#fbf6ea'; fctx.fillText(deg + '°', H.rot.x + 34, H.rot.y);
      }
      fctx.restore();
    } else if(S.motif && mouse.in && !motifAt(mouse.x, mouse.y)){
      const h = Render.toHoop(mouse.x, mouse.y);
      ghost(S.motif, {cx:h.x, cy:h.y, size:S.motifSize, ...S.ghost}, 0.33);
    }
  }
  // 针
  let tgt = needle.target;
  if(!needle.busy){
    if(S.mode === 'text' && cursor){ const p = Render.toScreen(cursor.x + cursor.s*0.5, cursor.y + cursor.s*0.16); tgt = {x:p.x, y:p.y + Math.sin(t/500)*2}; }
    else if(mouse.in && S.mode === 'draw'){ tgt = {x:mouse.x + 18, y:mouse.y - 22}; }
    else tgt = {x:S.hoop.cx + S.hoop.r*0.72, y:S.hoop.cy - S.hoop.r*0.55 + Math.sin(t/600)*3};
  }
  const kk = needle.busy ? 0.55 : 0.12;
  needle.x += (tgt.x - needle.x)*kk; needle.y += (tgt.y - needle.y)*kk;
  // 线从针眼垂下，一直牵到桌上的线轴
  const spr = spoolRect(SPOOLS[S.spool]);
  const sp = {x:spr.x + spr.w*0.5, y:spr.y + spr.h*0.12};
  // 鼠标停在线轴上：描一圈细光
  if(S.hoverSpool >= 0){
    const r = spoolRect(SPOOLS[S.hoverSpool]);
    fctx.save(); fctx.strokeStyle = 'rgba(255,240,210,.85)'; fctx.lineWidth = 1.2; fctx.shadowColor = 'rgba(255,230,180,.9)'; fctx.shadowBlur = 8;
    fctx.beginPath(); fctx.roundRect(r.x - 2, r.y - 2, r.w + 4, r.h + 4, 4); fctx.stroke(); fctx.restore();
  }
  // 游戏里改成「从背面绣」：针线画在人物后面（舞台层），正面只露针和一小段线
  if(S.mode === 'dance'){ window.drawNeedleLayer = c => drawNeedleBack(c, t, sp, pc, pb); }
  else { window.drawNeedleLayer = null; drawNeedleFront(fctx, t, sp, pc, pb); }
  Dance.drawFx(fctx);
}

function drawNeedleBody(c){
  const ang = -0.95, ux = Math.cos(ang), uy = Math.sin(ang);   // 针身朝右上
  const g = c.createLinearGradient(needle.x - uy*2, needle.y + ux*2, needle.x + uy*2, needle.y - ux*2);
  g.addColorStop(0, '#6b6f75'); g.addColorStop(0.5, '#f4f6f8'); g.addColorStop(1, '#7c8086');
  c.lineCap = 'round';
  c.strokeStyle = 'rgba(0,0,0,.22)'; c.lineWidth = 2.3;
  c.beginPath(); c.moveTo(needle.x + 2, needle.y + 3); c.lineTo(needle.x + ux*40 + 2, needle.y + uy*40 + 3); c.stroke();
  c.strokeStyle = g; c.lineWidth = 1.9;
  c.beginPath(); c.moveTo(needle.x, needle.y); c.lineTo(needle.x + ux*40, needle.y + uy*40); c.stroke();
  const eye = {x:needle.x + ux*34, y:needle.y + uy*34};
  c.strokeStyle = '#555a60'; c.lineWidth = 0.7;
  c.beginPath(); c.ellipse(eye.x, eye.y, 2.6, 0.9, ang, 0, Math.PI*2); c.stroke();
  return eye;
}
// 正面：线从针眼垂下，一直牵到桌上的线轴
function drawNeedleFront(c, t, sp, pc, pb){
  const ang = -0.95, eye = {x:needle.x + Math.cos(ang)*34, y:needle.y + Math.sin(ang)*34};
  const dist = Math.hypot(sp.x - eye.x, sp.y - eye.y);
  const c1 = {x:eye.x + 40, y:eye.y - 30}, c2 = {x:(eye.x + sp.x)/2, y:Math.max(eye.y, sp.y) + Math.min(260, dist*0.22) + Math.sin(t/700)*5};
  c.lineCap = 'round';
  c.strokeStyle = 'rgba(0,0,0,.16)'; c.lineWidth = 2.2;
  c.beginPath(); c.moveTo(eye.x + 2, eye.y + 3); c.bezierCurveTo(c1.x + 2, c1.y + 3, c2.x + 2, c2.y + 8, sp.x, sp.y + 4); c.stroke();
  c.strokeStyle = `rgb(${pc.join(',')})`; c.lineWidth = 1.7;
  c.beginPath(); c.moveTo(eye.x, eye.y); c.bezierCurveTo(c1.x, c1.y, c2.x, c2.y, sp.x, sp.y); c.stroke();
  c.strokeStyle = `rgba(${pb.join(',')},.75)`; c.lineWidth = 0.6;
  c.beginPath(); c.moveTo(eye.x, eye.y - 0.5); c.bezierCurveTo(c1.x, c1.y - 0.5, c2.x, c2.y - 0.5, sp.x, sp.y - 0.5); c.stroke();
  drawNeedleBody(c);
}
// 背面：长线走在绢后面，透过绢只看得到一道淡影；出了绷圈才是实线
function drawNeedleBack(c, t, sp, pc, pb){
  const h = S.hoop, ang = -0.95, eye = {x:needle.x + Math.cos(ang)*34, y:needle.y + Math.sin(ang)*34};
  const hole = {x:needle.x - 7, y:needle.y + 9};                       // 线从这里穿进绢
  const dist = Math.hypot(sp.x - hole.x, sp.y - hole.y);
  const c2 = {x:(hole.x + sp.x)/2, y:Math.max(hole.y, sp.y) + Math.min(220, dist*0.18) + Math.sin(t/700)*5};
  const longLine = () => { c.beginPath(); c.moveTo(hole.x, hole.y); c.quadraticCurveTo(c2.x, c2.y, sp.x, sp.y); c.stroke(); };
  c.save(); c.lineCap = 'round';
  // 绷圈里：隔着绢的影子
  c.save(); c.beginPath(); c.arc(h.cx, h.cy, h.r*0.995, 0, Math.PI*2); c.clip();
  c.globalAlpha = 0.2; c.filter = 'blur(1.4px)'; c.strokeStyle = `rgb(${pc.join(',')})`; c.lineWidth = 2.4; longLine();
  c.restore();
  // 绷圈外：实线
  c.save(); c.beginPath(); c.rect(0, 0, S.W, S.H); c.arc(h.cx, h.cy, h.r*1.06, 0, Math.PI*2, true); c.clip();
  c.strokeStyle = `rgb(${pc.join(',')})`; c.lineWidth = 1.7; longLine();
  c.restore();
  // 正面：针眼到穿绢处一小段
  c.strokeStyle = `rgb(${pc.join(',')})`; c.lineWidth = 1.7;
  c.beginPath(); c.moveTo(eye.x, eye.y); c.bezierCurveTo(eye.x + 16, eye.y - 14, hole.x + 10, hole.y + 2, hole.x, hole.y); c.stroke();
  c.strokeStyle = `rgba(${pb.join(',')},.75)`; c.lineWidth = 0.6;
  c.beginPath(); c.moveTo(eye.x, eye.y - 0.5); c.bezierCurveTo(eye.x + 16, eye.y - 14.5, hole.x + 10, hole.y + 1.5, hole.x, hole.y - 0.5); c.stroke();
  c.fillStyle = 'rgba(40,24,14,.55)'; c.beginPath(); c.ellipse(hole.x, hole.y, 1.8, 1.2, 0, 0, Math.PI*2); c.fill();
  drawNeedleBody(c);
  c.restore();
}

// ───────── 主循环 ─────────
function frame(t){
  stitchStep();
  if(performance.now() - lastMouse > 2500) lightTarget += 0.002;
  let d = lightTarget - lightA; d = Math.atan2(Math.sin(d), Math.cos(d)); lightA += d*0.06;
  Scene.update(t);
  Dance.update();
  Render.frame(lightA, S.fabric.c, t/1000, Scene.wind || 0.25);
  drawFx(t);
  requestAnimationFrame(frame);
}

// ───────── 版面：房间图铺满，圆绷尽量大而完整 ─────────
function resize(){
  const W = innerWidth, H = innerHeight;
  S.W = W; S.H = H;
  const cover = Math.max(W/ROOM.iw, H/ROOM.ih);
  const fit = Math.min(W*0.96, H*0.96)/(2*ROOM.r*ROOM.ih);       // 圆绷必须整个看得见
  const s = Math.min(cover, fit);
  const iw = ROOM.iw*s, ih = ROOM.ih*s;
  let x = W/2 - ROOM.cx*iw, y = H/2 - ROOM.cy*ih;                 // 圆心放到屏幕中央
  x = Math.min(0, Math.max(W - iw, x)); y = Math.min(0, Math.max(H - ih, y));
  if(iw < W) x = (W - iw)/2; if(ih < H) y = (H - ih)/2;
  S.img = {x, y, s};
  Object.assign(bgImg.style, {left:x + 'px', top:y + 'px', width:iw + 'px', height:ih + 'px'});
  S.hoop = {cx:x + ROOM.cx*iw, cy:y + ROOM.cy*ih, r:ROOM.r*ih};
  fx.width = Math.round(W*DPR); fx.height = Math.round(H*DPR);
  Render.resize(W, H, S.hoop);
  Scene.resize(W, H, S.img);
  redrawAll(); afterChange();
}
let rt; addEventListener('resize', () => { clearTimeout(rt); rt = setTimeout(resize, 120); });

// ───────── 面板 ─────────
function swatches(rowId, list, get, set, colorOf){
  const row = document.getElementById(rowId);
  list.forEach(o => {
    const s = document.createElement('button'); s.className = 'sw'; s.title = o.name; s.style.background = colorOf(o);
    s.onclick = () => { set(o); row.querySelectorAll('.sw').forEach(e => e.classList.toggle('on', e === s)); refocus(); };
    if(get() === o) s.classList.add('on');
    row.appendChild(s);
  });
}
const refocus = () => { if(S.mode === 'text') ime.focus(); };
swatches('fabRow', FABRICS, () => S.fabric, o => S.fabric = o, o => `rgb(${o.c.join(',')})`);
swatches('palRow', PALETTES, () => S.palette, o => { S.palette = o; const j = SPOOLS.findIndex(sp => PALETTES[sp.pal] === o); if(j >= 0) S.spool = j; }, o => `linear-gradient(135deg,rgb(${o.a.join(',')}),rgb(${o.b.join(',')}))`);

function setMode(m){
  const prev = S.mode;
  S.mode = m;
  if(prev === 'dance' && m !== 'dance') Dance.leave();
  if(m === 'dance') Dance.enter();
  afterChange();
  if(m !== 'motif' && S.sel) select(null);
  document.querySelectorAll('.tab').forEach(b => b.classList.toggle('on', b.dataset.mode === m));
  document.querySelectorAll('.pane').forEach(p => p.hidden = p.dataset.mode !== m);
  document.getElementById('tip').textContent = {
    text:'敲键盘，一字一字绣上去　回车换列　退格拆线',
    draw:'在绷面上拖动，画出的笔迹会绣成缎面　⌘Z 拆线',
    dance:'空格开演　F 接左边红绣球　J 接右边青花绣球　Esc 停',
    motif:'点绷面落针　点纹样选中后可拖动、转把手、拉四角　滚轮缩放　⇧滚轮/Q E 旋转　F V 翻转',
  }[m];
  refocus();
}
document.querySelectorAll('.tab').forEach(b => b.onclick = () => setMode(b.dataset.mode));
document.querySelectorAll('.bw').forEach(b => b.onclick = () => {
  S.brushW = +b.dataset.w; document.querySelectorAll('.bw').forEach(e => e.classList.toggle('on', e === b));
});
document.querySelectorAll('.fs').forEach(b => b.onclick = () => { setFontSize(+b.dataset.c); refocus(); });
document.getElementById('autoBtn').onclick = () => { autoLayout(); refocus(); };
document.getElementById('flowBtn').onclick = () => { flowLayout(); refocus(); };
document.getElementById('sizeR').oninput = e => previewEdit({size:+e.target.value});
document.getElementById('rotR').oninput = e => previewEdit({rot:+e.target.value*Math.PI/180});
document.getElementById('sizeR').onchange = document.getElementById('rotR').onchange = () => { if(S.drag && S.drag.type === 'edit') previewEdit({}, true); };
document.getElementById('fxBtn').onclick = () => previewEdit({fx:!editT().fx}, true);
document.getElementById('fyBtn').onclick = () => previewEdit({fy:!editT().fy}, true);
document.getElementById('rot0Btn').onclick = () => previewEdit({rot:0}, true);
document.getElementById('delBtn').onclick = () => { if(S.sel) removeItem(S.sel); };
document.getElementById('undoBtn').onclick = () => { undo(); refocus(); };
document.getElementById('clearBtn').onclick = () => { clearAll(); refocus(); };

function motifSVG(m){
  const paths = m.regions.map(r => {
    const d = r.polys.map(p => { let s = ''; for(let i = 0; i < p.length; i += 2) s += (i ? 'L' : 'M') + p[i] + ' ' + p[i+1]; return s + 'Z'; }).join('');
    return `<path d="${d}" fill="rgb(${r.col.join(',')})" fill-rule="evenodd"/>`;
  }).join('');
  return `<svg viewBox="-0.56 -0.56 1.12 1.12">${paths}</svg>`;
}
function buildMotifPane(){
  const tabs = document.getElementById('catTabs'), grid = document.getElementById('motifGrid');
  tabs.innerHTML = '';
  CATS.forEach(c => {
    const b = document.createElement('button'); b.className = 'cat' + (c === S.cat ? ' on' : ''); b.textContent = c;
    b.onclick = () => { S.cat = c; buildMotifPane(); };
    tabs.appendChild(b);
  });
  grid.innerHTML = '';
  MOTIFS.filter(m => m.cat === S.cat).forEach(m => {
    const b = document.createElement('button'); b.className = 'mo' + (S.motif === m ? ' on' : ''); b.title = m.name;
    b.innerHTML = motifSVG(m) + `<span>${m.name}</span>`;
    b.onclick = () => { S.motif = m; select(null); buildMotifPane(); };
    grid.appendChild(b);
  });
}
MOTIFS = window.MOTIFS_DATA || []; S.motif = MOTIFS[0] || null; buildMotifPane(); syncMotifUI();

// ───────── 启动 ─────────
S.frame = flowFrame(S.fontC);
Dance.bindUI();
bgImg.onload = () => document.body.classList.add('ready');
bgImg.src = R(ROOM.src);
document.getElementById('icoGame').src = R('assets/game/pose_3.webp');
Scene.init().then(ok => { if(ok){ Scene.resize(S.W, S.H, S.img); document.body.classList.add('live'); } });
resize();
needle.x = S.hoop.cx + S.hoop.r*0.7; needle.y = S.hoop.cy - S.hoop.r*0.5;
// ───────── 开始菜单：刺绣 / 绣舞 ─────────
function openMenu(){
  if(S.mode === 'dance') setMode('text');
  document.body.classList.add('menu');
  ime.blur();
}
function chooseApp(kind){
  S.app = kind;
  document.querySelectorAll('.tab').forEach(b => b.hidden = kind === 'game' ? b.dataset.mode !== 'dance' : b.dataset.mode === 'dance');
  document.querySelector('.tabs').style.display = kind === 'game' ? 'none' : '';
  document.getElementById('appTitle').textContent = kind === 'game' ? '绣舞' : '苏绣';
  document.body.classList.remove('menu');
  setMode(kind === 'game' ? 'dance' : 'text');
}
document.querySelectorAll('.opt').forEach(b => b.onclick = () => { initAudio(); chooseApp(b.dataset.go); });
document.getElementById('menuBtn').onclick = openMenu;
{ const m = MOTIFS.find(x => x.name === '牡丹'); if(m) document.getElementById('icoCraft').innerHTML = motifSVG(m); }
setMode('text');
document.body.classList.add('menu');
requestAnimationFrame(frame);
// 调试入口
window.__type = s => { for(const ch of s) addChar(ch); };
window.__S = S;
