package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Paid courses (docs/COURSES_PLAN.md): sold through payment orders with order_type = course and
// delivered as a netdisk link with its codes. The delivery is stored encrypted and versioned (a
// leaked or blocked link is replaced; buyers see the newest one) and only returned to buyers
// whose enrollment is active and whose order was not refunded; every view is logged.

const (
	CourseStatusDraft     = "draft"
	CourseStatusPublished = "published"
	CourseStatusArchived  = "archived" // off sale; buyers keep access

	CourseEnrollmentActive  = "active"
	CourseEnrollmentRevoked = "revoked"

	CourseSourcePurchase = "purchase"
	CourseSourceAdmin    = "admin"
	CourseSourceRedeem   = "redeem"

	// settingCourseAffiliateRate: percent of a course order credited to the buyer's inviter (0 = off).
	settingCourseAffiliateRate = "course_affiliate_rate_percent"

	CourseMediaPublicPrefix = "/api/v1/courses/media/"

	// courseDeliveryViewsPerMinute bounds how often one buyer may fetch a course's link.
	courseDeliveryViewsPerMinute = 10
	courseMaxPrice               = 100000
)

var (
	ErrCourseNotFound      = infraerrors.NotFound("COURSE_NOT_FOUND", "课程不存在或已下架（Course not found）")
	ErrCourseOwned         = infraerrors.Conflict("COURSE_ALREADY_OWNED", "你已经购买过这门课程，可以在「我的课程」里查看（Already purchased）")
	ErrCourseNotEnrolled   = infraerrors.Forbidden("COURSE_NOT_ENROLLED", "购买后才能获取课程（Purchase the course first）")
	ErrCourseNoDelivery    = infraerrors.NotFound("COURSE_NO_DELIVERY", "课程资料还在准备中，请稍后再来或联系客服（Course materials are not ready）")
	ErrCourseSlugTaken     = infraerrors.Conflict("COURSE_SLUG_TAKEN", "课程地址已被占用，换一个（Slug taken）")
	ErrCourseHasStudents   = infraerrors.Conflict("COURSE_HAS_STUDENTS", "已有学员的课程不能删除，可以下架（Course has students）")
	ErrCourseCurrency      = infraerrors.BadRequest("COURSE_CURRENCY", "课程只能用人民币支付方式购买（Courses are paid in CNY）")
	ErrCourseTooManyViews  = infraerrors.TooManyRequests("COURSE_TOO_MANY_VIEWS", "查看太频繁，请稍后再试（Too many requests）")
	ErrCourseUserNotFound  = infraerrors.NotFound("COURSE_USER_NOT_FOUND", "找不到这个用户，请填写用户 ID 或注册邮箱（User not found）")
	ErrCourseImageRequired = infraerrors.BadRequest("COURSE_IMAGE_REQUIRED", "请选择一张图片（Pick an image）")
	ErrCourseRedeemInvalid = infraerrors.BadRequest("COURSE_REDEEM_INVALID", "这个兑换码对应的课程不存在或已下架（Course not available）")
)

func errCourseInvalid(what string) error {
	return infraerrors.BadRequest("COURSE_INVALID", fmt.Sprintf("课程信息不完整：%s（Invalid course）", what))
}

var courseSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

// CourseLesson is one line of the outline; Trial marks lessons shown in the free preview.
type CourseLesson struct {
	Title    string `json:"title"`
	Duration string `json:"duration"`
	Trial    bool   `json:"trial"`
}

type CourseSection struct {
	Title   string         `json:"title"`
	Lessons []CourseLesson `json:"lessons"`
}

