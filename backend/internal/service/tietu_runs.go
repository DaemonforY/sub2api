package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// The background runs of AI 做贴图 (tietu.go).

// --- suggest ---------------------------------------------------------------------------------

func (s *TietuService) runSuggest(ctx context.Context, p *TietuProject, key string) error {
	s.event(p, "step", "按主题挑诗…")
	s.save(p)
	msgs := []LearnMessage{{Role: "system", Content: tietuSuggestPrompt()}, {Role: "user", Content: tietuSuggestRequest(p.Brief)}}
	var reply tietuSuggestReply
	if err := s.chatJSON(ctx, p, key, msgs, tietuSuggestTokens, &reply); err != nil {
		return err
	}
	out := []TietuSuggestion{}
	for _, sg := range reply.Suggestions {
		sg.Title, sg.Author = strings.TrimSpace(sg.Title), strings.TrimSpace(sg.Author)
		if sg.Title == "" || len(out) >= 3 {
			continue
		}
		sg.Reason = articleClip(strings.TrimSpace(sg.Reason), 80)
		// Only poems the library has: the text must come from a source, never from the model.
		if lib := s.findLibPoem(ctx, sg.Title+" "+sg.Author); lib != nil {
			sg.LibID, sg.Dynasty = lib.ID, lib.Dynasty
			out = append(out, sg)
		}
	}
	if len(out) == 0 {
		return errLearnRunFailed("推荐的诗在诗词库里都没找到，换个主题或直接写篇名")
	}
	p.Suggestions = out
	p.Status = TietuSuggested
	s.event(p, "done", fmt.Sprintf("推荐了 %d 首，选一首继续", len(out)))
	return nil
}

// chatJSON asks for a JSON reply and retries once when it doesn't parse.
func (s *TietuService) chatJSON(ctx context.Context, p *TietuProject, key string, msgs []LearnMessage, maxTokens int, v any) error {
	model := s.model(ctx)
	text, res, err := s.chat(ctx, key, model, msgs, maxTokens)
	s.usage(p, res)
	if err != nil {
		return err
	}
	perr := parseTietuJSON(text, v)
	if perr == nil {
		return nil
	}
	msgs = append(msgs, LearnMessage{Role: "assistant", Content: articleClip(text, 6000)},
		LearnMessage{Role: "user", Content: "上面的输出不是要求的 JSON（" + perr.Error() + "）。只输出一个合法的 JSON 对象，不要任何其他文字。"})
	text, res, err = s.chat(ctx, key, model, msgs, maxTokens)
	s.usage(p, res)
	if err != nil {
		return err
	}
	if parseTietuJSON(text, v) != nil {
		return errLearnRunFailed("模型没有给出可用的结果，点「重试」再来一次")
	}
	return nil
}

// --- plan ------------------------------------------------------------------------------------

func (s *TietuService) runPlan(ctx context.Context, p *TietuProject, key string) error {
	if p.Poem == nil {
		s.event(p, "step", "查找原文，核对诗词库和维基文库…")
		s.save(p)
		po, err := s.fetchPoem(ctx, p, key)
		if err != nil {
			return err
		}
		p.Poem = po
		p.Title = articleClip(po.Title, 200)
		s.event(p, "step", tietuPoemSummary(po))
	}
	s.event(p, "step", "构思画面：立意、细节和分镜…")
	s.save(p)
	msgs := []LearnMessage{{Role: "system", Content: tietuPlanPrompt(p.Brief)}, {Role: "user", Content: tietuPlanRequest(p)}}
	var raw TietuPlan
	if err := s.chatJSON(ctx, p, key, msgs, tietuPlanTokens, &raw); err != nil {
		return err
	}
	plan, err := cleanTietuPlan(raw, p.Brief)
	if err != nil {
		return errLearnRunFailed("模型给出的方案不完整，点「重试」再来一次")
	}
	p.Plan = plan
	p.Feedback = ""
	p.Status = TietuPlanned
	kind := "连环画"
	if plan.Kind == "single" {
		kind = "主画面 + 局部特写"
	}
	s.event(p, "done", fmt.Sprintf("方案好了：%s，封面 + %d 张。看看是否满意，可以直接修改，或写下意见让 AI 重做", kind, len(plan.Panels)))
	return nil
}

