//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSignupFingerprintRepository_FindRecentTrial(t *testing.T) {
	ctx := context.Background()
	repo := NewSignupFingerprintRepository(integrationDB)
	base := time.Now().UnixNano() % 1_000_000_000
	ids := []int64{-base - 1, -base - 2, -base - 3, -base - 4}
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(context.Background(), `DELETE FROM signup_fingerprints WHERE user_id = ANY($1)`, pq.Array(ids))
		require.NoError(t, err)
	})
	vid := "it-visitor-" + time.Now().Format("150405.000000")
	since := time.Now().Add(-time.Hour)

	got, err := repo.FindRecentTrial(ctx, vid, since)
	require.NoError(t, err)
	require.Zero(t, got.UserID)

	require.NoError(t, repo.Save(ctx, service.SignupFingerprint{UserID: ids[0], VisitorID: vid, IPPrefix: "198.18.7.0", TrialGranted: true}))
	require.NoError(t, repo.Save(ctx, service.SignupFingerprint{UserID: ids[1], VisitorID: vid + "-b", IPPrefix: "198.18.9.0", TrialGranted: false}))
	// Saving the same user again keeps the first row.
	require.NoError(t, repo.Save(ctx, service.SignupFingerprint{UserID: ids[0], VisitorID: "other", IPPrefix: "", TrialGranted: false}))

	got, err = repo.FindRecentTrial(ctx, vid, since)
	require.NoError(t, err)
	require.Equal(t, service.FingerprintMatch{UserID: ids[0], ByVisitor: true}, got, "same browser")
	got, err = repo.FindRecentTrial(ctx, vid+"-b", since)
	require.NoError(t, err)
	require.Zero(t, got.UserID, "a sign-up that got no trial never matches")
	got, err = repo.FindRecentTrial(ctx, "", since)
	require.NoError(t, err)
	require.Zero(t, got.UserID, "empty values never match")
	got, err = repo.FindRecentTrial(ctx, vid, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, got.UserID, "outside the window")
}
