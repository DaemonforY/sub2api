(function () {
'use strict';

/* ------------------------------------------------------------------ *
 *  水墨流体 — GPU Navier–Stokes ink on procedural 宣纸 (rice paper)   *
 * ------------------------------------------------------------------ */

var $ = function (s) { return document.querySelector(s); };
var canvas = $('#c');
var FONT = '"Kaiti SC","STKaiti","KaiTi","KaiTi_GB2312","BiauKai","PingFang SC","Hiragino Sans GB","Noto Serif SC","Noto Sans CJK SC","Microsoft YaHei",serif';
var reduceMotion = false, coarse = false;
try { reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches; coarse = matchMedia('(pointer: coarse)').matches; } catch (e) {}

var LEVELS = [
  { ch: '一', title: '一画开天', type: 'char', tip: '沿着描红，从左往右拖过去', need: 0.78, spill: 0.22, cur: 0, perm: 1.3, dep: 1.15 },
  { ch: '月', title: '满月', type: 'moon', tip: '按住不动，墨会慢慢晕开', need: 0.78, spill: 0.28, cur: 0, perm: 1.8, dep: 0.95 },
  { ch: '山', title: '山', type: 'char', tip: '一笔一笔描，墨会自己流', need: 0.70, spill: 0.32, cur: 0, perm: 1.4, dep: 1.05 },
  { ch: '峦', title: '远山', type: 'hills', tip: '水在流动，顺着它把墨铺开', need: 0.72, spill: 0.32, cur: 7, perm: 1.7, dep: 0.95 },
  { ch: '水', title: '水', type: 'char', tip: '暗流涌动，先稳住再下笔', need: 0.70, spill: 0.36, cur: 6, perm: 1.5, dep: 1.0 },
  { ch: '永', title: '永字八法', type: 'char', tip: '生宣吃墨快，晕得也快', need: 0.70, spill: 0.4, cur: 0, perm: 2.3, dep: 1.0 },
  { ch: '福', title: '福', type: 'char', tip: '笔画多，耐心描', need: 0.70, spill: 0.4, cur: 5, perm: 1.7, dep: 1.0 },
  { ch: '龙', title: '龙', type: 'char', tip: '最后一关：暗流加生宣', need: 0.70, spill: 0.45, cur: 9, perm: 2.2, dep: 0.92 }
];
var NUM = ['一', '二', '三', '四', '五', '六', '七', '八', '九', '十'];

/* ---------------- shaders ---------------- */
var VS = [
  'precision highp float;',
  'attribute vec2 aPos;',
  'varying vec2 vUv; varying vec2 vL; varying vec2 vR; varying vec2 vT; varying vec2 vB;',
  'uniform vec2 texelSize;',
  'void main(){ vUv=aPos*0.5+0.5; vL=vUv-vec2(texelSize.x,0.0); vR=vUv+vec2(texelSize.x,0.0);',
  ' vT=vUv+vec2(0.0,texelSize.y); vB=vUv-vec2(0.0,texelSize.y); gl_Position=vec4(aPos,0.0,1.0); }'
].join('\n');

var FS_HEAD = [
  '#ifdef GL_FRAGMENT_PRECISION_HIGH',
  'precision highp float; precision highp sampler2D;',
  '#else',
  'precision mediump float; precision mediump sampler2D;',
  '#endif',
  'varying vec2 vUv; varying vec2 vL; varying vec2 vR; varying vec2 vT; varying vec2 vB;',
  'uniform vec2 texelSize;',
  'vec4 samp(sampler2D t, vec2 uv, vec2 ts){',
  '#ifdef MANUAL_FILTERING',
  ' vec2 st=uv/ts-0.5; vec2 i=floor(st); vec2 f=fract(st);',
  ' vec4 a=texture2D(t,(i+vec2(0.5,0.5))*ts); vec4 b=texture2D(t,(i+vec2(1.5,0.5))*ts);',
  ' vec4 c=texture2D(t,(i+vec2(0.5,1.5))*ts); vec4 d=texture2D(t,(i+vec2(1.5,1.5))*ts);',
  ' return mix(mix(a,b,f.x),mix(c,d,f.x),f.y);',
  '#else',
  ' return texture2D(t,uv);',
  '#endif',
  '}',
  'float hash12(vec2 p){ vec3 p3=fract(vec3(p.xyx)*0.1031); p3+=dot(p3,p3.yzx+33.33); return fract((p3.x+p3.y)*p3.z); }',
  ''
].join('\n');

var SRC = {};
SRC.copy = 'uniform sampler2D uTex; void main(){ gl_FragColor=texture2D(uTex,vUv); }';
SRC.scale = 'uniform sampler2D uTex; uniform vec4 s; void main(){ gl_FragColor=texture2D(uTex,vUv)*s; }';
SRC.seg = 'float segD(vec2 p, vec2 a, vec2 b){ vec2 pa=p-a, ba=b-a; float h=clamp(dot(pa,ba)/max(dot(ba,ba),1e-9),0.0,1.0); return length(pa-ba*h); }\n';
SRC.splatVel = SRC.seg + [
  'uniform sampler2D uTarget; uniform float aspect; uniform vec2 pA; uniform vec2 pB; uniform float radius; uniform vec3 value;',
  'void main(){ vec2 p=vec2(vUv.x*aspect,vUv.y); vec2 a=vec2(pA.x*aspect,pA.y); vec2 b=vec2(pB.x*aspect,pB.y);',
  ' float d=segD(p,a,b); float g=exp(-d*d/(radius*radius));',
  ' vec4 c=texture2D(uTarget,vUv); c.xy=mix(c.xy,value.xy,clamp(g*value.z,0.0,1.0)); gl_FragColor=c; }'
].join('\n');
SRC.splatInk = SRC.seg + [
  'uniform sampler2D uTarget; uniform sampler2D uPaper; uniform float aspect; uniform vec2 pA; uniform vec2 pB; uniform float radius; uniform float wr; uniform vec3 ink;',
  'void main(){ vec2 p=vec2(vUv.x*aspect,vUv.y); vec2 a=vec2(pA.x*aspect,pA.y); vec2 b=vec2(pB.x*aspect,pB.y);',
  ' float d=segD(p,a,b); float q=d*d/(radius*radius); float g=exp(-q*q);',
  ' float pa=texture2D(uPaper,vUv).a; float gg=g*(0.7+0.6*pa);',
  ' vec4 c=texture2D(uTarget,vUv);',
  ' c.r+=ink.x*gg*max(0.0,1.0-c.r/4.0); c.g+=ink.y*gg*max(0.0,1.0-c.g/4.0);',
  ' float gw=exp(-d*d/(radius*radius*wr*wr))*(0.3+1.1*pa); c.b=max(c.b,min(c.b+ink.z*gw,1.6)); gl_FragColor=c; }'
].join('\n');
SRC.texInk = [
  'uniform sampler2D uTarget; uniform sampler2D uMask; uniform sampler2D uPaper; uniform vec4 w; uniform float water; uniform float red;',
  'void main(){ float m=dot(texture2D(uMask,vUv),w); float pa=texture2D(uPaper,vUv).a; float mm=m*(0.75+0.5*pa);',
  ' vec4 c=texture2D(uTarget,vUv); c.r+=mm*(1.0-red); c.g+=mm*red; c.b=max(c.b,min(c.b+m*water,1.6)); gl_FragColor=c; }'
].join('\n');
SRC.stamp = [
  'uniform sampler2D uTarget; uniform sampler2D uSeal; uniform vec4 rect; uniform float amt;',
  'void main(){ vec4 c=texture2D(uTarget,vUv); vec2 q=(vUv-rect.xy)/rect.zw; float a=0.0;',
  ' if(q.x>0.0&&q.y>0.0&&q.x<1.0&&q.y<1.0) a=texture2D(uSeal,q).a;',
  ' c.g+=a*amt; c.r*=1.0-0.5*a; gl_FragColor=c; }'
].join('\n');
SRC.curl = [
  'uniform sampler2D uVelocity;',
  'void main(){ float L=texture2D(uVelocity,vL).y; float R=texture2D(uVelocity,vR).y; float T=texture2D(uVelocity,vT).x; float B=texture2D(uVelocity,vB).x;',
  ' gl_FragColor=vec4(0.5*(R-L-T+B),0.0,0.0,1.0); }'
].join('\n');
SRC.vort = [
  'uniform sampler2D uVelocity; uniform sampler2D uCurl; uniform float curl; uniform float dt;',
  'void main(){ float L=texture2D(uCurl,vL).x; float R=texture2D(uCurl,vR).x; float T=texture2D(uCurl,vT).x; float B=texture2D(uCurl,vB).x; float C=texture2D(uCurl,vUv).x;',
  ' vec2 f=0.5*vec2(abs(T)-abs(B),abs(R)-abs(L)); f/=length(f)+0.0001; f*=curl*C; f.y*=-1.0;',
  ' vec2 v=texture2D(uVelocity,vUv).xy+f*dt; gl_FragColor=vec4(clamp(v,-1500.0,1500.0),0.0,1.0); }'
].join('\n');
SRC.div = [
  'uniform sampler2D uVelocity;',
  'void main(){ float L=texture2D(uVelocity,vL).x; float R=texture2D(uVelocity,vR).x; float T=texture2D(uVelocity,vT).y; float B=texture2D(uVelocity,vB).y;',
  ' vec2 C=texture2D(uVelocity,vUv).xy; if(vL.x<0.0)L=-C.x; if(vR.x>1.0)R=-C.x; if(vT.y>1.0)T=-C.y; if(vB.y<0.0)B=-C.y;',
  ' gl_FragColor=vec4(0.5*(R-L+T-B),0.0,0.0,1.0); }'
].join('\n');
SRC.press = [
  'uniform sampler2D uPressure; uniform sampler2D uDivergence;',
  'void main(){ float L=texture2D(uPressure,vL).x; float R=texture2D(uPressure,vR).x; float T=texture2D(uPressure,vT).x; float B=texture2D(uPressure,vB).x;',
  ' float d=texture2D(uDivergence,vUv).x; gl_FragColor=vec4((L+R+B+T-d)*0.25,0.0,0.0,1.0); }'
].join('\n');
SRC.grad = [
  'uniform sampler2D uPressure; uniform sampler2D uVelocity;',
  'void main(){ float L=texture2D(uPressure,vL).x; float R=texture2D(uPressure,vR).x; float T=texture2D(uPressure,vT).x; float B=texture2D(uPressure,vB).x;',
  ' vec2 v=texture2D(uVelocity,vUv).xy-vec2(R-L,T-B); gl_FragColor=vec4(v,0.0,1.0); }'
].join('\n');
SRC.advect = [
  'uniform sampler2D uVelocity; uniform sampler2D uSource; uniform vec2 dyeTexel; uniform float dt; uniform vec4 diss;',
  'void main(){ vec2 coord=vUv-dt*samp(uVelocity,vUv,texelSize).xy*texelSize;',
  ' gl_FragColor=samp(uSource,coord,dyeTexel)/(1.0+diss*dt); }'
].join('\n');
SRC.current = [
  'uniform sampler2D uVelocity; uniform float amp; uniform float t; uniform float aspect; uniform float dt;',
  'void main(){ vec2 p=vec2(vUv.x*aspect,vUv.y)*3.2; float a=p.x+0.21*t; float b=0.8*p.y-0.13*t; float c=1.7*p.y+0.6*p.x+0.09*t;',
  ' vec2 f=vec2(-0.8*sin(a)*sin(b)+0.85*cos(c), -cos(a)*cos(b)-0.3*cos(c));',
  ' vec2 v=texture2D(uVelocity,vUv).xy+f*amp*dt; gl_FragColor=vec4(v,0.0,1.0); }'
].join('\n');
/* shared: wet-edge + absorption so the deposit pass and bleed pass agree exactly */
SRC.wetCommon = [
  'uniform sampler2D uWet; uniform sampler2D uPaper; uniform sampler2D uVelocity;',
  'uniform float dt; uniform float spread; uniform float depRate;',
  'float wAt(vec2 o){ return texture2D(uWet,vUv+o*texelSize*spread).b; }',
  'float avgW(){ return (wAt(vec2(1.0,0.0))+wAt(vec2(-1.0,0.0))+wAt(vec2(0.0,1.0))+wAt(vec2(0.0,-1.0))',
  '  +0.5*(wAt(vec2(1.0,1.0))+wAt(vec2(-1.0,1.0))+wAt(vec2(1.0,-1.0))+wAt(vec2(-1.0,-1.0))))/6.0; }',
  'float absorbAmt(vec4 c, float edge, vec4 pp, float speed){',
  ' float pa=0.65*pp.r+0.35*pp.a;',
  ' float dry=1.0-smoothstep(0.02,0.26,c.b);',
  ' float r=depRate*(0.6+0.8*pa)*(1.0+3.0*edge)/(1.0+speed*0.035)+2.6*dry;',
  ' return clamp(r*dt,0.0,0.85); }',
  ''
].join('\n');
SRC.deposit = SRC.wetCommon + [
  'uniform sampler2D uDep;',
  'void main(){ vec4 c=texture2D(uWet,vUv); float avg=avgW(); float edge=clamp((c.b-avg)/(c.b*0.6+0.03)*1.6,0.0,1.0)*smoothstep(0.008,0.05,c.b);',
  ' vec4 pp=texture2D(uPaper,vUv); float sp=length(texture2D(uVelocity,vUv).xy);',
  ' float a=absorbAmt(c,edge,pp,sp); vec4 d=texture2D(uDep,vUv);',
  ' d.rg+=c.rg*a; d.b=min(d.b+edge*c.b*dt*0.9,1.0); gl_FragColor=d; }'
].join('\n');
SRC.bleed = SRC.wetCommon + [
  'uniform float perm; uniform float evap; uniform float pigMob;',
  'float permOf(float a){ return 0.1+1.65*a*a; }',
  'void nb(vec2 o, float wt, vec4 c, float pc, float k0, inout vec3 flow){',
  ' vec2 uv=vUv+o*texelSize*spread; vec4 n=texture2D(uWet,uv); float pn=permOf(texture2D(uPaper,uv).a);',
  ' float k=k0*wt*sqrt(pc*pn)*min(n.b+c.b,1.6);',
  ' float f=k*(n.b-c.b);',
  ' vec2 conc=f>0.0 ? n.rg/max(n.b,0.05) : c.rg/max(c.b,0.05);',
  ' flow.rg+=f*conc*pigMob; flow.b+=f; }',
  'void main(){ vec4 c=texture2D(uWet,vUv); vec4 pp=texture2D(uPaper,vUv); float pc=permOf(pp.a);',
  ' float k0=min(perm*dt*1.4,0.05); vec3 flow=vec3(0.0);',
  ' nb(vec2(1.0,0.0),1.0,c,pc,k0,flow); nb(vec2(-1.0,0.0),1.0,c,pc,k0,flow); nb(vec2(0.0,1.0),1.0,c,pc,k0,flow); nb(vec2(0.0,-1.0),1.0,c,pc,k0,flow);',
  ' nb(vec2(1.0,1.0),0.5,c,pc,k0,flow); nb(vec2(-1.0,1.0),0.5,c,pc,k0,flow); nb(vec2(1.0,-1.0),0.5,c,pc,k0,flow); nb(vec2(-1.0,-1.0),0.5,c,pc,k0,flow);',
  ' float avg=avgW(); float edge=clamp((c.b-avg)/(c.b*0.6+0.03)*1.6,0.0,1.0)*smoothstep(0.008,0.05,c.b); float sp=length(texture2D(uVelocity,vUv).xy);',
  ' float a=absorbAmt(c,edge,pp,sp);',
  ' vec4 o; o.rg=max(c.rg+flow.rg-c.rg*a,0.0); o.b=max(c.b+flow.b-dt*(evap+0.11*c.b),0.0); o.a=1.0; gl_FragColor=o; }'
].join('\n');
SRC.display = [
  'uniform sampler2D uWet; uniform sampler2D uDep; uniform sampler2D uPaper; uniform sampler2D uMask;',
  'uniform vec2 dyeTexel; uniform vec2 res; uniform float dpr; uniform vec4 rect;',
  'void main(){',
  ' vec4 pp=texture2D(uPaper,vUv);',
  ' vec4 w=samp(uWet,vUv,dyeTexel); vec4 d=samp(uDep,vUv,dyeTexel);',
  ' vec2 o=dyeTexel*1.6;',
  ' float dav=0.25*(samp(uDep,vUv+vec2(o.x,0.0),dyeTexel).r+samp(uDep,vUv-vec2(o.x,0.0),dyeTexel).r+samp(uDep,vUv+vec2(0.0,o.y),dyeTexel).r+samp(uDep,vUv-vec2(0.0,o.y),dyeTexel).r);',
  ' float rim=max(d.r-dav,0.0);',
  /* paper */
  ' vec3 col=vec3(0.946,0.914,0.846)*(1.0+(pp.r-0.5)*0.10+pp.g*0.028+(pp.b-0.5)*0.035);',
  ' col.b-=(pp.r-0.5)*0.025;',
  ' float wetp=smoothstep(0.04,1.1,w.b); col*=1.0-0.075*wetp; col.r-=0.01*wetp;',
  ' col*=1.0-0.06*d.b;',
  /* guide: 米字格 + 描红 outline */
  ' if(rect.w>0.001){',
  '  vec2 p=vUv*res; vec2 q=p-rect.xy; float h=rect.z; float lw=dpr*0.75;',
  '  float inf=max(abs(q.x),abs(q.y)); float inside=step(inf,h);',
  '  float box=1.0-smoothstep(lw*0.6,lw*1.8,abs(inf-h));',
  '  float dash=step(0.45,fract(length(q)/(dpr*9.0)));',
  '  float crs=max(1.0-smoothstep(lw*0.3,lw*1.3,abs(q.x)),1.0-smoothstep(lw*0.3,lw*1.3,abs(q.y)));',
  '  float dg=1.0-smoothstep(lw*0.3,lw*1.3,abs(abs(q.x)-abs(q.y))*0.7071);',
  '  float grid=max(box,inside*dash*max(crs,dg)*0.75);',
  '  vec2 mo=vec2(1.7*dpr)/res; float m=texture2D(uMask,vUv).a;',
  '  float gx=texture2D(uMask,vUv+vec2(mo.x,0.0)).a-texture2D(uMask,vUv-vec2(mo.x,0.0)).a;',
  '  float gy=texture2D(uMask,vUv+vec2(0.0,mo.y)).a-texture2D(uMask,vUv-vec2(0.0,mo.y)).a;',
  '  float edge=smoothstep(0.12,0.55,length(vec2(gx,gy)));',
  '  float ga=rect.w*clamp(grid*0.30+edge*0.62+m*0.10,0.0,0.85);',
  '  col=mix(col,vec3(0.80,0.33,0.24),ga);',
  ' }',
  /* black ink */
  ' float grain=pp.b; float fib=pp.g;',
  ' float bd=d.r+w.r*0.92+rim*1.1;',
  ' float tex=0.80+0.40*grain+0.25*fib; float dens=bd*mix(tex,1.0,smoothstep(0.5,1.5,bd));',
  ' float a=1.0-exp(-dens*1.75);',
  ' vec3 thin=vec3(0.50,0.505,0.52); vec3 thick=vec3(0.052,0.048,0.054);',
  ' vec3 inkc=mix(thin,thick,smoothstep(0.2,0.97,a));',
  ' col*=mix(vec3(1.0),inkc,a);',
  /* wet ink: a touch cooler and deeper */
  ' float wa=1.0-exp(-w.r*1.5); col*=1.0-0.10*wa*wetp; col.b+=0.012*wa*wetp;',
  /* cinnabar */
  ' float rd=(d.g+w.g*0.9)*mix(0.6+0.8*grain+0.3*fib,1.0,smoothstep(0.8,2.2,d.g+w.g));',
  ' float ar=1.0-exp(-rd*1.6);',
  ' col*=mix(vec3(1.0),vec3(0.80,0.20,0.12),ar);',
  /* vignette + dither */
  ' vec2 vv=vUv-0.5; col*=1.0-0.11*pow(length(vv)*1.35,2.2);',
  ' col+=(hash12(gl_FragCoord.xy)-0.5)/255.0;',
  ' gl_FragColor=vec4(col,1.0); }'
].join('\n');
SRC.readback = [
  'uniform sampler2D uWet; uniform sampler2D uDep; uniform vec2 dyeTexel; uniform vec2 cell;',
  'float dn(vec2 uv){ vec4 w=samp(uWet,uv,dyeTexel); vec4 d=samp(uDep,uv,dyeTexel); return 1.0-exp(-(d.r+w.r*0.92)*1.75); }',
  'void main(){ vec2 c=cell*0.25; float a=0.25*(dn(vUv+vec2(c.x,c.y))+dn(vUv+vec2(-c.x,c.y))+dn(vUv+vec2(c.x,-c.y))+dn(vUv+vec2(-c.x,-c.y)));',
  ' gl_FragColor=vec4(a,0.0,0.0,1.0); }'
].join('\n');
SRC.paper = [
  'uniform vec2 res; uniform float seed; uniform float scale;',
  'float vn(vec2 p){ vec2 i=floor(p); vec2 f=fract(p); vec2 u=f*f*(3.0-2.0*f); vec2 s=vec2(seed,seed*1.37);',
  ' return mix(mix(hash12(i+s),hash12(i+vec2(1.0,0.0)+s),u.x),mix(hash12(i+vec2(0.0,1.0)+s),hash12(i+vec2(1.0,1.0)+s),u.x),u.y); }',
  'float fbm(vec2 p){ float s=0.0; float a=0.5; for(int i=0;i<5;i++){ s+=a*vn(p); p=mat2(1.6,1.2,-1.2,1.6)*p+3.1; a*=0.5; } return s/0.97; }',
  'mat2 rot(float a){ float c=cos(a); float s=sin(a); return mat2(c,-s,s,c); }',
  'float strands(vec2 p, float ang, float len, float wid, float thr){',
  ' vec2 q=rot(ang)*p; q.y+=(vn(q*0.011)-0.5)*26.0;',
  ' float n=vn(vec2(q.x/len,q.y/wid)); float m=vn(vec2(q.x/(len*2.7)+7.0,q.y/(wid*11.0)));',
  ' return smoothstep(thr,1.0,n)*smoothstep(0.38,0.72,m); }',
  'void main(){ vec2 p=vUv*res/scale;',
  ' float cloud=fbm(p/260.0); float mid=fbm(p/36.0+11.0);',
  ' float fib=strands(p,0.35,44.0,1.0,0.80)+strands(p+vec2(91.0,17.0),1.95,36.0,0.85,0.81)',
  '  +strands(p+vec2(-53.0,140.0),-0.85,58.0,1.2,0.80)+strands(p+vec2(220.0,-33.0),2.9,30.0,0.8,0.83)',
  '  +0.6*strands(p+vec2(13.0,71.0),1.2,120.0,1.6,0.86);',
  ' fib=clamp(fib,0.0,1.0);',
  ' float grain=vn(p/1.25)*0.55+vn(p/3.1+5.0)*0.45;',
  ' float speck=smoothstep(0.93,1.0,vn(p/2.0+40.0))*step(0.8,vn(p/70.0+3.0));',
  ' vec2 pw=p+(vec2(vn(p/45.0),vn(p/45.0+5.3))-0.5)*36.0; float big=fbm(p/90.0+23.0);',
  ' float ab=(vn(rot(0.35)*pw/vec2(10.0,2.6))+vn(rot(1.95)*pw/vec2(8.0,2.2)+3.0)+vn(rot(-0.85)*pw/vec2(12.0,2.8)+9.0))/3.0;',
  ' float absorb=clamp(0.1+0.5*smoothstep(0.28,0.75,ab)+0.3*(mid-0.5)+0.45*(big-0.5)+0.25*fib+0.12*(grain-0.5),0.0,1.0);',
  ' gl_FragColor=vec4(clamp(cloud-speck*0.5,0.0,1.0),fib,grain,absorb); }'
].join('\n');

/* ---------------- GL state ---------------- */
var gl = null, isGL2 = false, fmt = null, P = {}, lost = false, glOK = false;
var DPR = 1, cssW = 0, cssH = 0, W = 0, H = 0;
var QUAL = [
  { sim: 96, dye: 320, iters: 10 },
  { sim: 128, dye: 448, iters: 14 },
  { sim: 168, dye: 720, iters: 20 }
];
var quality = coarse ? 1 : 2, downgrades = 0;
var simW, simH, dyeW, dyeH;
var velocity, divF, curlF, pressure, wet, dep, paperF, maskT, titleT, rbF;
var sealTex = {};
var gw = 0, gh = 0, rbBuf = null, maskIn = null, maskDil = null, maskCount = 1;

var DBG_GL1 = /[?&]gl=1\b/.test(location.search), DBG_NOLINEAR = /[?&]nolinear\b/.test(location.search);
var useGL2 = null;
function getGL() {
  var o = { alpha: false, depth: false, stencil: false, antialias: false, preserveDrawingBuffer: false, powerPreference: 'high-performance' };
  if (useGL2 === null) {
    // probe WebGL2 + renderable float formats on a scratch canvas, so we can still fall back to WebGL1 on the real one
    useGL2 = false;
    if (!DBG_GL1) {
      try {
        var pc = document.createElement('canvas'), pg = pc.getContext('webgl2');
        if (pg) {
          var keep = gl; gl = pg; isGL2 = true;
          useGL2 = !!detectFormat();
          var lc = pg.getExtension('WEBGL_lose_context'); if (lc) lc.loseContext();
          gl = keep;
        }
      } catch (e) { useGL2 = false; }
    }
  }
  var g = null;
  if (useGL2) { try { g = canvas.getContext('webgl2', o); } catch (e) {} }
  isGL2 = !!g;
  if (!g) { try { g = canvas.getContext('webgl', o) || canvas.getContext('experimental-webgl', o); } catch (e) {} }
  return g;
}

function canRender(internal, format, type) {
  var t = gl.createTexture();
  gl.bindTexture(gl.TEXTURE_2D, t);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
  try { gl.texImage2D(gl.TEXTURE_2D, 0, internal, 4, 4, 0, format, type, null); } catch (e) { gl.deleteTexture(t); return false; }
  var f = gl.createFramebuffer();
  gl.bindFramebuffer(gl.FRAMEBUFFER, f);
  gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t, 0);
  var ok = gl.checkFramebufferStatus(gl.FRAMEBUFFER) === gl.FRAMEBUFFER_COMPLETE;
  gl.bindFramebuffer(gl.FRAMEBUFFER, null);
  gl.deleteFramebuffer(f); gl.deleteTexture(t);
  return ok;
}

function detectFormat() {
  if (isGL2) {
    gl.getExtension('EXT_color_buffer_float');
    gl.getExtension('EXT_color_buffer_half_float');
    if (canRender(gl.RGBA16F, gl.RGBA, gl.HALF_FLOAT)) return { internal: gl.RGBA16F, format: gl.RGBA, type: gl.HALF_FLOAT, linear: true };
    var fl = !!gl.getExtension('OES_texture_float_linear');
    if (canRender(gl.RGBA32F, gl.RGBA, gl.FLOAT)) return { internal: gl.RGBA32F, format: gl.RGBA, type: gl.FLOAT, linear: fl };
    return null;
  }
  var hf = gl.getExtension('OES_texture_half_float');
  gl.getExtension('EXT_color_buffer_half_float');
  if (hf) {
    var hl = !!gl.getExtension('OES_texture_half_float_linear');
    if (canRender(gl.RGBA, gl.RGBA, hf.HALF_FLOAT_OES)) return { internal: gl.RGBA, format: gl.RGBA, type: hf.HALF_FLOAT_OES, linear: hl };
  }
  var f = gl.getExtension('OES_texture_float');
  gl.getExtension('WEBGL_color_buffer_float');
  if (f) {
    var fl2 = !!gl.getExtension('OES_texture_float_linear');
    if (canRender(gl.RGBA, gl.RGBA, gl.FLOAT)) return { internal: gl.RGBA, format: gl.RGBA, type: gl.FLOAT, linear: fl2 };
  }
  return null;
}

function compile(type, src) {
  var s = gl.createShader(type);
  gl.shaderSource(s, src);
  gl.compileShader(s);
  if (!gl.getShaderParameter(s, gl.COMPILE_STATUS) && !gl.isContextLost()) throw new Error('shader: ' + gl.getShaderInfoLog(s));
  return s;
}
var vsObj = null;
function makeProgram(fsBody) {
  var defs = fmt && !fmt.linear ? '#define MANUAL_FILTERING\n' : '';
  var p = gl.createProgram();
  gl.attachShader(p, vsObj);
  gl.attachShader(p, compile(gl.FRAGMENT_SHADER, defs + FS_HEAD + fsBody));
  gl.bindAttribLocation(p, 0, 'aPos');
  gl.linkProgram(p);
  if (!gl.getProgramParameter(p, gl.LINK_STATUS) && !gl.isContextLost()) throw new Error('link: ' + gl.getProgramInfoLog(p));
  var u = {}, n = gl.getProgramParameter(p, gl.ACTIVE_UNIFORMS) || 0;
  for (var i = 0; i < n; i++) { var info = gl.getActiveUniform(p, i); u[info.name] = gl.getUniformLocation(p, info.name); }
  return { p: p, u: u };
}

function initGL() {
  gl = getGL();
  if (!gl) return false;
  fmt = detectFormat();
  if (!fmt) return false;
  if (DBG_NOLINEAR) fmt.linear = false;
  vsObj = compile(gl.VERTEX_SHADER, VS);
  for (var k in SRC) { if (k === 'seg' || k === 'wetCommon') continue; P[k] = makeProgram(SRC[k]); }
  var buf = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, buf);
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, -1, 1, 1, 1, 1, -1]), gl.STATIC_DRAW);
  var ib = gl.createBuffer();
  gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, ib);
  gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, new Uint16Array([0, 1, 2, 0, 2, 3]), gl.STATIC_DRAW);
  gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);
  gl.enableVertexAttribArray(0);
  gl.disable(gl.BLEND);
  sealTex = {};
  return true;
}

