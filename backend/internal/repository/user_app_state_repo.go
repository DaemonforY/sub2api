package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userAppStateRepository struct {
	db *sql.DB
}

// NewUserAppStateRepository creates the synced app-state repository (raw SQL).
func NewUserAppStateRepository(db *sql.DB) service.UserAppStateRepository {
	return &userAppStateRepository{db: db}
}

func (r *userAppStateRepository) Get(ctx context.Context, userID int64, namespace string) (*service.UserAppState, error) {
	return scanUserAppState(r.db.QueryRowContext(ctx,
		`SELECT namespace, value, version, updated_at FROM user_app_states WHERE user_id = $1 AND namespace = $2`,
		userID, namespace,
	))
}

func (r *userAppStateRepository) CompareAndSet(ctx context.Context, userID int64, namespace string, value json.RawMessage, baseVersion int64) (*service.UserAppState, bool, error) {
	var row *sql.Row
	if baseVersion == 0 {
		row = r.db.QueryRowContext(ctx, `
INSERT INTO user_app_states (user_id, namespace, value, version, updated_at)
VALUES ($1, $2, $3::jsonb, 1, NOW())
ON CONFLICT (user_id, namespace) DO NOTHING
RETURNING namespace, value, version, updated_at`, userID, namespace, string(value))
	} else {
		row = r.db.QueryRowContext(ctx, `
UPDATE user_app_states
SET value = $3::jsonb, version = version + 1, updated_at = NOW()
WHERE user_id = $1 AND namespace = $2 AND version = $4
RETURNING namespace, value, version, updated_at`, userID, namespace, string(value), baseVersion)
	}
	state, err := scanUserAppState(row)
	if err != nil {
		return nil, false, err
	}
	if state != nil {
		return state, true, nil
	}
	// Someone else wrote first (or the row vanished): hand back what is stored now.
	current, err := r.Get(ctx, userID, namespace)
	if err != nil {
		return nil, false, err
	}
	if current == nil {
		current = &service.UserAppState{Namespace: namespace, Value: json.RawMessage("{}"), Version: 0}
	}
	return current, false, nil
}

func scanUserAppState(row *sql.Row) (*service.UserAppState, error) {
	var (
		state     service.UserAppState
		value     []byte
		updatedAt time.Time
	)
	err := row.Scan(&state.Namespace, &value, &state.Version, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read user app state: %w", err)
	}
	state.Value = json.RawMessage(value)
	state.UpdatedAt = &updatedAt
	return &state, nil
}

// ---------------------------------------------------------------------------
// Synced images (user_app_blobs)
// ---------------------------------------------------------------------------

type userAppBlobRepository struct {
	db *sql.DB
}

// NewUserAppBlobRepository creates the synced image metadata repository (raw SQL).
func NewUserAppBlobRepository(db *sql.DB) service.UserAppBlobRepository {
	return &userAppBlobRepository{db: db}
}

const userAppBlobColumns = `id, user_id, sha256, mime_type, size_bytes, created_at, last_used_at`

func scanUserAppBlob(scan func(dest ...any) error) (*service.UserAppBlob, error) {
	var b service.UserAppBlob
	if err := scan(&b.ID, &b.UserID, &b.SHA256, &b.MimeType, &b.SizeBytes, &b.CreatedAt, &b.LastUsedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *userAppBlobRepository) queryOne(ctx context.Context, query string, args ...any) (*service.UserAppBlob, error) {
	blob, err := scanUserAppBlob(r.db.QueryRowContext(ctx, query, args...).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read user app blob: %w", err)
	}
	return blob, nil
}

func (r *userAppBlobRepository) FindBlobBySHA(ctx context.Context, userID int64, sha string) (*service.UserAppBlob, error) {
	return r.queryOne(ctx, `SELECT `+userAppBlobColumns+` FROM user_app_blobs WHERE user_id = $1 AND sha256 = $2`, userID, sha)
}

func (r *userAppBlobRepository) GetBlob(ctx context.Context, userID int64, id string) (*service.UserAppBlob, error) {
	return r.queryOne(ctx, `SELECT `+userAppBlobColumns+` FROM user_app_blobs WHERE user_id = $1 AND id = $2`, userID, id)
}

func (r *userAppBlobRepository) InsertBlob(ctx context.Context, blob *service.UserAppBlob) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO user_app_blobs (id, user_id, sha256, mime_type, size_bytes, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING created_at, last_used_at`, blob.ID, blob.UserID, blob.SHA256, blob.MimeType, blob.SizeBytes).Scan(&blob.CreatedAt, &blob.LastUsedAt)
	if err != nil {
		return fmt.Errorf("insert user app blob: %w", err)
	}
	return nil
}

func (r *userAppBlobRepository) TouchBlob(ctx context.Context, userID int64, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_app_blobs SET last_used_at = NOW() WHERE user_id = $1 AND id = $2`, userID, id)
	return err
}

func (r *userAppBlobRepository) BlobsOverQuota(ctx context.Context, userID int64, maxCount int, maxTotal int64) ([]service.UserAppBlob, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+userAppBlobColumns+` FROM (
    SELECT *, ROW_NUMBER() OVER w AS rn, SUM(size_bytes) OVER w AS running
    FROM user_app_blobs WHERE user_id = $1
    WINDOW w AS (ORDER BY last_used_at DESC, created_at DESC)
) ranked
WHERE rn > $2 OR running > $3
ORDER BY last_used_at ASC, created_at ASC`, userID, maxCount, maxTotal)
	if err != nil {
		return nil, fmt.Errorf("list user app blobs over quota: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []service.UserAppBlob
	for rows.Next() {
		blob, scanErr := scanUserAppBlob(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan user app blob: %w", scanErr)
		}
		out = append(out, *blob)
	}
	return out, rows.Err()
}

func (r *userAppBlobRepository) DeleteBlobs(ctx context.Context, userID int64, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_app_blobs WHERE user_id = $1 AND id = ANY($2)`, userID, pq.Array(ids))
	return err
}
