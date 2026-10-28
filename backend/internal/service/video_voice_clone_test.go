//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeWAV is seconds of 16 kHz mono 16-bit silence.
func fakeWAV(seconds float64) []byte {
	n := int(seconds * 16000 * 2)
	b := make([]byte, 44+n)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+n))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 16000)
	binary.LittleEndian.PutUint32(b[28:], 32000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(n))
	return b
}

type fakeCloneRepo struct {
	mu      sync.Mutex
	next    int64
	rows    map[int64]*VideoVoiceClone
	deleted map[int64]bool
	charged []float64
}

func newFakeCloneRepo() *fakeCloneRepo {
	return &fakeCloneRepo{rows: map[int64]*VideoVoiceClone{}, deleted: map[int64]bool{}}
}

func (f *fakeCloneRepo) ClaimClone(_ context.Context, c *VideoVoiceClone, slots []string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range slots {
		taken := false
		for id, r := range f.rows {
			if r.SpeakerID == s && !f.deleted[id] {
				taken = true
			}
		}
		if !taken {
			f.next++
			c.ID, c.SpeakerID, c.CreatedAt = f.next, s, time.Now()
			cp := *c
			f.rows[c.ID] = &cp
			return nil
		}
	}
	return ErrVideoCloneFull
}

func (f *fakeCloneRepo) ListClones(_ context.Context, userID int64) ([]VideoVoiceClone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []VideoVoiceClone{}
	for id := int64(1); id <= f.next; id++ {
		if r, ok := f.rows[id]; ok && r.UserID == userID && !f.deleted[id] {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (f *fakeCloneRepo) GetClone(_ context.Context, id int64) (*VideoVoiceClone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok || f.deleted[id] {
		return nil, ErrVideoCloneNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeCloneRepo) UpdateClone(_ context.Context, c *VideoVoiceClone) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.rows[c.ID]
	r.Name, r.Trainings, r.CheckedAt = c.Name, c.Trainings, c.CheckedAt
	if r.Status != VideoCloneDisabled {
		r.Status, r.Error = c.Status, c.Error
	}
	return nil
}

func (f *fakeCloneRepo) DeleteClone(_ context.Context, userID, id int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok || r.UserID != userID || f.deleted[id] {
		return false, nil
	}
	f.deleted[id] = true
	return true, nil
}

func (f *fakeCloneRepo) MarkCloneReady(_ context.Context, id int64) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.rows[id]
	if r.Status != VideoCloneTraining || f.deleted[id] {
		return 0, nil
	}
	r.Status, r.Error = VideoCloneReady, ""
	if r.Charged {
		return 0, nil
	}
	r.Charged = true
	f.charged = append(f.charged, r.Price)
	return r.Price, nil
}

func (f *fakeCloneRepo) AdminListClones(context.Context, int) ([]VideoAdminClone, error) {
	return nil, nil
}

func (f *fakeCloneRepo) SetCloneStatus(_ context.Context, id int64, status, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[id].Status, f.rows[id].Error = status, message
	return nil
}

type fakeTrainer struct {
	mu        sync.Mutex
	trained   []string
	readyNow  bool
	status    map[string]string // "" training, "ready", or a failure reason
	trainErr  error
	statusErr error
}

func (f *fakeTrainer) Train(_ context.Context, speakerID string, _ []byte, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.trainErr != nil {
		return false, f.trainErr
	}
	f.trained = append(f.trained, speakerID)
	return f.readyNow, nil
}

func (f *fakeTrainer) Status(_ context.Context, speakerID string) (bool, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.status[speakerID]
	return s == "ready", map[bool]string{true: "", false: s}[s == "ready" || s == ""], f.statusErr
}

type voiceCapturingTTS struct {
	mu     sync.Mutex
	voices []string
}

func (f *voiceCapturingTTS) Synthesize(_ context.Context, text, voice string, _ int, _ string) (*VideoSpeech, error) {
	f.mu.Lock()
	f.voices = append(f.voices, voice)
	f.mu.Unlock()
	return &VideoSpeech{Audio: []byte("paid:" + text), Duration: 2}, nil
}

func setVideoSettings(svc *VideoService, st VideoSettings) {
	svc.settingsCache.mu.Lock()
	svc.settingsCache.val, svc.settingsCache.at = &st, time.Now()
	svc.settingsCache.mu.Unlock()
}

func newCloneTestService(t *testing.T) (*VideoService, *fakeCloneRepo, *fakeTrainer, *voiceCapturingTTS, *fakeTTSBilling) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	repo, trainer, tts, billing := newFakeCloneRepo(), &fakeTrainer{status: map[string]string{}}, &voiceCapturingTTS{}, &fakeTTSBilling{balance: 100}
	svc.SetPaidTTS(tts, billing, nil)
	svc.SetVoiceClones(repo, trainer)
	setVideoSettings(svc, VideoSettings{CloneSlots: []string{"S_slot1", "S_slot2"}, ClonePrice: 30, CloneMaxPerUser: 1})
	return svc, repo, trainer, tts, billing
}

func TestVideoCloneLifecycle(t *testing.T) {
	svc, repo, trainer, tts, _ := newCloneTestService(t)
	ctx := context.Background()
	wav := fakeWAV(15)

	_, err := svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "我的声音", Audio: wav})
	require.ErrorIs(t, err, ErrVideoCloneConsent)
	_, err = svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "我的声音", Audio: fakeWAV(2), Consent: true})
	require.ErrorIs(t, err, ErrVideoCloneLength)
	_, err = svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "我的声音", Audio: []byte(strings.Repeat("x", 5000)), Consent: true})
	require.ErrorIs(t, err, ErrVideoCloneAudio)

	c, err := svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "我的声音", Audio: wav, Consent: true, IP: "1.2.3.4"})
	require.NoError(t, err)
	require.Equal(t, VideoCloneTraining, c.Status)
	require.Equal(t, "clone:1", c.Voice)
	require.Equal(t, []string{"S_slot1"}, trainer.trained)
	_, err = svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "第二个", Audio: wav, Consent: true})
	require.ErrorIs(t, err, ErrVideoCloneLimit)

	// Not usable while training.
	_, err = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Voice: c.Voice}})
	require.ErrorIs(t, err, ErrVideoCloneNotReady)

	// Training finishes: the fee is taken once.
	trainer.status["S_slot1"] = "ready"
	list, err := svc.ListClones(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, VideoCloneReady, list[0].Status)
	require.Equal(t, []float64{30}, repo.charged)
	_, _ = svc.ListClones(ctx, 7)
	require.Equal(t, []float64{30}, repo.charged)

	// Someone else cannot narrate with it.
	other := &APIKey{ID: 2, UserID: 8, Key: "sk-b"}
	_, err = svc.Create(ctx, other, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Voice: c.Voice}})
	require.ErrorIs(t, err, ErrVideoCloneNotFound)

	// The owner's film speaks through the slot.
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Voice: c.Voice}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.NotEmpty(t, tts.voices)
	for _, v := range tts.voices {
		require.Equal(t, "S_slot1", v)
	}

	// Deleted: re-narration stops, the slot goes to the next user.
	require.NoError(t, svc.DeleteClone(ctx, 7, c.ID))
	_, err = svc.SetVoice(ctx, videoKeyA, p.ID, c.Voice, 0)
	require.ErrorIs(t, err, ErrVideoCloneNotFound)
	c2, err := svc.CreateClone(ctx, other, VideoCloneInput{Name: "B", Audio: wav, Consent: true})
	require.NoError(t, err)
	require.Equal(t, []string{"S_slot1", "S_slot1"}, trainer.trained)
	require.Equal(t, int64(8), c2.UserID)
}