function texParams(filter) {
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
}
function createFBO(w, h, byte) {
  gl.activeTexture(gl.TEXTURE0);
  var t = gl.createTexture();
  gl.bindTexture(gl.TEXTURE_2D, t);
  var filter = byte ? gl.LINEAR : (fmt.linear ? gl.LINEAR : gl.NEAREST);
  texParams(filter);
  if (byte) gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
  else gl.texImage2D(gl.TEXTURE_2D, 0, fmt.internal, w, h, 0, fmt.format, fmt.type, null);
  var f = gl.createFramebuffer();
  gl.bindFramebuffer(gl.FRAMEBUFFER, f);
  gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t, 0);
  gl.viewport(0, 0, w, h);
  gl.clearColor(0, 0, 0, 0);
  gl.clear(gl.COLOR_BUFFER_BIT);
  return { tex: t, fbo: f, w: w, h: h, tx: 1 / w, ty: 1 / h };
}
function createDouble(w, h) {
  var a = createFBO(w, h), b = createFBO(w, h);
  return { w: w, h: h, tx: 1 / w, ty: 1 / h, get read() { return a; }, get write() { return b; }, swap: function () { var t = a; a = b; b = t; } };
}
function freeFBO(o) { if (!o) return; gl.deleteTexture(o.tex); gl.deleteFramebuffer(o.fbo); }
function freeDouble(d) { if (!d) return; freeFBO(d.read); freeFBO(d.write); }
function bindTex(unit, tex) { gl.activeTexture(gl.TEXTURE0 + unit); gl.bindTexture(gl.TEXTURE_2D, tex); return unit; }
function blit(target) {
  if (target) { gl.bindFramebuffer(gl.FRAMEBUFFER, target.fbo); gl.viewport(0, 0, target.w, target.h); }
  else { gl.bindFramebuffer(gl.FRAMEBUFFER, null); gl.viewport(0, 0, W, H); }
  gl.drawElements(gl.TRIANGLES, 6, gl.UNSIGNED_SHORT, 0);
}
function use(p) { gl.useProgram(p.p); return p.u; }
function clearFBO(o) { gl.bindFramebuffer(gl.FRAMEBUFFER, o.fbo); gl.viewport(0, 0, o.w, o.h); gl.clearColor(0, 0, 0, 0); gl.clear(gl.COLOR_BUFFER_BIT); }
function clearDouble(d) { clearFBO(d.read); clearFBO(d.write); }

