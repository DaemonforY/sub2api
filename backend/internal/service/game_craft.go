package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 小游戏「AI 万物合成」(/games/craft). Players start from 金木水火土 and drag two elements together;
// the text model (the 课本诗画 text model on the admin's key) invents what they make.
//
//   - Every pair is asked only once for the whole site: the answer is stored and everyone who tries the
//     pair later gets it from the database, so the spend shrinks as the recipe book fills up.
//   - The first person to make an element is its discoverer; the page tells them 「全站首个发现」.
//   - Inputs must be elements that already exist, so players can't send the model free text.
//   - Open to visitors. Redis counters bound tries per minute, new pairs (model calls) per visitor per
//     day and model calls for the whole site per day. Days are Beijing days.
//   - The setting game_craft_model overrides the model ("" = the 课本诗画 text model).

const (
	SettingKeyGameCraftModel = "game_craft_model"

	craftPerMinute      = 40   // combines per visitor per minute
	craftNewGuestPerDay = 40   // new pairs (model calls) per IP per day
	craftNewUserPerDay  = 150  // … per signed-in user per day
	craftNewSitePerDay  = 4000 // … for everyone together
	craftLockTTL        = 40 * time.Second
	craftModelTimeout   = 35 * time.Second
	craftRecentCount    = 12
	craftMaxNameRunes   = 8
	craftMaxDescRunes   = 40
)

var craftStarterNames = []string{"金", "木", "水", "火", "土"}

var (
	ErrCraftInvalid  = infraerrors.BadRequest("CRAFT_INVALID", "这两个东西还没被合成出来过，刷新页面再试（Unknown element）")
	ErrCraftFailed   = infraerrors.ServiceUnavailable("CRAFT_FAILED", "造物引擎这次没想出来，再拖一次试试（The model returned nothing usable, try again）")
	ErrCraftNoModel  = infraerrors.ServiceUnavailable("CRAFT_NO_MODEL", "合成功能暂时不可用，请稍后再来（Text model not configured）")
	ErrCraftBusy     = infraerrors.TooManyRequests("CRAFT_BUSY", "有人正在合成同一对，稍等几秒再试（Someone is making this pair right now）")
	ErrCraftSiteCap  = infraerrors.TooManyRequests("CRAFT_SITE_CAP", "今天全站的新合成次数用完了，已发现的组合还能继续玩，明天再来探索新的（Daily limit for everyone reached）")
	ErrCraftRejected = infraerrors.BadRequest("CRAFT_REJECTED", "这两个合不出能展示的东西，换个组合试试（This pair has no showable result）")
)

func errCraftTooFast() error {
	return infraerrors.TooManyRequests("CRAFT_RATE", fmt.Sprintf("合成得太快了，每分钟最多 %d 次，歇一下再拖（Too many tries per minute）", craftPerMinute))
}

func errCraftDaily(perDay int, guest bool) error {
	if guest {
		return infraerrors.TooManyRequests("CRAFT_DAILY", fmt.Sprintf("今天已经发现了 %d 个新组合，登录后每天能探索更多；已发现过的组合还能继续玩（Daily limit reached; sign in for more）", perDay))
	}
	return infraerrors.TooManyRequests("CRAFT_DAILY", fmt.Sprintf("今天已经发现了 %d 个新组合，明天再来；已发现过的组合还能继续玩（Daily limit reached）", perDay))
}

// CraftElement is one element of the game. ID is its place in the site-wide order of discovery.
type CraftElement struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Emoji       string    `json:"emoji"`
	Description string    `json:"desc"`
	CreatedAt   time.Time `json:"created_at"`
}

// CraftResult is what a combine returns. First is true only for the request that made the element.
type CraftResult struct {
	Element CraftElement `json:"element"`
	First   bool         `json:"first"`
}

type CraftStart struct {
	Starters []CraftElement `json:"starters"`
	Total    int64          `json:"total"`
	Recent   []CraftElement `json:"recent"`
}

type GameCraftRepository interface {
	// Elements returns the named elements that exist, by name.
	Elements(ctx context.Context, names []string) (map[string]CraftElement, error)
	// Recipe returns the stored result of a sorted pair, nil when the pair was never made.
	Recipe(ctx context.Context, a, b string) (*CraftElement, error)
	// Save stores the pair's result, creating the element when it is new (created = true). When the
	// pair was stored meanwhile, the stored result wins and is returned.
	Save(ctx context.Context, a, b string, el CraftElement, userID int64) (CraftElement, bool, error)
	// Stats returns how many elements exist and the newest ones (starters excluded).
	Stats(ctx context.Context, recent int) (int64, []CraftElement, error)
}

type GameCraftService struct {
	repo     GameCraftRepository
	chat     libChatFunc
	counters AssistantQuotaCache
	settings SettingRepository
	now      func() time.Time
	wait     time.Duration // poll step while another request makes the same pair
}

