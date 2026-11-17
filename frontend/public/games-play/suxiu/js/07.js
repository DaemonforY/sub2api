'use strict';
// ════════════════════════════════════════════════════════════════
//  针法：把「笔画轮廓＋中线」「色块多边形」「点」变成一根根针脚
//  所有针脚都落在绷面坐标里：圆心 (0,0)，半径 500，y 向下
//  针脚记录：{x1,y1,x2,y2, c:[r,g,b], w:线宽, h:拱起高度 0..1}
// ════════════════════════════════════════════════════════════════
const ST = {
  SP: 2.6,      // 相邻两针的间距（绷面单位，约 2px）
  LW: 2.7,      // 丝线宽度：略宽于针距，线挨线，线间留浅沟
  MAXLEN: 26,   // 一针最长；更宽的面要分成长短针（套针）
};

function mulberry(seed){
  let a = seed >>> 0;
  return () => { a |= 0; a = a + 0x6D2B79F5 | 0; let t = Math.imul(a ^ a >>> 15, 1 | a);
    t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t; return ((t ^ t >>> 14) >>> 0) / 4294967296; };
}
// 一维平滑噪声：手绣时针的方向、边线的进出都是缓慢漂移的
function smoothNoise(rnd, len){
  const knots = []; for(let i = 0; i < len + 3; i++) knots.push(rnd()*2 - 1);
  return t => { const x = Math.max(0, t), i = Math.floor(x), f = x - i, s = f*f*(3 - 2*f);
    return knots[Math.min(i, knots.length-2)]*(1-s) + knots[Math.min(i+1, knots.length-1)]*s; };
}
const clamp01 = v => v < 0 ? 0 : v > 1 ? 1 : v;
const mixc = (a, b, t) => [a[0]+(b[0]-a[0])*t, a[1]+(b[1]-a[1])*t, a[2]+(b[2]-a[2])*t];
const scalec = (c, k) => [Math.min(255, c[0]*k), Math.min(255, c[1]*k), Math.min(255, c[2]*k)];
const vnorm = (x, y) => { const l = Math.hypot(x, y) || 1; return {x:x/l, y:y/l}; };

function resample(pts, step){
  const P = pts.map(p => Array.isArray(p) ? {x:p[0], y:p[1]} : p);
  const out = [{x:P[0].x, y:P[0].y}];
  let carry = 0;
  for(let i = 1; i < P.length; i++){
    const a = P[i-1], b = P[i], len = Math.hypot(b.x-a.x, b.y-a.y);
    if(len < 1e-9) continue;
    let d = step - carry;
    while(d <= len){ out.push({x:a.x+(b.x-a.x)*d/len, y:a.y+(b.y-a.y)*d/len}); d += step; }
    carry = len - (d - step);
  }
  const last = P[P.length-1], o = out[out.length-1];
  if(Math.hypot(o.x-last.x, o.y-last.y) > step*0.4) out.push({x:last.x, y:last.y});
  if(out.length < 2) out.push({x:last.x + step*0.01, y:last.y});
  return out;
}

const _hit = document.createElement('canvas').getContext('2d');