function resFor(base) {
  var asp = cssW / cssH, a = asp < 1 ? 1 / asp : asp;
  var mn = Math.round(base), mx = Math.round(base * a);
  return asp >= 1 ? [mx, mn] : [mn, mx];
}
function copyInto(src, dst) {
  var u = use(P.copy);
  gl.uniform2f(u.texelSize, src.tx, src.ty);
  gl.uniform1i(u.uTex, bindTex(0, src.tex));
  blit(dst);
}
function resizeDouble(old, w, h) {
  var n = createDouble(w, h);
  if (old) { copyInto(old.read, n.read); freeDouble(old); }
  return n;
}

function allocSim(keep) {
  var q = QUAL[quality];
  var s = resFor(q.sim);
  var minPx = Math.min(W, H);
  var d = resFor(Math.min(q.dye, Math.max(160, minPx)));
  simW = s[0]; simH = s[1]; dyeW = d[0]; dyeH = d[1];
  freeDouble(velocity); freeDouble(pressure); freeFBO(divF); freeFBO(curlF);
  velocity = createDouble(simW, simH);
  pressure = createDouble(simW, simH);
  divF = createFBO(simW, simH);
  curlF = createFBO(simW, simH);
  wet = keep ? resizeDouble(wet, dyeW, dyeH) : (freeDouble(wet), createDouble(dyeW, dyeH));
  dep = keep ? resizeDouble(dep, dyeW, dyeH) : (freeDouble(dep), createDouble(dyeW, dyeH));
}

