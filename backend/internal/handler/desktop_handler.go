package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// DesktopHandler serves HiveGPT 桌面助手 sign-in (see service/desktop_login.go) and its skills (service/desktop_skills.go):
// the app starts and polls without a session; the phone page reads and decides while signed in.
type DesktopHandler struct {
	login  *service.DesktopLoginService
	skills *service.DesktopSkillService
}

func NewDesktopHandler(login *service.DesktopLoginService, skills *service.DesktopSkillService) *DesktopHandler {
	return &DesktopHandler{login: login, skills: skills}
}

type desktopStartRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	DeviceName string `json:"device_name"`
}

// Start creates a sign-in request for the app.
// POST /api/v1/auth/desktop/start
func (h *DesktopHandler) Start(c *gin.Context) {
	var req desktopStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopLoginBadDevice)
		return
	}
	started, err := h.login.Start(c.Request.Context(), strings.TrimSpace(req.DeviceID), req.DeviceName)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, started)
}

type desktopPollRequest struct {
	TicketID   string `json:"ticket_id" binding:"required"`
	DeviceCode string `json:"device_code" binding:"required"`
}

// Poll returns pending / approved / rejected / expired, or done with the device key.
// POST /api/v1/auth/desktop/poll (POST so the device code stays out of URLs and access logs)
func (h *DesktopHandler) Poll(c *gin.Context) {
	var req desktopPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopLoginNotFound)
		return
	}
	res, err := h.login.Poll(c.Request.Context(), strings.TrimSpace(req.TicketID), strings.TrimSpace(req.DeviceCode))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, res)
}

// Info shows the signed-in phone which computer is asking.
// GET /api/v1/desktop/login?t=
func (h *DesktopHandler) Info(c *gin.Context) {
	info, err := h.login.Info(c.Request.Context(), strings.TrimSpace(c.Query("t")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, info)
}

type desktopDecideRequest struct {
	TicketID string `json:"ticket_id" binding:"required"`
	Approve  bool   `json:"approve"`
}

// Decide approves or rejects the request for the signed-in user.
// POST /api/v1/desktop/login/decide
func (h *DesktopHandler) Decide(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req desktopDecideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopLoginNotFound)
		return
	}
	status, err := h.login.Decide(c.Request.Context(), strings.TrimSpace(req.TicketID), subject.UserID, req.Approve)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": status})
}
