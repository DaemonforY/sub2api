package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContentEngineHandler serves 后台「内容运营」 (/api/v1/admin/content). The signed-in admin is the
// owner: settings, topics, articles and rules are theirs.
type ContentEngineHandler struct {
	svc *service.ContentEngineService
}

func NewContentEngineHandler(svc *service.ContentEngineService) *ContentEngineHandler {
	return &ContentEngineHandler{svc: svc}
}

// Suggestions and 立即运行 make a model call; they finish even when the page is closed.
const contentSlowTimeout = 4 * time.Minute

func contentOwner(c *gin.Context) (int64, bool) {
	subject, ok := requireAuth(c)
	return subject.UserID, ok
}

func contentID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "参数格式不正确（Invalid id）")
		return 0, false
	}
	return id, true
}

// contentOwnerID reads both the owner and the :id path parameter.
func contentOwnerID(c *gin.Context) (int64, int64, bool) {
	owner, ok := contentOwner(c)
	if !ok {
		return 0, 0, false
	}
	id, ok := contentID(c)
	return owner, id, ok
}

// Overview GET /admin/content/overview
func (h *ContentEngineHandler) Overview(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Overview(c.Request.Context(), owner)
		learnReply(c, out, err)
	}
}

// Settings GET /admin/content/settings
func (h *ContentEngineHandler) Settings(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Settings(c.Request.Context(), owner)
		learnReply(c, out, err)
	}
}

// SaveSettings PUT /admin/content/settings
func (h *ContentEngineHandler) SaveSettings(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in service.ContentSettings
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), owner, in)
	learnReply(c, out, err)
}

// Keys GET /admin/content/keys — the admin's own GPT keys.
func (h *ContentEngineHandler) Keys(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Keys(c.Request.Context(), owner)
		learnReply(c, out, err)
	}
}

// TestNotify POST /admin/content/notify/test
func (h *ContentEngineHandler) TestNotify(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.TestNotify(c.Request.Context(), owner))
	}
}

// RunNow POST /admin/content/run — the daily run, now (tops up topics if needed).
func (h *ContentEngineHandler) RunNow(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), contentSlowTimeout)
	defer cancel()
	learnReply(c, gin.H{"ok": true}, h.svc.RunDaily(ctx, owner))
}

// Topics GET /admin/content/topics?status=pending|used|dismissed|all
func (h *ContentEngineHandler) Topics(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Topics(c.Request.Context(), owner, c.Query("status"))
		learnReply(c, out, err)
	}
}

// CreateTopic POST /admin/content/topics
func (h *ContentEngineHandler) CreateTopic(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in service.ContentTopicInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.CreateTopic(c.Request.Context(), owner, in)
	learnReply(c, out, err)
}

// SuggestTopics POST /admin/content/topics/suggest
func (h *ContentEngineHandler) SuggestTopics(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), contentSlowTimeout)
	defer cancel()
	out, err := h.svc.SuggestTopics(ctx, owner)
	learnReply(c, out, err)
}

