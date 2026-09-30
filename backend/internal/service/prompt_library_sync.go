package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	promptSyncInterval   = 24 * time.Hour
	promptSyncCheckEvery = time.Hour
	promptSyncStartDelay = 2 * time.Minute
	promptSyncTimeout    = 10 * time.Minute
	promptSyncMaxBody    = 64 << 20
)

// YouMind's use-case files map onto the canvas scenes.
var youmindCategoryScenes = map[string]string{
	"profile-avatar":         "portrait",
	"social-media-post":      "social",
	"infographic-edu-visual": "infographic",
	"youtube-thumbnail":      "poster",
	"comic-storyboard":       "illustration",
	"product-marketing":      "ecommerce",
	"ecommerce-main-image":   "ecommerce",
	"game-asset":             "creative",
	"poster-flyer":           "poster",
	"app-web-design":         "ui",
}

// PromptLibrarySyncService pulls the community sources into prompt_items: once a day per source,
// or on demand from the admin page.
type PromptLibrarySyncService struct {
	repo   PromptLibraryRepository
	client *http.Client

	mu      sync.Mutex
	running map[string]bool
	stop    chan struct{}
	wg      sync.WaitGroup
}

func NewPromptLibrarySyncService(repo PromptLibraryRepository, client *http.Client) *PromptLibrarySyncService {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Minute}
	}
	return &PromptLibrarySyncService{repo: repo, client: client, running: map[string]bool{}, stop: make(chan struct{})}
}

// Start runs the daily sync loop.
func (s *PromptLibrarySyncService) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		timer := time.NewTimer(promptSyncStartDelay)
		defer timer.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-timer.C:
				s.syncDue()
				timer.Reset(promptSyncCheckEvery)
			}
		}
	}()
}

func (s *PromptLibrarySyncService) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	s.wg.Wait()
}

func (s *PromptLibrarySyncService) syncDue() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	sources, err := s.repo.ListSources(ctx)
	cancel()
	if err != nil {
		slog.Warn("prompt library: list sources failed", "error", err)
		return
	}
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		if src.LastSyncedAt != nil && time.Since(*src.LastSyncedAt) < promptSyncInterval {
			continue
		}
		select {
		case <-s.stop:
			return
		default:
		}
		_ = s.SyncNow(context.Background(), src.ID)
	}
}

// IsSyncing reports whether a source is being synced right now.
func (s *PromptLibrarySyncService) IsSyncing(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[id]
}

// ListSources returns the sources with their live sync state.
func (s *PromptLibrarySyncService) ListSources(ctx context.Context) ([]PromptSource, error) {
	sources, err := s.repo.ListSources(ctx)
	if err != nil {
		return nil, err
	}
	for i := range sources {
		sources[i].Syncing = s.IsSyncing(sources[i].ID)
	}
	return sources, nil
}

func (s *PromptLibrarySyncService) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if _, err := s.source(ctx, id); err != nil {
		return err
	}
	return s.repo.SetSourceEnabled(ctx, id, enabled)
}

func (s *PromptLibrarySyncService) source(ctx context.Context, id string) (*PromptSource, error) {
	src, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, ErrPromptSourceUnknown
	}
	return src, nil
}

// SyncAsync starts a sync in the background (admin "sync now"); a running sync is not restarted.
func (s *PromptLibrarySyncService) SyncAsync(ctx context.Context, id string) error {
	if _, err := s.source(ctx, id); err != nil {
		return err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = s.SyncNow(context.Background(), id)
	}()
	return nil
}

// SyncNow fetches one source and writes its items.
func (s *PromptLibrarySyncService) SyncNow(ctx context.Context, id string) error {
	s.mu.Lock()
	if s.running[id] {
		s.mu.Unlock()
		return nil
	}
	s.running[id] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, id)
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, promptSyncTimeout)
	defer cancel()
	src, err := s.source(ctx, id)
	if err != nil {
		return err
	}
	items, err := s.fetch(ctx, src)
	if err == nil && len(items) == 0 {
		err = errors.New("来源没有返回任何提示词")
	}
	if err == nil {
		err = s.repo.UpsertSourceItems(ctx, src.ID, items)
	}
	if err == nil {
		err = s.repo.RefreshDuplicates(ctx)
	}
	msg := ""
	count := src.ItemCount
	if err != nil {
		msg = truncateRunes(err.Error(), 500)
		slog.Warn("prompt library: sync failed", "source", src.ID, "error", err)
	} else {
		count = len(items)
		slog.Info("prompt library: synced", "source", src.ID, "items", count)
	}
	if recErr := s.repo.RecordSourceSync(context.Background(), src.ID, count, msg); recErr != nil {
		slog.Warn("prompt library: record sync failed", "source", src.ID, "error", recErr)
	}
	return err
}

