package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// Admin settings for the image tools and the usage records shown to users and admins.

var (
	ErrImageToolsInvalidPrice     = infraerrors.BadRequest("IMAGE_TOOLS_INVALID_PRICE", "价格需要在 0 到 100 之间（Invalid price）")
	ErrImageToolsInvalidFreeDaily = infraerrors.BadRequest("IMAGE_TOOLS_INVALID_FREE_DAILY", "每天免费次数需要在 0 到 1000 之间（Invalid free allowance）")
	ErrImageToolsSettingsReadOnly = infraerrors.ServiceUnavailable("IMAGE_TOOLS_SETTINGS_UNAVAILABLE", "设置暂时无法保存，请稍后再试（Settings are unavailable）")
)

type ImageToolsSettings struct {
	Enabled       bool    `json:"enabled"`
	PriceRemoveBg float64 `json:"price_remove_bg"`
	PriceUpscale  float64 `json:"price_upscale"`
	FreeDaily     int     `json:"free_daily"`
	// ServiceConfigured is false when IMAGE_TOOLS_BASE_URL is empty (the switch then has no effect).
	ServiceConfigured bool `json:"service_configured"`
}

type ImageToolsSettingsInput struct {
	Enabled       bool    `json:"enabled"`
	PriceRemoveBg float64 `json:"price_remove_bg"`
	PriceUpscale  float64 `json:"price_upscale"`
	FreeDaily     int     `json:"free_daily"`
}

// ImageToolUseRecord is one recorded run.
type ImageToolUseRecord struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	UserEmail   string    `json:"user_email,omitempty"`
	APIKeyID    *int64    `json:"api_key_id,omitempty"`
	APIKeyName  string    `json:"api_key_name"`
	Tool        string    `json:"tool"`
	Free        bool      `json:"free"`
	Cost        float64   `json:"cost"`
	InputBytes  int       `json:"input_bytes"`
	OutputBytes int       `json:"output_bytes"`
	DurationMs  int       `json:"duration_ms"`
	CreatedAt   time.Time `json:"created_at"`
}

type ImageToolUseQuery struct {
	UserID   int64
	Keyword  string // admin: user email contains
	Tool     string
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

// ImageToolStat sums runs by tool over a period.
type ImageToolStat struct {
	Tool     string  `json:"tool"`
	Runs     int64   `json:"runs"`
	FreeRuns int64   `json:"free_runs"`
	Cost     float64 `json:"cost"`
	Users    int64   `json:"users"`
}

type ImageToolUseList struct {
	Items    []ImageToolUseRecord `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type ImageToolsStats struct {
	Today []ImageToolStat `json:"today"`
	Week  []ImageToolStat `json:"week"`
	Month []ImageToolStat `json:"month"`
}

// ImageToolsUserSummary heads the user's own record page.
type ImageToolsUserSummary struct {
	Quota *ImageToolsQuota `json:"quota"`
	Month []ImageToolStat  `json:"month"`
}

func (s *ImageToolsService) Settings(ctx context.Context) ImageToolsSettings {
	c := s.config(ctx)
	return ImageToolsSettings{Enabled: !c.Disabled, PriceRemoveBg: c.PriceRemoveBg, PriceUpscale: c.PriceUpscale, FreeDaily: c.FreeDaily, ServiceConfigured: c.BaseURL != ""}
}

func (s *ImageToolsService) SaveSettings(ctx context.Context, in ImageToolsSettingsInput) (ImageToolsSettings, error) {
	if s.settings == nil {
		return ImageToolsSettings{}, ErrImageToolsSettingsReadOnly
	}
	for _, price := range []float64{in.PriceRemoveBg, in.PriceUpscale} {
		if price < 0 || price > imageToolMaxPrice {
			return ImageToolsSettings{}, ErrImageToolsInvalidPrice
		}
	}
	if in.FreeDaily < 0 || in.FreeDaily > imageToolMaxFreeDaily {
		return ImageToolsSettings{}, ErrImageToolsInvalidFreeDaily
	}
	err := s.settings.SetMultiple(ctx, map[string]string{
		settingImageToolsEnabled:       strconv.FormatBool(in.Enabled),
		settingImageToolsPriceRemoveBg: strconv.FormatFloat(in.PriceRemoveBg, 'f', -1, 64),
		settingImageToolsPriceUpscale:  strconv.FormatFloat(in.PriceUpscale, 'f', -1, 64),
		settingImageToolsFreeDaily:     strconv.Itoa(in.FreeDaily),
	})
	if err != nil {
		return ImageToolsSettings{}, err
	}
	s.cacheMu.Lock()
	s.cached = nil
	s.cacheMu.Unlock()
	return s.Settings(ctx), nil
}

func normalizeImageToolUseQuery(q ImageToolUseQuery) ImageToolUseQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	if q.Tool != ImageToolRemoveBg && q.Tool != ImageToolUpscale {
		q.Tool = ""
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	return q
}

// ListUses lists runs: the user's own when q.UserID is set, everyone's for admins.
func (s *ImageToolsService) ListUses(ctx context.Context, q ImageToolUseQuery) (*ImageToolUseList, error) {
	q = normalizeImageToolUseQuery(q)
	items, total, err := s.repo.ListUses(ctx, q)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ImageToolUseRecord{}
	}
	return &ImageToolUseList{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// UserSummary is today's quota plus this month's runs and spending.
func (s *ImageToolsService) UserSummary(ctx context.Context, userID int64) (*ImageToolsUserSummary, error) {
	quota, err := s.Quota(ctx, userID)
	if err != nil {
		return nil, err
	}
	month, err := s.repo.Stats(ctx, userID, timezone.StartOfMonth(s.now()))
	if err != nil {
		return nil, err
	}
	return &ImageToolsUserSummary{Quota: quota, Month: month}, nil
}

func (s *ImageToolsService) Stats(ctx context.Context) (*ImageToolsStats, error) {
	now := s.now()
	today, err := s.repo.Stats(ctx, 0, timezone.StartOfDay(now))
	if err != nil {
		return nil, err
	}
	week, err := s.repo.Stats(ctx, 0, timezone.StartOfDay(now).AddDate(0, 0, -6))
	if err != nil {
		return nil, err
	}
	month, err := s.repo.Stats(ctx, 0, timezone.StartOfMonth(now))
	if err != nil {
		return nil, err
	}
	return &ImageToolsStats{Today: today, Week: week, Month: month}, nil
}
