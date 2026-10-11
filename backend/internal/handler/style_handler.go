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

// StyleHandler serves 测测你的写作风格 (/api/v1/style). Fetching an article needs no login; the
// analysis runs on the signed-in reader's own key.
type StyleHandler struct {
	svc *service.WritingStyleService
}

func NewStyleHandler(svc *service.WritingStyleService) *StyleHandler {
	return &StyleHandler{svc: svc}
}

// Fetch POST /style/fetch {"url":"https://mp.weixin.qq.com/s/..."}
func (h *StyleHandler) Fetch(c *gin.Context) {
	var in struct {
		URL string `json:"url"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Fetch(c.Request.Context(), in.URL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Quote GET /style/quote?key_id=7&chars=12000 → {"price": 0.05} (no price when it can't be worked out)
func (h *StyleHandler) Quote(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keyID, _ := strconv.ParseInt(c.Query("key_id"), 10, 64)
	chars, _ := strconv.Atoi(c.Query("chars"))
	response.Success(c, gin.H{"price": h.svc.Quote(c.Request.Context(), subject.UserID, keyID, chars)})
}

// Analyze POST /style/analyze (SSE): {"progress":n} while the AI writes, then {"done":true,"analysis":{...}}
// or {"error":"..."}. Errors before the AI starts come back as a normal JSON error.
func (h *StyleHandler) Analyze(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.StyleAnalyzeInput
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
		if started {
			return
		}
		started = true
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
	}
	last := 0
	out, err := h.svc.Analyze(c.Request.Context(), subject.UserID, in, func(n int) error {
		start()
		if n-last < 40 && last > 0 {
			return nil
		}
		last = n
		return send(gin.H{"progress": n})
	})
	if !started {
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		start()
	}
	if err != nil {
		_ = send(gin.H{"error": infraerrors.Message(err)})
		return
	}
	_ = send(gin.H{"done": true, "analysis": out})
}
