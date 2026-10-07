package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// EditorHandler serves the 公众号 editor's API (/api/v1/editor): WeChat draft box, article import,
// the image proxy and AI. All routes need a signed-in user.
type EditorHandler struct {
	svc *service.EditorService
}

func NewEditorHandler(svc *service.EditorService) *EditorHandler {
	return &EditorHandler{svc: svc}
}

const editorUploadMax = 10<<20 + 1<<20

// WechatCheck POST /editor/wechat/check
func (h *EditorHandler) WechatCheck(c *gin.Context) {
	if _, ok := requireAuth(c); !ok {
		return
	}
	var in service.EditorCredentials
	if !bindLearn(c, &in) {
		return
	}
	if err := h.svc.WechatCheck(c.Request.Context(), in); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// WechatUpload POST /editor/wechat/upload (multipart: appid, secret, kind, file)
func (h *EditorHandler) WechatUpload(c *gin.Context) {
	if _, ok := requireAuth(c); !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, editorUploadMax)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择要上传的图片，最大 10MB（File required）")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, editorUploadMax))
	if err != nil {
		response.BadRequest(c, "读取图片失败（Read failed）")
		return
	}
	creds := service.EditorCredentials{AppID: c.PostForm("appid"), Secret: c.PostForm("secret")}
	out, err := h.svc.WechatUpload(c.Request.Context(), creds, c.PostForm("kind"), header.Filename, data)
	learnReply(c, out, err)
}

// WechatDraft POST /editor/wechat/draft
func (h *EditorHandler) WechatDraft(c *gin.Context) {
	if _, ok := requireAuth(c); !ok {
		return
	}
	var in service.EditorDraftInput
	if !bindLearn(c, &in) {
		return
	}
	id, err := h.svc.WechatDraft(c.Request.Context(), in)
	learnReply(c, gin.H{"media_id": id}, err)
}

// ImportArticle POST /editor/article/import {url}
func (h *EditorHandler) ImportArticle(c *gin.Context) {
	if _, ok := requireAuth(c); !ok {
		return
	}
	var in struct {
		URL string `json:"url"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.ImportArticle(c.Request.Context(), in.URL)
	learnReply(c, out, err)
}

// Image GET /editor/image?url=
func (h *EditorHandler) Image(c *gin.Context) {
	if _, ok := requireAuth(c); !ok {
		return
	}
	data, mime, err := h.svc.FetchImage(c.Request.Context(), c.Query("url"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, mime, data)
}

// AIText POST /editor/ai/text — streams {"delta"} events, then {"done":true} or {"error"}.
func (h *EditorHandler) AIText(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.EditorAITextInput
	if !bindLearn(c, &in) {
		return
	}
	started := false
	send := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", b); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	err := h.svc.AIText(c.Request.Context(), subject.UserID, in, func(delta string) error {
		if !started {
			started = true
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			c.Status(http.StatusOK)
		}
		return send(gin.H{"delta": delta})
	})
	if !started {
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, gin.H{"done": true})
		return
	}
	if err != nil {
		_ = send(gin.H{"error": infraerrors.Message(err)})
		return
	}
	_ = send(gin.H{"done": true})
}

// AIImage POST /editor/ai/image
func (h *EditorHandler) AIImage(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.EditorAIImageInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.AIImage(c.Request.Context(), subject.UserID, in)
	learnReply(c, out, err)
}
