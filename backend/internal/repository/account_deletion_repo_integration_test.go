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

func TestAccountDeletionRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "del-" + suffix + "@del.test", Username: "del" + suffix[len(suffix)-4:], Notes: "vip"})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	})
	repo := NewAccountDeletionRepository(integrationDB)

	tutors := NewTutorRepository(integrationDB)
	tu := &service.Tutor{UserID: user.ID, KeyID: 9, Name: "助教", Template: "qa", AnswerMode: "guide", ModelTier: "standard", ShareCode: "d" + suffix[len(suffix)-7:],
		PassCode: "8023", PerStudentDay: 20, DailyCap: 300, Enabled: true}
	require.NoError(t, tutors.Create(ctx, tu))

	pending, err := repo.HasPendingWithdrawal(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, pending)

	require.NoError(t, repo.PurgeOwnedContent(ctx, user.ID))
	list, err := tutors.List(ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, list)

	// Only a soft-deleted row is anonymized.
	require.NoError(t, repo.Anonymize(ctx, user.ID))
	var email string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, user.ID).Scan(&email))
	require.Equal(t, "del-"+suffix+"@del.test", email)

	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, user.ID)
	require.NoError(t, err)
	require.NoError(t, repo.Anonymize(ctx, user.ID))
	var username, notes string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT email, username, notes FROM users WHERE id = $1`, user.ID).Scan(&email, &username, &notes))
	require.Equal(t, fmt.Sprintf("deleted-%d@deleted.invalid", user.ID), email)
	require.Empty(t, username)
	require.Empty(t, notes)
}
