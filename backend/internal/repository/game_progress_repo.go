package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type gameProgressRepository struct {
	db *sql.DB
}

// NewGameProgressRepository stores 小游戏 cloud saves and best scores (game_saves / game_scores,
// migration 329).
func NewGameProgressRepository(db *sql.DB) service.GameProgressRepository {
	return &gameProgressRepository{db: db}
}

func (r *gameProgressRepository) GetSave(ctx context.Context, userID int64, game string) (*service.GameSave, error) {
	var data []byte
	var at time.Time
	err := r.db.QueryRowContext(ctx, `SELECT data, updated_at FROM game_saves WHERE user_id = $1 AND game = $2`, userID, game).Scan(&data, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &service.GameSave{Data: json.RawMessage(data), UpdatedAt: &at}, nil
}

func (r *gameProgressRepository) PutSave(ctx context.Context, userID int64, game string, data json.RawMessage) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO game_saves (user_id, game, data, updated_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, game) DO UPDATE SET data = EXCLUDED.data, updated_at = NOW()
		RETURNING updated_at`, userID, game, []byte(data)).Scan(&at)
	return at, err
}

func (r *gameProgressRepository) BoardsOf(ctx context.Context, userID int64, game string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT board FROM game_scores WHERE user_id = $1 AND game = $2`, userID, game)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// SubmitScore keeps the best score; updated_at is when that best was reached, which breaks ties.
func (r *gameProgressRepository) SubmitScore(ctx context.Context, userID int64, game, board string, score int64, detail json.RawMessage) (int64, bool, error) {
	var best int64
	var improved bool
	err := r.db.QueryRowContext(ctx, `
		WITH prev AS (SELECT score FROM game_scores WHERE user_id = $1 AND game = $2 AND board = $3),
		up AS (
			INSERT INTO game_scores (user_id, game, board, score, detail) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, game, board) DO UPDATE
				SET score = EXCLUDED.score, detail = EXCLUDED.detail, updated_at = NOW()
				WHERE game_scores.score < EXCLUDED.score
			RETURNING score
		)
		SELECT COALESCE((SELECT score FROM up), (SELECT score FROM prev)),
		       EXISTS (SELECT 1 FROM up) AND COALESCE((SELECT score FROM prev), -1) < $4`,
		userID, game, board, score, []byte(detail)).Scan(&best, &improved)
	return best, improved, err
}

const gameScoreSelect = `
	SELECT s.user_id, COALESCE(u.username, ''), COALESCE(u.email, ''), s.score, s.detail, s.updated_at,
	       RANK() OVER (ORDER BY s.score DESC, s.updated_at ASC, s.user_id ASC) AS rk
	FROM game_scores s JOIN users u ON u.id = s.user_id
	WHERE s.game = $1 AND s.board = $2 AND u.deleted_at IS NULL`

func scanGameScore(sc interface{ Scan(...any) error }) (service.GameScoreRow, error) {
	var row service.GameScoreRow
	var detail []byte
	err := sc.Scan(&row.UserID, &row.Username, &row.Email, &row.Score, &detail, &row.UpdatedAt, &row.Rank)
	row.Detail = json.RawMessage(detail)
	return row, err
}

func (r *gameProgressRepository) Leaderboard(ctx context.Context, game, board string, limit int) ([]service.GameScoreRow, error) {
	rows, err := r.db.QueryContext(ctx, gameScoreSelect+` ORDER BY rk LIMIT $3`, game, board, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GameScoreRow
	for rows.Next() {
		row, err := scanGameScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *gameProgressRepository) Players(ctx context.Context, game, board string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM game_scores s JOIN users u ON u.id = s.user_id
		WHERE s.game = $1 AND s.board = $2 AND u.deleted_at IS NULL`, game, board).Scan(&n)
	return n, err
}

func (r *gameProgressRepository) EntryFor(ctx context.Context, userID int64, game, board string) (*service.GameScoreRow, error) {
	row, err := scanGameScore(r.db.QueryRowContext(ctx, `SELECT * FROM (`+gameScoreSelect+`) t WHERE t.user_id = $3`, game, board, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
