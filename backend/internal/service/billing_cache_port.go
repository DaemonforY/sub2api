package service

import (
	"time"
)

// SubscriptionCacheData represents cached subscription data
type SubscriptionCacheData struct {
	Status       string
	ExpiresAt    time.Time
	DailyUsage   float64
	WeeklyUsage  float64
	MonthlyUsage float64
	// ResetUntil: unix end of a bought 「重置今日额度」 day (0 = none); while it is in the future
	// usage only adds to DailyUsage and the weekly/monthly limits are not checked.
	ResetUntil int64
	Version    int64
}
