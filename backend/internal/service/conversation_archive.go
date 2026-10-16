package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// 对话归档 (conversation archive): when the admin enables it, every API request's new input
// items (deduped by session), the full model response and generated images are written to the
// operator's private S3-compatible bucket (Tencent COS) as gzip JSON with server-side
// encryption, indexed in PostgreSQL and kept retention_days. Only admins can browse it, and every
// browse is written to the shared access trail. Capture never changes a request: the request path
// pays one atomic load while it is off, all work after the response is async, memory is bounded
// (count + bytes) and a full queue drops (and counts) instead of blocking.

const (
	SettingKeyConversationArchiveConfig = "conversation_archive_config"

	ContentLogActionArchiveView   = "archive_view"
	ContentLogActionArchiveConfig = "archive_config"

	archiveDefaultPrefix        = "archive/"
	archiveDefaultRetentionDays = 180
	archiveMinRetentionDays     = 7
	archiveMaxRetentionDays     = 730
	archiveLifecycleRuleID      = "sub2api-conversation-archive"

	archiveReloadInterval      = 30 * time.Second
	archiveQueueMaxItems       = 4096
	archiveQueueMaxBytes       = 256 << 20
	archiveWorkerCount         = 2
	archiveUploadAttempts      = 3
	archiveUploadTimeout       = 60 * time.Second
	archiveDrainTimeout        = 5 * time.Second
	archiveRetentionInterval   = time.Hour
	archiveRetentionStartDelay = 3 * time.Minute
	archiveRetentionBatch      = 400
	archiveDeleteBatch         = 1000
	archiveDropLogInterval     = time.Minute
	archiveFetchParallel       = 8
	archiveObjectMaxBytes      = 32 << 20 // decoded size cap when reading a turn back
	archiveTurnsPageBytes      = 64 << 20 // compressed bytes (index column) per turns page
	archivePurgeInterval       = 10 * time.Minute
	archivePurgeBatch          = 20
	archivePurgeStaleAfter     = 24 * time.Hour
	archiveMaxPreviousPrefixes = 20
	archiveSessionsDefault     = 30
	archiveSessionsMax         = 100
	archiveTurnsDefault        = 20
	archiveTurnsMax            = 50
	archiveDefaultBrowseDays   = 30
	archiveStatsDays           = 30
	archivePurgedRemember      = 24 * time.Hour
)

var (
	ErrArchiveNotConfigured = infraerrors.BadRequest("CONVERSATION_ARCHIVE_NOT_CONFIGURED", "对话归档存储尚未配置（Archive storage is not configured）")
	ErrArchiveIncomplete    = infraerrors.BadRequest("CONVERSATION_ARCHIVE_INCOMPLETE", "开启归档前请填写存储桶、Access Key ID 和 Secret Access Key（Bucket, access key ID and secret access key are required）")
	ErrArchiveRetention     = infraerrors.BadRequest("CONVERSATION_ARCHIVE_RETENTION", "保留天数须在 7 到 730 之间（retention_days must be between 7 and 730）")
	ErrArchiveTurnNotFound  = infraerrors.NotFound("CONVERSATION_ARCHIVE_TURN_NOT_FOUND", "未找到该请求的归档（Archived turn not found）")
	ErrArchiveImageNotFound = infraerrors.NotFound("CONVERSATION_ARCHIVE_IMAGE_NOT_FOUND", "未找到该图片（Archived image not found）")
	ErrArchiveSessionParams = infraerrors.BadRequest("CONVERSATION_ARCHIVE_SESSION_REQUIRED", "请指定 user_id 和 session_key（user_id and session_key are required）")
	ErrArchiveRequestID     = infraerrors.BadRequest("CONVERSATION_ARCHIVE_REQUEST_ID_REQUIRED", "请指定 request_id（request_id is required）")
)

func archiveStorageError(code, zh, en string, err error) error {
	return infraerrors.BadRequest(code, fmt.Sprintf("%s（%s）：%v", zh, en, err))
}

// ArchiveSettings is the stored configuration (settings key conversation_archive_config). The
// secret is stored encrypted and never returned.
type ArchiveSettings struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key,omitempty"` //nolint:revive // AWS naming
	ForcePathStyle  bool   `json:"force_path_style"`
	RetentionDays   int    `json:"retention_days"`
	SSE             bool   `json:"sse"`
	// PreviousPrefixes are prefixes used before the current one (account deletion sweeps them too).
	PreviousPrefixes []string `json:"previous_prefixes,omitempty"`
}

// ArchiveConfigInput is the admin's PUT / test body. An empty secret keeps the stored one; a nil
// SSE means on.
type ArchiveConfigInput struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"` //nolint:revive // AWS naming
	ForcePathStyle  bool   `json:"force_path_style"`
	RetentionDays   int    `json:"retention_days"`
	SSE             *bool  `json:"sse"`
}

// ArchiveStoreConfig is what the object store factory needs (secret in plain text).
type ArchiveStoreConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
}

// ArchiveObjectStore is the bucket (S3-compatible; implemented in repository).
type ArchiveObjectStore interface {
	PutObject(ctx context.Context, key string, body []byte, contentType, contentEncoding string, sse bool) error
	// GetObject returns the stored bytes as stored (no transparent decompression) and the size
	// when known (-1 otherwise).
	GetObject(ctx context.Context, key string) (io.ReadCloser, int64, error)
	// DeleteObjects deletes up to 1000 keys.
	DeleteObjects(ctx context.Context, keys []string) error
	// DeletePrefix deletes every object under prefix and returns how many were deleted.
	DeletePrefix(ctx context.Context, prefix string) (int, error)
	// ApplyLifecycle adds (or replaces) the rule expiring prefix after days, keeping other rules.
	ApplyLifecycle(ctx context.Context, ruleID, prefix string, days int) error
}

// ArchiveObjectStoreFactory builds a store for a configuration (repository provides it).
type ArchiveObjectStoreFactory func(ctx context.Context, cfg ArchiveStoreConfig) (ArchiveObjectStore, error)

// ArchiveTurn is one index row.
type ArchiveTurn struct {
	ID              int64
	UserID          int64
	APIKeyID        *int64
	GroupID         *int64
	SessionKey      string
	RequestID       string
	CreatedAt       time.Time
	Model           string
	InboundEndpoint string
	UserAgent       string
	IPAddress       string
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
	ObjectKey       string
	Bytes           int
	NewItemCount    int
	Reset           bool
	ImageCount      int
	Preview         string
	// Email is the owner's email (filled by GetTurnByRequestID only).
	Email string
}

// ArchiveImage is one image index row.
type ArchiveImage struct {
	ID        int64
	TurnID    int64
	UserID    int64
	ObjectKey string
	Mime      string
	Bytes     int
	CreatedAt time.Time
}

// ArchivePurge is a deleted account whose archived objects still have to be removed.
type ArchivePurge struct {
	UserID      int64
	Prefixes    []string
	CreatedAt   time.Time
	Attempts    int
	UserDeleted bool
}

// ArchiveExpiredTurn is a turn past retention with the keys to delete.
type ArchiveExpiredTurn struct {
	ID        int64
	ObjectKey string
	ImageKeys []string
}

// ArchiveSessionFilter selects sessions, newest last turn first.
type ArchiveSessionFilter struct {
	UserID int64      `json:"user_id,omitempty"`
	Q      string     `json:"q,omitempty"`
	From   *time.Time `json:"from,omitempty"`
	To     *time.Time `json:"to,omitempty"`
	Before *time.Time `json:"before,omitempty"`
	Limit  int        `json:"limit"`
}

