package service

import (
	"context"
	"path/filepath"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
)

// gatewayArticleSearcher searches through the gateway's 联网搜索 providers (后台 → 设置 → 联网搜索模拟).
// Articles only need a provider with a key; the gateway-wide emulation switch is not required.
type gatewayArticleSearcher struct {
	settings *SettingService
}

func (g gatewayArticleSearcher) Available(ctx context.Context) bool {
	if g.settings == nil || getWebSearchManager() == nil {
		return false
	}
	cfg, err := g.settings.GetWebSearchEmulationConfig(ctx)
	if err != nil || cfg == nil {
		return false
	}
	now := time.Now().Unix()
	for _, p := range cfg.Providers {
		if p.APIKey != "" && (p.ExpiresAt == nil || *p.ExpiresAt > now) {
			return true
		}
	}
	return false
}

func (g gatewayArticleSearcher) Search(ctx context.Context, query string, max int) ([]websearch.SearchResult, error) {
	m := getWebSearchManager()
	if m == nil {
		return nil, errLearnRunFailed("联网搜索未开启")
	}
	res, _, err := m.SearchWithBestProvider(ctx, websearch.SearchRequest{Query: query, MaxResults: max})
	if err != nil {
		return nil, err
	}
	return res.Results, nil
}

// ProvideArticleAgentService wires AI 写文章: pictures are kept next to the video files
// (<data dir>/articles), and runs a previous process left behind are marked failed.
func ProvideArticleAgentService(repo ArticleProjectRepository, learn *LearnService, settings *SettingService, cfg *config.Config) *ArticleAgentService {
	dir := "./data/articles"
	if cfg != nil && cfg.Video.Dir != "" {
		dir = filepath.Join(filepath.Dir(filepath.Clean(cfg.Video.Dir)), "articles")
	}
	var search ArticleSearcher
	if settings != nil {
		search = gatewayArticleSearcher{settings: settings}
	}
	svc := NewArticleAgentService(repo, learn, search, dir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	svc.RecoverInterrupted(ctx)
	return svc
}
