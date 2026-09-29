package service

import "context"

// EduService manages education (school email) verification and its perks.
type EduService struct{}

// DiscountedPrice returns the price userID pays after the education discount, and
// whether a discount was applied.
func (s *EduService) DiscountedPrice(ctx context.Context, userID int64, price float64) (float64, bool) {
	return price, false
}
