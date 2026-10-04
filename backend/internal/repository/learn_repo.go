package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type learnRepository struct {
	db *sql.DB
}

// NewLearnRepository stores lesson progress and example runs (raw SQL).
func NewLearnRepository(db *sql.DB) service.LearnRepository {
	return &learnRepository{db: db}
}

func (r *learnRepository) Progress(ctx context.Context, userID int64) (map[string]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT lesson_id, completed_at FROM learn_progress WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, err
		}
		out[id] = at
	}
	return out, rows.Err()
}

func (r *learnRepository) MarkDone(ctx context.Context, userID int64, lessonIDs []string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO learn_progress (user_id, lesson_id) SELECT $1, UNNEST($2::text[])
ON CONFLICT (user_id, lesson_id) DO NOTHING`, userID, pq.Array(lessonIDs))
	return err
}

func (r *learnRepository) CountRuns(ctx context.Context, userID int64, kind string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM learn_runs
WHERE created_at >= $2 AND status <> 'failed' AND key_id IS NULL AND ($1 = 0 OR user_id = $1) AND ($3 = '' OR kind = $3)`,
		userID, since, kind).Scan(&n)
	return n, err
}

func (r *learnRepository) StartRun(ctx context.Context, userID int64, lessonID, kind string, keyID int64) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `INSERT INTO learn_runs (user_id, lesson_id, kind, key_id) VALUES ($1, $2, $3, NULLIF($4, 0)) RETURNING id`,
		userID, lessonID, kind, keyID).Scan(&id)
	return id, err
}

func (r *learnRepository) FinishRun(ctx context.Context, runID int64, status string, latencyMs, promptTokens, completionTokens int) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE learn_runs SET status = $2, latency_ms = $3, prompt_tokens = $4, completion_tokens = $5 WHERE id = $1`,
		runID, status, latencyMs, promptTokens, completionTokens)
	return err
}

func (r *learnRepository) Stats(ctx context.Context, today time.Time) (*service.LearnStats, error) {
	out := &service.LearnStats{Lessons: []service.LearnLessonStat{}}
	week := today.AddDate(0, 0, -6)
	if err := r.db.QueryRowContext(ctx, `
SELECT (SELECT COUNT(DISTINCT user_id) FROM learn_progress),
       (SELECT COUNT(*) FROM learn_runs WHERE created_at >= $1 AND status <> 'failed'),
       (SELECT COUNT(DISTINCT user_id) FROM learn_runs WHERE created_at >= $1),
       (SELECT COUNT(*) FROM learn_runs WHERE created_at >= $2 AND status <> 'failed'),
       (SELECT COUNT(*) FROM learn_runs WHERE created_at >= $2 AND status = 'failed'),
       (SELECT COALESCE(SUM(prompt_tokens + completion_tokens), 0) FROM learn_runs WHERE created_at >= $2),
       (SELECT COUNT(*) FROM learn_certificates WHERE revoked_at IS NULL),
       (SELECT COUNT(*) FROM learn_quiz_results WHERE total > 0 AND correct * 100 >= total * 80),
       (SELECT COUNT(*) FROM learn_runs WHERE created_at >= $2 AND kind = 'tutor' AND status = 'ok'),
       (SELECT COUNT(*) FROM learn_interviews WHERE created_at >= $2),
       (SELECT COUNT(*) FROM learn_runs WHERE created_at >= $2 AND key_id IS NOT NULL AND status = 'ok')`,
		today, week).Scan(&out.Learners, &out.RunsToday, &out.LearnersToday, &out.Runs7d, &out.FailedRuns7d, &out.Tokens7d,
		&out.Certificates, &out.QuizPassed, &out.Tutor7d, &out.Interviews7d, &out.OwnKeyRuns7d); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT lesson_id, SUM(done), SUM(runs) FROM (
    SELECT lesson_id, 1 AS done, 0 AS runs FROM learn_progress
    UNION ALL
    SELECT lesson_id, 0, 1 FROM learn_runs WHERE status = 'ok' AND kind = 'run'
) x GROUP BY lesson_id ORDER BY lesson_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var s service.LearnLessonStat
		if err := rows.Scan(&s.LessonID, &s.Completed, &s.Runs); err != nil {
			return nil, err
		}
		out.Lessons = append(out.Lessons, s)
	}
	return out, rows.Err()
}

func (r *learnRepository) QuizResults(ctx context.Context, userID int64) (map[string]service.LearnQuizResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT lesson_id, correct, total, attempts, updated_at FROM learn_quiz_results WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]service.LearnQuizResult{}
	for rows.Next() {
		var id string
		var q service.LearnQuizResult
		if err := rows.Scan(&id, &q.Correct, &q.Total, &q.Attempts, &q.UpdatedAt); err != nil {
			return nil, err
		}
		out[id] = q
	}
	return out, rows.Err()
}

func (r *learnRepository) SaveQuizResult(ctx context.Context, userID int64, lessonID string, correct, total int) (service.LearnQuizResult, error) {
	var q service.LearnQuizResult
	// A better score replaces the stored one (a changed quiz: the latest total wins).
	err := r.db.QueryRowContext(ctx, `
