//go:build unit

package service

import (
	"bytes"
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// swapModel writes a three-scene script around img1 / img2 and answers the 换图 plan.
type swapModel struct {
	mu          sync.Mutex
	planImages  int
	planUser    string
	changes     map[string]string // scene → the "Change this scene" message
	sceneImages map[string]int    // scene → pictures attached to its last code call
	plan        string
}

var swapSceneRe = regexp.MustCompile(`Scene (s\d+)`)

func (m *swapModel) stream(_ context.Context, _, _ string, msgs []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
	sys, last := msgs[0].Content, msgs[len(msgs)-1]
	var ans string
	m.mu.Lock()
	switch {
	case strings.Contains(sys, "scriptwriter"):
		ans = `{"title":"杯子","scenes":[` +
			`{"title":"一","narration":"第一段旁白。","images":["img1"],"visual":"img1 放大"},` +
			`{"title":"二","narration":"第二段旁白。","images":["img2"],"visual":"img2 居中"},` +
			`{"title":"三","narration":"第三段旁白。","images":[],"visual":"文字"}],` +
			`"image_notes":[{"id":"img1","desc":"白色杯子"},{"id":"img2","desc":"红色盒子"}]}`
	case strings.Contains(sys, "shared visual system"):
		ans = "```js\n" + goodKit + "\n```"
	case strings.Contains(sys, "changed the pictures"):
		m.planImages, m.planUser = len(last.Images), last.Content
		ans = m.plan
	default:
		scene := swapSceneRe.FindStringSubmatch(msgs[1].Content)[1]
		if m.sceneImages == nil {
			m.sceneImages, m.changes = map[string]int{}, map[string]string{}
		}
		m.sceneImages[scene] = len(msgs[1].Images)
		if strings.HasPrefix(last.Content, "Change this scene") {
			m.changes[scene] = last.Content
		}
		ans = "```js\n" + goodScene + "\n```"
	}
	m.mu.Unlock()
	if err := onDelta(ans); err != nil {
		return nil, err
	}
	return &LearnTutorResult{PromptTokens: 10, CompletionTokens: 5}, nil
}

func stagePicture(t *testing.T, svc *VideoService, w, h int) string {
	t.Helper()
	pic := testPicture(t, w, h, false, "jpeg")
	st, err := svc.StageImage(context.Background(), videoKeyA, VideoUploadFile{Name: "a.jpg", Size: int64(len(pic)), Reader: bytes.NewReader(pic)})
	require.NoError(t, err)
	return st.Token
}

func TestVideoChangeImages(t *testing.T) {
	model := &swapModel{plan: `{"reply":"好的，新盒子放进了第三段。","image_notes":[{"id":"img1","desc":"蓝色新杯子"},{"id":"img3","desc":"绿色盒子"}],` +
		`"scenes":[{"id":"s3","images":["img3","img9"],"visual":"img3 居中，右侧标注","instruction":"放上绿色盒子"}]}`}
	tts := &fakeTTS{}
	repo := newFakeVideoRepo()
	dir := t.TempDir()
	svc := newVideoService(repo, tts, model.stream, dir)
	ctx := context.Background()

	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "介绍杯子", Images: []string{stagePicture(t, svc, 400, 300), stagePicture(t, svc, 300, 400)}})
	require.NoError(t, err)
	svc.Wait()
	before, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, before.Status, before.Error)
	ttsCalls := tts.calls

	fresh, added := stagePicture(t, svc, 500, 300), stagePicture(t, svc, 320, 240)
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{})
	require.ErrorIs(t, err, ErrVideoImageNoChange)
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Replace: map[string]string{"img7": fresh}})
	require.ErrorIs(t, err, ErrVideoImageUnknown)
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Replace: map[string]string{"img2": fresh}, Remove: []string{"img2"}})
	require.ErrorIs(t, err, ErrVideoImageUnknown)
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Add: make([]string, VideoImagesMax-1)})
	require.ErrorIs(t, err, ErrVideoImageCount)
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Add: []string{"../../etc/passwd"}})
	require.ErrorIs(t, err, ErrVideoImageGone)
	// The same file again is not a change.
	sameToken := strings.TrimPrefix(before.Options.Images[0].File, "img-")
	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Replace: map[string]string{"img1": sameToken}})
	require.ErrorIs(t, err, ErrVideoImageNoChange)
	_, err = svc.ChangeImages(ctx, &APIKey{ID: 2, UserID: 8, Key: "sk-b"}, p.ID, VideoImagesChange{Remove: []string{"img1"}})
	require.ErrorIs(t, err, ErrVideoNotFound)

	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Replace: map[string]string{"img1": fresh}, Add: []string{added}, Remove: []string{"img2"}, Note: "新盒子放第三段"})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)

	// Pictures: img1 is the new file, img2 is gone, the added one is img3.
	require.Len(t, got.Options.Images, 2)
	require.Equal(t, "img1", got.Options.Images[0].ID)
	require.Equal(t, "img-"+fresh, got.Options.Images[0].File)
	require.Equal(t, 500, got.Options.Images[0].Width)
	require.Equal(t, "img3", got.Options.Images[1].ID)
	require.Equal(t, "蓝色新杯子", got.Spec.Images[0].Desc)
	require.Equal(t, "绿色盒子", got.Spec.Images[1].Desc)

	// The director saw the two new pictures and the user's note.
	require.Equal(t, 2, model.planImages)
	require.Contains(t, model.planUser, "img2: REMOVED")
	require.Contains(t, model.planUser, "新盒子放第三段")

	// s1 (replaced picture) and s2 (removed picture) are rebuilt although the plan left them out;
	// s3 gets the new picture as planned. The narration is not re-recorded.
	require.Len(t, model.changes, 3)
	require.Contains(t, model.changes["s1"], "img1 已换成另一张图片")
	require.Contains(t, model.changes["s2"], "图片 img2 已被删除")
	require.Contains(t, model.changes["s3"], "放上绿色盒子")
	require.Equal(t, 1, model.sceneImages["s1"], "the designer sees the new picture")
	require.Equal(t, 1, model.sceneImages["s3"])
	require.Equal(t, []string{"img1"}, got.Spec.Scenes[0].Images)
	require.Empty(t, got.Spec.Scenes[1].Images)
	require.Equal(t, []string{"img3"}, got.Spec.Scenes[2].Images, "unknown ids are dropped")
	require.Equal(t, "img3 居中，右侧标注", got.Spec.Scenes[2].Visual)
	require.Equal(t, ttsCalls, tts.calls)
	for i := range got.Spec.Scenes {
		require.Equal(t, before.Spec.Scenes[i].Audio.File, got.Spec.Scenes[i].Audio.File)
	}

	// The new picture is served; restoring 换图前 brings the old pictures (still on disk) back.
	exp, sig := linkQuery(t, got.Spec.Images[1].URL)
	_, err = svc.MediaFile(ctx, p.ID, got.Spec.Images[1].File, exp, sig)
	require.NoError(t, err)
	vs, _ := svc.Versions(ctx, 7, p.ID)
	sort.Slice(vs, func(i, j int) bool { return vs[i].ID > vs[j].ID })
	require.Equal(t, "换图前", vs[0].Note)
	_, err = svc.Restore(ctx, 7, p.ID, vs[0].ID)
	require.NoError(t, err)
	restored, _ := svc.Get(ctx, 7, p.ID)
	require.Len(t, restored.Options.Images, 2)
	require.Equal(t, before.Options.Images[0].File, restored.Options.Images[0].File)
	require.Equal(t, "img2", restored.Options.Images[1].ID)
	exp, sig = linkQuery(t, restored.Spec.Images[1].URL)
	_, err = svc.MediaFile(ctx, p.ID, restored.Spec.Images[1].File, exp, sig)
	require.NoError(t, err)
}

func TestVideoChangeImagesWithoutAPlan(t *testing.T) {
	// An unreadable plan still rebuilds the scenes whose picture was replaced.
	model := &swapModel{plan: "抱歉"}
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, model.stream, t.TempDir())
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "介绍杯子", Images: []string{stagePicture(t, svc, 400, 300), stagePicture(t, svc, 300, 400)}})
	require.NoError(t, err)
	svc.Wait()

	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Replace: map[string]string{"img2": stagePicture(t, svc, 640, 360)}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, []string{"s2"}, keysOf(model.changes))
	require.Empty(t, got.Spec.Images[1].Desc, "the old description no longer fits")
	require.Equal(t, "白色杯子", got.Spec.Images[0].Desc)
}

func TestVideoChangeImagesMotion(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "杯子转一圈", Images: []string{stagePicture(t, svc, 400, 300)}})
	require.NoError(t, err)
	svc.Wait()

	_, err = svc.ChangeImages(ctx, videoKeyA, p.ID, VideoImagesChange{Add: []string{stagePicture(t, svc, 300, 300)}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, []string{"img1", "img2"}, got.Spec.Scenes[0].Images)
	require.Equal(t, []string{"s1"}, model.revisions)
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