// ArchiveSession is one row of the session list.
type ArchiveSession struct {
	SessionKey   string    `json:"session_key"`
	UserID       int64     `json:"user_id"`
	Email        string    `json:"email"`
	FirstAt      time.Time `json:"first_at"`
	LastAt       time.Time `json:"last_at"`
	Turns        int64     `json:"turns"`
	Models       []string  `json:"models"`
	Endpoints    []string  `json:"endpoints"`
	UserAgent    string    `json:"user_agent"`
	Preview      string    `json:"preview"`
	ImageCount   int64     `json:"image_count"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
}

// ArchiveSessionPage is a page of sessions.
type ArchiveSessionPage struct {
	Items      []ArchiveSession `json:"items"`
	NextBefore *time.Time       `json:"next_before"`
}

// ArchiveImageView is an image of a turn as the admin API shows it.
type ArchiveImageView struct {
	ID    int64  `json:"id"`
	Mime  string `json:"mime"`
	Bytes int    `json:"bytes"`
}

// ArchiveTurnView is one turn with its archived object decoded.
type ArchiveTurnView struct {
	ID              int64                      `json:"id"`
	UserID          int64                      `json:"user_id"`
	Email           string                     `json:"email,omitempty"`
	SessionKey      string                     `json:"session_key"`
	Thread          string                     `json:"thread,omitempty"`
	Protocol        string                     `json:"protocol,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	RequestID       string                     `json:"request_id"`
	Model           string                     `json:"model"`
	InboundEndpoint string                     `json:"inbound_endpoint"`
	UserAgent       string                     `json:"user_agent"`
	IPAddress       string                     `json:"ip_address"`
	InputTokens     int                        `json:"input_tokens"`
	OutputTokens    int                        `json:"output_tokens"`
	CacheReadTokens int                        `json:"cache_read_tokens"`
	Reset           bool                       `json:"reset"`
	PrefixItems     int                        `json:"prefix_items"`
	Instructions    *string                    `json:"instructions,omitempty"`
	NewItems        []json.RawMessage          `json:"new_items"`
	RequestParams   map[string]json.RawMessage `json:"request_params"`
	Response        *ArchiveResponse           `json:"response"`
	Images          []ArchiveImageView         `json:"images"`
	ImageURLs       []string                   `json:"image_urls,omitempty"`
	Truncated       *archiveRecordTruncated    `json:"truncated,omitempty"`
	FetchError      string                     `json:"fetch_error,omitempty"`
}

// ArchiveTurnPage is a page of turns, newest first.
type ArchiveTurnPage struct {
	Items        []ArchiveTurnView `json:"items"`
	NextBeforeID *int64            `json:"next_before_id"`
}

// ArchiveDayStats is one day of the stats view.
type ArchiveDayStats struct {
	Day    string `json:"day"`
	Turns  int64  `json:"turns"`
	Users  int64  `json:"users"`
	Bytes  int64  `json:"bytes"`
	Images int64  `json:"images"`
}

// ArchiveStatus is the live state of the archive writer.
type ArchiveStatus struct {
	Active        bool       `json:"active"`
	QueueLen      int64      `json:"queue_len"`
	QueueBytes    int64      `json:"queue_bytes"`
	DroppedTotal  uint64     `json:"dropped_total"`
	FailedTotal   uint64     `json:"failed_total"`
	UploadedToday int64      `json:"uploaded_today"`
	LastUploadAt  *time.Time `json:"last_upload_at"`
	LastError     string     `json:"last_error"`
	LastErrorAt   *time.Time `json:"last_error_at"`
	// Bucket lifecycle rule, as last applied by this instance (on save while enabled, or by hand).
	LifecycleApplied   bool       `json:"lifecycle_applied"`
	LifecycleAppliedAt *time.Time `json:"lifecycle_applied_at"`
	LifecycleError     string     `json:"lifecycle_error"`
	PendingPurges      int        `json:"pending_purges"`
}

// ArchiveConfigView is GET/PUT /config.
type ArchiveConfigView struct {
	Enabled          bool          `json:"enabled"`
	Endpoint         string        `json:"endpoint"`
	Region           string        `json:"region"`
	Bucket           string        `json:"bucket"`
	Prefix           string        `json:"prefix"`
	AccessKeyID      string        `json:"access_key_id"`
	SecretConfigured bool          `json:"secret_configured"`
	ForcePathStyle   bool          `json:"force_path_style"`
	RetentionDays    int           `json:"retention_days"`
	SSE              bool          `json:"sse"`
	Status           ArchiveStatus `json:"status"`
}

// ConversationArchiveRepository is the index (and the settings row + access trail it writes).
type ConversationArchiveRepository interface {
	GetConfigRaw(ctx context.Context) (string, error)
	// SaveConfig writes the settings row and its audit row in one transaction.
	SaveConfig(ctx context.Context, raw string, audit ContentLogAccess) error
	InsertAccess(ctx context.Context, a ContentLogAccess) error
	// InsertTurn writes the turn and its images in one transaction and returns the turn id.
	InsertTurn(ctx context.Context, turn *ArchiveTurn, images []ArchiveImage) (int64, error)
	ListSessions(ctx context.Context, f ArchiveSessionFilter) ([]ArchiveSession, error)
	ListTurns(ctx context.Context, userID int64, sessionKey string, beforeID int64, limit int) ([]ArchiveTurn, error)
	GetTurnByRequestID(ctx context.Context, requestID string) (*ArchiveTurn, error)
	ListTurnImages(ctx context.Context, turnIDs []int64) (map[int64][]ArchiveImage, error)
	GetImage(ctx context.Context, id int64) (*ArchiveImage, error)
	DailyStats(ctx context.Context, since time.Time, tz string) ([]ArchiveDayStats, error)
	ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]ArchiveExpiredTurn, error)
	DeleteTurns(ctx context.Context, ids []int64) (int64, error)
	// ResolveUserID matches a user id, email or username (0 when none).
	ResolveUserID(ctx context.Context, q string) (int64, error)
	// QueuePurge records (or extends) a pending object purge for a deleted account.
	QueuePurge(ctx context.Context, userID int64, prefixes []string) error
	// ListPurges returns pending purges, oldest first, with whether the account is deleted.
	ListPurges(ctx context.Context, limit int) ([]ArchivePurge, error)
	FinishPurge(ctx context.Context, userID int64) error
	FailPurge(ctx context.Context, userID int64, errText string) error
}

// ConversationArchiveRecorder is the usage-recording hook.
type ConversationArchiveRecorder interface {
	EnqueueArchive(usageLog *UsageLog, capture *ArchiveCapture)
}

type archiveJob struct {
	userID          int64
	apiKeyID        int64
	groupID         *int64
	requestID       string
	sessionID       string
	model           string
	upstreamModel   string
	inboundEndpoint string
	userAgent       string
	ipAddress       string
	createdAt       time.Time
	inputTokens     int
	outputTokens    int
	cacheReadTokens int
	costUSD         float64
	capture         *ArchiveCapture
	size            int64
}

// archiveRuntime is the parsed configuration and the store built from it.
type archiveRuntime struct {
	settings ArchiveSettings
	store    ArchiveObjectStore // nil when the configuration is incomplete or the build failed
}

type ConversationArchiveService struct {
	repo          ConversationArchiveRepository
	factory       ArchiveObjectStoreFactory
	encryptor     SecretEncryptor
	encryptionKey bool
	dedupe        ArchiveDedupeStore

	cfgMu   sync.RWMutex
	runtime *archiveRuntime
	raw     string
	loaded  bool
	// cfgVersion counts installed configurations; a reload that started before a save is dropped.
	cfgVersion uint64
	enabled    atomic.Bool

	queues     []chan *archiveJob
	maxItems   int64
	maxBytes   int64
	queueLen   atomic.Int64
	queueBytes atomic.Int64
	dropped    atomic.Uint64
	failed     atomic.Uint64
	lastDrop   atomic.Int64

	statusMu      sync.Mutex
	uploadedDay   string
	uploadedToday int64
	lastUploadAt  *time.Time
	lastError     string
	lastErrorAt   *time.Time
	lifecycleOK   bool
	lifecycleAt   *time.Time
	lifecycleErr  string
	pendingPurges int

	purgedMu  sync.Mutex
	purged    map[int64]time.Time
	purgeKick chan struct{}

	ctx  context.Context
	stop context.CancelFunc
	// opCtx parents every storage / Redis / DB call of the workers; it is cancelled at the end of
	// the shutdown drain so Stop never waits on a slow upload.
	opCtx        context.Context
	opCancel     context.CancelFunc
	drainTimeout time.Duration
	wg           sync.WaitGroup
	startOnce    sync.Once
	stopOnce     sync.Once
	now          func() time.Time
	sleep        func(ctx context.Context, d time.Duration) bool
}

