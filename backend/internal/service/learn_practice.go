package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // decode generated / uploaded pictures
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// 学画画 practice (docs/DRAWING_COURSE_PLAN.md): exercises inside a lesson. A draw is one image
// generation through this site's own gateway with the learner's prompt, then a critique of the picture
// by the run model against the exercise's rubric (from the catalog). A critique alone does the same
// for a picture the learner uploads or an appreciation they write. Pictures are not stored: the page
// gets the drawing back and keeps it; the server keeps the prompt and the scores.

const (
	learnKindDraw     = "draw"
	learnKindCritique = "critique"

	learnExerciseDraw  = "draw"
	learnExerciseImage = "image"
	learnExerciseText  = "text"

	settingLearnDrawModel    = "learn_draw_model"
	settingLearnDrawQuality  = "learn_draw_quality"
	settingLearnDrawFree     = "learn_draw_free_per_day"
	settingLearnCritiqueFree = "learn_critique_free_per_day"
	settingLearnDrawCap      = "learn_draw_daily_cap"

	learnDefaultDrawModel    = "gpt-image-2"
	learnDefaultDrawQuality  = "low"
	learnDefaultDrawFree     = 2
	learnDefaultCritiqueFree = 10
	learnDefaultDrawCap      = 200

	learnDrawTimeout        = 4 * time.Minute
	learnMaxPromptRunes     = 1000
	learnMaxCritiqueText    = 1500
	learnMaxUploadBytes     = 3 << 20
	learnCritiqueImageEdge  = 768
	learnDrawImageEdge      = 1536
	learnCritiqueMaxTokens  = 2000
	learnPracticeNoteRunes  = 80
	learnPracticeTextRunes  = 400
	learnRevisedPromptRunes = learnMaxPromptRunes
)

var (
	ErrLearnExercise      = infraerrors.BadRequest("LEARN_EXERCISE_INVALID", "练习编号不正确（Invalid exercise）")
	ErrLearnDrawQuota     = infraerrors.TooManyRequests("LEARN_DRAW_QUOTA", "今天的免费生图次数用完了，可以选择自己的 Key 继续画（按实际用量计费），或者明天再来（Daily free drawings used up）")
	ErrLearnCritiqueQuota = infraerrors.TooManyRequests("LEARN_CRITIQUE_QUOTA", "今天的免费点评次数用完了，可以选择自己的 Key 继续（按实际用量计费），或者明天再来（Daily free critiques used up）")
	ErrLearnDrawBusy      = infraerrors.TooManyRequests("LEARN_DRAW_BUSY", "今天全站的免费生图名额已用完，可以选择自己的 Key 继续画（Daily drawing capacity reached）")
)

var learnExerciseRe = regexp.MustCompile(`^[a-z][0-9]{1,2}-[a-z0-9-]{1,40}$`)

// learnDrawSizes: what gpt-image models draw (square, portrait, landscape).
var learnDrawSizes = map[string]bool{"1024x1024": true, "1024x1536": true, "1536x1024": true}

var learnDrawQualities = map[string]bool{"low": true, "medium": true, "high": true}

type LearnDrawInput struct {
	Exercise string `json:"exercise"`
	Prompt   string `json:"prompt"`
	Size     string `json:"size"`
	KeyID    int64  `json:"key_id,omitempty"`
}

type LearnCritiqueInput struct {
	Exercise string `json:"exercise"`
	// Image: a data: URL (png / jpeg / webp) for image exercises (and draw exercises re-checked).
	Image string `json:"image,omitempty"`
	// Text: the appreciation (text exercises), or a note / the prompt that goes with the picture.
	Text  string `json:"text,omitempty"`
	KeyID int64  `json:"key_id,omitempty"`
}

type LearnCritiqueItem struct {
	Dim   string `json:"dim"`
	Score int    `json:"score"`
	Note  string `json:"note"`
}

type LearnCritique struct {
	Score         int                 `json:"score"`
	Pass          int                 `json:"pass"`
	Passed        bool                `json:"passed"`
	Items         []LearnCritiqueItem `json:"items"`
	Praise        string              `json:"praise"`
	Suggestion    string              `json:"suggestion"`
	RevisedPrompt string              `json:"revised_prompt,omitempty"`
}

type LearnDrawResult struct {
	// Image is the drawing as a data: URL (JPEG).
	Image    string         `json:"image"`
	Critique *LearnCritique `json:"critique,omitempty"`
	// CritiqueError: the drawing worked but the critique didn't.
	CritiqueError string `json:"critique_error,omitempty"`
	DrawsLeft     int    `json:"draws_left"`
	OwnKey        bool   `json:"own_key,omitempty"`
	LatencyMs     int    `json:"latency_ms"`
}

