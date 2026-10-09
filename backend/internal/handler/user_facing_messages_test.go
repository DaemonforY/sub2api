//go:build unit

package handler

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBillingErrorDetailsInsufficientBalanceLinksToRecharge(t *testing.T) {
	status, code, msg, retry := billingErrorDetails(service.ErrInsufficientBalance)
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, "billing_error", code)
	require.Zero(t, retry)
	require.Contains(t, msg, "账户余额不足")
	require.Contains(t, msg, service.RechargeURL)
	require.Contains(t, msg, "insufficient balance", "ops classifiers match on the English text")
}
