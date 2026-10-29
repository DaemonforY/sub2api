package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Changing the pictures of a finished video: the user replaces, adds or removes pictures on the
// project page. The narration stays as recorded; the director looks at the new pictures, re-plans
// the scenes that show them, and only those scenes are rebuilt (with their current code as the
// starting point, so the rest of the look is kept).

var (
	ErrVideoImageUnknown  = infraerrors.BadRequest("VIDEO_IMAGE_UNKNOWN", "要更换的图片不在这个作品里，请刷新页面后再试（Unknown picture）")
	ErrVideoImageNoChange = infraerrors.BadRequest("VIDEO_IMAGE_NO_CHANGE", "没有要更换、添加或删除的图片（Nothing to change）")
)

const videoImageNoteMaxChars = 500

// VideoImagesChange is a change of a project's pictures.
type VideoImagesChange struct {
	Replace map[string]string `json:"replace,omitempty"` // picture id → token of the staged new picture
	Add     []string          `json:"add,omitempty"`     // staged tokens, appended as new ids
	Remove  []string          `json:"remove,omitempty"`  // picture ids
	Note    string            `json:"note,omitempty"`    // what the user wants done with them
}

// videoImageChanges is what a change did, by picture id.
type videoImageChanges struct {
	Replaced, Added, Removed []string
	Note                     string
	// Showed: scene id → the removed pictures it showed (they are gone from the spec by then).
	Showed map[string][]string
}

func (c videoImageChanges) touches(id string) bool {
	return slices.Contains(c.Replaced, id) || slices.Contains(c.Removed, id)
}

// videoSceneUses reports whether a scene shows a picture: listed in its images, or loaded by its code.
func videoSceneUses(sc VideoScene, id string) bool {
	return slices.Contains(sc.Images, id) || strings.Contains(sc.Code, "'"+id+"'") || strings.Contains(sc.Code, `"`+id+`"`) || strings.Contains(sc.Code, "`"+id+"`")
}

// ChangeImages swaps the pictures of a finished video and rebuilds the scenes that show them.
func (s *VideoService) ChangeImages(ctx context.Context, key *APIKey, id string, in VideoImagesChange) (*VideoProject, error) {
	in.Note = strings.TrimSpace(in.Note)
	if len([]rune(in.Note)) > videoImageNoteMaxChars {
		return nil, ErrVideoInvalid
	}
	if len(in.Replace) == 0 && len(in.Add) == 0 && len(in.Remove) == 0 {
		return nil, ErrVideoImageNoChange
	}
	var changes videoImageChanges
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil || p.Status == VideoStatusQuestions {
			return ErrVideoNotReady
		}
		// The version is the spec as it was; applyImageChange edits p.Spec in place.
		var old VideoSpec
		raw, err := json.Marshal(p.Spec)
		if err == nil {
			err = json.Unmarshal(raw, &old)
		}
		if err != nil {
			return err
		}
		if changes, err = s.applyImageChange(p, in); err != nil {
			return err
		}
		_ = s.repo.AddVersion(ctx, p.ID, "换图前", &old)
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: videoImageChangeText(changes)})
	}, func(ctx context.Context, r *videoRun) error { return s.swapImages(ctx, r, changes) })
}

