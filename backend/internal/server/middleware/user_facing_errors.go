package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// User-facing gateway rejections. End users mostly read these in their client (Codex, Cherry Studio,
// the canvas…) rather than in the dashboard, so each message says in Chinese what went wrong and what
// to do next. The original English text stays in parentheses at the end: some clients and our own ops
// classifiers match on it, and it keeps the message searchable.

const (
	msgAPIKeyRequired       = "缺少 API Key：请在请求头 Authorization: Bearer <你的 Key>（或 x-api-key）里填写控制台「API 密钥」中的 Key（API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header）"
	msgInvalidAPIKey        = "API Key 无效或已被删除，请到控制台「API 密钥」复制一个有效的 Key 后重试（Invalid API key）"
	msgAPIKeyDisabled       = "API Key 已停用，请到控制台「API 密钥」启用它或换一个 Key（API key is disabled）"
	msgUserInactive         = "账号已被停用，如有疑问请联系客服（User account is not active）"
	msgSubscriptionNotFound = "这个 Key 属于订阅分组，但你没有该分组的有效订阅：请到 " + service.RechargeURL + " 购买或续费套餐，或换一个按量计费分组的 Key（No active subscription found for this group）"
)

func insufficientBalanceMessage(balance float64) string {
	return fmt.Sprintf("账户余额不足（当前余额 %.2f），请到 %s 充值，或改用订阅分组的 Key（Insufficient account balance）", balance, service.RechargeURL)
}

// subscriptionLimitMessage explains a subscription rejection from ValidateAndCheckLimits, including
// how much was used and when the window resets.
// With smart billing the same key falls back to balance by itself; the user only lacks balance.
const (
	msgSubscriptionNotFoundSmart = "这个 Key 属于订阅分组，你目前没有该分组的有效订阅，余额也不足：充值后这个 Key 会自动按量计费继续使用，或购买 / 续费套餐：" + service.RechargeURL + "（No active subscription found for this group）"
	smartBillingHintOld          = "；急用可换一个按量计费分组的 Key"
	smartBillingHintNew          = "；余额也不足，充值后这个 Key 会自动改用余额继续，不用换 Key"
)

func subscriptionNotFoundMessage(smart bool) string {
	if smart {
		return msgSubscriptionNotFoundSmart
	}
	return msgSubscriptionNotFound
}

func smartBillingLimitMessage(msg string) string {
	return strings.Replace(msg, smartBillingHintOld, smartBillingHintNew, 1)
}

func subscriptionLimitMessage(err error, sub *service.UserSubscription, group *service.Group) string {
	if sub == nil || group == nil {
		return err.Error()
	}
	switch {
	case errors.Is(err, service.ErrSubscriptionExpired):
		return fmt.Sprintf("订阅已于 %s 到期，请到 %s 续费后再试（subscription has expired）", formatUserTime(sub.ExpiresAt), service.RechargeURL)
	case errors.Is(err, service.ErrSubscriptionSuspended):
		return "订阅已被暂停，如有疑问请联系客服（subscription is suspended）"
	case errors.Is(err, service.ErrDailyLimitExceeded):
		return windowLimitMessage("今日", "daily", sub.DailyUsageUSD, group.DailyLimitUSD, sub.DailyResetTime())
	case errors.Is(err, service.ErrWeeklyLimitExceeded):
		return windowLimitMessage("本周", "weekly", sub.WeeklyUsageUSD, group.WeeklyLimitUSD, sub.WeeklyResetTime())
	case errors.Is(err, service.ErrMonthlyLimitExceeded):
		return windowLimitMessage("本月", "monthly", sub.MonthlyUsageUSD, group.MonthlyLimitUSD, sub.MonthlyResetTime())
	}
	return err.Error()
}

func windowLimitMessage(periodZh, periodEn string, used float64, limit *float64, resetAt *time.Time) string {
	usage := ""
	if limit != nil {
		usage = fmt.Sprintf("（已用 $%.2f / 额度 $%.2f）", used, *limit)
	}
	reset := "，等额度重置后再试"
	if resetAt != nil {
		reset = fmt.Sprintf("，将于 %s 重置", formatUserTime(*resetAt))
	}
	return fmt.Sprintf("订阅%s额度已用完%s%s；急用可换一个按量计费分组的 Key（%s usage limit exceeded）", periodZh, usage, reset, periodEn)
}

func formatUserTime(t time.Time) string {
	return t.In(timezone.Location()).Format("2006-01-02 15:04")
}
