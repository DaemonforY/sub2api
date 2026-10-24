//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeMP3 is n MPEG-2 layer III frames at 24 kHz, 48 kbit/s (144 bytes, 24 ms each).
func fakeMP3(n int) []byte {
	frame := make([]byte, 144)
	copy(frame, []byte{0xff, 0xf3, 0x64, 0xc4})
	var out []byte
	for i := 0; i < n; i++ {
		out = append(out, frame...)
	}
	return out
}

func TestMP3Duration(t *testing.T) {
	require.InDelta(t, 2.4, mp3Duration(fakeMP3(100)), 1e-9)
	tagged := append([]byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 10}, make([]byte, 10)...)
	require.InDelta(t, 0.24, mp3Duration(append(tagged, fakeMP3(10)...)), 1e-9)
	require.Zero(t, mp3Duration([]byte("not audio")))
}

func TestDoubaoTTSProtocol(t *testing.T) {
	var gotHeader http.Header
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		audio := base64.StdEncoding.EncodeToString(fakeMP3(50))
		_, _ = io.WriteString(w, `{"code":0,"message":"","data":"`+audio[:len(audio)/2]+`"}`+"\n")
		_, _ = io.WriteString(w, `{"code":0,"message":"","data":"`+audio[len(audio)/2:]+`"}`+"\n")
		_, _ = io.WriteString(w, `{"code":0,"message":"","data":null,"sentence":{"text":"你好，世界。","words":[{"word":"你","startTime":0.1,"endTime":0.3},{"word":"好，","startTime":0.3,"endTime":0.5},{"word":"世界。","startTime":0.6,"endTime":1.0}]}}`+"\n")
		_, _ = io.WriteString(w, `{"code":20000000,"message":"OK","data":null}`+"\n")
	}))
	defer srv.Close()
	d := NewDoubaoTTS("", "app", "token")
	d.url = srv.URL
	sp, err := d.Synthesize(context.Background(), "你好，世界。", "zh_male_m191_uranus_bigtts", 10, "沉稳地讲解")
	require.NoError(t, err)
	require.Len(t, sp.Audio, 50*144)
	require.InDelta(t, 1.2, sp.Duration, 1e-9)
	require.Equal(t, []VideoWord{{Text: "你", Start: 0.1, End: 0.3}, {Text: "好", Start: 0.3, End: 0.5}, {Text: "世界", Start: 0.6, End: 1.0}}, sp.Words)

	require.Equal(t, "app", gotHeader.Get("X-Api-App-Id"))
	require.Equal(t, "token", gotHeader.Get("X-Api-Access-Key"))
	require.Equal(t, "seed-tts-2.0", gotHeader.Get("X-Api-Resource-Id"))
	require.NotEmpty(t, gotHeader.Get("X-Api-Request-Id"))
	req := gotBody["req_params"].(map[string]any)
	require.Equal(t, "zh_male_m191_uranus_bigtts", req["speaker"])
	require.Equal(t, float64(10), req["audio_params"].(map[string]any)["speech_rate"])
	require.Contains(t, req["additions"], "沉稳地讲解")
}

func TestDoubaoTTSErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{http.StatusUnauthorized, `{"error":"bad token"}`},
		{http.StatusOK, `{"code":45000000,"message":"quota exceeded"}` + "\n"},
		{http.StatusOK, `{"code":20000000,"message":"OK"}` + "\n"},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = io.WriteString(w, tc.body)
		}))
		d := NewDoubaoTTS("", "app", "token")
		d.url = srv.URL
		_, err := d.Synthesize(context.Background(), "你好", "zh_female_vv_uranus_bigtts", 0, "")
		require.ErrorIs(t, err, errEdgeTTS, tc.body) // shown as 「配音服务暂时不可用」
		srv.Close()
	}
	require.Nil(t, NewDoubaoTTS("", "", "token"))

	// The new console's API Key goes in X-Api-Key.
	var gotKey, gotApp string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotApp = r.Header.Get("X-Api-Key"), r.Header.Get("X-Api-App-Id")
		_, _ = io.WriteString(w, `{"code":0,"data":"`+base64.StdEncoding.EncodeToString(fakeMP3(5))+`"}`+"\n"+`{"code":20000000}`+"\n")
	}))
	defer srv.Close()
	d := NewDoubaoTTS(" key ", "", "")
	d.url = srv.URL
	_, err := d.Synthesize(context.Background(), "你好", "zh_female_vv_uranus_bigtts", 0, "")
	require.NoError(t, err)
	require.Equal(t, "key", gotKey)
	require.Empty(t, gotApp)
}

type fakePaidTTS struct {
	mu    sync.Mutex
	tones []string
}

func (f *fakePaidTTS) Synthesize(_ context.Context, text, _ string, _ int, tone string) (*VideoSpeech, error) {
	f.mu.Lock()
	f.tones = append(f.tones, tone)
	f.mu.Unlock()
	return &VideoSpeech{Audio: []byte("paid:" + text), Duration: 2}, nil
}

type fakeTTSBilling struct {
	mu      sync.Mutex
	balance float64
	charges []VideoTTSCharge
}

func (f *fakeTTSBilling) Balance(context.Context, int64) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.balance, nil
}

func (f *fakeTTSBilling) ChargeTTS(_ context.Context, c VideoTTSCharge) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.balance -= c.Cost
	f.charges = append(f.charges, c)
	return nil
}

func TestVideoPaidVoiceIsChargedPerScene(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, free := newTestVideoService(t, model)
	ctx := context.Background()
	voice := "zh_male_m191_uranus_bigtts"

	// Off until 豆包 is configured.
	_, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Voice: voice}})
	require.ErrorIs(t, err, ErrVideoPaidVoiceOff)
	require.Len(t, svc.Voices(), len(VideoVoices))

	paid, billing := &fakePaidTTS{}, &fakeTTSBilling{balance: 1}
	svc.SetPaidTTS(paid, billing, nil)
	require.Len(t, svc.Voices(), len(VideoVoices)+len(DoubaoVideoVoices))
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Voice: voice}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Zero(t, free.calls)
	require.Len(t, billing.charges, 2)
	chars := 0
	for _, c := range billing.charges {
		require.Equal(t, int64(7), c.UserID)
		require.Equal(t, voice, c.Voice)
		require.InDelta(t, float64(c.Chars)*videoDefaultTTSPrice/10000, c.Cost, 1e-12)
		chars += c.Chars
	}
	require.Equal(t, len([]rune("第一段旁白。第二段旁白。")), chars)
	require.InDelta(t, videoTTSCost(chars, videoDefaultTTSPrice), got.Usage.TTSCost, 1e-12)

	// Re-recording with too little balance is refused before anything is recorded.
	billing.balance = 0
	_, err = svc.SetVoice(ctx, videoKeyA, p.ID, "zh_female_vv_uranus_bigtts", 0)
	require.NoError(t, err)
	svc.Wait()
	got, _ = svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusFailed, got.Status)
	require.True(t, strings.HasPrefix(got.Error, "余额不足"), got.Error)
	require.Len(t, billing.charges, 2)
}