// UpdateTopic PUT /admin/content/topics/:id
func (h *ContentEngineHandler) UpdateTopic(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentTopicInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.UpdateTopic(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// SetTopicStatus POST /admin/content/topics/:id/status {status: pending|dismissed}
func (h *ContentEngineHandler) SetTopicStatus(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SetTopicStatus(c.Request.Context(), owner, id, in.Status)
	learnReply(c, out, err)
}

// StartTopic POST /admin/content/topics/:id/start
func (h *ContentEngineHandler) StartTopic(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.StartTopic(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// DeleteTopic DELETE /admin/content/topics/:id
func (h *ContentEngineHandler) DeleteTopic(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.DeleteTopic(c.Request.Context(), owner, id))
	}
}

// Items GET /admin/content/items?status=
func (h *ContentEngineHandler) Items(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Items(c.Request.Context(), owner, c.Query("status"))
		learnReply(c, out, err)
	}
}

// Item GET /admin/content/items/:id
func (h *ContentEngineHandler) Item(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.Item(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// Retry POST /admin/content/items/:id/retry
func (h *ContentEngineHandler) Retry(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.Retry(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// Cancel POST /admin/content/items/:id/cancel
func (h *ContentEngineHandler) Cancel(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.Cancel(c.Request.Context(), owner, id))
	}
}

// Reject POST /admin/content/items/:id/reject {reason, as_rule}
func (h *ContentEngineHandler) Reject(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
		AsRule bool   `json:"as_rule"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Reject(c.Request.Context(), owner, id, in.Reason, in.AsRule)
	learnReply(c, out, err)
}

// DeleteItem DELETE /admin/content/items/:id
func (h *ContentEngineHandler) DeleteItem(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.DeleteItem(c.Request.Context(), owner, id))
	}
}

// UpdateVersion PUT /admin/content/versions/:id
func (h *ContentEngineHandler) UpdateVersion(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentVariantInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.UpdateVariant(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// ApproveVersion POST /admin/content/versions/:id/approve
func (h *ContentEngineHandler) ApproveVersion(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.ApproveVariant(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// SkipVersion POST /admin/content/versions/:id/skip
func (h *ContentEngineHandler) SkipVersion(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.SkipVariant(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// PublishVersion POST /admin/content/versions/:id/publish {post_url}
func (h *ContentEngineHandler) PublishVersion(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		PostURL string `json:"post_url"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.PublishVariant(c.Request.Context(), owner, id, in.PostURL)
	learnReply(c, out, err)
}

// RegenerateVersion POST /admin/content/versions/:id/regenerate {feedback}
func (h *ContentEngineHandler) RegenerateVersion(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Feedback string `json:"feedback"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.RegenerateVariant(c.Request.Context(), owner, id, in.Feedback)
	learnReply(c, out, err)
}

// Rules GET /admin/content/rules
func (h *ContentEngineHandler) Rules(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Rules(c.Request.Context(), owner)
		learnReply(c, out, err)
	}
}

// CreateRule POST /admin/content/rules {rule}
func (h *ContentEngineHandler) CreateRule(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in struct {
		Rule string `json:"rule"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.CreateRule(c.Request.Context(), owner, in.Rule)
	learnReply(c, out, err)
}

// UpdateRule PUT /admin/content/rules/:id {enabled}
func (h *ContentEngineHandler) UpdateRule(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !bindLearn(c, &in) {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.SetRuleEnabled(c.Request.Context(), owner, id, in.Enabled))
}

// DeleteRule DELETE /admin/content/rules/:id
func (h *ContentEngineHandler) DeleteRule(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.DeleteRule(c.Request.Context(), owner, id))
	}
}

// Stats GET /admin/content/stats?days=
func (h *ContentEngineHandler) Stats(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil {
		days = 30
	}
	out, err := h.svc.Stats(c.Request.Context(), owner, days)
	learnReply(c, out, err)
}

// --- 写作台 ------------------------------------------------------------------------------------

// Persona GET /admin/content/persona
func (h *ContentEngineHandler) Persona(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.Persona(c.Request.Context(), owner)
		learnReply(c, out, err)
	}
}

// SavePersona PUT /admin/content/persona
func (h *ContentEngineHandler) SavePersona(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in service.ContentPersona
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SavePersona(c.Request.Context(), owner, in)
	learnReply(c, out, err)
}

// AddSample POST /admin/content/persona/samples {url} or {title, text}
func (h *ContentEngineHandler) AddSample(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in struct {
		URL   string `json:"url"`
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.AddPersonaSample(c.Request.Context(), owner, in.URL, in.Title, in.Text)
	learnReply(c, out, err)
}

// DeleteSample DELETE /admin/content/persona/samples/:sid
func (h *ContentEngineHandler) DeleteSample(c *gin.Context) {
	if owner, ok := contentOwner(c); ok {
		out, err := h.svc.DeletePersonaSample(c.Request.Context(), owner, c.Param("sid"))
		learnReply(c, out, err)
	}
}

// DistillPersona POST /admin/content/persona/distill — a proposed 作者档案 from the samples.
func (h *ContentEngineHandler) DistillPersona(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), contentSlowTimeout)
	defer cancel()
	out, err := h.svc.DistillPersona(ctx, owner)
	learnReply(c, out, err)
}

// CreateDesk POST /admin/content/desk {topic_id | title, idea, notes}
func (h *ContentEngineHandler) CreateDesk(c *gin.Context) {
	owner, ok := contentOwner(c)
	if !ok {
		return
	}
	var in service.ContentDeskInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.CreateDesk(c.Request.Context(), owner, in)
	learnReply(c, out, err)
}

// deskAction runs a desk step that takes no body.
func (h *ContentEngineHandler) deskAction(c *gin.Context, fn func(ctx context.Context, owner, id int64) (*service.ContentItem, error)) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := fn(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

// Research POST /admin/content/items/:id/desk/research {hint}
func (h *ContentEngineHandler) Research(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Hint string `json:"hint"`
	}
	_ = c.ShouldBindJSON(&in)
	out, err := h.svc.Research(c.Request.Context(), owner, id, in.Hint)
	learnReply(c, out, err)
}