type Course struct {
	ID            int64           `json:"id"`
	Slug          string          `json:"slug"`
	Title         string          `json:"title"`
	Subtitle      string          `json:"subtitle"`
	Category      string          `json:"category"`
	CoverFile     string          `json:"-"`
	CoverURL      string          `json:"cover_url"`
	Price         float64         `json:"price"`
	OriginalPrice float64         `json:"original_price"`
	IntroMD       string          `json:"intro_md,omitempty"`
	Outline       []CourseSection `json:"outline"`
	TrialMD       string          `json:"trial_md,omitempty"`
	FaqMD         string          `json:"faq_md,omitempty"`
	Status        string          `json:"status"`
	SortOrder     int             `json:"sort_order"`
	// SalePrice applies until SaleEndsAt (limited-time price); 0 = none.
	SalePrice  float64    `json:"sale_price"`
	SaleEndsAt *time.Time `json:"sale_ends_at,omitempty"`
	// EduDiscount: verified students / teachers get the education discount on this course.
	EduDiscount bool `json:"edu_discount"`
	// TrialVideoURL: a direct video link (OSS / CDN) or a Bilibili page, shown as the free preview.
	TrialVideoURL string `json:"trial_video_url"`
	// CurrentPrice is what the viewer pays now (limited-time price, then their education discount).
	CurrentPrice float64 `json:"current_price"`
	SaleActive   bool    `json:"sale_active"`
	EduApplied   bool    `json:"edu_applied"`
	// EduPercent (when the viewer is not verified): the discount verification would give.
	EduPercent   float64 `json:"edu_percent,omitempty"`
	LessonCount  int     `json:"lesson_count"`
	StudentCount int     `json:"student_count"`
	// Owned: the viewer has an active enrollment.
	Owned     bool      `json:"owned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Admin only.
	Revenue         float64 `json:"revenue,omitempty"`
	DeliveryVersion int     `json:"delivery_version,omitempty"`
	// Last 30 days (admin only): page views, orders created, orders paid; refunds ever.
	Views30d  int `json:"views_30d,omitempty"`
	Orders30d int `json:"orders_30d,omitempty"`
	Paid30d   int `json:"paid_30d,omitempty"`
	Refunds   int `json:"refunds,omitempty"`
}

// CourseDeliveryRecord is a stored delivery version (fields encrypted).
type CourseDeliveryRecord struct {
	ID          int64
	CourseID    int64
	Version     int
	LinkEnc     string
	CodeEnc     string
	PasswordEnc string
	Note        string
	CreatedBy   int64
	CreatedAt   time.Time
}

// CourseDelivery is what a buyer (or the admin) sees.
type CourseDelivery struct {
	Version   int       `json:"version"`
	Link      string    `json:"link"`
	Code      string    `json:"code"`
	Password  string    `json:"password"`
	Note      string    `json:"note"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CourseEnrollment struct {
	UserID        int64      `json:"user_id"`
	CourseID      int64      `json:"course_id"`
	OrderID       int64      `json:"order_id,omitempty"`
	Source        string     `json:"source"`
	Status        string     `json:"status"`
	Note          string     `json:"note,omitempty"`
	FirstViewedAt *time.Time `json:"first_viewed_at,omitempty"`
	LastViewedAt  *time.Time `json:"last_viewed_at,omitempty"`
	ViewCount     int        `json:"view_count"`
	SeenVersion   int        `json:"seen_version"`
	// Refunded: the order was refunded (access ends).
	Refunded  bool      `json:"refunded"`
	CreatedAt time.Time `json:"created_at"`
	// Admin list only.
	UserEmail string `json:"user_email,omitempty"`
	Username  string `json:"username,omitempty"`
	// RecentIPs: distinct IPs that fetched the link in the last 7 days (sharing shows up here).
	RecentIPs int `json:"recent_ips"`
}

// Active: the enrollment gives access.
func (e *CourseEnrollment) Active() bool {
	return e != nil && e.Status == CourseEnrollmentActive && !e.Refunded
}

// MyCourse is a course in the buyer's list.
type MyCourse struct {
	Course     Course           `json:"course"`
	Enrollment CourseEnrollment `json:"enrollment"`
	// Updated: the link changed since the buyer last looked.
	Updated bool `json:"updated"`
}

type CourseRepository interface {
	ListCourses(ctx context.Context, statuses []string) ([]Course, error)
	GetCourse(ctx context.Context, id int64) (*Course, error)
	GetCourseBySlug(ctx context.Context, slug string) (*Course, error)
	CreateCourse(ctx context.Context, c *Course) error
	UpdateCourse(ctx context.Context, c *Course) error
	DeleteCourse(ctx context.Context, id int64) error

	GetEnrollment(ctx context.Context, userID, courseID int64) (*CourseEnrollment, error)
	OwnedCourses(ctx context.Context, userID int64) (map[int64]bool, error)
	UpsertEnrollment(ctx context.Context, e *CourseEnrollment) error
	SetEnrollmentStatus(ctx context.Context, userID, courseID int64, status string) error
	ListMyCourses(ctx context.Context, userID int64) ([]MyCourse, error)
	ListEnrollments(ctx context.Context, courseID int64, limit, offset int) ([]CourseEnrollment, int, error)

	LatestDelivery(ctx context.Context, courseID int64) (*CourseDeliveryRecord, error)
	ListDeliveries(ctx context.Context, courseID int64) ([]CourseDeliveryRecord, error)
	AddDelivery(ctx context.Context, d *CourseDeliveryRecord) error
	CountAccess(ctx context.Context, courseID, userID int64, since time.Time) (int, error)
	RecordAccess(ctx context.Context, courseID, userID int64, ip string, version int) error

	FindUserID(ctx context.Context, idOrEmail string) (int64, error)
	AddView(ctx context.Context, courseID int64) error
	ActiveStudents(ctx context.Context, courseID int64) ([]CourseStudent, error)
}

// CourseStudent is a buyer to notify.
type CourseStudent struct {
	UserID   int64
	Email    string
	Username string
}

// coursePricer prices for verified students (GrowthService).
type coursePricer interface {
	DiscountedPrice(ctx context.Context, userID int64, price float64) (float64, bool)
	GetSettings(ctx context.Context) (*GrowthSettings, error)
}

// courseSettings stores the course rebate rate.
type courseSettings interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, values map[string]string) error
}

