package service

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"
)

// 一个 Key，订阅和按量自动切换. A key is bound to one group, so a member with a subscription
// key who ran out (or let it lapse) got errors although they had balance, and a member with a
// pay-as-you-go key kept paying from balance after buying a subscription. When smart billing is
// on, each request is served by:
//   - a subscription key: its own subscription while usable; else another usable subscription of
//     the same platform; else the platform's pay-as-you-go group, billed to balance (when the
//     balance passes the auth check);
//   - a pay-as-you-go key: a usable subscription of the same platform for text requests (the
//     caller decides which endpoints may move), else the key's own group.
//
// "Usable" means active and within its daily / weekly / monthly limits. The switch is per
// request; usage records show the group that actually served it.
const (
	SettingKeySmartBillingEnabled = "smart_billing_enabled"

	billingRouteSubsTTL     = 30 * time.Second
	billingRouteGroupsTTL   = time.Minute
	billingRouteSettingTTL  = 30 * time.Second
	billingRouteMaxUserSubs = 20000
)

type billingRouteUserSubs struct {
	at   time.Time
	subs []UserSubscription
}

type BillingRouteService struct {
	subs     *SubscriptionService
	groups   GroupRepository
	settings SettingRepository
	now      func() time.Time

	mu        sync.Mutex
	userSubs  map[int64]billingRouteUserSubs
	groupList []Group
	groupsAt  time.Time
	enabled   bool
	enabledAt time.Time
}

func NewBillingRouteService(subs *SubscriptionService, groups GroupRepository, settings SettingRepository) *BillingRouteService {
	s := &BillingRouteService{subs: subs, groups: groups, settings: settings, now: time.Now, userSubs: map[int64]billingRouteUserSubs{}}
	subs.SetUserSubsChangedHook(s.Forget)
	return s
}

// Enabled: on unless an admin turned it off.
func (s *BillingRouteService) Enabled(ctx context.Context) bool {
	if s == nil || s.settings == nil {
		return false
	}
	s.mu.Lock()
	if !s.enabledAt.IsZero() && s.now().Sub(s.enabledAt) < billingRouteSettingTTL {
		on := s.enabled
		s.mu.Unlock()
		return on
	}
	s.mu.Unlock()
	v, err := s.settings.GetValue(ctx, SettingKeySmartBillingEnabled)
	on := err != nil || v != "false" // a missing setting reads as an error: default on
	s.mu.Lock()
	s.enabled, s.enabledAt = on, s.now()
	s.mu.Unlock()
	return on
}

// SetEnabled turns smart billing on or off (takes effect at once on this instance).
func (s *BillingRouteService) SetEnabled(ctx context.Context, on bool) error {
	if err := s.settings.Set(ctx, SettingKeySmartBillingEnabled, strconv.FormatBool(on)); err != nil {
		return err
	}
	s.mu.Lock()
	s.enabled, s.enabledAt = on, s.now()
	s.mu.Unlock()
	return nil
}

// Forget drops what is cached for a user (after a purchase, so the new subscription is used at once).
func (s *BillingRouteService) Forget(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.userSubs, userID)
	s.mu.Unlock()
}

// Route returns the group that should serve and pay for this request instead of the key's own,
// or nil to keep the key's group. upgrade: a pay-as-you-go request may move onto a subscription.
// balanceOK: the user's balance passes the auth threshold (needed to fall back to balance).
func (s *BillingRouteService) Route(ctx context.Context, apiKey *APIKey, upgrade, balanceOK bool) *Group {
	if s == nil || s.subs == nil || s.groups == nil || apiKey == nil || apiKey.Group == nil || apiKey.User == nil || !s.Enabled(ctx) {
		return nil
	}
	own := apiKey.Group
	user := apiKey.User
	if own.IsSubscriptionType() {
		if s.subscriptionUsable(ctx, user.ID, own) {
			return nil
		}
		if g := s.usableSubscriptionGroup(ctx, user, own.Platform, own.ID); g != nil {
			return g
		}
		if balanceOK {
			return s.payAsYouGoGroup(ctx, user, own.Platform)
		}
		return nil
	}
	if !upgrade {
		return nil
	}
	return s.usableSubscriptionGroup(ctx, user, own.Platform, 0)
}