INSERT INTO learn_quiz_results (user_id, lesson_id, correct, total, attempts) VALUES ($1, $2, $3, $4, 1)
ON CONFLICT (user_id, lesson_id) DO UPDATE SET
    attempts = learn_quiz_results.attempts + 1,
    correct = CASE WHEN learn_quiz_results.total <> EXCLUDED.total OR EXCLUDED.correct > learn_quiz_results.correct
                   THEN EXCLUDED.correct ELSE learn_quiz_results.correct END,
    total = EXCLUDED.total,
    updated_at = NOW()
RETURNING correct, total, attempts, updated_at`, userID, lessonID, correct, total).Scan(&q.Correct, &q.Total, &q.Attempts, &q.UpdatedAt)
	return q, err
}

func (r *learnRepository) Checkpoints(ctx context.Context, userID int64) (map[string]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT checkpoint_id, passed_at FROM learn_checkpoints WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, err
		}
		out[id] = at
	}
	return out, rows.Err()
}

func (r *learnRepository) PassCheckpoint(ctx context.Context, userID int64, checkpointID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO learn_checkpoints (user_id, checkpoint_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, checkpointID)
	return err
}

const learnCertColumns = `c.code, c.user_id, COALESCE(u.email, ''), c.track, c.display_name, c.project_url, c.quiz_score, c.issued_at, c.revoked_at, c.showcase, c.showcase_hidden`

func scanLearnCert(row interface{ Scan(...any) error }) (service.LearnCertificate, error) {
	var c service.LearnCertificate
	var revoked sql.NullTime
	err := row.Scan(&c.Code, &c.UserID, &c.UserEmail, &c.Track, &c.DisplayName, &c.ProjectURL, &c.QuizScore, &c.IssuedAt, &revoked, &c.Showcase, &c.ShowcaseHidden)
	if revoked.Valid {
		c.RevokedAt = &revoked.Time
	}
	return c, err
}

func (r *learnRepository) CreateCertificate(ctx context.Context, c *service.LearnCertificate) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO learn_certificates (code, user_id, track, display_name, project_url, quiz_score, issued_at, showcase)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id, track) WHERE revoked_at IS NULL DO NOTHING`,
		c.Code, c.UserID, c.Track, c.DisplayName, c.ProjectURL, c.QuizScore, c.IssuedAt, c.Showcase)
	return err
}

