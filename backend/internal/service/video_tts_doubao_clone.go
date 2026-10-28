package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 豆包声音复刻 2.0 V3 API (docs 6561/2534906 音色注册, 2535742 查询): POST voice_clone with a
// base64 recording trains a slot (the reply already carries its status), POST get_voice reads it.
// Status 2 (Success) and 4 (Active) can speak; synthesis then uses resource seed-icl-2.0.
// Failures come back as a non-200 HTTP status with {code, message}.

const (
	doubaoCloneTrainURL  = "https://openspeech.bytedance.com/api/v3/tts/voice_clone"
	doubaoCloneStatusURL = "https://openspeech.bytedance.com/api/v3/tts/get_voice"
	// The old console's App ID + Access Token (X-Api-App-Key) are for this resource.
	doubaoCloneTrainResource = "volc.megatts.voiceclone"
)

// Codes about the recording itself (noise, silence, voiceprint, content audit, length): the user can
// fix them by recording again.
var doubaoCloneRejections = map[int]string{
	45001102: "听不清录音内容",
	45001104: "声纹检测没通过（不能复刻名人或已注册的声音）",
	45001105: "录音文件读不出来",
	45001108: "录音格式无法转换",
	45001112: "背景噪音太大",
	45001114: "录音质量较差",
	45001117: "录音超过 30 秒",
	45001122: "没有检测到人声",
	45001124: "录音内容没通过审核",
	45001127: "录音没通过审核",
	45001128: "录音内容没通过审核",
}

// DoubaoVoiceTrainer trains voice slots with the same credentials as DoubaoTTS.
type DoubaoVoiceTrainer struct {
	apiKey, appID, accessKey string
	trainURL, statusURL      string
	client                   *http.Client
}

func NewDoubaoVoiceTrainer(apiKey, appID, accessKey string) *DoubaoVoiceTrainer {
	apiKey, appID, accessKey = strings.TrimSpace(apiKey), strings.TrimSpace(appID), strings.TrimSpace(accessKey)
	if apiKey == "" && (appID == "" || accessKey == "") {
		return nil
	}
	return &DoubaoVoiceTrainer{apiKey: apiKey, appID: appID, accessKey: accessKey, trainURL: doubaoCloneTrainURL, statusURL: doubaoCloneStatusURL,
		client: &http.Client{Timeout: 120 * time.Second}}
}

type doubaoVoiceReply struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Left     *int   `json:"available_training_times"`
	Speakers []struct {
		ModelType int `json:"model_type"`
	} `json:"speaker_status"`
}

func (r *doubaoVoiceReply) ready() bool { return r.Status == 2 || r.Status == 4 }

func (d *DoubaoVoiceTrainer) Train(ctx context.Context, speakerID string, audio []byte, format string) (bool, error) {
	reply, err := d.post(ctx, d.trainURL, map[string]any{
		"speaker_id": speakerID,
		"audio":      map[string]string{"data": base64.StdEncoding.EncodeToString(audio), "format": format},
		"language":   0,
		// Home recordings are rarely clean; 豆包 suggests denoising noisy samples.
		"extra_params": map[string]any{"enable_audio_denoise": true},
	})
	if err != nil {
		return false, err
	}
	if reply.Status == 3 {
		return false, &videoCloneRejected{reason: "复刻没有成功"}
	}
	return reply.ready(), nil
}

func (d *DoubaoVoiceTrainer) Status(ctx context.Context, speakerID string) (bool, string, error) {
	reply, err := d.post(ctx, d.statusURL, map[string]any{"speaker_id": speakerID})
	if err != nil {
		return false, "", err
	}
	switch {
	case reply.ready():
		return true, "", nil
	case reply.Status == 3:
		return false, "豆包没能从这段录音复刻出声音", nil
	case reply.Status == 0:
		return false, "音色位里没有找到训练记录", nil
	}
	return false, "", nil
}

func (d *DoubaoVoiceTrainer) post(ctx context.Context, url string, payload map[string]any) (*doubaoVoiceReply, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	reqID := make([]byte, 16)
	_, _ = rand.Read(reqID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Request-Id", hex.EncodeToString(reqID))
	if d.apiKey != "" {
		req.Header.Set("X-Api-Key", d.apiKey)
	} else {
		req.Header.Set("X-Api-App-Key", d.appID)
		req.Header.Set("X-Api-App-Id", d.appID)
		req.Header.Set("X-Api-Access-Key", d.accessKey)
		req.Header.Set("X-Api-Resource-Id", doubaoCloneTrainResource)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: clone request: %v", errDoubaoTTS, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var reply doubaoVoiceReply
	_ = json.Unmarshal(raw, &reply)
	if resp.StatusCode != http.StatusOK || (reply.Code != 0 && reply.Code != doubaoTTSDone) {
		if reason, ok := doubaoCloneRejections[reply.Code]; ok {
			return nil, &videoCloneRejected{reason: reason}
		}
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("%w: clone HTTP %d code %s: %s", errDoubaoTTS, resp.StatusCode, strconv.Itoa(reply.Code), snippet)
	}
	return &reply, nil
}