// applyImageChange stores the new pictures and updates options.images and spec.images. Old files
// stay on disk: earlier versions still show them.
func (s *VideoService) applyImageChange(p *VideoProject, in VideoImagesChange) (videoImageChanges, error) {
	var c videoImageChanges
	c.Note = in.Note
	index := map[string]int{}
	next := 0
	for i, img := range p.Options.Images {
		index[img.ID] = i
		if n, err := strconv.Atoi(strings.TrimPrefix(img.ID, "img")); err == nil && n > next {
			next = n
		}
	}
	removed := map[string]bool{}
	for _, id := range in.Remove {
		if _, ok := index[id]; !ok {
			return c, ErrVideoImageUnknown
		}
		if !removed[id] {
			removed[id] = true
			c.Removed = append(c.Removed, id)
		}
	}
	for id := range in.Replace {
		if _, ok := index[id]; !ok || removed[id] {
			return c, ErrVideoImageUnknown
		}
	}
	if len(p.Options.Images)-len(c.Removed)+len(in.Add) > VideoImagesMax {
		return c, ErrVideoImageCount
	}
	if err := os.MkdirAll(s.projectDir(p.ID), 0o755); err != nil {
		return c, err
	}
	images := make([]VideoImage, 0, len(p.Options.Images)+len(in.Add))
	for _, img := range p.Options.Images {
		if removed[img.ID] {
			continue
		}
		if token := in.Replace[img.ID]; token != "" {
			fresh, err := s.stagedImage(p, token, img.ID)
			if err != nil {
				return c, err
			}
			if fresh.File != img.File {
				c.Replaced = append(c.Replaced, img.ID)
				img = fresh
			}
		}
		img.URL, img.Desc = "", ""
		images = append(images, img)
	}
	for _, token := range in.Add {
		next++
		img, err := s.stagedImage(p, token, fmt.Sprintf("img%d", next))
		if err != nil {
			return c, err
		}
		c.Added = append(c.Added, img.ID)
		images = append(images, img)
	}
	if len(c.Replaced)+len(c.Added)+len(c.Removed) == 0 {
		return c, ErrVideoImageNoChange // the "new" pictures are the same files
	}
	p.Options.Images = images

	descs := map[string]string{}
	for _, img := range p.Spec.Images {
		if !c.touches(img.ID) {
			descs[img.ID] = img.Desc
		}
	}
	p.Spec.Images = specImages(p, descs)
	c.Showed = map[string][]string{}
	for i := range p.Spec.Scenes {
		sc := &p.Spec.Scenes[i]
		for _, id := range c.Removed {
			if videoSceneUses(*sc, id) {
				c.Showed[sc.ID] = append(c.Showed[sc.ID], id)
			}
		}
		sc.Images = slices.DeleteFunc(sc.Images, func(id string) bool { return removed[id] })
		if p.Mode == VideoModeMotion {
			sc.Images = nil
			for _, img := range p.Spec.Images {
				sc.Images = append(sc.Images, img.ID)
			}
		}
	}
	return c, nil
}

func videoImageChangeText(c videoImageChanges) string {
	var parts []string
	label := func(ids []string) string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = "图" + strings.TrimPrefix(id, "img")
		}
		return strings.Join(out, "、")
	}
	if len(c.Replaced) > 0 {
		parts = append(parts, "更换 "+label(c.Replaced))
	}
	if len(c.Added) > 0 {
		parts = append(parts, "新增 "+label(c.Added))
	}
	if len(c.Removed) > 0 {
		parts = append(parts, "删除 "+label(c.Removed))
	}
	text := "换图：" + strings.Join(parts, "；")
	if c.Note != "" {
		text += "\n" + c.Note
	}
	return text
}

// videoImageSwapPlan is the director's answer to a change of pictures.
type videoImageSwapPlan struct {
	Reply      string `json:"reply"`
	ImageNotes []struct {
		ID   string `json:"id"`
		Desc string `json:"desc"`
	} `json:"image_notes"`
	Scenes []struct {
		ID          string   `json:"id"`
		Images      []string `json:"images"`
		Visual      string   `json:"visual"`
		Instruction string   `json:"instruction"`
	} `json:"scenes"`
}

func (s *VideoService) swapImages(ctx context.Context, r *videoRun, c videoImageChanges) error {
	p := r.p
	redo := map[string]string{}
	if p.Mode == VideoModeMotion {
		redo[p.Spec.Scenes[0].ID] = videoMotionImageInstruction(p, c)
	} else {
		var plan videoImageSwapPlan
		fresh := append(append([]string(nil), c.Replaced...), c.Added...)
		if len(fresh) > 0 || c.Note != "" {
			s.step(ctx, r, "plan", "正在看新图片、重新安排分镜…")
			msgs := videoImageSwapPrompt(p, c)
			data := s.imagesForModel(p)
			for _, id := range fresh {
				if d := data[id]; d != "" {
					msgs[len(msgs)-1].Images = append(msgs[len(msgs)-1].Images, d)
				}
			}
			answer, err := s.chat(ctx, r, msgs, 6000)
			if err != nil {
				return err
			}
			if raw := extractJSONObject(answer); raw != "" {
				_ = json.Unmarshal([]byte(raw), &plan)
			}
		}
		r.mu.Lock()
		redo = applyImageSwapPlan(p, c, plan)
		r.mu.Unlock()
		if reply := strings.TrimSpace(plan.Reply); reply != "" {
			_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "assistant", Text: reply})
		}
	}
	s.save(ctx, r)
	if len(redo) == 0 {
		s.done(ctx, r, "图片已更新，没有分镜需要重做")
		return nil
	}
	if err := s.codeScenes(ctx, r, redo); err != nil {
		return err
	}
	s.done(ctx, r, fmt.Sprintf("换图完成（重做了 %d 个分镜）", len(redo)))
	return nil
}

