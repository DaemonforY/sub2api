package service

import (
	"context"
	"path/filepath"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// ProvideTietuService wires AI 做贴图: pictures live next to the articles (<data dir>/tietu), poems
// come from the poem library, and runs a previous process left behind are marked failed.
func ProvideTietuService(repo TietuProjectRepository, learn *LearnService, inspiration *InspirationService, openai *OpenAIGatewayService, cfg *config.Config) *TietuService {
	dir := "./data/tietu"
	if cfg != nil && cfg.Video.Dir != "" {
		dir = filepath.Join(filepath.Dir(filepath.Clean(cfg.Video.Dir)), "tietu")
	}
	var poems TietuPoemSource
	if inspiration != nil {
		poems = inspiration
	}
	svc := NewTietuService(repo, learn, poems, dir)
	if openai != nil {
		svc.SetPricer(openai)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	svc.RecoverInterrupted(ctx)
	return svc
}
