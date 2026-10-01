//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type imageToolsRepoStub struct {
	freeUsed int
	balance  float64
	charges  []ImageToolUse
}

func (r *imageToolsRepoStub) CountFreeSince(context.Context, int64, time.Time) (int, error) {
	return r.freeUsed, nil
}
func (r *imageToolsRepoStub) Balance(context.Context, int64) (float64, error) { return r.balance, nil }
func (r *imageToolsRepoStub) Charge(_ context.Context, use ImageToolUse, freeDaily int, _ time.Time) (bool, error) {
	r.charges = append(r.charges, use)
	if use.Subscribed && r.freeUsed < freeDaily {
		r.freeUsed++
		return true, nil
	}
	if r.balance < use.Price {
		return false, ErrInsufficientBalance
	}
	r.balance -= use.Price
	return false, nil
}

func (r *imageToolsRepoStub) ListUses(context.Context, ImageToolUseQuery) ([]ImageToolUseRecord, int64, error) {
	return nil, 0, nil
}
func (r *imageToolsRepoStub) Stats(context.Context, int64, time.Time) ([]ImageToolStat, error) {
	return nil, nil
}

type imageToolSettingsStub struct{ values map[string]string }

func (s *imageToolSettingsStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := s.values[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}
func (s *imageToolSettingsStub) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		s.values[k] = v
	}
	return nil
}

type imageToolSubsStub struct{ active bool }

func (s imageToolSubsStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	if s.active {
		return []UserSubscription{{ID: 1}}, nil
	}
	return nil, nil
}

type balanceCacheStub struct{ invalidated int }

func (c *balanceCacheStub) InvalidateUserBalance(context.Context, int64) error {
	c.invalidated++
	return nil
}

func newImageToolsTest(t *testing.T, repo *imageToolsRepoStub, subscribed bool, handler http.HandlerFunc) (*ImageToolsService, *balanceCacheStub, *[]string) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	cache := &balanceCacheStub{}
	svc := NewImageToolsService(repo, imageToolSubsStub{active: subscribed}, cache, ImageToolsConfig{BaseURL: srv.URL + "/", PriceRemoveBg: 0.02, PriceUpscale: 0.05, FreeDaily: 3})
	return svc, cache, &paths
}

func okImage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write([]byte("png"))
}

func TestImageToolsSubscribersGetFreeRunsThenPay(t *testing.T) {
	repo := &imageToolsRepoStub{freeUsed: 2, balance: 1}
	svc, cache, paths := newImageToolsTest(t, repo, true, okImage)
	ctx := context.Background()

	res, err := svc.Run(ctx, 7, 9, ImageToolRemoveBg, ImageToolOptions{}, []byte("img"))
	require.NoError(t, err)
	require.True(t, res.Free)
	require.Zero(t, res.Cost)
	require.Equal(t, 0, res.FreeLeft)
	require.Equal(t, []byte("png"), res.Data)
	require.Equal(t, "image/png", res.ContentType)

	res, err = svc.Run(ctx, 7, 9, ImageToolUpscale, ImageToolOptions{Scale: 2}, []byte("img"))
	require.NoError(t, err)
	require.False(t, res.Free)
	require.Equal(t, 0.05, res.Cost)
	require.InDelta(t, 0.95, repo.balance, 1e-9)
	require.Equal(t, 1, cache.invalidated, "a paid run refreshes the cached balance")
	require.Equal(t, []string{"/remove-bg?model=isnet", "/upscale?scale=2"}, *paths)
	require.Equal(t, int64(9), repo.charges[1].APIKeyID)
}

func TestImageToolsWithoutBalanceOrSubscriptionNeverProcess(t *testing.T) {
	repo := &imageToolsRepoStub{balance: 0.01}
	svc, _, paths := newImageToolsTest(t, repo, false, okImage)
	_, err := svc.Run(context.Background(), 7, 0, ImageToolRemoveBg, ImageToolOptions{}, []byte("img"))
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, infraerrors.Code(err))
	require.Contains(t, infraerrors.Message(err), "余额不足")
	require.Empty(t, *paths, "no balance: the image is not processed")
	require.Empty(t, repo.charges)

	q, err := svc.Quota(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, q.Subscribed)
	require.Zero(t, q.FreeLeft)
	require.Equal(t, 0.05, q.Prices[ImageToolUpscale])
}

