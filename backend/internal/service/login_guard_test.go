//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type memLoginGuardCache struct {
	ints map[string]int64
	strs map[string]string
	ttls map[string]time.Duration
}

func newMemLoginGuardCache() *memLoginGuardCache {
	return &memLoginGuardCache{ints: map[string]int64{}, strs: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (m *memLoginGuardCache) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.ints[key]++
	if m.ints[key] == 1 {
		m.ttls[key] = ttl
	}
	return m.ints[key], nil
}
func (m *memLoginGuardCache) Get(_ context.Context, key string) (int64, error) {
	return m.ints[key], nil
}
func (m *memLoginGuardCache) TTL(_ context.Context, key string) (time.Duration, error) {
	return m.ttls[key], nil
}
func (m *memLoginGuardCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.strs[key], m.ttls[key] = value, ttl
	return nil
}
func (m *memLoginGuardCache) Take(_ context.Context, key string) (string, error) {
	v := m.strs[key]
	delete(m.strs, key)
	return v, nil
}
func (m *memLoginGuardCache) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.ints, k)
		delete(m.strs, k)
		delete(m.ttls, k)
	}
	return nil
}

func captchaFlagged(err error) bool {
	var appErr *infraerrors.ApplicationError
	return errors.As(err, &appErr) && appErr.Metadata["captcha_required"] == "true"
}

func TestLoginGuardAsksForCaptchaAfterThreeFailuresFromAnIP(t *testing.T) {
	ctx := context.Background()
	g := NewLoginGuardService(newMemLoginGuardCache())

	for i, mail := range []string{"a@x.com", "b@x.com"} {
		err := g.RecordFailure(ctx, "1.2.3.4", mail)
		require.ErrorIs(t, err, ErrInvalidCredentials)
		require.False(t, captchaFlagged(err), "failure %d", i+1)
	}
	require.False(t, g.CaptchaRequired(ctx, "1.2.3.4", "c@x.com"))

	err := g.RecordFailure(ctx, "1.2.3.4", "c@x.com")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.True(t, captchaFlagged(err))
	require.True(t, g.CaptchaRequired(ctx, "1.2.3.4", "d@x.com"))
	require.False(t, g.CaptchaRequired(ctx, "5.6.7.8", "d@x.com"), "other IPs and emails are unaffected")
}

func TestLoginGuardAsksForCaptchaForAnEmailAttackedFromManyIPs(t *testing.T) {
	ctx := context.Background()
	g := NewLoginGuardService(newMemLoginGuardCache())
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		_ = g.RecordFailure(ctx, ip, "Victim@X.com")
	}
	require.True(t, g.CaptchaRequired(ctx, "9.9.9.9", " victim@x.com "), "email matched case-insensitively")

	g.RecordSuccess(ctx, "victim@x.com")
	require.False(t, g.CaptchaRequired(ctx, "9.9.9.9", "victim@x.com"))
}

func TestLoginGuardAsksEveryoneForCaptchaDuringASiteWideSurge(t *testing.T) {
	ctx := context.Background()
	cache := newMemLoginGuardCache()
	g := NewLoginGuardService(cache)
	cache.ints[loginGuardSiteKey()] = loginCaptchaSiteWideFails
	require.True(t, g.CaptchaRequired(ctx, "8.8.8.8", "new@x.com"))
}

func TestLoginGuardBlocksAnIPAfterTwentyFailures(t *testing.T) {
	ctx := context.Background()
	g := NewLoginGuardService(newMemLoginGuardCache())
	for i := 0; i < loginBlockAfterIPFails-1; i++ {
		_ = g.RecordFailure(ctx, "6.6.6.6", "u@x.com")
	}
	require.NoError(t, g.CheckBlocked(ctx, "6.6.6.6"))

	_ = g.RecordFailure(ctx, "6.6.6.6", "u@x.com")
	err := g.CheckBlocked(ctx, "6.6.6.6")
	require.Error(t, err)
	var appErr *infraerrors.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "LOGIN_TOO_MANY_ATTEMPTS", appErr.Reason)
	require.Equal(t, int32(429), appErr.Code)
	require.Equal(t, "15", appErr.Metadata["retry_after_minutes"])
	require.Contains(t, appErr.Message, "15 分钟")
	require.NoError(t, g.CheckBlocked(ctx, "7.7.7.7"))
}

func TestLoginGuardCaptchaIsOneTimeAndWrongAnswersCount(t *testing.T) {
	ctx := context.Background()
	cache := newMemLoginGuardCache()
	g := NewLoginGuardService(cache)

	c, err := g.NewCaptcha(ctx)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(c.Image, "data:image/png;base64,"))
	code := cache.strs[loginCaptchaKey(c.ID)]
	require.Len(t, code, loginCaptchaLength)
	require.Equal(t, loginCaptchaTTL, cache.ttls[loginCaptchaKey(c.ID)])

	require.NoError(t, g.VerifyCaptcha(ctx, "1.2.3.4", c.ID, " "+strings.ToLower(code)+" "))
	err = g.VerifyCaptcha(ctx, "1.2.3.4", c.ID, code)
	require.ErrorIs(t, err, ErrLoginCaptchaInvalid, "a captcha works once")
	require.True(t, captchaFlagged(err))
	require.EqualValues(t, 1, cache.ints[loginGuardIPKey("1.2.3.4")], "a wrong captcha counts against the IP")

	err = g.VerifyCaptcha(ctx, "1.2.3.4", "", "")
	require.ErrorIs(t, err, ErrLoginCaptchaRequired)
	require.True(t, captchaFlagged(err))
}

func TestRenderLoginCaptchaIsAPNG(t *testing.T) {
	data, err := renderLoginCaptcha("AB7K")
	require.NoError(t, err)
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Equal(t, loginCaptchaWidth, img.Bounds().Dx())
	require.Equal(t, loginCaptchaHeight, img.Bounds().Dy())
}

func TestLoginCaptchaCodesUseTheUnambiguousAlphabet(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := randomLoginCaptchaCode()
		require.NoError(t, err)
		for _, ch := range code {
			require.True(t, strings.ContainsRune(loginCaptchaAlphabet, ch), code)
		}
	}
}