type CourseService struct {
	repo      CourseRepository
	media     *CommunityMediaStore
	encryptor SecretEncryptor
	now       func() time.Time
	pricer    coursePricer
	settings  courseSettings
	mailer    *NotificationEmailService
}

// SetPricer enables education-discounted course prices.
func (s *CourseService) SetPricer(p coursePricer) { s.pricer = p }

// SetSettings enables the course rebate rate setting.
func (s *CourseService) SetSettings(st courseSettings) { s.settings = st }

// SetMailer enables course emails (purchase, link updated).
func (s *CourseService) SetMailer(m *NotificationEmailService) { s.mailer = m }

func NewCourseService(repo CourseRepository, media *CommunityMediaStore, encryptor SecretEncryptor) *CourseService {
	return &CourseService{repo: repo, media: media, encryptor: encryptor, now: time.Now}
}

func (s *CourseService) Media() *CommunityMediaStore { return s.media }

func CourseMediaURL(name string) string {
	if name == "" {
		return ""
	}
	return CourseMediaPublicPrefix + name
}

func lessonCount(outline []CourseSection) int {
	n := 0
	for _, sec := range outline {
		n += len(sec.Lessons)
	}
	return n
}

func (s *CourseService) decorate(c *Course, owned bool, full bool) {
	c.CoverURL = CourseMediaURL(c.CoverFile)
	if c.Outline == nil {
		c.Outline = []CourseSection{}
	}
	c.LessonCount = lessonCount(c.Outline)
	c.Owned = owned
	if !full {
		c.IntroMD, c.TrialMD, c.FaqMD = "", "", ""
	}
}

