//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type withdrawRepoStub struct {
	elig    WithdrawEligibility
	created []NewWithdrawal
	quotas  []float64
	list    []AffiliateWithdrawal
}

func (r *withdrawRepoStub) Eligibility(context.Context, int64, time.Time) (*WithdrawEligibility, error) {
	e := r.elig
	return &e, nil
}

func (r *withdrawRepoStub) Create(_ context.Context, userID int64, in NewWithdrawal, _ time.Time, plan WithdrawPlanFunc) (*AffiliateWithdrawal, error) {
	e := r.elig
	quota, err := plan(&e)
	if err != nil {
		return nil, err
	}
	r.created = append(r.created, in)
	r.quotas = append(r.quotas, quota)
	return &AffiliateWithdrawal{ID: 1, UserID: userID, QuotaAmount: quota, CNYAmount: in.CNYAmount, Method: in.Method,
		Account: in.Account, RealName: in.RealName, Status: WithdrawStatusPending}, nil
}

func (r *withdrawRepoStub) ListByUser(context.Context, int64, int) ([]AffiliateWithdrawal, error) {
	return append([]AffiliateWithdrawal(nil), r.list...), nil
}

func (r *withdrawRepoStub) List(context.Context, WithdrawFilter) ([]AffiliateWithdrawal, int64, error) {
	return nil, 0, nil
}
func (r *withdrawRepoStub) CountPending(context.Context) (int64, error) { return 0, nil }
func (r *withdrawRepoStub) MarkPaid(context.Context, int64, string, int64) (*AffiliateWithdrawal, error) {
	return nil, errors.New("unused")
}
func (r *withdrawRepoStub) Return(context.Context, int64, string, string, int64, int64) (*AffiliateWithdrawal, error) {
	return nil, errors.New("unused")
}

// withdrawTestEncryptor marks ciphertext so tests can tell it apart from plaintext.
type withdrawTestEncryptor struct{}

func (withdrawTestEncryptor) Encrypt(p string) (string, error) { return "enc:" + p, nil }
func (withdrawTestEncryptor) Decrypt(c string) (string, error) {
	if !strings.HasPrefix(c, "enc:") {
		return "", errors.New("not encrypted")
	}
	return strings.TrimPrefix(c, "enc:"), nil
}

func newWithdrawServiceForTest(repo *withdrawRepoStub, st WithdrawSettings) *AffiliateWithdrawService {
	growth := &GrowthService{cached: &GrowthSettings{WithdrawSettings: st}, cachedAt: time.Now()}
	return NewAffiliateWithdrawService(repo, growth, withdrawTestEncryptor{}, nil)
}

func TestWithdrawableCNY(t *testing.T) {
	// $10 of cash rebate that cost ¥72 (ratio 7.2), nothing withdrawn, all still available.
	e := &WithdrawEligibility{AvailableQuota: 10, CashQuota: 10, CashCNY: 72}
	require.Equal(t, 72.0, withdrawableCNY(e))

	// Part already withdrawn.
	e.WithdrawnCNY = 30
	e.AvailableQuota = 10 - 30/7.2
	require.InDelta(t, 42.0, withdrawableCNY(e), 0.01)

	// The rest was moved to balance: only what's still available can be cashed.
	e.AvailableQuota = 2
	require.Equal(t, 14.4, withdrawableCNY(e))

	// Rebates that came from admin top-ups or redeem codes are not cash.
	require.Zero(t, withdrawableCNY(&WithdrawEligibility{AvailableQuota: 50}))
	// Never negative, never rounds up.
	require.Zero(t, withdrawableCNY(&WithdrawEligibility{AvailableQuota: 5, CashQuota: 5, CashCNY: 5, WithdrawnCNY: 9}))
	require.Equal(t, 3.33, withdrawableCNY(&WithdrawEligibility{AvailableQuota: 1, CashQuota: 3, CashCNY: 9.999}))
}

func TestQuotaForCNYUsesAverageRatioAndCapsAtAvailable(t *testing.T) {
	e := &WithdrawEligibility{AvailableQuota: 10, CashQuota: 10, CashCNY: 72}
	require.Equal(t, 5.0, quotaForCNY(e, 36))
	require.Equal(t, 10.0, quotaForCNY(e, 72.004))
}

