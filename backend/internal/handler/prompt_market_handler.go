package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PromptMarketHandler serves 提示词交易 to the canvas (signed in with the main-site account) and the admin.
type PromptMarketHandler struct {
	market *service.PromptMarketService
}

func NewPromptMarketHandler(market *service.PromptMarketService) *PromptMarketHandler {
	return &PromptMarketHandler{market: market}
}

// optionalViewer is the signed-in canvas user, 0 when signed out.
func optionalViewer(c *gin.Context) int64 {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	return 0
}

func marketPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrPromptListingNotFound)
		return 0, false
	}
	return id, true
}

func marketListQuery(c *gin.Context) service.PromptMarketListQuery {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	return service.PromptMarketListQuery{Keyword: c.Query("q"), Sort: c.Query("sort"), Status: c.Query("status"), Page: page, PageSize: size}
}

// Config GET /api/v1/canvas/market/config — whether selling is open, the fee and the price range.
func (h *PromptMarketHandler) Config(c *gin.Context) {
	response.Success(c, h.market.Config(c.Request.Context()))
}

// List GET /api/v1/canvas/market/listings?q=&sort=&page=
func (h *PromptMarketHandler) List(c *gin.Context) {
	items, total, err := h.market.ListPublic(c.Request.Context(), optionalViewer(c), marketListQuery(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// Get GET /api/v1/canvas/market/listings/:id
func (h *PromptMarketHandler) Get(c *gin.Context) {
	id, ok := marketPathID(c)
	if !ok {
		return
	}
	l, err := h.market.Get(c.Request.Context(), optionalViewer(c), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, l)
}

// Purchase POST /api/v1/canvas/market/listings/:id/purchase  {price}
func (h *PromptMarketHandler) Purchase(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := marketPathID(c)
	if !ok {
		return
	}
	var in struct {
		Price float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return
	}
	l, err := h.market.Purchase(c.Request.Context(), userID, id, in.Price)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, l)
}

// Purchases GET /api/v1/canvas/market/purchases — what the user bought, with the full prompts.
func (h *PromptMarketHandler) Purchases(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	items, err := h.market.Purchases(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

// Selling GET /api/v1/canvas/market/selling — the user's listings and earnings.
func (h *PromptMarketHandler) Selling(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	items, summary, err := h.market.Selling(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "summary": summary})
}

func bindListingInput(c *gin.Context) (service.PromptListingInput, bool) {
	var in service.PromptListingInput
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "请求格式不正确（Invalid request）")
		return in, false
	}
	return in, true
}

// CreateListing POST /api/v1/canvas/market/selling
func (h *PromptMarketHandler) CreateListing(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	in, ok := bindListingInput(c)
	if !ok {
		return
	}
	l, err := h.market.CreateListing(c.Request.Context(), userID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, l)
}

// UpdateListing PUT /api/v1/canvas/market/selling/:id
func (h *PromptMarketHandler) UpdateListing(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := marketPathID(c)
	if !ok {
		return
	}
	in, ok := bindListingInput(c)
	if !ok {
		return
	}
	l, err := h.market.UpdateListing(c.Request.Context(), userID, id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, l)
}

// SetListed POST /api/v1/canvas/market/selling/:id/withdraw | /relist
func (h *PromptMarketHandler) SetListed(listed bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := appStateUserID(c)
		if !ok {
			return
		}
		id, ok := marketPathID(c)
		if !ok {
			return
		}
		l, err := h.market.SetListed(c.Request.Context(), userID, id, listed)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, l)
	}
}

// --- admin ---

// AdminGetConfig GET /api/v1/admin/prompt-market/config
func (h *PromptMarketHandler) AdminGetConfig(c *gin.Context) {
	response.Success(c, h.market.Config(c.Request.Context()))
}

// AdminSaveConfig PUT /api/v1/admin/prompt-market/config
func (h *PromptMarketHandler) AdminSaveConfig(c *gin.Context) {
	var cfg service.PromptMarketConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	saved, err := h.market.SaveConfig(c.Request.Context(), cfg)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, saved)
}

// AdminList GET /api/v1/admin/prompt-market/listings?status=review|pending|active|…&q=&page=
func (h *PromptMarketHandler) AdminList(c *gin.Context) {
	items, total, err := h.market.AdminList(c.Request.Context(), marketListQuery(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

type marketNoteRequest struct {
	Note string `json:"note"`
}

// AdminReview POST /api/v1/admin/prompt-market/listings/:id/{approve,reject,hide,restore}  {note}
func (h *PromptMarketHandler) AdminReview(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := marketPathID(c)
		if !ok {
			return
		}
		var in marketNoteRequest
		_ = c.ShouldBindJSON(&in)
		ctx := c.Request.Context()
		var (
			l   *service.PromptListing
			err error
		)
		switch action {
		case "approve":
			l, err = h.market.Approve(ctx, id)
		case "reject":
			l, err = h.market.Reject(ctx, id, in.Note)
		case "hide":
			l, err = h.market.SetHidden(ctx, id, true, in.Note)
		default:
			l, err = h.market.SetHidden(ctx, id, false, "")
		}
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, l)
	}
}

// AdminPurchases GET /api/v1/admin/prompt-market/purchases?status=&page=
func (h *PromptMarketHandler) AdminPurchases(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	items, total, err := h.market.AdminPurchases(c.Request.Context(), service.PromptPurchaseQuery{Status: c.Query("status"), Page: page, PageSize: size})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// AdminRefund POST /api/v1/admin/prompt-market/purchases/:id/refund
func (h *PromptMarketHandler) AdminRefund(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrPromptPurchaseNotFound)
		return
	}
	p, err := h.market.Refund(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}