// priceFor sets what userID pays now: the limited-time price while it runs, then the education
// discount for verified students (or, for others, the discount verification would give).
func (s *CourseService) priceFor(ctx context.Context, c *Course, userID int64, edu *GrowthSettings) {
	price := c.Price
	c.SaleActive = c.SalePrice > 0 && c.SalePrice < c.Price && c.SaleEndsAt != nil && s.now().Before(*c.SaleEndsAt)
	if c.SaleActive {
		price = c.SalePrice
	}
	c.CurrentPrice, c.EduApplied, c.EduPercent = price, false, 0
	if !c.EduDiscount || s.pricer == nil {
		return
	}
	if userID > 0 {
		if discounted, ok := s.pricer.DiscountedPrice(ctx, userID, price); ok {
			c.CurrentPrice, c.EduApplied = discounted, true
			return
		}
	}
	if edu != nil && edu.EduVerifyEnabled && edu.EduDiscountPercent > 0 {
		c.EduPercent = edu.EduDiscountPercent
	}
}

func (s *CourseService) eduSettings(ctx context.Context) *GrowthSettings {
	if s.pricer == nil {
		return nil
	}
	settings, err := s.pricer.GetSettings(ctx)
	if err != nil {
		return nil
	}
	return settings
}

func hideAdminFields(c *Course) {
	c.Revenue, c.DeliveryVersion, c.Views30d, c.Orders30d, c.Paid30d, c.Refunds = 0, 0, 0, 0, 0, 0
}

// Courses on sale, with the viewer's purchases marked.
func (s *CourseService) Courses(ctx context.Context, viewerID int64) ([]Course, error) {
	list, err := s.repo.ListCourses(ctx, []string{CourseStatusPublished})
	if err != nil {
		return nil, err
	}
	owned := map[int64]bool{}
	if viewerID > 0 {
		if owned, err = s.repo.OwnedCourses(ctx, viewerID); err != nil {
			return nil, err
		}
	}
	if list == nil {
		list = []Course{}
	}
	edu := s.eduSettings(ctx)
	for i := range list {
		s.decorate(&list[i], owned[list[i].ID], false)
		s.priceFor(ctx, &list[i], viewerID, edu)
		hideAdminFields(&list[i])
	}
	return list, nil
}

