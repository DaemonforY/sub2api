//go:build unit

package handler

import (
	"strings"
	"testing"

	pkgerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Localized gateway errors still read as local business limits to the ops classifier, which
// matches lowercased substrings of the original texts.
func TestLocalizedLimitErrorsStillClassifyAsBusinessLimits(t *testing.T) {
	for _, err := range []error{
		service.ErrAPIKeyRateLimit5hExceeded,
		service.ErrAPIKeyRateLimit1dExceeded,
		service.ErrAPIKeyRateLimit7dExceeded,
		service.ErrUserPlatformDailyQuotaExhausted,
		service.ErrUserPlatformWeeklyQuotaExhausted,
		service.ErrUserPlatformMonthlyQuotaExhausted,
		service.ErrSubscriptionInvalid,
		service.ErrDailyLimitExceeded,
		service.ErrWeeklyLimitExceeded,
		service.ErrMonthlyLimitExceeded,
	} {
		msg := pkgerrors.Message(err)
		require.True(t, isOpsLocalBusinessLimitError("", strings.ToLower(msg)), msg)
		require.Contains(t, msg, "（", "Chinese text with the English kept in parentheses: %s", msg)
	}
}
