'use strict';
// ════════════════════════════════════════════════════════════════
//  舞：戏曲人物随节拍起舞的轻音游
//  绣球沿圆绷边从顶上分左右两路滚下，到底部针眼处判定：左路红绣球按 F，右路青花绣球按 J。
//  人物是一段连续舞蹈的逐帧动画：踩中就往下跳，停手就慢慢停住，踩空打个趔趄。
//  绷边每三四拍绣一个大纹样（可选），踩中一拍绣一份，踩空那份就空着。
// ════════════════════════════════════════════════════════════════
const Dance = (() => {
  const WIN_PERFECT = 0.07, WIN_GOOD = 0.14, WIN_MISS = 0.16;
  const RT = 470, RW = 392;                      // 绣球轨道半径、绷边纹样环半径（绷面单位）
  const BORDERS = ['随连击', '梅', '牡丹', '荷', '蝴蝶', '祥云', '金鱼', '燕子', '兰', '竹', '红日祥云', '仙鹤', '混搭'];
  // 随连击：连得越多，绣的纹样越华丽；一失手就退回第一档
  const TIERS = [{c:0, n:'兰'}, {c:5, n:'梅'}, {c:10, n:'荷'}, {c:20, n:'牡丹'}, {c:30, n:'蝴蝶'}, {c:45, n:'仙鹤'}];
  const tierOf = combo => { let i = 0; TIERS.forEach((t, k) => { if(combo >= t.c) i = k; }); return i; };
  const MIX = ['梅', '蝴蝶', '牡丹', '祥云', '荷', '燕子'];
  const cvs = document.getElementById('dance'), ctx = cvs.getContext('2d');

  // ── 人物：逐帧序列（data/dancers.js），没有就用花旦站姿顶上 ──
  const DANCERS = window.DANCERS || [];
  const frames = {};                             // id → [Image]
  function loadDancer(d){
    if(frames[d.id]) return;
    frames[d.id] = [...Array(d.n)].map((_, i) => { const im = new Image(); im.src = R(`${d.dir}/${String(i).padStart(3, '0')}.webp`); return im; });
  }
  const fallback = new Image(); fallback.src = R('assets/game/pose_1.webp');

  // 细斜纹罩层：让人物像丝线织出来的
  const grain = document.createElement('canvas'); grain.width = grain.height = 64;
  { const g = grain.getContext('2d');
    for(let k = -64; k < 128; k += 3){ g.strokeStyle = k % 6 ? 'rgba(255,255,255,0.07)' : 'rgba(60,20,30,0.06)'; g.lineWidth = 1.3;
      g.beginPath(); g.moveTo(k, 0); g.lineTo(k + 36, 64); g.stroke(); } }
  const work = document.createElement('canvas'), wctx = work.getContext('2d');

  // ── 绣球贴图（程序画，一次画好）──
  function makeBall(base, dark, light, pat, gold){
    const N = 160, c = document.createElement('canvas'); c.width = c.height = N;
    const x = c.getContext('2d'), r = N*0.46, cx = N/2, cy = N/2;
    const g = x.createRadialGradient(cx - r*0.35, cy - r*0.4, r*0.1, cx, cy, r);
    g.addColorStop(0, light); g.addColorStop(0.55, base); g.addColorStop(1, dark);
    x.fillStyle = g; x.beginPath(); x.arc(cx, cy, r, 0, Math.PI*2); x.fill();
    x.save(); x.beginPath(); x.arc(cx, cy, r, 0, Math.PI*2); x.clip();
    // 球面上绣的小花：按球面投影排布，越靠边越扁
    for(let la = -70; la <= 70; la += 28) for(let lo = -80; lo <= 80; lo += 32){
      const L = (la + (lo % 64 ? 14 : 0))*Math.PI/180, O = lo*Math.PI/180;
      const px = cx + r*Math.cos(L)*Math.sin(O), py = cy - r*Math.sin(L), sq = Math.cos(L)*Math.cos(O);
      if(sq < 0.15) continue;
      const fr = r*0.13;
      for(let k = 0; k < 5; k++){
        const a = k*Math.PI*2/5;
        x.fillStyle = pat; x.beginPath();
        x.ellipse(px + Math.cos(a)*fr*0.55*Math.max(0.3, Math.cos(O)), py + Math.sin(a)*fr*0.55*Math.max(0.3, Math.cos(L)), fr*0.42*Math.max(0.35, Math.cos(O)), fr*0.32*Math.max(0.35, Math.cos(L)), a, 0, Math.PI*2);
        x.fill();
      }
      x.fillStyle = gold; x.beginPath(); x.arc(px, py, fr*0.22*sq, 0, Math.PI*2); x.fill();
    }
    // 金线经纬
    x.strokeStyle = gold; x.lineWidth = 2.2; x.globalAlpha = 0.9;
    for(const k of [-0.55, 0, 0.55]){ x.beginPath(); x.ellipse(cx, cy, r*Math.abs(k || 0.02)*1.0 + (k ? 0 : 1), r, 0, 0, Math.PI*2); x.stroke(); }
    x.beginPath(); x.ellipse(cx, cy, r, r*0.28, 0, 0, Math.PI*2); x.stroke();
    x.globalAlpha = 1;
    // 高光
    const hl = x.createRadialGradient(cx - r*0.38, cy - r*0.42, 0, cx - r*0.38, cy - r*0.42, r*0.5);
    hl.addColorStop(0, 'rgba(255,255,255,.75)'); hl.addColorStop(1, 'rgba(255,255,255,0)');
    x.fillStyle = hl; x.fillRect(0, 0, N, N);
    x.restore();
    x.strokeStyle = 'rgba(40,20,10,.55)'; x.lineWidth = 1.5; x.beginPath(); x.arc(cx, cy, r, 0, Math.PI*2); x.stroke();
    return c;
  }
  const BALLS = [
    {img:makeBall('#c0182c', '#5a0612', '#ff8a98', 'rgba(255,214,140,.9)', '#e8b54a'), tassel:'#b5122a', track:'170,30,52', name:'红绣球'},
    {img:makeBall('#eef0f2', '#8f9aa8', '#ffffff', 'rgba(36,74,150,.88)', '#c9a24a'), tassel:'#2c4f96', track:'44,84,160', name:'青花绣球'},
  ];

  const G = {state:'idle', song:0, diff:'轻松', dancer:0, border:'随连击', tier:0, toast:null, offsetMs:0, audio:null, clock0:0,
             notes:[], idx:0, score:0, combo:0, maxCombo:0, counts:{妙:0, 好:0, 失:0},
             pos:0, dir:1, speed:0, lastHit:-99, stumble:0, pops:[], fxs:[], stash:null, flash:[0, 0], endT:0, slots:[]};

  function songTime(){
    if(!G.audio) return -1;
    const now = performance.now()/1000;
    if(!G.audio.paused){ const real = G.audio.currentTime, est = now - G.clock0; if(Math.abs(est - real) > 0.05) G.clock0 = now - real; }
    return now - G.clock0 - G.offsetMs/1000;
  }
  const chart = () => window.CHARTS[G.song];
  const dancer = () => DANCERS[G.dancer];
  const beatLen = () => 60/chart().bpm;
  // 当前在一拍里的位置（0~1），按谱面里逐拍跟踪出的拍点算
  function beatPhase(t){
    const B = chart().beats;
    if(!B || t < B[0] || t >= B[B.length - 1]) return (((t - chart().beat0)/beatLen()) % 1 + 1) % 1;
    let lo = 0, hi = B.length - 1;
    while(hi - lo > 1){ const m = (lo + hi) >> 1; if(B[m] <= t) lo = m; else hi = m; }
    return (t - B[lo])/(B[hi] - B[lo]);
  }

  // ── 声音：梆子（中）、碰铃（妙）、闷鼓（失）──
  function tone(f, t0, dur, vol, type = 'sine', bend = 1){
    const o = AC.createOscillator(), g = AC.createGain();
    o.type = type; o.frequency.setValueAtTime(f, t0); o.frequency.exponentialRampToValueAtTime(f*bend, t0 + dur);
    g.gain.setValueAtTime(0, t0); g.gain.linearRampToValueAtTime(vol, t0 + 0.004); g.gain.exponentialRampToValueAtTime(0.0005, t0 + dur);
    o.connect(g).connect(AC.destination); o.start(t0); o.stop(t0 + dur + 0.02);
  }
  function sfx(kind){
    if(!AC) return;
    const t = AC.currentTime;
    if(kind === '妙' || kind === '好'){
      const f = kind === '妙' ? 1180 : 940;                       // 梆子：两个泛音、很短
      tone(f, t, 0.09, 0.32); tone(f*2.76, t, 0.05, 0.12);
      if(kind === '妙'){ tone(2640, t + 0.01, 0.7, 0.07); tone(3960, t + 0.01, 0.45, 0.04); }   // 碰铃
    } else {
      tone(150, t, 0.28, 0.45, 'sine', 0.45);                      // 闷鼓往下沉
      tone(95, t, 0.32, 0.3, 'triangle', 0.6);
    }
  }

  // ── 开演 / 停 / 结束 ──
  function start(){
    if(!window.CHARTS) return;
    stop();
    if(!G.stash) G.stash = S.items;
    S.items = []; S.sel = null; S.drag = null; redrawAll(); afterChange();
    const C = chart();
    G.notes = C.charts[G.diff].map(([t, lane], i) => ({t, lane, i, hit:null}));
    G.idx = 0; G.score = 0; G.combo = 0; G.maxCombo = 0; G.counts = {妙:0, 好:0, 失:0};
    G.pops = []; G.fxs = []; G.lastHit = -99; G.speed = 0; G.stumble = 0; G.tier = 0; G.toast = null;
    G.endT = C.end;
    planSlots();
    G.audio = new Audio(R(C.src)); G.audio.volume = 0.85;
    G.audio.addEventListener('playing', () => { G.clock0 = performance.now()/1000 - G.audio.currentTime; });
    G.audio.play().catch(() => {});
    G.clock0 = performance.now()/1000;
    G.state = 'play';
    hideResult(); syncUI();
  }
  function stop(){ if(G.audio){ G.audio.pause(); G.audio = null; } }
  function finish(){
    const a = G.audio;
    if(a){ const v0 = a.volume, t0 = performance.now();
      const fade = () => { const k = (performance.now() - t0)/1500; if(k >= 1){ a.pause(); return; } a.volume = v0*(1 - k); requestAnimationFrame(fade); }; fade(); }
    G.audio = null; G.state = 'done';
    showResult(); syncUI();
  }

  // ── 绷边纹样：每 K 拍一个，针脚按拍分份 ──
  function planSlots(){
    const N = G.notes.length, K = G.diff === '轻松' ? 3 : 4, n = Math.ceil(N/K);
    const spacing = 2*Math.PI*RW/n, size = Math.min(150, spacing*1.0);
    G.slots = [...Array(n)].map((_, j) => {
      const name = G.border === '混搭' ? MIX[j % MIX.length] : G.border === '随连击' ? null : G.border;
      const m = name ? (MOTIFS.find(x => x.name === name) || MOTIFS[0]) : null;
      const ph = -Math.PI/2 + 2*Math.PI*(j + 0.5)/n;
      return {m, K, ph, base:size, size:m ? size/Math.max(m.w, m.h)*0.92 : 0, chunks:null, cx:Math.cos(ph)*RW, cy:Math.sin(ph)*RW};
    });
  }
  function slotChunks(sl, j){
    if(sl.chunks) return sl.chunks;
    if(!sl.m){                                     // 随连击：这一格开绣时按当前档位定纹样
      sl.m = MOTIFS.find(x => x.name === TIERS[G.tier].n) || MOTIFS[0];
      sl.size = sl.base/Math.max(sl.m.w, sl.m.h)*0.92;
    }
    // 纹样的「上」朝外，左半圈的鸟兽翻个面，朝着前进方向
    const T = {cx:sl.cx, cy:sl.cy, size:sl.size, rot:sl.ph + Math.PI/2, fx:Math.cos(sl.ph) < -0.05, fy:false};
    const parts = genMotifParts({m:sl.m, T, seed:j*977 + 13});
    const all = parts.flat(), per = Math.ceil(all.length/sl.K);
    sl.chunks = [...Array(sl.K)].map((_, k) => all.slice(k*per, (k + 1)*per));
    return sl.chunks;
  }
  function embroider(n){
    const K = G.slots[0].K, j = Math.floor(n.i/K), sl = G.slots[j];
    if(!sl) return;
    const ch = slotChunks(sl, j)[n.i % K];
    if(ch && ch.length) S.items.push({kind:'flower', parts:[ch], ready:true, pi:0, k:0, done:false});
  }

  // ── 判定 ──
  function press(lane){
    if(G.state !== 'play') return;
    const t = songTime();
    G.flash[lane] = performance.now();
    for(let i = G.idx; i < G.notes.length; i++){
      const n = G.notes[i];
      if(n.t - t > WIN_GOOD) break;
      if(n.hit || n.lane !== lane) continue;
      if(Math.abs(n.t - t) <= WIN_GOOD){ judge(n, Math.abs(n.t - t) <= WIN_PERFECT ? '妙' : '好'); return; }
    }
  }
  function judge(n, kind){
    n.hit = kind; G.counts[kind]++;
    const now = performance.now();
    sfx(kind);
    if(kind === '失'){
      G.combo = 0; G.tier = 0; G.stumble = now; G.speed *= 0.2;
      G.fxs.push({type:'drop', lane:n.lane, t:now});
    } else {
      G.combo++; G.maxCombo = Math.max(G.maxCombo, G.combo);
      const nt = tierOf(G.combo);
      if(nt > G.tier && G.border === '随连击') G.toast = {name:TIERS[nt].n, t:now};
      G.tier = nt;
      G.score += Math.round((kind === '妙' ? 300 : 120)*(1 + Math.min(0.5, G.combo*0.01)));
      G.lastHit = songTime();
      G.boost = kind === '妙' ? 1.15 : 1;
      embroider(n);
      G.fxs.push({type:kind === '妙' ? 'burst' : 'ring', lane:n.lane, t:now});
      if(G.combo % 25 === 0) Scene.gustBurst();
    }
    G.pops.push({kind, t:now, lane:n.lane});
  }

  // ── 每帧：推进判定、人物播放头 ──
  let lastT = performance.now();
  function update(){
    const on = S.mode === 'dance';
    cvs.style.display = on ? 'block' : 'none';
    const now = performance.now(), dt = Math.min(0.05, (now - lastT)/1000); lastT = now;
    if(!on) return;
    if(cvs.width !== Math.round(S.W*DPR)){ cvs.width = Math.round(S.W*DPR); cvs.height = Math.round(S.H*DPR); }
    if(G.state === 'play'){
      const t = songTime();
      while(G.idx < G.notes.length && (G.notes[G.idx].hit || G.notes[G.idx].t < t - WIN_MISS)){
        const n = G.notes[G.idx]; if(!n.hit) judge(n, '失'); G.idx++;
      }
      if(t > G.endT + 0.4) finish();
    }
    // 播放头：最近一拍半内踩中过就跳，否则慢慢停；来回往返播，不跳帧
    const d = dancer();
    if(d){
      const going = G.state === 'play' && songTime() - G.lastHit < beatLen()*1.6 && now - G.stumble > 420;
      const target = going ? (G.boost || 1) : 0;
      G.speed += (target - G.speed)*(target > G.speed ? 0.18 : 0.06);
      G.pos += G.speed*d.fps*dt*G.dir;
      if(G.pos >= d.n - 1){ G.pos = d.n - 1; G.dir = -1; }
      if(G.pos <= 0){ G.pos = 0; G.dir = 1; }
    }
    drawDancer();
  }

  function drawDancer(){
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, cvs.width, cvs.height);
    const d = dancer(), h = S.hoop, k = DPR;
    // 先画针线（在人物后面）
    if(window.drawNeedleLayer){ ctx.save(); ctx.scale(k, k); window.drawNeedleLayer(ctx); ctx.restore(); }
    let img, figH, base, w, hh;
    if(d && frames[d.id]){
      const fr = frames[d.id], i = Math.round(G.pos);
      img = fr[i] && fr[i].complete && fr[i].naturalWidth ? fr[i] : fr.find(f => f.complete && f.naturalWidth);
      figH = d.figH; base = d.base; w = d.w; hh = d.h;
    }
    if(!img){ if(!fallback.complete) return; img = fallback; figH = 987; base = 1005; w = 760; hh = 1030; }
    // 先在小画布里叠上丝线斜纹
    if(work.width !== w || work.height !== hh){ work.width = w; work.height = hh; }
    wctx.globalCompositeOperation = 'source-over'; wctx.clearRect(0, 0, w, hh); wctx.drawImage(img, 0, 0, w, hh);
    wctx.globalCompositeOperation = 'source-atop'; wctx.fillStyle = wctx.createPattern(grain, 'repeat'); wctx.fillRect(0, 0, w, hh);
    const sc = 1.12*h.r/figH;
    // 趔趄：一歪一晃再回正；跟着拍子轻轻一沉
    const st = (performance.now() - G.stumble)/1000;
    const wob = st < 0.6 ? Math.exp(-st*6)*Math.sin(st*26) : 0;
    let pulse = 0;
    if(G.state === 'play'){ pulse = Math.exp(-beatPhase(songTime())*7)*Math.min(1, G.speed); }
    ctx.save(); ctx.scale(k, k);
    ctx.beginPath(); ctx.arc(h.cx, h.cy, h.r*0.995, 0, Math.PI*2); ctx.clip();
    ctx.translate(h.cx + wob*10, h.cy + 0.6*h.r);
    ctx.rotate(wob*0.09);
    ctx.scale(sc*(1 + 0.006*pulse), sc*(1 - 0.012*pulse));
    ctx.shadowColor = 'rgba(40,20,10,0.28)'; ctx.shadowBlur = 10; ctx.shadowOffsetX = 2*k; ctx.shadowOffsetY = 4*k;
    if(st < 0.5){ ctx.filter = `saturate(${0.5 + st}) brightness(${0.85 + st*0.3})`; }
    ctx.drawImage(work, -w/2, -base);
    ctx.restore();
  }

  // ── 轨道、绣球、判定反馈、分数（画在最上层 fx）──
  function drawFx(fx){
    if(S.mode !== 'dance') return;
    const C = chart(); if(!C) return;
    const kz = Render.view.k, h = S.hoop, P = (a, r) => ({x:h.cx + Math.cos(a)*r*kz, y:h.cy + Math.sin(a)*r*kz});
    const lead = G.diff === '轻松' ? 2.4 : 2.0, now = performance.now();
    const ballR = Math.max(15, 21*kz*1.25);                 // 绣球半径（屏幕像素）
    fx.save(); fx.lineCap = 'round';
    // 两根丝线轨道
    for(const lane of [0, 1]){
      const fl = Math.max(0, 1 - (now - G.flash[lane])/200);
      fx.strokeStyle = `rgba(${BALLS[lane].track},${0.4 + 0.45*fl})`; fx.lineWidth = 2 + 2.5*fl; fx.setLineDash([8, 6]);
      fx.beginPath();
      if(lane === 0) fx.arc(h.cx, h.cy, RT*kz, Math.PI/2, Math.PI*1.5); else fx.arc(h.cx, h.cy, RT*kz, -Math.PI/2, Math.PI/2);
      fx.stroke();
    }
    fx.setLineDash([]);
    // 针眼（判定点）：跟拍子一缩一放
    const tgt = P(Math.PI/2, RT);
    let pulse = 0;
    if(G.state === 'play'){ pulse = Math.exp(-beatPhase(songTime())*6); }
    const lastFx = G.fxs[G.fxs.length - 1], hitFl = lastFx ? Math.max(0, 1 - (now - lastFx.t)/260) : 0;
    const ringCol = lastFx && hitFl > 0 ? (lastFx.type === 'burst' ? '214,160,50' : lastFx.type === 'ring' ? '250,246,238' : '120,24,30') : '70,46,28';
    fx.lineWidth = 3 + 2*hitFl;
    fx.strokeStyle = `rgba(${ringCol},.95)`;
    fx.beginPath(); fx.ellipse(tgt.x, tgt.y, ballR*1.25 + 4*pulse, ballR*0.8 + 3*pulse, 0, 0, Math.PI*2); fx.stroke();
    fx.strokeStyle = 'rgba(255,244,220,.8)'; fx.lineWidth = 1;
    fx.beginPath(); fx.ellipse(tgt.x, tgt.y, ballR*1.25 + 4*pulse - 2.5, ballR*0.8 + 3*pulse - 2.5, 0, 0, Math.PI*2); fx.stroke();
    fx.font = '700 18px "Songti SC","SimSun",serif'; fx.textAlign = 'center'; fx.textBaseline = 'middle';
    fx.fillStyle = 'rgba(160,24,44,.95)'; fx.fillText('F', tgt.x - ballR*2.1, tgt.y + 2);
    fx.fillStyle = 'rgba(40,70,140,.95)'; fx.fillText('J', tgt.x + ballR*2.1, tgt.y + 2);
    // 绣球：沿轨道滚，边滚边转，后面拖一缕丝穗
    if(G.state === 'play'){
      const t = songTime();
      for(let i = G.idx; i < G.notes.length; i++){
        const n = G.notes[i], dt = n.t - t;
        if(dt > lead) break;
        if(n.hit) continue;
        const u = Math.max(-0.08, dt/lead), sgn = n.lane ? -1 : 1, a = Math.PI/2 + sgn*Math.PI*u;
        const p = P(a, RT), rr = ballR*(0.78 + 0.22*(1 - Math.max(0, u)));
        const back = {x:Math.cos(a + sgn*0.06), y:Math.sin(a + sgn*0.06)};           // 来的方向
        const tx = h.cx + back.x*RT*kz - p.x, ty = h.cy + back.y*RT*kz - p.y, tl = Math.hypot(tx, ty) || 1;
        drawTassel(fx, p, {x:tx/tl, y:ty/tl}, rr, BALLS[n.lane].tassel, t + i);
        fx.save(); fx.translate(p.x, p.y); fx.rotate(-sgn*u*Math.PI*RT*kz/rr*0.5);
        fx.shadowColor = 'rgba(30,15,5,.35)'; fx.shadowBlur = 6; fx.shadowOffsetY = 3;
        fx.drawImage(BALLS[n.lane].img, -rr*1.087, -rr*1.087, rr*2.174, rr*2.174);
        fx.restore();
      }
      // 进度：绷圈外沿一道细金线
      const prog = Math.min(1, Math.max(0, (t - C.beat0)/(C.end - C.beat0)));
      fx.strokeStyle = 'rgba(214,170,80,.85)'; fx.lineWidth = 2;
      fx.beginPath(); fx.arc(h.cx, h.cy, h.r*1.115, -Math.PI/2, -Math.PI/2 + prog*Math.PI*2); fx.stroke();
      if(G.notes[0] && t < G.notes[0].t - 0.5){
        fx.font = '24px "Songti SC","SimSun",serif'; fx.fillStyle = 'rgba(70,46,28,.75)';
        const left = Math.ceil((G.notes[0].t - t)/beatLen());
        fx.fillText(left > 4 ? '预　备' : '一二三四'.slice(0, Math.max(1, 5 - left)), h.cx, h.cy - h.r*0.66);
      }
    }
    // 进度条：圆绷正下方一条丝带，针走在最前面，按乐句（8 小节）画刻度
    if(G.state !== 'idle'){
      const t = G.state === 'play' ? songTime() : C.end;
      const prog = Math.min(1, Math.max(0, (t - C.beat0)/(C.end - C.beat0)));
      const bw = Math.min(460, S.W*0.36), bx = h.cx - bw/2, by = Math.min(S.H - 64, h.cy + h.r*1.2 + 26), bh = 8;
      const fmt = s => `${Math.floor(s/60)}:${String(Math.floor(s%60)).padStart(2, '0')}`;
      fx.save();
      fx.fillStyle = 'rgba(40,26,16,.5)'; fx.beginPath(); fx.roundRect(bx - 12, by - 14, bw + 92, bh + 28, 4); fx.fill();
      fx.fillStyle = 'rgba(251,244,228,.25)'; fx.beginPath(); fx.roundRect(bx, by, bw, bh, 4); fx.fill();
      const gr = fx.createLinearGradient(bx, 0, bx + bw, 0);
      gr.addColorStop(0, '#8e1a2c'); gr.addColorStop(0.6, '#c2503a'); gr.addColorStop(1, '#e2b04a');
      fx.fillStyle = gr; fx.beginPath(); fx.roundRect(bx, by, Math.max(bh, bw*prog), bh, 4); fx.fill();
      // 丝带上的细纹
      fx.strokeStyle = 'rgba(255,240,210,.25)'; fx.lineWidth = 1;
      fx.beginPath(); fx.moveTo(bx + 3, by + bh/2); fx.lineTo(bx + Math.max(bh, bw*prog) - 3, by + bh/2); fx.stroke();
      // 乐句刻度
      const B = C.beats, total = C.end - C.beat0;
      if(B) for(let q = 32; q < B.length - 1; q += 32){
        const x = bx + bw*(B[q] - C.beat0)/total;
        fx.fillStyle = 'rgba(251,244,228,.55)'; fx.fillRect(x - 0.5, by - 3, 1, bh + 6);
      }
      // 走在前面的针
      const nx = bx + bw*prog, ny = by + bh/2;
      fx.strokeStyle = '#eef1f4'; fx.lineWidth = 2; fx.lineCap = 'round';
      fx.beginPath(); fx.moveTo(nx - 2, ny + 9); fx.lineTo(nx + 6, ny - 11); fx.stroke();
      fx.strokeStyle = '#4b5056'; fx.lineWidth = 0.8;
      fx.beginPath(); fx.ellipse(nx + 4.6, ny - 7.6, 1.8, 0.7, -1.2, 0, Math.PI*2); fx.stroke();
      fx.fillStyle = '#fbf3e2'; fx.font = '13px "Songti SC","SimSun",serif'; fx.textAlign = 'left'; fx.textBaseline = 'middle';
      fx.fillText(`${fmt(Math.max(0, t - C.beat0))} / ${fmt(total)}`, bx + bw + 12, ny + 1);
      fx.restore();
    }
    // 判定特效
    G.fxs = G.fxs.filter(e => now - e.t < 800);
    for(const e of G.fxs){
      const k = (now - e.t)/1000;
      if(e.type === 'burst' || e.type === 'ring'){
        const gold = e.type === 'burst', R = ballR*(1.1 + k*(gold ? 6 : 4)), al = Math.max(0, 1 - k/(gold ? 0.5 : 0.35));
        fx.strokeStyle = gold ? `rgba(226,176,64,${al})` : `rgba(255,250,240,${al})`; fx.lineWidth = gold ? 4*al + 1 : 3*al + 1;
        fx.beginPath(); fx.ellipse(tgt.x, tgt.y, R*1.2, R*0.8, 0, 0, Math.PI*2); fx.stroke();
        if(gold){                                                      // 金线迸开 + 花瓣
          for(let q = 0; q < 10; q++){
            const a = q/10*Math.PI*2 + 0.3, r0 = ballR*(1 + k*5), r1 = r0 + ballR*0.9*al;
            fx.strokeStyle = `rgba(232,186,80,${al})`; fx.lineWidth = 2;
            fx.beginPath(); fx.moveTo(tgt.x + Math.cos(a)*r0*1.2, tgt.y + Math.sin(a)*r0*0.8); fx.lineTo(tgt.x + Math.cos(a)*r1*1.2, tgt.y + Math.sin(a)*r1*0.8); fx.stroke();
          }
          for(let q = 0; q < 6; q++){
            const a = q/6*Math.PI*2 + e.lane, d = ballR*(0.5 + k*7), py = tgt.y + Math.sin(a)*d*0.7 + k*k*220;
            fx.fillStyle = e.lane ? `rgba(60,96,170,${al})` : `rgba(200,30,50,${al})`;
            fx.beginPath(); fx.ellipse(tgt.x + Math.cos(a)*d, py, ballR*0.32, ballR*0.18, a + k*6, 0, Math.PI*2); fx.fill();
          }
        }
      } else if(e.type === 'drop'){                                     // 绣球掉下去
        const al = Math.max(0, 1 - k/0.7), rr = ballR*0.85;
        fx.save(); fx.globalAlpha = al;
        fx.translate(tgt.x + (e.lane ? 1 : -1)*(ballR*1.4 + k*60), tgt.y + k*k*600);
        fx.rotate(k*5*(e.lane ? 1 : -1)); fx.filter = 'grayscale(0.7) brightness(0.8)';
        fx.drawImage(BALLS[e.lane].img, -rr*1.087, -rr*1.087, rr*2.174, rr*2.174);
        fx.restore();
      }
    }
    // 判定字：妙 / 好 / 失（大字，失会抖）
    G.pops = G.pops.filter(p => now - p.t < 700);
    for(const p of G.pops){
      const k = (now - p.t)/700, s = (p.kind === '妙' ? 1.25 : 1.05)*(1 + 0.5*Math.exp(-k*10));
      const shake = p.kind === '失' ? Math.sin(k*60)*6*(1 - k) : 0;
      fx.save(); fx.globalAlpha = 1 - k*k;
      fx.translate(tgt.x + shake, tgt.y - ballR*3.2 - k*24); fx.scale(s, s);
      fx.font = '700 38px "STKaiti","Kaiti SC","KaiTi","Songti SC","SimSun",serif';
      fx.lineWidth = 6; fx.strokeStyle = 'rgba(255,250,240,.95)';
      fx.fillStyle = p.kind === '妙' ? '#b07a12' : p.kind === '好' ? '#8e2a2e' : '#4a4440';
      fx.strokeText(p.kind, 0, 0); fx.fillText(p.kind, 0, 0);
      fx.restore();
    }
    // 绣纹升档提示：绷面上方，一个小纹样 + 字
    if(G.toast && now - G.toast.t < 1800){
      const k = (now - G.toast.t)/1800, al = k < 0.15 ? k/0.15 : k > 0.75 ? (1 - k)/0.25 : 1;
      const m = MOTIFS.find(x => x.name === G.toast.name), y = h.cy - h.r*0.62 - (1 - al)*8;
      fx.save(); fx.globalAlpha = al;
      fx.fillStyle = 'rgba(251,244,228,.88)'; fx.strokeStyle = 'rgba(142,42,46,.6)'; fx.lineWidth = 1;
      fx.beginPath(); fx.roundRect(h.cx - 92, y - 24, 184, 48, 4); fx.fill(); fx.stroke();
      if(m){
        const sc = 34/Math.max(m.w, m.h);
        fx.save(); fx.translate(h.cx - 58, y); fx.scale(sc, sc);
        m.regions.forEach(r => { fx.fillStyle = `rgb(${r.col.join(',')})`; fx.beginPath();
          r.polys.forEach(pp => { for(let q = 0; q < pp.length; q += 2) q ? fx.lineTo(pp[q], pp[q+1]) : fx.moveTo(pp[q], pp[q+1]); fx.closePath(); });
          fx.fill('evenodd'); });
        fx.restore();
      }
      fx.textAlign = 'left'; fx.fillStyle = '#8e2a2e'; fx.font = '600 18px "Songti SC","SimSun",serif';
      fx.fillText(`绣纹 · ${G.toast.name}`, h.cx - 32, y + 1);
      fx.restore();
    }
    // 连击和分数：圆绷右侧竖排
    if(G.state !== 'idle'){
      fx.textAlign = 'center';
      const x = h.cx + h.r*1.28, y0 = h.cy - h.r*0.55;
      fx.fillStyle = 'rgba(40,26,16,.45)'; fx.fillRect(x - 34, y0 - 28, 68, 156);
      const big = G.combo >= 10 ? 1 + 0.15*Math.max(0, 1 - (now - (G.pops[G.pops.length-1]?.t || 0))/200) : 1;
      fx.fillStyle = G.combo >= 25 ? '#f2cf7a' : '#fbf3e2';
      fx.font = `600 ${Math.round(30*big)}px "Songti SC","SimSun",serif`; fx.fillText(String(G.combo), x, y0 + 4);
      fx.fillStyle = '#fbf3e2'; fx.font = '13px "Songti SC","SimSun",serif'; fx.fillText('连　击', x, y0 + 30);
      fx.font = '600 17px "Songti SC","SimSun",serif'; fx.fillText(String(G.score), x, y0 + 72);
      fx.font = '13px "Songti SC","SimSun",serif'; fx.fillText('得　分', x, y0 + 96);
      if(G.border === '随连击'){ fx.fillStyle = '#f2cf7a'; fx.font = '600 14px "Songti SC","SimSun",serif'; fx.fillText('绣 ' + TIERS[G.tier].n, x, y0 + 116); }
    }
    fx.restore();
  }
  function drawTassel(fx, p, dir, rr, col, seed){
    const sw = Math.sin(performance.now()/180 + seed)*0.25;
    const ax = dir.x*Math.cos(sw) - dir.y*Math.sin(sw), ay = dir.x*Math.sin(sw) + dir.y*Math.cos(sw);
    const kx = p.x + ax*rr*1.15, ky = p.y + ay*rr*1.15;
    fx.strokeStyle = col; fx.lineWidth = 2.2;
    fx.beginPath(); fx.moveTo(p.x + ax*rr*0.9, p.y + ay*rr*0.9); fx.lineTo(kx, ky); fx.stroke();
    fx.fillStyle = '#d9a640'; fx.beginPath(); fx.arc(kx, ky, rr*0.16, 0, Math.PI*2); fx.fill();
    for(let q = -3; q <= 3; q++){
      const a = q*0.09, ex = kx + (ax*Math.cos(a) - ay*Math.sin(a))*rr*1.05, ey = ky + (ax*Math.sin(a) + ay*Math.cos(a))*rr*1.05;
      fx.strokeStyle = col; fx.lineWidth = 1.1; fx.beginPath(); fx.moveTo(kx, ky); fx.lineTo(ex, ey); fx.stroke();
    }
  }

  // ── 结算 ──
  function grade(){
    const N = G.notes.length || 1, acc = (G.counts.妙 + G.counts.好*0.6)/N;
    return {acc, name: acc >= 0.95 ? '神品' : acc >= 0.85 ? '妙品' : acc >= 0.7 ? '能品' : '习作'};
  }
  function showResult(){
    const g = grade(), el = document.getElementById('result');
    el.querySelector('.grade').textContent = g.name;
    el.querySelector('.stat').innerHTML =
      `得分 <b>${G.score}</b>　准确 <b>${Math.round(g.acc*100)}%</b><br>妙 ${G.counts.妙}　好 ${G.counts.好}　失 ${G.counts.失}　最长连击 ${G.maxCombo}`;
    el.querySelector('#keepBtn').textContent = `留下这圈${G.border === '混搭' || G.border === '随连击' ? '花边' : G.border}`;
    el.hidden = false;
  }
  function hideResult(){ document.getElementById('result').hidden = true; }
  function keep(){
    if(G.stash){ S.items = G.stash.concat(S.items); G.stash = null; redrawAll(); afterChange(); }
    hideResult(); G.state = 'idle'; syncUI();
  }
  function restore(){
    stop(); G.state = 'idle'; hideResult();
    if(G.stash){ S.items = G.stash; G.stash = null; redrawAll(); afterChange(); }
    syncUI();
  }

  function key(e){
    if(e.repeat) return true;
    const k = e.key.toLowerCase();
    if(k === 'f' || k === 'd' || e.key === 'ArrowLeft'){ press(0); return true; }
    if(k === 'j' || k === 'k' || e.key === 'ArrowRight'){ press(1); return true; }
    if(e.key === ' '){ if(G.state !== 'play') start(); return true; }
    if(e.key === 'Escape'){ if(G.state === 'play'){ stop(); G.state = 'idle'; syncUI(); } return true; }
    return false;
  }
  function enter(){ const d = dancer(); if(d) loadDancer(d); syncUI(); }
  function leave(){ if(G.state === 'play'){ stop(); G.state = 'idle'; } if(G.stash) restore(); hideResult(); }

  // ── 面板 ──
  function syncUI(){
    const q = (sel, f) => document.querySelectorAll(sel).forEach(f);
    q('.song', b => b.classList.toggle('on', +b.dataset.i === G.song));
    q('.diff', b => b.classList.toggle('on', b.dataset.d === G.diff));
    q('.who', b => b.classList.toggle('on', +b.dataset.i === G.dancer));
    q('.bd', b => b.classList.toggle('on', b.dataset.n === G.border));
    const btn = document.getElementById('playBtn');
    if(btn) btn.textContent = G.state === 'play' ? '演出中…（Esc 停）' : '开　演（空格）';
  }
  function bindUI(){
    const songs = document.getElementById('songRow'), who = document.getElementById('whoRow'), bd = document.getElementById('borderGrid');
    (window.CHARTS || []).forEach((c, i) => { const b = document.createElement('button'); b.className = 'chip song'; b.dataset.i = i; b.textContent = c.name; songs.appendChild(b); });
    (DANCERS.length ? DANCERS : [{name:'京剧花旦'}]).forEach((d, i) => { const b = document.createElement('button'); b.className = 'chip who'; b.dataset.i = i; b.textContent = d.name; who.appendChild(b); });
    BORDERS.forEach(n => {
      const m = MOTIFS.find(x => x.name === n), b = document.createElement('button');
      b.className = 'bd'; b.dataset.n = n; b.title = n;
      const label = n === '随连击' ? '升级' : '混搭';
      b.innerHTML = (m ? motifSVG(m) : `<svg viewBox="-0.56 -0.56 1.12 1.12"><text x="0" y="0.12" font-size="0.32" text-anchor="middle" fill="#8e2a2e">${label}</text></svg>`) + `<span>${n}</span>`;
      bd.appendChild(b);
    });
    const free = f => e => { if(G.state !== 'play'){ f(e.currentTarget); syncUI(); } };
    document.querySelectorAll('.song').forEach(b => b.onclick = free(b => G.song = +b.dataset.i));
    document.querySelectorAll('.diff').forEach(b => b.onclick = free(b => G.diff = b.dataset.d));
    document.querySelectorAll('.who').forEach(b => b.onclick = free(b => { G.dancer = +b.dataset.i; G.pos = 0; const d = dancer(); if(d) loadDancer(d); }));
    document.querySelectorAll('.bd').forEach(b => b.onclick = free(b => G.border = b.dataset.n));
    document.getElementById('playBtn').onclick = () => { if(G.state === 'play'){ stop(); G.state = 'idle'; syncUI(); } else start(); };
    document.getElementById('calR').oninput = e => { G.offsetMs = +e.target.value; document.getElementById('calV').textContent = (G.offsetMs > 0 ? '+' : '') + G.offsetMs + ' 毫秒'; };
    document.getElementById('againBtn').onclick = start;
    document.getElementById('keepBtn').onclick = keep;
    document.getElementById('backBtn').onclick = restore;
    syncUI();
  }

  return {update, drawFx, key, enter, leave, bindUI, start, press, songTime, get G(){ return G; }};
})();
