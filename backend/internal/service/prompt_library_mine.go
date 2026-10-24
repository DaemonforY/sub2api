package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 我的提示词: people keep their prompts in notes and chats, so the canvas lets them paste or drop a
// whole note, split it into prompts and save them at once (ImportMine), file them into folders
// and pin the ones they use most. Everything here only touches the user's own prompts.

const (
	promptFolderNameMaxRunes = 20
	promptMaxFolders         = 50
	promptImportMaxItems     = 100
	promptBatchMaxIDs        = 500
)

var (
	ErrPromptFolderNotFound = infraerrors.NotFound("PROMPT_FOLDER_NOT_FOUND", "文件夹不存在或已删除，请刷新后重试（Folder not found）")
	ErrPromptFolderName     = infraerrors.BadRequest("PROMPT_FOLDER_NAME", "文件夹名称需为 1–20 个字（Folder name must be 1–20 characters）")
	ErrPromptFolderExists   = infraerrors.Conflict("PROMPT_FOLDER_EXISTS", "已经有同名文件夹了，换个名字吧（A folder with this name exists）")
	ErrPromptFolderTooMany  = infraerrors.BadRequest("PROMPT_FOLDER_TOO_MANY", "最多建 50 个文件夹（Too many folders）")
	ErrPromptImportEmpty    = infraerrors.BadRequest("PROMPT_IMPORT_EMPTY", "没有可导入的提示词，请先粘贴或选择文件（Nothing to import）")
	ErrPromptImportTooMany  = infraerrors.BadRequest("PROMPT_IMPORT_TOO_MANY", "一次最多导入 100 条，请分批导入（At most 100 prompts per import）")
	ErrPromptMineBatchOp    = infraerrors.BadRequest("PROMPT_MINE_BATCH_INVALID", "批量操作参数不正确，请刷新后重试（Invalid batch operation）")
)

