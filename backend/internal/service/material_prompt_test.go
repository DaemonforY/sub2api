//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func materialPromptFixture() (*MaterialSegmentContext, []MaterialSegmentImagery, map[int64]MaterialImagery, []MaterialStyle) {
	seg := &MaterialSegmentContext{
		MaterialSegment: MaterialSegment{ID: 1, Label: "第3联", Content: "大漠孤烟直，长河落日圆。",
			SceneDesc: "黄昏的大漠上，一缕烟直直升起，远处长河映着落日。"},
		Kind: "shi", Title: "使至塞上", AuthorName: "王维", Dynasty: "唐", LicenseStatus: MaterialLicensePublicDomain,
	}
	byID := map[int64]MaterialImagery{
		1: {ID: 1, Name: "大漠", Category: "地理", VisualEn: "an endless desert of rolling dunes", Emotions: []string{"壮阔", "孤寂"}, Status: MaterialStatusPublished},
		2: {ID: 2, Name: "落日", Category: "天象", VisualEn: "a huge round setting sun low on the horizon.", Emotions: []string{"壮阔"}, Status: MaterialStatusPublished, StyleIDs: []int64{30}},
		3: {ID: 3, Name: "烟", Category: "气候", VisualZh: "一缕笔直的烟", Emotions: []string{"孤寂"}, Status: MaterialStatusDraft},
		4: {ID: 4, Name: "旧", Category: "器物", VisualEn: "old thing", Status: MaterialStatusHidden},
	}
	links := []MaterialSegmentImagery{
		{SegmentID: 1, ImageryID: 3, Weight: 2, Source: "ai"},
		{SegmentID: 1, ImageryID: 1, Weight: 3, Source: "ai"},
		{SegmentID: 1, ImageryID: 2, Weight: 3, Source: "ai"},
		{SegmentID: 1, ImageryID: 4, Weight: 1, Source: "ai"},
	}
	styles := []MaterialStyle{
		{ID: 10, Name: "工笔青花", PromptEn: "Gongbi in blue and white.", SuitsEmotions: []string{"思念"}, DefaultSize: "1024x1536", DefaultQuality: "high", Enabled: true},
		{ID: 20, Name: "水墨", PromptEn: "Chinese ink-wash painting.", NegativeEn: "photorealism", SuitsEmotions: []string{"孤寂", "壮阔"}, SuitsCategories: []string{"地理"},
			DefaultSize: "1024x1536", DefaultQuality: "high", Enabled: true, Options: json.RawMessage(`{"moods":[{"name":"大漠黄昏","prompt_en":"Dusty amber dusk over the desert."}]}`)},
		{ID: 30, Name: "版画", PromptEn: "Woodblock print.", SuitsEmotions: []string{"质朴"}, DefaultSize: "1536x1024", DefaultQuality: "medium", Enabled: true},
		{ID: 40, Name: "停用", PromptEn: "x", SuitsEmotions: []string{"壮阔", "孤寂"}, Enabled: false},
	}
	return seg, links, byID, styles
}

func TestRecommendMaterialStyles(t *testing.T) {
	_, links, byID, styles := materialPromptFixture()
	use, confirmed := materialImageryOf(links, byID)
	require.False(t, confirmed)
	require.Len(t, use, 3, "hidden imagery is left out")
	require.Equal(t, 3, use[0].Weight)

	got := RecommendMaterialStyles(styles, use, byID)
	require.Len(t, got, 3, "disabled styles are left out")
	// 水墨: 大漠 壮阔+孤寂 (3+3) + category 地理 (1.5) + 落日 壮阔 (3) + 烟 孤寂 (2) = 12.5
	require.Equal(t, "水墨", got[0].Style.Name)
	require.InDelta(t, 12.5, got[0].Score, 0.001)
	require.Contains(t, got[0].Reasons, "情绪「壮阔」")
	// 版画: 落日 recommends it, 3 × 3 = 9
	require.Equal(t, "版画", got[1].Style.Name)
	require.InDelta(t, 9, got[1].Score, 0.001)
	require.Contains(t, got[1].Reasons, "「落日」推荐此画风")
	require.Equal(t, "工笔青花", got[2].Style.Name)
	require.Zero(t, got[2].Score)
	require.Equal(t, []string{}, got[2].Reasons)
	// A style's own options come before the defaults.
	require.Equal(t, "大漠黄昏", got[0].Options.Moods[0].Name)
	require.Equal(t, len(DefaultMaterialStyleOptions.Moods)+1, len(got[0].Options.Moods))

	// Once any tag is confirmed, only confirmed tags count.
	links[1].Confirmed = true
	use, confirmed = materialImageryOf(links, byID)
	require.True(t, confirmed)
	require.Len(t, use, 1)
	require.Equal(t, int64(1), use[0].ImageryID)
}

