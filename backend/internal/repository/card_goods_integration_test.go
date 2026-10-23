//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newGoodsOrder(t *testing.T, buyer *service.User, goodsID int64, status, tradeNo string) *dbent.PaymentOrder {
	t.Helper()
	b := integrationEntClient.PaymentOrder.Create().
		SetUserID(buyer.ID).SetUserEmail(buyer.Email).SetUserName(buyer.Username).
		SetAmount(29.9).SetPayAmount(29.9).SetFeeRate(0).SetRechargeCode("").
		SetOutTradeNo(tradeNo).SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeGoods).SetGoodsID(goodsID).SetStatus(status).
		SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("test")
	if status != service.OrderStatusPending {
		b.SetPaidAt(time.Now())
	}
	o, err := b.Save(context.Background())
	require.NoError(t, err)
	return o
}

func TestCardGoodsOrderFulfillment(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	buyer := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "goods-" + suffix + "@goods.test", Username: "gd" + suffix[len(suffix)-6:]}))
	other := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "goods2-" + suffix + "@goods.test", Username: "gx" + suffix[len(suffix)-6:]}))

	goods := service.NewCardGoodsService(NewCardGoodsRepository(integrationDB), reverseEncryptor{})
	payments := service.NewPaymentService(integrationEntClient, nil, nil, nil, nil, nil, nil, nil, nil)
	payments.SetCardGoodsService(goods)

	g, err := goods.Save(ctx, 0, service.CardGoods{Slug: "muse-" + suffix[len(suffix)-8:], Name: "Muse 月卡", Price: 29.9, UsageNote: "打开 Muse 输入激活码", PerUserLimit: 2, Enabled: true})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM card_goods WHERE id = $1`, g.ID) })

	res, err := goods.Import(ctx, g.ID, "KEY-AAAA-0001\n\n  KEY-AAAA-0002  \nKEY-AAAA-0001\n", "batch 1")
	require.NoError(t, err)
	require.Equal(t, service.CardGoodsImportResult{Added: 2, Duplicates: 1}, res)
	res, err = goods.Import(ctx, g.ID, "KEY-AAAA-0002", "")
	require.NoError(t, err)
	require.Equal(t, 0, res.Added, "a key already in stock is not imported twice")
	require.Equal(t, 1, res.Duplicates)

	var stored string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT content_enc FROM card_goods_items WHERE goods_id = $1 ORDER BY id LIMIT 1`, g.ID).Scan(&stored))
	require.NotContains(t, stored, "KEY-AAAA", "keys are stored encrypted")

	// A pending order of another user holds one of the two keys.
	newGoodsOrder(t, other, g.ID, service.OrderStatusPending, "gp"+suffix)
	_, err = goods.GoodsForOrder(ctx, buyer.ID, g.ID)
	require.NoError(t, err)
	page, err := goods.Product(ctx, g.Slug, buyer.ID)
	require.NoError(t, err)
	require.Equal(t, 1, page.Stock)

	paid := newGoodsOrder(t, buyer, g.ID, service.OrderStatusPaid, "g1"+suffix)
	_, err = goods.GoodsForOrder(ctx, buyer.ID, g.ID)
	require.ErrorIs(t, err, service.ErrCardGoodsSoldOut, "both keys are promised to open orders")

	require.NoError(t, payments.ExecuteGoodsFulfillment(ctx, paid.ID))
	require.NoError(t, payments.ExecuteGoodsFulfillment(ctx, paid.ID)) // idempotent
	done, err := integrationEntClient.PaymentOrder.Get(ctx, paid.ID)
	require.NoError(t, err)
	require.Equal(t, service.OrderStatusCompleted, done.Status)

	d, err := goods.OrderDelivery(ctx, buyer.ID, paid.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, "KEY-AAAA-0001", d.Content, "the oldest key goes first")
	require.Equal(t, "打开 Muse 输入激活码", d.UsageNote)
	none, err := goods.OrderDelivery(ctx, other.ID, paid.ID)
	require.NoError(t, err)
	require.Nil(t, none, "another user can't read the key")

	// The second paid order takes the last key; a third one finds the stock empty and waits.
	second := newGoodsOrder(t, buyer, g.ID, service.OrderStatusPaid, "g2"+suffix)
	require.NoError(t, payments.ExecuteGoodsFulfillment(ctx, second.ID))
	_, err = goods.GoodsForOrder(ctx, buyer.ID, g.ID)
	require.Error(t, err, "limit of 2 per user reached")
	require.Contains(t, err.Error(), "限购 2 件")

	late := newGoodsOrder(t, other, g.ID, service.OrderStatusPaid, "g3"+suffix)
	require.Error(t, payments.ExecuteGoodsFulfillment(ctx, late.ID))
	failed, err := integrationEntClient.PaymentOrder.Get(ctx, late.ID)
	require.NoError(t, err)
	require.Equal(t, service.OrderStatusFailed, failed.Status)
	require.Contains(t, *failed.FailedReason, "库存不足")

	waiting, err := goods.WaitingOrderIDs(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, []int64{late.ID}, waiting)
	_, err = goods.Import(ctx, g.ID, "KEY-BBBB-0003", "restock")
	require.NoError(t, err)
	require.NoError(t, payments.RetryFulfillment(ctx, late.ID))
	d, err = goods.OrderDelivery(ctx, other.ID, late.ID)
	require.NoError(t, err)
	require.Equal(t, "KEY-BBBB-0003", d.Content)

	mine, err := goods.Deliveries(ctx, buyer.ID, 0)
	require.NoError(t, err)
	require.Len(t, mine, 2)

	list, err := goods.AdminList(ctx)
	require.NoError(t, err)
	var row *service.CardGoodsAdminRow
	for i := range list {
		if list[i].ID == g.ID {
			row = &list[i]
		}
	}
	require.NotNil(t, row)
	require.Equal(t, service.CardGoodsStock{Available: 0, Sold: 3, Reserved: 1}, row.Stock, "the other user's pending order still holds a key")
	require.Zero(t, row.Waiting)
	require.ErrorIs(t, goods.Delete(ctx, g.ID), service.ErrCardGoodsHasSales)

	items, total, err := goods.Items(ctx, g.ID, service.CardGoodsItemSold, 1, 50)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, "KEY-…0003", items[0].Preview)
	require.Equal(t, other.Email, items[0].UserEmail)
}

