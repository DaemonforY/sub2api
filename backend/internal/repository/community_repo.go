package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type communityRepository struct {
	db *sql.DB
}

// NewCommunityRepository stores the canvas community (raw SQL).
func NewCommunityRepository(db *sql.DB) service.CommunityRepository {
	return &communityRepository{db: db}
}

// Profiles ---------------------------------------------------------------------------------------

const profileColumns = `p.user_id, p.handle, p.display_name, p.avatar_file, p.bio, p.status, p.works_count, p.followers_count,
       p.following_count, p.likes_received, p.created_at`

func scanProfile(row rowScanner) (*service.CommunityProfile, error) {
	var p service.CommunityProfile
	if err := row.Scan(&p.UserID, &p.Handle, &p.DisplayName, &p.AvatarFile, &p.Bio, &p.Status, &p.WorksCount, &p.FollowersCount,
		&p.FollowingCount, &p.LikesReceived, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *communityRepository) profileWhere(ctx context.Context, where string, arg any) (*service.CommunityProfile, error) {
	p, err := scanProfile(r.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM user_profiles p WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *communityRepository) GetProfileByUser(ctx context.Context, userID int64) (*service.CommunityProfile, error) {
	return r.profileWhere(ctx, "p.user_id = $1", userID)
}

func (r *communityRepository) GetProfileByHandle(ctx context.Context, handle string) (*service.CommunityProfile, error) {
	return r.profileWhere(ctx, "LOWER(p.handle) = LOWER($1)", handle)
}

func (r *communityRepository) CreateProfile(ctx context.Context, p *service.CommunityProfile) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_profiles (user_id, handle, display_name, avatar_file, bio, status) VALUES ($1, $2, $3, $4, $5, $6)`,
		p.UserID, p.Handle, p.DisplayName, p.AvatarFile, p.Bio, p.Status)
	if isUniqueViolation(err) {
		return service.ErrCommunityHandleTaken
	}
	if err != nil {
		return err
	}
	return r.RecountProfile(ctx, p.UserID)
}

func (r *communityRepository) UpdateProfile(ctx context.Context, p *service.CommunityProfile) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE user_profiles SET handle = $2, display_name = $3, avatar_file = $4, bio = $5, updated_at = NOW() WHERE user_id = $1`,
		p.UserID, p.Handle, p.DisplayName, p.AvatarFile, p.Bio)
	if isUniqueViolation(err) {
		return service.ErrCommunityHandleTaken
	}
	return err
}

func (r *communityRepository) SetProfileStatus(ctx context.Context, userID int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_profiles SET status = $2, updated_at = NOW() WHERE user_id = $1`, userID, status)
	return err
}

func (r *communityRepository) ListRestrictedProfiles(ctx context.Context, limit int) ([]service.RestrictedAuthor, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT p.user_id, COALESCE(u.email, ''), p.handle, p.display_name,
       (SELECT COUNT(*) FROM works w WHERE w.user_id = p.user_id), p.updated_at
FROM user_profiles p LEFT JOIN users u ON u.id = p.user_id
WHERE p.status = 'banned' ORDER BY p.updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.RestrictedAuthor
	for rows.Next() {
		var a service.RestrictedAuthor
		if err := rows.Scan(&a.UserID, &a.Email, &a.Handle, &a.DisplayName, &a.WorksCount, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *communityRepository) UserCreatedAt(ctx context.Context, userID int64) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRowContext(ctx, `SELECT created_at FROM users WHERE id = $1`, userID).Scan(&at)
	return at, err
}

func (r *communityRepository) AccountAvatarURL(ctx context.Context, userID int64) (string, error) {
	var url string
	err := r.db.QueryRowContext(ctx, `SELECT url FROM user_avatars WHERE user_id = $1`, userID).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return url, err
}

func (r *communityRepository) RecountProfile(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE user_profiles SET
    works_count = (SELECT COUNT(*) FROM works WHERE user_id = $1 AND visibility = 'public' AND status = 'approved'),
    likes_received = (SELECT COALESCE(SUM(like_count), 0) FROM works WHERE user_id = $1),
    followers_count = (SELECT COUNT(*) FROM follows WHERE followee_id = $1),
    following_count = (SELECT COUNT(*) FROM follows WHERE follower_id = $1)
WHERE user_id = $1`, userID)
	return err
}

// Works ------------------------------------------------------------------------------------------

