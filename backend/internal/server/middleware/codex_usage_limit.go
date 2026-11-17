package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Codex (CLI, Desktop, IDE) treats an ordinary 429 as retryable. After its retries it shows only
// "exceeded retry limit, last status: 429 Too Many Requests", and our message never reaches the
// user. Codex stops and explains only one 429 shape: error.type "usage_limit_reached". It then prints
//
//	You’ve hit your usage limit. <x-codex-promo-message>, or try again at <local reset time>.
//
// using error.resets_at (unix seconds). Codex reads headers with HeaderValue::to_str, which rejects
// non-ASCII, so the promo header is English; the Chinese text stays in error.message.
// plan_type is left out on purpose: it would switch Codex to ChatGPT "upgrade to Pro" copy.

const (
	codexPromoMessageHeader = "x-codex-promo-message"
	codexUsageLimitType     = "usage_limit_reached"
	usageLimitExceededCode  = "USAGE_LIMIT_EXCEEDED"
)

// CodexUsageLimitBody is the OpenAI-style error object Codex parses for usage limits.
type CodexUsageLimitBody struct {
	Message            string `json:"message"`
	Type               string `json:"type"`
	Code               string `json:"code"`
	ResetsAt           *int64 `json:"resets_at,omitempty"`
	LimitWindowMinutes int    `json:"limit_window_minutes,omitempty"`
}

type codexUsageLimitResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Error   CodexUsageLimitBody `json:"error"`
}

func isCodexClientRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator"))
}

// abortWithCodexUsageLimit answers a subscription window limit in the shape Codex understands.
// It returns false when err is not a daily/weekly/monthly limit, leaving the caller's default path.
func abortWithCodexUsageLimit(c *gin.Context, err error, sub *service.UserSubscription, group *service.Group, message string, smartBilling bool) bool {
	if sub == nil || group == nil {
		return false
	}
	var (
		periodEn string
		minutes  int
		used     float64
		limit    *float64
	)
	switch {
	case errors.Is(err, service.ErrDailyLimitExceeded):
		periodEn, minutes, used, limit = "daily", 24*60, sub.DailyUsageUSD, group.DailyLimitUSD
	case errors.Is(err, service.ErrWeeklyLimitExceeded):
		periodEn, minutes, used, limit = "weekly", 7*24*60, sub.WeeklyUsageUSD, group.WeeklyLimitUSD
	case errors.Is(err, service.ErrMonthlyLimitExceeded):
		periodEn, minutes, used, limit = "monthly", 30*24*60, sub.MonthlyUsageUSD, group.MonthlyLimitUSD
	default:
		return false
	}

	body := CodexUsageLimitBody{
		Message:            message,
		Type:               codexUsageLimitType,
		Code:               usageLimitExceededCode,
		LimitWindowMinutes: minutes,
	}
	if resetAt := subscriptionResetTime(err, sub); resetAt != nil {
		ts := resetAt.Unix()
		body.ResetsAt = &ts
	}

	c.Header(codexPromoMessageHeader, codexUsageLimitPromo(periodEn, used, limit, smartBilling))
	c.JSON(http.StatusTooManyRequests, codexUsageLimitResponse{
		Code:    usageLimitExceededCode,
		Message: message,
		Error:   body,
	})
	c.Abort()
	return true
}

// codexUsageLimitPromo is the ASCII sentence Codex puts between "You’ve hit your usage limit." and
// ", or try again at …".
func codexUsageLimitPromo(periodEn string, used float64, limit *float64, smartBilling bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your HiveGPT plan's %s quota is used up", periodEn)
	if limit != nil {
		fmt.Fprintf(&b, " ($%.2f of $%.2f)", used, *limit)
	}
	if smartBilling {
		_, _ = b.WriteString(". Top up at " + service.RechargeURL + " and this key continues on balance")
	} else {
		_, _ = b.WriteString(". Use a pay-as-you-go key now")
	}
	return b.String()
}

func subscriptionResetTime(err error, sub *service.UserSubscription) *time.Time {
	switch {
	case errors.Is(err, service.ErrDailyLimitExceeded):
		return sub.DailyResetTime()
	case errors.Is(err, service.ErrWeeklyLimitExceeded):
		return sub.WeeklyResetTime()
	case errors.Is(err, service.ErrMonthlyLimitExceeded):
		return sub.MonthlyResetTime()
	}
	return nil
}
