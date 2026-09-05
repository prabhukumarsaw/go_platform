package analytics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	admin := router.Group("/analytics")
	admin.Get("/overview", h.GetOverview)
	admin.Get("/authors", h.GetAuthorLeaderboard)
}

func (h *Handler) GetOverview(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	overview, err := h.service.GetOverview(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to get analytics overview: "+err.Error())
	}

	return response.Success(c, overview)
}

func (h *Handler) GetAuthorLeaderboard(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	leaderboard, err := h.service.GetAuthorLeaderboard(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to get author leaderboard: "+err.Error())
	}

	return response.Success(c, leaderboard)
}
