package service

// 我的风格, 先审镜头剧本 and 复盘: the series workflow of HiveGPT 视频.
//
// The ideas follow 一只桌子's Table-Video SOP (https://github.com/duoduoler-ops/Table-VideoSOP,
// CC BY 4.0), rewritten for this site: an account keeps one look across its films (a style card in
// place of DESIGN.md), the shot script with one 看点 per scene is reviewed before anything is made
// (text is cheaper to change than pictures, pictures cheaper than animation), and after a film a
// retrospective turns the user's rework into a few house rules the next films follow.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	videoStylesPerUser   = 20
	videoStyleNameMax    = 20
	videoStyleRulesMax   = 12
	videoStyleRuleMax    = 120
	videoRetroMaxTokens  = 3000
	videoRetroTranscript = 6000 // runes of the film's conversation shown to the retrospective
)

var (
	ErrVideoStyleNotFound = infraerrors.NotFound("VIDEO_STYLE_NOT_FOUND", "找不到这个风格卡，可能已被删除（Style not found）")
	ErrVideoStyleFull     = infraerrors.BadRequest("VIDEO_STYLE_FULL", "最多保存 20 个风格卡，请先删掉不用的（At most 20 styles）")
	ErrVideoStyleNoKit    = infraerrors.BadRequest("VIDEO_STYLE_NO_KIT", "这个作品没有整片统一的画面风格（较早的作品或单个动画），不能存为风格卡；用它新做一个多分镜视频再存（This film has no shared look to save）")
	ErrVideoStyleOff      = infraerrors.ServiceUnavailable("VIDEO_STYLE_OFF", "风格卡暂不可用（Styles are not available）")
)

// VideoStyleCard is a saved series look (我的风格).
type VideoStyleCard struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"-"`
	Name          string     `json:"name"`
	Ratio         string     `json:"ratio"`
	BaseStyle     string     `json:"base_style"`
	Theme         VideoTheme `json:"theme"`
	Common        string     `json:"-"` // the kit code: large, the cards list does not need it
	Character     string     `json:"character"`
	Rules         []string   `json:"rules"`
	SourceProject string     `json:"source_project,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// VideoStyleCardRepository stores style cards; the video repository implements it.
type VideoStyleCardRepository interface {
	ListStyles(ctx context.Context, userID int64) ([]VideoStyleCard, error)
	GetStyle(ctx context.Context, userID, id int64) (*VideoStyleCard, error) // nil when missing or not the user's
	CreateStyle(ctx context.Context, st *VideoStyleCard) error
	UpdateStyle(ctx context.Context, st *VideoStyleCard) error
	DeleteStyle(ctx context.Context, userID, id int64) (bool, error)
}

func (s *VideoService) styleRepo() (VideoStyleCardRepository, bool) {
	st, ok := s.repo.(VideoStyleCardRepository)
	return st, ok
}

// --- style cards ---------------------------------------------------------------------------------

func (s *VideoService) ListStyles(ctx context.Context, userID int64) ([]VideoStyleCard, error) {
	repo, ok := s.styleRepo()
	if !ok {
		return []VideoStyleCard{}, nil
	}
	items, err := repo.ListStyles(ctx, userID)
	if items == nil {
		items = []VideoStyleCard{}
	}
	return items, err
}

