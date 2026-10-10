package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// What the user types to confirm deleting their own account (the English page asks for DELETE).
const (
	AccountDeletionConfirm   = "注销账号"
	AccountDeletionConfirmEN = "DELETE"
)

var (
	ErrAccountDeletionConfirm = infraerrors.BadRequest("ACCOUNT_DELETION_CONFIRM", "请输入「"+AccountDeletionConfirm+"」确认注销")
	ErrAccountDeletionAdmin   = infraerrors.Forbidden("ACCOUNT_DELETION_ADMIN", "管理员账号不能自助注销")
	ErrAccountDeletionPending = infraerrors.Conflict("ACCOUNT_DELETION_PENDING", "你还有一笔返利提现在处理中，请等它处理完或先撤销，再注销账号")
)

// AccountDeletionRepository holds the raw SQL around self-service account deletion.
type AccountDeletionRepository interface {
	HasPendingWithdrawal(ctx context.Context, userID int64) (bool, error)
	// PurgeOwnedContent deletes what the user built that would otherwise outlive the
	// account: AI tutors (with their students' messages) and article projects.
	PurgeOwnedContent(ctx context.Context, userID int64) error
	// Anonymize clears the personal fields of a (soft-)deleted user row; the row
	// itself stays so orders and usage records keep their owner ID.
	Anonymize(ctx context.Context, userID int64) error
}

// AccountDeletionService lets a user delete their own account (个人信息保护法 第四十七条).
type AccountDeletionService struct {
	users UserRepository
	admin AdminService
	repo  AccountDeletionRepository
}

func NewAccountDeletionService(users UserRepository, admin AdminService, repo AccountDeletionRepository) *AccountDeletionService {
	return &AccountDeletionService{users: users, admin: admin, repo: repo}
}

// DeleteSelf deletes the caller's account: API keys stop working at once, owned
// content is removed, and the personal fields are cleared. The remaining balance
// is forfeited; the page says so before the user confirms.
func (s *AccountDeletionService) DeleteSelf(ctx context.Context, userID int64, confirm string) error {
	if c := strings.TrimSpace(confirm); c != AccountDeletionConfirm && c != AccountDeletionConfirmEN {
		return ErrAccountDeletionConfirm
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Role == RoleAdmin {
		return ErrAccountDeletionAdmin
	}
	pending, err := s.repo.HasPendingWithdrawal(ctx, userID)
	if err != nil {
		return err
	}
	if pending {
		return ErrAccountDeletionPending
	}
	if err := s.repo.PurgeOwnedContent(ctx, userID); err != nil {
		return err
	}
	if err := s.admin.DeleteUser(ctx, userID); err != nil {
		return err
	}
	// The account is already gone; a failed cleanup must not tell the user it wasn't.
	if err := s.repo.Anonymize(ctx, userID); err != nil {
		logger.LegacyPrintf("service.account_deletion", "anonymize deleted user failed: user_id=%d err=%v", userID, err)
	}
	return nil
}
