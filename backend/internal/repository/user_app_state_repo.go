package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
