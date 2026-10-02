//go:build integration

package repository

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lib/pq"
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

func TestCommunityWorkContestEntries(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	author := mustCreateUser(t, integrationEntClient, &service.User{Email: "wc-" + suffix + "@test.local", Username: "wc" + short})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "wc2-" + suffix + "@test.local", Username: "wd" + short})
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = ANY($1)`, pq.Array([]int64{author.ID, other.ID}))
	require.NoError(t, err)

	contests := service.NewContestService(NewContestRepository(integrationDB), NewUserRepository(integrationEntClient, integrationDB),
		NewRedeemCodeRepository(integrationEntClient), nil, nil, service.NewContestImageStore(t.TempDir()))
	now := time.Now().UTC()
	contest, err := contests.CreateContest(ctx, &service.Contest{
		Title: "作品投稿赛 " + short, Status: service.ContestStatusPublished, MaxEntriesPerUser: 3, VotesPerUser: 3,
		SubmissionStartAt: now.Add(-time.Hour), SubmissionEndAt: now.Add(time.Hour),
		VotingStartAt: now.Add(-time.Hour), VotingEndAt: now.Add(2 * time.Hour),
	}, 0)
	require.NoError(t, err)

	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), nil)
	svc.SetContests(contests)
	_, err = svc.SaveProfile(ctx, author.ID, service.ProfileInput{Handle: "wc" + short})
	require.NoError(t, err)
	work, err := svc.Publish(ctx, author.ID, service.PublishInput{
		Images: [][]byte{testPNG(t, 64, 64, color.White), testPNG(t, 32, 48, color.Black)}, Title: "月光庭院", Description: "说明", Prompt: "隐藏的提示词", ShowPrompt: false, Visibility: "public",
	})
	require.NoError(t, err)
	private, err := svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 20, 20, color.White)}, Title: "私密", Visibility: "private"})
	require.NoError(t, err)

	in := service.WorkContestInput{ContestID: contest.ID, WorkID: work.ID, ImageIndex: 1}
	_, err = svc.EnterContest(ctx, other.ID, in)
	require.ErrorIs(t, err, service.ErrCommunityWorkNotFound, "only the author may enter a work")
	_, err = svc.EnterContest(ctx, author.ID, service.WorkContestInput{ContestID: contest.ID, WorkID: private.ID})
	require.ErrorIs(t, err, service.ErrCommunityWorkPrivate)
	_, err = svc.EnterContest(ctx, author.ID, service.WorkContestInput{ContestID: contest.ID, WorkID: work.ID, ImageIndex: 5})
	require.ErrorIs(t, err, service.ErrCommunityWorkImageNotFound)

	entry, err := svc.EnterContest(ctx, author.ID, in)
	require.NoError(t, err)
	require.Equal(t, "月光庭院", entry.Title, "the work's title is the default")
	require.Equal(t, "说明", entry.Description)
	require.Empty(t, entry.Prompt, "a hidden prompt stays hidden")
	require.NotNil(t, entry.WorkID)
	require.Equal(t, work.ID, *entry.WorkID)
	_, err = svc.EnterContest(ctx, author.ID, in)
	require.ErrorIs(t, err, service.ErrContestWorkEntered, "one live entry per work and contest")

	entries, _, err := contests.ListPublicEntries(ctx, contest.ID, 0, "new", 1, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, work.ID, *entries[0].WorkID)

	shown, err := svc.Work(ctx, work.ID, 0)
	require.NoError(t, err)
	require.Len(t, shown.Contests, 1)
	require.Equal(t, contest.ID, shown.Contests[0].ContestID)
	require.Equal(t, entry.ID, shown.Contests[0].EntryID)

	// Withdrawing frees the work for another entry.
	require.NoError(t, contests.WithdrawEntry(ctx, contest.ID, entry.ID, author.ID))
	shown, err = svc.Work(ctx, work.ID, 0)
	require.NoError(t, err)
	require.Empty(t, shown.Contests)
	_, err = svc.EnterContest(ctx, author.ID, service.WorkContestInput{ContestID: contest.ID, WorkID: work.ID, Title: "新标题"})
	require.NoError(t, err)
}

func TestCommunityCreatorStats(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	author := mustCreateUser(t, integrationEntClient, &service.User{Email: "cs-" + suffix + "@test.local", Username: "cs" + short})
	fan := mustCreateUser(t, integrationEntClient, &service.User{Email: "cf-" + suffix + "@test.local", Username: "cf" + short})
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = ANY($1)`, pq.Array([]int64{author.ID, fan.ID}))
	require.NoError(t, err)
	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), nil)
	_, err = svc.SaveProfile(ctx, author.ID, service.ProfileInput{Handle: "cs" + short})
	require.NoError(t, err)
	_, err = svc.SaveProfile(ctx, fan.ID, service.ProfileInput{Handle: "cf" + short})
	require.NoError(t, err)
	hit, err := svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 40, 40, color.White)}, Title: "爆款", Visibility: "public"})
	require.NoError(t, err)
	quiet, err := svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 40, 40, color.Black)}, Title: "冷门", Visibility: "public"})
	require.NoError(t, err)

	// Today: 3 views and a remix by the fan, a like and a favorite, a follow; the author's own like doesn't count.
	for range 3 {
		_, err = svc.Work(ctx, hit.ID, fan.ID)
		require.NoError(t, err)
	}
	_, err = svc.Work(ctx, hit.ID, author.ID) // own views are not counted
	require.NoError(t, err)
	require.NoError(t, svc.Remix(ctx, fan.ID, hit.ID))
	_, err = svc.SetLike(ctx, fan.ID, hit.ID, true)
	require.NoError(t, err)
	_, err = svc.SetLike(ctx, author.ID, quiet.ID, true)
	require.NoError(t, err)
	_, err = svc.SetFavorite(ctx, fan.ID, hit.ID, true)
	require.NoError(t, err)
	_, err = svc.SetFollow(ctx, fan.ID, "cs"+short, true)
	require.NoError(t, err)
	// A like from 10 days ago lands in the previous 7-day period.
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO work_likes (work_id, user_id, created_at) VALUES ($1, $2, NOW() - INTERVAL '10 days')`, quiet.ID, fan.ID)
	require.NoError(t, err)

	stats, err := svc.CreatorStats(ctx, author.ID, 7)
	require.NoError(t, err)
	require.Equal(t, 7, stats.Days)
	require.Len(t, stats.Series, 7)
	today := stats.Series[6]
	require.Equal(t, time.Now().In(service.CreatorStatsLocation).Format(time.DateOnly), today.Day)
	require.Equal(t, service.CreatorCounts{Views: 3, Likes: 1, Favorites: 1, Remixes: 1, Followers: 1}, today.CreatorCounts)
	require.Equal(t, today.CreatorCounts, stats.Period)
	require.Equal(t, service.CreatorCounts{Likes: 1}, stats.Previous)
	require.Equal(t, 2, stats.Totals.Works)
	require.Equal(t, 2, stats.Totals.PublicWorks)
	require.Equal(t, 3, stats.Totals.Views)
	require.Equal(t, 1, stats.Totals.Followers)
	require.NotEmpty(t, stats.TrackedSince)
	require.Len(t, stats.TopWorks, 2)
	require.Equal(t, hit.ID, stats.TopWorks[0].Work.ID, "the work with remixes and favorites ranks first")
	require.Equal(t, 3, stats.TopWorks[0].PeriodViews)
	require.Equal(t, 1, stats.TopWorks[0].PeriodRemixes)
	require.NotEmpty(t, stats.TopWorks[0].Work.CoverThumbURL)

	// An unknown range falls back to 30 days.
	stats, err = svc.CreatorStats(ctx, author.ID, 12)
	require.NoError(t, err)
	require.Equal(t, 30, stats.Days)
	require.Len(t, stats.Series, 30)
	require.Equal(t, 2, stats.Period.Likes, "the 10-day-old like is inside 30 days")
}

func TestCommunitySiteWorks(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	author := mustCreateUser(t, integrationEntClient, &service.User{Email: "sw-" + suffix + "@test.local"})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "sw2-" + suffix + "@test.local"})
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = ANY($1)`, pq.Array([]int64{author.ID, other.ID}))
	require.NoError(t, err)
	newSite := func(userID int64, name, status string, version int) int64 {
		var id int64
		require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO sites (user_id, name, title, status, version) VALUES ($1, $2, $3, $4, $5) RETURNING id`, userID, name, "站点 "+name, status, version).Scan(&id))
		return id
	}
	live := newSite(author.ID, "live"+short, "active", 1)
	pending := newSite(author.ID, "pend"+short, "pending", 0)
	locked := newSite(author.ID, "lock"+short, "active", 1)
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET password_hash = 'x' WHERE id = $1`, locked)
	require.NoError(t, err)
	theirs := newSite(other.ID, "them"+short, "active", 1)

	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), nil)
	svc.SetSites(NewSiteHostingRepository(integrationDB), "s.example.test")
	_, err = svc.SaveProfile(ctx, author.ID, service.ProfileInput{Handle: "sw" + short})
	require.NoError(t, err)
	publish := func(siteID int64) (*service.Work, error) {
		return svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 64, 40, color.White)}, Title: "落地页", Visibility: "public", Source: "site", SiteID: siteID})
	}
	_, err = publish(theirs)
	require.ErrorIs(t, err, service.ErrCommunitySiteNotFound)
	_, err = publish(pending)
	require.ErrorIs(t, err, service.ErrCommunitySiteNotLive)
	_, err = publish(locked)
	require.ErrorIs(t, err, service.ErrCommunitySitePassword)

	work, err := publish(live)
	require.NoError(t, err)
	require.Equal(t, service.WorkKindSite, work.Kind)
	_, err = publish(live)
	require.ErrorIs(t, err, service.ErrCommunitySiteAlreadyPublished)
	image, err := svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 30, 30, color.Black)}, Title: "图片", Visibility: "public"})
	require.NoError(t, err)

	shown, err := svc.Work(ctx, work.ID, other.ID)
	require.NoError(t, err)
	require.Equal(t, "https://live"+short+".s.example.test", shown.Site.URL)
	require.Equal(t, "站点 live"+short, shown.Site.Title)

	ids := func(kind string) []int64 {
		works, err := svc.Works(ctx, service.WorkQuery{Feed: "user", UserID: author.ID, ViewerID: other.ID, Kind: kind})
		require.NoError(t, err)
		return workIDs(works)
	}
	require.Equal(t, []int64{work.ID}, ids("site"))
	require.Equal(t, []int64{image.ID}, ids("image"))
	require.ElementsMatch(t, []int64{work.ID, image.ID}, ids(""))

	// The site goes offline: the work disappears for others, the author still sees it.
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET status = 'disabled' WHERE id = $1`, live)
	require.NoError(t, err)
	require.Empty(t, ids("site"))
	_, err = svc.Work(ctx, work.ID, other.ID)
	require.ErrorIs(t, err, service.ErrCommunityWorkNotFound)
	mine, err := svc.Work(ctx, work.ID, author.ID)
	require.NoError(t, err)
	require.NotEmpty(t, mine.Site.URL, "the author keeps the link")
	out, err := svc.ShareMetaHTML(ctx, fmt.Sprintf("/w/%d", work.ID), "https://hivegpt.example.test", "")
	require.NoError(t, err)
	require.Empty(t, out, "no share card for an offline site")

	sites, err := svc.MySites(ctx, author.ID)
	require.NoError(t, err)
	reasons := map[int64]string{}
	for _, s := range sites {
		reasons[s.ID] = s.Reason
	}
	require.Equal(t, map[int64]string{live: "not_live", pending: "not_live", locked: "password"}, reasons)
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET status = 'active' WHERE id = $1`, live)
	require.NoError(t, err)
	sites, err = svc.MySites(ctx, author.ID)
	require.NoError(t, err)
	for _, s := range sites {
		if s.ID == live {
			require.Equal(t, "published", s.Reason)
			require.Equal(t, work.ID, s.WorkID)
		}
	}
}

