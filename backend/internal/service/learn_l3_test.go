//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLearnShowcase(t *testing.T) {
	ctx := context.Background()
	works := &learnWorksStub{handle: "mia"}
	svc, repo := newL2Service(t, "http://127.0.0.1:1", LearnSources{Works: works, CanvasURL: "https://canvas.example.test"})
	now := time.Now()
	repo.certs = []LearnCertificate{
		{Code: "AAAAAAAAAA", UserID: 1, Track: "a", DisplayName: "小林", ProjectURL: "https://github.com/x/y", IssuedAt: now, Showcase: true},
		{Code: "BBBBBBBBBB", UserID: 2, Track: "b", DisplayName: "阿明", IssuedAt: now, Showcase: false},
		{Code: "CCCCCCCCCC", UserID: 3, Track: "d", DisplayName: "小红", IssuedAt: now, Showcase: true},
	}
	works.works = []Work{
		{ID: 5, Kind: "image", Title: "猫", CoverThumb: "t5.webp", Status: WorkStatusApproved, Visibility: WorkVisibilityPublic},
		{ID: 6, Kind: "site", Title: "我的机器人", CoverFile: "c6.webp", Status: WorkStatusApproved, Visibility: WorkVisibilityPublic},
		{ID: 7, Kind: "image", CoverThumb: "t7.webp", Status: WorkStatusPending, Visibility: WorkVisibilityPublic},
		{ID: 8, Kind: "image", CoverThumb: "t8.webp", Status: WorkStatusApproved, Visibility: WorkVisibilityPrivate},
	}

	items, err := svc.Showcase(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, items, 2) // only opted-in certificates
	require.Equal(t, "CCCCCCCCCC", items[0].Code)
	require.Empty(t, items[0].Works) // interview track: nothing to show
	a := items[1]
	require.Equal(t, "AI 应用开发入门", a.TrackTitle)
	require.Len(t, a.Works, 2) // public approved works only, the site first for track A
	require.Equal(t, "site", a.Works[0].Kind)
	require.Equal(t, "https://canvas.example.test/w/6", a.Works[0].URL)
	require.Contains(t, a.Works[0].Image, "c6.webp")

	// Opting in / out and an admin hiding an entry clear the cache.
	require.NoError(t, svc.SetShowcase(ctx, 2, "b", true))
	items, _ = svc.Showcase(ctx, "", 0)
	require.Len(t, items, 3)
	require.NoError(t, svc.SetShowcaseHidden(ctx, "aaaaaaaaaa", true))
	items, _ = svc.Showcase(ctx, "", 0)
	require.Len(t, items, 2)
	items, _ = svc.Showcase(ctx, "b", 0)
	require.Len(t, items, 1)
	_, err = svc.Showcase(ctx, "z", 0)
	require.Error(t, err)
	require.Error(t, svc.SetShowcase(ctx, 9, "a", true))

	// Claiming with showcase on puts the new certificate on the wall.
	repo.certs = nil
	svc.showcase.clear()
	for _, id := range svc.catalog.track("d").Lessons {
		repo.done[id] = now
	}
	for _, topic := range svc.catalog.InterviewTopics {
		repo.interviews = append(repo.interviews, LearnInterview{ID: int64(len(repo.interviews) + 1), UserID: 4, Topic: topic, Status: learnInterviewStatusEnd, Score: 80})
	}
	cert, err := svc.ClaimCertificate(ctx, 4, "d", "小周", "", true)
	require.NoError(t, err)
	require.True(t, cert.Showcase)
	items, _ = svc.Showcase(ctx, "d", 0)
	require.Len(t, items, 1)
}

func TestLearnInsights(t *testing.T) {
	ctx := context.Background()
	svc, repo := newL2Service(t, "http://127.0.0.1:1", LearnSources{})
	for _, id := range []string{"a1", "a2", "a3", "a4", "b1"} {
		repo.done[id] = time.Now()
	}
	repo.certs = []LearnCertificate{{Code: "X", Track: "b"}}
	in, err := svc.Insights(ctx)
	require.NoError(t, err)
	require.Len(t, in.Funnels, 4)
	a := in.Funnels[0]
	require.Equal(t, 8, a.Lessons)
	require.Equal(t, 1, a.Started)
	require.Equal(t, 1, a.Half) // 4 of 8
	require.Equal(t, 0, a.Finished)
	require.Equal(t, 1, in.Funnels[1].Certificates)
	require.Len(t, in.Days, learnInsightDays) // gaps filled
	require.Equal(t, 3, in.Days[len(in.Days)-1].Runs)
	require.Equal(t, 100, in.Quizzes["a1"].AvgScore)
}
