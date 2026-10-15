package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

// 请求内容日志: for accounts an admin explicitly turns on, keep a small redacted record of each API
// request — the latest user message, the tool calls the model made and the head of its reply —
// for 7 days. Every enable / disable / view by an admin is written to an access log (180 days).
// Invisible to the account owner: capture never changes a request, never adds latency to it (the
// write is async and bounded) and is never exposed through user-facing APIs.

const (
	ContentLogRetentionDays       = 7
	ContentLogAccessRetentionDays = 180

	ContentLogActionEnable  = "enable"
	ContentLogActionDisable = "disable"
	ContentLogActionView    = "view"
	ContentLogActionConfig  = "config"

	// SettingKeyContentLogRedactSecrets: "false" keeps captured text as sent (no secret masking);
	// anything else (or no row) masks secrets before storage.
	SettingKeyContentLogRedactSecrets = "content_log_redact_secrets"

	contentLogReloadInterval      = 30 * time.Second
	contentLogQueueCapacity       = 1024
	contentLogBatchSize           = 100
	contentLogFlushInterval       = time.Second
	contentLogRetentionInterval   = time.Hour
	contentLogRetentionStartDelay = 2 * time.Minute
	contentLogRetentionBatchSize  = 5000
	contentLogDropLogInterval     = time.Minute
	contentLogNoteMaxRunes        = 500
	contentLogListDefaultLimit    = 50
	contentLogListMaxLimit        = 200
	contentLogAccessDefaultLimit  = 100
	contentLogAccessMaxLimit      = 500
)

var ErrContentLogUserNotFound = infraerrors.NotFound("CONTENT_LOG_USER_NOT_FOUND", "用户不存在（User not found）")
var ErrContentLogUserIDRequired = infraerrors.BadRequest("CONTENT_LOG_USER_ID_REQUIRED", "请指定用户 ID（user_id is required）")

// RequestContentLog is one captured request.
type RequestContentLog struct {
	ID              int64             `json:"id"`
	CreatedAt       time.Time         `json:"created_at"`
	UserID          int64             `json:"user_id"`
	APIKeyID        *int64            `json:"api_key_id"`
	GroupID         *int64            `json:"group_id"`
	RequestID       string            `json:"request_id"`
	SessionID       string            `json:"session_id"`
	Model           string            `json:"model"`
	InboundEndpoint string            `json:"inbound_endpoint"`
	UserAgent       string            `json:"user_agent"`
	IPAddress       string            `json:"ip_address"`
	UserText        string            `json:"user_text"`
	ToolCalls       []ContentToolCall `json:"tool_calls"`
	ReplyHead       string            `json:"reply_head"`
	InputTokens     int               `json:"input_tokens"`
	OutputTokens    int               `json:"output_tokens"`
	CacheReadTokens int               `json:"cache_read_tokens"`
}

// ContentLogTarget is an account under content logging, with what the admin list shows.
type ContentLogTarget struct {
	UserID         int64      `json:"user_id"`
	Email          string     `json:"email"`
	Username       string     `json:"username"`
	Note           string     `json:"note"`
	CreatedBy      *int64     `json:"created_by"`
	CreatedByEmail string     `json:"created_by_email"`
	CreatedAt      time.Time  `json:"created_at"`
	LogCount       int64      `json:"log_count"`
	LastLoggedAt   *time.Time `json:"last_logged_at"`
}

