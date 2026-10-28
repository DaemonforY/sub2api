package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// MaterialHandler serves the 诗画素材库 admin pages (/api/v1/admin/materials/*).
type MaterialHandler struct {
	svc *service.MaterialService
}

func NewMaterialHandler(svc *service.MaterialService) *MaterialHandler {
	return &MaterialHandler{svc: svc}
}

func (h *MaterialHandler) respond(c *gin.Context, data any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// ListTexts GET /admin/materials/texts?kind=&status=&license=&q=&limit=&offset=
func (h *MaterialHandler) ListTexts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	texts, total, err := h.svc.ListTexts(c.Request.Context(), service.MaterialTextQuery{
		Kind: c.Query("kind"), Status: c.Query("status"), License: c.Query("license"), Q: c.Query("q"),
		Limit: limit, Offset: offset,
	})
	h.respond(c, gin.H{"items": texts, "total": total}, err)
}

// GetText GET /admin/materials/texts/:id
func (h *MaterialHandler) GetText(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialTextNotFound)
	if !ok {
		return
	}
	t, err := h.svc.GetText(c.Request.Context(), id)
	h.respond(c, t, err)
}

// ImportShihua POST /admin/materials/import/shihua copies twice-checked 课本诗画 works in.
func (h *MaterialHandler) ImportShihua(c *gin.Context) {
	res, err := h.svc.ImportShihua(c.Request.Context(), shihuaAdminID(c))
	h.respond(c, res, err)
}

// ListStyles GET /admin/materials/styles
func (h *MaterialHandler) ListStyles(c *gin.Context) {
	styles, err := h.svc.ListStyles(c.Request.Context())
	h.respond(c, gin.H{"items": styles}, err)
}

// ListBlocklist GET /admin/materials/blocklist
func (h *MaterialHandler) ListBlocklist(c *gin.Context) {
	terms, err := h.svc.ListBlocklist(c.Request.Context())
	h.respond(c, gin.H{"items": terms}, err)
}

type materialCheckRequest struct {
	Input  string `json:"input"`
	Target string `json:"target"` // prompt | text
}

// CheckPrompt POST /admin/materials/check {input, target}
func (h *MaterialHandler) CheckPrompt(c *gin.Context) {
	var in materialCheckRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	if in.Target == "" {
		in.Target = "prompt"
	}
	res, err := h.svc.CheckPrompt(c.Request.Context(), in.Input, in.Target)
	h.respond(c, res, err)
}

// ListImagery GET /admin/materials/imagery?status=&category=&q=&limit=&offset=
func (h *MaterialHandler) ListImagery(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, total, err := h.svc.ListImagery(c.Request.Context(), service.MaterialImageryQuery{
		Status: c.Query("status"), Category: c.Query("category"), Q: c.Query("q"), Limit: limit, Offset: offset,
	})
	h.respond(c, gin.H{"items": items, "total": total, "categories": service.MaterialImageryCategories}, err)
}

// UpdateImagery PUT /admin/materials/imagery/:id
func (h *MaterialHandler) UpdateImagery(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialImageryNotFound)
	if !ok {
		return
	}
	var in service.MaterialImagery
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	in.ID = id
	im, err := h.svc.UpdateImagery(c.Request.Context(), &in)
	h.respond(c, im, err)
}

type materialStatusRequest struct {
	IDs    []int64 `json:"ids"`
	Status string  `json:"status"`
}

// SetImageryStatus POST /admin/materials/imagery/status {ids, status}
func (h *MaterialHandler) SetImageryStatus(c *gin.Context) {
	var in materialStatusRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	n, err := h.svc.SetImageryStatus(c.Request.Context(), in.IDs, in.Status)
	h.respond(c, gin.H{"updated": n}, err)
}

// ImageryCandidates GET /admin/materials/imagery/candidates
func (h *MaterialHandler) ImageryCandidates(c *gin.Context) {
	items, err := h.svc.ImageryCandidates(c.Request.Context())
	h.respond(c, gin.H{"items": items}, err)
}

type materialSeedRequest struct {
	Limit int `json:"limit"`
}

// DraftImagerySeeds POST /admin/materials/imagery/seed {limit ≤ 30}
func (h *MaterialHandler) DraftImagerySeeds(c *gin.Context) {
	var in materialSeedRequest
	_ = c.ShouldBindJSON(&in)
	res, err := h.svc.DraftImagerySeeds(c.Request.Context(), shihuaAdminID(c), in.Limit)
	h.respond(c, res, err)
}

// GetSegmentImagery GET /admin/materials/segments/:id/imagery
func (h *MaterialHandler) GetSegmentImagery(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	seg, links, err := h.svc.SegmentImagery(c.Request.Context(), id)
	h.respond(c, gin.H{"segment": seg, "links": links}, err)
}

// ExtractSegmentImagery POST /admin/materials/segments/:id/imagery/extract
func (h *MaterialHandler) ExtractSegmentImagery(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	res, err := h.svc.ExtractSegmentImagery(c.Request.Context(), shihuaAdminID(c), id)
	h.respond(c, res, err)
}

type materialSegmentImageryRequest struct {
	Links     []service.MaterialSegmentImageryInput `json:"links"`
	SceneDesc *string                               `json:"scene_desc"`
}

// SaveSegmentImagery PUT /admin/materials/segments/:id/imagery {links, scene_desc?}
func (h *MaterialHandler) SaveSegmentImagery(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	var in materialSegmentImageryRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	res, err := h.svc.SaveSegmentImagery(c.Request.Context(), id, in.Links, in.SceneDesc)
	h.respond(c, res, err)
}

