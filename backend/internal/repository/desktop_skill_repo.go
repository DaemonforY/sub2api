package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type desktopSkillRepository struct {
	db *sql.DB
}

// NewDesktopSkillRepository stores the 桌面助手技能库 and reads the 桌面助手 overview (raw SQL).
func NewDesktopSkillRepository(db *sql.DB) service.DesktopSkillRepository {
	return &desktopSkillRepository{db: db}
}

const desktopSkillColumns = `id, slug, title, icon, summary, prepare, prompt, sample_prompt, instructions, needs_office,
    status, sort_order, version, created_at, updated_at`

func scanDesktopSkill(row interface{ Scan(...any) error }) (*service.DesktopSkill, error) {
	var s service.DesktopSkill
	err := row.Scan(&s.ID, &s.Slug, &s.Title, &s.Icon, &s.Summary, &s.Prepare, &s.Prompt, &s.SamplePrompt,
		&s.Instructions, &s.NeedsOffice, &s.Status, &s.SortOrder, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *desktopSkillRepository) ListDesktopSkills(ctx context.Context, publishedOnly bool) ([]service.DesktopSkill, error) {
	q := `SELECT ` + desktopSkillColumns + ` FROM desktop_skills`
	if publishedOnly {
		q += ` WHERE status = 'published'`
	}
	q += ` ORDER BY sort_order, id`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.DesktopSkill{}
	for rows.Next() {
		s, err := scanDesktopSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *desktopSkillRepository) GetDesktopSkill(ctx context.Context, id int64) (*service.DesktopSkill, error) {
	s, err := scanDesktopSkill(r.db.QueryRowContext(ctx, `SELECT `+desktopSkillColumns+` FROM desktop_skills WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrDesktopSkillNotFound
	}
	return s, err
}

func (r *desktopSkillRepository) CreateDesktopSkill(ctx context.Context, s *service.DesktopSkill, adminID int64) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
INSERT INTO desktop_skills (slug, title, icon, summary, prepare, prompt, sample_prompt, instructions, needs_office,
    status, sort_order, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, 0), NULLIF($12, 0))
RETURNING id`,
		s.Slug, s.Title, s.Icon, s.Summary, s.Prepare, s.Prompt, s.SamplePrompt, s.Instructions, s.NeedsOffice,
		s.Status, s.SortOrder, adminID).Scan(&id)
	if isUniqueViolation(err) {
		return 0, service.ErrDesktopSkillSlugUsed
	}
	return id, err
}

// withRevision runs change in a transaction after copying the skill's current row into
// desktop_skill_revisions (action = what replaces it). No row → ErrDesktopSkillNotFound, nothing kept.
func (r *desktopSkillRepository) withRevision(ctx context.Context, id int64, action string, adminID int64, change func(tx *sql.Tx) (sql.Result, error)) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
INSERT INTO desktop_skill_revisions (skill_id, version, action, actor_id, snapshot)
SELECT id, version, $2, NULLIF($3, 0), to_jsonb(s) FROM desktop_skills s WHERE id = $1`, id, action, adminID)
	if err := desktopSkillOneRow(res, err); err != nil {
		return err
	}
	res, err = change(tx)
	if isUniqueViolation(err) {
		return service.ErrDesktopSkillSlugUsed
	}
	if err := desktopSkillOneRow(res, err); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *desktopSkillRepository) ListDesktopSkillRevisions(ctx context.Context, skillID int64, limit int) ([]service.DesktopSkillRevision, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT rv.id, rv.skill_id, rv.version, rv.action, COALESCE(rv.actor_id, 0), COALESCE(u.email, ''), rv.snapshot, rv.created_at
FROM desktop_skill_revisions rv LEFT JOIN users u ON u.id = rv.actor_id
WHERE rv.skill_id = $1 ORDER BY rv.id DESC LIMIT $2`, skillID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.DesktopSkillRevision{}
	for rows.Next() {
		var rv service.DesktopSkillRevision
		var snap []byte
		if err := rows.Scan(&rv.ID, &rv.SkillID, &rv.Version, &rv.Action, &rv.ActorID, &rv.ActorEmail, &snap, &rv.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snap, &rv.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func (r *desktopSkillRepository) UpdateDesktopSkill(ctx context.Context, s *service.DesktopSkill, adminID int64) error {
	// Status is changed only through SetDesktopSkillStatus; an empty status here keeps the current one.
	return r.withRevision(ctx, s.ID, "update", adminID, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `
UPDATE desktop_skills SET slug = $2, title = $3, icon = $4, summary = $5, prepare = $6, prompt = $7,
    sample_prompt = $8, instructions = $9, needs_office = $10, sort_order = $11,
    status = COALESCE(NULLIF($12, ''), status), version = version + 1, updated_by = NULLIF($13, 0), updated_at = NOW()
WHERE id = $1`,
			s.ID, s.Slug, s.Title, s.Icon, s.Summary, s.Prepare, s.Prompt, s.SamplePrompt, s.Instructions, s.NeedsOffice,
			s.SortOrder, s.Status, adminID)
	})
}

func (r *desktopSkillRepository) SetDesktopSkillStatus(ctx context.Context, id int64, status string, adminID int64) error {
	// The app only compares versions, so a status change bumps it too (a re-published skill is re-sent).
	return r.withRevision(ctx, id, "status", adminID, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `
UPDATE desktop_skills SET status = $2, version = version + 1, updated_by = NULLIF($3, 0), updated_at = NOW()
WHERE id = $1`, id, status, adminID)
	})
}

func (r *desktopSkillRepository) DeleteDesktopSkill(ctx context.Context, id int64, adminID int64) error {
	return r.withRevision(ctx, id, "delete", adminID, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `DELETE FROM desktop_skills WHERE id = $1`, id)
	})
}

