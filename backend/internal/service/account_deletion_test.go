//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountDeletionAdminStub struct {
	AdminService
	deleted []int64
	err     error
}

func (s *accountDeletionAdminStub) DeleteUser(_ context.Context, id int64) error {
	if s.err != nil {
		return s.err
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type accountDeletionRepoStub struct {
	pending      bool
	steps        []string
	anonymizeErr error
}

func (s *accountDeletionRepoStub) HasPendingWithdrawal(context.Context, int64) (bool, error) {
	return s.pending, nil
}

func (s *accountDeletionRepoStub) PurgeOwnedContent(context.Context, int64) error {
	s.steps = append(s.steps, "purge")
	return nil
}

func (s *accountDeletionRepoStub) Anonymize(context.Context, int64) error {
	s.steps = append(s.steps, "anonymize")
	return s.anonymizeErr
}

func TestAccountDeletion(t *testing.T) {
	ctx := context.Background()
	newSvc := func(u *User) (*AccountDeletionService, *accountDeletionAdminStub, *accountDeletionRepoStub) {
		admin := &accountDeletionAdminStub{}
		repo := &accountDeletionRepoStub{}
		return NewAccountDeletionService(&userRepoStub{user: u}, admin, repo), admin, repo
	}

	t.Run("deletes, purges and anonymizes", func(t *testing.T) {
		svc, admin, repo := newSvc(&User{ID: 7, Role: RoleUser})
		require.NoError(t, svc.DeleteSelf(ctx, 7, " 注销账号 "))
		require.Equal(t, []int64{7}, admin.deleted)
		require.Equal(t, []string{"purge", "anonymize"}, repo.steps)
	})

	t.Run("needs the confirm phrase", func(t *testing.T) {
		svc, admin, _ := newSvc(&User{ID: 7, Role: RoleUser})
		require.ErrorIs(t, svc.DeleteSelf(ctx, 7, "yes"), ErrAccountDeletionConfirm)
		require.ErrorIs(t, svc.DeleteSelf(ctx, 7, "delete"), ErrAccountDeletionConfirm)
		require.Empty(t, admin.deleted)
	})

	t.Run("the English phrase works too", func(t *testing.T) {
		svc, admin, _ := newSvc(&User{ID: 7, Role: RoleUser})
		require.NoError(t, svc.DeleteSelf(ctx, 7, AccountDeletionConfirmEN))
		require.Equal(t, []int64{7}, admin.deleted)
	})

	t.Run("admins cannot delete themselves", func(t *testing.T) {
		svc, admin, _ := newSvc(&User{ID: 1, Role: RoleAdmin})
		require.ErrorIs(t, svc.DeleteSelf(ctx, 1, AccountDeletionConfirm), ErrAccountDeletionAdmin)
		require.Empty(t, admin.deleted)
	})

	t.Run("waits for a pending withdrawal", func(t *testing.T) {
		svc, admin, repo := newSvc(&User{ID: 7, Role: RoleUser})
		repo.pending = true
		require.ErrorIs(t, svc.DeleteSelf(ctx, 7, AccountDeletionConfirm), ErrAccountDeletionPending)
		require.Empty(t, admin.deleted)
		require.Empty(t, repo.steps)
	})

	t.Run("a failed delete is reported and not anonymized", func(t *testing.T) {
		svc, admin, repo := newSvc(&User{ID: 7, Role: RoleUser})
		admin.err = errors.New("db down")
		require.Error(t, svc.DeleteSelf(ctx, 7, AccountDeletionConfirm))
		require.Equal(t, []string{"purge"}, repo.steps)
	})

	t.Run("a failed anonymize still reports success", func(t *testing.T) {
		svc, admin, repo := newSvc(&User{ID: 7, Role: RoleUser})
		repo.anonymizeErr = errors.New("db down")
		require.NoError(t, svc.DeleteSelf(ctx, 7, AccountDeletionConfirm))
		require.Equal(t, []int64{7}, admin.deleted)
	})
}
