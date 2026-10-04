//go:build unit

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestLearnCatalogMatchesSite(t *testing.T) {
	c, err := loadLearnCatalog(learnCatalogJSON)
	require.NoError(t, err)

	// Every lesson on the site is in the catalog (a certificate needs all of them), in order.
	raw, err := os.ReadFile("../../../learn/.vitepress/theme/tracks.ts")
	require.NoError(t, err)
	site := map[string][]string{}
	for _, m := range regexp.MustCompile(`id: '([a-z])(\d{1,2})', title: '[^']*', minutes: \d+, ready: true`).FindAllStringSubmatch(string(raw), -1) {
		site[m[1]] = append(site[m[1]], m[1]+m[2])
	}
	for _, track := range c.Tracks {
		require.Equal(t, site[track.ID], track.Lessons, "track %s", track.ID)
		for _, id := range track.Lessons {
			_, err := os.Stat(fmt.Sprintf("../../../learn/%s/%s.md", track.ID, id))
			require.NoError(t, err, "lesson page %s", id)
		}
	}
	require.Len(t, site, len(c.Tracks))

	// Pages use only checkpoints and interview topics the server knows.
	for _, track := range c.Tracks {
		for _, id := range track.Lessons {
			page, err := os.ReadFile(fmt.Sprintf("../../../learn/%s/%s.md", track.ID, id))
			require.NoError(t, err)
			for _, m := range regexp.MustCompile(`<Checkpoint id="([a-z0-9]+)"`).FindAllStringSubmatch(string(page), -1) {
				require.Contains(t, c.Checkpoints, m[1], "%s checkpoint", id)
			}
			for _, m := range regexp.MustCompile(`<MockInterview topic="([a-z]+)"`).FindAllStringSubmatch(string(page), -1) {
				require.Contains(t, c.Interviews, m[1], "%s interview", id)
			}
		}
	}

	bad := strings.Replace(string(learnCatalogJSON), `"answer": [`, `"answer": [9, `, 1)
	_, err = loadLearnCatalog([]byte(bad))
	require.Error(t, err)
}

func newL2Service(t *testing.T, gw string, src LearnSources) (*LearnService, *learnRepoStub) {
	t.Helper()
	repo := newLearnRepoStub()
	settings := mapSettings{}
	svc := NewLearnService(repo, settings, prefixEncryptor{}, gw, src)
	_, err := svc.SaveSettings(context.Background(), LearnSettings{RunEnabled: true, Model: "gpt-5.5", FreeRunsPerDay: 1, DailyCap: 100,
		TutorFree: 1, InterviewsFree: 1, APIKey: "sk-learn"})
	require.NoError(t, err)
	return svc, repo
}

func TestLearnQuiz(t *testing.T) {
	ctx := context.Background()
	svc, repo := newL2Service(t, "http://127.0.0.1:1", LearnSources{})
	view, err := svc.Quiz(ctx, 0, "a3")
	require.NoError(t, err)
	b, _ := json.Marshal(view)
	require.NotContains(t, string(b), "answer")
	require.NotContains(t, string(b), "explain")

	qs := svc.catalog.Quizzes["a3"]
	right := make([][]int, len(qs))
	wrong := make([][]int, len(qs))
	for i, q := range qs {
		right[i] = q.Answer
		wrong[i] = []int{(q.Answer[0] + 1) % len(q.Options)}
	}
	_, err = svc.SubmitQuiz(ctx, 1, "a3", right[:1])
	require.Equal(t, "LEARN_QUIZ_ANSWERS", infraerrors.Reason(err))
	_, err = svc.SubmitQuiz(ctx, 1, "zz", right)
	require.Equal(t, "LEARN_QUIZ_NOT_FOUND", infraerrors.Reason(err))

	g, err := svc.SubmitQuiz(ctx, 1, "a3", wrong)
	require.NoError(t, err)
	require.False(t, g.Passed)
	require.NotContains(t, repo.done, "a3")
	g, err = svc.SubmitQuiz(ctx, 1, "a3", right)
	require.NoError(t, err)
	require.True(t, g.Passed)
	require.Equal(t, len(qs), g.Correct)
	require.Contains(t, repo.done, "a3") // passing marks the lesson done
	g, err = svc.SubmitQuiz(ctx, 1, "a3", wrong)
	require.NoError(t, err)
	require.Equal(t, len(qs), g.Best.Correct) // the best result stays
	require.Equal(t, 3, g.Best.Attempts)

	// Multiple choice needs exactly the right set, in any order.
	require.True(t, sameAnswer([]int{2, 0}, []int{0, 2}))
	require.False(t, sameAnswer([]int{0}, []int{0, 2}))
	require.False(t, sameAnswer([]int{0, 0}, []int{0, 2}))
}