// PromptFolder is one of a user's folders; Count is filled by listings.
type PromptFolder struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	Count     int64     `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PromptFolderList is the folder bar of 「我的」.
type PromptFolderList struct {
	Folders  []PromptFolder `json:"folders"`
	Unfiled  int64          `json:"unfiled"`
	Total    int64          `json:"total"`
	MaxItems int            `json:"max_items"`
}

// PromptMineBatchOp is a batch action on the user's own prompts.
type PromptMineBatchOp struct {
	// move / pin / unpin / delete
	Action string `json:"action"`
	// FolderID for move: 0 = 未分类.
	FolderID int64 `json:"folder_id"`
}

// PromptImportResult says what an import did.
type PromptImportResult struct {
	Created int `json:"created"`
	// Duplicates were skipped: the user already has the same prompt (or it was in the batch twice).
	Duplicates int          `json:"duplicates"`
	Items      []PromptItem `json:"items"`
}

// Template variables as the canvas writes them: {主体}, {主体=橘猫}, {argument name="主体" default="橘猫"}.
var (
	promptArgumentVar = regexp.MustCompile(`\{argument\s+name="([^"]{1,30})"(?:\s+default="([^"]*)")?\s*\}`)
	promptSimpleVar   = regexp.MustCompile(`\{([^{}"'\n=:：,，]{1,12})(?:=([^{}"\n]{0,60}))?\}`)
)

// promptTemplateText reads a template as plain text (defaults, else the variable names), so a
// generated title says 「极简产品图，香水」 rather than 「极简产品图，{产品=香水」.
func promptTemplateText(prompt string) string {
	fill := func(re *regexp.Regexp, text string) string {
		return re.ReplaceAllStringFunc(text, func(m string) string {
			parts := re.FindStringSubmatch(m)
			if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
				return strings.TrimSpace(parts[2])
			}
			return strings.TrimSpace(parts[1])
		})
	}
	return fill(promptSimpleVar, fill(promptArgumentVar, prompt))
}

func folderRef(id *int64) *int64 {
	if id == nil || *id <= 0 {
		return nil
	}
	v := *id
	return &v
}

func normalizeFolderName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	n := utf8.RuneCountInString(name)
	if n == 0 || n > promptFolderNameMaxRunes {
		return "", ErrPromptFolderName
	}
	return name, nil
}

// ListFolders returns the user's folders with counts, plus 未分类 and the total.
func (s *PromptLibraryService) ListFolders(ctx context.Context, userID int64) (*PromptFolderList, error) {
	folders, err := s.repo.ListFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountByFolder(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &PromptFolderList{Folders: []PromptFolder{}, Unfiled: counts[0], MaxItems: promptUserMaxItems}
	for _, n := range counts {
		out.Total += n
	}
	for _, f := range folders {
		f.Count = counts[f.ID]
		out.Folders = append(out.Folders, f)
	}
	return out, nil
}

func (s *PromptLibraryService) CreateFolder(ctx context.Context, userID int64, name string) (*PromptFolder, error) {
	name, err := normalizeFolderName(name)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	if n >= promptMaxFolders {
		return nil, ErrPromptFolderTooMany
	}
	folder := &PromptFolder{UserID: userID, Name: name, SortOrder: n}
	if err := s.repo.CreateFolder(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

// UpdateFolder renames or reorders one of the user's folders.
func (s *PromptLibraryService) UpdateFolder(ctx context.Context, userID, id int64, name string, sortOrder int) (*PromptFolder, error) {
	name, err := normalizeFolderName(name)
	if err != nil {
		return nil, err
	}
	folder := &PromptFolder{ID: id, UserID: userID, Name: name, SortOrder: sortOrder}
	if err := s.repo.UpdateFolder(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *PromptLibraryService) DeleteFolder(ctx context.Context, userID, id int64, withItems bool) error {
	ok, err := s.repo.DeleteFolder(ctx, userID, id, withItems)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPromptFolderNotFound
	}
	return nil
}

// BatchMine moves, pins, unpins or deletes several of the user's prompts.
func (s *PromptLibraryService) BatchMine(ctx context.Context, userID int64, ids []int64, op PromptMineBatchOp) (int64, error) {
	if len(ids) == 0 || len(ids) > promptBatchMaxIDs {
		return 0, ErrPromptMineBatchOp
	}
	switch op.Action {
	case "move":
		if op.FolderID < 0 {
			return 0, ErrPromptMineBatchOp
		}
		if op.FolderID > 0 {
			owned, err := s.repo.FolderOwnedBy(ctx, op.FolderID, userID)
			if err != nil {
				return 0, err
			}
			if !owned {
				return 0, ErrPromptFolderNotFound
			}
		}
	case "pin", "unpin", "delete":
	default:
		return 0, ErrPromptMineBatchOp
	}
	return s.repo.BatchMine(ctx, userID, ids, op)
}

// ImportMine saves many private prompts at once (a pasted note or a file split by the canvas).
// Prompts the user already has are skipped, so importing the same note twice adds nothing.
func (s *PromptLibraryService) ImportMine(ctx context.Context, userID int64, inputs []PromptUserInput, folderID int64) (*PromptImportResult, error) {
	if len(inputs) == 0 {
		return nil, ErrPromptImportEmpty
	}
	if len(inputs) > promptImportMaxItems {
		return nil, ErrPromptImportTooMany
	}
	if folderID > 0 {
		owned, err := s.repo.FolderOwnedBy(ctx, folderID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrPromptFolderNotFound
		}
	}
	valid := make([]PromptUserInput, 0, len(inputs))
	keys := make([]string, 0, len(inputs))
	for _, in := range inputs {
		// Covers and sharing are set per prompt afterwards; an import is always private.
		in.CoverURL = ""
		in.Share = false
		in.FolderID = &folderID
		if strings.TrimSpace(in.Prompt) == "" {
			continue
		}
		if err := s.validateUserInput(ctx, userID, &in); err != nil {
			return nil, err
		}
		valid = append(valid, in)
		keys = append(keys, PromptDedupeKey(in.Prompt))
	}
	if len(valid) == 0 {
		return nil, ErrPromptImportEmpty
	}
	existing, err := s.repo.OwnedDedupeKeys(ctx, userID, keys)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountOwned(ctx, userID)
	if err != nil {
		return nil, err
	}
	res := &PromptImportResult{Items: []PromptItem{}}
	seen := map[string]bool{}
	for i, in := range valid {
		key := keys[i]
		if existing[key] || seen[key] {
			res.Duplicates++
			continue
		}
		seen[key] = true
		if count >= promptUserMaxItems {
			if res.Created == 0 {
				return nil, ErrPromptTooMany
			}
			break
		}
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return nil, fmt.Errorf("generate prompt id: %w", err)
		}
		owner := userID
		item := &PromptItem{SourceID: PromptSourceUser, ExternalID: hex.EncodeToString(buf[:]), OwnerUserID: &owner, ReferenceImageURLs: []string{}, SourceTags: []string{}}
		s.applyUserInput(item, in)
		item.FolderID = folderRef(in.FolderID)
		setPromptSharing(item, false, true)
		if err := s.repo.Insert(ctx, item); err != nil {
			return nil, err
		}
		count++
		res.Created++
		item.Mine = true
		res.Items = append(res.Items, item.PublicView())
	}
	return res, nil
}