func NewConversationArchiveService(repo ConversationArchiveRepository, factory ArchiveObjectStoreFactory, encryptor SecretEncryptor, encryptionKeyConfigured bool, dedupe ArchiveDedupeStore) *ConversationArchiveService {
	ctx, cancel := context.WithCancel(context.Background())
	opCtx, opCancel := context.WithCancel(context.Background())
	s := &ConversationArchiveService{
		opCtx:         opCtx,
		opCancel:      opCancel,
		drainTimeout:  archiveDrainTimeout,
		purgeKick:     make(chan struct{}, 1),
		repo:          repo,
		factory:       factory,
		encryptor:     encryptor,
		encryptionKey: encryptionKeyConfigured,
		dedupe:        dedupe,
		runtime:       &archiveRuntime{},
		purged:        map[int64]time.Time{},
		ctx:           ctx,
		stop:          cancel,
		now:           time.Now,
		sleep:         archiveSleep,
		maxItems:      archiveQueueMaxItems,
		maxBytes:      archiveQueueMaxBytes,
	}
	s.queues = make([]chan *archiveJob, archiveWorkerCount)
	for i := range s.queues {
		s.queues[i] = make(chan *archiveJob, archiveQueueMaxItems/archiveWorkerCount)
	}
	return s
}

func archiveSleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// ProvideConversationArchiveService loads the configuration, starts the workers, the reload and
// the retention loops, and hooks into both gateway services' usage recording and account deletion.
func ProvideConversationArchiveService(
	repo ConversationArchiveRepository,
	factory ArchiveObjectStoreFactory,
	encryptor SecretEncryptor,
	cfg *config.Config,
	dedupe ArchiveDedupeStore,
	openai *OpenAIGatewayService,
	gateway *GatewayService,
	deletion *AccountDeletionService,
) *ConversationArchiveService {
	keyConfigured := cfg != nil && cfg.Totp.EncryptionKeyConfigured
	s := NewConversationArchiveService(repo, factory, encryptor, keyConfigured, dedupe)
	if openai != nil {
		openai.SetConversationArchiveRecorder(s)
	}
	if gateway != nil {
		gateway.SetConversationArchiveRecorder(s)
	}
	if deletion != nil {
		deletion.SetArchivePurger(s)
	}
	s.Start()
	return s
}

func (s *ConversationArchiveService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		s.reload(ctx)
		cancel()
		s.wg.Add(3 + len(s.queues))
		go s.runReloadLoop()
		go s.runRetentionLoop()
		go s.runPurgeLoop()
		for i := range s.queues {
			go s.runWorker(s.queues[i])
		}
	})
}

// Stop stops accepting jobs, gives the queued ones a short drain and stops the loops. Call it
// after the usage worker pool has drained.
func (s *ConversationArchiveService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.stop()
		hard := time.AfterFunc(s.drainTimeout, s.opCancel)
		s.wg.Wait()
		hard.Stop()
		s.opCancel()
		s.cfgMu.Lock()
		s.setEnabled(false)
		s.cfgMu.Unlock()
	})
}

// Enabled reports whether new requests are archived (atomic; refreshed on save and every 30s).
func (s *ConversationArchiveService) Enabled() bool {
	return s != nil && s.enabled.Load()
}

func (s *ConversationArchiveService) setEnabled(on bool) {
	s.enabled.Store(on)
	archiveCaptureOn.Store(on)
}

func (s *ConversationArchiveService) currentRuntime() *archiveRuntime {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.runtime
}

// --- configuration ---

func normalizeArchiveSettings(in *ArchiveSettings) {
	in.Endpoint = strings.TrimRight(strings.TrimSpace(in.Endpoint), "/")
	in.Region = strings.TrimSpace(in.Region)
	in.Bucket = strings.TrimSpace(in.Bucket)
	in.AccessKeyID = strings.TrimSpace(in.AccessKeyID)
	in.SecretAccessKey = strings.TrimSpace(in.SecretAccessKey)
	prefix := strings.TrimLeft(strings.TrimSpace(in.Prefix), "/")
	if prefix == "" {
		prefix = archiveDefaultPrefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	in.Prefix = prefix
	if in.RetentionDays == 0 {
		in.RetentionDays = archiveDefaultRetentionDays
	}
}

func (st *ArchiveSettings) complete() bool {
	return st.Bucket != "" && st.AccessKeyID != "" && st.SecretAccessKey != ""
}

func parseArchiveSettings(raw string) (ArchiveSettings, bool) {
	var st ArchiveSettings
	if strings.TrimSpace(raw) == "" {
		st.SSE = true
		normalizeArchiveSettings(&st)
		return st, false
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		st = ArchiveSettings{SSE: true}
		normalizeArchiveSettings(&st)
		return st, false
	}
	normalizeArchiveSettings(&st)
	return st, true
}

func (s *ConversationArchiveService) decryptSecret(stored string) string {
	if stored == "" || s.encryptor == nil {
		return stored
	}
	plain, err := s.encryptor.Decrypt(stored)
	if err != nil {
		logger.LegacyPrintf("service.conversation_archive", "[Archive] secret decrypt failed; treating the stored value as plaintext: %v", err)
		return stored
	}
	return plain
}

func (s *ConversationArchiveService) buildStore(ctx context.Context, st ArchiveSettings) (ArchiveObjectStore, error) {
	if s.factory == nil {
		return nil, errors.New("archive store factory unavailable")
	}
	return s.factory(ctx, ArchiveStoreConfig{
		Endpoint:        st.Endpoint,
		Region:          st.Region,
		Bucket:          st.Bucket,
		AccessKeyID:     st.AccessKeyID,
		SecretAccessKey: s.decryptSecret(st.SecretAccessKey),
		ForcePathStyle:  st.ForcePathStyle,
	})
}

// buildRuntime parses a raw settings value and builds its store (no lock held).
func (s *ConversationArchiveService) buildRuntime(ctx context.Context, raw string) *archiveRuntime {
	st, _ := parseArchiveSettings(raw)
	rt := &archiveRuntime{settings: st}
	if st.complete() {
		store, err := s.buildStore(ctx, st)
		if err != nil {
			s.recordError(fmt.Sprintf("build storage client: %v", err))
		} else {
			rt.store = store
		}
	}
	return rt
}

// installLocked switches to a runtime (cfgMu held).
func (s *ConversationArchiveService) installLocked(rt *archiveRuntime, raw string) {
	s.runtime = rt
	s.raw = raw
	s.loaded = true
	s.cfgVersion++
	s.setEnabled(rt.settings.Enabled && rt.store != nil && s.ctx.Err() == nil)
}

// applyNow installs a configuration just saved here.
func (s *ConversationArchiveService) applyNow(ctx context.Context, raw string) *archiveRuntime {
	rt := s.buildRuntime(ctx, raw)
	s.cfgMu.Lock()
	s.installLocked(rt, raw)
	s.cfgMu.Unlock()
	return rt
}

// reload installs the stored configuration unless it is unchanged or a save happened meanwhile.
func (s *ConversationArchiveService) reload(ctx context.Context) {
	s.cfgMu.RLock()
	base := s.cfgVersion
	s.cfgMu.RUnlock()
	raw, err := s.repo.GetConfigRaw(ctx)
	if err != nil {
		logger.LegacyPrintf("service.conversation_archive", "[Archive] reload config failed: %v", err)
		return
	}
	s.cfgMu.RLock()
	skip := s.cfgVersion != base || (s.loaded && s.raw == raw)
	s.cfgMu.RUnlock()
	if skip {
		return
	}
	rt := s.buildRuntime(ctx, raw)
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	if s.cfgVersion != base {
		return
	}
	s.installLocked(rt, raw)
}

func (s *ConversationArchiveService) runReloadLoop() {
	defer s.wg.Done()
	t := time.NewTicker(archiveReloadInterval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			archiveSafeStep("reload", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				s.reload(ctx)
			})
		}
	}
}

// GetConfig returns the configuration (without the secret) and the writer status.
func (s *ConversationArchiveService) GetConfig(ctx context.Context) (*ArchiveConfigView, error) {
	raw, err := s.repo.GetConfigRaw(ctx)
	if err != nil {
		return nil, err
	}
	st, _ := parseArchiveSettings(raw)
	return s.view(st), nil
}