// applyImageSwapPlan writes the plan into the spec and returns the scenes to rebuild with their
// instructions. Scenes that showed a replaced or removed picture are always rebuilt, whatever the
// plan says (their labels and crops point at the old picture).
func applyImageSwapPlan(p *VideoProject, c videoImageChanges, plan videoImageSwapPlan) map[string]string {
	known := map[string]bool{}
	for _, img := range p.Spec.Images {
		known[img.ID] = true
	}
	for _, n := range plan.ImageNotes {
		for i := range p.Spec.Images {
			if img := &p.Spec.Images[i]; img.ID == n.ID && strings.TrimSpace(n.Desc) != "" {
				img.Desc = clipRunes(strings.TrimSpace(n.Desc), 300)
			}
		}
	}
	// What each scene showed before the plan moved pictures around.
	before := map[string]VideoScene{}
	for _, sc := range p.Spec.Scenes {
		before[sc.ID] = sc
	}
	redo := map[string]string{}
	planned := map[string]bool{}
	for _, change := range plan.Scenes {
		for i := range p.Spec.Scenes {
			sc := &p.Spec.Scenes[i]
			if sc.ID != change.ID {
				continue
			}
			planned[sc.ID] = true
			if change.Images != nil {
				var ids []string
				for _, id := range change.Images {
					if known[id] && !slices.Contains(ids, id) {
						ids = append(ids, id)
					}
				}
				sc.Images = ids
			}
			if v := strings.TrimSpace(change.Visual); v != "" {
				sc.Visual = clipRunes(v, 2000)
			}
			redo[sc.ID] = firstNonEmpty(strings.TrimSpace(change.Instruction), "按新的画面设计和图片重做这个分镜，旁白和节奏不变。")
		}
	}
	for i := range p.Spec.Scenes {
		sc := &p.Spec.Scenes[i]
		var notes []string
		for _, id := range c.Replaced {
			if videoSceneUses(before[sc.ID], id) || slices.Contains(sc.Images, id) {
				notes = append(notes, fmt.Sprintf("%s 已换成另一张图片（%s）。旧代码里针对原图的标注坐标、裁切和镜头焦点都要按新图重新计算。", id, videoImageDesc(p, id)))
			}
		}
		for _, id := range c.Showed[sc.ID] {
			notes = append(notes, fmt.Sprintf("图片 %s 已被删除：不要再调用 S.image('%s')，改用其余图片或线条图示、文字表现同样的内容。", id, id))
		}
		if len(notes) == 0 {
			continue
		}
		if planned[sc.ID] {
			redo[sc.ID] = strings.Join(notes, "\n") + "\n" + redo[sc.ID]
		} else {
			redo[sc.ID] = strings.Join(notes, "\n") + "\n其余画面和节奏不变。"
		}
	}
	return redo
}

func videoImageDesc(p *VideoProject, id string) string {
	for _, img := range p.Spec.Images {
		if img.ID == id {
			if img.Desc != "" {
				return fmt.Sprintf("%d×%d，%s", img.Width, img.Height, img.Desc)
			}
			return fmt.Sprintf("%d×%d，见附图", img.Width, img.Height)
		}
	}
	return "见附图"
}