// Course is a course page: on sale, or off sale for its buyers.
func (s *CourseService) Course(ctx context.Context, slug string, viewerID int64) (*Course, error) {
	c, err := s.repo.GetCourseBySlug(ctx, strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	owned := false
	if viewerID > 0 {
		e, err := s.repo.GetEnrollment(ctx, viewerID, c.ID)
		if err != nil {
			return nil, err
		}
		owned = e.Active()
	}
	visible := c.Status == CourseStatusPublished || (c.Status == CourseStatusArchived && owned)
	if !visible {
		return nil, ErrCourseNotFound
	}
	s.decorate(c, owned, true)
	s.priceFor(ctx, c, viewerID, s.eduSettings(ctx))
	hideAdminFields(c)
	if !owned {
		_ = s.repo.AddView(ctx, c.ID)
	}
	return c, nil
}

// MyCourses lists the user's courses (refunded ones stay listed, marked).
func (s *CourseService) MyCourses(ctx context.Context, userID int64) ([]MyCourse, error) {
	list, err := s.repo.ListMyCourses(ctx, userID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []MyCourse{}
	}
	for i := range list {
		s.decorate(&list[i].Course, list[i].Enrollment.Active(), false)
		list[i].Updated = list[i].Enrollment.SeenVersion > 0 && list[i].Course.DeliveryVersion > list[i].Enrollment.SeenVersion
		hideAdminFields(&list[i].Course)
	}
	return list, nil
}

// Delivery returns the course's link to a buyer and logs the view.
func (s *CourseService) Delivery(ctx context.Context, userID, courseID int64, ip string) (*CourseDelivery, error) {
	e, err := s.repo.GetEnrollment(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	if !e.Active() {
		return nil, ErrCourseNotEnrolled
	}
	recent, err := s.repo.CountAccess(ctx, courseID, userID, s.now().Add(-time.Minute))
	if err != nil {
		return nil, err
	}
	if recent >= courseDeliveryViewsPerMinute {
		return nil, ErrCourseTooManyViews
	}
	rec, err := s.repo.LatestDelivery(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrCourseNoDelivery
	}
	d, err := s.decrypt(rec)
	if err != nil {
		return nil, err
	}
	if len(ip) > 64 {
		ip = ip[:64]
	}
	if err := s.repo.RecordAccess(ctx, courseID, userID, ip, rec.Version); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *CourseService) decrypt(rec *CourseDeliveryRecord) (*CourseDelivery, error) {
	open := func(v string) (string, error) {
		if v == "" {
			return "", nil
		}
		return s.encryptor.Decrypt(v)
	}
	link, err := open(rec.LinkEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt course link: %w", err)
	}
	code, err := open(rec.CodeEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt course code: %w", err)
	}
	password, err := open(rec.PasswordEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt course password: %w", err)
	}
	return &CourseDelivery{Version: rec.Version, Link: link, Code: code, Password: password, Note: rec.Note, UpdatedAt: rec.CreatedAt}, nil
}

// --- Payment hooks ---------------------------------------------------------------------------

// CourseForOrder prices a course order (CurrentPrice is the buyer's price): on sale and not
// already bought.
func (s *CourseService) CourseForOrder(ctx context.Context, userID, courseID int64) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.Status != CourseStatusPublished || c.Price <= 0 {
		return nil, ErrCourseNotFound
	}
	e, err := s.repo.GetEnrollment(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	if e.Active() {
		return nil, ErrCourseOwned
	}
	s.priceFor(ctx, c, userID, nil)
	return c, nil
}

// EnrollFromOrder grants a paid course (idempotent; a refunded or revoked enrollment is renewed).
func (s *CourseService) EnrollFromOrder(ctx context.Context, userID, courseID, orderID int64) error {
	return s.repo.UpsertEnrollment(ctx, &CourseEnrollment{UserID: userID, CourseID: courseID, OrderID: orderID, Source: CourseSourcePurchase, Status: CourseEnrollmentActive})
}

// --- Admin -----------------------------------------------------------------------------------

// CourseInput is the editable part of a course.
type CourseInput struct {
	Slug          string          `json:"slug"`
	Title         string          `json:"title"`
	Subtitle      string          `json:"subtitle"`
	Category      string          `json:"category"`
	Price         float64         `json:"price"`
	OriginalPrice float64         `json:"original_price"`
	IntroMD       string          `json:"intro_md"`
	Outline       []CourseSection `json:"outline"`
	TrialMD       string          `json:"trial_md"`
	FaqMD         string          `json:"faq_md"`
	Status        string          `json:"status"`
	SortOrder     int             `json:"sort_order"`
	SalePrice     float64         `json:"sale_price"`
	SaleEndsAt    *time.Time      `json:"sale_ends_at"`
	EduDiscount   bool            `json:"edu_discount"`
	TrialVideoURL string          `json:"trial_video_url"`
}

func roundCents(v float64) float64 { return math.Round(v*100) / 100 }

func normalizeOutline(in []CourseSection) []CourseSection {
	out := []CourseSection{}
	for _, sec := range in {
		sec.Title = cleanText(sec.Title, 100)
		lessons := []CourseLesson{}
		for _, l := range sec.Lessons {
			l.Title, l.Duration = cleanText(l.Title, 120), cleanText(l.Duration, 20)
			if l.Title != "" {
				lessons = append(lessons, l)
			}
		}
		sec.Lessons = lessons
		if sec.Title != "" || len(lessons) > 0 {
			out = append(out, sec)
		}
	}
	return out
}

func (in CourseInput) apply(c *Course) error {
	c.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if !courseSlugRe.MatchString(c.Slug) {
		return errCourseInvalid("地址只能用 3–64 位小写字母、数字和连字符")
	}
	c.Title = cleanText(in.Title, 120)
	if c.Title == "" {
		return errCourseInvalid("标题")
	}
	c.Subtitle, c.Category = cleanText(in.Subtitle, 200), cleanText(in.Category, 32)
	c.Price, c.OriginalPrice = roundCents(in.Price), roundCents(in.OriginalPrice)
	if c.Price <= 0 || c.Price > courseMaxPrice || math.IsNaN(c.Price) {
		return errCourseInvalid("价格须大于 0")
	}
	if c.OriginalPrice < 0 || c.OriginalPrice > courseMaxPrice || math.IsNaN(c.OriginalPrice) || (c.OriginalPrice > 0 && c.OriginalPrice <= c.Price) {
		c.OriginalPrice = 0
	}
	c.IntroMD, c.TrialMD, c.FaqMD = strings.TrimSpace(in.IntroMD), strings.TrimSpace(in.TrialMD), strings.TrimSpace(in.FaqMD)
	if len(c.IntroMD)+len(c.TrialMD)+len(c.FaqMD) > 200_000 {
		return errCourseInvalid("介绍太长")
	}
	c.Outline = normalizeOutline(in.Outline)
	switch in.Status {
	case CourseStatusDraft, CourseStatusPublished, CourseStatusArchived:
		c.Status = in.Status
	default:
		c.Status = CourseStatusDraft
	}
	c.SortOrder = in.SortOrder
	c.SalePrice, c.SaleEndsAt = roundCents(in.SalePrice), in.SaleEndsAt
	if c.SalePrice < 0 || math.IsNaN(c.SalePrice) || c.SalePrice >= c.Price {
		c.SalePrice = 0
	}
	if c.SalePrice > 0 && c.SaleEndsAt == nil {
		return errCourseInvalid("限时价需要设置截止时间")
	}
	if c.SalePrice == 0 {
		c.SaleEndsAt = nil
	}
	c.EduDiscount = in.EduDiscount
	c.TrialVideoURL = strings.TrimSpace(in.TrialVideoURL)
	if c.TrialVideoURL != "" && (!strings.HasPrefix(c.TrialVideoURL, "https://") || len(c.TrialVideoURL) > 500) {
		return errCourseInvalid("试看视频须是 https:// 开头的链接")
	}
	return nil
}

// AdminCourses lists every course with students, revenue and the delivery version.
func (s *CourseService) AdminCourses(ctx context.Context) ([]Course, error) {
	list, err := s.repo.ListCourses(ctx, nil)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Course{}
	}
	for i := range list {
		s.decorate(&list[i], false, false)
	}
	return list, nil
}

func (s *CourseService) AdminCourse(ctx context.Context, id int64) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	s.decorate(c, false, true)
	return c, nil
}