function makePaper() {
  freeFBO(paperF);
  var pw = Math.min(W, 2600), ph = Math.min(H, 2600);
  paperF = createFBO(pw, ph, true);
  var u = use(P.paper);
  gl.uniform2f(u.texelSize, 1 / pw, 1 / ph);
  gl.uniform2f(u.res, pw, ph);
  gl.uniform1f(u.seed, 17.0);
  gl.uniform1f(u.scale, DPR * pw / W);
  blit(paperF);
}

function uploadCanvasTex(tex, src) {
  if (!tex) tex = gl.createTexture();
  gl.activeTexture(gl.TEXTURE0);
  gl.bindTexture(gl.TEXTURE_2D, tex);
  texParams(gl.LINEAR);
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
  if (src.data) gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, src.w, src.h, 0, gl.RGBA, gl.UNSIGNED_BYTE, src.data);
  else gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, src);
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, false);
  return tex;
}

/* ---------------- 2D helpers: glyphs, masks, seals ---------------- */
function mulberry32(a) { return function () { a |= 0; a = a + 0x6D2B79F5 | 0; var t = Math.imul(a ^ a >>> 15, 1 | a); t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t; return ((t ^ t >>> 14) >>> 0) / 4294967296; }; }

function drawGlyph(ctx, ch, cx, cy, box, thick) {
  var fs = box;
  ctx.textAlign = 'left'; ctx.textBaseline = 'alphabetic';
  function m() {
    ctx.font = fs + 'px ' + FONT;
    var mt = ctx.measureText(ch);
    var asc = mt.actualBoundingBoxAscent, desc = mt.actualBoundingBoxDescent, l = mt.actualBoundingBoxLeft, r = mt.actualBoundingBoxRight;
    if (!(asc >= 0) || isNaN(asc)) { asc = fs * 0.82; desc = fs * 0.1; l = 0; r = mt.width; }
    return { asc: asc, desc: desc, l: l, r: r };
  }
  var b = m();
  var k = Math.min(box / Math.max(b.l + b.r, 1), box / Math.max(b.asc + b.desc, 1), 1.5);
  fs = fs * k; b = m();
  var x = cx - (b.r - b.l) / 2, y = cy + (b.asc - b.desc) / 2;
  if (thick > 0) { ctx.lineWidth = thick; ctx.lineJoin = 'round'; ctx.lineCap = 'round'; ctx.strokeText(ch, x, y); }
  ctx.fillText(ch, x, y);
}

function makeSeal(text, size, color) {
  var c = document.createElement('canvas');
  c.width = c.height = size;
  var x = c.getContext('2d');
  var seed = 7; for (var i = 0; i < text.length; i++) seed = seed * 31 + text.charCodeAt(i);
  var rnd = mulberry32(seed);
  x.fillStyle = color || '#fff';
  var m = size * 0.05, j = size * 0.014, N = 9;
  x.beginPath();
  for (var s = 0; s < 4; s++) {
    for (var k = 0; k < N; k++) {
      var t = k / N, px, py;
      if (s === 0) { px = m + t * (size - 2 * m); py = m; }
      else if (s === 1) { px = size - m; py = m + t * (size - 2 * m); }
      else if (s === 2) { px = size - m - t * (size - 2 * m); py = size - m; }
      else { px = m; py = size - m - t * (size - 2 * m); }
      px += (rnd() - 0.5) * j * 2; py += (rnd() - 0.5) * j * 2;
      if (s === 0 && k === 0) x.moveTo(px, py); else x.lineTo(px, py);
    }
  }
  x.closePath(); x.fill();
  x.globalCompositeOperation = 'destination-out';
  x.fillStyle = '#000'; x.strokeStyle = '#000';
  if (text.length === 1) drawGlyph(x, text, size / 2, size / 2, size * 0.7, size * 0.035);
  else { drawGlyph(x, text[0], size / 2, size * 0.3, size * 0.4, size * 0.03); drawGlyph(x, text[1], size / 2, size * 0.7, size * 0.4, size * 0.03); }
  for (i = 0; i < 90; i++) { x.beginPath(); x.arc(rnd() * size, rnd() * size, (0.003 + rnd() * rnd() * 0.02) * size, 0, 6.2832); x.fill(); }
  for (i = 0; i < 14; i++) { var e = rnd() * 4 | 0, tt = rnd() * size, rr = (0.01 + rnd() * 0.025) * size; x.beginPath(); x.arc(e === 0 ? tt : e === 1 ? size - m : e === 2 ? tt : m, e === 0 ? m : e === 1 ? tt : e === 2 ? size - m : tt, rr, 0, 6.2832); x.fill(); }
  x.globalCompositeOperation = 'source-over';
  return c;
}
function getSealTex(text) {
  if (!sealTex[text]) sealTex[text] = uploadCanvasTex(null, makeSeal(text, 256));
  return sealTex[text];
}

