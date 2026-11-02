package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
)

// muse2api（github.com/czg86389-hub/muse2api）把 Muse 网页账号反代成 OpenAI 风格的视频接口：
//
//	POST /v1/videos/generations → {"id":"task_x","status":"queued",...}
//	GET  /v1/videos/{id}        → {"status":"queued|processing|completed|failed",
//	                               "url":"/v1/media/x.mp4","video":{"url":"/v1/media/x.mp4"},
//	                               "result":{...},"account":"<muse 账号>","error":"..."}
//
// 路径与 xAI 一致，但状态值和结果地址不同：完成是 completed 而不是 done，结果是 muse2api
// 自己的相对路径，而且不需要鉴权就能下载。Grok 平台的 API Key 账号打开
// accounts.extra.grok_video_muse2api_compat 后，网关在这里把响应改写成 xAI 官方形状，
// 后续的任务绑定、完成计费和内容代理都沿用 Grok 视频的原有逻辑。
const grokVideoMuse2APICompatExtraKey = "grok_video_muse2api_compat"

// IsGrokVideoMuse2APICompat 返回 Grok API Key 账号的上游是否是 muse2api。
func (a *Account) IsGrokVideoMuse2APICompat() bool {
	if a == nil || !a.IsGrok() || a.Type != AccountTypeAPIKey || a.Extra == nil {
		return false
	}
	enabled, _ := a.Extra[grokVideoMuse2APICompatExtraKey].(bool)
	return enabled
}

// muse2api 响应里不能交给用户的字段：结果的直链（不鉴权）、Muse 账号 ID、
// 网关拼给 Muse 的完整提示词。
var muse2APIVideoPrivateKeys = []string{"url", "result", "account", "api_prompt"}

// normalizeMuse2APIVideoBody 把 muse2api 的任务响应改写成 xAI 官方形状：
// status 归一到 pending/done/failed，结果地址只保留在 video.url。
func normalizeMuse2APIVideoBody(body []byte) []byte {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil || root == nil {
		return body
	}

	videoURL := ""
	if video, ok := root["video"].(map[string]any); ok {
		videoURL, _ = video["url"].(string)
	}
	if strings.TrimSpace(videoURL) == "" {
		videoURL, _ = root["url"].(string)
	}
	if strings.TrimSpace(videoURL) == "" {
		if result, ok := root["result"].(map[string]any); ok {
			videoURL, _ = result["url"].(string)
		}
	}
	for _, key := range muse2APIVideoPrivateKeys {
		delete(root, key)
	}

	status, _ := root["status"].(string)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "succeeded", "success", "done":
		root["status"] = "done"
	case "failed", "error", "cancelled", "canceled":
		root["status"] = "failed"
	case "expired":
		root["status"] = "expired"
	case "":
	default:
		root["status"] = "pending"
	}

	if strings.TrimSpace(videoURL) != "" {
		video, ok := root["video"].(map[string]any)
		if !ok {
			video = map[string]any{}
		}
		video["url"] = videoURL
		root["video"] = video
	} else {
		delete(root, "video")
	}

	rewritten, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return rewritten
}

// muse2APIVideoContentURL 把 muse2api 状态里的结果地址（通常是相对路径 /v1/media/x.mp4）
// 解析到账号配置的上游地址上。绝对地址必须与上游同源，避免状态响应把网关引到别的主机。
func muse2APIVideoContentURL(account *Account, statusBody []byte) (string, error) {
	rawURL := strings.TrimSpace(gjson.GetBytes(statusBody, "video.url").String())
	if rawURL == "" {
		return "", fmt.Errorf("muse2api video status has no video url")
	}
	base, err := url.Parse(account.GetGrokMediaBaseURL())
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("muse2api account base url is invalid")
	}
	ref, err := url.Parse(rawURL)
	if err != nil || ref.User != nil {
		return "", fmt.Errorf("muse2api video status returned an invalid video url")
	}
	if ref.IsAbs() || ref.Host != "" {
		if !strings.EqualFold(ref.Scheme, base.Scheme) || !strings.EqualFold(ref.Host, base.Host) {
			return "", fmt.Errorf("muse2api video status returned a video url on another host")
		}
	}
	return base.ResolveReference(ref).String(), nil
}