type LearnCritiqueResult struct {
	Critique      *LearnCritique `json:"critique"`
	CritiquesLeft int            `json:"critiques_left"`
	OwnKey        bool           `json:"own_key,omitempty"`
}

// LearnPractice is one critiqued attempt, kept for the learner's record and certificates.
type LearnPractice struct {
	UserID   int64
	Lesson   string
	Exercise string
	Kind     string
	Prompt   string
	Score    int
	Items    []LearnCritiqueItem
	KeyID    int64
}

// LearnExerciseView is an exercise as the page shows it: what is scored and the pass mark.
type LearnExerciseView struct {
	ID     string            `json:"id"`
	Lesson string            `json:"lesson"`
	Title  string            `json:"title"`
	Kind   string            `json:"kind"`
	Task   string            `json:"task"`
	Rubric []LearnRubricItem `json:"rubric"`
	Pass   int               `json:"pass"`
}

// Exercise is public (the rubric is no secret: learners should know what is looked at).
func (s *LearnService) Exercise(id string) (*LearnExerciseView, error) {
	id, ex, err := s.exercise(id)
	if err != nil {
		return nil, err
	}
	return &LearnExerciseView{ID: id, Lesson: ex.Lesson, Title: ex.Title, Kind: ex.Kind, Task: ex.Task, Rubric: ex.Rubric, Pass: ex.Pass}, nil
}

func (s *LearnService) exercise(id string) (string, LearnExerciseDef, error) {
	id = strings.TrimSpace(id)
	ex, ok := s.catalog.Exercises[id]
	if !ok {
		return "", LearnExerciseDef{}, ErrLearnExercise
	}
	return id, ex, nil
}

// Draw draws the learner's prompt for a draw exercise and critiques the picture.
func (s *LearnService) Draw(ctx context.Context, userID int64, in LearnDrawInput) (*LearnDrawResult, error) {
	id, ex, err := s.exercise(in.Exercise)
	if err != nil {
		return nil, err
	}
	if ex.Kind != learnExerciseDraw {
		return nil, ErrLearnExercise
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return nil, errLearnRunInvalid("提示词不能为空")
	}
	if utf8.RuneCountInString(prompt) > learnMaxPromptRunes {
		return nil, errLearnRunInvalid(fmt.Sprintf("提示词最多 %d 字", learnMaxPromptRunes))
	}
	size := in.Size
	if size == "" {
		size = "1024x1024"
	}
	if !learnDrawSizes[size] {
		return nil, errLearnRunInvalid("画幅")
	}
	st, siteKey := s.loadSettings(ctx)
	if in.KeyID == 0 && st.DrawDailyCap > 0 {
		all, err := s.repo.CountRuns(ctx, 0, learnKindDraw, learnDayStart(s.now()))
		if err != nil {
			return nil, err
		}
		if all >= st.DrawDailyCap {
			return nil, ErrLearnDrawBusy
		}
	}
	call, st, err := s.acquire(ctx, userID, ex.Lesson, learnKindDraw, in.KeyID, freeOnSiteKeyN(st.DrawFree, siteKey != ""), ErrLearnDrawQuota)
	if err != nil {
		return nil, err
	}
	started := s.now()
	picture, err := s.drawImage(ctx, call.key, st.DrawModel, st.DrawQuality, prompt, size)
	if err != nil {
		s.finish(ctx, call, started, err, 0, 0)
		return nil, err
	}
	out := &LearnDrawResult{DrawsLeft: call.left, OwnKey: call.keyID > 0}
	full, small, err := learnPictureVariants(picture)
	if err != nil {
		s.finish(ctx, call, started, err, 0, 0)
		return nil, errLearnRunFailed("图片无法识别")
	}
	out.Image = full
	critique, usage, cerr := s.critique(ctx, call.key, st.Model, ex, "学员写的提示词：\n"+prompt+"\n\n下面是按这段提示词生成的图片。", small)
	out.LatencyMs = s.finish(ctx, call, started, nil, usage.prompt, usage.completion)
	if cerr != nil {
		out.CritiqueError = infraerrors.Message(cerr)
		return out, nil
	}
	out.Critique = critique
	s.savePractice(ctx, userID, id, ex, prompt, critique, call.keyID)
	return out, nil
}

// freeOnSiteKeyN: free calls are paid with the site's learning key, so there are none without it.
func freeOnSiteKeyN(n int, siteKey bool) int {
	if !siteKey {
		return 0
	}
	return n
}

