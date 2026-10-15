package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 公众号 cgi-bin 接口：stable_token（带缓存 + singleflight）、带参二维码、用户信息。

// FetchUnionID reads the follower's unionid (may be empty) with cgi-bin/user/info.
func (s *WeChatMPLoginService) FetchUnionID(ctx context.Context, openid string) (string, error) {
	cfg, err := s.Config(ctx)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("openid", strings.TrimSpace(openid))
	query.Set("lang", "zh_CN")
	raw, err := s.callAPI(ctx, cfg, http.MethodGet, "/cgi-bin/user/info", query, nil)
	if err != nil {
		return "", err
	}
	var info struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return "", infraerrors.ServiceUnavailable("WECHAT_MP_API_ERROR", "微信接口返回异常，请稍后再试（invalid user info response）").WithCause(err)
	}
	return strings.TrimSpace(info.UnionID), nil
}

type weChatMPAPIError struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func weChatMPTokenInvalid(code int64) bool {
	return code == 40001 || code == 42001 || code == 40014
}

func weChatMPAPIErrorFor(code int64, msg string) error {
	if code == 40164 {
		return ErrWeChatMPIPNotWhitelisted
	}
	return infraerrors.ServiceUnavailable("WECHAT_MP_API_ERROR",
		fmt.Sprintf("微信接口暂时不可用，请稍后再试（WeChat API error %d: %s）", code, strings.TrimSpace(msg)))
}

// callAPI calls a cgi-bin endpoint with the access token. When WeChat says the token is invalid or
// expired it refreshes once (see refreshAccessToken) and retries. Returns the body when errcode is 0.
func (s *WeChatMPLoginService) callAPI(ctx context.Context, cfg WeChatMPConfig, method, path string, query url.Values, body any) ([]byte, error) {
	token, err := s.accessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		raw, apiErr, err := s.doAPI(ctx, method, path, token, query, body)
		if err != nil {
			return nil, err
		}
		if apiErr.ErrCode == 0 {
			return raw, nil
		}
		if weChatMPTokenInvalid(apiErr.ErrCode) && attempt == 0 {
			token, err = s.refreshAccessToken(ctx, cfg, token)
			if err != nil {
				return nil, err
			}
			continue
		}
		return nil, weChatMPAPIErrorFor(apiErr.ErrCode, apiErr.ErrMsg)
	}
}

func (s *WeChatMPLoginService) doAPI(ctx context.Context, method, path, token string, query url.Values, body any) ([]byte, weChatMPAPIError, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	if token != "" {
		q.Set("access_token", token)
	}
	endpoint := s.apiBaseURL + path
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, weChatMPAPIError{}, fmt.Errorf("encode wechat request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, weChatMPAPIError{}, fmt.Errorf("build wechat request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, weChatMPAPIError{}, infraerrors.ServiceUnavailable("WECHAT_MP_API_UNREACHABLE", "连不上微信接口，请稍后再试（WeChat API unreachable）").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, weChatMPAPIError{}, infraerrors.ServiceUnavailable("WECHAT_MP_API_UNREACHABLE", "连不上微信接口，请稍后再试（WeChat API read failed）").WithCause(err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, weChatMPAPIError{}, infraerrors.ServiceUnavailable("WECHAT_MP_API_ERROR", fmt.Sprintf("微信接口暂时不可用，请稍后再试（WeChat API status %d）", resp.StatusCode))
	}
	var apiErr weChatMPAPIError
	_ = json.Unmarshal(raw, &apiErr)
	return raw, apiErr, nil
}

func (s *WeChatMPLoginService) checkTokenConfig(cfg WeChatMPConfig) error {
	if err := s.ready(); err != nil {
		return err
	}
	if !cfg.MPReady {
		return infraerrors.ServiceUnavailable("WECHAT_MP_NOT_CONFIGURED", "公众号的 AppID / AppSecret 还没配置（WeChat official account not configured）")
	}
	return nil
}

func (s *WeChatMPLoginService) cachedAccessToken(ctx context.Context, appID string) string {
	token, ok, err := s.store.Get(ctx, weChatMPAccessTokenKey(appID))
	if err != nil || !ok {
		return ""
	}
	return token
}

// accessToken returns the cached stable access token; concurrent misses share one fetch.
func (s *WeChatMPLoginService) accessToken(ctx context.Context, cfg WeChatMPConfig) (string, error) {
	if err := s.checkTokenConfig(cfg); err != nil {
		return "", err
	}
	if token := s.cachedAccessToken(ctx, cfg.AppID); token != "" {
		return token, nil
	}
	v, err, _ := s.tokenSF.Do("get:"+cfg.AppID, func() (any, error) {
		if token := s.cachedAccessToken(ctx, cfg.AppID); token != "" {
			return token, nil
		}
		return s.fetchAccessToken(ctx, cfg, false)
	})
	if err != nil {
		return "", err
	}
	token, _ := v.(string)
	return token, nil
}

// refreshAccessToken replaces a token WeChat rejected. It force-refreshes only when the cache still
// holds that same token; if another request already refreshed it, the new cached token is used.
// Concurrent refreshes share one call (force_refresh is limited to a few per day).
func (s *WeChatMPLoginService) refreshAccessToken(ctx context.Context, cfg WeChatMPConfig, failed string) (string, error) {
	if err := s.checkTokenConfig(cfg); err != nil {
		return "", err
	}
	v, err, _ := s.tokenSF.Do("refresh:"+cfg.AppID, func() (any, error) {
		if token := s.cachedAccessToken(ctx, cfg.AppID); token != "" && token != failed {
			return token, nil
		}
		return s.fetchAccessToken(ctx, cfg, true)
	})
	if err != nil {
		return "", err
	}
	token, _ := v.(string)
	return token, nil
}

func (s *WeChatMPLoginService) fetchAccessToken(ctx context.Context, cfg WeChatMPConfig, force bool) (string, error) {
	body := map[string]any{
		"grant_type":    "client_credential",
		"appid":         cfg.AppID,
		"secret":        cfg.AppSecret,
		"force_refresh": force,
	}
	raw, apiErr, err := s.doAPI(ctx, http.MethodPost, "/cgi-bin/stable_token", "", nil, body)
	if err != nil {
		return "", err
	}
	if apiErr.ErrCode != 0 {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] stable_token error %d: %s", apiErr.ErrCode, apiErr.ErrMsg)
		return "", weChatMPAPIErrorFor(apiErr.ErrCode, apiErr.ErrMsg)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil || strings.TrimSpace(tok.AccessToken) == "" {
		return "", infraerrors.ServiceUnavailable("WECHAT_MP_API_ERROR", "微信没有返回 access_token，请稍后再试（missing access_token）")
	}
	ttl := time.Duration(tok.ExpiresIn)*time.Second - weChatMPAccessTokenEarlyExpiry
	if ttl < time.Minute {
		ttl = time.Minute
	}
	if err := s.store.Set(ctx, weChatMPAccessTokenKey(cfg.AppID), tok.AccessToken, ttl); err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] cache access token failed: %v", err)
	}
	return tok.AccessToken, nil
}
