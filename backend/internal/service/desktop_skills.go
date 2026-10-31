package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// HiveGPT 桌面助手技能库 (docs/DESKTOP_ASSISTANT_PHASE2_PLAN.md, 二期 B).
//
// Operators edit skills in 后台 → 桌面助手 with a plain form. Each published skill reaches the app
// through GET /api/v1/desktop/skills twice over: as a card in the empty chat (icon, title, summary,
// prepare, prompt, sample prompt) and as a Codex skill, SKILL.md, which the app writes into its private
// CODEX_HOME so Codex also follows it when the user only describes the task in their own words.
// The overview reads the app's desktop_* analytics events and the device keys.

const (
	DesktopSkillDraft     = "draft"
	DesktopSkillPublished = "published"
	DesktopSkillHidden    = "hidden"

	desktopSkillMaxTitle        = 40
	desktopSkillMaxSummary      = 200
	desktopSkillMaxPrompt       = 8000
	desktopSkillMaxInstructions = 20000
)

var desktopSkillSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,46}$`)

var (
	ErrDesktopSkillNotFound = infraerrors.NotFound("DESKTOP_SKILL_NOT_FOUND", "技能不存在（Desktop skill not found）")
	ErrDesktopSkillInvalid  = infraerrors.BadRequest("DESKTOP_SKILL_INVALID", "技能内容不完整或格式不对（Invalid desktop skill）")
	ErrDesktopSkillSlug     = infraerrors.BadRequest("DESKTOP_SKILL_SLUG", "标识只能用小写字母、数字和短横线，2–47 个字符（Invalid skill slug）")
	ErrDesktopSkillSlugUsed = infraerrors.Conflict("DESKTOP_SKILL_SLUG_USED", "这个标识已经被别的技能用了（Skill slug already used）")
	ErrDesktopSkillStatus   = infraerrors.BadRequest("DESKTOP_SKILL_STATUS", "状态只能是草稿、上架或下架（Invalid skill status）")
)

// DesktopSkill is one row of desktop_skills.
type DesktopSkill struct {
	ID           int64     `json:"id"`
	Slug         string    `json:"slug"`
	Title        string    `json:"title"`
	Icon         string    `json:"icon"`
	Summary      string    `json:"summary"`
	Prepare      string    `json:"prepare"`
	Prompt       string    `json:"prompt"`
	SamplePrompt string    `json:"sample_prompt"`
	Instructions string    `json:"instructions"`
	NeedsOffice  bool      `json:"needs_office"`
	Status       string    `json:"status"`
	SortOrder    int       `json:"sort_order"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DesktopSkillForApp is what the app receives: the card plus the ready-made SKILL.md.
type DesktopSkillForApp struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Icon         string `json:"icon"`
	Summary      string `json:"summary"`
	Prepare      string `json:"prepare"`
	Prompt       string `json:"prompt"`
	SamplePrompt string `json:"sample_prompt"`
	NeedsOffice  bool   `json:"needs_office"`
	Version      int    `json:"version"`
	// SkillName is the Codex skill name (hivegpt-<slug>), also the folder the app writes it to.
	SkillName string `json:"skill_name"`
	SkillMD   string `json:"skill_md"`
}

// DesktopFunnelStep counts one desktop_* event over the window.
type DesktopFunnelStep struct {
	Event    string `json:"event"`
	Events   int64  `json:"events"`
	Visitors int64  `json:"visitors"`
}

// DesktopTemplateStat is how one template / skill did (template = slug, sample:<slug> or custom).
type DesktopTemplateStat struct {
	Template   string  `json:"template"`
	Starts     int64   `json:"starts"`
	Done       int64   `json:"done"`
	Failed     int64   `json:"failed"`
	AvgDoneSec float64 `json:"avg_done_sec"`
	TopFailure string  `json:"top_failure"`
}

