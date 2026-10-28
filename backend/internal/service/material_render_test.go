//go:build unit

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func (r *materialQueueRepo) SaveArtworkRender(_ context.Context, m *MaterialRender) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, x := range r.renders {
		if x.ArtworkID == m.ArtworkID && x.Aspect == m.Aspect {
			m.ID = x.ID
			r.renders[i] = *m
			return nil
		}
	}
	m.ID = int64(len(r.renders) + 1)
	m.CreatedAt = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	r.renders = append(r.renders, *m)
	return nil
}

func (r *materialQueueRepo) ListArtworkRenders(_ context.Context, ids []int64) ([]MaterialRender, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	want := map[int64]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := []MaterialRender{}
	for _, x := range r.renders {
		if want[x.ArtworkID] {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Aspect < out[j].Aspect })
	return out, nil
}

func TestMaterialByline(t *testing.T) {
	require.Equal(t, "王维《使至塞上》", MaterialByline(" 王维 ", "使至塞上"))
	require.Equal(t, "《佚名诗》", MaterialByline("", "佚名诗"))
	require.Equal(t, "王维", MaterialByline("王维", ""))
}

func TestCheckMaterialLayoutWords(t *testing.T) {
	poem := &MaterialSegmentContext{MaterialSegment: MaterialSegment{Content: "大漠孤烟直，长河落日圆。"}, Title: "使至塞上", AuthorName: "王维",
		LicenseStatus: MaterialLicensePublicDomain, Body: "单车欲问边，属国过居延。征蓬出汉塞，归雁入胡天。大漠孤烟直，长河落日圆。萧关逢候骑，都护在燕然。"}

	require.NoError(t, CheckMaterialLayoutWords("大漠孤烟直，长河落日圆。", "王维《使至塞上》", poem))
	require.NoError(t, CheckMaterialLayoutWords("大漠孤烟直\n长河落日圆", "", poem), "punctuation and line breaks don't matter")
	require.NoError(t, CheckMaterialLayoutWords("征蓬出汉塞，归雁入胡天。大漠孤烟直，长河落日圆。", "", poem), "any part of the text")
	require.NoError(t, CheckMaterialLayoutWords("", "", poem))
	require.ErrorIs(t, CheckMaterialLayoutWords("大漠孤烟直，长河落日园。", "", poem), ErrMaterialWordsMismatch, "one wrong character")
	require.ErrorIs(t, CheckMaterialLayoutWords("，。", "", poem), ErrMaterialWordsMismatch, "punctuation only")
	err := CheckMaterialLayoutWords("", "王维《使至塞下》", poem)
	require.Error(t, err)
	require.Contains(t, err.Error(), "王维《使至塞上》")

	// Without a stored body, the segment's own words are the reference.
	short := *poem
	short.Body = ""
	require.NoError(t, CheckMaterialLayoutWords("长河落日圆", "", &short))
	require.ErrorIs(t, CheckMaterialLayoutWords("单车欲问边", "", &short), ErrMaterialWordsMismatch)

	// Reference-only: a byline at most.
	ref := &MaterialSegmentContext{MaterialSegment: MaterialSegment{SceneDesc: "雨夜的街口"}, Title: "测试歌曲", AuthorName: "测试词作者", LicenseStatus: MaterialLicenseReference}
	require.NoError(t, CheckMaterialLayoutWords("", "测试词作者《测试歌曲》", ref))
	require.ErrorIs(t, CheckMaterialLayoutWords("任何文字", "", ref), ErrMaterialWordsRefOnly)

	// The segment is gone: nothing at all.
	require.NoError(t, CheckMaterialLayoutWords(" ", "", nil))
	require.ErrorIs(t, CheckMaterialLayoutWords("", "王维《使至塞上》", nil), ErrMaterialWordsMismatch)
}