/* level target */
var maskCanvas = document.createElement('canvas');
var target = null; // {cx, cy, s}
function layoutTarget() {
  var top = coarse || cssW < 520 ? 104 : 112, bottom = 66;
  var avail = Math.max(120, cssH - top - bottom);
  var s = Math.min(cssW * 0.86, avail * 0.88, 600);
  return { cx: cssW / 2, cy: top + avail / 2, s: s };
}
function drawTarget(ctx, lv, t) {
  ctx.fillStyle = '#fff'; ctx.strokeStyle = '#fff';
  var cx = t.cx, cy = t.cy, s = t.s;
  if (lv.type === 'char') {
    drawGlyph(ctx, lv.ch, cx, cy, s * 0.8, s * 0.05);
  } else if (lv.type === 'moon') {
    ctx.beginPath(); ctx.arc(cx, cy, s * 0.3, 0, Math.PI * 2); ctx.fill();
  } else if (lv.type === 'hills') {
    var pts = [[-0.47, 0.2], [-0.37, 0.02], [-0.28, 0.09], [-0.13, -0.24], [-0.02, -0.05], [0.09, -0.15], [0.21, 0.04], [0.33, -0.07], [0.47, 0.2]];
    ctx.beginPath();
    ctx.moveTo(cx + pts[0][0] * s, cy + pts[0][1] * s);
    for (var i = 1; i < pts.length - 1; i++) {
      var a = pts[i], b = pts[i + 1];
      var mx = (a[0] + b[0]) / 2, my = (a[1] + b[1]) / 2;
      ctx.quadraticCurveTo(cx + a[0] * s, cy + a[1] * s, cx + mx * s, cy + my * s);
    }
    ctx.lineTo(cx + 0.47 * s, cy + 0.2 * s);
    ctx.closePath(); ctx.fill();
    ctx.lineWidth = s * 0.04; ctx.lineJoin = 'round'; ctx.stroke();
  }
}
function buildMask(lv) {
  target = layoutTarget();
  maskCanvas.width = Math.max(2, Math.round(cssW)); maskCanvas.height = Math.max(2, Math.round(cssH));
  var ctx = maskCanvas.getContext('2d');
  ctx.clearRect(0, 0, maskCanvas.width, maskCanvas.height);
  drawTarget(ctx, lv, target);
  maskT = uploadCanvasTex(maskT, maskCanvas);
  // grid for scoring
  var g = document.createElement('canvas'); g.width = gw; g.height = gh;
  var gx = g.getContext('2d', { willReadFrequently: true });
  gx.drawImage(maskCanvas, 0, 0, gw, gh);
  var data = gx.getImageData(0, 0, gw, gh).data;
  maskIn = new Uint8Array(gw * gh); maskDil = new Uint8Array(gw * gh); maskCount = 0;
  for (var i = 0; i < gw * gh; i++) { if (data[i * 4 + 3] > 120) { maskIn[i] = 1; maskCount++; } }
  maskCount = Math.max(1, maskCount);
  var cellPx = cssW / gw, R = Math.max(2, Math.round(brushPx() * 1.3 / cellPx));
  for (var y = 0; y < gh; y++) for (var x = 0; x < gw; x++) {
    if (!maskIn[y * gw + x]) continue;
    for (var dy = -R; dy <= R; dy++) { var yy = y + dy; if (yy < 0 || yy >= gh) continue;
      for (var dx = -R; dx <= R; dx++) { var xx = x + dx; if (xx < 0 || xx >= gw || dx * dx + dy * dy > R * R) continue; maskDil[yy * gw + xx] = 1; } }
  }
}
function emptyMask() {
  if (!maskT) { maskCanvas.width = maskCanvas.height = 2; maskT = uploadCanvasTex(maskT, maskCanvas); }
}

/* title ink: four characters, one per RGBA channel so they can bloom one after another */
var titleLay = null;
function layoutTitle() {
  var fs = Math.min(cssW * 0.84 / 4, cssH * 0.14, 138);
  var cy = Math.max(cssH * 0.25, fs * 0.75 + 56);
  var sealS = Math.round(fs * 0.36);
  var sx, sy;
  if (cssW / 2 + 2 * fs + sealS * 1.5 < cssW - 10) { sx = cssW / 2 + 2 * fs + sealS * 0.8; sy = cy + fs * 0.22; }
  else { sx = cssW / 2 + 2 * fs - sealS * 0.75; sy = cy + fs * 0.52 + sealS * 0.62; }
  var subTop = Math.max(cy + fs * 0.66, sy + sealS * 0.62) + 16;
  return { fs: fs, cy: cy, sealS: sealS, sx: sx, sy: sy, subTop: subTop };
}
function buildTitleTex() {
  titleLay = layoutTitle();
  var w = Math.max(2, Math.round(cssW)), h = Math.max(2, Math.round(cssH));
  var out = new Uint8Array(w * h * 4);
  var c = document.createElement('canvas'); c.width = w; c.height = h;
  var x = c.getContext('2d', { willReadFrequently: true });
  var chars = '水墨流体';
  for (var i = 0; i < 4; i++) {
    x.clearRect(0, 0, w, h); x.fillStyle = '#fff'; x.strokeStyle = '#fff';
    drawGlyph(x, chars[i], cssW / 2 + (i - 1.5) * titleLay.fs * 1.0, titleLay.cy, titleLay.fs * 0.9, titleLay.fs * 0.012);
    var d = x.getImageData(0, 0, w, h).data;
    for (var p = 0; p < w * h; p++) out[p * 4 + i] = d[p * 4 + 3];
  }
  titleT = uploadCanvasTex(titleT, { data: out, w: w, h: h });
  document.documentElement.style.setProperty('--sub-top', titleLay.subTop + 'px');
}

/* ---------------- simulation ---------------- */
var params = { perm: 1.5, dep: 1.0, cur: 0, curl: 22, velDiss: 1.8, evap: 0.045, pigMob: 0.62, spread: 2.0 };
var simTime = 0;

function aspect() { return cssW / cssH; }
function brushPx() { return Math.max(9, Math.min(24, Math.min(cssW, cssH) * 0.03)); }

function splatVel(ax, ay, bx, by, vx, vy, r, k) {
  var u = use(P.splatVel);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform1i(u.uTarget, bindTex(0, velocity.read.tex));
  gl.uniform1f(u.aspect, aspect());
  gl.uniform2f(u.pA, ax, ay); gl.uniform2f(u.pB, bx, by);
  gl.uniform1f(u.radius, r);
  gl.uniform3f(u.value, vx, vy, k);
  blit(velocity.write); velocity.swap();
}
function splatInk(ax, ay, bx, by, r, black, red, water, wr) {
  var u = use(P.splatInk);
  gl.uniform2f(u.texelSize, wet.tx, wet.ty);
  gl.uniform1i(u.uTarget, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uPaper, bindTex(1, paperF.tex));
  gl.uniform1f(u.aspect, aspect());
  gl.uniform2f(u.pA, ax, ay); gl.uniform2f(u.pB, bx, by);
  gl.uniform1f(u.radius, r);
  gl.uniform1f(u.wr, wr || 1);
  gl.uniform3f(u.ink, black, red, water);
  blit(wet.write); wet.swap();
}
function texInk(tex, w4, water, red) {
  var u = use(P.texInk);
  gl.uniform2f(u.texelSize, wet.tx, wet.ty);
  gl.uniform1i(u.uTarget, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uMask, bindTex(1, tex));
  gl.uniform1i(u.uPaper, bindTex(2, paperF.tex));
  gl.uniform4f(u.w, w4[0], w4[1], w4[2], w4[3]);
  gl.uniform1f(u.water, water); gl.uniform1f(u.red, red);
  blit(wet.write); wet.swap();
}
function stampGL(text, cx, cy, s, amt) {
  var st = getSealTex(text);
  var u = use(P.stamp);
  gl.uniform2f(u.texelSize, dep.tx, dep.ty);
  gl.uniform1i(u.uTarget, bindTex(0, dep.read.tex));
  gl.uniform1i(u.uSeal, bindTex(1, st));
  gl.uniform4f(u.rect, (cx - s / 2) / cssW, 1 - (cy + s / 2) / cssH, s / cssW, s / cssH);
  gl.uniform1f(u.amt, amt);
  blit(dep.write); dep.swap();
}
function scaleDouble(d, f) {
  var u = use(P.scale);
  gl.uniform2f(u.texelSize, d.tx, d.ty);
  gl.uniform1i(u.uTex, bindTex(0, d.read.tex));
  gl.uniform4f(u.s, f, f, f, f);
  blit(d.write); d.swap();
}

function step(dt) {
  simTime += dt;
  var u;
  if (params.cur > 0) {
    u = use(P.current);
    gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
    gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
    gl.uniform1f(u.amp, params.cur * simH / 128);
    gl.uniform1f(u.t, simTime); gl.uniform1f(u.aspect, aspect()); gl.uniform1f(u.dt, dt);
    blit(velocity.write); velocity.swap();
  }
  applyInput(dt);

  u = use(P.curl);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
  blit(curlF);

  u = use(P.vort);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
  gl.uniform1i(u.uCurl, bindTex(1, curlF.tex));
  gl.uniform1f(u.curl, params.curl * (reduceMotion ? 0.45 : 1));
  gl.uniform1f(u.dt, dt);
  blit(velocity.write); velocity.swap();

  u = use(P.div);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
  blit(divF);

  u = use(P.scale);
  gl.uniform2f(u.texelSize, pressure.tx, pressure.ty);
  gl.uniform1i(u.uTex, bindTex(0, pressure.read.tex));
  gl.uniform4f(u.s, 0.8, 0.8, 0.8, 0.8);
  blit(pressure.write); pressure.swap();

  u = use(P.press);
  gl.uniform2f(u.texelSize, pressure.tx, pressure.ty);
  gl.uniform1i(u.uDivergence, bindTex(1, divF.tex));
  var it = QUAL[quality].iters;
  for (var i = 0; i < it; i++) {
    gl.uniform1i(u.uPressure, bindTex(0, pressure.read.tex));
    blit(pressure.write); pressure.swap();
  }

  u = use(P.grad);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform1i(u.uPressure, bindTex(0, pressure.read.tex));
  gl.uniform1i(u.uVelocity, bindTex(1, velocity.read.tex));
  blit(velocity.write); velocity.swap();

  u = use(P.advect);
  gl.uniform2f(u.texelSize, velocity.tx, velocity.ty);
  gl.uniform2f(u.dyeTexel, velocity.tx, velocity.ty);
  gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
  gl.uniform1i(u.uSource, bindTex(0, velocity.read.tex));
  gl.uniform1f(u.dt, dt);
  gl.uniform4f(u.diss, params.velDiss, params.velDiss, 0, 0);
  blit(velocity.write); velocity.swap();

  gl.uniform2f(u.dyeTexel, wet.tx, wet.ty);
  gl.uniform1i(u.uVelocity, bindTex(0, velocity.read.tex));
  gl.uniform1i(u.uSource, bindTex(1, wet.read.tex));
  gl.uniform4f(u.diss, 0, 0, 0, 0);
  blit(wet.write); wet.swap();

  // ink settles into the paper (deposit) and wet water creeps along fibres (bleed)
  var sp = params.spread * Math.max(1, dyeW / Math.max(1, cssW * 1.15));
  u = use(P.deposit);
  gl.uniform2f(u.texelSize, wet.tx, wet.ty);
  gl.uniform1i(u.uWet, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uPaper, bindTex(1, paperF.tex));
  gl.uniform1i(u.uVelocity, bindTex(2, velocity.read.tex));
  gl.uniform1i(u.uDep, bindTex(3, dep.read.tex));
  gl.uniform1f(u.dt, dt); gl.uniform1f(u.spread, sp); gl.uniform1f(u.depRate, params.dep);
  blit(dep.write); dep.swap();

  u = use(P.bleed);
  gl.uniform2f(u.texelSize, wet.tx, wet.ty);
  gl.uniform1i(u.uWet, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uPaper, bindTex(1, paperF.tex));
  gl.uniform1i(u.uVelocity, bindTex(2, velocity.read.tex));
  gl.uniform1f(u.dt, dt); gl.uniform1f(u.spread, sp); gl.uniform1f(u.depRate, params.dep);
  gl.uniform1f(u.perm, params.perm); gl.uniform1f(u.evap, params.evap); gl.uniform1f(u.pigMob, params.pigMob);
  blit(wet.write); wet.swap();
}

