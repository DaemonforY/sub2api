//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// materialMemRepo keeps imagery, segments and tags in memory.
type materialMemRepo struct {
	MaterialRepository
	imagery  []MaterialImagery
	segments map[int64]*MaterialSegmentContext
	links    map[int64][]MaterialSegmentImagery
	terms    []MaterialBlockTerm
}

func newMaterialMemRepo() *materialMemRepo {
	return &materialMemRepo{segments: map[int64]*MaterialSegmentContext{}, links: map[int64][]MaterialSegmentImagery{}}
}

func (r *materialMemRepo) AllImagery(context.Context) ([]MaterialImagery, error) {
	return append([]MaterialImagery(nil), r.imagery...), nil
}

func (r *materialMemRepo) CreateImagery(_ context.Context, im *MaterialImagery) (bool, error) {
	for _, x := range r.imagery {
		if x.Name == im.Name {
			im.ID = 0
			return false, nil
		}
	}
	im.ID = int64(len(r.imagery) + 1)
	r.imagery = append(r.imagery, *im)
	return true, nil
}

func (r *materialMemRepo) ListBlocklist(context.Context, bool) ([]MaterialBlockTerm, error) {
	return r.terms, nil
}

func (r *materialMemRepo) GetSegment(_ context.Context, id int64) (*MaterialSegmentContext, error) {
	if s, ok := r.segments[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, nil
}

func (r *materialMemRepo) UpdateSegmentScene(_ context.Context, id int64, scene string) error {
	r.segments[id].SceneDesc = scene
	return nil
}

func (r *materialMemRepo) ListSegmentImagery(_ context.Context, id int64) ([]MaterialSegmentImagery, error) {
	out := []MaterialSegmentImagery{}
	for _, l := range r.links[id] {
		im := r.imagery[l.ImageryID-1]
		l.Name, l.Category, l.Status = im.Name, im.Category, im.Status
		out = append(out, l)
	}
	return out, nil
}

func (r *materialMemRepo) ReplaceAISegmentImagery(_ context.Context, id int64, links []MaterialSegmentImagery) error {
	kept := []MaterialSegmentImagery{}
	have := map[int64]bool{}
	for _, l := range r.links[id] {
		if l.Source != "ai" || l.Confirmed {
			kept = append(kept, l)
			have[l.ImageryID] = true
		}
	}
	for _, l := range links {
		if !have[l.ImageryID] {
			kept = append(kept, l)
		}
	}
	r.links[id] = kept
	return nil
}

func (r *materialMemRepo) SetSegmentImagery(_ context.Context, id int64, links []MaterialSegmentImagery) error {
	r.links[id] = links
	return nil
}

func materialTestChat(answer string, err error) (materialChatFunc, *[]string) {
	var users []string
	return func(_ context.Context, system, user string) (string, string, error) {
		users = append(users, user)
		return answer, "gpt-test", err
	}, &users
}

func TestNormalizeMaterialImagery(t *testing.T) {
	im := MaterialImagery{Name: " 柳 ", Aliases: []string{"杨柳", "柳", "杨柳", " "}, Category: "树木",
		Seasons: []string{"春", "春", "夏"}, Status: "bogus"}
	require.NoError(t, normalizeMaterialImagery(&im))
	require.Equal(t, "柳", im.Name)
	require.Equal(t, []string{"杨柳"}, im.Aliases)
	require.Equal(t, "", im.Category)
	require.Equal(t, []string{"春", "夏"}, im.Seasons)
	require.Equal(t, MaterialStatusDraft, im.Status)
	require.Equal(t, []string{}, im.Eras)

	require.ErrorIs(t, normalizeMaterialImagery(&MaterialImagery{Name: ""}), ErrMaterialImageryName)
	require.ErrorIs(t, normalizeMaterialImagery(&MaterialImagery{Name: "一二三四五六七八九"}), ErrMaterialImageryName)
}

func TestMaterialImageryIndexNamesWinOverAliases(t *testing.T) {
	idx := newMaterialImageryIndex([]MaterialImagery{
		{ID: 1, Name: "烟", Aliases: []string{"孤烟", "炊烟"}},
		{ID: 2, Name: "孤烟"},
	})
	require.Equal(t, int64(2), idx["孤烟"])
	require.Equal(t, int64(1), idx["炊烟"])
}

func TestParseMaterialJSON(t *testing.T) {
	var v struct {
		A int `json:"a"`
	}
	require.NoError(t, parseMaterialJSON("好的：\n```json\n{\"a\": 3}\n```", &v))
	require.Equal(t, 3, v.A)
	require.ErrorIs(t, parseMaterialJSON("没有 JSON", &v), ErrMaterialModelAnswer)
	require.ErrorIs(t, parseMaterialJSON("{bad}", &v), ErrMaterialModelAnswer)
}

func TestMaterialCandidatesEmbedded(t *testing.T) {
	c := materialCandidates()
	require.NotEmpty(t, c)
	require.Equal(t, 1, c[0].Rank)
	seen := map[string]bool{}
	for _, x := range c {
		require.False(t, seen[x.Name], x.Name)
		seen[x.Name] = true
		require.True(t, materialImageryCategorySet[x.Category], x.Name)
	}
}

func TestExtractSegmentImagery(t *testing.T) {
	repo := newMaterialMemRepo()
	repo.imagery = []MaterialImagery{
		{ID: 1, Name: "大漠", Aliases: []string{"沙漠"}, Category: "地理", Status: MaterialStatusPublished},
		{ID: 2, Name: "落日", Aliases: []string{"夕阳"}, Category: "天象", Status: MaterialStatusPublished},
		{ID: 3, Name: "旧条目", Status: MaterialStatusHidden},
	}
	repo.terms = []MaterialBlockTerm{{Term: "(?i)ghibli", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true}}
	repo.segments[10] = &MaterialSegmentContext{
		MaterialSegment: MaterialSegment{ID: 10, TextID: 1, Seq: 3, Label: "第3联", Content: "大漠孤烟直，长河落日圆。"},
		Kind:            "shi", Title: "使至塞上", AuthorName: "王维", Dynasty: "唐", LicenseStatus: MaterialLicensePublicDomain,
		Body: "单车欲问边，属国过居延。征蓬出汉塞，归雁入胡天。大漠孤烟直，长河落日圆。萧关逢候骑，都护在燕然。",
	}
	// A manual tag survives re-extraction.
	repo.links[10] = []MaterialSegmentImagery{{SegmentID: 10, ImageryID: 2, Weight: 1, Source: "manual", Confirmed: true}}

	answer := "```json\n" + `{"imagery":[
		{"name":"大漠","weight":3,"new":false},
		{"name":"夕阳","weight":2,"new":false},
		{"name":"孤烟","weight":3,"new":true,"category":"气候","meaning":"边塞的烽烟","visual_zh":"一缕笔直的烟","visual_en":"a single straight column of smoke, Ghibli style","emotions":["壮阔"]},
		{"name":"长河","weight":9,"new":true,"category":"地理","visual_en":"a long winding river reflecting the sunset"},
		{"name":"大漠","weight":1}
	],"scene_desc":"黄昏的大漠上，一缕烟直直升起，远处长河映着圆圆的落日。"}` + "\n```"
	chat, users := materialTestChat(answer, nil)
	svc := &MaterialService{repo: repo, chat: chat}

	res, err := svc.ExtractSegmentImagery(context.Background(), 5, 10)
	require.NoError(t, err)
	require.Equal(t, "gpt-test", res.Model)
	require.Equal(t, []string{"孤烟", "长河"}, res.Created)
	require.Len(t, res.Warnings, 1)

	// The prompt carries the segment, the full text and existing names with aliases, not hidden ones.
	require.Len(t, *users, 1)
	u := (*users)[0]
	require.Contains(t, u, "大漠孤烟直，长河落日圆。")
	require.Contains(t, u, "全文供理解")
	require.Contains(t, u, "落日（夕阳）")
	require.NotContains(t, u, "旧条目")

	byName := map[string]MaterialSegmentImagery{}
	for _, l := range res.Links {
		byName[l.Name] = l
	}
	require.Len(t, byName, 4)
	require.Equal(t, 3, byName["大漠"].Weight)
	require.Equal(t, "manual", byName["落日"].Source, "the confirmed manual tag stays")
	require.Equal(t, 3, byName["长河"].Weight, "weight is clamped")
	require.False(t, byName["孤烟"].Confirmed)

	gu := repo.imagery[3]
	require.Equal(t, "孤烟", gu.Name)
	require.Equal(t, MaterialStatusDraft, gu.Status)
	require.Equal(t, "", gu.VisualEn, "a visual naming a studio is dropped")
	require.Equal(t, int64(5), *gu.CreatedBy)
	require.Equal(t, "黄昏的大漠上，一缕烟直直升起，远处长河映着圆圆的落日。", repo.segments[10].SceneDesc)

	// Running it again replaces the unconfirmed AI tags and keeps an existing scene.
	repo.segments[10].SceneDesc = "管理员改过的描述"
	_, err = svc.ExtractSegmentImagery(context.Background(), 5, 10)
	require.NoError(t, err)
	require.Len(t, repo.links[10], 4)
	require.Equal(t, "管理员改过的描述", repo.segments[10].SceneDesc)
	require.Len(t, repo.imagery, 5)
}

func TestExtractSegmentImageryReferenceOnly(t *testing.T) {
	repo := newMaterialMemRepo()
	repo.segments[1] = &MaterialSegmentContext{
		MaterialSegment: MaterialSegment{ID: 1, Label: "副歌", SceneDesc: "雨夜街灯下的背影"},
		Kind:            "song", Title: "测试歌曲", LicenseStatus: MaterialLicenseReference,
	}
	chat, users := materialTestChat(`{"imagery":[{"name":"街灯","weight":3,"new":true,"category":"器物"}],"scene_desc":"x"}`, nil)
	svc := &MaterialService{repo: repo, chat: chat}
	_, err := svc.ExtractSegmentImagery(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Contains(t, (*users)[0], "画面描述：\n雨夜街灯下的背影")
	require.NotContains(t, (*users)[0], "全文供理解")
	require.Equal(t, "雨夜街灯下的背影", repo.segments[1].SceneDesc)

	repo.segments[2] = &MaterialSegmentContext{MaterialSegment: MaterialSegment{ID: 2}, LicenseStatus: MaterialLicenseReference}
	_, err = svc.ExtractSegmentImagery(context.Background(), 1, 2)
	require.Error(t, err)
}

func TestExtractSegmentImageryErrors(t *testing.T) {
	repo := newMaterialMemRepo()
	repo.segments[1] = &MaterialSegmentContext{MaterialSegment: MaterialSegment{ID: 1, Content: "床前明月光，疑是地上霜。"}, LicenseStatus: MaterialLicensePublicDomain}

	_, err := (&MaterialService{repo: repo}).ExtractSegmentImagery(context.Background(), 1, 1)
	require.ErrorIs(t, err, ErrMaterialNoChat)

	chat, _ := materialTestChat("", errors.New("boom"))
	_, err = (&MaterialService{repo: repo, chat: chat}).ExtractSegmentImagery(context.Background(), 1, 1)
	require.Error(t, err)

	chat, _ = materialTestChat(`{"imagery":[]}`, nil)
	_, err = (&MaterialService{repo: repo, chat: chat}).ExtractSegmentImagery(context.Background(), 1, 1)
	require.ErrorIs(t, err, ErrMaterialModelAnswer)

	_, err = (&MaterialService{repo: repo, chat: chat}).ExtractSegmentImagery(context.Background(), 1, 99)
	require.ErrorIs(t, err, ErrMaterialSegmentNotFound)
}

func TestSaveSegmentImagery(t *testing.T) {
	repo := newMaterialMemRepo()
	repo.imagery = []MaterialImagery{{ID: 1, Name: "月"}, {ID: 2, Name: "霜"}, {ID: 3, Name: "床"}}
	repo.segments[1] = &MaterialSegmentContext{MaterialSegment: MaterialSegment{ID: 1}}
	repo.links[1] = []MaterialSegmentImagery{{ImageryID: 1, Weight: 3, Source: "ai"}, {ImageryID: 3, Weight: 1, Source: "ai"}}
	svc := &MaterialService{repo: repo}

	scene := "  月光洒在地上  "
	res, err := svc.SaveSegmentImagery(context.Background(), 1, []MaterialSegmentImageryInput{
		{ImageryID: 1, Weight: 3}, {ImageryID: 2, Weight: 0}, {ImageryID: 2, Weight: 3}, {ImageryID: -1},
	}, &scene)
	require.NoError(t, err)
	require.Len(t, res.Links, 2)
	require.Equal(t, "ai", res.Links[0].Source)
	require.Equal(t, "manual", res.Links[1].Source)
	require.Equal(t, 1, res.Links[1].Weight)
	require.True(t, res.Links[0].Confirmed && res.Links[1].Confirmed)
	require.Equal(t, "月光洒在地上", repo.segments[1].SceneDesc)

	_, err = svc.SaveSegmentImagery(context.Background(), 1, make([]MaterialSegmentImageryInput, 9), nil)
	require.ErrorIs(t, err, ErrMaterialInvalid)
}

func TestDraftImagerySeeds(t *testing.T) {
	cands := materialCandidates()
	require.GreaterOrEqual(t, len(cands), 13)
	repo := newMaterialMemRepo()
	// The top candidate is already in the library under an alias of another entry.
	repo.imagery = []MaterialImagery{{ID: 1, Name: "别名占位", Aliases: []string{cands[0].Name}}}

	var calls atomic.Int32 // batches are drafted in parallel
	chat := func(_ context.Context, _, user string) (string, string, error) {
		calls.Add(1)
		var items []string
		for _, line := range strings.Split(user, "\n") {
			if !strings.HasPrefix(line, "- ") {
				continue
			}
			name := strings.TrimPrefix(line, "- ")
			if i := strings.Index(name, "（"); i >= 0 {
				name = name[:i]
			}
			if name == cands[2].Name {
				continue // the model forgets one
			}
			items = append(items, `{"name":"`+name+`","category":"不是类别","meaning":"含义","visual_zh":"画法","visual_en":"a quiet scene","emotions":["闲适"]}`)
		}
		return `{"items":[` + strings.Join(items, ",") + `]}`, "gpt-test", nil
	}
	svc := &MaterialService{repo: repo, chat: chat}

	res, err := svc.DraftImagerySeeds(context.Background(), 3, 12)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load(), "12 candidates in batches of 10")
	require.Equal(t, "gpt-test", res.Model)
	require.Equal(t, []string{cands[2].Name}, res.Failed)
	require.Len(t, res.Drafted, 11)
	require.NotContains(t, res.Drafted, cands[0].Name)
	require.Equal(t, len(cands)-1-11, res.Left)

	second := repo.imagery[1]
	require.Equal(t, cands[1].Name, second.Name)
	require.Equal(t, cands[1].Category, second.Category, "an unknown category falls back to the candidate's")
	require.Equal(t, MaterialStatusDraft, second.Status)
	for _, a := range cands[1].Aliases[:min(len(cands[1].Aliases), materialMaxImageryAlias)] {
		require.Contains(t, second.Aliases, a)
	}

	// The model failing outright is an error when nothing was drafted.
	failing := &MaterialService{repo: newMaterialMemRepo(), chat: func(context.Context, string, string) (string, string, error) {
		return "", "", errors.New("upstream down")
	}}
	_, err = failing.DraftImagerySeeds(context.Background(), 3, 5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream down")
}
