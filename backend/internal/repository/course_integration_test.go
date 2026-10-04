//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// reverseEncryptor stands in for the AES encryptor: stored values must differ from the plaintext.
type reverseEncryptor struct{}

func (reverseEncryptor) Encrypt(s string) (string, error) { return "enc:" + reverse(s), nil }
func (reverseEncryptor) Decrypt(s string) (string, error) {
	return reverse(strings.TrimPrefix(s, "enc:")), nil
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func TestCourses(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	buyer := mustCreateUser(t, integrationEntClient, &service.User{Email: "co-b-" + suffix + "@test.local", Username: "cob" + suffix[len(suffix)-6:]})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "co-o-" + suffix + "@test.local", Username: "coo" + suffix[len(suffix)-6:]})
	svc := service.NewCourseService(NewCourseRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), reverseEncryptor{})
	payments := service.NewPaymentService(integrationEntClient, nil, nil, nil, nil, nil, nil, nil, nil)
	payments.SetCourseService(svc)
	reason := func(err error) string { return infraerrors.Reason(err) }
	slug := "ai-agent-" + suffix[len(suffix)-6:]

	// Validation, then a draft: not listed, not sellable.
	_, err := svc.CreateCourse(ctx, service.CourseInput{Slug: "Bad Slug", Title: "x", Price: 99})
	require.Equal(t, "COURSE_INVALID", reason(err))
	_, err = svc.CreateCourse(ctx, service.CourseInput{Slug: slug, Title: "x", Price: 0})
	require.Equal(t, "COURSE_INVALID", reason(err))
	course, err := svc.CreateCourse(ctx, service.CourseInput{
		Slug: slug, Title: "AI Agent 实战", Subtitle: "从零做一个能上线的 Agent", Price: 199, OriginalPrice: 299,
		IntroMD: "## 适合谁", Outline: []service.CourseSection{{Title: "第一章", Lessons: []service.CourseLesson{{Title: "Agent 是什么", Duration: "12:00", Trial: true}, {Title: ""}}}},
	})
	require.NoError(t, err)
	require.Equal(t, service.CourseStatusDraft, course.Status)
	require.Equal(t, 1, course.LessonCount)
	_, err = svc.CreateCourse(ctx, service.CourseInput{Slug: slug, Title: "重复", Price: 1})
	require.Equal(t, "COURSE_SLUG_TAKEN", reason(err))
	_, err = svc.Course(ctx, slug, 0)
	require.Equal(t, "COURSE_NOT_FOUND", reason(err))
	_, err = svc.CourseForOrder(ctx, buyer.ID, course.ID)
	require.Equal(t, "COURSE_NOT_FOUND", reason(err))

	// Published: on the list, with intro on the page only; no delivery yet.
	in := service.CourseInput{Slug: slug, Title: course.Title, Price: 199, OriginalPrice: 299, IntroMD: "## 适合谁", Outline: course.Outline, Status: service.CourseStatusPublished}
	_, err = svc.UpdateCourse(ctx, course.ID, in)
	require.NoError(t, err)
	list, err := svc.Courses(ctx, buyer.ID)
	require.NoError(t, err)
	var listed *service.Course
	for i := range list {
		if list[i].ID == course.ID {
			listed = &list[i]
		}
	}
	require.NotNil(t, listed)
	require.Empty(t, listed.IntroMD)
	require.False(t, listed.Owned)
	page, err := svc.Course(ctx, slug, buyer.ID)
	require.NoError(t, err)
	require.Equal(t, "## 适合谁", page.IntroMD)
	priced, err := svc.CourseForOrder(ctx, buyer.ID, course.ID)
	require.NoError(t, err)
	require.Equal(t, 199.0, priced.Price)
	_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
	require.Equal(t, "COURSE_NOT_ENROLLED", reason(err))

	// A paid course order: fulfilment opens the course.
	order, err := integrationEntClient.PaymentOrder.Create().
		SetUserID(buyer.ID).SetUserEmail(buyer.Email).SetUserName(buyer.Username).
		SetAmount(199).SetPayAmount(199).SetFeeRate(0).SetRechargeCode("").
		SetOutTradeNo("co" + suffix).SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeCourse).SetCourseID(course.ID).SetStatus(service.OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, payments.ExecuteCourseFulfillment(ctx, order.ID))
	require.NoError(t, payments.ExecuteCourseFulfillment(ctx, order.ID)) // idempotent
	done, err := integrationEntClient.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, service.OrderStatusCompleted, done.Status)
	_, err = svc.CourseForOrder(ctx, buyer.ID, course.ID)
	require.Equal(t, "COURSE_ALREADY_OWNED", reason(err))
	page, err = svc.Course(ctx, slug, buyer.ID)
	require.NoError(t, err)
	require.True(t, page.Owned)
	require.Equal(t, 1, page.StudentCount)

	// Bought before the materials are up: a clear "not ready".
	_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
	require.Equal(t, "COURSE_NO_DELIVERY", reason(err))

	// The admin adds the link (stored encrypted); the buyer gets it, others do not.
	_, err = svc.SaveDelivery(ctx, course.ID, 0, service.DeliveryInput{Link: "http://pan.baidu.com/s/x"}, "https://hivegpt.test")
	require.Equal(t, "COURSE_INVALID", reason(err))
	_, err = svc.SaveDelivery(ctx, course.ID, 0, service.DeliveryInput{Link: "https://pan.baidu.com/s/1abc", Code: "ab12", Note: "解压后先看 README"}, "https://hivegpt.test")
	require.NoError(t, err)
	var stored string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT link_enc FROM course_deliveries WHERE course_id = $1`, course.ID).Scan(&stored))
	require.NotContains(t, stored, "pan.baidu.com")
	d, err := svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
	require.NoError(t, err)
	require.Equal(t, "https://pan.baidu.com/s/1abc", d.Link)
	require.Equal(t, "ab12", d.Code)
	require.Equal(t, 1, d.Version)
	_, err = svc.Delivery(ctx, other.ID, course.ID, "2.2.2.2")
	require.Equal(t, "COURSE_NOT_ENROLLED", reason(err))

	// A replaced link: "updated" until the buyer looks again.
	_, err = svc.SaveDelivery(ctx, course.ID, 0, service.DeliveryInput{Link: "https://pan.baidu.com/s/2new", Code: "cd34"}, "https://hivegpt.test")
	require.NoError(t, err)
	mine, err := svc.MyCourses(ctx, buyer.ID)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	require.True(t, mine[0].Updated)
	require.Empty(t, mine[0].Course.IntroMD)
	d, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.2")
	require.NoError(t, err)
	require.Equal(t, "https://pan.baidu.com/s/2new", d.Link)
	mine, err = svc.MyCourses(ctx, buyer.ID)
	require.NoError(t, err)
	require.False(t, mine[0].Updated)
	versions, err := svc.Deliveries(ctx, course.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 2, versions[0].Version)

	// Views are logged; the admin sees counts and distinct IPs; too many in a minute are refused.
	students, total, err := svc.Enrollments(ctx, course.ID, 1, 30)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, 2, students[0].ViewCount)
	require.Equal(t, 2, students[0].RecentIPs)
	require.NotNil(t, students[0].FirstViewedAt)
	for i := 0; i < 8; i++ {
		_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
		require.NoError(t, err)
	}
	_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
	require.Equal(t, "COURSE_TOO_MANY_VIEWS", reason(err))

	// With students the course cannot be deleted; off sale it stays visible to its buyer only.
	require.Equal(t, "COURSE_HAS_STUDENTS", reason(svc.DeleteCourse(ctx, course.ID)))
	in.Status = service.CourseStatusArchived
	_, err = svc.UpdateCourse(ctx, course.ID, in)
	require.NoError(t, err)
	_, err = svc.Course(ctx, slug, buyer.ID)
	require.NoError(t, err)
	_, err = svc.Course(ctx, slug, other.ID)
	require.Equal(t, "COURSE_NOT_FOUND", reason(err))

	// Admin grant by email, then revoke.
	require.Equal(t, "COURSE_USER_NOT_FOUND", reason(svc.Grant(ctx, course.ID, "nobody-"+suffix+"@test.local", "")))
	require.NoError(t, svc.Grant(ctx, course.ID, strings.ToUpper(other.Email), "线下付款"))
	page, err = svc.Course(ctx, slug, other.ID)
	require.NoError(t, err)
	require.True(t, page.Owned)
	require.NoError(t, svc.Revoke(ctx, course.ID, other.ID))
	_, err = svc.Delivery(ctx, other.ID, course.ID, "3.3.3.3")
	require.Equal(t, "COURSE_NOT_ENROLLED", reason(err))

	// A full refund of the order ends access; the course stays listed as refunded.
	_, err = integrationEntClient.PaymentOrder.UpdateOneID(order.ID).SetStatus(service.OrderStatusRefunded).Save(ctx)
	require.NoError(t, err)
	_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.9")
	require.Equal(t, "COURSE_NOT_ENROLLED", reason(err))
	mine, err = svc.MyCourses(ctx, buyer.ID)
	require.NoError(t, err)
	require.True(t, mine[0].Enrollment.Refunded)
	require.False(t, mine[0].Course.Owned)
}

func TestCoursesC2(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	buyer := mustCreateUser(t, integrationEntClient, &service.User{Email: "c2-b-" + suffix + "@test.local", Username: "c2b" + suffix[len(suffix)-6:]})
	svc := service.NewCourseService(NewCourseRepository(integrationDB), service.NewCommunityMediaStore(t.TempDir()), reverseEncryptor{})
	reason := func(err error) string { return infraerrors.Reason(err) }
	ends := time.Now().Add(48 * time.Hour)
	slug := "c2-" + suffix[len(suffix)-8:]

	// Limited-time price needs an end; trial video must be https.
	_, err := svc.CreateCourse(ctx, service.CourseInput{Slug: slug, Title: "限时课", Price: 199, SalePrice: 99, Status: service.CourseStatusPublished})
	require.Equal(t, "COURSE_INVALID", reason(err))
	_, err = svc.CreateCourse(ctx, service.CourseInput{Slug: slug, Title: "限时课", Price: 199, TrialVideoURL: "http://x/v.mp4", Status: service.CourseStatusPublished})
	require.Equal(t, "COURSE_INVALID", reason(err))
	course, err := svc.CreateCourse(ctx, service.CourseInput{Slug: slug, Title: "限时课", Price: 199, SalePrice: 99, SaleEndsAt: &ends, EduDiscount: true,
		TrialVideoURL: "https://www.bilibili.com/video/BV1xx411c7mD", Status: service.CourseStatusPublished})
	require.NoError(t, err)
	require.Equal(t, 99.0, course.SalePrice)
	require.True(t, course.EduDiscount)
	require.NotNil(t, course.SaleEndsAt)

	// The page shows the running sale; visitors are counted, admin sees them.
	page, err := svc.Course(ctx, slug, buyer.ID)
	require.NoError(t, err)
	require.True(t, page.SaleActive)
	require.Equal(t, 99.0, page.CurrentPrice)
	require.Zero(t, page.Views30d) // admin-only numbers stay hidden
	_, err = svc.Course(ctx, slug, 0)
	require.NoError(t, err)
	admin, err := svc.AdminCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Equal(t, 2, admin.Views30d)
	priced, err := svc.CourseForOrder(ctx, buyer.ID, course.ID)
	require.NoError(t, err)
	require.Equal(t, 99.0, priced.CurrentPrice)

	// A course redeem code opens the course inside the redeem transaction: a rollback leaves nothing.
	require.NoError(t, svc.CheckRedeem(ctx, buyer.ID, course.ID))
	require.Equal(t, "COURSE_REDEEM_INVALID", reason(svc.CheckRedeem(ctx, buyer.ID, course.ID+100000)))
	tx, err := integrationEntClient.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.EnrollRedeem(dbent.NewTxContext(ctx, tx), buyer.ID, course.ID, "CODE-1"))
	require.NoError(t, tx.Rollback())
	_, err = svc.Delivery(ctx, buyer.ID, course.ID, "1.1.1.1")
	require.Equal(t, "COURSE_NOT_ENROLLED", reason(err))
	tx, err = integrationEntClient.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.EnrollRedeem(dbent.NewTxContext(ctx, tx), buyer.ID, course.ID, "CODE-1"))
	require.NoError(t, tx.Commit())
	require.Equal(t, "COURSE_ALREADY_OWNED", reason(svc.CheckRedeem(ctx, buyer.ID, course.ID)))
	students, _, err := svc.Enrollments(ctx, course.ID, 1, 30)
	require.NoError(t, err)
	require.Equal(t, service.CourseSourceRedeem, students[0].Source)
	require.Equal(t, "兑换码 CODE-1", students[0].Note)

	// Owners' visits are not counted.
	_, err = svc.Course(ctx, slug, buyer.ID)
	require.NoError(t, err)
	admin, err = svc.AdminCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Equal(t, 2, admin.Views30d)
}
