package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Monthly contests from a template. While the switch is on, the contest loop creates and publishes
// each month's contest LeadDays before the 1st (China time): submissions from the 1st to the end of
// SubmissionEndDay, votes from the 1st to VotingEndTime on the last day. Settlement is the usual
// automatic one; prizes are still granted by an admin. Turning the switch off stops future months
// only. A month is created at most once, so a cancelled or deleted contest is not brought back.

// ContestMonthlyCover is the cover the site shows for scheduled contests that set none
// (frontend/public/contest-media; the frontend's MONTHLY_CONTEST_COVER).
const ContestMonthlyCover = "/contest-media/monthly-draw.webp"

// ContestCoverOrDefault is the contest's cover, or the monthly cover for scheduled contests.
func ContestCoverOrDefault(c *Contest) string {
	if c.CoverImage == "" && c.AutoPeriod != "" {
		return ContestMonthlyCover
	}
	return c.CoverImage
}

// contestAutoZone: the site's contests run on China time, whatever the server's TZ.
var contestAutoZone = time.FixedZone("Asia/Shanghai", 8*3600)

var contestAutoTimeRe = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// ContestAutoTemplate is the monthly contest. {year} and {month} in the texts become the month's.
type ContestAutoTemplate struct {
	TitleTemplate      string         `json:"title_template"`
	Description        string         `json:"description"`
	Rules              string         `json:"rules"`
	CoverImage         string         `json:"cover_image"`
	SubmissionEndDay   int            `json:"submission_end_day"` // submissions close at the end of this day (1–28)
	VotingEndTime      string         `json:"voting_end_time"`    // "20:00" on the month's last day
	LeadDays           int            `json:"lead_days"`          // published this many days before the 1st (0–10)
	MaxEntriesPerUser  int            `json:"max_entries_per_user"`
	VotesPerUser       int            `json:"votes_per_user"`
	AllowSelfVote      bool           `json:"allow_self_vote"`
	RequireReview      bool           `json:"require_review"`
	MinAccountAgeHours int            `json:"min_account_age_hours"`
	OnePrizePerUser    bool           `json:"one_prize_per_user"`
	MinVotesForPrize   int            `json:"min_votes_for_prize"`
	Prizes             []ContestPrize `json:"prizes"`
}

// ContestAutoState is the stored switch, template and the last month created.
type ContestAutoState struct {
	Enabled    bool
	Template   ContestAutoTemplate
	LastPeriod string
	UpdatedBy  *int64
	UpdatedAt  time.Time
}

// ContestAutoMonth is how one month stands, for the admin page.
type ContestAutoMonth struct {
	Period            string    `json:"period"` // "2026-11"
	Title             string    `json:"title"`
	SubmissionStartAt time.Time `json:"submission_start_at"`
	SubmissionEndAt   time.Time `json:"submission_end_at"`
	VotingEndAt       time.Time `json:"voting_end_at"`
	CreateAt          time.Time `json:"create_at"`
	// Status: created / scheduled (on, waiting for CreateAt) / paused (switch off) /
	// missed (submissions already closed, never created) / skipped (created before, then cancelled or deleted).
	Status    string `json:"status"`
	ContestID int64  `json:"contest_id,omitempty"`
	Phase     string `json:"phase,omitempty"`
}

// ContestAutoSettings is the admin view.
type ContestAutoSettings struct {
	Enabled   bool                `json:"enabled"`
	Template  ContestAutoTemplate `json:"template"`
	UpdatedAt *time.Time          `json:"updated_at,omitempty"`
	Months    []ContestAutoMonth  `json:"months"`
}

// DefaultContestAutoTemplate is the monthly drawing contest agreed on 2026-10-29.
func DefaultContestAutoTemplate() ContestAutoTemplate {
	return ContestAutoTemplate{
		TitleTemplate: "{month} 月 AI 画图赛",
		Description:   "用 AI 画一张图来投稿，主题不限。每月 1 日开始，月底按点赞数排名，第一名获得 1 个月 ¥120 月度订阅。",
		Rules: "1. 投稿须是在本站用 AI 生成的作品，每人最多投 3 件，审核通过后展示。\n" +
			"2. 注册用户每人 5 个赞，同一作品只能点 1 个，不能给自己点。\n" +
			"3. 点赞截止时按赞数排名，赞数相同时先达到的优先；至少 5 个有效赞才能获奖。\n" +
			"4. 刷票的作品取消资格。结果公示 3 天，无异议后发奖。",
		SubmissionEndDay:   22,
		VotingEndTime:      "20:00",
		LeadDays:           3,
		MaxEntriesPerUser:  3,
		VotesPerUser:       5,
		RequireReview:      true,
		MinAccountAgeHours: 0,
		OnePrizePerUser:    true,
		MinVotesForPrize:   5,
		Prizes:             []ContestPrize{{RankFrom: 1, RankTo: 1, Type: ContestPrizeCustom, Label: "¥120 月度订阅（1 个月）"}},
	}
}

