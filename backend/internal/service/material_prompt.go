package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// A segment becomes a prompt in a fixed order (docs/MATERIAL_LIBRARY_PLAN.md §3):
//
//	frame → composition → scene + period → imagery by weight → light / weather → style → palette →
//	admin's extra → no-text rule
//
// Every part is kept apart in prompt_parts so a drawing can be redone with one part swapped. The
// blocklist runs over everything but the fixed no-text rule before a prompt is accepted.

const (
	materialMaxPromptRunes = 3800 // same ceiling as 课本诗画
	materialMaxExtraRunes  = 600
)

var ErrMaterialPromptBlocked = infraerrors.BadRequest("MATERIAL_PROMPT_BLOCKED", "提示词里有禁止出现的内容，请按提示修改（Prompt hits the blocklist）")

// materialNoText closes every prompt: the image model gets characters wrong, so words are set later.
const materialNoText = "Absolutely no text, letters, numbers, Chinese characters, calligraphy, seals, signatures, logos or watermarks anywhere in the image. People, if any, are anonymous figures, not real or famous people."

// materialStyleRule follows the style: imagery descriptions name colors (a crimson sun, ochre sand),
// and without it they win over an ink-wash style's black-and-grey palette.
const materialStyleRule = "Colors named in the elements above are hints only; this style and the palette decide the final colors."

var materialFrames = map[string]string{
	"1024x1536": "Tall vertical portrait composition (2:3), arranged from top to bottom.",
	"1536x1024": "Wide horizontal landscape composition (3:2).",
	"1024x1024": "Square composition (1:1).",
}

// MaterialStyleOption is a named prompt fragment an admin can pick (composition, palette or mood).
type MaterialStyleOption struct {
	Name     string `json:"name"`
	PromptEn string `json:"prompt_en"`
}

// MaterialStyleOptions is mat_styles.options; phase 1 keeps compositions, palettes and moods here
// instead of separate tables.
type MaterialStyleOptions struct {
	Compositions []MaterialStyleOption `json:"compositions,omitempty"`
	Palettes     []MaterialStyleOption `json:"palettes,omitempty"`
	Moods        []MaterialStyleOption `json:"moods,omitempty"`
}

// DefaultMaterialStyleOptions are offered with every style; a style's own options come first.
var DefaultMaterialStyleOptions = MaterialStyleOptions{
	Compositions: []MaterialStyleOption{
		{"远景小人", "Vast landscape with a small distant figure, the figure small against the scenery."},
		{"大留白", "Generous empty space; the subject occupies a third of the frame, the rest is calm sky, mist or water."},
		{"背影", "The figure is seen from behind, looking into the scene."},
		{"近景特写", "Close-up of the main object in the foreground, the background soft and simplified."},
		{"框景", "The scene is framed by a window, a moon gate or overhanging branches in the foreground."},
		{"俯瞰", "High viewpoint looking down over the whole scene, clear near, middle and far layers."},
	},
	Palettes: []MaterialStyleOption{
		{"青绿", "Blue-green landscape palette: mineral azurite and malachite on warm silk tones."},
		{"浅绛", "Light ochre and pale umber washes over ink, quiet and warm."},
		{"青花", "Cobalt blue and white with soft grey, nothing else."},
		{"宋画绢色", "Aged silk tones: muted beige, soft brown, faded green, low contrast."},
		{"墨色一点红", "Black and grey ink with a single small accent of vermilion red."},
	},
	Moods: []MaterialStyleOption{
		{"暖黄夕光", "Low golden evening light, long soft shadows, warm haze."},
		{"冷白月光", "Cool silver moonlight, deep blue night, long pale shadows."},
		{"薄雾清晨", "Early morning thin mist, soft diffused light, muted colors."},
		{"细雨", "Fine drizzle, wet surfaces and soft reflections, grey-green air."},
		{"雪后", "After snowfall, white ground and branches, crisp quiet air."},
		{"烈日长风", "Harsh bright daylight and strong wind, dust and moving grass."},
	},
}

