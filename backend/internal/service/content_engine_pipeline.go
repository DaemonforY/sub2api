package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// The 内容引擎 pipeline: outline → draft → review → humanize → one version per platform → checks →
// 待审核. Each step saves its output on the article, so a run that fails resumes at the step that
// failed (contentResumeStatus) instead of paying for the earlier steps again.

const (
	contentOutlineTokens  = 4000
	contentDraftTokens    = 12000
	contentReviewTokens   = 3000
	contentHumanizeTokens = 12000
	contentAdaptTokens    = 10000
	contentTopicsTokens   = 4000
	contentMaxSearches    = 3
	contentPassages       = 4
	contentLandingChoices = 6
	contentAdaptParallel  = 2
	contentSuggestCount   = 8
	contentMinDraftChars  = 300
)

var contentJSONFence = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
var contentMDFence = regexp.MustCompile("(?s)^```(?:markdown|md)?\\s*\n(.*?)\n```$")

// contentParseJSON reads the JSON object in a model's answer (fences and chatter around it are
// tolerated).
func contentParseJSON(text string, v any) error {
	text = strings.TrimSpace(text)
	if m := contentJSONFence.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	if i, j := strings.Index(text, "{"), strings.LastIndex(text, "}"); i >= 0 && j > i {
		text = text[i : j+1]
	}
	if err := json.Unmarshal([]byte(text), v); err != nil {
		return errors.New("JSON 解析失败")
	}
	return nil
}

// contentCleanMarkdown drops a ```markdown fence the model may wrap the article in.
func contentCleanMarkdown(text string) string {
	text = strings.TrimSpace(text)
	if m := contentMDFence.FindStringSubmatch(text); m != nil {
		text = strings.TrimSpace(m[1])
	}
	return text
}

var contentH1 = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// contentRun is what every step of one run needs.
type contentRun struct {
	key      string
	model    string
	product  string
	style    string
	brand    string
	rules    []string
	passages []assistantHit
	usageMu  sync.Mutex
}

func (s *ContentEngineService) newRun(ctx context.Context, it *ContentItem, key string) (*contentRun, *ContentOwnerRow, error) {
	row, err := s.loadRow(ctx, it.OwnerUserID)
	if err != nil {
		return nil, nil, err
	}
	st := row.Settings
	r := &contentRun{key: key, model: s.model(ctx, st), product: st.Product, style: st.StylePrompt, brand: s.siteName(ctx),
		rules: s.activeRules(ctx, it.OwnerUserID)}
	if s.sources.Search != nil {
		r.passages = s.sources.Search(it.Brief.Topic+" "+it.Brief.Angle, contentPassages)
	}
	return r, row, nil
}

