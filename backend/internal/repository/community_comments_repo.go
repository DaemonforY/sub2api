package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// Comments of community works (see service/community_comments.go).

// commentFrom selects comments with their author and the author answered, from source (aliased c).
func commentFrom(source string) string {
	return `
SELECT c.id, c.work_id, c.user_id, COALESCE(c.parent_id, 0), COALESCE(c.reply_to_id, 0), COALESCE(c.reply_to_user_id, 0),
       COALESCE(p.handle, ''), COALESCE(p.display_name, ''), COALESCE(p.avatar_file, ''),
       COALESCE(rp.handle, ''), COALESCE(rp.display_name, ''), COALESCE(rp.avatar_file, ''),
       c.body, c.status, c.review_flags, c.reply_count, c.report_count, c.created_at, COALESCE(w.title, '')
FROM ` + source + ` c
LEFT JOIN user_profiles p ON p.user_id = c.user_id
LEFT JOIN user_profiles rp ON rp.user_id = c.reply_to_user_id
LEFT JOIN works w ON w.id = c.work_id`
}

// visibleComment: approved, or the viewer's own waiting for review ($2 is the viewer).
const visibleComment = `(c.status = 'approved' OR (c.status = 'pending' AND c.user_id = $2))`

func scanComment(row rowScanner) (*service.WorkComment, error) {
	var (
		c      service.WorkComment
		author service.CommunityAuthor
		to     service.CommunityAuthor
	)
	if err := row.Scan(&c.ID, &c.WorkID, &c.UserID, &c.ParentID, &c.ReplyToID, &c.ReplyToUser,
		&author.Handle, &author.DisplayName, &author.AvatarURL, &to.Handle, &to.DisplayName, &to.AvatarURL,
		&c.Body, &c.Status, pq.Array(&c.ReviewFlags), &c.ReplyCount, &c.ReportCount, &c.CreatedAt, &c.WorkTitle); err != nil {
		return nil, err
	}
	author.UserID = c.UserID
	c.Author = &author
	if c.ReplyToUser != 0 && to.Handle != "" {
		to.UserID = c.ReplyToUser
		c.ReplyTo = &to
	}
	return &c, nil
}

func (r *communityRepository) queryComments(ctx context.Context, query string, args ...any) ([]service.WorkComment, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.WorkComment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ListComments: top-level comments, newest first; removed ones stay while they have replies.
func (r *communityRepository) ListComments(ctx context.Context, workID, viewerID int64, limit, offset int) ([]service.WorkComment, error) {
	return r.queryComments(ctx, commentFrom("work_comments")+`
WHERE c.work_id = $1 AND c.parent_id IS NULL AND (`+visibleComment+` OR c.reply_count > 0)
ORDER BY c.created_at DESC, c.id DESC LIMIT $3 OFFSET $4`, workID, viewerID, limit, offset)
}

// FirstReplies: the first n replies of each top-level comment, oldest first.
func (r *communityRepository) FirstReplies(ctx context.Context, parentIDs []int64, viewerID int64, n int) (map[int64][]service.WorkComment, error) {
	list, err := r.queryComments(ctx, commentFrom(`(
    SELECT c.*, ROW_NUMBER() OVER (PARTITION BY c.parent_id ORDER BY c.created_at, c.id) AS rn
    FROM work_comments c WHERE c.parent_id = ANY($1) AND `+visibleComment+`
)`)+`
WHERE c.rn <= $3 ORDER BY c.parent_id, c.created_at, c.id`, pq.Array(parentIDs), viewerID, n)
	if err != nil {
		return nil, err
	}
	out := map[int64][]service.WorkComment{}
	for _, c := range list {
		out[c.ParentID] = append(out[c.ParentID], c)
	}
	return out, nil
}

func (r *communityRepository) ListReplies(ctx context.Context, parentID, viewerID int64, limit, offset int) ([]service.WorkComment, error) {
	return r.queryComments(ctx, commentFrom("work_comments")+`
WHERE c.parent_id = $1 AND `+visibleComment+`
ORDER BY c.created_at, c.id LIMIT $3 OFFSET $4`, parentID, viewerID, limit, offset)
}

func (r *communityRepository) GetComment(ctx context.Context, id int64) (*service.WorkComment, error) {
	c, err := scanComment(r.db.QueryRowContext(ctx, commentFrom("work_comments")+` WHERE c.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// recountComments refreshes the work's comment count and, for a reply, its parent's reply count.
func recountComments(ctx context.Context, tx *sql.Tx, workID, parentID int64) error {
	if parentID != 0 {
		if _, err := tx.ExecContext(ctx, `
UPDATE work_comments SET reply_count = (SELECT COUNT(*) FROM work_comments r WHERE r.parent_id = $1 AND r.status = 'approved')
WHERE id = $1`, parentID); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
UPDATE works SET comment_count = (SELECT COUNT(*) FROM work_comments c WHERE c.work_id = $1 AND c.status = 'approved')
WHERE id = $1`, workID)
	return err
}

func (r *communityRepository) CreateComment(ctx context.Context, c *service.WorkComment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if c.ReviewFlags == nil {
		c.ReviewFlags = []string{}
	}
	if err := tx.QueryRowContext(ctx, `
INSERT INTO work_comments (work_id, user_id, parent_id, reply_to_id, reply_to_user_id, body, status, review_flags, ip)
VALUES ($1, $2, NULLIF($3, 0), NULLIF($4, 0), NULLIF($5, 0), $6, $7, $8, $9) RETURNING id, created_at`,
		c.WorkID, c.UserID, c.ParentID, c.ReplyToID, c.ReplyToUser, c.Body, c.Status, pq.Array(c.ReviewFlags), c.IP).Scan(&c.ID, &c.CreatedAt); err != nil {
		return err
	}
	if err := recountComments(ctx, tx, c.WorkID, c.ParentID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *communityRepository) SetCommentStatus(ctx context.Context, id int64, status string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var workID, parentID int64
	if err := tx.QueryRowContext(ctx, `
UPDATE work_comments SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING work_id, COALESCE(parent_id, 0)`, id, status).Scan(&workID, &parentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrCommunityCommentNotFound
		}
		return err
	}
	if err := recountComments(ctx, tx, workID, parentID); err != nil {
		return err
	}
	return tx.Commit()
}

// CountComments counts the user's comments since (any status, for rate limits).
func (r *communityRepository) CountComments(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_comments WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *communityRepository) HasRecentComment(ctx context.Context, userID, workID int64, body string, since time.Time) (bool, error) {
	var found bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (SELECT 1 FROM work_comments WHERE user_id = $1 AND work_id = $2 AND body = $3 AND created_at >= $4)`,
		userID, workID, body, since).Scan(&found)
	return found, err
}

// AdminListComments: pending | reported (still shown) | hidden | "" (all), newest first.
func (r *communityRepository) AdminListComments(ctx context.Context, status string, limit, offset int) ([]service.WorkComment, error) {
	where := "TRUE"
	order := "c.created_at DESC, c.id DESC"
	switch status {
	case "pending":
		where = "c.status = 'pending'"
	case "reported":
		where = "c.report_count > 0 AND c.status IN ('approved', 'pending')"
		order = "c.report_count DESC, " + order
	case "hidden":
		where = "c.status = 'hidden'"
	}
	return r.queryComments(ctx, commentFrom("work_comments")+`
WHERE `+where+` ORDER BY `+order+` LIMIT $1 OFFSET $2`, limit, offset)
}