// ContentLogAccess is one admin action on the content log.
type ContentLogAccess struct {
	ID           int64     `json:"id"`
	AdminID      int64     `json:"admin_id"`
	AdminEmail   string    `json:"admin_email"`
	Action       string    `json:"action"`
	TargetUserID int64     `json:"target_user_id"`
	TargetEmail  string    `json:"target_email"`
	Detail       string    `json:"detail"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// ContentLogFilter selects captured requests, newest first.
type ContentLogFilter struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	BeforeID  int64  `json:"before_id,omitempty"`
	Limit     int    `json:"limit"`
}

// ContentLogPage is a page of captured requests.
type ContentLogPage struct {
	Items         []RequestContentLog `json:"items"`
	NextBeforeID  *int64              `json:"next_before_id"`
	Enabled       bool                `json:"enabled"`
	RetentionDays int                 `json:"retention_days"`
	RedactSecrets bool                `json:"redact_secrets"`
}

type ContentLogRepository interface {
	ListTargetIDs(ctx context.Context) ([]int64, error)
	ListTargets(ctx context.Context) ([]ContentLogTarget, error)
	UserExists(ctx context.Context, userID int64) (bool, error)
	// EnableTarget upserts the target (a repeat only updates the note) and writes the audit row in
	// one transaction.
	EnableTarget(ctx context.Context, userID int64, note string, audit ContentLogAccess) error
	// DisableTarget removes the target and writes the audit row in one transaction.
	DisableTarget(ctx context.Context, userID int64, audit ContentLogAccess) error
	InsertAccess(ctx context.Context, a ContentLogAccess) error
	ListAccess(ctx context.Context, targetUserID int64, limit int) ([]ContentLogAccess, error)
	BatchInsertLogs(ctx context.Context, logs []*RequestContentLog) (int64, error)
	ListLogs(ctx context.Context, f ContentLogFilter) ([]RequestContentLog, error)
	DeleteLogsBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
	DeleteAccessBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
	// GetRedactSecrets reads the masking switch (true when unset).
	GetRedactSecrets(ctx context.Context) (bool, error)
	// SetRedactSecrets writes the masking switch and its audit row in one transaction.
	SetRedactSecrets(ctx context.Context, on bool, audit ContentLogAccess) error
}

// ContentLogRecorder is the usage-recording hook: the gateway services hand over the usage row and
// the request's capture; the recorder decides whether to keep it.
type ContentLogRecorder interface {
	RecordUsageContent(usageLog *UsageLog, capture *ContentCapture)
}

type ContentLogService struct {
	repo ContentLogRepository

	mu      sync.RWMutex
	targets map[int64]struct{}
	// version counts target changes made here; a reload that started before one is discarded.
	version uint64
	// redact masks secrets before storage (the 遮盖密钥 switch); refreshed with the targets.
	redact atomic.Bool

	queue chan *RequestContentLog
	ctx   context.Context
	stop  context.CancelFunc
	wg    sync.WaitGroup

	startOnce   sync.Once
	stopOnce    sync.Once
	dropped     uint64
	lastDropLog atomic.Int64
	now         func() time.Time
}

func NewContentLogService(repo ContentLogRepository) *ContentLogService {
	ctx, cancel := context.WithCancel(context.Background())
	s := &ContentLogService{
		repo:    repo,
		targets: map[int64]struct{}{},
		queue:   make(chan *RequestContentLog, contentLogQueueCapacity),
		ctx:     ctx,
		stop:    cancel,
		now:     time.Now,
	}
	s.redact.Store(true)
	return s
}

// ProvideContentLogService loads the targets, starts the writer, the reload and the retention
// loops, and hooks the recorder into both gateway services' usage recording.
func ProvideContentLogService(repo ContentLogRepository, openai *OpenAIGatewayService, gateway *GatewayService) *ContentLogService {
	s := NewContentLogService(repo)
	if openai != nil {
		openai.SetContentLogRecorder(s)
	}
	if gateway != nil {
		gateway.SetContentLogRecorder(s)
	}
	s.Start()
	return s
}

func (s *ContentLogService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		s.reload(ctx)
		cancel()
		s.wg.Add(3)
		go s.runReloadLoop()
		go s.runWriter()
		go s.runRetentionLoop()
	})
}

// Stop stops the loops and flushes what is still queued.
func (s *ContentLogService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.stop()
		s.wg.Wait()
	})
}

func (s *ContentLogService) reload(ctx context.Context) {
	if on, err := s.repo.GetRedactSecrets(ctx); err != nil {
		logger.LegacyPrintf("service.content_log", "[ContentLog] reload redact switch failed: %v", err)
	} else {
		s.redact.Store(on)
	}
	s.mu.RLock()
	startVersion := s.version
	s.mu.RUnlock()
	ids, err := s.repo.ListTargetIDs(ctx)
	if err != nil {
		logger.LegacyPrintf("service.content_log", "[ContentLog] reload targets failed: %v", err)
		return
	}
	m := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	s.mu.Lock()
	if s.version == startVersion {
		s.targets = m
	}
	s.mu.Unlock()
}

func (s *ContentLogService) runReloadLoop() {
	defer s.wg.Done()
	t := time.NewTicker(contentLogReloadInterval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			contentLogSafeStep("reload", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				s.reload(ctx)
			})
		}
	}
}

// IsTarget reports whether the account is under content logging (in-memory, refreshed every 30s
// and on every change made here). This is the only cost a non-target request pays.
func (s *ContentLogService) IsTarget(userID int64) bool {
	if s == nil || userID <= 0 {
		return false
	}
	s.mu.RLock()
	_, ok := s.targets[userID]
	s.mu.RUnlock()
	return ok
}

func (s *ContentLogService) setTarget(userID int64, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[int64]struct{}, len(s.targets)+1)
	for id := range s.targets {
		next[id] = struct{}{}
	}
	if on {
		next[userID] = struct{}{}
	} else {
		delete(next, userID)
	}
	s.targets = next
	s.version++
}

// RecordUsageContent enqueues one captured request, built from the usage row being written (same
// request_id). Never blocks and never panics; drops when the queue is full.
func (s *ContentLogService) RecordUsageContent(usageLog *UsageLog, capture *ContentCapture) {
	if s == nil || usageLog == nil || capture == nil {
		return
	}
	defer func() { _ = recover() }()
	if !s.IsTarget(usageLog.UserID) {
		return
	}
	row := &RequestContentLog{
		CreatedAt: usageLog.CreatedAt,
		UserID:    usageLog.UserID,
		RequestID: usageLog.RequestID,
		Model:     strings.TrimSpace(usageLog.RequestedModel),
		// Cut to twice the stored size before queueing (bounds queue memory); the writer redacts
		// with the slack still there, then cuts to the stored size.
		UserText:        contentLogHeadRunes(capture.UserText, 2*ContentLogUserTextMaxRunes),
		ReplyHead:       contentLogHeadRunes(capture.ReplyHead, 2*ContentLogReplyHeadMaxRunes),
		InputTokens:     usageLog.InputTokens,
		OutputTokens:    usageLog.OutputTokens,
		CacheReadTokens: usageLog.CacheReadTokens,
	}
	if len(capture.ToolCalls) > 0 {
		calls := capture.ToolCalls
		if len(calls) > ContentLogMaxToolCalls {
			calls = calls[:ContentLogMaxToolCalls]
		}
		row.ToolCalls = make([]ContentToolCall, 0, len(calls))
		for _, call := range calls {
			row.ToolCalls = append(row.ToolCalls, ContentToolCall{
				Name:      contentLogHeadRunes(call.Name, 100),
				Arguments: contentLogHeadRunes(call.Arguments, 2*ContentLogToolArgumentsMaxRunes),
			})
		}
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = s.now()
	}
	if row.Model == "" {
		row.Model = usageLog.Model
	}
	if usageLog.APIKeyID > 0 {
		id := usageLog.APIKeyID
		row.APIKeyID = &id
	}
	if usageLog.GroupID != nil {
		id := *usageLog.GroupID
		row.GroupID = &id
	}
	row.SessionID = derefString(usageLog.SessionID)
	row.InboundEndpoint = derefString(usageLog.InboundEndpoint)
	row.UserAgent = derefString(usageLog.UserAgent)
	row.IPAddress = derefString(usageLog.IPAddress)
	s.enqueue(row)
}

func (s *ContentLogService) enqueue(row *RequestContentLog) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}
	select {
	case s.queue <- row:
	default:
		n := atomic.AddUint64(&s.dropped, 1)
		now := s.now().UnixNano()
		last := s.lastDropLog.Load()
		if now-last >= int64(contentLogDropLogInterval) && s.lastDropLog.CompareAndSwap(last, now) {
			logger.LegacyPrintf("service.content_log", "[ContentLog] queue full, dropped %d rows so far", n)
		}
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// prepareContentLogRow truncates everything that is stored and, when redact is on, masks secrets
// first (runs in the writer).
func prepareContentLogRow(row *RequestContentLog, redact bool) {
	clean := func(s string, maxRunes int) string {
		if redact {
			return redactAndTrim(s, maxRunes)
		}
		return contentLogTrimRunes(strings.TrimSpace(strings.ToValidUTF8(s, "")), maxRunes)
	}
	row.UserText = clean(row.UserText, ContentLogUserTextMaxRunes)
	row.ReplyHead = clean(row.ReplyHead, ContentLogReplyHeadMaxRunes)
	calls := make([]ContentToolCall, 0, len(row.ToolCalls))
	for _, call := range row.ToolCalls {
		if len(calls) >= ContentLogMaxToolCalls {
			break
		}
		calls = append(calls, ContentToolCall{
			Name:      contentLogTrimRunes(strings.TrimSpace(call.Name), 100),
			Arguments: clean(call.Arguments, ContentLogToolArgumentsMaxRunes),
		})
	}
	row.ToolCalls = calls
	row.RequestID = contentLogTrimRunes(row.RequestID, 128)
	row.SessionID = contentLogTrimRunes(row.SessionID, 256)
	row.Model = contentLogTrimRunes(row.Model, 128)
	row.InboundEndpoint = contentLogTrimRunes(row.InboundEndpoint, 256)
	row.UserAgent = contentLogTrimRunes(row.UserAgent, 512)
	row.IPAddress = contentLogTrimRunes(row.IPAddress, 64)
}

func redactAndTrim(s string, maxRunes int) string {
	s = strings.ToValidUTF8(s, "")
	return contentLogTrimRunes(strings.TrimSpace(logredact.RedactContent(s)), maxRunes)
}

// contentLogHeadRunes keeps the first maxRunes runes, without a marker.
func contentLogHeadRunes(s string, maxRunes int) string {
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}

func contentLogTrimRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i] + "…"
		}
		count++
	}
	return s
}

// contentLogSafeStep runs one unit of background work; a panic is logged and the loop goes on.
func contentLogSafeStep(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.LegacyPrintf("service.content_log", "[ContentLog] %s panicked: %v", name, r)
		}
	}()
	fn()
}

func (s *ContentLogService) runWriter() {
	defer s.wg.Done()
	ticker := time.NewTicker(contentLogFlushInterval)
	defer ticker.Stop()

	batch := make([]*RequestContentLog, 0, contentLogBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		contentLogSafeStep("write", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := s.repo.BatchInsertLogs(ctx, batch); err != nil {
				logger.LegacyPrintf("service.content_log", "[ContentLog] write %d rows failed: %v", len(batch), err)
			}
		})
		batch = batch[:0]
	}
	add := func(row *RequestContentLog) {
		if row == nil {
			return
		}
		ok := func() (ok bool) {
			defer func() {
				if recover() != nil {
					ok = false
				}
			}()
			prepareContentLogRow(row, s.redact.Load())
			return true
		}()
		if !ok {
			return
		}
		batch = append(batch, row)
		if len(batch) >= contentLogBatchSize {
			flush()
		}
	}
	for {
		select {
		case <-s.ctx.Done():
			for {
				select {
				case row := <-s.queue:
					add(row)
				default:
					flush()
					return
				}
			}
		case row := <-s.queue:
			add(row)
		case <-ticker.C:
			flush()
		}
	}
}

func (s *ContentLogService) runRetentionLoop() {
	defer s.wg.Done()
	timer := time.NewTimer(contentLogRetentionStartDelay)
	defer timer.Stop()
	select {
	case <-s.ctx.Done():
		return
	case <-timer.C:
	}
	contentLogSafeStep("retention", s.runRetentionOnce)
	ticker := time.NewTicker(contentLogRetentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			contentLogSafeStep("retention", s.runRetentionOnce)
		}
	}
}

func (s *ContentLogService) runRetentionOnce() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()
	now := s.now().UTC()
	purge := func(name string, cutoff time.Time, del func(context.Context, time.Time, int) (int64, error)) {
		for {
			n, err := del(ctx, cutoff, contentLogRetentionBatchSize)
			if err != nil {
				logger.LegacyPrintf("service.content_log", "[ContentLog] retention cleanup of %s failed: %v", name, err)
				return
			}
			if n < contentLogRetentionBatchSize {
				return
			}
		}
	}
	purge("request_content_logs", now.AddDate(0, 0, -ContentLogRetentionDays), s.repo.DeleteLogsBefore)
	purge("content_log_access_logs", now.AddDate(0, 0, -ContentLogAccessRetentionDays), s.repo.DeleteAccessBefore)
}

// --- admin ---

// ListTargets returns the accounts under content logging.
func (s *ContentLogService) ListTargets(ctx context.Context) ([]ContentLogTarget, error) {
	items, err := s.repo.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ContentLogTarget{}
	}
	return items, nil
}

// Enable turns content logging on for userID (idempotent; updates the note) and audits it.
func (s *ContentLogService) Enable(ctx context.Context, adminID, userID int64, note, ip, ua string) error {
	if userID <= 0 {
		return ErrContentLogUserNotFound
	}
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrContentLogUserNotFound
	}
	note = contentLogTrimRunes(strings.TrimSpace(note), contentLogNoteMaxRunes)
	audit := ContentLogAccess{AdminID: adminID, Action: ContentLogActionEnable, TargetUserID: userID, Detail: note, IPAddress: ip, UserAgent: ua}
	if err := s.repo.EnableTarget(ctx, userID, note, audit); err != nil {
		return err
	}
	s.setTarget(userID, true) // only after the change and its audit row committed
	return nil
}

// Disable turns content logging off for userID and audits it. Rows already kept stay until they expire.
func (s *ContentLogService) Disable(ctx context.Context, adminID, userID int64, ip, ua string) error {
	if userID <= 0 {
		return ErrContentLogUserNotFound
	}
	audit := ContentLogAccess{AdminID: adminID, Action: ContentLogActionDisable, TargetUserID: userID, IPAddress: ip, UserAgent: ua}
	if err := s.repo.DisableTarget(ctx, userID, audit); err != nil {
		return err
	}
	s.setTarget(userID, false)
	return nil
}

// ListLogs returns a page of the account's captured requests; every call is audited as a view.
func (s *ContentLogService) ListLogs(ctx context.Context, adminID int64, f ContentLogFilter, ip, ua string) (*ContentLogPage, error) {
	if f.UserID <= 0 {
		return nil, ErrContentLogUserIDRequired
	}
	if f.Limit <= 0 {
		f.Limit = contentLogListDefaultLimit
	}
	if f.Limit > contentLogListMaxLimit {
		f.Limit = contentLogListMaxLimit
	}
	f.SessionID = strings.TrimSpace(f.SessionID)
	if f.BeforeID < 0 {
		f.BeforeID = 0
	}
	detail, _ := json.Marshal(f)
	if err := s.repo.InsertAccess(ctx, ContentLogAccess{AdminID: adminID, Action: ContentLogActionView, TargetUserID: f.UserID, Detail: string(detail), IPAddress: ip, UserAgent: ua}); err != nil {
		// No audit row, no data.
		return nil, err
	}
	items, err := s.repo.ListLogs(ctx, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []RequestContentLog{}
	}
	for i := range items {
		if items[i].ToolCalls == nil {
			items[i].ToolCalls = []ContentToolCall{}
		}
	}
	page := &ContentLogPage{Items: items, Enabled: s.IsTarget(f.UserID), RetentionDays: ContentLogRetentionDays, RedactSecrets: s.RedactSecrets()}
	if len(items) == f.Limit {
		last := items[len(items)-1].ID
		page.NextBeforeID = &last
	}
	return page, nil
}

// ListAccess returns the audit trail, newest first (targetUserID 0 = every account). Not audited itself.
func (s *ContentLogService) ListAccess(ctx context.Context, targetUserID int64, limit int) ([]ContentLogAccess, error) {
	if limit <= 0 {
		limit = contentLogAccessDefaultLimit
	}
	if limit > contentLogAccessMaxLimit {
		limit = contentLogAccessMaxLimit
	}
	items, err := s.repo.ListAccess(ctx, targetUserID, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ContentLogAccess{}
	}
	return items, nil
}

// SetContentLogRecorder hooks the content log into OpenAI usage recording (wired at startup).
func (s *OpenAIGatewayService) SetContentLogRecorder(r ContentLogRecorder) {
	if s != nil {
		s.contentLog = r
	}
}

// SetContentLogRecorder hooks the content log into gateway usage recording (wired at startup).
func (s *GatewayService) SetContentLogRecorder(r ContentLogRecorder) {
	if s != nil {
		s.contentLog = r
	}
}

// recordContentLog hands a captured request to the content log; a no-op unless the request was captured.
func recordContentLog(r ContentLogRecorder, usageLog *UsageLog, capture *ContentCapture) {
	if r == nil || usageLog == nil || capture == nil {
		return
	}
	defer func() { _ = recover() }()
	r.RecordUsageContent(usageLog, capture)
}

// RedactSecrets reports whether secrets are masked before storage.
func (s *ContentLogService) RedactSecrets() bool {
	if s == nil {
		return true
	}
	return s.redact.Load()
}

// SetRedactSecrets turns secret masking on or off for rows written from now on, and audits it.
// Rows already stored keep whatever they were stored with.
func (s *ContentLogService) SetRedactSecrets(ctx context.Context, adminID int64, on bool, ip, ua string) error {
	audit := ContentLogAccess{AdminID: adminID, Action: ContentLogActionConfig, Detail: fmt.Sprintf("redact_secrets=%t", on), IPAddress: ip, UserAgent: ua}
	if err := s.repo.SetRedactSecrets(ctx, on, audit); err != nil {
		return err
	}
	s.redact.Store(on)
	return nil
}
