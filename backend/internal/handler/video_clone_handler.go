package handler

import (
	"io"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 我的音色 (voice clones) on HiveGPT 视频.

func videoCloneParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrVideoCloneNotFound)
		return 0, false
	}
	return id, true
}

// readCloneAudio reads the "audio" part of a multipart form; it answers the request when that fails.
func readCloneAudio(c *gin.Context) ([]byte, map[string][]string, bool) {
	form, ok := parseVideoUpload(c, service.VideoCloneMaxBytes, service.ErrVideoCloneAudio)
	if !ok {
		return nil, nil, false
	}
	defer func() { _ = form.RemoveAll() }()
	file, closeFile, err := videoFormFile(form, "audio")
	defer closeFile()
	if err != nil || file == nil || file.Size > service.VideoCloneMaxBytes {
		response.ErrorFrom(c, service.ErrVideoCloneAudio)
		return nil, nil, false
	}
	audio, err := io.ReadAll(io.NewSectionReader(file.Reader, 0, file.Size))
	if err != nil {
		response.ErrorFrom(c, service.ErrVideoCloneAudio)
		return nil, nil, false
	}
	return audio, form.Value, true
}

func formValue(values map[string][]string, name string) string {
	if v := values[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// CloneInfo GET /api/v1/video/clone-info — whether cloning is on, the fee, the script and the consent.
func (h *VideoHandler) CloneInfo(c *gin.Context) {
	response.Success(c, h.service.CloneInfo(c.Request.Context()))
}

// Clones GET /api/v1/video/clones — the user's voices (training ones are checked with 豆包).
func (h *VideoHandler) Clones(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	list, err := h.service.ListClones(c.Request.Context(), key.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": list, "info": h.service.CloneInfo(c.Request.Context())})
}

// CreateClone POST /api/v1/video/clones — multipart: audio, name, consent=1.
func (h *VideoHandler) CreateClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	audio, values, ok := readCloneAudio(c)
	if !ok {
		return
	}
	clone, err := h.service.CreateClone(c.Request.Context(), key, service.VideoCloneInput{
		Name: formValue(values, "name"), Audio: audio, Consent: formValue(values, "consent") == "1", IP: c.ClientIP(),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, clone)
}

// RetrainClone POST /api/v1/video/clones/:cid/retrain — multipart: audio.
func (h *VideoHandler) RetrainClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoCloneParam(c)
	if !ok {
		return
	}
	audio, _, ok := readCloneAudio(c)
	if !ok {
		return
	}
	clone, err := h.service.RetrainClone(c.Request.Context(), key, id, audio)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, clone)
}

// RenameClone PUT /api/v1/video/clones/:cid {name}.
func (h *VideoHandler) RenameClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoCloneParam(c)
	if !ok {
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if !bindVideo(c, &in) {
		return
	}
	clone, err := h.service.RenameClone(c.Request.Context(), key.UserID, id, in.Name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, clone)
}

// DeleteClone DELETE /api/v1/video/clones/:cid.
func (h *VideoHandler) DeleteClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoCloneParam(c)
	if !ok {
		return
	}
	if err := h.service.DeleteClone(c.Request.Context(), key.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// PreviewClone POST /api/v1/video/clones/:cid/preview {text} — an MP3 of a short sentence.
func (h *VideoHandler) PreviewClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoCloneParam(c)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !bindVideo(c, &in) {
		return
	}
	speech, err := h.service.PreviewClone(c.Request.Context(), key.UserID, id, in.Text)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(200, "audio/mpeg", speech.Audio)
}

// AdminClones GET /api/v1/video/admin/clones — every live clone with its slot and consent record.
func (h *VideoHandler) AdminClones(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	list, err := h.service.AdminClones(c.Request.Context(), key.User)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// AdminDisableClone POST /api/v1/video/admin/clones/:cid/disable {disable}.
func (h *VideoHandler) AdminDisableClone(c *gin.Context) {
	key, ok := videoKey(c)
	if !ok {
		return
	}
	id, ok := videoCloneParam(c)
	if !ok {
		return
	}
	var in struct {
		Disable bool `json:"disable"`
	}
	if !bindVideo(c, &in) {
		return
	}
	if err := h.service.AdminDisableClone(c.Request.Context(), key.User, id, in.Disable); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
