package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AssistantHandler serves the homepage support assistant (/api/v1/assistant, open to visitors) and
// its admin settings (/api/v1/admin/assistant).
type AssistantHandler struct {
	svc *service.AssistantService
}

func NewAssistantHandler(svc *service.AssistantService) *AssistantHandler {
	return &AssistantHandler{svc: svc}
}

// SetPages gives the assistant the built learning-site pages.
func (h *AssistantHandler) SetPages(fn func() []service.AssistantPage) { h.svc.SetPages(fn) }

// asker: the signed-in user (optional login), else the visitor's IP.
func assistantAsker(c *gin.Context) service.AssistantAsker {
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		return service.AssistantAsker{UserID: subject.UserID}
	}
	return service.AssistantAsker{IP: middleware2.SecurityClientIP(c)}
}

// Config GET /assistant/config
func (h *AssistantHandler) Config(c *gin.Context) {
	response.Success(c, h.svc.Config(c.Request.Context(), assistantAsker(c)))
}

// Chat POST /assistant/chat {messages} — streams {"delta"} and {"tool": label} (an account lookup
// starting) events, then {"done":true,"sources","left"} or {"error"}.
func (h *AssistantHandler) Chat(c *gin.Context) {
	var in service.AssistantChatInput
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
	out, err := h.svc.Chat(c.Request.Context(), assistantAsker(c), in, func(delta string) error {
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
		response.Success(c, gin.H{"done": true, "sources": out.Sources, "left": out.Left})
		return
	}
	if err != nil {
		_ = send(gin.H{"error": infraerrors.Message(err)})
		return
	}
	_ = send(gin.H{"done": true, "sources": out.Sources, "left": out.Left})
}

// AdminSettings GET /admin/assistant/settings
func (h *AssistantHandler) AdminSettings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// AdminSaveSettings PUT /admin/assistant/settings — the key must be the signed-in admin's own.
func (h *AssistantHandler) AdminSaveSettings(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.AssistantSettings
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), subject.UserID, in)
	learnReply(c, out, err)
}

// AdminKeys GET /admin/assistant/keys — the signed-in admin's own GPT keys.
func (h *AssistantHandler) AdminKeys(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keys, err := h.svc.AdminKeys(c.Request.Context(), subject.UserID)
	learnReply(c, keys, err)
}

// AdminRuns GET /admin/assistant/runs?page= — the latest questions with the lookups they made.
func (h *AssistantHandler) AdminRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	runs, total, err := h.svc.Runs(c.Request.Context(), page)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": runs, "total": total})
}
