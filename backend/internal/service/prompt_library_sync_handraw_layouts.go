package service

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
)

type handrawLayout struct {
	ID         flexString   `json:"id"`
	Category   flexString   `json:"category"`
	Name       flexString   `json:"name"`
	NameEn     flexString   `json:"name_en"`
	Image      flexString   `json:"image"`
	PromptFile flexString   `json:"prompt_file"`
	Keywords   []flexString `json:"keywords"`
}

// handrawLayoutID is a layout code such as "IG-037" (two letters name the category).
var handrawLayoutID = regexp.MustCompile(`^[A-Z]{2}-\d{3}$`)

// handrawLayoutPromptFile is a path under references/ such as "layouts/IG-037.md" (no "..", no scheme).
var handrawLayoutPromptFile = regexp.MustCompile(`^layouts/[A-Za-z0-9._-]+\.md$`)

// Categories of layouts.json: Chinese label and the library scene they belong to.
var handrawLayoutCategories = map[string]struct{ label, scene string }{
	"comic-storyboard": {"漫画分镜", "illustration"},
	"infographic":      {"信息图", "infographic"},
	"ecommerce":        {"电商图", "ecommerce"},
	"social-card":      {"社媒卡片", "social"},
	"ip-character":     {"IP 角色", "brand"},
}

const (
	handrawLayoutFetchWorkers = 6
	handrawLayoutMaxPrompt    = 64 << 10
)

// fetchHandrawLayouts reads yang0/handraw-style's layouts.json (next to styles.json): ~170 layouts
// (comic storyboards, infographics, e-commerce images, social cards, IP sheets), each with a name,
// a sample image and a Markdown file whose 「<!-- zh -->」 section is the prompt. A layout says how the
// picture is arranged, so the prompt starts with a line for the user's theme. Layouts whose file
// cannot be read are skipped this time; if most fail, the sync fails and the library keeps the last copy.
func (s *PromptLibrarySyncService) fetchHandrawLayouts(ctx context.Context, src *PromptSource) ([]PromptItem, error) {
	body, err := s.open(ctx, src.URL)
	if err != nil {
		return nil, err
	}
	var layouts []handrawLayout
	seen := map[string]bool{}
	err = decodeJSONArray(body, func(_ int, rec handrawLayout) {
		id := strings.TrimSpace(string(rec.ID))
		file := strings.TrimSpace(string(rec.PromptFile))
		if !handrawLayoutID.MatchString(id) || strings.TrimSpace(string(rec.Name)) == "" || !handrawLayoutPromptFile.MatchString(file) || seen[id] {
			return
		}
		seen[id] = true
		layouts = append(layouts, rec)
	})
	_ = body.Close()
	if err != nil {
		return nil, err
	}
	if len(layouts) == 0 {
		return nil, nil
	}

	prompts := make([]string, len(layouts))
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed int
		next   = make(chan int)
	)
	for w := 0; w < handrawLayoutFetchWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				text, err := s.readHandrawLayoutPrompt(ctx, absolutePromptURL(src.URL, strings.TrimSpace(string(layouts[i].PromptFile))))
				mu.Lock()
				if err != nil || text == "" {
					failed++
				} else {
					prompts[i] = text
				}
				mu.Unlock()
			}
		}()
	}
	for i := range layouts {
		select {
		case next <- i:
		case <-ctx.Done():
		}
	}
	close(next)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if failed*2 > len(layouts) {
		return nil, fmt.Errorf("读取图型提示词失败：%d / %d 个文件读取不到", failed, len(layouts))
	}

	repo := strings.TrimRight(strings.TrimSpace(src.Homepage), "/")
	if repo == "" {
		repo = "https://github.com/yang0/handraw-style"
	}
	items := make([]PromptItem, 0, len(layouts))
	for i, rec := range layouts {
		if prompts[i] == "" {
			continue
		}
		id := strings.TrimSpace(string(rec.ID))
		name := strings.TrimSpace(string(rec.Name))
		category := handrawLayoutCategories[strings.TrimSpace(string(rec.Category))]
		hints := []string{"illustration"}
		tags := []string{"手绘图型"}
		if category.label != "" {
			hints = []string{category.scene}
			tags = append(tags, category.label)
		}
		labels := make([]string, 0, len(rec.Keywords))
		for _, k := range rec.Keywords {
			if v := strings.TrimSpace(string(k)); v != "" && v != name {
				labels = append(labels, v)
			}
		}
		description := "手绘图型 " + id
		if category.label != "" {
			description += " · " + category.label
		}
		if en := strings.TrimSpace(string(rec.NameEn)); en != "" {
			description += "（" + en + "）"
		}
		description += "。图型决定画面的版式和结构，把第一行换成你的主题；想要特定画风，可以再加一个「手绘风格」。来自 yang0/handraw-style。"
		cover := ""
		if image := strings.TrimSpace(string(rec.Image)); image != "" && !strings.Contains(image, "://") {
			cover = absolutePromptURL(src.URL, image)
		}
		items = append(items, buildSourcePromptItem(sourcePromptFields{
			externalID: id, title: "图型 " + id + " · " + name, prompt: "【主题：在这里写你的主题或内容】\n\n" + prompts[i], description: description,
			cover: cover, tags: tags, labels: labels, author: "yang0", sourceURL: repo, modelHint: "gpt-image-2", sceneHints: hints,
		}))
	}
	return items, nil
}

// readHandrawLayoutPrompt returns the 「<!-- zh -->」 section of a layout file (the whole file without markers).
func (s *PromptLibrarySyncService) readHandrawLayoutPrompt(ctx context.Context, rawURL string) (string, error) {
	body, err := s.open(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(body, handrawLayoutMaxPrompt))
	if err != nil {
		return "", err
	}
	return handrawLayoutZh(string(raw)), nil
}

var handrawLayoutMarker = regexp.MustCompile(`(?m)^\s*<!--\s*([a-zA-Z-]+)\s*-->\s*$`)

func handrawLayoutZh(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	marks := handrawLayoutMarker.FindAllStringSubmatchIndex(text, -1)
	if len(marks) == 0 {
		return truncateRunes(strings.TrimSpace(text), promptTextMaxRunes-200)
	}
	for i, m := range marks {
		if strings.EqualFold(text[m[2]:m[3]], "zh") {
			end := len(text)
			if i+1 < len(marks) {
				end = marks[i+1][0]
			}
			return truncateRunes(strings.TrimSpace(text[m[1]:end]), promptTextMaxRunes-200)
		}
	}
	return ""
}
