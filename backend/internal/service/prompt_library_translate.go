package service

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Chinese titles for library entries whose source title is English. A dictionary of the titles
// known at release time ships with the binary; titles that appear later are translated by an
// OpenAI-compatible model the admin configures (e.g. this site's own gateway with an admin key).

//go:embed prompt_title_zh.json.gz
var bundledPromptTitleZh []byte

var (
	bundledTitlesOnce sync.Once
	bundledTitles     map[string]string
)

// BundledPromptTitleTranslations returns the dictionary shipped with the binary (English → Chinese).
func BundledPromptTitleTranslations() map[string]string {
	bundledTitlesOnce.Do(func() {
		bundledTitles = map[string]string{}
		zr, err := gzip.NewReader(bytes.NewReader(bundledPromptTitleZh))
		if err != nil {
			slog.Warn("prompt library: bundled titles unreadable", "error", err)
			return
		}
		defer func() { _ = zr.Close() }()
		if err := json.NewDecoder(zr).Decode(&bundledTitles); err != nil {
			slog.Warn("prompt library: bundled titles unreadable", "error", err)
		}
	})
	return bundledTitles
}

const (
	settingPromptTranslateBaseURL = "prompt_library_translate_base_url"
	settingPromptTranslateModel   = "prompt_library_translate_model"
	settingPromptTranslateAPIKey  = "prompt_library_translate_api_key"

	promptTranslateBatch      = 40
	promptTranslateAfterSync  = 500
	promptTranslateManualMax  = 20000
	promptTranslateTitleRunes = 30

	promptSceneBatch = 20

	// After a failed model pass the hourly sync waits before trying again: 2h, 4h, 8h … up to a day.
	promptTranslateBackoffBase = 2 * time.Hour
	promptTranslateBackoffMax  = 24 * time.Hour
)

var (
	ErrPromptTranslateNotConfigured = infraerrors.BadRequest("PROMPT_TRANSLATE_NOT_CONFIGURED", "还没有配置翻译模型：请先填写接口地址、模型和 API Key")
	ErrPromptTranslateInvalidURL    = infraerrors.BadRequest("PROMPT_TRANSLATE_INVALID_URL", "接口地址需要以 http:// 或 https:// 开头，例如 http://127.0.0.1:8080/v1")
)

type PromptTranslateConfig struct {
	BaseURL          string `json:"base_url"`
	Model            string `json:"model"`
	APIKeyConfigured bool   `json:"api_key_configured"`
}

