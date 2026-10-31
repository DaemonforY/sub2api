//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memDesktopSkills struct {
	rows []DesktopSkill
}

func (m *memDesktopSkills) ListDesktopSkills(_ context.Context, publishedOnly bool) ([]DesktopSkill, error) {
	out := []DesktopSkill{}
	for _, r := range m.rows {
		if !publishedOnly || r.Status == DesktopSkillPublished {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *memDesktopSkills) GetDesktopSkill(_ context.Context, id int64) (*DesktopSkill, error) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			return &m.rows[i], nil
		}
	}
	return nil, ErrDesktopSkillNotFound
}
func (m *memDesktopSkills) CreateDesktopSkill(_ context.Context, s *DesktopSkill, _ int64) (int64, error) {
	s.ID = int64(len(m.rows) + 1)
	s.Version = 1
	m.rows = append(m.rows, *s)
	return s.ID, nil
}
func (m *memDesktopSkills) UpdateDesktopSkill(context.Context, *DesktopSkill, int64) error {
	return nil
}
func (m *memDesktopSkills) SetDesktopSkillStatus(context.Context, int64, string, int64) error {
	return nil
}
func (m *memDesktopSkills) DeleteDesktopSkill(context.Context, int64) error { return nil }
func (m *memDesktopSkills) DesktopOverview(_ context.Context, since time.Time, prefix string) (*DesktopOverview, error) {
	return &DesktopOverview{}, nil
}

func TestDesktopSkillCreateValidates(t *testing.T) {
	svc := NewDesktopSkillService(&memDesktopSkills{})
	ctx := context.Background()
	ok := DesktopSkill{Slug: " Invoices ", Title: "发票整理", Summary: "把发票整理成表", Prompt: "整理发票"}

	s, err := svc.Create(ctx, ok, 1)
	require.NoError(t, err)
	require.Equal(t, "invoices", s.Slug, "slug is trimmed and lower-cased")
	require.Equal(t, DesktopSkillDraft, s.Status, "new skills start as drafts")

	bad := ok
	bad.Slug = "发票"
	_, err = svc.Create(ctx, bad, 1)
	require.ErrorIs(t, err, ErrDesktopSkillSlug)

	bad = ok
	bad.Slug, bad.Prompt = "x-1", ""
	_, err = svc.Create(ctx, bad, 1)
	require.ErrorIs(t, err, ErrDesktopSkillInvalid)

	bad = ok
	bad.Slug, bad.Status = "x-2", "live"
	_, err = svc.Create(ctx, bad, 1)
	require.ErrorIs(t, err, ErrDesktopSkillStatus)

	require.ErrorIs(t, svc.SetStatus(ctx, 1, "gone", 1), ErrDesktopSkillStatus)
}

func TestDesktopSkillForAppOnlyPublishedWithSkillMD(t *testing.T) {
	repo := &memDesktopSkills{rows: []DesktopSkill{
		{ID: 1, Slug: "invoices", Title: "发票整理", Summary: "把发票、报销单整理成 \"Excel\"\n表", Prompt: "p", Instructions: "# 发票整理\n\n做法", Status: DesktopSkillPublished, Version: 3},
		{ID: 2, Slug: "draft-one", Title: "草稿", Summary: "s", Prompt: "p", Status: DesktopSkillDraft},
		{ID: 3, Slug: "no-body", Title: "没有正文", Summary: "s", Prompt: "按这个做", Status: DesktopSkillPublished},
	}}
	got, err := NewDesktopSkillService(repo).ForApp(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2, "drafts are not sent")
	require.Equal(t, "hivegpt-invoices", got[0].SkillName)
	require.Equal(t, 3, got[0].Version)
	md := got[0].SkillMD
	require.True(t, strings.HasPrefix(md, "---\nname: hivegpt-invoices\ndescription: \"把发票、报销单整理成 \\\"Excel\\\" 表\"\n---\n\n# 发票整理"), md)
	require.Contains(t, got[1].SkillMD, "# 没有正文\n\n按这个做", "without instructions the card prompt is the body")
}