func (s *PromptLibrarySyncService) fetch(ctx context.Context, src *PromptSource) ([]PromptItem, error) {
	switch src.Format {
	case "youmind":
		return s.fetchYouMind(ctx, src)
	default:
		return s.fetchRegistry(ctx, src)
	}
}

func (s *PromptLibrarySyncService) open(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求来源失败：%w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("来源返回 HTTP %d", resp.StatusCode)
	}
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(resp.Body, promptSyncMaxBody), resp.Body}, nil
}

// decodeJSONArray streams the elements of a top-level JSON array into fn.
func decodeJSONArray[T any](r io.Reader, fn func(index int, value T)) error {
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("解析来源失败：%w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return errors.New("来源格式不正确：需要 JSON 数组")
	}
	for i := 0; dec.More(); i++ {
		var value T
		if err := dec.Decode(&value); err != nil {
			return fmt.Errorf("解析来源第 %d 条失败：%w", i+1, err)
		}
		fn(i, value)
	}
	return nil
}

type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexString(n.String())
		return nil
	}
	*f = ""
	return nil
}

type flexStrings []string

func (f *flexStrings) UnmarshalJSON(b []byte) error {
	var raw []flexString
	if err := json.Unmarshal(b, &raw); err != nil {
		*f = nil
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s := strings.TrimSpace(string(v)); s != "" {
			out = append(out, s)
		}
	}
	*f = out
	return nil
}

type registryPrompt struct {
	ID                 flexString  `json:"id"`
	Title              flexString  `json:"title"`
	Prompt             flexString  `json:"prompt"`
	Description        flexString  `json:"description"`
	CoverURL           flexString  `json:"coverUrl"`
	ReferenceImageURLs flexStrings `json:"referenceImageUrls"`
	Tags               flexStrings `json:"tags"`
	CreatedAt          flexString  `json:"createdAt"`
	UpdatedAt          flexString  `json:"updatedAt"`
	Author             flexString  `json:"author"`
	SourceURL          flexString  `json:"sourceUrl"`
	ImageModel         flexString  `json:"imageModel"`
}

// fetchRegistry reads a JSON array in the yukkcat/image-prompts format, normalized exactly like the
// canvas' local source runtime so item ids (and therefore favorites) stay the same.
func (s *PromptLibrarySyncService) fetchRegistry(ctx context.Context, src *PromptSource) ([]PromptItem, error) {
	body, err := s.open(ctx, src.URL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	seen := map[string]bool{}
	var items []PromptItem
	err = decodeJSONArray(body, func(index int, rec registryPrompt) {
		title := strings.TrimSpace(string(rec.Title))
		prompt := strings.TrimSpace(string(rec.Prompt))
		if title == "" || prompt == "" {
			return
		}
		id := strings.TrimSpace(string(rec.ID))
		if id == "" {
			id = fmt.Sprintf("%s-%04d", src.ID, index+1)
		}
		if seen[id] || len(id) > 160 {
			return
		}
		seen[id] = true
		refs := make([]string, 0, len(rec.ReferenceImageURLs))
		for _, u := range rec.ReferenceImageURLs {
			if abs := absolutePromptURL(src.URL, u); abs != "" {
				refs = append(refs, abs)
			}
		}
		cover := absolutePromptURL(src.URL, string(rec.CoverURL))
		if cover == "" && len(refs) > 0 {
			cover = refs[0]
		}
		published := parsePromptTime(string(rec.CreatedAt))
		if published == nil {
			published = parsePromptTime(string(rec.UpdatedAt))
		}
		items = append(items, buildSourcePromptItem(sourcePromptFields{
			externalID: id, title: title, prompt: prompt, description: string(rec.Description), cover: cover, refs: refs,
			tags: []string(rec.Tags), author: string(rec.Author), sourceURL: absolutePromptURL(src.URL, string(rec.SourceURL)),
			modelHint: string(rec.ImageModel), published: published,
		}))
	})
	return items, err
}

type youmindManifest struct {
	Categories []struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
		File  string `json:"file"`
	} `json:"categories"`
}

type youmindPrompt struct {
	ID                 flexString  `json:"id"`
	Content            flexString  `json:"content"`
	Title              flexString  `json:"title"`
	Description        flexString  `json:"description"`
	SourceMedia        flexStrings `json:"sourceMedia"`
	NeedReferenceImage bool        `json:"needReferenceImages"`
}

