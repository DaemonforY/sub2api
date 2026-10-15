//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type fakeContentLogRepo struct {
	mu        sync.Mutex
	targets   map[int64]string
	users     map[int64]bool
	access    []ContentLogAccess
	logs      []*RequestContentLog
	listed    []ContentLogFilter
	accessAt  []int // len(access) when ListLogs ran
	failAudit bool
	// listHook runs inside ListTargetIDs (after the snapshot is taken).
	listHook func()
}

func newFakeContentLogRepo() *fakeContentLogRepo {
	return &fakeContentLogRepo{targets: map[int64]string{}, users: map[int64]bool{45: true}}
}

func (f *fakeContentLogRepo) ListTargetIDs(context.Context) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []int64{}
	for id := range f.targets {
		out = append(out, id)
	}
	hook := f.listHook
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	f.mu.Lock()
	return out, nil
}
func (f *fakeContentLogRepo) ListTargets(context.Context) ([]ContentLogTarget, error) {
	return nil, nil
}
func (f *fakeContentLogRepo) UserExists(_ context.Context, id int64) (bool, error) {
	return f.users[id], nil
}
func (f *fakeContentLogRepo) EnableTarget(_ context.Context, id int64, note string, audit ContentLogAccess) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAudit {
		return errors.New("audit write failed")
	}
	f.targets[id] = note
	f.access = append(f.access, audit)
	return nil
}
func (f *fakeContentLogRepo) DisableTarget(_ context.Context, id int64, audit ContentLogAccess) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAudit {
		return errors.New("audit write failed")
	}
	delete(f.targets, id)
	f.access = append(f.access, audit)
	return nil
}
func (f *fakeContentLogRepo) InsertAccess(_ context.Context, a ContentLogAccess) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.access = append(f.access, a)
	return nil
}
func (f *fakeContentLogRepo) ListAccess(context.Context, int64, int) ([]ContentLogAccess, error) {
	return nil, nil
}
func (f *fakeContentLogRepo) BatchInsertLogs(_ context.Context, logs []*RequestContentLog) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, logs...)
	return int64(len(logs)), nil
}
func (f *fakeContentLogRepo) ListLogs(_ context.Context, filter ContentLogFilter) ([]RequestContentLog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listed = append(f.listed, filter)
	f.accessAt = append(f.accessAt, len(f.access))
	out := make([]RequestContentLog, 0, filter.Limit)
	for i := 0; i < filter.Limit; i++ {
		out = append(out, RequestContentLog{ID: int64(100 - i), UserID: filter.UserID})
	}
	return out, nil
}
func (f *fakeContentLogRepo) DeleteLogsBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}
func (f *fakeContentLogRepo) DeleteAccessBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func TestContentLogService_EnableDisableAudited(t *testing.T) {
	repo := newFakeContentLogRepo()
	s := NewContentLogService(repo)
	ctx := context.Background()

	err := s.Enable(ctx, 1, 999, "x", "1.2.3.4", "ua")
	require.True(t, infraerrors.IsNotFound(err))
	require.False(t, s.IsTarget(999))

	require.NoError(t, s.Enable(ctx, 1, 45, "  suspicious relay  ", "1.2.3.4", "ua"))
	require.True(t, s.IsTarget(45))
	require.Equal(t, "suspicious relay", repo.targets[45])
	require.NoError(t, s.Disable(ctx, 1, 45, "1.2.3.4", "ua"))
	require.False(t, s.IsTarget(45))

	require.Len(t, repo.access, 2)
	require.Equal(t, ContentLogActionEnable, repo.access[0].Action)
	require.Equal(t, "suspicious relay", repo.access[0].Detail)
	require.Equal(t, ContentLogActionDisable, repo.access[1].Action)
	require.Equal(t, int64(45), repo.access[1].TargetUserID)
}

func TestContentLogService_ListLogsAuditsFirst(t *testing.T) {
	repo := newFakeContentLogRepo()
	s := NewContentLogService(repo)
	_, err := s.ListLogs(context.Background(), 1, ContentLogFilter{}, "", "")
	require.Error(t, err)

	page, err := s.ListLogs(context.Background(), 1, ContentLogFilter{UserID: 45, SessionID: " s1 ", Limit: 999}, "ip", "ua")
	require.NoError(t, err)
	require.Equal(t, []int{1}, repo.accessAt, "the view must be audited before data is read")
	require.Equal(t, ContentLogActionView, repo.access[0].Action)
	require.Contains(t, repo.access[0].Detail, `"session_id":"s1"`)
	require.Equal(t, contentLogListMaxLimit, repo.listed[0].Limit)
	require.Len(t, page.Items, contentLogListMaxLimit)
	require.NotNil(t, page.NextBeforeID)
	require.Equal(t, page.Items[len(page.Items)-1].ID, *page.NextBeforeID)
	require.Equal(t, ContentLogRetentionDays, page.RetentionDays)
	for _, item := range page.Items {
		require.NotNil(t, item.ToolCalls)
	}
}

