//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubUserAppStateRepo struct {
	puts int
}

func (r *stubUserAppStateRepo) Get(context.Context, int64, string) (*UserAppState, error) {
	return nil, nil
}

func (r *stubUserAppStateRepo) CompareAndSet(_ context.Context, _ int64, namespace string, value json.RawMessage, base int64) (*UserAppState, bool, error) {
	r.puts++
	return &UserAppState{Namespace: namespace, Value: value, Version: base + 1}, true, nil
}

func TestUserAppStateServiceValidation(t *testing.T) {
	repo := &stubUserAppStateRepo{}
	svc := NewUserAppStateService(repo)
	ctx := context.Background()

	_, err := svc.Get(ctx, 1, "anything")
	require.ErrorIs(t, err, ErrUserAppStateNamespace)

	empty, err := svc.Get(ctx, 1, "canvas.drafts")
	require.NoError(t, err)
	require.Equal(t, int64(0), empty.Version)
	require.Equal(t, "{}", string(empty.Value))

	for _, bad := range []string{``, `[]`, `"x"`, `{"a":`} {
		_, _, err = svc.Put(ctx, 1, "canvas.drafts", json.RawMessage(bad), 0)
		require.ErrorIs(t, err, ErrUserAppStateInvalid, bad)
	}
	_, _, err = svc.Put(ctx, 1, "canvas.drafts", json.RawMessage(`{"a":"`+strings.Repeat("x", UserAppStateMaxBytes)+`"}`), 0)
	require.ErrorIs(t, err, ErrUserAppStateTooLarge)
	_, _, err = svc.Put(ctx, 1, "users.passwords", json.RawMessage(`{}`), 0)
	require.ErrorIs(t, err, ErrUserAppStateNamespace)
	require.Equal(t, 0, repo.puts)

	state, applied, err := svc.Put(ctx, 1, "canvas.favorites", json.RawMessage(`  {"items":{}}  `), 3)
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, int64(4), state.Version)
	require.Equal(t, `{"items":{}}`, string(state.Value))
}
