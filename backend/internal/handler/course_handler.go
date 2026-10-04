package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CourseHandler serves the paid course pages (public), the buyer's courses and their delivery.
type CourseHandler struct {
	svc *service.CourseService
}

func NewCourseHandler(svc *service.CourseService) *CourseHandler {
	return &CourseHandler{svc: svc}
}

func courseViewerID(c *gin.Context) int64 {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	return 0
}

// List GET /api/v1/courses — courses on sale (owned ones marked for a signed-in viewer).
func (h *CourseHandler) List(c *gin.Context) {
	list, err := h.svc.Courses(c.Request.Context(), courseViewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Get GET /api/v1/courses/:slug
func (h *CourseHandler) Get(c *gin.Context) {
	course, err := h.svc.Course(c.Request.Context(), c.Param("slug"), courseViewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// Media GET /api/v1/courses/media/:file — covers and the images of course pages.
func (h *CourseHandler) Media(c *gin.Context) {
	path, ok := h.svc.Media().Path(c.Param("file"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}

// Mine GET /api/v1/user/courses
func (h *CourseHandler) Mine(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.svc.MyCourses(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Delivery GET /api/v1/user/courses/:id/delivery — the netdisk link of a bought course (logged).
func (h *CourseHandler) Delivery(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return
	}
	d, err := h.svc.Delivery(c.Request.Context(), subject.UserID, id, ip.GetClientIP(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, d)
}