const workSelect = `
SELECT w.id, w.user_id, COALESCE(p.handle, ''), COALESCE(p.display_name, ''), COALESCE(p.avatar_file, ''),
       w.title, w.description, w.prompt, w.show_prompt, w.model, w.params, w.source, w.tags, w.visibility, w.status,
       w.review_reason, w.review_flags, w.featured_at IS NOT NULL, COALESCE(m.file, ''), COALESCE(m.thumb_file, ''),
       w.cover_width, w.cover_height, (SELECT COUNT(*) FROM work_media x WHERE x.work_id = w.id),
       w.like_count, w.favorite_count, w.remix_count, w.view_count, w.report_count, w.created_at, w.updated_at
FROM works w
LEFT JOIN user_profiles p ON p.user_id = w.user_id
LEFT JOIN work_media m ON m.work_id = w.id AND m.position = 0`

func scanWork(row rowScanner) (*service.Work, error) {
	var w service.Work
	var params []byte
	if err := row.Scan(&w.ID, &w.UserID, &w.Author.Handle, &w.Author.DisplayName, &w.Author.AvatarURL,
		&w.Title, &w.Description, &w.Prompt, &w.ShowPrompt, &w.Model, &params, &w.Source, pq.Array(&w.Tags), &w.Visibility, &w.Status,
		&w.ReviewReason, pq.Array(&w.ReviewFlags), &w.Featured, &w.CoverFile, &w.CoverThumb,
		&w.CoverWidth, &w.CoverHeight, &w.ImageCount,
		&w.LikeCount, &w.FavoriteCount, &w.RemixCount, &w.ViewCount, &w.ReportCount, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	w.Author.UserID = w.UserID
	w.Params = json.RawMessage(params)
	return &w, nil
}

func (r *communityRepository) CreateWork(ctx context.Context, w *service.Work, media []service.WorkMedia) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	params := []byte(w.Params)
	if len(params) == 0 {
		params = []byte(`{}`)
	}
	if err := tx.QueryRowContext(ctx, `
INSERT INTO works (user_id, title, description, prompt, show_prompt, model, params, source, tags, visibility, status,
                   review_reason, review_flags, cover_width, cover_height)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id, created_at, updated_at`,
		w.UserID, w.Title, w.Description, w.Prompt, w.ShowPrompt, w.Model, params, w.Source, pq.Array(w.Tags), w.Visibility, w.Status,
		w.ReviewReason, pq.Array(w.ReviewFlags), w.CoverWidth, w.CoverHeight).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return err
	}
	for _, m := range media {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO work_media (work_id, position, file, thumb_file, mime_type, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			w.ID, m.Position, m.File, m.ThumbFile, m.MimeType, m.Width, m.Height, m.SizeBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *communityRepository) GetWork(ctx context.Context, id int64) (*service.Work, error) {
	w, err := scanWork(r.db.QueryRowContext(ctx, workSelect+` WHERE w.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT position, file, thumb_file, mime_type, width, height, size_bytes FROM work_media WHERE work_id = $1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m service.WorkMedia
		if err := rows.Scan(&m.Position, &m.File, &m.ThumbFile, &m.MimeType, &m.Width, &m.Height, &m.SizeBytes); err != nil {
			return nil, err
		}
		w.Media = append(w.Media, m)
	}
	return w, rows.Err()
}

func (r *communityRepository) UpdateWork(ctx context.Context, w *service.Work) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE works SET title = $2, description = $3, show_prompt = $4, tags = $5, visibility = $6, status = $7,
       review_reason = $8, review_flags = $9, updated_at = NOW()
WHERE id = $1`, w.ID, w.Title, w.Description, w.ShowPrompt, pq.Array(w.Tags), w.Visibility, w.Status, w.ReviewReason, pq.Array(w.ReviewFlags))
	return err
}

func (r *communityRepository) DeleteWork(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
UPDATE collections SET works_count = GREATEST(works_count - 1, 0), updated_at = NOW()
WHERE id IN (SELECT collection_id FROM collection_items WHERE work_id = $1)`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM works WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

const publicWork = `w.visibility = 'public' AND w.status = 'approved' AND COALESCE(p.status, 'active') = 'active'`

func (r *communityRepository) ListWorks(ctx context.Context, q service.WorkQuery) ([]service.Work, error) {
	var (
		join  string
		where []string
		args  []any
		order = "w.created_at DESC, w.id DESC"
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	visible := publicWork
	switch q.Feed {
	case "recommended":
		where = append(where, publicWork)
		// Likes, favorites and remixes (and an editor's pick) count; age decays the score.
		order = `(w.like_count + 2 * w.favorite_count + 3 * w.remix_count + CASE WHEN w.featured_at IS NOT NULL THEN 20 ELSE 0 END + 1)
                 / POWER(EXTRACT(EPOCH FROM (NOW() - w.created_at)) / 3600 + 2, 1.5) DESC, w.id DESC`
	case "latest":
		where = append(where, publicWork)
	case "following":
		where = append(where, publicWork, "w.user_id IN (SELECT followee_id FROM follows WHERE follower_id = "+arg(q.ViewerID)+")")
	case "user":
		where = append(where, "w.user_id = "+arg(q.UserID))
		if !q.IncludeAll {
			where = append(where, visible)
		}
	case "favorites":
		join = " JOIN work_favorites f ON f.work_id = w.id AND f.user_id = " + arg(q.ViewerID)
		where = append(where, "("+visible+" OR w.user_id = "+arg(q.ViewerID)+")")
		order = "f.created_at DESC"
	case "collection":
		join = " JOIN collection_items ci ON ci.work_id = w.id AND ci.collection_id = " + arg(q.CollectionID)
		if !q.IncludeAll {
			where = append(where, `w.visibility <> 'private' AND w.status = 'approved' AND COALESCE(p.status, 'active') = 'active'`)
		}
		order = "ci.position, ci.added_at DESC"
	case "admin":
		switch q.Status {
		case "reported":
			where = append(where, "w.report_count > 0", "w.status IN ('approved', 'pending')")
			order = "w.report_count DESC, w.created_at DESC"
		case "pending", "approved", "hidden", "rejected":
			where = append(where, "w.status = "+arg(q.Status))
		}
	default:
		return nil, fmt.Errorf("unknown feed %q", q.Feed)
	}
	if q.Tag != "" {
		where = append(where, arg(q.Tag)+" = ANY(w.tags)")
	}
	query := workSelect + join
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY " + order + " LIMIT " + arg(q.Limit) + " OFFSET " + arg(q.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.Work
	for rows.Next() {
		w, err := scanWork(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func (r *communityRepository) CountWorks(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM works WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *communityRepository) SetWorkStatus(ctx context.Context, id int64, status, reason string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE works SET status = $2, review_reason = $3, updated_at = NOW() WHERE id = $1`, id, status, reason)
	return err
}

func (r *communityRepository) SetWorkFeatured(ctx context.Context, id int64, featured bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE works SET featured_at = CASE WHEN $2 THEN COALESCE(featured_at, NOW()) ELSE NULL END WHERE id = $1`, id, featured)
	return err
}

func (r *communityRepository) AddWorkView(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE works SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

func (r *communityRepository) AddWorkRemix(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE works SET remix_count = remix_count + 1 WHERE id = $1`, id)
	return err
}