func (s *ConversationArchiveService) view(st ArchiveSettings) *ArchiveConfigView {
	return &ArchiveConfigView{
		Enabled:          st.Enabled,
		Endpoint:         st.Endpoint,
		Region:           st.Region,
		Bucket:           st.Bucket,
		Prefix:           st.Prefix,
		AccessKeyID:      st.AccessKeyID,
		SecretConfigured: st.SecretAccessKey != "",
		ForcePathStyle:   st.ForcePathStyle,
		RetentionDays:    st.RetentionDays,
		SSE:              st.SSE,
		Status:           s.Status(),
	}
}

// settingsFromInput merges the admin input over the stored settings (secret kept when empty).
func (s *ConversationArchiveService) settingsFromInput(in ArchiveConfigInput, old ArchiveSettings) (ArchiveSettings, bool, error) {
	st := ArchiveSettings{
		Enabled:         in.Enabled,
		Endpoint:        in.Endpoint,
		Region:          in.Region,
		Bucket:          in.Bucket,
		Prefix:          in.Prefix,
		AccessKeyID:     in.AccessKeyID,
		SecretAccessKey: in.SecretAccessKey,
		ForcePathStyle:  in.ForcePathStyle,
		RetentionDays:   in.RetentionDays,
		SSE:             in.SSE == nil || *in.SSE,
	}
	normalizeArchiveSettings(&st)
	if st.RetentionDays < archiveMinRetentionDays || st.RetentionDays > archiveMaxRetentionDays {
		return st, false, ErrArchiveRetention
	}
	secretChanged := st.SecretAccessKey != ""
	if !secretChanged {
		st.SecretAccessKey = old.SecretAccessKey
	}
	return st, secretChanged, nil
}

// UpdateConfig saves the configuration (audited as archive_config with the changed non-secret
// fields) and applies it at once.
func (s *ConversationArchiveService) UpdateConfig(ctx context.Context, adminID int64, in ArchiveConfigInput, ip, ua string) (*ArchiveConfigView, error) {
	oldRaw, err := s.repo.GetConfigRaw(ctx)
	if err != nil {
		return nil, err
	}
	old, _ := parseArchiveSettings(oldRaw)
	st, secretChanged, err := s.settingsFromInput(in, old)
	if err != nil {
		return nil, err
	}
	st.PreviousPrefixes = archivePreviousPrefixes(old, st.Prefix, strings.TrimSpace(oldRaw) != "")
	if secretChanged {
		if !s.encryptionKey || s.encryptor == nil {
			return nil, ErrSecretEncryptionKeyNotConfigured
		}
		encrypted, err := s.encryptor.Encrypt(st.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt secret: %w", err)
		}
		st.SecretAccessKey = encrypted
	}
	if st.Enabled && !st.complete() {
		return nil, ErrArchiveIncomplete
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	detail := archiveConfigDiff(old, st, secretChanged)
	audit := ContentLogAccess{AdminID: adminID, Action: ContentLogActionArchiveConfig, Detail: detail, IPAddress: ip, UserAgent: ua}
	if err := s.repo.SaveConfig(ctx, string(raw), audit); err != nil {
		return nil, err
	}
	rt := s.applyNow(ctx, string(raw))
	if st.Enabled && rt.store != nil {
		// Safety net for retention; a failure doesn't undo the save (shown in status.lifecycle_error).
		_ = s.applyLifecycle(ctx, rt) // outcome is in Status().LifecycleApplied / LifecycleError
	}
	return s.view(st), nil
}

// archivePreviousPrefixes keeps every prefix the archive wrote under, so account deletion can
// sweep them all.
func archivePreviousPrefixes(old ArchiveSettings, newPrefix string, oldSaved bool) []string {
	out := append([]string(nil), old.PreviousPrefixes...)
	if oldSaved && old.Prefix != "" && old.Prefix != newPrefix {
		out = append(out, old.Prefix)
	}
	seen := map[string]bool{newPrefix: true}
	uniq := out[:0]
	for _, p := range out {
		if p != "" && !seen[p] {
			seen[p] = true
			uniq = append(uniq, p)
		}
	}
	if len(uniq) > archiveMaxPreviousPrefixes {
		uniq = uniq[len(uniq)-archiveMaxPreviousPrefixes:]
	}
	if len(uniq) == 0 {
		return nil
	}
	return uniq
}

// applyLifecycle applies the bucket rule and records the outcome in the status.
func (s *ConversationArchiveService) applyLifecycle(ctx context.Context, rt *archiveRuntime) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := rt.store.ApplyLifecycle(ctx, archiveLifecycleRuleID, rt.settings.Prefix, rt.settings.RetentionDays+1)
	now := s.now()
	s.statusMu.Lock()
	if err != nil {
		s.lifecycleOK = false
		s.lifecycleErr = truncateBytesUTF8(err.Error(), 1000)
	} else {
		s.lifecycleOK = true
		s.lifecycleAt = &now
		s.lifecycleErr = ""
	}
	s.statusMu.Unlock()
	return err
}

func archiveConfigDiff(old, st ArchiveSettings, secretChanged bool) string {
	changed := map[string]any{}
	if old.Enabled != st.Enabled {
		changed["enabled"] = st.Enabled
	}
	if old.Endpoint != st.Endpoint {
		changed["endpoint"] = st.Endpoint
	}
	if old.Region != st.Region {
		changed["region"] = st.Region
	}
	if old.Bucket != st.Bucket {
		changed["bucket"] = st.Bucket
	}
	if old.Prefix != st.Prefix {
		changed["prefix"] = st.Prefix
	}
	if old.AccessKeyID != st.AccessKeyID {
		changed["access_key_id"] = st.AccessKeyID
	}
	if old.ForcePathStyle != st.ForcePathStyle {
		changed["force_path_style"] = st.ForcePathStyle
	}
	if old.RetentionDays != st.RetentionDays {
		changed["retention_days"] = st.RetentionDays
	}
	if old.SSE != st.SSE {
		changed["sse"] = st.SSE
	}
	if secretChanged {
		changed["secret_access_key"] = "changed"
	}
	raw, _ := json.Marshal(changed)
	return string(raw)
}

// TestConnection writes, reads back and deletes {prefix}_healthcheck/{ts}.txt. With a nil input
// (or no bucket in it) the saved configuration is tested; an empty secret uses the saved one.
func (s *ConversationArchiveService) TestConnection(ctx context.Context, in *ArchiveConfigInput) error {
	raw, err := s.repo.GetConfigRaw(ctx)
	if err != nil {
		return err
	}
	old, _ := parseArchiveSettings(raw)
	st := old
	if in != nil && strings.TrimSpace(in.Bucket) != "" {
		probe := *in
		probe.Enabled = false
		if probe.RetentionDays == 0 {
			probe.RetentionDays = archiveDefaultRetentionDays
		}
		st, _, err = s.settingsFromInput(probe, old)
		if err != nil {
			return err
		}
		if strings.TrimSpace(in.SecretAccessKey) != "" {
			// A secret typed into the form is plain text; mark it so buildStore doesn't decrypt.
			st.SecretAccessKey = strings.TrimSpace(in.SecretAccessKey)
			return s.testWithStore(ctx, st, true)
		}
	}
	if !st.complete() {
		return ErrArchiveIncomplete
	}
	return s.testWithStore(ctx, st, false)
}

