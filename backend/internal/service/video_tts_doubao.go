package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// Paid narration through 豆包语音 2.0 (火山引擎, seed-tts-2.0): far more natural than Edge, reads the
// emotion from the text, and follows a short tone hint per scene (additions.context_texts). Billed per
// character from the user's balance (see voiceScenes). V3 HTTP unidirectional API: one POST, the reply
// is one JSON object per line — base64 audio chunks, sentences with word timings, then code 20000000.

const (
	doubaoTTSURL        = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
	doubaoTTSResourceID = "seed-tts-2.0"
	// Speaking with a cloned voice (声音复刻 2.0) bills a different resource.
	doubaoCloneResourceID = "seed-icl-2.0"
	doubaoTTSDone         = 20000000
	// Requests longer than this (bytes of text) are split.
	doubaoMaxChunkBytes = 1500
)

var errDoubaoTTS = fmt.Errorf("%w: doubao", errEdgeTTS)

// DoubaoVideoVoices are the paid narrators (豆包语音 2.0 preset voices suited to narration).
var DoubaoVideoVoices = []VideoVoice{
	{ID: "zh_female_vv_uranus_bigtts", Name: "Vivi", Gender: "女", Style: "自然亲切，通用讲解", Lang: "zh", Paid: true},
	{ID: "zh_female_xiaohe_uranus_bigtts", Name: "小何", Gender: "女", Style: "温柔知性，适合科普和课程", Lang: "zh", Paid: true},
	{ID: "zh_female_cancan_uranus_bigtts", Name: "知性灿灿", Gender: "女", Style: "知性大方，适合讲解和访谈", Lang: "zh", Paid: true},
	{ID: "zh_female_linjianvhai_uranus_bigtts", Name: "邻家女孩", Gender: "女", Style: "轻松活泼，适合生活和种草", Lang: "zh", Paid: true},
	{ID: "zh_female_xiaoxue_uranus_bigtts", Name: "儿童绘本", Gender: "女", Style: "温暖童趣，适合绘本和儿童内容", Lang: "zh", Paid: true},
	{ID: "zh_male_m191_uranus_bigtts", Name: "云舟", Gender: "男", Style: "沉稳大气，适合纪录片和课程", Lang: "zh", Paid: true},
	{ID: "zh_male_taocheng_uranus_bigtts", Name: "小天", Gender: "男", Style: "阳光清爽，适合科普和宣传", Lang: "zh", Paid: true},
	{ID: "zh_male_yuanboxiaoshu_uranus_bigtts", Name: "渊博小叔", Gender: "男", Style: "博学从容，适合知识科普", Lang: "zh", Paid: true},
	{ID: "zh_male_jieshuoxiaoming_uranus_bigtts", Name: "解说小明", Gender: "男", Style: "节奏明快，适合解说", Lang: "zh", Paid: true},
	{ID: "zh_male_cixingjieshuonan_uranus_bigtts", Name: "磁性解说", Gender: "男", Style: "低沉磁性，适合影视解说", Lang: "zh", Paid: true},
	{ID: "zh_male_xuanyijieshuo_uranus_bigtts", Name: "悬疑解说", Gender: "男", Style: "紧张神秘，适合悬疑和故事", Lang: "zh", Paid: true},
	{ID: "zh_male_shenyeboke_uranus_bigtts", Name: "深夜播客", Gender: "男", Style: "松弛低语，适合播客和情感", Lang: "zh", Paid: true},
}

// videoVoicePaid: the preset 豆包 voices and the users' own clones (clone:<id>) are billed per character.
func videoVoicePaid(id string) bool {
	if _, ok := videoCloneID(id); ok {
		return true
	}
	for _, v := range DoubaoVideoVoices {
		if v.ID == id {
			return true
		}
	}
	return false
}

// DoubaoTTS speaks through 豆包语音 2.0.
type DoubaoTTS struct {
	apiKey, appID, accessKey, url string
	client                        *http.Client
}

// NewDoubaoTTS takes the console's API Key (X-Api-Key), or the old console's App ID + Access Token;
// it returns nil when neither is set (paid voices off).
func NewDoubaoTTS(apiKey, appID, accessKey string) *DoubaoTTS {
	apiKey, appID, accessKey = strings.TrimSpace(apiKey), strings.TrimSpace(appID), strings.TrimSpace(accessKey)
	if apiKey == "" && (appID == "" || accessKey == "") {
		return nil
	}
	return &DoubaoTTS{apiKey: apiKey, appID: appID, accessKey: accessKey, url: doubaoTTSURL, client: &http.Client{Timeout: 90 * time.Second}}
}