// ── 缎面针（平针）：沿中线一针挨一针横跨笔画 ──────────────────────
// outline: Path2D（局部坐标），median: 中线点列（局部坐标）
// map(x,y) → 绷面坐标；unit: 一个局部单位等于多少绷面单位
function genSatin(outline, median, {map, unit, pal, rnd}){
  const inP = (x, y) => _hit.isPointInPath(outline, x, y);
  // 这一笔自己的脾气：斜度、疏密、松紧、色偏都略有不同
  const slant   = 0.30 + (rnd() - 0.5)*0.34;
  const density = 0.86 + rnd()*0.28;
  const tension = 0.80 + rnd()*0.28;
  const shift   = (rnd() - 0.5)*0.16;
  const step = ST.SP*density/unit;
  let pts = resample(median, step);
  const extend = (p, t, sign) => {
    const arr = [];
    for(let k = step; k < 400/unit; k += step){
      const q = {x:p.x + t.x*k*sign, y:p.y + t.y*k*sign};
      if(!inP(q.x, q.y)) break; arr.push(q);
    }
    return arr;
  };
  const n0 = pts.length;
  const t0 = vnorm(pts[1].x-pts[0].x, pts[1].y-pts[0].y), te = vnorm(pts[n0-1].x-pts[n0-2].x, pts[n0-1].y-pts[n0-2].y);
  pts = [...extend(pts[0], t0, -1).reverse(), ...pts, ...extend(pts[n0-1], te, 1)];
  const N = pts.length;
  // 切线角：展开后高斯平滑，起笔收笔不散
  const ang = new Array(N);
  for(let i = 0; i < N; i++){
    const a = pts[Math.max(0,i-2)], b = pts[Math.min(N-1,i+2)];
    ang[i] = Math.atan2(b.y-a.y, b.x-a.x);
    if(i) ang[i] = ang[i-1] + Math.atan2(Math.sin(ang[i]-ang[i-1]), Math.cos(ang[i]-ang[i-1]));
  }
  const sig = Math.max(2, 18/(step*unit)), R = Math.ceil(sig*2.5), sm = new Array(N);
  for(let i = 0; i < N; i++){
    let s = 0, w = 0;
    for(let k = -R; k <= R; k++){ const j = Math.min(N-1, Math.max(0, i+k)), g = Math.exp(-k*k/(2*sig*sig)); s += ang[j]*g; w += g; }
    sm[i] = s/w;
  }
  const ms = 0.35/unit;
  const march = (p, u) => { let d = 0; while(d < 400/unit && inP(p.x+u.x*(d+ms), p.y+u.y*(d+ms))) d += ms; return d; };
  // 手的漂移：方向慢慢摆、两条边慢慢进出
  const wob = smoothNoise(rnd, 40), edgeA = smoothNoise(rnd, 40), edgeB = smoothNoise(rnd, 40);
  const out = [];
  for(let i = 0; i < N; i++){
    const p = pts[i];
    if(!inP(p.x, p.y)) continue;
    const tt = i*step*unit/60;                      // 每 60 单位一个噪声节点
    const a = sm[i] + Math.PI/2 - slant + wob(tt)*0.07 + (rnd()-0.5)*0.05;
    const u = {x:Math.cos(a), y:Math.sin(a)};
    const da = march(p, u) + (edgeA(tt)*0.7 + (rnd()-0.4)*0.6)/unit;
    const db = march(p, {x:-u.x, y:-u.y}) + (edgeB(tt)*0.7 + (rnd()-0.4)*0.6)/unit;
    const A = map(p.x - u.x*db, p.y - u.y*db), B = map(p.x + u.x*da, p.y + u.y*da);
    const f = i/(N-1);
    pushSplit(out, A, B, f, i, {pal, rnd, shift, tension});
  }
  return out;
}

// 一针太长就拆成长短针：分界参差，分段颜色从深到浅（套针晕色）
function pushSplit(out, A, B, f, i, {pal, rnd, shift, tension}){
  const L = Math.hypot(B.x-A.x, B.y-A.y);
  const nSeg = Math.max(1, Math.ceil(L/ST.MAXLEN));
  const cuts = [0];
  for(let s = 1; s < nSeg; s++) cuts.push(clamp01(s/nSeg + (i % 2 ? 0.16 : -0.16)/nSeg + (rnd()-0.5)*0.12/nSeg));
  cuts.push(1);
  const lum = 1 + (rnd()-0.5)*0.14;                 // 每根线自己的明暗
  const loose = rnd() < 0.035 ? 1.18 : 1;          // 偶尔一针松一点、鼓一点
  for(let s = 0; s < nSeg; s++){
    const t0 = cuts[s], t1 = cuts[s+1];
    const tc = nSeg > 1 ? (s + 0.5)/nSeg : 0.5;
    const ct = clamp01(nSeg > 1 ? 0.08 + 0.62*tc + 0.25*f + shift : 0.2 + 0.6*f + shift);
    out.push({
      x1:A.x + (B.x-A.x)*t0, y1:A.y + (B.y-A.y)*t0,
      x2:A.x + (B.x-A.x)*t1, y2:A.y + (B.y-A.y)*t1,
      c:scalec(mixc(pal.a, pal.b, ct), lum*loose*0.97),
      w:ST.LW*(0.88 + rnd()*0.24)*loose, h:clamp01(tension*loose*(0.85 + rnd()*0.2)),
    });
  }
}

