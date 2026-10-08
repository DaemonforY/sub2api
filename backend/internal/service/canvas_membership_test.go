//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memMembershipRepo struct {
	until  map[int64]time.Time
	terms  map[int64]bool
	saves  []CanvasUnmarkedSave
	now    func() time.Time
	pruned int
}

func newMemMembershipRepo(now func() time.Time) *memMembershipRepo {
	return &memMembershipRepo{until: map[int64]time.Time{}, terms: map[int64]bool{}, now: now}
}

func (r *memMembershipRepo) Until(_ context.Context, userID int64) (*time.Time, error) {
	if u, ok := r.until[userID]; ok {
		return &u, nil
	}
	return nil, nil
}

func (r *memMembershipRepo) Extend(_ context.Context, userID int64, days int, acceptTerms bool) (time.Time, error) {
	base := r.now()
	if u, ok := r.until[userID]; ok && u.After(base) {
		base = u
	}
	r.until[userID] = base.AddDate(0, 0, days)
	if acceptTerms {
		r.terms[userID] = true
	}
	return r.until[userID], nil
}

func (r *memMembershipRepo) Shorten(_ context.Context, userID int64, days int) error {
	if u, ok := r.until[userID]; ok {
		next := u.AddDate(0, 0, -days)
		if next.Before(r.now()) {
			next = r.now()
		}
		r.until[userID] = next
	}
	return nil
}

func (r *memMembershipRepo) AcceptTerms(_ context.Context, userID int64) error {
	r.terms[userID] = true
	if _, ok := r.until[userID]; !ok {
		r.until[userID] = r.now()
	}
	return nil
}

func (r *memMembershipRepo) TermsAccepted(_ context.Context, userID int64) (bool, error) {
	return r.terms[userID], nil
}

func (r *memMembershipRepo) LogUnmarkedSave(_ context.Context, save CanvasUnmarkedSave) error {
	r.saves = append(r.saves, save)
	return nil
}

func (r *memMembershipRepo) PruneUnmarkedSaves(context.Context, time.Time) (int64, error) {
	r.pruned++
	return 0, nil
}

func (r *memMembershipRepo) ListMembers(context.Context, bool, int) ([]CanvasMember, error) {
	return nil, nil
}

type memSettings struct {
	SettingRepository
	values map[string]string
}

func (s *memSettings) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", errors.New("setting not found")
}

func (s *memSettings) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func newTestMembership() (*CanvasMembershipService, *memMembershipRepo, *time.Time) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	repo := newMemMembershipRepo(clock)
	svc := NewCanvasMembershipService(repo, &memSettings{values: map[string]string{}})
	svc.now = clock
	return svc, repo, &now
}

func TestCanvasMembershipConfigAssignsStableIDs(t *testing.T) {
	svc, _, _ := newTestMembership()
	ctx := context.Background()

	cfg, err := svc.SaveConfig(ctx, CanvasMembershipConfig{Enabled: true, Plans: []CanvasMembershipPlan{{Name: " 月卡 ", Days: 30, Price: 19.9}, {Name: "年卡", Days: 365, Price: 168}}})
	require.NoError(t, err)
	require.Equal(t, int64(1), cfg.Plans[0].ID)
	require.Equal(t, int64(2), cfg.Plans[1].ID)
	require.Equal(t, "月卡", cfg.Plans[0].Name)

	// Dropping the first plan and adding one keeps id 2 and never reuses id 1's slot for another plan.
	cfg, err = svc.SaveConfig(ctx, CanvasMembershipConfig{Enabled: true, Plans: []CanvasMembershipPlan{{ID: 2, Name: "年卡", Days: 365, Price: 158}, {Name: "季卡", Days: 90, Price: 49}}})
	require.NoError(t, err)
	require.Equal(t, int64(2), cfg.Plans[0].ID)
	require.Equal(t, int64(3), cfg.Plans[1].ID)

	plan, err := svc.PlanForOrder(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, 158.0, plan.Price)
	_, err = svc.PlanForOrder(ctx, 1)
	require.ErrorIs(t, err, ErrCanvasMembershipPlan)
}

func TestCanvasMembershipConfigRejectsBadPlans(t *testing.T) {
	svc, _, _ := newTestMembership()
	ctx := context.Background()
	for _, cfg := range []CanvasMembershipConfig{
		{Enabled: true},
		{Plans: []CanvasMembershipPlan{{Name: "", Days: 30, Price: 10}}},
		{Plans: []CanvasMembershipPlan{{Name: "月卡", Days: 0, Price: 10}}},
		{Plans: []CanvasMembershipPlan{{Name: "月卡", Days: 30, Price: 0}}},
	} {
		_, err := svc.SaveConfig(ctx, cfg)
		require.Error(t, err)
	}
}

func TestCanvasMembershipSalesOffRejectsOrders(t *testing.T) {
	svc, _, _ := newTestMembership()
	ctx := context.Background()
	_, err := svc.SaveConfig(ctx, CanvasMembershipConfig{Enabled: false, Plans: []CanvasMembershipPlan{{Name: "月卡", Days: 30, Price: 19.9}}})
	require.NoError(t, err)
	require.False(t, svc.OnSale(ctx))
	_, err = svc.PlanForOrder(ctx, 1)
	require.ErrorIs(t, err, ErrCanvasMembershipUnavailable)
}

func TestCanvasMembershipExtendRefundAndExpiry(t *testing.T) {
	svc, _, now := newTestMembership()
	ctx := context.Background()

	until, err := svc.Until(ctx, 7)
	require.NoError(t, err)
	require.Nil(t, until)

	require.NoError(t, svc.FulfillOrder(ctx, 7, 30))
	require.NoError(t, svc.FulfillOrder(ctx, 7, 30))
	until, err = svc.Until(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, now.AddDate(0, 0, 60), *until)

	require.NoError(t, svc.RefundOrder(ctx, 7, 30))
	until, _ = svc.Until(ctx, 7)
	require.Equal(t, now.AddDate(0, 0, 30), *until)

	// Refunding more than is left ends the membership now, not in the past.
	require.NoError(t, svc.RefundOrder(ctx, 7, 90))
	until, err = svc.Until(ctx, 7)
	require.NoError(t, err)
	require.Nil(t, until)
}

func TestCanvasMembershipTermsAndUnmarkedSaves(t *testing.T) {
	svc, repo, _ := newTestMembership()
	ctx := context.Background()

	require.ErrorIs(t, svc.RequireTerms(ctx, 7), ErrCanvasMembershipTerms)
	require.NoError(t, svc.AcceptTerms(ctx, 7))
	require.NoError(t, svc.RequireTerms(ctx, 7))

	// Accepting the terms alone does not make a member, so saves stay watermarked.
	err := svc.RecordUnmarkedSave(ctx, CanvasUnmarkedSave{UserID: 7, Width: 1024, Height: 1536})
	require.Error(t, err)
	require.Empty(t, repo.saves)

	_, err = svc.Grant(ctx, 7, 7)
	require.NoError(t, err)
	require.NoError(t, svc.RecordUnmarkedSave(ctx, CanvasUnmarkedSave{UserID: 7, Width: -5, Height: 999999, ClientIP: "1.2.3.4"}))
	require.Len(t, repo.saves, 1)
	require.Equal(t, 0, repo.saves[0].Width)
	require.Equal(t, 100000, repo.saves[0].Height)

	_, err = svc.Grant(ctx, 7, 0)
	require.Error(t, err)
}
