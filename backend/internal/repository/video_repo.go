package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type videoRepository struct {
	db *sql.DB
}

// NewVideoRepository creates the HiveGPT 视频 project store (raw SQL).
func NewVideoRepository(db *sql.DB) service.VideoRepository {
	return &videoRepository{db: db}
}

const videoProjectColumns = `p.id, p.user_id, p.api_key_id, p.mode, p.title, p.prompt, p.options, p.status, p.stage, p.error, p.spec,
p.duration, p.width, p.height, p.usage, p.visibility, p.category, p.featured, p.views, p.remixes, p.remix_of,
p.created_at, p.updated_at, p.published_at, COALESCE(u.username, ''), COALESCE(u.email, '')`

func videoJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func (r *videoRepository) Create(ctx context.Context, p *service.VideoProject) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO video_projects (user_id, api_key_id, mode, title, prompt, options, status, width, height, visibility, category, remix_of)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12)
RETURNING id, created_at, updated_at`,
		p.UserID, p.APIKeyID, p.Mode, p.Title, p.Prompt, videoJSON(p.Options), p.Status, p.Width, p.Height, p.Visibility, p.Category, p.RemixOf,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *videoRepository) Get(ctx context.Context, id string) (*service.VideoProject, error) {
	if !videoLooksLikeUUID(id) {
		return nil, nil
	}
	p, err := scanVideoProject(r.db.QueryRowContext(ctx, `SELECT `+videoProjectColumns+` FROM video_projects p LEFT JOIN users u ON u.id = p.user_id WHERE p.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *videoRepository) Save(ctx context.Context, p *service.VideoProject) error {
	var spec any
	if p.Spec != nil {
		spec = videoJSON(p.Spec)
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE video_projects
SET title = $2, status = $3, stage = $4, error = $5, spec = $6::jsonb, duration = $7, usage = $8::jsonb, options = $9::jsonb,
    api_key_id = $10, updated_at = NOW()
WHERE id = $1`,
		p.ID, p.Title, p.Status, p.Stage, p.Error, spec, p.Duration, videoJSON(p.Usage), videoJSON(p.Options), p.APIKeyID)
	return err
}

func (r *videoRepository) Delete(ctx context.Context, userID int64, id string) (bool, error) {
	if !videoLooksLikeUUID(id) {
		return false, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM video_projects WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

const videoCardColumns = `p.id, p.mode, p.title, LEFT(p.prompt, 300), p.status, p.stage, p.visibility, p.category, p.featured, p.views, p.remixes,
p.duration, p.width, p.height, COALESCE(p.options->>'style', ''), COALESCE(u.username, ''), COALESCE(u.email, ''), p.created_at, p.updated_at`

func (r *videoRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]service.VideoCard, error) {
	return r.cards(ctx, `SELECT `+videoCardColumns+` FROM video_projects p LEFT JOIN users u ON u.id = p.user_id WHERE p.user_id = $1 ORDER BY p.updated_at DESC LIMIT $2`, userID, limit)
}

func (r *videoRepository) CountRunning(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_projects WHERE user_id = $1 AND status = 'running'`, userID).Scan(&n)
	return n, err
}

func (r *videoRepository) FailRunning(ctx context.Context, message string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `UPDATE video_projects SET status = 'failed', error = $1, stage = '', updated_at = NOW() WHERE status = 'running' RETURNING id`, message)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *videoRepository) AddEvent(ctx context.Context, projectID string, e *service.VideoEvent) error {
	var data any
	if len(e.Data) > 0 {
		data = string(e.Data)
	}
	return r.db.QueryRowContext(ctx, `INSERT INTO video_project_events (project_id, kind, text, data) VALUES ($1, $2, $3, $4::jsonb) RETURNING id, created_at`,
		projectID, e.Kind, e.Text, data).Scan(&e.ID, &e.CreatedAt)
}

func (r *videoRepository) ListEvents(ctx context.Context, projectID string, afterID int64, limit int) ([]service.VideoEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, kind, text, data, created_at FROM video_project_events WHERE project_id = $1 AND id > $2 ORDER BY id LIMIT $3`, projectID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoEvent{}
	for rows.Next() {
		var e service.VideoEvent
		var data []byte
		if err := rows.Scan(&e.ID, &e.Kind, &e.Text, &data, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(data) > 0 {
			e.Data = json.RawMessage(data)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *videoRepository) AddVersion(ctx context.Context, projectID, note string, spec *service.VideoSpec) error {
	if spec == nil {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO video_project_versions (project_id, note, spec) VALUES ($1, $2, $3::jsonb)`, projectID, note, videoJSON(spec)); err != nil {
		return err
	}
	// Keep the newest 30 snapshots.
	_, err := r.db.ExecContext(ctx, `DELETE FROM video_project_versions WHERE project_id = $1 AND id NOT IN (SELECT id FROM video_project_versions WHERE project_id = $1 ORDER BY id DESC LIMIT 30)`, projectID)
	return err
}

