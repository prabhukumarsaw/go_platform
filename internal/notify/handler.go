package notify

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for notifications, web push, and newsletter subscriptions.
type Handler struct {
	service *Service
}

// NewHandler creates a new notify handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterPublicRoutes registers newsletter and push subscription routes on public API.
func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Post("/newsletter/subscribe", h.Subscribe)
	router.Post("/newsletter/unsubscribe", h.Unsubscribe)
	router.Post("/notifications/subscribe", h.SubscribePush)
}

// RegisterAdminRoutes registers notification broadcasting endpoints for editors.
func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	router.Post("/notifications/broadcast", h.BroadcastPush)
}

// Subscribe handles newsletter opt-in.
func (h *Handler) Subscribe(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var input SubscribeInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if input.Email == "" {
		return response.BadRequest(c, "Email is required")
	}

	var userID *int64
	if sess := middleware.SessionFromCtx(c); sess != nil {
		userID = &sess.UserID
	}

	sub, err := h.service.Subscribe(c.Context(), tx, userID, input)
	if err != nil {
		return response.InternalError(c, "Failed to subscribe")
	}

	return response.Success(c, sub)
}

// Unsubscribe handles newsletter opt-out.
func (h *Handler) Unsubscribe(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if body.Email == "" {
		return response.BadRequest(c, "Email is required")
	}

	if err := h.service.Unsubscribe(c.Context(), tx, body.Email); err != nil {
		return response.InternalError(c, "Failed to unsubscribe")
	}

	return response.Success(c, fiber.Map{"message": "Unsubscribed successfully"})
}

// SubscribePush handles browser Web Push subscription registration.
func (h *Handler) SubscribePush(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var input PushSubscribeInput
	if err := c.BodyParser(&input); err != nil || input.Endpoint == "" || input.P256dhKey == "" || input.AuthKey == "" {
		return response.BadRequest(c, "Endpoint, p256dh_key, and auth_key are required")
	}

	input.UserAgent = c.Get("User-Agent")

	var userID *int64
	if sess := middleware.SessionFromCtx(c); sess != nil {
		userID = &sess.UserID
	}

	ps, err := h.service.SavePushSubscription(c.Context(), tx, userID, input)
	if err != nil {
		return response.InternalError(c, "Failed to save push subscription: "+err.Error())
	}

	return response.Created(c, ps)
}

// BroadcastPush dispatches breaking news push notification to subscribers.
func (h *Handler) BroadcastPush(c *fiber.Ctx) error {
	var body struct {
		DistrictID *int   `json:"district_id"`
		Title      string `json:"title"`
		Slug       string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil || body.Title == "" {
		return response.BadRequest(c, "Title and slug are required")
	}

	count, err := h.service.BroadcastBreakingNews(c.Context(), body.DistrictID, body.Title, body.Slug)
	if err != nil {
		return response.InternalError(c, "Broadcast failed: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"dispatched_count": count,
		"status":           "delivered",
	})
}
