package service

// HiveGPT 视频: the choices offered on the create page — visual styles (each with the theme tokens the
// player and the model use), film genres, gallery categories and canvas ratios. Served to the app by
// GET /api/v1/video/catalog so both sides agree.

type VideoTheme struct {
	Background  string `json:"bg"`
	Surface     string `json:"surface"`
	Text        string `json:"text"`
	SubText     string `json:"sub"`
	Accent      string `json:"accent"`
	Accent2     string `json:"accent2"`
	Border      string `json:"border"`
	Font        string `json:"font"`
	DisplayFont string `json:"display"`
	Radius      int    `json:"radius"`
	Dark        bool   `json:"dark"`
}

type VideoStyle struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Genre    string     `json:"genre"`
	Summary  string     `json:"summary"`
	Theme    VideoTheme `json:"theme"`
	Brief    string     `json:"-"`
	Featured bool       `json:"featured,omitempty"`
}

type VideoGenre struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Brief   string `json:"-"`
}

type VideoCategory struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Mode    string `json:"mode"` // film / motion: which mode the create page opens
	Example string `json:"example"`
	Brief   string `json:"-"`
}

type VideoRatio struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Single quotes inside the stacks: the model interpolates them into double-quoted SVG attributes.
const (
	sansStack    = `'PingFang SC','Microsoft YaHei','Noto Sans SC',system-ui,sans-serif`
	displayStack = `'Smiley Sans','PingFang SC','Microsoft YaHei',system-ui,sans-serif`
	handStack    = `'LXGW WenKai','Kaiti SC','STKaiti','KaiTi',serif`
	monoStack    = `'JetBrains Mono','SF Mono',Menlo,Consolas,monospace`
)

var VideoGenres = []VideoGenre{
	{ID: "explain", Name: "知识讲解", Summary: "原理、架构、流程、数据与历史，旁白驱动", Brief: "A narrated explainer: each scene makes one idea visible (diagram, process, comparison, timeline, data), with short on-screen keywords that match the narration."},
	{ID: "lesson", Name: "课堂教学", Summary: "目标明确、步骤清楚，有例题和小结", Brief: "A lesson for students: learning goal first, then step-by-step build-up with worked examples, a quick check question and a recap scene."},
	{ID: "story", Name: "讲故事", Summary: "有角色、有情节：寓言、绘本、品牌故事", Brief: "A story with characters and a plot (setup, conflict, turn, ending). Characters are simple vector figures with expressive poses; scenes are illustrated settings."},
	{ID: "promo", Name: "产品宣传", Summary: "发布、亮点与促销，节奏快、冲击力强", Brief: "A punchy promo: bold kinetic typography, fast cuts between short scenes, product UI mock-ups and a clear call to action at the end."},
	{ID: "other", Name: "其他", Summary: "MG 动画、Vlog、活动开场等", Brief: "Whatever format fits the request best."},
}

