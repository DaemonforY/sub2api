package service

import (
	"context"
	"errors"
	"time"
)

// 账号「近 1 小时请求」：20 格、每格 3 分钟，最后一格是当前进行中的 3 分钟。
const (
	AccountRecentRequestBucketSeconds = 180
	AccountRecentRequestBucketCount   = 20
)

// AccountRequestBucket 一格内的请求数。成功来自 usage_logs；失败来自 ops_error_logs：
// 一个请求在某账号上失败过（含之后切到别的账号成功的）记为该账号 1 次失败。
type AccountRequestBucket struct {
	Success int64 `json:"s"`
	Failure int64 `json:"f"`
}

type AccountRecentRequests struct {
	Success int64                  `json:"success"`
	Failure int64                  `json:"failure"`
	Buckets []AccountRequestBucket `json:"buckets"`
}

type AccountRecentRequestsBatch struct {
	WindowStart   time.Time                        `json:"window_start"`
	BucketSeconds int                              `json:"bucket_seconds"`
	Stats         map[int64]*AccountRecentRequests `json:"stats"`
}

type accountRecentRequestsBatchReader interface {
	GetAccountRequestBucketsBatch(ctx context.Context, accountIDs []int64, windowStart time.Time, bucketSeconds, bucketCount int) (map[int64]*AccountRecentRequests, error)
}

// accountRecentRequestsWindowStart 把窗口对齐到 3 分钟边界，最后一格包含 now。
func accountRecentRequestsWindowStart(now time.Time) time.Time {
	bucket := time.Duration(AccountRecentRequestBucketSeconds) * time.Second
	end := now.Truncate(bucket).Add(bucket)
	return end.Add(-bucket * AccountRecentRequestBucketCount)
}

// GetRecentRequestsBatch 返回账号近 1 小时按 3 分钟分格的成功/失败请求数。
func (s *AccountUsageService) GetRecentRequestsBatch(ctx context.Context, accountIDs []int64) (*AccountRecentRequestsBatch, error) {
	reader, ok := s.usageLogRepo.(accountRecentRequestsBatchReader)
	if !ok {
		return nil, errors.New("recent request stats unavailable")
	}
	start := accountRecentRequestsWindowStart(time.Now())
	stats, err := reader.GetAccountRequestBucketsBatch(ctx, accountIDs, start, AccountRecentRequestBucketSeconds, AccountRecentRequestBucketCount)
	if err != nil {
		return nil, err
	}
	for _, id := range accountIDs {
		if stats[id] == nil {
			stats[id] = &AccountRecentRequests{Buckets: make([]AccountRequestBucket, AccountRecentRequestBucketCount)}
		}
	}
	return &AccountRecentRequestsBatch{
		WindowStart:   start,
		BucketSeconds: AccountRecentRequestBucketSeconds,
		Stats:         stats,
	}, nil
}
