//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---- stubs ----

type geoRepoStub struct {
	mu        sync.Mutex
	questions []GeoQuestion
	engines   []GeoEngine
	checks    []GeoCheck
	nextID    int64
}

func (r *geoRepoStub) ListQuestions(context.Context) ([]GeoQuestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]GeoQuestion(nil), r.questions...), nil
}
func (r *geoRepoStub) GetQuestion(_ context.Context, id int64) (*GeoQuestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, q := range r.questions {
		if q.ID == id {
			return &q, nil
		}
	}
	return nil, ErrGeoQuestionNotFound
}
func (r *geoRepoStub) CreateQuestion(_ context.Context, q *GeoQuestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	q.ID = r.nextID
	r.questions = append(r.questions, *q)
	return nil
}
func (r *geoRepoStub) UpdateQuestion(context.Context, *GeoQuestion) error { return nil }
func (r *geoRepoStub) DeleteQuestion(context.Context, int64) error        { return nil }
func (r *geoRepoStub) ListEngines(context.Context) ([]GeoEngine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]GeoEngine(nil), r.engines...), nil
}
func (r *geoRepoStub) GetEngine(_ context.Context, id int64) (*GeoEngine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.engines {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, ErrGeoEngineNotFound
}
func (r *geoRepoStub) CreateEngine(_ context.Context, e *GeoEngine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	e.ID = r.nextID
	r.engines = append(r.engines, *e)
	return nil
}
func (r *geoRepoStub) UpdateEngine(_ context.Context, e *GeoEngine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.engines {
		if r.engines[i].ID == e.ID {
			r.engines[i] = *e
		}
	}
	return nil
}
func (r *geoRepoStub) DeleteEngine(context.Context, int64) error { return nil }
func (r *geoRepoStub) InsertCheck(_ context.Context, c *GeoCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	c.ID = r.nextID
	c.CreatedAt = time.Now()
	r.checks = append(r.checks, *c)
	return nil
}
func (r *geoRepoStub) ListChecks(context.Context, GeoCheckFilter) ([]GeoCheck, int64, error) {
	return nil, 0, nil
}
func (r *geoRepoStub) DeleteCheck(context.Context, int64) error { return nil }
func (r *geoRepoStub) LatestChecks(context.Context) ([]GeoCheck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]GeoCheck(nil), r.checks...), nil
}
func (r *geoRepoStub) WeeklyRates(context.Context, time.Time) ([]GeoWeekRate, error) {
	return []GeoWeekRate{{EngineName: "Perplexity", WeekStart: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Total: 4, Mentioned: 1}}, nil
}

type geoSettingsStub struct {
	mu sync.Mutex
	m  map[string]string
}

func (s *geoSettingsStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for _, k := range keys {
		out[k] = s.m[k]
	}
	return out, nil
}
func (s *geoSettingsStub) SetMultiple(_ context.Context, v map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, val := range v {
		s.m[k] = val
	}
	return nil
}

func newGeoTestService() (*GeoMonitorService, *geoRepoStub, *geoSettingsStub) {
	repo := &geoRepoStub{}
	st := &geoSettingsStub{m: map[string]string{}}
	return NewGeoMonitorService(repo, st, prefixEncryptor{}, nil), repo, st
}

// ---- detection ----