func NewGameCraftService(repo GameCraftRepository, shihuaSvc *ShihuaService, counters AssistantQuotaCache, settings SettingRepository) *GameCraftService {
	s := &GameCraftService{repo: repo, counters: counters, settings: settings, wait: 500 * time.Millisecond}
	if shihuaSvc != nil {
		s.chat = shihuaSvc.AdminChatModel
	}
	return s
}

func (s *GameCraftService) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *GameCraftService) dayKey() string { return shihuaDayStart(s.clock()).Format("20060102") }

// Start is what the page needs first: the five starters, the size of the site's element book and the
// latest discoveries.
func (s *GameCraftService) Start(ctx context.Context) (*CraftStart, error) {
	got, err := s.repo.Elements(ctx, craftStarterNames)
	if err != nil {
		return nil, err
	}
	out := &CraftStart{Starters: make([]CraftElement, 0, len(craftStarterNames))}
	for _, n := range craftStarterNames {
		if el, ok := got[n]; ok {
			out.Starters = append(out.Starters, el)
		}
	}
	out.Total, out.Recent, err = s.repo.Stats(ctx, craftRecentCount)
	if out.Recent == nil {
		out.Recent = []CraftElement{}
	}
	return out, err
}

// Combine makes a and b into one element.
func (s *GameCraftService) Combine(ctx context.Context, v LibVisitor, a, b string) (*CraftResult, error) {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" || utf8.RuneCountInString(a) > craftMaxNameRunes || utf8.RuneCountInString(b) > craftMaxNameRunes {
		return nil, ErrCraftInvalid
	}
	if b < a {
		a, b = b, a
	}
	if err := s.count(ctx, "craft:m:"+v.key()+":"+s.clock().Format("200601021504"), 2*time.Minute, craftPerMinute, errCraftTooFast()); err != nil {
		return nil, err
	}
	if el, err := s.repo.Recipe(ctx, a, b); err != nil || el != nil {
		if el == nil {
			return nil, err
		}
		return &CraftResult{Element: *el}, nil
	}
	have, err := s.repo.Elements(ctx, []string{a, b})
	if err != nil {
		return nil, err
	}
	ea, okA := have[a]
	eb, okB := have[b]
	if !okA || !okB {
		return nil, ErrCraftInvalid
	}
	if s.chat == nil {
		return nil, ErrCraftNoModel
	}

	// One model call per pair: whoever takes the lock asks, the others wait for the stored answer.
	lock := "craft:lock:" + a + "|" + b
	if s.counters != nil {
		n, lerr := s.counters.Incr(ctx, lock, craftLockTTL)
		if lerr == nil && n > 1 {
			return s.waitFor(ctx, a, b)
		}
		if lerr == nil {
			defer func() { _ = s.counters.Decr(context.WithoutCancel(ctx), lock) }()
		}
	}

	perDay := craftNewGuestPerDay
	if v.UserID > 0 {
		perDay = craftNewUserPerDay
	}
	day := s.dayKey()
	if err := s.count(ctx, "craft:d:"+v.key()+":"+day, 26*time.Hour, perDay, errCraftDaily(perDay, v.UserID == 0)); err != nil {
		return nil, err
	}
	if err := s.count(ctx, "craft:d:all:"+day, 26*time.Hour, craftNewSitePerDay, ErrCraftSiteCap); err != nil {
		return nil, err
	}

	el, err := s.invent(ctx, ea, eb)
	if err != nil {
		return nil, err
	}
	saved, created, err := s.repo.Save(ctx, a, b, el, v.UserID)
	if err != nil {
		return nil, err
	}
	return &CraftResult{Element: saved, First: created}, nil
}

// count adds one to a Redis counter and fails with over when it passes max. Redis trouble lets the
// request through (logged): the game is not worth an outage.
func (s *GameCraftService) count(ctx context.Context, key string, ttl time.Duration, max int, over error) error {
	if s.counters == nil {
		return nil
	}
	n, err := s.counters.Incr(ctx, key, ttl)
	if err != nil {
		slog.Warn("craft: counter", "key", key, "err", err)
		return nil
	}
	if n > int64(max) {
		return over
	}
	return nil
}

func (s *GameCraftService) waitFor(ctx context.Context, a, b string) (*CraftResult, error) {
	deadline := s.clock().Add(craftModelTimeout + 5*time.Second)
	for s.clock().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.wait):
		}
		if el, err := s.repo.Recipe(ctx, a, b); err != nil || el != nil {
			if el == nil {
				return nil, err
			}
			return &CraftResult{Element: *el}, nil
		}
	}
	return nil, ErrCraftBusy
}

