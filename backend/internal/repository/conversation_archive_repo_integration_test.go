//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestConversationArchiveRepository(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	admin := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "ca-admin-" + sfx + "@test.local", Username: "caa" + sfx[len(sfx)-6:]}))
	owner := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "ca-owner-" + sfx + "@test.local", Username: "cao" + sfx[len(sfx)-6:]}))
	ids := pq.Array([]int64{admin.ID, owner.ID})
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, `DELETE FROM conversation_archive_turns WHERE user_id = ANY($1)`, ids)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `DELETE FROM content_log_access_logs WHERE admin_id = ANY($1)`, ids)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `DELETE FROM conversation_archive_purges WHERE user_id = ANY($1)`, ids)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, service.SettingKeyConversationArchiveConfig)
		require.NoError(t, err)
	})
	repo := NewConversationArchiveRepository(integrationDB)

	// Settings row + audit in one transaction; a bad audit action rolls the save back.
	raw, err := repo.GetConfigRaw(ctx)
	require.NoError(t, err)
	require.Empty(t, raw)
	require.Error(t, repo.SaveConfig(ctx, `{"enabled":true}`, service.ContentLogAccess{AdminID: admin.ID, Action: "bogus"}))
	raw, err = repo.GetConfigRaw(ctx)
	require.NoError(t, err)
	require.Empty(t, raw)
	require.NoError(t, repo.SaveConfig(ctx, `{"enabled":false}`, service.ContentLogAccess{AdminID: admin.ID, Action: service.ContentLogActionArchiveConfig, Detail: `{"enabled":false}`}))
	raw, err = repo.GetConfigRaw(ctx)
	require.NoError(t, err)
	require.Equal(t, `{"enabled":false}`, raw)
	require.NoError(t, repo.InsertAccess(ctx, service.ContentLogAccess{AdminID: admin.ID, Action: service.ContentLogActionArchiveView, TargetUserID: owner.ID, Detail: "{}"}))

	now := time.Now().UTC().Truncate(time.Millisecond)
	key := int64(35)
	insert := func(session, requestID string, at time.Time, model, preview string, images int) int64 {
		t.Helper()
		var imgs []service.ArchiveImage
		for i := 0; i < images; i++ {
			imgs = append(imgs, service.ArchiveImage{UserID: owner.ID, ObjectKey: fmt.Sprintf("archive/img/%d/%s_%d.png", owner.ID, requestID, i), Mime: "image/png", Bytes: 10, CreatedAt: at})
		}
		id, err := repo.InsertTurn(ctx, &service.ArchiveTurn{
			UserID: owner.ID, APIKeyID: &key, SessionKey: session, RequestID: requestID, CreatedAt: at, Model: model,
			InboundEndpoint: "/v1/responses", UserAgent: "codex", IPAddress: "1.1.1.1", InputTokens: 10, OutputTokens: 2,
			ObjectKey: "archive/conv/" + requestID, Bytes: 100, NewItemCount: 1, ImageCount: images, Preview: preview,
		}, imgs)
		require.NoError(t, err)
		return id
	}
	t1 := insert("sess-1", "req-"+sfx+"-1", now.Add(-3*time.Hour), "gpt-a", "first question", 0)
	t2 := insert("sess-1", "req-"+sfx+"-2", now.Add(-2*time.Hour), "gpt-b", "follow up\x00", 2)
	t3 := insert("sess-2", "req-"+sfx+"-3", now.Add(-1*time.Hour), "gpt-a", "", 0)
	old := insert("sess-old", "req-"+sfx+"-old", now.AddDate(0, 0, -200), "gpt-a", "ancient", 1)

	// Sessions: newest last turn first, aggregated over the whole session.
	from := now.AddDate(0, 0, -30)
	sessions, err := repo.ListSessions(ctx, service.ArchiveSessionFilter{UserID: owner.ID, From: &from, Limit: 10})
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	require.Equal(t, "sess-2", sessions[0].SessionKey)
	require.Equal(t, "sess-1", sessions[1].SessionKey)
	s1 := sessions[1]
	require.Equal(t, owner.Email, s1.Email)
	require.EqualValues(t, 2, s1.Turns)
	require.ElementsMatch(t, []string{"gpt-a", "gpt-b"}, s1.Models)
	require.Equal(t, []string{"/v1/responses"}, s1.Endpoints)
	require.Equal(t, "first question", s1.Preview)
	require.EqualValues(t, 2, s1.ImageCount)
	require.EqualValues(t, 20, s1.InputTokens)
	require.True(t, s1.FirstAt.Equal(now.Add(-3*time.Hour)))
	require.True(t, s1.LastAt.Equal(now.Add(-2*time.Hour)))
	// Paging by last_at.
	page2, err := repo.ListSessions(ctx, service.ArchiveSessionFilter{UserID: owner.ID, From: &from, Before: &sessions[0].LastAt, Limit: 10})
	require.NoError(t, err)
	require.Len(t, page2, 1)
	require.Equal(t, "sess-1", page2[0].SessionKey)
	// No user filter (default range from the service), to exclusive.
	to := now.Add(-90 * time.Minute)
	all, err := repo.ListSessions(ctx, service.ArchiveSessionFilter{From: &from, To: &to, Limit: 100})
	require.NoError(t, err)
	found := false
	for _, s := range all {
		require.NotEqual(t, "sess-2", s.SessionKey)
		if s.UserID == owner.ID && s.SessionKey == "sess-1" {
			found = true
		}
	}
	require.True(t, found)

	// Turns newest first, paged by id.
	turns, err := repo.ListTurns(ctx, owner.ID, "sess-1", 0, 10)
	require.NoError(t, err)
	require.Len(t, turns, 2)
	require.Equal(t, t2, turns[0].ID)
	require.Equal(t, "follow up", turns[0].Preview, "NUL bytes stripped")
	require.Equal(t, key, *turns[0].APIKeyID)
	require.Nil(t, turns[0].GroupID)
	turns, err = repo.ListTurns(ctx, owner.ID, "sess-1", t2, 10)
	require.NoError(t, err)
	require.Len(t, turns, 1)
	require.Equal(t, t1, turns[0].ID)

	// Transcript order is (created_at, id): a turn indexed later with an earlier time sorts by time.
	o1 := insert("sess-order", "req-"+sfx+"-o1", now.Add(-50*time.Minute), "m", "", 0)
	o3 := insert("sess-order", "req-"+sfx+"-o3", now.Add(-10*time.Minute), "m", "", 0)
	o2 := insert("sess-order", "req-"+sfx+"-o2", now.Add(-30*time.Minute), "m", "", 0)
	order := func(beforeID int64, limit int) []int64 {
		t.Helper()
		got, err := repo.ListTurns(ctx, owner.ID, "sess-order", beforeID, limit)
		require.NoError(t, err)
		ids := []int64{}
		for _, g := range got {
			ids = append(ids, g.ID)
		}
		return ids
	}
	require.Equal(t, []int64{o3, o2, o1}, order(0, 10))
	require.Equal(t, []int64{o3, o2}, order(0, 2))
	require.Equal(t, []int64{o1}, order(o2, 2), "exact cursor: strictly older than o2")
	require.Equal(t, []int64{o2, o1}, order(o3, 10))
	// A cursor id that isn't in the session (the UI's id + 1 jump) starts at the row with the
	// largest smaller id, inclusive, and goes back in time from there.
	require.Equal(t, []int64{o2, o1}, order(o2+1000, 10))

	byReq, err := repo.GetTurnByRequestID(ctx, "req-"+sfx+"-3")
	require.NoError(t, err)
	require.Equal(t, t3, byReq.ID)
	require.Equal(t, owner.Email, byReq.Email)
	missing, err := repo.GetTurnByRequestID(ctx, "nope-"+sfx)
	require.NoError(t, err)
	require.Nil(t, missing)

	imgs, err := repo.ListTurnImages(ctx, []int64{t1, t2})
	require.NoError(t, err)
	require.Len(t, imgs[t2], 2)
	require.Empty(t, imgs[t1])
	img, err := repo.GetImage(ctx, imgs[t2][0].ID)
	require.NoError(t, err)
	require.Equal(t, "image/png", img.Mime)
	none, err := repo.GetImage(ctx, -1)
	require.NoError(t, err)
	require.Nil(t, none)

	// Image wall: newest first by id, with the turn's model, session and prompt; filters and cursor.
	wall, err := repo.ListImages(ctx, service.ArchiveImageFilter{UserID: owner.ID, From: &from, Limit: 10})
	require.NoError(t, err)
	require.Len(t, wall, 2, "the 200-day-old image is outside the range")
	require.Equal(t, imgs[t2][1].ID, wall[0].ID)
	require.Equal(t, t2, wall[0].TurnID)
	require.Equal(t, owner.Email, wall[0].Email)
	require.Equal(t, "gpt-b", wall[0].Model)
	require.Equal(t, "sess-1", wall[0].SessionKey)
	require.Equal(t, "req-"+sfx+"-2", wall[0].RequestID)
	require.Equal(t, "follow up", wall[0].Prompt)
	wall, err = repo.ListImages(ctx, service.ArchiveImageFilter{UserID: owner.ID, From: &from, BeforeID: imgs[t2][1].ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, wall, 1)
	require.Equal(t, imgs[t2][0].ID, wall[0].ID)
	wall, err = repo.ListImages(ctx, service.ArchiveImageFilter{UserID: owner.ID, Model: "gpt-a", Limit: 10})
	require.NoError(t, err)
	require.Len(t, wall, 1)
	require.Equal(t, old, wall[0].TurnID)
	models, err := repo.ListImageModels(ctx, from)
	require.NoError(t, err)
	require.Contains(t, models, "gpt-b")
	require.NotContains(t, models, "m", "turns without images are not listed")

	stats, err := repo.DailyStats(ctx, now.AddDate(0, 0, -2), "Asia/Shanghai")
	require.NoError(t, err)
	var turnsTotal int64
	for _, d := range stats {
		turnsTotal += d.Turns
	}
	require.GreaterOrEqual(t, turnsTotal, int64(3))

	// q resolves id, email and username.
	for _, q := range []string{fmt.Sprint(owner.ID), owner.Email, "CA-OWNER-" + sfx + "@TEST.LOCAL", owner.Username} {
		id, err := repo.ResolveUserID(ctx, q)
		require.NoError(t, err)
		require.Equal(t, owner.ID, id, q)
	}
	id, err := repo.ResolveUserID(ctx, "nobody-"+sfx+"@x")
	require.NoError(t, err)
	require.Zero(t, id)

	// Retention: expired rows with their image keys, then deletion cascades to images.
	expired, err := repo.ListExpired(ctx, now.AddDate(0, 0, -180), 1000)
	require.NoError(t, err)
	var mine *service.ArchiveExpiredTurn
	for i := range expired {
		if expired[i].ID == old {
			mine = &expired[i]
		}
	}
	require.NotNil(t, mine)
	require.Len(t, mine.ImageKeys, 1)
	n, err := repo.DeleteTurns(ctx, []int64{old})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	var imgLeft int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_archive_images WHERE turn_id = $1`, old).Scan(&imgLeft))
	require.Zero(t, imgLeft)

	// Pending purges: prefixes merge, attempts count, the user's deletion state is reported.
	require.NoError(t, repo.QueuePurge(ctx, owner.ID, []string{"archive/"}))
	require.NoError(t, repo.QueuePurge(ctx, owner.ID, []string{"archive/", "old/"}))
	require.NoError(t, repo.FailPurge(ctx, owner.ID, "boom\x00"))
	purges, err := repo.ListPurges(ctx, 100)
	require.NoError(t, err)
	var pending *service.ArchivePurge
	for i := range purges {
		if purges[i].UserID == owner.ID {
			pending = &purges[i]
		}
	}
	require.NotNil(t, pending)
	require.ElementsMatch(t, []string{"archive/", "old/"}, pending.Prefixes)
	require.Equal(t, 1, pending.Attempts)
	require.False(t, pending.UserDeleted)
	require.NoError(t, repo.FinishPurge(ctx, owner.ID))
	purges, err = repo.ListPurges(ctx, 100)
	require.NoError(t, err)
	for _, p := range purges {
		require.NotEqual(t, owner.ID, p.UserID)
	}

	// Account deletion purges the index rows.
	require.NoError(t, NewAccountDeletionRepository(integrationDB).PurgeOwnedContent(ctx, owner.ID))
	var left int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_archive_turns WHERE user_id = $1`, owner.ID).Scan(&left))
	require.Zero(t, left)
}