func (r *communityRepository) idSet(ctx context.Context, query string, args ...any) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *communityRepository) FillViewerState(ctx context.Context, viewerID int64, works []service.Work) error {
	ids := make([]int64, 0, len(works))
	authors := make([]int64, 0, len(works))
	for _, w := range works {
		ids = append(ids, w.ID)
		authors = append(authors, w.UserID)
	}
	liked, err := r.idSet(ctx, `SELECT work_id FROM work_likes WHERE user_id = $1 AND work_id = ANY($2)`, viewerID, pq.Array(ids))
	if err != nil {
		return err
	}
	faved, err := r.idSet(ctx, `SELECT work_id FROM work_favorites WHERE user_id = $1 AND work_id = ANY($2)`, viewerID, pq.Array(ids))
	if err != nil {
		return err
	}
	followed, err := r.idSet(ctx, `SELECT followee_id FROM follows WHERE follower_id = $1 AND followee_id = ANY($2)`, viewerID, pq.Array(authors))
	if err != nil {
		return err
	}
	for i := range works {
		w := &works[i]
		w.LikedByMe, w.FavoritedByMe, w.Author.FollowedByMe, w.IsMine = liked[w.ID], faved[w.ID], followed[w.UserID], w.UserID == viewerID
	}
	return nil
}

// Interactions -----------------------------------------------------------------------------------

