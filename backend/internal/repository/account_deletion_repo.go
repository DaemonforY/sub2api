package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountDeletionRepository struct {
	db *sql.DB
}

func NewAccountDeletionRepository(db *sql.DB) service.AccountDeletionRepository {
	return &accountDeletionRepository{db: db}
}

func (r *accountDeletionRepository) HasPendingWithdrawal(ctx context.Context, userID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM affiliate_withdrawals WHERE user_id = $1 AND status = 'pending'`, userID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check pending withdrawal: %w", err)
	}
	return n > 0, nil
}

func (r *accountDeletionRepository) PurgeOwnedContent(ctx context.Context, userID int64) error {
	// Users are soft-deleted, so ON DELETE CASCADE never fires: remove these explicitly.
	// Tutor materials, students and messages cascade from tutors.
	for _, q := range []string{
		`DELETE FROM tutors WHERE user_id = $1`,
		`DELETE FROM article_projects WHERE user_id = $1`,
		`DELETE FROM user_attribute_values WHERE user_id = $1`,
	} {
		if _, err := r.db.ExecContext(ctx, q, userID); err != nil {
			return fmt.Errorf("purge owned content: %w", err)
		}
	}
	return nil
}

func (r *accountDeletionRepository) Anonymize(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE users SET
    email = 'deleted-' || id || '@deleted.invalid',
    username = '',
    notes = '',
    password_hash = '',
    totp_secret_encrypted = NULL,
    totp_enabled = FALSE,
    totp_enabled_at = NULL,
    balance_notify_enabled = FALSE,
    balance_notify_extra_emails = '[]',
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NOT NULL`, userID)
	if err != nil {
		return fmt.Errorf("anonymize user: %w", err)
	}
	return nil
}
