package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type agentRunRepository struct {
	db *sql.DB
}

// NewAgentRunRepository creates the tool-using assistants' run log (raw SQL, agent_runs).
func NewAgentRunRepository(db *sql.DB) service.AgentRunRepository {
	return &agentRunRepository{db: db}
}

func (r *agentRunRepository) Create(ctx context.Context, run *service.AgentRunRecord) (int64, error) {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return 0, err
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `
INSERT INTO agent_runs (agent, user_id, question, answer, steps, model, model_calls, prompt_tokens, completion_tokens, status, error, duration_ms)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12)
RETURNING id`,
		run.Agent, run.UserID, run.Question, run.Answer, string(steps), run.Model, run.ModelCalls, run.PromptTokens,
		run.CompletionTokens, run.Status, run.Error, run.DurationMs).Scan(&id)
	return id, err
}

func (r *agentRunRepository) List(ctx context.Context, agent string, page, pageSize int) ([]service.AgentRunRecord, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_runs WHERE agent = $1`, agent).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT r.id, r.agent, r.user_id, COALESCE(u.email, ''), r.question, r.answer, r.steps, r.model, r.model_calls,
       r.prompt_tokens, r.completion_tokens, r.status, r.error, r.duration_ms, r.created_at
FROM agent_runs r LEFT JOIN users u ON u.id = r.user_id
WHERE r.agent = $1
ORDER BY r.created_at DESC, r.id DESC
LIMIT $2 OFFSET $3`, agent, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.AgentRunRecord{}
	for rows.Next() {
		var run service.AgentRunRecord
		var userID sql.NullInt64
		var steps []byte
		if err := rows.Scan(&run.ID, &run.Agent, &userID, &run.UserEmail, &run.Question, &run.Answer, &steps, &run.Model,
			&run.ModelCalls, &run.PromptTokens, &run.CompletionTokens, &run.Status, &run.Error, &run.DurationMs, &run.CreatedAt); err != nil {
			return nil, 0, err
		}
		if userID.Valid {
			run.UserID = &userID.Int64
		}
		if json.Unmarshal(steps, &run.Steps) != nil || run.Steps == nil {
			run.Steps = []service.AgentStep{}
		}
		out = append(out, run)
	}
	return out, total, rows.Err()
}

func (r *agentRunRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	out, err := r.db.ExecContext(ctx, `DELETE FROM agent_runs WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}
