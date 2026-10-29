package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prompts for the HiveGPT 视频 agent. The scene API below is implemented by the player runtime
// (video/public/player/runtime.js); keep the two in step.

const videoRuntimeDoc = `## Scene API (the player runtime)

Write the BODY of a JavaScript function that receives S and returns an update function:

    // your code here — build the scene once
    return function update(t) { /* t = seconds since this scene started, 0 … S.duration */ };

The player calls update(t) for every frame, in any order (seeking, export). update(t) must set the
complete state of the frame from t alone: no state carried between calls, no timers, no
requestAnimationFrame, no Date, no network, no external libraries (except S.three below). Build all elements once before
returning; inside update only change styles / attributes / text / canvas drawing.

S.root      the scene's <div> (S.width × S.height px, position:relative, overflow:hidden). Put everything in it.
S.width, S.height, S.duration (seconds), S.index (scene number from 0), S.total (number of scenes)
S.theme     { bg, surface, text, sub, accent, accent2, border, font, display, radius, dark } — use these colours / fonts.
S.h(tag, props?, ...children) → HTMLElement. props: { class, style: {…} or "css", text, html, any attribute }.
            Children may be elements or strings. Example: S.h('div', {class:'card', style:{left:'120px'}}, S.h('b', {text:'标题'}))
S.svg(markup) → the <svg> element parsed from a string (use for diagrams, shapes, paths).
S.css(text)  adds CSS. Scenes share one page, so prefix every class name with S.id, e.g. S.css('.' + S.id + '-card{padding:24px}').
S.id        this scene's id (a CSS-safe string).
S.icon(name, {size=48, color, stroke=2}) → SVG icon (Lucide names: brain, cpu, database, server, cloud, globe, rocket,
            zap, shield, lock, key, user, users, heart, star, check, x, alert-triangle, lightbulb, book-open,
            graduation-cap, flask-conical, atom, dna, chart-bar, chart-line, chart-pie, trending-up, clock, calendar,
            map-pin, car, plane, phone, smartphone, laptop, monitor, camera, music, mic, message-circle, mail,
            search, settings, code, terminal, git-branch, layers, box, package, gift, dollar-sign, shopping-cart,
            sun, moon, leaf, flame, droplet, mountain, home, building, flag, target, trophy, play, arrow-right,
            arrow-up, arrow-down, refresh-cw, sparkles, eye, hand, thumbs-up, wifi, battery).
S.canvas()  → { el, ctx } a 2D canvas covering the stage (already scaled; draw in S.width × S.height units).
            Clear and redraw it completely inside update(t).
S.three     the three.js library (r170; also S.three.RoomEnvironment, S.three.RoundedBoxGeometry) for real 3D.
            Make a renderer on your own canvas: const c = S.h('canvas'); S.root.appendChild(c);
            const r = new S.three.WebGLRenderer({ canvas: c, antialias: true }); r.setPixelRatio(1); r.setSize(S.width, S.height, false);
            c.style.cssText = 'position:absolute;inset:0;width:100%;height:100%'. Build meshes once; inside update(t) set
            positions / rotations / camera from t and call r.render(scene, camera) once. Use it only when 3D adds something.
S.p(t, start, dur)   progress 0…1 of a segment (clamped).
S.ease.linear / inQuad / outQuad / inOutQuad / outCubic / inOutCubic / outQuart / outExpo / outBack / outElastic / outBounce / inOutSine
S.lerp(a, b, k), S.clamp(x, lo, hi)
S.set(el, { x, y, scale, rotate, opacity, blur })   absolute transform from the element's layout position (px, deg).
S.tween(el, t, start, dur, from, to, ease = S.ease.outCubic)   interpolates those keys from → to over [start, start+dur]
            and applies them (before start → from, after → to).
S.draw(svgShape, k)   reveals an SVG path / line / circle / polyline stroke progressively (k 0…1).
S.type(el, text, k)   shows the first ⌈k·length⌉ characters of text (typewriter).
S.count(el, from, to, k, { decimals = 0, prefix = '', suffix = '', comma = true })   counting number.
S.random()  seeded random 0…1 (same sequence every time — call it only while building, never inside update).
S.word(i)   { start, end } seconds (scene time) when the i-th word of this scene's narration is spoken, or null.
S.at(text)  seconds (scene time) when the narration reaches the first occurrence of text, or null — sync visuals to speech.

Layout: position main elements with absolute px coordinates inside the S.width × S.height stage (or flex/grid
containers with explicit sizes). Keep a safe margin of 6% on every side; the bottom 14% is reserved for subtitles
when the video has narration — keep text out of it. Body text ≥ 34px, headings 64–120px on a 1920×1080 stage
(scale proportionally for other sizes). Never let text overflow its box: set widths and line-heights, and keep
lines short. Use S.theme.font for text and S.theme.display for big titles.
Fill the stage: the main visual (picture, diagram, chart, object) takes most of it. Avoid slide layouts — a small
title in a corner, a few small cards, half the stage empty; with few elements, make them large.
People: draw them only when the visual plan needs them, as clean silhouettes or hands with realistic proportions —
no blob bodies, round heads or dot eyes — unless the visual style is explicitly cartoon-like. A serious or
documentary tone needs restrained, realistic graphics, never toy-like figures.

Code size: keep the function body compact — aim for under 400 lines. Build repeated parts (characters' limbs,
rows of items, particles) with loops and small helper functions instead of copying them; keep SVG paths short.
When changing an existing scene, keep it about as long as it was.

Motion: enter elements in order, keep everything readable for most of the scene, add one or two layers of
subtle secondary motion (drift, pulse, shimmer) so it never looks frozen, and settle the scene before it ends
(the player cross-fades 0.5 s into the next scene).`

