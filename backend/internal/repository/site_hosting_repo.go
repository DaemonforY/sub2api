package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type siteHostingRepository struct {
	db *sql.DB
}

// NewSiteHostingRepository creates the static-site hosting repository (raw SQL).
func NewSiteHostingRepository(db *sql.DB) service.SiteHostingRepository {
	return &siteHostingRepository{db: db}
}

const siteSelect = `
SELECT s.id, s.user_id, COALESCE(u.email, ''), s.name, s.title, s.status, s.status_reason, s.version, s.size_bytes, s.file_count,
       s.paid, s.paid_until, s.lapsed_at, s.created_at, s.updated_at, s.pending_version, s.password_hash,
       COALESCE((SELECT SUM(views) FROM site_daily_stats d WHERE d.site_id = s.id AND d.day >= CURRENT_DATE - 6), 0),
       COALESCE((SELECT SUM(views) FROM site_daily_stats d WHERE d.site_id = s.id), 0)
FROM sites s LEFT JOIN users u ON u.id = s.user_id`

func scanSite(row rowScanner) (*service.Site, error) {
	var site service.Site
	var paidUntil, lapsedAt sql.NullTime
	if err := row.Scan(&site.ID, &site.UserID, &site.UserEmail, &site.Name, &site.Title, &site.Status, &site.StatusReason, &site.Version,
		&site.SizeBytes, &site.FileCount, &site.Paid, &paidUntil, &lapsedAt, &site.CreatedAt, &site.UpdatedAt, &site.PendingVersion, &site.PasswordHash,
		&site.Views7d, &site.ViewsTotal); err != nil {
		return nil, err
	}
	site.HasPassword = site.PasswordHash != ""
	if paidUntil.Valid {
		site.PaidUntil = &paidUntil.Time
	}
	if lapsedAt.Valid {
		site.LapsedAt = &lapsedAt.Time
	}
	return &site, nil
}

func (r *siteHostingRepository) querySites(ctx context.Context, query string, args ...any) ([]service.Site, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *site)
	}
	return out, rows.Err()
}

func (r *siteHostingRepository) getOne(ctx context.Context, where string, arg any) (*service.Site, error) {
	site, err := scanSite(r.db.QueryRowContext(ctx, siteSelect+" WHERE "+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return site, err
}

func (r *siteHostingRepository) CreateSite(ctx context.Context, site *service.Site) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO sites (user_id, name, title, status, paid) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		site.UserID, site.Name, site.Title, site.Status, site.Paid).Scan(&site.ID, &site.CreatedAt, &site.UpdatedAt)
}

func (r *siteHostingRepository) GetSite(ctx context.Context, id int64) (*service.Site, error) {
	return r.getOne(ctx, "s.id = $1", id)
}

func (r *siteHostingRepository) GetSiteByName(ctx context.Context, name string) (*service.Site, error) {
	return r.getOne(ctx, "s.name = $1", name)
}

func (r *siteHostingRepository) ListSitesByUser(ctx context.Context, userID int64) ([]service.Site, error) {
	return r.querySites(ctx, siteSelect+" WHERE s.user_id = $1 ORDER BY s.created_at, s.id", userID)
}

func (r *siteHostingRepository) ListSites(ctx context.Context, q service.SiteListQuery) ([]service.Site, int64, error) {
	where := []string{"TRUE"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.Keyword != "" {
		add("(s.name ILIKE ? OR s.title ILIKE ? OR u.email ILIKE ?)", "%"+escapeLike(q.Keyword)+"%")
	}
	if q.Status != "" {
		add("s.status = ?", q.Status)
	}
	cond := " WHERE " + strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sites s LEFT JOIN users u ON u.id = s.user_id`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	sites, err := r.querySites(ctx, siteSelect+cond+" ORDER BY s.updated_at DESC, s.id DESC LIMIT $"+strconv.Itoa(len(args)-1)+" OFFSET $"+strconv.Itoa(len(args)), args...)
	return sites, total, err
}

func (r *siteHostingRepository) ListSiteOwners(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT user_id FROM sites ORDER BY user_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *siteHostingRepository) LatestVersion(ctx context.Context, siteID int64) (int, error) {
	var v int
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM site_versions WHERE site_id = $1`, siteID).Scan(&v)
	return v, err
}

