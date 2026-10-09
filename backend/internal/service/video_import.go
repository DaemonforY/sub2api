package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Import adds a finished work an admin wrote outside the agent (showcase works for the gallery):
// the scenes come with their code, narration is recorded here like the agent does, and the work
// is published at once. It belongs to the admin's account and key like any other project.
type VideoImportInput struct {
	Mode     string       `json:"mode"`
	Title    string       `json:"title"`
	Prompt   string       `json:"prompt"`
	Options  VideoOptions `json:"options"`
	Spec     VideoSpec    `json:"spec"`
	Featured bool         `json:"featured"`
	Private  bool         `json:"private"` // keep it in the admin's history only
}

func (s *VideoService) Import(ctx context.Context, key *APIKey, in VideoImportInput) (*VideoProject, error) {
	if key == nil || key.User == nil || key.User.Role != RoleAdmin {
		return nil, ErrVideoForbidden
	}
	if in.Mode != VideoModeMotion {
		in.Mode = VideoModeFilm
	}
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.Prompt == "" || len([]rune(in.Prompt)) > videoPromptMaxChars {
		return nil, ErrVideoPrompt
	}
	if len(in.Spec.Scenes) == 0 || len(in.Spec.Scenes) > 20 {
		return nil, ErrVideoInvalid
	}
	normalizeVideoOptions(in.Mode, &in.Options)
	ratio := findVideoRatio(in.Options.Ratio)
	spec := in.Spec
	if spec.Width <= 0 || spec.Height <= 0 {
		spec.Width, spec.Height = ratio.Width, ratio.Height
	}
	if spec.Theme.Background == "" {
		spec.Theme = findVideoStyle(in.Options.Style).Theme
	}
	spec.Loop = in.Mode == VideoModeMotion
	for i := range spec.Scenes {
		sc := &spec.Scenes[i]
		if sc.ID == "" {
			sc.ID = fmt.Sprintf("s%d", i+1)
		}
		if problem := checkSceneCode(sc.Code); problem != "" {
			return nil, infraerrors.BadRequest("VIDEO_IMPORT_CODE", fmt.Sprintf("分镜 %d 的代码有问题：%s", i+1, problem))
		}
		sc.Audio, sc.Error = nil, ""
		if in.Mode == VideoModeMotion {
			sc.Narration = ""
		}
		if sc.Duration <= 0 {
			sc.Duration = videoMinSceneSeconds
		}
	}
	title := clipRunes(firstNonEmpty(strings.TrimSpace(in.Title), spec.Title, videoTitleFromPrompt(in.Prompt)), videoTitleMaxChars)
	spec.Title = title
	category := in.Options.Category
	if category == "" && in.Mode == VideoModeFilm {
		category = "film"
	}
	p := &VideoProject{
		UserID: key.UserID, APIKeyID: key.ID, Mode: in.Mode, Title: title, Prompt: in.Prompt, Options: in.Options,
		Status: VideoStatusRunning, Width: spec.Width, Height: spec.Height, Visibility: VideoVisibilityPrivate, Category: category,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: p.Prompt})
	p.Spec = &spec
	r := &videoRun{userID: p.UserID, p: p}
	if err := s.voiceScenes(ctx, r); err != nil {
		p.Status, p.Error = VideoStatusFailed, videoErrorText(err)
		s.save(ctx, r)
		return nil, err
	}
	p.Status, p.Stage, p.Error = VideoStatusReady, "", ""
	s.save(ctx, r)
	s.done(ctx, r, fmt.Sprintf("完成：%d 个分镜，共 %.0f 秒。可以在右边继续让 AI 修改。", len(spec.Scenes), spec.TotalDuration()))
	_ = s.repo.AddVersion(ctx, p.ID, "初版", p.Spec)
	if !in.Private {
		featured := in.Featured
		if err := s.repo.SetVisibility(ctx, p.ID, VideoVisibilityPublic, category, title, &featured); err != nil {
			return nil, err
		}
	}
	return s.repo.Get(ctx, p.ID)
}
