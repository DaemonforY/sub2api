package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// accountRequestBucketsSQL 统计账号在窗口内每格的成功/失败请求数。
//
// 成功：usage_logs 的计费行。
// 失败：ops_error_logs 里与上游有关的行——最终失败且归因上游（error_owner='provider'，
// 不含业务限流），或最终成功但中途有上游失败（status_code<400 的 recovered 行）。
// 一行里每个出现在 upstream_errors 的账号记 1 次失败，时间取该账号最早一次失败；
// 没有 upstream_errors 的行记到行上的 account_id。
const accountRequestBucketsSQL = `
WITH success AS (
	SELECT account_id,
		FLOOR(EXTRACT(EPOCH FROM (created_at - $2::timestamptz)) / $3)::int AS bucket,
		COUNT(*) AS n
	FROM usage_logs
	WHERE account_id = ANY($1) AND created_at >= $2::timestamptz
	GROUP BY 1, 2
),
failure_attempts AS (
	SELECT e.id AS log_id,
		COALESCE(NULLIF(ev->>'account_id', '')::bigint, e.account_id) AS account_id,
		COALESCE(to_timestamp(NULLIF(ev->>'at_unix_ms', '')::double precision / 1000.0), e.created_at) AS at
	FROM ops_error_logs e
	LEFT JOIN LATERAL jsonb_array_elements(
		CASE WHEN jsonb_typeof(e.upstream_errors) = 'array' THEN e.upstream_errors ELSE '[]'::jsonb END
	) ev ON TRUE
	WHERE e.created_at >= $2::timestamptz
		AND (
			-- recovered 行只认 upstream_errors 里的账号，行上的 account_id 是最后成功的那个
			(COALESCE(e.status_code, 0) < 400 AND ev IS NOT NULL)
			OR (COALESCE(e.status_code, 0) >= 400 AND e.error_owner = 'provider' AND NOT e.is_business_limited)
		)
),
failure AS (
	SELECT account_id,
		FLOOR(EXTRACT(EPOCH FROM (first_at - $2::timestamptz)) / $3)::int AS bucket,
		COUNT(*) AS n
	FROM (
		SELECT log_id, account_id, MIN(at) AS first_at
		FROM failure_attempts
		WHERE account_id = ANY($1)
		GROUP BY log_id, account_id
	) per_request
	WHERE first_at >= $2::timestamptz
	GROUP BY 1, 2
)
SELECT account_id, bucket, n, FALSE AS failed FROM success
UNION ALL
SELECT account_id, bucket, n, TRUE AS failed FROM failure
`

// GetAccountRequestBucketsBatch 返回窗口内每个账号按格的成功/失败请求数。
func (r *usageLogRepository) GetAccountRequestBucketsBatch(ctx context.Context, accountIDs []int64, windowStart time.Time, bucketSeconds, bucketCount int) (map[int64]*service.AccountRecentRequests, error) {
	result := make(map[int64]*service.AccountRecentRequests, len(accountIDs))
	if len(accountIDs) == 0 || bucketSeconds <= 0 || bucketCount <= 0 {
		return result, nil
	}
	rows, err := r.sql.QueryContext(ctx, accountRequestBucketsSQL, pq.Array(accountIDs), windowStart, bucketSeconds)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			accountID int64
			bucket    int
			n         int64
			failed    bool
		)
		if err := rows.Scan(&accountID, &bucket, &n, &failed); err != nil {
			return nil, err
		}
		// 写入时间略晚于窗口末尾（时钟差）时归到最后一格。
		if bucket < 0 {
			continue
		}
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		stats := result[accountID]
		if stats == nil {
			stats = &service.AccountRecentRequests{Buckets: make([]service.AccountRequestBucket, bucketCount)}
			result[accountID] = stats
		}
		if failed {
			stats.Buckets[bucket].Failure += n
			stats.Failure += n
		} else {
			stats.Buckets[bucket].Success += n
			stats.Success += n
		}
	}
	return result, rows.Err()
}
