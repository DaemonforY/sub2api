//go:build unit

package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type learnRunRec struct {
	user   int64
	kind   string
	keyID  int64
	status string
}

type learnRepoStub struct {
	mu          sync.Mutex
	done        map[string]time.Time
	runs        []learnRunRec
	quizzes     map[string]LearnQuizResult
	checkpoints map[string]time.Time
	certs       []LearnCertificate
	interviews  []LearnInterview
}

func newLearnRepoStub() *learnRepoStub {
	return &learnRepoStub{done: map[string]time.Time{}, quizzes: map[string]LearnQuizResult{}, checkpoints: map[string]time.Time{}}
}

func (r *learnRepoStub) Progress(context.Context, int64) (map[string]time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]time.Time{}
	for k, v := range r.done {
		out[k] = v
	}
	return out, nil
}
func (r *learnRepoStub) MarkDone(_ context.Context, _ int64, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		r.done[id] = time.Now()
	}
	return nil
}
func (r *learnRepoStub) CountRuns(_ context.Context, userID int64, kind string, _ time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, run := range r.runs {
		if run.status != "failed" && run.keyID == 0 && (userID == 0 || run.user == userID) && (kind == "" || run.kind == kind) {
			n++
		}
	}
	return n, nil
}
func (r *learnRepoStub) StartRun(_ context.Context, userID int64, _ string, kind string, keyID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs = append(r.runs, learnRunRec{user: userID, kind: kind, keyID: keyID, status: "pending"})
	return int64(len(r.runs)), nil
}
func (r *learnRepoStub) FinishRun(_ context.Context, id int64, status string, _, _, _ int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[id-1].status = status
	return nil
}
func (r *learnRepoStub) Stats(context.Context, time.Time) (*LearnStats, error) {
	return &LearnStats{}, nil
}
func (r *learnRepoStub) QuizResults(context.Context, int64) (map[string]LearnQuizResult, error) {
	out := map[string]LearnQuizResult{}
	for k, v := range r.quizzes {
		out[k] = v
	}
	return out, nil
}
func (r *learnRepoStub) SaveQuizResult(_ context.Context, _ int64, lesson string, correct, total int) (LearnQuizResult, error) {
	q := r.quizzes[lesson]
	q.Attempts++
	if q.Total != total || correct > q.Correct {
		q.Correct = correct
	}
	q.Total = total
	r.quizzes[lesson] = q
	return q, nil
}
func (r *learnRepoStub) Checkpoints(context.Context, int64) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	for k, v := range r.checkpoints {
		out[k] = v
	}
	return out, nil
}
func (r *learnRepoStub) PassCheckpoint(_ context.Context, _ int64, id string) error {
	r.checkpoints[id] = time.Now()
	return nil
}
func (r *learnRepoStub) CreateCertificate(_ context.Context, c *LearnCertificate) error {
	for _, x := range r.certs {
		if x.UserID == c.UserID && x.Track == c.Track && x.RevokedAt == nil {
			return nil
		}
	}
	r.certs = append(r.certs, *c)
	return nil
}
func (r *learnRepoStub) CertificateByCode(_ context.Context, code string) (*LearnCertificate, error) {
	for _, x := range r.certs {
		if x.Code == code {
			c := x
			return &c, nil
		}
	}
	return nil, nil
}
func (r *learnRepoStub) CertificatesByUser(_ context.Context, userID int64) ([]LearnCertificate, error) {
	out := []LearnCertificate{}
	for _, x := range r.certs {
		if x.UserID == userID {
			out = append(out, x)
		}
	}
	return out, nil
}
func (r *learnRepoStub) ListCertificates(context.Context, int, int) ([]LearnCertificate, int, error) {
	return r.certs, len(r.certs), nil
}
func (r *learnRepoStub) SetCertificateRevoked(_ context.Context, code string, revoked bool) error {
	for i := range r.certs {
		if r.certs[i].Code == code {
			if revoked {
				now := time.Now()
				r.certs[i].RevokedAt = &now
			} else {
				r.certs[i].RevokedAt = nil
			}
			return nil
		}
	}
	return ErrLearnCertNotFound
}
func (r *learnRepoStub) CreateInterview(_ context.Context, iv *LearnInterview) error {
	iv.ID = int64(len(r.interviews) + 1)
	r.interviews = append(r.interviews, *iv)
	return nil
}
func (r *learnRepoStub) GetInterview(_ context.Context, id int64) (*LearnInterview, error) {
	if id < 1 || int(id) > len(r.interviews) {
		return nil, nil
	}
	iv := r.interviews[id-1]
	iv.Answers = append([]LearnInterviewAnswer{}, iv.Answers...)
	return &iv, nil
}
func (r *learnRepoStub) SaveInterview(_ context.Context, iv *LearnInterview) error {
	if len(r.interviews[iv.ID-1].Answers) != len(iv.Answers)-1 {
		return ErrLearnInterviewDone
	}
	r.interviews[iv.ID-1] = *iv
	return nil
}
func (r *learnRepoStub) ListInterviews(_ context.Context, userID int64, _ int) ([]LearnInterview, error) {
	out := []LearnInterview{}
	for _, iv := range r.interviews {
		if iv.UserID == userID {
			out = append(out, iv)
		}
	}
	return out, nil
}
func (r *learnRepoStub) CountInterviews(_ context.Context, userID int64, _ time.Time) (int, error) {
	n := 0
	for _, iv := range r.interviews {
		if iv.UserID == userID && iv.KeyID == 0 {
			n++
		}
	}
	return n, nil
}
func (r *learnRepoStub) BestInterviewScores(_ context.Context, userID int64) (map[string]int, error) {
	out := map[string]int{}
	for _, iv := range r.interviews {
		if iv.UserID == userID && iv.Status == learnInterviewStatusEnd && iv.Score > out[iv.Topic] {
			out[iv.Topic] = iv.Score
		}
	}
	return out, nil
}

// Sources for checkpoints / own keys.

type learnKeysStub struct{ keys []APIKey }

func (s *learnKeysStub) GetByID(_ context.Context, id int64) (*APIKey, error) {
	for _, k := range s.keys {
		if k.ID == id {
			k := k
			return &k, nil
		}
	}
	return nil, ErrAPIKeyNotFound
}
func (s *learnKeysStub) CountByUserID(_ context.Context, userID int64) (int64, error) {
	var n int64
	for _, k := range s.keys {
		if k.UserID == userID {
			n++
		}
	}
	return n, nil
}
func (s *learnKeysStub) ListByUserID(_ context.Context, userID int64, _ pagination.PaginationParams, _ APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	out := []APIKey{}
	for _, k := range s.keys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	return out, nil, nil
}

type learnSitesStub struct{ sites []Site }

func (s *learnSitesStub) ListSitesByUser(context.Context, int64) ([]Site, error) { return s.sites, nil }

type learnWorksStub struct {
	works  []Work
	handle string
}

func (s *learnWorksStub) ListWorks(context.Context, WorkQuery) ([]Work, error) { return s.works, nil }
func (s *learnWorksStub) GetProfileByUser(context.Context, int64) (*CommunityProfile, error) {
	return &CommunityProfile{CommunityAuthor: CommunityAuthor{Handle: s.handle}}, nil
}