func TestCommunityRecommendations(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	tag, liked := "rk"+short, "lk"+short // tags only this test uses keep the shared database's works out
	users := make([]*service.User, 3)
	for i := range users {
		users[i] = mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("rk%d-%s@test.local", i, suffix)})
	}
	prolific, other, viewer := users[0], users[1], users[2]
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = ANY($1)`, pq.Array([]int64{prolific.ID, other.ID, viewer.ID}))
	require.NoError(t, err)
	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), nil)
	for i, u := range users {
		_, err := svc.SaveProfile(ctx, u.ID, service.ProfileInput{Handle: fmt.Sprintf("rk%d%s", i, short)})
		require.NoError(t, err)
	}
	publish := func(u *service.User, likes int, model string, tags ...string) int64 {
		w, err := svc.Publish(ctx, u.ID, service.PublishInput{Images: [][]byte{testPNG(t, 32, 32, color.White)}, Title: "作品", Model: model, Tags: tags, Visibility: "public"})
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `UPDATE works SET like_count = $2, created_at = NOW() - INTERVAL '2 hours' WHERE id = $1`, w.ID, likes)
		require.NoError(t, err)
		return w.ID
	}
	a1 := publish(prolific, 10, "m1", tag)
	a2 := publish(prolific, 10, "m1", tag)
	a3 := publish(prolific, 10, "m1", tag, liked)
	b1 := publish(other, 7, "m2", tag)
	feed := func(viewerID int64) []int64 {
		works, err := svc.Works(ctx, service.WorkQuery{Feed: "recommended", Tag: tag, ViewerID: viewerID})
		require.NoError(t, err)
		return workIDs(works)
	}

	// Equal works by one author: each next one counts 0.6×, so the other author's lesser work comes second.
	require.Equal(t, []int64{a3, b1, a2, a1}, feed(0))

	// Followed authors rank higher for the follower only.
	_, err = svc.SetFollow(ctx, viewer.ID, fmt.Sprintf("rk1%s", short), true)
	require.NoError(t, err)
	require.Equal(t, b1, feed(viewer.ID)[0])
	require.Equal(t, a3, feed(0)[0])
	_, err = svc.SetFollow(ctx, viewer.ID, fmt.Sprintf("rk1%s", short), false)
	require.NoError(t, err)

	// Tags of works the viewer liked rank higher (a2 is liked outside this feed's tag, carrying "liked").
	elsewhere := publish(other, 0, "", liked)
	_, err = svc.SetLike(ctx, viewer.ID, elsewhere, true)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE works SET like_count = 9 WHERE id = $1`, a3)
	require.NoError(t, err)
	require.Equal(t, a2, feed(0)[0], "without taste the 10-like work leads")
	require.Equal(t, a3, feed(viewer.ID)[0], "the liked tag lifts the 9-like work")

	// Already liked works rank lower for the viewer (liking a2 makes it 11 likes).
	_, err = svc.SetLike(ctx, viewer.ID, a2, true)
	require.NoError(t, err)
	_, err = svc.SetLike(ctx, viewer.ID, elsewhere, false)
	require.NoError(t, err)
	require.Equal(t, a2, feed(0)[0])
	require.Equal(t, a1, feed(viewer.ID)[0])

	// A pinned feed time leaves out works published since, so later pages do not shift.
	pinned := time.Now().Add(-time.Minute)
	fresh := publish(other, 50, "", tag)
	_, err = integrationDB.ExecContext(ctx, `UPDATE works SET created_at = NOW() WHERE id = $1`, fresh)
	require.NoError(t, err)
	for _, name := range []string{"recommended", "latest"} {
		works, err := svc.Works(ctx, service.WorkQuery{Feed: name, Tag: tag, At: pinned})
		require.NoError(t, err)
		require.NotContains(t, workIDs(works), fresh, name)
		works, err = svc.Works(ctx, service.WorkQuery{Feed: name, Tag: tag})
		require.NoError(t, err)
		require.Contains(t, workIDs(works), fresh, name)
	}
	require.WithinDuration(t, time.Now(), service.FeedSnapshot(time.Now().Add(-7*time.Hour).Unix(), time.Now()), 2*time.Second, "stale feed times fall back to now")
	require.WithinDuration(t, pinned, service.FeedSnapshot(pinned.Unix(), time.Now()), time.Second)

	// Related: other authors only; shared tags first, then the same model.
	base := publish(prolific, 0, "m9", tag, liked)
	twoTags := publish(other, 0, "", tag, liked)
	sameModel := publish(other, 0, "m9", tag)
	related, err := svc.RelatedWorks(ctx, base, viewer.ID, 24)
	require.NoError(t, err)
	ids := workIDs(related)
	require.GreaterOrEqual(t, len(ids), 3)
	require.Equal(t, []int64{twoTags, sameModel}, ids[:2])
	for _, id := range []int64{base, a1, a2, a3} {
		require.NotContains(t, ids, id, "the author's own works are listed separately")
	}
}

