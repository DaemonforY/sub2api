package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContestHandler manages contests for admins.
type ContestHandler struct {
	service *service.ContestService
}

// NewContestHandler creates the admin contest handler.
func NewContestHandler(svc *service.ContestService) *ContestHandler {
	return &ContestHandler{service: svc}
}

type contestRequest struct {
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	Rules              string                 `json:"rules"`
	CoverImage         string                 `json:"cover_image"`
	Status             string                 `json:"status"`
	SubmissionStartAt  time.Time              `json:"submission_start_at"`
	SubmissionEndAt    time.Time              `json:"submission_end_at"`
	VotingStartAt      time.Time              `json:"voting_start_at"`
	VotingEndAt        time.Time              `json:"voting_end_at"`
	MaxEntriesPerUser  int                    `json:"max_entries_per_user"`
	VotesPerUser       int                    `json:"votes_per_user"`
	AllowSelfVote      bool                   `json:"allow_self_vote"`
	RequireReview      bool                   `json:"require_review"`
	MinAccountAgeHours int                    `json:"min_account_age_hours"`
	OnePrizePerUser    bool                   `json:"one_prize_per_user"`
	MinVotesForPrize   int                    `json:"min_votes_for_prize"`
	Prizes             []service.ContestPrize `json:"prizes"`
}

func (r *contestRequest) toContest() *service.Contest {
	return &service.Contest{
		Title: r.Title, Description: r.Description, Rules: r.Rules, CoverImage: r.CoverImage, Status: r.Status,
		SubmissionStartAt: r.SubmissionStartAt, SubmissionEndAt: r.SubmissionEndAt,
		VotingStartAt: r.VotingStartAt, VotingEndAt: r.VotingEndAt,
		MaxEntriesPerUser: r.MaxEntriesPerUser, VotesPerUser: r.VotesPerUser,
		AllowSelfVote: r.AllowSelfVote, RequireReview: r.RequireReview,
		MinAccountAgeHours: r.MinAccountAgeHours, OnePrizePerUser: r.OnePrizePerUser,
		MinVotesForPrize: r.MinVotesForPrize, Prizes: r.Prizes,
	}
}

func contestPathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}

// List GET /api/v1/admin/contests
func (h *ContestHandler) List(c *gin.Context) {
	list, err := h.service.AdminListContests(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// Get GET /api/v1/admin/contests/:id
func (h *ContestHandler) Get(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	contest, err := h.service.AdminGetContest(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, contest)
}

// Create POST /api/v1/admin/contests
func (h *ContestHandler) Create(c *gin.Context) {
	var req contestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var adminID int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		adminID = subject.UserID
	}
	contest, err := h.service.CreateContest(c.Request.Context(), req.toContest(), adminID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, contest)
}

// Update PUT /api/v1/admin/contests/:id
func (h *ContestHandler) Update(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	var req contestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	contest, err := h.service.UpdateContest(c.Request.Context(), id, req.toContest())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, contest)
}

// Delete DELETE /api/v1/admin/contests/:id
func (h *ContestHandler) Delete(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteContest(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Cancel POST /api/v1/admin/contests/:id/cancel
func (h *ContestHandler) Cancel(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	contest, err := h.service.CancelContest(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, contest)
}

// Settle POST /api/v1/admin/contests/:id/settle
func (h *ContestHandler) Settle(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	contest, err := h.service.SettleContest(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, contest)
}

// ListEntries GET /api/v1/admin/contests/:id/entries?status=pending,approved&sort=votes|new|rank
func (h *ContestHandler) ListEntries(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	var statuses []string
	for _, s := range strings.Split(c.Query("status"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			statuses = append(statuses, s)
		}
	}
	entries, total, err := h.service.AdminListEntries(c.Request.Context(), id, statuses, c.Query("sort"), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, int64(total), page, pageSize)
}

type reviewEntryRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

// ReviewEntry PUT /api/v1/admin/contests/:id/entries/:entryId/status
func (h *ContestHandler) ReviewEntry(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	entryID, ok := contestPathID(c, "entryId")
	if !ok {
		return
	}
	var req reviewEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "status is required")
		return
	}
	if err := h.service.ReviewEntry(c.Request.Context(), id, entryID, req.Status, req.Note); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": req.Status})
}

// ListEntryVotes GET /api/v1/admin/contests/:id/entries/:entryId/votes
func (h *ContestHandler) ListEntryVotes(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	entryID, ok := contestPathID(c, "entryId")
	if !ok {
		return
	}
	votes, err := h.service.AdminListEntryVotes(c.Request.Context(), id, entryID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, votes)
}

// VoidVote DELETE /api/v1/admin/contests/:id/votes/:voteId
func (h *ContestHandler) VoidVote(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	voteID, ok := contestPathID(c, "voteId")
	if !ok {
		return
	}
	if err := h.service.VoidVote(c.Request.Context(), id, voteID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"voided": true})
}

// ListAwards GET /api/v1/admin/contests/:id/awards
func (h *ContestHandler) ListAwards(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	awards, err := h.service.ListAwards(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, awards)
}

type grantAwardRequest struct {
	Note string `json:"note"`
}

// GrantAward POST /api/v1/admin/contests/:id/awards/:awardId/grant
func (h *ContestHandler) GrantAward(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	awardID, ok := contestPathID(c, "awardId")
	if !ok {
		return
	}
	var req grantAwardRequest
	_ = c.ShouldBindJSON(&req)
	award, err := h.service.GrantAward(c.Request.Context(), id, awardID, req.Note)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, award)
}

// GrantAllBalance POST /api/v1/admin/contests/:id/awards/grant-balance
func (h *ContestHandler) GrantAllBalance(c *gin.Context) {
	id, ok := contestPathID(c, "id")
	if !ok {
		return
	}
	granted, failed, err := h.service.GrantAllBalanceAwards(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"granted": granted, "failed": failed})
}
