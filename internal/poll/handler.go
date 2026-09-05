package poll

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/polls/active", h.GetActivePoll)
	router.Post("/polls/:id/vote", h.Vote)
}

func (h *Handler) GetActivePoll(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	language := c.Query("language", "hi")

	poll, err := h.service.GetActivePoll(c.Context(), tx, language)
	if err != nil {
		return response.InternalError(c, "Failed to get active poll: "+err.Error())
	}
	if poll == nil {
		return response.NotFound(c, "No active poll found")
	}

	return response.Success(c, poll)
}

func (h *Handler) Vote(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid poll ID format")
	}

	var req struct {
		OptionID int `json:"option_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.OptionID <= 0 {
		return response.BadRequest(c, "Valid option_id is required")
	}

	var userID *int64
	if sess != nil && sess.UserID > 0 {
		userID = &sess.UserID
	}

	ipAddress := c.IP()

	updatedPoll, err := h.service.Vote(c.Context(), tx, pollID, req.OptionID, userID, ipAddress)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, updatedPoll)
}