// --- covers ----------------------------------------------------------------------------------

func (s *TietuService) runCovers(ctx context.Context, p *TietuProject, key string) error {
	if len(p.Covers) == 0 {
		for i := 1; i <= tietuCovers; i++ {
			p.Covers = append(p.Covers, TietuImage{Name: fmt.Sprintf("c%d", i), Kind: "candidate", N: i, Status: "pending"})
		}
	}
	s.event(p, "step", fmt.Sprintf("画 %d 张封面候选…", len(p.Covers)))
	s.save(p)
	prompt := tietuCoverPrompt(p)
	if err := s.drawList(ctx, p, key, p.Covers, func(TietuImage) (string, []byte) { return prompt, nil }); err != nil {
		return err
	}
	ok := 0
	for i := range p.Covers {
		if p.Covers[i].Status == "ok" {
			ok++
		}
	}
	if ok == 0 {
		return errLearnRunFailed("封面都没画成，点「重试」再来一次")
	}
	s.checkList(ctx, p, key, p.Covers)
	p.Status = TietuCovered
	s.event(p, "done", fmt.Sprintf("%d 张封面画好了，挑一张继续；都不满意可以写下意见重画", ok))
	return nil
}

// --- panels + check + post -------------------------------------------------------------------

func tietuPanelSlots(pl *TietuPlan) []TietuImage {
	out := []TietuImage{}
	if pl == nil {
		return out
	}
	for i := range pl.Panels {
		out = append(out, TietuImage{Name: fmt.Sprintf("p%d", i+1), Kind: "panel", N: i + 1, Status: "pending"})
	}
	return out
}

func (s *TietuService) runPanels(ctx context.Context, p *TietuProject, key string) error {
	if len(p.Panels) == 0 {
		p.Panels = tietuPanelSlots(p.Plan)
	}
	ref, err := os.ReadFile(s.imagePath(p.ID, fmt.Sprintf("c%d", p.CoverPick)))
	if err != nil {
		return errLearnRunFailed("找不到选中的封面，请重新选择封面")
	}
	todo := 0
	for _, im := range p.Panels {
		if im.Status != "ok" {
			todo++
		}
	}
	if todo > 0 {
		s.event(p, "step", fmt.Sprintf("以封面为参考，画 %d 张分镜…", todo))
		s.save(p)
		if err := s.drawList(ctx, p, key, p.Panels, func(im TietuImage) (string, []byte) {
			return tietuPanelPrompt(p, im.N, im.Note), ref
		}); err != nil {
			return err
		}
	}
	p.Status = TietuChecking
	s.event(p, "step", "看图自检：文字、时代、叙事顺序、找一找、小图是否看得清…")
	s.save(p)
	s.checkList(ctx, p, key, p.Panels)
	if p.Post == nil || !p.PostEdited {
		post := tietuComposePost(p)
		p.Post = &post
	}
	p.Status = TietuDone
	ok, flagged := 0, 0
	for _, im := range p.Panels {
		if im.Status == "ok" {
			ok++
		}
		if im.Check != nil && !im.Check.OK && !im.Check.Skipped {
			flagged++
		}
	}
	msg := fmt.Sprintf("好了：封面 + %d 张分镜，标题和正文也写好了", ok)
	if failed := len(p.Panels) - ok; failed > 0 {
		msg += fmt.Sprintf("；%d 张没画成，可以点「重画」", failed)
	}
	if flagged > 0 {
		msg += fmt.Sprintf("；自检标出 %d 张有问题，看看要不要重画", flagged)
	}
	s.event(p, "done", msg)
	return nil
}

