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

Motion: enter elements in order, keep everything readable for most of the scene, add one or two layers of
subtle secondary motion (drift, pulse, shimmer) so it never looks frozen, and settle the scene before it ends
(the player cross-fades 0.5 s into the next scene).`

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
		"language": "zh-CN",
		"scenes":   []map[string]any{{"id": "s1", "title": "开场：一个看似矛盾的公设", "narration": "旁白原文，口语化，一句一句……", "visual": "画面设计：要出现什么、怎么动、关键字幕、图表数据"}},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are the director and scriptwriter of short narrated explainer videos rendered as HTML animation.\n")
	fmt.Fprintf(&b, "Write the script for the user's request: %d–%d scenes, about %d seconds in total.\n", minScenes, maxScenes, seconds)
	_, _ = b.WriteString("- narration: what the narrator says in this scene, spoken language, in the user's language. No stage directions inside narration.\n")
	fmt.Fprintf(&b, "- Length budget: the narration of all scenes together is at most %d Chinese characters (about %d English words) — the narrator reads ~4.5 characters per second, so a longer script makes a longer video than the user asked for. Count before answering.\n", seconds*9/2, seconds*5/2)
	_, _ = b.WriteString("- visual: a concrete animation plan a motion designer can build without guessing — layout, the elements and diagrams, what moves when (tied to phrases of the narration), exact numbers / labels / data shown on screen. Visuals must be makeable with HTML, SVG and canvas (diagrams, charts, icons, typography, simple vector characters); no photos or video footage.\n")
	_, _ = b.WriteString("- The first scene hooks the viewer with a question or surprising fact; the last scene sums up in one memorable line.\n")
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
	_, _ = b.WriteString("Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema))
	return []LearnMessage{{Role: "system", Content: b.String()}, {Role: "user", Content: p.Prompt}}
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
}

func videoScenePrompt(in videoSceneBrief, previous, instruction, failure string) []LearnMessage {
	var b strings.Builder
	_, _ = b.WriteString("You are a senior motion designer who writes hand-crafted HTML/SVG/canvas animation code for a video player.\n\n")
	_, _ = b.WriteString(videoRuntimeDoc)
	_, _ = b.WriteString("\n\n## This video\n")
	fmt.Fprintf(&b, "Stage: %d × %d px. Theme: %s\n", in.Spec.Width, in.Spec.Height, videoJSONSchemaHint(in.Spec.Theme))
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
	}
	_, _ = b.WriteString("\nAnswer with the function body only, inside one ```js code block. No explanations.")

	var u strings.Builder
	if in.Motion {
		_, _ = u.WriteString("Request: " + in.Prompt + "\n")
	} else {
		fmt.Fprintf(&u, "Scene %s — %s\nDuration: %.2f s\nNarration: %s\nVisual plan: %s\n", in.Scene.ID, in.Scene.Title, in.Scene.Duration, in.Scene.Narration, in.Scene.Visual)
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
	msgs := []LearnMessage{{Role: "system", Content: b.String()}, {Role: "user", Content: u.String()}}
	if previous != "" {
		msgs = append(msgs, LearnMessage{Role: "assistant", Content: "```js\n" + previous + "\n```"})
		switch {
		case failure != "":
			msgs = append(msgs, LearnMessage{Role: "user", Content: "Running this code failed:\n" + failure + "\nFix the cause and return the complete corrected function body."})
		case instruction != "":
			msgs = append(msgs, LearnMessage{Role: "user", Content: "Change this scene: " + instruction + "\nKeep everything else. Return the complete updated function body."})
		}
	}
	return msgs
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
}

func videoRevisionPrompt(p *VideoProject, message, focusScene string) []LearnMessage {
	var outline strings.Builder
	for _, sc := range p.Spec.Scenes {
		fmt.Fprintf(&outline, "- %s 「%s」 (%.1fs)\n  narration: %s\n  visual: %s\n", sc.ID, sc.Title, sc.Duration, sc.Narration, sc.Visual)
	}
	schema := map[string]any{"reply": "好的，我把第 2 个分镜的背景改成深蓝色，并放慢了图表的动画。", "scenes": []map[string]any{{"id": "s2", "instruction": "背景改为深蓝色 #0b1f3a；柱状图上升放慢到 1.5 秒", "narration": "（只有要改旁白时才填，填完整的新旁白）"}}, "title": "（只有要改标题时才填）"}
	sys := "You are the director of an HTML video. The user asks for changes. Decide which scenes must change and write a precise instruction for the motion designer for each (what to change, keep the rest). Only include narration when the spoken words must change (it will be re-recorded). For changes that affect the whole video (colours, fonts, style) list every scene. Reply to the user in one or two sentences in their language. Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema)
	user := "Video: " + p.Spec.Title + "\nScenes:\n" + outline.String()
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