// call runs one model call (with tools, up to calls model calls) and counts the tokens.
func (s *ContentEngineService) call(ctx context.Context, it *ContentItem, r *contentRun, msgs []AgentMessage, maxTokens int, tools []AgentTool, calls int) (string, error) {
	if calls <= 0 {
		calls = 1
	}
	res, err := s.agent(ctx, AgentRunInput{Key: r.key, Model: r.model, Messages: msgs, Tools: tools, MaxModelCalls: calls, MaxOutputTokens: maxTokens,
		OnTool: func(_, _ string) error { return ctx.Err() }})
	if it != nil && res != nil {
		r.usageMu.Lock()
		s.usage(it, res)
		r.usageMu.Unlock()
	}
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

// callJSON is call for an answer that must be a JSON object: one more try, without tools, showing
// the model what went wrong.
func (s *ContentEngineService) callJSON(ctx context.Context, it *ContentItem, r *contentRun, system, user string, maxTokens int, tools []AgentTool, calls int, out any) error {
	msgs := []AgentMessage{agentText("system", system), agentText("user", user)}
	text, err := s.call(ctx, it, r, msgs, maxTokens, tools, calls)
	if err != nil {
		return err
	}
	perr := contentParseJSON(text, out)
	if perr == nil {
		return nil
	}
	msgs = append(msgs, agentText("assistant", truncateRunes(text, 6000)), agentText("user", "上面的输出不是要求的 JSON（"+perr.Error()+"）。只输出一个合法的 JSON 对象，不要任何其他文字。"))
	if text, err = s.call(ctx, it, r, msgs, maxTokens, nil, 1); err != nil {
		return err
	}
	if contentParseJSON(text, out) != nil {
		return errLearnRunFailed("模型没有给出可用的结果")
	}
	return nil
}

func (s *ContentEngineService) step(it *ContentItem, status, text string) {
	it.Status = status
	s.event(it, "step", text)
	s.save(it)
}

// runPipeline runs the steps still to do.
func (s *ContentEngineService) runPipeline(ctx context.Context, it *ContentItem, key string) error {
	r, row, err := s.newRun(ctx, it, key)
	if err != nil {
		return err
	}
	if it.Outline == nil {
		if err := s.runOutline(ctx, it, r); err != nil {
			return err
		}
	}
	if it.Draft == "" {
		s.step(it, ContentDrafting, "写初稿…")
		text, err := s.call(ctx, it, r, []AgentMessage{agentText("system", contentDraftPrompt(r.product, r.style, r.rules)),
			agentText("user", contentDraftRequest(it, r.passages))}, contentDraftTokens, nil, 1)
		if err != nil {
			return err
		}
		if text = contentCleanMarkdown(text); runeLen(text) < contentMinDraftChars {
			return errLearnRunFailed("初稿太短，点「重试」再写一次")
		}
		it.Draft = text
		s.save(it)
	}
	if !it.Reviewed {
		s.step(it, ContentReviewing, "审校…")
		var rev struct {
			Notes   []string `json:"notes"`
			AIScore int      `json:"ai_score"`
		}
		if err := s.callJSON(ctx, it, r, contentReviewPrompt, contentReviewRequest(it, r.passages), contentReviewTokens, nil, 1, &rev); err != nil {
			return err
		}
		it.ReviewNotes = []string{}
		for _, n := range rev.Notes {
			if n = truncateRunes(strings.TrimSpace(n), 300); n != "" && len(it.ReviewNotes) < 12 {
				it.ReviewNotes = append(it.ReviewNotes, n)
			}
		}
		it.AIScore, it.Reviewed = max(0, min(rev.AIScore, 100)), true
		s.event(it, "info", fmt.Sprintf("审校意见 %d 条", len(it.ReviewNotes)))
		s.save(it)
	}
	if it.Markdown == "" {
		s.step(it, ContentHumanizing, "按审校意见修改、去 AI 味…")
		md, err := s.humanize(ctx, it, r, it.Draft, it.ReviewNotes)
		if err != nil {
			return err
		}
		// Too many 套话 left: one more pass naming them.
		if hits := contentBannedHits(md); len(hits) >= contentBannedWarnAt {
			s.event(it, "step", "套话还偏多，再改一遍…")
			if md2, err := s.humanize(ctx, it, r, md, []string{"这些 AI 套话还在，全部换成具体的说法：" + strings.Join(contentUnique(hits, 10), "、")}); err == nil {
				md = md2
			}
		}
		it.Markdown = md
		if m := contentH1.FindStringSubmatch(md); m != nil {
			it.Title = truncateRunes(strings.TrimSpace(m[1]), 200)
		}
		s.save(it)
	}
	s.step(it, ContentAdapting, "改写成各平台版本…")
	if err := s.adaptAll(ctx, it, r); err != nil {
		return err
	}
	earlier, _ := s.repo.RecentTitles(ctx, it.OwnerUserID, 80)
	it.Checks = checkContentMaster(it, earlier)
	it.Status, it.Error = ContentPending, ""
	s.event(it, "done", "已生成，等你审核")
	s.save(it)
	s.notifyPending(ctx, row, it)
	return nil
}

// humanize rewrites text along the notes, in the owner's style, without the AI flavour.
func (s *ContentEngineService) humanize(ctx context.Context, it *ContentItem, r *contentRun, text string, notes []string) (string, error) {
	text, err := s.call(ctx, it, r, []AgentMessage{agentText("system", contentHumanizePrompt(r.product, r.style, r.rules)),
		agentText("user", contentHumanizeRequest(text, notes))}, contentHumanizeTokens, nil, 1)
	if err != nil {
		return "", err
	}
	if text = contentCleanMarkdown(text); runeLen(text) < contentMinDraftChars {
		return "", errLearnRunFailed("修改后的稿子太短，点「重试」再来一次")
	}
	return text, nil
}

func (s *ContentEngineService) runOutline(ctx context.Context, it *ContentItem, r *contentRun) error {
	landing := s.landingChoices(it)
	it.Materials = landing
	var tools []AgentTool
	calls := 1
	if it.Brief.Search && s.searchAvailable(ctx) {
		s.step(it, ContentOutlining, "搜索资料、构思大纲…")
		tools, calls = []AgentTool{s.searchTool(it, false)}, contentMaxSearches+1
	} else {
		s.step(it, ContentOutlining, "构思大纲…")
	}
	var o ContentOutline
	if err := s.callJSON(ctx, it, r, contentOutlinePrompt(r.product, tools != nil), contentOutlineRequest(it, r.passages, landing),
		contentOutlineTokens, tools, calls, &o); err != nil {
		return err
	}
	o.Title = truncateRunes(strings.TrimSpace(o.Title), 64)
	if o.Title == "" {
		o.Title = truncateRunes(it.Brief.Topic, 64)
	}
	sections := []ContentSection{}
	for _, sec := range o.Sections {
		h := truncateRunes(strings.TrimSpace(sec.Heading), 60)
		if h == "" || len(sections) >= 8 {
			continue
		}
		points := []string{}
		for _, p := range sec.Points {
			if p = truncateRunes(strings.TrimSpace(p), 200); p != "" && len(points) < 6 {
				points = append(points, p)
			}
		}
		sections = append(sections, ContentSection{Heading: h, Points: points})
	}
	if len(sections) == 0 {
		return errLearnRunFailed("模型没有给出可用的大纲")
	}
	o.Sections = sections
	o.LandingPath = contentPickLanding(o.LandingPath, landing)
	it.Outline, it.LandingPath, it.Title = &o, o.LandingPath, o.Title
	s.event(it, "info", "大纲："+o.Title)
	s.save(it)
	return nil
}

// landingChoices: learning-site pages related to the topic, as landing page candidates.
func (s *ContentEngineService) landingChoices(it *ContentItem) []ContentLanding {
	out := []ContentLanding{}
	if s.sources.Search == nil {
		return out
	}
	seen := map[string]bool{}
	for _, h := range s.sources.Search(it.Brief.Topic+" "+it.Brief.Angle, contentLandingChoices*2) {
		path := h.URL
		if u, err := url.Parse(h.URL); err == nil && u.Path != "" {
			path = u.Path
		}
		if seen[path] || path == "/" || !channelLinkPathOK(path) || len(out) >= contentLandingChoices {
			continue
		}
		seen[path] = true
		out = append(out, ContentLanding{Path: path, Title: truncateRunes(h.Title, 60)})
	}
	return out
}

// contentPickLanding keeps the model's landing page only when it is one of the candidates.
func contentPickLanding(path string, choices []ContentLanding) string {
	path = strings.TrimSpace(path)
	for _, c := range choices {
		if c.Path == path {
			return path
		}
	}
	return "/"
}

// searchTool searches the web, keeping what it finds as the article's sources; quiet leaves out
// the progress events (topic suggestions have no article to show them on).
func (s *ContentEngineService) searchTool(it *ContentItem, quiet bool) AgentTool {
	seen := map[string]bool{}
	for _, src := range it.Sources {
		seen[src.URL] = true
	}
	return AgentTool{
		Name:        "web_search",
		Description: fmt.Sprintf("联网搜索最新资料。每次一个查询，最多 %d 次；用具体关键词，中文选题优先用中文。", contentMaxSearches),
		Label:       "正在搜索",
		Parameters: map[string]any{"type": "object", "required": []string{"query"}, "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索关键词"}}},
		Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var args struct {
				Query string `json:"query"`
			}
			_ = json.Unmarshal(raw, &args)
			q := truncateRunes(strings.TrimSpace(args.Query), 200)
			if q == "" {
				return nil, errors.New("query 不能为空")
			}
			if it.Searches >= contentMaxSearches*2 {
				return nil, errors.New("搜索次数已用完，请根据已有资料继续")
			}
			it.Searches++
			if !quiet {
				s.event(it, "step", "搜索：「"+q+"」")
				s.save(it)
			}
			results, err := s.search.Search(ctx, q, 6)
			if err != nil {
				return nil, errors.New("搜索服务暂时不可用，请根据已有资料继续")
			}
			out := []map[string]string{}
			for _, res := range results {
				out = append(out, map[string]string{"title": res.Title, "url": res.URL, "snippet": truncateRunes(res.Snippet, 400), "age": res.PageAge})
				if res.URL != "" && !seen[res.URL] && len(it.Sources) < 20 {
					seen[res.URL] = true
					it.Sources = append(it.Sources, ArticleSource{Title: truncateRunes(res.Title, 120), URL: res.URL, Snippet: truncateRunes(res.Snippet, 400)})
				}
			}
			return map[string]any{"query": q, "results": out}, nil
		},
	}
}

