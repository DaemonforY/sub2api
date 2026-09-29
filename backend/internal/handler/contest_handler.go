package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContestHandler serves the public contest pages and user actions (submit / vote).
type ContestHandler struct {
	service *service.ContestService
}

// NewContestHandler creates the handler.
func NewContestHandler(svc *service.ContestService) *ContestHandler {
	return &ContestHandler{service: svc}
}

func contestViewerID(c *gin.Context) int64 {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	return 0
}

func parseContestPathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}

// List GET /api/v1/contests
func (h *ContestHandler) List(c *gin.Context) {
	list, err := h.service.ListPublicContests(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Get GET /api/v1/contests/:id
func (h *ContestHandler) Get(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	detail, err := h.service.GetPublicContest(c.Request.Context(), id, contestViewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// ListEntries GET /api/v1/contests/:id/entries?sort=votes|new
func (h *ContestHandler) ListEntries(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 60 {
		pageSize = 60
	}
	entries, total, err := h.service.ListPublicEntries(c.Request.Context(), id, contestViewerID(c), c.Query("sort"), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, int64(total), page, pageSize)
}

// Leaderboard GET /api/v1/contests/:id/leaderboard
func (h *ContestHandler) Leaderboard(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, err := h.service.Leaderboard(c.Request.Context(), id, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rows)
}

// SubmitEntry POST /api/v1/contests/:id/entries (multipart: image, title, description, prompt)
func (h *ContestHandler) SubmitEntry(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	userID := contestViewerID(c)
	if userID <= 0 {
		response.Unauthorized(c, "Login required")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ContestImageMaxBytes+1<<20)
	file, err := c.FormFile("image")
	if err != nil {
		response.BadRequest(c, "image file is required")
		return
	}
	if file.Size > service.ContestImageMaxBytes {
		response.ErrorFrom(c, service.ErrContestImageTooLarge)
		return
	}
	f, err := file.Open()
	if err != nil {
		response.BadRequest(c, "cannot read image")
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, service.ContestImageMaxBytes+1))
	if err != nil {
		response.BadRequest(c, "cannot read image")
		return
	}
	entry, err := h.service.SubmitEntry(c.Request.Context(), id, userID, service.ContestEntryInput{
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		Prompt:      c.PostForm("prompt"),
		Image:       data,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, entry)
}

// WithdrawEntry DELETE /api/v1/contests/:id/entries/:entryId
func (h *ContestHandler) WithdrawEntry(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	entryID, ok := parseContestPathID(c, "entryId")
	if !ok {
		return
	}
	if err := h.service.WithdrawEntry(c.Request.Context(), id, entryID, contestViewerID(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"withdrawn": true})
}

// Vote POST /api/v1/contests/:id/entries/:entryId/vote
func (h *ContestHandler) Vote(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	entryID, ok := parseContestPathID(c, "entryId")
	if !ok {
		return
	}
	if err := h.service.Vote(c.Request.Context(), id, entryID, contestViewerID(c), ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"voted": true})
}

// Unvote DELETE /api/v1/contests/:id/entries/:entryId/vote
func (h *ContestHandler) Unvote(c *gin.Context) {
	id, ok := parseContestPathID(c, "id")
	if !ok {
		return
	}
	entryID, ok := parseContestPathID(c, "entryId")
	if !ok {
		return
	}
	if err := h.service.Unvote(c.Request.Context(), id, entryID, contestViewerID(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"voted": false})
}

// Image GET /api/v1/contest-images/:file
func (h *ContestHandler) Image(c *gin.Context) {
	path, ok := h.service.Images().Path(c.Param("file"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}