var VideoStyles = []VideoStyle{
	{ID: "auto", Name: "AI 自定风格", Genre: "", Summary: "由 AI 按主题决定视觉风格", Featured: true,
		Theme: VideoTheme{Background: "#f7f5f0", Surface: "#ffffff", Text: "#1f2328", SubText: "#6b6f76", Accent: "#f97316", Accent2: "#2563eb", Border: "#e5e1d8", Font: sansStack, DisplayFont: displayStack, Radius: 18},
		Brief: "Choose the visual style that best suits the subject and keep it consistent across scenes."},
	{ID: "tech-blogger", Name: "科技博主", Genre: "explain", Summary: "深色背景、霓虹高亮、终端与图表", Featured: true,
		Theme: VideoTheme{Background: "#0b1020", Surface: "#121a33", Text: "#e8edf7", SubText: "#8b97b5", Accent: "#22d3ee", Accent2: "#a78bfa", Border: "#263255", Font: sansStack, DisplayFont: displayStack, Radius: 14, Dark: true},
		Brief: "Tech YouTuber look: near-black navy background with a faint grid, glowing cyan / violet highlights, monospace labels, code and terminal panels, crisp data charts."},
	{ID: "whiteboard", Name: "手绘白板", Genre: "explain", Summary: "白板 + 马克笔，边画边讲", Featured: true,
		Theme: VideoTheme{Background: "#fbfaf6", Surface: "#ffffff", Text: "#1c1c1c", SubText: "#5f5f5f", Accent: "#e4572e", Accent2: "#2e86ab", Border: "#d9d6cc", Font: handStack, DisplayFont: handStack, Radius: 10},
		Brief: "Whiteboard explainer: off-white board, black marker strokes drawn on progressively (stroke-dashoffset), red / blue marker accents, hand-written labels, slightly wobbly lines (SVG turbulence filter)."},
	{ID: "chalkboard", Name: "黑板粉笔", Genre: "lesson", Summary: "教室黑板，粉笔板书", Featured: true,
		Theme: VideoTheme{Background: "#1f3a2e", Surface: "#25463a", Text: "#f4f1e8", SubText: "#c9d3c3", Accent: "#ffd166", Accent2: "#ef8a8a", Border: "#3b5d4f", Font: handStack, DisplayFont: handStack, Radius: 8, Dark: true},
		Brief: "Chalkboard lesson: dark green board with chalk texture, chalk-white writing that appears stroke by stroke, yellow chalk for key points, formulas and diagrams like a teacher drawing live."},
	{ID: "ai-glow", Name: "AI 光效讲解", Genre: "explain", Summary: "暗场光效、粒子与发光节点", Featured: true,
		Theme: VideoTheme{Background: "#05060a", Surface: "#10121c", Text: "#f2f4ff", SubText: "#9aa3c7", Accent: "#7c5cff", Accent2: "#00e0b8", Border: "#232842", Font: sansStack, DisplayFont: displayStack, Radius: 16, Dark: true},
		Brief: "Dark cinematic AI look: black stage, glowing nodes and connections, particle fields drawn on a canvas, soft bloom (blur filters), light sweeps across headings."},
	{ID: "map-globe", Name: "地图地球", Genre: "explain", Summary: "地图、路线与地理数据", Featured: true,
		Theme: VideoTheme{Background: "#0e2433", Surface: "#143247", Text: "#eaf4fb", SubText: "#94b4c7", Accent: "#ffb703", Accent2: "#35c9a5", Border: "#22506b", Font: sansStack, DisplayFont: displayStack, Radius: 14, Dark: true},
		Brief: "Geography look: stylised maps (simplified shapes, no exact borders needed), animated routes with moving dots, location pins with labels, a rotating globe drawn with canvas or SVG meridians."},
	{ID: "clean-report", Name: "数据报告", Genre: "explain", Summary: "白底商务、清晰图表", Featured: true,
		Theme: VideoTheme{Background: "#f5f7fb", Surface: "#ffffff", Text: "#0f172a", SubText: "#64748b", Accent: "#2563eb", Accent2: "#10b981", Border: "#e2e8f0", Font: sansStack, DisplayFont: sansStack, Radius: 16},
		Brief: "Clean business report: light background, white cards with soft shadows, precise charts that grow in, big highlighted numbers, restrained motion."},
	{ID: "cartoon-class", Name: "卡通课堂", Genre: "lesson", Summary: "圆润卡通、明快配色", Featured: true,
		Theme: VideoTheme{Background: "#fff7e8", Surface: "#ffffff", Text: "#2b2b2b", SubText: "#6d6d6d", Accent: "#ff7a59", Accent2: "#3fa7ff", Border: "#f0dcc0", Font: displayStack, DisplayFont: displayStack, Radius: 24},
		Brief: "Friendly cartoon classroom: warm cream background, rounded chunky shapes with thick dark outlines, bouncy easing, a simple mascot character pointing at things."},
	{ID: "picture-book", Name: "绘本水彩", Genre: "story", Summary: "柔和水彩、温暖叙事", Featured: true,
		Theme: VideoTheme{Background: "#f6efe3", Surface: "#fffaf2", Text: "#3b302a", SubText: "#7d6b5f", Accent: "#d9825b", Accent2: "#6a9c89", Border: "#e7d8c3", Font: handStack, DisplayFont: handStack, Radius: 20},
		Brief: "Picture-book watercolour: paper texture, soft blotchy shapes (blur + turbulence filters), gentle pastel palette, characters with simple faces, slow drifting camera pans."},
	{ID: "ink", Name: "水墨国风", Genre: "story", Summary: "宣纸水墨、留白意境",
		Theme: VideoTheme{Background: "#f3eee3", Surface: "#faf7f0", Text: "#1d1a17", SubText: "#6f665c", Accent: "#b22d2d", Accent2: "#3f5d4a", Border: "#d8cfbf", Font: handStack, DisplayFont: handStack, Radius: 6},
		Brief: "Chinese ink painting: rice-paper background, black ink strokes of varying weight with soft bleeding edges, mountains and water, a red seal accent, generous empty space, vertical calligraphy titles."},
	{ID: "retro-film", Name: "复古胶片", Genre: "story", Summary: "颗粒、暖色、老照片质感",
		Theme: VideoTheme{Background: "#211a14", Surface: "#2e241b", Text: "#f3e6cf", SubText: "#bda98a", Accent: "#e0a64b", Accent2: "#c8553d", Border: "#4a3a2a", Font: sansStack, DisplayFont: displayStack, Radius: 6, Dark: true},
		Brief: "Retro film: warm sepia palette, film grain (canvas noise), vignette, slight flicker, photo-frame cards, typewriter captions, dates stamped like old documentaries."},
	{ID: "pixel", Name: "像素游戏", Genre: "story", Summary: "8-bit 像素、游戏化",
		Theme: VideoTheme{Background: "#1b1d3a", Surface: "#262a52", Text: "#f8f8f2", SubText: "#a9adc9", Accent: "#ffcc00", Accent2: "#ff4f79", Border: "#3d4277", Font: monoStack, DisplayFont: monoStack, Radius: 0, Dark: true},
		Brief: "8-bit game: everything on a pixel grid (shapes built from squares, image-rendering pixelated canvas), limited palette, steps() timing, HUD-style score and level labels."},
	{ID: "apple-minimal", Name: "极简发布会", Genre: "promo", Summary: "大留白、大标题、精致质感", Featured: true,
		Theme: VideoTheme{Background: "#000000", Surface: "#111111", Text: "#f5f5f7", SubText: "#a1a1a6", Accent: "#2997ff", Accent2: "#bf5af2", Border: "#2a2a2a", Font: sansStack, DisplayFont: sansStack, Radius: 22, Dark: true},
		Brief: "Keynote-style launch: black stage, huge crisp headlines that fade and slide in, product shots as clean device mock-ups, gradient text highlights, slow elegant motion."},
	{ID: "kinetic", Name: "大字快剪", Genre: "promo", Summary: "超大字、强对比、快节奏", Featured: true,
		Theme: VideoTheme{Background: "#f2f0ea", Surface: "#ffffff", Text: "#111111", SubText: "#555555", Accent: "#ff4a1c", Accent2: "#1c3cff", Border: "#111111", Font: displayStack, DisplayFont: displayStack, Radius: 0},
		Brief: "Kinetic typography promo: giant bold words filling the frame, hard cuts every 1-2 seconds, high contrast black / orange, words slamming in with overshoot, diagonal colour blocks."},
	{ID: "glass", Name: "毛玻璃 App", Genre: "promo", Summary: "渐变背景、玻璃卡片、App 界面",
		Theme: VideoTheme{Background: "#1e1b4b", Surface: "rgba(255,255,255,0.12)", Text: "#ffffff", SubText: "#c7c3f5", Accent: "#f472b6", Accent2: "#60a5fa", Border: "rgba(255,255,255,0.25)", Font: sansStack, DisplayFont: displayStack, Radius: 24, Dark: true},
		Brief: "Glassmorphism app promo: animated blurred gradient blobs behind frosted translucent cards, app UI screens with cursor taps, feature callouts that pop in."},
	{ID: "mg-flat", Name: "MG 扁平", Genre: "other", Summary: "扁平插画、图形转场",
		Theme: VideoTheme{Background: "#fdf6ec", Surface: "#ffffff", Text: "#22223b", SubText: "#6c6c80", Accent: "#ef476f", Accent2: "#118ab2", Border: "#ead8c0", Font: displayStack, DisplayFont: displayStack, Radius: 18},
		Brief: "Motion-graphics flat illustration: geometric shapes morph into icons and scenes, shape-wipe transitions, bright flat palette, lots of secondary motion."},
	{ID: "isometric", Name: "3D 等距", Genre: "other", Summary: "等距视角的立体场景",
		Theme: VideoTheme{Background: "#eef2ff", Surface: "#ffffff", Text: "#1e1b4b", SubText: "#5b5f8a", Accent: "#6366f1", Accent2: "#f59e0b", Border: "#c7d2fe", Font: sansStack, DisplayFont: displayStack, Radius: 12},
		Brief: "Isometric 3D: scenes built from isometric blocks (SVG polygons with three shaded faces), small buildings / servers / devices, objects rising into place, gentle camera drift."},
	{ID: "journal", Name: "手账 Vlog", Genre: "other", Summary: "纸张、贴纸、胶带拼贴",
		Theme: VideoTheme{Background: "#f8f1e7", Surface: "#fffdf8", Text: "#3a3029", SubText: "#8a7a6c", Accent: "#e76f51", Accent2: "#2a9d8f", Border: "#e3d3bf", Font: handStack, DisplayFont: handStack, Radius: 12},
		Brief: "Scrapbook vlog: paper cards tilted at small angles, washi-tape strips, stickers and doodles, handwritten notes, items dropping in with a bounce."},
}

