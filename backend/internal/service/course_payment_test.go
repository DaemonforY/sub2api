//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

// courseEnrollRepoStub records enrollments; other repository calls are not made by fulfilment.
type courseEnrollRepoStub struct {
	CourseRepository
	enrolled []CourseEnrollment
	sales    []int64
}

func (r *courseEnrollRepoStub) RecordCreatorSale(_ context.Context, orderID int64, _ float64, _ int) error {
	r.sales = append(r.sales, orderID)
	return nil
}

func (r *courseEnrollRepoStub) UpsertEnrollment(_ context.Context, e *CourseEnrollment) error {
	r.enrolled = append(r.enrolled, *e)
	return nil
}

type courseSettingsStub struct{ values map[string]string }

func (s *courseSettingsStub) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	return s.values, nil
}
func (s *courseSettingsStub) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		s.values[k] = v
	}
	return nil
}

func TestExecuteCourseFulfillmentEnrollsAndPaysCourseRebate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		rate      string
		wantCalls int
		want      float64
	}{
		{name: "course rate, not the inviter's usual 15%", rate: "20", wantCalls: 1, want: 39.8},
		{name: "rate 0: no rebate", rate: "0", wantCalls: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := newPaymentConfigServiceTestClient(t)
			ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
			user, err := client.User.Create().SetEmail("course-buyer@example.com").SetPasswordHash("hash").SetUsername("course-buyer").Save(ctx)
			require.NoError(t, err)
			order, err := client.PaymentOrder.Create().
				SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).
				SetAmount(199).SetPayAmount(199).SetFeeRate(0).SetRechargeCode("").
				SetOutTradeNo("course_rebate_" + tc.rate).SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("t").
				SetOrderType(payment.OrderTypeCourse).SetCourseID(5).SetStatus(OrderStatusPaid).
				SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("hivegpt.cn").
				Save(ctx)
			require.NoError(t, err)

			inviterID := int64(9001)
			affiliateRepo := &paymentFulfillmentAffiliateRepoStub{
				inviteeSummary: &AffiliateSummary{UserID: user.ID, AffCode: "INVITEE", InviterID: &inviterID, CreatedAt: time.Now().Add(-24 * time.Hour)},
				inviterSummary: &AffiliateSummary{UserID: inviterID, AffCode: "INVITER", CreatedAt: time.Now().Add(-48 * time.Hour)},
			}
			settingSvc := NewSettingService(&paymentFulfillmentSettingRepoStub{values: map[string]string{
				SettingKeyAffiliateEnabled:           "true",
				SettingKeyAffiliateRebateRate:        "15",
				SettingKeyAffiliateRebateFreezeHours: "0",
			}}, nil)
			repo := &courseEnrollRepoStub{}
			courses := NewCourseService(repo, nil, nil)
			courses.SetSettings(&courseSettingsStub{values: map[string]string{settingCourseAffiliateRate: tc.rate}})
			svc := &PaymentService{entClient: client, affiliateService: NewAffiliateService(affiliateRepo, settingSvc, nil, nil), courses: courses}

			require.NoError(t, svc.ExecuteCourseFulfillment(ctx, order.ID))
			reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
			require.NoError(t, err)
			require.Equal(t, OrderStatusCompleted, reloaded.Status)
			require.Len(t, repo.enrolled, 1)
			require.Equal(t, CourseSourcePurchase, repo.enrolled[0].Source)
			require.Equal(t, order.ID, repo.enrolled[0].OrderID)
			require.Equal(t, []int64{order.ID}, repo.sales, "the creator's share is recorded with the enrollment")
			require.Len(t, affiliateRepo.accrueCalls, tc.wantCalls)
			if tc.wantCalls > 0 {
				require.InDelta(t, tc.want, affiliateRepo.accrueCalls[0].amount, 1e-9)
				require.Equal(t, inviterID, affiliateRepo.accrueCalls[0].inviterID)
			}
		})
	}
}

type fakeCoursePricer struct {
	verified map[int64]bool
	percent  float64
}

func (p fakeCoursePricer) DiscountedPrice(_ context.Context, userID int64, price float64) (float64, bool) {
	if !p.verified[userID] {
		return price, false
	}
	return roundCents(price * (100 - p.percent) / 100), true
}

func (p fakeCoursePricer) GetSettings(_ context.Context) (*GrowthSettings, error) {
	return &GrowthSettings{EduVerifyEnabled: true, EduDiscountPercent: p.percent}, nil
}

func TestCoursePriceFor(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	svc := NewCourseService(nil, nil, nil)
	svc.now = func() time.Time { return now }
	svc.SetPricer(fakeCoursePricer{verified: map[int64]bool{7: true}, percent: 10})
	later, earlier := now.Add(time.Hour), now.Add(-time.Hour)
	edu := &GrowthSettings{EduVerifyEnabled: true, EduDiscountPercent: 10}

	c := Course{Price: 199, SalePrice: 149, SaleEndsAt: &later, EduDiscount: true}
	svc.priceFor(ctx, &c, 7, edu)
	require.True(t, c.SaleActive)
	require.True(t, c.EduApplied)
	require.Equal(t, 134.1, c.CurrentPrice) // 10% off the limited-time price

	c = Course{Price: 199, SalePrice: 149, SaleEndsAt: &earlier, EduDiscount: true}
	svc.priceFor(ctx, &c, 8, edu)
	require.False(t, c.SaleActive)
	require.Equal(t, 199.0, c.CurrentPrice)
	require.Equal(t, 10.0, c.EduPercent) // not verified: shown what verification gives

	c = Course{Price: 199, EduDiscount: false}
	svc.priceFor(ctx, &c, 7, edu)
	require.False(t, c.EduApplied)
	require.Zero(t, c.EduPercent)
	require.Equal(t, 199.0, c.CurrentPrice)
}
