'use strict';
// ════════════════════════════════════════════════════════════════
//  园林书房动态场景（three.js）
//  后层 #scene（在绣面下）：窗外园景（风吹摆动、视差）→ 房间照片（窗洞挖空、竹影光斑）
//  前层 #air（在绣面上）：光柱、浮尘、飘进屋的竹叶和梅花瓣
//  世界坐标 = 屏幕像素，原点在屏幕中心，y 向上；z=0 平面与屏幕一一对应，
//  鼠标带动相机平移 + 视窗偏移，z=0 不动（圆绷不动），越远的层移得越多。
// ════════════════════════════════════════════════════════════════
const Scene = (() => {
  const st = {ready:false, W:1, H:1, img:{x:0, y:0, s:1}, iw:3840, ih:2160,
              wind:0.25, gust:0, mouse:{x:-9999, y:-9999}, par:{x:0, y:0}, burst:0};
  let THREE, back, air, camB, camA, D = 1000;
  let roomMat, gardenMat, gardenMesh, roomMesh, beams = [], dust, leaves = [];
  const FOV = 30, ZG = 650;

  const GLSL_NOISE = `
    float h1(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
    float n2(vec2 p){vec2 i=floor(p),f=fract(p);f=f*f*(3.-2.*f);
      return mix(mix(h1(i),h1(i+vec2(1,0)),f.x),mix(h1(i+vec2(0,1)),h1(i+vec2(1,1)),f.x),f.y);}
    float fbm(vec2 p){float s=0.,a=.5;for(int i=0;i<4;i++){s+=a*n2(p);p=p*2.03+17.1;a*=.5;}return s;}`;

  async function init(){
    try{
      // 优先用本地打包的 three（双击打开也能用），没有再去网上取
      THREE = window.THREE || await import('https://cdn.jsdelivr.net/npm/three@0.170.0/build/three.module.js');
    }catch(e){ console.warn('three.js 没加载到，用静态背景', e); return false; }
    const A = window.SCENE_ASSETS;
    // 照片贴图不做色彩空间解码：自定义着色器直接原样输出，颜色和原图一致
    const load = src => new Promise(res => { const t = new THREE.TextureLoader().load(src, () => res(t)); t.colorSpace = THREE.NoColorSpace; });
    const [roomT, gardenT, winT] = await Promise.all([load(A.room), load(A.garden), load(A.win)]);

    back = new THREE.WebGLRenderer({canvas:document.getElementById('scene'), antialias:false, alpha:false});
    air = new THREE.WebGLRenderer({canvas:document.getElementById('air'), antialias:true, alpha:true, premultipliedAlpha:true});
    for(const r of [back, air]){ r.setPixelRatio(Math.min(devicePixelRatio || 1, 1.5)); r.outputColorSpace = THREE.SRGBColorSpace; }
    air.setClearColor(0x000000, 0);
    camB = new THREE.PerspectiveCamera(FOV, 1, 10, 5000);
    camA = new THREE.PerspectiveCamera(FOV, 1, 10, 5000);
    const sceneB = new THREE.Scene(), sceneA = new THREE.Scene();
    sceneB.background = new THREE.Color(0x2a1d14);

    // ── 窗外园景：叶子随风摆，太湖石不动 ──
    gardenMat = new THREE.ShaderMaterial({
      uniforms:{uTex:{value:gardenT}, uT:{value:0}, uWind:{value:0.3}},
      vertexShader:`varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.);}`,
      fragmentShader:`uniform sampler2D uTex;uniform float uT,uWind;varying vec2 vUv;${GLSL_NOISE}
        void main(){
          vec3 c0=texture2D(uTex,vUv).rgb;
          float green=smoothstep(0.0,0.08,c0.g-max(c0.r,c0.b)*0.92);
          vec2 q=vUv*vec2(9.,6.);
          vec2 off=vec2(fbm(q+vec2(uT*0.35,0.)),fbm(q+vec2(5.2,uT*0.3)))-0.5;
          off*=0.006*(0.4+uWind*1.6)*(0.3+0.7*vUv.y);
          vec3 c=texture2D(uTex,vUv+off*mix(0.25,1.0,green)).rgb;
          c*=0.97+0.06*fbm(vUv*3.+uT*0.08);          // 云影掠过
          gl_FragColor=vec4(c,1.);
        }`,
    });
    gardenMesh = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), gardenMat);
    gardenMesh.position.z = -ZG;
    sceneB.add(gardenMesh);

    // ── 房间：窗洞挖空；阳光穿过竹叶，在墙和桌上投下会动的光斑 ──
    roomMat = new THREE.ShaderMaterial({
      uniforms:{uTex:{value:roomT}, uWin:{value:winT}, uT:{value:0}, uWind:{value:0.3}, uRes:{value:new THREE.Vector2(1, 1)}},
      transparent:true, depthWrite:false,
      vertexShader:`varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.);}`,
      fragmentShader:`uniform sampler2D uTex,uWin;uniform float uT,uWind;varying vec2 vUv;${GLSL_NOISE}
        void main(){
          vec3 c=texture2D(uTex,vUv).rgb;
          float win=texture2D(uWin,vUv).r;
          // 光斑：窗外竹叶的影子，风大时晃得快
          vec2 p=vec2(vUv.x*16./9.,vUv.y)*7.;
          float sway=uT*(0.12+uWind*0.35);
          float d=fbm(p+vec2(sin(sway)*0.6,cos(sway*0.8)*0.4)+vec2(uT*0.02,0.));
          float spot=smoothstep(0.52,0.72,d);
          float lum=dot(c,vec3(0.3,0.59,0.11));
          // 只落在受光面上（墙、桌面、绷架），越亮越明显；窗外不加
          float recv=smoothstep(0.25,0.75,lum)*(1.-win);
          // 光从右上窗口来：越靠右上越强
          float fromWin=smoothstep(-0.2,1.0,vUv.x*0.7+vUv.y*0.6);
          c*=1.0+recv*fromWin*(spot*0.16-0.05);
          c+=vec3(1.,.9,.72)*recv*fromWin*spot*0.05;
          c*=0.985+0.03*fbm(vec2(uT*0.05,1.3));
          gl_FragColor=vec4(c,1.-win);
        }`,
    });
    roomMesh = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), roomMat);
    roomMesh.renderOrder = 10;
    sceneB.add(roomMesh);

    // ── 前层：光柱 ──
    const beamMat = () => new THREE.ShaderMaterial({
      uniforms:{uT:{value:0}, uI:{value:0.06}, uSeed:{value:Math.random()*10}},
      transparent:true, depthWrite:false, blending:THREE.AdditiveBlending,
      vertexShader:`varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.);}`,
      fragmentShader:`uniform float uT,uI,uSeed;varying vec2 vUv;${GLSL_NOISE}
        void main(){
          float across=smoothstep(0.,0.35,vUv.x)*smoothstep(1.,0.65,vUv.x);
          float along=smoothstep(0.,0.25,vUv.y)*smoothstep(1.,0.45,vUv.y);
          float flick=0.6+0.4*fbm(vec2(vUv.x*3.+uSeed,uT*0.25));
          float a=across*along*flick*uI;
          gl_FragColor=vec4(vec3(1.,.93,.78)*a,a);
        }`,
    });
    for(let i = 0; i < 4; i++){
      const m = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), beamMat());
      m.userData = {i};
      beams.push(m); sceneA.add(m);
    }

    // ── 前层：浮尘（在光里才看得见；鼠标搅动）──
    const N = 520, pos = new Float32Array(N*3), seed = new Float32Array(N);
    for(let i = 0; i < N; i++){ seed[i] = Math.random(); }
    const dg = new THREE.BufferGeometry();
    dg.setAttribute('position', new THREE.BufferAttribute(pos, 3));
    dg.setAttribute('aSeed', new THREE.BufferAttribute(seed, 1));
    dust = new THREE.Points(dg, new THREE.ShaderMaterial({
      uniforms:{uT:{value:0}, uPx:{value:1}},
      transparent:true, depthWrite:false, blending:THREE.AdditiveBlending,
      vertexShader:`attribute float aSeed;uniform float uT,uPx;varying float vA;
        void main(){vec4 mv=modelViewMatrix*vec4(position,1.);gl_Position=projectionMatrix*mv;
          float tw=0.55+0.45*sin(uT*(0.8+aSeed*1.7)+aSeed*40.);
          vA=tw*(0.35+0.65*aSeed);
          gl_PointSize=(1.2+aSeed*2.2)*uPx*(1000./-mv.z);}`,
      fragmentShader:`varying float vA;void main(){float d=length(gl_PointCoord-.5);float a=smoothstep(.5,0.,d)*vA*0.55;
          gl_FragColor=vec4(vec3(1.,.95,.82)*a,a);}`,
    }));
    dust.userData = {N, vel:new Float32Array(N*3), init:false};
    sceneA.add(dust);

    // ── 前层：飘进来的竹叶、梅花瓣（3D 翻转）──
    const kinds = [
      {tex:makeLeafTex(THREE, '#4e7a3e', '#86ab63', 'bamboo', 3), w:34, h:8, n:12},
      {tex:makeLeafTex(THREE, '#e9a6b4', '#f8dbe0', 'petal', 4), w:12, h:11, n:16},
      {tex:makeLeafTex(THREE, '#8fae6a', '#c4d69a', 'willow', 3), w:26, h:5, n:8},
    ];
    for(const k of kinds){
      const mesh = new THREE.InstancedMesh(new THREE.PlaneGeometry(1, 1),
        new THREE.MeshBasicMaterial({map:k.tex, transparent:true, side:THREE.DoubleSide, depthWrite:false}), k.n);
      mesh.userData = {k, P:[...Array(k.n)].map(() => ({alive:false}))};
      mesh.frustumCulled = false;
      leaves.push(mesh); sceneA.add(mesh);
    }

    st.sceneB = sceneB; st.sceneA = sceneA;
    st.ready = true;
    layout();
    return true;
  }

  // 程序画叶子 / 花瓣贴图（带一点虚焦）
  function makeLeafTex(THREE, c1, c2, kind, blur){
    const cv = document.createElement('canvas'); cv.width = 256; cv.height = kind === 'petal' ? 256 : 64;
    const x = cv.getContext('2d'), w = cv.width, h = cv.height;
    x.filter = `blur(${blur}px)`;
    const g = x.createLinearGradient(0, 0, w, h); g.addColorStop(0, c1); g.addColorStop(1, c2);
    x.fillStyle = g;
    x.beginPath();
    if(kind === 'petal'){
      x.ellipse(w/2, h/2, w*0.42, h*0.4, 0, 0, Math.PI*2);
      x.fill();
      x.globalCompositeOperation = 'destination-out';
      x.beginPath(); x.arc(w/2, h*0.04, w*0.09, 0, Math.PI*2); x.fill();
    } else {
      x.moveTo(4, h/2);
      x.quadraticCurveTo(w*0.35, kind === 'willow' ? h*0.12 : 2, w - 4, h/2);
      x.quadraticCurveTo(w*0.35, kind === 'willow' ? h*0.88 : h - 2, 4, h/2);
      x.fill();
      x.strokeStyle = 'rgba(255,255,255,.25)'; x.lineWidth = 1.5;
      x.beginPath(); x.moveTo(8, h/2); x.lineTo(w - 10, h/2); x.stroke();
    }
    const t = new THREE.CanvasTexture(cv); t.colorSpace = THREE.SRGBColorSpace; return t;
  }

  // 图像归一化坐标 → 世界坐标（z=0 平面）
  const imgToWorld = (u, v) => ({x:st.img.x + u*st.iw*st.img.s - st.W/2, y:st.H/2 - (st.img.y + v*st.ih*st.img.s)});

  function layout(){
    if(!st.ready) return;
    const {W, H} = st;
    D = (H/2)/Math.tan(FOV*Math.PI/360);
    for(const c of [camB, camA]){ c.aspect = W/H; c.position.set(0, 0, D); c.updateProjectionMatrix(); }
    back.setSize(W, H, false); air.setSize(W, H, false);
    const iw = st.iw*st.img.s, ih = st.ih*st.img.s;
    const cx = st.img.x + iw/2 - W/2, cy = H/2 - (st.img.y + ih/2);
    roomMesh.position.set(cx, cy, 0); roomMesh.scale.set(iw, ih, 1);
    const f = (D + ZG)/D;
    gardenMesh.position.set(cx*f, cy*f, -ZG); gardenMesh.scale.set(iw*f*1.04, ih*f*1.04, 1);
    // 光柱：从右上的窗斜斜照下来
    const tl = imgToWorld(0.62, 0.0);
    beams.forEach((m, i) => {
      const len = H*1.1, wid = iw*(0.05 + 0.03*(i % 2));
      m.scale.set(wid, len, 1);
      const p = imgToWorld(0.40 + i*0.13, 0.42);
      m.position.set(p.x, p.y, 40 + i*30);
      m.rotation.z = 0.42;
    });
    dust.material.uniforms.uPx.value = Math.min(devicePixelRatio || 1, 1.5);
    roomMat.uniforms.uRes.value.set(W, H);
  }

  function resize(W, H, img){ st.W = W; st.H = H; st.img = img; layout(); }

  // 窗的范围（图像归一化），用来判断鼠标是否在窗前
  function overWindow(sx, sy){
    const u = (sx - st.img.x)/(st.iw*st.img.s), v = (sy - st.img.y)/(st.ih*st.img.s);
    return (u > 0.255 && u < 0.86 && v > 0 && v < 0.585) || (u > 0.963 && v < 0.555);
  }

  function spawnLeaf(burst){
    const mesh = leaves[Math.floor(Math.random()*leaves.length)];
    const P = mesh.userData.P.find(p => !p.alive);
    if(!P) return;
    const s = imgToWorld(0.3 + Math.random()*0.56, 0.02 + Math.random()*0.3);
    Object.assign(P, {alive:true, x:s.x, y:s.y, z:-80 + Math.random()*160,
      vx:-(25 + Math.random()*50)*(burst ? 2.2 : 1), vy:-(18 + Math.random()*22), vz:(Math.random()*60 + 20),
      rx:Math.random()*6, ry:Math.random()*6, rz:Math.random()*6,
      wx:(Math.random()-0.5)*4, wy:(Math.random()-0.5)*4, wz:(Math.random()-0.5)*3,
      age:0, life:9 + Math.random()*6, sc:0.8 + Math.random()*0.5});
  }

  const tmpM = () => new THREE.Matrix4();
  let last = 0, spawnAcc = 0;
  function update(tms){
    if(!st.ready) return;
    const t = tms/1000, dt = Math.min(0.05, t - last || 0.016); last = t;
    // 风：常风 + 鼠标在窗前时起风 + 点窗时一阵大风
    const hover = overWindow(st.mouse.x, st.mouse.y);
    st.gust += ((hover ? 1 : 0) - st.gust)*(hover ? 0.02 : 0.01);
    st.burst *= 0.97;
    st.wind = 0.25 + st.gust*0.75 + st.burst;
    // 视差：鼠标偏离中心，相机平移；视窗反向偏移让 z=0 平面不动
    const tx = st.mouse.x > -9000 ? (st.mouse.x/st.W - 0.5) : 0, ty = st.mouse.y > -9000 ? (st.mouse.y/st.H - 0.5) : 0;
    st.par.x += (tx*70 - st.par.x)*0.05; st.par.y += (-ty*40 - st.par.y)*0.05;
    for(const c of [camB, camA]){
      c.position.set(st.par.x, st.par.y, D);
      c.setViewOffset(st.W, st.H, -st.par.x, st.par.y, st.W, st.H);
    }
    gardenMat.uniforms.uT.value = t; gardenMat.uniforms.uWind.value = st.wind;
    roomMat.uniforms.uT.value = t; roomMat.uniforms.uWind.value = st.wind;
    beams.forEach((m, i) => { m.material.uniforms.uT.value = t + i*3.1; m.material.uniforms.uI.value = 0.028 + 0.012*Math.sin(t*0.13 + i); });
    // 浮尘
    const g = dust.geometry, pos = g.attributes.position.array, vel = dust.userData.vel, N = dust.userData.N;
    const box = {x0:imgToWorld(0.12, 0).x, x1:imgToWorld(0.92, 0).x, y0:imgToWorld(0, 0.86).y, y1:imgToWorld(0, 0.02).y};
    if(!dust.userData.init){
      for(let i = 0; i < N; i++){ pos[i*3] = box.x0 + Math.random()*(box.x1 - box.x0); pos[i*3+1] = box.y0 + Math.random()*(box.y1 - box.y0); pos[i*3+2] = -120 + Math.random()*340; }
      dust.userData.init = true;
    }
    const mx = st.mouse.x - st.W/2, my = st.H/2 - st.mouse.y;
    for(let i = 0; i < N; i++){
      const k = i*3;
      vel[k]   += (Math.sin(t*0.3 + i)*0.6 - vel[k]*0.02 - st.wind*0.4)*dt*6;
      vel[k+1] += (Math.cos(t*0.27 + i*1.7)*0.5 - vel[k+1]*0.02 + 0.05)*dt*6;
      const dx = pos[k] - mx, dy = pos[k+1] - my, d2 = dx*dx + dy*dy;
      if(d2 < 140*140){ const d = Math.sqrt(d2) + 1, f = (1 - d/140)*90; vel[k] += (-dy/d*f + dx/d*f*0.4)*dt; vel[k+1] += (dx/d*f + dy/d*f*0.4)*dt; }
      pos[k] += vel[k]*dt*8; pos[k+1] += vel[k+1]*dt*8;
      if(pos[k] < box.x0) pos[k] = box.x1; if(pos[k] > box.x1) pos[k] = box.x0;
      if(pos[k+1] < box.y0) pos[k+1] = box.y1; if(pos[k+1] > box.y1) pos[k+1] = box.y0;
    }
    g.attributes.position.needsUpdate = true;
    dust.material.uniforms.uT.value = t;
    // 落叶
    spawnAcc += dt*(0.08 + st.wind*0.7 + st.burst*6);
    while(spawnAcc > 1){ spawnAcc -= 1; spawnLeaf(st.burst > 0.3); }
    const floorY = imgToWorld(0, 0.9).y;
    const M = new THREE.Matrix4(), Q = new THREE.Quaternion(), E = new THREE.Euler(), V = new THREE.Vector3(), SC = new THREE.Vector3();
    for(const mesh of leaves){
      const {k, P} = mesh.userData;
      P.forEach((p, i) => {
        if(!p.alive){ M.makeScale(0, 0, 0); mesh.setMatrixAt(i, M); return; }
        p.age += dt;
        const flutter = Math.sin(p.age*3 + i)*30;
        p.x += (p.vx - st.wind*40 + flutter*0.6)*dt; p.y += (p.vy + Math.cos(p.age*2.2 + i)*14)*dt; p.z += p.vz*dt*0.3;
        p.rx += p.wx*dt*(1 + st.wind); p.ry += p.wy*dt*(1 + st.wind); p.rz += p.wz*dt;
        let s = p.sc*st.img.s*2.1;
        if(p.y < floorY){ p.vy = 0; p.vx *= 0.9; p.wx *= 0.9; p.wy *= 0.9; p.age = Math.max(p.age, p.life - 2.5); }
        if(p.age > p.life - 2) s *= Math.max(0, (p.life - p.age)/2);
        if(p.age > p.life || p.x < -st.W*0.7){ p.alive = false; }
        E.set(p.rx, p.ry, p.rz); Q.setFromEuler(E); V.set(p.x, p.y, p.z); SC.set(k.w*s, k.h*s, 1);
        M.compose(V, Q, SC); mesh.setMatrixAt(i, M);
      });
      mesh.instanceMatrix.needsUpdate = true;
    }
    back.render(st.sceneB, camB);
    air.render(st.sceneA, camA);
  }

  return {
    init, resize, update, overWindow,
    setMouse(x, y){ st.mouse.x = x; st.mouse.y = y; },
    gustBurst(){ st.burst = 1.2; },
    get ready(){ return st.ready; }, get wind(){ return st.wind; },
  };
})();
