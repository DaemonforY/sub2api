//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestCourseCreators(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	committed := trackCommitted(t)
	var courseIDs []int64
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(context.Background(), `DELETE FROM courses WHERE id = ANY($1)`, pq.Array(courseIDs))
		require.NoError(t, err)
	})

	settings := NewSettingRepository(integrationEntClient)
	keys := []string{"course_creator_enabled", "course_creator_commission_percent", "course_creator_settle_days", "course_creator_withdraw_min_cny", "course_affiliate_rate_percent"}
	previous, err := settings.GetMultiple(ctx, keys)
	require.NoError(t, err)
	t.Cleanup(func() {
		restore := map[string]string{}
		for _, k := range keys {
			restore[k] = previous[k]
		}
		_ = settings.SetMultiple(context.Background(), restore)
	})

	svc := service.NewCourseService(NewCourseRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), reverseEncryptor{})
	svc.SetSettings(settings)
	payments := service.NewPaymentService(integrationEntClient, nil, nil, nil, nil, nil, nil, nil, nil)
	payments.SetCourseService(svc)
	reason := func(err error) string { return infraerrors.Reason(err) }

	creator := committed.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "cr-" + sfx + "@test.local", Username: "cr" + sfx[len(sfx)-6:]}))
	buyer := committed.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "crb-" + sfx + "@test.local", Username: "crb" + sfx[len(sfx)-6:]}))
	apply := service.CreatorApplication{DisplayName: "王老师", Contact: "wx: wang", Plan: "AI 办公课，10 节"}

	// Applications are closed until the admin opens them.
	_, err = svc.SaveSettings(ctx, service.CourseSettings{CreatorEnabled: false, CreatorCommissionPercent: 20, CreatorSettleDays: 0, CreatorWithdrawMinCNY: 10})
	require.NoError(t, err)
	_, err = svc.ApplyCreator(ctx, creator.ID, apply)
	require.Equal(t, "CREATOR_CLOSED", reason(err))
	_, err = svc.SaveSettings(ctx, service.CourseSettings{CreatorEnabled: true, CreatorCommissionPercent: 20, CreatorSettleDays: 0, CreatorWithdrawMinCNY: 10})
	require.NoError(t, err)
	app, err := svc.ApplyCreator(ctx, creator.ID, apply)
	require.NoError(t, err)
	require.Equal(t, service.CreatorStatusPending, app.Status)
	_, err = svc.ApplyCreator(ctx, creator.ID, apply)
	require.Equal(t, "CREATOR_ALREADY_APPLIED", reason(err))
	_, err = svc.CreateCreatorCourse(ctx, creator.ID, service.CourseInput{Title: "x", Slug: "x-" + sfx[len(sfx)-6:], Price: 1})
	require.Equal(t, "CREATOR_NOT_APPROVED", reason(err))

	// Approved with their own 10% commission.
	ten := 10.0
	_, err = svc.UpdateCreator(ctx, creator.ID, service.CreatorUpdate{Status: service.CreatorStatusApproved, CommissionPercent: &ten})
	require.NoError(t, err)

	slug := "cr-" + sfx[len(sfx)-8:]
	course, err := svc.CreateCreatorCourse(ctx, creator.ID, service.CourseInput{Slug: slug, Title: "AI 办公", Price: 103, IntroMD: "## 介绍", Status: service.CourseStatusPublished, SortOrder: 99})
	require.NoError(t, err)
	courseIDs = append(courseIDs, course.ID)
	require.Equal(t, service.CourseStatusDraft, course.Status, "creators can't publish by themselves")
	require.Equal(t, 0, course.SortOrder)
	require.Equal(t, creator.ID, course.OwnerID)
	require.Equal(t, service.CourseReviewDraft, course.ReviewStatus)

	// Another user can't touch it.
	_, err = svc.CreatorCourse(ctx, buyer.ID, course.ID)
	require.Equal(t, "COURSE_NOT_FOUND", reason(err))

	// The netdisk link comes before review.
	_, err = svc.SubmitCreatorCourse(ctx, creator.ID, course.ID)
	require.Equal(t, "CREATOR_NEEDS_DELIVERY", reason(err))
	_, err = svc.SaveCreatorDelivery(ctx, creator.ID, course.ID, service.DeliveryInput{Link: "https://pan.baidu.com/s/1x", Code: "ab12"}, "https://test")
	require.NoError(t, err)
	course, err = svc.SubmitCreatorCourse(ctx, creator.ID, course.ID)
	require.NoError(t, err)
	require.Equal(t, service.CourseReviewPending, course.ReviewStatus)
	_, err = svc.SetCreatorCourseOnSale(ctx, creator.ID, course.ID, true)
	require.Equal(t, "CREATOR_NEVER_APPROVED", reason(err))
	pending, err := svc.ReviewCourses(ctx)
	require.NoError(t, err)
	require.Contains(t, courseIDsOf(pending), course.ID)

	// Approved: on sale with the creator's name.
	_, err = svc.ReviewCourse(ctx, course.ID, true, "")
	require.NoError(t, err)
	page, err := svc.Course(ctx, slug, 0)
	require.NoError(t, err)
	require.Equal(t, "王老师", page.CreatorName)
	require.Nil(t, page.Draft)
	require.Zero(t, page.OwnerID)

	// A later price change waits for review; the live price stays.
	_, err = svc.UpdateCreatorCourse(ctx, creator.ID, course.ID, service.CourseInput{Title: "AI 办公（新版）", Price: 9, IntroMD: "## 新介绍"})
	require.NoError(t, err)
	page, err = svc.Course(ctx, slug, 0)
	require.NoError(t, err)
	require.Equal(t, 103.0, page.Price)
	require.Equal(t, "AI 办公", page.Title)
	_, err = svc.SubmitCreatorCourse(ctx, creator.ID, course.ID)
	require.NoError(t, err)
	_, err = svc.ReviewCourse(ctx, course.ID, false, "")
	require.Equal(t, "CREATOR_REASON_REQUIRED", reason(err))
	_, err = svc.ReviewCourse(ctx, course.ID, false, "价格太低，和介绍不符")
	require.NoError(t, err)
	mine, err := svc.CreatorCourse(ctx, creator.ID, course.ID)
	require.NoError(t, err)
	require.Equal(t, service.CourseReviewRejected, mine.ReviewStatus)
	require.Equal(t, "价格太低，和介绍不符", mine.ReviewNote)
	require.Equal(t, 9.0, mine.Draft.Price)
	require.Equal(t, 103.0, mine.Price)

	// A sale: ¥103 paid incl. a 3% payment fee → ¥100 gross, 10% commission → ¥90 for the creator.
	order, err := integrationEntClient.PaymentOrder.Create().
		SetUserID(buyer.ID).SetUserEmail(buyer.Email).SetUserName(buyer.Username).
		SetAmount(100).SetPayAmount(103).SetFeeRate(3).SetRechargeCode("").
		SetOutTradeNo("cr" + sfx).SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeCourse).SetCourseID(course.ID).SetStatus(service.OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, payments.ExecuteCourseFulfillment(ctx, order.ID))
	require.NoError(t, svc.RecordCreatorSale(ctx, order.ID)) // idempotent
	home, err := svc.CreatorHome(ctx, creator.ID)
	require.NoError(t, err)
	require.Equal(t, 1, home.Balance.Orders)
	require.InDelta(t, 100, home.Balance.Gross, 1e-9)
	require.InDelta(t, 90, home.Balance.Net, 1e-9)
	require.InDelta(t, 90, home.Balance.Available, 1e-9)
	sales, total, err := svc.CreatorSales(ctx, creator.ID, 1)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, "available", sales[0].Status)
	require.NotContains(t, sales[0].Buyer, buyer.Email, "buyers are masked")

	// Withdraw: above the balance fails, one pending at a time, then paid by the admin.
	req := service.CreatorWithdrawRequest{CNYAmount: 91, Method: "alipay", Account: "a@alipay", RealName: "王某"}
	_, err = svc.RequestCreatorWithdrawal(ctx, creator.ID, req)
	require.Equal(t, "WITHDRAW_OVER_AVAILABLE", reason(err))
	req.CNYAmount = 50
	w, err := svc.RequestCreatorWithdrawal(ctx, creator.ID, req)
	require.NoError(t, err)
	require.Equal(t, "a@alipay", w.Account)
	_, err = svc.RequestCreatorWithdrawal(ctx, creator.ID, service.CreatorWithdrawRequest{CNYAmount: 10, Method: "alipay", Account: "a", RealName: "王"})
	require.Equal(t, "WITHDRAW_PENDING", reason(err))
	home, err = svc.CreatorHome(ctx, creator.ID)
	require.NoError(t, err)
	require.InDelta(t, 40, home.Balance.Available, 1e-9)
	require.Equal(t, "a@alipay", home.LastAccount)
	_, err = svc.ResolveCreatorWithdrawal(ctx, w.ID, 0, false, "")
	require.Equal(t, "CREATOR_REASON_REQUIRED", reason(err))
	_, err = svc.ResolveCreatorWithdrawal(ctx, w.ID, 0, true, "支付宝转账 123")
	require.NoError(t, err)
	_, err = svc.CancelCreatorWithdrawal(ctx, creator.ID, w.ID)
	require.Equal(t, "WITHDRAW_NOT_PENDING", reason(err))

	// A refund after the payout leaves the creator owing it.
	_, err = integrationEntClient.PaymentOrder.UpdateOneID(order.ID).SetStatus(service.OrderStatusRefunded).Save(ctx)
	require.NoError(t, err)
	home, err = svc.CreatorHome(ctx, creator.ID)
	require.NoError(t, err)
	require.InDelta(t, 0, home.Balance.Net, 1e-9)
	require.InDelta(t, -50, home.Balance.Available, 1e-9)

	// Suspending takes the course off sale; buyers keep what they bought.
	_, err = svc.UpdateCreator(ctx, creator.ID, service.CreatorUpdate{Status: service.CreatorStatusSuspended})
	require.NoError(t, err)
	_, err = svc.Course(ctx, slug, 0)
	require.Equal(t, "COURSE_NOT_FOUND", reason(err))
	_, err = svc.UpdateCreatorCourse(ctx, creator.ID, course.ID, service.CourseInput{Title: "x", Price: 1})
	require.Equal(t, "CREATOR_NOT_APPROVED", reason(err))
}

func courseIDsOf(list []service.Course) []int64 {
	out := make([]int64, 0, len(list))
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}