func (r *siteHostingRepository) AddSiteVersion(ctx context.Context, siteID int64, version int, size int64, files int, review service.SiteReview) error {
	flags, _ := json.Marshal(nonNilStrings(review.Flags))
	_, err := r.db.ExecContext(ctx, `
INSERT INTO site_versions (site_id, version, size_bytes, file_count, review_status, review_reason, review_flags, reviewed_by, excerpt)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
ON CONFLICT (site_id, version) DO UPDATE SET size_bytes = EXCLUDED.size_bytes, file_count = EXCLUDED.file_count, review_status = EXCLUDED.review_status,
    review_reason = EXCLUDED.review_reason, review_flags = EXCLUDED.review_flags, reviewed_by = EXCLUDED.reviewed_by, excerpt = EXCLUDED.excerpt, created_at = NOW()`,
		siteID, version, size, files, review.Status, review.Reason, string(flags), review.By, review.Excerpt)
	return err
}

func (r *siteHostingRepository) ServeSiteVersion(ctx context.Context, siteID int64, version int, clearPending bool) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE sites s SET version = v.version, size_bytes = v.size_bytes, file_count = v.file_count,
    pending_version = CASE WHEN $3 THEN 0 ELSE s.pending_version END, updated_at = NOW()
FROM site_versions v WHERE s.id = $1 AND v.site_id = s.id AND v.version = $2`, siteID, version, clearPending)
	return err
}

func (r *siteHostingRepository) SetPendingVersion(ctx context.Context, siteID int64, version int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
UPDATE site_versions SET review_status = 'superseded'
WHERE site_id = $1 AND review_status = 'pending' AND version <> $2`, siteID, version); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sites SET pending_version = $2, updated_at = NOW() WHERE id = $1`, siteID, version); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *siteHostingRepository) SetVersionReview(ctx context.Context, siteID int64, version int, status, reason, by string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE site_versions SET review_status = $3, review_reason = $4, reviewed_by = $5 WHERE site_id = $1 AND version = $2`,
		siteID, version, status, reason, by)
	return err
}

func (r *siteHostingRepository) ListVersions(ctx context.Context, siteID int64) ([]service.SiteVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT version, size_bytes, file_count, review_status, review_reason, reviewed_by, created_at
FROM site_versions WHERE site_id = $1 ORDER BY version DESC LIMIT 20`, siteID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SiteVersion
	for rows.Next() {
		var v service.SiteVersion
		if err := rows.Scan(&v.Version, &v.SizeBytes, &v.FileCount, &v.ReviewStatus, &v.ReviewReason, &v.ReviewedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *siteHostingRepository) ListPendingReviews(ctx context.Context, page, pageSize int) ([]service.SiteReviewItem, int64, error) {
	const from = ` FROM site_versions v JOIN sites s ON s.id = v.site_id AND s.pending_version = v.version LEFT JOIN users u ON u.id = s.user_id WHERE v.review_status = 'pending'`
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT s.id, s.name, s.title, COALESCE(u.email, ''), s.status, v.version, v.size_bytes, v.file_count, v.review_reason, v.review_flags, v.excerpt, v.created_at`+from+
		` ORDER BY v.created_at, v.id LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SiteReviewItem
	for rows.Next() {
		var it service.SiteReviewItem
		var flags []byte
		if err := rows.Scan(&it.SiteID, &it.SiteName, &it.Title, &it.OwnerEmail, &it.SiteStatus, &it.Version, &it.SizeBytes, &it.FileCount, &it.Reason, &flags, &it.Excerpt, &it.CreatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(flags, &it.Flags)
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *siteHostingRepository) SetSitePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sites SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, hash)
	return err
}

func (r *siteHostingRepository) AddDailyStats(ctx context.Context, stats []service.SiteDailyStat) error {
	for _, st := range stats {
		// The day is the local calendar day; pass it as text so the session time zone cannot shift it.
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO site_daily_stats (site_id, day, views, visitors, bytes) SELECT $1, $2::date, $3, $4, $5 WHERE EXISTS (SELECT 1 FROM sites WHERE id = $1)
ON CONFLICT (site_id, day) DO UPDATE SET views = site_daily_stats.views + EXCLUDED.views,
    visitors = site_daily_stats.visitors + EXCLUDED.visitors, bytes = site_daily_stats.bytes + EXCLUDED.bytes`,
			st.SiteID, st.Day.Format("2006-01-02"), st.Views, st.Visitors, st.Bytes); err != nil {
			return err
		}
	}
	return nil
}

