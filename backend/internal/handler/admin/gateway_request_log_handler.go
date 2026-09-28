package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GatewayRequestLogHandler exposes the full gateway request log to admins.
type GatewayRequestLogHandler struct {
	service *service.GatewayRequestLogService
}

// NewGatewayRequestLogHandler creates the handler.
func NewGatewayRequestLogHandler(svc *service.GatewayRequestLogService) *GatewayRequestLogHandler {
	return &GatewayRequestLogHandler{service: svc}
}

func parseOptionalPositiveInt64(c *gin.Context, name string) (*int64, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return nil, false
	}
	return &id, true
}

func parseOptionalRFC3339(c *gin.Context, name string) (*time.Time, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		response.BadRequest(c, "Invalid "+name+", expect RFC3339")
		return nil, false
	}
	return &t, true
}

// List returns a page of gateway request logs.
// GET /api/v1/admin/request-logs
func (h *GatewayRequestLogHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}
	filter := &service.GatewayRequestLogFilter{
		Page:     page,
		PageSize: pageSize,
		Method:   strings.TrimSpace(c.Query("method")),
		Model:    strings.TrimSpace(c.Query("model")),
		Query:    strings.TrimSpace(c.Query("q")),
	}

	var ok bool
	if filter.UserID, ok = parseOptionalPositiveInt64(c, "user_id"); !ok {
		return
	}
	if filter.APIKeyID, ok = parseOptionalPositiveInt64(c, "api_key_id"); !ok {
		return
	}
	if filter.StartTime, ok = parseOptionalRFC3339(c, "start_time"); !ok {
		return
	}
	if filter.EndTime, ok = parseOptionalRFC3339(c, "end_time"); !ok {
		return
	}
	if v := strings.TrimSpace(c.Query("success")); v != "" {
		success := v == "true"
		filter.Success = &success
	}
	if v := strings.TrimSpace(c.Query("status_code")); v != "" {
		code, err := strconv.Atoi(v)
		if err != nil || code < 100 || code > 599 {
			response.BadRequest(c, "Invalid status_code")
			return
		}
		filter.StatusCode = &code
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, result.Logs, int64(result.Total), result.Page, result.PageSize)
}
