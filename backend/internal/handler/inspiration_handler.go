package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// InspirationHandler serves the 灵感库 admin page (/api/v1/admin/inspiration/*). Song lyrics are
// returned by GetSong only, and only under the admin routes.
type InspirationHandler struct {
	svc *service.InspirationService
}

func NewInspirationHandler(svc *service.InspirationService) *InspirationHandler {
	return &InspirationHandler{svc: svc}
}

func (h *InspirationHandler) respond(c *gin.Context, data any, err error) {
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// SearchPoems GET /admin/inspiration/poems?q=&dynasty=&collection=&author=&famous=1&starred=1&random=1&limit=&offset=
func (h *InspirationHandler) SearchPoems(c *gin.Context) {
	poems, more, err := h.svc.SearchPoems(c.Request.Context(), service.LibPoemQuery{
		Q: c.Query("q"), Dynasty: c.Query("dynasty"), Collection: c.Query("collection"), Author: c.Query("author"),
		Famous: c.Query("famous") == "1", Starred: c.Query("starred") == "1", Random: c.Query("random") == "1",
		Limit: queryInt(c, "limit"), Offset: queryInt(c, "offset"),
	})
	h.respond(c, gin.H{"items": poems, "has_more": more}, err)
}

// PoemCounts GET /admin/inspiration/poems/counts
func (h *InspirationHandler) PoemCounts(c *gin.Context) {
	counts, err := h.svc.PoemCounts(c.Request.Context())
	h.respond(c, gin.H{"counts": counts, "names": service.LibCollections}, err)
}

// GetPoem GET /admin/inspiration/poems/:id
func (h *InspirationHandler) GetPoem(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibPoemNotFound)
	if !ok {
		return
	}
	p, err := h.svc.GetPoem(c.Request.Context(), id)
	h.respond(c, p, err)
}

type libStarInput struct {
	Starred bool `json:"starred"`
}

// StarPoem POST /admin/inspiration/poems/:id/star
func (h *InspirationHandler) StarPoem(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibPoemNotFound)
	if !ok {
		return
	}
	var in libStarInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	h.respond(c, gin.H{"starred": in.Starred}, h.svc.StarPoem(c.Request.Context(), id, in.Starred))
}

// ToShihua POST /admin/inspiration/poems/:id/to-shihua {stage}
func (h *InspirationHandler) ToShihua(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibPoemNotFound)
	if !ok {
		return
	}
	var in struct {
		Stage string `json:"stage"`
	}
	_ = c.ShouldBindJSON(&in)
	w, created, err := h.svc.ToShihua(c.Request.Context(), id, in.Stage)
	h.respond(c, gin.H{"work": w, "created": created}, err)
}

// Find POST /admin/inspiration/find {scene, dynasty, collection, count, rerank}
func (h *InspirationHandler) Find(c *gin.Context) {
	var in service.LibFindRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	res, err := h.svc.FindByScene(c.Request.Context(), in)
	h.respond(c, res, err)
}

// ListSongs GET /admin/inspiration/songs?q=&artist=&tag=&year_from=&year_to=&starred=1&limit=&offset=
func (h *InspirationHandler) ListSongs(c *gin.Context) {
	songs, total, err := h.svc.ListSongs(c.Request.Context(), service.LibSongQuery{
		Q: c.Query("q"), Artist: c.Query("artist"), Tag: c.Query("tag"), YearFrom: queryInt(c, "year_from"), YearTo: queryInt(c, "year_to"),
		Starred: c.Query("starred") == "1", Limit: queryInt(c, "limit"), Offset: queryInt(c, "offset"),
	})
	h.respond(c, gin.H{"items": songs, "total": total}, err)
}

// SongFacets GET /admin/inspiration/songs/facets
func (h *InspirationHandler) SongFacets(c *gin.Context) {
	f, err := h.svc.SongFacets(c.Request.Context())
	h.respond(c, f, err)
}

// GetSong GET /admin/inspiration/songs/:id (with the lyrics)
func (h *InspirationHandler) GetSong(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibSongNotFound)
	if !ok {
		return
	}
	s, err := h.svc.GetSong(c.Request.Context(), id)
	h.respond(c, s, err)
}

// CreateSong POST /admin/inspiration/songs
func (h *InspirationHandler) CreateSong(c *gin.Context) {
	var in service.LibSong
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	s, err := h.svc.CreateSong(c.Request.Context(), shihuaAdminID(c), in)
	h.respond(c, s, err)
}

// UpdateSong PUT /admin/inspiration/songs/:id
func (h *InspirationHandler) UpdateSong(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibSongNotFound)
	if !ok {
		return
	}
	var in service.LibSong
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	s, err := h.svc.UpdateSong(c.Request.Context(), id, in)
	h.respond(c, s, err)
}

// DeleteSong DELETE /admin/inspiration/songs/:id
func (h *InspirationHandler) DeleteSong(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibSongNotFound)
	if !ok {
		return
	}
	h.respond(c, gin.H{"deleted": true}, h.svc.DeleteSong(c.Request.Context(), id))
}

// StarSong POST /admin/inspiration/songs/:id/star
func (h *InspirationHandler) StarSong(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibSongNotFound)
	if !ok {
		return
	}
	var in libStarInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	h.respond(c, gin.H{"starred": in.Starred}, h.svc.StarSong(c.Request.Context(), id, in.Starred))
}

// AnalyzeSong POST /admin/inspiration/songs/:id/analyze (a draft; nothing is saved)
func (h *InspirationHandler) AnalyzeSong(c *gin.Context) {
	id, ok := shihuaPathID(c, service.ErrLibSongNotFound)
	if !ok {
		return
	}
	a, err := h.svc.AnalyzeSong(c.Request.Context(), id)
	h.respond(c, a, err)
}

// CheckLyrics POST /admin/inspiration/lyrics-check {text}: which lyric lines a text contains.
func (h *InspirationHandler) CheckLyrics(c *gin.Context) {
	var in struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || len(in.Text) > 20000 {
		response.ErrorFrom(c, service.ErrLibInvalid)
		return
	}
	hits, err := h.svc.Guard().Check(c.Request.Context(), in.Text)
	if hits == nil {
		hits = []service.LyricsHit{}
	}
	h.respond(c, gin.H{"hits": hits}, err)
}