var guideA = 0, guideTarget = 0;
function render() {
  var u = use(P.display);
  gl.uniform2f(u.texelSize, 1 / W, 1 / H);
  gl.uniform1i(u.uWet, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uDep, bindTex(1, dep.read.tex));
  gl.uniform1i(u.uPaper, bindTex(2, paperF.tex));
  gl.uniform1i(u.uMask, bindTex(3, maskT));
  gl.uniform2f(u.dyeTexel, wet.tx, wet.ty);
  gl.uniform2f(u.res, W, H);
  gl.uniform1f(u.dpr, DPR);
  if (mode === 'level' && target && guideA > 0.001) gl.uniform4f(u.rect, target.cx * DPR, (cssH - target.cy) * DPR, target.s / 2 * DPR, guideA);
  else gl.uniform4f(u.rect, 0, 0, 0, 0);
  blit(null);
}

function setupGrid() {
  var base = 90;
  if (cssW < cssH) { gw = base; gh = Math.round(base * cssH / cssW); } else { gh = base; gw = Math.round(base * cssW / cssH); }
  freeFBO(rbF);
  rbF = createFBO(gw, gh, true);
  rbBuf = new Uint8Array(gw * gh * 4);
}
var THR = 0.55;
function readCoverage() {
  var u = use(P.readback);
  gl.uniform2f(u.texelSize, 1 / gw, 1 / gh);
  gl.uniform1i(u.uWet, bindTex(0, wet.read.tex));
  gl.uniform1i(u.uDep, bindTex(1, dep.read.tex));
  gl.uniform2f(u.dyeTexel, wet.tx, wet.ty);
  gl.uniform2f(u.cell, 1 / gw, 1 / gh);
  blit(rbF);
  gl.readPixels(0, 0, gw, gh, gl.RGBA, gl.UNSIGNED_BYTE, rbBuf);
  var cov = 0, spill = 0, t = THR * 255;
  for (var y = 0; y < gh; y++) {
    var row = (gh - 1 - y) * gw, rrow = y * gw;
    for (var x = 0; x < gw; x++) {
      var mi = row + x, a = rbBuf[(rrow + x) * 4];
      if (maskIn[mi]) { if (a > t) cov++; }
      else if (!maskDil[mi] && a > t) spill++;
    }
  }
  var lv = LEVELS[levelIdx];
  coverage = cov / maskCount; spillFrac = spill / maskCount;
  score = Math.max(0, coverage - lv.spill * spillFrac);
}

/* ---------------- input ---------------- */
var pointers = new Map();
var inkRed = false;
function toUV(e) { var r = canvas.getBoundingClientRect(); return [(e.clientX - r.left) / r.width, 1 - (e.clientY - r.top) / r.height]; }
canvas.addEventListener('pointerdown', function (e) {
  e.preventDefault();
  try { canvas.setPointerCapture(e.pointerId); } catch (err) {}
  var uv = toUV(e);
  pointers.set(e.pointerId, { x: uv[0], y: uv[1], fx: uv[0], fy: uv[1], still: 0, fresh: true, speed: 0 });
  Snd.init(); Snd.plop();
  onFirstTouch();
});
canvas.addEventListener('pointermove', function (e) {
  var p = pointers.get(e.pointerId);
  if (!p) return;
  e.preventDefault();
  var uv = toUV(e); p.x = uv[0]; p.y = uv[1];
});
function endPtr(e) { pointers.delete(e.pointerId); }
canvas.addEventListener('pointerup', endPtr);
canvas.addEventListener('pointercancel', endPtr);
canvas.addEventListener('lostpointercapture', endPtr);
canvas.addEventListener('contextmenu', function (e) { e.preventDefault(); });
document.addEventListener('gesturestart', function (e) { e.preventDefault(); });
document.addEventListener('dblclick', function (e) { e.preventDefault(); });
document.addEventListener('touchmove', function (e) { if (e.target === canvas) e.preventDefault(); }, { passive: false });

function applyInput(dt) {
  var r = brushPx() / cssH, k = dt * 60, maxSpeed = 0;
  var b = inkRed ? 0 : 1, rd = inkRed ? 1 : 0;
  pointers.forEach(function (p) {
    var dx = p.x - p.fx, dy = p.y - p.fy;
    var distPx = Math.hypot(dx * cssW, dy * cssH);
    if (p.fresh) { splatInk(p.x, p.y, p.x, p.y, r * 1.15, 0.9 * b, 0.9 * rd, 1.0); p.fresh = false; }
    if (distPx > 0.9) {
      var vx = dx / dt * simW, vy = dy / dt * simH, m = Math.hypot(vx, vy), lim = 1400;
      if (m > lim) { vx *= lim / m; vy *= lim / m; }
      splatVel(p.fx, p.fy, p.x, p.y, vx * 0.65, vy * 0.65, r * 1.7, 0.7);
      splatInk(p.fx, p.fy, p.x, p.y, r, 0.30 * k * b, 0.30 * k * rd, 0.42 * k, 1.25);
      p.still = 0;
      p.speed = distPx / dt;
    } else {
      p.still += dt; p.speed *= 0.8;
      var sq = Math.sqrt(p.still);
      var rr = r * (1 + Math.min(0.9, sq * 0.45));
      splatInk(p.x, p.y, p.x, p.y, rr, 0.05 * k * b, 0.05 * k * rd, 0.10 * k, 1 + Math.min(1.8, sq * 0.9));
    }
    maxSpeed = Math.max(maxSpeed, p.speed);
    p.fx = p.x; p.fy = p.y;
  });
  Snd.swish(maxSpeed);
}

/* ---------------- sound (WebAudio, generated) ---------------- */
var Snd = {
  ctx: null, master: null, g: null, bp: null, muted: false,
  init: function () {
    if (this.ctx) { if (this.ctx.state === 'suspended') this.ctx.resume(); return; }
    try {
      var AC = window.AudioContext || window.webkitAudioContext; if (!AC) return;
      var c = this.ctx = new AC();
      var len = c.sampleRate * 2, buf = c.createBuffer(1, len, c.sampleRate), d = buf.getChannelData(0), last = 0;
      for (var i = 0; i < len; i++) { var wv = Math.random() * 2 - 1; last = (last + 0.03 * wv) / 1.03; d[i] = last * 3.2 + wv * 0.08; }
      var src = c.createBufferSource(); src.buffer = buf; src.loop = true;
      this.bp = c.createBiquadFilter(); this.bp.type = 'bandpass'; this.bp.frequency.value = 700; this.bp.Q.value = 0.7;
      this.g = c.createGain(); this.g.gain.value = 0;
      this.master = c.createGain(); this.master.gain.value = this.muted ? 0 : 1;
      src.connect(this.bp); this.bp.connect(this.g); this.g.connect(this.master); this.master.connect(c.destination);
      src.start();
    } catch (e) { this.ctx = null; }
  },
  swish: function (speed) {
    if (!this.ctx || !this.g) return;
    var t = this.ctx.currentTime, v = Math.min(0.11, speed / 2400 * 0.11);
    this.g.gain.setTargetAtTime(v, t, 0.08);
    this.bp.frequency.setTargetAtTime(420 + Math.min(speed, 2500) * 0.45, t, 0.1);
  },
  tone: function (f0, f1, dur, vol, type, delay) {
    if (!this.ctx || this.muted) return;
    var c = this.ctx, t = c.currentTime + (delay || 0), o = c.createOscillator(), g = c.createGain();
    o.type = type || 'sine'; o.frequency.setValueAtTime(f0, t); o.frequency.exponentialRampToValueAtTime(f1, t + dur * 0.7);
    g.gain.setValueAtTime(0.0001, t); g.gain.exponentialRampToValueAtTime(vol, t + 0.012); g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    o.connect(g); g.connect(this.master); o.start(t); o.stop(t + dur + 0.05);
  },
  plop: function () { this.tone(520 + Math.random() * 120, 170, 0.18, 0.05, 'sine'); },
  thud: function () { this.tone(140, 60, 0.28, 0.14, 'sine'); this.tone(900, 300, 0.06, 0.03, 'triangle'); },
  chime: function () { var n = [587.33, 659.25, 880, 1174.66]; for (var i = 0; i < n.length; i++) this.tone(n[i], n[i] * 0.998, 1.6, 0.035, 'triangle', 0.5 + i * 0.13); },
  setMuted: function (m) { this.muted = m; if (this.master) this.master.gain.value = m ? 0 : 1; }
};
try { Snd.muted = localStorage.getItem('ink_mute') === '1'; } catch (e) {}

