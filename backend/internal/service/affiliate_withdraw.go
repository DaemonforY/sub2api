package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 邀请返利提现（人工打款）。
//
// 用户申请时立即从可用返利（aff_quota）里扣出申请额度，管理员线下打款后标记「已打款」；
// 驳回或用户撤销时把额度退回。只有被邀请人真实付费订单（非余额支付、已完成、已过冻结期）
// 产生的返利能提现，并按这些订单的实付金额（不含手续费）折算成人民币，
// 所以后台加余额、兑换码带来的返利只能转入余额，不能变成现金。

const (
	WithdrawStatusPending   = "pending"
	WithdrawStatusPaid      = "paid"
	WithdrawStatusRejected  = "rejected"
	WithdrawStatusCancelled = "cancelled"

	WithdrawMethodAlipay = "alipay"
	WithdrawMethodWechat = "wechat"

	withdrawMinCNYDefault       = 50.0
	withdrawMinCNYFloor         = 1.0
	withdrawMonthlyLimitDefault = 2
	withdrawMonthlyLimitMax     = 31
	withdrawUserHistoryLimit    = 20
)

var (
	ErrWithdrawDisabled      = infraerrors.Forbidden("WITHDRAW_DISABLED", "提现功能暂未开放，返利可以先转入余额使用（Withdrawals are not available）")
	ErrWithdrawPending       = infraerrors.Conflict("WITHDRAW_PENDING", "你还有一笔提现在处理中，处理完成后才能再申请（A withdrawal is already pending）")
	ErrWithdrawMonthlyLimit  = infraerrors.TooManyRequests("WITHDRAW_MONTHLY_LIMIT", "本月提现次数已用完，下个月再来（Monthly withdrawal limit reached）")
	ErrWithdrawBelowMin      = infraerrors.BadRequest("WITHDRAW_BELOW_MIN", "提现金额低于最低提现金额（Amount is below the minimum）")
	ErrWithdrawOverAvailable = infraerrors.BadRequest("WITHDRAW_OVER_AVAILABLE", "提现金额超过可提现金额，请刷新后重试（Amount exceeds the withdrawable amount）")
	ErrWithdrawInvalidMethod = infraerrors.BadRequest("WITHDRAW_INVALID_METHOD", "请选择支付宝或微信收款（Invalid payout method）")
	ErrWithdrawInvalidPayee  = infraerrors.BadRequest("WITHDRAW_INVALID_PAYEE", "请填写收款账号和真实姓名（Payout account and real name are required）")
	ErrWithdrawNotFound      = infraerrors.NotFound("WITHDRAW_NOT_FOUND", "提现申请不存在（Withdrawal not found）")
	ErrWithdrawNotPending    = infraerrors.Conflict("WITHDRAW_NOT_PENDING", "这笔提现已经处理过了，请刷新页面（Withdrawal is no longer pending）")
)

// WithdrawSettings are the admin-editable withdrawal rules (stored with the growth settings).
type WithdrawSettings struct {
	Enabled      bool    `json:"withdraw_enabled"`
	MinCNY       float64 `json:"withdraw_min_cny"`
	MonthlyLimit int     `json:"withdraw_monthly_limit"`
}

// WithdrawEligibility is what the repository measures for one user, all amounts in rebate (aff_quota) units
// unless named CNY.
type WithdrawEligibility struct {
	AvailableQuota float64 // aff_quota after thawing matured rebates
	CashQuota      float64 // matured rebates from completed, non-balance-paid orders
	CashCNY        float64 // what those orders paid for them, fee excluded
	WithdrawnQuota float64 // pending + paid withdrawals
	WithdrawnCNY   float64
	MonthCount     int // pending + paid withdrawals since the start of this month
	HasPending     bool
}

// AffiliateWithdrawal is one withdrawal request. Account / RealName are plaintext only after the service decrypts them.
type AffiliateWithdrawal struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	UserEmail   string     `json:"user_email,omitempty"`
	Username    string     `json:"username,omitempty"`
	QuotaAmount float64    `json:"quota_amount"`
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

// NewWithdrawal is what the repository stores; Account / RealName are already encrypted.
type NewWithdrawal struct {
	CNYAmount float64
	Method    string
	Account   string
	RealName  string
	UserNote  string
}

// WithdrawFilter filters the admin list.
type WithdrawFilter struct {
	Status   string
	Search   string
	Page     int
	PageSize int
}

// WithdrawPlanFunc runs inside the repository's transaction with the row lock held and returns the
// rebate quota to deduct, or an error to abort.
type WithdrawPlanFunc func(e *WithdrawEligibility) (quota float64, err error)