func (r *learnRepository) CertificateByCode(ctx context.Context, code string) (*service.LearnCertificate, error) {
	c, err := scanLearnCert(r.db.QueryRowContext(ctx, `SELECT `+learnCertColumns+`
FROM learn_certificates c LEFT JOIN users u ON u.id = c.user_id WHERE c.code = $1`, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *learnRepository) queryCerts(ctx context.Context, query string, args ...any) ([]service.LearnCertificate, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LearnCertificate{}
	for rows.Next() {
		c, err := scanLearnCert(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *learnRepository) CertificatesByUser(ctx context.Context, userID int64) ([]service.LearnCertificate, error) {
	return r.queryCerts(ctx, `SELECT `+learnCertColumns+`
FROM learn_certificates c LEFT JOIN users u ON u.id = c.user_id WHERE c.user_id = $1 ORDER BY c.issued_at`, userID)
}

func (r *learnRepository) ListCertificates(ctx context.Context, limit, offset int) ([]service.LearnCertificate, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM learn_certificates`).Scan(&total); err != nil {
		return nil, 0, err
	}
	certs, err := r.queryCerts(ctx, `SELECT `+learnCertColumns+`
FROM learn_certificates c LEFT JOIN users u ON u.id = c.user_id ORDER BY c.issued_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	return certs, total, err
}

func (r *learnRepository) SetCertificateRevoked(ctx context.Context, code string, revoked bool) error {
	q := `UPDATE learn_certificates SET revoked_at = NOW() WHERE code = $1 AND revoked_at IS NULL`
	if !revoked {
		q = `UPDATE learn_certificates SET revoked_at = NULL WHERE code = $1`
	}
	res, err := r.db.ExecContext(ctx, q, code)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrLearnCertRestore
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrLearnCertNotFound
	}
	return nil
}

func scanLearnInterview(row interface{ Scan(...any) error }) (service.LearnInterview, error) {
	var iv service.LearnInterview
	var questions, answers []byte
	var keyID sql.NullInt64
	var finished sql.NullTime
	if err := row.Scan(&iv.ID, &iv.UserID, &iv.Topic, &questions, &answers, &iv.Status, &iv.Score, &keyID, &iv.CreatedAt, &finished); err != nil {
		return iv, err
	}
	if err := json.Unmarshal(questions, &iv.Questions); err != nil {
		return iv, err
	}
	if err := json.Unmarshal(answers, &iv.Answers); err != nil {
		return iv, err
	}
	iv.KeyID = keyID.Int64
	if finished.Valid {
		iv.FinishedAt = &finished.Time
	}
	return iv, nil
}

const learnInterviewColumns = `id, user_id, topic, questions, answers, status, score, key_id, created_at, finished_at`

func (r *learnRepository) CreateInterview(ctx context.Context, iv *service.LearnInterview) error {
	questions, err := json.Marshal(iv.Questions)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO learn_interviews (user_id, topic, questions, status, key_id) VALUES ($1, $2, $3, $4, NULLIF($5, 0)) RETURNING id, created_at`,
		iv.UserID, iv.Topic, string(questions), iv.Status, iv.KeyID).Scan(&iv.ID, &iv.CreatedAt)
}

func (r *learnRepository) GetInterview(ctx context.Context, id int64) (*service.LearnInterview, error) {
	iv, err := scanLearnInterview(r.db.QueryRowContext(ctx, `SELECT `+learnInterviewColumns+` FROM learn_interviews WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &iv, nil
}

func (r *learnRepository) SaveInterview(ctx context.Context, iv *service.LearnInterview) error {
	answers, err := json.Marshal(iv.Answers)
	if err != nil {
		return err
	}
	// Only on top of the answers this one was graded after (a concurrent answer loses).
	res, err := r.db.ExecContext(ctx, `
UPDATE learn_interviews SET answers = $2, status = $3, score = $4, finished_at = $5
WHERE id = $1 AND jsonb_array_length(answers) = $6`, iv.ID, string(answers), iv.Status, iv.Score, iv.FinishedAt, len(iv.Answers)-1)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrLearnInterviewDone
	}
	return nil
}

func (r *learnRepository) ListInterviews(ctx context.Context, userID int64, limit int) ([]service.LearnInterview, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+learnInterviewColumns+` FROM learn_interviews WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LearnInterview{}
	for rows.Next() {
		iv, err := scanLearnInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, iv)
	}
	return out, rows.Err()
}

func (r *learnRepository) CountInterviews(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM learn_interviews WHERE user_id = $1 AND created_at >= $2 AND key_id IS NULL`, userID, since).Scan(&n)
	return n, err
}

func (r *learnRepository) BestInterviewScores(ctx context.Context, userID int64) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT topic, MAX(score) FROM learn_interviews WHERE user_id = $1 AND status = 'finished' GROUP BY topic`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var topic string
		var score int
		if err := rows.Scan(&topic, &score); err != nil {
			return nil, err
		}
		out[topic] = score
	}
	return out, rows.Err()
}

func (r *learnRepository) SetShowcase(ctx context.Context, userID int64, track string, on bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE learn_certificates SET showcase = $3 WHERE user_id = $1 AND track = $2 AND revoked_at IS NULL`, userID, track, on)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrLearnCertNotFound
	}
	return nil
}