func parseMaterialStyleOptions(raw json.RawMessage) MaterialStyleOptions {
	var o MaterialStyleOptions
	_ = json.Unmarshal(raw, &o)
	merge := func(own, def []MaterialStyleOption) []MaterialStyleOption {
		out := []MaterialStyleOption{}
		seen := map[string]bool{}
		for _, list := range [][]MaterialStyleOption{own, def} {
			for _, x := range list {
				x.Name, x.PromptEn = strings.TrimSpace(x.Name), strings.TrimSpace(x.PromptEn)
				if x.Name == "" || x.PromptEn == "" || seen[x.Name] {
					continue
				}
				seen[x.Name] = true
				out = append(out, x)
			}
		}
		return out
	}
	return MaterialStyleOptions{
		Compositions: merge(o.Compositions, DefaultMaterialStyleOptions.Compositions),
		Palettes:     merge(o.Palettes, DefaultMaterialStyleOptions.Palettes),
		Moods:        merge(o.Moods, DefaultMaterialStyleOptions.Moods),
	}
}

func materialPickOption(list []MaterialStyleOption, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", true
	}
	for _, x := range list {
		if x.Name == name {
			return x.PromptEn, true
		}
	}
	return "", false
}

// ---- recommendation ----

type MaterialStyleScore struct {
	Style   MaterialStyle        `json:"style"`
	Score   float64              `json:"score"`
	Reasons []string             `json:"reasons"`
	Options MaterialStyleOptions `json:"options"`
}

// materialImageryOf returns the imagery a segment's prompt uses: confirmed tags when there are any,
// otherwise every tag; hidden imagery is never used.
func materialImageryOf(links []MaterialSegmentImagery, byID map[int64]MaterialImagery) ([]MaterialSegmentImagery, bool) {
	confirmed := false
	for _, l := range links {
		if l.Confirmed {
			confirmed = true
			break
		}
	}
	out := []MaterialSegmentImagery{}
	for _, l := range links {
		im, ok := byID[l.ImageryID]
		if !ok || im.Status == MaterialStatusHidden || (confirmed && !l.Confirmed) {
			continue
		}
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Weight > out[j].Weight })
	return out, confirmed
}