// ── 色块：长短针铺满（纹样用）────────────────────────────────────
// polys: 若干闭合多边形（绷面坐标），外轮廓和洞一起按奇偶规则
function genFill(polys, {angle, pal, rnd}){
  const ca = Math.cos(angle), sa = Math.sin(angle);
  const rot = polys.map(P => P.map(p => ({x:p.x*ca + p.y*sa, y:-p.x*sa + p.y*ca})));
  let ymin = Infinity, ymax = -Infinity;
  rot.forEach(P => P.forEach(p => { ymin = Math.min(ymin, p.y); ymax = Math.max(ymax, p.y); }));
  const density = 0.9 + rnd()*0.2, shift = (rnd()-0.5)*0.12, tension = 0.82 + rnd()*0.2;
  const wob = smoothNoise(rnd, 60), edge = smoothNoise(rnd, 60);
  const back = (x, y) => ({x:x*ca - y*sa, y:x*sa + y*ca});
  const out = [];
  let row = 0;
  for(let y = ymin + ST.SP*0.5; y < ymax; y += ST.SP*density*(0.9 + rnd()*0.2), row++){
    const xs = [];
    rot.forEach(P => { for(let i = 0, j = P.length-1; i < P.length; j = i++){
      const a = P[i], b = P[j];
      if((a.y > y) !== (b.y > y)) xs.push(a.x + (y - a.y)*(b.x - a.x)/(b.y - a.y));
    }});
    xs.sort((a, b) => a - b);
    const segs = [];
    for(let k = 0; k + 1 < xs.length; k += 2) if(xs[k+1] - xs[k] > 0.8) segs.push([xs[k], xs[k+1]]);
    if(row % 2) segs.reverse();
    const fy = (y - ymin)/Math.max(1, ymax - ymin);
    for(const [xa0, xb0] of segs){
      const xa = xa0 + edge(xa0/40 + y/60)*0.5 - 0.2, xb = xb0 - edge(xb0/40 + y/50 + 7)*0.5 + 0.2;
      const L = xb - xa; if(L < 0.6) continue;
      // 长短针：每行的断点错开半针，再加一点随机
      const len = ST.MAXLEN*(0.75 + rnd()*0.3);
      const pieces = [];
      let x = xa, first = (row % 2 ? 0.5 : 0) * len + rnd()*len*0.3;
      if(first < len*0.3 || first > L) first = Math.min(L, len);
      let nx = xa + first;
      while(x < xb - 0.3){ nx = Math.min(xb, nx); pieces.push([x, nx]); x = nx; nx = x + len*(0.85 + rnd()*0.3); }
      if(row % 2) pieces.reverse();
      for(const [p0, p1] of pieces){
        const tilt = wob(y/50 + p0/80)*0.05 + (rnd()-0.5)*0.035;
        const dy = (p1 - p0)*Math.sin(tilt)*0.5;
        const A = back(p0, y - dy), B = back(p1, y + dy);
        const ct = clamp01(0.18 + 0.55*fy + 0.15*((p0+p1)/2 - xa)/L + shift);
        out.push({x1:A.x, y1:A.y, x2:B.x, y2:B.y,
          c:scalec(mixc(pal.a, pal.b, ct), 1 + (rnd()-0.5)*0.14),
          w:ST.LW*(0.88 + rnd()*0.24), h:clamp01(tension*(0.82 + rnd()*0.2))});
      }
    }
  }
  return out;
}

// ── 打籽绣：每颗籽是几段短线绕成的小圈 ──────────────────────────
function genKnotsAt(points, r, {pal, rnd}){
  const out = [];
  points.forEach((k, i) => {
    const a0 = rnd()*Math.PI*2, f = i/Math.max(1, points.length-1);
    const c = scalec(mixc(pal.a, pal.b, 0.3 + 0.5*f), 1 + (rnd()-0.5)*0.16);
    const rr = r*(0.85 + rnd()*0.3);
    for(let m = 0; m < 7; m++){
      const a = a0 + m*Math.PI*2/7, cx = k.x + Math.cos(a)*rr*0.42, cy = k.y + Math.sin(a)*rr*0.42;
      const ta = a + Math.PI/2, hl = rr*0.42;
      out.push({x1:cx - Math.cos(ta)*hl, y1:cy - Math.sin(ta)*hl, x2:cx + Math.cos(ta)*hl, y2:cy + Math.sin(ta)*hl,
        c, w:ST.LW*1.35, h:0.95});
    }
  });
  return out;
}

