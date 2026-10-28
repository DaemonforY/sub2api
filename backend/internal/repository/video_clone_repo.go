package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// --- 我的音色 (video_voice_clones) ---------------------------------------------------------------

const videoCloneColumns = `id, user_id, speaker_id, name, status, error, trainings, price, charged, consent_ip, consent_at,
checked_at, created_at, updated_at`

func scanVideoClone(row videoScanner) (*service.VideoVoiceClone, error) {
	var c service.VideoVoiceClone
	var checked sql.NullTime
	err := row.Scan(&c.ID, &c.UserID, &c.SpeakerID, &c.Name, &c.Status, &c.Error, &c.Trainings, &c.Price, &c.Charged, &c.ConsentIP,
		&c.ConsentAt, &checked, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if checked.Valid {
		c.CheckedAt = &checked.Time
	}
	return &c, nil
}

// ClaimClone takes the free slot that has been trained least (豆包 limits trainings per slot), so
// slots wear evenly; a concurrent claim of the same slot hits the unique index and tries the next.
func (r *videoRepository) ClaimClone(ctx context.Context, c *service.VideoVoiceClone, slots []string, consentText string) error {
	for attempt := 0; attempt < 5; attempt++ {
		var speaker string
		err := r.db.QueryRowContext(ctx, `
SELECT t.s FROM unnest($1::text[]) WITH ORDINALITY AS t(s, n)
WHERE NOT EXISTS (SELECT 1 FROM video_voice_clones v WHERE v.speaker_id = t.s AND v.deleted_at IS NULL)
ORDER BY (SELECT COALESCE(SUM(v.trainings), 0) FROM video_voice_clones v WHERE v.speaker_id = t.s), t.n
LIMIT 1`, pq.Array(slots)).Scan(&speaker)
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrVideoCloneFull
		}
		if err != nil {
			return err
		}
		err = r.db.QueryRowContext(ctx, `
INSERT INTO video_voice_clones (user_id, speaker_id, name, status, trainings, price, consent_text, consent_ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, consent_at, created_at, updated_at`,
			c.UserID, speaker, c.Name, c.Status, c.Trainings, c.Price, consentText, c.ConsentIP).Scan(&c.ID, &c.ConsentAt, &c.CreatedAt, &c.UpdatedAt)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			continue
		}
		if err != nil {
			return err
		}
		c.SpeakerID = speaker
		return nil
	}
	return service.ErrVideoCloneFull
}

func (r *videoRepository) ListClones(ctx context.Context, userID int64) ([]service.VideoVoiceClone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+videoCloneColumns+` FROM video_voice_clones
WHERE user_id = $1 AND deleted_at IS NULL ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoVoiceClone{}
	for rows.Next() {
		c, err := scanVideoClone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *videoRepository) GetClone(ctx context.Context, id int64) (*service.VideoVoiceClone, error) {
	c, err := scanVideoClone(r.db.QueryRowContext(ctx, `SELECT `+videoCloneColumns+` FROM video_voice_clones
WHERE id = $1 AND deleted_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoCloneNotFound
	}
	return c, err
}

// UpdateClone writes name, status, error, trainings and checked_at (not over an admin's disable).
func (r *videoRepository) UpdateClone(ctx context.Context, c *service.VideoVoiceClone) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE video_voice_clones SET name = $2,
  status = CASE WHEN status = 'disabled' THEN status ELSE $3 END,
  error = CASE WHEN status = 'disabled' THEN error ELSE $4 END,
  trainings = $5, checked_at = $6, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, c.ID, c.Name, c.Status, c.Error, c.Trainings, c.CheckedAt)
	return err
}

func (r *videoRepository) DeleteClone(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE video_voice_clones SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// MarkCloneReady sets a training clone ready; the first time, its price leaves the balance and is
// recorded in video_tts_charges (project "voice-clone"), all in one transaction.
func (r *videoRepository) MarkCloneReady(ctx context.Context, id int64) (float64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var userID int64
	var speaker string
	var price float64
	var charged bool
	err = tx.QueryRowContext(ctx, `
UPDATE video_voice_clones SET status = 'ready', error = '', checked_at = NOW(), updated_at = NOW(), charged = TRUE
WHERE id = $1 AND deleted_at IS NULL AND status = 'training'
RETURNING user_id, speaker_id, price, (SELECT charged FROM video_voice_clones WHERE id = $1)`, id).Scan(&userID, &speaker, &price, &charged)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil // not training any more (deleted, disabled or already ready)
	}
	if err != nil {
		return 0, err
	}
	if charged || price <= 0 {
		return 0, tx.Commit()
	}
	res, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, price, userID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, service.ErrUserNotFound
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO video_tts_charges (user_id, project_id, scene_id, voice, chars, cost) VALUES ($1, 'voice-clone', $2, $3, 0, $4)`,
		userID, "clone-"+strconv.FormatInt(id, 10), speaker, price); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return price, nil
}

func (r *videoRepository) AdminListClones(ctx context.Context, limit int) ([]service.VideoAdminClone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+prefixColumns("c", videoCloneColumns)+`, COALESCE(u.email, '')
FROM video_voice_clones c LEFT JOIN users u ON u.id = c.user_id
WHERE c.deleted_at IS NULL ORDER BY c.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.VideoAdminClone{}
	for rows.Next() {
		var a service.VideoAdminClone
		var checked sql.NullTime
		c := &a.VideoVoiceClone
		if err := rows.Scan(&c.ID, &c.UserID, &c.SpeakerID, &c.Name, &c.Status, &c.Error, &c.Trainings, &c.Price, &c.Charged, &c.ConsentIP,
			&c.ConsentAt, &checked, &c.CreatedAt, &c.UpdatedAt, &a.UserEmail); err != nil {
			return nil, err
		}
		a.SpeakerID, a.ConsentIP = c.SpeakerID, c.ConsentIP
		out = append(out, a)
	}
	return out, rows.Err()
}

// SetCloneStatus is an admin disabling (or restoring) a clone.
func (r *videoRepository) SetCloneStatus(ctx context.Context, id int64, status, message string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE video_voice_clones SET status = $2, error = $3, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, id, status, message)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrVideoCloneNotFound
	}
	return nil
}
