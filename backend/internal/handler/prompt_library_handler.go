package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PromptLibraryHandler serves the canvas prompt library: anonymous listing, and API-key
// authenticated usage events, recommendations and the user's own prompts.
type PromptLibraryHandler struct {
	service *service.PromptLibraryService
}

func NewPromptLibraryHandler(svc *service.PromptLibraryService) *PromptLibraryHandler {
	return &PromptLibraryHandler{service: svc}
}

// PromptListQueryFromRequest reads the listing filters shared by the public and admin APIs.
func PromptListQueryFromRequest(c *gin.Context) service.PromptListQuery {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	return service.PromptListQuery{
		Keyword:  c.Query("q"),
		Scene:    c.Query("scene"),
		Model:    c.Query("model"),
		SourceID: strings.TrimSpace(c.Query("source")),
		Kind:     c.Query("kind"),
		Tag:      c.Query("tag"),
		Sort:     c.Query("sort"),
		Page:     page,
		PageSize: pageSize,
	}
}

// promptFolderQuery reads ?folder= of a 「我的」 listing: "unfiled" = 未分类, an id = that folder.
func promptFolderQuery(c *gin.Context) int64 {
	raw := strings.TrimSpace(c.Query("folder"))
	if raw == "unfiled" {
		return -1
	}
	id, _ := strconv.ParseInt(raw, 10, 64)
	if id < 0 {
		return 0
	}
	return id
}

func promptPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrPromptNotFound)
		return 0, false
	}
	return id, true
}

// List GET /api/v1/prompt-library/items
func (h *PromptLibraryHandler) List(c *gin.Context) {
	res, err := h.service.ListPublic(c.Request.Context(), PromptListQueryFromRequest(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, res)
}

// Cover GET /api/v1/prompt-library/covers/:file
func (h *PromptLibraryHandler) Cover(c *gin.Context) {
	path, ok := h.service.Covers().Path(c.Param("file"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}

// Recommendations GET /api/v1/prompt-library/recommendations?kind=image&limit=24
func (h *PromptLibraryHandler) Recommendations(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, err := h.service.Recommend(c.Request.Context(), userID, c.Query("kind"), limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

// Use POST /api/v1/prompt-library/items/:id/use — the user drew with or copied the prompt.
func (h *PromptLibraryHandler) Use(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := promptPathID(c)
	if !ok {
		return
	}
	count, err := h.service.RecordUse(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"use_count": count})
}

// Favorite PUT /api/v1/prompt-library/items/:id/favorite  body: {"favorited": true}
func (h *PromptLibraryHandler) Favorite(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := promptPathID(c)
	if !ok {
		return
	}
	var req struct {
		Favorited bool `json:"favorited"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return
	}
	count, err := h.service.SetFavorite(c.Request.Context(), userID, id, req.Favorited)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"favorite_count": count})
}

// ListMine GET /api/v1/prompt-library/mine
func (h *PromptLibraryHandler) ListMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	q := PromptListQueryFromRequest(c)
	q.FolderID = promptFolderQuery(c)
	res, err := h.service.ListMine(c.Request.Context(), userID, q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

func bindPromptUserInput(c *gin.Context) (service.PromptUserInput, bool) {
	var in service.PromptUserInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确")
		return in, false
	}
	return in, true
}

// CreateMine POST /api/v1/prompt-library/mine
func (h *PromptLibraryHandler) CreateMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	in, ok := bindPromptUserInput(c)
	if !ok {
		return
	}
	item, err := h.service.CreateMine(c.Request.Context(), userID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateMine PUT /api/v1/prompt-library/mine/:id
func (h *PromptLibraryHandler) UpdateMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := promptPathID(c)
	if !ok {
		return
	}
	in, ok := bindPromptUserInput(c)
	if !ok {
		return
	}
	item, err := h.service.UpdateMine(c.Request.Context(), userID, id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// DeleteMine DELETE /api/v1/prompt-library/mine/:id
func (h *PromptLibraryHandler) DeleteMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := promptPathID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteMine(c.Request.Context(), userID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// UploadCover POST /api/v1/prompt-library/covers  multipart "file"
func (h *PromptLibraryHandler) UploadCover(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.PromptCoverMaxBytes+1<<20)
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, service.ErrPromptCoverInvalid)
		return
	}
	if file.Size > service.PromptCoverMaxBytes {
		response.ErrorFrom(c, service.ErrPromptCoverTooLarge)
		return
	}
	f, err := file.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrPromptCoverInvalid)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, service.PromptCoverMaxBytes+1))
	if err != nil {
		response.ErrorFrom(c, service.ErrPromptCoverInvalid)
		return
	}
	url, err := h.service.UploadCover(c.Request.Context(), userID, data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"url": url})
}

// ImportMine POST /api/v1/prompt-library/mine/import  {items, folder_id}
func (h *PromptLibraryHandler) ImportMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	var in struct {
		Items    []service.PromptUserInput `json:"items"`
		FolderID int64                     `json:"folder_id"`
	}
	// 100 prompts of up to 8000 characters (3 bytes each) plus titles and tags.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 3<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return
	}
	res, err := h.service.ImportMine(c.Request.Context(), userID, in.Items, in.FolderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

// BatchMine POST /api/v1/prompt-library/mine/batch  {ids, action, folder_id}
func (h *PromptLibraryHandler) BatchMine(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	var in struct {
		IDs []int64 `json:"ids"`
		service.PromptMineBatchOp
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return
	}
	n, err := h.service.BatchMine(c.Request.Context(), userID, in.IDs, in.PromptMineBatchOp)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"affected": n})
}

// ListFolders GET /api/v1/prompt-library/folders
func (h *PromptLibraryHandler) ListFolders(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	res, err := h.service.ListFolders(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

type promptFolderRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// CreateFolder POST /api/v1/prompt-library/folders  {name}
func (h *PromptLibraryHandler) CreateFolder(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	var in promptFolderRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return
	}
	folder, err := h.service.CreateFolder(c.Request.Context(), userID, in.Name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, folder)
}

// UpdateFolder PUT /api/v1/prompt-library/folders/:id  {name, sort_order}
func (h *PromptLibraryHandler) UpdateFolder(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrPromptFolderNotFound)
		return
	}
	var in promptFolderRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return
	}
	folder, err := h.service.UpdateFolder(c.Request.Context(), userID, id, in.Name, in.SortOrder)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, folder)
}

// DeleteFolder DELETE /api/v1/prompt-library/folders/:id[?with_items=1]
func (h *PromptLibraryHandler) DeleteFolder(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrPromptFolderNotFound)
		return
	}
	if err := h.service.DeleteFolder(c.Request.Context(), userID, id, c.Query("with_items") == "1"); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
