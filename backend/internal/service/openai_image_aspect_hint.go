package service

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// gpt-image requests often come back in another orientation than the size asked for: the upstream
// treats size as a hint. Measured on hivegpt.cn over 7 days (2026-10-29): 6.4 % of portrait, 20.6 % of
// landscape and 68 % of square requests returned the wrong orientation. Saying the orientation and
// ratio in the prompt fixed it in a test (5 of 6 landscape before, 6 of 6 portrait after), so the
// gateway appends that sentence to gpt-image prompts that ask for an explicit WxH size. Admins can turn
// it off (图片工具 → 生图画幅提示).

const (
	SettingKeyOpenAIImageAspectHint = "openai_image_aspect_hint"
	openAIImageAspectHintMarker     = "【画幅】"
)

var openAIImageSizeRe = regexp.MustCompile(`^(\d{2,5})x(\d{2,5})$`)

// Common ratios the hint names (a size is described by the closest one).
var openAIImageRatios = [][2]int{{1, 1}, {2, 3}, {3, 2}, {3, 4}, {4, 3}, {4, 5}, {5, 4}, {9, 16}, {16, 9}, {9, 21}, {21, 9}, {1, 2}, {2, 1}}

// openAIImageAspectHint is the sentence for a WxH size ("" for auto or anything else).
func openAIImageAspectHint(size string) string {
	m := openAIImageSizeRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(size)))
	if m == nil {
		return ""
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	if w <= 0 || h <= 0 {
		return ""
	}
	best, bestDiff := openAIImageRatios[0], math.MaxFloat64
	for _, r := range openAIImageRatios {
		if diff := math.Abs(math.Log(float64(w)/float64(h)) - math.Log(float64(r[0])/float64(r[1]))); diff < bestDiff {
			best, bestDiff = r, diff
		}
	}
	zh, en := "方形", "square"
	switch {
	case w > h:
		zh, en = "横版", "landscape"
	case w < h:
		zh, en = "竖版", "portrait"
	}
	ratio := fmt.Sprintf("%d:%d", best[0], best[1])
	return fmt.Sprintf("%s%s构图，画幅宽:高 = %s（%d×%d），请按这个比例构图。Aspect ratio: %s %s.", openAIImageAspectHintMarker, zh, ratio, w, h, en, ratio)
}

// withOpenAIImageAspectHint appends the hint to a prompt that doesn't carry one yet.
func withOpenAIImageAspectHint(prompt, size string) (string, bool) {
	hint := openAIImageAspectHint(size)
	if hint == "" || strings.TrimSpace(prompt) == "" || strings.Contains(prompt, openAIImageAspectHintMarker) {
		return prompt, false
	}
	return strings.TrimRight(prompt, " \n") + "\n\n" + hint, true
}

// applyOpenAIImageAspectHint returns the body and parsed request to forward: with the hint in the prompt
// when it is on and applies, else unchanged. parsed is copied, never modified.
func (s *OpenAIGatewayService) applyOpenAIImageAspectHint(ctx context.Context, body []byte, parsed *OpenAIImagesRequest, model string) ([]byte, *OpenAIImagesRequest, error) {
	if s == nil || s.settingService == nil || !s.settingService.IsOpenAIImageAspectHintEnabled(ctx) {
		return body, parsed, nil
	}
	if !IsGPTImageGenerationModel(model) && !IsGPTImageGenerationModel(parsed.Model) {
		return body, parsed, nil
	}
	prompt, ok := withOpenAIImageAspectHint(parsed.Prompt, parsed.Size)
	if !ok {
		return body, parsed, nil
	}
	rewritten, contentType, err := rewriteOpenAIImagesField(body, parsed.ContentType, "prompt", prompt)
	if err != nil {
		return nil, nil, err
	}
	hinted := *parsed
	hinted.Prompt, hinted.ContentType = prompt, contentType
	return rewritten, &hinted, nil
}

// ---- the admin switch (default on), cached for a minute ----

type openAIImageAspectHintCache struct {
	mu      sync.Mutex
	value   bool
	expires time.Time
}

var aspectHintCache openAIImageAspectHintCache

func (s *SettingService) IsOpenAIImageAspectHintEnabled(ctx context.Context) bool {
	aspectHintCache.mu.Lock()
	defer aspectHintCache.mu.Unlock()
	if time.Now().Before(aspectHintCache.expires) {
		return aspectHintCache.value
	}
	value := true
	if s != nil && s.settingRepo != nil {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyOpenAIImageAspectHint)
		cancel()
		if err == nil && strings.TrimSpace(raw) != "" {
			value = strings.TrimSpace(raw) != "false"
		}
	}
	aspectHintCache.value, aspectHintCache.expires = value, time.Now().Add(time.Minute)
	return value
}

// InvalidateOpenAIImageAspectHint makes the next request read the switch again.
func InvalidateOpenAIImageAspectHint() {
	aspectHintCache.mu.Lock()
	aspectHintCache.expires = time.Time{}
	aspectHintCache.mu.Unlock()
}

// ---- admin: the switch and how often the orientation came back wrong ----

// ImageOrientationStat is gpt-image requests with an explicit WxH size over a period, by the orientation
// asked for, and how many came back in another orientation.
type ImageOrientationStat struct {
	Asked string `json:"asked"` // portrait / landscape / square
	Total int64  `json:"total"`
	Wrong int64  `json:"wrong"`
}

type ImageAspectHintView struct {
	Enabled bool                   `json:"enabled"`
	Day     []ImageOrientationStat `json:"day"`
	Week    []ImageOrientationStat `json:"week"`
}

func (s *ImageToolsService) AspectHint(ctx context.Context) (*ImageAspectHintView, error) {
	v := &ImageAspectHintView{Enabled: true}
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, []string{SettingKeyOpenAIImageAspectHint})
		if err != nil {
			return nil, err
		}
		v.Enabled = strings.TrimSpace(values[SettingKeyOpenAIImageAspectHint]) != "false"
	}
	now := s.now()
	var err error
	if v.Day, err = s.repo.OrientationStats(ctx, now.Add(-24*time.Hour)); err != nil {
		return nil, err
	}
	if v.Week, err = s.repo.OrientationStats(ctx, now.Add(-7*24*time.Hour)); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *ImageToolsService) SaveAspectHint(ctx context.Context, enabled bool) (*ImageAspectHintView, error) {
	if s.settings == nil {
		return nil, ErrImageToolsSettingsReadOnly
	}
	if err := s.settings.SetMultiple(ctx, map[string]string{SettingKeyOpenAIImageAspectHint: strconv.FormatBool(enabled)}); err != nil {
		return nil, err
	}
	InvalidateOpenAIImageAspectHint()
	return s.AspectHint(ctx)
}
