package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// Server-side image tools for the canvas: background removal and super-resolution, run by the
// internal imagetools container (deploy/imagetools). Subscribers get a few free runs a day; other
// runs are charged to the balance after they succeed, so failures never cost anything.

const (
	ImageToolRemoveBg = "remove_bg"
	ImageToolUpscale  = "upscale"

	// ImageToolMaxInputBytes matches the canvas' upload limit for references.
	ImageToolMaxInputBytes  = 20 << 20
	imageToolMaxOutputBytes = 64 << 20
	imageToolMaxWaiting     = 6
)

var (
	ErrImageToolsDisabled = infraerrors.ServiceUnavailable("IMAGE_TOOLS_DISABLED", "图片处理服务暂未开放，请稍后再试（Image tools are not available）")
	ErrImageToolsBusy     = infraerrors.TooManyRequests("IMAGE_TOOLS_BUSY", "排队处理的图片太多了，请等一分钟再试；本次未扣费（Image tools are busy）")
	ErrImageToolsFailed   = infraerrors.ServiceUnavailable("IMAGE_TOOLS_FAILED", "图片处理失败，请稍后重试；本次未扣费（Image processing failed）")
	ErrImageToolsTooLarge = infraerrors.New(http.StatusRequestEntityTooLarge, "IMAGE_TOOLS_TOO_LARGE", "图片不能超过 20MB：请先在「图片工具」里压缩后再试（Image too large）")
	ErrImageToolsNoImage  = infraerrors.BadRequest("IMAGE_TOOLS_NO_IMAGE", "请上传一张图片（No image uploaded）")
	ErrImageToolsBadTool  = infraerrors.BadRequest("IMAGE_TOOLS_BAD_OPTION", "不支持的处理参数（Unsupported option）")
)

// ImageToolsConfig holds prices (in balance units) and the free allowance.
type ImageToolsConfig struct {
	BaseURL       string
	PriceRemoveBg float64
	PriceUpscale  float64
	FreeDaily     int
	// Disabled is the admin's off switch (the tools also need a BaseURL).
	Disabled bool
}

const (
	settingImageToolsEnabled       = "image_tools_enabled"
	settingImageToolsPriceRemoveBg = "image_tools_price_remove_bg"
	settingImageToolsPriceUpscale  = "image_tools_price_upscale"
	settingImageToolsFreeDaily     = "image_tools_free_daily"
	imageToolSettingsTTL           = 30 * time.Second
	imageToolMaxPrice              = 100
	imageToolMaxFreeDaily          = 1000
)

type imageToolSettings interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
}

type ImageToolsQuota struct {
	Enabled    bool `json:"enabled"`
	Subscribed bool `json:"subscribed"`
	// FreeDaily is what subscribers get; FreeLeft is what this user has left today (0 without a subscription).
	FreeDaily int                `json:"free_daily"`
	FreeUsed  int                `json:"free_used"`
	FreeLeft  int                `json:"free_left"`
	Balance   float64            `json:"balance"`
	Prices    map[string]float64 `json:"prices"`
}

// ImageToolUse is one successful run, as recorded and charged.
type ImageToolUse struct {
	UserID      int64
	APIKeyID    int64
	Tool        string
	Price       float64
	Subscribed  bool
	InputBytes  int
	OutputBytes int
	DurationMs  int
}

type ImageToolsRepository interface {
	CountFreeSince(ctx context.Context, userID int64, since time.Time) (int, error)
	Balance(ctx context.Context, userID int64) (float64, error)
	// Charge records a run: free while a subscriber has free runs left today (decided under a per-user
	// lock), otherwise the price is taken from the balance. ErrInsufficientBalance when it can't be.
	Charge(ctx context.Context, use ImageToolUse, freeDaily int, since time.Time) (free bool, err error)
	ListUses(ctx context.Context, q ImageToolUseQuery) ([]ImageToolUseRecord, int64, error)
	// Stats sums runs since a time by tool, for one user or (userID 0) everyone.
	Stats(ctx context.Context, userID int64, since time.Time) ([]ImageToolStat, error)
}

type imageToolSubscriptions interface {
	ListActiveByUserID(ctx context.Context, userID int64) ([]UserSubscription, error)
}