func TestAssembleMaterialPrompt(t *testing.T) {
	seg, links, byID, styles := materialPromptFixture()
	ink := styles[1]
	p, err := AssembleMaterialPrompt(seg, links, byID, ink, MaterialPromptInput{StyleID: 20, Composition: "远景小人",
		Palette: "墨色一点红", Mood: "大漠黄昏", Extra: "A lone rider far away on the left."}, nil)
	require.NoError(t, err)
	require.False(t, p.Blocked)
	require.Equal(t, "1024x1536", p.Size, "the style's default size")
	require.Equal(t, "high", p.Quality)
	require.True(t, p.TextOverlayAllowed)
	require.Equal(t, []int64{1, 2, 3}, p.ImageryIDs)

	lines := strings.Split(p.Prompt, "\n")
	require.Equal(t, materialFrames["1024x1536"], lines[0])
	require.Equal(t, "Vast landscape with a small distant figure, the figure small against the scenery.", lines[1])
	require.Equal(t, "Scene: 黄昏的大漠上，一缕烟直直升起，远处长河映着落日。", lines[2])
	require.Contains(t, lines[3], "Tang dynasty")
	require.Equal(t, "Main subject: an endless desert of rolling dunes.", lines[4])
	require.Equal(t, "Main subject: a huge round setting sun low on the horizon.", lines[5])
	require.Equal(t, "Supporting element: 一缕笔直的烟.", lines[6], "falls back to the Chinese visual")
	require.Equal(t, "Dusty amber dusk over the desert.", lines[7])
	require.Equal(t, "Style: Chinese ink-wash painting. "+materialStyleRule, lines[8])
	require.Equal(t, "Avoid: photorealism", lines[9])
	require.True(t, strings.HasPrefix(lines[10], "Palette: Black and grey ink"))
	require.Equal(t, "A lone rider far away on the left.", lines[11])
	require.Equal(t, materialNoText, lines[12])
	require.Len(t, lines, 13)
	require.Equal(t, lines[4:7], p.Parts.Imagery)

	require.Len(t, p.Notes, 2)
	require.Contains(t, p.Notes[0], "还没人工确认")
	require.Contains(t, p.Notes[1], "烟")
	require.NotContains(t, p.Prompt, "old thing")
}

func TestAssembleMaterialPromptChoices(t *testing.T) {
	seg, links, byID, styles := materialPromptFixture()

	// The admin's own pick: untagged imagery gets weight 2, hidden is skipped, order by weight.
	p, err := AssembleMaterialPrompt(seg, links, byID, styles[2], MaterialPromptInput{ImageryIDs: []int64{3, 2, 4, 2}}, nil)
	require.NoError(t, err)
	require.Equal(t, "1536x1024", p.Size)
	require.Equal(t, "medium", p.Quality)
	require.Equal(t, []int64{2, 3}, p.ImageryIDs)
	require.NotContains(t, strings.Join(p.Notes, ""), "还没人工确认")

	// Unknown sizes and option names are rejected.
	_, err = AssembleMaterialPrompt(seg, links, byID, styles[1], MaterialPromptInput{Size: "512x512"}, nil)
	require.ErrorIs(t, err, ErrMaterialInvalid)
	_, err = AssembleMaterialPrompt(seg, links, byID, styles[1], MaterialPromptInput{Mood: "不存在"}, nil)
	require.ErrorIs(t, err, ErrMaterialInvalid)

	// No scene and no imagery: nothing to draw.
	empty := *seg
	empty.SceneDesc = ""
	_, err = AssembleMaterialPrompt(&empty, nil, byID, styles[1], MaterialPromptInput{}, nil)
	require.Error(t, err)

	// A modern text gets no period line; reference_only text may not be set on the picture.
	modern := *seg
	modern.Dynasty, modern.LicenseStatus = "现代", MaterialLicenseReference
	p, err = AssembleMaterialPrompt(&modern, links, byID, styles[1], MaterialPromptInput{}, nil)
	require.NoError(t, err)
	require.Empty(t, p.Parts.Period)
	require.False(t, p.TextOverlayAllowed)

	// Overlong extras are cut, overlong prompts refused.
	p, err = AssembleMaterialPrompt(seg, links, byID, styles[1], MaterialPromptInput{Extra: strings.Repeat("长", 700)}, nil)
	require.NoError(t, err)
	require.Equal(t, materialMaxExtraRunes, len([]rune(p.Parts.Extra)))
	big := *seg
	big.SceneDesc = strings.Repeat("景", materialMaxPromptRunes)
	_, err = AssembleMaterialPrompt(&big, links, byID, styles[1], MaterialPromptInput{}, nil)
	require.Error(t, err)
}

