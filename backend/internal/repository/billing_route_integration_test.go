//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBillingRoute(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	settings := NewSettingRepository(integrationEntClient)
	previous, _ := settings.GetValue(ctx, service.SettingKeySmartBillingEnabled)
	require.NoError(t, settings.Set(ctx, service.SettingKeySmartBillingEnabled, "true"))
	t.Cleanup(func() { _ = settings.Set(ctx, service.SettingKeySmartBillingEnabled, previous) })
	// The groups are committed; GroupRepoSuite's filters would see them if left behind.
	committed := trackCommitted(t)
	groupRepo := NewGroupRepository(integrationEntClient, integrationDB)
	subs := service.NewSubscriptionService(groupRepo, NewUserSubscriptionRepository(integrationEntClient), nil, integrationEntClient, &config.Config{})
	router := service.NewBillingRouteService(subs, groupRepo, settings)

	limit := func(v float64) *float64 { return &v }
	mkGroup := func(name, typ string, exclusive bool, sortOrder int, daily *float64) *service.Group {
		g := mustCreateGroup(t, integrationEntClient, &service.Group{Name: name + "-" + sfx, Platform: service.PlatformOpenAI,
			SubscriptionType: typ, IsExclusive: exclusive, RateMultiplier: 1, DailyLimitUSD: daily})
		_, err := integrationDB.ExecContext(ctx, `UPDATE groups SET sort_order = $1 WHERE id = $2`, sortOrder, g.ID)
		require.NoError(t, err)
		g.SortOrder = sortOrder
		committed.groups = append(committed.groups, g.ID)
		return g
	}
	// Lowest sort orders so other tests' groups in the shared database don't get picked.
	hidden := mkGroup("paygo-exclusive", service.SubscriptionTypeStandard, true, -3000, nil)
	paygo := mkGroup("paygo", service.SubscriptionTypeStandard, false, -2000, nil)
	subA := mkGroup("sub-a", service.SubscriptionTypeSubscription, false, -1900, limit(10))
	subB := mkGroup("sub-b", service.SubscriptionTypeSubscription, false, -1800, limit(10))

	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "route-" + sfx + "@test.local", Username: "route" + sfx[len(sfx)-6:]})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "route2-" + sfx + "@test.local", Username: "route2" + sfx[len(sfx)-6:]})
	committed.users = append(committed.users, user.ID, other.ID)
	start := time.Now().Add(-time.Hour)
	sa := mustCreateSubscription(t, integrationEntClient, &service.UserSubscription{UserID: user.ID, GroupID: subA.ID, StartsAt: start, ExpiresAt: time.Now().AddDate(0, 0, 10)})
	useUp := func(subID int64, used float64) {
		_, err := integrationDB.ExecContext(ctx, `UPDATE user_subscriptions SET daily_usage_usd = $1, daily_window_start = NOW() WHERE id = $2`, used, subID)
		require.NoError(t, err)
	}
	useUp(sa.ID, 2)
	keyOn := func(g *service.Group, u *service.User) *service.APIKey {
		return &service.APIKey{GroupID: &g.ID, Group: g, User: u, UserID: u.ID}
	}
	routeID := func(g *service.Group) int64 {
		if g == nil {
			return 0
		}
		return g.ID
	}

	// A subscription key with quota left stays where it is.
	require.Nil(t, router.Route(ctx, keyOn(subA, user), true, true))
	// A pay-as-you-go key uses the subscription for text requests only.
	require.Equal(t, subA.ID, routeID(router.Route(ctx, keyOn(paygo, user), true, true)))
	require.Nil(t, router.Route(ctx, keyOn(paygo, user), false, true))

	// Today's quota used up: the subscription key falls back to balance, if there is balance.
	useUp(sa.ID, 12)
	require.Equal(t, paygo.ID, routeID(router.Route(ctx, keyOn(subA, user), true, true)), "exclusive groups the user can't use are skipped")
	require.Nil(t, router.Route(ctx, keyOn(subA, user), true, false))
	require.Nil(t, router.Route(ctx, keyOn(paygo, user), true, true))

	// A second subscription with quota is preferred over balance.
	mustCreateSubscription(t, integrationEntClient, &service.UserSubscription{UserID: user.ID, GroupID: subB.ID, StartsAt: start, ExpiresAt: time.Now().AddDate(0, 0, 5)})
	router.Forget(user.ID)
	require.Equal(t, subB.ID, routeID(router.Route(ctx, keyOn(subA, user), true, true)))
	require.Equal(t, subB.ID, routeID(router.Route(ctx, keyOn(paygo, user), true, true)))

	// No subscription at all: a subscription key goes to balance.
	require.Equal(t, paygo.ID, routeID(router.Route(ctx, keyOn(subA, other), true, true)))
	_ = hidden

	// Turned off: keys keep their own group.
	require.NoError(t, settings.Set(ctx, service.SettingKeySmartBillingEnabled, "false"))
	off := service.NewBillingRouteService(subs, groupRepo, settings)
	require.False(t, off.Enabled(ctx))
	require.Nil(t, off.Route(ctx, keyOn(subA, other), true, true))
}

func TestStarterKey(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	settings := NewSettingRepository(integrationEntClient)
	groupRepo := NewGroupRepository(integrationEntClient, integrationDB)
	userRepo := NewUserRepository(integrationEntClient, integrationDB)
	subRepo := NewUserSubscriptionRepository(integrationEntClient)
	subs := service.NewSubscriptionService(groupRepo, subRepo, nil, integrationEntClient, &config.Config{})
	router := service.NewBillingRouteService(subs, groupRepo, settings)
	keys := service.NewAPIKeyService(NewAPIKeyRepository(integrationEntClient, integrationDB), userRepo, groupRepo, subRepo, NewUserGroupRateRepository(integrationDB), nil, &config.Config{})
	starter := service.NewStarterKeyService(keys, userRepo, router)
	// Committed rows; other suites list groups and keys, so remove them afterwards.
	committed := trackCommitted(t)

	paygo := mustCreateGroup(t, integrationEntClient, &service.Group{Name: "starter-paygo-" + sfx, Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1})
	_, err := integrationDB.ExecContext(ctx, `UPDATE groups SET sort_order = -5000 WHERE id = $1`, paygo.ID)
	require.NoError(t, err)
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "starter-" + sfx + "@test.local", Username: "starter" + sfx[len(sfx)-6:]})
	committed.groups = append(committed.groups, paygo.ID)
	committed.users = append(committed.users, user.ID)

	k, created, err := starter.Ensure(ctx, user.ID)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, service.StarterKeyName, k.Name)
	require.NotNil(t, k.GroupID)
	require.Equal(t, paygo.ID, *k.GroupID)
	require.NotEmpty(t, k.Key)

	again, created, err := starter.Ensure(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, k.ID, again.ID)
	var n int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND deleted_at IS NULL`, user.ID).Scan(&n))
	require.Equal(t, 1, n)
}
