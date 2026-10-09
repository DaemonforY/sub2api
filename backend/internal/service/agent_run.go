package service

import (
	"context"
	"time"
)

// Run log for tool-using assistants (agent_runs): what was asked, which tools ran with what result,
// and the usage — so admins can see what the assistant looked up when an answer went wrong.

const (
	AgentSupport        = "support"
	agentRunRetention   = 30 * 24 * time.Hour
	agentRunCleanupEach = 100 // delete old rows when a new run's ID is a multiple of this
)

type AgentRunRecord struct {
	ID               int64       `json:"id"`
	Agent            string      `json:"agent"`
	UserID           *int64      `json:"user_id"`
	UserEmail        string      `json:"user_email,omitempty"`
	Question         string      `json:"question"`
	Answer           string      `json:"answer"`
	Steps            []AgentStep `json:"steps"`
	Model            string      `json:"model"`
	ModelCalls       int         `json:"model_calls"`
	PromptTokens     int         `json:"prompt_tokens"`
	CompletionTokens int         `json:"completion_tokens"`
	Status           string      `json:"status"` // ok | error
	Error            string      `json:"error,omitempty"`
	DurationMs       int64       `json:"duration_ms"`
	CreatedAt        time.Time   `json:"created_at"`
}

type AgentRunRepository interface {
	Create(ctx context.Context, run *AgentRunRecord) (int64, error)
	List(ctx context.Context, agent string, page, pageSize int) ([]AgentRunRecord, int, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// recordAgentRun saves a run in the background-safe way: it never fails the caller.
func recordAgentRun(ctx context.Context, repo AgentRunRepository, run *AgentRunRecord, now time.Time) {
	if repo == nil || run == nil {
		return
	}
	if run.Steps == nil {
		run.Steps = []AgentStep{}
	}
	run.Question = agentCut(run.Question, 2000)
	run.Answer = agentCut(run.Answer, 8000)
	run.Error = agentCut(run.Error, 500)
	id, err := repo.Create(ctx, run)
	if err == nil && id%agentRunCleanupEach == 0 {
		_, _ = repo.DeleteBefore(ctx, now.Add(-agentRunRetention))
	}
}