type AffiliateWithdrawRepository interface {
	Eligibility(ctx context.Context, userID int64, monthStart time.Time) (*WithdrawEligibility, error)
	Create(ctx context.Context, userID int64, in NewWithdrawal, monthStart time.Time, plan WithdrawPlanFunc) (*AffiliateWithdrawal, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]AffiliateWithdrawal, error)
	List(ctx context.Context, filter WithdrawFilter) ([]AffiliateWithdrawal, int64, error)
	CountPending(ctx context.Context) (int64, error)
	// MarkPaid moves a pending withdrawal to paid.
	MarkPaid(ctx context.Context, id int64, adminNote string, reviewerID int64) (*AffiliateWithdrawal, error)
	// Return moves a pending withdrawal to rejected / cancelled and gives the quota back.
	// When ownerID > 0 only that user's withdrawal matches.
	Return(ctx context.Context, id int64, status, adminNote string, reviewerID, ownerID int64) (*AffiliateWithdrawal, error)
}

// AffiliateWithdrawService runs 返利提现.
type AffiliateWithdrawService struct {
	repo      AffiliateWithdrawRepository
	growth    *GrowthService
	encryptor SecretEncryptor
	affiliate *AffiliateService
	now       func() time.Time
}

func NewAffiliateWithdrawService(repo AffiliateWithdrawRepository, growth *GrowthService, encryptor SecretEncryptor, affiliate *AffiliateService) *AffiliateWithdrawService {
	return &AffiliateWithdrawService{repo: repo, growth: growth, encryptor: encryptor, affiliate: affiliate, now: time.Now}
}

// WithdrawStatus is what the user's 邀请返利 page shows.
type WithdrawStatus struct {
	Enabled         bool                  `json:"enabled"`
	MinCNY          float64               `json:"min_cny"`
	MonthlyLimit    int                   `json:"monthly_limit"`
	MonthUsed       int                   `json:"month_used"`
	WithdrawableCNY float64               `json:"withdrawable_cny"`
	AvailableQuota  float64               `json:"available_quota"`
	CashCNY         float64               `json:"cash_cny"`      // 付费订单带来的可提现返利（累计，人民币）
	WithdrawnCNY    float64               `json:"withdrawn_cny"` // 已提现 + 处理中
	HasPending      bool                  `json:"has_pending"`
	LastMethod      string                `json:"last_method,omitempty"`
	LastAccount     string                `json:"last_account,omitempty"`
	LastRealName    string                `json:"last_real_name,omitempty"`
	Withdrawals     []AffiliateWithdrawal `json:"withdrawals"`
	FreezeHours     int                   `json:"freeze_hours"`
}

func (s *AffiliateWithdrawService) settings(ctx context.Context) WithdrawSettings {
	if s.growth == nil {
		return WithdrawSettings{MinCNY: withdrawMinCNYDefault, MonthlyLimit: withdrawMonthlyLimitDefault}
	}
	gs, err := s.growth.GetSettings(ctx)
	if err != nil || gs == nil {
		return WithdrawSettings{MinCNY: withdrawMinCNYDefault, MonthlyLimit: withdrawMonthlyLimitDefault}
	}
	return gs.WithdrawSettings
}

// enabled: the switch and the affiliate program itself must both be on.
func (s *AffiliateWithdrawService) enabled(ctx context.Context, st WithdrawSettings) bool {
	if !st.Enabled {
		return false
	}
	return s.affiliate == nil || s.affiliate.IsEnabled(ctx)
}

// monthStart is the first instant of the current month in China time, matching how users count months.
func (s *AffiliateWithdrawService) monthStart() time.Time {
	loc := time.FixedZone("CST", 8*3600)
	now := s.now().In(loc)
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
}

// withdrawableCNY: the cash-backed rebate not yet withdrawn, never more than the rebate still available.
func withdrawableCNY(e *WithdrawEligibility) float64 {
	if e == nil || e.CashQuota <= 0 || e.CashCNY <= 0 {
		return 0
	}
	ratio := e.CashCNY / e.CashQuota
	remaining := e.CashCNY - e.WithdrawnCNY
	if byAvailable := e.AvailableQuota * ratio; byAvailable < remaining {
		remaining = byAvailable
	}
	return floorCents(remaining)
}

// quotaForCNY is the rebate quota a withdrawal of cny takes, at the user's average cash ratio.
func quotaForCNY(e *WithdrawEligibility, cny float64) float64 {
	if e == nil || e.CashCNY <= 0 {
		return 0
	}
	q := roundTo(cny*e.CashQuota/e.CashCNY, 8)
	if q > e.AvailableQuota {
		q = e.AvailableQuota
	}
	return q
}

