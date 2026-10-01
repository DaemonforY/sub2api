package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Prompt library: community sources synced by the server (PromptLibrarySyncService), prompts users
// save for themselves or share (reviewed before they go public), admin curation, usage counts and
// "for you" recommendations. The canvas reads it cross-origin; listing is anonymous, everything
// personal is API-key authenticated like the other companion-app endpoints.

const (
	PromptSourceUser     = "user"
	PromptSourceOfficial = "official"

	PromptStatusActive    = "active"
	PromptStatusHidden    = "hidden"
	PromptStatusPending   = "pending"
	PromptStatusRejected  = "rejected"
	PromptStatusDuplicate = "duplicate"

	PromptVisibilityPublic  = "public"
	PromptVisibilityPrivate = "private"

	PromptSortPopular     = "popular"
	PromptSortRecommended = "recommended"
	PromptSortLatest      = "latest"

	promptTitleMaxRunes       = 60
	promptTextMaxRunes        = 8000
	promptDescriptionMaxRunes = 500
	promptTagMaxRunes         = 16
	promptMaxTags             = 8
	promptMaxScenes           = 4
	promptUserMaxItems        = 500
	promptListMaxPageSize     = 60
	promptRecommendMax        = 60
	// A repeated use by the same user only counts again after this long (the global "most used" order).
	promptUseCountWindow = 30 * time.Minute
)

var promptStatuses = map[string]bool{PromptStatusActive: true, PromptStatusHidden: true, PromptStatusPending: true, PromptStatusRejected: true, PromptStatusDuplicate: true}

var (
	ErrPromptNotFound      = infraerrors.NotFound("PROMPT_NOT_FOUND", "提示词不存在或已下架")
	ErrPromptTitleRequired = infraerrors.BadRequest("PROMPT_TITLE_REQUIRED", "请填写标题")
	ErrPromptTextRequired  = infraerrors.BadRequest("PROMPT_TEXT_REQUIRED", "请填写提示词内容")
	ErrPromptTooLong       = infraerrors.BadRequest("PROMPT_TOO_LONG", "提示词太长（上限 8000 字），请精简后再保存")
	ErrPromptTooMany       = infraerrors.BadRequest("PROMPT_TOO_MANY", "最多保存 500 条自己的提示词，请先删除一些不用的")
	ErrPromptInvalidScene  = infraerrors.BadRequest("PROMPT_INVALID_SCENE", "场景分类不正确")
	ErrPromptInvalidStatus = infraerrors.BadRequest("PROMPT_INVALID_STATUS", "状态不正确")
	ErrPromptInvalidKind   = infraerrors.BadRequest("PROMPT_INVALID_KIND", "类型只能是图片或视频")
	ErrPromptInvalidCover  = infraerrors.BadRequest("PROMPT_INVALID_COVER", "封面图无效，请重新上传")
	ErrPromptSourceItem    = infraerrors.BadRequest("PROMPT_SOURCE_ITEM", "社区来源的提示词不能删除（下次同步会恢复），请改为隐藏")
	ErrPromptBatchInvalid  = infraerrors.BadRequest("PROMPT_BATCH_INVALID", "批量操作参数不正确")
	ErrPromptSourceUnknown = infraerrors.NotFound("PROMPT_SOURCE_NOT_FOUND", "提示词来源不存在")
)

