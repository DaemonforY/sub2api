package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// TutorHandler serves AI 助教: teachers' settings, materials and 学情 (/api/v1/tutors, login), and the
// students' page (/api/v1/t/:code, no account: a class password and a name, then a token).
type TutorHandler struct {
	svc *service.TutorService
}

func NewTutorHandler(svc *service.TutorService) *TutorHandler {
	return &TutorHandler{svc: svc}
}

const tutorTokenHeader = "X-Tutor-Token"

func (h *TutorHandler) tutorID(c *gin.Context) (int64, int64, bool) {
	subject, ok := requireAuth(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "助教编号不正确（Invalid id）")
		return 0, 0, false
	}
	return subject.UserID, id, true
}

// stream answers with SSE: {"delta"} events, then done (with extra fields) or {"error"}.
func tutorStream(c *gin.Context, run func(onDelta func(string) error) (gin.H, error)) {
	started := false
	send := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", b); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	done, err := run(func(delta string) error {
		if !started {
			started = true
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			c.Status(http.StatusOK)
		}
		return send(gin.H{"delta": delta})
	})
	if !started {
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if done == nil {
			done = gin.H{}
		}
		done["done"] = true
		response.Success(c, done)
		return
	}
	if err != nil {
		_ = send(gin.H{"error": infraerrors.Message(err)})
		return
	}
	if done == nil {
		done = gin.H{}
	}
	done["done"] = true
	_ = send(done)
}

// --- teacher ---

// Templates GET /tutors/templates
func (h *TutorHandler) Templates(c *gin.Context) {
	response.Success(c, service.TutorTemplates())
}

// List GET /tutors
func (h *TutorHandler) List(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), subject.UserID)
	learnReply(c, list, err)
}

// Create POST /tutors
func (h *TutorHandler) Create(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.TutorInput
	if !bindLearn(c, &in) {
		return
	}
	t, err := h.svc.Create(c.Request.Context(), subject.UserID, in)
	learnReply(c, t, err)
}

// Get GET /tutors/:id
func (h *TutorHandler) Get(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	t, err := h.svc.Get(c.Request.Context(), userID, id)
	learnReply(c, t, err)
}

// Update PUT /tutors/:id
func (h *TutorHandler) Update(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	var in service.TutorInput
	if !bindLearn(c, &in) {
		return
	}
	t, err := h.svc.Update(c.Request.Context(), userID, id, in)
	learnReply(c, t, err)
}

// Delete DELETE /tutors/:id
func (h *TutorHandler) Delete(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.Delete(c.Request.Context(), userID, id))
}

const tutorUploadLimit = 20<<20 + 1<<20

// AddMaterial POST /tutors/:id/materials — multipart `file`, or JSON {name, text} for pasted text.
func (h *TutorHandler) AddMaterial(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, tutorUploadLimit)
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			response.BadRequest(c, "请选择文件，单个最多 20MB（File required）")
			return
		}
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(io.LimitReader(file, tutorUploadLimit))
		if err != nil {
			response.BadRequest(c, "读取文件失败（Read failed）")
			return
		}
		m, err := h.svc.AddMaterial(c.Request.Context(), userID, id, header.Filename, data, "")
		learnReply(c, m, err)
		return
	}
	var in struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if !bindLearn(c, &in) {
		return
	}
	m, err := h.svc.AddMaterial(c.Request.Context(), userID, id, in.Name, nil, in.Text)
	learnReply(c, m, err)
}

// DeleteMaterial DELETE /tutors/:id/materials/:mid
func (h *TutorHandler) DeleteMaterial(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	mid, err := strconv.ParseInt(c.Param("mid"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, "资料编号不正确（Invalid id）")
		return
	}
	learnReply(c, gin.H{"ok": true}, h.svc.DeleteMaterial(c.Request.Context(), userID, id, mid))
}

// Preview POST /tutors/:id/preview {messages} — the teacher tries it as a student (SSE).
func (h *TutorHandler) Preview(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	var in service.TutorChatInput
	if !bindLearn(c, &in) {
		return
	}
	tutorStream(c, func(onDelta func(string) error) (gin.H, error) {
		return nil, h.svc.Preview(c.Request.Context(), userID, id, in, onDelta)
	})
}

// Stats GET /tutors/:id/stats
func (h *TutorHandler) Stats(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	st, err := h.svc.Stats(c.Request.Context(), userID, id)
	learnReply(c, st, err)
}

// Insights POST /tutors/:id/insights — AI summary of the last week's questions (SSE).
func (h *TutorHandler) Insights(c *gin.Context) {
	userID, id, ok := h.tutorID(c)
	if !ok {
		return
	}
	tutorStream(c, func(onDelta func(string) error) (gin.H, error) {
		return nil, h.svc.Insights(c.Request.Context(), userID, id, onDelta)
	})
}

// --- students ---

// Public GET /t/:code
func (h *TutorHandler) Public(c *gin.Context) {
	p, err := h.svc.Public(c.Request.Context(), c.Param("code"))
	learnReply(c, p, err)
}

// Join POST /t/:code/join {name, pass}
func (h *TutorHandler) Join(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
		Pass string `json:"pass"`
	}
	if !bindLearn(c, &in) {
		return
	}
	j, err := h.svc.Join(c.Request.Context(), c.Param("code"), in.Name, in.Pass)
	learnReply(c, j, err)
}

// Chat POST /t/:code/chat {messages} with the X-Tutor-Token header (SSE).
func (h *TutorHandler) Chat(c *gin.Context) {
	var in service.TutorChatInput
	if !bindLearn(c, &in) {
		return
	}
	token := c.GetHeader(tutorTokenHeader)
	tutorStream(c, func(onDelta func(string) error) (gin.H, error) {
		res, err := h.svc.Chat(c.Request.Context(), c.Param("code"), token, in, onDelta)
		if err != nil {
			return nil, err
		}
		return gin.H{"left": res.Left}, nil
	})
}
