package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PromptLibraryHandler lets admins curate the canvas prompt library: fix scenes and tags, hide or
// feature items, review prompts users share, and manage the synced sources.
type PromptLibraryHandler struct {
	service *service.PromptLibraryService
	sync    *service.PromptLibrarySyncService
}

func NewPromptLibraryHandler(svc *service.PromptLibraryService, sync *service.PromptLibrarySyncService) *PromptLibraryHandler {
	return &PromptLibraryHandler{service: svc, sync: sync}
}

func optionalBoolQuery(c *gin.Context, name string) *bool {
	switch strings.TrimSpace(c.Query(name)) {
	case "true", "1", "yes":
		v := true
		return &v
	case "false", "0", "no":
		v := false
		return &v
	}
	return nil
}

// List GET /api/v1/admin/prompt-library/items
func (h *PromptLibraryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	q := service.PromptListQuery{
		Keyword: c.Query("q"), Scene: c.Query("scene"), Model: c.Query("model"), SourceID: strings.TrimSpace(c.Query("source")),
		Kind: c.Query("kind"), Tag: c.Query("tag"), Sort: c.Query("sort"), Page: page, PageSize: pageSize,
	}
	q.Status = strings.TrimSpace(c.Query("status"))
	q.Curated = optionalBoolQuery(c, "curated")
	q.Featured = optionalBoolQuery(c, "featured")
	if q.Sort == "" {
		q.Sort = service.PromptSortLatest
	}
	res, err := h.service.ListAdmin(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

func promptID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}

// Get GET /api/v1/admin/prompt-library/items/:id
func (h *PromptLibraryHandler) Get(c *gin.Context) {
	id, ok := promptID(c)
	if !ok {
		return
	}
	item, err := h.service.AdminGet(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// Create POST /api/v1/admin/prompt-library/items
func (h *PromptLibraryHandler) Create(c *gin.Context) {
	var in service.PromptAdminInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return
	}
	item, err := h.service.AdminCreate(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// Update PUT /api/v1/admin/prompt-library/items/:id
func (h *PromptLibraryHandler) Update(c *gin.Context) {
	id, ok := promptID(c)
	if !ok {
		return
	}
	var in service.PromptAdminInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return
	}
	item, err := h.service.AdminUpdate(c.Request.Context(), id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// Delete DELETE /api/v1/admin/prompt-library/items/:id
func (h *PromptLibraryHandler) Delete(c *gin.Context) {
	id, ok := promptID(c)
	if !ok {
		return
	}
	if err := h.service.AdminDelete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Batch POST /api/v1/admin/prompt-library/items/batch
func (h *PromptLibraryHandler) Batch(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
		service.PromptBatchOp
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return
	}
	n, err := h.service.AdminBatch(c.Request.Context(), req.IDs, req.PromptBatchOp)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": n})
}

// Stats GET /api/v1/admin/prompt-library/stats
func (h *PromptLibraryHandler) Stats(c *gin.Context) {
	stats, err := h.service.Stats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// Tags GET /api/v1/admin/prompt-library/tags
func (h *PromptLibraryHandler) Tags(c *gin.Context) {
	tags, err := h.service.TagCounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tags)
}

// Sources GET /api/v1/admin/prompt-library/sources
func (h *PromptLibraryHandler) Sources(c *gin.Context) {
	sources, err := h.sync.ListSources(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sources)
}

// UpdateSource PUT /api/v1/admin/prompt-library/sources/:id  body: {"enabled": bool}
func (h *PromptLibraryHandler) UpdateSource(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return
	}
	if err := h.sync.SetEnabled(c.Request.Context(), strings.TrimSpace(c.Param("id")), req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": req.Enabled})
}

// SyncSource POST /api/v1/admin/prompt-library/sources/:id/sync — runs in the background.
func (h *PromptLibraryHandler) SyncSource(c *gin.Context) {
	if err := h.sync.SyncAsync(c.Request.Context(), strings.TrimSpace(c.Param("id"))); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"started": true})
}