// DesktopReasonCount is an engine / task failure reason and how often it happened.
type DesktopReasonCount struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// DesktopDevice is one computer's key.
type DesktopDevice struct {
	KeyID      int64      `json:"key_id"`
	Name       string     `json:"name"`
	UserID     int64      `json:"user_id"`
	UserEmail  string     `json:"user_email"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// DesktopOverview is the 后台 → 桌面助手 概览 tab.
type DesktopOverview struct {
	Days          int                   `json:"days"`
	Devices       int64                 `json:"devices"`
	ActiveDevices int64                 `json:"active_devices"` // key used in the last 7 days
	Funnel        []DesktopFunnelStep   `json:"funnel"`
	Returned      int64                 `json:"returned"` // computers opened on two or more days
	Templates     []DesktopTemplateStat `json:"templates"`
	Failures      []DesktopReasonCount  `json:"failures"`
	RecentDevices []DesktopDevice       `json:"recent_devices"`
}

// DesktopFunnelEvents is the order the funnel is shown in.
var DesktopFunnelEvents = []string{
	"desktop_installed", "desktop_open", "desktop_engine_ready", "desktop_login_ok",
	"desktop_task_start", "desktop_task_done", "desktop_undo", "desktop_history_open", "desktop_recharge_click",
}

type DesktopSkillRepository interface {
	ListDesktopSkills(ctx context.Context, publishedOnly bool) ([]DesktopSkill, error)
	GetDesktopSkill(ctx context.Context, id int64) (*DesktopSkill, error)
	// CreateDesktopSkill returns ErrDesktopSkillSlugUsed when the slug is taken.
	CreateDesktopSkill(ctx context.Context, s *DesktopSkill, adminID int64) (int64, error)
	// UpdateDesktopSkill bumps version; ErrDesktopSkillNotFound / ErrDesktopSkillSlugUsed as above.
	UpdateDesktopSkill(ctx context.Context, s *DesktopSkill, adminID int64) error
	SetDesktopSkillStatus(ctx context.Context, id int64, status string, adminID int64) error
	DeleteDesktopSkill(ctx context.Context, id int64) error
	DesktopOverview(ctx context.Context, since time.Time, keyNamePrefix string) (*DesktopOverview, error)
}

type DesktopSkillService struct {
	repo DesktopSkillRepository
	now  func() time.Time
}

func NewDesktopSkillService(repo DesktopSkillRepository) *DesktopSkillService {
	return &DesktopSkillService{repo: repo, now: time.Now}
}

// ForApp lists published skills in display order, each with its SKILL.md.
func (s *DesktopSkillService) ForApp(ctx context.Context) ([]DesktopSkillForApp, error) {
	rows, err := s.repo.ListDesktopSkills(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]DesktopSkillForApp, 0, len(rows))
	for _, r := range rows {
		out = append(out, DesktopSkillForApp{
			Slug: r.Slug, Title: r.Title, Icon: r.Icon, Summary: r.Summary, Prepare: r.Prepare,
			Prompt: r.Prompt, SamplePrompt: r.SamplePrompt, NeedsOffice: r.NeedsOffice, Version: r.Version,
			SkillName: DesktopSkillName(r.Slug), SkillMD: RenderDesktopSkillMD(&r),
		})
	}
	return out, nil
}

func (s *DesktopSkillService) List(ctx context.Context) ([]DesktopSkill, error) {
	return s.repo.ListDesktopSkills(ctx, false)
}

func (s *DesktopSkillService) Get(ctx context.Context, id int64) (*DesktopSkill, error) {
	return s.repo.GetDesktopSkill(ctx, id)
}

func (s *DesktopSkillService) Create(ctx context.Context, in DesktopSkill, adminID int64) (*DesktopSkill, error) {
	if in.Status == "" {
		in.Status = DesktopSkillDraft
	}
	if err := normalizeDesktopSkill(&in); err != nil {
		return nil, err
	}
	id, err := s.repo.CreateDesktopSkill(ctx, &in, adminID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDesktopSkill(ctx, id)
}

func (s *DesktopSkillService) Update(ctx context.Context, id int64, in DesktopSkill, adminID int64) (*DesktopSkill, error) {
	in.ID = id
	if err := normalizeDesktopSkill(&in); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDesktopSkill(ctx, &in, adminID); err != nil {
		return nil, err
	}
	return s.repo.GetDesktopSkill(ctx, id)
}

func (s *DesktopSkillService) SetStatus(ctx context.Context, id int64, status string, adminID int64) error {
	if !validDesktopSkillStatus(status) {
		return ErrDesktopSkillStatus
	}
	return s.repo.SetDesktopSkillStatus(ctx, id, status, adminID)
}

func (s *DesktopSkillService) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteDesktopSkill(ctx, id)
}

// Overview covers the last `days` days (1–90, default 30).
func (s *DesktopSkillService) Overview(ctx context.Context, days int) (*DesktopOverview, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	o, err := s.repo.DesktopOverview(ctx, s.now().Add(-time.Duration(days)*24*time.Hour), DesktopKeyNamePrefix)
	if err != nil {
		return nil, err
	}
	o.Days = days
	return o, nil
}

func validDesktopSkillStatus(st string) bool {
	return st == DesktopSkillDraft || st == DesktopSkillPublished || st == DesktopSkillHidden
}

func normalizeDesktopSkill(s *DesktopSkill) error {
	s.Slug = strings.ToLower(strings.TrimSpace(s.Slug))
	s.Title = strings.TrimSpace(s.Title)
	s.Icon = strings.TrimSpace(s.Icon)
	s.Summary = strings.TrimSpace(s.Summary)
	s.Prepare = strings.TrimSpace(s.Prepare)
	s.Prompt = strings.TrimSpace(s.Prompt)
	s.SamplePrompt = strings.TrimSpace(s.SamplePrompt)
	s.Instructions = strings.TrimSpace(s.Instructions)
	if !desktopSkillSlugPattern.MatchString(s.Slug) {
		return ErrDesktopSkillSlug
	}
	if s.Status != "" && !validDesktopSkillStatus(s.Status) {
		return ErrDesktopSkillStatus
	}
	n := utf8.RuneCountInString
	if s.Title == "" || n(s.Title) > desktopSkillMaxTitle || n(s.Icon) > 8 || n(s.Summary) > desktopSkillMaxSummary ||
		s.Summary == "" || s.Prompt == "" || n(s.Prompt) > desktopSkillMaxPrompt || n(s.SamplePrompt) > desktopSkillMaxPrompt ||
		n(s.Prepare) > 1000 || n(s.Instructions) > desktopSkillMaxInstructions {
		return ErrDesktopSkillInvalid
	}
	return nil
}

// DesktopSkillName is the Codex skill (and folder) name for a slug.
func DesktopSkillName(slug string) string { return "hivegpt-" + slug }

// RenderDesktopSkillMD builds the SKILL.md Codex reads: front matter with name and description (the
// summary, which Codex shows the model to decide when to use the skill), then the instructions.
// Without instructions the card prompt is used, so a skill is never empty.
func RenderDesktopSkillMD(s *DesktopSkill) string {
	desc := strings.Join(strings.Fields(s.Summary), " ")
	body := strings.TrimSpace(s.Instructions)
	if body == "" {
		body = "# " + s.Title + "\n\n" + s.Prompt
	}
	return "---\nname: " + DesktopSkillName(s.Slug) + "\ndescription: " + strconv.Quote(desc) + "\n---\n\n" + body + "\n"
}