type PromptTranslateStatus struct {
	PromptTranslateConfig
	Untranslated    int64      `json:"untranslated"`
	UncheckedScenes int64      `json:"unchecked_scenes"`
	Running         bool       `json:"running"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastTranslated  int        `json:"last_translated"`
	LastScenes      int        `json:"last_scenes"`
	LastError       string     `json:"last_error"`
	// NextAutoRunAt: after failures the hourly sync skips the model until then (the button still runs).
	NextAutoRunAt *time.Time `json:"next_auto_run_at,omitempty"`
}

// PromptSceneCandidate is a source item whose automatic scenes a model should check.
type PromptSceneCandidate struct {
	ID          int64
	Title       string
	Description string
	Prompt      string
	Scenes      []string
}

type PromptTranslateConfigInput struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	// APIKey replaces the stored key when non-empty; ClearAPIKey removes it.
	APIKey      string `json:"api_key"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

// PromptTitleTranslator fills title_zh from the bundled dictionary and the configured model, and
// has the same model check the automatic scenes of items that arrived after the bundled corrections.
type PromptTitleTranslator struct {
	repo     PromptLibraryRepository
	settings SettingRepository
	client   *http.Client

	mu         sync.Mutex
	running    bool
	lastRunAt  *time.Time
	lastCount  int
	lastScenes int
	lastError  string
	// Failed model passes in a row, and when the hourly sync may try the model again.
	failStreak int
	nextAuto   time.Time
	now        func() time.Time
}

func NewPromptTitleTranslator(repo PromptLibraryRepository, settings SettingRepository) *PromptTitleTranslator {
	return &PromptTitleTranslator{repo: repo, settings: settings, client: &http.Client{Timeout: 2 * time.Minute}, now: time.Now}
}

type promptTranslateSecrets struct {
	PromptTranslateConfig
	apiKey string
}

func (t *PromptTitleTranslator) config(ctx context.Context) (promptTranslateSecrets, error) {
	var cfg promptTranslateSecrets
	if t.settings == nil {
		return cfg, nil
	}
	values, err := t.settings.GetMultiple(ctx, []string{settingPromptTranslateBaseURL, settingPromptTranslateModel, settingPromptTranslateAPIKey})
	if err != nil {
		return cfg, err
	}
	cfg.BaseURL = values[settingPromptTranslateBaseURL]
	cfg.Model = values[settingPromptTranslateModel]
	cfg.apiKey = values[settingPromptTranslateAPIKey]
	cfg.APIKeyConfigured = cfg.apiKey != ""
	return cfg, nil
}

func (c promptTranslateSecrets) ready() bool {
	return c.BaseURL != "" && c.Model != "" && c.apiKey != ""
}

// Status reports the configuration (never the key) and how much is left to translate.
func (t *PromptTitleTranslator) Status(ctx context.Context) (*PromptTranslateStatus, error) {
	cfg, err := t.config(ctx)
	if err != nil {
		return nil, err
	}
	left, err := t.repo.CountUntranslated(ctx)
	if err != nil {
		return nil, err
	}
	unchecked, err := t.repo.CountUncheckedScenes(ctx)
	if err != nil {
		return nil, err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := &PromptTranslateStatus{PromptTranslateConfig: cfg.PromptTranslateConfig, Untranslated: left, UncheckedScenes: unchecked, Running: t.running,
		LastRunAt: t.lastRunAt, LastTranslated: t.lastCount, LastScenes: t.lastScenes, LastError: t.lastError}
	if !t.nextAuto.IsZero() && t.nextAuto.After(t.now()) {
		next := t.nextAuto
		out.NextAutoRunAt = &next
	}
	return out, nil
}

func (t *PromptTitleTranslator) SaveConfig(ctx context.Context, in PromptTranslateConfigInput) error {
	base := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if base != "" && !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return ErrPromptTranslateInvalidURL
	}
	values := map[string]string{settingPromptTranslateBaseURL: base, settingPromptTranslateModel: strings.TrimSpace(in.Model)}
	if key := strings.TrimSpace(in.APIKey); key != "" {
		values[settingPromptTranslateAPIKey] = key
	} else if in.ClearAPIKey {
		values[settingPromptTranslateAPIKey] = ""
	}
	if err := t.settings.SetMultiple(ctx, values); err != nil {
		return err
	}
	// A new configuration gets tried at the next sync.
	t.mu.Lock()
	t.failStreak, t.nextAuto = 0, time.Time{}
	t.mu.Unlock()
	return nil
}

// AfterSync applies the bundled corrections and, when a model is configured, translates the titles
// and checks the scenes of a few hundred new items. Called by the sync job; errors are logged, not returned.
func (t *PromptTitleTranslator) AfterSync(ctx context.Context) {
	if n, err := t.repo.ApplySceneOverrides(ctx, BundledPromptScenes()); err != nil {
		slog.Warn("prompt library: apply bundled scenes failed", "error", err)
	} else if n > 0 {
		slog.Info("prompt library: applied bundled scenes", "items", n)
	}
	if n, err := t.repo.ApplyTitleTranslations(ctx, BundledPromptTitleTranslations()); err != nil {
		slog.Warn("prompt library: apply bundled titles failed", "error", err)
	} else if n > 0 {
		slog.Info("prompt library: applied bundled titles", "items", n)
	}
	cfg, err := t.config(ctx)
	if err != nil || !cfg.ready() {
		return
	}
	t.autoRun(ctx, cfg)
}

// autoRun is the hourly model pass, skipped while backing off after failures: a broken
// configuration (wrong model, key or quota) would otherwise fail — and log errors — every hour.
func (t *PromptTitleTranslator) autoRun(ctx context.Context, cfg promptTranslateSecrets) bool {
	t.mu.Lock()
	wait := t.nextAuto
	t.mu.Unlock()
	if !wait.IsZero() && t.now().Before(wait) {
		slog.Debug("prompt library: model pass skipped after failures", "until", wait)
		return false
	}
	_, _ = t.run(ctx, cfg, promptTranslateAfterSync)
	return true
}

// RunAsync translates every remaining English title and checks every remaining scene in the background (admin button).
func (t *PromptTitleTranslator) RunAsync(ctx context.Context) error {
	cfg, err := t.config(ctx)
	if err != nil {
		return err
	}
	if !cfg.ready() {
		return ErrPromptTranslateNotConfigured
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		_, _ = t.run(bg, cfg, promptTranslateManualMax)
	}()
	return nil
}

func (t *PromptTitleTranslator) run(ctx context.Context, cfg promptTranslateSecrets, max int) (int, error) {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return 0, nil
	}
	t.running = true
	t.mu.Unlock()
	translated, err := t.translate(ctx, cfg, max)
	if err != nil {
		slog.Warn("prompt library: title translation failed", "error", err, "translated", translated)
	}
	checked, sceneErr := 0, error(nil)
	if !isPromptModelConfigError(err) {
		checked, sceneErr = t.checkScenes(ctx, cfg, max)
	}
	if sceneErr != nil {
		slog.Warn("prompt library: scene check failed", "error", sceneErr, "checked", checked)
		if err == nil {
			err = sceneErr
		}
	}
	if translated > 0 || checked > 0 {
		slog.Info("prompt library: model pass done", "titles", translated, "scenes", checked)
	}
	now := t.now()
	t.mu.Lock()
	t.running = false
	t.lastRunAt = &now
	t.lastCount = translated
	t.lastScenes = checked
	t.lastError = ""
	if err != nil {
		t.lastError = truncateRunes(err.Error(), 300)
		t.failStreak++
		wait := promptTranslateBackoffBase << min(t.failStreak-1, 8)
		t.nextAuto = now.Add(min(wait, promptTranslateBackoffMax))
	} else {
		t.failStreak, t.nextAuto = 0, time.Time{}
	}
	t.mu.Unlock()
	return translated, err
}