func (r *videoRepository) ListVersions(ctx context.Context, projectID string, limit int) ([]service.VideoVersion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, note, created_at FROM video_project_versions WHERE project_id = $1 ORDER BY id DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoVersion{}
	for rows.Next() {
		var v service.VideoVersion
		if err := rows.Scan(&v.ID, &v.Note, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *videoRepository) GetVersion(ctx context.Context, projectID string, id int64) (*service.VideoVersion, error) {
	var v service.VideoVersion
	var spec []byte
	err := r.db.QueryRowContext(ctx, `SELECT id, note, spec, created_at FROM video_project_versions WHERE project_id = $1 AND id = $2`, projectID, id).Scan(&v.ID, &v.Note, &spec, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.Spec = &service.VideoSpec{}
	if err := json.Unmarshal(spec, v.Spec); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *videoRepository) SetVisibility(ctx context.Context, id, visibility, category, title string, featured *bool) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE video_projects
SET visibility = $2::text, category = $3, title = $4, featured = COALESCE($5::boolean, featured),
    published_at = CASE WHEN $2::text = 'public' AND published_at IS NULL THEN NOW() WHEN $2::text = 'private' THEN NULL ELSE published_at END,
    updated_at = NOW()
WHERE id = $1`, id, visibility, category, title, featured)
	return err
}

func (r *videoRepository) Gallery(ctx context.Context, q service.VideoGalleryQuery) ([]service.VideoCard, int, error) {
	where := []string{"p.visibility = 'public'"}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.Category == "featured" {
		where = append(where, "p.featured")
	} else if q.Category != "" && q.Category != "all" {
		add("p.category = ?", q.Category)
	}
	if q.Mode != "" {
		add("p.mode = ?", q.Mode)
	}
	if q.Search != "" {
		// Both placeholders take the same argument.
		add("(p.title ILIKE ? OR p.prompt ILIKE ?)", "%"+videoEscapeLike(q.Search)+"%")
	}
	order := "p.featured DESC, p.published_at DESC"
	switch q.Sort {
	case "new":
		order = "p.published_at DESC"
	case "hot":
		order = "(p.views + p.remixes * 5) DESC, p.published_at DESC"
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_projects p WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	cards, err := r.cards(ctx, `SELECT `+videoCardColumns+` FROM video_projects p LEFT JOIN users u ON u.id = p.user_id WHERE `+cond+
		` ORDER BY `+order+` LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	return cards, total, err
}

func (r *videoRepository) Pending(ctx context.Context, limit int) ([]service.VideoCard, error) {
	return r.cards(ctx, `SELECT `+videoCardColumns+` FROM video_projects p LEFT JOIN users u ON u.id = p.user_id WHERE p.visibility = 'pending' ORDER BY p.updated_at LIMIT $1`, limit)
}

func (r *videoRepository) AddView(ctx context.Context, id string) error {
	if !videoLooksLikeUUID(id) {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE video_projects SET views = views + 1 WHERE id = $1 AND visibility = 'public'`, id)
	return err
}

func (r *videoRepository) AddRemix(ctx context.Context, id string) error {
	if !videoLooksLikeUUID(id) {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE video_projects SET remixes = remixes + 1 WHERE id = $1 AND visibility = 'public'`, id)
	return err
}

func (r *videoRepository) cards(ctx context.Context, query string, args ...any) ([]service.VideoCard, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoCard{}
	for rows.Next() {
		var c service.VideoCard
		var username, email string
		if err := rows.Scan(&c.ID, &c.Mode, &c.Title, &c.Prompt, &c.Status, &c.Stage, &c.Visibility, &c.Category, &c.Featured, &c.Views, &c.Remixes,
			&c.Duration, &c.Width, &c.Height, &c.Style, &username, &email, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Author = service.VideoAuthorName(username, email)
		out = append(out, c)
	}
	return out, rows.Err()
}

type videoScanner interface {
	Scan(dest ...any) error
}

func scanVideoProject(row videoScanner) (*service.VideoProject, error) {
	var p service.VideoProject
	var options, spec, usage []byte
	var published pq.NullTime
	var username, email string
	if err := row.Scan(&p.ID, &p.UserID, &p.APIKeyID, &p.Mode, &p.Title, &p.Prompt, &options, &p.Status, &p.Stage, &p.Error, &spec,
		&p.Duration, &p.Width, &p.Height, &usage, &p.Visibility, &p.Category, &p.Featured, &p.Views, &p.Remixes, &p.RemixOf,
		&p.CreatedAt, &p.UpdatedAt, &published, &username, &email); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(options, &p.Options)
	_ = json.Unmarshal(usage, &p.Usage)
	if len(spec) > 0 && string(spec) != "null" {
		p.Spec = &service.VideoSpec{}
		if err := json.Unmarshal(spec, p.Spec); err != nil {
			return nil, err
		}
	}
	if published.Valid {
		p.PublishedAt = &published.Time
	}
	p.Author = service.VideoAuthorName(username, email)
	return &p, nil
}

func videoLooksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'):
		default:
			return false
		}
	}
	return true
}

func videoEscapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (r *videoRepository) GetUserKey(ctx context.Context, userID, groupID int64) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT api_key_id FROM video_user_keys WHERE user_id = $1 AND group_id = $2`, userID, groupID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

func (r *videoRepository) SetUserKey(ctx context.Context, userID, groupID, apiKeyID int64) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO video_user_keys (user_id, group_id, api_key_id) VALUES ($1, $2, $3)
ON CONFLICT (user_id, group_id) DO UPDATE SET api_key_id = EXCLUDED.api_key_id, created_at = NOW()`, userID, groupID, apiKeyID)
	return err
}