func floorCents(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Floor(v*100+1e-6) / 100
}

// Status returns the withdrawal box of the user's 邀请返利 page.
func (s *AffiliateWithdrawService) Status(ctx context.Context, userID int64) (*WithdrawStatus, error) {
	st := s.settings(ctx)
	out := &WithdrawStatus{
		Enabled:      s.enabled(ctx, st),
		MinCNY:       st.MinCNY,
		MonthlyLimit: st.MonthlyLimit,
		Withdrawals:  []AffiliateWithdrawal{},
	}
	if s.affiliate != nil && s.affiliate.settingService != nil {
		out.FreezeHours = s.affiliate.settingService.GetAffiliateRebateFreezeHours(ctx)
	}
	e, err := s.repo.Eligibility(ctx, userID, s.monthStart())
	if err != nil {
		return nil, err
	}
	out.WithdrawableCNY = withdrawableCNY(e)
	out.AvailableQuota = roundTo(e.AvailableQuota, 8)
	out.CashCNY = floorCents(e.CashCNY)
	out.WithdrawnCNY = roundTo(e.WithdrawnCNY, 2)
	out.MonthUsed = e.MonthCount
	out.HasPending = e.HasPending

	list, err := s.repo.ListByUser(ctx, userID, withdrawUserHistoryLimit)
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.decrypt(&list[i])
	}
	if len(list) > 0 {
		out.LastMethod, out.LastAccount, out.LastRealName = list[0].Method, list[0].Account, list[0].RealName
	}
	out.Withdrawals = list
	return out, nil
}

// WithdrawRequest is the user's 申请提现 form.
type WithdrawRequest struct {
	CNYAmount float64 `json:"cny_amount"`
	Method    string  `json:"method"`
	Account   string  `json:"account"`
	RealName  string  `json:"real_name"`
	Note      string  `json:"note"`
}

func normalizeWithdrawRequest(in WithdrawRequest) (WithdrawRequest, error) {
	in.Method = strings.ToLower(strings.TrimSpace(in.Method))
	if in.Method != WithdrawMethodAlipay && in.Method != WithdrawMethodWechat {
		return in, ErrWithdrawInvalidMethod
	}
	in.Account = strings.TrimSpace(in.Account)
	in.RealName = strings.TrimSpace(in.RealName)
	in.Note = strings.TrimSpace(in.Note)
	if in.Account == "" || in.RealName == "" || utf8.RuneCountInString(in.Account) > 100 || utf8.RuneCountInString(in.RealName) > 50 {
		return in, ErrWithdrawInvalidPayee
	}
	in.Note = truncateRunes(in.Note, 200)
	if math.IsNaN(in.CNYAmount) || math.IsInf(in.CNYAmount, 0) || in.CNYAmount <= 0 {
		return in, ErrWithdrawBelowMin
	}
	in.CNYAmount = math.Round(in.CNYAmount*100) / 100
	return in, nil
}

// Request files a withdrawal and takes its rebate quota out at once.
func (s *AffiliateWithdrawService) Request(ctx context.Context, userID int64, in WithdrawRequest) (*AffiliateWithdrawal, error) {
	st := s.settings(ctx)
	if !s.enabled(ctx, st) {
		return nil, ErrWithdrawDisabled
	}
	in, err := normalizeWithdrawRequest(in)
	if err != nil {
		return nil, err
	}
	if in.CNYAmount < st.MinCNY {
		return nil, infraerrors.BadRequest("WITHDRAW_BELOW_MIN",
			fmt.Sprintf("最低提现 ¥%.2f（Minimum withdrawal is %.2f CNY）", st.MinCNY, st.MinCNY))
	}
	if s.encryptor == nil {
		return nil, infraerrors.ServiceUnavailable("WITHDRAW_UNAVAILABLE", "提现暂不可用，请联系客服（Withdrawals are unavailable）")
	}
	account, err := s.encryptor.Encrypt(in.Account)
	if err != nil {
		return nil, fmt.Errorf("encrypt withdraw account: %w", err)
	}
	realName, err := s.encryptor.Encrypt(in.RealName)
	if err != nil {
		return nil, fmt.Errorf("encrypt withdraw real name: %w", err)
	}

	plan := func(e *WithdrawEligibility) (float64, error) {
		if e.HasPending {
			return 0, ErrWithdrawPending
		}
		if st.MonthlyLimit > 0 && e.MonthCount >= st.MonthlyLimit {
			return 0, ErrWithdrawMonthlyLimit
		}
		if in.CNYAmount > withdrawableCNY(e)+1e-9 {
			return 0, ErrWithdrawOverAvailable
		}
		quota := quotaForCNY(e, in.CNYAmount)
		if quota <= 0 {
			return 0, ErrWithdrawOverAvailable
		}
		return quota, nil
	}
	w, err := s.repo.Create(ctx, userID, NewWithdrawal{
		CNYAmount: in.CNYAmount, Method: in.Method, Account: account, RealName: realName, UserNote: in.Note,
	}, s.monthStart(), plan)
	if err != nil {
		return nil, err
	}
	s.decrypt(w)
	s.invalidate(ctx, userID)
	return w, nil
}

