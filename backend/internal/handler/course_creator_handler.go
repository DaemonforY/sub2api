package handler

import (
	"io"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// Creator center (/api/v1/user/creator): apply, write courses (reviewed before they go live), set
// their netdisk link, see sales and withdraw the creator's share.

func creatorCourseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}

func creatorImage(c *gin.Context) ([]byte, bool) {
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

func creatorSiteURL(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") == "http" {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

// reply writes v or the error.
func reply[T any](c *gin.Context, v T, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}

// CreatorHome GET /user/creator
func (h *CourseHandler) CreatorHome(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	home, err := h.svc.CreatorHome(c.Request.Context(), subject.UserID)
	c.Header("Cache-Control", "no-store")
	reply(c, home, err)
}

// CreatorApply POST /user/creator/apply
func (h *CourseHandler) CreatorApply(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.CreatorApplication
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	creator, err := h.svc.ApplyCreator(c.Request.Context(), subject.UserID, in)
	reply(c, creator, err)
}

// CreatorCourses GET /user/creator/courses
func (h *CourseHandler) CreatorCourses(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.svc.CreatorCourses(c.Request.Context(), subject.UserID)
	reply(c, list, err)
}

// CreatorCreate POST /user/creator/courses
func (h *CourseHandler) CreatorCreate(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.CourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	course, err := h.svc.CreateCreatorCourse(c.Request.Context(), subject.UserID, in)
	reply(c, course, err)
}

// CreatorGet GET /user/creator/courses/:id
func (h *CourseHandler) CreatorGet(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	course, err := h.svc.CreatorCourse(c.Request.Context(), subject.UserID, id)
	reply(c, course, err)
}

// CreatorUpdate PUT /user/creator/courses/:id — saves the draft.
func (h *CourseHandler) CreatorUpdate(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	var in service.CourseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	course, err := h.svc.UpdateCreatorCourse(c.Request.Context(), subject.UserID, id, in)
	reply(c, course, err)
}

// CreatorDelete DELETE /user/creator/courses/:id — only a course that was never listed.
func (h *CourseHandler) CreatorDelete(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	reply(c, gin.H{"deleted": true}, h.svc.DeleteCreatorCourse(c.Request.Context(), subject.UserID, id))
}

// CreatorCover POST /user/creator/courses/:id/cover (multipart "image")
func (h *CourseHandler) CreatorCover(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	data, ok := creatorImage(c)
	if !ok {
		return
	}
	course, err := h.svc.SetCreatorCover(c.Request.Context(), subject.UserID, id, data)
	reply(c, course, err)
}

// CreatorImage POST /user/creator/images (multipart "image") — an image for course Markdown.
func (h *CourseHandler) CreatorImage(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	data, ok := creatorImage(c)
	if !ok {
		return
	}
	url, err := h.svc.CreatorUploadImage(c.Request.Context(), subject.UserID, data)
	reply(c, gin.H{"url": url}, err)
}

// CreatorSubmit POST /user/creator/courses/:id/submit — send the draft for review.
func (h *CourseHandler) CreatorSubmit(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	course, err := h.svc.SubmitCreatorCourse(c.Request.Context(), subject.UserID, id)
	reply(c, course, err)
}

// CreatorSale POST /user/creator/courses/:id/sale {on_sale}
func (h *CourseHandler) CreatorSale(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	var in struct {
		OnSale bool `json:"on_sale"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	course, err := h.svc.SetCreatorCourseOnSale(c.Request.Context(), subject.UserID, id, in.OnSale)
	reply(c, course, err)
}

// CreatorDeliveries GET /user/creator/courses/:id/deliveries
func (h *CourseHandler) CreatorDeliveries(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	list, err := h.svc.CreatorDeliveries(c.Request.Context(), subject.UserID, id)
	c.Header("Cache-Control", "no-store")
	reply(c, list, err)
}

// CreatorSaveDelivery POST /user/creator/courses/:id/deliveries — goes live at once.
func (h *CourseHandler) CreatorSaveDelivery(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	var in service.DeliveryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	d, err := h.svc.SaveCreatorDelivery(c.Request.Context(), subject.UserID, id, in, creatorSiteURL(c))
	reply(c, d, err)
}

// CreatorStudents GET /user/creator/courses/:id/students?page=
func (h *CourseHandler) CreatorStudents(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	list, total, err := h.svc.CreatorStudents(c.Request.Context(), subject.UserID, id, page)
	reply(c, gin.H{"items": list, "total": total}, err)
}

// CreatorSales GET /user/creator/sales?page=
func (h *CourseHandler) CreatorSales(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	list, total, err := h.svc.CreatorSales(c.Request.Context(), subject.UserID, page)
	reply(c, gin.H{"items": list, "total": total}, err)
}

// CreatorWithdraw POST /user/creator/withdrawals {cny_amount, method, account, real_name, note}
func (h *CourseHandler) CreatorWithdraw(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.CreatorWithdrawRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	w, err := h.svc.RequestCreatorWithdrawal(c.Request.Context(), subject.UserID, in)
	reply(c, w, err)
}

// CreatorCancelWithdraw POST /user/creator/withdrawals/:id/cancel
func (h *CourseHandler) CreatorCancelWithdraw(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := creatorCourseID(c)
	if !ok {
		return
	}
	w, err := h.svc.CancelCreatorWithdrawal(c.Request.Context(), subject.UserID, id)
	reply(c, w, err)
}
