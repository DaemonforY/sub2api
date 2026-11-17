//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeGameRepo struct {
	boards []string
	saved  json.RawMessage
	scores map[string]int64
}

func (f *fakeGameRepo) GetSave(context.Context, int64, string) (*GameSave, error) { return nil, nil }
func (f *fakeGameRepo) PutSave(_ context.Context, _ int64, _ string, d json.RawMessage) (time.Time, error) {
	f.saved = d
	return time.Now(), nil
}
func (f *fakeGameRepo) BoardsOf(context.Context, int64, string) ([]string, error) { return f.boards, nil }
func (f *fakeGameRepo) SubmitScore(_ context.Context, _ int64, _, board string, score int64, _ json.RawMessage) (int64, bool, error) {
	if f.scores == nil {
		f.scores = map[string]int64{}
	}
	if score > f.scores[board] {
		f.scores[board] = score
		return score, true, nil
	}
	return f.scores[board], false, nil
}
func (f *fakeGameRepo) Leaderboard(context.Context, string, string, int) ([]GameScoreRow, error) {
	return []GameScoreRow{{UserID: 7, Username: "阿浪", Score: 9, Rank: 1, Detail: json.RawMessage(`{}`)}, {UserID: 8, Email: "kid@example.com", Score: 5, Rank: 2}}, nil
}
func (f *fakeGameRepo) Players(context.Context, string, string) (int, error) { return 2, nil }
func (f *fakeGameRepo) EntryFor(_ context.Context, uid int64, _, _ string) (*GameScoreRow, error) {
	if uid == 8 {
		return &GameScoreRow{UserID: 8, Email: "kid@example.com", Score: 5, Rank: 2}, nil
	}
	return nil, nil
}

func TestGameProgress_ValidatesInput(t *testing.T) {
	ctx := context.Background()
	repo := &fakeGameRepo{}
	s := NewGameProgressService(repo)

	_, err := s.GetSave(ctx, 1, "nope")
	require.ErrorIs(t, err, ErrGameUnknown)
	_, err = s.PutSave(ctx, 1, "hutu", json.RawMessage(`[1,2]`))
	require.ErrorIs(t, err, ErrGameBadInput)
	_, err = s.PutSave(ctx, 1, "hutu", json.RawMessage(`{"x":"`+strings.Repeat("a", gameSaveMaxBytes)+`"}`))
	require.ErrorIs(t, err, ErrGameSaveSize)
	out, err := s.PutSave(ctx, 1, "hutu", json.RawMessage(`{"level":3}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"level":3}`, string(out.Data))

	_, err = s.SubmitScore(ctx, 1, "cotton", "main", -1, nil)
	require.ErrorIs(t, err, ErrGameBadInput)
	_, err = s.SubmitScore(ctx, 1, "hutu", "main", 1<<40, nil)
	require.ErrorIs(t, err, ErrGameBadInput)
	_, err = s.SubmitScore(ctx, 1, "cotton", "Bad Board", 1, nil)
	require.ErrorIs(t, err, ErrGameBadInput)
	_, err = s.SubmitScore(ctx, 1, "cotton", "", 1, json.RawMessage(`{"x":"`+strings.Repeat("a", 600)+`"}`))
	require.ErrorIs(t, err, ErrGameBadInput)
}

func TestGameProgress_ScoresAndBoards(t *testing.T) {
	ctx := context.Background()
	repo := &fakeGameRepo{}
	s := NewGameProgressService(repo)

	r, err := s.SubmitScore(ctx, 1, "cotton", "", 120, nil)
	require.NoError(t, err)
	require.True(t, r.Improved)
	require.EqualValues(t, 120, r.Best)
	require.Equal(t, 2, r.Players)
	r, err = s.SubmitScore(ctx, 1, "cotton", "", 80, nil)
	require.NoError(t, err)
	require.False(t, r.Improved)
	require.EqualValues(t, 120, r.Best)

	// A player with the maximum number of boards can update those, not open a new one.
	repo.boards = []string{"a", "b", "c", "d"}
	_, err = s.SubmitScore(ctx, 1, "cotton", "e", 1, nil)
	require.ErrorIs(t, err, ErrGameBoards)
	_, err = s.SubmitScore(ctx, 1, "cotton", "a", 1, nil)
	require.NoError(t, err)
}

func TestGameProgress_LeaderboardMasksNames(t *testing.T) {
	s := NewGameProgressService(&fakeGameRepo{})
	lb, err := s.Leaderboard(context.Background(), "island", "", 8)
	require.NoError(t, err)
	require.Equal(t, "main", lb.Board)
	require.Equal(t, 2, lb.Players)
	require.Equal(t, "阿***", lb.Top[0].DisplayName)
	require.Nil(t, lb.Top[0].Detail)
	require.NotContains(t, lb.Top[1].DisplayName, "kid@")
	require.True(t, lb.Top[1].IsCurrentUser)
	require.NotNil(t, lb.Me)
	require.Equal(t, 2, lb.Me.Rank)
}
