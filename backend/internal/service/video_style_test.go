//go:build unit

package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeStyleRepo adds 我的风格 cards to the fake project store.
type fakeStyleRepo struct {
	*fakeVideoRepo
	smu    sync.Mutex
	styles map[int64]*VideoStyleCard
	next   int64
}

func (r *fakeStyleRepo) ListStyles(_ context.Context, userID int64) ([]VideoStyleCard, error) {
	r.smu.Lock()
	defer r.smu.Unlock()
	out := []VideoStyleCard{}
	for _, st := range r.styles {
		if st.UserID == userID {
			out = append(out, *st)
		}
	}
	return out, nil
}

func (r *fakeStyleRepo) GetStyle(_ context.Context, userID, id int64) (*VideoStyleCard, error) {
	r.smu.Lock()
	defer r.smu.Unlock()
	if st := r.styles[id]; st != nil && st.UserID == userID {
		cp := *st
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeStyleRepo) CreateStyle(_ context.Context, st *VideoStyleCard) error {
	r.smu.Lock()
	defer r.smu.Unlock()
	r.next++
	st.ID, st.CreatedAt, st.UpdatedAt = r.next, time.Now(), time.Now()
	cp := *st
	r.styles[st.ID] = &cp
	return nil
}

func (r *fakeStyleRepo) UpdateStyle(_ context.Context, st *VideoStyleCard) error {
	r.smu.Lock()
	defer r.smu.Unlock()
	cp := *st
	r.styles[st.ID] = &cp
	return nil
}

func (r *fakeStyleRepo) DeleteStyle(_ context.Context, userID, id int64) (bool, error) {
	r.smu.Lock()
	defer r.smu.Unlock()
	if st := r.styles[id]; st != nil && st.UserID == userID {
		delete(r.styles, id)
		return true, nil
	}
	return false, nil
}

func newStyleTestService(t *testing.T, model *fakeModel) (*VideoService, *fakeStyleRepo, *fakeTTS) {
	repo := &fakeStyleRepo{fakeVideoRepo: newFakeVideoRepo(), styles: map[int64]*VideoStyleCard{}}
	tts := &fakeTTS{}
	return newVideoService(repo, tts, model.stream, t.TempDir()), repo, tts
}

func TestVideoScriptConfirmStopsBeforeVoiceAndCode(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, repo, tts := newStyleTestService(t, model)
	ctx := context.Background()

	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Confirm: true, ScriptOK: true}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusScript, got.Status, got.Error)
	require.False(t, got.Options.ScriptOK, "the client cannot pre-approve")
	require.Len(t, got.Spec.Scenes, 2)
	for _, sc := range got.Spec.Scenes {
		require.Empty(t, sc.Code)
		require.Nil(t, sc.Audio)
	}
	require.Zero(t, tts.calls)
	require.Equal(t, []string{"script"}, model.calls)
	events, _ := repo.ListEvents(ctx, p.ID, 0, 100)
	require.Equal(t, "script", events[len(events)-1].Kind)

	// While the script waits, chat, resume and the normal script editor are refused.
	_, err = svc.Message(ctx, videoKeyA, p.ID, "改一下", "")
	require.ErrorIs(t, err, ErrVideoScriptWait)
	_, err = svc.Resume(ctx, videoKeyA, p.ID)
	require.ErrorIs(t, err, ErrVideoScriptWait)
	_, err = svc.EditScript(ctx, videoKeyA, p.ID, []VideoSceneEdit{{ID: "s1", Narration: "x"}})
	require.ErrorIs(t, err, ErrVideoScriptWait)

	_, err = svc.ApproveScript(ctx, videoKeyA, p.ID, []VideoSceneEdit{{ID: "s1", Narration: "改过的第一段旁白，长一点。", Hook: "小机器人把账本撕成两半", Camera: "推近"}})
	require.NoError(t, err)
	svc.Wait()
	got, _ = svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.True(t, got.Options.ScriptOK)
	require.Equal(t, "改过的第一段旁白，长一点。", got.Spec.Scenes[0].Narration)
	require.Equal(t, "小机器人把账本撕成两半", got.Spec.Scenes[0].Hook)
	require.Equal(t, 2, tts.calls)
	for _, sc := range got.Spec.Scenes {
		require.NotEmpty(t, sc.Code)
	}
	// The hook reaches the scene's designer.
	joined := strings.Join(model.calls, ",")
	require.Equal(t, 1, strings.Count(joined, "script"), "approving does not rewrite the script")

	_, err = svc.ApproveScript(ctx, videoKeyA, p.ID, nil)
	require.ErrorIs(t, err, ErrVideoNoScriptWait)
}