// SaveStyle saves the look of one of the user's films as a new style card.
func (s *VideoService) SaveStyle(ctx context.Context, userID int64, projectID, name string, rules []string) (*VideoStyleCard, error) {
	repo, ok := s.styleRepo()
	if !ok {
		return nil, ErrVideoStyleOff
	}
	p, err := s.own(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if s.running(p.ID) || p.Status == VideoStatusRunning {
		return nil, ErrVideoBusy
	}
	if p.Mode != VideoModeFilm || p.Spec == nil || strings.TrimSpace(p.Spec.Common) == "" {
		return nil, ErrVideoStyleNoKit
	}
	existing, err := repo.ListStyles(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= videoStylesPerUser {
		return nil, ErrVideoStyleFull
	}
	name = clipRunes(name, videoStyleNameMax)
	if name == "" {
		name = clipRunes(p.Title, videoStyleNameMax)
	}
	merged := append(append([]string(nil), p.Spec.Rules...), rules...)
	st := &VideoStyleCard{
		UserID: userID, Name: name, Ratio: findVideoRatioBySize(p.Width, p.Height), BaseStyle: p.Options.Style,
		Theme: p.Spec.Theme, Common: p.Spec.Common, Character: p.Spec.Character, Rules: cleanVideoRules(merged), SourceProject: p.ID,
	}
	if err := repo.CreateStyle(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// UpdateStyle renames a card and/or replaces its rules (nil keeps them).
func (s *VideoService) UpdateStyle(ctx context.Context, userID, id int64, name *string, rules []string) (*VideoStyleCard, error) {
	repo, ok := s.styleRepo()
	if !ok {
		return nil, ErrVideoStyleOff
	}
	st, err := repo.GetStyle(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrVideoStyleNotFound
	}
	if name != nil {
		if n := clipRunes(*name, videoStyleNameMax); n != "" {
			st.Name = n
		}
	}
	if rules != nil {
		st.Rules = cleanVideoRules(rules)
	}
	if err := repo.UpdateStyle(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *VideoService) DeleteStyle(ctx context.Context, userID, id int64) error {
	repo, ok := s.styleRepo()
	if !ok {
		return ErrVideoStyleOff
	}
	found, err := repo.DeleteStyle(ctx, userID, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrVideoStyleNotFound
	}
	return nil
}

// cleanVideoRules trims, drops empty and repeated rules and keeps at most videoStyleRulesMax.
func cleanVideoRules(rules []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, r := range rules {
		r = clipRunes(strings.Join(strings.Fields(r), " "), videoStyleRuleMax)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	if len(out) > videoStyleRulesMax {
		out = out[len(out)-videoStyleRulesMax:] // the newest rules win
	}
	return out
}

func findVideoRatioBySize(w, h int) string {
	for _, r := range VideoRatios {
		if r.Width == w && r.Height == h {
			return r.ID
		}
	}
	return VideoRatios[0].ID
}

// applyStyleCard sets a new project's ratio and base style from the user's card; false when the card
// is gone (the film is then made without it).
func (s *VideoService) applyStyleCard(ctx context.Context, userID int64, o *VideoOptions) (bool, error) {
	if o.MyStyle <= 0 {
		o.MyStyle, o.MyStyleName = 0, ""
		return false, nil
	}
	repo, ok := s.styleRepo()
	if !ok {
		o.MyStyle, o.MyStyleName = 0, ""
		return false, nil
	}
	st, err := repo.GetStyle(ctx, userID, o.MyStyle)
	if err != nil {
		return false, err
	}
	if st == nil {
		return false, ErrVideoStyleNotFound
	}
	o.MyStyleName = st.Name
	if st.Ratio != "" {
		o.Ratio = findVideoRatio(st.Ratio).ID
	}
	if st.BaseStyle != "" {
		o.Style = findVideoStyle(st.BaseStyle).ID
	}
	return true, nil
}

// styleCard loads the project's style card at run time; nil when it has none or it was deleted.
func (s *VideoService) styleCard(ctx context.Context, p *VideoProject) *VideoStyleCard {
	if p.Options.MyStyle <= 0 {
		return nil
	}
	repo, ok := s.styleRepo()
	if !ok {
		return nil
	}
	st, err := repo.GetStyle(ctx, p.UserID, p.Options.MyStyle)
	if err != nil || st == nil {
		return nil
	}
	return st
}

// videoStyleKitInstruction adapts a series' saved kit to a new episode.
const videoStyleKitInstruction = "This is a NEW video of the same series. The kit above is the series' approved look: keep it exactly — palette, fonts, type scale, corner radius, backdrop, chrome positions, card style, motion feel, the recurring character's drawing and every K member with its signature. Change only what belongs to the previous video: titles, series / episode labels and any text or helper that names the previous video's topic; make them fit this video. Add a helper only if this video's scene plans clearly need one."

// --- 先审镜头剧本 ---------------------------------------------------------------------------------

// scriptGate stops a run after the script when the user wants to confirm it first. It reports
// whether the run stopped.
func (s *VideoService) scriptGate(ctx context.Context, r *videoRun) bool {
	p := r.p
	if p.Mode != VideoModeFilm || !p.Options.Confirm || p.Options.ScriptOK || p.Spec == nil {
		return false
	}
	r.mu.Lock()
	p.Status, p.Stage = VideoStatusScript, "script"
	n, total := len(p.Spec.Scenes), p.Spec.TotalDuration()
	r.mu.Unlock()
	s.save(ctx, r)
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "script", Text: fmt.Sprintf(
		"镜头剧本写好了：%d 段，约 %.0f 秒。先别急着做动画——到「脚本」页看一遍每段的看点、旁白、画面和镜头，直接改文字；满意后点「确认，开始制作」，才会配音和写动画。", n, total)})
	return true
}

// ApproveScript applies the user's edits to the shot script and starts making the film.
func (s *VideoService) ApproveScript(ctx context.Context, key *APIKey, id string, edits []VideoSceneEdit) (*VideoProject, error) {
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Status != VideoStatusScript || p.Spec == nil {
			return ErrVideoNoScriptWait
		}
		changed := 0
		for _, e := range edits {
			for i := range p.Spec.Scenes {
				sc := &p.Spec.Scenes[i]
				if sc.ID != e.ID {
					continue
				}
				before := [5]string{sc.Title, sc.Narration, sc.Visual, sc.Hook, sc.Camera}
				if t := clipRunes(e.Title, 40); t != "" {
					sc.Title = t
				}
				if n := clipRunes(e.Narration, 1500); n != "" && !videoSilent(p) {
					sc.Narration = n
					sc.Duration = math.Max(videoMinSceneSeconds, float64(len([]rune(n)))/4.5+videoSceneTailPause)
				}
				if v := clipRunes(e.Visual, 3000); v != "" {
					sc.Visual = v
				}
				if h := clipRunes(e.Hook, 200); h != "" {
					sc.Hook = h
				}
				if c := clipRunes(e.Camera, 300); c != "" {
					sc.Camera = c
				}
				if [5]string{sc.Title, sc.Narration, sc.Visual, sc.Hook, sc.Camera} != before {
					changed++
				}
			}
		}
		p.Options.ScriptOK = true
		text := "确认镜头剧本，开始制作"
		if changed > 0 {
			text = fmt.Sprintf("改了 %d 段镜头剧本，确认，开始制作", changed)
		}
		_ = s.repo.AddVersion(ctx, p.ID, "确认的镜头剧本", p.Spec)
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: text})
	}, s.build)
}