func (t *PromptTitleTranslator) translate(ctx context.Context, cfg promptTranslateSecrets, max int) (int, error) {
	titles, err := t.repo.UntranslatedTitles(ctx, max)
	if err != nil {
		return 0, err
	}
	translated := 0
	failures := 0
	for start := 0; start < len(titles); start += promptTranslateBatch {
		end := min(start+promptTranslateBatch, len(titles))
		content, err := t.callModel(ctx, cfg, promptTranslateInstruction, titles[start:end])
		var result map[string]string
		if err == nil {
			result = parseTitleTranslations(content, titles[start:end])
		}
		if err != nil {
			failures++
			// A few bad batches are tolerated; a broken configuration (or a rate limit) stops at once.
			if isPromptModelConfigError(err) || (failures >= 3 && translated == 0) {
				return translated, err
			}
			continue
		}
		n, err := t.repo.ApplyTitleTranslations(ctx, result)
		if err != nil {
			return translated, err
		}
		translated += int(n)
	}
	return translated, nil
}

const promptTranslateInstruction = `You translate titles of AI image prompts for a Chinese prompt library.
Input: a JSON array of titles (mostly English). Output: ONLY a JSON object mapping each input title, unchanged, to a concise natural Simplified Chinese title (at most 20 Chinese characters).
Keep brand, product and franchise names recognizable (iPhone, LEGO, Pokémon, YouTube...). Use common design terms: poster 海报, infographic 信息图, mockup 样机, thumbnail 缩略图, isometric 等距, chibi Q版, cinematic 电影感.
If a title is meaningless (punctuation, timestamps, template fragments), map it to "".`

// callModel sends one Chat Completions request and returns the reply text.
func (t *PromptTitleTranslator) callModel(ctx context.Context, cfg promptTranslateSecrets, instruction string, payload any) (string, error) {
	return openAIChat(ctx, t.client, cfg.BaseURL, cfg.apiKey, cfg.Model, instruction, payload)
}