// PromptItem is one library entry.
type PromptItem struct {
	ID          int64  `json:"id"`
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	ExternalID  string `json:"external_id"`
	OwnerUserID *int64 `json:"owner_user_id,omitempty"`
	OwnerEmail  string `json:"owner_email,omitempty"`
	Mine        bool   `json:"mine,omitempty"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	// OriginalTitle is the source title when a Chinese translation is shown instead.
	OriginalTitle      string     `json:"original_title,omitempty"`
	TitleZh            string     `json:"-"`
	Prompt             string     `json:"prompt"`
	Description        string     `json:"description"`
	CoverURL           string     `json:"cover_url"`
	ReferenceImageURLs []string   `json:"reference_image_urls"`
	SourceTags         []string   `json:"source_tags"`
	Scenes             []string   `json:"scenes"`
	Tags               []string   `json:"tags"`
	Model              string     `json:"model"`
	Lang               string     `json:"lang"`
	NeedsReference     bool       `json:"needs_reference"`
	AutoFlags          []string   `json:"auto_flags,omitempty"`
	Author             string     `json:"author"`
	SourceURL          string     `json:"source_url"`
	Visibility         string     `json:"visibility"`
	Status             string     `json:"status"`
	ReviewNote         string     `json:"review_note,omitempty"`
	Curated            bool       `json:"curated"`
	Featured           bool       `json:"featured"`
	QualityScore       int        `json:"quality_score"`
	UseCount           int64      `json:"use_count"`
	FavoriteCount      int64      `json:"favorite_count"`
	DedupeKey          string     `json:"-"`
	SyncHash           string     `json:"-"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// PublicView drops moderation details before an item leaves the admin API.
func (p PromptItem) PublicView() PromptItem {
	p.OwnerUserID = nil
	p.OwnerEmail = ""
	p.AutoFlags = nil
	p.QualityScore = 0
	p.Curated = false
	if !p.Mine {
		p.ReviewNote = ""
	}
	if p.SourceID == PromptSourceUser && !p.Mine {
		p.Author = ""
	}
	return p
}

// PromptListQuery filters a listing. The public listing always means status=active, visibility=public.
type PromptListQuery struct {
	Keyword  string
	Scene    string
	Model    string // "" | "here" (usable with this site's models) | "other"
	SourceID string
	Kind     string
	Tag      string
	Sort     string
	Page     int
	PageSize int

	// Admin-only filters.
	Admin    bool
	Status   string // "" = all statuses (admin) / active (public)
	Curated  *bool
	Featured *bool
	// Owner lists one user's own items (any status).
	OwnerUserID int64
}

func (q *PromptListQuery) normalize() {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if utf8.RuneCountInString(q.Keyword) > 100 {
		q.Keyword = string([]rune(q.Keyword)[:100])
	}
	if !IsPromptScene(q.Scene) {
		q.Scene = ""
	}
	if q.Model != "here" && q.Model != "other" {
		q.Model = ""
	}
	if q.Kind != "image" && q.Kind != "video" {
		q.Kind = ""
	}
	if q.Sort != PromptSortRecommended && q.Sort != PromptSortLatest {
		q.Sort = PromptSortPopular
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 24
	}
	maxSize := promptListMaxPageSize
	if q.Admin {
		maxSize = 200
	}
	if q.PageSize > maxSize {
		q.PageSize = maxSize
	}
	if !q.Admin {
		q.Status = ""
		q.Curated = nil
		q.Featured = nil
	} else if !promptStatuses[q.Status] {
		q.Status = ""
	}
	q.Tag = strings.TrimSpace(q.Tag)
}

type PromptListResult struct {
	Items       []PromptItem      `json:"items"`
	Total       int64             `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	SceneCounts map[string]int64  `json:"scene_counts"`
	Sources     []PromptSourceRef `json:"sources,omitempty"`
}

type PromptSourceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PromptSource is one community source synced by the server.
type PromptSource struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Format       string     `json:"format"`
	URL          string     `json:"url"`
	Homepage     string     `json:"homepage"`
	Enabled      bool       `json:"enabled"`
	ItemCount    int        `json:"item_count"`
	ActiveCount  int64      `json:"active_count"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	LastError    string     `json:"last_error"`
	Syncing      bool       `json:"syncing"`
}

// PromptUseHistory is one row of a user's usage, joined with the item's classification.
type PromptUseHistory struct {
	ItemID     int64
	Scenes     []string
	Model      string
	Lang       string
	Uses       int
	Favorited  bool
	LastUsedAt time.Time
}

// PromptBatchOp is an admin bulk edit.
type PromptBatchOp struct {
	Action   string   `json:"action"` // add_scenes | remove_scenes | set_scenes | add_tags | remove_tags | set_status | set_featured | mark_reviewed
	Scenes   []string `json:"scenes"`
	Tags     []string `json:"tags"`
	Status   string   `json:"status"`
	Note     string   `json:"note"`
	Featured bool     `json:"featured"`
}

type PromptTagCount struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type PromptLibraryStats struct {
	StatusCounts map[string]int64 `json:"status_counts"`
	PendingUser  int64            `json:"pending_user"`
	Uncurated    int64            `json:"uncurated"`
	TotalUses    int64            `json:"total_uses"`
}

type PromptLibraryRepository interface {
	List(ctx context.Context, q PromptListQuery) ([]PromptItem, int64, error)
	SceneCounts(ctx context.Context, q PromptListQuery) (map[string]int64, error)
	// Get returns nil when the item does not exist.
	Get(ctx context.Context, id int64) (*PromptItem, error)
	Insert(ctx context.Context, item *PromptItem) error
	// Update writes the editable fields of an item (content, classification, status, flags).
	Update(ctx context.Context, item *PromptItem) error
	Delete(ctx context.Context, id int64) error
	CountOwned(ctx context.Context, userID int64) (int, error)
	Batch(ctx context.Context, ids []int64, op PromptBatchOp) (int64, error)
	Stats(ctx context.Context) (*PromptLibraryStats, error)
	TagCounts(ctx context.Context, limit int) ([]PromptTagCount, error)

	// RecordUse counts a use; the global use_count only moves when the user's previous use is older than window.
	RecordUse(ctx context.Context, userID, itemID int64, window time.Duration) (int64, error)
	SetFavorite(ctx context.Context, userID, itemID int64, favorited bool) (int64, error)
	UserHistory(ctx context.Context, userID int64, limit int) ([]PromptUseHistory, error)
	// Candidates returns public items in any of scenes (all when empty) the user has not used, most used first.
	Candidates(ctx context.Context, userID int64, kind string, scenes []string, limit int) ([]PromptItem, error)

	ListSources(ctx context.Context) ([]PromptSource, error)
	GetSource(ctx context.Context, id string) (*PromptSource, error)
	SetSourceEnabled(ctx context.Context, id string, enabled bool) error
	RecordSourceSync(ctx context.Context, id string, count int, syncErr string) error
	// UpsertSourceItems writes a source's items; curated items keep their title, classification and status.
	UpsertSourceItems(ctx context.Context, sourceID string, items []PromptItem) error
	// RefreshDuplicates keeps one visible copy of prompts published by several sources.
	RefreshDuplicates(ctx context.Context) error
	// ApplyTitleTranslations fills title_zh for source items whose title matches a key.
	ApplyTitleTranslations(ctx context.Context, translations map[string]string) (int64, error)
	UntranslatedTitles(ctx context.Context, limit int) ([]string, error)
	// ApplySceneOverrides sets scenes by "<source_id>:<external_id>" on items no admin has edited.
	ApplySceneOverrides(ctx context.Context, scenes map[string][]string) (int64, error)
	CountUntranslated(ctx context.Context) (int64, error)

	InsertCover(ctx context.Context, file string, userID, size int64) error
	CountCoversSince(ctx context.Context, userID int64, since time.Time) (int, error)
	CoverOwnedBy(ctx context.Context, file string, userID int64) (bool, error)
}

// PromptLibraryService implements the library for the canvas, users and admins.
type PromptLibraryService struct {
	repo   PromptLibraryRepository
	covers *PromptCoverStore
	now    func() time.Time

	// Anonymous listings are identical for everyone; short searches cannot use the trigram index,
	// so results are kept for a minute.
	cacheMu sync.Mutex
	cache   map[string]promptListCacheEntry
}

type promptListCacheEntry struct {
	res     *PromptListResult
	expires time.Time
}

const (
	promptListCacheTTL = time.Minute
	promptListCacheMax = 500
)

func NewPromptLibraryService(repo PromptLibraryRepository, covers *PromptCoverStore) *PromptLibraryService {
	return &PromptLibraryService{repo: repo, covers: covers, now: time.Now, cache: map[string]promptListCacheEntry{}}
}

func (s *PromptLibraryService) Covers() *PromptCoverStore { return s.covers }

// ListPublic is the canvas library listing (anonymous).
func (s *PromptLibraryService) ListPublic(ctx context.Context, q PromptListQuery) (*PromptListResult, error) {
	q.Admin = false
	q.OwnerUserID = 0
	q.normalize()
	key := fmt.Sprintf("%q|%s|%s|%s|%s|%q|%s|%d|%d", q.Keyword, q.Scene, q.Model, q.SourceID, q.Kind, q.Tag, q.Sort, q.Page, q.PageSize)
	now := s.now()
	s.cacheMu.Lock()
	if entry, ok := s.cache[key]; ok && now.Before(entry.expires) {
		s.cacheMu.Unlock()
		return entry.res, nil
	}
	s.cacheMu.Unlock()
	res, err := s.list(ctx, q, true)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	if len(s.cache) >= promptListCacheMax {
		s.cache = map[string]promptListCacheEntry{}
	}
	s.cache[key] = promptListCacheEntry{res: res, expires: now.Add(promptListCacheTTL)}
	s.cacheMu.Unlock()
	return res, nil
}

// ListAdmin lists every item with moderation filters.
func (s *PromptLibraryService) ListAdmin(ctx context.Context, q PromptListQuery) (*PromptListResult, error) {
	q.Admin = true
	return s.list(ctx, q, false)
}

// ListMine lists the prompts a user saved (every status).
func (s *PromptLibraryService) ListMine(ctx context.Context, userID int64, q PromptListQuery) (*PromptListResult, error) {
	q.Admin = false
	q.OwnerUserID = userID
	if q.Sort == "" {
		q.Sort = PromptSortLatest
	}
	res, err := s.list(ctx, q, false)
	if err != nil {
		return nil, err
	}
	for i := range res.Items {
		res.Items[i].Mine = true
		res.Items[i] = res.Items[i].PublicView()
	}
	return res, nil
}

func (s *PromptLibraryService) list(ctx context.Context, q PromptListQuery, public bool) (*PromptListResult, error) {
	q.normalize()
	// The scene counts scan the same rows as the listing; run them side by side.
	type countsResult struct {
		counts map[string]int64
		err    error
	}
	countsCh := make(chan countsResult, 1)
	go func() {
		counts, err := s.repo.SceneCounts(ctx, q)
		countsCh <- countsResult{counts, err}
	}()
	items, total, err := s.repo.List(ctx, q)
	cr := <-countsCh
	if err != nil {
		return nil, err
	}
	if cr.err != nil {
		return nil, cr.err
	}
	counts := cr.counts
	if public {
		for i := range items {
			items[i] = items[i].PublicView()
		}
	}
	res := &PromptListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize, SceneCounts: counts}
	if public {
		sources, err := s.repo.ListSources(ctx)
		if err != nil {
			return nil, err
		}
		for _, src := range sources {
			if src.Enabled && src.ActiveCount > 0 {
				res.Sources = append(res.Sources, PromptSourceRef{ID: src.ID, Name: src.Name})
			}
		}
		res.Sources = append(res.Sources, PromptSourceRef{ID: PromptSourceUser, Name: "社区分享"})
	}
	return res, nil
}