var VideoCategories = []VideoCategory{
	{ID: "film", Name: "HTML 视频", Mode: "film", Summary: "带配音和字幕的多分镜讲解视频", Example: "用 60 秒讲清楚区块链是怎么防篡改的"},
	{ID: "science", Name: "理工科普", Mode: "film", Summary: "物理、数学、化学、生物的原理动画", Example: "光的折射：光线从空气射入水中发生偏折，标出入射角和折射角",
		Brief: "A science / maths explainer for students: one concept shown step by step with labelled parts and a one-line takeaway. Accuracy matters more than decoration."},
	{ID: "map", Name: "地图动画", Mode: "motion", Summary: "路线、迁徙、地理分布", Example: "丝绸之路：从长安出发，途经敦煌、撒马尔罕到罗马的路线动画",
		Brief: "A map animation: a stylised map, routes drawn progressively with a moving marker, pins and labels popping in, distances or dates counting up."},
	{ID: "data", Name: "数据可视化", Mode: "motion", Summary: "柱状、折线、饼图与数字", Example: "1–6 月用户数 120、180、260、410、530、800，折线逐点画出",
		Brief: "An animated data chart: axes and labels first, values grow in with easing, the key number is called out. Plot every value given; invent plausible round numbers only if none were given and say so."},
	{ID: "3d", Name: "3D 动画", Mode: "motion", Summary: "立体、透视与空间感", Example: "一个旋转的魔方，逐层转动后还原",
		Brief: "A 3D-looking animation using CSS 3D transforms (perspective, rotateX/Y, preserve-3d) or canvas projection math. No external libraries."},
	{ID: "hand-drawn", Name: "手绘动画", Mode: "motion", Summary: "像手画出来一样的线稿", Example: "一笔画出一只猫，最后眼睛眨一下",
		Brief: "A hand-drawn animation: strokes traced progressively with uneven weight and a slight wobble, then flat fills fade in."},
	{ID: "text", Name: "文字动画", Mode: "motion", Summary: "标题、口号与动态排版", Example: "「一个 Key，畅用主流 AI」逐字出现，「主流 AI」放大变色",
		Brief: "Kinetic typography: words enter one by one or letter by letter with emphasis on the key word. Large legible type that always fits the canvas."},
	{ID: "logo", Name: "Logo 动画", Mode: "motion", Summary: "品牌标志的出场动画", Example: "HiveGPT：六边形蜂巢逐格点亮，最后出现字标",
		Brief: "A logo reveal: the mark builds itself, the wordmark follows, then a hold. One idea, clean geometry, at most 3 colours, centred."},
	{ID: "product", Name: "产品演示", Mode: "motion", Summary: "App 界面、功能操作演示", Example: "一个待办 App：点击加号、输入任务、勾选完成的操作演示",
		Brief: "A product demo: a device or browser frame with an app UI; a cursor moves and clicks, screens change, feature labels call out what happens."},
	{ID: "flowchart", Name: "流程图", Mode: "motion", Summary: "步骤、分支与系统架构", Example: "HTTPS 握手：客户端问候 → 服务器证书 → 密钥交换 → 加密通信",
		Brief: "An animated flowchart: nodes appear in order, connectors draw between them, a highlight travels along the path. Short labels inside nodes."},
	{ID: "loader", Name: "加载动画", Mode: "motion", Summary: "Loading、进度与微交互", Example: "三个圆点依次跳动，像波浪一样",
		Brief: "A seamless loading / progress loop, small and centred, 1–2.4 s cycle."},
	{ID: "stick-figure", Name: "火柴人", Mode: "motion", Summary: "火柴人动作与小剧场", Example: "火柴人跑步、起跳、翻越栏杆的循环动画",
		Brief: "A stick-figure animation: figures built from lines and circles with joint angles interpolated over time (simple forward kinematics), expressive poses."},
	{ID: "line-drawing", Name: "线条描绘", Mode: "motion", Summary: "线稿逐笔描出", Example: "上海天际线线稿逐段描出，东方明珠最后亮起",
		Brief: "A line-drawing animation: paths traced in a deliberate order, then accents fade in and hold."},
	{ID: "game", Name: "网页小游戏", Mode: "motion", Summary: "可自动演示的小游戏画面", Example: "贪吃蛇自动吃到 5 个果子的演示",
		Brief: "An auto-playing mini game shown as a deterministic demo (the moves are scripted from time t, not from input)."},
	{ID: "other", Name: "其他", Mode: "motion", Summary: "任意创意动画", Example: "地球自转，周围卫星绕轨道飞行，夜晚一侧城市亮灯",
		Brief: "Make whatever animation best fits the request."},
}

var VideoRatios = []VideoRatio{
	{ID: "16:9", Width: 1920, Height: 1080},
	{ID: "9:16", Width: 1080, Height: 1920},
	{ID: "1:1", Width: 1080, Height: 1080},
	{ID: "4:3", Width: 1440, Height: 1080},
}

func findVideoStyle(id string) VideoStyle {
	for _, s := range VideoStyles {
		if s.ID == id {
			return s
		}
	}
	return VideoStyles[0]
}

func findVideoGenre(id string) (VideoGenre, bool) {
	for _, g := range VideoGenres {
		if g.ID == id {
			return g, true
		}
	}
	return VideoGenre{}, false
}

func findVideoCategory(id string) (VideoCategory, bool) {
	for _, c := range VideoCategories {
		if c.ID == id {
			return c, true
		}
	}
	return VideoCategory{}, false
}

func findVideoRatio(id string) VideoRatio {
	for _, r := range VideoRatios {
		if r.ID == id {
			return r
		}
	}
	return VideoRatios[0]
}
