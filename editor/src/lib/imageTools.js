// Canvas helpers for the 公众号 draft flow: base64 → Blob, "under 1 MB JPEG/PNG" for content
// images (media/uploadimg), and the 2.35:1 cover crop.

export const CONTENT_IMAGE_MAX_BYTES = 1000 * 1000; // 公众号正文图片要求小于 1MB，留一点余量
export const COVER_RATIO = 2.35;
export const COVER_MIN_WIDTH = 900;
export const COVER_MIN_HEIGHT = 383;
const COVER_MAX_WIDTH = 1880;

export function b64ToBlob(b64, mime = 'image/png') {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new Blob([bytes], { type: mime });
}

export function dataUrlToBlob(dataUrl) {
  const m = /^data:([^;,]+)?(;base64)?,(.*)$/s.exec(dataUrl);
  if (!m) throw new Error('无效的 data URL');
  const mime = m[1] || 'application/octet-stream';
  if (m[2]) return b64ToBlob(m[3], mime);
  return new Blob([decodeURIComponent(m[3])], { type: mime });
}

export function loadImageFromBlob(blob) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob);
    const img = new Image();
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('图片解码失败'));
    };
    img.src = url;
  });
}

/** Loads a remote image with CORS so it can be drawn to a canvas (img-src allows https:). */
export function loadImageCrossOrigin(src) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('图片无法读取（可能是跨域限制）'));
    img.src = src;
  });
}

export function canvasToBlob(canvas, type, quality) {
  return new Promise((resolve, reject) => {
    canvas.toBlob(b => (b ? resolve(b) : reject(new Error('图片导出失败'))), type, quality);
  });
}

function drawToCanvas(img, width, height, background) {
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  if (background) {
    ctx.fillStyle = background;
    ctx.fillRect(0, 0, width, height);
  }
  ctx.drawImage(img, 0, 0, width, height);
  return canvas;
}

/** Draws an <img> element into a PNG Blob (for remote images loaded via loadImageCrossOrigin). */
export async function imageElementToBlob(img) {
  const canvas = drawToCanvas(img, img.naturalWidth, img.naturalHeight, null);
  return canvasToBlob(canvas, 'image/png');
}

/**
 * Returns a JPEG or PNG Blob smaller than maxBytes (公众号 media/uploadimg only takes jpg/png < 1MB).
 * Small JPEG/PNG pass through untouched; anything else is re-encoded as JPEG, lowering the quality
 * and then the size until it fits.
 */
export async function compressUnder(blob, maxBytes = CONTENT_IMAGE_MAX_BYTES) {
  const okType = blob.type === 'image/jpeg' || blob.type === 'image/png';
  if (okType && blob.size < maxBytes) return blob;

  const img = await loadImageFromBlob(blob);
  let width = img.naturalWidth;
  let height = img.naturalHeight;
  const maxDim = 1920;
  const scale0 = Math.min(1, maxDim / Math.max(width, height));
  width = Math.max(1, Math.round(width * scale0));
  height = Math.max(1, Math.round(height * scale0));

  for (let round = 0; round < 8; round++) {
    const canvas = drawToCanvas(img, width, height, '#ffffff');
    for (const q of [0.9, 0.82, 0.74, 0.66, 0.58]) {
      const out = await canvasToBlob(canvas, 'image/jpeg', q);
      if (out.size < maxBytes) return out;
    }
    width = Math.max(1, Math.round(width * 0.8));
    height = Math.max(1, Math.round(height * 0.8));
  }
  throw new Error('图片压缩到 1MB 以内失败');
}

/**
 * Center-crops an image Blob to 2.35:1 (公众号封面比例) and returns a JPEG of at least 900×383.
 */
export async function cropToCover(blob) {
  const img = await loadImageFromBlob(blob);
  const sw0 = img.naturalWidth;
  const sh0 = img.naturalHeight;
  let sw = sw0;
  let sh = Math.round(sw0 / COVER_RATIO);
  if (sh > sh0) {
    sh = sh0;
    sw = Math.round(sh0 * COVER_RATIO);
  }
  const sx = Math.round((sw0 - sw) / 2);
  const sy = Math.round((sh0 - sh) / 2);

  const outW = Math.min(COVER_MAX_WIDTH, Math.max(COVER_MIN_WIDTH, sw));
  const outH = Math.max(COVER_MIN_HEIGHT, Math.round(outW / COVER_RATIO));

  const canvas = document.createElement('canvas');
  canvas.width = outW;
  canvas.height = outH;
  const ctx = canvas.getContext('2d');
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, outW, outH);
  ctx.imageSmoothingQuality = 'high';
  ctx.drawImage(img, sx, sy, sw, sh, 0, 0, outW, outH);
  return canvasToBlob(canvas, 'image/jpeg', 0.9);
}

export function extensionFor(blob) {
  if (blob.type === 'image/png') return 'png';
  if (blob.type === 'image/gif') return 'gif';
  return 'jpg';
}