// modelHTTPError is a non-200 reply from a helper model.
type modelHTTPError struct {
	status int
	body   string
}

func (e *modelHTTPError) Error() string {
	return fmt.Sprintf("模型返回 HTTP %d：%s", e.status, e.body)
}

// isPromptModelConfigError: replies that more requests in the same pass won't fix — a bad request or
// model (400 / 404), a bad key (401 / 403) or a rate limit (429).
func isPromptModelConfigError(err error) bool {
	var he *modelHTTPError
	if !errors.As(err, &he) {
		return false
	}
	switch he.status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests:
		return true
	}
	return false
}

// openAIChat sends one OpenAI-compatible Chat Completions request (payload JSON-encoded as the user
// message) and returns the reply text. Used by the admin-configured helper models.
func openAIChat(ctx context.Context, client *http.Client, baseURL, apiKey, model, instruction string, payload any) (string, error) {
	input, _ := json.Marshal(payload)
	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"temperature": 0.2,
		"messages": []map[string]string{
			{"role": "system", "content": instruction},
			{"role": "user", "content": string(input)},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用模型失败：%w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return "", &modelHTTPError{status: resp.StatusCode, body: truncateRunes(strings.TrimSpace(string(raw)), 200)}
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", errors.New("模型的返回格式不正确（需要 OpenAI Chat Completions 格式）")
	}
	return parsed.Choices[0].Message.Content, nil
}

// parseTitleTranslations reads the model's JSON object (tolerating code fences and chatter) and
// keeps only usable Chinese titles for titles that were asked for.
func parseTitleTranslations(content string, asked []string) map[string]string {
	out := map[string]string{}
	startIdx := strings.Index(content, "{")
	endIdx := strings.LastIndex(content, "}")
	if startIdx < 0 || endIdx <= startIdx {
		return out
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(content[startIdx:endIdx+1]), &got); err != nil {
		return out
	}
	want := make(map[string]bool, len(asked))
	for _, title := range asked {
		want[title] = true
	}
	for en, zh := range got {
		zh = strings.TrimSpace(zh)
		if !want[en] || zh == "" || !promptCJK.MatchString(zh) || utf8.RuneCountInString(zh) > promptTranslateTitleRunes {
			continue
		}
		out[en] = zh
	}
	return out
}

// Scene corrections made with a model for the entries known at release time (key
// "<source_id>:<external_id>" → scenes, main scene first). Applied after every sync to items an
// admin has not edited; entries that arrive later keep the automatic classification.
//
//go:embed prompt_scenes.json.gz
var bundledPromptScenes []byte

var (
	bundledScenesOnce sync.Once
	bundledScenes     map[string][]string
)

func BundledPromptScenes() map[string][]string {
	bundledScenesOnce.Do(func() {
		raw := map[string][]string{}
		zr, err := gzip.NewReader(bytes.NewReader(bundledPromptScenes))
		if err == nil {
			err = json.NewDecoder(zr).Decode(&raw)
			_ = zr.Close()
		}
		if err != nil {
			slog.Warn("prompt library: bundled scenes unreadable", "error", err)
		}
		bundledScenes = make(map[string][]string, len(raw))
		for key, scenes := range raw {
			clean, err := normalizePromptScenes(scenes)
			if err == nil && len(clean) > 0 {
				bundledScenes[key] = clean
			}
		}
	})
	return bundledScenes
}

func (t *PromptTitleTranslator) checkScenes(ctx context.Context, cfg promptTranslateSecrets, max int) (int, error) {
	items, err := t.repo.UncheckedScenePrompts(ctx, max)
	if err != nil {
		return 0, err
	}
	checked := 0
	failures := 0
	for start := 0; start < len(items); start += promptSceneBatch {
		batch := items[start:min(start+promptSceneBatch, len(items))]
		input := make([]map[string]any, len(batch))
		for i, item := range batch {
			input[i] = map[string]any{"k": strconv.FormatInt(item.ID, 10), "t": item.Title, "d": item.Description, "p": item.Prompt, "s": item.Scenes}
		}
		content, err := t.callModel(ctx, cfg, promptSceneInstruction, input)
		if err != nil {
			failures++
			if isPromptModelConfigError(err) || (failures >= 3 && checked == 0) {
				return checked, err
			}
			continue
		}
		n, err := t.repo.ApplyCheckedScenes(ctx, parseSceneLabels(content, batch))
		if err != nil {
			return checked, err
		}
		checked += int(n)
	}
	return checked, nil
}

