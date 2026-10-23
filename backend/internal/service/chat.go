package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// AI 对话 (/chat): a web chat page for users who want a ChatGPT-style conversation instead of a
// client app. Every model call goes through this site's own gateway on a key the user picked, so
// usage, billing and the key's limits work as for any API call. Conversations are kept on the
// server; pictures (uploaded, or drawn in 画图 mode) are files under the chat media dir.

const (
	chatTitleMax          = 40
	chatSystemPromptMax   = 8000
	chatContentMax        = 30000 // runes in one user message
	chatFilesPerMessage   = 4
	chatMessagesPerConv   = 400
	chatListLimit         = 300
	chatUploadsPerDay     = 200
	chatContextMessages   = 40    // earlier messages sent with a question
	chatContextRunes      = 80000 // runes of earlier text sent with a question
	chatContextImageTurns = 4     // user messages whose pictures are sent again
	chatTextTimeout       = 10 * time.Minute
	chatImageTimeout      = 6 * time.Minute
	chatModelsCacheTTL    = 5 * time.Minute
	chatAnthropicMaxOut   = 16384
	chatImageModel        = "gpt-image-2"
	chatUserAgent         = "hivegpt-chat/1"
	chatUnsentFileTTL     = 24 * time.Hour
)

const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
	ChatKindText      = "text"
	ChatKindImage     = "image"
	ChatStatusPending = "pending"
	ChatStatusDone    = "done"
	ChatStatusStopped = "stopped"
	ChatStatusError   = "error"
)

var (
	ErrChatNotFound      = infraerrors.NotFound("CHAT_NOT_FOUND", "对话不存在或已删除（Conversation not found）")
	ErrChatMessageGone   = infraerrors.NotFound("CHAT_MESSAGE_NOT_FOUND", "这条消息不存在，请刷新页面（Message not found）")
	ErrChatKeyInvalid    = infraerrors.BadRequest("CHAT_KEY_INVALID", "这个 Key 不可用，请换一个启用中的 Key，或到「API 密钥」新建一个（Key unavailable）")
	ErrChatKeyNoGroup    = infraerrors.BadRequest("CHAT_KEY_NO_GROUP", "这个 Key 还没有分组，请到「API 密钥」给它选一个分组（Key has no group）")
	ErrChatKeyCodeOnly   = infraerrors.BadRequest("CHAT_KEY_CODE_ONLY", "这个 Key 的分组只能在 Claude Code 里用，网页对话请换一个其它分组的 Key（Group is Claude Code only）")
	ErrChatModel         = infraerrors.BadRequest("CHAT_MODEL_INVALID", "请先选择一个模型（Model required）")
	ErrChatEmpty         = infraerrors.BadRequest("CHAT_EMPTY", "请输入内容（Message is empty）")
	ErrChatTooLong       = infraerrors.BadRequest("CHAT_TOO_LONG", fmt.Sprintf("一次最多发送 %d 字，请分几次发（Message too long）", chatContentMax))
	ErrChatTooManyFiles  = infraerrors.BadRequest("CHAT_TOO_MANY_FILES", fmt.Sprintf("一次最多带 %d 张图片（Too many images）", chatFilesPerMessage))
	ErrChatFileGone      = infraerrors.BadRequest("CHAT_FILE_INVALID", "图片已失效，请重新上传（Image not found）")
	ErrChatConvFull      = infraerrors.BadRequest("CHAT_CONVERSATION_FULL", "这个对话太长了，请新建一个对话继续（Conversation too long）")
	ErrChatNoDraw        = infraerrors.BadRequest("CHAT_DRAW_UNSUPPORTED", "画图要用 GPT 分组（OpenAI）的 Key，请在上方换一个 Key（Images need an OpenAI group key）")
	ErrChatImageSize     = infraerrors.BadRequest("CHAT_IMAGE_SIZE", "图片尺寸不支持（Invalid image size）")
	ErrChatUploadLimit   = infraerrors.TooManyRequests("CHAT_UPLOAD_LIMIT", fmt.Sprintf("今天上传的图片已达 %d 张，明天再来（Daily upload limit）", chatUploadsPerDay))
	ErrChatNothingToRedo = infraerrors.BadRequest("CHAT_NOTHING_TO_REGENERATE", "没有可以重新生成的消息（Nothing to regenerate）")
	ErrChatBusy          = infraerrors.Conflict("CHAT_BUSY", "上一条还在生成，请等它结束或先点停止（A reply is still running）")
)

