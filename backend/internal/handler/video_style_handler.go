package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// 我的风格 style cards, 先审镜头剧本 and 复盘 (service/video_style.go).

func videoStyleParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrVideoStyleNotFound)
		return 0, false
	}
	return id, true
}

// Styles GET /api/v1/video/styles — the user's style cards.
func (h *VideoHandler) Styles(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	items, err := h.service.ListStyles(c.Request.Context(), key.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

// SaveStyle POST /api/v1/video/styles — save a film's look as a style card.
func (h *VideoHandler) SaveStyle(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		ProjectID string   `json:"project_id"`
		Name      string   `json:"name"`
		Rules     []string `json:"rules"`
	}
	if !bindVideo(c, &in) {
		return
	}
	st, err := h.service.SaveStyle(c.Request.Context(), key.UserID, in.ProjectID, in.Name, in.Rules)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, st)
}

// UpdateStyle PUT /api/v1/video/styles/:sid — rename and/or replace the rules.
func (h *VideoHandler) UpdateStyle(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoStyleParam(c)
	if !ok {
		return
	}
	var in struct {
		Name  *string  `json:"name"`
		Rules []string `json:"rules"`
	}
	if !bindVideo(c, &in) {
		return
	}
	st, err := h.service.UpdateStyle(c.Request.Context(), key.UserID, id, in.Name, in.Rules)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, st)
}

// DeleteStyle DELETE /api/v1/video/styles/:sid
func (h *VideoHandler) DeleteStyle(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoStyleParam(c)
	if !ok {
		return
	}
	if err := h.service.DeleteStyle(c.Request.Context(), key.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ApproveScript POST /api/v1/video/projects/:id/script/approve — confirm the shot script (with the
// user's edits) and start making the film.
func (h *VideoHandler) ApproveScript(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Scenes []service.VideoSceneEdit `json:"scenes"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.ApproveScript(c.Request.Context(), key, c.Param("id"), in.Scenes)
	h.reply(c, p, err)
}

// Retro POST /api/v1/video/projects/:id/retro — the film's 复盘 (one model call on the user's key).
func (h *VideoHandler) Retro(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	out, err := h.service.Retro(c.Request.Context(), key, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