func TestLearnCheckpointsAndCertificate(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	keys := &learnKeysStub{}
	sites := &learnSitesStub{}
	works := &learnWorksStub{handle: "mia"}
	svc, repo := newL2Service(t, "http://127.0.0.1:1", LearnSources{Keys: keys, Sites: sites, Works: works,
		SiteURL:    func(name string) string { return "https://" + name + ".s.example.test" },
		CanvasURL:  "https://canvas.example.test",
		InviteCode: func(context.Context, int64) string { return "AFF123" }})

	res, err := svc.VerifyCheckpoint(ctx, 1, "key")
	require.NoError(t, err)
	require.False(t, res.Passed)
	require.NotEmpty(t, res.Hint)
	keys.keys = []APIKey{{ID: 7, UserID: 1, Key: "sk-mine", Name: "mine", Status: StatusActive}}
	res, err = svc.VerifyCheckpoint(ctx, 1, "key")
	require.NoError(t, err)
	require.True(t, res.Passed)
	_, err = svc.VerifyCheckpoint(ctx, 1, "nope")
	require.Equal(t, "LEARN_CHECKPOINT_NOT_FOUND", infraerrors.Reason(err))

	// Track A: not ready until every lesson, quiz and checkpoint is done.
	st, err := svc.CertStatus(ctx, 1, "a")
	require.NoError(t, err)
	require.False(t, st.Eligible)
	require.True(t, st.ProjectLink)
	_, err = svc.ClaimCertificate(ctx, 1, "a", "小林", "https://github.com/x/y", false)
	require.Equal(t, "LEARN_CERT_NOT_READY", infraerrors.Reason(err))

	track := svc.catalog.track("a")
	for _, id := range track.Lessons {
		repo.done[id] = now
		if qs, ok := svc.catalog.Quizzes[id]; ok {
			repo.quizzes[id] = LearnQuizResult{Correct: len(qs) - 1, Total: len(qs)} // one wrong per lesson: 75%
		}
	}
	keys.keys[0].LastUsedAt = &now
	st, err = svc.CertStatus(ctx, 1, "a")
	require.NoError(t, err)
	require.False(t, st.Eligible)
	require.Equal(t, 75, st.QuizScore)
	for _, id := range track.Lessons[:4] {
		qs := svc.catalog.Quizzes[id]
		repo.quizzes[id] = LearnQuizResult{Correct: len(qs), Total: len(qs)}
	}
	st, err = svc.CertStatus(ctx, 1, "a")
	require.NoError(t, err)
	require.True(t, st.Eligible, "%+v", st.Items)
	require.Contains(t, repo.checkpoints, "call") // verified on the way

	// The project: a link (no site) or one of the learner's sites.
	_, err = svc.ClaimCertificate(ctx, 1, "a", "小林", "", false)
	require.Equal(t, "LEARN_CERT_PROJECT", infraerrors.Reason(err))
	_, err = svc.ClaimCertificate(ctx, 1, "a", "小林", "javascript:alert(1)", false)
	require.Equal(t, "LEARN_CERT_PROJECT", infraerrors.Reason(err))
	_, err = svc.ClaimCertificate(ctx, 1, "a", "<b>x</b>", "https://github.com/x/y", false)
	require.Equal(t, "LEARN_CERT_NAME", infraerrors.Reason(err))
	sites.sites = []Site{{Name: "mybot", Status: SiteStatusActive, Version: 1}, {Name: "draft", Status: SiteStatusPending}}
	cert, err := svc.ClaimCertificate(ctx, 1, "a", " 小林 ", "", false)
	require.NoError(t, err)
	require.Equal(t, "https://mybot.s.example.test", cert.ProjectURL)
	require.Equal(t, "小林", cert.DisplayName)
	require.Len(t, cert.Code, learnCertCodeLen)
	again, err := svc.ClaimCertificate(ctx, 1, "a", "别的名字", "", false)
	require.NoError(t, err)
	require.Equal(t, cert.Code, again.Code) // one per track

	pub, err := svc.Certificate(ctx, strings.ToLower(cert.Code))
	require.NoError(t, err)
	require.Equal(t, "AFF123", pub.InviteCode)
	require.Equal(t, "AI 应用开发入门", pub.TrackTitle)
	require.NoError(t, svc.SetCertificateRevoked(ctx, cert.Code, true))
	_, err = svc.Certificate(ctx, cert.Code)
	require.Equal(t, "LEARN_CERT_NOT_FOUND", infraerrors.Reason(err))

	// Track B: published works make the project, linking the canvas profile.
	works.works = []Work{{Status: WorkStatusApproved, Visibility: WorkVisibilityPublic}, {Status: WorkStatusPending, Visibility: WorkVisibilityPublic}}
	st, err = svc.CertStatus(ctx, 1, "b")
	require.NoError(t, err)
	last := st.Items[len(st.Items)-1]
	require.False(t, last.Done)
	require.Contains(t, last.Detail, "已公开发布 1 个")
	for i := 0; i < 3; i++ {
		works.works = append(works.works, Work{Status: WorkStatusApproved, Visibility: WorkVisibilityPublic})
	}
	st, err = svc.CertStatus(ctx, 1, "b")
	require.NoError(t, err)
	require.Equal(t, "https://canvas.example.test/u/mia", st.Projects[0].URL)

	// Track D: a passed interview in every topic.
	st, err = svc.CertStatus(ctx, 1, "d")
	require.NoError(t, err)
	require.Contains(t, st.Items[len(st.Items)-1].Detail, "已通过 0 / 4")
}