// --- platform versions -------------------------------------------------------------------------

type contentAdaptOut struct {
	Kind     string        `json:"kind"`
	Title    string        `json:"title"`
	Question string        `json:"question"`
	Digest   string        `json:"digest"`
	Body     string        `json:"body"`
	Micro    string        `json:"micro"`
	Tags     []string      `json:"tags"`
	Category string        `json:"category"`
	Cards    []ContentCard `json:"cards"`
}

func contentKind(platform, kind string) string {
	switch platform {
	case ContentPlatformZhihu:
		if kind == "answer" {
			return "answer"
		}
		return "article"
	case ContentPlatformXHS:
		return "note"
	default:
		return "article"
	}
}

// writeVariant writes one platform's version (two tries) into v.
func (s *ContentEngineService) writeVariant(ctx context.Context, it *ContentItem, r *contentRun, v *ContentVariant, feedback string) error {
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		var out contentAdaptOut
		if err := s.callJSON(ctx, it, r, contentAdaptPrompt(v.Platform, r.product, r.style, r.brand, r.rules), contentAdaptRequest(it, feedback),
			contentAdaptTokens, nil, 1, &out); err != nil {
			if ctx.Err() != nil {
				return err
			}
			last = err
			continue
		}
		out.Body = contentCleanMarkdown(out.Body)
		if strings.TrimSpace(out.Title) == "" || runeLen(out.Body) < 50 {
			last = errLearnRunFailed("模型给的版本缺标题或正文")
			continue
		}
		v.Kind = contentKind(v.Platform, out.Kind)
		v.Title = truncateRunes(strings.TrimSpace(out.Title), 200)
		v.Body = truncateRunes(out.Body, contentMaxBody)
		v.Question = ""
		if v.Kind == "answer" {
			v.Question = truncateRunes(strings.TrimSpace(out.Question), 200)
		}
		v.Digest = truncateRunes(strings.TrimSpace(out.Digest), 120)
		v.Micro = truncateRunes(strings.TrimSpace(out.Micro), 1000)
		v.Category = truncateRunes(strings.TrimSpace(out.Category), 50)
		v.Tags, v.Cards = cleanContentTags(out.Tags), nil
		if contentPlatformSpecs[v.Platform].Cards {
			v.Cards = cleanContentCards(out.Cards)
		}
		v.Edited, v.Error, v.Feedback = false, "", feedback
		return nil
	}
	return last
}

