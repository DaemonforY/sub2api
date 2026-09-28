package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Gin context keys written by the gateway handlers (see handler/ops_error_logger.go)
// and read back by the gateway request log middleware after the handler chain runs.
const (
	OpsModelContextKey     = "ops_model"
	OpsAccountIDContextKey = "ops_account_id"
)

const (
	gatewayRequestLogQueueCapacity = 8192
	gatewayRequestLogBatchSize     = 200
	gatewayRequestLogFlushInterval = time.Second

	gatewayRequestLogRetentionInterval     = 6 * time.Hour
	gatewayRequestLogRetentionStartupDelay = 3 * time.Minute
	gatewayRequestLogRetentionBatchSize    = 5000

	// DefaultGatewayRequestLogRetentionDays is used when request_log.retention_days is unset.
	DefaultGatewayRequestLogRetentionDays = 30
)

// GatewayRequestLog is one API gateway request, recorded whether it succeeded or not.
type GatewayRequestLog struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	URL        string    `json:"url"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	Success    bool      `json:"success"`
	ErrorCode  string    `json:"error_code"`
	DurationMs int64     `json:"duration_ms"`
	// APIKey is the full credential the caller presented (also for keys that failed lookup).
	APIKey    string `json:"api_key"`
	APIKeyID  *int64 `json:"api_key_id,omitempty"`
	UserID    *int64 `json:"user_id,omitempty"`
	GroupID   *int64 `json:"group_id,omitempty"`
	AccountID *int64 `json:"account_id,omitempty"`
	Model     string `json:"model"`
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`

	// Read-side joins (not persisted on this table).
	UserEmail  string `json:"user_email,omitempty"`
	APIKeyName string `json:"api_key_name,omitempty"`
}

// GatewayRequestLogFilter holds list query conditions.
type GatewayRequestLogFilter struct {
	Page     int
	PageSize int

	StartTime *time.Time
	EndTime   *time.Time
	UserID    *int64
	APIKeyID  *int64
	// Success: nil = all; true = status < 400; false = status >= 400.
	Success    *bool
	StatusCode *int
	Method     string
	Model      string
	// Query fuzzy-matches url, api_key, model, client_ip and the user's email.
	Query string
}

// GatewayRequestLogList is a paginated result.
type GatewayRequestLogList struct {
	Logs     []*GatewayRequestLog
	Total    int
	Page     int
	PageSize int
}

// GatewayRequestLogRepository persists gateway request logs.
type GatewayRequestLogRepository interface {
	BatchInsert(ctx context.Context, logs []*GatewayRequestLog) (int64, error)
	List(ctx context.Context, filter *GatewayRequestLogFilter) (*GatewayRequestLogList, error)
	DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
}

// GatewayRequestLogService records every gateway request asynchronously (never
// blocking the request path) and prunes rows past the retention window.
type GatewayRequestLogService struct {
	repo          GatewayRequestLogRepository
	enabled       bool
	retentionDays int

	queue chan *GatewayRequestLog

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	droppedCount uint64
	writeFailed  uint64
	writtenCount uint64
}

// NewGatewayRequestLogService creates the service. retentionDays <= 0 keeps rows forever.
func NewGatewayRequestLogService(repo GatewayRequestLogRepository, enabled bool, retentionDays int) *GatewayRequestLogService {
	ctx, cancel := context.WithCancel(context.Background())
	return &GatewayRequestLogService{
		repo:          repo,
		enabled:       enabled,
		retentionDays: retentionDays,
		queue:         make(chan *GatewayRequestLog, gatewayRequestLogQueueCapacity),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Enabled reports whether new requests are being recorded.
func (s *GatewayRequestLogService) Enabled() bool {
	return s != nil && s.enabled && s.repo != nil
}

// Start launches the writer and retention goroutines.
func (s *GatewayRequestLogService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.wg.Add(2)
	go s.runWriter()
	go s.runRetentionLoop()
}

// Stop stops the service and flushes whatever is still queued.
func (s *GatewayRequestLogService) Stop() {
	if s == nil {
		return
	}
	s.cancel()
	s.wg.Wait()
}

// Record enqueues one entry without blocking; entries are dropped (and counted) when the queue is full.
func (s *GatewayRequestLogService) Record(entry *GatewayRequestLog) {
	if !s.Enabled() || entry == nil {
		return
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	select {
	case <-s.ctx.Done():
		return
	default:
	}
	select {
	case s.queue <- entry:
	default:
		atomic.AddUint64(&s.droppedCount, 1)
	}
}

// List returns a page of logs.
func (s *GatewayRequestLogService) List(ctx context.Context, filter *GatewayRequestLogFilter) (*GatewayRequestLogList, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("gateway request log service unavailable")
	}
	return s.repo.List(ctx, filter)
}

func (s *GatewayRequestLogService) runWriter() {
	defer s.wg.Done()

	ticker := time.NewTicker(gatewayRequestLogFlushInterval)
	defer ticker.Stop()

	batch := make([]*GatewayRequestLog, 0, gatewayRequestLogBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		inserted, err := s.repo.BatchInsert(ctx, batch)
		cancel()
		if err != nil {
			atomic.AddUint64(&s.writeFailed, uint64(len(batch)))
			_, _ = fmt.Fprintf(os.Stderr, "time=%s level=WARN msg=\"gateway request log flush failed\" err=%v batch=%d\n",
				time.Now().Format(time.RFC3339Nano), err, len(batch))
		} else {
			atomic.AddUint64(&s.writtenCount, uint64(inserted))
		}
		batch = batch[:0]
	}
	add := func(item *GatewayRequestLog) {
		if item == nil {
			return
		}
		batch = append(batch, item)
		if len(batch) >= gatewayRequestLogBatchSize {
			flush()
		}
	}

	for {
		select {
		case <-s.ctx.Done():
			for {
				select {
				case item := <-s.queue:
					add(item)
				default:
					flush()
					return
				}
			}
		case item := <-s.queue:
			add(item)
		case <-ticker.C:
			flush()
		}
	}
}

func (s *GatewayRequestLogService) runRetentionLoop() {
	defer s.wg.Done()
	if s.retentionDays <= 0 {
		return
	}

	startupTimer := time.NewTimer(gatewayRequestLogRetentionStartupDelay)
	defer startupTimer.Stop()
	select {
	case <-s.ctx.Done():
		return
	case <-startupTimer.C:
	}

	ticker := time.NewTicker(gatewayRequestLogRetentionInterval)
	defer ticker.Stop()

	s.runRetentionOnce()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runRetentionOnce()
		}
	}
}

func (s *GatewayRequestLogService) runRetentionOnce() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()

	cutoff := time.Now().UTC().AddDate(0, 0, -s.retentionDays)
	for {
		deleted, err := s.repo.DeleteBefore(ctx, cutoff, gatewayRequestLogRetentionBatchSize)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "time=%s level=WARN msg=\"gateway request log retention cleanup failed\" err=%v\n",
				time.Now().Format(time.RFC3339Nano), err)
			return
		}
		if deleted == 0 {
			return
		}
	}
}