func TestLearnOwnKeyRun(t *testing.T) {
	ctx := context.Background()
	var auth string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"model":"gpt-5.5","choices":[{"finish_reason":"stop","message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer gw.Close()
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 1, Key: "sk-mine", Status: StatusActive},
		{ID: 8, UserID: 2, Key: "sk-other", Status: StatusActive},
		{ID: 9, UserID: 1, Key: "sk-off", Status: "disabled"},
	}}
	svc, repo := newL2Service(t, gw.URL, LearnSources{Keys: keys})
	in := LearnRunInput{Lesson: "a1", Messages: []LearnMessage{{Role: "user", Content: "hi"}}}

	_, err := svc.Run(ctx, 1, in)
	require.NoError(t, err)
	require.Equal(t, "Bearer sk-learn", auth)
	_, err = svc.Run(ctx, 1, in)
	require.Equal(t, "LEARN_RUN_QUOTA", infraerrors.Reason(err))

	for _, id := range []int64{8, 9, 99} {
		in.KeyID = id
		_, err = svc.Run(ctx, 1, in)
		require.Equal(t, "LEARN_KEY_INVALID", infraerrors.Reason(err), "key %d", id)
	}
	in.KeyID = 7
	res, err := svc.Run(ctx, 1, in)
	require.NoError(t, err)
	require.Equal(t, "Bearer sk-mine", auth)
	require.True(t, res.OwnKey)
	require.Equal(t, int64(7), repo.runs[len(repo.runs)-1].keyID)

	// Tool results in an agent loop are allowed.
	loop := LearnRunInput{Lesson: "a6", Messages: []LearnMessage{
		{Role: "user", Content: "3 杯拿铁多少钱"},
		{Role: "assistant", ToolCalls: json.RawMessage(`[{"id":"c1","type":"function","function":{"name":"calculator","arguments":"{}"}}]`)},
		{Role: "tool", ToolCallID: "c1", Content: "84"},
	}, KeyID: 7}
	_, err = svc.Run(ctx, 1, loop)
	require.NoError(t, err)
	loop.Messages[2].ToolCallID = ""
	_, err = svc.Run(ctx, 1, loop)
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
}

