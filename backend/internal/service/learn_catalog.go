package service

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// The AI 学习 catalog: what the server needs to know about the lessons on /learn. The pages
// themselves live in learn/ (VitePress); quiz answers, checkpoint rules, what a certificate needs
// and the interview question banks stay here so they are never sent to the browser.

//go:embed learn_catalog.json
var learnCatalogJSON []byte

type LearnTrackDef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Lessons that are online, in order (a certificate needs all of them).
	Lessons     []string `json:"lessons"`
	Checkpoints []string `json:"checkpoints"`
	// Project: "site" (a hosted site of theirs), "works" (published community works) or
	// "interviews" (a passed mock interview in each of Interviews).
	Project    string   `json:"project"`
	Interviews []string `json:"interviews,omitempty"`
	// CertClosed: the track is still being written, so its certificate can't be claimed yet.
	CertClosed bool `json:"cert_closed,omitempty"`
}

type LearnCheckpointDef struct {
	Title string `json:"title"`
	Hint  string `json:"hint"`
	// Rule: key | call | site | works (N public works, optional Kind).
	Rule string `json:"rule"`
	N    int    `json:"n,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type LearnQuizQuestion struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	// Answer: the correct option indexes (several: a multiple-choice question).
	Answer  []int  `json:"answer"`
	Explain string `json:"explain"`
}

type LearnInterviewQuestion struct {
	ID string `json:"id"`
	Q  string `json:"q"`
	// Points a good answer covers (for grading only).
	Points string `json:"points"`
}

type LearnInterviewTopic struct {
	Title string `json:"title"`
	// Track: "d" (counts for the AI interview certificate) or "bigdata".
	Track string `json:"track"`
	// Role the interviewer hires for (in the grading prompt), e.g. "AI 应用开发".
	Role      string                   `json:"role"`
	Questions []LearnInterviewQuestion `json:"questions"`
}

// LearnExerciseDef is an in-lesson practice exercise (学画画): the learner draws in the page
// (kind draw), uploads a picture (image) or writes an appreciation (text), and a model critiques it
// against the rubric.
type LearnExerciseDef struct {
	Lesson string `json:"lesson"`
	Title  string `json:"title"`
	Kind   string `json:"kind"`
	// Task is what the exercise asks, as the critic reads it.
	Task   string            `json:"task"`
	Rubric []LearnRubricItem `json:"rubric"`
	// Pass: the total score (0–100) that passes.
	Pass int `json:"pass"`
}

// LearnRubricItem is one thing the critic scores; the total is the weighted average.
type LearnRubricItem struct {
	Dim    string `json:"dim"`
	Look   string `json:"look"`
	Weight int    `json:"weight"`
}

type learnCatalog struct {
	Tracks      []LearnTrackDef                `json:"tracks"`
	Checkpoints map[string]LearnCheckpointDef  `json:"checkpoints"`
	Quizzes     map[string][]LearnQuizQuestion `json:"quizzes"`
	// InterviewTopics in order (the keys of Interviews).
	InterviewTopics []string                       `json:"interview_topics"`
	Interviews      map[string]LearnInterviewTopic `json:"interviews"`
	Exercises       map[string]LearnExerciseDef    `json:"exercises"`
}

func loadLearnCatalog(raw []byte) (*learnCatalog, error) {
	var c learnCatalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *learnCatalog) validate() error {
	seen := map[string]bool{}
	for _, t := range c.Tracks {
		for _, id := range t.Lessons {
			if !learnLessonRe.MatchString(id) || seen[id] {
				return fmt.Errorf("learn catalog: lesson %q", id)
			}
			seen[id] = true
		}
		for _, cp := range t.Checkpoints {
			if _, ok := c.Checkpoints[cp]; !ok {
				return fmt.Errorf("learn catalog: track %s checkpoint %q", t.ID, cp)
			}
		}
		switch t.Project {
		case "site", "works", "practice":
		case "interviews":
			if len(t.Interviews) == 0 {
				return fmt.Errorf("learn catalog: track %s needs interview topics", t.ID)
			}
			for _, topic := range t.Interviews {
				if _, ok := c.Interviews[topic]; !ok {
					return fmt.Errorf("learn catalog: track %s interview %q", t.ID, topic)
				}
			}
		default:
			return fmt.Errorf("learn catalog: track %s project %q", t.ID, t.Project)
		}
	}
	for id, cp := range c.Checkpoints {
		switch cp.Rule {
		case "key", "call", "site":
		case "works":
			if cp.N < 1 {
				return fmt.Errorf("learn catalog: checkpoint %s n", id)
			}
		default:
			return fmt.Errorf("learn catalog: checkpoint %s rule %q", id, cp.Rule)
		}
	}
	for lesson, qs := range c.Quizzes {
		if !seen[lesson] {
			return fmt.Errorf("learn catalog: quiz for unknown lesson %s", lesson)
		}
		for i, q := range qs {
			if q.Q == "" || len(q.Options) < 2 || len(q.Answer) == 0 {
				return fmt.Errorf("learn catalog: quiz %s #%d", lesson, i+1)
			}
			for _, a := range q.Answer {
				if a < 0 || a >= len(q.Options) {
					return fmt.Errorf("learn catalog: quiz %s #%d answer", lesson, i+1)
				}
			}
		}
	}
	for id, ex := range c.Exercises {
		if !seen[ex.Lesson] || !learnExerciseRe.MatchString(id) || !strings.HasPrefix(id, ex.Lesson+"-") {
			return fmt.Errorf("learn catalog: exercise %s", id)
		}
		if ex.Title == "" || ex.Task == "" || ex.Pass < 1 || ex.Pass > 100 || len(ex.Rubric) < 2 || len(ex.Rubric) > 6 {
			return fmt.Errorf("learn catalog: exercise %s fields", id)
		}
		switch ex.Kind {
		case learnExerciseDraw, learnExerciseImage, learnExerciseText:
		default:
			return fmt.Errorf("learn catalog: exercise %s kind %q", id, ex.Kind)
		}
		dims := map[string]bool{}
		for _, r := range ex.Rubric {
			if r.Dim == "" || r.Look == "" || r.Weight < 1 || dims[r.Dim] {
				return fmt.Errorf("learn catalog: exercise %s rubric %q", id, r.Dim)
			}
			dims[r.Dim] = true
		}
	}
	if len(c.InterviewTopics) != len(c.Interviews) {
		return fmt.Errorf("learn catalog: interview topics")
	}
	for _, topic := range c.InterviewTopics {
		t, ok := c.Interviews[topic]
		if !ok || len(t.Questions) < learnInterviewQuestions || (t.Track != "d" && t.Track != "bigdata") || t.Role == "" {
			return fmt.Errorf("learn catalog: interview topic %s", topic)
		}
		ids := map[string]bool{}
		for _, q := range t.Questions {
			if q.ID == "" || q.Q == "" || ids[q.ID] {
				return fmt.Errorf("learn catalog: interview %s question %q", topic, q.ID)
			}
			ids[q.ID] = true
		}
	}
	return nil
}

func (c *learnCatalog) track(id string) *LearnTrackDef {
	for i := range c.Tracks {
		if c.Tracks[i].ID == id {
			return &c.Tracks[i]
		}
	}
	return nil
}

func mustLearnCatalog() *learnCatalog {
	c, err := loadLearnCatalog(learnCatalogJSON)
	if err != nil {
		panic(err)
	}
	return c
}

// exercises of a track, in lesson order then by ID.
func (c *learnCatalog) trackExercises(t *LearnTrackDef) []string {
	var out []string
	for _, lesson := range t.Lessons {
		var ids []string
		for id, ex := range c.Exercises {
			if ex.Lesson == lesson {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		out = append(out, ids...)
	}
	return out
}
