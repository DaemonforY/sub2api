package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GameCraftHandler serves 小游戏「AI 万物合成」 (/api/v1/games/craft/*), open to visitors; limits are in
// the service.
type GameCraftHandler struct {
	svc *service.GameCraftService
}

func NewGameCraftHandler(svc *service.GameCraftService) *GameCraftHandler {
	return &GameCraftHandler{svc: svc}
}

// Start GET /games/craft/start
func (h *GameCraftHandler) Start(c *gin.Context) {
	out, err := h.svc.Start(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Combine POST /games/craft/combine {a, b}
func (h *GameCraftHandler) Combine(c *gin.Context) {
	var in struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrCraftInvalid)
		return
	}
	out, err := h.svc.Combine(c.Request.Context(), libVisitor(c), in.A, in.B)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
