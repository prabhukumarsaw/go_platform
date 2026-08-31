package ai

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterStudioRoutes(router fiber.Router) {
	aiGroup := router.Group("/studio/ai")
	aiGroup.Post("/assist", h.EditorialAssist)
}

type AssistRequest struct {
	Title    string `json:"title"`
	BodyText string `json:"body_text"`
}

func (h *Handler) EditorialAssist(c *fiber.Ctx) error {
	var req AssistRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	result, err := h.service.GenerateEditorialAssistance(c.Context(), req.Title, req.BodyText)
	if err != nil {
		return response.InternalError(c, "Failed to generate editorial assistance")
	}

	return response.Success(c, result)
}
