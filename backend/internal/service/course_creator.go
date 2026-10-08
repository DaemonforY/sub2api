package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Course creators: users apply to sell their own courses; once an admin approves them they write
// courses whose edits are kept as a draft until an admin approves them (first listing and every
// later change — the netdisk delivery is theirs to change at once). Every paid order of a creator's
// course records what the buyer paid (payment fee excluded) and the creator's commission rate at
// that moment; after the settlement period the rest is theirs to withdraw, paid by hand like 返利提现.
// Refunds are read from the order, so a refund after a withdrawal leaves a negative balance that
// later sales make up.

const (
	CreatorStatusPending   = "pending"
	CreatorStatusApproved  = "approved"
	CreatorStatusRejected  = "rejected"
	CreatorStatusSuspended = "suspended"

	// Review states of a creator course's draft ('' = nothing to review, e.g. platform courses).
	CourseReviewDraft    = "draft"    // edited, not submitted
	CourseReviewPending  = "pending"  // waiting for an admin
	CourseReviewApproved = "approved" // live matches the draft
	CourseReviewRejected = "rejected" // see ReviewNote

	settingCreatorEnabled     = "course_creator_enabled"
	settingCreatorCommission  = "course_creator_commission_percent"
	settingCreatorSettleDays  = "course_creator_settle_days"
	settingCreatorWithdrawMin = "course_creator_withdraw_min_cny"

	creatorCommissionDefault  = 20.0
	creatorSettleDaysDefault  = 7
	creatorWithdrawMinDefault = 50.0
	creatorMaxCourses         = 20
)

var (
	ErrCreatorClosed         = infraerrors.Forbidden("CREATOR_CLOSED", "暂未开放课程创作者申请（Creator applications are closed）")
	ErrCreatorNotApproved    = infraerrors.Forbidden("CREATOR_NOT_APPROVED", "审核通过成为创作者后才能发布课程（Approved creators only）")
	ErrCreatorAlreadyApplied = infraerrors.Conflict("CREATOR_ALREADY_APPLIED", "你已经是创作者或申请正在审核中（Already applied）")
	ErrCreatorNotFound       = infraerrors.NotFound("CREATOR_NOT_FOUND", "找不到这个创作者（Creator not found）")
	ErrCreatorTooManyCourses = infraerrors.Conflict("CREATOR_TOO_MANY_COURSES", fmt.Sprintf("每位创作者最多 %d 门课程（Too many courses）", creatorMaxCourses))
	ErrCreatorReviewPending  = infraerrors.Conflict("CREATOR_REVIEW_PENDING", "课程正在审核中，审核完成前不能重复提交（Review pending）")
	ErrCreatorNothingToSub   = infraerrors.Conflict("CREATOR_NOTHING_TO_SUBMIT", "没有需要审核的修改（Nothing to submit）")
	ErrCreatorNeedsDelivery  = infraerrors.BadRequest("CREATOR_NEEDS_DELIVERY", "请先填写网盘链接再提交审核，学员购买后要靠它拿到课程（Add the netdisk link first）")
	ErrCreatorNeverApproved  = infraerrors.Conflict("CREATOR_NEVER_APPROVED", "课程还没通过审核，请先提交审核（Not approved yet）")
	ErrCreatorCourseLive     = infraerrors.Conflict("CREATOR_COURSE_LIVE", "上架过的课程不能删除，可以下架（Listed courses can't be deleted）")
	ErrCreatorNotReviewable  = infraerrors.Conflict("CREATOR_NOT_REVIEWABLE", "这门课程没有待审核的修改，请刷新页面（Nothing to review）")
	ErrCreatorReasonRequired = infraerrors.BadRequest("CREATOR_REASON_REQUIRED", "请填写原因，创作者会看到（A reason is required）")
)

