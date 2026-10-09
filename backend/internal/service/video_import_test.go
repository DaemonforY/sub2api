//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoImportRecordsNarrationAndPublishes(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, tts := newTestVideoService(t, model)
	ctx := context.Background()
	in := VideoImportInput{Mode: "film", Title: "光的折射", Prompt: "讲讲光的折射", Featured: true,
		Options: VideoOptions{Style: "chalkboard", Category: "science", Model: "secret-model"},
		Spec: VideoSpec{Scenes: []VideoScene{
			{Title: "开场", Narration: "筷子插进水里，看起来像折断了。", Code: goodScene},
			{Title: "原理", Narration: "光从空气进入水中会改变方向。", Code: goodScene},
		}}}

	_, err := svc.Import(ctx, videoKeyA, in)
	require.ErrorIs(t, err, ErrVideoForbidden, "only admins import")

	admin := *videoKeyA
	admin.User = &User{ID: admin.UserID, Role: RoleAdmin}
	bad := in
	bad.Spec.Scenes = []VideoScene{{Code: "return function(t) {"}}
	_, err = svc.Import(ctx, &admin, bad)
	require.Error(t, err, "scene code is syntax-checked")

	p, err := svc.Import(ctx, &admin, in)
	require.NoError(t, err)
	require.Equal(t, VideoStatusReady, p.Status)
	require.Equal(t, VideoVisibilityPublic, p.Visibility)
	require.True(t, p.Featured)
	require.Equal(t, "science", p.Category)
	require.Equal(t, 2, tts.calls)
	require.Empty(t, model.calls, "no model calls")
	require.Equal(t, "s1", p.Spec.Scenes[0].ID)
	require.NotNil(t, p.Spec.Scenes[1].Audio)
	require.InDelta(t, p.Spec.TotalDuration(), p.Duration, 0.01)

	w, err := svc.Work(ctx, p.ID)
	require.NoError(t, err)
	require.Empty(t, w.Options.Model, "the model is not shown on public works")
}
