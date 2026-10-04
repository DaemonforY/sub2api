//go:build unit

package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// A course bought inside WeChat goes through the OAuth round trip: the course must survive it.
func TestApplyWeChatPaymentResumeClaimsKeepsCourse(t *testing.T) {
	req := CreateOrderRequest{PaymentType: payment.TypeWxpay}
	require.NoError(t, applyWeChatPaymentResumeClaims(&req, &service.WeChatPaymentResumeClaims{
		OpenID: "openid-1", PaymentType: payment.TypeWxpay, OrderType: payment.OrderTypeCourse, CourseID: 42,
	}))
	require.Equal(t, payment.OrderTypeCourse, req.OrderType)
	require.Equal(t, int64(42), req.CourseID)

	raw, err := encodeWeChatPaymentOAuthContext(wechatPaymentOAuthContext{PaymentType: payment.TypeWxpay, OrderType: payment.OrderTypeCourse, CourseID: 42})
	require.NoError(t, err)
	ctx, err := decodeWeChatPaymentOAuthContext(raw)
	require.NoError(t, err)
	require.Equal(t, int64(42), ctx.CourseID)
}