// CourseCreator is a creator's profile and application.
type CourseCreator struct {
	UserID       int64      `json:"user_id"`
	Status       string     `json:"status"`
	DisplayName  string     `json:"display_name"`
	Bio          string     `json:"bio"`
	Contact      string     `json:"contact"`
	Plan         string     `json:"plan"`
	AdminNote    string     `json:"admin_note"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UserEmail    string     `json:"user_email,omitempty"`
	Username     string     `json:"username,omitempty"`
	CourseCount  int        `json:"course_count"`
	OnSaleCount  int        `json:"on_sale_count"`
	PendingCount int        `json:"pending_count"`
	// CommissionPercent is the creator's own rate (nil = the site default).
	CommissionPercent *float64 `json:"commission_percent"`
	// Admin list only.
	Balance *CreatorBalance `json:"balance,omitempty"`
}

// CourseDraft is a creator's edit waiting to go live.
type CourseDraft struct {
	CourseInput
	CoverFile string `json:"cover_file"`
	CoverURL  string `json:"cover_url,omitempty"`
}

// CreatorBalance is a creator's money in CNY; Available can go below zero after a late refund.
type CreatorBalance struct {
	Orders    int     `json:"orders"`
	Gross     float64 `json:"gross"`     // buyers paid, payment fees and refunds excluded
	Net       float64 `json:"net"`       // the creator's share of Gross
	Frozen    float64 `json:"frozen"`    // share still in the settlement period (or in a refund)
	Paid      float64 `json:"paid"`      // withdrawn and paid
	Pending   float64 `json:"pending"`   // withdrawal waiting for payment
	Available float64 `json:"available"` // settled share − paid − pending
}

// CreatorSale is one order of a creator's course.
type CreatorSale struct {
	OrderID           int64   `json:"order_id"`
	CourseID          int64   `json:"course_id"`
	CourseTitle       string  `json:"course_title"`
	Buyer             string  `json:"buyer"`
	Gross             float64 `json:"gross"`
	CommissionPercent float64 `json:"commission_percent"`
	Net               float64 `json:"net"`
	// Status: frozen / available / refunding / refunded / partially_refunded.
	Status      string    `json:"status"`
	AvailableAt time.Time `json:"available_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreatorWithdrawal is a withdrawal request; Account / RealName are plaintext once decrypted.
type CreatorWithdrawal struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	UserEmail   string     `json:"user_email,omitempty"`
	DisplayName string     `json:"display_name,omitempty"`
	CNYAmount   float64    `json:"cny_amount"`
	Method      string     `json:"method"`
	Account     string     `json:"account"`
	RealName    string     `json:"real_name"`
	UserNote    string     `json:"user_note"`
	Status      string     `json:"status"`
	AdminNote   string     `json:"admin_note"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreatorWithdrawFilter filters withdrawal lists (UserID > 0: one creator's).
type CreatorWithdrawFilter struct {
	UserID   int64
	Status   string
	Page     int
	PageSize int
}

// CourseCreatorRepository is the creator part of CourseRepository.
type CourseCreatorRepository interface {
	GetCreator(ctx context.Context, userID int64) (*CourseCreator, error)
	// SaveCreatorApplication files or refiles an application (not over an approved or suspended one).
	SaveCreatorApplication(ctx context.Context, c *CourseCreator) error
	ListCreators(ctx context.Context, status string) ([]CourseCreator, error)
	UpdateCreator(ctx context.Context, userID int64, status string, commission *float64, note string) error
	// ArchiveCreatorCourses takes a creator's courses off sale.
	ArchiveCreatorCourses(ctx context.Context, userID int64) error

	// RecordCreatorSale stores the creator's share of a completed course order (idempotent; no-op
	// for platform courses).
	RecordCreatorSale(ctx context.Context, orderID int64, defaultPercent float64, settleDays int) error
	CreatorBalance(ctx context.Context, userID int64) (*CreatorBalance, error)
	CreatorSales(ctx context.Context, userID int64, limit, offset int) ([]CreatorSale, int, error)

	// CreateCreatorWithdrawal checks the balance with check under the creator's row lock, then stores w.
	CreateCreatorWithdrawal(ctx context.Context, w *CreatorWithdrawal, check func(*CreatorBalance) error) error
	ListCreatorWithdrawals(ctx context.Context, f CreatorWithdrawFilter) ([]CreatorWithdrawal, int, error)
	// ResolveCreatorWithdrawal moves a pending withdrawal to status (ownerID > 0: only that user's).
	ResolveCreatorWithdrawal(ctx context.Context, id int64, status, note string, reviewerID, ownerID int64) (*CreatorWithdrawal, error)
}

// --- Settings ------------------------------------------------------------------------------------

func (s *CourseService) settingValues(ctx context.Context, keys ...string) map[string]string {
	if s.settings == nil {
		return map[string]string{}
	}
	values, err := s.settings.GetMultiple(ctx, keys)
	if err != nil || values == nil {
		return map[string]string{}
	}
	return values
}

func parseSettingFloat(v string, def, lo, hi float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(f) || f < lo || f > hi {
		return def
	}
	return f
}

// creatorSettings fills the creator part of CourseSettings.
func (s *CourseService) creatorSettings(ctx context.Context, out *CourseSettings) {
	v := s.settingValues(ctx, settingCreatorEnabled, settingCreatorCommission, settingCreatorSettleDays, settingCreatorWithdrawMin)
	out.CreatorEnabled = strings.TrimSpace(v[settingCreatorEnabled]) == "true"
	out.CreatorCommissionPercent = parseSettingFloat(v[settingCreatorCommission], creatorCommissionDefault, 0, 100)
	out.CreatorSettleDays = int(parseSettingFloat(v[settingCreatorSettleDays], creatorSettleDaysDefault, 0, 365))
	out.CreatorWithdrawMinCNY = parseSettingFloat(v[settingCreatorWithdrawMin], creatorWithdrawMinDefault, 1, 100000)
}

func validateCreatorSettings(in CourseSettings) error {
	if math.IsNaN(in.CreatorCommissionPercent) || in.CreatorCommissionPercent < 0 || in.CreatorCommissionPercent > 100 {
		return errCourseInvalid("手续费比例须在 0–100% 之间")
	}
	if in.CreatorSettleDays < 0 || in.CreatorSettleDays > 365 {
		return errCourseInvalid("结算期须在 0–365 天之间")
	}
	if math.IsNaN(in.CreatorWithdrawMinCNY) || in.CreatorWithdrawMinCNY < 1 || in.CreatorWithdrawMinCNY > 100000 {
		return errCourseInvalid("最低提现金额须在 1–100000 元之间")
	}
	return nil
}

func creatorSettingValues(in CourseSettings) map[string]string {
	return map[string]string{
		settingCreatorEnabled:     strconv.FormatBool(in.CreatorEnabled),
		settingCreatorCommission:  strconv.FormatFloat(roundCents(in.CreatorCommissionPercent), 'f', -1, 64),
		settingCreatorSettleDays:  strconv.Itoa(in.CreatorSettleDays),
		settingCreatorWithdrawMin: strconv.FormatFloat(roundCents(in.CreatorWithdrawMinCNY), 'f', -1, 64),
	}
}

// --- Applications --------------------------------------------------------------------------------

// CreatorApplication is the 申请成为创作者 form.
type CreatorApplication struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Contact     string `json:"contact"`
	Plan        string `json:"plan"`
}

// CreatorHome is the creator center's header: the profile (nil before applying), the rules and money.
type CreatorHome struct {
	Enabled           bool                `json:"enabled"`
	Creator           *CourseCreator      `json:"creator"`
	CommissionPercent float64             `json:"commission_percent"`
	SettleDays        int                 `json:"settle_days"`
	WithdrawMinCNY    float64             `json:"withdraw_min_cny"`
	MaxCourses        int                 `json:"max_courses"`
	Balance           *CreatorBalance     `json:"balance,omitempty"`
	Withdrawals       []CreatorWithdrawal `json:"withdrawals"`
	LastMethod        string              `json:"last_method,omitempty"`
	LastAccount       string              `json:"last_account,omitempty"`
	LastRealName      string              `json:"last_real_name,omitempty"`
}

func (s *CourseService) commissionFor(c *CourseCreator, st CourseSettings) float64 {
	if c != nil && c.CommissionPercent != nil {
		return *c.CommissionPercent
	}
	return st.CreatorCommissionPercent
}

// CreatorHome returns the creator center for userID.
func (s *CourseService) CreatorHome(ctx context.Context, userID int64) (*CreatorHome, error) {
	st := s.Settings(ctx)
	c, err := s.repo.GetCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &CreatorHome{Enabled: st.CreatorEnabled, Creator: c, CommissionPercent: s.commissionFor(c, st), SettleDays: st.CreatorSettleDays,
		WithdrawMinCNY: st.CreatorWithdrawMinCNY, MaxCourses: creatorMaxCourses, Withdrawals: []CreatorWithdrawal{}}
	if c == nil || (c.Status != CreatorStatusApproved && c.Status != CreatorStatusSuspended) {
		return out, nil
	}
	if out.Balance, err = s.repo.CreatorBalance(ctx, userID); err != nil {
		return nil, err
	}
	list, _, err := s.repo.ListCreatorWithdrawals(ctx, CreatorWithdrawFilter{UserID: userID, Page: 1, PageSize: 20})
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.decryptPayee(&list[i])
	}
	if len(list) > 0 {
		out.LastMethod, out.LastAccount, out.LastRealName = list[0].Method, list[0].Account, list[0].RealName
	}
	out.Withdrawals = list
	return out, nil
}

// ApplyCreator files (or, after a rejection, refiles) an application.
func (s *CourseService) ApplyCreator(ctx context.Context, userID int64, in CreatorApplication) (*CourseCreator, error) {
	if !s.Settings(ctx).CreatorEnabled {
		return nil, ErrCreatorClosed
	}
	existing, err := s.repo.GetCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != CreatorStatusRejected {
		return nil, ErrCreatorAlreadyApplied
	}
	c := &CourseCreator{UserID: userID, Status: CreatorStatusPending, DisplayName: cleanText(in.DisplayName, 40), Bio: cleanText(in.Bio, 1000),
		Contact: cleanText(in.Contact, 100), Plan: cleanText(in.Plan, 2000)}
	if c.DisplayName == "" || c.Contact == "" || c.Plan == "" {
		return nil, errCourseInvalid("请填写讲师名称、联系方式和课程计划")
	}
	if err := s.repo.SaveCreatorApplication(ctx, c); err != nil {
		return nil, err
	}
	return s.repo.GetCreator(ctx, userID)
}

// approvedCreator returns userID's profile when they may write courses.
func (s *CourseService) approvedCreator(ctx context.Context, userID int64) (*CourseCreator, error) {
	c, err := s.repo.GetCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.Status != CreatorStatusApproved {
		return nil, ErrCreatorNotApproved
	}
	return c, nil
}

// --- Creator courses -----------------------------------------------------------------------------

// CreatorCourses lists userID's own courses with their drafts.
func (s *CourseService) CreatorCourses(ctx context.Context, userID int64) ([]Course, error) {
	list, err := s.repo.ListCourses(ctx, CourseFilter{OwnerID: userID})
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Course{}
	}
	for i := range list {
		s.decorate(&list[i], false, false)
		list[i].Draft = nil
	}
	return list, nil
}

// ownCourse loads a course owned by userID.
func (s *CourseService) ownCourse(ctx context.Context, userID, id int64) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.OwnerID != userID || userID <= 0 {
		return nil, ErrCourseNotFound
	}
	return c, nil
}

// CreatorCourse is one of userID's courses, the draft included.
func (s *CourseService) CreatorCourse(ctx context.Context, userID, id int64) (*Course, error) {
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	s.decorate(c, false, true)
	return c, nil
}

// draftFrom validates a creator's input against c (slug, order and status stay the platform's).
func draftFrom(c *Course, in CourseInput) (*CourseDraft, error) {
	in.Slug, in.Status, in.SortOrder = c.Slug, CourseStatusDraft, c.SortOrder
	probe := *c
	if err := in.apply(&probe); err != nil {
		return nil, err
	}
	in.Title, in.Subtitle, in.Category = probe.Title, probe.Subtitle, probe.Category
	in.Price, in.OriginalPrice, in.SalePrice, in.SaleEndsAt = probe.Price, probe.OriginalPrice, probe.SalePrice, probe.SaleEndsAt
	in.IntroMD, in.TrialMD, in.FaqMD, in.Outline, in.TrialVideoURL = probe.IntroMD, probe.TrialMD, probe.FaqMD, probe.Outline, probe.TrialVideoURL
	cover := c.CoverFile
	if c.Draft != nil {
		cover = c.Draft.CoverFile
	}
	return &CourseDraft{CourseInput: in, CoverFile: cover}, nil
}

// neverListed: the live fields are not public yet, so they follow the draft.
func neverListed(c *Course) bool { return c.ApprovedAt == nil }

func applyDraftLive(c *Course, d *CourseDraft, status string) error {
	in := d.CourseInput
	in.Slug, in.Status, in.SortOrder = c.Slug, status, c.SortOrder
	if err := in.apply(c); err != nil {
		return err
	}
	c.CoverFile = d.CoverFile
	return nil
}

// CreateCreatorCourse starts a course for an approved creator (a draft until approved).
func (s *CourseService) CreateCreatorCourse(ctx context.Context, userID int64, in CourseInput) (*Course, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	existing, err := s.repo.ListCourses(ctx, CourseFilter{OwnerID: userID})
	if err != nil {
		return nil, err
	}
	if len(existing) >= creatorMaxCourses {
		return nil, ErrCreatorTooManyCourses
	}
	in.Status, in.SortOrder = CourseStatusDraft, 0
	c := &Course{}
	if err := in.apply(c); err != nil {
		return nil, err
	}
	c.OwnerID, c.ReviewStatus = userID, CourseReviewDraft
	if c.Draft, err = draftFrom(c, in); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.CreatorCourse(ctx, userID, c.ID)
}

// UpdateCreatorCourse saves a creator's edit as the draft (a pending submission is withdrawn).
func (s *CourseService) UpdateCreatorCourse(ctx context.Context, userID, id int64, in CourseInput) (*Course, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	d, err := draftFrom(c, in)
	if err != nil {
		return nil, err
	}
	c.Draft, c.ReviewStatus, c.ReviewNote = d, CourseReviewDraft, ""
	if neverListed(c) {
		if err := applyDraftLive(c, d, CourseStatusDraft); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.CreatorCourse(ctx, userID, id)
}

// SetCreatorCover replaces the draft's cover (and the live one before the first listing).
func (s *CourseService) SetCreatorCover(ctx context.Context, userID, id int64, data []byte) (*Course, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	stored, err := s.media.SaveImage(data)
	if err != nil {
		return nil, err
	}
	s.media.Remove(stored.ThumbFile)
	if c.Draft == nil {
		if c.Draft, err = draftFrom(c, courseInputOf(c)); err != nil {
			return nil, err
		}
	}
	old := c.Draft.CoverFile
	c.Draft.CoverFile, c.ReviewStatus, c.ReviewNote = stored.File, CourseReviewDraft, ""
	if neverListed(c) {
		c.CoverFile = stored.File
	}
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		s.media.Remove(stored.File)
		return nil, err
	}
	if old != "" && old != c.CoverFile {
		s.media.Remove(old)
	}
	return s.CreatorCourse(ctx, userID, id)
}

// courseInputOf is the live course as an input.
func courseInputOf(c *Course) CourseInput {
	return CourseInput{Slug: c.Slug, Title: c.Title, Subtitle: c.Subtitle, Category: c.Category, Price: c.Price, OriginalPrice: c.OriginalPrice,
		IntroMD: c.IntroMD, Outline: c.Outline, TrialMD: c.TrialMD, FaqMD: c.FaqMD, Status: c.Status, SortOrder: c.SortOrder,
		SalePrice: c.SalePrice, SaleEndsAt: c.SaleEndsAt, EduDiscount: c.EduDiscount, TrialVideoURL: c.TrialVideoURL}
}

// CreatorUploadImage stores an image for an approved creator's course Markdown.
func (s *CourseService) CreatorUploadImage(ctx context.Context, userID int64, data []byte) (string, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return "", err
	}
	return s.UploadImage(data)
}

// SubmitCreatorCourse sends the draft for review; the netdisk link must be there.
func (s *CourseService) SubmitCreatorCourse(ctx context.Context, userID, id int64) (*Course, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	switch c.ReviewStatus {
	case CourseReviewPending:
		return nil, ErrCreatorReviewPending
	case CourseReviewApproved:
		return nil, ErrCreatorNothingToSub
	}
	if c.Draft == nil {
		return nil, ErrCreatorNothingToSub
	}
	if c.DeliveryVersion == 0 {
		return nil, ErrCreatorNeedsDelivery
	}
	now := s.now()
	c.ReviewStatus, c.ReviewNote, c.SubmittedAt = CourseReviewPending, "", &now
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.CreatorCourse(ctx, userID, id)
}

// SetCreatorCourseOnSale lists or unlists an approved course (no review: the live version was approved).
func (s *CourseService) SetCreatorCourseOnSale(ctx context.Context, userID, id int64, onSale bool) (*Course, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !onSale {
		if c.Status == CourseStatusPublished {
			c.Status = CourseStatusArchived
		}
	} else {
		if neverListed(c) {
			return nil, ErrCreatorNeverApproved
		}
		c.Status = CourseStatusPublished
	}
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.CreatorCourse(ctx, userID, id)
}

// DeleteCreatorCourse deletes a course that was never listed.
func (s *CourseService) DeleteCreatorCourse(ctx context.Context, userID, id int64) error {
	c, err := s.ownCourse(ctx, userID, id)
	if err != nil {
		return err
	}
	if !neverListed(c) || c.StudentCount > 0 {
		return ErrCreatorCourseLive
	}
	if err := s.repo.DeleteCourse(ctx, id); err != nil {
		return err
	}
	s.media.Remove(c.CoverFile)
	if c.Draft != nil && c.Draft.CoverFile != c.CoverFile {
		s.media.Remove(c.Draft.CoverFile)
	}
	return nil
}

// CreatorDeliveries / SaveCreatorDelivery: the netdisk link is the creator's to change at once.
func (s *CourseService) CreatorDeliveries(ctx context.Context, userID, id int64) ([]CourseDelivery, error) {
	if _, err := s.ownCourse(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.Deliveries(ctx, id)
}

func (s *CourseService) SaveCreatorDelivery(ctx context.Context, userID, id int64, in DeliveryInput, siteURL string) (*CourseDelivery, error) {
	if _, err := s.approvedCreator(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := s.ownCourse(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.SaveDelivery(ctx, id, userID, in, siteURL)
}

// CreatorStudents lists a creator's buyers with masked emails.
func (s *CourseService) CreatorStudents(ctx context.Context, userID, id int64, page int) ([]CourseEnrollment, int, error) {
	if _, err := s.ownCourse(ctx, userID, id); err != nil {
		return nil, 0, err
	}
	list, total, err := s.Enrollments(ctx, id, page, 30)
	for i := range list {
		list[i].UserEmail, list[i].Note, list[i].RecentIPs = maskEmail(list[i].UserEmail), "", 0
	}
	return list, total, err
}

// --- Admin review --------------------------------------------------------------------------------

// ReviewCourses lists creator courses waiting for review, oldest first.
func (s *CourseService) ReviewCourses(ctx context.Context) ([]Course, error) {
	list, err := s.repo.ListCourses(ctx, CourseFilter{ReviewPending: true})
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Course{}
	}
	for i := range list {
		s.decorate(&list[i], false, true)
	}
	return list, nil
}

// ReviewCourse approves a creator's draft (it goes live and on sale) or rejects it with a reason.
func (s *CourseService) ReviewCourse(ctx context.Context, id int64, approve bool, note string) (*Course, error) {
	c, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCourseNotFound
	}
	if c.OwnerID == 0 || c.ReviewStatus != CourseReviewPending || c.Draft == nil {
		return nil, ErrCreatorNotReviewable
	}
	note = cleanText(note, 500)
	if !approve {
		if note == "" {
			return nil, ErrCreatorReasonRequired
		}
		c.ReviewStatus, c.ReviewNote = CourseReviewRejected, note
	} else {
		if _, err := s.approvedCreator(ctx, c.OwnerID); err != nil {
			return nil, err
		}
		oldCover := c.CoverFile
		if err := applyDraftLive(c, c.Draft, CourseStatusPublished); err != nil {
			return nil, err
		}
		now := s.now()
		c.ReviewStatus, c.ReviewNote, c.ApprovedAt = CourseReviewApproved, note, &now
		if err := s.repo.UpdateCourse(ctx, c); err != nil {
			return nil, err
		}
		if oldCover != "" && oldCover != c.CoverFile {
			s.media.Remove(oldCover)
		}
		return s.AdminCourse(ctx, id)
	}
	if err := s.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return s.AdminCourse(ctx, id)
}

// AdminCreators lists creators (status ” = all) with their balances.
func (s *CourseService) AdminCreators(ctx context.Context, status string) ([]CourseCreator, error) {
	list, err := s.repo.ListCreators(ctx, status)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []CourseCreator{}
	}
	for i := range list {
		if list[i].Status == CreatorStatusApproved || list[i].Status == CreatorStatusSuspended {
			if list[i].Balance, err = s.repo.CreatorBalance(ctx, list[i].UserID); err != nil {
				return nil, err
			}
		}
	}
	return list, nil
}

// CreatorUpdate is the admin's decision on a creator.
type CreatorUpdate struct {
	Status string `json:"status"`
	// CommissionPercent: the creator's own rate; nil = the site default.
	CommissionPercent *float64 `json:"commission_percent"`
	AdminNote         string   `json:"admin_note"`
}

// UpdateCreator approves, rejects or suspends a creator and sets their commission; suspending
// takes their courses off sale (buyers keep access).
func (s *CourseService) UpdateCreator(ctx context.Context, userID int64, in CreatorUpdate) (*CourseCreator, error) {
	c, err := s.repo.GetCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCreatorNotFound
	}
	switch in.Status {
	case CreatorStatusPending, CreatorStatusApproved, CreatorStatusRejected, CreatorStatusSuspended:
	default:
		return nil, errCourseInvalid("状态")
	}
	if in.CommissionPercent != nil {
		v := *in.CommissionPercent
		if math.IsNaN(v) || v < 0 || v > 100 {
			return nil, errCourseInvalid("手续费比例须在 0–100% 之间")
		}
		v = roundCents(v)
		in.CommissionPercent = &v
	}
	note := cleanText(in.AdminNote, 500)
	if in.Status == CreatorStatusRejected && note == "" {
		return nil, ErrCreatorReasonRequired
	}
	if err := s.repo.UpdateCreator(ctx, userID, in.Status, in.CommissionPercent, note); err != nil {
		return nil, err
	}
	if in.Status == CreatorStatusSuspended || in.Status == CreatorStatusRejected {
		if err := s.repo.ArchiveCreatorCourses(ctx, userID); err != nil {
			return nil, err
		}
	}
	return s.repo.GetCreator(ctx, userID)
}

// --- Sales and withdrawals -------------------------------------------------------------------------

// RecordCreatorSale credits a creator's share of a completed course order (payment hook).
func (s *CourseService) RecordCreatorSale(ctx context.Context, orderID int64) error {
	st := s.Settings(ctx)
	return s.repo.RecordCreatorSale(ctx, orderID, st.CreatorCommissionPercent, st.CreatorSettleDays)
}

// CreatorSales lists a creator's orders, newest first.
func (s *CourseService) CreatorSales(ctx context.Context, userID int64, page int) ([]CreatorSale, int, error) {
	if page < 1 {
		page = 1
	}
	list, total, err := s.repo.CreatorSales(ctx, userID, 30, (page-1)*30)
	if list == nil {
		list = []CreatorSale{}
	}
	for i := range list {
		list[i].Buyer = maskEmail(list[i].Buyer)
	}
	return list, total, err
}

// CreatorWithdrawRequest is the creator's 申请提现 form.
type CreatorWithdrawRequest = WithdrawRequest

// RequestCreatorWithdrawal takes cny out of the creator's available balance for a manual payout.
func (s *CourseService) RequestCreatorWithdrawal(ctx context.Context, userID int64, in CreatorWithdrawRequest) (*CreatorWithdrawal, error) {
	c, err := s.repo.GetCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	if c == nil || (c.Status != CreatorStatusApproved && c.Status != CreatorStatusSuspended) {
		return nil, ErrCreatorNotApproved
	}
	in, err = normalizeWithdrawRequest(in)
	if err != nil {
		return nil, err
	}
	minCNY := s.Settings(ctx).CreatorWithdrawMinCNY
	if in.CNYAmount < minCNY {
		return nil, infraerrors.BadRequest("WITHDRAW_BELOW_MIN", fmt.Sprintf("最低提现 ¥%.2f（Minimum withdrawal is %.2f CNY）", minCNY, minCNY))
	}
	w := &CreatorWithdrawal{UserID: userID, CNYAmount: in.CNYAmount, Method: in.Method, UserNote: in.Note}
	if w.Account, err = s.encryptor.Encrypt(in.Account); err != nil {
		return nil, fmt.Errorf("encrypt creator payee: %w", err)
	}
	if w.RealName, err = s.encryptor.Encrypt(in.RealName); err != nil {
		return nil, fmt.Errorf("encrypt creator payee: %w", err)
	}
	err = s.repo.CreateCreatorWithdrawal(ctx, w, func(b *CreatorBalance) error {
		if b.Pending > 0 {
			return ErrWithdrawPending
		}
		if in.CNYAmount > floorCents(b.Available)+1e-9 {
			return ErrWithdrawOverAvailable
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.decryptPayee(w)
	return w, nil
}

// CancelCreatorWithdrawal lets the creator take back a pending withdrawal.
func (s *CourseService) CancelCreatorWithdrawal(ctx context.Context, userID, id int64) (*CreatorWithdrawal, error) {
	w, err := s.repo.ResolveCreatorWithdrawal(ctx, id, WithdrawStatusCancelled, "", 0, userID)
	if err != nil {
		return nil, err
	}
	s.decryptPayee(w)
	return w, nil
}

// AdminCreatorWithdrawals lists withdrawals for review with payee details.
func (s *CourseService) AdminCreatorWithdrawals(ctx context.Context, f CreatorWithdrawFilter) ([]CreatorWithdrawal, int, error) {
	switch f.Status {
	case "", "all":
		f.Status = ""
	case WithdrawStatusPending, WithdrawStatusPaid, WithdrawStatusRejected, WithdrawStatusCancelled:
	default:
		return nil, 0, infraerrors.BadRequest("INVALID_STATUS", "invalid status")
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	f.UserID = 0
	list, total, err := s.repo.ListCreatorWithdrawals(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		s.decryptPayee(&list[i])
	}
	return list, total, nil
}

// ResolveCreatorWithdrawal marks a withdrawal paid, or rejects it (the reason is shown to the creator).
func (s *CourseService) ResolveCreatorWithdrawal(ctx context.Context, id, adminID int64, paid bool, note string) (*CreatorWithdrawal, error) {
	note = truncateRunes(strings.TrimSpace(note), 500)
	status := WithdrawStatusPaid
	if !paid {
		if note == "" {
			return nil, ErrCreatorReasonRequired
		}
		status = WithdrawStatusRejected
	}
	w, err := s.repo.ResolveCreatorWithdrawal(ctx, id, status, note, adminID, 0)
	if err != nil {
		return nil, err
	}
	s.decryptPayee(w)
	return w, nil
}

func (s *CourseService) decryptPayee(w *CreatorWithdrawal) {
	if w == nil {
		return
	}
	open := func(v string) string {
		if out, err := s.encryptor.Decrypt(v); err == nil {
			return out
		}
		return "（无法解密）"
	}
	w.Account, w.RealName = open(w.Account), open(w.RealName)
}