// drawList draws the pending pictures of list in parallel (with ref as the reference picture when
// the prompt function returns one) and saves each as a JPEG.
func (s *TietuService) drawList(ctx context.Context, p *TietuProject, key string, list []TietuImage, prompt func(TietuImage) (string, []byte)) error {
	if err := os.MkdirAll(s.projectDir(p.ID), 0o755); err != nil {
		return errLearnRunFailed("保存图片失败")
	}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(tietuImageParallel)
	for i := range list {
		if list[i].Status == "ok" {
			continue
		}
		g.Go(func() error {
			mu.Lock()
			im := list[i]
			mu.Unlock()
			text, ref := prompt(im)
			var raw []byte
			var err error
			if ref != nil {
				raw, err = s.edit(gctx, key, text, ref)
			} else {
				raw, err = s.draw(gctx, key, text)
			}
			var data []byte
			if err == nil {
				data, err = tietuJPEG(raw)
			}
			if err == nil {
				err = os.WriteFile(s.imagePath(p.ID, im.Name), data, 0o644)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				msg := articleErrorText(p.ID, err)
				list[i].Status, list[i].Error = "failed", msg
				s.event(p, "error", tietuImageLabel(im)+"没画成："+msg)
			} else {
				list[i].Status, list[i].Error, list[i].Check = "ok", "", nil
				p.ImagesDrawn++
				s.event(p, "step", tietuImageLabel(im)+"画好了")
			}
			s.save(p)
			return nil
		})
	}
	return g.Wait()
}

func tietuImageLabel(im TietuImage) string {
	if im.Kind == "candidate" {
		return fmt.Sprintf("封面候选 %d ", im.N)
	}
	return fmt.Sprintf("第 %d 张", im.N)
}

// checkList runs the vision check on the drawn pictures that have no check yet. A failed check is
// recorded as skipped: the pictures are still usable.
func (s *TietuService) checkList(ctx context.Context, p *TietuProject, key string, list []TietuImage) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, tietuImageParallel)
	for i := range list {
		if list[i].Status != "ok" || list[i].Check != nil {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			mu.Lock()
			im := list[i]
			mu.Unlock()
			c := s.checkOne(ctx, p, key, im, &mu)
			mu.Lock()
			list[i].Check = c
			if !c.Skipped && !c.OK {
				s.event(p, "info", tietuImageLabel(im)+"自检发现问题："+strings.Join(c.Issues, "；"))
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	s.save(p)
}

func (s *TietuService) checkOne(ctx context.Context, p *TietuProject, key string, im TietuImage, mu *sync.Mutex) *TietuCheck {
	data, err := os.ReadFile(s.imagePath(p.ID, im.Name))
	if err != nil {
		return &TietuCheck{OK: true, Skipped: true, Issues: []string{}, Minor: []string{}}
	}
	img, err := tietuVisionURL(data, 768)
	if err != nil {
		return &TietuCheck{OK: true, Skipped: true, Issues: []string{}, Minor: []string{}}
	}
	msgs := []LearnMessage{{Role: "system", Content: tietuCheckPrompt()}, {Role: "user", Content: tietuCheckRequest(p, im), Images: []string{img}}}
	text, res, err := s.chat(ctx, key, s.model(ctx), msgs, tietuCheckTokens)
	mu.Lock()
	s.usage(p, res)
	mu.Unlock()
	var c TietuCheck
	if err != nil || parseTietuJSON(text, &c) != nil {
		return &TietuCheck{OK: true, Skipped: true, Issues: []string{}, Minor: []string{}}
	}
	issues := []string{}
	for _, it := range c.Issues {
		if it = articleClip(strings.TrimSpace(it), 40); it != "" && len(issues) < 6 {
			issues = append(issues, it)
		}
	}
	c.Issues = issues
	c.OK = c.OK && len(issues) == 0
	minor := []string{}
	for _, it := range c.Minor {
		if it = articleClip(strings.TrimSpace(it), 40); it != "" && len(minor) < 6 {
			minor = append(minor, it)
		}
	}
	c.Minor = minor
	c.FindSeen = articleClip(tietuCleanFind(c.FindSeen), 30)
	if c.Focal != "weak" {
		c.Focal = "clear"
	}
	return &c
}