// promptSceneInstruction is the guide the bundled corrections were made with.
const promptSceneInstruction = `You label prompts of a Chinese AI-image prompt library with scenes.
Input: a JSON array of {"k": key, "t": title, "d": description (cut), "p": start of the prompt (cut), "s": current automatic scenes (often wrong)}.
Assign each item 1-3 scene ids, the FIRST being the main one. Allowed ids only:
poster: posters, flyers, covers, banners, YouTube/video thumbnails, typography / lettering layouts, event & movie posters, magazine covers
ecommerce: product shots, e-commerce main images, ads for a product, packaging, product mockups, exploded product views
ui: app / web / dashboard UI, landing pages, game HUD or screenshots, phone screen mockups
infographic: infographics, slides / PPT, diagrams, charts, tutorials, educational explainers, notes, maps with labels, recipe cards, comparison sheets
portrait: a person (or the user's own photo) as the subject: portraits, avatars, profile pictures, ID photos, selfies, fashion/outfit shots of a person, character-from-photo transforms
photo: realistic photography where the photo style itself matters and it is not mainly a person portrait: street, landscape, food, animal, macro, film/analog look, cinematic stills
illustration: illustration, anime, manga / comics, storyboards, children's book art, watercolor / ink painting, cards, stickers drawn by hand
3d: 3D renders, figurines / action figures, miniatures, dioramas, isometric scenes, clay / plush / LEGO / toy looks, blind boxes, sculptures
brand: logos, brand identity / VI systems, mascots for a brand, brand guidelines, merchandise design for a brand
social: memes, emoji / sticker packs, social-media post formats (Instagram grid, 朋友圈九宫格, Xiaohongshu cover), reaction images, funny social content
life: interiors, architecture, home & room design, travel scenes, food scenes, everyday life spaces
history: Chinese traditional / 国风 / ancient costume, historical scenes, classical art styles, dynasties, ink-wash heritage
creative: surreal / conceptual / playful ideas, visual tricks, mashups, game assets & concept art, sci-fi or fantasy worlds
video: prompts for generating video (shots, camera moves, seconds)
other: only if nothing above fits
Judge from what the prompt actually makes, not from "s". Use 1 scene when one clearly fits; add a 2nd/3rd only when they genuinely apply (a 3D figurine of the user's photo: ["3d","portrait"]; a movie poster with a realistic actor: ["poster","portrait"]). Avoid "social" unless it is really memes/stickers/social-post formats. Avoid "other".
Output ONLY a JSON object {key: [scene ids]} covering every input key.`

// parseSceneLabels reads the model's JSON object and keeps valid scenes for items that were asked for.
func parseSceneLabels(content string, asked []PromptSceneCandidate) map[int64][]string {
	out := map[int64][]string{}
	startIdx := strings.Index(content, "{")
	endIdx := strings.LastIndex(content, "}")
	if startIdx < 0 || endIdx <= startIdx {
		return out
	}
	var got map[string][]string
	if err := json.Unmarshal([]byte(content[startIdx:endIdx+1]), &got); err != nil {
		return out
	}
	want := make(map[int64]bool, len(asked))
	for _, item := range asked {
		want[item.ID] = true
	}
	for key, scenes := range got {
		id, err := strconv.ParseInt(strings.TrimSpace(key), 10, 64)
		if err != nil || !want[id] {
			continue
		}
		clean := make([]string, 0, len(scenes))
		for _, scene := range scenes {
			if scene = strings.ToLower(strings.TrimSpace(scene)); IsPromptScene(scene) {
				clean = append(clean, scene)
			}
		}
		if clean = uniquePromptStrings(clean, promptMaxScenes); len(clean) > 0 {
			out[id] = clean
		}
	}
	return out
}
