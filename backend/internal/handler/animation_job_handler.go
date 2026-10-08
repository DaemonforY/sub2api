package handler

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnimationJobHandler runs the canvas' AI 动画 generations in the background, authenticated with the
// user's API key (which the server then uses to call its own gateway).
type AnimationJobHandler struct {
	service *service.AnimationJobService
}

func NewAnimationJobHandler(svc *service.AnimationJobService) *AnimationJobHandler {
	return &AnimationJobHandler{service: svc}
}

// Create POST /api/v1/animation-jobs  body: {"model": "...", "messages": [...], "meta": {...}}
func (h *AnimationJobHandler) Create(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	key, _ := middleware.GetAPIKeyFromContext(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	var in service.AnimationJobInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrAnimationJobInvalid)
		return
	}
	job, err := h.service.Create(c.Request.Context(), userID, key, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}

// List GET /api/v1/animation-jobs — the newest 30 jobs, without SVGs.
func (h *AnimationJobHandler) List(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	jobs, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"jobs": jobs})
}

// Get GET /api/v1/animation-jobs/:id — includes the SVG once it succeeded.
func (h *AnimationJobHandler) Get(c *gin.Context) {
	userID, id, ok := animationJobParams(c)
	if !ok {
		return
	}
	job, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}

// Cancel POST /api/v1/animation-jobs/:id/cancel
func (h *AnimationJobHandler) Cancel(c *gin.Context) {
	userID, id, ok := animationJobParams(c)
	if !ok {
		return
	}
	job, err := h.service.Cancel(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}

func animationJobParams(c *gin.Context) (int64, int64, bool) {
	userID, ok := appStateUserID(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrAnimationJobNotFound)
		return 0, 0, false
	}
	return userID, id, true
}
