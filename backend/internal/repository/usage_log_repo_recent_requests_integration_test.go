//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestGetAccountRequestBucketsBatch(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())

	user := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "rr-" + sfx + "@test.local", Username: "rr" + sfx[len(sfx)-6:]}))
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: user.ID, Key: "sk-rr-" + sfx})
	accA := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "rr-a-" + sfx}))
	accB := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "rr-b-" + sfx}))

	start := time.Now().Add(-30 * time.Minute).Truncate(time.Minute)
	at := func(bucket int) time.Time { return start.Add(time.Duration(bucket)*3*time.Minute + 30*time.Second) }

	usage := func(accountID int64, created time.Time, n int) {
		for i := 0; i < n; i++ {
			_, err := integrationEntClient.UsageLog.Create().SetUserID(user.ID).SetAPIKeyID(key.ID).SetAccountID(accountID).
				SetRequestID(fmt.Sprintf("rr-%s-%d-%d-%d", sfx, accountID, created.Unix(), i)).SetModel("gpt-5").SetCreatedAt(created).Save(ctx)
			require.NoError(t, err)
		}
	}
	usage(accA.ID, at(0), 2)
	usage(accA.ID, at(5), 1)
	usage(accB.ID, at(5), 1)
	usage(accA.ID, start.Add(-time.Minute), 1) // before the window

	var opsIDs []int64
	opsRow := func(accountID int64, status int, owner string, businessLimited bool, created time.Time, events []map[string]any) {
		var raw any
		if events != nil {
			b, err := json.Marshal(events)
			require.NoError(t, err)
			raw = string(b)
		}
		var id int64
		err := integrationDB.QueryRowContext(ctx, `
			INSERT INTO ops_error_logs (account_id, error_phase, error_type, severity, status_code, error_owner, is_business_limited, upstream_errors, created_at)
			VALUES ($1, 'upstream', 'upstream_error', 'error', $2, $3, $4, $5::jsonb, $6) RETURNING id`,
			accountID, status, owner, businessLimited, raw, created).Scan(&id)
		require.NoError(t, err)
		opsIDs = append(opsIDs, id)
	}
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM ops_error_logs WHERE id = ANY($1)`, pq.Array(opsIDs))
	})

	// Recovered: A failed twice (retry) then the request succeeded on B → A gets one failure, B none.
	opsRow(accB.ID, 200, "provider", false, at(2), []map[string]any{
		{"account_id": accA.ID, "at_unix_ms": at(1).UnixMilli(), "kind": "retry"},
		{"account_id": accA.ID, "at_unix_ms": at(2).UnixMilli(), "kind": "failover"},
	})
	// Final provider failure without events → attributed to the row's account.
	opsRow(accB.ID, 502, "provider", false, at(3), nil)
	// Client error and business limit are not account failures.
	opsRow(accA.ID, 400, "client", false, at(3), []map[string]any{{"account_id": accA.ID, "at_unix_ms": at(3).UnixMilli()}})
	opsRow(accA.ID, 429, "provider", true, at(3), nil)
	// Recovered row without events cannot be attributed and is skipped.
	opsRow(accA.ID, 200, "provider", false, at(4), nil)

	repo := newUsageLogRepositoryWithSQL(integrationEntClient, integrationDB)
	got, err := repo.GetAccountRequestBucketsBatch(ctx, []int64{accA.ID, accB.ID}, start, 180, 20)
	require.NoError(t, err)

	a := got[accA.ID]
	require.NotNil(t, a)
	require.Equal(t, int64(3), a.Success)
	require.Equal(t, int64(1), a.Failure)
	require.Equal(t, service.AccountRequestBucket{Success: 2}, a.Buckets[0])
	require.Equal(t, service.AccountRequestBucket{Failure: 1}, a.Buckets[1])
	require.Equal(t, service.AccountRequestBucket{Success: 1}, a.Buckets[5])

	b := got[accB.ID]
	require.NotNil(t, b)
	require.Equal(t, int64(1), b.Success)
	require.Equal(t, int64(1), b.Failure)
	require.Equal(t, service.AccountRequestBucket{Failure: 1}, b.Buckets[3])
	require.Equal(t, service.AccountRequestBucket{Success: 1}, b.Buckets[5])
}
