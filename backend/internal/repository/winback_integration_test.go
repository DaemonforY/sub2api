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

func TestWinbackRepositoryDue(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewWinbackRepository(integrationDB)
	acc := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "wb-acc-" + sfx}))
	now := time.Now()
	mk := func(tag string, calls ...time.Duration) *service.User {
		u := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "wb-" + tag + "-" + sfx + "@test.local", Username: "wb" + tag + sfx[len(sfx)-6:]}))
		if len(calls) == 0 {
			return u
		}
		key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Key: "sk-wb-" + tag + sfx})
		for i, ago := range calls {
			_, err := integrationEntClient.UsageLog.Create().SetUserID(u.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).
				SetRequestID(fmt.Sprintf("wb-%s-%d-%s", tag, i, sfx)).SetModel("gpt-5").SetCreatedAt(now.Add(-ago)).Save(ctx)
			require.NoError(t, err)
		}
		return u
	}
	day := 24 * time.Hour
	lapsed := mk("lapsed", 40*day, 20*day) // last call 20 days ago: due
	active := mk("active", 30*day, 2*day)  // still using it
	ancient := mk("ancient", 90*day)       // gone too long
	never := mk("never")                   // never called

	due, err := repo.DueLapsedUsers(ctx, now.Add(-60*day), now.Add(-14*day), 1000)
	require.NoError(t, err)
	got := map[int64]service.LapsedUser{}
	for _, u := range due {
		got[u.UserID] = u
	}
	require.Contains(t, got, lapsed.ID)
	require.WithinDuration(t, now.Add(-20*day), got[lapsed.ID].LastUsed, time.Minute)
	for _, u := range []*service.User{active, ancient, never} {
		require.NotContains(t, got, u.ID, u.Email)
	}
}
