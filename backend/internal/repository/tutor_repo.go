package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type tutorRepository struct {
	db *sql.DB
}

// NewTutorRepository creates the AI 助教 store (raw SQL: tutors, tutor_materials, tutor_students, tutor_messages).
func NewTutorRepository(db *sql.DB) service.TutorRepository {
	return &tutorRepository{db: db}
}

const tutorColumns = `id, user_id, api_key_id, name, template, subject, grade, style, answer_mode, model_tier, task, suggestions, rules, greeting, share_code,
pass_code, per_student_day, daily_cap, enabled, created_at, updated_at`

func scanTutor(row rowScanner, extra ...any) (*service.Tutor, error) {
	var t service.Tutor
	var suggestions []byte
	dest := []any{&t.ID, &t.UserID, &t.KeyID, &t.Name, &t.Template, &t.Subject, &t.Grade, &t.Style, &t.AnswerMode, &t.ModelTier, &t.Task, &suggestions, &t.Rules,
		&t.Greeting, &t.ShareCode, &t.PassCode, &t.PerStudentDay, &t.DailyCap, &t.Enabled, &t.CreatedAt, &t.UpdatedAt}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if json.Unmarshal(suggestions, &t.Suggestions) != nil || t.Suggestions == nil {
		t.Suggestions = []string{}
	}
	return &t, nil
}

func tutorSuggestionsJSON(t *service.Tutor) string {
	if t.Suggestions == nil {
		return "[]"
	}
	b, _ := json.Marshal(t.Suggestions)
	return string(b)
}

func (r *tutorRepository) Create(ctx context.Context, t *service.Tutor) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO tutors (user_id, api_key_id, name, template, subject, grade, style, answer_mode, rules, greeting, share_code, pass_code, per_student_day, daily_cap, enabled, model_tier, task, suggestions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18::jsonb)
RETURNING id, created_at, updated_at`,
		t.UserID, t.KeyID, t.Name, t.Template, t.Subject, t.Grade, t.Style, t.AnswerMode, t.Rules, t.Greeting, t.ShareCode, t.PassCode,
		t.PerStudentDay, t.DailyCap, t.Enabled, t.ModelTier, t.Task, tutorSuggestionsJSON(t)).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *tutorRepository) Update(ctx context.Context, t *service.Tutor) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE tutors SET api_key_id = $3, name = $4, template = $5, subject = $6, grade = $7, style = $8, answer_mode = $9, rules = $10,
       greeting = $11, pass_code = $12, per_student_day = $13, daily_cap = $14, enabled = $15, model_tier = $16, task = $17, suggestions = $18::jsonb, updated_at = NOW()
WHERE id = $1 AND user_id = $2`,
		t.ID, t.UserID, t.KeyID, t.Name, t.Template, t.Subject, t.Grade, t.Style, t.AnswerMode, t.Rules, t.Greeting, t.PassCode,
		t.PerStudentDay, t.DailyCap, t.Enabled, t.ModelTier, t.Task, tutorSuggestionsJSON(t))
	return err
}

func (r *tutorRepository) Get(ctx context.Context, userID, id int64) (*service.Tutor, error) {
	return scanTutor(r.db.QueryRowContext(ctx, `SELECT `+tutorColumns+` FROM tutors WHERE id = $1 AND user_id = $2`, id, userID))
}

func (r *tutorRepository) GetByCode(ctx context.Context, code string) (*service.Tutor, error) {
	return scanTutor(r.db.QueryRowContext(ctx, `SELECT `+tutorColumns+` FROM tutors WHERE share_code = $1`, code))
}

