package service

import (
	"context"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// Starter key: half the members who signed up never created an API key, so the dashboard's
// 「三步开始使用」 asks for one to be made for them — in the first pay-as-you-go group of the
// default platform they may use. With smart billing the same key also draws on any
// subscription they buy later, so the group choice doesn't trap them.
const (
	StarterKeyName     = "我的第一个 Key"
	starterKeyPlatform = PlatformOpenAI
)

type StarterKeyService struct {
	keys    *APIKeyService
	users   UserRepository
	routing *BillingRouteService
	locks   sync.Map // userID → *sync.Mutex, so a double click makes one key
}

func NewStarterKeyService(keys *APIKeyService, users UserRepository, routing *BillingRouteService) *StarterKeyService {
	return &StarterKeyService{keys: keys, users: users, routing: routing}
}

// Ensure returns the user's newest key, creating the starter key when they have none.
// created tells whether it was made now.
func (s *StarterKeyService) Ensure(ctx context.Context, userID int64) (key *APIKey, created bool, err error) {
	v, _ := s.locks.LoadOrStore(userID, &sync.Mutex{})
	if mu, ok := v.(*sync.Mutex); ok {
		mu.Lock()
		defer mu.Unlock()
	}

	existing, _, err := s.keys.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 1}, APIKeyListFilters{})
	if err != nil {
		return nil, false, err
	}
	if len(existing) > 0 {
		k := existing[0]
		return &k, false, nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	req := CreateAPIKeyRequest{Name: StarterKeyName}
	if g := s.routing.DefaultPayAsYouGoGroup(ctx, user, starterKeyPlatform); g != nil {
		id := g.ID
		req.GroupID = &id
	}
	k, err := s.keys.Create(ctx, userID, req)
	if err != nil {
		return nil, false, err
	}
	return k, true, nil
}