func TestMaterialSaveRenderAndExport(t *testing.T) {
	svc, repo, _, _ := newMaterialQueueFixture(t)
	repo.segments[1].Body = "单车欲问边，属国过居延。征蓬出汉塞，归雁入胡天。大漠孤烟直，长河落日圆。萧关逢候骑，都护在燕然。"
	ctx := context.Background()
	_, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 2)
	require.NoError(t, err)
	require.True(t, drawOne(t, svc))
	require.True(t, drawOne(t, svc))
	png := shihuaPNG(t, 40, 60)
	layout := json.RawMessage(`{"aspect":"2x3","text":"大漠孤烟直，长河落日圆。","byline":"王维《使至塞上》","font":"wenkai"}`)

	_, err = svc.SaveRender(ctx, 1, "2x3", layout, png)
	require.ErrorIs(t, err, ErrMaterialNotApproved)
	_, err = svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: MaterialReviewApproved, Rating: 5})
	require.NoError(t, err)

	_, err = svc.SaveRender(ctx, 1, "5x7", layout, png)
	require.ErrorIs(t, err, ErrMaterialInvalid)
	_, err = svc.SaveRender(ctx, 1, "2x3", json.RawMessage(`{"text":"大漠孤烟直，长河落日园。"}`), png)
	require.ErrorIs(t, err, ErrMaterialWordsMismatch)
	_, err = svc.SaveRender(ctx, 1, "2x3", layout, []byte("not an image"))
	require.ErrorIs(t, err, ErrMaterialInvalid)
	_, err = svc.SaveRender(ctx, 1, "2x3", json.RawMessage(`not json`), png)
	require.ErrorIs(t, err, ErrMaterialInvalid)

	r, err := svc.SaveRender(ctx, 1, "2x3", layout, png)
	require.NoError(t, err)
	require.Equal(t, "artworks/0/1-2x3.png", r.File)
	require.Equal(t, 40, r.Width)
	link, err := url.Parse(r.URL)
	require.NoError(t, err)
	_, ok := svc.OpenFile(strings.TrimPrefix(link.Path, materialFileRoute), link.Query().Get("e"), link.Query().Get("s"))
	require.True(t, ok, "the poster opens by its signed link")
	// Saving the aspect again replaces it.
	r2, err := svc.SaveRender(ctx, 1, "2x3", layout, shihuaPNG(t, 44, 66))
	require.NoError(t, err)
	require.Equal(t, r.ID, r2.ID)
	_, err = svc.SaveRender(ctx, 1, "16x9", json.RawMessage(`{"text":"","byline":"王维《使至塞上》"}`), png)
	require.NoError(t, err)

	renders, err := svc.ListRenders(ctx, []int64{1, 2})
	require.NoError(t, err)
	require.Len(t, renders, 2)
	require.NotEmpty(t, renders[0].URL)
	_, err = svc.ListRenders(ctx, nil)
	require.ErrorIs(t, err, ErrMaterialInvalid)

	// Artwork 2 is not approved and has no posters: it is left out.
	var buf bytes.Buffer
	require.NoError(t, svc.ExportZip(ctx, &buf, []int64{1, 2}, []string{"2x3", "16x9"}))
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	names := map[string]*zip.File{}
	for _, f := range zr.File {
		names[f.Name] = f
	}
	require.Len(t, names, 3)
	require.Contains(t, names, "2x3/001-使至塞上-1-2比3.png")
	require.Contains(t, names, "16x9/001-使至塞上-1-横版16比9.png")
	note, err := names["说明/001-使至塞上-1.txt"].Open()
	require.NoError(t, err)
	text, err := io.ReadAll(note)
	require.NoError(t, err)
	require.Contains(t, string(text), "篇名：使至塞上")
	require.Contains(t, string(text), "原文：大漠孤烟直，长河落日圆。")
	require.Contains(t, string(text), "版权状态：公有领域")
	require.Contains(t, string(text), "配图由 AI 生成")

	// Only some aspects.
	buf.Reset()
	require.NoError(t, svc.ExportZip(ctx, &buf, []int64{1}, []string{"16x9"}))
	zr, err = zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)

	// Storage off.
	off := &MaterialService{repo: repo}
	_, err = off.SaveRender(ctx, 1, "2x3", layout, png)
	require.ErrorIs(t, err, ErrMaterialFilesOff)
}
