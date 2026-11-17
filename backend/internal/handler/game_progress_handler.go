package handler

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GameProgressHandler serves 小游戏可选登录 (/api/v1/games/progress/:game/*): cloud saves and best scores
// for signed-in players; the leaderboard is public.
type GameProgressHandler struct {
	svc *service.GameProgressService
}

func NewGameProgressHandler(svc *service.GameProgressService) *GameProgressHandler {
	return &GameProgressHandler{svc: svc}
}

func gameUser(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "登录后才能保存进度和上排行榜（Sign in to save progress）")
		return 0, false
	}
	return subject.UserID, true
}

// GetSave GET /games/progress/:game/save
func (h *GameProgressHandler) GetSave(c *gin.Context) {
	uid, ok := gameUser(c)
	if !ok {
		return
	}
	out, err := h.svc.GetSave(c.Request.Context(), uid, c.Param("game"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// PutSave PUT /games/progress/:game/save {data}
func (h *GameProgressHandler) PutSave(c *gin.Context) {
	uid, ok := gameUser(c)
	if !ok {
		return
	}
	var in struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrGameBadInput)
		return
	}
	out, err := h.svc.PutSave(c.Request.Context(), uid, c.Param("game"), in.Data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// SubmitScore POST /games/progress/:game/score {board, score, detail}
func (h *GameProgressHandler) SubmitScore(c *gin.Context) {
	uid, ok := gameUser(c)
	if !ok {
		return
	}
	var in struct {
		Board  string          `json:"board"`
		Score  int64           `json:"score"`
		Detail json.RawMessage `json:"detail"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrGameBadInput)
		return
	}
	out, err := h.svc.SubmitScore(c.Request.Context(), uid, c.Param("game"), in.Board, in.Score, in.Detail)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Leaderboard GET /games/progress/:game/leaderboard?board=
func (h *GameProgressHandler) Leaderboard(c *gin.Context) {
	var viewer int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		viewer = subject.UserID
	}
	out, err := h.svc.Leaderboard(c.Request.Context(), c.Param("game"), c.Query("board"), viewer)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