func TestVideoCloneTrainingFailures(t *testing.T) {
	svc, repo, trainer, _, billing := newCloneTestService(t)
	ctx := context.Background()
	wav := fakeWAV(15)

	// A rejected recording frees the slot and costs nothing.
	trainer.trainErr = &videoCloneRejected{reason: "背景噪音太大"}
	_, err := svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "A", Audio: wav, Consent: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "背景噪音太大")
	list, _ := svc.ListClones(ctx, 7)
	require.Empty(t, list)
	trainer.trainErr = nil

	// Training fails later: shown, retrain allowed, still no fee.
	c, err := svc.CreateClone(ctx, videoKeyA, VideoCloneInput{Name: "A", Audio: wav, Consent: true})
	require.NoError(t, err)
	trainer.status[c.SpeakerID] = "豆包没能从这段录音复刻出声音"
	list, _ = svc.ListClones(ctx, 7)
	require.Equal(t, VideoCloneFailed, list[0].Status)
	require.Empty(t, repo.charged)
	trainer.readyNow = true
	c, err = svc.RetrainClone(ctx, videoKeyA, c.ID, wav)
	require.NoError(t, err)
	require.Equal(t, VideoCloneReady, c.Status)
	require.Equal(t, 2, c.Trainings)
	require.Equal(t, []float64{30}, repo.charged)

	// Too little balance is refused before training.
	billing.balance = 5
	other := &APIKey{ID: 2, UserID: 8, Key: "sk-b"}
	_, err = svc.CreateClone(ctx, other, VideoCloneInput{Name: "B", Audio: wav, Consent: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "余额不足")

	// No free slot.
	billing.balance = 100
	_, err = svc.CreateClone(ctx, other, VideoCloneInput{Name: "B", Audio: wav, Consent: true})
	require.NoError(t, err)
	third := &APIKey{ID: 3, UserID: 9, Key: "sk-c"}
	_, err = svc.CreateClone(ctx, third, VideoCloneInput{Name: "C", Audio: wav, Consent: true})
	require.ErrorIs(t, err, ErrVideoCloneFull)

	// Off without slots.
	setVideoSettings(svc, VideoSettings{})
	_, err = svc.CreateClone(ctx, third, VideoCloneInput{Name: "C", Audio: wav, Consent: true})
	require.ErrorIs(t, err, ErrVideoCloneOff)
}

