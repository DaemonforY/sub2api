//go:build integration

package repository

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func testPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestCommunityLifecycle(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	alice := mustCreateUser(t, integrationEntClient, &service.User{Email: "cm-a-" + suffix + "@test.local", Username: "ca" + short})
	bob := mustCreateUser(t, integrationEntClient, &service.User{Email: "cm-b-" + suffix + "@test.local", Username: "cb" + short})
	// Old enough accounts: not limited to the new-account daily cap.
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id IN ($1, $2)`, alice.ID, bob.ID)
	require.NoError(t, err)
	dir := t.TempDir()
	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(dir), nil)
	reason := func(err error) string { return infraerrors.Reason(err) }

	// No profile yet: publishing asks for one.
	_, err = svc.Publish(ctx, alice.ID, service.PublishInput{Images: [][]byte{testPNG(t, 64, 48, color.White)}})
	require.Equal(t, "COMMUNITY_PROFILE_REQUIRED", reason(err))
	_, err = svc.SaveProfile(ctx, alice.ID, service.ProfileInput{Handle: "1bad"})
	require.Equal(t, "COMMUNITY_HANDLE_INVALID", reason(err))
	_, err = svc.SaveProfile(ctx, alice.ID, service.ProfileInput{Handle: "hivegpt_x"})
	require.Equal(t, "COMMUNITY_HANDLE_RESERVED", reason(err))

	ha, hb := "al"+short, "bo"+short
	pa, err := svc.SaveProfile(ctx, alice.ID, service.ProfileInput{Handle: ha, DisplayName: "小林画画", Bio: "AI 插画", Avatar: testPNG(t, 300, 200, color.RGBA{200, 100, 50, 255})})
	require.NoError(t, err)
	require.Equal(t, ha, pa.Handle)
	require.Contains(t, pa.AvatarURL, service.CommunityMediaPublicPrefix)
	_, err = svc.SaveProfile(ctx, bob.ID, service.ProfileInput{Handle: ha})
	require.Equal(t, "COMMUNITY_HANDLE_TAKEN", reason(err))
	_, err = svc.SaveProfile(ctx, bob.ID, service.ProfileInput{Handle: hb, DisplayName: "Bob"})
	require.NoError(t, err)

	// Publish: two images, thumbnail written, prompt kept.
	w, err := svc.Publish(ctx, alice.ID, service.PublishInput{
		Images: [][]byte{testPNG(t, 1600, 900, color.RGBA{0, 0, 255, 255}), testPNG(t, 300, 300, color.Black)},
		Title:  "月光下的庭院", Prompt: "月光下的古风庭院，水墨风格", ShowPrompt: false, Model: "gpt-image-2",
		Params: []byte(`{"size":"16:9"}`), Tags: []string{"国风", "#插画", "国风"}, Visibility: "public",
	})
	require.NoError(t, err)
	require.Equal(t, service.WorkStatusApproved, w.Status)
	require.Equal(t, []string{"国风", "插画"}, w.Tags)
	require.Len(t, w.Media, 2)
	require.Equal(t, 1600, w.CoverWidth)
	thumb, ok := svc.Media().Path(filepath.Base(w.Media[0].ThumbURL))
	require.True(t, ok)
	cfg, _, err := image.DecodeConfig(mustOpen(t, thumb))
	require.NoError(t, err)
	require.Equal(t, 640, cfg.Width, "thumbnails are 640px wide")
	require.Equal(t, 360, cfg.Height)

	// Bob sees it without the hidden prompt; Alice sees her own prompt.
	seen, err := svc.Work(ctx, w.ID, bob.ID)
	require.NoError(t, err)
	require.Empty(t, seen.Prompt)
	require.Equal(t, ha, seen.Author.Handle)
	mine, err := svc.Work(ctx, w.ID, alice.ID)
	require.NoError(t, err)
	require.NotEmpty(t, mine.Prompt)
	require.True(t, mine.IsMine)

	// Feeds.
	latest, err := svc.Works(ctx, service.WorkQuery{Feed: "latest", ViewerID: bob.ID, Limit: 60})
	require.NoError(t, err)
	require.Contains(t, workIDs(latest), w.ID)
	tagged, err := svc.Works(ctx, service.WorkQuery{Feed: "latest", Tag: "插画", Limit: 60})
	require.NoError(t, err)
	require.Contains(t, workIDs(tagged), w.ID)

	// Risky text goes to review and is not public.
	risky, err := svc.Publish(ctx, alice.ID, service.PublishInput{Images: [][]byte{testPNG(t, 50, 50, color.White)}, Title: "百家乐 下注"})
	require.NoError(t, err)
	require.Equal(t, service.WorkStatusPending, risky.Status)
	_, err = svc.Work(ctx, risky.ID, bob.ID)
	require.Equal(t, "COMMUNITY_WORK_NOT_FOUND", reason(err))
	pending, err := svc.AdminWorks(ctx, "pending", 1, 30)
	require.NoError(t, err)
	require.Contains(t, workIDs(pending), risky.ID)

	// Like, favorite, follow; notifications are deduplicated while unread.
	st, err := svc.SetLike(ctx, bob.ID, w.ID, true)
	require.NoError(t, err)
	require.Equal(t, 1, st.LikeCount)
	require.True(t, st.LikedByMe)
	_, err = svc.SetLike(ctx, bob.ID, w.ID, true)
	require.NoError(t, err)
	_, _ = svc.SetLike(ctx, bob.ID, w.ID, false)
	st, err = svc.SetLike(ctx, bob.ID, w.ID, true)
	require.NoError(t, err)
	require.Equal(t, 1, st.LikeCount, "liking twice counts once")
	_, err = svc.SetFavorite(ctx, bob.ID, w.ID, true)
	require.NoError(t, err)
	prof, err := svc.SetFollow(ctx, bob.ID, ha, true)
	require.NoError(t, err)
	require.True(t, prof.FollowedByMe)
	require.Equal(t, 1, prof.FollowersCount)
	require.Equal(t, 1, prof.LikesReceived)
	_, err = svc.SetFollow(ctx, alice.ID, ha, true)
	require.Equal(t, "COMMUNITY_SELF_FOLLOW", reason(err))

	notes, err := svc.Notifications(ctx, alice.ID)
	require.NoError(t, err)
	kinds := map[string]int{}
	for _, n := range notes {
		kinds[n.Kind]++
		require.Equal(t, hb, n.Actor.Handle)
	}
	require.Equal(t, map[string]int{"like": 1, "favorite": 1, "follow": 1}, kinds)
	unread, _ := svc.UnreadNotifications(ctx, alice.ID)
	require.Equal(t, 3, unread)
	require.NoError(t, svc.MarkNotificationsRead(ctx, alice.ID))
	unread, _ = svc.UnreadNotifications(ctx, alice.ID)
	require.Equal(t, 0, unread)

	following, err := svc.Works(ctx, service.WorkQuery{Feed: "following", ViewerID: bob.ID})
	require.NoError(t, err)
	require.Equal(t, []int64{w.ID}, workIDs(following))
	favs, err := svc.Works(ctx, service.WorkQuery{Feed: "favorites", ViewerID: bob.ID})
	require.NoError(t, err)
	require.Equal(t, []int64{w.ID}, workIDs(favs))
	feed, err := svc.Works(ctx, service.WorkQuery{Feed: "recommended", ViewerID: bob.ID, Limit: 60})
	require.NoError(t, err)
	for _, item := range feed {
		if item.ID == w.ID {
			require.True(t, item.LikedByMe)
			require.True(t, item.Author.FollowedByMe)
		}
	}

	// Collections.
	col, err := svc.CreateCollection(ctx, alice.ID, service.CollectionInput{Title: "国风系列"})
	require.NoError(t, err)
	require.NoError(t, svc.SetCollectionItem(ctx, alice.ID, col.ID, w.ID, true))
	require.Error(t, svc.SetCollectionItem(ctx, bob.ID, col.ID, w.ID, true), "only the owner")
	col, err = svc.Collection(ctx, col.ID, bob.ID)
	require.NoError(t, err)
	require.Equal(t, 1, col.WorksCount)
	require.Len(t, col.CoverURLs, 1)
	inCol, err := svc.Works(ctx, service.WorkQuery{Feed: "collection", CollectionID: col.ID, ViewerID: bob.ID})
	require.NoError(t, err)
	require.Equal(t, []int64{w.ID}, workIDs(inCol))
	cols, err := svc.Collections(ctx, ha, bob.ID)
	require.NoError(t, err)
	require.Len(t, cols, 1)

	// Reports and moderation.
	require.NoError(t, svc.Report(ctx, bob.ID, w.ID, "copyright", "像我的作品", "9.9.9.9"))
	require.Equal(t, "COMMUNITY_REPORT_INVALID", reason(svc.Report(ctx, bob.ID, w.ID, "whatever", "", "9.9.9.9")))
	reported, err := svc.AdminWorks(ctx, "reported", 1, 30)
	require.NoError(t, err)
	require.Contains(t, workIDs(reported), w.ID)
	require.NoError(t, svc.AdminModerate(ctx, w.ID, "feature", ""))
	require.NoError(t, svc.AdminModerate(ctx, w.ID, "hide", "侵权"))
	_, err = svc.Work(ctx, w.ID, bob.ID)
	require.Equal(t, "COMMUNITY_WORK_NOT_FOUND", reason(err))
	notes, _ = svc.Notifications(ctx, alice.ID)
	require.Equal(t, "hidden", notes[0].Kind)
	require.Equal(t, "侵权", notes[0].Detail)
	require.NoError(t, svc.AdminModerate(ctx, w.ID, "approve", ""))

	// Banned authors cannot publish and their page disappears for others.
	require.NoError(t, svc.AdminBan(ctx, alice.ID, true))
	_, err = svc.Publish(ctx, alice.ID, service.PublishInput{Images: [][]byte{testPNG(t, 10, 10, color.White)}})
	require.Equal(t, "COMMUNITY_BANNED", reason(err))
	_, err = svc.Profile(ctx, ha, bob.ID)
	require.Equal(t, "COMMUNITY_USER_NOT_FOUND", reason(err))
	require.NoError(t, svc.AdminBan(ctx, alice.ID, false))

	// Deleting a work removes its files.
	file, _ := svc.Media().Path(filepath.Base(w.Media[0].URL))
	require.FileExists(t, file)
	require.Error(t, svc.DeleteWork(ctx, bob.ID, w.ID), "only the owner")
	require.NoError(t, svc.DeleteWork(ctx, alice.ID, w.ID))
	_, err = os.Stat(file)
	require.True(t, os.IsNotExist(err))
	col, err = svc.Collection(ctx, col.ID, alice.ID)
	require.NoError(t, err)
	require.Equal(t, 0, col.WorksCount)
}

func workIDs(works []service.Work) []int64 {
	out := []int64{}
	for _, w := range works {
		out = append(out, w.ID)
	}
	return out
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestCommunityShareCardsAndRestrictedAuthors(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "share-" + suffix + "@test.local", Username: "sh" + short})
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = $1`, user.ID)
	require.NoError(t, err)
	repo := NewCommunityRepository(integrationDB)
	svc := service.NewCommunityService(repo, service.NewCommunityMediaStore(t.TempDir()), nil)
	handle := "sh" + short
	_, err = svc.SaveProfile(ctx, user.ID, service.ProfileInput{Handle: handle, DisplayName: "分享者", Bio: "画 <国风>"})
	require.NoError(t, err)
	public, err := svc.Publish(ctx, user.ID, service.PublishInput{Images: [][]byte{testPNG(t, 800, 400, color.White)}, Title: "庭院 & 月光", Prompt: "水墨风格", ShowPrompt: true, Visibility: "public"})
	require.NoError(t, err)
	private, err := svc.Publish(ctx, user.ID, service.PublishInput{Images: [][]byte{testPNG(t, 50, 50, color.Black)}, Title: "私密", Visibility: "private"})
	require.NoError(t, err)
	col, err := svc.CreateCollection(ctx, user.ID, service.CollectionInput{Title: "合集"})
	require.NoError(t, err)
	require.NoError(t, svc.SetCollectionItem(ctx, user.ID, col.ID, public.ID, true))

	page := fmt.Sprintf("https://canvas.example.test/w/%d", public.ID)
	out, err := svc.ShareMetaHTML(ctx, fmt.Sprintf("/w/%d", public.ID), "https://hivegpt.example.test", page)
	require.NoError(t, err)
	require.Contains(t, out, `<meta property="og:title" content="庭院 &amp; 月光 · 分享者" />`)
	require.Contains(t, out, `content="水墨风格"`, "the shown prompt describes the work")
	require.Contains(t, out, `og:image" content="https://hivegpt.example.test/api/v1/community/media/`)
	require.Contains(t, out, `<meta property="og:image:width" content="640" />`)
	require.Contains(t, out, `<meta property="og:image:height" content="320" />`)
	require.Contains(t, out, page)

	for _, path := range []string{fmt.Sprintf("/w/%d", private.ID), "/w/999999999", "/w/abc", "/x/1", "/u/nobody_here", "/"} {
		out, err := svc.ShareMetaHTML(ctx, path, "https://hivegpt.example.test", "")
		require.NoError(t, err)
		require.Empty(t, out, path)
	}
	out, err = svc.ShareMetaHTML(ctx, "/u/"+strings.ToUpper(handle), "https://hivegpt.example.test", "")
	require.NoError(t, err)
	require.Contains(t, out, "分享者 (@"+handle+")")
	require.Contains(t, out, "画 &lt;国风&gt;")
	out, err = svc.ShareMetaHTML(ctx, fmt.Sprintf("/c/%d", col.ID), "https://hivegpt.example.test", "")
	require.NoError(t, err)
	require.Contains(t, out, "合集 · 分享者 的作品集")
	require.Contains(t, out, "og:image")

	// Restricted authors: listed for admins, no cards, and lifting the restriction restores them.
	require.NoError(t, svc.AdminBan(ctx, user.ID, true))
	restricted, err := svc.AdminRestricted(ctx)
	require.NoError(t, err)
	found := false
	for _, a := range restricted {
		if a.UserID == user.ID {
			found = true
			require.Equal(t, handle, a.Handle)
			require.Equal(t, 2, a.WorksCount)
		}
	}
	require.True(t, found)
	fresh := service.NewCommunityService(repo, service.NewCommunityMediaStore(t.TempDir()), nil)
	out, err = fresh.ShareMetaHTML(ctx, fmt.Sprintf("/w/%d", public.ID), "https://hivegpt.example.test", "")
	require.NoError(t, err)
	require.Empty(t, out, "a restricted author's works get no card")
	require.NoError(t, svc.AdminBan(ctx, user.ID, false))
	restricted, err = svc.AdminRestricted(ctx)
	require.NoError(t, err)
	for _, a := range restricted {
		require.NotEqual(t, user.ID, a.UserID)
	}
	_, err = svc.Publish(ctx, user.ID, service.PublishInput{Images: [][]byte{testPNG(t, 20, 20, color.White)}})
	require.NoError(t, err, "can publish again")
}