func contestAutoPeriod(year int, month time.Month) string {
	return fmt.Sprintf("%04d-%02d", year, int(month))
}

func contestAutoText(s string, year int, month time.Month) string {
	return strings.NewReplacer("{year}", fmt.Sprint(year), "{month}", fmt.Sprint(int(month))).Replace(s)
}

// BuildAutoContest is the template's contest for one month (published, China time).
func BuildAutoContest(t ContestAutoTemplate, year int, month time.Month) (*Contest, error) {
	m := contestAutoTimeRe.FindStringSubmatch(t.VotingEndTime)
	if m == nil {
		return nil, contestInvalid("点赞截止时间格式应为 HH:MM，如 20:00")
	}
	if t.SubmissionEndDay < 1 || t.SubmissionEndDay > 28 {
		return nil, contestInvalid("投稿截止日应在 1–28 号之间")
	}
	var hh, mm int
	_, _ = fmt.Sscanf(m[1]+" "+m[2], "%d %d", &hh, &mm)
	start := time.Date(year, month, 1, 0, 0, 0, 0, contestAutoZone)
	next := start.AddDate(0, 1, 0)
	votingEnd := time.Date(year, month, next.AddDate(0, 0, -1).Day(), hh, mm, 0, 0, contestAutoZone)
	// 23:59:59 rather than the next midnight, so pages show 「22 日 23:59」, not 「23 日 00:00」.
	submissionEnd := time.Date(year, month, t.SubmissionEndDay, 23, 59, 59, 0, contestAutoZone)
	if submissionEnd.After(votingEnd) {
		submissionEnd = votingEnd
	}
	c := &Contest{
		Title:       contestAutoText(t.TitleTemplate, year, month),
		Description: contestAutoText(t.Description, year, month),
		Rules:       contestAutoText(t.Rules, year, month),
		CoverImage:  t.CoverImage,
		Status:      ContestStatusPublished,
		AutoPeriod:  contestAutoPeriod(year, month),

		SubmissionStartAt: start, SubmissionEndAt: submissionEnd,
		VotingStartAt: start, VotingEndAt: votingEnd,
		MaxEntriesPerUser: t.MaxEntriesPerUser, VotesPerUser: t.VotesPerUser,
		AllowSelfVote: t.AllowSelfVote, RequireReview: t.RequireReview,
		MinAccountAgeHours: t.MinAccountAgeHours, OnePrizePerUser: t.OnePrizePerUser,
		MinVotesForPrize: t.MinVotesForPrize, Prizes: append([]ContestPrize(nil), t.Prizes...),
	}
	if err := NormalizeAndValidateContest(c); err != nil {
		return nil, err
	}
	return c, nil
}

// NormalizeContestAutoTemplate trims the template and checks it builds a valid contest.
func NormalizeContestAutoTemplate(t *ContestAutoTemplate) error {
	t.TitleTemplate = strings.TrimSpace(t.TitleTemplate)
	t.VotingEndTime = strings.TrimSpace(t.VotingEndTime)
	if t.TitleTemplate == "" {
		return contestInvalid("请填写比赛标题")
	}
	if t.LeadDays < 0 || t.LeadDays > 10 {
		return contestInvalid("提前发布天数应在 0–10 之间")
	}
	c, err := BuildAutoContest(*t, 2026, time.February) // the shortest month catches day-range mistakes
	if err != nil {
		return err
	}
	t.Description, t.Rules, t.CoverImage = strings.TrimSpace(t.Description), strings.TrimSpace(t.Rules), c.CoverImage
	t.Prizes = c.Prizes
	return nil
}

// contestAutoMonths are this month and next month, as (year, month) in China time.
func contestAutoMonths(now time.Time) [2][2]int {
	cur := now.In(contestAutoZone)
	first := time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, contestAutoZone)
	nxt := first.AddDate(0, 1, 0)
	return [2][2]int{{first.Year(), int(first.Month())}, {nxt.Year(), int(nxt.Month())}}
}

