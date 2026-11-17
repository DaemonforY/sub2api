#!/usr/bin/env python3
"""Rebuild frontend/public/games-play/rusted-cannon from the upstream repo.

Upstream: https://github.com/CyrilDieumegard/rusted-cannon (code and art MIT, music CC0).
Usage:   python3 tools/rusted_cannon_build.py /path/to/rusted-cannon-checkout

What it does, so the game loads over slow links (mainland visitors often get 5–15 KB/s):
- splits the 18 MB base64 atlas into WebP files: HQ art at half its 6× density, duplicates shared,
  and writes assets/atlas.js as key -> URL plus a load priority (low-res sprites, then stage 1, 2, 3);
- re-encodes the three soundtracks to Opus (Ogg) with an AAC (.m4a) fallback for Safari;
- copies the main script to game.js with small patches (priority loader, Chinese text wrapping,
  no service worker, 「离开」 returns to /games).
Hand-written files in that folder (index.html, boot.js, i18n.js, track.js) are left alone.
Needs: Pillow, cwebp, ffmpeg.
"""
import base64
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import sys

from PIL import Image

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, 'frontend', 'public', 'games-play', 'rusted-cannon')


def stage_of(key: str) -> int:
    """0 low-res, 1 common + first stage, 2 second, 3 third."""
    if not key.startswith('hq_'):
        return 0
    k = key[3:]
    if re.search(r'(_|boss)1(_|$)', k) or k in ('dock_span', 'decor_palm', 'decor_bush'):
        return 2
    if re.search(r'(_|boss)2(_|$)', k) or k in ('boss_defeat',):
        return 3
    return 1