func (r *learnRepository) SetShowcaseHidden(ctx context.Context, code string, hidden bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE learn_certificates SET showcase_hidden = $2 WHERE code = $1`, code, hidden)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrLearnCertNotFound
	}
	return nil
}

func (r *learnRepository) Showcase(ctx context.Context, track string, limit int) ([]service.LearnCertificate, error) {
	return r.queryCerts(ctx, `SELECT `+learnCertColumns+`
FROM learn_certificates c LEFT JOIN users u ON u.id = c.user_id
WHERE c.showcase AND NOT c.showcase_hidden AND c.revoked_at IS NULL AND ($1 = '' OR c.track = $1)
ORDER BY c.issued_at DESC LIMIT $2`, track, limit)
}

func (r *learnRepository) TrackProgress(ctx context.Context) ([]service.LearnTrackProgress, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT LEFT(lesson_id, 1), user_id, COUNT(*) FROM learn_progress GROUP BY 1, 2`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LearnTrackProgress{}
	for rows.Next() {
		var p service.LearnTrackProgress
		if err := rows.Scan(&p.Track, &p.UserID, &p.Lessons); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *learnRepository) CertificateCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT track, COUNT(*) FROM learn_certificates WHERE revoked_at IS NULL GROUP BY track`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var track string
		var n int
		if err := rows.Scan(&track, &n); err != nil {
			return nil, err
		}
		out[track] = n
	}
	return out, rows.Err()
}

// Daily counts per Beijing-time day since the time (days without activity are absent).
func (r *learnRepository) Daily(ctx context.Context, since time.Time) ([]service.LearnDay, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH runs AS (
    SELECT (created_at AT TIME ZONE 'Asia/Shanghai')::date AS d, kind, user_id FROM learn_runs WHERE created_at >= $1 AND status = 'ok'
), done AS (
    SELECT (completed_at AT TIME ZONE 'Asia/Shanghai')::date AS d, user_id FROM learn_progress WHERE completed_at >= $1
), certs AS (
    SELECT (issued_at AT TIME ZONE 'Asia/Shanghai')::date AS d FROM learn_certificates WHERE issued_at >= $1
), days AS (
    SELECT d FROM runs UNION SELECT d FROM done UNION SELECT d FROM certs
)
SELECT to_char(days.d, 'YYYY-MM-DD'),
       (SELECT COUNT(*) FROM runs WHERE runs.d = days.d AND kind = 'run'),
       (SELECT COUNT(*) FROM runs WHERE runs.d = days.d AND kind = 'tutor'),
       (SELECT COUNT(*) FROM runs WHERE runs.d = days.d AND kind = 'interview'),
       (SELECT COUNT(*) FROM done WHERE done.d = days.d),
       (SELECT COUNT(DISTINCT user_id) FROM (SELECT user_id FROM runs WHERE runs.d = days.d UNION SELECT user_id FROM done WHERE done.d = days.d) a),
       (SELECT COUNT(*) FROM certs WHERE certs.d = days.d)
FROM days ORDER BY days.d`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.LearnDay{}
	for rows.Next() {
		var d service.LearnDay
		if err := rows.Scan(&d.Date, &d.Runs, &d.Tutor, &d.Interviews, &d.Completions, &d.Learners, &d.Certificates); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *learnRepository) QuizStats(ctx context.Context) (map[string]service.LearnQuizStat, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT lesson_id, COUNT(*), COUNT(*) FILTER (WHERE total > 0 AND correct * 100 >= total * 80),
       COALESCE(ROUND(AVG(CASE WHEN total > 0 THEN correct * 100.0 / total END)), 0), COALESCE(SUM(attempts), 0)
FROM learn_quiz_results GROUP BY lesson_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]service.LearnQuizStat{}
	for rows.Next() {
		var id string
		var q service.LearnQuizStat
		if err := rows.Scan(&id, &q.Takers, &q.Passed, &q.AvgScore, &q.Attempts); err != nil {
			return nil, err
		}
		out[id] = q
	}
	return out, rows.Err()
}
