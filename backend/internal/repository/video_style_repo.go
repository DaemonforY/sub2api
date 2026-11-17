package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 我的风格 style cards (migration 327).

var _ service.VideoStyleCardRepository = (*videoRepository)(nil)

const videoStyleColumns = `id, user_id, name, ratio, base_style, theme, common, character_desc, rules, COALESCE(source_project::text, ''), created_at, updated_at`

func scanVideoStyle(row videoScanner) (*service.VideoStyleCard, error) {
	var st service.VideoStyleCard
	var theme, rules []byte
	if err := row.Scan(&st.ID, &st.UserID, &st.Name, &st.Ratio, &st.BaseStyle, &theme, &st.Common, &st.Character, &rules, &st.SourceProject, &st.CreatedAt, &st.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(theme, &st.Theme)
	_ = json.Unmarshal(rules, &st.Rules)
	if st.Rules == nil {
		st.Rules = []string{}
	}
	return &st, nil
}

func (r *videoRepository) ListStyles(ctx context.Context, userID int64) ([]service.VideoStyleCard, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+videoStyleColumns+` FROM video_styles WHERE user_id = $1 ORDER BY updated_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoStyleCard{}
	for rows.Next() {
		st, err := scanVideoStyle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, rows.Err()
}

func (r *videoRepository) GetStyle(ctx context.Context, userID, id int64) (*service.VideoStyleCard, error) {
	st, err := scanVideoStyle(r.db.QueryRowContext(ctx, `SELECT `+videoStyleColumns+` FROM video_styles WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return st, err
}

func (r *videoRepository) CreateStyle(ctx context.Context, st *service.VideoStyleCard) error {
	var source any
	if videoLooksLikeUUID(st.SourceProject) {
		source = st.SourceProject
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO video_styles (user_id, name, ratio, base_style, theme, common, character_desc, rules, source_project)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::jsonb, $9)
RETURNING id, created_at, updated_at`,
		st.UserID, st.Name, st.Ratio, st.BaseStyle, videoJSON(st.Theme), st.Common, st.Character, videoJSON(st.Rules), source,
	).Scan(&st.ID, &st.CreatedAt, &st.UpdatedAt)
}

func (r *videoRepository) UpdateStyle(ctx context.Context, st *service.VideoStyleCard) error {
	return r.db.QueryRowContext(ctx, `
UPDATE video_styles SET name = $3, rules = $4::jsonb, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING updated_at`, st.ID, st.UserID, st.Name, videoJSON(st.Rules)).Scan(&st.UpdatedAt)
}

func (r *videoRepository) DeleteStyle(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM video_styles WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