/* ---------------- game state / UI ---------------- */
var mode = 'title'; // title | level | free
var levelIdx = 0, started = false, frozen = false, levelTime = 0, rbTimer = 0;
var coverage = 0, spillFrac = 0, score = 0;
var washT = 0, washCb = null, titleT0 = -1, titleSealDone = false, firstTouched = false;
var store = { done: [], best: {} };
try { store = JSON.parse(localStorage.getItem('ink_progress') || 'null') || store; } catch (e) {}
if (!Array.isArray(store.done)) store.done = [];
if (!store.best || typeof store.best !== 'object') store.best = {};
function saveStore() { try { localStorage.setItem('ink_progress', JSON.stringify(store)); } catch (e) {} }

var el = {
  title: $('#title'), hud: $('#hud'), result: $('#result'), levels: $('#levels'), hint: $('#hint'),
  lvName: $('#lvName'), lvTip: $('#lvTip'), timer: $('#timer'), progFill: $('#progFill'), progGoal: $('#progGoal'), progTxt: $('#progTxt'),
  rTitle: $('#rTitle'), rLine: $('#rLine'), rBest: $('#rBest'), nextBtn: $('#nextBtn'), snd: $('#sndBtn'),
  back: $('#backGames'), stamp: $('#stampAnim'), toast: $('#toast'), save: $('#saveView'), svImg: $('#svImg'), svDl: $('#svDl'),
  start: $('#startBtn'), prog: $('#prog'), hudBottom: $('#hud .bottom')
};
function fmtTime(s) { s = Math.max(0, Math.floor(s)); return Math.floor(s / 60) + ':' + ('0' + s % 60).slice(-2); }
function toast(msg, ms) { el.toast.textContent = msg; el.toast.classList.add('show'); clearTimeout(toast._t); toast._t = setTimeout(function () { el.toast.classList.remove('show'); }, ms || 2200); }
function unlockedUpTo() { var m = 0; for (var i = 0; i < LEVELS.length; i++) if (store.done.indexOf(i) >= 0) m = i + 1; return Math.min(m, LEVELS.length - 1); }
function renderLevelRow() {
  var up = unlockedUpTo(), h = '';
  for (var i = 0; i < LEVELS.length; i++) {
    var done = store.done.indexOf(i) >= 0;
    h += '<button class="lv' + (done ? ' done' : '') + '" data-act="level" data-i="' + i + '"' + (i > up ? ' disabled' : '') + ' title="第' + NUM[i] + '关 · ' + LEVELS[i].title + '">' + LEVELS[i].ch + '<small>' + NUM[i] + '</small></button>';
  }
  el.levels.innerHTML = h;
  var next = 0; for (i = 0; i < LEVELS.length; i++) { if (store.done.indexOf(i) < 0) { next = i; break; } next = i; }
  el.start.textContent = store.done.length ? (store.done.length >= LEVELS.length ? '再来一遍' : '继续 · 第' + NUM[next] + '关') : '开始';
  el.start.dataset.i = store.done.length >= LEVELS.length ? 0 : next;
}
function updateSndBtn() { el.snd.textContent = Snd.muted ? '静音' : '有声'; el.snd.setAttribute('aria-pressed', Snd.muted ? 'false' : 'true'); }

function showScreen(name) {
  el.title.classList.toggle('hide', name !== 'title');
  el.back.classList.toggle('hide', name !== 'title');
  el.hud.classList.toggle('hide', name === 'title');
  var lvOnly = document.querySelectorAll('.level-only'), frOnly = document.querySelectorAll('.free-only');
  for (var i = 0; i < lvOnly.length; i++) lvOnly[i].classList.toggle('hide', name !== 'level');
  for (i = 0; i < frOnly.length; i++) frOnly[i].classList.toggle('hide', name !== 'free');
  el.prog.classList.toggle('hide', name !== 'level');
  el.timer.classList.toggle('hide', name !== 'level');
  hideResult();
}
function hideResult() { el.result.classList.remove('show'); el.result.classList.add('hide'); el.hudBottom.classList.remove('hide'); }

function wash(cb) { washT = reduceMotion ? 0.12 : 0.5; washCb = cb || null; pointers.clear(); }
function clearAll() { clearDouble(wet); clearDouble(dep); clearDouble(velocity); clearDouble(pressure); }

function goTitle() {
  mode = 'title'; frozen = false; guideTarget = 0;
  params = Object.assign(params, { perm: 1.5, dep: 1.0, cur: 0 });
  showScreen('title'); renderLevelRow();
  wash(function () { titleT0 = 0; titleSealDone = false; });
}
function startLevel(i) {
  levelIdx = Math.max(0, Math.min(LEVELS.length - 1, i | 0));
  var lv = LEVELS[levelIdx];
  mode = 'level'; frozen = false; started = false; levelTime = 0; rbTimer = 0; score = coverage = spillFrac = 0;
  inkRed = false; titleT0 = -1;
  params = Object.assign(params, { perm: lv.perm, dep: lv.dep, cur: lv.cur });
  showScreen('level');
  el.lvName.textContent = '第' + NUM[levelIdx] + '关 · ' + lv.title;
  el.lvTip.textContent = lv.tip;
  el.progGoal.style.left = (lv.need * 100) + '%';
  updateHud(true);
  buildMask(lv);
  guideTarget = 0;
  wash(function () { guideTarget = 1; });
  send('game_start', { game: 'ink', kind: 'level' });
}
function startFree() {
  mode = 'free'; frozen = false; guideTarget = 0; titleT0 = -1;
  params = Object.assign(params, { perm: 1.7, dep: 0.95, cur: 0 });
  showScreen('free');
  el.lvName.textContent = '自由模式';
  el.lvTip.textContent = '随意搅动 · 按住不动会晕开';
  setInk(false);
  wash();
  send('game_start', { game: 'ink', kind: 'free' });
}
function setInk(red) { inkRed = red; $('#inkBlack').classList.toggle('on', !red); $('#inkRed').classList.toggle('on', red); }
var lastShown = '';
function updateHud(force) {
  if (mode !== 'level') return;
  var t = fmtTime(levelTime);
  if (force || t !== lastShown) { el.timer.textContent = t; lastShown = t; }
  var lv = LEVELS[levelIdx], pct = Math.round(score * 100);
  el.progFill.style.width = Math.min(100, score * 100) + '%';
  el.progTxt.textContent = pct + '% / ' + Math.round(lv.need * 100) + '%';
}
function onFirstTouch() {
  if (!firstTouched) { firstTouched = true; el.hint.classList.add('gone'); }
  if (mode === 'level' && !started && !frozen) started = true;
}

function complete() {
  frozen = true; pointers.clear(); Snd.swish(0);
  var lv = LEVELS[levelIdx], secs = levelTime;
  var prevBest = store.best[levelIdx];
  var isBest = !(prevBest > 0) || secs < prevBest;
  if (isBest) store.best[levelIdx] = Math.round(secs * 10) / 10;
  if (store.done.indexOf(levelIdx) < 0) store.done.push(levelIdx);
  saveStore();
  send('game_finish', { game: 'ink', type: String(levelIdx + 1) });
  guideTarget = 0;
  var s = Math.round(Math.max(40, Math.min(target.s * 0.15, 78)));
  var sx = Math.min(cssW - s * 0.75, target.cx + target.s * 0.43);
  var sy = Math.min(cssH - 190 - s * 0.5, target.cy + target.s * 0.08);
  setTimeout(function () {
    dropSeal('成', sx, sy, s, function () {
      Snd.chime();
      el.rTitle.textContent = '第' + NUM[levelIdx] + '关 · ' + lv.title + ' · 成';
      el.rLine.textContent = '用时 ' + fmtTime(secs) + ' · 完成度 ' + Math.round(score * 100) + '%';
      el.rBest.textContent = isBest && prevBest > 0 ? '新纪录！上次 ' + fmtTime(prevBest) : (prevBest > 0 ? '最佳 ' + fmtTime(Math.min(prevBest, secs)) : '');
      var last = levelIdx >= LEVELS.length - 1;
      el.nextBtn.textContent = last ? '自由模式' : '下一关';
      el.nextBtn.dataset.act = last ? 'free' : 'next';
      el.result.classList.remove('hide'); el.hudBottom.classList.add('hide');
      requestAnimationFrame(function () { el.result.classList.add('show'); });
      if (last) toast('八关全部写成 · 去自由模式随手画一幅吧', 3200);
    });
  }, reduceMotion ? 100 : 450);
}

function dropSeal(text, cx, cy, s, done) {
  var c = el.stamp;
  var dpr = Math.min(2, window.devicePixelRatio || 1);
  c.width = c.height = Math.round(s * dpr);
  c.style.width = c.style.height = s + 'px';
  c.style.left = (cx - s / 2) + 'px'; c.style.top = (cy - s / 2) + 'px';
  var x = c.getContext('2d'); x.clearRect(0, 0, c.width, c.height);
  x.drawImage(makeSeal(text, c.width, '#b3311e'), 0, 0);
  c.classList.remove('fade', 'drop'); c.style.opacity = '';
  void c.offsetWidth;
  c.classList.add('drop');
  var fired = false;
  function land() {
    if (fired) return; fired = true;
    stampGL(text, cx, cy, s, 1.5);
    Snd.thud();
    c.classList.add('fade');
    setTimeout(function () { c.classList.remove('drop', 'fade'); c.style.opacity = '0'; if (done) done(); }, 380);
  }
  c.addEventListener('animationend', land, { once: true });
  setTimeout(land, 520);
}

