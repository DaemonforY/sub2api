package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContentLogHandler serves 请求内容日志 (admin only) and provides the gateway capture middleware.
type ContentLogHandler struct {
	svc *service.ContentLogService
}

func NewContentLogHandler(svc *service.ContentLogService) *ContentLogHandler {
	return &ContentLogHandler{svc: svc}
}

var errContentLogNoAdmin = infraerrors.Unauthorized("CONTENT_LOG_NO_ADMIN", "未登录或登录已过期（Not authenticated）")

func contentLogAdminID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return 0, false
	}
	return subject.UserID, true
}

func contentLogUserIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的用户 ID（Invalid user id）")
		return 0, false
	}
	return id, true
}

// Targets GET /api/v1/admin/user-insights/content-log/targets
func (h *ContentLogHandler) Targets(c *gin.Context) {
	items, err := h.svc.ListTargets(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

type contentLogEnableRequest struct {
	Note string `json:"note"`
}

// EnableTarget PUT /api/v1/admin/user-insights/content-log/targets/:id {note}
func (h *ContentLogHandler) EnableTarget(c *gin.Context) {
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	userID, ok := contentLogUserIDParam(c)
	if !ok {
		return
	}
	var req contentLogEnableRequest
	if c.Request.Body != nil {
		if err := json.NewDecoder(io.LimitReader(c.Request.Body, 64<<10)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			response.BadRequest(c, "请求格式错误（Invalid request）")
			return
		}
	}
	if err := h.svc.Enable(c.Request.Context(), adminID, userID, req.Note, ip.GetClientIP(c), c.GetHeader("User-Agent")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": true})
}

// DisableTarget DELETE /api/v1/admin/user-insights/content-log/targets/:id
func (h *ContentLogHandler) DisableTarget(c *gin.Context) {
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	userID, ok := contentLogUserIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.Disable(c.Request.Context(), adminID, userID, ip.GetClientIP(c), c.GetHeader("User-Agent")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": false})
}

// Config GET /api/v1/admin/user-insights/content-log/config
func (h *ContentLogHandler) Config(c *gin.Context) {
	response.Success(c, gin.H{"redact_secrets": h.svc.RedactSecrets()})
}

type contentLogConfigRequest struct {
	RedactSecrets *bool `json:"redact_secrets"`
}

// UpdateConfig PUT /api/v1/admin/user-insights/content-log/config {redact_secrets}
func (h *ContentLogHandler) UpdateConfig(c *gin.Context) {
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	var req contentLogConfigRequest
	if err := json.NewDecoder(io.LimitReader(c.Request.Body, 64<<10)).Decode(&req); err != nil || req.RedactSecrets == nil {
		response.BadRequest(c, "请求格式错误（Invalid request）")
		return
	}
	if err := h.svc.SetRedactSecrets(c.Request.Context(), adminID, *req.RedactSecrets, ip.GetClientIP(c), c.GetHeader("User-Agent")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"redact_secrets": *req.RedactSecrets})
}

// Logs GET /api/v1/admin/user-insights/content-log?user_id=&session_id=&before_id=&limit=
// Every call is written to the access log.
func (h *ContentLogHandler) Logs(c *gin.Context) {
	adminID, ok := contentLogAdminID(c)
	if !ok {
		response.ErrorFrom(c, errContentLogNoAdmin)
		return
	}
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)
	if userID <= 0 {
		response.ErrorFrom(c, service.ErrContentLogUserIDRequired)
		return
	}
	beforeID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("before_id")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	page, err := h.svc.ListLogs(c.Request.Context(), adminID, service.ContentLogFilter{
		UserID: userID, SessionID: c.Query("session_id"), BeforeID: beforeID, Limit: limit,
	}, ip.GetClientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

// Access GET /api/v1/admin/user-insights/content-log/access?user_id=&limit= (not audited itself)
func (h *ContentLogHandler) Access(c *gin.Context) {
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.ListAccess(c.Request.Context(), userID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

// --- gateway capture ---

// contentCaptureMaxBodyBytes: larger requests are not captured at all.
const contentCaptureMaxBodyBytes = 64 << 20

// CaptureMiddleware runs on the gateway routes right after API-key auth. For accounts under content
// logging it reads the request body, puts identical bytes back for the handler, and records the
// latest user message in the gin.Context; the response side is collected by the upstream response
// observer. Every other request only pays the in-memory IsTarget lookup. It never fails a request.
func (h *ContentLogHandler) CaptureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h != nil && h.svc != nil {
			captureContentLogRequest(c, h.svc)
		}
		c.Next()
	}
}

type contentLogTargetChecker interface {
	IsTarget(userID int64) bool
}

func captureContentLogRequest(c *gin.Context, targets contentLogTargetChecker) {
	defer func() { _ = recover() }()
	// POST only. WebSocket Responses (a GET upgrade) is deliberately not captured in this version.
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || !targets.IsTarget(apiKey.UserID) {
		return
	}
	protocol := contentLogProtocolForPath(c.Request.URL.Path)
	if protocol == "" {
		return
	}
	body, ok := bufferRequestBodyForCapture(c.Request)
	if !ok {
		return
	}
	service.SetContentCaptureRequest(c, extractContentLogUserText(c.Request, protocol, body))
}

// replayBody serves the buffered bytes (and, when the body was only partly read, the rest of the
// original stream or the original read error) and closes the original body.
type replayBody struct {
	io.Reader
	orig io.ReadCloser
}

func (b *replayBody) Close() error { return b.orig.Close() }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// bufferRequestBodyForCapture reads the body (up to the capture limit) and replaces req.Body with a
// reader that yields exactly the same bytes, so the handler sees the request unchanged. ok is
// false when the body is too large or could not be read; the handler then still gets the same
// bytes followed by the rest of the stream or the same read error.
func bufferRequestBodyForCapture(req *http.Request) ([]byte, bool) {
	orig := req.Body
	buf, err := io.ReadAll(io.LimitReader(orig, contentCaptureMaxBodyBytes+1))
	switch {
	case err != nil:
		req.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(buf), errorReader{err: err}), orig: orig}
		return nil, false
	case len(buf) > contentCaptureMaxBodyBytes:
		req.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(buf), orig), orig: orig}
		return nil, false
	default:
		req.Body = &replayBody{Reader: bytes.NewReader(buf), orig: orig}
		return buf, true
	}
}