// 非汉字：字形栅格化，每个点打一颗籽
function genGlyphKnots(ch, cell, {pal, rnd}){
  const N = 40, c = document.createElement('canvas'); c.width = c.height = N;
  const x = c.getContext('2d');
  x.font = `600 ${N*0.82}px "PingFang SC","Helvetica Neue",sans-serif`;
  x.textAlign = 'center'; x.textBaseline = 'middle';
  const corner = /[，。、；：！？,.;:!?]/.test(ch);
  x.fillText(ch, N/2 + (corner ? N*0.3 : 0), N/2 + 1 - (corner ? N*0.3 : 0));
  const px = x.getImageData(0, 0, N, N).data, g = 2.6, pts = [];
  for(let gy = 1; gy < N; gy += g) for(let gx = 1; gx < N; gx += g){
    const ox = gx + (Math.floor(gy/g) % 2)*g/2;
    if(px[(Math.round(gy)*N + Math.round(Math.min(N-1, ox)))*4 + 3] > 110)
      pts.push({x:cell.x + ox/N*cell.s + (rnd()-0.5)*0.6, y:cell.y + gy/N*cell.s + (rnd()-0.5)*0.6});
  }
  return genKnotsAt(pts, cell.s/N*1.05, {pal, rnd});
}

// ── 手绘笔迹 → 带粗细的笔画轮廓 + 中线，再走缎面针 ────────────────
function brushToStroke(raw, width){
  // 平滑：Chaikin 两遍
  let P = raw.map(p => ({x:p.x, y:p.y}));
  for(let it = 0; it < 2 && P.length > 2; it++){
    const Q = [P[0]];
    for(let i = 0; i < P.length-1; i++){
      const a = P[i], b = P[i+1];
      Q.push({x:a.x*0.75 + b.x*0.25, y:a.y*0.75 + b.y*0.25}, {x:a.x*0.25 + b.x*0.75, y:a.y*0.25 + b.y*0.75});
    }
    Q.push(P[P.length-1]); P = Q;
  }
  const M = resample(P, 2);
  if(M.length < 2) return null;
  // 粗细：起笔收笔收尖，中段按手速略有起伏
  const n = M.length, L = (n - 1)*2;
  const left = [], right = [];
  for(let i = 0; i < n; i++){
    const a = M[Math.max(0, i-2)], b = M[Math.min(n-1, i+2)];
    const t = vnorm(b.x-a.x, b.y-a.y), nn = {x:-t.y, y:t.x};
    const s = i*2, taper = Math.min(1, s/Math.min(26, L*0.3) , (L - s)/Math.min(34, L*0.35));
    const w = width*0.5*(0.25 + 0.75*Math.sqrt(Math.max(0, taper)))*(0.92 + 0.08*Math.sin(s/37));
    left.push({x:M[i].x + nn.x*w, y:M[i].y + nn.y*w}); right.push({x:M[i].x - nn.x*w, y:M[i].y - nn.y*w});
  }
  const path = new Path2D();
  // 两头加半圆帽
  path.moveTo(left[0].x, left[0].y);
  left.forEach(p => path.lineTo(p.x, p.y));
  const e = M[n-1], r1 = Math.hypot(left[n-1].x-e.x, left[n-1].y-e.y);
  const ae = Math.atan2(left[n-1].y-e.y, left[n-1].x-e.x);
  path.arc(e.x, e.y, r1, ae, ae - Math.PI, true);
  for(let i = n-1; i >= 0; i--) path.lineTo(right[i].x, right[i].y);
  const s0 = M[0], r0 = Math.hypot(right[0].x-s0.x, right[0].y-s0.y);
  const as = Math.atan2(right[0].y-s0.y, right[0].x-s0.x);
  path.arc(s0.x, s0.y, r0, as, as - Math.PI, true);
  path.closePath();
  return {path, median:M};
}

// 多边形主轴方向（PCA）和细长程度
function principal(polys){
  let n = 0, mx = 0, my = 0;
  polys[0].forEach(p => { mx += p.x; my += p.y; n++; }); mx /= n; my /= n;
  let sxx = 0, syy = 0, sxy = 0;
  polys[0].forEach(p => { const dx = p.x-mx, dy = p.y-my; sxx += dx*dx; syy += dy*dy; sxy += dx*dy; });
  const ang = 0.5*Math.atan2(2*sxy, sxx - syy);
  const tr = sxx + syy, det = sxx*syy - sxy*sxy, d = Math.sqrt(Math.max(0, tr*tr/4 - det));
  const l1 = tr/2 + d, l2 = Math.max(1e-6, tr/2 - d);
  return {ang, aspect:Math.sqrt(l1/l2), cx:mx, cy:my};
}