// toggle inserts or deletes a (work|user, user) row and runs the counter updates when it changed.
func (r *communityRepository) toggle(ctx context.Context, on bool, insert, remove string, args []any, onChange func(tx *sql.Tx, delta int) error) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, delta := remove, -1
	if on {
		stmt, delta = insert, 1
	}
	res, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return false, err
	}
	if err := onChange(tx, delta); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (r *communityRepository) SetLike(ctx context.Context, workID, userID int64, on bool) (bool, error) {
	return r.toggle(ctx, on,
		`INSERT INTO work_likes (work_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		`DELETE FROM work_likes WHERE work_id = $1 AND user_id = $2`, []any{workID, userID},
		func(tx *sql.Tx, delta int) error {
			if _, err := tx.ExecContext(ctx, `UPDATE works SET like_count = GREATEST(like_count + $2, 0) WHERE id = $1`, workID, delta); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `
UPDATE user_profiles SET likes_received = GREATEST(likes_received + $2, 0) WHERE user_id = (SELECT user_id FROM works WHERE id = $1)`, workID, delta)
			return err
		})
}

func (r *communityRepository) SetFavorite(ctx context.Context, workID, userID int64, on bool) (bool, error) {
	return r.toggle(ctx, on,
		`INSERT INTO work_favorites (work_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		`DELETE FROM work_favorites WHERE work_id = $1 AND user_id = $2`, []any{workID, userID},
		func(tx *sql.Tx, delta int) error {
			_, err := tx.ExecContext(ctx, `UPDATE works SET favorite_count = GREATEST(favorite_count + $2, 0) WHERE id = $1`, workID, delta)
			return err
		})
}

func (r *communityRepository) SetFollow(ctx context.Context, followerID, followeeID int64, on bool) (bool, error) {
	return r.toggle(ctx, on,
		`INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		`DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2`, []any{followerID, followeeID},
		func(tx *sql.Tx, delta int) error {
			if _, err := tx.ExecContext(ctx, `UPDATE user_profiles SET followers_count = GREATEST(followers_count + $2, 0) WHERE user_id = $1`, followeeID, delta); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE user_profiles SET following_count = GREATEST(following_count + $2, 0) WHERE user_id = $1`, followerID, delta)
			return err
		})
}

func (r *communityRepository) IsFollowing(ctx context.Context, followerID, followeeID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2)`, followerID, followeeID).Scan(&ok)
	return ok, err
}

func (r *communityRepository) ListFollows(ctx context.Context, userID int64, followers bool, limit, offset int) ([]service.CommunityProfile, error) {
	join, filter := "f.follower_id", "f.followee_id"
	if !followers {
		join, filter = "f.followee_id", "f.follower_id"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+profileColumns+` FROM follows f JOIN user_profiles p ON p.user_id = `+join+`
WHERE `+filter+` = $1 AND p.status = 'active' ORDER BY f.created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CommunityProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// Collections ------------------------------------------------------------------------------------

const collectionSelect = `
SELECT c.id, c.user_id, c.title, c.description, c.visibility, c.works_count, c.created_at, c.updated_at,
       COALESCE(p.handle, ''), COALESCE(p.display_name, ''), COALESCE(p.avatar_file, ''),
       ARRAY(SELECT m.thumb_file FROM collection_items ci JOIN works w ON w.id = ci.work_id
             JOIN work_media m ON m.work_id = w.id AND m.position = 0
             WHERE ci.collection_id = c.id AND w.status = 'approved' AND w.visibility <> 'private'
             ORDER BY ci.position, ci.added_at DESC LIMIT 4)
FROM collections c LEFT JOIN user_profiles p ON p.user_id = c.user_id`

func scanCollection(row rowScanner) (*service.Collection, error) {
	var c service.Collection
	author := service.CommunityAuthor{}
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.Description, &c.Visibility, &c.WorksCount, &c.CreatedAt, &c.UpdatedAt,
		&author.Handle, &author.DisplayName, &author.AvatarURL, pq.Array(&c.CoverFiles)); err != nil {
		return nil, err
	}
	author.UserID = c.UserID
	c.Author = &author
	return &c, nil
}

func (r *communityRepository) CreateCollection(ctx context.Context, c *service.Collection) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO collections (user_id, title, description, visibility) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		c.UserID, c.Title, c.Description, c.Visibility).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *communityRepository) UpdateCollection(ctx context.Context, c *service.Collection) error {
	_, err := r.db.ExecContext(ctx, `UPDATE collections SET title = $2, description = $3, visibility = $4, updated_at = NOW() WHERE id = $1`,
		c.ID, c.Title, c.Description, c.Visibility)
	return err
}

