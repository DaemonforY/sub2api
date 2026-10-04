package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Mock interviews (track D): questions picked from a topic's bank, the learner answers one at a
// time and the model grades each answer (score 0–10, comment, what a strong answer covers). The
// interview's score is the average × 10.

const (
	learnInterviewQuestions = 5
	learnInterviewMaxAnswer = 2000
	learnInterviewStatusOn  = "active"
	learnInterviewStatusEnd = "finished"
)

var (
	ErrLearnInterviewTopic    = infraerrors.NotFound("LEARN_INTERVIEW_TOPIC", "面试题组不存在（Unknown topic）")
	ErrLearnInterviewNotFound = infraerrors.NotFound("LEARN_INTERVIEW_NOT_FOUND", "面试记录不存在（Interview not found）")
	ErrLearnInterviewDone     = infraerrors.BadRequest("LEARN_INTERVIEW_DONE", "这场面试已经结束了，可以再开始一场（Interview finished）")
	ErrLearnInterviewAnswer   = infraerrors.BadRequest("LEARN_INTERVIEW_ANSWER", "回答需要 1–2000 字；不会可以直接写「不会」（Answer 1–2000 characters）")
	ErrLearnInterviewQuota    = infraerrors.TooManyRequests("LEARN_INTERVIEW_QUOTA", "今天的免费模拟面试次数用完了，明天再来，或者选择用自己的 Key 继续（Daily free interviews used up）")
)

type LearnInterviewQ struct {
	ID string `json:"id"`
	Q  string `json:"q"`
}

type LearnInterviewAnswer struct {
	Answer  string `json:"answer"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
	Better  string `json:"better"`
}

type LearnInterview struct {
	ID         int64                  `json:"id"`
	UserID     int64                  `json:"-"`
	Topic      string                 `json:"topic"`
	TopicTitle string                 `json:"topic_title"`
	Questions  []LearnInterviewQ      `json:"questions"`
	Answers    []LearnInterviewAnswer `json:"answers"`
	Status     string                 `json:"status"`
	Score      int                    `json:"score"`
	KeyID      int64                  `json:"-"`
	OwnKey     bool                   `json:"own_key,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	FinishedAt *time.Time             `json:"finished_at,omitempty"`
	// Left: free interviews left today (start only).
	Left *int `json:"interviews_left,omitempty"`
}

// view hides the questions not asked yet.
func (s *LearnService) interviewView(iv *LearnInterview) *LearnInterview {
	out := *iv
	if t, ok := s.catalog.Interviews[iv.Topic]; ok {
		out.TopicTitle = t.Title
	}
	shown := min(len(iv.Answers)+1, len(iv.Questions))
	if iv.Status == learnInterviewStatusEnd {
		shown = len(iv.Questions)
	}
	out.Questions = iv.Questions[:shown]
	out.OwnKey = iv.KeyID > 0
	return &out
}

type LearnInterviewTopicView struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Track string `json:"track"`
	Count int    `json:"count"`
	Best  int    `json:"best"`
}

// InterviewTopics lists the topics (with the learner's best score when userID > 0).
func (s *LearnService) InterviewTopics(ctx context.Context, userID int64) ([]LearnInterviewTopicView, error) {
	best := map[string]int{}
	if userID > 0 {
		var err error
		if best, err = s.repo.BestInterviewScores(ctx, userID); err != nil {
			return nil, err
		}
	}
	out := make([]LearnInterviewTopicView, 0, len(s.catalog.InterviewTopics))
	for _, id := range s.catalog.InterviewTopics {
		t := s.catalog.Interviews[id]
		out = append(out, LearnInterviewTopicView{ID: id, Title: t.Title, Track: t.Track, Count: len(t.Questions), Best: best[id]})
	}
	return out, nil
}

// StartInterview picks the questions for a new interview (free ones a day, or on the learner's key).
func (s *LearnService) StartInterview(ctx context.Context, userID int64, topic string, keyID int64) (*LearnInterview, error) {
	t, ok := s.catalog.Interviews[topic]
	if !ok {
		return nil, ErrLearnInterviewTopic
	}
	st, learnKey := s.loadSettings(ctx)
	if !st.RunEnabled || learnKey == "" || st.Model == "" {
		return nil, ErrLearnRunDisabled
	}
	left := 0
	used, err := s.repo.CountInterviews(ctx, userID, s.learnSince())
	if err != nil {
		return nil, err
	}
	if keyID > 0 {
		if _, err := s.ownKey(ctx, userID, keyID); err != nil {
			return nil, err
		}
		left = max(st.InterviewsFree-used, 0)
	} else {
		if used >= st.InterviewsFree {
			return nil, ErrLearnInterviewQuota
		}
		left = st.InterviewsFree - used - 1
	}
	order := rand.Perm(len(t.Questions))[:learnInterviewQuestions]
	iv := &LearnInterview{UserID: userID, Topic: topic, Status: learnInterviewStatusOn, KeyID: keyID,
		Questions: make([]LearnInterviewQ, 0, learnInterviewQuestions), Answers: []LearnInterviewAnswer{}, CreatedAt: s.now()}
	for _, i := range order {
		iv.Questions = append(iv.Questions, LearnInterviewQ{ID: t.Questions[i].ID, Q: t.Questions[i].Q})
	}
	if err := s.repo.CreateInterview(ctx, iv); err != nil {
		return nil, err
	}
	out := s.interviewView(iv)
	out.Left = &left
	return out, nil
}

