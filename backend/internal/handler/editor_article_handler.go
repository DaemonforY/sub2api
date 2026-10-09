package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AI 写文章 (/api/v1/editor/articles): the editor starts an article, polls it, edits / confirms the
// outline and fetches the pictures; pushing to the draft box stays in the browser.

func (h *EditorHandler) articleID(c *gin.Context) (int64, int64, bool) {
	subject, ok := requireAuth(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "文章编号不正确（Invalid id）")
		return 0, 0, false
	}
	return subject.UserID, id, true
}

// ArticleConfig GET /editor/articles/config?key_id= — with a key, also the price of one picture on it.
func (h *EditorHandler) ArticleConfig(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keyID, _ := strconv.ParseInt(c.Query("key_id"), 10, 64)
	response.Success(c, h.articles.Config(c.Request.Context(), subject.UserID, keyID))
}

// ArticleList GET /editor/articles
func (h *EditorHandler) ArticleList(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.articles.List(c.Request.Context(), subject.UserID)
	learnReply(c, list, err)
}

// ArticleCreate POST /editor/articles {key_id, topic, materials, audience, tone, length, images, search}
func (h *EditorHandler) ArticleCreate(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.ArticleCreateInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.articles.Create(c.Request.Context(), subject.UserID, in)
	learnReply(c, p, err)
}

// ArticleGet GET /editor/articles/:id
func (h *EditorHandler) ArticleGet(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	p, err := h.articles.Get(c.Request.Context(), userID, id)
	learnReply(c, p, err)
}

// ArticleOutline POST /editor/articles/:id/outline {outline?, feedback?, confirm}
func (h *EditorHandler) ArticleOutline(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	var in service.ArticleOutlineInput
	if !bindLearn(c, &in) {
		return
	}
	p, err := h.articles.Outline(c.Request.Context(), userID, id, in)
	learnReply(c, p, err)
}

// ArticleRetry POST /editor/articles/:id/retry
func (h *EditorHandler) ArticleRetry(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	p, err := h.articles.Retry(c.Request.Context(), userID, id)
	learnReply(c, p, err)
}

// ArticleCancel POST /editor/articles/:id/cancel
func (h *EditorHandler) ArticleCancel(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.articles.Cancel(c.Request.Context(), userID, id))
}

// ArticlePushed POST /editor/articles/:id/pushed — the editor pushed it to the draft box.
func (h *EditorHandler) ArticlePushed(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.articles.Pushed(c.Request.Context(), userID, id))
}

// ArticleDelete DELETE /editor/articles/:id
func (h *EditorHandler) ArticleDelete(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.articles.Delete(c.Request.Context(), userID, id))
}

// ArticleImage GET /editor/articles/:id/images/:n — one of the article's pictures (JPEG).
func (h *EditorHandler) ArticleImage(c *gin.Context) {
	userID, id, ok := h.articleID(c)
	if !ok {
		return
	}
	n, err := strconv.Atoi(c.Param("n"))
	if err != nil || n < 0 {
		response.BadRequest(c, "图片编号不正确（Invalid image）")
		return
	}
	path, err := h.articles.ImageFile(c.Request.Context(), userID, id, n)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Content-Type", "image/jpeg")
	c.File(path)
}
