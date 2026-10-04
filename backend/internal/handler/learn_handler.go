package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// LearnHandler serves /learn's API: public config, the learner's progress and example runs.
type LearnHandler struct {
	svc *service.LearnService
}

func NewLearnHandler(svc *service.LearnService) *LearnHandler {
	return &LearnHandler{svc: svc}
}

// Config GET /api/v1/learn/config
func (h *LearnHandler) Config(c *gin.Context) {
	response.Success(c, h.svc.Config(c.Request.Context()))
}

// Me GET /api/v1/learn/me — completed lessons, free runs left today.
func (h *LearnHandler) Me(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	me, err := h.svc.Me(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, me)
}

// Progress POST /api/v1/learn/progress {lesson_ids} — mark lessons done (or merge local progress).
func (h *LearnHandler) Progress(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		LessonIDs []string `json:"lesson_ids"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLearnLessonID)
		return
	}
	done, err := h.svc.MarkDone(c.Request.Context(), subject.UserID, in.LessonIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"completed": done})
}

// Run POST /api/v1/learn/run {lesson, messages, response_format?, tools?}
func (h *LearnHandler) Run(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.LearnRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确（Invalid request）")
		return
	}
	out, err := h.svc.Run(c.Request.Context(), subject.UserID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
