//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluateCapacity(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		used    float64
		left    time.Duration
		alert   bool
		project float64
	}{
		{"80% with 3 days left", 80, 72 * time.Hour, true, 140},
		{"61% four days in", 61, 72 * time.Hour, true, 107}, // on pace to run out
		{"32% four days in", 32, 72 * time.Hour, false, 56},
		{"85% but resets tomorrow", 85, 20 * time.Hour, false, 96},
		{"pace not judged on day one", 30, 6 * 24 * time.Hour, false, 0},
		{"window already reset", 90, -time.Hour, false, 0},
	}
	for _, c := range cases {
		a := OpsCapacityAccount{UsedPct: c.used, ResetAt: now.Add(c.left)}
		evaluateCapacity(&a, now)
		require.Equal(t, c.alert, a.Reason != "", c.name)
		require.InDelta(t, c.project, a.ProjectedPct, 1, c.name)
	}
}

func TestCapacityAccounts(t *testing.T) {
	rows := []map[string]any{
		{"id": float64(9), "name": "outlook ph", "status": "active", "codex_7d_used_percent": float64(32), "codex_7d_reset_at": "2026-10-14T03:30:50Z"},
		{"id": float64(8), "name": "201ai", "status": "active", "codex_7d_used_percent": "61", "codex_7d_reset_at": "2026-10-14T11:30:16+08:00"},
		{"id": float64(1), "name": "apikey", "status": "active"},                                                                                 // no window
		{"id": float64(5), "name": "off", "status": "disabled", "codex_7d_used_percent": float64(99), "codex_7d_reset_at": "2026-10-14T03:30:50Z"}, // disabled
	}
	got := capacityAccounts(rows)
	require.Len(t, got, 2)
	require.Equal(t, "201ai", got[0].Name)
	require.InDelta(t, 61, got[0].UsedPct, 1e-9)
	require.True(t, got[0].ResetAt.Equal(time.Date(2026, 10, 14, 3, 30, 16, 0, time.UTC)))
}

func TestOpsCapacityCheck(t *testing.T) {
	ctx := context.Background()
	used := "61"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "admin-key", r.Header.Get("x-api-key"))
		require.Equal(t, "/api/v1/admin/accounts", r.URL.Path)
		reset := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
		_, _ = io.WriteString(w, `{"code":0,"message":"success","data":{"total":2,"items":[
			{"id":8,"name":"201ai","status":"active","extra":{"codex_7d_used_percent":`+used+`,"codex_7d_reset_at":"`+reset+`"}},
			{"id":9,"name":"outlook ph","status":"active","extra":{"codex_7d_used_percent":32,"codex_7d_reset_at":"`+reset+`"}}]}}`)
	}))
	t.Cleanup(srv.Close)

	svc, settings, _, _ := newOpsAgentForTest(t, "http://127.0.0.1:1")
	var mails []string
	svc.capacityMail = func(subject, body string) { mails = append(mails, subject+"\n"+body) }

	// Not configured: nothing happens.
	require.NoError(t, svc.CheckCapacity(ctx))
	require.Empty(t, mails)

	st, err := svc.SaveSettings(ctx, 1, OpsAgentSettings{Model: "gpt-5.5", UpstreamName: "gorustai", UpstreamURL: srv.URL, UpstreamKey: "admin-key", CapacityAlert: true})
	require.NoError(t, err)
	require.True(t, st.CapacityAlert)

	require.NoError(t, svc.CheckCapacity(ctx))
	require.Len(t, mails, 1)
	require.Contains(t, mails[0], "[容量告警] 上游 gorustai")
	require.Contains(t, mails[0], "201ai")
	require.NotContains(t, mails[0][:len("[容量告警] 上游 gorustai 的账号 7 天额度快用完：201ai")+5], "outlook ph", "only 201ai is alerting")
	require.Contains(t, mails[0], "<td>outlook ph</td><td>32%</td>", "the table lists every account")

	// Once per account per window.
	used = "85"
	require.NoError(t, svc.CheckCapacity(ctx))
	require.Len(t, mails, 1)
	require.Contains(t, settings[settingOpsCapacityAlerted], `"8":`)

	// Switched off: quiet.
	settings[settingOpsCapacityAlerted] = ""
	_, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Model: "gpt-5.5", UpstreamURL: srv.URL, CapacityAlert: false})
	require.NoError(t, err)
	require.False(t, svc.Settings(ctx).CapacityAlert)
	require.NoError(t, svc.CheckCapacity(ctx))
	require.Len(t, mails, 1)
}