// adaptAll writes the versions still missing (or failed), then checks them all together.
func (s *ContentEngineService) adaptAll(ctx context.Context, it *ContentItem, r *contentRun) error {
	existing, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return err
	}
	have := map[string]ContentVariant{}
	for _, v := range existing {
		have[v.Platform] = v
	}
	var todo []*ContentVariant
	for _, p := range it.Brief.Platforms {
		if _, ok := contentPlatformSpecs[p]; !ok {
			continue
		}
		if v, ok := have[p]; ok && v.Status != ContentVariantFailed {
			continue
		}
		v := &ContentVariant{ItemID: it.ID, OwnerUserID: it.OwnerUserID, Platform: p, Kind: contentKind(p, ""), Status: ContentVariantGenerating}
		if old, ok := have[p]; ok {
			v.ID = old.ID
		}
		if err := s.repo.UpsertVariant(ctx, v); err != nil {
			return err
		}
		todo = append(todo, v)
	}
	g := new(errgroup.Group)
	g.SetLimit(contentAdaptParallel)
	for _, v := range todo {
		g.Go(func() error {
			if err := s.writeVariant(ctx, it, r, v, ""); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				v.Status, v.Error = ContentVariantFailed, contentErrorText(it.ID, err)
			} else {
				v.Status = ContentVariantPending
			}
			if err := s.saveVariant(v); err != nil {
				slog.Warn("content: save version", "item", it.ID, "platform", v.Platform, "err", err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	all, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return err
	}
	ok := 0
	for i := range all {
		v := &all[i]
		if v.Status == ContentVariantFailed {
			s.event(it, "error", contentPlatformSpecs[v.Platform].Name+"版本没写成："+v.Error)
			continue
		}
		ok++
		if v.Status != ContentVariantPending {
			continue // approved or published versions stay as they are
		}
		v.Checks = checkContentVariant(v, all, r.brand)
		if err := s.saveVariant(v); err != nil {
			return err
		}
	}
	if ok == 0 {
		return errLearnRunFailed("各平台版本都没写成功，点「重试」再来一次")
	}
	return nil
}

// regenerate writes one version again with the owner's feedback. Failing leaves the article as it
// was, with the version marked failed.
func (s *ContentEngineService) regenerate(ctx context.Context, it *ContentItem, key, platform, feedback string) error {
	r, _, err := s.newRun(ctx, it, key)
	if err != nil {
		return err
	}
	v, err := s.findVariant(ctx, it, platform)
	if err != nil {
		return err
	}
	s.event(it, "step", "重写"+contentPlatformSpecs[platform].Name+"版本…")
	s.save(it)
	if err := s.writeVariant(ctx, it, r, v, feedback); err != nil {
		if ctx.Err() != nil {
			return err
		}
		v.Status, v.Error = ContentVariantFailed, contentErrorText(it.ID, err)
		s.event(it, "error", contentPlatformSpecs[platform].Name+"版本没写成："+v.Error)
		return s.saveVariant(v)
	}
	all, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return err
	}
	v.Status = ContentVariantPending
	v.Checks = checkContentVariant(v, all, r.brand)
	if err := s.saveVariant(v); err != nil {
		return err
	}
	if it.Status == ContentDone {
		it.Status = ContentPending
	}
	s.event(it, "done", contentPlatformSpecs[platform].Name+"版本已重写")
	return nil
}

func (s *ContentEngineService) findVariant(ctx context.Context, it *ContentItem, platform string) (*ContentVariant, error) {
	vs, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return nil, err
	}
	for i := range vs {
		if vs[i].Platform == platform {
			return &vs[i], nil
		}
	}
	return nil, ErrContentNotFound
}