type imageToolBalanceCache interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// ImageToolOptions are the per-run options (validated by Run).
type ImageToolOptions struct {
	Model string // remove_bg: isnet (default) | u2net
	Scale int    // upscale: 2 | 4 (default)
}

type ImageToolResult struct {
	Data        []byte
	ContentType string
	Free        bool
	Cost        float64
	FreeLeft    int
}

type ImageToolsService struct {
	settings imageToolSettings
	cacheMu  sync.Mutex
	cached   *ImageToolsConfig
	cachedAt time.Time

	repo    ImageToolsRepository
	subs    imageToolSubscriptions
	cache   imageToolBalanceCache
	cfg     ImageToolsConfig
	client  *http.Client
	waiting atomic.Int32
	now     func() time.Time
}

func NewImageToolsService(repo ImageToolsRepository, subs imageToolSubscriptions, cache imageToolBalanceCache, cfg ImageToolsConfig) *ImageToolsService {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	return &ImageToolsService{repo: repo, subs: subs, cache: cache, cfg: cfg, client: &http.Client{Timeout: 3 * time.Minute}, now: timezone.Now}
}

// WithSettings lets admins override prices, the free allowance and the on/off switch at runtime.
func (s *ImageToolsService) WithSettings(settings imageToolSettings) *ImageToolsService {
	s.settings = settings
	return s
}

// config is the file / env configuration with the admin's overrides on top (cached briefly).
func (s *ImageToolsService) config(ctx context.Context) ImageToolsConfig {
	if s.settings == nil {
		return s.cfg
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cached != nil && time.Since(s.cachedAt) < imageToolSettingsTTL {
		return *s.cached
	}
	c := s.cfg
	values, err := s.settings.GetMultiple(ctx, []string{settingImageToolsEnabled, settingImageToolsPriceRemoveBg, settingImageToolsPriceUpscale, settingImageToolsFreeDaily})
	if err != nil {
		return c
	}
	if values[settingImageToolsEnabled] == "false" {
		c.Disabled = true
	}
	if v, err := strconv.ParseFloat(values[settingImageToolsPriceRemoveBg], 64); err == nil && v >= 0 {
		c.PriceRemoveBg = v
	}
	if v, err := strconv.ParseFloat(values[settingImageToolsPriceUpscale], 64); err == nil && v >= 0 {
		c.PriceUpscale = v
	}
	if v, err := strconv.Atoi(values[settingImageToolsFreeDaily]); err == nil && v >= 0 {
		c.FreeDaily = v
	}
	s.cached, s.cachedAt = &c, time.Now()
	return c
}

func (c ImageToolsConfig) enabled() bool { return c.BaseURL != "" && !c.Disabled }

func (c ImageToolsConfig) price(tool string) float64 {
	if tool == ImageToolUpscale {
		return c.PriceUpscale
	}
	return c.PriceRemoveBg
}

func (s *ImageToolsService) subscribed(ctx context.Context, userID int64) (bool, error) {
	if s.subs == nil {
		return false, nil
	}
	subs, err := s.subs.ListActiveByUserID(ctx, userID)
	return len(subs) > 0, err
}

// Quota reports today's free runs, the balance and the prices.
func (s *ImageToolsService) Quota(ctx context.Context, userID int64) (*ImageToolsQuota, error) {
	subscribed, err := s.subscribed(ctx, userID)
	if err != nil {
		return nil, err
	}
	used, err := s.repo.CountFreeSince(ctx, userID, timezone.StartOfDay(s.now()))
	if err != nil {
		return nil, err
	}
	balance, err := s.repo.Balance(ctx, userID)
	if err != nil {
		return nil, err
	}
	c := s.config(ctx)
	q := &ImageToolsQuota{Enabled: c.enabled(), Subscribed: subscribed, FreeDaily: c.FreeDaily, FreeUsed: used, Balance: balance,
		Prices: map[string]float64{ImageToolRemoveBg: c.PriceRemoveBg, ImageToolUpscale: c.PriceUpscale}}
	if subscribed {
		q.FreeLeft = max(0, c.FreeDaily-used)
	}
	return q, nil
}

func (s *ImageToolsService) insufficient(q *ImageToolsQuota, price float64) error {
	need := strconv.FormatFloat(price, 'f', -1, 64)
	if q.Subscribed {
		return infraerrors.Forbidden("IMAGE_TOOLS_INSUFFICIENT_BALANCE", fmt.Sprintf("今天的 %d 次免费额度已用完，本次需要 ¥%s，余额不足：请到 hivegpt.cn 充值后再试（Insufficient balance）", q.FreeDaily, need))
	}
	return infraerrors.Forbidden("IMAGE_TOOLS_INSUFFICIENT_BALANCE", fmt.Sprintf("本次需要 ¥%s，余额不足：请到 hivegpt.cn 充值，订阅用户每天还可免费使用 %d 次（Insufficient balance）", need, q.FreeDaily))
}

// Run processes one image and charges for it once it succeeded.
func (s *ImageToolsService) Run(ctx context.Context, userID, apiKeyID int64, tool string, opts ImageToolOptions, image []byte) (*ImageToolResult, error) {
	path, err := imageToolPath(tool, opts)
	if err != nil {
		return nil, err
	}
	if len(image) == 0 {
		return nil, ErrImageToolsNoImage
	}
	if len(image) > ImageToolMaxInputBytes {
		return nil, ErrImageToolsTooLarge
	}
	c := s.config(ctx)
	if !c.enabled() {
		return nil, ErrImageToolsDisabled
	}
	quota, err := s.Quota(ctx, userID)
	if err != nil {
		return nil, err
	}
	price := c.price(tool)
	if quota.FreeLeft == 0 && quota.Balance < price {
		return nil, s.insufficient(quota, price)
	}
	if s.waiting.Add(1) > imageToolMaxWaiting {
		s.waiting.Add(-1)
		return nil, ErrImageToolsBusy
	}
	started := time.Now()
	data, contentType, err := s.process(ctx, path, image)
	s.waiting.Add(-1)
	if err != nil {
		return nil, err
	}
	use := ImageToolUse{UserID: userID, APIKeyID: apiKeyID, Tool: tool, Price: price, Subscribed: quota.Subscribed,
		InputBytes: len(image), OutputBytes: len(data), DurationMs: int(time.Since(started).Milliseconds())}
	free, err := s.repo.Charge(ctx, use, c.FreeDaily, timezone.StartOfDay(s.now()))
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			return nil, s.insufficient(quota, price)
		}
		return nil, err
	}
	res := &ImageToolResult{Data: data, ContentType: contentType, Free: free, FreeLeft: quota.FreeLeft}
	if free {
		res.FreeLeft = max(0, quota.FreeLeft-1)
	} else {
		res.Cost = price
		if s.cache != nil {
			_ = s.cache.InvalidateUserBalance(ctx, userID)
		}
	}
	return res, nil
}

