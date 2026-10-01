package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

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
	h.service.ServeSite(c.Writer, c.Request, name)
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

// readSiteUpload reads the multipart form: "file" (optional when only the title changes) and "title".
func (h *SiteHostingHandler) readSiteUpload(c *gin.Context, requireFile bool) (service.SiteUpload, bool) {
	limit := int64(h.service.Config(c.Request.Context()).MaxMB) << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+1<<20)
	up := service.SiteUpload{Title: c.PostForm("title")}
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

// Create POST /api/v1/sites (multipart "file" + "title")
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