func (s *ContentEngineService) notifyPending(ctx context.Context, row *ContentOwnerRow, it *ContentItem) {
	if row == nil || row.Settings.NotifyType == "" || row.NotifyKeyEnc == "" {
		return
	}
	vs, _ := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	var names []string
	warnings := len(it.Checks)
	for _, v := range vs {
		if v.Status == ContentVariantPending {
			names = append(names, contentPlatformSpecs[v.Platform].Name)
		}
		for _, c := range v.Checks {
			if c.Level != "info" {
				warnings++
			}
		}
	}
	body := fmt.Sprintf("「%s」写好了：%s。\n\n自动检查有 %d 条需要看一下。用了 %d 个 token。", it.Title, strings.Join(names, "、"), warnings, it.PromptTokens+it.CompletionTokens)
	if err := s.push(context.WithoutCancel(ctx), row, "【待审核】"+truncateRunes(it.Title, 40), body, fmt.Sprintf("%s/admin/content?item=%d", s.siteURL(ctx), it.ID)); err != nil {
		slog.Warn("content: push", "item", it.ID, "err", err)
	}
}

// --- topic suggestions -------------------------------------------------------------------------

// SuggestTopics asks the model for new topics from the support questions, the GEO questions, the
// learning site and the web, and adds them to the pool.
func (s *ContentEngineService) SuggestTopics(ctx context.Context, owner int64) ([]ContentTopic, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return nil, err
	}
	key, err := s.key(ctx, owner, row.Settings.KeyID)
	if err != nil {
		return nil, err
	}
	done, err := s.repo.RecentTitles(ctx, owner, 80)
	if err != nil {
		return nil, err
	}
	r := &contentRun{key: key.Key, model: s.model(ctx, row.Settings), product: row.Settings.Product}
	search := row.Settings.Search && s.searchAvailable(ctx)
	var tools []AgentTool
	calls := 1
	scratch := &ContentItem{OwnerUserID: owner, ContentItemData: ContentItemData{Sources: []ArticleSource{}, Events: []ArticleEvent{}}}
	if search {
		tools, calls = []AgentTool{s.searchTool(scratch, true)}, 3
	}
	var out struct {
		Topics []struct {
			Title  string `json:"title"`
			Angle  string `json:"angle"`
			Source string `json:"source"`
			Reason string `json:"reason"`
			Score  int    `json:"score"`
		} `json:"topics"`
	}
	if err := s.callJSON(ctx, scratch, r, contentTopicsPrompt(r.product, row.Settings.Platforms, search, contentSuggestCount),
		contentTopicsRequest(s.supportQuestions(ctx), s.geoQuestions(ctx), s.catalogSample(), done), contentTopicsTokens, tools, calls, &out); err != nil {
		return nil, err
	}
	added := []ContentTopic{}
	taken := append([]string(nil), done...)
	for _, t := range out.Topics {
		title := truncateRunes(strings.TrimSpace(t.Title), contentMaxTopic)
		if title == "" || contentSimilarTitle(title, taken) || len(added) >= contentSuggestCount {
			continue
		}
		source := t.Source
		switch source {
		case "support", "geo", "learn", "hot", "idea":
		default:
			source = "idea"
		}
		topic := ContentTopic{OwnerUserID: owner, Title: title, Angle: truncateRunes(strings.TrimSpace(t.Angle), contentMaxAngle), Source: source,
			Reason: truncateRunes(strings.TrimSpace(t.Reason), 500), Score: max(0, min(t.Score, 100)), Status: ContentTopicPending}
		if err := s.repo.CreateTopic(ctx, &topic); err != nil {
			return nil, err
		}
		taken = append(taken, title)
		added = append(added, topic)
	}
	slog.Info("content: topics suggested", "owner", owner, "added", len(added), "tokens", scratch.PromptTokens+scratch.CompletionTokens)
	return added, nil
}