func (s *CourseService) CreateCourse(ctx context.Context, in CourseInput) (*Course, error) {
	c := &Course{}
	if err := in.apply(c); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.AdminCourse(ctx, c.ID)
}

func (s *CourseService) UpdateCourse(ctx context.Context, id int64, in CourseInput) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	if err := in.apply(c); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.AdminCourse(ctx, id)
}

func (s *CourseService) DeleteCourse(ctx context.Context, id int64) error {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCourseNotFound
	}
	if c.StudentCount > 0 {
		return ErrCourseHasStudents
	}
	if err := s.repo.DeleteCourse(ctx, id); err != nil {
		return err
	}
	s.media.Remove(c.CoverFile)
	return nil
}

// SetCover replaces a course's cover image.
func (s *CourseService) SetCover(ctx context.Context, id int64, data []byte) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	stored, err := s.media.SaveImage(data)
	if err != nil {
		return nil, err
	}
	s.media.Remove(stored.ThumbFile)
	old := c.CoverFile
	c.CoverFile = stored.File
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		s.media.Remove(stored.File)
		return nil, err
	}
	s.media.Remove(old)
	return s.AdminCourse(ctx, id)
}

// UploadImage stores an image for a course's Markdown and returns its URL.
func (s *CourseService) UploadImage(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrCourseImageRequired
	}
	stored, err := s.media.SaveImage(data)
	if err != nil {
		return "", err
	}
	s.media.Remove(stored.ThumbFile)
	return CourseMediaURL(stored.File), nil
}

// DeliveryInput is a new delivery version.
type DeliveryInput struct {
	Link     string `json:"link"`
	Code     string `json:"code"`
	Password string `json:"password"`
	Note     string `json:"note"`
	// Notify emails the course's buyers that the link changed.
	Notify bool `json:"notify"`
}

