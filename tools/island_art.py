#!/usr/bin/env python3
"""Turn the raw gpt-image-2 art for 椰风岛大冒险 into native-resolution pixel layers.

usage: python3 tools/island_art.py <raw-dir>   (raw-dir holds keyart.png, bg-*.png, mid-*.png)
Writes frontend/public/games-media/island/*.webp. Far layers are downscaled and palette-quantised so they
match the 400x224 game; mid layers are magenta-keyed with a hard alpha edge.
"""
import sys
from pathlib import Path
from PIL import Image

RAW = Path(sys.argv[1])
OUT = Path(__file__).resolve().parent.parent / 'frontend/public/games-media/island'
OUT.mkdir(parents=True, exist_ok=True)

def quant(im, n=64):
    return im.convert('RGB').quantize(colors=n, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE).convert('RGB')

# far layers: width 512, crop a 224-tall window (top offset per theme)
FAR = {'beach': 70, 'jungle': 40, 'volcano': 70}
for name, top in FAR.items():
    im = Image.open(RAW / f'bg-{name}.png').convert('RGB')
    w = 512
    h = round(im.height * w / im.width)
    im = im.resize((w, h), Image.LANCZOS).crop((0, top, w, top + 224))
    quant(im, 48).save(OUT / f'far-{name}.webp', lossless=True)

# mid layers: key out magenta, downscale premultiplied, hard alpha, crop to content
MID = {'beach': 400, 'jungle': 440, 'volcano': 430}
# atmospheric haze toward each theme's sky so the mid layer sits behind the play field
HAZE = {'beach': ((150, 205, 245), 0.28), 'jungle': ((120, 170, 160), 0.30), 'volcano': ((120, 50, 80), 0.22)}
for name, w in MID.items():
    im = Image.open(RAW / f'mid-{name}.png').convert('RGBA')
    px = im.load()
    for y in range(im.height):
        for x in range(im.width):
            r, g, b, a = px[x, y]
            if r > 170 and b > 170 and g < 110 and abs(r - b) < 90:
                px[x, y] = (0, 0, 0, 0)
    h = round(im.height * w / im.width)
    small = im.resize((w, h), Image.BOX)
    sp = small.load()
    rgb = Image.new('RGB', small.size)
    rp = rgb.load()
    mask = Image.new('L', small.size)
    mp = mask.load()
    for y in range(h):
        for x in range(w):
            r, g, b, a = sp[x, y]
            if a >= 140:
                (hr, hg, hb), k = HAZE[name]
                rp[x, y] = (round(r + (hr - r) * k), round(g + (hg - g) * k), round(b + (hb - b) * k))
                mp[x, y] = 255
    q = quant(rgb, 48).convert('RGBA')
    q.putalpha(mask)
    bbox = mask.getbbox()
    q = q.crop((0, bbox[1], w, h))
    q.save(OUT / f'mid-{name}.webp', lossless=True)
    print(name, q.size)

key = Image.open(RAW / 'keyart.png').convert('RGB')
key.resize((1536, 1024), Image.LANCZOS).save(OUT / 'keyart.webp', quality=84)
c = key.crop((0, 96, 1536, 96 + 864)).resize((768, 432), Image.LANCZOS)
c.save(OUT / 'cover.webp', quality=86)
print('done')
