package service

import (
	"context"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Adding an inviter after signing up: a new user who skipped (or mistyped) a friend's invite code
// may add it within AffiliateLateBindWindow of registering; admins may set or change it any time.
// Rebates still run from the invitee's registration (affiliate_rebate_duration_days), so a late
// inviter gets no longer window.

// AffiliateLateBindWindow is how long after registering a user may add an inviter.
const AffiliateLateBindWindow = 7 * 24 * time.Hour

var (
	ErrAffiliateDisabled        = infraerrors.BadRequest("AFFILIATE_DISABLED", "邀请返利暂未开放（Invites are turned off）")
	ErrAffiliateLateBindExpired = infraerrors.BadRequest("AFFILIATE_LATE_BIND_EXPIRED", "注册超过 7 天后不能再填写邀请人（An inviter can only be added within 7 days of signing up）")
	ErrAffiliateSelfCode        = infraerrors.BadRequest("AFFILIATE_SELF_CODE", "不能填写自己的邀请码（You can't use your own invite code）")
	ErrAffiliateCodeCycle       = infraerrors.BadRequest("AFFILIATE_CODE_CYCLE", "不能填写你邀请的好友的邀请码（You can't use the code of someone you invited）")
)

// AffiliateLateBindDeadline is the last moment a user registered at registeredAt may add an inviter.
func AffiliateLateBindDeadline(registeredAt time.Time) time.Time {
	return registeredAt.Add(AffiliateLateBindWindow)
}

func normalizeAffiliateCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

// inviterByCode resolves a code to its owner (ErrAffiliateCodeInvalid when unknown or malformed).
func (s *AffiliateService) inviterByCode(ctx context.Context, raw string) (*AffiliateSummary, error) {
	code := normalizeAffiliateCode(raw)
	if !isValidAffiliateCodeFormat(code) {
		return nil, ErrAffiliateCodeInvalid
	}
	inviter, err := s.repo.GetAffiliateByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrAffiliateProfileNotFound) {
			return nil, ErrAffiliateCodeInvalid
		}
		return nil, err
	}
	if inviter == nil || inviter.UserID <= 0 {
		return nil, ErrAffiliateCodeInvalid
	}
	return inviter, nil
}

// CheckCode tells whether raw is a usable invite code (the sign-up form checks it as you type).
func (s *AffiliateService) CheckCode(ctx context.Context, raw string) error {
	if s == nil || s.repo == nil || !s.IsEnabled(ctx) {
		return ErrAffiliateDisabled
	}
	_, err := s.inviterByCode(ctx, raw)
	return err
}

// FillLateBind tells the user's affiliate page whether they may still add an inviter.
func (s *AffiliateService) FillLateBind(ctx context.Context, detail *AffiliateDetail, registeredAt time.Time) {
	if detail == nil || detail.InviterID != nil || registeredAt.IsZero() || !s.IsEnabled(ctx) {
		return
	}
	deadline := AffiliateLateBindDeadline(registeredAt)
	if time.Now().Before(deadline) {
		detail.CanBindInviter, detail.BindInviterDeadline = true, &deadline
	}
}

// BindInviterLate adds the inviter of a user who registered at registeredAt, within the window.
func (s *AffiliateService) BindInviterLate(ctx context.Context, userID int64, registeredAt time.Time, raw string) error {
	if s == nil || s.repo == nil || !s.IsEnabled(ctx) {
		return ErrAffiliateDisabled
	}
	if registeredAt.IsZero() || time.Now().After(AffiliateLateBindDeadline(registeredAt)) {
		return ErrAffiliateLateBindExpired
	}
	self, err := s.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return err
	}
	if self.InviterID != nil {
		return ErrAffiliateAlreadyBound
	}
	inviter, err := s.checkedInviter(ctx, userID, raw)
	if err != nil {
		return err
	}
	bound, err := s.repo.BindInviter(ctx, userID, inviter.UserID)
	if err != nil {
		return err
	}
	if !bound {
		return ErrAffiliateAlreadyBound
	}
	s.afterInviterBound(ctx, userID, inviter.UserID)
	return nil
}

// checkedInviter resolves the code for userID: not their own, not someone they invited.
func (s *AffiliateService) checkedInviter(ctx context.Context, userID int64, raw string) (*AffiliateSummary, error) {
	inviter, err := s.inviterByCode(ctx, raw)
	if err != nil {
		return nil, err
	}
	if inviter.UserID == userID {
		return nil, ErrAffiliateSelfCode
	}
	if inviter.InviterID != nil && *inviter.InviterID == userID {
		return nil, ErrAffiliateCodeCycle
	}
	return inviter, nil
}

// AdminSetInviter sets the user's inviter to the owner of raw, or clears it when raw is empty.
func (s *AffiliateService) AdminSetInviter(ctx context.Context, userID int64, raw string) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if userID <= 0 {
		return ErrUserNotFound
	}
	if strings.TrimSpace(raw) == "" {
		return s.repo.SetInviter(ctx, userID, nil)
	}
	inviter, err := s.checkedInviter(ctx, userID, raw)
	if err != nil {
		return err
	}
	return s.repo.SetInviter(ctx, userID, &inviter.UserID)
}
