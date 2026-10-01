package repository

import (
	"context"
	"database/sql"
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
       s.paid, s.paid_until, s.lapsed_at, s.created_at, s.updated_at
FROM sites s LEFT JOIN users u ON u.id = s.user_id`

func scanSite(row rowScanner) (*service.Site, error) {
	var site service.Site
	var paidUntil, lapsedAt sql.NullTime
	if err := row.Scan(&site.ID, &site.UserID, &site.UserEmail, &site.Name, &site.Title, &site.Status, &site.StatusReason, &site.Version,
		&site.SizeBytes, &site.FileCount, &site.Paid, &paidUntil, &lapsedAt, &site.CreatedAt, &site.UpdatedAt); err != nil {
		return nil, err
	}
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

func (r *siteHostingRepository) SetSiteVersion(ctx context.Context, id int64, version int, size int64, files int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE sites SET version = $2, size_bytes = $3, file_count = $4, updated_at = NOW() WHERE id = $1`, id, version, size, files); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO site_versions (site_id, version, size_bytes, file_count) VALUES ($1, $2, $3, $4)
ON CONFLICT (site_id, version) DO UPDATE SET size_bytes = EXCLUDED.size_bytes, file_count = EXCLUDED.file_count, created_at = NOW()`, id, version, size, files); err != nil {
		return err
	}
	return tx.Commit()
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