// --- 复盘 ----------------------------------------------------------------------------------------

// VideoRetro is a film's retrospective: what had to be redone and why, what worked, and rules to keep.
type VideoRetro struct {
	Summary string             `json:"summary"`
	Rework  []VideoRetroRework `json:"rework"`
	Keep    []string           `json:"keep"`
	Rules   []string           `json:"rules"` // proposed new house rules
	Drop    []string           `json:"drop"`  // existing rules the film suggests deleting
}

type VideoRetroRework struct {
	What  string `json:"what"`
	Cause string `json:"cause"` // 规则没写 / 写了没执行 / 判断失误 / 其他
}

// Retro asks the model for a retrospective of a finished film. The user picks which rules to keep.
func (s *VideoService) Retro(ctx context.Context, key *APIKey, projectID string) (*VideoRetro, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	p, err := s.own(ctx, key.UserID, projectID)
	if err != nil {
		return nil, err
	}
	if p.Mode != VideoModeFilm || p.Spec == nil || s.running(p.ID) || p.Status == VideoStatusRunning {
		return nil, ErrVideoNotReady
	}
	if s.stream == nil {
		return nil, ErrVideoStyleOff
	}
	if err := s.checkBalance(ctx, key); err != nil {
		return nil, err
	}
	runKey, model, err := s.runKey(ctx, p, key.Key)
	if err != nil {
		return nil, err
	}
	events, err := s.repo.ListEvents(ctx, p.ID, 0, 500)
	if err != nil {
		return nil, err
	}
	var rules []string
	if card := s.styleCard(ctx, p); card != nil {
		rules = card.Rules
	} else {
		rules = p.Spec.Rules
	}
	r := &videoRun{userID: p.UserID, key: runKey, model: model, p: p}
	answer, err := s.chat(ctx, r, videoRetroPrompt(p, events, rules), videoRetroMaxTokens)
	if err != nil {
		return nil, err
	}
	var out VideoRetro
	raw := extractJSONObject(answer)
	if raw == "" || json.Unmarshal([]byte(raw), &out) != nil {
		return nil, errLearnRunFailed("复盘没写成，请再试一次")
	}
	out.Summary = clipRunes(out.Summary, 300)
	var rework []VideoRetroRework
	for _, w := range out.Rework {
		if w.What = clipRunes(w.What, 160); w.What != "" && len(rework) < 8 {
			w.Cause = clipRunes(w.Cause, 20)
			rework = append(rework, w)
		}
	}
	out.Rework = rework
	out.Keep = clipVideoList(out.Keep, 5, 160)
	out.Rules = clipVideoList(out.Rules, 5, videoStyleRuleMax)
	out.Drop = clipVideoList(out.Drop, 5, videoStyleRuleMax)
	if out.Rework == nil {
		out.Rework = []VideoRetroRework{}
	}
	// The call is billed like the film's other calls: keep its usage with the project.
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "assistant", Text: "复盘：" + out.Summary})
	return &out, nil
}

