//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPromptSyncParsesHandrawLayouts(t *testing.T) {
	const ref = "/yang0/handraw-style/master/skills/handdraw-style-prompter/references/"
	mux := http.NewServeMux()
	mux.HandleFunc(ref+"layouts.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id": "IG-037", "category": "infographic", "name": "高精细手绘等距轴测地图信息图", "image": "../../../images/layouts/infographics/IG-037.webp",
			 "prompt_file": "layouts/IG-037.md", "keywords": ["高精细手绘等距轴测地图信息图", "旅游导览地图"], "name_en": "High-Detail Isometric Travel Guide Map Infographic"},
			{"id": "SB-001", "category": "comic-storyboard", "name": "规则网格", "image": "../../../images/layouts/storyboards/SB-001.webp", "prompt_file": "layouts/SB-001.md", "keywords": []},
			{"id": "EC-002", "category": "ecommerce", "name": "缺文件", "image": "", "prompt_file": "layouts/EC-002.md"},
			{"id": "IG-037", "category": "infographic", "name": "dup", "prompt_file": "layouts/IG-037.md"},
			{"id": "XX-001", "category": "infographic", "name": "bad path", "prompt_file": "../../secrets.md"},
			{"id": "bad", "category": "infographic", "name": "bad id", "prompt_file": "layouts/bad.md"}
		]`))
	})
	mux.HandleFunc(ref+"layouts/IG-037.md", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!-- zh -->\n高精细手绘等距轴测地图信息图，鸟瞰视角。根据主题进行信息标注。\n\n<!-- en -->\nHigh-detail hand-drawn isometric map.\n"))
	})
	mux.HandleFunc(ref+"layouts/SB-001.md", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("漫画分镜。排版：规则网格。"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	sync := NewPromptLibrarySyncService(nil, srv.Client(), nil)
	src := &PromptSource{ID: "handraw-layouts", Format: "handraw-layouts", Homepage: "https://github.com/yang0/handraw-style", URL: srv.URL + ref + "layouts.json"}

	items, err := sync.fetch(context.Background(), src)
	require.NoError(t, err)
	require.Len(t, items, 2, "a missing file, a duplicate, a path outside layouts/ and a bad id are skipped")

	ig := items[0]
	require.Equal(t, "IG-037", ig.ExternalID)
	require.Equal(t, "图型 IG-037 · 高精细手绘等距轴测地图信息图", ig.Title)
	require.Equal(t, srv.URL+"/yang0/handraw-style/master/images/layouts/infographics/IG-037.webp", ig.CoverURL)
	require.Equal(t, "【主题：在这里写你的主题或内容】\n\n高精细手绘等距轴测地图信息图，鸟瞰视角。根据主题进行信息标注。", ig.Prompt, "only the Chinese section")
	require.Contains(t, ig.Description, "信息图")
	require.Contains(t, ig.Description, "High-Detail Isometric")
	require.Contains(t, ig.Scenes, "infographic")
	require.Contains(t, ig.SourceTags, "旅游导览地图")
	require.Equal(t, "yang0", ig.Author)
	require.Equal(t, "https://github.com/yang0/handraw-style", ig.SourceURL)

	require.Equal(t, "【主题：在这里写你的主题或内容】\n\n漫画分镜。排版：规则网格。", items[1].Prompt, "a file without markers is used whole")
}

func TestPromptSyncHandrawLayoutsFailsWhenMostFilesAreMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/r/layouts.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id": "IG-001", "category": "infographic", "name": "a", "prompt_file": "layouts/IG-001.md"}, {"id": "IG-002", "category": "infographic", "name": "b", "prompt_file": "layouts/IG-002.md"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	sync := NewPromptLibrarySyncService(nil, srv.Client(), nil)
	_, err := sync.fetch(context.Background(), &PromptSource{ID: "x", Format: "handraw-layouts", URL: srv.URL + "/r/layouts.json"})
	require.ErrorContains(t, err, "2 / 2", "the library keeps the last copy instead of emptying the source")
}

func TestHandrawLayoutZhSection(t *testing.T) {
	require.Equal(t, "中文", handrawLayoutZh("<!-- en -->\nEnglish\n<!-- zh -->\n中文\n"))
	require.Equal(t, "", handrawLayoutZh("<!-- en -->\nEnglish only\n"))
}

// Run against the real repository: HANDRAW_LIVE=1 go test -tags=unit -run TestPromptSyncHandrawLayoutsLive ./internal/service/
func TestPromptSyncHandrawLayoutsLive(t *testing.T) {
	if os.Getenv("HANDRAW_LIVE") == "" {
		t.Skip("set HANDRAW_LIVE=1 to fetch the real layouts")
	}
	sync := NewPromptLibrarySyncService(nil, &http.Client{Timeout: time.Minute}, nil)
	items, err := sync.fetch(context.Background(), &PromptSource{ID: "handraw-layouts", Format: "handraw-layouts", Homepage: "https://github.com/yang0/handraw-style",
		URL: "https://raw.githubusercontent.com/yang0/handraw-style/master/skills/handdraw-style-prompter/references/layouts.json"})
	require.NoError(t, err)
	t.Logf("items: %d", len(items))
	for _, item := range items {
		if item.ExternalID == "IG-037" || item.ExternalID == "EC-001" {
			t.Logf("%s | %s | %v | %s\n%s", item.ExternalID, item.Title, item.Scenes, item.CoverURL, item.Prompt)
		}
		require.NotEmpty(t, item.Prompt)
		require.Equal(t, PromptStatusActive, item.Status, "%s flagged: %v", item.ExternalID, item.AutoFlags)
	}
}
