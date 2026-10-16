package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ConversationArchiveHandler serves 对话归档 (admin only):
// /api/v1/admin/conversation-archive/{config,config/test,config/lifecycle,sessions,sessions/turns,
// turns/by-request,images/:id,stats}. Every browse call except images and stats is audited by the
// service before any data is read.
type ConversationArchiveHandler struct {
	svc conversationArchiveAPI
}

// conversationArchiveAPI is what the handler needs from the service (an interface for tests).
type conversationArchiveAPI interface {
	GetConfig(ctx context.Context) (*service.ArchiveConfigView, error)
	UpdateConfig(ctx context.Context, adminID int64, in service.ArchiveConfigInput, ip, ua string) (*service.ArchiveConfigView, error)
	TestConnection(ctx context.Context, in *service.ArchiveConfigInput) error
	ApplyLifecycle(ctx context.Context, adminID int64, ip, ua string) error
	ListSessions(ctx context.Context, adminID int64, f service.ArchiveSessionFilter, ip, ua string) (*service.ArchiveSessionPage, error)
	ListTurns(ctx context.Context, adminID, userID int64, sessionKey string, beforeID int64, limit int, ip, ua string) (*service.ArchiveTurnPage, error)
	GetTurnByRequest(ctx context.Context, adminID int64, requestID, ip, ua string) (*service.ArchiveTurnView, error)
	OpenImage(ctx context.Context, id int64) (io.ReadCloser, *service.ArchiveImage, error)
	Stats(ctx context.Context) ([]service.ArchiveDayStats, error)
}

func NewConversationArchiveHandler(svc *service.ConversationArchiveService) *ConversationArchiveHandler {
	if svc == nil {
		return &ConversationArchiveHandler{}
	}
	return &ConversationArchiveHandler{svc: svc}
}

const conversationArchiveMaxBody = 64 << 10

func (h *ConversationArchiveHandler) ready(c *gin.Context) bool {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, service.ErrArchiveNotConfigured)
		return false
	}
	return true
}

// GetConfig GET /config
func (h *ConversationArchiveHandler) GetConfig(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	view, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, view)
}

func decodeArchiveConfigInput(c *gin.Context, allowEmpty bool) (*service.ArchiveConfigInput, bool) {
	if c.Request.Body == nil {
		if allowEmpty {
			return nil, true
		}
		response.BadRequest(c, "请求格式错误（Invalid request）")
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, conversationArchiveMaxBody))
	if err != nil {
		response.BadRequest(c, "请求格式错误（Invalid request）")
		return nil, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		if allowEmpty {
			return nil, true
		}
		response.BadRequest(c, "请求格式错误（Invalid request）")
		return nil, false
	}
	var in service.ArchiveConfigInput
	if err := json.Unmarshal(raw, &in); err != nil {
		response.BadRequest(c, "请求格式错误（Invalid request）")
		return nil, false
	}
	return &in, true
}

// UpdateConfig PUT /config (step-up auth). A missing or empty secret keeps the stored one.
func (h *ConversationArchiveHandler) UpdateConfig(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	in, ok := decodeArchiveConfigInput(c, false)
	if !ok {
		return
	}
	view, err := h.svc.UpdateConfig(c.Request.Context(), adminID, *in, ip.GetClientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, view)
}

// TestConfig POST /config/test with the (possibly unsaved) form; an empty body tests the saved
// configuration and a missing secret uses the saved one.
func (h *ConversationArchiveHandler) TestConfig(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	in, ok := decodeArchiveConfigInput(c, true)
	if !ok {
		return
	}
	if err := h.svc.TestConnection(c.Request.Context(), in); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ApplyLifecycle POST /config/lifecycle (step-up auth)
func (h *ConversationArchiveHandler) ApplyLifecycle(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	if err := h.svc.ApplyLifecycle(c.Request.Context(), adminID, ip.GetClientIP(c), c.GetHeader("User-Agent")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Sessions GET /sessions?user_id=&q=&from=&to=&before=&limit= (audited)
func (h *ConversationArchiveHandler) Sessions(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	f := service.ArchiveSessionFilter{Q: c.Query("q")}
	f.UserID, _ = strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)
	f.Limit, _ = strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	var bad bool
	if f.From, bad = parseArchiveTimeParam(c, "from"); bad {
		return
	}
	if f.To, bad = parseArchiveTimeParam(c, "to"); bad {
		return
	}
	if f.Before, bad = parseArchiveTimeParam(c, "before"); bad {
		return
	}
	page, err := h.svc.ListSessions(c.Request.Context(), adminID, f, ip.GetClientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

// Turns GET /sessions/turns?user_id=&session_key=&before_id=&limit= (audited)
func (h *ConversationArchiveHandler) Turns(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)
	beforeID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("before_id")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	page, err := h.svc.ListTurns(c.Request.Context(), adminID, userID, c.Query("session_key"), beforeID, limit, ip.GetClientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

// TurnByRequest GET /turns/by-request?request_id= (audited)
func (h *ConversationArchiveHandler) TurnByRequest(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	item, err := h.svc.GetTurnByRequest(c.Request.Context(), adminID, c.Query("request_id"), ip.GetClientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// Image GET /images/:id streams the image bytes from the bucket (not the JSON wrapper).
func (h *ConversationArchiveHandler) Image(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的图片 ID（Invalid image id）")
		return
	}
	rc, img, err := h.svc.OpenImage(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = rc.Close() }()
	mime := strings.TrimSpace(img.Mime)
	if !strings.HasPrefix(mime, "image/") {
		mime = "application/octet-stream"
	}
	header := c.Writer.Header()
	header.Set("Content-Type", mime)
	header.Set("Cache-Control", "private, max-age=300")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Disposition", "inline")
	if img.Bytes > 0 {
		header.Set("Content-Length", strconv.Itoa(img.Bytes))
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

// Stats GET /stats (not audited)
func (h *ConversationArchiveHandler) Stats(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	days, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"days": days})
}

var errArchiveBadTime = infraerrors.BadRequest("CONVERSATION_ARCHIVE_BAD_TIME", "时间格式错误，请使用 RFC3339（Invalid time, use RFC3339）")

// parseArchiveTimeParam returns (value, aborted).
func parseArchiveTimeParam(c *gin.Context, name string) (*time.Time, bool) {
	t, ok := service.ParseArchiveTime(c.Query(name))
	if !ok {
		response.ErrorFrom(c, errArchiveBadTime)
		return nil, true
	}
	return t, false
}