func (s *ConversationArchiveService) testWithStore(ctx context.Context, st ArchiveSettings, plainSecret bool) error {
	var store ArchiveObjectStore
	var err error
	if plainSecret {
		if s.factory == nil {
			return ErrArchiveNotConfigured
		}
		store, err = s.factory(ctx, ArchiveStoreConfig{Endpoint: st.Endpoint, Region: st.Region, Bucket: st.Bucket, AccessKeyID: st.AccessKeyID, SecretAccessKey: st.SecretAccessKey, ForcePathStyle: st.ForcePathStyle})
	} else {
		store, err = s.buildStore(ctx, st)
	}
	if err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "创建存储客户端失败", "Failed to build the storage client", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	key := fmt.Sprintf("%s_healthcheck/%d.txt", st.Prefix, s.now().UnixNano())
	payload := []byte("sub2api conversation archive healthcheck " + s.now().UTC().Format(time.RFC3339))
	if err := store.PutObject(ctx, key, payload, "text/plain; charset=utf-8", "", st.SSE); err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "写入测试对象失败", "PutObject failed", err)
	}
	rc, _, err := store.GetObject(ctx, key)
	if err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "读取测试对象失败", "GetObject failed", err)
	}
	got, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	_ = rc.Close()
	if err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "读取测试对象失败", "GetObject failed", err)
	}
	if !bytes.Equal(got, payload) {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "读回的测试对象内容不一致", "Read-back content mismatch", fmt.Errorf("%d bytes", len(got)))
	}
	if err := store.DeleteObjects(ctx, []string{key}); err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_TEST_FAILED", "删除测试对象失败", "DeleteObjects failed", err)
	}
	return nil
}

// ApplyLifecycle adds the bucket lifecycle rule expiring {prefix} after retention_days + 1. The
// audit row (archive_config) is written first; without it nothing is changed.
func (s *ConversationArchiveService) ApplyLifecycle(ctx context.Context, adminID int64, ip, ua string) error {
	rt := s.currentRuntime()
	if rt == nil || rt.store == nil {
		return ErrArchiveNotConfigured
	}
	days := rt.settings.RetentionDays + 1
	detail, _ := json.Marshal(map[string]any{"lifecycle": "apply", "prefix": rt.settings.Prefix, "expire_days": days})
	if err := s.repo.InsertAccess(ctx, ContentLogAccess{AdminID: adminID, Action: ContentLogActionArchiveConfig, Detail: string(detail), IPAddress: ip, UserAgent: ua}); err != nil {
		return err
	}
	if err := s.applyLifecycle(ctx, rt); err != nil {
		return archiveStorageError("CONVERSATION_ARCHIVE_LIFECYCLE_FAILED", "应用生命周期规则失败", "PutBucketLifecycleConfiguration failed", err)
	}
	return nil
}

// Status is the writer's live state.
func (s *ConversationArchiveService) Status() ArchiveStatus {
	st := ArchiveStatus{
		Active:       s.Enabled(),
		QueueLen:     s.queueLen.Load(),
		QueueBytes:   s.queueBytes.Load(),
		DroppedTotal: s.dropped.Load(),
		FailedTotal:  s.failed.Load(),
	}
	s.statusMu.Lock()
	if s.uploadedDay == s.today() {
		st.UploadedToday = s.uploadedToday
	}
	if s.lastUploadAt != nil {
		t := *s.lastUploadAt
		st.LastUploadAt = &t
	}
	st.LastError = s.lastError
	if s.lastErrorAt != nil {
		t := *s.lastErrorAt
		st.LastErrorAt = &t
	}
	st.LifecycleApplied = s.lifecycleOK
	if s.lifecycleAt != nil {
		t := *s.lifecycleAt
		st.LifecycleAppliedAt = &t
	}
	st.LifecycleError = s.lifecycleErr
	st.PendingPurges = s.pendingPurges
	s.statusMu.Unlock()
	return st
}

func (s *ConversationArchiveService) today() string {
	return s.now().In(timezone.Location()).Format("2006-01-02")
}

func (s *ConversationArchiveService) recordError(msg string) {
	now := s.now()
	s.statusMu.Lock()
	s.lastError = truncateBytesUTF8(msg, 1000)
	s.lastErrorAt = &now
	s.statusMu.Unlock()
	logger.LegacyPrintf("service.conversation_archive", "[Archive] %s", msg)
}

func (s *ConversationArchiveService) recordUpload() {
	now := s.now()
	day := s.today()
	s.statusMu.Lock()
	if s.uploadedDay != day {
		s.uploadedDay = day
		s.uploadedToday = 0
	}
	s.uploadedToday++
	s.lastUploadAt = &now
	s.statusMu.Unlock()
}

// --- queue ---

// SetConversationArchiveRecorder hooks the archive into OpenAI usage recording (wired at startup).
func (s *OpenAIGatewayService) SetConversationArchiveRecorder(r ConversationArchiveRecorder) {
	if s != nil {
		s.archive = r
	}
}

// SetConversationArchiveRecorder hooks the archive into gateway usage recording (wired at startup).
func (s *GatewayService) SetConversationArchiveRecorder(r ConversationArchiveRecorder) {
	if s != nil {
		s.archive = r
	}
}

// recordConversationArchive hands a captured request to the archive; a no-op unless the request
// was captured for it.
func recordConversationArchive(r ConversationArchiveRecorder, usageLog *UsageLog, capture *ContentCapture) {
	if r == nil || usageLog == nil || capture == nil || capture.Archive == nil {
		return
	}
	defer func() { _ = recover() }()
	r.EnqueueArchive(usageLog, capture.Archive)
}

// EnqueueArchive queues one request for archiving. Never blocks and never panics; drops (and
// counts) when the queue is over its count or byte budget.
func (s *ConversationArchiveService) EnqueueArchive(usageLog *UsageLog, capture *ArchiveCapture) {
	if s == nil || usageLog == nil || capture == nil || !s.Enabled() {
		return
	}
	defer func() { _ = recover() }()
	if s.ctx.Err() != nil {
		return
	}
	job := &archiveJob{
		userID:          usageLog.UserID,
		apiKeyID:        usageLog.APIKeyID,
		requestID:       usageLog.RequestID,
		sessionID:       derefString(usageLog.SessionID),
		model:           strings.TrimSpace(usageLog.RequestedModel),
		upstreamModel:   derefString(usageLog.UpstreamModel),
		inboundEndpoint: derefString(usageLog.InboundEndpoint),
		userAgent:       derefString(usageLog.UserAgent),
		ipAddress:       derefString(usageLog.IPAddress),
		createdAt:       usageLog.CreatedAt,
		inputTokens:     usageLog.InputTokens,
		outputTokens:    usageLog.OutputTokens,
		cacheReadTokens: usageLog.CacheReadTokens,
		costUSD:         usageLog.ActualCost,
		capture:         capture,
		size:            capture.SizeBytes() + 1024,
	}
	if job.model == "" {
		job.model = usageLog.Model
	}
	if usageLog.GroupID != nil {
		id := *usageLog.GroupID
		job.groupID = &id
	}
	if job.createdAt.IsZero() {
		job.createdAt = s.now()
	}
	if s.queueLen.Add(1) > s.maxItems {
		s.queueLen.Add(-1)
		s.drop()
		return
	}
	if s.queueBytes.Add(job.size) > s.maxBytes {
		s.queueBytes.Add(-job.size)
		s.queueLen.Add(-1)
		s.drop()
		return
	}
	select {
	case s.queues[s.shard(job)] <- job:
	default:
		s.release(job)
		s.drop()
	}
}

// shard keeps one session (or, without a session id, one API key) on one worker, so its turns
// are deduped in order.
func (s *ConversationArchiveService) shard(job *archiveJob) int {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%d|", job.userID)
	if job.sessionID != "" {
		_, _ = h.Write([]byte(job.sessionID))
	} else {
		_, _ = fmt.Fprintf(h, "k%d", job.apiKeyID)
	}
	return int(h.Sum32() % uint32(len(s.queues)))
}

func (s *ConversationArchiveService) release(job *archiveJob) {
	s.queueLen.Add(-1)
	s.queueBytes.Add(-job.size)
}

func (s *ConversationArchiveService) drop() {
	n := s.dropped.Add(1)
	now := s.now().UnixNano()
	last := s.lastDrop.Load()
	if now-last >= int64(archiveDropLogInterval) && s.lastDrop.CompareAndSwap(last, now) {
		logger.LegacyPrintf("service.conversation_archive", "[Archive] queue full, dropped %d requests so far", n)
	}
}

func archiveSafeStep(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.LegacyPrintf("service.conversation_archive", "[Archive] %s panicked: %v", name, r)
		}
	}()
	fn()
}

