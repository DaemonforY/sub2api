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

func TestPromptLibraryFoldersAndPins(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	alice := mustCreateUser(t, integrationEntClient, &service.User{Email: "pf-a-" + suffix + "@prompt.test", Username: "pfa" + suffix[len(suffix)-6:]})
	bob := mustCreateUser(t, integrationEntClient, &service.User{Email: "pf-b-" + suffix + "@prompt.test", Username: "pfb" + suffix[len(suffix)-6:]})
	repo := NewPromptLibraryRepository(integrationDB)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM prompt_items WHERE owner_user_id IN ($1, $2)`, alice.ID, bob.ID)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM prompt_folders WHERE user_id IN ($1, $2)`, alice.ID, bob.ID)
	})

	folder := &service.PromptFolder{UserID: alice.ID, Name: "海报"}
	require.NoError(t, repo.CreateFolder(ctx, folder))
	require.ErrorIs(t, repo.CreateFolder(ctx, &service.PromptFolder{UserID: alice.ID, Name: "海报"}), service.ErrPromptFolderExists)
	require.NoError(t, repo.CreateFolder(ctx, &service.PromptFolder{UserID: bob.ID, Name: "海报"}), "names are per user")
	owned, err := repo.FolderOwnedBy(ctx, folder.ID, bob.ID)
	require.NoError(t, err)
	require.False(t, owned)

	insert := func(owner int64, prompt string, folderID *int64) int64 {
		o := owner
		item := &service.PromptItem{SourceID: service.PromptSourceUser, ExternalID: prompt + suffix, OwnerUserID: &o, Kind: "image", Title: prompt,
			Prompt: prompt, ReferenceImageURLs: []string{}, SourceTags: []string{}, Scenes: []string{"other"}, Model: "unknown", Lang: "zh",
			Visibility: service.PromptVisibilityPrivate, Status: service.PromptStatusActive, DedupeKey: service.PromptDedupeKey(prompt + suffix), FolderID: folderID}
		require.NoError(t, repo.Insert(ctx, item))
		return item.ID
	}
	a1 := insert(alice.ID, "a1", &folder.ID)
	a2 := insert(alice.ID, "a2", nil)
	a3 := insert(alice.ID, "a3", nil)
	b1 := insert(bob.ID, "b1", nil)

	counts, err := repo.CountByFolder(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, map[int64]int64{folder.ID: 1, 0: 2}, counts)

	// Bob's id in Alice's batch is ignored.
	n, err := repo.BatchMine(ctx, alice.ID, []int64{a2, b1}, service.PromptMineBatchOp{Action: "move", FolderID: folder.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	_, err = repo.BatchMine(ctx, alice.ID, []int64{a3}, service.PromptMineBatchOp{Action: "pin"})
	require.NoError(t, err)

	list, total, err := repo.List(ctx, service.PromptListQuery{OwnerUserID: alice.ID, FolderID: folder.ID, Sort: service.PromptSortLatest, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.ElementsMatch(t, []int64{a1, a2}, []int64{list[0].ID, list[1].ID})
	list, _, err = repo.List(ctx, service.PromptListQuery{OwnerUserID: alice.ID, Sort: service.PromptSortLatest, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, a3, list[0].ID, "pinned first")
	require.True(t, list[0].Pinned)

	// "used" puts the last used first (after pins).
	_, err = repo.RecordUse(ctx, alice.ID, a1, time.Minute)
	require.NoError(t, err)
	list, _, err = repo.List(ctx, service.PromptListQuery{OwnerUserID: alice.ID, Sort: service.PromptSortUsed, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, []int64{a3, a1, a2}, []int64{list[0].ID, list[1].ID, list[2].ID})
	list, _, err = repo.List(ctx, service.PromptListQuery{OwnerUserID: alice.ID, Sort: service.PromptSortMostUsed, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, a1, list[1].ID)

	keys, err := repo.OwnedDedupeKeys(ctx, alice.ID, []string{service.PromptDedupeKey("a1" + suffix), service.PromptDedupeKey("b1" + suffix)})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{service.PromptDedupeKey("a1" + suffix): true}, keys)

	// Deleting a folder keeps its prompts (now 未分类) unless asked to delete them.
	ok, err := repo.DeleteFolder(ctx, bob.ID, folder.ID, true)
	require.NoError(t, err)
	require.False(t, ok, "not Bob's folder")
	ok, err = repo.DeleteFolder(ctx, alice.ID, folder.ID, false)
	require.NoError(t, err)
	require.True(t, ok)
	got, err := repo.Get(ctx, a1)
	require.NoError(t, err)
	require.Nil(t, got.FolderID)

	second := &service.PromptFolder{UserID: alice.ID, Name: "头像"}
	require.NoError(t, repo.CreateFolder(ctx, second))
	_, err = repo.BatchMine(ctx, alice.ID, []int64{a1}, service.PromptMineBatchOp{Action: "move", FolderID: second.ID})
	require.NoError(t, err)
	ok, err = repo.DeleteFolder(ctx, alice.ID, second.ID, true)
	require.NoError(t, err)
	require.True(t, ok)
	got, err = repo.Get(ctx, a1)
	require.NoError(t, err)
	require.Nil(t, got, "deleted with its folder")
}
