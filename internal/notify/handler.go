package notify

import (
	"fmt"

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
	router.Get("/notifications/alerts", h.GetRecentAlerts)
}

// RegisterAdminRoutes registers notification broadcasting endpoints for editors.
func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	router.Post("/notifications/broadcast", h.BroadcastPush)
	router.Get("/notifications/stats", h.GetStats)
	router.Get("/notifications/history", h.GetHistory)
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
	var input BroadcastInput
	if err := c.BodyParser(&input); err != nil || input.Title == "" {
		return response.BadRequest(c, "Title is required")
	}

	if sess := middleware.SessionFromCtx(c); sess != nil && input.Sender == "" {
		input.Sender = fmt.Sprintf("Staff #%d", sess.UserID)
	}

	item, err := h.service.BroadcastBreakingNews(c.Context(), input)
	if err != nil {
		return response.InternalError(c, "Broadcast failed: "+err.Error())
	}

	return response.Success(c, item)
}

// GetStats returns notification subscriber metrics and daily counters.
func (h *Handler) GetStats(c *fiber.Ctx) error {
	stats, err := h.service.GetNotificationStats(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to load notification stats: "+err.Error())
	}
	return response.Success(c, stats)
}

// GetHistory returns recent broadcast notifications.
func (h *Handler) GetHistory(c *fiber.Ctx) error {
	history, err := h.service.GetBroadcastHistory(c.Context(), 30)
	if err != nil {
		return response.InternalError(c, "Failed to load broadcast history: "+err.Error())
	}
	return response.Success(c, history)
}

// GetRecentAlerts returns recent breaking alerts for the public reader notification sheet.
func (h *Handler) GetRecentAlerts(c *fiber.Ctx) error {
	alerts, err := h.service.GetBroadcastHistory(c.Context(), 20)
	if err != nil {
		return response.Success(c, []map[string]interface{}{})
	}
	return response.Success(c, alerts)
}
