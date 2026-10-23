//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChatRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "chat-" + suffix + "@chat.test", Username: "chat" + suffix[len(suffix)-4:]})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "chat2-" + suffix + "@chat.test", Username: "chat2" + suffix[len(suffix)-4:]})
	t.Cleanup(func() {
		for _, id := range []int64{user.ID, other.ID} {
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM chat_files WHERE user_id = $1`, id)
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM chat_conversations WHERE user_id = $1`, id)
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
		}
	})
	repo := NewChatRepository(integrationDB)

	keyID := int64(42)
	conv := &service.ChatConversation{UserID: user.ID, Title: "翻译 100%_好", SystemPrompt: "你是翻译", APIKeyID: &keyID, Model: "gpt-5.5"}
	require.NoError(t, repo.CreateConversation(ctx, conv))
	second := &service.ChatConversation{UserID: user.ID, Title: "第二个"}
	require.NoError(t, repo.CreateConversation(ctx, second))

	got, err := repo.GetConversation(ctx, user.ID, conv.ID)
	require.NoError(t, err)
	require.Equal(t, "你是翻译", got.SystemPrompt)
	require.Equal(t, keyID, *got.APIKeyID)
	none, err := repo.GetConversation(ctx, other.ID, conv.ID)
	require.NoError(t, err)
	require.Nil(t, none)

	// Pinned first, then most recently updated; the search escapes LIKE wildcards.
	conv.Pinned = true
	require.NoError(t, repo.UpdateConversation(ctx, conv, true))
	list, err := repo.ListConversations(ctx, user.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, conv.ID, list[0].ID)
	found, err := repo.ListConversations(ctx, user.ID, "100%_", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	found, err = repo.ListConversations(ctx, user.ID, "0%好", 10)
	require.NoError(t, err)
	require.Empty(t, found)

	// Messages and pictures.
	newFile := func(name string) {
		require.NoError(t, repo.CreateFile(ctx, &service.ChatFile{Name: name, UserID: user.ID, Mime: "image/png", Size: 10, Width: 4, Height: 3}))
	}
	a, b, c := suffix[:10]+"a.png", suffix[:10]+"b.png", suffix[:10]+"c.png"
	newFile(a)
	newFile(b)
	newFile(c)
	q := &service.ChatMessage{ConversationID: conv.ID, Role: service.ChatRoleUser, Kind: service.ChatKindText, Content: "看图", Files: []string{a, b}, Status: service.ChatStatusDone}
	require.NoError(t, repo.CreateMessage(ctx, q))
	require.NoError(t, repo.AttachFiles(ctx, q.ID, q.Files))
	reply := &service.ChatMessage{ConversationID: conv.ID, Role: service.ChatRoleAssistant, Kind: service.ChatKindText, Status: service.ChatStatusPending}
	require.NoError(t, repo.CreateMessage(ctx, reply))
	reply.Content, reply.Reasoning, reply.Status, reply.PromptTokens = "是一只猫", "看看", service.ChatStatusDone, 30
	require.NoError(t, repo.UpdateMessage(ctx, reply))

	msgs, err := repo.ListMessages(ctx, conv.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, []string{a, b}, msgs[0].Files)
	require.Equal(t, []string{}, msgs[1].Files)
	require.Equal(t, "是一只猫", msgs[1].Content)
	require.Equal(t, 30, msgs[1].PromptTokens)
	n, err := repo.CountMessages(ctx, conv.ID)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	f, err := repo.GetFile(ctx, a)
	require.NoError(t, err)
	require.Equal(t, q.ID, *f.MessageID)
	require.Equal(t, conv.ID, f.ConversationID)
	unsent, err := repo.GetFile(ctx, c)
	require.NoError(t, err)
	require.Nil(t, unsent.MessageID)
	count, err := repo.CountFilesSince(ctx, user.ID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Editing the question keeps picture a and drops b with the messages.
	gone, err := repo.DeleteMessagesFrom(ctx, conv.ID, q.ID, []string{a})
	require.NoError(t, err)
	require.Equal(t, []string{b}, gone)
	msgs, err = repo.ListMessages(ctx, conv.ID)
	require.NoError(t, err)
	require.Empty(t, msgs)
	kept, err := repo.GetFile(ctx, a)
	require.NoError(t, err)
	require.Nil(t, kept.MessageID)
	known, err := repo.KnownFiles(ctx, []string{a, b, c})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{a: true, c: true}, known)

	// Unsent pictures are swept after the cut-off.
	swept, err := repo.DeleteUnsentFiles(ctx, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{a, c}, swept)

	// Pending replies left by a previous process are failed.
	p := &service.ChatMessage{ConversationID: second.ID, Role: service.ChatRoleAssistant, Kind: service.ChatKindImage, Status: service.ChatStatusPending}
	require.NoError(t, repo.CreateMessage(ctx, p))
	newFile(c)
	require.NoError(t, repo.AttachFiles(ctx, p.ID, []string{c}))
	failed, err := repo.FailPending(ctx, "中断")
	require.NoError(t, err)
	require.GreaterOrEqual(t, failed, int64(1))
	msgs, err = repo.ListMessages(ctx, second.ID)
	require.NoError(t, err)
	require.Equal(t, service.ChatStatusError, msgs[0].Status)
	require.Equal(t, "中断", msgs[0].Error)

	// Deleting a conversation reports its pictures; another user can't delete it.
	ok, _, err := repo.DeleteConversation(ctx, other.ID, second.ID)
	require.NoError(t, err)
	require.False(t, ok)
	ok, files, err := repo.DeleteConversation(ctx, user.ID, second.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{c}, files)
	known, err = repo.KnownFiles(ctx, []string{c})
	require.NoError(t, err)
	require.Empty(t, known)
}
