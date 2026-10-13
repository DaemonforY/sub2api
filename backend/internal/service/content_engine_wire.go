package service

import (
	"context"
	"time"
)

// ProvideContentEngineService wires the 内容引擎: channel links and promo codes for approved versions,
// the support questions, GEO questions and learning-site pages as topic material, the gateway's web
// search; runs a previous process left behind are marked failed and the daily schedule starts.
func ProvideContentEngineService(repo ContentEngineRepository, learn *LearnService, settings SettingRepository, encryptor SecretEncryptor,
	settingService *SettingService, links *ChannelLinkService, promo *PromoService, assistant *AssistantService,
	runs AgentRunRepository, geo GeoMonitorRepository) *ContentEngineService {
	var store learnSettingStore
	if settings != nil {
		store = settings
	}
	svc := NewContentEngineService(repo, learn, store, encryptor)
	var search ArticleSearcher
	if settingService != nil {
		search = gatewayArticleSearcher{settings: settingService}
	}
	sources := ContentSources{Runs: runs, Geo: geo}
	if assistant != nil {
		sources.Search, sources.Catalog = assistant.SearchPages, assistant.PageCatalog
	}
	var linker contentLinker
	if links != nil {
		linker = links
	}
	var promoter contentPromoter
	if promo != nil {
		promoter = promo
	}
	svc.SetDeps(search, linker, promoter, sources)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	svc.RecoverInterrupted(ctx)
	svc.Start()
	return svc
}