var (
	chatModelRe    = regexp.MustCompile(`^[A-Za-z0-9._:/@+\-\[\]]{1,100}$`)
	chatImageSizes = map[string]bool{"1024x1024": true, "1536x1024": true, "1024x1536": true, "auto": true}
	// Ids in a /v1/models list that are not chat models.
	chatNonChatModelRe = regexp.MustCompile(`(?i)(image|embedding|tts|whisper|dall-e|sora|moderation|transcribe|realtime|audio|veo|imagen|-video|search-preview)`)
)

type ChatConversation struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"-"`
	Title        string    `json:"title"`
	SystemPrompt string    `json:"system_prompt"`
	APIKeyID     *int64    `json:"api_key_id"`
	Model        string    `json:"model"`
	Pinned       bool      `json:"pinned"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID               int64     `json:"id"`
	ConversationID   int64     `json:"conversation_id"`
	Role             string    `json:"role"`
	Kind             string    `json:"kind"`
	Content          string    `json:"content"`
	Reasoning        string    `json:"reasoning,omitempty"`
	Files            []string  `json:"files"`
	Model            string    `json:"model,omitempty"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	PromptTokens     int       `json:"prompt_tokens,omitempty"`
	CompletionTokens int       `json:"completion_tokens,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type ChatFile struct {
	Name      string
	UserID    int64
	MessageID *int64
	// ConversationID is the conversation of the message the file is attached to (0 if unsent).
	ConversationID int64
	Mime           string
	Size           int
	Width          int
	Height         int
	CreatedAt      time.Time
}

type ChatRepository interface {
	ListConversations(ctx context.Context, userID int64, query string, limit int) ([]ChatConversation, error)
	// GetConversation returns nil when the conversation is not the user's.
	GetConversation(ctx context.Context, userID, id int64) (*ChatConversation, error)
	CreateConversation(ctx context.Context, c *ChatConversation) error
	// UpdateConversation writes title, system_prompt, api_key_id, model and pinned; touch also moves updated_at.
	UpdateConversation(ctx context.Context, c *ChatConversation, touch bool) error
	// DeleteConversation returns the picture files that went with it.
	DeleteConversation(ctx context.Context, userID, id int64) (bool, []string, error)
	ListMessages(ctx context.Context, conversationID int64) ([]ChatMessage, error)
	CountMessages(ctx context.Context, conversationID int64) (int, error)
	CreateMessage(ctx context.Context, m *ChatMessage) error
	UpdateMessage(ctx context.Context, m *ChatMessage) error
	// DeleteMessagesFrom deletes the conversation's messages with id >= fromID. Files named in keep are
	// detached first so they survive; the names of the files that went are returned.
	DeleteMessagesFrom(ctx context.Context, conversationID, fromID int64, keep []string) ([]string, error)
	// FailPending marks replies left pending by a previous process as failed.
	FailPending(ctx context.Context, msg string) (int64, error)

	CreateFile(ctx context.Context, f *ChatFile) error
	GetFile(ctx context.Context, name string) (*ChatFile, error)
	AttachFiles(ctx context.Context, messageID int64, names []string) error
	CountFilesSince(ctx context.Context, userID int64, since time.Time) (int, error)
	// DeleteUnsentFiles removes files never sent and uploaded before the cut-off, returning their names.
	DeleteUnsentFiles(ctx context.Context, before time.Time) ([]string, error)
	// KnownFiles returns which of the names have a row.
	KnownFiles(ctx context.Context, names []string) (map[string]bool, error)
}

type ChatService struct {
	repo       ChatRepository
	keys       APIKeyRepository
	files      *ChatFileStore
	gatewayURL string
	client     *http.Client
	now        func() time.Time

	mu      sync.Mutex
	running map[int64]bool // conversations with a reply being written
	models  map[int64]chatModelsEntry
}