// RecommendMaterialStyles scores the enabled styles for a segment's imagery: each imagery emotion a
// style suits counts by the imagery's weight, a matching category counts half, and a style the
// imagery itself recommends counts 3× its weight. Ties keep the styles' own order.
func RecommendMaterialStyles(styles []MaterialStyle, links []MaterialSegmentImagery, byID map[int64]MaterialImagery) []MaterialStyleScore {
	out := []MaterialStyleScore{}
	for _, st := range styles {
		if !st.Enabled {
			continue
		}
		emotions := map[string]bool{}
		for _, e := range st.SuitsEmotions {
			emotions[e] = true
		}
		categories := map[string]bool{}
		for _, c := range st.SuitsCategories {
			categories[c] = true
		}
		score := 0.0
		matched := map[string]bool{}
		var reasons []string
		for _, l := range links {
			im := byID[l.ImageryID]
			w := float64(l.Weight)
			for _, sid := range im.StyleIDs {
				if sid == st.ID {
					score += 3 * w
					reasons = append(reasons, "「"+im.Name+"」推荐此画风")
				}
			}
			for _, e := range im.Emotions {
				if emotions[e] {
					score += w
					if !matched[e] {
						matched[e] = true
						reasons = append(reasons, "情绪「"+e+"」")
					}
				}
			}
			if categories[im.Category] {
				score += w / 2
			}
		}
		out = append(out, MaterialStyleScore{Style: st, Score: score, Reasons: nonNilReasons(reasons), Options: parseMaterialStyleOptions(st.Options)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func nonNilReasons(r []string) []string {
	if r == nil {
		return []string{}
	}
	return r
}

// ---- assembly ----

// MaterialPromptInput is what an admin picks for one drawing.
type MaterialPromptInput struct {
	StyleID     int64   `json:"style_id"`
	Size        string  `json:"size"`        // 1024x1536 | 1536x1024 | 1024x1024; "" = the style's default
	Composition string  `json:"composition"` // option names; "" = none
	Palette     string  `json:"palette"`
	Mood        string  `json:"mood"`
	Extra       string  `json:"extra"`       // free text added before the no-text rule
	ImageryIDs  []int64 `json:"imagery_ids"` // nil = the segment's tags
}

// MaterialPromptParts are the pieces of a prompt, stored with each drawing.
type MaterialPromptParts struct {
	Frame       string   `json:"frame"`
	Composition string   `json:"composition,omitempty"`
	Scene       string   `json:"scene,omitempty"`
	Period      string   `json:"period,omitempty"`
	Imagery     []string `json:"imagery"`
	Mood        string   `json:"mood,omitempty"`
	Style       string   `json:"style"`
	Avoid       string   `json:"avoid,omitempty"`
	Palette     string   `json:"palette,omitempty"`
	Extra       string   `json:"extra,omitempty"`
	NoText      string   `json:"no_text"`
}

type MaterialPrompt struct {
	Prompt             string              `json:"prompt"`
	Parts              MaterialPromptParts `json:"parts"`
	Size               string              `json:"size"`
	Quality            string              `json:"quality"`
	StyleID            int64               `json:"style_id"`
	ImageryIDs         []int64             `json:"imagery_ids"`
	Blocked            bool                `json:"blocked"`
	Hits               []MaterialBlockHit  `json:"hits"`
	Notes              []string            `json:"notes"` // things the admin should know (unreviewed imagery, trimmed text…)
	TextOverlayAllowed bool                `json:"text_overlay_allowed"`
}

// materialImageryLine is how one imagery reads in a prompt.
func materialImageryLine(im MaterialImagery, weight int) string {
	visual := strings.TrimSpace(im.VisualEn)
	if visual == "" {
		visual = strings.TrimSpace(im.VisualZh)
	}
	if visual == "" {
		visual = im.Name
	}
	role := map[int]string{3: "Main subject", 2: "Supporting element", 1: "Small detail"}[weight]
	if role == "" {
		role = "Element"
	}
	return role + ": " + strings.TrimRight(visual, ". 。") + "."
}

var materialDynastyEn = map[string]string{
	"先秦": "pre-Qin", "周": "Zhou dynasty", "春秋": "Spring and Autumn period", "战国": "Warring States period",
	"秦": "Qin dynasty", "汉": "Han dynasty", "西汉": "Western Han dynasty", "东汉": "Eastern Han dynasty",
	"三国": "Three Kingdoms period", "魏": "Three Kingdoms period", "蜀": "Three Kingdoms period", "吴": "Three Kingdoms period",
	"晋": "Jin dynasty", "西晋": "Western Jin dynasty", "东晋": "Eastern Jin dynasty",
	"南北朝": "Northern and Southern dynasties", "南朝": "Southern dynasties", "北朝": "Northern dynasties",
	"北魏": "Northern Wei dynasty", "隋": "Sui dynasty", "唐": "Tang dynasty", "五代": "Five Dynasties period",
	"五代十国": "Five Dynasties period", "宋": "Song dynasty", "北宋": "Northern Song dynasty", "南宋": "Southern Song dynasty",
	"辽": "Liao dynasty", "金": "Jin (Jurchen) dynasty", "元": "Yuan dynasty", "明": "Ming dynasty", "清": "Qing dynasty",
}

// materialPeriod adds the setting for ancient works, so clothing and buildings match the dynasty.
func materialPeriod(dynasty string) string {
	d := strings.TrimSpace(dynasty)
	if !materialAncientDynasties[d] {
		return ""
	}
	era := materialDynastyEn[d]
	if era == "" {
		era = "ancient China (" + d + ")"
	}
	return fmt.Sprintf("Setting: China in the %s; clothing, hairstyles, buildings and objects belong to that period, nothing modern.", era)
}

// AssembleMaterialPrompt builds the prompt for a segment. It is pure: everything it needs is passed in.
func AssembleMaterialPrompt(seg *MaterialSegmentContext, links []MaterialSegmentImagery, byID map[int64]MaterialImagery,
	style MaterialStyle, in MaterialPromptInput, terms []MaterialBlockTerm) (*MaterialPrompt, error) {
	size := strings.TrimSpace(in.Size)
	if size == "" {
		size = style.DefaultSize
	}
	frame, ok := materialFrames[size]
	if !ok {
		return nil, ErrMaterialInvalid
	}
	opts := parseMaterialStyleOptions(style.Options)
	composition, ok1 := materialPickOption(opts.Compositions, in.Composition)
	palette, ok2 := materialPickOption(opts.Palettes, in.Palette)
	mood, ok3 := materialPickOption(opts.Moods, in.Mood)
	if !ok1 || !ok2 || !ok3 {
		return nil, ErrMaterialInvalid
	}
	quality := style.DefaultQuality
	if quality == "" {
		quality = shihuaDefaultQuality
	}

	out := &MaterialPrompt{Size: size, Quality: quality, StyleID: style.ID, Hits: []MaterialBlockHit{}, Notes: []string{},
		ImageryIDs: []int64{}, TextOverlayAllowed: MaterialTextOverlayAllowed(seg.LicenseStatus)}

	// Imagery: the admin's pick (in the segment's weights, weight 2 when not tagged) or the tags.
	use, confirmed := materialImageryOf(links, byID)
	if in.ImageryIDs != nil {
		weights := map[int64]int{}
		for _, l := range links {
			weights[l.ImageryID] = l.Weight
		}
		use = []MaterialSegmentImagery{}
		seen := map[int64]bool{}
		for _, id := range in.ImageryIDs {
			im, ok := byID[id]
			if !ok || seen[id] || im.Status == MaterialStatusHidden {
				continue
			}
			seen[id] = true
			w := weights[id]
			if w == 0 {
				w = 2
			}
			use = append(use, MaterialSegmentImagery{ImageryID: id, Weight: w})
		}
		sort.SliceStable(use, func(i, j int) bool { return use[i].Weight > use[j].Weight })
	} else if len(use) > 0 && !confirmed {
		out.Notes = append(out.Notes, "片段的意象还没人工确认，用的是 AI 标注")
	}
	var drafts []string
	parts := MaterialPromptParts{Frame: frame, Composition: composition, Imagery: []string{}, Mood: mood,
		Style: strings.TrimSpace(style.PromptEn), Avoid: strings.TrimSpace(style.NegativeEn), Palette: palette, NoText: materialNoText}
	for _, l := range use {
		im := byID[l.ImageryID]
		parts.Imagery = append(parts.Imagery, materialImageryLine(im, l.Weight))
		out.ImageryIDs = append(out.ImageryIDs, im.ID)
		if im.Status != MaterialStatusPublished {
			drafts = append(drafts, im.Name)
		}
	}
	if len(drafts) > 0 {
		out.Notes = append(out.Notes, "用到了还没审核发布的意象："+strings.Join(drafts, "、"))
	}

	scene := strings.TrimSpace(seg.SceneDesc)
	if scene != "" {
		parts.Scene = "Scene: " + scene
	}
	if scene == "" && len(parts.Imagery) == 0 {
		return nil, infraerrors.BadRequest("MATERIAL_PROMPT_EMPTY", "这个片段还没有画面描述，也没有意象，请先拆意象（Segment has no scene or imagery）")
	}
	parts.Period = materialPeriod(seg.Dynasty)
	if extra := strings.TrimSpace(in.Extra); extra != "" {
		if utf8.RuneCountInString(extra) > materialMaxExtraRunes {
			extra = materialTrimRunes(extra, materialMaxExtraRunes)
			out.Notes = append(out.Notes, fmt.Sprintf("补充要求超过 %d 字，已截断", materialMaxExtraRunes))
		}
		parts.Extra = extra
	}

	// Blocklist over everything the admin or the library wrote; the fixed rule is ours.
	body := materialJoinParts(parts, false)
	check := CheckMaterialBlocklist(body, "prompt", terms)
	out.Hits, out.Blocked = check.Hits, check.Blocked
	if check.Prompt != body {
		// strip terms were removed: rebuild the parts the same way so prompt_parts match the prompt.
		parts = materialStripParts(parts, terms)
		out.Notes = append(out.Notes, "已自动删掉禁忌词表里标为「删除」的词")
	}
	out.Parts = parts
	out.Prompt = materialJoinParts(parts, true)
	if utf8.RuneCountInString(out.Prompt) > materialMaxPromptRunes {
		return nil, infraerrors.BadRequest("MATERIAL_PROMPT_TOO_LONG", fmt.Sprintf("提示词超过 %d 字，请删减意象或补充要求（Prompt too long）", materialMaxPromptRunes))
	}
	return out, nil
}

func materialJoinParts(p MaterialPromptParts, withNoText bool) string {
	lines := []string{p.Frame}
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			lines = append(lines, s)
		}
	}
	add(p.Composition)
	add(p.Scene)
	add(p.Period)
	for _, s := range p.Imagery {
		add(s)
	}
	add(p.Mood)
	if p.Style != "" {
		add("Style: " + p.Style + " " + materialStyleRule)
	}
	if p.Avoid != "" {
		add("Avoid: " + p.Avoid)
	}
	if p.Palette != "" {
		add("Palette: " + p.Palette)
	}
	add(p.Extra)
	if withNoText {
		add(p.NoText)
	}
	return strings.Join(lines, "\n")
}

func materialStripParts(p MaterialPromptParts, terms []MaterialBlockTerm) MaterialPromptParts {
	strip := func(s string) string {
		var only []MaterialBlockTerm
		for _, t := range terms {
			if t.Action == MaterialBlockActionStrip {
				only = append(only, t)
			}
		}
		return CheckMaterialBlocklist(s, "prompt", only).Prompt
	}
	p.Composition, p.Scene, p.Mood, p.Style, p.Avoid, p.Palette, p.Extra =
		strip(p.Composition), strip(p.Scene), strip(p.Mood), strip(p.Style), strip(p.Avoid), strip(p.Palette), strip(p.Extra)
	for i := range p.Imagery {
		p.Imagery[i] = strip(p.Imagery[i])
	}
	return p
}

// ---- service ----

func (s *MaterialService) segmentPromptInputs(ctx context.Context, segmentID int64) (*MaterialSegmentContext, []MaterialSegmentImagery, map[int64]MaterialImagery, []MaterialStyle, error) {
	seg, links, err := s.SegmentImagery(ctx, segmentID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	all, err := s.repo.AllImagery(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	byID := make(map[int64]MaterialImagery, len(all))
	for _, im := range all {
		byID[im.ID] = im
	}
	styles, err := s.repo.ListStyles(ctx, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return seg, links, byID, styles, nil
}

// RecommendStyles ranks the enabled styles for a segment.
func (s *MaterialService) RecommendStyles(ctx context.Context, segmentID int64) ([]MaterialStyleScore, error) {
	_, links, byID, styles, err := s.segmentPromptInputs(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	use, _ := materialImageryOf(links, byID)
	return RecommendMaterialStyles(styles, use, byID), nil
}

// BuildPrompt assembles the prompt for a segment and a style. A blocked prompt is returned with
// Blocked set (so the page can show the hits); drawing refuses it.
func (s *MaterialService) BuildPrompt(ctx context.Context, segmentID int64, in MaterialPromptInput) (*MaterialPrompt, error) {
	seg, links, byID, styles, err := s.segmentPromptInputs(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	var style *MaterialStyle
	for i := range styles {
		if styles[i].ID == in.StyleID {
			style = &styles[i]
		}
	}
	if style == nil {
		return nil, infraerrors.BadRequest("MATERIAL_STYLE", "没有这个画风，或它已停用（Style not found or disabled）")
	}
	terms, err := s.repo.ListBlocklist(ctx, true)
	if err != nil {
		return nil, err
	}
	return AssembleMaterialPrompt(seg, links, byID, *style, in, terms)
}
