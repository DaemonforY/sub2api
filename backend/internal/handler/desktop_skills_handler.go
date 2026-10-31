package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// Skills lists the published skills for the app: cards + SKILL.md. Public (the content is what the
// app shows anyway); cached briefly since every app launch asks.
// GET /api/v1/desktop/skills
func (h *DesktopHandler) Skills(c *gin.Context) {
	skills, err := h.skills.ForApp(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, gin.H{"skills": skills})
}

type desktopSkillRequest struct {
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Icon         string `json:"icon"`
	Summary      string `json:"summary"`
	Prepare      string `json:"prepare"`
	Prompt       string `json:"prompt"`
	SamplePrompt string `json:"sample_prompt"`
	Instructions string `json:"instructions"`
	NeedsOffice  bool   `json:"needs_office"`
	Status       string `json:"status"`
	SortOrder    int    `json:"sort_order"`
}

func (r desktopSkillRequest) skill() service.DesktopSkill {
	return service.DesktopSkill{
		Slug: r.Slug, Title: r.Title, Icon: r.Icon, Summary: r.Summary, Prepare: r.Prepare, Prompt: r.Prompt,
		SamplePrompt: r.SamplePrompt, Instructions: r.Instructions, NeedsOffice: r.NeedsOffice, Status: r.Status,
		SortOrder: r.SortOrder,
	}
}

func (h *DesktopHandler) respond(c *gin.Context, data any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// AdminSkills GET /admin/desktop/skills (every status, display order)
func (h *DesktopHandler) AdminSkills(c *gin.Context) {
	skills, err := h.skills.List(c.Request.Context())
	h.respond(c, gin.H{"items": skills}, err)
}

// AdminGetSkill GET /admin/desktop/skills/:id (also returns the SKILL.md the app will get)
func (h *DesktopHandler) AdminGetSkill(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrDesktopSkillNotFound)
	if !ok {
		return
	}
	s, err := h.skills.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"skill": s, "skill_md": service.RenderDesktopSkillMD(s)})
}

// AdminCreateSkill POST /admin/desktop/skills (new skills start as drafts unless status is given)
func (h *DesktopHandler) AdminCreateSkill(c *gin.Context) {
	var req desktopSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopSkillInvalid)
		return
	}
	s, err := h.skills.Create(c.Request.Context(), req.skill(), shihuaAdminID(c))
	h.respond(c, s, err)
}

// AdminUpdateSkill PUT /admin/desktop/skills/:id (every save bumps the version the app compares)
func (h *DesktopHandler) AdminUpdateSkill(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrDesktopSkillNotFound)
	if !ok {
		return
	}
	var req desktopSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopSkillInvalid)
		return
	}
	s, err := h.skills.Update(c.Request.Context(), id, req.skill(), shihuaAdminID(c))
	h.respond(c, s, err)
}

// AdminSetSkillStatus POST /admin/desktop/skills/:id/status {"status": "draft|published|hidden"}
func (h *DesktopHandler) AdminSetSkillStatus(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrDesktopSkillNotFound)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopSkillStatus)
		return
	}
	h.respond(c, gin.H{"ok": true}, h.skills.SetStatus(c.Request.Context(), id, req.Status, shihuaAdminID(c)))
}

// AdminDeleteSkill DELETE /admin/desktop/skills/:id
func (h *DesktopHandler) AdminDeleteSkill(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrDesktopSkillNotFound)
	if !ok {
		return
	}
	h.respond(c, gin.H{"ok": true}, h.skills.Delete(c.Request.Context(), id))
}

// AdminOverview GET /admin/desktop/overview?days=30 — devices, funnel, per-template results.
func (h *DesktopHandler) AdminOverview(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	o, err := h.skills.Overview(c.Request.Context(), days)
	h.respond(c, o, err)
}
