//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSignupTrialStartRepository(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewSignupTrialStartRepository(integrationDB)
	group := rows.group(mustCreateGroup(t, integrationEntClient, &service.Group{Name: "sts-" + sfx, Platform: service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeSubscription, IsExclusive: true, RateMultiplier: 1}))
	acc := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "sts-acc-" + sfx}))

	type fixture struct {
		user *service.User
		sub  *service.UserSubscription
		key  int64
	}
	mk := func(tag string, window time.Duration) fixture {
		u := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "sts-" + tag + "-" + sfx + "@test.local", Username: "sts" + tag}))
		_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '30 hours' WHERE id = $1`, u.ID)
		require.NoError(t, err)
		sub := mustCreateSubscription(t, integrationEntClient, &service.UserSubscription{UserID: u.ID, GroupID: group.ID,
			StartsAt: time.Now().Add(-30 * time.Hour), ExpiresAt: time.Now().Add(window), Notes: "auto assigned by signup defaults"})
		require.NoError(t, repo.Create(ctx, sub.ID, u.ID, 1))
		require.NoError(t, repo.Create(ctx, sub.ID, u.ID, 1), "idempotent")
		key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Key: "sk-sts-" + tag + "-" + sfx})
		return fixture{user: u, sub: sub, key: key.ID}
	}
	call := func(f fixture, at time.Time) {
		_, err := integrationEntClient.UsageLog.Create().SetUserID(f.user.ID).SetAPIKeyID(f.key).SetAccountID(acc.ID).
			SetSubscriptionID(f.sub.ID).SetGroupID(group.ID).SetRequestID(fmt.Sprintf("sts-%d-%d", f.sub.ID, at.UnixNano())).
			SetModel("gpt-5").SetCreatedAt(at).Save(ctx)
		require.NoError(t, err)
	}
	used := mk("used", 6*24*time.Hour)
	idle := mk("idle", 6*24*time.Hour)
	bought := mk("bought", 6*24*time.Hour)
	lapsed := mk("lapsed", -time.Hour)

	firstCall := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	call(used, firstCall.Add(time.Hour))
	call(used, firstCall)
	// A purchase extended this one before its first call: the new expiry stands.
	_, err := integrationDB.ExecContext(ctx, `UPDATE user_subscriptions SET expires_at = expires_at + INTERVAL '30 days' WHERE id = $1`, bought.sub.ID)
	require.NoError(t, err)
	call(bought, time.Now().Add(-time.Hour))

	userIDs := []int64{used.user.ID, idle.user.ID, bought.user.ID, lapsed.user.ID}
	pending, err := repo.PendingForUsers(ctx, userIDs)
	require.NoError(t, err)
	require.Contains(t, pending, used.sub.ID, "used but not yet moved by the job")
	require.Contains(t, pending, idle.sub.ID)
	require.Contains(t, pending, lapsed.sub.ID)
	require.NotContains(t, pending, bought.sub.ID)
	require.Equal(t, 1, pending[idle.sub.ID].TrialDays)

	// The activation reminder sees the idle user's pending trial.
	due, err := NewActivationReminderRepository(integrationDB).DueUsers(ctx, time.Now().Add(-72*time.Hour), time.Now().Add(-24*time.Hour), 500)
	require.NoError(t, err)
	var idleDue *service.ActivationReminderUser
	for i := range due {
		if due[i].ID == idle.user.ID {
			idleDue = &due[i]
		}
		require.NotEqual(t, used.user.ID, due[i].ID, "made a call")
	}
	require.NotNil(t, idleDue)
	require.Equal(t, 1, idleDue.TrialDays)
	require.Equal(t, group.Name, idleDue.TrialGroup)
	require.WithinDuration(t, pending[idle.sub.ID].PendingExpiresAt, idleDue.TrialUntil, time.Second)

	started, err := repo.StartUsed(ctx, 100)
	require.NoError(t, err)
	var mine []service.SignupTrialStarted
	for _, st := range started {
		for _, id := range userIDs {
			if st.UserID == id {
				mine = append(mine, st)
			}
		}
	}
	require.Len(t, mine, 1)
	require.Equal(t, used.sub.ID, mine[0].SubscriptionID)
	require.Equal(t, group.ID, mine[0].GroupID)
	require.WithinDuration(t, firstCall, mine[0].StartsAt, time.Second)
	require.WithinDuration(t, firstCall.Add(24*time.Hour), mine[0].ExpiresAt, time.Second)

	var startsAt, expiresAt time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT starts_at, expires_at FROM user_subscriptions WHERE id = $1`, used.sub.ID).Scan(&startsAt, &expiresAt))
	require.WithinDuration(t, firstCall.Add(24*time.Hour), expiresAt, time.Second)

	again, err := repo.StartUsed(ctx, 100)
	require.NoError(t, err)
	for _, st := range again {
		require.NotEqual(t, used.sub.ID, st.SubscriptionID, "started once")
	}

	_, err = repo.CloseStale(ctx, time.Now())
	require.NoError(t, err)
	reason := func(subID int64) string {
		var r *string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT closed_reason FROM signup_trial_starts WHERE subscription_id = $1`, subID).Scan(&r))
		if r == nil {
			return ""
		}
		return *r
	}
	require.Equal(t, "started", reason(used.sub.ID))
	require.Equal(t, "", reason(idle.sub.ID))
	require.Equal(t, "changed", reason(bought.sub.ID))
	require.Equal(t, "unused", reason(lapsed.sub.ID))

	var boughtExpiry time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT expires_at FROM user_subscriptions WHERE id = $1`, bought.sub.ID).Scan(&boughtExpiry))
	require.True(t, boughtExpiry.After(time.Now().Add(30*24*time.Hour)), "the purchase's expiry is untouched")

	pending, err = repo.PendingForUsers(ctx, userIDs)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Contains(t, pending, idle.sub.ID)
}
