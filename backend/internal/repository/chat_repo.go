package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type chatRepository struct {
	db *sql.DB
}

// NewChatRepository creates the AI 对话 store (raw SQL).
func NewChatRepository(db *sql.DB) service.ChatRepository {
	return &chatRepository{db: db}
}

const chatConversationColumns = `id, user_id, title, system_prompt, api_key_id, model, pinned, created_at, updated_at`

func scanChatConversation(row interface{ Scan(...any) error }) (*service.ChatConversation, error) {
	var c service.ChatConversation
	var keyID sql.NullInt64
	if err := row.Scan(&c.ID, &c.UserID, &c.Title, &c.SystemPrompt, &keyID, &c.Model, &c.Pinned, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	if keyID.Valid {
		c.APIKeyID = &keyID.Int64
	}
	return &c, nil
}

func (r *chatRepository) ListConversations(ctx context.Context, userID int64, query string, limit int) ([]service.ChatConversation, error) {
	args := []any{userID, limit}
	where := `user_id = $1`
	if query != "" {
		args = append(args, "%"+escapeLikeWildcards(query)+"%")
		where += ` AND title ILIKE $3`
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+chatConversationColumns+` FROM chat_conversations WHERE `+where+`
ORDER BY pinned DESC, updated_at DESC, id DESC LIMIT $2`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ChatConversation{}
	for rows.Next() {
		c, err := scanChatConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *chatRepository) GetConversation(ctx context.Context, userID, id int64) (*service.ChatConversation, error) {
	c, err := scanChatConversation(r.db.QueryRowContext(ctx, `SELECT `+chatConversationColumns+` FROM chat_conversations WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *chatRepository) CreateConversation(ctx context.Context, c *service.ChatConversation) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO chat_conversations (user_id, title, system_prompt, api_key_id, model, pinned)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, updated_at`,
		c.UserID, c.Title, c.SystemPrompt, c.APIKeyID, c.Model, c.Pinned,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *chatRepository) UpdateConversation(ctx context.Context, c *service.ChatConversation, touch bool) error {
	return r.db.QueryRowContext(ctx, `
UPDATE chat_conversations
SET title = $3, system_prompt = $4, api_key_id = $5, model = $6, pinned = $7,
    updated_at = CASE WHEN $8 THEN NOW() ELSE updated_at END
WHERE id = $1 AND user_id = $2
RETURNING updated_at`,
		c.ID, c.UserID, c.Title, c.SystemPrompt, c.APIKeyID, c.Model, c.Pinned, touch,
	).Scan(&c.UpdatedAt)
}

func (r *chatRepository) DeleteConversation(ctx context.Context, userID, id int64) (bool, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	files, err := queryNames(ctx, tx, `
SELECT f.name FROM chat_files f JOIN chat_messages m ON m.id = f.message_id
WHERE m.conversation_id = $1 AND f.user_id = $2`, id, userID)
	if err != nil {
		return false, nil, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM chat_conversations WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if n == 0 {
		return false, nil, nil
	}
	return true, files, tx.Commit()
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func queryNames(ctx context.Context, q queryer, query string, args ...any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const chatMessageColumns = `id, conversation_id, role, kind, content, reasoning, files, model, status, error, prompt_tokens, completion_tokens, created_at`

func (r *chatRepository) ListMessages(ctx context.Context, conversationID int64) ([]service.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+chatMessageColumns+` FROM chat_messages WHERE conversation_id = $1 ORDER BY id`, conversationID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ChatMessage{}
	for rows.Next() {
		var m service.ChatMessage
		var files []byte
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Kind, &m.Content, &m.Reasoning, &files, &m.Model, &m.Status, &m.Error,
			&m.PromptTokens, &m.CompletionTokens, &m.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(files, &m.Files)
		if m.Files == nil {
			m.Files = []string{}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *chatRepository) CountMessages(ctx context.Context, conversationID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE conversation_id = $1`, conversationID).Scan(&n)
	return n, err
}

func chatFilesJSON(files []string) string {
	if files == nil {
		files = []string{}
	}
	b, _ := json.Marshal(files)
	return string(b)
}

func (r *chatRepository) CreateMessage(ctx context.Context, m *service.ChatMessage) error {
	if m.Files == nil {
		m.Files = []string{}
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO chat_messages (conversation_id, role, kind, content, reasoning, files, model, status, error, prompt_tokens, completion_tokens)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11)
RETURNING id, created_at`,
		m.ConversationID, m.Role, m.Kind, m.Content, m.Reasoning, chatFilesJSON(m.Files), m.Model, m.Status, m.Error, m.PromptTokens, m.CompletionTokens,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *chatRepository) UpdateMessage(ctx context.Context, m *service.ChatMessage) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE chat_messages
SET content = $2, reasoning = $3, files = $4::jsonb, model = $5, status = $6, error = $7, prompt_tokens = $8, completion_tokens = $9
WHERE id = $1`,
		m.ID, m.Content, m.Reasoning, chatFilesJSON(m.Files), m.Model, m.Status, m.Error, m.PromptTokens, m.CompletionTokens)
	return err
}

func (r *chatRepository) DeleteMessagesFrom(ctx context.Context, conversationID, fromID int64, keep []string) ([]string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if len(keep) > 0 {
		if _, err := tx.ExecContext(ctx, `
UPDATE chat_files SET message_id = NULL
WHERE name = ANY($3) AND message_id IN (SELECT id FROM chat_messages WHERE conversation_id = $1 AND id >= $2)`,
			conversationID, fromID, pq.Array(keep)); err != nil {
			return nil, err
		}
	}
	files, err := queryNames(ctx, tx, `
SELECT f.name FROM chat_files f JOIN chat_messages m ON m.id = f.message_id
WHERE m.conversation_id = $1 AND m.id >= $2`, conversationID, fromID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chat_messages WHERE conversation_id = $1 AND id >= $2`, conversationID, fromID); err != nil {
		return nil, err
	}
	return files, tx.Commit()
}

func (r *chatRepository) FailPending(ctx context.Context, msg string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE chat_messages SET status = 'error', error = $1 WHERE status = 'pending'`, msg)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *chatRepository) CreateFile(ctx context.Context, f *service.ChatFile) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO chat_files (name, user_id, mime, size, width, height) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING created_at`, f.Name, f.UserID, f.Mime, f.Size, f.Width, f.Height).Scan(&f.CreatedAt)
}

func (r *chatRepository) GetFile(ctx context.Context, name string) (*service.ChatFile, error) {
	var f service.ChatFile
	var msgID, convID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
SELECT f.name, f.user_id, f.message_id, m.conversation_id, f.mime, f.size, f.width, f.height, f.created_at
FROM chat_files f LEFT JOIN chat_messages m ON m.id = f.message_id WHERE f.name = $1`, name).
		Scan(&f.Name, &f.UserID, &msgID, &convID, &f.Mime, &f.Size, &f.Width, &f.Height, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if msgID.Valid {
		f.MessageID = &msgID.Int64
	}
	f.ConversationID = convID.Int64
	return &f, nil
}

func (r *chatRepository) AttachFiles(ctx context.Context, messageID int64, names []string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE chat_files SET message_id = $1 WHERE name = ANY($2) AND message_id IS NULL`, messageID, pq.Array(names))
	return err
}

func (r *chatRepository) CountFilesSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_files WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *chatRepository) DeleteUnsentFiles(ctx context.Context, before time.Time) ([]string, error) {
	return queryNames(ctx, r.db, `DELETE FROM chat_files WHERE message_id IS NULL AND created_at < $1 RETURNING name`, before)
}

func (r *chatRepository) KnownFiles(ctx context.Context, names []string) (map[string]bool, error) {
	found, err := queryNames(ctx, r.db, `SELECT name FROM chat_files WHERE name = ANY($1)`, pq.Array(names))
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(found))
	for _, n := range found {
		out[n] = true
	}
	return out, nil
}
