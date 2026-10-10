package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SiteBuilderHandler serves AI 建站 (/api/v1/site-drafts): start a page, poll it, change it by
// chat, undo, fetch its pictures and publish it through site hosting.
type SiteBuilderHandler struct {
	svc *service.SiteBuilderService
}

func NewSiteBuilderHandler(svc *service.SiteBuilderService) *SiteBuilderHandler {
	return &SiteBuilderHandler{svc: svc}
}

func (h *SiteBuilderHandler) draftID(c *gin.Context) (int64, int64, bool) {
	subject, ok := requireAuth(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "草稿编号不正确（Invalid id）")
		return 0, 0, false
	}
	return subject.UserID, id, true
}

// Config GET /site-drafts/config?key_id= — with a key, also the prices on it.
func (h *SiteBuilderHandler) Config(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keyID, _ := strconv.ParseInt(c.Query("key_id"), 10, 64)
	response.Success(c, h.svc.Config(c.Request.Context(), subject.UserID, keyID))
}

// List GET /site-drafts
func (h *SiteBuilderHandler) List(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), subject.UserID)
	learnReply(c, list, err)
}

// Create POST /site-drafts {key_id, description, style, images}
func (h *SiteBuilderHandler) Create(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.SiteDraftCreateInput
	if !bindLearn(c, &in) {
		return
	}
	d, err := h.svc.Create(c.Request.Context(), subject.UserID, in)
	learnReply(c, d, err)
}

// Get GET /site-drafts/:id
func (h *SiteBuilderHandler) Get(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	d, err := h.svc.Get(c.Request.Context(), userID, id)
	learnReply(c, d, err)
}

// Revise POST /site-drafts/:id/revise {instruction}
func (h *SiteBuilderHandler) Revise(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	var in struct {
		Instruction string `json:"instruction"`
	}
	if !bindLearn(c, &in) {
		return
	}
	d, err := h.svc.Revise(c.Request.Context(), userID, id, in.Instruction)
	learnReply(c, d, err)
}

// Retry POST /site-drafts/:id/retry
func (h *SiteBuilderHandler) Retry(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	d, err := h.svc.Retry(c.Request.Context(), userID, id)
	learnReply(c, d, err)
}

// Undo POST /site-drafts/:id/undo
func (h *SiteBuilderHandler) Undo(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	d, err := h.svc.Undo(c.Request.Context(), userID, id)
	learnReply(c, d, err)
}

// Cancel POST /site-drafts/:id/cancel
func (h *SiteBuilderHandler) Cancel(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.Cancel(c.Request.Context(), userID, id))
}

// Delete DELETE /site-drafts/:id
func (h *SiteBuilderHandler) Delete(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.Delete(c.Request.Context(), userID, id))
}

// Publish POST /site-drafts/:id/publish {site_id?, name?, title?}
func (h *SiteBuilderHandler) Publish(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	var in service.SiteDraftPublishInput
	if !bindLearn(c, &in) {
		return
	}
	site, err := h.svc.Publish(c.Request.Context(), userID, id, in)
	learnReply(c, site, err)
}

// Image GET /site-drafts/:id/images/:n — one of the page's pictures (JPEG).
func (h *SiteBuilderHandler) Image(c *gin.Context) {
	userID, id, ok := h.draftID(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("n"))
	if err != nil || n < 1 {
		response.BadRequest(c, "图片编号不正确（Invalid image）")
		return
	}
	path, err := h.svc.ImageFile(c.Request.Context(), userID, id, n)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	c.Header("Content-Type", "image/jpeg")
	c.File(path)
}
