//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDesktopSkillRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewDesktopSkillRepository(integrationDB)
	tag := fmt.Sprintf("it-dsk-%d", time.Now().UnixNano()%1_000_000_000)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM desktop_skills WHERE slug LIKE $1`, tag+"%")
	})

	// The migration seeds the 8 office templates as published skills.
	published, err := repo.ListDesktopSkills(ctx, true)
	require.NoError(t, err)
	seeded := map[string]bool{}
	for _, s := range published {
		seeded[s.Slug] = true
	}
	for _, slug := range []string{"invoices", "rename", "merge-excel", "word-format", "meeting", "pdf-table", "ppt-outline", "report"} {
		require.True(t, seeded[slug], slug)
	}

	s := &service.DesktopSkill{Slug: tag + "-a", Title: "测试技能", Icon: "🧪", Summary: "测试", Prompt: "做点什么",
		NeedsOffice: true, Status: service.DesktopSkillDraft, SortOrder: 5}
	id, err := repo.CreateDesktopSkill(ctx, s, 0)
	require.NoError(t, err)
	_, err = repo.CreateDesktopSkill(ctx, s, 0)
	require.ErrorIs(t, err, service.ErrDesktopSkillSlugUsed)

	got, err := repo.GetDesktopSkill(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 1, got.Version)
	require.Equal(t, service.DesktopSkillDraft, got.Status)

	// Saving bumps the version and keeps the status when none is given.
	got.Title, got.Status = "改过的标题", ""
	require.NoError(t, repo.UpdateDesktopSkill(ctx, got, 0))
	got, err = repo.GetDesktopSkill(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 2, got.Version)
	require.Equal(t, "改过的标题", got.Title)
	require.Equal(t, service.DesktopSkillDraft, got.Status)

	// Publishing bumps it too and makes it visible to the app.
	require.NoError(t, repo.SetDesktopSkillStatus(ctx, id, service.DesktopSkillPublished, 0))
	published, err = repo.ListDesktopSkills(ctx, true)
	require.NoError(t, err)
	found := false
	for _, p := range published {
		if p.ID == id {
			found = true
			require.Equal(t, 3, p.Version)
		}
	}
	require.True(t, found)

	require.NoError(t, repo.DeleteDesktopSkill(ctx, id))
	_, err = repo.GetDesktopSkill(ctx, id)
	require.ErrorIs(t, err, service.ErrDesktopSkillNotFound)
	require.ErrorIs(t, repo.DeleteDesktopSkill(ctx, id), service.ErrDesktopSkillNotFound)
	require.ErrorIs(t, repo.SetDesktopSkillStatus(ctx, id, service.DesktopSkillHidden, 0), service.ErrDesktopSkillNotFound)
}

func TestDesktopOverview(t *testing.T) {
	ctx := context.Background()
	repo := NewDesktopSkillRepository(integrationDB)
	rows := trackCommitted(t)
	tag := fmt.Sprintf("itdsk%d", time.Now().UnixNano()%1_000_000_000)
	prefix := tag + " · "

	u := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: tag + "@example.com"}))
	now := time.Now()
	mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Name: prefix + "电脑 A #1", Key: "sk-" + tag + "a"})
	mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Name: prefix + "电脑 B #2", Key: "sk-" + tag + "b"})
	mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Name: "别的 key", Key: "sk-" + tag + "c"})
	// The fixture doesn't set last_used_at; computer A was used just now.
	_, err := integrationDB.ExecContext(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE key = $1`, "sk-"+tag+"a")
	require.NoError(t, err)

	visitor1, visitor2 := tag+"-v1", tag+"-v2"
	rows.visitors = append(rows.visitors, visitor1, visitor2)
	tpl := tag + "-tpl"
	insert := func(visitor, event string, at time.Time, props map[string]string) {
		b, _ := json.Marshal(props)
		_, err := integrationDB.ExecContext(ctx, `INSERT INTO analytics_events (created_at, event, app, visitor_id, props) VALUES ($1, $2, 'desktop', $3, $4)`,
			at, event, visitor, string(b))
		require.NoError(t, err)
	}
	insert(visitor1, "desktop_open", now.Add(-50*time.Hour), nil)
	insert(visitor1, "desktop_open", now.Add(-time.Hour), nil) // a second day: came back
	insert(visitor2, "desktop_open", now.Add(-time.Hour), nil)
	insert(visitor1, "desktop_task_start", now, map[string]string{"template": tpl})
	insert(visitor1, "desktop_task_done", now, map[string]string{"template": tpl, "secs": "100"})
	insert(visitor1, "desktop_task_start", now, map[string]string{"template": tpl})
	insert(visitor1, "desktop_task_done", now, map[string]string{"template": tpl, "secs": "200"})
	insert(visitor2, "desktop_task_start", now, map[string]string{"template": tpl})
	insert(visitor2, "desktop_task_failed", now, map[string]string{"template": tpl, "reason": "quota"})
	insert(visitor2, "desktop_engine_failed", now, map[string]string{"step": "office", "reason": tag})

	o, err := repo.DesktopOverview(ctx, now.Add(-72*time.Hour), prefix)
	require.NoError(t, err)
	require.EqualValues(t, 2, o.Devices)
	require.EqualValues(t, 1, o.ActiveDevices)
	require.Len(t, o.RecentDevices, 2)
	require.Equal(t, u.Email, o.RecentDevices[0].UserEmail)
	require.Equal(t, prefix+"电脑 A #1", o.RecentDevices[0].Name, "most recently used first")
	require.Len(t, o.Funnel, len(service.DesktopFunnelEvents))
	require.Equal(t, "desktop_installed", o.Funnel[0].Event)
	require.GreaterOrEqual(t, o.Returned, int64(1))

	var stat *service.DesktopTemplateStat
	for i := range o.Templates {
		if o.Templates[i].Template == tpl {
			stat = &o.Templates[i]
		}
	}
	require.NotNil(t, stat)
	require.EqualValues(t, 3, stat.Starts)
	require.EqualValues(t, 2, stat.Done)
	require.EqualValues(t, 1, stat.Failed)
	require.InDelta(t, 150, stat.AvgDoneSec, 0.01)
	require.Equal(t, "quota", stat.TopFailure)

	foundFailure := false
	for _, f := range o.Failures {
		if f.Reason == tag {
			foundFailure = f.Step == "office" && f.Count == 1
		}
	}
	require.True(t, foundFailure)
}
