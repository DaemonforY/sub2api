//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// mineRepo keeps a user's prompts in memory for the import / batch rules.
type mineRepo struct {
	PromptLibraryRepository
	items   []PromptItem
	folders map[int64]int64 // folder id -> owner
	batches []PromptMineBatchOp
}

func (r *mineRepo) CountOwned(_ context.Context, userID int64) (int, error) {
	n := 0
	for _, item := range r.items {
		if item.OwnerUserID != nil && *item.OwnerUserID == userID {
			n++
		}
	}
	return n, nil
}
func (r *mineRepo) Insert(_ context.Context, item *PromptItem) error {
	item.ID = int64(len(r.items) + 1)
	r.items = append(r.items, *item)
	return nil
}
func (r *mineRepo) OwnedDedupeKeys(_ context.Context, userID int64, keys []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, item := range r.items {
		for _, key := range keys {
			if item.OwnerUserID != nil && *item.OwnerUserID == userID && item.DedupeKey == key {
				out[key] = true
			}
		}
	}
	return out, nil
}
func (r *mineRepo) FolderOwnedBy(_ context.Context, id, userID int64) (bool, error) {
	return r.folders[id] == userID, nil
}
func (r *mineRepo) BatchMine(_ context.Context, _ int64, ids []int64, op PromptMineBatchOp) (int64, error) {
	r.batches = append(r.batches, op)
	return int64(len(ids)), nil
}

func TestImportMineSkipsDuplicatesAndStaysPrivate(t *testing.T) {
	repo := &mineRepo{folders: map[int64]int64{3: 7, 4: 8}}
	svc := NewPromptLibraryService(repo, nil)
	ctx := context.Background()

	res, err := svc.ImportMine(ctx, 7, []PromptUserInput{
		{Prompt: "一只橘猫坐在窗台上，午后阳光，胶片质感", Share: true, CoverURL: "/api/v1/prompt-library/covers/x.png"},
		{Prompt: "  "},
		{Prompt: "一只橘猫坐在窗台上，午后阳光，胶片质感"},
		{Title: "海报", Prompt: "咖啡店开业海报，暖色调", Tags: []string{"#海报"}},
	}, 3)
	require.NoError(t, err)
	require.Equal(t, 2, res.Created)
	require.Equal(t, 1, res.Duplicates, "the same prompt twice in one import is saved once")
	for _, item := range repo.items {
		require.Equal(t, PromptVisibilityPrivate, item.Visibility, "imports are never shared")
		require.Empty(t, item.CoverURL)
		require.NotNil(t, item.FolderID)
		require.Equal(t, int64(3), *item.FolderID)
		require.NotEmpty(t, item.Title, "a title is generated when missing")
	}
	require.Equal(t, []string{"海报"}, repo.items[1].Tags)

	// Importing the same note again adds nothing.
	res, err = svc.ImportMine(ctx, 7, []PromptUserInput{{Prompt: "咖啡店开业海报，暖色调"}}, 0)
	require.NoError(t, err)
	require.Equal(t, 0, res.Created)
	require.Equal(t, 1, res.Duplicates)

	// Someone else's folder is refused.
	_, err = svc.ImportMine(ctx, 7, []PromptUserInput{{Prompt: "新的提示词"}}, 4)
	require.ErrorIs(t, err, ErrPromptFolderNotFound)

	_, err = svc.ImportMine(ctx, 7, []PromptUserInput{{Prompt: " "}}, 0)
	require.ErrorIs(t, err, ErrPromptImportEmpty)
	_, err = svc.ImportMine(ctx, 7, make([]PromptUserInput, promptImportMaxItems+1), 0)
	require.ErrorIs(t, err, ErrPromptImportTooMany)
}

func TestBatchMineValidatesTheAction(t *testing.T) {
	repo := &mineRepo{folders: map[int64]int64{3: 7}}
	svc := NewPromptLibraryService(repo, nil)
	ctx := context.Background()

	n, err := svc.BatchMine(ctx, 7, []int64{1, 2}, PromptMineBatchOp{Action: "move", FolderID: 3})
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	_, err = svc.BatchMine(ctx, 7, []int64{1}, PromptMineBatchOp{Action: "move"})
	require.NoError(t, err, "move to 未分类")
	_, err = svc.BatchMine(ctx, 8, []int64{1}, PromptMineBatchOp{Action: "move", FolderID: 3})
	require.ErrorIs(t, err, ErrPromptFolderNotFound)
	_, err = svc.BatchMine(ctx, 7, []int64{1}, PromptMineBatchOp{Action: "share"})
	require.ErrorIs(t, err, ErrPromptMineBatchOp)
	_, err = svc.BatchMine(ctx, 7, nil, PromptMineBatchOp{Action: "pin"})
	require.ErrorIs(t, err, ErrPromptMineBatchOp)
	require.Len(t, repo.batches, 2)
}

func TestFolderNamesAndOwnSorts(t *testing.T) {
	name, err := normalizeFolderName("  小红书   封面 ")
	require.NoError(t, err)
	require.Equal(t, "小红书 封面", name)
	_, err = normalizeFolderName("   ")
	require.ErrorIs(t, err, ErrPromptFolderName)
	_, err = normalizeFolderName("一二三四五六七八九十一二三四五六七八九十一")
	require.ErrorIs(t, err, ErrPromptFolderName)

	q := PromptListQuery{OwnerUserID: 7, Sort: PromptSortUsed, FolderID: -1}
	q.normalize()
	require.Equal(t, PromptSortUsed, q.Sort)
	require.Equal(t, int64(-1), q.FolderID)
	q = PromptListQuery{Sort: PromptSortUsed, FolderID: 5}
	q.normalize()
	require.Equal(t, PromptSortPopular, q.Sort, "own sorts are only for 「我的」")
	require.Equal(t, int64(0), q.FolderID)

	item := PromptItem{FolderID: &q.FolderID, Pinned: true}
	view := item.PublicView()
	require.Nil(t, view.FolderID, "others never see the owner's folders")
	require.False(t, view.Pinned)
}

func TestPromptTemplateTextForTitles(t *testing.T) {
	require.Equal(t, "极简产品图，香水，白底", promptTemplateText("极简产品图，{产品=香水}，白底"))
	require.Equal(t, "主体坐在窗台，胶片", promptTemplateText(`{主体}坐在窗台，{argument name="风格" default="胶片"}`))
	require.Equal(t, `{"style": "anime"}`, promptTemplateText(`{"style": "anime"}`))
}