func TestDetectGeoMentionsAndCitations(t *testing.T) {
	answer := "推荐 HiveGPT（https://hivegpt.cn/learn/connect/）。也可以看 https://docs.example.com/a，或者 OpenRouter。"
	d := DetectGeo(answer, []string{"https://www.hivegpt.cn/pricing", "https://docs.example.com/a", "ftp://x.y"},
		[]string{"hivegpt", "hivegpt.cn"}, []string{"openrouter", "aihubmix"})
	require.True(t, d.Mentioned)
	require.Equal(t, []string{"https://www.hivegpt.cn/pricing", "https://docs.example.com/a", "https://hivegpt.cn/learn/connect/"}, d.CitedURLs)
	require.Equal(t, []string{"https://www.hivegpt.cn/pricing", "https://hivegpt.cn/learn/connect/"}, d.OurURLs)
	require.Equal(t, []string{"openrouter"}, d.Competitors)

	// A cited link alone counts as a mention; a look-alike domain is not ours.
	d = DetectGeo("看这里", []string{"https://hivegpt.cn/x", "https://nothivegpt.cn.evil.com/"}, []string{"hivegpt.cn"}, nil)
	require.True(t, d.Mentioned)
	require.Equal(t, []string{"https://hivegpt.cn/x"}, d.OurURLs)

	d = DetectGeo("用 OpenAI 官方 API 就行。", nil, []string{"hivegpt"}, nil)
	require.False(t, d.Mentioned)
	require.Empty(t, d.CitedURLs)
	require.Empty(t, d.OurURLs)
	require.Empty(t, d.Competitors)
}

func TestDetectGeoCapsCitations(t *testing.T) {
	var urls []string
	for i := 0; i < 80; i++ {
		urls = append(urls, "https://e.com/"+strings.Repeat("a", i+1), "https://e.com/a")
	}
	require.Len(t, DetectGeo("", urls, nil, nil).CitedURLs, geoMaxCitedURLs)
}

func TestParseGeoChatResponseCitations(t *testing.T) {
	perplexity := `{"choices":[{"message":{"content":"答案 [1]"}}],"citations":["https://hivegpt.cn/","https://a.com/"],"search_results":[{"url":"https://b.com/"}]}`
	answer, cited, err := parseGeoChatResponse([]byte(perplexity))
	require.NoError(t, err)
	require.Equal(t, "答案 [1]", answer)
	require.Equal(t, []string{"https://hivegpt.cn/", "https://a.com/", "https://b.com/"}, cited)

	openai := `{"choices":[{"message":{"content":[{"type":"text","text":"见 "},{"type":"text","text":"文档"}],
	  "annotations":[{"type":"url_citation","url_citation":{"url":"https://c.com/x"}}]}}]}`
	answer, cited, err = parseGeoChatResponse([]byte(openai))
	require.NoError(t, err)
	require.Equal(t, "见 文档", answer)
	require.Equal(t, []string{"https://c.com/x"}, cited)

	_, _, err = parseGeoChatResponse([]byte("<html>"))
	require.Error(t, err)
}

func TestBuildGeoRequestBodyCannotOverrideCore(t *testing.T) {
	b, err := buildGeoRequestBody(json.RawMessage(`{"enable_search":true,"model":"evil","messages":[],"stream":true}`), "qwen-plus", "问题？")
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, true, got["enable_search"])
	require.Equal(t, "qwen-plus", got["model"])
	require.Equal(t, false, got["stream"])
	msgs := got["messages"].([]any)
	require.Len(t, msgs, 1)
	require.Equal(t, "问题？", msgs[0].(map[string]any)["content"])

	_, err = buildGeoRequestBody(json.RawMessage(`[1]`), "m", "q")
	require.ErrorIs(t, err, errGeoBadExtraBody)
}

func TestGeoScheduleDue(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	require.False(t, geoScheduleDue(GeoScheduleOff, time.Time{}, now))
	require.True(t, geoScheduleDue(GeoScheduleDaily, time.Time{}, now))
	require.True(t, geoScheduleDue(GeoScheduleDaily, now.Add(-24*time.Hour), now))
	require.True(t, geoScheduleDue(GeoScheduleDaily, now.Add(-23*time.Hour-40*time.Minute), now), "an hourly tick a little early still runs")
	require.False(t, geoScheduleDue(GeoScheduleDaily, now.Add(-20*time.Hour), now))
	require.False(t, geoScheduleDue(GeoScheduleWeekly, now.Add(-3*24*time.Hour), now))
	require.True(t, geoScheduleDue(GeoScheduleWeekly, now.Add(-7*24*time.Hour), now))
}