// visibleTo loads an item the user may see: public and active, or their own.
func (s *PromptLibraryService) visibleTo(ctx context.Context, userID, id int64) (*PromptItem, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrPromptNotFound
	}
	owned := item.OwnerUserID != nil && *item.OwnerUserID == userID
	if !owned && (item.Status != PromptStatusActive || item.Visibility != PromptVisibilityPublic) {
		return nil, ErrPromptNotFound
	}
	item.Mine = owned
	return item, nil
}

// RecordUse is called when the user draws with or copies a prompt.
func (s *PromptLibraryService) RecordUse(ctx context.Context, userID, id int64) (int64, error) {
	if _, err := s.visibleTo(ctx, userID, id); err != nil {
		return 0, err
	}
	return s.repo.RecordUse(ctx, userID, id, promptUseCountWindow)
}

// SetFavorite mirrors the canvas favorite toggle (the favorites list itself syncs via app-state).
func (s *PromptLibraryService) SetFavorite(ctx context.Context, userID, id int64, favorited bool) (int64, error) {
	if _, err := s.visibleTo(ctx, userID, id); err != nil {
		return 0, err
	}
	return s.repo.SetFavorite(ctx, userID, id, favorited)
}

// PromptProfileScene is one scene of a user's taste profile.
type PromptProfileScene struct {
	Scene string  `json:"scene"`
	Share float64 `json:"share"`
}

