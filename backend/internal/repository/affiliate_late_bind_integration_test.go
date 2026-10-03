//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateLateBindAndAdminSetInviter(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingRepository(integrationEntClient)
	previous, _ := settings.GetValue(ctx, service.SettingKeyAffiliateEnabled)
	require.NoError(t, settings.Set(ctx, service.SettingKeyAffiliateEnabled, "true"))
	t.Cleanup(func() { _ = settings.Set(ctx, service.SettingKeyAffiliateEnabled, previous) })
	repo := NewAffiliateRepository(integrationEntClient, integrationDB)
	svc := service.NewAffiliateService(repo, service.NewSettingService(settings, &config.Config{}), nil, nil)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := func(name string) *service.User {
		return mustCreateUser(t, integrationEntClient, &service.User{Email: name + "-" + suffix + "@test.local"})
	}
	code := func(u *service.User) string {
		s, err := svc.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		return s.AffCode
	}
	inviterOf := func(u *service.User) *int64 {
		s, err := svc.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		return s.InviterID
	}
	count := func(u *service.User) int {
		s, err := svc.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		return s.AffCount
	}
	alice, bob, carol := user("alice"), user("bob"), user("carol")

	// The sign-up form's check.
	require.NoError(t, svc.CheckCode(ctx, " "+code(alice)+" "))
	require.NoError(t, svc.CheckCode(ctx, strings.ToLower(code(alice))))
	require.ErrorIs(t, svc.CheckCode(ctx, "NOPE0000ZZZZ"), service.ErrAffiliateCodeInvalid)
	require.ErrorIs(t, svc.CheckCode(ctx, "!!"), service.ErrAffiliateCodeInvalid)

	now := time.Now()
	// Bob signed up yesterday without a code: he may add Alice's.
	require.ErrorIs(t, svc.BindInviterLate(ctx, bob.ID, now.Add(-24*time.Hour), code(bob)), service.ErrAffiliateSelfCode)
	require.ErrorIs(t, svc.BindInviterLate(ctx, bob.ID, now.Add(-8*24*time.Hour), code(alice)), service.ErrAffiliateLateBindExpired)
	require.NoError(t, svc.BindInviterLate(ctx, bob.ID, now.Add(-24*time.Hour), code(alice)))
	require.Equal(t, alice.ID, *inviterOf(bob))
	require.Equal(t, 1, count(alice))
	require.ErrorIs(t, svc.BindInviterLate(ctx, bob.ID, now.Add(-24*time.Hour), code(carol)), service.ErrAffiliateAlreadyBound)
	// Alice cannot then name Bob, whom she invited.
	require.ErrorIs(t, svc.BindInviterLate(ctx, alice.ID, now.Add(-24*time.Hour), code(bob)), service.ErrAffiliateCodeCycle)

	// The affiliate page offers the form only while unbound and within the window.
	detail, err := svc.GetAffiliateDetail(ctx, carol.ID)
	require.NoError(t, err)
	svc.FillLateBind(ctx, detail, now.Add(-2*24*time.Hour))
	require.True(t, detail.CanBindInviter)
	require.WithinDuration(t, now.Add(5*24*time.Hour), *detail.BindInviterDeadline, time.Minute)
	detail, err = svc.GetAffiliateDetail(ctx, bob.ID)
	require.NoError(t, err)
	svc.FillLateBind(ctx, detail, now.Add(-24*time.Hour))
	require.False(t, detail.CanBindInviter, "already has an inviter")

	// Admins move Bob to Carol (counts follow), then clear him.
	require.NoError(t, svc.AdminSetInviter(ctx, bob.ID, code(carol)))
	require.Equal(t, carol.ID, *inviterOf(bob))
	require.Equal(t, 0, count(alice))
	require.Equal(t, 1, count(carol))
	overview, err := svc.AdminGetUserOverview(ctx, bob.ID)
	require.NoError(t, err)
	require.Equal(t, carol.ID, overview.InviterID)
	require.Equal(t, code(carol), overview.InviterAffCode)
	require.NoError(t, svc.AdminSetInviter(ctx, bob.ID, code(carol)), "unchanged is fine")
	require.Equal(t, 1, count(carol))
	require.ErrorIs(t, svc.AdminSetInviter(ctx, bob.ID, code(bob)), service.ErrAffiliateSelfCode)
	require.NoError(t, svc.AdminSetInviter(ctx, bob.ID, ""))
	require.Nil(t, inviterOf(bob))
	require.Equal(t, 0, count(carol))

	// With invites turned off nothing binds.
	require.NoError(t, settings.Set(ctx, service.SettingKeyAffiliateEnabled, "false"))
	svc = service.NewAffiliateService(repo, service.NewSettingService(settings, &config.Config{}), nil, nil)
	require.ErrorIs(t, svc.CheckCode(ctx, code(alice)), service.ErrAffiliateDisabled)
	require.ErrorIs(t, svc.BindInviterLate(ctx, carol.ID, now, code(alice)), service.ErrAffiliateDisabled)
}