type chatModelsEntry struct {
	models []string
	at     time.Time
}

func NewChatService(repo ChatRepository, keys APIKeyRepository, files *ChatFileStore, gatewayURL string) *ChatService {
	return &ChatService{repo: repo, keys: keys, files: files, gatewayURL: strings.TrimRight(gatewayURL, "/"),
		client: &http.Client{}, now: time.Now, running: map[int64]bool{}, models: map[int64]chatModelsEntry{}}
}

func (s *ChatService) Files() *ChatFileStore { return s.files }

// Keys -------------------------------------------------------------------------------------------

// ChatKeyOption is one of the user's keys offered on the page (no secret).
type ChatKeyOption struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Group    string  `json:"group"`
	Platform string  `json:"platform"`
	Rate     float64 `json:"rate"`
	// CanDraw: the group is OpenAI, so 画图 (gpt-image-2) works on it.
	CanDraw bool `json:"can_draw"`
}

func chatKeyUsable(k *APIKey) error {
	if k == nil || !k.IsActive() || k.Key == "" {
		return ErrChatKeyInvalid
	}
	if k.Group == nil {
		return ErrChatKeyNoGroup
	}
	if k.Group.ClaudeCodeOnly {
		return ErrChatKeyCodeOnly
	}
	return nil
}

func chatCanDraw(k *APIKey) bool { return k.Group != nil && k.Group.Platform == PlatformOpenAI }

func (s *ChatService) Keys(ctx context.Context, userID int64) ([]ChatKeyOption, error) {
	out := []ChatKeyOption{}
	keys, _, err := s.keys.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	for i := range keys {
		k := &keys[i]
		if chatKeyUsable(k) != nil {
			continue
		}
		out = append(out, ChatKeyOption{ID: k.ID, Name: k.Name, Group: k.Group.Name, Platform: k.Group.Platform,
			Rate: k.Group.RateMultiplier, CanDraw: chatCanDraw(k)})
	}
	return out, nil
}

func (s *ChatService) ownKey(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if keyID <= 0 {
		return nil, ErrChatKeyInvalid
	}
	k, err := s.keys.GetByID(ctx, keyID)
	if err != nil || k == nil || k.UserID != userID {
		return nil, ErrChatKeyInvalid
	}
	if err := chatKeyUsable(k); err != nil {
		return nil, err
	}
	return k, nil
}

// ChatModels is what the page offers for a key: the group's chat models, and whether it can draw.
type ChatModels struct {
	Models  []string `json:"models"`
	CanDraw bool     `json:"can_draw"`
}

// Models lists the chat models of the key's group, as the gateway's /v1/models reports them.
func (s *ChatService) Models(ctx context.Context, userID, keyID int64) (*ChatModels, error) {
	k, err := s.ownKey(ctx, userID, keyID)
	if err != nil {
		return nil, err
	}
	out := &ChatModels{CanDraw: chatCanDraw(k)}
	s.mu.Lock()
	if e, ok := s.models[keyID]; ok && s.now().Sub(e.at) < chatModelsCacheTTL {
		s.mu.Unlock()
		out.Models = e.models
		return out, nil
	}
	s.mu.Unlock()
	models, err := s.fetchModels(ctx, k.Key)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.models[keyID] = chatModelsEntry{models: models, at: s.now()}
	s.mu.Unlock()
	out.Models = models
	return out, nil
}

func (s *ChatService) fetchModels(ctx context.Context, key string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.gatewayURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", chatUserAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, chatFailed("读取模型列表超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, chatUpstreamFailed(raw, resp.StatusCode)
	}
	return chatModelIDs(raw), nil
}