// RecommendStyles GET /admin/materials/segments/:id/styles
func (h *MaterialHandler) RecommendStyles(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	items, err := h.svc.RecommendStyles(c.Request.Context(), id)
	h.respond(c, gin.H{"items": items}, err)
}

// BuildPrompt POST /admin/materials/segments/:id/prompt {style_id, size, composition, palette, mood, extra, imagery_ids?}
// previews the prompt a drawing would use; nothing is saved.
func (h *MaterialHandler) BuildPrompt(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	var in service.MaterialPromptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	p, err := h.svc.BuildPrompt(c.Request.Context(), id, in)
	h.respond(c, p, err)
}

type materialDrawRequest struct {
	service.MaterialPromptInput
	Count int `json:"count"`
}

// DrawSegment POST /admin/materials/segments/:id/draw {style_id, size, composition, palette, mood, extra, imagery_ids?, count ≤ 4}
func (h *MaterialHandler) DrawSegment(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialSegmentNotFound)
	if !ok {
		return
	}
	var in materialDrawRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	arts, err := h.svc.DrawSegment(c.Request.Context(), shihuaAdminID(c), id, in.MaterialPromptInput, in.Count)
	h.respond(c, gin.H{"items": arts}, err)
}

// ListArtworks GET /admin/materials/artworks?segment_id=&text_id=&status=&review_status=&limit=&offset=
func (h *MaterialHandler) ListArtworks(c *gin.Context) {
	segmentID, _ := strconv.ParseInt(c.Query("segment_id"), 10, 64)
	textID, _ := strconv.ParseInt(c.Query("text_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	items, total, err := h.svc.ListArtworks(c.Request.Context(), service.MaterialArtworkQuery{
		SegmentID: segmentID, TextID: textID, Status: c.Query("status"), ReviewStatus: c.Query("review_status"),
		Limit: limit, Offset: offset,
	})
	h.respond(c, gin.H{"items": items, "total": total}, err)
}

// GetArtwork GET /admin/materials/artworks/:id
func (h *MaterialHandler) GetArtwork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialArtworkNotFound)
	if !ok {
		return
	}
	a, err := h.svc.GetArtwork(c.Request.Context(), id)
	h.respond(c, a, err)
}

// Redraw POST /admin/materials/artworks/:id/redraw
func (h *MaterialHandler) Redraw(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialArtworkNotFound)
	if !ok {
		return
	}
	a, err := h.svc.Redraw(c.Request.Context(), shihuaAdminID(c), id)
	h.respond(c, a, err)
}

// ReviewArtwork POST /admin/materials/artworks/:id/review {review_status, rating, reject_reason}
func (h *MaterialHandler) ReviewArtwork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialArtworkNotFound)
	if !ok {
		return
	}
	var in service.MaterialReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	a, err := h.svc.ReviewArtwork(c.Request.Context(), id, in)
	h.respond(c, a, err)
}

// RetryArtwork POST /admin/materials/artworks/:id/retry
func (h *MaterialHandler) RetryArtwork(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialArtworkNotFound)
	if !ok {
		return
	}
	a, err := h.svc.RetryArtwork(c.Request.Context(), id)
	h.respond(c, a, err)
}

type materialIDsRequest struct {
	IDs []int64 `json:"ids"`
}

// CancelArtworks POST /admin/materials/artworks/cancel {ids}
func (h *MaterialHandler) CancelArtworks(c *gin.Context) {
	var in materialIDsRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	n, err := h.svc.CancelArtworks(c.Request.Context(), in.IDs)
	h.respond(c, gin.H{"canceled": n}, err)
}

// Queue GET /admin/materials/queue
func (h *MaterialHandler) Queue(c *gin.Context) {
	v, err := h.svc.QueueView(c.Request.Context())
	h.respond(c, v, err)
}

// File GET /api/v1/materials/files/*path?e=&s= serves a picture by signed link (no login: the
// admin page puts these in <img>).
func (h *MaterialHandler) File(c *gin.Context) {
	path, ok := h.svc.OpenFile(c.Param("path"), c.Query("e"), c.Query("s"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=43200")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}

// SaveRender PUT /admin/materials/artworks/:id/renders/:aspect (multipart: file, layout)
func (h *MaterialHandler) SaveRender(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrMaterialArtworkNotFound)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 31<<20))
	if err != nil {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	r, err := h.svc.SaveRender(c.Request.Context(), id, c.Param("aspect"), json.RawMessage(c.PostForm("layout")), data)
	h.respond(c, r, err)
}

// ListRenders GET /admin/materials/renders?artwork_ids=1,2
func (h *MaterialHandler) ListRenders(c *gin.Context) {
	items, err := h.svc.ListRenders(c.Request.Context(), shihuaIDs(c.Query("artwork_ids")))
	h.respond(c, gin.H{"items": items}, err)
}

// Export GET /admin/materials/export?artwork_ids=1,2&aspects=2x3,3x4 streams a zip of the posters.
func (h *MaterialHandler) Export(c *gin.Context) {
	ids := shihuaIDs(c.Query("artwork_ids"))
	if len(ids) == 0 {
		response.ErrorFrom(c, service.ErrMaterialInvalid)
		return
	}
	var aspects []string
	for _, a := range strings.Split(c.Query("aspects"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			aspects = append(aspects, a)
		}
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="materials-`+time.Now().Format("20060102-1504")+`.zip"`)
	if err := h.svc.ExportZip(c.Request.Context(), c.Writer, ids, aspects); err != nil {
		_ = c.Error(err)
	}
}