func (d *DoubaoTTS) Synthesize(ctx context.Context, text, voice string, rate int, tone string) (*VideoSpeech, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return &VideoSpeech{}, nil
	}
	resource := doubaoTTSResourceID
	switch {
	case videoSpeakerIDOK(voice):
		resource = doubaoCloneResourceID // a cloned voice slot
	case !videoVoicePaid(voice) || strings.HasPrefix(voice, VideoClonePrefix):
		voice = DoubaoVideoVoices[0].ID
	}
	out := &VideoSpeech{}
	for _, chunk := range splitSpeechText(text, doubaoMaxChunkBytes) {
		part, err := d.synthesizeChunk(ctx, chunk, voice, resource, max(-50, min(rate, 100)), tone)
		if err != nil {
			return nil, err
		}
		for _, w := range part.Words {
			out.Words = append(out.Words, VideoWord{Text: w.Text, Start: w.Start + out.Duration, End: w.End + out.Duration})
		}
		out.Audio = append(out.Audio, part.Audio...)
		out.Duration += part.Duration
	}
	return out, nil
}

func (d *DoubaoTTS) synthesizeChunk(ctx context.Context, text, voice, resource string, rate int, tone string) (*VideoSpeech, error) {
	additions := map[string]any{"disable_markdown_filter": false}
	// Clones speak with the standard 2.0 model, which takes no tone hint (only the expressive one does).
	if tone = strings.TrimSpace(tone); tone != "" && resource != doubaoCloneResourceID {
		additions["context_texts"] = []string{clipRunes(tone, 60)}
	}
	addJSON, _ := json.Marshal(additions)
	body, _ := json.Marshal(map[string]any{
		"user": map[string]string{"uid": "hivegpt-video"},
		"req_params": map[string]any{
			"text":    text,
			"speaker": voice,
			"audio_params": map[string]any{
				"format": "mp3", "sample_rate": 24000, "speech_rate": rate,
				"enable_subtitle": true,
			},
			"additions": string(addJSON),
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	reqID := make([]byte, 16)
	_, _ = rand.Read(reqID)
	req.Header.Set("Content-Type", "application/json")
	if d.apiKey != "" {
		req.Header.Set("X-Api-Key", d.apiKey)
	} else {
		req.Header.Set("X-Api-App-Id", d.appID)
		req.Header.Set("X-Api-Access-Key", d.accessKey)
	}
	req.Header.Set("X-Api-Resource-Id", resource)
	req.Header.Set("X-Api-Request-Id", hex.EncodeToString(reqID))
	resp, err := d.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: request: %v", errDoubaoTTS, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("%w: HTTP %d: %s", errDoubaoTTS, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return parseDoubaoStream(resp.Body)
}

type doubaoFrame struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Data     string `json:"data"`
	Sentence *struct {
		Words []struct {
			Word      string  `json:"word"`
			StartTime float64 `json:"startTime"`
			EndTime   float64 `json:"endTime"`
		} `json:"words"`
	} `json:"sentence"`
}

func parseDoubaoStream(r io.Reader) (*VideoSpeech, error) {
	var audio bytes.Buffer
	var words []VideoWord
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16<<20)
	done := false
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var f doubaoFrame
		if err := json.Unmarshal(line, &f); err != nil {
			return nil, fmt.Errorf("%w: bad frame: %v", errDoubaoTTS, err)
		}
		switch {
		case f.Code == doubaoTTSDone:
			done = true
		case f.Code != 0:
			return nil, fmt.Errorf("%w: code %d: %s", errDoubaoTTS, f.Code, f.Message)
		}
		if done {
			break
		}
		if f.Data != "" {
			b, err := base64.StdEncoding.DecodeString(f.Data)
			if err != nil {
				return nil, fmt.Errorf("%w: bad audio: %v", errDoubaoTTS, err)
			}
			_, _ = audio.Write(b)
		}
		if f.Sentence != nil {
			for _, w := range f.Sentence.Words {
				if t := strings.TrimRightFunc(w.Word, unicode.IsPunct); strings.TrimSpace(t) != "" {
					words = append(words, VideoWord{Text: t, Start: w.StartTime, End: w.EndTime})
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%w: read: %v", errDoubaoTTS, err)
	}
	if audio.Len() == 0 {
		return nil, fmt.Errorf("%w: no audio", errDoubaoTTS)
	}
	duration := mp3Duration(audio.Bytes())
	// Timings are in seconds; convert if a version reports milliseconds.
	if n := len(words); n > 0 && duration > 0 && words[n-1].End > duration*5 {
		for i := range words {
			words[i].Start, words[i].End = words[i].Start/1000, words[i].End/1000
		}
	}
	return &VideoSpeech{Audio: audio.Bytes(), Duration: duration, Words: words}, nil
}

var (
	mp3BitratesV1L3 = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	mp3BitratesV2L3 = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	mp3SampleRates  = [4][3]int{{11025, 12000, 8000}, {0, 0, 0}, {22050, 24000, 16000}, {44100, 48000, 32000}} // by version bits
)

// mp3Duration walks the MPEG audio layer III frames (after an optional ID3v2 tag) and sums their length.
func mp3Duration(b []byte) float64 {
	i := 0
	if len(b) >= 10 && string(b[:3]) == "ID3" {
		i = 10 + (int(b[6]&0x7f)<<21 | int(b[7]&0x7f)<<14 | int(b[8]&0x7f)<<7 | int(b[9]&0x7f))
	}
	total := 0.0
	for i+4 <= len(b) {
		h := b[i : i+4]
		if h[0] != 0xff || h[1]&0xe0 != 0xe0 {
			i++
			continue
		}
		version := int(h[1]>>3) & 3 // 3 = MPEG1, 2 = MPEG2, 0 = MPEG2.5
		layer := int(h[1]>>1) & 3   // 1 = layer III
		brIdx, srIdx, pad := int(h[2]>>4), int(h[2]>>2)&3, int(h[2]>>1)&1
		if version == 1 || layer != 1 || srIdx == 3 || brIdx == 0 || brIdx == 15 {
			i++
			continue
		}
		sr := mp3SampleRates[version][srIdx]
		var size, samples int
		if version == 3 {
			size, samples = 144000*mp3BitratesV1L3[brIdx]/sr+pad, 1152
		} else {
			size, samples = 72000*mp3BitratesV2L3[brIdx]/sr+pad, 576
		}
		if size < 4 {
			i++
			continue
		}
		total += float64(samples) / float64(sr)
		i += size
	}
	return total
}

// videoTTSFor picks the engine for a voice: paid voices need 豆包 configured.
// A clone (clone:<id>) speaks through its slot, for its owner only.
func (s *VideoService) videoTTSFor(ctx context.Context, userID int64, voice string) (func(ctx context.Context, text, voice string, rate int, tone string) (*VideoSpeech, error), error) {
	if _, ok := videoCloneID(voice); ok {
		if s.paidTTS == nil {
			return nil, ErrVideoPaidVoiceOff
		}
		speaker, err := s.cloneSpeaker(ctx, userID, voice)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, text, _ string, rate int, tone string) (*VideoSpeech, error) {
			return s.paidTTS.Synthesize(ctx, text, speaker, rate, tone)
		}, nil
	}
	if !videoVoicePaid(voice) {
		return func(ctx context.Context, text, voice string, rate int, _ string) (*VideoSpeech, error) {
			return s.tts.Synthesize(ctx, text, voice, rate)
		}, nil
	}
	if s.paidTTS == nil {
		return nil, ErrVideoPaidVoiceOff
	}
	return s.paidTTS.Synthesize, nil
}

// VideoPaidTTS is the paid engine (豆包语音 2.0); tone is a short delivery hint for the scene.
type VideoPaidTTS interface {
	Synthesize(ctx context.Context, text, voice string, rate int, tone string) (*VideoSpeech, error)
}

// VideoTTSBilling takes paid narration from the balance (videoRepository).
type VideoTTSBilling interface {
	Balance(ctx context.Context, userID int64) (float64, error)
	ChargeTTS(ctx context.Context, c VideoTTSCharge) error
}

type VideoTTSCharge struct {
	UserID    int64
	ProjectID string
	SceneID   string
	Voice     string
	Chars     int
	Cost      float64
}

type videoBalanceCache interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// SetPaidTTS enables the paid narrators. billing must be set for them to be offered.
func (s *VideoService) SetPaidTTS(tts VideoPaidTTS, billing VideoTTSBilling, cache videoBalanceCache) {
	if tts == nil || billing == nil {
		return
	}
	s.paidTTS, s.ttsBilling, s.balanceCache = tts, billing, cache
}

// afterTTSCharge refreshes the cached balance so the next API call sees the deduction.
func (s *VideoService) afterTTSCharge(userID int64) {
	if s.balanceCache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.balanceCache.InvalidateUserBalance(ctx, userID); err != nil {
		slog.Warn("video: invalidate balance cache", "user", userID, "err", err)
	}
}

// PaidVoicesEnabled reports whether the paid narrators can be used.
func (s *VideoService) PaidVoicesEnabled() bool { return s.paidTTS != nil }

// Voices lists the narrators for the create page: the free Edge voices, then the paid ones when on.
func (s *VideoService) Voices() []VideoVoice {
	if s.paidTTS == nil {
		return VideoVoices
	}
	return append(append([]VideoVoice{}, VideoVoices...), DoubaoVideoVoices...)
}

// videoTTSCost is what a paid recording of chars characters costs, in balance dollars.
func videoTTSCost(chars int, pricePer10k float64) float64 {
	return float64(chars) * pricePer10k / 10000
}
