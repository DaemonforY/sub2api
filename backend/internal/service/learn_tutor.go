package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// The AI tutor on lesson pages: answers about the lesson the learner is on, with the lesson's text
// in the system prompt, streamed back. Free questions a day, then the learner's own key.

const (
	learnTutorMaxMessages  = 10
	learnTutorMaxQuestion  = 1000
	learnTutorMaxReply     = 4000
	learnTutorMaxLesson    = 8000
	learnTutorMaxArticle   = 12000
	learnTutorOutputTokens = 900
)

var ErrLearnTutorQuota = infraerrors.TooManyRequests("LEARN_TUTOR_QUOTA", "今天的免费提问次数用完了，明天再来，或者选择用自己的 Key 继续提问（Daily free questions used up）")

type LearnTutorInput struct {
	Lesson   string         `json:"lesson"`
	Messages []LearnMessage `json:"messages"`
	KeyID    int64          `json:"key_id,omitempty"`
}

type LearnTutorResult struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TutorLeft        int    `json:"tutor_left"`
	OwnKey           bool   `json:"own_key,omitempty"`
}

// learnArticleRe: 大数据 articles the tutor can answer on (their page path).
var learnArticleRe = regexp.MustCompile(`^bigdata/(spark|flink|paimon)/[a-z0-9-]{1,40}$`)

func validateTutor(in *LearnTutorInput) error {
	if !learnLessonRe.MatchString(in.Lesson) && !learnArticleRe.MatchString(in.Lesson) {
		return ErrLearnLessonID
	}
	if len(in.Messages) == 0 || len(in.Messages) > learnTutorMaxMessages {
		return errLearnRunInvalid("一次对话最多 5 轮，请点「新对话」重新开始")
	}
	for i, m := range in.Messages {
		want := "user"
		if i%2 == 1 {
			want = "assistant"
		}
		n := utf8.RuneCountInString(strings.TrimSpace(m.Content))
		limit := learnTutorMaxQuestion
		if want == "assistant" {
			limit = learnTutorMaxReply
		}
		if m.Role != want || n == 0 || n > limit || len(m.ToolCalls) > 0 || m.ToolCallID != "" {
			return errLearnRunInvalid(fmt.Sprintf("每个问题 1–%d 字", learnTutorMaxQuestion))
		}
	}
	if in.Messages[len(in.Messages)-1].Role != "user" {
		return errLearnRunInvalid("最后一条须是问题")
	}
	return nil
}

func (s *LearnService) tutorSystemPrompt(lesson string) string {
	title, text := "", ""
	if s.lessonText != nil {
		title, text = s.lessonText(lesson)
	}
	article := learnArticleRe.MatchString(lesson)
	limit := learnTutorMaxLesson
	if article {
		limit = learnTutorMaxArticle
	}
	if r := []rune(text); len(r) > limit {
		text = string(r[:limit]) + "…"
	}
	if article {
		prompt := "你是 HiveGPT「AI 学习」大数据专区的助教，熟悉 Spark、Flink、Paimon 的原理和源码。"
		if title != "" {
			prompt += fmt.Sprintf("学员正在读文章「%s」。", title)
		}
		if text != "" {
			prompt += "下面是这篇文章的正文，回答时以它为准；文中标注的源码版本、类名和行号优先于你的记忆：\n<article>\n" + text + "\n</article>\n"
		}
		return prompt + `回答要求：
- 用中文，先直接回答，再按需要补充；一般不超过 400 字，需要时给出简短的代码或源码位置。
- 只回答和这篇文章、大数据组件的原理与源码、大数据开发和面试有关的问题；其他问题礼貌地说明你只能回答这些。
- 文章没写到的细节，说明是你的补充，并提醒学员以对应版本的源码为准；不确定就说不确定。
- 学员贴出报错或日志时，先说最可能的原因，再给出排查步骤。`
	}
	prompt := "你是 HiveGPT「AI 学习」的助教。"
	if title != "" {
		prompt += fmt.Sprintf("学员正在学习课时「%s」。", title)
	}
	if text != "" {
		prompt += "下面是这节课的正文，回答时以它为准：\n<lesson>\n" + text + "\n</lesson>\n"
	}
	return prompt + `回答要求：
- 用中文，先直接回答，再按需要补充；一般不超过 300 字，需要时给出简短的代码。
- 只回答和这节课、AI 应用开发、AI 绘画与视频创作、HiveGPT 使用有关的问题；其他问题礼貌地说明你只能回答课程相关的问题。
- 不要编造 HiveGPT 没有的功能、价格或模型；不确定就说不确定，建议学员看课程正文或联系客服。
- 学员贴出报错时，先说最可能的原因，再给出检查步骤。`
}

// Tutor answers the learner's question about a lesson, calling onDelta with each piece of the reply.
// Errors returned before the first onDelta mean nothing was sent.
func (s *LearnService) Tutor(ctx context.Context, userID int64, in LearnTutorInput, onDelta func(string) error) (*LearnTutorResult, error) {
	if err := validateTutor(&in); err != nil {
		return nil, err
	}
	call, st, err := s.acquire(ctx, userID, in.Lesson, learnKindTutor, in.KeyID, -1, ErrLearnTutorQuota)
	if err != nil {
		return nil, err
	}
	messages := append([]LearnMessage{{Role: "system", Content: s.tutorSystemPrompt(in.Lesson)}}, in.Messages...)
	started := s.now()
	res, callErr := s.streamGateway(ctx, call.key, st.Model, messages, learnTutorOutputTokens, onDelta)
	if callErr != nil {
		s.finish(ctx, call, started, callErr, 0, 0)
		return nil, callErr
	}
	s.finish(ctx, call, started, nil, res.PromptTokens, res.CompletionTokens)
	res.TutorLeft = call.left
	res.OwnKey = call.keyID > 0
	return res, nil
}

// streamGateway makes a streamed chat completion through this site's gateway.
func (s *LearnService) streamGateway(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
	payload, err := json.Marshal(map[string]any{
		"model":                 model,
		"messages":              messages,
		"max_completion_tokens": maxTokens,
		"stream":                true,
		"stream_options":        map[string]any{"include_usage": true},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gatewayURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "hivegpt-learn/1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errLearnRunFailed("连接模型服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}
	out := &LearnTutorResult{}
	sent := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}
		if chunk.Error != nil {
			return nil, errLearnRunFailed(upstreamErrorText([]byte(data), http.StatusBadGateway))
		}
		if chunk.Model != "" {
			out.Model = chunk.Model
		}
		if chunk.Usage != nil {
			out.PromptTokens, out.CompletionTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content == "" {
				continue
			}
			if err := onDelta(c.Delta.Content); err != nil {
				return nil, err
			}
			sent = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, errLearnRunFailed("读取结果中断")
	}
	if !sent {
		return nil, errLearnRunFailed("模型没有返回内容")
	}
	return out, nil
}

// learnSince is the start of today (Beijing time) for counters.
func (s *LearnService) learnSince() time.Time { return learnDayStart(s.now()) }