func (s *ConversationArchiveService) runWorker(queue chan *archiveJob) {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			// Drain until opCtx is cancelled (archiveDrainTimeout after Stop), then drop the rest.
			for {
				select {
				case job := <-queue:
					if s.opCtx.Err() == nil {
						archiveSafeStep("process", func() { s.process(job) })
					} else {
						s.drop()
					}
					s.release(job)
				default:
					return
				}
			}
		case job := <-queue:
			archiveSafeStep("process", func() { s.process(job) })
			s.release(job)
		}
	}
}

// putWithRetry uploads with up to 3 attempts and backoff; no retries once stopping.
func (s *ConversationArchiveService) putWithRetry(store ArchiveObjectStore, key string, body []byte, contentType, contentEncoding string, sse bool) error {
	var err error
	for attempt := 0; attempt < archiveUploadAttempts; attempt++ {
		if attempt > 0 {
			if !s.sleep(s.ctx, time.Duration(attempt*attempt)*time.Second) {
				break
			}
		}
		ctx, cancel := context.WithTimeout(s.opCtx, archiveUploadTimeout)
		err = store.PutObject(ctx, key, body, contentType, contentEncoding, sse)
		cancel()
		if err == nil {
			return nil
		}
	}
	return err
}

func (s *ConversationArchiveService) isPurged(userID int64) bool {
	s.purgedMu.Lock()
	defer s.purgedMu.Unlock()
	at, ok := s.purged[userID]
	if ok && s.now().Sub(at) > archivePurgedRemember {
		delete(s.purged, userID)
		return false
	}
	return ok
}

// process archives one request: extract, dedupe, upload images and the turn, then index it.
func (s *ConversationArchiveService) process(job *archiveJob) {
	if job == nil || job.capture == nil || s.isPurged(job.userID) {
		return
	}
	rt := s.currentRuntime()
	if rt == nil || rt.store == nil {
		return
	}
	prefix, sse := rt.settings.Prefix, rt.settings.SSE
	capture := job.capture

	parts := extractArchiveRequest(capture.Protocol, capture.RequestBody, capture.Prompt)
	sessionKey := archiveSessionKey(job.sessionID, job.apiKeyID, parts.Items, job.requestID)
	hashes := make([]string, len(parts.Items))
	for i, item := range parts.Items {
		hashes[i] = archiveItemHash(item)
	}
	instrHash := ""
	if parts.HasInstructions {
		instrHash = archiveSHA256Hex(parts.Instructions)[:24]
	}
	// Thread: the first input item (canonical hash) within the session; previous_response_id
	// chains carry only new items, so they share one "chain" state for the instructions.
	thread := "chain"
	if parts.PreviousResponseID == "" {
		thread = ""
		if len(hashes) > 0 {
			thread = hashes[0][:12]
		}
	}
	dedupeKey := archiveDedupeKey(job.userID, sessionKey, thread)
	seenKey := archiveSeenKey(job.userID, sessionKey)
	var prev *ArchiveDedupeState
	var loadErr error
	newThread := false
	if capture.Protocol == ArchiveProtocolImages {
		prev, loadErr = nil, nil
	} else if s.dedupe == nil {
		loadErr = errors.New("redis unavailable")
	} else {
		ctx, cancel := context.WithTimeout(s.opCtx, 3*time.Second)
		prev, loadErr = s.dedupe.Load(ctx, dedupeKey)
		if loadErr == nil && prev == nil && parts.PreviousResponseID == "" {
			seen, err := s.dedupe.Load(ctx, seenKey)
			newThread = err == nil && seen != nil
		}
		cancel()
	}
	dd := computeArchiveDedupe(prev, loadErr, parts, hashes, instrHash)
	if newThread {
		// Another context (sub-agent, side call, compaction) in a session that already has turns:
		// its whole input is archived and marked as a fresh context.
		dd.Reset = true
	}
	newItems, requestCut := capArchiveItems(dd.NewItems, archiveRequestItemsMax)

	at := job.createdAt.In(timezone.Location())
	record := &archiveRecord{
		V:               archiveRecordVersion,
		RequestID:       job.requestID,
		UserID:          job.userID,
		APIKeyID:        job.apiKeyID,
		GroupID:         job.groupID,
		SessionKey:      sessionKey,
		Thread:          thread,
		CreatedAt:       job.createdAt.UTC().Format(time.RFC3339Nano),
		Model:           job.model,
		UpstreamModel:   job.upstreamModel,
		InboundEndpoint: job.inboundEndpoint,
		Protocol:        capture.Protocol,
		UserAgent:       job.userAgent,
		IPAddress:       job.ipAddress,
		Usage:           archiveRecordUsage{InputTokens: job.inputTokens, OutputTokens: job.outputTokens, CacheReadTokens: job.cacheReadTokens, CostUSD: job.costUSD},
		Reset:           dd.Reset,
		PrefixItems:     dd.PrefixItems,
		NewItems:        newItems,
		RequestParams:   parts.Params,
		Response:        capture.Response,
		Images:          []archiveRecordImage{},
		Truncated:       archiveRecordTruncated{Request: requestCut, Response: capture.ResponseTruncated},
	}
	if record.NewItems == nil {
		record.NewItems = []json.RawMessage{}
	}
	if dd.IncludeInstructions {
		instr := parts.Instructions
		record.Instructions = &instr
	}

	// Images first, so the turn object can point at them. uploaded is cleaned up if the turn
	// can't be stored or indexed.
	var images []ArchiveImage
	var uploaded []string
	cleanup := func() {
		if len(uploaded) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(s.opCtx, archiveUploadTimeout)
		defer cancel()
		if err := rt.store.DeleteObjects(ctx, uploaded); err != nil {
			s.recordError(fmt.Sprintf("cleanup %d objects of %s: %v", len(uploaded), job.requestID, err))
		}
	}
	for i, img := range capture.Images {
		data, mime, ext, err := decodeArchiveImage(img.B64)
		if err != nil {
			s.failed.Add(1)
			s.recordError(fmt.Sprintf("image %d of %s: %v", i, job.requestID, err))
			record.Truncated.Response = true
			continue
		}
		key := archiveImageKey(prefix, job.userID, at, job.requestID, i, ext)
		if err := s.putWithRetry(rt.store, key, data, mime, "", sse); err != nil {
			s.failed.Add(1)
			s.recordError(fmt.Sprintf("upload image %s: %v", key, err))
			record.Truncated.Response = true
			continue
		}
		uploaded = append(uploaded, key)
		images = append(images, ArchiveImage{UserID: job.userID, ObjectKey: key, Mime: mime, Bytes: len(data), CreatedAt: job.createdAt})
		record.Images = append(record.Images, archiveRecordImage{Key: key, Mime: mime, Bytes: len(data), Source: img.Source})
	}
	for _, u := range capture.ImageURLs {
		record.Images = append(record.Images, archiveRecordImage{URL: u, Source: archiveImageSourceResponse})
	}

	body, err := gzipJSON(record)
	if err != nil {
		s.failed.Add(1)
		s.recordError(fmt.Sprintf("encode %s: %v", job.requestID, err))
		cleanup()
		return
	}
	turnKey := archiveTurnKey(prefix, job.userID, at, sessionKey, job.requestID)
	if err := s.putWithRetry(rt.store, turnKey, body, "application/json", "gzip", sse); err != nil {
		s.failed.Add(1)
		s.recordError(fmt.Sprintf("upload %s: %v", turnKey, err))
		cleanup()
		return
	}
	uploaded = append(uploaded, turnKey)

	preview := ""
	if capture.Protocol == ArchiveProtocolImages {
		preview = archiveHeadRunes(strings.TrimSpace(firstNonEmptyString(capture.Prompt, gjsonPromptOf(parts.Items))), archivePreviewMaxRunes)
	} else {
		preview = archiveLatestUserText(capture.Protocol, newItems)
	}
	turn := &ArchiveTurn{
		UserID:          job.userID,
		GroupID:         job.groupID,
		SessionKey:      sessionKey,
		RequestID:       truncateBytesUTF8(job.requestID, 256),
		CreatedAt:       job.createdAt,
		Model:           truncateBytesUTF8(job.model, 200),
		InboundEndpoint: truncateBytesUTF8(job.inboundEndpoint, 256),
		UserAgent:       truncateBytesUTF8(job.userAgent, 512),
		IPAddress:       truncateBytesUTF8(job.ipAddress, 64),
		InputTokens:     job.inputTokens,
		OutputTokens:    job.outputTokens,
		CacheReadTokens: job.cacheReadTokens,
		ObjectKey:       turnKey,
		Bytes:           len(body),
		NewItemCount:    len(newItems),
		Reset:           dd.Reset,
		ImageCount:      len(images),
		Preview:         preview,
	}
	if job.apiKeyID > 0 {
		id := job.apiKeyID
		turn.APIKeyID = &id
	}
	ctx, cancel := context.WithTimeout(s.opCtx, 10*time.Second)
	_, err = s.repo.InsertTurn(ctx, turn, images)
	cancel()
	if err != nil {
		s.failed.Add(1)
		s.recordError(fmt.Sprintf("index %s: %v", turnKey, err))
		cleanup()
		return
	}
	if s.dedupe != nil && capture.Protocol != ArchiveProtocolImages {
		ctx, cancel := context.WithTimeout(s.opCtx, 3*time.Second)
		if err := s.dedupe.Save(ctx, dedupeKey, dd.Next, archiveDedupeTTL); err != nil && loadErr == nil {
			logger.LegacyPrintf("service.conversation_archive", "[Archive] save dedupe state failed: %v", err)
		}
		if parts.PreviousResponseID == "" {
			_ = s.dedupe.Save(ctx, seenKey, &ArchiveDedupeState{}, archiveDedupeTTL)
		}
		cancel()
	}
	s.recordUpload()
}

