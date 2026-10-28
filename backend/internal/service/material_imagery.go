package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Imagery is the bridge from words to pictures: 孤烟, 柳, 孤灯 — what each means and how to paint it.
// A text model drafts entries (seeds, or new imagery found in a segment); an admin reviews them before
// they are used in prompts. Segments are tagged by the model too, and an admin confirms the tags.

const (
	materialImageryTimeout   = 3 * time.Minute
	materialSeedBatch        = 10 // imagery per model call
	materialMaxSeedRequest   = 30 // per request; the doc reviews seeds 30 at a time
	materialMaxSegmentLinks  = 8
	materialMaxImageryName   = 8 // runes
	materialMaxImageryField  = 600
	materialMaxContextRunes  = 1500
	materialMaxImageryPage   = 500
	materialSeedConcurrency  = 3
	materialMaxImageryAlias  = 6
	materialMaxImageryLabels = 6
)

var (
	ErrMaterialImageryNotFound = infraerrors.NotFound("MATERIAL_IMAGERY_NOT_FOUND", "找不到这个意象（Imagery not found）")
	ErrMaterialSegmentNotFound = infraerrors.NotFound("MATERIAL_SEGMENT_NOT_FOUND", "找不到这个片段（Segment not found）")
	ErrMaterialModelAnswer     = infraerrors.BadRequest("MATERIAL_MODEL_ANSWER", "文本模型没有返回可用的结果，请重试（The text model returned nothing usable）")
	ErrMaterialNoChat          = infraerrors.BadRequest("MATERIAL_NO_CHAT", "文本模型不可用，请先在「课本诗画 → 设置」里选一把 API Key（No text model configured）")
	ErrMaterialImageryName     = infraerrors.BadRequest("MATERIAL_IMAGERY_NAME", "意象名称要 1–8 个字（Imagery name must be 1–8 characters）")
)

// MaterialImageryCategories are the allowed mat_imagery.category values.
var MaterialImageryCategories = []string{"天象", "气候", "地理", "植物", "动物", "器物", "建筑", "人物动作"}

var materialImageryCategorySet = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range MaterialImageryCategories {
		m[c] = true
	}
	return m
}()

var materialStatuses = map[string]bool{
	MaterialStatusDraft: true, MaterialStatusReview: true, MaterialStatusPublished: true, MaterialStatusHidden: true,
}

type MaterialImagery struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Aliases   []string  `json:"aliases"`
	Category  string    `json:"category"`
	Meaning   string    `json:"meaning"`
	VisualZh  string    `json:"visual_zh"`
	VisualEn  string    `json:"visual_en"`
	Emotions  []string  `json:"emotions"`
	Seasons   []string  `json:"seasons"`
	Times     []string  `json:"times"`
	Weathers  []string  `json:"weathers"`
	Eras      []string  `json:"eras"`
	StyleIDs  []int64   `json:"style_ids"` // styles an admin recommends for this imagery
	Status    string    `json:"status"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Lists fill SegmentCount: how many segments are tagged with it.
	SegmentCount int `json:"segment_count"`
}

type MaterialImageryQuery struct {
	Status   string
	Category string
	Q        string
	Limit    int
	Offset   int
}

// MaterialSegmentImagery is one tag of a segment.
type MaterialSegmentImagery struct {
	SegmentID int64  `json:"segment_id"`
	ImageryID int64  `json:"imagery_id"`
	Weight    int    `json:"weight"`
	Source    string `json:"source"` // ai | manual
	Confirmed bool   `json:"confirmed"`
	// Joined from mat_imagery.
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"`
	VisualZh string `json:"visual_zh"`
}

// MaterialSegmentContext is a segment with what a prompt about it needs from its text.
type MaterialSegmentContext struct {
	MaterialSegment
	Kind          string `json:"kind"`
	Title         string `json:"title"`
	AuthorName    string `json:"author_name"`
	Dynasty       string `json:"dynasty"`
	LicenseStatus string `json:"license_status"`
	Body          string `json:"body"`
}

// materialChatFunc runs one text-model call; it returns the answer and the model used.
type materialChatFunc func(ctx context.Context, system, user string) (string, string, error)

//go:embed material_imagery_candidates.json
var materialImageryCandidatesJSON []byte

// MaterialImageryCandidate is a seed candidate ranked by how many poems use it
// (tools/material_imagery_freq.py over open-chinese/poetry-collection).
type MaterialImageryCandidate struct {
	Rank      int      `json:"rank"`
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Aliases   []string `json:"aliases"`
	Poems     int      `json:"poems"`
	InLibrary bool     `json:"in_library"`
}

