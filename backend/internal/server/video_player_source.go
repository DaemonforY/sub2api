package server

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// videoPlayerSource gives the frontend server the scene data of public works for /play/<id>.
// Gallery pages open many cards at once, so the encoded data is cached briefly.
type videoPlayerSource struct {
	videos *service.VideoService

	mu    sync.Mutex
	cache map[string]videoPlayerEntry
}

type videoPlayerEntry struct {
	data  []byte
	ok    bool
	until time.Time
}

const (
	videoPlayerTTL      = time.Minute
	videoPlayerCacheMax = 500
)

func newVideoPlayerSource(videos *service.VideoService) *videoPlayerSource {
	return &videoPlayerSource{videos: videos, cache: map[string]videoPlayerEntry{}}
}

type videoPlayerScene struct {
	ID       string              `json:"id"`
	Duration float64             `json:"duration"`
	Code     string              `json:"code"`
	Words    []service.VideoWord `json:"words"`
}

func (s *videoPlayerSource) PlayerData(ctx context.Context, id string) ([]byte, bool) {
	if s == nil || s.videos == nil {
		return nil, false
	}
	s.mu.Lock()
	if e, hit := s.cache[id]; hit && time.Now().Before(e.until) {
		s.mu.Unlock()
		return e.data, e.ok
	}
	s.mu.Unlock()

	var data []byte
	ok := false
	if p, err := s.videos.Work(ctx, id); err == nil && p != nil && p.Spec != nil {
		spec := p.Spec
		scenes := make([]videoPlayerScene, 0, len(spec.Scenes))
		for _, sc := range spec.Scenes {
			vs := videoPlayerScene{ID: sc.ID, Duration: sc.Duration, Code: sc.Code, Words: []service.VideoWord{}}
			if sc.Audio != nil && sc.Audio.Words != nil {
				vs.Words = sc.Audio.Words
			}
			scenes = append(scenes, vs)
		}
		data, err = json.Marshal(map[string]any{"width": spec.Width, "height": spec.Height, "theme": spec.Theme, "loop": spec.Loop, "poster": spec.Poster, "scenes": scenes})
		ok = err == nil
	} else if err != nil && !isVideoNotFound(err) {
		return nil, false // a database hiccup: do not cache
	}

	s.mu.Lock()
	if len(s.cache) >= videoPlayerCacheMax {
		s.cache = map[string]videoPlayerEntry{}
	}
	s.cache[id] = videoPlayerEntry{data: data, ok: ok, until: time.Now().Add(videoPlayerTTL)}
	s.mu.Unlock()
	return data, ok
}

func isVideoNotFound(err error) bool {
	return errors.Is(err, service.ErrVideoNotFound) || (err != nil && err.Error() == service.ErrVideoNotFound.Error())
}
