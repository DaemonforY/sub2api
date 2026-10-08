package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Narration for HiveGPT 视频 comes from Microsoft Edge's read-aloud service (free neural voices, with
// word timings that drive the subtitles). It is not an official API, so it sits behind VideoTTS and
// can be swapped for a paid provider.

// VideoTTS turns one scene's narration into speech.
type VideoTTS interface {
	Synthesize(ctx context.Context, text, voice string, rate int) (*VideoSpeech, error)
}

// VideoSpeech is MP3 audio plus when each word is spoken (seconds from the start of the audio).
type VideoSpeech struct {
	Audio    []byte      `json:"-"`
	Duration float64     `json:"duration"`
	Words    []VideoWord `json:"words"`
}

type VideoWord struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// VideoVoice is one selectable narrator.
type VideoVoice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Style  string `json:"style"`
	Lang   string `json:"lang"`
}

// VideoVoices are the narrators offered in the UI (Edge neural voices).
var VideoVoices = []VideoVoice{
	{ID: "zh-CN-XiaoxiaoNeural", Name: "晓晓", Gender: "女", Style: "温暖自然，适合讲解和故事", Lang: "zh"},
	{ID: "zh-CN-YunxiNeural", Name: "云希", Gender: "男", Style: "阳光少年感，适合科普和宣传", Lang: "zh"},
	{ID: "zh-CN-YunjianNeural", Name: "云健", Gender: "男", Style: "浑厚有力，适合纪录片和体育", Lang: "zh"},
	{ID: "zh-CN-XiaoyiNeural", Name: "晓伊", Gender: "女", Style: "活泼可爱，适合儿童内容", Lang: "zh"},
	{ID: "zh-CN-YunyangNeural", Name: "云扬", Gender: "男", Style: "专业播音，适合新闻和课程", Lang: "zh"},
	{ID: "zh-CN-YunxiaNeural", Name: "云夏", Gender: "男", Style: "童声，适合绘本和儿歌", Lang: "zh"},
	{ID: "zh-CN-liaoning-XiaobeiNeural", Name: "晓北", Gender: "女", Style: "东北话，幽默接地气", Lang: "zh"},
	{ID: "zh-CN-shaanxi-XiaoniNeural", Name: "晓妮", Gender: "女", Style: "陕西话，亲切有味道", Lang: "zh"},
	{ID: "zh-HK-HiuMaanNeural", Name: "曉曼", Gender: "女", Style: "粤语", Lang: "zh-HK"},
	{ID: "zh-TW-HsiaoChenNeural", Name: "曉臻", Gender: "女", Style: "台湾腔，温柔", Lang: "zh-TW"},
	{ID: "en-US-AriaNeural", Name: "Aria", Gender: "女", Style: "English, friendly", Lang: "en"},
	{ID: "en-US-GuyNeural", Name: "Guy", Gender: "男", Style: "English, narrator", Lang: "en"},
}

// DefaultVideoVoice is used when the request names no (or an unknown) voice.
const DefaultVideoVoice = "zh-CN-XiaoxiaoNeural"

func videoVoiceKnown(id string) bool {
	for _, v := range VideoVoices {
		if v.ID == id {
			return true
		}
	}
	return false
}

const (
	edgeTrustedClientToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	edgeChromiumVersion    = "143.0.3650.75"
	edgeWSSURL             = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	edgeOutputFormat       = "audio-24khz-48kbitrate-mono-mp3"
	edgeBitrate            = 48000 // bits per second of edgeOutputFormat (CBR)
	// Windows file-time epoch offset, for the Sec-MS-GEC token.
	edgeWinEpoch = 11644473600
	// Requests longer than this (bytes of SSML text) are split.
	edgeMaxChunkBytes = 3000
)

var (
	errEdgeTTS       = errors.New("edge tts failed")
	errEdgeClockSkew = fmt.Errorf("%w: clock skew", errEdgeTTS)
)

// EdgeTTS speaks through Edge's read-aloud websocket.
type EdgeTTS struct {
	dialer *websocket.Dialer
	url    string // override in tests
	now    func() time.Time
	// skew corrects the token's clock when the service rejects it (its Date header is the truth).
	skew atomic.Int64
}

func NewEdgeTTS() *EdgeTTS {
	return &EdgeTTS{dialer: &websocket.Dialer{HandshakeTimeout: 15 * time.Second, EnableCompression: true}, url: edgeWSSURL, now: time.Now}
}

// edgeSecMSGEC is the anti-abuse token the service expects: SHA-256 of the Windows file time
// (rounded down to 5 minutes, in 100 ns ticks) followed by the trusted client token.
func edgeSecMSGEC(now time.Time) string {
	ticks := now.Unix() + edgeWinEpoch
	ticks -= ticks % 300
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d%s", ticks*10_000_000, edgeTrustedClientToken)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func edgeID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func edgeTimestamp(t time.Time) string {
	return t.UTC().Format("Mon Jan 02 2006 15:04:05") + " GMT+0000 (Coordinated Universal Time)"
}

