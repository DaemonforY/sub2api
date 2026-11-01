package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AI 做贴图 (/api/v1/editor/tietu): the editor starts a 贴图 from a poem or a theme, polls it, confirms
// the plan, picks the cover, redraws panels and fetches the pictures; framing and pushing to the
// draft box stay in the browser. Admins only for now.

func (h *EditorHandler) tietuUser(c *gin.Context) (int64, bool) {
	subject, ok := requireAuth(c)
	if !ok {
		return 0, false
	}
	if h.tietu == nil {
		response.ErrorFrom(c, service.ErrTietuAdmin)
		return 0, false
	}
	if role, ok := middleware2.GetUserRoleFromContext(c); !ok || role != service.RoleAdmin {
		response.ErrorFrom(c, service.ErrTietuAdmin)
		return 0, false
	}
	return subject.UserID, true
}

func (h *EditorHandler) tietuID(c *gin.Context) (int64, int64, bool) {
	userID, ok := h.tietuUser(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "编号不正确（Invalid id）")
		return 0, 0, false
	}
	return userID, id, true
}

// TietuConfig GET /editor/tietu/config?key_id= — styles, panel limits and the price of one picture.
func (h *EditorHandler) TietuConfig(c *gin.Context) {
	userID, ok := h.tietuUser(c)
	if !ok {
		return
	}
	keyID, _ := strconv.ParseInt(c.Query("key_id"), 10, 64)
	response.Success(c, h.tietu.Config(c.Request.Context(), userID, keyID))
}

// TietuList GET /editor/tietu
func (h *EditorHandler) TietuList(c *gin.Context) {
	userID, ok := h.tietuUser(c)
	if !ok {
		return
	}
	list, err := h.tietu.List(c.Request.Context(), userID)
	learnReply(c, list, err)
}

// TietuCreate POST /editor/tietu {key_id, input, theme, mode, panels, style, style_text, notes}
func (h *EditorHandler) TietuCreate(c *gin.Context) {
	userID, ok := h.tietuUser(c)
	if !ok {
		return
	}
	var in service.TietuCreateInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.Create(c.Request.Context(), userID, in)
	learnReply(c, p, err)
}

// TietuGet GET /editor/tietu/:id
func (h *EditorHandler) TietuGet(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	p, err := h.tietu.Get(c.Request.Context(), userID, id)
	learnReply(c, p, err)
}

// TietuPick POST /editor/tietu/:id/pick {index}
func (h *EditorHandler) TietuPick(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	var in struct {
		Index int `json:"index"`
	}
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.Pick(c.Request.Context(), userID, id, in.Index)
	learnReply(c, p, err)
}

// TietuPlan POST /editor/tietu/:id/plan {plan?, feedback?, confirm}
func (h *EditorHandler) TietuPlan(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	var in service.TietuPlanInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.Plan(c.Request.Context(), userID, id, in)
	learnReply(c, p, err)
}

// TietuCover POST /editor/tietu/:id/cover {pick} | {redraw, note}
func (h *EditorHandler) TietuCover(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	var in service.TietuCoverInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.Cover(c.Request.Context(), userID, id, in)
	learnReply(c, p, err)
}

// TietuRedraw POST /editor/tietu/:id/redraw {name, note}
func (h *EditorHandler) TietuRedraw(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	var in service.TietuRedrawInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.Redraw(c.Request.Context(), userID, id, in)
	learnReply(c, p, err)
}

// TietuPost PUT /editor/tietu/:id/post {title, body}
func (h *EditorHandler) TietuPost(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	var in service.TietuPostInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.tietu.SavePost(c.Request.Context(), userID, id, in)
	learnReply(c, p, err)
}

// TietuRetry POST /editor/tietu/:id/retry
func (h *EditorHandler) TietuRetry(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	p, err := h.tietu.Retry(c.Request.Context(), userID, id)
	learnReply(c, p, err)
}

// TietuCancel POST /editor/tietu/:id/cancel
func (h *EditorHandler) TietuCancel(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.tietu.Cancel(c.Request.Context(), userID, id))
}

// TietuPushed POST /editor/tietu/:id/pushed
func (h *EditorHandler) TietuPushed(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.tietu.Pushed(c.Request.Context(), userID, id))
}

// TietuDelete DELETE /editor/tietu/:id
func (h *EditorHandler) TietuDelete(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.tietu.Delete(c.Request.Context(), userID, id))
}

// TietuImage GET /editor/tietu/:id/images/:name — c1..c3 cover candidates, p1..p8 panels (JPEG).
// Pictures can be redrawn under the same name, so the browser revalidates every time.
func (h *EditorHandler) TietuImage(c *gin.Context) {
	userID, id, ok := h.tietuID(c)
	if !ok {
		return
	}
	path, err := h.tietu.ImageFile(c.Request.Context(), userID, id, c.Param("name"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	c.Header("Content-Type", "image/jpeg")
	c.File(path)
}