func imageToolPath(tool string, opts ImageToolOptions) (string, error) {
	switch tool {
	case ImageToolRemoveBg:
		model := opts.Model
		if model == "" {
			model = "isnet"
		}
		if model != "isnet" && model != "u2net" {
			return "", ErrImageToolsBadTool
		}
		return "/remove-bg?model=" + model, nil
	case ImageToolUpscale:
		scale := opts.Scale
		if scale == 0 {
			scale = 4
		}
		if scale != 2 && scale != 4 {
			return "", ErrImageToolsBadTool
		}
		return "/upscale?" + url.Values{"scale": {strconv.Itoa(scale)}}.Encode(), nil
	}
	return "", ErrImageToolsBadTool
}

func (s *ImageToolsService) process(ctx context.Context, path string, image []byte) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+path, bytes.NewReader(image))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		return nil, "", ErrImageToolsFailed
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, imageToolMaxOutputBytes))
	if err != nil {
		return nil, "", ErrImageToolsFailed
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return body, resp.Header.Get("Content-Type"), nil
	case http.StatusUnprocessableEntity:
		// The service explains what is wrong with the image (unreadable, already big enough...).
		var msg struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &msg) == nil && msg.Error != "" {
			return nil, "", infraerrors.BadRequest("IMAGE_TOOLS_BAD_IMAGE", msg.Error+"；本次未扣费")
		}
	case http.StatusRequestEntityTooLarge:
		return nil, "", ErrImageToolsTooLarge
	}
	return nil, "", ErrImageToolsFailed
}