func TestImageToolsFailuresAreNotCharged(t *testing.T) {
	repo := &imageToolsRepoStub{balance: 5}
	svc, _, _ := newImageToolsTest(t, repo, false, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upscale" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"图片长边已经有 4096px"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := svc.Run(context.Background(), 7, 0, ImageToolUpscale, ImageToolOptions{}, []byte("img"))
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Contains(t, infraerrors.Message(err), "4096px")
	_, err = svc.Run(context.Background(), 7, 0, ImageToolRemoveBg, ImageToolOptions{Model: "u2net"}, []byte("img"))
	require.ErrorIs(t, err, ErrImageToolsFailed)
	require.Empty(t, repo.charges)
	require.Equal(t, 5.0, repo.balance)
}

func TestImageToolsRejectBadInput(t *testing.T) {
	svc, _, paths := newImageToolsTest(t, &imageToolsRepoStub{balance: 5}, true, okImage)
	ctx := context.Background()
	_, err := svc.Run(ctx, 7, 0, ImageToolUpscale, ImageToolOptions{Scale: 3}, []byte("img"))
	require.ErrorIs(t, err, ErrImageToolsBadTool)
	_, err = svc.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{Model: "bria"}, []byte("img"))
	require.ErrorIs(t, err, ErrImageToolsBadTool)
	_, err = svc.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{}, nil)
	require.ErrorIs(t, err, ErrImageToolsNoImage)
	_, err = svc.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{}, make([]byte, ImageToolMaxInputBytes+1))
	require.ErrorIs(t, err, ErrImageToolsTooLarge)
	require.Empty(t, *paths)

	disabled := NewImageToolsService(&imageToolsRepoStub{}, nil, nil, ImageToolsConfig{})
	_, err = disabled.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{}, []byte("img"))
	require.ErrorIs(t, err, ErrImageToolsDisabled)
}

func TestImageToolsAdminSettingsOverrideTheConfig(t *testing.T) {
	repo := &imageToolsRepoStub{freeUsed: 1, balance: 1}
	svc, _, _ := newImageToolsTest(t, repo, true, okImage)
	settings := &imageToolSettingsStub{values: map[string]string{}}
	svc.WithSettings(settings)
	ctx := context.Background()

	got := svc.Settings(ctx)
	require.Equal(t, ImageToolsSettings{Enabled: true, PriceRemoveBg: 0.02, PriceUpscale: 0.05, FreeDaily: 3, ServiceConfigured: true}, got, "defaults come from the config")

	_, err := svc.SaveSettings(ctx, ImageToolsSettingsInput{Enabled: true, PriceRemoveBg: -1, PriceUpscale: 0.05, FreeDaily: 3})
	require.ErrorIs(t, err, ErrImageToolsInvalidPrice)
	_, err = svc.SaveSettings(ctx, ImageToolsSettingsInput{Enabled: true, PriceRemoveBg: 0.1, PriceUpscale: 0.05, FreeDaily: 5000})
	require.ErrorIs(t, err, ErrImageToolsInvalidFreeDaily)

	got, err = svc.SaveSettings(ctx, ImageToolsSettingsInput{Enabled: true, PriceRemoveBg: 0.1, PriceUpscale: 0.3, FreeDaily: 1})
	require.NoError(t, err)
	require.Equal(t, 0.1, got.PriceRemoveBg)
	q, err := svc.Quota(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 0, q.FreeLeft, "the new allowance applies at once")
	require.Equal(t, 0.3, q.Prices[ImageToolUpscale])
	res, err := svc.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{}, []byte("img"))
	require.NoError(t, err)
	require.Equal(t, 0.1, res.Cost)

	_, err = svc.SaveSettings(ctx, ImageToolsSettingsInput{Enabled: false, PriceRemoveBg: 0.1, PriceUpscale: 0.3, FreeDaily: 1})
	require.NoError(t, err)
	_, err = svc.Run(ctx, 7, 0, ImageToolRemoveBg, ImageToolOptions{}, []byte("img"))
	require.ErrorIs(t, err, ErrImageToolsDisabled, "admins can switch the tools off")
	q, _ = svc.Quota(ctx, 7)
	require.False(t, q.Enabled)
}
