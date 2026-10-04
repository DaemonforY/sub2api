package repository

import (
	"context"
	"database/sql"
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

func (r *learnRepository) CountRuns(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM learn_runs WHERE created_at >= $2 AND status <> 'failed' AND ($1 = 0 OR user_id = $1)`, userID, since).Scan(&n)
	return n, err
}

func (r *learnRepository) StartRun(ctx context.Context, userID int64, lessonID string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `INSERT INTO learn_runs (user_id, lesson_id) VALUES ($1, $2) RETURNING id`, userID, lessonID).Scan(&id)
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
       (SELECT COALESCE(SUM(prompt_tokens + completion_tokens), 0) FROM learn_runs WHERE created_at >= $2)`,
		today, week).Scan(&out.Learners, &out.RunsToday, &out.LearnersToday, &out.Runs7d, &out.FailedRuns7d, &out.Tokens7d); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT lesson_id, SUM(done), SUM(runs) FROM (
    SELECT lesson_id, 1 AS done, 0 AS runs FROM learn_progress
    UNION ALL
    SELECT lesson_id, 0, 1 FROM learn_runs WHERE status = 'ok'
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