func contentLogProtocolForPath(path string) string {
	p := strings.ToLower(strings.TrimRight(path, "/"))
	switch {
	case strings.HasSuffix(p, "/count_tokens"), strings.HasSuffix(p, "/async"):
		return ""
	case strings.HasSuffix(p, "/images/generations"), strings.HasSuffix(p, "/images/edits"):
		return securityaudit.LatestUserTextImages
	case strings.HasSuffix(p, "/chat/completions"):
		return securityaudit.LatestUserTextChat
	case strings.HasSuffix(p, "/responses"), strings.HasSuffix(p, "/responses/compact"):
		// Only the Responses endpoints that record usage (not input_tokens, cancel, …).
		return securityaudit.LatestUserTextResponses
	case strings.HasSuffix(p, "/messages"):
		return securityaudit.LatestUserTextAnthropic
	case strings.HasPrefix(p, "/v1beta/") && (strings.HasSuffix(p, ":generatecontent") || strings.HasSuffix(p, ":streamgeneratecontent")):
		return securityaudit.LatestUserTextGemini
	default:
		return ""
	}
}

// extractContentLogUserText returns the latest user message (or the image prompt). It works on a
// decoded copy when the body is compressed and never touches the request itself.
func extractContentLogUserText(req *http.Request, protocol string, body []byte) (text string) {
	defer func() {
		if recover() != nil {
			text = ""
		}
	}()
	if enc := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Encoding"))); enc != "" && enc != "identity" {
		clone := &http.Request{Header: req.Header.Clone(), Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}
		decoded, err := pkghttputil.ReadRequestBodyWithPrealloc(clone)
		if err != nil {
			return ""
		}
		body = decoded
	}
	if protocol == securityaudit.LatestUserTextImages {
		if mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type")); err == nil && strings.HasPrefix(mediaType, "multipart/") {
			return multipartPromptField(body, params["boundary"])
		}
	}
	return securityaudit.ExtractLatestUserText(protocol, body)
}

// multipartPromptField reads only the `prompt` form field; image parts are skipped unread.
func multipartPromptField(body []byte, boundary string) string {
	if boundary == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			return ""
		}
		if part.FormName() == "prompt" && part.FileName() == "" {
			value, _ := io.ReadAll(io.LimitReader(part, 64<<10))
			_ = part.Close()
			return strings.TrimSpace(string(value))
		}
		_ = part.Close()
	}
}