func gjsonPromptOf(items []json.RawMessage) string {
	if len(items) == 0 {
		return ""
	}
	var v struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(items[0], &v)
	return v.Content
}

// --- retention and purge ---

func (s *ConversationArchiveService) runRetentionLoop() {
	defer s.wg.Done()
	timer := time.NewTimer(archiveRetentionStartDelay)
	defer timer.Stop()
	select {
	case <-s.ctx.Done():
		return
	case <-timer.C:
	}
	archiveSafeStep("retention", s.RunRetentionOnce)
	ticker := time.NewTicker(archiveRetentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			archiveSafeStep("retention", s.RunRetentionOnce)
		}
	}
}

// RunRetentionOnce deletes turns older than retention_days, their objects first (batched
// DeleteObjects), then their index rows. Stops at the first storage error (retried next hour).
func (s *ConversationArchiveService) RunRetentionOnce() {
	rt := s.currentRuntime()
	if rt == nil || rt.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Minute)
	defer cancel()
	cutoff := s.now().AddDate(0, 0, -rt.settings.RetentionDays)
	for ctx.Err() == nil {
		turns, err := s.repo.ListExpired(ctx, cutoff, archiveRetentionBatch)
		if err != nil {
			s.recordError(fmt.Sprintf("retention list: %v", err))
			return
		}
		if len(turns) == 0 {
			return
		}
		keys := make([]string, 0, len(turns)*2)
		ids := make([]int64, 0, len(turns))
		for _, t := range turns {
			ids = append(ids, t.ID)
			keys = append(keys, t.ObjectKey)
			keys = append(keys, t.ImageKeys...)
		}
		for start := 0; start < len(keys); start += archiveDeleteBatch {
			end := min(start+archiveDeleteBatch, len(keys))
			if err := rt.store.DeleteObjects(ctx, keys[start:end]); err != nil {
				s.recordError(fmt.Sprintf("retention delete objects: %v", err))
				return
			}
		}
		if _, err := s.repo.DeleteTurns(ctx, ids); err != nil {
			s.recordError(fmt.Sprintf("retention delete rows: %v", err))
			return
		}
		if len(turns) < archiveRetentionBatch {
			return
		}
	}
}

// QueueUserPurge is called by self-service account deletion before the account is deleted: it
// records the prefixes to sweep (the current one and every previous one) in
// conversation_archive_purges, which the purge job works through until done, also when the
// storage isn't configured yet. Queued archive jobs of the user are skipped from now on.
func (s *ConversationArchiveService) QueueUserPurge(ctx context.Context, userID int64) error {
	if s == nil || userID <= 0 {
		return nil
	}
	raw, err := s.repo.GetConfigRaw(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return nil // never configured: nothing was archived
	}
	st, _ := parseArchiveSettings(raw)
	prefixes := append([]string{st.Prefix}, st.PreviousPrefixes...)
	if err := s.repo.QueuePurge(ctx, userID, prefixes); err != nil {
		return err
	}
	s.purgedMu.Lock()
	s.purged[userID] = s.now()
	s.purgedMu.Unlock()
	return nil
}

// RunPendingPurges asks the purge job to run now (after the account was deleted).
func (s *ConversationArchiveService) RunPendingPurges() {
	if s == nil {
		return
	}
	select {
	case s.purgeKick <- struct{}{}:
	default:
	}
}

func (s *ConversationArchiveService) runPurgeLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(archivePurgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		case <-s.purgeKick:
		}
		archiveSafeStep("purge", s.RunPurgesOnce)
	}
}

// RunPurgesOnce sweeps {prefix}conv/{uid}/ and {prefix}img/{uid}/ for every pending purge of a
// deleted account; failures are kept for the next run. A row whose account was never deleted
// (the deletion failed afterwards) is dropped after a day.
func (s *ConversationArchiveService) RunPurgesOnce() {
	ctx, cancel := context.WithTimeout(s.opCtx, 30*time.Minute)
	defer cancel()
	purges, err := s.repo.ListPurges(ctx, archivePurgeBatch)
	if err != nil {
		s.recordError(fmt.Sprintf("purge list: %v", err))
		return
	}
	s.statusMu.Lock()
	s.pendingPurges = len(purges)
	s.statusMu.Unlock()
	rt := s.currentRuntime()
	for _, p := range purges {
		if ctx.Err() != nil {
			return
		}
		if !p.UserDeleted {
			if s.now().Sub(p.CreatedAt) > archivePurgeStaleAfter {
				_ = s.repo.FinishPurge(ctx, p.UserID)
			}
			continue
		}
		if rt == nil || rt.store == nil {
			return // retried once the storage is configured
		}
		var failure error
		for _, prefix := range p.Prefixes {
			for _, kind := range []string{"conv", "img"} {
				target := fmt.Sprintf("%s%s/%d/", prefix, kind, p.UserID)
				if _, err := rt.store.DeletePrefix(ctx, target); err != nil && failure == nil {
					failure = fmt.Errorf("%s: %w", target, err)
				}
			}
		}
		if failure != nil {
			s.recordError(fmt.Sprintf("purge user %d: %v", p.UserID, failure))
			_ = s.repo.FailPurge(ctx, p.UserID, failure.Error())
			continue
		}
		if err := s.repo.FinishPurge(ctx, p.UserID); err != nil {
			s.recordError(fmt.Sprintf("purge user %d finish: %v", p.UserID, err))
		}
	}
}

// --- admin browse ---

func (s *ConversationArchiveService) audit(ctx context.Context, adminID, targetUserID int64, detail any, ip, ua string) error {
	raw, _ := json.Marshal(detail)
	return s.repo.InsertAccess(ctx, ContentLogAccess{AdminID: adminID, Action: ContentLogActionArchiveView, TargetUserID: targetUserID, Detail: string(raw), IPAddress: ip, UserAgent: ua})
}

