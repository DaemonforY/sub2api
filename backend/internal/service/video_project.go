package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// HiveGPT 视频 (video.<domain>): AI-made narrated HTML videos ("film") and single-scene animations
// ("motion"). A project holds the script, each scene's narration audio and the scene code that the
// player runs; the agent (video_agent.go) fills them in on the server.

const (
	VideoModeFilm   = "film"
	VideoModeMotion = "motion"

	VideoStatusQuestions = "questions" // waiting for the user's answers
	VideoStatusRunning   = "running"
	VideoStatusReady     = "ready"
	VideoStatusFailed    = "failed"
	VideoStatusStopped   = "stopped"

	VideoVisibilityPrivate  = "private"
	VideoVisibilityPending  = "pending" // submitted to the gallery, waiting for review
	VideoVisibilityPublic   = "public"
	VideoVisibilityRejected = "rejected"

	videoPromptMaxChars = 5000
	videoTitleMaxChars  = 60
)

var (
	ErrVideoNotFound   = infraerrors.NotFound("VIDEO_NOT_FOUND", "找不到这个作品，可能已被删除（Video not found）")
	ErrVideoInvalid    = infraerrors.BadRequest("VIDEO_INVALID", "请求格式不正确，请刷新页面后重试（Invalid request）")
	ErrVideoPrompt     = infraerrors.BadRequest("VIDEO_PROMPT", "请先写下想做的视频或动画（最多 5000 字）（Describe your video, up to 5000 characters）")
	ErrVideoBusy       = infraerrors.Conflict("VIDEO_BUSY", "AI 正在处理这个作品，请等它完成或先停止（The agent is still working）")
	ErrVideoTooMany    = infraerrors.TooManyRequests("VIDEO_TOO_MANY", "同时进行的生成已达 2 个，请等其中一个完成后再试（Two generations are already running）")
	ErrVideoKey        = infraerrors.Unauthorized("VIDEO_KEY", "请先用 HiveGPT 账号登录（Sign in with your HiveGPT account）")
	ErrVideoNotReady   = infraerrors.BadRequest("VIDEO_NOT_READY", "作品还没生成完成，完成后才能发布（Finish the video before publishing）")
	ErrVideoForbidden  = infraerrors.Forbidden("VIDEO_FORBIDDEN", "只有管理员可以审核作品（Admins only）")
	ErrVideoNoQuestion = infraerrors.BadRequest("VIDEO_NO_QUESTION", "这个作品现在没有需要回答的问题（Nothing to answer）")
)

// VideoOptions are the create-page choices, kept with the project.
type VideoOptions struct {
	Ratio    string            `json:"ratio"`
	Style    string            `json:"style"`
	Genre    string            `json:"genre,omitempty"`
	Category string            `json:"category,omitempty"`
	Voice    string            `json:"voice,omitempty"`
	Rate     int               `json:"rate,omitempty"`
	Model    string            `json:"model"`
	Length   string            `json:"length,omitempty"` // short / standard / long
	Ask      bool              `json:"ask,omitempty"`    // ask clarifying questions first
	Answers  map[string]string `json:"answers,omitempty"`
	Seconds  int               `json:"seconds,omitempty"` // motion: loop length
}

// VideoSpec is the script and everything the player needs.
type VideoSpec struct {
	Title    string       `json:"title"`
	Summary  string       `json:"summary,omitempty"`
	Audience string       `json:"audience,omitempty"`
	Goal     string       `json:"goal,omitempty"`
	Language string       `json:"language,omitempty"`
	Theme    VideoTheme   `json:"theme"`
	Width    int          `json:"width"`
	Height   int          `json:"height"`
	Loop     bool         `json:"loop,omitempty"`
	Poster   float64      `json:"poster,omitempty"` // seconds: the frame cards and the player show before playing
	Scenes   []VideoScene `json:"scenes"`
}

type VideoScene struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Narration string            `json:"narration,omitempty"`
	Visual    string            `json:"visual"`
	Duration  float64           `json:"duration"`
	Audio     *VideoSceneAudio  `json:"audio,omitempty"`
	Code      string            `json:"code,omitempty"`
	Error     string            `json:"error,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

type VideoSceneAudio struct {
	File     string      `json:"file"`
	Duration float64     `json:"duration"`
	Words    []VideoWord `json:"words,omitempty"`
}

func (s *VideoSpec) TotalDuration() float64 {
	total := 0.0
	for _, sc := range s.Scenes {
		total += sc.Duration
	}
	return total
}

type VideoUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Calls            int `json:"calls"`
	TTSChars         int `json:"tts_chars"`
}

type VideoProject struct {
	ID          string       `json:"id"`
	UserID      int64        `json:"-"`
	APIKeyID    int64        `json:"-"`
	Mode        string       `json:"mode"`
	Title       string       `json:"title"`
	Prompt      string       `json:"prompt"`
	Options     VideoOptions `json:"options"`
	Status      string       `json:"status"`
	Stage       string       `json:"stage,omitempty"`
	Error       string       `json:"error,omitempty"`
	Spec        *VideoSpec   `json:"spec,omitempty"`
	Duration    float64      `json:"duration"`
	Width       int          `json:"width"`
	Height      int          `json:"height"`
	Visibility  string       `json:"visibility"`
	Category    string       `json:"category,omitempty"`
	Featured    bool         `json:"featured,omitempty"`
	Views       int          `json:"views"`
	Remixes     int          `json:"remixes"`
	RemixOf     string       `json:"remix_of,omitempty"`
	Usage       VideoUsage   `json:"usage"`
	Author      string       `json:"author,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	PublishedAt *time.Time   `json:"published_at,omitempty"`
}

