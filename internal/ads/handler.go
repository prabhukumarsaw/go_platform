package ads

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for ad management.
type Handler struct {
	service *Service
}

// NewHandler creates a new ads handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers ad management routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	ads := router.Group("/ads")
	ads.Get("/slots", h.ListSlots)
	ads.Post("/slots", h.CreateSlot)
	ads.Put("/slots/:id", h.UpdateSlot)
	ads.Delete("/slots/:id", h.DeleteSlot)
	ads.Patch("/slots/:id/toggle", h.ToggleSlot)
}

// ListSlots returns all ad slots for the current tenant.
func (h *Handler) ListSlots(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	slots, err := h.service.ListAdSlots(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to list ad slots")
	}
	return response.Success(c, slots)
}

// CreateSlot creates a new ad slot.
func (h *Handler) CreateSlot(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var input CreateAdSlotInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	if input.Name == "" || input.SlotType == "" {
		return response.BadRequest(c, "Name and slot_type are required")
	}

	slot, err := h.service.CreateAdSlot(c.Context(), tx, int(sess.ActiveTenantID), input)
	if err != nil {
		return response.InternalError(c, "Failed to create ad slot")
	}
	return response.Created(c, slot)
}

// UpdateSlot updates an ad slot.
func (h *Handler) UpdateSlot(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid slot ID")
	}

	var input CreateAdSlotInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.UpdateAdSlot(c.Context(), tx, id, input); err != nil {
		return response.InternalError(c, "Failed to update ad slot")
	}
	return response.Success(c, fiber.Map{"message": "Ad slot updated"})
}

// DeleteSlot removes an ad slot.
func (h *Handler) DeleteSlot(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid slot ID")
	}

	if err := h.service.DeleteAdSlot(c.Context(), tx, id); err != nil {
		return response.InternalError(c, "Failed to delete ad slot")
	}
	return response.Success(c, fiber.Map{"message": "Ad slot deleted"})
}

// ToggleSlot enables/disables an ad slot.
func (h *Handler) ToggleSlot(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid slot ID")
	}

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.ToggleAdSlot(c.Context(), tx, id, body.IsActive); err != nil {
		return response.InternalError(c, "Failed to toggle ad slot")
	}
	return response.Success(c, fiber.Map{"message": "Ad slot toggled"})
}
