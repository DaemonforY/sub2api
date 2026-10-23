package handler

import (
	"log/slog"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CardGoodsHandler serves 卡密商品: the product page, the buyer's keys and the admin's products and
// stock (see service.CardGoodsService).
type CardGoodsHandler struct {
	goods    *service.CardGoodsService
	payments *service.PaymentService
}

func NewCardGoodsHandler(goods *service.CardGoodsService, payments *service.PaymentService) *CardGoodsHandler {
	return &CardGoodsHandler{goods: goods, payments: payments}
}

func cardGoodsIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}

// Product GET /api/v1/goods/:slug — the product page (a signed-in viewer also gets their keys).
func (h *CardGoodsHandler) Product(c *gin.Context) {
	var viewer int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		viewer = subject.UserID
	}
	p, err := h.goods.Product(c.Request.Context(), c.Param("slug"), viewer)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

// Deliveries GET /api/v1/user/goods/deliveries — every key the user bought.
func (h *CardGoodsHandler) Deliveries(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.goods.Deliveries(c.Request.Context(), subject.UserID, 0)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// OrderDelivery GET /api/v1/user/goods/orders/:id/delivery — the key of one order (null until delivered).
func (h *CardGoodsHandler) OrderDelivery(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, ok := cardGoodsIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.goods.OrderDelivery(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"delivery": d})
}

// --- Admin ---

// AdminList GET /api/v1/admin/goods
func (h *CardGoodsHandler) AdminList(c *gin.Context) {
	list, err := h.goods.AdminList(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

func (h *CardGoodsHandler) save(c *gin.Context, id int64) {
	var in service.CardGoods
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	g, err := h.goods.Save(c.Request.Context(), id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, g)
}

// AdminCreate POST /api/v1/admin/goods
func (h *CardGoodsHandler) AdminCreate(c *gin.Context) {
	h.save(c, 0)
}

// AdminUpdate PUT /api/v1/admin/goods/:id
func (h *CardGoodsHandler) AdminUpdate(c *gin.Context) {
	if id, ok := cardGoodsIDParam(c, "id"); ok {
		h.save(c, id)
	}
}

// AdminDelete DELETE /api/v1/admin/goods/:id
func (h *CardGoodsHandler) AdminDelete(c *gin.Context) {
	id, ok := cardGoodsIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.goods.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// AdminImport POST /api/v1/admin/goods/:id/import — adds keys (one per line), then delivers the paid
// orders that were waiting for stock.
func (h *CardGoodsHandler) AdminImport(c *gin.Context) {
	id, ok := cardGoodsIDParam(c, "id")
	if !ok {
		return
	}
	var in struct {
		Keys      string `json:"keys"`
		BatchNote string `json:"batch_note"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := c.Request.Context()
	res, err := h.goods.Import(ctx, id, in.Keys, in.BatchNote)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if res.Added > 0 && h.payments != nil {
		waiting, err := h.goods.WaitingOrderIDs(ctx, id)
		if err != nil {
			slog.Warn("card goods: list waiting orders failed", "goods_id", id, "error", err)
		}
		for _, oid := range waiting {
			if err := h.payments.RetryFulfillment(ctx, oid); err != nil {
				slog.Warn("card goods: deliver waiting order failed", "goods_id", id, "order_id", oid, "error", err)
				continue
			}
			res.Retried++
		}
	}
	response.Success(c, res)
}

// AdminItems GET /api/v1/admin/goods/:id/items?status=&page=&page_size=
func (h *CardGoodsHandler) AdminItems(c *gin.Context) {
	id, ok := cardGoodsIDParam(c, "id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	items, total, err := h.goods.Items(c.Request.Context(), id, c.Query("status"), page, size)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// AdminVoidItems POST /api/v1/admin/goods/:id/items/void — removes keys still in stock.
func (h *CardGoodsHandler) AdminVoidItems(c *gin.Context) {
	id, ok := cardGoodsIDParam(c, "id")
	if !ok {
		return
	}
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	n, err := h.goods.VoidItems(c.Request.Context(), id, in.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"voided": n})
}
