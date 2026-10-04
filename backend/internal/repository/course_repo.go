package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type courseRepository struct {
	db *sql.DB
}

// NewCourseRepository stores paid courses, their deliveries and enrollments (raw SQL).
func NewCourseRepository(db *sql.DB) service.CourseRepository {
	return &courseRepository{db: db}
}

// activeEnrollment: not revoked and its order (if any) not fully refunded.
const activeEnrollment = `e.status = 'active' AND NOT EXISTS (
    SELECT 1 FROM payment_orders po WHERE po.id = e.order_id AND po.status = 'REFUNDED')`

const courseSelect = `
SELECT c.id, c.slug, c.title, c.subtitle, c.category, c.cover_file, c.price::float8, c.original_price::float8,
       c.intro_md, c.outline, c.trial_md, c.faq_md, c.status, c.sort_order, c.created_at, c.updated_at,
       (SELECT COUNT(*) FROM course_enrollments e WHERE e.course_id = c.id AND ` + activeEnrollment + `) AS student_count,
       COALESCE((SELECT SUM(po.pay_amount)::float8 FROM payment_orders po
                 WHERE po.order_type = 'course' AND po.course_id = c.id AND po.status = 'COMPLETED'), 0) AS revenue,
       COALESCE((SELECT MAX(d.version) FROM course_deliveries d WHERE d.course_id = c.id), 0) AS delivery_version
FROM courses c`

func scanCourse(row rowScanner) (*service.Course, error) {
	var c service.Course
	var outline []byte
	if err := row.Scan(&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Category, &c.CoverFile, &c.Price, &c.OriginalPrice,
		&c.IntroMD, &outline, &c.TrialMD, &c.FaqMD, &c.Status, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
		&c.StudentCount, &c.Revenue, &c.DeliveryVersion); err != nil {
		return nil, err
	}
	c.Outline = service.UnmarshalOutline(outline)
	return &c, nil
}

func (r *courseRepository) ListCourses(ctx context.Context, statuses []string) ([]service.Course, error) {
	query, args := courseSelect, []any{}
	if len(statuses) > 0 {
		query += ` WHERE c.status = ANY($1)`
		args = append(args, pq.Array(statuses))
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY c.sort_order DESC, c.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *courseRepository) courseWhere(ctx context.Context, where string, arg any) (*service.Course, error) {
	c, err := scanCourse(r.db.QueryRowContext(ctx, courseSelect+` WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *courseRepository) GetCourse(ctx context.Context, id int64) (*service.Course, error) {
	return r.courseWhere(ctx, "c.id = $1", id)
}

func (r *courseRepository) GetCourseBySlug(ctx context.Context, slug string) (*service.Course, error) {
	return r.courseWhere(ctx, "c.slug = $1", slug)
}

func (r *courseRepository) CreateCourse(ctx context.Context, c *service.Course) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO courses (slug, title, subtitle, category, cover_file, price, original_price, intro_md, outline, trial_md, faq_md, status, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id, created_at, updated_at`,
		c.Slug, c.Title, c.Subtitle, c.Category, c.CoverFile, c.Price, c.OriginalPrice, c.IntroMD, service.MarshalOutline(c.Outline),
		c.TrialMD, c.FaqMD, c.Status, c.SortOrder).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if isUniqueViolation(err) {
		return service.ErrCourseSlugTaken
	}
	return err
}

func (r *courseRepository) UpdateCourse(ctx context.Context, c *service.Course) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE courses SET slug = $2, title = $3, subtitle = $4, category = $5, cover_file = $6, price = $7, original_price = $8,
       intro_md = $9, outline = $10, trial_md = $11, faq_md = $12, status = $13, sort_order = $14, updated_at = NOW()
WHERE id = $1`,
		c.ID, c.Slug, c.Title, c.Subtitle, c.Category, c.CoverFile, c.Price, c.OriginalPrice, c.IntroMD, service.MarshalOutline(c.Outline),
		c.TrialMD, c.FaqMD, c.Status, c.SortOrder)
	if isUniqueViolation(err) {
		return service.ErrCourseSlugTaken
	}
	return err
}

func (r *courseRepository) DeleteCourse(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM courses WHERE id = $1`, id)
	return err
}

