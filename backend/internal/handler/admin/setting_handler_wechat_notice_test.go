//go:build unit

package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeWeChatNoticeTester struct {
	event  string
	userID int64
	calls  int
}

func (f *fakeWeChatNoticeTester) SendTest(_ context.Context, event string, callerUserID int64) (*service.WeChatNoticeTestResult, error) {
	f.calls++
	f.event, f.userID = event, callerUserID
	return &service.WeChatNoticeTestResult{ErrCode: 0, ErrMsg: "ok", Message: "发送成功，请在微信里查看（Sent）"}, nil
}

func TestTestWeChatTemplateNoticeSendsOnlyToCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tester := &fakeWeChatNoticeTester{}
	h := &SettingHandler{}
	h.SetWeChatTemplateNotices(tester)

	run := func(body string, subject *middleware2.AuthSubject) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/wechat/template-notices/test", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if subject != nil {
			c.Set(string(middleware2.ContextKeyUser), *subject)
		}
		h.TestWeChatTemplateNotice(c)
		return rec
	}

	// A user_id in the body is ignored: the sample always goes to the caller.
	rec := run(`{"event":"balance_low","user_id":999,"openid":"someone-else"}`, &middleware2.AuthSubject{UserID: 77})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, 1, tester.calls)
	require.Equal(t, int64(77), tester.userID)
	require.Equal(t, "balance_low", tester.event)
	require.Contains(t, rec.Body.String(), "发送成功")

	rec = run(`{"event":"balance_low"}`, nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, 1, tester.calls)
}