def build_images(src: str) -> None:
    atlas = open(os.path.join(src, 'assets', 'atlas.js'), encoding='utf-8').read()
    items = re.findall(r'"([A-Za-z0-9_\-]+)":\s*"data:image/png;base64,([A-Za-z0-9+/=]+)"', atlas)
    img_dir = os.path.join(OUT, 'assets', 'img')
    shutil.rmtree(img_dir, ignore_errors=True)
    os.makedirs(img_dir)
    urls: dict[str, str] = {}
    prio: dict[str, int] = {}
    by_hash: dict[str, str] = {}
    total = 0
    for key, b64 in items:
        raw = base64.b64decode(b64)
        h = hashlib.sha1(raw).hexdigest()[:10]
        if h not in by_hash:
            im = Image.open(io.BytesIO(raw))
            if key.startswith('hq_'):
                im = im.resize((max(1, im.width // 2), max(1, im.height // 2)), Image.LANCZOS)
            tmp = os.path.join(img_dir, '_tmp.png')
            im.save(tmp)
            name = f'{key}.webp'
            subprocess.run(['cwebp', '-quiet', '-q', '80', '-alpha_q', '85', '-m', '6', tmp, '-o', os.path.join(img_dir, name)], check=True)
            os.remove(tmp)
            total += os.path.getsize(os.path.join(img_dir, name))
            by_hash[h] = name
        urls[key] = 'assets/img/' + by_hash[h]
        prio[key] = stage_of(key)
    with open(os.path.join(OUT, 'assets', 'atlas.js'), 'w', encoding='utf-8') as f:
        f.write('// Rusted Cannon sprites (MIT, Cyril Dieumegard), split out of the upstream base64 atlas by\n')
        f.write('// tools/rusted_cannon_build.py. HQ frames are stored at half density (3x); game.js knows.\n')
        f.write('window.SPRITE_ATLAS = ' + json.dumps(urls, indent=0) + ';\n')
        f.write('window.SPRITE_PRIORITY = ' + json.dumps(prio, separators=(',', ':')) + ';\n')
    print(f'images: {len(items)} keys, {len(by_hash)} files, {total // 1024} KB')


TRACKS = ['stage-1-superhero-cc0', 'stage-2-jungle-battle-cc0', 'stage-3-oldschool-action-cc0']


def build_audio(src: str) -> None:
    a_dir = os.path.join(OUT, 'assets', 'audio')
    shutil.rmtree(a_dir, ignore_errors=True)
    os.makedirs(a_dir)
    for t in TRACKS:
        inp = os.path.join(src, 'assets', 'audio', t + '.ogg')
        subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', inp, '-c:a', 'libopus', '-b:a', '32k', '-vbr', 'on', '-application', 'audio', os.path.join(a_dir, t + '.opus.ogg')], check=True)
        subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', inp, '-c:a', 'aac', '-b:a', '48k', '-ac', '1', '-movflags', '+faststart', os.path.join(a_dir, t + '.m4a')], check=True)
    sizes = {f: os.path.getsize(os.path.join(a_dir, f)) // 1024 for f in sorted(os.listdir(a_dir))}
    print('audio KB:', sizes)


def patch(s: str, old: str, new: str) -> str:
    if old not in s:
        raise SystemExit(f'patch target not found: {old[:80]!r}')
    return s.replace(old, new, 1)


def build_game(src: str) -> None:
    html = open(os.path.join(src, 'index.html'), encoding='utf-8').read()
    start = html.index('<script src="assets/atlas.js"></script>\n<script>') + len('<script src="assets/atlas.js"></script>\n<script>')
    end = html.rindex('</script>')
    js = html[start:end]

    # load sprites from files, a few at a time, in priority order; HQ files are half density
    js = patch(js, """  atlasKeys.forEach(k => {
    const im = new Image();
    // HQ frames carry six times the display pixels while retaining the exact
    // logical box of the approved gameplay sprite.
    if (k.startsWith('hq_')) im.__logicalScale = HQ_ASSET_SCALE;
    imgCache.set(k, im);
    let done=false; const once=()=>{ if(done) return; done=true; noteAsset(); };
    im.onload = once; im.onerror = once;
    try { im.src = atlas[k]; } catch (e) { once(); return; }
    if (im.complete && im.naturalWidth > 0) once();
  });""", """  // HiveGPT: sprites are separate WebP files (HQ ones at half density, so 3x), fetched four at a
  // time in priority order: low-res sprites, then stage 1, 2, 3. Until a file arrives the game
  // draws the sprite's low-res or procedural fallback, and a late arrival replaces it.
  const PRIO = (typeof window !== 'undefined' && window.SPRITE_PRIORITY) || {};
  const queue = atlasKeys.slice().sort((a, b) => (PRIO[a] || 0) - (PRIO[b] || 0));
  atlasKeys.forEach(k => {
    const im = new Image();
    if (k.startsWith('hq_')) im.__logicalScale = HQ_ASSET_SCALE / 2;
    im.decoding = 'async';
    imgCache.set(k, im);
  });
  let loading = 0;
  function pumpAssets() {
    while (loading < 4 && queue.length) {
      const k = queue.shift();
      const im = imgCache.get(k);
      loading++;
      let done=false; const once=()=>{ if(done) return; done=true; loading--; noteAsset(); pumpAssets(); };
      im.onload = once; im.onerror = once;
      try { im.src = atlas[k]; } catch (e) { once(); }
    }
  }
  pumpAssets();""")

    # Opus where the browser plays it, AAC elsewhere (Safari)
    js = patch(js, """  const TRACKS = [
    'assets/audio/stage-1-superhero-cc0.ogg',
    'assets/audio/stage-2-jungle-battle-cc0.ogg',
    'assets/audio/stage-3-oldschool-action-cc0.ogg'
  ];""", """  const OPUS = (() => { try { return !!new Audio().canPlayType('audio/ogg; codecs="opus"'); } catch (e) { return false; } })();
  const TRACKS = [
    'assets/audio/stage-1-superhero-cc0',
    'assets/audio/stage-2-jungle-battle-cc0',
    'assets/audio/stage-3-oldschool-action-cc0'
  ].map(t => t + (OPUS ? '.opus.ogg' : '.m4a'));""")

    # Chinese has no spaces: wrap it by width instead of by word
    js = patch(js, """function wrapText(text, maxChars) {
  const words = String(text || '').split(/\\s+/);""", """function wrapText(text, maxChars) {
  text = (typeof window !== 'undefined' && window.RC_T) ? window.RC_T(String(text || '')) : String(text || '');
  if (/[\\u3400-\\u9fff]/.test(text)) {
    const per = Math.max(4, Math.floor(maxChars * 0.56));
    const out = [];
    for (let i = 0; i < text.length; i += per) {
      let piece = text.slice(i, i + per);
      // keep closing punctuation on the line it belongs to
      while (i + per < text.length && /[，。！？、；：”）…]/.test(text[i + per])) { piece += text[i + per]; i++; }
      out.push(piece);
    }
    return out.length ? out : [''];
  }
  const words = text.split(/\\s+/);""")

    # menu words are baked into English artwork; in Chinese draw them as sharp text instead
    js = patch(js, """  function menuLabel(name='PLAY') {
    const normalized=String(name).toLowerCase();""", """  function menuLabel(name='PLAY') {
    const normalized=String(name).toLowerCase();
    if (typeof window !== 'undefined' && window.RC_LANG === 'zh') {
      const c = make('zh_menu_'+normalized, 512, 80, (g)=>{
        g.font='bold 52px "PingFang SC","Microsoft YaHei",sans-serif'; g.textAlign='center'; g.textBaseline='middle';
        g.lineJoin='round'; g.strokeStyle='#151a21'; g.lineWidth=14; g.strokeText(String(name).toUpperCase(),256,42);
        g.strokeStyle='#d59b38'; g.lineWidth=6; g.strokeText(String(name).toUpperCase(),256,42);
        g.fillStyle='#f3efe6'; g.fillText(String(name).toUpperCase(),256,42); g.textAlign='left';
      });
      c.__logicalScale = 4;
      return c;
    }""")

    # the 5x7 bitmap font has no Chinese: translated labels are drawn as regular text
    js = patch(js, """function drawPixelText(text,x,y,options={}) {
  const value=String(text).toUpperCase();""", """function drawPixelText(text,x,y,options={}) {
  if (typeof window !== 'undefined' && window.RC_LANG === 'zh' && window.RC_T) {
    const zh = window.RC_T(String(text));
    if (/[\\u3400-\\u9fff]/.test(zh)) {
      const sc = Math.max(1, Math.round(options.scale || 1));
      ctx.save();
      ctx.font = 'bold ' + (8 * sc) + 'px "PingFang SC","Microsoft YaHei",sans-serif';
      ctx.textBaseline = 'top';
      ctx.textAlign = options.align || 'left';
      const top = Math.round(y) - sc;
      if (options.shadow !== false) { ctx.fillStyle = options.shadowColor || 'rgba(0,0,0,0.78)'; ctx.fillText(zh, x + sc, top + sc); }
      ctx.fillStyle = options.color || '#f8fafc';
      ctx.fillText(zh, x, top);
      const width = ctx.measureText(zh).width;
      ctx.restore();
      const align = options.align || 'left';
      return { x: Math.round(align === 'center' ? x - width / 2 : align === 'right' ? x - width : x), y: top, width, height: 8 * sc };
    }
  }
  const value=String(text).toUpperCase();""")

    js = patch(js, "'RUSTED CANNON — Arcade Run & Gun';", "(window.RC_LANG === 'zh' ? '锈炮突击 Rusted Cannon · 像素横版射击 - HiveGPT 小游戏' : 'Rusted Cannon — Arcade Run & Gun · HiveGPT');")

    # 「离开」 goes back to the games page
    js = patch(js, """    if (!escaped) caption('CLOSE THIS TAB TO EXIT.', 'SYSTEM', 2.5);""", """    if (!escaped) { location.href = '/games'; escaped = true; }""")

    # no service worker: the site has its own caching, and a worker here could outlive the game
    js = re.sub(r"if \(typeof navigator!=='undefined' && 'serviceWorker' in navigator[^\n]*\n[^\n]*serviceWorker\.register[^\n]*\n\}", '// HiveGPT: service worker registration removed', js)
    if 'serviceWorker.register' in js:
        raise SystemExit('service worker registration still present')

    header = ('// Rusted Cannon — https://github.com/CyrilDieumegard/rusted-cannon\n'
              '// Copyright (c) 2026 Cyril Dieumegard, MIT License (see LICENSE in this folder).\n'
              '// Hosted on HiveGPT with small changes; built by tools/rusted_cannon_build.py.\n')
    with open(os.path.join(OUT, 'game.js'), 'w', encoding='utf-8') as f:
        f.write(header + js.strip() + '\n')
    print('game.js', os.path.getsize(os.path.join(OUT, 'game.js')) // 1024, 'KB')


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit(__doc__)
    src = sys.argv[1]
    os.makedirs(os.path.join(OUT, 'assets'), exist_ok=True)
    build_images(src)
    build_audio(src)
    build_game(src)
    shutil.copy(os.path.join(src, 'LICENSE'), os.path.join(OUT, 'LICENSE'))
    credits = open(os.path.join(src, 'AUDIO_CREDITS.md'), encoding='utf-8').read().rstrip()
    credits += ('\n\nOn HiveGPT the tracks are re-encoded to Opus 32 kbps (`.opus.ogg`) and AAC 48 kbps mono (`.m4a`)\n'
                'for slow connections; the music itself is unchanged.\n')
    with open(os.path.join(OUT, 'AUDIO_CREDITS.md'), 'w', encoding='utf-8') as f:
        f.write(credits)


if __name__ == '__main__':
    main()
