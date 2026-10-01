package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SiteHostingHandler serves hosted sites by Host and the signed-in user's site management.
type SiteHostingHandler struct {
	service *service.SiteHostingService
}

func NewSiteHostingHandler(svc *service.SiteHostingService) *SiteHostingHandler {
	return &SiteHostingHandler{service: svc}
}

// HostMiddleware answers every request for <name>.<hosting domain> with the hosted site; nothing
// else of the main site (API, panel, cookies) is reachable on those hosts.
func (h *SiteHostingHandler) HostMiddleware(c *gin.Context) {
	name, ok := h.service.SiteNameFromHost(c.Request.Host)
	if !ok {
		c.Next()
		return
	}
	h.service.ServeSite(c.Writer, c.Request, name, c.ClientIP())
	c.Abort()
}

// TLSCheck GET /api/v1/sites/tls-check?domain= — Caddy's on-demand TLS "ask" endpoint.
func (h *SiteHostingHandler) TLSCheck(c *gin.Context) {
	if h.service.KnownSite(c.Request.Context(), c.Query("domain")) {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusNotFound)
}

func siteUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "请先登录")
		return 0, false
	}
	return subject.UserID, true
}

func sitePathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrSiteNotFound)
		return 0, false
	}
	return id, true
}

// readSiteUpload reads the multipart form: "file" (optional when only the title changes), "title" and "name" (creating only).
func (h *SiteHostingHandler) readSiteUpload(c *gin.Context, requireFile bool) (service.SiteUpload, bool) {
	limit := int64(h.service.Config(c.Request.Context()).MaxMB) << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+1<<20)
	up := service.SiteUpload{Title: c.PostForm("title"), Name: c.PostForm("name")}
	file, err := c.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.ErrorFrom(c, service.ErrSiteUploadTooLarge)
			return up, false
		}
		if requireFile {
			response.ErrorFrom(c, service.ErrSiteUploadInvalid)
			return up, false
		}
		return up, true
	}
	f, err := file.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrSiteUploadInvalid)
		return up, false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		response.ErrorFrom(c, service.ErrSiteUploadInvalid)
		return up, false
	}
	up.FileName, up.Data = file.Filename, data
	return up, true
}

// CheckName GET /api/v1/sites/name-check?name=&site_id= — whether a name can be used (site_id: renaming that site).
func (h *SiteHostingHandler) CheckName(c *gin.Context) {
	if _, ok := siteUserID(c); !ok {
		return
	}
	siteID, _ := strconv.ParseInt(c.Query("site_id"), 10, 64)
	res, err := h.service.CheckName(c.Request.Context(), siteID, c.Query("name"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

// Rename PUT /api/v1/sites/:id/name {name}
func (h *SiteHostingHandler) Rename(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteNameInvalid)
		return
	}
	site, err := h.service.Rename(c.Request.Context(), userID, id, in.Name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// Mine GET /api/v1/sites
func (h *SiteHostingHandler) Mine(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	res, err := h.service.Mine(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, res)
}

// Create POST /api/v1/sites (multipart "file" + "title", optional "name")
func (h *SiteHostingHandler) Create(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	up, ok := h.readSiteUpload(c, true)
	if !ok {
		return
	}
	site, err := h.service.Create(c.Request.Context(), userID, up)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// Update PUT /api/v1/sites/:id (multipart: optional "file", "title")
func (h *SiteHostingHandler) Update(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	up, ok := h.readSiteUpload(c, false)
	if !ok {
		return
	}
	site, err := h.service.Update(c.Request.Context(), userID, id, up)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// Delete DELETE /api/v1/sites/:id
func (h *SiteHostingHandler) Delete(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Renew POST /api/v1/sites/:id/renew
func (h *SiteHostingHandler) Renew(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	site, err := h.service.Renew(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// Report POST /api/v1/site-reports {site, reason, detail, contact} — public.
func (h *SiteHostingHandler) Report(c *gin.Context) {
	var in struct {
		Site    string `json:"site"`
		Reason  string `json:"reason"`
		Detail  string `json:"detail"`
		Contact string `json:"contact"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteReportInvalid)
		return
	}
	if err := h.service.Report(c.Request.Context(), in.Site, in.Reason, in.Detail, in.Contact, c.ClientIP()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"reported": true})
}

// Versions GET /api/v1/sites/:id/versions
func (h *SiteHostingHandler) Versions(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	versions, err := h.service.Versions(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, versions)
}

// Rollback POST /api/v1/sites/:id/rollback {version}
func (h *SiteHostingHandler) Rollback(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	var in struct {
		Version int `json:"version"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteVersionNotFound)
		return
	}
	site, err := h.service.Rollback(c.Request.Context(), userID, id, in.Version)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// SetPassword PUT /api/v1/sites/:id/password {password} ("" removes it)
func (h *SiteHostingHandler) SetPassword(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSitePasswordInvalid)
		return
	}
	site, err := h.service.SetPassword(c.Request.Context(), userID, id, in.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// Stats GET /api/v1/sites/:id/stats?days=
func (h *SiteHostingHandler) Stats(c *gin.Context) {
	userID, ok := siteUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	days, _ := strconv.Atoi(c.Query("days"))
	stats, err := h.service.Stats(c.Request.Context(), userID, id, days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// Open API (API key) --------------------------------------------------------------------------

// keyUpload reads a publish request from the API: multipart ("file", "title") or JSON {title, html}.
func (h *SiteHostingHandler) keyUpload(c *gin.Context, requireContent bool) (service.SiteUpload, bool) {
	if !strings.HasPrefix(c.ContentType(), "application/json") {
		return h.readSiteUpload(c, requireContent)
	}
	limit := int64(h.service.Config(c.Request.Context()).MaxMB) << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+1<<20)
	var in struct {
		Title string `json:"title"`
		Name  string `json:"name"`
		HTML  string `json:"html"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || (requireContent && strings.TrimSpace(in.HTML) == "") {
		response.ErrorFrom(c, service.ErrSiteUploadInvalid)
		return service.SiteUpload{}, false
	}
	up := service.SiteUpload{Title: in.Title, Name: in.Name}
	if strings.TrimSpace(in.HTML) != "" {
		up.FileName, up.Data = "index.html", []byte(in.HTML)
	}
	return up, true
}

// KeyList GET /api/v1/hosting/sites — the key owner's sites.
func (h *SiteHostingHandler) KeyList(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	res, err := h.service.Mine(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"sites": res.Sites, "quota": res.Quota})
}

// KeyCreate POST /api/v1/hosting/sites — publish (JSON {title, html} or multipart).
func (h *SiteHostingHandler) KeyCreate(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	up, ok := h.keyUpload(c, true)
	if !ok {
		return
	}
	site, err := h.service.Create(c.Request.Context(), userID, up)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

// KeyUpdate PUT /api/v1/hosting/sites/:id — publish a new version and/or rename.
func (h *SiteHostingHandler) KeyUpdate(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	id, ok := sitePathID(c)
	if !ok {
		return
	}
	up, ok := h.keyUpload(c, false)
	if !ok {
		return
	}
	site, err := h.service.Update(c.Request.Context(), userID, id, up)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}