// videoImagesDoc documents S.image, only for videos with the user's pictures.
const videoImagesDoc = `
S.image(id, props?) → a new <img> of the user's picture id (img1, img2 …), already loaded. props like S.h
            (class, style …). It is position:absolute with object-fit:cover by default — set left/top/width/height
            (px) or style to place it. Call it once per use while building; animate it with S.set / S.tween
            (pan & zoom: scale 1 → 1.08 over the scene, x/y drift), reveal with opacity / clip-path, frame it
            (rounded corners, shadow, device mockup), point at details with labels / arrows / circles drawn on top.
S.images    { img1: { width, height }, … } natural sizes, to keep the aspect ratio (object-fit:contain shows all of it).
Never redraw a picture by hand or describe it with shapes: show the real image.
Show pictures big: a picture is the hero of its scene. When its shape differs from the stage (a tall poster on a
wide stage), crop and zoom into the part the narration is about (an overflow:hidden frame with a scaled image, or
object-position) instead of shrinking the whole picture into a small card; text inside a picture must end up
readable or be cropped out.
Pictures can already carry labels, captions, logos or watermarks (see the picture notes). Never draw new labels on
top of or overlapping the ones in the picture: point at the existing labels, or crop to a clean area and add your
own there. Do not crop so that a watermark or caption is cut in half at the stage edge.`