const craftSystemPrompt = `你是网页小游戏「AI 万物合成」的造物引擎。玩家从金、木、水、火、土出发，把两个东西合在一起，你决定合成出来的那一个新东西。

规则：
- 结果是一个具体、有画面感、大家一看就懂的名词：物质、自然现象、动植物、器物、建筑、地点、职业、神话传说、诗词意象、历史文化、科技、网络热梗都可以。
- 要让人「哦！对！」或者「哈哈，妙啊」：合理是底线，有想象力是加分。原料越复杂，结果可以越奇妙、越宏大。
- 名字用中文，1 到 6 个字，不要标点、空格、引号、括号，不要解释。不能和任一原料同名。同一个东西和自己合成也要有结果。
- 不要涉及：现实政治和政治人物、国家领导人、领土主权、宗教争议、色情低俗、血腥暴力、毒品、赌博、歧视、真实在世的人、商标品牌。原料指向这些时，给一个无害、自然的结果。
- emoji：一个最贴切的 emoji。
- desc：一句有趣的中文描述，不超过 24 个字。

例：水+火=蒸汽💨；土+水=泥巴🟫；木+火=木炭；水+水=湖；蒸汽+土=间歇泉；泥巴+火=陶罐；湖+木=荷花；荷花+诗人=爱莲说。

只输出 JSON：{"name":"…","emoji":"…","desc":"…"}`

// craftName: Chinese characters, letters, digits and the middle dot only.
var craftName = regexp.MustCompile(`^[\p{Han}A-Za-z0-9·]{1,8}$`)

// craftBlocked are words a result may never show, whatever the model says.
var craftBlocked = []string{
	"习近平", "毛泽东", "邓小平", "江泽民", "胡锦涛", "共产党", "国民党", "法轮", "台独", "藏独", "疆独", "六四", "天安门事件",
	"主席", "总书记", "总统", "特朗普", "拜登", "普京", "希特勒", "纳粹", "恐怖分子", "色情", "裸体", "做爱", "强奸", "毒品",
	"冰毒", "海洛因", "赌博", "赌场", "自杀", "血腥", "屠杀", "邪教",
}

func craftShowable(s string) bool {
	for _, w := range craftBlocked {
		if strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// invent asks the model what a + b makes, twice at most.
func (s *GameCraftService) invent(ctx context.Context, a, b CraftElement) (CraftElement, error) {
	model := ""
	if s.settings != nil {
		if m, err := s.settings.GetValue(ctx, SettingKeyGameCraftModel); err == nil {
			model = strings.TrimSpace(m)
		}
	}
	user := a.Name + " + " + b.Name
	var lastErr error = ErrCraftFailed
	for try := 0; try < 2 && ctx.Err() == nil; try++ {
		cctx, cancel := context.WithTimeout(ctx, craftModelTimeout)
		answer, _, err := s.chat(cctx, model, craftSystemPrompt, user)
		cancel()
		if err != nil {
			slog.Warn("craft: model", "a", a.Name, "b", b.Name, "err", err)
			if errors.Is(err, ErrShihuaKey) {
				return CraftElement{}, ErrCraftNoModel
			}
			continue
		}
		el, perr := parseCraftAnswer(answer, a.Name, b.Name)
		if perr == nil {
			return el, nil
		}
		slog.Warn("craft: unusable answer", "a", a.Name, "b", b.Name, "answer", truncateForLog([]byte(answer), 300))
		lastErr = perr
	}
	return CraftElement{}, lastErr
}

// CraftStarterNames are the five elements every player starts with.
func CraftStarterNames() []string { return append([]string(nil), craftStarterNames...) }

func parseCraftAnswer(answer, a, b string) (CraftElement, error) {
	answer = strings.TrimSpace(answer)
	if i, j := strings.Index(answer, "{"), strings.LastIndex(answer, "}"); i >= 0 && j > i {
		answer = answer[i : j+1]
	}
	var got struct {
		Name  string `json:"name"`
		Emoji string `json:"emoji"`
		Desc  string `json:"desc"`
	}
	if err := json.Unmarshal([]byte(answer), &got); err != nil {
		return CraftElement{}, ErrCraftFailed
	}
	name := strings.TrimSpace(got.Name)
	if !craftName.MatchString(name) || name == a || name == b {
		return CraftElement{}, ErrCraftFailed
	}
	desc := strings.TrimSpace(strings.ReplaceAll(got.Desc, "\n", " "))
	if utf8.RuneCountInString(desc) > craftMaxDescRunes {
		desc = string([]rune(desc)[:craftMaxDescRunes])
	}
	if !craftShowable(name) || !craftShowable(desc) {
		return CraftElement{}, ErrCraftRejected
	}
	return CraftElement{Name: name, Emoji: craftEmoji(got.Emoji), Description: desc}, nil
}

// craftEmoji keeps the first emoji-ish cluster (no letters or digits), "✨" when there is none.
func craftEmoji(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == ',' || r == '，' || (r < 0x2000 && r != 0x00A9 && r != 0x00AE) {
			if b.Len() > 0 {
				break
			}
			continue
		}
		if b.Len()+utf8.RuneLen(r) > 28 {
			break
		}
		_, _ = b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "✨"
	}
	return b.String()
}
