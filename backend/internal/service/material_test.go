//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSuggestMaterialLicense(t *testing.T) {
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	year := func(y int) *int { return &y }

	l, _ := SuggestMaterialLicense("唐", nil, now)
	require.Equal(t, MaterialLicensePublicDomain, l)
	l, _ = SuggestMaterialLicense("现代", nil, now)
	require.Equal(t, MaterialLicenseReference, l)
	// Protected to 31 December of the 50th year after death.
	l, _ = SuggestMaterialLicense("现代", year(1931), now)
	require.Equal(t, MaterialLicensePublicDomain, l)
	l, reason := SuggestMaterialLicense("当代", year(1989), now)
	require.Equal(t, MaterialLicenseReference, l)
	require.Contains(t, reason, "2040")
	l, _ = SuggestMaterialLicense("现代", year(1975), now)
	require.Equal(t, MaterialLicensePublicDomain, l)
	l, _ = SuggestMaterialLicense("现代", year(1976), now)
	require.Equal(t, MaterialLicenseReference, l)

	require.True(t, MaterialTextOverlayAllowed(MaterialLicenseOriginal))
	require.False(t, MaterialTextOverlayAllowed(MaterialLicenseReference))
	require.False(t, MaterialTextOverlayAllowed(""))
}

func TestSplitMaterialSegments(t *testing.T) {
	shi := SplitMaterialSegments("shi", "床前明月光，疑是地上霜。\r\n举头望明月，低头思故乡。", "举头望明月，低头思故乡")
	require.Len(t, shi, 2)
	require.Equal(t, "第1联", shi[0].Label)
	require.Equal(t, "床前明月光，疑是地上霜。", shi[0].Content)
	require.False(t, shi[0].IsFamous)
	require.True(t, shi[1].IsFamous)
	require.Equal(t, 2, shi[1].Seq)

	// A 词 in two paragraphs gets 上阕 / 下阕.
	ci := SplitMaterialSegments("ci", "明月几时有？把酒问青天。\n\n转朱阁，低绮户，照无眠。", "")
	require.Len(t, ci, 3)
	require.Equal(t, "上阕·第1句", ci[0].Label)
	require.Equal(t, "上阕·第2句", ci[1].Label)
	require.Equal(t, "下阕·第1句", ci[2].Label)

	// Closing quotes stay with their sentence; a tail without an end mark is kept.
	wen := SplitMaterialSegments("wen", "曰：“何陋之有？”孔子云", "")
	require.Len(t, wen, 2)
	require.Equal(t, "曰：“何陋之有？”", wen[0].Content)
	require.Equal(t, "孔子云", wen[1].Content)
	require.Equal(t, "第2句", wen[1].Label)

	require.Empty(t, SplitMaterialSegments("shi", "  \n\n ", ""))
}

func TestCheckMaterialBlocklist(t *testing.T) {
	terms := []MaterialBlockTerm{
		{Term: "(?i)ghibli", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true},
		{Term: "宫崎骏", Kind: "artist", Match: "contains", Action: MaterialBlockActionBlock, AppliesTo: "both", Enabled: true},
		{Term: "(?i)\\bcaptions?\\b", Kind: "text", Match: "regex", Action: MaterialBlockActionWarn, AppliesTo: "prompt", Enabled: true},
		{Term: "(?i)trending on \\w+", Kind: "text", Match: "regex", Action: MaterialBlockActionStrip, AppliesTo: "prompt", Enabled: true},
		{Term: "(?i)disabled", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: false},
		{Term: "([", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true},
	}

	res := CheckMaterialBlocklist("rice fields at dusk, trending on artstation, with a caption", "prompt", terms)
	require.False(t, res.Blocked)
	require.Len(t, res.Hits, 2)
	require.Equal(t, MaterialBlockActionWarn, res.Hits[0].Action)
	require.Equal(t, "rice fields at dusk, , with a caption", res.Prompt)

	res = CheckMaterialBlocklist("A GHIBLI-like village, disabled term", "prompt", terms)
	require.True(t, res.Blocked)
	require.Equal(t, "GHIBLI", res.Hits[0].Match)

	// "prompt"-only terms are not applied to texts; "both" terms are.
	res = CheckMaterialBlocklist("ghibli 宫崎骏", "text", terms)
	require.True(t, res.Blocked)
	require.Len(t, res.Hits, 1)
	require.Equal(t, "宫崎骏", res.Hits[0].Term)
}

