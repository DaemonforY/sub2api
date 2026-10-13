package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// WeChat pushes for the 内容引擎 (待审核, a daily run that could not start), through a service that
// forwards to personal WeChat. The owner registers with it and pastes the key:
//   - serverchan: Server酱 (sct.ftqq.com), SendKey. Turbo keys (SCT…) and Server酱³ keys (sctp<uid>t…).
//   - pushplus: PushPlus (pushplus.plus), token.
//   - wecom: a 企业微信 group robot's webhook URL (shows in 企业微信, and in WeChat once 微信插件 is on).

const (
	contentNotifyServerChan = "serverchan"
	contentNotifyPushPlus   = "pushplus"
	contentNotifyWecom      = "wecom"
)

func contentNotifyTypeOK(t string) bool {
	switch t {
	case "", contentNotifyServerChan, contentNotifyPushPlus, contentNotifyWecom:
		return true
	}
	return false
}

type contentNotifyMessage struct {
	Title string
	Body  string
	URL   string
}

type contentNotifier interface {
	Send(ctx context.Context, kind, key string, msg contentNotifyMessage) error
}

type contentHTTPNotifier struct {
	client *http.Client
	// Endpoints (tests point them at a local server).
	serverChan  string // %s = SendKey
	serverChan3 string // %s = uid, %s = SendKey
	pushPlus    string
	wecomPrefix string
}

func newContentHTTPNotifier() *contentHTTPNotifier {
	return &contentHTTPNotifier{
		client:      &http.Client{Timeout: 15 * time.Second},
		serverChan:  "https://sctapi.ftqq.com/%s.send",
		serverChan3: "https://%s.push.ft07.com/send/%s.send",
		pushPlus:    "https://www.pushplus.plus/send",
		wecomPrefix: "https://qyapi.weixin.qq.com/",
	}
}

var (
	contentNotifyTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
	contentServerChan3Pattern = regexp.MustCompile(`^sctp(\d+)t`)
)

func errContentPush(detail string) error {
	return infraerrors.New(http.StatusBadGateway, "CONTENT_PUSH_FAILED", "微信推送没发出去："+detail+"（Push failed）")
}

func (n *contentHTTPNotifier) Send(ctx context.Context, kind, key string, msg contentNotifyMessage) error {
	key = strings.TrimSpace(key)
	text := msg.Body
	if msg.URL != "" {
		text += "\n\n[打开后台审核](" + msg.URL + ")"
	}
	switch kind {
	case contentNotifyServerChan:
		if !contentNotifyTokenPattern.MatchString(key) {
			return errContentPush("SendKey 格式不对")
		}
		endpoint := fmt.Sprintf(n.serverChan, key)
		if m := contentServerChan3Pattern.FindStringSubmatch(key); m != nil {
			endpoint = fmt.Sprintf(n.serverChan3, m[1], key)
		}
		form := url.Values{"title": {msg.Title}, "desp": {text}}
		return n.post(ctx, endpoint, "application/x-www-form-urlencoded", []byte(form.Encode()), func(r map[string]any) bool { return jsonNumber(r["code"]) == 0 })
	case contentNotifyPushPlus:
		if !contentNotifyTokenPattern.MatchString(key) {
			return errContentPush("Token 格式不对")
		}
		body, _ := json.Marshal(map[string]string{"token": key, "title": msg.Title, "content": text, "template": "markdown"})
		return n.post(ctx, n.pushPlus, "application/json", body, func(r map[string]any) bool { return jsonNumber(r["code"]) == 200 })
	case contentNotifyWecom:
		if !strings.HasPrefix(key, n.wecomPrefix) {
			return ErrContentNotifyURL
		}
		body, _ := json.Marshal(map[string]any{"msgtype": "markdown", "markdown": map[string]string{"content": "**" + msg.Title + "**\n" + text}})
		return n.post(ctx, key, "application/json", body, func(r map[string]any) bool { return jsonNumber(r["errcode"]) == 0 })
	}
	return ErrContentNotify
}

func (n *contentHTTPNotifier) post(ctx context.Context, endpoint, contentType string, body []byte, ok func(map[string]any) bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errContentPush("地址不对")
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := n.client.Do(req)
	if err != nil {
		return errContentPush("连不上推送服务，稍后再试")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var r map[string]any
	if json.Unmarshal(raw, &r) != nil || resp.StatusCode >= 300 || !ok(r) {
		detail := ""
		for _, k := range []string{"message", "msg", "errmsg", "info"} {
			if s, _ := r[k].(string); s != "" {
				detail = s
				break
			}
		}
		if detail == "" {
			detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return errContentPush("推送服务返回：" + truncateRunes(detail, 100) + "，检查 Key 是否正确、今天的免费额度是否用完")
	}
	return nil
}

// jsonNumber reads a JSON number (float64 after decoding); anything else is -1.
func jsonNumber(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}
