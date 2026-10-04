//go:build unit

package service

import (
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCourseInputApply(t *testing.T) {
	var c Course
	require.Error(t, CourseInput{Slug: "-bad", Title: "t", Price: 1}.apply(&c))
	require.Error(t, CourseInput{Slug: "ok-slug", Title: " ", Price: 1}.apply(&c))
	require.Error(t, CourseInput{Slug: "ok-slug", Title: "t", Price: -1}.apply(&c))

	require.NoError(t, CourseInput{
		Slug: "Agent-101", Title: "  Agent  ", Price: 99.999, OriginalPrice: 50, Status: "weird",
		Outline: []CourseSection{{Title: "", Lessons: []CourseLesson{{Title: ""}}}, {Title: "第一章", Lessons: []CourseLesson{{Title: "开场", Duration: "3:00", Trial: true}}}},
	}.apply(&c))
	require.Equal(t, "agent-101", c.Slug)
	require.Equal(t, 100.0, c.Price)
	require.Zero(t, c.OriginalPrice) // not above the price: no strikethrough
	require.Equal(t, CourseStatusDraft, c.Status)
	require.Len(t, c.Outline, 1)
	require.Equal(t, 1, lessonCount(c.Outline))
}

func TestEnrollmentActive(t *testing.T) {
	var none *CourseEnrollment
	require.False(t, none.Active())
	require.True(t, (&CourseEnrollment{Status: CourseEnrollmentActive}).Active())
	require.False(t, (&CourseEnrollment{Status: CourseEnrollmentActive, Refunded: true}).Active())
	require.False(t, (&CourseEnrollment{Status: CourseEnrollmentRevoked}).Active())
}

func TestWeChatOAuthStartURLCarriesCourse(t *testing.T) {
	raw, err := buildWeChatPaymentOAuthStartURL(CreateOrderRequest{PaymentType: payment.TypeWxpay, OrderType: payment.OrderTypeCourse, CourseID: 9}, "snsapi_base")
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "9", u.Query().Get("course_id"))
	require.Equal(t, payment.OrderTypeCourse, u.Query().Get("order_type"))
}
