package admin

import (
	"io"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CourseHandler manages paid courses: editing, covers, the netdisk delivery and students.
type CourseHandler struct {
	svc *service.CourseService
}

func NewCourseHandler(svc *service.CourseService) *CourseHandler {
	return &CourseHandler{svc: svc}
}

func courseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}

// readImage reads the "image" form file (bounded like community uploads).
func readImage(c *gin.Context) ([]byte, bool) {
	fh, err := c.FormFile("image")
	if err != nil {
		response.ErrorFrom(c, service.ErrCourseImageRequired)
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrCourseImageRequired)
		return nil, false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, service.CommunityImageMaxBytes+1))
	if err != nil {
		response.ErrorFrom(c, service.ErrCommunityImageInvalid)
		return nil, false
	}
	return data, true
}

// List GET /api/v1/admin/courses
func (h *CourseHandler) List(c *gin.Context) {
	list, err := h.svc.AdminCourses(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Get GET /api/v1/admin/courses/:id
func (h *CourseHandler) Get(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	course, err := h.svc.AdminCourse(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// Create POST /api/v1/admin/courses
func (h *CourseHandler) Create(c *gin.Context) {
	var in service.CourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	course, err := h.svc.CreateCourse(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// Update PUT /api/v1/admin/courses/:id
func (h *CourseHandler) Update(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	var in service.CourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	course, err := h.svc.UpdateCourse(c.Request.Context(), id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// Delete DELETE /api/v1/admin/courses/:id (only without students)
func (h *CourseHandler) Delete(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCourse(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Cover POST /api/v1/admin/courses/:id/cover (multipart: image)
func (h *CourseHandler) Cover(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	data, ok := readImage(c)
	if !ok {
		return
	}
	course, err := h.svc.SetCover(c.Request.Context(), id, data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// UploadImage POST /api/v1/admin/courses/images (multipart: image) — an image for course Markdown.
func (h *CourseHandler) UploadImage(c *gin.Context) {
	data, ok := readImage(c)
	if !ok {
		return
	}
	url, err := h.svc.UploadImage(data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"url": url})
}

// Deliveries GET /api/v1/admin/courses/:id/deliveries — versions, newest first.
func (h *CourseHandler) Deliveries(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	list, err := h.svc.Deliveries(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// requestSiteURL is the site the admin is on (for links in emails).
func requestSiteURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") == "http" {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

// Settings GET /api/v1/admin/courses/settings {affiliate_rate_percent}
func (h *CourseHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// SaveSettings PUT /api/v1/admin/courses/settings {affiliate_rate_percent}
func (h *CourseHandler) SaveSettings(c *gin.Context) {
	var in service.CourseSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// SaveDelivery POST /api/v1/admin/courses/:id/deliveries {link, code, password, note, notify}
func (h *CourseHandler) SaveDelivery(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	var in service.DeliveryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var adminID int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		adminID = subject.UserID
	}
	d, err := h.svc.SaveDelivery(c.Request.Context(), id, adminID, in, requestSiteURL(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, d)
}

// Enrollments GET /api/v1/admin/courses/:id/enrollments?page=
func (h *CourseHandler) Enrollments(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	list, total, err := h.svc.Enrollments(c.Request.Context(), id, page, 30)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": list, "total": total})
}

// Grant POST /api/v1/admin/courses/:id/enrollments {user: id or email, note}
func (h *CourseHandler) Grant(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	var in struct {
		User string `json:"user"`
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.Grant(c.Request.Context(), id, in.User, in.Note); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Revoke DELETE /api/v1/admin/courses/:id/enrollments/:user_id
func (h *CourseHandler) Revoke(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), id, userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// --- Creators --------------------------------------------------------------------------------------

// Creators GET /api/v1/admin/courses/creators?status=
func (h *CourseHandler) Creators(c *gin.Context) {
	list, err := h.svc.AdminCreators(c.Request.Context(), c.Query("status"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// UpdateCreator PUT /api/v1/admin/courses/creators/:user_id {status, commission_percent, admin_note}
func (h *CourseHandler) UpdateCreator(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user id")
		return
	}
	var in service.CreatorUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	creator, err := h.svc.UpdateCreator(c.Request.Context(), userID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, creator)
}

// Reviews GET /api/v1/admin/courses/reviews — creator courses waiting for review, with drafts.
func (h *CourseHandler) Reviews(c *gin.Context) {
	list, err := h.svc.ReviewCourses(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Review POST /api/v1/admin/courses/:id/review {approve, note}
func (h *CourseHandler) Review(c *gin.Context) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	var in struct {
		Approve bool   `json:"approve"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	course, err := h.svc.ReviewCourse(c.Request.Context(), id, in.Approve, in.Note)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, course)
}

// CreatorWithdrawals GET /api/v1/admin/courses/creator-withdrawals?status=&page=
func (h *CourseHandler) CreatorWithdrawals(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	list, total, err := h.svc.AdminCreatorWithdrawals(c.Request.Context(), service.CreatorWithdrawFilter{Status: c.Query("status"), Page: page, PageSize: 20})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, gin.H{"items": list, "total": total})
}

// ResolveCreatorWithdrawal POST /api/v1/admin/courses/creator-withdrawals/:id/(paid|reject) {note}
func (h *CourseHandler) resolveCreatorWithdrawal(c *gin.Context, paid bool) {
	id, ok := courseID(c)
	if !ok {
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&in)
	var adminID int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		adminID = subject.UserID
	}
	w, err := h.svc.ResolveCreatorWithdrawal(c.Request.Context(), id, adminID, paid, in.Note)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}

func (h *CourseHandler) PayCreatorWithdrawal(c *gin.Context)    { h.resolveCreatorWithdrawal(c, true) }
func (h *CourseHandler) RejectCreatorWithdrawal(c *gin.Context) { h.resolveCreatorWithdrawal(c, false) }
