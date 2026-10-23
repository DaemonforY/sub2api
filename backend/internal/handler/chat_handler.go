package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ChatHandler serves AI 对话 (/api/v1/chat): conversations, pictures and streamed turns on the
// user's own keys. All routes need a signed-in user.
type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func chatConvID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "对话编号不正确（Invalid conversation）")
		return 0, false
	}
	return id, true
}

// Keys GET /chat/keys
func (h *ChatHandler) Keys(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	out, err := h.svc.Keys(c.Request.Context(), subject.UserID)
	learnReply(c, out, err)
}

// Models GET /chat/models?key_id=
func (h *ChatHandler) Models(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keyID, _ := strconv.ParseInt(c.Query("key_id"), 10, 64)
	out, err := h.svc.Models(c.Request.Context(), subject.UserID, keyID)
	learnReply(c, out, err)
}

// List GET /chat/conversations?q=
func (h *ChatHandler) List(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	out, err := h.svc.List(c.Request.Context(), subject.UserID, c.Query("q"))
	learnReply(c, out, err)
}

// Create POST /chat/conversations {title?, system_prompt?}
func (h *ChatHandler) Create(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.ChatConversationInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), subject.UserID, in)
	learnReply(c, out, err)
}

// Get GET /chat/conversations/:id
func (h *ChatHandler) Get(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := chatConvID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), subject.UserID, id)
	learnReply(c, out, err)
}

// Update PUT /chat/conversations/:id {title?, system_prompt?, pinned?}
func (h *ChatHandler) Update(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := chatConvID(c)
	if !ok {
		return
	}
	var in service.ChatConversationInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), subject.UserID, id, in)
	learnReply(c, out, err)
}

// Delete DELETE /chat/conversations/:id
func (h *ChatHandler) Delete(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := chatConvID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.Delete(c.Request.Context(), subject.UserID, id))
}

// Send POST /chat/conversations/:id/send — streams {"user","assistant","conversation"} once the turn
// is saved, then {"delta"} / {"reasoning"} pieces, then {"done": reply} (with {"error"} if it failed).
// Errors before the turn is saved are a JSON error reply.
func (h *ChatHandler) Send(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := chatConvID(c)
	if !ok {
		return
	}
	var in service.ChatSendInput
	if !bindLearn(c, &in) {
		return
	}
	var (
		mu      sync.Mutex
		started bool
		closed  bool
	)
	send := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return io.ErrClosedPipe
		}
		if !started {
			started = true
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			c.Status(http.StatusOK)
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", b); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	// A picture takes a minute or two: keep the connection alive meanwhile.
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-c.Request.Context().Done():
				return
			case <-t.C:
				mu.Lock()
				if started && !closed {
					_, _ = fmt.Fprint(c.Writer, ": ping\n\n")
					c.Writer.Flush()
				}
				mu.Unlock()
			}
		}
	}()
	err := h.svc.Send(c.Request.Context(), subject.UserID, id, in, func(ev service.ChatEvent) error { return send(ev) })
	close(stop)
	mu.Lock()
	wasStarted := started
	closed = true
	mu.Unlock()
	if err != nil && !wasStarted {
		response.ErrorFrom(c, err)
	}
}

// Upload POST /chat/files (multipart: file)
func (h *ChatHandler) Upload(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ChatFileMaxBytes+1<<20)
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择图片，最大 20MB（File required）")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, service.ChatFileMaxBytes+1))
	if err != nil {
		response.BadRequest(c, "读取图片失败（Read failed）")
		return
	}
	out, err := h.svc.Upload(c.Request.Context(), subject.UserID, data)
	learnReply(c, out, err)
}

// File GET /chat/files/:name — one of the user's pictures.
func (h *ChatHandler) File(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	data, mime, err := h.svc.File(c.Request.Context(), subject.UserID, c.Param("name"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=604800, immutable")
	c.Data(http.StatusOK, mime, data)
}