func (s *LearnService) ownInterview(ctx context.Context, userID, id int64) (*LearnInterview, error) {
	iv, err := s.repo.GetInterview(ctx, id)
	if err != nil || iv == nil || iv.UserID != userID {
		return nil, ErrLearnInterviewNotFound
	}
	return iv, nil
}

// Interview returns one of the learner's interviews.
func (s *LearnService) Interview(ctx context.Context, userID, id int64) (*LearnInterview, error) {
	iv, err := s.ownInterview(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.interviewView(iv), nil
}

// Interviews lists the learner's recent interviews.
func (s *LearnService) Interviews(ctx context.Context, userID int64) ([]LearnInterview, error) {
	list, err := s.repo.ListInterviews(ctx, userID, 20)
	if err != nil {
		return nil, err
	}
	out := make([]LearnInterview, 0, len(list))
	for i := range list {
		v := s.interviewView(&list[i])
		v.Answers = nil
		out = append(out, *v)
	}
	return out, nil
}

var learnGradeSchema = json.RawMessage(`{"type":"json_schema","json_schema":{"name":"grade","strict":true,"schema":{"type":"object","properties":{"score":{"type":"integer","description":"0–10 分"},"comment":{"type":"string","description":"点评：答对了什么、缺了什么、哪里不准确，80–200 字"},"better":{"type":"string","description":"一个好回答应该覆盖的要点，分条，150–300 字"}},"required":["score","comment","better"],"additionalProperties":false}}}`)

func (s *LearnService) findInterviewQuestion(topic, id string) LearnInterviewQuestion {
	for _, q := range s.catalog.Interviews[topic].Questions {
		if q.ID == id {
			return q
		}
	}
	return LearnInterviewQuestion{ID: id}
}

// AnswerInterview grades the answer to the current question; the last one finishes the interview.
func (s *LearnService) AnswerInterview(ctx context.Context, userID, id int64, answer string) (*LearnInterview, error) {
	answer = strings.TrimSpace(answer)
	if n := utf8.RuneCountInString(answer); n == 0 || n > learnInterviewMaxAnswer {
		return nil, ErrLearnInterviewAnswer
	}
	iv, err := s.ownInterview(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if iv.Status != learnInterviewStatusOn || len(iv.Answers) >= len(iv.Questions) {
		return nil, ErrLearnInterviewDone
	}
	q := s.findInterviewQuestion(iv.Topic, iv.Questions[len(iv.Answers)].ID)
	topic := s.catalog.Interviews[iv.Topic]

	// Grading is part of the interview (the free count is per interview), so no free quota here.
	call, st, err := s.acquire(ctx, userID, "d0", learnKindInterview, iv.KeyID, 0, nil)
	if err != nil {
		return nil, err
	}
	system := fmt.Sprintf(`你是%s岗位的技术面试官，正在进行「%s」方向的模拟面试。请为候选人对下面这道题的回答打分并点评。
评分标准（0–10 分）：0–3 没答到要点或有明显错误；4–6 答到部分要点；7–8 要点基本完整、表达清楚；9–10 完整准确，还有自己的实践和取舍。
回答「不会」或与题目无关时给 0–1 分，并在 better 里讲清楚这道题该怎么答。
点评要具体，指出答对了什么、缺了什么；不要因为回答长就给高分。用中文。
题目：%s
参考要点（候选人看不到）：%s`, topic.Role, topic.Title, q.Q, q.Points)
	in := LearnRunInput{Lesson: "d0", Messages: []LearnMessage{{Role: "system", Content: system}, {Role: "user", Content: "候选人的回答：\n" + answer}},
		ResponseFormat: learnGradeSchema}
	started := s.now()
	res, callErr := s.callGateway(ctx, call.key, st.Model, in)
	var grade LearnInterviewAnswer
	if callErr == nil && json.Unmarshal([]byte(res.Content), &grade) != nil {
		callErr = errLearnRunFailed("评分结果无法识别")
	}
	if callErr != nil {
		s.finish(ctx, call, started, callErr, 0, 0)
		return nil, callErr
	}
	s.finish(ctx, call, started, nil, res.PromptTokens, res.CompletionTokens)
	grade.Answer = answer
	grade.Score = min(max(grade.Score, 0), 10)
	iv.Answers = append(iv.Answers, grade)
	if len(iv.Answers) == len(iv.Questions) {
		sum := 0
		for _, a := range iv.Answers {
			sum += a.Score
		}
		iv.Score = int(math.Round(float64(sum) * 10 / float64(len(iv.Answers))))
		iv.Status = learnInterviewStatusEnd
		now := s.now()
		iv.FinishedAt = &now
	}
	if err := s.repo.SaveInterview(ctx, iv); err != nil {
		return nil, err
	}
	return s.interviewView(iv), nil
}