func TestContentLogService_RecordOnlyTargetsAndRedacts(t *testing.T) {
	repo := newFakeContentLogRepo()
	repo.targets[45] = ""
	s := NewContentLogService(repo)
	s.Start()

	session := "sess-1"
	endpoint := "/v1/responses"
	group := int64(2)
	usage := &UsageLog{UserID: 45, APIKeyID: 35, GroupID: &group, RequestID: "req-1", Model: "gpt-x", RequestedModel: "gpt-6.1-sol",
		SessionID: &session, InboundEndpoint: &endpoint, InputTokens: 10, OutputTokens: 5, CacheReadTokens: 3, CreatedAt: time.Now()}
	capture := &ContentCapture{
		UserText:  "my key is sk-" + "proj-ABCDEFGHIJKLMNOPQRSTUV and " + strings.Repeat("长", 3000),
		ToolCalls: []ContentToolCall{{Name: "shell", Arguments: `{"command":"export API_KEY=abc123456789 && ` + strings.Repeat("x", 500) + `"}`}},
		ReplyHead: "ok",
	}
	recordContentLog(s, usage, capture)
	other := *usage
	other.UserID = 7
	recordContentLog(s, &other, capture)
	recordContentLog(s, usage, nil)
	recordContentLog(nil, usage, capture)
	s.Stop()

	require.Len(t, repo.logs, 1)
	row := repo.logs[0]
	require.Equal(t, "req-1", row.RequestID)
	require.Equal(t, "gpt-6.1-sol", row.Model)
	require.Equal(t, "sess-1", row.SessionID)
	require.Equal(t, "/v1/responses", row.InboundEndpoint)
	require.Equal(t, int64(35), *row.APIKeyID)
	require.Equal(t, int64(2), *row.GroupID)
	require.Equal(t, 3, row.CacheReadTokens)
	require.NotContains(t, row.UserText, "ABCDEFGHIJKLMNOP")
	require.LessOrEqual(t, len([]rune(row.UserText)), ContentLogUserTextMaxRunes+1)
	require.Len(t, row.ToolCalls, 1)
	require.NotContains(t, row.ToolCalls[0].Arguments, "abc123456789")
	require.LessOrEqual(t, len([]rune(row.ToolCalls[0].Arguments)), ContentLogToolArgumentsMaxRunes+1)
	require.Equal(t, "ok", row.ReplyHead)
}

func TestContentLogService_QueueFullDrops(t *testing.T) {
	repo := newFakeContentLogRepo()
	repo.targets[45] = ""
	s := NewContentLogService(repo) // not started: nothing drains the queue
	s.reload(context.Background())
	usage := &UsageLog{UserID: 45, RequestID: "r"}
	for i := 0; i < contentLogQueueCapacity+10; i++ {
		s.RecordUsageContent(usage, &ContentCapture{UserText: "x"})
	}
	require.Equal(t, contentLogQueueCapacity, len(s.queue))
	require.Equal(t, uint64(10), s.dropped)
}

func TestContentLogService_FailedAuditChangesNothing(t *testing.T) {
	repo := newFakeContentLogRepo()
	repo.failAudit = true
	s := NewContentLogService(repo)
	require.Error(t, s.Enable(context.Background(), 1, 45, "n", "", ""))
	require.False(t, s.IsTarget(45))
}

func TestContentLogService_ReloadDoesNotUndoFreshChange(t *testing.T) {
	repo := newFakeContentLogRepo()
	s := NewContentLogService(repo)
	// The reload reads an empty target list, then an admin enables #45 before it applies.
	repo.listHook = func() {
		repo.listHook = nil
		require.NoError(t, s.Enable(context.Background(), 1, 45, "", "", ""))
	}
	s.reload(context.Background())
	require.True(t, s.IsTarget(45), "a stale reload must not drop a target enabled meanwhile")
	s.reload(context.Background())
	require.True(t, s.IsTarget(45))
}
