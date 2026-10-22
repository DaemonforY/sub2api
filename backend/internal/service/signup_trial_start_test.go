//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memTrialStartRepo struct {
	created  []PendingSignupTrial
	users    map[int64]int64 // subscription -> user
	started  []SignupTrialStarted
	closedAt time.Time
}

func (m *memTrialStartRepo) Create(_ context.Context, subID, userID int64, days int) error {
	until := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	if m.users == nil {
		m.users = map[int64]int64{}
	}
	m.users[subID] = userID
	m.created = append(m.created, PendingSignupTrial{SubscriptionID: subID, TrialDays: days, PendingExpiresAt: until})
	return nil
}
func (m *memTrialStartRepo) StartUsed(context.Context, int) ([]SignupTrialStarted, error) {
	return m.started, nil
}
func (m *memTrialStartRepo) CloseStale(_ context.Context, now time.Time) (int64, error) {
	m.closedAt = now
	return 0, nil
}
func (m *memTrialStartRepo) PendingForUsers(_ context.Context, userIDs []int64) (map[int64]PendingSignupTrial, error) {
	out := map[int64]PendingSignupTrial{}
	for _, p := range m.created {
		for _, uid := range userIDs {
			if m.users[p.SubscriptionID] == uid {
				out[p.SubscriptionID] = p
			}
		}
	}
	return out, nil
}

// assigner that hands out subscription IDs and a real expiry, like SubscriptionService.
type trialAssignerStub struct {
	calls    []AssignSubscriptionInput
	extended bool
}

func (s *trialAssignerStub) AssignOrExtendSubscription(_ context.Context, in *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	s.calls = append(s.calls, *in)
	return &UserSubscription{ID: int64(100 + len(s.calls)), UserID: in.UserID, GroupID: in.GroupID,
		ExpiresAt: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}, s.extended, nil
}

func newTrialSignupService(t *testing.T, subs string) (*AuthService, *trialAssignerStub, *memTrialStartRepo) {
	t.Helper()
	repo := &userRepoStub{nextID: 42}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                subs,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)
	assigner := &trialAssignerStub{}
	svc.defaultSubAssigner = assigner
	trials := &memTrialStartRepo{}
	svc.SetSignupTrialStarts(NewSignupTrialStartService(trials, nil, nil)) // no growth service: default 7 days
	return svc, assigner, trials
}

func TestSignupTrialStartsOnFirstCall(t *testing.T) {
	svc, assigner, trials := newTrialSignupService(t, `[{"group_id":8,"validity_days":1},{"group_id":9,"validity_days":30}]`)
	_, user, err := svc.RegisterWithVerification(context.Background(), "a@test.com", "password", "", "", "", "")
	require.NoError(t, err)

	require.Len(t, assigner.calls, 2)
	require.Equal(t, AssignSubscriptionInput{UserID: user.ID, GroupID: 8, ValidityDays: 7, Notes: signupDefaultsNote}, assigner.calls[0],
		"the 1-day trial is granted for the 7-day unused window")
	require.Equal(t, 30, assigner.calls[1].ValidityDays, "trials as long as the window start at sign-up")
	require.Equal(t, []PendingSignupTrial{{SubscriptionID: 101, TrialDays: 1, PendingExpiresAt: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}}, trials.created)
}

func TestSignupTrialExtendedSubscriptionIsNotPending(t *testing.T) {
	svc, assigner, trials := newTrialSignupService(t, `[{"group_id":8,"validity_days":1}]`)
	assigner.extended = true // the account somehow already had this group
	_, _, err := svc.RegisterWithVerification(context.Background(), "a@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.Empty(t, trials.created)
}

func TestSignupTrialWindowOffStartsAtSignup(t *testing.T) {
	svc, assigner, trials := newTrialSignupService(t, `[{"group_id":8,"validity_days":1}]`)
	svc.SetSignupTrialStarts(nil)
	_, _, err := svc.RegisterWithVerification(context.Background(), "a@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, 1, assigner.calls[0].ValidityDays)
	require.Empty(t, trials.created)
}

func TestSignupTrialStartRunOnceClosesStale(t *testing.T) {
	repo := &memTrialStartRepo{started: []SignupTrialStarted{{SubscriptionID: 5, UserID: 7, GroupID: 8}}}
	s := NewSignupTrialStartService(repo, nil, nil)
	n, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.False(t, repo.closedAt.IsZero())
}

func TestMarkPendingTrials(t *testing.T) {
	repo := &memTrialStartRepo{}
	require.NoError(t, repo.Create(context.Background(), 11, 3, 1))
	s := NewSignupTrialStartService(repo, nil, nil)
	subs := []UserSubscription{{ID: 10, UserID: 3}, {ID: 11, UserID: 3}}
	markPendingTrials(context.Background(), s, 3, subs)
	require.Zero(t, subs[0].TrialPendingDays)
	require.Equal(t, 1, subs[1].TrialPendingDays)
}

func TestExpiryRemindersSkipPendingTrials(t *testing.T) {
	repo := &memTrialStartRepo{}
	require.NoError(t, repo.Create(context.Background(), 11, 3, 1))
	e := NewSubscriptionExpiryService(nil, time.Minute)
	e.SetPendingTrialLookup(NewSignupTrialStartService(repo, nil, nil))
	pending := e.pendingTrialsOf(context.Background(), []UserSubscription{{ID: 10, UserID: 3}, {ID: 11, UserID: 3}, {ID: 12, UserID: 4}})
	_, ok := pending[11]
	require.True(t, ok)
	require.Len(t, pending, 1)
}

func TestParseGrowthSettings_SignupTrialUnusedDays(t *testing.T) {
	require.Equal(t, 7, parseGrowthSettings(map[string]string{}).SignupTrialUnusedDays)
	require.Equal(t, 0, parseGrowthSettings(map[string]string{SettingKeyGrowthSignupTrialUnusedDays: "0"}).SignupTrialUnusedDays)
	require.Equal(t, 90, parseGrowthSettings(map[string]string{SettingKeyGrowthSignupTrialUnusedDays: "500"}).SignupTrialUnusedDays)
}