// chatModelIDs keeps the chat models of a /v1/models reply, deduplicated, in the listed order.
func chatModelIDs(raw []byte) []string {
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	out := []string{}
	if json.Unmarshal(raw, &list) != nil {
		return out
	}
	seen := map[string]bool{}
	for _, m := range list.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] || !chatModelRe.MatchString(id) || chatNonChatModelRe.MatchString(id) {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func chatFailed(detail string) error {
	return infraerrors.New(http.StatusBadGateway, "CHAT_FAILED", fmt.Sprintf("%s，请稍后再试（Chat failed）", detail))
}

// chatUpstreamFailed reports an error reply of the gateway. Its own rejections (余额不足, 分组不支持
// 这个模型 …) are already Chinese and say what to do, so they are shown as they are.
func chatUpstreamFailed(raw []byte, status int) error {
	msg := upstreamErrorText(raw, status)
	for _, r := range msg {
		if unicode.Is(unicode.Han, r) {
			return infraerrors.New(http.StatusBadGateway, "CHAT_FAILED", msg)
		}
	}
	return chatFailed("模型服务出错：" + msg)
}

// Conversations ----------------------------------------------------------------------------------

func (s *ChatService) List(ctx context.Context, userID int64, query string) ([]ChatConversation, error) {
	q := strings.TrimSpace(query)
	if utf8.RuneCountInString(q) > 50 {
		q = string([]rune(q)[:50])
	}
	return s.repo.ListConversations(ctx, userID, q, chatListLimit)
}

// ChatConversationInput creates or edits a conversation; nil fields are left as they are.
type ChatConversationInput struct {
	Title        *string `json:"title"`
	SystemPrompt *string `json:"system_prompt"`
	Pinned       *bool   `json:"pinned"`
}

func chatCut(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func (in ChatConversationInput) apply(c *ChatConversation) error {
	if in.Title != nil {
		c.Title = chatCut(strings.ReplaceAll(*in.Title, "\n", " "), chatTitleMax)
	}
	if in.SystemPrompt != nil {
		if utf8.RuneCountInString(*in.SystemPrompt) > chatSystemPromptMax {
			return infraerrors.BadRequest("CHAT_SYSTEM_PROMPT_TOO_LONG", fmt.Sprintf("系统提示词最多 %d 字（System prompt too long）", chatSystemPromptMax))
		}
		c.SystemPrompt = strings.TrimSpace(*in.SystemPrompt)
	}
	if in.Pinned != nil {
		c.Pinned = *in.Pinned
	}
	return nil
}

func (s *ChatService) Create(ctx context.Context, userID int64, in ChatConversationInput) (*ChatConversation, error) {
	c := &ChatConversation{UserID: userID}
	if err := in.apply(c); err != nil {
		return nil, err
	}
	if err := s.repo.CreateConversation(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ChatService) get(ctx context.Context, userID, id int64) (*ChatConversation, error) {
	c, err := s.repo.GetConversation(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrChatNotFound
	}
	return c, nil
}

func (s *ChatService) Update(ctx context.Context, userID, id int64, in ChatConversationInput) (*ChatConversation, error) {
	c, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := in.apply(c); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateConversation(ctx, c, false); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ChatService) Delete(ctx context.Context, userID, id int64) error {
	ok, files, err := s.repo.DeleteConversation(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrChatNotFound
	}
	s.files.Remove(files...)
	return nil
}

// ChatConversationDetail is a conversation with its messages.
type ChatConversationDetail struct {
	Conversation *ChatConversation `json:"conversation"`
	Messages     []ChatMessage     `json:"messages"`
	Running      bool              `json:"running"`
}

func (s *ChatService) Get(ctx context.Context, userID, id int64) (*ChatConversationDetail, error) {
	c, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	msgs, err := s.repo.ListMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	running := s.running[id]
	s.mu.Unlock()
	return &ChatConversationDetail{Conversation: c, Messages: msgs, Running: running}, nil
}

// Files ------------------------------------------------------------------------------------------

// ChatUploaded is a stored picture, ready to be sent with a message.
type ChatUploaded struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (s *ChatService) Upload(ctx context.Context, userID int64, data []byte) (*ChatUploaded, error) {
	n, err := s.repo.CountFilesSince(ctx, userID, learnDayStart(s.now()))
	if err != nil {
		return nil, err
	}
	if n >= chatUploadsPerDay {
		return nil, ErrChatUploadLimit
	}
	f, err := s.saveFile(ctx, userID, data)
	if err != nil {
		return nil, err
	}
	return &ChatUploaded{Name: f.Name, Width: f.Width, Height: f.Height}, nil
}

func (s *ChatService) saveFile(ctx context.Context, userID int64, data []byte) (*ChatFile, error) {
	stored, err := s.files.Save(data)
	if err != nil {
		return nil, err
	}
	f := &ChatFile{Name: stored.Name, UserID: userID, Mime: stored.Mime, Size: len(data), Width: stored.Width, Height: stored.Height}
	if err := s.repo.CreateFile(ctx, f); err != nil {
		s.files.Remove(stored.Name)
		return nil, err
	}
	return f, nil
}

// File returns a picture of the user's: the original, or with preview a smaller JPEG for the page.
func (s *ChatService) File(ctx context.Context, userID int64, name string, preview bool) ([]byte, string, error) {
	f, err := s.repo.GetFile(ctx, name)
	if err != nil {
		return nil, "", err
	}
	if f == nil || f.UserID != userID {
		return nil, "", ErrChatFileGone
	}
	if preview {
		data, mime, err := s.files.Preview(name)
		if err != nil {
			return nil, "", ErrChatFileGone
		}
		return data, mime, nil
	}
	data, err := s.files.Read(name)
	if err != nil {
		return nil, "", ErrChatFileGone
	}
	return data, f.Mime, nil
}

// attachable resolves the pictures sent with a message: the user's unsent uploads are used as they
// are; a picture already in one of the user's conversations (a drawn image sent back to edit it) is
// copied, so deleting the original later doesn't take it away.
func (s *ChatService) attachable(ctx context.Context, userID int64, names []string, keep map[string]bool) ([]string, error) {
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		f, err := s.repo.GetFile(ctx, name)
		if err != nil {
			return nil, err
		}
		if f == nil || f.UserID != userID {
			return nil, ErrChatFileGone
		}
		if f.MessageID == nil || keep[name] {
			out = append(out, name)
			continue
		}
		data, err := s.files.Read(name)
		if err != nil {
			return nil, ErrChatFileGone
		}
		cp, err := s.saveFile(ctx, userID, data)
		if err != nil {
			return nil, err
		}
		out = append(out, cp.Name)
	}
	return out, nil
}

// Sending ----------------------------------------------------------------------------------------

// ChatSendInput is one turn. Mode "image" draws a picture from the text (with the attached pictures
// as references). Regenerate answers the last question again; EditID replaces that question of the
// user's (and everything after it) with this one.
type ChatSendInput struct {
	KeyID      int64    `json:"key_id"`
	Model      string   `json:"model"`
	Mode       string   `json:"mode"`
	Content    string   `json:"content"`
	Files      []string `json:"files"`
	ImageSize  string   `json:"image_size"`
	Regenerate bool     `json:"regenerate"`
	EditID     int64    `json:"edit_id"`
}

// ChatEvent is one server-sent event of a turn.
type ChatEvent struct {
	User         *ChatMessage      `json:"user,omitempty"`
	Assistant    *ChatMessage      `json:"assistant,omitempty"`
	Conversation *ChatConversation `json:"conversation,omitempty"`
	Delta        string            `json:"delta,omitempty"`
	Reasoning    string            `json:"reasoning,omitempty"`
	Done         *ChatMessage      `json:"done,omitempty"`
	Error        string            `json:"error,omitempty"`
}

func (s *ChatService) claim(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[id] {
		return false
	}
	s.running[id] = true
	return true
}

func (s *ChatService) release(id int64) {
	s.mu.Lock()
	delete(s.running, id)
	s.mu.Unlock()
}

// Send runs one turn. Errors before the reply starts are returned (the handler answers with JSON);
// after that they arrive as an Error event and are saved on the reply. In text mode the reply stops
// when ctx ends (the user pressed 停止 or left), keeping what arrived; a picture is finished and
// saved even if the page was closed, since it is billed either way.
func (s *ChatService) Send(ctx context.Context, userID, convID int64, in ChatSendInput, emit func(ChatEvent) error) error {
	conv, err := s.get(ctx, userID, convID)
	if err != nil {
		return err
	}
	mode := in.Mode
	if mode != ChatKindImage {
		mode = ChatKindText
	}
	key, err := s.ownKey(ctx, userID, in.KeyID)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(in.Model)
	if mode == ChatKindImage {
		if !chatCanDraw(key) {
			return ErrChatNoDraw
		}
		if in.ImageSize == "" {
			in.ImageSize = "1024x1024"
		}
		if !chatImageSizes[in.ImageSize] {
			return ErrChatImageSize
		}
		model = chatImageModel
	} else if !chatModelRe.MatchString(model) {
		return ErrChatModel
	}
	if !s.claim(convID) {
		return ErrChatBusy
	}
	defer s.release(convID)

	user, err := s.prepareQuestion(ctx, userID, conv, mode, in)
	if err != nil {
		return err
	}
	if user.Kind == ChatKindImage && mode != ChatKindImage {
		mode = ChatKindImage
		model = chatImageModel
		if !chatCanDraw(key) {
			return ErrChatNoDraw
		}
	}
	if conv.Title == "" {
		conv.Title = chatAutoTitle(user.Content, user.Kind)
	}
	conv.APIKeyID = &key.ID
	if mode == ChatKindText {
		conv.Model = model
	}
	if err := s.repo.UpdateConversation(ctx, conv, true); err != nil {
		return err
	}
	reply := &ChatMessage{ConversationID: convID, Role: ChatRoleAssistant, Kind: mode, Files: []string{}, Model: model, Status: ChatStatusPending}
	if err := s.repo.CreateMessage(ctx, reply); err != nil {
		return err
	}
	// From here on the turn is saved; failures are reported on the reply.
	_ = emit(ChatEvent{User: user, Assistant: reply, Conversation: conv})
	if mode == ChatKindImage {
		s.draw(ctx, key.Key, conv, user, reply, in.ImageSize, emit)
	} else {
		s.answer(ctx, key, conv, reply, emit)
	}
	return nil
}

// prepareQuestion saves the question of this turn (or finds it, for regenerate) and clears what
// came after it.
func (s *ChatService) prepareQuestion(ctx context.Context, userID int64, conv *ChatConversation, mode string, in ChatSendInput) (*ChatMessage, error) {
	msgs, err := s.repo.ListMessages(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	if in.Regenerate {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role != ChatRoleUser {
				continue
			}
			q := msgs[i]
			if i+1 < len(msgs) {
				gone, err := s.repo.DeleteMessagesFrom(ctx, conv.ID, msgs[i+1].ID, nil)
				if err != nil {
					return nil, err
				}
				s.files.Remove(gone...)
			}
			return &q, nil
		}
		return nil, ErrChatNothingToRedo
	}

	content := strings.TrimSpace(in.Content)
	if utf8.RuneCountInString(content) > chatContentMax {
		return nil, ErrChatTooLong
	}
	if len(in.Files) > chatFilesPerMessage {
		return nil, ErrChatTooManyFiles
	}
	if content == "" && (mode == ChatKindImage || len(in.Files) == 0) {
		return nil, ErrChatEmpty
	}
	keep := map[string]bool{}
	if in.EditID > 0 {
		var edited *ChatMessage
		for i := range msgs {
			if msgs[i].ID == in.EditID && msgs[i].Role == ChatRoleUser {
				edited = &msgs[i]
			}
		}
		if edited == nil {
			return nil, ErrChatMessageGone
		}
		// Pictures of the edited question that are sent again stay as they are.
		for _, f := range edited.Files {
			keep[f] = true
		}
		var kept []string
		for _, f := range in.Files {
			if keep[f] {
				kept = append(kept, f)
			}
		}
		gone, err := s.repo.DeleteMessagesFrom(ctx, conv.ID, edited.ID, kept)
		if err != nil {
			return nil, err
		}
		s.files.Remove(gone...)
		keep = map[string]bool{}
		for _, f := range kept {
			keep[f] = true
		}
	} else if len(msgs)+2 > chatMessagesPerConv {
		return nil, ErrChatConvFull
	}
	files, err := s.attachable(ctx, userID, in.Files, keep)
	if err != nil {
		return nil, err
	}
	q := &ChatMessage{ConversationID: conv.ID, Role: ChatRoleUser, Kind: mode, Content: content, Files: files, Status: ChatStatusDone}
	if err := s.repo.CreateMessage(ctx, q); err != nil {
		return nil, err
	}
	if len(files) > 0 {
		if err := s.repo.AttachFiles(ctx, q.ID, files); err != nil {
			return nil, err
		}
	}
	return q, nil
}

func chatAutoTitle(content, kind string) string {
	t := strings.Join(strings.Fields(content), " ")
	if t == "" {
		t = "图片对话"
	}
	if kind == ChatKindImage {
		t = "画图：" + t
	}
	return chatCut(t, 24)
}

// finish saves the reply with a context of its own (the request's may be gone).
func (s *ChatService) finish(reply *ChatMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.repo.UpdateMessage(ctx, reply); err != nil {
		logger.LegacyPrintf("service.chat", "save reply failed: message_id=%d err=%v", reply.ID, err)
	}
}

func (s *ChatService) answer(ctx context.Context, key *APIKey, conv *ChatConversation, reply *ChatMessage, emit func(ChatEvent) error) {
	msgs, err := s.repo.ListMessages(ctx, conv.ID)
	if err != nil {
		reply.Status, reply.Error = ChatStatusError, "读取对话失败，请重试"
		s.finish(reply)
		_ = emit(ChatEvent{Error: reply.Error, Done: reply})
		return
	}
	body := map[string]any{
		"model":          reply.Model,
		"messages":       s.buildContext(conv, msgs, reply.ID),
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if key.Group != nil && key.Group.Platform == PlatformAnthropic {
		body["max_tokens"] = chatAnthropicMaxOut
	}
	runCtx, cancel := context.WithTimeout(ctx, chatTextTimeout)
	defer cancel()
	res, err := s.stream(runCtx, key.Key, body, func(delta, reasoning string) error {
		reply.Content += delta
		reply.Reasoning += reasoning
		return emit(ChatEvent{Delta: delta, Reasoning: reasoning})
	})
	if res != nil {
		if res.model != "" {
			reply.Model = res.model
		}
		reply.PromptTokens, reply.CompletionTokens = res.promptTokens, res.completionTokens
	}
	switch {
	case err == nil:
		reply.Status = ChatStatusDone
	case ctx.Err() != nil:
		// 停止, or the page went away: keep what arrived.
		reply.Status = ChatStatusStopped
	default:
		reply.Status = ChatStatusError
		reply.Error = chatErrText(err)
	}
	s.finish(reply)
	if reply.Status == ChatStatusError {
		_ = emit(ChatEvent{Error: reply.Error, Done: reply})
		return
	}
	_ = emit(ChatEvent{Done: reply})
}

func chatErrText(err error) string {
	var ae *infraerrors.ApplicationError
	if errors.As(err, &ae) && ae.Message != "" {
		return ae.Message
	}
	return "生成失败，请重试（Generation failed）"
}

// buildContext turns the conversation before the reply into chat-completions messages: the system
// prompt, then as many recent messages as fit. Pictures are sent for the last few questions that had
// some; older ones are mentioned as text.
func (s *ChatService) buildContext(conv *ChatConversation, msgs []ChatMessage, replyID int64) []map[string]any {
	system := "当前日期：" + s.now().In(learnDayZone).Format("2006-01-02") + "。回答使用 Markdown；用户用中文提问时用中文回答。"
	if conv.SystemPrompt != "" {
		system = conv.SystemPrompt + "\n\n" + system
	}
	var history []ChatMessage
	for _, m := range msgs {
		if m.ID == replyID {
			break
		}
		if m.Role == ChatRoleAssistant && (m.Status == ChatStatusError || m.Status == ChatStatusPending) {
			continue
		}
		history = append(history, m)
	}
	// Walk back from the newest message until the budget is spent.
	start, runes := len(history), 0
	for start > 0 && len(history)-start < chatContextMessages {
		r := utf8.RuneCountInString(history[start-1].Content)
		if runes+r > chatContextRunes && start < len(history) {
			break
		}
		runes += r
		start--
	}
	history = history[start:]
	// A conversation must not start with the model's turn.
	for len(history) > 0 && history[0].Role != ChatRoleUser {
		history = history[1:]
	}
	withImages := map[int64]bool{}
	for i := len(history) - 1; i >= 0 && len(withImages) < chatContextImageTurns; i-- {
		if history[i].Role == ChatRoleUser && len(history[i].Files) > 0 {
			withImages[history[i].ID] = true
		}
	}
	out := []map[string]any{{"role": "system", "content": system}}
	for _, m := range history {
		text := m.Content
		if m.Kind == ChatKindImage {
			if m.Role == ChatRoleUser {
				text = "（请求画图）" + text
			} else {
				text = "（已按要求画好图片）"
				if m.Content != "" {
					text += m.Content
				}
			}
		}
		if m.Role == ChatRoleAssistant {
			out = append(out, map[string]any{"role": "assistant", "content": text})
			continue
		}
		if !withImages[m.ID] {
			if len(m.Files) > 0 {
				text = strings.TrimSpace(fmt.Sprintf("[附了 %d 张图片] %s", len(m.Files), text))
			}
			out = append(out, map[string]any{"role": "user", "content": text})
			continue
		}
		parts := []map[string]any{}
		if text != "" {
			parts = append(parts, map[string]any{"type": "text", "text": text})
		}
		for _, f := range m.Files {
			if u := s.files.DataURL(f); u != "" {
				parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": u}})
			}
		}
		if len(parts) == 0 {
			parts = append(parts, map[string]any{"type": "text", "text": "（图片已失效）"})
		}
		out = append(out, map[string]any{"role": "user", "content": parts})
	}
	return out
}

func (s *ChatService) draw(ctx context.Context, key string, conv *ChatConversation, user, reply *ChatMessage, size string, emit func(ChatEvent) error) {
	// The picture is finished and saved even if the page goes away.
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), chatImageTimeout)
	defer cancel()
	var refs []chatImageRef
	for _, f := range user.Files {
		data, err := s.files.Read(f)
		if err != nil {
			continue
		}
		refs = append(refs, chatImageRef{name: f, data: data})
	}
	data, err := s.gatewayDraw(runCtx, key, user.Content, size, refs)
	if err == nil {
		var f *ChatFile
		f, err = s.saveFile(runCtx, conv.UserID, data)
		if err == nil {
			reply.Files = []string{f.Name}
			err = s.repo.AttachFiles(runCtx, reply.ID, reply.Files)
		}
	}
	if err != nil {
		reply.Status, reply.Error = ChatStatusError, chatErrText(err)
		s.finish(reply)
		_ = emit(ChatEvent{Error: reply.Error, Done: reply})
		return
	}
	reply.Status = ChatStatusDone
	s.finish(reply)
	_ = emit(ChatEvent{Done: reply})
}

// RecoverInterrupted marks replies a previous process left pending.
func (s *ChatService) RecoverInterrupted(ctx context.Context) {
	if n, err := s.repo.FailPending(ctx, "服务重启，这条回答中断了，请点重新生成（Interrupted）"); err != nil {
		logger.LegacyPrintf("service.chat", "recover pending replies failed: %v", err)
	} else if n > 0 {
		logger.LegacyPrintf("service.chat", "marked %d interrupted replies", n)
	}
}

// Sweep removes uploads never sent within a day, and files left on disk without a row (deleted
// conversations or accounts).
func (s *ChatService) Sweep(ctx context.Context) {
	names, err := s.repo.DeleteUnsentFiles(ctx, s.now().Add(-chatUnsentFileTTL))
	if err != nil {
		logger.LegacyPrintf("service.chat", "sweep unsent files failed: %v", err)
	}
	s.files.Remove(names...)
	orphans := s.files.OlderThan(s.now().Add(-time.Hour))
	sort.Strings(orphans)
	for len(orphans) > 0 {
		n := min(len(orphans), 500)
		batch := orphans[:n]
		orphans = orphans[n:]
		known, err := s.repo.KnownFiles(ctx, batch)
		if err != nil {
			logger.LegacyPrintf("service.chat", "sweep orphan files failed: %v", err)
			return
		}
		for _, name := range batch {
			if !known[name] {
				s.files.Remove(name)
			}
		}
	}
}

// StartSweeper runs Sweep hourly.
func (s *ChatService) StartSweeper() {
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for range t.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			s.Sweep(ctx)
			cancel()
		}
	}()
}
