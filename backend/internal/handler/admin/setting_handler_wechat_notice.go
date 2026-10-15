package admin

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// wechatNoticeTester sends a sample 微信提醒 to a user's own bound WeChat.
type wechatNoticeTester interface {
	SendTest(ctx context.Context, event string, callerUserID int64) (*service.WeChatNoticeTestResult, error)
}

// SetWeChatTemplateNotices enables the 微信提醒 test endpoint.
func (h *SettingHandler) SetWeChatTemplateNotices(tester wechatNoticeTester) {
	h.wechatNotices = tester
}

type testWeChatTemplateNoticeRequest struct {
	Event string `json:"event"`
}

// TestWeChatTemplateNotice sends a sample of one 微信提醒 to the calling admin's own bound WeChat
// (never to anyone else) and returns WeChat's answer.
// POST /api/v1/admin/wechat/template-notices/test {event}
func (h *SettingHandler) TestWeChatTemplateNotice(c *gin.Context) {
	if h.wechatNotices == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("WECHAT_NOTICE_NOT_READY", "微信提醒未就绪（WeChat notices not ready）"))
		return
	}
	var req testWeChatTemplateNoticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数不正确（Invalid request）")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorFrom(c, infraerrors.Unauthorized("UNAUTHORIZED", "请先登录（authentication required）"))
		return
	}
	result, err := h.wechatNotices.SendTest(c.Request.Context(), strings.TrimSpace(req.Event), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
