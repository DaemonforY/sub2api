// 贴图 pictures are drawn at 2:3; the 公众号 feed shows 3:4 cards. Instead of cropping, the picture is
// scaled to the card height and centred on paper-coloured side margins (the colour comes from the
// picture's top edge), like a mounted painting. The cover gets a small 「画面由 AI 生成」 label.

const W = 1080;
const H = 1440;

function loadImage(blob) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob);
    const img = new Image();
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('图片无法读取'));
    };
    img.src = url;
  });
}

/** The paper colour of a picture (its top edge, a little darker): use the cover's for the whole set. */
export async function tietuPaperColor(blob) {
  return paperColor(await loadImage(blob));
}

function paperColor(img) {
  const c = document.createElement('canvas');
  c.width = 64;
  c.height = 4;
  const g = c.getContext('2d');
  g.drawImage(img, 0, 0, img.naturalWidth, Math.max(4, Math.round(img.naturalHeight * 0.02)), 0, 0, 64, 4);
  const d = g.getImageData(0, 0, 64, 4).data;
  let r = 0, gr = 0, b = 0;
  for (let i = 0; i < d.length; i += 4) {
    r += d[i];
    gr += d[i + 1];
    b += d[i + 2];
  }
  const n = d.length / 4;
  return [r / n, gr / n, b / n].map(v => Math.round(v * 0.97));
}

/**
 * → a 1080×1440 JPEG Blob. label: draw 「画面由 AI 生成」 at the bottom right. paper: [r, g, b] for the
 * margins (pass the cover's so a set looks alike); defaults to this picture's own.
 */
export async function frameTietu(blob, { label = false, paper = null } = {}) {
  const img = await loadImage(blob);
  const canvas = document.createElement('canvas');
  canvas.width = W;
  canvas.height = H;
  const g = canvas.getContext('2d');
  const [r, gr, b] = paper || paperColor(img);
  g.fillStyle = `rgb(${r},${gr},${b})`;
  g.fillRect(0, 0, W, H);
  // A faint paper grain on the margins.
  for (let i = 0; i < 2600; i++) {
    const x = Math.random() * W;
    const y = Math.random() * H;
    g.fillStyle = `rgba(0,0,0,${(Math.random() * 0.035).toFixed(3)})`;
    g.fillRect(x, y, 1 + Math.random() * 2, 1);
  }
  const scale = Math.min(W / img.naturalWidth, H / img.naturalHeight);
  const w = Math.round(img.naturalWidth * scale);
  const h = Math.round(img.naturalHeight * scale);
  g.imageSmoothingQuality = 'high';
  g.drawImage(img, Math.round((W - w) / 2), Math.round((H - h) / 2), w, h);
  if (label) {
    g.font = '22px "PingFang SC", "Microsoft YaHei", sans-serif';
    g.textAlign = 'right';
    g.textBaseline = 'alphabetic';
    const x = Math.round((W + w) / 2) - 14;
    g.fillStyle = 'rgba(0,0,0,0.55)';
    g.fillText('画面由 AI 生成', x + 1, H - 21);
    g.fillStyle = '#fff';
    g.fillText('画面由 AI 生成', x, H - 22);
  }
  return new Promise((resolve, reject) => {
    canvas.toBlob(out => (out ? resolve(out) : reject(new Error('图片导出失败'))), 'image/jpeg', 0.92);
  });
}
