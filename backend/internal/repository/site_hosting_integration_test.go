//go:build integration

package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type siteSubs struct{ active map[int64]bool }

func (s *siteSubs) ListActiveByUserID(_ context.Context, userID int64) ([]service.UserSubscription, error) {
	if s.active[userID] {
		return []service.UserSubscription{{ID: 1}}, nil
	}
	return nil, nil
}

func TestSiteHostingLifecycle(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "sites-" + suffix + "@test.local", Username: "st" + suffix[len(suffix)-6:], Balance: 7})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "sites2-" + suffix + "@test.local", Username: "su" + suffix[len(suffix)-6:]})
	repo := NewSiteHostingRepository(integrationDB)
	subs := &siteSubs{active: map[int64]bool{user.ID: true}}
	dir := t.TempDir()
	cfg := service.DefaultSiteHostingConfig("s.example.test")
	cfg.MaxPerUser, cfg.FreePerUser, cfg.ExtraPrice, cfg.GraceDays, cfg.RetentionDays = 3, 1, 5, 7, 30
	svc := service.NewSiteHostingService(repo, subs, nil, nil, cfg, dir, "https://hivegpt.cn")
	page := func(text string) service.SiteUpload {
		return service.SiteUpload{Title: "我的页面", FileName: "index.html", Data: []byte("<html><body>" + text + "</body></html>")}
	}
	reason := func(err error) string { return infraerrors.Reason(err) }

	_, err := svc.Create(ctx, other.ID, page("x"))
	require.Equal(t, "SITE_SUBSCRIPTION_REQUIRED", reason(err))

	free, err := svc.Create(ctx, user.ID, page("first"))
	require.NoError(t, err)
	require.False(t, free.Paid)
	require.Equal(t, "https://"+free.Name+".s.example.test", free.URL)
	require.FileExists(t, filepath.Join(dir, fmt.Sprint(free.ID), "v1", "index.html"))

	paid, err := svc.Create(ctx, user.ID, page("second"))
	require.NoError(t, err)
	require.True(t, paid.Paid, "beyond the free allowance a site costs 5 / 30 days")
	require.NotNil(t, paid.PaidUntil)
	mine, err := svc.Mine(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 2, mine.Quota.Balance, 1e-9)
	require.Len(t, mine.Charges, 1)
	require.Equal(t, 1, mine.Quota.FreeUsed)

	_, err = svc.Create(ctx, user.ID, page("third"))
	require.Equal(t, "SITE_INSUFFICIENT_BALANCE", reason(err), "balance 2 < 5")
	mine, _ = svc.Mine(ctx, user.ID)
	require.Len(t, mine.Sites, 2, "a site that could not be paid for is not kept")

	// New versions replace the old ones; only the last three stay on disk.
	for i := 0; i < 4; i++ {
		_, err = svc.Update(ctx, user.ID, free.ID, page(fmt.Sprintf("v%d", i+2)))
		require.NoError(t, err)
	}
	updated, _ := repo.GetSite(ctx, free.ID)
	require.Equal(t, 5, updated.Version)
	_, err = os.Stat(filepath.Join(dir, fmt.Sprint(free.ID), "v2"))
	require.True(t, os.IsNotExist(err))
	require.DirExists(t, filepath.Join(dir, fmt.Sprint(free.ID), "v3"))
	_, err = svc.Update(ctx, other.ID, free.ID, page("hijack"))
	require.Equal(t, "SITE_NOT_FOUND", reason(err), "only the owner can update")

	// Served by host.
	w := httptest.NewRecorder()
	name, ok := svc.SiteNameFromHost(free.Name + ".s.example.test")
	require.True(t, ok)
	svc.ServeSite(w, httptest.NewRequest(http.MethodGet, "/", nil), name)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "v5")
	require.True(t, svc.KnownSite(ctx, free.Name+".s.example.test"))
	require.False(t, svc.KnownSite(ctx, "nosuch12.s.example.test"))

	// Renewal: the paid period ended and the balance (2) cannot cover it → offline until paid.
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET paid_until = NOW() - interval '1 hour' WHERE id = $1`, paid.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Maintain(ctx))
	got, _ := repo.GetSite(ctx, paid.ID)
	require.Equal(t, service.SiteStatusUnpaid, got.Status)
	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET balance = 10 WHERE id = $1`, user.ID)
	require.NoError(t, err)
	_, err = svc.Renew(ctx, user.ID, paid.ID)
	require.NoError(t, err)
	got, _ = repo.GetSite(ctx, paid.ID)
	require.Equal(t, service.SiteStatusActive, got.Status)
	require.True(t, got.PaidUntil.After(time.Now().Add(29*24*time.Hour)))

	// Deleting the free site frees its slot: at the next renewal the paid site becomes free.
	require.NoError(t, svc.Delete(ctx, user.ID, free.ID))
	_, err = os.Stat(filepath.Join(dir, fmt.Sprint(free.ID)))
	require.True(t, os.IsNotExist(err))
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET paid_until = NOW() - interval '1 hour' WHERE id = $1`, paid.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Maintain(ctx))
	got, _ = repo.GetSite(ctx, paid.ID)
	require.False(t, got.Paid)
	require.Nil(t, got.PaidUntil)

	// Subscription ends: grace period, then offline, then back when renewed; deleted after retention.
	subs.active[user.ID] = false
	require.NoError(t, svc.Maintain(ctx))
	got, _ = repo.GetSite(ctx, paid.ID)
	require.Equal(t, service.SiteStatusActive, got.Status, "still up during the grace period")
	require.NotNil(t, got.LapsedAt)
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET lapsed_at = NOW() - interval '8 days' WHERE user_id = $1`, user.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Maintain(ctx))
	got, _ = repo.GetSite(ctx, paid.ID)
	require.Equal(t, service.SiteStatusLapsed, got.Status)
	subs.active[user.ID] = true
	require.NoError(t, svc.Maintain(ctx))
	got, _ = repo.GetSite(ctx, paid.ID)
	require.Equal(t, service.SiteStatusActive, got.Status)
	require.Nil(t, got.LapsedAt)
	subs.active[user.ID] = false
	require.NoError(t, svc.Maintain(ctx))
	_, err = integrationDB.ExecContext(ctx, `UPDATE sites SET lapsed_at = NOW() - interval '40 days' WHERE user_id = $1`, user.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Maintain(ctx))
	got, _ = repo.GetSite(ctx, paid.ID)
	require.Nil(t, got, "deleted after grace + retention")

	// Reports and admin take-downs.
	subs.active[user.ID] = true
	site, err := svc.Create(ctx, user.ID, page("again"))
	require.NoError(t, err)
	require.NoError(t, svc.Report(ctx, site.Name, "phishing", "仿冒银行登录页", "a@b.c", "203.0.113.9"))
	require.Equal(t, "SITE_REPORT_INVALID", reason(svc.Report(ctx, site.Name, "boring", "", "", "203.0.113.9")))
	reports, total, err := svc.AdminReports(ctx, "open", 1, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.Equal(t, user.Email, reports[0].OwnerEmail)
	require.NoError(t, svc.AdminSetReportStatus(ctx, reports[0].ID, "resolved"))
	require.NoError(t, svc.AdminSetStatus(ctx, site.ID, service.SiteStatusDisabled, "钓鱼页面"))
	_, err = svc.Update(ctx, user.ID, site.ID, page("back"))
	require.Equal(t, "SITE_DISABLED", reason(err))
	list, total, err := svc.AdminList(ctx, service.SiteListQuery{Keyword: "sites-" + suffix, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "钓鱼页面", list[0].StatusReason)
	require.NoError(t, svc.AdminDelete(ctx, site.ID))
}
