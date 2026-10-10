package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type geoMonitorRepository struct {
	db *sql.DB
}

// NewGeoMonitorRepository stores GEO 监测 questions, engines (key encrypted by the service) and
// answers (raw SQL).
func NewGeoMonitorRepository(db *sql.DB) service.GeoMonitorRepository {
	return &geoMonitorRepository{db: db}
}

// ---- questions ----

const geoQuestionColumns = `id, question, category, enabled, sort, created_at, updated_at`

func scanGeoQuestion(row interface{ Scan(...any) error }, q *service.GeoQuestion) error {
	return row.Scan(&q.ID, &q.Question, &q.Category, &q.Enabled, &q.Sort, &q.CreatedAt, &q.UpdatedAt)
}

func (r *geoMonitorRepository) ListQuestions(ctx context.Context) ([]service.GeoQuestion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+geoQuestionColumns+` FROM geo_questions ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GeoQuestion
	for rows.Next() {
		var q service.GeoQuestion
		if err := scanGeoQuestion(rows, &q); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r *geoMonitorRepository) GetQuestion(ctx context.Context, id int64) (*service.GeoQuestion, error) {
	var q service.GeoQuestion
	err := scanGeoQuestion(r.db.QueryRowContext(ctx, `SELECT `+geoQuestionColumns+` FROM geo_questions WHERE id = $1`, id), &q)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrGeoQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *geoMonitorRepository) CreateQuestion(ctx context.Context, q *service.GeoQuestion) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO geo_questions (question, category, enabled, sort) VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at`, q.Question, q.Category, q.Enabled, q.Sort).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
}

func (r *geoMonitorRepository) UpdateQuestion(ctx context.Context, q *service.GeoQuestion) error {
	err := r.db.QueryRowContext(ctx, `
UPDATE geo_questions SET question = $2, category = $3, enabled = $4, sort = $5, updated_at = NOW()
WHERE id = $1 RETURNING created_at, updated_at`, q.ID, q.Question, q.Category, q.Enabled, q.Sort).Scan(&q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrGeoQuestionNotFound
	}
	return err
}

func (r *geoMonitorRepository) DeleteQuestion(ctx context.Context, id int64) error {
	return geoExecOne(ctx, r.db, service.ErrGeoQuestionNotFound, `DELETE FROM geo_questions WHERE id = $1`, id)
}

// ---- engines ----

const geoEngineColumns = `id, name, base_url, api_key_encrypted, model, extra_body, enabled, created_at, updated_at`

func scanGeoEngine(row interface{ Scan(...any) error }, e *service.GeoEngine) error {
	var extra []byte
	if err := row.Scan(&e.ID, &e.Name, &e.BaseURL, &e.APIKeyEncrypted, &e.Model, &extra, &e.Enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return err
	}
	if len(extra) > 0 && string(extra) != "null" {
		e.ExtraBody = json.RawMessage(extra)
	}
	return nil
}

func geoExtraParam(extra json.RawMessage) any {
	if len(extra) == 0 {
		return nil
	}
	return string(extra)
}

func (r *geoMonitorRepository) ListEngines(ctx context.Context) ([]service.GeoEngine, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+geoEngineColumns+` FROM geo_engines ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GeoEngine
	for rows.Next() {
		var e service.GeoEngine
		if err := scanGeoEngine(rows, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *geoMonitorRepository) GetEngine(ctx context.Context, id int64) (*service.GeoEngine, error) {
	var e service.GeoEngine
	err := scanGeoEngine(r.db.QueryRowContext(ctx, `SELECT `+geoEngineColumns+` FROM geo_engines WHERE id = $1`, id), &e)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrGeoEngineNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func geoEngineNameErr(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrGeoEngineNameTaken
	}
	return err
}

func (r *geoMonitorRepository) CreateEngine(ctx context.Context, e *service.GeoEngine) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO geo_engines (name, base_url, api_key_encrypted, model, extra_body, enabled) VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING id, created_at, updated_at`, e.Name, e.BaseURL, e.APIKeyEncrypted, e.Model, geoExtraParam(e.ExtraBody), e.Enabled,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	return geoEngineNameErr(err)
}

func (r *geoMonitorRepository) UpdateEngine(ctx context.Context, e *service.GeoEngine) error {
	err := r.db.QueryRowContext(ctx, `
UPDATE geo_engines SET name = $2, base_url = $3, api_key_encrypted = $4, model = $5, extra_body = $6::jsonb, enabled = $7, updated_at = NOW()
WHERE id = $1 RETURNING created_at, updated_at`, e.ID, e.Name, e.BaseURL, e.APIKeyEncrypted, e.Model, geoExtraParam(e.ExtraBody), e.Enabled,
	).Scan(&e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrGeoEngineNotFound
	}
	return geoEngineNameErr(err)
}

