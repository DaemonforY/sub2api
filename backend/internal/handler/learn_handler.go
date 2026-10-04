package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// LearnHandler serves /learn's API: public config, progress, example runs, quizzes, checkpoints,
// certificates, the AI tutor and mock interviews.
type LearnHandler struct {
	svc *service.LearnService
}

func NewLearnHandler(svc *service.LearnService) *LearnHandler {
	return &LearnHandler{svc: svc}
}

// SetLessonText gives the tutor the built lesson pages.
func (h *LearnHandler) SetLessonText(fn func(lessonID string) (title, text string)) {
	h.svc.SetLessonText(fn)
}

func bindLearn(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.BadRequest(c, "参数格式不正确（Invalid request）")
		return false
	}
	return true
}

func learnReply(c *gin.Context, v any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}

// Config GET /api/v1/learn/config
func (h *LearnHandler) Config(c *gin.Context) {
	response.Success(c, h.svc.Config(c.Request.Context()))
}

// Me GET /api/v1/learn/me — progress, quizzes, checkpoints, certificates, free calls left today.
func (h *LearnHandler) Me(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	me, err := h.svc.Me(c.Request.Context(), subject.UserID)
	learnReply(c, me, err)
}

// Progress POST /api/v1/learn/progress {lesson_ids} — mark lessons done (or merge local progress).
func (h *LearnHandler) Progress(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		LessonIDs []string `json:"lesson_ids"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLearnLessonID)
		return
	}
	done, err := h.svc.MarkDone(c.Request.Context(), subject.UserID, in.LessonIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"completed": done})
}

// Run POST /api/v1/learn/run {lesson, messages, response_format?, tools?, key_id?}
func (h *LearnHandler) Run(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.LearnRunInput
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.Run(c.Request.Context(), subject.UserID, in)
	learnReply(c, out, err)
}

// Keys GET /api/v1/learn/keys — the learner's active keys, for going on past the free calls.
func (h *LearnHandler) Keys(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	keys, err := h.svc.OwnKeys(c.Request.Context(), subject.UserID)
	learnReply(c, keys, err)
}

// Quiz GET /api/v1/learn/quiz/:lesson (public: the questions without answers)
func (h *LearnHandler) Quiz(c *gin.Context) {
	quiz, err := h.svc.Quiz(c.Request.Context(), 0, c.Param("lesson"))
	learnReply(c, quiz, err)
}

// SubmitQuiz POST /api/v1/learn/quiz/:lesson {answers: [[0], [1, 2], ...]}
func (h *LearnHandler) SubmitQuiz(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		Answers [][]int `json:"answers"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.SubmitQuiz(c.Request.Context(), subject.UserID, c.Param("lesson"), in.Answers)
	learnReply(c, out, err)
}

// Checkpoint GET /api/v1/learn/checkpoints/:id (public: title and hint)
func (h *LearnHandler) Checkpoint(c *gin.Context) {
	def, ok := h.svc.CheckpointInfo(c.Param("id"))
	if !ok {
		response.ErrorFrom(c, service.ErrLearnCheckpoint)
		return
	}
	response.Success(c, gin.H{"id": c.Param("id"), "title": def.Title, "hint": def.Hint})
}

// VerifyCheckpoint POST /api/v1/learn/checkpoints/:id/verify
func (h *LearnHandler) VerifyCheckpoint(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	out, err := h.svc.VerifyCheckpoint(c.Request.Context(), subject.UserID, c.Param("id"))
	learnReply(c, out, err)
}

// CertStatus GET /api/v1/learn/certificates/:track — the track's checklist and certificate.
func (h *LearnHandler) CertStatus(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	out, err := h.svc.CertStatus(c.Request.Context(), subject.UserID, c.Param("track"))
	learnReply(c, out, err)
}

// ClaimCert POST /api/v1/learn/certificates/:track {display_name, project_url?}
func (h *LearnHandler) ClaimCert(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		DisplayName string `json:"display_name"`
		ProjectURL  string `json:"project_url"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.ClaimCertificate(c.Request.Context(), subject.UserID, c.Param("track"), in.DisplayName, in.ProjectURL)
	learnReply(c, out, err)
}

// Certificate GET /api/v1/learn/cert/:code (public)
func (h *LearnHandler) Certificate(c *gin.Context) {
	out, err := h.svc.Certificate(c.Request.Context(), c.Param("code"))
	learnReply(c, out, err)
}

// Tutor POST /api/v1/learn/tutor {lesson, messages, key_id?} — the reply streams as SSE:
// data: {"delta": "..."} pieces, then data: {"done": true, ...} or data: {"error": "..."}.
// Errors before the reply starts come back as a normal JSON error.
func (h *LearnHandler) Tutor(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in service.LearnTutorInput
	if !bindLearn(c, &in) {
		return
	}
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
	res, err := h.svc.Tutor(c.Request.Context(), subject.UserID, in, func(delta string) error {
		if !started {
			started = true
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			c.Status(200)
		}
		return send(gin.H{"delta": delta})
	})
	if !started {
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, res)
		return
	}
	if err != nil {
		_ = send(gin.H{"error": err.Error()})
		return
	}
	_ = send(gin.H{"done": true, "tutor_left": res.TutorLeft, "own_key": res.OwnKey, "model": res.Model})
}

// Interviews GET /api/v1/learn/interviews — topics with my best scores, and recent interviews.
func (h *LearnHandler) Interviews(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	topics, err := h.svc.InterviewTopics(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	recent, err := h.svc.Interviews(c.Request.Context(), subject.UserID)
	learnReply(c, gin.H{"topics": topics, "recent": recent}, err)
}

// StartInterview POST /api/v1/learn/interviews {topic, key_id?}
func (h *LearnHandler) StartInterview(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		Topic string `json:"topic"`
		KeyID int64  `json:"key_id"`
	}
	if !bindLearn(c, &in) {
		return
	}
	out, err := h.svc.StartInterview(c.Request.Context(), subject.UserID, in.Topic, in.KeyID)
	learnReply(c, out, err)
}

// Interview GET /api/v1/learn/interviews/:id
func (h *LearnHandler) Interview(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.Interview(c.Request.Context(), subject.UserID, id)
	learnReply(c, out, err)
}

// AnswerInterview POST /api/v1/learn/interviews/:id/answer {answer}
func (h *LearnHandler) AnswerInterview(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var in struct {
		Answer string `json:"answer"`
	}
	if !bindLearn(c, &in) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	out, err := h.svc.AnswerInterview(c.Request.Context(), subject.UserID, id, in.Answer)
	learnReply(c, out, err)
}