// ---- engines & settings ----

func TestGeoEngineKeyNeverReturned(t *testing.T) {
	s, repo, _ := newGeoTestService()
	ctx := context.Background()
	_, err := s.CreateEngine(ctx, GeoEngineInput{Name: "通义", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions", Model: "qwen-plus"})
	require.ErrorIs(t, err, errGeoEngineKey)
	_, err = s.CreateEngine(ctx, GeoEngineInput{Name: "通义", BaseURL: "ftp://x", APIKey: "k", Model: "m"})
	require.ErrorIs(t, err, errGeoBadEngine)

	v, err := s.CreateEngine(ctx, GeoEngineInput{Name: "通义", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
		APIKey: "sk-secret-123456abcd", Model: "qwen-plus", ExtraBody: json.RawMessage(`"{\"enable_search\": true}"`)})
	require.NoError(t, err)
	require.Equal(t, "https://dashscope.aliyuncs.com/compatible-mode/v1", v.BaseURL)
	require.True(t, v.HasKey)
	require.Equal(t, "••••abcd", v.KeyMasked)
	require.JSONEq(t, `{"enable_search":true}`, string(v.ExtraBody))
	b, _ := json.Marshal(v)
	require.NotContains(t, string(b), "secret")
	require.Equal(t, "enc:sk-secret-123456abcd", repo.engines[0].APIKeyEncrypted)

	// An empty key on update keeps the stored one.
	_, err = s.UpdateEngine(ctx, v.ID, GeoEngineInput{Name: "通义千问", BaseURL: v.BaseURL, Model: "qwen-max"})
	require.NoError(t, err)
	require.Equal(t, "enc:sk-secret-123456abcd", repo.engines[0].APIKeyEncrypted)
	require.Equal(t, "qwen-max", repo.engines[0].Model)
}

func TestGeoSettingsDefaultsAndValidation(t *testing.T) {
	s, _, _ := newGeoTestService()
	ctx := context.Background()
	st := s.Settings(ctx)
	require.Equal(t, GeoScheduleOff, st.Schedule)
	require.Equal(t, []string{"hivegpt", "hivegpt.cn"}, st.BrandKeywords)
	require.Equal(t, []string{}, st.CompetitorKeywords)

	_, err := s.SaveSettings(ctx, GeoSettings{Schedule: "hourly", BrandKeywords: []string{"x"}})
	require.ErrorIs(t, err, errGeoBadSettings)
	_, err = s.SaveSettings(ctx, GeoSettings{Schedule: GeoScheduleDaily, BrandKeywords: []string{" "}})
	require.ErrorIs(t, err, errGeoBadSettings)
	st, err = s.SaveSettings(ctx, GeoSettings{Schedule: GeoScheduleWeekly, BrandKeywords: []string{"HiveGPT", "hivegpt", " hivegpt.cn "}, CompetitorKeywords: []string{"OpenRouter"}})
	require.NoError(t, err)
	require.Equal(t, GeoSettings{Schedule: GeoScheduleWeekly, BrandKeywords: []string{"HiveGPT", "hivegpt.cn"}, CompetitorKeywords: []string{"OpenRouter"}}, st)
}

func TestGeoManualEntry(t *testing.T) {
	s, repo, _ := newGeoTestService()
	ctx := context.Background()
	q, err := s.CreateQuestion(ctx, GeoQuestionInput{Question: "国内怎么调用 GPT API？", Category: "接入"})
	require.NoError(t, err)
	_, err = s.AddManual(ctx, GeoManualInput{QuestionID: q.ID, EngineName: "豆包"})
	require.ErrorIs(t, err, errGeoBadManual)
	c, err := s.AddManual(ctx, GeoManualInput{QuestionID: q.ID, EngineName: " 豆包 ", Answer: "可以用中转平台。", CitedURLs: []string{"https://hivegpt.cn/learn/"}})
	require.NoError(t, err)
	require.Equal(t, GeoSourceManual, c.Source)
	require.Equal(t, "豆包", c.EngineName)
	require.Equal(t, "国内怎么调用 GPT API？", c.Question)
	require.True(t, c.Mentioned)
	require.Equal(t, []string{"https://hivegpt.cn/learn/"}, c.OurURLs)
	require.Len(t, repo.checks, 1)
}

// ---- run ----

func TestGeoRunAsksEveryPairAndRecordsErrors(t *testing.T) {
	var mu sync.Mutex
	var seen []map[string]any
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer good-key-123456", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		mu.Lock()
		seen = append(seen, m)
		mu.Unlock()
		q := m["messages"].([]any)[0].(map[string]any)["content"].(string)
		answer := "可以试试 OpenRouter。"
		if strings.Contains(q, "GPT") {
			answer = "推荐 HiveGPT。"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices":   []any{map[string]any{"message": map[string]any{"content": answer}}},
			"citations": []string{"https://hivegpt.cn/learn/connect/"},
		})
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided: bad-key-999999"}}`))
	}))
	defer bad.Close()

	s, repo, st := newGeoTestService()
	ctx := context.Background()
	_, _ = s.SaveSettings(ctx, GeoSettings{Schedule: GeoScheduleOff, BrandKeywords: []string{"hivegpt", "hivegpt.cn"}, CompetitorKeywords: []string{"openrouter"}})
	q1, _ := s.CreateQuestion(ctx, GeoQuestionInput{Question: "国内怎么调用 GPT API？"})
	_, _ = s.CreateQuestion(ctx, GeoQuestionInput{Question: "哪里能学 AI 应用开发？"})
	off := false
	_, _ = s.CreateQuestion(ctx, GeoQuestionInput{Question: "停用的问题", Enabled: &off})
	_, err := s.CreateEngine(ctx, GeoEngineInput{Name: "Perplexity", BaseURL: good.URL + "/v1", APIKey: "good-key-123456", Model: "sonar",
		ExtraBody: json.RawMessage(`{"search_mode":"web","model":"x"}`)})
	require.NoError(t, err)
	_, err = s.CreateEngine(ctx, GeoEngineInput{Name: "坏的", BaseURL: bad.URL, APIKey: "bad-key-999999", Model: "m"})
	require.NoError(t, err)
	_, err = s.CreateEngine(ctx, GeoEngineInput{Name: "停用", BaseURL: bad.URL, APIKey: "k", Model: "m", Enabled: &off})
	require.NoError(t, err)

	runID, err := s.StartRun(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, runID)
	s.runs.Wait()

	require.Len(t, repo.checks, 4) // 2 enabled questions × 2 enabled engines
	for _, m := range seen {
		require.Equal(t, "sonar", m["model"])
		require.Equal(t, "web", m["search_mode"])
		require.Equal(t, false, m["stream"])
	}
	var okCount, errCount int
	for _, c := range repo.checks {
		require.Equal(t, runID, c.RunID)
		require.Equal(t, GeoSourceAuto, c.Source)
		if c.EngineName == "坏的" {
			errCount++
			require.Contains(t, c.Error, "HTTP 401")
			require.NotContains(t, c.Error, "bad-key-999999")
			require.False(t, c.Mentioned)
			continue
		}
		okCount++
		require.Empty(t, c.Error)
		require.True(t, c.Mentioned, "the cited link is ours")
		require.Equal(t, []string{"https://hivegpt.cn/learn/connect/"}, c.OurURLs)
		if *c.QuestionID != q1.ID {
			require.Equal(t, []string{"openrouter"}, c.Competitors)
		}
	}
	require.Equal(t, 2, okCount)
	require.Equal(t, 2, errCount)

	status, err := s.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.Running)
	require.NotNil(t, status.LastRunAt)
	require.NotEmpty(t, st.m[settingGeoLastRunAt])

	// Summary: the latest result per pair, API engines first.
	sum, err := s.Summary(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, sum.Totals.Questions)
	require.Equal(t, 2, sum.Totals.Engines)
	require.Equal(t, "Perplexity", sum.Engines[0].Name)
	require.Equal(t, "2026-W41", sum.Weekly[0].Week)
	require.Equal(t, "2026-10-05", sum.Weekly[0].WeekOf)
	require.InDelta(t, 0.25, sum.Weekly[0].Rate, 1e-9)
}