// ListSessions returns sessions by last activity (audited as archive_view).
func (s *ConversationArchiveService) ListSessions(ctx context.Context, adminID int64, f ArchiveSessionFilter, ip, ua string) (*ArchiveSessionPage, error) {
	f.Q = strings.TrimSpace(f.Q)
	if f.Limit <= 0 {
		f.Limit = archiveSessionsDefault
	}
	if f.Limit > archiveSessionsMax {
		f.Limit = archiveSessionsMax
	}
	if f.UserID < 0 {
		f.UserID = 0
	}
	unresolved := false
	if f.UserID == 0 && f.Q != "" {
		id, err := s.repo.ResolveUserID(ctx, f.Q)
		if err != nil {
			return nil, err
		}
		f.UserID = id
		unresolved = id == 0
	}
	if f.UserID == 0 && f.From == nil {
		from := s.now().AddDate(0, 0, -archiveDefaultBrowseDays)
		f.From = &from
	}
	if err := s.audit(ctx, adminID, f.UserID, map[string]any{"op": "sessions", "filter": f}, ip, ua); err != nil {
		return nil, err
	}
	page := &ArchiveSessionPage{Items: []ArchiveSession{}}
	if unresolved {
		return page, nil
	}
	items, err := s.repo.ListSessions(ctx, f)
	if err != nil {
		return nil, err
	}
	if items != nil {
		page.Items = items
	}
	for i := range page.Items {
		if page.Items[i].Models == nil {
			page.Items[i].Models = []string{}
		}
		if page.Items[i].Endpoints == nil {
			page.Items[i].Endpoints = []string{}
		}
	}
	if len(page.Items) == f.Limit {
		last := page.Items[len(page.Items)-1].LastAt
		page.NextBefore = &last
	}
	return page, nil
}

// ListTurns returns a session's turns newest first with their objects decoded (audited).
func (s *ConversationArchiveService) ListTurns(ctx context.Context, adminID, userID int64, sessionKey string, beforeID int64, limit int, ip, ua string) (*ArchiveTurnPage, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if userID <= 0 || sessionKey == "" {
		return nil, ErrArchiveSessionParams
	}
	if limit <= 0 {
		limit = archiveTurnsDefault
	}
	if limit > archiveTurnsMax {
		limit = archiveTurnsMax
	}
	if beforeID < 0 {
		beforeID = 0
	}
	if err := s.audit(ctx, adminID, userID, map[string]any{"op": "turns", "session_key": sessionKey, "before_id": beforeID, "limit": limit}, ip, ua); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListTurns(ctx, userID, sessionKey, beforeID, limit)
	if err != nil {
		return nil, err
	}
	// Bound what one page reads back from the bucket (compressed size from the index).
	full := len(rows) == limit
	budget := 0
	for i, row := range rows {
		if i > 0 && budget+row.Bytes > archiveTurnsPageBytes {
			rows, full = rows[:i], true
			break
		}
		budget += row.Bytes
	}
	items, err := s.loadTurns(ctx, rows)
	if err != nil {
		return nil, err
	}
	page := &ArchiveTurnPage{Items: items}
	if full && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		page.NextBeforeID = &last
	}
	return page, nil
}

// GetTurnByRequest returns one turn by its usage request id (audited).
func (s *ConversationArchiveService) GetTurnByRequest(ctx context.Context, adminID int64, requestID, ip, ua string) (*ArchiveTurnView, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, ErrArchiveRequestID
	}
	row, err := s.repo.GetTurnByRequestID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		if err := s.audit(ctx, adminID, 0, map[string]any{"op": "turn_by_request", "request_id": requestID}, ip, ua); err != nil {
			return nil, err
		}
		return nil, ErrArchiveTurnNotFound
	}
	if err := s.audit(ctx, adminID, row.UserID, map[string]any{"op": "turn_by_request", "request_id": requestID}, ip, ua); err != nil {
		return nil, err
	}
	items, err := s.loadTurns(ctx, []ArchiveTurn{*row})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *ConversationArchiveService) loadTurns(ctx context.Context, rows []ArchiveTurn) ([]ArchiveTurnView, error) {
	out := make([]ArchiveTurnView, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	images, err := s.repo.ListTurnImages(ctx, ids)
	if err != nil {
		return nil, err
	}
	rt := s.currentRuntime()
	sem := make(chan struct{}, archiveFetchParallel)
	var wg sync.WaitGroup
	for i := range rows {
		row := rows[i]
		view := &out[i]
		*view = ArchiveTurnView{
			ID: row.ID, UserID: row.UserID, Email: row.Email, SessionKey: row.SessionKey, CreatedAt: row.CreatedAt, RequestID: row.RequestID,
			Model: row.Model, InboundEndpoint: row.InboundEndpoint, UserAgent: row.UserAgent, IPAddress: row.IPAddress,
			InputTokens: row.InputTokens, OutputTokens: row.OutputTokens, CacheReadTokens: row.CacheReadTokens, Reset: row.Reset,
			NewItems: []json.RawMessage{}, RequestParams: map[string]json.RawMessage{}, Images: []ArchiveImageView{},
		}
		for _, img := range images[row.ID] {
			view.Images = append(view.Images, ArchiveImageView{ID: img.ID, Mime: img.Mime, Bytes: img.Bytes})
		}
		if rt == nil || rt.store == nil {
			view.FetchError = "对话归档存储尚未配置（Archive storage is not configured）"
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					view.FetchError = fmt.Sprintf("读取归档失败（Fetch failed）：%v", r)
				}
			}()
			rec, err := fetchArchiveRecord(ctx, rt.store, row.ObjectKey)
			if err != nil {
				view.FetchError = fmt.Sprintf("读取归档失败（Fetch failed）：%v", err)
				return
			}
			view.Protocol = rec.Protocol
			view.Thread = rec.Thread
			view.PrefixItems = rec.PrefixItems
			view.Instructions = rec.Instructions
			if rec.NewItems != nil {
				view.NewItems = rec.NewItems
			}
			if rec.RequestParams != nil {
				view.RequestParams = rec.RequestParams
			}
			resp := rec.Response
			if resp.ToolCalls == nil {
				resp.ToolCalls = []ArchiveToolCall{}
			}
			if resp.Items == nil {
				resp.Items = []json.RawMessage{}
			}
			view.Response = &resp
			for _, img := range rec.Images {
				if img.URL != "" {
					view.ImageURLs = append(view.ImageURLs, img.URL)
				}
			}
			truncated := rec.Truncated
			view.Truncated = &truncated
		}()
	}
	wg.Wait()
	return out, nil
}

func fetchArchiveRecord(ctx context.Context, store ArchiveObjectStore, key string) (*archiveRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rc, _, err := store.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(io.LimitReader(rc, archiveObjectMaxBytes+1))
	if err != nil {
		return nil, err
	}
	// Some S3 clients / proxies already undo Content-Encoding: gzip.
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		raw, err = io.ReadAll(io.LimitReader(zr, archiveObjectMaxBytes+1))
		_ = zr.Close()
		if err != nil {
			return nil, err
		}
	}
	if len(raw) > archiveObjectMaxBytes {
		return nil, errors.New("archived object too large")
	}
	var rec archiveRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// OpenImage streams an archived image (not audited: the turns view that lists it is).
func (s *ConversationArchiveService) OpenImage(ctx context.Context, id int64) (io.ReadCloser, *ArchiveImage, error) {
	if id <= 0 {
		return nil, nil, ErrArchiveImageNotFound
	}
	img, err := s.repo.GetImage(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if img == nil {
		return nil, nil, ErrArchiveImageNotFound
	}
	rt := s.currentRuntime()
	if rt == nil || rt.store == nil {
		return nil, nil, ErrArchiveNotConfigured
	}
	rc, size, err := rt.store.GetObject(ctx, img.ObjectKey)
	if err != nil {
		return nil, nil, archiveStorageError("CONVERSATION_ARCHIVE_IMAGE_FETCH_FAILED", "读取图片失败", "Failed to fetch the image", err)
	}
	if size > 0 && size <= int64(^uint(0)>>1) {
		img.Bytes = int(size)
	}
	return rc, img, nil
}

// Stats returns the last 30 days from the index (not audited).
func (s *ConversationArchiveService) Stats(ctx context.Context) ([]ArchiveDayStats, error) {
	loc := timezone.Location()
	start := s.now().In(loc)
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(archiveStatsDays - 1))
	items, err := s.repo.DailyStats(ctx, start, loc.String())
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ArchiveDayStats{}
	}
	return items, nil
}

// ParseArchiveTime accepts RFC3339 or a date (YYYY-MM-DD, local time zone).
func ParseArchiveTime(value string) (*time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return nil, true
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return &t, true
	}
	if t, err := time.ParseInLocation("2006-01-02", value, timezone.Location()); err == nil {
		return &t, true
	}
	if ms, err := strconv.ParseInt(value, 10, 64); err == nil && ms > 0 {
		t := time.UnixMilli(ms)
		return &t, true
	}
	return nil, false
}
