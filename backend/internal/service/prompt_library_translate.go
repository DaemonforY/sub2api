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
	Untranslated   int64      `json:"untranslated"`
	Running        bool       `json:"running"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastTranslated int        `json:"last_translated"`
	LastError      string     `json:"last_error"`
}

type PromptTranslateConfigInput struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	// APIKey replaces the stored key when non-empty; ClearAPIKey removes it.
	APIKey      string `json:"api_key"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

// PromptTitleTranslator fills title_zh from the bundled dictionary and the configured model.
type PromptTitleTranslator struct {
	repo     PromptLibraryRepository
	settings SettingRepository
	client   *http.Client

	mu        sync.Mutex
	running   bool
	lastRunAt *time.Time
	lastCount int
	lastError string
}

func NewPromptTitleTranslator(repo PromptLibraryRepository, settings SettingRepository) *PromptTitleTranslator {
	return &PromptTitleTranslator{repo: repo, settings: settings, client: &http.Client{Timeout: 2 * time.Minute}}
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
	t.mu.Lock()
	defer t.mu.Unlock()
	return &PromptTranslateStatus{PromptTranslateConfig: cfg.PromptTranslateConfig, Untranslated: left, Running: t.running,
		LastRunAt: t.lastRunAt, LastTranslated: t.lastCount, LastError: t.lastError}, nil
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
	return t.settings.SetMultiple(ctx, values)
}

// AfterSync applies the bundled dictionary and, when a model is configured, translates a few
// hundred new titles. Called by the sync job; errors are logged, not returned.
func (t *PromptTitleTranslator) AfterSync(ctx context.Context) {
	if n, err := t.repo.ApplyTitleTranslations(ctx, BundledPromptTitleTranslations()); err != nil {
		slog.Warn("prompt library: apply bundled titles failed", "error", err)
	} else if n > 0 {
		slog.Info("prompt library: applied bundled titles", "items", n)
	}
	cfg, err := t.config(ctx)
	if err != nil || !cfg.ready() {
		return
	}
	_, _ = t.run(ctx, cfg, promptTranslateAfterSync)
}

// RunAsync translates every remaining English title in the background (admin button).
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
	now := time.Now()
	t.mu.Lock()
	t.running = false
	t.lastRunAt = &now
	t.lastCount = translated
	t.lastError = ""
	if err != nil {
		t.lastError = truncateRunes(err.Error(), 300)
		slog.Warn("prompt library: title translation failed", "error", err, "translated", translated)
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
		result, err := t.callModel(ctx, cfg, titles[start:end])
		if err != nil {
			failures++
			// A few bad batches are tolerated; a broken configuration stops early.
			if failures >= 3 && translated == 0 {
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

func (t *PromptTitleTranslator) callModel(ctx context.Context, cfg promptTranslateSecrets, titles []string) (map[string]string, error) {
	input, _ := json.Marshal(titles)
	body, _ := json.Marshal(map[string]any{
		"model":       cfg.Model,
		"temperature": 0.2,
		"messages": []map[string]string{
			{"role": "system", "content": promptTranslateInstruction},
			{"role": "user", "content": string(input)},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用翻译模型失败：%w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("翻译模型返回 HTTP %d：%s", resp.StatusCode, truncateRunes(strings.TrimSpace(string(raw)), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return nil, errors.New("翻译模型的返回格式不正确（需要 OpenAI Chat Completions 格式）")
	}
	return parseTitleTranslations(parsed.Choices[0].Message.Content, titles), nil
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
