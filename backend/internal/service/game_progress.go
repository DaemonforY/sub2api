package service

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 小游戏可选登录: visitors play without an account; a signed-in player keeps progress in the cloud
// (game_saves) and gets onto the site-wide leaderboard (game_scores). The game decides what its
// progress JSON holds; the server only checks the size.

// gameProgressGames lists the games that keep progress, each with a score cap so a forged request
// can't take the board with 2^63.
var gameProgressGames = map[string]int64{
	"hutu":   10_000_000,
	"island": 10_000_000,
	"cotton": 100_000_000,
	"planet": 10_000_000,
}

const (
	gameSaveMaxBytes        = 32 << 10
	gameScoreDetailMaxBytes = 512
	gameBoardMaxPerGame     = 4
	gameLeaderboardSize     = 20
)

var gameBoardRe = regexp.MustCompile(`^[a-z0-9-]{1,24}$`)

var (
	ErrGameUnknown  = infraerrors.NotFound("GAME_UNKNOWN", "没有这个小游戏（Unknown game）")
	ErrGameSaveSize = infraerrors.BadRequest("GAME_SAVE_TOO_BIG", "进度数据太大，存不下（Save data too large）")
	ErrGameBadInput = infraerrors.BadRequest("GAME_BAD_INPUT", "成绩数据不对，刷新页面再试（Invalid score data）")
	ErrGameBoards   = infraerrors.BadRequest("GAME_TOO_MANY_BOARDS", "这个游戏的排行榜太多了（Too many leaderboards for this game）")
)

type GameSave struct {
	Data      json.RawMessage `json:"data"`
	UpdatedAt *time.Time      `json:"updated_at"`
}

type GameScoreRow struct {
	UserID    int64
	Username  string
	Email     string
	Score     int64
	Detail    json.RawMessage
	Rank      int
	UpdatedAt time.Time
}

type GameLeaderboardEntry struct {
	Rank          int             `json:"rank"`
	DisplayName   string          `json:"display_name"`
	Score         int64           `json:"score"`
	Detail        json.RawMessage `json:"detail,omitempty"`
	IsCurrentUser bool            `json:"is_current_user"`
}

type GameLeaderboard struct {
	Game    string                 `json:"game"`
	Board   string                 `json:"board"`
	Players int                    `json:"players"`
	Top     []GameLeaderboardEntry `json:"top"`
	Me      *GameLeaderboardEntry  `json:"me,omitempty"`
}

type GameScoreResult struct {
	Best     int64 `json:"best"`
	Improved bool  `json:"improved"`
	Rank     int   `json:"rank"`
	Players  int   `json:"players"`
}

type GameProgressRepository interface {
	GetSave(ctx context.Context, userID int64, game string) (*GameSave, error)
	PutSave(ctx context.Context, userID int64, game string, data json.RawMessage) (time.Time, error)
	BoardsOf(ctx context.Context, userID int64, game string) ([]string, error)
	// SubmitScore keeps the higher of the stored and the new score and returns the best one.
	SubmitScore(ctx context.Context, userID int64, game, board string, score int64, detail json.RawMessage) (best int64, improved bool, err error)
	// Leaderboard is ordered by score, then by who reached it first.
	Leaderboard(ctx context.Context, game, board string, limit int) ([]GameScoreRow, error)
	Players(ctx context.Context, game, board string) (int, error)
	// EntryFor returns the player's row with its rank, or nil when they have no score on the board.
	EntryFor(ctx context.Context, userID int64, game, board string) (*GameScoreRow, error)
}

type GameProgressService struct {
	repo GameProgressRepository
}

func NewGameProgressService(repo GameProgressRepository) *GameProgressService {
	return &GameProgressService{repo: repo}
}

func checkGame(game string) (int64, error) {
	max, ok := gameProgressGames[game]
	if !ok {
		return 0, ErrGameUnknown
	}
	return max, nil
}

func normBoard(board string) (string, error) {
	board = strings.TrimSpace(board)
	if board == "" {
		return "main", nil
	}
	if !gameBoardRe.MatchString(board) {
		return "", ErrGameBadInput
	}
	return board, nil
}