type PromptRecommendations struct {
	Items []PromptItem `json:"items"`
	// Basis is "history" (from the user's own usage) or "popular" (not enough history yet).
	Basis  string               `json:"basis"`
	Scenes []PromptProfileScene `json:"scenes"`
}

type promptProfile struct {
	scenes  map[string]float64
	zhShare float64
	total   float64
}

// buildPromptProfile weighs the user's history: favorites count triple, repeated uses are capped,
// and older activity fades (half-life three weeks).
func buildPromptProfile(history []PromptUseHistory, now time.Time) promptProfile {
	p := promptProfile{scenes: map[string]float64{}}
	var zh float64
	for _, h := range history {
		uses := h.Uses
		if uses > 10 {
			uses = 10
		}
		weight := float64(uses)
		if h.Favorited {
			weight += 3
		}
		if weight <= 0 {
			continue
		}
		days := now.Sub(h.LastUsedAt).Hours() / 24
		if days < 0 {
			days = 0
		}
		weight *= math.Pow(0.5, days/21)
		for i, scene := range h.Scenes {
			if scene == "other" {
				continue
			}
			share := 1.0
			if i > 0 {
				share = 0.5
			}
			p.scenes[scene] += weight * share
		}
		if h.Lang == "zh" {
			zh += weight
		}
		p.total += weight
	}
	if p.total > 0 {
		p.zhShare = zh / p.total
	}
	return p
}