// fetchYouMind reads the YouMind references manifest and every use-case file it lists. A prompt
// listed under several use cases becomes one item with several scene hints.
func (s *PromptLibrarySyncService) fetchYouMind(ctx context.Context, src *PromptSource) ([]PromptItem, error) {
	body, err := s.open(ctx, src.URL)
	if err != nil {
		return nil, err
	}
	var manifest youmindManifest
	err = json.NewDecoder(body).Decode(&manifest)
	_ = body.Close()
	if err != nil {
		return nil, fmt.Errorf("解析来源清单失败：%w", err)
	}
	type entry struct {
		rec        youmindPrompt
		hints      []string
		categories []string
	}
	byID := map[string]*entry{}
	var order []string
	for _, cat := range manifest.Categories {
		file := strings.TrimSpace(cat.File)
		if file == "" || strings.Contains(file, "..") || strings.Contains(file, "/") {
			continue
		}
		fileURL := absolutePromptURL(src.URL, file)
		r, err := s.open(ctx, fileURL)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", file, err)
		}
		hint := youmindCategoryScenes[cat.Slug]
		err = decodeJSONArray(r, func(_ int, rec youmindPrompt) {
			id := strings.TrimSpace(string(rec.ID))
			if id == "" || strings.TrimSpace(string(rec.Content)) == "" {
				return
			}
			e, ok := byID[id]
			if !ok {
				e = &entry{rec: rec}
				byID[id] = e
				order = append(order, id)
			}
			if hint != "" {
				e.hints = append(e.hints, hint)
			}
			if cat.Title != "" {
				e.categories = append(e.categories, cat.Title)
			}
		})
		_ = r.Close()
		if err != nil {
			return nil, fmt.Errorf("%s：%w", file, err)
		}
	}
	items := make([]PromptItem, 0, len(order))
	for _, id := range order {
		e := byID[id]
		media := []string(e.rec.SourceMedia)
		cover := ""
		if len(media) > 0 {
			cover = media[0]
		}
		item := buildSourcePromptItem(sourcePromptFields{
			externalID: id, title: string(e.rec.Title), prompt: string(e.rec.Content), description: string(e.rec.Description),
			cover: cover, refs: media, labels: uniquePromptStrings(e.categories, 0), modelHint: string(e.rec.Description),
			sceneHints: e.hints, needsReference: e.rec.NeedReferenceImage,
		})
		items = append(items, item)
	}
	return items, nil
}

type sourcePromptFields struct {
	externalID, title, prompt, description, cover, author, sourceURL, modelHint string
	refs, tags, sceneHints                                                      []string
	// labels are stored as source tags but not classified (the scene hints already cover them).
	labels         []string
	published      *time.Time
	needsReference bool
}

func buildSourcePromptItem(f sourcePromptFields) PromptItem {
	prompt := strings.TrimSpace(f.prompt)
	tags := uniquePromptStrings(f.tags, 20)
	sourceTags := uniquePromptStrings(append(append([]string{}, tags...), f.labels...), 20)
	traits := ClassifyPrompt(PromptTraitInput{Title: f.title, Prompt: prompt, Description: f.description, Tags: tags, ModelHint: f.modelHint, SceneHints: f.sceneHints})
	title := PromptDisplayTitle(f.title, prompt, tags)
	kind := "image"
	if traits.Scenes[0] == "video" {
		kind = "video"
	}
	flags := promptAutoFlags(traits)
	status := PromptStatusActive
	if len(flags) > 0 {
		status = PromptStatusHidden
	}
	refs := f.refs
	if refs == nil {
		refs = []string{}
	}
	if len(refs) > 8 {
		refs = refs[:8]
	}
	item := PromptItem{
		ExternalID: f.externalID, Kind: kind, Title: truncateRunes(title, 200), Prompt: prompt,
		Description: truncateRunes(strings.TrimSpace(f.description), 2000), CoverURL: f.cover, ReferenceImageURLs: refs,
		SourceTags: sourceTags, Scenes: traits.Scenes, Tags: []string{}, Model: traits.Model, Lang: traits.Lang,
		NeedsReference: traits.NeedsReference || f.needsReference, AutoFlags: flags,
		Author: truncateRunes(strings.TrimSpace(f.author), 200), SourceURL: f.sourceURL,
		Visibility: PromptVisibilityPublic, Status: status, QualityScore: PromptQualityScore(f.cover, title, traits),
		DedupeKey: PromptDedupeKey(prompt), PublishedAt: f.published,
	}
	item.SyncHash = promptSyncHash(item)
	return item
}

func promptSyncHash(item PromptItem) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	refs := append([]string(nil), item.ReferenceImageURLs...)
	tags := append([]string(nil), item.SourceTags...)
	sort.Strings(tags)
	_ = enc.Encode([]any{item.Kind, item.Title, item.Prompt, item.Description, item.CoverURL, refs, tags, item.Scenes, item.Model, item.Lang,
		item.NeedsReference, item.AutoFlags, item.Author, item.SourceURL, item.Status, item.QualityScore, item.PublishedAt})
	return hex.EncodeToString(h.Sum(nil))
}

func absolutePromptURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	abs := b.ResolveReference(r)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return ""
	}
	return abs.String()
}

func parsePromptTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil && n > 0 {
		if n > 1e12 {
			n /= 1000
		}
		t := time.Unix(n, 0).UTC()
		return &t
	}
	return nil
}
