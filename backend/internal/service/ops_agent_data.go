package service

import (
	"context"
	"time"
)

// What the ops assistant (运维助手) reads about this site: failed requests (ops_error_logs) next to
// successful ones (usage_logs), the upstream accounts' state and the alert events. All read-only.

// OpsAgentErrorGroup is one kind of failure in a window: same status, upstream status, phase, type
// and message.
type OpsAgentErrorGroup struct {
	StatusCode      int       `json:"status_code"`
	UpstreamStatus  *int      `json:"upstream_status,omitempty"`
	Phase           string    `json:"phase"`
	Type            string    `json:"type"`
	Message         string    `json:"message"`
	BusinessLimited bool      `json:"business_limited,omitempty"`
	Count           int       `json:"count"`
	Users           int       `json:"users"`
	Models          []string  `json:"models"`
	Accounts        []string  `json:"accounts"`
	First           time.Time `json:"first"`
	Last            time.Time `json:"last"`
}

// OpsAgentCount: successes and failures for one account, model or user.
type OpsAgentCount struct {
	Key     string `json:"key"`
	Success int    `json:"success"`
	Errors  int    `json:"errors"`
}

type OpsAgentErrorSummary struct {
	Success         int                  `json:"success"`
	Errors          int                  `json:"errors"`
	BusinessLimited int                  `json:"business_limited"`
	Groups          []OpsAgentErrorGroup `json:"groups"`
	ByAccount       []OpsAgentCount      `json:"by_account"`
	ByModel         []OpsAgentCount      `json:"by_model"`
	ByUser          []OpsAgentCount      `json:"by_user"`
}

type OpsAgentErrorFilter struct {
	Start, End time.Time
	Model      string
	AccountID  int64
	UserID     int64
	StatusCode int
	Limit      int
}

// OpsAgentErrorRow is one failed request. Tried lists the accounts the gateway tried, in order.
type OpsAgentErrorRow struct {
	ID             int64     `json:"id"`
	Time           time.Time `json:"time"`
	UserID         *int64    `json:"user_id,omitempty"`
	UserEmail      string    `json:"user_email,omitempty"`
	Group          string    `json:"group,omitempty"`
	Account        string    `json:"account,omitempty"`
	Model          string    `json:"model,omitempty"`
	Path           string    `json:"path,omitempty"`
	Stream         bool      `json:"stream"`
	StatusCode     int       `json:"status_code"`
	UpstreamStatus *int      `json:"upstream_status,omitempty"`
	Phase          string    `json:"phase"`
	Type           string    `json:"type"`
	Message        string    `json:"message"`
	UpstreamDetail string    `json:"upstream_detail,omitempty"`
	Tried          []string  `json:"tried,omitempty"`
	DurationMs     *int      `json:"duration_ms,omitempty"`
}

type OpsAgentTimelineBucket struct {
	Start   time.Time `json:"start"`
	Success int       `json:"success"`
	Errors  int       `json:"errors"`
}

// OpsAgentAccount is an upstream account's scheduling state; BaseHost is the API host of an
// API-key account (never the key itself).
type OpsAgentAccount struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Platform               string     `json:"platform"`
	Type                   string     `json:"type"`
	Status                 string     `json:"status"`
	Schedulable            bool       `json:"schedulable"`
	Concurrency            int        `json:"concurrency"`
	Priority               int        `json:"priority"`
	BaseHost               string     `json:"base_host,omitempty"`
	Groups                 []string   `json:"groups"`
	ErrorMessage           string     `json:"error_message,omitempty"`
	RateLimitedAt          *time.Time `json:"rate_limited_at,omitempty"`
	RateLimitResetAt       *time.Time `json:"rate_limit_reset_at,omitempty"`
	OverloadUntil          *time.Time `json:"overload_until,omitempty"`
	TempUnschedulableUntil *time.Time `json:"temp_unschedulable_until,omitempty"`
	TempUnschedulableWhy   string     `json:"temp_unschedulable_reason,omitempty"`
	LastUsedAt             *time.Time `json:"last_used_at,omitempty"`
}

type OpsAgentAlert struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Severity    string     `json:"severity"`
	Status      string     `json:"status"`
	FiredAt     time.Time  `json:"fired_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type OpsAgentRepository interface {
	ErrorSummary(ctx context.Context, start, end time.Time) (*OpsAgentErrorSummary, error)
	ListErrors(ctx context.Context, f OpsAgentErrorFilter) ([]OpsAgentErrorRow, error)
	Timeline(ctx context.Context, start, end time.Time, bucket time.Duration) ([]OpsAgentTimelineBucket, error)
	Accounts(ctx context.Context) ([]OpsAgentAccount, error)
	AlertEvents(ctx context.Context, since time.Time, limit int) ([]OpsAgentAlert, error)
}