func TestCheckCloneAudio(t *testing.T) {
	f, err := checkCloneAudio(fakeWAV(12))
	require.NoError(t, err)
	require.Equal(t, "wav", f)
	require.InDelta(t, 12, wavDuration(fakeWAV(12)), 1e-9)
	_, err = checkCloneAudio(fakeWAV(45))
	require.ErrorIs(t, err, ErrVideoCloneLength)
	f, err = checkCloneAudio(fakeMP3(500)) // 12 s
	require.NoError(t, err)
	require.Equal(t, "mp3", f)
	m4a := append([]byte{0, 0, 0, 0x20}, []byte("ftypM4A ")...)
	f, err = checkCloneAudio(append(m4a, make([]byte, 2000)...))
	require.NoError(t, err)
	require.Equal(t, "m4a", f)
}

func TestNormalizeCloneSlots(t *testing.T) {
	got, err := normalizeCloneSlots([]string{" S_abc123 ", "", "S_abc123", "S_x-y_z"})
	require.NoError(t, err)
	require.Equal(t, []string{"S_abc123", "S_x-y_z"}, got)
	_, err = normalizeCloneSlots([]string{"zh_female_vv_uranus_bigtts"})
	require.Error(t, err)
	_, err = normalizeCloneSlots([]string{"S_a b"})
	require.Error(t, err)
}