// Synthesize speaks text with voice; rate is a percentage change of speed (-50 … +100).
func (e *EdgeTTS) Synthesize(ctx context.Context, text, voice string, rate int) (*VideoSpeech, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return &VideoSpeech{}, nil
	}
	if !videoVoiceKnown(voice) {
		voice = DefaultVideoVoice
	}
	rate = max(-50, min(rate, 100))
	out := &VideoSpeech{}
	for _, chunk := range splitSpeechText(text, edgeMaxChunkBytes) {
		part, err := e.synthesizeChunk(ctx, chunk, voice, rate)
		if errors.Is(err, errEdgeClockSkew) {
			part, err = e.synthesizeChunk(ctx, chunk, voice, rate)
		}
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

func (e *EdgeTTS) synthesizeChunk(ctx context.Context, text, voice string, rate int) (*VideoSpeech, error) {
	now := e.now().Add(time.Duration(e.skew.Load()))
	url := fmt.Sprintf("%s?TrustedClientToken=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=1-%s&ConnectionId=%s",
		e.url, edgeTrustedClientToken, edgeSecMSGEC(now), edgeChromiumVersion, edgeID())
	header := http.Header{}
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "no-cache")
	header.Set("Origin", "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold")
	header.Set("Accept-Language", "en-US,en;q=0.9")
	major, _, _ := strings.Cut(edgeChromiumVersion, ".")
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/"+major+".0.0.0 Safari/537.36 Edg/"+major+".0.0.0")
	header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	header.Set("Cookie", "muid="+strings.ToUpper(edgeID())+";")
	conn, resp, err := e.dialer.DialContext(ctx, url, header)
	status := 0
	if resp != nil {
		status = resp.StatusCode
		if status == http.StatusForbidden {
			if served, perr := http.ParseTime(resp.Header.Get("Date")); perr == nil && served.Sub(now).Abs() > time.Minute {
				e.skew.Store(int64(served.Sub(e.now())))
				_ = resp.Body.Close()
				return nil, errEdgeClockSkew
			}
		}
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: connect (HTTP %d): %v", errEdgeTTS, status, err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(now.Add(60 * time.Second))
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	config := "X-Timestamp:" + edgeTimestamp(now) + "\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"true"},"outputFormat":"` + edgeOutputFormat + `"}}}}` + "\r\n"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(config)); err != nil {
		return nil, fmt.Errorf("%w: config: %v", errEdgeTTS, err)
	}
	lang := "zh-CN"
	if parts := strings.SplitN(voice, "-", 3); len(parts) >= 2 {
		lang = parts[0] + "-" + parts[1]
	}
	ssml := fmt.Sprintf("<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='%s'><voice name='%s'><prosody pitch='+0Hz' rate='%+d%%' volume='+0%%'>%s</prosody></voice></speak>",
		lang, voice, rate, html.EscapeString(text))
	msg := "X-RequestId:" + edgeID() + "\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:" + edgeTimestamp(now) + "Z\r\nPath:ssml\r\n\r\n" + ssml
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return nil, fmt.Errorf("%w: ssml: %v", errEdgeTTS, err)
	}

	var audio bytes.Buffer
	var words []VideoWord
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("%w: read: %v", errEdgeTTS, err)
		}
		switch kind {
		case websocket.TextMessage:
			path, body := edgeSplitMessage(data)
			switch path {
			case "audio.metadata":
				words = append(words, parseEdgeWordBoundaries(body)...)
			case "turn.end":
				if audio.Len() == 0 {
					return nil, fmt.Errorf("%w: no audio", errEdgeTTS)
				}
				return &VideoSpeech{Audio: audio.Bytes(), Duration: float64(audio.Len()*8) / edgeBitrate, Words: words}, nil
			}
		case websocket.BinaryMessage:
			chunk, ok := edgeAudioPayload(data)
			if ok {
				_, _ = audio.Write(chunk)
			}
		}
	}
}

// edgeSplitMessage returns the Path header and the body of a text frame.
func edgeSplitMessage(data []byte) (string, []byte) {
	head, body, _ := bytes.Cut(data, []byte("\r\n\r\n"))
	for _, line := range strings.Split(string(head), "\r\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Path") {
			return strings.TrimSpace(v), body
		}
	}
	return "", body
}

// edgeAudioPayload extracts the MP3 bytes of a binary frame: a 2-byte big-endian header length,
// the headers, then the audio.
func edgeAudioPayload(data []byte) ([]byte, bool) {
	if len(data) < 2 {
		return nil, false
	}
	n := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+n {
		return nil, false
	}
	headers := string(data[2 : 2+n])
	if !strings.Contains(headers, "Path:audio") {
		return nil, false
	}
	return data[2+n:], true
}

func parseEdgeWordBoundaries(body []byte) []VideoWord {
	var meta struct {
		Metadata []struct {
			Type string `json:"Type"`
			Data struct {
				Offset   int64 `json:"Offset"`
				Duration int64 `json:"Duration"`
				Text     struct {
					Text string `json:"Text"`
				} `json:"text"`
			} `json:"Data"`
		} `json:"Metadata"`
	}
	if json.Unmarshal(body, &meta) != nil {
		return nil
	}
	var out []VideoWord
	for _, m := range meta.Metadata {
		if m.Type != "WordBoundary" || strings.TrimSpace(m.Data.Text.Text) == "" {
			continue
		}
		start := float64(m.Data.Offset) / 1e7
		out = append(out, VideoWord{Text: m.Data.Text.Text, Start: start, End: start + float64(m.Data.Duration)/1e7})
	}
	return out
}

// splitSpeechText cuts long narration at sentence ends so every request stays under maxBytes.
func splitSpeechText(text string, maxBytes int) []string {
	if len(text) <= maxBytes {
		return []string{text}
	}
	var chunks []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			chunks = append(chunks, s)
		}
		cur.Reset()
	}
	for _, r := range text {
		if cur.Len()+len(string(r)) > maxBytes {
			flush()
		}
		_, _ = cur.WriteRune(r)
		if strings.ContainsRune("。！？!?；;\n", r) && cur.Len() > maxBytes/2 {
			flush()
		}
	}
	flush()
	return chunks
}