// SaveMaterials PUT /admin/content/items/:id/desk/materials {facts, answers}
func (h *ContentEngineHandler) SaveMaterials(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentMaterialsInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveMaterials(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// Angles POST /admin/content/items/:id/desk/angles
func (h *ContentEngineHandler) Angles(c *gin.Context) { h.deskAction(c, h.svc.Angles) }

// ChooseAngle PUT /admin/content/items/:id/desk/angle
func (h *ContentEngineHandler) ChooseAngle(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentAngle
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.ChooseAngle(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// Reference POST /admin/content/items/:id/desk/reference {url | text}
func (h *ContentEngineHandler) Reference(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		URL  string `json:"url"`
		Text string `json:"text"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Reference(c.Request.Context(), owner, id, in.URL, in.Text)
	learnReply(c, out, err)
}

// ClearReference DELETE /admin/content/items/:id/desk/reference
func (h *ContentEngineHandler) ClearReference(c *gin.Context) { h.deskAction(c, h.svc.ClearReference) }

// Drafts POST /admin/content/items/:id/desk/drafts
func (h *ContentEngineHandler) Drafts(c *gin.Context) { h.deskAction(c, h.svc.Drafts) }

// ChooseBase POST /admin/content/items/:id/desk/base {draft_id, highlights}
func (h *ContentEngineHandler) ChooseBase(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		DraftID    string   `json:"draft_id"`
		Highlights []string `json:"highlights"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.ChooseBase(c.Request.Context(), owner, id, in.DraftID, in.Highlights)
	learnReply(c, out, err)
}

// SaveBody PUT /admin/content/items/:id/desk/body {body, label}
func (h *ContentEngineHandler) SaveBody(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Body  string `json:"body"`
		Label string `json:"label"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveBody(c.Request.Context(), owner, id, in.Body, in.Label)
	learnReply(c, out, err)
}

// Assist POST /admin/content/items/:id/desk/assist
func (h *ContentEngineHandler) Assist(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentAssistInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Assist(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// AddFact POST /admin/content/items/:id/desk/facts
func (h *ContentEngineHandler) AddFact(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in service.ContentFact
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.AddFact(c.Request.Context(), owner, id, in)
	learnReply(c, out, err)
}

// Check POST /admin/content/items/:id/desk/check
func (h *ContentEngineHandler) Check(c *gin.Context) { h.deskAction(c, h.svc.Check) }

// SaveComments PUT /admin/content/items/:id/desk/comments {comments}
func (h *ContentEngineHandler) SaveComments(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Comments []service.ContentComment `json:"comments"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SaveComments(c.Request.Context(), owner, id, in.Comments)
	learnReply(c, out, err)
}

// Finalize POST /admin/content/items/:id/desk/finalize
func (h *ContentEngineHandler) Finalize(c *gin.Context) { h.deskAction(c, h.svc.Finalize) }

// ApplyLearn POST /admin/content/items/:id/desk/learn {id, accept}
func (h *ContentEngineHandler) ApplyLearn(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		ID     string `json:"id"`
		Accept bool   `json:"accept"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.ApplyLearn(c.Request.Context(), owner, id, in.ID, in.Accept)
	learnReply(c, out, err)
}

// Platforms POST /admin/content/items/:id/desk/platforms
func (h *ContentEngineHandler) Platforms(c *gin.Context) { h.deskAction(c, h.svc.Platforms) }

// CancelDeskJob POST /admin/content/items/:id/desk/cancel
func (h *ContentEngineHandler) CancelDeskJob(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		learnReply(c, gin.H{"ok": true}, h.svc.CancelDeskJob(c.Request.Context(), owner, id))
	}
}

// SetStage POST /admin/content/items/:id/desk/stage {stage}
func (h *ContentEngineHandler) SetStage(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	var in struct {
		Stage string `json:"stage"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SetStage(c.Request.Context(), owner, id, in.Stage)
	learnReply(c, out, err)
}

// Versions GET /admin/content/items/:id/versions
func (h *ContentEngineHandler) Versions(c *gin.Context) {
	if owner, id, ok := contentOwnerID(c); ok {
		out, err := h.svc.Versions(c.Request.Context(), owner, id)
		learnReply(c, out, err)
	}
}

func contentVID(c *gin.Context) (int64, bool) {
	v, err := strconv.ParseInt(c.Param("vid"), 10, 64)
	if err != nil || v <= 0 {
		response.BadRequest(c, "参数格式不正确（Invalid version id）")
		return 0, false
	}
	return v, true
}

// Version GET /admin/content/items/:id/versions/:vid
func (h *ContentEngineHandler) Version(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	if vid, ok := contentVID(c); ok {
		out, err := h.svc.Version(c.Request.Context(), owner, id, vid)
		learnReply(c, out, err)
	}
}

// RestoreVersion POST /admin/content/items/:id/versions/:vid/restore
func (h *ContentEngineHandler) RestoreVersion(c *gin.Context) {
	owner, id, ok := contentOwnerID(c)
	if !ok {
		return
	}
	if vid, ok := contentVID(c); ok {
		out, err := h.svc.RestoreVersion(c.Request.Context(), owner, id, vid)
		learnReply(c, out, err)
	}
}