func TestGeoRunOneAtATime(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	repo := &geoRepoStub{}
	s := NewGeoMonitorService(repo, &geoSettingsStub{m: map[string]string{}}, prefixEncryptor{}, nil)
	ctx := context.Background()
	_, err := s.StartRun(ctx)
	require.ErrorIs(t, err, ErrGeoNothingToRun)
	_, _ = s.CreateQuestion(ctx, GeoQuestionInput{Question: "Q"})
	_, _ = s.CreateEngine(ctx, GeoEngineInput{Name: "E", BaseURL: srv.URL, APIKey: "key-1234567890", Model: "m"})

	runID, err := s.StartRun(ctx)
	require.NoError(t, err)
	_, err = s.StartRun(ctx)
	require.ErrorIs(t, err, ErrGeoRunning)
	st, err := s.Status(ctx)
	require.NoError(t, err)
	require.True(t, st.Running)
	require.Equal(t, runID, st.RunID)
	require.Equal(t, 1, st.Total)
	require.NotNil(t, st.StartedAt)

	close(release)
	s.runs.Wait()
	st, _ = s.Status(ctx)
	require.False(t, st.Running)
	require.Len(t, repo.checks, 1)
}

func TestGeoRunScheduled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	s, repo, st := newGeoTestService()
	ctx := context.Background()
	_, _ = s.CreateQuestion(ctx, GeoQuestionInput{Question: "Q"})
	_, _ = s.CreateEngine(ctx, GeoEngineInput{Name: "E", BaseURL: srv.URL, APIKey: "key-1234567890", Model: "m"})

	require.Empty(t, s.RunScheduled(ctx), "schedule off")
	_, _ = s.SaveSettings(ctx, GeoSettings{Schedule: GeoScheduleDaily, BrandKeywords: []string{"hivegpt"}})
	require.NotEmpty(t, s.RunScheduled(ctx))
	s.runs.Wait()
	require.Empty(t, s.RunScheduled(ctx), "just ran")
	st.m[settingGeoLastRunAt] = time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339)
	require.NotEmpty(t, s.RunScheduled(ctx))
	s.runs.Wait()
	require.Len(t, repo.checks, 2)
}