/* ---------------- save as image ---------------- */
function saveImage() {
  if (!glOK) return;
  render();
  var out = document.createElement('canvas');
  out.width = W; out.height = H;
  var x = out.getContext('2d');
  x.drawImage(canvas, 0, 0);
  var fs = Math.max(11, Math.round(Math.min(W, H) * 0.022));
  x.font = fs + 'px ' + FONT; x.textAlign = 'right'; x.textBaseline = 'alphabetic';
  x.fillStyle = 'rgba(30,26,22,0.5)';
  var label = mode === 'level' ? '第' + NUM[levelIdx] + '关 · ' + LEVELS[levelIdx].title + '   ' : '';
  x.fillText(label + '水墨流体 · hivegpt.cn', W - fs * 1.2, H - fs * 1.3);
  var done = function (url) {
    el.svImg.src = url; el.svDl.href = url;
    el.svDl.download = '水墨流体-' + new Date().toISOString().slice(0, 10) + '.png';
    el.save.classList.remove('hide');
  };
  try {
    if (out.toBlob) out.toBlob(function (b) { done(b ? URL.createObjectURL(b) : out.toDataURL('image/png')); }, 'image/png');
    else done(out.toDataURL('image/png'));
  } catch (e) { toast('保存失败，请截图保存'); }
}

/* ---------------- buttons ---------------- */
document.addEventListener('click', function (e) {
  var b = e.target.closest && e.target.closest('[data-act]');
  if (!b) return;
  var act = b.dataset.act;
  Snd.init();
  if (act === 'start') startLevel(+b.dataset.i || 0);
  else if (act === 'level') startLevel(+b.dataset.i);
  else if (act === 'free') startFree();
  else if (act === 'home') goTitle();
  else if (act === 'restart') { if (mode === 'level') startLevel(levelIdx); }
  else if (act === 'next') startLevel(levelIdx + 1);
  else if (act === 'clear') { if (mode === 'level' && frozen) return; wash(); }
  else if (act === 'save') saveImage();
  else if (act === 'closeSave') { el.save.classList.add('hide'); }
  else if (act === 'sound') { Snd.setMuted(!Snd.muted); try { localStorage.setItem('ink_mute', Snd.muted ? '1' : '0'); } catch (err) {} updateSndBtn(); }
  else if (act === 'inkBlack') setInk(false);
  else if (act === 'inkRed') setInk(true);
});
el.save.addEventListener('click', function (e) { if (e.target === el.save) el.save.classList.add('hide'); });

/* ---------------- frame loop ---------------- */
var lastT = 0, perfAcc = 0, perfN = 0, slowWindows = 0;
function update(dt) {
  if (!glOK || lost) return;
  gl.disable(gl.BLEND);
  if (washT > 0) {
    var f = Math.pow(0.015, dt / 0.5);
    scaleDouble(wet, f); scaleDouble(dep, f); scaleDouble(velocity, f);
    washT -= dt;
    if (washT <= 0) { washT = 0; clearAll(); var cb = washCb; washCb = null; if (cb) cb(); }
  } else if (!frozen) {
    if (mode === 'title' && titleT0 >= 0) titleInk(dt);
    step(dt);
    if (mode === 'level' && started) levelTime += dt;
  }
  guideA += (guideTarget - guideA) * Math.min(1, dt * (guideTarget > guideA ? 3 : 2.2));
  if (mode === 'level' && started && !frozen && washT <= 0) {
    rbTimer += dt;
    if (rbTimer >= 0.5) {
      rbTimer = 0; readCoverage();
      if (score >= LEVELS[levelIdx].need) { updateHud(true); complete(); }
    }
  }
  updateHud(false);
  render();
}
function titleInk(dt) {
  var t0 = titleT0, t1 = titleT0 + dt; titleT0 = t1;
  if (reduceMotion) { if (t0 === 0) texInk(titleT, [2.2, 2.2, 2.2, 2.2], 1.2, 0); }
  else {
    var w = [0, 0, 0, 0], any = false;
    for (var i = 0; i < 4; i++) {
      var a = 0.12 + i * 0.28, b = a + 0.55;
      var ov = Math.max(0, Math.min(t1, b) - Math.max(t0, a));
      if (ov > 0) { w[i] = 4.2 * ov / 0.55; any = true; }
    }
    if (any) texInk(titleT, w, 1.1, 0);
  }
  if (!titleSealDone && t1 > (reduceMotion ? 0.2 : 1.85)) {
    titleSealDone = true;
    dropSeal('墨戏', titleLay.sx, titleLay.sy, titleLay.sealS);
  }
}
function frame(t) {
  var dt = lastT ? (t - lastT) / 1000 : 1 / 60;
  lastT = t;
  if (dt > 0 && dt < 0.25 && document.visibilityState === 'visible') {
    perfAcc += dt; perfN++;
    if (perfN >= 150) {
      var avg = perfAcc / perfN; perfAcc = 0; perfN = 0;
      if (avg > 1 / 40) slowWindows++; else slowWindows = 0;
      if (slowWindows >= 2 && quality > 0 && downgrades < 2) { quality--; downgrades++; slowWindows = 0; allocSim(true); }
    }
  }
  if ((Math.max(1, window.innerWidth) !== cssW || Math.max(1, window.innerHeight) !== cssH) && !resizeTimer) window.dispatchEvent(new Event('resize'));
  update(Math.min(Math.max(dt, 0.001), 1 / 30));
  requestAnimationFrame(frame);
}

/* ---------------- sizing / context ---------------- */
function measure() {
  cssW = Math.max(1, window.innerWidth); cssH = Math.max(1, window.innerHeight);
  DPR = Math.min(2, window.devicePixelRatio || 1);
  W = Math.max(1, Math.round(cssW * DPR)); H = Math.max(1, Math.round(cssH * DPR));
}
function setupAll(keep) {
  measure();
  canvas.width = W; canvas.height = H;
  allocSim(keep);
  makePaper();
  setupGrid();
  emptyMask();
  buildTitleTex();
  if (mode === 'level') buildMask(LEVELS[levelIdx]);
}
var resizeTimer = 0, lastSize = '';
window.addEventListener('resize', function () {
  clearTimeout(resizeTimer);
  resizeTimer = setTimeout(function () {
    resizeTimer = 0;
    if (!glOK || lost) return;
    var key = window.innerWidth + 'x' + window.innerHeight + '@' + window.devicePixelRatio;
    if (key === lastSize) return; lastSize = key;
    setupAll(true);
    if (mode === 'title') wash(function () { titleT0 = 0; titleSealDone = false; });
  }, 160);
});
canvas.addEventListener('webglcontextlost', function (e) { e.preventDefault(); lost = true; }, false);
canvas.addEventListener('webglcontextrestored', function () {
  try {
    wet = dep = velocity = pressure = divF = curlF = paperF = rbF = null; maskT = titleT = null;
    if (!initGL()) return;
    lost = false;
    setupAll(false);
    if (mode === 'title') { titleT0 = 0; titleSealDone = true; }
  } catch (err) { console.error(err); }
}, false);
document.addEventListener('visibilitychange', function () { if (document.hidden) { pointers.clear(); Snd.swish(0); } });

/* ---------------- analytics: HiveGPT 小游戏 events ---------------- */
var send = function () {};
(function () {
  if (navigator.webdriver) return;
  function rid() { var b = new Uint8Array(12); crypto.getRandomValues(b); return Array.from(b, function (x) { return x.toString(16).padStart(2, '0'); }).join(''); }
  function id(st, key, ok) { try { var v = st.getItem(key) || ''; if (!ok(v)) { v = rid(); st.setItem(key, v); } return v; } catch (e) { return rid(); } }
  function attr() {
    try { var saved = JSON.parse(localStorage.getItem('hg_attr') || 'null'); if (saved) return saved; } catch (e) {}
    var q = new URLSearchParams(location.search), ref = '';
    try { if (document.referrer && new URL(document.referrer).host !== location.host) ref = document.referrer.slice(0, 300); } catch (e) {}
    return { source: (q.get('utm_source') || '').slice(0, 64), medium: (q.get('utm_medium') || '').slice(0, 64), campaign: (q.get('utm_campaign') || '').slice(0, 64), aff: (q.get('aff') || '').slice(0, 64), referrer: ref, landing: location.pathname, first_seen: Date.now() };
  }
  send = function (name, props) {
    try {
      var ev = { name: name, path: location.pathname };
      if (props) ev.props = props;
      var body = JSON.stringify({
        visitor_id: id(localStorage, 'hg_vid', function (v) { return /^[A-Za-z0-9_-]{8,64}$/.test(v); }),
        session_id: id(sessionStorage, 'hg_sid', function (v) { return !!v; }),
        app: 'main', attr: attr(), events: [ev]
      });
      fetch('/api/v1/events', { method: 'POST', keepalive: true, headers: { 'Content-Type': 'application/json' }, body: body }).catch(function () {});
    } catch (e) {}
  };
  send('page_view');
})();

/* ---------------- boot ---------------- */
updateSndBtn();
renderLevelRow();
try {
  measure();
  canvas.width = W; canvas.height = H;
  glOK = initGL();
  if (glOK) { setupAll(false); lastSize = window.innerWidth + 'x' + window.innerHeight + '@' + window.devicePixelRatio; }
} catch (err) { console.error(err); glOK = false; }
if (!glOK) {
  document.body.style.background = '#efe8d8';
  toast('你的浏览器不支持 WebGL 浮点纹理，换 Chrome 或 Safari 试试', 60000);
} else {
  titleT0 = 0;
  requestAnimationFrame(frame);
}

/* test hook (harmless in production): drive frames manually when the tab is throttled */
window.__ink = {
  tick: function (n, dt) { for (var i = 0; i < (n || 1); i++) update(dt || 1 / 60); },
  get state() { return { mode: mode, levelIdx: levelIdx, started: started, frozen: frozen, score: score, coverage: coverage, spill: spillFrac, time: levelTime, quality: quality, sim: [simW, simH], dye: [dyeW, dyeH], gl2: isGL2, linear: fmt && fmt.linear, grid: [gw, gh] }; },
  target: function () { return target; },
  mask: function () { return { gw: gw, gh: gh, maskIn: maskIn }; },
  startLevel: startLevel, startFree: startFree, goTitle: goTitle, read: function () { readCoverage(); return score; }, render: render,
  setQuality: function (q) { quality = q; allocSim(true); },
  levels: LEVELS, setThr: function (v) { THR = v; }
};
})();