func (p promptProfile) top(n int) []PromptProfileScene {
	var sum float64
	out := make([]PromptProfileScene, 0, len(p.scenes))
	for scene, w := range p.scenes {
		out = append(out, PromptProfileScene{Scene: scene, Share: w})
		sum += w
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Share != out[j].Share {
			return out[i].Share > out[j].Share
		}
		return out[i].Scene < out[j].Scene
	})
	if len(out) > n {
		out = out[:n]
	}
	if sum > 0 {
		for i := range out {
			out[i].Share = math.Round(out[i].Share/sum*1000) / 1000
		}
	}
	return out
}

// Recommend builds the "for you" list: popular prompts in the scenes the user draws most, which
// they have not used yet, with a daily shuffle so the list does not freeze. Without history it
// falls back to what everyone uses.
func (s *PromptLibraryService) Recommend(ctx context.Context, userID int64, kind string, limit int) (*PromptRecommendations, error) {
	if limit <= 0 || limit > promptRecommendMax {
		limit = 24
	}
	if kind != "video" {
		kind = "image"
	}
	history, err := s.repo.UserHistory(ctx, userID, 300)
	if err != nil {
		return nil, err
	}
	now := s.now()
	profile := buildPromptProfile(history, now)
	top := profile.top(4)
	res := &PromptRecommendations{Basis: "popular", Scenes: top, Items: []PromptItem{}}
	picked := map[int64]bool{}

	if len(top) > 0 && profile.total > 0 {
		res.Basis = "history"
		scenes := make([]string, len(top))
		share := map[string]float64{}
		for i, sc := range top {
			scenes[i] = sc.Scene
			share[sc.Scene] = sc.Share
		}
		candidates, err := s.repo.Candidates(ctx, userID, kind, scenes, 600)
		if err != nil {
			return nil, err
		}
		day := now.UTC().Format("2006-01-02")
		type scored struct {
			item  PromptItem
			score float64
		}
		list := make([]scored, 0, len(candidates))
		for _, item := range candidates {
			var affinity float64
			for i, scene := range item.Scenes {
				w := share[scene]
				if i > 0 {
					w *= 0.5
				}
				affinity += w
			}
			score := affinity*12 + float64(item.QualityScore) + math.Log1p(float64(item.UseCount))*1.5 + math.Log1p(float64(item.FavoriteCount))
			if item.Featured {
				score += 4
			}
			if profile.zhShare >= 0.6 && item.Lang == "zh" {
				score += 2
			}
			score += promptJitter(userID, day, item.ID) * 3
			list = append(list, scored{item: item, score: score})
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
		// Keep the mix close to the user's taste instead of filling the page with one scene.
		perScene := map[string]int{}
		for _, entry := range list {
			if len(res.Items) >= limit {
				break
			}
			primary := "other"
			if len(entry.item.Scenes) > 0 {
				primary = entry.item.Scenes[0]
			}
			capacity := int(math.Round(share[primary]*float64(limit))) + 2
			if perScene[primary] >= capacity {
				continue
			}
			perScene[primary]++
			picked[entry.item.ID] = true
			res.Items = append(res.Items, entry.item.PublicView())
		}
	}

	if len(res.Items) < limit {
		popular, err := s.repo.Candidates(ctx, userID, kind, nil, limit*2)
		if err != nil {
			return nil, err
		}
		for _, item := range popular {
			if len(res.Items) >= limit {
				break
			}
			if picked[item.ID] {
				continue
			}
			picked[item.ID] = true
			res.Items = append(res.Items, item.PublicView())
		}
	}
	return res, nil
}

func promptJitter(userID int64, day string, itemID int64) float64 {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%d|%s|%d", userID, day, itemID)
	return float64(h.Sum32()%1000) / 1000
}

// PromptUserInput is what a user submits for one of their own prompts.
type PromptUserInput struct {
	Title       string   `json:"title"`
	Prompt      string   `json:"prompt"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Scenes      []string `json:"scenes"`
	Tags        []string `json:"tags"`
	CoverURL    string   `json:"cover_url"`
	// Share asks for the prompt to be listed publicly (after review).
	Share bool `json:"share"`
}

func normalizePromptTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(tag), "#＃"))
		if tag == "" {
			continue
		}
		out = append(out, truncateRunes(tag, promptTagMaxRunes))
	}
	return uniquePromptStrings(out, promptMaxTags)
}

func normalizePromptScenes(scenes []string) ([]string, error) {
	out := make([]string, 0, len(scenes))
	for _, scene := range scenes {
		scene = strings.TrimSpace(scene)
		if scene == "" {
			continue
		}
		if !IsPromptScene(scene) {
			return nil, ErrPromptInvalidScene
		}
		out = append(out, scene)
	}
	return uniquePromptStrings(out, promptMaxScenes), nil
}

func (s *PromptLibraryService) validateUserInput(ctx context.Context, userID int64, in *PromptUserInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Prompt = strings.TrimSpace(in.Prompt)
	in.Description = truncateRunes(strings.TrimSpace(in.Description), promptDescriptionMaxRunes)
	if in.Prompt == "" {
		return ErrPromptTextRequired
	}
	if utf8.RuneCountInString(in.Prompt) > promptTextMaxRunes {
		return ErrPromptTooLong
	}
	if in.Title == "" {
		in.Title = PromptDisplayTitle("", in.Prompt, nil)
	}
	if in.Title == "" {
		return ErrPromptTitleRequired
	}
	in.Title = truncateRunes(in.Title, promptTitleMaxRunes)
	if in.Kind == "" {
		in.Kind = "image"
	}
	if in.Kind != "image" && in.Kind != "video" {
		return ErrPromptInvalidKind
	}
	scenes, err := normalizePromptScenes(in.Scenes)
	if err != nil {
		return err
	}
	in.Scenes = scenes
	in.Tags = normalizePromptTags(in.Tags)
	in.CoverURL = strings.TrimSpace(in.CoverURL)
	if in.CoverURL != "" {
		file, ok := PromptCoverFileFromURL(in.CoverURL)
		if !ok {
			return ErrPromptInvalidCover
		}
		owned, err := s.repo.CoverOwnedBy(ctx, file, userID)
		if err != nil {
			return err
		}
		if !owned {
			return ErrPromptInvalidCover
		}
		in.CoverURL = PromptCoverURL(file)
	}
	return nil
}

func (s *PromptLibraryService) applyUserInput(item *PromptItem, in PromptUserInput) {
	traits := ClassifyPrompt(PromptTraitInput{Title: in.Title, Prompt: in.Prompt, Description: in.Description, Tags: in.Tags})
	item.Kind = in.Kind
	item.Title = in.Title
	item.Prompt = in.Prompt
	item.Description = in.Description
	item.CoverURL = in.CoverURL
	item.Tags = in.Tags
	item.Scenes = in.Scenes
	if len(item.Scenes) == 0 {
		// Users rarely tag carefully: also look at the title and the start of the prompt.
		auto := matchPromptScenes(append(append([]string{}, in.Tags...), in.Title, truncateRunes(in.Prompt, 200)))
		if len(auto) == 0 {
			auto = []string{"other"}
		}
		item.Scenes = uniquePromptStrings(auto, 3)
		if in.Kind == "video" {
			item.Scenes = uniquePromptStrings(append([]string{"video"}, auto...), promptMaxScenes)
		}
	}
	item.Model = traits.Model
	item.Lang = traits.Lang
	item.NeedsReference = traits.NeedsReference
	item.AutoFlags = promptAutoFlags(traits)
	item.QualityScore = PromptQualityScore(item.CoverURL, item.Title, traits)
	item.DedupeKey = PromptDedupeKey(item.Prompt)
}

func promptAutoFlags(traits PromptTraits) []string {
	flags := []string{}
	if traits.NSFW {
		flags = append(flags, "nsfw")
	}
	if traits.Sensitive {
		flags = append(flags, "sensitive")
	}
	return flags
}

// CreateMine saves a prompt for the user. Shared prompts wait for review before they are listed.
func (s *PromptLibraryService) CreateMine(ctx context.Context, userID int64, in PromptUserInput) (*PromptItem, error) {
	if err := s.validateUserInput(ctx, userID, &in); err != nil {
		return nil, err
	}
	count, err := s.repo.CountOwned(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= promptUserMaxItems {
		return nil, ErrPromptTooMany
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, fmt.Errorf("generate prompt id: %w", err)
	}
	owner := userID
	item := &PromptItem{SourceID: PromptSourceUser, ExternalID: hex.EncodeToString(buf[:]), OwnerUserID: &owner, ReferenceImageURLs: []string{}, SourceTags: []string{}}
	s.applyUserInput(item, in)
	setPromptSharing(item, in.Share, true)
	if err := s.repo.Insert(ctx, item); err != nil {
		return nil, err
	}
	item.Mine = true
	view := item.PublicView()
	return &view, nil
}

// setPromptSharing moves a user item between private and shared. A shared prompt goes (back) to
// review whenever its content changed or it was not public before.
func setPromptSharing(item *PromptItem, share, contentChanged bool) {
	if !share {
		item.Visibility = PromptVisibilityPrivate
		if item.Status != PromptStatusHidden {
			item.Status = PromptStatusActive
		}
		item.ReviewNote = ""
		return
	}
	wasPublic := item.Visibility == PromptVisibilityPublic && item.Status == PromptStatusActive
	item.Visibility = PromptVisibilityPublic
	if !wasPublic || contentChanged {
		item.Status = PromptStatusPending
		item.ReviewNote = ""
	}
}

// UpdateMine edits one of the user's prompts.
func (s *PromptLibraryService) UpdateMine(ctx context.Context, userID, id int64, in PromptUserInput) (*PromptItem, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.OwnerUserID == nil || *item.OwnerUserID != userID {
		return nil, ErrPromptNotFound
	}
	if err := s.validateUserInput(ctx, userID, &in); err != nil {
		return nil, err
	}
	changed := item.Title != in.Title || item.Prompt != in.Prompt || item.Description != in.Description || item.CoverURL != in.CoverURL ||
		strings.Join(item.Tags, "\x00") != strings.Join(in.Tags, "\x00")
	s.applyUserInput(item, in)
	setPromptSharing(item, in.Share, changed)
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	item.Mine = true
	view := item.PublicView()
	return &view, nil
}

// DeleteMine removes one of the user's prompts.
func (s *PromptLibraryService) DeleteMine(ctx context.Context, userID, id int64) error {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if item == nil || item.OwnerUserID == nil || *item.OwnerUserID != userID {
		return ErrPromptNotFound
	}
	return s.repo.Delete(ctx, id)
}

// PromptAdminInput is an admin edit of any item.
type PromptAdminInput struct {
	Title          string   `json:"title"`
	Prompt         string   `json:"prompt"`
	Description    string   `json:"description"`
	CoverURL       string   `json:"cover_url"`
	Kind           string   `json:"kind"`
	Scenes         []string `json:"scenes"`
	Tags           []string `json:"tags"`
	Model          string   `json:"model"`
	NeedsReference bool     `json:"needs_reference"`
	Status         string   `json:"status"`
	Visibility     string   `json:"visibility"`
	Featured       bool     `json:"featured"`
	ReviewNote     string   `json:"review_note"`
}

var promptModels = map[string]bool{"gpt-image-2": true, "nano-banana": true, "gpt-4o": true, "unknown": true}

func (s *PromptLibraryService) validateAdminInput(in *PromptAdminInput) error {
	in.Title = truncateRunes(strings.TrimSpace(in.Title), 80)
	in.Prompt = strings.TrimSpace(in.Prompt)
	in.Description = truncateRunes(strings.TrimSpace(in.Description), 2000)
	in.CoverURL = strings.TrimSpace(in.CoverURL)
	in.ReviewNote = truncateRunes(strings.TrimSpace(in.ReviewNote), 200)
	if in.Title == "" {
		return ErrPromptTitleRequired
	}
	if in.Prompt == "" {
		return ErrPromptTextRequired
	}
	if utf8.RuneCountInString(in.Prompt) > 20000 {
		return ErrPromptTooLong
	}
	if in.Kind != "image" && in.Kind != "video" {
		return ErrPromptInvalidKind
	}
	if !promptStatuses[in.Status] {
		return ErrPromptInvalidStatus
	}
	if in.Visibility != PromptVisibilityPrivate {
		in.Visibility = PromptVisibilityPublic
	}
	if !promptModels[in.Model] {
		in.Model = "unknown"
	}
	if in.CoverURL != "" && !strings.HasPrefix(in.CoverURL, "https://") && !strings.HasPrefix(in.CoverURL, "http://") && !strings.HasPrefix(in.CoverURL, PromptCoverPublicPrefix) {
		return ErrPromptInvalidCover
	}
	scenes, err := normalizePromptScenes(in.Scenes)
	if err != nil {
		return err
	}
	if len(scenes) == 0 {
		scenes = []string{"other"}
	}
	in.Scenes = scenes
	in.Tags = normalizePromptTags(in.Tags)
	return nil
}

func (s *PromptLibraryService) applyAdminInput(item *PromptItem, in PromptAdminInput) {
	approving := item.Status != PromptStatusActive && in.Status == PromptStatusActive
	// The editor shows the displayed (possibly translated) title; what the admin saves is final.
	item.Title, item.TitleZh, item.OriginalTitle = in.Title, "", ""
	item.Prompt, item.Description, item.CoverURL = in.Prompt, in.Description, in.CoverURL
	item.Kind, item.Scenes, item.Tags, item.Model = in.Kind, in.Scenes, in.Tags, in.Model
	item.NeedsReference, item.Status, item.Featured, item.ReviewNote = in.NeedsReference, in.Status, in.Featured, in.ReviewNote
	if item.SourceID == PromptSourceUser {
		item.Visibility = in.Visibility
	} else {
		item.Visibility = PromptVisibilityPublic
	}
	item.Curated = true
	lang := "en"
	if promptCJK.MatchString(item.Prompt) {
		lang = "zh"
	}
	item.Lang = lang
	item.QualityScore = PromptQualityScore(item.CoverURL, item.Title, PromptTraits{Scenes: item.Scenes, Model: item.Model, Lang: lang})
	item.DedupeKey = PromptDedupeKey(item.Prompt)
	if approving && item.PublishedAt == nil {
		now := s.now()
		item.PublishedAt = &now
	}
}

// AdminGet returns any item.
func (s *PromptLibraryService) AdminGet(ctx context.Context, id int64) (*PromptItem, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrPromptNotFound
	}
	return item, nil
}

// AdminCreate adds an official prompt.
func (s *PromptLibraryService) AdminCreate(ctx context.Context, in PromptAdminInput) (*PromptItem, error) {
	if err := s.validateAdminInput(&in); err != nil {
		return nil, err
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, fmt.Errorf("generate prompt id: %w", err)
	}
	item := &PromptItem{SourceID: PromptSourceOfficial, ExternalID: hex.EncodeToString(buf[:]), ReferenceImageURLs: []string{}, SourceTags: []string{}, AutoFlags: []string{}, Status: PromptStatusPending}
	s.applyAdminInput(item, in)
	if err := s.repo.Insert(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// AdminUpdate edits any item; the item becomes curated so source syncs keep the edit.
func (s *PromptLibraryService) AdminUpdate(ctx context.Context, id int64, in PromptAdminInput) (*PromptItem, error) {
	item, err := s.AdminGet(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateAdminInput(&in); err != nil {
		return nil, err
	}
	s.applyAdminInput(item, in)
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// AdminDelete deletes user / official items; source items can only be hidden.
func (s *PromptLibraryService) AdminDelete(ctx context.Context, id int64) error {
	item, err := s.AdminGet(ctx, id)
	if err != nil {
		return err
	}
	if item.SourceID != PromptSourceUser && item.SourceID != PromptSourceOfficial {
		return ErrPromptSourceItem
	}
	return s.repo.Delete(ctx, id)
}

// AdminBatch applies one bulk edit to up to 500 items.
func (s *PromptLibraryService) AdminBatch(ctx context.Context, ids []int64, op PromptBatchOp) (int64, error) {
	clean := make([]int64, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 || len(clean) > 500 {
		return 0, ErrPromptBatchInvalid
	}
	switch op.Action {
	case "add_scenes", "remove_scenes", "set_scenes":
		scenes, err := normalizePromptScenes(op.Scenes)
		if err != nil {
			return 0, err
		}
		if len(scenes) == 0 {
			return 0, ErrPromptBatchInvalid
		}
		op.Scenes = scenes
	case "add_tags", "remove_tags":
		op.Tags = normalizePromptTags(op.Tags)
		if len(op.Tags) == 0 {
			return 0, ErrPromptBatchInvalid
		}
	case "set_status":
		if !promptStatuses[op.Status] {
			return 0, ErrPromptInvalidStatus
		}
		op.Note = truncateRunes(strings.TrimSpace(op.Note), 200)
	case "set_featured", "mark_reviewed":
	default:
		return 0, ErrPromptBatchInvalid
	}
	return s.repo.Batch(ctx, clean, op)
}

func (s *PromptLibraryService) Stats(ctx context.Context) (*PromptLibraryStats, error) {
	return s.repo.Stats(ctx)
}

func (s *PromptLibraryService) TagCounts(ctx context.Context) ([]PromptTagCount, error) {
	return s.repo.TagCounts(ctx, 200)
}