// jsonObject accepts only a JSON object no larger than max bytes.
func jsonObject(raw json.RawMessage, max int, tooBig error) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if len(raw) > max {
		return nil, tooBig
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, ErrGameBadInput
	}
	return raw, nil
}

func (s *GameProgressService) GetSave(ctx context.Context, userID int64, game string) (*GameSave, error) {
	if _, err := checkGame(game); err != nil {
		return nil, err
	}
	save, err := s.repo.GetSave(ctx, userID, game)
	if err != nil {
		return nil, err
	}
	if save == nil {
		return &GameSave{Data: json.RawMessage(`{}`)}, nil
	}
	return save, nil
}

func (s *GameProgressService) PutSave(ctx context.Context, userID int64, game string, data json.RawMessage) (*GameSave, error) {
	if _, err := checkGame(game); err != nil {
		return nil, err
	}
	data, err := jsonObject(data, gameSaveMaxBytes, ErrGameSaveSize)
	if err != nil {
		return nil, err
	}
	at, err := s.repo.PutSave(ctx, userID, game, data)
	if err != nil {
		return nil, err
	}
	return &GameSave{Data: data, UpdatedAt: &at}, nil
}

func (s *GameProgressService) SubmitScore(ctx context.Context, userID int64, game, board string, score int64, detail json.RawMessage) (*GameScoreResult, error) {
	max, err := checkGame(game)
	if err != nil {
		return nil, err
	}
	if board, err = normBoard(board); err != nil {
		return nil, err
	}
	if score < 0 || score > max {
		return nil, ErrGameBadInput
	}
	if detail, err = jsonObject(detail, gameScoreDetailMaxBytes, ErrGameBadInput); err != nil {
		return nil, err
	}
	boards, err := s.repo.BoardsOf(ctx, userID, game)
	if err != nil {
		return nil, err
	}
	if len(boards) >= gameBoardMaxPerGame && !slices.Contains(boards, board) {
		return nil, ErrGameBoards
	}
	best, improved, err := s.repo.SubmitScore(ctx, userID, game, board, score, detail)
	if err != nil {
		return nil, err
	}
	out := &GameScoreResult{Best: best, Improved: improved}
	if me, err := s.repo.EntryFor(ctx, userID, game, board); err == nil && me != nil {
		out.Rank = me.Rank
	}
	if players, err := s.repo.Players(ctx, game, board); err == nil {
		out.Players = players
	}
	return out, nil
}

func (s *GameProgressService) Leaderboard(ctx context.Context, game, board string, viewerID int64) (*GameLeaderboard, error) {
	if _, err := checkGame(game); err != nil {
		return nil, err
	}
	board, err := normBoard(board)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Leaderboard(ctx, game, board, gameLeaderboardSize)
	if err != nil {
		return nil, err
	}
	players, err := s.repo.Players(ctx, game, board)
	if err != nil {
		return nil, err
	}
	out := &GameLeaderboard{Game: game, Board: board, Players: players, Top: make([]GameLeaderboardEntry, 0, len(rows))}
	for _, r := range rows {
		out.Top = append(out.Top, publicGameEntry(r, viewerID))
	}
	if viewerID > 0 {
		me, err := s.repo.EntryFor(ctx, viewerID, game, board)
		if err != nil {
			return nil, err
		}
		if me != nil {
			e := publicGameEntry(*me, viewerID)
			out.Me = &e
		}
	}
	return out, nil
}

// publicGameEntry shows only a masked name, like the invite leaderboard.
func publicGameEntry(r GameScoreRow, viewerID int64) GameLeaderboardEntry {
	name := maskSegment(strings.TrimSpace(r.Username))
	if strings.TrimSpace(r.Username) == "" {
		name = maskEmail(r.Email)
	}
	detail := r.Detail
	if len(detail) == 0 || string(detail) == "{}" {
		detail = nil
	}
	return GameLeaderboardEntry{Rank: r.Rank, DisplayName: name, Score: r.Score, Detail: detail, IsCurrentUser: viewerID > 0 && r.UserID == viewerID}
}
