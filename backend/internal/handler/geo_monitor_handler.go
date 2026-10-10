package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const geoMaxBody = 256 << 10

// GeoMonitorHandler serves GEO 监测 under /api/v1/admin/geo.
type GeoMonitorHandler struct {
	svc *service.GeoMonitorService
}

func NewGeoMonitorHandler(svc *service.GeoMonitorService) *GeoMonitorHandler {
	return &GeoMonitorHandler{svc: svc}
}

func bindGeo(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, geoMaxBody)
	if err := c.ShouldBindJSON(v); err != nil {
		response.ErrorFrom(c, service.ErrGeoBadRequest)
		return false
	}
	return true
}

func geoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrGeoBadRequest)
		return 0, false
	}
	return id, true
}

func geoReply(c *gin.Context, out any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// ListQuestions GET /admin/geo/questions
func (h *GeoMonitorHandler) ListQuestions(c *gin.Context) {
	out, err := h.svc.ListQuestions(c.Request.Context())
	geoReply(c, out, err)
}

// CreateQuestion POST /admin/geo/questions
func (h *GeoMonitorHandler) CreateQuestion(c *gin.Context) {
	var in service.GeoQuestionInput
	if !bindGeo(c, &in) {
		return
	}
	out, err := h.svc.CreateQuestion(c.Request.Context(), in)
	geoReply(c, out, err)
}

// UpdateQuestion PUT /admin/geo/questions/:id
func (h *GeoMonitorHandler) UpdateQuestion(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	var in service.GeoQuestionInput
	if !bindGeo(c, &in) {
		return
	}
	out, err := h.svc.UpdateQuestion(c.Request.Context(), id, in)
	geoReply(c, out, err)
}

// DeleteQuestion DELETE /admin/geo/questions/:id (its answers stay, with the question text)
func (h *GeoMonitorHandler) DeleteQuestion(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	geoReply(c, gin.H{"deleted": true}, h.svc.DeleteQuestion(c.Request.Context(), id))
}

// ListEngines GET /admin/geo/engines — keys are never returned, only has_key and the last 4.
func (h *GeoMonitorHandler) ListEngines(c *gin.Context) {
	out, err := h.svc.ListEngines(c.Request.Context())
	geoReply(c, out, err)
}

// CreateEngine POST /admin/geo/engines
func (h *GeoMonitorHandler) CreateEngine(c *gin.Context) {
	var in service.GeoEngineInput
	if !bindGeo(c, &in) {
		return
	}
	out, err := h.svc.CreateEngine(c.Request.Context(), in)
	geoReply(c, out, err)
}

// UpdateEngine PUT /admin/geo/engines/:id — an empty api_key keeps the stored one.
func (h *GeoMonitorHandler) UpdateEngine(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	var in service.GeoEngineInput
	if !bindGeo(c, &in) {
		return
	}
	out, err := h.svc.UpdateEngine(c.Request.Context(), id, in)
	geoReply(c, out, err)
}

// DeleteEngine DELETE /admin/geo/engines/:id
func (h *GeoMonitorHandler) DeleteEngine(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	geoReply(c, gin.H{"deleted": true}, h.svc.DeleteEngine(c.Request.Context(), id))
}

// TestEngine POST /admin/geo/engines/:id/test — one short question, nothing stored.
func (h *GeoMonitorHandler) TestEngine(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	out, err := h.svc.TestEngine(c.Request.Context(), id)
	geoReply(c, out, err)
}

// ListChecks GET /admin/geo/checks?question_id=&engine=&source=&mentioned=&limit=&offset=
func (h *GeoMonitorHandler) ListChecks(c *gin.Context) {
	out, err := h.svc.ListChecks(c.Request.Context(), service.ParseGeoCheckFilter(c.Query))
	geoReply(c, out, err)
}

// geoManualBody: cited_urls is an array of links or one text with a link per line.
type geoManualBody struct {
	QuestionID int64           `json:"question_id"`
	EngineName string          `json:"engine_name"`
	Answer     string          `json:"answer"`
	CitedURLs  json.RawMessage `json:"cited_urls"`
}

func parseGeoCitedURLs(raw json.RawMessage) ([]string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, true
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list, true
	}
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return nil, false
	}
	return strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' || r == ' ' || r == '\t' }), true
}

// AddManual POST /admin/geo/checks — an answer pasted from an app without an API.
func (h *GeoMonitorHandler) AddManual(c *gin.Context) {
	var b geoManualBody
	if !bindGeo(c, &b) {
		return
	}
	urls, ok := parseGeoCitedURLs(b.CitedURLs)
	if !ok {
		response.ErrorFrom(c, service.ErrGeoBadRequest)
		return
	}
	out, err := h.svc.AddManual(c.Request.Context(), service.GeoManualInput{QuestionID: b.QuestionID, EngineName: b.EngineName, Answer: b.Answer, CitedURLs: urls})
	geoReply(c, out, err)
}

// DeleteCheck DELETE /admin/geo/checks/:id
func (h *GeoMonitorHandler) DeleteCheck(c *gin.Context) {
	id, ok := geoID(c)
	if !ok {
		return
	}
	geoReply(c, gin.H{"deleted": true}, h.svc.DeleteCheck(c.Request.Context(), id))
}

// Run POST /admin/geo/run → 202 {run_id}; 409 when a run is already going.
func (h *GeoMonitorHandler) Run(c *gin.Context) {
	id, err := h.svc.StartRun(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Accepted(c, gin.H{"run_id": id})
}

// Status GET /admin/geo/status
func (h *GeoMonitorHandler) Status(c *gin.Context) {
	out, err := h.svc.Status(c.Request.Context())
	geoReply(c, out, err)
}

// Summary GET /admin/geo/summary
func (h *GeoMonitorHandler) Summary(c *gin.Context) {
	out, err := h.svc.Summary(c.Request.Context())
	geoReply(c, out, err)
}

// Settings GET /admin/geo/settings
func (h *GeoMonitorHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// SaveSettings PUT /admin/geo/settings {schedule, brand_keywords, competitor_keywords}
func (h *GeoMonitorHandler) SaveSettings(c *gin.Context) {
	var in service.GeoSettings
	if !bindGeo(c, &in) {
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), in)
	geoReply(c, out, err)
}