func TestAssembleMaterialPromptBlocklist(t *testing.T) {
	seg, links, byID, styles := materialPromptFixture()
	terms := []MaterialBlockTerm{
		{Term: "(?i)ghibli", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true},
		{Term: "(?i)\\bcaption\\b", Kind: "text", Match: "regex", Action: MaterialBlockActionWarn, AppliesTo: "prompt", Enabled: true},
		{Term: "(?i),? ?trending on artstation", Kind: "text", Match: "regex", Action: MaterialBlockActionStrip, AppliesTo: "prompt", Enabled: true},
		// The fixed no-text rule mentions calligraphy; it must never trip the blocklist.
		{Term: "(?i)calligraphy", Kind: "text", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true},
	}

	p, err := AssembleMaterialPrompt(seg, links, byID, styles[1], MaterialPromptInput{Extra: "Add a caption, trending on artstation"}, terms)
	require.NoError(t, err)
	require.False(t, p.Blocked)
	require.Len(t, p.Hits, 2)
	require.Equal(t, "Add a caption", p.Parts.Extra)
	require.NotContains(t, p.Prompt, "artstation")
	require.Contains(t, p.Prompt, "calligraphy", "the no-text rule is still there")
	require.Contains(t, strings.Join(p.Notes, ""), "已自动删掉")

	p, err = AssembleMaterialPrompt(seg, links, byID, styles[1], MaterialPromptInput{Extra: "in Ghibli style"}, terms)
	require.NoError(t, err)
	require.True(t, p.Blocked)
	require.Equal(t, "studio", p.Hits[0].Kind)
}

func TestMaterialServiceBuildPrompt(t *testing.T) {
	seg, links, byID, styles := materialPromptFixture()
	repo := &materialPromptRepo{materialMemRepo: newMaterialMemRepo(), styles: styles}
	for id := int64(1); id <= 4; id++ {
		repo.imagery = append(repo.imagery, byID[id])
	}
	repo.segments[1] = seg
	repo.links[1] = links
	svc := &MaterialService{repo: repo}

	recs, err := svc.RecommendStyles(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "水墨", recs[0].Style.Name)

	p, err := svc.BuildPrompt(context.Background(), 1, MaterialPromptInput{StyleID: 20})
	require.NoError(t, err)
	require.Contains(t, p.Prompt, "Chinese ink-wash painting.")

	_, err = svc.BuildPrompt(context.Background(), 1, MaterialPromptInput{StyleID: 40})
	require.Error(t, err, "disabled style")
	_, err = svc.BuildPrompt(context.Background(), 99, MaterialPromptInput{StyleID: 20})
	require.ErrorIs(t, err, ErrMaterialSegmentNotFound)
}

type materialPromptRepo struct {
	*materialMemRepo
	styles []MaterialStyle
}

func (r *materialPromptRepo) ListStyles(_ context.Context, includeDisabled bool) ([]MaterialStyle, error) {
	out := []MaterialStyle{}
	for _, s := range r.styles {
		if s.Enabled || includeDisabled {
			out = append(out, s)
		}
	}
	return out, nil
}

func TestMaterialPeriod(t *testing.T) {
	require.Contains(t, materialPeriod("唐"), "China in the Tang dynasty;")
	require.Contains(t, materialPeriod("南宋"), "Southern Song dynasty")
	require.Contains(t, materialPeriod("齐"), "ancient China (齐)")
	require.Empty(t, materialPeriod("现代"))
	for d := range materialAncientDynasties {
		require.NotEmpty(t, materialPeriod(d), d)
	}
}
