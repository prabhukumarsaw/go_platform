package moderation

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for comments and moderation.
type Handler struct {
	service *Service
}

// NewHandler creates a new moderation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterPublicRoutes registers public commenting and feedback endpoints.
func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/articles/:articleId/comments", h.ListApprovedComments)
	router.Post("/articles/:articleId/comments", h.AddComment)
	router.Post("/feedback", h.SubmitFeedback)
}

// RegisterAdminRoutes registers moderation and feedback dashboard endpoints.
func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	mod := router.Group("/moderation")
	mod.Get("/comments/pending", h.ListPendingComments)
	mod.Post("/comments/:id/action", h.ModerateComment)
	mod.Get("/feedback", h.ListFeedbacks)
	mod.Patch("/feedback/:id/status", h.UpdateFeedbackStatus)
}

// ListApprovedComments returns public approved comments for an article.
func (h *Handler) ListApprovedComments(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	articleID, err := uuid.Parse(c.Params("articleId"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID")
	}

	comments, err := h.service.ListApprovedComments(c.Context(), tx, articleID)
	if err != nil {
		return response.InternalError(c, "Failed to load comments")
	}

	return response.Success(c, comments)
}

// AddComment submits a new comment (requires auth).
func (h *Handler) AddComment(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "You must be logged in to comment")
	}
	tx := c.Locals("tx").(pgx.Tx)

	articleID, err := uuid.Parse(c.Params("articleId"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID")
	}

	var input PostCommentInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	input.ArticleID = articleID

	if input.Body == "" {
		return response.BadRequest(c, "Comment body cannot be empty")
	}

	comment, err := h.service.AddComment(c.Context(), tx, sess.UserID, input)
	if err != nil {
		return response.InternalError(c, "Failed to post comment")
	}

	return response.Created(c, comment)
}

// ListPendingComments returns pending comments for moderators.
func (h *Handler) ListPendingComments(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 20)

	list, total, err := h.service.ListPendingQueue(c.Context(), tx, page, perPage)
	if err != nil {
		return response.InternalError(c, "Failed to fetch moderation queue")
	}

	return response.Paginated(c, list, page, perPage, total)
}

// ModerateComment performs an action on a comment (approve/reject/spam).
func (h *Handler) ModerateComment(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	commentID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid comment ID")
	}

	var body struct {
		Status string `json:"status"` // approved, rejected, spam
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.ModerateComment(c.Context(), tx, commentID, body.Status, sess.UserID); err != nil {
		return response.InternalError(c, "Moderation action failed: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Comment status updated", "status": body.Status})
}

// SubmitFeedback allows readers to report factual corrections, bug reports, and editorial feedback.
func (h *Handler) SubmitFeedback(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	}

	var input SubmitFeedbackInput
	if err := c.BodyParser(&input); err != nil || input.Name == "" || input.Email == "" || input.Message == "" {
		return response.BadRequest(c, "Name, email, and message are required")
	}

	var userID *int64
	if sess != nil {
		userID = &sess.UserID
	}

	fb, err := h.service.SubmitFeedback(c.Context(), tx, tenantID, userID, input)
	if err != nil {
		return response.InternalError(c, "Failed to submit feedback: "+err.Error())
	}

	return response.Created(c, fb)
}

// ListFeedbacks returns the editorial feedback queue for newsroom review.
func (h *Handler) ListFeedbacks(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	tenantID := 1
	if sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	status := c.Query("status")

	list, err := h.service.ListFeedbacks(c.Context(), tx, tenantID, status)
	if err != nil {
		return response.InternalError(c, "Failed to load feedbacks: "+err.Error())
	}

	return response.Success(c, list)
}

// UpdateFeedbackStatus updates the review state of reader feedback.
func (h *Handler) UpdateFeedbackStatus(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid feedback ID")
	}

	var body struct {
		Status string `json:"status"` // open, reviewed, resolved
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return response.BadRequest(c, "Valid status is required")
	}

	if err := h.service.UpdateFeedbackStatus(c.Context(), tx, id, body.Status); err != nil {
		return response.InternalError(c, "Failed to update feedback status")
	}

	return response.Success(c, fiber.Map{"message": "Feedback status updated"})
}
