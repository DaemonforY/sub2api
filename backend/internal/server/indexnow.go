package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
)

// IndexNow: shortly after start and then daily, push the public pages that are new or changed since
// the last successful push. Fingerprints of pushed pages live in one Redis hash, so a deploy that only
// touches three tutorials pushes three URLs.

const (
	indexNowRedisKey   = "seo:indexnow:pages"
	indexNowBatchSize  = 1000 // the protocol allows 10,000 per request
	indexNowFirstDelay = 2 * time.Minute
	indexNowInterval   = 24 * time.Hour
)

var indexNowKeyPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,128}$`)

type indexNowSource interface {
	IndexNowPages(ctx context.Context) map[string]string
}

func indexNowEnabled(cfg config.IndexNowConfig) bool {
	return indexNowKeyPattern.MatchString(strings.TrimSpace(cfg.Key)) && strings.HasPrefix(strings.TrimSpace(cfg.Site), "https://")
}

func startIndexNow(cfg config.IndexNowConfig, src indexNowSource, rdb *redis.Client) {
	if !indexNowEnabled(cfg) || src == nil || rdb == nil {
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	go func() {
		time.Sleep(indexNowFirstDelay) // let settings and the SEO content load first
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			n, err := runIndexNow(ctx, cfg, src, rdb, client)
			cancel()
			if err != nil {
				slog.Warn("indexnow_push_failed", "error", err, "pushed", n)
			} else if n > 0 {
				slog.Info("indexnow_pushed", "urls", n)
			}
			time.Sleep(indexNowInterval)
		}
	}()
}

// runIndexNow pushes the changed pages and records them; it returns how many URLs were accepted.
func runIndexNow(ctx context.Context, cfg config.IndexNowConfig, src indexNowSource, rdb *redis.Client, client *http.Client) (int, error) {
	site := strings.TrimRight(strings.TrimSpace(cfg.Site), "/")
	u, err := url.Parse(site)
	if err != nil || u.Host == "" {
		return 0, fmt.Errorf("bad indexnow.site %q", cfg.Site)
	}
	pages := src.IndexNowPages(ctx)
	sent, err := rdb.HGetAll(ctx, indexNowRedisKey).Result()
	if err != nil {
		return 0, err
	}
	var changed []string
	for path, fp := range pages {
		if sent[path] != fp {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.indexnow.org/indexnow"
	}
	key := strings.TrimSpace(cfg.Key)
	pushed := 0
	for start := 0; start < len(changed); start += indexNowBatchSize {
		batch := changed[start:min(start+indexNowBatchSize, len(changed))]
		urls := make([]string, len(batch))
		for i, p := range batch {
			urls[i] = site + p
		}
		body, _ := json.Marshal(map[string]any{
			"host": u.Host, "key": key, "keyLocation": site + "/" + key + ".txt", "urlList": urls,
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return pushed, err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(req)
		if err != nil {
			return pushed, err
		}
		_ = resp.Body.Close()
		// 200 = accepted, 202 = accepted, key validation pending.
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			return pushed, fmt.Errorf("indexnow answered %d", resp.StatusCode)
		}
		fields := make(map[string]any, len(batch))
		for _, p := range batch {
			fields[p] = pages[p]
		}
		if err := rdb.HSet(ctx, indexNowRedisKey, fields).Err(); err != nil {
			return pushed, err
		}
		pushed += len(batch)
	}
	return pushed, nil
}