func TestDoubaoVoiceTrainerProtocol(t *testing.T) {
	var gotHeader http.Header
	var gotBody map[string]any
	reply := `{"code":0,"message":"","status":1,"speaker_id":"S_abc","available_training_times":14}`
	code := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		gotBody = nil
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(code)
		_, _ = io.WriteString(w, reply)
	}))
	defer srv.Close()
	d := NewDoubaoVoiceTrainer("key", "", "")
	d.trainURL, d.statusURL = srv.URL+"/train", srv.URL+"/status"
	ctx := context.Background()

	ready, err := d.Train(ctx, "S_abc", []byte("RIFFdata"), "wav")
	require.NoError(t, err)
	require.False(t, ready)
	require.Equal(t, "key", gotHeader.Get("X-Api-Key"))
	require.NotEmpty(t, gotHeader.Get("X-Api-Request-Id"))
	require.Equal(t, "S_abc", gotBody["speaker_id"])
	audio := gotBody["audio"].(map[string]any)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("RIFFdata")), audio["data"])
	require.Equal(t, "wav", audio["format"])
	require.Nil(t, gotBody["audios"]) // V3 takes one audio object

	reply = `{"code":0,"status":2,"speaker_id":"S_abc"}`
	ready, err = d.Train(ctx, "S_abc", []byte("x"), "wav")
	require.NoError(t, err)
	require.True(t, ready)

	reply = `{"code":0,"status":4}`
	ok, failed, err := d.Status(ctx, "S_abc")
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, failed)
	require.Equal(t, map[string]any{"speaker_id": "S_abc"}, gotBody)
	reply = `{"code":0,"status":1}`
	ok, failed, err = d.Status(ctx, "S_abc")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, failed)
	reply = `{"code":0,"status":3}`
	_, failed, err = d.Status(ctx, "S_abc")
	require.NoError(t, err)
	require.NotEmpty(t, failed)

	// A recording problem is the user's to fix; anything else is the service.
	code, reply = http.StatusBadRequest, `{"code":45001114,"message":"audio quality is poor"}`
	_, err = d.Train(ctx, "S_abc", []byte("x"), "wav")
	var rejected *videoCloneRejected
	require.ErrorAs(t, err, &rejected)
	require.Contains(t, cloneTrainError(err).Error(), "录音质量较差")
	code, reply = http.StatusInternalServerError, `{"code":55001301,"message":"db"}`
	_, err = d.Train(ctx, "S_abc", []byte("x"), "wav")
	require.ErrorIs(t, err, errEdgeTTS)
	require.ErrorIs(t, cloneTrainError(err), ErrVideoCloneTrain)

	// The old console's App ID + Access Token.
	code, reply = http.StatusOK, `{"code":0,"status":2}`
	old := NewDoubaoVoiceTrainer("", "app", "token")
	old.trainURL = srv.URL
	_, err = old.Train(ctx, "S_abc", []byte("x"), "wav")
	require.NoError(t, err)
	require.Equal(t, "app", gotHeader.Get("X-Api-App-Key"))
	require.Equal(t, "token", gotHeader.Get("X-Api-Access-Key"))
	require.Empty(t, gotHeader.Get("X-Api-Key"))
}

func TestDoubaoTTSClonedVoiceResource(t *testing.T) {
	var gotHeader http.Header
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = io.WriteString(w, `{"code":0,"data":"`+base64.StdEncoding.EncodeToString(fakeMP3(10))+`"}`+"\n"+`{"code":20000000}`+"\n")
	}))
	defer srv.Close()
	d := NewDoubaoTTS("key", "", "")
	d.url = srv.URL
	_, err := d.Synthesize(context.Background(), "你好", "S_abc", 0, "欢快地")
	require.NoError(t, err)
	require.Equal(t, "seed-icl-2.0", gotHeader.Get("X-Api-Resource-Id"))
	req := gotBody["req_params"].(map[string]any)
	require.Equal(t, "S_abc", req["speaker"])
	require.NotContains(t, req["additions"], "欢快地")

	// A project's clone:<id> never reaches 豆包 as a speaker.
	_, err = d.Synthesize(context.Background(), "你好", "clone:3", 0, "")
	require.NoError(t, err)
	require.Equal(t, "seed-tts-2.0", gotHeader.Get("X-Api-Resource-Id"))
	require.Equal(t, DoubaoVideoVoices[0].ID, gotBody["req_params"].(map[string]any)["speaker"])
}