var materialCandidates = sync.OnceValue(func() []MaterialImageryCandidate {
	var doc struct {
		Items []MaterialImageryCandidate `json:"items"`
	}
	if err := json.Unmarshal(materialImageryCandidatesJSON, &doc); err != nil {
		panic("material_imagery_candidates.json: " + err.Error())
	}
	return doc.Items
})

// ---- normalisation ----

func materialTrimRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func materialCleanList(in []string, maxItems, maxRunes int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range in {
		v = materialTrimRunes(v, maxRunes)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
		if len(out) == maxItems {
			break
		}
	}
	return out
}

// normalizeMaterialImagery trims every field; it fails when the name is missing or too long.
func normalizeMaterialImagery(im *MaterialImagery) error {
	im.Name = strings.TrimSpace(im.Name)
	if n := utf8.RuneCountInString(im.Name); n == 0 || n > materialMaxImageryName {
		return ErrMaterialImageryName
	}
	im.Aliases = materialCleanList(im.Aliases, materialMaxImageryAlias, materialMaxImageryName)
	for i, a := range im.Aliases {
		if a == im.Name {
			im.Aliases = append(im.Aliases[:i], im.Aliases[i+1:]...)
			break
		}
	}
	if !materialImageryCategorySet[im.Category] {
		im.Category = ""
	}
	im.Meaning = materialTrimRunes(im.Meaning, materialMaxImageryField)
	im.VisualZh = materialTrimRunes(im.VisualZh, materialMaxImageryField)
	im.VisualEn = materialTrimRunes(im.VisualEn, materialMaxImageryField)
	im.Emotions = materialCleanList(im.Emotions, materialMaxImageryLabels, 8)
	im.Seasons = materialCleanList(im.Seasons, 4, 4)
	im.Times = materialCleanList(im.Times, 4, 8)
	im.Weathers = materialCleanList(im.Weathers, 4, 8)
	im.Eras = materialCleanList(im.Eras, 4, 16)
	ids := []int64{}
	seen := map[int64]bool{}
	for _, id := range im.StyleIDs {
		if id > 0 && !seen[id] && len(ids) < materialMaxImageryLabels {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	im.StyleIDs = ids
	if !materialStatuses[im.Status] {
		im.Status = MaterialStatusDraft
	}
	return nil
}

// materialImageryIndex maps names and aliases to imagery ids.
type materialImageryIndex map[string]int64

func newMaterialImageryIndex(all []MaterialImagery) materialImageryIndex {
	idx := materialImageryIndex{}
	for _, im := range all {
		idx[im.Name] = im.ID
	}
	for _, im := range all { // names win over aliases
		for _, a := range im.Aliases {
			if _, ok := idx[a]; !ok {
				idx[a] = im.ID
			}
		}
	}
	return idx
}

// parseMaterialJSON reads the model's JSON object, tolerating a code fence or a sentence around it.
func parseMaterialJSON(answer string, v any) error {
	start, end := strings.Index(answer, "{"), strings.LastIndex(answer, "}")
	if start < 0 || end <= start {
		return ErrMaterialModelAnswer
	}
	if err := json.Unmarshal([]byte(answer[start:end+1]), v); err != nil {
		return ErrMaterialModelAnswer
	}
	return nil
}

// materialDraftEntry is how the model writes an imagery entry (seeds and new imagery in segments).
type materialDraftEntry struct {
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases"`
	Category string   `json:"category"`
	Meaning  string   `json:"meaning"`
	VisualZh string   `json:"visual_zh"`
	VisualEn string   `json:"visual_en"`
	Emotions []string `json:"emotions"`
	Seasons  []string `json:"seasons"`
	Times    []string `json:"times"`
	Weathers []string `json:"weathers"`
	Eras     []string `json:"eras"`
}

func (e materialDraftEntry) imagery() MaterialImagery {
	return MaterialImagery{Name: e.Name, Aliases: e.Aliases, Category: e.Category, Meaning: e.Meaning, VisualZh: e.VisualZh,
		VisualEn: e.VisualEn, Emotions: e.Emotions, Seasons: e.Seasons, Times: e.Times, Weathers: e.Weathers, Eras: e.Eras,
		Status: MaterialStatusDraft}
}

// screenMaterialDraft drops an English visual that names an artist, studio or character; the admin
// rewrites it during review. It returns the blocklist hits for the result.
func screenMaterialDraft(im *MaterialImagery, terms []MaterialBlockTerm) []MaterialBlockHit {
	res := CheckMaterialBlocklist(im.VisualEn, "prompt", terms)
	if res.Blocked {
		im.VisualEn = ""
	} else {
		im.VisualEn = res.Prompt
	}
	return res.Hits
}

// materialModelError reports a failed text-model call to the admin. Plain errors (network, upstream
// status) keep their text, which the generic "internal error" message would hide.
func materialModelError(err error) error {
	msg := infraerrors.Message(err)
	var app *infraerrors.ApplicationError
	if !errors.As(err, &app) {
		msg = materialTrimRunes(err.Error(), 300)
	}
	return infraerrors.BadRequest("MATERIAL_MODEL_ANSWER", "文本模型调用失败："+msg)
}

// ---- prompts ----

const materialDraftFields = `每条字段：
- name：意象名称（原样返回给定名称；新建时用 1–4 个字的常用词）
- aliases：常见同义写法，0–4 个
- category：天象 / 气候 / 地理 / 植物 / 动物 / 器物 / 建筑 / 人物动作 之一
- meaning：文化含义与常见用法，一两句，用你自己的话
- visual_zh：画面里具体画什么、怎么画，一两句
- visual_en：英文画面描述短语，15–40 个词；不出现任何人名、画家、工作室或作品名，不要求画出文字
- emotions：常带的情绪，1–4 个，如 离别、思乡、孤独、壮阔、闲适、怀古
- seasons：春 / 夏 / 秋 / 冬，可为空数组
- times：清晨 / 白天 / 黄昏 / 夜晚，可为空数组
- weathers：晴 / 雨 / 雪 / 雾 / 风，可为空数组
- eras：有明确时代特征时写朝代范围（如 ["唐","宋"]），否则空数组`

const materialSeedSystemPrompt = `你是中国古典诗词与中国画的美术指导，正在为「诗画素材库」编写意象条目。每个意象是古诗里常见、能画出来的事物，条目用来指导 AI 绘画。

` + materialDraftFields + `

只输出一个 JSON 对象，不要任何其它文字：{"items":[{...}, ...]}，条目顺序与给定名称一致。`

const materialExtractSystemPrompt = `你是中国古典诗词与中国画的美术指导。任务：找出给定片段里能画出来的「意象」，并对应到意象库。

规则：
1. 只取片段里出现或直接描写的、画面上看得见的事物（天象、气候、地理、植物、动物、器物、建筑、人物动作）。抽象情绪、典故人名不算意象。
2. 优先用「已有意象」里的名称（括号里是别名，返回主名称）；没有合适的才新建，新建的 new 为 true 并补全字段。
3. weight：3 = 画面主体，2 = 重要陪衬，1 = 点缀。每段 2–6 个意象。
4. scene_desc：用你自己的话写这一段的画面，60–120 字，只写看得见的东西（季节、时辰、天气、主体、远近层次），不引用、不复述原句。

新建意象的字段：
` + materialDraftFields + `

只输出一个 JSON 对象，不要任何其它文字：
{"imagery":[{"name":"…","weight":3,"new":false},{"name":"…","weight":2,"new":true,"category":"…","meaning":"…","visual_zh":"…","visual_en":"…","emotions":["…"],"seasons":[]}],"scene_desc":"…"}`

func materialSeedUserPrompt(batch []MaterialImageryCandidate) string {
	var b strings.Builder
	_, _ = b.WriteString("请为下列意象各写一条（括号里是语料中常见的写法，可作 aliases 参考）：\n")
	for _, c := range batch {
		if len(c.Aliases) > 0 {
			fmt.Fprintf(&b, "- %s（%s）\n", c.Name, strings.Join(c.Aliases, "、"))
		} else {
			fmt.Fprintf(&b, "- %s\n", c.Name)
		}
	}
	return b.String()
}

func materialExtractUserPrompt(seg *MaterialSegmentContext, all []MaterialImagery) string {
	var b strings.Builder
	fmt.Fprintf(&b, "篇名：%s\n作者：%s %s\n体裁：%s\n", seg.Title, seg.Dynasty, seg.AuthorName, seg.Kind)
	if MaterialTextOverlayAllowed(seg.LicenseStatus) && seg.Content != "" {
		fmt.Fprintf(&b, "要分析的片段（%s）：\n%s\n", seg.Label, seg.Content)
		if body := materialTrimRunes(seg.Body, materialMaxContextRunes); body != "" && body != seg.Content {
			fmt.Fprintf(&b, "\n全文供理解：\n%s\n", body)
		}
	} else {
		// reference_only: no original words are stored, only our own description.
		fmt.Fprintf(&b, "要分析的片段（%s）的画面描述：\n%s\n", seg.Label, seg.SceneDesc)
	}
	_, _ = b.WriteString("\n已有意象：")
	n := 0
	for _, im := range all {
		if im.Status == MaterialStatusHidden {
			continue
		}
		if n > 0 {
			_, _ = b.WriteString("、")
		}
		_, _ = b.WriteString(im.Name)
		if len(im.Aliases) > 0 {
			fmt.Fprintf(&b, "（%s）", strings.Join(im.Aliases, "/"))
		}
		n++
	}
	if n == 0 {
		_, _ = b.WriteString("（暂无）")
	}
	_, _ = b.WriteString("\n")
	return b.String()
}

// ---- service ----

func (s *MaterialService) chatOrErr() (materialChatFunc, error) {
	if s.chat == nil {
		return nil, ErrMaterialNoChat
	}
	return s.chat, nil
}

func (s *MaterialService) ListImagery(ctx context.Context, q MaterialImageryQuery) ([]MaterialImagery, int, error) {
	if q.Limit <= 0 || q.Limit > materialMaxImageryPage {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	return s.repo.ListImagery(ctx, q)
}

// UpdateImagery saves an admin's edit of an entry.
func (s *MaterialService) UpdateImagery(ctx context.Context, im *MaterialImagery) (*MaterialImagery, error) {
	if err := normalizeMaterialImagery(im); err != nil {
		return nil, err
	}
	ok, err := s.repo.UpdateImagery(ctx, im)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMaterialImageryNotFound
	}
	return im, nil
}

// SetImageryStatus moves entries through review (draft → published, or hidden).
func (s *MaterialService) SetImageryStatus(ctx context.Context, ids []int64, status string) (int64, error) {
	if !materialStatuses[status] || len(ids) == 0 || len(ids) > materialMaxImageryPage {
		return 0, ErrMaterialInvalid
	}
	return s.repo.SetImageryStatus(ctx, ids, status)
}

// ImageryCandidates lists the seed candidates by rank, marking those already in the library.
func (s *MaterialService) ImageryCandidates(ctx context.Context) ([]MaterialImageryCandidate, error) {
	all, err := s.repo.AllImagery(ctx)
	if err != nil {
		return nil, err
	}
	idx := newMaterialImageryIndex(all)
	out := make([]MaterialImageryCandidate, 0, len(materialCandidates()))
	for _, c := range materialCandidates() {
		_, c.InLibrary = idx[c.Name]
		out = append(out, c)
	}
	return out, nil
}

type MaterialSeedResult struct {
	Drafted  []string           `json:"drafted"`
	Existing int                `json:"existing"`
	Failed   []string           `json:"failed"`
	Warnings []MaterialBlockHit `json:"warnings"`
	Model    string             `json:"model"`
	Left     int                `json:"left"` // candidates still not in the library
}

// DraftImagerySeeds drafts the next `limit` top-ranked candidates that are not in the library yet.
// Drafts stay in status draft until an admin publishes them.
func (s *MaterialService) DraftImagerySeeds(ctx context.Context, adminID int64, limit int) (*MaterialSeedResult, error) {
	chat, err := s.chatOrErr()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > materialMaxSeedRequest {
		limit = materialMaxSeedRequest
	}
	candidates, err := s.ImageryCandidates(ctx)
	if err != nil {
		return nil, err
	}
	var todo []MaterialImageryCandidate
	for _, c := range candidates {
		if !c.InLibrary {
			todo = append(todo, c)
		}
	}
	res := &MaterialSeedResult{Drafted: []string{}, Failed: []string{}, Warnings: []MaterialBlockHit{}}
	if len(todo) > limit {
		todo = todo[:limit]
	}
	terms, err := s.repo.ListBlocklist(ctx, true)
	if err != nil {
		return nil, err
	}

	var batches [][]MaterialImageryCandidate
	for i := 0; i < len(todo); i += materialSeedBatch {
		batches = append(batches, todo[i:min(i+materialSeedBatch, len(todo))])
	}
	type batchOut struct {
		entries []materialDraftEntry
		model   string
		err     error
	}
	outs := make([]batchOut, len(batches))
	var wg sync.WaitGroup
	sem := make(chan struct{}, materialSeedConcurrency)
	for i, batch := range batches {
		wg.Add(1)
		go func(i int, batch []MaterialImageryCandidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cctx, cancel := context.WithTimeout(ctx, materialImageryTimeout)
			defer cancel()
			answer, model, err := chat(cctx, materialSeedSystemPrompt, materialSeedUserPrompt(batch))
			if err != nil {
				outs[i] = batchOut{err: err}
				return
			}
			var doc struct {
				Items []materialDraftEntry `json:"items"`
			}
			outs[i] = batchOut{model: model, err: parseMaterialJSON(answer, &doc)}
			outs[i].entries = doc.Items
		}(i, batch)
	}
	wg.Wait()

	by := adminID
	for i, batch := range batches {
		out := outs[i]
		if out.model != "" {
			res.Model = out.model
		}
		byName := map[string]materialDraftEntry{}
		for _, e := range out.entries {
			byName[strings.TrimSpace(e.Name)] = e
		}
		for _, c := range batch {
			e, ok := byName[c.Name]
			if out.err != nil || !ok {
				res.Failed = append(res.Failed, c.Name)
				continue
			}
			im := e.imagery()
			im.Name = c.Name
			if !materialImageryCategorySet[im.Category] {
				im.Category = c.Category
			}
			im.Aliases = append(im.Aliases, c.Aliases...)
			im.CreatedBy = &by
			if err := normalizeMaterialImagery(&im); err != nil {
				res.Failed = append(res.Failed, c.Name)
				continue
			}
			for _, h := range screenMaterialDraft(&im, terms) {
				h.Term = c.Name + "：" + h.Term
				res.Warnings = append(res.Warnings, h)
			}
			created, err := s.repo.CreateImagery(ctx, &im)
			if err != nil {
				return nil, err
			}
			if created {
				res.Drafted = append(res.Drafted, c.Name)
			} else {
				res.Existing++
			}
		}
	}
	if len(res.Drafted) == 0 && len(res.Failed) > 0 {
		for _, o := range outs {
			if o.err != nil {
				return nil, materialModelError(o.err)
			}
		}
	}
	after, err := s.ImageryCandidates(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range after {
		if !c.InLibrary {
			res.Left++
		}
	}
	return res, nil
}

// SegmentImagery returns a segment with its tags.
func (s *MaterialService) SegmentImagery(ctx context.Context, segmentID int64) (*MaterialSegmentContext, []MaterialSegmentImagery, error) {
	seg, err := s.repo.GetSegment(ctx, segmentID)
	if err != nil {
		return nil, nil, err
	}
	if seg == nil {
		return nil, nil, ErrMaterialSegmentNotFound
	}
	links, err := s.repo.ListSegmentImagery(ctx, segmentID)
	return seg, links, err
}

type MaterialExtractResult struct {
	Segment  *MaterialSegmentContext  `json:"segment"`
	Links    []MaterialSegmentImagery `json:"links"`
	Created  []string                 `json:"created"` // new draft imagery
	Warnings []MaterialBlockHit       `json:"warnings"`
	Model    string                   `json:"model"`
}

// ExtractSegmentImagery asks the text model which imagery a segment carries. Existing imagery is
// matched by name or alias; unknown imagery becomes a draft entry. The model's tags replace earlier
// unconfirmed AI tags (confirmed tags and manual ones stay), and an empty scene_desc is filled in.
func (s *MaterialService) ExtractSegmentImagery(ctx context.Context, adminID, segmentID int64) (*MaterialExtractResult, error) {
	chat, err := s.chatOrErr()
	if err != nil {
		return nil, err
	}
	seg, err := s.repo.GetSegment(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	if seg == nil {
		return nil, ErrMaterialSegmentNotFound
	}
	if seg.Content == "" && strings.TrimSpace(seg.SceneDesc) == "" {
		return nil, infraerrors.BadRequest("MATERIAL_SEGMENT_EMPTY", "这个片段没有原文也没有画面描述，请先写画面描述（Segment has no text or scene）")
	}
	all, err := s.repo.AllImagery(ctx)
	if err != nil {
		return nil, err
	}
	terms, err := s.repo.ListBlocklist(ctx, true)
	if err != nil {
		return nil, err
	}

	cctx, cancel := context.WithTimeout(ctx, materialImageryTimeout)
	defer cancel()
	answer, model, err := chat(cctx, materialExtractSystemPrompt, materialExtractUserPrompt(seg, all))
	if err != nil {
		return nil, materialModelError(err)
	}
	var doc struct {
		Imagery []struct {
			materialDraftEntry
			Weight int  `json:"weight"`
			New    bool `json:"new"`
		} `json:"imagery"`
		SceneDesc string `json:"scene_desc"`
	}
	if err := parseMaterialJSON(answer, &doc); err != nil {
		return nil, err
	}

	res := &MaterialExtractResult{Created: []string{}, Warnings: []MaterialBlockHit{}, Model: model}
	idx := newMaterialImageryIndex(all)
	by := adminID
	var links []MaterialSegmentImagery
	seen := map[int64]bool{}
	for _, item := range doc.Imagery {
		name := strings.TrimSpace(item.Name)
		id, ok := idx[name]
		if !ok {
			im := item.imagery()
			im.CreatedBy = &by
			if err := normalizeMaterialImagery(&im); err != nil {
				continue
			}
			for _, h := range screenMaterialDraft(&im, terms) {
				h.Term = im.Name + "：" + h.Term
				res.Warnings = append(res.Warnings, h)
			}
			if _, err := s.repo.CreateImagery(ctx, &im); err != nil {
				return nil, err
			}
			if im.ID == 0 { // lost a race with another insert of the same name
				fresh, err := s.repo.AllImagery(ctx)
				if err != nil {
					return nil, err
				}
				idx = newMaterialImageryIndex(fresh)
				if id, ok = idx[im.Name]; !ok {
					continue
				}
			} else {
				id = im.ID
				idx[im.Name] = id
				res.Created = append(res.Created, im.Name)
			}
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		links = append(links, MaterialSegmentImagery{SegmentID: segmentID, ImageryID: id, Weight: min(max(item.Weight, 1), 3), Source: "ai"})
		if len(links) == materialMaxSegmentLinks {
			break
		}
	}
	if len(links) == 0 {
		return nil, ErrMaterialModelAnswer
	}
	if err := s.repo.ReplaceAISegmentImagery(ctx, segmentID, links); err != nil {
		return nil, err
	}
	if strings.TrimSpace(seg.SceneDesc) == "" {
		if scene := materialTrimRunes(doc.SceneDesc, materialMaxImageryField); scene != "" {
			if err := s.repo.UpdateSegmentScene(ctx, segmentID, scene); err != nil {
				return nil, err
			}
		}
	}
	res.Segment, res.Links, err = s.SegmentImagery(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// MaterialSegmentImageryInput is one tag an admin keeps on a segment.
type MaterialSegmentImageryInput struct {
	ImageryID int64 `json:"imagery_id"`
	Weight    int   `json:"weight"`
}

// SaveSegmentImagery replaces a segment's tags with the admin's list; all of them become confirmed.
// Tags the model proposed keep source "ai" so the hit rate of the extraction can be measured.
func (s *MaterialService) SaveSegmentImagery(ctx context.Context, segmentID int64, in []MaterialSegmentImageryInput, scene *string) (*MaterialExtractResult, error) {
	if len(in) > materialMaxSegmentLinks {
		return nil, ErrMaterialInvalid
	}
	seg, current, err := s.SegmentImagery(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	sources := map[int64]string{}
	for _, l := range current {
		sources[l.ImageryID] = l.Source
	}
	links := []MaterialSegmentImagery{}
	seen := map[int64]bool{}
	for _, l := range in {
		if l.ImageryID <= 0 || seen[l.ImageryID] {
			continue
		}
		seen[l.ImageryID] = true
		src := sources[l.ImageryID]
		if src == "" {
			src = "manual"
		}
		links = append(links, MaterialSegmentImagery{SegmentID: segmentID, ImageryID: l.ImageryID,
			Weight: min(max(l.Weight, 1), 3), Source: src, Confirmed: true})
	}
	if err := s.repo.SetSegmentImagery(ctx, segmentID, links); err != nil {
		return nil, err
	}
	if scene != nil {
		if err := s.repo.UpdateSegmentScene(ctx, segmentID, materialTrimRunes(*scene, materialMaxImageryField)); err != nil {
			return nil, err
		}
	}
	res := &MaterialExtractResult{Created: []string{}, Warnings: []MaterialBlockHit{}}
	res.Segment, res.Links, err = s.SegmentImagery(ctx, seg.ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}