// contestAutoDue says whether the month's contest should be created now.
func contestAutoDue(c *Contest, leadDays int, lastPeriod string, now time.Time) bool {
	if c.AutoPeriod <= lastPeriod {
		return false
	}
	open := c.SubmissionStartAt.AddDate(0, 0, -leadDays)
	return !now.Before(open) && now.Before(c.SubmissionEndAt)
}

func (s *ContestService) autoState(ctx context.Context) (*ContestAutoState, error) {
	st, err := s.repo.GetAutoState(ctx)
	if err != nil {
		return nil, err
	}
	if st == nil {
		st = &ContestAutoState{Template: DefaultContestAutoTemplate()}
	}
	return st, nil
}

// GetAutoSettings returns the switch, the template and how this month and next month stand.
func (s *ContestService) GetAutoSettings(ctx context.Context) (*ContestAutoSettings, error) {
	st, err := s.autoState(ctx)
	if err != nil {
		return nil, err
	}
	out := &ContestAutoSettings{Enabled: st.Enabled, Template: st.Template, Months: []ContestAutoMonth{}}
	if !st.UpdatedAt.IsZero() {
		at := st.UpdatedAt
		out.UpdatedAt = &at
	}
	existing := map[string]*Contest{}
	if list, err := s.repo.ListContests(ctx, nil); err == nil {
		for _, c := range list {
			if c.AutoPeriod != "" {
				existing[c.AutoPeriod] = s.decorate(c)
			}
		}
	} else {
		return nil, err
	}
	now := s.now()
	for _, ym := range contestAutoMonths(now) {
		c, err := BuildAutoContest(st.Template, ym[0], time.Month(ym[1]))
		if err != nil {
			continue // a template saved before a rule change; the editor shows the error on save
		}
		m := ContestAutoMonth{
			Period: c.AutoPeriod, Title: c.Title,
			SubmissionStartAt: c.SubmissionStartAt, SubmissionEndAt: c.SubmissionEndAt, VotingEndAt: c.VotingEndAt,
			CreateAt: c.SubmissionStartAt.AddDate(0, 0, -st.Template.LeadDays),
		}
		switch {
		case existing[m.Period] != nil:
			m.Status, m.ContestID, m.Phase, m.Title = "created", existing[m.Period].ID, existing[m.Period].Phase, existing[m.Period].Title
		case !now.Before(c.SubmissionEndAt):
			m.Status = "missed"
		case m.Period <= st.LastPeriod:
			m.Status = "skipped"
		case !st.Enabled:
			m.Status = "paused"
		default:
			m.Status = "scheduled"
		}
		out.Months = append(out.Months, m)
	}
	return out, nil
}

// SaveAutoSettings stores the switch and template, then creates whatever is due right away.
func (s *ContestService) SaveAutoSettings(ctx context.Context, enabled bool, tpl ContestAutoTemplate, adminID int64) (*ContestAutoSettings, error) {
	if err := NormalizeContestAutoTemplate(&tpl); err != nil {
		return nil, err
	}
	var by *int64
	if adminID > 0 {
		by = &adminID
	}
	if err := s.repo.SaveAutoState(ctx, enabled, tpl, by); err != nil {
		return nil, err
	}
	if enabled {
		s.RunAutoContests(ctx)
	}
	return s.GetAutoSettings(ctx)
}

// RunAutoContests creates this month's or next month's contest when it is due (called every minute).
func (s *ContestService) RunAutoContests(ctx context.Context) {
	st, err := s.repo.GetAutoState(ctx)
	if err != nil {
		logger.LegacyPrintf("service.contest", "auto contests: load settings failed: %v", err)
		return
	}
	if st == nil || !st.Enabled {
		return
	}
	now := s.now()
	for _, ym := range contestAutoMonths(now) {
		c, err := BuildAutoContest(st.Template, ym[0], time.Month(ym[1]))
		if err != nil {
			logger.LegacyPrintf("service.contest", "auto contests: template invalid: %v", err)
			return
		}
		if !contestAutoDue(c, st.Template.LeadDays, st.LastPeriod, now) {
			continue
		}
		created, err := s.repo.CreateAutoContest(ctx, c)
		if err != nil {
			logger.LegacyPrintf("service.contest", "auto contests: create %s failed: %v", c.AutoPeriod, err)
			return
		}
		if created {
			logger.LegacyPrintf("service.contest", "auto contests: created contest %d for %s", c.ID, c.AutoPeriod)
		}
		st.LastPeriod = c.AutoPeriod
	}
}