func contentSimilarTitle(title string, taken []string) bool {
	for _, t := range taken {
		if t == title || contentJaccard(t, title) >= contentHistoryWarnAt {
			return true
		}
	}
	return false
}

func (s *ContentEngineService) supportQuestions(ctx context.Context) []string {
	if s.sources.Runs == nil {
		return nil
	}
	runs, _, err := s.sources.Runs.List(ctx, AgentSupport, 1, 60)
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, r := range runs {
		q := truncateRunes(strings.TrimSpace(r.Question), 120)
		if q == "" || seen[q] || runeLen(q) < 6 || len(out) >= 40 {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}
	return out
}

func (s *ContentEngineService) geoQuestions(ctx context.Context) []string {
	if s.sources.Geo == nil {
		return nil
	}
	qs, err := s.sources.Geo.ListQuestions(ctx)
	if err != nil {
		return nil
	}
	var out []string
	for _, q := range qs {
		if q.Enabled && len(out) < 30 {
			out = append(out, truncateRunes(q.Question, 120))
		}
	}
	return out
}

// catalogSample: up to 40 learning-site page titles, a different sample each time.
func (s *ContentEngineService) catalogSample() []string {
	if s.sources.Catalog == nil {
		return nil
	}
	pages := s.sources.Catalog()
	rand.Shuffle(len(pages), func(i, j int) { pages[i], pages[j] = pages[j], pages[i] })
	var out []string
	for _, p := range pages {
		if len(out) >= 40 {
			break
		}
		out = append(out, fmt.Sprintf("%s（%s）", truncateRunes(p.Title, 60), p.URL))
	}
	return out
}
