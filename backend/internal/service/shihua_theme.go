package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// The scene of a work is drafted by a text model through this site's gateway (on the admin's key)
// and then edited by the admin before anything is drawn.

const shihuaThemeTimeout = 3 * time.Minute

const shihuaThemeSystemPrompt = `你是中国古典文学与中国画的美术指导。根据给出的课本古诗文，为一张「新中式细线设色装饰插画海报」写画面描述。

要求：
1. 忠实再现原文的意境与关键意象，每个关键意象都要能在画面里找到；不添加原文没有的情节。说理文、议论文用能体现主旨的人物与场景来象征。
2. 人物服饰、发式、建筑、器物符合作品的朝代；不得出现现代物品、领导人、国旗、招牌或任何文字。
3. 写清季节、时辰、天气，主体人物（数量、动作、朝向、服饰），主要场景与近中远景层次，以及主色调。
4. 构图：画面左上方要竖排书法标题，那里放深色或高对比的连续色面（夜空、山体、树荫等）；画面右上方留一块连续的暖米白天空、云气或水面留白（约占宽 30%、高 40%），用来后期排正文；主体放在画面中部和下部。
5. prompt_theme 是直接拼进生图提示词的一段中文，120–260 字，只描述画面内容，不写画风，不写需要画出的文字。

只输出一个 JSON 对象，不要任何其它文字：
{"subject":"主体","scene":"场景","season":"季节","time":"时辰与天气","figures":"人物与服饰","key_imagery":["原文意象 → 画面元素"],"palette":"主色调","avoid":["需要避免的元素"],"prompt_theme":"……"}`

// GenerateTheme asks the text model for a scene draft and saves it on the work.
func (s *ShihuaService) GenerateTheme(ctx context.Context, workID int64) (*ShihuaWork, error) {
	st := s.Settings(ctx)
	key, err := s.adminKey(ctx, st.KeyID, 0)
	if err != nil {
		return nil, err
	}
	w, err := s.GetWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, shihuaThemeTimeout)
	defer cancel()
	answer, err := s.chat(ctx, key.Key, st.TextModel, []LearnMessage{
		{Role: "system", Content: shihuaThemeSystemPrompt},
		{Role: "user", Content: shihuaThemeUserPrompt(w)},
	})
	if err != nil {
		return nil, infraerrors.BadRequest("SHIHUA_THEME_FAILED", "文本模型调用失败："+infraerrors.Message(err))
	}
	theme, err := parseShihuaTheme(answer)
	if err != nil {
		return nil, err
	}
	now := s.now()
	theme.Model, theme.GeneratedAt = st.TextModel, &now
	w.Theme = *theme
	if err := s.repo.UpdateWork(ctx, w); err != nil {
		return nil, err
	}
	return s.GetWork(ctx, workID)
}

func shihuaThemeUserPrompt(w *ShihuaWork) string {
	var b strings.Builder
	fmt.Fprintf(&b, "篇名：%s\n作者：%s %s\n", w.Title, w.Dynasty, w.Author)
	if w.Grade != "" {
		fmt.Fprintf(&b, "教材：%s\n", w.Grade)
	}
	if w.DisplayText != "" {
		fmt.Fprintf(&b, "海报只画这一段（节选）：\n%s\n\n全文供理解：\n%s\n", w.DisplayText, w.FullText)
	} else {
		fmt.Fprintf(&b, "原文：\n%s\n", w.FullText)
	}
	if strings.TrimSpace(w.ExcerptNote) != "" {
		fmt.Fprintf(&b, "节选说明：%s\n", w.ExcerptNote)
	}
	return b.String()
}

// parseShihuaTheme reads the model's JSON (tolerating a code fence or a sentence around it).
func parseShihuaTheme(answer string) (*ShihuaTheme, error) {
	start, end := strings.Index(answer, "{"), strings.LastIndex(answer, "}")
	if start < 0 || end <= start {
		return nil, ErrShihuaThemeFailed
	}
	var t ShihuaTheme
	if err := json.Unmarshal([]byte(answer[start:end+1]), &t); err != nil {
		return nil, ErrShihuaThemeFailed
	}
	t.PromptTheme = strings.TrimSpace(t.PromptTheme)
	if t.PromptTheme == "" || utf8.RuneCountInString(t.PromptTheme) > 1500 {
		return nil, ErrShihuaThemeFailed
	}
	return &t, nil
}