// Critique critiques a picture the learner uploads or an appreciation they write.
func (s *LearnService) Critique(ctx context.Context, userID int64, in LearnCritiqueInput) (*LearnCritiqueResult, error) {
	id, ex, err := s.exercise(in.Exercise)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(in.Text)
	if utf8.RuneCountInString(text) > learnMaxCritiqueText {
		return nil, errLearnRunInvalid(fmt.Sprintf("文字最多 %d 字", learnMaxCritiqueText))
	}
	var small, user string
	switch ex.Kind {
	case learnExerciseText:
		if utf8.RuneCountInString(text) < 10 {
			return nil, errLearnRunInvalid("至少写 10 个字")
		}
		user = "学员写的内容：\n" + text
	default:
		raw, err := decodeLearnUpload(in.Image)
		if err != nil {
			return nil, err
		}
		if _, small, err = learnPictureVariants(raw); err != nil {
			return nil, errLearnRunInvalid("图片无法识别，请换一张 PNG、JPG 或 WebP")
		}
		user = "下面是学员提交的图片。"
		if ex.Kind == learnExerciseDraw && text != "" {
			user = "学员写的提示词：\n" + text + "\n\n下面是按这段提示词生成的图片。"
		} else if text != "" {
			user = "学员的说明：" + text + "\n\n下面是学员提交的图片。"
		}
	}
	st, key := s.loadSettings(ctx)
	call, st, err := s.acquire(ctx, userID, ex.Lesson, learnKindCritique, in.KeyID, freeOnSiteKeyN(st.CritiqueFree, key != ""), ErrLearnCritiqueQuota)
	if err != nil {
		return nil, err
	}
	started := s.now()
	critique, usage, err := s.critique(ctx, call.key, st.Model, ex, user, small)
	s.finish(ctx, call, started, err, usage.prompt, usage.completion)
	if err != nil {
		return nil, err
	}
	prompt := ""
	if ex.Kind != learnExerciseImage {
		prompt = text
	}
	s.savePractice(ctx, userID, id, ex, prompt, critique, call.keyID)
	return &LearnCritiqueResult{Critique: critique, CritiquesLeft: call.left, OwnKey: call.keyID > 0}, nil
}

func (s *LearnService) savePractice(ctx context.Context, userID int64, id string, ex LearnExerciseDef, prompt string, c *LearnCritique, keyID int64) {
	_ = s.repo.SavePractice(context.WithoutCancel(ctx), LearnPractice{UserID: userID, Lesson: ex.Lesson, Exercise: id, Kind: ex.Kind,
		Prompt: prompt, Score: c.Score, Items: c.Items, KeyID: keyID})
}

// decodeLearnUpload reads a data: URL picture the learner uploads.
func decodeLearnUpload(dataURL string) ([]byte, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return nil, errLearnRunInvalid("请先选择一张图片")
	}
	head, payload, ok := strings.Cut(dataURL, ",")
	if !ok || !strings.HasSuffix(head, ";base64") {
		return nil, errLearnRunInvalid("图片格式")
	}
	switch strings.TrimSuffix(strings.TrimPrefix(head, "data:"), ";base64") {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return nil, errLearnRunInvalid("只支持 PNG、JPG 和 WebP 图片")
	}
	if base64.StdEncoding.DecodedLen(len(payload)) > learnMaxUploadBytes {
		return nil, errLearnRunInvalid("图片太大，请小于 3 MB")
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, errLearnRunInvalid("图片数据损坏")
	}
	return raw, nil
}

// learnPictureVariants: the picture as a JPEG data URL for the page (long edge at most 1536) and a
// smaller one for the critic (long edge at most 768).
func learnPictureVariants(raw []byte) (full, small string, err error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	if full, err = learnJPEGDataURL(img, learnDrawImageEdge, 88); err != nil {
		return "", "", err
	}
	if small, err = learnJPEGDataURL(img, learnCritiqueImageEdge, 82); err != nil {
		return "", "", err
	}
	return full, small, nil
}