func (r *tutorRepository) List(ctx context.Context, userID int64) ([]service.Tutor, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+tutorColumns+`, COALESCE((SELECT SUM(chars) FROM tutor_materials m WHERE m.tutor_id = tutors.id), 0)
FROM tutors WHERE user_id = $1 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.Tutor{}
	for rows.Next() {
		var chars int
		t, err := scanTutor(rows, &chars)
		if err != nil {
			return nil, err
		}
		t.MaterialChars = chars
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *tutorRepository) Count(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tutors WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *tutorRepository) Delete(ctx context.Context, userID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tutors WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *tutorRepository) AddMaterial(ctx context.Context, tutorID int64, name, content string) (*service.TutorMaterial, error) {
	m := service.TutorMaterial{Name: name, Content: content}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO tutor_materials (tutor_id, name, chars, content) VALUES ($1, $2, char_length($3), $3)
RETURNING id, chars, created_at`, tutorID, name, content).Scan(&m.ID, &m.Chars, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *tutorRepository) ListMaterials(ctx context.Context, tutorID int64, withContent bool) ([]service.TutorMaterial, error) {
	content := `''`
	if withContent {
		content = `content`
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, chars, `+content+`, created_at FROM tutor_materials WHERE tutor_id = $1 ORDER BY id`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.TutorMaterial{}
	for rows.Next() {
		var m service.TutorMaterial
		if err := rows.Scan(&m.ID, &m.Name, &m.Chars, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *tutorRepository) DeleteMaterial(ctx context.Context, tutorID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tutor_materials WHERE id = $1 AND tutor_id = $2`, id, tutorID)
	return err
}

const tutorStudentColumns = `id, tutor_id, name, questions, created_at, last_seen_at`

func scanTutorStudent(row rowScanner) (*service.TutorStudent, error) {
	var s service.TutorStudent
	if err := row.Scan(&s.ID, &s.TutorID, &s.Name, &s.Questions, &s.CreatedAt, &s.LastSeenAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *tutorRepository) UpsertStudent(ctx context.Context, tutorID int64, name, tokenHash string) (*service.TutorStudent, error) {
	return scanTutorStudent(r.db.QueryRowContext(ctx, `
INSERT INTO tutor_students (tutor_id, name, token_hash) VALUES ($1, $2, $3)
ON CONFLICT (tutor_id, name) DO UPDATE SET token_hash = EXCLUDED.token_hash, last_seen_at = NOW()
RETURNING `+tutorStudentColumns, tutorID, name, tokenHash))
}

func (r *tutorRepository) StudentByToken(ctx context.Context, tutorID int64, tokenHash string) (*service.TutorStudent, error) {
	return scanTutorStudent(r.db.QueryRowContext(ctx, `SELECT `+tutorStudentColumns+` FROM tutor_students WHERE tutor_id = $1 AND token_hash = $2`, tutorID, tokenHash))
}

func (r *tutorRepository) ListStudents(ctx context.Context, tutorID int64) ([]service.TutorStudent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+tutorStudentColumns+` FROM tutor_students WHERE tutor_id = $1 ORDER BY last_seen_at DESC, id DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.TutorStudent{}
	for rows.Next() {
		s, err := scanTutorStudent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *tutorRepository) AddMessage(ctx context.Context, tutorID, studentID int64, question, answer string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO tutor_messages (tutor_id, student_id, question, answer) VALUES ($1, $2, $3, $4)`,
		tutorID, studentID, question, answer); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tutor_students SET questions = questions + 1, last_seen_at = NOW() WHERE id = $1 AND tutor_id = $2`,
		studentID, tutorID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *tutorRepository) ListMessages(ctx context.Context, tutorID int64, since time.Time, limit int) ([]service.TutorMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT m.id, m.student_id, s.name, m.question, m.answer, m.created_at
FROM tutor_messages m JOIN tutor_students s ON s.id = m.student_id
WHERE m.tutor_id = $1 AND m.created_at >= $2
ORDER BY m.created_at DESC, m.id DESC LIMIT $3`, tutorID, since, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.TutorMessage{}
	for rows.Next() {
		var m service.TutorMessage
		if err := rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Question, &m.Answer, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *tutorRepository) CountMessages(ctx context.Context, tutorID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tutor_messages WHERE tutor_id = $1 AND created_at >= $2`, tutorID, since).Scan(&n)
	return n, err
}

func (r *tutorRepository) DeleteMessagesBefore(ctx context.Context, before time.Time) (int64, error) {
	out, err := r.db.ExecContext(ctx, `DELETE FROM tutor_messages WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}
