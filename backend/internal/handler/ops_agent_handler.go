package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// OpsAgentHandler serves the ops assistant (运维助手) under /api/v1/admin/ops-agent.
type OpsAgentHandler struct {
	svc *service.OpsAgentService
}

func NewOpsAgentHandler(svc *service.OpsAgentService) *OpsAgentHandler {
	return &OpsAgentHandler{svc: svc}
}

// Chat POST /admin/ops-agent/chat {messages} — streams {"delta"} and {"tool": label} events, then
// {"done":true,"model"} or {"error"}.
func (h *OpsAgentHandler) Chat(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.OpsAgentChatInput
	if !bindLearn(c, &in) {
		return
	}
	started := false
	send := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", b); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	start := func() {
		if !started {
			started = true
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			c.Status(http.StatusOK)
		}
	}
	out, err := h.svc.Chat(c.Request.Context(), subject.UserID, in, func(delta string) error {
		start()
		return send(gin.H{"delta": delta})
	}, func(label string) error {
		start()
		return send(gin.H{"tool": label})
	})
	if !started {
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, gin.H{"done": true, "model": out.Model})
		return
	}
	if err != nil {
		_ = send(gin.H{"error": infraerrors.Message(err)})
		return
	}
	_ = send(gin.H{"done": true, "model": out.Model})
}

// Settings GET /admin/ops-agent/settings
func (h *OpsAgentHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// SaveSettings PUT /admin/ops-agent/settings — the key must be the signed-in admin's own.
func (h *OpsAgentHandler) SaveSettings(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.OpsAgentSettings
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), subject.UserID, in)
	learnReply(c, out, err)
}

// Keys GET /admin/ops-agent/keys — the signed-in admin's own GPT keys.
func (h *OpsAgentHandler) Keys(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keys, err := h.svc.AdminKeys(c.Request.Context(), subject.UserID)
	learnReply(c, keys, err)
}

// TestUpstream POST /admin/ops-agent/upstream/test — reads the upstream site's last 15 minutes.
func (h *OpsAgentHandler) TestUpstream(c *gin.Context) {
	out, err := h.svc.TestUpstream(c.Request.Context())
	learnReply(c, out, err)
}

// Runs GET /admin/ops-agent/runs?page=
func (h *OpsAgentHandler) Runs(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	runs, total, err := h.svc.Runs(c.Request.Context(), page)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": runs, "total": total})
}
