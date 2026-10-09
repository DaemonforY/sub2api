package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// VideoHandler serves HiveGPT 视频 (video.<domain>): the signed-in user's projects (API-key auth —
// the agent calls the gateway with that key) and the public gallery.
type VideoHandler struct {
	service *service.VideoService
}

func NewVideoHandler(svc *service.VideoService) *VideoHandler {
	return &VideoHandler{service: svc}
}

func videoKey(c *gin.Context) (*service.APIKey, bool) {
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key == nil {
		response.ErrorFrom(c, service.ErrVideoKey)
		return nil, false
	}
	return key, true
}

func bindVideo(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(v); err != nil {
		response.ErrorFrom(c, service.ErrVideoInvalid)
		return false
	}
	return true
}

// Catalog GET /api/v1/video/catalog — styles, genres, categories, ratios and voices.
func (h *VideoHandler) Catalog(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	response.Success(c, gin.H{"styles": service.VideoStyles, "genres": service.VideoGenres, "categories": service.VideoCategories, "ratios": service.VideoRatios, "voices": service.VideoVoices})
}

// Me GET /api/v1/video/me — who is signed in, for the header.
func (h *VideoHandler) Me(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	out := gin.H{"key_name": key.Name}
	if key.User != nil {
		out["name"] = service.VideoAuthorName(key.User.Username, key.User.Email)
		out["balance"] = key.User.Balance
		out["admin"] = key.User.Role == service.RoleAdmin
		out["avatar"] = key.User.AvatarURL
	}
	if key.Group != nil {
		out["group"] = key.Group.Name
	}
	response.Success(c, out)
}

func (h *VideoHandler) List(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	items, err := h.service.List(c.Request.Context(), key.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *VideoHandler) Create(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in service.VideoCreateInput
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.Create(c.Request.Context(), key, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

func (h *VideoHandler) Get(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	p, err := h.service.Get(c.Request.Context(), key.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

func (h *VideoHandler) Delete(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), key.UserID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *VideoHandler) Events(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	after, _ := strconv.ParseInt(c.Query("after"), 10, 64)
	events, err := h.service.Events(c.Request.Context(), key.UserID, c.Param("id"), after)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"events": events})
}

func (h *VideoHandler) Answer(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Answers map[string]string `json:"answers"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.Answer(c.Request.Context(), key, c.Param("id"), in.Answers)
	h.reply(c, p, err)
}

func (h *VideoHandler) Message(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Text  string `json:"text"`
		Scene string `json:"scene"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.Message(c.Request.Context(), key, c.Param("id"), in.Text, in.Scene)
	h.reply(c, p, err)
}

func (h *VideoHandler) Repair(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Scene string `json:"scene"`
		Error string `json:"error"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.Repair(c.Request.Context(), key, c.Param("id"), in.Scene, in.Error)
	h.reply(c, p, err)
}

func (h *VideoHandler) Resume(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	p, err := h.service.Resume(c.Request.Context(), key, c.Param("id"))
	h.reply(c, p, err)
}

func (h *VideoHandler) Stop(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	if err := h.service.Stop(c.Request.Context(), key.UserID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"stopped": true})
}

func (h *VideoHandler) EditScript(c *gin.Context) {
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
	p, err := h.service.EditScript(c.Request.Context(), key, c.Param("id"), in.Scenes)
	h.reply(c, p, err)
}

func (h *VideoHandler) SetVoice(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Voice string `json:"voice"`
		Rate  int    `json:"rate"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.SetVoice(c.Request.Context(), key, c.Param("id"), in.Voice, in.Rate)
	h.reply(c, p, err)
}

func (h *VideoHandler) Versions(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	items, err := h.service.Versions(c.Request.Context(), key.UserID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *VideoHandler) Restore(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	vid, _ := strconv.ParseInt(c.Param("vid"), 10, 64)
	p, err := h.service.Restore(c.Request.Context(), key.UserID, c.Param("id"), vid)
	h.reply(c, p, err)
}

func (h *VideoHandler) Publish(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Category string `json:"category"`
		Title    string `json:"title"`
	}
	if !bindVideo(c, &in) {
		return
	}
	p, err := h.service.Publish(c.Request.Context(), key.UserID, c.Param("id"), in.Category, in.Title)
	h.reply(c, p, err)
}

func (h *VideoHandler) Unpublish(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	p, err := h.service.Unpublish(c.Request.Context(), key.UserID, c.Param("id"))
	h.reply(c, p, err)
}

// OwnAudio GET /api/v1/video/projects/:id/audio/:file
func (h *VideoHandler) OwnAudio(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	h.audio(c, key.UserID, key.User != nil && key.User.Role == service.RoleAdmin)
}

// WorkAudio GET /api/v1/video/works/:id/audio/:file (public works only).
func (h *VideoHandler) WorkAudio(c *gin.Context) {
	h.audio(c, 0, false)
}

func (h *VideoHandler) audio(c *gin.Context, userID int64, admin bool) {
	path, err := h.service.AudioPath(c.Request.Context(), userID, admin, c.Param("id"), c.Param("file"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// File names carry a hash of voice + narration, so a name never changes content.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("Content-Type", "audio/mpeg")
	c.File(path)
}

func (h *VideoHandler) Gallery(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	if size == 0 {
		size = 24
	}
	items, total, err := h.service.Gallery(c.Request.Context(), service.VideoGalleryQuery{
		Category: strings.TrimSpace(c.Query("category")), Mode: strings.TrimSpace(c.Query("mode")), Search: strings.TrimSpace(c.Query("q")),
		Sort: c.Query("sort"), Page: page, PageSize: size,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=30")
	response.Success(c, gin.H{"items": items, "total": total})
}

func (h *VideoHandler) Work(c *gin.Context) {
	p, err := h.service.Work(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, p)
}

func (h *VideoHandler) View(c *gin.Context) {
	h.service.View(c.Request.Context(), c.Param("id"))
	response.Success(c, gin.H{"ok": true})
}

func (h *VideoHandler) Pending(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	items, err := h.service.Pending(c.Request.Context(), key.User)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *VideoHandler) AdminWork(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	p, err := h.service.AdminWork(c.Request.Context(), key.User, c.Param("id"))
	h.reply(c, p, err)
}

func (h *VideoHandler) Review(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in struct {
		Decision string `json:"decision"`
		Category string `json:"category"`
		Featured *bool  `json:"featured"`
	}
	if !bindVideo(c, &in) {
		return
	}
	if err := h.service.Review(c.Request.Context(), key.User, c.Param("id"), in.Decision, in.Category, in.Featured); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Import POST /api/v1/video/admin/import — an admin adds a finished work (scene code included).
func (h *VideoHandler) Import(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	var in service.VideoImportInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrVideoInvalid)
		return
	}
	p, err := h.service.Import(c.Request.Context(), key, in)
	h.reply(c, p, err)
}

func (h *VideoHandler) reply(c *gin.Context, p *service.VideoProject, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

// Service is the video service (the server reads public works for page titles and the sitemap).
func (h *VideoHandler) Service() *service.VideoService { return h.service }
