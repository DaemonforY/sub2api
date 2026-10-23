//go:build unit

package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCardKeys(t *testing.T) {
	keys, dup, err := parseCardKeys("  k1 \r\nk2\n\nk1\nuser----pass\n")
	require.NoError(t, err)
	require.Equal(t, []string{"k1", "k2", "user----pass"}, keys)
	require.Equal(t, 1, dup)

	_, _, err = parseCardKeys(" \n\n")
	require.ErrorIs(t, err, errCardGoodsNoKeys)

	_, _, err = parseCardKeys(strings.Repeat("x", cardGoodsMaxKeyLen+1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "超过")

	many := make([]string, cardGoodsMaxImport+1)
	for i := range many {
		many[i] = "key-" + strconv.Itoa(i)
	}
	_, _, err = parseCardKeys(strings.Join(many, "\n"))
	require.ErrorIs(t, err, errCardGoodsTooMany)
}

func TestCardKeyPreview(t *testing.T) {
	require.Equal(t, "MUSE…7890", cardKeyPreview("MUSE-ABCD-EFGH-7890"))
	require.Equal(t, "a…f", cardKeyPreview("abcdef"))
	require.Equal(t, "**", cardKeyPreview("ab"))
	require.Equal(t, "账号小明…码一二三", cardKeyPreview("账号小明----密码一二三"))
}

func TestNormalizeCardGoods(t *testing.T) {
	g, err := normalizeCardGoods(CardGoods{Slug: " Muse ", Name: " Muse 月卡 ", Price: 29.999, OriginalPrice: 0})
	require.NoError(t, err)
	require.Equal(t, "muse", g.Slug)
	require.Equal(t, "Muse 月卡", g.Name)
	require.Equal(t, 30.0, g.Price)

	for _, bad := range []CardGoods{
		{Slug: "", Name: "x", Price: 1},
		{Slug: "has space", Name: "x", Price: 1},
		{Slug: "-lead", Name: "x", Price: 1},
		{Slug: "ok", Name: "", Price: 1},
		{Slug: "ok", Name: "x", Price: 0},
		{Slug: "ok", Name: "x", Price: 1, OriginalPrice: -1},
		{Slug: "ok", Name: "x", Price: 1, PerUserLimit: -1},
		{Slug: "ok", Name: strings.Repeat("名", 61), Price: 1},
	} {
		_, err := normalizeCardGoods(bad)
		require.ErrorIs(t, err, errCardGoodsBadInput, "%+v", bad)
	}
}

func TestCardGoodsStockForSale(t *testing.T) {
	require.Equal(t, 3, CardGoodsStock{Available: 5, Reserved: 2}.ForSale())
	require.Equal(t, 0, CardGoodsStock{Available: 1, Reserved: 2}.ForSale())
}