func learnJPEGDataURL(img image.Image, maxEdge, quality int) (string, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("empty image")
	}
	if long := max(w, h); long > maxEdge {
		scale := float64(maxEdge) / float64(long)
		w, h = max(int(math.Round(float64(w)*scale)), 1), max(int(math.Round(float64(h)*scale)), 1)
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// drawImage asks the gateway for one picture and returns its bytes.
func (s *LearnService) drawImage(ctx context.Context, key, model, quality, prompt, size string) ([]byte, error) {
	payload, err := json.Marshal(map[string]any{"model": model, "prompt": prompt, "size": size, "quality": quality, "n": 1})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, learnDrawTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gatewayURL+"/v1/images/generations", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hivegpt-learn/1")
	client := s.drawClient
	if client == nil {
		client = &http.Client{Timeout: learnDrawTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errLearnRunFailed("生图超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 48<<20))
	if err != nil {
		return nil, errLearnRunFailed("读取图片失败")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}
	var out struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Data) == 0 {
		return nil, errLearnRunFailed("没有返回图片")
	}
	data, _, err := NewImageResultUploader(nil, "", 0, client).fetchImageBytes(ctx, out.Data[0])
	if err != nil {
		return nil, errLearnRunFailed("没有返回图片")
	}
	return data, nil
}

type learnUsage struct{ prompt, completion int }

var learnCritiqueSchema = json.RawMessage(`{"type":"json_schema","json_schema":{"name":"critique","strict":true,"schema":{` +
	`"type":"object","additionalProperties":false,"required":["items","praise","suggestion","revised_prompt"],"properties":{` +
	`"items":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["dim","score","note"],"properties":{` +
	`"dim":{"type":"string"},"score":{"type":"integer"},"note":{"type":"string"}}}},` +
	`"praise":{"type":"string"},"suggestion":{"type":"string"},"revised_prompt":{"type":"string"}}}}}`)

func learnCritiqueSystem(ex LearnExerciseDef) string {
	rubric := ""
	for _, r := range ex.Rubric {
		rubric += fmt.Sprintf("- %s：%s\n", r.Dim, r.Look)
	}
	revised := "- revised_prompt：留空字符串。\n"
	if ex.Kind == learnExerciseDraw {
		revised = "- revised_prompt：在学员原提示词的基础上按建议改写的一版中文提示词，保留学员的主题和想法，不超过 300 字。\n"
	}
	return fmt.Sprintf(`你是一位耐心、专业的美术老师，正在批改零基础学员的练习。
练习：%s
题目要求：%s

按下面每一项打分（0–100 的整数），每项写一句具体的评语（不超过 40 字，说画面里看到了什么）：
%s
规则：
- items 按上面的顺序，每项一个，dim 用上面的名称原文。
- 评分要诚实：没做到题目要求的项给低分（60 以下），做得好才给 85 以上。
- praise：一句话说做得最好的地方。
- suggestion：只给一条最重要、最具体的修改建议（说清楚改哪里、怎么改），不超过 80 字。
- 只评画面和文字本身，不评论画中人物的身份、长相或身材。
%s`, ex.Title, ex.Task, rubric, revised)
}

// critique asks the run model to grade against the rubric; the total is computed here.
func (s *LearnService) critique(ctx context.Context, key, model string, ex LearnExerciseDef, user, picture string) (*LearnCritique, learnUsage, error) {
	msg := LearnMessage{Role: "user", Content: user}
	if picture != "" {
		msg.Images = []string{picture}
	}
	res, err := s.callGatewayMax(ctx, key, model, LearnRunInput{
		Messages:       []LearnMessage{{Role: "system", Content: learnCritiqueSystem(ex)}, msg},
		ResponseFormat: learnCritiqueSchema,
	}, learnCritiqueMaxTokens)
	if err != nil {
		return nil, learnUsage{}, err
	}
	usage := learnUsage{res.PromptTokens, res.CompletionTokens}
	var raw struct {
		Items         []LearnCritiqueItem `json:"items"`
		Praise        string              `json:"praise"`
		Suggestion    string              `json:"suggestion"`
		RevisedPrompt string              `json:"revised_prompt"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Content)), &raw); err != nil {
		return nil, usage, errLearnRunFailed("点评格式无法识别")
	}
	c, err := gradeLearnCritique(ex, raw.Items)
	if err != nil {
		return nil, usage, err
	}
	c.Praise = clipRunes(raw.Praise, learnPracticeTextRunes)
	c.Suggestion = clipRunes(raw.Suggestion, learnPracticeTextRunes)
	if ex.Kind == learnExerciseDraw {
		c.RevisedPrompt = clipRunes(raw.RevisedPrompt, learnRevisedPromptRunes)
	}
	return c, usage, nil
}

// gradeLearnCritique matches the critic's items to the rubric and computes the weighted total.
func gradeLearnCritique(ex LearnExerciseDef, got []LearnCritiqueItem) (*LearnCritique, error) {
	byDim := map[string]LearnCritiqueItem{}
	for _, it := range got {
		byDim[strings.TrimSpace(it.Dim)] = it
	}
	c := &LearnCritique{Pass: ex.Pass, Items: make([]LearnCritiqueItem, 0, len(ex.Rubric))}
	sum, weights := 0, 0
	for i, r := range ex.Rubric {
		it, ok := byDim[r.Dim]
		if !ok {
			if i >= len(got) {
				return nil, errLearnRunFailed("点评缺少评分项")
			}
			it = got[i] // the critic renamed a dimension: take it by position
		}
		score := min(max(it.Score, 0), 100)
		c.Items = append(c.Items, LearnCritiqueItem{Dim: r.Dim, Score: score, Note: clipRunes(strings.TrimSpace(it.Note), learnPracticeNoteRunes)})
		sum += score * r.Weight
		weights += r.Weight
	}
	c.Score = int(math.Round(float64(sum) / float64(weights)))
	c.Passed = c.Score >= ex.Pass
	return c, nil
}
