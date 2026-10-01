"""Background removal and super-resolution for HiveGPT's image tools.

Internal service: only the sub2api gateway calls it (it authenticates and bills users).
Models (all permissive licenses, CPU via onnxruntime):
  - isnet-general-use (Apache-2.0) and u2net (Apache-2.0), as packaged by rembg (MIT)
  - Real-ESRGAN realesr-general-x4v3 (BSD-3-Clause), exported to ONNX at build time

POST /remove-bg?model=isnet|u2net   body: image bytes -> PNG with transparency
POST /upscale?scale=2|4             body: image bytes -> JPEG (PNG when the input has transparency)
GET  /health
One job runs at a time; the gateway limits how many wait.
"""

import io
import json
import os
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

import numpy as np
import onnxruntime as ort
from PIL import Image, ImageOps

MODELS = os.environ.get("MODEL_DIR", "/models")
THREADS = int(os.environ.get("THREADS", "3"))
MAX_BODY = 25 * 1024 * 1024
MAX_EDGE = 4096  # longest edge of any output
UPSCALE_INPUT_EDGE = 1024  # the model runs on at most this; ~10s on 3 cores
TILE, PAD = 256, 10
Image.MAX_IMAGE_PIXELS = 50_000_000


def session(name):
    opts = ort.SessionOptions()
    opts.intra_op_num_threads = THREADS
    opts.inter_op_num_threads = 1
    return ort.InferenceSession(os.path.join(MODELS, name), opts, providers=["CPUExecutionProvider"])


SEGMENT = {
    # model, input size, mean, std (same preprocessing as rembg)
    "isnet": (session("isnet-general-use.onnx"), 1024, (0.5, 0.5, 0.5), (1.0, 1.0, 1.0)),
    "u2net": (session("u2net.onnx"), 320, (0.485, 0.456, 0.406), (0.229, 0.224, 0.225)),
}
ESRGAN = session("realesr-general-x4v3.onnx")
LOCK = threading.Lock()


class BadInput(Exception):
    pass


def load(data):
    try:
        img = Image.open(io.BytesIO(data))
        img.load()
    except Image.DecompressionBombError:
        raise BadInput("图片像素太多（超过 5000 万像素），请先缩小后再试")
    except Exception:
        raise BadInput("无法读取这张图片，请换成 JPG、PNG 或 WebP 格式")
    return ImageOps.exif_transpose(img)


def fit(img, edge):
    scale = edge / max(img.size)
    return img.resize((max(1, round(img.width * scale)), max(1, round(img.height * scale))), Image.LANCZOS) if scale < 1 else img


def remove_bg(img, model):
    sess, size, mean, std = SEGMENT[model]
    img = fit(img.convert("RGB"), MAX_EDGE)
    x = np.asarray(img.resize((size, size), Image.LANCZOS), dtype=np.float32)
    x = x / max(float(x.max()), 1e-6)
    x = ((x - np.array(mean, np.float32)) / np.array(std, np.float32)).transpose(2, 0, 1)[None]
    pred = sess.run(None, {sess.get_inputs()[0].name: x})[0][0, 0]
    pred = (pred - pred.min()) / max(float(pred.max() - pred.min()), 1e-6)
    mask = Image.fromarray((pred * 255).astype(np.uint8), mode="L").resize(img.size, Image.LANCZOS)
    out = img.convert("RGBA")
    out.putalpha(mask)
    buf = io.BytesIO()
    out.save(buf, "PNG", compress_level=6)
    return buf.getvalue(), "image/png"