func videoJSONSchemaHint(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// --- clarifying questions -------------------------------------------------------------------

func videoQuestionsPrompt(p *VideoProject) []LearnMessage {
	schema := map[string]any{"questions": []map[string]any{{"id": "depth", "title": "讲解深度", "options": []map[string]any{{"label": "直观思维实验", "recommended": true}, {"label": "带公式推导"}, {"label": "科普入门"}}}}}
	return []LearnMessage{
		{Role: "system", Content: "You plan narrated explainer videos. Before writing the script, ask the user at most 3 short multiple-choice questions whose answers change the video the most (typical: depth / level, length, tone, audience, focus). Each question has 2–4 options; mark exactly one option as recommended — the best default for this request. Write questions and options in the user's language, each option under 12 characters. Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema)},
		{Role: "user", Content: p.Prompt},
	}
}

// --- script ----------------------------------------------------------------------------------

func videoScenePlan(length string) (int, int, int) {
	switch length {
	case "short":
		return 3, 4, 30
	case "long":
		return 8, 10, 150
	default:
		return 5, 7, 70
	}
}

func videoScriptPrompt(p *VideoProject) []LearnMessage {
	minScenes, maxScenes, seconds := videoScenePlan(p.Options.Length)
	style := findVideoStyle(p.Options.Style)
	genre, _ := findVideoGenre(p.Options.Genre)
	var answers []string
	for k, v := range p.Options.Answers {
		answers = append(answers, k+": "+v)
	}
	schema := map[string]any{
		"title": "狭义相对论：当光速重塑时空", "summary": "一句话说明这个视频讲什么", "audience": "大学生", "goal": "看完能说清楚时间膨胀从哪来",
		"language":  "zh-CN",
		"character": "",
		"facts":     []string{"光速 c ≈ 30 万千米/秒", "1905 年 爱因斯坦发表狭义相对论"},
		"scenes":    []map[string]any{{"id": "s1", "title": "开场：一个看似矛盾的公设", "narration": "旁白原文，口语化，一句一句……", "tone": "好奇地抛出问题，语速稍快", "visual": "画面设计：要出现什么、怎么动、关键字幕、图表数据"}},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are the director and scriptwriter of short narrated explainer videos rendered as HTML animation.\n")
	fmt.Fprintf(&b, "Write the script for the user's request: %d–%d scenes, about %d seconds in total.\n", minScenes, maxScenes, seconds)
	silent := videoSilent(p)
	if silent {
		_, _ = b.WriteString("- This video has NO narrator and no subtitles (the user will add their own voice-over or music). The message must come across from the picture alone: every scene shows its key text on screen — a short headline and at most two short lines, in the user's language.\n")
		fmt.Fprintf(&b, "- duration: seconds this scene stays on screen (%.1f–%.0f), long enough to read its text (about 6 Chinese characters per second plus 1.5 s) and to let the animation play. The durations add up to about %d seconds.\n", videoSilentSceneMin, videoSilentSceneMax, seconds)
	} else {
		_, _ = b.WriteString("- narration: what the narrator says in this scene, spoken language, in the user's language. No stage directions inside narration.\n")
		_, _ = b.WriteString("- tone: how the narrator should sound in this scene, one short phrase in Chinese (e.g. 「好奇地抛出问题」「沉稳地讲解」「兴奋地揭晓答案」「放慢语速，娓娓道来」). It guides the voice, it is not read aloud.\n")
		fmt.Fprintf(&b, "- Length budget: the narration of all scenes together is at most %d Chinese characters (about %d English words) — the narrator reads ~4.5 characters per second, so a longer script makes a longer video than the user asked for. Count before answering.\n", seconds*9/2, seconds*5/2)
	}
	images := p.Options.Images
	if len(images) == 0 {
		_, _ = b.WriteString("- visual: a concrete animation plan a motion designer can build without guessing — layout, the elements and diagrams, what moves when (tied to phrases of the narration), exact numbers / labels / data shown on screen. Visuals must be makeable with HTML, SVG and canvas (diagrams, charts, icons, typography, simple vector characters); no photos or video footage.\n")
	} else {
		ids := make([]string, len(images))
		for i, img := range images {
			ids[i] = fmt.Sprintf("%s (%d×%d)", img.ID, img.Width, img.Height)
		}
		fmt.Fprintf(&b, "- The user attached %d pictures, in this order: %s. They are the main material of the video: look at each one carefully, build the script around what they show (the product, the place, the people, the screenshot's content) and show every picture in at least one scene. Do not invent details the pictures and the request do not support.\n", len(images), strings.Join(ids, ", "))
		_, _ = b.WriteString("- images: the ids of the pictures a scene shows (usually one or two; [] when none).\n")
		_, _ = b.WriteString("- visual: a concrete animation plan a motion designer can build without guessing — which picture goes where and how it moves (slow pan / zoom, reveal, split screen, before / after), labels or arrows pointing at details in it (say where in the picture), plus text, icons and simple diagrams around it, tied to phrases of the narration. Plan pictures large: crop and zoom into the part being talked about (especially a tall picture on a wide stage) rather than showing it small; where a picture already has its own labels, use them or crop them away instead of stacking new labels on top. Besides the user's pictures, only HTML, SVG and canvas graphics; no other photos or footage.\n")
		_, _ = b.WriteString("- image_notes: one line per picture saying what is in it and where (e.g. 「白色保温杯居中，杯身左侧有 logo，背景浅灰」), including any text, labels, logos or watermarks already in the picture and where they are, and its shape (portrait / landscape) — the designer uses it to place labels and crops.\n")
	}
	_, _ = b.WriteString("- The first scene hooks the viewer with a question or surprising fact; the last scene sums up in one memorable line.\n")
	_, _ = b.WriteString("- character: when the video has a recurring character, mascot or host, describe its look once and concretely (body shape, colours, face, clothes or accessories) so every scene draws the same one; \"\" when there is none.\n")
	_, _ = b.WriteString("- facts: the facts, numbers, names and dates the video shows on screen, each written exactly as it must appear in every scene (e.g. 「订阅费 ¥1,450/月」). Take them from the request or well-established knowledge; never invent dates, prices or statistics — an illustrative number is written here once and reused unchanged. Every number in the visual plans must match this list.\n")
	_, _ = b.WriteString("- Be factually accurate. If the user gave a full script, keep their wording and only split it into scenes.\n")
	if genre.Brief != "" {
		_, _ = b.WriteString("Format: " + genre.Brief + "\n")
	}
	if style.Brief != "" {
		_, _ = b.WriteString("Visual style: " + style.Brief + "\n")
	}
	if len(answers) > 0 {
		_, _ = b.WriteString("The user's choices: " + strings.Join(answers, "; ") + "\n")
	}
	if silent {
		scene := map[string]any{"id": "s1", "title": "开场：一个看似矛盾的公设", "duration": 5, "visual": "画面设计，含屏幕上的原文文字：标题「……」，副标题「……」；元素怎么动"}
		if len(images) > 0 {
			scene["images"] = []string{"img1"}
		}
		schema["scenes"] = []map[string]any{scene}
		if len(images) > 0 {
			schema["image_notes"] = []map[string]any{{"id": "img1", "desc": "白色保温杯居中，杯身左侧有 logo，背景浅灰"}}
		}
	} else if len(images) > 0 {
		schema["scenes"] = []map[string]any{{"id": "s1", "title": "开场：这只杯子为什么能保温 12 小时", "narration": "旁白原文……", "tone": "好奇地抛出问题", "images": []string{"img1"}, "visual": "img1 居中缓慢放大，杯盖处出现标注……"}}
		schema["image_notes"] = []map[string]any{{"id": "img1", "desc": "白色保温杯居中，杯身左侧有 logo，背景浅灰"}}
	}
	_, _ = b.WriteString("Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema))
	user := p.Prompt
	if len(images) > 0 {
		user += fmt.Sprintf("\n\n（附 %d 张图片，按顺序是 %s）", len(images), videoImageIDs(images))
	}
	return []LearnMessage{{Role: "system", Content: b.String()}, {Role: "user", Content: user}}
}

// --- scene code ------------------------------------------------------------------------------

type videoSceneBrief struct {
	Index      int
	Scene      VideoScene
	Outline    []string // titles of all scenes, for continuity
	Spec       *VideoSpec
	StyleBrief string
	Category   string
	Motion     bool
	Prompt     string
	ImageData  []string // data URLs of the pictures this scene shows, for the model to look at
	Silent     bool     // a film without narration or subtitles
}

func videoImageIDs(images []VideoImage) string {
	ids := make([]string, len(images))
	for i, img := range images {
		ids[i] = img.ID
	}
	return strings.Join(ids, "、")
}

func videoScenePrompt(in videoSceneBrief, previous, instruction, failure string) []LearnMessage {
	var b strings.Builder
	_, _ = b.WriteString("You are a senior motion designer who writes hand-crafted HTML/SVG/canvas animation code for a video player.\n\n")
	_, _ = b.WriteString(videoRuntimeDoc)
	if len(in.Spec.Images) > 0 {
		_, _ = b.WriteString("\n" + videoImagesDoc)
	}
	if in.Spec.Common != "" && !in.Motion {
		_, _ = b.WriteString(videoKitUsage(in.Spec.Common))
	}
	_, _ = b.WriteString("\n\n## This video\n")
	fmt.Fprintf(&b, "Stage: %d × %d px. Theme: %s\n", in.Spec.Width, in.Spec.Height, videoJSONSchemaHint(in.Spec.Theme))
	_, _ = b.WriteString(videoSettingSheet(in.Spec))
	if in.StyleBrief != "" {
		_, _ = b.WriteString("Visual style: " + in.StyleBrief + "\n")
	}
	if in.Category != "" {
		_, _ = b.WriteString("Kind of animation: " + in.Category + "\n")
	}
	if in.Motion {
		fmt.Fprintf(&b, "This is a single looping animation of %.1f s without narration or subtitles (you may use the whole stage). The state at t = duration must equal the state at t = 0 so the loop is seamless.\n", in.Scene.Duration)
	} else {
		fmt.Fprintf(&b, "Video: %q — %s\nScenes: %s\n", in.Spec.Title, in.Spec.Summary, strings.Join(in.Outline, " / "))
		fmt.Fprintf(&b, "This is scene %d of %d. Keep the look consistent with the other scenes (same theme, same title position and sizes).\n", in.Index+1, len(in.Outline))
		if in.Silent {
			_, _ = b.WriteString("This video has no narration and no subtitles: the on-screen text carries the message. Show the exact text from the visual plan, large and readable, each line on screen for at least 2.5 s; the bottom of the stage is free (no subtitle area). Fill the scene's whole duration with motion.\n")
		}
	}
	if len(in.Spec.Images) > 0 {
		_, _ = b.WriteString("The user's pictures:\n")
		for _, img := range in.Spec.Images {
			fmt.Fprintf(&b, "- %s: %d×%d", img.ID, img.Width, img.Height)
			if img.Desc != "" {
				_, _ = b.WriteString(" — " + img.Desc)
			}
			_, _ = b.WriteString("\n")
		}
		if in.Motion {
			_, _ = b.WriteString("Build the animation around these pictures (they are attached).\n")
		} else if len(in.Scene.Images) > 0 {
			fmt.Fprintf(&b, "This scene shows %s (attached): use S.image('%s') etc., not drawings of them.\n", strings.Join(in.Scene.Images, ", "), in.Scene.Images[0])
		} else {
			_, _ = b.WriteString("This scene does not need a picture, but may show one if the visual plan asks for it.\n")
		}
	}
	_, _ = b.WriteString("\nAnswer with the function body only, inside one ```js code block. No explanations.")

	var u strings.Builder
	if in.Motion {
		_, _ = u.WriteString("Request: " + in.Prompt + "\n")
	} else {
		if in.Silent {
			fmt.Fprintf(&u, "Scene %s — %s\nDuration: %.2f s\nVisual plan: %s\n", in.Scene.ID, in.Scene.Title, in.Scene.Duration, in.Scene.Visual)
		} else {
			fmt.Fprintf(&u, "Scene %s — %s\nDuration: %.2f s\nNarration: %s\nVisual plan: %s\n", in.Scene.ID, in.Scene.Title, in.Scene.Duration, in.Scene.Narration, in.Scene.Visual)
		}
		if in.Scene.Audio != nil && len(in.Scene.Audio.Words) > 0 {
			_, _ = u.WriteString("Narration timing (word @ seconds): ")
			for i, w := range in.Scene.Audio.Words {
				if i > 0 {
					_, _ = u.WriteString(" ")
				}
				fmt.Fprintf(&u, "%s@%.1f", w.Text, w.Start)
			}
			_, _ = u.WriteString("\nTime the visuals to these words (use S.at('…') or the numbers above).\n")
		}
	}
	msgs := []LearnMessage{{Role: "system", Content: b.String()}, {Role: "user", Content: u.String(), Images: in.ImageData}}
	if previous != "" {
		msgs = append(msgs, LearnMessage{Role: "assistant", Content: "```js\n" + previous + "\n```"})
		switch {
		case failure != "":
			fix := "Running this code failed:\n" + failure + "\nFix the cause and return the complete corrected function body."
			if in.Spec.Common != "" && !in.Motion {
				fix += " If the failure comes from the shared kit, stop using the broken K member in this scene and build that part yourself (you cannot change the kit)."
			}
			msgs = append(msgs, LearnMessage{Role: "user", Content: fix})
		case instruction != "":
			msgs = append(msgs, LearnMessage{Role: "user", Content: "Change this scene: " + instruction + "\nKeep everything else. Return the complete updated function body."})
		}
	}
	return msgs
}

// --- shared kit --------------------------------------------------------------------------------

// videoKitDoc is the contract of spec.common: code the player runs before every scene's own code.
const videoKitDoc = `## What you write: the shared kit

Your code is prepended to EVERY scene's function body (it runs once per scene, with that scene's S, before
the scene's own code), so all scenes share one look. Scene designers then write each scene using your kit.

Write plain JavaScript statements (no function wrapper, no return):
1. Declare exactly one top-level name: var K = (function () { … return { … }; })(); Keep everything else
   inside that closure — scenes declare their own variables.
2. When it runs, build the shared backdrop and chrome inside S.root right away: the stage background (gradient,
   texture, grid, glow, whatever suits the style) and anything every scene shows in the same place (a series or
   episode label, the scene counter S.index + 1 / S.total, a logo mark). Scenes never rebuild these.
3. Return these members:
   K.frame(t)          animates the backdrop and chrome (drift, pulse); every scene calls it first in update(t)
   K.c                 palette { bg, surface, text, sub, accent, accent2, line, good, bad } as hex strings
   K.font, K.display, K.mono   font stacks (family names in single quotes)
   K.size              type scale in px for this stage { hero, title, kicker, body, small }
   K.area              content area in px { left, right, top, bottom, width, height }: inside the safe margins,
                       below the chrome, above the subtitle strip
   K.title(kicker, text)   adds the scene headline at the house position and returns { kicker, title } elements
                       (text may contain <em>…</em> for the accent colour; kicker may be ""); they start visible,
                       the scene animates them
   K.card(left, top, width, height)   adds a panel in the house style to S.root and returns it
   K.el(tag, css, html)   an absolutely positioned element with the kit's font and text colour (not yet added)
   K.character(size)   only when the setting sheet describes a recurring character: adds nothing, returns
                       { el, set(t, opts) } — an SVG of exactly that character, size × size px, with
                       opts { mood: 'smile' | 'worry' | 'surprised' | 'determined', look (px), talk, wave, still }
                       and idle motion (blink, gentle bob) computed from t. Omit it when there is no character.
   Add other helpers the scenes will repeat (badge, stamp, big number, progress bar) when the plans need them.
4. Begin with a comment block that lists every K member in one line each: scene designers read it first.

Keep it under 220 lines (the character a few dozen lines of SVG, not a detailed illustration), deterministic (no timers or Date; S.random only while building), class names prefixed
with S.id, and leave the stage mostly empty for the scenes: the backdrop must not compete with the content.`

func videoKitPrompt(spec *VideoSpec, styleBrief string, silent bool, previous, instruction string) []LearnMessage {
	var b strings.Builder
	_, _ = b.WriteString("You are the art director of a multi-scene HTML animation video. You write the code of its shared visual system.\n\n")
	_, _ = b.WriteString(videoRuntimeDoc)
	_, _ = b.WriteString("\n\n" + videoKitDoc)
	fmt.Fprintf(&b, "\n\n## This video\nStage: %d × %d px. Theme (starting point, refine it): %s\n", spec.Width, spec.Height, videoJSONSchemaHint(spec.Theme))
	if styleBrief != "" {
		_, _ = b.WriteString("Visual style: " + styleBrief + "\n")
	}
	if silent {
		_, _ = b.WriteString("No narration and no subtitles: K.area may reach the bottom safe margin.\n")
	} else {
		_, _ = b.WriteString("Narrated with subtitles: the bottom 14% of the stage is the subtitle strip, so K.area.bottom stays above it.\n")
	}
	_, _ = b.WriteString("\nAnswer with the kit code only, inside one ```js code block. No explanations.")

	var u strings.Builder
	fmt.Fprintf(&u, "Video: %q — %s\n", spec.Title, spec.Summary)
	_, _ = u.WriteString(videoSettingSheet(spec))
	_, _ = u.WriteString("Scenes:\n")
	for i, sc := range spec.Scenes {
		fmt.Fprintf(&u, "%d. %s — %s\n", i+1, sc.Title, sc.Visual)
	}
	msgs := []LearnMessage{{Role: "system", Content: b.String()}, {Role: "user", Content: u.String()}}
	if previous != "" {
		msgs = append(msgs, LearnMessage{Role: "assistant", Content: "```js\n" + previous + "\n```"})
		msgs = append(msgs, LearnMessage{Role: "user", Content: "Change the kit: " + instruction + "\nKeep every K member and its signature (the scenes already use them); keep everything else. Return the complete updated kit."})
	}
	return msgs
}

// videoSettingSheet is the script's character and facts, for the kit and every scene.
func videoSettingSheet(spec *VideoSpec) string {
	var b strings.Builder
	if spec.Character != "" {
		_, _ = b.WriteString("Recurring character (draw exactly this, the same in every scene): " + spec.Character + "\n")
	}
	if len(spec.Facts) > 0 {
		_, _ = b.WriteString("Facts shown on screen — write them exactly like this, never contradict them, never invent other dates or numbers:\n")
		for _, f := range spec.Facts {
			_, _ = b.WriteString("- " + f + "\n")
		}
	}
	return b.String()
}

// videoKitUsage tells a scene designer how to build on the shared kit.
func videoKitUsage(common string) string {
	return "\n\n## Shared kit (runs before your code in this scene)\n```js\n" + common + "\n```\n" +
		"Build on it: call K.frame(t) first in update(t); put the scene headline with K.title(…) when the plan has one; " +
		"use K.c colours, K.size, K.area and K.card / K.el and the other K helpers; draw the recurring character only " +
		"with K.character(…), never your own version; do not rebuild the background or the chrome; never declare a " +
		"top-level name K. A scene that ignores the kit breaks the video's look."
}

// --- revising from chat ----------------------------------------------------------------------

type videoRevisionPlan struct {
	Reply  string `json:"reply"`
	Scenes []struct {
		ID          string `json:"id"`
		Instruction string `json:"instruction"`
		Narration   string `json:"narration,omitempty"`
		Title       string `json:"title,omitempty"`
	} `json:"scenes"`
	Title string `json:"title,omitempty"`
	Kit   string `json:"kit,omitempty"` // a change to the shared kit (the whole look), for films that have one
}

func videoRevisionPrompt(p *VideoProject, message, focusScene string, earlier []string) []LearnMessage {
	var outline strings.Builder
	for _, sc := range p.Spec.Scenes {
		fmt.Fprintf(&outline, "- %s 「%s」 (%.1fs)\n  narration: %s\n  visual: %s\n", sc.ID, sc.Title, sc.Duration, sc.Narration, sc.Visual)
		if len(sc.Images) > 0 {
			fmt.Fprintf(&outline, "  pictures: %s\n", strings.Join(sc.Images, ", "))
		}
	}
	for _, img := range p.Spec.Images {
		fmt.Fprintf(&outline, "Picture %s: %s\n", img.ID, img.Desc)
	}
	schema := map[string]any{"reply": "好的，我把第 2 个分镜的背景改成深蓝色，并放慢了图表的动画。", "scenes": []map[string]any{{"id": "s2", "instruction": "背景改为深蓝色 #0b1f3a；柱状图上升放慢到 1.5 秒", "narration": "（只有要改旁白时才填，填完整的新旁白）"}}, "title": "（只有要改标题时才填）"}
	whole := "For changes that affect the whole video (colours, fonts, style) list every scene."
	if p.Spec.Common != "" {
		schema["kit"] = "（只有改整片的配色、字体、背景、页眉或角色造型时才填）"
		whole = "Changes to the whole look go in kit (see below)."
	}
	sys := "You are the director of an HTML video. The user asks for changes. Decide which scenes must change and write a precise instruction for the motion designer for each (what to change, keep the rest). Only include narration when the spoken words must change (it will be re-recorded). " + whole + " Reply to the user in one or two sentences in their language. Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema)
	if videoSilent(p) {
		sys += "\nThis video has no narration (on-screen text only): never fill narration; put text changes in the instruction."
	}
	if p.Spec.Common != "" {
		sys += "\nThis video has a shared kit that every scene runs first: the palette, fonts, background, header / counter and the recurring character's design. A change to any of those for the whole video goes in \"kit\" (one precise instruction, e.g. 「主色改成深蓝 #1e3a8a，背景去掉网格」) instead of listing every scene; list scenes only when their own content must change too. Leave kit empty otherwise."
	}
	sys += "\nIf the user says an earlier change did not take effect, or that the video still looks like the first version, do that earlier change again: list the scenes it concerns with a precise instruction. You cannot see the video, so never claim what the user is seeing and never answer with an empty scene list in that case."
	user := "Video: " + p.Spec.Title + "\nScenes:\n" + outline.String()
	if len(earlier) > 0 {
		user += "\nThe user's earlier messages in this conversation (oldest first):\n- " + strings.Join(earlier, "\n- ") + "\n"
	}
	if focusScene != "" {
		user += "\nThe user is looking at scene " + focusScene + ".\n"
	}
	user += "\nRequest: " + message
	return []LearnMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}}
}

// --- parsing answers -------------------------------------------------------------------------

// extractJSONObject returns the first top-level {...} in a model answer (fences and prose around it
// are ignored), or "".
func extractJSONObject(answer string) string {
	start := strings.IndexByte(answer, '{')
	if start < 0 {
		return ""
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(answer); i++ {
		c := answer[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inString:
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return answer[start : i+1]
			}
		}
	}
	return ""
}

// extractCodeBlock returns the content of the first ```js / ```javascript fence, or the whole answer
// when there is no fence.
func extractCodeBlock(answer string) string {
	for _, fence := range []string{"```javascript", "```js", "```jsx", "```"} {
		i := strings.Index(answer, fence)
		if i < 0 {
			continue
		}
		rest := answer[i+len(fence):]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 && nl < 40 {
			rest = rest[nl+1:]
		}
		if end := strings.Index(rest, "```"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(answer)
}