// VideoCard is a project in lists (history, gallery) — no scene code.
type VideoCard struct {
	ID         string    `json:"id"`
	Mode       string    `json:"mode"`
	Title      string    `json:"title"`
	Prompt     string    `json:"prompt"`
	Status     string    `json:"status"`
	Stage      string    `json:"stage,omitempty"`
	Visibility string    `json:"visibility"`
	Category   string    `json:"category,omitempty"`
	Featured   bool      `json:"featured,omitempty"`
	Views      int       `json:"views"`
	Remixes    int       `json:"remixes"`
	Duration   float64   `json:"duration"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Style      string    `json:"style,omitempty"`
	Author     string    `json:"author,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VideoEvent is one line of the agent panel: the user's messages, the agent's steps, its questions
// and replies.
type VideoEvent struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"` // user / step / question / answer / assistant / error
	Text      string          `json:"text"`
	Data      json.RawMessage `json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type VideoVersion struct {
	ID        int64      `json:"id"`
	Note      string     `json:"note"`
	Spec      *VideoSpec `json:"spec,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type VideoGalleryQuery struct {
	Category string
	Mode     string
	Search   string
	Sort     string // featured / new / hot
	Page     int
	PageSize int
}

type VideoRepository interface {
	Create(ctx context.Context, p *VideoProject) error
	Get(ctx context.Context, id string) (*VideoProject, error)
	// Save writes title, status, stage, error, spec, duration, usage and options.
	Save(ctx context.Context, p *VideoProject) error
	Delete(ctx context.Context, userID int64, id string) (bool, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]VideoCard, error)
	CountRunning(ctx context.Context, userID int64) (int, error)
	FailRunning(ctx context.Context, message string) ([]string, error)
	AddEvent(ctx context.Context, projectID string, e *VideoEvent) error
	ListEvents(ctx context.Context, projectID string, afterID int64, limit int) ([]VideoEvent, error)
	AddVersion(ctx context.Context, projectID, note string, spec *VideoSpec) error
	ListVersions(ctx context.Context, projectID string, limit int) ([]VideoVersion, error)
	GetVersion(ctx context.Context, projectID string, id int64) (*VideoVersion, error)
	SetVisibility(ctx context.Context, id, visibility, category, title string, featured *bool) error
	Gallery(ctx context.Context, q VideoGalleryQuery) ([]VideoCard, int, error)
	Pending(ctx context.Context, limit int) ([]VideoCard, error)
	AddView(ctx context.Context, id string) error
	AddRemix(ctx context.Context, id string) error
}

// VideoAuthorName shows a user in the gallery without exposing their email.
func VideoAuthorName(username, email string) string {
	if name := strings.TrimSpace(username); name != "" {
		return name
	}
	local, _, _ := strings.Cut(email, "@")
	if utf8.RuneCountInString(local) <= 2 {
		return "HiveGPT 用户"
	}
	r := []rune(local)
	return string(r[:2]) + "***"
}

func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// videoTitleFromPrompt is the provisional title until the script names the video: the first line,
// or its part before a colon ("HiveGPT：六边形…" → "HiveGPT"), cut at a clause end, never leaving an
// unclosed bracket.
func videoTitleFromPrompt(prompt string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(prompt), "\n")
	line = strings.TrimSpace(strings.TrimLeft(line, "#*- "))
	if head, _, ok := strings.Cut(line, "："); ok && utf8.RuneCountInString(strings.TrimSpace(head)) >= 2 {
		line = strings.TrimSpace(head)
	}
	r := []rune(line)
	if len(r) > 24 {
		cut := 24
		for i := 23; i >= 10; i-- {
			if strings.ContainsRune("，。、；！？,;", r[i]) {
				cut = i
				break
			}
		}
		r = r[:cut]
	}
	line = strings.TrimRight(string(r), "，。、；：！？,;:「『（(《“ ")
	if line == "" {
		return "未命名作品"
	}
	return line
}
