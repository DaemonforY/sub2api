package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 我的提示词 folders, pinning and bulk import (see service/prompt_library_mine.go).

func (r *promptLibraryRepository) ListFolders(ctx context.Context, userID int64) ([]service.PromptFolder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, name, sort_order, created_at, updated_at FROM prompt_folders
WHERE user_id = $1 ORDER BY sort_order, id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptFolder{}
	for rows.Next() {
		var f service.PromptFolder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *promptLibraryRepository) CountByFolder(ctx context.Context, userID int64) (map[int64]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT COALESCE(folder_id, 0), COUNT(*) FROM prompt_items WHERE owner_user_id = $1 GROUP BY 1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]int64{}
	for rows.Next() {
		var id, n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

func (r *promptLibraryRepository) CreateFolder(ctx context.Context, f *service.PromptFolder) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO prompt_folders (user_id, name, sort_order) VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at`, f.UserID, f.Name, f.SortOrder).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	if isUniqueViolation(err) {
		return service.ErrPromptFolderExists
	}
	return err
}

func (r *promptLibraryRepository) UpdateFolder(ctx context.Context, f *service.PromptFolder) error {
	err := r.db.QueryRowContext(ctx, `
UPDATE prompt_folders SET name = $3, sort_order = $4, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING created_at, updated_at`, f.ID, f.UserID, f.Name, f.SortOrder).Scan(&f.CreatedAt, &f.UpdatedAt)
	switch {
	case isUniqueViolation(err):
		return service.ErrPromptFolderExists
	case errors.Is(err, sql.ErrNoRows):
		return service.ErrPromptFolderNotFound
	}
	return err
}

func (r *promptLibraryRepository) DeleteFolder(ctx context.Context, userID, id int64, withItems bool) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	if withItems {
		if _, err := tx.ExecContext(ctx, `DELETE FROM prompt_items WHERE owner_user_id = $1 AND folder_id = $2`, userID, id); err != nil {
			return false, err
		}
	}
	// Remaining prompts fall back to 未分类 through ON DELETE SET NULL.
	res, err := tx.ExecContext(ctx, `DELETE FROM prompt_folders WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, nil
	}
	return true, tx.Commit()
}

func (r *promptLibraryRepository) FolderOwnedBy(ctx context.Context, id, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM prompt_folders WHERE id = $1 AND user_id = $2)`, id, userID).Scan(&ok)
	return ok, err
}

func (r *promptLibraryRepository) CountFolders(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prompt_folders WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *promptLibraryRepository) BatchMine(ctx context.Context, userID int64, ids []int64, op service.PromptMineBatchOp) (int64, error) {
	var (
		res sql.Result
		err error
	)
	switch op.Action {
	case "move":
		var folder any
		if op.FolderID > 0 {
			folder = op.FolderID
		}
		res, err = r.db.ExecContext(ctx, `UPDATE prompt_items SET folder_id = $3, updated_at = NOW() WHERE owner_user_id = $1 AND id = ANY($2)`, userID, pq.Array(ids), folder)
	case "pin", "unpin":
		res, err = r.db.ExecContext(ctx, `UPDATE prompt_items SET pinned = $3 WHERE owner_user_id = $1 AND id = ANY($2)`, userID, pq.Array(ids), op.Action == "pin")
	case "delete":
		res, err = r.db.ExecContext(ctx, `DELETE FROM prompt_items WHERE owner_user_id = $1 AND id = ANY($2)`, userID, pq.Array(ids))
	default:
		return 0, service.ErrPromptMineBatchOp
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *promptLibraryRepository) OwnedDedupeKeys(ctx context.Context, userID int64, keys []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT dedupe_key FROM prompt_items WHERE owner_user_id = $1 AND dedupe_key = ANY($2)`, userID, pq.Array(keys))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out[key] = true
	}
	return out, rows.Err()
}
