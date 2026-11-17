'use strict';
// ════════════════════════════════════════════════════════════════
//  渲染：颜色层 + 丝向/高度层（2D）→ WebGL 合成绢面、丝光、影子
//  绷面坐标（圆心 0,0，半径 500，y 向下）→ 屏幕：view.cx + x*view.k
// ════════════════════════════════════════════════════════════════
const DPR = Math.min(window.devicePixelRatio || 1, 1.5);
const HOOP_R = 500;

const Render = (() => {
  const glc = document.getElementById('gl');
  const colorCv = document.createElement('canvas'), dirCv = document.createElement('canvas');
  const cctx = colorCv.getContext('2d'), dctx = dirCv.getContext('2d');
  const view = {cx:0, cy:0, k:1, rpx:0, W:0, H:0};
  let dirty = true;

  function resize(W, H, hoop){
    view.W = W; view.H = H;
    view.cx = hoop.cx; view.cy = hoop.cy; view.rpx = hoop.r; view.k = hoop.r/HOOP_R;
    for(const cv of [glc, colorCv, dirCv]){ cv.width = Math.round(W*DPR); cv.height = Math.round(H*DPR); }
    gl.viewport(0, 0, glc.width, glc.height);
    clear();
  }
  function clear(){
    cctx.setTransform(1,0,0,1,0,0); dctx.setTransform(1,0,0,1,0,0);
    cctx.clearRect(0, 0, colorCv.width, colorCv.height);
    dctx.clearRect(0, 0, dirCv.width, dirCv.height);
    dirty = true;
  }
  const toScreen = (x, y) => ({x:view.cx + x*view.k, y:view.cy + y*view.k});
  const toHoop = (sx, sy) => ({x:(sx - view.cx)/view.k, y:(sy - view.cy)/view.k});

  // 画一批针脚
  function drawStitches(list, from, to){
    const k = view.k*DPR;
    const m = [k, 0, 0, k, view.cx*DPR, view.cy*DPR];
    cctx.setTransform(...m); dctx.setTransform(...m);
    cctx.lineCap = dctx.lineCap = 'round';
    for(let i = from; i < to; i++){
      const s = list[i];
      cctx.strokeStyle = `rgb(${s.c[0]|0},${s.c[1]|0},${s.c[2]|0})`;
      cctx.lineWidth = s.w;
      cctx.beginPath(); cctx.moveTo(s.x1, s.y1); cctx.lineTo(s.x2, s.y2); cctx.stroke();
      // 丝向：屏幕 y 向下，着色器里 y 向上，所以角度取反
      const th = Math.atan2(-(s.y2 - s.y1), s.x2 - s.x1);
      const R = (Math.cos(2*th)*0.5 + 0.5)*255 | 0, G = (Math.sin(2*th)*0.5 + 0.5)*255 | 0;
      const prof = peak => {
        const g = dctx.createLinearGradient(s.x1, s.y1, s.x2, s.y2);
        const p = peak*s.h;
        g.addColorStop(0, `rgb(${R},${G},${p*0.4|0})`);
        g.addColorStop(0.16, `rgb(${R},${G},${p*0.78|0})`);
        g.addColorStop(0.5, `rgb(${R},${G},${p|0})`);
        g.addColorStop(0.84, `rgb(${R},${G},${p*0.78|0})`);
        g.addColorStop(1, `rgb(${R},${G},${p*0.4|0})`);
        return g;
      };
      dctx.lineWidth = s.w; dctx.strokeStyle = prof(115);
      dctx.beginPath(); dctx.moveTo(s.x1, s.y1); dctx.lineTo(s.x2, s.y2); dctx.stroke();
      dctx.lineWidth = s.w*0.46; dctx.strokeStyle = prof(255);
      dctx.beginPath(); dctx.moveTo(s.x1, s.y1); dctx.lineTo(s.x2, s.y2); dctx.stroke();
    }
    dirty = true;
  }

  // ── WebGL ──
  const gl = glc.getContext('webgl', {premultipliedAlpha:false, antialias:false, alpha:true});
  const VS = `attribute vec2 p;void main(){gl_Position=vec4(p,0.,1.);}`;
  const FS = `precision highp float;
uniform sampler2D uC,uD;uniform vec2 uRes;uniform vec2 uL;uniform vec3 uFab;uniform float uDpr;
uniform vec3 uHoop; // 圆心 x,y（设备像素，y 向上）和半径
uniform float uT,uWind;
float h1(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
float n2(vec2 p){vec2 i=floor(p),f=fract(p);f=f*f*(3.-2.*f);return mix(mix(h1(i),h1(i+vec2(1,0)),f.x),mix(h1(i+vec2(0,1)),h1(i+vec2(1,1)),f.x),f.y);}
float H(vec2 uv){return texture2D(uD,uv).b*texture2D(uC,uv).a;}
float fbm(vec2 p){float s=0.,a=.5;for(int i=0;i<4;i++){s+=a*n2(p);p=p*2.03+17.1;a*=.5;}return s;}
void main(){
  vec2 uv=gl_FragCoord.xy/uRes;
  vec2 q=(gl_FragCoord.xy-uHoop.xy)/uHoop.z;          // 绷面内归一化坐标
  float rr=length(q);
  float mask=clamp((1.0-rr)*uHoop.z/1.2,0.,1.);
  if(mask<=0.){gl_FragColor=vec4(0.);return;}
  vec2 p=gl_FragCoord.xy/uDpr;
  vec4 c=texture2D(uC,uv);vec4 d=texture2D(uD,uv);
  // 绢：平纹组织，经纬一上一下交织；每根丝粗细明暗略有不同，偶有粗节；
  // 交织处的小孔略暗；侧光时顺光的那组丝泛光（光从左右来亮经线，从上下来亮纬线）
  vec2 g=gl_FragCoord.xy/2.5;                       // 一根丝约 2.5 个设备像素
  g+=vec2(n2(vec2(gl_FragCoord.y*0.035,floor(g.x)*0.37))-0.5,n2(vec2(gl_FragCoord.x*0.03,floor(g.y)*0.41))-0.5)*0.55;   // 丝不是笔直的
  vec2 gi=floor(g),gf=fract(g);
  float warpTop=mod(gi.x+gi.y,2.0);                // 1 = 经线压在上面
  float wv=h1(vec2(gi.x,7.13)),fv=h1(vec2(3.37,gi.y));
  float slubRow=step(0.93,h1(vec2(gi.y,1.71)))*smoothstep(0.35,0.75,n2(vec2(gl_FragCoord.x*0.011+gi.y*3.1,gi.y)));
  float slubCol=step(0.96,h1(vec2(gi.x,5.9)))*smoothstep(0.4,0.8,n2(vec2(gi.x,gl_FragCoord.y*0.01+gi.x*1.7)));
  float prof=mix(sin(3.14159*gf.y),sin(3.14159*gf.x),warpTop);
  float tv=mix(fv+slubRow*0.9,wv+slubCol*0.9,warpTop);
  float hole=(1.-smoothstep(0.,0.2,min(gf.x,1.-gf.x)))*(1.-smoothstep(0.,0.2,min(gf.y,1.-gf.y)));
  float sheen=mix(abs(uL.y),abs(uL.x),warpTop)*prof*prof;
  float weave=0.952+0.03*prof+0.062*(tv-0.5)-0.025*hole+0.055*sheen;
  float mott=fbm(p*0.011)-0.5;                     // 大片的轻微色差（染色不匀）
  vec3 fab=uFab*(weave+0.03*mott)+vec3(1.,.97,.9)*0.025*sheen*(0.6+0.4*tv);
  float phiF=atan(uL.y,uL.x);
  fab*=1.0+0.07*dot(uL,q)-0.05*dot(q,q);
  fab+=uFab*0.04*(0.5+0.5*cos(2.*phiF));
  // 绷圈压着绢的边：一圈暗影，背光那边更深
  float rim=smoothstep(0.86,1.0,rr);
  fab*=1.0-rim*(0.22+0.18*clamp(-dot(normalize(q+1e-5),uL),0.,1.));
  // 窗外竹叶的影子在绢面上晃（和房间里的光斑同一套）
  float sway=uT*(0.12+uWind*0.35);
  float dap=smoothstep(0.52,0.72,fbm(q*2.2+vec2(sin(sway)*0.6,cos(sway*0.8)*0.4)+vec2(uT*0.02,0.)));
  float leafLight=1.0+(dap*0.10-0.03)*smoothstep(-1.2,1.0,q.x*0.6+q.y*0.5);
  fab*=leafLight;
  // 丝线落在绢上的影子
  vec2 o=uL*uDpr/uRes;
  float sh=texture2D(uC,uv+o*1.6).a*.5+texture2D(uC,uv+o*3.2).a*.3+texture2D(uC,uv+o*5.0+vec2(o.y,-o.x)).a*.2;
  fab*=1.0-0.30*sh;
  vec3 col=fab;
  if(c.a>0.003){
    vec2 cs=d.rg*2.-1.;
    float th=0.5*atan(cs.y,cs.x);
    vec2 e=1.0/uRes;
    float h=d.b*c.a;
    float hx=H(uv+vec2(e.x,0.))-H(uv-vec2(e.x,0.));
    float hy=H(uv+vec2(0.,e.y))-H(uv-vec2(0.,e.y));
    vec3 N=normalize(vec3(-hx*1.1,-hy*1.1,1.0));
    vec3 L=normalize(vec3(uL*0.8,0.6));
    float dif=clamp(dot(N,L),0.,1.);
    vec3 T=vec3(cos(th),sin(th),0.);
    T=normalize(T-N*dot(T,N));
    vec3 Hv=normalize(L+vec3(0.,0.,1.));
    float tH=dot(T,Hv);
    float sk=sqrt(max(0.,1.-tH*tH));
    float spec=pow(sk,80.)*0.8+pow(sk,14.)*0.22;
    float ao=0.7+0.3*smoothstep(0.,0.7,h);
    vec3 t=c.rgb*(0.40+0.72*dif)*ao
          +mix(c.rgb,vec3(1.,.97,.92),0.5)*spec*1.1*smoothstep(0.1,0.7,h);
    t*=1.0-rim*0.25;
    t*=leafLight;
    col=mix(fab,t,c.a);
  }
  gl_FragColor=vec4(col,mask);
}`;
  function sh(type, src){ const s = gl.createShader(type); gl.shaderSource(s, src); gl.compileShader(s);
    if(!gl.getShaderParameter(s, gl.COMPILE_STATUS)) console.error(gl.getShaderInfoLog(s)); return s; }
  const prog = gl.createProgram();
  gl.attachShader(prog, sh(gl.VERTEX_SHADER, VS)); gl.attachShader(prog, sh(gl.FRAGMENT_SHADER, FS));
  gl.linkProgram(prog); gl.useProgram(prog);
  const buf = gl.createBuffer(); gl.bindBuffer(gl.ARRAY_BUFFER, buf);
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1,-1,3,-1,-1,3]), gl.STATIC_DRAW);
  const loc = gl.getAttribLocation(prog, 'p'); gl.enableVertexAttribArray(loc); gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
  function mkTex(unit){ const t = gl.createTexture(); gl.activeTexture(gl.TEXTURE0 + unit); gl.bindTexture(gl.TEXTURE_2D, t);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR); gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE); gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE); return t; }
  const texC = mkTex(0), texD = mkTex(1);
  gl.uniform1i(gl.getUniformLocation(prog, 'uC'), 0); gl.uniform1i(gl.getUniformLocation(prog, 'uD'), 1);
  const U = n => gl.getUniformLocation(prog, n);
  const uRes = U('uRes'), uL = U('uL'), uFab = U('uFab'), uDpr = U('uDpr'), uHoop = U('uHoop'), uT = U('uT'), uWind = U('uWind');
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);

  function frame(lightA, fabric, t = 0, wind = 0.25){
    if(!colorCv.width || !colorCv.height) return;
    if(dirty){
      gl.activeTexture(gl.TEXTURE0); gl.bindTexture(gl.TEXTURE_2D, texC);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, colorCv);
      gl.activeTexture(gl.TEXTURE1); gl.bindTexture(gl.TEXTURE_2D, texD);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, dirCv);
      dirty = false;
    }
    gl.clearColor(0, 0, 0, 0); gl.clear(gl.COLOR_BUFFER_BIT);
    gl.uniform2f(uRes, glc.width, glc.height);
    gl.uniform2f(uL, Math.cos(lightA), Math.sin(lightA));
    gl.uniform3f(uFab, fabric[0]/255, fabric[1]/255, fabric[2]/255);
    gl.uniform1f(uDpr, DPR);
    gl.uniform1f(uT, t); gl.uniform1f(uWind, wind);
    gl.uniform3f(uHoop, view.cx*DPR, glc.height - view.cy*DPR, view.rpx*DPR);
    gl.drawArrays(gl.TRIANGLES, 0, 3);
  }
  return {resize, clear, drawStitches, frame, toScreen, toHoop, view};
})();
