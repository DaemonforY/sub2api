package service

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// UserAppStateMaxBytes bounds one stored document (favorites snapshots are the largest).
const UserAppStateMaxBytes = 1 << 20

// Namespaces companion apps may store. Anything else is rejected so the table cannot become a
// general-purpose blob store.
var userAppStateNamespaces = map[string]bool{
	"canvas.favorites": true,
	"canvas.drafts":    true,
}

var (
	ErrUserAppStateNamespace = infraerrors.BadRequest("APP_STATE_NAMESPACE_INVALID", "不支持的同步数据类型")
	ErrUserAppStateInvalid   = infraerrors.BadRequest("APP_STATE_INVALID", "同步数据格式不正确，需要是 JSON 对象")
	ErrUserAppStateTooLarge  = infraerrors.BadRequest("APP_STATE_TOO_LARGE", "同步数据太大（上限 1MB），请减少收藏数量后再试")
)

// UserAppState is one synced document. Version 0 means "nothing stored yet".
type UserAppState struct {
	Namespace string          `json:"namespace"`
	Value     json.RawMessage `json:"value"`
	Version   int64           `json:"version"`
	UpdatedAt *time.Time      `json:"updated_at,omitempty"`
}

type UserAppStateRepository interface {
	// Get returns nil when the user has no document in the namespace.
	Get(ctx context.Context, userID int64, namespace string) (*UserAppState, error)
	// CompareAndSet writes value when the stored version equals baseVersion (0 = must not exist)
	// and returns the stored document; applied is false (and nothing written) on a version mismatch.
	CompareAndSet(ctx context.Context, userID int64, namespace string, value json.RawMessage, baseVersion int64) (state *UserAppState, applied bool, err error)
}

type UserAppStateService struct {
	repo UserAppStateRepository
}

func NewUserAppStateService(repo UserAppStateRepository) *UserAppStateService {
	return &UserAppStateService{repo: repo}
}

func (s *UserAppStateService) Get(ctx context.Context, userID int64, namespace string) (*UserAppState, error) {
	if !userAppStateNamespaces[namespace] {
		return nil, ErrUserAppStateNamespace
	}
	state, err := s.repo.Get(ctx, userID, namespace)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return &UserAppState{Namespace: namespace, Value: json.RawMessage("{}"), Version: 0}, nil
	}
	return state, nil
}

// Put stores value if nobody else wrote since baseVersion. On a conflict the current document is
// returned with applied=false so the client can merge and retry.
func (s *UserAppStateService) Put(ctx context.Context, userID int64, namespace string, value json.RawMessage, baseVersion int64) (*UserAppState, bool, error) {
	if !userAppStateNamespaces[namespace] {
		return nil, false, ErrUserAppStateNamespace
	}
	if len(value) > UserAppStateMaxBytes {
		return nil, false, ErrUserAppStateTooLarge
	}
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, false, ErrUserAppStateInvalid
	}
	if baseVersion < 0 {
		baseVersion = 0
	}
	return s.repo.CompareAndSet(ctx, userID, namespace, json.RawMessage(trimmed), baseVersion)
}
