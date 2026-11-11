package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type gameCraftRepository struct {
	db *sql.DB
}

// NewGameCraftRepository is the store of 小游戏「AI 万物合成」 (game_craft_elements / game_craft_recipes,
// migration 326).
func NewGameCraftRepository(db *sql.DB) service.GameCraftRepository {
	return &gameCraftRepository{db: db}
}

const craftElementColumns = `id, name, emoji, description, created_at`

func scanCraftElement(row interface{ Scan(...any) error }) (service.CraftElement, error) {
	var el service.CraftElement
	err := row.Scan(&el.ID, &el.Name, &el.Emoji, &el.Description, &el.CreatedAt)
	return el, err
}

func (r *gameCraftRepository) Elements(ctx context.Context, names []string) (map[string]service.CraftElement, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+craftElementColumns+` FROM game_craft_elements WHERE name = ANY($1)`, pq.Array(names))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]service.CraftElement, len(names))
	for rows.Next() {
		el, err := scanCraftElement(rows)
		if err != nil {
			return nil, err
		}
		out[el.Name] = el
	}
	return out, rows.Err()
}

func (r *gameCraftRepository) Recipe(ctx context.Context, a, b string) (*service.CraftElement, error) {
	el, err := scanCraftElement(r.db.QueryRowContext(ctx, `
		SELECT e.id, e.name, e.emoji, e.description, e.created_at
		FROM game_craft_recipes rc JOIN game_craft_elements e ON e.name = rc.result
		WHERE rc.a = $1 AND rc.b = $2`, a, b))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &el, nil
}

func (r *gameCraftRepository) Save(ctx context.Context, a, b string, el service.CraftElement, userID int64) (out service.CraftElement, created bool, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	// The pair may have been stored by a request that lost the lock; its answer wins.
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT result FROM game_craft_recipes WHERE a = $1 AND b = $2`, a, b).Scan(&existing)
	switch {
	case err == nil:
		el.Name = existing
	case errors.Is(err, sql.ErrNoRows):
		err = nil
		var by sql.NullInt64
		if userID > 0 {
			by = sql.NullInt64{Int64: userID, Valid: true}
		}
		res, ierr := tx.ExecContext(ctx, `
			INSERT INTO game_craft_elements (name, emoji, description, discovered_by) VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING`, el.Name, el.Emoji, el.Description, by)
		if ierr != nil {
			return out, false, ierr
		}
		n, _ := res.RowsAffected()
		created = n == 1
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO game_craft_recipes (a, b, result) VALUES ($1, $2, $3) ON CONFLICT (a, b) DO NOTHING`, a, b, el.Name); err != nil {
			return out, false, err
		}
	default:
		return out, false, err
	}
	out, err = scanCraftElement(tx.QueryRowContext(ctx, `SELECT `+craftElementColumns+` FROM game_craft_elements WHERE name = $1`, el.Name))
	if err != nil {
		return out, false, err
	}
	if err = tx.Commit(); err != nil {
		return out, false, err
	}
	return out, created, nil
}

func (r *gameCraftRepository) Stats(ctx context.Context, recent int) (int64, []service.CraftElement, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM game_craft_elements`).Scan(&total); err != nil {
		return 0, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+craftElementColumns+` FROM game_craft_elements
		WHERE name <> ALL($2) ORDER BY id DESC LIMIT $1`, recent, pq.Array(service.CraftStarterNames()))
	if err != nil {
		return total, nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CraftElement
	for rows.Next() {
		el, err := scanCraftElement(rows)
		if err != nil {
			return total, nil, err
		}
		out = append(out, el)
	}
	return total, out, rows.Err()
}
