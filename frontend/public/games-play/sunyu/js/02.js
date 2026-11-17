
var mp=Object.defineProperty;var gp=(i,e)=>{for(var t in e)mp(i,t,{get:e[t],enumerable:!0})};var Gh={};gp(Gh,{attempt:()=>Ff,cur:()=>re,demo:()=>zh,hint:()=>Vh,hooks:()=>wt,load:()=>kl,select:()=>$s,setActive:()=>Bh,toggleXray:()=>kh,tween:()=>$n});var Do="186",Ti={LEFT:0,MIDDLE:1,RIGHT:2,ROTATE:0,DOLLY:1,PAN:2},wi={ROTATE:0,PAN:1,DOLLY_PAN:2,DOLLY_ROTATE:3},qu=0,Dc=1,Yu=2;var Xr=1,No=2,Os=3,Ai=0,Xt=1,Wn=2,Xn=0,Bs=1,Nc=2,Uc=3,Fc=4,Zu=5;var Xi=100,$u=101,Ju=102,ju=103,Ku=104,Qu=200,ed=201,td=202,nd=203,Oc=204,Bc=205,id=206,sd=207,rd=208,ad=209,od=210,ld=211,cd=212,hd=213,ud=214,Ya=0,Za=1,$a=2,vs=3,Ja=4,ja=5,Ka=6,Qa=7,Uo=0,dd=1,fd=2,Dn=0,zc=1,kc=2,Vc=3,qr=4,Hc=5,Gc=6,Wc=7;var Xc=300,Ri=301,qi=302,Fo=303,Oo=304,Yr=306,Ms=1e3,kn=1001,eo=1002,kt=1003,pd=1004;var Zr=1005;var Gt=1006,Bo=1007;var Ci=1008;var rn=1009,qc=1010,Yc=1011,zs=1012,zo=1013,Nn=1014,vn=1015,Un=1016,ko=1017,Vo=1018,ks=1020,Zc=35902,$c=35899,Jc=1021,jc=1022,Mn=1023,Vn=1026,Pi=1027,Ho=1028,Go=1029,Ii=1030,Wo=1031;var Xo=1033,$r=33776,Jr=33777,jr=33778,Kr=33779,qo=35840,Yo=35841,Zo=35842,$o=35843,Jo=36196,jo=37492,Ko=37496,Qo=37488,el=37489,Qr=37490,tl=37491,nl=37808,il=37809,sl=37810,rl=37811,al=37812,ol=37813,ll=37814,cl=37815,hl=37816,ul=37817,dl=37818,fl=37819,pl=37820,ml=37821,gl=36492,_l=36494,xl=36495,yl=36283,vl=36284,ea=36285,Ml=36286;var ur=2300,to=2301,Xa=2302,yc=2303,vc=2400,Mc=2401,bc=2402;var md=3200;var ta=0,gd=1,si="",Vt="srgb",dr="srgb-linear",fr="linear",ut="srgb";var qa=7680;var _d=519,xd=512,yd=513,vd=514,bl=515,Md=516,bd=517,Sl=518,Sd=519,Ed=35044;var Kc="300 es",Pn=2e3,bs=2001;function _p(i){for(let e=i.length-1;e>=0;--e)if(i[e]>=65535)return!0;return!1}function xp(i){return ArrayBuffer.isView(i)&&!(i instanceof DataView)}function pr(i){return document.createElementNS("http://www.w3.org/1999/xhtml",i)}function Td(){let i=pr("canvas");return i.style.display="block",i}var lu={},Ss=null;function Qc(...i){let e="THREE."+i.shift();Ss?Ss("log",e,...i):console.log(e,...i)}function wd(i){let e=i[0];if(typeof e=="string"&&e.startsWith("TSL:")){let t=i[1];t&&t.isStackTrace?i[0]+=" "+t.getLocation():i[1]='Stack trace not available. Enable "THREE.Node.captureStackTrace" to capture stack traces.'}return i}function Oe(...i){i=wd(i);let e="THREE."+i.shift();if(Ss)Ss("warn",e,...i);else{let t=i[0];t&&t.isStackTrace?console.warn(t.getError(e)):console.warn(e,...i)}}function Ve(...i){i=wd(i);let e="THREE."+i.shift();if(Ss)Ss("error",e,...i);else{let t=i[0];t&&t.isStackTrace?console.error(t.getError(e)):console.error(e,...i)}}function Vi(...i){let e=i.join(" ");e in lu||(lu[e]=!0,Oe(...i))}function Ad(i,e,t){return new Promise(function(n,s){function r(){switch(i.clientWaitSync(e,i.SYNC_FLUSH_COMMANDS_BIT,0)){case i.WAIT_FAILED:s();break;case i.TIMEOUT_EXPIRED:setTimeout(r,t);break;default:n()}}setTimeout(r,t)})}var Rd={[Ya]:Za,[$a]:Ka,[Ja]:Qa,[vs]:ja,[Za]:Ya,[Ka]:$a,[Qa]:Ja,[ja]:vs},In=class{addEventListener(e,t){this._listeners===void 0&&(this._listeners={});let n=this._listeners;n[e]===void 0&&(n[e]=[]),n[e].indexOf(t)===-1&&n[e].push(t)}hasEventListener(e,t){let n=this._listeners;return n===void 0?!1:n[e]!==void 0&&n[e].indexOf(t)!==-1}removeEventListener(e,t){let n=this._listeners;if(n===void 0)return;let s=n[e];if(s!==void 0){let r=s.indexOf(t);r!==-1&&s.splice(r,1)}}dispatchEvent(e){let t=this._listeners;if(t===void 0)return;let n=t[e.type];if(n!==void 0){e.target=this;let s=n.slice(0);for(let r=0,a=s.length;r<a;r++)s[r].call(this,e);e.target=null}}},Zt=["00","01","02","03","04","05","06","07","08","09","0a","0b","0c","0d","0e","0f","10","11","12","13","14","15","16","17","18","19","1a","1b","1c","1d","1e","1f","20","21","22","23","24","25","26","27","28","29","2a","2b","2c","2d","2e","2f","30","31","32","33","34","35","36","37","38","39","3a","3b","3c","3d","3e","3f","40","41","42","43","44","45","46","47","48","49","4a","4b","4c","4d","4e","4f","50","51","52","53","54","55","56","57","58","59","5a","5b","5c","5d","5e","5f","60","61","62","63","64","65","66","67","68","69","6a","6b","6c","6d","6e","6f","70","71","72","73","74","75","76","77","78","79","7a","7b","7c","7d","7e","7f","80","81","82","83","84","85","86","87","88","89","8a","8b","8c","8d","8e","8f","90","91","92","93","94","95","96","97","98","99","9a","9b","9c","9d","9e","9f","a0","a1","a2","a3","a4","a5","a6","a7","a8","a9","aa","ab","ac","ad","ae","af","b0","b1","b2","b3","b4","b5","b6","b7","b8","b9","ba","bb","bc","bd","be","bf","c0","c1","c2","c3","c4","c5","c6","c7","c8","c9","ca","cb","cc","cd","ce","cf","d0","d1","d2","d3","d4","d5","d6","d7","d8","d9","da","db","dc","dd","de","df","e0","e1","e2","e3","e4","e5","e6","e7","e8","e9","ea","eb","ec","ed","ee","ef","f0","f1","f2","f3","f4","f5","f6","f7","f8","f9","fa","fb","fc","fd","fe","ff"],cu=1234567,xs=Math.PI/180,Es=180/Math.PI;function Yi(){let i=Math.random()*4294967295|0,e=Math.random()*4294967295|0,t=Math.random()*4294967295|0,n=Math.random()*4294967295|0;return(Zt[i&255]+Zt[i>>8&255]+Zt[i>>16&255]+Zt[i>>24&255]+"-"+Zt[e&255]+Zt[e>>8&255]+"-"+Zt[e>>16&15|64]+Zt[e>>24&255]+"-"+Zt[t&63|128]+Zt[t>>8&255]+"-"+Zt[t>>16&255]+Zt[t>>24&255]+Zt[n&255]+Zt[n>>8&255]+Zt[n>>16&255]+Zt[n>>24&255]).toLowerCase()}function je(i,e,t){return Math.max(e,Math.min(t,i))}function eh(i,e){return(i%e+e)%e}function yp(i,e,t,n,s){return n+(i-e)*(s-n)/(t-e)}function vp(i,e,t){return i!==e?(t-i)/(e-i):0}function lr(i,e,t){return(1-t)*i+t*e}function Mp(i,e,t,n){return lr(i,e,1-Math.exp(-t*n))}function bp(i,e=1){return e-Math.abs(eh(i,e*2)-e)}function Sp(i,e,t){return i<=e?0:i>=t?1:(i=(i-e)/(t-e),i*i*(3-2*i))}function Ep(i,e,t){return i<=e?0:i>=t?1:(i=(i-e)/(t-e),i*i*i*(i*(i*6-15)+10))}function Tp(i,e){return i+Math.floor(Math.random()*(e-i+1))}function wp(i,e){return i+Math.random()*(e-i)}function Ap(i){return i*(.5-Math.random())}function Rp(i){i!==void 0&&(cu=i);let e=cu+=1831565813;return e=Math.imul(e^e>>>15,e|1),e^=e+Math.imul(e^e>>>7,e|61),((e^e>>>14)>>>0)/4294967296}function Cp(i){return i*xs}function Pp(i){return i*Es}function Ip(i){return i>0&&Number.isInteger(i)&&2**Math.round(Math.log2(i))===i}function Lp(i){return Math.pow(2,Math.ceil(Math.log(i)/Math.LN2))}function Dp(i){return Math.pow(2,Math.floor(Math.log(i)/Math.LN2))}function Np(i,e,t,n,s){let r=Math.cos,a=Math.sin,o=r(t/2),l=a(t/2),c=r((e+n)/2),h=a((e+n)/2),d=r((e-n)/2),u=a((e-n)/2),f=r((n-e)/2),g=a((n-e)/2);switch(s){case"XYX":i.set(o*h,l*d,l*u,o*c);break;case"YZY":i.set(l*u,o*h,l*d,o*c);break;case"ZXZ":i.set(l*d,l*u,o*h,o*c);break;case"XZX":i.set(o*h,l*g,l*f,o*c);break;case"YXY":i.set(l*f,o*h,l*g,o*c);break;case"ZYZ":i.set(l*g,l*f,o*h,o*c);break;default:Oe("MathUtils: .setQuaternionFromProperEuler() encountered an unknown order: "+s)}}function _s(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return i/4294967295;case Uint16Array:return i/65535;case Uint8Array:case Uint8ClampedArray:return i/255;case Int32Array:return Math.max(i/2147483647,-1);case Int16Array:return Math.max(i/32767,-1);case Int8Array:return Math.max(i/127,-1);default:throw new Error("THREE.MathUtils: Invalid component type.")}}function Qt(i,e){switch(e.constructor){case Float32Array:return i;case Uint32Array:return Math.round(i*4294967295);case Uint16Array:return Math.round(i*65535);case Uint8Array:case Uint8ClampedArray:return Math.round(i*255);case Int32Array:return Math.round(i*2147483647);case Int16Array:return Math.round(i*32767);case Int8Array:return Math.round(i*127);default:throw new Error("THREE.MathUtils: Invalid component type.")}}var na={DEG2RAD:xs,RAD2DEG:Es,generateUUID:Yi,clamp:je,euclideanModulo:eh,mapLinear:yp,inverseLerp:vp,lerp:lr,damp:Mp,pingpong:bp,smoothstep:Sp,smootherstep:Ep,randInt:Tp,randFloat:wp,randFloatSpread:Ap,seededRandom:Rp,degToRad:Cp,radToDeg:Pp,isPowerOfTwo:Ip,ceilPowerOfTwo:Lp,floorPowerOfTwo:Dp,setQuaternionFromProperEuler:Np,normalize:Qt,denormalize:_s},ah=class ah{constructor(e=0,t=0){this.x=e,this.y=t}get width(){return this.x}set width(e){this.x=e}get height(){return this.y}set height(e){this.y=e}set(e,t){return this.x=e,this.y=t,this}setScalar(e){return this.x=e,this.y=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;default:throw new Error("THREE.Vector2: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;default:throw new Error("THREE.Vector2: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y)}copy(e){return this.x=e.x,this.y=e.y,this}add(e){return this.x+=e.x,this.y+=e.y,this}addScalar(e){return this.x+=e,this.y+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this}subScalar(e){return this.x-=e,this.y-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this}multiply(e){return this.x*=e.x,this.y*=e.y,this}multiplyScalar(e){return this.x*=e,this.y*=e,this}divide(e){return this.x/=e.x,this.y/=e.y,this}divideScalar(e){return this.multiplyScalar(1/e)}applyMatrix3(e){let t=this.x,n=this.y,s=e.elements;return this.x=s[0]*t+s[3]*n+s[6],this.y=s[1]*t+s[4]*n+s[7],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this}clamp(e,t){return this.x=je(this.x,e.x,t.x),this.y=je(this.y,e.y,t.y),this}clampScalar(e,t){return this.x=je(this.x,e,t),this.y=je(this.y,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(je(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this}negate(){return this.x=-this.x,this.y=-this.y,this}dot(e){return this.x*e.x+this.y*e.y}cross(e){return this.x*e.y-this.y*e.x}lengthSq(){return this.x*this.x+this.y*this.y}length(){return Math.sqrt(this.x*this.x+this.y*this.y)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)}normalize(){return this.divideScalar(this.length()||1)}angle(){return Math.atan2(-this.y,-this.x)+Math.PI}angleTo(e){let t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;let n=this.dot(e)/t;return Math.acos(je(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){let t=this.x-e.x,n=this.y-e.y;return t*t+n*n}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this}equals(e){return e.x===this.x&&e.y===this.y}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this}rotateAround(e,t){let n=Math.cos(t),s=Math.sin(t),r=this.x-e.x,a=this.y-e.y;return this.x=r*n-a*s+e.x,this.y=r*s+a*n+e.y,this}random(){return this.x=Math.random(),this.y=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y}};ah.prototype.isVector2=!0;var ae=ah,fn=class{constructor(e=0,t=0,n=0,s=1){this.isQuaternion=!0,this._x=e,this._y=t,this._z=n,this._w=s}static slerpFlat(e,t,n,s,r,a,o){let l=n[s+0],c=n[s+1],h=n[s+2],d=n[s+3],u=r[a+0],f=r[a+1],g=r[a+2],y=r[a+3];if(d!==y||l!==u||c!==f||h!==g){let m=l*u+c*f+h*g+d*y;m<0&&(u=-u,f=-f,g=-g,y=-y,m=-m);let p=1-o;if(m<.9995){let w=Math.acos(m),A=Math.sin(w);p=Math.sin(p*w)/A,o=Math.sin(o*w)/A,l=l*p+u*o,c=c*p+f*o,h=h*p+g*o,d=d*p+y*o}else{l=l*p+u*o,c=c*p+f*o,h=h*p+g*o,d=d*p+y*o;let w=1/Math.sqrt(l*l+c*c+h*h+d*d);l*=w,c*=w,h*=w,d*=w}}e[t]=l,e[t+1]=c,e[t+2]=h,e[t+3]=d}static multiplyQuaternionsFlat(e,t,n,s,r,a){let o=n[s],l=n[s+1],c=n[s+2],h=n[s+3],d=r[a],u=r[a+1],f=r[a+2],g=r[a+3];return e[t]=o*g+h*d+l*f-c*u,e[t+1]=l*g+h*u+c*d-o*f,e[t+2]=c*g+h*f+o*u-l*d,e[t+3]=h*g-o*d-l*u-c*f,e}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get w(){return this._w}set w(e){this._w=e,this._onChangeCallback()}set(e,t,n,s){return this._x=e,this._y=t,this._z=n,this._w=s,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._w)}copy(e){return this._x=e.x,this._y=e.y,this._z=e.z,this._w=e.w,this._onChangeCallback(),this}setFromEuler(e,t=!0){let n=e._x,s=e._y,r=e._z,a=e._order,o=Math.cos,l=Math.sin,c=o(n/2),h=o(s/2),d=o(r/2),u=l(n/2),f=l(s/2),g=l(r/2);switch(a){case"XYZ":this._x=u*h*d+c*f*g,this._y=c*f*d-u*h*g,this._z=c*h*g+u*f*d,this._w=c*h*d-u*f*g;break;case"YXZ":this._x=u*h*d+c*f*g,this._y=c*f*d-u*h*g,this._z=c*h*g-u*f*d,this._w=c*h*d+u*f*g;break;case"ZXY":this._x=u*h*d-c*f*g,this._y=c*f*d+u*h*g,this._z=c*h*g+u*f*d,this._w=c*h*d-u*f*g;break;case"ZYX":this._x=u*h*d-c*f*g,this._y=c*f*d+u*h*g,this._z=c*h*g-u*f*d,this._w=c*h*d+u*f*g;break;case"YZX":this._x=u*h*d+c*f*g,this._y=c*f*d+u*h*g,this._z=c*h*g-u*f*d,this._w=c*h*d-u*f*g;break;case"XZY":this._x=u*h*d-c*f*g,this._y=c*f*d-u*h*g,this._z=c*h*g+u*f*d,this._w=c*h*d+u*f*g;break;default:Oe("Quaternion: .setFromEuler() encountered an unknown order: "+a)}return t===!0&&this._onChangeCallback(),this}setFromAxisAngle(e,t){let n=t/2,s=Math.sin(n);return this._x=e.x*s,this._y=e.y*s,this._z=e.z*s,this._w=Math.cos(n),this._onChangeCallback(),this}setFromRotationMatrix(e){let t=e.elements,n=t[0],s=t[4],r=t[8],a=t[1],o=t[5],l=t[9],c=t[2],h=t[6],d=t[10],u=n+o+d;if(u>0){let f=.5/Math.sqrt(u+1);this._w=.25/f,this._x=(h-l)*f,this._y=(r-c)*f,this._z=(a-s)*f}else if(n>o&&n>d){let f=2*Math.sqrt(1+n-o-d);this._w=(h-l)/f,this._x=.25*f,this._y=(s+a)/f,this._z=(r+c)/f}else if(o>d){let f=2*Math.sqrt(1+o-n-d);this._w=(r-c)/f,this._x=(s+a)/f,this._y=.25*f,this._z=(l+h)/f}else{let f=2*Math.sqrt(1+d-n-o);this._w=(a-s)/f,this._x=(r+c)/f,this._y=(l+h)/f,this._z=.25*f}return this._onChangeCallback(),this}setFromUnitVectors(e,t){let n=e.dot(t)+1;return n<1e-8?(n=0,Math.abs(e.x)>Math.abs(e.z)?(this._x=-e.y,this._y=e.x,this._z=0,this._w=n):(this._x=0,this._y=-e.z,this._z=e.y,this._w=n)):(this._x=e.y*t.z-e.z*t.y,this._y=e.z*t.x-e.x*t.z,this._z=e.x*t.y-e.y*t.x,this._w=n),this.normalize()}angleTo(e){return 2*Math.acos(Math.abs(je(this.dot(e),-1,1)))}rotateTowards(e,t){let n=this.angleTo(e);if(n===0)return this;let s=Math.min(1,t/n);return this.slerp(e,s),this}identity(){return this.set(0,0,0,1)}invert(){return this.conjugate()}conjugate(){return this._x*=-1,this._y*=-1,this._z*=-1,this._onChangeCallback(),this}dot(e){return this._x*e._x+this._y*e._y+this._z*e._z+this._w*e._w}lengthSq(){return this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w}length(){return Math.sqrt(this._x*this._x+this._y*this._y+this._z*this._z+this._w*this._w)}normalize(){let e=this.length();return e===0?(this._x=0,this._y=0,this._z=0,this._w=1):(e=1/e,this._x=this._x*e,this._y=this._y*e,this._z=this._z*e,this._w=this._w*e),this._onChangeCallback(),this}multiply(e){return this.multiplyQuaternions(this,e)}premultiply(e){return this.multiplyQuaternions(e,this)}multiplyQuaternions(e,t){let n=e._x,s=e._y,r=e._z,a=e._w,o=t._x,l=t._y,c=t._z,h=t._w;return this._x=n*h+a*o+s*c-r*l,this._y=s*h+a*l+r*o-n*c,this._z=r*h+a*c+n*l-s*o,this._w=a*h-n*o-s*l-r*c,this._onChangeCallback(),this}slerp(e,t){let n=e._x,s=e._y,r=e._z,a=e._w,o=this.dot(e);o<0&&(n=-n,s=-s,r=-r,a=-a,o=-o);let l=1-t;if(o<.9995){let c=Math.acos(o),h=Math.sin(c);l=Math.sin(l*c)/h,t=Math.sin(t*c)/h,this._x=this._x*l+n*t,this._y=this._y*l+s*t,this._z=this._z*l+r*t,this._w=this._w*l+a*t,this._onChangeCallback()}else this._x=this._x*l+n*t,this._y=this._y*l+s*t,this._z=this._z*l+r*t,this._w=this._w*l+a*t,this.normalize();return this}slerpQuaternions(e,t,n){return this.copy(e).slerp(t,n)}random(){let e=2*Math.PI*Math.random(),t=2*Math.PI*Math.random(),n=Math.random(),s=Math.sqrt(1-n),r=Math.sqrt(n);return this.set(s*Math.sin(e),s*Math.cos(e),r*Math.sin(t),r*Math.cos(t))}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._w===this._w}fromArray(e,t=0){return this._x=e[t],this._y=e[t+1],this._z=e[t+2],this._w=e[t+3],this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._w,e}fromBufferAttribute(e,t){return this._x=e.getX(t),this._y=e.getY(t),this._z=e.getZ(t),this._w=e.getW(t),this._onChangeCallback(),this}toJSON(){return this.toArray()}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._w}},oh=class oh{constructor(e=0,t=0,n=0){this.x=e,this.y=t,this.z=n}set(e,t,n){return n===void 0&&(n=this.z),this.x=e,this.y=t,this.z=n,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;default:throw new Error("THREE.Vector3: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;default:throw new Error("THREE.Vector3: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this}multiplyVectors(e,t){return this.x=e.x*t.x,this.y=e.y*t.y,this.z=e.z*t.z,this}applyEuler(e){return this.applyQuaternion(hu.setFromEuler(e))}applyAxisAngle(e,t){return this.applyQuaternion(hu.setFromAxisAngle(e,t))}applyMatrix3(e){let t=this.x,n=this.y,s=this.z,r=e.elements;return this.x=r[0]*t+r[3]*n+r[6]*s,this.y=r[1]*t+r[4]*n+r[7]*s,this.z=r[2]*t+r[5]*n+r[8]*s,this}applyNormalMatrix(e){return this.applyMatrix3(e).normalize()}applyMatrix4(e){let t=this.x,n=this.y,s=this.z,r=e.elements,a=1/(r[3]*t+r[7]*n+r[11]*s+r[15]);return this.x=(r[0]*t+r[4]*n+r[8]*s+r[12])*a,this.y=(r[1]*t+r[5]*n+r[9]*s+r[13])*a,this.z=(r[2]*t+r[6]*n+r[10]*s+r[14])*a,this}applyQuaternion(e){let t=this.x,n=this.y,s=this.z,r=e.x,a=e.y,o=e.z,l=e.w,c=2*(a*s-o*n),h=2*(o*t-r*s),d=2*(r*n-a*t);return this.x=t+l*c+a*d-o*h,this.y=n+l*h+o*c-r*d,this.z=s+l*d+r*h-a*c,this}project(e){return this.applyMatrix4(e.matrixWorldInverse).applyMatrix4(e.projectionMatrix)}unproject(e){return this.applyMatrix4(e.projectionMatrixInverse).applyMatrix4(e.matrixWorld)}transformDirection(e){let t=this.x,n=this.y,s=this.z,r=e.elements;return this.x=r[0]*t+r[4]*n+r[8]*s,this.y=r[1]*t+r[5]*n+r[9]*s,this.z=r[2]*t+r[6]*n+r[10]*s,this.normalize()}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this}divideScalar(e){return this.multiplyScalar(1/e)}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this}clamp(e,t){return this.x=je(this.x,e.x,t.x),this.y=je(this.y,e.y,t.y),this.z=je(this.z,e.z,t.z),this}clampScalar(e,t){return this.x=je(this.x,e,t),this.y=je(this.y,e,t),this.z=je(this.z,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(je(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this}cross(e){return this.crossVectors(this,e)}crossVectors(e,t){let n=e.x,s=e.y,r=e.z,a=t.x,o=t.y,l=t.z;return this.x=s*l-r*o,this.y=r*a-n*l,this.z=n*o-s*a,this}projectOnVector(e){let t=e.lengthSq();if(t===0)return this.set(0,0,0);let n=e.dot(this)/t;return this.copy(e).multiplyScalar(n)}projectOnPlane(e){return Yl.copy(this).projectOnVector(e),this.sub(Yl)}reflect(e){return this.sub(Yl.copy(e).multiplyScalar(2*this.dot(e)))}angleTo(e){let t=Math.sqrt(this.lengthSq()*e.lengthSq());if(t===0)return Math.PI/2;let n=this.dot(e)/t;return Math.acos(je(n,-1,1))}distanceTo(e){return Math.sqrt(this.distanceToSquared(e))}distanceToSquared(e){let t=this.x-e.x,n=this.y-e.y,s=this.z-e.z;return t*t+n*n+s*s}manhattanDistanceTo(e){return Math.abs(this.x-e.x)+Math.abs(this.y-e.y)+Math.abs(this.z-e.z)}setFromSpherical(e){return this.setFromSphericalCoords(e.radius,e.phi,e.theta)}setFromSphericalCoords(e,t,n){let s=Math.sin(t)*e;return this.x=s*Math.sin(n),this.y=Math.cos(t)*e,this.z=s*Math.cos(n),this}setFromCylindrical(e){return this.setFromCylindricalCoords(e.radius,e.theta,e.y)}setFromCylindricalCoords(e,t,n){return this.x=e*Math.sin(t),this.y=n,this.z=e*Math.cos(t),this}setFromMatrixPosition(e){let t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this}setFromMatrixScale(e){let t=this.setFromMatrixColumn(e,0).length(),n=this.setFromMatrixColumn(e,1).length(),s=this.setFromMatrixColumn(e,2).length();return this.x=t,this.y=n,this.z=s,this}setFromMatrixColumn(e,t){return this.fromArray(e.elements,t*4)}setFromMatrix3Column(e,t){return this.fromArray(e.elements,t*3)}setFromEuler(e){return this.x=e._x,this.y=e._y,this.z=e._z,this}setFromColor(e){return this.x=e.r,this.y=e.g,this.z=e.b,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this}randomDirection(){let e=Math.random()*Math.PI*2,t=Math.random()*2-1,n=Math.sqrt(1-t*t);return this.x=n*Math.cos(e),this.y=t,this.z=n*Math.sin(e),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z}};oh.prototype.isVector3=!0;var R=oh,Yl=new R,hu=new fn,lh=class lh{constructor(e,t,n,s,r,a,o,l,c){this.elements=[1,0,0,0,1,0,0,0,1],e!==void 0&&this.set(e,t,n,s,r,a,o,l,c)}set(e,t,n,s,r,a,o,l,c){let h=this.elements;return h[0]=e,h[1]=s,h[2]=o,h[3]=t,h[4]=r,h[5]=l,h[6]=n,h[7]=a,h[8]=c,this}identity(){return this.set(1,0,0,0,1,0,0,0,1),this}copy(e){let t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],this}extractBasis(e,t,n){return e.setFromMatrix3Column(this,0),t.setFromMatrix3Column(this,1),n.setFromMatrix3Column(this,2),this}setFromMatrix4(e){let t=e.elements;return this.set(t[0],t[4],t[8],t[1],t[5],t[9],t[2],t[6],t[10]),this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){let n=e.elements,s=t.elements,r=this.elements,a=n[0],o=n[3],l=n[6],c=n[1],h=n[4],d=n[7],u=n[2],f=n[5],g=n[8],y=s[0],m=s[3],p=s[6],w=s[1],A=s[4],M=s[7],E=s[2],S=s[5],C=s[8];return r[0]=a*y+o*w+l*E,r[3]=a*m+o*A+l*S,r[6]=a*p+o*M+l*C,r[1]=c*y+h*w+d*E,r[4]=c*m+h*A+d*S,r[7]=c*p+h*M+d*C,r[2]=u*y+f*w+g*E,r[5]=u*m+f*A+g*S,r[8]=u*p+f*M+g*C,this}multiplyScalar(e){let t=this.elements;return t[0]*=e,t[3]*=e,t[6]*=e,t[1]*=e,t[4]*=e,t[7]*=e,t[2]*=e,t[5]*=e,t[8]*=e,this}determinant(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],a=e[4],o=e[5],l=e[6],c=e[7],h=e[8];return t*a*h-t*o*c-n*r*h+n*o*l+s*r*c-s*a*l}invert(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],a=e[4],o=e[5],l=e[6],c=e[7],h=e[8],d=h*a-o*c,u=o*l-h*r,f=c*r-a*l,g=t*d+n*u+s*f;if(g===0)return this.set(0,0,0,0,0,0,0,0,0);let y=1/g;return e[0]=d*y,e[1]=(s*c-h*n)*y,e[2]=(o*n-s*a)*y,e[3]=u*y,e[4]=(h*t-s*l)*y,e[5]=(s*r-o*t)*y,e[6]=f*y,e[7]=(n*l-c*t)*y,e[8]=(a*t-n*r)*y,this}transpose(){let e,t=this.elements;return e=t[1],t[1]=t[3],t[3]=e,e=t[2],t[2]=t[6],t[6]=e,e=t[5],t[5]=t[7],t[7]=e,this}getNormalMatrix(e){return this.setFromMatrix4(e).invert().transpose()}transposeIntoArray(e){let t=this.elements;return e[0]=t[0],e[1]=t[3],e[2]=t[6],e[3]=t[1],e[4]=t[4],e[5]=t[7],e[6]=t[2],e[7]=t[5],e[8]=t[8],this}setUvTransform(e,t,n,s,r,a,o){let l=Math.cos(r),c=Math.sin(r);return this.set(n*l,n*c,-n*(l*a+c*o)+a+e,-s*c,s*l,-s*(-c*a+l*o)+o+t,0,0,1),this}scale(e,t){return Vi("Matrix3: .scale() is deprecated. Use .makeScale() instead."),this.premultiply(Zl.makeScale(e,t)),this}rotate(e){return Vi("Matrix3: .rotate() is deprecated. Use .makeRotation() instead."),this.premultiply(Zl.makeRotation(-e)),this}translate(e,t){return Vi("Matrix3: .translate() is deprecated. Use .makeTranslation() instead."),this.premultiply(Zl.makeTranslation(e,t)),this}makeTranslation(e,t){return e.isVector2?this.set(1,0,e.x,0,1,e.y,0,0,1):this.set(1,0,e,0,1,t,0,0,1),this}makeRotation(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,n,t,0,0,0,1),this}makeScale(e,t){return this.set(e,0,0,0,t,0,0,0,1),this}equals(e){let t=this.elements,n=e.elements;for(let s=0;s<9;s++)if(t[s]!==n[s])return!1;return!0}fromArray(e,t=0){for(let n=0;n<9;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){let n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e}clone(){return new this.constructor().fromArray(this.elements)}};lh.prototype.isMatrix3=!0;var Xe=lh,Zl=new Xe,uu=new Xe().set(.4123908,.3575843,.1804808,.212639,.7151687,.0721923,.0193308,.1191948,.9505322),du=new Xe().set(3.2409699,-1.5373832,-.4986108,-.9692436,1.8759675,.0415551,.0556301,-.203977,1.0569715);function Up(){let i={enabled:!0,workingColorSpace:dr,spaces:{},convert:function(s,r,a){return this.enabled===!1||r===a||!r||!a||(this.spaces[r].transfer===ut&&(s.r=ni(s.r),s.g=ni(s.g),s.b=ni(s.b)),this.spaces[r].primaries!==this.spaces[a].primaries&&(s.applyMatrix3(this.spaces[r].toXYZ),s.applyMatrix3(this.spaces[a].fromXYZ)),this.spaces[a].transfer===ut&&(s.r=ys(s.r),s.g=ys(s.g),s.b=ys(s.b))),s},workingToColorSpace:function(s,r){return this.convert(s,this.workingColorSpace,r)},colorSpaceToWorking:function(s,r){return this.convert(s,r,this.workingColorSpace)},getPrimaries:function(s){return this.spaces[s].primaries},getTransfer:function(s){return s===si?fr:this.spaces[s].transfer},getToneMappingMode:function(s){return this.spaces[s].outputColorSpaceConfig.toneMappingMode||"standard"},getLuminanceCoefficients:function(s,r=this.workingColorSpace){return s.fromArray(this.spaces[r].luminanceCoefficients)},define:function(s){Object.assign(this.spaces,s)},_getMatrix:function(s,r,a){return s.copy(this.spaces[r].toXYZ).multiply(this.spaces[a].fromXYZ)},_getDrawingBufferColorSpace:function(s){return this.spaces[s].outputColorSpaceConfig.drawingBufferColorSpace},_getUnpackColorSpace:function(s=this.workingColorSpace){return this.spaces[s].workingColorSpaceConfig.unpackColorSpace},fromWorkingColorSpace:function(s,r){return Vi("ColorManagement: .fromWorkingColorSpace() has been renamed to .workingToColorSpace()."),i.workingToColorSpace(s,r)},toWorkingColorSpace:function(s,r){return Vi("ColorManagement: .toWorkingColorSpace() has been renamed to .colorSpaceToWorking()."),i.colorSpaceToWorking(s,r)}},e=[.64,.33,.3,.6,.15,.06],t=[.2126,.7152,.0722],n=[.3127,.329];return i.define({[dr]:{primaries:e,whitePoint:n,transfer:fr,toXYZ:uu,fromXYZ:du,luminanceCoefficients:t,workingColorSpaceConfig:{unpackColorSpace:Vt},outputColorSpaceConfig:{drawingBufferColorSpace:Vt}},[Vt]:{primaries:e,whitePoint:n,transfer:ut,toXYZ:uu,fromXYZ:du,luminanceCoefficients:t,outputColorSpaceConfig:{drawingBufferColorSpace:Vt}}}),i}var it=Up();function ni(i){return i<.04045?i*.0773993808:Math.pow(i*.9478672986+.0521327014,2.4)}function ys(i){return i<.0031308?i*12.92:1.055*Math.pow(i,.41666)-.055}var is,no=class{static getDataURL(e,t="image/png"){if(/^data:/i.test(e.src)||typeof HTMLCanvasElement>"u")return e.src;let n;if(e instanceof HTMLCanvasElement)n=e;else{is===void 0&&(is=pr("canvas")),is.width=e.width,is.height=e.height;let s=is.getContext("2d");e instanceof ImageData?s.putImageData(e,0,0):s.drawImage(e,0,0,e.width,e.height),n=is}return n.toDataURL(t)}static sRGBToLinear(e){if(typeof HTMLImageElement<"u"&&e instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&e instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&e instanceof ImageBitmap){let t=pr("canvas");t.width=e.width,t.height=e.height;let n=t.getContext("2d");n.drawImage(e,0,0,e.width,e.height);let s=n.getImageData(0,0,e.width,e.height),r=s.data;for(let a=0;a<r.length;a++)r[a]=ni(r[a]/255)*255;return n.putImageData(s,0,0),t}else if(e.data){let t=e.data.slice(0);for(let n=0;n<t.length;n++)t instanceof Uint8Array||t instanceof Uint8ClampedArray?t[n]=Math.floor(ni(t[n]/255)*255):t[n]=ni(t[n]);return{data:t,width:e.width,height:e.height}}else return Oe("ImageUtils.sRGBToLinear(): Unsupported image type. No color space conversion applied."),e}},Fp=0,Ts=class{constructor(e=null){this.isTextureSource=!0,Object.defineProperty(this,"id",{value:Fp++}),this.uuid=Yi(),this.data=e,this.dataReady=!0,this.version=0}getSize(e){let t=this.data;return typeof HTMLVideoElement<"u"&&t instanceof HTMLVideoElement?e.set(t.videoWidth,t.videoHeight,0):typeof VideoFrame<"u"&&t instanceof VideoFrame?e.set(t.displayWidth,t.displayHeight,0):t!==null?e.set(t.width,t.height,t.depth||0):e.set(0,0,0),e}set needsUpdate(e){e===!0&&this.version++}toJSON(e){let t=e===void 0||typeof e=="string";if(!t&&e.images[this.uuid]!==void 0)return e.images[this.uuid];let n={uuid:this.uuid,url:""},s=this.data;if(s!==null){let r;if(Array.isArray(s)){r=[];for(let a=0,o=s.length;a<o;a++)s[a].isDataTexture?r.push($l(s[a].image)):r.push($l(s[a]))}else r=$l(s);n.url=r}return t||(e.images[this.uuid]=n),n}};function $l(i){return typeof HTMLImageElement<"u"&&i instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&i instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&i instanceof ImageBitmap?no.getDataURL(i):i.data?{data:Array.from(i.data),width:i.width,height:i.height,type:i.data.constructor.name}:(Oe("Texture: Unable to serialize Texture."),{})}var Op=0,Jl=new R,en=class i extends In{constructor(e=i.DEFAULT_IMAGE,t=i.DEFAULT_MAPPING,n=kn,s=kn,r=Gt,a=Ci,o=Mn,l=rn,c=i.DEFAULT_ANISOTROPY,h=si){super(),this.isTexture=!0,Object.defineProperty(this,"id",{value:Op++}),this.uuid=Yi(),this.name="",this.source=new Ts(e),this.mipmaps=[],this.mapping=t,this.channel=0,this.wrapS=n,this.wrapT=s,this.magFilter=r,this.minFilter=a,this.anisotropy=c,this.format=o,this.internalFormat=null,this.type=l,this.offset=new ae(0,0),this.repeat=new ae(1,1),this.center=new ae(0,0),this.rotation=0,this.matrixAutoUpdate=!0,this.matrix=new Xe,this.generateMipmaps=!0,this.premultiplyAlpha=!1,this.flipY=!0,this.unpackAlignment=4,this.colorSpace=h,this.userData={},this.updateRanges=[],this.version=0,this.onUpdate=null,this.renderTarget=null,this.isRenderTargetTexture=!1,this.isArrayTexture=!!(e&&e.depth&&e.depth>1),this.pmremVersion=0,this.normalized=!1}get width(){return this.source.getSize(Jl).x}get height(){return this.source.getSize(Jl).y}get depth(){return this.source.getSize(Jl).z}get image(){return this.source.data}set image(e){this.source.data=e}updateMatrix(){this.matrix.setUvTransform(this.offset.x,this.offset.y,this.repeat.x,this.repeat.y,this.rotation,this.center.x,this.center.y)}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}clone(){return new this.constructor().copy(this)}copy(e){return this.name=e.name,this.source=e.source,this.mipmaps=e.mipmaps.slice(0),this.mapping=e.mapping,this.channel=e.channel,this.wrapS=e.wrapS,this.wrapT=e.wrapT,this.magFilter=e.magFilter,this.minFilter=e.minFilter,this.anisotropy=e.anisotropy,this.format=e.format,this.internalFormat=e.internalFormat,this.type=e.type,this.normalized=e.normalized,this.offset.copy(e.offset),this.repeat.copy(e.repeat),this.center.copy(e.center),this.rotation=e.rotation,this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrix.copy(e.matrix),this.generateMipmaps=e.generateMipmaps,this.premultiplyAlpha=e.premultiplyAlpha,this.flipY=e.flipY,this.unpackAlignment=e.unpackAlignment,this.colorSpace=e.colorSpace,this.renderTarget=e.renderTarget,this.isRenderTargetTexture=e.isRenderTargetTexture,this.isArrayTexture=e.isArrayTexture,this.userData=JSON.parse(JSON.stringify(e.userData)),this.needsUpdate=!0,this}setValues(e){for(let t in e){let n=e[t];if(n===void 0){Oe(`Texture.setValues(): parameter '${t}' has value of undefined.`);continue}let s=this[t];if(s===void 0){Oe(`Texture.setValues(): property '${t}' does not exist.`);continue}s&&n&&s.isVector2&&n.isVector2||s&&n&&s.isVector3&&n.isVector3||s&&n&&s.isMatrix3&&n.isMatrix3?s.copy(n):this[t]=n}}toJSON(e){let t=e===void 0||typeof e=="string";if(!t&&e.textures[this.uuid]!==void 0)return e.textures[this.uuid];let n={metadata:{version:4.7,type:"Texture",generator:"Texture.toJSON"},uuid:this.uuid,name:this.name,image:this.source.toJSON(e).uuid,mapping:this.mapping,channel:this.channel,repeat:[this.repeat.x,this.repeat.y],offset:[this.offset.x,this.offset.y],center:[this.center.x,this.center.y],rotation:this.rotation,wrap:[this.wrapS,this.wrapT],format:this.format,internalFormat:this.internalFormat,type:this.type,normalized:this.normalized,colorSpace:this.colorSpace,minFilter:this.minFilter,magFilter:this.magFilter,anisotropy:this.anisotropy,flipY:this.flipY,generateMipmaps:this.generateMipmaps,premultiplyAlpha:this.premultiplyAlpha,unpackAlignment:this.unpackAlignment};return Object.keys(this.userData).length>0&&(n.userData=this.userData),t||(e.textures[this.uuid]=n),n}dispose(){this.dispatchEvent({type:"dispose"})}transformUv(e){if(this.mapping!==Xc)return e;if(e.applyMatrix3(this.matrix),e.x<0||e.x>1)switch(this.wrapS){case Ms:e.x=e.x-Math.floor(e.x);break;case kn:e.x=e.x<0?0:1;break;case eo:Math.abs(Math.floor(e.x)%2)===1?e.x=Math.ceil(e.x)-e.x:e.x=e.x-Math.floor(e.x);break}if(e.y<0||e.y>1)switch(this.wrapT){case Ms:e.y=e.y-Math.floor(e.y);break;case kn:e.y=e.y<0?0:1;break;case eo:Math.abs(Math.floor(e.y)%2)===1?e.y=Math.ceil(e.y)-e.y:e.y=e.y-Math.floor(e.y);break}return this.flipY&&(e.y=1-e.y),e}set needsUpdate(e){e===!0&&(this.version++,this.source.needsUpdate=!0)}set needsPMREMUpdate(e){e===!0&&this.pmremVersion++}};en.DEFAULT_IMAGE=null;en.DEFAULT_MAPPING=Xc;en.DEFAULT_ANISOTROPY=1;var ch=class ch{constructor(e=0,t=0,n=0,s=1){this.x=e,this.y=t,this.z=n,this.w=s}get width(){return this.z}set width(e){this.z=e}get height(){return this.w}set height(e){this.w=e}set(e,t,n,s){return this.x=e,this.y=t,this.z=n,this.w=s,this}setScalar(e){return this.x=e,this.y=e,this.z=e,this.w=e,this}setX(e){return this.x=e,this}setY(e){return this.y=e,this}setZ(e){return this.z=e,this}setW(e){return this.w=e,this}setComponent(e,t){switch(e){case 0:this.x=t;break;case 1:this.y=t;break;case 2:this.z=t;break;case 3:this.w=t;break;default:throw new Error("THREE.Vector4: index is out of range: "+e)}return this}getComponent(e){switch(e){case 0:return this.x;case 1:return this.y;case 2:return this.z;case 3:return this.w;default:throw new Error("THREE.Vector4: index is out of range: "+e)}}clone(){return new this.constructor(this.x,this.y,this.z,this.w)}copy(e){return this.x=e.x,this.y=e.y,this.z=e.z,this.w=e.w!==void 0?e.w:1,this}add(e){return this.x+=e.x,this.y+=e.y,this.z+=e.z,this.w+=e.w,this}addScalar(e){return this.x+=e,this.y+=e,this.z+=e,this.w+=e,this}addVectors(e,t){return this.x=e.x+t.x,this.y=e.y+t.y,this.z=e.z+t.z,this.w=e.w+t.w,this}addScaledVector(e,t){return this.x+=e.x*t,this.y+=e.y*t,this.z+=e.z*t,this.w+=e.w*t,this}sub(e){return this.x-=e.x,this.y-=e.y,this.z-=e.z,this.w-=e.w,this}subScalar(e){return this.x-=e,this.y-=e,this.z-=e,this.w-=e,this}subVectors(e,t){return this.x=e.x-t.x,this.y=e.y-t.y,this.z=e.z-t.z,this.w=e.w-t.w,this}multiply(e){return this.x*=e.x,this.y*=e.y,this.z*=e.z,this.w*=e.w,this}multiplyScalar(e){return this.x*=e,this.y*=e,this.z*=e,this.w*=e,this}applyMatrix4(e){let t=this.x,n=this.y,s=this.z,r=this.w,a=e.elements;return this.x=a[0]*t+a[4]*n+a[8]*s+a[12]*r,this.y=a[1]*t+a[5]*n+a[9]*s+a[13]*r,this.z=a[2]*t+a[6]*n+a[10]*s+a[14]*r,this.w=a[3]*t+a[7]*n+a[11]*s+a[15]*r,this}divide(e){return this.x/=e.x,this.y/=e.y,this.z/=e.z,this.w/=e.w,this}divideScalar(e){return this.multiplyScalar(1/e)}setAxisAngleFromQuaternion(e){this.w=2*Math.acos(e.w);let t=Math.sqrt(1-e.w*e.w);return t<1e-4?(this.x=1,this.y=0,this.z=0):(this.x=e.x/t,this.y=e.y/t,this.z=e.z/t),this}setAxisAngleFromRotationMatrix(e){let t,n,s,r,l=e.elements,c=l[0],h=l[4],d=l[8],u=l[1],f=l[5],g=l[9],y=l[2],m=l[6],p=l[10];if(Math.abs(h-u)<.01&&Math.abs(d-y)<.01&&Math.abs(g-m)<.01){if(Math.abs(h+u)<.1&&Math.abs(d+y)<.1&&Math.abs(g+m)<.1&&Math.abs(c+f+p-3)<.1)return this.set(1,0,0,0),this;t=Math.PI;let A=(c+1)/2,M=(f+1)/2,E=(p+1)/2,S=(h+u)/4,C=(d+y)/4,x=(g+m)/4;return A>M&&A>E?A<.01?(n=0,s=.707106781,r=.707106781):(n=Math.sqrt(A),s=S/n,r=C/n):M>E?M<.01?(n=.707106781,s=0,r=.707106781):(s=Math.sqrt(M),n=S/s,r=x/s):E<.01?(n=.707106781,s=.707106781,r=0):(r=Math.sqrt(E),n=C/r,s=x/r),this.set(n,s,r,t),this}let w=Math.sqrt((m-g)*(m-g)+(d-y)*(d-y)+(u-h)*(u-h));return Math.abs(w)<.001&&(w=1),this.x=(m-g)/w,this.y=(d-y)/w,this.z=(u-h)/w,this.w=Math.acos((c+f+p-1)/2),this}setFromMatrixPosition(e){let t=e.elements;return this.x=t[12],this.y=t[13],this.z=t[14],this.w=t[15],this}min(e){return this.x=Math.min(this.x,e.x),this.y=Math.min(this.y,e.y),this.z=Math.min(this.z,e.z),this.w=Math.min(this.w,e.w),this}max(e){return this.x=Math.max(this.x,e.x),this.y=Math.max(this.y,e.y),this.z=Math.max(this.z,e.z),this.w=Math.max(this.w,e.w),this}clamp(e,t){return this.x=je(this.x,e.x,t.x),this.y=je(this.y,e.y,t.y),this.z=je(this.z,e.z,t.z),this.w=je(this.w,e.w,t.w),this}clampScalar(e,t){return this.x=je(this.x,e,t),this.y=je(this.y,e,t),this.z=je(this.z,e,t),this.w=je(this.w,e,t),this}clampLength(e,t){let n=this.length();return this.divideScalar(n||1).multiplyScalar(je(n,e,t))}floor(){return this.x=Math.floor(this.x),this.y=Math.floor(this.y),this.z=Math.floor(this.z),this.w=Math.floor(this.w),this}ceil(){return this.x=Math.ceil(this.x),this.y=Math.ceil(this.y),this.z=Math.ceil(this.z),this.w=Math.ceil(this.w),this}round(){return this.x=Math.round(this.x),this.y=Math.round(this.y),this.z=Math.round(this.z),this.w=Math.round(this.w),this}roundToZero(){return this.x=Math.trunc(this.x),this.y=Math.trunc(this.y),this.z=Math.trunc(this.z),this.w=Math.trunc(this.w),this}negate(){return this.x=-this.x,this.y=-this.y,this.z=-this.z,this.w=-this.w,this}dot(e){return this.x*e.x+this.y*e.y+this.z*e.z+this.w*e.w}lengthSq(){return this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w}length(){return Math.sqrt(this.x*this.x+this.y*this.y+this.z*this.z+this.w*this.w)}manhattanLength(){return Math.abs(this.x)+Math.abs(this.y)+Math.abs(this.z)+Math.abs(this.w)}normalize(){return this.divideScalar(this.length()||1)}setLength(e){return this.normalize().multiplyScalar(e)}lerp(e,t){return this.x+=(e.x-this.x)*t,this.y+=(e.y-this.y)*t,this.z+=(e.z-this.z)*t,this.w+=(e.w-this.w)*t,this}lerpVectors(e,t,n){return this.x=e.x+(t.x-e.x)*n,this.y=e.y+(t.y-e.y)*n,this.z=e.z+(t.z-e.z)*n,this.w=e.w+(t.w-e.w)*n,this}equals(e){return e.x===this.x&&e.y===this.y&&e.z===this.z&&e.w===this.w}fromArray(e,t=0){return this.x=e[t],this.y=e[t+1],this.z=e[t+2],this.w=e[t+3],this}toArray(e=[],t=0){return e[t]=this.x,e[t+1]=this.y,e[t+2]=this.z,e[t+3]=this.w,e}fromBufferAttribute(e,t){return this.x=e.getX(t),this.y=e.getY(t),this.z=e.getZ(t),this.w=e.getW(t),this}random(){return this.x=Math.random(),this.y=Math.random(),this.z=Math.random(),this.w=Math.random(),this}*[Symbol.iterator](){yield this.x,yield this.y,yield this.z,yield this.w}};ch.prototype.isVector4=!0;var St=ch,io=class extends In{constructor(e=1,t=1,n={}){super(),n=Object.assign({generateMipmaps:!1,internalFormat:null,minFilter:Gt,depthBuffer:!0,stencilBuffer:!1,resolveColorBuffer:!0,resolveDepthBuffer:!0,resolveStencilBuffer:!0,storeMultisampledColorBuffer:!0,storeMultisampledDepthBuffer:!0,storeMultisampledStencilBuffer:!0,depthTexture:null,samples:0,count:1,depth:1,multiview:!1,useArrayDepthTexture:!1},n),this.isRenderTarget=!0,this.width=e,this.height=t,this.depth=n.depth,this.scissor=new St(0,0,e,t),this.scissorTest=!1,this.viewport=new St(0,0,e,t),this.textures=[];let s={width:e,height:t,depth:n.depth},r=new en(s),a=n.count;for(let o=0;o<a;o++)this.textures[o]=r.clone(),this.textures[o].isRenderTargetTexture=!0,this.textures[o].renderTarget=this;this._setTextureOptions(n),this.depthBuffer=n.depthBuffer,this.stencilBuffer=n.stencilBuffer,this.resolveColorBuffer=n.resolveColorBuffer,this.resolveDepthBuffer=n.resolveDepthBuffer,this.resolveStencilBuffer=n.resolveStencilBuffer,this.storeMultisampledColorBuffer=n.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=n.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=n.storeMultisampledStencilBuffer,this._depthTexture=null,this.depthTexture=n.depthTexture,this.samples=n.samples,this.multiview=n.multiview,this.useArrayDepthTexture=n.useArrayDepthTexture}_setTextureOptions(e={}){let t={minFilter:Gt,generateMipmaps:!1,flipY:!1,internalFormat:null};e.mapping!==void 0&&(t.mapping=e.mapping),e.wrapS!==void 0&&(t.wrapS=e.wrapS),e.wrapT!==void 0&&(t.wrapT=e.wrapT),e.wrapR!==void 0&&(t.wrapR=e.wrapR),e.magFilter!==void 0&&(t.magFilter=e.magFilter),e.minFilter!==void 0&&(t.minFilter=e.minFilter),e.format!==void 0&&(t.format=e.format),e.type!==void 0&&(t.type=e.type),e.anisotropy!==void 0&&(t.anisotropy=e.anisotropy),e.colorSpace!==void 0&&(t.colorSpace=e.colorSpace),e.flipY!==void 0&&(t.flipY=e.flipY),e.generateMipmaps!==void 0&&(t.generateMipmaps=e.generateMipmaps),e.internalFormat!==void 0&&(t.internalFormat=e.internalFormat);for(let n=0;n<this.textures.length;n++)this.textures[n].setValues(t)}get texture(){return this.textures[0]}set texture(e){this.textures[0]=e}set depthTexture(e){this._depthTexture!==null&&this._depthTexture.renderTarget===this&&(this._depthTexture.renderTarget=null),e!==null&&e.renderTarget===null&&(e.renderTarget=this),this._depthTexture=e}get depthTexture(){return this._depthTexture}setSize(e,t,n=1){if(this.width!==e||this.height!==t||this.depth!==n){this.width=e,this.height=t,this.depth=n;for(let s=0,r=this.textures.length;s<r;s++)this.textures[s].image.width=e,this.textures[s].image.height=t,this.textures[s].image.depth=n,this.textures[s].isData3DTexture!==!0&&(this.textures[s].isArrayTexture=this.textures[s].image.depth>1);this.dispose()}this.viewport.set(0,0,e,t),this.scissor.set(0,0,e,t)}clone(){return new this.constructor().copy(this)}copy(e){this.width=e.width,this.height=e.height,this.depth=e.depth,this.scissor.copy(e.scissor),this.scissorTest=e.scissorTest,this.viewport.copy(e.viewport),this.textures.length=0;for(let t=0,n=e.textures.length;t<n;t++){this.textures[t]=e.textures[t].clone(),this.textures[t].isRenderTargetTexture=!0,this.textures[t].renderTarget=this;let s=Object.assign({},e.textures[t].image);this.textures[t].source=new Ts(s)}if(this.depthBuffer=e.depthBuffer,this.stencilBuffer=e.stencilBuffer,this.resolveColorBuffer=e.resolveColorBuffer,this.resolveDepthBuffer=e.resolveDepthBuffer,this.resolveStencilBuffer=e.resolveStencilBuffer,this.storeMultisampledColorBuffer=e.storeMultisampledColorBuffer,this.storeMultisampledDepthBuffer=e.storeMultisampledDepthBuffer,this.storeMultisampledStencilBuffer=e.storeMultisampledStencilBuffer,e.depthTexture!==null)if(e.depthTexture.renderTarget===e){let t=e.depthTexture.clone();t.renderTarget=null,this.depthTexture=t}else this.depthTexture=e.depthTexture;return this.samples=e.samples,this.multiview=e.multiview,this.useArrayDepthTexture=e.useArrayDepthTexture,this}dispose(){this.dispatchEvent({type:"dispose"})}},sn=class extends io{constructor(e=1,t=1,n={}){super(e,t,n),this.isWebGLRenderTarget=!0}},mr=class extends en{constructor(e=null,t=1,n=1,s=1){super(null),this.isDataArrayTexture=!0,this.image={data:e,width:t,height:n,depth:s},this.magFilter=kt,this.minFilter=kt,this.wrapR=kn,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1,this.layerUpdates=new Set}copy(e){return super.copy(e),this.wrapR=e.wrapR,this}addLayerUpdate(e){this.layerUpdates.add(e)}clearLayerUpdates(){this.layerUpdates.clear()}};var so=class extends en{constructor(e=null,t=1,n=1,s=1){super(null),this.isData3DTexture=!0,this.image={data:e,width:t,height:n,depth:s},this.magFilter=kt,this.minFilter=kt,this.wrapR=kn,this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}copy(e){return super.copy(e),this.wrapR=e.wrapR,this}};var Lo=class Lo{constructor(e,t,n,s,r,a,o,l,c,h,d,u,f,g,y,m){this.elements=[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1],e!==void 0&&this.set(e,t,n,s,r,a,o,l,c,h,d,u,f,g,y,m)}set(e,t,n,s,r,a,o,l,c,h,d,u,f,g,y,m){let p=this.elements;return p[0]=e,p[4]=t,p[8]=n,p[12]=s,p[1]=r,p[5]=a,p[9]=o,p[13]=l,p[2]=c,p[6]=h,p[10]=d,p[14]=u,p[3]=f,p[7]=g,p[11]=y,p[15]=m,this}identity(){return this.set(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1),this}clone(){return new Lo().fromArray(this.elements)}copy(e){let t=this.elements,n=e.elements;return t[0]=n[0],t[1]=n[1],t[2]=n[2],t[3]=n[3],t[4]=n[4],t[5]=n[5],t[6]=n[6],t[7]=n[7],t[8]=n[8],t[9]=n[9],t[10]=n[10],t[11]=n[11],t[12]=n[12],t[13]=n[13],t[14]=n[14],t[15]=n[15],this}copyPosition(e){let t=this.elements,n=e.elements;return t[12]=n[12],t[13]=n[13],t[14]=n[14],this}setFromMatrix3(e){let t=e.elements;return this.set(t[0],t[3],t[6],0,t[1],t[4],t[7],0,t[2],t[5],t[8],0,0,0,0,1),this}extractBasis(e,t,n){return this.determinantAffine()===0?(e.set(1,0,0),t.set(0,1,0),n.set(0,0,1),this):(e.setFromMatrixColumn(this,0),t.setFromMatrixColumn(this,1),n.setFromMatrixColumn(this,2),this)}makeBasis(e,t,n){return this.set(e.x,t.x,n.x,0,e.y,t.y,n.y,0,e.z,t.z,n.z,0,0,0,0,1),this}extractRotation(e){if(e.determinantAffine()===0)return this.identity();let t=this.elements,n=e.elements,s=1/ss.setFromMatrixColumn(e,0).length(),r=1/ss.setFromMatrixColumn(e,1).length(),a=1/ss.setFromMatrixColumn(e,2).length();return t[0]=n[0]*s,t[1]=n[1]*s,t[2]=n[2]*s,t[3]=0,t[4]=n[4]*r,t[5]=n[5]*r,t[6]=n[6]*r,t[7]=0,t[8]=n[8]*a,t[9]=n[9]*a,t[10]=n[10]*a,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromEuler(e){let t=this.elements,n=e.x,s=e.y,r=e.z,a=Math.cos(n),o=Math.sin(n),l=Math.cos(s),c=Math.sin(s),h=Math.cos(r),d=Math.sin(r);if(e.order==="XYZ"){let u=a*h,f=a*d,g=o*h,y=o*d;t[0]=l*h,t[4]=-l*d,t[8]=c,t[1]=f+g*c,t[5]=u-y*c,t[9]=-o*l,t[2]=y-u*c,t[6]=g+f*c,t[10]=a*l}else if(e.order==="YXZ"){let u=l*h,f=l*d,g=c*h,y=c*d;t[0]=u+y*o,t[4]=g*o-f,t[8]=a*c,t[1]=a*d,t[5]=a*h,t[9]=-o,t[2]=f*o-g,t[6]=y+u*o,t[10]=a*l}else if(e.order==="ZXY"){let u=l*h,f=l*d,g=c*h,y=c*d;t[0]=u-y*o,t[4]=-a*d,t[8]=g+f*o,t[1]=f+g*o,t[5]=a*h,t[9]=y-u*o,t[2]=-a*c,t[6]=o,t[10]=a*l}else if(e.order==="ZYX"){let u=a*h,f=a*d,g=o*h,y=o*d;t[0]=l*h,t[4]=g*c-f,t[8]=u*c+y,t[1]=l*d,t[5]=y*c+u,t[9]=f*c-g,t[2]=-c,t[6]=o*l,t[10]=a*l}else if(e.order==="YZX"){let u=a*l,f=a*c,g=o*l,y=o*c;t[0]=l*h,t[4]=y-u*d,t[8]=g*d+f,t[1]=d,t[5]=a*h,t[9]=-o*h,t[2]=-c*h,t[6]=f*d+g,t[10]=u-y*d}else if(e.order==="XZY"){let u=a*l,f=a*c,g=o*l,y=o*c;t[0]=l*h,t[4]=-d,t[8]=c*h,t[1]=u*d+y,t[5]=a*h,t[9]=f*d-g,t[2]=g*d-f,t[6]=o*h,t[10]=y*d+u}return t[3]=0,t[7]=0,t[11]=0,t[12]=0,t[13]=0,t[14]=0,t[15]=1,this}makeRotationFromQuaternion(e){return this.compose(Bp,e,zp)}lookAt(e,t,n){let s=this.elements;return hn.subVectors(e,t),hn.lengthSq()===0&&(hn.z=1),hn.normalize(),pi.crossVectors(n,hn),pi.lengthSq()===0&&(Math.abs(n.z)===1?hn.x+=1e-4:hn.z+=1e-4,hn.normalize(),pi.crossVectors(n,hn)),pi.normalize(),ga.crossVectors(hn,pi),s[0]=pi.x,s[4]=ga.x,s[8]=hn.x,s[1]=pi.y,s[5]=ga.y,s[9]=hn.y,s[2]=pi.z,s[6]=ga.z,s[10]=hn.z,this}multiply(e){return this.multiplyMatrices(this,e)}premultiply(e){return this.multiplyMatrices(e,this)}multiplyMatrices(e,t){let n=e.elements,s=t.elements,r=this.elements,a=n[0],o=n[4],l=n[8],c=n[12],h=n[1],d=n[5],u=n[9],f=n[13],g=n[2],y=n[6],m=n[10],p=n[14],w=n[3],A=n[7],M=n[11],E=n[15],S=s[0],C=s[4],x=s[8],T=s[12],I=s[1],U=s[5],O=s[9],H=s[13],D=s[2],B=s[6],X=s[10],W=s[14],ie=s[3],q=s[7],K=s[11],te=s[15];return r[0]=a*S+o*I+l*D+c*ie,r[4]=a*C+o*U+l*B+c*q,r[8]=a*x+o*O+l*X+c*K,r[12]=a*T+o*H+l*W+c*te,r[1]=h*S+d*I+u*D+f*ie,r[5]=h*C+d*U+u*B+f*q,r[9]=h*x+d*O+u*X+f*K,r[13]=h*T+d*H+u*W+f*te,r[2]=g*S+y*I+m*D+p*ie,r[6]=g*C+y*U+m*B+p*q,r[10]=g*x+y*O+m*X+p*K,r[14]=g*T+y*H+m*W+p*te,r[3]=w*S+A*I+M*D+E*ie,r[7]=w*C+A*U+M*B+E*q,r[11]=w*x+A*O+M*X+E*K,r[15]=w*T+A*H+M*W+E*te,this}multiplyScalar(e){let t=this.elements;return t[0]*=e,t[4]*=e,t[8]*=e,t[12]*=e,t[1]*=e,t[5]*=e,t[9]*=e,t[13]*=e,t[2]*=e,t[6]*=e,t[10]*=e,t[14]*=e,t[3]*=e,t[7]*=e,t[11]*=e,t[15]*=e,this}determinant(){let e=this.elements,t=e[0],n=e[4],s=e[8],r=e[12],a=e[1],o=e[5],l=e[9],c=e[13],h=e[2],d=e[6],u=e[10],f=e[14],g=e[3],y=e[7],m=e[11],p=e[15],w=l*f-c*u,A=o*f-c*d,M=o*u-l*d,E=a*f-c*h,S=a*u-l*h,C=a*d-o*h;return t*(y*w-m*A+p*M)-n*(g*w-m*E+p*S)+s*(g*A-y*E+p*C)-r*(g*M-y*S+m*C)}determinantAffine(){let e=this.elements,t=e[0],n=e[4],s=e[8],r=e[1],a=e[5],o=e[9],l=e[2],c=e[6],h=e[10];return t*(a*h-o*c)-n*(r*h-o*l)+s*(r*c-a*l)}transpose(){let e=this.elements,t;return t=e[1],e[1]=e[4],e[4]=t,t=e[2],e[2]=e[8],e[8]=t,t=e[6],e[6]=e[9],e[9]=t,t=e[3],e[3]=e[12],e[12]=t,t=e[7],e[7]=e[13],e[13]=t,t=e[11],e[11]=e[14],e[14]=t,this}setPosition(e,t,n){let s=this.elements;return e.isVector3?(s[12]=e.x,s[13]=e.y,s[14]=e.z):(s[12]=e,s[13]=t,s[14]=n),this}invert(){let e=this.elements,t=e[0],n=e[1],s=e[2],r=e[3],a=e[4],o=e[5],l=e[6],c=e[7],h=e[8],d=e[9],u=e[10],f=e[11],g=e[12],y=e[13],m=e[14],p=e[15],w=t*o-n*a,A=t*l-s*a,M=t*c-r*a,E=n*l-s*o,S=n*c-r*o,C=s*c-r*l,x=h*y-d*g,T=h*m-u*g,I=h*p-f*g,U=d*m-u*y,O=d*p-f*y,H=u*p-f*m,D=w*H-A*O+M*U+E*I-S*T+C*x;if(D===0)return this.set(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0);let B=1/D;return e[0]=(o*H-l*O+c*U)*B,e[1]=(s*O-n*H-r*U)*B,e[2]=(y*C-m*S+p*E)*B,e[3]=(u*S-d*C-f*E)*B,e[4]=(l*I-a*H-c*T)*B,e[5]=(t*H-s*I+r*T)*B,e[6]=(m*M-g*C-p*A)*B,e[7]=(h*C-u*M+f*A)*B,e[8]=(a*O-o*I+c*x)*B,e[9]=(n*I-t*O-r*x)*B,e[10]=(g*S-y*M+p*w)*B,e[11]=(d*M-h*S-f*w)*B,e[12]=(o*T-a*U-l*x)*B,e[13]=(t*U-n*T+s*x)*B,e[14]=(y*A-g*E-m*w)*B,e[15]=(h*E-d*A+u*w)*B,this}scale(e){let t=this.elements,n=e.x,s=e.y,r=e.z;return t[0]*=n,t[4]*=s,t[8]*=r,t[1]*=n,t[5]*=s,t[9]*=r,t[2]*=n,t[6]*=s,t[10]*=r,t[3]*=n,t[7]*=s,t[11]*=r,this}getMaxScaleOnAxis(){let e=this.elements,t=e[0]*e[0]+e[1]*e[1]+e[2]*e[2],n=e[4]*e[4]+e[5]*e[5]+e[6]*e[6],s=e[8]*e[8]+e[9]*e[9]+e[10]*e[10];return Math.sqrt(Math.max(t,n,s))}makeTranslation(e,t,n){return e.isVector3?this.set(1,0,0,e.x,0,1,0,e.y,0,0,1,e.z,0,0,0,1):this.set(1,0,0,e,0,1,0,t,0,0,1,n,0,0,0,1),this}makeRotationX(e){let t=Math.cos(e),n=Math.sin(e);return this.set(1,0,0,0,0,t,-n,0,0,n,t,0,0,0,0,1),this}makeRotationY(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,0,n,0,0,1,0,0,-n,0,t,0,0,0,0,1),this}makeRotationZ(e){let t=Math.cos(e),n=Math.sin(e);return this.set(t,-n,0,0,n,t,0,0,0,0,1,0,0,0,0,1),this}makeRotationAxis(e,t){let n=Math.cos(t),s=Math.sin(t),r=1-n,a=e.x,o=e.y,l=e.z,c=r*a,h=r*o;return this.set(c*a+n,c*o-s*l,c*l+s*o,0,c*o+s*l,h*o+n,h*l-s*a,0,c*l-s*o,h*l+s*a,r*l*l+n,0,0,0,0,1),this}makeScale(e,t,n){return this.set(e,0,0,0,0,t,0,0,0,0,n,0,0,0,0,1),this}makeShear(e,t,n,s,r,a){return this.set(1,n,r,0,e,1,a,0,t,s,1,0,0,0,0,1),this}compose(e,t,n){let s=this.elements,r=t._x,a=t._y,o=t._z,l=t._w,c=r+r,h=a+a,d=o+o,u=r*c,f=r*h,g=r*d,y=a*h,m=a*d,p=o*d,w=l*c,A=l*h,M=l*d,E=n.x,S=n.y,C=n.z;return s[0]=(1-(y+p))*E,s[1]=(f+M)*E,s[2]=(g-A)*E,s[3]=0,s[4]=(f-M)*S,s[5]=(1-(u+p))*S,s[6]=(m+w)*S,s[7]=0,s[8]=(g+A)*C,s[9]=(m-w)*C,s[10]=(1-(u+y))*C,s[11]=0,s[12]=e.x,s[13]=e.y,s[14]=e.z,s[15]=1,this}decompose(e,t,n){let s=this.elements;e.x=s[12],e.y=s[13],e.z=s[14];let r=this.determinantAffine();if(r===0)return n.set(1,1,1),t.identity(),this;let a=ss.set(s[0],s[1],s[2]).length(),o=ss.set(s[4],s[5],s[6]).length(),l=ss.set(s[8],s[9],s[10]).length();r<0&&(a=-a),An.copy(this);let c=1/a,h=1/o,d=1/l;return An.elements[0]*=c,An.elements[1]*=c,An.elements[2]*=c,An.elements[4]*=h,An.elements[5]*=h,An.elements[6]*=h,An.elements[8]*=d,An.elements[9]*=d,An.elements[10]*=d,t.setFromRotationMatrix(An),n.x=a,n.y=o,n.z=l,this}makePerspective(e,t,n,s,r,a,o=Pn,l=!1){let c=this.elements,h=2*r/(t-e),d=2*r/(n-s),u=(t+e)/(t-e),f=(n+s)/(n-s),g,y;if(l)g=r/(a-r),y=a*r/(a-r);else if(o===Pn)g=-(a+r)/(a-r),y=-2*a*r/(a-r);else if(o===bs)g=-a/(a-r),y=-a*r/(a-r);else throw new Error("THREE.Matrix4.makePerspective(): Invalid coordinate system: "+o);return c[0]=h,c[4]=0,c[8]=u,c[12]=0,c[1]=0,c[5]=d,c[9]=f,c[13]=0,c[2]=0,c[6]=0,c[10]=g,c[14]=y,c[3]=0,c[7]=0,c[11]=-1,c[15]=0,this}makeOrthographic(e,t,n,s,r,a,o=Pn,l=!1){let c=this.elements,h=2/(t-e),d=2/(n-s),u=-(t+e)/(t-e),f=-(n+s)/(n-s),g,y;if(l)g=1/(a-r),y=a/(a-r);else if(o===Pn)g=-2/(a-r),y=-(a+r)/(a-r);else if(o===bs)g=-1/(a-r),y=-r/(a-r);else throw new Error("THREE.Matrix4.makeOrthographic(): Invalid coordinate system: "+o);return c[0]=h,c[4]=0,c[8]=0,c[12]=u,c[1]=0,c[5]=d,c[9]=0,c[13]=f,c[2]=0,c[6]=0,c[10]=g,c[14]=y,c[3]=0,c[7]=0,c[11]=0,c[15]=1,this}equals(e){let t=this.elements,n=e.elements;for(let s=0;s<16;s++)if(t[s]!==n[s])return!1;return!0}fromArray(e,t=0){for(let n=0;n<16;n++)this.elements[n]=e[n+t];return this}toArray(e=[],t=0){let n=this.elements;return e[t]=n[0],e[t+1]=n[1],e[t+2]=n[2],e[t+3]=n[3],e[t+4]=n[4],e[t+5]=n[5],e[t+6]=n[6],e[t+7]=n[7],e[t+8]=n[8],e[t+9]=n[9],e[t+10]=n[10],e[t+11]=n[11],e[t+12]=n[12],e[t+13]=n[13],e[t+14]=n[14],e[t+15]=n[15],e}};Lo.prototype.isMatrix4=!0;var dt=Lo,ss=new R,An=new dt,Bp=new R(0,0,0),zp=new R(1,1,1),pi=new R,ga=new R,hn=new R,fu=new dt,pu=new fn,Hn=class i{constructor(e=0,t=0,n=0,s=i.DEFAULT_ORDER){this.isEuler=!0,this._x=e,this._y=t,this._z=n,this._order=s}get x(){return this._x}set x(e){this._x=e,this._onChangeCallback()}get y(){return this._y}set y(e){this._y=e,this._onChangeCallback()}get z(){return this._z}set z(e){this._z=e,this._onChangeCallback()}get order(){return this._order}set order(e){this._order=e,this._onChangeCallback()}set(e,t,n,s=this._order){return this._x=e,this._y=t,this._z=n,this._order=s,this._onChangeCallback(),this}clone(){return new this.constructor(this._x,this._y,this._z,this._order)}copy(e){return this._x=e._x,this._y=e._y,this._z=e._z,this._order=e._order,this._onChangeCallback(),this}setFromRotationMatrix(e,t=this._order,n=!0){let s=e.elements,r=s[0],a=s[4],o=s[8],l=s[1],c=s[5],h=s[9],d=s[2],u=s[6],f=s[10];switch(t){case"XYZ":this._y=Math.asin(je(o,-1,1)),Math.abs(o)<.9999999?(this._x=Math.atan2(-h,f),this._z=Math.atan2(-a,r)):(this._x=Math.atan2(u,c),this._z=0);break;case"YXZ":this._x=Math.asin(-je(h,-1,1)),Math.abs(h)<.9999999?(this._y=Math.atan2(o,f),this._z=Math.atan2(l,c)):(this._y=Math.atan2(-d,r),this._z=0);break;case"ZXY":this._x=Math.asin(je(u,-1,1)),Math.abs(u)<.9999999?(this._y=Math.atan2(-d,f),this._z=Math.atan2(-a,c)):(this._y=0,this._z=Math.atan2(l,r));break;case"ZYX":this._y=Math.asin(-je(d,-1,1)),Math.abs(d)<.9999999?(this._x=Math.atan2(u,f),this._z=Math.atan2(l,r)):(this._x=0,this._z=Math.atan2(-a,c));break;case"YZX":this._z=Math.asin(je(l,-1,1)),Math.abs(l)<.9999999?(this._x=Math.atan2(-h,c),this._y=Math.atan2(-d,r)):(this._x=0,this._y=Math.atan2(o,f));break;case"XZY":this._z=Math.asin(-je(a,-1,1)),Math.abs(a)<.9999999?(this._x=Math.atan2(u,c),this._y=Math.atan2(o,r)):(this._x=Math.atan2(-h,f),this._y=0);break;default:Oe("Euler: .setFromRotationMatrix() encountered an unknown order: "+t)}return this._order=t,n===!0&&this._onChangeCallback(),this}setFromQuaternion(e,t,n){return fu.makeRotationFromQuaternion(e),this.setFromRotationMatrix(fu,t,n)}setFromVector3(e,t=this._order){return this.set(e.x,e.y,e.z,t)}reorder(e){return pu.setFromEuler(this),this.setFromQuaternion(pu,e)}equals(e){return e._x===this._x&&e._y===this._y&&e._z===this._z&&e._order===this._order}fromArray(e){return this._x=e[0],this._y=e[1],this._z=e[2],e[3]!==void 0&&(this._order=e[3]),this._onChangeCallback(),this}toArray(e=[],t=0){return e[t]=this._x,e[t+1]=this._y,e[t+2]=this._z,e[t+3]=this._order,e}_onChange(e){return this._onChangeCallback=e,this}_onChangeCallback(){}*[Symbol.iterator](){yield this._x,yield this._y,yield this._z,yield this._order}};Hn.DEFAULT_ORDER="XYZ";var ws=class{constructor(){this.mask=1}set(e){this.mask=(1<<e|0)>>>0}enable(e){this.mask|=1<<e|0}enableAll(){this.mask=-1}toggle(e){this.mask^=1<<e|0}disable(e){this.mask&=~(1<<e|0)}disableAll(){this.mask=0}test(e){return(this.mask&e.mask)!==0}isEnabled(e){return(this.mask&(1<<e|0))!==0}},kp=0,mu=new R,rs=new fn,Jn=new dt,_a=new R,er=new R,Vp=new R,Hp=new fn,gu=new R(1,0,0),_u=new R(0,1,0),xu=new R(0,0,1),yu={type:"added"},Gp={type:"removed"},as={type:"childadded",child:null},jl={type:"childremoved",child:null},Lt=class i extends In{constructor(){super(),this.isObject3D=!0,Object.defineProperty(this,"id",{value:kp++}),this.uuid=Yi(),this.name="",this.type="Object3D",this.parent=null,this.children=[],this.up=i.DEFAULT_UP.clone();let e=new R,t=new Hn,n=new fn,s=new R(1,1,1);function r(){n.setFromEuler(t,!1)}function a(){t.setFromQuaternion(n,void 0,!1)}t._onChange(r),n._onChange(a),Object.defineProperties(this,{position:{configurable:!0,enumerable:!0,value:e},rotation:{configurable:!0,enumerable:!0,value:t},quaternion:{configurable:!0,enumerable:!0,value:n},scale:{configurable:!0,enumerable:!0,value:s},modelViewMatrix:{value:new dt},normalMatrix:{value:new Xe}}),this.matrix=new dt,this.matrixWorld=new dt,this.matrixAutoUpdate=i.DEFAULT_MATRIX_AUTO_UPDATE,this.matrixWorldAutoUpdate=i.DEFAULT_MATRIX_WORLD_AUTO_UPDATE,this.matrixWorldNeedsUpdate=!1,this.layers=new ws,this.visible=!0,this.castShadow=!1,this.receiveShadow=!1,this.frustumCulled=!0,this.renderOrder=0,this.animations=[],this.customDepthMaterial=void 0,this.customDistanceMaterial=void 0,this.static=!1,this.userData={},this.pivot=null}onBeforeShadow(){}onAfterShadow(){}onBeforeRender(){}onAfterRender(){}applyMatrix4(e){this.matrixAutoUpdate&&this.updateMatrix(),this.matrix.premultiply(e),this.matrix.decompose(this.position,this.quaternion,this.scale)}applyQuaternion(e){return this.quaternion.premultiply(e),this}setRotationFromAxisAngle(e,t){this.quaternion.setFromAxisAngle(e,t)}setRotationFromEuler(e){this.quaternion.setFromEuler(e,!0)}setRotationFromMatrix(e){this.quaternion.setFromRotationMatrix(e)}setRotationFromQuaternion(e){this.quaternion.copy(e)}rotateOnAxis(e,t){return rs.setFromAxisAngle(e,t),this.quaternion.multiply(rs),this}rotateOnWorldAxis(e,t){return rs.setFromAxisAngle(e,t),this.quaternion.premultiply(rs),this}rotateX(e){return this.rotateOnAxis(gu,e)}rotateY(e){return this.rotateOnAxis(_u,e)}rotateZ(e){return this.rotateOnAxis(xu,e)}translateOnAxis(e,t){return mu.copy(e).applyQuaternion(this.quaternion),this.position.add(mu.multiplyScalar(t)),this}translateX(e){return this.translateOnAxis(gu,e)}translateY(e){return this.translateOnAxis(_u,e)}translateZ(e){return this.translateOnAxis(xu,e)}localToWorld(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(this.matrixWorld)}worldToLocal(e){return this.updateWorldMatrix(!0,!1),e.applyMatrix4(Jn.copy(this.matrixWorld).invert())}lookAt(e,t,n){e.isVector3?_a.copy(e):_a.set(e,t,n);let s=this.parent;this.updateWorldMatrix(!0,!1),er.setFromMatrixPosition(this.matrixWorld),this.isCamera||this.isLight?Jn.lookAt(er,_a,this.up):Jn.lookAt(_a,er,this.up),this.quaternion.setFromRotationMatrix(Jn),s&&(Jn.extractRotation(s.matrixWorld),rs.setFromRotationMatrix(Jn),this.quaternion.premultiply(rs.invert()))}add(e){if(arguments.length>1){for(let t=0;t<arguments.length;t++)this.add(arguments[t]);return this}return e===this?(Ve("Object3D.add: object can't be added as a child of itself.",e),this):(e&&e.isObject3D?(e.removeFromParent(),e.parent=this,this.children.push(e),e.dispatchEvent(yu),as.child=e,this.dispatchEvent(as),as.child=null):Ve("Object3D.add: object not an instance of THREE.Object3D.",e),this)}remove(e){if(arguments.length>1){for(let n=0;n<arguments.length;n++)this.remove(arguments[n]);return this}let t=this.children.indexOf(e);return t!==-1&&(e.parent=null,this.children.splice(t,1),e.dispatchEvent(Gp),jl.child=e,this.dispatchEvent(jl),jl.child=null),this}removeFromParent(){let e=this.parent;return e!==null&&e.remove(this),this}clear(){return this.remove(...this.children)}attach(e){return this.updateWorldMatrix(!0,!1),Jn.copy(this.matrixWorld).invert(),e.parent!==null&&(e.parent.updateWorldMatrix(!0,!1),Jn.multiply(e.parent.matrixWorld)),e.applyMatrix4(Jn),e.removeFromParent(),e.parent=this,this.children.push(e),e.updateWorldMatrix(!1,!0),e.dispatchEvent(yu),as.child=e,this.dispatchEvent(as),as.child=null,this}getObjectById(e){return this.getObjectByProperty("id",e)}getObjectByName(e){return this.getObjectByProperty("name",e)}getObjectByProperty(e,t){if(this[e]===t)return this;for(let n=0,s=this.children.length;n<s;n++){let a=this.children[n].getObjectByProperty(e,t);if(a!==void 0)return a}}getObjectsByProperty(e,t,n=[]){this[e]===t&&n.push(this);let s=this.children;for(let r=0,a=s.length;r<a;r++)s[r].getObjectsByProperty(e,t,n);return n}getWorldPosition(e){return this.updateWorldMatrix(!0,!1),e.setFromMatrixPosition(this.matrixWorld)}getWorldQuaternion(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(er,e,Vp),e}getWorldScale(e){return this.updateWorldMatrix(!0,!1),this.matrixWorld.decompose(er,Hp,e),e}getWorldDirection(e){this.updateWorldMatrix(!0,!1);let t=this.matrixWorld.elements;return e.set(t[8],t[9],t[10]).normalize()}raycast(){}intersectsFrustum(){}traverse(e){e(this);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].traverse(e)}traverseVisible(e){if(this.visible===!1)return;e(this);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].traverseVisible(e)}traverseAncestors(e){let t=this.parent;t!==null&&(e(t),t.traverseAncestors(e))}updateMatrix(){this.matrix.compose(this.position,this.quaternion,this.scale);let e=this.pivot;if(e!==null){let t=e.x,n=e.y,s=e.z,r=this.matrix.elements;r[12]+=t-r[0]*t-r[4]*n-r[8]*s,r[13]+=n-r[1]*t-r[5]*n-r[9]*s,r[14]+=s-r[2]*t-r[6]*n-r[10]*s}this.matrixWorldNeedsUpdate=!0}updateMatrixWorld(e){this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||e)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,e=!0);let t=this.children;for(let n=0,s=t.length;n<s;n++)t[n].updateMatrixWorld(e)}updateWorldMatrix(e,t,n=!1){let s=this.parent;if(e===!0&&s!==null&&s.updateWorldMatrix(!0,!1),this.matrixAutoUpdate&&this.updateMatrix(),(this.matrixWorldNeedsUpdate||n)&&(this.matrixWorldAutoUpdate===!0&&(this.parent===null?this.matrixWorld.copy(this.matrix):this.matrixWorld.multiplyMatrices(this.parent.matrixWorld,this.matrix)),this.matrixWorldNeedsUpdate=!1,n=!0),t===!0){let r=this.children;for(let a=0,o=r.length;a<o;a++)r[a].updateWorldMatrix(!1,!0,n)}}toJSON(e){let t=e===void 0||typeof e=="string",n={};t&&(e={geometries:{},materials:{},textures:{},images:{},shapes:{},skeletons:{},animations:{},nodes:{}},n.metadata={version:4.7,type:"Object",generator:"Object3D.toJSON"});let s={};s.uuid=this.uuid,s.type=this.type,s.name=this.name,s.castShadow=this.castShadow,s.receiveShadow=this.receiveShadow,s.visible=this.visible,s.frustumCulled=this.frustumCulled,s.renderOrder=this.renderOrder,s.static=this.static,s.matrixAutoUpdate=this.matrixAutoUpdate,Object.keys(this.userData).length>0&&(s.userData=this.userData),s.layers=this.layers.mask,s.matrix=this.matrix.toArray(),s.up=this.up.toArray(),this.pivot!==null&&(s.pivot=this.pivot.toArray()),this.morphTargetDictionary!==void 0&&(s.morphTargetDictionary=Object.assign({},this.morphTargetDictionary)),this.morphTargetInfluences!==void 0&&(s.morphTargetInfluences=this.morphTargetInfluences.slice()),this.isInstancedMesh&&(s.type="InstancedMesh",s.count=this.count,s.instanceMatrix=this.instanceMatrix.toJSON(),this.instanceColor!==null&&(s.instanceColor=this.instanceColor.toJSON())),this.isBatchedMesh&&(s.type="BatchedMesh",s.perObjectFrustumCulled=this.perObjectFrustumCulled,s.sortObjects=this.sortObjects,s.drawRanges=this._drawRanges,s.reservedRanges=this._reservedRanges,s.geometryInfo=this._geometryInfo.map(o=>({...o,boundingBox:o.boundingBox?o.boundingBox.toJSON():void 0,boundingSphere:o.boundingSphere?o.boundingSphere.toJSON():void 0})),s.instanceInfo=this._instanceInfo.map(o=>({...o})),s.availableInstanceIds=this._availableInstanceIds.slice(),s.availableGeometryIds=this._availableGeometryIds.slice(),s.nextIndexStart=this._nextIndexStart,s.nextVertexStart=this._nextVertexStart,s.geometryCount=this._geometryCount,s.maxInstanceCount=this._maxInstanceCount,s.maxVertexCount=this._maxVertexCount,s.maxIndexCount=this._maxIndexCount,s.geometryInitialized=this._geometryInitialized,s.matricesTexture=this._matricesTexture.toJSON(e),s.indirectTexture=this._indirectTexture.toJSON(e),this._colorsTexture!==null&&(s.colorsTexture=this._colorsTexture.toJSON(e)),this.boundingSphere!==null&&(s.boundingSphere=this.boundingSphere.toJSON()),this.boundingBox!==null&&(s.boundingBox=this.boundingBox.toJSON()));function r(o,l){return o[l.uuid]===void 0&&(o[l.uuid]=l.toJSON(e)),l.uuid}if(this.isScene)this.background&&(this.background.isColor?s.background=this.background.toJSON():this.background.isTexture&&(s.background=this.background.toJSON(e).uuid)),this.environment&&this.environment.isTexture&&this.environment.isRenderTargetTexture!==!0&&(s.environment=this.environment.toJSON(e).uuid);else if(this.isMesh||this.isLine||this.isPoints){s.geometry=r(e.geometries,this.geometry);let o=this.geometry.parameters;if(o!==void 0&&o.shapes!==void 0){let l=o.shapes;if(Array.isArray(l))for(let c=0,h=l.length;c<h;c++){let d=l[c];r(e.shapes,d)}else r(e.shapes,l)}}if(this.isSkinnedMesh&&(s.bindMode=this.bindMode,s.bindMatrix=this.bindMatrix.toArray(),this.skeleton!==void 0&&(r(e.skeletons,this.skeleton),s.skeleton=this.skeleton.uuid)),this.material!==void 0)if(Array.isArray(this.material)){let o=[];for(let l=0,c=this.material.length;l<c;l++)o.push(r(e.materials,this.material[l]));s.material=o}else s.material=r(e.materials,this.material);if(this.children.length>0){s.children=[];for(let o=0;o<this.children.length;o++)s.children.push(this.children[o].toJSON(e).object)}if(this.animations.length>0){s.animations=[];for(let o=0;o<this.animations.length;o++){let l=this.animations[o];s.animations.push(r(e.animations,l))}}if(t){let o=a(e.geometries),l=a(e.materials),c=a(e.textures),h=a(e.images),d=a(e.shapes),u=a(e.skeletons),f=a(e.animations),g=a(e.nodes);o.length>0&&(n.geometries=o),l.length>0&&(n.materials=l),c.length>0&&(n.textures=c),h.length>0&&(n.images=h),d.length>0&&(n.shapes=d),u.length>0&&(n.skeletons=u),f.length>0&&(n.animations=f),g.length>0&&(n.nodes=g)}return n.object=s,n;function a(o){let l=[];for(let c in o){let h=o[c];delete h.metadata,l.push(h)}return l}}clone(e){return new this.constructor().copy(this,e)}copy(e,t=!0){if(this.name=e.name,this.up.copy(e.up),this.position.copy(e.position),this.rotation.order=e.rotation.order,this.quaternion.copy(e.quaternion),this.scale.copy(e.scale),this.pivot=e.pivot!==null?e.pivot.clone():null,this.matrix.copy(e.matrix),this.matrixWorld.copy(e.matrixWorld),this.matrixAutoUpdate=e.matrixAutoUpdate,this.matrixWorldAutoUpdate=e.matrixWorldAutoUpdate,this.matrixWorldNeedsUpdate=e.matrixWorldNeedsUpdate,this.layers.mask=e.layers.mask,this.visible=e.visible,this.castShadow=e.castShadow,this.receiveShadow=e.receiveShadow,this.frustumCulled=e.frustumCulled,this.renderOrder=e.renderOrder,this.static=e.static,this.animations=e.animations.slice(),this.userData=JSON.parse(JSON.stringify(e.userData)),t===!0)for(let n=0;n<e.children.length;n++){let s=e.children[n];this.add(s.clone())}return this}dispose(){this.dispatchEvent({type:"dispose"})}};Lt.DEFAULT_UP=new R(0,1,0);Lt.DEFAULT_MATRIX_AUTO_UPDATE=!0;Lt.DEFAULT_MATRIX_WORLD_AUTO_UPDATE=!0;var nn=class extends Lt{constructor(){super(),this.isGroup=!0,this.type="Group"}},Wp={type:"move"},As=class{constructor(){this._targetRay=null,this._grip=null,this._hand=null}getHandSpace(){return this._hand===null&&(this._hand=new nn,this._hand.matrixAutoUpdate=!1,this._hand.visible=!1,this._hand.joints={},this._hand.inputState={pinching:!1}),this._hand}getTargetRaySpace(){return this._targetRay===null&&(this._targetRay=new nn,this._targetRay.matrixAutoUpdate=!1,this._targetRay.visible=!1,this._targetRay.hasLinearVelocity=!1,this._targetRay.linearVelocity=new R,this._targetRay.hasAngularVelocity=!1,this._targetRay.angularVelocity=new R),this._targetRay}getGripSpace(){return this._grip===null&&(this._grip=new nn,this._grip.matrixAutoUpdate=!1,this._grip.visible=!1,this._grip.hasLinearVelocity=!1,this._grip.linearVelocity=new R,this._grip.hasAngularVelocity=!1,this._grip.angularVelocity=new R,this._grip.eventsEnabled=!1),this._grip}dispatchEvent(e){return this._targetRay!==null&&this._targetRay.dispatchEvent(e),this._grip!==null&&this._grip.dispatchEvent(e),this._hand!==null&&this._hand.dispatchEvent(e),this}connect(e){if(e&&e.hand){let t=this._hand;if(t)for(let n of e.hand.values())this._getHandJoint(t,n)}return this.dispatchEvent({type:"connected",data:e}),this}disconnect(e){return this.dispatchEvent({type:"disconnected",data:e}),this._targetRay!==null&&(this._targetRay.visible=!1),this._grip!==null&&(this._grip.visible=!1),this._hand!==null&&(this._hand.visible=!1),this}update(e,t,n){let s=null,r=null,a=null,o=this._targetRay,l=this._grip,c=this._hand;if(e&&t.session.visibilityState!=="visible-blurred"){if(c&&e.hand){a=!0;for(let y of e.hand.values()){let m=t.getJointPose(y,n),p=this._getHandJoint(c,y);m!==null&&(p.matrix.fromArray(m.transform.matrix),p.matrix.decompose(p.position,p.rotation,p.scale),p.matrixWorldNeedsUpdate=!0,p.jointRadius=m.radius),p.visible=m!==null}let h=c.joints["index-finger-tip"],d=c.joints["thumb-tip"],u=h.position.distanceTo(d.position),f=.02,g=.005;c.inputState.pinching&&u>f+g?(c.inputState.pinching=!1,this.dispatchEvent({type:"pinchend",handedness:e.handedness,target:this})):!c.inputState.pinching&&u<=f-g&&(c.inputState.pinching=!0,this.dispatchEvent({type:"pinchstart",handedness:e.handedness,target:this}))}else l!==null&&e.gripSpace&&(r=t.getPose(e.gripSpace,n),r!==null&&(l.matrix.fromArray(r.transform.matrix),l.matrix.decompose(l.position,l.rotation,l.scale),l.matrixWorldNeedsUpdate=!0,r.linearVelocity?(l.hasLinearVelocity=!0,l.linearVelocity.copy(r.linearVelocity)):l.hasLinearVelocity=!1,r.angularVelocity?(l.hasAngularVelocity=!0,l.angularVelocity.copy(r.angularVelocity)):l.hasAngularVelocity=!1,l.eventsEnabled&&l.dispatchEvent({type:"gripUpdated",data:e,target:this})));o!==null&&(s=t.getPose(e.targetRaySpace,n),s===null&&r!==null&&(s=r),s!==null&&(o.matrix.fromArray(s.transform.matrix),o.matrix.decompose(o.position,o.rotation,o.scale),o.matrixWorldNeedsUpdate=!0,s.linearVelocity?(o.hasLinearVelocity=!0,o.linearVelocity.copy(s.linearVelocity)):o.hasLinearVelocity=!1,s.angularVelocity?(o.hasAngularVelocity=!0,o.angularVelocity.copy(s.angularVelocity)):o.hasAngularVelocity=!1,this.dispatchEvent(Wp)))}return o!==null&&(o.visible=s!==null),l!==null&&(l.visible=r!==null),c!==null&&(c.visible=a!==null),this}_getHandJoint(e,t){if(e.joints[t.jointName]===void 0){let n=new nn;n.matrixAutoUpdate=!1,n.visible=!1,e.joints[t.jointName]=n,e.add(n)}return e.joints[t.jointName]}},Cd={aliceblue:15792383,antiquewhite:16444375,aqua:65535,aquamarine:8388564,azure:15794175,beige:16119260,bisque:16770244,black:0,blanchedalmond:16772045,blue:255,blueviolet:9055202,brown:10824234,burlywood:14596231,cadetblue:6266528,chartreuse:8388352,chocolate:13789470,coral:16744272,cornflowerblue:6591981,cornsilk:16775388,crimson:14423100,cyan:65535,darkblue:139,darkcyan:35723,darkgoldenrod:12092939,darkgray:11119017,darkgreen:25600,darkgrey:11119017,darkkhaki:12433259,darkmagenta:9109643,darkolivegreen:5597999,darkorange:16747520,darkorchid:10040012,darkred:9109504,darksalmon:15308410,darkseagreen:9419919,darkslateblue:4734347,darkslategray:3100495,darkslategrey:3100495,darkturquoise:52945,darkviolet:9699539,deeppink:16716947,deepskyblue:49151,dimgray:6908265,dimgrey:6908265,dodgerblue:2003199,firebrick:11674146,floralwhite:16775920,forestgreen:2263842,fuchsia:16711935,gainsboro:14474460,ghostwhite:16316671,gold:16766720,goldenrod:14329120,gray:8421504,green:32768,greenyellow:11403055,grey:8421504,honeydew:15794160,hotpink:16738740,indianred:13458524,indigo:4915330,ivory:16777200,khaki:15787660,lavender:15132410,lavenderblush:16773365,lawngreen:8190976,lemonchiffon:16775885,lightblue:11393254,lightcoral:15761536,lightcyan:14745599,lightgoldenrodyellow:16448210,lightgray:13882323,lightgreen:9498256,lightgrey:13882323,lightpink:16758465,lightsalmon:16752762,lightseagreen:2142890,lightskyblue:8900346,lightslategray:7833753,lightslategrey:7833753,lightsteelblue:11584734,lightyellow:16777184,lime:65280,limegreen:3329330,linen:16445670,magenta:16711935,maroon:8388608,mediumaquamarine:6737322,mediumblue:205,mediumorchid:12211667,mediumpurple:9662683,mediumseagreen:3978097,mediumslateblue:8087790,mediumspringgreen:64154,mediumturquoise:4772300,mediumvioletred:13047173,midnightblue:1644912,mintcream:16121850,mistyrose:16770273,moccasin:16770229,navajowhite:16768685,navy:128,oldlace:16643558,olive:8421376,olivedrab:7048739,orange:16753920,orangered:16729344,orchid:14315734,palegoldenrod:15657130,palegreen:10025880,paleturquoise:11529966,palevioletred:14381203,papayawhip:16773077,peachpuff:16767673,peru:13468991,pink:16761035,plum:14524637,powderblue:11591910,purple:8388736,rebeccapurple:6697881,red:16711680,rosybrown:12357519,royalblue:4286945,saddlebrown:9127187,salmon:16416882,sandybrown:16032864,seagreen:3050327,seashell:16774638,sienna:10506797,silver:12632256,skyblue:8900331,slateblue:6970061,slategray:7372944,slategrey:7372944,snow:16775930,springgreen:65407,steelblue:4620980,tan:13808780,teal:32896,thistle:14204888,tomato:16737095,turquoise:4251856,violet:15631086,wheat:16113331,white:16777215,whitesmoke:16119285,yellow:16776960,yellowgreen:10145074},mi={h:0,s:0,l:0},xa={h:0,s:0,l:0};function Kl(i,e,t){return t<0&&(t+=1),t>1&&(t-=1),t<1/6?i+(e-i)*6*t:t<1/2?e:t<2/3?i+(e-i)*6*(2/3-t):i}var Ye=class{constructor(e,t,n){return this.isColor=!0,this.r=1,this.g=1,this.b=1,this.set(e,t,n)}set(e,t,n){if(t===void 0&&n===void 0){let s=e;s&&s.isColor?this.copy(s):typeof s=="number"?this.setHex(s):typeof s=="string"&&this.setStyle(s)}else this.setRGB(e,t,n);return this}setScalar(e){return this.r=e,this.g=e,this.b=e,this}setHex(e,t=Vt){return e=Math.floor(e),this.r=(e>>16&255)/255,this.g=(e>>8&255)/255,this.b=(e&255)/255,it.colorSpaceToWorking(this,t),this}setRGB(e,t,n,s=it.workingColorSpace){return this.r=e,this.g=t,this.b=n,it.colorSpaceToWorking(this,s),this}setHSL(e,t,n,s=it.workingColorSpace){if(e=eh(e,1),t=je(t,0,1),n=je(n,0,1),t===0)this.r=this.g=this.b=n;else{let r=n<=.5?n*(1+t):n+t-n*t,a=2*n-r;this.r=Kl(a,r,e+1/3),this.g=Kl(a,r,e),this.b=Kl(a,r,e-1/3)}return it.colorSpaceToWorking(this,s),this}setStyle(e,t=Vt){function n(r){r!==void 0&&parseFloat(r)<1&&Oe("Color: Alpha component of "+e+" will be ignored.")}let s;if(s=/^(\w+)\(([^\)]*)\)/.exec(e)){let r,a=s[1],o=s[2];switch(a){case"rgb":case"rgba":if(r=/^\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(r[4]),this.setRGB(Math.min(255,parseInt(r[1],10))/255,Math.min(255,parseInt(r[2],10))/255,Math.min(255,parseInt(r[3],10))/255,t);if(r=/^\s*(\d+)\%\s*,\s*(\d+)\%\s*,\s*(\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(r[4]),this.setRGB(Math.min(100,parseInt(r[1],10))/100,Math.min(100,parseInt(r[2],10))/100,Math.min(100,parseInt(r[3],10))/100,t);break;case"hsl":case"hsla":if(r=/^\s*(\d*\.?\d+)\s*,\s*(\d*\.?\d+)\%\s*,\s*(\d*\.?\d+)\%\s*(?:,\s*(\d*\.?\d+)\s*)?$/.exec(o))return n(r[4]),this.setHSL(parseFloat(r[1])/360,parseFloat(r[2])/100,parseFloat(r[3])/100,t);break;default:Oe("Color: Unknown color model "+e)}}else if(s=/^\#([A-Fa-f\d]+)$/.exec(e)){let r=s[1],a=r.length;if(a===3)return this.setRGB(parseInt(r.charAt(0),16)/15,parseInt(r.charAt(1),16)/15,parseInt(r.charAt(2),16)/15,t);if(a===6)return this.setHex(parseInt(r,16),t);Oe("Color: Invalid hex color "+e)}else if(e&&e.length>0)return this.setColorName(e,t);return this}setColorName(e,t=Vt){let n=Cd[e.toLowerCase()];return n!==void 0?this.setHex(n,t):Oe("Color: Unknown color "+e),this}clone(){return new this.constructor(this.r,this.g,this.b)}copy(e){return this.r=e.r,this.g=e.g,this.b=e.b,this}copySRGBToLinear(e){return this.r=ni(e.r),this.g=ni(e.g),this.b=ni(e.b),this}copyLinearToSRGB(e){return this.r=ys(e.r),this.g=ys(e.g),this.b=ys(e.b),this}convertSRGBToLinear(){return this.copySRGBToLinear(this),this}convertLinearToSRGB(){return this.copyLinearToSRGB(this),this}getHex(e=Vt){return it.workingToColorSpace($t.copy(this),e),Math.round(je($t.r*255,0,255))*65536+Math.round(je($t.g*255,0,255))*256+Math.round(je($t.b*255,0,255))}getHexString(e=Vt){return("000000"+this.getHex(e).toString(16)).slice(-6)}getHSL(e,t=it.workingColorSpace){it.workingToColorSpace($t.copy(this),t);let n=$t.r,s=$t.g,r=$t.b,a=Math.max(n,s,r),o=Math.min(n,s,r),l,c,h=(o+a)/2;if(o===a)l=0,c=0;else{let d=a-o;switch(c=h<=.5?d/(a+o):d/(2-a-o),a){case n:l=(s-r)/d+(s<r?6:0);break;case s:l=(r-n)/d+2;break;case r:l=(n-s)/d+4;break}l/=6}return e.h=l,e.s=c,e.l=h,e}getRGB(e,t=it.workingColorSpace){return it.workingToColorSpace($t.copy(this),t),e.r=$t.r,e.g=$t.g,e.b=$t.b,e}getStyle(e=Vt){it.workingToColorSpace($t.copy(this),e);let t=$t.r,n=$t.g,s=$t.b;return e!==Vt?`color(${e} ${t.toFixed(3)} ${n.toFixed(3)} ${s.toFixed(3)})`:`rgb(${Math.round(t*255)},${Math.round(n*255)},${Math.round(s*255)})`}offsetHSL(e,t,n){return this.getHSL(mi),this.setHSL(mi.h+e,mi.s+t,mi.l+n)}add(e){return this.r+=e.r,this.g+=e.g,this.b+=e.b,this}addColors(e,t){return this.r=e.r+t.r,this.g=e.g+t.g,this.b=e.b+t.b,this}addScalar(e){return this.r+=e,this.g+=e,this.b+=e,this}sub(e){return this.r=Math.max(0,this.r-e.r),this.g=Math.max(0,this.g-e.g),this.b=Math.max(0,this.b-e.b),this}multiply(e){return this.r*=e.r,this.g*=e.g,this.b*=e.b,this}multiplyScalar(e){return this.r*=e,this.g*=e,this.b*=e,this}lerp(e,t){return this.r+=(e.r-this.r)*t,this.g+=(e.g-this.g)*t,this.b+=(e.b-this.b)*t,this}lerpColors(e,t,n){return this.r=e.r+(t.r-e.r)*n,this.g=e.g+(t.g-e.g)*n,this.b=e.b+(t.b-e.b)*n,this}lerpHSL(e,t){this.getHSL(mi),e.getHSL(xa);let n=lr(mi.h,xa.h,t),s=lr(mi.s,xa.s,t),r=lr(mi.l,xa.l,t);return this.setHSL(n,s,r),this}setFromVector3(e){return this.r=e.x,this.g=e.y,this.b=e.z,this}applyMatrix3(e){let t=this.r,n=this.g,s=this.b,r=e.elements;return this.r=r[0]*t+r[3]*n+r[6]*s,this.g=r[1]*t+r[4]*n+r[7]*s,this.b=r[2]*t+r[5]*n+r[8]*s,this}equals(e){return e.r===this.r&&e.g===this.g&&e.b===this.b}fromArray(e,t=0){return this.r=e[t],this.g=e[t+1],this.b=e[t+2],this}toArray(e=[],t=0){return e[t]=this.r,e[t+1]=this.g,e[t+2]=this.b,e}fromBufferAttribute(e,t){return this.r=e.getX(t),this.g=e.getY(t),this.b=e.getZ(t),this}toJSON(){return this.getHex()}*[Symbol.iterator](){yield this.r,yield this.g,yield this.b}},$t=new Ye;Ye.NAMES=Cd;var Hi=class extends Lt{constructor(){super(),this.isScene=!0,this.type="Scene",this.background=null,this.environment=null,this.fog=null,this.backgroundBlurriness=0,this.backgroundIntensity=1,this.backgroundRotation=new Hn,this.environmentIntensity=1,this.environmentRotation=new Hn,this.overrideMaterial=null,typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}copy(e,t){return super.copy(e,t),e.background!==null&&(this.background=e.background.clone()),e.environment!==null&&(this.environment=e.environment.clone()),e.fog!==null&&(this.fog=e.fog.clone()),this.backgroundBlurriness=e.backgroundBlurriness,this.backgroundIntensity=e.backgroundIntensity,this.backgroundRotation.copy(e.backgroundRotation),this.environmentIntensity=e.environmentIntensity,this.environmentRotation.copy(e.environmentRotation),e.overrideMaterial!==null&&(this.overrideMaterial=e.overrideMaterial.clone()),this.matrixAutoUpdate=e.matrixAutoUpdate,this}toJSON(e){let t=super.toJSON(e);return this.fog!==null&&(t.object.fog=this.fog.toJSON()),t.object.backgroundBlurriness=this.backgroundBlurriness,t.object.backgroundIntensity=this.backgroundIntensity,t.object.backgroundRotation=this.backgroundRotation.toArray(),t.object.environmentIntensity=this.environmentIntensity,t.object.environmentRotation=this.environmentRotation.toArray(),t}},Rn=new R,jn=new R,Ql=new R,Kn=new R,os=new R,ls=new R,vu=new R,ec=new R,tc=new R,nc=new R,ic=new St,sc=new St,rc=new St,ti=class i{constructor(e=new R,t=new R,n=new R){this.a=e,this.b=t,this.c=n}static getNormal(e,t,n,s){s.subVectors(n,t),Rn.subVectors(e,t),s.cross(Rn);let r=s.lengthSq();return r>0?s.multiplyScalar(1/Math.sqrt(r)):s.set(0,0,0)}static getBarycoord(e,t,n,s,r){Rn.subVectors(s,t),jn.subVectors(n,t),Ql.subVectors(e,t);let a=Rn.dot(Rn),o=Rn.dot(jn),l=Rn.dot(Ql),c=jn.dot(jn),h=jn.dot(Ql),d=a*c-o*o;if(d===0)return r.set(0,0,0),null;let u=1/d,f=(c*l-o*h)*u,g=(a*h-o*l)*u;return r.set(1-f-g,g,f)}static containsPoint(e,t,n,s){return this.getBarycoord(e,t,n,s,Kn)===null?!1:Kn.x>=0&&Kn.y>=0&&Kn.x+Kn.y<=1}static getInterpolation(e,t,n,s,r,a,o,l){return this.getBarycoord(e,t,n,s,Kn)===null?(l.x=0,l.y=0,"z"in l&&(l.z=0),"w"in l&&(l.w=0),null):(l.setScalar(0),l.addScaledVector(r,Kn.x),l.addScaledVector(a,Kn.y),l.addScaledVector(o,Kn.z),l)}static getInterpolatedAttribute(e,t,n,s,r,a){return ic.setScalar(0),sc.setScalar(0),rc.setScalar(0),ic.fromBufferAttribute(e,t),sc.fromBufferAttribute(e,n),rc.fromBufferAttribute(e,s),a.setScalar(0),a.addScaledVector(ic,r.x),a.addScaledVector(sc,r.y),a.addScaledVector(rc,r.z),a}static isFrontFacing(e,t,n,s){return Rn.subVectors(n,t),jn.subVectors(e,t),Rn.cross(jn).dot(s)<0}set(e,t,n){return this.a.copy(e),this.b.copy(t),this.c.copy(n),this}setFromPointsAndIndices(e,t,n,s){return this.a.copy(e[t]),this.b.copy(e[n]),this.c.copy(e[s]),this}setFromAttributeAndIndices(e,t,n,s){return this.a.fromBufferAttribute(e,t),this.b.fromBufferAttribute(e,n),this.c.fromBufferAttribute(e,s),this}clone(){return new this.constructor().copy(this)}copy(e){return this.a.copy(e.a),this.b.copy(e.b),this.c.copy(e.c),this}getArea(){return Rn.subVectors(this.c,this.b),jn.subVectors(this.a,this.b),Rn.cross(jn).length()*.5}getMidpoint(e){return e.addVectors(this.a,this.b).add(this.c).multiplyScalar(1/3)}getNormal(e){return i.getNormal(this.a,this.b,this.c,e)}getPlane(e){return e.setFromCoplanarPoints(this.a,this.b,this.c)}getBarycoord(e,t){return i.getBarycoord(e,this.a,this.b,this.c,t)}getInterpolation(e,t,n,s,r){return i.getInterpolation(e,this.a,this.b,this.c,t,n,s,r)}containsPoint(e){return i.containsPoint(e,this.a,this.b,this.c)}isFrontFacing(e){return i.isFrontFacing(this.a,this.b,this.c,e)}intersectsBox(e){return e.intersectsTriangle(this)}closestPointToPoint(e,t){let n=this.a,s=this.b,r=this.c,a,o;os.subVectors(s,n),ls.subVectors(r,n),ec.subVectors(e,n);let l=os.dot(ec),c=ls.dot(ec);if(l<=0&&c<=0)return t.copy(n);tc.subVectors(e,s);let h=os.dot(tc),d=ls.dot(tc);if(h>=0&&d<=h)return t.copy(s);let u=l*d-h*c;if(u<=0&&l>=0&&h<=0)return a=l/(l-h),t.copy(n).addScaledVector(os,a);nc.subVectors(e,r);let f=os.dot(nc),g=ls.dot(nc);if(g>=0&&f<=g)return t.copy(r);let y=f*c-l*g;if(y<=0&&c>=0&&g<=0)return o=c/(c-g),t.copy(n).addScaledVector(ls,o);let m=h*g-f*d;if(m<=0&&d-h>=0&&f-g>=0)return vu.subVectors(r,s),o=(d-h)/(d-h+(f-g)),t.copy(s).addScaledVector(vu,o);let p=1/(m+y+u);return a=y*p,o=u*p,t.copy(n).addScaledVector(os,a).addScaledVector(ls,o)}equals(e){return e.a.equals(this.a)&&e.b.equals(this.b)&&e.c.equals(this.c)}},Nt=class{constructor(e=new R(1/0,1/0,1/0),t=new R(-1/0,-1/0,-1/0)){this.isBox3=!0,this.min=e,this.max=t}set(e,t){return this.min.copy(e),this.max.copy(t),this}setFromArray(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t+=3)this.expandByPoint(Cn.fromArray(e,t));return this}setFromBufferAttribute(e){this.makeEmpty();for(let t=0,n=e.count;t<n;t++)this.expandByPoint(Cn.fromBufferAttribute(e,t));return this}setFromPoints(e){this.makeEmpty();for(let t=0,n=e.length;t<n;t++)this.expandByPoint(e[t]);return this}setFromCenterAndSize(e,t){let n=Cn.copy(t).multiplyScalar(.5);return this.min.copy(e).sub(n),this.max.copy(e).add(n),this}setFromObject(e,t=!1){return this.makeEmpty(),this.expandByObject(e,t)}clone(){return new this.constructor().copy(this)}copy(e){return this.min.copy(e.min),this.max.copy(e.max),this}makeEmpty(){return this.min.x=this.min.y=this.min.z=1/0,this.max.x=this.max.y=this.max.z=-1/0,this}isEmpty(){return this.max.x<this.min.x||this.max.y<this.min.y||this.max.z<this.min.z}getCenter(e){return this.isEmpty()?e.set(0,0,0):e.addVectors(this.min,this.max).multiplyScalar(.5)}getSize(e){return this.isEmpty()?e.set(0,0,0):e.subVectors(this.max,this.min)}expandByPoint(e){return this.min.min(e),this.max.max(e),this}expandByVector(e){return this.min.sub(e),this.max.add(e),this}expandByScalar(e){return this.min.addScalar(-e),this.max.addScalar(e),this}expandByObject(e,t=!1){e.updateWorldMatrix(!1,!1);let n=e.geometry;if(n!==void 0){let r=n.getAttribute("position");if(t===!0&&r!==void 0&&e.isInstancedMesh!==!0)for(let a=0,o=r.count;a<o;a++)e.isMesh===!0?e.getVertexPosition(a,Cn):Cn.fromBufferAttribute(r,a),Cn.applyMatrix4(e.matrixWorld),this.expandByPoint(Cn);else e.boundingBox!==void 0?(e.boundingBox===null&&e.computeBoundingBox(),ya.copy(e.boundingBox)):(n.boundingBox===null&&n.computeBoundingBox(),ya.copy(n.boundingBox)),ya.applyMatrix4(e.matrixWorld),this.union(ya)}let s=e.children;for(let r=0,a=s.length;r<a;r++)this.expandByObject(s[r],t);return this}containsPoint(e){return e.x>=this.min.x&&e.x<=this.max.x&&e.y>=this.min.y&&e.y<=this.max.y&&e.z>=this.min.z&&e.z<=this.max.z}containsBox(e){return this.min.x<=e.min.x&&e.max.x<=this.max.x&&this.min.y<=e.min.y&&e.max.y<=this.max.y&&this.min.z<=e.min.z&&e.max.z<=this.max.z}getParameter(e,t){return t.set((e.x-this.min.x)/(this.max.x-this.min.x),(e.y-this.min.y)/(this.max.y-this.min.y),(e.z-this.min.z)/(this.max.z-this.min.z))}intersectsBox(e){return e.max.x>=this.min.x&&e.min.x<=this.max.x&&e.max.y>=this.min.y&&e.min.y<=this.max.y&&e.max.z>=this.min.z&&e.min.z<=this.max.z}intersectsSphere(e){return this.clampPoint(e.center,Cn),Cn.distanceToSquared(e.center)<=e.radius*e.radius}intersectsPlane(e){let t,n;return e.normal.x>0?(t=e.normal.x*this.min.x,n=e.normal.x*this.max.x):(t=e.normal.x*this.max.x,n=e.normal.x*this.min.x),e.normal.y>0?(t+=e.normal.y*this.min.y,n+=e.normal.y*this.max.y):(t+=e.normal.y*this.max.y,n+=e.normal.y*this.min.y),e.normal.z>0?(t+=e.normal.z*this.min.z,n+=e.normal.z*this.max.z):(t+=e.normal.z*this.max.z,n+=e.normal.z*this.min.z),t<=-e.constant&&n>=-e.constant}intersectsTriangle(e){if(this.isEmpty())return!1;this.getCenter(tr),va.subVectors(this.max,tr),cs.subVectors(e.a,tr),hs.subVectors(e.b,tr),us.subVectors(e.c,tr),gi.subVectors(hs,cs),_i.subVectors(us,hs),Fi.subVectors(cs,us);let t=[0,-gi.z,gi.y,0,-_i.z,_i.y,0,-Fi.z,Fi.y,gi.z,0,-gi.x,_i.z,0,-_i.x,Fi.z,0,-Fi.x,-gi.y,gi.x,0,-_i.y,_i.x,0,-Fi.y,Fi.x,0];return!ac(t,cs,hs,us,va)||(t=[1,0,0,0,1,0,0,0,1],!ac(t,cs,hs,us,va))?!1:(Ma.crossVectors(gi,_i),t=[Ma.x,Ma.y,Ma.z],ac(t,cs,hs,us,va))}clampPoint(e,t){return t.copy(e).clamp(this.min,this.max)}distanceToPoint(e){return this.clampPoint(e,Cn).distanceTo(e)}getBoundingSphere(e){return this.isEmpty()?e.makeEmpty():(this.getCenter(e.center),e.radius=this.getSize(Cn).length()*.5),e}intersect(e){return this.min.max(e.min),this.max.min(e.max),this.isEmpty()&&this.makeEmpty(),this}union(e){return this.min.min(e.min),this.max.max(e.max),this}applyMatrix4(e){return this.isEmpty()?this:(Qn[0].set(this.min.x,this.min.y,this.min.z).applyMatrix4(e),Qn[1].set(this.min.x,this.min.y,this.max.z).applyMatrix4(e),Qn[2].set(this.min.x,this.max.y,this.min.z).applyMatrix4(e),Qn[3].set(this.min.x,this.max.y,this.max.z).applyMatrix4(e),Qn[4].set(this.max.x,this.min.y,this.min.z).applyMatrix4(e),Qn[5].set(this.max.x,this.min.y,this.max.z).applyMatrix4(e),Qn[6].set(this.max.x,this.max.y,this.min.z).applyMatrix4(e),Qn[7].set(this.max.x,this.max.y,this.max.z).applyMatrix4(e),this.setFromPoints(Qn),this)}translate(e){return this.min.add(e),this.max.add(e),this}equals(e){return e.min.equals(this.min)&&e.max.equals(this.max)}toJSON(){return{min:this.min.toArray(),max:this.max.toArray()}}fromJSON(e){return this.min.fromArray(e.min),this.max.fromArray(e.max),this}},Qn=[new R,new R,new R,new R,new R,new R,new R,new R],Cn=new R,ya=new Nt,cs=new R,hs=new R,us=new R,gi=new R,_i=new R,Fi=new R,tr=new R,va=new R,Ma=new R,Oi=new R;function ac(i,e,t,n,s){for(let r=0,a=i.length-3;r<=a;r+=3){Oi.fromArray(i,r);let o=s.x*Math.abs(Oi.x)+s.y*Math.abs(Oi.y)+s.z*Math.abs(Oi.z),l=e.dot(Oi),c=t.dot(Oi),h=n.dot(Oi);if(Math.max(-Math.max(l,c,h),Math.min(l,c,h))>o)return!1}return!0}var Pt=new R,ba=new ae,Xp=0,It=class extends In{constructor(e,t,n=!1){if(super(),Array.isArray(e))throw new TypeError("THREE.BufferAttribute: array should be a Typed Array.");this.isBufferAttribute=!0,Object.defineProperty(this,"id",{value:Xp++}),this.name="",this.array=e,this.itemSize=t,this.count=e!==void 0?e.length/t:0,this.normalized=n,this.usage=Ed,this.updateRanges=[],this.gpuType=vn,this.version=0}onUploadCallback(){}set needsUpdate(e){e===!0&&this.version++}setUsage(e){return this.usage=e,this}addUpdateRange(e,t){this.updateRanges.push({start:e,count:t})}clearUpdateRanges(){this.updateRanges.length=0}copy(e){return this.name=e.name,this.array=new e.array.constructor(e.array),this.itemSize=e.itemSize,this.count=e.count,this.normalized=e.normalized,this.usage=e.usage,this.gpuType=e.gpuType,this}copyAt(e,t,n){e*=this.itemSize,n*=t.itemSize;for(let s=0,r=this.itemSize;s<r;s++)this.array[e+s]=t.array[n+s];return this}copyArray(e){return this.array.set(e),this}applyMatrix3(e){if(this.itemSize===2)for(let t=0,n=this.count;t<n;t++)ba.fromBufferAttribute(this,t),ba.applyMatrix3(e),this.setXY(t,ba.x,ba.y);else if(this.itemSize===3)for(let t=0,n=this.count;t<n;t++)Pt.fromBufferAttribute(this,t),Pt.applyMatrix3(e),this.setXYZ(t,Pt.x,Pt.y,Pt.z);return this}applyMatrix4(e){for(let t=0,n=this.count;t<n;t++)Pt.fromBufferAttribute(this,t),Pt.applyMatrix4(e),this.setXYZ(t,Pt.x,Pt.y,Pt.z);return this}applyNormalMatrix(e){for(let t=0,n=this.count;t<n;t++)Pt.fromBufferAttribute(this,t),Pt.applyNormalMatrix(e),this.setXYZ(t,Pt.x,Pt.y,Pt.z);return this}transformDirection(e){for(let t=0,n=this.count;t<n;t++)Pt.fromBufferAttribute(this,t),Pt.transformDirection(e),this.setXYZ(t,Pt.x,Pt.y,Pt.z);return this}set(e,t=0){return this.array.set(e,t),this}getComponent(e,t){let n=this.array[e*this.itemSize+t];return this.normalized&&(n=_s(n,this.array)),n}setComponent(e,t,n){return this.normalized&&(n=Qt(n,this.array)),this.array[e*this.itemSize+t]=n,this}getX(e){let t=this.array[e*this.itemSize];return this.normalized&&(t=_s(t,this.array)),t}setX(e,t){return this.normalized&&(t=Qt(t,this.array)),this.array[e*this.itemSize]=t,this}getY(e){let t=this.array[e*this.itemSize+1];return this.normalized&&(t=_s(t,this.array)),t}setY(e,t){return this.normalized&&(t=Qt(t,this.array)),this.array[e*this.itemSize+1]=t,this}getZ(e){let t=this.array[e*this.itemSize+2];return this.normalized&&(t=_s(t,this.array)),t}setZ(e,t){return this.normalized&&(t=Qt(t,this.array)),this.array[e*this.itemSize+2]=t,this}getW(e){let t=this.array[e*this.itemSize+3];return this.normalized&&(t=_s(t,this.array)),t}setW(e,t){return this.normalized&&(t=Qt(t,this.array)),this.array[e*this.itemSize+3]=t,this}setXY(e,t,n){return e*=this.itemSize,this.normalized&&(t=Qt(t,this.array),n=Qt(n,this.array)),this.array[e+0]=t,this.array[e+1]=n,this}setXYZ(e,t,n,s){return e*=this.itemSize,this.normalized&&(t=Qt(t,this.array),n=Qt(n,this.array),s=Qt(s,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=s,this}setXYZW(e,t,n,s,r){return e*=this.itemSize,this.normalized&&(t=Qt(t,this.array),n=Qt(n,this.array),s=Qt(s,this.array),r=Qt(r,this.array)),this.array[e+0]=t,this.array[e+1]=n,this.array[e+2]=s,this.array[e+3]=r,this}onUpload(e){return this.onUploadCallback=e,this}clone(){return new this.constructor(this.array,this.itemSize).copy(this)}toJSON(){let e={itemSize:this.itemSize,type:this.array.constructor.name,array:Array.from(this.array),normalized:this.normalized};return e.name=this.name,e.usage=this.usage,e.gpuType=this.gpuType,e}dispose(){this.dispatchEvent({type:"dispose"})}};var gr=class extends It{constructor(e,t,n){super(new Uint16Array(e),t,n)}};var _r=class extends It{constructor(e,t,n){super(new Uint32Array(e),t,n)}};var Tt=class extends It{constructor(e,t,n){super(new Float32Array(e),t,n)}},qp=new Nt,nr=new R,oc=new R,yn=class{constructor(e=new R,t=-1){this.isSphere=!0,this.center=e,this.radius=t}set(e,t){return this.center.copy(e),this.radius=t,this}setFromPoints(e,t){let n=this.center;t!==void 0?n.copy(t):qp.setFromPoints(e).getCenter(n);let s=0;for(let r=0,a=e.length;r<a;r++)s=Math.max(s,n.distanceToSquared(e[r]));return this.radius=Math.sqrt(s),this}copy(e){return this.center.copy(e.center),this.radius=e.radius,this}isEmpty(){return this.radius<0}makeEmpty(){return this.center.set(0,0,0),this.radius=-1,this}containsPoint(e){return e.distanceToSquared(this.center)<=this.radius*this.radius}distanceToPoint(e){return e.distanceTo(this.center)-this.radius}intersectsSphere(e){let t=this.radius+e.radius;return e.center.distanceToSquared(this.center)<=t*t}intersectsBox(e){return e.intersectsSphere(this)}intersectsPlane(e){return Math.abs(e.distanceToPoint(this.center))<=this.radius}clampPoint(e,t){let n=this.center.distanceToSquared(e);return t.copy(e),n>this.radius*this.radius&&(t.sub(this.center).normalize(),t.multiplyScalar(this.radius).add(this.center)),t}getBoundingBox(e){return this.isEmpty()?(e.makeEmpty(),e):(e.set(this.center,this.center),e.expandByScalar(this.radius),e)}applyMatrix4(e){return this.center.applyMatrix4(e),this.radius=this.radius*e.getMaxScaleOnAxis(),this}translate(e){return this.center.add(e),this}expandByPoint(e){if(this.isEmpty())return this.center.copy(e),this.radius=0,this;nr.subVectors(e,this.center);let t=nr.lengthSq();if(t>this.radius*this.radius){let n=Math.sqrt(t),s=(n-this.radius)*.5;this.center.addScaledVector(nr,s/n),this.radius+=s}return this}union(e){return e.isEmpty()?this:this.isEmpty()?(this.copy(e),this):(this.center.equals(e.center)===!0?this.radius=Math.max(this.radius,e.radius):(oc.subVectors(e.center,this.center).setLength(e.radius),this.expandByPoint(nr.copy(e.center).add(oc)),this.expandByPoint(nr.copy(e.center).sub(oc))),this)}equals(e){return e.center.equals(this.center)&&e.radius===this.radius}clone(){return new this.constructor().copy(this)}toJSON(){return{radius:this.radius,center:this.center.toArray()}}fromJSON(e){return this.radius=e.radius,this.center.fromArray(e.center),this}},Yp=0,xn=new dt,lc=new Lt,ds=new R,un=new Nt,ir=new Nt,zt=new R,Ut=class i extends In{constructor(){super(),this.isBufferGeometry=!0,Object.defineProperty(this,"id",{value:Yp++}),this.uuid=Yi(),this.name="",this.type="BufferGeometry",this.index=null,this.indirect=null,this.indirectOffset=0,this.attributes={},this.morphAttributes={},this.morphTargetsRelative=!1,this.groups=[],this.boundingBox=null,this.boundingSphere=null,this.drawRange={start:0,count:1/0},this.userData={},this._transformed=!1}getIndex(){return this.index}setIndex(e){return Array.isArray(e)?this.index=new(_p(e)?_r:gr)(e,1):this.index=e,this}setIndirect(e,t=0){return this.indirect=e,this.indirectOffset=t,this}getIndirect(){return this.indirect}getAttribute(e){return this.attributes[e]}setAttribute(e,t){return this.attributes[e]=t,this}deleteAttribute(e){return delete this.attributes[e],this}hasAttribute(e){return this.attributes[e]!==void 0}addGroup(e,t,n=0){this.groups.push({start:e,count:t,materialIndex:n})}clearGroups(){this.groups=[]}setDrawRange(e,t){this.drawRange.start=e,this.drawRange.count=t}applyMatrix4(e){let t=this.attributes.position;t!==void 0&&(t.applyMatrix4(e),t.needsUpdate=!0);let n=this.attributes.normal;if(n!==void 0){let r=new Xe().getNormalMatrix(e);n.applyNormalMatrix(r),n.needsUpdate=!0}let s=this.attributes.tangent;return s!==void 0&&(s.transformDirection(e),s.needsUpdate=!0),this.boundingBox!==null&&this.computeBoundingBox(),this.boundingSphere!==null&&this.computeBoundingSphere(),this._transformed=!0,this}applyQuaternion(e){return xn.makeRotationFromQuaternion(e),this.applyMatrix4(xn),this}rotateX(e){return xn.makeRotationX(e),this.applyMatrix4(xn),this}rotateY(e){return xn.makeRotationY(e),this.applyMatrix4(xn),this}rotateZ(e){return xn.makeRotationZ(e),this.applyMatrix4(xn),this}translate(e,t,n){return xn.makeTranslation(e,t,n),this.applyMatrix4(xn),this}scale(e,t,n){return xn.makeScale(e,t,n),this.applyMatrix4(xn),this}lookAt(e){return lc.lookAt(e),lc.updateMatrix(),this.applyMatrix4(lc.matrix),this}center(){return this.computeBoundingBox(),this.boundingBox.getCenter(ds).negate(),this.translate(ds.x,ds.y,ds.z),this}setFromPoints(e){let t=this.getAttribute("position");if(t===void 0){let n=[];for(let s=0,r=e.length;s<r;s++){let a=e[s];n.push(a.x,a.y,a.z||0)}this.setAttribute("position",new Tt(n,3))}else{let n=Math.min(e.length,t.count);for(let s=0;s<n;s++){let r=e[s];t.setXYZ(s,r.x,r.y,r.z||0)}e.length>t.count&&Oe("BufferGeometry: Buffer size too small for points data. Use .dispose() and create a new geometry."),t.needsUpdate=!0}return this}computeBoundingBox(){this.boundingBox===null&&(this.boundingBox=new Nt);let e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){Ve("BufferGeometry.computeBoundingBox(): GLBufferAttribute requires a manual bounding box.",this),this.boundingBox.set(new R(-1/0,-1/0,-1/0),new R(1/0,1/0,1/0));return}if(e!==void 0){if(this.boundingBox.setFromBufferAttribute(e),t)for(let n=0,s=t.length;n<s;n++){let r=t[n];un.setFromBufferAttribute(r),this.morphTargetsRelative?(zt.addVectors(this.boundingBox.min,un.min),this.boundingBox.expandByPoint(zt),zt.addVectors(this.boundingBox.max,un.max),this.boundingBox.expandByPoint(zt)):(this.boundingBox.expandByPoint(un.min),this.boundingBox.expandByPoint(un.max))}}else this.boundingBox.makeEmpty();(isNaN(this.boundingBox.min.x)||isNaN(this.boundingBox.min.y)||isNaN(this.boundingBox.min.z))&&Ve('BufferGeometry.computeBoundingBox(): Computed min/max have NaN values. The "position" attribute is likely to have NaN values.',this)}computeBoundingSphere(){this.boundingSphere===null&&(this.boundingSphere=new yn);let e=this.attributes.position,t=this.morphAttributes.position;if(e&&e.isGLBufferAttribute){Ve("BufferGeometry.computeBoundingSphere(): GLBufferAttribute requires a manual bounding sphere.",this),this.boundingSphere.set(new R,1/0);return}if(e){let n=this.boundingSphere.center;if(un.setFromBufferAttribute(e),t)for(let r=0,a=t.length;r<a;r++){let o=t[r];ir.setFromBufferAttribute(o),this.morphTargetsRelative?(zt.addVectors(un.min,ir.min),un.expandByPoint(zt),zt.addVectors(un.max,ir.max),un.expandByPoint(zt)):(un.expandByPoint(ir.min),un.expandByPoint(ir.max))}un.getCenter(n);let s=0;for(let r=0,a=e.count;r<a;r++)zt.fromBufferAttribute(e,r),s=Math.max(s,n.distanceToSquared(zt));if(t)for(let r=0,a=t.length;r<a;r++){let o=t[r],l=this.morphTargetsRelative;for(let c=0,h=o.count;c<h;c++)zt.fromBufferAttribute(o,c),l&&(ds.fromBufferAttribute(e,c),zt.add(ds)),s=Math.max(s,n.distanceToSquared(zt))}this.boundingSphere.radius=Math.sqrt(s),isNaN(this.boundingSphere.radius)&&Ve('BufferGeometry.computeBoundingSphere(): Computed radius is NaN. The "position" attribute is likely to have NaN values.',this)}}computeTangents(){let e=this.index,t=this.attributes;if(e===null||t.position===void 0||t.normal===void 0||t.uv===void 0){Ve("BufferGeometry: .computeTangents() failed. Missing required attributes (index, position, normal or uv)");return}let n=t.position,s=t.normal,r=t.uv,a=this.getAttribute("tangent");(a===void 0||a.count!==n.count)&&(a=new It(new Float32Array(4*n.count),4),this.setAttribute("tangent",a));let o=[],l=[];for(let x=0;x<n.count;x++)o[x]=new R,l[x]=new R;let c=new R,h=new R,d=new R,u=new ae,f=new ae,g=new ae,y=new R,m=new R;function p(x,T,I){c.fromBufferAttribute(n,x),h.fromBufferAttribute(n,T),d.fromBufferAttribute(n,I),u.fromBufferAttribute(r,x),f.fromBufferAttribute(r,T),g.fromBufferAttribute(r,I),h.sub(c),d.sub(c),f.sub(u),g.sub(u);let U=1/(f.x*g.y-g.x*f.y);isFinite(U)&&(y.copy(h).multiplyScalar(g.y).addScaledVector(d,-f.y).multiplyScalar(U),m.copy(d).multiplyScalar(f.x).addScaledVector(h,-g.x).multiplyScalar(U),o[x].add(y),o[T].add(y),o[I].add(y),l[x].add(m),l[T].add(m),l[I].add(m))}let w=this.groups;w.length===0&&(w=[{start:0,count:e.count}]);for(let x=0,T=w.length;x<T;++x){let I=w[x],U=I.start,O=I.count;for(let H=U,D=U+O;H<D;H+=3)p(e.getX(H+0),e.getX(H+1),e.getX(H+2))}let A=new R,M=new R,E=new R,S=new R;function C(x){E.fromBufferAttribute(s,x),S.copy(E);let T=o[x];A.copy(T),A.sub(E.multiplyScalar(E.dot(T))).normalize(),M.crossVectors(S,T);let U=M.dot(l[x])<0?-1:1;a.setXYZW(x,A.x,A.y,A.z,U)}for(let x=0,T=w.length;x<T;++x){let I=w[x],U=I.start,O=I.count;for(let H=U,D=U+O;H<D;H+=3)C(e.getX(H+0)),C(e.getX(H+1)),C(e.getX(H+2))}this._transformed=!0}computeVertexNormals(){let e=this.index,t=this.getAttribute("position");if(t!==void 0){let n=this.getAttribute("normal");if(n===void 0||n.count!==t.count)n=new It(new Float32Array(t.count*3),3),this.setAttribute("normal",n);else for(let u=0,f=n.count;u<f;u++)n.setXYZ(u,0,0,0);let s=new R,r=new R,a=new R,o=new R,l=new R,c=new R,h=new R,d=new R;if(e)for(let u=0,f=e.count;u<f;u+=3){let g=e.getX(u+0),y=e.getX(u+1),m=e.getX(u+2);s.fromBufferAttribute(t,g),r.fromBufferAttribute(t,y),a.fromBufferAttribute(t,m),h.subVectors(a,r),d.subVectors(s,r),h.cross(d),o.fromBufferAttribute(n,g),l.fromBufferAttribute(n,y),c.fromBufferAttribute(n,m),o.add(h),l.add(h),c.add(h),n.setXYZ(g,o.x,o.y,o.z),n.setXYZ(y,l.x,l.y,l.z),n.setXYZ(m,c.x,c.y,c.z)}else for(let u=0,f=t.count;u<f;u+=3)s.fromBufferAttribute(t,u+0),r.fromBufferAttribute(t,u+1),a.fromBufferAttribute(t,u+2),h.subVectors(a,r),d.subVectors(s,r),h.cross(d),n.setXYZ(u+0,h.x,h.y,h.z),n.setXYZ(u+1,h.x,h.y,h.z),n.setXYZ(u+2,h.x,h.y,h.z);this.normalizeNormals(),n.needsUpdate=!0}}normalizeNormals(){let e=this.attributes.normal;for(let t=0,n=e.count;t<n;t++)zt.fromBufferAttribute(e,t),zt.normalize(),e.setXYZ(t,zt.x,zt.y,zt.z)}toNonIndexed(){function e(o,l){let c=o.array,h=o.itemSize,d=o.normalized,u=new c.constructor(l.length*h),f=0,g=0;for(let y=0,m=l.length;y<m;y++){o.isInterleavedBufferAttribute?f=l[y]*o.data.stride+o.offset:f=l[y]*h;for(let p=0;p<h;p++)u[g++]=c[f++]}return new It(u,h,d)}if(this.index===null)return Oe("BufferGeometry.toNonIndexed(): BufferGeometry is already non-indexed."),this;let t=new i,n=this.index.array,s=this.attributes;for(let o in s){let l=s[o],c=e(l,n);t.setAttribute(o,c)}let r=this.morphAttributes;for(let o in r){let l=[],c=r[o];for(let h=0,d=c.length;h<d;h++){let u=c[h],f=e(u,n);l.push(f)}t.morphAttributes[o]=l}t.morphTargetsRelative=this.morphTargetsRelative;let a=this.groups;for(let o=0,l=a.length;o<l;o++){let c=a[o];t.addGroup(c.start,c.count,c.materialIndex)}return t}toJSON(){let e={metadata:{version:4.7,type:"BufferGeometry",generator:"BufferGeometry.toJSON"}};if(e.uuid=this.uuid,e.type=this.parameters!==void 0&&this._transformed===!0?"BufferGeometry":this.type,e.name=this.name,Object.keys(this.userData).length>0&&(e.userData=this.userData),this.parameters!==void 0&&this._transformed!==!0){let l=this.parameters;for(let c in l)l[c]!==void 0&&(e[c]=l[c]);return e}e.data={attributes:{}};let t=this.index;t!==null&&(e.data.index={type:t.array.constructor.name,array:Array.prototype.slice.call(t.array)});let n=this.attributes;for(let l in n){let c=n[l];e.data.attributes[l]=c.toJSON(e.data)}let s={},r=!1;for(let l in this.morphAttributes){let c=this.morphAttributes[l],h=[];for(let d=0,u=c.length;d<u;d++){let f=c[d];h.push(f.toJSON(e.data))}h.length>0&&(s[l]=h,r=!0)}r&&(e.data.morphAttributes=s,e.data.morphTargetsRelative=this.morphTargetsRelative);let a=this.groups;a.length>0&&(e.data.groups=JSON.parse(JSON.stringify(a)));let o=this.boundingSphere;return o!==null&&(e.data.boundingSphere=o.toJSON()),e}clone(){return new this.constructor().copy(this)}copy(e){this.index=null,this.attributes={},this.morphAttributes={},this.groups=[],this.boundingBox=null,this.boundingSphere=null;let t={};this.name=e.name;let n=e.index;n!==null&&this.setIndex(n.clone());let s=e.attributes;for(let c in s){let h=s[c];this.setAttribute(c,h.clone(t))}let r=e.morphAttributes;for(let c in r){let h=[],d=r[c];for(let u=0,f=d.length;u<f;u++)h.push(d[u].clone(t));this.morphAttributes[c]=h}this.morphTargetsRelative=e.morphTargetsRelative;let a=e.groups;for(let c=0,h=a.length;c<h;c++){let d=a[c];this.addGroup(d.start,d.count,d.materialIndex)}let o=e.boundingBox;o!==null&&(this.boundingBox=o.clone());let l=e.boundingSphere;return l!==null&&(this.boundingSphere=l.clone()),this.drawRange.start=e.drawRange.start,this.drawRange.count=e.drawRange.count,this.userData=e.userData,this._transformed=e._transformed,this}dispose(){this.dispatchEvent({type:"dispose"})}};var cc=new R,Zp=new R,$p=new Xe,dn=class{constructor(e=new R(1,0,0),t=0){this.isPlane=!0,this.normal=e,this.constant=t}set(e,t){return this.normal.copy(e),this.constant=t,this}setComponents(e,t,n,s){return this.normal.set(e,t,n),this.constant=s,this}setFromNormalAndCoplanarPoint(e,t){return this.normal.copy(e),this.constant=-t.dot(this.normal),this}setFromCoplanarPoints(e,t,n){let s=cc.subVectors(n,t).cross(Zp.subVectors(e,t)).normalize();return this.setFromNormalAndCoplanarPoint(s,e),this}copy(e){return this.normal.copy(e.normal),this.constant=e.constant,this}normalize(){let e=1/this.normal.length();return this.normal.multiplyScalar(e),this.constant*=e,this}negate(){return this.constant*=-1,this.normal.negate(),this}distanceToPoint(e){return this.normal.dot(e)+this.constant}distanceToSphere(e){return this.distanceToPoint(e.center)-e.radius}projectPoint(e,t){return t.copy(e).addScaledVector(this.normal,-this.distanceToPoint(e))}intersectLine(e,t,n=!0){let s=e.delta(cc),r=this.normal.dot(s);if(r===0)return this.distanceToPoint(e.start)===0?t.copy(e.start):null;let a=-(e.start.dot(this.normal)+this.constant)/r;return n===!0&&(a<0||a>1)?null:t.copy(e.start).addScaledVector(s,a)}intersectsLine(e){let t=this.distanceToPoint(e.start),n=this.distanceToPoint(e.end);return t<0&&n>0||n<0&&t>0}intersectsBox(e){return e.intersectsPlane(this)}intersectsSphere(e){return e.intersectsPlane(this)}coplanarPoint(e){return e.copy(this.normal).multiplyScalar(-this.constant)}applyMatrix4(e,t){let n=t||$p.getNormalMatrix(e),s=this.coplanarPoint(cc).applyMatrix4(e),r=this.normal.applyMatrix3(n).normalize();return this.constant=-s.dot(r),this}translate(e){return this.constant-=e.dot(this.normal),this}equals(e){return e.normal.equals(this.normal)&&e.constant===this.constant}clone(){return new this.constructor().copy(this)}toJSON(){return{normal:this.normal.toArray(),constant:this.constant}}fromJSON(e){return this.normal.fromArray(e.normal),this.constant=e.constant,this}},Jp=0,Ln=class extends In{constructor(){super(),this.isMaterial=!0,Object.defineProperty(this,"id",{value:Jp++}),this.uuid=Yi(),this.name="",this.type="Material",this.blending=Bs,this.side=Ai,this.vertexColors=!1,this.opacity=1,this.transparent=!1,this.alphaHash=!1,this.blendSrc=Oc,this.blendDst=Bc,this.blendEquation=Xi,this.blendSrcAlpha=null,this.blendDstAlpha=null,this.blendEquationAlpha=null,this.blendColor=new Ye(0,0,0),this.blendAlpha=0,this.depthFunc=vs,this.depthTest=!0,this.depthWrite=!0,this.stencilWriteMask=255,this.stencilFunc=_d,this.stencilRef=0,this.stencilFuncMask=255,this.stencilFail=qa,this.stencilZFail=qa,this.stencilZPass=qa,this.stencilWrite=!1,this.clippingPlanes=null,this.clipIntersection=!1,this.clipShadows=!1,this.shadowSide=null,this.colorWrite=!0,this.precision=null,this.polygonOffset=!1,this.polygonOffsetFactor=0,this.polygonOffsetUnits=0,this.dithering=!1,this.alphaToCoverage=!1,this.premultipliedAlpha=!1,this.forceSinglePass=!1,this.allowOverride=!0,this.visible=!0,this.toneMapped=!0,this.userData={},this.version=0,this._alphaTest=0}get alphaTest(){return this._alphaTest}set alphaTest(e){this._alphaTest>0!=e>0&&this.version++,this._alphaTest=e}onBeforeRender(){}onBeforeCompile(){}customProgramCacheKey(){return this.onBeforeCompile.toString()}setValues(e){if(e!==void 0)for(let t in e){let n=e[t];if(n===void 0){Oe(`Material: parameter '${t}' has value of undefined.`);continue}let s=this[t];if(s===void 0){Oe(`Material: '${t}' is not a property of THREE.${this.type}.`);continue}s&&s.isColor?s.set(n):s&&s.isVector2&&n&&n.isVector2||s&&s.isEuler&&n&&n.isEuler||s&&s.isVector3&&n&&n.isVector3?s.copy(n):this[t]=n}}toJSON(e){let t=e===void 0||typeof e=="string";t&&(e={textures:{},images:{}});let n={metadata:{version:4.7,type:"Material",generator:"Material.toJSON"}};n.uuid=this.uuid,n.type=this.type,n.blending=this.blending,n.side=this.side,n.shadowSide=this.shadowSide,n.vertexColors=this.vertexColors,n.opacity=this.opacity,n.transparent=this.transparent,n.blendSrc=this.blendSrc,n.blendDst=this.blendDst,n.blendEquation=this.blendEquation,n.blendSrcAlpha=this.blendSrcAlpha,n.blendDstAlpha=this.blendDstAlpha,n.blendEquationAlpha=this.blendEquationAlpha,n.blendColor=this.blendColor.getHex(),n.blendAlpha=this.blendAlpha,n.depthFunc=this.depthFunc,n.depthTest=this.depthTest,n.depthWrite=this.depthWrite,n.colorWrite=this.colorWrite,n.clipIntersection=this.clipIntersection,n.clipShadows=this.clipShadows,n.stencilWriteMask=this.stencilWriteMask,n.stencilFunc=this.stencilFunc,n.stencilRef=this.stencilRef,n.stencilFuncMask=this.stencilFuncMask,n.stencilFail=this.stencilFail,n.stencilZFail=this.stencilZFail,n.stencilZPass=this.stencilZPass,n.stencilWrite=this.stencilWrite,n.polygonOffset=this.polygonOffset,n.polygonOffsetFactor=this.polygonOffsetFactor,n.polygonOffsetUnits=this.polygonOffsetUnits,n.dithering=this.dithering,n.alphaTest=this.alphaTest,n.alphaHash=this.alphaHash,n.alphaToCoverage=this.alphaToCoverage,n.premultipliedAlpha=this.premultipliedAlpha,n.forceSinglePass=this.forceSinglePass,n.allowOverride=this.allowOverride,n.visible=this.visible,n.toneMapped=this.toneMapped,n.name=this.name,this.color&&this.color.isColor&&(n.color=this.color.getHex()),this.roughness!==void 0&&(n.roughness=this.roughness),this.metalness!==void 0&&(n.metalness=this.metalness),this.sheen!==void 0&&(n.sheen=this.sheen),this.sheenColor&&this.sheenColor.isColor&&(n.sheenColor=this.sheenColor.getHex()),this.sheenRoughness!==void 0&&(n.sheenRoughness=this.sheenRoughness),this.emissive&&this.emissive.isColor&&(n.emissive=this.emissive.getHex()),this.emissiveIntensity!==void 0&&(n.emissiveIntensity=this.emissiveIntensity),this.specular&&this.specular.isColor&&(n.specular=this.specular.getHex()),this.specularIntensity!==void 0&&(n.specularIntensity=this.specularIntensity),this.specularColor&&this.specularColor.isColor&&(n.specularColor=this.specularColor.getHex()),this.shininess!==void 0&&(n.shininess=this.shininess),this.clearcoat!==void 0&&(n.clearcoat=this.clearcoat),this.clearcoatRoughness!==void 0&&(n.clearcoatRoughness=this.clearcoatRoughness),this.clearcoatMap&&this.clearcoatMap.isTexture&&(n.clearcoatMap=this.clearcoatMap.toJSON(e).uuid),this.clearcoatRoughnessMap&&this.clearcoatRoughnessMap.isTexture&&(n.clearcoatRoughnessMap=this.clearcoatRoughnessMap.toJSON(e).uuid),this.clearcoatNormalMap&&this.clearcoatNormalMap.isTexture&&(n.clearcoatNormalMap=this.clearcoatNormalMap.toJSON(e).uuid,n.clearcoatNormalScale=this.clearcoatNormalScale.toArray()),this.sheenColorMap&&this.sheenColorMap.isTexture&&(n.sheenColorMap=this.sheenColorMap.toJSON(e).uuid),this.sheenRoughnessMap&&this.sheenRoughnessMap.isTexture&&(n.sheenRoughnessMap=this.sheenRoughnessMap.toJSON(e).uuid),this.dispersion!==void 0&&(n.dispersion=this.dispersion),this.retroreflectivity!==void 0&&(n.retroreflectivity=this.retroreflectivity),this.iridescence!==void 0&&(n.iridescence=this.iridescence),this.iridescenceIOR!==void 0&&(n.iridescenceIOR=this.iridescenceIOR),this.iridescenceThicknessRange!==void 0&&(n.iridescenceThicknessRange=this.iridescenceThicknessRange),this.iridescenceMap&&this.iridescenceMap.isTexture&&(n.iridescenceMap=this.iridescenceMap.toJSON(e).uuid),this.iridescenceThicknessMap&&this.iridescenceThicknessMap.isTexture&&(n.iridescenceThicknessMap=this.iridescenceThicknessMap.toJSON(e).uuid),this.anisotropy!==void 0&&(n.anisotropy=this.anisotropy),this.anisotropyRotation!==void 0&&(n.anisotropyRotation=this.anisotropyRotation),this.anisotropyMap&&this.anisotropyMap.isTexture&&(n.anisotropyMap=this.anisotropyMap.toJSON(e).uuid),this.map&&this.map.isTexture&&(n.map=this.map.toJSON(e).uuid),this.matcap&&this.matcap.isTexture&&(n.matcap=this.matcap.toJSON(e).uuid),this.alphaMap&&this.alphaMap.isTexture&&(n.alphaMap=this.alphaMap.toJSON(e).uuid),this.lightMap&&this.lightMap.isTexture&&(n.lightMap=this.lightMap.toJSON(e).uuid,n.lightMapIntensity=this.lightMapIntensity),this.aoMap&&this.aoMap.isTexture&&(n.aoMap=this.aoMap.toJSON(e).uuid,n.aoMapIntensity=this.aoMapIntensity),this.bumpMap&&this.bumpMap.isTexture&&(n.bumpMap=this.bumpMap.toJSON(e).uuid,n.bumpScale=this.bumpScale),this.normalMap&&this.normalMap.isTexture&&(n.normalMap=this.normalMap.toJSON(e).uuid,n.normalMapType=this.normalMapType,n.normalScale=this.normalScale.toArray()),this.displacementMap&&this.displacementMap.isTexture&&(n.displacementMap=this.displacementMap.toJSON(e).uuid,n.displacementScale=this.displacementScale,n.displacementBias=this.displacementBias),this.roughnessMap&&this.roughnessMap.isTexture&&(n.roughnessMap=this.roughnessMap.toJSON(e).uuid),this.metalnessMap&&this.metalnessMap.isTexture&&(n.metalnessMap=this.metalnessMap.toJSON(e).uuid),this.emissiveMap&&this.emissiveMap.isTexture&&(n.emissiveMap=this.emissiveMap.toJSON(e).uuid),this.specularMap&&this.specularMap.isTexture&&(n.specularMap=this.specularMap.toJSON(e).uuid),this.specularIntensityMap&&this.specularIntensityMap.isTexture&&(n.specularIntensityMap=this.specularIntensityMap.toJSON(e).uuid),this.specularColorMap&&this.specularColorMap.isTexture&&(n.specularColorMap=this.specularColorMap.toJSON(e).uuid),this.envMap&&this.envMap.isTexture&&(n.envMap=this.envMap.toJSON(e).uuid,this.combine!==void 0&&(n.combine=this.combine)),this.envMapRotation!==void 0&&(n.envMapRotation=this.envMapRotation.toArray()),this.envMapIntensity!==void 0&&(n.envMapIntensity=this.envMapIntensity),this.reflectivity!==void 0&&(n.reflectivity=this.reflectivity),this.refractionRatio!==void 0&&(n.refractionRatio=this.refractionRatio),this.gradientMap&&this.gradientMap.isTexture&&(n.gradientMap=this.gradientMap.toJSON(e).uuid),this.transmission!==void 0&&(n.transmission=this.transmission),this.transmissionMap&&this.transmissionMap.isTexture&&(n.transmissionMap=this.transmissionMap.toJSON(e).uuid),this.thickness!==void 0&&(n.thickness=this.thickness),this.thicknessMap&&this.thicknessMap.isTexture&&(n.thicknessMap=this.thicknessMap.toJSON(e).uuid),this.attenuationDistance!==void 0&&(n.attenuationDistance=this.attenuationDistance),this.attenuationColor!==void 0&&(n.attenuationColor=this.attenuationColor.getHex()),this.size!==void 0&&(n.size=this.size),this.sizeAttenuation!==void 0&&(n.sizeAttenuation=this.sizeAttenuation),Array.isArray(this.clippingPlanes)&&this.clippingPlanes.length>0&&(n.clippingPlanes=this.clippingPlanes.map(r=>r.toJSON())),this.rotation!==void 0&&(n.rotation=this.rotation),this.depthPacking!==void 0&&(n.depthPacking=this.depthPacking),this.linewidth!==void 0&&(n.linewidth=this.linewidth),this.linecap!==void 0&&(n.linecap=this.linecap),this.linejoin!==void 0&&(n.linejoin=this.linejoin),this.dashSize!==void 0&&(n.dashSize=this.dashSize),this.gapSize!==void 0&&(n.gapSize=this.gapSize),this.scale!==void 0&&(n.scale=this.scale),this.wireframe!==void 0&&(n.wireframe=this.wireframe),this.wireframeLinewidth!==void 0&&(n.wireframeLinewidth=this.wireframeLinewidth),this.wireframeLinecap!==void 0&&(n.wireframeLinecap=this.wireframeLinecap),this.wireframeLinejoin!==void 0&&(n.wireframeLinejoin=this.wireframeLinejoin),this.flatShading!==void 0&&(n.flatShading=this.flatShading),this.fog!==void 0&&(n.fog=this.fog),Object.keys(this.userData).length>0&&(n.userData=this.userData);function s(r){let a=[];for(let o in r){let l=r[o];delete l.metadata,a.push(l)}return a}if(t){let r=s(e.textures),a=s(e.images);r.length>0&&(n.textures=r),a.length>0&&(n.images=a)}return n}fromJSON(e,t){if(e.uuid!==void 0&&(this.uuid=e.uuid),e.name!==void 0&&(this.name=e.name),e.color!==void 0&&this.color!==void 0&&this.color.setHex(e.color),e.roughness!==void 0&&(this.roughness=e.roughness),e.metalness!==void 0&&(this.metalness=e.metalness),e.sheen!==void 0&&(this.sheen=e.sheen),e.sheenColor!==void 0&&(this.sheenColor=new Ye().setHex(e.sheenColor)),e.sheenRoughness!==void 0&&(this.sheenRoughness=e.sheenRoughness),e.emissive!==void 0&&this.emissive!==void 0&&this.emissive.setHex(e.emissive),e.specular!==void 0&&this.specular!==void 0&&this.specular.setHex(e.specular),e.specularIntensity!==void 0&&(this.specularIntensity=e.specularIntensity),e.specularColor!==void 0&&this.specularColor!==void 0&&this.specularColor.setHex(e.specularColor),e.shininess!==void 0&&(this.shininess=e.shininess),e.clearcoat!==void 0&&(this.clearcoat=e.clearcoat),e.clearcoatRoughness!==void 0&&(this.clearcoatRoughness=e.clearcoatRoughness),e.dispersion!==void 0&&(this.dispersion=e.dispersion),e.retroreflectivity!==void 0&&(this.retroreflectivity=e.retroreflectivity),e.iridescence!==void 0&&(this.iridescence=e.iridescence),e.iridescenceIOR!==void 0&&(this.iridescenceIOR=e.iridescenceIOR),e.iridescenceThicknessRange!==void 0&&(this.iridescenceThicknessRange=e.iridescenceThicknessRange),e.transmission!==void 0&&(this.transmission=e.transmission),e.thickness!==void 0&&(this.thickness=e.thickness),e.attenuationDistance!==void 0&&(this.attenuationDistance=e.attenuationDistance),e.attenuationColor!==void 0&&this.attenuationColor!==void 0&&this.attenuationColor.setHex(e.attenuationColor),e.anisotropy!==void 0&&(this.anisotropy=e.anisotropy),e.anisotropyRotation!==void 0&&(this.anisotropyRotation=e.anisotropyRotation),e.fog!==void 0&&(this.fog=e.fog),e.flatShading!==void 0&&(this.flatShading=e.flatShading),e.blending!==void 0&&(this.blending=e.blending),e.combine!==void 0&&(this.combine=e.combine),e.side!==void 0&&(this.side=e.side),e.shadowSide!==void 0&&(this.shadowSide=e.shadowSide),e.opacity!==void 0&&(this.opacity=e.opacity),e.transparent!==void 0&&(this.transparent=e.transparent),e.alphaTest!==void 0&&(this.alphaTest=e.alphaTest),e.alphaHash!==void 0&&(this.alphaHash=e.alphaHash),e.depthFunc!==void 0&&(this.depthFunc=e.depthFunc),e.depthTest!==void 0&&(this.depthTest=e.depthTest),e.depthWrite!==void 0&&(this.depthWrite=e.depthWrite),e.colorWrite!==void 0&&(this.colorWrite=e.colorWrite),e.clippingPlanes!==void 0&&(this.clippingPlanes=e.clippingPlanes.map(n=>new dn().fromJSON(n))),e.clipIntersection!==void 0&&(this.clipIntersection=e.clipIntersection),e.clipShadows!==void 0&&(this.clipShadows=e.clipShadows),e.depthPacking!==void 0&&(this.depthPacking=e.depthPacking),e.blendSrc!==void 0&&(this.blendSrc=e.blendSrc),e.blendDst!==void 0&&(this.blendDst=e.blendDst),e.blendEquation!==void 0&&(this.blendEquation=e.blendEquation),e.blendSrcAlpha!==void 0&&(this.blendSrcAlpha=e.blendSrcAlpha),e.blendDstAlpha!==void 0&&(this.blendDstAlpha=e.blendDstAlpha),e.blendEquationAlpha!==void 0&&(this.blendEquationAlpha=e.blendEquationAlpha),e.blendColor!==void 0&&this.blendColor!==void 0&&this.blendColor.setHex(e.blendColor),e.blendAlpha!==void 0&&(this.blendAlpha=e.blendAlpha),e.stencilWriteMask!==void 0&&(this.stencilWriteMask=e.stencilWriteMask),e.stencilFunc!==void 0&&(this.stencilFunc=e.stencilFunc),e.stencilRef!==void 0&&(this.stencilRef=e.stencilRef),e.stencilFuncMask!==void 0&&(this.stencilFuncMask=e.stencilFuncMask),e.stencilFail!==void 0&&(this.stencilFail=e.stencilFail),e.stencilZFail!==void 0&&(this.stencilZFail=e.stencilZFail),e.stencilZPass!==void 0&&(this.stencilZPass=e.stencilZPass),e.stencilWrite!==void 0&&(this.stencilWrite=e.stencilWrite),e.wireframe!==void 0&&(this.wireframe=e.wireframe),e.wireframeLinewidth!==void 0&&(this.wireframeLinewidth=e.wireframeLinewidth),e.wireframeLinecap!==void 0&&(this.wireframeLinecap=e.wireframeLinecap),e.wireframeLinejoin!==void 0&&(this.wireframeLinejoin=e.wireframeLinejoin),e.rotation!==void 0&&(this.rotation=e.rotation),e.linewidth!==void 0&&(this.linewidth=e.linewidth),e.linecap!==void 0&&(this.linecap=e.linecap),e.linejoin!==void 0&&(this.linejoin=e.linejoin),e.dashSize!==void 0&&(this.dashSize=e.dashSize),e.gapSize!==void 0&&(this.gapSize=e.gapSize),e.scale!==void 0&&(this.scale=e.scale),e.polygonOffset!==void 0&&(this.polygonOffset=e.polygonOffset),e.polygonOffsetFactor!==void 0&&(this.polygonOffsetFactor=e.polygonOffsetFactor),e.polygonOffsetUnits!==void 0&&(this.polygonOffsetUnits=e.polygonOffsetUnits),e.dithering!==void 0&&(this.dithering=e.dithering),e.alphaToCoverage!==void 0&&(this.alphaToCoverage=e.alphaToCoverage),e.premultipliedAlpha!==void 0&&(this.premultipliedAlpha=e.premultipliedAlpha),e.forceSinglePass!==void 0&&(this.forceSinglePass=e.forceSinglePass),e.allowOverride!==void 0&&(this.allowOverride=e.allowOverride),e.visible!==void 0&&(this.visible=e.visible),e.toneMapped!==void 0&&(this.toneMapped=e.toneMapped),e.userData!==void 0&&(this.userData=e.userData),e.vertexColors!==void 0&&(typeof e.vertexColors=="number"?this.vertexColors=e.vertexColors>0:this.vertexColors=e.vertexColors),e.size!==void 0&&(this.size=e.size),e.sizeAttenuation!==void 0&&(this.sizeAttenuation=e.sizeAttenuation),e.map!==void 0&&(this.map=t[e.map]||null),e.matcap!==void 0&&(this.matcap=t[e.matcap]||null),e.alphaMap!==void 0&&(this.alphaMap=t[e.alphaMap]||null),e.bumpMap!==void 0&&(this.bumpMap=t[e.bumpMap]||null),e.bumpScale!==void 0&&(this.bumpScale=e.bumpScale),e.normalMap!==void 0&&(this.normalMap=t[e.normalMap]||null),e.normalMapType!==void 0&&(this.normalMapType=e.normalMapType),e.normalScale!==void 0){let n=e.normalScale;Array.isArray(n)===!1&&(n=[n,n]),this.normalScale=new ae().fromArray(n)}return e.displacementMap!==void 0&&(this.displacementMap=t[e.displacementMap]||null),e.displacementScale!==void 0&&(this.displacementScale=e.displacementScale),e.displacementBias!==void 0&&(this.displacementBias=e.displacementBias),e.roughnessMap!==void 0&&(this.roughnessMap=t[e.roughnessMap]||null),e.metalnessMap!==void 0&&(this.metalnessMap=t[e.metalnessMap]||null),e.emissiveMap!==void 0&&(this.emissiveMap=t[e.emissiveMap]||null),e.emissiveIntensity!==void 0&&(this.emissiveIntensity=e.emissiveIntensity),e.specularMap!==void 0&&(this.specularMap=t[e.specularMap]||null),e.specularIntensityMap!==void 0&&(this.specularIntensityMap=t[e.specularIntensityMap]||null),e.specularColorMap!==void 0&&(this.specularColorMap=t[e.specularColorMap]||null),e.envMap!==void 0&&(this.envMap=t[e.envMap]||null),e.envMapRotation!==void 0&&this.envMapRotation.fromArray(e.envMapRotation),e.envMapIntensity!==void 0&&(this.envMapIntensity=e.envMapIntensity),e.reflectivity!==void 0&&(this.reflectivity=e.reflectivity),e.refractionRatio!==void 0&&(this.refractionRatio=e.refractionRatio),e.lightMap!==void 0&&(this.lightMap=t[e.lightMap]||null),e.lightMapIntensity!==void 0&&(this.lightMapIntensity=e.lightMapIntensity),e.aoMap!==void 0&&(this.aoMap=t[e.aoMap]||null),e.aoMapIntensity!==void 0&&(this.aoMapIntensity=e.aoMapIntensity),e.gradientMap!==void 0&&(this.gradientMap=t[e.gradientMap]||null),e.clearcoatMap!==void 0&&(this.clearcoatMap=t[e.clearcoatMap]||null),e.clearcoatRoughnessMap!==void 0&&(this.clearcoatRoughnessMap=t[e.clearcoatRoughnessMap]||null),e.clearcoatNormalMap!==void 0&&(this.clearcoatNormalMap=t[e.clearcoatNormalMap]||null),e.clearcoatNormalScale!==void 0&&(this.clearcoatNormalScale=new ae().fromArray(e.clearcoatNormalScale)),e.iridescenceMap!==void 0&&(this.iridescenceMap=t[e.iridescenceMap]||null),e.iridescenceThicknessMap!==void 0&&(this.iridescenceThicknessMap=t[e.iridescenceThicknessMap]||null),e.transmissionMap!==void 0&&(this.transmissionMap=t[e.transmissionMap]||null),e.thicknessMap!==void 0&&(this.thicknessMap=t[e.thicknessMap]||null),e.anisotropyMap!==void 0&&(this.anisotropyMap=t[e.anisotropyMap]||null),e.sheenColorMap!==void 0&&(this.sheenColorMap=t[e.sheenColorMap]||null),e.sheenRoughnessMap!==void 0&&(this.sheenRoughnessMap=t[e.sheenRoughnessMap]||null),this}clone(){return new this.constructor().copy(this)}copy(e){this.name=e.name,this.blending=e.blending,this.side=e.side,this.vertexColors=e.vertexColors,this.opacity=e.opacity,this.transparent=e.transparent,this.blendSrc=e.blendSrc,this.blendDst=e.blendDst,this.blendEquation=e.blendEquation,this.blendSrcAlpha=e.blendSrcAlpha,this.blendDstAlpha=e.blendDstAlpha,this.blendEquationAlpha=e.blendEquationAlpha,this.blendColor.copy(e.blendColor),this.blendAlpha=e.blendAlpha,this.depthFunc=e.depthFunc,this.depthTest=e.depthTest,this.depthWrite=e.depthWrite,this.stencilWriteMask=e.stencilWriteMask,this.stencilFunc=e.stencilFunc,this.stencilRef=e.stencilRef,this.stencilFuncMask=e.stencilFuncMask,this.stencilFail=e.stencilFail,this.stencilZFail=e.stencilZFail,this.stencilZPass=e.stencilZPass,this.stencilWrite=e.stencilWrite;let t=e.clippingPlanes,n=null;if(t!==null){let s=t.length;n=new Array(s);for(let r=0;r!==s;++r)n[r]=t[r].clone()}return this.clippingPlanes=n,this.clipIntersection=e.clipIntersection,this.clipShadows=e.clipShadows,this.shadowSide=e.shadowSide,this.colorWrite=e.colorWrite,this.precision=e.precision,this.polygonOffset=e.polygonOffset,this.polygonOffsetFactor=e.polygonOffsetFactor,this.polygonOffsetUnits=e.polygonOffsetUnits,this.dithering=e.dithering,this.alphaTest=e.alphaTest,this.alphaHash=e.alphaHash,this.alphaToCoverage=e.alphaToCoverage,this.premultipliedAlpha=e.premultipliedAlpha,this.forceSinglePass=e.forceSinglePass,this.allowOverride=e.allowOverride,this.visible=e.visible,this.toneMapped=e.toneMapped,this.userData=JSON.parse(JSON.stringify(e.userData)),this}dispose(){this.dispatchEvent({type:"dispose"})}set needsUpdate(e){e===!0&&this.version++}};var ei=new R,hc=new R,Sa=new R,Ea=new R,ii=class{constructor(e=new R,t=new R(0,0,-1)){this.origin=e,this.direction=t}set(e,t){return this.origin.copy(e),this.direction.copy(t),this}copy(e){return this.origin.copy(e.origin),this.direction.copy(e.direction),this}at(e,t){return t.copy(this.origin).addScaledVector(this.direction,e)}lookAt(e){return this.direction.copy(e).sub(this.origin).normalize(),this}recast(e){return this.origin.copy(this.at(e,ei)),this}closestPointToPoint(e,t){t.subVectors(e,this.origin);let n=t.dot(this.direction);return n<0?t.copy(this.origin):t.copy(this.origin).addScaledVector(this.direction,n)}distanceToPoint(e){return Math.sqrt(this.distanceSqToPoint(e))}distanceSqToPoint(e){let t=ei.subVectors(e,this.origin).dot(this.direction);return t<0?this.origin.distanceToSquared(e):(ei.copy(this.origin).addScaledVector(this.direction,t),ei.distanceToSquared(e))}distanceSqToSegment(e,t,n,s){hc.copy(e).add(t).multiplyScalar(.5),Sa.copy(t).sub(e).normalize(),Ea.copy(this.origin).sub(hc);let r=e.distanceTo(t)*.5,a=-this.direction.dot(Sa),o=Ea.dot(this.direction),l=-Ea.dot(Sa),c=Ea.lengthSq(),h=Math.abs(1-a*a),d,u,f,g;if(h>0)if(d=a*l-o,u=a*o-l,g=r*h,d>=0)if(u>=-g)if(u<=g){let y=1/h;d*=y,u*=y,f=d*(d+a*u+2*o)+u*(a*d+u+2*l)+c}else u=r,d=Math.max(0,-(a*u+o)),f=-d*d+u*(u+2*l)+c;else u=-r,d=Math.max(0,-(a*u+o)),f=-d*d+u*(u+2*l)+c;else u<=-g?(d=Math.max(0,-(-a*r+o)),u=d>0?-r:Math.min(Math.max(-r,-l),r),f=-d*d+u*(u+2*l)+c):u<=g?(d=0,u=Math.min(Math.max(-r,-l),r),f=u*(u+2*l)+c):(d=Math.max(0,-(a*r+o)),u=d>0?r:Math.min(Math.max(-r,-l),r),f=-d*d+u*(u+2*l)+c);else u=a>0?-r:r,d=Math.max(0,-(a*u+o)),f=-d*d+u*(u+2*l)+c;return n&&n.copy(this.origin).addScaledVector(this.direction,d),s&&s.copy(hc).addScaledVector(Sa,u),f}intersectSphere(e,t){if(e.radius<0)return null;ei.subVectors(e.center,this.origin);let n=ei.dot(this.direction),s=ei.dot(ei)-n*n,r=e.radius*e.radius;if(s>r)return null;let a=Math.sqrt(r-s),o=n-a,l=n+a;return l<0?null:o<0?this.at(l,t):this.at(o,t)}intersectsSphere(e){return e.radius<0?!1:this.distanceSqToPoint(e.center)<=e.radius*e.radius}distanceToPlane(e){let t=e.normal.dot(this.direction);if(t===0)return e.distanceToPoint(this.origin)===0?0:null;let n=-(this.origin.dot(e.normal)+e.constant)/t;return n>=0?n:null}intersectPlane(e,t){let n=this.distanceToPlane(e);return n===null?null:this.at(n,t)}intersectsPlane(e){let t=e.distanceToPoint(this.origin);return t===0||e.normal.dot(this.direction)*t<0}intersectBox(e,t){let n,s,r,a,o,l,c=1/this.direction.x,h=1/this.direction.y,d=1/this.direction.z,u=this.origin;return c>=0?(n=(e.min.x-u.x)*c,s=(e.max.x-u.x)*c):(n=(e.max.x-u.x)*c,s=(e.min.x-u.x)*c),h>=0?(r=(e.min.y-u.y)*h,a=(e.max.y-u.y)*h):(r=(e.max.y-u.y)*h,a=(e.min.y-u.y)*h),n>a||r>s||((r>n||isNaN(n))&&(n=r),(a<s||isNaN(s))&&(s=a),d>=0?(o=(e.min.z-u.z)*d,l=(e.max.z-u.z)*d):(o=(e.max.z-u.z)*d,l=(e.min.z-u.z)*d),n>l||o>s)||((o>n||n!==n)&&(n=o),(l<s||s!==s)&&(s=l),s<0)?null:this.at(n>=0?n:s,t)}intersectsBox(e){return this.intersectBox(e,ei)!==null}intersectTriangle(e,t,n,s,r){let a=this.origin,o=this.direction,l=o.x,c=o.y,h=o.z,d=e.x-a.x,u=e.y-a.y,f=e.z-a.z,g=t.x-a.x,y=t.y-a.y,m=t.z-a.z,p=n.x-a.x,w=n.y-a.y,A=n.z-a.z,M=Math.abs(l),E=Math.abs(c),S=Math.abs(h),C,x,T,I,U,O,H,D,B,X,W,ie;if(M>=E&&M>=S?(T=l,O=d,B=g,ie=p,l>=0?(C=c,x=h,I=u,U=f,H=y,D=m,X=w,W=A):(C=h,x=c,I=f,U=u,H=m,D=y,X=A,W=w)):E>=S?(T=c,O=u,B=y,ie=w,c>=0?(C=h,x=l,I=f,U=d,H=m,D=g,X=A,W=p):(C=l,x=h,I=d,U=f,H=g,D=m,X=p,W=A)):(T=h,O=f,B=m,ie=A,h>=0?(C=l,x=c,I=d,U=u,H=g,D=y,X=p,W=w):(C=c,x=l,I=u,U=d,H=y,D=g,X=w,W=p)),T===0)return null;let q=C/T,K=x/T,te=1/T,Ie=I-q*O,we=U-K*O,ot=H-q*B,et=D-K*B,rt=X-q*ie,$=W-K*ie,Q=rt*et-$*ot,xe=Ie*$-we*rt,He=ot*we-et*Ie;if(s){if(Q<0||xe<0||He<0)return null}else if((Q<0||xe<0||He<0)&&(Q>0||xe>0||He>0))return null;let Se=Q+xe+He;if(Se===0)return null;let Ge=te*(Q*O+xe*B+He*ie);return(Se>0?Ge<0:Ge>0)?null:this.at(Ge/Se,r)}applyMatrix4(e){return this.origin.applyMatrix4(e),this.direction.transformDirection(e),this}equals(e){return e.origin.equals(this.origin)&&e.direction.equals(this.direction)}clone(){return new this.constructor().copy(this)}},yi=class extends Ln{constructor(e){super(),this.isMeshBasicMaterial=!0,this.type="MeshBasicMaterial",this.color=new Ye(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Hn,this.combine=Uo,this.reflectivity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.fog=e.fog,this}},Mu=new dt,Bi=new ii,Ta=new yn,bu=new R,wa=new R,Aa=new R,Ra=new R,uc=new R,Ca=new R,Su=new R,Pa=new R,mt=class extends Lt{constructor(e=new Ut,t=new yi){super(),this.isMesh=!0,this.type="Mesh",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.count=1,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),e.morphTargetInfluences!==void 0&&(this.morphTargetInfluences=e.morphTargetInfluences.slice()),e.morphTargetDictionary!==void 0&&(this.morphTargetDictionary=Object.assign({},e.morphTargetDictionary)),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,a=s.length;r<a;r++){let o=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[o]=r}}}}getVertexPosition(e,t){let n=this.geometry,s=n.attributes.position,r=n.morphAttributes.position,a=n.morphTargetsRelative;t.fromBufferAttribute(s,e);let o=this.morphTargetInfluences;if(r&&o){Ca.set(0,0,0);for(let l=0,c=r.length;l<c;l++){let h=o[l],d=r[l];h!==0&&(uc.fromBufferAttribute(d,e),a?Ca.addScaledVector(uc,h):Ca.addScaledVector(uc.sub(t),h))}t.add(Ca)}return t}intersectsFrustum(e){return e.intersectsObject(this)}raycast(e,t){let n=this.geometry,s=this.material,r=this.matrixWorld;s!==void 0&&(n.boundingSphere===null&&n.computeBoundingSphere(),Ta.copy(n.boundingSphere),Ta.applyMatrix4(r),Bi.copy(e.ray).recast(e.near),!(Ta.containsPoint(Bi.origin)===!1&&(Bi.intersectSphere(Ta,bu)===null||Bi.origin.distanceToSquared(bu)>(e.far-e.near)**2))&&(Mu.copy(r).invert(),Bi.copy(e.ray).applyMatrix4(Mu),!(n.boundingBox!==null&&Bi.intersectsBox(n.boundingBox)===!1)&&this._computeIntersections(e,t,Bi)))}_computeIntersections(e,t,n){let s,r=this.geometry,a=this.material,o=r.index,l=r.attributes.position,c=r.attributes.uv,h=r.attributes.uv1,d=r.attributes.normal,u=r.groups,f=r.drawRange;if(o!==null)if(Array.isArray(a))for(let g=0,y=u.length;g<y;g++){let m=u[g],p=a[m.materialIndex],w=Math.max(m.start,f.start),A=Math.min(o.count,Math.min(m.start+m.count,f.start+f.count));for(let M=w,E=A;M<E;M+=3){let S=o.getX(M),C=o.getX(M+1),x=o.getX(M+2);s=Ia(this,p,e,n,c,h,d,S,C,x),s&&(s.faceIndex=Math.floor(M/3),s.face.materialIndex=m.materialIndex,t.push(s))}}else{let g=Math.max(0,f.start),y=Math.min(o.count,f.start+f.count);for(let m=g,p=y;m<p;m+=3){let w=o.getX(m),A=o.getX(m+1),M=o.getX(m+2);s=Ia(this,a,e,n,c,h,d,w,A,M),s&&(s.faceIndex=Math.floor(m/3),t.push(s))}}else if(l!==void 0)if(Array.isArray(a))for(let g=0,y=u.length;g<y;g++){let m=u[g],p=a[m.materialIndex],w=Math.max(m.start,f.start),A=Math.min(l.count,Math.min(m.start+m.count,f.start+f.count));for(let M=w,E=A;M<E;M+=3){let S=M,C=M+1,x=M+2;s=Ia(this,p,e,n,c,h,d,S,C,x),s&&(s.faceIndex=Math.floor(M/3),s.face.materialIndex=m.materialIndex,t.push(s))}}else{let g=Math.max(0,f.start),y=Math.min(l.count,f.start+f.count);for(let m=g,p=y;m<p;m+=3){let w=m,A=m+1,M=m+2;s=Ia(this,a,e,n,c,h,d,w,A,M),s&&(s.faceIndex=Math.floor(m/3),t.push(s))}}}};function jp(i,e,t,n,s,r,a,o){let l;if(e.side===Xt?l=n.intersectTriangle(a,r,s,!0,o):l=n.intersectTriangle(s,r,a,e.side===Ai,o),l===null)return null;Pa.copy(o),Pa.applyMatrix4(i.matrixWorld);let c=t.ray.origin.distanceTo(Pa);return c<t.near||c>t.far?null:{distance:c,point:Pa.clone(),object:i}}function Ia(i,e,t,n,s,r,a,o,l,c){i.getVertexPosition(o,wa),i.getVertexPosition(l,Aa),i.getVertexPosition(c,Ra);let h=jp(i,e,t,n,wa,Aa,Ra,Su);if(h){let d=new R;ti.getBarycoord(Su,wa,Aa,Ra,d),s&&(h.uv=ti.getInterpolatedAttribute(s,o,l,c,d,new ae)),r&&(h.uv1=ti.getInterpolatedAttribute(r,o,l,c,d,new ae)),a&&(h.normal=ti.getInterpolatedAttribute(a,o,l,c,d,new R),h.normal.dot(n.direction)>0&&h.normal.multiplyScalar(-1));let u={a:o,b:l,c,normal:new R,materialIndex:0};ti.getNormal(wa,Aa,Ra,u.normal),h.face=u,h.barycoord=d}return h}var xr=class extends en{constructor(e=null,t=1,n=1,s,r,a,o,l,c=kt,h=kt,d,u){super(null,a,o,l,c,h,s,r,d,u),this.isDataTexture=!0,this.image={data:e,width:t,height:n},this.generateMipmaps=!1,this.flipY=!1,this.unpackAlignment=1}};var yr=class extends It{constructor(e,t,n,s=1){super(e,t,n),this.isInstancedBufferAttribute=!0,this.meshPerAttribute=s}copy(e){return super.copy(e),this.meshPerAttribute=e.meshPerAttribute,this}toJSON(){let e=super.toJSON();return e.meshPerAttribute=this.meshPerAttribute,e.isInstancedBufferAttribute=!0,e}},fs=new dt,Eu=new dt,La=[],Tu=new Nt,Kp=new dt,sr=new mt,rr=new yn,vr=class extends mt{constructor(e,t,n){super(e,t),this.isInstancedMesh=!0,this.instanceMatrix=new yr(new Float32Array(n*16),16),this.instanceColor=null,this.morphTexture=null,this.count=n,this.boundingBox=null,this.boundingSphere=null;for(let s=0;s<n;s++)this.setMatrixAt(s,Kp)}computeBoundingBox(){let e=this.geometry,t=this.count;this.boundingBox===null&&(this.boundingBox=new Nt),e.boundingBox===null&&e.computeBoundingBox(),this.boundingBox.makeEmpty();for(let n=0;n<t;n++)this.getMatrixAt(n,fs),Tu.copy(e.boundingBox).applyMatrix4(fs),this.boundingBox.union(Tu)}computeBoundingSphere(){let e=this.geometry,t=this.count;this.boundingSphere===null&&(this.boundingSphere=new yn),e.boundingSphere===null&&e.computeBoundingSphere(),this.boundingSphere.makeEmpty();for(let n=0;n<t;n++)this.getMatrixAt(n,fs),rr.copy(e.boundingSphere).applyMatrix4(fs),this.boundingSphere.union(rr)}copy(e,t){return super.copy(e,t),this.instanceMatrix.copy(e.instanceMatrix),e.morphTexture!==null&&(this.morphTexture=e.morphTexture.clone()),e.instanceColor!==null&&(this.instanceColor=e.instanceColor.clone()),this.count=e.count,e.boundingBox!==null&&(this.boundingBox=e.boundingBox.clone()),e.boundingSphere!==null&&(this.boundingSphere=e.boundingSphere.clone()),this}getColorAt(e,t){return this.instanceColor===null?t.setRGB(1,1,1):t.fromArray(this.instanceColor.array,e*3)}getMatrixAt(e,t){return t.fromArray(this.instanceMatrix.array,e*16)}getMorphAt(e,t){let n=t.morphTargetInfluences,s=this.morphTexture.source.data.data,r=n.length+1,a=e*r+1;for(let o=0;o<n.length;o++)n[o]=s[a+o]}raycast(e,t){let n=this.matrixWorld,s=this.count;if(sr.geometry=this.geometry,sr.material=this.material,sr.material!==void 0&&(this.boundingSphere===null&&this.computeBoundingSphere(),rr.copy(this.boundingSphere),rr.applyMatrix4(n),e.ray.intersectsSphere(rr)!==!1))for(let r=0;r<s;r++){this.getMatrixAt(r,fs),Eu.multiplyMatrices(n,fs),sr.matrixWorld=Eu,sr.raycast(e,La);for(let a=0,o=La.length;a<o;a++){let l=La[a];l.instanceId=r,l.object=this,t.push(l)}La.length=0}}setColorAt(e,t){return this.instanceColor===null&&(this.instanceColor=new yr(new Float32Array(this.instanceMatrix.count*3).fill(1),3)),t.toArray(this.instanceColor.array,e*3),this}setMatrixAt(e,t){return t.toArray(this.instanceMatrix.array,e*16),this}setMorphAt(e,t){let n=t.morphTargetInfluences,s=n.length+1;this.morphTexture===null&&(this.morphTexture=new xr(new Float32Array(s*this.count),s,this.count,Ho,vn));let r=this.morphTexture.source.data.data,a=0;for(let c=0;c<n.length;c++)a+=n[c];let o=this.geometry.morphTargetsRelative?1:1-a,l=s*e;return r[l]=o,r.set(n,l+1),this}updateMorphTargets(){}dispose(){super.dispose(),this.morphTexture!==null&&(this.morphTexture.dispose(),this.morphTexture=null)}},zi=new yn,Qp=new ae(.5,.5),Da=new R,Rs=class{constructor(e=new dn,t=new dn,n=new dn,s=new dn,r=new dn,a=new dn){this.planes=[e,t,n,s,r,a]}set(e,t,n,s,r,a){let o=this.planes;return o[0].copy(e),o[1].copy(t),o[2].copy(n),o[3].copy(s),o[4].copy(r),o[5].copy(a),this}copy(e){let t=this.planes;for(let n=0;n<6;n++)t[n].copy(e.planes[n]);return this}setFromProjectionMatrix(e,t=Pn,n=!1){let s=this.planes,r=e.elements,a=r[0],o=r[1],l=r[2],c=r[3],h=r[4],d=r[5],u=r[6],f=r[7],g=r[8],y=r[9],m=r[10],p=r[11],w=r[12],A=r[13],M=r[14],E=r[15];if(s[0].setComponents(c-a,f-h,p-g,E-w).normalize(),s[1].setComponents(c+a,f+h,p+g,E+w).normalize(),s[2].setComponents(c+o,f+d,p+y,E+A).normalize(),s[3].setComponents(c-o,f-d,p-y,E-A).normalize(),n)s[4].setComponents(l,u,m,M).normalize(),s[5].setComponents(c-l,f-u,p-m,E-M).normalize();else if(s[4].setComponents(c-l,f-u,p-m,E-M).normalize(),t===Pn)s[5].setComponents(c+l,f+u,p+m,E+M).normalize();else if(t===bs)s[5].setComponents(l,u,m,M).normalize();else throw new Error("THREE.Frustum.setFromProjectionMatrix(): Invalid coordinate system: "+t);return this}intersectsObject(e){if(e.boundingSphere!==void 0)e.boundingSphere===null&&e.computeBoundingSphere(),zi.copy(e.boundingSphere).applyMatrix4(e.matrixWorld);else{let t=e.geometry;t.boundingSphere===null&&t.computeBoundingSphere(),zi.copy(t.boundingSphere).applyMatrix4(e.matrixWorld)}return this.intersectsSphere(zi)}intersectsSprite(e){zi.center.set(0,0,0);let t=Qp.distanceTo(e.center);return zi.radius=.7071067811865476+t,zi.applyMatrix4(e.matrixWorld),this.intersectsSphere(zi)}intersectsSphere(e){let t=this.planes,n=e.center,s=-e.radius;for(let r=0;r<6;r++)if(t[r].distanceToPoint(n)<s)return!1;return!0}intersectsBox(e){let t=this.planes;for(let n=0;n<6;n++){let s=t[n];if(Da.x=s.normal.x>0?e.max.x:e.min.x,Da.y=s.normal.y>0?e.max.y:e.min.y,Da.z=s.normal.z>0?e.max.z:e.min.z,s.distanceToPoint(Da)<0)return!1}return!0}containsPoint(e){let t=this.planes;for(let n=0;n<6;n++)if(t[n].distanceToPoint(e)<0)return!1;return!0}clone(){return new this.constructor().copy(this)}};var Cs=class extends Ln{constructor(e){super(),this.isLineBasicMaterial=!0,this.type="LineBasicMaterial",this.color=new Ye(16777215),this.map=null,this.linewidth=1,this.linecap="round",this.linejoin="round",this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.linewidth=e.linewidth,this.linecap=e.linecap,this.linejoin=e.linejoin,this.fog=e.fog,this}},ro=new R,ao=new R,wu=new dt,ar=new ii,Na=new yn,dc=new R,Au=new R,oo=class extends Lt{constructor(e=new Ut,t=new Cs){super(),this.isLine=!0,this.type="Line",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}computeLineDistances(){let e=this.geometry;if(e.index===null){let t=e.attributes.position,n=[0];for(let s=1,r=t.count;s<r;s++)ro.fromBufferAttribute(t,s-1),ao.fromBufferAttribute(t,s),n[s]=n[s-1],n[s]+=ro.distanceTo(ao);e.setAttribute("lineDistance",new Tt(n,1))}else Oe("Line.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}intersectsFrustum(e){return e.intersectsObject(this)}raycast(e,t){let n=this.geometry,s=this.matrixWorld,r=e.params.Line.threshold,a=n.drawRange;if(n.boundingSphere===null&&n.computeBoundingSphere(),Na.copy(n.boundingSphere),Na.applyMatrix4(s),Na.radius+=r,e.ray.intersectsSphere(Na)===!1)return;wu.copy(s).invert(),ar.copy(e.ray).applyMatrix4(wu);let o=r/((this.scale.x+this.scale.y+this.scale.z)/3),l=o*o,c=this.isLineSegments?2:1,h=n.index,u=n.attributes.position;if(h!==null){let f=Math.max(0,a.start),g=Math.min(h.count,a.start+a.count);for(let y=f,m=g-1;y<m;y+=c){let p=h.getX(y),w=h.getX(y+1),A=Ua(this,e,ar,l,p,w,y);A&&t.push(A)}if(this.isLineLoop){let y=h.getX(g-1),m=h.getX(f),p=Ua(this,e,ar,l,y,m,g-1);p&&t.push(p)}}else{let f=Math.max(0,a.start),g=Math.min(u.count,a.start+a.count);for(let y=f,m=g-1;y<m;y+=c){let p=Ua(this,e,ar,l,y,y+1,y);p&&t.push(p)}if(this.isLineLoop){let y=Ua(this,e,ar,l,g-1,f,g-1);y&&t.push(y)}}}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,a=s.length;r<a;r++){let o=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[o]=r}}}}};function Ua(i,e,t,n,s,r,a){let o=i.geometry.attributes.position;if(ro.fromBufferAttribute(o,s),ao.fromBufferAttribute(o,r),t.distanceSqToSegment(ro,ao,dc,Au)>n)return;dc.applyMatrix4(i.matrixWorld);let c=e.ray.origin.distanceTo(dc);if(!(c<e.near||c>e.far))return{distance:c,point:Au.clone().applyMatrix4(i.matrixWorld),index:a,face:null,faceIndex:null,barycoord:null,object:i}}var Ru=new R,Cu=new R,Mr=class extends oo{constructor(e,t){super(e,t),this.isLineSegments=!0,this.type="LineSegments"}computeLineDistances(){let e=this.geometry;if(e.index===null){let t=e.attributes.position,n=[];for(let s=0,r=t.count;s<r;s+=2)Ru.fromBufferAttribute(t,s),Cu.fromBufferAttribute(t,s+1),n[s]=s===0?0:n[s-1],n[s+1]=n[s]+Ru.distanceTo(Cu);e.setAttribute("lineDistance",new Tt(n,1))}else Oe("LineSegments.computeLineDistances(): Computation only possible with non-indexed BufferGeometry.");return this}};var lo=class extends Ln{constructor(e){super(),this.isPointsMaterial=!0,this.type="PointsMaterial",this.color=new Ye(16777215),this.map=null,this.alphaMap=null,this.size=1,this.sizeAttenuation=!0,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.alphaMap=e.alphaMap,this.size=e.size,this.sizeAttenuation=e.sizeAttenuation,this.fog=e.fog,this}},Pu=new dt,Sc=new ii,Fa=new yn,Oa=new R,br=class extends Lt{constructor(e=new Ut,t=new lo){super(),this.isPoints=!0,this.type="Points",this.geometry=e,this.material=t,this.morphTargetDictionary=void 0,this.morphTargetInfluences=void 0,this.updateMorphTargets()}copy(e,t){return super.copy(e,t),this.material=Array.isArray(e.material)?e.material.slice():e.material,this.geometry=e.geometry,this}intersectsFrustum(e){return e.intersectsObject(this)}raycast(e,t){let n=this.geometry,s=this.matrixWorld,r=e.params.Points.threshold,a=n.drawRange;if(n.boundingSphere===null&&n.computeBoundingSphere(),Fa.copy(n.boundingSphere),Fa.applyMatrix4(s),Fa.radius+=r,e.ray.intersectsSphere(Fa)===!1)return;Pu.copy(s).invert(),Sc.copy(e.ray).applyMatrix4(Pu);let o=r/((this.scale.x+this.scale.y+this.scale.z)/3),l=o*o,c=n.index,d=n.attributes.position;if(c!==null){let u=Math.max(0,a.start),f=Math.min(c.count,a.start+a.count);for(let g=u,y=f;g<y;g++){let m=c.getX(g);Oa.fromBufferAttribute(d,m),Iu(Oa,m,l,s,e,t,this)}}else{let u=Math.max(0,a.start),f=Math.min(d.count,a.start+a.count);for(let g=u,y=f;g<y;g++)Oa.fromBufferAttribute(d,g),Iu(Oa,g,l,s,e,t,this)}}updateMorphTargets(){let t=this.geometry.morphAttributes,n=Object.keys(t);if(n.length>0){let s=t[n[0]];if(s!==void 0){this.morphTargetInfluences=[],this.morphTargetDictionary={};for(let r=0,a=s.length;r<a;r++){let o=s[r].name||String(r);this.morphTargetInfluences.push(0),this.morphTargetDictionary[o]=r}}}}};function Iu(i,e,t,n,s,r,a){let o=Sc.distanceSqToPoint(i);if(o<t){let l=new R;Sc.closestPointToPoint(i,l),l.applyMatrix4(n);let c=s.ray.origin.distanceTo(l);if(c<s.near||c>s.far)return;r.push({distance:c,distanceToRay:Math.sqrt(o),point:l,index:e,face:null,faceIndex:null,barycoord:null,object:a})}}var Sr=class extends en{constructor(e=[],t=Ri,n,s,r,a,o,l,c,h){super(e,t,n,s,r,a,o,l,c,h),this.isCubeTexture=!0,this.flipY=!1}get images(){return this.image}set images(e){this.image=e}},Er=class extends en{constructor(e,t,n,s,r,a,o,l,c){super(e,t,n,s,r,a,o,l,c),this.isCanvasTexture=!0,this.needsUpdate=!0}};var vi=class extends en{constructor(e,t,n=Nn,s,r,a,o=kt,l=kt,c,h=Vn,d=1){if(h!==Vn&&h!==Pi)throw new Error("THREE.DepthTexture: format must be either THREE.DepthFormat or THREE.DepthStencilFormat");let u={width:e,height:t,depth:d};super(u,s,r,a,o,l,h,n,c),this.isDepthTexture=!0,this.flipY=!1,this.generateMipmaps=!1,this.compareFunction=null}copy(e){return super.copy(e),this.source=new Ts(Object.assign({},e.image)),this.compareFunction=e.compareFunction,this}toJSON(e){let t=super.toJSON(e);return t.compareFunction=this.compareFunction,t}},co=class extends vi{constructor(e,t=Nn,n=Ri,s,r,a=kt,o=kt,l,c=Vn){let h={width:e,height:e,depth:1},d=[h,h,h,h,h,h];super(e,e,t,n,s,r,a,o,l,c),this.image=d,this.isCubeDepthTexture=!0,this.isCubeTexture=!0}get images(){return this.image}set images(e){this.image=e}},Tr=class extends en{constructor(e=null){super(),this.sourceTexture=e,this.isExternalTexture=!0}copy(e){return super.copy(e),this.sourceTexture=e.sourceTexture,this}},Gn=class i extends Ut{constructor(e=1,t=1,n=1,s=1,r=1,a=1){super(),this.type="BoxGeometry",this.parameters={width:e,height:t,depth:n,widthSegments:s,heightSegments:r,depthSegments:a};let o=this;s=Math.floor(s),r=Math.floor(r),a=Math.floor(a);let l=[],c=[],h=[],d=[],u=0,f=0;g("z","y","x",-1,-1,n,t,e,a,r,0),g("z","y","x",1,-1,n,t,-e,a,r,1),g("x","z","y",1,1,e,n,t,s,a,2),g("x","z","y",1,-1,e,n,-t,s,a,3),g("x","y","z",1,-1,e,t,n,s,r,4),g("x","y","z",-1,-1,e,t,-n,s,r,5),this.setIndex(l),this.setAttribute("position",new Tt(c,3)),this.setAttribute("normal",new Tt(h,3)),this.setAttribute("uv",new Tt(d,2));function g(y,m,p,w,A,M,E,S,C,x,T){let I=M/C,U=E/x,O=M/2,H=E/2,D=S/2,B=C+1,X=x+1,W=0,ie=0,q=new R;for(let K=0;K<X;K++){let te=K*U-H;for(let Ie=0;Ie<B;Ie++){let we=Ie*I-O;q[y]=we*w,q[m]=te*A,q[p]=D,c.push(q.x,q.y,q.z),q[y]=0,q[m]=0,q[p]=S>0?1:-1,h.push(q.x,q.y,q.z),d.push(Ie/C),d.push(1-K/x),W+=1}}for(let K=0;K<x;K++)for(let te=0;te<C;te++){let Ie=u+te+B*K,we=u+te+B*(K+1),ot=u+(te+1)+B*(K+1),et=u+(te+1)+B*K;l.push(Ie,we,et),l.push(we,ot,et),ie+=6}o.addGroup(f,ie,T),f+=ie,u+=W}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.width,e.height,e.depth,e.widthSegments,e.heightSegments,e.depthSegments)}};var Wt=class i extends Ut{constructor(e=1,t=1,n=1,s=32,r=1,a=!1,o=0,l=Math.PI*2){super(),this.type="CylinderGeometry",this.parameters={radiusTop:e,radiusBottom:t,height:n,radialSegments:s,heightSegments:r,openEnded:a,thetaStart:o,thetaLength:l};let c=this;s=Math.floor(s),r=Math.floor(r);let h=[],d=[],u=[],f=[],g=0,y=[],m=n/2,p=0;w(),a===!1&&(e>0&&A(!0),t>0&&A(!1)),this.setIndex(h),this.setAttribute("position",new Tt(d,3)),this.setAttribute("normal",new Tt(u,3)),this.setAttribute("uv",new Tt(f,2));function w(){let M=new R,E=new R,S=0,C=(t-e)/n;for(let x=0;x<=r;x++){let T=[],I=x/r,U=I*(t-e)+e;for(let O=0;O<=s;O++){let H=O/s,D=H*l+o,B=Math.sin(D),X=Math.cos(D);E.x=U*B,E.y=-I*n+m,E.z=U*X,d.push(E.x,E.y,E.z),M.set(B,C,X).normalize(),u.push(M.x,M.y,M.z),f.push(H,1-I),T.push(g++)}y.push(T)}for(let x=0;x<s;x++)for(let T=0;T<r;T++){let I=y[T][x],U=y[T+1][x],O=y[T+1][x+1],H=y[T][x+1];(e>0||T!==0)&&(h.push(I,U,H),S+=3),(t>0||T!==r-1)&&(h.push(U,O,H),S+=3)}c.addGroup(p,S,0),p+=S}function A(M){let E=g,S=new ae,C=new R,x=0,T=M===!0?e:t,I=M===!0?1:-1;for(let O=1;O<=s;O++)d.push(0,m*I,0),u.push(0,I,0),f.push(.5,.5),g++;let U=g;for(let O=0;O<=s;O++){let D=O/s*l+o,B=Math.cos(D),X=Math.sin(D);C.x=T*X,C.y=m*I,C.z=T*B,d.push(C.x,C.y,C.z),u.push(0,I,0),S.x=B*.5+.5,S.y=X*.5*I+.5,f.push(S.x,S.y),g++}for(let O=0;O<s;O++){let H=E+O,D=U+O;M===!0?h.push(D,D+1,H):h.push(D+1,D,H),x+=3}c.addGroup(p,x,M===!0?1:2),p+=x}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.radiusTop,e.radiusBottom,e.height,e.radialSegments,e.heightSegments,e.openEnded,e.thetaStart,e.thetaLength)}},wr=class i extends Wt{constructor(e=1,t=1,n=32,s=1,r=!1,a=0,o=Math.PI*2){super(0,e,t,n,s,r,a,o),this.type="ConeGeometry",this.parameters={radius:e,height:t,radialSegments:n,heightSegments:s,openEnded:r,thetaStart:a,thetaLength:o}}static fromJSON(e){return new i(e.radius,e.height,e.radialSegments,e.heightSegments,e.openEnded,e.thetaStart,e.thetaLength)}};var Ba=new R,za=new R,fc=new R,ka=new ti,Ar=class extends Ut{constructor(e=null,t=1){if(super(),this.type="EdgesGeometry",this.parameters={geometry:e,thresholdAngle:t},e!==null){let s=Math.pow(10,4),r=Math.cos(xs*t),a=e.getIndex(),o=e.getAttribute("position"),l=a?a.count:o.count,c=[0,0,0],h=["a","b","c"],d=new Array(3),u={},f=[];for(let g=0;g<l;g+=3){a?(c[0]=a.getX(g),c[1]=a.getX(g+1),c[2]=a.getX(g+2)):(c[0]=g,c[1]=g+1,c[2]=g+2);let{a:y,b:m,c:p}=ka;if(y.fromBufferAttribute(o,c[0]),m.fromBufferAttribute(o,c[1]),p.fromBufferAttribute(o,c[2]),ka.getNormal(fc),d[0]=`${Math.round(y.x*s)},${Math.round(y.y*s)},${Math.round(y.z*s)}`,d[1]=`${Math.round(m.x*s)},${Math.round(m.y*s)},${Math.round(m.z*s)}`,d[2]=`${Math.round(p.x*s)},${Math.round(p.y*s)},${Math.round(p.z*s)}`,!(d[0]===d[1]||d[1]===d[2]||d[2]===d[0]))for(let w=0;w<3;w++){let A=(w+1)%3,M=d[w],E=d[A],S=ka[h[w]],C=ka[h[A]],x=`${M}_${E}`,T=`${E}_${M}`;T in u&&u[T]?(fc.dot(u[T].normal)<=r&&(f.push(S.x,S.y,S.z),f.push(C.x,C.y,C.z)),u[T]=null):x in u||(u[x]={index0:c[w],index1:c[A],normal:fc.clone()})}}for(let g in u)if(u[g]){let{index0:y,index1:m}=u[g];Ba.fromBufferAttribute(o,y),za.fromBufferAttribute(o,m),f.push(Ba.x,Ba.y,Ba.z),f.push(za.x,za.y,za.z)}this.setAttribute("position",new Tt(f,3))}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}},pn=class{constructor(){this.type="Curve",this.arcLengthDivisions=200,this.needsUpdate=!1,this.cacheArcLengths=null}getPoint(){Oe("Curve: .getPoint() not implemented.")}getPointAt(e,t){let n=this.getUtoTmapping(e);return this.getPoint(n,t)}getPoints(e=5){let t=[];for(let n=0;n<=e;n++)t.push(this.getPoint(n/e));return t}getSpacedPoints(e=5){let t=[];for(let n=0;n<=e;n++)t.push(this.getPointAt(n/e));return t}getLength(){let e=this.getLengths();return e[e.length-1]}getLengths(e=this.arcLengthDivisions){if(this.cacheArcLengths&&this.cacheArcLengths.length===e+1&&!this.needsUpdate)return this.cacheArcLengths;this.needsUpdate=!1;let t=[],n,s=this.getPoint(0),r=0;t.push(0);for(let a=1;a<=e;a++)n=this.getPoint(a/e),r+=n.distanceTo(s),t.push(r),s=n;return this.cacheArcLengths=t,t}updateArcLengths(){this.needsUpdate=!0,this.getLengths()}getUtoTmapping(e,t=null){let n=this.getLengths(),s=0,r=n.length,a;t?a=t:a=e*n[r-1];let o=0,l=r-1,c;for(;o<=l;)if(s=Math.floor(o+(l-o)/2),c=n[s]-a,c<0)o=s+1;else if(c>0)l=s-1;else{l=s;break}if(s=l,n[s]===a)return s/(r-1);let h=n[s],u=n[s+1]-h,f=(a-h)/u;return(s+f)/(r-1)}getTangent(e,t){let s=e-1e-4,r=e+1e-4;s<0&&(s=0),r>1&&(r=1);let a=this.getPoint(s),o=this.getPoint(r),l=t||(a.isVector2?new ae:new R);return l.copy(o).sub(a).normalize(),l}getTangentAt(e,t){let n=this.getUtoTmapping(e);return this.getTangent(n,t)}computeFrenetFrames(e,t=!1){let n=new R,s=[],r=[],a=[],o=new R,l=new dt;for(let f=0;f<=e;f++){let g=f/e;s[f]=this.getTangentAt(g,new R)}r[0]=new R,a[0]=new R;let c=Number.MAX_VALUE,h=Math.abs(s[0].x),d=Math.abs(s[0].y),u=Math.abs(s[0].z);h<=c&&(c=h,n.set(1,0,0)),d<=c&&(c=d,n.set(0,1,0)),u<=c&&n.set(0,0,1),o.crossVectors(s[0],n).normalize(),r[0].crossVectors(s[0],o),a[0].crossVectors(s[0],r[0]);for(let f=1;f<=e;f++){if(r[f]=r[f-1].clone(),a[f]=a[f-1].clone(),o.crossVectors(s[f-1],s[f]),o.length()>Number.EPSILON){o.normalize();let g=Math.acos(je(s[f-1].dot(s[f]),-1,1));r[f].applyMatrix4(l.makeRotationAxis(o,g))}a[f].crossVectors(s[f],r[f])}if(t===!0){let f=Math.acos(je(r[0].dot(r[e]),-1,1));f/=e,s[0].dot(o.crossVectors(r[0],r[e]))>0&&(f=-f);for(let g=1;g<=e;g++)r[g].applyMatrix4(l.makeRotationAxis(s[g],f*g)),a[g].crossVectors(s[g],r[g])}return{tangents:s,normals:r,binormals:a}}clone(){return new this.constructor().copy(this)}copy(e){return this.arcLengthDivisions=e.arcLengthDivisions,this}toJSON(){let e={metadata:{version:4.7,type:"Curve",generator:"Curve.toJSON"}};return e.arcLengthDivisions=this.arcLengthDivisions,e.type=this.type,e}fromJSON(e){return this.arcLengthDivisions=e.arcLengthDivisions,this}},Ps=class extends pn{constructor(e=0,t=0,n=1,s=1,r=0,a=Math.PI*2,o=!1,l=0){super(),this.isEllipseCurve=!0,this.type="EllipseCurve",this.aX=e,this.aY=t,this.xRadius=n,this.yRadius=s,this.aStartAngle=r,this.aEndAngle=a,this.aClockwise=o,this.aRotation=l}getPoint(e,t=new ae){let n=t,s=Math.PI*2,r=this.aEndAngle-this.aStartAngle,a=Math.abs(r)<Number.EPSILON;for(;r<0;)r+=s;for(;r>s;)r-=s;r<Number.EPSILON&&(a?r=0:r=s),this.aClockwise===!0&&!a&&(r===s?r=-s:r=r-s);let o=this.aStartAngle+e*r,l=this.aX+this.xRadius*Math.cos(o),c=this.aY+this.yRadius*Math.sin(o);if(this.aRotation!==0){let h=Math.cos(this.aRotation),d=Math.sin(this.aRotation),u=l-this.aX,f=c-this.aY;l=u*h-f*d+this.aX,c=u*d+f*h+this.aY}return n.set(l,c)}copy(e){return super.copy(e),this.aX=e.aX,this.aY=e.aY,this.xRadius=e.xRadius,this.yRadius=e.yRadius,this.aStartAngle=e.aStartAngle,this.aEndAngle=e.aEndAngle,this.aClockwise=e.aClockwise,this.aRotation=e.aRotation,this}toJSON(){let e=super.toJSON();return e.aX=this.aX,e.aY=this.aY,e.xRadius=this.xRadius,e.yRadius=this.yRadius,e.aStartAngle=this.aStartAngle,e.aEndAngle=this.aEndAngle,e.aClockwise=this.aClockwise,e.aRotation=this.aRotation,e}fromJSON(e){return super.fromJSON(e),this.aX=e.aX,this.aY=e.aY,this.xRadius=e.xRadius,this.yRadius=e.yRadius,this.aStartAngle=e.aStartAngle,this.aEndAngle=e.aEndAngle,this.aClockwise=e.aClockwise,this.aRotation=e.aRotation,this}},ho=class extends Ps{constructor(e,t,n,s,r,a){super(e,t,n,n,s,r,a),this.isArcCurve=!0,this.type="ArcCurve"}};function th(){let i=0,e=0,t=0,n=0;function s(r,a,o,l){i=r,e=o,t=-3*r+3*a-2*o-l,n=2*r-2*a+o+l}return{initCatmullRom:function(r,a,o,l,c){s(a,o,c*(o-r),c*(l-a))},initNonuniformCatmullRom:function(r,a,o,l,c,h,d){let u=(a-r)/c-(o-r)/(c+h)+(o-a)/h,f=(o-a)/h-(l-a)/(h+d)+(l-o)/d;u*=h,f*=h,s(a,o,u,f)},calc:function(r){let a=r*r,o=a*r;return i+e*r+t*a+n*o}}}var Lu=new R,Du=new R,pc=new th,mc=new th,gc=new th,uo=class extends pn{constructor(e=[],t=!1,n="centripetal",s=.5){super(),this.isCatmullRomCurve3=!0,this.type="CatmullRomCurve3",this.points=e,this.closed=t,this.curveType=n,this.tension=s}getPoint(e,t=new R){let n=t,s=this.points,r=s.length,a=(r-(this.closed?0:1))*e,o=Math.floor(a),l=a-o;this.closed?o+=o>0?0:(Math.floor(Math.abs(o)/r)+1)*r:l===0&&o===r-1&&(o=r-2,l=1);let c,h;this.closed||o>0?c=s[(o-1)%r]:(Du.subVectors(s[0],s[1]).add(s[0]),c=Du);let d=s[o%r],u=s[(o+1)%r];if(this.closed||o+2<r?h=s[(o+2)%r]:(Lu.subVectors(s[r-1],s[r-2]).add(s[r-1]),h=Lu),this.curveType==="centripetal"||this.curveType==="chordal"){let f=this.curveType==="chordal"?.5:.25,g=Math.pow(c.distanceToSquared(d),f),y=Math.pow(d.distanceToSquared(u),f),m=Math.pow(u.distanceToSquared(h),f);y<1e-4&&(y=1),g<1e-4&&(g=y),m<1e-4&&(m=y),pc.initNonuniformCatmullRom(c.x,d.x,u.x,h.x,g,y,m),mc.initNonuniformCatmullRom(c.y,d.y,u.y,h.y,g,y,m),gc.initNonuniformCatmullRom(c.z,d.z,u.z,h.z,g,y,m)}else this.curveType==="catmullrom"&&(pc.initCatmullRom(c.x,d.x,u.x,h.x,this.tension),mc.initCatmullRom(c.y,d.y,u.y,h.y,this.tension),gc.initCatmullRom(c.z,d.z,u.z,h.z,this.tension));return n.set(pc.calc(l),mc.calc(l),gc.calc(l)),n}copy(e){super.copy(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(s.clone())}return this.closed=e.closed,this.curveType=e.curveType,this.tension=e.tension,this}toJSON(){let e=super.toJSON();e.points=[];for(let t=0,n=this.points.length;t<n;t++){let s=this.points[t];e.points.push(s.toArray())}return e.closed=this.closed,e.curveType=this.curveType,e.tension=this.tension,e}fromJSON(e){super.fromJSON(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(new R().fromArray(s))}return this.closed=e.closed,this.curveType=e.curveType,this.tension=e.tension,this}};function Nu(i,e,t,n,s){let r=(n-e)*.5,a=(s-t)*.5,o=i*i,l=i*o;return(2*t-2*n+r+a)*l+(-3*t+3*n-2*r-a)*o+r*i+t}function em(i,e){let t=1-i;return t*t*e}function tm(i,e){return 2*(1-i)*i*e}function nm(i,e){return i*i*e}function cr(i,e,t,n){return em(i,e)+tm(i,t)+nm(i,n)}function im(i,e){let t=1-i;return t*t*t*e}function sm(i,e){let t=1-i;return 3*t*t*i*e}function rm(i,e){return 3*(1-i)*i*i*e}function am(i,e){return i*i*i*e}function hr(i,e,t,n,s){return im(i,e)+sm(i,t)+rm(i,n)+am(i,s)}var Rr=class extends pn{constructor(e=new ae,t=new ae,n=new ae,s=new ae){super(),this.isCubicBezierCurve=!0,this.type="CubicBezierCurve",this.v0=e,this.v1=t,this.v2=n,this.v3=s}getPoint(e,t=new ae){let n=t,s=this.v0,r=this.v1,a=this.v2,o=this.v3;return n.set(hr(e,s.x,r.x,a.x,o.x),hr(e,s.y,r.y,a.y,o.y)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this.v3.copy(e.v3),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e.v3=this.v3.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this.v3.fromArray(e.v3),this}},fo=class extends pn{constructor(e=new R,t=new R,n=new R,s=new R){super(),this.isCubicBezierCurve3=!0,this.type="CubicBezierCurve3",this.v0=e,this.v1=t,this.v2=n,this.v3=s}getPoint(e,t=new R){let n=t,s=this.v0,r=this.v1,a=this.v2,o=this.v3;return n.set(hr(e,s.x,r.x,a.x,o.x),hr(e,s.y,r.y,a.y,o.y),hr(e,s.z,r.z,a.z,o.z)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this.v3.copy(e.v3),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e.v3=this.v3.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this.v3.fromArray(e.v3),this}},Cr=class extends pn{constructor(e=new ae,t=new ae){super(),this.isLineCurve=!0,this.type="LineCurve",this.v1=e,this.v2=t}getPoint(e,t=new ae){let n=t;return e===1?n.copy(this.v2):(n.copy(this.v2).sub(this.v1),n.multiplyScalar(e).add(this.v1)),n}getPointAt(e,t){return this.getPoint(e,t)}getTangent(e,t=new ae){return t.subVectors(this.v2,this.v1).normalize()}getTangentAt(e,t){return this.getTangent(e,t)}copy(e){return super.copy(e),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},po=class extends pn{constructor(e=new R,t=new R){super(),this.isLineCurve3=!0,this.type="LineCurve3",this.v1=e,this.v2=t}getPoint(e,t=new R){let n=t;return e===1?n.copy(this.v2):(n.copy(this.v2).sub(this.v1),n.multiplyScalar(e).add(this.v1)),n}getPointAt(e,t){return this.getPoint(e,t)}getTangent(e,t=new R){return t.subVectors(this.v2,this.v1).normalize()}getTangentAt(e,t){return this.getTangent(e,t)}copy(e){return super.copy(e),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},Pr=class extends pn{constructor(e=new ae,t=new ae,n=new ae){super(),this.isQuadraticBezierCurve=!0,this.type="QuadraticBezierCurve",this.v0=e,this.v1=t,this.v2=n}getPoint(e,t=new ae){let n=t,s=this.v0,r=this.v1,a=this.v2;return n.set(cr(e,s.x,r.x,a.x),cr(e,s.y,r.y,a.y)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},mo=class extends pn{constructor(e=new R,t=new R,n=new R){super(),this.isQuadraticBezierCurve3=!0,this.type="QuadraticBezierCurve3",this.v0=e,this.v1=t,this.v2=n}getPoint(e,t=new R){let n=t,s=this.v0,r=this.v1,a=this.v2;return n.set(cr(e,s.x,r.x,a.x),cr(e,s.y,r.y,a.y),cr(e,s.z,r.z,a.z)),n}copy(e){return super.copy(e),this.v0.copy(e.v0),this.v1.copy(e.v1),this.v2.copy(e.v2),this}toJSON(){let e=super.toJSON();return e.v0=this.v0.toArray(),e.v1=this.v1.toArray(),e.v2=this.v2.toArray(),e}fromJSON(e){return super.fromJSON(e),this.v0.fromArray(e.v0),this.v1.fromArray(e.v1),this.v2.fromArray(e.v2),this}},Ir=class extends pn{constructor(e=[]){super(),this.isSplineCurve=!0,this.type="SplineCurve",this.points=e}getPoint(e,t=new ae){let n=t,s=this.points,r=(s.length-1)*e,a=Math.floor(r),o=r-a,l=s[a===0?a:a-1],c=s[a],h=s[a>s.length-2?s.length-1:a+1],d=s[a>s.length-3?s.length-1:a+2];return n.set(Nu(o,l.x,c.x,h.x,d.x),Nu(o,l.y,c.y,h.y,d.y)),n}copy(e){super.copy(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(s.clone())}return this}toJSON(){let e=super.toJSON();e.points=[];for(let t=0,n=this.points.length;t<n;t++){let s=this.points[t];e.points.push(s.toArray())}return e}fromJSON(e){super.fromJSON(e),this.points=[];for(let t=0,n=e.points.length;t<n;t++){let s=e.points[t];this.points.push(new ae().fromArray(s))}return this}},Ec=Object.freeze({__proto__:null,ArcCurve:ho,CatmullRomCurve3:uo,CubicBezierCurve:Rr,CubicBezierCurve3:fo,EllipseCurve:Ps,LineCurve:Cr,LineCurve3:po,QuadraticBezierCurve:Pr,QuadraticBezierCurve3:mo,SplineCurve:Ir}),go=class extends pn{constructor(){super(),this.type="CurvePath",this.curves=[],this.autoClose=!1}add(e){this.curves.push(e)}closePath(){let e=this.curves[0].getPoint(0),t=this.curves[this.curves.length-1].getPoint(1);if(!e.equals(t)){let n=e.isVector2===!0?"LineCurve":"LineCurve3";this.curves.push(new Ec[n](t,e))}return this}getPoint(e,t){let n=e*this.getLength(),s=this.getCurveLengths(),r=0;for(;r<s.length;){if(s[r]>=n){let a=s[r]-n,o=this.curves[r],l=o.getLength(),c=l===0?0:1-a/l;return o.getPointAt(c,t)}r++}return null}getLength(){let e=this.getCurveLengths();return e[e.length-1]}updateArcLengths(){this.needsUpdate=!0,this.cacheLengths=null,this.getCurveLengths()}getCurveLengths(){if(this.cacheLengths&&this.cacheLengths.length===this.curves.length)return this.cacheLengths;let e=[],t=0;for(let n=0,s=this.curves.length;n<s;n++)t+=this.curves[n].getLength(),e.push(t);return this.cacheLengths=e,e}getSpacedPoints(e=40){let t=[];for(let n=0;n<=e;n++)t.push(this.getPoint(n/e));return this.autoClose&&t.push(t[0]),t}getPoints(e=12){let t=[],n;for(let s=0,r=this.curves;s<r.length;s++){let a=r[s],o=a.isEllipseCurve?e*2:a.isLineCurve||a.isLineCurve3?1:a.isSplineCurve?e*a.points.length:e,l=a.getPoints(o);for(let c=0;c<l.length;c++){let h=l[c];n&&n.equals(h)||(t.push(h),n=h)}}return this.autoClose&&t.length>1&&!t[t.length-1].equals(t[0])&&t.push(t[0]),t}copy(e){super.copy(e),this.curves=[];for(let t=0,n=e.curves.length;t<n;t++){let s=e.curves[t];this.curves.push(s.clone())}return this.autoClose=e.autoClose,this}toJSON(){let e=super.toJSON();e.autoClose=this.autoClose,e.curves=[];for(let t=0,n=this.curves.length;t<n;t++){let s=this.curves[t];e.curves.push(s.toJSON())}return e}fromJSON(e){super.fromJSON(e),this.autoClose=e.autoClose,this.curves=[];for(let t=0,n=e.curves.length;t<n;t++){let s=e.curves[t];this.curves.push(new Ec[s.type]().fromJSON(s))}return this}},Gi=class extends go{constructor(e){super(),this.type="Path",this.currentPoint=new ae,e&&this.setFromPoints(e)}setFromPoints(e){this.moveTo(e[0].x,e[0].y);for(let t=1,n=e.length;t<n;t++)this.lineTo(e[t].x,e[t].y);return this}moveTo(e,t){return this.currentPoint.set(e,t),this}lineTo(e,t){let n=new Cr(this.currentPoint.clone(),new ae(e,t));return this.curves.push(n),this.currentPoint.set(e,t),this}quadraticCurveTo(e,t,n,s){let r=new Pr(this.currentPoint.clone(),new ae(e,t),new ae(n,s));return this.curves.push(r),this.currentPoint.set(n,s),this}bezierCurveTo(e,t,n,s,r,a){let o=new Rr(this.currentPoint.clone(),new ae(e,t),new ae(n,s),new ae(r,a));return this.curves.push(o),this.currentPoint.set(r,a),this}splineThru(e){let t=[this.currentPoint.clone()].concat(e),n=new Ir(t);return this.curves.push(n),this.currentPoint.copy(e[e.length-1]),this}arc(e,t,n,s,r,a){let o=this.currentPoint.x,l=this.currentPoint.y;return this.absarc(e+o,t+l,n,s,r,a),this}absarc(e,t,n,s,r,a){return this.absellipse(e,t,n,n,s,r,a),this}ellipse(e,t,n,s,r,a,o,l){let c=this.currentPoint.x,h=this.currentPoint.y;return this.absellipse(e+c,t+h,n,s,r,a,o,l),this}absellipse(e,t,n,s,r,a,o,l){let c=new Ps(e,t,n,s,r,a,o,l);if(this.curves.length>0){let d=c.getPoint(0);d.equals(this.currentPoint)||this.lineTo(d.x,d.y)}this.curves.push(c);let h=c.getPoint(1);return this.currentPoint.copy(h),this}copy(e){return super.copy(e),this.currentPoint.copy(e.currentPoint),this}toJSON(){let e=super.toJSON();return e.currentPoint=this.currentPoint.toArray(),e}fromJSON(e){return super.fromJSON(e),this.currentPoint.fromArray(e.currentPoint),this}},Is=class extends Gi{constructor(e){super(e),this.uuid=Yi(),this.type="Shape",this.holes=[]}getPointsHoles(e){let t=[];for(let n=0,s=this.holes.length;n<s;n++)t[n]=this.holes[n].getPoints(e);return t}extractPoints(e){return{shape:this.getPoints(e),holes:this.getPointsHoles(e)}}copy(e){super.copy(e),this.holes=[];for(let t=0,n=e.holes.length;t<n;t++){let s=e.holes[t];this.holes.push(s.clone())}return this}toJSON(){let e=super.toJSON();e.uuid=this.uuid,e.holes=[];for(let t=0,n=this.holes.length;t<n;t++){let s=this.holes[t];e.holes.push(s.toJSON())}return e}fromJSON(e){super.fromJSON(e),this.uuid=e.uuid,this.holes=[];for(let t=0,n=e.holes.length;t<n;t++){let s=e.holes[t];this.holes.push(new Gi().fromJSON(s))}return this}};function om(i,e,t=2){let n=e&&e.length,s=n?e[0]*t:i.length,r=Pd(i,0,s,t,!0),a=[];if(!r||r.next===r.prev)return a;let o,l,c;if(n&&(r=dm(i,e,r,t)),i.length>80*t){o=i[0],l=i[1];let h=o,d=l;for(let u=t;u<s;u+=t){let f=i[u],g=i[u+1];f<o&&(o=f),g<l&&(l=g),f>h&&(h=f),g>d&&(d=g)}c=Math.max(h-o,d-l),c=c!==0?32767/c:0}return Lr(r,a,t,o,l,c,0),a}function Pd(i,e,t,n,s){let r;if(s===Sm(i,e,t,n)>0)for(let a=e;a<t;a+=n)r=Uu(a/n|0,i[a],i[a+1],r);else for(let a=t-n;a>=e;a-=n)r=Uu(a/n|0,i[a],i[a+1],r);return r&&Ls(r,r.next)&&(Nr(r),r=r.next),r}function Wi(i,e){if(!i)return i;e||(e=i);let t=i,n;do if(n=!1,!t.steiner&&(Ls(t,t.next)||Et(t.prev,t,t.next)===0)){if(Nr(t),t=e=t.prev,t===t.next)break;n=!0}else t=t.next;while(n||t!==e);return e}function Lr(i,e,t,n,s,r,a){if(!i)return;!a&&r&&_m(i,n,s,r);let o=i;for(;i.prev!==i.next;){let l=i.prev,c=i.next;if(r?cm(i,n,s,r):lm(i)){e.push(l.i,i.i,c.i),Nr(i),i=c.next,o=c.next;continue}if(i=c,i===o){a?a===1?(i=hm(Wi(i),e),Lr(i,e,t,n,s,r,2)):a===2&&um(i,e,t,n,s,r):Lr(Wi(i),e,t,n,s,r,1);break}}}function lm(i){let e=i.prev,t=i,n=i.next;if(Et(e,t,n)>=0)return!1;let s=e.x,r=t.x,a=n.x,o=e.y,l=t.y,c=n.y,h=Math.min(s,r,a),d=Math.min(o,l,c),u=Math.max(s,r,a),f=Math.max(o,l,c),g=n.next;for(;g!==e;){if(g.x>=h&&g.x<=u&&g.y>=d&&g.y<=f&&or(s,o,r,l,a,c,g.x,g.y)&&Et(g.prev,g,g.next)>=0)return!1;g=g.next}return!0}function cm(i,e,t,n){let s=i.prev,r=i,a=i.next;if(Et(s,r,a)>=0)return!1;let o=s.x,l=r.x,c=a.x,h=s.y,d=r.y,u=a.y,f=Math.min(o,l,c),g=Math.min(h,d,u),y=Math.max(o,l,c),m=Math.max(h,d,u),p=Tc(f,g,e,t,n),w=Tc(y,m,e,t,n),A=i.prevZ,M=i.nextZ;for(;A&&A.z>=p&&M&&M.z<=w;){if(A.x>=f&&A.x<=y&&A.y>=g&&A.y<=m&&A!==s&&A!==a&&or(o,h,l,d,c,u,A.x,A.y)&&Et(A.prev,A,A.next)>=0||(A=A.prevZ,M.x>=f&&M.x<=y&&M.y>=g&&M.y<=m&&M!==s&&M!==a&&or(o,h,l,d,c,u,M.x,M.y)&&Et(M.prev,M,M.next)>=0))return!1;M=M.nextZ}for(;A&&A.z>=p;){if(A.x>=f&&A.x<=y&&A.y>=g&&A.y<=m&&A!==s&&A!==a&&or(o,h,l,d,c,u,A.x,A.y)&&Et(A.prev,A,A.next)>=0)return!1;A=A.prevZ}for(;M&&M.z<=w;){if(M.x>=f&&M.x<=y&&M.y>=g&&M.y<=m&&M!==s&&M!==a&&or(o,h,l,d,c,u,M.x,M.y)&&Et(M.prev,M,M.next)>=0)return!1;M=M.nextZ}return!0}function hm(i,e){let t=i;do{let n=t.prev,s=t.next.next;!Ls(n,s)&&Ld(n,t,t.next,s)&&Dr(n,s)&&Dr(s,n)&&(e.push(n.i,t.i,s.i),Nr(t),Nr(t.next),t=i=s),t=t.next}while(t!==i);return Wi(t)}function um(i,e,t,n,s,r){let a=i;do{let o=a.next.next;for(;o!==a.prev;){if(a.i!==o.i&&vm(a,o)){let l=Dd(a,o);a=Wi(a,a.next),l=Wi(l,l.next),Lr(a,e,t,n,s,r,0),Lr(l,e,t,n,s,r,0);return}o=o.next}a=a.next}while(a!==i)}function dm(i,e,t,n){let s=[];for(let r=0,a=e.length;r<a;r++){let o=e[r]*n,l=r<a-1?e[r+1]*n:i.length,c=Pd(i,o,l,n,!1);c===c.next&&(c.steiner=!0),s.push(ym(c))}s.sort(fm);for(let r=0;r<s.length;r++)t=pm(s[r],t);return t}function fm(i,e){let t=i.x-e.x;if(t===0&&(t=i.y-e.y,t===0)){let n=(i.next.y-i.y)/(i.next.x-i.x),s=(e.next.y-e.y)/(e.next.x-e.x);t=n-s}return t}function pm(i,e){let t=mm(i,e);if(!t)return e;let n=Dd(t,i);return Wi(n,n.next),Wi(t,t.next)}function mm(i,e){let t=e,n=i.x,s=i.y,r=-1/0,a;if(Ls(i,t))return t;do{if(Ls(i,t.next))return t.next;if(s<=t.y&&s>=t.next.y&&t.next.y!==t.y){let d=t.x+(s-t.y)*(t.next.x-t.x)/(t.next.y-t.y);if(d<=n&&d>r&&(r=d,a=t.x<t.next.x?t:t.next,d===n))return a}t=t.next}while(t!==e);if(!a)return null;let o=a,l=a.x,c=a.y,h=1/0;t=a;do{if(n>=t.x&&t.x>=l&&n!==t.x&&Id(s<c?n:r,s,l,c,s<c?r:n,s,t.x,t.y)){let d=Math.abs(s-t.y)/(n-t.x);Dr(t,i)&&(d<h||d===h&&(t.x>a.x||t.x===a.x&&gm(a,t)))&&(a=t,h=d)}t=t.next}while(t!==o);return a}function gm(i,e){return Et(i.prev,i,e.prev)<0&&Et(e.next,i,i.next)<0}function _m(i,e,t,n){let s=i;do s.z===0&&(s.z=Tc(s.x,s.y,e,t,n)),s.prevZ=s.prev,s.nextZ=s.next,s=s.next;while(s!==i);s.prevZ.nextZ=null,s.prevZ=null,xm(s)}function xm(i){let e,t=1;do{let n=i,s;i=null;let r=null;for(e=0;n;){e++;let a=n,o=0;for(let c=0;c<t&&(o++,a=a.nextZ,!!a);c++);let l=t;for(;o>0||l>0&&a;)o!==0&&(l===0||!a||n.z<=a.z)?(s=n,n=n.nextZ,o--):(s=a,a=a.nextZ,l--),r?r.nextZ=s:i=s,s.prevZ=r,r=s;n=a}r.nextZ=null,t*=2}while(e>1);return i}function Tc(i,e,t,n,s){return i=(i-t)*s|0,e=(e-n)*s|0,i=(i|i<<8)&16711935,i=(i|i<<4)&252645135,i=(i|i<<2)&858993459,i=(i|i<<1)&1431655765,e=(e|e<<8)&16711935,e=(e|e<<4)&252645135,e=(e|e<<2)&858993459,e=(e|e<<1)&1431655765,i|e<<1}function ym(i){let e=i,t=i;do(e.x<t.x||e.x===t.x&&e.y<t.y)&&(t=e),e=e.next;while(e!==i);return t}function Id(i,e,t,n,s,r,a,o){return(s-a)*(e-o)>=(i-a)*(r-o)&&(i-a)*(n-o)>=(t-a)*(e-o)&&(t-a)*(r-o)>=(s-a)*(n-o)}function or(i,e,t,n,s,r,a,o){return!(i===a&&e===o)&&Id(i,e,t,n,s,r,a,o)}function vm(i,e){return i.next.i!==e.i&&i.prev.i!==e.i&&!Mm(i,e)&&(Dr(i,e)&&Dr(e,i)&&bm(i,e)&&(Et(i.prev,i,e.prev)||Et(i,e.prev,e))||Ls(i,e)&&Et(i.prev,i,i.next)>0&&Et(e.prev,e,e.next)>0)}function Et(i,e,t){return(e.y-i.y)*(t.x-e.x)-(e.x-i.x)*(t.y-e.y)}function Ls(i,e){return i.x===e.x&&i.y===e.y}function Ld(i,e,t,n){let s=Ha(Et(i,e,t)),r=Ha(Et(i,e,n)),a=Ha(Et(t,n,i)),o=Ha(Et(t,n,e));return!!(s!==r&&a!==o||s===0&&Va(i,t,e)||r===0&&Va(i,n,e)||a===0&&Va(t,i,n)||o===0&&Va(t,e,n))}function Va(i,e,t){return e.x<=Math.max(i.x,t.x)&&e.x>=Math.min(i.x,t.x)&&e.y<=Math.max(i.y,t.y)&&e.y>=Math.min(i.y,t.y)}function Ha(i){return i>0?1:i<0?-1:0}function Mm(i,e){let t=i;do{if(t.i!==i.i&&t.next.i!==i.i&&t.i!==e.i&&t.next.i!==e.i&&Ld(t,t.next,i,e))return!0;t=t.next}while(t!==i);return!1}function Dr(i,e){return Et(i.prev,i,i.next)<0?Et(i,e,i.next)>=0&&Et(i,i.prev,e)>=0:Et(i,e,i.prev)<0||Et(i,i.next,e)<0}function bm(i,e){let t=i,n=!1,s=(i.x+e.x)/2,r=(i.y+e.y)/2;do t.y>r!=t.next.y>r&&t.next.y!==t.y&&s<(t.next.x-t.x)*(r-t.y)/(t.next.y-t.y)+t.x&&(n=!n),t=t.next;while(t!==i);return n}function Dd(i,e){let t=wc(i.i,i.x,i.y),n=wc(e.i,e.x,e.y),s=i.next,r=e.prev;return i.next=e,e.prev=i,t.next=s,s.prev=t,n.next=t,t.prev=n,r.next=n,n.prev=r,n}function Uu(i,e,t,n){let s=wc(i,e,t);return n?(s.next=n.next,s.prev=n,n.next.prev=s,n.next=s):(s.prev=s,s.next=s),s}function Nr(i){i.next.prev=i.prev,i.prev.next=i.next,i.prevZ&&(i.prevZ.nextZ=i.nextZ),i.nextZ&&(i.nextZ.prevZ=i.prevZ)}function wc(i,e,t){return{i,x:e,y:t,prev:null,next:null,z:0,prevZ:null,nextZ:null,steiner:!1}}function Sm(i,e,t,n){let s=0;for(let r=e,a=t-n;r<t;r+=n)s+=(i[a]-i[r])*(i[r+1]+i[a+1]),a=r;return s}var Ac=class{static triangulate(e,t,n=2){return om(e,t,n)}},ki=class i{static area(e){let t=e.length,n=0;for(let s=t-1,r=0;r<t;s=r++)n+=e[s].x*e[r].y-e[r].x*e[s].y;return n*.5}static isClockWise(e){return i.area(e)<0}static triangulateShape(e,t){let n=[],s=[],r=[];Fu(e),Ou(n,e);let a=e.length;t.forEach(Fu);for(let l=0;l<t.length;l++)s.push(a),a+=t[l].length,Ou(n,t[l]);let o=Ac.triangulate(n,s);for(let l=0;l<o.length;l+=3)r.push(o.slice(l,l+3));return r}};function Fu(i){let e=i.length;e>2&&i[e-1].equals(i[0])&&i.pop()}function Ou(i,e){for(let t=0;t<e.length;t++)i.push(e[t].x),i.push(e[t].y)}var Ur=class i extends Ut{constructor(e=new Is([new ae(.5,.5),new ae(-.5,.5),new ae(-.5,-.5),new ae(.5,-.5)]),t={}){super(),this.type="ExtrudeGeometry",this.parameters={shapes:e,options:t},e=Array.isArray(e)?e:[e];let n=this,s=[],r=[];for(let o=0,l=e.length;o<l;o++){let c=e[o];a(c)}this.setAttribute("position",new Tt(s,3)),this.setAttribute("uv",new Tt(r,2)),this.computeVertexNormals();function a(o){let l=[],c=t.curveSegments!==void 0?t.curveSegments:12,h=t.steps!==void 0?t.steps:1,d=t.depth!==void 0?t.depth:1,u=t.bevelEnabled!==void 0?t.bevelEnabled:!0,f=t.bevelThickness!==void 0?t.bevelThickness:.2,g=t.bevelSize!==void 0?t.bevelSize:f-.1,y=t.bevelOffset!==void 0?t.bevelOffset:0,m=t.bevelSegments!==void 0?t.bevelSegments:3,p=t.extrudePath,w=t.UVGenerator!==void 0?t.UVGenerator:Em,A,M=!1,E,S,C,x;if(p){A=p.getSpacedPoints(h),M=!0,u=!1;let ee=p.isCatmullRomCurve3?p.closed:!1;E=p.computeFrenetFrames(h,ee),S=new R,C=new R,x=new R}u||(m=0,f=0,g=0,y=0);let T=o.extractPoints(c),I=T.shape,U=T.holes;if(!ki.isClockWise(I)){I=I.reverse();for(let ee=0,se=U.length;ee<se;ee++){let oe=U[ee];ki.isClockWise(oe)&&(U[ee]=oe.reverse())}}function H(ee){let oe=10000000000000001e-36,le=ee[0];for(let ue=1;ue<=ee.length;ue++){let ze=ue%ee.length,Ue=ee[ze],We=Ue.x-le.x,qe=Ue.y-le.y,P=We*We+qe*qe,lt=Math.max(Math.abs(Ue.x),Math.abs(Ue.y),Math.abs(le.x),Math.abs(le.y)),tt=oe*lt*lt;if(P<=tt){ee.splice(ze,1),ue--;continue}le=Ue}}H(I),U.forEach(H);let D=U.length,B=I;for(let ee=0;ee<D;ee++){let se=U[ee];I=I.concat(se)}function X(ee,se,oe){return se||Ve("ExtrudeGeometry: vec does not exist"),ee.clone().addScaledVector(se,oe)}let W=I.length;function ie(ee,se,oe){let le,ue,ze,Ue=ee.x-se.x,We=ee.y-se.y,qe=oe.x-ee.x,P=oe.y-ee.y,lt=Ue*Ue+We*We,tt=Ue*P-We*qe;if(Math.abs(tt)>Number.EPSILON){let b=Math.sqrt(lt),_=Math.sqrt(qe*qe+P*P),F=se.x-We/b,V=se.y+Ue/b,Y=oe.x-P/_,ce=oe.y+qe/_,he=((Y-F)*P-(ce-V)*qe)/(Ue*P-We*qe);le=F+Ue*he-ee.x,ue=V+We*he-ee.y;let Z=le*le+ue*ue;if(Z<=2)return new ae(le,ue);ze=Math.sqrt(Z/2)}else{let b=!1;Ue>Number.EPSILON?qe>Number.EPSILON&&(b=!0):Ue<-Number.EPSILON?qe<-Number.EPSILON&&(b=!0):Math.sign(We)===Math.sign(P)&&(b=!0),b?(le=-We,ue=Ue,ze=Math.sqrt(lt)):(le=Ue,ue=We,ze=Math.sqrt(lt/2))}return new ae(le/ze,ue/ze)}let q=[];for(let ee=0,se=B.length,oe=se-1,le=ee+1;ee<se;ee++,oe++,le++)oe===se&&(oe=0),le===se&&(le=0),q[ee]=ie(B[ee],B[oe],B[le]);let K=[],te,Ie=q.concat();for(let ee=0,se=D;ee<se;ee++){let oe=U[ee];te=[];for(let le=0,ue=oe.length,ze=ue-1,Ue=le+1;le<ue;le++,ze++,Ue++)ze===ue&&(ze=0),Ue===ue&&(Ue=0),te[le]=ie(oe[le],oe[ze],oe[Ue]);K.push(te),Ie=Ie.concat(te)}let we;if(m===0)we=ki.triangulateShape(B,U);else{let ee=[],se=[];for(let oe=0;oe<m;oe++){let le=oe/m,ue=f*Math.cos(le*Math.PI/2),ze=g*Math.sin(le*Math.PI/2)+y;for(let Ue=0,We=B.length;Ue<We;Ue++){let qe=X(B[Ue],q[Ue],ze);xe(qe.x,qe.y,-ue),le===0&&ee.push(qe)}for(let Ue=0,We=D;Ue<We;Ue++){let qe=U[Ue];te=K[Ue];let P=[];for(let lt=0,tt=qe.length;lt<tt;lt++){let b=X(qe[lt],te[lt],ze);xe(b.x,b.y,-ue),le===0&&P.push(b)}le===0&&se.push(P)}}we=ki.triangulateShape(ee,se)}let ot=we.length,et=g+y;for(let ee=0;ee<W;ee++){let se=u?X(I[ee],Ie[ee],et):I[ee];M?(C.copy(E.normals[0]).multiplyScalar(se.x),S.copy(E.binormals[0]).multiplyScalar(se.y),x.copy(A[0]).add(C).add(S),xe(x.x,x.y,x.z)):xe(se.x,se.y,0)}for(let ee=1;ee<=h;ee++)for(let se=0;se<W;se++){let oe=u?X(I[se],Ie[se],et):I[se];M?(C.copy(E.normals[ee]).multiplyScalar(oe.x),S.copy(E.binormals[ee]).multiplyScalar(oe.y),x.copy(A[ee]).add(C).add(S),xe(x.x,x.y,x.z)):xe(oe.x,oe.y,d/h*ee)}for(let ee=m-1;ee>=0;ee--){let se=ee/m,oe=f*Math.cos(se*Math.PI/2),le=g*Math.sin(se*Math.PI/2)+y;for(let ue=0,ze=B.length;ue<ze;ue++){let Ue=X(B[ue],q[ue],le);xe(Ue.x,Ue.y,d+oe)}for(let ue=0,ze=U.length;ue<ze;ue++){let Ue=U[ue];te=K[ue];for(let We=0,qe=Ue.length;We<qe;We++){let P=X(Ue[We],te[We],le);M?xe(P.x,P.y+A[h-1].y,A[h-1].x+oe):xe(P.x,P.y,d+oe)}}}rt(),$();function rt(){let ee=s.length/3;if(u){let se=0,oe=W*se;for(let le=0;le<ot;le++){let ue=we[le];He(ue[2]+oe,ue[1]+oe,ue[0]+oe)}se=h+m*2,oe=W*se;for(let le=0;le<ot;le++){let ue=we[le];He(ue[0]+oe,ue[1]+oe,ue[2]+oe)}}else{for(let se=0;se<ot;se++){let oe=we[se];He(oe[2],oe[1],oe[0])}for(let se=0;se<ot;se++){let oe=we[se];He(oe[0]+W*h,oe[1]+W*h,oe[2]+W*h)}}n.addGroup(ee,s.length/3-ee,0)}function $(){let ee=s.length/3,se=0;Q(B,se),se+=B.length;for(let oe=0,le=U.length;oe<le;oe++){let ue=U[oe];Q(ue,se),se+=ue.length}n.addGroup(ee,s.length/3-ee,1)}function Q(ee,se){let oe=ee.length;for(;--oe>=0;){let le=oe,ue=oe-1;ue<0&&(ue=ee.length-1);for(let ze=0,Ue=h+m*2;ze<Ue;ze++){let We=W*ze,qe=W*(ze+1),P=se+le+We,lt=se+ue+We,tt=se+ue+qe,b=se+le+qe;Se(P,lt,tt,b)}}}function xe(ee,se,oe){l.push(ee),l.push(se),l.push(oe)}function He(ee,se,oe){Ge(ee),Ge(se),Ge(oe);let le=s.length/3,ue=w.generateTopUV(n,s,le-3,le-2,le-1);pt(ue[0]),pt(ue[1]),pt(ue[2])}function Se(ee,se,oe,le){Ge(ee),Ge(se),Ge(le),Ge(se),Ge(oe),Ge(le);let ue=s.length/3,ze=w.generateSideWallUV(n,s,ue-6,ue-3,ue-2,ue-1);pt(ze[0]),pt(ze[1]),pt(ze[3]),pt(ze[1]),pt(ze[2]),pt(ze[3])}function Ge(ee){s.push(l[ee*3+0]),s.push(l[ee*3+1]),s.push(l[ee*3+2])}function pt(ee){r.push(ee.x),r.push(ee.y)}}}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}toJSON(){let e=super.toJSON(),t=this.parameters.shapes,n=this.parameters.options;return Tm(t,n,e)}static fromJSON(e,t){let n=[];for(let r=0,a=e.shapes.length;r<a;r++){let o=t[e.shapes[r]];n.push(o)}let s=e.options.extrudePath;return s!==void 0&&(e.options.extrudePath=new Ec[s.type]().fromJSON(s)),new i(n,e.options)}},Em={generateTopUV:function(i,e,t,n,s){let r=e[t*3],a=e[t*3+1],o=e[n*3],l=e[n*3+1],c=e[s*3],h=e[s*3+1];return[new ae(r,a),new ae(o,l),new ae(c,h)]},generateSideWallUV:function(i,e,t,n,s,r){let a=e[t*3],o=e[t*3+1],l=e[t*3+2],c=e[n*3],h=e[n*3+1],d=e[n*3+2],u=e[s*3],f=e[s*3+1],g=e[s*3+2],y=e[r*3],m=e[r*3+1],p=e[r*3+2];return Math.abs(o-h)<Math.abs(a-c)?[new ae(a,1-l),new ae(c,1-d),new ae(u,1-g),new ae(y,1-p)]:[new ae(o,1-l),new ae(h,1-d),new ae(f,1-g),new ae(m,1-p)]}};function Tm(i,e,t){if(t.shapes=[],Array.isArray(i))for(let n=0,s=i.length;n<s;n++){let r=i[n];t.shapes.push(r.uuid)}else t.shapes.push(i.uuid);return t.options=Object.assign({},e),e.extrudePath!==void 0&&(t.options.extrudePath=e.extrudePath.toJSON()),t}var Fr=class i extends Ut{constructor(e=1,t=1,n=1,s=1){super(),this.type="PlaneGeometry",this.parameters={width:e,height:t,widthSegments:n,heightSegments:s};let r=e/2,a=t/2,o=Math.floor(n),l=Math.floor(s),c=o+1,h=l+1,d=e/o,u=t/l,f=[],g=[],y=[],m=[];for(let p=0;p<h;p++){let w=p*u-a;for(let A=0;A<c;A++){let M=A*d-r;g.push(M,-w,0),y.push(0,0,1),m.push(A/o),m.push(1-p/l)}}for(let p=0;p<l;p++)for(let w=0;w<o;w++){let A=w+c*p,M=w+c*(p+1),E=w+1+c*(p+1),S=w+1+c*p;f.push(A,M,S),f.push(M,E,S)}this.setIndex(f),this.setAttribute("position",new Tt(g,3)),this.setAttribute("normal",new Tt(y,3)),this.setAttribute("uv",new Tt(m,2))}copy(e){return super.copy(e),this.parameters=Object.assign({},e.parameters),this}static fromJSON(e){return new i(e.width,e.height,e.widthSegments,e.heightSegments)}};function Zi(i){let e={};for(let t in i){e[t]={};for(let n in i[t]){let s=i[t][n];if(Bu(s))s.isRenderTargetTexture?(Oe("UniformsUtils: Textures of render targets cannot be cloned via cloneUniforms() or mergeUniforms()."),e[t][n]=null):e[t][n]=s.clone();else if(Array.isArray(s))if(Bu(s[0])){let r=[];for(let a=0,o=s.length;a<o;a++)r[a]=s[a].clone();e[t][n]=r}else e[t][n]=s.slice();else e[t][n]=s}}return e}function Jt(i){let e={};for(let t=0;t<i.length;t++){let n=Zi(i[t]);for(let s in n)e[s]=n[s]}return e}function Bu(i){return i&&(i.isColor||i.isMatrix3||i.isMatrix4||i.isVector2||i.isVector3||i.isVector4||i.isTexture||i.isQuaternion)}function wm(i){let e=[];for(let t=0;t<i.length;t++)e.push(i[t].clone());return e}function nh(i){let e=i.getRenderTarget();return e===null?i.outputColorSpace:e.isXRRenderTarget===!0?e.texture.colorSpace:it.workingColorSpace}var Nd={clone:Zi,merge:Jt},Am=`void main() {
	gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
}`,Rm=`void main() {
	gl_FragColor = vec4( 1.0, 0.0, 0.0, 1.0 );
}`,tn=class extends Ln{constructor(e){super(),this.isShaderMaterial=!0,this.type="ShaderMaterial",this.defines={},this.uniforms={},this.uniformsGroups=[],this.vertexShader=Am,this.fragmentShader=Rm,this.linewidth=1,this.wireframe=!1,this.wireframeLinewidth=1,this.fog=!1,this.lights=!1,this.clipping=!1,this.forceSinglePass=!0,this.extensions={clipCullDistance:!1,multiDraw:!1},this.defaultAttributeValues={color:[1,1,1],uv:[0,0],uv1:[0,0]},this.index0AttributeName=void 0,this.uniformsNeedUpdate=!1,this.glslVersion=null,e!==void 0&&this.setValues(e)}copy(e){return super.copy(e),this.fragmentShader=e.fragmentShader,this.vertexShader=e.vertexShader,this.uniforms=Zi(e.uniforms),this.uniformsGroups=wm(e.uniformsGroups),this.defines=Object.assign({},e.defines),this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.fog=e.fog,this.lights=e.lights,this.clipping=e.clipping,this.extensions=Object.assign({},e.extensions),this.glslVersion=e.glslVersion,this.defaultAttributeValues=Object.assign({},e.defaultAttributeValues),this.index0AttributeName=e.index0AttributeName,this.uniformsNeedUpdate=e.uniformsNeedUpdate,this}toJSON(e){let t=super.toJSON(e);t.glslVersion=this.glslVersion,t.uniforms={};for(let s in this.uniforms){let a=this.uniforms[s].value;a&&a.isTexture?t.uniforms[s]={type:"t",value:a.toJSON(e).uuid}:a&&a.isColor?t.uniforms[s]={type:"c",value:a.getHex()}:a&&a.isVector2?t.uniforms[s]={type:"v2",value:a.toArray()}:a&&a.isVector3?t.uniforms[s]={type:"v3",value:a.toArray()}:a&&a.isVector4?t.uniforms[s]={type:"v4",value:a.toArray()}:a&&a.isMatrix3?t.uniforms[s]={type:"m3",value:a.toArray()}:a&&a.isMatrix4?t.uniforms[s]={type:"m4",value:a.toArray()}:t.uniforms[s]={value:a}}Object.keys(this.defines).length>0&&(t.defines=this.defines),t.vertexShader=this.vertexShader,t.fragmentShader=this.fragmentShader,t.lights=this.lights,t.clipping=this.clipping;let n={};for(let s in this.extensions)this.extensions[s]===!0&&(n[s]=!0);return Object.keys(n).length>0&&(t.extensions=n),t}fromJSON(e,t){if(super.fromJSON(e,t),e.uniforms!==void 0)for(let n in e.uniforms){let s=e.uniforms[n];switch(this.uniforms[n]={},s.type){case"t":this.uniforms[n].value=t[s.value]||null;break;case"c":this.uniforms[n].value=new Ye().setHex(s.value);break;case"v2":this.uniforms[n].value=new ae().fromArray(s.value);break;case"v3":this.uniforms[n].value=new R().fromArray(s.value);break;case"v4":this.uniforms[n].value=new St().fromArray(s.value);break;case"m3":this.uniforms[n].value=new Xe().fromArray(s.value);break;case"m4":this.uniforms[n].value=new dt().fromArray(s.value);break;default:this.uniforms[n].value=s.value}}if(e.defines!==void 0&&(this.defines=e.defines),e.vertexShader!==void 0&&(this.vertexShader=e.vertexShader),e.fragmentShader!==void 0&&(this.fragmentShader=e.fragmentShader),e.glslVersion!==void 0&&(this.glslVersion=e.glslVersion),e.extensions!==void 0)for(let n in e.extensions)this.extensions[n]=e.extensions[n];return e.lights!==void 0&&(this.lights=e.lights),e.clipping!==void 0&&(this.clipping=e.clipping),this}},_o=class extends tn{constructor(e){super(e),this.isRawShaderMaterial=!0,this.type="RawShaderMaterial"}},Mi=class extends Ln{constructor(e){super(),this.isMeshStandardMaterial=!0,this.type="MeshStandardMaterial",this.defines={STANDARD:""},this.color=new Ye(16777215),this.roughness=1,this.metalness=0,this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new Ye(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=ta,this.normalScale=new ae(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.roughnessMap=null,this.metalnessMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Hn,this.envMapIntensity=1,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.defines={STANDARD:""},this.color.copy(e.color),this.roughness=e.roughness,this.metalness=e.metalness,this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.emissive.copy(e.emissive),this.emissiveMap=e.emissiveMap,this.emissiveIntensity=e.emissiveIntensity,this.bumpMap=e.bumpMap,this.bumpScale=e.bumpScale,this.normalMap=e.normalMap,this.normalMapType=e.normalMapType,this.normalScale.copy(e.normalScale),this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.roughnessMap=e.roughnessMap,this.metalnessMap=e.metalnessMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.envMapIntensity=e.envMapIntensity,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.flatShading=e.flatShading,this.fog=e.fog,this}};var Or=class extends Ln{constructor(e){super(),this.isMeshLambertMaterial=!0,this.type="MeshLambertMaterial",this.color=new Ye(16777215),this.map=null,this.lightMap=null,this.lightMapIntensity=1,this.aoMap=null,this.aoMapIntensity=1,this.emissive=new Ye(0),this.emissiveIntensity=1,this.emissiveMap=null,this.bumpMap=null,this.bumpScale=1,this.normalMap=null,this.normalMapType=ta,this.normalScale=new ae(1,1),this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.specularMap=null,this.alphaMap=null,this.envMap=null,this.envMapRotation=new Hn,this.combine=Uo,this.reflectivity=1,this.envMapIntensity=1,this.refractionRatio=.98,this.wireframe=!1,this.wireframeLinewidth=1,this.wireframeLinecap="round",this.wireframeLinejoin="round",this.flatShading=!1,this.fog=!0,this.setValues(e)}copy(e){return super.copy(e),this.color.copy(e.color),this.map=e.map,this.lightMap=e.lightMap,this.lightMapIntensity=e.lightMapIntensity,this.aoMap=e.aoMap,this.aoMapIntensity=e.aoMapIntensity,this.emissive.copy(e.emissive),this.emissiveMap=e.emissiveMap,this.emissiveIntensity=e.emissiveIntensity,this.bumpMap=e.bumpMap,this.bumpScale=e.bumpScale,this.normalMap=e.normalMap,this.normalMapType=e.normalMapType,this.normalScale.copy(e.normalScale),this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.specularMap=e.specularMap,this.alphaMap=e.alphaMap,this.envMap=e.envMap,this.envMapRotation.copy(e.envMapRotation),this.combine=e.combine,this.reflectivity=e.reflectivity,this.envMapIntensity=e.envMapIntensity,this.refractionRatio=e.refractionRatio,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this.wireframeLinecap=e.wireframeLinecap,this.wireframeLinejoin=e.wireframeLinejoin,this.flatShading=e.flatShading,this.fog=e.fog,this}},xo=class extends Ln{constructor(e){super(),this.isMeshDepthMaterial=!0,this.type="MeshDepthMaterial",this.depthPacking=md,this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.wireframe=!1,this.wireframeLinewidth=1,this.setValues(e)}copy(e){return super.copy(e),this.depthPacking=e.depthPacking,this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this.wireframe=e.wireframe,this.wireframeLinewidth=e.wireframeLinewidth,this}},yo=class extends Ln{constructor(e){super(),this.isMeshDistanceMaterial=!0,this.type="MeshDistanceMaterial",this.map=null,this.alphaMap=null,this.displacementMap=null,this.displacementScale=1,this.displacementBias=0,this.setValues(e)}copy(e){return super.copy(e),this.map=e.map,this.alphaMap=e.alphaMap,this.displacementMap=e.displacementMap,this.displacementScale=e.displacementScale,this.displacementBias=e.displacementBias,this}};function ps(i,e){return!i||i.constructor===e?i:typeof e.BYTES_PER_ELEMENT=="number"?new e(i):Array.prototype.slice.call(i)}function _c(i){return i!==void 0&&i.inTangents!==void 0&&i.outTangents!==void 0}var bi=class{constructor(e,t,n,s){this.parameterPositions=e,this._cachedIndex=0,this.resultBuffer=s!==void 0?s:new t.constructor(n),this.sampleValues=t,this.valueSize=n,this.settings=null,this.DefaultSettings_={}}evaluate(e){let t=this.parameterPositions,n=this._cachedIndex,s=t[n],r=t[n-1];n:{e:{let a;t:{i:if(!(e<s)){for(let o=n+2;;){if(s===void 0){if(e<r)break i;return n=t.length,this._cachedIndex=n,this.copySampleValue_(n-1)}if(n===o)break;if(r=s,s=t[++n],e<s)break e}a=t.length;break t}if(!(e>=r)){let o=t[1];e<o&&(n=2,r=o);for(let l=n-2;;){if(r===void 0)return this._cachedIndex=0,this.copySampleValue_(0);if(n===l)break;if(s=r,r=t[--n-1],e>=r)break e}a=n,n=0;break t}break n}for(;n<a;){let o=n+a>>>1;e<t[o]?a=o:n=o+1}if(s=t[n],r=t[n-1],r===void 0)return this._cachedIndex=0,this.copySampleValue_(0);if(s===void 0)return n=t.length,this._cachedIndex=n,this.copySampleValue_(n-1)}this._cachedIndex=n,this.intervalChanged_(n,r,s)}return this.interpolate_(n,r,e,s)}getSettings_(){return this.settings||this.DefaultSettings_}copySampleValue_(e){let t=this.resultBuffer,n=this.sampleValues,s=this.valueSize,r=e*s;for(let a=0;a!==s;++a)t[a]=n[r+a];return t}interpolate_(){throw new Error("THREE.Interpolant: Call to abstract method.")}intervalChanged_(){}},vo=class extends bi{constructor(e,t,n,s){super(e,t,n,s),this._weightPrev=-0,this._offsetPrev=-0,this._weightNext=-0,this._offsetNext=-0,this.DefaultSettings_={endingStart:vc,endingEnd:vc}}intervalChanged_(e,t,n){let s=this.parameterPositions,r=e-2,a=e+1,o=s[r],l=s[a];if(o===void 0)switch(this.getSettings_().endingStart){case Mc:r=e,o=2*t-n;break;case bc:r=s.length-2,o=t+s[r]-s[r+1];break;default:r=e,o=n}if(l===void 0)switch(this.getSettings_().endingEnd){case Mc:a=e,l=2*n-t;break;case bc:a=1,l=n+s[1]-s[0];break;default:a=e-1,l=t}let c=(n-t)*.5,h=this.valueSize;this._weightPrev=c/(t-o),this._weightNext=c/(l-n),this._offsetPrev=r*h,this._offsetNext=a*h}interpolate_(e,t,n,s){let r=this.resultBuffer,a=this.sampleValues,o=this.valueSize,l=e*o,c=l-o,h=this._offsetPrev,d=this._offsetNext,u=this._weightPrev,f=this._weightNext,g=(n-t)/(s-t),y=g*g,m=y*g,p=-u*m+2*u*y-u*g,w=(1+u)*m+(-1.5-2*u)*y+(-.5+u)*g+1,A=(-1-f)*m+(1.5+f)*y+.5*g,M=f*m-f*y;for(let E=0;E!==o;++E)r[E]=p*a[h+E]+w*a[c+E]+A*a[l+E]+M*a[d+E];return r}},Mo=class extends bi{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e,t,n,s){let r=this.resultBuffer,a=this.sampleValues,o=this.valueSize,l=e*o,c=l-o,h=(n-t)/(s-t),d=1-h;for(let u=0;u!==o;++u)r[u]=a[c+u]*d+a[l+u]*h;return r}},bo=class extends bi{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e){return this.copySampleValue_(e-1)}},So=class extends bi{interpolate_(e,t,n,s){let r=this.resultBuffer,a=this.sampleValues,o=this.valueSize,l=e*o,c=l-o,h=this.inTangents,d=this.outTangents;if(!h||!d){let g=(n-t)/(s-t),y=1-g;for(let m=0;m!==o;++m)r[m]=a[c+m]*y+a[l+m]*g;return r}let u=o*2,f=e-1;for(let g=0;g!==o;++g){let y=a[c+g],m=a[l+g],p=f*u+g*2,w=d[p],A=d[p+1],M=e*u+g*2,E=h[M],S=h[M+1],C=Pm(n,t,w,E,s);r[g]=Ud(C,y,A,S,m)}return r}};function Ud(i,e,t,n,s){let r=1-i;return r*r*r*e+3*r*r*i*t+3*r*i*i*n+i*i*i*s}function Cm(i,e,t,n,s){let r=1-i;return 3*r*r*(t-e)+6*r*i*(n-t)+3*i*i*(s-n)}function Pm(i,e,t,n,s){let r=(i-e)/(s-e);for(let a=0;a<8;a++){let o=Ud(r,e,t,n,s)-i;if(Math.abs(o)<1e-10)break;let l=Cm(r,e,t,n,s);if(Math.abs(l)<1e-10)break;r=Math.max(0,Math.min(1,r-o/l))}return r}var mn=class{constructor(e,t,n,s){if(e===void 0)throw new Error("THREE.KeyframeTrack: track name is undefined");if(t===void 0||t.length===0)throw new Error("THREE.KeyframeTrack: no keyframes in track named "+e);this.name=e,this.times=ps(t,this.TimeBufferType),this.values=ps(n,this.ValueBufferType),this.setInterpolation(s||this.DefaultInterpolation)}static toJSON(e){let t=e.constructor,n;if(t.toJSON!==this.toJSON)n=t.toJSON(e);else{n={name:e.name,times:ps(e.times,Array),values:ps(e.values,Array)};let s=e.getInterpolation();s!==e.DefaultInterpolation&&(n.interpolation=s),_c(e.settings)&&(n.settings={inTangents:ps(e.settings.inTangents,Array),outTangents:ps(e.settings.outTangents,Array)})}return n.type=e.ValueTypeName,n}InterpolantFactoryMethodDiscrete(e){return new bo(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodLinear(e){return new Mo(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodSmooth(e){return new vo(this.times,this.values,this.getValueSize(),e)}InterpolantFactoryMethodBezier(e){let t=new So(this.times,this.values,this.getValueSize(),e);return this.settings&&(t.inTangents=this.settings.inTangents,t.outTangents=this.settings.outTangents),t}setInterpolation(e){let t;switch(e){case ur:t=this.InterpolantFactoryMethodDiscrete;break;case to:t=this.InterpolantFactoryMethodLinear;break;case Xa:t=this.InterpolantFactoryMethodSmooth;break;case yc:t=this.InterpolantFactoryMethodBezier;break}if(t===void 0){let n="unsupported interpolation for "+this.ValueTypeName+" keyframe track named "+this.name;if(this.createInterpolant===void 0)if(e!==this.DefaultInterpolation)this.setInterpolation(this.DefaultInterpolation);else throw new Error(n);return Oe("KeyframeTrack:",n),this}return this.createInterpolant=t,this}getInterpolation(){switch(this.createInterpolant){case this.InterpolantFactoryMethodDiscrete:return ur;case this.InterpolantFactoryMethodLinear:return to;case this.InterpolantFactoryMethodSmooth:return Xa;case this.InterpolantFactoryMethodBezier:return yc}}getValueSize(){return this.values.length/this.times.length}shift(e){if(e!==0){let t=this.times;for(let n=0,s=t.length;n!==s;++n)t[n]+=e}return this}scale(e){if(e!==1){let t=this.times;for(let n=0,s=t.length;n!==s;++n)t[n]*=e;_c(this.settings)&&(zu(this.settings.inTangents,e),zu(this.settings.outTangents,e))}return this}trim(e,t){let n=this.times,s=n.length,r=0,a=s-1;for(;r!==s&&n[r]<e;)++r;for(;a!==-1&&n[a]>t;)--a;if(++a,r!==0||a!==s){r>=a&&(a=Math.max(a,1),r=a-1);let o=this.getValueSize();this.times=n.slice(r,a),this.values=this.values.slice(r*o,a*o)}return this}validate(){let e=!0,t=this.getValueSize();t-Math.floor(t)!==0&&(Ve("KeyframeTrack: Invalid value size in track.",this),e=!1);let n=this.times,s=this.values,r=n.length;r===0&&(Ve("KeyframeTrack: Track is empty.",this),e=!1);let a=null;for(let o=0;o!==r;o++){let l=n[o];if(typeof l=="number"&&isNaN(l)){Ve("KeyframeTrack: Time is not a valid number.",this,o,l),e=!1;break}if(a!==null&&a>l){Ve("KeyframeTrack: Out of order keys.",this,o,l,a),e=!1;break}a=l}if(s!==void 0&&xp(s))for(let o=0,l=s.length;o!==l;++o){let c=s[o];if(isNaN(c)){Ve("KeyframeTrack: Value is not a valid number.",this,o,c),e=!1;break}}return e}optimize(){let e=this.times.slice(),t=this.values.slice(),n=this.getValueSize(),s=this.getInterpolation()===Xa,r=e.length-1,a=1;for(let o=1;o<r;++o){let l=!1,c=e[o],h=e[o+1];if(c!==h&&(o!==1||c!==e[0]))if(s)l=!0;else{let d=o*n,u=d-n,f=d+n;for(let g=0;g!==n;++g){let y=t[d+g];if(y!==t[u+g]||y!==t[f+g]){l=!0;break}}}if(l){if(o!==a){e[a]=e[o];let d=o*n,u=a*n;for(let f=0;f!==n;++f)t[u+f]=t[d+f]}++a}}if(r>0){e[a]=e[r];for(let o=r*n,l=a*n,c=0;c!==n;++c)t[l+c]=t[o+c];++a}return a!==e.length?(this.times=e.slice(0,a),this.values=t.slice(0,a*n)):(this.times=e,this.values=t),this}clone(){let e=this.times.slice(),t=this.values.slice(),n=this.constructor,s=new n(this.name,e,t);return s.createInterpolant=this.createInterpolant,_c(this.settings)&&(s.settings={inTangents:this.settings.inTangents.slice(),outTangents:this.settings.outTangents.slice()}),s}};function zu(i,e){for(let t=0,n=i.length;t!==n;t+=2)i[t]*=e}mn.prototype.ValueTypeName="";mn.prototype.TimeBufferType=Float32Array;mn.prototype.ValueBufferType=Float32Array;mn.prototype.DefaultInterpolation=to;var Si=class extends mn{constructor(e,t,n){super(e,t,n)}};Si.prototype.ValueTypeName="bool";Si.prototype.ValueBufferType=Array;Si.prototype.DefaultInterpolation=ur;Si.prototype.InterpolantFactoryMethodLinear=void 0;Si.prototype.InterpolantFactoryMethodSmooth=void 0;var Eo=class extends mn{constructor(e,t,n,s){super(e,t,n,s)}};Eo.prototype.ValueTypeName="color";var To=class extends mn{constructor(e,t,n,s){super(e,t,n,s)}};To.prototype.ValueTypeName="number";var wo=class extends bi{constructor(e,t,n,s){super(e,t,n,s)}interpolate_(e,t,n,s){let r=this.resultBuffer,a=this.sampleValues,o=this.valueSize,l=(n-t)/(s-t),c=e*o;for(let h=c+o;c!==h;c+=4)fn.slerpFlat(r,0,a,c-o,a,c,l);return r}},Br=class extends mn{constructor(e,t,n,s){super(e,t,n,s)}InterpolantFactoryMethodLinear(e){return new wo(this.times,this.values,this.getValueSize(),e)}};Br.prototype.ValueTypeName="quaternion";Br.prototype.InterpolantFactoryMethodSmooth=void 0;var Ei=class extends mn{constructor(e,t,n){super(e,t,n)}};Ei.prototype.ValueTypeName="string";Ei.prototype.ValueBufferType=Array;Ei.prototype.DefaultInterpolation=ur;Ei.prototype.InterpolantFactoryMethodLinear=void 0;Ei.prototype.InterpolantFactoryMethodSmooth=void 0;var Ao=class extends mn{constructor(e,t,n,s){super(e,t,n,s)}};Ao.prototype.ValueTypeName="vector";var Ro=class{constructor(e,t,n){let s=this,r=!1,a=0,o=0,l,c=[];this.onStart=void 0,this.onLoad=e,this.onProgress=t,this.onError=n,this._abortController=null,this.itemStart=function(h){o++,r===!1&&s.onStart!==void 0&&s.onStart(h,a,o),r=!0},this.itemEnd=function(h){a++,s.onProgress!==void 0&&s.onProgress(h,a,o),a===o&&(r=!1,s.onLoad!==void 0&&s.onLoad())},this.itemError=function(h){s.onError!==void 0&&s.onError(h)},this.resolveURL=function(h){return h=h.normalize("NFC"),l?l(h):h},this.setURLModifier=function(h){return l=h,this},this.addHandler=function(h,d){return c.push(h,d),this},this.removeHandler=function(h){let d=c.indexOf(h);return d!==-1&&c.splice(d,2),this},this.getHandler=function(h){for(let d=0,u=c.length;d<u;d+=2){let f=c[d],g=c[d+1];if(f.global&&(f.lastIndex=0),f.test(h))return g}return null},this.abort=function(){return this.abortController.abort(),this._abortController=null,this}}get abortController(){return this._abortController||(this._abortController=new AbortController),this._abortController}},Fd=new Ro,Co=class{constructor(e){this.manager=e!==void 0?e:Fd,this.crossOrigin="anonymous",this.withCredentials=!1,this.path="",this.resourcePath="",this.requestHeader={},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}load(){}loadAsync(e,t){let n=this;return new Promise(function(s,r){n.load(e,s,t,r)})}parse(){}setCrossOrigin(e){return this.crossOrigin=e,this}setWithCredentials(e){return this.withCredentials=e,this}setPath(e){return this.path=e,this}setResourcePath(e){return this.resourcePath=e,this}setRequestHeader(e){return this.requestHeader=e,this}abort(){return this}};Co.DEFAULT_MATERIAL_NAME="__DEFAULT";var Ds=class extends Lt{constructor(e,t=1){super(),this.isLight=!0,this.type="Light",this.color=new Ye(e),this.intensity=t}copy(e,t){return super.copy(e,t),this.color.copy(e.color),this.intensity=e.intensity,this}toJSON(e){let t=super.toJSON(e);return t.object.color=this.color.getHex(),t.object.intensity=this.intensity,t}},zr=class extends Ds{constructor(e,t,n){super(e,n),this.isHemisphereLight=!0,this.type="HemisphereLight",this.position.copy(Lt.DEFAULT_UP),this.updateMatrix(),this.groundColor=new Ye(t)}copy(e,t){return super.copy(e,t),this.groundColor.copy(e.groundColor),this}toJSON(e){let t=super.toJSON(e);return t.object.groundColor=this.groundColor.getHex(),t}},xc=new dt,ku=new R,Vu=new R,kr=class{constructor(e){this.camera=e,this.intensity=1,this.bias=0,this.biasNode=null,this.normalBias=0,this.radius=1,this.blurSamples=8,this.mapSize=new ae(512,512),this.mapType=rn,this.map=null,this.mapPass=null,this.matrix=new dt,this.autoUpdate=!0,this.needsUpdate=!1,this._frustum=new Rs,this._frameExtents=new ae(1,1),this._viewportCount=1,this._viewports=[new St(0,0,1,1)]}getViewportCount(){return this._viewportCount}getCamera(){return this.camera}getFrustum(){return this._frustum}updateMatrices(e){let t=this.camera;ku.setFromMatrixPosition(e.matrixWorld),t.position.copy(ku),Vu.setFromMatrixPosition(e.target.matrixWorld),t.lookAt(Vu),t.updateMatrixWorld(),this._updateMatrix(t,this.matrix,this._frustum)}_updateMatrix(e,t,n,s){xc.multiplyMatrices(e.projectionMatrix,e.matrixWorldInverse),n.setFromProjectionMatrix(xc,e.coordinateSystem,e.reversedDepth);let r=this._frameExtents,a=s?s.z/r.x:1,o=s?s.w/r.y:1,l=s?s.x/r.x:0,c=s?s.y/r.y:0;e.coordinateSystem===bs||e.reversedDepth?t.set(.5*a,0,0,.5*a+l,0,.5*o,0,.5*o+c,0,0,1,0,0,0,0,1):t.set(.5*a,0,0,.5*a+l,0,.5*o,0,.5*o+c,0,0,.5,.5,0,0,0,1),t.multiply(xc)}getViewport(e){return this._viewports[e]}getFrameExtents(){return this._frameExtents}dispose(){this.map&&this.map.dispose(),this.mapPass&&this.mapPass.dispose()}copy(e){return this.camera=e.camera.clone(),this.intensity=e.intensity,this.bias=e.bias,this.radius=e.radius,this.autoUpdate=e.autoUpdate,this.needsUpdate=e.needsUpdate,this.normalBias=e.normalBias,this.blurSamples=e.blurSamples,this.mapSize.copy(e.mapSize),this.biasNode=e.biasNode,this}clone(){return new this.constructor().copy(this)}toJSON(){let e={};return e.intensity=this.intensity,e.bias=this.bias,e.normalBias=this.normalBias,e.radius=this.radius,e.blurSamples=this.blurSamples,e.mapSize=this.mapSize.toArray(),e.camera=this.camera.toJSON(!1).object,delete e.camera.matrix,e}},Ga=new R,Wa=new fn,zn=new R,Vr=class extends Lt{constructor(){super(),this.isCamera=!0,this.type="Camera",this.matrixWorldInverse=new dt,this.projectionMatrix=new dt,this.projectionMatrixInverse=new dt,this.coordinateSystem=Pn,this._reversedDepth=!1}get reversedDepth(){return this._reversedDepth}copy(e,t){return super.copy(e,t),this.matrixWorldInverse.copy(e.matrixWorldInverse),this.projectionMatrix.copy(e.projectionMatrix),this.projectionMatrixInverse.copy(e.projectionMatrixInverse),this.coordinateSystem=e.coordinateSystem,this}getWorldDirection(e){return super.getWorldDirection(e).negate()}updateMatrixWorld(e){super.updateMatrixWorld(e),this.matrixWorld.decompose(Ga,Wa,zn),zn.x===1&&zn.y===1&&zn.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(Ga,Wa,zn.set(1,1,1)).invert()}updateWorldMatrix(e,t,n=!1){super.updateWorldMatrix(e,t,n),this.matrixWorld.decompose(Ga,Wa,zn),zn.x===1&&zn.y===1&&zn.z===1?this.matrixWorldInverse.copy(this.matrixWorld).invert():this.matrixWorldInverse.compose(Ga,Wa,zn.set(1,1,1)).invert()}clone(){return new this.constructor().copy(this)}},xi=new R,Hu=new ae,Gu=new ae,Ht=class extends Vr{constructor(e=50,t=1,n=.1,s=2e3){super(),this.isPerspectiveCamera=!0,this.type="PerspectiveCamera",this.fov=e,this.zoom=1,this.near=n,this.far=s,this.focus=10,this.aspect=t,this.view=null,this.filmGauge=35,this.filmOffset=0,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.fov=e.fov,this.zoom=e.zoom,this.near=e.near,this.far=e.far,this.focus=e.focus,this.aspect=e.aspect,this.view=e.view===null?null:Object.assign({},e.view),this.filmGauge=e.filmGauge,this.filmOffset=e.filmOffset,this}setFocalLength(e){let t=.5*this.getFilmHeight()/e;this.fov=Es*2*Math.atan(t),this.updateProjectionMatrix()}getFocalLength(){let e=Math.tan(xs*.5*this.fov);return .5*this.getFilmHeight()/e}getEffectiveFOV(){return Es*2*Math.atan(Math.tan(xs*.5*this.fov)/this.zoom)}getFilmWidth(){return this.filmGauge*Math.min(this.aspect,1)}getFilmHeight(){return this.filmGauge/Math.max(this.aspect,1)}getViewBounds(e,t,n){xi.set(-1,-1,.5).applyMatrix4(this.projectionMatrixInverse),t.set(xi.x,xi.y).multiplyScalar(-e/xi.z),xi.set(1,1,.5).applyMatrix4(this.projectionMatrixInverse),n.set(xi.x,xi.y).multiplyScalar(-e/xi.z)}getViewSize(e,t){return this.getViewBounds(e,Hu,Gu),t.subVectors(Gu,Hu)}setViewOffset(e,t,n,s,r,a){this.aspect=e/t,this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=s,this.view.width=r,this.view.height=a,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){let e=this.near,t=e*Math.tan(xs*.5*this.fov)/this.zoom,n=2*t,s=this.aspect*n,r=-.5*s,a=this.view;if(this.view!==null&&this.view.enabled){let l=a.fullWidth,c=a.fullHeight;r+=a.offsetX*s/l,t-=a.offsetY*n/c,s*=a.width/l,n*=a.height/c}let o=this.filmOffset;o!==0&&(r+=e*o/this.getFilmWidth()),this.projectionMatrix.makePerspective(r,r+s,t,t-n,e,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){let t=super.toJSON(e);return t.object.fov=this.fov,t.object.zoom=this.zoom,t.object.near=this.near,t.object.far=this.far,t.object.focus=this.focus,t.object.aspect=this.aspect,this.view!==null&&(t.object.view=Object.assign({},this.view)),t.object.filmGauge=this.filmGauge,t.object.filmOffset=this.filmOffset,t}};var Rc=class extends kr{constructor(){super(new Ht(90,1,.5,500)),this.isPointLightShadow=!0}},Hr=class extends Ds{constructor(e,t,n=0,s=2){super(e,t),this.isPointLight=!0,this.type="PointLight",this.distance=n,this.decay=s,this.shadow=new Rc}get power(){return this.intensity*4*Math.PI}set power(e){this.intensity=e/(4*Math.PI)}dispose(){super.dispose(),this.shadow.dispose()}copy(e,t){return super.copy(e,t),this.distance=e.distance,this.decay=e.decay,this.shadow=e.shadow.clone(),this}toJSON(e){let t=super.toJSON(e);return t.object.distance=this.distance,t.object.decay=this.decay,t.object.shadow=this.shadow.toJSON(),t}},Ns=class extends Vr{constructor(e=-1,t=1,n=1,s=-1,r=.1,a=2e3){super(),this.isOrthographicCamera=!0,this.type="OrthographicCamera",this.zoom=1,this.view=null,this.left=e,this.right=t,this.top=n,this.bottom=s,this.near=r,this.far=a,this.updateProjectionMatrix()}copy(e,t){return super.copy(e,t),this.left=e.left,this.right=e.right,this.top=e.top,this.bottom=e.bottom,this.near=e.near,this.far=e.far,this.zoom=e.zoom,this.view=e.view===null?null:Object.assign({},e.view),this}setViewOffset(e,t,n,s,r,a){this.view===null&&(this.view={enabled:!0,fullWidth:1,fullHeight:1,offsetX:0,offsetY:0,width:1,height:1}),this.view.enabled=!0,this.view.fullWidth=e,this.view.fullHeight=t,this.view.offsetX=n,this.view.offsetY=s,this.view.width=r,this.view.height=a,this.updateProjectionMatrix()}clearViewOffset(){this.view!==null&&(this.view.enabled=!1),this.updateProjectionMatrix()}updateProjectionMatrix(){let e=(this.right-this.left)/(2*this.zoom),t=(this.top-this.bottom)/(2*this.zoom),n=(this.right+this.left)/2,s=(this.top+this.bottom)/2,r=n-e,a=n+e,o=s+t,l=s-t;if(this.view!==null&&this.view.enabled){let c=(this.right-this.left)/this.view.fullWidth/this.zoom,h=(this.top-this.bottom)/this.view.fullHeight/this.zoom;r+=c*this.view.offsetX,a=r+c*this.view.width,o-=h*this.view.offsetY,l=o-h*this.view.height}this.projectionMatrix.makeOrthographic(r,a,o,l,this.near,this.far,this.coordinateSystem,this.reversedDepth),this.projectionMatrixInverse.copy(this.projectionMatrix).invert()}toJSON(e){let t=super.toJSON(e);return t.object.zoom=this.zoom,t.object.left=this.left,t.object.right=this.right,t.object.top=this.top,t.object.bottom=this.bottom,t.object.near=this.near,t.object.far=this.far,this.view!==null&&(t.object.view=Object.assign({},this.view)),t}},Cc=class extends kr{constructor(){super(new Ns(-5,5,5,-5,.5,500)),this.isDirectionalLightShadow=!0}},Us=class extends Ds{constructor(e,t){super(e,t),this.isDirectionalLight=!0,this.type="DirectionalLight",this.position.copy(Lt.DEFAULT_UP),this.updateMatrix(),this.target=new Lt,this.shadow=new Cc}dispose(){super.dispose(),this.shadow.dispose()}copy(e){return super.copy(e),this.target=e.target.clone(),this.shadow=e.shadow.clone(),this}toJSON(e){let t=super.toJSON(e);return t.object.shadow=this.shadow.toJSON(),t.object.target=this.target.uuid,t}};var ms=-90,gs=1,Po=class extends Lt{constructor(e,t,n){super(),this.type="CubeCamera",this.renderTarget=n,this.coordinateSystem=null,this.activeMipmapLevel=0;let s=new Ht(ms,gs,e,t);s.layers=this.layers,this.add(s);let r=new Ht(ms,gs,e,t);r.layers=this.layers,this.add(r);let a=new Ht(ms,gs,e,t);a.layers=this.layers,this.add(a);let o=new Ht(ms,gs,e,t);o.layers=this.layers,this.add(o);let l=new Ht(ms,gs,e,t);l.layers=this.layers,this.add(l);let c=new Ht(ms,gs,e,t);c.layers=this.layers,this.add(c)}updateCoordinateSystem(){let e=this.coordinateSystem,t=this.children.concat(),[n,s,r,a,o,l]=t;for(let c of t)this.remove(c);if(e===Pn)n.up.set(0,1,0),n.lookAt(1,0,0),s.up.set(0,1,0),s.lookAt(-1,0,0),r.up.set(0,0,-1),r.lookAt(0,1,0),a.up.set(0,0,1),a.lookAt(0,-1,0),o.up.set(0,1,0),o.lookAt(0,0,1),l.up.set(0,1,0),l.lookAt(0,0,-1);else if(e===bs)n.up.set(0,-1,0),n.lookAt(-1,0,0),s.up.set(0,-1,0),s.lookAt(1,0,0),r.up.set(0,0,1),r.lookAt(0,1,0),a.up.set(0,0,-1),a.lookAt(0,-1,0),o.up.set(0,-1,0),o.lookAt(0,0,1),l.up.set(0,-1,0),l.lookAt(0,0,-1);else throw new Error("THREE.CubeCamera.updateCoordinateSystem(): Invalid coordinate system: "+e);for(let c of t)this.add(c),c.updateMatrixWorld()}update(e,t){this.parent===null&&this.updateMatrixWorld();let{renderTarget:n,activeMipmapLevel:s}=this;this.coordinateSystem!==e.coordinateSystem&&(this.coordinateSystem=e.coordinateSystem,this.updateCoordinateSystem());let[r,a,o,l,c,h]=this.children,d=e.getRenderTarget(),u=e.getActiveCubeFace(),f=e.getActiveMipmapLevel(),g=e.xr.enabled;e.xr.enabled=!1;let y=n.texture.generateMipmaps;n.texture.generateMipmaps=!1;let m=!1;e.isWebGLRenderer===!0?m=e.state.buffers.depth.getReversed():m=e.reversedDepthBuffer,e.setRenderTarget(n,0,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,r),e.setRenderTarget(n,1,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,a),e.setRenderTarget(n,2,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,o),e.setRenderTarget(n,3,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,l),e.setRenderTarget(n,4,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,c),n.texture.generateMipmaps=y,e.setRenderTarget(n,5,s),m&&e.autoClear===!1&&e.clearDepth(),e.render(t,h),e.setRenderTarget(d,u,f),e.xr.enabled=g,n.texture.needsPMREMUpdate=!0}},Io=class extends Ht{constructor(e=[]){super(),this.isArrayCamera=!0,this.isMultiViewCamera=!1,this.cameras=e}};var ih="\\[\\]\\.:\\/",Im=new RegExp("["+ih+"]","g"),sh="[^"+ih+"]",Lm="[^"+ih.replace("\\.","")+"]",Dm=/((?:WC+[\/:])*)/.source.replace("WC",sh),Nm=/(WCOD+)?/.source.replace("WCOD",Lm),Um=/(?:\.(WC+)(?:\[(.+)\])?)?/.source.replace("WC",sh),Fm=/\.(WC+)(?:\[(.+)\])?/.source.replace("WC",sh),Om=new RegExp("^"+Dm+Nm+Um+Fm+"$"),Bm=["material","materials","bones","map"],Pc=class{constructor(e,t,n){let s=n||bt.parseTrackName(t);this._targetGroup=e,this._bindings=e.subscribe_(t,s)}getValue(e,t){this.bind();let n=this._targetGroup.nCachedObjects_,s=this._bindings[n];s!==void 0&&s.getValue(e,t)}setValue(e,t){let n=this._bindings;for(let s=this._targetGroup.nCachedObjects_,r=n.length;s!==r;++s)n[s].setValue(e,t)}bind(){let e=this._bindings;for(let t=this._targetGroup.nCachedObjects_,n=e.length;t!==n;++t)e[t].bind()}unbind(){let e=this._bindings;for(let t=this._targetGroup.nCachedObjects_,n=e.length;t!==n;++t)e[t].unbind()}},bt=class i{constructor(e,t,n){this.path=t,this.parsedPath=n||i.parseTrackName(t),this.node=i.findNode(e,this.parsedPath.nodeName),this.rootNode=e,this.getValue=this._getValue_unbound,this.setValue=this._setValue_unbound}static create(e,t,n){return e&&e.isAnimationObjectGroup?new i.Composite(e,t,n):new i(e,t,n)}static sanitizeNodeName(e){return e.replace(/\s/g,"_").replace(Im,"")}static parseTrackName(e){let t=Om.exec(e);if(t===null)throw new Error("THREE.PropertyBinding: Cannot parse trackName: "+e);let n={nodeName:t[2],objectName:t[3],objectIndex:t[4],propertyName:t[5],propertyIndex:t[6]},s=n.nodeName&&n.nodeName.lastIndexOf(".");if(s!==void 0&&s!==-1){let r=n.nodeName.substring(s+1);Bm.indexOf(r)!==-1&&(n.nodeName=n.nodeName.substring(0,s),n.objectName=r)}if(n.propertyName===null||n.propertyName.length===0)throw new Error("THREE.PropertyBinding: can not parse propertyName from trackName: "+e);return n}static findNode(e,t){if(t===void 0||t===""||t==="."||t===-1||t===e.name||t===e.uuid)return e;if(e.skeleton){let n=e.skeleton.getBoneByName(t);if(n!==void 0)return n}if(e.children){let n=function(r){for(let a=0;a<r.length;a++){let o=r[a];if(o.name===t||o.uuid===t)return o;let l=n(o.children);if(l)return l}return null},s=n(e.children);if(s)return s}return null}_getValue_unavailable(){}_setValue_unavailable(){}_getValue_direct(e,t){e[t]=this.targetObject[this.propertyName]}_getValue_array(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)e[t++]=n[s]}_getValue_arrayElement(e,t){e[t]=this.resolvedProperty[this.propertyIndex]}_getValue_toArray(e,t){this.resolvedProperty.toArray(e,t)}_setValue_direct(e,t){this.targetObject[this.propertyName]=e[t]}_setValue_direct_setNeedsUpdate(e,t){this.targetObject[this.propertyName]=e[t],this.targetObject.needsUpdate=!0}_setValue_direct_setMatrixWorldNeedsUpdate(e,t){this.targetObject[this.propertyName]=e[t],this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_array(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++]}_setValue_array_setNeedsUpdate(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++];this.targetObject.needsUpdate=!0}_setValue_array_setMatrixWorldNeedsUpdate(e,t){let n=this.resolvedProperty;for(let s=0,r=n.length;s!==r;++s)n[s]=e[t++];this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_arrayElement(e,t){this.resolvedProperty[this.propertyIndex]=e[t]}_setValue_arrayElement_setNeedsUpdate(e,t){this.resolvedProperty[this.propertyIndex]=e[t],this.targetObject.needsUpdate=!0}_setValue_arrayElement_setMatrixWorldNeedsUpdate(e,t){this.resolvedProperty[this.propertyIndex]=e[t],this.targetObject.matrixWorldNeedsUpdate=!0}_setValue_fromArray(e,t){this.resolvedProperty.fromArray(e,t)}_setValue_fromArray_setNeedsUpdate(e,t){this.resolvedProperty.fromArray(e,t),this.targetObject.needsUpdate=!0}_setValue_fromArray_setMatrixWorldNeedsUpdate(e,t){this.resolvedProperty.fromArray(e,t),this.targetObject.matrixWorldNeedsUpdate=!0}_getValue_unbound(e,t){this.bind(),this.getValue(e,t)}_setValue_unbound(e,t){this.bind(),this.setValue(e,t)}bind(){let e=this.node,t=this.parsedPath,n=t.objectName,s=t.propertyName,r=t.propertyIndex;if(e||(e=i.findNode(this.rootNode,t.nodeName),this.node=e),this.getValue=this._getValue_unavailable,this.setValue=this._setValue_unavailable,!e){Oe("PropertyBinding: No target node found for track: "+this.path+".");return}if(n){let c=t.objectIndex;switch(n){case"materials":if(!e.material){Ve("PropertyBinding: Can not bind to material as node does not have a material.",this);return}if(!e.material.materials){Ve("PropertyBinding: Can not bind to material.materials as node.material does not have a materials array.",this);return}e=e.material.materials;break;case"bones":if(!e.skeleton){Ve("PropertyBinding: Can not bind to bones as node does not have a skeleton.",this);return}e=e.skeleton.bones;for(let h=0;h<e.length;h++)if(e[h].name===c){c=h;break}break;case"map":if("map"in e){e=e.map;break}if(!e.material){Ve("PropertyBinding: Can not bind to material as node does not have a material.",this);return}if(!e.material.map){Ve("PropertyBinding: Can not bind to material.map as node.material does not have a map.",this);return}e=e.material.map;break;default:if(e[n]===void 0){Ve("PropertyBinding: Can not bind to objectName of node undefined.",this);return}e=e[n]}if(c!==void 0){if(e[c]===void 0){Ve("PropertyBinding: Trying to bind to objectIndex of objectName, but is undefined.",this,e);return}e=e[c]}}let a=e[s];if(a===void 0){let c=t.nodeName;Ve("PropertyBinding: Trying to update property for track: "+c+"."+s+" but it wasn't found.",e);return}let o=this.Versioning.None;this.targetObject=e,e.isMaterial===!0?o=this.Versioning.NeedsUpdate:e.isObject3D===!0&&(o=this.Versioning.MatrixWorldNeedsUpdate);let l=this.BindingType.Direct;if(r!==void 0){if(s==="morphTargetInfluences"){if(!e.geometry){Ve("PropertyBinding: Can not bind to morphTargetInfluences because node does not have a geometry.",this);return}if(!e.geometry.morphAttributes){Ve("PropertyBinding: Can not bind to morphTargetInfluences because node does not have a geometry.morphAttributes.",this);return}e.morphTargetDictionary[r]!==void 0&&(r=e.morphTargetDictionary[r])}l=this.BindingType.ArrayElement,this.resolvedProperty=a,this.propertyIndex=r}else a.fromArray!==void 0&&a.toArray!==void 0?(l=this.BindingType.HasFromToArray,this.resolvedProperty=a):Array.isArray(a)?(l=this.BindingType.EntireArray,this.resolvedProperty=a):this.propertyName=s;this.getValue=this.GetterByBindingType[l],this.setValue=this.SetterByBindingTypeAndVersioning[l][o]}unbind(){this.node=null,this.getValue=this._getValue_unbound,this.setValue=this._setValue_unbound}};bt.Composite=Pc;bt.prototype.BindingType={Direct:0,EntireArray:1,ArrayElement:2,HasFromToArray:3};bt.prototype.Versioning={None:0,NeedsUpdate:1,MatrixWorldNeedsUpdate:2};bt.prototype.GetterByBindingType=[bt.prototype._getValue_direct,bt.prototype._getValue_array,bt.prototype._getValue_arrayElement,bt.prototype._getValue_toArray];bt.prototype.SetterByBindingTypeAndVersioning=[[bt.prototype._setValue_direct,bt.prototype._setValue_direct_setNeedsUpdate,bt.prototype._setValue_direct_setMatrixWorldNeedsUpdate],[bt.prototype._setValue_array,bt.prototype._setValue_array_setNeedsUpdate,bt.prototype._setValue_array_setMatrixWorldNeedsUpdate],[bt.prototype._setValue_arrayElement,bt.prototype._setValue_arrayElement_setNeedsUpdate,bt.prototype._setValue_arrayElement_setMatrixWorldNeedsUpdate],[bt.prototype._setValue_fromArray,bt.prototype._setValue_fromArray_setNeedsUpdate,bt.prototype._setValue_fromArray_setMatrixWorldNeedsUpdate]];var Gv=new Float32Array(1);var Wu=new dt,Gr=class{constructor(e,t,n=0,s=1/0){this.ray=new ii(e,t),this.near=n,this.far=s,this.camera=null,this.layers=new ws,this.params={Mesh:{},Line:{threshold:1},LOD:{},Points:{threshold:1},Sprite:{}}}set(e,t){this.ray.set(e,t)}setFromCamera(e,t){t.isPerspectiveCamera?(this.ray.origin.setFromMatrixPosition(t.matrixWorld),this.ray.direction.set(e.x,e.y,.5).unproject(t).sub(this.ray.origin).normalize(),this.camera=t):t.isOrthographicCamera?(this.ray.origin.set(e.x,e.y,t.projectionMatrix.elements[14]).unproject(t),this.ray.direction.set(0,0,-1).transformDirection(t.matrixWorld),this.camera=t):Ve("Raycaster: Unsupported camera type: "+t.type)}setFromXRController(e){return Wu.identity().extractRotation(e.matrixWorld),this.ray.origin.setFromMatrixPosition(e.matrixWorld),this.ray.direction.set(0,0,-1).applyMatrix4(Wu),this}intersectObject(e,t=!0,n=[]){return Ic(e,this,n,t),n.sort(Xu),n}intersectObjects(e,t=!0,n=[]){for(let s=0,r=e.length;s<r;s++)Ic(e[s],this,n,t);return n.sort(Xu),n}};function Xu(i,e){return i.distance-e.distance}function Ic(i,e,t,n){let s=!0;if(i.layers.test(e.layers)&&i.raycast(e,t)===!1&&(s=!1),s===!0&&n===!0){let r=i.children;for(let a=0,o=r.length;a<o;a++)Ic(r[a],e,t,!0)}}var Fs=class{constructor(e=1,t=0,n=0){this.radius=e,this.phi=t,this.theta=n}set(e,t,n){return this.radius=e,this.phi=t,this.theta=n,this}copy(e){return this.radius=e.radius,this.phi=e.phi,this.theta=e.theta,this}makeSafe(){return this.phi=je(this.phi,1e-6,Math.PI-1e-6),this}setFromVector3(e){return this.setFromCartesianCoords(e.x,e.y,e.z)}setFromCartesianCoords(e,t,n){return this.radius=Math.sqrt(e*e+t*t+n*n),this.radius===0?(this.theta=0,this.phi=0):(this.theta=Math.atan2(e,n),this.phi=Math.acos(je(t/this.radius,-1,1))),this}clone(){return new this.constructor().copy(this)}};var hh=class hh{constructor(e,t,n,s){this.elements=[1,0,0,1],e!==void 0&&this.set(e,t,n,s)}identity(){return this.set(1,0,0,1),this}fromArray(e,t=0){for(let n=0;n<4;n++)this.elements[n]=e[n+t];return this}set(e,t,n,s){let r=this.elements;return r[0]=e,r[2]=t,r[1]=n,r[3]=s,this}};hh.prototype.isMatrix2=!0;var Lc=hh;var Wr=class extends In{constructor(e,t=null){super(),this.object=e,this.domElement=t,this.enabled=!0,this.state=-1,this.keys={},this.mouseButtons={LEFT:null,MIDDLE:null,RIGHT:null},this.touches={ONE:null,TWO:null}}connect(e){this.domElement!==null&&this.disconnect(),this.domElement=e}disconnect(){}dispose(){}update(){}};function rh(i,e,t,n){let s=zm(n);switch(t){case Jc:return i*e;case Ho:return i*e/s.components*s.byteLength;case Go:return i*e/s.components*s.byteLength;case Ii:return i*e*2/s.components*s.byteLength;case Wo:return i*e*2/s.components*s.byteLength;case jc:return i*e*3/s.components*s.byteLength;case Mn:return i*e*4/s.components*s.byteLength;case Xo:return i*e*4/s.components*s.byteLength;case $r:case Jr:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case jr:case Kr:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case Yo:case $o:return Math.max(i,16)*Math.max(e,8)/4;case qo:case Zo:return Math.max(i,8)*Math.max(e,8)/2;case Jo:case jo:case Qo:case el:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*8;case Ko:case Qr:case tl:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case nl:return Math.floor((i+3)/4)*Math.floor((e+3)/4)*16;case il:return Math.floor((i+4)/5)*Math.floor((e+3)/4)*16;case sl:return Math.floor((i+4)/5)*Math.floor((e+4)/5)*16;case rl:return Math.floor((i+5)/6)*Math.floor((e+4)/5)*16;case al:return Math.floor((i+5)/6)*Math.floor((e+5)/6)*16;case ol:return Math.floor((i+7)/8)*Math.floor((e+4)/5)*16;case ll:return Math.floor((i+7)/8)*Math.floor((e+5)/6)*16;case cl:return Math.floor((i+7)/8)*Math.floor((e+7)/8)*16;case hl:return Math.floor((i+9)/10)*Math.floor((e+4)/5)*16;case ul:return Math.floor((i+9)/10)*Math.floor((e+5)/6)*16;case dl:return Math.floor((i+9)/10)*Math.floor((e+7)/8)*16;case fl:return Math.floor((i+9)/10)*Math.floor((e+9)/10)*16;case pl:return Math.floor((i+11)/12)*Math.floor((e+9)/10)*16;case ml:return Math.floor((i+11)/12)*Math.floor((e+11)/12)*16;case gl:case _l:case xl:return Math.ceil(i/4)*Math.ceil(e/4)*16;case yl:case vl:return Math.ceil(i/4)*Math.ceil(e/4)*8;case ea:case Ml:return Math.ceil(i/4)*Math.ceil(e/4)*16}throw new Error(`Unable to determine texture byte length for ${t} format.`)}function zm(i){switch(i){case rn:case qc:return{byteLength:1,components:1};case zs:case Yc:case Un:return{byteLength:2,components:1};case ko:case Vo:return{byteLength:2,components:4};case Nn:case zo:case vn:return{byteLength:4,components:1};case Zc:case $c:return{byteLength:4,components:3}}throw new Error(`THREE.TextureUtils: Unknown texture type ${i}.`)}typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("register",{detail:{revision:Do}}));typeof window<"u"&&(window.__THREE__?Oe("WARNING: Multiple instances of Three.js being imported."):window.__THREE__=Do);function rf(){let i=null,e=!1,t=null,n=null;function s(r,a){n=i.requestAnimationFrame(s),t(r,a)}return{start:function(){e!==!0&&t!==null&&i!==null&&(n=i.requestAnimationFrame(s),e=!0)},stop:function(){i!==null&&i.cancelAnimationFrame(n),e=!1},setAnimationLoop:function(r){t=r},setContext:function(r){i=r}}}function km(i){let e=new WeakMap;function t(o,l){let c=o.array,h=o.usage,d=c.byteLength,u=i.createBuffer();i.bindBuffer(l,u),i.bufferData(l,c,h),o.onUploadCallback();let f;if(c instanceof Float32Array)f=i.FLOAT;else if(typeof Float16Array<"u"&&c instanceof Float16Array)f=i.HALF_FLOAT;else if(c instanceof Uint16Array)o.isFloat16BufferAttribute?f=i.HALF_FLOAT:f=i.UNSIGNED_SHORT;else if(c instanceof Int16Array)f=i.SHORT;else if(c instanceof Uint32Array)f=i.UNSIGNED_INT;else if(c instanceof Int32Array)f=i.INT;else if(c instanceof Int8Array)f=i.BYTE;else if(c instanceof Uint8Array)f=i.UNSIGNED_BYTE;else if(c instanceof Uint8ClampedArray)f=i.UNSIGNED_BYTE;else throw new Error("THREE.WebGLAttributes: Unsupported buffer data format: "+c);return{buffer:u,type:f,bytesPerElement:c.BYTES_PER_ELEMENT,version:o.version,size:d}}function n(o,l,c){let h=l.array,d=l.updateRanges;if(i.bindBuffer(c,o),d.length===0)i.bufferSubData(c,0,h);else{d.sort((f,g)=>f.start-g.start);let u=0;for(let f=1;f<d.length;f++){let g=d[u],y=d[f];y.start<=g.start+g.count+1?g.count=Math.max(g.count,y.start+y.count-g.start):(++u,d[u]=y)}d.length=u+1;for(let f=0,g=d.length;f<g;f++){let y=d[f];i.bufferSubData(c,y.start*h.BYTES_PER_ELEMENT,h,y.start,y.count)}l.clearUpdateRanges()}l.onUploadCallback()}function s(o){return o.isInterleavedBufferAttribute&&(o=o.data),e.get(o)}function r(o){o.isInterleavedBufferAttribute&&(o=o.data);let l=e.get(o);l&&(i.deleteBuffer(l.buffer),e.delete(o))}function a(o,l){if(o.isInterleavedBufferAttribute&&(o=o.data),o.isGLBufferAttribute){let h=e.get(o);(!h||h.version<o.version)&&e.set(o,{buffer:o.buffer,type:o.type,bytesPerElement:o.elementSize,version:o.version});return}let c=e.get(o);if(c===void 0)e.set(o,t(o,l));else if(c.version<o.version){if(c.size!==o.array.byteLength)throw new Error("THREE.WebGLAttributes: The size of the buffer attribute's array buffer does not match the original size. Resizing buffer attributes is not supported.");n(c.buffer,o,l),c.version=o.version}}return{get:s,remove:r,update:a}}var Vm=`#ifdef USE_ALPHAHASH
	if ( diffuseColor.a < getAlphaHashThreshold( vPosition ) ) discard;
#endif`,Hm=`#ifdef USE_ALPHAHASH
	const float ALPHA_HASH_SCALE = 0.05;
	float hash2D( vec2 value ) {
		return fract( 1.0e4 * sin( 17.0 * value.x + 0.1 * value.y ) * ( 0.1 + abs( sin( 13.0 * value.y + value.x ) ) ) );
	}
	float hash3D( vec3 value ) {
		return hash2D( vec2( hash2D( value.xy ), value.z ) );
	}
	float getAlphaHashThreshold( vec3 position ) {
		float maxDeriv = max(
			length( dFdx( position.xyz ) ),
			length( dFdy( position.xyz ) )
		);
		float pixScale = 1.0 / ( ALPHA_HASH_SCALE * maxDeriv );
		vec2 pixScales = vec2(
			exp2( floor( log2( pixScale ) ) ),
			exp2( ceil( log2( pixScale ) ) )
		);
		vec2 alpha = vec2(
			hash3D( floor( pixScales.x * position.xyz ) ),
			hash3D( floor( pixScales.y * position.xyz ) )
		);
		float lerpFactor = fract( log2( pixScale ) );
		float x = ( 1.0 - lerpFactor ) * alpha.x + lerpFactor * alpha.y;
		float a = min( lerpFactor, 1.0 - lerpFactor );
		vec3 cases = vec3(
			x * x / ( 2.0 * a * ( 1.0 - a ) ),
			( x - 0.5 * a ) / ( 1.0 - a ),
			1.0 - ( ( 1.0 - x ) * ( 1.0 - x ) / ( 2.0 * a * ( 1.0 - a ) ) )
		);
		float threshold = ( x < ( 1.0 - a ) )
			? ( ( x < a ) ? cases.x : cases.y )
			: cases.z;
		return clamp( threshold , 1.0e-6, 1.0 );
	}
#endif`,Gm=`#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, vAlphaMapUv ).g;
#endif`,Wm=`#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Xm=`#ifdef USE_ALPHATEST
	#ifdef ALPHA_TO_COVERAGE
	diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );
	if ( diffuseColor.a == 0.0 ) discard;
	#else
	if ( diffuseColor.a < alphaTest ) discard;
	#endif
#endif`,qm=`#ifdef USE_ALPHATEST
	uniform float alphaTest;
#endif`,Ym=`#ifdef USE_AOMAP
	float ambientOcclusion = ( texture2D( aoMap, vAoMapUv ).r - 1.0 ) * aoMapIntensity + 1.0;
	reflectedLight.indirectDiffuse *= ambientOcclusion;
	#if defined( USE_CLEARCOAT ) 
		clearcoatSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_SHEEN ) 
		sheenSpecularIndirect *= ambientOcclusion;
	#endif
	#if defined( USE_ENVMAP ) && defined( STANDARD )
		float dotNV = saturate( dot( geometryNormal, geometryViewDir ) );
		reflectedLight.indirectSpecular *= computeSpecularOcclusion( dotNV, ambientOcclusion, material.roughness );
	#endif
#endif`,Zm=`#ifdef USE_AOMAP
	uniform sampler2D aoMap;
	uniform float aoMapIntensity;
#endif`,$m=`#ifdef USE_BATCHING
	#if ! defined( GL_ANGLE_multi_draw )
	#define gl_DrawID _gl_DrawID
	uniform int _gl_DrawID;
	#endif
	uniform highp sampler2D batchingTexture;
	uniform highp usampler2D batchingIdTexture;
	mat4 getBatchingMatrix( const in float i ) {
		int size = textureSize( batchingTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( batchingTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( batchingTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( batchingTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( batchingTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
	float getIndirectIndex( const in int i ) {
		int size = textureSize( batchingIdTexture, 0 ).x;
		int x = i % size;
		int y = i / size;
		return float( texelFetch( batchingIdTexture, ivec2( x, y ), 0 ).r );
	}
#endif
#ifdef USE_BATCHING_COLOR
	uniform sampler2D batchingColorTexture;
	vec4 getBatchingColor( const in float i ) {
		int size = textureSize( batchingColorTexture, 0 ).x;
		int j = int( i );
		int x = j % size;
		int y = j / size;
		return texelFetch( batchingColorTexture, ivec2( x, y ), 0 );
	}
#endif`,Jm=`#ifdef USE_BATCHING
	mat4 batchingMatrix = getBatchingMatrix( getIndirectIndex( gl_DrawID ) );
#endif`,jm=`vec3 transformed = vec3( position );
#ifdef USE_ALPHAHASH
	vPosition = vec3( position );
#endif`,Km=`vec3 objectNormal = vec3( normal );
#ifdef USE_TANGENT
	vec3 objectTangent = vec3( tangent.xyz );
#endif`,Qm=`float G_BlinnPhong_Implicit( ) {
	return 0.25;
}
float D_BlinnPhong( const in float shininess, const in float dotNH ) {
	return RECIPROCAL_PI * ( shininess * 0.5 + 1.0 ) * pow( dotNH, shininess );
}
vec3 BRDF_BlinnPhong( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in vec3 specularColor, const in float shininess ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( specularColor, 1.0, dotVH );
	float G = G_BlinnPhong_Implicit( );
	float D = D_BlinnPhong( shininess, dotNH );
	return F * ( G * D );
} // validated`,eg=`#ifdef USE_IRIDESCENCE
	const mat3 XYZ_TO_REC709 = mat3(
		 3.2404542, -0.9692660,  0.0556434,
		-1.5371385,  1.8760108, -0.2040259,
		-0.4985314,  0.0415560,  1.0572252
	);
	vec3 Fresnel0ToIor( vec3 fresnel0 ) {
		vec3 sqrtF0 = sqrt( fresnel0 );
		return ( vec3( 1.0 ) + sqrtF0 ) / ( vec3( 1.0 ) - sqrtF0 );
	}
	vec3 IorToFresnel0( vec3 transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - vec3( incidentIor ) ) / ( transmittedIor + vec3( incidentIor ) ) );
	}
	float IorToFresnel0( float transmittedIor, float incidentIor ) {
		return pow2( ( transmittedIor - incidentIor ) / ( transmittedIor + incidentIor ));
	}
	vec3 evalSensitivity( float OPD, vec3 shift ) {
		float phase = 2.0 * PI * OPD * 1.0e-9;
		vec3 val = vec3( 5.4856e-13, 4.4201e-13, 5.2481e-13 );
		vec3 pos = vec3( 1.6810e+06, 1.7953e+06, 2.2084e+06 );
		vec3 var = vec3( 4.3278e+09, 9.3046e+09, 6.6121e+09 );
		vec3 xyz = val * sqrt( 2.0 * PI * var ) * cos( pos * phase + shift ) * exp( - pow2( phase ) * var );
		xyz.x += 9.7470e-14 * sqrt( 2.0 * PI * 4.5282e+09 ) * cos( 2.2399e+06 * phase + shift[ 0 ] ) * exp( - 4.5282e+09 * pow2( phase ) );
		xyz /= 1.0685e-7;
		vec3 rgb = XYZ_TO_REC709 * xyz;
		return rgb;
	}
	vec3 evalIridescence( float outsideIOR, float eta2, float cosTheta1, float thinFilmThickness, vec3 baseF0 ) {
		vec3 I;
		float iridescenceIOR = mix( outsideIOR, eta2, smoothstep( 0.0, 0.03, thinFilmThickness ) );
		float sinTheta2Sq = pow2( outsideIOR / iridescenceIOR ) * ( 1.0 - pow2( cosTheta1 ) );
		float cosTheta2Sq = 1.0 - sinTheta2Sq;
		if ( cosTheta2Sq < 0.0 ) {
			return vec3( 1.0 );
		}
		float cosTheta2 = sqrt( cosTheta2Sq );
		float R0 = IorToFresnel0( iridescenceIOR, outsideIOR );
		float R12 = F_Schlick( R0, 1.0, cosTheta1 );
		float T121 = 1.0 - R12;
		float phi12 = 0.0;
		if ( iridescenceIOR < outsideIOR ) phi12 = PI;
		float phi21 = PI - phi12;
		vec3 baseIOR = Fresnel0ToIor( clamp( baseF0, 0.0, 0.9999 ) );		vec3 R1 = IorToFresnel0( baseIOR, iridescenceIOR );
		vec3 R23 = F_Schlick( R1, 1.0, cosTheta2 );
		vec3 phi23 = vec3( 0.0 );
		if ( baseIOR[ 0 ] < iridescenceIOR ) phi23[ 0 ] = PI;
		if ( baseIOR[ 1 ] < iridescenceIOR ) phi23[ 1 ] = PI;
		if ( baseIOR[ 2 ] < iridescenceIOR ) phi23[ 2 ] = PI;
		float OPD = 2.0 * iridescenceIOR * thinFilmThickness * cosTheta2;
		vec3 phi = vec3( phi21 ) + phi23;
		vec3 R123 = clamp( R12 * R23, 1e-5, 0.9999 );
		vec3 r123 = sqrt( R123 );
		vec3 Rs = pow2( T121 ) * R23 / ( vec3( 1.0 ) - R123 );
		vec3 C0 = R12 + Rs;
		I = C0;
		vec3 Cm = Rs - T121;
		for ( int m = 1; m <= 2; ++ m ) {
			Cm *= r123;
			vec3 Sm = 2.0 * evalSensitivity( float( m ) * OPD, float( m ) * phi );
			I += Cm * Sm;
		}
		return max( I, vec3( 0.0 ) );
	}
#endif`,tg=`#ifdef USE_BUMPMAP
	uniform sampler2D bumpMap;
	uniform float bumpScale;
	vec2 dHdxy_fwd() {
		vec2 dSTdx = dFdx( vBumpMapUv );
		vec2 dSTdy = dFdy( vBumpMapUv );
		float Hll = bumpScale * texture2D( bumpMap, vBumpMapUv ).x;
		float dBx = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdx ).x - Hll;
		float dBy = bumpScale * texture2D( bumpMap, vBumpMapUv + dSTdy ).x - Hll;
		return vec2( dBx, dBy );
	}
	vec3 perturbNormalArb( vec3 surf_pos, vec3 surf_norm, vec2 dHdxy, float faceDirection ) {
		vec3 vSigmaX = normalize( dFdx( surf_pos.xyz ) );
		vec3 vSigmaY = normalize( dFdy( surf_pos.xyz ) );
		vec3 vN = surf_norm;
		vec3 R1 = cross( vSigmaY, vN );
		vec3 R2 = cross( vN, vSigmaX );
		float fDet = dot( vSigmaX, R1 ) * faceDirection;
		vec3 vGrad = sign( fDet ) * ( dHdxy.x * R1 + dHdxy.y * R2 );
		return normalize( abs( fDet ) * surf_norm - vGrad );
	}
#endif`,ng=`#if NUM_CLIPPING_PLANES > 0
	vec4 plane;
	#ifdef ALPHA_TO_COVERAGE
		float distanceToPlane, distanceGradient;
		float clipOpacity = 1.0;
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
			distanceGradient = fwidth( distanceToPlane ) / 2.0;
			clipOpacity *= smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			if ( clipOpacity == 0.0 ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			float unionClipOpacity = 1.0;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				distanceToPlane = - dot( vClipPosition, plane.xyz ) + plane.w;
				distanceGradient = fwidth( distanceToPlane ) / 2.0;
				unionClipOpacity *= 1.0 - smoothstep( - distanceGradient, distanceGradient, distanceToPlane );
			}
			#pragma unroll_loop_end
			clipOpacity *= 1.0 - unionClipOpacity;
		#endif
		diffuseColor.a *= clipOpacity;
		if ( diffuseColor.a == 0.0 ) discard;
	#else
		#pragma unroll_loop_start
		for ( int i = 0; i < UNION_CLIPPING_PLANES; i ++ ) {
			plane = clippingPlanes[ i ];
			if ( dot( vClipPosition, plane.xyz ) > plane.w ) discard;
		}
		#pragma unroll_loop_end
		#if UNION_CLIPPING_PLANES < NUM_CLIPPING_PLANES
			bool clipped = true;
			#pragma unroll_loop_start
			for ( int i = UNION_CLIPPING_PLANES; i < NUM_CLIPPING_PLANES; i ++ ) {
				plane = clippingPlanes[ i ];
				clipped = ( dot( vClipPosition, plane.xyz ) > plane.w ) && clipped;
			}
			#pragma unroll_loop_end
			if ( clipped ) discard;
		#endif
	#endif
#endif`,ig=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
	uniform vec4 clippingPlanes[ NUM_CLIPPING_PLANES ];
#endif`,sg=`#if NUM_CLIPPING_PLANES > 0
	varying vec3 vClipPosition;
#endif`,rg=`#if NUM_CLIPPING_PLANES > 0
	vClipPosition = - mvPosition.xyz;
#endif`,ag=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	diffuseColor *= vColor;
#endif`,og=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA )
	varying vec4 vColor;
#endif`,lg=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	varying vec4 vColor;
#endif`,cg=`#if defined( USE_COLOR ) || defined( USE_COLOR_ALPHA ) || defined( USE_INSTANCING_COLOR ) || defined( USE_BATCHING_COLOR )
	vColor = vec4( 1.0 );
#endif
#ifdef USE_COLOR_ALPHA
	vColor *= color;
#elif defined( USE_COLOR )
	vColor.rgb *= color;
#endif
#ifdef USE_INSTANCING_COLOR
	vColor.rgb *= instanceColor.rgb;
#endif
#ifdef USE_BATCHING_COLOR
	vColor *= getBatchingColor( getIndirectIndex( gl_DrawID ) );
#endif`,hg=`#define PI 3.141592653589793
#define PI2 6.283185307179586
#define PI_HALF 1.5707963267948966
#define RECIPROCAL_PI 0.3183098861837907
#define RECIPROCAL_PI2 0.15915494309189535
#define EPSILON 1e-6
#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
#define whiteComplement( a ) ( 1.0 - saturate( a ) )
float pow2( const in float x ) { return x*x; }
vec3 pow2( const in vec3 x ) { return x*x; }
float pow3( const in float x ) { return x*x*x; }
float pow4( const in float x ) { float x2 = x*x; return x2*x2; }
float max3( const in vec3 v ) { return max( max( v.x, v.y ), v.z ); }
float average( const in vec3 v ) { return dot( v, vec3( 0.3333333 ) ); }
highp float rand( const in vec2 uv ) {
	const highp float a = 12.9898, b = 78.233, c = 43758.5453;
	highp float dt = dot( uv.xy, vec2( a,b ) ), sn = mod( dt, PI );
	return fract( sin( sn ) * c );
}
#ifdef HIGH_PRECISION
	float precisionSafeLength( vec3 v ) { return length( v ); }
#else
	float precisionSafeLength( vec3 v ) {
		float maxComponent = max3( abs( v ) );
		return length( v / maxComponent ) * maxComponent;
	}
#endif
struct IncidentLight {
	vec3 color;
	vec3 direction;
	bool visible;
};
struct ReflectedLight {
	vec3 directDiffuse;
	vec3 directSpecular;
	vec3 indirectDiffuse;
	vec3 indirectSpecular;
};
#ifdef USE_ALPHAHASH
	varying vec3 vPosition;
#endif
vec3 transformDirection( in vec3 dir, in mat4 matrix ) {
	return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );
}
#define inverseTransformDirection transformDirectionByInverseViewMatrix
vec3 transformNormalByInverseViewMatrix( in vec3 normal, in mat4 viewMatrix ) {
	return normalize( ( vec4( normal, 0.0 ) * viewMatrix ).xyz );
}
vec3 transformDirectionByInverseViewMatrix( in vec3 dir, in mat4 viewMatrix ) {
	return normalize( ( vec4( dir, 0.0 ) * viewMatrix ).xyz );
}
bool isPerspectiveMatrix( mat4 m ) {
	return m[ 2 ][ 3 ] == - 1.0;
}
vec2 equirectUv( in vec3 dir ) {
	float u = atan( dir.z, dir.x ) * RECIPROCAL_PI2 + 0.5;
	float v = asin( clamp( dir.y, - 1.0, 1.0 ) ) * RECIPROCAL_PI + 0.5;
	return vec2( u, v );
}
vec3 BRDF_Lambert( const in vec3 diffuseColor ) {
	return RECIPROCAL_PI * diffuseColor;
}
vec3 F_Schlick( const in vec3 f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
}
float F_Schlick( const in float f0, const in float f90, const in float dotVH ) {
	float fresnel = exp2( ( - 5.55473 * dotVH - 6.98316 ) * dotVH );
	return f0 * ( 1.0 - fresnel ) + ( f90 * fresnel );
} // validated`,ug=`#ifdef ENVMAP_TYPE_CUBE_UV
	#define cubeUV_minMipLevel 4.0
	#define cubeUV_minTileSize 16.0
	float getFace( vec3 direction ) {
		vec3 absDirection = abs( direction );
		float face = - 1.0;
		if ( absDirection.x > absDirection.z ) {
			if ( absDirection.x > absDirection.y )
				face = direction.x > 0.0 ? 0.0 : 3.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		} else {
			if ( absDirection.z > absDirection.y )
				face = direction.z > 0.0 ? 2.0 : 5.0;
			else
				face = direction.y > 0.0 ? 1.0 : 4.0;
		}
		return face;
	}
	vec2 getUV( vec3 direction, float face ) {
		vec2 uv;
		if ( face == 0.0 ) {
			uv = vec2( direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 1.0 ) {
			uv = vec2( - direction.x, - direction.z ) / abs( direction.y );
		} else if ( face == 2.0 ) {
			uv = vec2( - direction.x, direction.y ) / abs( direction.z );
		} else if ( face == 3.0 ) {
			uv = vec2( - direction.z, direction.y ) / abs( direction.x );
		} else if ( face == 4.0 ) {
			uv = vec2( - direction.x, direction.z ) / abs( direction.y );
		} else {
			uv = vec2( direction.x, direction.y ) / abs( direction.z );
		}
		return 0.5 * ( uv + 1.0 );
	}
	vec3 bilinearCubeUV( sampler2D envMap, vec3 direction, float mipInt ) {
		float face = getFace( direction );
		float filterInt = max( cubeUV_minMipLevel - mipInt, 0.0 );
		mipInt = max( mipInt, cubeUV_minMipLevel );
		float faceSize = exp2( mipInt );
		highp vec2 uv = getUV( direction, face ) * ( faceSize - 2.0 ) + 1.0;
		if ( face > 2.0 ) {
			uv.y += faceSize;
			face -= 3.0;
		}
		uv.x += face * faceSize;
		uv.x += filterInt * 3.0 * cubeUV_minTileSize;
		uv.y += 4.0 * ( exp2( CUBEUV_MAX_MIP ) - faceSize );
		uv.x *= CUBEUV_TEXEL_WIDTH;
		uv.y *= CUBEUV_TEXEL_HEIGHT;
		#ifdef texture2DGradEXT
			return texture2DGradEXT( envMap, uv, vec2( 0.0 ), vec2( 0.0 ) ).rgb;
		#else
			return texture2D( envMap, uv ).rgb;
		#endif
	}
	#define cubeUV_r0 1.0
	#define cubeUV_m0 - 2.0
	#define cubeUV_r1 0.8
	#define cubeUV_m1 - 1.0
	#define cubeUV_r4 0.4
	#define cubeUV_m4 2.0
	#define cubeUV_r5 0.305
	#define cubeUV_m5 3.0
	#define cubeUV_r6 0.21
	#define cubeUV_m6 4.0
	float roughnessToMip( float roughness ) {
		float mip = 0.0;
		if ( roughness >= cubeUV_r1 ) {
			mip = ( cubeUV_r0 - roughness ) * ( cubeUV_m1 - cubeUV_m0 ) / ( cubeUV_r0 - cubeUV_r1 ) + cubeUV_m0;
		} else if ( roughness >= cubeUV_r4 ) {
			mip = ( cubeUV_r1 - roughness ) * ( cubeUV_m4 - cubeUV_m1 ) / ( cubeUV_r1 - cubeUV_r4 ) + cubeUV_m1;
		} else if ( roughness >= cubeUV_r5 ) {
			mip = ( cubeUV_r4 - roughness ) * ( cubeUV_m5 - cubeUV_m4 ) / ( cubeUV_r4 - cubeUV_r5 ) + cubeUV_m4;
		} else if ( roughness >= cubeUV_r6 ) {
			mip = ( cubeUV_r5 - roughness ) * ( cubeUV_m6 - cubeUV_m5 ) / ( cubeUV_r5 - cubeUV_r6 ) + cubeUV_m5;
		} else {
			mip = - 2.0 * log2( 1.16 * roughness );		}
		return mip;
	}
	vec4 textureCubeUV( sampler2D envMap, vec3 sampleDir, float roughness ) {
		float mip = clamp( roughnessToMip( roughness ), cubeUV_m0, CUBEUV_MAX_MIP );
		float mipF = fract( mip );
		float mipInt = floor( mip );
		vec3 color0 = bilinearCubeUV( envMap, sampleDir, mipInt );
		if ( mipF == 0.0 ) {
			return vec4( color0, 1.0 );
		} else {
			vec3 color1 = bilinearCubeUV( envMap, sampleDir, mipInt + 1.0 );
			return vec4( mix( color0, color1, mipF ), 1.0 );
		}
	}
#endif`,dg=`vec3 transformedNormal = objectNormal;
#ifdef USE_TANGENT
	vec3 transformedTangent = objectTangent;
#endif
#ifdef USE_BATCHING
	mat3 bm = mat3( batchingMatrix );
	transformedNormal /= vec3( dot( bm[ 0 ], bm[ 0 ] ), dot( bm[ 1 ], bm[ 1 ] ), dot( bm[ 2 ], bm[ 2 ] ) );
	transformedNormal = bm * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = bm * transformedTangent;
	#endif
#endif
#ifdef USE_INSTANCING
	mat3 im = mat3( instanceMatrix );
	transformedNormal /= vec3( dot( im[ 0 ], im[ 0 ] ), dot( im[ 1 ], im[ 1 ] ), dot( im[ 2 ], im[ 2 ] ) );
	transformedNormal = im * transformedNormal;
	#ifdef USE_TANGENT
		transformedTangent = im * transformedTangent;
	#endif
#endif
transformedNormal = normalMatrix * transformedNormal;
#ifdef FLIP_SIDED
	transformedNormal = - transformedNormal;
#endif
#ifdef USE_TANGENT
	transformedTangent = ( modelViewMatrix * vec4( transformedTangent, 0.0 ) ).xyz;
#endif`,fg=`#ifdef USE_DISPLACEMENTMAP
	uniform sampler2D displacementMap;
	uniform float displacementScale;
	uniform float displacementBias;
#endif`,pg=`#ifdef USE_DISPLACEMENTMAP
	transformed += normalize( objectNormal ) * ( texture2D( displacementMap, vDisplacementMapUv ).x * displacementScale + displacementBias );
#endif`,mg=`#ifdef USE_EMISSIVEMAP
	vec4 emissiveColor = texture2D( emissiveMap, vEmissiveMapUv );
	#ifdef DECODE_VIDEO_TEXTURE_EMISSIVE
		emissiveColor = sRGBTransferEOTF( emissiveColor );
	#endif
	totalEmissiveRadiance *= emissiveColor.rgb;
#endif`,gg=`#ifdef USE_EMISSIVEMAP
	uniform sampler2D emissiveMap;
#endif`,_g="gl_FragColor = linearToOutputTexel( gl_FragColor );",xg=`vec4 LinearTransferOETF( in vec4 value ) {
	return value;
}
vec4 sRGBTransferEOTF( in vec4 value ) {
	return vec4( mix( pow( value.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), value.rgb * 0.0773993808, vec3( lessThanEqual( value.rgb, vec3( 0.04045 ) ) ) ), value.a );
}
vec4 sRGBTransferOETF( in vec4 value ) {
	return vec4( mix( pow( value.rgb, vec3( 0.41666 ) ) * 1.055 - vec3( 0.055 ), value.rgb * 12.92, vec3( lessThanEqual( value.rgb, vec3( 0.0031308 ) ) ) ), value.a );
}`,yg=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vec3 cameraToFrag;
		if ( isOrthographic ) {
			cameraToFrag = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToFrag = normalize( vWorldPosition - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vec3 reflectVec = reflect( cameraToFrag, worldNormal );
		#else
			vec3 reflectVec = refract( cameraToFrag, worldNormal, refractionRatio );
		#endif
	#else
		vec3 reflectVec = vReflect;
	#endif
	#ifdef ENVMAP_TYPE_CUBE
		vec4 envColor = textureCube( envMap, envMapRotation * reflectVec );
		#ifdef ENVMAP_BLENDING_MULTIPLY
			outgoingLight = mix( outgoingLight, outgoingLight * envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_MIX )
			outgoingLight = mix( outgoingLight, envColor.xyz, specularStrength * reflectivity );
		#elif defined( ENVMAP_BLENDING_ADD )
			outgoingLight += envColor.xyz * specularStrength * reflectivity;
		#endif
	#endif
#endif`,vg=`#ifdef USE_ENVMAP
	uniform float envMapIntensity;
	uniform mat3 envMapRotation;
	#ifdef ENVMAP_TYPE_CUBE
		uniform samplerCube envMap;
	#else
		uniform sampler2D envMap;
	#endif
#endif`,Mg=`#ifdef USE_ENVMAP
	uniform float reflectivity;
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		varying vec3 vWorldPosition;
		uniform float refractionRatio;
	#else
		varying vec3 vReflect;
	#endif
#endif`,bg=`#ifdef USE_ENVMAP
	#if defined( USE_BUMPMAP ) || defined( USE_NORMALMAP ) || defined( PHONG ) || defined( LAMBERT )
		#define ENV_WORLDPOS
	#endif
	#ifdef ENV_WORLDPOS
		
		varying vec3 vWorldPosition;
	#else
		varying vec3 vReflect;
		uniform float refractionRatio;
	#endif
#endif`,Sg=`#ifdef USE_ENVMAP
	#ifdef ENV_WORLDPOS
		vWorldPosition = worldPosition.xyz;
	#else
		vec3 cameraToVertex;
		if ( isOrthographic ) {
			cameraToVertex = normalize( vec3( - viewMatrix[ 0 ][ 2 ], - viewMatrix[ 1 ][ 2 ], - viewMatrix[ 2 ][ 2 ] ) );
		} else {
			cameraToVertex = normalize( worldPosition.xyz - cameraPosition );
		}
		vec3 worldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
		#ifdef ENVMAP_MODE_REFLECTION
			vReflect = reflect( cameraToVertex, worldNormal );
		#else
			vReflect = refract( cameraToVertex, worldNormal, refractionRatio );
		#endif
	#endif
#endif`,Eg=`#ifdef USE_FOG
	vFogDepth = - mvPosition.z;
#endif`,Tg=`#ifdef USE_FOG
	varying float vFogDepth;
#endif`,wg=`#ifdef USE_FOG
	#ifdef FOG_EXP2
		float fogFactor = 1.0 - exp( - fogDensity * fogDensity * vFogDepth * vFogDepth );
	#else
		float fogFactor = smoothstep( fogNear, fogFar, vFogDepth );
	#endif
	gl_FragColor.rgb = mix( gl_FragColor.rgb, fogColor, fogFactor );
#endif`,Ag=`#ifdef USE_FOG
	uniform vec3 fogColor;
	varying float vFogDepth;
	#ifdef FOG_EXP2
		uniform float fogDensity;
	#else
		uniform float fogNear;
		uniform float fogFar;
	#endif
#endif`,Rg=`#ifdef USE_GRADIENTMAP
	uniform sampler2D gradientMap;
#endif
vec3 getGradientIrradiance( vec3 normal, vec3 lightDirection ) {
	float dotNL = dot( normal, lightDirection );
	vec2 coord = vec2( dotNL * 0.5 + 0.5, 0.0 );
	#ifdef USE_GRADIENTMAP
		return vec3( texture2D( gradientMap, coord ).r );
	#else
		vec2 fw = fwidth( coord ) * 0.5;
		return mix( vec3( 0.7 ), vec3( 1.0 ), smoothstep( 0.7 - fw.x, 0.7 + fw.x, coord.x ) );
	#endif
}`,Cg=`#ifdef USE_LIGHTMAP
	uniform sampler2D lightMap;
	uniform float lightMapIntensity;
#endif`,Pg=`LambertMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularStrength = specularStrength;`,Ig=`varying vec3 vViewPosition;
struct LambertMaterial {
	vec3 diffuseColor;
	float specularStrength;
};
void RE_Direct_Lambert( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Lambert( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in LambertMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Lambert
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Lambert`,Lg=`uniform bool receiveShadow;
uniform vec3 ambientLightColor;
#if defined( USE_LIGHT_PROBES )
	uniform vec3 lightProbe[ 9 ];
#endif
vec3 shGetIrradianceAt( in vec3 normal, in vec3 shCoefficients[ 9 ] ) {
	float x = normal.x, y = normal.y, z = normal.z;
	vec3 result = shCoefficients[ 0 ] * 0.886227;
	result += shCoefficients[ 1 ] * 2.0 * 0.511664 * y;
	result += shCoefficients[ 2 ] * 2.0 * 0.511664 * z;
	result += shCoefficients[ 3 ] * 2.0 * 0.511664 * x;
	result += shCoefficients[ 4 ] * 2.0 * 0.429043 * x * y;
	result += shCoefficients[ 5 ] * 2.0 * 0.429043 * y * z;
	result += shCoefficients[ 6 ] * ( 0.743125 * z * z - 0.247708 );
	result += shCoefficients[ 7 ] * 2.0 * 0.429043 * x * z;
	result += shCoefficients[ 8 ] * 0.429043 * ( x * x - y * y );
	return result;
}
vec3 getLightProbeIrradiance( const in vec3 lightProbe[ 9 ], const in vec3 normal ) {
	vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec3 irradiance = shGetIrradianceAt( worldNormal, lightProbe );
	return irradiance;
}
vec3 getAmbientLightIrradiance( const in vec3 ambientLightColor ) {
	vec3 irradiance = ambientLightColor;
	return irradiance;
}
float getDistanceAttenuation( const in float lightDistance, const in float cutoffDistance, const in float decayExponent ) {
	float distanceFalloff = 1.0 / max( pow( lightDistance, decayExponent ), 0.01 );
	if ( cutoffDistance > 0.0 ) {
		distanceFalloff *= pow2( saturate( 1.0 - pow4( lightDistance / cutoffDistance ) ) );
	}
	return distanceFalloff;
}
float getSpotAttenuation( const in float coneCosine, const in float penumbraCosine, const in float angleCosine ) {
	return smoothstep( coneCosine, penumbraCosine, angleCosine );
}
#if NUM_SUN_LIGHTS > 0
	struct SunLight {
		vec3 direction;
		vec3 color;
	};
	uniform SunLight sunLights[ NUM_SUN_LIGHTS ];
	void getSunLightInfo( const in SunLight sunLight, out IncidentLight light ) {
		light.color = sunLight.color;
		light.direction = sunLight.direction;
		light.visible = true;
	}
#endif
#if NUM_DIR_LIGHTS > 0
	struct DirectionalLight {
		vec3 direction;
		vec3 color;
	};
	uniform DirectionalLight directionalLights[ NUM_DIR_LIGHTS ];
	void getDirectionalLightInfo( const in DirectionalLight directionalLight, out IncidentLight light ) {
		light.color = directionalLight.color;
		light.direction = directionalLight.direction;
		light.visible = true;
	}
#endif
#if NUM_POINT_LIGHTS > 0
	struct PointLight {
		vec3 position;
		vec3 color;
		float distance;
		float decay;
	};
	uniform PointLight pointLights[ NUM_POINT_LIGHTS ];
	void getPointLightInfo( const in PointLight pointLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = pointLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float lightDistance = length( lVector );
		light.color = pointLight.color;
		light.color *= getDistanceAttenuation( lightDistance, pointLight.distance, pointLight.decay );
		light.visible = ( light.color != vec3( 0.0 ) );
	}
#endif
#if NUM_SPOT_LIGHTS > 0
	struct SpotLight {
		vec3 position;
		vec3 direction;
		vec3 color;
		float distance;
		float decay;
		float coneCos;
		float penumbraCos;
	};
	uniform SpotLight spotLights[ NUM_SPOT_LIGHTS ];
	void getSpotLightInfo( const in SpotLight spotLight, const in vec3 geometryPosition, out IncidentLight light ) {
		vec3 lVector = spotLight.position - geometryPosition;
		light.direction = normalize( lVector );
		float angleCos = dot( light.direction, spotLight.direction );
		float spotAttenuation = getSpotAttenuation( spotLight.coneCos, spotLight.penumbraCos, angleCos );
		if ( spotAttenuation > 0.0 ) {
			float lightDistance = length( lVector );
			light.color = spotLight.color * spotAttenuation;
			light.color *= getDistanceAttenuation( lightDistance, spotLight.distance, spotLight.decay );
			light.visible = ( light.color != vec3( 0.0 ) );
		} else {
			light.color = vec3( 0.0 );
			light.visible = false;
		}
	}
#endif
#if NUM_RECT_AREA_LIGHTS > 0
	struct RectAreaLight {
		vec3 color;
		vec3 position;
		vec3 halfWidth;
		vec3 halfHeight;
	};
	uniform sampler2D ltc_1;	uniform sampler2D ltc_2;
	uniform RectAreaLight rectAreaLights[ NUM_RECT_AREA_LIGHTS ];
#endif
#if NUM_HEMI_LIGHTS > 0
	struct HemisphereLight {
		vec3 direction;
		vec3 skyColor;
		vec3 groundColor;
	};
	uniform HemisphereLight hemisphereLights[ NUM_HEMI_LIGHTS ];
	vec3 getHemisphereLightIrradiance( const in HemisphereLight hemiLight, const in vec3 normal ) {
		float dotNL = dot( normal, hemiLight.direction );
		float hemiDiffuseWeight = 0.5 * dotNL + 0.5;
		vec3 irradiance = mix( hemiLight.groundColor, hemiLight.skyColor, hemiDiffuseWeight );
		return irradiance;
	}
#endif
#include <lightprobes_pars_fragment>`,Dg=`#ifdef USE_ENVMAP
	vec3 getIBLIrradiance( const in vec3 normal ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 worldNormal = transformNormalByInverseViewMatrix( normal, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * worldNormal, 1.0 );
			return PI * envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	vec3 getIBLRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
		#ifdef ENVMAP_TYPE_CUBE_UV
			vec3 reflectVec = reflect( - viewDir, normal );
			reflectVec = normalize( mix( reflectVec, normal, pow4( roughness ) ) );
			reflectVec = transformDirectionByInverseViewMatrix( reflectVec, viewMatrix );
			vec4 envMapColor = textureCubeUV( envMap, envMapRotation * reflectVec, roughness );
			return envMapColor.rgb * envMapIntensity;
		#else
			return vec3( 0.0 );
		#endif
	}
	#ifdef USE_RETROREFLECTION
		vec3 getIBLRetroRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 retroVec = normalize( mix( viewDir, normal, pow4( roughness ) ) );
				retroVec = transformDirectionByInverseViewMatrix( retroVec, viewMatrix );
				vec4 envMapColor = textureCubeUV( envMap, envMapRotation * retroVec, roughness );
				return envMapColor.rgb * envMapIntensity;
			#else
				return vec3( 0.0 );
			#endif
		}
	#endif
	#ifdef USE_ANISOTROPY
		vec3 getIBLAnisotropyRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
			#ifdef ENVMAP_TYPE_CUBE_UV
				vec3 bentNormal = cross( bitangent, viewDir );
				bentNormal = normalize( cross( bentNormal, bitangent ) );
				bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
				return getIBLRadiance( viewDir, bentNormal, roughness );
			#else
				return vec3( 0.0 );
			#endif
		}
		#ifdef USE_RETROREFLECTION
			vec3 getIBLAnisotropyRetroRadiance( const in vec3 viewDir, const in vec3 normal, const in float roughness, const in vec3 bitangent, const in float anisotropy ) {
				#ifdef ENVMAP_TYPE_CUBE_UV
					vec3 bentNormal = cross( bitangent, viewDir );
					bentNormal = normalize( cross( bentNormal, bitangent ) );
					bentNormal = normalize( mix( bentNormal, normal, pow2( pow2( 1.0 - anisotropy * ( 1.0 - roughness ) ) ) ) );
					return getIBLRetroRadiance( viewDir, bentNormal, roughness );
				#else
					return vec3( 0.0 );
				#endif
			}
		#endif
	#endif
#endif`,Ng=`ToonMaterial material;
material.diffuseColor = diffuseColor.rgb;`,Ug=`varying vec3 vViewPosition;
struct ToonMaterial {
	vec3 diffuseColor;
};
void RE_Direct_Toon( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 irradiance = getGradientIrradiance( geometryNormal, directLight.direction ) * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
void RE_IndirectDiffuse_Toon( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in ToonMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_Toon
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Toon`,Fg=`BlinnPhongMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.specularColor = specular;
material.specularShininess = shininess;
material.specularStrength = specularStrength;`,Og=`varying vec3 vViewPosition;
struct BlinnPhongMaterial {
	vec3 diffuseColor;
	vec3 specularColor;
	float specularShininess;
	float specularStrength;
};
void RE_Direct_BlinnPhong( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
	reflectedLight.directSpecular += irradiance * BRDF_BlinnPhong( directLight.direction, geometryViewDir, geometryNormal, material.specularColor, material.specularShininess ) * material.specularStrength;
}
void RE_IndirectDiffuse_BlinnPhong( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in BlinnPhongMaterial material, inout ReflectedLight reflectedLight ) {
	reflectedLight.indirectDiffuse += irradiance * BRDF_Lambert( material.diffuseColor );
}
#define RE_Direct				RE_Direct_BlinnPhong
#define RE_IndirectDiffuse		RE_IndirectDiffuse_BlinnPhong`,Bg=`PhysicalMaterial material;
material.diffuseColor = diffuseColor.rgb;
material.diffuseContribution = diffuseColor.rgb * ( 1.0 - metalnessFactor );
material.metalness = metalnessFactor;
vec3 dxy = max( abs( dFdx( nonPerturbedNormal ) ), abs( dFdy( nonPerturbedNormal ) ) );
float geometryRoughness = max( max( dxy.x, dxy.y ), dxy.z );
material.roughness = max( roughnessFactor, 0.0525 );material.roughness += geometryRoughness;
material.roughness = min( material.roughness, 1.0 );
#ifdef IOR
	material.ior = ior;
	#ifdef USE_SPECULAR
		float specularIntensityFactor = specularIntensity;
		vec3 specularColorFactor = specularColor;
		#ifdef USE_SPECULAR_COLORMAP
			specularColorFactor *= texture2D( specularColorMap, vSpecularColorMapUv ).rgb;
		#endif
		#ifdef USE_SPECULAR_INTENSITYMAP
			specularIntensityFactor *= texture2D( specularIntensityMap, vSpecularIntensityMapUv ).a;
		#endif
		material.specularF90 = mix( specularIntensityFactor, 1.0, metalnessFactor );
	#else
		float specularIntensityFactor = 1.0;
		vec3 specularColorFactor = vec3( 1.0 );
		material.specularF90 = 1.0;
	#endif
	material.specularColor = min( pow2( ( material.ior - 1.0 ) / ( material.ior + 1.0 ) ) * specularColorFactor, vec3( 1.0 ) ) * specularIntensityFactor;
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
#else
	material.specularColor = vec3( 0.04 );
	material.specularColorBlended = mix( material.specularColor, diffuseColor.rgb, metalnessFactor );
	material.specularF90 = 1.0;
#endif
#ifdef USE_CLEARCOAT
	material.clearcoat = clearcoat;
	material.clearcoatRoughness = clearcoatRoughness;
	material.clearcoatF0 = vec3( 0.04 );
	material.clearcoatF90 = 1.0;
	#ifdef USE_CLEARCOATMAP
		material.clearcoat *= texture2D( clearcoatMap, vClearcoatMapUv ).x;
	#endif
	#ifdef USE_CLEARCOAT_ROUGHNESSMAP
		material.clearcoatRoughness *= texture2D( clearcoatRoughnessMap, vClearcoatRoughnessMapUv ).y;
	#endif
	material.clearcoat = saturate( material.clearcoat );	material.clearcoatRoughness = max( material.clearcoatRoughness, 0.0525 );
	material.clearcoatRoughness += geometryRoughness;
	material.clearcoatRoughness = min( material.clearcoatRoughness, 1.0 );
#endif
#ifdef USE_DISPERSION
	material.dispersion = dispersion;
#endif
#ifdef USE_RETROREFLECTION
	material.retroreflectivity = retroreflectivity;
#endif
#ifdef USE_IRIDESCENCE
	material.iridescence = iridescence;
	material.iridescenceIOR = iridescenceIOR;
	#ifdef USE_IRIDESCENCEMAP
		material.iridescence *= texture2D( iridescenceMap, vIridescenceMapUv ).r;
	#endif
	#ifdef USE_IRIDESCENCE_THICKNESSMAP
		material.iridescenceThickness = (iridescenceThicknessMaximum - iridescenceThicknessMinimum) * texture2D( iridescenceThicknessMap, vIridescenceThicknessMapUv ).g + iridescenceThicknessMinimum;
	#else
		material.iridescenceThickness = iridescenceThicknessMaximum;
	#endif
#endif
#ifdef USE_SHEEN
	material.sheenColor = sheenColor;
	#ifdef USE_SHEEN_COLORMAP
		material.sheenColor *= texture2D( sheenColorMap, vSheenColorMapUv ).rgb;
	#endif
	material.sheenRoughness = clamp( sheenRoughness, 0.0001, 1.0 );
	#ifdef USE_SHEEN_ROUGHNESSMAP
		material.sheenRoughness *= texture2D( sheenRoughnessMap, vSheenRoughnessMapUv ).a;
	#endif
#endif
#ifdef USE_ANISOTROPY
	#ifdef USE_ANISOTROPYMAP
		mat2 anisotropyMat = mat2( anisotropyVector.x, anisotropyVector.y, - anisotropyVector.y, anisotropyVector.x );
		vec3 anisotropyPolar = texture2D( anisotropyMap, vAnisotropyMapUv ).rgb;
		vec2 anisotropyV = anisotropyMat * normalize( 2.0 * anisotropyPolar.rg - vec2( 1.0 ) ) * anisotropyPolar.b;
	#else
		vec2 anisotropyV = anisotropyVector;
	#endif
	material.anisotropy = length( anisotropyV );
	if( material.anisotropy == 0.0 ) {
		anisotropyV = vec2( 1.0, 0.0 );
	} else {
		anisotropyV /= material.anisotropy;
		material.anisotropy = saturate( material.anisotropy );
	}
	material.alphaT = mix( pow2( material.roughness ), 1.0, pow2( material.anisotropy ) );
	material.anisotropyT = tbn[ 0 ] * anisotropyV.x + tbn[ 1 ] * anisotropyV.y;
	material.anisotropyB = tbn[ 1 ] * anisotropyV.x - tbn[ 0 ] * anisotropyV.y;
#endif`,zg=`uniform sampler2D dfgLUT;
struct PhysicalMaterial {
	vec3 diffuseColor;
	vec3 diffuseContribution;
	vec3 specularColor;
	vec3 specularColorBlended;
	float roughness;
	float metalness;
	float specularF90;
	float dispersion;
	vec2 dfg;
	vec3 multiScatteringCompensation;
	#ifdef USE_RETROREFLECTION
		float retroreflectivity;
	#endif
	#ifdef USE_CLEARCOAT
		float clearcoat;
		float clearcoatRoughness;
		vec3 clearcoatF0;
		float clearcoatF90;
	#endif
	#ifdef USE_IRIDESCENCE
		float iridescence;
		float iridescenceIOR;
		float iridescenceThickness;
		vec3 iridescenceFresnel;
		vec3 iridescenceF0Dielectric;
		vec3 iridescenceF0Metallic;
	#endif
	#ifdef USE_SHEEN
		vec3 sheenColor;
		float sheenRoughness;
	#endif
	#ifdef IOR
		float ior;
	#endif
	#ifdef USE_TRANSMISSION
		float transmission;
		float transmissionAlpha;
		float thickness;
		float attenuationDistance;
		vec3 attenuationColor;
	#endif
	#ifdef USE_ANISOTROPY
		float anisotropy;
		float alphaT;
		vec3 anisotropyT;
		vec3 anisotropyB;
	#endif
};
vec3 clearcoatSpecularDirect = vec3( 0.0 );
vec3 clearcoatSpecularIndirect = vec3( 0.0 );
vec3 sheenSpecularDirect = vec3( 0.0 );
vec3 sheenSpecularIndirect = vec3(0.0 );
vec3 Schlick_to_F0( const in vec3 f, const in float f90, const in float dotVH ) {
    float x = clamp( 1.0 - dotVH, 0.0, 1.0 );
    float x2 = x * x;
    float x5 = clamp( x * x2 * x2, 0.0, 0.9999 );
    return ( f - vec3( f90 ) * x5 ) / ( 1.0 - x5 );
}
float V_GGX_SmithCorrelated( const in float alpha, const in float dotNL, const in float dotNV ) {
	float a2 = pow2( alpha );
	float gv = dotNL * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNV ) );
	float gl = dotNV * sqrt( a2 + ( 1.0 - a2 ) * pow2( dotNL ) );
	return 0.5 / max( gv + gl, EPSILON );
}
float D_GGX( const in float alpha, const in float dotNH ) {
	float a2 = pow2( alpha );
	float denom = pow2( dotNH ) * ( a2 - 1.0 ) + 1.0;
	return RECIPROCAL_PI * a2 / pow2( denom );
}
#ifdef USE_ANISOTROPY
	float V_GGX_SmithCorrelated_Anisotropic( const in float alphaT, const in float alphaB, const in float dotTV, const in float dotBV, const in float dotTL, const in float dotBL, const in float dotNV, const in float dotNL ) {
		float gv = dotNL * length( vec3( alphaT * dotTV, alphaB * dotBV, dotNV ) );
		float gl = dotNV * length( vec3( alphaT * dotTL, alphaB * dotBL, dotNL ) );
		return 0.5 / max( gv + gl, EPSILON );
	}
	float D_GGX_Anisotropic( const in float alphaT, const in float alphaB, const in float dotNH, const in float dotTH, const in float dotBH ) {
		float a2 = alphaT * alphaB;
		highp vec3 v = vec3( alphaB * dotTH, alphaT * dotBH, a2 * dotNH );
		highp float v2 = dot( v, v );
		float w2 = a2 / v2;
		return RECIPROCAL_PI * a2 * pow2 ( w2 );
	}
#endif
#ifdef USE_CLEARCOAT
	vec3 BRDF_GGX_Clearcoat( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material) {
		vec3 f0 = material.clearcoatF0;
		float f90 = material.clearcoatF90;
		float roughness = material.clearcoatRoughness;
		float alpha = pow2( roughness );
		vec3 halfDir = normalize( lightDir + viewDir );
		float dotNL = saturate( dot( normal, lightDir ) );
		float dotNV = saturate( dot( normal, viewDir ) );
		float dotNH = saturate( dot( normal, halfDir ) );
		float dotVH = saturate( dot( viewDir, halfDir ) );
		vec3 F = F_Schlick( f0, f90, dotVH );
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
		return F * ( V * D );
	}
#endif
vec3 BRDF_GGX( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, const in PhysicalMaterial material ) {
	vec3 f0 = material.specularColorBlended;
	float f90 = material.specularF90;
	float roughness = material.roughness;
	float alpha = pow2( roughness );
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float dotVH = saturate( dot( viewDir, halfDir ) );
	vec3 F = F_Schlick( f0, f90, dotVH );
	#ifdef USE_IRIDESCENCE
		F = mix( F, material.iridescenceFresnel, material.iridescence );
	#endif
	#ifdef USE_ANISOTROPY
		float dotTL = dot( material.anisotropyT, lightDir );
		float dotTV = dot( material.anisotropyT, viewDir );
		float dotTH = dot( material.anisotropyT, halfDir );
		float dotBL = dot( material.anisotropyB, lightDir );
		float dotBV = dot( material.anisotropyB, viewDir );
		float dotBH = dot( material.anisotropyB, halfDir );
		float V = V_GGX_SmithCorrelated_Anisotropic( material.alphaT, alpha, dotTV, dotBV, dotTL, dotBL, dotNV, dotNL );
		float D = D_GGX_Anisotropic( material.alphaT, alpha, dotNH, dotTH, dotBH );
	#else
		float V = V_GGX_SmithCorrelated( alpha, dotNL, dotNV );
		float D = D_GGX( alpha, dotNH );
	#endif
	return F * ( V * D );
}
vec2 LTC_Uv( const in vec3 N, const in vec3 V, const in float roughness ) {
	const float LUT_SIZE = 64.0;
	const float LUT_SCALE = ( LUT_SIZE - 1.0 ) / LUT_SIZE;
	const float LUT_BIAS = 0.5 / LUT_SIZE;
	float dotNV = saturate( dot( N, V ) );
	vec2 uv = vec2( roughness, sqrt( 1.0 - dotNV ) );
	uv = uv * LUT_SCALE + LUT_BIAS;
	return uv;
}
float LTC_ClippedSphereFormFactor( const in vec3 f ) {
	float l = length( f );
	return max( ( l * l + f.z ) / ( l + 1.0 ), 0.0 );
}
vec3 LTC_EdgeVectorFormFactor( const in vec3 v1, const in vec3 v2 ) {
	float x = dot( v1, v2 );
	float y = abs( x );
	float a = 0.8543985 + ( 0.4965155 + 0.0145206 * y ) * y;
	float b = 3.4175940 + ( 4.1616724 + y ) * y;
	float v = a / b;
	float theta_sintheta = ( x > 0.0 ) ? v : 0.5 * inversesqrt( max( 1.0 - x * x, 1e-7 ) ) - v;
	return cross( v1, v2 ) * theta_sintheta;
}
vec3 LTC_Evaluate( const in vec3 N, const in vec3 V, const in vec3 P, const in mat3 mInv, const in vec3 rectCoords[ 4 ] ) {
	vec3 v1 = rectCoords[ 1 ] - rectCoords[ 0 ];
	vec3 v2 = rectCoords[ 3 ] - rectCoords[ 0 ];
	vec3 lightNormal = cross( v1, v2 );
	if( dot( lightNormal, P - rectCoords[ 0 ] ) < 0.0 ) return vec3( 0.0 );
	vec3 T1, T2;
	T1 = normalize( V - N * dot( V, N ) );
	T2 = - cross( N, T1 );
	mat3 mat = mInv * transpose( mat3( T1, T2, N ) );
	vec3 coords[ 4 ];
	coords[ 0 ] = mat * ( rectCoords[ 0 ] - P );
	coords[ 1 ] = mat * ( rectCoords[ 1 ] - P );
	coords[ 2 ] = mat * ( rectCoords[ 2 ] - P );
	coords[ 3 ] = mat * ( rectCoords[ 3 ] - P );
	coords[ 0 ] = normalize( coords[ 0 ] );
	coords[ 1 ] = normalize( coords[ 1 ] );
	coords[ 2 ] = normalize( coords[ 2 ] );
	coords[ 3 ] = normalize( coords[ 3 ] );
	vec3 vectorFormFactor = vec3( 0.0 );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 0 ], coords[ 1 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 1 ], coords[ 2 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 2 ], coords[ 3 ] );
	vectorFormFactor += LTC_EdgeVectorFormFactor( coords[ 3 ], coords[ 0 ] );
	float result = LTC_ClippedSphereFormFactor( vectorFormFactor );
	return vec3( result );
}
#if defined( USE_SHEEN )
float D_Charlie( float roughness, float dotNH ) {
	float alpha = pow2( roughness );
	float invAlpha = 1.0 / alpha;
	float cos2h = dotNH * dotNH;
	float sin2h = max( 1.0 - cos2h, 0.0078125 );
	return ( 2.0 + invAlpha ) * pow( sin2h, invAlpha * 0.5 ) / ( 2.0 * PI );
}
float V_Neubelt( float dotNV, float dotNL ) {
	return saturate( 1.0 / ( 4.0 * ( dotNL + dotNV - dotNL * dotNV ) ) );
}
vec3 BRDF_Sheen( const in vec3 lightDir, const in vec3 viewDir, const in vec3 normal, vec3 sheenColor, const in float sheenRoughness ) {
	vec3 halfDir = normalize( lightDir + viewDir );
	float dotNL = saturate( dot( normal, lightDir ) );
	float dotNV = saturate( dot( normal, viewDir ) );
	float dotNH = saturate( dot( normal, halfDir ) );
	float D = D_Charlie( sheenRoughness, dotNH );
	float V = V_Neubelt( dotNV, dotNL );
	return sheenColor * ( D * V );
}
#endif
float IBLSheenBRDF( const in vec3 normal, const in vec3 viewDir, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	float r2 = roughness * roughness;
	float rInv = 1.0 / ( roughness + 0.1 );
	float a = -1.9362 + 1.0678 * roughness + 0.4573 * r2 - 0.8469 * rInv;
	float b = -0.6014 + 0.5538 * roughness - 0.4670 * r2 - 0.1255 * rInv;
	float DG = exp( a * dotNV + b );
	return saturate( DG );
}
vec3 EnvironmentBRDF( const in vec3 normal, const in vec3 viewDir, const in vec3 specularColor, const in float specularF90, const in float roughness ) {
	float dotNV = saturate( dot( normal, viewDir ) );
	vec2 fab = texture2D( dfgLUT, vec2( roughness, dotNV ) ).rg;
	return specularColor * fab.x + specularF90 * fab.y;
}
#ifdef USE_IRIDESCENCE
void computeMultiscatteringIridescence( const in vec2 fab, const in vec3 specularColor, const in float specularF90, const in float iridescence, const in vec3 iridescenceF0, inout vec3 singleScatter, inout vec3 multiScatter ) {
#else
void computeMultiscattering( const in vec2 fab, const in vec3 specularColor, const in float specularF90, inout vec3 singleScatter, inout vec3 multiScatter ) {
#endif
	#ifdef USE_IRIDESCENCE
		vec3 Fr = mix( specularColor, iridescenceF0, iridescence );
	#else
		vec3 Fr = specularColor;
	#endif
	vec3 FssEss = Fr * fab.x + specularF90 * fab.y;
	float Ess = fab.x + fab.y;
	float Ems = 1.0 - Ess;
	vec3 Favg = Fr + ( 1.0 - Fr ) * 0.047619;	vec3 Fms = FssEss * Favg / ( 1.0 - Ems * Favg );
	singleScatter += FssEss;
	multiScatter += Fms * Ems;
}
#if NUM_RECT_AREA_LIGHTS > 0
	void RE_Direct_RectArea_Physical( const in RectAreaLight rectAreaLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
		vec3 normal = geometryNormal;
		vec3 viewDir = geometryViewDir;
		vec3 position = geometryPosition;
		vec3 lightPos = rectAreaLight.position;
		vec3 halfWidth = rectAreaLight.halfWidth;
		vec3 halfHeight = rectAreaLight.halfHeight;
		vec3 lightColor = rectAreaLight.color;
		float roughness = material.roughness;
		vec3 rectCoords[ 4 ];
		rectCoords[ 0 ] = lightPos + halfWidth - halfHeight;		rectCoords[ 1 ] = lightPos - halfWidth - halfHeight;
		rectCoords[ 2 ] = lightPos - halfWidth + halfHeight;
		rectCoords[ 3 ] = lightPos + halfWidth + halfHeight;
		vec2 uv = LTC_Uv( normal, viewDir, roughness );
		vec4 t1 = texture2D( ltc_1, uv );
		vec4 t2 = texture2D( ltc_2, uv );
		mat3 mInv = mat3(
			vec3( t1.x, 0, t1.y ),
			vec3(    0, 1,    0 ),
			vec3( t1.z, 0, t1.w )
		);
		vec3 fresnel = ( material.specularColorBlended * t2.x + ( material.specularF90 - material.specularColorBlended ) * t2.y );
		reflectedLight.directSpecular += lightColor * fresnel * LTC_Evaluate( normal, viewDir, position, mInv, rectCoords );
		reflectedLight.directDiffuse += lightColor * material.diffuseContribution * LTC_Evaluate( normal, viewDir, position, mat3( 1.0 ), rectCoords );
		#ifdef USE_CLEARCOAT
			vec3 Ncc = geometryClearcoatNormal;
			vec2 uvClearcoat = LTC_Uv( Ncc, viewDir, material.clearcoatRoughness );
			vec4 t1Clearcoat = texture2D( ltc_1, uvClearcoat );
			vec4 t2Clearcoat = texture2D( ltc_2, uvClearcoat );
			mat3 mInvClearcoat = mat3(
				vec3( t1Clearcoat.x, 0, t1Clearcoat.y ),
				vec3(             0, 1,             0 ),
				vec3( t1Clearcoat.z, 0, t1Clearcoat.w )
			);
			vec3 fresnelClearcoat = material.clearcoatF0 * t2Clearcoat.x + ( material.clearcoatF90 - material.clearcoatF0 ) * t2Clearcoat.y;
			clearcoatSpecularDirect += lightColor * fresnelClearcoat * LTC_Evaluate( Ncc, viewDir, position, mInvClearcoat, rectCoords );
		#endif
	}
#endif
void RE_Direct_Physical( const in IncidentLight directLight, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	float dotNL = saturate( dot( geometryNormal, directLight.direction ) );
	vec3 irradiance = dotNL * directLight.color;
	#ifdef USE_CLEARCOAT
		float dotNLcc = saturate( dot( geometryClearcoatNormal, directLight.direction ) );
		vec3 ccIrradiance = dotNLcc * directLight.color;
		clearcoatSpecularDirect += ccIrradiance * BRDF_GGX_Clearcoat( directLight.direction, geometryViewDir, geometryClearcoatNormal, material );
	#endif
	#ifdef USE_SHEEN
 
 		sheenSpecularDirect += irradiance * BRDF_Sheen( directLight.direction, geometryViewDir, geometryNormal, material.sheenColor, material.sheenRoughness );
 
 		float sheenAlbedoV = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
 		float sheenAlbedoL = IBLSheenBRDF( geometryNormal, directLight.direction, material.sheenRoughness );
 
 		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * max( sheenAlbedoV, sheenAlbedoL );
 
 		irradiance *= sheenEnergyComp;
 
 	#endif
	vec3 specularBRDF = BRDF_GGX( directLight.direction, geometryViewDir, geometryNormal, material );
	#ifdef USE_RETROREFLECTION
		vec3 retroViewDir = reflect( - geometryViewDir, geometryNormal );
		vec3 retroSpecularBRDF = BRDF_GGX( directLight.direction, retroViewDir, geometryNormal, material );
		specularBRDF = mix( specularBRDF, retroSpecularBRDF, saturate( material.retroreflectivity ) );
	#endif
	reflectedLight.directSpecular += irradiance * specularBRDF * material.multiScatteringCompensation;
	vec3 halfDir = normalize( directLight.direction + geometryViewDir );
	float dotVH = saturate( dot( geometryViewDir, halfDir ) );
	vec3 F = F_Schlick( material.specularColor, material.specularF90, dotVH );
	#ifdef USE_RETROREFLECTION
		vec3 retroHalfDir = normalize( directLight.direction + retroViewDir );
		float dotRetroVH = saturate( dot( retroViewDir, retroHalfDir ) );
		vec3 retroF = F_Schlick( material.specularColor, material.specularF90, dotRetroVH );
		F = mix( F, retroF, saturate( material.retroreflectivity ) );
	#endif
	reflectedLight.directDiffuse += irradiance * BRDF_Lambert( material.diffuseContribution ) * ( 1.0 - F );
}
void RE_IndirectDiffuse_Physical( const in vec3 irradiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight ) {
	vec3 singleScattering = vec3( 0.0 );
	vec3 multiScattering = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( material.dfg, material.specularColor, material.specularF90, material.iridescence, material.iridescenceF0Dielectric, singleScattering, multiScattering );
	#else
		computeMultiscattering( material.dfg, material.specularColor, material.specularF90, singleScattering, multiScattering );
	#endif
	vec3 diffuse = irradiance * BRDF_Lambert( material.diffuseContribution ) * ( 1.0 - singleScattering - multiScattering );
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		sheenSpecularIndirect += irradiance * material.sheenColor * sheenAlbedo * RECIPROCAL_PI;
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		diffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectDiffuse += diffuse;
}
void RE_IndirectSpecular_Physical( const in vec3 radiance, const in vec3 irradiance, const in vec3 clearcoatRadiance, const in vec3 geometryPosition, const in vec3 geometryNormal, const in vec3 geometryViewDir, const in vec3 geometryClearcoatNormal, const in PhysicalMaterial material, inout ReflectedLight reflectedLight) {
	#ifdef USE_CLEARCOAT
		clearcoatSpecularIndirect += clearcoatRadiance * EnvironmentBRDF( geometryClearcoatNormal, geometryViewDir, material.clearcoatF0, material.clearcoatF90, material.clearcoatRoughness );
	#endif
	#ifdef USE_SHEEN
		sheenSpecularIndirect += irradiance * material.sheenColor * IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness ) * RECIPROCAL_PI;
 	#endif
	vec3 singleScatteringDielectric = vec3( 0.0 );
	vec3 multiScatteringDielectric = vec3( 0.0 );
	vec3 singleScatteringMetallic = vec3( 0.0 );
	vec3 multiScatteringMetallic = vec3( 0.0 );
	#ifdef USE_IRIDESCENCE
		computeMultiscatteringIridescence( material.dfg, material.specularColor, material.specularF90, material.iridescence, material.iridescenceF0Dielectric, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscatteringIridescence( material.dfg, material.diffuseColor, material.specularF90, material.iridescence, material.iridescenceF0Metallic, singleScatteringMetallic, multiScatteringMetallic );
	#else
		computeMultiscattering( material.dfg, material.specularColor, material.specularF90, singleScatteringDielectric, multiScatteringDielectric );
		computeMultiscattering( material.dfg, material.diffuseColor, material.specularF90, singleScatteringMetallic, multiScatteringMetallic );
	#endif
	vec3 singleScattering = mix( singleScatteringDielectric, singleScatteringMetallic, material.metalness );
	vec3 multiScattering = mix( multiScatteringDielectric, multiScatteringMetallic, material.metalness );
	vec3 totalScatteringDielectric = singleScatteringDielectric + multiScatteringDielectric;
	vec3 diffuse = material.diffuseContribution * ( 1.0 - totalScatteringDielectric );
	vec3 cosineWeightedIrradiance = irradiance * RECIPROCAL_PI;
	vec3 indirectSpecular = radiance * singleScattering;
	indirectSpecular += multiScattering * cosineWeightedIrradiance;
	vec3 indirectDiffuse = diffuse * cosineWeightedIrradiance;
	#ifdef USE_SHEEN
		float sheenAlbedo = IBLSheenBRDF( geometryNormal, geometryViewDir, material.sheenRoughness );
		float sheenEnergyComp = 1.0 - max3( material.sheenColor ) * sheenAlbedo;
		indirectSpecular *= sheenEnergyComp;
		indirectDiffuse *= sheenEnergyComp;
	#endif
	reflectedLight.indirectSpecular += indirectSpecular;
	reflectedLight.indirectDiffuse += indirectDiffuse;
}
#define RE_Direct				RE_Direct_Physical
#define RE_Direct_RectArea		RE_Direct_RectArea_Physical
#define RE_IndirectDiffuse		RE_IndirectDiffuse_Physical
#define RE_IndirectSpecular		RE_IndirectSpecular_Physical
float computeSpecularOcclusion( const in float dotNV, const in float ambientOcclusion, const in float roughness ) {
	return saturate( pow( dotNV + ambientOcclusion, exp2( - 16.0 * roughness - 1.0 ) ) - 1.0 + ambientOcclusion );
}`,kg=`
vec3 geometryPosition = - vViewPosition;
vec3 geometryNormal = normal;
vec3 geometryViewDir = ( isOrthographic ) ? vec3( 0, 0, 1 ) : normalize( vViewPosition );
vec3 geometryClearcoatNormal = vec3( 0.0 );
#ifdef USE_CLEARCOAT
	geometryClearcoatNormal = clearcoatNormal;
#endif
#ifdef USE_IRIDESCENCE
	float dotNVi = saturate( dot( normal, geometryViewDir ) );
	if ( material.iridescenceThickness == 0.0 ) {
		material.iridescence = 0.0;
	} else {
		material.iridescence = saturate( material.iridescence );
	}
	if ( material.iridescence > 0.0 ) {
		vec3 iridescenceFresnelDielectric = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.specularColor );
		vec3 iridescenceFresnelMetallic = evalIridescence( 1.0, material.iridescenceIOR, dotNVi, material.iridescenceThickness, material.diffuseColor );
		material.iridescenceFresnel = mix( iridescenceFresnelDielectric, iridescenceFresnelMetallic, material.metalness );
		material.iridescenceF0Dielectric = Schlick_to_F0( iridescenceFresnelDielectric, 1.0, dotNVi );
		material.iridescenceF0Metallic = Schlick_to_F0( iridescenceFresnelMetallic, 1.0, dotNVi );
	}
#endif
#ifdef STANDARD
	float dotNVms = saturate( dot( geometryNormal, geometryViewDir ) );
	material.dfg = texture2D( dfgLUT, vec2( material.roughness, dotNVms ) ).rg;
	#if ( NUM_SUN_LIGHTS > 0 || NUM_DIR_LIGHTS > 0 || NUM_POINT_LIGHTS > 0 || NUM_SPOT_LIGHTS > 0 )
		float EssMs = material.dfg.x + material.dfg.y;
		material.multiScatteringCompensation = 1.0 + material.specularColorBlended * ( 1.0 / EssMs - 1.0 );
	#endif
#endif
IncidentLight directLight;
#if ( NUM_POINT_LIGHTS > 0 ) && defined( RE_Direct )
	PointLight pointLight;
	#if defined( USE_SHADOWMAP ) && NUM_POINT_LIGHT_SHADOWS > 0
	PointLightShadow pointLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHTS; i ++ ) {
		pointLight = pointLights[ i ];
		getPointLightInfo( pointLight, geometryPosition, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_POINT_LIGHT_SHADOWS ) && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
		pointLightShadow = pointLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getPointShadow( pointShadowMap[ i ], pointLightShadow.shadowMapSize, pointLightShadow.shadowIntensity, pointLightShadow.shadowBias, pointLightShadow.shadowRadius, vPointShadowCoord[ i ], pointLightShadow.shadowCameraNear, pointLightShadow.shadowCameraFar ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SPOT_LIGHTS > 0 ) && defined( RE_Direct )
	SpotLight spotLight;
	vec4 spotColor;
	vec3 spotLightCoord;
	bool inSpotLightMap;
	#if defined( USE_SHADOWMAP ) && NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHTS; i ++ ) {
		spotLight = spotLights[ i ];
		getSpotLightInfo( spotLight, geometryPosition, directLight );
		#if ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#define SPOT_LIGHT_MAP_INDEX UNROLLED_LOOP_INDEX
		#elif ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		#define SPOT_LIGHT_MAP_INDEX NUM_SPOT_LIGHT_MAPS
		#else
		#define SPOT_LIGHT_MAP_INDEX ( UNROLLED_LOOP_INDEX - NUM_SPOT_LIGHT_SHADOWS + NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS )
		#endif
		#if ( SPOT_LIGHT_MAP_INDEX < NUM_SPOT_LIGHT_MAPS )
			spotLightCoord = vSpotLightCoord[ i ].xyz / vSpotLightCoord[ i ].w;
			inSpotLightMap = all( lessThan( abs( spotLightCoord * 2. - 1. ), vec3( 1.0 ) ) );
			spotColor = texture2D( spotLightMap[ SPOT_LIGHT_MAP_INDEX ], spotLightCoord.xy );
			directLight.color = inSpotLightMap ? directLight.color * spotColor.rgb : directLight.color;
		#endif
		#undef SPOT_LIGHT_MAP_INDEX
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
		spotLightShadow = spotLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( spotShadowMap[ i ], spotLightShadow.shadowMapSize, spotLightShadow.shadowIntensity, spotLightShadow.shadowBias, spotLightShadow.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_SUN_LIGHTS > 0 ) && defined( RE_Direct )
	SunLight sunLight;
	#if defined( USE_SHADOWMAP ) && NUM_SUN_LIGHT_SHADOWS > 0
	SunLightShadow sunLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SUN_LIGHTS; i ++ ) {
		sunLight = sunLights[ i ];
		getSunLightInfo( sunLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_SUN_LIGHT_SHADOWS )
		sunLightShadow = sunLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getSunShadow( sunShadowMap[ i ], sunLightShadow, UNROLLED_LOOP_INDEX ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_DIR_LIGHTS > 0 ) && defined( RE_Direct )
	DirectionalLight directionalLight;
	#if defined( USE_SHADOWMAP ) && NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLightShadow;
	#endif
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHTS; i ++ ) {
		directionalLight = directionalLights[ i ];
		getDirectionalLightInfo( directionalLight, directLight );
		#if defined( USE_SHADOWMAP ) && ( UNROLLED_LOOP_INDEX < NUM_DIR_LIGHT_SHADOWS )
		directionalLightShadow = directionalLightShadows[ i ];
		directLight.color *= ( directLight.visible && receiveShadow ) ? getShadow( directionalShadowMap[ i ], directionalLightShadow.shadowMapSize, directionalLightShadow.shadowIntensity, directionalLightShadow.shadowBias, directionalLightShadow.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
		#endif
		RE_Direct( directLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if ( NUM_RECT_AREA_LIGHTS > 0 ) && defined( RE_Direct_RectArea )
	RectAreaLight rectAreaLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_RECT_AREA_LIGHTS; i ++ ) {
		rectAreaLight = rectAreaLights[ i ];
		RE_Direct_RectArea( rectAreaLight, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
	}
	#pragma unroll_loop_end
#endif
#if defined( RE_IndirectDiffuse )
	vec3 iblIrradiance = vec3( 0.0 );
	vec3 irradiance = getAmbientLightIrradiance( ambientLightColor );
	#if defined( USE_LIGHT_PROBES )
		irradiance += getLightProbeIrradiance( lightProbe, geometryNormal );
	#endif
	#if ( NUM_HEMI_LIGHTS > 0 )
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_HEMI_LIGHTS; i ++ ) {
			irradiance += getHemisphereLightIrradiance( hemisphereLights[ i ], geometryNormal );
		}
		#pragma unroll_loop_end
	#endif
	#ifdef USE_LIGHT_PROBES_GRID
		vec3 probeWorldPos = ( ( vec4( geometryPosition, 1.0 ) - viewMatrix[ 3 ] ) * viewMatrix ).xyz;
		vec3 probeWorldNormal = transformNormalByInverseViewMatrix( geometryNormal, viewMatrix );
		irradiance += getLightProbeGridIrradiance( probeWorldPos, probeWorldNormal );
	#endif
#endif
#if defined( RE_IndirectSpecular )
	vec3 radiance = vec3( 0.0 );
	vec3 clearcoatRadiance = vec3( 0.0 );
#endif`,Vg=`#if defined( RE_IndirectDiffuse )
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		vec3 lightMapIrradiance = lightMapTexel.rgb * lightMapIntensity;
		irradiance += lightMapIrradiance;
	#endif
	#if defined( USE_ENVMAP ) && defined( ENVMAP_TYPE_CUBE_UV )
		#if defined( STANDARD ) || defined( LAMBERT ) || defined( PHONG )
			iblIrradiance += getIBLIrradiance( geometryNormal );
		#endif
	#endif
#endif
#if defined( USE_ENVMAP ) && defined( RE_IndirectSpecular )
	#ifdef USE_ANISOTROPY
		vec3 iblRadiance = getIBLAnisotropyRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
	#else
		vec3 iblRadiance = getIBLRadiance( geometryViewDir, geometryNormal, material.roughness );
	#endif
	#ifdef USE_RETROREFLECTION
		#ifdef USE_ANISOTROPY
			vec3 retroIBLRadiance = getIBLAnisotropyRetroRadiance( geometryViewDir, geometryNormal, material.roughness, material.anisotropyB, material.anisotropy );
		#else
			vec3 retroIBLRadiance = getIBLRetroRadiance( geometryViewDir, geometryNormal, material.roughness );
		#endif
		iblRadiance = mix( iblRadiance, retroIBLRadiance, saturate( material.retroreflectivity ) );
	#endif
	radiance += iblRadiance;
	#ifdef USE_CLEARCOAT
		clearcoatRadiance += getIBLRadiance( geometryViewDir, geometryClearcoatNormal, material.clearcoatRoughness );
	#endif
#endif`,Hg=`#if defined( RE_IndirectDiffuse )
	#if defined( LAMBERT ) || defined( PHONG )
		irradiance += iblIrradiance;
	#endif
	RE_IndirectDiffuse( irradiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif
#if defined( RE_IndirectSpecular )
	RE_IndirectSpecular( radiance, iblIrradiance, clearcoatRadiance, geometryPosition, geometryNormal, geometryViewDir, geometryClearcoatNormal, material, reflectedLight );
#endif`,Gg=`#ifdef USE_LIGHT_PROBES_GRID
uniform highp sampler3D probesSH;
uniform vec3 probesMin;
uniform vec3 probesMax;
uniform vec3 probesResolution;
vec3 getLightProbeGridIrradiance( vec3 worldPos, vec3 worldNormal ) {
	vec3 res = probesResolution;
	vec3 gridRange = probesMax - probesMin;
	vec3 resMinusOne = res - 1.0;
	vec3 probeSpacing = gridRange / resMinusOne;
	vec3 samplePos = worldPos + worldNormal * probeSpacing * 0.5;
	vec3 uvw = clamp( ( samplePos - probesMin ) / gridRange, 0.0, 1.0 );
	uvw = uvw * resMinusOne / res + 0.5 / res;
	float nz          = res.z;
	float paddedSlices = nz + 2.0;
	float atlasDepth  = 7.0 * paddedSlices;
	float uvZBase     = uvw.z * nz + 1.0;
	vec4 s0 = texture( probesSH, vec3( uvw.xy, ( uvZBase                       ) / atlasDepth ) );
	vec4 s1 = texture( probesSH, vec3( uvw.xy, ( uvZBase +       paddedSlices   ) / atlasDepth ) );
	vec4 s2 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 2.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s3 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 3.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s4 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 4.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s5 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 5.0 * paddedSlices   ) / atlasDepth ) );
	vec4 s6 = texture( probesSH, vec3( uvw.xy, ( uvZBase + 6.0 * paddedSlices   ) / atlasDepth ) );
	vec3 c0 = s0.xyz;
	vec3 c1 = vec3( s0.w, s1.xy );
	vec3 c2 = vec3( s1.zw, s2.x );
	vec3 c3 = s2.yzw;
	vec3 c4 = s3.xyz;
	vec3 c5 = vec3( s3.w, s4.xy );
	vec3 c6 = vec3( s4.zw, s5.x );
	vec3 c7 = s5.yzw;
	vec3 c8 = s6.xyz;
	float x = worldNormal.x, y = worldNormal.y, z = worldNormal.z;
	vec3 result = c0 * 0.886227;
	result += c1 * 2.0 * 0.511664 * y;
	result += c2 * 2.0 * 0.511664 * z;
	result += c3 * 2.0 * 0.511664 * x;
	result += c4 * 2.0 * 0.429043 * x * y;
	result += c5 * 2.0 * 0.429043 * y * z;
	result += c6 * ( 0.743125 * z * z - 0.247708 );
	result += c7 * 2.0 * 0.429043 * x * z;
	result += c8 * 0.429043 * ( x * x - y * y );
	return max( result, vec3( 0.0 ) );
}
#endif`,Wg=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	gl_FragDepth = vIsPerspective == 0.0 ? gl_FragCoord.z : log2( vFragDepth ) * logDepthBufFC * 0.5;
#endif`,Xg=`#if defined( USE_LOGARITHMIC_DEPTH_BUFFER )
	uniform float logDepthBufFC;
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,qg=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	varying float vFragDepth;
	varying float vIsPerspective;
#endif`,Yg=`#ifdef USE_LOGARITHMIC_DEPTH_BUFFER
	vFragDepth = 1.0 + gl_Position.w;
	vIsPerspective = float( isPerspectiveMatrix( projectionMatrix ) );
#endif`,Zg=`#ifdef USE_MAP
	vec4 sampledDiffuseColor = texture2D( map, vMapUv );
	#ifdef DECODE_VIDEO_TEXTURE
		sampledDiffuseColor = sRGBTransferEOTF( sampledDiffuseColor );
	#endif
	diffuseColor *= sampledDiffuseColor;
#endif`,$g=`#ifdef USE_MAP
	uniform sampler2D map;
#endif`,Jg=`#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
	#if defined( USE_POINTS_UV )
		vec2 uv = vUv;
	#else
		vec2 uv = ( uvTransform * vec3( gl_PointCoord.x, 1.0 - gl_PointCoord.y, 1 ) ).xy;
	#endif
#endif
#ifdef USE_MAP
	diffuseColor *= texture2D( map, uv );
#endif
#ifdef USE_ALPHAMAP
	diffuseColor.a *= texture2D( alphaMap, uv ).g;
#endif`,jg=`#if defined( USE_POINTS_UV )
	varying vec2 vUv;
#else
	#if defined( USE_MAP ) || defined( USE_ALPHAMAP )
		uniform mat3 uvTransform;
	#endif
#endif
#ifdef USE_MAP
	uniform sampler2D map;
#endif
#ifdef USE_ALPHAMAP
	uniform sampler2D alphaMap;
#endif`,Kg=`float metalnessFactor = metalness;
#ifdef USE_METALNESSMAP
	vec4 texelMetalness = texture2D( metalnessMap, vMetalnessMapUv );
	metalnessFactor *= texelMetalness.b;
#endif`,Qg=`#ifdef USE_METALNESSMAP
	uniform sampler2D metalnessMap;
#endif`,e0=`#ifdef USE_INSTANCING_MORPH
	float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	float morphTargetBaseInfluence = texelFetch( morphTexture, ivec2( 0, gl_InstanceID ), 0 ).r;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		morphTargetInfluences[i] =  texelFetch( morphTexture, ivec2( i + 1, gl_InstanceID ), 0 ).r;
	}
#endif`,t0=`#if defined( USE_MORPHCOLORS )
	vColor *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		#if defined( USE_COLOR_ALPHA )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ) * morphTargetInfluences[ i ];
		#elif defined( USE_COLOR )
			if ( morphTargetInfluences[ i ] != 0.0 ) vColor += getMorph( gl_VertexID, i, 2 ).rgb * morphTargetInfluences[ i ];
		#endif
	}
#endif`,n0=`#ifdef USE_MORPHNORMALS
	objectNormal *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) objectNormal += getMorph( gl_VertexID, i, 1 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,i0=`#ifdef USE_MORPHTARGETS
	#ifndef USE_INSTANCING_MORPH
		uniform float morphTargetBaseInfluence;
		uniform float morphTargetInfluences[ MORPHTARGETS_COUNT ];
	#endif
	uniform sampler2DArray morphTargetsTexture;
	uniform ivec2 morphTargetsTextureSize;
	vec4 getMorph( const in int vertexIndex, const in int morphTargetIndex, const in int offset ) {
		int texelIndex = vertexIndex * MORPHTARGETS_TEXTURE_STRIDE + offset;
		int y = texelIndex / morphTargetsTextureSize.x;
		int x = texelIndex - y * morphTargetsTextureSize.x;
		ivec3 morphUV = ivec3( x, y, morphTargetIndex );
		return texelFetch( morphTargetsTexture, morphUV, 0 );
	}
#endif`,s0=`#ifdef USE_MORPHTARGETS
	transformed *= morphTargetBaseInfluence;
	for ( int i = 0; i < MORPHTARGETS_COUNT; i ++ ) {
		if ( morphTargetInfluences[ i ] != 0.0 ) transformed += getMorph( gl_VertexID, i, 0 ).xyz * morphTargetInfluences[ i ];
	}
#endif`,r0=`float faceDirection = gl_FrontFacing ? 1.0 : - 1.0;
#ifdef FLAT_SHADED
	vec3 fdx = dFdx( vViewPosition );
	vec3 fdy = dFdy( vViewPosition );
	vec3 normal = normalize( cross( fdx, fdy ) );
#else
	vec3 normal = normalize( vNormal );
	#ifdef DOUBLE_SIDED
		normal *= faceDirection;
	#endif
#endif
#if defined( USE_NORMALMAP_TANGENTSPACE ) || defined( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY )
	#ifdef USE_TANGENT
		mat3 tbn = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn = getTangentFrame( - vViewPosition, normal,
		#if defined( USE_NORMALMAP )
			vNormalMapUv
		#elif defined( USE_CLEARCOAT_NORMALMAP )
			vClearcoatNormalMapUv
		#else
			vUv
		#endif
		);
	#endif
	#ifdef DOUBLE_SIDED
		tbn[0] *= faceDirection;
		tbn[1] *= faceDirection;
	#endif
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	#ifdef USE_TANGENT
		mat3 tbn2 = mat3( normalize( vTangent ), normalize( vBitangent ), normal );
	#else
		mat3 tbn2 = getTangentFrame( - vViewPosition, normal, vClearcoatNormalMapUv );
	#endif
	#ifdef DOUBLE_SIDED
		tbn2[0] *= faceDirection;
		tbn2[1] *= faceDirection;
	#endif
#endif
vec3 nonPerturbedNormal = normal;`,a0=`#ifdef USE_NORMALMAP_OBJECTSPACE
	normal = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#ifdef FLIP_SIDED
		normal = - normal;
	#endif
	#ifdef DOUBLE_SIDED
		normal = normal * faceDirection;
	#endif
	normal = normalize( normalMatrix * normal );
#elif defined( USE_NORMALMAP_TANGENTSPACE )
	vec3 mapN = texture2D( normalMap, vNormalMapUv ).xyz * 2.0 - 1.0;
	#if defined( USE_PACKED_NORMALMAP )
		mapN = vec3( mapN.xy, sqrt( saturate( 1.0 - dot( mapN.xy, mapN.xy ) ) ) );
	#endif
	mapN.xy *= normalScale;
	normal = normalize( tbn * mapN );
#elif defined( USE_BUMPMAP )
	normal = perturbNormalArb( - vViewPosition, normal, dHdxy_fwd(), faceDirection );
#endif`,o0=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,l0=`#ifndef FLAT_SHADED
	varying vec3 vNormal;
	#ifdef USE_TANGENT
		varying vec3 vTangent;
		varying vec3 vBitangent;
	#endif
#endif`,c0=`#ifndef FLAT_SHADED
	vNormal = normalize( transformedNormal );
	#ifdef USE_TANGENT
		vTangent = normalize( transformedTangent );
		vBitangent = normalize( cross( vNormal, vTangent ) * tangent.w );
		#ifdef FLIP_SIDED
			vBitangent = - vBitangent;
		#endif
	#endif
#endif`,h0=`#ifdef USE_NORMALMAP
	uniform sampler2D normalMap;
	uniform vec2 normalScale;
#endif
#ifdef USE_NORMALMAP_OBJECTSPACE
	uniform mat3 normalMatrix;
#endif
#if ! defined ( USE_TANGENT ) && ( defined ( USE_NORMALMAP_TANGENTSPACE ) || defined ( USE_CLEARCOAT_NORMALMAP ) || defined( USE_ANISOTROPY ) )
	mat3 getTangentFrame( vec3 eye_pos, vec3 surf_norm, vec2 uv ) {
		vec3 q0 = dFdx( eye_pos.xyz );
		vec3 q1 = dFdy( eye_pos.xyz );
		vec2 st0 = dFdx( uv.st );
		vec2 st1 = dFdy( uv.st );
		vec3 N = surf_norm;
		vec3 q1perp = cross( q1, N );
		vec3 q0perp = cross( N, q0 );
		vec3 T = q1perp * st0.x + q0perp * st1.x;
		vec3 B = q1perp * st0.y + q0perp * st1.y;
		float det = max( dot( T, T ), dot( B, B ) );
		float scale = ( det == 0.0 ) ? 0.0 : inversesqrt( det );
		return mat3( T * scale, B * scale, N );
	}
#endif`,u0=`#ifdef USE_CLEARCOAT
	vec3 clearcoatNormal = nonPerturbedNormal;
#endif`,d0=`#ifdef USE_CLEARCOAT_NORMALMAP
	vec3 clearcoatMapN = texture2D( clearcoatNormalMap, vClearcoatNormalMapUv ).xyz * 2.0 - 1.0;
	clearcoatMapN.xy *= clearcoatNormalScale;
	clearcoatNormal = normalize( tbn2 * clearcoatMapN );
#endif`,f0=`#ifdef USE_CLEARCOATMAP
	uniform sampler2D clearcoatMap;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform sampler2D clearcoatNormalMap;
	uniform vec2 clearcoatNormalScale;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform sampler2D clearcoatRoughnessMap;
#endif`,p0=`#ifdef USE_IRIDESCENCEMAP
	uniform sampler2D iridescenceMap;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform sampler2D iridescenceThicknessMap;
#endif`,m0=`#ifdef OPAQUE
diffuseColor.a = 1.0;
#endif
#ifdef USE_TRANSMISSION
diffuseColor.a *= material.transmissionAlpha;
#endif
gl_FragColor = vec4( outgoingLight, diffuseColor.a );`,g0=`vec3 packNormalToRGB( const in vec3 normal ) {
	return normalize( normal ) * 0.5 + 0.5;
}
vec3 unpackRGBToNormal( const in vec3 rgb ) {
	return 2.0 * rgb.xyz - 1.0;
}
const float PackUpscale = 256. / 255.;const float UnpackDownscale = 255. / 256.;const float ShiftRight8 = 1. / 256.;
const float Inv255 = 1. / 255.;
const vec4 PackFactors = vec4( 1.0, 256.0, 256.0 * 256.0, 256.0 * 256.0 * 256.0 );
const vec2 UnpackFactors2 = vec2( UnpackDownscale, 1.0 / PackFactors.g );
const vec3 UnpackFactors3 = vec3( UnpackDownscale / PackFactors.rg, 1.0 / PackFactors.b );
const vec4 UnpackFactors4 = vec4( UnpackDownscale / PackFactors.rgb, 1.0 / PackFactors.a );
vec4 packDepthToRGBA( const in float v ) {
	if( v <= 0.0 )
		return vec4( 0., 0., 0., 0. );
	if( v >= 1.0 )
		return vec4( 1., 1., 1., 1. );
	float vuf;
	float af = modf( v * PackFactors.a, vuf );
	float bf = modf( vuf * ShiftRight8, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec4( vuf * Inv255, gf * PackUpscale, bf * PackUpscale, af );
}
vec3 packDepthToRGB( const in float v ) {
	if( v <= 0.0 )
		return vec3( 0., 0., 0. );
	if( v >= 1.0 )
		return vec3( 1., 1., 1. );
	float vuf;
	float bf = modf( v * PackFactors.b, vuf );
	float gf = modf( vuf * ShiftRight8, vuf );
	return vec3( vuf * Inv255, gf * PackUpscale, bf );
}
vec2 packDepthToRG( const in float v ) {
	if( v <= 0.0 )
		return vec2( 0., 0. );
	if( v >= 1.0 )
		return vec2( 1., 1. );
	float vuf;
	float gf = modf( v * 256., vuf );
	return vec2( vuf * Inv255, gf );
}
float unpackRGBAToDepth( const in vec4 v ) {
	return dot( v, UnpackFactors4 );
}
float unpackRGBToDepth( const in vec3 v ) {
	return dot( v, UnpackFactors3 );
}
float unpackRGToDepth( const in vec2 v ) {
	return v.r * UnpackFactors2.r + v.g * UnpackFactors2.g;
}
vec4 pack2HalfToRGBA( const in vec2 v ) {
	vec4 r = vec4( v.x, fract( v.x * 255.0 ), v.y, fract( v.y * 255.0 ) );
	return vec4( r.x - r.y / 255.0, r.y, r.z - r.w / 255.0, r.w );
}
vec2 unpackRGBATo2Half( const in vec4 v ) {
	return vec2( v.x + ( v.y / 255.0 ), v.z + ( v.w / 255.0 ) );
}
float viewZToOrthographicDepth( const in float viewZ, const in float near, const in float far ) {
	return ( viewZ + near ) / ( near - far );
}
float orthographicDepthToViewZ( const in float depth, const in float near, const in float far ) {
	#ifdef USE_REVERSED_DEPTH_BUFFER
	
		return depth * ( far - near ) - far;
	#else
		return depth * ( near - far ) - near;
	#endif
}
float viewZToPerspectiveDepth( const in float viewZ, const in float near, const in float far ) {
	return ( ( near + viewZ ) * far ) / ( ( far - near ) * viewZ );
}
float perspectiveDepthToViewZ( const in float depth, const in float near, const in float far ) {
	
	#ifdef USE_REVERSED_DEPTH_BUFFER
		return ( near * far ) / ( ( near - far ) * depth - near );
	#else
		return ( near * far ) / ( ( far - near ) * depth - far );
	#endif
}`,_0=`#ifdef PREMULTIPLIED_ALPHA
	gl_FragColor.rgb *= gl_FragColor.a;
#endif`,x0=`vec4 mvPosition = vec4( transformed, 1.0 );
#ifdef USE_BATCHING
	mvPosition = batchingMatrix * mvPosition;
#endif
#ifdef USE_INSTANCING
	mvPosition = instanceMatrix * mvPosition;
#endif
mvPosition = modelViewMatrix * mvPosition;
gl_Position = projectionMatrix * mvPosition;`,y0=`#ifdef DITHERING
	gl_FragColor.rgb = dithering( gl_FragColor.rgb );
#endif`,v0=`#ifdef DITHERING
	vec3 dithering( vec3 color ) {
		float grid_position = rand( gl_FragCoord.xy );
		vec3 dither_shift_RGB = vec3( 0.25 / 255.0, -0.25 / 255.0, 0.25 / 255.0 );
		dither_shift_RGB = mix( 2.0 * dither_shift_RGB, -2.0 * dither_shift_RGB, grid_position );
		return color + dither_shift_RGB;
	}
#endif`,M0=`float roughnessFactor = roughness;
#ifdef USE_ROUGHNESSMAP
	vec4 texelRoughness = texture2D( roughnessMap, vRoughnessMapUv );
	roughnessFactor *= texelRoughness.g;
#endif`,b0=`#ifdef USE_ROUGHNESSMAP
	uniform sampler2D roughnessMap;
#endif`,S0=`#if NUM_SPOT_LIGHT_COORDS > 0
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#if NUM_SPOT_LIGHT_MAPS > 0
	uniform sampler2D spotLightMap[ NUM_SPOT_LIGHT_MAPS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
		#define SUN_LIGHT_CASCADES 2
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow sunShadowMap[ NUM_SUN_LIGHT_SHADOWS ];
		#else
			uniform sampler2D sunShadowMap[ NUM_SUN_LIGHT_SHADOWS ];
		#endif
		uniform mat4 sunShadowMatrix[ NUM_SUN_LIGHT_SHADOWS * SUN_LIGHT_CASCADES ];
		uniform vec4 sunShadowCascade[ NUM_SUN_LIGHT_SHADOWS * SUN_LIGHT_CASCADES ];
		varying vec4 vSunShadowWorldPosition;
		varying vec3 vSunShadowWorldNormal;
		struct SunLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SunLightShadow sunLightShadows[ NUM_SUN_LIGHT_SHADOWS ];
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#else
			uniform sampler2D directionalShadowMap[ NUM_DIR_LIGHT_SHADOWS ];
		#endif
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform sampler2DShadow spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#else
			uniform sampler2D spotShadowMap[ NUM_SPOT_LIGHT_SHADOWS ];
		#endif
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#if defined( SHADOWMAP_TYPE_PCF )
			uniform samplerCubeShadow pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#elif defined( SHADOWMAP_TYPE_BASIC )
			uniform samplerCube pointShadowMap[ NUM_POINT_LIGHT_SHADOWS ];
		#endif
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float interleavedGradientNoise( vec2 position ) {
			return fract( 52.9829189 * fract( dot( position, vec2( 0.06711056, 0.00583715 ) ) ) );
		}
		vec2 vogelDiskSample( int sampleIndex, int samplesCount, float phi ) {
			const float goldenAngle = 2.399963229728653;
			float r = sqrt( ( float( sampleIndex ) + 0.5 ) / float( samplesCount ) );
			float theta = float( sampleIndex ) * goldenAngle + phi;
			return vec2( cos( theta ), sin( theta ) ) * r;
		}
	#endif
	#if defined( SHADOWMAP_TYPE_PCF )
		float getShadow( sampler2DShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			shadowCoord.z += shadowBias;
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 texelSize = vec2( 1.0 ) / shadowMapSize;
				float radius = shadowRadius * texelSize.x;
				float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
				shadow = (
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 0, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 1, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 2, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 3, 5, phi ) * radius, shadowCoord.z ) ) +
					texture( shadowMap, vec3( shadowCoord.xy + vogelDiskSample( 4, 5, phi ) * radius, shadowCoord.z ) )
				) * 0.2;
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#elif defined( SHADOWMAP_TYPE_VSM )
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				vec2 distribution = texture2D( shadowMap, shadowCoord.xy ).rg;
				float mean = distribution.x;
				float variance = distribution.y * distribution.y;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					float hard_shadow = step( mean, shadowCoord.z );
				#else
					float hard_shadow = step( shadowCoord.z, mean );
				#endif
				
				if ( hard_shadow == 1.0 ) {
					shadow = 1.0;
				} else {
					variance = max( variance, 0.0000001 );
					float d = shadowCoord.z - mean;
					float p_max = variance / ( variance + d * d );
					p_max = clamp( ( p_max - 0.3 ) / 0.65, 0.0, 1.0 );
					shadow = max( hard_shadow, p_max );
				}
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#else
		float getShadow( sampler2D shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
			float shadow = 1.0;
			shadowCoord.xyz /= shadowCoord.w;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				shadowCoord.z -= shadowBias;
			#else
				shadowCoord.z += shadowBias;
			#endif
			bool inFrustum = shadowCoord.x >= 0.0 && shadowCoord.x <= 1.0 && shadowCoord.y >= 0.0 && shadowCoord.y <= 1.0;
			bool frustumTest = inFrustum && shadowCoord.z <= 1.0;
			if ( frustumTest ) {
				float depth = texture2D( shadowMap, shadowCoord.xy ).r;
				#ifdef USE_REVERSED_DEPTH_BUFFER
					shadow = step( depth, shadowCoord.z );
				#else
					shadow = step( shadowCoord.z, depth );
				#endif
			}
			return mix( 1.0, shadow, shadowIntensity );
		}
	#endif
	#if NUM_SUN_LIGHT_SHADOWS > 0
		float getSunShadow(
			#if defined( SHADOWMAP_TYPE_PCF )
				sampler2DShadow shadowMap,
			#else
				sampler2D shadowMap,
			#endif
			SunLightShadow sunLightShadow,
			int shadowIndex
		) {
			vec4 shadowWorldPosition = vec4( vSunShadowWorldPosition.xyz + vSunShadowWorldNormal * sunLightShadow.shadowNormalBias, 1.0 );
			float viewDepth = vSunShadowWorldPosition.w;
			int cascadeOffset = shadowIndex * SUN_LIGHT_CASCADES;
			float shadow = 1.0;
			for ( int i = SUN_LIGHT_CASCADES - 1; i >= 0; i -- ) {
				vec4 cascade = sunShadowCascade[ cascadeOffset + i ];
				if ( viewDepth >= cascade.x && viewDepth < cascade.y ) {
					float cascadeShadow = getShadow(
						shadowMap,
						sunLightShadow.shadowMapSize,
						sunLightShadow.shadowIntensity,
						sunLightShadow.shadowBias,
						sunLightShadow.shadowRadius,
						sunShadowMatrix[ cascadeOffset + i ] * shadowWorldPosition
					);
					shadow = mix( cascadeShadow, shadow, smoothstep( cascade.z, cascade.y, viewDepth ) );
				}
			}
			return shadow;
		}
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
	#if defined( SHADOWMAP_TYPE_PCF )
	float getPointShadow( samplerCubeShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 bd3D = normalize( lightToPosition );
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			#ifdef USE_REVERSED_DEPTH_BUFFER
				float dp = ( shadowCameraNear * ( shadowCameraFar - viewSpaceZ ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp -= shadowBias;
			#else
				float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
				dp += shadowBias;
			#endif
			float texelSize = shadowRadius / shadowMapSize.x;
			vec3 absDir = abs( bd3D );
			vec3 tangent = absDir.x > absDir.z ? vec3( 0.0, 1.0, 0.0 ) : vec3( 1.0, 0.0, 0.0 );
			tangent = normalize( cross( bd3D, tangent ) );
			vec3 bitangent = cross( bd3D, tangent );
			float phi = interleavedGradientNoise( gl_FragCoord.xy ) * PI2;
			vec2 sample0 = vogelDiskSample( 0, 5, phi );
			vec2 sample1 = vogelDiskSample( 1, 5, phi );
			vec2 sample2 = vogelDiskSample( 2, 5, phi );
			vec2 sample3 = vogelDiskSample( 3, 5, phi );
			vec2 sample4 = vogelDiskSample( 4, 5, phi );
			shadow = (
				texture( shadowMap, vec4( bd3D + ( tangent * sample0.x + bitangent * sample0.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample1.x + bitangent * sample1.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample2.x + bitangent * sample2.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample3.x + bitangent * sample3.y ) * texelSize, dp ) ) +
				texture( shadowMap, vec4( bd3D + ( tangent * sample4.x + bitangent * sample4.y ) * texelSize, dp ) )
			) * 0.2;
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#elif defined( SHADOWMAP_TYPE_BASIC )
	float getPointShadow( samplerCube shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord, float shadowCameraNear, float shadowCameraFar ) {
		float shadow = 1.0;
		vec3 lightToPosition = shadowCoord.xyz;
		vec3 absVec = abs( lightToPosition );
		float viewSpaceZ = max( max( absVec.x, absVec.y ), absVec.z );
		if ( viewSpaceZ - shadowCameraFar <= 0.0 && viewSpaceZ - shadowCameraNear >= 0.0 ) {
			float dp = ( shadowCameraFar * ( viewSpaceZ - shadowCameraNear ) ) / ( viewSpaceZ * ( shadowCameraFar - shadowCameraNear ) );
			dp += shadowBias;
			vec3 bd3D = normalize( lightToPosition );
			float depth = textureCube( shadowMap, bd3D ).r;
			#ifdef USE_REVERSED_DEPTH_BUFFER
				depth = 1.0 - depth;
			#endif
			shadow = step( dp, depth );
		}
		return mix( 1.0, shadow, shadowIntensity );
	}
	#endif
	#endif
#endif`,E0=`#if NUM_SPOT_LIGHT_COORDS > 0
	uniform mat4 spotLightMatrix[ NUM_SPOT_LIGHT_COORDS ];
	varying vec4 vSpotLightCoord[ NUM_SPOT_LIGHT_COORDS ];
#endif
#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
		varying vec4 vSunShadowWorldPosition;
		varying vec3 vSunShadowWorldNormal;
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		uniform mat4 directionalShadowMatrix[ NUM_DIR_LIGHT_SHADOWS ];
		varying vec4 vDirectionalShadowCoord[ NUM_DIR_LIGHT_SHADOWS ];
		struct DirectionalLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform DirectionalLightShadow directionalLightShadows[ NUM_DIR_LIGHT_SHADOWS ];
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
		struct SpotLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
		};
		uniform SpotLightShadow spotLightShadows[ NUM_SPOT_LIGHT_SHADOWS ];
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		uniform mat4 pointShadowMatrix[ NUM_POINT_LIGHT_SHADOWS ];
		varying vec4 vPointShadowCoord[ NUM_POINT_LIGHT_SHADOWS ];
		struct PointLightShadow {
			float shadowIntensity;
			float shadowBias;
			float shadowNormalBias;
			float shadowRadius;
			vec2 shadowMapSize;
			float shadowCameraNear;
			float shadowCameraFar;
		};
		uniform PointLightShadow pointLightShadows[ NUM_POINT_LIGHT_SHADOWS ];
	#endif
#endif`,T0=`#if ( defined( USE_SHADOWMAP ) && ( NUM_DIR_LIGHT_SHADOWS > 0 || NUM_SUN_LIGHT_SHADOWS > 0 || NUM_POINT_LIGHT_SHADOWS > 0 ) ) || ( NUM_SPOT_LIGHT_COORDS > 0 )
	#ifdef HAS_NORMAL
		vec3 shadowWorldNormal = transformNormalByInverseViewMatrix( transformedNormal, viewMatrix );
	#else
		vec3 shadowWorldNormal = vec3( 0.0 );
	#endif
	vec4 shadowWorldPosition;
#endif
#if defined( USE_SHADOWMAP )
	#if NUM_SUN_LIGHT_SHADOWS > 0
		vSunShadowWorldPosition = vec4( worldPosition.xyz, - mvPosition.z );
		vSunShadowWorldNormal = shadowWorldNormal;
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * directionalLightShadows[ i ].shadowNormalBias, 0 );
			vDirectionalShadowCoord[ i ] = directionalShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0
		#pragma unroll_loop_start
		for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
			shadowWorldPosition = worldPosition + vec4( shadowWorldNormal * pointLightShadows[ i ].shadowNormalBias, 0 );
			vPointShadowCoord[ i ] = pointShadowMatrix[ i ] * shadowWorldPosition;
		}
		#pragma unroll_loop_end
	#endif
#endif
#if NUM_SPOT_LIGHT_COORDS > 0
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_COORDS; i ++ ) {
		shadowWorldPosition = worldPosition;
		#if ( defined( USE_SHADOWMAP ) && UNROLLED_LOOP_INDEX < NUM_SPOT_LIGHT_SHADOWS )
			shadowWorldPosition.xyz += shadowWorldNormal * spotLightShadows[ i ].shadowNormalBias;
		#endif
		vSpotLightCoord[ i ] = spotLightMatrix[ i ] * shadowWorldPosition;
	}
	#pragma unroll_loop_end
#endif`,w0=`float getShadowMask() {
	float shadow = 1.0;
	#ifdef USE_SHADOWMAP
	#if NUM_SUN_LIGHT_SHADOWS > 0
	SunLightShadow sunLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SUN_LIGHT_SHADOWS; i ++ ) {
		sunLight = sunLightShadows[ i ];
		shadow *= receiveShadow ? getSunShadow( sunShadowMap[ i ], sunLight, UNROLLED_LOOP_INDEX ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_DIR_LIGHT_SHADOWS > 0
	DirectionalLightShadow directionalLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_DIR_LIGHT_SHADOWS; i ++ ) {
		directionalLight = directionalLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( directionalShadowMap[ i ], directionalLight.shadowMapSize, directionalLight.shadowIntensity, directionalLight.shadowBias, directionalLight.shadowRadius, vDirectionalShadowCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_SPOT_LIGHT_SHADOWS > 0
	SpotLightShadow spotLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_SPOT_LIGHT_SHADOWS; i ++ ) {
		spotLight = spotLightShadows[ i ];
		shadow *= receiveShadow ? getShadow( spotShadowMap[ i ], spotLight.shadowMapSize, spotLight.shadowIntensity, spotLight.shadowBias, spotLight.shadowRadius, vSpotLightCoord[ i ] ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#if NUM_POINT_LIGHT_SHADOWS > 0 && ( defined( SHADOWMAP_TYPE_PCF ) || defined( SHADOWMAP_TYPE_BASIC ) )
	PointLightShadow pointLight;
	#pragma unroll_loop_start
	for ( int i = 0; i < NUM_POINT_LIGHT_SHADOWS; i ++ ) {
		pointLight = pointLightShadows[ i ];
		shadow *= receiveShadow ? getPointShadow( pointShadowMap[ i ], pointLight.shadowMapSize, pointLight.shadowIntensity, pointLight.shadowBias, pointLight.shadowRadius, vPointShadowCoord[ i ], pointLight.shadowCameraNear, pointLight.shadowCameraFar ) : 1.0;
	}
	#pragma unroll_loop_end
	#endif
	#endif
	return shadow;
}`,A0=`#ifdef USE_SKINNING
	mat4 boneMatX = getBoneMatrix( skinIndex.x );
	mat4 boneMatY = getBoneMatrix( skinIndex.y );
	mat4 boneMatZ = getBoneMatrix( skinIndex.z );
	mat4 boneMatW = getBoneMatrix( skinIndex.w );
#endif`,R0=`#ifdef USE_SKINNING
	uniform mat4 bindMatrix;
	uniform mat4 bindMatrixInverse;
	uniform highp sampler2D boneTexture;
	mat4 getBoneMatrix( const in float i ) {
		int size = textureSize( boneTexture, 0 ).x;
		int j = int( i ) * 4;
		int x = j % size;
		int y = j / size;
		vec4 v1 = texelFetch( boneTexture, ivec2( x, y ), 0 );
		vec4 v2 = texelFetch( boneTexture, ivec2( x + 1, y ), 0 );
		vec4 v3 = texelFetch( boneTexture, ivec2( x + 2, y ), 0 );
		vec4 v4 = texelFetch( boneTexture, ivec2( x + 3, y ), 0 );
		return mat4( v1, v2, v3, v4 );
	}
#endif`,C0=`#ifdef USE_SKINNING
	vec4 skinVertex = bindMatrix * vec4( transformed, 1.0 );
	vec4 skinned = vec4( 0.0 );
	skinned += boneMatX * skinVertex * skinWeight.x;
	skinned += boneMatY * skinVertex * skinWeight.y;
	skinned += boneMatZ * skinVertex * skinWeight.z;
	skinned += boneMatW * skinVertex * skinWeight.w;
	transformed = ( bindMatrixInverse * skinned ).xyz;
#endif`,P0=`#ifdef USE_SKINNING
	mat4 skinMatrix = mat4( 0.0 );
	skinMatrix += skinWeight.x * boneMatX;
	skinMatrix += skinWeight.y * boneMatY;
	skinMatrix += skinWeight.z * boneMatZ;
	skinMatrix += skinWeight.w * boneMatW;
	skinMatrix = bindMatrixInverse * skinMatrix * bindMatrix;
	objectNormal = vec4( skinMatrix * vec4( objectNormal, 0.0 ) ).xyz;
	#ifdef USE_TANGENT
		objectTangent = vec4( skinMatrix * vec4( objectTangent, 0.0 ) ).xyz;
	#endif
#endif`,I0=`float specularStrength;
#ifdef USE_SPECULARMAP
	vec4 texelSpecular = texture2D( specularMap, vSpecularMapUv );
	specularStrength = texelSpecular.r;
#else
	specularStrength = 1.0;
#endif`,L0=`#ifdef USE_SPECULARMAP
	uniform sampler2D specularMap;
#endif`,D0=`#if defined( TONE_MAPPING )
	gl_FragColor.rgb = toneMapping( gl_FragColor.rgb );
#endif`,N0=`#ifndef saturate
#define saturate( a ) clamp( a, 0.0, 1.0 )
#endif
uniform float toneMappingExposure;
vec3 LinearToneMapping( vec3 color ) {
	return saturate( toneMappingExposure * color );
}
vec3 ReinhardToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	return saturate( color / ( vec3( 1.0 ) + color ) );
}
vec3 CineonToneMapping( vec3 color ) {
	color *= toneMappingExposure;
	color = max( vec3( 0.0 ), color - 0.004 );
	return pow( ( color * ( 6.2 * color + 0.5 ) ) / ( color * ( 6.2 * color + 1.7 ) + 0.06 ), vec3( 2.2 ) );
}
vec3 RRTAndODTFit( vec3 v ) {
	vec3 a = v * ( v + 0.0245786 ) - 0.000090537;
	vec3 b = v * ( 0.983729 * v + 0.4329510 ) + 0.238081;
	return a / b;
}
vec3 ACESFilmicToneMapping( vec3 color ) {
	const mat3 ACESInputMat = mat3(
		vec3( 0.59719, 0.07600, 0.02840 ),		vec3( 0.35458, 0.90834, 0.13383 ),
		vec3( 0.04823, 0.01566, 0.83777 )
	);
	const mat3 ACESOutputMat = mat3(
		vec3(  1.60475, -0.10208, -0.00327 ),		vec3( -0.53108,  1.10813, -0.07276 ),
		vec3( -0.07367, -0.00605,  1.07602 )
	);
	color *= toneMappingExposure / 0.6;
	color = ACESInputMat * color;
	color = RRTAndODTFit( color );
	color = ACESOutputMat * color;
	return saturate( color );
}
const mat3 LINEAR_REC2020_TO_LINEAR_SRGB = mat3(
	vec3( 1.6605, - 0.1246, - 0.0182 ),
	vec3( - 0.5876, 1.1329, - 0.1006 ),
	vec3( - 0.0728, - 0.0083, 1.1187 )
);
const mat3 LINEAR_SRGB_TO_LINEAR_REC2020 = mat3(
	vec3( 0.6274, 0.0691, 0.0164 ),
	vec3( 0.3293, 0.9195, 0.0880 ),
	vec3( 0.0433, 0.0113, 0.8956 )
);
vec3 agxDefaultContrastApprox( vec3 x ) {
	vec3 x2 = x * x;
	vec3 x4 = x2 * x2;
	return + 15.5 * x4 * x2
		- 40.14 * x4 * x
		+ 31.96 * x4
		- 6.868 * x2 * x
		+ 0.4298 * x2
		+ 0.1191 * x
		- 0.00232;
}
vec3 AgXToneMapping( vec3 color ) {
	const mat3 AgXInsetMatrix = mat3(
		vec3( 0.856627153315983, 0.137318972929847, 0.11189821299995 ),
		vec3( 0.0951212405381588, 0.761241990602591, 0.0767994186031903 ),
		vec3( 0.0482516061458583, 0.101439036467562, 0.811302368396859 )
	);
	const mat3 AgXOutsetMatrix = mat3(
		vec3( 1.1271005818144368, - 0.1413297634984383, - 0.14132976349843826 ),
		vec3( - 0.11060664309660323, 1.157823702216272, - 0.11060664309660294 ),
		vec3( - 0.016493938717834573, - 0.016493938717834257, 1.2519364065950405 )
	);
	const float AgxMinEv = - 12.47393;	const float AgxMaxEv = 4.026069;
	color *= toneMappingExposure;
	color = LINEAR_SRGB_TO_LINEAR_REC2020 * color;
	color = AgXInsetMatrix * color;
	color = max( color, 1e-10 );	color = log2( color );
	color = ( color - AgxMinEv ) / ( AgxMaxEv - AgxMinEv );
	color = clamp( color, 0.0, 1.0 );
	color = agxDefaultContrastApprox( color );
	color = AgXOutsetMatrix * color;
	color = pow( max( vec3( 0.0 ), color ), vec3( 2.2 ) );
	color = LINEAR_REC2020_TO_LINEAR_SRGB * color;
	color = clamp( color, 0.0, 1.0 );
	return color;
}
vec3 NeutralToneMapping( vec3 color ) {
	const float StartCompression = 0.8 - 0.04;
	const float Desaturation = 0.15;
	color *= toneMappingExposure;
	float x = min( color.r, min( color.g, color.b ) );
	float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
	color -= offset;
	float peak = max( color.r, max( color.g, color.b ) );
	if ( peak < StartCompression ) return color;
	float d = 1. - StartCompression;
	float newPeak = 1. - d * d / ( peak + d - StartCompression );
	color *= newPeak / peak;
	float g = 1. - 1. / ( Desaturation * ( peak - newPeak ) + 1. );
	return mix( color, vec3( newPeak ), g );
}
vec3 CustomToneMapping( vec3 color ) { return color; }`,U0=`#ifdef USE_TRANSMISSION
	material.transmission = transmission;
	material.transmissionAlpha = 1.0;
	material.thickness = thickness;
	material.attenuationDistance = attenuationDistance;
	material.attenuationColor = attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		material.transmission *= texture2D( transmissionMap, vTransmissionMapUv ).r;
	#endif
	#ifdef USE_THICKNESSMAP
		material.thickness *= texture2D( thicknessMap, vThicknessMapUv ).g;
	#endif
	vec3 pos = vWorldPosition;
	vec3 v = normalize( cameraPosition - pos );
	vec3 n = transformNormalByInverseViewMatrix( normal, viewMatrix );
	vec4 transmitted = getIBLVolumeRefraction(
		n, v, material.roughness, material.diffuseContribution, material.specularColorBlended, material.specularF90,
		pos, modelMatrix, viewMatrix, projectionMatrix, material.dispersion, material.ior, material.thickness,
		material.attenuationColor, material.attenuationDistance );
	material.transmissionAlpha = mix( material.transmissionAlpha, transmitted.a, material.transmission );
	totalDiffuse = mix( totalDiffuse, transmitted.rgb, material.transmission );
#endif`,F0=`#ifdef USE_TRANSMISSION
	uniform float transmission;
	uniform float thickness;
	uniform float attenuationDistance;
	uniform vec3 attenuationColor;
	#ifdef USE_TRANSMISSIONMAP
		uniform sampler2D transmissionMap;
	#endif
	#ifdef USE_THICKNESSMAP
		uniform sampler2D thicknessMap;
	#endif
	uniform vec2 transmissionSamplerSize;
	uniform sampler2D transmissionSamplerMap;
	uniform mat4 modelMatrix;
	uniform mat4 projectionMatrix;
	varying vec3 vWorldPosition;
	float w0( float a ) {
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - a + 3.0 ) - 3.0 ) + 1.0 );
	}
	float w1( float a ) {
		return ( 1.0 / 6.0 ) * ( a *  a * ( 3.0 * a - 6.0 ) + 4.0 );
	}
	float w2( float a ){
		return ( 1.0 / 6.0 ) * ( a * ( a * ( - 3.0 * a + 3.0 ) + 3.0 ) + 1.0 );
	}
	float w3( float a ) {
		return ( 1.0 / 6.0 ) * ( a * a * a );
	}
	float g0( float a ) {
		return w0( a ) + w1( a );
	}
	float g1( float a ) {
		return w2( a ) + w3( a );
	}
	float h0( float a ) {
		return - 1.0 + w1( a ) / ( w0( a ) + w1( a ) );
	}
	float h1( float a ) {
		return 1.0 + w3( a ) / ( w2( a ) + w3( a ) );
	}
	vec4 bicubic( sampler2D tex, vec2 uv, vec4 texelSize, float lod ) {
		uv = uv * texelSize.zw + 0.5;
		vec2 iuv = floor( uv );
		vec2 fuv = fract( uv );
		float g0x = g0( fuv.x );
		float g1x = g1( fuv.x );
		float h0x = h0( fuv.x );
		float h1x = h1( fuv.x );
		float h0y = h0( fuv.y );
		float h1y = h1( fuv.y );
		vec2 p0 = ( vec2( iuv.x + h0x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p1 = ( vec2( iuv.x + h1x, iuv.y + h0y ) - 0.5 ) * texelSize.xy;
		vec2 p2 = ( vec2( iuv.x + h0x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		vec2 p3 = ( vec2( iuv.x + h1x, iuv.y + h1y ) - 0.5 ) * texelSize.xy;
		return g0( fuv.y ) * ( g0x * textureLod( tex, p0, lod ) + g1x * textureLod( tex, p1, lod ) ) +
			g1( fuv.y ) * ( g0x * textureLod( tex, p2, lod ) + g1x * textureLod( tex, p3, lod ) );
	}
	vec4 textureBicubic( sampler2D sampler, vec2 uv, float lod ) {
		vec2 fLodSize = vec2( textureSize( sampler, int( lod ) ) );
		vec2 cLodSize = vec2( textureSize( sampler, int( lod + 1.0 ) ) );
		vec2 fLodSizeInv = 1.0 / fLodSize;
		vec2 cLodSizeInv = 1.0 / cLodSize;
		vec4 fSample = bicubic( sampler, uv, vec4( fLodSizeInv, fLodSize ), floor( lod ) );
		vec4 cSample = bicubic( sampler, uv, vec4( cLodSizeInv, cLodSize ), ceil( lod ) );
		return mix( fSample, cSample, fract( lod ) );
	}
	vec3 getVolumeTransmissionRay( const in vec3 n, const in vec3 v, const in float thickness, const in float ior, const in mat4 modelMatrix ) {
		vec3 refractionVector = refract( - v, normalize( n ), 1.0 / ior );
		vec3 modelScale;
		modelScale.x = length( vec3( modelMatrix[ 0 ].xyz ) );
		modelScale.y = length( vec3( modelMatrix[ 1 ].xyz ) );
		modelScale.z = length( vec3( modelMatrix[ 2 ].xyz ) );
		return normalize( refractionVector ) * thickness * modelScale;
	}
	float applyIorToRoughness( const in float roughness, const in float ior ) {
		return roughness * clamp( ior * 2.0 - 2.0, 0.0, 1.0 );
	}
	vec4 getTransmissionSample( const in vec2 fragCoord, const in float roughness, const in float ior ) {
		float lod = log2( transmissionSamplerSize.x ) * applyIorToRoughness( roughness, ior );
		return textureBicubic( transmissionSamplerMap, fragCoord.xy, lod );
	}
	vec3 volumeAttenuation( const in float transmissionDistance, const in vec3 attenuationColor, const in float attenuationDistance ) {
		if ( isinf( attenuationDistance ) ) {
			return vec3( 1.0 );
		} else {
			vec3 attenuationCoefficient = -log( attenuationColor ) / attenuationDistance;
			vec3 transmittance = exp( - attenuationCoefficient * transmissionDistance );			return transmittance;
		}
	}
	vec4 getIBLVolumeRefraction( const in vec3 n, const in vec3 v, const in float roughness, const in vec3 diffuseColor,
		const in vec3 specularColor, const in float specularF90, const in vec3 position, const in mat4 modelMatrix,
		const in mat4 viewMatrix, const in mat4 projMatrix, const in float dispersion, const in float ior, const in float thickness,
		const in vec3 attenuationColor, const in float attenuationDistance ) {
		vec4 transmittedLight;
		vec3 transmittance;
		#ifdef USE_DISPERSION
			float halfSpread = ( ior - 1.0 ) * 0.025 * dispersion;
			vec3 iors = vec3( ior - halfSpread, ior, ior + halfSpread );
			for ( int i = 0; i < 3; i ++ ) {
				vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, iors[ i ], modelMatrix );
				vec3 refractedRayExit = position + transmissionRay;
				vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
				vec2 refractionCoords = ndcPos.xy / ndcPos.w;
				refractionCoords += 1.0;
				refractionCoords /= 2.0;
				vec4 transmissionSample = getTransmissionSample( refractionCoords, roughness, iors[ i ] );
				transmittedLight[ i ] = transmissionSample[ i ];
				transmittedLight.a += transmissionSample.a;
				transmittance[ i ] = diffuseColor[ i ] * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance )[ i ];
			}
			transmittedLight.a /= 3.0;
		#else
			vec3 transmissionRay = getVolumeTransmissionRay( n, v, thickness, ior, modelMatrix );
			vec3 refractedRayExit = position + transmissionRay;
			vec4 ndcPos = projMatrix * viewMatrix * vec4( refractedRayExit, 1.0 );
			vec2 refractionCoords = ndcPos.xy / ndcPos.w;
			refractionCoords += 1.0;
			refractionCoords /= 2.0;
			transmittedLight = getTransmissionSample( refractionCoords, roughness, ior );
			transmittance = diffuseColor * volumeAttenuation( length( transmissionRay ), attenuationColor, attenuationDistance );
		#endif
		vec3 attenuatedColor = transmittance * transmittedLight.rgb;
		vec3 F = EnvironmentBRDF( n, v, specularColor, specularF90, roughness );
		float transmittanceFactor = ( transmittance.r + transmittance.g + transmittance.b ) / 3.0;
		return vec4( ( 1.0 - F ) * attenuatedColor, 1.0 - ( 1.0 - transmittedLight.a ) * transmittanceFactor );
	}
#endif`,O0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_SPECULARMAP
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,B0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	varying vec2 vUv;
#endif
#ifdef USE_MAP
	uniform mat3 mapTransform;
	varying vec2 vMapUv;
#endif
#ifdef USE_ALPHAMAP
	uniform mat3 alphaMapTransform;
	varying vec2 vAlphaMapUv;
#endif
#ifdef USE_LIGHTMAP
	uniform mat3 lightMapTransform;
	varying vec2 vLightMapUv;
#endif
#ifdef USE_AOMAP
	uniform mat3 aoMapTransform;
	varying vec2 vAoMapUv;
#endif
#ifdef USE_BUMPMAP
	uniform mat3 bumpMapTransform;
	varying vec2 vBumpMapUv;
#endif
#ifdef USE_NORMALMAP
	uniform mat3 normalMapTransform;
	varying vec2 vNormalMapUv;
#endif
#ifdef USE_DISPLACEMENTMAP
	uniform mat3 displacementMapTransform;
	varying vec2 vDisplacementMapUv;
#endif
#ifdef USE_EMISSIVEMAP
	uniform mat3 emissiveMapTransform;
	varying vec2 vEmissiveMapUv;
#endif
#ifdef USE_METALNESSMAP
	uniform mat3 metalnessMapTransform;
	varying vec2 vMetalnessMapUv;
#endif
#ifdef USE_ROUGHNESSMAP
	uniform mat3 roughnessMapTransform;
	varying vec2 vRoughnessMapUv;
#endif
#ifdef USE_ANISOTROPYMAP
	uniform mat3 anisotropyMapTransform;
	varying vec2 vAnisotropyMapUv;
#endif
#ifdef USE_CLEARCOATMAP
	uniform mat3 clearcoatMapTransform;
	varying vec2 vClearcoatMapUv;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	uniform mat3 clearcoatNormalMapTransform;
	varying vec2 vClearcoatNormalMapUv;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	uniform mat3 clearcoatRoughnessMapTransform;
	varying vec2 vClearcoatRoughnessMapUv;
#endif
#ifdef USE_SHEEN_COLORMAP
	uniform mat3 sheenColorMapTransform;
	varying vec2 vSheenColorMapUv;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	uniform mat3 sheenRoughnessMapTransform;
	varying vec2 vSheenRoughnessMapUv;
#endif
#ifdef USE_IRIDESCENCEMAP
	uniform mat3 iridescenceMapTransform;
	varying vec2 vIridescenceMapUv;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	uniform mat3 iridescenceThicknessMapTransform;
	varying vec2 vIridescenceThicknessMapUv;
#endif
#ifdef USE_SPECULARMAP
	uniform mat3 specularMapTransform;
	varying vec2 vSpecularMapUv;
#endif
#ifdef USE_SPECULAR_COLORMAP
	uniform mat3 specularColorMapTransform;
	varying vec2 vSpecularColorMapUv;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	uniform mat3 specularIntensityMapTransform;
	varying vec2 vSpecularIntensityMapUv;
#endif
#ifdef USE_TRANSMISSIONMAP
	uniform mat3 transmissionMapTransform;
	varying vec2 vTransmissionMapUv;
#endif
#ifdef USE_THICKNESSMAP
	uniform mat3 thicknessMapTransform;
	varying vec2 vThicknessMapUv;
#endif`,z0=`#if defined( USE_UV ) || defined( USE_ANISOTROPY )
	vUv = vec3( uv, 1 ).xy;
#endif
#ifdef USE_MAP
	vMapUv = ( mapTransform * vec3( MAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ALPHAMAP
	vAlphaMapUv = ( alphaMapTransform * vec3( ALPHAMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_LIGHTMAP
	vLightMapUv = ( lightMapTransform * vec3( LIGHTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_AOMAP
	vAoMapUv = ( aoMapTransform * vec3( AOMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_BUMPMAP
	vBumpMapUv = ( bumpMapTransform * vec3( BUMPMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_NORMALMAP
	vNormalMapUv = ( normalMapTransform * vec3( NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_DISPLACEMENTMAP
	vDisplacementMapUv = ( displacementMapTransform * vec3( DISPLACEMENTMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_EMISSIVEMAP
	vEmissiveMapUv = ( emissiveMapTransform * vec3( EMISSIVEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_METALNESSMAP
	vMetalnessMapUv = ( metalnessMapTransform * vec3( METALNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ROUGHNESSMAP
	vRoughnessMapUv = ( roughnessMapTransform * vec3( ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_ANISOTROPYMAP
	vAnisotropyMapUv = ( anisotropyMapTransform * vec3( ANISOTROPYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOATMAP
	vClearcoatMapUv = ( clearcoatMapTransform * vec3( CLEARCOATMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_NORMALMAP
	vClearcoatNormalMapUv = ( clearcoatNormalMapTransform * vec3( CLEARCOAT_NORMALMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_CLEARCOAT_ROUGHNESSMAP
	vClearcoatRoughnessMapUv = ( clearcoatRoughnessMapTransform * vec3( CLEARCOAT_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCEMAP
	vIridescenceMapUv = ( iridescenceMapTransform * vec3( IRIDESCENCEMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_IRIDESCENCE_THICKNESSMAP
	vIridescenceThicknessMapUv = ( iridescenceThicknessMapTransform * vec3( IRIDESCENCE_THICKNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_COLORMAP
	vSheenColorMapUv = ( sheenColorMapTransform * vec3( SHEEN_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SHEEN_ROUGHNESSMAP
	vSheenRoughnessMapUv = ( sheenRoughnessMapTransform * vec3( SHEEN_ROUGHNESSMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULARMAP
	vSpecularMapUv = ( specularMapTransform * vec3( SPECULARMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_COLORMAP
	vSpecularColorMapUv = ( specularColorMapTransform * vec3( SPECULAR_COLORMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_SPECULAR_INTENSITYMAP
	vSpecularIntensityMapUv = ( specularIntensityMapTransform * vec3( SPECULAR_INTENSITYMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_TRANSMISSIONMAP
	vTransmissionMapUv = ( transmissionMapTransform * vec3( TRANSMISSIONMAP_UV, 1 ) ).xy;
#endif
#ifdef USE_THICKNESSMAP
	vThicknessMapUv = ( thicknessMapTransform * vec3( THICKNESSMAP_UV, 1 ) ).xy;
#endif`,k0=`#if defined( USE_ENVMAP ) || defined( DISTANCE ) || defined ( USE_SHADOWMAP ) || defined ( USE_TRANSMISSION ) || NUM_SPOT_LIGHT_COORDS > 0
	vec4 worldPosition = vec4( transformed, 1.0 );
	#ifdef USE_BATCHING
		worldPosition = batchingMatrix * worldPosition;
	#endif
	#ifdef USE_INSTANCING
		worldPosition = instanceMatrix * worldPosition;
	#endif
	worldPosition = modelMatrix * worldPosition;
#endif`,V0=`varying vec2 vUv;
uniform mat3 uvTransform;
void main() {
	vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	gl_Position = vec4( position.xy, 1.0, 1.0 );
}`,H0=`uniform sampler2D t2D;
uniform float backgroundIntensity;
varying vec2 vUv;
void main() {
	vec4 texColor = texture2D( t2D, vUv );
	#ifdef DECODE_VIDEO_TEXTURE
		texColor = vec4( mix( pow( texColor.rgb * 0.9478672986 + vec3( 0.0521327014 ), vec3( 2.4 ) ), texColor.rgb * 0.0773993808, vec3( lessThanEqual( texColor.rgb, vec3( 0.04045 ) ) ) ), texColor.w );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,G0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,W0=`#ifdef ENVMAP_TYPE_CUBE
	uniform samplerCube envMap;
#elif defined( ENVMAP_TYPE_CUBE_UV )
	uniform sampler2D envMap;
#endif
uniform float backgroundBlurriness;
uniform float backgroundIntensity;
uniform mat3 backgroundRotation;
varying vec3 vWorldDirection;
#include <cube_uv_reflection_fragment>
void main() {
	#ifdef ENVMAP_TYPE_CUBE
		vec4 texColor = textureCube( envMap, backgroundRotation * vWorldDirection );
	#elif defined( ENVMAP_TYPE_CUBE_UV )
		vec4 texColor = textureCubeUV( envMap, backgroundRotation * vWorldDirection, backgroundBlurriness );
	#else
		vec4 texColor = vec4( 0.0, 0.0, 0.0, 1.0 );
	#endif
	texColor.rgb *= backgroundIntensity;
	gl_FragColor = texColor;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,X0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
	gl_Position.z = gl_Position.w;
}`,q0=`uniform samplerCube tCube;
uniform float tFlip;
uniform float opacity;
varying vec3 vWorldDirection;
void main() {
	vec4 texColor = textureCube( tCube, vec3( tFlip * vWorldDirection.x, vWorldDirection.yz ) );
	gl_FragColor = texColor;
	gl_FragColor.a *= opacity;
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,Y0=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
varying vec2 vHighPrecisionZW;
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vHighPrecisionZW = gl_Position.zw;
}`,Z0=`#if DEPTH_PACKING == 3200
	uniform float opacity;
#endif
#include <common>
#include <packing>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
varying vec2 vHighPrecisionZW;
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#if DEPTH_PACKING == 3200
		diffuseColor.a = opacity;
	#endif
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <logdepthbuf_fragment>
	#ifdef USE_REVERSED_DEPTH_BUFFER
		float fragCoordZ = vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ];
	#else
		float fragCoordZ = 0.5 * vHighPrecisionZW[ 0 ] / vHighPrecisionZW[ 1 ] + 0.5;
	#endif
	#if DEPTH_PACKING == 3200
		gl_FragColor = vec4( vec3( 1.0 - fragCoordZ ), opacity );
	#elif DEPTH_PACKING == 3201
		gl_FragColor = packDepthToRGBA( fragCoordZ );
	#elif DEPTH_PACKING == 3202
		gl_FragColor = vec4( packDepthToRGB( fragCoordZ ), 1.0 );
	#elif DEPTH_PACKING == 3203
		gl_FragColor = vec4( packDepthToRG( fragCoordZ ), 0.0, 1.0 );
	#endif
}`,$0=`#define DISTANCE
varying vec3 vWorldPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <skinbase_vertex>
	#include <morphinstance_vertex>
	#ifdef USE_DISPLACEMENTMAP
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <worldpos_vertex>
	#include <clipping_planes_vertex>
	vWorldPosition = worldPosition.xyz;
}`,J0=`#define DISTANCE
uniform vec3 referencePosition;
uniform float nearDistance;
uniform float farDistance;
varying vec3 vWorldPosition;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 1.0 );
	#include <clipping_planes_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	float dist = length( vWorldPosition - referencePosition );
	dist = ( dist - nearDistance ) / ( farDistance - nearDistance );
	dist = saturate( dist );
	gl_FragColor = vec4( dist, 0.0, 0.0, 1.0 );
}`,j0=`varying vec3 vWorldDirection;
#include <common>
void main() {
	vWorldDirection = transformDirection( position, modelMatrix );
	#include <begin_vertex>
	#include <project_vertex>
}`,K0=`uniform sampler2D tEquirect;
varying vec3 vWorldDirection;
#include <common>
void main() {
	vec3 direction = normalize( vWorldDirection );
	vec2 sampleUV = equirectUv( direction );
	gl_FragColor = texture2D( tEquirect, sampleUV );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
}`,Q0=`uniform float scale;
attribute float lineDistance;
varying float vLineDistance;
#include <common>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	vLineDistance = scale * lineDistance;
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,e_=`uniform vec3 diffuse;
uniform float opacity;
uniform float dashSize;
uniform float totalSize;
varying float vLineDistance;
#include <common>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	if ( mod( vLineDistance, totalSize ) > dashSize ) {
		discard;
	}
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,t_=`#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#if defined ( USE_ENVMAP ) || defined ( USE_SKINNING )
		#include <beginnormal_vertex>
		#include <morphnormal_vertex>
		#include <skinbase_vertex>
		#include <skinnormal_vertex>
		#include <defaultnormal_vertex>
	#endif
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <fog_vertex>
}`,n_=`uniform vec3 diffuse;
uniform float opacity;
#ifndef FLAT_SHADED
	varying vec3 vNormal;
#endif
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <fog_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	#ifdef USE_LIGHTMAP
		vec4 lightMapTexel = texture2D( lightMap, vLightMapUv );
		reflectedLight.indirectDiffuse += lightMapTexel.rgb * lightMapIntensity * RECIPROCAL_PI;
	#else
		reflectedLight.indirectDiffuse += vec3( 1.0 );
	#endif
	#include <aomap_fragment>
	reflectedLight.indirectDiffuse *= diffuseColor.rgb;
	vec3 outgoingLight = reflectedLight.indirectDiffuse;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,i_=`#define LAMBERT
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,s_=`#define LAMBERT
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_lambert_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_lambert_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,r_=`#define MATCAP
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <color_pars_vertex>
#include <displacementmap_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
	vViewPosition = - mvPosition.xyz;
}`,a_=`#define MATCAP
uniform vec3 diffuse;
uniform float opacity;
uniform sampler2D matcap;
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	vec3 viewDir = normalize( vViewPosition );
	vec3 x = normalize( vec3( viewDir.z, 0.0, - viewDir.x ) );
	vec3 y = cross( viewDir, x );
	vec2 uv = vec2( dot( x, normal ), dot( y, normal ) ) * 0.495 + 0.5;
	#ifdef USE_MATCAP
		vec4 matcapColor = texture2D( matcap, uv );
	#else
		vec4 matcapColor = vec4( vec3( mix( 0.2, 0.8, uv.y ) ), 1.0 );
	#endif
	vec3 outgoingLight = diffuseColor.rgb * matcapColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,o_=`#define NORMAL
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	vViewPosition = - mvPosition.xyz;
#endif
}`,l_=`#define NORMAL
uniform float opacity;
#if defined( FLAT_SHADED ) || defined( USE_BUMPMAP ) || defined( USE_NORMALMAP_TANGENTSPACE )
	varying vec3 vViewPosition;
#endif
#include <uv_pars_fragment>
#include <normal_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( 0.0, 0.0, 0.0, opacity );
	#include <clipping_planes_fragment>
	#include <logdepthbuf_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	gl_FragColor = vec4( normalize( normal ) * 0.5 + 0.5, diffuseColor.a );
	#ifdef OPAQUE
		gl_FragColor.a = 1.0;
	#endif
}`,c_=`#define PHONG
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <envmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <envmap_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,h_=`#define PHONG
uniform vec3 diffuse;
uniform vec3 emissive;
uniform vec3 specular;
uniform float shininess;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_phong_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <specularmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <specularmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_phong_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + reflectedLight.directSpecular + reflectedLight.indirectSpecular + totalEmissiveRadiance;
	#include <envmap_fragment>
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,u_=`#define STANDARD
varying vec3 vViewPosition;
#ifdef USE_TRANSMISSION
	varying vec3 vWorldPosition;
#endif
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
#ifdef USE_TRANSMISSION
	vWorldPosition = worldPosition.xyz;
#endif
}`,d_=`#define STANDARD
#ifdef PHYSICAL
	#define IOR
	#define USE_SPECULAR
#endif
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float roughness;
uniform float metalness;
uniform float opacity;
#ifdef IOR
	uniform float ior;
#endif
#ifdef USE_SPECULAR
	uniform float specularIntensity;
	uniform vec3 specularColor;
	#ifdef USE_SPECULAR_COLORMAP
		uniform sampler2D specularColorMap;
	#endif
	#ifdef USE_SPECULAR_INTENSITYMAP
		uniform sampler2D specularIntensityMap;
	#endif
#endif
#ifdef USE_CLEARCOAT
	uniform float clearcoat;
	uniform float clearcoatRoughness;
#endif
#ifdef USE_DISPERSION
	uniform float dispersion;
#endif
#ifdef USE_RETROREFLECTION
	uniform float retroreflectivity;
#endif
#ifdef USE_IRIDESCENCE
	uniform float iridescence;
	uniform float iridescenceIOR;
	uniform float iridescenceThicknessMinimum;
	uniform float iridescenceThicknessMaximum;
#endif
#ifdef USE_SHEEN
	uniform vec3 sheenColor;
	uniform float sheenRoughness;
	#ifdef USE_SHEEN_COLORMAP
		uniform sampler2D sheenColorMap;
	#endif
	#ifdef USE_SHEEN_ROUGHNESSMAP
		uniform sampler2D sheenRoughnessMap;
	#endif
#endif
#ifdef USE_ANISOTROPY
	uniform vec2 anisotropyVector;
	#ifdef USE_ANISOTROPYMAP
		uniform sampler2D anisotropyMap;
	#endif
#endif
varying vec3 vViewPosition;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <iridescence_fragment>
#include <cube_uv_reflection_fragment>
#include <envmap_common_pars_fragment>
#include <envmap_physical_pars_fragment>
#include <fog_pars_fragment>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_physical_pars_fragment>
#include <transmission_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <clearcoat_pars_fragment>
#include <iridescence_pars_fragment>
#include <roughnessmap_pars_fragment>
#include <metalnessmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <roughnessmap_fragment>
	#include <metalnessmap_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <clearcoat_normal_fragment_begin>
	#include <clearcoat_normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_physical_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 totalDiffuse = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse;
	vec3 totalSpecular = reflectedLight.directSpecular + reflectedLight.indirectSpecular;
	#include <transmission_fragment>
	vec3 outgoingLight = totalDiffuse + totalSpecular + totalEmissiveRadiance;
	#ifdef USE_SHEEN
 
		outgoingLight = outgoingLight + sheenSpecularDirect + sheenSpecularIndirect;
 
 	#endif
	#ifdef USE_CLEARCOAT
		float dotNVcc = saturate( dot( geometryClearcoatNormal, geometryViewDir ) );
		vec3 Fcc = F_Schlick( material.clearcoatF0, material.clearcoatF90, dotNVcc );
		outgoingLight = outgoingLight * ( 1.0 - material.clearcoat * Fcc ) + ( clearcoatSpecularDirect + clearcoatSpecularIndirect ) * material.clearcoat;
	#endif
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,f_=`#define TOON
varying vec3 vViewPosition;
#include <common>
#include <batching_pars_vertex>
#include <uv_pars_vertex>
#include <displacementmap_pars_vertex>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <normal_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <shadowmap_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <normal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <displacementmap_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	vViewPosition = - mvPosition.xyz;
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,p_=`#define TOON
uniform vec3 diffuse;
uniform vec3 emissive;
uniform float opacity;
#include <common>
#include <dithering_pars_fragment>
#include <color_pars_fragment>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <aomap_pars_fragment>
#include <lightmap_pars_fragment>
#include <emissivemap_pars_fragment>
#include <gradientmap_pars_fragment>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <normal_pars_fragment>
#include <lights_toon_pars_fragment>
#include <shadowmap_pars_fragment>
#include <bumpmap_pars_fragment>
#include <normalmap_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	ReflectedLight reflectedLight = ReflectedLight( vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ), vec3( 0.0 ) );
	vec3 totalEmissiveRadiance = emissive;
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <color_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	#include <normal_fragment_begin>
	#include <normal_fragment_maps>
	#include <emissivemap_fragment>
	#include <lights_toon_fragment>
	#include <lights_fragment_begin>
	#include <lights_fragment_maps>
	#include <lights_fragment_end>
	#include <aomap_fragment>
	vec3 outgoingLight = reflectedLight.directDiffuse + reflectedLight.indirectDiffuse + totalEmissiveRadiance;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
	#include <dithering_fragment>
}`,m_=`uniform float size;
uniform float scale;
#include <common>
#include <color_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
#ifdef USE_POINTS_UV
	varying vec2 vUv;
	uniform mat3 uvTransform;
#endif
void main() {
	#ifdef USE_POINTS_UV
		vUv = ( uvTransform * vec3( uv, 1 ) ).xy;
	#endif
	#include <color_vertex>
	#include <morphinstance_vertex>
	#include <morphcolor_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <project_vertex>
	gl_PointSize = size;
	#ifdef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) gl_PointSize *= ( scale / - mvPosition.z );
	#endif
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <worldpos_vertex>
	#include <fog_vertex>
}`,g_=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <color_pars_fragment>
#include <map_particle_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_particle_fragment>
	#include <color_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,__=`#include <common>
#include <batching_pars_vertex>
#include <fog_pars_vertex>
#include <morphtarget_pars_vertex>
#include <skinning_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <shadowmap_pars_vertex>
void main() {
	#include <batching_vertex>
	#include <beginnormal_vertex>
	#include <morphinstance_vertex>
	#include <morphnormal_vertex>
	#include <skinbase_vertex>
	#include <skinnormal_vertex>
	#include <defaultnormal_vertex>
	#include <begin_vertex>
	#include <morphtarget_vertex>
	#include <skinning_vertex>
	#include <project_vertex>
	#include <logdepthbuf_vertex>
	#include <worldpos_vertex>
	#include <shadowmap_vertex>
	#include <fog_vertex>
}`,x_=`uniform vec3 color;
uniform float opacity;
#include <common>
#include <fog_pars_fragment>
#include <bsdfs>
#include <lights_pars_begin>
#include <logdepthbuf_pars_fragment>
#include <shadowmap_pars_fragment>
#include <shadowmask_pars_fragment>
void main() {
	#include <logdepthbuf_fragment>
	gl_FragColor = vec4( color, opacity * ( 1.0 - getShadowMask() ) );
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
	#include <premultiplied_alpha_fragment>
}`,y_=`uniform float rotation;
uniform vec2 center;
#include <common>
#include <uv_pars_vertex>
#include <fog_pars_vertex>
#include <logdepthbuf_pars_vertex>
#include <clipping_planes_pars_vertex>
void main() {
	#include <uv_vertex>
	vec4 mvPosition = modelViewMatrix[ 3 ];
	vec2 scale = vec2( length( modelMatrix[ 0 ].xyz ), length( modelMatrix[ 1 ].xyz ) );
	#ifndef USE_SIZEATTENUATION
		bool isPerspective = isPerspectiveMatrix( projectionMatrix );
		if ( isPerspective ) scale *= - mvPosition.z;
	#endif
	vec2 alignedPosition = ( position.xy - ( center - vec2( 0.5 ) ) ) * scale;
	vec2 rotatedPosition;
	rotatedPosition.x = cos( rotation ) * alignedPosition.x - sin( rotation ) * alignedPosition.y;
	rotatedPosition.y = sin( rotation ) * alignedPosition.x + cos( rotation ) * alignedPosition.y;
	mvPosition.xy += rotatedPosition;
	gl_Position = projectionMatrix * mvPosition;
	#include <logdepthbuf_vertex>
	#include <clipping_planes_vertex>
	#include <fog_vertex>
}`,v_=`uniform vec3 diffuse;
uniform float opacity;
#include <common>
#include <uv_pars_fragment>
#include <map_pars_fragment>
#include <alphamap_pars_fragment>
#include <alphatest_pars_fragment>
#include <alphahash_pars_fragment>
#include <fog_pars_fragment>
#include <logdepthbuf_pars_fragment>
#include <clipping_planes_pars_fragment>
void main() {
	vec4 diffuseColor = vec4( diffuse, opacity );
	#include <clipping_planes_fragment>
	vec3 outgoingLight = vec3( 0.0 );
	#include <logdepthbuf_fragment>
	#include <map_fragment>
	#include <alphamap_fragment>
	#include <alphatest_fragment>
	#include <alphahash_fragment>
	outgoingLight = diffuseColor.rgb;
	#include <opaque_fragment>
	#include <tonemapping_fragment>
	#include <colorspace_fragment>
	#include <fog_fragment>
}`,Ke={alphahash_fragment:Vm,alphahash_pars_fragment:Hm,alphamap_fragment:Gm,alphamap_pars_fragment:Wm,alphatest_fragment:Xm,alphatest_pars_fragment:qm,aomap_fragment:Ym,aomap_pars_fragment:Zm,batching_pars_vertex:$m,batching_vertex:Jm,begin_vertex:jm,beginnormal_vertex:Km,bsdfs:Qm,iridescence_fragment:eg,bumpmap_pars_fragment:tg,clipping_planes_fragment:ng,clipping_planes_pars_fragment:ig,clipping_planes_pars_vertex:sg,clipping_planes_vertex:rg,color_fragment:ag,color_pars_fragment:og,color_pars_vertex:lg,color_vertex:cg,common:hg,cube_uv_reflection_fragment:ug,defaultnormal_vertex:dg,displacementmap_pars_vertex:fg,displacementmap_vertex:pg,emissivemap_fragment:mg,emissivemap_pars_fragment:gg,colorspace_fragment:_g,colorspace_pars_fragment:xg,envmap_fragment:yg,envmap_common_pars_fragment:vg,envmap_pars_fragment:Mg,envmap_pars_vertex:bg,envmap_physical_pars_fragment:Dg,envmap_vertex:Sg,fog_vertex:Eg,fog_pars_vertex:Tg,fog_fragment:wg,fog_pars_fragment:Ag,gradientmap_pars_fragment:Rg,lightmap_pars_fragment:Cg,lights_lambert_fragment:Pg,lights_lambert_pars_fragment:Ig,lights_pars_begin:Lg,lights_toon_fragment:Ng,lights_toon_pars_fragment:Ug,lights_phong_fragment:Fg,lights_phong_pars_fragment:Og,lights_physical_fragment:Bg,lights_physical_pars_fragment:zg,lights_fragment_begin:kg,lights_fragment_maps:Vg,lights_fragment_end:Hg,lightprobes_pars_fragment:Gg,logdepthbuf_fragment:Wg,logdepthbuf_pars_fragment:Xg,logdepthbuf_pars_vertex:qg,logdepthbuf_vertex:Yg,map_fragment:Zg,map_pars_fragment:$g,map_particle_fragment:Jg,map_particle_pars_fragment:jg,metalnessmap_fragment:Kg,metalnessmap_pars_fragment:Qg,morphinstance_vertex:e0,morphcolor_vertex:t0,morphnormal_vertex:n0,morphtarget_pars_vertex:i0,morphtarget_vertex:s0,normal_fragment_begin:r0,normal_fragment_maps:a0,normal_pars_fragment:o0,normal_pars_vertex:l0,normal_vertex:c0,normalmap_pars_fragment:h0,clearcoat_normal_fragment_begin:u0,clearcoat_normal_fragment_maps:d0,clearcoat_pars_fragment:f0,iridescence_pars_fragment:p0,opaque_fragment:m0,packing:g0,premultiplied_alpha_fragment:_0,project_vertex:x0,dithering_fragment:y0,dithering_pars_fragment:v0,roughnessmap_fragment:M0,roughnessmap_pars_fragment:b0,shadowmap_pars_fragment:S0,shadowmap_pars_vertex:E0,shadowmap_vertex:T0,shadowmask_pars_fragment:w0,skinbase_vertex:A0,skinning_pars_vertex:R0,skinning_vertex:C0,skinnormal_vertex:P0,specularmap_fragment:I0,specularmap_pars_fragment:L0,tonemapping_fragment:D0,tonemapping_pars_fragment:N0,transmission_fragment:U0,transmission_pars_fragment:F0,uv_pars_fragment:O0,uv_pars_vertex:B0,uv_vertex:z0,worldpos_vertex:k0,background_vert:V0,background_frag:H0,backgroundCube_vert:G0,backgroundCube_frag:W0,cube_vert:X0,cube_frag:q0,depth_vert:Y0,depth_frag:Z0,distance_vert:$0,distance_frag:J0,equirect_vert:j0,equirect_frag:K0,linedashed_vert:Q0,linedashed_frag:e_,meshbasic_vert:t_,meshbasic_frag:n_,meshlambert_vert:i_,meshlambert_frag:s_,meshmatcap_vert:r_,meshmatcap_frag:a_,meshnormal_vert:o_,meshnormal_frag:l_,meshphong_vert:c_,meshphong_frag:h_,meshphysical_vert:u_,meshphysical_frag:d_,meshtoon_vert:f_,meshtoon_frag:p_,points_vert:m_,points_frag:g_,shadow_vert:__,shadow_frag:x_,sprite_vert:y_,sprite_frag:v_},_e={common:{diffuse:{value:new Ye(16777215)},opacity:{value:1},map:{value:null},mapTransform:{value:new Xe},alphaMap:{value:null},alphaMapTransform:{value:new Xe},alphaTest:{value:0}},specularmap:{specularMap:{value:null},specularMapTransform:{value:new Xe}},envmap:{envMap:{value:null},envMapRotation:{value:new Xe},reflectivity:{value:1},ior:{value:1.5},refractionRatio:{value:.98},dfgLUT:{value:null}},aomap:{aoMap:{value:null},aoMapIntensity:{value:1},aoMapTransform:{value:new Xe}},lightmap:{lightMap:{value:null},lightMapIntensity:{value:1},lightMapTransform:{value:new Xe}},bumpmap:{bumpMap:{value:null},bumpMapTransform:{value:new Xe},bumpScale:{value:1}},normalmap:{normalMap:{value:null},normalMapTransform:{value:new Xe},normalScale:{value:new ae(1,1)}},displacementmap:{displacementMap:{value:null},displacementMapTransform:{value:new Xe},displacementScale:{value:1},displacementBias:{value:0}},emissivemap:{emissiveMap:{value:null},emissiveMapTransform:{value:new Xe}},metalnessmap:{metalnessMap:{value:null},metalnessMapTransform:{value:new Xe}},roughnessmap:{roughnessMap:{value:null},roughnessMapTransform:{value:new Xe}},gradientmap:{gradientMap:{value:null}},fog:{fogDensity:{value:25e-5},fogNear:{value:1},fogFar:{value:2e3},fogColor:{value:new Ye(16777215)}},lights:{ambientLightColor:{value:[]},lightProbe:{value:[]},sunLights:{value:[],properties:{direction:{},color:{}}},sunLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},sunShadowMatrix:{value:[]},sunShadowCascade:{value:[]},directionalLights:{value:[],properties:{direction:{},color:{}}},directionalLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},directionalShadowMatrix:{value:[]},spotLights:{value:[],properties:{color:{},position:{},direction:{},distance:{},coneCos:{},penumbraCos:{},decay:{}}},spotLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{}}},spotLightMap:{value:[]},spotLightMatrix:{value:[]},pointLights:{value:[],properties:{color:{},position:{},decay:{},distance:{}}},pointLightShadows:{value:[],properties:{shadowIntensity:1,shadowBias:{},shadowNormalBias:{},shadowRadius:{},shadowMapSize:{},shadowCameraNear:{},shadowCameraFar:{}}},pointShadowMatrix:{value:[]},hemisphereLights:{value:[],properties:{direction:{},skyColor:{},groundColor:{}}},rectAreaLights:{value:[],properties:{color:{},position:{},width:{},height:{}}},ltc_1:{value:null},ltc_2:{value:null},probesSH:{value:null},probesMin:{value:new R},probesMax:{value:new R},probesResolution:{value:new R}},points:{diffuse:{value:new Ye(16777215)},opacity:{value:1},size:{value:1},scale:{value:1},map:{value:null},alphaMap:{value:null},alphaMapTransform:{value:new Xe},alphaTest:{value:0},uvTransform:{value:new Xe}},sprite:{diffuse:{value:new Ye(16777215)},opacity:{value:1},center:{value:new ae(.5,.5)},rotation:{value:0},map:{value:null},mapTransform:{value:new Xe},alphaMap:{value:null},alphaMapTransform:{value:new Xe},alphaTest:{value:0}}},Yn={basic:{uniforms:Jt([_e.common,_e.specularmap,_e.envmap,_e.aomap,_e.lightmap,_e.fog]),vertexShader:Ke.meshbasic_vert,fragmentShader:Ke.meshbasic_frag},lambert:{uniforms:Jt([_e.common,_e.specularmap,_e.envmap,_e.aomap,_e.lightmap,_e.emissivemap,_e.bumpmap,_e.normalmap,_e.displacementmap,_e.fog,_e.lights,{emissive:{value:new Ye(0)},envMapIntensity:{value:1}}]),vertexShader:Ke.meshlambert_vert,fragmentShader:Ke.meshlambert_frag},phong:{uniforms:Jt([_e.common,_e.specularmap,_e.envmap,_e.aomap,_e.lightmap,_e.emissivemap,_e.bumpmap,_e.normalmap,_e.displacementmap,_e.fog,_e.lights,{emissive:{value:new Ye(0)},specular:{value:new Ye(1118481)},shininess:{value:30},envMapIntensity:{value:1}}]),vertexShader:Ke.meshphong_vert,fragmentShader:Ke.meshphong_frag},standard:{uniforms:Jt([_e.common,_e.envmap,_e.aomap,_e.lightmap,_e.emissivemap,_e.bumpmap,_e.normalmap,_e.displacementmap,_e.roughnessmap,_e.metalnessmap,_e.fog,_e.lights,{emissive:{value:new Ye(0)},roughness:{value:1},metalness:{value:0},envMapIntensity:{value:1}}]),vertexShader:Ke.meshphysical_vert,fragmentShader:Ke.meshphysical_frag},toon:{uniforms:Jt([_e.common,_e.aomap,_e.lightmap,_e.emissivemap,_e.bumpmap,_e.normalmap,_e.displacementmap,_e.gradientmap,_e.fog,_e.lights,{emissive:{value:new Ye(0)}}]),vertexShader:Ke.meshtoon_vert,fragmentShader:Ke.meshtoon_frag},matcap:{uniforms:Jt([_e.common,_e.bumpmap,_e.normalmap,_e.displacementmap,_e.fog,{matcap:{value:null}}]),vertexShader:Ke.meshmatcap_vert,fragmentShader:Ke.meshmatcap_frag},points:{uniforms:Jt([_e.points,_e.fog]),vertexShader:Ke.points_vert,fragmentShader:Ke.points_frag},dashed:{uniforms:Jt([_e.common,_e.fog,{scale:{value:1},dashSize:{value:1},totalSize:{value:2}}]),vertexShader:Ke.linedashed_vert,fragmentShader:Ke.linedashed_frag},depth:{uniforms:Jt([_e.common,_e.displacementmap]),vertexShader:Ke.depth_vert,fragmentShader:Ke.depth_frag},normal:{uniforms:Jt([_e.common,_e.bumpmap,_e.normalmap,_e.displacementmap,{opacity:{value:1}}]),vertexShader:Ke.meshnormal_vert,fragmentShader:Ke.meshnormal_frag},sprite:{uniforms:Jt([_e.sprite,_e.fog]),vertexShader:Ke.sprite_vert,fragmentShader:Ke.sprite_frag},background:{uniforms:{uvTransform:{value:new Xe},t2D:{value:null},backgroundIntensity:{value:1}},vertexShader:Ke.background_vert,fragmentShader:Ke.background_frag},backgroundCube:{uniforms:{envMap:{value:null},backgroundBlurriness:{value:0},backgroundIntensity:{value:1},backgroundRotation:{value:new Xe}},vertexShader:Ke.backgroundCube_vert,fragmentShader:Ke.backgroundCube_frag},cube:{uniforms:{tCube:{value:null},tFlip:{value:-1},opacity:{value:1}},vertexShader:Ke.cube_vert,fragmentShader:Ke.cube_frag},equirect:{uniforms:{tEquirect:{value:null}},vertexShader:Ke.equirect_vert,fragmentShader:Ke.equirect_frag},distance:{uniforms:Jt([_e.common,_e.displacementmap,{referencePosition:{value:new R},nearDistance:{value:1},farDistance:{value:1e3}}]),vertexShader:Ke.distance_vert,fragmentShader:Ke.distance_frag},shadow:{uniforms:Jt([_e.lights,_e.fog,{color:{value:new Ye(0)},opacity:{value:1}}]),vertexShader:Ke.shadow_vert,fragmentShader:Ke.shadow_frag}};Yn.physical={uniforms:Jt([Yn.standard.uniforms,{clearcoat:{value:0},clearcoatMap:{value:null},clearcoatMapTransform:{value:new Xe},clearcoatNormalMap:{value:null},clearcoatNormalMapTransform:{value:new Xe},clearcoatNormalScale:{value:new ae(1,1)},clearcoatRoughness:{value:0},clearcoatRoughnessMap:{value:null},clearcoatRoughnessMapTransform:{value:new Xe},dispersion:{value:0},retroreflectivity:{value:0},iridescence:{value:0},iridescenceMap:{value:null},iridescenceMapTransform:{value:new Xe},iridescenceIOR:{value:1.3},iridescenceThicknessMinimum:{value:100},iridescenceThicknessMaximum:{value:400},iridescenceThicknessMap:{value:null},iridescenceThicknessMapTransform:{value:new Xe},sheen:{value:0},sheenColor:{value:new Ye(0)},sheenColorMap:{value:null},sheenColorMapTransform:{value:new Xe},sheenRoughness:{value:1},sheenRoughnessMap:{value:null},sheenRoughnessMapTransform:{value:new Xe},transmission:{value:0},transmissionMap:{value:null},transmissionMapTransform:{value:new Xe},transmissionSamplerSize:{value:new ae},transmissionSamplerMap:{value:null},thickness:{value:0},thicknessMap:{value:null},thicknessMapTransform:{value:new Xe},attenuationDistance:{value:0},attenuationColor:{value:new Ye(0)},specularColor:{value:new Ye(1,1,1)},specularColorMap:{value:null},specularColorMapTransform:{value:new Xe},specularIntensity:{value:1},specularIntensityMap:{value:null},specularIntensityMapTransform:{value:new Xe},anisotropyVector:{value:new ae},anisotropyMap:{value:null},anisotropyMapTransform:{value:new Xe}}]),vertexShader:Ke.meshphysical_vert,fragmentShader:Ke.meshphysical_frag};var El={r:0,b:0,g:0},M_=new dt,af=new Xe;af.set(-1,0,0,0,1,0,0,0,1);function b_(i,e,t,n,s,r){let a=new Ye(0),o=s===!0?0:1,l,c,h=null,d=0,u=null;function f(w){let A=w.isScene===!0?w.background:null;if(A&&A.isTexture){let M=w.backgroundBlurriness>0;A=e.get(A,M)}return A}function g(w){let A=!1,M=f(w);M===null?m(a,o):M&&M.isColor&&(m(M,1),A=!0);let E=i.xr.getEnvironmentBlendMode();E==="additive"?t.buffers.color.setClear(0,0,0,1,r):E==="alpha-blend"&&t.buffers.color.setClear(0,0,0,0,r),(i.autoClear||A)&&(t.buffers.depth.setTest(!0),t.buffers.depth.setMask(!0),t.buffers.color.setMask(!0),i.clear(i.autoClearColor,i.autoClearDepth,i.autoClearStencil))}function y(w,A){let M=f(A);M&&(M.isCubeTexture||M.mapping===Yr)?(c===void 0&&(c=new mt(new Gn(1,1,1),new tn({name:"BackgroundCubeMaterial",uniforms:Zi(Yn.backgroundCube.uniforms),vertexShader:Yn.backgroundCube.vertexShader,fragmentShader:Yn.backgroundCube.fragmentShader,side:Xt,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),c.geometry.deleteAttribute("normal"),c.geometry.deleteAttribute("uv"),c.onBeforeRender=function(E,S,C){this.matrixWorld.copyPosition(C.matrixWorld)},Object.defineProperty(c.material,"envMap",{get:function(){return this.uniforms.envMap.value}}),n.update(c)),c.material.uniforms.envMap.value=M,c.material.uniforms.backgroundBlurriness.value=A.backgroundBlurriness,c.material.uniforms.backgroundIntensity.value=A.backgroundIntensity,c.material.uniforms.backgroundRotation.value.setFromMatrix4(M_.makeRotationFromEuler(A.backgroundRotation)).transpose(),M.isCubeTexture&&M.isRenderTargetTexture===!1&&c.material.uniforms.backgroundRotation.value.premultiply(af),c.material.toneMapped=it.getTransfer(M.colorSpace)!==ut,(h!==M||d!==M.version||u!==i.toneMapping)&&(c.material.needsUpdate=!0,h=M,d=M.version,u=i.toneMapping),c.layers.enableAll(),w.unshift(c,c.geometry,c.material,0,0,null)):M&&M.isTexture&&(l===void 0&&(l=new mt(new Fr(2,2),new tn({name:"BackgroundMaterial",uniforms:Zi(Yn.background.uniforms),vertexShader:Yn.background.vertexShader,fragmentShader:Yn.background.fragmentShader,side:Ai,depthTest:!1,depthWrite:!1,fog:!1,allowOverride:!1})),l.geometry.deleteAttribute("normal"),Object.defineProperty(l.material,"map",{get:function(){return this.uniforms.t2D.value}}),n.update(l)),l.material.uniforms.t2D.value=M,l.material.uniforms.backgroundIntensity.value=A.backgroundIntensity,l.material.toneMapped=it.getTransfer(M.colorSpace)!==ut,M.matrixAutoUpdate===!0&&M.updateMatrix(),l.material.uniforms.uvTransform.value.copy(M.matrix),(h!==M||d!==M.version||u!==i.toneMapping)&&(l.material.needsUpdate=!0,h=M,d=M.version,u=i.toneMapping),l.layers.enableAll(),w.unshift(l,l.geometry,l.material,0,0,null))}function m(w,A){w.getRGB(El,nh(i)),t.buffers.color.setClear(El.r,El.g,El.b,A,r)}function p(){c!==void 0&&(c.geometry.dispose(),c.material.dispose(),c=void 0),l!==void 0&&(l.geometry.dispose(),l.material.dispose(),l=void 0)}return{getClearColor:function(){return a},setClearColor:function(w,A=1){a.set(w),o=A,m(a,o)},getClearAlpha:function(){return o},setClearAlpha:function(w){o=w,m(a,o)},render:g,addToRenderList:y,dispose:p}}function S_(i,e){let t=i.getParameter(i.MAX_VERTEX_ATTRIBS),n={},s=u(null),r=s,a=!1;function o(U,O,H,D,B){let X=!1,W=d(U,D,H,O);r!==W&&(r=W,c(r.object)),X=f(U,D,H,B),X&&g(U,D,H,B),B!==null&&e.update(B,i.ELEMENT_ARRAY_BUFFER),(X||a)&&(a=!1,M(U,O,H,D),B!==null&&i.bindBuffer(i.ELEMENT_ARRAY_BUFFER,e.get(B).buffer))}function l(){return i.createVertexArray()}function c(U){return i.bindVertexArray(U)}function h(U){return i.deleteVertexArray(U)}function d(U,O,H,D){let B=D.wireframe===!0,X=n[O.id];X===void 0&&(X={},n[O.id]=X);let W=U.isInstancedMesh===!0?U.id:0,ie=X[W];ie===void 0&&(ie={},X[W]=ie);let q=ie[H.id];q===void 0&&(q={},ie[H.id]=q);let K=q[B];return K===void 0&&(K=u(l()),q[B]=K),K}function u(U){let O=[],H=[],D=[];for(let B=0;B<t;B++)O[B]=0,H[B]=0,D[B]=0;return{geometry:null,program:null,wireframe:!1,newAttributes:O,enabledAttributes:H,attributeDivisors:D,object:U,attributes:{},index:null}}function f(U,O,H,D){let B=r.attributes,X=O.attributes,W=0,ie=H.getAttributes();for(let q in ie)if(ie[q].location>=0){let te=B[q],Ie=X[q];if(Ie===void 0&&(q==="instanceMatrix"&&U.instanceMatrix&&(Ie=U.instanceMatrix),q==="instanceColor"&&U.instanceColor&&(Ie=U.instanceColor)),te===void 0||te.attribute!==Ie||Ie&&te.data!==Ie.data)return!0;W++}return r.attributesNum!==W||r.index!==D}function g(U,O,H,D){let B={},X=O.attributes,W=0,ie=H.getAttributes();for(let q in ie)if(ie[q].location>=0){let te=X[q];te===void 0&&(q==="instanceMatrix"&&U.instanceMatrix&&(te=U.instanceMatrix),q==="instanceColor"&&U.instanceColor&&(te=U.instanceColor));let Ie={};Ie.attribute=te,te&&te.data&&(Ie.data=te.data),B[q]=Ie,W++}r.attributes=B,r.attributesNum=W,r.index=D}function y(){let U=r.newAttributes;for(let O=0,H=U.length;O<H;O++)U[O]=0}function m(U){p(U,0)}function p(U,O){let H=r.newAttributes,D=r.enabledAttributes,B=r.attributeDivisors;H[U]=1,D[U]===0&&(i.enableVertexAttribArray(U),D[U]=1),B[U]!==O&&(i.vertexAttribDivisor(U,O),B[U]=O)}function w(){let U=r.newAttributes,O=r.enabledAttributes;for(let H=0,D=O.length;H<D;H++)O[H]!==U[H]&&(i.disableVertexAttribArray(H),O[H]=0)}function A(U,O,H,D,B,X,W){W===!0?i.vertexAttribIPointer(U,O,H,B,X):i.vertexAttribPointer(U,O,H,D,B,X)}function M(U,O,H,D){y();let B=D.attributes,X=H.getAttributes(),W=O.defaultAttributeValues;for(let ie in X){let q=X[ie];if(q.location>=0){let K=B[ie];if(K===void 0&&(ie==="instanceMatrix"&&U.instanceMatrix&&(K=U.instanceMatrix),ie==="instanceColor"&&U.instanceColor&&(K=U.instanceColor)),K!==void 0){let te=K.normalized,Ie=K.itemSize,we=e.get(K);if(we===void 0)continue;let ot=we.buffer,et=we.type,rt=we.bytesPerElement,$=et===i.INT||et===i.UNSIGNED_INT||K.gpuType===zo;if(K.isInterleavedBufferAttribute){let Q=K.data,xe=Q.stride,He=K.offset;if(Q.isInstancedInterleavedBuffer){for(let Se=0;Se<q.locationSize;Se++)p(q.location+Se,Q.meshPerAttribute);U.isInstancedMesh!==!0&&D._maxInstanceCount===void 0&&(D._maxInstanceCount=Q.meshPerAttribute*Q.count)}else for(let Se=0;Se<q.locationSize;Se++)m(q.location+Se);i.bindBuffer(i.ARRAY_BUFFER,ot);for(let Se=0;Se<q.locationSize;Se++)A(q.location+Se,Ie/q.locationSize,et,te,xe*rt,(He+Ie/q.locationSize*Se)*rt,$)}else{if(K.isInstancedBufferAttribute){for(let Q=0;Q<q.locationSize;Q++)p(q.location+Q,K.meshPerAttribute);U.isInstancedMesh!==!0&&D._maxInstanceCount===void 0&&(D._maxInstanceCount=K.meshPerAttribute*K.count)}else for(let Q=0;Q<q.locationSize;Q++)m(q.location+Q);i.bindBuffer(i.ARRAY_BUFFER,ot);for(let Q=0;Q<q.locationSize;Q++)A(q.location+Q,Ie/q.locationSize,et,te,Ie*rt,Ie/q.locationSize*Q*rt,$)}}else if(W!==void 0){let te=W[ie];if(te!==void 0)switch(te.length){case 2:i.vertexAttrib2fv(q.location,te);break;case 3:i.vertexAttrib3fv(q.location,te);break;case 4:i.vertexAttrib4fv(q.location,te);break;default:i.vertexAttrib1fv(q.location,te)}}}}w()}function E(){T();for(let U in n){let O=n[U];for(let H in O){let D=O[H];for(let B in D){let X=D[B];for(let W in X)h(X[W].object),delete X[W];delete D[B]}}delete n[U]}}function S(U){if(n[U.id]===void 0)return;let O=n[U.id];for(let H in O){let D=O[H];for(let B in D){let X=D[B];for(let W in X)h(X[W].object),delete X[W];delete D[B]}}delete n[U.id]}function C(U){for(let O in n){let H=n[O];for(let D in H){let B=H[D];if(B[U.id]===void 0)continue;let X=B[U.id];for(let W in X)h(X[W].object),delete X[W];delete B[U.id]}}}function x(U){for(let O in n){let H=n[O],D=U.isInstancedMesh===!0?U.id:0,B=H[D];if(B!==void 0){for(let X in B){let W=B[X];for(let ie in W)h(W[ie].object),delete W[ie];delete B[X]}delete H[D],Object.keys(H).length===0&&delete n[O]}}}function T(){I(),a=!0,r!==s&&(r=s,c(r.object))}function I(){s.geometry=null,s.program=null,s.wireframe=!1}return{setup:o,reset:T,resetDefaultState:I,dispose:E,releaseStatesOfGeometry:S,releaseStatesOfObject:x,releaseStatesOfProgram:C,initAttributes:y,enableAttribute:m,disableUnusedAttributes:w}}function E_(i,e,t){let n;function s(l){n=l}function r(l,c){i.drawArrays(n,l,c),t.update(c,n,1)}function a(l,c,h){h!==0&&(i.drawArraysInstanced(n,l,c,h),t.update(c,n,h))}function o(l,c,h){if(h===0)return;e.get("WEBGL_multi_draw").multiDrawArraysWEBGL(n,l,0,c,0,h);let u=0;for(let f=0;f<h;f++)u+=c[f];t.update(u,n,1)}this.setMode=s,this.render=r,this.renderInstances=a,this.renderMultiDraw=o}function T_(i,e,t,n){let s;function r(){if(s!==void 0)return s;if(e.has("EXT_texture_filter_anisotropic")===!0){let C=e.get("EXT_texture_filter_anisotropic");s=i.getParameter(C.MAX_TEXTURE_MAX_ANISOTROPY_EXT)}else s=0;return s}function a(C){return!(C!==Mn&&n.convert(C)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_FORMAT))}function o(C){let x=C===Un&&(e.has("EXT_color_buffer_half_float")||e.has("EXT_color_buffer_float"));return!(C!==rn&&C!==vn&&!x&&n.convert(C)!==i.getParameter(i.IMPLEMENTATION_COLOR_READ_TYPE))}function l(C){if(C==="highp"){if(i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.HIGH_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.HIGH_FLOAT).precision>0)return"highp";C="mediump"}return C==="mediump"&&i.getShaderPrecisionFormat(i.VERTEX_SHADER,i.MEDIUM_FLOAT).precision>0&&i.getShaderPrecisionFormat(i.FRAGMENT_SHADER,i.MEDIUM_FLOAT).precision>0?"mediump":"lowp"}let c=t.precision!==void 0?t.precision:"highp",h=l(c);h!==c&&(Oe("WebGLRenderer:",c,"not supported, using",h,"instead."),c=h);let d=t.logarithmicDepthBuffer===!0,u=t.reversedDepthBuffer===!0&&e.has("EXT_clip_control");t.reversedDepthBuffer===!0&&u===!1&&Oe("WebGLRenderer: Unable to use reversed depth buffer due to missing EXT_clip_control extension. Fallback to default depth buffer.");let f=i.getParameter(i.MAX_TEXTURE_IMAGE_UNITS),g=i.getParameter(i.MAX_VERTEX_TEXTURE_IMAGE_UNITS),y=i.getParameter(i.MAX_TEXTURE_SIZE),m=i.getParameter(i.MAX_CUBE_MAP_TEXTURE_SIZE),p=i.getParameter(i.MAX_VERTEX_ATTRIBS),w=i.getParameter(i.MAX_VERTEX_UNIFORM_VECTORS),A=i.getParameter(i.MAX_VARYING_VECTORS),M=i.getParameter(i.MAX_FRAGMENT_UNIFORM_VECTORS),E=i.getParameter(i.MAX_SAMPLES),S=i.getParameter(i.SAMPLES);return{isWebGL2:!0,getMaxAnisotropy:r,getMaxPrecision:l,textureFormatReadable:a,textureTypeReadable:o,precision:c,logarithmicDepthBuffer:d,reversedDepthBuffer:u,maxTextures:f,maxVertexTextures:g,maxTextureSize:y,maxCubemapSize:m,maxAttributes:p,maxVertexUniforms:w,maxVaryings:A,maxFragmentUniforms:M,maxSamples:E,samples:S}}function w_(i){let e=this,t=null,n=0,s=!1,r=!1,a=new dn,o=new Xe,l={value:null,needsUpdate:!1};this.uniform=l,this.numPlanes=0,this.numIntersection=0,this.init=function(d,u){let f=d.length!==0||u||n!==0||s;return s=u,n=d.length,f},this.beginShadows=function(){r=!0,h(null)},this.endShadows=function(){r=!1},this.setGlobalState=function(d,u){t=h(d,u,0)},this.setState=function(d,u,f){let g=d.clippingPlanes,y=d.clipIntersection,m=d.clipShadows,p=i.get(d);if(!s||g===null||g.length===0||r&&!m)r?h(null):c();else{let w=r?0:n,A=w*4,M=p.clippingState||null;l.value=M,M=h(g,u,A,f);for(let E=0;E!==A;++E)M[E]=t[E];p.clippingState=M,this.numIntersection=y?this.numPlanes:0,this.numPlanes+=w}};function c(){l.value!==t&&(l.value=t,l.needsUpdate=n>0),e.numPlanes=n,e.numIntersection=0}function h(d,u,f,g){let y=d!==null?d.length:0,m=null;if(y!==0){if(m=l.value,g!==!0||m===null){let p=f+y*4,w=u.matrixWorldInverse;o.getNormalMatrix(w),(m===null||m.length<p)&&(m=new Float32Array(p));for(let A=0,M=f;A!==y;++A,M+=4)a.copy(d[A]).applyMatrix4(w,o),a.normal.toArray(m,M),m[M+3]=a.constant}l.value=m,l.needsUpdate=!0}return e.numPlanes=y,e.numIntersection=0,m}}var Hs=4,A_=6,R_=20,C_=256,ia=new Ns,Od=new Ye,uh=null,dh=0,fh=0,ph=!1,P_=new R,$i=new R,Ws=class{constructor(e){this._renderer=e,this._pingPongRenderTarget=null,this._lodMax=0,this._cubeSize=0,this._sizeLods=[],this._lodMeshes=[],this._backgroundBox=null,this._cubemapMaterial=null,this._equirectMaterial=null,this._blurMaterial=null,this._ggxMaterial=null}fromScene(e,t=0,n=.1,s=100,r={}){let{size:a=256,position:o=P_}=r;uh=this._renderer.getRenderTarget(),dh=this._renderer.getActiveCubeFace(),fh=this._renderer.getActiveMipmapLevel(),ph=this._renderer.xr.enabled,this._renderer.xr.enabled=!1,this._setSize(a);let l=this._allocateTargets();return l.depthBuffer=!0,this._sceneToCubeUV(e,n,s,l,o),t>0&&this._blur(l,0,0,t),this._applyPMREM(l),this._cleanup(l),l}fromEquirectangular(e,t=null){return this._fromTexture(e,t)}fromCubemap(e,t=null){return this._fromTexture(e,t)}compileCubemapShader(){this._cubemapMaterial===null&&(this._cubemapMaterial=kd(),this._compileMaterial(this._cubemapMaterial))}compileEquirectangularShader(){this._equirectMaterial===null&&(this._equirectMaterial=zd(),this._compileMaterial(this._equirectMaterial))}dispose(){this._dispose(),this._cubemapMaterial!==null&&this._cubemapMaterial.dispose(),this._equirectMaterial!==null&&this._equirectMaterial.dispose(),this._backgroundBox!==null&&(this._backgroundBox.geometry.dispose(),this._backgroundBox.material.dispose())}_setSize(e){this._lodMax=Math.floor(Math.log2(e)),this._cubeSize=Math.pow(2,this._lodMax)}_dispose(){this._blurMaterial!==null&&this._blurMaterial.dispose(),this._ggxMaterial!==null&&this._ggxMaterial.dispose(),this._pingPongRenderTarget!==null&&this._pingPongRenderTarget.dispose();for(let e=0;e<this._lodMeshes.length;e++)this._lodMeshes[e].geometry.dispose()}_cleanup(e){this._renderer.setRenderTarget(uh,dh,fh),this._renderer.xr.enabled=ph,e.scissorTest=!1,Vs(e,0,0,e.width,e.height)}_fromTexture(e,t){e.mapping===Ri||e.mapping===qi?this._setSize(e.image.length===0?16:e.image[0].width||e.image[0].image.width):this._setSize(e.image.width/4),uh=this._renderer.getRenderTarget(),dh=this._renderer.getActiveCubeFace(),fh=this._renderer.getActiveMipmapLevel(),ph=this._renderer.xr.enabled,this._renderer.xr.enabled=!1;let n=t||this._allocateTargets();return this._textureToCubeUV(e,n),this._applyPMREM(n),this._cleanup(n),n}_allocateTargets(){let e=3*Math.max(this._cubeSize,112),t=4*this._cubeSize,n={magFilter:Gt,minFilter:Gt,generateMipmaps:!1,type:Un,format:Mn,colorSpace:dr,depthBuffer:!1},s=Bd(e,t,n);if(this._pingPongRenderTarget===null||this._pingPongRenderTarget.width!==e||this._pingPongRenderTarget.height!==t){this._pingPongRenderTarget!==null&&this._dispose(),this._pingPongRenderTarget=Bd(e,t,n);let{_lodMax:r}=this;({lodMeshes:this._lodMeshes,sizeLods:this._sizeLods}=I_(r)),this._blurMaterial=D_(r,e,t),this._ggxMaterial=L_(r,e,t)}return s}_compileMaterial(e){let t=new mt(new Ut,e);this._renderer.compile(t,ia)}_sceneToCubeUV(e,t,n,s,r){let l=new Ht(90,1,t,n),c=[1,-1,1,1,1,1],h=[1,1,1,-1,-1,-1],d=this._renderer,u=d.autoClear,f=d.toneMapping;d.getClearColor(Od),d.toneMapping=Dn,d.autoClear=!1,d.state.buffers.depth.getReversed()&&(d.setRenderTarget(s),d.clearDepth(),d.setRenderTarget(null)),this._backgroundBox===null&&(this._backgroundBox=new mt(new Gn,new yi({name:"PMREM.Background",side:Xt,depthWrite:!1,depthTest:!1})));let y=this._backgroundBox,m=y.material,p=!1,w=e.background;w?w.isColor&&(m.color.copy(w),e.background=null,p=!0):(m.color.copy(Od),p=!0);for(let A=0;A<6;A++){let M=A%3;M===0?(l.up.set(0,c[A],0),l.position.set(r.x,r.y,r.z),l.lookAt(r.x+h[A],r.y,r.z)):M===1?(l.up.set(0,0,c[A]),l.position.set(r.x,r.y,r.z),l.lookAt(r.x,r.y+h[A],r.z)):(l.up.set(0,c[A],0),l.position.set(r.x,r.y,r.z),l.lookAt(r.x,r.y,r.z+h[A]));let E=this._cubeSize;Vs(s,M*E,A>2?E:0,E,E),d.setRenderTarget(s),p&&d.render(y,l),d.render(e,l)}d.toneMapping=f,d.autoClear=u,e.background=w}_textureToCubeUV(e,t){let n=this._renderer,s=e.mapping===Ri||e.mapping===qi;s?(this._cubemapMaterial===null&&(this._cubemapMaterial=kd()),this._cubemapMaterial.uniforms.flipEnvMap.value=e.isRenderTargetTexture===!1?-1:1):this._equirectMaterial===null&&(this._equirectMaterial=zd());let r=s?this._cubemapMaterial:this._equirectMaterial,a=this._lodMeshes[0];a.material=r;let o=r.uniforms;o.envMap.value=e;let l=this._cubeSize;Vs(t,0,0,3*l,2*l),n.setRenderTarget(t),n.render(a,ia)}_applyPMREM(e){let t=this._renderer,n=t.autoClear;t.autoClear=!1;let s=this._lodMeshes.length;for(let r=1;r<s;r++)this._applyGGXFilter(e,r-1,r);t.autoClear=n}_applyGGXFilter(e,t,n){let s=this._renderer,r=this._pingPongRenderTarget,a=this._ggxMaterial,o=this._lodMeshes[n];o.material=a;let l=a.uniforms,c=n/(this._lodMeshes.length-1),h=t/(this._lodMeshes.length-1),d=Math.sqrt(c*c-h*h),u=c*1.25,f=d*u,{_lodMax:g}=this,y=this._sizeLods[n],m=3*y*(n>g-Hs?n-g+Hs:0),p=4*(this._cubeSize-y);l.envMap.value=e.texture,l.roughness.value=f,l.mipInt.value=g-t,Vs(r,m,p,3*y,2*y),s.setRenderTarget(r),s.render(o,ia),l.envMap.value=r.texture,l.roughness.value=0,l.mipInt.value=g-n,Vs(e,m,p,3*y,2*y),s.setRenderTarget(e),s.render(o,ia)}_blur(e,t,n,s){let r=this._pingPongRenderTarget,a=Math.min(s,Math.PI)/Math.SQRT2;this._blurPass(e,r,t,n,a),this._blurPass(r,e,n,n,a)}_blurPass(e,t,n,s,r){let a=this._renderer,o=this._blurMaterial,l=this._lodMeshes[s];l.material=o;let c=o.uniforms;c.envMap.value=e.texture,c.sigma.value=r,c.mipInt.value=this._lodMax-n;let h=this._sizeLods[s],d=3*h*(s>this._lodMax-Hs?s-this._lodMax+Hs:0),u=4*(this._cubeSize-h);Vs(t,d,u,3*h,2*h),a.setRenderTarget(t),a.render(l,ia)}};function I_(i){let e=[],t=[],n=i,s=i-Hs+1+A_;for(let r=0;r<s;r++){let a=Math.pow(2,n);e.push(a);let o=1/(a-2),l=-o,c=1+o,h=[l,l,c,l,c,c,l,l,c,c,l,c],d=6,u=6,f=3,g=new Float32Array(f*u*d),y=new Float32Array(f*u*d);for(let p=0;p<d;p++){let w=p%3*2/3-1,A=p>2?0:-1,M=[w,A,0,w+2/3,A,0,w+2/3,A+1,0,w,A,0,w+2/3,A+1,0,w,A+1,0];g.set(M,f*u*p);for(let E=0;E<u;E++){let S=h[E*2]*2-1,C=h[E*2+1]*2-1;p===0?$i.set(1,C,S):p===1?$i.set(-S,1,-C):p===2?$i.set(-S,C,1):p===3?$i.set(-1,C,-S):p===4?$i.set(-S,-1,C):$i.set(S,C,-1),$i.toArray(y,(p*u+E)*f)}}let m=new Ut;m.setAttribute("position",new It(g,f)),m.setAttribute("outputDirection",new It(y,f)),t.push(new mt(m,null)),n>Hs&&n--}return{lodMeshes:t,sizeLods:e}}function Bd(i,e,t){let n=new sn(i,e,t);return n.texture.mapping=Yr,n.texture.name="PMREM.cubeUv",n.scissorTest=!0,n}function Vs(i,e,t,n,s){i.viewport.set(e,t,n,s),i.scissor.set(e,t,n,s)}function L_(i,e,t){return new tn({name:"PMREMGGXConvolution",defines:{GGX_SAMPLES:C_,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},roughness:{value:0},mipInt:{value:0}},vertexShader:Rl(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float roughness;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359

			// Van der Corput radical inverse
			float radicalInverse_VdC(uint bits) {
				bits = (bits << 16u) | (bits >> 16u);
				bits = ((bits & 0x55555555u) << 1u) | ((bits & 0xAAAAAAAAu) >> 1u);
				bits = ((bits & 0x33333333u) << 2u) | ((bits & 0xCCCCCCCCu) >> 2u);
				bits = ((bits & 0x0F0F0F0Fu) << 4u) | ((bits & 0xF0F0F0F0u) >> 4u);
				bits = ((bits & 0x00FF00FFu) << 8u) | ((bits & 0xFF00FF00u) >> 8u);
				return float(bits) * 2.3283064365386963e-10; // / 0x100000000
			}

			// Hammersley sequence
			vec2 hammersley(uint i, uint N) {
				return vec2(float(i) / float(N), radicalInverse_VdC(i));
			}

			// GGX VNDF importance sampling (Eric Heitz 2018)
			// "Sampling the GGX Distribution of Visible Normals"
			// https://jcgt.org/published/0007/04/01/
			vec3 importanceSampleGGX_VNDF(vec2 Xi, vec3 V, float roughness) {
				float alpha = roughness * roughness;

				// Section 4.1: Orthonormal basis
				vec3 T1 = vec3(1.0, 0.0, 0.0);
				vec3 T2 = cross(V, T1);

				// Section 4.2: Parameterization of projected area
				float r = sqrt(Xi.x);
				float phi = 2.0 * PI * Xi.y;
				float t1 = r * cos(phi);
				float t2 = r * sin(phi);
				float s = 0.5 * (1.0 + V.z);
				t2 = (1.0 - s) * sqrt(1.0 - t1 * t1) + s * t2;

				// Section 4.3: Reprojection onto hemisphere
				vec3 Nh = t1 * T1 + t2 * T2 + sqrt(max(0.0, 1.0 - t1 * t1 - t2 * t2)) * V;

				// Section 3.4: Transform back to ellipsoid configuration
				return normalize(vec3(alpha * Nh.x, alpha * Nh.y, max(0.0, Nh.z)));
			}

			void main() {
				vec3 N = normalize(vOutputDirection);
				vec3 V = N; // Assume view direction equals normal for pre-filtering

				vec3 prefilteredColor = vec3(0.0);
				float totalWeight = 0.0;

				// For very low roughness, just sample the environment directly
				if (roughness < 0.001) {
					gl_FragColor = vec4(bilinearCubeUV(envMap, N, mipInt), 1.0);
					return;
				}

				// Tangent space basis for VNDF sampling
				vec3 up = abs(N.z) < 0.999 ? vec3(0.0, 0.0, 1.0) : vec3(1.0, 0.0, 0.0);
				vec3 tangent = normalize(cross(up, N));
				vec3 bitangent = cross(N, tangent);

				for(uint i = 0u; i < uint(GGX_SAMPLES); i++) {
					vec2 Xi = hammersley(i, uint(GGX_SAMPLES));

					// For PMREM, V = N, so in tangent space V is always (0, 0, 1)
					vec3 H_tangent = importanceSampleGGX_VNDF(Xi, vec3(0.0, 0.0, 1.0), roughness);

					// Transform H back to world space
					vec3 H = normalize(tangent * H_tangent.x + bitangent * H_tangent.y + N * H_tangent.z);
					vec3 L = normalize(2.0 * dot(V, H) * H - V);

					float NdotL = max(dot(N, L), 0.0);

					if(NdotL > 0.0) {
						// Sample environment at fixed mip level
						// VNDF importance sampling handles the distribution filtering
						vec3 sampleColor = bilinearCubeUV(envMap, L, mipInt);

						// Weight by NdotL for the split-sum approximation
						// VNDF PDF naturally accounts for the visible microfacet distribution
						prefilteredColor += sampleColor * NdotL;
						totalWeight += NdotL;
					}
				}

				if (totalWeight > 0.0) {
					prefilteredColor = prefilteredColor / totalWeight;
				}

				gl_FragColor = vec4(prefilteredColor, 1.0);
			}
		`,blending:Xn,depthTest:!1,depthWrite:!1})}function D_(i,e,t){return new tn({name:"SphericalGaussianBlur",defines:{SAMPLES:R_,CUBEUV_TEXEL_WIDTH:1/e,CUBEUV_TEXEL_HEIGHT:1/t,CUBEUV_MAX_MIP:`${i}.0`},uniforms:{envMap:{value:null},sigma:{value:0},mipInt:{value:0}},vertexShader:Rl(),fragmentShader:`

			precision highp float;
			precision highp int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;
			uniform float sigma;
			uniform float mipInt;

			#define ENVMAP_TYPE_CUBE_UV
			#include <cube_uv_reflection_fragment>

			#define PI 3.14159265359
			#define GOLDEN_ANGLE 2.39996322973

			void main() {

				if ( sigma == 0.0 ) {

					gl_FragColor = vec4( bilinearCubeUV( envMap, vOutputDirection, mipInt ), 1.0 );
					return;

				}

				vec3 outputDirection = normalize( vOutputDirection );

				vec3 up = abs( outputDirection.z ) < 0.999 ? vec3( 0.0, 0.0, 1.0 ) : vec3( 1.0, 0.0, 0.0 );
				vec3 tangent = normalize( cross( up, outputDirection ) );
				vec3 bitangent = cross( outputDirection, tangent );

				// Truncate the kernel at three standard deviations or at the antipode.
				float thetaMax = min( 3.0 * sigma, PI );
				float truncation = 1.0 - exp( - 0.5 * thetaMax * thetaMax / ( sigma * sigma ) );

				vec3 accumColor = vec3( 0.0 );
				float accumWeight = 0.0;

				for ( int i = 0; i < SAMPLES; i ++ ) {

					// Stratified inverse-CDF sampling of the Gaussian, placed on a golden-angle spiral.
					float stratum = ( float( i ) + 0.5 ) / float( SAMPLES );
					float theta = sigma * sqrt( - 2.0 * log( 1.0 - stratum * truncation ) );
					float phi = float( i ) * GOLDEN_ANGLE;

					vec3 offset = cos( phi ) * tangent + sin( phi ) * bitangent;
					vec3 sampleDirection = cos( theta ) * outputDirection + sin( theta ) * offset;

					// Correct the planar sample density to solid angle.
					float weight = sin( theta ) / theta;

					accumColor += weight * bilinearCubeUV( envMap, sampleDirection, mipInt );
					accumWeight += weight;

				}

				gl_FragColor = vec4( accumColor / accumWeight, 1.0 );

			}
		`,blending:Xn,depthTest:!1,depthWrite:!1})}function zd(){return new tn({name:"EquirectangularToCubeUV",uniforms:{envMap:{value:null}},vertexShader:Rl(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			varying vec3 vOutputDirection;

			uniform sampler2D envMap;

			#include <common>

			void main() {

				vec3 outputDirection = normalize( vOutputDirection );
				vec2 uv = equirectUv( outputDirection );

				gl_FragColor = vec4( texture2D ( envMap, uv ).rgb, 1.0 );

			}
		`,blending:Xn,depthTest:!1,depthWrite:!1})}function kd(){return new tn({name:"CubemapToCubeUV",uniforms:{envMap:{value:null},flipEnvMap:{value:-1}},vertexShader:Rl(),fragmentShader:`

			precision mediump float;
			precision mediump int;

			uniform float flipEnvMap;

			varying vec3 vOutputDirection;

			uniform samplerCube envMap;

			void main() {

				gl_FragColor = textureCube( envMap, vec3( flipEnvMap * vOutputDirection.x, vOutputDirection.yz ) );

			}
		`,blending:Xn,depthTest:!1,depthWrite:!1})}function Rl(){return`

		precision mediump float;
		precision mediump int;

		attribute vec3 outputDirection;

		varying vec3 vOutputDirection;

		void main() {

			vOutputDirection = outputDirection;
			gl_Position = vec4( position, 1.0 );

		}
	`}var wl=class extends sn{constructor(e=1,t={}){super(e,e,t),this.isWebGLCubeRenderTarget=!0;let n={width:e,height:e,depth:1},s=[n,n,n,n,n,n];this.texture=new Sr(s),this._setTextureOptions(t),this.texture.isRenderTargetTexture=!0}fromEquirectangularTexture(e,t){this.texture.type=t.type,this.texture.colorSpace=t.colorSpace,this.texture.generateMipmaps=t.generateMipmaps,this.texture.minFilter=t.minFilter,this.texture.magFilter=t.magFilter;let n={uniforms:{tEquirect:{value:null}},vertexShader:`

				varying vec3 vWorldDirection;

				vec3 transformDirection( in vec3 dir, in mat4 matrix ) {

					return normalize( ( matrix * vec4( dir, 0.0 ) ).xyz );

				}

				void main() {

					vWorldDirection = transformDirection( position, modelMatrix );

					#include <begin_vertex>
					#include <project_vertex>

				}
			`,fragmentShader:`

				uniform sampler2D tEquirect;

				varying vec3 vWorldDirection;

				#include <common>

				void main() {

					vec3 direction = normalize( vWorldDirection );

					vec2 sampleUV = equirectUv( direction );

					gl_FragColor = texture2D( tEquirect, sampleUV );

				}
			`},s=new Gn(5,5,5),r=new tn({name:"CubemapFromEquirect",uniforms:Zi(n.uniforms),vertexShader:n.vertexShader,fragmentShader:n.fragmentShader,side:Xt,blending:Xn});r.uniforms.tEquirect.value=t;let a=new mt(s,r),o=t.minFilter;return t.minFilter===Ci&&(t.minFilter=Gt),new Po(1,10,this).update(e,a),t.minFilter=o,a.geometry.dispose(),a.material.dispose(),this}clear(e,t=!0,n=!0,s=!0){let r=e.getRenderTarget();for(let a=0;a<6;a++)e.setRenderTarget(this,a),e.clear(t,n,s);e.setRenderTarget(r)}};function N_(i){let e=new WeakMap,t=new WeakMap,n=null;function s(u,f=!1){return u==null?null:f?a(u):r(u)}function r(u){if(u&&u.isTexture){let f=u.mapping;if(f===Fo||f===Oo)if(e.has(u)){let g=e.get(u).texture;return o(g,u.mapping)}else{let g=u.image;if(g&&g.height>0){let y=new wl(g.height);return y.fromEquirectangularTexture(i,u),e.set(u,y),u.addEventListener("dispose",c),o(y.texture,u.mapping)}else return null}}return u}function a(u){if(u&&u.isTexture){let f=u.mapping,g=f===Fo||f===Oo,y=f===Ri||f===qi;if(g||y){let m=t.get(u),p=m!==void 0?m.texture.pmremVersion:0;if(u.isRenderTargetTexture&&u.pmremVersion!==p)return n===null&&(n=new Ws(i)),m=g?n.fromEquirectangular(u,m):n.fromCubemap(u,m),m.texture.pmremVersion=u.pmremVersion,t.set(u,m),m.texture;if(m!==void 0)return m.texture;{let w=u.image;return g&&w&&w.height>0||y&&w&&l(w)?(n===null&&(n=new Ws(i)),m=g?n.fromEquirectangular(u):n.fromCubemap(u),m.texture.pmremVersion=u.pmremVersion,t.set(u,m),u.addEventListener("dispose",h),m.texture):null}}}return u}function o(u,f){return f===Fo?u.mapping=Ri:f===Oo&&(u.mapping=qi),u}function l(u){let f=0,g=6;for(let y=0;y<g;y++)u[y]!==void 0&&f++;return f===g}function c(u){let f=u.target;f.removeEventListener("dispose",c);let g=e.get(f);g!==void 0&&(e.delete(f),g.dispose())}function h(u){let f=u.target;f.removeEventListener("dispose",h);let g=t.get(f);g!==void 0&&(t.delete(f),g.dispose())}function d(){e=new WeakMap,t=new WeakMap,n!==null&&(n.dispose(),n=null)}return{get:s,dispose:d}}function U_(i){let e={};function t(n){if(e[n]!==void 0)return e[n];let s=i.getExtension(n);return e[n]=s,s}return{has:function(n){return t(n)!==null},init:function(){t("EXT_color_buffer_float"),t("WEBGL_clip_cull_distance"),t("OES_texture_float_linear"),t("EXT_color_buffer_half_float"),t("WEBGL_multisampled_render_to_texture"),t("WEBGL_render_shared_exponent")},get:function(n){let s=t(n);return s===null&&Vi("WebGLRenderer: "+n+" extension not supported."),s}}}function F_(i,e,t,n){let s={},r=new WeakMap;function a(d){let u=d.target;u.index!==null&&e.remove(u.index);for(let g in u.attributes)e.remove(u.attributes[g]);u.removeEventListener("dispose",a),delete s[u.id];let f=r.get(u);f&&(e.remove(f),r.delete(u)),n.releaseStatesOfGeometry(u),u.isInstancedBufferGeometry===!0&&delete u._maxInstanceCount,t.memory.geometries--}function o(d,u){return s[u.id]===!0||(u.addEventListener("dispose",a),s[u.id]=!0,t.memory.geometries++),u}function l(d){let u=d.attributes;for(let f in u)e.update(u[f],i.ARRAY_BUFFER)}function c(d){let u=[],f=d.index,g=d.attributes.position,y=0;if(g===void 0)return;if(f!==null){let w=f.array;y=f.version;for(let A=0,M=w.length;A<M;A+=3){let E=w[A+0],S=w[A+1],C=w[A+2];u.push(E,S,S,C,C,E)}}else{let w=g.array;y=g.version;for(let A=0,M=w.length/3-1;A<M;A+=3){let E=A+0,S=A+1,C=A+2;u.push(E,S,S,C,C,E)}}let m=new(g.count>=65535?_r:gr)(u,1);m.version=y;let p=r.get(d);p&&e.remove(p),r.set(d,m)}function h(d){let u=r.get(d);if(u){let f=d.index;f!==null&&u.version<f.version&&c(d)}else c(d);return r.get(d)}return{get:o,update:l,getWireframeAttribute:h}}function O_(i,e,t){let n;function s(d){n=d}let r,a;function o(d){r=d.type,a=d.bytesPerElement}function l(d,u){i.drawElements(n,u,r,d*a),t.update(u,n,1)}function c(d,u,f){f!==0&&(i.drawElementsInstanced(n,u,r,d*a,f),t.update(u,n,f))}function h(d,u,f){if(f===0)return;e.get("WEBGL_multi_draw").multiDrawElementsWEBGL(n,u,0,r,d,0,f);let y=0;for(let m=0;m<f;m++)y+=u[m];t.update(y,n,1)}this.setMode=s,this.setIndex=o,this.render=l,this.renderInstances=c,this.renderMultiDraw=h}function B_(i){let e={geometries:0,textures:0},t={frame:0,calls:0,triangles:0,points:0,lines:0};function n(r,a,o){switch(t.calls++,a){case i.TRIANGLES:t.triangles+=o*(r/3);break;case i.LINES:t.lines+=o*(r/2);break;case i.LINE_STRIP:t.lines+=o*(r-1);break;case i.LINE_LOOP:t.lines+=o*r;break;case i.POINTS:t.points+=o*r;break;default:Ve("WebGLInfo: Unknown draw mode:",a);break}}function s(){t.calls=0,t.triangles=0,t.points=0,t.lines=0}return{memory:e,render:t,programs:null,autoReset:!0,reset:s,update:n}}function z_(i,e,t){let n=new WeakMap,s=new St;function r(a,o,l){let c=a.morphTargetInfluences,h=o.morphAttributes.position||o.morphAttributes.normal||o.morphAttributes.color,d=h!==void 0?h.length:0,u=n.get(o);if(u===void 0||u.count!==d){let T=function(){C.dispose(),n.delete(o),o.removeEventListener("dispose",T)};u!==void 0&&u.texture.dispose();let f=o.morphAttributes.position!==void 0,g=o.morphAttributes.normal!==void 0,y=o.morphAttributes.color!==void 0,m=o.morphAttributes.position||[],p=o.morphAttributes.normal||[],w=o.morphAttributes.color||[],A=0;f===!0&&(A=1),g===!0&&(A=2),y===!0&&(A=3);let M=o.attributes.position.count*A,E=1;M>e.maxTextureSize&&(E=Math.ceil(M/e.maxTextureSize),M=e.maxTextureSize);let S=new Float32Array(M*E*4*d),C=new mr(S,M,E,d);C.type=vn,C.needsUpdate=!0;let x=A*4;for(let I=0;I<d;I++){let U=m[I],O=p[I],H=w[I],D=M*E*4*I;for(let B=0;B<U.count;B++){let X=B*x;f===!0&&(s.fromBufferAttribute(U,B),S[D+X+0]=s.x,S[D+X+1]=s.y,S[D+X+2]=s.z,S[D+X+3]=0),g===!0&&(s.fromBufferAttribute(O,B),S[D+X+4]=s.x,S[D+X+5]=s.y,S[D+X+6]=s.z,S[D+X+7]=0),y===!0&&(s.fromBufferAttribute(H,B),S[D+X+8]=s.x,S[D+X+9]=s.y,S[D+X+10]=s.z,S[D+X+11]=H.itemSize===4?s.w:1)}}u={count:d,texture:C,size:new ae(M,E)},n.set(o,u),o.addEventListener("dispose",T)}if(a.isInstancedMesh===!0&&a.morphTexture!==null)l.getUniforms().setValue(i,"morphTexture",a.morphTexture,t);else{let f=0;for(let y=0;y<c.length;y++)f+=c[y];let g=o.morphTargetsRelative?1:1-f;l.getUniforms().setValue(i,"morphTargetBaseInfluence",g),l.getUniforms().setValue(i,"morphTargetInfluences",c)}l.getUniforms().setValue(i,"morphTargetsTexture",u.texture,t),l.getUniforms().setValue(i,"morphTargetsTextureSize",u.size)}return{update:r}}function k_(i,e,t,n,s){let r=new WeakMap;function a(c){let h=s.render.frame,d=c.geometry,u=e.get(c,d);if(r.get(u)!==h&&(e.update(u),r.set(u,h)),c.isInstancedMesh&&(c.hasEventListener("dispose",l)===!1&&c.addEventListener("dispose",l),r.get(c)!==h&&(t.update(c.instanceMatrix,i.ARRAY_BUFFER),c.instanceColor!==null&&t.update(c.instanceColor,i.ARRAY_BUFFER),r.set(c,h))),c.isSkinnedMesh){let f=c.skeleton;r.get(f)!==h&&(f.update(),r.set(f,h))}return u}function o(){r=new WeakMap}function l(c){let h=c.target;h.removeEventListener("dispose",l),n.releaseStatesOfObject(h),t.remove(h.instanceMatrix),h.instanceColor!==null&&t.remove(h.instanceColor)}return{update:a,dispose:o}}var V_={[zc]:"LINEAR_TONE_MAPPING",[kc]:"REINHARD_TONE_MAPPING",[Vc]:"CINEON_TONE_MAPPING",[qr]:"ACES_FILMIC_TONE_MAPPING",[Gc]:"AGX_TONE_MAPPING",[Wc]:"NEUTRAL_TONE_MAPPING",[Hc]:"CUSTOM_TONE_MAPPING"};function H_(i,e,t,n,s,r){let a=new sn(e,t,{type:i,depthBuffer:s,stencilBuffer:r,samples:n?4:0,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,resolveDepthBuffer:!1,resolveStencilBuffer:!1}),o=null,l=null,c=new Ut;c.setAttribute("position",new Tt([-1,3,0,-1,-1,0,3,-1,0],3)),c.setAttribute("uv",new Tt([0,2,0,0,2,0],2));let h=new _o({uniforms:{tDiffuse:{value:null}},vertexShader:`
			precision highp float;

			uniform mat4 modelViewMatrix;
			uniform mat4 projectionMatrix;

			attribute vec3 position;
			attribute vec2 uv;

			varying vec2 vUv;

			void main() {
				vUv = uv;
				gl_Position = projectionMatrix * modelViewMatrix * vec4( position, 1.0 );
			}`,fragmentShader:`
			precision highp float;

			uniform sampler2D tDiffuse;

			varying vec2 vUv;

			#include <tonemapping_pars_fragment>
			#include <colorspace_pars_fragment>

			void main() {
				gl_FragColor = texture2D( tDiffuse, vUv );

				#ifdef LINEAR_TONE_MAPPING
					gl_FragColor.rgb = LinearToneMapping( gl_FragColor.rgb );
				#elif defined( REINHARD_TONE_MAPPING )
					gl_FragColor.rgb = ReinhardToneMapping( gl_FragColor.rgb );
				#elif defined( CINEON_TONE_MAPPING )
					gl_FragColor.rgb = CineonToneMapping( gl_FragColor.rgb );
				#elif defined( ACES_FILMIC_TONE_MAPPING )
					gl_FragColor.rgb = ACESFilmicToneMapping( gl_FragColor.rgb );
				#elif defined( AGX_TONE_MAPPING )
					gl_FragColor.rgb = AgXToneMapping( gl_FragColor.rgb );
				#elif defined( NEUTRAL_TONE_MAPPING )
					gl_FragColor.rgb = NeutralToneMapping( gl_FragColor.rgb );
				#elif defined( CUSTOM_TONE_MAPPING )
					gl_FragColor.rgb = CustomToneMapping( gl_FragColor.rgb );
				#endif

				#ifdef SRGB_TRANSFER
					gl_FragColor = sRGBTransferOETF( gl_FragColor );
				#endif
			}`,depthTest:!1,depthWrite:!1}),d=new mt(c,h),u=new Ns(-1,1,1,-1,0,1),f=null,g=null,y=!1,m,p=null,w=[],A=!1;this.setSize=function(M,E){a.setSize(M,E),o!==null&&o.setSize(M,E),l!==null&&l.setSize(M,E);for(let S=0;S<w.length;S++){let C=w[S];C.setSize&&C.setSize(M,E)}},this.setEffects=function(M){w=M,A=w.length>0&&w[0].isRenderPass===!0;let E=a.width,S=a.height;w.length>0&&o===null&&(o=new sn(E,S,{type:Un,depthBuffer:!1,stencilBuffer:!1}),l=new sn(E,S,{type:Un,depthBuffer:!1,stencilBuffer:!1}));for(let C=0;C<w.length;C++){let x=w[C];x.setSize&&x.setSize(E,S)}},this.begin=function(M,E){if(y||M.toneMapping===Dn&&w.length===0)return!1;if(p=E,E!==null){let S=E.width,C=E.height;(a.width!==S||a.height!==C)&&this.setSize(S,C)}return A===!1&&M.setRenderTarget(a),m=M.toneMapping,M.toneMapping=Dn,!0},this.hasRenderPass=function(){return A},this.end=function(M,E){M.toneMapping=m,y=!0;let S=a,C=o;for(let x=0;x<w.length;x++){let T=w[x];T.enabled!==!1&&(T.render(M,C,S,E),T.needsSwap!==!1&&(S=C,C=C===o?l:o))}if(f!==M.outputColorSpace||g!==M.toneMapping){f=M.outputColorSpace,g=M.toneMapping,h.defines={},it.getTransfer(f)===ut&&(h.defines.SRGB_TRANSFER="");let x=V_[g];x&&(h.defines[x]=""),h.needsUpdate=!0}h.uniforms.tDiffuse.value=S.texture,M.setRenderTarget(p),M.render(d,u),p=null,y=!1},this.isCompositing=function(){return y},this.dispose=function(){a.dispose(),o!==null&&o.dispose(),l!==null&&l.dispose(),c.dispose(),h.dispose()}}var of=new en,_h=new vi(1,1),lf=new mr,cf=new so,hf=new Sr,Vd=[],Hd=[],Gd=new Float32Array(16),Wd=new Float32Array(9),Xd=new Float32Array(4);function Xs(i,e,t){let n=i[0];if(n<=0||n>0)return i;let s=e*t,r=Vd[s];if(r===void 0&&(r=new Float32Array(s),Vd[s]=r),e!==0){n.toArray(r,0);for(let a=1,o=0;a!==e;++a)o+=t,i[a].toArray(r,o)}return r}function Ft(i,e){if(i.length!==e.length)return!1;for(let t=0,n=i.length;t<n;t++)if(i[t]!==e[t])return!1;return!0}function Ot(i,e){for(let t=0,n=e.length;t<n;t++)i[t]=e[t]}function Cl(i,e){let t=Hd[e];t===void 0&&(t=new Int32Array(e),Hd[e]=t);for(let n=0;n!==e;++n)t[n]=i.allocateTextureUnit();return t}function G_(i,e){let t=this.cache;t[0]!==e&&(i.uniform1f(this.addr,e),t[0]=e)}function W_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2f(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(Ft(t,e))return;i.uniform2fv(this.addr,e),Ot(t,e)}}function X_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3f(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else if(e.r!==void 0)(t[0]!==e.r||t[1]!==e.g||t[2]!==e.b)&&(i.uniform3f(this.addr,e.r,e.g,e.b),t[0]=e.r,t[1]=e.g,t[2]=e.b);else{if(Ft(t,e))return;i.uniform3fv(this.addr,e),Ot(t,e)}}function q_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4f(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(Ft(t,e))return;i.uniform4fv(this.addr,e),Ot(t,e)}}function Y_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(Ft(t,e))return;i.uniformMatrix2fv(this.addr,!1,e),Ot(t,e)}else{if(Ft(t,n))return;Xd.set(n),i.uniformMatrix2fv(this.addr,!1,Xd),Ot(t,n)}}function Z_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(Ft(t,e))return;i.uniformMatrix3fv(this.addr,!1,e),Ot(t,e)}else{if(Ft(t,n))return;Wd.set(n),i.uniformMatrix3fv(this.addr,!1,Wd),Ot(t,n)}}function $_(i,e){let t=this.cache,n=e.elements;if(n===void 0){if(Ft(t,e))return;i.uniformMatrix4fv(this.addr,!1,e),Ot(t,e)}else{if(Ft(t,n))return;Gd.set(n),i.uniformMatrix4fv(this.addr,!1,Gd),Ot(t,n)}}function J_(i,e){let t=this.cache;t[0]!==e&&(i.uniform1i(this.addr,e),t[0]=e)}function j_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2i(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(Ft(t,e))return;i.uniform2iv(this.addr,e),Ot(t,e)}}function K_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3i(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(Ft(t,e))return;i.uniform3iv(this.addr,e),Ot(t,e)}}function Q_(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4i(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(Ft(t,e))return;i.uniform4iv(this.addr,e),Ot(t,e)}}function ex(i,e){let t=this.cache;t[0]!==e&&(i.uniform1ui(this.addr,e),t[0]=e)}function tx(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y)&&(i.uniform2ui(this.addr,e.x,e.y),t[0]=e.x,t[1]=e.y);else{if(Ft(t,e))return;i.uniform2uiv(this.addr,e),Ot(t,e)}}function nx(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z)&&(i.uniform3ui(this.addr,e.x,e.y,e.z),t[0]=e.x,t[1]=e.y,t[2]=e.z);else{if(Ft(t,e))return;i.uniform3uiv(this.addr,e),Ot(t,e)}}function ix(i,e){let t=this.cache;if(e.x!==void 0)(t[0]!==e.x||t[1]!==e.y||t[2]!==e.z||t[3]!==e.w)&&(i.uniform4ui(this.addr,e.x,e.y,e.z,e.w),t[0]=e.x,t[1]=e.y,t[2]=e.z,t[3]=e.w);else{if(Ft(t,e))return;i.uniform4uiv(this.addr,e),Ot(t,e)}}function sx(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s);let r;this.type===i.SAMPLER_2D_SHADOW?(_h.compareFunction=t.isReversedDepthBuffer()?Sl:bl,r=_h):r=of,t.setTexture2D(e||r,s)}function rx(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTexture3D(e||cf,s)}function ax(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTextureCube(e||hf,s)}function ox(i,e,t){let n=this.cache,s=t.allocateTextureUnit();n[0]!==s&&(i.uniform1i(this.addr,s),n[0]=s),t.setTexture2DArray(e||lf,s)}function lx(i){switch(i){case 5126:return G_;case 35664:return W_;case 35665:return X_;case 35666:return q_;case 35674:return Y_;case 35675:return Z_;case 35676:return $_;case 5124:case 35670:return J_;case 35667:case 35671:return j_;case 35668:case 35672:return K_;case 35669:case 35673:return Q_;case 5125:return ex;case 36294:return tx;case 36295:return nx;case 36296:return ix;case 35678:case 36198:case 36298:case 36306:case 35682:return sx;case 35679:case 36299:case 36307:return rx;case 35680:case 36300:case 36308:case 36293:return ax;case 36289:case 36303:case 36311:case 36292:return ox}}function cx(i,e){i.uniform1fv(this.addr,e)}function hx(i,e){let t=Xs(e,this.size,2);i.uniform2fv(this.addr,t)}function ux(i,e){let t=Xs(e,this.size,3);i.uniform3fv(this.addr,t)}function dx(i,e){let t=Xs(e,this.size,4);i.uniform4fv(this.addr,t)}function fx(i,e){let t=Xs(e,this.size,4);i.uniformMatrix2fv(this.addr,!1,t)}function px(i,e){let t=Xs(e,this.size,9);i.uniformMatrix3fv(this.addr,!1,t)}function mx(i,e){let t=Xs(e,this.size,16);i.uniformMatrix4fv(this.addr,!1,t)}function gx(i,e){i.uniform1iv(this.addr,e)}function _x(i,e){i.uniform2iv(this.addr,e)}function xx(i,e){i.uniform3iv(this.addr,e)}function yx(i,e){i.uniform4iv(this.addr,e)}function vx(i,e){i.uniform1uiv(this.addr,e)}function Mx(i,e){i.uniform2uiv(this.addr,e)}function bx(i,e){i.uniform3uiv(this.addr,e)}function Sx(i,e){i.uniform4uiv(this.addr,e)}function Ex(i,e,t){let n=this.cache,s=e.length,r=Cl(t,s);Ft(n,r)||(i.uniform1iv(this.addr,r),Ot(n,r));let a;this.type===i.SAMPLER_2D_SHADOW?a=_h:a=of;for(let o=0;o!==s;++o)t.setTexture2D(e[o]||a,r[o])}function Tx(i,e,t){let n=this.cache,s=e.length,r=Cl(t,s);Ft(n,r)||(i.uniform1iv(this.addr,r),Ot(n,r));for(let a=0;a!==s;++a)t.setTexture3D(e[a]||cf,r[a])}function wx(i,e,t){let n=this.cache,s=e.length,r=Cl(t,s);Ft(n,r)||(i.uniform1iv(this.addr,r),Ot(n,r));for(let a=0;a!==s;++a)t.setTextureCube(e[a]||hf,r[a])}function Ax(i,e,t){let n=this.cache,s=e.length,r=Cl(t,s);Ft(n,r)||(i.uniform1iv(this.addr,r),Ot(n,r));for(let a=0;a!==s;++a)t.setTexture2DArray(e[a]||lf,r[a])}function Rx(i){switch(i){case 5126:return cx;case 35664:return hx;case 35665:return ux;case 35666:return dx;case 35674:return fx;case 35675:return px;case 35676:return mx;case 5124:case 35670:return gx;case 35667:case 35671:return _x;case 35668:case 35672:return xx;case 35669:case 35673:return yx;case 5125:return vx;case 36294:return Mx;case 36295:return bx;case 36296:return Sx;case 35678:case 36198:case 36298:case 36306:case 35682:return Ex;case 35679:case 36299:case 36307:return Tx;case 35680:case 36300:case 36308:case 36293:return wx;case 36289:case 36303:case 36311:case 36292:return Ax}}var xh=class{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.setValue=lx(t.type)}},yh=class{constructor(e,t,n){this.id=e,this.addr=n,this.cache=[],this.type=t.type,this.size=t.size,this.setValue=Rx(t.type)}},vh=class{constructor(e){this.id=e,this.seq=[],this.map={}}setValue(e,t,n){let s=this.seq;for(let r=0,a=s.length;r!==a;++r){let o=s[r];o.setValue(e,t[o.id],n)}}},mh=/(\w+)(\])?(\[|\.)?/g;function qd(i,e){i.seq.push(e),i.map[e.id]=e}function Cx(i,e,t){let n=i.name,s=n.length;for(mh.lastIndex=0;;){let r=mh.exec(n),a=mh.lastIndex,o=r[1],l=r[2]==="]",c=r[3];if(l&&(o=o|0),c===void 0||c==="["&&a+2===s){qd(t,c===void 0?new xh(o,i,e):new yh(o,i,e));break}else{let d=t.map[o];d===void 0&&(d=new vh(o),qd(t,d)),t=d}}}var Gs=class{constructor(e,t){this.seq=[],this.map={};let n=e.getProgramParameter(t,e.ACTIVE_UNIFORMS);for(let a=0;a<n;++a){let o=e.getActiveUniform(t,a),l=e.getUniformLocation(t,o.name);Cx(o,l,this)}let s=[],r=[];for(let a of this.seq)a.type===e.SAMPLER_2D_SHADOW||a.type===e.SAMPLER_CUBE_SHADOW||a.type===e.SAMPLER_2D_ARRAY_SHADOW?s.push(a):r.push(a);s.length>0&&(this.seq=s.concat(r))}setValue(e,t,n,s){let r=this.map[t];r!==void 0&&r.setValue(e,n,s)}setOptional(e,t,n){let s=t[n];s!==void 0&&this.setValue(e,n,s)}static upload(e,t,n,s){for(let r=0,a=t.length;r!==a;++r){let o=t[r],l=n[o.id];l.needsUpdate!==!1&&o.setValue(e,l.value,s)}}static seqWithValue(e,t){let n=[];for(let s=0,r=e.length;s!==r;++s){let a=e[s];a.id in t&&n.push(a)}return n}};function Yd(i,e,t){let n=i.createShader(e);return i.shaderSource(n,t),i.compileShader(n),n}var Px=37297,Ix=0;function Lx(i,e){let t=i.split(`
`),n=[],s=Math.max(e-6,0),r=Math.min(e+6,t.length);for(let a=s;a<r;a++){let o=a+1;n.push(`${o===e?">":" "} ${o}: ${t[a]}`)}return n.join(`
`)}var Zd=new Xe;function Dx(i){it._getMatrix(Zd,it.workingColorSpace,i);let e=`mat3( ${Zd.elements.map(t=>t.toFixed(4))} )`;switch(it.getTransfer(i)){case fr:return[e,"LinearTransferOETF"];case ut:return[e,"sRGBTransferOETF"];default:return Oe("WebGLProgram: Unsupported color space: ",i),[e,"LinearTransferOETF"]}}function $d(i,e,t){let n=i.getShaderParameter(e,i.COMPILE_STATUS),r=(i.getShaderInfoLog(e)||"").trim();if(n&&r==="")return"";let a=/ERROR: 0:(\d+)/.exec(r);if(a){let o=parseInt(a[1]);return t.toUpperCase()+`

`+r+`

`+Lx(i.getShaderSource(e),o)}else return r}function Nx(i,e){let t=Dx(e);return[`vec4 ${i}( vec4 value ) {`,`	return ${t[1]}( vec4( value.rgb * ${t[0]}, value.a ) );`,"}"].join(`
`)}var Ux={[zc]:"Linear",[kc]:"Reinhard",[Vc]:"Cineon",[qr]:"ACESFilmic",[Gc]:"AgX",[Wc]:"Neutral",[Hc]:"Custom"};function Fx(i,e){let t=Ux[e];return t===void 0?(Oe("WebGLProgram: Unsupported toneMapping:",e),"vec3 "+i+"( vec3 color ) { return LinearToneMapping( color ); }"):"vec3 "+i+"( vec3 color ) { return "+t+"ToneMapping( color ); }"}var Tl=new R;function Ox(){it.getLuminanceCoefficients(Tl);let i=Tl.x.toFixed(4),e=Tl.y.toFixed(4),t=Tl.z.toFixed(4);return["float luminance( const in vec3 rgb ) {",`	const vec3 weights = vec3( ${i}, ${e}, ${t} );`,"	return dot( weights, rgb );","}"].join(`
`)}function Bx(i){return[i.extensionClipCullDistance?"#extension GL_ANGLE_clip_cull_distance : require":"",i.extensionMultiDraw?"#extension GL_ANGLE_multi_draw : require":""].filter(ra).join(`
`)}function zx(i){let e=[];for(let t in i){let n=i[t];n!==!1&&e.push("#define "+t+" "+n)}return e.join(`
`)}function kx(i,e){let t={},n=i.getProgramParameter(e,i.ACTIVE_ATTRIBUTES);for(let s=0;s<n;s++){let r=i.getActiveAttrib(e,s),a=r.name,o=1;r.type===i.FLOAT_MAT2&&(o=2),r.type===i.FLOAT_MAT3&&(o=3),r.type===i.FLOAT_MAT4&&(o=4),t[a]={type:r.type,location:i.getAttribLocation(e,a),locationSize:o}}return t}function ra(i){return i!==""}function Jd(i,e){let t=e.numSpotLightShadows+e.numSpotLightMaps-e.numSpotLightShadowsWithMaps;return i.replace(/NUM_SUN_LIGHTS/g,e.numSunLights).replace(/NUM_DIR_LIGHTS/g,e.numDirLights).replace(/NUM_SPOT_LIGHTS/g,e.numSpotLights).replace(/NUM_SPOT_LIGHT_MAPS/g,e.numSpotLightMaps).replace(/NUM_SPOT_LIGHT_COORDS/g,t).replace(/NUM_RECT_AREA_LIGHTS/g,e.numRectAreaLights).replace(/NUM_POINT_LIGHTS/g,e.numPointLights).replace(/NUM_HEMI_LIGHTS/g,e.numHemiLights).replace(/NUM_SUN_LIGHT_SHADOWS/g,e.numSunLightShadows).replace(/NUM_DIR_LIGHT_SHADOWS/g,e.numDirLightShadows).replace(/NUM_SPOT_LIGHT_SHADOWS_WITH_MAPS/g,e.numSpotLightShadowsWithMaps).replace(/NUM_SPOT_LIGHT_SHADOWS/g,e.numSpotLightShadows).replace(/NUM_POINT_LIGHT_SHADOWS/g,e.numPointLightShadows)}function jd(i,e){return i.replace(/NUM_CLIPPING_PLANES/g,e.numClippingPlanes).replace(/UNION_CLIPPING_PLANES/g,e.numClippingPlanes-e.numClipIntersection)}var Vx=/^[ \t]*#include +<([\w\d./]+)>/gm;function Mh(i){return i.replace(Vx,Gx)}var Hx=new Map;function Gx(i,e){let t=Ke[e];if(t===void 0){let n=Hx.get(e);if(n!==void 0)t=Ke[n],Oe('WebGLRenderer: Shader chunk "%s" has been deprecated. Use "%s" instead.',e,n);else throw new Error("THREE.WebGLProgram: Can not resolve #include <"+e+">")}return Mh(t)}var Wx=/#pragma unroll_loop_start\s+for\s*\(\s*int\s+i\s*=\s*(\d+)\s*;\s*i\s*<\s*(\d+)\s*;\s*i\s*\+\+\s*\)\s*{([\s\S]+?)}\s+#pragma unroll_loop_end/g;function Kd(i){return i.replace(Wx,Xx)}function Xx(i,e,t,n){let s="";for(let r=parseInt(e);r<parseInt(t);r++)s+=n.replace(/\[\s*i\s*\]/g,"[ "+r+" ]").replace(/UNROLLED_LOOP_INDEX/g,r);return s}function Qd(i){let e=`precision ${i.precision} float;
	precision ${i.precision} int;
	precision ${i.precision} sampler2D;
	precision ${i.precision} samplerCube;
	precision ${i.precision} sampler3D;
	precision ${i.precision} sampler2DArray;
	precision ${i.precision} sampler2DShadow;
	precision ${i.precision} samplerCubeShadow;
	precision ${i.precision} sampler2DArrayShadow;
	precision ${i.precision} isampler2D;
	precision ${i.precision} isampler3D;
	precision ${i.precision} isamplerCube;
	precision ${i.precision} isampler2DArray;
	precision ${i.precision} usampler2D;
	precision ${i.precision} usampler3D;
	precision ${i.precision} usamplerCube;
	precision ${i.precision} usampler2DArray;
	`;return i.precision==="highp"?e+=`
#define HIGH_PRECISION`:i.precision==="mediump"?e+=`
#define MEDIUM_PRECISION`:i.precision==="lowp"&&(e+=`
#define LOW_PRECISION`),e}var qx={[Xr]:"SHADOWMAP_TYPE_PCF",[Os]:"SHADOWMAP_TYPE_VSM"};function Yx(i){return qx[i.shadowMapType]||"SHADOWMAP_TYPE_BASIC"}var Zx={[Ri]:"ENVMAP_TYPE_CUBE",[qi]:"ENVMAP_TYPE_CUBE",[Yr]:"ENVMAP_TYPE_CUBE_UV"};function $x(i){return i.envMap===!1?"ENVMAP_TYPE_CUBE":Zx[i.envMapMode]||"ENVMAP_TYPE_CUBE"}var Jx={[qi]:"ENVMAP_MODE_REFRACTION"};function jx(i){return i.envMap===!1?"ENVMAP_MODE_REFLECTION":Jx[i.envMapMode]||"ENVMAP_MODE_REFLECTION"}var Kx={[Uo]:"ENVMAP_BLENDING_MULTIPLY",[dd]:"ENVMAP_BLENDING_MIX",[fd]:"ENVMAP_BLENDING_ADD"};function Qx(i){return i.envMap===!1?"ENVMAP_BLENDING_NONE":Kx[i.combine]||"ENVMAP_BLENDING_NONE"}function ey(i){let e=i.envMapCubeUVHeight;if(e===null)return null;let t=Math.log2(e)-2,n=1/e;return{texelWidth:1/(3*Math.max(Math.pow(2,t),7*16)),texelHeight:n,maxMip:t}}function ty(i,e,t,n){let s=i.getContext(),r=t.defines,a=t.vertexShader,o=t.fragmentShader,l=Yx(t),c=$x(t),h=jx(t),d=Qx(t),u=ey(t),f=Bx(t),g=zx(r),y=s.createProgram(),m,p,w=t.glslVersion?"#version "+t.glslVersion+`
`:"";t.isRawShaderMaterial?(m=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,g].filter(ra).join(`
`),m.length>0&&(m+=`
`),p=["#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,g].filter(ra).join(`
`),p.length>0&&(p+=`
`)):(m=[Qd(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,g,t.extensionClipCullDistance?"#define USE_CLIP_DISTANCE":"",t.batching?"#define USE_BATCHING":"",t.batchingColor?"#define USE_BATCHING_COLOR":"",t.instancing?"#define USE_INSTANCING":"",t.instancingColor?"#define USE_INSTANCING_COLOR":"",t.instancingMorph?"#define USE_INSTANCING_MORPH":"",t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.map?"#define USE_MAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+h:"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.displacementMap?"#define USE_DISPLACEMENTMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.mapUv?"#define MAP_UV "+t.mapUv:"",t.alphaMapUv?"#define ALPHAMAP_UV "+t.alphaMapUv:"",t.lightMapUv?"#define LIGHTMAP_UV "+t.lightMapUv:"",t.aoMapUv?"#define AOMAP_UV "+t.aoMapUv:"",t.emissiveMapUv?"#define EMISSIVEMAP_UV "+t.emissiveMapUv:"",t.bumpMapUv?"#define BUMPMAP_UV "+t.bumpMapUv:"",t.normalMapUv?"#define NORMALMAP_UV "+t.normalMapUv:"",t.displacementMapUv?"#define DISPLACEMENTMAP_UV "+t.displacementMapUv:"",t.metalnessMapUv?"#define METALNESSMAP_UV "+t.metalnessMapUv:"",t.roughnessMapUv?"#define ROUGHNESSMAP_UV "+t.roughnessMapUv:"",t.anisotropyMapUv?"#define ANISOTROPYMAP_UV "+t.anisotropyMapUv:"",t.clearcoatMapUv?"#define CLEARCOATMAP_UV "+t.clearcoatMapUv:"",t.clearcoatNormalMapUv?"#define CLEARCOAT_NORMALMAP_UV "+t.clearcoatNormalMapUv:"",t.clearcoatRoughnessMapUv?"#define CLEARCOAT_ROUGHNESSMAP_UV "+t.clearcoatRoughnessMapUv:"",t.iridescenceMapUv?"#define IRIDESCENCEMAP_UV "+t.iridescenceMapUv:"",t.iridescenceThicknessMapUv?"#define IRIDESCENCE_THICKNESSMAP_UV "+t.iridescenceThicknessMapUv:"",t.sheenColorMapUv?"#define SHEEN_COLORMAP_UV "+t.sheenColorMapUv:"",t.sheenRoughnessMapUv?"#define SHEEN_ROUGHNESSMAP_UV "+t.sheenRoughnessMapUv:"",t.specularMapUv?"#define SPECULARMAP_UV "+t.specularMapUv:"",t.specularColorMapUv?"#define SPECULAR_COLORMAP_UV "+t.specularColorMapUv:"",t.specularIntensityMapUv?"#define SPECULAR_INTENSITYMAP_UV "+t.specularIntensityMapUv:"",t.transmissionMapUv?"#define TRANSMISSIONMAP_UV "+t.transmissionMapUv:"",t.thicknessMapUv?"#define THICKNESSMAP_UV "+t.thicknessMapUv:"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexNormals?"#define HAS_NORMAL":"",t.vertexColors?"#define USE_COLOR":"",t.vertexAlphas?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.flatShading?"#define FLAT_SHADED":"",t.skinning?"#define USE_SKINNING":"",t.morphTargets?"#define USE_MORPHTARGETS":"",t.morphNormals&&t.flatShading===!1?"#define USE_MORPHNORMALS":"",t.morphColors?"#define USE_MORPHCOLORS":"",t.morphTargetsCount>0?"#define MORPHTARGETS_TEXTURE_STRIDE "+t.morphTextureStride:"",t.morphTargetsCount>0?"#define MORPHTARGETS_COUNT "+t.morphTargetsCount:"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+l:"",t.sizeAttenuation?"#define USE_SIZEATTENUATION":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 modelMatrix;","uniform mat4 modelViewMatrix;","uniform mat4 projectionMatrix;","uniform mat4 viewMatrix;","uniform mat3 normalMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;","#ifdef USE_INSTANCING","	attribute mat4 instanceMatrix;","#endif","#ifdef USE_INSTANCING_COLOR","	attribute vec3 instanceColor;","#endif","#ifdef USE_INSTANCING_MORPH","	uniform sampler2D morphTexture;","#endif","attribute vec3 position;","attribute vec3 normal;","attribute vec2 uv;","#ifdef USE_UV1","	attribute vec2 uv1;","#endif","#ifdef USE_UV2","	attribute vec2 uv2;","#endif","#ifdef USE_UV3","	attribute vec2 uv3;","#endif","#ifdef USE_TANGENT","	attribute vec4 tangent;","#endif","#if defined( USE_COLOR_ALPHA )","	attribute vec4 color;","#elif defined( USE_COLOR )","	attribute vec3 color;","#endif","#ifdef USE_SKINNING","	attribute vec4 skinIndex;","	attribute vec4 skinWeight;","#endif",`
`].filter(ra).join(`
`),p=[Qd(t),"#define SHADER_TYPE "+t.shaderType,"#define SHADER_NAME "+t.shaderName,g,t.useFog&&t.fog?"#define USE_FOG":"",t.useFog&&t.fogExp2?"#define FOG_EXP2":"",t.alphaToCoverage?"#define ALPHA_TO_COVERAGE":"",t.map?"#define USE_MAP":"",t.matcap?"#define USE_MATCAP":"",t.envMap?"#define USE_ENVMAP":"",t.envMap?"#define "+c:"",t.envMap?"#define "+h:"",t.envMap?"#define "+d:"",u?"#define CUBEUV_TEXEL_WIDTH "+u.texelWidth:"",u?"#define CUBEUV_TEXEL_HEIGHT "+u.texelHeight:"",u?"#define CUBEUV_MAX_MIP "+u.maxMip+".0":"",t.lightMap?"#define USE_LIGHTMAP":"",t.aoMap?"#define USE_AOMAP":"",t.bumpMap?"#define USE_BUMPMAP":"",t.normalMap?"#define USE_NORMALMAP":"",t.normalMapObjectSpace?"#define USE_NORMALMAP_OBJECTSPACE":"",t.normalMapTangentSpace?"#define USE_NORMALMAP_TANGENTSPACE":"",t.packedNormalMap?"#define USE_PACKED_NORMALMAP":"",t.emissiveMap?"#define USE_EMISSIVEMAP":"",t.anisotropy?"#define USE_ANISOTROPY":"",t.anisotropyMap?"#define USE_ANISOTROPYMAP":"",t.clearcoat?"#define USE_CLEARCOAT":"",t.clearcoatMap?"#define USE_CLEARCOATMAP":"",t.clearcoatRoughnessMap?"#define USE_CLEARCOAT_ROUGHNESSMAP":"",t.clearcoatNormalMap?"#define USE_CLEARCOAT_NORMALMAP":"",t.dispersion?"#define USE_DISPERSION":"",t.retroreflection?"#define USE_RETROREFLECTION":"",t.iridescence?"#define USE_IRIDESCENCE":"",t.iridescenceMap?"#define USE_IRIDESCENCEMAP":"",t.iridescenceThicknessMap?"#define USE_IRIDESCENCE_THICKNESSMAP":"",t.specularMap?"#define USE_SPECULARMAP":"",t.specularColorMap?"#define USE_SPECULAR_COLORMAP":"",t.specularIntensityMap?"#define USE_SPECULAR_INTENSITYMAP":"",t.roughnessMap?"#define USE_ROUGHNESSMAP":"",t.metalnessMap?"#define USE_METALNESSMAP":"",t.alphaMap?"#define USE_ALPHAMAP":"",t.alphaTest?"#define USE_ALPHATEST":"",t.alphaHash?"#define USE_ALPHAHASH":"",t.sheen?"#define USE_SHEEN":"",t.sheenColorMap?"#define USE_SHEEN_COLORMAP":"",t.sheenRoughnessMap?"#define USE_SHEEN_ROUGHNESSMAP":"",t.transmission?"#define USE_TRANSMISSION":"",t.transmissionMap?"#define USE_TRANSMISSIONMAP":"",t.thicknessMap?"#define USE_THICKNESSMAP":"",t.vertexTangents&&t.flatShading===!1?"#define USE_TANGENT":"",t.vertexColors||t.instancingColor?"#define USE_COLOR":"",t.vertexAlphas||t.batchingColor?"#define USE_COLOR_ALPHA":"",t.vertexUv1s?"#define USE_UV1":"",t.vertexUv2s?"#define USE_UV2":"",t.vertexUv3s?"#define USE_UV3":"",t.pointsUvs?"#define USE_POINTS_UV":"",t.gradientMap?"#define USE_GRADIENTMAP":"",t.flatShading?"#define FLAT_SHADED":"",t.doubleSided?"#define DOUBLE_SIDED":"",t.flipSided?"#define FLIP_SIDED":"",t.shadowMapEnabled?"#define USE_SHADOWMAP":"",t.shadowMapEnabled?"#define "+l:"",t.premultipliedAlpha?"#define PREMULTIPLIED_ALPHA":"",t.numLightProbes>0?"#define USE_LIGHT_PROBES":"",t.numLightProbeGrids>0?"#define USE_LIGHT_PROBES_GRID":"",t.decodeVideoTexture?"#define DECODE_VIDEO_TEXTURE":"",t.decodeVideoTextureEmissive?"#define DECODE_VIDEO_TEXTURE_EMISSIVE":"",t.logarithmicDepthBuffer?"#define USE_LOGARITHMIC_DEPTH_BUFFER":"",t.reversedDepthBuffer?"#define USE_REVERSED_DEPTH_BUFFER":"","uniform mat4 viewMatrix;","uniform vec3 cameraPosition;","uniform bool isOrthographic;",t.toneMapping!==Dn?"#define TONE_MAPPING":"",t.toneMapping!==Dn?Ke.tonemapping_pars_fragment:"",t.toneMapping!==Dn?Fx("toneMapping",t.toneMapping):"",t.dithering?"#define DITHERING":"",t.opaque?"#define OPAQUE":"",Ke.colorspace_pars_fragment,Nx("linearToOutputTexel",t.outputColorSpace),Ox(),t.useDepthPacking?"#define DEPTH_PACKING "+t.depthPacking:"",`
`].filter(ra).join(`
`)),a=Mh(a),a=Jd(a,t),a=jd(a,t),o=Mh(o),o=Jd(o,t),o=jd(o,t),a=Kd(a),o=Kd(o),t.isRawShaderMaterial!==!0&&(w=`#version 300 es
`,m=[f,"#define attribute in","#define varying out","#define texture2D texture"].join(`
`)+`
`+m,p=["#define varying in",t.glslVersion===Kc?"":"layout(location = 0) out highp vec4 pc_fragColor;",t.glslVersion===Kc?"":"#define gl_FragColor pc_fragColor","#define gl_FragDepthEXT gl_FragDepth","#define texture2D texture","#define textureCube texture","#define texture2DProj textureProj","#define texture2DLodEXT textureLod","#define texture2DProjLodEXT textureProjLod","#define textureCubeLodEXT textureLod","#define texture2DGradEXT textureGrad","#define texture2DProjGradEXT textureProjGrad","#define textureCubeGradEXT textureGrad"].join(`
`)+`
`+p);let A=w+m+a,M=w+p+o,E=Yd(s,s.VERTEX_SHADER,A),S=Yd(s,s.FRAGMENT_SHADER,M);s.attachShader(y,E),s.attachShader(y,S),t.index0AttributeName!==void 0?s.bindAttribLocation(y,0,t.index0AttributeName):t.hasPositionAttribute===!0&&s.bindAttribLocation(y,0,"position"),s.linkProgram(y);function C(U){if(i.debug.checkShaderErrors){let O=s.getProgramInfoLog(y)||"",H=s.getShaderInfoLog(E)||"",D=s.getShaderInfoLog(S)||"",B=O.trim(),X=H.trim(),W=D.trim(),ie=!0,q=!0;if(s.getProgramParameter(y,s.LINK_STATUS)===!1)if(ie=!1,typeof i.debug.onShaderError=="function")i.debug.onShaderError(s,y,E,S);else{let K=$d(s,E,"vertex"),te=$d(s,S,"fragment");Ve("WebGLProgram: Shader Error "+s.getError()+" - VALIDATE_STATUS "+s.getProgramParameter(y,s.VALIDATE_STATUS)+`

Material Name: `+U.name+`
Material Type: `+U.type+`

Program Info Log: `+B+`
`+K+`
`+te)}else B!==""?Oe("WebGLProgram: Program Info Log:",B):(X===""||W==="")&&(q=!1);q&&(U.diagnostics={runnable:ie,programLog:B,vertexShader:{log:X,prefix:m},fragmentShader:{log:W,prefix:p}})}s.deleteShader(E),s.deleteShader(S),x=new Gs(s,y),T=kx(s,y)}let x;this.getUniforms=function(){return x===void 0&&C(this),x};let T;this.getAttributes=function(){return T===void 0&&C(this),T};let I=t.rendererExtensionParallelShaderCompile===!1;return this.isReady=function(){return I===!1&&(I=s.getProgramParameter(y,Px)),I},this.destroy=function(){n.releaseStatesOfProgram(this),s.deleteProgram(y),this.program=void 0},this.type=t.shaderType,this.name=t.shaderName,this.id=Ix++,this.cacheKey=e,this.usedTimes=1,this.program=y,this.vertexShader=E,this.fragmentShader=S,this}var ny=0,bh=class{constructor(){this.shaderCache=new Map,this.materialCache=new Map}update(e,t,n){let s=this._getShaderCacheForMaterial(e);return s.has(t)===!1&&(s.add(t),t.usedTimes++),s.has(n)===!1&&(s.add(n),n.usedTimes++),this}remove(e){let t=this.materialCache.get(e);for(let n of t)n.usedTimes--,n.usedTimes===0&&this.shaderCache.delete(n.code);return this.materialCache.delete(e),this}getVertexShaderStage(e){return this._getShaderStage(e.vertexShader)}getFragmentShaderStage(e){return this._getShaderStage(e.fragmentShader)}dispose(){this.shaderCache.clear(),this.materialCache.clear()}_getShaderCacheForMaterial(e){let t=this.materialCache,n=t.get(e);return n===void 0&&(n=new Set,t.set(e,n)),n}_getShaderStage(e){let t=this.shaderCache,n=t.get(e);return n===void 0&&(n=new Sh(e),t.set(e,n)),n}},Sh=class{constructor(e){this.id=ny++,this.code=e,this.usedTimes=0}};function iy(i){return i===Ii||i===Qr||i===ea}function sy(i,e,t,n,s,r){let a=new ws,o=new bh,l=new Set,c=[],h=new Map,d=n.logarithmicDepthBuffer,u=n.precision,f={MeshDepthMaterial:"depth",MeshDistanceMaterial:"distance",MeshNormalMaterial:"normal",MeshBasicMaterial:"basic",MeshLambertMaterial:"lambert",MeshPhongMaterial:"phong",MeshToonMaterial:"toon",MeshStandardMaterial:"physical",MeshPhysicalMaterial:"physical",MeshMatcapMaterial:"matcap",LineBasicMaterial:"basic",LineDashedMaterial:"dashed",PointsMaterial:"points",ShadowMaterial:"shadow",SpriteMaterial:"sprite"};function g(x){return l.add(x),x===0?"uv":`uv${x}`}function y(x,T,I,U,O,H){let D=U.fog,B=O.geometry,X=x.isMeshStandardMaterial||x.isMeshLambertMaterial||x.isMeshPhongMaterial?U.environment:null,W=x.isMeshStandardMaterial||x.isMeshLambertMaterial&&!x.envMap||x.isMeshPhongMaterial&&!x.envMap,ie=e.get(x.envMap||X,W),q=ie&&ie.mapping===Yr?ie.image.height:null,K=f[x.type];x.precision!==null&&(u=n.getMaxPrecision(x.precision),u!==x.precision&&Oe("WebGLProgram.getParameters:",x.precision,"not supported, using",u,"instead."));let te=B.morphAttributes.position||B.morphAttributes.normal||B.morphAttributes.color,Ie=te!==void 0?te.length:0,we=0;B.morphAttributes.position!==void 0&&(we=1),B.morphAttributes.normal!==void 0&&(we=2),B.morphAttributes.color!==void 0&&(we=3);let ot,et,rt,$;if(K){let yt=Yn[K];ot=yt.vertexShader,et=yt.fragmentShader}else{ot=x.vertexShader,et=x.fragmentShader;let yt=o.getVertexShaderStage(x),ct=o.getFragmentShaderStage(x);o.update(x,yt,ct),rt=yt.id,$=ct.id}let Q=i.getRenderTarget(),xe=i.state.buffers.depth.getReversed(),He=O.isInstancedMesh===!0,Se=O.isBatchedMesh===!0,Ge=!!x.map,pt=!!x.matcap,ee=!!ie,se=!!x.aoMap,oe=!!x.lightMap,le=!!x.bumpMap&&x.wireframe===!1,ue=!!x.normalMap,ze=!!x.displacementMap,Ue=!!x.emissiveMap,We=!!x.metalnessMap,qe=!!x.roughnessMap,P=x.anisotropy>0,lt=x.clearcoat>0,tt=x.dispersion>0,b=x.retroreflectivity>0,_=x.iridescence>0,F=x.sheen>0,V=x.transmission>0,Y=P&&!!x.anisotropyMap,ce=lt&&!!x.clearcoatMap,he=lt&&!!x.clearcoatNormalMap,Z=lt&&!!x.clearcoatRoughnessMap,j=_&&!!x.iridescenceMap,de=_&&!!x.iridescenceThicknessMap,Le=F&&!!x.sheenColorMap,ge=F&&!!x.sheenRoughnessMap,fe=!!x.specularMap,De=!!x.specularColorMap,ke=!!x.specularIntensityMap,Ze=V&&!!x.transmissionMap,N=V&&!!x.thicknessMap,pe=!!x.gradientMap,J=!!x.alphaMap,me=x.alphaTest>0,be=!!x.alphaHash,ne=!!x.extensions,Ne=Dn;x.toneMapped&&(Q===null||Q.isXRRenderTarget===!0)&&(Ne=i.toneMapping);let Ce={shaderID:K,shaderType:x.type,shaderName:x.name,vertexShader:ot,fragmentShader:et,defines:x.defines,customVertexShaderID:rt,customFragmentShaderID:$,isRawShaderMaterial:x.isRawShaderMaterial===!0,glslVersion:x.glslVersion,precision:u,batching:Se,batchingColor:Se&&O._colorsTexture!==null,instancing:He,instancingColor:He&&O.instanceColor!==null,instancingMorph:He&&O.morphTexture!==null,outputColorSpace:Q===null?i.outputColorSpace:Q.isXRRenderTarget===!0?Q.texture.colorSpace:it.workingColorSpace,alphaToCoverage:!!x.alphaToCoverage,map:Ge,matcap:pt,envMap:ee,envMapMode:ee&&ie.mapping,envMapCubeUVHeight:q,aoMap:se,lightMap:oe,bumpMap:le,normalMap:ue,displacementMap:ze,emissiveMap:Ue,normalMapObjectSpace:ue&&x.normalMapType===gd,normalMapTangentSpace:ue&&x.normalMapType===ta,packedNormalMap:ue&&x.normalMapType===ta&&iy(x.normalMap.format),metalnessMap:We,roughnessMap:qe,anisotropy:P,anisotropyMap:Y,clearcoat:lt,clearcoatMap:ce,clearcoatNormalMap:he,clearcoatRoughnessMap:Z,dispersion:tt,retroreflection:b,iridescence:_,iridescenceMap:j,iridescenceThicknessMap:de,sheen:F,sheenColorMap:Le,sheenRoughnessMap:ge,specularMap:fe,specularColorMap:De,specularIntensityMap:ke,transmission:V,transmissionMap:Ze,thicknessMap:N,gradientMap:pe,opaque:x.transparent===!1&&x.blending===Bs&&x.alphaToCoverage===!1,alphaMap:J,alphaTest:me,alphaHash:be,combine:x.combine,mapUv:Ge&&g(x.map.channel),aoMapUv:se&&g(x.aoMap.channel),lightMapUv:oe&&g(x.lightMap.channel),bumpMapUv:le&&g(x.bumpMap.channel),normalMapUv:ue&&g(x.normalMap.channel),displacementMapUv:ze&&g(x.displacementMap.channel),emissiveMapUv:Ue&&g(x.emissiveMap.channel),metalnessMapUv:We&&g(x.metalnessMap.channel),roughnessMapUv:qe&&g(x.roughnessMap.channel),anisotropyMapUv:Y&&g(x.anisotropyMap.channel),clearcoatMapUv:ce&&g(x.clearcoatMap.channel),clearcoatNormalMapUv:he&&g(x.clearcoatNormalMap.channel),clearcoatRoughnessMapUv:Z&&g(x.clearcoatRoughnessMap.channel),iridescenceMapUv:j&&g(x.iridescenceMap.channel),iridescenceThicknessMapUv:de&&g(x.iridescenceThicknessMap.channel),sheenColorMapUv:Le&&g(x.sheenColorMap.channel),sheenRoughnessMapUv:ge&&g(x.sheenRoughnessMap.channel),specularMapUv:fe&&g(x.specularMap.channel),specularColorMapUv:De&&g(x.specularColorMap.channel),specularIntensityMapUv:ke&&g(x.specularIntensityMap.channel),transmissionMapUv:Ze&&g(x.transmissionMap.channel),thicknessMapUv:N&&g(x.thicknessMap.channel),alphaMapUv:J&&g(x.alphaMap.channel),vertexTangents:!!B.attributes.tangent&&(ue||P),vertexNormals:!!B.attributes.normal,vertexColors:x.vertexColors,vertexAlphas:x.vertexColors===!0&&!!B.attributes.color&&B.attributes.color.itemSize===4,pointsUvs:O.isPoints===!0&&!!B.attributes.uv&&(Ge||J),fog:!!D,useFog:x.fog===!0,fogExp2:!!D&&D.isFogExp2,flatShading:x.wireframe===!1&&(x.flatShading===!0||B.attributes.normal===void 0&&ue===!1&&(x.isMeshLambertMaterial||x.isMeshPhongMaterial||x.isMeshStandardMaterial||x.isMeshPhysicalMaterial)),sizeAttenuation:x.sizeAttenuation===!0,logarithmicDepthBuffer:d,reversedDepthBuffer:xe,skinning:O.isSkinnedMesh===!0,hasPositionAttribute:B.attributes.position!==void 0,morphTargets:B.morphAttributes.position!==void 0,morphNormals:B.morphAttributes.normal!==void 0,morphColors:B.morphAttributes.color!==void 0,morphTargetsCount:Ie,morphTextureStride:we,numSunLights:T.sun.length,numDirLights:T.directional.length,numPointLights:T.point.length,numSpotLights:T.spot.length,numSpotLightMaps:T.spotLightMap.length,numRectAreaLights:T.rectArea.length,numHemiLights:T.hemi.length,numSunLightShadows:T.sunShadowMap.length,numDirLightShadows:T.directionalShadowMap.length,numPointLightShadows:T.pointShadowMap.length,numSpotLightShadows:T.spotShadowMap.length,numSpotLightShadowsWithMaps:T.numSpotLightShadowsWithMaps,numLightProbes:T.numLightProbes,numLightProbeGrids:H.length,numClippingPlanes:r.numPlanes,numClipIntersection:r.numIntersection,dithering:x.dithering,shadowMapEnabled:i.shadowMap.enabled&&I.length>0,shadowMapType:i.shadowMap.type,toneMapping:Ne,decodeVideoTexture:Ge&&x.map.isVideoTexture===!0&&it.getTransfer(x.map.colorSpace)===ut,decodeVideoTextureEmissive:Ue&&x.emissiveMap.isVideoTexture===!0&&it.getTransfer(x.emissiveMap.colorSpace)===ut,premultipliedAlpha:x.premultipliedAlpha,doubleSided:x.side===Wn,flipSided:x.side===Xt,useDepthPacking:x.depthPacking>=0,depthPacking:x.depthPacking||0,index0AttributeName:x.index0AttributeName,extensionClipCullDistance:ne&&x.extensions.clipCullDistance===!0&&t.has("WEBGL_clip_cull_distance"),extensionMultiDraw:(ne&&x.extensions.multiDraw===!0||Se)&&t.has("WEBGL_multi_draw"),rendererExtensionParallelShaderCompile:t.has("KHR_parallel_shader_compile"),customProgramCacheKey:x.customProgramCacheKey()};return Ce.vertexUv1s=l.has(1),Ce.vertexUv2s=l.has(2),Ce.vertexUv3s=l.has(3),l.clear(),Ce}function m(x){let T=[];if(x.shaderID?T.push(x.shaderID):(T.push(x.customVertexShaderID),T.push(x.customFragmentShaderID)),x.defines!==void 0)for(let I in x.defines)T.push(I),T.push(x.defines[I]);return x.isRawShaderMaterial===!1&&(p(T,x),w(T,x),T.push(i.outputColorSpace)),T.push(x.customProgramCacheKey),T.join()}function p(x,T){x.push(T.precision),x.push(T.outputColorSpace),x.push(T.envMapMode),x.push(T.envMapCubeUVHeight),x.push(T.mapUv),x.push(T.alphaMapUv),x.push(T.lightMapUv),x.push(T.aoMapUv),x.push(T.bumpMapUv),x.push(T.normalMapUv),x.push(T.displacementMapUv),x.push(T.emissiveMapUv),x.push(T.metalnessMapUv),x.push(T.roughnessMapUv),x.push(T.anisotropyMapUv),x.push(T.clearcoatMapUv),x.push(T.clearcoatNormalMapUv),x.push(T.clearcoatRoughnessMapUv),x.push(T.iridescenceMapUv),x.push(T.iridescenceThicknessMapUv),x.push(T.sheenColorMapUv),x.push(T.sheenRoughnessMapUv),x.push(T.specularMapUv),x.push(T.specularColorMapUv),x.push(T.specularIntensityMapUv),x.push(T.transmissionMapUv),x.push(T.thicknessMapUv),x.push(T.combine),x.push(T.fogExp2),x.push(T.sizeAttenuation),x.push(T.morphTargetsCount),x.push(T.morphAttributeCount),x.push(T.numSunLights),x.push(T.numDirLights),x.push(T.numPointLights),x.push(T.numSpotLights),x.push(T.numSpotLightMaps),x.push(T.numHemiLights),x.push(T.numRectAreaLights),x.push(T.numSunLightShadows),x.push(T.numDirLightShadows),x.push(T.numPointLightShadows),x.push(T.numSpotLightShadows),x.push(T.numSpotLightShadowsWithMaps),x.push(T.numLightProbes),x.push(T.shadowMapType),x.push(T.toneMapping),x.push(T.numClippingPlanes),x.push(T.numClipIntersection),x.push(T.depthPacking)}function w(x,T){a.disableAll(),T.instancing&&a.enable(0),T.instancingColor&&a.enable(1),T.instancingMorph&&a.enable(2),T.matcap&&a.enable(3),T.envMap&&a.enable(4),T.normalMapObjectSpace&&a.enable(5),T.normalMapTangentSpace&&a.enable(6),T.clearcoat&&a.enable(7),T.iridescence&&a.enable(8),T.alphaTest&&a.enable(9),T.vertexColors&&a.enable(10),T.vertexAlphas&&a.enable(11),T.vertexUv1s&&a.enable(12),T.vertexUv2s&&a.enable(13),T.vertexUv3s&&a.enable(14),T.vertexTangents&&a.enable(15),T.anisotropy&&a.enable(16),T.alphaHash&&a.enable(17),T.batching&&a.enable(18),T.dispersion&&a.enable(19),T.retroreflection&&a.enable(24),T.batchingColor&&a.enable(20),T.gradientMap&&a.enable(21),T.packedNormalMap&&a.enable(22),T.vertexNormals&&a.enable(23),x.push(a.mask),a.disableAll(),T.fog&&a.enable(0),T.useFog&&a.enable(1),T.flatShading&&a.enable(2),T.logarithmicDepthBuffer&&a.enable(3),T.reversedDepthBuffer&&a.enable(4),T.skinning&&a.enable(5),T.morphTargets&&a.enable(6),T.morphNormals&&a.enable(7),T.morphColors&&a.enable(8),T.premultipliedAlpha&&a.enable(9),T.shadowMapEnabled&&a.enable(10),T.doubleSided&&a.enable(11),T.flipSided&&a.enable(12),T.useDepthPacking&&a.enable(13),T.dithering&&a.enable(14),T.transmission&&a.enable(15),T.sheen&&a.enable(16),T.opaque&&a.enable(17),T.pointsUvs&&a.enable(18),T.decodeVideoTexture&&a.enable(19),T.decodeVideoTextureEmissive&&a.enable(20),T.alphaToCoverage&&a.enable(21),T.numLightProbeGrids>0&&a.enable(22),T.hasPositionAttribute&&a.enable(23),x.push(a.mask)}function A(x){let T=f[x.type],I;if(T){let U=Yn[T];I=Nd.clone(U.uniforms)}else I=x.uniforms;return I}function M(x,T){let I=h.get(T);return I!==void 0?++I.usedTimes:(I=new ty(i,T,x,s),c.push(I),h.set(T,I)),I}function E(x){if(--x.usedTimes===0){let T=c.indexOf(x);c[T]=c[c.length-1],c.pop(),h.delete(x.cacheKey),x.destroy()}}function S(x){o.remove(x)}function C(){o.dispose()}return{getParameters:y,getProgramCacheKey:m,getUniforms:A,acquireProgram:M,releaseProgram:E,releaseShaderCache:S,programs:c,dispose:C}}function ry(){let i=new WeakMap;function e(a){return i.has(a)}function t(a){let o=i.get(a);return o===void 0&&(o={},i.set(a,o)),o}function n(a){i.delete(a)}function s(a,o,l){i.get(a)[o]=l}function r(){i=new WeakMap}return{has:e,get:t,remove:n,update:s,dispose:r}}function ay(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.material.id!==e.material.id?i.material.id-e.material.id:i.materialVariant!==e.materialVariant?i.materialVariant-e.materialVariant:i.z!==e.z?i.z-e.z:i.id-e.id}function ef(i,e){return i.groupOrder!==e.groupOrder?i.groupOrder-e.groupOrder:i.renderOrder!==e.renderOrder?i.renderOrder-e.renderOrder:i.z!==e.z?e.z-i.z:i.id-e.id}function tf(){let i=[],e=0,t=[],n=[],s=[];function r(){e=0,t.length=0,n.length=0,s.length=0}function a(u){let f=0;return u.isInstancedMesh&&(f+=2),u.isSkinnedMesh&&(f+=1),f}function o(u,f,g,y,m,p){let w=i[e];return w===void 0?(w={id:u.id,object:u,geometry:f,material:g,materialVariant:a(u),groupOrder:y,renderOrder:u.renderOrder,z:m,group:p},i[e]=w):(w.id=u.id,w.object=u,w.geometry=f,w.material=g,w.materialVariant=a(u),w.groupOrder=y,w.renderOrder=u.renderOrder,w.z=m,w.group=p),e++,w}function l(u,f,g,y,m,p,w){w.reversedDepth===!0&&(m=-m);let A=o(u,f,g,y,m,p);g.transmission>0?n.push(A):g.transparent===!0?s.push(A):t.push(A)}function c(u,f,g,y,m,p){let w=o(u,f,g,y,m,p);g.transmission>0?n.unshift(w):g.transparent===!0?s.unshift(w):t.unshift(w)}function h(u,f){t.length>1&&t.sort(u||ay),n.length>1&&n.sort(f||ef),s.length>1&&s.sort(f||ef)}function d(){for(let u=e,f=i.length;u<f;u++){let g=i[u];if(g.id===null)break;g.id=null,g.object=null,g.geometry=null,g.material=null,g.group=null}}return{opaque:t,transmissive:n,transparent:s,init:r,push:l,unshift:c,finish:d,sort:h}}function oy(){let i=new WeakMap;function e(n,s){let r=i.get(n),a;return r===void 0?(a=new tf,i.set(n,[a])):s>=r.length?(a=new tf,r.push(a)):a=r[s],a}function t(){i=new WeakMap}return{get:e,dispose:t}}function ly(){let i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"SunLight":case"DirectionalLight":t={direction:new R,color:new Ye};break;case"SpotLight":t={position:new R,direction:new R,color:new Ye,distance:0,coneCos:0,penumbraCos:0,decay:0};break;case"PointLight":t={position:new R,color:new Ye,distance:0,decay:0};break;case"HemisphereLight":t={direction:new R,skyColor:new Ye,groundColor:new Ye};break;case"RectAreaLight":t={color:new Ye,position:new R,halfWidth:new R,halfHeight:new R};break}return i[e.id]=t,t}}}function cy(){let i={};return{get:function(e){if(i[e.id]!==void 0)return i[e.id];let t;switch(e.type){case"SunLight":case"DirectionalLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new ae};break;case"SpotLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new ae};break;case"PointLight":t={shadowIntensity:1,shadowBias:0,shadowNormalBias:0,shadowRadius:1,shadowMapSize:new ae,shadowCameraNear:1,shadowCameraFar:1e3};break}return i[e.id]=t,t}}}var hy=0;function uy(i,e){return(e.castShadow?2:0)-(i.castShadow?2:0)+(e.map?1:0)-(i.map?1:0)}function dy(i){let e=new ly,t=cy(),n={version:0,hash:{sunLength:-1,directionalLength:-1,pointLength:-1,spotLength:-1,rectAreaLength:-1,hemiLength:-1,numSunShadows:-1,numDirectionalShadows:-1,numPointShadows:-1,numSpotShadows:-1,numSpotMaps:-1,numLightProbes:-1},ambient:[0,0,0],probe:[],sun:[],sunShadow:[],sunShadowMap:[],sunShadowMatrix:[],sunShadowCascade:[],directional:[],directionalShadow:[],directionalShadowMap:[],directionalShadowMatrix:[],spot:[],spotLightMap:[],spotShadow:[],spotShadowMap:[],spotLightMatrix:[],rectArea:[],rectAreaLTC1:null,rectAreaLTC2:null,point:[],pointShadow:[],pointShadowMap:[],pointShadowMatrix:[],hemi:[],numSpotLightShadowsWithMaps:0,numLightProbes:0};for(let c=0;c<9;c++)n.probe.push(new R);let s=new R,r=new dt,a=new dt;function o(c){let h=0,d=0,u=0;for(let O=0;O<9;O++)n.probe[O].set(0,0,0);let f=0,g=0,y=0,m=0,p=0,w=0,A=0,M=0,E=0,S=0,C=0,x=0,T=0,I=0;c.sort(uy);for(let O=0,H=c.length;O<H;O++){let D=c[O],B=D.color,X=D.intensity,W=D.distance,ie=null;if(D.shadow&&D.shadow.map&&(D.shadow.map.texture.format===Ii?ie=D.shadow.map.texture:ie=D.shadow.map.depthTexture||D.shadow.map.texture),D.isAmbientLight)h+=B.r*X,d+=B.g*X,u+=B.b*X;else if(D.isLightProbe){for(let q=0;q<9;q++)n.probe[q].addScaledVector(D.sh.coefficients[q],X);I++}else if(D.isSunLight){let q=e.get(D);if(q.color.copy(D.color).multiplyScalar(D.intensity),D.castShadow){let K=D.shadow,te=t.get(D);te.shadowIntensity=K.intensity,te.shadowBias=K.bias,te.shadowNormalBias=K.normalBias,te.shadowRadius=K.radius,te.shadowMapSize.copy(K.mapSize).multiply(K.getFrameExtents()),n.sunShadow[g]=te,n.sunShadowMap[g]=ie;let Ie=K.getViewportCount();for(let we=0;we<Ie;we++)n.sunShadowMatrix[y+we]=K.getMatrix(we),n.sunShadowCascade[y+we]=K._cascadeData[we];y+=Ie,g++}n.sun[f]=q,f++}else if(D.isDirectionalLight){let q=e.get(D);if(q.color.copy(D.color).multiplyScalar(D.intensity),D.castShadow){let K=D.shadow,te=t.get(D);te.shadowIntensity=K.intensity,te.shadowBias=K.bias,te.shadowNormalBias=K.normalBias,te.shadowRadius=K.radius,te.shadowMapSize=K.mapSize,n.directionalShadow[m]=te,n.directionalShadowMap[m]=ie,n.directionalShadowMatrix[m]=D.shadow.matrix,E++}n.directional[m]=q,m++}else if(D.isSpotLight){let q=e.get(D);q.position.setFromMatrixPosition(D.matrixWorld),q.color.copy(B).multiplyScalar(X),q.distance=W,q.coneCos=Math.cos(D.angle),q.penumbraCos=Math.cos(D.angle*(1-D.penumbra)),q.decay=D.decay,n.spot[w]=q;let K=D.shadow;if(D.map&&(n.spotLightMap[x]=D.map,x++,K.updateMatrices(D),D.castShadow&&T++),n.spotLightMatrix[w]=K.matrix,D.castShadow){let te=t.get(D);te.shadowIntensity=K.intensity,te.shadowBias=K.bias,te.shadowNormalBias=K.normalBias,te.shadowRadius=K.radius,te.shadowMapSize=K.mapSize,n.spotShadow[w]=te,n.spotShadowMap[w]=ie,C++}w++}else if(D.isRectAreaLight){let q=e.get(D);q.color.copy(B).multiplyScalar(X),q.halfWidth.set(D.width*.5,0,0),q.halfHeight.set(0,D.height*.5,0),n.rectArea[A]=q,A++}else if(D.isPointLight){let q=e.get(D);if(q.color.copy(D.color).multiplyScalar(D.intensity),q.distance=D.distance,q.decay=D.decay,D.castShadow){let K=D.shadow,te=t.get(D);te.shadowIntensity=K.intensity,te.shadowBias=K.bias,te.shadowNormalBias=K.normalBias,te.shadowRadius=K.radius,te.shadowMapSize=K.mapSize,te.shadowCameraNear=K.camera.near,te.shadowCameraFar=K.camera.far,n.pointShadow[p]=te,n.pointShadowMap[p]=ie,n.pointShadowMatrix[p]=D.shadow.matrix,S++}n.point[p]=q,p++}else if(D.isHemisphereLight){let q=e.get(D);q.skyColor.copy(D.color).multiplyScalar(X),q.groundColor.copy(D.groundColor).multiplyScalar(X),n.hemi[M]=q,M++}}A>0&&(i.has("OES_texture_float_linear")===!0?(n.rectAreaLTC1=_e.LTC_FLOAT_1,n.rectAreaLTC2=_e.LTC_FLOAT_2):(n.rectAreaLTC1=_e.LTC_HALF_1,n.rectAreaLTC2=_e.LTC_HALF_2)),n.ambient[0]=h,n.ambient[1]=d,n.ambient[2]=u;let U=n.hash;(U.sunLength!==f||U.directionalLength!==m||U.pointLength!==p||U.spotLength!==w||U.rectAreaLength!==A||U.hemiLength!==M||U.numSunShadows!==g||U.numDirectionalShadows!==E||U.numPointShadows!==S||U.numSpotShadows!==C||U.numSpotMaps!==x||U.numLightProbes!==I)&&(n.sun.length=f,n.directional.length=m,n.spot.length=w,n.rectArea.length=A,n.point.length=p,n.hemi.length=M,n.sunShadow.length=g,n.sunShadowMap.length=g,n.sunShadowMatrix.length=y,n.sunShadowCascade.length=y,n.directionalShadow.length=E,n.directionalShadowMap.length=E,n.directionalShadowMatrix.length=E,n.pointShadow.length=S,n.pointShadowMap.length=S,n.pointShadowMatrix.length=S,n.spotShadow.length=C,n.spotShadowMap.length=C,n.spotLightMatrix.length=C+x-T,n.spotLightMap.length=x,n.numSpotLightShadowsWithMaps=T,n.numLightProbes=I,U.sunLength=f,U.directionalLength=m,U.pointLength=p,U.spotLength=w,U.rectAreaLength=A,U.hemiLength=M,U.numSunShadows=g,U.numDirectionalShadows=E,U.numPointShadows=S,U.numSpotShadows=C,U.numSpotMaps=x,U.numLightProbes=I,n.version=hy++)}function l(c,h){let d=0,u=0,f=0,g=0,y=0,m=0,p=h.matrixWorldInverse;for(let w=0,A=c.length;w<A;w++){let M=c[w];if(M.isSunLight){let E=n.sun[d];E.direction.setFromMatrixPosition(M.matrixWorld),E.direction.transformDirection(p),d++}else if(M.isDirectionalLight){let E=n.directional[u];E.direction.setFromMatrixPosition(M.matrixWorld),s.setFromMatrixPosition(M.target.matrixWorld),E.direction.sub(s),E.direction.transformDirection(p),u++}else if(M.isSpotLight){let E=n.spot[g];E.position.setFromMatrixPosition(M.matrixWorld),E.position.applyMatrix4(p),E.direction.setFromMatrixPosition(M.matrixWorld),s.setFromMatrixPosition(M.target.matrixWorld),E.direction.sub(s),E.direction.transformDirection(p),g++}else if(M.isRectAreaLight){let E=n.rectArea[y];E.position.setFromMatrixPosition(M.matrixWorld),E.position.applyMatrix4(p),a.identity(),r.copy(M.matrixWorld),r.premultiply(p),a.extractRotation(r),E.halfWidth.set(M.width*.5,0,0),E.halfHeight.set(0,M.height*.5,0),E.halfWidth.applyMatrix4(a),E.halfHeight.applyMatrix4(a),y++}else if(M.isPointLight){let E=n.point[f];E.position.setFromMatrixPosition(M.matrixWorld),E.position.applyMatrix4(p),f++}else if(M.isHemisphereLight){let E=n.hemi[m];E.direction.setFromMatrixPosition(M.matrixWorld),E.direction.transformDirection(p),m++}}}return{setup:o,setupView:l,state:n}}function nf(i){let e=new dy(i),t=[],n=[],s=[];function r(u){d.camera=u,t.length=0,n.length=0,s.length=0}function a(u){t.push(u)}function o(u){n.push(u)}function l(u){s.push(u)}function c(){e.setup(t)}function h(u){e.setupView(t,u)}let d={lightsArray:t,shadowsArray:n,lightProbeGridArray:s,camera:null,lights:e,transmissionRenderTarget:{},textureUnits:0};return{init:r,state:d,setupLights:c,setupLightsView:h,pushLight:a,pushShadow:o,pushLightProbeGrid:l}}function fy(i){let e=new WeakMap;function t(s,r=0){let a=e.get(s),o;return a===void 0?(o=new nf(i),e.set(s,[o])):r>=a.length?(o=new nf(i),a.push(o)):o=a[r],o}function n(){e=new WeakMap}return{get:t,dispose:n}}var py=`void main() {
	gl_Position = vec4( position, 1.0 );
}`,my=`uniform sampler2D shadow_pass;
uniform vec2 resolution;
uniform float radius;
void main() {
	const float samples = float( VSM_SAMPLES );
	float mean = 0.0;
	float squared_mean = 0.0;
	float uvStride = samples <= 1.0 ? 0.0 : 2.0 / ( samples - 1.0 );
	float uvStart = samples <= 1.0 ? 0.0 : - 1.0;
	for ( float i = 0.0; i < samples; i ++ ) {
		float uvOffset = uvStart + i * uvStride;
		#ifdef HORIZONTAL_PASS
			vec2 distribution = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( uvOffset, 0.0 ) * radius ) / resolution ).rg;
			mean += distribution.x;
			squared_mean += distribution.y * distribution.y + distribution.x * distribution.x;
		#else
			float depth = texture2D( shadow_pass, ( gl_FragCoord.xy + vec2( 0.0, uvOffset ) * radius ) / resolution ).r;
			mean += depth;
			squared_mean += depth * depth;
		#endif
	}
	mean = mean / samples;
	squared_mean = squared_mean / samples;
	float std_dev = sqrt( max( 0.0, squared_mean - mean * mean ) );
	gl_FragColor = vec4( mean, std_dev, 0.0, 1.0 );
}`,gy=[new R(1,0,0),new R(-1,0,0),new R(0,1,0),new R(0,-1,0),new R(0,0,1),new R(0,0,-1)],_y=[new R(0,-1,0),new R(0,-1,0),new R(0,0,1),new R(0,0,-1),new R(0,-1,0),new R(0,-1,0)],sf=new dt,sa=new R,gh=new R;function xy(i,e,t){let n=new Rs,s=new ae,r=new ae,a=new St,o=new xo,l=new yo,c={},h=t.maxTextureSize,d={[Ai]:Xt,[Xt]:Ai,[Wn]:Wn},u=new tn({defines:{VSM_SAMPLES:8},uniforms:{shadow_pass:{value:null},resolution:{value:new ae},radius:{value:4}},vertexShader:py,fragmentShader:my}),f=u.clone();f.defines.HORIZONTAL_PASS=1;let g=new Ut;g.setAttribute("position",new It(new Float32Array([-1,-1,.5,3,-1,.5,-1,3,.5]),3));let y=new mt(g,u),m=this;this.enabled=!1,this.autoUpdate=!0,this.needsUpdate=!1,this.type=Xr;let p=this.type;this.render=function(S,C,x){if(m.enabled===!1||m.autoUpdate===!1&&m.needsUpdate===!1||S.length===0)return;this.type===No&&(Oe("WebGLShadowMap: PCFSoftShadowMap has been removed. Using PCFShadowMap instead."),this.type=Xr);let T=i.getRenderTarget(),I=i.getActiveCubeFace(),U=i.getActiveMipmapLevel(),O=i.state;O.setBlending(Xn),O.buffers.depth.getReversed()===!0?O.buffers.color.setClear(0,0,0,0):O.buffers.color.setClear(1,1,1,1),O.buffers.depth.setTest(!0),O.setScissorTest(!1);let H=p!==this.type;H&&C.traverse(function(D){D.material&&(Array.isArray(D.material)?D.material.forEach(B=>B.needsUpdate=!0):D.material.needsUpdate=!0)});for(let D=0,B=S.length;D<B;D++){let X=S[D],W=X.shadow;if(W===void 0){Oe("WebGLShadowMap:",X,"has no shadow.");continue}if(W.autoUpdate===!1&&W.needsUpdate===!1)continue;s.copy(W.mapSize);let ie=W.getFrameExtents();s.multiply(ie),r.copy(W.mapSize),(s.x>h||s.y>h)&&(s.x>h&&(r.x=Math.floor(h/ie.x),s.x=r.x*ie.x,W.mapSize.x=r.x),s.y>h&&(r.y=Math.floor(h/ie.y),s.y=r.y*ie.y,W.mapSize.y=r.y));let q=i.state.buffers.depth.getReversed();if(W.camera._reversedDepth=q,W.map===null||H===!0){if(W.map!==null&&(W.map.depthTexture!==null&&(W.map.depthTexture.dispose(),W.map.depthTexture=null),W.map.dispose()),this.type===Os){if(X.isPointLight){Oe("WebGLShadowMap: VSM shadow maps are not supported for PointLights. Use PCF or BasicShadowMap instead.");continue}W.map=new sn(s.x,s.y,{format:Ii,type:Un,minFilter:Gt,magFilter:Gt,generateMipmaps:!1}),W.map.texture.name=X.name+".shadowMap",W.map.depthTexture=new vi(s.x,s.y,vn),W.map.depthTexture.name=X.name+".shadowMapDepth",W.map.depthTexture.format=Vn,W.map.depthTexture.compareFunction=null,W.map.depthTexture.minFilter=kt,W.map.depthTexture.magFilter=kt}else X.isPointLight?(W.map=new wl(s.x),W.map.depthTexture=new co(s.x,Nn)):(W.map=new sn(s.x,s.y),W.map.depthTexture=new vi(s.x,s.y,Nn)),W.map.depthTexture.name=X.name+".shadowMap",W.map.depthTexture.format=Vn,this.type===Xr?(W.map.depthTexture.compareFunction=q?Sl:bl,W.map.depthTexture.minFilter=Gt,W.map.depthTexture.magFilter=Gt):(W.map.depthTexture.compareFunction=null,W.map.depthTexture.minFilter=kt,W.map.depthTexture.magFilter=kt);W.camera.updateProjectionMatrix()}W.map.isWebGLCubeRenderTarget!==!0&&(W.map.width!==s.x||W.map.height!==s.y)&&W.map.setSize(s.x,s.y);let K=W.map.isWebGLCubeRenderTarget?6:W.getViewportCount();X.isPointLight!==!0&&W.updateMatrices(X,x);for(let te=0;te<K;te++){let Ie=W.getCamera(te);if(X.isPointLight){let we=W.camera,ot=W.matrix,et=X.distance||we.far;et!==we.far&&(we.far=et,we.updateProjectionMatrix()),sa.setFromMatrixPosition(X.matrixWorld),we.position.copy(sa),gh.copy(we.position),gh.add(gy[te]),we.up.copy(_y[te]),we.lookAt(gh),we.updateMatrixWorld(),ot.makeTranslation(-sa.x,-sa.y,-sa.z),sf.multiplyMatrices(we.projectionMatrix,we.matrixWorldInverse),W._frustum.setFromProjectionMatrix(sf,we.coordinateSystem,we.reversedDepth)}if(W.map.isWebGLCubeRenderTarget)i.setRenderTarget(W.map,te),i.clear();else{te===0&&(i.setRenderTarget(W.map),i.clear());let we=W.getViewport(te);a.set(r.x*we.x,r.y*we.y,r.x*we.z,r.y*we.w),O.viewport(a)}n=W.getFrustum(te),M(C,x,Ie,X,this.type)}W.isPointLightShadow!==!0&&this.type===Os&&w(W,x),W.needsUpdate=!1}p=this.type,m.needsUpdate=!1,i.setRenderTarget(T,I,U)};function w(S,C){let x=e.update(y);u.defines.VSM_SAMPLES!==S.blurSamples&&(u.defines.VSM_SAMPLES=S.blurSamples,f.defines.VSM_SAMPLES=S.blurSamples,u.needsUpdate=!0,f.needsUpdate=!0),S.mapPass===null?S.mapPass=new sn(s.x,s.y,{format:Ii,type:Un}):(S.mapPass.width!==S.map.width||S.mapPass.height!==S.map.height)&&S.mapPass.setSize(S.map.width,S.map.height),u.uniforms.shadow_pass.value=S.map.depthTexture,u.uniforms.resolution.value.set(S.map.width,S.map.height),u.uniforms.radius.value=S.radius,i.setRenderTarget(S.mapPass),i.clear(),i.renderBufferDirect(C,null,x,u,y,null),f.uniforms.shadow_pass.value=S.mapPass.texture,f.uniforms.resolution.value.set(S.map.width,S.map.height),f.uniforms.radius.value=S.radius,i.setRenderTarget(S.map),i.clear(),i.renderBufferDirect(C,null,x,f,y,null)}function A(S,C,x,T){let I=null,U=x.isPointLight===!0?S.customDistanceMaterial:S.customDepthMaterial;if(U!==void 0)I=U;else if(I=x.isPointLight===!0?l:o,i.localClippingEnabled&&C.clipShadows===!0&&Array.isArray(C.clippingPlanes)&&C.clippingPlanes.length!==0||C.displacementMap&&C.displacementScale!==0||C.alphaMap&&C.alphaTest>0||C.map&&C.alphaTest>0||C.alphaToCoverage===!0){let O=I.uuid,H=C.uuid,D=c[O];D===void 0&&(D={},c[O]=D);let B=D[H];B===void 0&&(B=I.clone(),D[H]=B,C.addEventListener("dispose",E)),I=B}if(I.visible=C.visible,I.wireframe=C.wireframe,T===Os?I.side=C.shadowSide!==null?C.shadowSide:C.side:I.side=C.shadowSide!==null?C.shadowSide:d[C.side],I.alphaMap=C.alphaMap,I.alphaTest=C.alphaToCoverage===!0?.5:C.alphaTest,I.map=C.map,I.clipShadows=C.clipShadows,I.clippingPlanes=C.clippingPlanes,I.clipIntersection=C.clipIntersection,I.displacementMap=C.displacementMap,I.displacementScale=C.displacementScale,I.displacementBias=C.displacementBias,I.wireframeLinewidth=C.wireframeLinewidth,I.linewidth=C.linewidth,x.isPointLight===!0&&I.isMeshDistanceMaterial===!0){let O=i.properties.get(I);O.light=x}return I}function M(S,C,x,T,I){if(S.visible===!1)return;if(S.layers.test(C.layers)&&(S.isMesh||S.isLine||S.isPoints)&&(S.castShadow||S.receiveShadow&&I===Os)&&(!S.frustumCulled||S.intersectsFrustum(n))){S.modelViewMatrix.multiplyMatrices(x.matrixWorldInverse,S.matrixWorld);let H=e.update(S),D=S.material;if(Array.isArray(D)){let B=H.groups;for(let X=0,W=B.length;X<W;X++){let ie=B[X],q=D[ie.materialIndex];if(q&&q.visible){let K=A(S,q,T,I);S.onBeforeShadow(i,S,C,x,H,K,ie),i.renderBufferDirect(x,null,H,K,S,ie),S.onAfterShadow(i,S,C,x,H,K,ie)}}}else if(D.visible){let B=A(S,D,T,I);S.onBeforeShadow(i,S,C,x,H,B,null),i.renderBufferDirect(x,null,H,B,S,null),S.onAfterShadow(i,S,C,x,H,B,null)}}let O=S.children;for(let H=0,D=O.length;H<D;H++)M(O[H],C,x,T,I)}function E(S){S.target.removeEventListener("dispose",E);for(let x in c){let T=c[x],I=S.target.uuid;I in T&&(T[I].dispose(),delete T[I])}}}function yy(i,e){function t(){let N=!1,pe=new St,J=null,me=new St(0,0,0,0);return{setMask:function(be){J!==be&&!N&&(i.colorMask(be,be,be,be),J=be)},setLocked:function(be){N=be},setClear:function(be,ne,Ne,Ce,yt){yt===!0&&(be*=Ce,ne*=Ce,Ne*=Ce),pe.set(be,ne,Ne,Ce),me.equals(pe)===!1&&(i.clearColor(be,ne,Ne,Ce),me.copy(pe))},reset:function(){N=!1,J=null,me.set(-1,0,0,0)}}}function n(){let N=!1,pe=!1,J=null,me=null,be=null;return{setReversed:function(ne){if(pe!==ne){let Ne=e.get("EXT_clip_control");ne?Ne.clipControlEXT(Ne.LOWER_LEFT_EXT,Ne.ZERO_TO_ONE_EXT):Ne.clipControlEXT(Ne.LOWER_LEFT_EXT,Ne.NEGATIVE_ONE_TO_ONE_EXT),pe=ne;let Ce=be;be=null,this.setClear(Ce)}},getReversed:function(){return pe},setTest:function(ne){ne?Q(i.DEPTH_TEST):xe(i.DEPTH_TEST)},setMask:function(ne){J!==ne&&!N&&(i.depthMask(ne),J=ne)},setFunc:function(ne){if(pe&&(ne=Rd[ne]),me!==ne){switch(ne){case Ya:i.depthFunc(i.NEVER);break;case Za:i.depthFunc(i.ALWAYS);break;case $a:i.depthFunc(i.LESS);break;case vs:i.depthFunc(i.LEQUAL);break;case Ja:i.depthFunc(i.EQUAL);break;case ja:i.depthFunc(i.GEQUAL);break;case Ka:i.depthFunc(i.GREATER);break;case Qa:i.depthFunc(i.NOTEQUAL);break;default:i.depthFunc(i.LEQUAL)}me=ne}},setLocked:function(ne){N=ne},setClear:function(ne){be!==ne&&(be=ne,pe&&(ne=1-ne),i.clearDepth(ne))},reset:function(){N=!1,J=null,me=null,be=null,pe=!1}}}function s(){let N=!1,pe=null,J=null,me=null,be=null,ne=null,Ne=null,Ce=null,yt=null;return{setTest:function(ct){N||(ct?Q(i.STENCIL_TEST):xe(i.STENCIL_TEST))},setMask:function(ct){pe!==ct&&!N&&(i.stencilMask(ct),pe=ct)},setFunc:function(ct,wn,On){(J!==ct||me!==wn||be!==On)&&(i.stencilFunc(ct,wn,On),J=ct,me=wn,be=On)},setOp:function(ct,wn,On){(ne!==ct||Ne!==wn||Ce!==On)&&(i.stencilOp(ct,wn,On),ne=ct,Ne=wn,Ce=On)},setLocked:function(ct){N=ct},setClear:function(ct){yt!==ct&&(i.clearStencil(ct),yt=ct)},reset:function(){N=!1,pe=null,J=null,me=null,be=null,ne=null,Ne=null,Ce=null,yt=null}}}let r=new t,a=new n,o=new s,l=new WeakMap,c=new WeakMap,h={},d={},u={},f=new WeakMap,g=[],y=null,m=!1,p=null,w=null,A=null,M=null,E=null,S=null,C=null,x=new Ye(0,0,0),T=0,I=!1,U=null,O=null,H=null,D=null,B=null,X=i.getParameter(i.MAX_COMBINED_TEXTURE_IMAGE_UNITS),W=!1,ie=0,q=i.getParameter(i.VERSION);q.indexOf("WebGL")!==-1?(ie=parseFloat(/^WebGL (\d)/.exec(q)[1]),W=ie>=1):q.indexOf("OpenGL ES")!==-1&&(ie=parseFloat(/^OpenGL ES (\d)/.exec(q)[1]),W=ie>=2);let K=null,te={},Ie=i.getParameter(i.SCISSOR_BOX),we=i.getParameter(i.VIEWPORT),ot=new St().fromArray(Ie),et=new St().fromArray(we);function rt(N,pe,J,me){let be=new Uint8Array(4),ne=i.createTexture();i.bindTexture(N,ne),i.texParameteri(N,i.TEXTURE_MIN_FILTER,i.NEAREST),i.texParameteri(N,i.TEXTURE_MAG_FILTER,i.NEAREST);for(let Ne=0;Ne<J;Ne++)N===i.TEXTURE_3D||N===i.TEXTURE_2D_ARRAY?i.texImage3D(pe,0,i.RGBA,1,1,me,0,i.RGBA,i.UNSIGNED_BYTE,be):i.texImage2D(pe+Ne,0,i.RGBA,1,1,0,i.RGBA,i.UNSIGNED_BYTE,be);return ne}let $={};$[i.TEXTURE_2D]=rt(i.TEXTURE_2D,i.TEXTURE_2D,1),$[i.TEXTURE_CUBE_MAP]=rt(i.TEXTURE_CUBE_MAP,i.TEXTURE_CUBE_MAP_POSITIVE_X,6),$[i.TEXTURE_2D_ARRAY]=rt(i.TEXTURE_2D_ARRAY,i.TEXTURE_2D_ARRAY,1,1),$[i.TEXTURE_3D]=rt(i.TEXTURE_3D,i.TEXTURE_3D,1,1),r.setClear(0,0,0,1),a.setClear(1),o.setClear(0),Q(i.DEPTH_TEST),a.setFunc(vs),le(!1),ue(Dc),Q(i.CULL_FACE),se(Xn);function Q(N){h[N]!==!0&&(i.enable(N),h[N]=!0)}function xe(N){h[N]!==!1&&(i.disable(N),h[N]=!1)}function He(N,pe){return u[N]!==pe?(i.bindFramebuffer(N,pe),u[N]=pe,N===i.DRAW_FRAMEBUFFER&&(u[i.FRAMEBUFFER]=pe),N===i.FRAMEBUFFER&&(u[i.DRAW_FRAMEBUFFER]=pe),!0):!1}function Se(N,pe){let J=g,me=!1;if(N){J=f.get(pe),J===void 0&&(J=[],f.set(pe,J));let be=N.textures;if(J.length!==be.length||J[0]!==i.COLOR_ATTACHMENT0){for(let ne=0,Ne=be.length;ne<Ne;ne++)J[ne]=i.COLOR_ATTACHMENT0+ne;J.length=be.length,me=!0}}else J[0]!==i.BACK&&(J[0]=i.BACK,me=!0);me&&i.drawBuffers(J)}function Ge(N){return y!==N?(i.useProgram(N),y=N,!0):!1}let pt={[Xi]:i.FUNC_ADD,[$u]:i.FUNC_SUBTRACT,[Ju]:i.FUNC_REVERSE_SUBTRACT};pt[ju]=i.MIN,pt[Ku]=i.MAX;let ee={[Qu]:i.ZERO,[ed]:i.ONE,[td]:i.SRC_COLOR,[Oc]:i.SRC_ALPHA,[od]:i.SRC_ALPHA_SATURATE,[rd]:i.DST_COLOR,[id]:i.DST_ALPHA,[nd]:i.ONE_MINUS_SRC_COLOR,[Bc]:i.ONE_MINUS_SRC_ALPHA,[ad]:i.ONE_MINUS_DST_COLOR,[sd]:i.ONE_MINUS_DST_ALPHA,[ld]:i.CONSTANT_COLOR,[cd]:i.ONE_MINUS_CONSTANT_COLOR,[hd]:i.CONSTANT_ALPHA,[ud]:i.ONE_MINUS_CONSTANT_ALPHA};function se(N,pe,J,me,be,ne,Ne,Ce,yt,ct){if(N===Xn){m===!0&&(xe(i.BLEND),m=!1);return}if(m===!1&&(Q(i.BLEND),m=!0),N!==Zu){if(N!==p||ct!==I){if((w!==Xi||E!==Xi)&&(i.blendEquation(i.FUNC_ADD),w=Xi,E=Xi),ct)switch(N){case Bs:i.blendFuncSeparate(i.ONE,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Nc:i.blendFunc(i.ONE,i.ONE);break;case Uc:i.blendFuncSeparate(i.ZERO,i.ONE_MINUS_SRC_COLOR,i.ZERO,i.ONE);break;case Fc:i.blendFuncSeparate(i.DST_COLOR,i.ONE_MINUS_SRC_ALPHA,i.ZERO,i.ONE);break;default:Ve("WebGLState: Invalid blending: ",N);break}else switch(N){case Bs:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE_MINUS_SRC_ALPHA,i.ONE,i.ONE_MINUS_SRC_ALPHA);break;case Nc:i.blendFuncSeparate(i.SRC_ALPHA,i.ONE,i.ONE,i.ONE);break;case Uc:Ve("WebGLState: SubtractiveBlending requires material.premultipliedAlpha = true");break;case Fc:Ve("WebGLState: MultiplyBlending requires material.premultipliedAlpha = true");break;default:Ve("WebGLState: Invalid blending: ",N);break}A=null,M=null,S=null,C=null,x.set(0,0,0),T=0,p=N,I=ct}return}be=be||pe,ne=ne||J,Ne=Ne||me,(pe!==w||be!==E)&&(i.blendEquationSeparate(pt[pe],pt[be]),w=pe,E=be),(J!==A||me!==M||ne!==S||Ne!==C)&&(i.blendFuncSeparate(ee[J],ee[me],ee[ne],ee[Ne]),A=J,M=me,S=ne,C=Ne),(Ce.equals(x)===!1||yt!==T)&&(i.blendColor(Ce.r,Ce.g,Ce.b,yt),x.copy(Ce),T=yt),p=N,I=!1}function oe(N,pe){N.side===Wn?xe(i.CULL_FACE):Q(i.CULL_FACE);let J=N.side===Xt;pe&&(J=!J),le(J),N.blending===Bs&&N.transparent===!1?se(Xn):se(N.blending,N.blendEquation,N.blendSrc,N.blendDst,N.blendEquationAlpha,N.blendSrcAlpha,N.blendDstAlpha,N.blendColor,N.blendAlpha,N.premultipliedAlpha),a.setFunc(N.depthFunc),a.setTest(N.depthTest),a.setMask(N.depthWrite),r.setMask(N.colorWrite);let me=N.stencilWrite;o.setTest(me),me&&(o.setMask(N.stencilWriteMask),o.setFunc(N.stencilFunc,N.stencilRef,N.stencilFuncMask),o.setOp(N.stencilFail,N.stencilZFail,N.stencilZPass)),Ue(N.polygonOffset,N.polygonOffsetFactor,N.polygonOffsetUnits),N.alphaToCoverage===!0?Q(i.SAMPLE_ALPHA_TO_COVERAGE):xe(i.SAMPLE_ALPHA_TO_COVERAGE)}function le(N){U!==N&&(N?i.frontFace(i.CW):i.frontFace(i.CCW),U=N)}function ue(N){N!==qu?(Q(i.CULL_FACE),N!==O&&(N===Dc?i.cullFace(i.BACK):N===Yu?i.cullFace(i.FRONT):i.cullFace(i.FRONT_AND_BACK))):xe(i.CULL_FACE),O=N}function ze(N){N!==H&&(W&&i.lineWidth(N),H=N)}function Ue(N,pe,J){N?(Q(i.POLYGON_OFFSET_FILL),(D!==pe||B!==J)&&(D=pe,B=J,a.getReversed()&&(pe=-pe),i.polygonOffset(pe,J))):xe(i.POLYGON_OFFSET_FILL)}function We(N){N?Q(i.SCISSOR_TEST):xe(i.SCISSOR_TEST)}function qe(N){N===void 0&&(N=i.TEXTURE0+X-1),K!==N&&(i.activeTexture(N),K=N)}function P(N,pe,J){J===void 0&&(K===null?J=i.TEXTURE0+X-1:J=K);let me=te[J];me===void 0&&(me={type:void 0,texture:void 0},te[J]=me),(me.type!==N||me.texture!==pe)&&(K!==J&&(i.activeTexture(J),K=J),i.bindTexture(N,pe||$[N]),me.type=N,me.texture=pe)}function lt(){let N=te[K];N!==void 0&&N.type!==void 0&&(i.bindTexture(N.type,null),N.type=void 0,N.texture=void 0)}function tt(){try{i.compressedTexImage2D(...arguments)}catch(N){Ve("WebGLState:",N)}}function b(){try{i.compressedTexImage3D(...arguments)}catch(N){Ve("WebGLState:",N)}}function _(){try{i.texSubImage2D(...arguments)}catch(N){Ve("WebGLState:",N)}}function F(){try{i.texSubImage3D(...arguments)}catch(N){Ve("WebGLState:",N)}}function V(){try{i.compressedTexSubImage2D(...arguments)}catch(N){Ve("WebGLState:",N)}}function Y(){try{i.compressedTexSubImage3D(...arguments)}catch(N){Ve("WebGLState:",N)}}function ce(){try{i.texStorage2D(...arguments)}catch(N){Ve("WebGLState:",N)}}function he(){try{i.texStorage3D(...arguments)}catch(N){Ve("WebGLState:",N)}}function Z(){try{i.texImage2D(...arguments)}catch(N){Ve("WebGLState:",N)}}function j(){try{i.texImage3D(...arguments)}catch(N){Ve("WebGLState:",N)}}function de(N){return d[N]!==void 0?d[N]:i.getParameter(N)}function Le(N,pe){d[N]!==pe&&(i.pixelStorei(N,pe),d[N]=pe)}function ge(N){ot.equals(N)===!1&&(i.scissor(N.x,N.y,N.z,N.w),ot.copy(N))}function fe(N){et.equals(N)===!1&&(i.viewport(N.x,N.y,N.z,N.w),et.copy(N))}function De(N,pe){let J=c.get(pe);J===void 0&&(J=new WeakMap,c.set(pe,J));let me=J.get(N);me===void 0&&(me=i.getUniformBlockIndex(pe,N.name),J.set(N,me))}function ke(N,pe){let me=c.get(pe).get(N);l.get(pe)!==me&&(i.uniformBlockBinding(pe,me,N.__bindingPointIndex),l.set(pe,me))}function Ze(){i.disable(i.BLEND),i.disable(i.CULL_FACE),i.disable(i.DEPTH_TEST),i.disable(i.POLYGON_OFFSET_FILL),i.disable(i.SCISSOR_TEST),i.disable(i.STENCIL_TEST),i.disable(i.SAMPLE_ALPHA_TO_COVERAGE),i.blendEquation(i.FUNC_ADD),i.blendFunc(i.ONE,i.ZERO),i.blendFuncSeparate(i.ONE,i.ZERO,i.ONE,i.ZERO),i.blendColor(0,0,0,0),i.colorMask(!0,!0,!0,!0),i.clearColor(0,0,0,0),i.depthMask(!0),i.depthFunc(i.LESS),a.setReversed(!1),i.clearDepth(1),i.stencilMask(4294967295),i.stencilFunc(i.ALWAYS,0,4294967295),i.stencilOp(i.KEEP,i.KEEP,i.KEEP),i.clearStencil(0),i.cullFace(i.BACK),i.frontFace(i.CCW),i.polygonOffset(0,0),i.activeTexture(i.TEXTURE0),i.bindFramebuffer(i.FRAMEBUFFER,null),i.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),i.bindFramebuffer(i.READ_FRAMEBUFFER,null),i.useProgram(null),i.lineWidth(1),i.scissor(0,0,i.canvas.width,i.canvas.height),i.viewport(0,0,i.canvas.width,i.canvas.height),i.pixelStorei(i.PACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_ALIGNMENT,4),i.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,!1),i.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,!1),i.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,i.BROWSER_DEFAULT_WEBGL),i.pixelStorei(i.PACK_ROW_LENGTH,0),i.pixelStorei(i.PACK_SKIP_PIXELS,0),i.pixelStorei(i.PACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_ROW_LENGTH,0),i.pixelStorei(i.UNPACK_IMAGE_HEIGHT,0),i.pixelStorei(i.UNPACK_SKIP_PIXELS,0),i.pixelStorei(i.UNPACK_SKIP_ROWS,0),i.pixelStorei(i.UNPACK_SKIP_IMAGES,0),h={},d={},K=null,te={},u={},f=new WeakMap,g=[],y=null,m=!1,p=null,w=null,A=null,M=null,E=null,S=null,C=null,x=new Ye(0,0,0),T=0,I=!1,U=null,O=null,H=null,D=null,B=null,ot.set(0,0,i.canvas.width,i.canvas.height),et.set(0,0,i.canvas.width,i.canvas.height),r.reset(),a.reset(),o.reset()}return{buffers:{color:r,depth:a,stencil:o},enable:Q,disable:xe,bindFramebuffer:He,drawBuffers:Se,useProgram:Ge,setBlending:se,setMaterial:oe,setFlipSided:le,setCullFace:ue,setLineWidth:ze,setPolygonOffset:Ue,setScissorTest:We,activeTexture:qe,bindTexture:P,unbindTexture:lt,compressedTexImage2D:tt,compressedTexImage3D:b,texImage2D:Z,texImage3D:j,pixelStorei:Le,getParameter:de,updateUBOMapping:De,uniformBlockBinding:ke,texStorage2D:ce,texStorage3D:he,texSubImage2D:_,texSubImage3D:F,compressedTexSubImage2D:V,compressedTexSubImage3D:Y,scissor:ge,viewport:fe,reset:Ze}}function vy(i,e,t,n,s,r,a){let o=e.has("WEBGL_multisampled_render_to_texture")?e.get("WEBGL_multisampled_render_to_texture"):null,l=typeof navigator>"u"?!1:/OculusBrowser/g.test(navigator.userAgent),c=new ae,h=new WeakMap,d=new Set,u,f=new WeakMap,g=!1;try{g=typeof OffscreenCanvas<"u"&&new OffscreenCanvas(1,1).getContext("2d")!==null}catch{}function y(b,_){return g?new OffscreenCanvas(b,_):pr("canvas")}function m(b,_,F){let V=1,Y=tt(b);if((Y.width>F||Y.height>F)&&(V=F/Math.max(Y.width,Y.height)),V<1)if(typeof HTMLImageElement<"u"&&b instanceof HTMLImageElement||typeof HTMLCanvasElement<"u"&&b instanceof HTMLCanvasElement||typeof ImageBitmap<"u"&&b instanceof ImageBitmap||typeof VideoFrame<"u"&&b instanceof VideoFrame){let ce=Math.floor(V*Y.width),he=Math.floor(V*Y.height);u===void 0&&(u=y(ce,he));let Z=_?y(ce,he):u;return Z.width=ce,Z.height=he,Z.getContext("2d").drawImage(b,0,0,ce,he),Oe("WebGLRenderer: Texture has been resized from ("+Y.width+"x"+Y.height+") to ("+ce+"x"+he+")."),Z}else return"data"in b&&Oe("WebGLRenderer: Image in DataTexture is too big ("+Y.width+"x"+Y.height+")."),b;return b}function p(b){return b.generateMipmaps}function w(b){i.generateMipmap(b)}function A(b){return b.isWebGLCubeRenderTarget?i.TEXTURE_CUBE_MAP:b.isWebGL3DRenderTarget?i.TEXTURE_3D:b.isWebGLArrayRenderTarget||b.isCompressedArrayTexture?i.TEXTURE_2D_ARRAY:i.TEXTURE_2D}function M(b,_,F,V,Y,ce=!1){if(b!==null){if(i[b]!==void 0)return i[b];Oe("WebGLRenderer: Attempt to use non-existing WebGL internal format '"+b+"'")}let he;V&&(he=e.get("EXT_texture_norm16"),he||Oe("WebGLRenderer: Unable to use normalized textures without EXT_texture_norm16 extension"));let Z=_;if(_===i.RED&&(F===i.FLOAT&&(Z=i.R32F),F===i.HALF_FLOAT&&(Z=i.R16F),F===i.UNSIGNED_BYTE&&(Z=i.R8),F===i.UNSIGNED_SHORT&&he&&(Z=he.R16_EXT),F===i.SHORT&&he&&(Z=he.R16_SNORM_EXT)),_===i.RED_INTEGER&&(F===i.UNSIGNED_BYTE&&(Z=i.R8UI),F===i.UNSIGNED_SHORT&&(Z=i.R16UI),F===i.UNSIGNED_INT&&(Z=i.R32UI),F===i.BYTE&&(Z=i.R8I),F===i.SHORT&&(Z=i.R16I),F===i.INT&&(Z=i.R32I)),_===i.RG&&(F===i.FLOAT&&(Z=i.RG32F),F===i.HALF_FLOAT&&(Z=i.RG16F),F===i.UNSIGNED_BYTE&&(Z=i.RG8),F===i.UNSIGNED_SHORT&&he&&(Z=he.RG16_EXT),F===i.SHORT&&he&&(Z=he.RG16_SNORM_EXT)),_===i.RG_INTEGER&&(F===i.UNSIGNED_BYTE&&(Z=i.RG8UI),F===i.UNSIGNED_SHORT&&(Z=i.RG16UI),F===i.UNSIGNED_INT&&(Z=i.RG32UI),F===i.BYTE&&(Z=i.RG8I),F===i.SHORT&&(Z=i.RG16I),F===i.INT&&(Z=i.RG32I)),_===i.RGB_INTEGER&&(F===i.UNSIGNED_BYTE&&(Z=i.RGB8UI),F===i.UNSIGNED_SHORT&&(Z=i.RGB16UI),F===i.UNSIGNED_INT&&(Z=i.RGB32UI),F===i.BYTE&&(Z=i.RGB8I),F===i.SHORT&&(Z=i.RGB16I),F===i.INT&&(Z=i.RGB32I)),_===i.RGBA_INTEGER&&(F===i.UNSIGNED_BYTE&&(Z=i.RGBA8UI),F===i.UNSIGNED_SHORT&&(Z=i.RGBA16UI),F===i.UNSIGNED_INT&&(Z=i.RGBA32UI),F===i.BYTE&&(Z=i.RGBA8I),F===i.SHORT&&(Z=i.RGBA16I),F===i.INT&&(Z=i.RGBA32I)),_===i.RGB&&(F===i.UNSIGNED_SHORT&&he&&(Z=he.RGB16_EXT),F===i.SHORT&&he&&(Z=he.RGB16_SNORM_EXT),F===i.UNSIGNED_INT_5_9_9_9_REV&&(Z=i.RGB9_E5),F===i.UNSIGNED_INT_10F_11F_11F_REV&&(Z=i.R11F_G11F_B10F)),_===i.RGBA){let j=ce?fr:it.getTransfer(Y);F===i.FLOAT&&(Z=i.RGBA32F),F===i.HALF_FLOAT&&(Z=i.RGBA16F),F===i.UNSIGNED_BYTE&&(Z=j===ut?i.SRGB8_ALPHA8:i.RGBA8),F===i.UNSIGNED_SHORT&&he&&(Z=he.RGBA16_EXT),F===i.SHORT&&he&&(Z=he.RGBA16_SNORM_EXT),F===i.UNSIGNED_SHORT_4_4_4_4&&(Z=i.RGBA4),F===i.UNSIGNED_SHORT_5_5_5_1&&(Z=i.RGB5_A1)}return(Z===i.R16F||Z===i.R32F||Z===i.RG16F||Z===i.RG32F||Z===i.RGBA16F||Z===i.RGBA32F)&&e.get("EXT_color_buffer_float"),Z}function E(b,_){let F;return b?_===null||_===Nn||_===ks?F=i.DEPTH24_STENCIL8:_===vn?F=i.DEPTH32F_STENCIL8:_===zs&&(F=i.DEPTH24_STENCIL8,Oe("DepthTexture: 16 bit depth attachment is not supported with stencil. Using 24-bit attachment.")):_===null||_===Nn||_===ks?F=i.DEPTH_COMPONENT24:_===vn?F=i.DEPTH_COMPONENT32F:_===zs&&(F=i.DEPTH_COMPONENT16),F}function S(b,_){return p(b)===!0||b.isFramebufferTexture&&b.minFilter!==kt&&b.minFilter!==Gt?Math.log2(Math.max(_.width,_.height))+1:b.mipmaps!==void 0&&b.mipmaps.length>0?b.mipmaps.length:b.isCompressedTexture&&Array.isArray(b.image)?_.mipmaps.length:1}function C(b){let _=b.target;_.removeEventListener("dispose",C),T(_),_.isVideoTexture&&h.delete(_),_.isHTMLTexture&&d.delete(_)}function x(b){let _=b.target;_.removeEventListener("dispose",x),U(_)}function T(b){let _=n.get(b);if(_.__webglInit===void 0)return;let F=b.source,V=f.get(F);if(V){let Y=V[_.__cacheKey];Y.usedTimes--,Y.usedTimes===0&&I(b),Object.keys(V).length===0&&f.delete(F)}n.remove(b)}function I(b){let _=n.get(b);i.deleteTexture(_.__webglTexture);let F=b.source,V=f.get(F);delete V[_.__cacheKey],a.memory.textures--}function U(b){let _=n.get(b);if(b.depthTexture&&(b.depthTexture.dispose(),n.remove(b.depthTexture)),b.isWebGLCubeRenderTarget)for(let V=0;V<6;V++){if(Array.isArray(_.__webglFramebuffer[V]))for(let Y=0;Y<_.__webglFramebuffer[V].length;Y++)i.deleteFramebuffer(_.__webglFramebuffer[V][Y]);else i.deleteFramebuffer(_.__webglFramebuffer[V]);_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer[V])}else{if(Array.isArray(_.__webglFramebuffer))for(let V=0;V<_.__webglFramebuffer.length;V++)i.deleteFramebuffer(_.__webglFramebuffer[V]);else i.deleteFramebuffer(_.__webglFramebuffer);if(_.__webglDepthbuffer&&i.deleteRenderbuffer(_.__webglDepthbuffer),_.__webglMultisampledFramebuffer&&i.deleteFramebuffer(_.__webglMultisampledFramebuffer),_.__webglColorRenderbuffer)for(let V=0;V<_.__webglColorRenderbuffer.length;V++)_.__webglColorRenderbuffer[V]&&i.deleteRenderbuffer(_.__webglColorRenderbuffer[V]);_.__webglDepthRenderbuffer&&i.deleteRenderbuffer(_.__webglDepthRenderbuffer)}let F=b.textures;for(let V=0,Y=F.length;V<Y;V++){let ce=n.get(F[V]);ce.__webglTexture&&(i.deleteTexture(ce.__webglTexture),a.memory.textures--),n.remove(F[V])}n.remove(b)}let O=0;function H(){O=0}function D(){return O}function B(b){O=b}function X(){let b=O;return b>=s.maxTextures&&Oe("WebGLTextures: Trying to use "+(b+1)+" texture units while this GPU supports only "+s.maxTextures),O+=1,b}function W(b){let _=[];return _.push(b.wrapS),_.push(b.wrapT),_.push(b.wrapR||0),_.push(b.magFilter),_.push(b.minFilter),_.push(b.anisotropy),_.push(b.internalFormat),_.push(b.format),_.push(b.type),_.push(b.generateMipmaps),_.push(b.premultiplyAlpha),_.push(b.flipY),_.push(b.unpackAlignment),_.push(b.colorSpace),_.join()}function ie(b,_){let F=n.get(b);if(b.isVideoTexture&&P(b),b.isRenderTargetTexture===!1&&b.isExternalTexture!==!0&&b.version>0&&F.__version!==b.version){let V=b.image;if(V===null)Oe("WebGLRenderer: Texture marked for update but no image data found.");else if(V.complete===!1)Oe("WebGLRenderer: Texture marked for update but image is incomplete");else{xe(F,b,_);return}}else b.isExternalTexture&&(F.__webglTexture=b.sourceTexture?b.sourceTexture:null);t.bindTexture(i.TEXTURE_2D,F.__webglTexture,i.TEXTURE0+_)}function q(b,_){let F=n.get(b);if(b.isRenderTargetTexture===!1&&b.version>0&&F.__version!==b.version){xe(F,b,_);return}else b.isExternalTexture&&(F.__webglTexture=b.sourceTexture?b.sourceTexture:null);t.bindTexture(i.TEXTURE_2D_ARRAY,F.__webglTexture,i.TEXTURE0+_)}function K(b,_){let F=n.get(b);if(b.isRenderTargetTexture===!1&&b.version>0&&F.__version!==b.version){xe(F,b,_);return}t.bindTexture(i.TEXTURE_3D,F.__webglTexture,i.TEXTURE0+_)}function te(b,_){let F=n.get(b);if(b.isCubeDepthTexture!==!0&&b.version>0&&F.__version!==b.version){He(F,b,_);return}t.bindTexture(i.TEXTURE_CUBE_MAP,F.__webglTexture,i.TEXTURE0+_)}let Ie={[Ms]:i.REPEAT,[kn]:i.CLAMP_TO_EDGE,[eo]:i.MIRRORED_REPEAT},we={[kt]:i.NEAREST,[pd]:i.NEAREST_MIPMAP_NEAREST,[Zr]:i.NEAREST_MIPMAP_LINEAR,[Gt]:i.LINEAR,[Bo]:i.LINEAR_MIPMAP_NEAREST,[Ci]:i.LINEAR_MIPMAP_LINEAR},ot={[xd]:i.NEVER,[Sd]:i.ALWAYS,[yd]:i.LESS,[bl]:i.LEQUAL,[vd]:i.EQUAL,[Sl]:i.GEQUAL,[Md]:i.GREATER,[bd]:i.NOTEQUAL};function et(b,_){if(_.type===vn&&e.has("OES_texture_float_linear")===!1&&(_.magFilter===Gt||_.magFilter===Bo||_.magFilter===Zr||_.magFilter===Ci||_.minFilter===Gt||_.minFilter===Bo||_.minFilter===Zr||_.minFilter===Ci)&&Oe("WebGLRenderer: Unable to use linear filtering with floating point textures. OES_texture_float_linear not supported on this device."),i.texParameteri(b,i.TEXTURE_WRAP_S,Ie[_.wrapS]),i.texParameteri(b,i.TEXTURE_WRAP_T,Ie[_.wrapT]),(b===i.TEXTURE_3D||b===i.TEXTURE_2D_ARRAY)&&i.texParameteri(b,i.TEXTURE_WRAP_R,Ie[_.wrapR]),i.texParameteri(b,i.TEXTURE_MAG_FILTER,we[_.magFilter]),i.texParameteri(b,i.TEXTURE_MIN_FILTER,we[_.minFilter]),_.compareFunction&&(i.texParameteri(b,i.TEXTURE_COMPARE_MODE,i.COMPARE_REF_TO_TEXTURE),i.texParameteri(b,i.TEXTURE_COMPARE_FUNC,ot[_.compareFunction])),e.has("EXT_texture_filter_anisotropic")===!0){if(_.magFilter===kt||_.minFilter!==Zr&&_.minFilter!==Ci||_.type===vn&&e.has("OES_texture_float_linear")===!1)return;if(_.anisotropy>1||n.get(_).__currentAnisotropy){let F=e.get("EXT_texture_filter_anisotropic");i.texParameterf(b,F.TEXTURE_MAX_ANISOTROPY_EXT,Math.min(_.anisotropy,s.getMaxAnisotropy())),n.get(_).__currentAnisotropy=_.anisotropy}}}function rt(b,_){let F=!1;b.__webglInit===void 0&&(b.__webglInit=!0,_.addEventListener("dispose",C));let V=_.source,Y=f.get(V);Y===void 0&&(Y={},f.set(V,Y));let ce=W(_);if(ce!==b.__cacheKey){Y[ce]===void 0&&(Y[ce]={texture:i.createTexture(),usedTimes:0},a.memory.textures++,F=!0),Y[ce].usedTimes++;let he=Y[b.__cacheKey];he!==void 0&&(Y[b.__cacheKey].usedTimes--,he.usedTimes===0&&I(_)),b.__cacheKey=ce,b.__webglTexture=Y[ce].texture}return F}function $(b,_,F){return Math.floor(Math.floor(b/F)/_)}function Q(b,_,F,V){let ce=b.updateRanges;if(ce.length===0)t.texSubImage2D(i.TEXTURE_2D,0,0,0,_.width,_.height,F,V,_.data);else{ce.sort((Le,ge)=>Le.start-ge.start);let he=0;for(let Le=1;Le<ce.length;Le++){let ge=ce[he],fe=ce[Le],De=ge.start+ge.count,ke=$(fe.start,_.width,4),Ze=$(ge.start,_.width,4);fe.start<=De+1&&ke===Ze&&$(fe.start+fe.count-1,_.width,4)===ke?ge.count=Math.max(ge.count,fe.start+fe.count-ge.start):(++he,ce[he]=fe)}ce.length=he+1;let Z=t.getParameter(i.UNPACK_ROW_LENGTH),j=t.getParameter(i.UNPACK_SKIP_PIXELS),de=t.getParameter(i.UNPACK_SKIP_ROWS);t.pixelStorei(i.UNPACK_ROW_LENGTH,_.width);for(let Le=0,ge=ce.length;Le<ge;Le++){let fe=ce[Le],De=Math.floor(fe.start/4),ke=Math.ceil(fe.count/4),Ze=De%_.width,N=Math.floor(De/_.width),pe=ke,J=1;t.pixelStorei(i.UNPACK_SKIP_PIXELS,Ze),t.pixelStorei(i.UNPACK_SKIP_ROWS,N),t.texSubImage2D(i.TEXTURE_2D,0,Ze,N,pe,J,F,V,_.data)}b.clearUpdateRanges(),t.pixelStorei(i.UNPACK_ROW_LENGTH,Z),t.pixelStorei(i.UNPACK_SKIP_PIXELS,j),t.pixelStorei(i.UNPACK_SKIP_ROWS,de)}}function xe(b,_,F){let V=i.TEXTURE_2D;(_.isDataArrayTexture||_.isCompressedArrayTexture)&&(V=i.TEXTURE_2D_ARRAY),_.isData3DTexture&&(V=i.TEXTURE_3D);let Y=rt(b,_),ce=_.source;t.bindTexture(V,b.__webglTexture,i.TEXTURE0+F);let he=n.get(ce);if(ce.version!==he.__version||Y===!0){if(t.activeTexture(i.TEXTURE0+F),(typeof ImageBitmap<"u"&&_.image instanceof ImageBitmap)===!1){let J=it.getPrimaries(it.workingColorSpace),me=_.colorSpace===si?null:it.getPrimaries(_.colorSpace),be=_.colorSpace===si||J===me?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,be)}t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment);let j=m(_.image,!1,s.maxTextureSize);j=lt(_,j);let de=r.convert(_.format,_.colorSpace),Le=r.convert(_.type),ge=M(_.internalFormat,de,Le,_.normalized,_.colorSpace,_.isVideoTexture);et(V,_);let fe,De=_.mipmaps,ke=_.isVideoTexture!==!0,Ze=he.__version===void 0||Y===!0,N=ce.dataReady,pe=S(_,j);if(_.isDepthTexture)ge=E(_.format===Pi,_.type),Ze&&(ke?t.texStorage2D(i.TEXTURE_2D,1,ge,j.width,j.height):t.texImage2D(i.TEXTURE_2D,0,ge,j.width,j.height,0,de,Le,null));else if(_.isDataTexture)if(De.length>0){ke&&Ze&&t.texStorage2D(i.TEXTURE_2D,pe,ge,De[0].width,De[0].height);for(let J=0,me=De.length;J<me;J++)fe=De[J],ke?N&&t.texSubImage2D(i.TEXTURE_2D,J,0,0,fe.width,fe.height,de,Le,fe.data):t.texImage2D(i.TEXTURE_2D,J,ge,fe.width,fe.height,0,de,Le,fe.data);_.generateMipmaps=!1}else ke?(Ze&&t.texStorage2D(i.TEXTURE_2D,pe,ge,j.width,j.height),N&&Q(_,j,de,Le)):t.texImage2D(i.TEXTURE_2D,0,ge,j.width,j.height,0,de,Le,j.data);else if(_.isCompressedTexture)if(_.isCompressedArrayTexture){ke&&Ze&&t.texStorage3D(i.TEXTURE_2D_ARRAY,pe,ge,De[0].width,De[0].height,j.depth);for(let J=0,me=De.length;J<me;J++)if(fe=De[J],_.format!==Mn)if(de!==null)if(ke){if(N)if(_.layerUpdates.size>0){let be=rh(fe.width,fe.height,_.format,_.type);for(let ne of _.layerUpdates){let Ne=fe.data.subarray(ne*be/fe.data.BYTES_PER_ELEMENT,(ne+1)*be/fe.data.BYTES_PER_ELEMENT);t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,J,0,0,ne,fe.width,fe.height,1,de,Ne)}}else t.compressedTexSubImage3D(i.TEXTURE_2D_ARRAY,J,0,0,0,fe.width,fe.height,j.depth,de,fe.data)}else t.compressedTexImage3D(i.TEXTURE_2D_ARRAY,J,ge,fe.width,fe.height,j.depth,0,fe.data,0,0);else Oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()");else ke?N&&t.texSubImage3D(i.TEXTURE_2D_ARRAY,J,0,0,0,fe.width,fe.height,j.depth,de,Le,fe.data):t.texImage3D(i.TEXTURE_2D_ARRAY,J,ge,fe.width,fe.height,j.depth,0,de,Le,fe.data);_.layerUpdates.size>0&&_.clearLayerUpdates()}else{ke&&Ze&&t.texStorage2D(i.TEXTURE_2D,pe,ge,De[0].width,De[0].height);for(let J=0,me=De.length;J<me;J++)fe=De[J],_.format!==Mn?de!==null?ke?N&&t.compressedTexSubImage2D(i.TEXTURE_2D,J,0,0,fe.width,fe.height,de,fe.data):t.compressedTexImage2D(i.TEXTURE_2D,J,ge,fe.width,fe.height,0,fe.data):Oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .uploadTexture()"):ke?N&&t.texSubImage2D(i.TEXTURE_2D,J,0,0,fe.width,fe.height,de,Le,fe.data):t.texImage2D(i.TEXTURE_2D,J,ge,fe.width,fe.height,0,de,Le,fe.data)}else if(_.isDataArrayTexture)if(ke){if(Ze&&t.texStorage3D(i.TEXTURE_2D_ARRAY,pe,ge,j.width,j.height,j.depth),N)if(_.layerUpdates.size>0){let J=rh(j.width,j.height,_.format,_.type);for(let me of _.layerUpdates){let be=j.data.subarray(me*J/j.data.BYTES_PER_ELEMENT,(me+1)*J/j.data.BYTES_PER_ELEMENT);t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,me,j.width,j.height,1,de,Le,be)}_.clearLayerUpdates()}else t.texSubImage3D(i.TEXTURE_2D_ARRAY,0,0,0,0,j.width,j.height,j.depth,de,Le,j.data)}else t.texImage3D(i.TEXTURE_2D_ARRAY,0,ge,j.width,j.height,j.depth,0,de,Le,j.data);else if(_.isData3DTexture)ke?(Ze&&t.texStorage3D(i.TEXTURE_3D,pe,ge,j.width,j.height,j.depth),N&&t.texSubImage3D(i.TEXTURE_3D,0,0,0,0,j.width,j.height,j.depth,de,Le,j.data)):t.texImage3D(i.TEXTURE_3D,0,ge,j.width,j.height,j.depth,0,de,Le,j.data);else if(_.isFramebufferTexture){if(Ze)if(ke)t.texStorage2D(i.TEXTURE_2D,pe,ge,j.width,j.height);else{let J=j.width,me=j.height;for(let be=0;be<pe;be++)t.texImage2D(i.TEXTURE_2D,be,ge,J,me,0,de,Le,null),J>>=1,me>>=1}}else if(_.isHTMLTexture){if("texElementImage2D"in i){let J=i.canvas;if(J.hasAttribute("layoutsubtree")||J.setAttribute("layoutsubtree","true"),j.parentNode!==J){J.appendChild(j),d.add(_),J.onpaint=me=>{let be=me.changedElements;for(let ne of d)be.includes(ne.image)&&(ne.needsUpdate=!0)},J.requestPaint();return}if(i.texElementImage2D.length===3)i.texElementImage2D(i.TEXTURE_2D,i.RGBA8,j);else{let be=i.RGBA,ne=i.RGBA,Ne=i.UNSIGNED_BYTE;i.texElementImage2D(i.TEXTURE_2D,0,be,ne,Ne,j)}i.texParameteri(i.TEXTURE_2D,i.TEXTURE_MIN_FILTER,i.LINEAR),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_S,i.CLAMP_TO_EDGE),i.texParameteri(i.TEXTURE_2D,i.TEXTURE_WRAP_T,i.CLAMP_TO_EDGE)}}else if(De.length>0){if(ke&&Ze){let J=tt(De[0]);t.texStorage2D(i.TEXTURE_2D,pe,ge,J.width,J.height)}for(let J=0,me=De.length;J<me;J++)fe=De[J],ke?N&&t.texSubImage2D(i.TEXTURE_2D,J,0,0,de,Le,fe):t.texImage2D(i.TEXTURE_2D,J,ge,de,Le,fe);_.generateMipmaps=!1}else if(ke){if(Ze){let J=tt(j);t.texStorage2D(i.TEXTURE_2D,pe,ge,J.width,J.height)}N&&t.texSubImage2D(i.TEXTURE_2D,0,0,0,de,Le,j)}else t.texImage2D(i.TEXTURE_2D,0,ge,de,Le,j);p(_)&&w(V),he.__version=ce.version,_.onUpdate&&_.onUpdate(_)}b.__version=_.version}function He(b,_,F){if(_.image.length!==6)return;let V=rt(b,_),Y=_.source;t.bindTexture(i.TEXTURE_CUBE_MAP,b.__webglTexture,i.TEXTURE0+F);let ce=n.get(Y);if(Y.version!==ce.__version||V===!0){t.activeTexture(i.TEXTURE0+F);let he=it.getPrimaries(it.workingColorSpace),Z=_.colorSpace===si?null:it.getPrimaries(_.colorSpace),j=_.colorSpace===si||he===Z?i.NONE:i.BROWSER_DEFAULT_WEBGL;t.pixelStorei(i.UNPACK_FLIP_Y_WEBGL,_.flipY),t.pixelStorei(i.UNPACK_PREMULTIPLY_ALPHA_WEBGL,_.premultiplyAlpha),t.pixelStorei(i.UNPACK_ALIGNMENT,_.unpackAlignment),t.pixelStorei(i.UNPACK_COLORSPACE_CONVERSION_WEBGL,j);let de=_.isCompressedTexture||_.image[0].isCompressedTexture,Le=_.image[0]&&_.image[0].isDataTexture,ge=[];for(let ne=0;ne<6;ne++)!de&&!Le?ge[ne]=m(_.image[ne],!0,s.maxCubemapSize):ge[ne]=Le?_.image[ne].image:_.image[ne],ge[ne]=lt(_,ge[ne]);let fe=ge[0],De=r.convert(_.format,_.colorSpace),ke=r.convert(_.type),Ze=M(_.internalFormat,De,ke,_.normalized,_.colorSpace),N=_.isVideoTexture!==!0,pe=ce.__version===void 0||V===!0,J=Y.dataReady,me=S(_,fe);et(i.TEXTURE_CUBE_MAP,_);let be;if(de){N&&pe&&t.texStorage2D(i.TEXTURE_CUBE_MAP,me,Ze,fe.width,fe.height);for(let ne=0;ne<6;ne++){be=ge[ne].mipmaps;for(let Ne=0;Ne<be.length;Ne++){let Ce=be[Ne];_.format!==Mn?De!==null?N?J&&t.compressedTexSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne,0,0,Ce.width,Ce.height,De,Ce.data):t.compressedTexImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne,Ze,Ce.width,Ce.height,0,Ce.data):Oe("WebGLRenderer: Attempt to load unsupported compressed texture format in .setTextureCube()"):N?J&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne,0,0,Ce.width,Ce.height,De,ke,Ce.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne,Ze,Ce.width,Ce.height,0,De,ke,Ce.data)}}}else{if(be=_.mipmaps,N&&pe){be.length>0&&me++;let ne=tt(ge[0]);t.texStorage2D(i.TEXTURE_CUBE_MAP,me,Ze,ne.width,ne.height)}for(let ne=0;ne<6;ne++)if(Le){N?J&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,0,0,0,ge[ne].width,ge[ne].height,De,ke,ge[ne].data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,0,Ze,ge[ne].width,ge[ne].height,0,De,ke,ge[ne].data);for(let Ne=0;Ne<be.length;Ne++){let yt=be[Ne].image[ne].image;N?J&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne+1,0,0,yt.width,yt.height,De,ke,yt.data):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne+1,Ze,yt.width,yt.height,0,De,ke,yt.data)}}else{N?J&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,0,0,0,De,ke,ge[ne]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,0,Ze,De,ke,ge[ne]);for(let Ne=0;Ne<be.length;Ne++){let Ce=be[Ne];N?J&&t.texSubImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne+1,0,0,De,ke,Ce.image[ne]):t.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+ne,Ne+1,Ze,De,ke,Ce.image[ne])}}}p(_)&&w(i.TEXTURE_CUBE_MAP),ce.__version=Y.version,_.onUpdate&&_.onUpdate(_)}b.__version=_.version}function Se(b,_,F,V,Y,ce){let he=r.convert(F.format,F.colorSpace),Z=r.convert(F.type),j=M(F.internalFormat,he,Z,F.normalized,F.colorSpace),de=n.get(_),Le=n.get(F);if(Le.__renderTarget=_,!de.__hasExternalTextures){let ge=Math.max(1,_.width>>ce),fe=Math.max(1,_.height>>ce);Y===i.TEXTURE_3D||Y===i.TEXTURE_2D_ARRAY?t.texImage3D(Y,ce,j,ge,fe,_.depth,0,he,Z,null):t.texImage2D(Y,ce,j,ge,fe,0,he,Z,null)}t.bindFramebuffer(i.FRAMEBUFFER,b),qe(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,V,Y,Le.__webglTexture,0,We(_)):(Y===i.TEXTURE_2D||Y>=i.TEXTURE_CUBE_MAP_POSITIVE_X&&Y<=i.TEXTURE_CUBE_MAP_NEGATIVE_Z)&&i.framebufferTexture2D(i.FRAMEBUFFER,V,Y,Le.__webglTexture,ce),t.bindFramebuffer(i.FRAMEBUFFER,null)}function Ge(b,_,F){if(i.bindRenderbuffer(i.RENDERBUFFER,b),_.depthBuffer){let V=_.depthTexture,Y=V&&V.isDepthTexture?V.type:null,ce=E(_.stencilBuffer,Y),he=_.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;qe(_)?o.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,We(_),ce,_.width,_.height):F?i.renderbufferStorageMultisample(i.RENDERBUFFER,We(_),ce,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,ce,_.width,_.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,he,i.RENDERBUFFER,b)}else{let V=_.textures;for(let Y=0;Y<V.length;Y++){let ce=V[Y],he=r.convert(ce.format,ce.colorSpace),Z=r.convert(ce.type),j=M(ce.internalFormat,he,Z,ce.normalized,ce.colorSpace);qe(_)?o.renderbufferStorageMultisampleEXT(i.RENDERBUFFER,We(_),j,_.width,_.height):F?i.renderbufferStorageMultisample(i.RENDERBUFFER,We(_),j,_.width,_.height):i.renderbufferStorage(i.RENDERBUFFER,j,_.width,_.height)}}i.bindRenderbuffer(i.RENDERBUFFER,null)}function pt(b,_,F){let V=_.isWebGLCubeRenderTarget===!0;if(t.bindFramebuffer(i.FRAMEBUFFER,b),!(_.depthTexture&&_.depthTexture.isDepthTexture))throw new Error("THREE.WebGLTextures: renderTarget.depthTexture must be an instance of THREE.DepthTexture.");let Y=n.get(_.depthTexture);if(Y.__renderTarget=_,(!Y.__webglTexture||_.depthTexture.image.width!==_.width||_.depthTexture.image.height!==_.height)&&(_.depthTexture.image.width=_.width,_.depthTexture.image.height=_.height,_.depthTexture.needsUpdate=!0),V){if(Y.__webglInit===void 0&&(Y.__webglInit=!0,_.depthTexture.addEventListener("dispose",C)),Y.__webglTexture===void 0){Y.__webglTexture=i.createTexture(),t.bindTexture(i.TEXTURE_CUBE_MAP,Y.__webglTexture),et(i.TEXTURE_CUBE_MAP,_.depthTexture);let de=r.convert(_.depthTexture.format),Le=r.convert(_.depthTexture.type),ge;_.depthTexture.format===Vn?ge=i.DEPTH_COMPONENT24:_.depthTexture.format===Pi&&(ge=i.DEPTH24_STENCIL8);for(let fe=0;fe<6;fe++)i.texImage2D(i.TEXTURE_CUBE_MAP_POSITIVE_X+fe,0,ge,_.width,_.height,0,de,Le,null)}}else ie(_.depthTexture,0);let ce=Y.__webglTexture,he=We(_),Z=V?i.TEXTURE_CUBE_MAP_POSITIVE_X+F:i.TEXTURE_2D,j=_.depthTexture.format===Pi?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;if(_.depthTexture.format===Vn)qe(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,j,Z,ce,0,he):i.framebufferTexture2D(i.FRAMEBUFFER,j,Z,ce,0);else if(_.depthTexture.format===Pi)qe(_)?o.framebufferTexture2DMultisampleEXT(i.FRAMEBUFFER,j,Z,ce,0,he):i.framebufferTexture2D(i.FRAMEBUFFER,j,Z,ce,0);else throw new Error("THREE.WebGLTextures: Unknown depthTexture format.")}function ee(b){let _=n.get(b),F=b.isWebGLCubeRenderTarget===!0;if(_.__boundDepthTexture!==b.depthTexture){let V=b.depthTexture;if(_.__depthDisposeCallback&&_.__depthDisposeCallback(),V){let Y=()=>{delete _.__boundDepthTexture,delete _.__depthDisposeCallback,V.removeEventListener("dispose",Y)};V.addEventListener("dispose",Y),_.__depthDisposeCallback=Y}_.__boundDepthTexture=V}if(b.depthTexture&&!_.__autoAllocateDepthBuffer)if(F)for(let V=0;V<6;V++)pt(_.__webglFramebuffer[V],b,V);else{let V=b.texture.mipmaps;V&&V.length>0?pt(_.__webglFramebuffer[0],b,0):pt(_.__webglFramebuffer,b,0)}else if(F){_.__webglDepthbuffer=[];for(let V=0;V<6;V++)if(t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[V]),_.__webglDepthbuffer[V]===void 0)_.__webglDepthbuffer[V]=i.createRenderbuffer(),Ge(_.__webglDepthbuffer[V],b,!1);else{let Y=b.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,ce=_.__webglDepthbuffer[V];i.bindRenderbuffer(i.RENDERBUFFER,ce),i.framebufferRenderbuffer(i.FRAMEBUFFER,Y,i.RENDERBUFFER,ce)}}else{let V=b.texture.mipmaps;if(V&&V.length>0?t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer[0]):t.bindFramebuffer(i.FRAMEBUFFER,_.__webglFramebuffer),_.__webglDepthbuffer===void 0)_.__webglDepthbuffer=i.createRenderbuffer(),Ge(_.__webglDepthbuffer,b,!1);else{let Y=b.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,ce=_.__webglDepthbuffer;i.bindRenderbuffer(i.RENDERBUFFER,ce),i.framebufferRenderbuffer(i.FRAMEBUFFER,Y,i.RENDERBUFFER,ce)}}t.bindFramebuffer(i.FRAMEBUFFER,null)}function se(b,_,F){let V=n.get(b);_!==void 0&&Se(V.__webglFramebuffer,b,b.texture,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,0),F!==void 0&&ee(b)}function oe(b){let _=b.texture,F=n.get(b),V=n.get(_);b.addEventListener("dispose",x);let Y=b.textures,ce=b.isWebGLCubeRenderTarget===!0,he=Y.length>1;if(he||(V.__webglTexture===void 0&&(V.__webglTexture=i.createTexture()),V.__version=_.version,a.memory.textures++),ce){F.__webglFramebuffer=[];for(let Z=0;Z<6;Z++)if(_.mipmaps&&_.mipmaps.length>0){F.__webglFramebuffer[Z]=[];for(let j=0;j<_.mipmaps.length;j++)F.__webglFramebuffer[Z][j]=i.createFramebuffer()}else F.__webglFramebuffer[Z]=i.createFramebuffer()}else{if(_.mipmaps&&_.mipmaps.length>0){F.__webglFramebuffer=[];for(let Z=0;Z<_.mipmaps.length;Z++)F.__webglFramebuffer[Z]=i.createFramebuffer()}else F.__webglFramebuffer=i.createFramebuffer();if(he)for(let Z=0,j=Y.length;Z<j;Z++){let de=n.get(Y[Z]);de.__webglTexture===void 0&&(de.__webglTexture=i.createTexture(),a.memory.textures++)}if(b.samples>0&&qe(b)===!1){F.__webglMultisampledFramebuffer=i.createFramebuffer(),F.__webglColorRenderbuffer=[],t.bindFramebuffer(i.FRAMEBUFFER,F.__webglMultisampledFramebuffer);for(let Z=0;Z<Y.length;Z++){let j=Y[Z];F.__webglColorRenderbuffer[Z]=i.createRenderbuffer(),i.bindRenderbuffer(i.RENDERBUFFER,F.__webglColorRenderbuffer[Z]);let de=r.convert(j.format,j.colorSpace),Le=r.convert(j.type),ge=M(j.internalFormat,de,Le,j.normalized,j.colorSpace,b.isXRRenderTarget===!0),fe=We(b);i.renderbufferStorageMultisample(i.RENDERBUFFER,fe,ge,b.width,b.height),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+Z,i.RENDERBUFFER,F.__webglColorRenderbuffer[Z])}i.bindRenderbuffer(i.RENDERBUFFER,null),b.depthBuffer&&(F.__webglDepthRenderbuffer=i.createRenderbuffer(),Ge(F.__webglDepthRenderbuffer,b,!0)),t.bindFramebuffer(i.FRAMEBUFFER,null)}}if(ce){t.bindTexture(i.TEXTURE_CUBE_MAP,V.__webglTexture),et(i.TEXTURE_CUBE_MAP,_);for(let Z=0;Z<6;Z++)if(_.mipmaps&&_.mipmaps.length>0)for(let j=0;j<_.mipmaps.length;j++)Se(F.__webglFramebuffer[Z][j],b,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+Z,j);else Se(F.__webglFramebuffer[Z],b,_,i.COLOR_ATTACHMENT0,i.TEXTURE_CUBE_MAP_POSITIVE_X+Z,0);p(_)&&w(i.TEXTURE_CUBE_MAP),t.unbindTexture()}else if(he){for(let Z=0,j=Y.length;Z<j;Z++){let de=Y[Z],Le=n.get(de),ge=i.TEXTURE_2D;(b.isWebGL3DRenderTarget||b.isWebGLArrayRenderTarget)&&(ge=b.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(ge,Le.__webglTexture),et(ge,de),Se(F.__webglFramebuffer,b,de,i.COLOR_ATTACHMENT0+Z,ge,0),p(de)&&w(ge)}t.unbindTexture()}else{let Z=i.TEXTURE_2D;if((b.isWebGL3DRenderTarget||b.isWebGLArrayRenderTarget)&&(Z=b.isWebGL3DRenderTarget?i.TEXTURE_3D:i.TEXTURE_2D_ARRAY),t.bindTexture(Z,V.__webglTexture),et(Z,_),_.mipmaps&&_.mipmaps.length>0)for(let j=0;j<_.mipmaps.length;j++)Se(F.__webglFramebuffer[j],b,_,i.COLOR_ATTACHMENT0,Z,j);else Se(F.__webglFramebuffer,b,_,i.COLOR_ATTACHMENT0,Z,0);p(_)&&w(Z),t.unbindTexture()}b.depthBuffer&&ee(b)}function le(b){let _=b.textures;for(let F=0,V=_.length;F<V;F++){let Y=_[F];if(p(Y)){let ce=A(b),he=n.get(Y).__webglTexture;t.bindTexture(ce,he),w(ce),t.unbindTexture()}}}let ue=[],ze=[];function Ue(b){if(b.samples>0){if(qe(b)===!1){let _=b.textures,F=b.width,V=b.height,Y=i.COLOR_BUFFER_BIT,ce=b.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT,he=n.get(b),Z=_.length>1;if(Z)for(let de=0;de<_.length;de++)t.bindFramebuffer(i.FRAMEBUFFER,he.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+de,i.RENDERBUFFER,null),t.bindFramebuffer(i.FRAMEBUFFER,he.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+de,i.TEXTURE_2D,null,0);t.bindFramebuffer(i.READ_FRAMEBUFFER,he.__webglMultisampledFramebuffer);let j=b.texture.mipmaps;j&&j.length>0?t.bindFramebuffer(i.DRAW_FRAMEBUFFER,he.__webglFramebuffer[0]):t.bindFramebuffer(i.DRAW_FRAMEBUFFER,he.__webglFramebuffer);for(let de=0;de<_.length;de++){if(b.resolveDepthBuffer&&(b.depthBuffer&&(Y|=i.DEPTH_BUFFER_BIT),b.stencilBuffer&&b.resolveStencilBuffer&&(Y|=i.STENCIL_BUFFER_BIT)),Z){i.framebufferRenderbuffer(i.READ_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.RENDERBUFFER,he.__webglColorRenderbuffer[de]);let Le=n.get(_[de]).__webglTexture;i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0,i.TEXTURE_2D,Le,0)}i.blitFramebuffer(0,0,F,V,0,0,F,V,Y,i.NEAREST),l===!0&&(ue.length=0,ze.length=0,ue.push(i.COLOR_ATTACHMENT0+de),b.depthBuffer&&b.storeMultisampledDepthBuffer===!1&&(ue.push(ce),ze.push(ce),i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,ze)),i.invalidateFramebuffer(i.READ_FRAMEBUFFER,ue))}if(t.bindFramebuffer(i.READ_FRAMEBUFFER,null),t.bindFramebuffer(i.DRAW_FRAMEBUFFER,null),Z)for(let de=0;de<_.length;de++){t.bindFramebuffer(i.FRAMEBUFFER,he.__webglMultisampledFramebuffer),i.framebufferRenderbuffer(i.FRAMEBUFFER,i.COLOR_ATTACHMENT0+de,i.RENDERBUFFER,he.__webglColorRenderbuffer[de]);let Le=n.get(_[de]).__webglTexture;t.bindFramebuffer(i.FRAMEBUFFER,he.__webglFramebuffer),i.framebufferTexture2D(i.DRAW_FRAMEBUFFER,i.COLOR_ATTACHMENT0+de,i.TEXTURE_2D,Le,0)}t.bindFramebuffer(i.DRAW_FRAMEBUFFER,he.__webglMultisampledFramebuffer)}else if(b.depthBuffer&&b.storeMultisampledDepthBuffer===!1&&l){let _=b.stencilBuffer?i.DEPTH_STENCIL_ATTACHMENT:i.DEPTH_ATTACHMENT;i.invalidateFramebuffer(i.DRAW_FRAMEBUFFER,[_])}}}function We(b){return Math.min(s.maxSamples,b.samples)}function qe(b){let _=n.get(b);return b.samples>0&&e.has("WEBGL_multisampled_render_to_texture")===!0&&_.__useRenderToTexture!==!1}function P(b){let _=a.render.frame;h.get(b)!==_&&(h.set(b,_),b.update())}function lt(b,_){let F=b.colorSpace,V=b.format,Y=b.type;return b.isCompressedTexture===!0||b.isVideoTexture===!0||F!==dr&&F!==si&&(it.getTransfer(F)===ut?(V!==Mn||Y!==rn)&&Oe("WebGLTextures: sRGB encoded textures have to use RGBAFormat and UnsignedByteType."):Ve("WebGLTextures: Unsupported texture color space:",F)),_}function tt(b){return typeof HTMLImageElement<"u"&&b instanceof HTMLImageElement?(c.width=b.naturalWidth||b.width,c.height=b.naturalHeight||b.height):typeof VideoFrame<"u"&&b instanceof VideoFrame?(c.width=b.displayWidth,c.height=b.displayHeight):(c.width=b.width,c.height=b.height),c}this.allocateTextureUnit=X,this.resetTextureUnits=H,this.getTextureUnits=D,this.setTextureUnits=B,this.setTexture2D=ie,this.setTexture2DArray=q,this.setTexture3D=K,this.setTextureCube=te,this.rebindTextures=se,this.setupRenderTarget=oe,this.updateRenderTargetMipmap=le,this.updateMultisampleRenderTarget=Ue,this.setupDepthRenderbuffer=ee,this.setupFrameBufferTexture=Se,this.useMultisampledRTT=qe,this.isReversedDepthBuffer=function(){return t.buffers.depth.getReversed()}}function My(i,e){function t(n,s=si){let r,a=it.getTransfer(s);if(n===rn)return i.UNSIGNED_BYTE;if(n===ko)return i.UNSIGNED_SHORT_4_4_4_4;if(n===Vo)return i.UNSIGNED_SHORT_5_5_5_1;if(n===Zc)return i.UNSIGNED_INT_5_9_9_9_REV;if(n===$c)return i.UNSIGNED_INT_10F_11F_11F_REV;if(n===qc)return i.BYTE;if(n===Yc)return i.SHORT;if(n===zs)return i.UNSIGNED_SHORT;if(n===zo)return i.INT;if(n===Nn)return i.UNSIGNED_INT;if(n===vn)return i.FLOAT;if(n===Un)return i.HALF_FLOAT;if(n===Jc)return i.ALPHA;if(n===jc)return i.RGB;if(n===Mn)return i.RGBA;if(n===Vn)return i.DEPTH_COMPONENT;if(n===Pi)return i.DEPTH_STENCIL;if(n===Ho)return i.RED;if(n===Go)return i.RED_INTEGER;if(n===Ii)return i.RG;if(n===Wo)return i.RG_INTEGER;if(n===Xo)return i.RGBA_INTEGER;if(n===$r||n===Jr||n===jr||n===Kr)if(a===ut)if(r=e.get("WEBGL_compressed_texture_s3tc_srgb"),r!==null){if(n===$r)return r.COMPRESSED_SRGB_S3TC_DXT1_EXT;if(n===Jr)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT1_EXT;if(n===jr)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT3_EXT;if(n===Kr)return r.COMPRESSED_SRGB_ALPHA_S3TC_DXT5_EXT}else return null;else if(r=e.get("WEBGL_compressed_texture_s3tc"),r!==null){if(n===$r)return r.COMPRESSED_RGB_S3TC_DXT1_EXT;if(n===Jr)return r.COMPRESSED_RGBA_S3TC_DXT1_EXT;if(n===jr)return r.COMPRESSED_RGBA_S3TC_DXT3_EXT;if(n===Kr)return r.COMPRESSED_RGBA_S3TC_DXT5_EXT}else return null;if(n===qo||n===Yo||n===Zo||n===$o)if(r=e.get("WEBGL_compressed_texture_pvrtc"),r!==null){if(n===qo)return r.COMPRESSED_RGB_PVRTC_4BPPV1_IMG;if(n===Yo)return r.COMPRESSED_RGB_PVRTC_2BPPV1_IMG;if(n===Zo)return r.COMPRESSED_RGBA_PVRTC_4BPPV1_IMG;if(n===$o)return r.COMPRESSED_RGBA_PVRTC_2BPPV1_IMG}else return null;if(n===Jo||n===jo||n===Ko||n===Qo||n===el||n===Qr||n===tl)if(r=e.get("WEBGL_compressed_texture_etc"),r!==null){if(n===Jo||n===jo)return a===ut?r.COMPRESSED_SRGB8_ETC2:r.COMPRESSED_RGB8_ETC2;if(n===Ko)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ETC2_EAC:r.COMPRESSED_RGBA8_ETC2_EAC;if(n===Qo)return r.COMPRESSED_R11_EAC;if(n===el)return r.COMPRESSED_SIGNED_R11_EAC;if(n===Qr)return r.COMPRESSED_RG11_EAC;if(n===tl)return r.COMPRESSED_SIGNED_RG11_EAC}else return null;if(n===nl||n===il||n===sl||n===rl||n===al||n===ol||n===ll||n===cl||n===hl||n===ul||n===dl||n===fl||n===pl||n===ml)if(r=e.get("WEBGL_compressed_texture_astc"),r!==null){if(n===nl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_4x4_KHR:r.COMPRESSED_RGBA_ASTC_4x4_KHR;if(n===il)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_5x4_KHR:r.COMPRESSED_RGBA_ASTC_5x4_KHR;if(n===sl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_5x5_KHR:r.COMPRESSED_RGBA_ASTC_5x5_KHR;if(n===rl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_6x5_KHR:r.COMPRESSED_RGBA_ASTC_6x5_KHR;if(n===al)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_6x6_KHR:r.COMPRESSED_RGBA_ASTC_6x6_KHR;if(n===ol)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x5_KHR:r.COMPRESSED_RGBA_ASTC_8x5_KHR;if(n===ll)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x6_KHR:r.COMPRESSED_RGBA_ASTC_8x6_KHR;if(n===cl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_8x8_KHR:r.COMPRESSED_RGBA_ASTC_8x8_KHR;if(n===hl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x5_KHR:r.COMPRESSED_RGBA_ASTC_10x5_KHR;if(n===ul)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x6_KHR:r.COMPRESSED_RGBA_ASTC_10x6_KHR;if(n===dl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x8_KHR:r.COMPRESSED_RGBA_ASTC_10x8_KHR;if(n===fl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_10x10_KHR:r.COMPRESSED_RGBA_ASTC_10x10_KHR;if(n===pl)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_12x10_KHR:r.COMPRESSED_RGBA_ASTC_12x10_KHR;if(n===ml)return a===ut?r.COMPRESSED_SRGB8_ALPHA8_ASTC_12x12_KHR:r.COMPRESSED_RGBA_ASTC_12x12_KHR}else return null;if(n===gl||n===_l||n===xl)if(r=e.get("EXT_texture_compression_bptc"),r!==null){if(n===gl)return a===ut?r.COMPRESSED_SRGB_ALPHA_BPTC_UNORM_EXT:r.COMPRESSED_RGBA_BPTC_UNORM_EXT;if(n===_l)return r.COMPRESSED_RGB_BPTC_SIGNED_FLOAT_EXT;if(n===xl)return r.COMPRESSED_RGB_BPTC_UNSIGNED_FLOAT_EXT}else return null;if(n===yl||n===vl||n===ea||n===Ml)if(r=e.get("EXT_texture_compression_rgtc"),r!==null){if(n===yl)return r.COMPRESSED_RED_RGTC1_EXT;if(n===vl)return r.COMPRESSED_SIGNED_RED_RGTC1_EXT;if(n===ea)return r.COMPRESSED_RED_GREEN_RGTC2_EXT;if(n===Ml)return r.COMPRESSED_SIGNED_RED_GREEN_RGTC2_EXT}else return null;return n===ks?i.UNSIGNED_INT_24_8:i[n]!==void 0?i[n]:null}return{convert:t}}var by=`
void main() {

	gl_Position = vec4( position, 1.0 );

}`,Sy=`
uniform sampler2DArray depthColor;
uniform float depthWidth;
uniform float depthHeight;

void main() {

	vec2 coord = vec2( gl_FragCoord.x / depthWidth, gl_FragCoord.y / depthHeight );

	if ( coord.x >= 1.0 ) {

		gl_FragDepth = texture( depthColor, vec3( coord.x - 1.0, coord.y, 1 ) ).r;

	} else {

		gl_FragDepth = texture( depthColor, vec3( coord.x, coord.y, 0 ) ).r;

	}

}`,Eh=class{constructor(){this.texture=null,this.mesh=null,this.depthNear=0,this.depthFar=0}init(e,t){if(this.texture===null){let n=new Tr(e.texture);(e.depthNear!==t.depthNear||e.depthFar!==t.depthFar)&&(this.depthNear=e.depthNear,this.depthFar=e.depthFar),this.texture=n}}getMesh(e){if(this.texture!==null&&this.mesh===null){let t=e.cameras[0].viewport,n=new tn({vertexShader:by,fragmentShader:Sy,uniforms:{depthColor:{value:this.texture},depthWidth:{value:t.z},depthHeight:{value:t.w}}});this.mesh=new mt(new Fr(20,20),n)}return this.mesh}reset(){this.texture=null,this.mesh=null}getDepthTexture(){return this.texture}},Th=class extends In{constructor(e,t){super();let n=this,s=null,r=1,a=null,o="local-floor",l=1,c=null,h=null,d=null,u=null,f=null,g=null,y=typeof XRWebGLBinding<"u",m=new Eh,p={},w=t.getContextAttributes(),A=null,M=null,E=[],S=[],C=new ae,x=null,T=null,I=new Ht;I.viewport=new St;let U=new Ht;U.viewport=new St;let O=[I,U],H=new Io,D=null,B=null;this.cameraAutoUpdate=!0,this.enabled=!1,this.isPresenting=!1,this.getController=function($){let Q=E[$];return Q===void 0&&(Q=new As,E[$]=Q),Q.getTargetRaySpace()},this.getControllerGrip=function($){let Q=E[$];return Q===void 0&&(Q=new As,E[$]=Q),Q.getGripSpace()},this.getHand=function($){let Q=E[$];return Q===void 0&&(Q=new As,E[$]=Q),Q.getHandSpace()};function X($){let Q=S.indexOf($.inputSource);if(Q===-1)return;let xe=E[Q];xe!==void 0&&(xe.update($.inputSource,$.frame,c||a),xe.dispatchEvent({type:$.type,data:$.inputSource}))}function W(){s.removeEventListener("select",X),s.removeEventListener("selectstart",X),s.removeEventListener("selectend",X),s.removeEventListener("squeeze",X),s.removeEventListener("squeezestart",X),s.removeEventListener("squeezeend",X),s.removeEventListener("end",W),s.removeEventListener("inputsourceschange",ie);for(let $=0;$<E.length;$++){let Q=S[$];Q!==null&&(S[$]=null,E[$].disconnect(Q))}D=null,B=null,m.reset();for(let $ in p)delete p[$];if(e.setRenderTarget(A),f=null,u=null,d=null,s=null,M=null,rt.stop(),n.isPresenting=!1,e.setPixelRatio(x),e.setSize(C.width,C.height,!1),T!==null){let $=T.camera;$.fov=T.fov,$.zoom=T.zoom,$.updateProjectionMatrix(),T=null}n.dispatchEvent({type:"sessionend"})}this.setFramebufferScaleFactor=function($){r=$,n.isPresenting===!0&&Oe("WebXRManager: Cannot change framebuffer scale while presenting.")},this.setReferenceSpaceType=function($){o=$,n.isPresenting===!0&&Oe("WebXRManager: Cannot change reference space type while presenting.")},this.getReferenceSpace=function(){return c||a},this.setReferenceSpace=function($){c=$},this.getBaseLayer=function(){return u!==null?u:f},this.getBinding=function(){return d===null&&y&&(d=new XRWebGLBinding(s,t)),d},this.getFrame=function(){return g},this.getSession=function(){return s},this.setSession=async function($){if(s=$,s!==null){if(A=e.getRenderTarget(),s.addEventListener("select",X),s.addEventListener("selectstart",X),s.addEventListener("selectend",X),s.addEventListener("squeeze",X),s.addEventListener("squeezestart",X),s.addEventListener("squeezeend",X),s.addEventListener("end",W),s.addEventListener("inputsourceschange",ie),w.xrCompatible!==!0&&await t.makeXRCompatible(),x=e.getPixelRatio(),e.getSize(C),y&&"createProjectionLayer"in XRWebGLBinding.prototype){let xe=null,He=null,Se=null;w.depth&&(Se=w.stencil?t.DEPTH24_STENCIL8:t.DEPTH_COMPONENT24,xe=w.stencil?Pi:Vn,He=w.stencil?ks:Nn);let Ge={colorFormat:t.RGBA8,depthFormat:Se,scaleFactor:r};d=this.getBinding(),u=d.createProjectionLayer(Ge),s.updateRenderState({layers:[u]}),e.setPixelRatio(1),e.setSize(u.textureWidth,u.textureHeight,!1),M=new sn(u.textureWidth,u.textureHeight,{format:Mn,type:rn,depthTexture:new vi(u.textureWidth,u.textureHeight,He,void 0,void 0,void 0,void 0,void 0,void 0,xe),stencilBuffer:w.stencil,colorSpace:e.outputColorSpace,samples:w.antialias?4:0,resolveDepthBuffer:u.ignoreDepthValues===!1,resolveStencilBuffer:u.ignoreDepthValues===!1,storeMultisampledDepthBuffer:u.ignoreDepthValues===!1,storeMultisampledStencilBuffer:u.ignoreDepthValues===!1})}else{let xe={antialias:w.antialias,alpha:!0,depth:w.depth,stencil:w.stencil,framebufferScaleFactor:r};f=new XRWebGLLayer(s,t,xe),s.updateRenderState({baseLayer:f}),e.setPixelRatio(1),e.setSize(f.framebufferWidth,f.framebufferHeight,!1),M=new sn(f.framebufferWidth,f.framebufferHeight,{format:Mn,type:rn,colorSpace:e.outputColorSpace,stencilBuffer:w.stencil,resolveDepthBuffer:f.ignoreDepthValues===!1,resolveStencilBuffer:f.ignoreDepthValues===!1,storeMultisampledDepthBuffer:f.ignoreDepthValues===!1,storeMultisampledStencilBuffer:f.ignoreDepthValues===!1})}M.isXRRenderTarget=!0,this.setFoveation(l),c=null,a=await s.requestReferenceSpace(o),rt.setContext(s),rt.start(),n.isPresenting=!0,n.dispatchEvent({type:"sessionstart"})}},this.getEnvironmentBlendMode=function(){if(s!==null)return s.environmentBlendMode},this.getDepthTexture=function(){return m.getDepthTexture()};function ie($){for(let Q=0;Q<$.removed.length;Q++){let xe=$.removed[Q],He=S.indexOf(xe);He>=0&&(S[He]=null,E[He].disconnect(xe))}for(let Q=0;Q<$.added.length;Q++){let xe=$.added[Q],He=S.indexOf(xe);if(He===-1){for(let Ge=0;Ge<E.length;Ge++)if(Ge>=S.length){S.push(xe),He=Ge;break}else if(S[Ge]===null){S[Ge]=xe,He=Ge;break}if(He===-1)break}let Se=E[He];Se&&Se.connect(xe)}}let q=new R,K=new R;function te($,Q,xe){q.setFromMatrixPosition(Q.matrixWorld),K.setFromMatrixPosition(xe.matrixWorld);let He=q.distanceTo(K),Se=Q.projectionMatrix.elements,Ge=xe.projectionMatrix.elements,pt=Se[14]/(Se[10]-1),ee=Se[14]/(Se[10]+1),se=(Se[9]+1)/Se[5],oe=(Se[9]-1)/Se[5],le=(Se[8]-1)/Se[0],ue=(Ge[8]+1)/Ge[0],ze=pt*le,Ue=pt*ue,We=He/(-le+ue),qe=We*-le;if(Q.matrixWorld.decompose($.position,$.quaternion,$.scale),$.translateX(qe),$.translateZ(We),$.matrixWorld.compose($.position,$.quaternion,$.scale),$.matrixWorldInverse.copy($.matrixWorld).invert(),Se[10]===-1)$.projectionMatrix.copy(Q.projectionMatrix),$.projectionMatrixInverse.copy(Q.projectionMatrixInverse);else{let P=pt+We,lt=ee+We,tt=ze-qe,b=Ue+(He-qe),_=se*ee/lt*P,F=oe*ee/lt*P;$.projectionMatrix.makePerspective(tt,b,_,F,P,lt),$.projectionMatrixInverse.copy($.projectionMatrix).invert()}}function Ie($,Q){Q===null?$.matrixWorld.copy($.matrix):$.matrixWorld.multiplyMatrices(Q.matrixWorld,$.matrix),$.matrixWorldInverse.copy($.matrixWorld).invert()}this.updateCamera=function($){if(s===null)return;let Q=$.near,xe=$.far;m.texture!==null&&(m.depthNear>0&&(Q=m.depthNear),m.depthFar>0&&(xe=m.depthFar)),H.near=U.near=I.near=Q,H.far=U.far=I.far=xe,(D!==H.near||B!==H.far)&&(s.updateRenderState({depthNear:H.near,depthFar:H.far}),D=H.near,B=H.far),H.layers.mask=$.layers.mask|6,I.layers.mask=H.layers.mask&-5,U.layers.mask=H.layers.mask&-3;let He=$.parent,Se=H.cameras;Ie(H,He);for(let Ge=0;Ge<Se.length;Ge++)Ie(Se[Ge],He);Se.length===2?te(H,I,U):H.projectionMatrix.copy(I.projectionMatrix),T===null&&$.isPerspectiveCamera&&(T={camera:$,fov:$.fov,zoom:$.zoom}),we($,H,He)};function we($,Q,xe){xe===null?$.matrix.copy(Q.matrixWorld):($.matrix.copy(xe.matrixWorld),$.matrix.invert(),$.matrix.multiply(Q.matrixWorld)),$.matrix.decompose($.position,$.quaternion,$.scale),$.updateMatrixWorld(!0),$.projectionMatrix.copy(Q.projectionMatrix),$.projectionMatrixInverse.copy(Q.projectionMatrixInverse),$.isPerspectiveCamera&&($.fov=Es*2*Math.atan(1/$.projectionMatrix.elements[5]),$.zoom=1)}this.getCamera=function(){return H},this.getFoveation=function(){if(!(u===null&&f===null))return l},this.setFoveation=function($){l=$,u!==null&&(u.fixedFoveation=$),f!==null&&f.fixedFoveation!==void 0&&(f.fixedFoveation=$)},this.hasDepthSensing=function(){return m.texture!==null},this.getDepthSensingMesh=function(){return m.getMesh(H)},this.getCameraTexture=function($){return p[$]};let ot=null;function et($,Q){if(h=Q.getViewerPose(c||a),g=Q,h!==null){let xe=h.views;f!==null&&(e.setRenderTargetFramebuffer(M,f.framebuffer),e.setRenderTarget(M));let He=!1;xe.length!==H.cameras.length&&(H.cameras.length=0,He=!0);for(let ee=0;ee<xe.length;ee++){let se=xe[ee],oe=null;if(f!==null)oe=f.getViewport(se);else{let ue=d.getViewSubImage(u,se);oe=ue.viewport,ee===0&&(e.setRenderTargetTextures(M,ue.colorTexture,ue.depthStencilTexture),e.setRenderTarget(M))}let le=O[ee];le===void 0&&(le=new Ht,le.layers.enable(ee),le.viewport=new St,O[ee]=le),le.matrix.fromArray(se.transform.matrix),le.matrix.decompose(le.position,le.quaternion,le.scale),le.projectionMatrix.fromArray(se.projectionMatrix),le.projectionMatrixInverse.copy(le.projectionMatrix).invert(),le.viewport.set(oe.x,oe.y,oe.width,oe.height),ee===0&&(H.matrix.copy(le.matrix),H.matrix.decompose(H.position,H.quaternion,H.scale)),He===!0&&H.cameras.push(le)}let Se=s.enabledFeatures;if(Se&&Se.includes("depth-sensing")&&s.depthUsage=="gpu-optimized"&&y){d=n.getBinding();let ee=d.getDepthInformation(xe[0]);ee&&ee.isValid&&ee.texture&&m.init(ee,s.renderState)}if(Se&&Se.includes("camera-access")&&y){e.state.unbindTexture(),d=n.getBinding();for(let ee=0;ee<xe.length;ee++){let se=xe[ee].camera;if(se){let oe=p[se];oe||(oe=new Tr,p[se]=oe);let le=d.getCameraImage(se);oe.sourceTexture=le}}}}for(let xe=0;xe<E.length;xe++){let He=S[xe],Se=E[xe];He!==null&&Se!==void 0&&Se.update(He,Q,c||a)}ot&&ot($,Q),Q.detectedPlanes&&n.dispatchEvent({type:"planesdetected",data:Q}),g=null}let rt=new rf;rt.setAnimationLoop(et),this.setAnimationLoop=function($){ot=$},this.dispose=function(){}}},Ey=new dt,uf=new Xe;uf.set(-1,0,0,0,1,0,0,0,1);function Ty(i,e){function t(m,p){m.matrixAutoUpdate===!0&&m.updateMatrix(),p.value.copy(m.matrix)}function n(m,p){p.color.getRGB(m.fogColor.value,nh(i)),p.isFog?(m.fogNear.value=p.near,m.fogFar.value=p.far):p.isFogExp2&&(m.fogDensity.value=p.density)}function s(m,p,w,A,M){p.isNodeMaterial?p.uniformsNeedUpdate=!1:p.isMeshBasicMaterial?r(m,p):p.isMeshLambertMaterial?(r(m,p),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)):p.isMeshToonMaterial?(r(m,p),d(m,p)):p.isMeshPhongMaterial?(r(m,p),h(m,p),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)):p.isMeshStandardMaterial?(r(m,p),u(m,p),p.isMeshPhysicalMaterial&&f(m,p,M)):p.isMeshMatcapMaterial?(r(m,p),g(m,p)):p.isMeshDepthMaterial?r(m,p):p.isMeshDistanceMaterial?(r(m,p),y(m,p)):p.isMeshNormalMaterial?r(m,p):p.isLineBasicMaterial?(a(m,p),p.isLineDashedMaterial&&o(m,p)):p.isPointsMaterial?l(m,p,w,A):p.isSpriteMaterial?c(m,p):p.isShadowMaterial?(m.color.value.copy(p.color),m.opacity.value=p.opacity):p.isShaderMaterial&&(p.uniformsNeedUpdate=!1)}function r(m,p){m.opacity.value=p.opacity,p.color&&m.diffuse.value.copy(p.color),p.emissive&&m.emissive.value.copy(p.emissive).multiplyScalar(p.emissiveIntensity),p.map&&(m.map.value=p.map,t(p.map,m.mapTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.bumpMap&&(m.bumpMap.value=p.bumpMap,t(p.bumpMap,m.bumpMapTransform),m.bumpScale.value=p.bumpScale,p.side===Xt&&(m.bumpScale.value*=-1)),p.normalMap&&(m.normalMap.value=p.normalMap,t(p.normalMap,m.normalMapTransform),m.normalScale.value.copy(p.normalScale),p.side===Xt&&m.normalScale.value.negate()),p.displacementMap&&(m.displacementMap.value=p.displacementMap,t(p.displacementMap,m.displacementMapTransform),m.displacementScale.value=p.displacementScale,m.displacementBias.value=p.displacementBias),p.emissiveMap&&(m.emissiveMap.value=p.emissiveMap,t(p.emissiveMap,m.emissiveMapTransform)),p.specularMap&&(m.specularMap.value=p.specularMap,t(p.specularMap,m.specularMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest);let w=e.get(p),A=w.envMap,M=w.envMapRotation;A&&(m.envMap.value=A,m.envMapRotation.value.setFromMatrix4(Ey.makeRotationFromEuler(M)).transpose(),A.isCubeTexture&&A.isRenderTargetTexture===!1&&m.envMapRotation.value.premultiply(uf),m.reflectivity.value=p.reflectivity,m.ior.value=p.ior,m.refractionRatio.value=p.refractionRatio),p.lightMap&&(m.lightMap.value=p.lightMap,m.lightMapIntensity.value=p.lightMapIntensity,t(p.lightMap,m.lightMapTransform)),p.aoMap&&(m.aoMap.value=p.aoMap,m.aoMapIntensity.value=p.aoMapIntensity,t(p.aoMap,m.aoMapTransform))}function a(m,p){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,p.map&&(m.map.value=p.map,t(p.map,m.mapTransform))}function o(m,p){m.dashSize.value=p.dashSize,m.totalSize.value=p.dashSize+p.gapSize,m.scale.value=p.scale}function l(m,p,w,A){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,m.size.value=p.size*w,m.scale.value=A*.5,p.map&&(m.map.value=p.map,t(p.map,m.uvTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest)}function c(m,p){m.diffuse.value.copy(p.color),m.opacity.value=p.opacity,m.rotation.value=p.rotation,p.map&&(m.map.value=p.map,t(p.map,m.mapTransform)),p.alphaMap&&(m.alphaMap.value=p.alphaMap,t(p.alphaMap,m.alphaMapTransform)),p.alphaTest>0&&(m.alphaTest.value=p.alphaTest)}function h(m,p){m.specular.value.copy(p.specular),m.shininess.value=Math.max(p.shininess,1e-4)}function d(m,p){p.gradientMap&&(m.gradientMap.value=p.gradientMap)}function u(m,p){m.metalness.value=p.metalness,p.metalnessMap&&(m.metalnessMap.value=p.metalnessMap,t(p.metalnessMap,m.metalnessMapTransform)),m.roughness.value=p.roughness,p.roughnessMap&&(m.roughnessMap.value=p.roughnessMap,t(p.roughnessMap,m.roughnessMapTransform)),p.envMap&&(m.envMapIntensity.value=p.envMapIntensity)}function f(m,p,w){m.ior.value=p.ior,p.sheen>0&&(m.sheenColor.value.copy(p.sheenColor).multiplyScalar(p.sheen),m.sheenRoughness.value=p.sheenRoughness,p.sheenColorMap&&(m.sheenColorMap.value=p.sheenColorMap,t(p.sheenColorMap,m.sheenColorMapTransform)),p.sheenRoughnessMap&&(m.sheenRoughnessMap.value=p.sheenRoughnessMap,t(p.sheenRoughnessMap,m.sheenRoughnessMapTransform))),p.clearcoat>0&&(m.clearcoat.value=p.clearcoat,m.clearcoatRoughness.value=p.clearcoatRoughness,p.clearcoatMap&&(m.clearcoatMap.value=p.clearcoatMap,t(p.clearcoatMap,m.clearcoatMapTransform)),p.clearcoatRoughnessMap&&(m.clearcoatRoughnessMap.value=p.clearcoatRoughnessMap,t(p.clearcoatRoughnessMap,m.clearcoatRoughnessMapTransform)),p.clearcoatNormalMap&&(m.clearcoatNormalMap.value=p.clearcoatNormalMap,t(p.clearcoatNormalMap,m.clearcoatNormalMapTransform),m.clearcoatNormalScale.value.copy(p.clearcoatNormalScale),p.side===Xt&&m.clearcoatNormalScale.value.negate())),p.dispersion>0&&(m.dispersion.value=p.dispersion),p.retroreflectivity>0&&(m.retroreflectivity.value=p.retroreflectivity),p.iridescence>0&&(m.iridescence.value=p.iridescence,m.iridescenceIOR.value=p.iridescenceIOR,m.iridescenceThicknessMinimum.value=p.iridescenceThicknessRange[0],m.iridescenceThicknessMaximum.value=p.iridescenceThicknessRange[1],p.iridescenceMap&&(m.iridescenceMap.value=p.iridescenceMap,t(p.iridescenceMap,m.iridescenceMapTransform)),p.iridescenceThicknessMap&&(m.iridescenceThicknessMap.value=p.iridescenceThicknessMap,t(p.iridescenceThicknessMap,m.iridescenceThicknessMapTransform))),p.transmission>0&&(m.transmission.value=p.transmission,m.transmissionSamplerMap.value=w.texture,m.transmissionSamplerSize.value.set(w.width,w.height),p.transmissionMap&&(m.transmissionMap.value=p.transmissionMap,t(p.transmissionMap,m.transmissionMapTransform)),m.thickness.value=p.thickness,p.thicknessMap&&(m.thicknessMap.value=p.thicknessMap,t(p.thicknessMap,m.thicknessMapTransform)),m.attenuationDistance.value=p.attenuationDistance,m.attenuationColor.value.copy(p.attenuationColor)),p.anisotropy>0&&(m.anisotropyVector.value.set(p.anisotropy*Math.cos(p.anisotropyRotation),p.anisotropy*Math.sin(p.anisotropyRotation)),p.anisotropyMap&&(m.anisotropyMap.value=p.anisotropyMap,t(p.anisotropyMap,m.anisotropyMapTransform))),m.specularIntensity.value=p.specularIntensity,m.specularColor.value.copy(p.specularColor),p.specularColorMap&&(m.specularColorMap.value=p.specularColorMap,t(p.specularColorMap,m.specularColorMapTransform)),p.specularIntensityMap&&(m.specularIntensityMap.value=p.specularIntensityMap,t(p.specularIntensityMap,m.specularIntensityMapTransform))}function g(m,p){p.matcap&&(m.matcap.value=p.matcap)}function y(m,p){let w=e.get(p).light;m.referencePosition.value.setFromMatrixPosition(w.matrixWorld),m.nearDistance.value=w.shadow.camera.near,m.farDistance.value=w.shadow.camera.far}return{refreshFogUniforms:n,refreshMaterialUniforms:s}}function wy(i,e,t,n){let s={},r={},a=[],o=i.getParameter(i.MAX_UNIFORM_BUFFER_BINDINGS);function l(M,E){let S=E.program;n.uniformBlockBinding(M,S)}function c(M,E){let S=s[M.id];S===void 0&&(m(M),S=h(M),s[M.id]=S,M.addEventListener("dispose",w));let C=E.program;n.updateUBOMapping(M,C);let x=e.render.frame;r[M.id]!==x&&(u(M),r[M.id]=x)}function h(M){let E=d();M.__bindingPointIndex=E;let S=i.createBuffer(),C=M.__size,x=M.usage;return i.bindBuffer(i.UNIFORM_BUFFER,S),i.bufferData(i.UNIFORM_BUFFER,C,x),i.bindBuffer(i.UNIFORM_BUFFER,null),i.bindBufferBase(i.UNIFORM_BUFFER,E,S),S}function d(){for(let M=0;M<o;M++)if(a.indexOf(M)===-1)return a.push(M),M;return Ve("WebGLRenderer: Maximum number of simultaneously usable uniforms groups reached."),0}function u(M){let E=s[M.id],S=M.uniforms,C=M.__cache;i.bindBuffer(i.UNIFORM_BUFFER,E);for(let x=0,T=S.length;x<T;x++){let I=S[x];if(Array.isArray(I))for(let U=0,O=I.length;U<O;U++)f(I[U],x,U,C);else f(I,x,0,C)}i.bindBuffer(i.UNIFORM_BUFFER,null)}function f(M,E,S,C){if(y(M,E,S,C)===!0){let x=M.__offset,T=M.value;if(Array.isArray(T)){let I=0;for(let U=0;U<T.length;U++){let O=T[U],H=p(O);g(O,M.__data,I),typeof O!="number"&&typeof O!="boolean"&&!O.isMatrix3&&!ArrayBuffer.isView(O)&&(I+=H.storage/Float32Array.BYTES_PER_ELEMENT)}}else g(T,M.__data,0);i.bufferSubData(i.UNIFORM_BUFFER,x,M.__data)}}function g(M,E,S){typeof M=="number"||typeof M=="boolean"?E[0]=M:M.isMatrix3?(E[0]=M.elements[0],E[1]=M.elements[1],E[2]=M.elements[2],E[3]=0,E[4]=M.elements[3],E[5]=M.elements[4],E[6]=M.elements[5],E[7]=0,E[8]=M.elements[6],E[9]=M.elements[7],E[10]=M.elements[8],E[11]=0):ArrayBuffer.isView(M)?E.set(new M.constructor(M.buffer,M.byteOffset,E.length)):M.toArray(E,S)}function y(M,E,S,C){let x=M.value,T=E+"_"+S;if(C[T]===void 0)return typeof x=="number"||typeof x=="boolean"?C[T]=x:ArrayBuffer.isView(x)?C[T]=x.slice():C[T]=x.clone(),!0;{let I=C[T];if(typeof x=="number"||typeof x=="boolean"){if(I!==x)return C[T]=x,!0}else{if(ArrayBuffer.isView(x))return!0;if(I.equals(x)===!1)return I.copy(x),!0}}return!1}function m(M){let E=M.uniforms,S=0,C=16;for(let T=0,I=E.length;T<I;T++){let U=Array.isArray(E[T])?E[T]:[E[T]];for(let O=0,H=U.length;O<H;O++){let D=U[O],B=Array.isArray(D.value)?D.value:[D.value];for(let X=0,W=B.length;X<W;X++){let ie=B[X],q=p(ie),K=S%C,te=K%q.boundary,Ie=K+te;S+=te,Ie!==0&&C-Ie<q.storage&&(S+=C-Ie),D.__data=new Float32Array(q.storage/Float32Array.BYTES_PER_ELEMENT),D.__offset=S,S+=q.storage}}}let x=S%C;return x>0&&(S+=C-x),M.__size=S,M.__cache={},this}function p(M){let E={boundary:0,storage:0};return typeof M=="number"||typeof M=="boolean"?(E.boundary=4,E.storage=4):M.isVector2?(E.boundary=8,E.storage=8):M.isVector3||M.isColor?(E.boundary=16,E.storage=12):M.isVector4?(E.boundary=16,E.storage=16):M.isMatrix3?(E.boundary=48,E.storage=48):M.isMatrix4?(E.boundary=64,E.storage=64):M.isTexture?Oe("WebGLRenderer: Texture samplers can not be part of an uniforms group."):ArrayBuffer.isView(M)?(E.boundary=16,E.storage=M.byteLength):Oe("WebGLRenderer: Unsupported uniform value type.",M),E}function w(M){let E=M.target;E.removeEventListener("dispose",w);let S=a.indexOf(E.__bindingPointIndex);a.splice(S,1),i.deleteBuffer(s[E.id]),delete s[E.id],delete r[E.id]}function A(){for(let M in s)i.deleteBuffer(s[M]);a=[],s={},r={}}return{bind:l,update:c,dispose:A}}var Ay=new Uint16Array([12469,15057,12620,14925,13266,14620,13807,14376,14323,13990,14545,13625,14713,13328,14840,12882,14931,12528,14996,12233,15039,11829,15066,11525,15080,11295,15085,10976,15082,10705,15073,10495,13880,14564,13898,14542,13977,14430,14158,14124,14393,13732,14556,13410,14702,12996,14814,12596,14891,12291,14937,11834,14957,11489,14958,11194,14943,10803,14921,10506,14893,10278,14858,9960,14484,14039,14487,14025,14499,13941,14524,13740,14574,13468,14654,13106,14743,12678,14818,12344,14867,11893,14889,11509,14893,11180,14881,10751,14852,10428,14812,10128,14765,9754,14712,9466,14764,13480,14764,13475,14766,13440,14766,13347,14769,13070,14786,12713,14816,12387,14844,11957,14860,11549,14868,11215,14855,10751,14825,10403,14782,10044,14729,9651,14666,9352,14599,9029,14967,12835,14966,12831,14963,12804,14954,12723,14936,12564,14917,12347,14900,11958,14886,11569,14878,11247,14859,10765,14828,10401,14784,10011,14727,9600,14660,9289,14586,8893,14508,8533,15111,12234,15110,12234,15104,12216,15092,12156,15067,12010,15028,11776,14981,11500,14942,11205,14902,10752,14861,10393,14812,9991,14752,9570,14682,9252,14603,8808,14519,8445,14431,8145,15209,11449,15208,11451,15202,11451,15190,11438,15163,11384,15117,11274,15055,10979,14994,10648,14932,10343,14871,9936,14803,9532,14729,9218,14645,8742,14556,8381,14461,8020,14365,7603,15273,10603,15272,10607,15267,10619,15256,10631,15231,10614,15182,10535,15118,10389,15042,10167,14963,9787,14883,9447,14800,9115,14710,8665,14615,8318,14514,7911,14411,7507,14279,7198,15314,9675,15313,9683,15309,9712,15298,9759,15277,9797,15229,9773,15166,9668,15084,9487,14995,9274,14898,8910,14800,8539,14697,8234,14590,7790,14479,7409,14367,7067,14178,6621,15337,8619,15337,8631,15333,8677,15325,8769,15305,8871,15264,8940,15202,8909,15119,8775,15022,8565,14916,8328,14804,8009,14688,7614,14569,7287,14448,6888,14321,6483,14088,6171,15350,7402,15350,7419,15347,7480,15340,7613,15322,7804,15287,7973,15229,8057,15148,8012,15046,7846,14933,7611,14810,7357,14682,7069,14552,6656,14421,6316,14251,5948,14007,5528,15356,5942,15356,5977,15353,6119,15348,6294,15332,6551,15302,6824,15249,7044,15171,7122,15070,7050,14949,6861,14818,6611,14679,6349,14538,6067,14398,5651,14189,5311,13935,4958,15359,4123,15359,4153,15356,4296,15353,4646,15338,5160,15311,5508,15263,5829,15188,6042,15088,6094,14966,6001,14826,5796,14678,5543,14527,5287,14377,4985,14133,4586,13869,4257,15360,1563,15360,1642,15358,2076,15354,2636,15341,3350,15317,4019,15273,4429,15203,4732,15105,4911,14981,4932,14836,4818,14679,4621,14517,4386,14359,4156,14083,3795,13808,3437,15360,122,15360,137,15358,285,15355,636,15344,1274,15322,2177,15281,2765,15215,3223,15120,3451,14995,3569,14846,3567,14681,3466,14511,3305,14344,3121,14037,2800,13753,2467,15360,0,15360,1,15359,21,15355,89,15346,253,15325,479,15287,796,15225,1148,15133,1492,15008,1749,14856,1882,14685,1886,14506,1783,14324,1608,13996,1398,13702,1183]),qn=null;function Ry(){return qn===null&&(qn=new xr(Ay,16,16,Ii,Un),qn.name="DFG_LUT",qn.minFilter=Gt,qn.magFilter=Gt,qn.wrapS=kn,qn.wrapT=kn,qn.generateMipmaps=!1,qn.needsUpdate=!0),qn}var Al=class{constructor(e={}){let{canvas:t=Td(),context:n=null,depth:s=!0,stencil:r=!1,alpha:a=!1,antialias:o=!1,premultipliedAlpha:l=!0,preserveDrawingBuffer:c=!1,powerPreference:h="default",failIfMajorPerformanceCaveat:d=!1,reversedDepthBuffer:u=!1,outputBufferType:f=rn}=e;this.isWebGLRenderer=!0;let g;if(n!==null){if(typeof WebGLRenderingContext<"u"&&n instanceof WebGLRenderingContext)throw new Error("THREE.WebGLRenderer: WebGL 1 is not supported since r163.");g=n.getContextAttributes().alpha}else g=a;let y=f,m=new Set([Xo,Wo,Go]),p=new Set([rn,Nn,zs,ks,ko,Vo]),w=new Uint32Array(4),A=new Int32Array(4),M=new R,E=null,S=null,C=[],x=[],T=null;this.domElement=t,this.debug={checkShaderErrors:!0,diagnostics:{keywords:!1},onShaderError:null},this.autoClear=!0,this.autoClearColor=!0,this.autoClearDepth=!0,this.autoClearStencil=!0,this.sortObjects=!0,this.clippingPlanes=[],this.localClippingEnabled=!1,this.toneMapping=Dn,this.toneMappingExposure=1,this.transmissionResolutionScale=1;let I=this,U=!1,O=null,H=null,D=null,B=null;this._outputColorSpace=Vt;let X=0,W=0,ie=null,q=-1,K=null,te=new St,Ie=new St,we=null,ot=new Ye(0),et=0,rt=t.width,$=t.height,Q=1,xe=null,He=null,Se=new St(0,0,rt,$),Ge=new St(0,0,rt,$),pt=!1,ee=new Rs,se=!1,oe=!1,le=new dt,ue=new R,ze=new St,Ue={background:null,fog:null,environment:null,overrideMaterial:null,isScene:!0},We=!1;function qe(){return ie===null?Q:1}let P=n;function lt(v,L){return t.getContext(v,L)}let tt,b,_,F,V,Y,ce,he,Z,j,de,Le,ge,fe,De,ke,Ze,N,pe,J,me,be,ne;try{let v={alpha:!0,depth:s,stencil:r,antialias:o,premultipliedAlpha:l,preserveDrawingBuffer:c,powerPreference:h,failIfMajorPerformanceCaveat:d};if("setAttribute"in t&&t.setAttribute("data-engine",`three.js r${Do}`),t.addEventListener("webglcontextlost",yt,!1),t.addEventListener("webglcontextrestored",ct,!1),t.addEventListener("webglcontextcreationerror",wn,!1),P===null){let L="webgl2";if(P=lt(L,v),P===null)throw lt(L)?new Error("THREE.WebGLRenderer: Error creating WebGL context with your selected attributes."):new Error("THREE.WebGLRenderer: Error creating WebGL context.")}Ne()}catch(v){throw t.removeEventListener("webglcontextlost",yt,!1),t.removeEventListener("webglcontextrestored",ct,!1),t.removeEventListener("webglcontextcreationerror",wn,!1),Ve("WebGLRenderer: "+v.message),v}function Ne(){tt=new U_(P),tt.init(),me=new My(P,tt),b=new T_(P,tt,e,me),_=new yy(P,tt),b.reversedDepthBuffer&&u&&_.buffers.depth.setReversed(!0),H=P.createFramebuffer(),D=P.createFramebuffer(),B=P.createFramebuffer(),F=new B_(P),V=new ry,Y=new vy(P,tt,_,V,b,me,F),ce=new N_(I),he=new km(P),be=new S_(P,he),Z=new F_(P,he,F,be),j=new k_(P,Z,he,be,F),N=new z_(P,b,Y),De=new w_(V),de=new sy(I,ce,tt,b,be,De),Le=new Ty(I,V),ge=new oy,fe=new fy(tt),Ze=new b_(I,ce,_,j,g,l),ke=new xy(I,j,b),ne=new wy(P,F,b,_),pe=new E_(P,tt,F),J=new O_(P,tt,F),F.programs=de.programs,I.capabilities=b,I.extensions=tt,I.properties=V,I.renderLists=ge,I.shadowMap=ke,I.state=_,I.info=F}y!==rn&&(T=new H_(y,t.width,t.height,o,s,r));let Ce=new Th(I,P);this.xr=Ce,this.getContext=function(){return P},this.getContextAttributes=function(){return P.getContextAttributes()},this.forceContextLoss=function(){let v=tt.get("WEBGL_lose_context");v&&v.loseContext()},this.forceContextRestore=function(){let v=tt.get("WEBGL_lose_context");v&&v.restoreContext()},this.getPixelRatio=function(){return Q},this.setPixelRatio=function(v){v!==void 0&&(Q=v,this.setSize(rt,$,!1))},this.getSize=function(v){return v.set(rt,$)},this.setSize=function(v,L,G=!0){if(Ce.isPresenting){Oe("WebGLRenderer: Can't change size while VR device is presenting.");return}rt=v,$=L,t.width=Math.floor(v*Q),t.height=Math.floor(L*Q),G===!0&&(t.style.width=v+"px",t.style.height=L+"px"),T!==null&&T.setSize(t.width,t.height),this.setViewport(0,0,v,L)},this.getDrawingBufferSize=function(v){return v.set(rt*Q,$*Q).floor()},this.setDrawingBufferSize=function(v,L,G){rt=v,$=L,Q=G,t.width=Math.floor(v*G),t.height=Math.floor(L*G),this.setViewport(0,0,v,L)},this.setEffects=function(v){if(y===rn){Ve("WebGLRenderer: setEffects() requires outputBufferType set to HalfFloatType or FloatType.");return}if(v){for(let L=0;L<v.length;L++)if(v[L].isOutputPass===!0){Oe("WebGLRenderer: OutputPass is not needed in setEffects(). Tone mapping and color space conversion are applied automatically.");break}}T.setEffects(v||[])},this.getCurrentViewport=function(v){return v.copy(te)},this.getViewport=function(v){return v.copy(Se)},this.setViewport=function(v,L,G,z){v.isVector4?Se.set(v.x,v.y,v.z,v.w):Se.set(v,L,G,z),_.viewport(te.copy(Se).multiplyScalar(Q).round())},this.getScissor=function(v){return v.copy(Ge)},this.setScissor=function(v,L,G,z){v.isVector4?Ge.set(v.x,v.y,v.z,v.w):Ge.set(v,L,G,z),_.scissor(Ie.copy(Ge).multiplyScalar(Q).round())},this.getScissorTest=function(){return pt},this.setScissorTest=function(v){_.setScissorTest(pt=v)},this.setOpaqueSort=function(v){xe=v},this.setTransparentSort=function(v){He=v},this.getClearColor=function(v){return v.copy(Ze.getClearColor())},this.setClearColor=function(){Ze.setClearColor(...arguments)},this.getClearAlpha=function(){return Ze.getClearAlpha()},this.setClearAlpha=function(){Ze.setClearAlpha(...arguments)},this.clear=function(v=!0,L=!0,G=!0){let z=0;if(v){let k=!1;if(ie!==null){let ve=ie.texture.format;k=m.has(ve)}if(k){let ve=ie.texture.type,Te=p.has(ve),ye=Ze.getClearColor(),Ae=Ze.getClearAlpha(),Pe=ye.r,Je=ye.g,nt=ye.b;Te?(w[0]=Pe,w[1]=Je,w[2]=nt,w[3]=Ae,P.clearBufferuiv(P.COLOR,0,w)):(A[0]=Pe,A[1]=Je,A[2]=nt,A[3]=Ae,P.clearBufferiv(P.COLOR,0,A))}else z|=P.COLOR_BUFFER_BIT}L&&(z|=P.DEPTH_BUFFER_BIT,this.state.buffers.depth.setMask(!0)),G&&(z|=P.STENCIL_BUFFER_BIT,this.state.buffers.stencil.setMask(4294967295)),z!==0&&P.clear(z)},this.clearColor=function(){this.clear(!0,!1,!1)},this.clearDepth=function(){this.clear(!1,!0,!1)},this.clearStencil=function(){this.clear(!1,!1,!0)},this.setNodesHandler=function(v){v.setRenderer(this),O=v},this.dispose=function(){t.removeEventListener("webglcontextlost",yt,!1),t.removeEventListener("webglcontextrestored",ct,!1),t.removeEventListener("webglcontextcreationerror",wn,!1),Ze.dispose(),ge.dispose(),fe.dispose(),V.dispose(),ce.dispose(),j.dispose(),be.dispose(),ne.dispose(),de.dispose(),Ce.dispose(),Ce.removeEventListener("sessionstart",Qh),Ce.removeEventListener("sessionend",eu),Ui.stop()};function yt(v){v.preventDefault(),Qc("WebGLRenderer: Context Lost."),U=!0}function ct(){Qc("WebGLRenderer: Context Restored."),U=!1;let v=F.autoReset,L=ke.enabled,G=ke.autoUpdate,z=ke.needsUpdate,k=ke.type;Ne(),F.autoReset=v,ke.enabled=L,ke.autoUpdate=G,ke.needsUpdate=z,ke.type=k}function wn(v){Ve("WebGLRenderer: A WebGL context could not be created. Reason: ",v.statusMessage)}function On(v){let L=v.target;L.removeEventListener("dispose",On),lp(L)}function lp(v){cp(v),V.remove(v)}function cp(v){let L=V.get(v).programs;L!==void 0&&(L.forEach(function(G){de.releaseProgram(G)}),v.isShaderMaterial&&de.releaseShaderCache(v))}this.renderBufferDirect=function(v,L,G,z,k,ve){L===null&&(L=Ue);let Te=k.isMesh&&k.matrixWorld.determinantAffine()<0,ye=dp(v,L,G,z,k);_.setMaterial(z,Te);let Ae=G.index,Pe=1;if(z.wireframe===!0){if(Ae=Z.getWireframeAttribute(G),Ae===void 0)return;Pe=2}let Je=G.drawRange,nt=G.attributes.position,Re=Je.start*Pe,ht=(Je.start+Je.count)*Pe;ve!==null&&(Re=Math.max(Re,ve.start*Pe),ht=Math.min(ht,(ve.start+ve.count)*Pe)),Ae!==null?(Re=Math.max(Re,0),ht=Math.min(ht,Ae.count)):nt!=null&&(Re=Math.max(Re,0),ht=Math.min(ht,nt.count));let Ct=ht-Re;if(Ct<0||Ct===1/0)return;be.setup(k,z,ye,G,Ae);let Mt,xt=pe;if(Ae!==null&&(Mt=he.get(Ae),xt=J,xt.setIndex(Mt)),k.isMesh)z.wireframe===!0?(_.setLineWidth(z.wireframeLinewidth*qe()),xt.setMode(P.LINES)):xt.setMode(P.TRIANGLES);else if(k.isLine){let Yt=z.linewidth;Yt===void 0&&(Yt=1),_.setLineWidth(Yt*qe()),k.isLineSegments?xt.setMode(P.LINES):k.isLineLoop?xt.setMode(P.LINE_LOOP):xt.setMode(P.LINE_STRIP)}else k.isPoints?xt.setMode(P.POINTS):k.isSprite&&xt.setMode(P.TRIANGLES);if(k.isBatchedMesh)if(tt.get("WEBGL_multi_draw"))xt.renderMultiDraw(k._multiDrawStarts,k._multiDrawCounts,k._multiDrawCount);else{let Yt=k._multiDrawStarts,Ee=k._multiDrawCounts,Kt=k._multiDrawCount,at=Ae?he.get(Ae).bytesPerElement:1,_n=V.get(z).currentProgram.getUniforms();for(let Bn=0;Bn<Kt;Bn++)_n.setValue(P,"_gl_DrawID",Bn),xt.render(Yt[Bn]/at,Ee[Bn])}else if(k.isInstancedMesh)xt.renderInstances(Re,Ct,k.count);else if(G.isInstancedBufferGeometry){let Yt=G._maxInstanceCount!==void 0?G._maxInstanceCount:1/0,Ee=Math.min(G.instanceCount,Yt);xt.renderInstances(Re,Ct,Ee)}else xt.render(Re,Ct)};function Kh(v,L,G,z){O!==null&&v.isNodeMaterial&&O.setObject(z,v),se===!0&&De.setState(v,G,!1),v.transparent===!0&&v.side===Wn&&v.forceSinglePass===!1?(v.side=Xt,v.needsUpdate=!0,ma(v,L,z),v.side=Ai,v.needsUpdate=!0,ma(v,L,z),v.side=Wn):ma(v,L,z)}this.compile=function(v,L,G=null){G===null&&(G=v),O!==null&&O.renderStart(v,L,G),S=fe.get(G),S.init(L),x.push(S),G.traverseVisible(function(k){k.isLight&&k.layers.test(L.layers)&&(S.pushLight(k),k.castShadow&&S.pushShadow(k))}),v!==G&&v.traverseVisible(function(k){k.isLight&&k.layers.test(L.layers)&&(S.pushLight(k),k.castShadow&&S.pushShadow(k))}),S.setupLights(),O!==null&&O.updateLights(S.state.lightsArray),oe=this.localClippingEnabled,se=De.init(this.clippingPlanes,oe),se===!0&&De.setGlobalState(this.clippingPlanes,L),O!==null&&ke.render(S.state.shadowsArray,G,L);let z=new Set;return v.traverse(function(k){if(!(k.isMesh||k.isPoints||k.isLine||k.isSprite))return;let ve=k.material;if(ve)if(Array.isArray(ve))for(let Te=0;Te<ve.length;Te++){let ye=ve[Te];Kh(ye,G,L,k),z.add(ye)}else Kh(ve,G,L,k),z.add(ve)}),S=x.pop(),O!==null&&O.renderEnd(),z},this.compileAsync=function(v,L,G=null){let z=this.compile(v,L,G);return new Promise(k=>{function ve(){if(z.forEach(function(Te){let Ae=V.get(Te).currentProgram;(Ae===void 0||Ae.isReady())&&z.delete(Te)}),z.size===0){k(v);return}setTimeout(ve,10)}tt.get("KHR_parallel_shader_compile")!==null?ve():setTimeout(ve,10)})};let Xl=null;function hp(v){Xl&&Xl(v)}function Qh(){Ui.stop()}function eu(){Ui.start()}let Ui=new rf;Ui.setAnimationLoop(hp),typeof self<"u"&&Ui.setContext(self),this.setAnimationLoop=function(v){Xl=v,Ce.setAnimationLoop(v),v===null?Ui.stop():Ui.start()},Ce.addEventListener("sessionstart",Qh),Ce.addEventListener("sessionend",eu),this.render=function(v,L){if(L!==void 0&&L.isCamera!==!0){Ve("WebGLRenderer.render: camera is not an instance of THREE.Camera.");return}if(U===!0)return;O!==null&&O.renderStart(v,L);let G=Ce.enabled===!0&&Ce.isPresenting===!0,z=T!==null&&(ie===null||G)&&T.begin(I,ie);if(v.matrixWorldAutoUpdate===!0&&v.updateMatrixWorld(),L.parent===null&&L.matrixWorldAutoUpdate===!0&&L.updateMatrixWorld(),Ce.enabled===!0&&Ce.isPresenting===!0&&(T===null||T.isCompositing()===!1)&&(Ce.cameraAutoUpdate===!0&&Ce.updateCamera(L),L=Ce.getCamera()),v.isScene===!0&&v.onBeforeRender(I,v,L,ie),S=fe.get(v,x.length),S.init(L),S.state.textureUnits=Y.getTextureUnits(),x.push(S),le.multiplyMatrices(L.projectionMatrix,L.matrixWorldInverse),ee.setFromProjectionMatrix(le,Pn,L.reversedDepth),oe=this.localClippingEnabled,se=De.init(this.clippingPlanes,oe),E=ge.get(v,C.length),E.init(),C.push(E),Ce.enabled===!0&&Ce.isPresenting===!0){let Te=I.xr.getDepthSensingMesh();Te!==null&&ql(Te,L,-1/0,I.sortObjects)}ql(v,L,0,I.sortObjects),E.finish(),O!==null&&O.updateLights(S.state.lightsArray),I.sortObjects===!0&&E.sort(xe,He),We=Ce.enabled===!1||Ce.isPresenting===!1||Ce.hasDepthSensing()===!1,We&&Ze.addToRenderList(E,v),this.info.render.frame++,this.info.autoReset===!0&&this.info.reset(),se===!0&&De.beginShadows();let k=S.state.shadowsArray;if(ke.render(k,v,L),se===!0&&De.endShadows(),(z&&T.hasRenderPass())===!1){let Te=E.opaque,ye=E.transmissive;if(S.setupLights(),L.isArrayCamera){let Ae=L.cameras;if(ye.length>0)for(let Pe=0,Je=Ae.length;Pe<Je;Pe++){let nt=Ae[Pe];nu(Te,ye,v,nt)}We&&Ze.render(v);for(let Pe=0,Je=Ae.length;Pe<Je;Pe++){let nt=Ae[Pe];tu(E,v,nt,nt.viewport)}}else ye.length>0&&nu(Te,ye,v,L),We&&Ze.render(v),tu(E,v,L)}ie!==null&&W===0&&(Y.updateMultisampleRenderTarget(ie),Y.updateRenderTargetMipmap(ie)),z&&T.end(I),v.isScene===!0&&v.onAfterRender(I,v,L),be.resetDefaultState(),q=-1,K=null,x.pop(),x.length>0?(S=x[x.length-1],Y.setTextureUnits(S.state.textureUnits),se===!0&&De.setGlobalState(I.clippingPlanes,S.state.camera)):S=null,C.pop(),C.length>0?E=C[C.length-1]:E=null,O!==null&&O.renderEnd()};function ql(v,L,G,z){if(v.visible===!1)return;if(v.layers.test(L.layers)){if(v.isGroup)G=v.renderOrder;else if(v.isLOD)v.autoUpdate===!0&&v.update(L);else if(v.isLightProbeGrid)S.pushLightProbeGrid(v);else if(v.isLight)S.pushLight(v),v.castShadow&&S.pushShadow(v);else if(v.isSprite){if(!v.frustumCulled||v.intersectsFrustum(ee)){z&&ze.setFromMatrixPosition(v.matrixWorld).applyMatrix4(le);let Te=j.update(v),ye=v.material;ye.visible&&E.push(v,Te,ye,G,ze.z,null,L)}}else if((v.isMesh||v.isLine||v.isPoints)&&(!v.frustumCulled||v.intersectsFrustum(ee))){let Te=j.update(v),ye=v.material;if(z&&(v.boundingSphere!==void 0?(v.boundingSphere===null&&v.computeBoundingSphere(),ze.copy(v.boundingSphere.center)):(Te.boundingSphere===null&&Te.computeBoundingSphere(),ze.copy(Te.boundingSphere.center)),ze.applyMatrix4(v.matrixWorld).applyMatrix4(le)),Array.isArray(ye)){let Ae=Te.groups;for(let Pe=0,Je=Ae.length;Pe<Je;Pe++){let nt=Ae[Pe],Re=ye[nt.materialIndex];Re&&Re.visible&&E.push(v,Te,Re,G,ze.z,nt,L)}}else ye.visible&&E.push(v,Te,ye,G,ze.z,null,L)}}let ve=v.children;for(let Te=0,ye=ve.length;Te<ye;Te++)ql(ve[Te],L,G,z)}function tu(v,L,G,z){let{opaque:k,transmissive:ve,transparent:Te}=v;S.setupLightsView(G),se===!0&&De.setGlobalState(I.clippingPlanes,G),z&&_.viewport(te.copy(z)),k.length>0&&pa(k,L,G),ve.length>0&&pa(ve,L,G),Te.length>0&&pa(Te,L,G),_.buffers.depth.setTest(!0),_.buffers.depth.setMask(!0),_.buffers.color.setMask(!0),_.setPolygonOffset(!1)}function nu(v,L,G,z){if((G.isScene===!0?G.overrideMaterial:null)!==null)return;if(S.state.transmissionRenderTarget[z.id]===void 0){let Re=tt.has("EXT_color_buffer_half_float")||tt.has("EXT_color_buffer_float");S.state.transmissionRenderTarget[z.id]=new sn(1,1,{generateMipmaps:!0,type:Re?Un:rn,minFilter:Ci,samples:Math.max(4,b.samples),stencilBuffer:r,resolveDepthBuffer:!1,resolveStencilBuffer:!1,storeMultisampledDepthBuffer:!1,storeMultisampledStencilBuffer:!1,colorSpace:it.workingColorSpace})}let ve=S.state.transmissionRenderTarget[z.id],Te=z.viewport||te;ve.setSize(Te.z*I.transmissionResolutionScale,Te.w*I.transmissionResolutionScale);let ye=I.getRenderTarget(),Ae=I.getActiveCubeFace(),Pe=I.getActiveMipmapLevel();I.setRenderTarget(ve),I.getClearColor(ot),et=I.getClearAlpha(),et<1&&I.setClearColor(16777215,.5),I.clear(),We&&Ze.render(G);let Je=I.toneMapping;I.toneMapping=Dn;let nt=z.viewport;if(z.viewport!==void 0&&(z.viewport=void 0),S.setupLightsView(z),se===!0&&De.setGlobalState(I.clippingPlanes,z),pa(v,G,z),Y.updateMultisampleRenderTarget(ve),Y.updateRenderTargetMipmap(ve),tt.has("WEBGL_multisampled_render_to_texture")===!1){let Re=!1;for(let ht=0,Ct=L.length;ht<Ct;ht++){let Mt=L[ht],{object:xt,geometry:Yt,material:Ee,group:Kt}=Mt;if(Ee.side===Wn&&xt.layers.test(z.layers)){let at=Ee.side;Ee.side=Xt,Ee.needsUpdate=!0,iu(xt,G,z,Yt,Ee,Kt),Ee.side=at,Ee.needsUpdate=!0,Re=!0}}Re===!0&&(Y.updateMultisampleRenderTarget(ve),Y.updateRenderTargetMipmap(ve))}I.setRenderTarget(ye,Ae,Pe),I.setClearColor(ot,et),nt!==void 0&&(z.viewport=nt),I.toneMapping=Je}function pa(v,L,G){let z=L.isScene===!0?L.overrideMaterial:null;for(let k=0,ve=v.length;k<ve;k++){let Te=v[k],{object:ye,geometry:Ae,group:Pe}=Te,Je=Te.material;Je.allowOverride===!0&&z!==null&&(Je=z),ye.layers.test(G.layers)&&iu(ye,L,G,Ae,Je,Pe)}}function iu(v,L,G,z,k,ve){O!==null&&k.isNodeMaterial&&O.setObject(v,k),v.onBeforeRender(I,L,G,z,k,ve),v.modelViewMatrix.multiplyMatrices(G.matrixWorldInverse,v.matrixWorld),v.normalMatrix.getNormalMatrix(v.modelViewMatrix),k.onBeforeRender(I,L,G,z,v,ve),k.transparent===!0&&k.side===Wn&&k.forceSinglePass===!1?(k.side=Xt,k.needsUpdate=!0,I.renderBufferDirect(G,L,z,k,v,ve),k.side=Ai,k.needsUpdate=!0,I.renderBufferDirect(G,L,z,k,v,ve),k.side=Wn):I.renderBufferDirect(G,L,z,k,v,ve),v.onAfterRender(I,L,G,z,k,ve)}function ma(v,L,G){L.isScene!==!0&&(L=Ue);let z=V.get(v),k=S.state.lights,ve=S.state.shadowsArray,Te=k.state.version,ye=de.getParameters(v,k.state,ve,L,G,S.state.lightProbeGridArray),Ae=de.getProgramCacheKey(ye),Pe=z.programs;z.environment=v.isMeshStandardMaterial||v.isMeshLambertMaterial||v.isMeshPhongMaterial?L.environment:null,z.fog=L.fog;let Je=v.isMeshStandardMaterial||v.isMeshLambertMaterial&&!v.envMap||v.isMeshPhongMaterial&&!v.envMap;z.envMap=ce.get(v.envMap||z.environment,Je),z.envMapRotation=z.environment!==null&&v.envMap===null?L.environmentRotation:v.envMapRotation,Pe===void 0&&(v.addEventListener("dispose",On),Pe=new Map,z.programs=Pe);let nt=Pe.get(Ae);if(nt!==void 0){if(z.currentProgram===nt&&z.lightsStateVersion===Te)return ru(v,ye),nt}else ye.uniforms=de.getUniforms(v),O!==null&&v.isNodeMaterial&&O.build(v,G,ye),v.onBeforeCompile(ye,I),nt=de.acquireProgram(ye,Ae),Pe.set(Ae,nt),z.uniforms=ye.uniforms;let Re=z.uniforms;return(!v.isShaderMaterial&&!v.isRawShaderMaterial||v.clipping===!0)&&(Re.clippingPlanes=De.uniform),ru(v,ye),z.needsLights=pp(v),z.lightsStateVersion=Te,z.needsLights&&(Re.ambientLightColor.value=k.state.ambient,Re.lightProbe.value=k.state.probe,Re.sunLights.value=k.state.sun,Re.sunLightShadows.value=k.state.sunShadow,Re.directionalLights.value=k.state.directional,Re.directionalLightShadows.value=k.state.directionalShadow,Re.spotLights.value=k.state.spot,Re.spotLightShadows.value=k.state.spotShadow,Re.rectAreaLights.value=k.state.rectArea,Re.ltc_1.value=k.state.rectAreaLTC1,Re.ltc_2.value=k.state.rectAreaLTC2,Re.pointLights.value=k.state.point,Re.pointLightShadows.value=k.state.pointShadow,Re.hemisphereLights.value=k.state.hemi,Re.sunShadowMatrix.value=k.state.sunShadowMatrix,Re.sunShadowCascade.value=k.state.sunShadowCascade,Re.directionalShadowMatrix.value=k.state.directionalShadowMatrix,Re.spotLightMatrix.value=k.state.spotLightMatrix,Re.spotLightMap.value=k.state.spotLightMap,Re.pointShadowMatrix.value=k.state.pointShadowMatrix),z.lightProbeGrid=S.state.lightProbeGridArray.length>0,z.currentProgram=nt,z.uniformsList=null,nt}function su(v){if(v.uniformsList===null){let L=v.currentProgram.getUniforms();v.uniformsList=Gs.seqWithValue(L.seq,v.uniforms)}return v.uniformsList}function ru(v,L){let G=V.get(v);G.outputColorSpace=L.outputColorSpace,G.batching=L.batching,G.batchingColor=L.batchingColor,G.instancing=L.instancing,G.instancingColor=L.instancingColor,G.instancingMorph=L.instancingMorph,G.skinning=L.skinning,G.morphTargets=L.morphTargets,G.morphNormals=L.morphNormals,G.morphColors=L.morphColors,G.morphTargetsCount=L.morphTargetsCount,G.numClippingPlanes=L.numClippingPlanes,G.numIntersection=L.numClipIntersection,G.vertexAlphas=L.vertexAlphas,G.vertexTangents=L.vertexTangents,G.toneMapping=L.toneMapping}function up(v,L){if(v.length===0)return null;if(v.length===1)return v[0].texture!==null?v[0]:null;M.setFromMatrixPosition(L.matrixWorld);for(let G=0,z=v.length;G<z;G++){let k=v[G];if(k.texture!==null&&k.boundingBox.containsPoint(M))return k}return null}function dp(v,L,G,z,k){L.isScene!==!0&&(L=Ue),Y.resetTextureUnits();let ve=L.fog,Te=z.isMeshStandardMaterial||z.isMeshLambertMaterial||z.isMeshPhongMaterial?L.environment:null,ye=ie===null?I.outputColorSpace:ie.isXRRenderTarget===!0?ie.texture.colorSpace:it.workingColorSpace,Ae=z.isMeshStandardMaterial||z.isMeshLambertMaterial&&!z.envMap||z.isMeshPhongMaterial&&!z.envMap,Pe=ce.get(z.envMap||Te,Ae),Je=z.vertexColors===!0&&!!G.attributes.color&&G.attributes.color.itemSize===4,nt=!!G.attributes.tangent&&(!!z.normalMap||z.anisotropy>0),Re=!!G.morphAttributes.position,ht=!!G.morphAttributes.normal,Ct=!!G.morphAttributes.color,Mt=Dn;z.toneMapped&&(ie===null||ie.isXRRenderTarget===!0)&&(Mt=I.toneMapping);let xt=G.morphAttributes.position||G.morphAttributes.normal||G.morphAttributes.color,Yt=xt!==void 0?xt.length:0,Ee=V.get(z),Kt=S.state.lights;if(se===!0&&(oe===!0||v!==K)){let vt=v===K&&z.id===q;De.setState(z,v,vt)}let at=!1;z.version===Ee.__version?(Ee.needsLights&&Ee.lightsStateVersion!==Kt.state.version||Ee.outputColorSpace!==ye||k.isBatchedMesh&&Ee.batching===!1||!k.isBatchedMesh&&Ee.batching===!0||k.isBatchedMesh&&Ee.batchingColor===!0&&k._colorsTexture===null||k.isBatchedMesh&&Ee.batchingColor===!1&&k._colorsTexture!==null||k.isInstancedMesh&&Ee.instancing===!1||!k.isInstancedMesh&&Ee.instancing===!0||k.isSkinnedMesh&&Ee.skinning===!1||!k.isSkinnedMesh&&Ee.skinning===!0||k.isInstancedMesh&&Ee.instancingColor===!0&&k.instanceColor===null||k.isInstancedMesh&&Ee.instancingColor===!1&&k.instanceColor!==null||k.isInstancedMesh&&Ee.instancingMorph===!0&&k.morphTexture===null||k.isInstancedMesh&&Ee.instancingMorph===!1&&k.morphTexture!==null||Ee.envMap!==Pe||z.fog===!0&&Ee.fog!==ve||Ee.numClippingPlanes!==void 0&&(Ee.numClippingPlanes!==De.numPlanes||Ee.numIntersection!==De.numIntersection)||Ee.vertexAlphas!==Je||Ee.vertexTangents!==nt||Ee.morphTargets!==Re||Ee.morphNormals!==ht||Ee.morphColors!==Ct||Ee.toneMapping!==Mt||Ee.morphTargetsCount!==Yt||!!Ee.lightProbeGrid!=S.state.lightProbeGridArray.length>0)&&(at=!0):(at=!0,Ee.__version=z.version);let _n=Ee.currentProgram;at===!0&&(_n=ma(z,L,k),O&&z.isNodeMaterial&&O.onUpdateProgram(z,_n,Ee));let Bn=!1,ui=!1,ts=!1,_t=_n.getUniforms(),At=Ee.uniforms;if(_.useProgram(_n.program)&&(Bn=!0,ui=!0,ts=!0),z.id!==q&&(q=z.id,ui=!0),Ee.needsLights){let vt=up(S.state.lightProbeGridArray,k);Ee.lightProbeGrid!==vt&&(Ee.lightProbeGrid=vt,ui=!0)}if(Bn||K!==v){_.buffers.depth.getReversed()&&v.reversedDepth!==!0&&(v._reversedDepth=!0,v.updateProjectionMatrix()),_t.setValue(P,"projectionMatrix",v.projectionMatrix),_t.setValue(P,"viewMatrix",v.matrixWorldInverse);let fi=_t.map.cameraPosition;fi!==void 0&&fi.setValue(P,ue.setFromMatrixPosition(v.matrixWorld)),b.logarithmicDepthBuffer&&_t.setValue(P,"logDepthBufFC",2/(Math.log(v.far+1)/Math.LN2)),(z.isMeshPhongMaterial||z.isMeshToonMaterial||z.isMeshLambertMaterial||z.isMeshBasicMaterial||z.isMeshStandardMaterial||z.isShaderMaterial)&&_t.setValue(P,"isOrthographic",v.isOrthographicCamera===!0),K!==v&&(K=v,ui=!0,ts=!0)}if(Ee.needsLights&&(Kt.state.sunShadowMap.length>0&&_t.setValue(P,"sunShadowMap",Kt.state.sunShadowMap,Y),Kt.state.directionalShadowMap.length>0&&_t.setValue(P,"directionalShadowMap",Kt.state.directionalShadowMap,Y),Kt.state.spotShadowMap.length>0&&_t.setValue(P,"spotShadowMap",Kt.state.spotShadowMap,Y),Kt.state.pointShadowMap.length>0&&_t.setValue(P,"pointShadowMap",Kt.state.pointShadowMap,Y)),k.isSkinnedMesh){_t.setOptional(P,k,"bindMatrix"),_t.setOptional(P,k,"bindMatrixInverse");let vt=k.skeleton;vt&&(vt.boneTexture===null&&vt.computeBoneTexture(),_t.setValue(P,"boneTexture",vt.boneTexture,Y))}k.isBatchedMesh&&(_t.setOptional(P,k,"batchingTexture"),_t.setValue(P,"batchingTexture",k._matricesTexture,Y),_t.setOptional(P,k,"batchingIdTexture"),_t.setValue(P,"batchingIdTexture",k._indirectTexture,Y),_t.setOptional(P,k,"batchingColorTexture"),k._colorsTexture!==null&&_t.setValue(P,"batchingColorTexture",k._colorsTexture,Y));let di=G.morphAttributes;if((di.position!==void 0||di.normal!==void 0||di.color!==void 0)&&N.update(k,G,_n),(ui||Ee.receiveShadow!==k.receiveShadow)&&(Ee.receiveShadow=k.receiveShadow,_t.setValue(P,"receiveShadow",k.receiveShadow)),(z.isMeshStandardMaterial||z.isMeshLambertMaterial||z.isMeshPhongMaterial)&&z.envMap===null&&L.environment!==null&&(At.envMapIntensity.value=L.environmentIntensity),At.dfgLUT!==void 0&&(At.dfgLUT.value=Ry()),ui){if(_t.setValue(P,"toneMappingExposure",I.toneMappingExposure),Ee.needsLights&&fp(At,ts),ve&&z.fog===!0&&Le.refreshFogUniforms(At,ve),Le.refreshMaterialUniforms(At,z,Q,$,S.state.transmissionRenderTarget[v.id]),Ee.needsLights&&Ee.lightProbeGrid){let vt=Ee.lightProbeGrid;At.probesSH.value=vt.texture,At.probesMin.value.copy(vt.boundingBox.min),At.probesMax.value.copy(vt.boundingBox.max),At.probesResolution.value.copy(vt.resolution)}Gs.upload(P,su(Ee),At,Y)}if(z.isShaderMaterial&&z.uniformsNeedUpdate===!0&&(Gs.upload(P,su(Ee),At,Y),z.uniformsNeedUpdate=!1),z.isSpriteMaterial&&_t.setValue(P,"center",k.center),_t.setValue(P,"modelViewMatrix",k.modelViewMatrix),_t.setValue(P,"normalMatrix",k.normalMatrix),_t.setValue(P,"modelMatrix",k.matrixWorld),z.uniformsGroups!==void 0){let vt=z.uniformsGroups;for(let fi=0,ns=vt.length;fi<ns;fi++){let ou=vt[fi];ne.update(ou,_n),ne.bind(ou,_n)}}return _n}function fp(v,L){v.ambientLightColor.needsUpdate=L,v.lightProbe.needsUpdate=L,v.sunLights.needsUpdate=L,v.sunLightShadows.needsUpdate=L,v.directionalLights.needsUpdate=L,v.directionalLightShadows.needsUpdate=L,v.pointLights.needsUpdate=L,v.pointLightShadows.needsUpdate=L,v.spotLights.needsUpdate=L,v.spotLightShadows.needsUpdate=L,v.rectAreaLights.needsUpdate=L,v.hemisphereLights.needsUpdate=L}function pp(v){return v.isMeshLambertMaterial||v.isMeshToonMaterial||v.isMeshPhongMaterial||v.isMeshStandardMaterial||v.isShadowMaterial||v.isShaderMaterial&&v.lights===!0}this.getActiveCubeFace=function(){return X},this.getActiveMipmapLevel=function(){return W},this.getRenderTarget=function(){return ie},this.setRenderTargetTextures=function(v,L,G){let z=V.get(v);z.__autoAllocateDepthBuffer=v.resolveDepthBuffer===!1,z.__autoAllocateDepthBuffer===!1&&(z.__useRenderToTexture=!1),V.get(v.texture).__webglTexture=L,V.get(v.depthTexture).__webglTexture=z.__autoAllocateDepthBuffer?void 0:G,z.__hasExternalTextures=!0},this.setRenderTargetFramebuffer=function(v,L){let G=V.get(v);G.__webglFramebuffer=L,G.__useDefaultFramebuffer=L===void 0},this.setRenderTarget=function(v,L=0,G=0){ie=v,X=L,W=G;let z=null,k=!1,ve=!1;if(v){let ye=V.get(v);if(ye.__useDefaultFramebuffer!==void 0){_.bindFramebuffer(P.FRAMEBUFFER,ye.__webglFramebuffer),te.copy(v.viewport),Ie.copy(v.scissor),we=v.scissorTest,_.viewport(te),_.scissor(Ie),_.setScissorTest(we),q=-1;return}else if(ye.__webglFramebuffer===void 0)Y.setupRenderTarget(v);else if(ye.__hasExternalTextures)Y.rebindTextures(v,V.get(v.texture).__webglTexture,V.get(v.depthTexture).__webglTexture);else if(v.depthBuffer){let Je=v.depthTexture;if(ye.__boundDepthTexture!==Je){if(Je!==null&&V.has(Je)&&(v.width!==Je.image.width||v.height!==Je.image.height))throw new Error("THREE.WebGLRenderer: Attached DepthTexture is initialized to the incorrect size.");Y.setupDepthRenderbuffer(v)}}let Ae=v.texture;(Ae.isData3DTexture||Ae.isDataArrayTexture||Ae.isCompressedArrayTexture)&&(ve=!0);let Pe=V.get(v).__webglFramebuffer;v.isWebGLCubeRenderTarget?(Array.isArray(Pe[L])?z=Pe[L][G]:z=Pe[L],k=!0):v.samples>0&&Y.useMultisampledRTT(v)===!1?z=V.get(v).__webglMultisampledFramebuffer:Array.isArray(Pe)?z=Pe[G]:z=Pe,te.copy(v.viewport),Ie.copy(v.scissor),we=v.scissorTest}else te.copy(Se).multiplyScalar(Q).floor(),Ie.copy(Ge).multiplyScalar(Q).floor(),we=pt;if(G!==0&&(z=H),_.bindFramebuffer(P.FRAMEBUFFER,z)&&_.drawBuffers(v,z),_.viewport(te),_.scissor(Ie),_.setScissorTest(we),k){let ye=V.get(v.texture);P.framebufferTexture2D(P.FRAMEBUFFER,P.COLOR_ATTACHMENT0,P.TEXTURE_CUBE_MAP_POSITIVE_X+L,ye.__webglTexture,G)}else if(ve){let ye=L;for(let Ae=0;Ae<v.textures.length;Ae++){let Pe=V.get(v.textures[Ae]);P.framebufferTextureLayer(P.FRAMEBUFFER,P.COLOR_ATTACHMENT0+Ae,Pe.__webglTexture,G,ye)}}else if(v!==null&&G!==0){let ye=V.get(v.texture);P.framebufferTexture2D(P.FRAMEBUFFER,P.COLOR_ATTACHMENT0,P.TEXTURE_2D,ye.__webglTexture,G)}q=-1};function au(v){let L=V.get(v);return(L.__readFormat!==v.format||L.__readType!==v.type)&&(L.__readFormat=v.format,L.__readType=v.type,L.__formatReadable=b.textureFormatReadable(v.format),L.__typeReadable=b.textureTypeReadable(v.type)),L}this.readRenderTargetPixels=function(v,L,G,z,k,ve,Te,ye=0){if(!(v&&v.isWebGLRenderTarget)){Ve("WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");return}let Ae=V.get(v).__webglFramebuffer;if(v.isWebGLCubeRenderTarget&&Te!==void 0&&(Ae=Ae[Te]),Ae){_.bindFramebuffer(P.FRAMEBUFFER,Ae);try{let Pe=v.textures[ye],Je=Pe.format,nt=Pe.type;v.textures.length>1&&P.readBuffer(P.COLOR_ATTACHMENT0+ye);let Re=au(Pe);if(Re.__formatReadable===!1){Ve("WebGLRenderer.readRenderTargetPixels: renderTarget is not in RGBA or implementation defined format.");return}if(Re.__typeReadable===!1){Ve("WebGLRenderer.readRenderTargetPixels: renderTarget is not in UnsignedByteType or implementation defined type.");return}L>=0&&L<=v.width-z&&G>=0&&G<=v.height-k&&P.readPixels(L,G,z,k,me.convert(Je),me.convert(nt),ve)}finally{let Pe=ie!==null?V.get(ie).__webglFramebuffer:null;_.bindFramebuffer(P.FRAMEBUFFER,Pe)}}},this.readRenderTargetPixelsAsync=async function(v,L,G,z,k,ve,Te,ye=0){if(!(v&&v.isWebGLRenderTarget))throw new Error("THREE.WebGLRenderer.readRenderTargetPixels: renderTarget is not THREE.WebGLRenderTarget.");let Ae=V.get(v).__webglFramebuffer;if(v.isWebGLCubeRenderTarget&&Te!==void 0&&(Ae=Ae[Te]),Ae)if(L>=0&&L<=v.width-z&&G>=0&&G<=v.height-k){_.bindFramebuffer(P.FRAMEBUFFER,Ae);let Pe=v.textures[ye],Je=Pe.format,nt=Pe.type;v.textures.length>1&&P.readBuffer(P.COLOR_ATTACHMENT0+ye);let Re=au(Pe);if(Re.__formatReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in RGBA or implementation defined format.");if(Re.__typeReadable===!1)throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: renderTarget is not in UnsignedByteType or implementation defined type.");let ht=P.createBuffer();P.bindBuffer(P.PIXEL_PACK_BUFFER,ht),P.bufferData(P.PIXEL_PACK_BUFFER,ve.byteLength,P.STREAM_READ),P.readPixels(L,G,z,k,me.convert(Je),me.convert(nt),0),P.bindBuffer(P.PIXEL_PACK_BUFFER,null);let Ct=ie!==null?V.get(ie).__webglFramebuffer:null;_.bindFramebuffer(P.FRAMEBUFFER,Ct);let Mt=P.fenceSync(P.SYNC_GPU_COMMANDS_COMPLETE,0);return P.flush(),await Ad(P,Mt,4),P.bindBuffer(P.PIXEL_PACK_BUFFER,ht),P.getBufferSubData(P.PIXEL_PACK_BUFFER,0,ve),P.bindBuffer(P.PIXEL_PACK_BUFFER,null),P.deleteBuffer(ht),P.deleteSync(Mt),ve}else throw new Error("THREE.WebGLRenderer.readRenderTargetPixelsAsync: requested read bounds are out of range.")},this.copyFramebufferToTexture=function(v,L=null,G=0){let z=Math.pow(2,-G),k=Math.floor(v.image.width*z),ve=Math.floor(v.image.height*z),Te=L!==null?L.x:0,ye=L!==null?L.y:0;Y.setTexture2D(v,0),P.copyTexSubImage2D(P.TEXTURE_2D,G,0,0,Te,ye,k,ve),_.unbindTexture()},this.copyTextureToTexture=function(v,L,G=null,z=null,k=0,ve=0){let Te,ye,Ae,Pe,Je,nt,Re,ht,Ct,Mt=v.isCompressedTexture?v.mipmaps[ve]:v.image;if(G!==null)Te=G.max.x-G.min.x,ye=G.max.y-G.min.y,Ae=G.isBox3?G.max.z-G.min.z:1,Pe=G.min.x,Je=G.min.y,nt=G.isBox3?G.min.z:0;else{let At=Math.pow(2,-k);Te=Math.floor(Mt.width*At),ye=Math.floor(Mt.height*At),v.isDataArrayTexture?Ae=Mt.depth:v.isData3DTexture?Ae=Math.floor(Mt.depth*At):Ae=1,Pe=0,Je=0,nt=0}z!==null?(Re=z.x,ht=z.y,Ct=z.z):(Re=0,ht=0,Ct=0);let xt=me.convert(L.format),Yt=me.convert(L.type),Ee;L.isData3DTexture?(Y.setTexture3D(L,0),Ee=P.TEXTURE_3D):L.isDataArrayTexture||L.isCompressedArrayTexture?(Y.setTexture2DArray(L,0),Ee=P.TEXTURE_2D_ARRAY):(Y.setTexture2D(L,0),Ee=P.TEXTURE_2D),_.activeTexture(P.TEXTURE0),_.pixelStorei(P.UNPACK_FLIP_Y_WEBGL,L.flipY),_.pixelStorei(P.UNPACK_PREMULTIPLY_ALPHA_WEBGL,L.premultiplyAlpha),_.pixelStorei(P.UNPACK_ALIGNMENT,L.unpackAlignment);let Kt=_.getParameter(P.UNPACK_ROW_LENGTH),at=_.getParameter(P.UNPACK_IMAGE_HEIGHT),_n=_.getParameter(P.UNPACK_SKIP_PIXELS),Bn=_.getParameter(P.UNPACK_SKIP_ROWS),ui=_.getParameter(P.UNPACK_SKIP_IMAGES);_.pixelStorei(P.UNPACK_ROW_LENGTH,Mt.width),_.pixelStorei(P.UNPACK_IMAGE_HEIGHT,Mt.height),_.pixelStorei(P.UNPACK_SKIP_PIXELS,Pe),_.pixelStorei(P.UNPACK_SKIP_ROWS,Je),_.pixelStorei(P.UNPACK_SKIP_IMAGES,nt);let ts=v.isDataArrayTexture||v.isData3DTexture,_t=L.isDataArrayTexture||L.isData3DTexture;if(v.isDepthTexture){let At=V.get(v),di=V.get(L),vt=V.get(At.__renderTarget),fi=V.get(di.__renderTarget);_.bindFramebuffer(P.READ_FRAMEBUFFER,vt.__webglFramebuffer),_.bindFramebuffer(P.DRAW_FRAMEBUFFER,fi.__webglFramebuffer);for(let ns=0;ns<Ae;ns++)ts&&(P.framebufferTextureLayer(P.READ_FRAMEBUFFER,P.COLOR_ATTACHMENT0,V.get(v).__webglTexture,k,nt+ns),P.framebufferTextureLayer(P.DRAW_FRAMEBUFFER,P.COLOR_ATTACHMENT0,V.get(L).__webglTexture,ve,Ct+ns)),P.blitFramebuffer(Pe,Je,Te,ye,Re,ht,Te,ye,P.DEPTH_BUFFER_BIT,P.NEAREST);_.bindFramebuffer(P.READ_FRAMEBUFFER,null),_.bindFramebuffer(P.DRAW_FRAMEBUFFER,null)}else if(k!==0||v.isRenderTargetTexture||V.has(v)){let At=V.get(v),di=V.get(L);_.bindFramebuffer(P.READ_FRAMEBUFFER,D),_.bindFramebuffer(P.DRAW_FRAMEBUFFER,B);for(let vt=0;vt<Ae;vt++)ts?P.framebufferTextureLayer(P.READ_FRAMEBUFFER,P.COLOR_ATTACHMENT0,At.__webglTexture,k,nt+vt):P.framebufferTexture2D(P.READ_FRAMEBUFFER,P.COLOR_ATTACHMENT0,P.TEXTURE_2D,At.__webglTexture,k),_t?P.framebufferTextureLayer(P.DRAW_FRAMEBUFFER,P.COLOR_ATTACHMENT0,di.__webglTexture,ve,Ct+vt):P.framebufferTexture2D(P.DRAW_FRAMEBUFFER,P.COLOR_ATTACHMENT0,P.TEXTURE_2D,di.__webglTexture,ve),k!==0?P.blitFramebuffer(Pe,Je,Te,ye,Re,ht,Te,ye,P.COLOR_BUFFER_BIT,P.NEAREST):_t?P.copyTexSubImage3D(Ee,ve,Re,ht,Ct+vt,Pe,Je,Te,ye):P.copyTexSubImage2D(Ee,ve,Re,ht,Pe,Je,Te,ye);_.bindFramebuffer(P.READ_FRAMEBUFFER,null),_.bindFramebuffer(P.DRAW_FRAMEBUFFER,null)}else _t?v.isDataTexture||v.isData3DTexture?P.texSubImage3D(Ee,ve,Re,ht,Ct,Te,ye,Ae,xt,Yt,Mt.data):L.isCompressedArrayTexture?P.compressedTexSubImage3D(Ee,ve,Re,ht,Ct,Te,ye,Ae,xt,Mt.data):P.texSubImage3D(Ee,ve,Re,ht,Ct,Te,ye,Ae,xt,Yt,Mt):v.isDataTexture?P.texSubImage2D(P.TEXTURE_2D,ve,Re,ht,Te,ye,xt,Yt,Mt.data):v.isCompressedTexture?P.compressedTexSubImage2D(P.TEXTURE_2D,ve,Re,ht,Mt.width,Mt.height,xt,Mt.data):P.texSubImage2D(P.TEXTURE_2D,ve,Re,ht,Te,ye,xt,Yt,Mt);_.pixelStorei(P.UNPACK_ROW_LENGTH,Kt),_.pixelStorei(P.UNPACK_IMAGE_HEIGHT,at),_.pixelStorei(P.UNPACK_SKIP_PIXELS,_n),_.pixelStorei(P.UNPACK_SKIP_ROWS,Bn),_.pixelStorei(P.UNPACK_SKIP_IMAGES,ui),ve===0&&L.generateMipmaps&&P.generateMipmap(Ee),_.unbindTexture()},this.initRenderTarget=function(v){V.get(v).__webglFramebuffer===void 0&&Y.setupRenderTarget(v)},this.initTexture=function(v){v.isCubeTexture?Y.setTextureCube(v,0):v.isData3DTexture?Y.setTexture3D(v,0):v.isDataArrayTexture||v.isCompressedArrayTexture?Y.setTexture2DArray(v,0):Y.setTexture2D(v,0),_.unbindTexture()},this.resetState=function(){X=0,W=0,ie=null,_.reset(),be.reset()},typeof __THREE_DEVTOOLS__<"u"&&__THREE_DEVTOOLS__.dispatchEvent(new CustomEvent("observe",{detail:this}))}get coordinateSystem(){return Pn}get outputColorSpace(){return this._outputColorSpace}set outputColorSpace(e){this._outputColorSpace=e;let t=this.getContext();t.drawingBufferColorSpace=it._getDrawingBufferColorSpace(e),t.unpackColorSpace=it._getUnpackColorSpace()}};var df={type:"change"},Ah={type:"start"},pf={type:"end"},Pl=new ii,ff=new dn,Cy=Math.cos(70*na.DEG2RAD),Bt=new R,an=2*Math.PI,gt={NONE:-1,ROTATE:0,DOLLY:1,PAN:2,TOUCH_ROTATE:3,TOUCH_PAN:4,TOUCH_DOLLY_PAN:5,TOUCH_DOLLY_ROTATE:6},wh=1e-6,Il=class extends Wr{constructor(e,t=null){super(e,t),this.state=gt.NONE,this.target=new R,this.cursor=new R,this.minDistance=0,this.maxDistance=1/0,this.minZoom=0,this.maxZoom=1/0,this.minTargetRadius=0,this.maxTargetRadius=1/0,this.minPolarAngle=0,this.maxPolarAngle=Math.PI,this.minAzimuthAngle=-1/0,this.maxAzimuthAngle=1/0,this.enableDamping=!1,this.dampingFactor=.05,this.enableZoom=!0,this.zoomSpeed=1,this.enableRotate=!0,this.rotateSpeed=1,this.keyRotateSpeed=1,this.enablePan=!0,this.panSpeed=1,this.screenSpacePanning=!0,this.keyPanSpeed=7,this.zoomToCursor=!1,this.autoRotate=!1,this.autoRotateSpeed=2,this.keys={LEFT:"ArrowLeft",UP:"ArrowUp",RIGHT:"ArrowRight",BOTTOM:"ArrowDown"},this.mouseButtons={LEFT:Ti.ROTATE,MIDDLE:Ti.DOLLY,RIGHT:Ti.PAN},this.touches={ONE:wi.ROTATE,TWO:wi.DOLLY_PAN},this.target0=this.target.clone(),this.position0=this.object.position.clone(),this.zoom0=this.object.zoom,this._cursorStyle="auto",this._domElementKeyEvents=null,this._lastPosition=new R,this._lastQuaternion=new fn,this._lastTargetPosition=new R,this._quat=new fn().setFromUnitVectors(e.up,new R(0,1,0)),this._quatInverse=this._quat.clone().invert(),this._spherical=new Fs,this._sphericalDelta=new Fs,this._scale=1,this._panOffset=new R,this._rotateStart=new ae,this._rotateEnd=new ae,this._rotateDelta=new ae,this._panStart=new ae,this._panEnd=new ae,this._panDelta=new ae,this._dollyStart=new ae,this._dollyEnd=new ae,this._dollyDelta=new ae,this._dollyDirection=new R,this._mouse=new ae,this._performCursorZoom=!1,this._pointers=[],this._pointerPositions={},this._controlActive=!1,this._onPointerMove=Iy.bind(this),this._onPointerDown=Py.bind(this),this._onPointerUp=Ly.bind(this),this._onContextMenu=zy.bind(this),this._onMouseWheel=Uy.bind(this),this._onKeyDown=Fy.bind(this),this._onTouchStart=Oy.bind(this),this._onTouchMove=By.bind(this),this._onMouseDown=Dy.bind(this),this._onMouseMove=Ny.bind(this),this._interceptControlDown=ky.bind(this),this._interceptControlUp=Vy.bind(this),this.domElement!==null&&this.connect(this.domElement),this.update()}set cursorStyle(e){this._cursorStyle=e,e==="grab"?this.domElement.style.cursor="grab":this.domElement.style.cursor="auto"}get cursorStyle(){return this._cursorStyle}connect(e){super.connect(e),this.domElement.addEventListener("pointerdown",this._onPointerDown),this.domElement.addEventListener("pointercancel",this._onPointerUp),this.domElement.addEventListener("contextmenu",this._onContextMenu),this.domElement.addEventListener("wheel",this._onMouseWheel,{passive:!1}),this.domElement.getRootNode().addEventListener("keydown",this._interceptControlDown,{passive:!0,capture:!0}),this.domElement.style.touchAction="none"}disconnect(){this.state=gt.NONE,this.domElement.removeEventListener("pointerdown",this._onPointerDown),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.domElement.removeEventListener("pointercancel",this._onPointerUp),this.domElement.removeEventListener("wheel",this._onMouseWheel),this.domElement.removeEventListener("contextmenu",this._onContextMenu),this.stopListenToKeyEvents();let e=this.domElement.getRootNode();e.removeEventListener("keydown",this._interceptControlDown,{capture:!0}),e.removeEventListener("keyup",this._interceptControlUp,{capture:!0}),this._controlActive=!1,this._pointers.length=0,this._pointerPositions={},this.domElement.style.touchAction="",this.domElement.style.cursor="auto"}dispose(){this.disconnect()}getPolarAngle(){return this._spherical.phi}getAzimuthalAngle(){return this._spherical.theta}getDistance(){return this.object.position.distanceTo(this.target)}listenToKeyEvents(e){e.addEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=e}stopListenToKeyEvents(){this._domElementKeyEvents!==null&&(this._domElementKeyEvents.removeEventListener("keydown",this._onKeyDown),this._domElementKeyEvents=null)}saveState(){this.target0.copy(this.target),this.position0.copy(this.object.position),this.zoom0=this.object.zoom}reset(){this.target.copy(this.target0),this.object.position.copy(this.position0),this.object.zoom=this.zoom0,this.object.updateProjectionMatrix(),this.dispatchEvent(df),this.update(),this.state=gt.NONE}pan(e,t){this._pan(e,t),this.update()}dollyIn(e){this._dollyIn(e),this.update()}dollyOut(e){this._dollyOut(e),this.update()}rotateLeft(e){this._rotateLeft(e),this.update()}rotateUp(e){this._rotateUp(e),this.update()}update(e=null){let t=this.object.position;Bt.copy(t).sub(this.target),Bt.applyQuaternion(this._quat),this._spherical.setFromVector3(Bt),this.autoRotate&&this.state===gt.NONE&&this._rotateLeft(this._getAutoRotationAngle(e)),this.enableDamping?(this._spherical.theta+=this._sphericalDelta.theta*this.dampingFactor,this._spherical.phi+=this._sphericalDelta.phi*this.dampingFactor):(this._spherical.theta+=this._sphericalDelta.theta,this._spherical.phi+=this._sphericalDelta.phi);let n=this.minAzimuthAngle,s=this.maxAzimuthAngle;isFinite(n)&&isFinite(s)&&(n<-Math.PI?n+=an:n>Math.PI&&(n-=an),s<-Math.PI?s+=an:s>Math.PI&&(s-=an),n<=s?this._spherical.theta=Math.max(n,Math.min(s,this._spherical.theta)):this._spherical.theta=this._spherical.theta>(n+s)/2?Math.max(n,this._spherical.theta):Math.min(s,this._spherical.theta)),this._spherical.phi=Math.max(this.minPolarAngle,Math.min(this.maxPolarAngle,this._spherical.phi)),this._spherical.makeSafe(),this.enableDamping===!0?this.target.addScaledVector(this._panOffset,this.dampingFactor):this.target.add(this._panOffset),this.target.sub(this.cursor),this.target.clampLength(this.minTargetRadius,this.maxTargetRadius),this.target.add(this.cursor);let r=!1;if(this.zoomToCursor&&this._performCursorZoom||this.object.isOrthographicCamera)this._spherical.radius=this._clampDistance(this._spherical.radius);else{let a=this._spherical.radius;this._spherical.radius=this._clampDistance(this._spherical.radius*this._scale),r=a!=this._spherical.radius}if(Bt.setFromSpherical(this._spherical),Bt.applyQuaternion(this._quatInverse),t.copy(this.target).add(Bt),this.object.lookAt(this.target),this.enableDamping===!0?(this._sphericalDelta.theta*=1-this.dampingFactor,this._sphericalDelta.phi*=1-this.dampingFactor,this._panOffset.multiplyScalar(1-this.dampingFactor)):(this._sphericalDelta.set(0,0,0),this._panOffset.set(0,0,0)),this.zoomToCursor&&this._performCursorZoom){let a=null;if(this.object.isPerspectiveCamera){let o=Bt.length();a=this._clampDistance(o*this._scale);let l=o-a;this.object.position.addScaledVector(this._dollyDirection,l),this.object.updateMatrixWorld(),r=!!l}else if(this.object.isOrthographicCamera){let o=new R(this._mouse.x,this._mouse.y,0);o.unproject(this.object);let l=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),this.object.updateProjectionMatrix(),r=l!==this.object.zoom;let c=new R(this._mouse.x,this._mouse.y,0);c.unproject(this.object),this.object.position.sub(c).add(o),this.object.updateMatrixWorld(),a=Bt.length()}else console.warn("WARNING: OrbitControls.js encountered an unknown camera type - zoom to cursor disabled."),this.zoomToCursor=!1;a!==null&&(this.screenSpacePanning?this.target.set(0,0,-1).transformDirection(this.object.matrix).multiplyScalar(a).add(this.object.position):(Pl.origin.copy(this.object.position),Pl.direction.set(0,0,-1).transformDirection(this.object.matrix),Math.abs(this.object.up.dot(Pl.direction))<Cy?this.object.lookAt(this.target):(ff.setFromNormalAndCoplanarPoint(this.object.up,this.target),Pl.intersectPlane(ff,this.target))))}else if(this.object.isOrthographicCamera){let a=this.object.zoom;this.object.zoom=Math.max(this.minZoom,Math.min(this.maxZoom,this.object.zoom/this._scale)),a!==this.object.zoom&&(this.object.updateProjectionMatrix(),r=!0)}return this._scale=1,this._performCursorZoom=!1,r||this._lastPosition.distanceToSquared(this.object.position)>wh||8*(1-this._lastQuaternion.dot(this.object.quaternion))>wh||this._lastTargetPosition.distanceToSquared(this.target)>wh?(this.dispatchEvent(df),this._lastPosition.copy(this.object.position),this._lastQuaternion.copy(this.object.quaternion),this._lastTargetPosition.copy(this.target),!0):!1}_getAutoRotationAngle(e){return e!==null?an/60*this.autoRotateSpeed*e:an/60/60*this.autoRotateSpeed}_getZoomScale(e){let t=Math.abs(e*.01);return Math.pow(.95,this.zoomSpeed*t)}_rotateLeft(e){this._sphericalDelta.theta-=e}_rotateUp(e){this._sphericalDelta.phi-=e}_panLeft(e,t){Bt.setFromMatrixColumn(t,0),Bt.multiplyScalar(-e),this._panOffset.add(Bt)}_panUp(e,t){this.screenSpacePanning===!0?Bt.setFromMatrixColumn(t,1):(Bt.setFromMatrixColumn(t,0),Bt.crossVectors(this.object.up,Bt)),Bt.multiplyScalar(e),this._panOffset.add(Bt)}_pan(e,t){let n=this.domElement;if(this.object.isPerspectiveCamera){let s=this.object.position;Bt.copy(s).sub(this.target);let r=Bt.length();r*=Math.tan(this.object.fov/2*Math.PI/180),this._panLeft(2*e*r/n.clientHeight,this.object.matrix),this._panUp(2*t*r/n.clientHeight,this.object.matrix)}else this.object.isOrthographicCamera?(this._panLeft(e*(this.object.right-this.object.left)/this.object.zoom/n.clientWidth,this.object.matrix),this._panUp(t*(this.object.top-this.object.bottom)/this.object.zoom/n.clientHeight,this.object.matrix)):(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - pan disabled."),this.enablePan=!1)}_dollyOut(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale/=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_dollyIn(e){this.object.isPerspectiveCamera||this.object.isOrthographicCamera?this._scale*=e:(console.warn("WARNING: OrbitControls.js encountered an unknown camera type - dolly/zoom disabled."),this.enableZoom=!1)}_updateZoomParameters(e,t){if(!this.zoomToCursor)return;this._performCursorZoom=!0;let n=this.domElement.getBoundingClientRect(),s=e-n.left,r=t-n.top,a=n.width,o=n.height;this._mouse.x=s/a*2-1,this._mouse.y=-(r/o)*2+1,this._dollyDirection.set(this._mouse.x,this._mouse.y,1).unproject(this.object).sub(this.object.position).normalize()}_clampDistance(e){return Math.max(this.minDistance,Math.min(this.maxDistance,e))}_handleMouseDownRotate(e){this._rotateStart.set(e.clientX,e.clientY)}_handleMouseDownDolly(e){this._updateZoomParameters(e.clientX,e.clientX),this._dollyStart.set(e.clientX,e.clientY)}_handleMouseDownPan(e){this._panStart.set(e.clientX,e.clientY)}_handleMouseMoveRotate(e){this._rotateEnd.set(e.clientX,e.clientY),this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);let t=this.domElement;this._rotateLeft(an*this._rotateDelta.x/t.clientHeight),this._rotateUp(an*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd),this.update()}_handleMouseMoveDolly(e){this._dollyEnd.set(e.clientX,e.clientY),this._dollyDelta.subVectors(this._dollyEnd,this._dollyStart),this._dollyDelta.y>0?this._dollyOut(this._getZoomScale(this._dollyDelta.y)):this._dollyDelta.y<0&&this._dollyIn(this._getZoomScale(this._dollyDelta.y)),this._dollyStart.copy(this._dollyEnd),this.update()}_handleMouseMovePan(e){this._panEnd.set(e.clientX,e.clientY),this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd),this.update()}_handleMouseWheel(e){this._updateZoomParameters(e.clientX,e.clientY),e.deltaY<0?this._dollyIn(this._getZoomScale(e.deltaY)):e.deltaY>0&&this._dollyOut(this._getZoomScale(e.deltaY)),this.update()}_handleKeyDown(e){let t=!1;switch(e.code){case this.keys.UP:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(an*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,this.keyPanSpeed),t=!0;break;case this.keys.BOTTOM:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateUp(-an*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(0,-this.keyPanSpeed),t=!0;break;case this.keys.LEFT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(an*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(this.keyPanSpeed,0),t=!0;break;case this.keys.RIGHT:e.ctrlKey||e.metaKey||e.shiftKey?this.enableRotate&&this._rotateLeft(-an*this.keyRotateSpeed/this.domElement.clientHeight):this.enablePan&&this._pan(-this.keyPanSpeed,0),t=!0;break}t&&(e.preventDefault(),this.update())}_handleTouchStartRotate(e){if(this._pointers.length===1)this._rotateStart.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._rotateStart.set(n,s)}}_handleTouchStartPan(e){if(this._pointers.length===1)this._panStart.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._panStart.set(n,s)}}_handleTouchStartDolly(e){let t=this._getSecondPointerPosition(e),n=e.pageX-t.x,s=e.pageY-t.y,r=Math.sqrt(n*n+s*s);this._dollyStart.set(0,r)}_handleTouchStartDollyPan(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enablePan&&this._handleTouchStartPan(e)}_handleTouchStartDollyRotate(e){this.enableZoom&&this._handleTouchStartDolly(e),this.enableRotate&&this._handleTouchStartRotate(e)}_handleTouchMoveRotate(e){if(this._pointers.length==1)this._rotateEnd.set(e.pageX,e.pageY);else{let n=this._getSecondPointerPosition(e),s=.5*(e.pageX+n.x),r=.5*(e.pageY+n.y);this._rotateEnd.set(s,r)}this._rotateDelta.subVectors(this._rotateEnd,this._rotateStart).multiplyScalar(this.rotateSpeed);let t=this.domElement;this._rotateLeft(an*this._rotateDelta.x/t.clientHeight),this._rotateUp(an*this._rotateDelta.y/t.clientHeight),this._rotateStart.copy(this._rotateEnd)}_handleTouchMovePan(e){if(this._pointers.length===1)this._panEnd.set(e.pageX,e.pageY);else{let t=this._getSecondPointerPosition(e),n=.5*(e.pageX+t.x),s=.5*(e.pageY+t.y);this._panEnd.set(n,s)}this._panDelta.subVectors(this._panEnd,this._panStart).multiplyScalar(this.panSpeed),this._pan(this._panDelta.x,this._panDelta.y),this._panStart.copy(this._panEnd)}_handleTouchMoveDolly(e){let t=this._getSecondPointerPosition(e),n=e.pageX-t.x,s=e.pageY-t.y,r=Math.sqrt(n*n+s*s);this._dollyEnd.set(0,r),this._dollyDelta.set(0,Math.pow(this._dollyEnd.y/this._dollyStart.y,this.zoomSpeed)),this._dollyOut(this._dollyDelta.y),this._dollyStart.copy(this._dollyEnd);let a=(e.pageX+t.x)*.5,o=(e.pageY+t.y)*.5;this._updateZoomParameters(a,o)}_handleTouchMoveDollyPan(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enablePan&&this._handleTouchMovePan(e)}_handleTouchMoveDollyRotate(e){this.enableZoom&&this._handleTouchMoveDolly(e),this.enableRotate&&this._handleTouchMoveRotate(e)}_addPointer(e){this._pointers.push(e.pointerId)}_removePointer(e){delete this._pointerPositions[e.pointerId];for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId){this._pointers.splice(t,1);return}}_isTrackingPointer(e){for(let t=0;t<this._pointers.length;t++)if(this._pointers[t]==e.pointerId)return!0;return!1}_trackPointer(e){let t=this._pointerPositions[e.pointerId];t===void 0&&(t=new ae,this._pointerPositions[e.pointerId]=t),t.set(e.pageX,e.pageY)}_getSecondPointerPosition(e){let t=e.pointerId===this._pointers[0]?this._pointers[1]:this._pointers[0];return this._pointerPositions[t]}_customWheelEvent(e){let t=e.deltaMode,n={clientX:e.clientX,clientY:e.clientY,deltaY:e.deltaY};switch(t){case 1:n.deltaY*=16;break;case 2:n.deltaY*=100;break}return e.ctrlKey&&!this._controlActive&&(n.deltaY*=10),n}};function Py(i){this.enabled!==!1&&(this._pointers.length===0&&(this.domElement.setPointerCapture(i.pointerId),this.domElement.ownerDocument.addEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.addEventListener("pointerup",this._onPointerUp)),!this._isTrackingPointer(i)&&(this._addPointer(i),i.pointerType==="touch"?this._onTouchStart(i):this._onMouseDown(i),this._cursorStyle==="grab"&&(this.domElement.style.cursor="grabbing")))}function Iy(i){this.enabled!==!1&&(i.pointerType==="touch"?this._onTouchMove(i):this._onMouseMove(i))}function Ly(i){switch(this._removePointer(i),this._pointers.length){case 0:this.domElement.releasePointerCapture(i.pointerId),this.domElement.ownerDocument.removeEventListener("pointermove",this._onPointerMove),this.domElement.ownerDocument.removeEventListener("pointerup",this._onPointerUp),this.dispatchEvent(pf),this.state=gt.NONE,this._cursorStyle==="grab"&&(this.domElement.style.cursor="grab");break;case 1:let e=this._pointers[0],t=this._pointerPositions[e];this._onTouchStart({pointerId:e,pageX:t.x,pageY:t.y});break}}function Dy(i){let e;switch(i.button){case 0:e=this.mouseButtons.LEFT;break;case 1:e=this.mouseButtons.MIDDLE;break;case 2:e=this.mouseButtons.RIGHT;break;default:e=-1}switch(e){case Ti.DOLLY:if(this.enableZoom===!1)return;this._handleMouseDownDolly(i),this.state=gt.DOLLY;break;case Ti.ROTATE:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=gt.PAN}else{if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=gt.ROTATE}break;case Ti.PAN:if(i.ctrlKey||i.metaKey||i.shiftKey){if(this.enableRotate===!1)return;this._handleMouseDownRotate(i),this.state=gt.ROTATE}else{if(this.enablePan===!1)return;this._handleMouseDownPan(i),this.state=gt.PAN}break;default:this.state=gt.NONE}this.state!==gt.NONE&&this.dispatchEvent(Ah)}function Ny(i){switch(this.state){case gt.ROTATE:if(this.enableRotate===!1)return;this._handleMouseMoveRotate(i);break;case gt.DOLLY:if(this.enableZoom===!1)return;this._handleMouseMoveDolly(i);break;case gt.PAN:if(this.enablePan===!1)return;this._handleMouseMovePan(i);break}}function Uy(i){this.enabled===!1||this.enableZoom===!1||this.state!==gt.NONE||(i.preventDefault(),this.dispatchEvent(Ah),this._handleMouseWheel(this._customWheelEvent(i)),this.dispatchEvent(pf))}function Fy(i){this.enabled!==!1&&this._handleKeyDown(i)}function Oy(i){switch(this._trackPointer(i),this._pointers.length){case 1:switch(this.touches.ONE){case wi.ROTATE:if(this.enableRotate===!1)return;this._handleTouchStartRotate(i),this.state=gt.TOUCH_ROTATE;break;case wi.PAN:if(this.enablePan===!1)return;this._handleTouchStartPan(i),this.state=gt.TOUCH_PAN;break;default:this.state=gt.NONE}break;case 2:switch(this.touches.TWO){case wi.DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchStartDollyPan(i),this.state=gt.TOUCH_DOLLY_PAN;break;case wi.DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchStartDollyRotate(i),this.state=gt.TOUCH_DOLLY_ROTATE;break;default:this.state=gt.NONE}break;default:this.state=gt.NONE}this.state!==gt.NONE&&this.dispatchEvent(Ah)}function By(i){switch(this._trackPointer(i),this.state){case gt.TOUCH_ROTATE:if(this.enableRotate===!1)return;this._handleTouchMoveRotate(i),this.update();break;case gt.TOUCH_PAN:if(this.enablePan===!1)return;this._handleTouchMovePan(i),this.update();break;case gt.TOUCH_DOLLY_PAN:if(this.enableZoom===!1&&this.enablePan===!1)return;this._handleTouchMoveDollyPan(i),this.update();break;case gt.TOUCH_DOLLY_ROTATE:if(this.enableZoom===!1&&this.enableRotate===!1)return;this._handleTouchMoveDollyRotate(i),this.update();break;default:this.state=gt.NONE}}function zy(i){this.enabled!==!1&&i.preventDefault()}function ky(i){i.key==="Control"&&(this._controlActive=!0,this.domElement.getRootNode().addEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}function Vy(i){i.key==="Control"&&(this._controlActive=!1,this.domElement.getRootNode().removeEventListener("keyup",this._interceptControlUp,{passive:!0,capture:!0}))}var Ll=class extends Hi{constructor(){super(),this.name="RoomEnvironment",this.position.y=-3.5;let e=new Gn;e.deleteAttribute("uv");let t=new Mi({side:Xt}),n=new Mi,s=new Hr(16777215,900,28,2);s.position.set(.418,16.199,.3),this.add(s);let r=new mt(e,t);r.position.set(-.757,13.219,.717),r.scale.set(31.713,28.305,28.591),this.add(r);let a=new vr(e,n,6),o=new Lt;o.position.set(-10.906,2.009,1.846),o.rotation.set(0,-.195,0),o.scale.set(2.328,7.905,4.651),o.updateMatrix(),a.setMatrixAt(0,o.matrix),o.position.set(-5.607,-.754,-.758),o.rotation.set(0,.994,0),o.scale.set(1.97,1.534,3.955),o.updateMatrix(),a.setMatrixAt(1,o.matrix),o.position.set(6.167,.857,7.803),o.rotation.set(0,.561,0),o.scale.set(3.927,6.285,3.687),o.updateMatrix(),a.setMatrixAt(2,o.matrix),o.position.set(-2.017,.018,6.124),o.rotation.set(0,.333,0),o.scale.set(2.002,4.566,2.064),o.updateMatrix(),a.setMatrixAt(3,o.matrix),o.position.set(2.291,-.756,-2.621),o.rotation.set(0,-.286,0),o.scale.set(1.546,1.552,1.496),o.updateMatrix(),a.setMatrixAt(4,o.matrix),o.position.set(-2.193,-.369,-5.547),o.rotation.set(0,.516,0),o.scale.set(3.875,3.487,2.986),o.updateMatrix(),a.setMatrixAt(5,o.matrix),this.add(a);let l=new mt(e,qs(50));l.position.set(-16.116,14.37,8.208),l.scale.set(.1,2.428,2.739),this.add(l);let c=new mt(e,qs(50));c.position.set(-16.109,18.021,-8.207),c.scale.set(.1,2.425,2.751),this.add(c);let h=new mt(e,qs(17));h.position.set(14.904,12.198,-1.832),h.scale.set(.15,4.265,6.331),this.add(h);let d=new mt(e,qs(43));d.position.set(-.462,8.89,14.52),d.scale.set(4.38,5.441,.088),this.add(d);let u=new mt(e,qs(20));u.position.set(3.235,11.486,-12.541),u.scale.set(2.5,2,.1),this.add(u);let f=new mt(e,qs(100));f.position.set(0,20,0),f.scale.set(1,.1,1),this.add(f)}dispose(){let e=new Set;this.traverse(t=>{t.isMesh&&(e.add(t.geometry),e.add(t.material))});for(let t of e)t.dispose()}};function qs(i){return new Or({color:0,emissive:16777215,emissiveIntensity:i})}function oa(i,e,t){let n=i*374761393+e*668265263+t*2147483647|0;return n=Math.imul(n^n>>>13,1274126177),((n^n>>>16)>>>0)/4294967296}function Dl(i,e,t){let n=Math.floor(i),s=Math.floor(e),r=i-n,a=e-s,o=r*r*(3-2*r),l=a*a*(3-2*a),c=oa(n,s,t),h=oa(n+1,s,t),d=oa(n,s+1,t),u=oa(n+1,s+1,t);return c+(h-c)*o+(d-c)*l+(c-h-d+u)*o*l}var Rh=i=>[i>>16&255,i>>8&255,i&255],Hy={huali:{name:"\u9EC4\u82B1\u68A8",base:11891772,dark:7290140,light:14262374,freq:22,seed:3},zitan:{name:"\u7D2B\u6A80",base:5580828,dark:2756618,light:8010280,freq:34,seed:7},jumu:{name:"\u6989\u6728",base:13211748,dark:9396022,light:15122574,freq:18,seed:11},jichi:{name:"\u9E21\u7FC5\u6728",base:9066552,dark:4138006,light:11566677,freq:40,seed:5},hongmu:{name:"\u7EA2\u9178\u679D",base:9322279,dark:5118992,light:11821130,freq:26,seed:13},stone:{name:"\u77F3",base:10130570,dark:7301730,light:12170152,freq:70,seed:21},iron:{name:"\u94C1",base:3947064,dark:2039068,light:5920080,freq:90,seed:23},paper:{name:"\u7EB8",base:14272931,dark:11048044,light:15655620,freq:60,seed:29},bench:{name:"\u8001\u6986\u6728",base:7165761,dark:4075298,light:9335897,freq:12,seed:17}};function Gy(i,e=1024,t=512){let n=document.createElement("canvas");n.width=e,n.height=t;let s=n.getContext("2d"),r=s.createImageData(e,t),a=Rh(i.base),o=Rh(i.dark),l=Rh(i.light);for(let h=0;h<t;h++)for(let d=0;d<e;d++){let u=d/e,f=h/t,g=2.2*Dl(u*3,f*2.5,i.seed)+.5*Dl(u*14,f*9,i.seed+1),y=f*i.freq+g*1.6,m=y-Math.floor(y),p=Math.pow(Math.max(0,1-Math.abs(m-.5)*2.4),3),w=Math.pow(m,5)*.5,A=Dl(u*400,f*40,i.seed+2)*.18,M=oa(d/6|0,h,i.seed)>.985?.35:0,E=Dl(u*2,f*6,i.seed+3),S=Math.min(1,p*.7+w+A+M),C=(h*e+d)*4;for(let x=0;x<3;x++){let T=a[x]+(l[x]-a[x])*E*.55;T=T+(o[x]-T)*S,r.data[C+x]=T}r.data[C+3]=255}s.putImageData(r,0,0);let c=new Er(n);return c.wrapS=c.wrapT=Ms,c.colorSpace=Vt,c.anisotropy=8,c}var Ch={};function Nl(i,e={}){return Ch[i]||(Ch[i]=Gy(Hy[i])),new Mi({map:Ch[i],roughness:e.roughness??.58,metalness:0,envMapIntensity:.55})}function Ul(i,e,t=.09,n=[0,0]){i.index&&(i=i.toNonIndexed()),i.attributes.normal||i.computeVertexNormals();let s=i.attributes.position,r=i.attributes.normal,a=new Float32Array(s.count*2);for(let o=0;o<s.count;o++){let l=[s.getX(o),s.getY(o),s.getZ(o)],c=[Math.abs(r.getX(o)),Math.abs(r.getY(o)),Math.abs(r.getZ(o))],h=c[0]>c[1]?c[0]>c[2]?0:2:c[1]>c[2]?1:2,d,u;if(h!==e){let f=3-h-e;d=l[e]*t,u=l[f]*t*1.4}else d=l[(e+1)%3]*t*.35,u=l[(e+2)%3]*t*2.2;a[o*2]=d+n[0],a[o*2+1]=u+n[1]}return i.setAttribute("uv",new It(a,2)),i}var ri=[[[-4,-2,-1],[-4,-2,0],[-4,-1,-1],[-4,-1,0],[-3,-2,-1],[-3,-2,0],[-3,-1,-1],[-3,-1,0],[-2,-2,-1],[-2,-2,0],[1,-2,-1],[1,-2,0],[2,-2,-1],[2,-2,0],[2,-1,-1],[2,-1,0],[3,-2,-1],[3,-2,0],[3,-1,-1],[3,-1,0],[-2,-1,-1],[-2,-1,0],[-1,-1,-1],[0,-2,-1],[0,-1,-1]],[[-4,0,-1],[-4,0,0],[-4,1,-1],[-4,1,0],[-3,0,-1],[-3,0,0],[-3,1,-1],[-3,1,0],[-2,1,-1],[-2,1,0],[1,1,-1],[1,1,0],[2,0,-1],[2,0,0],[2,1,-1],[2,1,0],[3,0,-1],[3,0,0],[3,1,-1],[3,1,0],[-1,1,-1],[0,0,-1],[0,1,-1],[1,0,-1],[1,0,0]],[[-1,-4,-2],[-1,-4,-1],[0,-4,-2],[0,-4,-1],[-1,-3,-2],[-1,-3,-1],[0,-3,-2],[0,-3,-1],[-1,-2,-2],[0,-2,-2],[-1,1,-2],[0,1,-2],[-1,2,-2],[-1,2,-1],[0,2,-2],[0,2,-1],[-1,3,-2],[-1,3,-1],[0,3,-2],[0,3,-1],[-1,-2,-1],[-1,-1,-2],[0,-1,-2],[0,0,-2]],[[-1,-4,0],[-1,-4,1],[0,-4,0],[0,-4,1],[-1,-3,0],[-1,-3,1],[0,-3,0],[0,-3,1],[-1,-2,1],[0,-2,1],[-1,1,1],[0,1,1],[-1,2,0],[-1,2,1],[0,2,0],[0,2,1],[-1,3,0],[-1,3,1],[0,3,0],[0,3,1],[-1,-2,0],[-1,-1,0],[0,-2,0],[0,-1,0],[-1,0,0],[-1,1,0],[0,0,0],[0,1,0],[-1,-1,1],[0,-1,1],[-1,0,1],[0,0,1]],[[-2,-1,-4],[-2,0,-4],[-1,-1,-4],[-1,0,-4],[-2,-1,-3],[-2,0,-3],[-1,-1,-3],[-1,0,-3],[-2,-1,-2],[-2,0,-2],[-2,-1,1],[-2,0,1],[-2,-1,2],[-2,0,2],[-1,-1,2],[-1,0,2],[-2,-1,3],[-2,0,3],[-1,-1,3],[-1,0,3],[-2,0,-1],[-2,0,0],[-1,0,-2]],[[0,-1,-4],[0,0,-4],[1,-1,-4],[1,0,-4],[0,-1,-3],[0,0,-3],[1,-1,-3],[1,0,-3],[1,-1,-2],[1,0,-2],[1,-1,1],[1,0,1],[0,-1,2],[0,0,2],[1,-1,2],[1,0,2],[0,-1,3],[0,0,3],[1,-1,3],[1,0,3],[1,-1,-1],[1,-1,0]]],la=3;var bn={"+x":[1,0,0],"-x":[-1,0,0],"+y":[0,1,0],"-y":[0,-1,0],"+z":[0,0,1],"-z":[0,0,-1]};function ft(i,e,t,n,{holes:s=[],bevel:r=.03}={}){let a=new Is(i.map(c=>new ae(c[0],c[1])));for(let c of s)a.holes.push(new Gi(c.map(h=>new ae(h[0],h[1]))));let o=r,l=new Ur(a,{depth:n-t-2*o,bevelEnabled:o>0,bevelThickness:o,bevelSize:o,bevelOffset:-o,bevelSegments:2,curveSegments:1,steps:1});l.translate(0,0,t+o);for(let c of["position","normal"]){let h=l.attributes[c];for(let d=0;d<h.count;d++){let u=h.getX(d),f=h.getY(d),g=h.getZ(d),y=[0,0,0];y[e]=g,y[(e+1)%3]=u,y[(e+2)%3]=f,h.setXYZ(d,y[0],y[1],y[2])}}return l}function Be(i,e,t,n,s,r,a,o=.03){let l=[[i,e],[t,n],[s,r]],c=l[(a+1)%3],h=l[(a+2)%3];return ft([[c[0],h[0]],[c[1],h[0]],[c[1],h[1]],[c[0],h[1]]],a,l[a][0],l[a][1],{bevel:o})}function Me(i,e,t,n,s){let r=new nn,a=Nl(n),o=[Math.random()*5,Math.random()*5];for(let l of t){let c=new mt(Ul(l,s,.15,o),a);c.castShadow=c.receiveShadow=!0,c.userData.pieceId=i,r.add(c)}return r.userData={id:i,name:e,mat:a},{id:i,name:e,group:r,mat:a}}function Ph(i){let e=new Set;return{...i,removed:e,tryMove(t,n){if(e.has(t))return{kind:"blocked",msg:"\u8FD9\u5757\u5DF2\u7ECF\u62C6\u4E0B\u6765\u4E86\u3002"};let s=i.rule(t,n,e);return typeof s=="string"?{kind:"blocked",msg:s}:(e.add(t),{kind:"move",to:bn[n].map(a=>a*s.out),out:!0})},done(){return i.isDone(e)}}}function Wy(){let i=Me("base","\u9762\u677F\uFF08\u71D5\u5C3E\u69FD\uFF09",[ft([[-4.5,0],[4.5,0],[4.5,2],[.8,2],[1.2,.8],[-1.2,.8],[-.8,2],[-4.5,2]],2,-3.5,3.5)],"jumu",0),e=Me("tongue","\u7A7F\u5E26\uFF08\u71D5\u5C3E\u69AB\uFF09",[ft([[-1.2,.8],[1.2,.8],[.8,2],[.8,3.4],[-.8,3.4],[-.8,2]],2,-3.5,3.5)],"hongmu",2);return Ph({id:"dovetail",title:"\u71D5\u5C3E\u69AB",pieces:[i,e],rootY:0,camera:{pos:[9,7.5,11],target:[0,1.6,0]},reveal:{pos:[16,9,12],target:[0,1.6,4.5]},xray:["base"],labels:[{id:"tongue",p:[1,1.4,-3.5],text:"\u71D5\u5C3E\u69AB\uFF1A\u5916\u5BBD\u5185\u7A84"},{id:"base",p:[-1,1.4,3.5],text:"\u71D5\u5C3E\u69FD\uFF08\u4E24\u5934\u901A\uFF09"}],rule(t,n){return t==="base"?"\u9762\u677F\u538B\u5728\u53F0\u9762\u4E0A\uFF0C\u52A8\u5B83\u6CA1\u7528\u3002\u8BD5\u8BD5\u4E0A\u9762\u90A3\u6839\u6728\u6761\u3002":n==="+z"||n==="-z"?{out:9}:n==="+y"?"\u5F80\u4E0A\u62D4\u2026\u2026\u659C\u9762\u5361\u4F4F\u4E86\u3002\u71D5\u5C3E\u5916\u5BBD\u5185\u7A84\uFF0C\u8D8A\u5F80\u4E0A\u62D4\u54AC\u5F97\u8D8A\u7D27\u3002":n==="-y"?"\u5E95\u4E0B\u5C31\u662F\u69FD\u5E95\uFF0C\u538B\u4E0D\u4E0B\u53BB\u3002":"\u4E24\u4FA7\u7684\u659C\u9762\u6321\u7740\uFF0C\u5DE6\u53F3\u63A8\u4E0D\u52A8\u3002"},isDone:t=>t.has("tongue")})}function Xy(){let i=Me("post","\u51F3\u817F\uFF08\u900F\u773C\uFF09",[ft([[0,-1.5],[10,-1.5],[10,1.5],[0,1.5]],0,-1.5,1.5,{holes:[[[4,-.6],[6,-.6],[6,.6],[4,.6]]]})],"huali",1),e=.012,t=Me("rail","\u6A2A\u67A8\uFF08\u900F\u69AB\uFF09",[Be(-9.5,-1.5,3.5,6.5,-1,1,0),Be(-1.55,.3,4+e,6-e,-.6+e,.6-e,0,0),Be(.3,1.8,5.08,6-e,-.6+e,.6-e,0,.015),Be(.3,1.8,4+e,4.92,-.6+e,.6-e,0,.015)],"huali",0),n=Me("wedge","\u7834\u5934\u6954",[ft([[.3,4.995],[2.45,4.9],[2.45,5.1],[.3,5.005]],2,-.55,.55,{bevel:0})],"zitan",0);return Ph({id:"through",title:"\u900F\u69AB\xB7\u7834\u5934\u6954",pieces:[i,t,n],rootY:0,camera:{pos:[10,9,15],target:[-2.5,4.5,0]},reveal:{pos:[3,9,23],target:[-3.5,5,0]},xray:["post"],labels:[{id:"post",p:[1.5,6.4,0],text:"\u536F\u773C\uFF08\u51FF\u7A7F\u7684\u900F\u773C\uFF09"},{id:"rail",p:[1.8,6,.6],text:"\u900F\u69AB\uFF1A\u7A7F\u51FA\u817F\u5916"},{id:"wedge",p:[2.45,5.1,.55],text:"\u7834\u5934\u6954"}],rule(s,r,a){return s==="post"?"\u51F3\u817F\u7ACB\u5728\u53F0\u9762\u4E0A\u3002\u5148\u4ECE\u6A2A\u67A8\u8FD9\u8FB9\u4E0B\u624B\u3002":s==="wedge"?r==="+x"?{out:3}:r==="-x"?"\u6954\u5B50\u5DF2\u7ECF\u6253\u5230\u5E95\u4E86\uFF0C\u518D\u5F80\u91CC\u53EA\u4F1A\u66F4\u7D27\u3002":"\u6954\u5B50\u5939\u5728\u69AB\u5934\u7AEF\u90E8\u7684\u952F\u53E3\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u952F\u53E3\u8FDB\u51FA\u3002":r==="+x"?"\u69AB\u80A9\u5DF2\u7ECF\u62B5\u7740\u51F3\u817F\u4E86\u3002":r!=="-x"?"\u536F\u773C\u56DB\u9762\u5305\u7740\u69AB\u5934\uFF0C\u53EA\u80FD\u987A\u7740\u69AB\u5934\u7684\u65B9\u5411\u8FDB\u51FA\u3002":a.has("wedge")?{out:7}:"\u62BD\u4E0D\u52A8\u3002\u7AEF\u5934\u90A3\u7247\u6954\u5B50\u628A\u69AB\u5934\u6491\u5F00\u4E86\uFF0C\u80C0\u6B7B\u5728\u536F\u773C\u91CC\u3002"},isDone:s=>s.has("rail")})}var mf=[[-8,-1],[3,-1],[3,-.75],[3.6,-.75],[3.6,-.25],[3,-.25],[3,0],[.5,0],[.5,-.5],[-.5,-.5],[-.5,0],[-3,0],[-3,.25],[-3.6,.25],[-3.6,.75],[-3,.75],[-3,1],[-8,1]];function qy(){let i=Me("left","\u6276\u624B\u5DE6\u6BB5",[ft(mf,2,-1,1)],"huali",0),e=Me("right","\u6276\u624B\u53F3\u6BB5",[ft(mf.map(([r,a])=>[-r,-a]),2,-1,1)],"huali",0),t=Me("pin","\u6954\u9489",[ft([[-1.45,-.5],[1.45,-.36],[1.45,.36],[-1.45,.5]],1,-.5,.5,{bevel:.02})],"zitan",2),n="\u4E24\u5934\u7684\u5C0F\u820C\u63D2\u5728\u5BF9\u65B9\u7684\u69FD\u91CC\uFF0C\u4E0A\u4E0B\u9519\u4E0D\u5F00\u3002",s="\u6954\u9489\u6A2A\u7A7F\u4E24\u7247\u642D\u53E3\uFF0C\u5DE6\u53F3\u62C9\u4E0D\u5F00\u3002";return Ph({id:"pin",title:"\u6954\u9489\u69AB",pieces:[i,e,t],rootY:1,camera:{pos:[5,8,15],target:[0,1,0]},reveal:{pos:[4,9,19],target:[0,1,-1.5]},xray:["left","right"],labels:[{id:"pin",p:[0,.5,-1.45],text:"\u6954\u9489\uFF1A\u4E00\u5934\u7C97\u4E00\u5934\u7EC6"},{id:"left",p:[3.6,-.5,1],text:"\u5C0F\u820C"},{id:"right",p:[-3.3,.75,1],text:"\u5C0F\u820C\u5165\u69FD"},{id:"left",p:[-1.8,0,1],text:"\u5408\u638C\u5F0F\u642D\u53E3"}],rule(r,a,o){if(r==="pin")return a==="-z"?{out:4.5}:a==="+z"?"\u8FD9\u5934\u7EC6\u3001\u90A3\u5934\u7C97\u3002\u5F80\u7EC6\u7684\u65B9\u5411\u63A8\uFF0C\u53EA\u4F1A\u8D8A\u63A8\u8D8A\u7D27\u3002":"\u6954\u9489\u7A7F\u5728\u65B9\u5B54\u91CC\uFF0C\u53EA\u80FD\u6CBF\u5B54\u8FDB\u51FA\u3002";let l=!o.has("pin");return a==="+z"||a==="-z"?"\u5C0F\u820C\u548C\u69FD\u662F\u6697\u505A\u7684\uFF0C\u4ECE\u4FA7\u9762\u63A8\u4E0D\u52A8\u3002":r==="left"?a==="-y"?"\u5E95\u4E0B\u662F\u53F0\u9762\u3002":a==="+y"?n:a==="+x"?"\u5DF2\u7ECF\u9876\u5230\u5E95\u4E86\u3002":l?s:{out:6}:a==="+y"||a==="-y"?n:a==="-x"?"\u5DF2\u7ECF\u9876\u5230\u5E95\u4E86\u3002":l?s:{out:6}},isDone:r=>r.has("pin")&&(r.has("left")||r.has("right"))})}function Yy(i){let e=new Set(i.map(a=>a.join(","))),t=[],n=[],s=[[[1,0,0],[[1,0,0],[1,1,0],[1,1,1],[1,0,1]]],[[-1,0,0],[[0,0,0],[0,0,1],[0,1,1],[0,1,0]]],[[0,1,0],[[0,1,0],[0,1,1],[1,1,1],[1,1,0]]],[[0,-1,0],[[0,0,0],[1,0,0],[1,0,1],[0,0,1]]],[[0,0,1],[[0,0,1],[1,0,1],[1,1,1],[0,1,1]]],[[0,0,-1],[[0,0,0],[0,1,0],[1,1,0],[1,0,0]]]];for(let a of i)for(let[o,l]of s)if(!e.has([a[0]+o[0],a[1]+o[1],a[2]+o[2]].join(",")))for(let c of[0,1,2,0,2,3])t.push(a[0]+l[c][0],a[1]+l[c][1],a[2]+l[c][2]),n.push(...o);let r=new Ut;return r.setAttribute("position",new Tt(t,3)),r.setAttribute("normal",new Tt(n,3)),r}function Zy(){let i=["huali","jumu","hongmu","huali","jumu","hongmu"],e=["\u6A2A\u6728\xB7\u4E0A","\u6A2A\u6728\xB7\u4E0B","\u7AD6\u6728\xB7\u524D","\u7AD6\u6728\xB7\u540E","\u7EB5\u6728\xB7\u5DE6","\u7EB5\u6728\xB7\u53F3"],t=ri.map((h,d)=>{let u=Math.floor(d/2),f=Me("b"+d,d===la?"\u94A5\u5319":e[d],[Yy(h)],d===la?"zitan":i[d],u),g=new Mr(new Ar(f.group.children[0].geometry,25),new Cs({color:2364427,transparent:!0,opacity:.45}));return f.group.add(g),f}),n=ri.map(()=>[0,0,0]),s=new Set,r=(h,d)=>{let u=new Set;return ri.forEach((f,g)=>{if(g===h||s.has("b"+g))return;let y=n[g];for(let m of f)u.add(m[0]+y[0]+","+(m[1]+y[1])+","+(m[2]+y[2]))}),u},a=(h,d,u)=>ri[h].every(f=>f[1]+d[1]>=-4&&!u.has(f[0]+d[0]+","+(f[1]+d[1])+","+(f[2]+d[2]))),o=(h,d,u)=>ri[h].every(f=>!u.has(f[0]+d[0]+","+(f[1]+d[1])+","+(f[2]+d[2]))),l=ri[la],c=Math.max(...l.map(h=>h[1]))+1;return{id:"burr",title:"\u9C81\u73ED\u9501",pieces:t,rootY:4,removed:s,stepwise:!0,camera:{pos:[13,10,15],target:[0,4,0]},reveal:{pos:[20,15,23],target:[0,9.5,0]},xray:[],labels:[{id:"b"+la,p:[0,c,.5],text:"\u94A5\u5319\uFF1A\u552F\u4E00\u6CA1\u6709\u7F3A\u53E3\u7684\u4E00\u6839"},{id:"b1",p:[4,1,0],text:"\u5176\u4F59\u4E94\u6839\u5404\u6709\u7F3A\u53E3\uFF0C\u4E92\u76F8\u54AC\u5408"}],tryMove(h,d){let u=+h.slice(1);if(s.has(h))return{kind:"blocked",msg:"\u8FD9\u6839\u5DF2\u7ECF\u53D6\u51FA\u6765\u4E86\u3002"};let f=bn[d],g=r(u),y=n[u].map((p,w)=>p+f[w]);if(!a(u,y,g))return{kind:"blocked",msg:ri[u].some(w=>w[1]+y[1]<-4)?"\u5E95\u4E0B\u662F\u53F0\u9762\u3002":"\u5361\u4F4F\u4E86\uFF0C\u6709\u522B\u7684\u6728\u6761\u6321\u7740\u3002"};let m=!0;for(let p=1;p<16;p++){let w=y.map((A,M)=>A+f[M]*p);if(!o(u,w,g)){m=!1;break}}return f[1]<0&&(m=!1),n[u]=y,m?(s.add(h),{kind:"move",to:y.map((p,w)=>p+f[w]*8),out:!0}):{kind:"move",to:y.slice(),out:!1}},done(){return s.size>=5},explode:{rootY:10,offsets:Object.fromEntries(ri.map((h,d)=>{let u=Math.floor(d/2),f=[0,1,2].map(y=>h.reduce((m,p)=>m+p[y]+.5,0)/h.length),g=[0,0,0];for(let y=0;y<3;y++)y!==u&&Math.abs(f[y])>.4&&(g[y]=Math.sign(f[y])*5.5);return["b"+d,g]}))}}}var gf={dovetail:Wy,through:Xy,pin:qy,burr:Zy};var $y=i=>i.map(([e,t])=>[t,e]),ca=(i,e,t,n)=>ft($y(i),1,e,t,n);function _f(i,e,t){return new Wt(i,i*1.04,t-e,40,1).translate(0,(e+t)/2,0)}function qt(i){let e=new Set,t={},n={},s=Object.fromEntries(i.pieces.map(a=>[a.id,a])),r=i.spec;return{...i,removed:e,tryMove(a,o){let l=r[a]||{};if(e.has(a))return{kind:"blocked",msg:"\u8FD9\u5757\u5DF2\u7ECF\u62C6\u4E0B\u6765\u4E86\u3002"};if(l.fixed)return{kind:"blocked",msg:l.fixed};let c=t[a]||0,h=l.steps[c];if(o!==h.dir){let y=l.block||{};return{kind:"blocked",msg:(typeof y=="function"?y(o,c):y[o+"@"+c]||y[o])||y.default||l.blockMsg||"\u63A8\u4E0D\u52A8\u3002"}}let d=(h.needs||[]).filter(y=>!e.has(y));if(d.length){let y=[...new Set(d.map(m=>s[m].name))].join("\u300D\u300C");return{kind:"blocked",msg:h.needMsg?h.needMsg.replace("{n}",y):`\u52A8\u4E0D\u4E86\uFF0C\u300C${y}\u300D\u8FD8\u538B\u7740\u5B83\u3002`}}let u=n[a]||[0,0,0],f=bn[o];n[a]=u.map((y,m)=>y+f[m]*h.d),t[a]=c+1;let g=c+1>=l.steps.length;return g&&e.add(a),{kind:"move",to:n[a].slice(),out:g,note:h.note}},done(){return i.doneWhen.every(a=>e.has(a))}}}var st=(i,e=[],t={})=>({steps:[{dir:"+y",d:i,needs:e}],...t});function Jy(){let i=[-1.8,1.8],e=[[-6,-3.5],[0,-3.5]];for(let l of i)e.push([0,l-.4],[-1.5,l-.85],[-1.5,l+.85],[0,l+.4]);e.push([0,3.5],[-6,3.5]);let t=Me("a","\u5DE6\u95E8\u677F",[Be(-6,0,0,.6,-3.5,3.5,2),ca(e,.6,1.2)],"jumu",2),n=Me("b","\u53F3\u95E8\u677F",[Be(0,6,0,.6,-3.5,3.5,2),ca(e.map(([l,c])=>[-l,c]),.6,1.2)],"jumu",2),s=(l,c,h)=>Me(c,h,[ca([[-1.5,l-.85],[0,l-.4],[1.5,l-.85],[1.5,l+.85],[0,l+.4],[-1.5,l+.85]],.6,1.2,{bevel:.02})],"zitan",0),r=s(i[0],"k1","\u94F6\u952D\u6263\xB7\u524D"),a=s(i[1],"k2","\u94F6\u952D\u6263\xB7\u540E"),o={default:"\u94F6\u952D\u4E24\u5934\u5927\u3001\u4E2D\u95F4\u7EC6\uFF0C\u6A2A\u7740\u63A8\uFF0C\u4E24\u5934\u90FD\u88AB\u95E8\u677F\u5361\u4F4F\u3002","-y":"\u5E95\u4E0B\u662F\u95E8\u677F\uFF0C\u538B\u4E0D\u4E0B\u53BB\u3002"};return qt({id:"silver",title:"\u94F6\u952D\u69AB",pieces:[t,n,r,a],rootY:0,camera:{pos:[6,12,11],target:[0,.6,0]},reveal:{pos:[5,13,14],target:[2,1.5,0]},xray:[],labels:[{id:"k1",p:[1.5,1.2,i[0]],text:"\u94F6\u952D\u6263\uFF1A\u4E24\u5934\u5927\uFF0C\u4E2D\u95F4\u7EC6"},{id:"a",p:[-1.5,1.2,i[1]+.85],text:"\u95E8\u677F\u4E0A\u5254\u51FA\u7684\u94F6\u952D\u69FD"}],spec:{a:{fixed:"\u5DE6\u8FB9\u8FD9\u5757\u662F\u95E8\u6846\u90A3\u4FA7\uFF0C\u9489\u5728\u53F0\u4E0A\u4E0D\u52A8\u3002"},k1:{steps:[{dir:"+y",d:3.2}],block:o},k2:{steps:[{dir:"+y",d:3.2}],block:o},b:{steps:[{dir:"+x",d:5,needs:["k1","k2"],needMsg:"\u62C9\u4E0D\u5F00\u3002\u300C{n}\u300D\u50CF\u4E00\u628A\u9501\uFF0C\u4E24\u5934\u5206\u522B\u54AC\u5728\u4E24\u5757\u677F\u91CC\u3002"}],block:{default:"\u4E24\u5757\u677F\u8D34\u7740\uFF0C\u53EA\u80FD\u5F80\u4E24\u8FB9\u5206\u5F00\u3002","+y":"\u5F80\u4E0A\u62AC\u4F1A\u628A\u94F6\u952D\u6263\u8FDE\u7740\u5DE6\u677F\u4E00\u8D77\u5E26\u8D77\u6765\u3002"}}},doneWhen:["b"]})}function Sn(i,e,t,n,s,r="hongmu"){return Me(i,e,[Be(t-.45,t+.45,s,s+.32,n-.45,n+.45,1,.02),Be(t-.62,t+.62,s+.32,s+.82,n-.62,n+.62,1,.03)],r,0)}function xf(){let i=[Be(-1.15,1.15,4,4.55,-1.15,1.15,1),Be(-1.5,1.5,4.55,5.2,-1.5,1.5,1)];for(let e of[-1,1])for(let t of[-1,1]){let n=[e*.4,e*1.5].sort((r,a)=>r-a),s=[t*.4,t*1.5].sort((r,a)=>r-a);i.push(Be(n[0],n[1],5.2,5.6,s[0],s[1],1,.02))}return Me("ludou","\u680C\u6597",i,"huali",0)}function yf(i=4){return Me("nidao","\u6CE5\u9053\u6831",[ft([[-i,5.6],[-i+.4,5.2],[i-.4,5.2],[i,5.6],[i,6.6],[.4,6.6],[.4,5.9],[-.4,5.9],[-.4,6.6],[-i,6.6]],2,-.4,.4)],"jumu",0)}function vf(i,e){return Me("huagong","\u534E\u6831",[ft([[5.6,-i],[5.2,-i+.4],[5.2,-.4],[5.9,-.4],[5.9,.4],[5.2,.4],[5.2,e-.4],[5.6,e],[6.6,e],[6.6,-i]],0,-.4,.4)],"jichi",2)}var En={default:"\u6597\u5E95\u6709\u4E00\u679A\u5C0F\u6728\u9500\u63D2\u8FDB\u4E0B\u9762\u7684\u6831\u91CC\uFF0C\u5E73\u63A8\u63A8\u4E0D\u52A8\uFF0C\u5F97\u5F80\u4E0A\u63D0\u3002","-y":"\u5E95\u4E0B\u662F\u6831\uFF0C\u538B\u4E0D\u4E0B\u53BB\u3002"},ha={default:"\u6831\u5361\u5728\u6597\u53E3\u91CC\uFF0C\u4E24\u8FB9\u90FD\u662F\u6597\u8033\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002","-y":"\u5E95\u4E0B\u662F\u6597\uFF0C\u538B\u4E0D\u4E0B\u53BB\u3002"};function jy(){let e=[Me("col","\u67F1\u5934",[_f(1.35,0,4)],"huali",1),xf(),yf(),vf(4,4),Sn("sd1","\u6563\u6597",-3.4,0,6.6),Sn("sd2","\u6563\u6597",3.4,0,6.6),Sn("jh1","\u4EA4\u4E92\u6597",0,-3.4,6.6),Sn("jh2","\u4EA4\u4E92\u6597",0,3.4,6.6),Sn("qx","\u9F50\u5FC3\u6597",0,0,6.6)],t={col:{fixed:"\u67F1\u5B50\u7ACB\u5728\u5730\u4E0A\uFF0C\u6597\u6831\u662F\u4ECE\u67F1\u5934\u5F80\u4E0A\u4E00\u5C42\u5C42\u645E\u8D77\u6765\u7684\u3002"},sd1:st(6.5,[],{block:En}),sd2:st(6.5,[],{block:En}),jh1:st(6.5,[],{block:En}),jh2:st(6.5,[],{block:En}),qx:st(6.5,[],{block:En}),huagong:st(4,["jh1","jh2","qx"],{block:ha}),nidao:st(2.2,["sd1","sd2","huagong"],{block:ha}),ludou:st(1,["nidao","huagong"],{block:{default:"\u680C\u6597\u5E95\u4E0B\u6709\u6697\u9500\u8FDE\u7740\u67F1\u5934\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}})};return qt({id:"dougong",title:"\u6597\u6831",pieces:e,rootY:0,spec:t,camera:{pos:[11,11,15],target:[0,5.5,0]},reveal:{pos:[15,13,19],target:[0,8.5,0]},xray:[],labels:[{id:"ludou",p:[1.5,5,1.5],text:"\u680C\u6597\uFF1A\u6574\u7EC4\u6597\u6831\u7684\u5E95\u5EA7"},{id:"nidao",p:[-4,6.4,.4],text:"\u6CE5\u9053\u6831\uFF08\u987A\u7740\u5899\uFF09"},{id:"huagong",p:[.4,6.4,4],text:"\u534E\u6831\uFF08\u5411\u5916\u51FA\u6311\uFF09"},{id:"huagong",p:[.4,5.6,0],text:"\u5341\u5B57\u642D\u4EA4\uFF1A\u4E0A\u4E0B\u5404\u5F00\u534A\u53E3"},{id:"qx",p:[.62,7.4,.62],text:"\u9F50\u5FC3\u6597"},{id:"sd2",p:[.62,7.4,.62],text:"\u6563\u6597"}],doneWhen:["ludou"]})}function Ky(){let i=Me("col","\u67F1\u5934",[_f(1.35,0,4)],"huali",1),e=7.4,t=Me("ling","\u4EE4\u6831",[ft([[-4,e+.4],[-3.6,e],[3.6,e],[4,e+.4],[4,e+1.4],[.4,e+1.4],[.4,e+.7],[-.4,e+.7],[-.4,e+1.4],[-4,e+1.4]],2,4,4.8)],"jumu",0),n=Me("shua","\u800D\u5934",[ft([[e,-.6],[e,4],[e+.7,4],[e+.7,4.8],[e,4.8],[e,6],[e+.5,6.6],[e+1.4,6.3],[e+1.4,-.6]],0,-.4,.4)],"hongmu",2),s=[i,xf(),yf(),vf(4,4.8),Sn("sd1","\u6563\u6597",-3.4,0,6.6),Sn("sd2","\u6563\u6597",3.4,0,6.6),Sn("jh1","\u4EA4\u4E92\u6597",0,-3.4,6.6),Sn("jh2","\u4EA4\u4E92\u6597",0,4.4,6.6),Sn("qx","\u9F50\u5FC3\u6597",0,0,6.6),t,Sn("ls1","\u6563\u6597",-3.4,4.4,e+1.4,"jumu"),Sn("ls2","\u6563\u6597",3.4,4.4,e+1.4,"jumu"),n],r={col:{fixed:"\u67F1\u5B50\u4E0D\u52A8\u3002\u4FEE\u6597\u6831\u662F\u4ECE\u6700\u4E0A\u9762\u4E00\u4EF6\u4E00\u4EF6\u5F80\u4E0B\u62C6\uFF0C\u6BCF\u4EF6\u90FD\u8981\u7F16\u53F7\u3002"},shua:st(10,[],{block:{default:"\u800D\u5934\u7684\u4E0B\u53E3\u9A91\u5728\u4EE4\u6831\u4E0A\uFF0C\u5E73\u63A8\u63A8\u4E0D\u52A8\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}}),ls1:st(9,[],{block:En}),ls2:st(9,[],{block:En}),ling:st(7,["ls1","ls2","shua"],{block:ha}),jh2:st(5.5,["ling"],{block:En}),qx:st(5.5,["shua"],{block:En}),sd1:st(5.5,[],{block:En}),sd2:st(5.5,[],{block:En}),jh1:st(5.5,[],{block:En}),huagong:st(3.6,["jh1","jh2","qx"],{block:ha}),nidao:st(2,["sd1","sd2","huagong"],{block:ha}),ludou:st(.9,["nidao","huagong"],{block:{default:"\u680C\u6597\u5E95\u4E0B\u6709\u6697\u9500\u8FDE\u7740\u67F1\u5934\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}})};return qt({id:"dougong2",title:"\u6597\u6831\xB7\u4E24\u8DF3",pieces:s,rootY:0,spec:r,camera:{pos:[12,12,17],target:[0,6.5,1.5]},reveal:{pos:[19,17,25],target:[0,11,1.5]},xray:[],labels:[{id:"shua",p:[0,e+1.4,6.3],text:"\u800D\u5934"},{id:"ling",p:[-4,e+1.2,4.8],text:"\u4EE4\u6831\uFF1A\u7B2C\u4E8C\u8DF3\u7684\u6A2A\u6831"},{id:"huagong",p:[.4,6.2,4.8],text:"\u534E\u6831\uFF1A\u5411\u5916\u51FA\u6311"},{id:"ludou",p:[1.5,5,1.5],text:"\u680C\u6597"},{id:"jh2",p:[.62,7.4,.62],text:"\u4EA4\u4E92\u6597"}],doneWhen:["ludou"]})}function Qy(){let i=[[-1.5,-1.5],[1.5,-1.5],[1.5,-.5],[.3,-.8],[.3,.8],[1.5,.5],[1.5,1.5],[-1.5,1.5],[-1.5,.5],[-.3,.8],[-.3,-.8],[-1.5,-.5]],e=Me("post","\u67F1",[Be(-1.5,1.5,0,6,-1.5,1.5,1),ca(i,6,9)],"huali",1),t=l=>ca([[-1.5,-.5],[-.3,-.8],[-.3,.8],[-1.5,.5]].map(([c,h])=>[c*l,h]),6.02,9),n=Me("eL","\u989D\u678B\xB7\u5DE6",[Be(-10,-1.5,6,9,-1,1,0),t(1)],"jumu",0),s=Me("eR","\u989D\u678B\xB7\u53F3",[Be(1.5,10,6,9,-1,1,0),t(-1)],"jumu",0),r=Me("pb","\u5E73\u677F\u678B",[Be(-10,10,9,9.8,-1.25,1.25,0)],"jichi",0),a=Me("zd","\u5750\u6597",[Be(-1.1,1.1,9.8,10.4,-1.1,1.1,1),Be(-1.45,1.45,10.4,11.2,-1.45,1.45,1)],"hongmu",0),o="\u71D5\u5C3E\u536F\u53E3\u5916\u7A84\u91CC\u5BBD\uFF0C\u989D\u678B\u5F80\u5916\u4E00\u62C9\uFF0C\u71D5\u5C3E\u5C31\u5361\u4F4F\u4E86\u2014\u2014\u8FD9\u5C31\u662F\u5B83\u6297\u62C9\u7684\u9053\u7406\u3002";return qt({id:"beam",title:"\u989D\u678B\u71D5\u5C3E\u69AB",pieces:[e,n,s,r,a],rootY:0,camera:{pos:[10,14,19],target:[0,7.5,0]},reveal:{pos:[9,18,26],target:[0,11,0]},xray:["post"],labels:[{id:"post",p:[-.3,9,.8],text:"\u71D5\u5C3E\u536F\u53E3\uFF08\u67F1\u5934\uFF09"},{id:"eR",p:[.3,9,-.8],text:"\u71D5\u5C3E\u69AB\uFF08\u989D\u678B\u7AEF\u5934\uFF09"},{id:"pb",p:[6,9.8,1.25],text:"\u5E73\u677F\u678B"},{id:"zd",p:[1.45,11.2,1.45],text:"\u5750\u6597"}],spec:{post:{fixed:"\u67F1\u5B50\u7ACB\u7740\u4E0D\u52A8\u3002\u989D\u678B\u662F\u4ECE\u4E0A\u5F80\u4E0B\u300C\u843D\u300D\u8FDB\u67F1\u5934\u7684\u3002"},zd:st(5,[],{block:{default:"\u5750\u6597\u5E95\u4E0B\u6709\u6697\u9500\u8FDE\u7740\u5E73\u677F\u678B\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}}),pb:st(3.4,["zd"],{block:{default:"\u5E73\u677F\u678B\u538B\u5728\u67F1\u5934\u548C\u989D\u678B\u4E0A\uFF0C\u5F97\u5F80\u4E0A\u62AC\u3002"}}),eL:st(1.6,["pb"],{block:{"-x":o,default:"\u989D\u678B\u53EA\u80FD\u987A\u7740\u536F\u53E3\u5F80\u4E0A\u63D0\u51FA\u6765\u3002","-y":"\u5DF2\u7ECF\u843D\u5230\u5E95\u4E86\u3002"}}),eR:st(1.6,["pb"],{block:{"+x":o,default:"\u989D\u678B\u53EA\u80FD\u987A\u7740\u536F\u53E3\u5F80\u4E0A\u63D0\u51FA\u6765\u3002","-y":"\u5DF2\u7ECF\u843D\u5230\u5E95\u4E86\u3002"}})},doneWhen:["eL","eR"]})}function ev(){let i=(o,l,c)=>Me(l,c,[ft([[-4.2,0],[-3.2,0],[-3.2,.4],[-3.5,.4],[-3.5,.7],[-3.2,.7],[-3.2,1.1],[-4.2,1.1]].map(([h,d])=>[h*o,d]),2,-5.3,5.3)],"huali",2),e=(o,l,c)=>Me(l,c,[ft([[0,4.3],[0,5.3],[1.1,5.3],[1.1,4.3],[.7,4.3],[.7,4.6],[.4,4.6],[.4,4.3]].map(([h,d])=>[h,d*o]),0,-3.2,3.2),Be(-3.9,-3.2,.2,.9,o>0?4.5:-5.1,o>0?5.1:-4.5,0,0),Be(3.2,3.9,.2,.9,o>0?4.5:-5.1,o>0?5.1:-4.5,0,0)],"huali",0),t=i(1,"sL","\u5927\u8FB9\xB7\u5DE6"),n=i(-1,"sR","\u5927\u8FB9\xB7\u53F3"),s=e(1,"rF","\u62B9\u5934\xB7\u524D"),r=e(-1,"rB","\u62B9\u5934\xB7\u540E"),a=Me("core","\u82AF\u677F",[Be(-3.45,3.45,.42,.68,-4.55,4.55,2,.02)],"jichi",2);return qt({id:"panel",title:"\u6512\u8FB9\u6253\u69FD\u88C5\u677F",pieces:[t,n,s,r,a],rootY:0,camera:{pos:[9,14,12],target:[0,.5,0]},reveal:{pos:[12,17,16],target:[5,.5,0]},xray:["sL","sR","rF","rB"],labels:[{id:"core",p:[0,.68,0],text:"\u82AF\u677F\uFF1A\u4E0D\u4E0A\u80F6\uFF0C\u6D6E\u5728\u69FD\u91CC"},{id:"sL",p:[-3.2,.7,2],text:"\u6253\u69FD"},{id:"rF",p:[3.9,.9,4.8],text:"\u62B9\u5934\u7684\u69AB\u5934"},{id:"sR",p:[4.2,1.1,0],text:"\u5927\u8FB9"}],spec:{sL:{fixed:"\u5DE6\u8FB9\u5927\u8FB9\u8FDE\u7740\u67DC\u8EAB\u7684\u5408\u9875\uFF0C\u4E0D\u52A8\u5B83\u3002"},sR:{steps:[{dir:"+x",d:12}],block:{default:"\u5927\u8FB9\u53EA\u80FD\u987A\u7740\u69AB\u5934\u7684\u65B9\u5411\u5F80\u5916\u9000\u3002"}},rF:{steps:[{dir:"+x",d:8,needs:["sR"],needMsg:"\u62B9\u5934\u4E24\u5934\u7684\u69AB\u63D2\u5728\u5927\u8FB9\u91CC\uFF0C\u300C{n}\u300D\u4E0D\u62FF\u6389\u9000\u4E0D\u51FA\u6765\u3002"}],block:{default:"\u82AF\u677F\u5D4C\u5728\u62B9\u5934\u7684\u69FD\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69FD\u6ED1\u3002"}},rB:{steps:[{dir:"+x",d:8,needs:["sR"],needMsg:"\u62B9\u5934\u4E24\u5934\u7684\u69AB\u63D2\u5728\u5927\u8FB9\u91CC\uFF0C\u300C{n}\u300D\u4E0D\u62FF\u6389\u9000\u4E0D\u51FA\u6765\u3002"}],block:{default:"\u82AF\u677F\u5D4C\u5728\u62B9\u5934\u7684\u69FD\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69FD\u6ED1\u3002"}},core:{steps:[{dir:"+x",d:3.4,needs:["sR","rF","rB"],needMsg:"\u82AF\u677F\u56DB\u8FB9\u90FD\u63D2\u5728\u69FD\u91CC\u3002\u5148\u62C6\u300C{n}\u300D\u3002"}],block:{default:"\u82AF\u677F\u56DB\u8FB9\u5D4C\u5728\u69FD\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69FD\u6ED1\u51FA\u53BB\u3002"}}},doneWhen:["core"]})}function tv(){let i=[[0,-1],[0,1],[1.5,1],[1.5,.35],[.6,.6],[.6,-.6],[1.5,-.35],[1.5,-1]],e=[[0,-1],[0,1],[1.5,1],[1.5,.62],[.6,.62],[.6,-.62],[1.5,-.62],[1.5,-1]],t=Me("base","\u4E0B\u56F4\u5B50\uFF08\u9500\u773C\uFF09",[Be(-5,-.8,0,1.5,-1,1,0),ft(i,0,-.8,.6,{bevel:0}),ft(e,0,.6,1.6,{bevel:0}),Be(1.6,5,0,1.5,-1,1,0)],"huali",0),n=ft([[.62,-.58],[.62,.58],[1.5,.34],[1.5,-.34]],0,-.78,.2,{bevel:.01}),s=Me("top","\u4E0A\u56F4\u5B50\uFF08\u683D\u9500\uFF09",[Be(-5,5,1.5,6,-.6,.6,0),n],"huali",0);return qt({id:"zouma",title:"\u8D70\u9A6C\u9500",pieces:[t,s],rootY:0,camera:{pos:[7,8,12],target:[0,2.5,0]},reveal:{pos:[8,9,14],target:[1,4,0]},xray:["base"],labels:[{id:"top",p:[.2,.62,.58],text:"\u683D\u9500\uFF1A\u9500\u5934\u505A\u6210\u71D5\u5C3E"},{id:"base",p:[1.1,1.5,.62],text:"\u9500\u773C\uFF1A\u4E00\u5934\u5BBD\uFF0C\u4E00\u5934\u7A84"}],spec:{base:{fixed:"\u4E0B\u56F4\u5B50\u88C5\u5728\u5E8A\u8EAB\u4E0A\uFF0C\u4E0D\u52A8\u5B83\u3002"},top:{steps:[{dir:"+x",d:1.4,note:"\u300C\u5494\u300D\u4E00\u58F0\u8F7B\u54CD\uFF0C\u9500\u5934\u6ED1\u5230\u4E86\u9500\u773C\u5BBD\u7684\u90A3\u5934\u3002"},{dir:"+y",d:5}],block:(r,a)=>a===0&&r==="+y"?"\u62D4\u4E0D\u8D77\u6765\u3002\u9500\u5934\u662F\u71D5\u5C3E\u5F62\u7684\uFF0C\u5361\u5728\u9500\u773C\u7A84\u7684\u90A3\u5934\u5E95\u4E0B\u3002":a===0&&r==="-x"?"\u5DF2\u7ECF\u63A8\u5230\u5934\u4E86\u3002":a===1&&r==="-x"?"\u63A8\u56DE\u53BB\u5C31\u53C8\u9501\u4E0A\u4E86\u3002":r==="-y"?"\u5E95\u4E0B\u662F\u4E0B\u56F4\u5B50\u3002":"\u56F4\u5B50\u4E24\u4FA7\u8D34\u7740\u5E8A\u8EAB\uFF0C\u524D\u540E\u63A8\u4E0D\u52A8\u3002"}},doneWhen:["top"]})}var Mf={silver:Jy,dougong:jy,dougong2:Ky,beam:Qy,panel:ev,zouma:tv};var nv=i=>i.map(([e,t])=>[t,e]),Ih=(i,e,t,n)=>ft(nv(i),1,e,t,n),Zn=(i,e,t,n=40)=>new Wt(i,i,t-e,n,1).translate(0,(e+t)/2,0);var iv=(i,e,t,n,s)=>new Wt(i,i,t-e,28).rotateZ(Math.PI/2).translate((e+t)/2,n,s),sv=(i,e,t,n,s)=>new Wt(i,i,t-e,28).rotateX(Math.PI/2).translate(s,n,(e+t)/2);function rv(i,e,t,n,s=18){let r=[];for(let a=0;a<=s;a++){let o=t+(n-t)*a/s;r.push([Math.cos(o)*e,Math.sin(o)*e])}for(let a=s;a>=0;a--){let o=t+(n-t)*a/s;r.push([Math.cos(o)*i,Math.sin(o)*i])}return r}function av(){let i=Me("leg","\u817F",[Be(-2,0,0,6,-2,0,1)],"huali",1),e=Me("da","\u5927\u8FB9",[Ih([[-9,-2],[-2,-2],[0,0],[-9,0]],6,7.6)],"huali",0),t=Me("mo","\u62B9\u5934",[Ih([[-2,-9],[0,-9],[0,0],[-2,-2]],6,7.6)],"hongmu",2),n=Me("core","\u9762\u5FC3\u677F",[Be(-8.6,-1.6,6.5,7,-8.6,-1.6,0,.02)],"jumu",0);return qt({id:"zongjiao",title:"\u7CBD\u89D2\u69AB",pieces:[i,e,t,n],rootY:0,camera:{pos:[9,13,10],target:[-3,5,-3]},reveal:{pos:[10,14,12],target:[-3,7,-3]},xray:["da","mo"],labels:[{id:"da",p:[-1,7.6,-1],text:"\u4E09\u9762\u90FD\u662F 45\xB0 \u659C\u89D2\uFF0C\u50CF\u7CBD\u5B50\u7684\u89D2"},{id:"leg",p:[-1,6,0],text:"\u817F\uFF1A\u9876\u7AEF\u51FA\u69AB"},{id:"core",p:[-5,7,-5],text:"\u9762\u5FC3\u677F"}],spec:{leg:{fixed:"\u817F\u7ACB\u5728\u5730\u4E0A\u4E0D\u52A8\u3002\u7CBD\u89D2\u69AB\u662F\u4E09\u6839\u6599\u5728\u4E00\u4E2A\u89D2\u4E0A\u6C47\u5408\u3002"},mo:st(3,[],{block:{default:"\u62B9\u5934\u548C\u5927\u8FB9\u5728\u89D2\u4E0A\u659C\u7740\u78B0\u5934\uFF0C\u6A2A\u63A8\u4F1A\u9876\u5230\u53E6\u4E00\u6839\u3002\u53EA\u80FD\u5148\u5F80\u4E0A\u63D0\u3002"}}),core:{steps:[{dir:"+x",d:7,needs:["mo"],needMsg:"\u9762\u5FC3\u677F\u7684\u8FB9\u63D2\u5728\u300C{n}\u300D\u7684\u69FD\u91CC\u3002"}],block:{default:"\u9762\u5FC3\u677F\u5D4C\u5728\u8FB9\u6846\u7684\u69FD\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69FD\u6ED1\u51FA\u53BB\u3002"}},da:st(1.6,["core","mo"],{block:{default:"\u5927\u8FB9\u53EA\u80FD\u5F80\u4E0A\u4ECE\u817F\u7684\u69AB\u5934\u4E0A\u63D0\u8D77\u6765\u3002"}})},doneWhen:["da"]})}function ov(){let i=a=>[Be(a-.8,a+.8,0,6.2,-.8,.8,1),Be(a-.8,a+.8,6.2,8,-.8,-.26,1),Be(a-.8,a+.8,6.2,8,.26,.8,1),Be(a-.5,a+.5,8,8.5,-.5,.5,1,.01)],e=Me("legL","\u5DE6\u817F",i(-4.5),"huali",1),t=Me("legR","\u53F3\u817F",i(4.5),"huali",1),s=Me("ya","\u7259\u6761\xB7\u7259\u5934",[ft([[-6.5,7.2],[-6.5,8],[6.5,8],[6.5,7.2],[5.6,7.2],[5.2,6.2],[4.8,5.6],[4.2,6.1],[3.8,7.2],[-3.8,7.2],[-4.2,6.1],[-4.8,5.6],[-5.2,6.2],[-5.6,7.2]],2,-.24,.24,{bevel:.02})],"hongmu",0),r=Me("top","\u6848\u9762",[Be(-7.6,7.6,8,8.8,-2.6,2.6,0)],"jumu",0);return qt({id:"jiatou",title:"\u5939\u5934\u69AB",pieces:[e,t,s,r],rootY:0,camera:{pos:[6,9,14],target:[0,5,0]},reveal:{pos:[7,10,16],target:[0,8,0]},xray:["legL","legR"],labels:[{id:"legR",p:[4.5,8,.8],text:"\u817F\u9876\u5F00\u53E3\uFF0C\u5939\u4F4F\u7259\u6761"},{id:"ya",p:[-4.8,5.6,.24],text:"\u7259\u5934"},{id:"top",p:[6,8.8,2.6],text:"\u6848\u9762\u538B\u5728\u817F\u7684\u69AB\u5934\u4E0A"}],spec:{legL:{fixed:"\u817F\u7ACB\u5728\u5730\u4E0A\u4E0D\u52A8\u3002"},legR:{fixed:"\u817F\u7ACB\u5728\u5730\u4E0A\u4E0D\u52A8\u3002"},top:st(4,[],{block:{default:"\u6848\u9762\u538B\u5728\u4E24\u6761\u817F\u7684\u69AB\u5934\u4E0A\uFF0C\u53EA\u80FD\u5F80\u4E0A\u62AC\u3002"}}),ya:st(2.4,["top"],{block:{default:"\u7259\u6761\u5361\u5728\u817F\u9876\u7684\u5F00\u53E3\u91CC\uFF0C\u524D\u540E\u90FD\u88AB\u817F\u5939\u7740\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"},needMsg:""})},doneWhen:["ya"]})}function lv(){let i=Me("stile","\u8FB9\u6883",[Be(-1,1,0,10,-.6,0,1),ft([[-1,0],[1,0],[1,4],[.4,5],[1,6],[1,10],[-1,10],[-1,6],[-.4,5],[-1,4]],2,0,.6)],"huali",1),e=(r,a,o)=>Me(a,o,[Be(r>0?1:-8,r>0?8:-1,4,6,-.6,0,0),ft([[1,4],[8,4],[8,6],[1,6],[.4,5]].map(([l,c])=>[l*r,c]),2,0,.6),Be(r>0?-.6:-1,r>0?1:.6,4.4,5.6,-.28,.28,0,0)],"jumu",0),t=e(1,"rR","\u53F3\u62B9\u5934"),n=e(-1,"rL","\u5DE6\u62B9\u5934"),s=Me("pin","\u6728\u9500",[new Wt(.16,.16,1.5,16).rotateX(Math.PI/2).translate(0,5,-.05)],"zitan",2);return qt({id:"gejian",title:"\u683C\u80A9\u69AB",pieces:[i,t,n,s],rootY:0,camera:{pos:[3,7,15],target:[0,5,0]},reveal:{pos:[3,8,18],target:[0,5,0]},xray:["stile"],labels:[{id:"rR",p:[0,5,.6],text:"\u683C\u80A9\uFF1A\u80A9\u8180\u659C\u5207\u6210 45\xB0"},{id:"pin",p:[0,5,.7],text:"\u6728\u9500\u9501\u4F4F\u4E24\u4E2A\u69AB\u5934"},{id:"rL",p:[-1,4.4,.28],text:"\u69AB\u5934\u85CF\u5728\u8FB9\u6883\u91CC"}],spec:{stile:{fixed:"\u8FB9\u6883\u662F\u95E8\u6846\u7684\u7AD6\u6599\uFF0C\u4E0D\u52A8\u3002"},pin:{steps:[{dir:"+z",d:3}],block:{default:"\u6728\u9500\u6A2A\u7A7F\u8FB9\u6883\u548C\u4E24\u4E2A\u69AB\u5934\uFF0C\u53EA\u80FD\u6CBF\u9500\u5B54\u5F80\u5916\u9000\u3002"}},rR:{steps:[{dir:"+x",d:6,needs:["pin"],needMsg:"\u300C{n}\u300D\u8FD8\u7A7F\u5728\u69AB\u5934\u91CC\uFF0C\u62BD\u4E0D\u51FA\u6765\u3002"}],block:{default:"\u683C\u80A9\u7684\u659C\u89D2\u5D4C\u5728\u8FB9\u6883\u7684 V \u53E3\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69AB\u5934\u62BD\u51FA\u53BB\u3002"}},rL:{steps:[{dir:"-x",d:6,needs:["pin"],needMsg:"\u300C{n}\u300D\u8FD8\u7A7F\u5728\u69AB\u5934\u91CC\uFF0C\u62BD\u4E0D\u51FA\u6765\u3002"}],block:{default:"\u683C\u80A9\u7684\u659C\u89D2\u5D4C\u5728\u8FB9\u6883\u7684 V \u53E3\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u69AB\u5934\u62BD\u51FA\u53BB\u3002"}}},doneWhen:["rR","rL"]})}function cv(){let i=Me("leg","\u684C\u817F",[Be(-.8,.8,0,8,-.8,.8,1)],"huali",1),e=Me("top","\u684C\u9762",[Be(-1,10,8,8.7,-1,5,0)],"jumu",0),t=[],n=o=>[.8+o*5,2.6+Math.pow(o,.7)*5.1];for(let o=0;o<=14;o++){let[l,c]=n(o/14);t.push([l,c+.45])}for(let o=14;o>=0;o--){let[l,c]=n(o/14);t.push([l+.25,c-.45])}let s=Me("brace","\u9738\u738B\u67A8",[ft(t,2,-.3,.3,{bevel:.02}),Be(.2,.8,2.3,3.1,-.22,.22,0,0)],"hongmu",0),r=Me("pin","\u9500\u9489",[new Wt(.13,.13,1.4,14).rotateX(Math.PI/2).translate(5.6,7.75,0)],"zitan",2),a=Me("wedge","\u6728\u6954",[ft([[.8,2.05],[1.6,2.05],[1.6,2.15],[.8,2.3]],2,-.2,.2,{bevel:0})],"zitan",0);return qt({id:"bawang",title:"\u9738\u738B\u67A8",pieces:[i,e,s,r,a],rootY:0,camera:{pos:[9,6,13],target:[3,5,0]},reveal:{pos:[10,7,15],target:[4,5,0]},xray:["leg"],labels:[{id:"brace",p:[.5,2.7,.3],text:"\u52FE\u6302\u69AB\uFF1A\u5148\u63D2\u8FDB\u53BB\uFF0C\u518D\u5F80\u4E0A\u4E00\u63A8\u5C31\u52FE\u4F4F"},{id:"wedge",p:[1.6,2.1,.2],text:"\u6728\u6954\uFF1A\u57AB\u4F4F\uFF0C\u4E0D\u8BA9\u5B83\u6389\u4E0B\u6765"},{id:"pin",p:[5.6,7.75,.7],text:"\u4E0A\u7AEF\u7528\u9500\u9489\u8FDE\u7740\u7A7F\u5E26"}],spec:{leg:{fixed:"\u684C\u817F\u4E0D\u52A8\u3002"},top:{fixed:"\u684C\u9762\u4E0D\u52A8\u2014\u2014\u6211\u4EEC\u53EA\u62C6\u5E95\u4E0B\u90A3\u6839\u5F2F\u67A8\u3002"},pin:{steps:[{dir:"-z",d:2.5}],block:{default:"\u9500\u9489\u53EA\u80FD\u6CBF\u7740\u9500\u5B54\u9000\u51FA\u6765\u3002"}},wedge:{steps:[{dir:"+x",d:2.5}],block:{default:"\u6728\u6954\u585E\u5728\u67A8\u5B50\u811A\u4E0B\u7684\u7F1D\u91CC\uFF0C\u53EA\u80FD\u5F80\u5916\u62BD\u3002"}},brace:{steps:[{dir:"-y",d:.45,needs:["pin","wedge"],needMsg:"\u300C{n}\u300D\u8FD8\u6CA1\u53D6\u6389\uFF0C\u67A8\u5B50\u88AB\u5361\u5F97\u6B7B\u6B7B\u7684\u3002",note:"\u67A8\u5B50\u5F80\u4E0B\u4E00\u843D\uFF0C\u52FE\u6302\u69AB\u8131\u5F00\u4E86\u94A9\u3002"},{dir:"+x",d:4}],block:(o,l)=>l===0&&o==="+x"?"\u62D4\u4E0D\u51FA\u6765\u3002\u69AB\u5934\u50CF\u94A9\u5B50\u4E00\u6837\u52FE\u5728\u817F\u91CC\uFF0C\u5F97\u5148\u5F80\u4E0B\u843D\u4E00\u70B9\u8BA9\u5B83\u8131\u94A9\u3002":l===0?"\u5148\u60F3\u60F3\u8FD9\u6839\u67A8\u5B50\u662F\u600E\u4E48\u300C\u6302\u300D\u4E0A\u53BB\u7684\u3002":"\u8131\u4E86\u94A9\uFF0C\u73B0\u5728\u987A\u7740\u69AB\u773C\u5F80\u5916\u62BD\u3002"}},doneWhen:["brace"]})}function hv(){let i=Me("stone","\u67F1\u9876\u77F3",[Be(-2.4,2.4,0,1.1,-2.4,2.4,0),Zn(1.9,1.1,1.4,48)],"stone",0),e=Me("core","\u67F1\u82AF",[Zn(.88,1.4,9),Zn(.4,9,9.6,20)],"jumu",1),t=(o,l)=>Ih(rv(.9,1.35,o,l),1.4,9),n=Me("shL","\u5305\u9576\xB7\u5DE6",[t(Math.PI/2,Math.PI*1.5)],"huali",1),s=Me("shR","\u5305\u9576\xB7\u53F3",[t(-Math.PI/2,Math.PI/2)],"huali",1),r=Me("hoop","\u94C1\u7B8D",[Zn(1.42,3,3.3),Zn(1.42,7,7.3)],"iron",1),a=Me("beam","\u6881",[Be(-6,6,9,11.2,-1.2,1.2,0)],"jichi",0);return qt({id:"guanjiao",title:"\u7BA1\u811A\u69AB \xB7 \u9992\u5934\u69AB",pieces:[i,e,n,s,r,a],rootY:0,camera:{pos:[9,9,13],target:[0,5,0]},reveal:{pos:[10,10,15],target:[0,6,0]},xray:["shL","shR"],labels:[{id:"core",p:[.4,9.6,0],text:"\u9992\u5934\u69AB\uFF1A\u67F1\u9876\u7684\u77ED\u69AB\uFF0C\u63D2\u8FDB\u6881\u5E95"},{id:"core",p:[.88,4,.2],text:"\u67F1\u82AF\uFF1A\u4FBF\u5B9C\u7684\u6749\u6728\uFF01"},{id:"shR",p:[1.35,6,0],text:"\u5916\u9762\u5305\u7684\u6960\u6728\u677F"},{id:"stone",p:[1.9,1.4,0],text:"\u67F1\u9876\u77F3\uFF1A\u5E95\u4E0B\u85CF\u7740\u7BA1\u811A\u69AB"}],spec:{stone:{fixed:"\u67F1\u9876\u77F3\u662F\u77F3\u5934\uFF0C\u4E0D\u52A8\u3002"},core:{fixed:"\u67F1\u82AF\u5E95\u4E0B\u6709\u7BA1\u811A\u69AB\u63D2\u5728\u67F1\u9876\u77F3\u91CC\u3002\u5148\u770B\u770B\u5B83\u5916\u9762\u5305\u7740\u4EC0\u4E48\u3002"},beam:st(3.6,[],{block:{default:"\u6881\u538B\u5728\u67F1\u9876\u7684\u9992\u5934\u69AB\u4E0A\uFF0C\u53EA\u80FD\u5F80\u4E0A\u62AC\u3002"}}),hoop:st(2.4,["beam"],{block:{default:"\u94C1\u7B8D\u5957\u5728\u67F1\u5B50\u4E0A\uFF0C\u53EA\u80FD\u987A\u7740\u67F1\u8EAB\u5F80\u4E0A\u892A\u3002"},needMsg:"\u300C{n}\u300D\u538B\u5728\u67F1\u9876\uFF0C\u94C1\u7B8D\u892A\u4E0D\u51FA\u53BB\u3002"}),shL:{steps:[{dir:"-x",d:3,needs:["hoop"],needMsg:"\u300C{n}\u300D\u8FD8\u7B8D\u7740\uFF0C\u5305\u9576\u677F\u63B0\u4E0D\u5F00\u3002"}],block:{default:"\u8FD9\u662F\u5DE6\u534A\u7247\uFF0C\u5F80\u5DE6\u5265\u3002"}},shR:{steps:[{dir:"+x",d:3,needs:["hoop"],needMsg:"\u300C{n}\u300D\u8FD8\u7B8D\u7740\uFF0C\u5305\u9576\u677F\u63B0\u4E0D\u5F00\u3002"}],block:{default:"\u8FD9\u662F\u53F3\u534A\u7247\uFF0C\u5F80\u53F3\u5265\u3002"}}},doneWhen:["shL","shR"]})}function uv(){let i=Me("col","\u67F1",[Zn(1.2,0,9)],"huali",1),e=Me("fang","\u989D\u678B",[Be(-8,8,9,11,-.9,.9,0)],"jumu",0),t=a=>{let o=[[1.1,9],[6,9]];for(let l=0;l<=10;l++){let c=l/10;o.push([6-c*4.9,9-Math.sin(c*Math.PI/2)*2.3-c*.2])}return o.map(([l,c])=>[l*a,c])},n=Me("qR","\u96C0\u66FF\xB7\u53F3",[ft(t(1),2,-.35,.35,{bevel:.02}),Be(.3,1.1,7.4,8.2,-.2,.2,0,0)],"hongmu",0),s=Me("qL","\u96C0\u66FF\xB7\u5DE6",[ft(t(-1),2,-.35,.35,{bevel:.02}),Be(-1.1,-.3,7.4,8.2,-.2,.2,0,0)],"hongmu",0),r=Me("book","\u6697\u8D26",[Be(2.4,3.8,8.1,8.75,-.22,.22,0,.02)],"paper",0);return qt({id:"queti",title:"\u96C0\u66FF",pieces:[i,e,n,s,r],rootY:0,camera:{pos:[4,7,15],target:[0,8,0]},reveal:{pos:[5,9,17],target:[0,9,0]},xray:["qR"],labels:[{id:"qL",p:[-6,9,.35],text:"\u96C0\u66FF\uFF1A\u6258\u5728\u6881\u678B\u4E0B\uFF0C\u7F29\u77ED\u8DE8\u5EA6"},{id:"qR",p:[.3,8.2,.2],text:"\u9760\u683D\u9500\u63D2\u8FDB\u67F1\u5B50"},{id:"book",p:[3.8,8.75,.22],text:"\u96C0\u66FF\u91CC\u638F\u7A7A\u7684\u6697\u683C"}],spec:{col:{fixed:"\u67F1\u5B50\u4E0D\u52A8\u3002"},fang:st(3,[],{block:{default:"\u989D\u678B\u538B\u5728\u96C0\u66FF\u548C\u67F1\u5934\u4E0A\uFF0C\u5148\u628A\u5B83\u62AC\u8D77\u6765\u3002"}}),qR:{steps:[{dir:"+x",d:4,needs:["fang"],needMsg:"\u300C{n}\u300D\u8FD8\u538B\u5728\u4E0A\u9762\u3002"}],block:{default:"\u96C0\u66FF\u9760\u683D\u9500\u63D2\u5728\u67F1\u5B50\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u9500\u5B50\u5F80\u5916\u62D4\u3002"}},qL:{steps:[{dir:"-x",d:4,needs:["fang"],needMsg:"\u300C{n}\u300D\u8FD8\u538B\u5728\u4E0A\u9762\u3002"}],block:{default:"\u96C0\u66FF\u9760\u683D\u9500\u63D2\u5728\u67F1\u5B50\u91CC\uFF0C\u53EA\u80FD\u987A\u7740\u9500\u5B50\u5F80\u5916\u62D4\u3002"}},book:{steps:[{dir:"+z",d:3,needs:["qR"],needMsg:"\u6697\u683C\u85CF\u5728\u300C{n}\u300D\u91CC\u9762\u3002"}],block:{default:"\u628A\u5B83\u62BD\u51FA\u6765\u770B\u770B\u3002"}}},doneWhen:["book","qL"]})}function dv(){let i=Me("frame","\u95E8\u6846",[Be(-6.2,-5.4,0,9.5,-.6,.6,1),Be(5.4,6.2,0,9.5,-.6,.6,1)],"hongmu",1),e=Me("kan","\u4E2D\u69DB",[Be(-6.2,6.2,8.2,9.5,0,.8,0)],"hongmu",0),t=Me("lian","\u8FDE\u6979",[Be(-5.4,5.4,8.3,9.2,-1.1,0,0)],"jumu",0),n=(c,h)=>Me(h,"\u95E8\u7C2A",[new Wt(.6,.6,.9,6).rotateX(Math.PI/2).translate(c,8.75,1.25),new Wt(.2,.2,2,12).rotateX(Math.PI/2).translate(c,8.75,-.2)],"zitan",2),s=n(-2.6,"z1"),r=n(2.6,"z2"),a=(c,h,d,u)=>Me(d,u,[Be(c,h,.6,8.2,-.9,-.3,1),Zn(.18,8.2,8.7,12).translate(c<0?c+.3:h-.3,0,-.6)],"jichi",1),o=a(-5.4,0,"dL","\u5DE6\u95E8\u6247"),l=a(0,5.4,"dR","\u53F3\u95E8\u6247");return qt({id:"menzan",title:"\u95E8\u7C2A",pieces:[i,e,t,s,r,o,l],rootY:0,camera:{pos:[6,7,16],target:[0,5,0]},reveal:{pos:[10,9,14],target:[0,5,-2]},xray:["kan"],labels:[{id:"z2",p:[2.6,9.35,1.7],text:"\u95E8\u7C2A\uFF1A\u4E00\u5934\u662F\u88C5\u9970\uFF0C\u4E00\u5934\u662F\u9500\u5B50"},{id:"lian",p:[5.4,9.2,-1.1],text:"\u8FDE\u6979\uFF1A\u6258\u4F4F\u95E8\u8F74\u7684\u6A2A\u6728"},{id:"dL",p:[-5.1,8.7,-.6],text:"\u95E8\u8F74"}],spec:{frame:{fixed:"\u95E8\u6846\u780C\u5728\u5899\u91CC\uFF0C\u4E0D\u52A8\u3002"},kan:{fixed:"\u4E2D\u69DB\u8FDE\u7740\u95E8\u6846\uFF0C\u4E0D\u52A8\u3002"},z1:{steps:[{dir:"+z",d:3}],block:{default:"\u95E8\u7C2A\u7A7F\u8FC7\u4E2D\u69DB\u548C\u8FDE\u6979\uFF0C\u53EA\u80FD\u5F80\u524D\u62D4\u3002"}},z2:{steps:[{dir:"+z",d:3}],block:{default:"\u95E8\u7C2A\u7A7F\u8FC7\u4E2D\u69DB\u548C\u8FDE\u6979\uFF0C\u53EA\u80FD\u5F80\u524D\u62D4\u3002"}},lian:{steps:[{dir:"-z",d:3,needs:["z1","z2"],needMsg:"\u8FDE\u6979\u88AB\u300C{n}\u300D\u9489\u5728\u4E2D\u69DB\u80CC\u540E\u3002"}],block:{default:"\u8FDE\u6979\u8D34\u5728\u4E2D\u69DB\u80CC\u540E\uFF0C\u53EA\u80FD\u5F80\u91CC\u64A4\u3002"}},dL:{steps:[{dir:"-z",d:4,needs:["lian"],needMsg:"\u95E8\u8F74\u8FD8\u63D2\u5728\u300C{n}\u300D\u7684\u8F74\u5B54\u91CC\u3002"}],block:{default:"\u95E8\u6247\u53EA\u80FD\u5F80\u91CC\u5378\u4E0B\u6765\u3002"}},dR:{steps:[{dir:"-z",d:4,needs:["lian"],needMsg:"\u95E8\u8F74\u8FD8\u63D2\u5728\u300C{n}\u300D\u7684\u8F74\u5B54\u91CC\u3002"}],block:{default:"\u95E8\u6247\u53EA\u80FD\u5F80\u91CC\u5378\u4E0B\u6765\u3002"}}},doneWhen:["dL"]})}function fv(){let i=Me("col","\u89D2\u67F1",[Zn(1.3,0,8)],"huali",1),e=Me("fz","\u6A90\u678B\xB7\u4FA7",[ft([[8,-10],[10.4,-10],[10.4,-.8],[9.2,-.8],[9.2,.8],[10.4,.8],[10.4,2.4],[8,2.4]].map(([s,r])=>[s,r]),0,-.8,.8)],"jumu",2),t=Me("fx","\u6A90\u678B\xB7\u6B63",[ft([[-10,8],[-.8,8],[-.8,9.2],[.8,9.2],[.8,8],[2.4,8],[2.4,10.4],[-10,10.4]],2,-.8,.8)],"jichi",0),n=Me("dou","\u5750\u6597",[Be(-.9,.9,10.4,10.8,-.9,.9,1),Be(-1.2,1.2,10.8,11.6,-1.2,1.2,1)],"hongmu",0);return qt({id:"gutou",title:"\u7B8D\u5934\u69AB",pieces:[i,e,t,n],rootY:0,camera:{pos:[10,14,10],target:[-2,7,-2]},reveal:{pos:[12,16,12],target:[-2,9,-2]},xray:[],labels:[{id:"fx",p:[2.4,10.4,.8],text:"\u7B8D\u5934\uFF1A\u678B\u5B50\u4F38\u51FA\u67F1\u5916\uFF0C\u7B8D\u4F4F\u67F1\u5934"},{id:"fz",p:[.8,9.2,2.4],text:"\u4E24\u6839\u678B\u5341\u5B57\u523B\u534A\uFF0C\u4E0A\u4E0B\u6263\u5728\u4E00\u8D77"}],spec:{col:{fixed:"\u89D2\u67F1\u4E0D\u52A8\u3002"},dou:st(4,[],{block:{default:"\u5750\u6597\u538B\u5728\u4E24\u6839\u678B\u7684\u4EA4\u53C9\u70B9\u4E0A\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}}),fx:st(2.6,["dou"],{block:{default:"\u7B8D\u5934\u4F38\u51FA\u67F1\u5916\uFF0C\u4E24\u6839\u678B\u5341\u5B57\u54AC\u7740\uFF0C\u6A2A\u7740\u62C9\u4E0D\u8131\u3002\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"}}),fz:st(1.3,["fx"],{block:{default:"\u5B83\u5728\u4E0B\u9762\u90A3\u5C42\uFF0C\u53EA\u80FD\u5F80\u4E0A\u63D0\u3002"},needMsg:"\u300C{n}\u300D\u6263\u5728\u5B83\u4E0A\u9762\u3002"})},doneWhen:["fz"]})}function pv(){let i=Me("cL","\u67F1\xB7\u5DE6",[Zn(.9,0,10)],"huali",1),e=Me("cR","\u67F1\xB7\u53F3",[Zn(.9,0,10)],"huali",1);i.group.children[0].geometry.translate(-4,0,0),e.group.children[0].geometry.translate(4,0,0);let t=(l,c,h)=>Me(c,h,[Be(-6.5,6.5,l-.6,l+.6,-.35,.35,0)],"jumu",0),n=t(7,"f1","\u7A7F\u678B\xB7\u4E0A"),s=t(3.5,"f2","\u7A7F\u678B\xB7\u4E0B"),r=(l,c,h)=>Me(h,"\u6728\u9500",[Be(l-.14,l+.14,c-.25,c+.25,-1.4,1.4,2,.01)],"zitan",2),a=[r(-4,7,"p1"),r(4,7,"p2"),r(-4,3.5,"p3"),r(4,3.5,"p4")],o={steps:[{dir:"+z",d:3}],block:{default:"\u6728\u9500\u6A2A\u63D2\u5728\u67F1\u5B50\u548C\u7A7F\u678B\u91CC\uFF0C\u53EA\u80FD\u5F80\u5916\u6572\u51FA\u6765\u3002"}};return qt({id:"chuandou",title:"\u7A7F\u6597 \xB7 \u6728\u9500",pieces:[i,e,n,s,...a],rootY:0,camera:{pos:[4,8,16],target:[0,5,0]},reveal:{pos:[6,9,18],target:[3,5,1]},xray:["cL","cR"],labels:[{id:"f1",p:[6.5,7.6,.35],text:"\u7A7F\u678B\uFF1A\u76F4\u63A5\u7A7F\u8FC7\u67F1\u8EAB"},{id:"p2",p:[4,7.25,1.4],text:"\u6728\u9500\uFF1A\u628A\u7A7F\u678B\u9501\u5728\u67F1\u91CC"},{id:"cL",p:[-4.9,9,0],text:"\u7A7F\u6597\u5F0F\uFF1A\u67F1\u5B50\u76F4\u63A5\u843D\u5730\u627F\u6AA9"}],spec:{cL:{fixed:"\u67F1\u5B50\u843D\u5730\uFF0C\u4E0D\u52A8\u3002"},cR:{fixed:"\u67F1\u5B50\u843D\u5730\uFF0C\u4E0D\u52A8\u3002"},p1:o,p2:o,p3:o,p4:o,f1:{steps:[{dir:"+x",d:13,needs:["p1","p2"],needMsg:"\u300C{n}\u300D\u8FD8\u9501\u7740\u5B83\u3002"}],block:{default:"\u7A7F\u678B\u7A7F\u8FC7\u4E24\u6839\u67F1\u5B50\uFF0C\u53EA\u80FD\u987A\u7740\u5F80\u5916\u62BD\u3002"}},f2:{steps:[{dir:"+x",d:13,needs:["p3","p4"],needMsg:"\u300C{n}\u300D\u8FD8\u9501\u7740\u5B83\u3002"}],block:{default:"\u7A7F\u678B\u7A7F\u8FC7\u4E24\u6839\u67F1\u5B50\uFF0C\u53EA\u80FD\u987A\u7740\u5F80\u5916\u62BD\u3002"}}},doneWhen:["f1","f2"]})}function mv(){let i=[],e={};for(let r=0;r<4;r++){let a=r%2===0?"x":"z";for(let o of[-1,1]){let l=`L${r}${o>0?"a":"b"}`,c=.55+r*.8,h=a==="x"?iv(.55,-5.4,5.4,c,o*4):sv(.55,-5.4,5.4,c,o*4);i.push(Me(l,`\u7B2C${"\u4E00\u4E8C\u4E09\u56DB"[r]}\u5C42\u539F\u6728`,[h],r%2?"jumu":"huali",a==="x"?0:2));let d=r+1<4?[`L${r+1}a`,`L${r+1}b`]:[];e[l]=r===0?{fixed:"\u6700\u5E95\u4E0B\u4E00\u5C42\u662F\u5730\u683F\uFF0C\u4E0D\u52A8\u3002"}:st(2+(4-r)*1.6,d,{block:{default:"\u539F\u6728\u4E24\u5934\u5F00\u4E86\u69FD\u53E3\uFF0C\u548C\u4E0A\u4E0B\u5C42\u54AC\u5728\u4E00\u8D77\uFF0C\u53EA\u80FD\u4E00\u5C42\u5C42\u5F80\u4E0A\u53D6\u3002"},needMsg:"\u4E0A\u4E00\u5C42\u7684\u300C{n}\u300D\u538B\u7740\u5B83\u3002"})}}return qt({id:"jinggan",title:"\u4E95\u5E72\u5F0F",pieces:i,rootY:0,camera:{pos:[10,11,12],target:[0,1.5,0]},reveal:{pos:[11,13,13],target:[0,4,0]},xray:[],labels:[{id:"L3a",p:[4.8,2.9,4],text:"\u539F\u6728\u4E24\u7AEF\u523B\u53E3\uFF0C\u4E0A\u4E0B\u5C42\u5341\u5B57\u54AC\u5408"},{id:"L0a",p:[-3,.55,4.55],text:"\u4E95\u5E72\uFF1A\u50CF\u4E95\u53E3\u7684\u56F4\u680F\u4E00\u6837\u5C42\u5C42\u53E0\u8D77"}],spec:e,doneWhen:["L1a","L1b"]})}var bf={zongjiao:av,jiatou:ov,gejian:lv,bawang:cv,guanjiao:hv,queti:uv,menzan:dv,gutou:fv,chuandou:pv,jinggan:mv};var Sf={...gf,...Mf,...bf};var Qe={ctx:null,muted:!1,get c(){return this.ctx||(this.ctx=new(window.AudioContext||window.webkitAudioContext)),this.ctx},unlock(){try{this.c.state==="suspended"&&this.c.resume()}catch{}},noise(i){let e=this.c,t=e.createBuffer(1,Math.ceil(e.sampleRate*i),e.sampleRate),n=t.getChannelData(0);for(let r=0;r<n.length;r++)n[r]=Math.random()*2-1;let s=e.createBufferSource();return s.buffer=t,s},knock(i=0,e=1,t=1){if(this.muted)return;let n=this.c,s=n.currentTime+i,r=this.noise(.09),a=n.createBiquadFilter(),o=n.createGain();a.type="bandpass",a.frequency.value=1e3*e,a.Q.value=2.5,o.gain.setValueAtTime(.45*t,s),o.gain.exponentialRampToValueAtTime(.001,s+.08),r.connect(a).connect(o).connect(n.destination),r.start(s);let l=n.createOscillator(),c=n.createGain();l.frequency.setValueAtTime(210*e,s),l.frequency.exponentialRampToValueAtTime(110*e,s+.1),c.gain.setValueAtTime(.32*t,s),c.gain.exponentialRampToValueAtTime(.001,s+.12),l.connect(c).connect(n.destination),l.start(s),l.stop(s+.13)},tick(){this.knock(0,2.2,.35)},slide(i){if(this.muted)return;let e=this.c,t=e.currentTime,n=this.noise(i),s=e.createBiquadFilter(),r=e.createGain();s.type="bandpass",s.frequency.value=650,s.Q.value=.7,r.gain.setValueAtTime(1e-4,t),r.gain.linearRampToValueAtTime(.15,t+i*.25),r.gain.linearRampToValueAtTime(1e-4,t+i),n.connect(s).connect(r).connect(e.destination),n.start(t)},page(){if(this.muted)return;let i=this.c,e=i.currentTime,t=this.noise(.18),n=i.createBiquadFilter(),s=i.createGain();n.type="highpass",n.frequency.value=2500,s.gain.setValueAtTime(1e-4,e),s.gain.linearRampToValueAtTime(.05,e+.04),s.gain.exponentialRampToValueAtTime(1e-4,e+.18),t.connect(n).connect(s).connect(i.destination),t.start(e)},gong(){if(this.muted)return;let i=this.c,e=i.currentTime;for(let[t,n]of[[110,.22],[221,.1],[331,.05]]){let s=i.createOscillator(),r=i.createGain();s.frequency.value=t,r.gain.setValueAtTime(1e-4,e),r.gain.linearRampToValueAtTime(n,e+.02),r.gain.exponentialRampToValueAtTime(1e-4,e+4),s.connect(r).connect(i.destination),s.start(e),s.stop(e+4.1)}},stamp(){this.knock(0,.6,1.4),this.knock(.05,.45,.8)},done(){this.knock(0,1.2),this.knock(.16,1.5),this.knock(.34,1)},tone(i,e,t=.2,n="sine",s=0,r=null){if(this.muted)return;let a=this.c,o=a.currentTime+s,l=a.createOscillator(),c=a.createGain();l.type=n,l.frequency.setValueAtTime(i,o),r&&l.frequency.exponentialRampToValueAtTime(r,o+e),c.gain.setValueAtTime(1e-4,o),c.gain.linearRampToValueAtTime(t,o+.01),c.gain.exponentialRampToValueAtTime(1e-4,o+e),l.connect(c).connect(a.destination),l.start(o),l.stop(o+e+.05)},burstNoise(i,e,t,n,s=0,r="bandpass"){if(this.muted)return;let a=this.c,o=a.currentTime+s,l=this.noise(i),c=a.createBiquadFilter(),h=a.createGain();c.type=r,c.frequency.value=e,c.Q.value=t,h.gain.setValueAtTime(n,o),h.gain.exponentialRampToValueAtTime(1e-4,o+i),l.connect(c).connect(h).connect(a.destination),l.start(o)},heartbeat(){this.tone(62,.16,.5,"sine",0,40),this.tone(58,.14,.35,"sine",.2,38)},clock(i){this.knock(0,i?2.6:2,i?.5:.3)},chime(i=!0){(i?[660,880,1320]:[520,390]).forEach((t,n)=>this.tone(t,.6,.08,"triangle",n*.09))},fail(){this.tone(220,1.2,.18,"sawtooth",0,110),this.burstNoise(.6,300,1,.3),this.knock(0,.4,1.5)},doorKnock(){[0,.22,.44,1.1,1.3].forEach(i=>this.knock(i,.55,1.3))},hit(){this.burstNoise(.25,200,.8,.8),this.knock(0,.35,1.6)},whistle(){this.tone(880,1.6,.06,"square",0,860),this.tone(1100,1.6,.04,"square",0,1080)},amb:null,ambient(i){if(this.amb&&this.amb.kind===i)return;if(this.amb){let o=this.amb;o.g.gain.setTargetAtTime(1e-4,this.c.currentTime,.4),setTimeout(()=>{try{o.src.stop(),o.lfo&&o.lfo.stop()}catch{}},1500),this.amb=null}if(!i||this.muted)return;let e=this.c,t=this.noise(4);t.loop=!0;let n=e.createBiquadFilter(),s=e.createGain(),r={rain:["highpass",1800,.5,.05],fire:["bandpass",900,.6,.07],night:["lowpass",500,.7,.05],engine:["lowpass",140,1.2,.22]}[i];if(!r)return;n.type=r[0],n.frequency.value=r[1],n.Q.value=r[2],s.gain.value=1e-4,s.gain.setTargetAtTime(r[3],e.currentTime,.6),t.connect(n).connect(s).connect(e.destination),t.start();let a=null;if(i==="fire"||i==="engine"){a=e.createOscillator();let o=e.createGain();a.frequency.value=i==="fire"?7:11,o.gain.value=r[3]*.6,a.connect(o).connect(s.gain),a.start()}this.amb={kind:i,src:t,g:s,lfo:a}},cue(i){if(i){if(["rain","fire","night","engine"].includes(i))return this.ambient(i);this.ambient(null),i==="knock"?this.doorKnock():i==="hit"?this.hit():i==="train"&&this.whistle()}},tracks:{theme:__A("art/music/theme.mp3"),drama:__A("art/music/drama.mp3"),tense:__A("art/music/tense.mp3"),warm:__A("art/music/warm.mp3")},now:null,music(i){if(this.now!==i){this.now=i;for(let[e,t]of Object.entries(this.tracks)){let n=document.getElementById("mus-"+e);n.src||(n.src=t);let s=!this.muted&&e===i?.42:0;s>0&&n.paused&&(n.volume=0,n.play().catch(()=>{})),gv(n,s)}}},setMuted(i){this.muted=i,i&&this.ambient(null);let e=this.now;this.now=null,e&&this.music(e)}};function gv(i,e){clearInterval(i._f),i._f=setInterval(()=>{let t=e-i.volume;if(Math.abs(t)<.02){i.volume=e,clearInterval(i._f),e===0&&i.pause();return}i.volume=Math.max(0,Math.min(1,i.volume+Math.sign(t)*.02))},60)}var _v=i=>i<.5?4*i*i*i:1-Math.pow(-2*i+2,3)/2,xv=i=>1+2.2*Math.pow(i-1,3)+1.2*Math.pow(i-1,2),wt={msg(){},solved(){},tools(){},moves(){},timer(){},timeout(){}},ji=document.querySelector("#c"),ai=new Al({canvas:ji,antialias:!0,alpha:!0});ai.setPixelRatio(Math.min(devicePixelRatio,2));ai.shadowMap.enabled=!0;ai.shadowMap.type=No;ai.toneMapping=qr;ai.outputColorSpace=Vt;var Fn=new Hi,yv=new Ws(ai);Fn.environment=yv.fromScene(new Ll,.04).texture;Fn.environmentIntensity=.55;var on=new Ht(32,1,.1,400);on.position.set(14,11,16);var da=new Il(on,ji);Object.assign(da,{enableDamping:!0,dampingFactor:.08,enablePan:!1,minDistance:7,maxDistance:70,maxPolarAngle:1.48});Fn.add(new zr(16773596,5916208,.6));var Qi=new Us(16770239,2.3);Qi.position.set(-9,22,11);Qi.castShadow=!0;Qi.shadow.mapSize.set(2048,2048);Object.assign(Qi.shadow.camera,{left:-22,right:22,top:22,bottom:-22,near:1,far:70});Qi.shadow.bias=-4e-4;Qi.shadow.normalBias=.03;Fn.add(Qi);var Af=new Us(14083327,.4);Af.position.set(12,6,-10);Fn.add(Af);{let i=new Gn(60,1.4,38).translate(0,-.7,0),e=new mt(Ul(i,0,.03),Nl("bench",{roughness:.78}));e.receiveShadow=!0,Fn.add(e)}function Oh(){let i=innerWidth,e=innerHeight;ai.setSize(i,e,!1);let t=i>820?360:0,n=i>820?0:Math.round(e*.3);on.aspect=(i+t)/Math.max(1,e+n),on.setViewOffset(i+t,e+n,t,n,i,e),on.updateProjectionMatrix()}addEventListener("resize",Oh);var Ol=[];function $n(i,e,t=_v){return new Promise(n=>Ol.push({t:0,dur:i,fn:e,ez:t,res:n}))}var Bl=i=>$n(i,()=>{}),ua=0;function Rf(i,e=1){let n=new Nt().setFromObject(re.root).getBoundingSphere(new yn),s=new R(...i.pos).sub(new R(...i.target)).normalize(),r=innerWidth>820,a=r?0:Math.round(innerHeight*.3),o=innerWidth-(r?360:0),l=r?innerHeight:innerHeight*.6,c=Math.atan(Math.tan(na.degToRad(on.fov)/2)*l/(innerHeight+a)),h=Math.atan(Math.tan(c)*o/l),d=n.radius/Math.sin(Math.min(c,h))*e;return{target:n.center.clone(),pos:n.center.clone().addScaledVector(s,d)}}function Cf(i,e=1.3){let t=on.position.clone(),n=da.target.clone(),s=i.target.clone(),r=i.pos.clone();return $n(e,a=>{on.position.lerpVectors(t,r,a),da.target.lerpVectors(n,s,a)})}var Js=500,js=new Ut,Pf=new Float32Array(Js*3),If=new Float32Array(Js*3),Lf=new Float32Array(Js),Nh=new Float32Array(Js);js.setAttribute("position",new It(Pf,3));js.setAttribute("color",new It(If,3));js.setAttribute("size",new It(Lf,1));js.setAttribute("alpha",new It(Nh,1));var vv=new tn({transparent:!0,depthWrite:!1,vertexShader:`attribute float size; attribute float alpha; attribute vec3 color; varying float vA; varying vec3 vC;
    void main(){ vA=alpha; vC=color; vec4 mv=modelViewMatrix*vec4(position,1.0); gl_PointSize=size*(300.0/-mv.z); gl_Position=projectionMatrix*mv; }`,fragmentShader:`varying float vA; varying vec3 vC;
    void main(){ vec2 d=gl_PointCoord-0.5; float r=length(d*vec2(1.0,1.8)); if(r>0.5) discard; gl_FragColor=vec4(vC, vA*smoothstep(0.5,0.25,r)); }`}),Df=new br(js,vv);Df.frustumCulled=!1;Fn.add(Df);var Ji=[];function zl(i,e,t=26,n=[0,1,0]){for(let s=0;s<t;s++){Ji.length>=Js&&Ji.shift();let r=e==="shaving",a=r?new Ye().setHSL(.08+Math.random()*.03,.55,.55+Math.random()*.2):new Ye().setHSL(.09,.15,.75+Math.random()*.1),o=r?3+Math.random()*4:.8+Math.random()*1.2,l=new R(n[0]*o*.6+(Math.random()-.5)*o,n[1]*o*.6+Math.random()*o*.8,n[2]*o*.6+(Math.random()-.5)*o);Ji.push({p:i.clone().add(new R((Math.random()-.5)*.8,(Math.random()-.5)*.5,(Math.random()-.5)*.8)),v:l,c:a,life:0,max:r?1.2+Math.random()*.8:.9+Math.random()*.6,size:r?.25+Math.random()*.25:.5+Math.random()*.5,g:r?9:-.4})}}function Mv(i){for(let e=Ji.length-1;e>=0;e--){let t=Ji[e];if(t.life+=i,t.life>t.max||t.p.y<.02){Ji.splice(e,1);continue}t.v.y-=t.g*i,t.v.multiplyScalar(1-1.6*i),t.p.addScaledVector(t.v,i)}for(let e=0;e<Js;e++){let t=Ji[e];if(!t){Nh[e]=0;continue}Pf.set([t.p.x,t.p.y,t.p.z],e*3),If.set([t.c.r,t.c.g,t.c.b],e*3);let n=t.life/t.max;Lf[e]=t.size*(t.g<0?1+n*2:1),Nh[e]=(1-n)*(t.g<0?.45:.95)}for(let e of["position","color","size","alpha"])js.attributes[e].needsUpdate=!0}var bv={x:12076335,y:3108486,z:12883242},ln=new nn;ln.visible=!1;Fn.add(ln);for(let i of Object.keys(bn)){let e=new nn,t=new yi({color:bv[i[1]],depthTest:!1,transparent:!0,opacity:.92}),n=new mt(new Wt(.075,.075,.9,12).translate(0,.45,0),t),s=new mt(new wr(.28,.6,24).translate(0,1.2,0),t),r=new mt(new Wt(.5,.5,1.6,8).translate(0,.8,0),new yi({transparent:!0,opacity:0,depthWrite:!1,depthTest:!1})),a=new nn;for(let o of[n,s,r])o.renderOrder=20,o.userData.dir=i,a.add(o);e.add(a),e.quaternion.setFromUnitVectors(new R(0,1,0),new R(...bn[i])),e.userData={dir:i,inner:a},ln.add(e)}function Nf(){let i=re&&re.selected&&Ki(re.selected);if(!i){ln.visible=!1;return}let e=new Nt().setFromObject(i.group),t=e.getCenter(new R),n=e.getSize(new R).multiplyScalar(.5);for(let s of ln.children){let r=bn[s.userData.dir],a=r.findIndex(o=>o!==0);s.position.copy(t).add(new R(...r).multiplyScalar(n.getComponent(a)+.55))}ln.visible=!0}var re=null,Ki=i=>re.chap.pieces.find(e=>e.id===i),Uf=!1;function Bh(i){Uf=i,i&&Oh()}async function kl(i,{free:e=!1,timer:t=0}={}){re&&Fn.remove(re.root);let n=Sf[i](),s=new nn;s.position.y=n.rootY,n.pieces.forEach(c=>s.add(c.group)),Fn.add(s),re={id:i,chap:n,root:s,mode:"intro",free:e,selected:null,hover:null,busy:!0,history:[],blocks:0,hint:0,xray:!1,moves:0,timer:t,left:t},wt.timer(t||null),ln.visible=!1,Ev(),wt.tools(),wt.moves(0),Cf(Rf(n.camera,.92),1.6);let r=new Nt().setFromObject(s).getCenter(new R),a=n.pieces.map((c,h)=>{let d=new Nt().setFromObject(c.group).getCenter(new R).sub(r);return d.length()<.5&&d.set(Math.cos(h*2.4),0,Math.sin(h*2.4)),d.y=Math.abs(d.y)+1,d.normalize().multiplyScalar(7).add(new R(0,5,0))});n.pieces.forEach((c,h)=>c.group.position.copy(a[h]));let o=n.pieces.map((c,h)=>h).sort((c,h)=>a[c].y-a[h].y),l=re;await Bl(.35);for(let[c,h]of o.entries()){let d=n.pieces[h].group,u=a[h];$n(.75,f=>d.position.copy(u).multiplyScalar(1-f),xv).then(()=>{re===l&&(Qe.knock(0,.9+Math.random()*.4),zl(new Nt().setFromObject(d).getCenter(new R),"dust",6))}),await Bl(n.pieces.length>8?.08:.13)}await Bl(.75),re===l&&(re.mode="puzzle",re.busy=!1,wt.tools())}function $s(i){re.selected&&Ki(re.selected).mat.emissive.setHex(0),re.selected=i,Nf()}async function Ff(i){if(!re||re.busy||re.mode!=="puzzle"||!re.selected)return;let e=re.selected,t=Ki(e),n=t.group,s=re.chap.tryMove(e,i),r=new R(...bn[i]);if(re.busy=!0,s.kind==="blocked"){Qe.knock(),re.blocks++,wt.msg(s.msg+(re.blocks===4?`
\uFF08\u5361\u4F4F\u4E86\uFF1F\u70B9\u300C\u6C42\u6559\u300D\u542C\u542C\u5E08\u7236\u600E\u4E48\u8BF4\u3002\uFF09`:""),"block");let h=n.position.clone(),d=new Nt().setFromObject(n).getCenter(new R).addScaledVector(r,1.2);zl(d,"dust",8,bn[i]),ua=.12,await $n(.09,u=>n.position.copy(h).addScaledVector(r,.16*u)),await $n(.22,u=>n.position.copy(h).addScaledVector(r,.16*(1-u)*Math.cos(u*9))),n.position.copy(h),re.busy=!1;return}let a=n.position.clone(),o=new R(...s.to);re.history.push({id:e,from:a,to:o}),re.moves++,wt.moves(re.moves),ln.visible=!1;let l=s.out?.85:.32;Qe.slide(l);let c=new Nt().setFromObject(n).getCenter(new R);zl(c,"shaving",s.out?22:8,bn[i]),await $n(l,h=>n.position.lerpVectors(a,o,h)),re.busy=!1,s.out?($s(null),wt.msg(`${t.name}\u53D6\u4E0B\u6765\u4E86\u3002`,"ok")):(Nf(),wt.msg(s.note||"\u632A\u52A8\u4E86\u4E00\u4E0B\u3002","ok")),re.chap.done()&&Sv()}async function Sv(){re.mode="reveal",wt.timer(null),$s(null),Qe.done();let i=new Nt().setFromObject(re.root).getCenter(new R);zl(i,"shaving",50),wt.solved(re.id,re.free),await Uh(!0),await Cf(Rf(re.chap.reveal,.95),1.5),Fh(!0),wt.tools()}function Uh(i){let e=re.chap.explode;if(!e)return Promise.resolve();let t={};for(let a of re.history)t[a.id]=a.to;let n=re.chap.pieces.map(a=>({g:a.group,from:a.group.position.clone(),to:i?new R(...e.offsets[a.id]):(t[a.id]||new R).clone()})),s=re.root.position.y,r=i?e.rootY:re.chap.rootY;return $n(1.2,a=>{n.forEach(o=>o.g.position.lerpVectors(o.from,o.to,a)),re.root.position.y=s+(r-s)*a})}async function zh(){if(!re||re.busy||!re.history.length)return;re.busy=!0,Fh(!1),await Uh(!1);let i=re.history.length>6;for(let e of[...re.history].reverse()){let t=Ki(e.id).group;Qe.slide(i?.25:.55),await $n(i?.28:.6,n=>t.position.lerpVectors(e.to,e.from,n))}Qe.knock(0,.9),await Bl(.9);for(let e of re.history){let t=Ki(e.id).group;Qe.slide(i?.25:.55),await $n(i?.28:.6,n=>t.position.lerpVectors(e.from,e.to,n))}await Uh(!0),Fh(!0),re.busy=!1}function kh(){if(re){re.xray=!re.xray;for(let i of re.chap.xray){let e=Ki(i).mat;e.transparent=re.xray,e.opacity=re.xray?.22:1,e.depthWrite=!re.xray,e.needsUpdate=!0}wt.tools()}}function Vh(){return re?re.hint++:null}var Hh=[],Ef=document.querySelector("#labels");function Ev(){Ef.innerHTML="",Hh=re.chap.labels.map(i=>{let e=document.createElement("div");return e.className="tag",e.innerHTML=`<span>${i.text}</span>`,Ef.appendChild(e),{el:e,l:i}})}function Fh(i){Hh.forEach(({el:e},t)=>setTimeout(()=>e.classList.toggle("on",i),i?t*160:0))}var Ys=new R;function Tv(){let i=innerWidth,e=innerHeight;for(let{el:t,l:n}of Hh)Ys.set(...n.p),Ki(n.id).group.localToWorld(Ys),Ys.project(on),t.style.transform=`translate(${(Ys.x*.5+.5)*i-4}px, ${(-Ys.y*.5+.5)*e-12}px)`,t.style.visibility=Ys.z<1?"visible":"hidden"}var Lh=new Gr,Tf=new ae,Zs=null,Fl=null;function Of(i){if(Tf.set(i.clientX/innerWidth*2-1,-(i.clientY/innerHeight)*2+1),Lh.setFromCamera(Tf,on),ln.visible){let t=Lh.intersectObject(ln,!0)[0];if(t)return{dir:t.object.userData.dir}}if(!re)return null;let e=Lh.intersectObject(re.root,!0).find(t=>t.object.isMesh);return e?{piece:e.object.userData.pieceId}:null}ji.addEventListener("pointerdown",i=>{Zs={x:i.clientX,y:i.clientY},Qe.unlock()});ji.addEventListener("pointerup",i=>{if(!Zs||Math.hypot(i.clientX-Zs.x,i.clientY-Zs.y)>6){Zs=null;return}if(Zs=null,!re||re.mode!=="puzzle"||re.busy)return;let e=Of(i);if(e&&e.dir)Ff(e.dir);else if(e&&e.piece){if(re.chap.removed.has(e.piece)){wt.msg("\u8FD9\u5757\u5DF2\u7ECF\u62C6\u4E0B\u6765\u4E86\u3002","tip");return}Qe.tick(),$s(e.piece===re.selected?null:e.piece)}else $s(null)});ji.addEventListener("pointermove",i=>{if(!re||re.mode!=="puzzle"){ji.style.cursor="",re&&(re.hover=null);return}let e=Of(i),t=e&&e.dir?ln.children.find(n=>n.userData.dir===e.dir):null;Fl!==t&&(Fl&&Fl.scale.setScalar(1),t&&t.scale.setScalar(1.3),Fl=t),re.hover=e&&e.piece&&!re.chap.removed.has(e.piece)?e.piece:null,ji.style.cursor=e?"pointer":"grab"});var wf=performance.now(),Dh=0;ai.setAnimationLoop(i=>{let e=Math.min(1,(i-wf)/1e3),t=Math.min(.05,e);wf=i,Dh+=t;for(let s=Ol.length-1;s>=0;s--){let r=Ol[s];r.t+=t;let a=Math.min(1,r.t/r.dur);r.fn(r.ez(a)),a>=1&&(Ol.splice(s,1),r.res())}if(re&&re.timer&&re.mode==="puzzle"&&(re.left-=e,wt.timer(Math.max(0,re.left)),re.left<=0&&(re.mode="failed",$s(null),ln.visible=!1,wt.timeout(re.id))),!Uf)return;if(da.update(t),re)for(let s of re.chap.pieces){let r=s.id===re.selected,a=s.id===re.hover,o=r?.55+.45*Math.sin(Dh*5):a?.35:0;s.mat.emissive.setRGB(.24*o,.12*o,.04*o)}if(ln.visible){let s=.75+on.position.distanceTo(da.target)*.012;for(let r of ln.children)r.userData.inner.scale.setScalar(s*(1+.06*Math.sin(Dh*6)))}Mv(t);let n=null;ua>0&&(ua=Math.max(0,ua-t),n=new R(Math.random()-.5,Math.random()-.5,0).multiplyScalar(ua*1.2),on.position.add(n)),ai.render(Fn,on),n&&on.position.sub(n),re&&Tv()});Oh();var oi={changgeng:{name:"\u987E\u957F\u5E9A",side:"left",bio:"\u9999\u5C71\u5E2E\u5B66\u5F92\uFF0C\u5341\u4E5D\u5C81\u968F\u5E08\u5317\u4E0A\u8425\u5EFA\u7D2B\u7981\u57CE\u3002"},ayuan:{name:"\u5B81\u6C85",side:"right",bio:"\u626C\u5DDE\u753B\u5320\u4E4B\u5973\uFF0C\u5728\u5927\u6BBF\u4E0A\u753B\u5F69\u753B\u3002",aff:!0},han:{name:"\u97E9\u5E08\u5085",side:"right",bio:"\u957F\u5E9A\u7684\u5E08\u7236\uFF0C\u9999\u5C71\u5E2E\u8001\u6728\u5320\uFF0C\u505A\u4E1C\u897F\u4ECE\u4E0D\u4E0A\u80F6\u3002",aff:!0},luqi:{name:"\u9646\u4E03",side:"right",bio:"\u957F\u5E9A\u7684\u5E08\u5144\uFF0C\u624B\u5FEB\uFF0C\u5BB6\u8D2B\uFF0C\u5A18\u5E38\u5E74\u5367\u75C5\u3002",aff:!0},cao:{name:"\u66F9\u5185\u5B98",side:"right",bio:"\u7763\u9020\u5927\u6BBF\u7684\u5185\u5B98\uFF0C\u7BA1\u7740\u6960\u6728\u7684\u8FDB\u51FA\u3002"},shouzhuo:{name:"\u987E\u5B88\u62D9",side:"left",bio:"\u957F\u5E9A\u7684\u540E\u4EBA\uFF0C\u5317\u5E73\u4E1C\u57CE\u6728\u4F5C\u94FA\u638C\u67DC\u3002"},xiuhe:{name:"\u79C0\u79BE",side:"right",bio:"\u5B88\u62D9\u7684\u59BB\u5B50\uFF0C\u5C31\u5FEB\u4E34\u76C6\u3002",aff:!0},zhou:{name:"\u5468\u5148\u751F",side:"right",bio:"\u4E2D\u56FD\u8425\u9020\u5B66\u793E\u7684\u5E74\u8F7B\u7ED8\u56FE\u5458\u3002",aff:!0},sun:{name:"\u5B59\u4E16\u660C",side:"right",bio:"\u79C0\u79BE\u7684\u8868\u54E5\uFF0C\u7409\u7483\u5382\u53E4\u73A9\u94FA\u638C\u67DC\uFF0C\u516B\u9762\u73B2\u73D1\u3002",aff:!0},ono:{name:"\u5C0F\u91CE",side:"right",bio:"\u968F\u519B\u800C\u6765\u7684\u65E5\u672C\u300C\u8C03\u67E5\u73ED\u300D\u5B66\u8005\u3002"},nian:{name:"\u987E\u5FF5",side:"left",bio:"\u4F60\u3002\u53E4\u5EFA\u4FEE\u7F2E\u961F\u5B66\u5F92\uFF0C\u5B88\u7740\u7237\u7237\u7559\u4E0B\u7684\u8001\u5B85\u548C\u6728\u5323\u3002"},lin:{name:"\u6797\u5DE5",side:"right",bio:"\u4FEE\u7F2E\u961F\u7684\u8001\u5DE5\u7A0B\u5E08\uFF0C\u5E26\u4E86\u4F60\u4E09\u5E74\u3002",aff:!0},xu:{name:"\u8BB8\u6674",side:"right",bio:"\u535A\u7269\u9986\u7814\u7A76\u5458\uFF0C\u4F60\u7684\u53D1\u5C0F\u3002",aff:!0},qian:{name:"\u94B1\u603B",side:"right",bio:"\u94B1\u6C0F\u6587\u65C5\u7684\u8001\u677F\uFF0C\u8981\u62C6\u8001\u5B85\u3002"},grandpa:{name:"\u7237\u7237\u7684\u4FE1",side:null}},Wh=["dovetail","silver","through","zongjiao","guanjiao","queti","menzan","dougong","pin","jiatou","beam","panel","chuandou","bawang","zouma","dougong2","jinggan","gutou","gejian","burr"],Bf={dovetail:"\u628A\u9762\u677F\u80CC\u540E\u7684\u7A7F\u5E26\u6728\u6761\u9000\u51FA\u6765\u3002",silver:"\u53D6\u4E0B\u4E24\u53EA\u65E7\u94F6\u952D\u6263\uFF0C\u628A\u53F3\u8FB9\u7684\u95E8\u677F\u62C9\u5F00\u3002",through:"\u62C6\u5F00\u51F3\u817F\u548C\u6A2A\u67A8\u3002",zongjiao:"\u62C6\u5F00\u753B\u7BB1\u7684\u8FD9\u4E2A\u89D2\u3002",guanjiao:"\u5265\u5F00\u70E7\u8FC7\u7684\u67F1\u5B50\uFF0C\u770B\u770B\u91CC\u9762\u662F\u4EC0\u4E48\u6728\u5934\u3002",queti:"\u627E\u5230\u85CF\u5728\u96C0\u66FF\u91CC\u7684\u6697\u8D26\u3002",menzan:"\u5378\u4E0B\u4E00\u6247\u95E8\uFF0C\u4ECE\u89D2\u95E8\u9003\u51FA\u53BB\uFF01",dougong:"\u628A\u8FD9\u7EC4\u6597\u6831\u4E00\u4EF6\u4E00\u4EF6\u62C6\u5230\u5E95\uFF0C\u8BB0\u4E0B\u6BCF\u4E00\u4EF6\u7684\u5C3A\u5BF8\u3002",pin:"\u62C6\u5F00\u5708\u6905\u6276\u624B\u7684\u63A5\u53E3\u3002",jiatou:"\u628A\u5E73\u5934\u6848\u7684\u6848\u9762\u548C\u7259\u6761\u5378\u4E0B\u6765\u3002",beam:"\u628A\u4E24\u6839\u989D\u678B\u4ECE\u67F1\u5934\u4E0A\u53D6\u4E0B\u6765\u3002",panel:"\u62C6\u5F00\u67DC\u95E8\uFF0C\u628A\u56FE\u7EB8\u85CF\u8FDB\u82AF\u677F\u80CC\u540E\uFF01",chuandou:"\u9000\u51FA\u6728\u9500\uFF0C\u62BD\u51FA\u7A7F\u678B\uFF0C\u597D\u6362\u6389\u673D\u574F\u7684\u90A3\u6839\u3002",bawang:"\u628A\u9738\u738B\u67A8\u5378\u4E0B\u6765\uFF0C\u56FE\u7EB8\u8981\u85CF\u8FDB\u684C\u5B50\u91CC\u3002",zouma:"\u628A\u4E0A\u56F4\u5B50\u4ECE\u5E8A\u8EAB\u4E0A\u53D6\u4E0B\u6765\u3002",dougong2:"\u6309\u987A\u5E8F\u843D\u67B6\uFF0C\u62C6\u5378\u8FD9\u7EC4\u4E24\u8DF3\u6597\u6831\u3002",jinggan:"\u4E00\u5C42\u5C42\u62C6\u5F00\u6728\u695E\u623F\u6A21\u578B\uFF0C\u627E\u5230\u6700\u5E95\u4E0B\u7684\u4E1C\u897F\u3002",gutou:"\u5728\u4FDD\u5B89\u56DE\u6765\u4E4B\u524D\uFF0C\u62C6\u5F00\u8F6C\u89D2\uFF0C\u770B\u6E05\u7B8D\u5934\u69AB\u4E0A\u7684\u8BB0\u53F7\uFF01",gejian:"\u62C6\u5F00\u7237\u7237\u7684\u683C\u5B50\u7A97\u3002",burr:"\u6253\u5F00\u6728\u5323\u5E95\u7684\u9C81\u73ED\u9501\uFF01"},Dt=(i,e)=>`\u7B2C${"\u4E00\u4E8C\u4E09"[i-1]}\u5377 \xB7 ${e}`,Vl={dovetail:{title:"\u71D5\u5C3E\u69AB",tag:Dt(1,"\u62BD\u5C49\u7A7F\u5E26"),era:1,text:["\u69AB\u5934\u505A\u6210\u68AF\u5F62\uFF0C\u5916\u5BBD\u5185\u7A84\uFF0C\u5F62\u4F3C\u71D5\u5C3E\u3002\u5782\u76F4\u4E8E\u69AB\u5934\u65B9\u5411\u62C9\u62D4\u65F6\uFF0C\u659C\u9762\u4E92\u76F8\u54AC\u4F4F\uFF0C\u8D8A\u62C9\u8D8A\u7D27\uFF1B\u53EA\u80FD\u987A\u7740\u69FD\u7684\u65B9\u5411\u6ED1\u8FDB\u6ED1\u51FA\u3002","\u5E38\u89C1\u4E8E\u7BB1\u5323\u8F6C\u89D2\u3001\u62BD\u5C49\uFF0C\u4EE5\u53CA\u684C\u9762\u3001\u67DC\u95E8\u80CC\u9762\u7684\u300C\u7A7F\u5E26\u300D\uFF1A\u5E26\u71D5\u5C3E\u7684\u6728\u6761\u6A2A\u7A7F\u9762\u677F\u80CC\u540E\u7684\u71D5\u5C3E\u69FD\uFF0C\u7BA1\u4F4F\u9762\u677F\u4E0D\u7FD8\uFF0C\u53C8\u4E0D\u59A8\u788D\u6728\u5934\u968F\u5B63\u8282\u4F38\u7F29\u3002"]},silver:{title:"\u94F6\u952D\u69AB",tag:Dt(1,"\u5BAB\u95E8\u8865\u7F1D"),era:1,text:["\u5F62\u5982\u4E24\u5934\u5927\u3001\u4E2D\u95F4\u7EC6\u7684\u94F6\u952D\uFF0C\u53C8\u53EB\u94F6\u952D\u6263\u3001\u7EC6\u8170\u3002\u5D4C\u5728\u4E24\u5757\u677F\u7684\u62FC\u7F1D\u4E0A\uFF0C\u4E24\u5934\u5206\u522B\u54AC\u4F4F\u4E24\u5757\u677F\uFF0C\u677F\u5C31\u62C9\u4E0D\u5F00\u4E86\u3002","\u6728\u4F5C\u91CC\u7528\u5B83\u62FC\u677F\u3001\u8865\u88C2\u7F1D\uFF0C\u9632\u6B62\u88C2\u7F1D\u7EE7\u7EED\u6269\u5927\u3002\u53E4\u5EFA\u7B51\u7684\u77F3\u680F\u677F\u3001\u6865\u9762\u77F3\u4E4B\u95F4\uFF0C\u4E5F\u5E38\u7528\u94C1\u6216\u77F3\u505A\u7684\u94F6\u952D\u6263\u8FDE\u63A5\u3002"]},through:{title:"\u900F\u69AB \xB7 \u7834\u5934\u6954",tag:Dt(1,"\u5DE5\u68DA\u957F\u51F3"),era:1,text:["\u69AB\u5934\u7A7F\u900F\u536F\u773C\u3001\u4ECE\u53E6\u4E00\u9762\u9732\u51FA\u5934\u6765\u7684\u53EB\u900F\u69AB\uFF0C\u4E5F\u53EB\u660E\u69AB\uFF1B\u4E0D\u7A7F\u900F\u7684\u53EB\u534A\u69AB\u6216\u6697\u69AB\u3002","\u5728\u900F\u69AB\u7AEF\u5934\u952F\u4E00\u9053\u53E3\uFF0C\u88C5\u597D\u540E\u6253\u8FDB\u6728\u6954\uFF0C\u628A\u69AB\u5934\u6491\u5F00\u3001\u80C0\u7D27\u5728\u536F\u773C\u91CC\uFF0C\u53EB\u300C\u7834\u5934\u6954\u300D\u3002\u62C6\u7684\u65F6\u5019\u53EA\u80FD\u5148\u9000\u6954\u5B50\uFF0C\u518D\u62BD\u69AB\u5934\u3002"]},zongjiao:{title:"\u7CBD\u89D2\u69AB",tag:Dt(1,"\u753B\u7BB1\u4E00\u89D2"),era:1,text:["\u4E09\u6839\u6728\u6599\u5728\u4E00\u4E2A\u89D2\u4E0A\u6C47\u5408\uFF0C\u6BCF\u4E00\u9762\u90FD\u662F 45\xB0 \u659C\u89D2\uFF0C\u5916\u5F62\u50CF\u7CBD\u5B50\u7684\u89D2\uFF0C\u6240\u4EE5\u53EB\u7CBD\u89D2\u69AB\u3002","\u4ECE\u4EFB\u4F55\u4E00\u9762\u770B\u90FD\u4E0D\u9732\u69AB\u5934\uFF0C\u5E72\u51C0\u5229\u843D\uFF0C\u660E\u5F0F\u4E66\u684C\u3001\u753B\u7BB1\u3001\u67DC\u5B50\u7684\u89D2\u4E0A\u5E38\u7528\u3002\u505A\u8D77\u6765\u6700\u8003\u624B\u827A\uFF1A\u4E09\u6839\u6599\u3001\u516D\u4E2A\u9762\uFF0C\u5DEE\u4E00\u70B9\u90FD\u5408\u4E0D\u4E25\u3002"]},guanjiao:{title:"\u7BA1\u811A\u69AB \xB7 \u9992\u5934\u69AB",tag:Dt(1,"\u70E7\u88C2\u7684\u67F1\u5B50"),era:1,arch:!0,text:["\u67F1\u5B50\u5E95\u4E0B\u7684\u77ED\u69AB\u63D2\u8FDB\u67F1\u9876\u77F3\u53EB\u300C\u7BA1\u811A\u69AB\u300D\uFF0C\u7BA1\u4F4F\u67F1\u811A\u4E0D\u6ED1\u52A8\uFF1B\u67F1\u9876\u7684\u77ED\u69AB\u63D2\u8FDB\u6881\u5E95\u53EB\u300C\u9992\u5934\u69AB\u300D\uFF0C\u628A\u6881\u548C\u67F1\u62C9\u5728\u4E00\u8D77\u3002","\u5927\u67F1\u7F3A\u5C11\u6574\u6839\u5927\u6599\u65F6\uFF0C\u4F1A\u7528\u5C0F\u6599\u62FC\u6210\u67F1\u82AF\uFF0C\u5916\u9762\u518D\u5305\u9576\u6728\u677F\u3001\u52A0\u94C1\u7B8D\uFF0C\u53EB\u300C\u5305\u9576\u67F1\u300D\u3002\u8FD9\u672C\u662F\u6B63\u5F53\u505A\u6CD5\u2014\u2014\u53EF\u82E5\u62A5\u7684\u662F\u6574\u6839\u6960\u6728\uFF0C\u91CC\u9762\u5374\u662F\u6749\u6728\uFF0C\u90A3\u5C31\u662F\u5077\u6881\u6362\u67F1\u3002"]},queti:{title:"\u96C0\u66FF",tag:Dt(1,"\u503C\u623F\u6697\u8D26"),era:1,arch:!0,text:["\u96C0\u66FF\u662F\u5B89\u5728\u6881\u678B\u548C\u67F1\u5B50\u76F8\u4EA4\u5904\u3001\u6881\u678B\u5E95\u4E0B\u7684\u6728\u6784\u4EF6\uFF0C\u4ECE\u67F1\u5B50\u91CC\u4F38\u51FA\u6765\u6258\u4F4F\u6881\u678B\uFF0C\u80FD\u7F29\u77ED\u6881\u678B\u7684\u51C0\u8DE8\uFF0C\u4E5F\u51CF\u5C11\u4EA4\u63A5\u5904\u7684\u526A\u529B\u3002","\u5B83\u9760\u683D\u9500\u6216\u69AB\u5934\u63D2\u8FDB\u67F1\u5B50\uFF0C\u4E0A\u9762\u88AB\u6881\u678B\u538B\u4F4F\u3002\u660E\u6E05\u4EE5\u540E\u96C0\u66FF\u8D8A\u505A\u8D8A\u534E\u4E3D\uFF0C\u6210\u4E86\u91CD\u8981\u7684\u88C5\u9970\u6784\u4EF6\u3002"]},menzan:{title:"\u95E8\u7C2A",tag:Dt(1,"\u5BAB\u57CE\u89D2\u95E8"),era:1,arch:!0,text:["\u95E8\u7C2A\u662F\u7A7F\u8FC7\u4E2D\u69DB\u3001\u628A\u95E8\u540E\u7684\u8FDE\u6979\u56FA\u5B9A\u4F4F\u7684\u6728\u9500\uFF0C\u9732\u5728\u95E8\u5916\u7684\u4E00\u5934\u505A\u6210\u516D\u89D2\u3001\u516B\u89D2\u6216\u82B1\u74E3\u5F62\uFF0C\u5E38\u523B\u5409\u7965\u5B57\uFF0C\u6240\u4EE5\u65E2\u662F\u6784\u4EF6\u4E5F\u662F\u88C5\u9970\u3002","\u8FDE\u6979\u4E0A\u5F00\u8F74\u5B54\uFF0C\u6258\u4F4F\u95E8\u6247\u7684\u4E0A\u8F74\u3002\u62D4\u51FA\u95E8\u7C2A\u3001\u64A4\u4E0B\u8FDE\u6979\uFF0C\u95E8\u6247\u5C31\u80FD\u6574\u6247\u5378\u4E0B\u6765\u3002"]},dougong:{title:"\u6597\u6831",tag:Dt(1,"\u5927\u6BBF\u6A90\u4E0B"),era:1,arch:!0,text:["\u6597\u6831\u662F\u67F1\u5934\u4E0E\u5C4B\u6A90\u4E4B\u95F4\u7684\u4E00\u7EC4\u6784\u4EF6\uFF1A\u65B9\u5F62\u7684\u300C\u6597\u300D\u548C\u5F13\u5F62\u7684\u300C\u6831\u300D\u4E00\u5C42\u5C42\u53E0\u8D77\u3001\u5411\u5916\u6311\u51FA\uFF0C\u628A\u5C4B\u6A90\u7684\u91CD\u91CF\u4F20\u5230\u67F1\u5B50\u4E0A\u3002\u5B8B\u300A\u8425\u9020\u6CD5\u5F0F\u300B\u79F0\u300C\u94FA\u4F5C\u300D\uFF0C\u6E05\u4EE3\u79F0\u300C\u6597\u79D1\u300D\u3002","\u6700\u4E0B\u9762\u7684\u5927\u6597\u53EB\u680C\u6597\uFF1B\u987A\u7740\u5899\u7684\u662F\u6CE5\u9053\u6831\uFF0C\u5411\u5916\u51FA\u6311\u7684\u662F\u534E\u6831\uFF0C\u4E24\u8005\u5341\u5B57\u76F8\u4EA4\uFF0C\u4E0A\u4E0B\u5404\u5F00\u4E00\u534A\u53E3\u5B50\u6263\u5728\u4E00\u8D77\u3002\u8282\u70B9\u4E0D\u7528\u9489\uFF0C\u5730\u9707\u65F6\u6784\u4EF6\u4E4B\u95F4\u80FD\u5FAE\u5FAE\u9519\u52A8\u3001\u6D88\u8017\u80FD\u91CF\u3002"]},pin:{title:"\u6954\u9489\u69AB",tag:Dt(2,"\u5708\u6905\u6276\u624B"),era:2,text:["\u5708\u6905\u7684\u6276\u624B\u662F\u4E00\u9053\u5706\u5F27\uFF0C\u8981\u7528\u51E0\u6BB5\u5F2F\u6599\u63A5\u6210\u3002\u63A5\u53E3\u5904\u4E24\u7247\u69AB\u5934\u300C\u5408\u638C\u5F0F\u300D\u5BF9\u642D\uFF0C\u4E24\u5934\u5404\u6709\u5C0F\u820C\u63D2\u8FDB\u5BF9\u65B9\u7684\u69FD\uFF0C\u7BA1\u4F4F\u4E0A\u4E0B\uFF1B\u4E2D\u90E8\u5F00\u65B9\u5B54\uFF0C\u7A7F\u8FDB\u4E00\u6839\u4E00\u5934\u7C97\u4E00\u5934\u7EC6\u7684\u6954\u9489\uFF0C\u7BA1\u4F4F\u5DE6\u53F3\u3002","\u6954\u9489\u4ECE\u7EC6\u5934\u6253\u5165\uFF0C\u8D8A\u6253\u8D8A\u7D27\uFF1B\u62C6\u65F6\u4ECE\u7C97\u5934\u5F80\u5916\u9000\u3002\u738B\u4E16\u8944\u300A\u660E\u5F0F\u5BB6\u5177\u7814\u7A76\u300B\u628A\u5B83\u5217\u4E3A\u5F27\u5F62\u5F2F\u6750\u7684\u5178\u578B\u63A5\u6CD5\u3002"]},jiatou:{title:"\u5939\u5934\u69AB",tag:Dt(2,"\u53E4\u73A9\u94FA\u7684\u5E73\u5934\u6848"),era:2,text:["\u6848\u817F\u9876\u7AEF\u5F00\u53E3\uFF0C\u628A\u7259\u6761\u548C\u7259\u5934\u5939\u5728\u4E2D\u95F4\uFF0C\u817F\u9876\u518D\u51FA\u69AB\u63D2\u8FDB\u6848\u9762\uFF0C\u8FD9\u5C31\u662F\u5939\u5934\u69AB\uFF0C\u662F\u660E\u5F0F\u6848\u7C7B\u5BB6\u5177\u6700\u5E38\u89C1\u7684\u7ED3\u6784\u3002","\u7259\u6761\u3001\u7259\u5934\u628A\u6848\u9762\u7684\u91CD\u91CF\u5206\u6563\u5230\u56DB\u6761\u817F\u4E0A\uFF0C\u53C8\u80FD\u9632\u6B62\u817F\u5F80\u5916\u6487\u3002\u62C6\u65F6\u5148\u5378\u6848\u9762\uFF0C\u7259\u6761\u624D\u80FD\u4ECE\u817F\u7684\u5F00\u53E3\u91CC\u63D0\u51FA\u6765\u3002"]},beam:{title:"\u989D\u678B\u71D5\u5C3E\u69AB",tag:Dt(2,"\u6545\u5BAB\u914D\u6BBF"),era:2,arch:!0,text:["\u989D\u678B\uFF08\u5B8B\u79F0\u9611\u989D\uFF09\u662F\u8FDE\u63A5\u4E24\u6839\u67F1\u5B50\u67F1\u5934\u7684\u6A2A\u6881\u3002\u6E05\u5F0F\u5927\u6728\u4F5C\u91CC\uFF0C\u989D\u678B\u7AEF\u5934\u5E38\u505A\u6210\u71D5\u5C3E\u69AB\uFF0C\u4ECE\u4E0A\u5F80\u4E0B\u300C\u843D\u300D\u8FDB\u67F1\u5934\u7684\u71D5\u5C3E\u536F\u53E3\u2014\u2014\u5916\u7A84\u91CC\u5BBD\uFF0C\u5F80\u5916\u62C9\u53CD\u800C\u62C9\u4E0D\u8131\u3002","\u989D\u678B\u4E4B\u4E0A\u518D\u538B\u5E73\u677F\u678B\u3001\u5750\u6597\uFF0C\u5C42\u5C42\u53E0\u538B\uFF0C\u6574\u5EA7\u6BBF\u7684\u67F1\u5B50\u5C31\u8FD9\u6837\u88AB\u62C9\u7ED3\u6210\u4E00\u4E2A\u6574\u4F53\u3002"]},panel:{title:"\u6512\u8FB9\u6253\u69FD\u88C5\u677F",tag:Dt(2,"\u67DC\u95E8\u5939\u5C42"),era:2,text:["\u56DB\u6839\u6728\u6761\u6512\u6210\u8FB9\u6846\uFF0C\u5185\u4FA7\u5F00\u69FD\uFF0C\u628A\u82AF\u677F\u5D4C\u8FDB\u69FD\u91CC\u3002\u82AF\u677F\u4E0D\u4E0A\u80F6\uFF0C\u56DB\u8FB9\u7559\u6709\u4F59\u5730\uFF0C\u6728\u5934\u968F\u5B63\u8282\u80C0\u7F29\uFF0C\u4E5F\u4E0D\u4F1A\u628A\u6846\u6491\u88C2\u3002","\u67DC\u95E8\u3001\u684C\u9762\u3001\u9694\u6247\u95E8\u591A\u7528\u8FD9\u79CD\u505A\u6CD5\u3002\u53EA\u8981\u62C6\u4E0B\u4E00\u6839\u5927\u8FB9\uFF0C\u6574\u6247\u95E8\u5C31\u80FD\u4E00\u4EF6\u4EF6\u9000\u51FA\u6765\u3002"]},chuandou:{title:"\u7A7F\u6597\u5F0F \xB7 \u6728\u9500",tag:Dt(2,"\u674E\u5E84\u8001\u5B85"),era:2,arch:!0,text:["\u7A7F\u6597\u5F0F\u662F\u5357\u65B9\u5E38\u89C1\u7684\u6728\u6784\u67B6\uFF1A\u67F1\u5B50\u76F4\u63A5\u843D\u5730\u3001\u76F4\u63A5\u627F\u6AA9\uFF0C\u67F1\u4E0E\u67F1\u4E4B\u95F4\u7528\u300C\u7A7F\u678B\u300D\u7A7F\u8FC7\u67F1\u8EAB\u4E32\u8D77\u6765\uFF0C\u8282\u70B9\u7528\u6728\u9500\u9501\u4F4F\u3002\u7528\u6599\u5C0F\u3001\u6574\u4F53\u6027\u597D\uFF0C\u56DB\u5DDD\u3001\u6C5F\u5357\u7684\u6C11\u5C45\u5927\u591A\u5982\u6B64\u3002","\u548C\u5317\u65B9\u5BAB\u6BBF\u7684\u62AC\u6881\u5F0F\u76F8\u6BD4\uFF0C\u7A7F\u6597\u5F0F\u4E0D\u9760\u5927\u6881\u5927\u67F1\uFF0C\u800C\u662F\u4E00\u6392\u6392\u7EC6\u67F1\u548C\u7A7F\u678B\u7EC7\u6210\u4E00\u5F20\u7F51\u3002"]},bawang:{title:"\u9738\u738B\u67A8",tag:Dt(2,"\u65B9\u684C\u5E95\u4E0B"),era:2,text:["\u4ECE\u684C\u817F\u5185\u4FA7\u659C\u4F38\u5230\u684C\u9762\u5E95\u4E0B\u7684\u4E00\u6839\u5F2F\u67A8\uFF0C\u50CF\u529B\u58EB\u4E3E\u81C2\u6258\u4F4F\u684C\u9762\uFF0C\u6240\u4EE5\u53EB\u9738\u738B\u67A8\u3002\u5B83\u8BA9\u684C\u5B50\u4E0D\u7528\u6A2A\u67A8\u4E5F\u7A33\u5F53\uFF0C\u817F\u4E0B\u7A7A\u7A7A\u7684\uFF0C\u597D\u770B\u53C8\u4E0D\u788D\u817F\u3002","\u4E0B\u7AEF\u7528\u300C\u52FE\u6302\u69AB\u300D\uFF1A\u69AB\u5934\u63D2\u8FDB\u817F\u91CC\uFF0C\u5F80\u4E0A\u4E00\u63A8\u5C31\u52FE\u4F4F\uFF0C\u518D\u5728\u4E0B\u9762\u585E\u4E00\u5757\u6728\u6954\u57AB\u6B7B\uFF1B\u4E0A\u7AEF\u7528\u9500\u9489\u8FDE\u7740\u684C\u9762\u4E0B\u7684\u7A7F\u5E26\u3002\u62C6\u65F6\u8981\u5148\u9000\u9500\u3001\u62BD\u6954\uFF0C\u518D\u8BA9\u67A8\u5B50\u5F80\u4E0B\u4E00\u843D\u8131\u94A9\u3002"]},zouma:{title:"\u8D70\u9A6C\u9500",tag:Dt(3,"\u7F57\u6C49\u5E8A\u56F4\u5B50"),era:3,text:["\u4E00\u79CD\u53EF\u62C6\u5378\u7684\u6697\u9500\u3002\u9500\u5B50\u4E00\u7AEF\u683D\u5728\u4E00\u4E2A\u6784\u4EF6\u4E0A\uFF0C\u53E6\u4E00\u7AEF\u505A\u6210\u71D5\u5C3E\u5F62\uFF1B\u53E6\u4E00\u6784\u4EF6\u4E0A\u5F00\u4E00\u5934\u5BBD\u3001\u4E00\u5934\u7A84\u7684\u9500\u773C\u3002\u9500\u5934\u4ECE\u5BBD\u5904\u63D2\u5165\uFF0C\u63A8\u5230\u7A84\u5904\u5C31\u9501\u4F4F\u3002","\u62C6\u65F6\u5148\u53CD\u5411\u63A8\u56DE\u5BBD\u5904\uFF0C\u518D\u5F80\u4E0A\u63D0\u8D77\u3002\u7F57\u6C49\u5E8A\u56F4\u5B50\u3001\u5C4F\u98CE\u8FD9\u7C7B\u8981\u62C6\u5F00\u642C\u8FD0\u7684\u5BB6\u5177\u5E38\u7528\u5B83\u3002"]},dougong2:{title:"\u4E24\u8DF3\u6597\u6831 \xB7 \u843D\u67B6\u4FEE\u7F2E",tag:Dt(3,"\u4FEE\u7F2E\u5DE5\u5730"),era:3,arch:!0,text:["\u534E\u6831\u51FA\u6311\u4E4B\u540E\uFF0C\u5728\u4EA4\u4E92\u6597\u4E0A\u518D\u67B6\u4EE4\u6831\u3001\u538B\u800D\u5934\uFF0C\u4E00\u8DF3\u63A5\u4E00\u8DF3\uFF0C\u5C4B\u6A90\u5C31\u80FD\u4F38\u5F97\u66F4\u8FDC\u3002\u6597\u6831\u8D8A\u590D\u6742\uFF0C\u62C6\u88C5\u7684\u5148\u540E\u987A\u5E8F\u5C31\u8D8A\u4E0D\u80FD\u9519\u3002","\u6587\u7269\u5EFA\u7B51\u4FEE\u7F2E\u65F6\uFF0C\u7CDF\u673D\u7684\u6597\u6831\u8981\u9010\u4EF6\u62CD\u7167\u3001\u7F16\u53F7\u3001\u62C6\u5378\uFF0C\u4FEE\u8865\u540E\u6309\u539F\u4F4D\u88C5\u56DE\uFF0C\u5C3D\u91CF\u4FDD\u7559\u539F\u6709\u7684\u8001\u6784\u4EF6\uFF0C\u8FD9\u662F\u300C\u6700\u5C0F\u5E72\u9884\u300D\u7684\u539F\u5219\u3002"]},jinggan:{title:"\u4E95\u5E72\u5F0F",tag:Dt(3,"\u535A\u7269\u9986\u7684\u6728\u695E\u623F"),era:3,arch:!0,text:["\u7528\u539F\u6728\u5C42\u5C42\u5E73\u53E0\u3001\u5728\u8F6C\u89D2\u5904\u523B\u53E3\u5341\u5B57\u54AC\u5408\uFF0C\u56F4\u6210\u5899\u4F53\uFF0C\u50CF\u53E4\u4EE3\u4E95\u53E3\u7684\u6728\u56F4\u680F\uFF0C\u6240\u4EE5\u53EB\u4E95\u5E72\u5F0F\u3002\u662F\u6700\u53E4\u8001\u7684\u6728\u6784\u505A\u6CD5\u4E4B\u4E00\u3002","\u4ECA\u5929\u5728\u4E91\u5357\u3001\u4E1C\u5317\u6797\u533A\u8FD8\u80FD\u89C1\u5230\u4E95\u5E72\u5F0F\u7684\u6728\u695E\u623F\u3001\u7CAE\u4ED3\u3002\u5B83\u51E0\u4E4E\u4E0D\u9700\u8981\u67F1\u5B50\uFF0C\u5899\u672C\u8EAB\u5C31\u662F\u627F\u91CD\u7ED3\u6784\u3002"]},gutou:{title:"\u7B8D\u5934\u69AB",tag:Dt(3,"\u8F6C\u89D2\u7684\u8BB0\u53F7"),era:3,arch:!0,text:["\u5728\u5EFA\u7B51\u8F6C\u89D2\u5904\uFF0C\u4E24\u4E2A\u65B9\u5411\u7684\u6A90\u678B\u90FD\u8981\u4F38\u8FC7\u89D2\u67F1\uFF0C\u7AEF\u5934\u4E92\u76F8\u5341\u5B57\u523B\u534A\u6263\u5728\u4E00\u8D77\uFF0C\u518D\u4F38\u51FA\u67F1\u5916\u4E00\u622A\uFF0C\u50CF\u4E00\u9053\u7B8D\u7B8D\u4F4F\u67F1\u5934\uFF0C\u8FD9\u5C31\u662F\u7B8D\u5934\u69AB\u3002","\u4F38\u51FA\u6765\u7684\u300C\u7B8D\u5934\u300D\u5E38\u505A\u6210\u9738\u738B\u62F3\u6216\u4E09\u5C94\u5934\u7684\u6837\u5B50\uFF0C\u65E2\u52A0\u56FA\u4E86\u8F6C\u89D2\uFF0C\u53C8\u662F\u4E00\u5904\u88C5\u9970\u3002"]},gejian:{title:"\u683C\u80A9\u69AB",tag:Dt(3,"\u7237\u7237\u7684\u683C\u5B50\u7A97"),era:3,text:["\u6A2A\u7AD6\u4E24\u6839\u6599\u4E01\u5B57\u76F8\u4EA4\u65F6\uFF0C\u628A\u6A2A\u6599\u7684\u80A9\u8180\u5207\u6210 45\xB0 \u659C\u89D2\uFF0C\u5D4C\u8FDB\u7AD6\u6599\u4E0A\u76F8\u5E94\u7684 V \u5F62\u53E3\u91CC\uFF0C\u8FD9\u5C31\u662F\u683C\u80A9\u69AB\u3002\u4ECE\u6B63\u9762\u770B\uFF0C\u4EA4\u63A5\u5904\u662F\u4E00\u9053\u6F02\u4EAE\u7684\u659C\u7EBF\u3002","\u95E8\u7A97\u3001\u683C\u6247\u3001\u684C\u6905\u7684\u6846\u67B6\u4E0A\u5230\u5904\u90FD\u662F\u5B83\u3002\u80A9\u8180\u659C\u7740\u54AC\u8FDB\u53BB\uFF0C\u6BD4\u5E73\u80A9\u66F4\u4E0D\u5BB9\u6613\u9519\u52A8\u3002"]},burr:{title:"\u9C81\u73ED\u9501",tag:Dt(3,"\u6728\u5323"),era:3,text:["\u53C8\u53EB\u5B54\u660E\u9501\u3001\u516D\u5B50\u8054\u65B9\u3002\u516D\u6839\u65B9\u6728\u4E2D\u6BB5\u5F00\u51FA\u6DF1\u6D45\u4E0D\u540C\u7684\u7F3A\u53E3\uFF0C\u4E09\u4E2A\u65B9\u5411\u4E24\u4E24\u7A7F\u63D2\uFF0C\u54AC\u5408\u6210\u4E00\u4E2A\u5341\u5B57\u7ACB\u65B9\uFF0C\u5916\u9762\u770B\u4E0D\u51FA\u4E00\u70B9\u673A\u5173\u3002","\u5176\u4E2D\u5F80\u5F80\u6709\u4E00\u6839\u5B8C\u6574\u6CA1\u6709\u7F3A\u53E3\u7684\u300C\u94A5\u5319\u300D\uFF0C\u6700\u540E\u88C5\u8FDB\u53BB\uFF0C\u6700\u5148\u53D6\u51FA\u6765\u3002\u590D\u6742\u4E00\u4E9B\u7684\u9501\uFF0C\u94A5\u5319\u53D6\u8D70\u540E\u8FD8\u5F97\u8BA9\u51E0\u6839\u6728\u6761\u4EA4\u66FF\u9519\u52A8\u51E0\u683C\uFF0C\u624D\u80FD\u4E00\u6839\u6839\u62C6\u5F00\u3002"]}},zf={dovetail:["\u5F80\u4E0A\u62D4\u8BD5\u8BD5\uFF1F\u71D5\u5C3E\u5916\u5BBD\u5185\u7A84\uFF0C\u5F80\u4E0A\u662F\u62D4\u4E0D\u51FA\u6765\u7684\u3002","\u8FD9\u79CD\u69FD\u4E24\u5934\u90FD\u5F00\u7740\u53E3\u3002","\u987A\u7740\u69FD\u7684\u65B9\u5411\u63A8\u2014\u2014\u9EC4\u8272\u7684\u7BAD\u5934\u3002"],silver:["\u4E24\u5757\u95E8\u677F\u4E4B\u95F4\u5D4C\u7740\u4E24\u4E2A\u6DF1\u8272\u7684\u5C0F\u6728\u5757\u3002","\u94F6\u952D\u6263\u662F\u4ECE\u4E0A\u9762\u5D4C\u8FDB\u53BB\u7684\uFF0C\u4E5F\u53EA\u80FD\u4ECE\u4E0A\u9762\u53D6\u51FA\u6765\uFF08\u84DD\u8272\u7BAD\u5934\u671D\u4E0A\uFF09\u3002","\u4E24\u4E2A\u94F6\u952D\u6263\u90FD\u53D6\u51FA\u6765\u4EE5\u540E\uFF0C\u628A\u53F3\u8FB9\u90A3\u5757\u95E8\u677F\u5F80\u53F3\u62C9\u5F00\u3002"],through:["\u5148\u770B\u69AB\u5934\u7684\u7AEF\u9762\u3002\u90A3\u7247\u6DF1\u8272\u7684\u8584\u6728\u7247\u662F\u505A\u4EC0\u4E48\u7684\uFF1F","\u5148\u9000\u5C0F\u7684\uFF1A\u628A\u6954\u5B50\u671D\u5916\u62D4\u51FA\u6765\uFF08\u7EA2\u8272\u7BAD\u5934\uFF0C\u671D\u5916\u90A3\u652F\uFF09\u3002","\u6954\u5B50\u51FA\u6765\u4E4B\u540E\uFF0C\u628A\u6A2A\u67A8\u5F80\u56DE\u62BD\u3002"],zongjiao:["\u4E09\u6839\u6599\u5728\u89D2\u4E0A\u659C\u7740\u78B0\u5934\u3002\u5148\u52A8\u989C\u8272\u6DF1\u7684\u90A3\u6839\u3002","\u62B9\u5934\u5F80\u4E0A\u4E00\u63D0\uFF0C\u9762\u5FC3\u677F\u5C31\u80FD\u4ECE\u90A3\u4E00\u4FA7\u6ED1\u51FA\u6765\u3002","\u6700\u540E\u628A\u5927\u8FB9\u4ECE\u817F\u4E0A\u63D0\u8D77\u6765\u3002"],guanjiao:["\u6881\u538B\u5728\u67F1\u9876\u4E0A\uFF0C\u5148\u628A\u6881\u62AC\u8D70\u3002","\u94C1\u7B8D\u7B8D\u7740\u5916\u9762\u7684\u5305\u9576\u677F\uFF0C\u987A\u7740\u67F1\u5B50\u5F80\u4E0A\u892A\u6389\u5B83\u3002","\u5DE6\u53F3\u4E24\u7247\u5305\u9576\u677F\uFF0C\u4E00\u7247\u5F80\u5DE6\u3001\u4E00\u7247\u5F80\u53F3\u5265\u5F00\u3002"],queti:["\u989D\u678B\u538B\u5728\u96C0\u66FF\u4E0A\uFF0C\u5148\u62AC\u8D70\u989D\u678B\u3002","\u4E24\u53EA\u96C0\u66FF\u90FD\u662F\u63D2\u5728\u67F1\u5B50\u91CC\u7684\uFF0C\u5F80\u5916\u62D4\u3002","\u53F3\u8FB9\u90A3\u53EA\u91CC\u9762\u662F\u7A7A\u7684\u2014\u2014\u6697\u8D26\u5C31\u85CF\u5728\u91CC\u9762\uFF0C\u5F80\u524D\u62BD\u51FA\u6765\u3002"],menzan:["\u95E8\u7C2A\u65E2\u662F\u88C5\u9970\u4E5F\u662F\u9500\u5B50\uFF0C\u5F80\u524D\u62D4\u6389\u5B83\u4EEC\u3002","\u95E8\u7C2A\u62D4\u6389\u540E\uFF0C\u95E8\u540E\u7684\u8FDE\u6979\u5C31\u80FD\u5F80\u91CC\u64A4\u4E0B\u6765\u3002","\u8FDE\u6979\u4E00\u64A4\uFF0C\u95E8\u8F74\u5C31\u8131\u5F00\u4E86\u2014\u2014\u628A\u5DE6\u95E8\u6247\u5F80\u91CC\u5378\u4E0B\u6765\uFF01"],dougong:["\u6597\u6831\u662F\u4E00\u5C42\u5C42\u645E\u8D77\u6765\u7684\uFF0C\u62C6\u8981\u4ECE\u6700\u4E0A\u9762\u5F00\u59CB\u3002","\u5148\u628A\u6700\u4E0A\u9762\u90A3\u51E0\u4E2A\u5C0F\u65B9\u5757\uFF08\u6597\uFF09\u4E00\u4E2A\u4E2A\u5F80\u4E0A\u63D0\u8D70\u3002","\u534E\u6831\u538B\u5728\u6CE5\u9053\u6831\u4E0A\u9762\uFF0C\u6240\u4EE5\u5148\u63D0\u534E\u6831\uFF0C\u518D\u63D0\u6CE5\u9053\u6831\uFF0C\u6700\u540E\u662F\u680C\u6597\u3002"],pin:["\u4E2D\u95F4\u90A3\u6839\u6DF1\u8272\u7684\u5C0F\u65B9\u9489\u662F\u5173\u952E\u3002","\u6954\u9489\u4E00\u5934\u7C97\u4E00\u5934\u7EC6\uFF0C\u4ECE\u7C97\u7684\u90A3\u5934\u5F80\u5916\u9000\u3002","\u9489\u5B50\u9000\u51FA\u6765\u540E\uFF0C\u628A\u5176\u4E2D\u4E00\u6BB5\u6CBF\u957F\u5EA6\u65B9\u5411\u62C9\u5F00\u3002\u5C0F\u820C\u63D2\u5728\u69FD\u91CC\uFF0C\u4E0A\u4E0B\u52A8\u4E0D\u4E86\u3002"],jiatou:["\u6848\u9762\u538B\u5728\u817F\u9876\u7684\u69AB\u5934\u4E0A\u3002","\u5148\u628A\u6848\u9762\u5F80\u4E0A\u62AC\u8D70\u3002","\u7259\u6761\u5939\u5728\u817F\u9876\u7684\u5F00\u53E3\u91CC\uFF0C\u6848\u9762\u4E00\u8D70\uFF0C\u5B83\u5C31\u80FD\u5F80\u4E0A\u63D0\u51FA\u6765\u3002"],beam:["\u989D\u678B\u662F\u4ECE\u4E0A\u9762\u300C\u843D\u300D\u8FDB\u67F1\u5934\u7684\uFF0C\u62C6\u4E5F\u5F97\u4ECE\u4E0A\u9762\u6765\u3002","\u6700\u4E0A\u9762\u662F\u5750\u6597\uFF0C\u4E0B\u9762\u538B\u7740\u5E73\u677F\u678B\u3002\u5148\u628A\u5B83\u4EEC\u4E00\u5C42\u5C42\u5F80\u4E0A\u63D0\u8D70\u3002","\u5E73\u677F\u678B\u62FF\u8D70\u540E\uFF0C\u4E24\u6839\u989D\u678B\u5C31\u90FD\u80FD\u5F80\u4E0A\u63D0\u51FA\u6765\u4E86\u3002"],panel:["\u5DE6\u8FB9\u90A3\u6839\u5927\u8FB9\u8FDE\u7740\u67DC\u8EAB\uFF0C\u62C6\u7684\u662F\u53F3\u8FB9\u3002","\u5148\u628A\u53F3\u8FB9\u7684\u5927\u8FB9\u5F80\u53F3\u9000\u51FA\u6765\u3002","\u7136\u540E\u4E24\u6839\u62B9\u5934\u5F80\u53F3\u62BD\uFF0C\u6700\u540E\u82AF\u677F\u987A\u7740\u69FD\u6ED1\u51FA\u53BB\u3002"],chuandou:["\u6BCF\u6839\u7A7F\u678B\u4E24\u5934\u90FD\u88AB\u6728\u9500\u9501\u5728\u67F1\u5B50\u91CC\u3002","\u628A\u6728\u9500\u4E00\u6839\u6839\u5F80\u5916\u6572\u51FA\u6765\uFF08\u9EC4\u8272\u7BAD\u5934\uFF09\u3002","\u4E00\u6839\u7A7F\u678B\u7684\u4E24\u4E2A\u9500\u90FD\u9000\u6389\uFF0C\u5B83\u5C31\u80FD\u987A\u7740\u5F80\u53F3\u62BD\u51FA\u53BB\u3002"],bawang:["\u5148\u62C6\u4E24\u5934\u7684\u5C0F\u4E1C\u897F\uFF1A\u4E0A\u9762\u7684\u9500\u9489\uFF0C\u4E0B\u9762\u7684\u6728\u6954\u3002","\u9500\u9489\u5F80\u91CC\u9000\uFF0C\u6728\u6954\u5F80\u5916\u62BD\u3002","\u7136\u540E\u8BA9\u67A8\u5B50\u5F80\u4E0B\u843D\u4E00\u70B9\u8131\u94A9\uFF0C\u518D\u5F80\u5916\u62BD\u51FA\u6765\u3002"],zouma:["\u5F80\u4E0A\u76F4\u63A5\u62D4\u662F\u62D4\u4E0D\u52A8\u7684\u3002","\u5148\u5F80\u65C1\u8FB9\u8BA9\u4E00\u6B65\uFF1A\u5F80\u53F3\u63A8\uFF08\u7EA2\u8272\u7BAD\u5934\uFF09\u3002","\u63A8\u5230\u9500\u773C\u5BBD\u7684\u90A3\u5934\u4E4B\u540E\uFF0C\u518D\u5F80\u4E0A\u63D0\u3002"],dougong2:["\u4FEE\u6597\u6831\u4ECE\u6700\u4E0A\u9762\u62C6\u3002\u6700\u4E0A\u9762\u662F\u54EA\u4E00\u4EF6\uFF1F","\u800D\u5934\u9A91\u5728\u4EE4\u6831\u4E0A\uFF0C\u4EE4\u6831\u4E0A\u8FD8\u5750\u7740\u4E24\u4E2A\u6563\u6597\u3002\u5148\u62FF\u6389\u5B83\u4EEC\u3002","\u7136\u540E\u4EE4\u6831\u3001\u4EA4\u4E92\u6597\u3001\u9F50\u5FC3\u6597\uFF0C\u518D\u5F80\u4E0B\u662F\u534E\u6831\u3001\u6CE5\u9053\u6831\u3001\u680C\u6597\u3002"],jinggan:["\u539F\u6728\u4E00\u5C42\u538B\u4E00\u5C42\uFF0C\u53EA\u80FD\u4ECE\u6700\u4E0A\u9762\u5F00\u59CB\u3002","\u6700\u4E0A\u9762\u4E00\u5C42\u7684\u4E24\u6839\u90FD\u53D6\u8D70\uFF0C\u4E0B\u4E00\u5C42\u624D\u80FD\u52A8\u3002","\u4E00\u5C42\u5C42\u5F80\u4E0A\u53D6\uFF0C\u76F4\u5230\u6700\u5E95\u4E0B\u90A3\u5C42\u9732\u51FA\u6765\u3002"],gutou:["\u5750\u6597\u538B\u5728\u4E24\u6839\u678B\u7684\u4EA4\u53C9\u5904\uFF0C\u5148\u63D0\u8D70\u5B83\u3002","\u4E0A\u9762\u90A3\u6839\u678B\u5F80\u4E0A\u63D0\u3002","\u6700\u540E\u628A\u4E0B\u9762\u90A3\u6839\u4E5F\u63D0\u8D77\u6765\uFF0C\u770B\u6E05\u5B83\u7684\u523B\u53E3\u3002"],gejian:["\u6B63\u4E2D\u90A3\u9897\u6DF1\u8272\u7684\u5C0F\u5706\u70B9\u662F\u6728\u9500\u3002","\u6728\u9500\u5F80\u524D\u9000\u51FA\u6765\u3002","\u4E24\u6839\u62B9\u5934\u4E00\u5DE6\u4E00\u53F3\u62BD\u51FA\u53BB\u3002"],burr:["\u516D\u6839\u91CC\u6709\u4E00\u6839\u6DF1\u8272\u7684\uFF0C\u6CA1\u6709\u4EFB\u4F55\u7F3A\u53E3\u3002\u5B83\u662F\u94A5\u5319\uFF0C\u6700\u5148\u53D6\u3002","\u94A5\u5319\u53D6\u8D70\u540E\uFF0C\u6709\u4E24\u6839\u8981\u4EA4\u66FF\u5F80\u540C\u4E00\u4E2A\u65B9\u5411\u632A\uFF0C\u4E00\u6B21\u4E00\u683C\u3002\u8BD5\u8BD5\u7EA2\u8272\u7684\u7BAD\u5934\u3002","\u632A\u8FC7\u4E09\u8F6E\u4E4B\u540E\uFF0C\u7AD6\u7740\u7684\u90A3\u6839\u5C31\u80FD\u5F80\u4E0A\u53D6\u51FA\u6765\u4E86\u3002"]},kf={true:{title:"\u69AB\u5408",desc:"\u6BBF\u6837\u91CD\u89C1\u5929\u65E5\uFF0C\u5077\u6362\u7684\u6784\u4EF6\u88AB\u8FFD\u56DE\uFF0C\u8001\u5B85\u7559\u4E86\u4E0B\u6765\u3002"},normal:{title:"\u5404\u5B88",desc:"\u4F60\u5B88\u4F4F\u4E86\u624B\u827A\uFF0C\u4F46\u8FD9\u4E00\u6B21\uFF0C\u6CA1\u80FD\u628A\u6240\u6709\u4E1C\u897F\u90FD\u5B88\u4F4F\u3002"},bad:{title:"\u6728\u673D",desc:"\u6728\u5323\u88AB\u62A2\u8D70\u4E86\u3002\u4F60\u624B\u91CC\u53EA\u5269\u4E0B\u4E00\u672C\u624B\u8BB0\u2014\u2014\u548C\u4E09\u53E5\u8BDD\u3002"}},Vf={start:{cut:{era:"\u4E8C\u3007\u4E8C\u516D\u5E74 \xB7 \u51AC",place:"\u5C71\u4E2D\u8001\u5B85",title:"\u5E8F \xB7 \u6728\u5323",bg:"e3_rain"},bg:"e3_rain",sfx:"rain",lines:["\u814A\u6708\uFF0C\u5C71\u91CC\u4E0B\u7740\u51B7\u96E8\u3002","\u4F60\u63A8\u5F00\u8001\u5B85\u7684\u95E8\uFF0C\u95E8\u8F74\u5431\u5440\u4E00\u58F0\uFF0C\u548C\u5C0F\u65F6\u5019\u4E00\u6A21\u4E00\u6837\u3002","\u7237\u7237\u987E\u627F\u8D70\u4E86\u4E09\u4E2A\u6708\u3002\u4ED6\u662F\u8FD9\u4E00\u5E26\u6700\u540E\u4E00\u4E2A\u8001\u6728\u5320\u3002","\u95E8\u677F\u4E0A\u8D34\u7740\u4E00\u5F20\u76D6\u4E86\u7EA2\u7AE0\u7684\u901A\u77E5\uFF1A\u300C\u9650\u4E03\u65E5\u5185\u642C\u79BB\uFF0C\u903E\u671F\u4F9D\u6CD5\u62C6\u9664\u300D\u3002\u95E8\u7F1D\u91CC\u8FD8\u5939\u7740\u4E00\u5F20\u70EB\u91D1\u540D\u7247\u2014\u2014\u94B1\u6C0F\u6587\u65C5\u3002"],next:"p2"},p2:{bg:"e3_box",days:7,lines:["\u5DE5\u4F5C\u53F0\u4E0A\u6446\u7740\u7237\u7237\u7559\u7ED9\u4F60\u7684\u6728\u5323\u3002\u5DF4\u638C\u9AD8\uFF0C\u770B\u4E0D\u5230\u4E00\u9897\u9489\u3001\u4E00\u4E2A\u9501\u5B54\uFF0C\u5323\u5E95\u662F\u516D\u6839\u65B9\u6728\u4EA4\u53C9\u6210\u7684\u5341\u5B57\u3002","\u5323\u5B50\u65C1\u8FB9\u538B\u7740\u4E00\u672C\u53D1\u9EC4\u7684\u518C\u5B50\uFF1A\u300A\u987E\u6C0F\u6728\u4F5C\u624B\u8BB0\u300B\u3002\u6700\u65E9\u7684\u4E00\u79CD\u7B14\u8FF9\u662F\u6BDB\u7B14\u5C0F\u6977\uFF0C\u843D\u6B3E\u300C\u6C38\u4E50\u5341\u4E94\u5E74\uFF0C\u987E\u957F\u5E9A\u300D\u3002","\u7B2C\u4E00\u9875\u53EA\u5199\u7740\u4E00\u53E5\u8BDD\uFF1A\u300C\u6BBF\u6837\u5728\uFF0C\u6BBF\u5C31\u5728\u3002\u300D"],choices:[{label:"\u7FFB\u5F00\u624B\u8BB0",go:"p_flash"},{label:"\u5148\u8BD5\u7740\u6253\u5F00\u6728\u5323",go:"p_try"}]},p_try:{bg:"e3_box",lines:["\u4F60\u634F\u4F4F\u5323\u5E95\u7684\u6728\u6761\uFF0C\u5DE6\u63A8\u53F3\u62C9\uFF0C\u7EB9\u4E1D\u4E0D\u52A8\u3002","\u6307\u5C16\u78B0\u5230\u6728\u5934\u7684\u4E00\u77AC\uFF0C\u4F60\u597D\u50CF\u95FB\u5230\u4E86\u4E00\u80A1\u7126\u5473\u3002"],next:"p_flash"},p_flash:{cg:"flash_a1",still:"e1_fire",caption:"\u706B\u5149\u91CC\uFF0C\u6709\u4EBA\u4ECE\u6BBF\u91CC\u62B1\u51FA\u4E00\u6837\u4E1C\u897F\u2026\u2026",next:"e1cut"},e1cut:{cut:{era:"\u660E \xB7 \u6C38\u4E50\u5341\u4E94\u5E74\uFF081417\uFF09",place:"\u5317\u4EAC",title:"\u7B2C\u4E00\u5377 \xB7 \u6728\u751F",bg:"e1_site",music:"theme"},next:"e1a"},e1a:{bg:"e1_site",lines:["\u6C38\u4E50\u5341\u4E94\u5E74\uFF0C\u5317\u4EAC\u57CE\u91CC\u5806\u6EE1\u4E86\u4ECE\u5357\u65B9\u8FD0\u6765\u7684\u5927\u6728\u3002","\u5341\u4E5D\u5C81\u7684\u987E\u957F\u5E9A\u8DDF\u7740\u5E08\u7236\u97E9\u8001\uFF0C\u4ECE\u82CF\u5DDE\u9999\u5C71\u5317\u4E0A\uFF0C\u6765\u9020\u7687\u5E1D\u7684\u65B0\u5BAB\u6BBF\u3002",["han","\u957F\u5E9A\uFF0C\u770B\u89C1\u90A3\u7247\u5C4B\u9876\u6CA1\u6709\uFF1F\u7B49\u5B83\u76D6\u4E0A\u74E6\uFF0C\u54B1\u4EEC\u8FD9\u4E9B\u4EBA\u7684\u624B\u827A\uFF0C\u5C31\u90FD\u85CF\u5728\u91CC\u5934\u4E86\u3002"],"\u8FDC\u5904\uFF0C\u4E00\u4E2A\u767D\u51C0\u65E0\u987B\u7684\u5185\u5B98\u637B\u7740\u5FF5\u73E0\uFF0C\u6B63\u6328\u6839\u70B9\u9A8C\u521A\u5378\u4E0B\u7684\u6960\u6728\u3002",["han:sad","\u90A3\u662F\u7763\u5DE5\u7684\u66F9\u5185\u5B98\u3002\u79BB\u4ED6\u8FDC\u70B9\u3002\u4ED6\u70B9\u8FC7\u7684\u6728\u5934\uFF0C\u6570\u76EE\u4ECE\u6765\u5BF9\u4E0D\u4E0A\u3002"]],choices:[{label:"\u8DDF\u5E08\u7236\u53BB\u6728\u6599\u573A",go:"e1b_yard",set:{early:!1}},{label:"\u5148\u6E9C\u53BB\u5927\u6BBF\u90A3\u8FB9\u770B\u770B",go:"e1b_hall",set:{early:!0}}]},e1b_yard:{bg:"e1_site",lines:["\u6728\u6599\u573A\u91CC\uFF0C\u4E00\u4E2A\u7CBE\u7626\u7684\u5E74\u8F7B\u5320\u4EBA\u6B63\u625B\u7740\u6728\u5934\uFF0C\u8D70\u5F97\u6BD4\u8C01\u90FD\u5FEB\u3002",["luqi:smile","\u4F60\u5C31\u662F\u5E08\u7236\u65B0\u5E26\u6765\u7684\uFF1F\u6211\u53EB\u9646\u4E03\uFF0C\u4F60\u5F97\u7BA1\u6211\u53EB\u5E08\u5144\u3002"],["luqi","\u6211\u5A18\u75C5\u7740\uFF0C\u6211\u5F97\u591A\u625B\u51E0\u8D9F\u3002\u66F9\u516C\u516C\u90A3\u8FB9\u7ED9\u7684\u5DE5\u94B1\uFF0C\u6BD4\u522B\u5904\u591A\u3002"],["changgeng","\u5E08\u5144\uFF0C\u5E08\u7236\u8BF4\u79BB\u66F9\u516C\u516C\u8FDC\u70B9\u2026\u2026"],["luqi:sad","\u5E08\u7236\u6709\u624B\u827A\uFF0C\u997F\u4E0D\u6B7B\u3002\u6211\u6CA1\u6709\u3002"]],choices:[{label:"\u5E2E\u5E08\u5144\u625B\u4E00\u8D9F",go:"e1b",aff:{luqi:1}},{label:"\u4E0D\u8BF4\u8BDD\uFF0C\u8DDF\u7740\u8D70",go:"e1b"}]},e1b_hall:{bg:"e1_dougong",lines:["\u5927\u6BBF\u6A90\u4E0B\u642D\u7740\u9AD8\u67B6\u3002\u4E00\u4E2A\u7A7F\u9752\u5E03\u886B\u7684\u59D1\u5A18\u8E2E\u7740\u811A\uFF0C\u6B63\u5F80\u6597\u6831\u4E0A\u52FE\u5F69\u753B\u7684\u58A8\u7EBF\u3002",["ayuan:angry","\u5582\u2014\u2014\u4F60\u8E29\u7740\u6211\u7684\u989C\u6599\u789F\u4E86\uFF01"],["changgeng","\u5BF9\u3001\u5BF9\u4E0D\u4F4F\u2026\u2026"],["ayuan:smile","\u4F60\u662F\u6728\u5320\uFF1F\u4F60\u4EEC\u505A\u7684\u8FD9\u4E9B\u6728\u7599\u7629\uFF0C\u4E00\u5C42\u538B\u4E00\u5C42\uFF0C\u6211\u753B\u4E86\u4E09\u5929\u90FD\u6CA1\u6570\u6E05\u6709\u51E0\u5757\u3002"],"\u5979\u53EB\u5B81\u6C85\uFF0C\u8DDF\u7740\u505A\u753B\u5320\u7684\u7236\u4EB2\u4ECE\u626C\u5DDE\u6765\u3002\u8FDC\u5904\u4F20\u6765\u5E08\u7236\u7684\u558A\u58F0\u3002"],choices:[{label:"\u300C\u6539\u5929\u6211\u6570\u7ED9\u4F60\u770B\u3002\u300D",go:"e1b",aff:{ayuan:1}},{label:"\u8D76\u7D27\u8DD1\u56DE\u53BB",go:"e1b"}]},e1b:{bg:"e1_shed",lines:[["han:angry","\u8DD1\u54EA\u513F\u53BB\u4E86\uFF1F\u2014\u2014\u7B97\u4E86\uFF0C\u8FC7\u6765\u3002","early"],["han","\u8FD9\u5757\u62BD\u5C49\u9762\u677F\u80CC\u540E\u7684\u6728\u6761\u53EB\u7A7F\u5E26\u3002\u4F60\u7ED9\u6211\u9000\u51FA\u6765\uFF0C\u770B\u770B\u5B83\u51ED\u4EC0\u4E48\u4E0D\u6389\u3002"],["luqi:smile","\u5E08\u7236\uFF0C\u8FD9\u4E2A\u6211\u95ED\u7740\u773C\u90FD\u4F1A\u3002"],["han","\u6211\u8003\u7684\u662F\u4ED6\u3002"]],puzzle:"dovetail",solved:"e1c"},e1c:{bg:"e1_shed",card:"dovetail",clue:"\u987A\u7740\u5B83\u7684\u8DEF\u8D70",lines:[["han","\u5916\u5BBD\u91CC\u7A84\uFF0C\u53EB\u71D5\u5C3E\u3002\u5F80\u4E0A\u62D4\u5B83\u54AC\u5F97\u8D8A\u7D27\uFF0C\u53EA\u80FD\u987A\u7740\u69FD\u8D70\u3002\u8BB0\u7740\uFF1A\u987A\u7740\u5B83\u7684\u8DEF\u8D70\uFF0C\u6728\u5934\u5C31\u542C\u4F60\u7684\u3002"],"\u9646\u4E03\u5728\u4E00\u65C1\u51B7\u7B11\u4E86\u4E00\u58F0\u3002",["luqi:angry","\u5E08\u7236\u504F\u5FC3\u3002\u6211\u8DDF\u4E86\u60A8\u4E94\u5E74\uFF0C\u60A8\u6CA1\u8FD9\u4E48\u6559\u8FC7\u6211\u3002"]],choices:[{label:"\u300C\u5E08\u5144\uFF0C\u4E0B\u56DE\u4F60\u5148\u6765\u3002\u300D",go:"e1d",aff:{luqi:1}},{label:"\u300C\u624B\u827A\u53C8\u4E0D\u662F\u62A2\u6765\u7684\u3002\u300D",go:"e1d",aff:{luqi:-1,han:1}}]},e1d:{bg:"e1_dougong",lines:["\u5165\u79CB\uFF0C\u4E00\u6247\u65B0\u505A\u7684\u5BAB\u95E8\u95E8\u677F\u88C2\u4E86\u4E00\u9053\u7F1D\uFF0C\u6B63\u597D\u88C2\u5728\u521A\u753B\u597D\u7684\u5F69\u753B\u4E0A\u3002",["ayuan:sad","\u53C8\u662F\u4F60\u554A\uFF0C\u6728\u5320\u3002\u8FD9\u4E0B\u597D\u4E86\uFF0C\u4E09\u5929\u7684\u6D3B\u5168\u767D\u753B\u4E86\u3002","early"],["ayuan:sad","\u8FD9\u4F4D\u6728\u5320\u5E08\u5085\uFF0C\u8FD9\u9053\u7F1D\u8FD8\u8865\u5F97\u597D\u5417\uFF1F\u6211\u753B\u4E86\u4E09\u5929\u3002","!early"],["changgeng","\u8865\u5F97\u597D\u3002"]],choices:[{label:"\u7528\u94C1\u9489\u9489\u4F4F\uFF0C\u5FEB",go:"e1d_nail"},{label:"\u7528\u94F6\u952D\u6263\uFF0C\u6162\u4E9B",go:"e1d_silver",aff:{han:1}}]},e1d_nail:{bg:"e1_dougong",lines:[["han:angry","\u9489\u5B50\uFF1F\u94C1\u4F1A\u9508\uFF0C\u9508\u4E86\u5C31\u80C0\uFF0C\u80C0\u4E86\u5C31\u628A\u6728\u5934\u6491\u88C2\u3002\u5BAB\u91CC\u7684\u95E8\u8981\u7AD9\u51E0\u767E\u5E74\uFF0C\u4F60\u8BA9\u5B83\u7AD9\u51E0\u5E74\uFF1F"],["changgeng:sad","\u2026\u2026\u6211\u9519\u4E86\uFF0C\u5E08\u7236\u3002"]],next:"e1d_silver"},e1d_silver:{bg:"e1_shed",lines:[["han","\u95E8\u677F\u80CC\u9762\u539F\u6765\u5C31\u6709\u4E24\u53EA\u65E7\u94F6\u952D\u6263\u3002\u5148\u62C6\u4E0B\u6765\uFF0C\u770B\u6E05\u5B83\u600E\u4E48\u54AC\u4F4F\u4E24\u5757\u677F\uFF0C\u518D\u7167\u7740\u505A\u65B0\u7684\u3002"]],puzzle:"silver",solved:"e1e"},e1e:{bg:"e1_paint",card:"silver",lines:["\u8865\u597D\u7684\u95E8\u677F\u9001\u56DE\u5F69\u753B\u4F5C\u574A\u3002\u5B81\u6C85\u6478\u4E86\u534A\u5929\uFF0C\u6CA1\u6478\u51FA\u90A3\u9053\u7F1D\u3002",["ayuan:smile","\u4F60\u4EEC\u6728\u5320\uFF0C\u4E0D\u7528\u9489\u5B50\u4E5F\u80FD\u628A\u4E1C\u897F\u7C98\u5728\u4E00\u8D77\uFF1F"],["changgeng","\u4E0D\u662F\u7C98\u3002\u662F\u8BA9\u5B83\u4EEC\u4E92\u76F8\u54AC\u7740\u3002"],["ayuan","\u2026\u2026\u957F\u5E9A\uFF0C\u6211\u8DDF\u4F60\u8BF4\u4EF6\u4E8B\u3002\u6211\u7239\u7BA1\u5F69\u753B\u7684\u8D26\uFF0C\u8D26\u4E0A\u8BB0\u7740\u6960\u6728\u591A\u5C11\u6839\u3001\u591A\u7C97\u3002"],["ayuan:sad","\u53EF\u66F9\u516C\u516C\u62A5\u4E0A\u53BB\u7684\u6960\u6728\uFF0C\u6BD4\u6211\u7239\u8D26\u4E0A\u753B\u8FC7\u7684\uFF0C\u591A\u51FA\u4E00\u500D\u3002"]],choices:[{label:"\u300C\u8FD9\u8BDD\u522B\u518D\u5BF9\u65C1\u4EBA\u8BF4\uFF0C\u5F53\u5FC3\u3002\u300D",go:"e1f",aff:{ayuan:1}},{label:"\u300C\u6211\u53BB\u544A\u8BC9\u5E08\u7236\u3002\u300D",go:"e1f",aff:{han:1}}]},e1f:{bg:"e1_yard",music:"tense",sfx:"night",lines:["\u90A3\u5929\u591C\u91CC\uFF0C\u957F\u5E9A\u7761\u4E0D\u7740\uFF0C\u6478\u9ED1\u53BB\u4E86\u6728\u6599\u573A\u3002","\u706F\u7B3C\u5E95\u4E0B\uFF0C\u51E0\u4E2A\u4EBA\u5F71\u6B63\u628A\u6574\u6839\u7684\u6960\u6728\u62AC\u51FA\u53BB\uFF0C\u53C8\u628A\u4E00\u6837\u7C97\u7EC6\u7684\u6749\u6728\u62AC\u8FDB\u6765\uFF0C\u5237\u4E0A\u4E00\u5C42\u6850\u6CB9\u3002","\u625B\u5728\u6700\u524D\u5934\u7684\u90A3\u4E2A\u4EBA\uFF0C\u662F\u9646\u4E03\u3002"],choices:[{label:"\u5F53\u573A\u558A\u7834",go:"e1f_shout",set:{caoWatch:!0},aff:{luqi:-2}},{label:"\u8EB2\u8D77\u6765\uFF0C\u660E\u5929\u79C1\u4E0B\u95EE\u4ED6",go:"e1f_ask",aff:{luqi:1}}],timer:7,onTimeout:{go:"e1f_ask",note:"\u4F60\u72B9\u8C6B\u4E86\u592A\u4E45\u2014\u2014\u9646\u4E03\u56DE\u5934\uFF0C\u6B63\u597D\u770B\u89C1\u4E86\u4F60\u3002"}},e1f_shout:{bg:"e1_yard",sfx:"hit",lines:[["changgeng:angry","\u4F60\u4EEC\u5728\u5E72\u4EC0\u4E48\uFF01"],"\u706F\u7B3C\u706D\u4E86\u3002\u9ED1\u6697\u91CC\u51E0\u6839\u6728\u68CD\u7838\u4E0B\u6765\uFF0C\u957F\u5E9A\u88AB\u6253\u5012\u5728\u6960\u6728\u5806\u8FB9\u3002",["cao:smile","\u5C0F\u6728\u5320\uFF0C\u591C\u91CC\u773C\u795E\u4E0D\u597D\uFF0C\u770B\u9519\u4E86\u5427\uFF1F"],"\u66F9\u5185\u5B98\u7684\u58F0\u97F3\u4E0D\u7D27\u4E0D\u6162\u3002\u9646\u4E03\u7AD9\u5728\u4ED6\u8EAB\u540E\uFF0C\u4E00\u76F4\u4F4E\u7740\u5934\u3002",["han:angry","\u4F60\u8FD9\u662F\u62FF\u547D\u53BB\u6362\u4E00\u53E5\u75DB\u5FEB\uFF01\u4ECE\u4ECA\u5F80\u540E\uFF0C\u66F9\u5185\u5B98\u7684\u773C\u775B\u5C31\u76EF\u4E0A\u4F60\u4E86\u3002"]],next:"e1g"},e1f_ask:{bg:"e1_shed",lines:[["luqi:sad","\u2026\u2026\u4F60\u90FD\u770B\u89C1\u4E86\u3002"],["luqi:sad","\u6211\u5A18\u7684\u836F\u4E00\u5929\u4E09\u5341\u6587\u3002\u66F9\u516C\u516C\u8BA9\u6211\u642C\uFF0C\u4E00\u665A\u4E0A\u7ED9\u4E00\u94B1\u94F6\u5B50\u3002\u6211\u53EA\u7BA1\u642C\uFF0C\u522B\u7684\u4EC0\u4E48\u90FD\u4E0D\u77E5\u9053\u3002"],["changgeng:sad","\u90A3\u662F\u8981\u780D\u5934\u7684\u4E8B\uFF0C\u5E08\u5144\u3002"]],choices:[{label:"\u300C\u6211\u5E2E\u4F60\u51D1\u836F\u94B1\uFF0C\u4F60\u6536\u624B\u3002\u300D",go:"e1g",aff:{luqi:2},set:{helpLu:!0}},{label:"\u300C\u4F60\u81EA\u5DF1\u60F3\u6E05\u695A\u3002\u300D",go:"e1g"}]},e1g:{bg:"e1_shed",music:"theme",lines:[["han","\u957F\u5E9A\uFF01\u5DE5\u68DA\u90A3\u6761\u957F\u51F3\u7684\u817F\u53C8\u677E\u4E86\uFF0C\u660E\u513F\u4E00\u65E9\u8981\u7528\uFF0C\u4F60\u53BB\u6536\u62FE\u3002"],"\u51F3\u817F\u4E0A\u7684\u900F\u69AB\u7AEF\u5934\u5D4C\u7740\u4E00\u7247\u6DF1\u8272\u6954\u5B50\u3002\u8981\u4FEE\uFF0C\u5F97\u5148\u62C6\u5F00\u3002"],puzzle:"through",solved:"e1h"},e1h:{bg:"e1_shed",card:"through",clue:"\u5148\u9000\u5C0F\u7684\uFF0C\u518D\u9000\u5927\u7684",lines:[["han","\u5148\u9000\u6954\u5B50\uFF0C\u518D\u62BD\u69AB\u5934\u3002\u62C6\u4E1C\u897F\u8DDF\u505A\u4EBA\u4E00\u6837\uFF1A\u5148\u9000\u5C0F\u7684\uFF0C\u518D\u9000\u5927\u7684\u3002"],["han:smile","\u6211\u505A\u4E1C\u897F\u4ECE\u4E0D\u4E0A\u80F6\u3002\u7559\u4E00\u6761\u9000\u8DEF\uFF0C\u540E\u4EBA\u624D\u4FEE\u5F97\u4E86\u3002"]],next:"e1i"},e1i:{bg:"e1_shed",music:"warm",lines:["\u51AC\u5929\u7684\u5DE5\u68DA\uFF0C\u591C\u91CC\u51B7\u5F97\u51FA\u5947\u3002\u5B81\u6C85\u5E38\u6765\u501F\u706B\u70E4\u624B\uFF0C\u987A\u4FBF\u770B\u957F\u5E9A\u5E72\u6D3B\u3002",["ayuan","\u957F\u5E9A\uFF0C\u4F60\u4EEC\u6728\u5320\u4E3A\u4EC0\u4E48\u90FD\u4E0D\u80AF\u7528\u9489\u5B50\uFF1F"]],choices:[{label:"\u300C\u6728\u5934\u4F1A\u547C\u5438\uFF0C\u9489\u5B50\u4E0D\u4F1A\u3002\u300D",go:"e1i_talk",aff:{ayuan:1}},{label:"\u300C\u6211\u7ED9\u4F60\u505A\u4E00\u53EA\u753B\u7BB1\u3002\u300D",go:"e1i_gift",set:{gift:!0}}]},e1i_talk:{bg:"e1_shed",music:"warm",lines:[["changgeng","\u6728\u5934\u51AC\u5929\u7F29\u3001\u590F\u5929\u80C0\uFF0C\u5B83\u662F\u6D3B\u7684\u3002\u9489\u5B50\u662F\u6B7B\u7684\uFF0C\u65E9\u665A\u628A\u5B83\u6491\u574F\u3002\u69AB\u536F\u8DDF\u7740\u5B83\u4E00\u8D77\u52A8\u3002"],["ayuan:smile","\u90A3\u4F60\u4EEC\u6728\u5320\uFF0C\u5012\u662F\u633A\u61C2\u5F97\u8BA9\u7740\u70B9\u513F\u7684\u3002"]],next:"e1fire"},e1i_gift:{bg:"e1_shed",music:"warm",lines:["\u957F\u5E9A\u62FF\u51FA\u4E00\u5757\u8FB9\u89D2\u6599\uFF0C\u6BD4\u5212\u7740\u8981\u7ED9\u5979\u505A\u4E00\u53EA\u88C5\u989C\u6599\u7684\u753B\u7BB1\u3002",["changgeng:smile","\u7BB1\u89D2\u7528\u7CBD\u89D2\u69AB\uFF0C\u4E09\u9762\u90FD\u662F\u659C\u89D2\uFF0C\u770B\u4E0D\u89C1\u4E00\u4E2A\u69AB\u5934\u3002\u4F60\u770B\u2014\u2014\u5B83\u662F\u8FD9\u4E48\u62C6\u5F00\u7684\u3002"]],puzzle:"zongjiao",solved:"e1i_gift2"},e1i_gift2:{bg:"e1_shed",card:"zongjiao",music:"warm",aff:{ayuan:2},lines:[["ayuan:smile","\u4E09\u6839\u6728\u5934\uFF0C\u4ECE\u54EA\u8FB9\u770B\u90FD\u4E0D\u9732\u7F1D\u2026\u2026\u4F60\u505A\u7684\u4E1C\u897F\uFF0C\u8DDF\u4F60\u8FD9\u4E2A\u4EBA\u4E00\u6837\u3002"],["changgeng","\u2026\u2026\u600E\u4E48\u6837\uFF1F"],["ayuan:smile","\u95F7\u3002\u53EF\u662F\u4E25\u5B9E\u3002"]],next:"e1fire"},e1fire:{cut:{era:"\u6C38\u4E50\u5341\u4E5D\u5E74 \xB7 \u56DB\u6708\u521D\u516B",place:"\u7D2B\u7981\u57CE",title:"\u4E09\u6BBF\u706B",bg:"e1_fire",music:"drama"},next:"e1fire_cg"},e1fire_cg:{cg:"flash_a2",still:"e1_fire",caption:"\u96F7\u706B\u843D\u5728\u5949\u5929\u6BBF\u4E0A\u3002\u706B\u820C\u987A\u7740\u67F1\u5B50\u5F80\u4E0A\u722C\uFF0C\u5FEB\u5F97\u4E0D\u50CF\u8BDD\u3002",next:"e1fire2"},e1fire2:{bg:"e1_fire",fx:"embers",sfx:"fire",music:"tense",lines:["\u5BAB\u6BBF\u843D\u6210\u624D\u4E00\u767E\u591A\u5929\u3002\u4E00\u4E2A\u96F7\u96E8\u591C\uFF0C\u5949\u5929\u3001\u534E\u76D6\u3001\u8C28\u8EAB\u4E09\u5927\u6BBF\u8D77\u4E86\u5927\u706B\u3002",["han:angry","\u6211\u7684\u6597\u6831\u6837\u5B50\u8FD8\u5728\u504F\u6BBF\u91CC\uFF01\u90A3\u662F\u6574\u5EA7\u6BBF\u7684\u5C3A\u5BF8\u2014\u2014"]],choices:[{label:"\u51B2\u8FDB\u53BB\u62A2\u51FA\u6597\u6831\u6837\u5B50",go:"e1fire_run",set:{fire:"run"}},{label:"\u62C9\u4F4F\u5E08\u7236\uFF0C\u4E00\u8D77\u4F20\u6C34",go:"e1fire_stay",set:{fire:"stay"}}],timer:8,onTimeout:{go:"e1fire_late",set:{fire:"late",hanHurt:!0},note:"\u4F60\u6123\u4F4F\u4E86\u4E00\u77AC\u3002\u5C31\u8FD9\u4E00\u77AC\uFF0C\u5E08\u7236\u5DF2\u7ECF\u51B2\u8FDB\u4E86\u706B\u91CC\u3002"}},e1fire_run:{bg:"e1_fire",fx:"embers",sfx:"fire",set:{saw:!0},lines:["\u957F\u5E9A\u51B2\u8FDB\u504F\u6BBF\uFF0C\u62B1\u51FA\u90A3\u7EC4\u6597\u6831\u6837\u5B50\u65F6\uFF0C\u8896\u5B50\u5DF2\u7ECF\u70E7\u7740\u4E86\u3002","\u56DE\u5934\u90A3\u4E00\u773C\uFF0C\u4ED6\u770B\u89C1\u4E00\u6839\u70E7\u88C2\u7684\u5927\u67F1\u2014\u2014\u5916\u9762\u662F\u6960\u6728\uFF0C\u88C2\u53E3\u91CC\u9732\u51FA\u6765\u7684\uFF0C\u5374\u662F\u767D\u751F\u751F\u7684\u6749\u6728\u82AF\u3002",["han:angry","\u7CCA\u6D82\uFF01\u6728\u5934\u70E7\u4E86\u8FD8\u80FD\u518D\u957F\uFF0C\u4EBA\u6CA1\u4E86\u5C31\u6CA1\u4E86\uFF01"],"\u5E08\u7236\u9A82\u7740\uFF0C\u624B\u5374\u4E00\u76F4\u5728\u6296\u3002"],next:"e1k"},e1fire_stay:{bg:"e1_fire",fx:"embers",sfx:"fire",lines:[["changgeng:angry","\u5E08\u7236\uFF01\u6837\u5B50\u70E7\u4E86\u8FD8\u80FD\u518D\u505A\uFF0C\u60A8\u8FDB\u53BB\u5C31\u51FA\u4E0D\u6765\u4E86\uFF01"],"\u4E24\u4EBA\u5728\u4EBA\u7FA4\u91CC\u4F20\u4E86\u4E00\u591C\u7684\u6C34\u3002\u5929\u4EAE\u65F6\uFF0C\u4E09\u5EA7\u5927\u6BBF\u53EA\u5269\u7126\u9ED1\u7684\u67F1\u7840\u3002",["han:sad","\u5C3A\u5BF8\u90FD\u5728\u6211\u8111\u5B50\u91CC\u3002\u53EA\u8981\u4EBA\u5728\uFF0C\u5C31\u80FD\u518D\u76D6\u8D77\u6765\u3002"]],next:"e1k"},e1fire_late:{bg:"e1_fire",fx:"embers",sfx:"fire",set:{saw:!0},lines:["\u957F\u5E9A\u8FFD\u8FDB\u53BB\uFF0C\u628A\u5E08\u7236\u4ECE\u584C\u4E0B\u6765\u7684\u6881\u5E95\u4E0B\u62D6\u51FA\u6765\u3002\u6597\u6831\u6837\u5B50\u62A2\u51FA\u6765\u4E86\uFF0C\u5E08\u7236\u7684\u534A\u8FB9\u624B\u81C2\u70E7\u5F97\u4E0D\u6210\u6837\u5B50\u3002",["han:sad","\u2026\u2026\u770B\u89C1\u4E86\u5417\uFF0C\u90A3\u6839\u67F1\u5B50\u3002\u88C2\u5F00\u7684\u5730\u65B9\uFF0C\u4E0D\u662F\u6960\u6728\u3002"]],next:"e1k"},e1k:{bg:"e1_ruins",music:"tense",lines:["\u706B\u706D\u4E86\u4E09\u5929\uFF0C\u5BAB\u91CC\u4F20\u51FA\u8BDD\u6765\uFF1A\u4E09\u6BBF\u5931\u706B\uFF0C\u662F\u6728\u4F5C\u5320\u4EBA\u591C\u91CC\u7528\u706B\u4E0D\u614E\u3002",["cao:smile","\u97E9\u5E08\u5085\u662F\u6728\u4F5C\u5934\uFF0C\u8FD9\u7F6A\uFF0C\u4ED6\u5F97\u62C5\u7740\u3002"],"\u5F53\u5929\uFF0C\u5E08\u7236\u88AB\u9501\u8FDB\u4E86\u5211\u90E8\u5927\u7262\u3002\u79CB\u540E\u95EE\u65A9\u3002",["cao","\u5C0F\u6728\u5320\uFF0C\u6211\u542C\u8BF4\u4F60\u706B\u91CC\u62A2\u51FA\u4E86\u70B9\u4E1C\u897F\uFF1F\u4EA4\u51FA\u6765\uFF0C\u6216\u8BB8\u8FD8\u80FD\u7ED9\u4F60\u5E08\u7236\u6C42\u4E2A\u60C5\u3002","fire=run|fire=late"],["cao","\u5C0F\u6728\u5320\uFF0C\u4F60\u5E08\u7236\u7684\u6597\u6831\u6837\u5B50\uFF0C\u53EF\u8FD8\u5728\u4F60\u624B\u4E0A\uFF1F\u4EA4\u51FA\u6765\uFF0C\u6216\u8BB8\u8FD8\u80FD\u7ED9\u4F60\u5E08\u7236\u6C42\u4E2A\u60C5\u3002","fire=stay"],"\u957F\u5E9A\u60F3\u8D77\u90A3\u6839\u70E7\u88C2\u7684\u67F1\u5B50\u3002\u53EA\u8981\u8BC1\u660E\u67F1\u82AF\u662F\u6749\u6728\uFF0C\u5C31\u80FD\u8BC1\u660E\u6709\u4EBA\u5077\u6362\u4E86\u6960\u6728\uFF0C\u706B\u624D\u70E7\u5F97\u90A3\u6837\u5FEB\u3002"],next:"e1l"},e1l:{bg:"e1_ruins",lines:["\u591C\u91CC\uFF0C\u957F\u5E9A\u6478\u56DE\u5E9F\u589F\uFF0C\u627E\u5230\u90A3\u6839\u6CA1\u5012\u7684\u7126\u67F1\u3002\u5265\u5F00\u5916\u9762\u7684\u5305\u9576\u677F\uFF0C\u5C31\u80FD\u770B\u89C1\u91CC\u9762\u3002"],puzzle:"guanjiao",solved:"e1m"},e1m:{bg:"e1_ruins",card:"guanjiao",lines:["\u94C1\u7B8D\u4E00\u892A\uFF0C\u5305\u9576\u677F\u5E94\u58F0\u800C\u5F00\u3002\u67F1\u82AF\u662F\u6749\u6728\uFF0C\u767D\u751F\u751F\u7684\uFF0C\u8FDE\u4E00\u5708\u6960\u6728\u7684\u7EB9\u90FD\u6CA1\u6709\u3002","\u53EF\u5149\u6709\u4E00\u6839\u67F1\u5B50\u4E0D\u591F\u3002\u8981\u7FFB\u6848\uFF0C\u8FD8\u5F97\u6709\u8D26\u3002"],route:[{if:"aff.luqi>=2",go:"e1n_a"},{go:"e1n_b"}]},e1n_a:{bg:"e1_ruins",lines:["\u8EAB\u540E\u6709\u811A\u6B65\u58F0\u3002\u662F\u9646\u4E03\u3002",["luqi:sad","\u6211\u90FD\u77E5\u9053\u4E86\u3002\u5E08\u7236\u662F\u66FF\u4ED6\u4EEC\u9876\u7F6A\u3002"],["luqi:sad","\u8D26\u5728\u66F9\u516C\u516C\u503C\u623F\u91CC\uFF0C\u85CF\u5728\u96C0\u66FF\u7684\u6697\u683C\u91CC\u2014\u2014\u90A3\u96C0\u66FF\u662F\u6211\u4EB2\u624B\u638F\u7A7A\u7684\u3002\u6211\u5E26\u4F60\u53BB\u3002"],["changgeng:sad","\u5E08\u5144\u2026\u2026"],["luqi","\u6211\u5A18\u8BF4\uFF0C\u836F\u94B1\u53EF\u4EE5\u6162\u6162\u8FD8\uFF0C\u826F\u5FC3\u6B20\u4E0B\u4E86\uFF0C\u4E00\u8F88\u5B50\u90FD\u8FD8\u4E0D\u4E0A\u3002"]],next:"e1o_a"},e1o_a:{bg:"e1_office",music:"tense",lines:["\u503C\u623F\u91CC\u4E00\u76CF\u706F\u4E5F\u6CA1\u70B9\u3002\u6708\u5149\u7167\u7740\u6881\u4E0B\u90A3\u53EA\u96D5\u82B1\u96C0\u66FF\u3002"],puzzle:"queti",solved:"e1p_a"},e1p_a:{bg:"e1_office",card:"queti",set:{hanSaved:!0,luRedeem:!0},lines:["\u6697\u8D26\u4E00\u9875\u9875\u8BB0\u7740\uFF1A\u6960\u6728\u591A\u5C11\u6839\uFF0C\u6362\u6210\u6749\u6728\u591A\u5C11\u6839\uFF0C\u94F6\u5B50\u8FDB\u4E86\u8C01\u7684\u53E3\u888B\u3002",["ayuan","\u6211\u7239\u7684\u5F69\u753B\u8D26\u4E5F\u5728\u8FD9\u91CC\u3002\u4E24\u672C\u4E00\u5BF9\uFF0C\u8C01\u90FD\u8D56\u4E0D\u6389\u3002","aff.ayuan>=2"],"\u5929\u4EAE\u524D\uFF0C\u8D26\u518C\u8FDE\u540C\u90A3\u6839\u7126\u67F1\u7684\u5305\u9576\u677F\uFF0C\u4E00\u8D77\u9001\u8FDB\u4E86\u90FD\u5BDF\u9662\u3002","\u534A\u4E2A\u6708\u540E\uFF0C\u66F9\u5185\u5B98\u4E0B\u4E86\u5927\u72F1\u3002\u5E08\u7236\u88AB\u653E\u4E86\u51FA\u6765\uFF0C\u7626\u5F97\u8131\u4E86\u5F62\u3002\u9646\u4E03\u81EA\u5DF1\u53BB\u8859\u95E8\u6295\u4E86\u6848\u3002"],next:"e1q"},e1n_b:{bg:"e1_lane",music:"tense",lines:["\u56DE\u5DE5\u68DA\u7684\u8DEF\u4E0A\uFF0C\u5939\u9053\u91CC\u4EAE\u8D77\u4E00\u4E32\u706F\u7B3C\u3002",["luqi:sad","\u2026\u2026\u5C31\u662F\u4ED6\uFF0C\u66F9\u516C\u516C\u3002\u6597\u6831\u6837\u5B50\u5728\u4ED6\u8EAB\u4E0A\u3002"],"\u9646\u4E03\u7AD9\u5728\u706F\u7B3C\u540E\u9762\uFF0C\u4E0D\u6562\u770B\u4ED6\u3002",["cao:smile","\u5C0F\u6728\u5320\uFF0C\u8DD1\u4EC0\u4E48\uFF1F"],"\u8EAB\u540E\u662F\u6B7B\u80E1\u540C\uFF0C\u53EA\u6709\u4E00\u6247\u4E0A\u4E86\u9501\u7684\u89D2\u95E8\u3002\u95E8\u4E0A\u4E24\u679A\u95E8\u7C2A\u2014\u2014\u957F\u5E9A\u5FFD\u7136\u60F3\u8D77\uFF0C\u8FD9\u79CD\u95E8\uFF0C\u662F\u80FD\u6574\u6247\u5378\u4E0B\u6765\u7684\u3002"],puzzle:"menzan",timer:75,solved:"e1o_b",failed:"e1o_caught"},e1o_b:{bg:"e1_lane",card:"menzan",lines:["\u95E8\u6247\u4E00\u5378\uFF0C\u957F\u5E9A\u4ECE\u95E8\u7F1D\u91CC\u94BB\u4E86\u51FA\u53BB\uFF0C\u80CC\u540E\u662F\u66F9\u5185\u5B98\u5C16\u7EC6\u7684\u53EB\u9A82\u58F0\u3002"],route:[{if:"aff.ayuan>=2",go:"e1o_b_ayuan"},{go:"e1o_b_alone"}]},e1o_b_ayuan:{bg:"e1_paint",music:"warm",set:{hanSaved:!0},lines:[["ayuan:sad","\u5FEB\u8FDB\u6765\uFF01"],"\u5F69\u753B\u4F5C\u574A\u7684\u95E8\u5F00\u4E86\u4E00\u6761\u7F1D\u3002\u5B81\u6C85\u628A\u4ED6\u85CF\u5728\u4E00\u5806\u753B\u7A3F\u540E\u9762\uFF0C\u53C8\u628A\u5979\u7239\u7684\u5F69\u753B\u8D26\u585E\u8FDB\u4ED6\u6000\u91CC\u3002",["ayuan","\u6960\u6728\u591A\u5C11\u6839\u3001\u591A\u7C97\uFF0C\u6211\u7239\u5168\u8BB0\u7740\u3002\u62FF\u53BB\uFF0C\u6551\u4F60\u5E08\u7236\u3002"],"\u8D26\u9001\u8FDB\u4E86\u90FD\u5BDF\u9662\u3002\u66F9\u5185\u5B98\u5012\u4E86\uFF0C\u5E08\u7236\u88AB\u653E\u4E86\u51FA\u6765\u3002\u9646\u4E03\u88AB\u4E00\u5E76\u95EE\u4E86\u7F6A\uFF0C\u53D1\u914D\u8FBD\u4E1C\u3002"],next:"e1q"},e1o_b_alone:{bg:"e1_jail",sfx:"rain",set:{hanSaved:!1},lines:["\u957F\u5E9A\u5728\u57CE\u5916\u7684\u7834\u5E99\u91CC\u8EB2\u4E86\u4E00\u4E2A\u51AC\u5929\u3002\u7B49\u4ED6\u518D\u8FDB\u57CE\uFF0C\u5211\u90E8\u5927\u7262\u7684\u95E8\u53E3\u8D34\u51FA\u4E86\u544A\u793A\u3002","\u97E9\u5E08\u5085\u6CA1\u7B49\u5230\u79CB\u540E\u3002\u4ED6\u6B7B\u5728\u7262\u91CC\uFF0C\u624B\u91CC\u6525\u7740\u4E00\u622A\u6728\u70AD\uFF0C\u5899\u4E0A\u753B\u6EE1\u4E86\u6597\u6831\u7684\u5C3A\u5BF8\u3002"],next:"e1q"},e1o_caught:{bg:"e1_jail",sfx:"rain",set:{hanSaved:!1,caught:!0},lines:["\u95E8\u7C2A\u62D4\u4E86\u4E00\u534A\uFF0C\u706F\u7B3C\u5C31\u7167\u5230\u4E86\u4ED6\u8138\u4E0A\u3002","\u957F\u5E9A\u548C\u5E08\u7236\u5173\u8FDB\u4E86\u540C\u4E00\u95F4\u7262\u623F\u3002",["han:sad","\u50BB\u5C0F\u5B50\u3002\u6837\u5B50\u4E22\u4E86\u4E0D\u8981\u7D27\u2026\u2026\u4F60\u542C\u7740\uFF0C\u6211\u628A\u5949\u5929\u6BBF\u7684\u5C3A\u5BF8\uFF0C\u4E00\u5BF8\u4E00\u5BF8\u5FF5\u7ED9\u4F60\u3002"],"\u90A3\u4E2A\u51AC\u5929\uFF0C\u5E08\u7236\u5FF5\uFF0C\u957F\u5E9A\u8BB0\u3002\u5F00\u6625\u524D\uFF0C\u5E08\u7236\u8D70\u4E86\u3002","\u540E\u6765\u66F9\u5185\u5B98\u56E0\u522B\u7684\u6848\u5B50\u5012\u53F0\uFF0C\u957F\u5E9A\u624D\u88AB\u653E\u51FA\u6765\u3002\u4ED6\u51FA\u72F1\u90A3\u5929\u4E00\u53E5\u8BDD\u4E5F\u6CA1\u8BF4\uFF0C\u56DE\u5230\u5DE5\u68DA\uFF0C\u94FA\u5F00\u4E86\u7EB8\u3002"],next:"e1q"},e1q:{bg:"e1_shed",music:"theme",lines:["\u957F\u5E9A\u51ED\u7740\u8BB0\u5FC6\u3001\u51ED\u7740\u90A3\u7EC4\u6597\u6831\u6837\u5B50\uFF0C\u5F00\u59CB\u753B\u6574\u5EA7\u5949\u5929\u6BBF\u7684\u300C\u6BBF\u6837\u300D\uFF1A\u6BCF\u4E00\u6839\u67F1\u5B50\u591A\u7C97\u3001\u6BCF\u4E00\u7EC4\u6597\u6831\u591A\u5927\uFF0C\u7528\u4EC0\u4E48\u6728\u5934\u3001\u4EC0\u4E48\u69AB\u3002",["han:smile","\u6709\u4E86\u8FD9\u5F20\u6837\u5B50\uFF0C\u5C06\u6765\u4E0D\u7BA1\u8C01\u6765\u4FEE\u6BBF\uFF0C\u5077\u6881\u6362\u67F1\u90FD\u7792\u4E0D\u8FC7\u53BB\u3002","hanSaved"],"\u6700\u96BE\u7684\u662F\u6597\u6831\u3002\u4ED6\u8981\u628A\u5B83\u4E00\u4EF6\u4EF6\u62C6\u5F00\uFF0C\u91CF\u51C6\u4E86\uFF0C\u518D\u4E00\u4EF6\u4EF6\u88C5\u56DE\u53BB\u3002"],puzzle:"dougong",solved:"e1r"},e1r:{bg:"e1_dougong",card:"dougong",lines:["\u6BBF\u6837\u753B\u6210\u90A3\u5929\uFF0C\u957F\u5E9A\u5728\u6700\u540E\u4E00\u9875\u5199\u4E0B\uFF1A\u300C\u6BBF\u6837\u5728\uFF0C\u6BBF\u5C31\u5728\u3002\u300D","\u6B63\u7EDF\u4E94\u5E74\uFF0C\u4E09\u5927\u6BBF\u91CD\u65B0\u52A8\u5DE5\uFF0C\u6B21\u5E74\u843D\u6210\u3002\u91CD\u5EFA\u4F9D\u7684\uFF0C\u6B63\u662F\u8FD9\u5F20\u6837\u5B50\u3002",[null,"\u540E\u6765\u4ED6\u53BB\u4E86\u4E00\u8D9F\u626C\u5DDE\u3002\u5B81\u6C85\u6CA1\u6709\u5AC1\u4EBA\u3002\u90A3\u53EA\u753B\u7BB1\uFF0C\u5979\u4E00\u76F4\u5E26\u5728\u8EAB\u8FB9\u3002","gift&aff.ayuan>=3"],[null,"\u5B81\u6C85\u540E\u6765\u56DE\u4E86\u626C\u5DDE\uFF0C\u518D\u6CA1\u6709\u6D88\u606F\u3002\u957F\u5E9A\u4E00\u8F88\u5B50\u6CA1\u6709\u5A36\u59BB\u3002","!gift|aff.ayuan<3"],[null,"\u9646\u4E03\u4ECE\u8859\u95E8\u51FA\u6765\u540E\uFF0C\u8DDF\u7740\u957F\u5E9A\u53C8\u5E72\u4E86\u4E8C\u5341\u5E74\u6728\u5320\u3002","luRedeem"],"\u957F\u5E9A\u7528\u91CD\u5EFA\u5269\u4E0B\u7684\u8FB9\u6599\u505A\u4E86\u4E00\u53EA\u6728\u5323\uFF0C\u5323\u5E95\u662F\u4E00\u628A\u9C81\u73ED\u9501\u3002\u6BBF\u6837\uFF0C\u5C31\u9501\u5728\u91CC\u9762\u3002"],next:"e1end"},e1end:{cg:"flash_a3",still:"e1_ruins",caption:"\u5E9F\u589F\u91CC\uFF0C\u4E00\u76CF\u706F\u4E0B\uFF0C\u4ED6\u4E00\u7B14\u4E00\u7B14\u753B\u7740\u6BBF\u6837\u3002",next:"e2cut"},e2cut:{cut:{era:"\u6C11\u56FD\u4E8C\u5341\u516D\u5E74\uFF081937\uFF09",place:"\u5317\u5E73",title:"\u7B2C\u4E8C\u5377 \xB7 \u5B88\u62D9",bg:"e2_hutong",music:"theme"},next:"e2a"},e2a:{bg:"e2_hutong",lines:["\u4E94\u767E\u591A\u5E74\u540E\uFF0C\u5317\u5E73\u3002\u6728\u5323\u4F20\u5230\u4E86\u987E\u5B88\u62D9\u624B\u91CC\u3002\u4ED6\u5728\u4E1C\u57CE\u5F00\u4E00\u95F4\u6728\u4F5C\u94FA\uFF0C\u4E09\u5341\u51FA\u5934\uFF0C\u8BDD\u5C11\uFF0C\u624B\u5DE7\u3002",["xiuhe:smile","\u5B88\u62D9\uFF0C\u5976\u5976\u90A3\u628A\u5708\u6905\u7684\u6276\u624B\u53C8\u677E\u4E86\u3002\u5B69\u5B50\u751F\u4E0B\u6765\uFF0C\u6211\u8FD8\u8981\u62B1\u7740\u4ED6\u5750\u90A3\u628A\u6905\u5B50\u5462\u3002"],"\u95E8\u5916\u6709\u4EBA\u7B11\u7740\u8FDB\u6765\uFF0C\u957F\u888D\u9A6C\u8902\uFF0C\u4E00\u8EAB\u70DF\u5473\u3002",["sun:smile","\u8868\u59B9\uFF0C\u8868\u59B9\u592B\uFF01\u6211\u8DEF\u8FC7\uFF0C\u6765\u77A7\u77A7\u4F60\u4EEC\u3002\u54DF\uFF0C\u8FD9\u5C31\u662F\u4F60\u4EEC\u987E\u5BB6\u90A3\u53EA\u660E\u671D\u7684\u6728\u5323\uFF1F"]],choices:[{label:"\u8BA9\u4ED6\u770B\u770B",go:"e2a2",aff:{sun:1},set:{sunSaw:!0}},{label:"\u628A\u6728\u5323\u6536\u8D77\u6765",go:"e2a2",aff:{xiuhe:1}}]},e2a2:{bg:"e2_hutong",lines:[["sun:smile","\u597D\u4E1C\u897F\u554A\u3002\u8981\u662F\u54EA\u5929\u60F3\u51FA\u624B\uFF0C\u627E\u6211\uFF0C\u4EF7\u94B1\u597D\u8BF4\u3002","sunSaw"],["sun","\u6536\u8FD9\u4E48\u7D27\u505A\u4EC0\u4E48\uFF0C\u6211\u53C8\u4E0D\u5077\u3002","!sunSaw"],["xiuhe:sad","\u8868\u54E5\uFF0C\u4F60\u90A3\u94FA\u5B50\u2026\u2026\u6700\u8FD1\u6765\u7684\u90FD\u662F\u4E9B\u4EC0\u4E48\u4EBA\uFF1F"],["sun","\u505A\u4E70\u5356\u561B\uFF0C\u8C01\u7ED9\u94B1\u5C31\u662F\u8C01\u3002"],"\u5B59\u4E16\u660C\u8D70\u540E\uFF0C\u5B88\u62D9\u53BB\u4FEE\u90A3\u628A\u5708\u6905\u3002"],puzzle:"pin",solved:"e2b"},e2b:{bg:"e2_hutong",card:"pin",clue:"\u6709\u65F6\u5019\uFF0C\u5F97\u5148\u8BA9\u4E00\u6B65",lines:["\u6954\u9489\u9000\u51FA\u6765\uFF0C\u4E24\u6BB5\u6276\u624B\u5C31\u5206\u5F00\u4E86\u3002\u5408\u638C\u7684\u642D\u53E3\u91CC\uFF0C\u523B\u7740\u4E00\u884C\u4E0D\u77E5\u54EA\u4E00\u8F88\u7956\u5B97\u7559\u4E0B\u7684\u5C0F\u5B57\u3002",["xiuhe:smile","\u4F60\u4EEC\u987E\u5BB6\u7684\u4EBA\uFF0C\u600E\u4E48\u90FD\u559C\u6B22\u5F80\u6728\u5934\u91CC\u85CF\u8BDD\uFF1F"],["shouzhuo","\u6728\u5934\u6BD4\u7EB8\u6D3B\u5F97\u4E45\u3002"]],next:"e2c"},e2c:{bg:"e2_curio",lines:["\u51E0\u5929\u540E\uFF0C\u5B59\u4E16\u660C\u6258\u4EBA\u634E\u8BDD\uFF1A\u94FA\u5B50\u91CC\u4E00\u5F20\u660E\u4EE3\u5E73\u5934\u6848\u6563\u4E86\u67B6\uFF0C\u8BF7\u5B88\u62D9\u53BB\u770B\u770B\u3002",["sun:smile","\u8FD9\u6848\u5B50\u662F\u8981\u5356\u7ED9\u4E00\u4F4D\u65E5\u672C\u5148\u751F\u7684\u3002\u5C0F\u91CE\u5148\u751F\uFF0C\u61C2\u884C\uFF0C\u51FA\u624B\u5927\u65B9\u3002"],["sun","\u4ED6\u8FD8\u6253\u542C\u5317\u5E73\u54EA\u5BB6\u85CF\u7740\u660E\u671D\u5BAB\u91CC\u7684\u4E1C\u897F\u2014\u2014\u5C24\u5176\u662F\u56FE\u6837\u3002"]],puzzle:"jiatou",solved:"e2c2"},e2c2:{bg:"e2_curio",card:"jiatou",lines:[["sun:smile","\u597D\u624B\u827A\uFF01\u8868\u59B9\u592B\uFF0C\u4F60\u5BB6\u90A3\u53EA\u6728\u5323\u91CC\uFF0C\u542C\u8BF4\u9501\u7740\u4E00\u5F20\u6C38\u4E50\u5E74\u95F4\u7684\u6BBF\u6837\uFF1F\u5C0F\u91CE\u5148\u751F\u8BF4\uFF0C\u5F00\u4E2A\u4EF7\u3002"]],choices:[{label:"\u300C\u7956\u5B97\u7684\u4E1C\u897F\uFF0C\u4E0D\u5356\u3002\u300D",go:"e2d",aff:{sun:-1,xiuhe:1}},{label:"\u300C\u8868\u54E5\uFF0C\u8FD9\u94B1\u4F60\u4E5F\u522B\u6323\u3002\u300D",go:"e2d",aff:{sun:1}}]},e2d:{bg:"e2_palace",aff:{zhou:1},lines:["\u90A3\u5E74\u6625\u5929\uFF0C\u4E2D\u56FD\u8425\u9020\u5B66\u793E\u5728\u6545\u5BAB\u6D4B\u7ED8\uFF0C\u8BF7\u5B88\u62D9\u5E2E\u5FD9\u770B\u5927\u6728\u4F5C\u3002",["zhou","\u6709\u4EBA\u8BF4\u4E2D\u56FD\u5DF2\u7ECF\u6CA1\u6709\u5510\u4EE3\u7684\u6728\u6784\u4E86\uFF0C\u8981\u770B\u5510\u671D\u7684\u623F\u5B50\uFF0C\u5F97\u53BB\u65E5\u672C\u5948\u826F\u3002\u6211\u4EEC\u753B\u56FE\uFF0C\u662F\u8981\u8BC1\u660E\u6211\u4EEC\u81EA\u5DF1\u7684\u4E1C\u897F\uFF0C\u6211\u4EEC\u81EA\u5DF1\u770B\u5F97\u61C2\u3001\u5B88\u5F97\u4F4F\u3002"],"\u5B88\u62D9\u72B9\u8C6B\u4E86\u5F88\u4E45\uFF0C\u628A\u6BBF\u6837\u5E26\u4E86\u6765\u3002",["zhou:smile","\u8FD9\u662F\u2026\u2026\u6C38\u4E50\u5E74\u95F4\u5949\u5929\u6BBF\u7684\u539F\u6837\uFF1F\u67F1\u5F84\u3001\u6597\u6831\u3001\u7528\u6750\uFF0C\u5168\u6709\uFF01"],["zhou","\u987E\u5E08\u5085\uFF0C\u5C06\u6765\u6545\u5BAB\u5927\u4FEE\uFF0C\u5C31\u5F97\u9760\u5B83\u3002\u5B83\u6BD4\u6211\u4EEC\u6240\u6709\u7684\u6D4B\u7A3F\u90FD\u91CD\u8981\u3002"],"\u914D\u6BBF\u4E00\u6839\u989D\u678B\u7CDF\u673D\u4E86\uFF0C\u6B63\u8981\u843D\u67B6\u3002\u5B88\u62D9\u8BF4\uFF0C\u62C6\u5F00\u770B\uFF0C\u548C\u6BBF\u6837\u5BF9\u4E00\u5BF9\u3002"],puzzle:"beam",solved:"e2e"},e2e:{bg:"e2_palace",card:"beam",lines:[["zhou:smile","\u71D5\u5C3E\u7684\u5C3A\u5BF8\uFF0C\u548C\u6BBF\u6837\u4E0A\u7684\u4E00\u5206\u4E0D\u5DEE\uFF01"],"\u4E03\u6708\u521D\uFF0C\u5362\u6C9F\u6865\u7684\u67AA\u58F0\u54CD\u4E86\u3002\u6D88\u606F\u4F20\u5230\u5C71\u897F\u65F6\uFF0C\u6881\u5148\u751F\u4ED6\u4EEC\u521A\u5728\u4E94\u53F0\u5C71\u7684\u4F5B\u5149\u5BFA\uFF0C\u627E\u5230\u4E86\u4E00\u5EA7\u5510\u4EE3\u5927\u4E2D\u5E74\u95F4\u7684\u6728\u6784\u5927\u6BBF\u3002","\u5317\u5E73\u6CA6\u9677\u3002"],next:"e2f"},e2f:{bg:"e2_occupied",music:"tense",lines:["\u5B66\u793E\u7684\u9662\u5B50\u88AB\u5360\u4E86\u3002\u4E00\u4E2A\u6234\u5706\u773C\u955C\u7684\u65E5\u672C\u4EBA\u7FFB\u7740\u5B66\u793E\u6765\u4E0D\u53CA\u5E26\u8D70\u7684\u6863\u6848\uFF0C\u5B59\u4E16\u660C\u5728\u4E00\u65C1\u5F2F\u7740\u8170\u3002",["ono:smile","\u6211\u662F\u5C0F\u91CE\u3002\u6211\u5BF9\u4E2D\u56FD\u7684\u53E4\u5EFA\u7B51\uFF0C\u975E\u5E38\u5C0A\u656C\u3002"],["ono","\u542C\u8BF4\u6C38\u4E50\u5E74\u95F4\u5949\u5929\u6BBF\u7684\u539F\u6837\uFF0C\u5728\u4E00\u4F4D\u987E\u59D3\u6728\u5320\u5BB6\u91CC\u3002\u8FD9\u6837\u7684\u4E1C\u897F\uFF0C\u5E94\u8BE5\u7531\u61C2\u5F97\u5B83\u7684\u4EBA\u4FDD\u7BA1\u3002"],["sun:sad","\u2026\u2026\u5C0F\u91CE\u5148\u751F\uFF0C\u6211\u53EA\u662F\u542C\u8BF4\uFF0C\u542C\u8BF4\u3002"],["zhou:angry","\u987E\u5E08\u5085\uFF0C\u4ED6\u4EEC\u4ECA\u665A\u5C31\u4F1A\u53BB\u4F60\u5BB6\uFF01\u5B66\u793E\u7684\u6D4B\u7A3F\u526F\u672C\u548C\u6BBF\u6837\uFF0C\u5FC5\u987B\u9A6C\u4E0A\u85CF\u597D\uFF01"]],next:"e2g"},e2g:{bg:"e2_search",sfx:"knock",music:"tense",lines:["\u534A\u591C\uFF0C\u80E1\u540C\u91CC\u54CD\u8D77\u76AE\u9774\u58F0\uFF0C\u624B\u7535\u5149\u4E00\u9053\u9053\u626B\u8FC7\u9662\u5899\u3002",["xiuhe:sad","\u5B88\u62D9\uFF0C\u67DC\u95E8\uFF01\u628A\u56FE\u5939\u8FDB\u67DC\u95E8\u91CC\uFF01"]],puzzle:"panel",timer:80,solved:"e2g_ok",failed:"e2g_fail"},e2g_ok:{bg:"e2_attic",card:"panel",sfx:"knock",set:{hidden:!0},lines:["\u6700\u540E\u4E00\u5757\u82AF\u677F\u63A8\u56DE\u69FD\u91CC\u7684\u65F6\u5019\uFF0C\u9662\u95E8\u88AB\u649E\u5F00\u4E86\u3002","\u4ED6\u4EEC\u7FFB\u4E86\u4E00\u4E2A\u65F6\u8FB0\uFF0C\u7FFB\u904D\u4E86\u5E8A\u5E95\u3001\u7076\u819B\u3001\u623F\u6881\uFF0C\u770B\u4E86\u90A3\u53E3\u67DC\u5B50\u597D\u51E0\u773C\uFF0C\u4EC0\u4E48\u4E5F\u6CA1\u627E\u5230\u3002",["ono","\u987E\u5148\u751F\uFF0C\u6211\u4EEC\u8FD8\u4F1A\u518D\u89C1\u7684\u3002"]],next:"e2g_sun"},e2g_fail:{bg:"e2_attic",sfx:"knock",set:{hidden:!1},lines:["\u82AF\u677F\u8FD8\u6CA1\u63A8\u56DE\u53BB\uFF0C\u9662\u95E8\u5C31\u88AB\u649E\u5F00\u4E86\u3002\u624B\u7535\u5149\u7167\u5728\u67DC\u95E8\u7684\u5939\u5C42\u4E0A\u3002",["ono:smile","\u627E\u5230\u4E86\u3002\u4E2D\u56FD\u7684\u6728\u5320\uFF0C\u679C\u7136\u806A\u660E\u3002"],"\u5B66\u793E\u7684\u6D4B\u7A3F\u526F\u672C\u88AB\u4E00\u5F20\u5F20\u6536\u8D70\u3002",["xiuhe:sad","\u2026\u2026\u6BBF\u6837\u5728\u6211\u8FD9\u513F\u3002\u6211\u628A\u5B83\u7F1D\u5728\u4E86\u8902\u5B50\u7684\u5939\u91CC\u3002","aff.xiuhe>=2"],[null,"\u6BBF\u6837\u4E5F\u88AB\u6536\u8D70\u4E86\u3002\u5B88\u62D9\u8DEA\u5728\u5730\u4E0A\uFF0C\u5F88\u4E45\u6CA1\u6709\u8D77\u6765\u3002","aff.xiuhe<2"]],route:[{if:"aff.xiuhe<2",go:"e2g_sun",set:{yangLost:!0}},{go:"e2g_sun"}]},e2g_sun:{bg:"e2_hutong",lines:["\u5929\u4EAE\u524D\uFF0C\u5B59\u4E16\u660C\u4E00\u4E2A\u4EBA\u6765\u4E86\uFF0C\u7AD9\u5728\u95E8\u53E3\uFF0C\u8138\u8272\u53D1\u767D\u3002"],route:[{if:"aff.sun>=1",go:"e2g_sun_good"},{go:"e2g_sun_bad"}]},e2g_sun_good:{bg:"e2_hutong",set:{sunRedeem:!0},lines:[["sun:sad","\u8868\u59B9\u2026\u2026\u662F\u6211\u591A\u5634\u3002\u6211\u539F\u4EE5\u4E3A\uFF0C\u65E5\u672C\u4EBA\u5C31\u662F\u6765\u4E70\u4E1C\u897F\u7684\u3002"],["sun:sad","\u6211\u5728\u5C0F\u91CE\u90A3\u513F\u8FD8\u80FD\u8BF4\u4E0A\u8BDD\u3002\u6BBF\u6837\u843D\u5230\u4ED6\u624B\u91CC\uFF0C\u6211\u60F3\u6CD5\u5B50\u5F04\u56DE\u6765\u3002","yangLost"],["sun","\u5B66\u793E\u4ECA\u665A\u6709\u4E00\u8D9F\u8F66\u5F80\u5357\u8D70\u3002\u4F60\u4EEC\u8981\u8D70\uFF0C\u5C31\u4ECA\u665A\u3002"]],next:"e2h"},e2g_sun_bad:{bg:"e2_hutong",lines:[["sun:smile","\u8868\u59B9\u592B\uFF0C\u4F55\u5FC5\u5462\uFF1F\u4E1C\u897F\u4EA4\u4E86\uFF0C\u5927\u5BB6\u90FD\u597D\u8FC7\u3002\u5C0F\u91CE\u5148\u751F\u8BF4\u4E86\uFF0C\u4F60\u8981\u662F\u80AF\u7ED9\u4ED6\u505A\u4E8B\uFF0C\u94FA\u5B50\u7167\u5F00\u3002"],["xiuhe:angry","\u4F60\u8D70\uFF01"]],next:"e2h"},e2h:{bg:"e2_station",sfx:"train",music:"tense",lines:["\u524D\u95E8\u4E1C\u7AD9\u6324\u6EE1\u4E86\u4EBA\u3002\u5468\u5148\u751F\u5728\u8F66\u95E8\u53E3\u62FC\u547D\u62DB\u624B\u3002",["zhou:angry","\u987E\u5E08\u5085\uFF01\u4E0A\u8F66\uFF01\u8DDF\u5B66\u793E\u5357\u4E0B\uFF0C\u6BBF\u6837\u548C\u6D4B\u7A3F\u6211\u4EEC\u4E00\u8D77\u62A4\u7740\uFF01"],["xiuhe:sad","\u6211\u5C31\u5FEB\u751F\u4E86\uFF0C\u8D70\u4E0D\u4E86\u8FD9\u4E48\u8FDC\u2026\u2026\u4F60\u53BB\u5427\u3002","aff.xiuhe>=2"],["xiuhe:sad","\u4F60\u8981\u8D70\uFF0C\u5C31\u8D70\u5427\u3002","aff.xiuhe<2"],"\u6C7D\u7B1B\u54CD\u4E86\u3002"],choices:[{label:"\u8DF3\u4E0A\u706B\u8F66\uFF0C\u968F\u5B66\u793E\u5357\u4E0B",go:"e2i_s",set:{path:"south"},aff:{zhou:1}},{label:"\u7559\u5728\u5317\u5E73\uFF0C\u5B88\u7740\u79C0\u79BE",go:"e2i_l",set:{path:"stay"},aff:{xiuhe:2}}],timer:10,onTimeout:{go:"e2i_l",set:{path:"stay"},note:"\u706B\u8F66\u5F00\u8D70\u4E86\u3002\u4F60\u7AD9\u5728\u6708\u53F0\u4E0A\uFF0C\u79C0\u79BE\u6293\u7740\u4F60\u7684\u8896\u5B50\u3002"}},e2i_s:{bg:"e2_lizhuang",sfx:"rain",lines:["\u5B66\u793E\u4E00\u8DEF\u5357\u4E0B\uFF0C\u957F\u6C99\u3001\u6606\u660E\uFF0C\u6700\u540E\u843D\u811A\u5728\u56DB\u5DDD\u957F\u6C5F\u8FB9\u7684\u674E\u5E84\u3002","\u4ED6\u4EEC\u501F\u4F4F\u7684\u8001\u5B85\u662F\u7A7F\u6597\u5F0F\u7684\uFF0C\u5E74\u4E45\u5931\u4FEE\uFF0C\u4E00\u6839\u7A7F\u678B\u673D\u4E86\u3002\u5B88\u62D9\u8BF4\uFF0C\u4ED6\u6765\u6362\u3002",["zhou:smile","\u987E\u5E08\u5085\uFF0C\u5317\u5E73\u6765\u4FE1\u4E86\u3002\u79C0\u79BE\u751F\u4E86\u4E2A\u7537\u5B69\uFF0C\u53EB\u987E\u627F\u3002"]],puzzle:"chuandou",solved:"e2j_s"},e2j_s:{bg:"e2_lizhuang",card:"chuandou",lines:["\u6BBF\u6837\u548C\u6D4B\u7A3F\u5C31\u9501\u5728\u957F\u5E9A\u7684\u6728\u5323\u91CC\uFF0C\u8DDF\u7740\u4ED6\u8D70\u4E86\u51E0\u5343\u91CC\u3002",["zhou","\u987E\u5E08\u5085\uFF0C\u6211\u4EEC\u753B\u7684\u8FD9\u4E9B\u56FE\uFF0C\u5C06\u6765\u662F\u8981\u8FD8\u7ED9\u4E2D\u56FD\u4EBA\u81EA\u5DF1\u7684\u3002"],["zhou:sad","\u6BBF\u6837\u4E0D\u5728\u4E86\u3002\u53EF\u4F60\u8111\u5B50\u91CC\u7684\u624B\u827A\uFF0C\u6BD4\u7EB8\u66F4\u96BE\u62A2\u8D70\u3002","yangLost"]],route:[{if:"yangLost&sunRedeem",go:"e2k",set:{yangLost:!1}},{go:"e2k"}]},e2i_l:{bg:"e2_occupied",lines:["\u5B88\u62D9\u7559\u4E86\u4E0B\u6765\u3002\u6CA1\u8FC7\u591A\u4E45\uFF0C\u5C0F\u91CE\u6D3E\u4EBA\u6765\u8BF7\uFF1A\u8981\u4ED6\u7ED9\u300C\u8C03\u67E5\u73ED\u300D\u505A\u4E00\u5957\u529E\u516C\u7528\u7684\u660E\u5F0F\u5BB6\u5177\u3002",["ono:smile","\u987E\u5148\u751F\uFF0C\u8FD9\u662F\u5C0A\u91CD\u4F60\u7684\u624B\u827A\u3002"],"\u5B88\u62D9\u63A5\u4E86\u6D3B\u3002\u4E00\u5F20\u65B9\u684C\u7684\u9738\u738B\u67A8\u8981\u62C6\u5F00\u91CD\u505A\u2014\u2014\u90A3\u91CC\u9762\uFF0C\u6B63\u597D\u80FD\u85CF\u4E0B\u6BBF\u6837\u3002"],puzzle:"bawang",solved:"e2j_l"},e2j_l:{bg:"e2_occupied",card:"bawang",lines:["\u6BBF\u6837\u85CF\u8FDB\u4E86\u684C\u5E95\u7684\u7A7F\u5E26\u91CC\uFF0C\u5C31\u6446\u5728\u5C0F\u91CE\u7684\u773C\u76AE\u5E95\u4E0B\u3002\u6700\u5371\u9669\u7684\u5730\u65B9\uFF0C\u6700\u6CA1\u4EBA\u60F3\u5230\u3002","\u4ED6\u7ED9\u8C03\u67E5\u73ED\u505A\u7684\u6BCF\u4E00\u4EF6\u5BB6\u5177\uFF0C\u90FD\u5728\u4E0D\u8D77\u773C\u7684\u5730\u65B9\u7559\u4E86\u4E00\u9053\u6D3B\u69AB\u2014\u2014\u7B49\u54EA\u5929\u8981\u62C6\uFF0C\u4E00\u62C6\u5C31\u6563\u3002",["sun:sad","\u8868\u59B9\u592B\uFF0C\u5C0F\u91CE\u90A3\u513F\u7684\u6BBF\u6837\uFF0C\u6211\u6362\u51FA\u6765\u4E86\u3002\u62FF\u4E00\u5F20\u65E7\u56FE\u9876\u7684\uFF0C\u4ED6\u770B\u4E0D\u51FA\u6765\u3002","yangLost&sunRedeem"],["xiuhe:smile","\u627F\u513F\u4F1A\u53EB\u7238\u7238\u4E86\u3002"]],route:[{if:"yangLost&sunRedeem",go:"e2k",set:{yangLost:!1}},{go:"e2k"}]},e2k:{bg:"e2_victory",music:"drama",lines:["\u6C11\u56FD\u4E09\u5341\u56DB\u5E74\uFF0C\u6297\u6218\u80DC\u5229\u3002\u5317\u5E73\u6EE1\u8857\u97AD\u70AE\u3002","\u5B66\u793E\u7684\u6D4B\u7A3F\u539F\u4EF6\u9646\u7EED\u8FD8\u4E86\u56DE\u6765\u3002",["shouzhuo:smile","\u6BBF\u6837\u8FD8\u5728\u3002","!yangLost"],[null,"\u53EA\u6709\u90A3\u5F20\u6BBF\u6837\uFF0C\u518D\u4E5F\u6CA1\u6709\u627E\u5230\u3002","yangLost"],[null,"\u5B59\u4E16\u660C\u540E\u6765\u53BB\u4E86\u5357\u65B9\uFF0C\u4E34\u8D70\u524D\u7559\u4E0B\u4E00\u53E5\u8BDD\uFF1A\u300C\u6211\u6B20\u987E\u5BB6\u7684\u3002\u300D","sunRedeem"],"\u5B88\u62D9\u628A\u6BBF\u6837\u3001\u6D4B\u7A3F\u7684\u526F\u672C\u548C\u81EA\u5DF1\u7684\u4FE1\u653E\u8FDB\u957F\u5E9A\u7684\u6728\u5323\uFF0C\u91CD\u65B0\u9501\u597D\uFF0C\u4EA4\u7ED9\u4E86\u513F\u5B50\u987E\u627F\u2014\u2014\u5C31\u662F\u4F60\u7684\u7237\u7237\u3002","\u4ED6\u5728\u624B\u8BB0\u91CC\u5199\u4E0B\uFF1A\u300C\u56FE\u5728\uFF0C\u4EBA\u5728\uFF0C\u624B\u827A\u5C31\u5728\u3002\u300D"],next:"e3cut"},e3cut:{cut:{era:"\u4E8C\u3007\u4E8C\u516D\u5E74",place:"\u5317\u4EAC",title:"\u7B2C\u4E09\u5377 \xB7 \u5FF5",bg:"e3_workshop",music:"theme"},next:"e3a"},e3a:{bg:"e3_demolish",days:6,sfx:"engine",lines:["\u5408\u4E0A\u624B\u8BB0\uFF0C\u5929\u5DF2\u7ECF\u4EAE\u4E86\u3002\u9662\u5916\u4F20\u6765\u53D1\u52A8\u673A\u7684\u58F0\u97F3\u3002","\u4E00\u53F0\u6316\u6398\u673A\u505C\u5728\u5899\u6839\uFF0C\u4E00\u4E2A\u6234\u91D1\u4E1D\u773C\u955C\u7684\u7537\u4EBA\u63A8\u95E8\u8FDB\u6765\u3002",["qian:smile","\u987E\u5148\u751F\uFF1F\u6211\u59D3\u94B1\uFF0C\u94B1\u6C0F\u6587\u65C5\u3002\u8FD9\u7247\u8981\u5EFA\u300C\u975E\u9057\u5C0F\u9547\u300D\uFF0C\u60A8\u8FD9\u8001\u5B85\uFF0C\u6B63\u597D\u5728\u89C4\u5212\u91CC\u3002"],["qian","\u53E6\u5916\uFF0C\u6211\u5BF9\u60A8\u5BB6\u90A3\u53EA\u6728\u5323\u5F88\u611F\u5174\u8DA3\u3002\u4E24\u767E\u4E07\uFF0C\u60A8\u5F00\u4E2A\u4EF7\u4E5F\u884C\u3002"]],choices:[{label:"\u300C\u4E0D\u5356\uFF0C\u4E5F\u4E0D\u642C\u3002\u300D",go:"e3a2",set:{qianRefuse:!0}},{label:"\u300C\u8BA9\u6211\u60F3\u60F3\u3002\u300D",go:"e3a2",set:{stall:!0}}]},e3a2:{bg:"e3_demolish",lines:[["qian:angry","\u90A3\u5C31\u6309\u901A\u77E5\u6765\u3002\u516D\u5929\u540E\uFF0C\u8FD9\u91CC\u5C31\u662F\u4E00\u7247\u5E73\u5730\u3002","qianRefuse"],["qian:smile","\u597D\uFF0C\u6211\u7B49\u60A8\u6D88\u606F\u3002\u4E0D\u8FC7\u65F6\u95F4\u4E0D\u7B49\u4EBA\uFF0C\u516D\u5929\u3002","stall"],"\u94B1\u603B\u8D70\u5230\u95E8\u53E3\uFF0C\u53C8\u56DE\u5934\u770B\u4E86\u4E00\u773C\u90A3\u53EA\u6728\u5323\u3002",["qian","\u5BF9\u4E86\uFF0C\u6211\u5916\u516C\u59D3\u5B59\uFF0C\u4EE5\u524D\u5728\u7409\u7483\u5382\u5F00\u53E4\u73A9\u94FA\u3002\u4ED6\u5E38\u8BF4\uFF0C\u987E\u5BB6\u6B20\u4ED6\u4E00\u6837\u4E1C\u897F\u3002"]],next:"e3a3"},e3a3:{bg:"e3_workshop",lines:["\u4F60\u53EB\u987E\u5FF5\uFF0C\u5728\u5317\u4EAC\u4E00\u652F\u53E4\u5EFA\u4FEE\u7F2E\u961F\u5F53\u4E86\u4E09\u5E74\u5B66\u5F92\u3002","\u5899\u8FB9\u7ACB\u7740\u7237\u7237\u7684\u7F57\u6C49\u5E8A\uFF0C\u56F4\u5B50\u677E\u4E86\u3002\u4F60\u60F3\u8D77\u5C0F\u65F6\u5019\u4ED6\u8BF4\u8FC7\uFF1A\u8FD9\u56F4\u5B50\uFF0C\u662F\u80FD\u62C6\u4E0B\u6765\u7684\u3002"],puzzle:"zouma",solved:"e3b"},e3b:{bg:"e3_workshop",card:"zouma",lines:["\u5148\u5F80\u65C1\u8FB9\u4E00\u63A8\uFF0C\u518D\u5F80\u4E0A\u4E00\u63D0\uFF0C\u56F4\u5B50\u5C31\u4E0B\u6765\u4E86\u3002","\u56F4\u5B50\u80CC\u9762\uFF0C\u7237\u7237\u7528\u94C5\u7B14\u5199\u7740\uFF1A\u300C\u5FF5\u5FF5\uFF0C\u5F00\u5323\u7684\u6CD5\u5B50\u5728\u624B\u8BB0\u91CC\u3002\u82E5\u6709\u4EBA\u6765\u8981\u6BBF\u6837\uFF0C\u53BB\u627E\u6797\u5DE5\u3002\u300D"],next:"e3c"},e3c:{bg:"e3_site",days:5,lines:["\u4F60\u56DE\u5230\u6545\u5BAB\u4FEE\u7F2E\u5DE5\u5730\u3002\u4E00\u5EA7\u6BBF\u7684\u6A90\u4E0B\uFF0C\u4E00\u7EC4\u6597\u6831\u7CDF\u673D\u4E86\uFF0C\u8981\u62C6\u4E0B\u6765\u4FEE\u8865\uFF0C\u518D\u539F\u6837\u88C5\u56DE\u3002",["lin","\u6BCF\u4E00\u4EF6\u90FD\u8981\u62CD\u7167\u3001\u626B\u63CF\u3001\u7F16\u53F7\u3002\u62C6\u9519\u4E00\u4EF6\uFF0C\u88C5\u56DE\u53BB\u5C31\u4E0D\u662F\u539F\u6765\u90A3\u7EC4\u4E86\u3002"],"\u4E00\u4E2A\u7A7F\u98CE\u8863\u7684\u4EBA\u6324\u8FC7\u6765\uFF0C\u80F8\u524D\u6302\u7740\u535A\u7269\u9986\u7684\u5DE5\u4F5C\u724C\u3002",["xu:smile","\u987E\u5FF5\uFF01\u771F\u662F\u4F60\u3002\u6211\u6765\u501F\u51E0\u4EF6\u8001\u6784\u4EF6\u505A\u5C55\u89C8\u2014\u2014\u94B1\u6C0F\u6587\u65C5\u6350\u7684\uFF0C\u8BF4\u662F\u660E\u4EE3\u539F\u7269\u3002"],["nian","\u8BB8\u6674\uFF1F\u2026\u2026\u94B1\u6C0F\u6587\u65C5\uFF1F"]],choices:[{label:"\u300C\u8FD9\u4E8B\u6709\u70B9\u8E4A\u8DF7\uFF0C\u56DE\u5934\u7EC6\u8BF4\u3002\u300D",go:"e3c_p",aff:{xu:1}},{label:"\u5148\u4E13\u5FC3\u5E72\u6D3B",go:"e3c_p",aff:{lin:1}}]},e3c_p:{bg:"e3_site",lines:["\u4ECE\u6700\u4E0A\u9762\u5F00\u59CB\uFF0C\u4E00\u4EF6\u4E00\u4EF6\u5F80\u4E0B\u62C6\u3002"],puzzle:"dougong2",solved:"e3d"},e3d:{bg:"e3_site",card:"dougong2",lines:[["lin:sad","\u5947\u602A\u3002\u8FD9\u51E0\u4EF6\u7684\u7F16\u53F7\u662F\u5BF9\u7684\uFF0C\u53EF\u6728\u5934\u662F\u65B0\u7684\u2014\u2014\u505A\u65E7\u505A\u5F97\u5F88\u597D\uFF0C\u4F46\u5E74\u8F6E\u4E0D\u5BF9\u3002"],["lin:angry","\u8001\u6784\u4EF6\u88AB\u4EBA\u6362\u8D70\u4E86\u3002\u627F\u5305\u8FD9\u7247\u5DE5\u7A0B\u7684\u5206\u5305\u5546\u2026\u2026\u662F\u94B1\u6C0F\u6587\u65C5\u3002"],["lin","\u6211\u4EEC\u8981\u8BC1\u636E\u3002\u5149\u8BF4\u5E74\u8F6E\u4E0D\u5BF9\uFF0C\u4ED6\u4EEC\u80FD\u63A8\u8BF4\u662F\u65E9\u5E74\u4FEE\u7F2E\u65F6\u6362\u7684\u3002"],["nian","\u5982\u679C\u6709\u4E00\u5F20\u6C38\u4E50\u5E74\u95F4\u7684\u539F\u6837\uFF0C\u6807\u7740\u6BCF\u4E00\u4EF6\u7684\u5C3A\u5BF8\u548C\u7528\u6599\u5462\uFF1F"],["lin:sad","\u2026\u2026\u4F60\u7237\u7237\u63D0\u8FC7\u3002\u4ED6\u8BF4\u90A3\u5F20\u6837\u5B50\uFF0C\u5728\u987E\u5BB6\u3002"]],choices:[{label:"\u300C\u6211\u4F1A\u628A\u5B83\u627E\u51FA\u6765\u3002\u300D",go:"e3e",aff:{lin:1}},{label:"\u300C\u6211\u5148\u53BB\u95EE\u95EE\u8BB8\u6674\u3002\u300D",go:"e3e",aff:{xu:1}}]},e3e:{bg:"e3_museum",days:4,lines:["\u535A\u7269\u9986\u7684\u5C55\u5385\u91CC\uFF0C\u8BB8\u6674\u9886\u4F60\u770B\u4E00\u5EA7\u4E95\u5E72\u5F0F\u6728\u695E\u623F\u7684\u6A21\u578B\u3002",["xu","\u8FD9\u6A21\u578B\u662F\u4F60\u7237\u7237\u516B\u5341\u5E74\u4EE3\u6350\u7684\u3002\u6863\u6848\u4E0A\u8BF4\uFF0C\u4ED6\u5728\u6700\u5E95\u4E0B\u4E00\u5C42\u539F\u6728\u91CC\u7559\u4E86\u4E1C\u897F\uFF0C\u8BA9\u300C\u987E\u5BB6\u540E\u4EBA\u300D\u6765\u53D6\u3002"],["xu:smile","\u6211\u7B49\u4E86\u597D\u591A\u5E74\uFF0C\u5C31\u7B49\u4F60\u6765\u62C6\u3002"]],puzzle:"jinggan",solved:"e3e2"},e3e2:{bg:"e3_museum",card:"jinggan",lines:["\u6700\u5E95\u5C42\u7684\u539F\u6728\u91CC\uFF0C\u5D4C\u7740\u4E00\u5F20\u6298\u5F97\u5F88\u5C0F\u7684\u7EB8\uFF0C\u662F\u7237\u7237\u7684\u5B57\uFF1A",["grandpa","\u300C\u94B1\u5BB6\u8981\u7684\u662F\u6BBF\u6837\u3002\u4ED6\u4EEC\u60F3\u62FF\u5B83\u7ED9\u5047\u6784\u4EF6\u4F5C\u8BC1\u2014\u2014\u6216\u8005\u70E7\u4E86\u5B83\uFF0C\u8BA9\u8C01\u4E5F\u5BF9\u4E0D\u4E0A\u3002\u522B\u8BA9\u4ED6\u4EEC\u5F97\u624B\u3002\u300D"],["xu:sad","\u6211\u67E5\u8FC7\u94B1\u6C0F\u6587\u65C5\u6350\u7684\u90A3\u4E9B\u300C\u660E\u4EE3\u539F\u7269\u300D\uFF0C\u6765\u8DEF\u5168\u662F\u7A7A\u7684\u3002"]],choices:[{label:"\u300C\u4ECA\u665A\u53BB\u5DE5\u5730\uFF0C\u4F60\u5728\u5916\u9762\u5E2E\u6211\u76EF\u7740\u3002\u300D",go:"e3f",aff:{xu:1}},{label:"\u300C\u6211\u4E00\u4E2A\u4EBA\u53BB\uFF0C\u4F60\u522B\u63BA\u548C\u3002\u300D",go:"e3f",aff:{xu:-1}}]},e3f:{bg:"e3_night",days:2,music:"tense",sfx:"night",lines:["\u540E\u534A\u591C\uFF0C\u5DE5\u5730\u4E0A\u505C\u7740\u4E00\u8F86\u6CA1\u5F00\u706F\u7684\u8D27\u8F66\u3002\u51E0\u4E2A\u4EBA\u6B63\u628A\u6597\u6831\u6784\u4EF6\u4E00\u4EF6\u4EF6\u5F80\u8F66\u4E0A\u642C\u3002",["xu","\u6211\u5728\u5916\u9762\u5F55\u7740\u3002\u8F6C\u89D2\u90A3\u7EC4\u7B8D\u5934\u69AB\u662F\u4ECA\u665A\u8981\u6362\u7684\u2014\u2014\u4F60\u53BB\u770B\u770B\u4E0A\u9762\u7684\u65E7\u58A8\u8BB0\uFF0C\u4FDD\u5B89\u5341\u5206\u949F\u4E00\u5708\u3002","aff.xu>=2"],"\u8F6C\u89D2\u90A3\u7EC4\u7B8D\u5934\u69AB\u7684\u523B\u53E3\u91CC\uFF0C\u5E94\u8BE5\u8FD8\u7559\u7740\u5F53\u5E74\u7684\u58A8\u8BB0\u3002\u53EA\u8981\u62CD\u4E0B\u6765\uFF0C\u5C31\u80FD\u5BF9\u4E0A\u6BBF\u6837\u3002"],puzzle:"gutou",timer:70,solved:"e3f_ok",failed:"e3f_fail"},e3f_ok:{bg:"e3_night",card:"gutou",set:{evidence:!0},lines:["\u523B\u53E3\u91CC\u679C\u7136\u6709\u4E00\u884C\u5C0F\u5B57\u58A8\u8BB0\uFF1A\u300C\u6B63\u7EDF\u516D\u5E74 \xB7 \u5949\u5929\u6BBF \xB7 \u4E1C\u5357\u89D2\u300D\u3002\u4F60\u62CD\u4E0B\u7167\u7247\uFF0C\u9000\u8FDB\u4E86\u9634\u5F71\u91CC\u3002",["xu:smile","\u8D27\u8F66\u7684\u8F66\u724C\uFF0C\u6211\u4E5F\u62CD\u5230\u4E86\u3002","aff.xu>=2"]],next:"e3g"},e3f_fail:{bg:"e3_night",sfx:"hit",lines:["\u624B\u7535\u5149\u7167\u4E86\u8FC7\u6765\u3002",["qian:angry","\u987E\u5148\u751F\uFF0C\u5927\u534A\u591C\u7684\uFF0C\u8DD1\u5230\u6211\u7684\u5DE5\u5730\u4E0A\u6765\u505A\u4EC0\u4E48\uFF1F"],"\u624B\u673A\u88AB\u593A\u8D70\uFF0C\u7167\u7247\u5220\u5F97\u4E00\u5E72\u4E8C\u51C0\u3002",["xu","\u6211\u5F55\u5230\u4E86\u3002\u4EBA\u3001\u8F66\u3001\u8F66\u724C\uFF0C\u90FD\u5728\u3002","aff.xu>=3"]],route:[{if:"aff.xu>=3",go:"e3g",set:{evidence:!0}},{go:"e3g",set:{evidence:!1}}]},e3g:{bg:"e3_office",days:1,lines:[["qian:smile","\u987E\u5148\u751F\uFF0C\u5F00\u95E8\u89C1\u5C71\u5427\u3002\u6BBF\u6837\u7ED9\u6211\uFF0C\u8001\u5B85\u6211\u4E0D\u62C6\u4E86\uFF0C\u4E24\u767E\u4E07\u7167\u4ED8\u3002"],["qian","\u4F60\u77E5\u9053\u6211\u5916\u516C\u5F53\u5E74\u4E3A\u4E86\u90A3\u5F20\u56FE\uFF0C\u5DEE\u70B9\u88AB\u65E5\u672C\u4EBA\u67AA\u6BD9\u5417\uFF1F\u987E\u5BB6\u6B20\u6211\u4EEC\u5B59\u5BB6\u7684\u3002"],["nian:angry","\u4F60\u5916\u516C\u540E\u6765\u628A\u6BBF\u6837\u6362\u4E86\u56DE\u6765\uFF0C\u8FD8\u7ED9\u4E86\u987E\u5BB6\u3002\u4ED6\u7559\u4E0B\u7684\u8BDD\u662F\u2014\u2014\u300C\u6211\u6B20\u987E\u5BB6\u7684\u300D\u3002","sunRedeem"],[null,"\u4F60\u6CA1\u6709\u8BF4\u8BDD\u3002\u624B\u8BB0\u91CC\u5173\u4E8E\u5B59\u4E16\u660C\u7684\u90A3\u51E0\u9875\uFF0C\u4F60\u8BFB\u5F97\u5F88\u6E05\u695A\u3002","!sunRedeem"],["qian:angry","\u660E\u5929\u65E9\u4E0A\u516B\u70B9\uFF0C\u6316\u6398\u673A\u8FDB\u573A\u3002"]],next:"e3h"},e3h:{bg:"e3_workshop",lines:["\u56DE\u5230\u8001\u5B85\uFF0C\u4F60\u7FFB\u5230\u624B\u8BB0\u6700\u540E\u4E00\u9875\uFF0C\u662F\u7237\u7237\u7684\u5B57\uFF1A\u300C\u94A5\u5319\u7684\u8D70\u6CD5\uFF0C\u523B\u5728\u683C\u5B50\u7A97\u7684\u69AB\u5934\u4E0A\u3002\u300D","\u7A97\u68C2\u6A2A\u7AD6\u76F8\u4EA4\uFF0C\u6BCF\u4E00\u5904\u90FD\u662F\u683C\u80A9\u69AB\u3002"],puzzle:"gejian",solved:"e3h2"},e3h2:{bg:"e3_workshop",card:"gejian",lines:["\u69AB\u5934\u4E0A\u523B\u7740\u4E09\u884C\u5C0F\u5B57\uFF0C\u662F\u4E09\u4EE3\u4EBA\u7559\u4E0B\u7684\u90A3\u4E09\u53E5\u8BDD\uFF1A","\u987A\u7740\u5B83\u7684\u8DEF\u8D70\u3002\u5148\u9000\u5C0F\u7684\uFF0C\u518D\u9000\u5927\u7684\u3002\u6709\u65F6\u5019\uFF0C\u5F97\u5148\u8BA9\u4E00\u6B65\u3002"],next:"e3i"},e3i:{bg:"e3_demolish",days:0,music:"tense",sfx:"engine",lines:["\u65E9\u4E0A\u4E03\u70B9\u4E94\u5341\u3002\u6316\u6398\u673A\u5DF2\u7ECF\u53D1\u52A8\u4E86\uFF0C\u94F2\u6597\u60AC\u5728\u9662\u5899\u4E0A\u65B9\u3002",["lin:angry","\u987E\u5FF5\uFF01\u6587\u7269\u5C40\u7684\u4EBA\u5728\u8DEF\u4E0A\u4E86\uFF0C\u4F60\u5F97\u5148\u628A\u6BBF\u6837\u62FF\u51FA\u6765\uFF01","aff.lin>=2"],["xu:sad","\u6211\u6321\u5728\u95E8\u53E3\uFF0C\u80FD\u62D6\u4E00\u4F1A\u513F\u662F\u4E00\u4F1A\u513F\u2014\u2014\u5FEB\uFF01","aff.xu>=2"],"\u4F60\u628A\u6728\u5323\u62B1\u5230\u5DE5\u4F5C\u53F0\u4E0A\u3002"],puzzle:"burr",timer:150,solved:"e3j_cg",failed:"e3i_fail"},e3i_fail:{bg:"e3_demolish",sfx:"hit",lines:["\u9662\u5899\u5012\u4E0B\u6765\u7684\u65F6\u5019\uFF0C\u4F60\u8FD8\u6CA1\u89E3\u5F00\u6700\u540E\u4E24\u6839\u6728\u6761\u3002\u94B1\u603B\u7684\u4EBA\u51B2\u8FDB\u6765\uFF0C\u4E00\u628A\u62A2\u8D70\u4E86\u6728\u5323\u3002",["xu:angry","\u4F4F\u624B\uFF01\u6211\u5168\u5F55\u4E0B\u6765\u4E86\u2014\u2014\u62A2\u593A\u6587\u7269\uFF0C\u4F60\u4EEC\u8DD1\u4E0D\u6389\uFF01","aff.xu>=2"],[null,"\u8F66\u5F00\u8D70\u4E86\u3002\u4F60\u8DEA\u5728\u788E\u7816\u91CC\uFF0C\u624B\u91CC\u53EA\u5269\u4E0B\u90A3\u672C\u624B\u8BB0\u3002","aff.xu<2"]],route:[{if:"aff.xu>=2",go:"e3j_cg",set:{boxRecovered:!0}},{go:"end",set:{boxLost:!0}}]},e3j_cg:{cg:"flash_a4",still:"e1_shed",caption:"\u516D\u767E\u5E74\u524D\uFF0C\u4E00\u4E2A\u5E74\u8F7B\u5320\u4EBA\u628A\u6BBF\u6837\u6298\u597D\uFF0C\u653E\u8FDB\u6728\u5323\uFF0C\u63D2\u56DE\u4E86\u6700\u540E\u4E00\u6839\u94A5\u5319\u3002",next:"e3j"},e3j:{bg:"e3_items",card:"burr",music:"drama",lines:["\u6700\u540E\u51E0\u6839\u6728\u6761\u5206\u5F00\uFF0C\u5323\u5E95\u8F7B\u8F7B\u843D\u4E86\u4E0B\u6765\u3002","\u5323\u91CC\u662F\u4E00\u5377\u6CB9\u7EB8\u5305\u7740\u7684\u6BBF\u6837\uFF0C\u4E00\u53E0\u53D1\u9EC4\u7684\u6D4B\u7A3F\uFF0C\u548C\u51E0\u5C01\u4FE1\u3002",[null,"\u53EA\u6709\u6BBF\u6837\u7684\u4F4D\u7F6E\u7A7A\u7740\uFF0C\u538B\u7740\u4E00\u5F20\u5C0F\u91CE\u5F53\u5E74\u7559\u4E0B\u7684\u6536\u636E\u3002","yangLost"],[null,"\u8FD8\u6709\u4E00\u5C0F\u7247\u892A\u4E86\u8272\u7684\u9752\u7EFF\u5F69\u753B\uFF0C\u662F\u5B81\u6C85\u753B\u5728\u753B\u7BB1\u4E0A\u7684\u3002","gift"]],choices:[{label:"\u8BFB\u90A3\u51E0\u5C01\u4FE1",go:"e3k"}]},e3k:{bg:"e3_items",lines:[["changgeng","\u300C\u540E\u6765\u7684\u4EBA\uFF1A\u6BBF\u6837\u5728\uFF0C\u6BBF\u5C31\u5728\u3002\u597D\u6728\u5934\uFF0C\u8981\u8BA9\u540E\u4EBA\u4E5F\u62C6\u5F97\u5F00\uFF1B\u597D\u623F\u5B50\uFF0C\u8981\u8BA9\u540E\u4EBA\u4E5F\u4FEE\u5F97\u56DE\u3002\u300D"],["changgeng","\u300C\u5E08\u7236\u8BF4\uFF0C\u624B\u827A\u4E0D\u9760\u9489\u5B50\uFF0C\u4EBA\u4E5F\u4E00\u6837\u3002\u4ED6\u6559\u6211\u7684\uFF0C\u6211\u90FD\u753B\u8FDB\u4E86\u6BBF\u6837\u91CC\u3002\u300D","hanSaved"],["changgeng:sad","\u300C\u5E08\u7236\u6CA1\u80FD\u51FA\u72F1\u3002\u6BBF\u6837\u4E0A\u7684\u6BCF\u4E00\u4E2A\u5C3A\u5BF8\uFF0C\u90FD\u662F\u4ED6\u5728\u7262\u91CC\u4E00\u5BF8\u4E00\u5BF8\u5FF5\u7ED9\u6211\u7684\u3002\u300D","!hanSaved"],["shouzhuo","\u300C\u56FE\u5728\uFF0C\u4EBA\u5728\uFF0C\u624B\u827A\u5C31\u5728\u3002\u79C0\u79BE\u4E00\u4E2A\u4EBA\u5E26\u5927\u4E86\u627F\u513F\uFF0C\u8FD9\u8F88\u5B50\u6211\u6B20\u5979\u7684\u3002\u300D","path=south"],["shouzhuo","\u300C\u90A3\u51E0\u5E74\uFF0C\u6211\u7ED9\u62FF\u8D70\u6211\u4EEC\u4E1C\u897F\u7684\u4EBA\u505A\u5BB6\u5177\uFF0C\u6BCF\u4E00\u4EF6\u90FD\u7559\u4E86\u6D3B\u69AB\u3002\u4ED6\u4EEC\u62FF\u4E0D\u8D70\u7684\u3002\u300D","path=stay"],["grandpa","\u300C\u5FF5\u5FF5\uFF1A\u6728\u5934\u548C\u6728\u5934\u4E4B\u95F4\uFF0C\u4E0D\u9760\u9489\u5B50\uFF0C\u9760\u4E92\u76F8\u54AC\u7740\uFF0C\u4E5F\u4E92\u76F8\u8BA9\u7740\u3002\u4EBA\u4E5F\u4E00\u6837\uFF0C\u4E00\u4E2A\u56FD\u5BB6\u4E5F\u4E00\u6837\u3002\u5F80\u540E\u7684\u4E8B\uFF0C\u4F60\u81EA\u5DF1\u62FF\u4E3B\u610F\u3002\u300D"]],choices:[{label:"\u628A\u6BBF\u6837\u4EA4\u7ED9\u6545\u5BAB\u4FEE\u7F2E\u961F",go:"end",set:{final:"give"},aff:{lin:1}},{label:"\u7559\u5728\u4F5C\u574A\uFF0C\u81EA\u5DF1\u5B88\u7740",go:"end",set:{final:"keep"}}]},end:{route:[{if:"boxLost",go:"@end:bad"},{if:"evidence&final=give&!yangLost",go:"end_true"},{go:"end_normal"}]},end_normal:{bg:"e3_workshop",lines:[[null,"\u6BBF\u6837\u4EA4\u4E86\u4E0A\u53BB\u3002\u53EF\u6CA1\u6709\u90A3\u665A\u7684\u8BC1\u636E\uFF0C\u94B1\u6C0F\u6587\u65C5\u63A8\u8BF4\u662F\u300C\u65E9\u5E74\u4FEE\u7F2E\u8BEF\u6362\u300D\uFF0C\u8D54\u4E86\u4E00\u7B14\u94B1\u4E86\u4E8B\u3002","final=give&!evidence"],[null,"\u6BBF\u6837\u65E9\u5DF2\u9057\u5931\u3002\u4F60\u4EEC\u53EA\u80FD\u51ED\u6D4B\u7A3F\u548C\u624B\u827A\uFF0C\u4E00\u4EF6\u4EF6\u53BB\u8FA8\u8BA4\u54EA\u4E9B\u662F\u539F\u7269\u3002","yangLost"],[null,"\u4F60\u628A\u6BBF\u6837\u9501\u56DE\u4E86\u6728\u5323\u3002\u8001\u5B85\u6700\u540E\u8FD8\u662F\u62C6\u4E86\uFF0C\u4F60\u628A\u4F5C\u574A\u642C\u8FDB\u4E86\u57CE\u91CC\u4E00\u95F4\u5C0F\u5C4B\u3002","final=keep"],"\u4F60\u5728\u95E8\u53E3\u6302\u4E0A\u4E00\u5757\u6728\u724C\uFF1A\u987E\u6C0F\u6728\u4F5C\u3002","\u6709\u4EBA\u95EE\u4F60\uFF0C\u5B88\u4E86\u516D\u767E\u5E74\uFF0C\u503C\u5417\uFF1F\u4F60\u60F3\u4E86\u60F3\uFF0C\u628A\u6728\u5323\u5E95\u7684\u516D\u6839\u6728\u6761\uFF0C\u4E00\u6839\u6839\u63D2\u4E86\u56DE\u53BB\u3002"],next:"@end:normal"},end_true:{bg:"e3_dawn",music:"drama",lines:["\u6BBF\u6837\u548C\u90A3\u51E0\u5F20\u58A8\u8BB0\u7167\u7247\uFF0C\u4E00\u8D77\u9001\u8FDB\u4E86\u6587\u7269\u5C40\u3002\u88AB\u6362\u8D70\u7684\u6784\u4EF6\u5728\u94B1\u6C0F\u6587\u65C5\u7684\u4ED3\u5E93\u91CC\u627E\u5230\u4E86\uFF0C\u4E00\u4EF6\u4E0D\u5C11\u3002",["lin:smile","\u8001\u5B85\u5217\u8FDB\u4E86\u6587\u7269\u4FDD\u62A4\u540D\u5F55\u3002\u62C6\u4E0D\u4E86\u4E86\u3002"],["xu:smile","\u5C55\u89C8\u6362\u4E86\u4E2A\u540D\u5B57\uFF0C\u53EB\u300C\u6BBF\u6837\u5728\uFF0C\u6BBF\u5C31\u5728\u300D\u3002\u6728\u5323\u653E\u5728\u6B63\u4E2D\u95F4\u3002","aff.xu>=2"],"\u90A3\u7EC4\u6597\u6831\u6309\u6BBF\u6837\u4E00\u4EF6\u4EF6\u88C5\u4E86\u56DE\u53BB\u3002\u6700\u540E\u4E00\u4EF6\u5F52\u4F4D\u7684\u65F6\u5019\uFF0C\u4F60\u542C\u89C1\u5F88\u8F7B\u7684\u4E00\u58F0\u300C\u5494\u300D\u3002"],cgAfter:"ending_b",next:"@end:true"}};var Fe=i=>document.querySelector(i),jt=Fe("#app"),gn=i=>new Promise(e=>setTimeout(e,i));function hi(i){jt.dataset.mode=i,Bh(i==="puzzle")}var Hf="\u4E00\u4E8C\u4E09\u56DB\u4E94\u516D\u4E03\u516B\u4E5D\u5341",Zf=i=>i<=10?Hf[i-1]:i<20?"\u5341"+Hf[i-11]:"\u4E8C\u5341",$f=i=>i.startsWith("e1")?"\u7B2C\u4E00\u5377 \xB7 \u6728\u751F":i.startsWith("e2")?"\u7B2C\u4E8C\u5377 \xB7 \u5B88\u62D9":i.startsWith("e3")||i.startsWith("end")?"\u7B2C\u4E09\u5377 \xB7 \u5FF5":"\u5E8F \xB7 \u6728\u5323",Jf="sunyu-save-v3",jf=()=>({node:null,cards:[],clues:[],flags:{},aff:{},met:[],solved:[],endings:[]}),$e=jf();try{Object.assign($e,JSON.parse(localStorage.getItem(Jf))||{})}catch{}var Tn=()=>{try{localStorage.setItem(Jf,JSON.stringify($e))}catch{}};function wv(i){let e=i[0]==="!";e&&(i=i.slice(1));let t,n=i.match(/^aff\.(\w+)(>=|<=|>|<|==)(-?\d+)$/);if(n){let s=$e.aff[n[1]]||0,r=+n[3];t={">=":s>=r,"<=":s<=r,">":s>r,"<":s<r,"==":s===r}[n[2]]}else if(i.includes("=")){let[s,r]=i.split("=");t=String($e.flags[s])===r}else t=!!$e.flags[i];return e?!t:t}var Xh=i=>!i||i.split("|").some(e=>e.split("&").every(wv));function Kf(i){for(let[e,t]of Object.entries(i||{}))$e.aff[e]=($e.aff[e]||0)+t,$e.met.includes(e)||$e.met.push(e),Av(e,t);Tn()}function Av(i,e){let t=document.createElement("div");t.className="toast"+(e<0?" down":""),t.innerHTML=`<img src="${__A(`art/ch/${i}.webp`)}" alt=""><span>${oi[i].name}</span><b>${e>0?"\u2665".repeat(Math.min(e,3))+" \u60C5\u5206\u52A0\u6DF1":"\u60C5\u5206\u758F\u8FDC"}</b>`,Fe("#toasts").appendChild(t),Qe.chime(e>0),setTimeout(()=>t.remove(),2700)}var Rv=["e1_site","e1_shed","e1_dougong","e1_fire","e1_farewell","e1_office","e1_jail","e1_ruins","e1_lane","e1_yard","e1_paint","e2_hutong","e2_palace","e2_attic","e2_gate","e2_road","e2_curio","e2_station","e2_occupied","e2_search","e2_lizhuang","e2_victory","e3_rain","e3_workshop","e3_site","e3_dawn","e3_box","e3_items","e3_demolish","e3_office","e3_museum","e3_night"],Cv=i=>{let e=new Image;e.src=i};setTimeout(()=>Rv.forEach(i=>Cv(__A(`art/bg/${i}.jpg`))),1500);var qh=0,Gf=null;function Qf(i){if(!i||Gf===i)return;let e=[...document.querySelectorAll(".bgl")],t=e[qh^1];t.style.backgroundImage=`url(${__A(`art/bg/${i}.jpg`)})`,t.classList.add("on"),e[qh].classList.remove("on"),qh^=1,Gf=i,Fe("#puzzle-bg").style.backgroundImage=`url(${__A(`art/bg/${i}.jpg`)})`,Iv()}var Li=Fe("#fx"),Rt=Li.getContext("2d"),li="dust",Ks=[];function ep(i){if(i===li&&Ks.length)return;li=i,Ks=[];let e={dust:46,embers:90,ash:70,rain:220,none:0}[i]||0;for(let t=0;t<e;t++)Ks.push(tp(!0))}function tp(i){let e=Li.width,t=Li.height,n=Math.random;return li==="embers"?{x:n()*e,y:i?n()*t:t+10,vx:(n()-.5)*30,vy:-(40+n()*90),s:1+n()*2.6,a:n(),ph:n()*6}:li==="ash"?{x:n()*e,y:i?n()*t:-10,vx:15+n()*30,vy:18+n()*30,s:1.5+n()*3,a:n(),ph:n()*6}:li==="rain"?{x:n()*e*1.2,y:i?n()*t:-40,vx:-120,vy:900+n()*400,s:12+n()*16,a:.15+n()*.25}:{x:n()*e,y:n()*t,vx:(n()-.5)*8,vy:-2-n()*8,s:.8+n()*2,a:n(),ph:n()*6}}var Wf=performance.now();function np(i){let e=Math.min(.05,(i-Wf)/1e3);Wf=i;let t=Math.min(devicePixelRatio,2);Li.width!==innerWidth*t&&(Li.width=innerWidth*t,Li.height=innerHeight*t,ep(li));let n=Li.width,s=Li.height;if(Rt.clearRect(0,0,n,s),jt.dataset.mode==="vn")for(let r=0;r<Ks.length;r++){let a=Ks[r];if(a.x+=a.vx*e*t,a.y+=a.vy*e*t,a.ph=(a.ph||0)+e*2,a.y<-30||a.y>s+50||a.x>n+50||a.x<-60){Ks[r]=tp(!1);continue}if(li==="rain")Rt.strokeStyle=`rgba(220,225,235,${a.a})`,Rt.lineWidth=t,Rt.beginPath(),Rt.moveTo(a.x,a.y),Rt.lineTo(a.x+a.vx*.02*t,a.y-a.s*t),Rt.stroke();else if(li==="embers"){let o=.5+.5*Math.sin(a.ph*3);Rt.fillStyle=`rgba(255,${150+60*o|0},70,${.35+.5*o})`,Rt.beginPath(),Rt.arc(a.x,a.y,a.s*t,0,6.3),Rt.fill(),a.vx+=Math.sin(a.ph)*10*e}else li==="ash"?(Rt.fillStyle=`rgba(70,62,56,${.25+.3*a.a})`,Rt.save(),Rt.translate(a.x,a.y),Rt.rotate(a.ph),Rt.fillRect(-a.s*t,-a.s*t*.5,a.s*2*t,a.s*t),Rt.restore()):(Rt.fillStyle=`rgba(255,246,225,${.18+.3*(.5+.5*Math.sin(a.ph))})`,Rt.beginPath(),Rt.arc(a.x,a.y,a.s*t,0,6.3),Rt.fill())}requestAnimationFrame(np)}requestAnimationFrame(np);var Pv=i=>["e1_fire"].includes(i)?"embers":["e2_gate","e2_road","e2_search","e2_victory"].includes(i)?"ash":["e3_rain","e1_jail","e2_lizhuang"].includes(i)?"rain":"dust",fa=Fe(".pt.left"),Gl=Fe(".pt.right");function Iv(){for(let i of[fa,Gl])i.classList.remove("on","dim"),i.dataset.k="",i.dataset.src=""}function Lv(i){let[e,t]=(i||"").split(":"),n=e&&oi[e];if(!n||!n.side){for(let o of[fa,Gl])o.dataset.k&&o.classList.add("dim");return}let s=n.side==="left"?fa:Gl,r=s===fa?Gl:fa,a=`${__A(`art/ch/${e}${t?"_"+t:""}.webp`)}`;s.dataset.k!==e?(s.classList.remove("on"),s.dataset.k=e,s.dataset.src=a,setTimeout(()=>{s.dataset.src===a&&(s.src=a,s.classList.add("on"),s.classList.remove("dim"))},s.getAttribute("src")?260:0)):(s.dataset.src!==a&&(s.dataset.src=a,s.classList.add("swap"),setTimeout(()=>{s.src=a,s.classList.remove("swap")},120)),s.classList.add("on"),s.classList.remove("dim")),r.dataset.k&&r.classList.add("dim"),oi[e].aff&&!$e.met.includes(e)&&($e.met.push(e),Tn())}var cn=0,Qs=null;function ip(){if(Qs){let i=Qs;Qs=null,i()}}Fe("#vn").addEventListener("click",i=>{i.target.closest("#choices")||ip()});addEventListener("keydown",i=>{(i.key===" "||i.key==="Enter")&&jt.dataset.mode==="vn"&&(i.preventDefault(),ip())});async function Xf(i,e,t){let n=Fe("#dialog"),s=Fe("#dialog .name"),r=Fe("#dialog .line"),a=i&&oi[i.split(":")[0]];Lv(i),s.textContent=a?a.name:"",s.classList.toggle("on",!!a),r.className="line"+(a?/^「/.test(e)?" letter":"":" narr"),n.classList.add("on"),n.classList.remove("ready"),r.textContent="";let o=!1;Qs=()=>{o=!0};for(let l=1;l<=e.length;l++){if(cn!==t)return!1;if(o)break;r.textContent=e.slice(0,l),await gn(/[，。！？：；…—]/.test(e[l-1])?90:30)}return r.textContent=e,n.classList.add("ready"),await new Promise(l=>{Qs=l}),Qe.page(),cn===t}function Dv(i,e,t,n){let s=Fe("#choices");return s.innerHTML="",s.classList.add("veil"),Fe("#dialog").classList.add("ready"),new Promise(r=>{let a=!1,o=null,l=c=>{a||(a=!0,clearInterval(o),Fe("#ctimer").hidden=!0,jt.classList.remove("danger"),s.innerHTML="",s.classList.remove("veil"),r(c))};if(i.forEach((c,h)=>{let d=document.createElement("button");d.className="choice",d.textContent=c.label,d.style.animationDelay=`${.12*h}s`,d.onclick=async u=>{u.stopPropagation(),!(cn!==e||a)&&(d.classList.add("picked"),Qe.tick(),await gn(260),l(c))},s.appendChild(d)}),t){let c=performance.now(),h=Fe("#ctimer"),d=h.querySelector(".bar"),u=h.querySelector("span");h.hidden=!1,jt.classList.add("danger");let f=-1;o=setInterval(()=>{if(cn!==e){l(null);return}let g=Math.max(0,t-(performance.now()-c)/1e3);d.style.transform=`scaleX(${g/t})`;let y=Math.ceil(g);u.textContent=y,h.classList.toggle("urgent",g<3.2),y!==f&&(f=y,Qe.heartbeat()),g<=0&&l({...n,timedOut:!0})},50)}})}async function Ni(i){let e=Fe("#wipe");e.className="",e.offsetWidth,e.classList.add("in"),await gn(760),await i(),e.classList.remove("in"),e.classList.add("out"),await gn(860),e.classList.remove("out")}function sp(i){return new Promise(e=>i.addEventListener("click",e,{once:!0}))}async function Nv(i,e){let t=Fe("#cut");return await Ni(async()=>{hi("cut"),t.classList.remove("play"),Fe(".cut-bg").style.backgroundImage=`url(${__A(`art/bg/${i.bg}.jpg`)})`,Fe(".cut-era").innerHTML=[...i.era].map((n,s)=>`<span style="animation-delay:${.9+s*.07}s">${n}</span>`).join(""),Fe(".cut-title").innerHTML=[...i.title].map((n,s)=>`<span style="animation-delay:${1.5+s*.12}s">${n===" "?"&nbsp;":n}</span>`).join(""),Fe(".cut-place").textContent=i.place,t.offsetWidth,t.classList.add("play")}),Qe.gong(),i.music&&Qe.music(i.music),await Promise.race([gn(6400),gn(900).then(()=>sp(t))]),cn!==e?!1:(Qf(i.bg),await Ni(async()=>{hi("vn")}),cn===e)}var Hl={};async function Uv(i){return i in Hl||(Hl[i]=!!window.__ASSETS[`art/cg/${i}.mp4`]),Hl[i]}async function qf(i,e,t,n){let s=Fe("#cg"),r=s.querySelector("video"),a=await Uv(i);s.classList.toggle("video",a),s.querySelector(".cg-tag").hidden=!i.startsWith("flash"),s.querySelector(".cg-still").style.backgroundImage=`url(${__A(`art/bg/${e}.jpg`)})`,s.querySelector(".cg-cap").textContent="",s.classList.remove("on"),s.offsetWidth,s.classList.add("on"),Qe.gong();let o;return a?(r.src=`${__A(`art/cg/${i}.mp4`)}`,r.currentTime=0,r.play().catch(()=>{}),o=new Promise(l=>{r.onended=l,r.onerror=()=>gn(4e3).then(l)})):o=gn(6500),(async()=>{await gn(900);for(let l=1;l<=t.length&&s.classList.contains("on");l++)s.querySelector(".cg-cap").textContent=t.slice(0,l),await gn(70)})(),await Promise.race([o,gn(1200).then(()=>sp(s))]),r.pause(),s.classList.remove("on"),await gn(700),cn===n}function Fv(i){let e=Vl[i],t=Fe("#card-ov");return t.querySelector(".c-tag").textContent=e.tag,t.querySelector("h3").innerHTML=e.title+(e.arch?'<span class="arch">\u53E4\u5EFA\u7B51</span>':""),t.querySelector(".c-body").innerHTML=e.text.map(n=>`<p>${n}</p>`).join(""),t.classList.add("on"),setTimeout(()=>Qe.stamp(),700),t._close&&t.removeEventListener("click",t._close),new Promise(n=>{let s=performance.now();t._close=()=>{performance.now()-s<900||(t.removeEventListener("click",t._close),t._close=null,t.classList.remove("on"),Qe.page(),setTimeout(n,350))},t.addEventListener("click",t._close)})}function rp(i="\u89E3"){let e=Fe("#stamp");e.querySelector(".seal").textContent=i,e.classList.remove("play"),e.offsetWidth,e.classList.add("play"),setTimeout(()=>Qe.stamp(),380)}function Zh(i){let e=Fe("#hud-days");i!==void 0&&(e.hidden=i===null,e.textContent=i===0?"\u4ECA\u5929\u5F3A\u62C6":`\u8DDD\u5F3A\u62C6 ${i} \u5929`,e.classList.remove("pulse"),e.offsetWidth,e.classList.add("pulse"))}function $h(){jt.classList.remove("shake"),jt.offsetWidth,jt.classList.add("shake")}function Yf(i){i.set&&Object.assign($e.flags,i.set),i.aff&&Kf(i.aff),Tn()}async function ci(i){let e=++cn;if(Qs=null,i.startsWith("@end"))return Bv(i.split(":")[1]||"normal");let t=Vf[i];if(!t){console.error("\u7F3A\u5C11\u8282\u70B9",i);return}if(t.route){Yf(t);let n=t.route.find(s=>Xh(s.if));return n.set&&Object.assign($e.flags,n.set),Tn(),ci(n.go)}if($e.node=i,Tn(),Fe("#hud-chap").textContent=$f(i),!i.startsWith("e3")&&!i.startsWith("end")&&i!=="p2"&&Zh(null),Fe("#choices").innerHTML="",Fe("#dialog").classList.remove("on"),!(t.cut&&!await Nv(t.cut,e))){if(t.cg)return jt.dataset.mode!=="vn"&&await Ni(async()=>hi("vn")),await qf(t.cg,t.still,t.caption,e)?ci(t.next):void 0;if(!t.cut&&jt.dataset.mode!=="vn"&&await Ni(async()=>hi("vn")),cn===e&&(t.music?Qe.music(t.music):Qe.now||Qe.music("theme"),Qe.cue(t.sfx||null),t.sfx==="hit"&&$h(),Qf(t.bg),ep(t.fx||Pv(t.bg)),t.days!==void 0&&Zh(t.days),Yf(t),!(t.card&&($e.cards.includes(t.card)||$e.cards.push(t.card),Tn(),Fe("#dialog").classList.remove("on"),await Fv(t.card),cn!==e)))){t.clue&&!$e.clues.includes(t.clue)&&($e.clues.push(t.clue),Tn(),Fe("#btn-journal").classList.add("flash"),setTimeout(()=>Fe("#btn-journal").classList.remove("flash"),3200));for(let n of t.lines||[]){let[s,r,a]=typeof n=="string"?[null,n,null]:n;if(Xh(a)&&!await Xf(s,r,e))return}if(t.choices){let n=await Dv(t.choices.filter(s=>Xh(s.if)),e,t.timer,t.onTimeout);return!n||cn!==e||(n.set&&Object.assign($e.flags,n.set),n.aff&&Kf(n.aff),Tn(),n.timedOut&&(Qe.fail(),$h(),n.note&&!await Xf(null,n.note,e)))?void 0:ci(n.go)}if(t.puzzle)return ap({...t,id:i},e);if(!(t.cgAfter&&!await qf(t.cgAfter,"e3_dawn","\u4E09\u4EE3\u4EBA\u7684\u624B\uFF0C\u62C6\u5F00\u8FC7\u3001\u53C8\u88C5\u56DE\u53BB\u7684\uFF0C\u662F\u540C\u4E00\u7EC4\u6597\u6831\u3002",e))&&t.next)return ci(t.next)}}}var Di=null;async function ap(i,e,t=!1){Di=t?null:i;let n=i.puzzle,s=Wh.indexOf(n),r=Vl[n];await Ni(async()=>{Fe("#dialog").classList.remove("on"),hi("puzzle"),Fe("#pz-tag").textContent=`\u7B2C${Zf(s+1)}\u5173 \xB7 ${t?"\u56FE\u8C31":$f(i.id||"").split(" \xB7 ")[0]}`,Fe("#pz-title").textContent=r.title,Fe("#pz-goal").textContent=Bf[n],es(s===0&&!t?`\u70B9\u4E00\u5757\u6728\u5934\u9009\u4E2D\u5B83\uFF0C\u518D\u70B9\u5B83\u65C1\u8FB9\u7684\u7BAD\u5934\u63A8\u4E00\u63A8\u3002
\u62D6\u52A8\u7A7A\u767D\u5904\u8F6C\u89C6\u89D2\uFF0C\u6EDA\u8F6E\u7F29\u653E\u3002`:i.timer&&!t?"\u6765\u4E0D\u53CA\u591A\u60F3\u4E86\uFF01":"\u5148\u770B\u6E05\u695A\uFF0C\u518D\u52A8\u624B\u3002",i.timer&&!t?"block":"tip"),kl(n,{free:t,timer:t?0:i.timer||0})})}function es(i,e="tip"){let t=Fe("#msg");t.hidden=!i,t.className=e,t.textContent=i||"",t.style.animation="none",t.offsetWidth,t.style.animation=""}wt.msg=es;wt.moves=i=>{Fe("#pz-moves").textContent=i?`\u5DF2\u52A8\u4E86 ${i} \u6B65`:""};var Yh=-1;wt.timer=i=>{let e=Fe("#pz-timer");if(i===null){e.hidden=!0,jt.classList.remove("danger"),Yh=-1;return}let t=re?re.timer:i;e.hidden=!1;let n=Math.ceil(i);e.querySelector("span").textContent=`${Math.floor(n/60)}:${String(n%60).padStart(2,"0")}`,e.querySelector(".arc").style.strokeDashoffset=119.4*(1-i/t);let s=i<15;e.classList.toggle("urgent",s),jt.classList.toggle("danger",s);let r=Math.ceil(i);r!==Yh&&re&&re.mode==="puzzle"&&(Yh=r,s?Qe.heartbeat():r%2===0&&Qe.clock(!1))};wt.timeout=async()=>{if(jt.classList.remove("danger"),Qe.fail(),$h(),rp("\u8FDF"),es("\u6765\u4E0D\u53CA\u4E86\u2026\u2026","block"),await gn(2200),Di&&Di.failed){let i=Di.failed;Di=null,ci(i)}};wt.solved=(i,e)=>{rp(),jt.classList.remove("danger"),$e.solved.includes(i)||$e.solved.push(i),Tn(),es(e?"\u62C6\u5F00\u4E86\u3002\u53EF\u4EE5\u70B9\u300C\u6F14\u793A\u62C6\u88C5\u300D\u770B\u5B83\u662F\u600E\u4E48\u5408\u4E0A\u7684\u3002":"\u62C6\u5F00\u4E86\uFF01\u770B\u770B\u5404\u90E8\u5206\u7684\u540D\u5B57\uFF0C\u51C6\u5907\u597D\u4E86\u5C31\u70B9\u300C\u7EE7\u7EED\u300D\u3002","ok")};wt.tools=Ov;function Ov(){let i=n=>document.querySelector(`[data-tool="${n}"]`),e=re,t=e?e.mode:"none";i("hint").hidden=t!=="puzzle",i("reset").hidden=!(t==="puzzle"&&!e.timer||t==="reveal"&&e.free),i("xray").hidden=!(e&&e.chap.xray.length&&t!=="intro"),i("xray").classList.toggle("on",!!(e&&e.xray)),i("demo").hidden=t!=="reveal",i("next").hidden=!(t==="reveal"&&!e.free),i("back").hidden=!(e&&e.free)}Fe("#tools").addEventListener("click",i=>{let e=i.target.dataset&&i.target.dataset.tool,t=re;if(!(!e||!t))if(Qe.tick(),e==="hint"){let n=zf[t.id];es("\u5E08\u7236\u8BF4\uFF1A"+n[Math.min(Vh(),n.length-1)],"hint")}else if(e==="reset")kl(t.id,{free:t.free}),es("\u90FD\u88C5\u56DE\u53BB\u4E86\uFF0C\u91CD\u65B0\u6765\u3002","tip");else if(e==="xray")kh();else if(e==="demo")zh();else if(e==="next"&&Di){let n=Di.solved;Di=null,i.target.hidden=!0,ci(n)}else e==="back"&&zv()});function Wl(){cn++,hi("title"),Qe.ambient(null),Fe(".t-bg").style.backgroundImage=`url(${__A("art/bg/title.jpg")})`;let i=Fe("#t-btns");i.innerHTML="";let e=(n,s)=>{let r=document.createElement("button");r.textContent=n,r.onclick=()=>{Qe.unlock(),Qe.tick(),s()},i.appendChild(r)},t=$e.node&&$e.node!=="start";t&&e("\u7EE7\u7EED",()=>ci($e.node)),e(t?"\u91CD\u65B0\u5F00\u59CB":"\u5F00\u59CB",()=>{let n={cards:$e.cards,endings:$e.endings,solved:$e.solved};$e={...jf(),...n},Tn(),Qe.music("theme"),ci("start")}),$e.cards.length&&e(`\u69AB\u536F\u56FE\u8C31 ${$e.cards.length}/20`,jh)}function Bv(i){cn++,Qe.ambient(null),Zh(null),$e.endings.includes(i)||$e.endings.push(i),Tn();let e=$e.flags,t=kf[i],n=[e.hanSaved?"\u97E9\u5E08\u5085\u6D17\u6E05\u4E86\u51A4\u5C48\u3002":"\u97E9\u5E08\u5085\u6CA1\u80FD\u8D70\u51FA\u5927\u7262\u3002",e.luRedeem?"\u9646\u4E03\u56DE\u4E86\u5934\u3002":"\u9646\u4E03\u8D70\u9519\u4E86\u4E00\u6B65\u3002",e.gift&&($e.aff.ayuan||0)>=3?"\u957F\u5E9A\u548C\u5B81\u6C85\uFF0C\u540E\u6765\u5728\u626C\u5DDE\u91CD\u9022\u3002":"\u957F\u5E9A\u548C\u5B81\u6C85\uFF0C\u6B64\u540E\u518D\u6CA1\u76F8\u89C1\u3002",e.path==="south"?"\u5B88\u62D9\u62A4\u7740\u6D4B\u7A3F\u968F\u5B66\u793E\u5357\u4E0B\u3002":"\u5B88\u62D9\u7559\u5728\u5317\u5E73\uFF0C\u5728\u654C\u4EBA\u773C\u76AE\u5E95\u4E0B\u85CF\u8D77\u6BBF\u6837\u3002",e.sunRedeem?"\u5B59\u4E16\u660C\u6700\u540E\u7AD9\u5728\u4E86\u987E\u5BB6\u8FD9\u8FB9\u3002":"\u5B59\u4E16\u660C\u6CA1\u6709\u56DE\u5934\u3002",e.evidence?"\u4F60\u62CD\u4E0B\u4E86\u5077\u6362\u6784\u4EF6\u7684\u8BC1\u636E\u3002":"\u5077\u6362\u6784\u4EF6\u7684\u4EBA\uFF0C\u8FD9\u4E00\u6B21\u6CA1\u6709\u88AB\u6293\u4F4F\u3002"];Ni(async()=>{hi("ending"),Fe(".end-box").innerHTML=`<h2>${t.title}</h2><p style="animation-delay:.9s">${t.desc}</p>
      ${n.map((s,r)=>`<p class="route" style="animation-delay:${1.6+r*.45}s">${s}</p>`).join("")}
      <p style="animation-delay:4.6s">\u5DF2\u89E3\u9501\u7ED3\u5C40 ${$e.endings.length}/3 \xB7 \u56FE\u8C31 ${$e.cards.length}/20</p>
      <button style="animation-delay:5.2s" id="end-codex">\u6253\u5F00\u56FE\u8C31</button> <button style="animation-delay:5.4s" id="end-title">\u56DE\u5230\u6807\u9898</button>`,Fe("#end-codex").onclick=jh,Fe("#end-title").onclick=()=>{$e.node=null,Tn(),Ni(async()=>Wl())}}),Qe.music(i==="bad"?"tense":"drama")}var Jh=null;function jh(){if(jt.dataset.mode==="puzzle"&&!(re&&re.free)){es("\u5148\u628A\u8FD9\u4E00\u5173\u62C6\u5B8C\uFF0C\u518D\u7FFB\u56FE\u8C31\u3002","tip");return}let i=Fe("#codex-list");i.innerHTML="",Wh.forEach((e,t)=>{let n=Vl[e],s=$e.cards.includes(e),r=document.createElement("button");r.className="cx",r.disabled=!s,r.innerHTML=`<div class="n">\u7B2C${Zf(t+1)}\u5173 \xB7 \u7B2C${"\u4E00\u4E8C\u4E09"[n.era-1]}\u5377</div><div class="t">${s?n.title+(n.arch?"<em>\u53E4\u5EFA\u7B51</em>":""):"\u672A\u89E3\u9501"}</div>
      <div class="d">${s?n.text[0].slice(0,40)+"\u2026\u2026":"\u5728\u6545\u4E8B\u91CC\u62C6\u5F00\u5B83\uFF0C\u624D\u4F1A\u6536\u8FDB\u56FE\u8C31\u3002\u6709\u4E9B\u5173\u5361\u53EA\u5728\u7279\u5B9A\u7684\u5206\u652F\u91CC\u51FA\u73B0\u3002"}</div>`,r.onclick=()=>{Fe("#codex").hidden=!0,op(e)},i.appendChild(r)}),Fe("#codex").hidden=!1}function op(i){re&&re.free||(Jh=jt.dataset.mode),ap({puzzle:i,id:""},cn,!0)}function zv(){let i=Jh;Jh=null,Ni(async()=>{i==="vn"?hi("vn"):i==="ending"?hi("ending"):Wl()})}function kv(){let i={\u987A\u7740\u5B83\u7684\u8DEF\u8D70:"\u6C38\u4E50\u5341\u4E94\u5E74 \xB7 \u97E9\u5E08\u5085","\u5148\u9000\u5C0F\u7684\uFF0C\u518D\u9000\u5927\u7684":"\u6C38\u4E50\u5341\u4E94\u5E74 \xB7 \u97E9\u5E08\u5085","\u6709\u65F6\u5019\uFF0C\u5F97\u5148\u8BA9\u4E00\u6B65":"\u6C11\u56FD\u4E8C\u5341\u516D\u5E74 \xB7 \u5708\u6905\u6276\u624B\u91CC"};Fe("#journal-body").innerHTML='<div class="jl">\u300C\u6BBF\u6837\u5728\uFF0C\u6BBF\u5C31\u5728\u3002\u300D<small>\u624B\u8BB0\u7B2C\u4E00\u9875 \xB7 \u987E\u957F\u5E9A</small></div>'+$e.clues.map(e=>`<div class="jl">\u300C${e}\u300D<small>${i[e]||""}</small></div>`).join(""),Fe("#journal").hidden=!1}function Vv(){let i=$e.met.filter(e=>oi[e]&&oi[e].aff);Fe("#people-list").innerHTML=i.length?i.map(e=>{let t=Math.max(0,Math.min(5,$e.aff[e]||0));return`<div class="pp"><img src="${__A(`art/ch/${e}${t>=3?"_smile":""}.webp`)}" alt=""><div><div class="nm">${oi[e].name}</div>
      <div class="hearts">${"\u2665".repeat(t)}${"\u2661".repeat(5-t)}</div><div class="bio">${oi[e].bio}</div></div></div>`}).join(""):'<p class="modal-sub">\u8FD8\u6CA1\u6709\u9047\u5230\u4EC0\u4E48\u4EBA\u3002</p>',Fe("#people").hidden=!1}document.querySelectorAll(".modal").forEach(i=>{i.addEventListener("click",e=>{(e.target===i||e.target.closest(".close"))&&(i.hidden=!0)})});Fe("#btn-codex").onclick=jh;Fe("#btn-journal").onclick=kv;Fe("#btn-people").onclick=Vv;Fe("#btn-sound").onclick=i=>{Qe.setMuted(!Qe.muted),i.currentTarget.classList.toggle("off",Qe.muted)};Wl();window.__game={goto:ci,get state(){return $e},E:Gh,titleScreen:Wl,openFree:op};
/*! Bundled license information:

three/three.core.js:
  (**
   * @license
   * Copyright 2010-2026 Three.js Authors
   * SPDX-License-Identifier: MIT
   *)

three/three.module.js:
  (**
   * @license
   * Copyright 2010-2026 Three.js Authors
   * SPDX-License-Identifier: MIT
   *)
*/