func TestCommunityComments(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-7:]
	users := make([]*service.User, 3)
	for i := range users {
		users[i] = mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("cm%d-%s@test.local", i, suffix)})
	}
	author, bob, cara := users[0], users[1], users[2]
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = ANY($1)`, pq.Array([]int64{author.ID, bob.ID, cara.ID}))
	require.NoError(t, err)
	settings := NewSettingRepository(integrationEntClient)
	svc := service.NewCommunityService(NewCommunityRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), settings)
	t.Cleanup(func() {
		_ = svc.AdminSaveSettings(ctx, false, &service.CommentSettings{CommentsEnabled: true}, nil)
	})
	_, err = svc.SaveProfile(ctx, author.ID, service.ProfileInput{Handle: "cm0" + short})
	require.NoError(t, err)
	work, err := svc.Publish(ctx, author.ID, service.PublishInput{Images: [][]byte{testPNG(t, 32, 32, color.White)}, Title: "作品", Visibility: "public"})
	require.NoError(t, err)

	// A profile is needed to comment.
	_, err = svc.AddComment(ctx, bob.ID, work.ID, service.CommentInput{Body: "好看"}, "10.0.0.1")
	require.ErrorIs(t, err, service.ErrCommunityProfileRequired)
	for i, u := range users[1:] {
		i++
		_, err := svc.SaveProfile(ctx, u.ID, service.ProfileInput{Handle: fmt.Sprintf("cm%d%s", i, short)})
		require.NoError(t, err)
	}
	add := func(u *service.User, body string, replyTo int64) (*service.WorkComment, error) {
		return svc.AddComment(ctx, u.ID, work.ID, service.CommentInput{Body: body, ReplyTo: replyTo}, "10.0.0.1")
	}
	_, err = add(bob, "   ", 0)
	require.ErrorIs(t, err, service.ErrCommunityCommentEmpty)
	_, err = add(bob, strings.Repeat("长", 501), 0)
	require.ErrorIs(t, err, service.ErrCommunityCommentEmpty)

	top, err := add(bob, "好看", 0)
	require.NoError(t, err)
	require.Equal(t, service.CommentStatusApproved, top.Status)
	require.True(t, top.IsMine)
	require.True(t, top.CanDelete)
	_, err = add(bob, "好看", 0)
	require.ErrorIs(t, err, service.ErrCommunityCommentDuplicate)
	reply, err := add(cara, "同感", top.ID)
	require.NoError(t, err)
	require.Equal(t, top.ID, reply.ParentID)
	answer, err := add(bob, "谢谢", reply.ID)
	require.NoError(t, err)
	require.Equal(t, top.ID, answer.ParentID, "replies stay one level deep")

	page, err := svc.Comments(ctx, work.ID, 0, 0)
	require.NoError(t, err)
	require.True(t, page.Enabled)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Comments, 1)
	require.Equal(t, 2, page.Comments[0].ReplyCount)
	require.Len(t, page.Comments[0].Replies, 2)
	require.Nil(t, page.Comments[0].Replies[0].ReplyTo, "a direct reply needs no 回复 @")
	require.NotNil(t, page.Comments[0].Replies[1].ReplyTo)
	require.Equal(t, fmt.Sprintf("cm2%s", short), page.Comments[0].Replies[1].ReplyTo.Handle)
	require.False(t, page.Comments[0].CanDelete, "visitors cannot delete")

	// Notifications: the author for each comment, the answered commenter for replies.
	kinds := func(u *service.User) []string {
		list, err := svc.Notifications(ctx, u.ID)
		require.NoError(t, err)
		var out []string
		for _, n := range list {
			if n.WorkID == work.ID {
				out = append(out, n.Kind)
			}
		}
		return out
	}
	require.ElementsMatch(t, []string{"comment", "comment"}, kinds(author), "bob's two comments collapse while unread")
	require.ElementsMatch(t, []string{"reply"}, kinds(bob))
	require.ElementsMatch(t, []string{"reply"}, kinds(cara))

	// Flagged words wait for review: only the commenter sees it.
	flagged, err := add(cara, "nsfw 版本有吗", 0)
	require.NoError(t, err)
	require.Equal(t, service.CommentStatusPending, flagged.Status)
	page, err = svc.Comments(ctx, work.ID, 0, 0)
	require.NoError(t, err)
	require.Len(t, page.Comments, 1)
	require.Equal(t, 3, page.Total)
	page, err = svc.Comments(ctx, work.ID, cara.ID, 0)
	require.NoError(t, err)
	require.Len(t, page.Comments, 2)
	require.Equal(t, service.CommentStatusPending, page.Comments[0].Status)

	// Deleting: others cannot; the work's author can. A removed comment with replies stays as a placeholder.
	require.ErrorIs(t, svc.DeleteComment(ctx, cara.ID, answer.ID), service.ErrCommunityCommentNotAllowed)
	require.NoError(t, svc.DeleteComment(ctx, author.ID, top.ID))
	page, err = svc.Comments(ctx, work.ID, 0, 0)
	require.NoError(t, err)
	require.Len(t, page.Comments, 1)
	require.Equal(t, service.CommentStatusRemoved, page.Comments[0].Status)
	require.Empty(t, page.Comments[0].Body)
	require.Nil(t, page.Comments[0].Author)
	require.Len(t, page.Comments[0].Replies, 2)
	require.Equal(t, 2, page.Total)
	_, err = add(cara, "还在吗", top.ID)
	require.ErrorIs(t, err, service.ErrCommunityCommentNotFound, "no replies to a removed comment")
	require.NoError(t, svc.DeleteComment(ctx, bob.ID, answer.ID))
	replies, more, err := svc.CommentReplies(ctx, top.ID, 0, 0)
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, []int64{reply.ID}, []int64{replies[0].ID})

	// Reports reach the admin queue with the comment.
	require.NoError(t, svc.ReportComment(ctx, author.ID, reply.ID, "spam", "", "10.0.0.9"))
	reported, err := svc.AdminComments(ctx, "reported", 1)
	require.NoError(t, err)
	require.Contains(t, commentIDs(reported), reply.ID)
	reports, err := svc.AdminReports(ctx, "open", 1)
	require.NoError(t, err)
	var found bool
	for _, rp := range reports {
		if rp.CommentID == reply.ID {
			found = rp.CommentBody == "同感" && rp.WorkID == work.ID
		}
	}
	require.True(t, found)

	// Moderation: approving the flagged comment publishes it; hiding tells the commenter.
	pending, err := svc.AdminComments(ctx, "pending", 1)
	require.NoError(t, err)
	require.Contains(t, commentIDs(pending), flagged.ID)
	require.NoError(t, svc.AdminModerateComment(ctx, flagged.ID, "approve"))
	require.NoError(t, svc.AdminModerateComment(ctx, reply.ID, "hide"))
	require.Contains(t, kinds(cara), "comment_hidden")
	page, err = svc.Comments(ctx, work.ID, 0, 0)
	require.NoError(t, err)
	require.Equal(t, []int64{flagged.ID}, commentIDs(page.Comments), "the placeholder goes once no replies remain")
	require.Equal(t, 1, page.Total)

	// The author closes comments; the admin can review all or turn comments off.
	closed := true
	_, err = svc.UpdateWork(ctx, author.ID, work.ID, service.UpdateWorkInput{Title: "作品", Visibility: "public", CommentsClosed: &closed})
	require.NoError(t, err)
	_, err = add(bob, "关了吗", 0)
	require.ErrorIs(t, err, service.ErrCommunityCommentsClosed)
	closed = false
	_, err = svc.UpdateWork(ctx, author.ID, work.ID, service.UpdateWorkInput{Title: "作品", Visibility: "public", CommentsClosed: &closed})
	require.NoError(t, err)
	require.NoError(t, svc.AdminSaveSettings(ctx, false, &service.CommentSettings{CommentsEnabled: true, CommentsReviewAll: true}, nil))
	held, err := add(bob, "先审后发", 0)
	require.NoError(t, err)
	require.Equal(t, service.CommentStatusPending, held.Status)
	require.NoError(t, svc.AdminSaveSettings(ctx, false, &service.CommentSettings{CommentsEnabled: false}, nil))
	_, err = add(bob, "关闭后", 0)
	require.ErrorIs(t, err, service.ErrCommunityCommentsOff)
	page, err = svc.Comments(ctx, work.ID, 0, 0)
	require.NoError(t, err)
	require.False(t, page.Enabled)
	require.Empty(t, page.Comments)
}

func commentIDs(list []service.WorkComment) []int64 {
	out := make([]int64, 0, len(list))
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}
