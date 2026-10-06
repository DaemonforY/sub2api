//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type mapSettings map[string]string

func (m mapSettings) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		out[k] = m[k]
	}
	return out, nil
}
func (m mapSettings) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		m[k] = v
	}
	return nil
}

type prefixEncryptor struct{}

func (prefixEncryptor) Encrypt(s string) (string, error) { return "enc:" + s, nil }
func (prefixEncryptor) Decrypt(s string) (string, error) { return strings.TrimPrefix(s, "enc:"), nil }

func TestLearnRun(t *testing.T) {
	ctx := context.Background()
	var got map[string]any
	var auth string
	fail := false
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		auth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		got = map[string]any{}
		_ = json.Unmarshal(body, &got)
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"no available accounts"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"model":"gpt-5.5","choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer gw.Close()

	repo := newLearnRepoStub()
	settings := mapSettings{}
	svc := NewLearnService(repo, settings, prefixEncryptor{}, gw.URL, LearnSources{})
	reason := func(err error) string { return infraerrors.Reason(err) }
	in := LearnRunInput{Lesson: "a4", Messages: []LearnMessage{{Role: "user", Content: "明天上海要带伞吗？"}},
		Tools: json.RawMessage(`[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object"}}}]`)}

	// Off until the admin sets a key and a model.
	_, err := svc.Run(ctx, 1, in)
	require.Equal(t, "LEARN_RUN_DISABLED", reason(err))
	require.False(t, svc.Config(ctx).RunEnabled)
	// On without a site key: own-key-only — nothing free, every run needs one of the learner's keys.
	_, err = svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 2, DailyCap: 3})
	require.NoError(t, err)
	cfg := svc.Config(ctx)
	require.True(t, cfg.RunEnabled)
	require.True(t, cfg.OwnKeyOnly)
	require.Zero(t, cfg.FreeRunsPerDay)
	_, err = svc.Run(ctx, 1, in)
	require.Equal(t, "LEARN_OWN_KEY_REQUIRED", reason(err))
	me, err := svc.Me(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, me.RunsLeft)
	st, err := svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 2, DailyCap: 3, APIKey: "sk-learn"})
	require.NoError(t, err)
	require.True(t, st.APIKeySet)
	require.Empty(t, st.APIKey) // never echoed
	require.Equal(t, "enc:sk-learn", settings[settingLearnAPIKey])
	require.True(t, svc.Config(ctx).RunEnabled)
	require.False(t, svc.Config(ctx).OwnKeyOnly)

	// Bad requests are refused before any call.
	bad := in
	bad.Lesson = "../x"
	_, err = svc.Run(ctx, 1, bad)
	require.Equal(t, "LEARN_LESSON_INVALID", reason(err))
	bad = in
	bad.Messages = []LearnMessage{{Role: "tool", Content: "x"}}
	_, err = svc.Run(ctx, 1, bad)
	require.Equal(t, "LEARN_RUN_INVALID", reason(err))
	bad = in
	bad.Tools = json.RawMessage(`[{"type":"code_interpreter"}]`)
	_, err = svc.Run(ctx, 1, bad)
	require.Equal(t, "LEARN_RUN_INVALID", reason(err))
	require.Empty(t, repo.runs)

	// A run: the learning key, the configured model and the output cap; tool calls come back.
	res, err := svc.Run(ctx, 1, in)
	require.NoError(t, err)
	require.Equal(t, "Bearer sk-learn", auth)
	require.Equal(t, "gpt-5.5", got["model"])
	require.EqualValues(t, learnMaxOutputTokens, got["max_completion_tokens"])
	require.NotNil(t, got["tools"])
	require.Equal(t, "tool_calls", res.FinishReason)
	require.Contains(t, string(res.ToolCalls), "get_weather")
	require.Equal(t, 1, res.RunsLeft)

	// A failed upstream call is reported and not counted.
	fail = true
	_, err = svc.Run(ctx, 1, in)
	require.Equal(t, "LEARN_RUN_FAILED", reason(err))
	require.Contains(t, err.Error(), "no available accounts")
	fail = false

	// Free runs per user, then the site-wide cap.
	_, err = svc.Run(ctx, 1, in)
	require.NoError(t, err)
	_, err = svc.Run(ctx, 1, in)
	require.Equal(t, "LEARN_RUN_QUOTA", reason(err))
	_, err = svc.Run(ctx, 2, in)
	require.NoError(t, err)
	_, err = svc.Run(ctx, 3, in)
	require.Equal(t, "LEARN_RUN_BUSY", reason(err))

	// Saving without a key keeps the stored one.
	_, err = svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.6", FreeRunsPerDay: 5, DailyCap: 0})
	require.NoError(t, err)
	require.Equal(t, "enc:sk-learn", settings[settingLearnAPIKey])
}

func TestLearnMarkDone(t *testing.T) {
	repo := newLearnRepoStub()
	svc := NewLearnService(repo, mapSettings{}, prefixEncryptor{}, "http://127.0.0.1:1", LearnSources{})
	_, err := svc.MarkDone(context.Background(), 1, []string{"a1", "drop table"})
	require.Equal(t, "LEARN_LESSON_INVALID", infraerrors.Reason(err))
	done, err := svc.MarkDone(context.Background(), 1, []string{"a1", "a1", "b3"})
	require.NoError(t, err)
	require.Len(t, done, 2)
}

func TestLearnDayStartIsBeijingMidnight(t *testing.T) {
	// 2026-10-04 17:30 UTC is 2026-10-05 01:30 in Beijing: the day began at 2026-10-04 16:00 UTC.
	start := learnDayStart(time.Date(2026, 10, 4, 17, 30, 0, 0, time.UTC))
	require.True(t, start.Equal(time.Date(2026, 10, 4, 16, 0, 0, 0, time.UTC)))
}

func TestLearnClearKey(t *testing.T) {
	ctx := context.Background()
	settings := mapSettings{}
	svc := NewLearnService(newLearnRepoStub(), settings, prefixEncryptor{}, "http://127.0.0.1:1", LearnSources{})
	_, err := svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 5, APIKey: "sk-site"})
	require.NoError(t, err)
	require.False(t, svc.Config(ctx).OwnKeyOnly)

	// Saving without a key keeps the stored one; clear_api_key removes it (learners then use their own keys).
	st, err := svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 5})
	require.NoError(t, err)
	require.True(t, st.APIKeySet)
	st, err = svc.SaveSettings(ctx, LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 5, ClearAPIKey: true})
	require.NoError(t, err)
	require.False(t, st.APIKeySet)
	require.Empty(t, settings[settingLearnAPIKey])
	cfg := svc.Config(ctx)
	require.True(t, cfg.RunEnabled)
	require.True(t, cfg.OwnKeyOnly)
	require.Zero(t, cfg.FreeRunsPerDay)
}