func (s *CourseService) seal(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	return s.encryptor.Encrypt(v)
}

// SaveDelivery stores a new version of the course's link (buyers see it at once); with Notify,
// buyers are emailed (in the background) to look under 我的课程 at siteURL.
func (s *CourseService) SaveDelivery(ctx context.Context, courseID, adminID int64, in DeliveryInput, siteURL string) (*CourseDelivery, error) {
	c, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	link := strings.TrimSpace(in.Link)
	if !strings.HasPrefix(link, "https://") || len(link) > 500 {
		return nil, errCourseInvalid("网盘链接须以 https:// 开头")
	}
	rec := &CourseDeliveryRecord{CourseID: courseID, Note: cleanText(in.Note, 2000), CreatedBy: adminID}
	if rec.LinkEnc, err = s.seal(link); err != nil {
		return nil, err
	}
	if rec.CodeEnc, err = s.seal(cleanText(in.Code, 32)); err != nil {
		return nil, err
	}
	if rec.PasswordEnc, err = s.seal(cleanText(in.Password, 64)); err != nil {
		return nil, err
	}
	if err := s.repo.AddDelivery(ctx, rec); err != nil {
		return nil, err
	}
	if in.Notify && rec.Version > 1 {
		go s.notifyDeliveryUpdated(context.WithoutCancel(ctx), c, siteURL)
	}
	return s.decrypt(rec)
}

// Deliveries lists a course's delivery versions, newest first (admin).
func (s *CourseService) Deliveries(ctx context.Context, courseID int64) ([]CourseDelivery, error) {
	recs, err := s.repo.ListDeliveries(ctx, courseID)
	if err != nil {
		return nil, err
	}
	out := make([]CourseDelivery, 0, len(recs))
	for i := range recs {
		d, err := s.decrypt(&recs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, nil
}

// Enrollments lists a course's students, newest first (admin).
func (s *CourseService) Enrollments(ctx context.Context, courseID int64, page, pageSize int) ([]CourseEnrollment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}
	list, total, err := s.repo.ListEnrollments(ctx, courseID, pageSize, (page-1)*pageSize)
	if list == nil {
		list = []CourseEnrollment{}
	}
	return list, total, err
}

// Grant opens a course for a user by hand (offline payment, gift).
func (s *CourseService) Grant(ctx context.Context, courseID int64, user, note string) error {
	c, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCourseNotFound
	}
	uid, err := s.repo.FindUserID(ctx, strings.TrimSpace(user))
	if err != nil {
		return err
	}
	if uid == 0 {
		return ErrCourseUserNotFound
	}
	return s.repo.UpsertEnrollment(ctx, &CourseEnrollment{UserID: uid, CourseID: courseID, Source: CourseSourceAdmin, Status: CourseEnrollmentActive, Note: cleanText(note, 200)})
}

// Revoke ends a user's access to a course (refunds end it on their own).
func (s *CourseService) Revoke(ctx context.Context, courseID, userID int64) error {
	return s.repo.SetEnrollmentStatus(ctx, userID, courseID, CourseEnrollmentRevoked)
}

// MarshalOutline / UnmarshalOutline store the outline as JSON.
func MarshalOutline(o []CourseSection) []byte {
	if o == nil {
		o = []CourseSection{}
	}
	b, _ := json.Marshal(o)
	return b
}

func UnmarshalOutline(b []byte) []CourseSection {
	var o []CourseSection
	if len(b) == 0 || json.Unmarshal(b, &o) != nil || o == nil {
		return []CourseSection{}
	}
	return o
}

// --- Emails --------------------------------------------------------------------------------------

func myCoursesURL(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + "/my-courses"
}