func (r *communityRepository) DeleteCollection(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id = $1`, id)
	return err
}

func (r *communityRepository) GetCollection(ctx context.Context, id int64) (*service.Collection, error) {
	c, err := scanCollection(r.db.QueryRowContext(ctx, collectionSelect+` WHERE c.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *communityRepository) ListCollections(ctx context.Context, userID int64, includePrivate bool) ([]service.Collection, error) {
	rows, err := r.db.QueryContext(ctx, collectionSelect+` WHERE c.user_id = $1 AND ($2 OR c.visibility = 'public') ORDER BY c.updated_at DESC`, userID, includePrivate)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *communityRepository) CountCollections(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM collections WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *communityRepository) SetCollectionItem(ctx context.Context, collectionID, workID int64, on bool) error {
	var err error
	if on {
		_, err = r.db.ExecContext(ctx, `
INSERT INTO collection_items (collection_id, work_id, position)
VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM collection_items WHERE collection_id = $1), 0))
ON CONFLICT DO NOTHING`, collectionID, workID)
	} else {
		_, err = r.db.ExecContext(ctx, `DELETE FROM collection_items WHERE collection_id = $1 AND work_id = $2`, collectionID, workID)
	}
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
UPDATE collections SET works_count = (SELECT COUNT(*) FROM collection_items WHERE collection_id = $1), updated_at = NOW() WHERE id = $1`, collectionID)
	return err
}

func (r *communityRepository) WorkCollections(ctx context.Context, workID, ownerID int64) ([]int64, error) {
	set, err := r.idSet(ctx, `
SELECT ci.collection_id FROM collection_items ci JOIN collections c ON c.id = ci.collection_id WHERE ci.work_id = $1 AND c.user_id = $2`, workID, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out, nil
}

// Notifications ----------------------------------------------------------------------------------

func (r *communityRepository) AddNotification(ctx context.Context, n *service.CommunityNotification) error {
	if n.ActorID != 0 && n.ActorID == n.UserID {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO community_notifications (user_id, kind, actor_id, work_id, detail) VALUES ($1, $2, NULLIF($3, 0), NULLIF($4, 0), $5)
ON CONFLICT (user_id, kind, COALESCE(actor_id, 0), COALESCE(work_id, 0)) WHERE read_at IS NULL DO NOTHING`,
		n.UserID, n.Kind, n.ActorID, n.WorkID, n.Detail)
	return err
}

func (r *communityRepository) ListNotifications(ctx context.Context, userID int64, limit int) ([]service.CommunityNotification, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT n.id, n.kind, COALESCE(n.actor_id, 0), COALESCE(ap.handle, ''), COALESCE(ap.display_name, ''), COALESCE(ap.avatar_file, ''),
       COALESCE(n.work_id, 0), COALESCE(w.title, ''), COALESCE(m.thumb_file, ''), n.detail, n.read_at IS NOT NULL, n.created_at
FROM community_notifications n
LEFT JOIN user_profiles ap ON ap.user_id = n.actor_id
LEFT JOIN works w ON w.id = n.work_id
LEFT JOIN work_media m ON m.work_id = n.work_id AND m.position = 0
WHERE n.user_id = $1 ORDER BY n.created_at DESC, n.id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CommunityNotification
	for rows.Next() {
		var n service.CommunityNotification
		actor := service.CommunityAuthor{}
		if err := rows.Scan(&n.ID, &n.Kind, &n.ActorID, &actor.Handle, &actor.DisplayName, &actor.AvatarURL,
			&n.WorkID, &n.WorkTitle, &n.WorkThumb, &n.Detail, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		if n.ActorID != 0 {
			actor.UserID = n.ActorID
			n.Actor = &actor
		}
		n.UserID = userID
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *communityRepository) MarkNotificationsRead(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE community_notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *communityRepository) UnreadNotifications(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM community_notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// Reports ----------------------------------------------------------------------------------------

func (r *communityRepository) CreateWorkReport(ctx context.Context, rep *service.WorkReport) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.QueryRowContext(ctx, `
INSERT INTO work_reports (work_id, reporter_id, reason, detail, ip) VALUES ($1, NULLIF($2, 0), $3, $4, $5) RETURNING id, created_at`,
		rep.WorkID, rep.ReporterID, rep.Reason, rep.Detail, rep.IP).Scan(&rep.ID, &rep.CreatedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE works SET report_count = report_count + 1 WHERE id = $1`, rep.WorkID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *communityRepository) CountWorkReportsSince(ctx context.Context, ip string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_reports WHERE ip = $1 AND created_at >= $2`, ip, since).Scan(&n)
	return n, err
}

func (r *communityRepository) ListWorkReports(ctx context.Context, status string, limit, offset int) ([]service.WorkReport, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT rp.id, rp.work_id, COALESCE(w.title, ''), rp.reason, rp.detail, rp.status, rp.created_at
FROM work_reports rp LEFT JOIN works w ON w.id = rp.work_id
WHERE ($1 = '' OR rp.status = $1) ORDER BY rp.created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.WorkReport
	for rows.Next() {
		var rp service.WorkReport
		if err := rows.Scan(&rp.ID, &rp.WorkID, &rp.WorkTitle, &rp.Reason, &rp.Detail, &rp.Status, &rp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, rows.Err()
}

func (r *communityRepository) SetWorkReportStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE work_reports SET status = $2 WHERE id = $1`, id, status)
	return err
}