func (r *geoMonitorRepository) DeleteEngine(ctx context.Context, id int64) error {
	return geoExecOne(ctx, r.db, service.ErrGeoEngineNotFound, `DELETE FROM geo_engines WHERE id = $1`, id)
}

// ---- checks ----

const geoCheckColumns = `id, question_id, question, engine_id, engine_name, source, answer, mentioned, cited_urls, our_urls, competitors, error, run_id, created_at`

func scanGeoCheck(row interface{ Scan(...any) error }, c *service.GeoCheck) error {
	var qid, eid sql.NullInt64
	var runID sql.NullString
	var cited, ours, comp []byte
	if err := row.Scan(&c.ID, &qid, &c.Question, &eid, &c.EngineName, &c.Source, &c.Answer, &c.Mentioned,
		&cited, &ours, &comp, &c.Error, &runID, &c.CreatedAt); err != nil {
		return err
	}
	if qid.Valid {
		c.QuestionID = &qid.Int64
	}
	if eid.Valid {
		c.EngineID = &eid.Int64
	}
	c.RunID = runID.String
	c.CitedURLs, c.OurURLs, c.Competitors = geoStrings(cited), geoStrings(ours), geoStrings(comp)
	return nil
}

func geoStrings(raw []byte) []string {
	out := []string{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func geoJSONList(v []string) string {
	if v == nil {
		v = []string{}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func (r *geoMonitorRepository) InsertCheck(ctx context.Context, c *service.GeoCheck) error {
	var runID any
	if c.RunID != "" {
		runID = c.RunID
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO geo_checks (question_id, question, engine_id, engine_name, source, answer, mentioned, cited_urls, our_urls, competitors, error, run_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12)
RETURNING id, created_at`,
		c.QuestionID, c.Question, c.EngineID, c.EngineName, c.Source, c.Answer, c.Mentioned,
		geoJSONList(c.CitedURLs), geoJSONList(c.OurURLs), geoJSONList(c.Competitors), c.Error, runID,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *geoMonitorRepository) ListChecks(ctx context.Context, f service.GeoCheckFilter) ([]service.GeoCheck, int64, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.QuestionID > 0 {
		where = append(where, "question_id = "+arg(f.QuestionID))
	}
	if f.Engine != "" {
		where = append(where, "engine_name = "+arg(f.Engine))
	}
	if f.Source != "" {
		where = append(where, "source = "+arg(f.Source))
	}
	if f.Mentioned != nil {
		where = append(where, "mentioned = "+arg(*f.Mentioned)+" AND error = ''")
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM geo_checks`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT ` + geoCheckColumns + ` FROM geo_checks` + cond + ` ORDER BY created_at DESC, id DESC LIMIT ` + arg(f.Limit) + ` OFFSET ` + arg(f.Offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GeoCheck
	for rows.Next() {
		var c service.GeoCheck
		if err := scanGeoCheck(rows, &c); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *geoMonitorRepository) DeleteCheck(ctx context.Context, id int64) error {
	return geoExecOne(ctx, r.db, service.ErrGeoCheckNotFound, `DELETE FROM geo_checks WHERE id = $1`, id)
}

func (r *geoMonitorRepository) LatestChecks(ctx context.Context) ([]service.GeoCheck, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (question_id, engine_name) `+geoCheckColumns+`
FROM geo_checks WHERE question_id IS NOT NULL AND error = ''
ORDER BY question_id, engine_name, created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GeoCheck
	for rows.Next() {
		var c service.GeoCheck
		if err := scanGeoCheck(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// WeeklyRates: weeks start on Monday, Beijing time; within a week only the latest answer to each
// question counts, so asking twice doesn't weigh double.
func (r *geoMonitorRepository) WeeklyRates(ctx context.Context, since time.Time) ([]service.GeoWeekRate, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH latest AS (
  SELECT DISTINCT ON (engine_name, question_id, date_trunc('week', created_at AT TIME ZONE 'Asia/Shanghai'))
    engine_name, (date_trunc('week', created_at AT TIME ZONE 'Asia/Shanghai'))::date AS week, mentioned
  FROM geo_checks
  WHERE question_id IS NOT NULL AND error = '' AND created_at >= $1
  ORDER BY engine_name, question_id, date_trunc('week', created_at AT TIME ZONE 'Asia/Shanghai'), created_at DESC, id DESC
)
SELECT engine_name, week::text, COUNT(*), COUNT(*) FILTER (WHERE mentioned)
FROM latest GROUP BY engine_name, week ORDER BY week, engine_name`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.GeoWeekRate
	for rows.Next() {
		var w service.GeoWeekRate
		var week string
		if err := rows.Scan(&w.EngineName, &week, &w.Total, &w.Mentioned); err != nil {
			return nil, err
		}
		if t, err := time.Parse("2006-01-02", week); err == nil {
			w.WeekStart = t
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func geoExecOne(ctx context.Context, db *sql.DB, notFound error, query string, args ...any) error {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notFound
	}
	return nil
}
