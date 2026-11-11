//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type craftRepoFake struct {
	mu       sync.Mutex
	elements map[string]CraftElement
	recipes  map[string]string
	nextID   int64
}

func newCraftRepoFake() *craftRepoFake {
	f := &craftRepoFake{elements: map[string]CraftElement{}, recipes: map[string]string{}}
	for _, n := range craftStarterNames {
		f.nextID++
		f.elements[n] = CraftElement{ID: f.nextID, Name: n, Emoji: "x"}
	}
	return f
}

func (f *craftRepoFake) Elements(_ context.Context, names []string) (map[string]CraftElement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]CraftElement{}
	for _, n := range names {
		if el, ok := f.elements[n]; ok {
			out[n] = el
		}
	}
	return out, nil
}

func (f *craftRepoFake) Recipe(_ context.Context, a, b string) (*CraftElement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.recipes[a+"|"+b]; ok {
		el := f.elements[r]
		return &el, nil
	}
	return nil, nil
}

func (f *craftRepoFake) Save(_ context.Context, a, b string, el CraftElement, _ int64) (CraftElement, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.recipes[a+"|"+b]; ok {
		return f.elements[r], false, nil
	}
	created := false
	if _, ok := f.elements[el.Name]; !ok {
		f.nextID++
		el.ID = f.nextID
		f.elements[el.Name] = el
		created = true
	}
	f.recipes[a+"|"+b] = el.Name
	return f.elements[el.Name], created, nil
}

func (f *craftRepoFake) Stats(_ context.Context, _ int) (int64, []CraftElement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.elements)), nil, nil
}

type craftCounterFake struct {
	mu sync.Mutex
	n  map[string]int64
}

func (c *craftCounterFake) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[key]++
	return c.n[key], nil
}

func (c *craftCounterFake) Decr(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[key]--
	return nil
}

func (c *craftCounterFake) Get(_ context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n[key], nil
}

func newCraftTestService(answer string) (*GameCraftService, *craftRepoFake, *int) {
	repo := newCraftRepoFake()
	calls := 0
	s := NewGameCraftService(repo, nil, &craftCounterFake{n: map[string]int64{}}, nil)
	s.chat = func(_ context.Context, _, _, user string) (string, string, error) {
		calls++
		return answer, "m", nil
	}
	return s, repo, &calls
}

func TestCraftCombineAsksOncePerPair(t *testing.T) {
	s, _, calls := newCraftTestService("```json\n{\"name\":\"蒸汽\",\"emoji\":\"💨 \",\"desc\":\"水遇火，升上天。\"}\n```")
	ctx := context.Background()

	first, err := s.Combine(ctx, LibVisitor{IP: "1.1.1.1"}, "火", "水")
	require.NoError(t, err)
	require.True(t, first.First)
	require.Equal(t, "蒸汽", first.Element.Name)
	require.Equal(t, "💨", first.Element.Emoji)

	// Same pair in the other order, from someone else: stored answer, no model call, not first.
	again, err := s.Combine(ctx, LibVisitor{UserID: 7}, "水", "火")
	require.NoError(t, err)
	require.False(t, again.First)
	require.Equal(t, "蒸汽", again.Element.Name)
	require.Equal(t, 1, *calls)

	// The new element can be used next.
	s.chat = func(context.Context, string, string, string) (string, string, error) {
		return `好的：{"name":"间歇泉","emoji":"⛲","desc":"地下的蒸汽冲破大地。"}`, "m", nil
	}
	geyser, err := s.Combine(ctx, LibVisitor{IP: "1.1.1.1"}, "蒸汽", "土")
	require.NoError(t, err)
	require.Equal(t, "间歇泉", geyser.Element.Name)
}

func TestCraftCombineRejectsUnknownAndBadAnswers(t *testing.T) {
	s, _, calls := newCraftTestService(`{"name":"金","emoji":"🪙","desc":"还是金"}`)
	ctx := context.Background()

	_, err := s.Combine(ctx, LibVisitor{IP: "1.1.1.1"}, "金", "不存在的东西")
	require.ErrorIs(t, err, ErrCraftInvalid)
	require.Equal(t, 0, *calls)

	// The answer repeats an input: unusable, retried once.
	_, err = s.Combine(ctx, LibVisitor{IP: "1.1.1.1"}, "金", "木")
	require.ErrorIs(t, err, ErrCraftFailed)
	require.Equal(t, 2, *calls)

	s2, _, _ := newCraftTestService(`{"name":"毒品工厂","emoji":"🏭","desc":"x"}`)
	_, err = s2.Combine(ctx, LibVisitor{IP: "1.1.1.1"}, "金", "木")
	require.ErrorIs(t, err, ErrCraftRejected)
}

func TestCraftDailyLimitOnlyCountsNewPairs(t *testing.T) {
	s, repo, _ := newCraftTestService("")
	n := 0
	s.chat = func(context.Context, string, string, string) (string, string, error) {
		n++
		return `{"name":"新` + string(rune('甲'+n)) + `","emoji":"✨","desc":"d"}`, "m", nil
	}
	ctx := context.Background()
	guest := LibVisitor{IP: "2.2.2.2"}
	// Seed enough elements to make craftNewGuestPerDay distinct pairs with 水.
	for i := 0; i <= craftNewGuestPerDay; i++ {
		name := "种" + string(rune('一'+i))
		repo.elements[name] = CraftElement{Name: name}
	}
	for i := 0; i < craftNewGuestPerDay; i++ {
		s.now = func() time.Time { return time.Date(2026, 11, 11, 10, i, 0, 0, time.UTC) } // dodge the per-minute limit
		_, err := s.Combine(ctx, guest, "水", "种"+string(rune('一'+i)))
		require.NoError(t, err, i)
	}
	s.now = func() time.Time { return time.Date(2026, 11, 11, 11, 0, 0, 0, time.UTC) }
	_, err := s.Combine(ctx, guest, "水", "种"+string(rune('一'+craftNewGuestPerDay)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "登录")
	// A known pair still works.
	_, err = s.Combine(ctx, guest, "水", "种一")
	require.NoError(t, err)
}

func TestCraftEmoji(t *testing.T) {
	require.Equal(t, "✨", craftEmoji(""))
	require.Equal(t, "✨", craftEmoji("abc"))
	require.Equal(t, "🐉", craftEmoji(" 🐉 龙"[0:5]))
	require.Equal(t, "👩‍🚀", craftEmoji("👩‍🚀"))
}