// Enrollments ------------------------------------------------------------------------------------

const enrollmentColumns = `e.user_id, e.course_id, COALESCE(e.order_id, 0), e.source, e.status, e.note, e.first_viewed_at, e.last_viewed_at,
       e.view_count, e.seen_version, EXISTS (SELECT 1 FROM payment_orders po WHERE po.id = e.order_id AND po.status = 'REFUNDED'), e.created_at`

func scanEnrollment(row rowScanner, extra ...any) (*service.CourseEnrollment, error) {
	var e service.CourseEnrollment
	var first, last sql.NullTime
	dest := []any{&e.UserID, &e.CourseID, &e.OrderID, &e.Source, &e.Status, &e.Note, &first, &last, &e.ViewCount, &e.SeenVersion, &e.Refunded, &e.CreatedAt}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return nil, err
	}
	if first.Valid {
		e.FirstViewedAt = &first.Time
	}
	if last.Valid {
		e.LastViewedAt = &last.Time
	}
	return &e, nil
}

func (r *courseRepository) GetEnrollment(ctx context.Context, userID, courseID int64) (*service.CourseEnrollment, error) {
	e, err := scanEnrollment(r.db.QueryRowContext(ctx, `SELECT `+enrollmentColumns+` FROM course_enrollments e WHERE e.user_id = $1 AND e.course_id = $2`, userID, courseID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

func (r *courseRepository) OwnedCourses(ctx context.Context, userID int64) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT e.course_id FROM course_enrollments e WHERE e.user_id = $1 AND `+activeEnrollment, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// UpsertEnrollment opens a course; buying again after a refund or a revoke renews the row.
func (r *courseRepository) UpsertEnrollment(ctx context.Context, e *service.CourseEnrollment) error {
	var orderID any
	if e.OrderID > 0 {
		orderID = e.OrderID
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO course_enrollments (user_id, course_id, order_id, source, status, note) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, course_id) DO UPDATE SET order_id = EXCLUDED.order_id, source = EXCLUDED.source, status = EXCLUDED.status,
    note = EXCLUDED.note, updated_at = NOW()`,
		e.UserID, e.CourseID, orderID, e.Source, e.Status, e.Note)
	return err
}

func (r *courseRepository) SetEnrollmentStatus(ctx context.Context, userID, courseID int64, status string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE course_enrollments SET status = $3, updated_at = NOW() WHERE user_id = $1 AND course_id = $2`, userID, courseID, status)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrCourseNotEnrolled
	}
	return nil
}

func (r *courseRepository) ListMyCourses(ctx context.Context, userID int64) ([]service.MyCourse, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT c.*, `+enrollmentColumns+`
FROM course_enrollments e
JOIN LATERAL (`+courseSelect+` WHERE c.id = e.course_id) c ON TRUE
WHERE e.user_id = $1 AND e.status = 'active'
ORDER BY e.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.MyCourse
	for rows.Next() {
		var c service.Course
		var outline []byte
		var e service.CourseEnrollment
		var first, last sql.NullTime
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Category, &c.CoverFile, &c.Price, &c.OriginalPrice,
			&c.IntroMD, &outline, &c.TrialMD, &c.FaqMD, &c.Status, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
			&c.StudentCount, &c.Revenue, &c.DeliveryVersion,
			&e.UserID, &e.CourseID, &e.OrderID, &e.Source, &e.Status, &e.Note, &first, &last, &e.ViewCount, &e.SeenVersion, &e.Refunded, &e.CreatedAt); err != nil {
			return nil, err
		}
		c.Outline = service.UnmarshalOutline(outline)
		if first.Valid {
			e.FirstViewedAt = &first.Time
		}
		if last.Valid {
			e.LastViewedAt = &last.Time
		}
		out = append(out, service.MyCourse{Course: c, Enrollment: e})
	}
	return out, rows.Err()
}

func (r *courseRepository) ListEnrollments(ctx context.Context, courseID int64, limit, offset int) ([]service.CourseEnrollment, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM course_enrollments WHERE course_id = $1`, courseID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT `+enrollmentColumns+`, COALESCE(u.email, ''), COALESCE(u.username, ''),
       (SELECT COUNT(DISTINCT l.ip) FROM course_access_logs l
        WHERE l.course_id = e.course_id AND l.user_id = e.user_id AND l.created_at > NOW() - INTERVAL '7 days')
FROM course_enrollments e
LEFT JOIN users u ON u.id = e.user_id
WHERE e.course_id = $1
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3`, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CourseEnrollment
	for rows.Next() {
		var email, username string
		var ips int
		e, err := scanEnrollment(rows, &email, &username, &ips)
		if err != nil {
			return nil, 0, err
		}
		e.UserEmail, e.Username, e.RecentIPs = email, username, ips
		out = append(out, *e)
	}
	return out, total, rows.Err()
}

// Deliveries -------------------------------------------------------------------------------------

const deliveryColumns = `id, course_id, version, link_enc, code_enc, password_enc, note, COALESCE(created_by, 0), created_at`

func scanDelivery(row rowScanner) (*service.CourseDeliveryRecord, error) {
	var d service.CourseDeliveryRecord
	if err := row.Scan(&d.ID, &d.CourseID, &d.Version, &d.LinkEnc, &d.CodeEnc, &d.PasswordEnc, &d.Note, &d.CreatedBy, &d.CreatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *courseRepository) LatestDelivery(ctx context.Context, courseID int64) (*service.CourseDeliveryRecord, error) {
	d, err := scanDelivery(r.db.QueryRowContext(ctx, `SELECT `+deliveryColumns+` FROM course_deliveries WHERE course_id = $1 ORDER BY version DESC LIMIT 1`, courseID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

func (r *courseRepository) ListDeliveries(ctx context.Context, courseID int64) ([]service.CourseDeliveryRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deliveryColumns+` FROM course_deliveries WHERE course_id = $1 ORDER BY version DESC LIMIT 50`, courseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CourseDeliveryRecord
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// AddDelivery stores the next version (versions of one course are serialised by the course row lock).
func (r *courseRepository) AddDelivery(ctx context.Context, d *service.CourseDeliveryRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM courses WHERE id = $1 FOR UPDATE`, d.CourseID); err != nil {
		return err
	}
	var createdBy any
	if d.CreatedBy > 0 {
		createdBy = d.CreatedBy
	}
	if err := tx.QueryRowContext(ctx, `
INSERT INTO course_deliveries (course_id, version, link_enc, code_enc, password_enc, note, created_by)
VALUES ($1, COALESCE((SELECT MAX(version) FROM course_deliveries WHERE course_id = $1), 0) + 1, $2, $3, $4, $5, $6)
RETURNING id, version, created_at`,
		d.CourseID, d.LinkEnc, d.CodeEnc, d.PasswordEnc, d.Note, createdBy).Scan(&d.ID, &d.Version, &d.CreatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *courseRepository) CountAccess(ctx context.Context, courseID, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM course_access_logs WHERE course_id = $1 AND user_id = $2 AND created_at > $3`, courseID, userID, since).Scan(&n)
	return n, err
}

// RecordAccess logs a view and marks the version the buyer has seen.
func (r *courseRepository) RecordAccess(ctx context.Context, courseID, userID int64, ip string, version int) error {
	if _, err := r.db.ExecContext(ctx, `INSERT INTO course_access_logs (course_id, user_id, ip) VALUES ($1, $2, $3)`, courseID, userID, ip); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE course_enrollments SET view_count = view_count + 1, first_viewed_at = COALESCE(first_viewed_at, NOW()), last_viewed_at = NOW(),
       seen_version = GREATEST(seen_version, $3)
WHERE user_id = $1 AND course_id = $2`, userID, courseID, version)
	return err
}

// FindUserID resolves a user ID or a sign-up email (0 when there is none).
func (r *courseRepository) FindUserID(ctx context.Context, idOrEmail string) (int64, error) {
	if idOrEmail == "" {
		return 0, nil
	}
	var id int64
	var err error
	if n, convErr := strconv.ParseInt(idOrEmail, 10, 64); convErr == nil {
		err = r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL`, n).Scan(&id)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE LOWER(email) = $1 AND deleted_at IS NULL`, strings.ToLower(idOrEmail)).Scan(&id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}