func TestLearnTutorStreams(t *testing.T) {
	ctx := context.Background()
	var got map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "text/event-stream")
		for _, piece := range []string{"结构化", "输出"} {
			_, _ = fmt.Fprintf(w, "data: {\"model\":\"gpt-5.5\",\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", piece)
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":30,\"completion_tokens\":4}}\n\ndata: [DONE]\n\n")
	}))
	defer gw.Close()
	svc, repo := newL2Service(t, gw.URL, LearnSources{})
	svc.SetLessonText(func(id string) (string, string) { return "结构化输出", "正文：用 JSON Schema。" })

	var out strings.Builder
	in := LearnTutorInput{Lesson: "a3", Messages: []LearnMessage{{Role: "user", Content: "strict 是什么意思？"}}}
	res, err := svc.Tutor(ctx, 1, in, func(s string) error { out.WriteString(s); return nil })
	require.NoError(t, err)
	require.Equal(t, "结构化输出", out.String())
	require.Equal(t, 4, res.CompletionTokens)
	require.Equal(t, true, got["stream"])
	msgs := got["messages"].([]any)
	system := msgs[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "课时「结构化输出」")
	require.Contains(t, system, "正文：用 JSON Schema。")
	require.Equal(t, "tutor", repo.runs[0].kind)

	_, err = svc.Tutor(ctx, 1, in, func(string) error { return nil })
	require.Equal(t, "LEARN_TUTOR_QUOTA", infraerrors.Reason(err))
	// Turns must alternate and end with a question.
	in.Messages = append(in.Messages, LearnMessage{Role: "user", Content: "again"})
	_, err = svc.Tutor(ctx, 2, in, func(string) error { return nil })
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
}

func TestLearnInterview(t *testing.T) {
	ctx := context.Background()
	score := 8
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.Contains(t, string(body), "json_schema")
		require.Contains(t, string(body), "参考要点")
		grade, _ := json.Marshal(map[string]any{"score": score, "comment": "要点基本完整", "better": "1. …"})
		resp, _ := json.Marshal(map[string]any{"model": "gpt-5.5", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(grade)}}}})
		_, _ = w.Write(resp)
	}))
	defer gw.Close()
	svc, repo := newL2Service(t, gw.URL, LearnSources{})

	_, err := svc.StartInterview(ctx, 1, "nope", 0)
	require.Equal(t, "LEARN_INTERVIEW_TOPIC", infraerrors.Reason(err))
	iv, err := svc.StartInterview(ctx, 1, "rag", 0)
	require.NoError(t, err)
	require.Len(t, iv.Questions, 1) // only the current question is shown
	require.Equal(t, 0, *iv.Left)
	_, err = svc.StartInterview(ctx, 1, "rag", 0)
	require.Equal(t, "LEARN_INTERVIEW_QUOTA", infraerrors.Reason(err))

	_, err = svc.AnswerInterview(ctx, 2, iv.ID, "答案")
	require.Equal(t, "LEARN_INTERVIEW_NOT_FOUND", infraerrors.Reason(err))
	_, err = svc.AnswerInterview(ctx, 1, iv.ID, "  ")
	require.Equal(t, "LEARN_INTERVIEW_ANSWER", infraerrors.Reason(err))
	for i := 0; i < learnInterviewQuestions; i++ {
		if i == learnInterviewQuestions-1 {
			score = 15 // clamped to 10
		}
		iv, err = svc.AnswerInterview(ctx, 1, iv.ID, fmt.Sprintf("第 %d 题的回答", i+1))
		require.NoError(t, err)
	}
	require.Equal(t, learnInterviewStatusEnd, iv.Status)
	require.Len(t, iv.Questions, learnInterviewQuestions)
	require.Equal(t, 84, iv.Score) // (8*4+10)/5*10
	_, err = svc.AnswerInterview(ctx, 1, iv.ID, "再答")
	require.Equal(t, "LEARN_INTERVIEW_DONE", infraerrors.Reason(err))
	best, _ := repo.BestInterviewScores(ctx, 1)
	require.Equal(t, 84, best["rag"])
	for _, run := range repo.runs {
		require.Equal(t, "interview", run.kind)
	}
}