func (s *CourseService) sendCourseEmail(ctx context.Context, event string, c *Course, student CourseStudent, siteURL string) error {
	if s.mailer == nil || student.Email == "" {
		return nil
	}
	return s.mailer.Send(ctx, NotificationEmailSendInput{
		Event:          event,
		RecipientEmail: student.Email,
		RecipientName:  firstNonEmpty(student.Username, student.Email),
		UserID:         student.UserID,
		SourceType:     "course",
		SourceID:       fmt.Sprintf("%d", c.ID),
		Variables:      map[string]string{"course_title": c.Title, "courses_url": myCoursesURL(siteURL)},
	})
}

func (s *CourseService) notifyDeliveryUpdated(ctx context.Context, c *Course, siteURL string) {
	students, err := s.repo.ActiveStudents(ctx, c.ID)
	if err != nil {
		slog.Warn("course link update: list students", "course_id", c.ID, "err", err)
		return
	}
	for _, st := range students {
		if err := s.sendCourseEmail(ctx, NotificationEmailEventCourseDeliveryUpdated, c, st, siteURL); err != nil {
			slog.Warn("course link update email failed", "course_id", c.ID, "user_id", st.UserID, "err", err)
		}
	}
}

// NotifyPurchased emails the buyer where to find a course they just bought.
func (s *CourseService) NotifyPurchased(ctx context.Context, courseID int64, student CourseStudent, siteURL string) error {
	c, err := s.repo.GetCourse(ctx, courseID)
	if err != nil || c == nil {
		return err
	}
	return s.sendCourseEmail(ctx, NotificationEmailEventCoursePurchaseSuccess, c, student, siteURL)
}

// --- Redeem codes (type course, value = course ID) ------------------------------------------------

// CheckRedeem: the code's course can be opened for userID (before the code is used up).
func (s *CourseService) CheckRedeem(ctx context.Context, userID, courseID int64) error {
	c, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if c == nil || c.Status == CourseStatusDraft {
		return ErrCourseRedeemInvalid
	}
	e, err := s.repo.GetEnrollment(ctx, userID, courseID)
	if err != nil {
		return err
	}
	if e.Active() {
		return ErrCourseOwned
	}
	return nil
}

// EnrollRedeem opens the course; inside the redeem transaction when ctx carries one.
func (s *CourseService) EnrollRedeem(ctx context.Context, userID, courseID int64, code string) error {
	return s.repo.UpsertEnrollment(ctx, &CourseEnrollment{UserID: userID, CourseID: courseID, Source: CourseSourceRedeem, Status: CourseEnrollmentActive, Note: "兑换码 " + cleanText(code, 64)})
}

// --- Settings ------------------------------------------------------------------------------------

// CourseSettings: the share of a course order credited to the buyer's inviter.
type CourseSettings struct {
	AffiliateRatePercent float64 `json:"affiliate_rate_percent"`
}

// AffiliateRate is the course rebate rate in percent (0 = course orders earn no rebate).
func (s *CourseService) AffiliateRate(ctx context.Context) float64 {
	if s.settings == nil {
		return 0
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingCourseAffiliateRate})
	if err != nil {
		return 0
	}
	rate, err := strconv.ParseFloat(strings.TrimSpace(values[settingCourseAffiliateRate]), 64)
	if err != nil || rate < 0 || rate > 100 || math.IsNaN(rate) {
		return 0
	}
	return rate
}

func (s *CourseService) Settings(ctx context.Context) CourseSettings {
	return CourseSettings{AffiliateRatePercent: s.AffiliateRate(ctx)}
}

func (s *CourseService) SaveSettings(ctx context.Context, in CourseSettings) (CourseSettings, error) {
	if in.AffiliateRatePercent < 0 || in.AffiliateRatePercent > 50 || math.IsNaN(in.AffiliateRatePercent) {
		return CourseSettings{}, errCourseInvalid("返利比例须在 0–50% 之间")
	}
	if s.settings == nil {
		return CourseSettings{}, nil
	}
	rate := strconv.FormatFloat(roundCents(in.AffiliateRatePercent), 'f', -1, 64)
	if err := s.settings.SetMultiple(ctx, map[string]string{settingCourseAffiliateRate: rate}); err != nil {
		return CourseSettings{}, err
	}
	return s.Settings(ctx), nil
}
