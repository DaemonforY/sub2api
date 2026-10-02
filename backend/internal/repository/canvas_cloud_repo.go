package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type canvasCloudRepository struct {
	db *sql.DB
}

// NewCanvasCloudRepository stores the index of users' canvas cloud files (raw SQL).
func NewCanvasCloudRepository(db *sql.DB) service.CanvasCloudRepository {
	return &canvasCloudRepository{db: db}
}

func (r *canvasCloudRepository) GetCanvasCloudFile(ctx context.Context, userID int64, path string) (*service.CanvasCloudFile, error) {
	var f service.CanvasCloudFile
	err := r.db.QueryRowContext(ctx, `SELECT path, file, size, mime, updated_at FROM canvas_cloud_files WHERE user_id = $1 AND path = $2`, userID, path).
		Scan(&f.Path, &f.File, &f.Size, &f.Mime, &f.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *canvasCloudRepository) ListCanvasCloudFiles(ctx context.Context, userID int64) ([]service.CanvasCloudFile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT path, file, size, mime, updated_at FROM canvas_cloud_files WHERE user_id = $1 ORDER BY path`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.CanvasCloudFile{}
	for rows.Next() {
		var f service.CanvasCloudFile
		if err := rows.Scan(&f.Path, &f.File, &f.Size, &f.Mime, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *canvasCloudRepository) CanvasCloudUsage(ctx context.Context, userID int64) (int64, int, error) {
	var (
		bytes int64
		files int
	)
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size), 0), COUNT(*) FROM canvas_cloud_files WHERE user_id = $1`, userID).Scan(&bytes, &files)
	return bytes, files, err
}

func (r *canvasCloudRepository) PutCanvasCloudFile(ctx context.Context, userID int64, f *service.CanvasCloudFile) (replaced string, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	// The old disk name comes back so the service can remove that file.
	var old sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT file FROM canvas_cloud_files WHERE user_id = $1 AND path = $2 FOR UPDATE`, userID, f.Path).Scan(&old); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if err = tx.QueryRowContext(ctx, `
INSERT INTO canvas_cloud_files (user_id, path, file, size, mime, updated_at) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (user_id, path) DO UPDATE SET file = EXCLUDED.file, size = EXCLUDED.size, mime = EXCLUDED.mime, updated_at = NOW()
RETURNING updated_at`, userID, f.Path, f.File, f.Size, f.Mime).Scan(&f.UpdatedAt); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return old.String, nil
}

func (r *canvasCloudRepository) DeleteCanvasCloudFile(ctx context.Context, userID int64, path string) (string, error) {
	var file string
	err := r.db.QueryRowContext(ctx, `DELETE FROM canvas_cloud_files WHERE user_id = $1 AND path = $2 RETURNING file`, userID, path).Scan(&file)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return file, err
}