func (r *siteHostingRepository) ListDailyStats(ctx context.Context, siteID int64, since time.Time) ([]service.SiteDailyStat, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT to_char(day, 'YYYY-MM-DD'), views, visitors, bytes FROM site_daily_stats WHERE site_id = $1 AND day >= $2::date ORDER BY day`, siteID, since.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SiteDailyStat
	for rows.Next() {
		var st service.SiteDailyStat
		var day string
		if err := rows.Scan(&day, &st.Views, &st.Visitors, &st.Bytes); err != nil {
			return nil, err
		}
		st.SiteID = siteID
		st.Day, _ = time.ParseInLocation("2006-01-02", day, since.Location())
		out = append(out, st)
	}
	return out, rows.Err()
}

func (r *siteHostingRepository) SetSiteTitle(ctx context.Context, id int64, title string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sites SET title = $2, updated_at = NOW() WHERE id = $1`, id, title)
	return err
}

func (r *siteHostingRepository) SetSiteStatus(ctx context.Context, id int64, status, reason string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sites SET status = $2, status_reason = $3, updated_at = NOW() WHERE id = $1`, id, status, reason)
	return err
}

func (r *siteHostingRepository) SetSiteFree(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sites SET paid = FALSE, paid_until = NULL, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *siteHostingRepository) SetLapsedAt(ctx context.Context, userID int64, at *time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sites SET lapsed_at = $2 WHERE user_id = $1`, userID, at)
	return err
}

func (r *siteHostingRepository) DeleteSite(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sites WHERE id = $1`, id)
	return err
}

func (r *siteHostingRepository) ChargeSite(ctx context.Context, site *service.Site, amount float64, periodEnd time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if amount > 0 {
		res, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, amount, site.UserID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return service.ErrInsufficientBalance
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO site_charges (site_id, site_name, user_id, amount, period_end) VALUES ($1, $2, $3, $4, $5)`,
			site.ID, site.Name, site.UserID, amount, periodEnd); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sites SET paid = TRUE, paid_until = $2, updated_at = NOW() WHERE id = $1`, site.ID, periodEnd); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *siteHostingRepository) ListCharges(ctx context.Context, userID int64, limit int) ([]service.SiteCharge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, site_id, site_name, amount, period_end, created_at FROM site_charges WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SiteCharge
	for rows.Next() {
		var c service.SiteCharge
		var siteID sql.NullInt64
		if err := rows.Scan(&c.ID, &siteID, &c.SiteName, &c.Amount, &c.PeriodEnd, &c.CreatedAt); err != nil {
			return nil, err
		}
		if siteID.Valid {
			c.SiteID = &siteID.Int64
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *siteHostingRepository) CreateReport(ctx context.Context, rep *service.SiteReport) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO site_reports (site_id, site_name, reason, detail, contact, reporter_ip) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, status, created_at`,
		rep.SiteID, rep.SiteName, rep.Reason, rep.Detail, rep.Contact, rep.ReporterIP).Scan(&rep.ID, &rep.Status, &rep.CreatedAt)
}

func (r *siteHostingRepository) CountReportsSince(ctx context.Context, ip string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_reports WHERE reporter_ip = $1 AND created_at >= $2`, ip, since).Scan(&n)
	return n, err
}

func (r *siteHostingRepository) ListReports(ctx context.Context, status string, page, pageSize int) ([]service.SiteReport, int64, error) {
	cond := ""
	args := []any{}
	if status != "" {
		cond = " WHERE r.status = $1"
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_reports r`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT r.id, r.site_id, r.site_name, COALESCE(s.status, ''), COALESCE(u.email, ''), r.reason, r.detail, r.contact, r.reporter_ip, r.status, r.handled_at, r.created_at
FROM site_reports r LEFT JOIN sites s ON s.id = r.site_id LEFT JOIN users u ON u.id = s.user_id`+cond+
		` ORDER BY r.created_at DESC, r.id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SiteReport
	for rows.Next() {
		var rep service.SiteReport
		var siteID sql.NullInt64
		var handled sql.NullTime
		if err := rows.Scan(&rep.ID, &siteID, &rep.SiteName, &rep.SiteStatus, &rep.OwnerEmail, &rep.Reason, &rep.Detail, &rep.Contact, &rep.ReporterIP, &rep.Status, &handled, &rep.CreatedAt); err != nil {
			return nil, 0, err
		}
		if siteID.Valid {
			rep.SiteID = &siteID.Int64
		}
		if handled.Valid {
			rep.HandledAt = &handled.Time
		}
		out = append(out, rep)
	}
	return out, total, rows.Err()
}

func (r *siteHostingRepository) SetReportStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE site_reports SET status = $2::varchar, handled_at = CASE WHEN $2::varchar = 'open' THEN NULL ELSE NOW() END WHERE id = $1`, id, status)
	return err
}

func (r *siteHostingRepository) Balance(ctx context.Context, userID int64) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return balance, err
}