func videoMotionImageInstruction(p *VideoProject, c videoImageChanges) string {
	var b strings.Builder
	_, _ = b.WriteString("用户换了这段动画的图片：")
	_, _ = b.WriteString(videoImageChangeText(c))
	ids := make([]string, len(p.Spec.Images))
	for i, img := range p.Spec.Images {
		ids[i] = img.ID
	}
	if len(ids) > 0 {
		fmt.Fprintf(&b, "\n现在的图片是 %s（已附上）。按现在的图片重新安排画面，旧代码里针对原图的坐标、裁切都要重新计算；删掉的图片不要再用。保持循环无缝。", strings.Join(ids, "、"))
	} else {
		_, _ = b.WriteString("\n现在没有图片了：不要再调用 S.image，用线条、图形和文字表现同样的内容。保持循环无缝。")
	}
	return b.String()
}

func videoImageSwapPrompt(p *VideoProject, c videoImageChanges) []LearnMessage {
	var outline strings.Builder
	for _, sc := range p.Spec.Scenes {
		fmt.Fprintf(&outline, "- %s 「%s」 (%.1fs)\n  narration: %s\n  visual: %s\n", sc.ID, sc.Title, sc.Duration, sc.Narration, sc.Visual)
		if len(sc.Images) > 0 {
			fmt.Fprintf(&outline, "  pictures now: %s\n", strings.Join(sc.Images, ", "))
		}
	}
	var pics strings.Builder
	for _, img := range p.Spec.Images {
		switch {
		case slices.Contains(c.Replaced, img.ID):
			fmt.Fprintf(&pics, "- %s (%d×%d): REPLACED by a new picture (attached)\n", img.ID, img.Width, img.Height)
		case slices.Contains(c.Added, img.ID):
			fmt.Fprintf(&pics, "- %s (%d×%d): NEW picture (attached)\n", img.ID, img.Width, img.Height)
		default:
			fmt.Fprintf(&pics, "- %s (%d×%d): unchanged — %s\n", img.ID, img.Width, img.Height, img.Desc)
		}
	}
	for _, id := range c.Removed {
		fmt.Fprintf(&pics, "- %s: REMOVED — no scene may show it any more\n", id)
	}
	schema := map[string]any{
		"reply":       "好的，我把新图放进了第 1、5 个分镜，并按新图重新安排了标注。",
		"image_notes": []map[string]any{{"id": "img2", "desc": "白色注射笔横放在浅灰台面上，笔帽在左（约 x15%），观察窗居中（约 x50%）"}},
		"scenes":      []map[string]any{{"id": "s1", "images": []string{"img2"}, "visual": "该分镜完整的新画面设计（同样的旁白节点）……", "instruction": "给动效设计师的修改说明：换成新图后哪些标注移到哪里、镜头怎么推"}},
	}
	sys := "You are the director of a narrated HTML explainer video. The user changed the pictures of the video: some were replaced by new pictures, some added, some removed. The narration stays exactly as it is (it is already recorded); only the picture side changes.\n" +
		"Look at the new pictures (attached, in the order they are listed as REPLACED / NEW) and answer with:\n" +
		"- image_notes: one line per new picture (replaced or added) saying what is in it and where (positions as % of width / height), so the designer can place labels.\n" +
		"- scenes: every scene that must change. images = the picture ids the scene shows now; visual = its complete new visual plan, tied to the same narration phrases, saying where in the picture labels and arrows point; instruction = a precise note for the motion designer about what to change in the current scene.\n" +
		"Rules: list every scene that shows a replaced picture (its labels, crops and focus point at things in the old picture). List every scene that showed a removed picture and give it another picture or drawn graphics instead. Put each new picture in at least one scene where it fits the narration, unless the user says otherwise. Follow the user's note when there is one. Leave other scenes out. Keep the video's style. Pictures that do not match what the narration says (e.g. a different product) are still used as the user asked; never invent details the picture does not show.\n" +
		"Reply to the user in one or two sentences in their language. Answer with JSON only, shaped like:\n" + videoJSONSchemaHint(schema)
	user := "Video: " + p.Spec.Title + "\nScenes:\n" + outline.String() + "\nPictures:\n" + pics.String()
	if c.Note != "" {
		user += "\nThe user's note: " + c.Note + "\n"
	}
	return []LearnMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}}
}
