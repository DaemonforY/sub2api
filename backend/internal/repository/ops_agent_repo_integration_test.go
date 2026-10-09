//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// The rows live in a window in 2001 so other suites' rows don't change the counts.
func TestOpsAgentRepository(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "ops-agent-" + suffix + "@x.test", Username: "oa" + suffix[len(suffix)-6:]}))
	group := rows.group(mustCreateGroup(t, integrationEntClient, &service.Group{Name: "ops-agent-" + suffix, Platform: service.PlatformOpenAI}))
	healthy := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "oa-healthy-" + suffix, Platform: service.PlatformOpenAI,
		Type: service.AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://gorustai.com/", "api_key": "sk-SECRET"}}))
	sick := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "oa-sick-" + suffix, Platform: service.PlatformOpenAI}))
	mustBindAccountToGroup(t, integrationEntClient, healthy.ID, group.ID, 1)
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: user.ID, Key: "sk-oa-" + suffix, Name: "oa"})

	base := time.Date(2001, 2, 3, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		_, err := integrationDB.ExecContext(ctx, `INSERT INTO usage_logs (user_id, api_key_id, account_id, group_id, model, request_id, created_at)
			VALUES ($1, $2, $3, $4, 'gpt-5.5', $5, $6)`, user.ID, key.ID, healthy.ID, group.ID, fmt.Sprintf("oa-%s-%d", suffix, i), base.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}
	var errorIDs []int64
	insertErr := func(at time.Time, status int, upstream any, msg string, account int64, business bool, chain string) {
		var id int64
		var chainArg any
		if chain != "" {
			chainArg = chain
		}
		err := integrationDB.QueryRowContext(ctx, `INSERT INTO ops_error_logs (user_id, account_id, group_id, model, request_path, stream,
			error_phase, error_type, severity, status_code, upstream_status_code, upstream_error_message, error_message, is_business_limited,
			upstream_errors, duration_ms, created_at)
			VALUES ($1, $2, $3, 'gpt-5.5', '/v1/responses', true, 'upstream', 'upstream_error', 'P1', $4, $5, $6, 'gateway text', $7, $8::jsonb, 1200, $9)
			RETURNING id`, user.ID, account, group.ID, status, upstream, msg, business, chainArg, at).Scan(&id)
		require.NoError(t, err)
		errorIDs = append(errorIDs, id)
	}
	chain := fmt.Sprintf(`[{"account_id": %d, "account_name": "oa-sick"}, {"account_id": %d}]`, sick.ID, sick.ID)
	insertErr(base.Add(1*time.Minute), 502, 503, "Our servers are currently overloaded.", sick.ID, false, chain)
	insertErr(base.Add(2*time.Minute), 502, 503, "Our servers are currently overloaded.", sick.ID, false, "")
	insertErr(base.Add(3*time.Minute), 200, 502, "", sick.ID, false, "")
	insertErr(base.Add(4*time.Minute), 403, nil, "", healthy.ID, true, "")
	var alertID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO ops_alert_events (rule_id, severity, status, title, description, fired_at)
		VALUES (0, 'P0', 'firing', 'P0: 成功率过低', 'success_rate < 95', $1) RETURNING id`, base.Add(5*time.Minute)).Scan(&alertID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM ops_error_logs WHERE id = ANY($1)`, pq.Array(errorIDs))
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM ops_alert_events WHERE id = $1`, alertID)
	})

	repo := NewOpsAgentRepository(integrationDB)
	start, end := base, base.Add(time.Hour)

	sum, err := repo.ErrorSummary(ctx, start, end)
	require.NoError(t, err)
	require.Equal(t, 3, sum.Success)
	require.Equal(t, 4, sum.Errors)
	require.Equal(t, 1, sum.BusinessLimited)
	require.Equal(t, 2, sum.Groups[0].Count, "the two overloaded 502s group together")
	require.Equal(t, "Our servers are currently overloaded.", sum.Groups[0].Message)
	require.Equal(t, 503, *sum.Groups[0].UpstreamStatus)
	require.Equal(t, []string{"gpt-5.5"}, sum.Groups[0].Models)
	require.Equal(t, []string{fmt.Sprintf("#%d %s", sick.ID, sick.Name)}, sum.Groups[0].Accounts)
	require.Equal(t, base.Add(time.Minute), sum.Groups[0].First.UTC())
	var mid *service.OpsAgentErrorGroup
	for i := range sum.Groups {
		if sum.Groups[i].StatusCode == 200 {
			mid = &sum.Groups[i]
		}
	}
	require.NotNil(t, mid)
	require.Equal(t, "gateway text", mid.Message, "falls back to the gateway's message")
	require.Equal(t, service.OpsAgentCount{Key: fmt.Sprintf("#%d %s", sick.ID, sick.Name), Errors: 3}, sum.ByAccount[0])
	require.Equal(t, service.OpsAgentCount{Key: fmt.Sprintf("#%d %s", healthy.ID, healthy.Name), Success: 3, Errors: 1}, sum.ByAccount[1])
	require.Equal(t, service.OpsAgentCount{Key: "gpt-5.5", Success: 3, Errors: 4}, sum.ByModel[0])
	require.Equal(t, fmt.Sprintf("#%d %s", user.ID, user.Email), sum.ByUser[0].Key)

	list, err := repo.ListErrors(ctx, service.OpsAgentErrorFilter{Start: start, End: end, StatusCode: 503, Limit: 10})
	require.NoError(t, err)
	require.Len(t, list, 2, "matches the upstream status too")
	oldest := list[1]
	require.Equal(t, []string{fmt.Sprintf("#%d oa-sick", sick.ID), fmt.Sprintf("#%d", sick.ID)}, oldest.Tried)
	require.Equal(t, group.Name, oldest.Group)
	require.Equal(t, user.Email, oldest.UserEmail)
	require.Equal(t, 1200, *oldest.DurationMs)
	list, err = repo.ListErrors(ctx, service.OpsAgentErrorFilter{Start: start, End: end, AccountID: healthy.ID, Model: "GPT"})
	require.NoError(t, err)
	require.Len(t, list, 1)

	timeline, err := repo.Timeline(ctx, start, end, 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, []service.OpsAgentTimelineBucket{
		{Start: base, Success: 2, Errors: 1},
		{Start: base.Add(2 * time.Minute), Success: 1, Errors: 2},
		{Start: base.Add(4 * time.Minute), Success: 0, Errors: 1},
	}, normalizeBuckets(timeline))

	accounts, err := repo.Accounts(ctx)
	require.NoError(t, err)
	var found *service.OpsAgentAccount
	for i := range accounts {
		if accounts[i].ID == healthy.ID {
			found = &accounts[i]
		}
	}
	require.NotNil(t, found)
	require.Equal(t, "gorustai.com", found.BaseHost)
	require.Equal(t, []string{group.Name}, found.Groups)

	alerts, err := repo.AlertEvents(ctx, base, 50)
	require.NoError(t, err)
	var alert *service.OpsAgentAlert
	for i := range alerts {
		if alerts[i].ID == alertID {
			alert = &alerts[i]
		}
	}
	require.NotNil(t, alert)
	require.Equal(t, "P0: 成功率过低", alert.Title)
	require.Nil(t, alert.ResolvedAt)
}

func normalizeBuckets(list []service.OpsAgentTimelineBucket) []service.OpsAgentTimelineBucket {
	for i := range list {
		list[i].Start = list[i].Start.UTC()
	}
	return list
}
