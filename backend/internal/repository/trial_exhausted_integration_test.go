//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestTrialExhaustedRepositoryDue(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewTrialExhaustedRepository(integrationDB)
	acc := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "trial-acc-" + sfx}))
	mk := func(tag string, balance float64, used bool) *service.User {
		u := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "tr-" + tag + "-" + sfx + "@test.local", Username: "tr" + tag + sfx[len(sfx)-6:]}))
		_, err := integrationDB.ExecContext(ctx, `UPDATE users SET balance = $1 WHERE id = $2`, balance, u.ID)
		require.NoError(t, err)
		if used {
			key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: u.ID, Key: "sk-tr-" + tag + sfx})
			_, err = integrationEntClient.UsageLog.Create().SetUserID(u.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).
				SetRequestID("tr-" + tag + sfx).SetModel("gpt-5").SetCreatedAt(time.Now()).Save(ctx)
			require.NoError(t, err)
		}
		return u
	}
	spent := mk("spent", 0.05, true)
	unused := mk("unused", 0.05, false) // never called: the activation email's job
	rich := mk("rich", 1.5, true)       // still has credit
	payer := mk("payer", 0.05, true)    // already paid once
	n := time.Now().UnixNano()
	_, err := integrationEntClient.PaymentOrder.Create().
		SetUserID(payer.ID).SetUserEmail(payer.Email).SetUserName(payer.Username).
		SetAmount(10).SetPayAmount(10).SetFeeRate(0).
		SetRechargeCode(fmt.Sprintf("TR-%d", n)).SetOutTradeNo(fmt.Sprintf("tr_%d", n)).
		SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("t").SetOrderType(payment.OrderTypeBalance).
		SetStatus(service.OrderStatusCompleted).SetPaidAt(time.Now()).SetExpiresAt(time.Now()).
		SetClientIP("127.0.0.1").SetSrcHost("test").Save(ctx)
	require.NoError(t, err)

	due, err := repo.DueTrialUsers(ctx, time.Now().Add(-time.Hour), 0.3, 1000)
	require.NoError(t, err)
	ids := map[int64]bool{}
	for _, u := range due {
		ids[u.UserID] = true
	}
	require.True(t, ids[spent.ID])
	for _, u := range []*service.User{unused, rich, payer} {
		require.False(t, ids[u.ID], u.Email)
	}
}