// Cancel lets the user take back a pending withdrawal.
func (s *AffiliateWithdrawService) Cancel(ctx context.Context, userID, id int64) (*AffiliateWithdrawal, error) {
	w, err := s.repo.Return(ctx, id, WithdrawStatusCancelled, "", 0, userID)
	if err != nil {
		return nil, err
	}
	s.decrypt(w)
	s.invalidate(ctx, userID)
	return w, nil
}

// AdminList lists withdrawals for 提现审核 with decrypted payee details.
func (s *AffiliateWithdrawService) AdminList(ctx context.Context, f WithdrawFilter) ([]AffiliateWithdrawal, int64, error) {
	switch f.Status {
	case "", "all":
		f.Status = ""
	case WithdrawStatusPending, WithdrawStatusPaid, WithdrawStatusRejected, WithdrawStatusCancelled:
	default:
		return nil, 0, infraerrors.BadRequest("INVALID_STATUS", "invalid status")
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	f.Search = strings.TrimSpace(f.Search)
	list, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		s.decrypt(&list[i])
	}
	return list, total, nil
}

func (s *AffiliateWithdrawService) PendingCount(ctx context.Context) (int64, error) {
	return s.repo.CountPending(ctx)
}

// MarkPaid records that the admin has transferred the money.
func (s *AffiliateWithdrawService) MarkPaid(ctx context.Context, id, adminID int64, note string) (*AffiliateWithdrawal, error) {
	w, err := s.repo.MarkPaid(ctx, id, truncateRunes(strings.TrimSpace(note), 500), adminID)
	if err != nil {
		return nil, err
	}
	s.decrypt(w)
	return w, nil
}

// Reject turns a withdrawal down and gives the rebate back; the reason is shown to the user.
func (s *AffiliateWithdrawService) Reject(ctx context.Context, id, adminID int64, reason string) (*AffiliateWithdrawal, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, infraerrors.BadRequest("WITHDRAW_REASON_REQUIRED", "请填写驳回原因，用户会看到（A reason is required）")
	}
	w, err := s.repo.Return(ctx, id, WithdrawStatusRejected, truncateRunes(reason, 500), adminID, 0)
	if err != nil {
		return nil, err
	}
	s.decrypt(w)
	s.invalidate(ctx, w.UserID)
	return w, nil
}

func (s *AffiliateWithdrawService) decrypt(w *AffiliateWithdrawal) {
	if w == nil || s.encryptor == nil {
		return
	}
	if v, err := s.encryptor.Decrypt(w.Account); err == nil {
		w.Account = v
	} else {
		w.Account = "（无法解密）"
	}
	if v, err := s.encryptor.Decrypt(w.RealName); err == nil {
		w.RealName = v
	} else {
		w.RealName = "（无法解密）"
	}
}

func (s *AffiliateWithdrawService) invalidate(ctx context.Context, userID int64) {
	if s.affiliate != nil {
		s.affiliate.invalidateAffiliateCaches(ctx, userID)
	}
}

// normalizeWithdrawSettings validates admin input.
func normalizeWithdrawSettings(in WithdrawSettings) (WithdrawSettings, error) {
	if math.IsNaN(in.MinCNY) || math.IsInf(in.MinCNY, 0) || in.MinCNY < withdrawMinCNYFloor || in.MinCNY > 100000 {
		return in, infraerrors.BadRequest("INVALID_WITHDRAW_MIN", "最低提现金额需在 1 到 100000 元之间（Minimum withdrawal must be between 1 and 100000 CNY）")
	}
	in.MinCNY = math.Round(in.MinCNY*100) / 100
	if in.MonthlyLimit < 0 || in.MonthlyLimit > withdrawMonthlyLimitMax {
		return in, infraerrors.BadRequest("INVALID_WITHDRAW_LIMIT", "每月提现次数需在 0 到 31 之间，0 表示不限（Monthly limit must be between 0 and 31）")
	}
	return in, nil
}
