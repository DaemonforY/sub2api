package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CommunityHandler serves the canvas community (/api/v1/canvas/community/*, canvas session) and
// its images (/api/v1/community/media/:file, public).
type CommunityHandler struct {
	svc *service.CommunityService
}

func NewCommunityHandler(svc *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{svc: svc}
}

var errCommunityBadRequest = infraerrors.BadRequest("COMMUNITY_BAD_REQUEST", "请求参数不正确（Invalid request）")

// publishBodyLimit: 9 images (the canvas shrinks them before uploading), or a video and its cover,
// plus the form fields.
const publishBodyLimit = 80 << 20

func viewerID(c *gin.Context) int64 {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	return 0
}

func mustViewer(c *gin.Context) (int64, bool) {
	if id := viewerID(c); id > 0 {
		return id, true
	}
	response.ErrorFrom(c, service.ErrCanvasSessionInvalid)
	return 0, false
}

func pathInt(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, errCommunityBadRequest)
		return 0, false
	}
	return id, true
}

func queryInt(c *gin.Context, name string) int {
	n, _ := strconv.Atoi(c.Query(name))
	return n
}

// Media GET /api/v1/community/media/:file
func (h *CommunityHandler) Media(c *gin.Context) {
	path, ok := h.svc.Media().Path(c.Param("file"))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.File(path)
}

// ShareMeta GET /api/v1/community/share-meta/*path — the <head> tags of a canvas page's share card
// (called by the canvas site's nginx through SSI). X-Share-Url, when it is on a canvas origin, becomes og:url.
func (h *CommunityHandler) ShareMeta(c *gin.Context, canvasOrigins []string) {
	scheme := "https"
	if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") == "http" {
		scheme = "http"
	}
	mainSite := scheme + "://" + c.Request.Host
	pageURL := service.TrustedShareURL(c.GetHeader("X-Share-Url"), canvasOrigins)
	out, err := h.svc.ShareMetaHTML(c.Request.Context(), c.Param("path"), mainSite, pageURL)
	if err != nil {
		out = ""
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

// Me GET /community/me — the viewer's profile (null before the first publish) and unread notices.
func (h *CommunityHandler) Me(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	profile, err := h.svc.MyProfile(c.Request.Context(), uid)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	unread, _ := h.svc.UnreadNotifications(c.Request.Context(), uid)
	response.Success(c, gin.H{"profile": profile, "unread_notifications": unread})
}

func readFormFile(fh *multipart.FileHeader) ([]byte, error) {
	return readFormFileMax(fh, service.CommunityImageMaxBytes)
}

func readFormFileMax(fh *multipart.FileHeader, limit int64) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, limit+1))
}

func tooLarge(err error) bool {
	var e *http.MaxBytesError
	return errors.As(err, &e)
}

// SaveProfile PUT /community/me/profile (multipart: handle, display_name, bio, avatar, clear_avatar)
func (h *CommunityHandler) SaveProfile(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.CommunityImageMaxBytes+(1<<20))
	in := service.ProfileInput{Handle: c.PostForm("handle"), DisplayName: c.PostForm("display_name"), Bio: c.PostForm("bio"), ClearAvatar: c.PostForm("clear_avatar") == "true"}
	if fh, err := c.FormFile("avatar"); err == nil {
		data, err := readFormFile(fh)
		if err != nil {
			response.ErrorFrom(c, service.ErrCommunityImageInvalid)
			return
		}
		in.Avatar = data
	} else if tooLarge(err) {
		response.ErrorFrom(c, service.ErrCommunityImageTooLarge)
		return
	}
	profile, err := h.svc.SaveProfile(c.Request.Context(), uid, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

// Profile GET /community/users/:handle
func (h *CommunityHandler) Profile(c *gin.Context) {
	p, err := h.svc.Profile(c.Request.Context(), c.Param("handle"), viewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

// Works GET /community/works?feed=recommended|latest|following|favorites&tag=&user=<handle>&collection=<id>&offset=&limit=
func (h *CommunityHandler) Works(c *gin.Context) {
	at, _ := strconv.ParseInt(c.Query("at"), 10, 64)
	q := service.WorkQuery{Feed: c.Query("feed"), ViewerID: viewerID(c), Tag: c.Query("tag"), Kind: c.Query("kind"), Offset: queryInt(c, "offset"), Limit: queryInt(c, "limit")}
	// Later pages send back the first page's "at" so the order holds while scrolling.
	q.At = service.FeedSnapshot(at, time.Now())
	if handle := c.Query("user"); handle != "" {
		p, err := h.svc.Profile(c.Request.Context(), handle, q.ViewerID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		q.Feed, q.UserID = "user", p.UserID
	}
	if id, _ := strconv.ParseInt(c.Query("collection"), 10, 64); id > 0 {
		q.Feed, q.CollectionID = "collection", id
	}
	works, err := h.svc.Works(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	limit := q.Limit
	if limit <= 0 || limit > 60 {
		limit = 24
	}
	response.Success(c, gin.H{"works": works, "next_offset": max(0, q.Offset) + len(works), "has_more": len(works) >= limit, "at": service.FeedSnapshotUnix(q.At)})
}

// PublicWorks GET /api/v1/community/works — the main site's home page wall: recommended or latest
// public works only (no per-user feeds, no viewer state), at most 24.
func (h *CommunityHandler) PublicWorks(c *gin.Context) {
	feed := c.Query("feed")
	if feed != "latest" {
		feed = "recommended"
	}
	limit := queryInt(c, "limit")
	if limit <= 0 || limit > 24 {
		limit = 12
	}
	works, err := h.svc.Works(c.Request.Context(), service.WorkQuery{Feed: feed, Tag: c.Query("tag"), Kind: c.Query("kind"), Limit: limit})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, gin.H{"works": works})
}

// Work GET /community/works/:id
func (h *CommunityHandler) Work(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	w, err := h.svc.Work(c.Request.Context(), id, viewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}

// RelatedWorks GET /canvas/community/works/:id/related — works like this one by other authors.
func (h *CommunityHandler) RelatedWorks(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	works, err := h.svc.RelatedWorks(c.Request.Context(), id, viewerID(c), queryInt(c, "limit"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"works": works})
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var tags []string
	if strings.HasPrefix(strings.TrimSpace(raw), "[") && json.Unmarshal([]byte(raw), &tags) == nil {
		return tags
	}
	return strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '，' || r == ' ' })
}

// Publish POST /community/works (multipart: images (1–9), title, description, prompt, show_prompt,
// model, params (JSON), source, tags, visibility, collection_id, remix_of, site_id; a video work
// sends video + video_duration_ms with its cover as the one image)
func (h *CommunityHandler) Publish(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, publishBodyLimit)
	form, err := c.MultipartForm()
	if err != nil {
		if tooLarge(err) {
			response.ErrorFrom(c, service.ErrCommunityImageTooLarge)
			return
		}
		response.ErrorFrom(c, service.ErrCommunityNoImages)
		return
	}
	in := service.PublishInput{
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		Prompt:      c.PostForm("prompt"),
		ShowPrompt:  c.PostForm("show_prompt") != "false",
		Model:       c.PostForm("model"),
		Params:      json.RawMessage(c.PostForm("params")),
		Source:      c.PostForm("source"),
		Tags:        splitTags(c.PostForm("tags")),
		Visibility:  c.PostForm("visibility"),
	}
	in.CollectionID, _ = strconv.ParseInt(c.PostForm("collection_id"), 10, 64)
	in.RemixOf, _ = strconv.ParseInt(c.PostForm("remix_of"), 10, 64)
	in.SiteID, _ = strconv.ParseInt(c.PostForm("site_id"), 10, 64)
	if videos := form.File["video"]; len(videos) > 0 {
		data, err := readFormFileMax(videos[0], service.CommunityVideoMaxBytes)
		if err != nil {
			response.ErrorFrom(c, service.ErrCommunityVideoInvalid)
			return
		}
		in.Video = data
		in.VideoDurationMs, _ = strconv.Atoi(c.PostForm("video_duration_ms"))
	}
	files := form.File["images"]
	if len(files) == 0 || len(files) > 9 {
		response.ErrorFrom(c, service.ErrCommunityNoImages)
		return
	}
	for _, fh := range files {
		data, err := readFormFile(fh)
		if err != nil {
			response.ErrorFrom(c, service.ErrCommunityImageInvalid)
			return
		}
		in.Images = append(in.Images, data)
	}
	w, err := h.svc.Publish(c.Request.Context(), uid, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}

// UpdateWork PUT /community/works/:id {title, description, show_prompt, tags, visibility}
func (h *CommunityHandler) UpdateWork(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		ShowPrompt  bool     `json:"show_prompt"`
		Tags        []string `json:"tags"`
		// Omitted: unchanged.
		CommentsClosed *bool  `json:"comments_closed"`
		Visibility     string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	w, err := h.svc.UpdateWork(c.Request.Context(), uid, id, service.UpdateWorkInput{Title: in.Title, Description: in.Description, ShowPrompt: in.ShowPrompt, Tags: in.Tags, Visibility: in.Visibility, CommentsClosed: in.CommentsClosed})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}

// DeleteWork DELETE /community/works/:id
func (h *CommunityHandler) DeleteWork(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteWork(c.Request.Context(), uid, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *CommunityHandler) interaction(c *gin.Context, set func(uid, id int64, on bool) (*service.InteractionState, error)) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	state, err := set(uid, id, c.Request.Method == http.MethodPut)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

// Like PUT|DELETE /community/works/:id/like
func (h *CommunityHandler) Like(c *gin.Context) {
	h.interaction(c, func(uid, id int64, on bool) (*service.InteractionState, error) {
		return h.svc.SetLike(c.Request.Context(), uid, id, on)
	})
}

// Favorite PUT|DELETE /community/works/:id/favorite
func (h *CommunityHandler) Favorite(c *gin.Context) {
	h.interaction(c, func(uid, id int64, on bool) (*service.InteractionState, error) {
		return h.svc.SetFavorite(c.Request.Context(), uid, id, on)
	})
}

// Remix POST /community/works/:id/remix — someone opened "做同款".
func (h *CommunityHandler) Remix(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Remix(c.Request.Context(), viewerID(c), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Report POST /community/works/:id/report {reason, detail}
func (h *CommunityHandler) Report(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
		Detail string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrCommunityReportInvalid)
		return
	}
	if err := h.svc.Report(c.Request.Context(), viewerID(c), id, in.Reason, in.Detail, c.ClientIP()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Comments GET /canvas/community/works/:id/comments?offset= — top-level comments with their first replies.
func (h *CommunityHandler) Comments(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	page, err := h.svc.Comments(c.Request.Context(), id, viewerID(c), queryInt(c, "offset"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, page)
}

// CommentReplies GET /canvas/community/comments/:id/replies?offset=
func (h *CommunityHandler) CommentReplies(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	offset := queryInt(c, "offset")
	list, more, err := h.svc.CommentReplies(c.Request.Context(), id, viewerID(c), offset)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"replies": list, "next_offset": max(0, offset) + len(list), "has_more": more})
}

// AddComment POST /canvas/community/works/:id/comments {body, reply_to}
func (h *CommunityHandler) AddComment(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in struct {
		Body    string `json:"body"`
		ReplyTo int64  `json:"reply_to"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrCommunityCommentEmpty)
		return
	}
	comment, err := h.svc.AddComment(c.Request.Context(), uid, id, service.CommentInput{Body: in.Body, ReplyTo: in.ReplyTo}, c.ClientIP())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, comment)
}

// DeleteComment DELETE /canvas/community/comments/:id — the commenter's or the work author's.
func (h *CommunityHandler) DeleteComment(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteComment(c.Request.Context(), uid, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ReportComment POST /canvas/community/comments/:id/report {reason, detail}
func (h *CommunityHandler) ReportComment(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
		Detail string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrCommunityReportInvalid)
		return
	}
	if err := h.svc.ReportComment(c.Request.Context(), viewerID(c), id, in.Reason, in.Detail, c.ClientIP()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Follow PUT|DELETE /community/users/:handle/follow
func (h *CommunityHandler) Follow(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	p, err := h.svc.SetFollow(c.Request.Context(), uid, c.Param("handle"), c.Request.Method == http.MethodPut)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, p)
}

// Followers GET /community/users/:handle/followers ; Following GET /community/users/:handle/following
func (h *CommunityHandler) Followers(c *gin.Context) { h.follows(c, true) }
func (h *CommunityHandler) Following(c *gin.Context) { h.follows(c, false) }

func (h *CommunityHandler) follows(c *gin.Context, followers bool) {
	list, err := h.svc.Follows(c.Request.Context(), c.Param("handle"), followers, viewerID(c), queryInt(c, "offset"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Collections GET /community/users/:handle/collections
func (h *CommunityHandler) Collections(c *gin.Context) {
	list, err := h.svc.Collections(c.Request.Context(), c.Param("handle"), viewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Collection GET /community/collections/:id
func (h *CommunityHandler) Collection(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	col, err := h.svc.Collection(c.Request.Context(), id, viewerID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, col)
}

type collectionBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

// CreateCollection POST /community/collections
func (h *CommunityHandler) CreateCollection(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	var in collectionBody
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	col, err := h.svc.CreateCollection(c.Request.Context(), uid, service.CollectionInput(in))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, col)
}

// UpdateCollection PUT /community/collections/:id
func (h *CommunityHandler) UpdateCollection(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in collectionBody
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	col, err := h.svc.UpdateCollection(c.Request.Context(), uid, id, service.CollectionInput(in))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, col)
}

// DeleteCollection DELETE /community/collections/:id
func (h *CommunityHandler) DeleteCollection(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteCollection(c.Request.Context(), uid, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// CollectionItem PUT|DELETE /community/collections/:id/works/:work_id
func (h *CommunityHandler) CollectionItem(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	workID, ok := pathInt(c, "work_id")
	if !ok {
		return
	}
	if err := h.svc.SetCollectionItem(c.Request.Context(), uid, id, workID, c.Request.Method == http.MethodPut); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// WorkCollections GET /community/works/:id/collections — which of my collections hold my work.
func (h *CommunityHandler) WorkCollections(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	ids, err := h.svc.WorkCollections(c.Request.Context(), uid, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ids)
}

// Notifications GET /community/notifications ; POST /community/notifications/read
func (h *CommunityHandler) Notifications(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	list, err := h.svc.Notifications(c.Request.Context(), uid)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

func (h *CommunityHandler) ReadNotifications(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	if err := h.svc.MarkNotificationsRead(c.Request.Context(), uid); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

type contestEntryRequest struct {
	ContestID   int64  `json:"contest_id"`
	WorkID      int64  `json:"work_id"`
	ImageIndex  int    `json:"image_index"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *CommunityHandler) enterContest(c *gin.Context, in contestEntryRequest) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	if in.ContestID <= 0 || in.WorkID <= 0 {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	entry, err := h.svc.EnterContest(c.Request.Context(), uid, service.WorkContestInput{
		ContestID: in.ContestID, WorkID: in.WorkID, ImageIndex: in.ImageIndex, Title: in.Title, Description: in.Description,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry)
}

// EnterContest POST /api/v1/canvas/community/works/:id/contest-entries — the canvas enters the
// signed-in user's work in a contest (body: contest_id, image_index, title, description).
func (h *CommunityHandler) EnterContest(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in contestEntryRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	in.WorkID = id
	h.enterContest(c, in)
}

// EnterContestFromSite POST /api/v1/contests/:id/work-entries — the main site's contest page enters
// one of the logged-in user's canvas works (body: work_id, image_index, title, description).
func (h *CommunityHandler) EnterContestFromSite(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var in contestEntryRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, errCommunityBadRequest)
		return
	}
	in.ContestID = id
	h.enterContest(c, in)
}

// MyWorksForSite GET /api/v1/user/community/works — the logged-in user's own works (all states),
// for the main site's "enter a canvas work" picker.
func (h *CommunityHandler) MyWorksForSite(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	works, err := h.svc.Works(c.Request.Context(), service.WorkQuery{Feed: "user", UserID: uid, ViewerID: uid, Offset: queryInt(c, "offset"), Limit: 60})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"works": works})
}

// CreatorStats GET /api/v1/canvas/community/me/stats?days=7|30|90 — the signed-in author's numbers.
func (h *CommunityHandler) CreatorStats(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	stats, err := h.svc.CreatorStats(c.Request.Context(), uid, queryInt(c, "days"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// MySites GET /api/v1/canvas/community/me/sites — the signed-in user's sites for "publish a web page".
func (h *CommunityHandler) MySites(c *gin.Context) {
	uid, ok := mustViewer(c)
	if !ok {
		return
	}
	sites, err := h.svc.MySites(c.Request.Context(), uid)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"sites": sites})
}