func TestGeoSummaryOrdersManualEngines(t *testing.T) {
	s, repo, _ := newGeoTestService()
	ctx := context.Background()
	q, _ := s.CreateQuestion(ctx, GeoQuestionInput{Question: "Q"})
	_, _ = s.CreateEngine(ctx, GeoEngineInput{Name: "Perplexity", BaseURL: "https://api.perplexity.ai", APIKey: "key-1234567890", Model: "sonar"})
	_, _ = s.AddManual(ctx, GeoManualInput{QuestionID: q.ID, EngineName: "元宝", Answer: "不知道"})
	_, _ = s.AddManual(ctx, GeoManualInput{QuestionID: q.ID, EngineName: "豆包", Answer: "hivegpt.cn 不错"})
	require.Len(t, repo.checks, 2)
	sum, err := s.Summary(ctx)
	require.NoError(t, err)
	var names []string
	for _, e := range sum.Engines {
		names = append(names, e.Name)
	}
	require.Equal(t, "Perplexity", names[0])
	require.True(t, sort.StringsAreSorted(names[1:]))
	require.Equal(t, 2, sum.Totals.Answered)
	require.Equal(t, 1, sum.Totals.Mentioned)
	require.InDelta(t, 0.5, sum.Totals.MentionRate, 1e-9)
}