func TestWithdrawRequestRules(t *testing.T) {
	ctx := context.Background()
	on := WithdrawSettings{Enabled: true, MinCNY: 50, MonthlyLimit: 2}
	elig := WithdrawEligibility{AvailableQuota: 20, CashQuota: 20, CashCNY: 144}
	good := WithdrawRequest{CNYAmount: 100, Method: " Alipay ", Account: " 13800000000 ", RealName: " 张三 "}

	t.Run("disabled", func(t *testing.T) {
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: elig}, WithdrawSettings{MinCNY: 50})
		_, err := svc.Request(ctx, 7, good)
		require.ErrorIs(t, err, ErrWithdrawDisabled)
	})

	t.Run("below minimum", func(t *testing.T) {
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: elig}, on)
		req := good
		req.CNYAmount = 49.99
		_, err := svc.Request(ctx, 7, req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "最低提现")
	})

	t.Run("payee required", func(t *testing.T) {
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: elig}, on)
		req := good
		req.RealName = "  "
		_, err := svc.Request(ctx, 7, req)
		require.ErrorIs(t, err, ErrWithdrawInvalidPayee)
		req = good
		req.Method = "bank"
		_, err = svc.Request(ctx, 7, req)
		require.ErrorIs(t, err, ErrWithdrawInvalidMethod)
	})

	t.Run("pending blocks a second one", func(t *testing.T) {
		e := elig
		e.HasPending = true
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: e}, on)
		_, err := svc.Request(ctx, 7, good)
		require.ErrorIs(t, err, ErrWithdrawPending)
	})

	t.Run("monthly limit", func(t *testing.T) {
		e := elig
		e.MonthCount = 2
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: e}, on)
		_, err := svc.Request(ctx, 7, good)
		require.ErrorIs(t, err, ErrWithdrawMonthlyLimit)

		unlimited := on
		unlimited.MonthlyLimit = 0
		svc = newWithdrawServiceForTest(&withdrawRepoStub{elig: e}, unlimited)
		_, err = svc.Request(ctx, 7, good)
		require.NoError(t, err)
	})

	t.Run("over withdrawable", func(t *testing.T) {
		svc := newWithdrawServiceForTest(&withdrawRepoStub{elig: elig}, on)
		req := good
		req.CNYAmount = 144.01
		_, err := svc.Request(ctx, 7, req)
		require.ErrorIs(t, err, ErrWithdrawOverAvailable)
	})

	t.Run("ok: encrypts payee, deducts quota at the cash ratio", func(t *testing.T) {
		repo := &withdrawRepoStub{elig: elig}
		svc := newWithdrawServiceForTest(repo, on)
		w, err := svc.Request(ctx, 7, good)
		require.NoError(t, err)
		require.Len(t, repo.created, 1)
		require.Equal(t, "alipay", repo.created[0].Method)
		require.Equal(t, "enc:13800000000", repo.created[0].Account)
		require.Equal(t, "enc:张三", repo.created[0].RealName)
		require.InDelta(t, 100/7.2, repo.quotas[0], 1e-6)
		// The response carries plaintext for the user's own confirmation.
		require.Equal(t, "13800000000", w.Account)
		require.Equal(t, "张三", w.RealName)
	})
}

func TestWithdrawStatusPrefillsLastPayee(t *testing.T) {
	repo := &withdrawRepoStub{
		elig: WithdrawEligibility{AvailableQuota: 20, CashQuota: 20, CashCNY: 144, WithdrawnCNY: 0, MonthCount: 1},
		list: []AffiliateWithdrawal{{ID: 3, Method: "wechat", Account: "enc:wx_abc", RealName: "enc:李四", Status: WithdrawStatusPaid}},
	}
	svc := newWithdrawServiceForTest(repo, WithdrawSettings{Enabled: true, MinCNY: 50, MonthlyLimit: 2})
	st, err := svc.Status(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, st.Enabled)
	require.Equal(t, 144.0, st.WithdrawableCNY)
	require.Equal(t, 1, st.MonthUsed)
	require.Equal(t, "wechat", st.LastMethod)
	require.Equal(t, "wx_abc", st.LastAccount)
	require.Equal(t, "李四", st.LastRealName)
	require.Equal(t, "wx_abc", st.Withdrawals[0].Account)
}

func TestParseGrowthSettingsWithdrawDefaults(t *testing.T) {
	s := parseGrowthSettings(map[string]string{})
	require.False(t, s.WithdrawSettings.Enabled, "withdrawals stay off until the admin turns them on")
	require.Equal(t, 50.0, s.WithdrawSettings.MinCNY)
	require.Equal(t, 2, s.WithdrawSettings.MonthlyLimit)

	s = parseGrowthSettings(map[string]string{
		SettingKeyGrowthWithdrawEnabled: "true",
		SettingKeyGrowthWithdrawMinCNY:  "0.2",
		SettingKeyGrowthWithdrawMonthly: "99",
	})
	require.True(t, s.WithdrawSettings.Enabled)
	require.Equal(t, 1.0, s.WithdrawSettings.MinCNY)
	require.Equal(t, 31, s.WithdrawSettings.MonthlyLimit)

	_, err := normalizeWithdrawSettings(WithdrawSettings{MinCNY: 0})
	require.Error(t, err)
	_, err = normalizeWithdrawSettings(WithdrawSettings{MinCNY: 10, MonthlyLimit: -1})
	require.Error(t, err)
	out, err := normalizeWithdrawSettings(WithdrawSettings{Enabled: true, MinCNY: 10.005, MonthlyLimit: 0})
	require.NoError(t, err)
	require.Equal(t, 10.01, out.MinCNY)
}