func TestCardGoodsVoidItems(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	goods := service.NewCardGoodsService(NewCardGoodsRepository(integrationDB), reverseEncryptor{})
	g, err := goods.Save(ctx, 0, service.CardGoods{Slug: "void-" + suffix[len(suffix)-8:], Name: "Void test", Price: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM card_goods WHERE id = $1`, g.ID) })

	_, err = goods.Import(ctx, g.ID, "a1\na2\na3", "")
	require.NoError(t, err)
	items, _, err := goods.Items(ctx, g.ID, "", 1, 50)
	require.NoError(t, err)
	require.Len(t, items, 3)
	n, err := goods.VoidItems(ctx, g.ID, []int64{items[0].ID, items[1].ID})
	require.NoError(t, err)
	require.Equal(t, 2, n)

	_, err = goods.Product(ctx, g.Slug, 0)
	require.ErrorIs(t, err, service.ErrCardGoodsNotFound, "a product not enabled is hidden")
	_, err = goods.Save(ctx, 0, service.CardGoods{Slug: g.Slug, Name: "dup", Price: 1})
	require.ErrorIs(t, err, service.ErrCardGoodsSlugTaken)

	g.Enabled = true
	_, err = goods.Save(ctx, g.ID, *g)
	require.NoError(t, err)
	page, err := goods.Product(ctx, g.Slug, 0)
	require.NoError(t, err)
	require.Equal(t, 1, page.Stock)
	require.NoError(t, goods.Delete(ctx, g.ID), "nothing sold, so it can be deleted")
}