// subscriptionUsable: the user's subscription to group exists, is active and within its limits.
func (s *BillingRouteService) subscriptionUsable(ctx context.Context, userID int64, group *Group) bool {
	sub, err := s.subs.GetActiveSubscription(ctx, userID, group.ID)
	if err != nil || sub == nil {
		return false
	}
	needsMaintenance, err := s.subs.ValidateAndCheckLimits(sub, group)
	if needsMaintenance {
		refreshed, mErr := s.subs.EnsureWindowMaintenance(ctx, sub)
		if mErr != nil {
			return false
		}
		_, err = s.subs.ValidateAndCheckLimits(refreshed, group)
	}
	return err == nil
}

// usableSubscriptionGroup: the first usable subscription of the user on platform (skipping
// skipGroupID), by the groups' sort order, then the one expiring first.
func (s *BillingRouteService) usableSubscriptionGroup(ctx context.Context, user *User, platform string, skipGroupID int64) *Group {
	subs := s.activeSubs(ctx, user.ID)
	if len(subs) == 0 {
		return nil
	}
	byID := s.activeGroups(ctx)
	type cand struct {
		group   Group
		expires time.Time
	}
	var cands []cand
	for _, sub := range subs {
		g, ok := byID[sub.GroupID]
		if !ok || g.ID == skipGroupID || !g.IsSubscriptionType() || g.Platform != platform {
			continue
		}
		cands = append(cands, cand{group: g, expires: sub.ExpiresAt})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].group.SortOrder != cands[j].group.SortOrder {
			return cands[i].group.SortOrder < cands[j].group.SortOrder
		}
		return cands[i].expires.Before(cands[j].expires)
	})
	for i := range cands {
		g := cands[i].group
		if s.subscriptionUsable(ctx, user.ID, &g) {
			return &g
		}
	}
	return nil
}

// payAsYouGoGroup: the platform's first active pay-as-you-go group the user may use.
func (s *BillingRouteService) payAsYouGoGroup(ctx context.Context, user *User, platform string) *Group {
	s.activeGroups(ctx)
	s.mu.Lock()
	list := s.groupList
	s.mu.Unlock()
	for i := range list { // already ordered by sort order, id
		g := list[i]
		if g.Platform == platform && !g.IsSubscriptionType() && user.CanBindGroup(g.ID, g.IsExclusive) {
			return &g
		}
	}
	return nil
}

func (s *BillingRouteService) activeSubs(ctx context.Context, userID int64) []UserSubscription {
	now := s.now()
	s.mu.Lock()
	if e, ok := s.userSubs[userID]; ok && now.Sub(e.at) < billingRouteSubsTTL {
		s.mu.Unlock()
		return e.subs
	}
	s.mu.Unlock()
	subs, err := s.subs.ListActiveUserSubscriptions(ctx, userID)
	if err != nil {
		return nil
	}
	s.mu.Lock()
	if len(s.userSubs) >= billingRouteMaxUserSubs {
		s.userSubs = map[int64]billingRouteUserSubs{}
	}
	s.userSubs[userID] = billingRouteUserSubs{at: now, subs: subs}
	s.mu.Unlock()
	return subs
}

func (s *BillingRouteService) activeGroups(ctx context.Context) map[int64]Group {
	now := s.now()
	s.mu.Lock()
	fresh := !s.groupsAt.IsZero() && now.Sub(s.groupsAt) < billingRouteGroupsTTL
	list := s.groupList
	s.mu.Unlock()
	if !fresh {
		if groups, err := s.groups.ListActive(ctx); err == nil {
			list = groups
			s.mu.Lock()
			s.groupList, s.groupsAt = groups, now
			s.mu.Unlock()
		}
	}
	out := make(map[int64]Group, len(list))
	for _, g := range list {
		out[g.ID] = g
	}
	return out
}