def esrgan_x4(rgb):
    x = np.asarray(rgb, dtype=np.float32).transpose(2, 0, 1)[None] / 255.0
    _, _, h, w = x.shape
    out = np.zeros((3, h * 4, w * 4), np.float32)
    name = ESRGAN.get_inputs()[0].name
    for y0 in range(0, h, TILE):
        for x0 in range(0, w, TILE):
            y1, x1 = min(y0 + TILE, h), min(x0 + TILE, w)
            py0, px0, py1, px1 = max(y0 - PAD, 0), max(x0 - PAD, 0), min(y1 + PAD, h), min(x1 + PAD, w)
            r = ESRGAN.run(None, {name: x[:, :, py0:py1, px0:px1]})[0][0]
            oy, ox = (y0 - py0) * 4, (x0 - px0) * 4
            out[:, y0 * 4 : y1 * 4, x0 * 4 : x1 * 4] = r[:, oy : oy + (y1 - y0) * 4, ox : ox + (x1 - x0) * 4]
    return Image.fromarray((np.clip(out.transpose(1, 2, 0), 0, 1) * 255).round().astype(np.uint8))


def upscale(img, scale):
    edge = max(img.size)
    target = min(edge * scale, MAX_EDGE)
    if target <= edge:
        raise BadInput(f"图片长边已经有 {edge}px，超分最大输出 {MAX_EDGE}px，不需要再放大")
    ratio = target / edge
    size = (max(1, round(img.width * ratio)), max(1, round(img.height * ratio)))
    alpha = img.getchannel("A") if img.mode in ("RGBA", "LA") or (img.mode == "P" and "transparency" in img.info) else None
    big = esrgan_x4(fit(img.convert("RGB"), UPSCALE_INPUT_EDGE))
    if big.size != size:
        big = big.resize(size, Image.LANCZOS)
    buf = io.BytesIO()
    if alpha is not None:
        big.putalpha(alpha.resize(size, Image.LANCZOS))
        big.save(buf, "PNG", compress_level=6)
        return buf.getvalue(), "image/png"
    big.save(buf, "JPEG", quality=94)
    return buf.getvalue(), "image/jpeg"


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def reply(self, status, body, ctype, extra=None):
        self.send_response(status)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        for k, v in (extra or {}).items():
            self.send_header(k, v)
        self.end_headers()
        self.wfile.write(body)

    def error(self, status, message):
        self.reply(status, json.dumps({"error": message}, ensure_ascii=False).encode(), "application/json; charset=utf-8")

    def do_GET(self):
        if self.path == "/health":
            return self.reply(200, b'{"ok":true}', "application/json")
        self.error(404, "not found")

    def do_POST(self):
        url = urlparse(self.path)
        query = {k: v[-1] for k, v in parse_qs(url.query).items()}
        length = int(self.headers.get("Content-Length") or 0)
        if length <= 0 or length > MAX_BODY:
            return self.error(413, "图片不能超过 25MB")
        data = self.rfile.read(length)
        try:
            if url.path == "/remove-bg":
                model = query.get("model", "isnet")
                if model not in SEGMENT:
                    return self.error(400, "unknown model")
                job = lambda img: remove_bg(img, model)
            elif url.path == "/upscale":
                scale = int(query.get("scale", "4"))
                if scale not in (2, 4):
                    return self.error(400, "scale must be 2 or 4")
                job = lambda img: upscale(img, scale)
            else:
                return self.error(404, "not found")
            img = load(data)
            with LOCK:
                started = time.time()
                body, ctype = job(img)
            ms = int((time.time() - started) * 1000)
            self.reply(200, body, ctype, {"X-Process-Ms": str(ms), "X-Image-Size": "%dx%d" % Image.open(io.BytesIO(body)).size})
        except BadInput as e:
            self.error(422, str(e))
        except Exception as e:  # noqa: BLE001
            self.log_error("job failed: %r", e)
            self.error(500, "图片处理失败，请稍后重试")

    def log_message(self, fmt, *args):
        print("%s %s" % (self.address_string(), fmt % args), flush=True)


if __name__ == "__main__":
    print("image tools ready", flush=True)
    ThreadingHTTPServer(("0.0.0.0", int(os.environ.get("PORT", "7000"))), Handler).serve_forever()