func TestVideoStyleCardCarriesTheSeriesLook(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, repo, _ := newStyleTestService(t, model)
	ctx := context.Background()

	first, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "第一期", Options: VideoOptions{Ratio: "9:16", Style: "tech-blogger"}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, first.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)

	// Another user cannot save it; a motion has no kit.
	_, err = svc.SaveStyle(ctx, 8, first.ID, "x", nil)
	require.ErrorIs(t, err, ErrVideoNotFound)
	card, err := svc.SaveStyle(ctx, 7, first.ID, "  我的科技频道  ", []string{"正文不小于 28px", "正文不小于 28px", " "})
	require.NoError(t, err)
	require.Equal(t, "我的科技频道", card.Name)
	require.Equal(t, "9:16", card.Ratio)
	require.Equal(t, goodKit, card.Common)
	require.Equal(t, "橙色圆形小机器人", card.Character)
	require.Equal(t, []string{"正文不小于 28px"}, card.Rules)

	model.kitAsks = nil
	second, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "第二期", Options: VideoOptions{Ratio: "16:9", Style: "auto", MyStyle: card.ID}})
	require.NoError(t, err)
	require.Equal(t, "9:16", second.Options.Ratio, "the card fixes the ratio")
	require.Equal(t, "tech-blogger", second.Options.Style)
	require.Equal(t, "我的科技频道", second.Options.MyStyleName)
	svc.Wait()
	got, _ = svc.Get(ctx, 7, second.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, []string{"正文不小于 28px"}, got.Spec.Rules)
	require.Equal(t, findVideoStyle("tech-blogger").Theme, got.Spec.Theme)
	require.Len(t, model.kitAsks, 1)
	require.Contains(t, model.kitAsks[0], "NEW video of the same series", "the saved kit is adapted, not redesigned")
	require.Contains(t, model.scenes[len(model.scenes)-1], "正文不小于 28px", "scenes follow the house rules")

	// 复盘 proposes rules; the user keeps them on the card.
	retro, err := svc.Retro(ctx, videoKeyA, second.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"正文不小于 28px"}, retro.Rules)
	require.Contains(t, model.retroAsks[0], "已有规则：\n- 正文不小于 28px")
	rules := append(card.Rules, "每段至少换一次构图")
	updated, err := svc.UpdateStyle(ctx, 7, card.ID, nil, rules)
	require.NoError(t, err)
	require.Equal(t, []string{"正文不小于 28px", "每段至少换一次构图"}, updated.Rules)
	require.Equal(t, "我的科技频道", updated.Name)

	// A deleted card cannot be picked; the user's list is empty again.
	require.NoError(t, svc.DeleteStyle(ctx, 7, card.ID))
	require.ErrorIs(t, svc.DeleteStyle(ctx, 7, card.ID), ErrVideoStyleNotFound)
	_, err = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "第三期", Options: VideoOptions{MyStyle: card.ID}})
	require.ErrorIs(t, err, ErrVideoStyleNotFound)
	items, _ := svc.ListStyles(ctx, 7)
	require.Empty(t, items)
	_ = repo
}

func TestCleanVideoRules(t *testing.T) {
	var many []string
	for i := 0; i < 15; i++ {
		many = append(many, strings.Repeat("规", i+1))
	}
	got := cleanVideoRules(append([]string{"  a   b ", "a b", ""}, many...))
	require.Len(t, got, videoStyleRulesMax)
	require.Equal(t, strings.Repeat("规", 15), got[len(got)-1], "the newest rules are kept")
	require.Equal(t, []string{}, cleanVideoRules(nil))
}