func clipVideoList(items []string, n, runes int) []string {
	out := []string{}
	for _, it := range items {
		if it = clipRunes(it, runes); it != "" && len(out) < n {
			out = append(out, it)
		}
	}
	return out
}

func videoRetroPrompt(p *VideoProject, events []VideoEvent, rules []string) []LearnMessage {
	schema := VideoRetro{
		Summary: "一句话：这部片子哪里顺、哪里返工最多",
		Rework:  []VideoRetroRework{{What: "第 3 段的流程图字太小，改了两次", Cause: "规则没写"}},
		Keep:    []string{"第 2 段任务纸撕成四份被吸走：看点清楚，以后做对比可以照这个"},
		Rules:   []string{"屏幕上的正文不小于 28px，一屏最多 3 行"},
		Drop:    []string{"（已有规则里这部片子证明多余或互相冲突的那条，原文照抄）"},
	}
	var sys strings.Builder
	_, _ = sys.WriteString("你是讲解视频团队的制片，在一部片子交付后带用户复盘。目标：让下一部同系列的片子少返工，而不是写总结报告。\n")
	_, _ = sys.WriteString("做法：\n")
	_, _ = sys.WriteString("1. 从对话记录里找出返工：用户要求改了什么、哪些分镜被重做或修复。每条写清改了什么，并判断原因：「规则没写」（以前没人提过这个要求）、「写了没执行」（已有规则里写了，这次还是没做到）、「判断失误」（方向就选错了）、「其他」。只写记录里真的发生过的，没有返工就留空数组。\n")
	_, _ = sys.WriteString("2. keep：这部片子里做得好、以后可以当对标的画面或做法（写具体分镜和原因），最多 3 条；看不出来就留空。\n")
	_, _ = sys.WriteString("3. rules：建议以后同系列都遵守的新规则，最多 3 条。每条一句话、可直接照做（写清对象和数值，如字号、时长、位置），只针对「规则没写」的返工，不重复已有规则，不写「画面要好看」这类空话。没有就留空——规则不是越多越好，太多会互相打架、执行不到。\n")
	_, _ = sys.WriteString("4. drop：已有规则里这部片子证明多余、过时或和别的规则冲突的，原文照抄，最多 2 条；没有就留空。\n")
	_, _ = sys.WriteString("用简体中文，只回答 JSON，结构如下：\n" + videoJSONSchemaHint(schema))

	var u strings.Builder
	fmt.Fprintf(&u, "片名：%s\n时长：约 %.0f 秒，%d 个分镜\n", p.Title, p.Spec.TotalDuration(), len(p.Spec.Scenes))
	_, _ = u.WriteString("分镜：\n")
	for i, sc := range p.Spec.Scenes {
		fmt.Fprintf(&u, "%d. %s", i+1, sc.Title)
		if sc.Hook != "" {
			_, _ = u.WriteString("（看点：" + sc.Hook + "）")
		}
		_, _ = u.WriteString("\n")
	}
	if len(rules) > 0 {
		_, _ = u.WriteString("已有规则：\n")
		for _, r := range rules {
			_, _ = u.WriteString("- " + r + "\n")
		}
	} else {
		_, _ = u.WriteString("已有规则：无\n")
	}
	var log strings.Builder
	for _, e := range events {
		var who string
		switch e.Kind {
		case "user":
			who = "用户"
		case "assistant", "done", "script":
			who = "AI"
		case "error":
			who = "出错"
		default:
			continue
		}
		fmt.Fprintf(&log, "[%s] %s\n", who, clipRunes(e.Text, 400))
	}
	transcript := []rune(log.String())
	if len(transcript) > videoRetroTranscript {
		transcript = transcript[len(transcript)-videoRetroTranscript:]
	}
	_, _ = u.WriteString("对话记录（从早到晚）：\n" + string(transcript))
	return []LearnMessage{{Role: "system", Content: sys.String()}, {Role: "user", Content: u.String()}}
}