// materialFakeShihua serves ListWorks; the other ShihuaRepository methods are never called.
type materialFakeShihua struct {
	ShihuaRepository
	works []ShihuaWork
	query ShihuaWorkQuery
}

func (f *materialFakeShihua) ListWorks(_ context.Context, q ShihuaWorkQuery) ([]ShihuaWork, error) {
	f.query = q
	return f.works, nil
}

type materialFakeRepo struct {
	MaterialRepository
	authors  map[string]int64
	texts    map[int64]*MaterialText
	segments map[int64][]MaterialSegment
}

func (f *materialFakeRepo) UpsertAuthor(_ context.Context, a *MaterialAuthor) (int64, error) {
	key := a.Name + "|" + a.Dynasty
	if id, ok := f.authors[key]; ok {
		return id, nil
	}
	f.authors[key] = int64(len(f.authors) + 1)
	return f.authors[key], nil
}

func (f *materialFakeRepo) ImportText(_ context.Context, t *MaterialText, segments []MaterialSegment) (bool, error) {
	if _, ok := f.texts[*t.ShihuaWorkID]; ok {
		return false, nil
	}
	f.texts[*t.ShihuaWorkID], f.segments[*t.ShihuaWorkID] = t, segments
	return true, nil
}

func TestMaterialImportShihua(t *testing.T) {
	by := int64(7)
	at := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	shihua := &materialFakeShihua{works: []ShihuaWork{
		{ID: 1, Slug: "shi-zhi-saishang", Title: "使至塞上", Author: "王维", Dynasty: "唐", Genre: "shi",
			FullText: "单车欲问边，属国过居延。征蓬出汉塞，归雁入胡天。", FamousLines: "征蓬出汉塞，归雁入胡天",
			Sources: "维基文库", Verify1By: &by, Verify1At: &at, Verify2By: &by, Verify2At: &at,
			Textbooks: []ShihuaTextbook{{Edition: "统编", Grade: "八上"}}},
		{ID: 2, Slug: "modern", Title: "现代诗", Author: "某人", Dynasty: "现代", Genre: "shi", FullText: "测试。"},
		{ID: 3, Slug: "empty", Title: "空篇", Author: "李白", Dynasty: "唐", Genre: "shi", FullText: " "},
		{ID: 4, Slug: "odd", Title: "未知体裁", Author: "李白", Dynasty: "唐", Genre: "xyz", FullText: "测试。"},
	}}
	repo := &materialFakeRepo{authors: map[string]int64{}, texts: map[int64]*MaterialText{}, segments: map[int64][]MaterialSegment{}}
	svc := NewMaterialService(repo, shihua, nil)

	res, err := svc.ImportShihua(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, ShihuaTextVerified2, shihua.query.Status)
	require.Equal(t, 1, res.Imported)
	require.Len(t, res.Skipped, 3)
	require.Contains(t, res.Skipped[0].Reason, "卒年")

	got := repo.texts[1]
	require.Equal(t, MaterialLicensePublicDomain, got.LicenseStatus)
	require.Equal(t, MaterialStatusPublished, got.Status)
	require.Equal(t, &by, got.Verify2By)
	require.Equal(t, int64(9), *got.CreatedBy)
	require.Contains(t, got.Sources, "维基文库")
	require.Contains(t, got.Sources, "课本诗画 #1")
	require.JSONEq(t, `[{"edition":"统编","grade":"八上"}]`, string(got.Textbook))
	require.Len(t, repo.segments[1], 2)
	require.True(t, repo.segments[1][1].IsFamous)

	// Running it again only counts what is already there.
	res, err = svc.ImportShihua(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, 0, res.Imported)
	require.Equal(t, 1, res.Existing)
}
