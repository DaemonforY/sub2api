package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ShihuaHandler serves the 课本诗画 admin pages (/api/v1/admin/shihua/*) and the signed picture links.
type ShihuaHandler struct {
	svc *service.ShihuaService
}

func NewShihuaHandler(svc *service.ShihuaService) *ShihuaHandler {
	return &ShihuaHandler{svc: svc}
}

func shihuaAdminID(c *gin.Context) int64 {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	return 0
}

func shihuaPathID(c *gin.Context, notFound error) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, notFound)
		return 0, false
	}
	return id, true
}

// shihuaIDs reads "1,2,3"; nil when the parameter is absent.
func shihuaIDs(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ids := []int64{}
	for _, part := range strings.Split(raw, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *ShihuaHandler) respond(c *gin.Context, data any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// GetSettings GET /admin/shihua/settings
func (h *ShihuaHandler) GetSettings(c *gin.Context) {
	v, err := h.svc.SettingsView(c.Request.Context(), shihuaAdminID(c))
	h.respond(c, v, err)
}

// SaveSettings PUT /admin/shihua/settings
func (h *ShihuaHandler) SaveSettings(c *gin.Context) {
	var in service.ShihuaSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	v, err := h.svc.SaveSettings(c.Request.Context(), shihuaAdminID(c), in)
	h.respond(c, v, err)
}

// ListWorks GET /admin/shihua/works?stage=&grade=&status=&q=&ids=
func (h *ShihuaHandler) ListWorks(c *gin.Context) {
	works, err := h.svc.ListWorks(c.Request.Context(), service.ShihuaWorkQuery{
		Stage: c.Query("stage"), Grade: c.Query("grade"), Status: c.Query("status"), Q: c.Query("q"), IDs: shihuaIDs(c.Query("ids")),
	})
	h.respond(c, gin.H{"items": works}, err)
}

// GetWork GET /admin/shihua/works/:id
func (h *ShihuaHandler) GetWork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	w, err := h.svc.GetWork(c.Request.Context(), id)
	h.respond(c, w, err)
}

// CreateWork POST /admin/shihua/works
func (h *ShihuaHandler) CreateWork(c *gin.Context) {
	var in service.ShihuaWorkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	w, err := h.svc.CreateWork(c.Request.Context(), in)
	h.respond(c, w, err)
}

// UpdateWork PUT /admin/shihua/works/:id
func (h *ShihuaHandler) UpdateWork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	var in service.ShihuaWorkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	w, err := h.svc.UpdateWork(c.Request.Context(), id, in)
	h.respond(c, w, err)
}

// DeleteWork DELETE /admin/shihua/works/:id
func (h *ShihuaHandler) DeleteWork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	h.respond(c, gin.H{"deleted": true}, h.svc.DeleteWork(c.Request.Context(), id))
}

type shihuaImportRequest struct {
	Works  []service.ShihuaWorkInput `json:"works"`
	DryRun bool                      `json:"dry_run"`
}

// ImportWorks POST /admin/shihua/works/import {works, dry_run}
func (h *ShihuaHandler) ImportWorks(c *gin.Context) {
	var in shihuaImportRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "导入文件格式不对，应为 {\"works\": [...]}（Invalid import file）: "+err.Error())
		return
	}
	res, err := h.svc.Import(c.Request.Context(), in.Works, in.DryRun)
	h.respond(c, res, err)
}

// VerifyWork POST /admin/shihua/works/:id/verify {level: 0|1|2}
func (h *ShihuaHandler) VerifyWork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	var in struct {
		Level int `json:"level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	w, err := h.svc.Verify(c.Request.Context(), shihuaAdminID(c), id, in.Level)
	h.respond(c, w, err)
}

// GenerateTheme POST /admin/shihua/works/:id/theme
func (h *ShihuaHandler) GenerateTheme(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	w, err := h.svc.GenerateTheme(c.Request.Context(), id)
	h.respond(c, w, err)
}

// Redraw POST /admin/shihua/works/:id/redraw {count, prompt_theme}
func (h *ShihuaHandler) Redraw(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaWorkNotFound)
	if !ok {
		return
	}
	var in service.ShihuaRedrawInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	images, err := h.svc.Redraw(c.Request.Context(), id, in)
	h.respond(c, gin.H{"items": images}, err)
}

// ListJobs GET /admin/shihua/jobs
func (h *ShihuaHandler) ListJobs(c *gin.Context) {
	jobs, err := h.svc.ListJobs(c.Request.Context())
	h.respond(c, gin.H{"items": jobs}, err)
}

// CreateJob POST /admin/shihua/jobs {name, work_ids, candidates}
func (h *ShihuaHandler) CreateJob(c *gin.Context) {
	var in service.ShihuaJobInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	job, err := h.svc.CreateJob(c.Request.Context(), shihuaAdminID(c), in)
	h.respond(c, job, err)
}

// CancelJob POST /admin/shihua/jobs/:id/cancel (id 0 = every queued drawing)
func (h *ShihuaHandler) CancelJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 0 {
		response.ErrorFrom(c, service.ErrShihuaJobNotFound)
		return
	}
	n, err := h.svc.CancelJob(c.Request.Context(), id)
	h.respond(c, gin.H{"canceled": n}, err)
}

// RetryJob POST /admin/shihua/jobs/:id/retry (failed and canceled drawings go back in the queue)
func (h *ShihuaHandler) RetryJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 0 {
		response.ErrorFrom(c, service.ErrShihuaJobNotFound)
		return
	}
	n, err := h.svc.RetryFailed(c.Request.Context(), id)
	h.respond(c, gin.H{"queued": n}, err)
}

// ListImages GET /admin/shihua/images?work_ids=&job_id=&status=
func (h *ShihuaHandler) ListImages(c *gin.Context) {
	jobID, _ := strconv.ParseInt(c.Query("job_id"), 10, 64)
	images, err := h.svc.ListImages(c.Request.Context(), service.ShihuaImageQuery{
		WorkIDs: shihuaIDs(c.Query("work_ids")), JobID: jobID, Status: c.Query("status"),
	})
	h.respond(c, gin.H{"items": images}, err)
}

// ReviewImage POST /admin/shihua/images/:id/review {action, title_check}
func (h *ShihuaHandler) ReviewImage(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaImageNotFound)
	if !ok {
		return
	}
	var in service.ShihuaReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	img, err := h.svc.Review(c.Request.Context(), id, in)
	h.respond(c, img, err)
}

// SaveRender PUT /admin/shihua/images/:id/renders/:aspect  multipart: file (JPEG / PNG), layout (JSON)
func (h *ShihuaHandler) SaveRender(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrShihuaImageNotFound)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 31<<20))
	if err != nil {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	r, err := h.svc.SaveRender(c.Request.Context(), id, c.Param("aspect"), json.RawMessage(c.PostForm("layout")), data)
	h.respond(c, r, err)
}

// ListRenders GET /admin/shihua/renders?work_ids=
func (h *ShihuaHandler) ListRenders(c *gin.Context) {
	renders, err := h.svc.ListRenders(c.Request.Context(), shihuaIDs(c.Query("work_ids")))
	h.respond(c, gin.H{"items": renders}, err)
}

// Export GET /admin/shihua/export?work_ids=&aspects=2x3,3x4 → zip
func (h *ShihuaHandler) Export(c *gin.Context) {
	ids := shihuaIDs(c.Query("work_ids"))
	if len(ids) == 0 {
		response.ErrorFrom(c, service.ErrShihuaInvalid)
		return
	}
	var aspects []string
	for _, a := range strings.Split(c.Query("aspects"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			aspects = append(aspects, a)
		}
	}
	name := "shihua-" + time.Now().Format("20060102-1504") + ".zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	if err := h.svc.ExportZip(c.Request.Context(), c.Writer, ids, aspects); err != nil {
		_ = c.Error(err)
	}
}

// File GET /api/v1/shihua/files/*path?e=&s= — a signed link to a picture (no login, so <img> works).
func (h *ShihuaHandler) File(c *gin.Context) {
	path, ok := h.svc.OpenFile(c.Param("path"), c.Query("e"), c.Query("s"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=43200")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