func desktopSkillOneRow(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrDesktopSkillNotFound
	}
	return nil
}

func (r *desktopSkillRepository) DesktopOverview(ctx context.Context, since time.Time, keyNamePrefix string) (*service.DesktopOverview, error) {
	o := &service.DesktopOverview{
		Funnel: []service.DesktopFunnelStep{}, Templates: []service.DesktopTemplateStat{},
		Failures: []service.DesktopReasonCount{}, RecentDevices: []service.DesktopDevice{},
	}
	like := escapeLikePattern(keyNamePrefix) + "%"

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*), COUNT(*) FILTER (WHERE last_used_at >= NOW() - INTERVAL '7 days')
FROM api_keys WHERE deleted_at IS NULL AND name LIKE $1`, like).Scan(&o.Devices, &o.ActiveDevices); err != nil {
		return nil, err
	}

	// Funnel: every desktop_* event, in the order the page shows them.
	counts := map[string]service.DesktopFunnelStep{}
	rows, err := r.db.QueryContext(ctx, `
SELECT event, COUNT(*), COUNT(DISTINCT visitor_id)
FROM analytics_events WHERE app = 'desktop' AND created_at >= $1 GROUP BY event`, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var st service.DesktopFunnelStep
		if err := rows.Scan(&st.Event, &st.Events, &st.Visitors); err != nil {
			_ = rows.Close()
			return nil, err
		}
		counts[st.Event] = st
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, ev := range service.DesktopFunnelEvents {
		st := counts[ev]
		st.Event = ev
		o.Funnel = append(o.Funnel, st)
	}

	// Came back: computers opened on two or more different days (China time).
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM (
    SELECT visitor_id FROM analytics_events
    WHERE app = 'desktop' AND event = 'desktop_open' AND created_at >= $1
    GROUP BY visitor_id HAVING COUNT(DISTINCT (created_at AT TIME ZONE 'Asia/Shanghai')::date) >= 2
) t`, since).Scan(&o.Returned); err != nil {
		return nil, err
	}

	// Per template: starts / done / failed, average seconds of the completed ones, most common failure.
	rows, err = r.db.QueryContext(ctx, `
WITH t AS (
    SELECT event, COALESCE(NULLIF(props->>'template', ''), 'custom') AS template, props
    FROM analytics_events
    WHERE app = 'desktop' AND created_at >= $1
      AND event IN ('desktop_task_start', 'desktop_task_done', 'desktop_task_failed')
), fail AS (
    SELECT template, props->>'reason' AS reason, COUNT(*) AS n,
           ROW_NUMBER() OVER (PARTITION BY template ORDER BY COUNT(*) DESC) AS rk
    FROM t WHERE event = 'desktop_task_failed' GROUP BY template, props->>'reason'
)
SELECT t.template,
       COUNT(*) FILTER (WHERE event = 'desktop_task_start'),
       COUNT(*) FILTER (WHERE event = 'desktop_task_done'),
       COUNT(*) FILTER (WHERE event = 'desktop_task_failed'),
       COALESCE(AVG((props->>'secs')::numeric) FILTER (WHERE event = 'desktop_task_done' AND props->>'secs' ~ '^[0-9]+$'), 0),
       COALESCE(MAX(fail.reason), '')
FROM t LEFT JOIN fail ON fail.template = t.template AND fail.rk = 1
GROUP BY t.template
ORDER BY 2 DESC, 1
LIMIT 100`, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var st service.DesktopTemplateStat
		if err := rows.Scan(&st.Template, &st.Starts, &st.Done, &st.Failed, &st.AvgDoneSec, &st.TopFailure); err != nil {
			_ = rows.Close()
			return nil, err
		}
		o.Templates = append(o.Templates, st)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Engine / office kit install failures by reason.
	rows, err = r.db.QueryContext(ctx, `
SELECT COALESCE(props->>'step', ''), COALESCE(props->>'reason', ''), COUNT(*)
FROM analytics_events
WHERE app = 'desktop' AND event = 'desktop_engine_failed' AND created_at >= $1
GROUP BY 1, 2 ORDER BY 3 DESC LIMIT 20`, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var f service.DesktopReasonCount
		if err := rows.Scan(&f.Step, &f.Reason, &f.Count); err != nil {
			_ = rows.Close()
			return nil, err
		}
		o.Failures = append(o.Failures, f)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Most recent computers.
	rows, err = r.db.QueryContext(ctx, `
SELECT k.id, k.name, k.user_id, COALESCE(u.email, ''), k.status, k.last_used_at, k.created_at
FROM api_keys k LEFT JOIN users u ON u.id = k.user_id
WHERE k.deleted_at IS NULL AND k.name LIKE $1
ORDER BY k.last_used_at DESC NULLS LAST, k.created_at DESC
LIMIT 50`, like)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var d service.DesktopDevice
		var last sql.NullTime
		if err := rows.Scan(&d.KeyID, &d.Name, &d.UserID, &d.UserEmail, &d.Status, &last, &d.CreatedAt); err != nil {
			return nil, err
		}
		if last.Valid {
			t := last.Time
			d.LastUsedAt = &t
		}
		o.RecentDevices = append(o.RecentDevices, d)
	}
	return o, rows.Err()
}
